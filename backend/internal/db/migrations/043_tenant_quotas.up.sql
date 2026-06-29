-- +migrate Up
-- Migration 043: Tenant Quota Management (P1-QUOTA)
--
-- Добавляет таблицу tenant_quotas для per-tenant лимитов ресурсов.
-- Real-time counters хранятся в Redis, а конфигурация лимитов — в PostgreSQL.
--
-- ⚠ Внимание: используется CREATE TABLE без IF NOT EXISTS.
-- Миграции применяются через golang-migrate с гарантией идемпотентности.
--
-- Compliance:
--   - ISO 27001 A.12.1.2 (Capacity management)
--   - IEC 62443-3-3 SR 3.1 (Resource management)
--   - IEC 62443-3-3 SR 7.1 (Audit trail — quota changes)
--   - OWASP ASVS V2.2.1 (Rate limiting)
--   - СТБ 34.101.27 п. 6.1 (Защита от DoS)
-- ═══════════════════════════════════════════════════════════════════════

-- Таблица конфигурации квот tenant'ов
CREATE TABLE tenant_quotas (
    tenant_id   TEXT PRIMARY KEY REFERENCES tenant_regions(tenant_id) ON DELETE CASCADE,
    devices     INT NOT NULL DEFAULT 100,
    users       INT NOT NULL DEFAULT 10,
    storage_gb  INT NOT NULL DEFAULT 1000,
    api_calls   INT NOT NULL DEFAULT 10000,
    work_orders INT NOT NULL DEFAULT 500,
    grace_days  INT NOT NULL DEFAULT 7,
    grace_until TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Функция автообновления updated_at
CREATE OR REPLACE FUNCTION update_tenant_quotas_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер на обновление updated_at
CREATE TRIGGER trg_tenant_quotas_updated_at
    BEFORE UPDATE ON tenant_quotas
    FOR EACH ROW
    EXECUTE FUNCTION update_tenant_quotas_updated_at();

-- Таблица истории изменений квот (аудит)
CREATE TABLE tenant_quota_history (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   TEXT NOT NULL REFERENCES tenant_regions(tenant_id) ON DELETE CASCADE,
    quota_type  TEXT NOT NULL,
    old_limit   INT NOT NULL,
    new_limit   INT NOT NULL,
    reason      TEXT NOT NULL DEFAULT '',
    changed_by  TEXT NOT NULL DEFAULT 'system',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для быстрого поиска по tenant_id и quota_type
CREATE INDEX IF NOT EXISTS idx_tenant_quota_history_tenant
    ON tenant_quota_history (tenant_id, quota_type);

-- Индекс для сортировки по времени
CREATE INDEX IF NOT EXISTS idx_tenant_quota_history_created
    ON tenant_quota_history (created_at DESC);

-- RLS для tenant_quotas
ALTER TABLE tenant_quotas ENABLE ROW LEVEL SECURITY;

-- Политика: tenant видит только свои квоты
CREATE POLICY tenant_quotas_tenant_policy ON tenant_quotas
    FOR ALL
    USING (tenant_id = current_setting('app.tenant_id')::TEXT);

-- Политика: admin видит все квоты
CREATE POLICY tenant_quotas_admin_policy ON tenant_quotas
    FOR ALL
    USING (current_setting('app.user_role')::TEXT = 'admin');

-- RLS для tenant_quota_history
ALTER TABLE tenant_quota_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_quota_history_tenant_policy ON tenant_quota_history
    FOR SELECT
    USING (tenant_id = current_setting('app.tenant_id')::TEXT);

CREATE POLICY tenant_quota_history_admin_policy ON tenant_quota_history
    FOR ALL
    USING (current_setting('app.user_role')::TEXT = 'admin');

-- Seed: добавляем квоты для существующих tenant'ов
INSERT INTO tenant_quotas (tenant_id)
SELECT tenant_id FROM tenant_regions
WHERE NOT EXISTS (SELECT 1 FROM tenant_quotas WHERE tenant_quotas.tenant_id = tenant_regions.tenant_id);
