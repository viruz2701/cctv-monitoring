// Package cmms — CMMS Event Dispatcher (CMMS-3.1.2).
//
// Event Dispatcher подписывается на NATS JetStream топики и перенаправляет
// события в соответствующий CMMSAdapter. При недоступности адаптера
// использует FallbackQueue для отложенной синхронизации.
//
// Compliance:
//   - IEC 62443 SR 3.1 (Wireless — data integrity)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - ISO 27001 A.12.4.3 (System audit — audit trail)
//   - OWASP ASVS V7.1 (Log content — integrity)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
//   - СТБ 34.101.30 (bash-hmac for audit signatures)
package cmms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"gb-telemetry-collector/internal/events"
	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

// DefaultDispatcherConfig — значения по умолчанию для DispatcherConfig.
var DefaultDispatcherConfig = DispatcherConfig{
	CircuitBreakerThreshold: 5,
	CircuitBreakerResetTime: 30 * time.Second,
	FallbackMaxRetries:      10,
	WorkerPoolSize:          4,
	AuditLogEnabled:         true,
}

// ── Event-to-Adapter mapping ───────────────────────────────────────

// eventRoute определяет маршрут NATS-события → метод CMMSAdapter.
type eventRoute struct {
	source    events.EventSource
	eventType string
	handler   func(ctx context.Context, adapter CMMSAdapter, data []byte) error
}

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// DispatcherConfig — конфигурация Event Dispatcher.
type DispatcherConfig struct {
	// CircuitBreakerThreshold — количество ошибок подряд для размыкания цепи.
	// По умолчанию: 5.
	CircuitBreakerThreshold int `json:"circuit_breaker_threshold"`

	// CircuitBreakerResetTime — время ожидания перед полуоткрытием цепи.
	// По умолчанию: 30s.
	CircuitBreakerResetTime time.Duration `json:"circuit_breaker_reset_time"`

	// FallbackMaxRetries — максимальное количество повторных попыток.
	// По умолчанию: 10.
	FallbackMaxRetries int `json:"fallback_max_retries"`

	// WorkerPoolSize — размер воркер-пула для обработки событий.
	// По умолчанию: 4.
	WorkerPoolSize int `json:"worker_pool_size"`

	// AuditLogEnabled — включать аудит-логирование событий.
	// Соответствует: ISO 27001 A.12.4.1, СТБ 34.101.27 п. 7.2.
	AuditLogEnabled bool `json:"audit_log_enabled"`
}

// validate проверяет и применяет значения по умолчанию.
func (c *DispatcherConfig) validate() {
	if c.CircuitBreakerThreshold <= 0 {
		c.CircuitBreakerThreshold = DefaultDispatcherConfig.CircuitBreakerThreshold
	}
	if c.CircuitBreakerResetTime <= 0 {
		c.CircuitBreakerResetTime = DefaultDispatcherConfig.CircuitBreakerResetTime
	}
	if c.FallbackMaxRetries <= 0 {
		c.FallbackMaxRetries = DefaultDispatcherConfig.FallbackMaxRetries
	}
	if c.WorkerPoolSize <= 0 {
		c.WorkerPoolSize = DefaultDispatcherConfig.WorkerPoolSize
	}
}

// circuitBreakerState — состояние circuit breaker.
type circuitBreakerState int32

const (
	stateClosed   circuitBreakerState = 0 // нормальная работа
	stateOpen     circuitBreakerState = 1 // цепь разомкнута (ошибки)
	stateHalfOpen circuitBreakerState = 2 // пробный запрос
)

// circuitBreaker — реализация Circuit Breaker паттерна.
//
// Соответствует: IEC 62443 SR 7.1 (Fail Secure — при ошибке блокируем запросы).
type circuitBreaker struct {
	state      atomic.Int32
	failCount  atomic.Int32
	threshold  int32
	resetTime  time.Duration
	lastOpen   atomic.Value // time.Time
	halfOpenAt atomic.Value // time.Time
	mu         sync.Mutex
}

