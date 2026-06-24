-- +migrate Up
-- Migration 023: Feature Flags (F-0.2.4)
--
-- Таблица для хранения feature flags с поддержкой multi-tenancy.
--
-- Compliance:
--   - IEC 62443 SR 5.1 (Network segmentation — feature gating)
--   - ISO 27001 A.12.6.1 (Capacity management — gradual rollout)
--   - OWASP ASVS V1.6 (Controlled change management)

CREATE TABLE feature_flags (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    key           TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL DEFAULT '',
    description   TEXT DEFAULT '',
    enabled       BOOLEAN NOT NULL DEFAULT false,
    tenant_id     TEXT DEFAULT '',
    is_global     BOOLEAN NOT NULL DEFAULT true,
    metadata      JSONB DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feature_flags_key ON feature_flags(key);
CREATE INDEX IF NOT EXISTS idx_feature_flags_tenant ON feature_flags(tenant_id);
CREATE INDEX IF NOT EXISTS idx_feature_flags_enabled ON feature_flags(enabled);
