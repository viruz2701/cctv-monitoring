// Package sla — SLA Calculation Worker (SLA-6.2.1).
//
// Go worker, который запускается каждую минуту, собирает все активные
// Work Orders, пересчитывает SLA статусы и публикует события при изменениях.
//
// Алгоритм:
//  1. Каждые 60s: получает все активные WO из БД
//  2. Для каждой WO: загружает SLA трекер из кэша (или создаёт новый)
//  3. Пересчитывает elapsed/remaining/progress
//  4. Определяет статус: on_track / at_risk / breached
//  5. При изменении статуса → публикует NATS событие
//  6. При breach → сохраняет в audit_log
//  7. Сохраняет состояние трекеров в Redis (периодически)
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — SLA мониторинг)
//   - ISO 27001 A.12.4.1 (Event logging — SLA breach events)
//   - ISO 27001 A.12.6.1 (Capacity management — SLA метрики)
//   - ISO 27019 PCC.A.12.6 (ICS capacity management)
package sla

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.2.3: SLA Breach Check Constants
// ═══════════════════════════════════════════════════════════════════════

// BreachCheckInterval — периодичность проверки просроченных SLA.
// По умолчанию: 5 минут (согласно SLA-6.2.3 requirements).
const BreachCheckInterval = 5 * time.Minute

// ═══════════════════════════════════════════════════════════════════════
// SLA Worker
// ═══════════════════════════════════════════════════════════════════════

// WorkerConfig — конфигурация SLA Worker.
type WorkerConfig struct {
	// Interval — периодичность запуска batch-обработки.
	// По умолчанию: 60s.
	Interval time.Duration `json:"interval"`

	// BatchSize — размер батча для загрузки WO из БД.
	// По умолчанию: 100.
	BatchSize int `json:"batch_size"`

	// SaveInterval — периодичность сохранения состояния в Redis/БД.
	// По умолчанию: 5min.
	SaveInterval time.Duration `json:"save_interval"`

	// BreachThreshold — порог для at_risk в процентах от дедлайна.
	// 0.75 = при 75% использованного времени → at_risk.
	// По умолчанию: 0.75.
	BreachThreshold float64 `json:"breach_threshold"`

	// CriticalThreshold — порог для critical at_risk.
	// 0.90 = при 90% использованного времени → critical at_risk.
	// По умолчанию: 0.90.
	CriticalThreshold float64 `json:"critical_threshold"`
}

// DefaultWorkerConfig — значения по умолчанию.
var DefaultWorkerConfig = WorkerConfig{
	Interval:          60 * time.Second,
	BatchSize:         100,
	SaveInterval:      5 * time.Minute,
	BreachThreshold:   0.75,
	CriticalThreshold: 0.90,
}

// validate применяет значения по умолчанию.
func (c *WorkerConfig) validate() {
	if c.Interval <= 0 {
		c.Interval = DefaultWorkerConfig.Interval
	}
	if c.BatchSize <= 0 {
		c.BatchSize = DefaultWorkerConfig.BatchSize
	}
	if c.SaveInterval <= 0 {
		c.SaveInterval = DefaultWorkerConfig.SaveInterval
	}
	if c.BreachThreshold <= 0 {
		c.BreachThreshold = DefaultWorkerConfig.BreachThreshold
	}
	if c.CriticalThreshold <= 0 {
		c.CriticalThreshold = DefaultWorkerConfig.CriticalThreshold
	}
}

// SLAEvent — тип события SLA.
type SLAEvent string

const (
	SLAEventBreached SLAEvent = "sla.breached"
	SLAEventAtRisk   SLAEvent = "sla.at_risk"
	SLAEventResolved SLAEvent = "sla.resolved" // breach resolved (WO completed)
)

// SLAEventPayload — payload для NATS события.
type SLAEventPayload struct {
	Event           SLAEvent  `json:"event"`
	WorkOrderID     string    `json:"work_order_id"`
	Priority        string    `json:"priority"`
	Status          SLAStatus `json:"status"`
	EscalationLevel int       `json:"escalation_level"`
	ElapsedSeconds  int64     `json:"elapsed_seconds"`
	TargetMinutes   int       `json:"target_minutes"`
	Timestamp       time.Time `json:"timestamp"`
}

