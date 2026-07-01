-- +migrate Down
-- Migration 058: Calendar Sync Idempotency (P1-HI-09) — rollback

DROP INDEX IF EXISTS idx_calendar_sync_log_idempotency;

ALTER TABLE calendar_sync_log
    DROP COLUMN IF EXISTS idempotency_key;
