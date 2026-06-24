-- +migrate Up
-- Migration 010: Advanced SLA Engine
--
-- SLA-6.1.x: Замена плоской SLAConfig на enterprise SLA-движок.
--
-- Compliance:
--   IEC 62443 SR 7.1 (Resource availability)
--   ISO 27001 A.12.6.1 (Capacity management)

-- ── SLA Policies (Standard/Premium/24×7) ─────────────────────────────

CREATE TABLE sla_policies (
    id              VARCHAR(64) PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    type            VARCHAR(16) NOT NULL CHECK (type IN ('standard', 'premium', '24x7')),
    description     TEXT DEFAULT '',
    is_default      BOOLEAN NOT NULL DEFAULT false,
    work_start_hour INTEGER NOT NULL DEFAULT 9 CHECK (work_start_hour >= 0 AND work_start_hour <= 23),
    work_end_hour   INTEGER NOT NULL DEFAULT 18 CHECK (work_end_hour >= 0 AND work_end_hour <= 23),
    work_days       INTEGER[] NOT NULL DEFAULT '{1,2,3,4,5}', -- time.Weekday: 0=Sun, 1=Mon, ..., 6=Sat
    response_time_minutes    INTEGER NOT NULL DEFAULT 120 CHECK (response_time_minutes > 0),
    resolution_time_minutes  INTEGER NOT NULL DEFAULT 960 CHECK (resolution_time_minutes > 0),
    escalation_1_after_minutes INTEGER DEFAULT 240,
    escalation_2_after_minutes INTEGER DEFAULT 480,
    escalation_3_after_minutes INTEGER DEFAULT 1440,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── SLA Matrix (Priority × Impact) ───────────────────────────────────

CREATE TABLE sla_matrix_entries (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id               VARCHAR(64) NOT NULL REFERENCES sla_policies(id) ON DELETE CASCADE,
    priority                VARCHAR(16) NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    impact                  VARCHAR(16) NOT NULL CHECK (impact IN ('extensive', 'significant', 'limited', 'minor')),
    response_time_minutes   INTEGER NOT NULL CHECK (response_time_minutes > 0),
    resolution_time_minutes INTEGER NOT NULL CHECK (resolution_time_minutes > 0),
    escalation_1_minutes    INTEGER DEFAULT 0,
    escalation_2_minutes    INTEGER DEFAULT 0,
    escalation_3_minutes    INTEGER DEFAULT 0,
    UNIQUE (policy_id, priority, impact)
);

-- ── Business Calendars ───────────────────────────────────────────────

CREATE TABLE sla_business_calendars (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id         VARCHAR(64) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    timezone        VARCHAR(64) NOT NULL DEFAULT 'UTC',
    work_start_hour INTEGER NOT NULL DEFAULT 9,
    work_end_hour   INTEGER NOT NULL DEFAULT 18,
    work_days       INTEGER[] NOT NULL DEFAULT '{1,2,3,4,5}',
    holidays        JSONB DEFAULT '[]'::jsonb,   -- [{date, name, recurring, half_day}]
    exceptions      JSONB DEFAULT '[]'::jsonb,   -- [{date, description, work_start, work_end}]
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── SLA Pause Rules ──────────────────────────────────────────────────

CREATE TABLE sla_pause_rules (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id   VARCHAR(64) NOT NULL REFERENCES sla_policies(id) ON DELETE CASCADE,
    status      VARCHAR(32) NOT NULL,
    description TEXT DEFAULT '',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    UNIQUE (policy_id, status)
);

-- ── SLA Tracker State (in-memory, но для persistence) ────────────────

CREATE TABLE sla_tracker_state (
    work_order_id               VARCHAR(64) PRIMARY KEY,
    policy_id                   VARCHAR(64) NOT NULL,
    priority                    VARCHAR(16) NOT NULL,
    impact                      VARCHAR(16) DEFAULT '',
    status                      VARCHAR(16) NOT NULL DEFAULT 'on_track',
    escalation_level            INTEGER NOT NULL DEFAULT 0,
    is_paused                   BOOLEAN NOT NULL DEFAULT false,
    total_pause_ms              BIGINT NOT NULL DEFAULT 0,
    response_target_minutes     INTEGER NOT NULL DEFAULT 0,
    resolution_target_minutes   INTEGER NOT NULL DEFAULT 0,
    response_deadline           TIMESTAMPTZ,
    resolution_deadline         TIMESTAMPTZ,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_sla_matrix_policy ON sla_matrix_entries (policy_id);
CREATE INDEX IF NOT EXISTS idx_sla_calendar_site ON sla_business_calendars (site_id);
CREATE INDEX IF NOT EXISTS idx_sla_pause_policy ON sla_pause_rules (policy_id);
CREATE INDEX IF NOT EXISTS idx_sla_tracker_status ON sla_tracker_state (status);
CREATE INDEX IF NOT EXISTS idx_sla_tracker_deadline ON sla_tracker_state (resolution_deadline);

-- Комментарии
COMMENT ON TABLE sla_policies IS 'SLA политики: Standard, Premium, 24×7';
COMMENT ON TABLE sla_matrix_entries IS 'Матрица SLA: Priority × Impact → targets';
COMMENT ON TABLE sla_business_calendars IS 'Бизнес-календари per site (timezone, shifts, holidays)';
COMMENT ON TABLE sla_pause_rules IS 'Правила приостановки SLA (ON_HOLD, AWAITING_*)';
COMMENT ON TABLE sla_tracker_state IS 'Состояние SLA трекера для Work Order';

-- Seed default policies + matrix
INSERT INTO sla_policies (id, name, type, is_default, work_start_hour, work_end_hour, work_days, response_time_minutes, resolution_time_minutes, escalation_1_after_minutes, escalation_2_after_minutes, escalation_3_after_minutes) VALUES
    ('sla-std', 'Standard', 'standard', true, 9, 18, '{1,2,3,4,5}', 120, 960, 240, 480, 1440),
    ('sla-prem', 'Premium', 'premium', false, 7, 22, '{1,2,3,4,5,6}', 30, 240, 60, 180, 480),
    ('sla-247', '24×7', '24x7', false, 0, 23, '{0,1,2,3,4,5,6}', 15, 60, 30, 90, 180)
ON CONFLICT (id) DO NOTHING;

-- Pause rules for standard policy
INSERT INTO sla_pause_rules (policy_id, status, description) VALUES
    ('sla-std', 'ON_HOLD', 'WO on hold by dispatcher'),
    ('sla-std', 'AWAITING_PARTS', 'Waiting for spare parts'),
    ('sla-std', 'AWAITING_VENDOR', 'Waiting for vendor'),
    ('sla-std', 'AWAITING_CLIENT', 'Waiting for client response')
ON CONFLICT (policy_id, status) DO NOTHING;

INSERT INTO sla_pause_rules (policy_id, status, description) VALUES
    ('sla-prem', 'ON_HOLD', 'WO on hold by dispatcher'),
    ('sla-prem', 'AWAITING_PARTS', 'Waiting for spare parts'),
    ('sla-prem', 'AWAITING_VENDOR', 'Waiting for vendor'),
    ('sla-prem', 'AWAITING_CLIENT', 'Waiting for client response')
ON CONFLICT (policy_id, status) DO NOTHING;

INSERT INTO sla_pause_rules (policy_id, status, description) VALUES
    ('sla-247', 'ON_HOLD', 'WO on hold by dispatcher'),
    ('sla-247', 'AWAITING_PARTS', 'Waiting for spare parts'),
    ('sla-247', 'AWAITING_VENDOR', 'Waiting for vendor'),
    ('sla-247', 'AWAITING_CLIENT', 'Waiting for client response')
ON CONFLICT (policy_id, status) DO NOTHING;
