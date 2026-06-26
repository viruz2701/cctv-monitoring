-- +migrate Up
-- Migration 036: Tenant Compliance Profile (P0-CE.5)
--
-- Добавляет compliance_region к tenant'ам для per-tenant compliance.
--
-- Compliance:
--   - GDPR Art. 44-49 (Data transfer — region pinning)
--   - ISO 27001 A.8.1 (Asset management — tenant classification)
--   - IEC 62443 SR 2.1 (Account management — tenant isolation)
--   - СТБ 34.101.27 п. 6.2 (Разграничение доступа по tenant'ам)
-- ═══════════════════════════════════════════════════════════════════════

-- Добавляем compliance_region в tenant_regions
ALTER TABLE tenant_regions
    ADD COLUMN IF NOT EXISTS compliance_region VARCHAR(10) NOT NULL DEFAULT 'INTL'
    CONSTRAINT valid_compliance_region CHECK (
        compliance_region IN ('BY', 'EU', 'INTL', 'RU', 'CN', 'US')
    );

-- Поле блокировки региона (immutable после first data creation)
ALTER TABLE tenant_regions
    ADD COLUMN IF NOT EXISTS compliance_locked BOOLEAN NOT NULL DEFAULT false;

-- Индекс для быстрого поиска по compliance_region
CREATE INDEX IF NOT EXISTS idx_tenant_regions_compliance
    ON tenant_regions (compliance_region);

-- Функция блокировки compliance региона при первом создании данных
CREATE OR REPLACE FUNCTION lock_compliance_region()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE tenant_regions
    SET compliance_locked = true
    WHERE tenant_id = NEW.tenant_id
      AND compliance_locked = false;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер на devices — блокировка региона при первом добавлении устройства
DROP TRIGGER IF EXISTS trg_lock_compliance_on_device ON devices;
CREATE TRIGGER trg_lock_compliance_on_device
    AFTER INSERT ON devices
    FOR EACH ROW
    EXECUTE FUNCTION lock_compliance_region();

-- Триггер на work_orders — блокировка региона при первом создании WO
DROP TRIGGER IF EXISTS trg_lock_compliance_on_wo ON work_orders;
CREATE TRIGGER trg_lock_compliance_on_wo
    AFTER INSERT ON work_orders
    FOR EACH ROW
    EXECUTE FUNCTION lock_compliance_region();

-- RLS: добавляем compliance_region в политики
-- Тенант видит данные только своего compliance региона
CREATE OR REPLACE FUNCTION get_tenant_compliance_region()
RETURNS TEXT AS $$
    SELECT current_setting('app.compliance_region', true);
$$ LANGUAGE SQL STABLE;

-- RLS политика для tenant_regions по compliance_region
ALTER TABLE tenant_regions ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_compliance_isolation ON tenant_regions;
CREATE POLICY tenant_compliance_isolation ON tenant_regions
    USING (
        compliance_region = get_tenant_compliance_region()
        OR current_setting('app.role', true) = 'admin'
    );

COMMENT ON COLUMN tenant_regions.compliance_region IS 'Compliance регион тенанта (BY, EU, INTL, RU, CN, US)';
COMMENT ON COLUMN tenant_regions.compliance_locked IS 'Флаг блокировки региона (immutable после first data)';
COMMENT ON FUNCTION lock_compliance_region() IS 'Блокирует compliance_region при первом создании данных тенанта';
