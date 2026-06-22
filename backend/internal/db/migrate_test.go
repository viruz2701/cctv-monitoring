package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrationsNoCreateTableIfNotExists проверяет, что в миграциях нет CREATE TABLE IF NOT EXISTS.
// Соответствует: ISO 27001 A.12.1.2, ISO 27001 A.14.2.2
func TestMigrationsNoCreateTableIfNotExists(t *testing.T) {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("migrations directory not found, skipping")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", entry.Name(), err)
			continue
		}

		content := string(data)
		if strings.Contains(content, "CREATE TABLE IF NOT EXISTS") {
			t.Errorf("FOUND CREATE TABLE IF NOT EXISTS in %s — must use CREATE TABLE without IF NOT EXISTS", entry.Name())
		}
	}
}

// TestMigrationFilesStructure проверяет, что файлы миграций имеют правильную структуру:
// {version}_{name}.up.sql и {version}_{name}.down.sql
func TestMigrationFilesStructure(t *testing.T) {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("migrations directory not found, skipping")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	// Собираем все .sql файлы (не .bak)
	var sqlFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}
		sqlFiles = append(sqlFiles, entry.Name())
	}

	// Проверяем что для каждого up есть down
	upFiles := make(map[string]bool)
	downFiles := make(map[string]bool)

	for _, f := range sqlFiles {
		if strings.HasSuffix(f, ".up.sql") {
			name := strings.TrimSuffix(f, ".up.sql")
			upFiles[name] = true
		}
		if strings.HasSuffix(f, ".down.sql") {
			name := strings.TrimSuffix(f, ".down.sql")
			downFiles[name] = true
		}
	}

	for name := range upFiles {
		if !downFiles[name] {
			t.Errorf("missing .down.sql for migration %s", name)
		}
	}
	for name := range downFiles {
		if !upFiles[name] {
			t.Errorf("missing .up.sql for migration %s", name)
		}
	}
}

// TestMigrationUpHasDown тестирует что в up-файлах нет DROP TABLE (они должны быть только в down)
func TestMigrationUpHasNoDrop(t *testing.T) {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("migrations directory not found, skipping")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", entry.Name(), err)
			continue
		}

		content := string(data)
		if strings.Contains(content, "DROP TABLE") {
			t.Errorf("FOUND DROP TABLE in up-migration %s — DROP should be in .down.sql only", entry.Name())
		}
	}
}

// TestMigrationDownHasDrop тестирует что в down-файлах есть DROP TABLE
func TestMigrationDownHasDrop(t *testing.T) {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("migrations directory not found, skipping")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".down.sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", entry.Name(), err)
			continue
		}

		content := string(data)
		if !strings.Contains(content, "DROP TABLE") && !strings.Contains(content, "DROP INDEX") {
			t.Errorf("no DROP statements found in down-migration %s", entry.Name())
		}
	}
}

// TestMigrationsHavePlusMigrateComment проверяет наличие директивы +migrate
func TestMigrationsHavePlusMigrateComment(t *testing.T) {
	migrationsDir := findMigrationsDir()
	if migrationsDir == "" {
		t.Skip("migrations directory not found, skipping")
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("failed to read migrations dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", entry.Name(), err)
			continue
		}

		content := string(data)
		if !strings.Contains(content, "-- +migrate") {
			t.Errorf("migration %s is missing -- +migrate directive", entry.Name())
		}
	}
}
