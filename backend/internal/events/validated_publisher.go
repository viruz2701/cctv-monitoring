// Package events — Validated Publisher с Schema Registry validation.
//
// Оборачивает Publisher и выполняет JSON Schema validation перед публикацией.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist validation)
//   - OWASP ASVS V5.3 (Input validation — structured data validation)
//   - ISO 27001 A.12.4.1 (Event logging — data quality enforcement)
//   - IEC 62443 SR 3.1 (Data integrity validation)
package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// ValidationMetrics — Prometheus-совместимые счётчики валидации.
// ═══════════════════════════════════════════════════════════════════════

// ValidationStats содержит атомарные счётчики для мониторинга валидации.
type ValidationStats struct {
	TotalValidations atomic.Int64 `json:"total_validations"`
	ValidEvents      atomic.Int64 `json:"valid_events"`
	InvalidEvents    atomic.Int64 `json:"invalid_events"`
	SchemaNotFound   atomic.Int64 `json:"schema_not_found"`
	ValidationErrors atomic.Int64 `json:"validation_errors"`
}

// Snapshot возвращает текущие значения всех счётчиков.
func (s *ValidationStats) Snapshot() map[string]int64 {
	return map[string]int64{
		"total_validations": s.TotalValidations.Load(),
		"valid_events":      s.ValidEvents.Load(),
		"invalid_events":    s.InvalidEvents.Load(),
		"schema_not_found":  s.SchemaNotFound.Load(),
		"validation_errors": s.ValidationErrors.Load(),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// ValidatedPublisher — Publisher с валидацией.
// ═══════════════════════════════════════════════════════════════════════

// ValidatedPublisher оборачивает Publisher и выполняет валидацию
// всех событий через SchemaRegistry перед публикацией.
type ValidatedPublisher struct {
	publisher *Publisher
	registry  *SchemaRegistry
	logger    *slog.Logger
	stats     *ValidationStats
	enabled   bool // можно отключить валидацию (например, для тестов)

	// Circuit breaker
	cbConfig   CircuitBreakerConfig
	cbState    CircuitBreakerState
	cbMu       sync.RWMutex
	cbOpenedAt time.Time
}

// ValidatedPublisherConfig — конфигурация ValidatedPublisher.
type ValidatedPublisherConfig struct {
	Publisher *Publisher
	Registry  *SchemaRegistry
	Logger    *slog.Logger
	Enabled   bool // включить валидацию (default: true)
}

// NewValidatedPublisher создаёт ValidatedPublisher.
func NewValidatedPublisher(cfg ValidatedPublisherConfig) *ValidatedPublisher {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &ValidatedPublisher{
		publisher: cfg.Publisher,
		registry:  cfg.Registry,
		logger:    cfg.Logger.With("component", "validated_publisher"),
		stats:     &ValidationStats{},
		enabled:   true,
		cbConfig:  DefaultCircuitBreakerConfig,
		cbState:   CBClosed,
	}
}

// Publisher возвращает внутренний Publisher для прямых операций.
func (vp *ValidatedPublisher) Publisher() *Publisher {
	return vp.publisher
}

// Registry возвращает SchemaRegistry.
func (vp *ValidatedPublisher) Registry() *SchemaRegistry {
	return vp.registry
}

// Stats возвращает статистику валидации.
func (vp *ValidatedPublisher) Stats() *ValidationStats {
	return vp.stats
}

// SetEnabled включает/отключает валидацию.
func (vp *ValidatedPublisher) SetEnabled(enabled bool) {
	vp.enabled = enabled
	vp.logger.Info("validation enabled", "enabled", enabled)
}

// PublishEvent публикует EventRecord после валидации.
func (vp *ValidatedPublisher) PublishEvent(record *EventRecord) error {
	subject := subjectForRecord(record)
	return vp.validateAndPublish(subject, record)
}

// PublishAlarm публикует событие тревоги через EventRecord.
func (vp *ValidatedPublisher) PublishAlarm(event AlarmEvent) error {
	record := eventToRecord(SourceAlarms, "alarm.created", event)
	return vp.PublishEvent(record)
}

// PublishCMMS публикует событие CMMS через EventRecord.
func (vp *ValidatedPublisher) PublishCMMS(event CMMSEvent) error {
	record := eventToRecord(SourceCMMS, "cmms.wo."+event.Event, event)
	return vp.PublishEvent(record)
}

// PublishPrediction публикует предиктивный прогноз через EventRecord.
func (vp *ValidatedPublisher) PublishPrediction(event PredictionEvent) error {
	record := eventToRecord(SourcePredictions, "prediction.created", event)
	return vp.PublishEvent(record)
}

// PublishTelemetry публикует телеметрию через EventRecord.
func (vp *ValidatedPublisher) PublishTelemetry(event TelemetryEvent) error {
	record := eventToRecord(SourceTelemetry, "telemetry.metric", event)
	return vp.PublishEvent(record)
}

// Close закрывает внутренний Publisher.
func (vp *ValidatedPublisher) Close() {
	vp.publisher.Close()
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// subjectForRecord определяет NATS subject для EventRecord.
func subjectForRecord(record *EventRecord) string {
	switch record.Source {
	case SourceAlarms:
		return fmt.Sprintf(TopicAlarms, record.AggregateID)
	case SourceCMMS:
		return fmt.Sprintf(TopicCMMSWO, record.EventType)
	case SourcePredictions:
		return fmt.Sprintf(TopicPredictions, record.AggregateID)
	case SourceTelemetry:
		return fmt.Sprintf(TopicTelemetry, record.AggregateID)
	default:
		return fmt.Sprintf("events.%s.%s", record.Source, record.EventType)
	}
}

// eventToRecord конвертирует доменное событие в EventRecord.
func eventToRecord(source EventSource, eventType string, data interface{}) *EventRecord {
	raw, _ := json.Marshal(data)
	return &EventRecord{
		Source:        source,
		EventType:     eventType,
		SchemaVersion: "1.0.0",
		Timestamp:     timeNow(),
		Data:          raw,
	}
}

// timeNow — обёртка для тестирования (можно переопределить).
var timeNow = func() time.Time {
	return time.Now().UTC()
}

// ═══════════════════════════════════════════════════════════════════════
// Circuit Breaker
// ═══════════════════════════════════════════════════════════════════════

// SetCircuitBreakerConfig устанавливает конфигурацию circuit breaker.
func (vp *ValidatedPublisher) SetCircuitBreakerConfig(cfg CircuitBreakerConfig) {
	vp.cbMu.Lock()
	defer vp.cbMu.Unlock()
	vp.cbConfig = cfg
	vp.logger.Info("circuit breaker configured",
		"failure_threshold", cfg.FailureThreshold,
		"min_count", cfg.MinValidationCount,
		"reset_interval", cfg.AutoResetInterval,
		"enabled", cfg.Enabled,
	)
}

// CircuitBreakerState возвращает текущее состояние circuit breaker.
func (vp *ValidatedPublisher) CircuitBreakerState() CircuitBreakerState {
	vp.cbMu.RLock()
	defer vp.cbMu.RUnlock()
	return vp.cbState
}

// checkCircuitBreaker проверяет, нужно ли отключить валидацию.
// Возвращает true если валидация должна быть активна.
func (vp *ValidatedPublisher) checkCircuitBreaker() bool {
	vp.cbMu.Lock()
	defer vp.cbMu.Unlock()

	// Auto-reset: если circuit open и прошло достаточно времени — закрываем
	if vp.cbState == CBOpen {
		if time.Since(vp.cbOpenedAt) > vp.cbConfig.AutoResetInterval {
			vp.cbState = CBClosed
			vp.logger.Warn("circuit breaker auto-reset: validation re-enabled",
				"failure_threshold", vp.cbConfig.FailureThreshold,
			)
		} else {
			return false
		}
	}

	// Проверяем, нужно ли открыть circuit
	total := vp.stats.TotalValidations.Load()
	if total < vp.cbConfig.MinValidationCount {
		return true // Слишком мало данных для решения
	}

	invalid := vp.stats.InvalidEvents.Load()
	failureRate := float64(invalid) / float64(total)

	if failureRate > vp.cbConfig.FailureThreshold && total >= vp.cbConfig.MinValidationCount {
		vp.cbState = CBOpen
		vp.cbOpenedAt = time.Now()
		vp.logger.Error("circuit breaker opened: validation disabled due to high failure rate",
			"failure_rate", failureRate,
			"threshold", vp.cbConfig.FailureThreshold,
			"total_validations", total,
			"invalid_events", invalid,
		)
		return false
	}

	return true
}

// updateValidateAndPublish — обновлённый метод с circuit breaker.
func (vp *ValidatedPublisher) validateAndPublish(subject string, record *EventRecord) error {
	if !vp.enabled {
		return vp.publisher.publishJSON(subject, record)
	}

	// Проверяем circuit breaker
	if !vp.checkCircuitBreaker() {
		vp.logger.Warn("event published without validation (circuit breaker open)",
			"subject", subject,
			"source", record.Source,
		)
		return vp.publisher.publishJSON(subject, record)
	}

	vp.stats.TotalValidations.Add(1)

	if err := vp.registry.Validate(record); err != nil {
		vp.stats.InvalidEvents.Add(1)
		vp.logger.Error("event validation failed",
			"subject", subject,
			"source", record.Source,
			"event_type", record.EventType,
			"error", err,
			"trace_id", record.TraceID,
		)

		// Проверяем circuit breaker после неудачной валидации
		vp.checkCircuitBreaker()

		return fmt.Errorf("validated publish: %w", err)
	}

	vp.stats.ValidEvents.Add(1)
	return vp.publisher.publishJSON(subject, record)
}
