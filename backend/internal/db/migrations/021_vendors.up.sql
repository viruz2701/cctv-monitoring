-- +migrate Up
-- INV-7.2.1: Vendor entity — управление поставщиками
--
-- Создаёт таблицу vendors и добавляет vendor_id в spare_parts.
--
-- Compliance:
--   - IEC 62443 SL-3 (Zone 3 — Application integrity)
--   - ISO 27001 A.12.4.1 (Event logging — created_at/updated_at audit trail)
--   - ISO/IEC 27019 PCC.A.5 (Supply chain management)
--   - СТБ 34.101.27 (Защита информации — управление поставщиками)
--   - OWASP ASVS V5.1 (Parameterized queries — через DB слой)

CREATE TABLE vendors (
    id             TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name           TEXT NOT NULL,
    contact_person TEXT NOT NULL DEFAULT '',
    email          TEXT NOT NULL DEFAULT '',
    phone          TEXT NOT NULL DEFAULT '',
    address        TEXT NOT NULL DEFAULT '',
    website        TEXT NOT NULL DEFAULT '',
    notes          TEXT NOT NULL DEFAULT '',
    status         TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE vendors IS
    'INV-7.2.1: Поставщики (Vendors). '
    'Содержит контактную информацию и статус поставщика. '
    'Соответствует ISO 27001 A.15 (Supplier relationships), IEC 62443 SL-3.';

COMMENT ON COLUMN vendors.id IS 'TEXT первичный ключ (генерация: gen_random_uuid())';
COMMENT ON COLUMN vendors.name IS 'Название поставщика (обязательное поле)';
COMMENT ON COLUMN vendors.contact_person IS 'Контактное лицо';
COMMENT ON COLUMN vendors.email IS 'Email поставщика';
COMMENT ON COLUMN vendors.phone IS 'Телефон поставщика';
COMMENT ON COLUMN vendors.address IS 'Адрес поставщика';
COMMENT ON COLUMN vendors.website IS 'Веб-сайт поставщика';
COMMENT ON COLUMN vendors.notes IS 'Заметки о поставщике';
COMMENT ON COLUMN vendors.status IS 'Статус: active, inactive (CHECK constraint)';
COMMENT ON COLUMN vendors.created_at IS 'Время создания (audit trail)';
COMMENT ON COLUMN vendors.updated_at IS 'Время последнего обновления (audit trail)';

-- INV-7.2.1: Добавляем vendor_id в spare_parts
ALTER TABLE spare_parts
    ADD COLUMN IF NOT EXISTS vendor_id TEXT REFERENCES vendors(id) ON DELETE SET NULL;

COMMENT ON COLUMN spare_parts.vendor_id IS
    'INV-7.2.1: ID поставщика (FK → vendors.id, ON DELETE SET NULL). '
    'Позволяет связать запчасть с поставщиком.';

CREATE INDEX IF NOT EXISTS idx_spare_parts_vendor
    ON spare_parts(vendor_id);