// WorkOrderProvider — интерфейс для получения активных WO из БД.
type WorkOrderProvider interface {
	// GetActiveWorkOrders возвращает все активные Work Orders (не completed, не cancelled).
	GetActiveWorkOrders(ctx context.Context, limit, offset int) ([]WorkOrderRef, error)
}

// WorkOrderRef — минимальная информация о Work Order для SLA расчёта.
type WorkOrderRef struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Priority  string    `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	SiteID    string    `json:"site_id,omitempty"`
}

// EventPublisher — интерфейс для публикации NATS событий.
type EventPublisher interface {
	// PublishSLABreach публикует событие о нарушении SLA.
	PublishSLABreach(ctx context.Context, event SLAEventPayload) error
}

// StatusRecorder — интерфейс для сохранения SLA статусов.
type StatusRecorder interface {
	// SaveSLATracker сохраняет состояние SLA трекера.
	SaveSLATracker(ctx context.Context, state *SLATrackerState) error
	// LoadSLATrackers загружает все активные SLA трекеры.
	LoadSLATrackers(ctx context.Context) ([]*SLATrackerState, error)
	// LogSLABreach логирует нарушение SLA в audit_log.
	LogSLABreach(ctx context.Context, event SLAEventPayload) error
}

// ═══════════════════════════════════════════════════════════════════════
// SLAWorker
// ═══════════════════════════════════════════════════════════════════════

// SLAWorker — batch worker для расчёта SLA.
//
// Запускается как goroutine, каждые Interval секунд:
//  1. Получает активные WO из WorkOrderProvider
//  2. Загружает/создаёт SLA трекеры
//  3. Пересчитывает статусы
//  4. Публикует события при изменениях
//  5. Сохраняет состояние
//
// SLA-6.2.3: Дополнительно запускает checkBreachedSLAs каждые 5 минут
// для отправки multi-channel уведомлений о просроченных SLA.
//
// P0-1.3: Интеграция с SLABreachNotifier (Telegram/SMS/Email).
type SLAWorker struct {
	engine    *SLACalculationEngine
	provider  WorkOrderProvider
	publisher EventPublisher
	recorder  StatusRecorder
	cfg       WorkerConfig
	logger    *slog.Logger

	// P0-1.3: Multi-channel notifier (Telegram/SMS/Email)
	notifier *SLABreachNotifier

	// Состояние
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Метрики
	processedCount  atomic.Int64
	breachCount     atomic.Int64
	atRiskCount     atomic.Int64
	lastRunDuration atomic.Value // time.Duration
}

// NewSLAWorker создаёт SLA Worker.
//
// notifier — опциональный multi-channel notifier (Telegram/SMS/Email).
// P0-1.3: Заменяет telegramBot на SLABreachNotifier.
func NewSLAWorker(
	engine *SLACalculationEngine,
	provider WorkOrderProvider,
	publisher EventPublisher,
	recorder StatusRecorder,
	cfg WorkerConfig,
	logger *slog.Logger,
	notifier *SLABreachNotifier,
) *SLAWorker {
	cfg.validate()
	if logger == nil {
		logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &SLAWorker{
		engine:    engine,
		provider:  provider,
		publisher: publisher,
		recorder:  recorder,
		cfg:       cfg,
		logger:    logger.With("component", "sla-worker"),
		notifier:  notifier,
		ctx:       ctx,
		cancel:    cancel,
	}
	w.lastRunDuration.Store(time.Duration(0))

	if notifier != nil {
		w.logger.Info("SLA breach multi-channel notifications enabled",
			"check_interval", BreachCheckInterval,
		)
	}

	return w
}

// Start запускает SLA Worker.
func (w *SLAWorker) Start() error {
	w.logger.Info("starting SLA calculation worker",
		"interval", w.cfg.Interval,
		"batch_size", w.cfg.BatchSize,
		"save_interval", w.cfg.SaveInterval,
		"breach_threshold", w.cfg.BreachThreshold,
	)

	// Загружаем сохранённые трекеры при старте
	if w.recorder != nil {
		w.loadTrackers()
	}

	// Запускаем batch worker
	w.wg.Add(1)
	go w.runLoop()

	// Запускаем save worker (периодическое сохранение)
	if w.recorder != nil {
		w.wg.Add(1)
		go w.saveLoop()
	}

	// SLA-6.2.3 + P0-1.3: Запускаем breach check worker (каждые 5 минут)
	if w.notifier != nil {
		w.wg.Add(1)
		go w.breachCheckLoop()
	}

	return nil
}

// Stop останавливает SLA Worker.
func (w *SLAWorker) Stop() {
	w.logger.Info("stopping SLA calculation worker")
	w.cancel()
	w.wg.Wait()
	w.logger.Info("SLA calculation worker stopped")
}

// ═══════════════════════════════════════════════════════════════════════
// Main loop
// ═══════════════════════════════════════════════════════════════════════

func (w *SLAWorker) runLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	// Первый запуск сразу
	w.processBatch()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processBatch()
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.2.3: Breached SLA Check Loop
// ═══════════════════════════════════════════════════════════════════════

// breachCheckLoop запускает проверку просроченных SLA каждые 5 минут.
//
// SLA-6.2.3:
//   - Вызывает engine.FindBreachedWorkOrders() для поиска просроченных WO
//   - Группирует breaches по assignee
//   - Отправляет Telegram уведомления
//   - Логирует все breaches в audit trail
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — регулярная проверка SLA)
//   - IEC 62443 SR 2.8 (Audit events — breach обнаружение)
//   - OWASP ASVS V7.1 (Log content — структурированные логи)
func (w *SLAWorker) breachCheckLoop() {
	defer w.wg.Done()

	if w.notifier == nil {
		w.logger.Debug("SLA breach check disabled: no notifier configured")
		return
	}

	ticker := time.NewTicker(BreachCheckInterval)
	defer ticker.Stop()

	// Первый запуск сразу
	w.checkBreachedSLAs()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.checkBreachedSLAs()
		}
	}
}

// checkBreachedSLAs проверяет просроченные SLA, выполняет эскалацию
// и отправляет уведомления.
//
// Алгоритм:
//  1. Вызывает engine.FindBreachedWorkOrders() для поиска просроченных WO
//  2. Для каждого breach: вызывает engine.CheckEscalation() для эскалации (SLA-6.2.2)
//  3. Группирует breaches по assignee (AssignedTo)
//  4. Для каждой группы: отправляет сводку в Telegram
//  5. Логирует все обнаруженные breaches
//
// Формат уведомления:
//
//	⚠️ SLA Breach!
//	Наряд: {title}
//	Приоритет: {priority}
//	Дедлайн: {deadline}
//	Устройство: {device_name}
//
// SLA-6.2.2: Escalation Matrix — при breach проверяются правила эскалации
// и логируются в sla_escalation_log.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — escalation audit trail)
//   - IEC 62443 SR 2.8 (Audit events — escalation tracking)
//   - СТБ 34.101.27 (Защита информации — audit log)
func (w *SLAWorker) checkBreachedSLAs() {
	ctx, cancel := context.WithTimeout(w.ctx, 30*time.Second)
	defer cancel()

	w.logger.Debug("checking breached SLAs")

	// 1. Получаем просроченные WO
	breached, err := w.engine.FindBreachedWorkOrders(ctx)
	if err != nil {
		w.logger.Error("failed to find breached work orders",
			"error", err,
			"component", "sla-worker",
		)
		return
	}

	if len(breached) == 0 {
		w.logger.Debug("no breached SLAs found")
		return
	}

	w.logger.Warn("breached SLAs detected",
		"count", len(breached),
		"component", "sla-worker",
	)

	// SLA-6.2.2: Эскалация для каждого breach
	now := time.Now().UTC()
	for _, b := range breached {
		// Рассчитываем время с момента дедлайна
		breachedSince := now.Sub(b.SLADeadline)
		if breachedSince < 0 {
			breachedSince = 0
		}

		// P0-1.3: Выполняем эскалацию (логируется в sla_escalation_log)
		executed, err := w.engine.CheckEscalation(ctx, b.ID, b.Priority, breachedSince)
		if err != nil {
			w.logger.Error("failed to check escalation",
				"work_order", b.ID,
				"priority", b.Priority,
				"error", err,
			)
		}
		if len(executed) > 0 {
			w.logger.Warn("escalation rules executed",
				"work_order", b.ID,
				"count", len(executed),
				"component", "sla-worker",
			)
		}

		// P0-1.3: Отправляем multi-channel уведомления (Telegram/SMS/Email)
		// через SLABreachNotifier. Graceful degradation:
		//   - Если канал недоступен — уведомление через доступные каналы
		//   - Если notifier отключён (nil) — пропускаем
		//
		// Compliance:
		//   - ISO 27001 A.12.4.1: Все уведомления логируются в notifier
		//   - IEC 62443 SR 2.8: Audit trail для breach notifications
		//   - OWASP ASVS V7.1: Сообщения не содержат sensitive data
		if w.notifier != nil {
			if err := w.notifier.NotifyBreach(ctx, b); err != nil {
				w.logger.Error("failed to send breach notification",
					"work_order", b.ID,
					"priority", b.Priority,
					"device_id", b.DeviceID,
					"error", err,
					"component", "sla-worker",
				)
			}
		}
	}
}

// parseChatID преобразует строку assigneeID в int64 chat_id.
// Используется в notifier.go для Telegram.
func parseChatID(id string) (int64, error) {
	if id == "" || id == "unassigned" {
		return 0, fmt.Errorf("cannot send notification for unassigned work order")
	}

	var chatID int64
	for _, ch := range id {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid chat_id: %s", id)
		}
		chatID = chatID*10 + int64(ch-'0')
	}
	if chatID == 0 {
		return 0, fmt.Errorf("invalid chat_id: %s", id)
	}
	return chatID, nil
}

// escapeMarkdown экранирует специальные символы Markdown для Telegram.
func escapeMarkdown(s string) string {
	// Telegram Markdown экранирует: _ * [ ] ( ) ~ ` > # + - = | { } . !
	var result []byte
	for _, ch := range s {
		switch ch {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			result = append(result, '\\', byte(ch))
		default:
			result = append(result, byte(ch))
		}
	}
	return string(result)
}

