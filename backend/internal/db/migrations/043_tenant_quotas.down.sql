-- +migrate Down
-- Migration 043: Tenant Quota Management (P1-QUOTA) — rollback

DROP TRIGGER IF EXISTS trg_tenant_quotas_updated_at ON tenant_quotas;
DROP FUNCTION IF EXISTS update_tenant_quotas_updated_at();

DROP TABLE IF EXISTS tenant_quota_history CASCADE;
DROP TABLE IF EXISTS tenant_quotas CASCADE;
