-- Migration 007: Add site fields + spare_part_categories table
-- +migrate Up

-- 1. Add organization, latitude, longitude to sites
ALTER TABLE sites ADD COLUMN IF NOT EXISTS organization TEXT;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS latitude DOUBLE PRECISION DEFAULT 0;
ALTER TABLE sites ADD COLUMN IF NOT EXISTS longitude DOUBLE PRECISION DEFAULT 0;

-- 2. Create spare_part_categories table
CREATE TABLE spare_part_categories (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name TEXT NOT NULL,
    description TEXT,
    color TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
