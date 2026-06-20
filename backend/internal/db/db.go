package db

import (
	"context"
	"encoding/json"
	"fmt"
	"gb-telemetry-collector/internal/auth"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
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
		pool.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

func (db *DB) initSchema() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	db.Logger.Info("Starting database schema initialization")

	// 1. TimescaleDB
	if _, err := tx.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS timescaledb;`); err != nil {
		return fmt.Errorf("failed to enable timescaledb: %w", err)
	}

	// 2. Users
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL CHECK (role IN ('admin', 'support', 'owner', 'manager', 'technician', 'viewer')),
			owner_id TEXT,
			email TEXT,
			avatar TEXT,
			sites TEXT[],
			status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
			last_login TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// 3. Sites
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			address TEXT,
			city TEXT,
			status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'maintenance')),
			last_sync TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create sites table: %w", err)
	}

	// 4. Devices (с GB28181 и ONVIF полями)
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS devices (
			device_id TEXT PRIMARY KEY,
			owner_id TEXT REFERENCES users(id) ON DELETE SET NULL,
			site_id TEXT REFERENCES sites(id) ON DELETE SET NULL,
			name TEXT,
			location TEXT,
			vendor_type TEXT,
			device_type TEXT DEFAULT 'camera' CHECK (device_type IN ('camera', 'nvr', 'dvr', 'switch')),
			status TEXT DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'warning')),
			health TEXT DEFAULT 'healthy' CHECK (health IN ('healthy', 'faulty', 'degraded')),
			recording_status TEXT DEFAULT 'recording' CHECK (recording_status IN ('recording', 'not_recording', 'scheduled')),
			last_seen TIMESTAMPTZ,
			registered_at TIMESTAMPTZ DEFAULT NOW(),
			heartbeat_interval INT,
			user_agent TEXT,
			log_raw_data BOOLEAN DEFAULT TRUE,

			-- Connection type (расширен GB28181 и ONVIF)
			connection_type TEXT DEFAULT 'ip' CHECK (connection_type IN (
				'ip', 'p2p', 'snmp', 'syslog', 'alarm', 'gb28181', 'onvif'
			)),

			-- P2P fields
			p2p_brand TEXT,
			p2p_serial TEXT,
			p2p_security_code TEXT,
			p2p_cloud_user TEXT,
			p2p_cloud_pass TEXT,
			cloud_status TEXT DEFAULT 'unknown',

			-- SNMP fields
			snmp_community TEXT,
			snmp_version TEXT DEFAULT 'v2c' CHECK (snmp_version IN ('v1', 'v2c', 'v3')),

			-- Syslog fields
			syslog_port INT,

			-- Alarm fields
			alarm_protocol TEXT DEFAULT 'http' CHECK (alarm_protocol IN ('http', 'sip', 'xml', 'mqtt')),

			-- GB28181 fields (NEW)
			gb28181_device_id TEXT,
			gb28181_device_type TEXT,
			gb28181_parent_id TEXT,
			gb28181_sip_port INT,
			gb28181_realm TEXT,
			gb28181_register_expires INT,
			gb28181_last_register TIMESTAMPTZ,
			gb28181_channel_count INT DEFAULT 0,

			-- ONVIF fields (NEW)
			onvif_url TEXT,
			onvif_username TEXT,
			onvif_password TEXT,

			-- Metadata
			manufacturer TEXT,
			serial_number TEXT,
			mac_address TEXT,
			firmware_version TEXT,

			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create devices table: %w", err)
	}

	// 5. Telemetry (hypertable)
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS telemetry (
			time TIMESTAMPTZ NOT NULL,
			device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
			status TEXT,
			last_seen TIMESTAMPTZ,
			heartbeat_interval INT
		);
	`); err != nil {
		return fmt.Errorf("failed to create telemetry table: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT create_hypertable('telemetry', 'time', if_not_exists => TRUE);`); err != nil {
		db.Logger.Warn("Failed to create telemetry hypertable", "error", err)
	}

	// 6. Alarms (hypertable)
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS alarms (
			id BIGSERIAL,
			time TIMESTAMPTZ NOT NULL,
			device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
			priority INT,
			method INT,
			description TEXT,
			image_path TEXT,
			status TEXT DEFAULT 'active' CHECK (status IN ('active', 'acknowledged', 'resolved')),
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create alarms table: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT create_hypertable('alarms', 'time', if_not_exists => TRUE);`); err != nil {
		db.Logger.Warn("Failed to create alarms hypertable", "error", err)
	}

	// 7. Parsed logs (hypertable)
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS parsed_logs (
			id BIGSERIAL,
			time TIMESTAMPTZ NOT NULL,
			device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
			log_level TEXT,
			event_code INT,
			message TEXT,
			source TEXT,
			raw TEXT
		);
	`); err != nil {
		return fmt.Errorf("failed to create parsed_logs table: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT create_hypertable('parsed_logs', 'time', if_not_exists => TRUE);`); err != nil {
		db.Logger.Warn("Failed to create parsed_logs hypertable", "error", err)
	}

	// 8. Predictions (hypertable)
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS predictions (
			id BIGSERIAL,
			device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
			prediction_date TIMESTAMPTZ NOT NULL,
			failure_probability FLOAT,
			expected_remaining_hours INT,
			explanation TEXT,
			model_version TEXT
		);
	`); err != nil {
		return fmt.Errorf("failed to create predictions table: %w", err)
	}
	if _, err := tx.Exec(ctx, `SELECT create_hypertable('predictions', 'prediction_date', if_not_exists => TRUE);`); err != nil {
		db.Logger.Warn("Failed to create predictions hypertable", "error", err)
	}

	// 9. Tickets
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS tickets (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			description TEXT,
			device_id TEXT REFERENCES devices(device_id) ON DELETE SET NULL,
			priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
			status TEXT DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'pending', 'resolved', 'closed')),
			assignee TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create tickets table: %w", err)
	}

	// 10. Ticket comments
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS ticket_comments (
			id TEXT PRIMARY KEY,
			ticket_id TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
			user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
			user_name TEXT,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create ticket_comments table: %w", err)
	}

	// 11. Notifications
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
			title TEXT NOT NULL,
			message TEXT,
			type TEXT DEFAULT 'info' CHECK (type IN ('success', 'warning', 'error', 'info')),
			link TEXT,
			read BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create notifications table: %w", err)
	}

	// 12. Reports
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS reports (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			format TEXT DEFAULT 'xlsx' CHECK (format IN ('xlsx', 'pdf')),
			date_range TEXT,
			file_url TEXT,
			file_name TEXT,
			size TEXT,
			status TEXT DEFAULT 'ready' CHECK (status IN ('ready', 'expired', 'generating')),
			generated_by TEXT REFERENCES users(id) ON DELETE SET NULL,
			generated_at TIMESTAMPTZ DEFAULT NOW(),
			expires_at TIMESTAMPTZ
		);
	`); err != nil {
		return fmt.Errorf("failed to create reports table: %w", err)
	}

	// 13. System settings (с GB28181!)
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS system_settings (
			key TEXT PRIMARY KEY,
			value JSONB NOT NULL,
			description TEXT,
			updated_by TEXT REFERENCES users(id) ON DELETE SET NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create system_settings table: %w", err)
	}

	// Дефолтные настройки сервисов (с services_gb28181!)
	if _, err := tx.Exec(ctx, `
		INSERT INTO system_settings (key, value, description) VALUES 
		('services_syslog', '{"enabled": true, "udp_port": 1514, "tcp_port": 1514, "max_message_size": 65535, "parse_vendor": true}', 'Syslog receiver settings'),
		('services_ftp', '{"enabled": true, "port": 2121, "user": "alarm", "password": "alarm_pass", "root_path": "/var/lib/gb-telemetry/ftp", "passive_mode": true}', 'FTP server settings'),
		('services_snmp', '{"enabled": true, "port": 162, "community": "public", "version": "v2c"}', 'SNMP trap receiver settings'),
		('services_http', '{"enabled": true, "port": 8083, "require_auth": false}', 'HTTP log receiver settings'),
		('services_dahua', '{"enabled": true, "ports": [37777, 37778]}', 'Dahua protocol settings'),
		('services_hisilicon', '{"enabled": true, "port": 15002}', 'Hisilicon protocol settings'),
		('services_tvt', '{"enabled": true, "port": 15003}', 'TVT protocol settings'),
		('services_sip', '{"enabled": true, "port": 5060, "host": "0.0.0.0"}', 'Legacy SIP settings'),
		('services_gb28181', '{
			"enabled": true,
			"host": "0.0.0.0",
			"port": 5060,
			"server_id": "34020000002000000001",
			"server_ip": "",
			"realm": "3402000000",
			"auth_enabled": false,
			"auth_user": "admin",
			"auth_password": "",
			"auto_catalog": true,
			"auto_device_info": true,
			"keepalive_interval": 60,
			"keepalive_timeout": 180,
			"max_sub_channels": 64,
			"log_sip_messages": false
		}', 'GB/T 28181 SIP server settings'),
		('services_p2p_gateway', '{"url": "http://localhost:8082", "api_key": "your-secret-api-key-12345", "enabled": true}', 'P2P gateway connection settings')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return fmt.Errorf("failed to initialize system_settings: %w", err)
	}

	// 14. Audit log
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS audit_log (
			id BIGSERIAL PRIMARY KEY,
			timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
			action TEXT NOT NULL,
			entity_type TEXT,
			entity_id TEXT,
			old_value JSONB,
			new_value JSONB,
			ip_address TEXT,
			user_agent TEXT
		);
	`); err != nil {
		return fmt.Errorf("failed to create audit_log table: %w", err)
	}

	// 15. API keys
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL,
			permissions TEXT[],
			expires_at TIMESTAMPTZ,
			last_used_at TIMESTAMPTZ,
			created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create api_keys table: %w", err)
	}

	// 16. User sessions
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create user_sessions table: %w", err)
	}

	// 17. Indexes
	db.Logger.Info("Creating indexes...")
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_devices_site_id ON devices(site_id);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_vendor_type ON devices(vendor_type);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_owner_id ON devices(owner_id);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_connection_type ON devices(connection_type);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_gb28181_id ON devices(gb28181_device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_devices_gb28181_parent ON devices(gb28181_parent_id);`,

		`CREATE INDEX IF NOT EXISTS idx_telemetry_device_id ON telemetry(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_telemetry_time_device ON telemetry(time DESC, device_id);`,

		`CREATE INDEX IF NOT EXISTS idx_alarms_device_id ON alarms(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_alarms_time_device ON alarms(time DESC, device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_alarms_status ON alarms(status);`,
		`CREATE INDEX IF NOT EXISTS idx_alarms_priority ON alarms(priority);`,

		`CREATE INDEX IF NOT EXISTS idx_parsed_logs_device_id ON parsed_logs(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_parsed_logs_time_device ON parsed_logs(time DESC, device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_parsed_logs_level ON parsed_logs(log_level);`,
		`CREATE INDEX IF NOT EXISTS idx_parsed_logs_source ON parsed_logs(source);`,

		`CREATE INDEX IF NOT EXISTS idx_predictions_device_id ON predictions(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_predictions_date ON predictions(prediction_date DESC);`,

		`CREATE INDEX IF NOT EXISTS idx_tickets_device_id ON tickets(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);`,
		`CREATE INDEX IF NOT EXISTS idx_tickets_assignee ON tickets(assignee);`,

		`CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);`,
		`CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);`,

		`CREATE INDEX IF NOT EXISTS idx_reports_generated_by ON reports(generated_by);`,
		`CREATE INDEX IF NOT EXISTS idx_reports_generated_at ON reports(generated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_reports_expires_at ON reports(expires_at);`,

		`CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);`,

		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);`,
	}

	// 17.1. TOTP columns for 2FA
	if _, err := tx.Exec(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret TEXT;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN DEFAULT FALSE;
	`); err != nil {
		db.Logger.Warn("Failed to add TOTP columns", "error", err)
	}

	// 17.2. Telegram columns
	if _, err := tx.Exec(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_chat_id TEXT;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_alerts BOOLEAN DEFAULT FALSE;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_2fa BOOLEAN DEFAULT FALSE;
	`); err != nil {
		db.Logger.Warn("Failed to add Telegram columns", "error", err)
	}

	// 17.3. Telegram link tokens table
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS telegram_link_tokens (
			token TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_telegram_link_tokens_expires_at ON telegram_link_tokens(expires_at);
	`); err != nil {
		db.Logger.Warn("Failed to create telegram_link_tokens table", "error", err)
	}

	// 17.4. Telegram login codes table
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS telegram_login_codes (
			user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			code TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		db.Logger.Warn("Failed to create telegram_login_codes table", "error", err)
	}

	// Password reset tokens
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS password_reset_tokens (
			user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
			token TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		db.Logger.Warn("Failed to create password_reset_tokens table", "error", err)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// CMMS Tables (Maintenance Schedules, Work Orders, Spare Parts, SLA)
	// ═══════════════════════════════════════════════════════════════════════

	// 18. Maintenance Schedules
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS maintenance_schedules (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			device_id TEXT REFERENCES devices(device_id) ON DELETE CASCADE,
			schedule_type TEXT NOT NULL CHECK (schedule_type IN ('daily', 'weekly', 'monthly', 'quarterly', 'custom')),
			interval_days INT,
			custom_cron TEXT,
			last_completed TIMESTAMPTZ,
			next_due TIMESTAMPTZ NOT NULL,
			assigned_to TEXT REFERENCES users(id) ON DELETE SET NULL,
			checklist JSONB NOT NULL DEFAULT '[]',
			estimated_minutes INT DEFAULT 30,
			priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
			notes TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create maintenance_schedules table: %w", err)
	}

	// 19. Work Orders
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS work_orders (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			schedule_id TEXT REFERENCES maintenance_schedules(id) ON DELETE SET NULL,
			device_id TEXT REFERENCES devices(device_id) ON DELETE CASCADE,
			type TEXT NOT NULL CHECK (type IN ('preventive', 'corrective', 'emergency')),
			status TEXT DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'completed', 'cancelled')),
			priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
			assigned_to TEXT REFERENCES users(id) ON DELETE SET NULL,
			sla_deadline TIMESTAMPTZ,
			checklist JSONB NOT NULL DEFAULT '[]',
			started_at TIMESTAMPTZ,
			completed_at TIMESTAMPTZ,
			notes TEXT,
			photos JSONB DEFAULT '[]',
			parts_used JSONB DEFAULT '[]',
			created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create work_orders table: %w", err)
	}

	// 20. Technician Skills & Workload columns
	if _, err := tx.Exec(ctx, `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS skills TEXT[] DEFAULT '{}';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS max_workload INT DEFAULT 5;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS current_workload INT DEFAULT 0;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS base_location TEXT;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS certifications TEXT[] DEFAULT '{}';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS push_token TEXT;
		ALTER TABLE users ADD COLUMN IF NOT EXISTS push_platform TEXT;
	`); err != nil {
		db.Logger.Warn("Failed to add technician columns", "error", err)
	}

	// 21. SLA Configuration
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS sla_config (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			priority TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
			response_time_minutes INT NOT NULL,
			resolution_time_minutes INT NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create sla_config table: %w", err)
	}

	// 22. Spare Parts Inventory
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS spare_parts (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			name TEXT NOT NULL,
			sku TEXT UNIQUE,
			category TEXT,
			stock INT DEFAULT 0,
			min_stock INT DEFAULT 5,
			location TEXT,
			compatible_devices TEXT[],
			cost DECIMAL(10, 2),
			supplier TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create spare_parts table: %w", err)
	}

	// 23. Part Usage Log
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS part_usage (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			work_order_id TEXT REFERENCES work_orders(id) ON DELETE CASCADE,
			part_id TEXT REFERENCES spare_parts(id) ON DELETE CASCADE,
			quantity INT NOT NULL,
			used_by TEXT REFERENCES users(id) ON DELETE SET NULL,
			used_at TIMESTAMPTZ DEFAULT NOW()
		);
	`); err != nil {
		return fmt.Errorf("failed to create part_usage table: %w", err)
	}

	// CMMS Indexes
	cmmsIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_device ON maintenance_schedules(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_next_due ON maintenance_schedules(next_due);`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_assigned ON maintenance_schedules(assigned_to);`,
		`CREATE INDEX IF NOT EXISTS idx_work_orders_device ON work_orders(device_id);`,
		`CREATE INDEX IF NOT EXISTS idx_work_orders_status ON work_orders(status);`,
		`CREATE INDEX IF NOT EXISTS idx_work_orders_assigned ON work_orders(assigned_to);`,
		`CREATE INDEX IF NOT EXISTS idx_work_orders_sla ON work_orders(sla_deadline);`,
		`CREATE INDEX IF NOT EXISTS idx_spare_parts_sku ON spare_parts(sku);`,
		`CREATE INDEX IF NOT EXISTS idx_part_usage_work_order ON part_usage(work_order_id);`,
	}
	for _, idx := range cmmsIndexes {
		if _, err := tx.Exec(ctx, idx); err != nil {
			db.Logger.Warn("Failed to create CMMS index", "sql", idx, "error", err)
		}
	}

	// Default SLA config
	if _, err := tx.Exec(ctx, `
		INSERT INTO sla_config (priority, response_time_minutes, resolution_time_minutes) VALUES
		('critical', 15, 60),
		('high', 30, 240),
		('medium', 60, 480),
		('low', 240, 1440)
		ON CONFLICT DO NOTHING;
	`); err != nil {
		db.Logger.Warn("Failed to insert default SLA config", "error", err)
	}

	// Technician Site Assignments
	if _, err := tx.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS technician_site_assignments (
			id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
			technician_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			site_id TEXT NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
			is_primary BOOLEAN DEFAULT false,
			assigned_at TIMESTAMPTZ DEFAULT NOW(),
			assigned_by TEXT REFERENCES users(id) ON DELETE SET NULL,
			UNIQUE(technician_id, site_id)
		);
	`); err != nil {
		return fmt.Errorf("failed to create technician_site_assignments table: %w", err)
	}

	for _, idx := range indexes {
		if _, err := tx.Exec(ctx, idx); err != nil {
			db.Logger.Warn("Failed to create index", "sql", idx, "error", err)
		}
	}

	// 18. Retention policies
	retentionQueries := []string{
		`SELECT add_retention_policy('telemetry', INTERVAL '30 days', if_not_exists => TRUE);`,
		`SELECT add_retention_policy('alarms', INTERVAL '90 days', if_not_exists => TRUE);`,
		`SELECT add_retention_policy('parsed_logs', INTERVAL '30 days', if_not_exists => TRUE);`,
		`SELECT add_retention_policy('predictions', INTERVAL '365 days', if_not_exists => TRUE);`,
	}
	for _, query := range retentionQueries {
		if _, err := tx.Exec(ctx, query); err != nil {
			db.Logger.Warn("Failed to add retention policy", "sql", query, "error", err)
		}
	}

	// 19. Default admin
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return fmt.Errorf("failed to check users count: %w", err)
	}
	if count == 0 {
		hashed, err := auth.HashPassword("admin123")
		if err != nil {
			return fmt.Errorf("failed to hash default password: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (id, username, password_hash, role, email)
			VALUES (gen_random_uuid()::text, 'admin', $1, 'admin', 'admin@example.com')
		`, hashed); err != nil {
			return fmt.Errorf("failed to create default admin: %w", err)
		}
		db.Logger.Info("Default admin user created: admin / admin123")
	}

	// 20. Default site
	var sitesCount int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM sites`).Scan(&sitesCount); err == nil && sitesCount == 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO sites (id, name, address, city, status)
			VALUES ('site-default', 'Default Site', '', '', 'active')
		`); err != nil {
			db.Logger.Warn("Failed to create default site", "error", err)
		} else {
			db.Logger.Info("Default site created")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	db.Logger.Info("Database schema initialization completed successfully")
	return nil
}

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
