-- +migrate Up
-- Migration 028: Black Box Incident Recorder (KF-15.2.4)
--
-- Автоматический сбор "пакета доказательств" при инцидентах.
-- Хранит снимки состояния устройств, логи, тревоги, статус записи,
-- даунтайм и SLA-данные в одном месте.
--
-- Compliance:
--   - IEC 62443 SR 7.1: Resource availability — evidence collection
--   - ISO 27001 A.12.4: Audit trail — неизменяемый журнал инцидентов
--   - ISO 27019 PCC.A.12: Incident management for ICS
--   - СТБ 34.101.27 п. 6.4: Регистрация инцидентов безопасности
--   - OWASP ASVS V7.1: Error handling — structured error evidence

-- ═══════════════════════════════════════════════════════════════════
-- 1. Таблица incident_reports — основной пакет доказательств
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS incident_reports (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    device_id       TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    site_id         TEXT,
    triggered_by    TEXT NOT NULL CHECK (triggered_by IN ('alarm', 'manual', 'sla_breach', 'downtime')),
    trigger_ref     TEXT NOT NULL DEFAULT '',
    "timestamp"     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Evidence Package (JSONB)
    device_snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
    recent_alerts       JSONB NOT NULL DEFAULT '[]'::jsonb,
    recent_logs         JSONB NOT NULL DEFAULT '[]'::jsonb,
    recording_status    TEXT NOT NULL DEFAULT '',
    downtime_history    JSONB NOT NULL DEFAULT '[]'::jsonb,
    sla_data            JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Media & Notes
    photos              TEXT[] NOT NULL DEFAULT '{}',
    notes               TEXT NOT NULL DEFAULT '',

    -- Status & Timestamps
    status              TEXT NOT NULL DEFAULT 'draft'
                        CHECK (status IN ('draft', 'finalized', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE incident_reports IS
    'KF-15.2.4: Black Box Incident Reports — пакет доказательств при инциденте. '
    'Соответствует IEC 62443 SR 7.1, ISO 27001 A.12.4, СТБ 34.101.27 п. 6.4';

COMMENT ON COLUMN incident_reports.triggered_by IS
    'Триггер создания: alarm — критическая тревога, manual — ручной вызов, '
    'sla_breach — нарушение SLA, downtime — неожиданный даунтайм';

COMMENT ON COLUMN incident_reports.device_snapshot IS
    'Снимок состояния устройства на момент инцидента (JSONB)';

COMMENT ON COLUMN incident_reports.recent_alerts IS
    'Последние N тревог, предшествовавших инциденту (JSONB array)';

COMMENT ON COLUMN incident_reports.recent_logs IS
    'Последние N логов устройства (JSONB array)';

COMMENT ON COLUMN incident_reports.downtime_history IS
    'История простоев устройства (JSONB array)';

COMMENT ON COLUMN incident_reports.sla_data IS
    'SLA compliance на момент инцидента (JSONB)';

COMMENT ON COLUMN incident_reports.photos IS
    'URIs прикреплённых фото/скриншотов';

-- ═══════════════════════════════════════════════════════════════════
-- 2. Таблица incident_triggers — audit trail для триггеров (ISO 27001 A.12.4)
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS incident_triggers (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    report_id       TEXT NOT NULL REFERENCES incident_reports(id) ON DELETE CASCADE,
    triggered_by    TEXT NOT NULL CHECK (triggered_by IN ('alarm', 'manual', 'sla_breach', 'downtime')),
    trigger_ref     TEXT NOT NULL DEFAULT '',
    triggered_by_user TEXT NOT NULL DEFAULT 'system',
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    "timestamp"     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE incident_triggers IS
    'KF-15.2.4: Audit trail для триггеров создания Black Box отчётов. '
    'Соответствует ISO 27001 A.12.4 (Audit trail), СТБ 34.101.27 п. 6.2';

CREATE INDEX IF NOT EXISTS idx_incident_triggers_report
    ON incident_triggers(report_id);
CREATE INDEX IF NOT EXISTS idx_incident_triggers_ts
    ON incident_triggers("timestamp");

-- ═══════════════════════════════════════════════════════════════════
-- 3. Индексы для incident_reports
-- ═══════════════════════════════════════════════════════════════════

CREATE INDEX IF NOT EXISTS idx_incident_reports_device
    ON incident_reports(device_id);
CREATE INDEX IF NOT EXISTS idx_incident_reports_ts
    ON incident_reports("timestamp" DESC);
CREATE INDEX IF NOT EXISTS idx_incident_reports_status
    ON incident_reports(status);
CREATE INDEX IF NOT EXISTS idx_incident_reports_trigger
    ON incident_reports(triggered_by);
CREATE INDEX IF NOT EXISTS idx_incident_reports_device_ts
    ON incident_reports(device_id, "timestamp" DESC);

-- ═══════════════════════════════════════════════════════════════════
-- 4. Функция авто-обновления updated_at
-- ═══════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION update_incident_reports_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_incident_reports_updated_at ON incident_reports;
CREATE TRIGGER trg_incident_reports_updated_at
    BEFORE UPDATE ON incident_reports
    FOR EACH ROW
    EXECUTE FUNCTION update_incident_reports_updated_at();

COMMENT ON TRIGGER trg_incident_reports_updated_at ON incident_reports IS
    'KF-15.2.4: Авто-обновление updated_at при изменении отчёта';
