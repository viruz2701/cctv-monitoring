-- +migrate Down
-- Migration 017: Rollback Escalation Matrix
--
-- Compliance: ISO 27001 A.12.4.1 (retention before drop)

-- Удаляем escalation log (с проверкой retention)
ALTER TABLE sla_escalation_log DROP COLUMN IF EXISTS escalation_level;

DROP TABLE IF EXISTS sla_escalation_log;
DROP TABLE IF EXISTS sla_escalation_rules;
