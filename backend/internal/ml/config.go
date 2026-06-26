// Package ml — Machine Learning integration for device failure prediction.
//
// P2-1.1: XGBoost модель для предсказания отказов устройств.
// Взаимодействует с Python predict.py через subprocess (JSONL stdout).
// Публикует предсказания в NATS топик ml.prediction.{device_id}.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — predictions as system events)
//   - IEC 62443 SR 3.3 (Security monitoring — predictive analytics)
//   - СТБ 34.101.27 п. 7.3 (Анализ защищённости — прогнозирование отказов)
package ml

// MLConfig — конфигурация ML сервиса.
// Маппится на секцию analytics/config.yaml → backend config.yaml.
type MLConfig struct {
	// PythonPath — путь к python3 интерпретатору.
	PythonPath string `mapstructure:"python_path"`

	// ScriptPath — путь к predict.py.
	ScriptPath string `mapstructure:"script_path"`

	// TrainScriptPath — путь к train.py.
	TrainScriptPath string `mapstructure:"train_script_path"`

	// ConfigPath — путь к analytics/config.yaml.
	ConfigPath string `mapstructure:"config_path"`

	// ModelVariant — активный вариант модели (A/B).
	ModelVariant string `mapstructure:"model_variant"`

	// PredictionInterval — интервал batch-предсказаний.
	PredictionInterval string `mapstructure:"prediction_interval"`

	// ProbabilityThreshold — порог is_actionable.
	ProbabilityThreshold float64 `mapstructure:"probability_threshold"`

	// MinConfidenceThreshold — минимальный confidence для публикации.
	MinConfidenceThreshold float64 `mapstructure:"min_confidence_threshold"`

	// ABTestingEnabled — включить A/B тестирование.
	ABTestingEnabled bool `mapstructure:"ab_testing_enabled"`

	// ABTestingRatio — доля устройств для variant B (0.0–1.0).
	ABTestingRatio float64 `mapstructure:"ab_testing_ratio"`

	// NATSURL — адрес NATS сервера.
	NATSURL string `mapstructure:"nats_url"`

	// NATSCreds — путь к NATS credentials файлу.
	NATSCreds string `mapstructure:"nats_creds"`

	// NATSTopicPrefix — префикс NATS топика (ml.prediction).
	NATSTopicPrefix string `mapstructure:"nats_topic_prefix"`
}

// DefaultMLConfig возвращает конфигурацию по умолчанию.
func DefaultMLConfig() MLConfig {
	return MLConfig{
		PythonPath:             "python3",
		ScriptPath:             "analytics/predict.py",
		TrainScriptPath:        "analytics/train.py",
		ConfigPath:             "analytics/config.yaml",
		ModelVariant:           "A",
		PredictionInterval:     "60m",
		ProbabilityThreshold:   0.5,
		MinConfidenceThreshold: 0.3,
		ABTestingEnabled:       true,
		ABTestingRatio:         0.5,
		NATSURL:                "nats://localhost:4222",
		NATSTopicPrefix:        "ml.prediction",
	}
}
