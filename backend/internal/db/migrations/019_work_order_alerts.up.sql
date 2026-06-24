-- +migrate Up
-- DM-1.3.1: WorkOrder ↔ Alert (Many-to-Many)
--
-- Таблица уже создана в 008 с колонками (id, work_order_id, alert_id, created_at).
-- Добавляем недостающие колонки для версии 019.
--
-- Compliance:
--   - IEC 62443 SL-3 (Zone 3 — Application)
--   - ISO 27001 A.12.4.1 (Event logging — linked_at audit trail)
--   - СТБ 34.101.27 (Защита информации — связь инцидентов)

ALTER TABLE work_order_alerts ADD COLUMN IF NOT EXISTS alert_id_text TEXT;
ALTER TABLE work_order_alerts ADD COLUMN IF NOT EXISTS linked_at TIMESTAMPTZ DEFAULT NOW();
ALTER TABLE work_order_alerts ADD COLUMN IF NOT EXISTS linked_by TEXT;

CREATE INDEX IF NOT EXISTS idx_wo_alerts_alert ON work_order_alerts(alert_id);
CREATE INDEX IF NOT EXISTS idx_wo_alerts_linked_at ON work_order_alerts(linked_at);
