-- +migrate Up
-- Migration 011: Meter entities + TimescaleDB hypertable
--
-- AH-5.3.1: Meter — метрики CCTV-устройств
-- AH-5.3.2: meter_readings — TimescaleDB hypertable
-- AH-5.3.3: meter_triggers — правила создания WO по метрикам
--
-- Compliance:
--   IEC 62443 SR 7.1 (Resource availability)
--   ISO 27001 A.12.6.1 (Capacity management)

-- ── Meters ───────────────────────────────────────────────────────────

CREATE TABLE meters (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id         VARCHAR(64) NOT NULL,
    kind              VARCHAR(32) NOT NULL,
    name              VARCHAR(100) NOT NULL,
    unit              VARCHAR(16) DEFAULT '',
    interval_seconds  INTEGER NOT NULL DEFAULT 60,
    retention_days    INTEGER NOT NULL DEFAULT 90,
    thresholds        JSONB DEFAULT '{"warning":0,"critical":0,"min":0,"max":0}'::jsonb,
    enabled           BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (device_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_meters_device ON meters (device_id);
CREATE INDEX IF NOT EXISTS idx_meters_kind ON meters (kind);

-- ── Meter Readings (TimescaleDB hypertable) ─────────────────────────

-- Создаём обычную таблицу, TimescaleDB hypertable создаётся отдельно
CREATE TABLE meter_readings (
    time        TIMESTAMPTZ NOT NULL,
    meter_id    UUID NOT NULL REFERENCES meters(id) ON DELETE CASCADE,
    device_id   VARCHAR(64) NOT NULL,
    kind        VARCHAR(32) NOT NULL,
    value       DOUBLE PRECISION NOT NULL,
    tags        JSONB DEFAULT '[]'::jsonb
);

-- Индексы для TimescaleDB
CREATE INDEX IF NOT EXISTS idx_readings_meter_time ON meter_readings (meter_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_readings_device_time ON meter_readings (device_id, time DESC);
CREATE INDEX IF NOT EXISTS idx_readings_kind_time ON meter_readings (kind, time DESC);

-- ⚠ TimescaleDB hypertable создаётся через отдельную команду:
-- SELECT create_hypertable('meter_readings', 'time', if_not_exists => TRUE);
-- Это должно выполняться в post-init скрипте или через API.

-- ── Meter Triggers ──────────────────────────────────────────────────

CREATE TABLE meter_triggers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              VARCHAR(100) NOT NULL,
    enabled           BOOLEAN NOT NULL DEFAULT true,
    meter_kind        VARCHAR(32) NOT NULL,
    condition         VARCHAR(16) NOT NULL,
    threshold         DOUBLE PRECISION NOT NULL DEFAULT 0,
    duration_seconds  INTEGER NOT NULL DEFAULT 0,
    device_ids        JSONB DEFAULT '[]'::jsonb,
    cooldown_minutes  INTEGER NOT NULL DEFAULT 60,
    action            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_triggers_kind ON meter_triggers (meter_kind);
CREATE INDEX IF NOT EXISTS idx_triggers_enabled ON meter_triggers (enabled);

-- ── Trigger Fired Log ───────────────────────────────────────────────

CREATE TABLE meter_trigger_fired (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger_id      UUID NOT NULL REFERENCES meter_triggers(id) ON DELETE CASCADE,
    meter_id        UUID NOT NULL REFERENCES meters(id),
    device_id       VARCHAR(64) NOT NULL,
    reading_value   DOUBLE PRECISION NOT NULL,
    fired_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    wo_created      BOOLEAN NOT NULL DEFAULT false,
    wo_id           VARCHAR(64)
);

CREATE INDEX IF NOT EXISTS idx_fired_trigger ON meter_trigger_fired (trigger_id, fired_at DESC);

-- Seed default triggers
INSERT INTO meter_triggers (name, enabled, meter_kind, condition, threshold, duration_seconds, cooldown_minutes, action) VALUES
    ('CPU Overheating', true, 'cpu_temp', 'gt', 85, 600, 120,
     '{"work_order_type":"preventive","priority":"high","title_template":"CPU overheating on {device_name}","desc_template":"CPU temperature on {device_name} is {value}°C (threshold: 85°C)","auto_approve":true}'::jsonb),
    ('High Packet Loss', true, 'packet_loss', 'gt', 5, 300, 60,
     '{"work_order_type":"corrective","priority":"high","title_template":"High packet loss on {device_name}","desc_template":"Packet loss on {device_name} is {value}% (threshold: 5%)","auto_approve":true}'::jsonb),
    ('Low Frame Rate', true, 'fps', 'lt', 10, 600, 120,
     '{"work_order_type":"corrective","priority":"medium","title_template":"Low frame rate on {device_name}","desc_template":"Frame rate on {device_name} is {value} FPS (threshold: 10 FPS)","auto_approve":false}'::jsonb),
    ('NVR Disk Almost Full', true, 'disk_usage', 'gt', 90, 1800, 1440,
     '{"work_order_type":"preventive","priority":"high","title_template":"NVR disk almost full on {device_name}","desc_template":"NVR disk usage is {value}% (threshold: 90%)","auto_approve":true}'::jsonb)
ON CONFLICT DO NOTHING;

COMMENT ON TABLE meters IS 'CCTV метрики: bitrate, fps, cpu_temp, error_count, offline_ratio и др.';
COMMENT ON TABLE meter_readings IS 'TimescaleDB hypertable: значения метрик во времени';
COMMENT ON TABLE meter_triggers IS 'Правила: условие по метрике → создание WorkOrder';
COMMENT ON TABLE meter_trigger_fired IS 'Лог срабатываний триггеров';
