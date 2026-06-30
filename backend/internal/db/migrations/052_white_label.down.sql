-- +migrate Down

DROP TABLE IF EXISTS tenant_branding_audit CASCADE;
DROP TABLE IF EXISTS tenant_domain_verifications CASCADE;
DROP TABLE IF EXISTS tenant_branding CASCADE;