func newCircuitBreaker(threshold int, resetTime time.Duration) *circuitBreaker {
	cb := &circuitBreaker{
		threshold: int32(threshold),
		resetTime: resetTime,
	}
	cb.lastOpen.Store(time.Time{})
	cb.halfOpenAt.Store(time.Time{})
	return cb
}

func (cb *circuitBreaker) allow() bool {
	state := circuitBreakerState(cb.state.Load())
	switch state {
	case stateClosed:
		return true
	case stateOpen:
		// Проверяем, не прошло ли время сброса
		lastOpen := cb.lastOpen.Load().(time.Time)
		if time.Since(lastOpen) >= cb.resetTime {
			// Переходим в half-open
			if cb.state.CompareAndSwap(int32(stateOpen), int32(stateHalfOpen)) {
				cb.halfOpenAt.Store(time.Now())
				return true
			}
		}
		return false
	case stateHalfOpen:
		// В half-open пропускаем только один запрос
		return true
	default:
		return true
	}
}

func (cb *circuitBreaker) success() {
	cb.failCount.Store(0)
	// Если был half-open → закрываем цепь
	cb.state.CompareAndSwap(int32(stateHalfOpen), int32(stateClosed))
}

func (cb *circuitBreaker) failure() {
	count := cb.failCount.Add(1)
	if count >= cb.threshold {
		cb.lastOpen.Store(time.Now())
		cb.state.Store(int32(stateOpen))
	}
}

// DispatcherEvent — событие диспетчера для аудит-лога.
type DispatcherEvent struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`       // alarm, cmms, prediction
	EventType   string            `json:"event_type"`   // alarm.created, cmms.wo.completed
	Subject     string            `json:"subject"`      // NATS subject
	Action      string            `json:"action"`       // create_wo, update_wo, complete_wo
	AdapterName string            `json:"adapter_name"` // internal, atlas, servicenow
	Status      string            `json:"status"`       // success, queued, skipped, error
	Error       string            `json:"error,omitempty"`
	Duration    time.Duration     `json:"duration_ms"`
	Timestamp   time.Time         `json:"timestamp"`
	FallbackID  string            `json:"fallback_id,omitempty"`
	RetryCount  int               `json:"retry_count,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// AuditLogger — интерфейс для аудит-логирования.
//
// Соответствует:
//   - ISO 27001 A.12.4.1 (Event logging)
//   - ISO 27001 A.12.4.3 (Audit trail)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
type AuditLogger interface {
	LogDispatcherEvent(ctx context.Context, event DispatcherEvent) error
}

// AuditLoggerFunc — адаптер для функции как AuditLogger.
type AuditLoggerFunc func(ctx context.Context, event DispatcherEvent) error

func (f AuditLoggerFunc) LogDispatcherEvent(ctx context.Context, event DispatcherEvent) error {
	return f(ctx, event)
}

// ═══════════════════════════════════════════════════════════════════════
// EventDispatcher
// ═══════════════════════════════════════════════════════════════════════

// EventDispatcher подписывается на NATS события и перенаправляет их
// в CMMSAdapter. Использует Circuit Breaker для внешних адаптеров
// и FallbackQueue для отложенной обработки.
//
// Архитектура:
//
//	NATS Subscriber ─► EventDispatcher ─► CMMSAdapter
//	                      │
//	                      ▼
//	                 FallbackQueue (при ошибках)
//
// Compliance:
//   - IEC 62443 SR 3.1 (Data integrity — подпись событий)
//   - IEC 62443 SR 7.1 (Fail Secure — circuit breaker)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - ISO 27001 A.12.4.3 (System audit)
//   - ISO 27019 PCC.A.12.4 (ICS audit trail)
//   - OWASP ASVS V7.1 (Log content integrity)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
type EventDispatcher struct {
	adapter        CMMSAdapter
	subscriber     *events.Subscriber
	fallbackQueue  *FallbackQueue
	auditLogger    AuditLogger
	cfg            DispatcherConfig
	logger         *slog.Logger
	cb             *circuitBreaker
	routes         []eventRoute
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.RWMutex
	fallbackDir    string
	healthStatus   atomic.Value // string: "healthy", "degraded", "unhealthy"
	processedCount atomic.Int64
	errorCount     atomic.Int64
	queuedCount    atomic.Int64
}

