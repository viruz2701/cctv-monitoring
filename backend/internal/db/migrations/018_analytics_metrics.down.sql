-- +migrate Down
-- AN-10.1.1: Откат метрик надёжности устройств

DROP INDEX IF EXISTS idx_mv_device_reliability;
DROP FUNCTION IF EXISTS refresh_device_reliability();
DROP MATERIALIZED VIEW IF EXISTS mv_device_reliability;
