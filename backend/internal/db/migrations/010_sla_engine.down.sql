-- +migrate Down
-- Rollback migration 010: Advanced SLA Engine
DROP TABLE IF EXISTS sla_tracker_state CASCADE;
DROP TABLE IF EXISTS sla_pause_rules CASCADE;
DROP TABLE IF EXISTS sla_business_calendars CASCADE;
DROP TABLE IF EXISTS sla_matrix_entries CASCADE;
DROP TABLE IF EXISTS sla_policies CASCADE;
