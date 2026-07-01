// Package ml — Machine Learning prediction service.
//
// PredictionService публикует задачи предсказания в NATS JetStream очередь
// (P0-CR-04) вместо запуска Python subprocess. Python predict_worker.py
// потребляет задачи асинхронно, сохраняет результаты в БД и публикует
// события в NATS топик ml.prediction.{device_id}.
//
// A/B testing: если включён, 50% устройств получают variant B.
// В NATS payload добавляется model_variant для анализа.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Queue-based processing with retries)
//   - ISO 27001 A.12.4.1 (Event logging — predictions as system events)
//   - IEC 62443 SR 3.3 (Security monitoring — predictive analytics)
//   - СТБ 34.101.27 п. 7.3 (Анализ защищённости — прогнозирование отказов)
package ml

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/nats-io/nats.go"
)

// ── Prediction Result ───────────────────────────────────────────────

// PredictionResult — одно предсказание (парсинг JSON строки из predict.py).
type PredictionResult struct {
	DeviceID             string              `json:"device_id"`
	FailureProbability   float64             `json:"failure_probability"`
	ConfidenceScore      float64             `json:"confidence_score"`
	ModelVersion         string              `json:"model_version"`
	ModelVariant         string              `json:"model_variant"`
	PredictionDate       string              `json:"prediction_date"`
	PredictionWindowDays int                 `json:"prediction_window_days"`
	IsActionable         bool                `json:"is_actionable"`
	IsAnomaly            bool                `json:"is_anomaly"`
	CalibrationBin       int                 `json:"calibration_bin"`
	TopFeatures          []FeatureImportance `json:"top_features"`
	FeaturesSnapshot     map[string]float64  `json:"features_snapshot"`
	TraceID              string              `json:"trace_id"`
}

// FeatureImportance — важность признака.
type FeatureImportance struct {
	Feature    string  `json:"feature"`
	Importance float64 `json:"importance"`
	Value      float64 `json:"value"`
}

// MetaInfo — мета-информация из последней строки JSONL.
type MetaInfo struct {
	Meta struct {
		Total          int     `json:"total"`
		Actionable     int     `json:"actionable"`
		AvgProbability float64 `json:"avg_probability"`
		Status         string  `json:"status"`
		Timestamp      string  `json:"timestamp"`
	} `json:"_meta"`
}

// ── NATS Event ──────────────────────────────────────────────────────

// PredictionEvent — событие для публикации в NATS.
// Соответствует формату ml.prediction.{device_id}.
type PredictionEvent struct {
	DeviceID           string              `json:"device_id"`
	FailureProbability float64             `json:"failure_probability"`
	ConfidenceScore    float64             `json:"confidence_score"`
	ModelVersion       string              `json:"model_version"`
	ModelVariant       string              `json:"model_variant"`
	PredictionDate     string              `json:"prediction_date"`
	PredictionWindow   int                 `json:"prediction_window_days"`
	IsActionable       bool                `json:"is_actionable"`
	IsAnomaly          bool                `json:"is_anomaly"`
	TopFeatures        []FeatureImportance `json:"top_features,omitempty"`
	TraceID            string              `json:"trace_id"`
}

// ── PredictionService ───────────────────────────────────────────────

// PredictionService управляет публикацией задач предсказания
// через NATS JetStream очередь. Не запускает subprocess напрямую.
type PredictionService struct {
	cfg    MLConfig
	logger *slog.Logger
	nc     *nats.Conn
	js     nats.JetStreamContext
	queue  *PredictionQueue
	mu     sync.Mutex
}

// NewPredictionService создаёт новый PredictionService.
func NewPredictionService(cfg MLConfig, nc *nats.Conn, logger *slog.Logger) (*PredictionService, error) {
	if logger == nil {
		logger = slog.Default()
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}

	// Создаём очередь для публикации задач
	queue, err := NewPredictionQueue(nc, cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("prediction queue: %w", err)
	}

	return &PredictionService{
		cfg:    cfg,
		logger: logger.With("service", "ml_prediction"),
		nc:     nc,
		js:     js,
		queue:  queue,
	}, nil
}

