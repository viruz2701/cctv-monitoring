-- Migration 004: Add soft delete support for devices
-- Соответствует: ISO 27001 A.8.1.2 (Asset disposal), GDPR Art. 17
-- +migrate Up

ALTER TABLE devices ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Index for soft delete queries
CREATE INDEX IF NOT EXISTS idx_devices_deleted_at ON devices(deleted_at)
  WHERE deleted_at IS NULL;

-- Update device status check constraint to include DELETED if needed
-- (keeping existing constraints, soft delete just sets deleted_at and status='OFFLINE')
