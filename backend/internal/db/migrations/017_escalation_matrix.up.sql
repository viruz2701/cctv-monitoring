-- +migrate Up
-- Migration 017: Escalation Matrix (SLA-6.2.2)
--
-- Добавляет эскалационную матрицу с 3 уровнями для SLA engine.
-- Определяет правила уведомлений при breach SLA в зависимости от
-- priority и escalation_level.
--
-- Compliance:
--   ISO 27001 A.12.4.1 (Event logging — escalation audit trail)
--   ISO 27001 A.12.6.1 (Capacity management — SLA escalation)
--   IEC 62443 SR 2.8 (Audit events — escalation tracking)
--   IEC 62443 SR 7.1 (Resource availability — SLA escalation)
--   СТБ 34.101.27 (Защита информации — audit log с HMAC)
--   OWASP ASVS V7.1 (Structured logging)
--   Приказ ОАЦ №66 п. 7.18.3 (Incident response escalation)

-- ── Escalation Rules ─────────────────────────────────────────────────

CREATE TABLE sla_escalation_rules (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    priority TEXT NOT NULL CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    escalation_level INT NOT NULL CHECK (escalation_level BETWEEN 1 AND 3),
    breach_minutes INT NOT NULL, -- через сколько минут после дедлайна
    notify_role TEXT NOT NULL,   -- manager, director, emergency
    notify_channel TEXT DEFAULT 'telegram', -- telegram, email, both
    repeat_interval_minutes INT DEFAULT 0,  -- 0 = одноразово
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Default escalation rules
INSERT INTO sla_escalation_rules (priority, escalation_level, breach_minutes, notify_role, notify_channel) VALUES
    ('critical', 1, 0, 'manager', 'both'),
    ('critical', 2, 30, 'director', 'both'),
    ('critical', 3, 60, 'emergency', 'both'),
    ('high', 1, 15, 'manager', 'telegram'),
    ('high', 2, 60, 'director', 'telegram'),
    ('medium', 1, 30, 'manager', 'telegram'),
    ('medium', 2, 120, 'director', 'telegram'),
    ('low', 1, 60, 'manager', 'telegram');

CREATE INDEX IF NOT EXISTS idx_escalation_rules_priority ON sla_escalation_rules(priority, escalation_level);

COMMENT ON TABLE sla_escalation_rules IS 'Правила эскалации для SLA breach (SLA-6.2.2)';
COMMENT ON COLUMN sla_escalation_rules.breach_minutes IS 'Через сколько минут после дедлайна срабатывает эскалация';
COMMENT ON COLUMN sla_escalation_rules.notify_role IS 'Роль уведомляемого: manager, director, emergency';
COMMENT ON COLUMN sla_escalation_rules.notify_channel IS 'Канал уведомления: telegram, email, both';
COMMENT ON COLUMN sla_escalation_rules.repeat_interval_minutes IS 'Интервал повторения: 0 = одноразово';

-- ── Escalation Log ───────────────────────────────────────────────────

CREATE TABLE sla_escalation_log (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    work_order_id TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    escalation_level INT NOT NULL,
    rule_id TEXT REFERENCES sla_escalation_rules(id),
    notified_at TIMESTAMPTZ DEFAULT NOW(),
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by TEXT,
    resolution_notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_escalation_log_wo ON sla_escalation_log(work_order_id);
CREATE INDEX IF NOT EXISTS idx_escalation_log_notified ON sla_escalation_log(notified_at);

COMMENT ON TABLE sla_escalation_log IS 'Журнал эскалаций SLA breach (SLA-6.2.2)';
COMMENT ON COLUMN sla_escalation_log.escalation_level IS 'Уровень эскалации: 1, 2, 3';
COMMENT ON COLUMN sla_escalation_log.rule_id IS 'ID правила эскалации, по которому сработало';
COMMENT ON COLUMN sla_escalation_log.notified_at IS 'Время отправки уведомления';
COMMENT ON COLUMN sla_escalation_log.acknowledged_at IS 'Время подтверждения получения';
COMMENT ON COLUMN sla_escalation_log.acknowledged_by IS 'Кто подтвердил (user_id)';
COMMENT ON COLUMN sla_escalation_log.resolution_notes IS 'Заметки по разрешению эскалации';

-- ── Add escalation_level to existing sla_tracker_state (если есть) ───

ALTER TABLE sla_escalation_log ADD COLUMN IF NOT EXISTS escalation_level INT DEFAULT 0;
