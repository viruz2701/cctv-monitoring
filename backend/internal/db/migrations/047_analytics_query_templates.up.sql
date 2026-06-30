-- +migrate Up
-- P2-BI: Embedded Self-Service Analytics Templates
--
-- Хранит шаблоны BI-запросов для self-service analytics.
-- Позволяет администраторам добавлять кастомные шаблоны через API.
--
-- Compliance:
--   - ISO 27001 A.12.4.1 (Event logging — created_at/updated_at)
--   - ISO 27001 A.12.6.1 (Capacity management — query performance tracking)
--   - OWASP ASVS V7.1 (Error handling — safe storage of SQL templates)
--   - IEC 62443 SR 3.1 (Input validation — template validation on write)

CREATE TABLE analytics_query_templates (
    id              VARCHAR(64) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    sql_template    TEXT NOT NULL,
    dimensions      JSONB NOT NULL DEFAULT '[]',
    measures        JSONB NOT NULL DEFAULT '[]',
    date_field      VARCHAR(128) NOT NULL DEFAULT '',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by      VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- SQL шаблон не может быть пустым
    CONSTRAINT chk_analytics_template_sql CHECK (sql_template != '')
);

-- Индекс для поиска активных шаблонов
CREATE INDEX idx_analytics_templates_active
    ON analytics_query_templates (is_active)
    WHERE is_active = TRUE;

-- Триггер автоматического обновления updated_at
CREATE OR REPLACE FUNCTION trigger_set_analytics_template_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_analytics_templates_updated_at
    BEFORE UPDATE ON analytics_query_templates
    FOR EACH ROW
    EXECUTE FUNCTION trigger_set_analytics_template_updated_at();

-- ── Вставка системных шаблонов ────────────────────────────────────────────────
-- Эти шаблоны создаются миграцией и помечаются is_system = TRUE.
-- Пользовательские шаблоны могут добавляться через API.

