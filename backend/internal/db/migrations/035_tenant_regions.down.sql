-- +migrate Down
-- Откат P3-1: Multi-Region Geo-Redundancy
ALTER TABLE users DROP COLUMN IF EXISTS region;
DROP TRIGGER IF EXISTS trg_tenant_regions_updated_at ON tenant_regions;
DROP FUNCTION IF EXISTS update_tenant_region_updated_at();
DROP TABLE IF EXISTS tenant_regions;
