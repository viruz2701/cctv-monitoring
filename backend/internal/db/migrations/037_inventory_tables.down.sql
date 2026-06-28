-- +migrate Down
-- P2-INV.1-4: Откат — удаление таблиц управления инвентарём

DROP INDEX IF EXISTS idx_reorder_rules_active;
DROP INDEX IF EXISTS idx_reorder_rules_part;
DROP TABLE IF EXISTS reorder_rules;

DROP INDEX IF EXISTS idx_lifecycle_costs_tco;
DROP INDEX IF EXISTS idx_lifecycle_costs_part;
DROP TABLE IF EXISTS lifecycle_costs;

DROP INDEX IF EXISTS idx_vendor_scorecards_score;
DROP TABLE IF EXISTS vendor_scorecards;

DROP INDEX IF EXISTS idx_auto_order_config_active;
DROP INDEX IF EXISTS idx_auto_order_config_part;
DROP TABLE IF EXISTS auto_order_config;
