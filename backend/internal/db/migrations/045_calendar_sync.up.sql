-- ═══════════════════════════════════════════════════════════════════════
-- P1-CALENDAR: External Calendar Sync (Google + Outlook)
--
-- Таблицы для OAuth2-подключения календарей и маппинга Work Order ↔
-- внешних событий.
--
-- Compliance:
--   - ISO 27001 A.10.1 (Encrypted tokens at rest)
--   - ISO 27001 A.12.4 (Audit trail — created_at, updated_at)
--   - IEC 62443-3-3 SL-3 (Zone 3 — Application data integrity)
--   - OWASP ASVS V6.2 (Stored credentials — encrypted)
-- ═══════════════════════════════════════════════════════════════════════

-- +migrate Up

-- ── Calendar Connections (OAuth2) ─────────────────────────────────────
CREATE TABLE calendar_connections (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         TEXT NOT NULL,
    provider        TEXT NOT NULL CHECK (provider IN ('google', 'outlook')),
    access_token    TEXT NOT NULL,                    -- encrypted at rest
    refresh_token   TEXT NOT NULL DEFAULT '',         -- encrypted at rest
    token_expiry    TIMESTAMPTZ,
    calendar_id     TEXT,                              -- external calendar ID
    calendar_name   TEXT DEFAULT '',                   -- human-readable name
    enabled         BOOLEAN NOT NULL DEFAULT true,
    tenant_id       TEXT NOT NULL DEFAULT '*',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, provider)
);

CREATE INDEX idx_calendar_connections_user ON calendar_connections(user_id);
CREATE INDEX idx_calendar_connections_provider ON calendar_connections(provider);
CREATE INDEX idx_calendar_connections_tenant ON calendar_connections(tenant_id);

-- Trigger: auto-update updated_at
CREATE TRIGGER trg_calendar_connections_updated_at
    BEFORE UPDATE ON calendar_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- RLS for multi-tenant isolation
ALTER TABLE calendar_connections ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS calendar_connections_tenant_policy ON calendar_connections;
CREATE POLICY calendar_connections_tenant_policy ON calendar_connections
    USING (tenant_id = current_setting('app.tenant_id', TRUE) OR tenant_id = '*');

COMMENT ON TABLE calendar_connections IS 'OAuth2-подключения к Google Calendar и Outlook';
COMMENT ON COLUMN calendar_connections.access_token IS 'Зашифрованный access_token (AES-256-GCM / belt-gcm)';
COMMENT ON COLUMN calendar_connections.refresh_token IS 'Зашифрованный refresh_token';
COMMENT ON COLUMN calendar_connections.calendar_id IS 'ID календаря у провайдера (primary)';

-- ── Calendar Events (WO ↔ External Event mapping) ─────────────────────
CREATE TABLE calendar_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wo_id           TEXT NOT NULL,
    provider        TEXT NOT NULL CHECK (provider IN ('google', 'outlook')),
    external_id     TEXT NOT NULL,                    -- ID события у провайдера
    event_url       TEXT DEFAULT '',                   -- ссылка на событие
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'updated', 'deleted')),
    last_synced     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (wo_id, provider)
);

CREATE INDEX idx_calendar_events_wo ON calendar_events(wo_id);
CREATE INDEX idx_calendar_events_external ON calendar_events(provider, external_id);
CREATE INDEX idx_calendar_events_synced ON calendar_events(last_synced);

COMMENT ON TABLE calendar_events IS 'Маппинг Work Order ↔ внешнее событие календаря';
COMMENT ON COLUMN calendar_events.external_id IS 'Идентификатор события у провайдера (Google event ID / Outlook event ID)';
COMMENT ON COLUMN calendar_events.event_url IS 'Ссылка на событие для открытия в UI провайдера';

-- ── Sync Log (audit trail for bi-directional sync) ────────────────────
CREATE TABLE calendar_sync_log (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wo_id           TEXT,
    provider        TEXT NOT NULL CHECK (provider IN ('google', 'outlook')),
    direction       TEXT NOT NULL CHECK (direction IN ('push', 'pull')),
    event_type      TEXT NOT NULL CHECK (event_type IN ('created', 'updated', 'deleted', 'skipped', 'conflict')),
    external_id     TEXT DEFAULT '',
    details         JSONB DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'success' CHECK (status IN ('success', 'error', 'conflict')),
    error_message   TEXT DEFAULT '',
    tenant_id       TEXT NOT NULL DEFAULT '*',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendar_sync_log_wo ON calendar_sync_log(wo_id);
CREATE INDEX idx_calendar_sync_log_provider ON calendar_sync_log(provider);
CREATE INDEX idx_calendar_sync_log_created ON calendar_sync_log(created_at DESC);
CREATE INDEX idx_calendar_sync_log_tenant ON calendar_sync_log(tenant_id);

-- Partition by day for performance (sync log is high-volume)
-- P2-MED-01: Явно задаём chunk_time_interval => INTERVAL '1 day'
-- для оптимального управления партициями и упрощения retention policy.
SELECT create_hypertable('calendar_sync_log', 'created_at',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE);

COMMENT ON TABLE calendar_sync_log IS 'Аудит синхронизации календарей (ISO 27001 A.12.4)';
COMMENT ON COLUMN calendar_sync_log.direction IS 'push — из WO в календарь, pull — из календаря в WO';
COMMENT ON COLUMN calendar_sync_log.event_type IS 'Тип операции синхронизации';