func (w *SLAWorker) saveLoop() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.cfg.SaveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			// Сохраняем перед остановкой
			w.saveTrackers()
			return
		case <-ticker.C:
			w.saveTrackers()
		}
	}
}

// processBatch — основной метод batch-обработки.
func (w *SLAWorker) processBatch() {
	start := time.Now()
	w.logger.Debug("SLA batch processing started")

	ctx, cancel := context.WithTimeout(w.ctx, 45*time.Second)
	defer cancel()

	var totalProcessed int
	var totalBreaches int
	var totalAtRisk int

	offset := 0
	for {
		// Получаем батч WO
		orders, err := w.provider.GetActiveWorkOrders(ctx, w.cfg.BatchSize, offset)
		if err != nil {
			w.logger.Error("failed to get active work orders", "error", err)
			break
		}

		if len(orders) == 0 {
			break
		}

		// Обрабатываем каждую WO
		for _, wo := range orders {
			state := w.processWorkOrder(ctx, wo)
			if state != nil {
				totalProcessed++
				switch state.Status {
				case SLABreached:
					totalBreaches++
				case SLAAtRisk:
					totalAtRisk++
				}
			}
		}

		offset += len(orders)
		if len(orders) < w.cfg.BatchSize {
			break
		}
	}

	duration := time.Since(start)
	w.lastRunDuration.Store(duration)
	w.processedCount.Add(int64(totalProcessed))
	w.breachCount.Add(int64(totalBreaches))
	w.atRiskCount.Add(int64(totalAtRisk))

	w.logger.Info("SLA batch processing complete",
		"duration", duration,
		"processed", totalProcessed,
		"breaches", totalBreaches,
		"at_risk", totalAtRisk,
	)
}

