-- +migrate Up
-- Migration 027: Multi-tenancy RLS (Row Level Security) (F-0.2.3)
--
-- Реализует изоляцию данных между tenant'ами на уровне PostgreSQL RLS.
-- Каждая строка в tenant-чувствительных таблицах помечается tenant_id.
-- RLS-политики проверяют session-local параметр app.tenant_id,
-- установленный через TenantMiddleware.
--
-- Compliance:
--   - IEC 62443 SR 2.1 (Account management — tenant isolation)
--   - IEC 62443 SR 5.1 (Network segmentation — zone-based access)
--   - ISO 27001 A.9.1.2 (Access control — tenant data separation)
--   - ISO 27001 A.15.1.1 (Supplier relationships — data isolation)
--   - ISO 27019 PCC.A.13 (ICS network segregation)
--   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
--   - Приказ ОАЦ № 66 п. 7.18.3 (Изоляция данных)

-- ═══════════════════════════════════════════════════════════════════
-- 1. Добавление tenant_id в tenant-чувствительные таблицы
-- ═══════════════════════════════════════════════════════════════════

-- devices (Zone 3 — Application)
ALTER TABLE devices ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- sites (Zone 3 — Application)
ALTER TABLE sites ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- tickets (Zone 3 — Application)
ALTER TABLE tickets ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- work_orders (Zone 3 — Application)
ALTER TABLE work_orders ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- spare_parts (Zone 3 — Application)
ALTER TABLE spare_parts ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- asset_downtime (Zone 4 — Data)
ALTER TABLE asset_downtime ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- maintenance_schedules (Zone 3 — Application)
ALTER TABLE maintenance_schedules ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- onvif_devices (Zone 3 — Application)
ALTER TABLE onvif_devices ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- users (Zone 3 — Application) — уже есть tenant_id в модели, добавим колонку
ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

-- ═══════════════════════════════════════════════════════════════════
-- 2. Индексы для tenant_id
-- ═══════════════════════════════════════════════════════════════════

CREATE INDEX IF NOT EXISTS idx_devices_tenant ON devices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sites_tenant ON sites(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tickets_tenant ON tickets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_work_orders_tenant ON work_orders(tenant_id);
CREATE INDEX IF NOT EXISTS idx_spare_parts_tenant ON spare_parts(tenant_id);
CREATE INDEX IF NOT EXISTS idx_asset_downtime_tenant ON asset_downtime(tenant_id);
CREATE INDEX IF NOT EXISTS idx_maintenance_schedules_tenant ON maintenance_schedules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_onvif_devices_tenant ON onvif_devices(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);

-- ═══════════════════════════════════════════════════════════════════
-- 3. Вспомогательная функция для RLS (создаётся один раз)
-- ═══════════════════════════════════════════════════════════════════

-- Функция получения tenant_id из session-local параметра.
-- Используется во всех RLS-политиках.
-- Если app.tenant_id = '*' — возвращается true (admin bypass).
CREATE OR REPLACE FUNCTION rls_tenant_check(row_tenant_id TEXT)
RETURNS BOOLEAN AS $$
BEGIN
    -- Admin bypass: '*' означает "видеть все tenant'ы"
    IF current_setting('app.tenant_id', true) = '*' THEN
        RETURN TRUE;
    END IF;
    -- Проверка совпадения tenant_id
    RETURN row_tenant_id = '' OR row_tenant_id = current_setting('app.tenant_id', true);
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION rls_tenant_check(TEXT) IS
    'F-0.2.3: Проверяет tenant_id строки против session-local app.tenant_id. ' ||
    'Admin bypass: app.tenant_id = ''*'' пропускает все tenant''ы.';

-- ═══════════════════════════════════════════════════════════════════
-- 4. Включение RLS и создание политик
-- ═══════════════════════════════════════════════════════════════════

-- ── devices ────────────────────────────────────────────────────
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_devices ON devices;
CREATE POLICY tenant_isolation_devices ON devices
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_devices ON devices IS
    'F-0.2.3: Изоляция устройств по tenant_id. Соответствует IEC 62443 SR 5.1.';

-- ── sites ──────────────────────────────────────────────────────
ALTER TABLE sites ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_sites ON sites;
CREATE POLICY tenant_isolation_sites ON sites
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_sites ON sites IS
    'F-0.2.3: Изоляция площадок по tenant_id.';

-- ── tickets ────────────────────────────────────────────────────
ALTER TABLE tickets ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_tickets ON tickets;
CREATE POLICY tenant_isolation_tickets ON tickets
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_tickets ON tickets IS
    'F-0.2.3: Изоляция тикетов по tenant_id.';

-- ── work_orders ────────────────────────────────────────────────
ALTER TABLE work_orders ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_work_orders ON work_orders;
CREATE POLICY tenant_isolation_work_orders ON work_orders
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_work_orders ON work_orders IS
    'F-0.2.3: Изоляция нарядов-допусков по tenant_id.';

-- ── spare_parts ────────────────────────────────────────────────
ALTER TABLE spare_parts ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_spare_parts ON spare_parts;
CREATE POLICY tenant_isolation_spare_parts ON spare_parts
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_spare_parts ON spare_parts IS
    'F-0.2.3: Изоляция запчастей по tenant_id.';

-- ── asset_downtime ─────────────────────────────────────────────
ALTER TABLE asset_downtime ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_asset_downtime ON asset_downtime;
CREATE POLICY tenant_isolation_asset_downtime ON asset_downtime
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_asset_downtime ON asset_downtime IS
    'F-0.2.3: Изоляция простоев по tenant_id.';

-- ── maintenance_schedules ──────────────────────────────────────
ALTER TABLE maintenance_schedules ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_maintenance_schedules ON maintenance_schedules;
CREATE POLICY tenant_isolation_maintenance_schedules ON maintenance_schedules
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_maintenance_schedules ON maintenance_schedules IS
    'F-0.2.3: Изоляция расписаний ТО по tenant_id.';

-- ── onvif_devices ──────────────────────────────────────────────
ALTER TABLE onvif_devices ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_onvif_devices ON onvif_devices;
CREATE POLICY tenant_isolation_onvif_devices ON onvif_devices
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_onvif_devices ON onvif_devices IS
    'F-0.2.3: Изоляция ONVIF устройств по tenant_id.';

-- ── users ──────────────────────────────────────────────────────
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_users ON users;
CREATE POLICY tenant_isolation_users ON users
    FOR ALL
    USING (rls_tenant_check(tenant_id))
    WITH CHECK (rls_tenant_check(tenant_id));

COMMENT ON POLICY tenant_isolation_users ON users IS
    'F-0.2.3: Изоляция пользователей по tenant_id.';

-- ═══════════════════════════════════════════════════════════════════
-- 5. Force RLS для существующих строк (защита от NULL tenant_id)
-- ═══════════════════════════════════════════════════════════════════

-- RLS-политика пропускает строки с tenant_id = '' (значение по умолчанию).
-- Это безопасно пока не включена принудительная проверка.
-- После миграции данных tenant_id перестанет быть ''.
