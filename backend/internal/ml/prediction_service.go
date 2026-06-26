// Package ml — Machine Learning prediction service.
//
// PredictionService запускает Python predict.py через subprocess,
// парсит JSONL вывод, публикует предсказания в NATS.
//
// A/B testing: если включён, 50% устройств получают variant B.
// В NATS payload добавляется model_variant для анализа.
package ml

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os/exec"
	"strings"
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

// PredictionService запускает batch-предсказания и публикует результаты в NATS.
type PredictionService struct {
	cfg     MLConfig
	logger  *slog.Logger
	nc      *nats.Conn
	js      nats.JetStreamContext
	mu      sync.Mutex
	running bool
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

	return &PredictionService{
		cfg:    cfg,
		logger: logger.With("service", "ml_prediction"),
		nc:     nc,
		js:     js,
	}, nil
}

// ── Public API ──────────────────────────────────────────────────────

// RunBatch запускает batch предсказание для всех устройств.
// Вызывает python3 predict.py, парсит JSONL, публикует в NATS.
func (s *PredictionService) RunBatch(ctx context.Context) (int, error) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return 0, fmt.Errorf("prediction already running")
	}
	s.running = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	s.logger.InfoContext(ctx, "starting batch prediction",
		"variant", s.cfg.ModelVariant,
		"script", s.cfg.ScriptPath,
	)

	// ── 1. Запускаем Python predict.py ──
	traceID := newTraceID()
	results, meta, err := s.runPython(ctx, traceID)
	if err != nil {
		return 0, fmt.Errorf("python prediction failed: %w", err)
	}

	if len(results) == 0 {
		s.logger.WarnContext(ctx, "no predictions generated")
		return 0, nil
	}

	// ── 2. A/B testing: распределяем устройства по variant'ам ──
	if s.cfg.ABTestingEnabled {
		results = s.assignVariants(results)
	}

	// ── 3. Публикуем каждое предсказание в NATS ──
	published := 0
	for _, r := range results {
		// Фильтр: публикуем только actionable или anomaly
		if !r.IsActionable && !r.IsAnomaly {
			continue
		}

		// Фильтр: минимальный confidence
		if r.ConfidenceScore < s.cfg.MinConfidenceThreshold {
			continue
		}

		if err := s.publishPrediction(ctx, r); err != nil {
			s.logger.ErrorContext(ctx, "failed to publish prediction",
				"device_id", r.DeviceID,
				"error", err,
			)
			continue
		}
		published++
	}

	s.logger.InfoContext(ctx, "batch prediction complete",
		"total", len(results),
		"published", published,
		"actionable", meta.Meta.Actionable,
		"avg_probability", fmt.Sprintf("%.2f", meta.Meta.AvgProbability),
	)

	return published, nil
}

// RunSingleDevice запускает предсказание для одного устройства.
func (s *PredictionService) RunSingleDevice(ctx context.Context, deviceID string) (*PredictionResult, error) {
	traceID := newTraceID()

	args := []string{
		s.cfg.ScriptPath,
		"--device", deviceID,
		"--variant", s.cfg.ModelVariant,
		"--trace", traceID,
	}
	cmd := exec.CommandContext(ctx, s.cfg.PythonPath, args...)
	cmd.Dir = "."

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start python: %w", err)
	}

	results, _, err := s.parseOutput(bufio.NewReader(stdout))
	if err != nil {
		_ = cmd.Wait()
		return nil, fmt.Errorf("parse output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("python exit: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no prediction for device %s", deviceID)
	}

	return &results[0], nil
}

// ── Internal: Python subprocess ─────────────────────────────────────

func (s *PredictionService) runPython(ctx context.Context, traceID string) ([]PredictionResult, MetaInfo, error) {
	args := []string{
		s.cfg.ScriptPath,
		"--variant", s.cfg.ModelVariant,
		"--trace", traceID,
	}

	cmd := exec.CommandContext(ctx, s.cfg.PythonPath, args...)
	cmd.Dir = "."

	s.logger.DebugContext(ctx, "running python prediction",
		"cmd", fmt.Sprintf("%s %s", s.cfg.PythonPath, strings.Join(args, " ")),
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, MetaInfo{}, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, MetaInfo{}, fmt.Errorf("stderr pipe: %w", err)
	}

	// Оборачиваем io.ReadCloser в bufio.Reader
	stdoutReader := bufio.NewReader(stdout)

	if err := cmd.Start(); err != nil {
		return nil, MetaInfo{}, fmt.Errorf("start: %w", err)
	}

	// Читаем stderr в фоне
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			s.logger.DebugContext(ctx, "[predict.py] "+scanner.Text())
		}
	}()

	results, meta, err := s.parseOutput(stdoutReader)
	if err != nil {
		_ = cmd.Wait()
		return nil, MetaInfo{}, fmt.Errorf("parse: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		// Если контекст отменён — это ожидаемо
		if ctx.Err() != nil {
			return results, meta, ctx.Err()
		}
		return nil, MetaInfo{}, fmt.Errorf("python exit code: %w", err)
	}

	return results, meta, nil
}

// ── Internal: JSONL parser ──────────────────────────────────────────

func (s *PredictionService) parseOutput(stdout *bufio.Reader) ([]PredictionResult, MetaInfo, error) {
	var results []PredictionResult
	var meta MetaInfo
	scanner := bufio.NewScanner(stdout)

	// Увеличиваем буфер для длинных строк с features_snapshot
	scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Проверяем, не мета-ли это строка
		if strings.Contains(line, `"_meta"`) {
			if err := json.Unmarshal([]byte(line), &meta); err != nil {
				s.logger.Warn("failed to parse meta line", "error", err)
			}
			continue
		}

		var result PredictionResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			s.logger.Warn("failed to parse prediction line",
				"error", err,
				"preview", truncateString(line, 200),
			)
			continue
		}

		if result.DeviceID == "" {
			s.logger.Warn("prediction without device_id, skipping")
			continue
		}

		results = append(results, result)
	}

	if err := scanner.Err(); err != nil {
		return results, meta, fmt.Errorf("scanner error: %w", err)
	}

	return results, meta, nil
}

// ── Internal: A/B variant assignment ────────────────────────────────

func (s *PredictionService) assignVariants(results []PredictionResult) []PredictionResult {
	ratio := s.cfg.ABTestingRatio
	if ratio <= 0 || ratio >= 1 {
		return results // не меняем variant'ы
	}

	// Детерминированное распределение по device_id (hash-based)
	for i, r := range results {
		hash := hashDeviceID(r.DeviceID)
		if float64(hash)/float64(math.MaxUint32) < ratio {
			results[i].ModelVariant = "B"
		} else {
			results[i].ModelVariant = "A"
		}
	}

	return results
}

// hashDeviceID — простой детерминированный хеш device_id.
func hashDeviceID(id string) uint32 {
	var h uint32
	for _, c := range id {
		h = h*31 + uint32(c)
	}
	return h
}

// ── Internal: NATS publish ──────────────────────────────────────────

func (s *PredictionService) publishPrediction(ctx context.Context, r PredictionResult) error {
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

	// Публикуем через NATS (без JetStream для простоты)
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

	s.logger.DebugContext(ctx, "prediction published",
		"subject", subject,
		"device_id", r.DeviceID,
		"probability", r.FailureProbability,
		"confidence", r.ConfidenceScore,
		"variant", r.ModelVariant,
	)

	return nil
}

// ── Utility ─────────────────────────────────────────────────────────

func newTraceID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Close освобождает ресурсы (не закрывает NATS — внешнее владение).
func (s *PredictionService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logger.Info("prediction service closed")
}
