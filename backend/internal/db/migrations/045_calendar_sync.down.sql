-- ═══════════════════════════════════════════════════════════════════════
-- P1-CALENDAR: External Calendar Sync — Down Migration
-- ═══════════════════════════════════════════════════════════════════════

-- +migrate Down

DROP TABLE IF EXISTS calendar_sync_log CASCADE;
DROP TABLE IF EXISTS calendar_events CASCADE;
DROP TABLE IF EXISTS calendar_connections CASCADE;
