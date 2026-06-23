-- Migration 006: ISO 27001:2022 Controls
-- A.8 Asset Management, A.9 Access Control, A.12 Operations Security
-- Соответствует: ISO 27001:2022 A.8.1, A.9.2, A.9.4, A.12.4
-- +migrate Up

-- ============================================================
-- A.8.1 Inventory of Assets
-- ============================================================

ALTER TABLE devices ADD COLUMN IF NOT EXISTS asset_category VARCHAR(50);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS asset_criticality VARCHAR(20)
    CHECK (asset_criticality IN ('critical', 'high', 'medium', 'low'));
ALTER TABLE devices ADD COLUMN IF NOT EXISTS asset_owner VARCHAR(100);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS asset_location_physical VARCHAR(200);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS asset_tag TEXT UNIQUE;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS asset_status VARCHAR(20) DEFAULT 'active'
    CHECK (asset_status IN ('active', 'in_maintenance', 'decommissioned', 'lost', 'stolen'));
ALTER TABLE devices ADD COLUMN IF NOT EXISTS warranty_expires_at TIMESTAMPTZ;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS purchase_date DATE;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS purchase_cost DECIMAL(12, 2);
ALTER TABLE devices ADD COLUMN IF NOT EXISTS vendor_contact TEXT;

-- A.8.1: Создаём индекс для поиска по asset_tag
CREATE INDEX IF NOT EXISTS idx_devices_asset_tag ON devices(asset_tag) WHERE asset_tag IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_devices_asset_criticality ON devices(asset_criticality);
CREATE INDEX IF NOT EXISTS idx_devices_asset_status ON devices(asset_status);

-- ============================================================
-- A.9.2 User Registration & De-registration
-- A.9.4 System & Application Access Control
-- ============================================================

-- Расширяем таблицу users для A.9.4
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_activity_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS session_timeout_minutes INT DEFAULT 30;
ALTER TABLE users ADD COLUMN IF NOT EXISTS max_concurrent_sessions INT DEFAULT 3;
ALTER TABLE users ADD COLUMN IF NOT EXISTS allowed_ips TEXT[] DEFAULT '{}';

-- Таблица для user approval workflow (A.9.2)
CREATE TABLE user_approval_queue (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    requested_role TEXT NOT NULL,
    requested_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    approved_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_approval_queue_status ON user_approval_queue(status);
CREATE INDEX IF NOT EXISTS idx_user_approval_queue_user ON user_approval_queue(user_id);

-- ============================================================
-- A.12.4 Logging & Monitoring — расширение audit_log
-- ============================================================

-- Добавляем hash chain для tamper-proof audit log (ISO 27001 A.12.4, СТБ 34.101.27)
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS prev_hash TEXT;
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS trace_id TEXT;
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS source_service TEXT DEFAULT 'backend';

-- Индекс для поиска по trace_id
CREATE INDEX IF NOT EXISTS idx_audit_log_trace_id ON audit_log(trace_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id);

-- ============================================================
-- A.12.6 Vulnerability Management
-- ============================================================

CREATE TABLE vulnerability_scans (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    scan_type TEXT NOT NULL CHECK (scan_type IN ('govulncheck', 'npm_audit', 'dependabot', 'manual')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed')),
    vulnerabilities_found INT DEFAULT 0,
    critical_count INT DEFAULT 0,
    high_count INT DEFAULT 0,
    medium_count INT DEFAULT 0,
    low_count INT DEFAULT 0,
    report JSONB,
    scanned_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vulnerability_scans_type ON vulnerability_scans(scan_type);
CREATE INDEX IF NOT EXISTS idx_vulnerability_scans_status ON vulnerability_scans(status);

-- ============================================================
-- A.14.2 Security in Development — CSRF tokens table
-- ============================================================

CREATE TABLE csrf_tokens (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_csrf_tokens_user ON csrf_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_token ON csrf_tokens(token);
CREATE INDEX IF NOT EXISTS idx_csrf_tokens_expires ON csrf_tokens(expires_at);
