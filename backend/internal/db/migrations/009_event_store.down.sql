-- +migrate Down
-- Rollback migration 009: Event Store metadata table
DROP TABLE IF EXISTS event_store_metadata;
