-- +migrate Down
-- P2-BI: Drop analytics query templates table
DROP TRIGGER IF EXISTS trg_analytics_templates_updated_at ON analytics_query_templates;
DROP FUNCTION IF EXISTS trigger_set_analytics_template_updated_at();
DROP TABLE IF EXISTS analytics_query_templates;
