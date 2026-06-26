-- +migrate Down
-- Migration 028: Black Box Incident Reports (Rollback)

DROP TABLE IF EXISTS incident_triggers;
DROP TABLE IF EXISTS incident_reports;
