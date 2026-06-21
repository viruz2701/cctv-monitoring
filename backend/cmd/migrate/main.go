// Command migrate — standalone CLI for database migrations.
// Usage:
//
//	go run ./cmd/migrate up       — apply all pending migrations
//	go run ./cmd/migrate down     — rollback all migrations
//	go run ./cmd/migrate version  — show current migration version
//	go run ./cmd/migrate create <name> — create new migration files
//
// Environment variables:
//
//	DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE
package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func dsn() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "gb_user")
	password := getEnv("DB_PASSWORD", "gb_password")
	dbname := getEnv("DB_NAME", "gb_telemetry")
	sslmode := getEnv("DB_SSLMODE", "disable")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
}

func findMigrationsDir() string {
	candidates := []string{
		"internal/db/migrations",
		"backend/internal/db/migrations",
		"../../internal/db/migrations",
	}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "internal", "db", "migrations")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: migrate <up|down|version|create> [name]\n")
		os.Exit(1)
	}

	cmd := os.Args[1]

	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		logger.Error("migrations directory not found — run from backend/ or set working directory")
		os.Exit(1)
	}

	sourceURL := "file://" + migrationsDir
	targetDSN := dsn()

	// Hide password in logs
	safeDSN := targetDSN
	if idx := strings.Index(targetDSN, "@"); idx > 0 {
		if colonIdx := strings.LastIndex(targetDSN[:idx], ":"); colonIdx > 0 {
			safeDSN = targetDSN[:colonIdx+1] + "***" + targetDSN[idx:]
		}
	}
	logger.Info("connecting", "dsn", safeDSN, "source", sourceURL)

	switch cmd {
	case "up":
		m, err := migrate.New(sourceURL, targetDSN)
		if err != nil {
			logger.Error("migrate init failed", "error", err)
			os.Exit(1)
		}
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			logger.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		v, dirty, err := m.Version()
		if err != nil && err != migrate.ErrNilVersion {
			logger.Error("version check failed", "error", err)
		} else {
			logger.Info("migrations complete", "version", v, "dirty", dirty)
		}
		m.Close()

	case "down":
		m, err := migrate.New(sourceURL, targetDSN)
		if err != nil {
			logger.Error("migrate init failed", "error", err)
			os.Exit(1)
		}
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			logger.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		logger.Info("all migrations rolled back")
		m.Close()

	case "version":
		m, err := migrate.New(sourceURL, targetDSN)
		if err != nil {
			logger.Error("migrate init failed", "error", err)
			os.Exit(1)
		}
		v, dirty, err := m.Version()
		if err != nil {
			if err == migrate.ErrNilVersion {
				fmt.Println("no migrations applied yet")
			} else {
				logger.Error("version check failed", "error", err)
			}
		} else {
			fmt.Printf("version: %d, dirty: %v\n", v, dirty)
		}
		m.Close()

	case "create":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Usage: migrate create <name>\n")
			os.Exit(1)
		}
		name := os.Args[2]
		upFile := filepath.Join(migrationsDir, fmt.Sprintf("%s.up.sql", name))
		downFile := filepath.Join(migrationsDir, fmt.Sprintf("%s.down.sql", name))
		if err := os.WriteFile(upFile, []byte("-- +migrate Up\n"), 0o644); err != nil {
			logger.Error("create up file failed", "error", err)
			os.Exit(1)
		}
		if err := os.WriteFile(downFile, []byte("-- +migrate Down\n"), 0o644); err != nil {
			logger.Error("create down file failed", "error", err)
			os.Exit(1)
		}
		logger.Info("migration files created", "up", upFile, "down", downFile)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: migrate <up|down|version|create> [name]\n", cmd)
		os.Exit(1)
	}
}
