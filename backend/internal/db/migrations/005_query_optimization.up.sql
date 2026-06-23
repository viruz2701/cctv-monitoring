-- Migration 005: Database query optimization
-- Composite indexes + connection pool tuning
-- Соответствует: ISO 27001 A.12.6.1 (Capacity management), IEC 62443 SR 7.1 (Resource availability)
--
-- ВАЖНО: golang-migrate запускает миграции в транзакции.
-- CREATE INDEX CONCURRENTLY запрещён внутри транзакции — используем обычный CREATE INDEX.
-- CREATE EXTENSION требует superuser — вынесено в bootstrap-скрипт.
-- +migrate Up

-- ============================================================
-- 1. Composite indexes для частых фильтров
-- ============================================================

-- Devices: фильтр по site + status (основной список устройств)
CREATE INDEX IF NOT EXISTS idx_devices_site_status
ON devices(site_id, status)
WHERE deleted_at IS NULL;

-- Devices: поиск по имени (ILIBC для поиска)
-- Требует расширения pg_trgm. Если расширение не установлено — индекс пропускается.
DO $body$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        CREATE INDEX IF NOT EXISTS idx_devices_name_trgm
        ON devices USING gin (name gin_trgm_ops)
        WHERE deleted_at IS NULL;
    END IF;
END;
$body$;

-- Devices: фильтр по vendor_type + status
CREATE INDEX IF NOT EXISTS idx_devices_vendor_status
ON devices(vendor_type, status)
WHERE deleted_at IS NULL;

-- Work Orders: статус + приоритет + дата создания (основной список)
CREATE INDEX IF NOT EXISTS idx_work_orders_status_priority
ON work_orders(status, priority, created_at DESC);

-- Work Orders: assigned_to + status (dashboard technician)
CREATE INDEX IF NOT EXISTS idx_work_orders_assigned_status
ON work_orders(assigned_to, status)
WHERE status IN ('open', 'in_progress');

-- Work Orders: SLA deadline (для мониторинга просрочек)
CREATE INDEX IF NOT EXISTS idx_work_orders_sla_active
ON work_orders(sla_deadline)
WHERE status IN ('open', 'in_progress');

-- Maintenance Schedules: next_due (календарь)
-- ВНИМАНИЕ: у таблицы maintenance_schedules нет колонки status,
-- поэтому используем простой индекс без WHERE.
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_next_due_active
ON maintenance_schedules(next_due)
WHERE next_due IS NOT NULL;

-- Alarms: time + device_id (TimescaleDB time-series)
CREATE INDEX IF NOT EXISTS idx_alarms_time_status
ON alarms(time DESC, status)
WHERE status = 'active';

-- Audit log: timestamp для поиска
CREATE INDEX IF NOT EXISTS idx_audit_log_timestamp_action
ON audit_log(timestamp DESC, action);

-- ============================================================
-- 2. View для топ-10 медленных запросов (только если pg_stat_statements доступен)
-- ============================================================
DO $body$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_stat_statements') THEN
        CREATE OR REPLACE VIEW slow_queries_top10 AS
        SELECT
            query,
            calls,
            ROUND(total_exec_time::numeric, 2) AS total_time_ms,
            ROUND(mean_exec_time::numeric, 2) AS avg_time_ms,
            ROUND(stddev_exec_time::numeric, 2) AS stddev_ms,
            rows,
            ROUND(shared_blks_hit::numeric / NULLIF(shared_blks_hit + shared_blks_read, 0) * 100, 2) AS cache_hit_ratio
        FROM pg_stat_statements
        WHERE query NOT LIKE '%pg_stat_statements%'
        ORDER BY mean_exec_time DESC
        LIMIT 10;
    END IF;
END;
$body$;

-- ============================================================
-- 3. Extension notes (устанавливаются отдельно superuser-ом)
-- ============================================================
-- Для работы slow_queries_top10 и полнотекстового поиска требуется:
--   CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
--   CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- Установите их от имени superuser:
--   psql -U postgres -d gb_telemetry -c "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;"
--   psql -U postgres -d gb_telemetry -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"