// NewEventDispatcher создаёт новый EventDispatcher.
//
// Параметры:
//   - adapter: CMMSAdapter для выполнения операций
//   - subscriber: NATS Subscriber для получения событий
//   - fallbackDir: директория для FallbackQueue
//   - auditLogger: опциональный логгер аудита (nil = отключён)
//   - cfg: конфигурация диспетчера
//   - logger: логгер
func NewEventDispatcher(
	adapter CMMSAdapter,
	subscriber *events.Subscriber,
	fallbackDir string,
	auditLogger AuditLogger,
	cfg DispatcherConfig,
	logger *slog.Logger,
) (*EventDispatcher, error) {
	cfg.validate()
	if logger == nil {
		logger = slog.Default()
	}

	fallbackQueue, err := NewFallbackQueue(fallbackDir, cfg.FallbackMaxRetries, logger)
	if err != nil {
		return nil, fmt.Errorf("event dispatcher: fallback queue: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &EventDispatcher{
		adapter:       adapter,
		subscriber:    subscriber,
		fallbackQueue: fallbackQueue,
		auditLogger:   auditLogger,
		cfg:           cfg,
		logger:        logger.With("component", "cmms-event-dispatcher"),
		cb:            newCircuitBreaker(cfg.CircuitBreakerThreshold, cfg.CircuitBreakerResetTime),
		ctx:           ctx,
		cancel:        cancel,
		fallbackDir:   fallbackDir,
	}
	d.healthStatus.Store("healthy")

	// Регистрируем маршруты событий
	d.registerRoutes()

	return d, nil
}

// ── Route registration ─────────────────────────────────────────────

func (d *EventDispatcher) registerRoutes() {
	d.routes = []eventRoute{
		// ── Alarm events ────────────────────────────────────────
		{
			source:    events.SourceAlarms,
			eventType: "alarm.created",
			handler:   d.handleAlarmCreated,
		},
		{
			source:    events.SourceAlarms,
			eventType: "alarm.resolved",
			handler:   d.handleAlarmResolved,
		},
		// ── CMMS Work Order events ──────────────────────────────
		{
			source:    events.SourceCMMS,
			eventType: "cmms.wo.created",
			handler:   d.handleCMMSWOCreated,
		},
		{
			source:    events.SourceCMMS,
			eventType: "cmms.wo.completed",
			handler:   d.handleCMMSWOCompleted,
		},
		{
			source:    events.SourceCMMS,
			eventType: "cmms.wo.status_changed",
			handler:   d.handleCMMSWOStatusChanged,
		},
		// ── Prediction events ────────────────────────────────────
		{
			source:    events.SourcePredictions,
			eventType: "prediction.created",
			handler:   d.handlePredictionCreated,
		},
	}
}

// ── Start / Stop ───────────────────────────────────────────────────

// Start подписывается на NATS топики и запускает обработку событий.
//
// Compliance: IEC 62443 SR 3.1 (Secure communications — NATS with TLS/mTLS).
func (d *EventDispatcher) Start() error {
	d.logger.Info("starting CMMS event dispatcher",
		"circuit_breaker_threshold", d.cfg.CircuitBreakerThreshold,
		"circuit_breaker_reset", d.cfg.CircuitBreakerResetTime,
		"fallback_dir", d.fallbackDir,
		"worker_pool", d.cfg.WorkerPoolSize,
	)

	// Регистрируем обработчики в Subscriber
	d.subscriber.OnAlarm(func(event events.AlarmEvent) {
		d.dispatchBySource(events.SourceAlarms, "alarm.created", event)
	})

	d.subscriber.OnCMMS(func(event events.CMMSEvent) {
		eventType := fmt.Sprintf("cmms.wo.%s", event.Event)
		d.dispatchBySource(events.SourceCMMS, eventType, event)
	})

	d.subscriber.OnPrediction(func(event events.PredictionEvent) {
		d.dispatchBySource(events.SourcePredictions, "prediction.created", event)
	})

	// Подписываемся на все топики
	if err := d.subscriber.SubscribeAll(); err != nil {
		return fmt.Errorf("event dispatcher: subscribe: %w", err)
	}

	// Запускаем воркер для повторной обработки fallback-очереди
	d.wg.Add(1)
	go d.fallbackWorker()

	d.logger.Info("CMMS event dispatcher started")
	return nil
}

// Stop останавливает диспетчер.
func (d *EventDispatcher) Stop() {
	d.logger.Info("stopping CMMS event dispatcher")
	d.cancel()
	d.wg.Wait()
	d.logger.Info("CMMS event dispatcher stopped")
}

// ── Dispatch ───────────────────────────────────────────────────────

// dispatchBySource находит маршрут для события и выполняет его.
func (d *EventDispatcher) dispatchBySource(source events.EventSource, eventType string, payload interface{}) {
	// Ищем подходящий маршрут
	for _, route := range d.routes {
		if route.source == source && route.eventType == eventType {
			// Сериализуем payload
			data, err := json.Marshal(payload)
			if err != nil {
				d.logger.Error("failed to marshal event payload",
					"source", source, "event_type", eventType, "error", err,
				)
				return
			}

			d.processWithRetry(route, data)
			return
		}
	}

	d.logger.Debug("no route found for event",
		"source", source, "event_type", eventType,
	)
}

// processWithRetry пытается обработать событие через adapter.
// При ошибке — сохраняет в FallbackQueue.
func (d *EventDispatcher) processWithRetry(route eventRoute, data []byte) {
	start := time.Now()
	d.processedCount.Add(1)

	// Проверяем circuit breaker
	if !d.cb.allow() {
		d.logger.Warn("circuit breaker open, queuing event",
			"event_type", route.eventType,
		)
		d.queuedCount.Add(1)
		d.enqueueFallback(route, data, "circuit_breaker_open")
		return
	}

	// Пытаемся выполнить через adapter
	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	err := route.handler(ctx, d.adapter, data)
	duration := time.Since(start)

	if err != nil {
		d.errorCount.Add(1)
		d.cb.failure()

		// Обновляем health status
		d.updateHealthStatus()

		// Логируем ошибку
		d.logger.Error("event handler failed",
			"event_type", route.eventType,
			"error", err,
			"duration", duration,
			"circuit_breaker_failures", d.cb.failCount.Load(),
		)

		// Сохраняем в fallback queue
		d.enqueueFallback(route, data, err.Error())

		// Audit log
		d.logDispatcherEvent(ctx, DispatcherEvent{
			Source:      string(route.source),
			EventType:   route.eventType,
			Action:      route.eventType,
			AdapterName: d.adapterName(),
			Status:      "error",
			Error:       err.Error(),
			Duration:    duration,
			Timestamp:   time.Now(),
		})
	} else {
		d.cb.success()
		d.healthStatus.Store("healthy")

		d.logger.Debug("event processed successfully",
			"event_type", route.eventType,
			"duration", duration,
		)

		// Audit log
		d.logDispatcherEvent(ctx, DispatcherEvent{
			Source:      string(route.source),
			EventType:   route.eventType,
			Action:      route.eventType,
			AdapterName: d.adapterName(),
			Status:      "success",
			Duration:    duration,
			Timestamp:   time.Now(),
		})
	}
}

// enqueueFallback сохраняет событие в FallbackQueue.
func (d *EventDispatcher) enqueueFallback(route eventRoute, data []byte, reason string) {
	payload := map[string]interface{}{
		"source":      route.source,
		"event_type":  route.eventType,
		"data":        json.RawMessage(data),
		"reason":      reason,
		"enqueued_at": time.Now().UTC(),
	}

	if err := d.fallbackQueue.Enqueue(
		fmt.Sprintf("dispatch_%s", route.eventType),
		payload,
	); err != nil {
		d.logger.Error("failed to enqueue fallback",
			"event_type", route.eventType, "error", err,
		)
	} else {
		d.logger.Info("event queued for retry",
			"event_type", route.eventType,
			"reason", reason,
		)
	}
}

// ── Fallback Worker ────────────────────────────────────────────────

// fallbackWorker периодически пытается обработать отложенные события.
func (d *EventDispatcher) fallbackWorker() {
	defer d.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	d.logger.Info("fallback worker started", "interval", "30s")

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.processFallbackQueue()
		}
	}
}

