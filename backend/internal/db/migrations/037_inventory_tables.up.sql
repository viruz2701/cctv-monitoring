-- +migrate Up
-- P2-INV.1-4: Inventory Management — таблицы для управления запчастями и перезаказом
--
-- Создаёт:
--   - auto_order_config: конфигурация авто-заказа
--   - vendor_scorecards: скор-карты поставщиков
--   - reorder_rules: правила перезаказа
--   - lifecycle_costs: расчёт стоимости владения
--
-- Compliance:
--   - IEC 62443-3-3 SL-3 (Zone 3 — Application integrity)
--   - ISO 27001 A.12.4.1 (Event logging — audit trail)
--   - ISO 27001 A.12.6.1 (Capacity management)
--   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
--   - ISO 27001 A.15 (Supplier relationships)
--   - СТБ 34.101.27 (Защита информации — учёт активов)
--   - OWASP ASVS V5.1 (Parameterized queries — через DB слой)

-- ═══════════════════════════════════════════════════════════════════════
-- P2-INV.1: Auto Parts — конфигурация авто-заказа
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS auto_order_config (
    id                   TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    part_id              TEXT NOT NULL REFERENCES spare_parts(id) ON DELETE CASCADE,
    low_stock_threshold  INT NOT NULL DEFAULT 50,    -- % от min_stock
    critical_threshold   INT NOT NULL DEFAULT 25,    -- % от min_stock
    auto_order_threshold INT NOT NULL DEFAULT 40,    -- % от min_stock
    default_order_qty    INT NOT NULL DEFAULT 10,
    max_order_qty        INT NOT NULL DEFAULT 100,
    prefer_preferred_vendor BOOLEAN NOT NULL DEFAULT TRUE,
    currency             TEXT NOT NULL DEFAULT 'USD',
    is_active            BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_auto_order_config_part
    ON auto_order_config(part_id);

CREATE INDEX IF NOT EXISTS idx_auto_order_config_active
    ON auto_order_config(is_active)
    WHERE is_active = TRUE;

COMMENT ON TABLE auto_order_config IS
    'P2-INV.1: Конфигурация автоматического заказа запчастей. '
    'Содержит пороги срабатывания и параметры заказа для каждой запчасти.';

-- ═══════════════════════════════════════════════════════════════════════
-- P2-INV.2: Vendor Scorecards — скор-карты поставщиков
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS vendor_scorecards (
    id                      TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    vendor_id               TEXT NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,

    -- Метрики (0.0 – 100.0)
    delivery_score          NUMERIC(5,1) NOT NULL DEFAULT 0,
    quality_score           NUMERIC(5,1) NOT NULL DEFAULT 0,
    price_score             NUMERIC(5,1) NOT NULL DEFAULT 0,
    reliability_score       NUMERIC(5,1) NOT NULL DEFAULT 0,

    -- Сырые данные для расчёта
    total_orders            INT NOT NULL DEFAULT 0,
    completed_orders        INT NOT NULL DEFAULT 0,
    avg_lead_time_days      NUMERIC(6,1) NOT NULL DEFAULT 0,
    price_competitiveness   NUMERIC(3,2) NOT NULL DEFAULT 0.50,
    defect_rate             NUMERIC(3,2) NOT NULL DEFAULT 0,
    on_time_delivery_rate   NUMERIC(3,2) NOT NULL DEFAULT 0,
    contract_compliance_rate NUMERIC(3,2) NOT NULL DEFAULT 1.0,

    last_order_date         TIMESTAMPTZ,
    notes                   TEXT NOT NULL DEFAULT '',
    is_active               BOOLEAN NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT unique_vendor_scorecard UNIQUE (vendor_id)
);

CREATE INDEX IF NOT EXISTS idx_vendor_scorecards_score
    ON vendor_scorecards((delivery_score + quality_score + price_score + reliability_score) / 4);

COMMENT ON TABLE vendor_scorecards IS
    'P2-INV.2: Скор-карты поставщиков. '
    'Содержит метрики: delivery, quality, price, reliability. '
    'Соответствует ISO 27001 A.15 (Supplier relationships).';

-- ═══════════════════════════════════════════════════════════════════════
-- P2-INV.3: Lifecycle Costs — стоимость владения
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS lifecycle_costs (
    id                    TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    part_id               TEXT NOT NULL REFERENCES spare_parts(id) ON DELETE CASCADE,
    currency              TEXT NOT NULL DEFAULT 'USD',

    -- Компоненты TCO
    purchase_cost         NUMERIC(12,2) NOT NULL DEFAULT 0,
    maintenance_cost      NUMERIC(12,2) NOT NULL DEFAULT 0,
    energy_cost           NUMERIC(12,2) NOT NULL DEFAULT 0,
    disposal_cost         NUMERIC(12,2) NOT NULL DEFAULT 0,
    installation_cost     NUMERIC(12,2) NOT NULL DEFAULT 0,
    training_cost         NUMERIC(12,2) NOT NULL DEFAULT 0,
    transport_cost        NUMERIC(12,2) NOT NULL DEFAULT 0,

    -- Временные параметры
    expected_lifespan_days INT NOT NULL DEFAULT 3650,
    operational_days      INT NOT NULL DEFAULT 0,
    purchase_date         TIMESTAMPTZ,
    last_maintenance_date TIMESTAMPTZ,

    -- Вычисляемые поля
    tco                   NUMERIC(12,2) GENERATED ALWAYS AS (
        purchase_cost + maintenance_cost + energy_cost + disposal_cost +
        installation_cost + training_cost + transport_cost
    ) STORED,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_costs_part
    ON lifecycle_costs(part_id);

CREATE INDEX IF NOT EXISTS idx_lifecycle_costs_tco
    ON lifecycle_costs(tco DESC);

COMMENT ON TABLE lifecycle_costs IS
    'P2-INV.3: Стоимость владения запчастями/активами (TCO). '
    'TCO = Purchase + Maintenance + Energy + Disposal + Installation + Training + Transport. '
    'Соответствует ISO 27001 A.12.6.1, IEC 62443 SR 7.1.';

-- ═══════════════════════════════════════════════════════════════════════
-- P2-INV.4: Reorder Rules — правила перезаказа
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS reorder_rules (
    id                    TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    part_id               TEXT NOT NULL REFERENCES spare_parts(id) ON DELETE CASCADE,

    min_stock             INT NOT NULL DEFAULT 0,
    max_stock             INT NOT NULL DEFAULT 0,
    reorder_qty           INT NOT NULL DEFAULT 0,       -- 0 = auto-calculate
    lead_time_days        INT NOT NULL DEFAULT 0,
    safety_stock_days     INT NOT NULL DEFAULT 7,
    daily_consumption     NUMERIC(10,2) NOT NULL DEFAULT 0,
    seasonal_multiplier   NUMERIC(3,2) NOT NULL DEFAULT 1.0,

    auto_approve          BOOLEAN NOT NULL DEFAULT FALSE,
    policy                TEXT NOT NULL DEFAULT 'min_max' CHECK (policy IN ('min_max', 'fixed', 'kanban', 'demand')),
    review_interval_days  INT NOT NULL DEFAULT 1,
    kanban_size           INT NOT NULL DEFAULT 0,

    preferred_vendor_id   TEXT REFERENCES vendors(id) ON DELETE SET NULL,
    is_active             BOOLEAN NOT NULL DEFAULT TRUE,
    notes                 TEXT NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reorder_rules_part
    ON reorder_rules(part_id);

CREATE INDEX IF NOT EXISTS idx_reorder_rules_active
    ON reorder_rules(is_active)
    WHERE is_active = TRUE;

COMMENT ON TABLE reorder_rules IS
    'P2-INV.4: Правила автоматического перезаказа запчастей. '
    'Поддерживает политики: min_max, fixed, kanban, demand. '
    'Учитывает сезонные коэффициенты и страховой запас.';
