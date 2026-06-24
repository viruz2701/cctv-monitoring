-- Migration 008: Epic 1 — Domain Model Evolution
-- Adds: WorkOrderBase fields, WorkOrderHistory, WorkOrderRelations,
--       Requests, PreventiveMaintenance, TimeEntries, Costs, PartsConsumption
--
-- Compliance: IEC 62443 SL-3, ISO 27001 A.12.4, СТБ 34.101.27, OWASP ASVS V5
-- Ref: Grash CMMS Domain Model
-- +migrate Up

-- ============================================================
-- 1. WorkOrder enhancements
-- ============================================================

ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS title TEXT DEFAULT '';
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS due_date TIMESTAMPTZ;
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS deleted_by TEXT REFERENCES users(id) ON DELETE SET NULL;

-- Update CHECK constraint for status to support 12-status model
ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_status_check;
ALTER TABLE work_orders ADD CONSTRAINT work_orders_status_check
    CHECK (status IN (
        'REQUESTED', 'APPROVED', 'OPEN', 'IN_PROGRESS', 'ON_HOLD',
        'AWAITING_PARTS', 'AWAITING_VENDOR', 'AWAITING_CLIENT',
        'COMPLETED', 'VERIFIED', 'CLOSED', 'REJECTED',
        'open', 'in_progress', 'completed', 'cancelled'  -- backward compat
    ));

-- Update CHECK constraint for type
ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_type_check;
ALTER TABLE work_orders ADD CONSTRAINT work_orders_type_check
    CHECK (type IN ('preventive', 'corrective', 'emergency', 'routine', 'inspection'));

-- Update CHECK constraint for priority
ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_priority_check;
ALTER TABLE work_orders ADD CONSTRAINT work_orders_priority_check
    CHECK (priority IN ('critical', 'high', 'medium', 'low'));

-- Index for soft-delete queries
CREATE INDEX IF NOT EXISTS idx_work_orders_deleted_at ON work_orders(deleted_at);
CREATE INDEX IF NOT EXISTS idx_work_orders_due_date ON work_orders(due_date);

-- ============================================================
-- 2. WorkOrder History (immutable timeline)
-- ============================================================