// ── Public API ──────────────────────────────────────────────────────

// RunBatch публикует задачи предсказания для всех устройств в очередь.
// Python predict_worker.py обработает их асинхронно.
//
// В отличие от старой subprocess-архитектуры:
//   - Нет блокировки на stdout/stderr pipes
//   - Backpressure через MaxAckPending (MaxActiveWorkers)
//   - Graceful shutdown через SIGTERM + drain consumer
//   - Per-device processing (нет OOM от загрузки всех устройств)
func (s *PredictionService) RunBatch(ctx context.Context, deviceIDs []string) (int, error) {
	if len(deviceIDs) == 0 {
		s.logger.WarnContext(ctx, "no devices to predict")
		return 0, nil
	}

	s.logger.InfoContext(ctx, "publishing batch prediction tasks",
		"devices", len(deviceIDs),
		"variant", s.cfg.ModelVariant,
	)

	// Создаём задачи для каждого устройства
	tasks := make([]PredictionTask, 0, len(deviceIDs))
	for _, deviceID := range deviceIDs {
		traceID := newTraceID()
		variant := s.cfg.ModelVariant

		// A/B testing: распределяем устройства по variant'ам
		if s.cfg.ABTestingEnabled {
			hash := hashDeviceID(deviceID)
			if float64(hash)/float64(math.MaxUint32) < s.cfg.ABTestingRatio {
				variant = "B"
			} else {
				variant = "A"
			}
		}

		tasks = append(tasks, PredictionTask{
			DeviceID:     deviceID,
			ModelVariant: variant,
			TraceID:      traceID,
		})
	}

	// Публикуем задачи в очередь
	published, err := s.queue.PublishTasks(ctx, tasks)
	if err != nil {
		return published, fmt.Errorf("publish tasks: %w", err)
	}

	s.logger.InfoContext(ctx, "batch prediction tasks published",
		"total", len(tasks),
		"published", published,
	)

	return published, nil
}

// ── Internal: NATS publish result ───────────────────────────────────

// PublishResult публикует результат предсказания в NATS.
// Вызывается из Python worker через NATS или из Go подписчика.
func (s *PredictionService) PublishResult(ctx context.Context, r PredictionResult) error {
	event := PredictionEvent{
		DeviceID:           r.DeviceID,
		FailureProbability: r.FailureProbability,
		ConfidenceScore:    r.ConfidenceScore,
		ModelVersion:       r.ModelVersion,
		ModelVariant:       r.ModelVariant,
		PredictionDate:     r.PredictionDate,
		PredictionWindow:   r.PredictionWindowDays,
		IsActionable:       r.IsActionable,
		IsAnomaly:          r.IsAnomaly,
		TopFeatures:        r.TopFeatures,
		TraceID:            r.TraceID,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	subject := fmt.Sprintf("%s.%s", s.cfg.NATSTopicPrefix, r.DeviceID)

	// Публикуем через NATS (без JetStream для быстрой доставки)
	if err := s.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("nats publish %s: %w", subject, err)
	}

	// Публикуем также в JetStream для durable хранения
	if _, err := s.js.Publish(subject, data); err != nil {
		s.logger.WarnContext(ctx, "jetstream publish failed",
			"subject", subject,
			"error", err,
		)
	}

	s.logger.DebugContext(ctx, "prediction result published",
		"subject", subject,
		"device_id", r.DeviceID,
		"probability", r.FailureProbability,
		"confidence", r.ConfidenceScore,
		"variant", r.ModelVariant,
	)

	return nil
}

// Queue возвращает PredictionQueue для прямого доступа (Consume).
func (s *PredictionService) Queue() *PredictionQueue {
	return s.queue
}

// ── Utility ─────────────────────────────────────────────────────────

func newTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// hashDeviceID — простой детерминированный хеш device_id.
func hashDeviceID(id string) uint32 {
	var h uint32
	for _, c := range id {
		h = h*31 + uint32(c)
	}
	return h
}

// Close освобождает ресурсы сервиса.
func (s *PredictionService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue.Close()
	s.logger.Info("prediction service closed")
}
