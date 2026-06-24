-- +migrate Up
-- AN-10.1.3: TCO (Total Cost of Ownership) per device — материализованное представление
--
-- Формула: TCO = Purchase + Labor + Parts + Downtime
--   - total_purchase_cost: стоимость запчастей, использованных для устройства
--     (извлекается из work_orders.parts_used JSONB)
--   - total_labor_cost: сумма labour cost из work_orders
--   - total_parts_cost: сумма parts cost из work_orders
--   - total_downtime_cost: сумма стоимости простоев из asset_downtime
--   - tco: итоговая стоимость владения
--
-- Compliance:
--   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
--   - IEC 62443 SR 7.1 (Resource availability — asset TCO)
--   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
--   - СТБ 34.101.27 (Защита информации — учёт стоимости активов)
--   - OWASP ASVS V5.1 (Parameterized queries — через DB слой)

CREATE MATERIALIZED VIEW IF NOT EXISTS mv_tco_per_device AS
SELECT
    d.device_id,
    d.name as device_name,
    d.vendor_type,
    d.device_type,
    d.manufacturer,
    -- Purchase cost (из spare_parts, использованных для этого device)
    -- Извлекаем total_price из JSONB массива parts_used
    COALESCE((
        SELECT SUM((p.value->>'total_price')::numeric)
        FROM work_orders wo,
        LATERAL jsonb_array_elements(wo.parts_used) p
        WHERE wo.device_id = d.device_id
    ), 0) as total_purchase_cost,
    -- Labor cost
    COALESCE(SUM(wo.total_labor_cost), 0) as total_labor_cost,
    -- Parts cost
    COALESCE(SUM(wo.total_parts_cost), 0) as total_parts_cost,
    -- Downtime cost
    COALESCE(SUM(dt.downtime_cost), 0) as total_downtime_cost,
    -- TCO = Purchase + Labor + Parts + Downtime
    COALESCE((
        SELECT SUM((p.value->>'total_price')::numeric)
        FROM work_orders wo,
        LATERAL jsonb_array_elements(wo.parts_used) p
        WHERE wo.device_id = d.device_id
    ), 0) + COALESCE(SUM(wo.total_labor_cost), 0) + COALESCE(SUM(wo.total_parts_cost), 0) + COALESCE(SUM(dt.downtime_cost), 0) as tco,
    COUNT(DISTINCT wo.id) as total_work_orders,
    COUNT(DISTINCT dt.id) as total_downtime_events
FROM devices d
LEFT JOIN work_orders wo ON d.device_id = wo.device_id
LEFT JOIN asset_downtime dt ON d.device_id = dt.device_id
GROUP BY d.device_id, d.name, d.vendor_type, d.device_type, d.manufacturer;

CREATE UNIQUE INDEX IF NOT EXISTS idx_mv_tco_device ON mv_tco_per_device(device_id);

COMMENT ON MATERIALIZED VIEW mv_tco_per_device IS
    'AN-10.1.3: TCO (Total Cost of Ownership) per device. '
    'Содержит агрегированные данные по стоимости владения устройством: '
    'purchase (запчасти), labor, parts, downtime, tco. '
    'Соответствует ISO 27001 A.12.6.1, IEC 62443 SR 7.1.';

COMMENT ON COLUMN mv_tco_per_device.total_purchase_cost IS
    'Стоимость запчастей (из work_orders.parts_used JSONB)';
COMMENT ON COLUMN mv_tco_per_device.total_labor_cost IS
    'Сумма labour cost из work_orders';
COMMENT ON COLUMN mv_tco_per_device.total_parts_cost IS
    'Сумма parts cost из work_orders';
COMMENT ON COLUMN mv_tco_per_device.total_downtime_cost IS
    'Сумма стоимости простоев из asset_downtime.downtime_cost';
COMMENT ON COLUMN mv_tco_per_device.tco IS
    'Total Cost of Ownership = Purchase + Labor + Parts + Downtime';
