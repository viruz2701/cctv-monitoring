-- +migrate Down
-- Откат P0-CE.5: Tenant Compliance Profile (SaaS)
--
-- Удаляет compliance_region, compliance_locked, триггеры и RLS политики.

-- Удаляем RLS политику
DROP POLICY IF EXISTS tenant_compliance_isolation ON tenant_regions;

-- Удаляем функцию compliance_region
DROP FUNCTION IF EXISTS get_tenant_compliance_region();

-- Удаляем триггеры блокировки
DROP TRIGGER IF EXISTS trg_lock_compliance_on_device ON devices;
DROP TRIGGER IF EXISTS trg_lock_compliance_on_wo ON work_orders;

-- Удаляем функцию блокировки
DROP FUNCTION IF EXISTS lock_compliance_region();

-- Удаляем индекс
DROP INDEX IF EXISTS idx_tenant_regions_compliance;

-- Удаляем колонки (CASCADE снимает constraint)
ALTER TABLE tenant_regions
    DROP COLUMN IF EXISTS compliance_locked;

ALTER TABLE tenant_regions
    DROP COLUMN IF EXISTS compliance_region;
