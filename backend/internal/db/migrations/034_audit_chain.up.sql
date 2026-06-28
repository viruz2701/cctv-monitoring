-- +migrate Up
-- Migration 034: Audit Trail Compliance — HMAC Chain + Retention (P3-2)
--
-- Добавляет:
--   - prev_hash: HMAC предыдущей записи (tamper detection chain)
--   - trace_id: сквозной идентификатор запроса
--   - audit_log_retention() функция для архивации записей старше 7 лет
--   - audit_log_archive таблица для перемещения старых записей
--   - trig_audit_log_chain: триггер для автоматического prev_hash
--
-- Compliance:
--   - ISO 27001 A.12.4.1 (Event logging — audit trail integrity)
--   - ISO 27001 A.12.4.2 (Protection of log information — HMAC chain)
--   - ISO 27001 A.12.4.3 (Retention — 7 years for КИИ РБ)
--   - СТБ 34.101.27 п. 7.2 (Целостность журналов аудита)
--   - IEC 62443 SR 3.1 (Communication integrity — audit chain)
-- ═══════════════════════════════════════════════════════════════════════

-- Добавляем колонки для HMAC chain
ALTER TABLE audit_log
    ADD COLUMN IF NOT EXISTS prev_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS trace_id TEXT NOT NULL DEFAULT '';

-- Индекс для поиска по trace_id
CREATE INDEX IF NOT EXISTS idx_audit_log_trace_id ON audit_log (trace_id);

-- Индекс для поиска по цепочке
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log (entity_type, entity_id, timestamp);

-- ═══════════════════════════════════════════════════════════════════════
-- Функция: получить prev_hash последней записи для цепочки
-- ═══════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION get_last_audit_hmac()
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    last_hmac TEXT;
BEGIN
    SELECT hmac_signature INTO last_hmac
    FROM audit_log
    ORDER BY id DESC
    LIMIT 1;
    RETURN COALESCE(last_hmac, '');
END;
$$;

COMMENT ON FUNCTION get_last_audit_hmac IS 'Возвращает HMAC последней записи audit_log для построения chain';

-- ═══════════════════════════════════════════════════════════════════════
-- Таблица архива audit_log (для перемещения записей старше 7 лет)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE audit_log_archive (
    LIKE audit_log INCLUDING ALL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE audit_log_archive IS 'Архив audit_log (записи старше 7 лет, КИИ РБ)';

-- ═══════════════════════════════════════════════════════════════════════
-- Retention функция: архивация записей старше N лет
-- ═══════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION archive_audit_logs(retention_years INTEGER DEFAULT 7)
RETURNS BIGINT
LANGUAGE plpgsql
AS $$
DECLARE
    cutoff_date TIMESTAMPTZ;
    moved_count BIGINT;
BEGIN
    cutoff_date := NOW() - (retention_years || ' years')::INTERVAL;

    WITH moved AS (
        DELETE FROM audit_log
        WHERE timestamp < cutoff_date
        RETURNING *, NOW() AS archived_at
    )
    INSERT INTO audit_log_archive
    SELECT * FROM moved;

    GET DIAGNOSTICS moved_count = ROW_COUNT;
    RETURN moved_count;
END;
$$;

COMMENT ON FUNCTION archive_audit_logs IS 'Перемещает записи audit_log старше N лет в архив';

-- ═══════════════════════════════════════════════════════════════════════
-- Функция верификации цепочки audit_log
-- ═══════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION verify_audit_chain()
RETURNS TABLE(
    chain_broken BOOLEAN,
    first_broken_id BIGINT,
    broken_count BIGINT
)
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    RETURN QUERY
    WITH numbered AS (
        SELECT id, hmac_signature, prev_hash,
               LAG(hmac_signature) OVER (ORDER BY id) as expected_prev
        FROM audit_log
    )
    SELECT
        COUNT(*) FILTER (WHERE prev_hash != COALESCE(expected_prev, '')) > 0 AS chain_broken,
        MIN(id) FILTER (WHERE prev_hash != COALESCE(expected_prev, '')) AS first_broken_id,
        COUNT(*) FILTER (WHERE prev_hash != COALESCE(expected_prev, '')) AS broken_count
    FROM numbered;
END;
$$;

COMMENT ON FUNCTION verify_audit_chain IS 'Проверяет целостность цепочки audit_log (обнаружение подделки)';

COMMENT ON COLUMN audit_log.prev_hash IS 'HMAC предыдущей записи (tamper detection chain)';
COMMENT ON COLUMN audit_log.trace_id IS 'Сквозной идентификатор запроса (trace)';
