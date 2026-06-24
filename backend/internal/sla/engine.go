// Package sla — SLA Calculation Engine.
//
// SLA-6.2.1: SLA Calculation Service — рассчитывает SLA дедлайны с учётом
// Business Calendar и Pause Rules.
//
// Алгоритм:
//  1. Старт таймера при создании WO (или при первом назначении)
//  2. Расчёт дедлайна: created_at + resolution_time (только рабочие часы)
//  3. Пауза таймера при ON_HOLD / AWAITING_* статусах
//  4. Возобновление при возврате в IN_PROGRESS
//  5. Проверка SLA статуса: on_track, at_risk, breached
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability)
//   - ISO 27001 A.12.6.1 (Capacity management)
package sla

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// BreachedWorkOrder (SLA-6.2.3)
// ═══════════════════════════════════════════════════════════════════════

// BreachedWorkOrder — информация о просроченном Work Order для алерта.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — SLA breach events)
//   - IEC 62443 SR 2.8 (Audit events — breach tracking)
//   - OWASP ASVS V7.1 (Log content — structured breach data)
type BreachedWorkOrder struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	DeviceID     string    `json:"device_id"`
	DeviceName   string    `json:"device_name"`
	Priority     string    `json:"priority"`
	SLADeadline  time.Time `json:"sla_deadline"`
	AssignedTo   string    `json:"assigned_to"`
	AssigneeName string    `json:"assignee_name"`
}

