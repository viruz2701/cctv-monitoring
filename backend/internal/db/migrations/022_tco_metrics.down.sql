-- +migrate Down
-- AN-10.1.3: Откат — удаление материализованного представления mv_tco_per_device

DROP INDEX IF EXISTS idx_mv_tco_device;

DROP MATERIALIZED VIEW IF EXISTS mv_tco_per_device;
