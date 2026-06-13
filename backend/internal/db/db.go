package db

import (
	"context"
	"fmt"
	"gb-telemetry-collector/internal/auth"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger
}

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func New(cfg Config, logger *slog.Logger) (*DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	db := &DB{Pool: pool, Logger: logger}
	if err := db.initSchema(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS timescaledb;`)
	if err != nil {
		return fmt.Errorf("failed to enable timescaledb: %w", err)
	}

	// Таблица устройств
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS devices (
            device_id TEXT PRIMARY KEY,
            owner_id TEXT,
            name TEXT,
            location TEXT,
            vendor_type TEXT,
            status TEXT,
            last_seen TIMESTAMPTZ,
            registered_at TIMESTAMPTZ,
            heartbeat_interval INT,
            user_agent TEXT,
            log_raw_data BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        );
    `)
	if err != nil {
		return err
	}

	// Таблица телеметрии
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS telemetry (
            time TIMESTAMPTZ NOT NULL,
            device_id TEXT NOT NULL,
            status TEXT,
            last_seen TIMESTAMPTZ,
            heartbeat_interval INT
        );
        SELECT create_hypertable('telemetry', 'time', if_not_exists => TRUE);
    `)
	if err != nil {
		return err
	}

	// Таблица тревог
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS alarms (
            time TIMESTAMPTZ NOT NULL,
            device_id TEXT NOT NULL,
            priority INT,
            method INT,
            description TEXT
        );
        SELECT create_hypertable('alarms', 'time', if_not_exists => TRUE);
    `)
	if err != nil {
		return err
	}

	// Таблица прогнозов
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS predictions (
            device_id TEXT NOT NULL,
            prediction_date TIMESTAMPTZ NOT NULL,
            failure_probability FLOAT,
            expected_remaining_hours INT,
            explanation TEXT,
            model_version TEXT
        );
        SELECT create_hypertable('predictions', 'prediction_date', if_not_exists => TRUE);
    `)
	if err != nil {
		return err
	}

	// Таблица аудита
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS audit_log (
            id BIGSERIAL PRIMARY KEY,
            timestamp TIMESTAMPTZ NOT NULL,
            user_uuid TEXT,
            action TEXT,
            entity_type TEXT,
            entity_id TEXT,
            old_value JSONB,
            new_value JSONB
        );
    `)
	if err != nil {
		return err
	}

	// Таблица пользователей
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            role TEXT NOT NULL CHECK (role IN ('admin', 'support', 'owner')),
            owner_id TEXT,
            created_at TIMESTAMPTZ DEFAULT NOW()
        );
    `)
	if err != nil {
		return err
	}

	// Таблица парсенных логов
	_, err = db.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS parsed_logs (
            time TIMESTAMPTZ NOT NULL,
            device_id TEXT NOT NULL,
            log_level TEXT,
            event_code INT,
            message TEXT,
            source TEXT,
            raw TEXT
        );
        SELECT create_hypertable('parsed_logs', 'time', if_not_exists => TRUE);
        SELECT add_retention_policy('parsed_logs', INTERVAL '30 days', if_not_exists => TRUE);
    `)
	if err != nil {
		return err
	}

	var count int
	err = db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		db.Logger.Warn("cannot check users count", "error", err)
	}
	if err == nil && count == 0 {
		hashed, err := auth.HashPassword("admin123")
		if err != nil {
			db.Logger.Error("failed to hash default password", "error", err)
		} else {
			_, err = db.Pool.Exec(ctx, `
            INSERT INTO users (id, username, password_hash, role)
            VALUES (gen_random_uuid()::text, 'admin', $1, 'admin')
        `, hashed)
			if err != nil {
				db.Logger.Error("failed to create default admin", "error", err)
			} else {
				db.Logger.Info("Default admin user created: admin / admin123")
			}
		}
	}
	// Retention для telemetry
	_, err = db.Pool.Exec(ctx, `
        SELECT add_retention_policy('telemetry', INTERVAL '30 days', if_not_exists => TRUE);
    `)
	if err != nil {
		db.Logger.Warn("Failed to add retention policy", "error", err)
	}

	db.Logger.Info("Database schema initialized")
	return nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
