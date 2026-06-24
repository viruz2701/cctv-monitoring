-- Migration 015: Add location hierarchy to sites (AH-5.1.1)
-- Иерархия: Building → Floor → Room → Rack
-- Соответствует: ISO 27001 A.8.1.1 (Asset inventory — location tracking)
-- +migrate Up

ALTER TABLE sites ADD COLUMN IF NOT EXISTS parent_location_id TEXT REFERENCES sites(id) ON DELETE SET NULL;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS location_type TEXT NOT NULL DEFAULT 'building'
    CHECK (location_type IN ('building', 'floor', 'room', 'rack'));

CREATE INDEX IF NOT EXISTS idx_sites_parent_location ON sites(parent_location_id);
CREATE INDEX IF NOT EXISTS idx_sites_location_type ON sites(location_type);

-- +migrate Down

DROP INDEX IF EXISTS idx_sites_location_type;
DROP INDEX IF EXISTS idx_sites_parent_location;
ALTER TABLE sites DROP COLUMN IF EXISTS location_type;
ALTER TABLE sites DROP COLUMN IF EXISTS parent_location_id;
