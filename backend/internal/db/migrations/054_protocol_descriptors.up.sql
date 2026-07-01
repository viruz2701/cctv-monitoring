-- +migrate Up
-- PROTO-03: Protocol Descriptors Registry
--
-- Хранит JSON-дескрипторы протоколов для Edge-агентов.
-- Дескрипторы описывают, как взаимодействовать с устройствами
-- различных вендоров (Hikvision ISAPI, Dahua CGI, ONVIF SOAP и т.д.).
--
-- Compliance:
--   - ISO 27001 A.12.4.1: Event logging
--   - IEC 62443-3-3 SR 1.1: Unique identification
--   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)

CREATE TABLE protocol_descriptors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor      VARCHAR(100) NOT NULL,
    version     VARCHAR(50) NOT NULL,
    descriptor  JSONB NOT NULL,
    signature   VARCHAR(256),           -- HMAC-подпись (bash-256)
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Один вендор = одна версия дескриптора
    CONSTRAINT uq_protocol_descriptor_vendor UNIQUE (vendor)
);

-- Индекс для быстрого поиска по вендору
CREATE UNIQUE INDEX idx_protocol_descriptors_vendor
    ON protocol_descriptors(vendor);

-- Trigger для updated_at
CREATE OR REPLACE FUNCTION trg_protocol_descriptors_updated()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_protocol_descriptors_updated
    BEFORE UPDATE ON protocol_descriptors
    FOR EACH ROW
    EXECUTE FUNCTION trg_protocol_descriptors_updated();

COMMENT ON TABLE protocol_descriptors IS
    'JSON-дескрипторы протоколов для Edge-агентов. '
    'Позволяют динамически добавлять поддержку новых вендоров без перекомпиляции агента.';

COMMENT ON COLUMN protocol_descriptors.descriptor IS
    'JSON-дескриптор протокола (см. ProtocolDescriptor в internal/protocols/descriptor/schema.go)';
COMMENT ON COLUMN protocol_descriptors.signature IS
    'HMAC-подпись дескриптора (bash-256, СТБ 34.101.77) для проверки целостности агентом';
