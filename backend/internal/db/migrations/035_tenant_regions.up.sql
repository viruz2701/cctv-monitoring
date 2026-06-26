-- +migrate Up
-- Migration 035: Tenant Region Mapping for Multi-Region Geo-Redundancy (P3-1)
--
-- Хранит привязку тенантов к регионам для DR и data residency.
--
-- Compliance:
--   - GDPR Art. 44-49 (Data transfer — region pinning)
--   - 152-ФЗ ст. 18 (Data localization — CIS-East)
--   - PDPL (Saudi Arabia — MENA-Gulf)
--   - ISO 27001 A.17.1 (Business continuity — DR)
--   - IEC 62443 SR 7.1 (Resource availability — multi-region)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE tenant_regions (
    tenant_id       TEXT PRIMARY KEY,
    primary_region  TEXT NOT NULL,
    failover_region TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'failover', 'migrating')),
    failover_count  INTEGER NOT NULL DEFAULT 0,
    last_failover_at TIMESTAMPTZ,
    pinned_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_region CHECK (
        primary_region IN ('eu-central', 'cis-east', 'mena-gulf', 'sea-hub')
    )
);

CREATE INDEX idx_tenant_regions_region ON tenant_regions (primary_region);
CREATE INDEX idx_tenant_regions_status ON tenant_regions (status);

CREATE OR REPLACE FUNCTION update_tenant_region_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_tenant_regions_updated_at
    BEFORE UPDATE ON tenant_regions
    FOR EACH ROW EXECUTE FUNCTION update_tenant_region_updated_at();

-- Добавляем region в users для multi-region аутентификации
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT 'eu-central';

COMMENT ON TABLE tenant_regions IS 'Привязка тенантов к регионам (P3-1 Multi-Region)';
COMMENT ON COLUMN tenant_regions.primary_region IS 'Основной регион (eu-central, cis-east, mena-gulf, sea-hub)';
COMMENT ON COLUMN tenant_regions.failover_region IS 'Регион DR для failover';
COMMENT ON COLUMN tenant_regions.status IS 'Статус: active, failover, migrating';
COMMENT ON COLUMN users.region IS 'Регион пользователя для multi-region routing';