INSERT INTO analytics_query_templates (id, name, description, sql_template, dimensions, measures, date_field, is_system) VALUES
(
    'mttr_by_device',
    'MTTR by Device',
    'Mean Time To Repair (minutes) grouped by device — ключевой SLA-показатель для CMMS',
    'SELECT\n\two.device_id,\n\td.name AS device_name,\n\td.vendor_type,\n\td.device_type,\n\tCOALESCE(t.full_name, ''Unassigned'') AS technician_name,\n\tEXTRACT(EPOCH FROM (wo.resolved_at - wo.started_at)) / 60 AS resolution_time_min\nFROM work_orders wo\nJOIN devices d ON d.device_id = wo.device_id\nLEFT JOIN technicians t ON t.id = wo.technician_id\nWHERE wo.status = ''resolved''\n  AND wo.started_at IS NOT NULL\n  AND wo.resolved_at IS NOT NULL',
    '[{"key":"device_id","label":"Device ID","type":"string"},{"key":"device_name","label":"Device Name","type":"string"},{"key":"vendor_type","label":"Vendor","type":"string"},{"key":"device_type","label":"Device Type","type":"string"},{"key":"technician_name","label":"Technician","type":"string"}]',
    '[{"key":"resolution_time_min","label":"Resolution Time (min)","type":"number"},{"key":"avg_mttr_min","label":"Avg MTTR (min)","type":"number","agg":"AVG","sql_expr":"resolution_time_min"},{"key":"max_mttr_min","label":"Max MTTR (min)","type":"number","agg":"MAX","sql_expr":"resolution_time_min"},{"key":"min_mttr_min","label":"Min MTTR (min)","type":"number","agg":"MIN","sql_expr":"resolution_time_min"},{"key":"wo_count","label":"Work Orders","type":"number","agg":"COUNT","sql_expr":"1"}]',
    'wo.resolved_at',
    TRUE
),
(
    'mtbf_by_device',
    'MTBF by Device',
    'Mean Time Between Failures (hours) — надёжность оборудования',
    'SELECT\n\td.device_id,\n\td.name AS device_name,\n\td.vendor_type,\n\td.device_type,\n\tEXTRACT(EPOCH FROM (d.last_seen - d.registered_at)) / 3600 AS uptime_hours,\n\t(SELECT COUNT(*) FROM work_orders wo2 WHERE wo2.device_id = d.device_id AND wo2.category = ''breakdown'') AS failure_count\nFROM devices d\nWHERE d.last_seen IS NOT NULL\n  AND d.registered_at IS NOT NULL',
    '[{"key":"device_id","label":"Device ID","type":"string"},{"key":"device_name","label":"Device Name","type":"string"},{"key":"vendor_type","label":"Vendor","type":"string"},{"key":"device_type","label":"Device Type","type":"string"}]',
    '[{"key":"uptime_hours","label":"Uptime (hours)","type":"number"},{"key":"failure_count","label":"Failures","type":"number"},{"key":"avg_uptime_hours","label":"Avg Uptime (hours)","type":"number","agg":"AVG","sql_expr":"uptime_hours"}]',
    'd.registered_at',
    TRUE
),
(
    'device_uptime',
    'Device Uptime',
    'Текущий аптайм устройств и статус здоровья по сайтам',
    'SELECT\n\td.device_id,\n\td.name AS device_name,\n\td.site_id,\n\ts.name AS site_name,\n\td.status,\n\td.health,\n\td.vendor_type,\n\td.device_type,\n\td.last_seen,\n\tCASE\n\t\tWHEN d.last_seen IS NULL THEN 0\n\t\tWHEN d.last_seen < NOW() - INTERVAL ''24 hours'' THEN 0\n\t\tELSE EXTRACT(EPOCH FROM (NOW() - d.last_seen)) / 3600\n\tEND AS hours_since_last_seen\nFROM devices d\nLEFT JOIN sites s ON s.id = d.site_id',
    '[{"key":"device_id","label":"Device ID","type":"string"},{"key":"device_name","label":"Device Name","type":"string"},{"key":"site_id","label":"Site ID","type":"string"},{"key":"site_name","label":"Site Name","type":"string"},{"key":"status","label":"Status","type":"string"},{"key":"health","label":"Health","type":"string"},{"key":"vendor_type","label":"Vendor","type":"string"},{"key":"device_type","label":"Device Type","type":"string"}]',
    '[{"key":"hours_since_last_seen","label":"Hours Since Last Seen","type":"number"},{"key":"device_count","label":"Device Count","type":"number","agg":"COUNT","sql_expr":"1"},{"key":"avg_hours_since_last_seen","label":"Avg Hours Since Last Seen","type":"number","agg":"AVG","sql_expr":"hours_since_last_seen"}]',
    'd.last_seen',
    TRUE
),
(
    'work_order_summary',
    'Work Order Summary',
    'Агрегированная сводка по Work Orders: статусы, приоритеты, категории',
    'SELECT\n\two.id AS wo_id,\n\two.device_id,\n\td.name AS device_name,\n\td.site_id,\n\ts.name AS site_name,\n\two.status,\n\two.priority,\n\two.category,\n\two.technician_id,\n\tCOALESCE(t.full_name, ''Unassigned'') AS technician_name,\n\tEXTRACT(EPOCH FROM (COALESCE(wo.resolved_at, NOW()) - wo.created_at)) / 3600 AS age_hours,\n\two.total_cost\nFROM work_orders wo\nJOIN devices d ON d.device_id = wo.device_id\nLEFT JOIN sites s ON s.id = d.site_id\nLEFT JOIN technicians t ON t.id = wo.technician_id',
    '[{"key":"wo_id","label":"Work Order ID","type":"string"},{"key":"device_id","label":"Device ID","type":"string"},{"key":"device_name","label":"Device Name","type":"string"},{"key":"site_id","label":"Site ID","type":"string"},{"key":"site_name","label":"Site Name","type":"string"},{"key":"status","label":"Status","type":"string"},{"key":"priority","label":"Priority","type":"string"},{"key":"category","label":"Category","type":"string"},{"key":"technician_name","label":"Technician","type":"string"}]',
    '[{"key":"age_hours","label":"Age (hours)","type":"number"},{"key":"total_cost","label":"Total Cost","type":"number"},{"key":"wo_count","label":"Work Orders","type":"number","agg":"COUNT","sql_expr":"1"},{"key":"total_cost_sum","label":"Total Cost Sum","type":"number","agg":"SUM","sql_expr":"total_cost"},{"key":"avg_cost","label":"Avg Cost","type":"number","agg":"AVG","sql_expr":"total_cost"},{"key":"max_age_hours","label":"Max Age (hours)","type":"number","agg":"MAX","sql_expr":"age_hours"}]',
    'wo.created_at',
    TRUE
),
(
    'tco_by_device',
    'TCO by Device',
    'Total Cost of Ownership: purchase + labor + parts + downtime',
    'SELECT\n\td.device_id,\n\td.name AS device_name,\n\td.vendor_type,\n\td.device_type,\n\td.site_id,\n\ts.name AS site_name,\n\tCOALESCE(tco.purchase_cost, 0) AS purchase_cost,\n\tCOALESCE(tco.labor_cost, 0) AS labor_cost,\n\tCOALESCE(tco.parts_cost, 0) AS parts_cost,\n\tCOALESCE(tco.downtime_cost, 0) AS downtime_cost,\n\tCOALESCE(tco.purchase_cost, 0) + COALESCE(tco.labor_cost, 0) + COALESCE(tco.parts_cost, 0) + COALESCE(tco.downtime_cost, 0) AS tco\nFROM devices d\nLEFT JOIN sites s ON s.id = d.site_id\nLEFT JOIN mv_tco_per_device tco ON tco.device_id = d.device_id',
    '[{"key":"device_id","label":"Device ID","type":"string"},{"key":"device_name","label":"Device Name","type":"string"},{"key":"vendor_type","label":"Vendor","type":"string"},{"key":"device_type","label":"Device Type","type":"string"},{"key":"site_id","label":"Site ID","type":"string"},{"key":"site_name","label":"Site Name","type":"string"}]',
    '[{"key":"purchase_cost","label":"Purchase Cost","type":"number"},{"key":"labor_cost","label":"Labor Cost","type":"number"},{"key":"parts_cost","label":"Parts Cost","type":"number"},{"key":"downtime_cost","label":"Downtime Cost","type":"number"},{"key":"tco","label":"Total TCO","type":"number"},{"key":"avg_tco","label":"Avg TCO","type":"number","agg":"AVG","sql_expr":"tco"},{"key":"total_tco","label":"Total TCO Sum","type":"number","agg":"SUM","sql_expr":"tco"}]',
    'd.registered_at',
    TRUE
);
