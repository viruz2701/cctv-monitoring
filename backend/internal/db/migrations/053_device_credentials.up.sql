-- +migrate Up
-- CRED-01: Device Credentials Storage
--
-- Безопасное хранение credentials устройств видеонаблюдения.
-- Username/password шифруются на уровне приложения (AES-256-GCM / belt-gcm)
-- перед записью в БД. В БД хранятся только зашифрованные BYTEA.
--
-- Compliance:
--   - ISO 27001 A.9.2.1: User registration and de-registration
--   - ISO 27001 A.9.4.2: Secure log-on procedures
--   - ISO 27001 A.10.1.1: Cryptographic controls (encryption at rest)
--   - IEC 62443-3-3 SR 1.1: Authentication for human users
--   - IEC 62443-3-3 SR 1.5: Authenticator management
--   - OWASP ASVS V2.1: Verify credentials are stored using approved cryptographic functions
--   - OWASP ASVS V2.5: Verify credentials are hashed/encrypted at rest
--   - СТБ 34.101.27 п. 7.18: Защита аутентификационных данных
--   - Приказ ОАЦ №66 п. 7.18.3: Криптографическая защита

-- ═══════════════════════════════════════════════════════════════════════
-- 1. Device Credentials Table
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS device_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL,
    username_enc    BYTEA NOT NULL,       -- зашифрованный username (AES-256-GCM / belt-gcm)
    password_enc    BYTEA NOT NULL,       -- зашифрованный password (AES-256-GCM / belt-gcm)
    algorithm       VARCHAR(50) NOT NULL DEFAULT 'aes-256-gcm',  -- алгоритм шифрования
    key_ref         VARCHAR(100) NOT NULL DEFAULT 'primary',       -- ссылка на ключ (для ротации)
    expires_at      TIMESTAMPTZ,          -- опциональный срок действия credentials
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID,                 -- кто создал credentials
    updated_by      UUID,                 -- кто последним обновил

    -- Foreign key с каскадным удалением
    CONSTRAINT fk_device_credentials_device
        FOREIGN KEY (device_id)
        REFERENCES devices(device_id)
        ON DELETE CASCADE,

    -- Одно устройство = одна запись credentials
    CONSTRAINT uq_device_credentials_device UNIQUE (device_id)
);

-- ═══════════════════════════════════════════════════════════════════════
-- 2. Индексы
-- ═══════════════════════════════════════════════════════════════════════

-- Индекс для быстрого поиска по device_id
CREATE UNIQUE INDEX idx_device_credentials_device_id
    ON device_credentials(device_id);

-- Индекс для поиска по сроку действия (истекшие)
CREATE INDEX idx_device_credentials_expires_at
    ON device_credentials(expires_at)
    WHERE expires_at IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════
-- 3. Audit Trigger (ISO 27001 A.12.4.1, Приказ ОАЦ №66 п. 7.18.3)
-- ═══════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION trg_device_credentials_updated()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_device_credentials_updated
    BEFORE UPDATE ON device_credentials
    FOR EACH ROW
    EXECUTE FUNCTION trg_device_credentials_updated();

-- ═══════════════════════════════════════════════════════════════════════
-- 4. RLS (Row-Level Security) для multi-tenant
-- ═══════════════════════════════════════════════════════════════════════

ALTER TABLE device_credentials ENABLE ROW LEVEL SECURITY;

-- Только администраторы tenant могут видеть credentials своих устройств
CREATE POLICY device_credentials_tenant_isolation ON device_credentials
    FOR ALL
    USING (
        device_id IN (
            SELECT d.device_id FROM devices d
            WHERE d.site_id IN (
                SELECT s.site_id FROM sites s
                WHERE s.tenant_id = current_setting('app.tenant_id')::UUID
            )
        )
    );

-- ═══════════════════════════════════════════════════════════════════════
-- 5. Комментарии к колонкам (для документации БД)
-- ═══════════════════════════════════════════════════════════════════════

COMMENT ON TABLE device_credentials IS
    'Зашифрованные credentials устройств видеонаблюдения. '
    'Username/password хранятся в BYTEA, зашифрованные AES-256-GCM или belt-gcm. '
    'Соответствует: ISO 27001 A.9.4.2, IEC 62443 SR 1.5, OWASP ASVS V2.5, '
    'СТБ 34.101.27 п. 7.18, Приказ ОАЦ №66 п. 7.18.3';

COMMENT ON COLUMN device_credentials.username_enc IS
    'Зашифрованный username устройства (AES-256-GCM / belt-gcm)';
COMMENT ON COLUMN device_credentials.password_enc IS
    'Зашифрованный password устройства (AES-256-GCM / belt-gcm)';
COMMENT ON COLUMN device_credentials.algorithm IS
    'Алгоритм шифрования: aes-256-gcm (INTL) или belt-gcm (BY, СТБ 34.101.31)';
COMMENT ON COLUMN device_credentials.key_ref IS
    'Ссылка на ключ шифрования (для ротации ключей)';
COMMENT ON COLUMN device_credentials.expires_at IS
    'Опциональный срок действия credentials (для автоматической ротации)';
