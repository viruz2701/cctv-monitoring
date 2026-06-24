-- Migration 015: Down — revert parent_location hierarchy
-- +migrate Down

DROP INDEX IF EXISTS idx_sites_parent_location;
ALTER TABLE sites DROP COLUMN IF EXISTS location_type;
ALTER TABLE sites DROP COLUMN IF EXISTS parent_location_id;
