-- +migrate Up
-- Migration 032: OAuth2 Tokens for External Adapters (P2-3.2)
--
-- Хранение зашифрованных OAuth2 токенов для внешних CMMS
-- (ServiceNow, Jira, 1С:ТОИР) с поддержкой multi-tenant RLS.
--
-- Compliance:
--   - IEC 62443 SL-3 (Zone 3 — Application data integrity)
--   - ISO 27001 A.10.1 (Cryptographic controls — encrypted at rest)
--   - СТБ 34.101.30 (Encryption via crypto.Encrypt — AES-256-GCM / belt-gcm)
--   - OWASP ASVS V6.2 (Stored credentials — encrypted)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE oauth2_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Провайдер: 'servicenow', 'jira', 'toir'
    provider    TEXT NOT NULL CHECK (provider IN ('servicenow', 'jira', 'toir')),

    -- Уникальный ключ в рамках провайдера (например, инстанс URL или tenant_id)
    provider_key TEXT NOT NULL,

    -- Зашифрованный токен (AES-256-GCM hex)
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL DEFAULT '',
    token_type    TEXT NOT NULL DEFAULT 'Bearer',
    expiry        TIMESTAMPTZ,

    -- Метаданные
    scopes      TEXT[] DEFAULT '{}',
    tenant_id   TEXT NOT NULL DEFAULT '*',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Уникальность: один токен на провайдер + ключ
    UNIQUE (provider, provider_key)
);

-- Индекс для быстрого поиска по провайдеру
CREATE INDEX idx_oauth2_tokens_provider
    ON oauth2_tokens (provider);

-- Индекс для поиска по tenant
CREATE INDEX idx_oauth2_tokens_tenant
    ON oauth2_tokens (tenant_id);

-- Trigger: auto-update updated_at
CREATE OR REPLACE FUNCTION update_oauth2_tokens_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_oauth2_tokens_updated_at ON oauth2_tokens;
CREATE TRIGGER trg_oauth2_tokens_updated_at
    BEFORE UPDATE ON oauth2_tokens
    FOR EACH ROW
    EXECUTE FUNCTION update_oauth2_tokens_updated_at();

-- RLS (Row-Level Security) для multi-tenant
ALTER TABLE oauth2_tokens ENABLE ROW LEVEL SECURITY;

-- Политика для tenant: видят только свои токены
DROP POLICY IF EXISTS oauth2_tokens_tenant_policy ON oauth2_tokens;
CREATE POLICY oauth2_tokens_tenant_policy ON oauth2_tokens
    USING (tenant_id = current_setting('app.tenant_id', TRUE) OR tenant_id = '*');

COMMENT ON TABLE oauth2_tokens IS 'Зашифрованные OAuth2 токены для внешних CMMS адаптеров';
COMMENT ON COLUMN oauth2_tokens.provider IS 'Провайдер: servicenow, jira, toir';
COMMENT ON COLUMN oauth2_tokens.provider_key IS 'Уникальный ключ (instance URL / tenant)';
COMMENT ON COLUMN oauth2_tokens.access_token IS 'Зашифрованный access_token (AES-256-GCM)';
COMMENT ON COLUMN oauth2_tokens.refresh_token IS 'Зашифрованный refresh_token (AES-256-GCM)';
COMMENT ON COLUMN oauth2_tokens.expiry IS 'Время истечения токена';
