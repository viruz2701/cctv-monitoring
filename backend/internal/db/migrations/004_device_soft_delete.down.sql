-- +migrate Down
DROP INDEX IF EXISTS idx_devices_deleted_at;
ALTER TABLE devices DROP COLUMN IF EXISTS deleted_at;
