-- +migrate Down
-- Откат P2-3.3: Webhook Delivery Logs
DROP TRIGGER IF EXISTS trg_webhook_delivery_logs_updated_at ON webhook_delivery_logs;
DROP TRIGGER IF EXISTS trg_webhook_endpoints_updated_at ON webhook_endpoints;
DROP FUNCTION IF EXISTS update_webhook_updated_at();
DROP TABLE IF EXISTS webhook_delivery_logs;
DROP TABLE IF EXISTS webhook_endpoints;
