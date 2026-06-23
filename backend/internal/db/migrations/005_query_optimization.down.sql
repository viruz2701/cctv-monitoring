-- +migrate Down
DROP VIEW IF EXISTS slow_queries_top10;
DROP INDEX IF EXISTS idx_devices_site_status;
DROP INDEX IF EXISTS idx_devices_name_trgm;
DROP INDEX IF EXISTS idx_devices_vendor_status;
DROP INDEX IF EXISTS idx_work_orders_status_priority;
DROP INDEX IF EXISTS idx_work_orders_assigned_status;
DROP INDEX IF EXISTS idx_work_orders_sla_active;
DROP INDEX IF EXISTS idx_maintenance_schedules_next_due_active;
DROP INDEX IF EXISTS idx_alarms_time_status;
DROP INDEX IF EXISTS idx_audit_log_timestamp_action;
