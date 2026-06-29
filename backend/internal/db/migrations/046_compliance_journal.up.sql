-- +migrate Up
-- P0-REG.3-5: Maintenance Compliance Engine — Electronic Journal
--
-- Создаёт таблицу compliance_journal для HMAC-подписанных актов ТО.
-- Обеспечивает tamper-evident audit trail для compliance-отчётности.
--
-- Compliance:
--   - ISO 27001 A.12.4 (Logging and Monitoring — audit trail)
--   - ISO 27019 PCC.A.12 (ICS compliance logging)
--   - IEC 62443-3-3 SR 3.1 (Audit log integrity)
--   - СТБ 34.101.27 п. 7.2 (Защита журналов аудита)
--   - Приказ ОАЦ № 66 п. 7.18.3 (Tamper-evident logging)
--   - OWASP ASVS V7 (Log content and integrity)

-- ═══════════════════════════════════════════════════════════════════════════
-- P0-REG.4: compliance_journal — электронный журнал ТО
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE compliance_journal (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    regulation_id   TEXT REFERENCES maintenance_regulations(id) ON DELETE SET NULL,
    wo_id           TEXT REFERENCES work_orders(id) ON DELETE SET NULL,
    region_code     VARCHAR(2) NOT NULL,
    act_data        JSONB NOT NULL,
    hmac_signature  TEXT,
    hmac_signed_at  TIMESTAMPTZ,
    verified_at     TIMESTAMPTZ,
    verified_by     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX idx_compliance_journal_region
    ON compliance_journal (region_code);
CREATE INDEX idx_compliance_journal_regulation
    ON compliance_journal (regulation_id);
CREATE INDEX idx_compliance_journal_wo
    ON compliance_journal (wo_id);
CREATE INDEX idx_compliance_journal_created
    ON compliance_journal (created_at DESC);
CREATE INDEX idx_compliance_journal_signed
    ON compliance_journal (hmac_signed_at)
    WHERE hmac_signature IS NOT NULL;

COMMENT ON TABLE compliance_journal IS
    'P0-REG.4: Электронный журнал ТО с HMAC-подписанными актами. '
    'Обеспечивает tamper-evident audit trail для compliance.';

COMMENT ON COLUMN compliance_journal.regulation_id IS
    'Ссылка на регламент ТО (maintenance_regulations)';
COMMENT ON COLUMN compliance_journal.wo_id IS
    'Ссылка на work order, в рамках которого выполнен ТО';
COMMENT ON COLUMN compliance_journal.region_code IS
    'ISO 3166-1 alpha-2 код региона (BY, RU, TR, VN, ID, BR, ZA)';
COMMENT ON COLUMN compliance_journal.act_data IS
    'JSONB с данными акта ТО (checklist, результаты, замечания)';
COMMENT ON COLUMN compliance_journal.hmac_signature IS
    'HMAC подпись акта (bash-256 / SHA-256 placeholder)';
COMMENT ON COLUMN compliance_journal.hmac_signed_at IS
    'Дата и время подписания акта HMAC';
COMMENT ON COLUMN compliance_journal.verified_at IS
    'Дата последней верификации подписи';
COMMENT ON COLUMN compliance_journal.verified_by IS
    'Кто верифицировал (user_id)';

-- ═══════════════════════════════════════════════════════════════════════════
-- P0-REG.5: Функция для авто-генерации WO из регламентов
-- ═══════════════════════════════════════════════════════════════════════════

-- get_due_regulations возвращает активные регламенты, у которых
-- интервал ТО истёк относительно last_maintenance_date.
-- Если last_maintenance_date NULL — берётся created_at.
CREATE OR REPLACE FUNCTION get_due_regulations()
RETURNS TABLE (
    id                  TEXT,
    region_code         VARCHAR(2),
    regulation_code     VARCHAR(20),
    name                TEXT,
    regulation_type     VARCHAR(4),
    interval_months     INT,
    estimated_minutes   INT,
    total_items         INT,
    compliance_standards TEXT[],
    license_requirements TEXT,
    docs_required       JSONB,
    last_maintenance_date TIMESTAMPTZ,
    days_overdue        INT
)
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    reg RECORD;
BEGIN
    FOR reg IN
        SELECT mr.*, mc.last_maintenance_date
        FROM maintenance_regulations mr
        LEFT JOIN LATERAL (
            SELECT MAX(cj.created_at) AS last_maintenance_date
            FROM compliance_journal cj
            WHERE cj.regulation_id = mr.id
        ) mc ON true
        WHERE mr.is_active = true
    LOOP
        -- Если нет last_maintenance_date — берём created_at регламента
        -- Если created_at + interval_months прошёл — регламент просрочен
        IF reg.last_maintenance_date IS NULL THEN
            IF reg.created_at + (reg.interval_months || ' months')::INTERVAL <= NOW() THEN
                id := reg.id;
                region_code := reg.region_code;
                regulation_code := reg.regulation_code;
                name := reg.name;
                regulation_type := reg.regulation_type;
                interval_months := reg.interval_months;
                estimated_minutes := reg.estimated_minutes;
                total_items := reg.total_items;
                compliance_standards := reg.compliance_standards;
                license_requirements := reg.license_requirements;
                docs_required := reg.docs_required;
                last_maintenance_date := reg.created_at;
                days_overdue := EXTRACT(DAY FROM NOW() - (reg.created_at + (reg.interval_months || ' months')::INTERVAL));
                RETURN NEXT;
            END IF;
        ELSE
            IF reg.last_maintenance_date + (reg.interval_months || ' months')::INTERVAL <= NOW() THEN
                id := reg.id;
                region_code := reg.region_code;
                regulation_code := reg.regulation_code;
                name := reg.name;
                regulation_type := reg.regulation_type;
                interval_months := reg.interval_months;
                estimated_minutes := reg.estimated_minutes;
                total_items := reg.total_items;
                compliance_standards := reg.compliance_standards;
                license_requirements := reg.license_requirements;
                docs_required := reg.docs_required;
                last_maintenance_date := reg.last_maintenance_date;
                days_overdue := EXTRACT(DAY FROM NOW() - (reg.last_maintenance_date + (reg.interval_months || ' months')::INTERVAL));
                RETURN NEXT;
            END IF;
        END IF;
    END LOOP;
END;
$$;

COMMENT ON FUNCTION get_due_regulations() IS
    'P0-REG.3: Возвращает активные регламенты ТО, у которых истёк интерлан.';

-- ═══════════════════════════════════════════════════════════════════════════
-- Функция: log_compliance_audit — запись в compliance audit log
-- ═══════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION log_compliance_audit(
    p_action        TEXT,
    p_regulation_id TEXT,
    p_wo_id         TEXT,
    p_region_code   VARCHAR(2),
    p_details       JSONB,
    p_trace_id      TEXT
) RETURNS UUID
LANGUAGE plpgsql
AS $$
DECLARE
    v_id UUID;
BEGIN
    INSERT INTO compliance_journal (
        regulation_id, wo_id, region_code, act_data
    ) VALUES (
        p_regulation_id, p_wo_id, p_region_code,
        jsonb_build_object(
            'action', p_action,
            'details', COALESCE(p_details, '{}'::jsonb),
            'trace_id', p_trace_id,
            'timestamp', NOW()
        )
    ) RETURNING id INTO v_id;

    RETURN v_id;
END;
$$;

COMMENT ON FUNCTION log_compliance_audit(TEXT, TEXT, TEXT, VARCHAR(2), JSONB, TEXT) IS
    'P0-REG.3: Записывает событие в compliance_journal.';

