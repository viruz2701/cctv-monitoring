-- Migration 007: Down — revert site fields + spare_part_categories
-- +migrate Down

-- 1. Remove columns from sites
ALTER TABLE sites DROP COLUMN IF EXISTS organization;
ALTER TABLE sites DROP COLUMN IF EXISTS latitude;
ALTER TABLE sites DROP COLUMN IF EXISTS longitude;

-- 2. Drop spare_part_categories table
DROP TABLE IF EXISTS spare_part_categories;
