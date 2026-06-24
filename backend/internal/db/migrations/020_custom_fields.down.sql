-- +migrate Down
-- INV-7.1.4: удаляем таблицу stock_adjustments
-- INV-7.1.2: удаляем колонку custom_fields

DROP TABLE IF EXISTS stock_adjustments;

ALTER TABLE spare_parts
    DROP COLUMN IF EXISTS custom_fields;
