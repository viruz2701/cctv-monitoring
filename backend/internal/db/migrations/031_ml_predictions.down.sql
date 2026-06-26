-- +migrate Down
-- Migration 031: Rollback ML Predictions

DROP FUNCTION IF EXISTS get_model_ab_metrics(TEXT, INT);
DROP FUNCTION IF EXISTS get_latest_prediction(TEXT);
DROP TABLE IF EXISTS predictions;
