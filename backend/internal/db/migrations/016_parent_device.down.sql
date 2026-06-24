-- Migration 016: Down — revert device hierarchy
-- +migrate Down

DROP INDEX IF EXISTS idx_devices_hierarchy_level;
DROP INDEX IF EXISTS idx_devices_parent_device;
ALTER TABLE devices DROP COLUMN IF EXISTS hierarchy_level;
ALTER TABLE devices DROP COLUMN IF EXISTS parent_device_id;
