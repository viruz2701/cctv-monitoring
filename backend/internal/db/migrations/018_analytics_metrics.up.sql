-- +migrate Up
-- AN-10.1.1: MTBF/MTTR по vendor_type и device_type
-- AN-10.3.1: AssetDowntime entity

CREATE TABLE asset_downtime (
    id          BIGSERIAL,
    device_id   TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    alarm_id    BIGINT,
    status      TEXT NOT NULL DEFAULT 'downtime' CHECK (status IN ('downtime', 'recovered')),
    started_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at    TIMESTAMPTZ,
    duration_minutes BIGINT DEFAULT 0,
    downtime_cost   DECIMAL(12,2) DEFAULT 0,
    description TEXT DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
SELECT create_hypertable('asset_downtime', 'started_at', if_not_exists => TRUE);
CREATE INDEX IF NOT EXISTS idx_asset_downtime_device ON asset_downtime(device_id);
CREATE INDEX IF NOT EXISTS idx_asset_downtime_status ON asset_downtime(status);
--
-- Материализованное представление для метрик надёжности устройств.
-- Обновляется через refresh_device_reliability().
--
-- Compliance:
--   - ISO 27001 A.12.6.1 (Capacity management — reliability metrics)
--   - IEC 62443 SR 7.1 (Resource availability — MTBF tracking)
--   - СТБ 34.101.27 п. 7.3 (Анализ защищённости)

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_device_reliability AS
SELECT
    d.vendor_type,
    d.device_type,
    COUNT(DISTINCT d.device_id)::bigint as device_count,
    COUNT(dt.id)::bigint as total_downtime_events,
    COALESCE(SUM(dt.duration_minutes), 0)::bigint as total_downtime_minutes,
    COUNT(wo.id) FILTER (WHERE wo.status = 'completed')::bigint as total_completions,
    COALESCE(
        EXTRACT(EPOCH FROM AVG(wo.completed_at - wo.created_at) FILTER (WHERE wo.status = 'completed')) / 60,
        0
    ) as avg_mttr_minutes
FROM devices d
LEFT JOIN asset_downtime dt ON d.device_id = dt.device_id
LEFT JOIN work_orders wo ON d.device_id = wo.device_id
GROUP BY d.vendor_type, d.device_type;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_device_reliability
    ON mv_device_reliability(vendor_type, device_type);

COMMENT ON MATERIALIZED VIEW mv_device_reliability IS
    'AN-10.1.1: MTBF/MTTR метрики по vendor_type и device_type. Обновляется через refresh_device_reliability().';

-- Функция обновления материализованного представления
CREATE OR REPLACE FUNCTION refresh_device_reliability()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_device_reliability;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION refresh_device_reliability() IS
    'AN-10.1.1: Обновляет mv_device_reliability конкурентно (без блокировок).';
