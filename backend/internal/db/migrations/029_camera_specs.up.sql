-- +migrate Up
-- Migration 029: Camera Specs Database (P0-9)
--
-- Хранит технические характеристики моделей камер для:
--   - Автозаполнения в Device Wizard (P1-7)
--   - Расчёта пропускной способности и хранилища
--   - Проверки совместимости (PoE, протоколы, ONVIF)
--
-- Compliance:
--   - ISO 27001 A.8.1.2: Asset inventory — каталог оборудования
--   - ISO 27019 PCC.A.8: Asset management для ICS
--   - IEC 62443 SR 3.1: Identification of IACS devices

-- ═══════════════════════════════════════════════════════════════════
-- 1. Таблица camera_specs — каталог моделей камер
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS camera_specs (
    id                  SERIAL PRIMARY KEY,
    brand               TEXT NOT NULL,
    model               TEXT NOT NULL,
    type                TEXT CHECK (type IN ('bullet', 'dome', 'ptz', 'fisheye', 'box', 'thermal', 'multi-sensor')),
    resolution          TEXT,               -- 2MP, 4MP, 8MP, 12MP
    max_fps             INTEGER,
    lens_mm             TEXT,               -- 2.8mm, 2.8-12mm
    infrared            BOOLEAN DEFAULT true,
    poe                 BOOLEAN DEFAULT true,
    poe_class           TEXT CHECK (poe_class IN ('802.3af', '802.3at', '802.3bt')),
    power_watts         NUMERIC(5,1),
    storage_days_estimate INTEGER,
    bandwidth_mbps      NUMERIC(5,1),
    protocols           TEXT[],             -- {'ONVIF','RTSP','Hikvision-CGI','Dahua-API'}
    onvif_profile       TEXT CHECK (onvif_profile IN ('S', 'T', 'G', 'Q')),
    audio_support       BOOLEAN DEFAULT false,
    outdoor_rating      TEXT,               -- IP67, IK10, IP66
    weight_grams        INTEGER,
    dimensions          TEXT,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_camera_specs_brand ON camera_specs(brand);
CREATE INDEX IF NOT EXISTS idx_camera_specs_model ON camera_specs(model);
CREATE UNIQUE INDEX IF NOT EXISTS idx_camera_specs_brand_model ON camera_specs(brand, model);

COMMENT ON TABLE camera_specs IS
    'P0-9: Camera Specs Database — каталог характеристик моделей камер. '
    'Соответствует ISO 27001 A.8.1.2, ISO 27019 PCC.A.8, IEC 62443 SR 3.1';

COMMENT ON COLUMN camera_specs.type IS
    'Тип камеры: bullet, dome, ptz, fisheye, box, thermal, multi-sensor';
COMMENT ON COLUMN camera_specs.poe_class IS
    'PoE класс: 802.3af (15.4W), 802.3at (30W), 802.3bt (60-100W)';
COMMENT ON COLUMN camera_specs.onvif_profile IS
    'ONVIF профиль: S (стриминг), T (аналитика), G (запись), Q (квант)';
