-- +migrate Up
-- Migration 058: Calendar Sync Idempotency (P1-HI-09)
--
-- Добавляет idempotency_key для предотвращения дублирования sync операций.
--
-- Проблема: При повторных push операциях (например, после таймаута) 
-- создаются duplicate записи в calendar_sync_log и потенциально duplicate
-- события в календарях провайдеров.
--
-- Решение: idempotency_key UUID + UNIQUE constraint → ON CONFLICT DO NOTHING
--
-- Compliance:
--   - ISO 27001 A.12.4.1 (Event logging — deduplication)
--   - IEC 62443 SR 3.1 (Data integrity)
-- ═══════════════════════════════════════════════════════════════════════

-- Добавляем idempotency_key в calendar_sync_log
ALTER TABLE calendar_sync_log
    ADD COLUMN IF NOT EXISTS idempotency_key UUID;

-- Уникальный индекс для idempotency (ON CONFLICT DO NOTHING)
CREATE UNIQUE INDEX IF NOT EXISTS idx_calendar_sync_log_idempotency
    ON calendar_sync_log (idempotency_key)
    WHERE idempotency_key IS NOT NULL;

COMMENT ON COLUMN calendar_sync_log.idempotency_key IS 'UUID для идемпотентности операций синхронизации (P1-HI-09)';
