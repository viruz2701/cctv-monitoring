// Package ai — Anomaly Detection Service.
//
// P2-AI.4: Anomaly Detection Service
//   - Объединяет AnomalyDetector, NATS publishing и WebSocket broadcast
//   - Предоставляет единый интерфейс для API хендлеров
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// ─── Broadcaster интерфейс ────────────────────────────────────────────────

// Broadcaster — интерфейс для отправки событий через WebSocket.
type Broadcaster interface {
	// Broadcast отправляет сообщение всем подключённым клиентам.
	Broadcast(data []byte)
}

// ─── AnomalyService ───────────────────────────────────────────────────────

// AnomalyService — сервис обнаружения аномалий.
//
// Предоставляет:
//   - Feed() для приёма метрик устройств
//   - Автоматическую публикацию в NATS при обнаружении аномалий
//   - WebSocket broadcast для real-time уведомлений
//   - API для получения/подтверждения/разрешения аномалий
type AnomalyService struct {
	cfg         AnomalyConfig
	detector    *AnomalyDetector
	nc          *nats.Conn
	js          nats.JetStreamContext
	broadcaster Broadcaster
	logger      *slog.Logger
}

// NewAnomalyService создаёт новый AnomalyService.
func NewAnomalyService(cfg AnomalyConfig, nc *nats.Conn, broadcaster Broadcaster, logger *slog.Logger) (*AnomalyService, error) {
	if logger == nil {
		logger = slog.Default()
	}

	var js nats.JetStreamContext
	if nc != nil {
		var err error
		js, err = nc.JetStream()
		if err != nil {
			logger.Warn("jetstream not available, nats publishing disabled", "error", err)
		}
	}

	return &AnomalyService{
		cfg:         cfg,
		detector:    NewAnomalyDetector(cfg, logger),
		nc:          nc,
		js:          js,
		broadcaster: broadcaster,
		logger:      logger.With("service", "anomaly"),
	}, nil
}

// FeedMetric добавляет метрику и проверяет на аномалию.
// Возвращает обнаруженную аномалию (nil если не аномалия).
func (s *AnomalyService) FeedMetric(ctx context.Context, m DeviceMetricPoint) *AnomalyResult {
	anomaly := s.detector.Feed(ctx, m)
	if anomaly != nil {
		s.publishEvent(ctx, AnomalyEventDetected, anomaly)
		s.broadcastAnomaly(AnomalyEventDetected, anomaly)
	}
	return anomaly
}

// GetActiveAnomalies возвращает активные аномалии с фильтрацией.
func (s *AnomalyService) GetActiveAnomalies(deviceID, metricType, severity string) []AnomalyResult {
	return s.detector.GetActiveAnomalies(deviceID, metricType, severity)
}

// GetAllAnomalies возвращает все аномалии с фильтрацией и лимитом.
func (s *AnomalyService) GetAllAnomalies(deviceID, metricType, severity, status string, limit int) []AnomalyResult {
	return s.detector.GetAllAnomalies(deviceID, metricType, severity, status, limit)
}

// AcknowledgeAnomaly подтверждает аномалию.
func (s *AnomalyService) AcknowledgeAnomaly(id string) error {
	return s.detector.AcknowledgeAnomaly(id)
}

// ResolveAnomaly разрешает аномалию.
func (s *AnomalyService) ResolveAnomaly(id string) error {
	anomaly := s.findAnomalyByID(id)
	if err := s.detector.ResolveAnomaly(id); err != nil {
		return err
	}
	// Шлём событие о разрешении
	if anomaly != nil {
		s.publishEvent(context.Background(), AnomalyEventResolved, anomaly)
		s.broadcastAnomaly(AnomalyEventResolved, anomaly)
	}
	return nil
}

// GetDetector возвращает внутренний детектор (для прямого доступа).
func (s *AnomalyService) GetDetector() *AnomalyDetector {
	return s.detector
}

// ─── Internal: NATS publishing ────────────────────────────────────────────

// publishEvent публикует событие аномалии в NATS.
func (s *AnomalyService) publishEvent(ctx context.Context, eventType AnomalyEventType, anomaly *AnomalyResult) {
	if s.nc == nil {
		return
	}

	event := AnomalyEvent{
		Type:    eventType,
		Payload: *anomaly,
	}

	data, err := json.Marshal(event)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal anomaly event",
			"error", err,
			"anomaly_id", anomaly.ID,
		)
		return
	}

	subject := fmt.Sprintf("%s.%s", s.cfg.NATSTopicPrefix, anomaly.DeviceID)

	// Публикуем в NATS
	if err := s.nc.Publish(subject, data); err != nil {
		s.logger.WarnContext(ctx, "nats publish failed",
			"subject", subject,
			"error", err,
		)
	}

	// Публикуем в JetStream для durability
	if s.js != nil {
		if _, err := s.js.Publish(subject, data); err != nil {
			s.logger.WarnContext(ctx, "jetstream publish failed",
				"subject", subject,
				"error", err,
			)
		}
	}

	s.logger.DebugContext(ctx, "anomaly event published",
		"subject", subject,
		"event_type", eventType,
		"anomaly_id", anomaly.ID,
	)
}

// broadcastAnomaly отправляет событие через WebSocket broadcaster.
func (s *AnomalyService) broadcastAnomaly(eventType AnomalyEventType, anomaly *AnomalyResult) {
	if s.broadcaster == nil {
		return
	}

	event := AnomalyEvent{
		Type:    eventType,
		Payload: *anomaly,
	}

	data, err := json.Marshal(map[string]interface{}{
		"type":  "anomaly",
		"event": event,
	})
	if err != nil {
		s.logger.Error("failed to marshal anomaly broadcast",
			"error", err,
		)
		return
	}

	s.broadcaster.Broadcast(data)
}

// findAnomalyByID ищет аномалию по ID во всех хранилищах.
func (s *AnomalyService) findAnomalyByID(id string) *AnomalyResult {
	all := s.detector.GetAllAnomalies("", "", "", "", 0)
	for i := range all {
		if all[i].ID == id {
			return &all[i]
		}
	}
	return nil
}

// ─── Health ───────────────────────────────────────────────────────────────

// Health возвращает статус сервиса.
func (s *AnomalyService) Health() map[string]interface{} {
	active := s.GetActiveAnomalies("", "", "")
	stats := s.detector.GetBufferStats()

	totalPoints := 0
	for _, count := range stats {
		totalPoints += count
	}

	return map[string]interface{}{
		"active_anomalies":    len(active),
		"metric_buffers":      len(stats),
		"total_metric_points": totalPoints,
		"nats_connected":      s.nc != nil,
		"last_evaluation":     time.Now().UTC().Format(time.RFC3339),
	}
}
