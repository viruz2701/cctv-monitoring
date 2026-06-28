package db

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"gb-telemetry-collector/internal/auth"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

type Config struct {
	Host              string
	Port              int
	User              string
	Password          string
	DBName            string
	SSLMode           string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func New(cfg Config, logger *slog.Logger) (*DB, error) {
	cfg = cfg.withDefaults()
	if cfg.MinConns > cfg.MaxConns {
		return nil, fmt.Errorf("database min connections (%d) cannot exceed max connections (%d)", cfg.MinConns, cfg.MaxConns)
	}

	connURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	poolConfig, err := pgxpool.ParseConfig(connURL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to connect to database at %s:%d/%s: %w",
			cfg.Host, cfg.Port, cfg.DBName, err)
	}

	logger.Info("database pool initialized",
		"max_conns", cfg.MaxConns,
		"min_conns", cfg.MinConns,
		"host", cfg.Host,
		"db", cfg.DBName,
	)
	return &DB{Pool: pool, Logger: logger}, nil
}

// withDefaults устанавливает значения по умолчанию для пула соединений.
// Соответствует: ISO 27001 A.12.1.2 (Change management), ISO 27001 A.12.6.1 (Technical vulnerabilities)
func (cfg Config) withDefaults() Config {
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 25
	}
	if cfg.MinConns <= 0 {
		cfg.MinConns = 5
	}
	if cfg.MaxConnLifetime <= 0 {
		cfg.MaxConnLifetime = 5 * time.Minute
	}
	if cfg.MaxConnIdleTime <= 0 {
		cfg.MaxConnIdleTime = 3 * time.Minute
	}
	if cfg.HealthCheckPeriod <= 0 {
		cfg.HealthCheckPeriod = 1 * time.Minute
	}
	return cfg
}

// DSN возвращает строку DSN для golang-migrate.
func (cfg Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)
}

// SeedDefaultAdmin создаёт администратора по умолчанию, если в БД нет пользователей.
// Вызывается ПОСЛЕ миграций.
func (db *DB) SeedDefaultAdmin() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("seed: check users count: %w", err)
	}
	if count > 0 {
		return nil
	}

	// ═══ БЕЗОПАСНОСТЬ: пароль администратора ТОЛЬКО из env ═══
	// GB_ADMIN_PASSWORD — обязателен при первом запуске (seed)
	// Если не задан — генерируем случайный 32-символьный пароль
	// ⚠ КРИТИЧЕСКИ: пароль НЕ ДОЛЖЕН логироваться (OWASP ASVS V6.2, ISO 27001 A.10.1.2)
	adminPassword := os.Getenv("GB_ADMIN_PASSWORD")
	if adminPassword == "" {
		randomBytes := make([]byte, 24)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("seed: failed to generate random password: %w", err)
		}
		adminPassword = hex.EncodeToString(randomBytes)
		db.Logger.Warn("GB_ADMIN_PASSWORD not set — generated random password. " +
			"Check server startup logs for the generated password (saved to /tmp/cctv-admin-credentials.txt once). " +
			"CHANGE THIS PASSWORD IMMEDIATELY after first login.")
		// Записываем пароль в отдельный защищённый файл, а не в логи
		if err := os.WriteFile("/tmp/cctv-admin-credentials.txt",
			[]byte("Admin password: "+adminPassword+"\nCHANGE IMMEDIATELY AFTER FIRST LOGIN\n"),
			0600); err != nil {
			db.Logger.Warn("Failed to save admin credentials file", "error", err)
		}
	}

	hashed, err := auth.HashPassword(adminPassword)
	if err != nil {
		return fmt.Errorf("seed: hash password: %w", err)
	}
	if _, err := db.Pool.Exec(ctx, `
		INSERT INTO users (id, username, password_hash, role, email, tenant_id)
		VALUES (gen_random_uuid()::text, 'admin', $1, 'admin', 'admin@example.com', '')
	`, hashed); err != nil {
		return fmt.Errorf("seed: create admin: %w", err)
	}

	db.Logger.Info("Default admin user created",
		"username", "admin",
		"password_source", map[bool]string{true: "env", false: "auto-generated"}[os.Getenv("GB_ADMIN_PASSWORD") != ""],
	)
	return nil
}

// initSchema удалён — replaced by golang-migrate in internal/db/migrate.go.
// См. backend/internal/db/migrations/001_initial_schema.up.sql

func (db *DB) Close() {
	db.Pool.Close()
}

// ═══════════════════════════════════════════════════════════════════════
// System Settings (ИСПРАВЛЕНО: правильное сканирование JSONB)
// ═══════════════════════════════════════════════════════════════════════

// GetSystemSettings возвращает ВСЕ настройки в виде map[string]json.RawMessage
// Ключи: "services_syslog", "services_gb28181", и т.д.
func (db *DB) GetSystemSettings() (map[string]json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `SELECT key, value FROM system_settings ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("query system_settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]json.RawMessage)
	for rows.Next() {
		var key string
		var value []byte // JSONB сканируется как []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan system_settings row: %w", err)
		}
		settings[key] = json.RawMessage(value)
	}

	return settings, rows.Err()
}

// GetServiceSetting возвращает одну настройку, десериализованную в указатель
func (db *DB) GetServiceSetting(key string, dest interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var rawValue []byte
	err := db.Pool.QueryRow(ctx, `SELECT value FROM system_settings WHERE key = $1`, key).Scan(&rawValue)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("setting %q not found", key)
		}
		return fmt.Errorf("query setting %q: %w", key, err)
	}

	if err := json.Unmarshal(rawValue, dest); err != nil {
		return fmt.Errorf("unmarshal setting %q: %w", key, err)
	}
	return nil
}

// UpdateSystemSettings обновляет (или создаёт) настройку.
// value может быть: map, struct, slice, string — всё будет сериализовано в JSON.
func (db *DB) UpdateSystemSettings(key string, value interface{}, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Сериализуем Go-объект в JSON для JSONB-колонки
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal setting %q: %w", key, err)
	}

	// Валидируем что это корректный JSON
	if !json.Valid(jsonBytes) {
		return fmt.Errorf("invalid JSON for setting %q", key)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO system_settings (key, value, updated_by, updated_at)
		VALUES ($1, $2::jsonb, $3, NOW())
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
	`, key, jsonBytes, userID)

	if err != nil {
		return fmt.Errorf("upsert setting %q: %w", key, err)
	}

	db.Logger.Info("System setting updated", "key", key, "updated_by", userID)
	return nil
}

// UpdateMultipleSettings обновляет несколько настроек в одной транзакции
func (db *DB) UpdateMultipleSettings(settings map[string]interface{}, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for key, value := range settings {
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal setting %q: %w", key, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO system_settings (key, value, updated_by, updated_at)
			VALUES ($1, $2::jsonb, $3, NOW())
			ON CONFLICT (key) DO UPDATE SET
				value = EXCLUDED.value,
				updated_by = EXCLUDED.updated_by,
				updated_at = NOW()
		`, key, jsonBytes, userID)

		if err != nil {
			return fmt.Errorf("upsert setting %q: %w", key, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	db.Logger.Info("Multiple system settings updated", "count", len(settings), "updated_by", userID)
	return nil
}
