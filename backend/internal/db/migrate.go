// Package db — database connection and migration runner.
package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending database migrations using golang-migrate.
// Uses embedded migrations from internal/db/migrations/ directory.
// Accepts the database DSN (e.g. "postgres://user:pass@localhost:5432/dbname?sslmode=disable").
func RunMigrations(dsn string, logger *slog.Logger) error {
	// Find migrations directory relative to the backend root
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		return fmt.Errorf("migrations directory not found")
	}

	sourceURL := "file://" + migrationsDir
	logger.Info("running database migrations", "source", sourceURL)

	m, err := migrate.New(sourceURL, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		logger.Warn("failed to get migration version", "error", err)
	} else {
		logger.Info("database migrations complete", "version", version, "dirty", dirty)
	}

	return nil
}

// findMigrationsDir находит директорию с миграциями относительно CWD или модуля.
func findMigrationsDir() string {
	// Try relative paths from CWD
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

	// Fallback: try relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "internal", "db", "migrations")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	return ""
}
