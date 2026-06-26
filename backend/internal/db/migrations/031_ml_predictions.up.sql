-- +migrate Up
-- Migration 031: ML Predictions table
--
-- Хранит результаты ML-предсказаний отказов устройств.
-- Позволяет A/B тестирование через model_variant (A/B).
-- Hypertable для TimescaleDB — автоматическое партиционирование по prediction_date.
--
-- Compliance:
--   ISO 27001 A.12.4.1 (Event logging — predictions as system events)
--   IEC 62443 SR 3.3 (Security monitoring — predictive analytics)
--   СТБ 34.101.27 п. 7.3 (Анализ защищённости — прогнозирование отказов)
--   Приказ ОАЦ №66 п. 7.18.3 (Audit trail для edge devices)

CREATE TABLE IF NOT EXISTS predictions (
    id                      BIGSERIAL,
    device_id               TEXT NOT NULL REFERENCES devices(device_id) ON DELETE CASCADE,
    prediction_date         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    failure_probability     DOUBLE PRECISION NOT NULL
                            CHECK (failure_probability >= 0 AND failure_probability <= 1),
    confidence_score        DOUBLE PRECISION NOT NULL DEFAULT 0
                            CHECK (confidence_score >= 0 AND confidence_score <= 1),
    model_version           TEXT NOT NULL DEFAULT 'xgboost_v1',
    model_variant           TEXT NOT NULL DEFAULT 'A'
                            CHECK (model_variant IN ('A', 'B')),
    features_snapshot       JSONB,
    top_features            JSONB,           -- топ-3 признака, повлиявшие на прогноз
    explanation             TEXT,            -- LLM-generated explanation (optional)
    prediction_window_days  INT NOT NULL DEFAULT 30,
    is_actionable           BOOLEAN DEFAULT false,
    is_anomaly              BOOLEAN DEFAULT false,  -- выброс / аномалия
    calibration_bin         INT DEFAULT 0,          -- бин калибровки [0-10]
    source                  TEXT DEFAULT 'batch',   -- batch | on_demand | edge
    trace_id                TEXT DEFAULT '',         -- W3C Trace Context
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- TimescaleDB hypertable для эффективного хранения по времени
SELECT create_hypertable('predictions', 'prediction_date', if_not_exists => TRUE);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_predictions_device_date
    ON predictions(device_id, prediction_date DESC);
CREATE INDEX IF NOT EXISTS idx_predictions_model_version
    ON predictions(model_version, model_variant);
CREATE INDEX IF NOT EXISTS idx_predictions_probability
    ON predictions(failure_probability DESC)
    WHERE failure_probability > 0.5;
CREATE INDEX IF NOT EXISTS idx_predictions_actionable
    ON predictions(device_id, prediction_date DESC)
    WHERE is_actionable = true;

-- Комментарии
COMMENT ON TABLE predictions IS
    'P2-1.1: ML предсказания отказов устройств. XGBoost, A/B тестирование, confidence score.';
COMMENT ON COLUMN predictions.failure_probability IS
    'Вероятность отказа в ближайшие prediction_window_days дней [0..1]';
COMMENT ON COLUMN predictions.confidence_score IS
    'Уверенность модели в прогнозе на основе калибровки [0..1]';
COMMENT ON COLUMN predictions.model_variant IS
    'Вариант модели для A/B тестирования: A (control) | B (treatment)';
COMMENT ON COLUMN predictions.features_snapshot IS
    'Значения признаков на момент предсказания (JSON)';
COMMENT ON COLUMN predictions.top_features IS
    'Топ-3 признака с наибольшим влиянием (feature importance)';
COMMENT ON COLUMN predictions.is_actionable IS
    'True если failure_probability > threshold (0.5) — требует внимания';
COMMENT ON COLUMN predictions.calibration_bin IS
    'Бин калибровки [0-10] для анализа reliability диаграммы';
COMMENT ON COLUMN predictions.source IS
    'Источник: batch (CRON) | on_demand (API) | edge (агент на камере)';

-- Функция: получить последнее предсказание для устройства
CREATE OR REPLACE FUNCTION get_latest_prediction(p_device_id TEXT)
RETURNS TABLE(
    device_id TEXT,
    failure_probability DOUBLE PRECISION,
    confidence_score DOUBLE PRECISION,
    model_version TEXT,
    model_variant TEXT,
    prediction_date TIMESTAMPTZ,
    is_actionable BOOLEAN
) LANGUAGE SQL STABLE AS $$
    SELECT device_id, failure_probability, confidence_score,
           model_version, model_variant, prediction_date, is_actionable
    FROM predictions
    WHERE device_id = p_device_id
    ORDER BY prediction_date DESC
    LIMIT 1;
$$;

COMMENT ON FUNCTION get_latest_prediction IS
    'P2-1.1: Возвращает последнее предсказание для указанного устройства.';

-- Функция: метрики качества модели по variant'ам (A/B comparison)
CREATE OR REPLACE FUNCTION get_model_ab_metrics(
    p_model_version TEXT,
    p_days INT DEFAULT 30
)
RETURNS TABLE(
    model_variant TEXT,
    total_predictions BIGINT,
    avg_probability DOUBLE PRECISION,
    avg_confidence DOUBLE PRECISION,
    actionable_ratio DOUBLE PRECISION,
    anomaly_ratio DOUBLE PRECISION
) LANGUAGE SQL STABLE AS $$
    SELECT
        model_variant,
        COUNT(*)::bigint,
        AVG(failure_probability)::double precision,
        AVG(confidence_score)::double precision,
        COUNT(*) FILTER (WHERE is_actionable)::float8 / NULLIF(COUNT(*), 0),
        COUNT(*) FILTER (WHERE is_anomaly)::float8 / NULLIF(COUNT(*), 0)
    FROM predictions
    WHERE model_version = p_model_version
      AND prediction_date >= NOW() - (p_days || ' days')::interval
    GROUP BY model_variant
    ORDER BY model_variant;
$$;

COMMENT ON FUNCTION get_model_ab_metrics IS
    'P2-1.1: A/B метрики по variant'ам для сравнения качества моделей.';
