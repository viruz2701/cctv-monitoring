-- +migrate Up
-- INV-7.1.2: Custom Fields (JSONB) для SparePart
-- INV-7.1.4: Stock Adjustments с audit trail
--
-- 1. Добавляем custom_fields JSONB колонку в spare_parts
-- 2. Создаём таблицу stock_adjustments для аудита корректировок остатков
--
-- Compliance:
--   - IEC 62443 SL-3 (Zone 3 — Application)
--   - ISO 27001 A.12.4.1 (Event logging — stock_adjustments audit trail)
--   - ISO/IEC 27019 PCC.A.12 (Operations security — inventory changes)
--   - СТБ 34.101.27 (Защита информации — audit trail для складских операций)
--   - OWASP ASVS V6 (Stored cryptography — JSONB для structured data)

-- INV-7.1.2: custom_fields JSONB для spare_parts
ALTER TABLE spare_parts
    ADD COLUMN IF NOT EXISTS custom_fields JSONB DEFAULT '{}';

COMMENT ON COLUMN spare_parts.custom_fields IS
    'INV-7.1.2: Произвольные поля для запчасти (JSONB). '
    'Используется для хранения дополнительных атрибутов, '
    'специфичных для конкретного поставщика или категории.';

-- INV-7.1.4: Stock Adjustments с audit trail
CREATE TABLE stock_adjustments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    part_id        TEXT NOT NULL REFERENCES spare_parts(id) ON DELETE CASCADE,
    previous_stock INT NOT NULL,
    new_stock      INT NOT NULL,
    delta          INT NOT NULL,
    reason         TEXT NOT NULL DEFAULT '',
    adjusted_by    TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stock_adjustments_part_id
    ON stock_adjustments(part_id);

CREATE INDEX IF NOT EXISTS idx_stock_adjustments_created_at
    ON stock_adjustments(created_at DESC);

COMMENT ON TABLE stock_adjustments IS
    'INV-7.1.4: Журнал корректировок остатков запчастей с audit trail. '
    'Каждая запись фиксирует: part_id, previous_stock, new_stock, delta, reason, adjusted_by. '
    'Соответствует ISO 27001 A.12.4.1 (Event logging), IEC 62443 SL-3, СТБ 34.101.27.';

COMMENT ON COLUMN stock_adjustments.id IS 'UUID первичный ключ';
COMMENT ON COLUMN stock_adjustments.part_id IS 'ID запчасти (FK → spare_parts.id)';
COMMENT ON COLUMN stock_adjustments.previous_stock IS 'Остаток до корректировки';
COMMENT ON COLUMN stock_adjustments.new_stock IS 'Остаток после корректировки';
COMMENT ON COLUMN stock_adjustments.delta IS 'Изменение остатка (new_stock - previous_stock)';
COMMENT ON COLUMN stock_adjustments.reason IS 'Причина корректировки (инвентаризация, списание, приход)';
COMMENT ON COLUMN stock_adjustments.adjusted_by IS 'Пользователь, выполнивший корректировку';
COMMENT ON COLUMN stock_adjustments.created_at IS 'Время корректировки (audit trail)';