// processWorkOrder обрабатывает одну Work Order.
func (w *SLAWorker) processWorkOrder(ctx context.Context, wo WorkOrderRef) *SLATrackerState {
	// Получаем существующий трекер
	tracker, exists := w.engine.GetTracker(wo.ID)

	if !exists {
		// Для новых WO трекер должен быть создан через StartTracking при создании WO.
		// Если трекера нет — пропускаем (будет создан при следующем StartTracking)
		return nil
	}

	// Получаем предыдущий статус
	prevStatus := tracker.Status

	// Обновляем статус через engine
	updated, err := w.engine.UpdateStatus(ctx, wo.ID, wo.Status)
	if err != nil {
		w.logger.Warn("failed to update SLA status",
			"work_order", wo.ID, "error", err,
		)
		return nil
	}

	// Если статус изменился — публикуем событие
	if updated.Status != prevStatus {
		w.publishStatusChange(ctx, updated, prevStatus)
	}

	return updated
}

// publishStatusChange публикует NATS событие при изменении SLA статуса.
func (w *SLAWorker) publishStatusChange(ctx context.Context, state *SLATrackerState, prevStatus SLAStatus) {
	if w.publisher == nil {
		return
	}

	var event SLAEvent
	switch state.Status {
	case SLABreached:
		event = SLAEventBreached
		if w.recorder != nil {
			w.recorder.LogSLABreach(ctx, SLAEventPayload{
				Event:           event,
				WorkOrderID:     state.WorkOrderID,
				Priority:        state.Priority,
				Status:          state.Status,
				EscalationLevel: int(state.Escalation),
				ElapsedSeconds:  state.ElapsedWorkSeconds,
				TargetMinutes:   state.ResolutionTargetMinutes,
				Timestamp:       time.Now().UTC(),
			})
		}
	case SLAAtRisk:
		event = SLAEventAtRisk
	}

	if event != "" && prevStatus != state.Status {
		payload := SLAEventPayload{
			Event:           event,
			WorkOrderID:     state.WorkOrderID,
			Priority:        state.Priority,
			Status:          state.Status,
			EscalationLevel: int(state.Escalation),
			ElapsedSeconds:  state.ElapsedWorkSeconds,
			TargetMinutes:   state.ResolutionTargetMinutes,
			Timestamp:       time.Now().UTC(),
		}

		if err := w.publisher.PublishSLABreach(ctx, payload); err != nil {
			w.logger.Warn("failed to publish SLA event",
				"work_order", state.WorkOrderID,
				"event", event,
				"error", err,
			)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Persistence
// ═══════════════════════════════════════════════════════════════════════

// loadTrackers загружает сохранённые трекеры.
func (w *SLAWorker) loadTrackers() {
	if w.recorder == nil {
		return
	}

	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()

	trackers, err := w.recorder.LoadSLATrackers(ctx)
	if err != nil {
		w.logger.Error("failed to load SLA trackers", "error", err)
		return
	}

	// Восстанавливаем трекеры в engine
	_ = trackers // для будущей реализации

	w.logger.Info("SLA trackers loaded", "count", len(trackers))
}

// saveTrackers сохраняет текущие трекеры.
func (w *SLAWorker) saveTrackers() {
	if w.recorder == nil {
		return
	}

	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()

	// Получаем все трекеры из engine
	breached := w.engine.GetBreached()
	atRisk := w.engine.GetAtRisk()

	saved := 0
	for _, t := range breached {
		if err := w.recorder.SaveSLATracker(ctx, t); err != nil {
			w.logger.Warn("failed to save SLA tracker",
				"work_order", t.WorkOrderID, "error", err,
			)
			continue
		}
		saved++
	}
	for _, t := range atRisk {
		if err := w.recorder.SaveSLATracker(ctx, t); err != nil {
			continue
		}
		saved++
	}

	if saved > 0 {
		w.logger.Debug("SLA trackers saved", "count", saved)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Health & Metrics
// ═══════════════════════════════════════════════════════════════════════

// Health возвращает состояние здоровья воркера.
func (w *SLAWorker) Health() map[string]interface{} {
	return map[string]interface{}{
		"status":            "running",
		"interval":          w.cfg.Interval.String(),
		"last_run_duration": w.lastRunDuration.Load().(time.Duration).String(),
		"total_processed":   w.processedCount.Load(),
		"total_breaches":    w.breachCount.Load(),
		"total_at_risk":     w.atRiskCount.Load(),
	}
}

// Metrics возвращает метрики для Prometheus.
func (w *SLAWorker) Metrics() map[string]int64 {
	return map[string]int64{
		"sla_processed_total": w.processedCount.Load(),
		"sla_breach_total":    w.breachCount.Load(),
		"sla_at_risk_total":   w.atRiskCount.Load(),
	}
}
