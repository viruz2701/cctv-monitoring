-- +migrate Up
-- Migration 033: Webhook Delivery Logs (P2-3.3)
--
-- Хранит:
--   - webhook_endpoints: настройки исходящих вебхуков
--   - webhook_delivery_logs: история доставки с retry
--
-- Compliance:
--   - OWASP ASVS V7.1 (Error handling — delivery failure tracking)
--   - IEC 62443 SR 7.1 (Resource availability — delivery monitoring)
--   - ISO 27001 A.12.4.1 (Event logging — delivery audit trail)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE webhook_endpoints (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    url             TEXT NOT NULL CHECK (url LIKE 'https://%'),
    secret          TEXT NOT NULL DEFAULT '',
    events          TEXT[] NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT true,
    retry_count     INTEGER NOT NULL DEFAULT 3,
    timeout_seconds INTEGER NOT NULL DEFAULT 10,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE webhook_delivery_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id      UUID NOT NULL REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type      TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('pending', 'success', 'failed', 'cancelled')),
    request_url     TEXT NOT NULL,
    request_body    TEXT NOT NULL DEFAULT '',
    response_status INTEGER NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    retry_attempt   INTEGER NOT NULL DEFAULT 0,
    max_retries     INTEGER NOT NULL DEFAULT 3,
    error_message   TEXT NOT NULL DEFAULT '',
    next_retry_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_delivery_logs_webhook_id
    ON webhook_delivery_logs (webhook_id, created_at DESC);

CREATE INDEX idx_webhook_delivery_logs_pending
    ON webhook_delivery_logs (next_retry_at)
    WHERE status = 'pending' AND next_retry_at IS NOT NULL;

CREATE OR REPLACE FUNCTION update_webhook_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_webhook_endpoints_updated_at
    BEFORE UPDATE ON webhook_endpoints
    FOR EACH ROW EXECUTE FUNCTION update_webhook_updated_at();

CREATE TRIGGER trg_webhook_delivery_logs_updated_at
    BEFORE UPDATE ON webhook_delivery_logs
    FOR EACH ROW EXECUTE FUNCTION update_webhook_updated_at();

COMMENT ON TABLE webhook_endpoints IS 'Настройки исходящих вебхуков для внешних систем';
COMMENT ON TABLE webhook_delivery_logs IS 'Логи доставки вебхуков с поддержкой retry';
COMMENT ON COLUMN webhook_delivery_logs.next_retry_at IS 'Время следующей попытки retry (exponential backoff)';
COMMENT ON COLUMN webhook_delivery_logs.max_retries IS 'Максимальное количество retry (настраивается в webhook_endpoints.retry_count)';
