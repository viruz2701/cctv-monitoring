-- +migrate Down
-- DM-1.3.1: Откат связи WorkOrder ↔ Alert

DROP INDEX IF EXISTS idx_wo_alerts_alert;
DROP INDEX IF EXISTS idx_wo_alerts_linked_at;
DROP TABLE IF EXISTS work_order_alerts;
