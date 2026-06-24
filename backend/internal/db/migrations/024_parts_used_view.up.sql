-- +migrate Up
-- Migration 024: Parts Used View (WO-4.4.5)
--
-- Создаёт представление parts_used на основе таблицы parts_consumption
-- для обеспечения обратной совместимости с аналитическими запросами
-- (GetWorkOrderCostSummary, GetWorkOrderCostBreakdown).
--
-- Compliance:
--   - IEC 62443 SR 3.1 (Data integrity — view abstraction layer)
--   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
--   - OWASP ASVS V5.1 (Parameterized queries в DB слое)

CREATE OR REPLACE VIEW parts_used AS
SELECT * FROM parts_consumption;
