-- Migration 001: Initial schema (core tables + hypertables + indexes)
-- +migrate Up

-- 1. TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- 2. Users
CREATE TABLE users (
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

-- 3. Sites
CREATE TABLE sites (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    address TEXT,
    city TEXT,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'maintenance')),
    last_sync TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Devices
CREATE TABLE devices (
    device_id TEXT PRIMARY KEY,
    owner_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    site_id TEXT REFERENCES sites(id) ON DELETE SET NULL,
    name TEXT,
    location TEXT,
    latitude DOUBLE PRECISION DEFAULT 0,
    longitude DOUBLE PRECISION DEFAULT 0,
    geofence_radius_meters DOUBLE PRECISION DEFAULT 500,
    vendor_type TEXT,
    device_type TEXT DEFAULT 'camera' CHECK (device_type IN ('camera', 'nvr', 'dvr', 'switch')),
    status TEXT DEFAULT 'OFFLINE' CHECK (status IN ('ONLINE', 'OFFLINE', 'WARNING')),
    health TEXT DEFAULT 'healthy' CHECK (health IN ('healthy', 'faulty', 'degraded')),
    recording_status TEXT DEFAULT 'recording' CHECK (recording_status IN ('recording', 'not_recording', 'scheduled')),
    last_seen TIMESTAMPTZ,
    registered_at TIMESTAMPTZ DEFAULT NOW(),
    heartbeat_interval INT,
    user_agent TEXT,
    log_raw_data BOOLEAN DEFAULT TRUE,
    connection_type TEXT DEFAULT 'ip' CHECK (connection_type IN ('ip', 'p2p', 'snmp', 'syslog', 'alarm', 'gb28181', 'onvif')),
    p2p_brand TEXT,
    p2p_serial TEXT,
    p2p_security_code TEXT,
    p2p_cloud_user TEXT,
    p2p_cloud_pass TEXT,
    cloud_status TEXT DEFAULT 'unknown',
    snmp_community TEXT,
    snmp_version TEXT DEFAULT 'v2c' CHECK (snmp_version IN ('v1', 'v2c', 'v3')),
    syslog_port INT,
    alarm_protocol TEXT DEFAULT 'http' CHECK (alarm_protocol IN ('http', 'sip', 'xml', 'mqtt')),
    gb28181_device_id TEXT,
    gb28181_device_type TEXT,
    gb28181_parent_id TEXT,
    gb28181_sip_port INT,
    gb28181_realm TEXT,
    gb28181_register_expires INT,
    gb28181_last_register TIMESTAMPTZ,
    gb28181_channel_count INT DEFAULT 0,
    onvif_url TEXT,
    onvif_username TEXT,
    onvif_password TEXT,
    manufacturer TEXT,
    serial_number TEXT,
    mac_address TEXT,
    firmware_version TEXT,
    asset_class TEXT DEFAULT 'internal' CHECK (asset_class IN ('critical', 'confidential', 'internal', 'public')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 5. Telemetry (hypertable)
CREATE TABLE telemetry (
    time TIMESTAMPTZ NOT NULL,
    device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    status TEXT,
    last_seen TIMESTAMPTZ,
    heartbeat_interval INT
);
SELECT create_hypertable('telemetry', 'time', if_not_exists => TRUE);

-- 6. Alarms (hypertable)
CREATE TABLE alarms (
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
SELECT create_hypertable('alarms', 'time', if_not_exists => TRUE);

-- 7. Parsed logs (hypertable)
CREATE TABLE parsed_logs (
    id BIGSERIAL,
    time TIMESTAMPTZ NOT NULL,
    device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    log_level TEXT,
    event_code INT,
    message TEXT,
    source TEXT,
    raw TEXT
);
SELECT create_hypertable('parsed_logs', 'time', if_not_exists => TRUE);

-- 8. Predictions (hypertable)
CREATE TABLE predictions (
    id BIGSERIAL,
    device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    prediction_date TIMESTAMPTZ NOT NULL,
    failure_probability FLOAT,
    expected_remaining_hours INT,
    explanation TEXT,
    model_version TEXT
);
SELECT create_hypertable('predictions', 'prediction_date', if_not_exists => TRUE);

-- 9. Tickets
CREATE TABLE tickets (
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

-- 10. Ticket comments
CREATE TABLE ticket_comments (
    id TEXT PRIMARY KEY,
    ticket_id TEXT NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    user_name TEXT,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 11. Notifications
CREATE TABLE notifications (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    message TEXT,
    type TEXT DEFAULT 'info' CHECK (type IN ('success', 'warning', 'error', 'info')),
    link TEXT,
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 12. Reports
CREATE TABLE reports (
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

-- 13. System settings
CREATE TABLE system_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Default service settings
INSERT INTO system_settings (key, value, description) VALUES
('services_syslog', '{"enabled": true, "udp_port": 1514, "tcp_port": 1514, "max_message_size": 65535, "parse_vendor": true}', 'Syslog receiver settings'),
('services_ftp', '{"enabled": true, "port": 2121, "user": "alarm", "password": "alarm_pass", "root_path": "/var/lib/gb-telemetry/ftp", "passive_mode": true}', 'FTP server settings'),
('services_snmp', '{"enabled": true, "port": 162, "community": "public", "version": "v2c"}', 'SNMP trap receiver settings'),
('services_http', '{"enabled": true, "port": 8083, "require_auth": false}', 'HTTP log receiver settings'),
('services_dahua', '{"enabled": true, "ports": [37777, 37778]}', 'Dahua protocol settings'),
('services_hisilicon', '{"enabled": true, "port": 15002}', 'Hisilicon protocol settings'),
('services_tvt', '{"enabled": true, "port": 15003}', 'TVT protocol settings'),
('services_sip', '{"enabled": true, "port": 5060, "host": "0.0.0.0"}', 'Legacy SIP settings'),
('services_gb28181', '{"enabled": true, "host": "0.0.0.0", "port": 5060, "server_id": "34020000002000000001", "server_ip": "", "realm": "3402000000", "auth_enabled": false, "auth_user": "admin", "auth_password": "", "auto_catalog": true, "auto_device_info": true, "keepalive_interval": 60, "keepalive_timeout": 180, "max_sub_channels": 64, "log_sip_messages": false}', 'GB/T 28181 SIP server settings'),
('services_p2p_gateway', '{"url": "http://localhost:8082", "api_key": "your-secret-api-key-12345", "enabled": true}', 'P2P gateway connection settings')
ON CONFLICT (key) DO NOTHING;

-- 14. Audit log
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    entity_type TEXT,
    entity_id TEXT,
    old_value JSONB,
    new_value JSONB,
    ip_address TEXT,
    user_agent TEXT,
    hmac_signature TEXT
);

-- 15. API keys
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    key_prefix TEXT NOT NULL DEFAULT '',
    permissions TEXT[],
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 16. User sessions
CREATE TABLE user_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 17. TOTP + Telegram + Password reset
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_chat_id TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_alerts BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS telegram_2fa BOOLEAN DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS skills TEXT[] DEFAULT '{}';
ALTER TABLE users ADD COLUMN IF NOT EXISTS max_workload INT DEFAULT 5;
ALTER TABLE users ADD COLUMN IF NOT EXISTS current_workload INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS base_location TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS certifications TEXT[] DEFAULT '{}';
ALTER TABLE users ADD COLUMN IF NOT EXISTS push_token TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS push_platform TEXT;

CREATE TABLE telegram_link_tokens (
    token TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE telegram_login_codes (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE password_reset_tokens (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 18. CMMS: Maintenance Schedules
CREATE TABLE maintenance_schedules (
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

-- 19. CMMS: Work Orders
CREATE TABLE work_orders (
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

-- 20. CMMS: SLA Config
CREATE TABLE sla_config (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    priority TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    response_time_minutes INT NOT NULL,
    resolution_time_minutes INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 21. CMMS: Spare Parts
CREATE TABLE spare_parts (
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

-- 22. CMMS: Part Usage
CREATE TABLE part_usage (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT REFERENCES work_orders(id) ON DELETE CASCADE,
    part_id TEXT REFERENCES spare_parts(id) ON DELETE CASCADE,
    quantity INT NOT NULL,
    used_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    used_at TIMESTAMPTZ DEFAULT NOW()
);

-- 23. External Work Order Status (ITSM sync)
CREATE TABLE external_work_order_status (
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('servicenow', 'jira', 'toir')),
    status TEXT,
    priority TEXT,
    summary TEXT,
    external_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    changes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS external_id TEXT;
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS external_source TEXT CHECK (external_source IN ('servicenow', 'jira', 'toir')) DEFAULT NULL;

-- Default data
INSERT INTO sla_config (priority, response_time_minutes, resolution_time_minutes) VALUES
('critical', 15, 60),
('high', 30, 240),
('medium', 60, 480),
('low', 240, 1440)
ON CONFLICT DO NOTHING;

INSERT INTO sites (id, name, address, city, status)
VALUES ('site-default', 'Default Site', '', '', 'active')
ON CONFLICT (id) DO NOTHING;

-- Retention policies
SELECT add_retention_policy('telemetry', INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('alarms', INTERVAL '90 days', if_not_exists => TRUE);
SELECT add_retention_policy('parsed_logs', INTERVAL '30 days', if_not_exists => TRUE);
SELECT add_retention_policy('predictions', INTERVAL '365 days', if_not_exists => TRUE);

-- ============================================================
-- INDEXES
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_devices_site_id ON devices(site_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_vendor_type ON devices(vendor_type);
CREATE INDEX IF NOT EXISTS idx_devices_owner_id ON devices(owner_id);
CREATE INDEX IF NOT EXISTS idx_devices_connection_type ON devices(connection_type);
CREATE INDEX IF NOT EXISTS idx_devices_gb28181_id ON devices(gb28181_device_id);
CREATE INDEX IF NOT EXISTS idx_devices_gb28181_parent ON devices(gb28181_parent_id);

CREATE INDEX IF NOT EXISTS idx_telemetry_device_id ON telemetry(device_id);
CREATE INDEX IF NOT EXISTS idx_telemetry_time_device ON telemetry(time DESC, device_id);

CREATE INDEX IF NOT EXISTS idx_alarms_device_id ON alarms(device_id);
CREATE INDEX IF NOT EXISTS idx_alarms_time_device ON alarms(time DESC, device_id);
CREATE INDEX IF NOT EXISTS idx_alarms_status ON alarms(status);
CREATE INDEX IF NOT EXISTS idx_alarms_priority ON alarms(priority);

CREATE INDEX IF NOT EXISTS idx_parsed_logs_device_id ON parsed_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_parsed_logs_time_device ON parsed_logs(time DESC, device_id);
CREATE INDEX IF NOT EXISTS idx_parsed_logs_level ON parsed_logs(log_level);
CREATE INDEX IF NOT EXISTS idx_parsed_logs_source ON parsed_logs(source);

CREATE INDEX IF NOT EXISTS idx_predictions_device_id ON predictions(device_id);
CREATE INDEX IF NOT EXISTS idx_predictions_date ON predictions(prediction_date DESC);

CREATE INDEX IF NOT EXISTS idx_tickets_device_id ON tickets(device_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee ON tickets(assignee);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_read ON notifications(read);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_reports_generated_by ON reports(generated_by);
CREATE INDEX IF NOT EXISTS idx_reports_generated_at ON reports(generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_reports_expires_at ON reports(expires_at);

CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp ON audit_log(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log(action);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at ON user_sessions(expires_at);

CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_telegram_link_tokens_expires_at ON telegram_link_tokens(expires_at);

CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_device ON maintenance_schedules(device_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_next_due ON maintenance_schedules(next_due);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_assigned ON maintenance_schedules(assigned_to);
CREATE INDEX IF NOT EXISTS idx_work_orders_device ON work_orders(device_id);
CREATE INDEX IF NOT EXISTS idx_work_orders_status ON work_orders(status);
CREATE INDEX IF NOT EXISTS idx_work_orders_assigned ON work_orders(assigned_to);
CREATE INDEX IF NOT EXISTS idx_work_orders_sla ON work_orders(sla_deadline);
CREATE INDEX IF NOT EXISTS idx_spare_parts_sku ON spare_parts(sku);
CREATE INDEX IF NOT EXISTS idx_part_usage_work_order ON part_usage(work_order_id);

CREATE INDEX IF NOT EXISTS idx_ext_wo_status_external ON external_work_order_status(external_id, source);
CREATE INDEX IF NOT EXISTS idx_ext_wo_status_changed ON external_work_order_status(external_changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_work_orders_external ON work_orders(external_id, external_source);