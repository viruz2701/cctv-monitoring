-- +migrate Up
-- P3-WL: White-Label Theming
--
-- Таблица tenant_branding для per-tenant кастомизации бренда:
--   - Логотип, фавиконка
--   - Цветовая схема (primary, secondary, accent)
--   - Кастомный домен (CNAME)
--   - Email/PDF шаблоны
--
-- Compliance:
--   - IEC 62443-3-3 SL-2 (Zone 2 — DMZ): Управление конфигурацией
--   - ISO 27001 A.8.1: Asset management — tenant assets
--   - ISO 27001 A.12.4.1: Event logging — audit trail
--   - OWASP ASVS V2.1.1: Input validation
--   - Приказ ОАЦ №66 п. 7.18.3: Аудит операций

-- +migrate Up

-- ═══════════════════════════════════════════════════════════════════════
-- 1. Tenant Branding Configuration
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE tenant_branding (
    id              VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id       VARCHAR(64) NOT NULL UNIQUE,
    
    -- Company info
    company_name    TEXT NOT NULL DEFAULT '',
    logo_url        TEXT NOT NULL DEFAULT '',
    favicon_url     TEXT NOT NULL DEFAULT '',
    
    -- Color scheme
    primary_color   TEXT NOT NULL DEFAULT '#2563eb',
    secondary_color TEXT NOT NULL DEFAULT '#6366f1',
    accent_color    TEXT NOT NULL DEFAULT '#06b6d4',
    
    -- Font & CSS customization
    font_family     TEXT NOT NULL DEFAULT 'Inter, system-ui, sans-serif',
    custom_css      TEXT NOT NULL DEFAULT '',
    
    -- Custom domain (CNAME)
    custom_domain   TEXT NOT NULL DEFAULT '',
    cname_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    cname_verified_at TIMESTAMPTZ,
    cname_verification_token TEXT NOT NULL DEFAULT '',
    
    -- Email branding
    email_header_logo_url TEXT NOT NULL DEFAULT '',
    email_footer_text     TEXT NOT NULL DEFAULT '',
    email_primary_color   TEXT NOT NULL DEFAULT '#2563eb',
    
    -- PDF branding
    pdf_logo_url          TEXT NOT NULL DEFAULT '',
    pdf_primary_color     TEXT NOT NULL DEFAULT '#2563eb',
    pdf_secondary_color   TEXT NOT NULL DEFAULT '#6366f1',
    pdf_footer_text       TEXT NOT NULL DEFAULT '',
    
    -- Active state
    is_active       BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Metadata
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    is_locked       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      TEXT NOT NULL DEFAULT '',

    -- Domain format validation
    CONSTRAINT chk_tenant_branding_domain CHECK (
        custom_domain = ''
        OR custom_domain ~ '^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$'
    ),
    CONSTRAINT chk_tenant_branding_colors CHECK (
        primary_color ~ '^#[0-9a-fA-F]{6}$'
        AND secondary_color ~ '^#[0-9a-fA-F]{6}$'
        AND accent_color ~ '^#[0-9a-fA-F]{6}$'
        AND email_primary_color ~ '^#[0-9a-fA-F]{6}$'
        AND pdf_primary_color ~ '^#[0-9a-fA-F]{6}$'
        AND pdf_secondary_color ~ '^#[0-9a-fA-F]{6}$'
    )
);

COMMENT ON TABLE tenant_branding IS
    'P3-WL: Per-tenant branding configuration. Содержит настройки бренда: '
    'логотип, цвета, кастомный домен, email/PDF шаблоны. '
    'Соответствует IEC 62443 SL-2, OWASP ASVS V2.1.1';

-- Индекс для поиска по tenant_id (уникальный, уже в UNIQUE)
COMMENT ON COLUMN tenant_branding.tenant_id IS 'Уникальный идентификатор tenant''а';

-- Индекс для поиска по кастомному домену
CREATE INDEX idx_tenant_branding_custom_domain
    ON tenant_branding(custom_domain)
    WHERE custom_domain != '' AND cname_verified = TRUE;

-- ═══════════════════════════════════════════════════════════════════════
-- 2. Domain Verification Log
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE tenant_domain_verifications (
    id              VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id       VARCHAR(64) NOT NULL,
    domain          TEXT NOT NULL,
    verification_token TEXT NOT NULL,
    verified        BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at     TIMESTAMPTZ,
    error_message   TEXT NOT NULL DEFAULT '',
    attempt_count   INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_domain_verifications_tenant
    ON tenant_domain_verifications(tenant_id, domain);

COMMENT ON TABLE tenant_domain_verifications IS
    'P3-WL: История верификации кастомных доменов tenant''ов. '
    'Содержит попытки верификации CNAME записей.';

-- ═══════════════════════════════════════════════════════════════════════
-- 3. Tenant Branding Audit Log
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE tenant_branding_audit (
    id              VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    tenant_id       VARCHAR(64) NOT NULL,
    action          TEXT NOT NULL,          -- 'updated', 'logo_uploaded', 'domain_verified', 'reset'
    field_name      TEXT NOT NULL DEFAULT '', -- изменённое поле
    old_value       TEXT NOT NULL DEFAULT '',
    new_value       TEXT NOT NULL DEFAULT '',
    changed_by      TEXT NOT NULL,
    ip_address      TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenant_branding_audit_tenant
    ON tenant_branding_audit(tenant_id, created_at DESC);

COMMENT ON TABLE tenant_branding_audit IS
    'P3-WL: Audit trail изменений бренда. '
    'Соответствует ISO 27001 A.12.4 (Event logging).';