// BreachedWorkOrderFinder — интерфейс для поиска просроченных Work Orders в БД.
//
// Реализуется внешним репозиторием (db.WorkOrderRepository).
// Compliance: SQL injection prevention через parameterized queries.
type BreachedWorkOrderFinder interface {
	// FindBreachedWorkOrders возвращает Work Orders у которых sla_deadline < NOW()
	// и статус НЕ в {completed, cancelled, closed, rejected}.
	FindBreachedWorkOrders(ctx context.Context) ([]BreachedWorkOrder, error)
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.2.2: Escalation Rule Resolver
// ═══════════════════════════════════════════════════════════════════════

// EscalationRuleResolver — интерфейс для получения правил эскалации и
// логирования эскалаций в БД.
//
// Реализуется db.DB (db.GetEscalationRules, db.LogEscalation).
// Использует интерфейс вместо прямого *db.DB для:
//   - Предотвращения циклических зависимостей (sla → db → sla)
//   - Возможности тестирования с моками
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Audit trail — escalation logging)
//   - IEC 62443 SR 2.8 (Audit events)
//   - OWASP ASVS V7.1 (Structured logging)
type EscalationRuleResolver interface {
	// GetEscalationRules возвращает правила эскалации для приоритета и времени после дедлайна.
	GetEscalationRules(ctx context.Context, priority string, breachMinutes int) ([]EscalationRule, error)
	// LogEscalation записывает событие эскалации в audit log.
	LogEscalation(ctx context.Context, entry *EscalationLogEntry) error
	// GetActiveEscalations возвращает активные (неподтверждённые) эскалации для WO.
	GetActiveEscalations(ctx context.Context, workOrderID string) ([]EscalationLogEntry, error)
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Status
// ═══════════════════════════════════════════════════════════════════════

// SLAStatus — статус SLA для Work Order.
type SLAStatus string

const (
	SLAOnTrack  SLAStatus = "on_track"
	SLAAtRisk   SLAStatus = "at_risk"
	SLABreached SLAStatus = "breached"
	SLAExempt   SLAStatus = "exempt" // SLA не применяется
	SLAPaused   SLAStatus = "paused" // таймер на паузе
)

// EscalationLevel — уровень эскалации.
type EscalationLevel int

const (
	EscalationNone EscalationLevel = 0
	EscalationL1   EscalationLevel = 1
	EscalationL2   EscalationLevel = 2
	EscalationL3   EscalationLevel = 3
)

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.2.2: Escalation Rule & Log Types
// ═══════════════════════════════════════════════════════════════════════

// EscalationRule — правило эскалации для SLA breach.
//
// Используется интерфейсом EscalationRuleResolver для передачи данных
// из БД в SLA engine без циклических зависимостей.
type EscalationRule struct {
	ID                   string `json:"id"`
	Priority             string `json:"priority"`
	EscalationLevel      int    `json:"escalation_level"`
	BreachMinutes        int    `json:"breach_minutes"`
	NotifyRole           string `json:"notify_role"`
	NotifyChannel        string `json:"notify_channel"`
	RepeatIntervalMinutes int   `json:"repeat_interval_minutes"`
}

// EscalationLogEntry — запись в журнале эскалации.
type EscalationLogEntry struct {
	ID               string     `json:"id"`
	WorkOrderID      string     `json:"work_order_id"`
	EscalationLevel  int        `json:"escalation_level"`
	RuleID           string     `json:"rule_id"`
	NotifiedAt       time.Time  `json:"notified_at"`
	AcknowledgedAt   *time.Time `json:"acknowledged_at,omitempty"`
	AcknowledgedBy   *string    `json:"acknowledged_by,omitempty"`
	ResolutionNotes  string     `json:"resolution_notes,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// SLATracker — отслеживает SLA для одного Work Order.
// ═══════════════════════════════════════════════════════════════════════

// SLATrackerState — полное состояние SLA трекера для Work Order.
type SLATrackerState struct {
	WorkOrderID string `json:"work_order_id"`
	PolicyID    string `json:"policy_id"`
	Priority    string `json:"priority"`
	Impact      string `json:"impact"`

	CreatedAt   time.Time  `json:"created_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty"`

	// SLA Targets (в рабочих минутах)
	ResponseTargetMinutes   int `json:"response_target_minutes"`
	ResolutionTargetMinutes int `json:"resolution_target_minutes"`

	// Deadline (абсолютное время)
	ResponseDeadline   *time.Time `json:"response_deadline,omitempty"`
	ResolutionDeadline *time.Time `json:"resolution_deadline,omitempty"`

	// Текущий статус
	Status     SLAStatus       `json:"status"`
	Escalation EscalationLevel `json:"escalation_level"`

	// Пауза
	IsPaused     bool       `json:"is_paused"`
	PausedSince  *time.Time `json:"paused_since,omitempty"`
	TotalPauseMs int64      `json:"total_pause_ms"`

	// Прогресс
	ElapsedWorkSeconds   int64   `json:"elapsed_work_seconds"`
	RemainingWorkSeconds int64   `json:"remaining_work_seconds"`
	ProgressPercent      float64 `json:"progress_percent"`
}

// SLACalculationEngine — главный движок расчёта SLA.
//
// Использует:
//   - SLAPolicy для базовых параметров
//   - SLAMatrixEntry для точных таргетов (Priority × Impact)
//   - BusinessCalendar для учёта рабочих часов
//   - SLAPauseRule для определения пауз
type SLACalculationEngine struct {
	mu     sync.RWMutex
	logger *slog.Logger

	trackers       map[string]*SLATrackerState  // work_order_id → tracker
	policies       map[string]*SLAPolicy        // policy_id → policy
	calendars      map[string]*BusinessCalendar // site_id → calendar
	matrix         map[string][]*SLAMatrixEntry // policy_id → entries
	pauseRules     map[string][]*SLAPauseRule   // policy_id → rules
	breachedFinder BreachedWorkOrderFinder      // SLA-6.2.3: DB finder for breached WOs

	// SLA-6.2.2: Escalation
	escalationResolver EscalationRuleResolver // resolver для правил эскалации
}

// NewEngine создаёт SLA Calculation Engine.
func NewEngine(logger *slog.Logger) *SLACalculationEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &SLACalculationEngine{
		logger:     logger.With("component", "sla-engine"),
		trackers:   make(map[string]*SLATrackerState),
		policies:   make(map[string]*SLAPolicy),
		calendars:  make(map[string]*BusinessCalendar),
		matrix:     make(map[string][]*SLAMatrixEntry),
		pauseRules: make(map[string][]*SLAPauseRule),
	}
}

// ── Escalation Management (SLA-6.2.2) ───────────────────────────────

// SetEscalationResolver регистрирует резолвер правил эскалации.
//
// Реализуется db.DB. Используется для получения правил эскалации
// и логирования эскалаций при SLA breach.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — escalation audit trail)
//   - IEC 62443 SR 2.8 (Audit events)
func (e *SLACalculationEngine) SetEscalationResolver(resolver EscalationRuleResolver) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.escalationResolver = resolver
	e.logger.Info("escalation rule resolver registered")
}

// CheckEscalation проверяет и выполняет эскалацию для просроченного Work Order.
//
// Алгоритм:
//  1. Получает правила эскалации для priority и breach_minutes
//  2. Для каждого правила проверяет, была ли уже выполнена эскалация этого уровня
//  3. Если не была — логирует эскалацию в БД
//  4. Возвращает список выполненных эскалаций
//
// Параметры:
//   - ctx: контекст
//   - woID: ID Work Order
//   - priority: приоритет (critical, high, medium, low)
//   - breachedSince: время, прошедшее с момента breach
//
// Returns:
//   - []EscalationRule: выполненные правила эскалации
//   - error: ошибка выполнения
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging)
//   - IEC 62443 SR 2.8 (Audit events)
//   - OWASP ASVS V7.1 (Structured logging)
//   - СТБ 34.101.27 (Audit trail)
func (e *SLACalculationEngine) CheckEscalation(ctx context.Context, woID, priority string, breachedSince time.Duration) ([]EscalationRule, error) {
	e.mu.RLock()
	resolver := e.escalationResolver
	e.mu.RUnlock()

	if resolver == nil {
		e.logger.Warn("escalation resolver not set, skipping escalation check",
			"work_order", woID,
		)
		return nil, nil
	}

	breachMinutes := int(breachedSince.Minutes())

	// 1. Получаем правила эскалации
	rules, err := resolver.GetEscalationRules(ctx, priority, breachMinutes)
	if err != nil {
		return nil, fmt.Errorf("get escalation rules: %w", err)
	}

	if len(rules) == 0 {
		return nil, nil
	}

	// 2. Получаем уже выполненные эскалации для этого WO
	activeEscalations, err := resolver.GetActiveEscalations(ctx, woID)
	if err != nil {
		e.logger.Warn("failed to get active escalations",
			"work_order", woID, "error", err,
		)
		// Продолжаем — логируем эскалации даже при ошибке получения активных
	}

	// 3. Создаём set уже выполненных уровней
	executedLevels := make(map[int]bool)
	for _, ae := range activeEscalations {
		executedLevels[ae.EscalationLevel] = true
	}

	// 4. Выполняем новые эскалации
	var executed []EscalationRule
	now := time.Now().UTC()

	for _, rule := range rules {
		if executedLevels[rule.EscalationLevel] {
			// Эскалация этого уровня уже выполнена
			continue
		}

		// Логируем эскалацию
		entry := &EscalationLogEntry{
			WorkOrderID:     woID,
			EscalationLevel: rule.EscalationLevel,
			RuleID:          rule.ID,
			NotifiedAt:      now,
		}

		if err := resolver.LogEscalation(ctx, entry); err != nil {
			e.logger.Error("failed to log escalation",
				"work_order", woID,
				"escalation_level", rule.EscalationLevel,
				"error", err,
			)
			continue
		}

		executed = append(executed, rule)
		executedLevels[rule.EscalationLevel] = true

		e.logger.Warn("escalation triggered",
			"work_order", woID,
			"priority", priority,
			"escalation_level", rule.EscalationLevel,
			"notify_role", rule.NotifyRole,
			"notify_channel", rule.NotifyChannel,
			"breach_minutes", breachMinutes,
			"component", "sla-engine",
		)
	}

	return executed, nil
}

// ── Policy management ────────────────────────────────────────────────

// SetPolicy регистрирует SLA политику.
func (e *SLACalculationEngine) SetPolicy(policy *SLAPolicy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies[policy.ID] = policy
}

// SetCalendar регистрирует Business Calendar для сайта.
func (e *SLACalculationEngine) SetCalendar(siteID string, cal *BusinessCalendar) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calendars[siteID] = cal
}

// SetMatrix регистрирует матрицу SLA для политики.
func (e *SLACalculationEngine) SetMatrix(policyID string, entries []*SLAMatrixEntry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.matrix[policyID] = entries
}

// SetPauseRules регистрирует правила паузы для политики.
func (e *SLACalculationEngine) SetPauseRules(policyID string, rules []*SLAPauseRule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.pauseRules[policyID] = rules
}

// ── Core SLA calculation ─────────────────────────────────────────────

// StartTracking начинает отслеживание SLA для Work Order.
//
// Рассчитывает дедлайны на основе:
//   - Priority × Impact → матрица SLA
//   - Business Calendar сайта
//
// Returns: начальное состояние трекера
func (e *SLACalculationEngine) StartTracking(ctx context.Context, woID, policyID, priority, impact, siteID string) (*SLATrackerState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Получаем политику
	policy, ok := e.policies[policyID]
	if !ok {
		return nil, fmt.Errorf("sla policy %s not found", policyID)
	}

	// Получаем таргеты из матрицы
	respTarget, resTarget := e.resolveTargets(policyID, priority, impact)
	if respTarget <= 0 {
		respTarget = policy.ResponseTimeMinutes
	}
	if resTarget <= 0 {
		resTarget = policy.ResolutionTimeMinutes
	}

	// Получаем календарь
	cal, _ := e.calendars[siteID]

	now := time.Now().UTC()

	// Рассчитываем дедлайны (в рабочих часах)
	var respDeadline, resDeadline *time.Time
	if cal != nil {
		rd := calculateDeadline(now, respTarget, cal)
		respDeadline = &rd
		rd2 := calculateDeadline(now, resTarget, cal)
		resDeadline = &rd2
	} else {
		// Без календаря — используем астрономическое время
		rd := now.Add(time.Duration(respTarget) * time.Minute)
		respDeadline = &rd
		rd2 := now.Add(time.Duration(resTarget) * time.Minute)
		resDeadline = &rd2
	}

	tracker := &SLATrackerState{
		WorkOrderID:             woID,
		PolicyID:                policyID,
		Priority:                priority,
		Impact:                  impact,
		CreatedAt:               now,
		ResponseTargetMinutes:   respTarget,
		ResolutionTargetMinutes: resTarget,
		ResponseDeadline:        respDeadline,
		ResolutionDeadline:      resDeadline,
		Status:                  SLAOnTrack,
	}

	e.trackers[woID] = tracker

	e.logger.Info("sla tracking started",
		"work_order", woID,
		"policy", policyID,
		"priority", priority,
		"resp_deadline", respDeadline,
		"res_deadline", resDeadline,
	)

	return tracker, nil
}

// UpdateStatus обновляет статус Work Order и пересчитывает SLA.
//
// Возвращает обновлённое состояние трекера.
func (e *SLACalculationEngine) UpdateStatus(ctx context.Context, woID, newStatus string) (*SLATrackerState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	tracker, ok := e.trackers[woID]
	if !ok {
		return nil, fmt.Errorf("sla tracker for %s not found", woID)
	}

	// Получаем правила паузы
	rules := e.pauseRules[tracker.PolicyID]
	now := time.Now().UTC()

	// Проверка паузы
	if IsPausedStatus(newStatus, rules) && !tracker.IsPaused {
		// Ставим на паузу
		tracker.IsPaused = true
		tracker.PausedSince = &now
		tracker.Status = SLAPaused
		e.logger.Info("sla paused", "work_order", woID, "status", newStatus)
	} else if !IsPausedStatus(newStatus, rules) && tracker.IsPaused {
		// Снимаем с паузы
		if tracker.PausedSince != nil {
			pauseDuration := now.Sub(*tracker.PausedSince)
			tracker.TotalPauseMs += pauseDuration.Milliseconds()
		}
		tracker.IsPaused = false
		tracker.PausedSince = nil
		e.logger.Info("sla resumed", "work_order", woID, "paused_ms", tracker.TotalPauseMs)
	}

	// Пересчитываем статус
	e.recalculate(tracker, now)

	return tracker, nil
}

// CompleteWorkOrder завершает отслеживание SLA.
func (e *SLACalculationEngine) CompleteWorkOrder(ctx context.Context, woID string) (*SLATrackerState, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	tracker, ok := e.trackers[woID]
	if !ok {
		return nil, fmt.Errorf("sla tracker for %s not found", woID)
	}

	now := time.Now().UTC()
	elapsed := now.Sub(tracker.CreatedAt).Seconds() - float64(tracker.TotalPauseMs)/1000.0

	// Проверка breach
	if tracker.ResolutionDeadline != nil && now.After(*tracker.ResolutionDeadline) {
		tracker.Status = SLABreached
	} else {
		tracker.Status = SLAOnTrack
	}

	tracker.ElapsedWorkSeconds = int64(elapsed)

	e.logger.Info("sla completed",
		"work_order", woID,
		"status", tracker.Status,
		"elapsed_seconds", elapsed,
	)

	return tracker, nil
}

// ── Query methods ────────────────────────────────────────────────────

// GetTracker возвращает состояние SLA трекера для Work Order.
func (e *SLACalculationEngine) GetTracker(woID string) (*SLATrackerState, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.trackers[woID]
	return t, ok
}

// SetBreachedFinder регистрирует finder для поиска просроченных Work Orders в БД.
//
// SLA-6.2.3: Необходим для checkBreachedSLAs в worker.
// Compliance:
//   - ISO 27001 A.12.4.1 — structured logging of SLA breaches
//   - IEC 62443 SR 2.8 — audit trail for breached work orders
func (e *SLACalculationEngine) SetBreachedFinder(finder BreachedWorkOrderFinder) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.breachedFinder = finder
	e.logger.Info("breached work order finder registered")
}

// FindBreachedWorkOrders находит Work Orders с просроченным SLA.
//
// Выполняет поиск через BreachedWorkOrderFinder (БД).
// Возвращает структурированные данные для алертов.
//
// SLA-6.2.3: Используется SLA Worker для отправки уведомлений.
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging)
//   - IEC 62443 SR 2.8 (Audit events)
//   - OWASP ASVS V7.1 (Log content — не разглашает sensitive data)
func (e *SLACalculationEngine) FindBreachedWorkOrders(ctx context.Context) ([]BreachedWorkOrder, error) {
	e.mu.RLock()
	finder := e.breachedFinder
	e.mu.RUnlock()

	if finder == nil {
		return nil, fmt.Errorf("breached work order finder not set: call SetBreachedFinder first")
	}

	breached, err := finder.FindBreachedWorkOrders(ctx)
	if err != nil {
		e.logger.Error("failed to find breached work orders",
			"error", err,
		)
		return nil, fmt.Errorf("find breached work orders: %w", err)
	}

	if len(breached) > 0 {
		e.logger.Warn("breached work orders detected",
			"count", len(breached),
			"component", "sla-engine",
		)
	}

	return breached, nil
}

// GetBreached возвращает все Work Orders с нарушением SLA.
func (e *SLACalculationEngine) GetBreached() []*SLATrackerState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*SLATrackerState, 0)
	for _, t := range e.trackers {
		if t.Status == SLABreached {
			result = append(result, t)
		}
	}
	return result
}

// GetAtRisk возвращает Work Orders под риском нарушения SLA.
func (e *SLACalculationEngine) GetAtRisk() []*SLATrackerState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*SLATrackerState, 0)
	for _, t := range e.trackers {
		if t.Status == SLAAtRisk {
			result = append(result, t)
		}
	}
	return result
}

// ── Internal ─────────────────────────────────────────────────────────

// resolveTargets находит таргеты из матрицы SLA.
func (e *SLACalculationEngine) resolveTargets(policyID, priority, impact string) (responseMin, resolutionMin int) {
	entries, ok := e.matrix[policyID]
	if !ok {
		return 0, 0
	}

	for _, entry := range entries {
		if entry.Priority == priority && string(entry.Impact) == impact {
			return entry.ResponseTimeMinutes, entry.ResolutionTimeMinutes
		}
	}
	return 0, 0
}

// recalculate пересчитывает SLA статус трекера.
func (e *SLACalculationEngine) recalculate(tracker *SLATrackerState, now time.Time) {
	if tracker.IsPaused {
		tracker.Status = SLAPaused
		return
	}

	elapsedSeconds := now.Sub(tracker.CreatedAt).Seconds() - float64(tracker.TotalPauseMs)/1000.0
	tracker.ElapsedWorkSeconds = int64(elapsedSeconds)

	totalTargetSeconds := float64(tracker.ResolutionTargetMinutes) * 60
	remainingSeconds := totalTargetSeconds - elapsedSeconds

	if remainingSeconds <= 0 {
		tracker.Status = SLABreached
		tracker.RemainingWorkSeconds = 0
		tracker.ProgressPercent = 100
		return
	}

	tracker.RemainingWorkSeconds = int64(remainingSeconds)
	tracker.ProgressPercent = math.Min(elapsedSeconds/totalTargetSeconds*100, 99.9)

	// Определяем статус
	if remainingSeconds < totalTargetSeconds*0.1 {
		tracker.Status = SLAAtRisk
		tracker.Escalation = EscalationL3
	} else if remainingSeconds < totalTargetSeconds*0.25 {
		tracker.Status = SLAAtRisk
		tracker.Escalation = EscalationL2
	} else if remainingSeconds < totalTargetSeconds*0.5 {
		tracker.Status = SLAAtRisk
		tracker.Escalation = EscalationL1
	} else {
		tracker.Status = SLAOnTrack
		tracker.Escalation = EscalationNone
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Deadline Calculator
// ═══════════════════════════════════════════════════════════════════════

// calculateDeadline рассчитывает абсолютное время дедлайна с учётом
// Business Calendar (только рабочие часы).
//
// Алгоритм:
//  1. Начинаем с now
//  2. Для каждой минуты таргета: двигаемся вперёд на 1 минуту
//  3. Если текущее время не рабочее — двигаемся к следующему рабочему
//  4. Возвращаем конечное время
func calculateDeadline(from time.Time, targetMinutes int, cal *BusinessCalendar) time.Time {
	current := from
	remaining := targetMinutes

	for remaining > 0 {
		if cal.IsWorkHour(current) {
			// Рабочее время — тикаем
			remaining--
			current = current.Add(time.Minute)
		} else {
			// Нерабочее время — прыгаем к следующему рабочему
			next := cal.NextWorkStart(current)
			current = next
		}
	}

	return current
}
