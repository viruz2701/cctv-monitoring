-- +migrate Up
-- CCTV-2.2.1: ONVIF Profile S/T Devices
--
-- Таблица для хранения ONVIF устройств, найденных через WS-Discovery
-- или зарегистрированных вручную.
--
-- Compliance:
--   - IEC 62443-3-3 SL-3: зона 3 (Application) — управление ONVIF устройствами
--   - ISO 27001 A.12.4: audit trail (изменения через audit_log)
--   - ISO 27019 PCC.A.13: ICS asset inventory
--   - Приказ ОАЦ №66 п.7.18: уникальная идентификация устройств
--   - СТБ 34.101.27 п.6.3: управление активами

-- ── ONVIF Devices ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS onvif_devices (
    device_id       TEXT NOT NULL PRIMARY KEY,
    manufacturer    TEXT NOT NULL DEFAULT '',
    model           TEXT NOT NULL DEFAULT '',
    firmware        TEXT NOT NULL DEFAULT '',
    hardware_id     TEXT NOT NULL DEFAULT '',
    serial_number   TEXT NOT NULL DEFAULT '',
    capabilities    JSONB NOT NULL DEFAULT '{}'::jsonb,
    xaddrs          TEXT[] NOT NULL DEFAULT '{}',
    scopes          TEXT[] NOT NULL DEFAULT '{}',
    discovery_date  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    connect_mode    TEXT NOT NULL DEFAULT 'direct'
                    CHECK (connect_mode IN ('direct', 'p2p', 'edge_agent')),
    p2p_session_id  TEXT DEFAULT '',
    active          BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_onvif_devices_device_id
    ON onvif_devices (device_id);

CREATE INDEX IF NOT EXISTS idx_onvif_devices_discovery_date
    ON onvif_devices (discovery_date DESC);

CREATE INDEX IF NOT EXISTS idx_onvif_devices_last_seen
    ON onvif_devices (last_seen DESC);

CREATE INDEX IF NOT EXISTS idx_onvif_devices_manufacturer
    ON onvif_devices (manufacturer);

CREATE INDEX IF NOT EXISTS idx_onvif_devices_connect_mode
    ON onvif_devices (connect_mode);

CREATE INDEX IF NOT EXISTS idx_onvif_devices_active
    ON onvif_devices (active)
    WHERE active = true;

-- Комментарии
COMMENT ON TABLE onvif_devices IS
    'CCTV-2.2.1: ONVIF Profile S/T устройства. Содержит информацию об ONVIF-совместимых '
    'камерах, найденных через WS-Discovery или зарегистрированных вручную.';

COMMENT ON COLUMN onvif_devices.device_id IS
    'Уникальный идентификатор устройства (XAddr or serial) — Приказ ОАЦ №66 п.7.18';
COMMENT ON COLUMN onvif_devices.capabilities IS
    'ONVIF capabilities в JSON: {media:bool, ptz:bool, events:bool, analytics:bool}';
COMMENT ON COLUMN onvif_devices.xaddrs IS
    'ONVIF XAddrs (Service addresses) из WS-Discovery ProbeMatch';
COMMENT ON COLUMN onvif_devices.scopes IS
    'ONVIF scopes из WS-Discovery ProbeMatch';
COMMENT ON COLUMN onvif_devices.connect_mode IS
    'Режим подключения: direct (прямое TCP/HTTP), p2p (через P2P gateway), edge_agent';
COMMENT ON COLUMN onvif_devices.p2p_session_id IS
    'ID сессии в P2P gateway для NAT traversal';
COMMENT ON COLUMN onvif_devices.active IS
    'Флаг активности. Если false — устройство исключено из мониторинга.';

-- ── Trigger: auto-update updated_at ────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_onvif_devices_updated_at()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_onvif_devices_updated_at ON onvif_devices;
CREATE TRIGGER trg_onvif_devices_updated_at
    BEFORE UPDATE ON onvif_devices
    FOR EACH ROW
    EXECUTE FUNCTION update_onvif_devices_updated_at();
