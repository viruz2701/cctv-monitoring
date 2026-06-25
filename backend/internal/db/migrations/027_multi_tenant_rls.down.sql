-- +migrate Down
-- Migration 027: Multi-tenancy RLS (Rollback)

-- 1. Удаляем RLS-политики
DROP POLICY IF EXISTS tenant_isolation_users ON users;
DROP POLICY IF EXISTS tenant_isolation_onvif_devices ON onvif_devices;
DROP POLICY IF EXISTS tenant_isolation_maintenance_schedules ON maintenance_schedules;
DROP POLICY IF EXISTS tenant_isolation_asset_downtime ON asset_downtime;
DROP POLICY IF EXISTS tenant_isolation_spare_parts ON spare_parts;
DROP POLICY IF EXISTS tenant_isolation_work_orders ON work_orders;
DROP POLICY IF EXISTS tenant_isolation_tickets ON tickets;
DROP POLICY IF EXISTS tenant_isolation_sites ON sites;
DROP POLICY IF EXISTS tenant_isolation_devices ON devices;

-- 2. Отключаем RLS
ALTER TABLE devices DISABLE ROW LEVEL SECURITY;
ALTER TABLE sites DISABLE ROW LEVEL SECURITY;
ALTER TABLE tickets DISABLE ROW LEVEL SECURITY;
ALTER TABLE work_orders DISABLE ROW LEVEL SECURITY;
ALTER TABLE spare_parts DISABLE ROW LEVEL SECURITY;
ALTER TABLE asset_downtime DISABLE ROW LEVEL SECURITY;
ALTER TABLE maintenance_schedules DISABLE ROW LEVEL SECURITY;
ALTER TABLE onvif_devices DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;

-- 3. Удаляем вспомогательную функцию
DROP FUNCTION IF EXISTS rls_tenant_check(TEXT);

-- 4. Удаляем индексы
DROP INDEX IF EXISTS idx_devices_tenant;
DROP INDEX IF EXISTS idx_sites_tenant;
DROP INDEX IF EXISTS idx_tickets_tenant;
DROP INDEX IF EXISTS idx_work_orders_tenant;
DROP INDEX IF EXISTS idx_spare_parts_tenant;
DROP INDEX IF EXISTS idx_asset_downtime_tenant;
DROP INDEX IF EXISTS idx_maintenance_schedules_tenant;
DROP INDEX IF EXISTS idx_onvif_devices_tenant;
DROP INDEX IF EXISTS idx_users_tenant;

-- 5. Удаляем колонки tenant_id
ALTER TABLE devices DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE sites DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE tickets DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE work_orders DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE spare_parts DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE asset_downtime DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE maintenance_schedules DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE onvif_devices DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;
