-- +migrate Down
-- Rollback migration 011: Meter tables
DROP TABLE IF EXISTS meter_trigger_fired CASCADE;
DROP TABLE IF EXISTS meter_triggers CASCADE;
DROP TABLE IF EXISTS meter_readings CASCADE;
DROP TABLE IF EXISTS meters CASCADE;
