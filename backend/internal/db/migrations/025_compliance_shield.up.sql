-- +migrate Up
-- KF-15.1.1: Compliance & Fines Shield
--
-- Конвертирует downtime CCTV-камер в денежный риск ($/час штрафа).
--
-- Compliance:
--   - IEC 62443-3-3 SR 7.1 (Resource availability — risk quantification)
--   - ISO 27001 A.12.4 (Audit trail — логирование расчётов)
--   - ISO 27019 PCC.A.13 (ICS asset risk assessment)
--   - СТБ 34.101.27 п. 6.3 (Оценка рисков)
--   - OWASP ASVS V6 (Stored cryptography — belt-gcm для audit)
--   - Приказ ОАЦ № 66 п. 7.18 (Идентификация устройств)

-- ── Compliance Risks (материализованное представление) ─────────────────

CREATE TABLE IF NOT EXISTS compliance_risks (
    device_id           TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    site_id             TEXT REFERENCES sites(id) ON DELETE SET NULL,
    device_type         TEXT NOT NULL DEFAULT 'camera',
    total_downtime_min  BIGINT NOT NULL DEFAULT 0,
    hourly_fine         DECIMAL(12,2) NOT NULL DEFAULT 0,
    total_exposure      DECIMAL(12,2) NOT NULL DEFAULT 0,
    risk_level          TEXT NOT NULL DEFAULT 'low'
                        CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (device_id)
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_compliance_risks_site
    ON compliance_risks (site_id);
CREATE INDEX IF NOT EXISTS idx_compliance_risks_risk_level
    ON compliance_risks (risk_level);
CREATE INDEX IF NOT EXISTS idx_compliance_risks_exposure
    ON compliance_risks (total_exposure DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_risks_updated
    ON compliance_risks (updated_at DESC);

-- Комментарии
COMMENT ON TABLE compliance_risks IS
    'KF-15.1.1: Compliance & Fines Shield. Содержит расчёт финансовых рисков '
    'на основе downtime устройств. Обновляется через refresh_compliance_risks().';

COMMENT ON COLUMN compliance_risks.total_downtime_min IS
    'Общее время простоя устройства в минутах';
COMMENT ON COLUMN compliance_risks.hourly_fine IS
    'Почасовой штраф для данного типа устройства ($/час)';
COMMENT ON COLUMN compliance_risks.total_exposure IS
    'Общий финансовый риск = (downtime_min / 60) * hourly_fine';
COMMENT ON COLUMN compliance_risks.risk_level IS
    'Уровень риска: low (<$1000), medium ($1000-5000), high ($5000-25000), critical (>$25000)';

-- ── Compliance Audit Log ──────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS compliance_audit_log (
    id              BIGSERIAL,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    device_id       TEXT NOT NULL,
    site_id         TEXT,
    total_exposure  DECIMAL(12,2) NOT NULL DEFAULT 0,
    risk_level      TEXT NOT NULL DEFAULT 'low',
    details         JSONB DEFAULT '{}'::jsonb,
    trace_id        TEXT DEFAULT '',
    PRIMARY KEY (id, recorded_at)
);

-- TimescaleDB hypertable для аудита compliance
SELECT create_hypertable('compliance_audit_log', 'recorded_at',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_compliance_audit_device
    ON compliance_audit_log (device_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_compliance_audit_risk
    ON compliance_audit_log (risk_level, recorded_at DESC);

COMMENT ON TABLE compliance_audit_log IS
    'ISO 27001 A.12.4: Audit trail для compliance расчётов. '
    'Retention: 7 лет (КИИ РБ). TimescaleDB hypertable с интервалом 7 дней.';

-- ── Функция обновления compliance_risks ───────────────────────────────

CREATE OR REPLACE FUNCTION refresh_compliance_risks()
RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_device RECORD;
    v_hourly_fine DECIMAL(12,2);
    v_total_exposure DECIMAL(12,2);
    v_risk_level TEXT;
    v_fines JSONB;
BEGIN
    v_fines := jsonb_build_object(
        'cash_register', 500,
        'perimeter', 200,
        'warehouse', 300,
        'office', 100,
        'camera', 100,
        'nvr', 250,
        'dvr', 200,
        'switch', 150,
        'server', 400,
        'encoder', 180,
        'ups', 120
    );

    FOR v_device IN
        SELECT
            d.device_id,
            d.site_id,
            COALESCE(d.device_type, 'camera') as device_type,
            COALESCE(SUM(dt.duration_minutes), 0)::bigint as total_downtime_min
        FROM devices d
        LEFT JOIN asset_downtime dt ON d.device_id = dt.device_id
            AND dt.status = 'recovered'
            AND dt.started_at >= NOW() - INTERVAL '90 days'
        GROUP BY d.device_id, d.site_id, d.device_type
    LOOP
        -- Определяем почасовой штраф
        v_hourly_fine := COALESCE(
            (v_fines->>v_device.device_type)::decimal,
            100.0  -- fallback
        );

        -- Расчёт экспозиции
        v_total_exposure := ROUND((v_device.total_downtime_min::decimal / 60.0) * v_hourly_fine, 2);

        -- Определение уровня риска
        v_risk_level := CASE
            WHEN v_total_exposure >= 25000 THEN 'critical'
            WHEN v_total_exposure >= 5000 THEN 'high'
            WHEN v_total_exposure >= 1000 THEN 'medium'
            ELSE 'low'
        END;

        -- UPSERT
        INSERT INTO compliance_risks (
            device_id, site_id, device_type,
            total_downtime_min, hourly_fine,
            total_exposure, risk_level, updated_at
        ) VALUES (
            v_device.device_id, v_device.site_id, v_device.device_type,
            v_device.total_downtime_min, v_hourly_fine,
            v_total_exposure, v_risk_level, NOW()
        )
        ON CONFLICT (device_id) DO UPDATE SET
            site_id = EXCLUDED.site_id,
            device_type = EXCLUDED.device_type,
            total_downtime_min = EXCLUDED.total_downtime_min,
            hourly_fine = EXCLUDED.hourly_fine,
            total_exposure = EXCLUDED.total_exposure,
            risk_level = EXCLUDED.risk_level,
            updated_at = NOW();
    END LOOP;
END;
$$;

COMMENT ON FUNCTION refresh_compliance_risks() IS
    'KF-15.1.1: Обновляет compliance_risks из asset_downtime. '
    'Анализирует последние 90 дней простоев.';

-- ── Первоначальное заполнение ─────────────────────────────────────────

SELECT refresh_compliance_risks();
