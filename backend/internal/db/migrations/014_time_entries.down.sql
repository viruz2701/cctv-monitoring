-- Migration 014: Down — revert time entries
-- +migrate Down

DROP INDEX IF EXISTS idx_work_orders_total_cost;
DROP INDEX IF EXISTS idx_time_entries_status;
DROP INDEX IF EXISTS idx_time_entries_user;
DROP INDEX IF EXISTS idx_time_entries_work_order;

ALTER TABLE work_orders DROP COLUMN IF EXISTS total_cost;
ALTER TABLE work_orders DROP COLUMN IF EXISTS total_parts_cost;
ALTER TABLE work_orders DROP COLUMN IF EXISTS total_labor_seconds;
ALTER TABLE work_orders DROP COLUMN IF EXISTS total_labor_cost;

DROP TABLE IF EXISTS time_entries;