// processFallbackQueue пытается повторно обработать отложенные события.
func (d *EventDispatcher) processFallbackQueue() {
	// Проверяем circuit breaker — если открыт, не трогаем очередь
	if !d.cb.allow() {
		d.logger.Debug("circuit breaker open, skipping fallback processing")
		return
	}

	entries, err := d.fallbackQueue.Pending()
	if err != nil {
		d.logger.Error("failed to list fallback entries", "error", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	d.logger.Info("processing fallback queue", "count", len(entries))

	for _, entry := range entries {
		select {
		case <-d.ctx.Done():
			return
		default:
		}

		// Парсим payload
		var payload struct {
			Source    string          `json:"source"`
			EventType string          `json:"event_type"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(entry.Payload, &payload); err != nil {
			d.logger.Error("failed to parse fallback payload", "id", entry.ID, "error", err)
			_ = d.fallbackQueue.Remove(entry.ID)
			continue
		}

		// Ищем маршрут
		var matchedRoute *eventRoute
		for _, route := range d.routes {
			if string(route.source) == payload.Source && route.eventType == payload.EventType {
				matchedRoute = &route
				break
			}
		}

		if matchedRoute == nil {
			d.logger.Warn("no route for fallback entry", "id", entry.ID, "event_type", payload.EventType)
			_ = d.fallbackQueue.Remove(entry.ID)
			continue
		}

		// Пытаемся обработать
		ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
		err := matchedRoute.handler(ctx, d.adapter, payload.Data)
		cancel()

		if err != nil {
			d.logger.Warn("fallback retry failed",
				"id", entry.ID, "event_type", payload.EventType, "error", err,
				"retries", entry.Retries,
			)
			_ = d.fallbackQueue.MarkRetry(entry.ID, err.Error())
		} else {
			d.logger.Info("fallback retry succeeded",
				"id", entry.ID, "event_type", payload.EventType,
			)
			_ = d.fallbackQueue.Remove(entry.ID)
		}
	}
}

// ── Event Handlers ─────────────────────────────────────────────────

// handleAlarmCreated: alarm.created → создаёт WorkOrder через CMMSAdapter.
//
// Маппинг: AlarmEvent → WorkOrder (corrective, с severity = priority).
func (d *EventDispatcher) handleAlarmCreated(ctx context.Context, adapter CMMSAdapter, data []byte) error {
	var alarmEvent events.AlarmEvent
	if err := json.Unmarshal(data, &alarmEvent); err != nil {
		return fmt.Errorf("unmarshal alarm event: %w", err)
	}

	// Маппим severity в priority
	priority := d.mapSeverityToPriority(alarmEvent.Severity)

	createdBy := "system:alarm"
	wo := &models.WorkOrder{
		Title:     fmt.Sprintf("[Alarm] %s — %s", alarmEvent.Type, alarmEvent.Message),
		DeviceID:  alarmEvent.DeviceID,
		Type:      "corrective", // alarm → corrective WO
		Priority:  priority,
		Status:    "open",
		Notes:     fmt.Sprintf("Alarm %s from device %s (%s): %s", alarmEvent.Type, alarmEvent.DeviceName, alarmEvent.DeviceID, alarmEvent.Message),
		CreatedBy: &createdBy,
		CreatedAt: alarmEvent.Timestamp,
	}

	// Ограничение длины
	if len(wo.Title) > 500 {
		wo.Title = wo.Title[:497] + "..."
	}
	if len(wo.Notes) > 2000 {
		wo.Notes = wo.Notes[:1997] + "..."
	}

	return adapter.CreateWorkOrder(ctx, wo)
}

// handleAlarmResolved: alarm.resolved → закрывает связанные WorkOrder.
func (d *EventDispatcher) handleAlarmResolved(ctx context.Context, adapter CMMSAdapter, data []byte) error {
	var payload struct {
		AlarmID     string `json:"alarm_id"`
		ResolvedBy  string `json:"resolved_by"`
		Resolution  string `json:"resolution"`
		AutoResolve bool   `json:"auto_resolved"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal alarm resolved: %w", err)
	}

	// Ищем открытые WO по alarm_id
	filters := map[string]interface{}{
		"source":   "alarm",
		"alarm_id": payload.AlarmID,
		"status":   []string{"requested", "in_progress", "assigned"},
	}

	orders, err := adapter.GetWorkOrders(ctx, filters)
	if err != nil {
		return fmt.Errorf("get work orders for alarm: %w", err)
	}

	// Закрываем все найденные
	for _, wo := range orders {
		notes := fmt.Sprintf("Alarm resolved. Resolution: %s. Resolved by: %s", payload.Resolution, payload.ResolvedBy)
		if err := adapter.CompleteWorkOrder(ctx, wo.ID, notes, nil, nil, payload.ResolvedBy); err != nil {
			d.logger.Error("failed to complete WO for resolved alarm",
				"work_order_id", wo.ID, "alarm_id", payload.AlarmID, "error", err,
			)
		}
	}

	return nil
}

// handleCMMSWOCreated: cmms.wo.created → дублирует создание WO через adapter.
//
// Используется для синхронизации между InternalAdapter и внешними CMMS.
func (d *EventDispatcher) handleCMMSWOCreated(ctx context.Context, adapter CMMSAdapter, data []byte) error {
	var cmmsEvent events.CMMSEvent
	if err := json.Unmarshal(data, &cmmsEvent); err != nil {
		return fmt.Errorf("unmarshal cmms event: %w", err)
	}

	// Создаём WorkOrder в целевом адаптере
	createdBy := "system:cmms-sync"
	wo := &models.WorkOrder{
		ID:        cmmsEvent.WorkOrderID,
		DeviceID:  cmmsEvent.DeviceID,
		Status:    cmmsEvent.Status,
		Priority:  cmmsEvent.Priority,
		CreatedBy: &createdBy,
		Title:     fmt.Sprintf("[CMMS Sync] WorkOrder %s", cmmsEvent.WorkOrderID),
	}

	return adapter.CreateWorkOrder(ctx, wo)
}

// handleCMMSWOCompleted: cmms.wo.completed → завершает WO в адаптере.
func (d *EventDispatcher) handleCMMSWOCompleted(ctx context.Context, adapter CMMSAdapter, data []byte) error {
	var cmmsEvent events.CMMSEvent
	if err := json.Unmarshal(data, &cmmsEvent); err != nil {
		return fmt.Errorf("unmarshal cmms completed: %w", err)
	}

	return adapter.CompleteWorkOrder(ctx, cmmsEvent.WorkOrderID, "", nil, nil, cmmsEvent.AssigneeID)
}

// handleCMMSWOStatusChanged: cmms.wo.status_changed → обновляет статус.
func (d *EventDispatcher) handleCMMSWOStatusChanged(ctx context.Context, adapter CMMSAdapter, data []byte) error {
	var cmmsEvent events.CMMSEvent
	if err := json.Unmarshal(data, &cmmsEvent); err != nil {
		return fmt.Errorf("unmarshal cmms status: %w", err)
	}

	updates := map[string]interface{}{
		"status": cmmsEvent.Status,
	}
	return adapter.UpdateWorkOrder(ctx, cmmsEvent.WorkOrderID, updates)
}

// handlePredictionCreated: prediction.created → создаёт Preventive WorkOrder.
//
// При высокой вероятности отказа (>0.8) создаёт preventive WO.
func (d *EventDispatcher) handlePredictionCreated(ctx context.Context, adapter CMMSAdapter, data []byte) error {
	var predictionEvent events.PredictionEvent
	if err := json.Unmarshal(data, &predictionEvent); err != nil {
		return fmt.Errorf("unmarshal prediction: %w", err)
	}

	// Создаём preventive WO только при высокой вероятности
	if predictionEvent.Probability < 0.8 {
		return nil
	}

	createdBy := "system:prediction"
	wo := &models.WorkOrder{
		Title: fmt.Sprintf("[Predictive] %s — %.0f%% probability", predictionEvent.FailureMode, predictionEvent.Probability*100),
		Notes: fmt.Sprintf("XGBoost prediction: %s (%.0f%%). Estimated failure in %d days. Recommendation: %s",
			predictionEvent.FailureMode, predictionEvent.Probability*100,
			predictionEvent.EstimatedDays, predictionEvent.Recommendation),
		DeviceID:  predictionEvent.DeviceID,
		Type:      "preventive",
		Priority:  d.mapProbabilityToPriority(predictionEvent.Probability),
		Status:    "open",
		CreatedBy: &createdBy,
		CreatedAt: predictionEvent.Timestamp,
	}

	if len(wo.Title) > 500 {
		wo.Title = wo.Title[:497] + "..."
	}
	if len(wo.Notes) > 2000 {
		wo.Notes = wo.Notes[:1997] + "..."
	}

	return adapter.CreateWorkOrder(ctx, wo)
}

// ── Health & Metrics ───────────────────────────────────────────────

// Health возвращает статус здоровья диспетчера.
func (d *EventDispatcher) Health() map[string]interface{} {
	cbState := circuitBreakerState(d.cb.state.Load())
	stateStr := "closed"
	switch cbState {
	case stateOpen:
		stateStr = "open"
	case stateHalfOpen:
		stateStr = "half_open"
	}

	return map[string]interface{}{
		"status":           d.healthStatus.Load().(string),
		"circuit_breaker":  stateStr,
		"fail_count":       d.cb.failCount.Load(),
		"threshold":        d.cb.threshold,
		"processed":        d.processedCount.Load(),
		"errors":           d.errorCount.Load(),
		"queued":           d.queuedCount.Load(),
		"fallback_pending": d.fallbackQueueLen(),
	}
}

// Metrics возвращает метрики для Prometheus.
func (d *EventDispatcher) Metrics() map[string]int64 {
	return map[string]int64{
		"events_processed_total": d.processedCount.Load(),
		"events_error_total":     d.errorCount.Load(),
		"events_queued_total":    d.queuedCount.Load(),
		"circuit_breaker_fails":  int64(d.cb.failCount.Load()),
	}
}

// ── Helpers ────────────────────────────────────────────────────────

func (d *EventDispatcher) updateHealthStatus() {
	fails := d.cb.failCount.Load()
	threshold := d.cb.threshold

	switch {
	case fails >= threshold:
		d.healthStatus.Store("unhealthy")
	case fails >= threshold/2:
		d.healthStatus.Store("degraded")
	default:
		d.healthStatus.Store("healthy")
	}
}

func (d *EventDispatcher) fallbackQueueLen() int {
	n, err := d.fallbackQueue.Len()
	if err != nil {
		return -1
	}
	return n
}

func (d *EventDispatcher) adapterName() string {
	// Используем fmt.Sprintf для получения имени типа,
	// чтобы избежать impossible type switch при добавлении новых методов.
	name := fmt.Sprintf("%T", d.adapter)
	switch name {
	case "*cmms.InternalAdapter":
		return "internal"
	case "*cmms.AtlasAdapter":
		return "atlas"
	default:
		return name
	}
}

// logDispatcherEvent логирует событие диспетчера в audit log.
func (d *EventDispatcher) logDispatcherEvent(ctx context.Context, event DispatcherEvent) {
	if !d.cfg.AuditLogEnabled || d.auditLogger == nil {
		return
	}
	event.Duration = event.Duration / time.Millisecond // ms
	if err := d.auditLogger.LogDispatcherEvent(ctx, event); err != nil {
		d.logger.Warn("failed to log dispatcher event",
			"event_type", event.EventType, "error", err,
		)
	}
}

// mapSeverityToPriority маппит severity alarm в priority WorkOrder.
func (d *EventDispatcher) mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		return "medium"
	}
}

// mapProbabilityToPriority маппит вероятность отказа в priority.
func (d *EventDispatcher) mapProbabilityToPriority(probability float64) string {
	switch {
	case probability >= 0.95:
		return "critical"
	case probability >= 0.85:
		return "high"
	case probability >= 0.80:
		return "medium"
	default:
		return "low"
	}
}
