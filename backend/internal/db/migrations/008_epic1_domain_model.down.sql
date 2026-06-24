-- Migration 008: Rollback Domain Model Evolution
-- +migrate Down

DROP TABLE IF EXISTS parts_consumption CASCADE;
DROP TABLE IF EXISTS additional_costs CASCADE;
DROP TABLE IF EXISTS labor_costs CASCADE;
DROP TABLE IF EXISTS time_entries CASCADE;
DROP TABLE IF EXISTS preventive_maintenance CASCADE;
DROP TABLE IF EXISTS requests CASCADE;
DROP TABLE IF EXISTS work_order_alerts CASCADE;
DROP TABLE IF EXISTS work_order_relations CASCADE;
DROP TABLE IF EXISTS work_order_history CASCADE;

-- Restore work_orders to original state
ALTER TABLE work_orders DROP COLUMN IF EXISTS deleted_by;
ALTER TABLE work_orders DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE work_orders DROP COLUMN IF EXISTS due_date;
ALTER TABLE work_orders DROP COLUMN IF EXISTS title;

ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_status_check;
ALTER TABLE work_orders ADD CONSTRAINT work_orders_status_check
    CHECK (status IN ('open', 'in_progress', 'completed', 'cancelled'));

ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_type_check;
ALTER TABLE work_orders ADD CONSTRAINT work_orders_type_check
    CHECK (type IN ('preventive', 'corrective', 'emergency'));

ALTER TABLE work_orders DROP CONSTRAINT IF EXISTS work_orders_priority_check;
ALTER TABLE work_orders ADD CONSTRAINT work_orders_priority_check
    CHECK (priority IN ('critical', 'high', 'medium', 'low'));

DROP INDEX IF EXISTS idx_work_orders_deleted_at;
DROP INDEX IF EXISTS idx_work_orders_due_date;