CREATE TABLE work_order_history (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    from_status TEXT NOT NULL,
    to_status TEXT NOT NULL,
    changed_by TEXT NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    comment TEXT,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    prev_hash TEXT NOT NULL DEFAULT '',  -- СТБ bash-256 HMAC chain (tamper detection)
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wo_history_work_order ON work_order_history(work_order_id);
CREATE INDEX IF NOT EXISTS idx_wo_history_changed_at ON work_order_history(changed_at DESC);
CREATE INDEX IF NOT EXISTS idx_wo_history_changed_by ON work_order_history(changed_by);

-- ============================================================
-- 3. WorkOrder Relations (parent/child, blocked_by, etc.)
-- ============================================================

CREATE TABLE work_order_relations (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    source_wo_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    target_wo_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    relation_type TEXT NOT NULL CHECK (relation_type IN (
        'PARENT_CHILD', 'BLOCKED_BY', 'DUPLICATE_OF', 'SPLIT_TO', 'RELATED_TO'
    )),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(source_wo_id, target_wo_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_wo_relations_source ON work_order_relations(source_wo_id);
CREATE INDEX IF NOT EXISTS idx_wo_relations_target ON work_order_relations(target_wo_id);

-- ============================================================
-- 4. WorkOrder ↔ Alert M2M
-- ============================================================

-- Note: без FK на alarms(id) — alarms это TimescaleDB hypertable,
-- а TimescaleDB не поддерживает FOREIGN KEY на hypertables.
-- Ссылочная целостность обеспечивается на уровне приложения.

CREATE TABLE work_order_alerts (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    alert_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(work_order_id, alert_id)
);

CREATE INDEX IF NOT EXISTS idx_wo_alerts_work_order ON work_order_alerts(work_order_id);
CREATE INDEX IF NOT EXISTS idx_wo_alerts_alert ON work_order_alerts(alert_id);

-- ============================================================
-- 5. Requests (заявки от пользователей)
-- ============================================================

CREATE TABLE requests (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    site_id TEXT REFERENCES sites(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT,
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    status TEXT DEFAULT 'REQUESTED' CHECK (status IN (
        'REQUESTED', 'APPROVED', 'REJECTED', 'CONVERTED'
    )),
    assignee TEXT REFERENCES users(id) ON DELETE SET NULL,
    due_date TIMESTAMPTZ,
    contact_name TEXT,
    contact_email TEXT,
    contact_phone TEXT,
    source TEXT DEFAULT 'portal' CHECK (source IN ('portal', 'email', 'phone', 'telegram', 'api')),
    converted_to_wo TEXT REFERENCES work_orders(id) ON DELETE SET NULL,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_requests_device ON requests(device_id);
CREATE INDEX IF NOT EXISTS idx_requests_status ON requests(status);
CREATE INDEX IF NOT EXISTS idx_requests_created_at ON requests(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_requests_source ON requests(source);

-- ============================================================
-- 6. Preventive Maintenance
-- ============================================================

CREATE TABLE preventive_maintenance (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    device_id TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    priority TEXT DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    status TEXT DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'INACTIVE', 'COMPLETED')),
    assignee TEXT REFERENCES users(id) ON DELETE SET NULL,
    due_date TIMESTAMPTZ,
    schedule_type TEXT NOT NULL CHECK (schedule_type IN ('daily', 'weekly', 'monthly', 'quarterly', 'custom')),
    interval_days INT DEFAULT 0,
    custom_cron TEXT,
    last_completed TIMESTAMPTZ,
    next_due TIMESTAMPTZ NOT NULL,
    checklist JSONB DEFAULT '[]',
    estimated_minutes INT DEFAULT 30,
    notes TEXT,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_pm_device ON preventive_maintenance(device_id);
CREATE INDEX IF NOT EXISTS idx_pm_next_due ON preventive_maintenance(next_due);
CREATE INDEX IF NOT EXISTS idx_pm_assignee ON preventive_maintenance(assignee);

-- ============================================================
-- 7. Time Entries
-- ============================================================

CREATE TABLE time_entries (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paused_at TIMESTAMPTZ,
    resumed_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    total_pause_seconds INT DEFAULT 0,
    duration_minutes INT DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_time_entries_wo ON time_entries(work_order_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_user ON time_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_time_entries_started ON time_entries(started_at DESC);

-- ============================================================
-- 8. Labor Costs
-- ============================================================

CREATE TABLE labor_costs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    technician_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hourly_rate DECIMAL(10, 2) NOT NULL DEFAULT 0,
    hours_worked DECIMAL(6, 2) NOT NULL DEFAULT 0,
    estimated_cost DECIMAL(12, 2) DEFAULT 0,
    actual_cost DECIMAL(12, 2) DEFAULT 0,
    currency TEXT DEFAULT 'USD',
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_labor_costs_wo ON labor_costs(work_order_id);

-- ============================================================
-- 9. Additional Costs
-- ============================================================

CREATE TABLE additional_costs (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    category TEXT NOT NULL CHECK (category IN ('travel', 'subcontractor', 'permit', 'equipment', 'other')),
    description TEXT,
    estimated_cost DECIMAL(12, 2) DEFAULT 0,
    actual_cost DECIMAL(12, 2) DEFAULT 0,
    currency TEXT DEFAULT 'USD',
    vendor_name TEXT,
    receipt_url TEXT,
    created_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_additional_costs_wo ON additional_costs(work_order_id);
CREATE INDEX IF NOT EXISTS idx_additional_costs_category ON additional_costs(category);

-- ============================================================
-- 10. Parts Consumption (с cost snapshot)
-- ============================================================

CREATE TABLE parts_consumption (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    part_id TEXT NOT NULL REFERENCES spare_parts(id) ON DELETE CASCADE,
    quantity INT NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(10, 2) NOT NULL DEFAULT 0,
    total_price DECIMAL(12, 2) NOT NULL DEFAULT 0,
    estimated_cost DECIMAL(12, 2) DEFAULT 0,
    actual_cost DECIMAL(12, 2) DEFAULT 0,
    currency TEXT DEFAULT 'USD',
    used_by TEXT REFERENCES users(id) ON DELETE SET NULL,
    used_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_parts_consumption_wo ON parts_consumption(work_order_id);
CREATE INDEX IF NOT EXISTS idx_parts_consumption_part ON parts_consumption(part_id);
CREATE INDEX IF NOT EXISTS idx_parts_consumption_used_at ON parts_consumption(used_at DESC);
