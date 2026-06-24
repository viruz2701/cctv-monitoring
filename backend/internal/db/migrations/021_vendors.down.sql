-- +migrate Down
-- INV-7.2.1: Откат — удаление таблицы vendors и колонки vendor_id из spare_parts

DROP INDEX IF EXISTS idx_spare_parts_vendor;

ALTER TABLE spare_parts DROP COLUMN IF EXISTS vendor_id;

DROP TABLE IF EXISTS vendors;
