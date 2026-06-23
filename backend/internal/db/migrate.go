// Package db — database connection and migration runner.
package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

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

	// === Dirty state recovery ===
	// Если FORCE_MIGRATION_VERSION установлен — сбрасываем dirty state
	// перед запуском миграций. Используется для восстановления после
	// неудачной миграции (например, "Dirty database version 5").
	// Установите FORCE_MIGRATION_VERSION=<последняя_чистая_версия>
	if forceVerStr := os.Getenv("FORCE_MIGRATION_VERSION"); forceVerStr != "" {
		forceVer, err := strconv.Atoi(forceVerStr)
		if err != nil {
			return fmt.Errorf("invalid FORCE_MIGRATION_VERSION: must be integer, got %q", forceVerStr)
		}
		logger.Warn("forcing migration version (dirty state recovery)",
			"force_version", forceVer,
			"source", sourceURL,
		)
		if err := m.Force(forceVer); err != nil {
			return fmt.Errorf("failed to force migration version %d: %w", forceVer, err)
		}
		logger.Info("migration version forced", "version", forceVer)
	}

	// === Check for dirty state (preventive diagnostics) ===
	currentVer, dirty, verErr := m.Version()
	if verErr != nil && verErr != migrate.ErrNilVersion {
		logger.Warn("failed to check migration version", "error", verErr)
	} else if dirty {
		logger.Error("database is in dirty state",
			"version", currentVer,
			"action", "set FORCE_MIGRATION_VERSION=<last_clean_version> to recover",
		)
		return fmt.Errorf("dirty database version %d. Set FORCE_MIGRATION_VERSION=<last_clean_version> and restart", currentVer)
	}

	// === Run pending migrations ===
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
