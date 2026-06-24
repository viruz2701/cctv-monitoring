-- Migration 016: Add device hierarchy parent_device_id (AH-5.2.1)
-- Иерархия: Site → Switch → NVR → Camera
-- Соответствует: ISO 27001 A.8.1.1 (Asset inventory — hierarchy tracking)
-- +migrate Up

ALTER TABLE devices ADD COLUMN IF NOT EXISTS parent_device_id TEXT REFERENCES devices(device_id) ON DELETE SET NULL;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS hierarchy_level INT DEFAULT 0;
-- hierarchy_level: 0=site, 1=switch, 2=nvr, 3=camera

CREATE INDEX IF NOT EXISTS idx_devices_parent_device ON devices(parent_device_id);
CREATE INDEX IF NOT EXISTS idx_devices_hierarchy_level ON devices(hierarchy_level);

-- +migrate Down

DROP INDEX IF EXISTS idx_devices_hierarchy_level;
DROP INDEX IF EXISTS idx_devices_parent_device;
ALTER TABLE devices DROP COLUMN IF EXISTS hierarchy_level;
ALTER TABLE devices DROP COLUMN IF EXISTS parent_device_id;
