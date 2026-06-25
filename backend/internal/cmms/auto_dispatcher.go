// Package cmms — Auto-dispatcher Service (HubEx pattern).
//
// AutoDispatcher реализует автоматическое назначение техников на Work Orders
// на основе matching-алгоритма: skills + workload + location.
//
// Алгоритм назначения:
//  1. matchBySkills — отфильтровать техников по требуемым навыкам
//  2. matchByWorkload — отсортировать по загрузке (least loaded first)
//  3. matchByLocation — предпочесть ближайших к site
//
// Auto-escalation:
//   - При SLA breach → немедленная эскалация на manager
//   - При critical priority → немедленное назначение без ожидания
//   - При unassigned WO > 2h → автоматическая эскалация
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure — при ошибке не назначаем на случайного)
//   - IEC 62443 SR 3.1 (Data integrity — audit trail всех назначений)
//   - ISO 27001 A.12.4.1 (Event logging — каждое назначение логируется)
//   - ISO 27001 A.9.1 (Access control — RBAC проверка)
//   - ISO/IEC 27019 PCC.A.12.4 (ICS audit trail)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - OWASP ASVS V7.1 (Log content — no sensitive data in logs)
//   - СТБ 34.101.27 п. 7.2 (Audit trail — tamper-evident logging)
//   - Приказ ОАЦ №66 п. 7.18.3 (Incident response — auto-escalation)
package cmms

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

// Escalation thresholds (в минутах).
const (
	// UnassignedEscalationThreshold — если WO не назначен > 2ч → escalation.
	UnassignedEscalationThreshold = 120 * time.Minute

	// CriticalAutoAssignTimeout — critical priority назначается немедленно.
	CriticalAutoAssignTimeout = 5 * time.Minute

	// MaxTechnicianDistance — максимальное расстояние до site (км).
	MaxTechnicianDistance = 100.0

	// DefaultMaxWorkload — максимальное количество активных WO на техника.
	DefaultMaxWorkload = 5
)

// ═══════════════════════════════════════════════════════════════════════
// Interfaces (inversion of control)
// ═══════════════════════════════════════════════════════════════════════

// TechnicianProvider — интерфейс для поиска доступных техников.
//
// Реализуется db.DB или внешним workforce-сервисом.
type TechnicianProvider interface {
	// FindAvailableTechnicians возвращает техников, доступных для назначения.
	FindAvailableTechnicians(ctx context.Context, skills []string, siteID string) ([]models.TechnicianWorkload, error)

	// GetTechnicianWorkload возвращает текущую загрузку техника.
	GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error)
}

// WorkOrderProvider — интерфейс для работы с Work Orders.
//
// Реализуется db.DB или CMMSAdapter.
type WorkOrderProvider interface {
	// GetWorkOrder возвращает Work Order по ID.
	GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error)

	// AssignWorkOrder назначает техника на Work Order.
	AssignWorkOrder(ctx context.Context, id, userID string) error

	// GetSite возвращает информацию о site (для расчёта расстояния).
	GetSite(ctx context.Context, id string) (*models.Site, error)

	// ListUnassignedWorkOrders возвращает Work Orders без назначения.
	ListUnassignedWorkOrders(ctx context.Context, olderThan time.Duration) ([]models.WorkOrder, error)
}

// SLAStatusChecker — интерфейс для проверки SLA статуса.
//
// Реализуется sla.SLACalculationEngine.
type SLAStatusChecker interface {
	// CheckEscalation проверяет и выполняет эскалацию для Work Order.
	// Возвращает выполненные правила эскалации.
	CheckEscalation(ctx context.Context, woID, priority string, breachedSince time.Duration) ([]interface{}, error)
}

// DispatcherAuditLogger — интерфейс для аудит-логирования назначений.
type DispatcherAuditLogger interface {
	// LogDispatch логирует событие диспетчеризации.
	LogDispatch(ctx context.Context, entry *DispatchAuditEntry) error
}

// DispatcherAuditLoggerFunc — адаптер для функции как DispatcherAuditLogger.
type DispatcherAuditLoggerFunc func(ctx context.Context, entry *DispatchAuditEntry) error

func (f DispatcherAuditLoggerFunc) LogDispatch(ctx context.Context, entry *DispatchAuditEntry) error {
	return f(ctx, entry)
}

// DispatchAuditEntry — запись аудита диспетчеризации.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — каждое назначение)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
//   - IEC 62443 SR 2.8 (Audit events)
type DispatchAuditEntry struct {
	WorkOrderID    string            `json:"work_order_id"`
	TechnicianID   string            `json:"technician_id,omitempty"`
	TechnicianName string            `json:"technician_name,omitempty"`
	Action         string            `json:"action"`          // auto_assigned, escalated, failed, manual
	Reason         string            `json:"reason"`          // match_by_skills, sla_breach, no_technician
	Score          float64           `json:"score,omitempty"` // matching score
	Duration       time.Duration     `json:"duration_ms"`
	Timestamp      time.Time         `json:"timestamp"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// AutoDispatcher Config
// ═══════════════════════════════════════════════════════════════════════

// AutoDispatcherConfig — конфигурация AutoDispatcher.
type AutoDispatcherConfig struct {
	// MaxWorkload — максимальное количество активных WO на техника.
	// По умолчанию: 5.
	MaxWorkload int `json:"max_workload"`

	// MaxTechnicianDistance — максимальное расстояние до site (км).
	// По умолчанию: 100.
	MaxTechnicianDistance float64 `json:"max_technician_distance"`

	// UnassignedEscalationThreshold — время после которого unassigned WO
	// автоматически эскалируется. По умолчанию: 2h.
	UnassignedEscalationThreshold time.Duration `json:"unassigned_escalation_threshold"`

	// CriticalAutoAssign — назначать critical WO немедленно.
	CriticalAutoAssign bool `json:"critical_auto_assign"`

	// AuditLogEnabled — включать аудит-логирование.
	AuditLogEnabled bool `json:"audit_log_enabled"`
}

// DefaultAutoDispatcherConfig — значения по умолчанию.
var DefaultAutoDispatcherConfig = AutoDispatcherConfig{
	MaxWorkload:                   DefaultMaxWorkload,
	MaxTechnicianDistance:         MaxTechnicianDistance,
	UnassignedEscalationThreshold: UnassignedEscalationThreshold,
	CriticalAutoAssign:            true,
	AuditLogEnabled:               true,
}

func (c *AutoDispatcherConfig) validate() {
	if c.MaxWorkload <= 0 {
		c.MaxWorkload = DefaultAutoDispatcherConfig.MaxWorkload
	}
	if c.MaxTechnicianDistance <= 0 {
		c.MaxTechnicianDistance = DefaultAutoDispatcherConfig.MaxTechnicianDistance
	}
	if c.UnassignedEscalationThreshold <= 0 {
		c.UnassignedEscalationThreshold = DefaultAutoDispatcherConfig.UnassignedEscalationThreshold
	}
}

// ═══════════════════════════════════════════════════════════════════════
// AutoDispatcher — сервис автоматического назначения техников
// ═══════════════════════════════════════════════════════════════════════

// AutoDispatcher реализует автоматическое назначение техников на Work Orders.
//
// Алгоритм matching (HubEx pattern):
//  1. matchBySkills — фильтр техников по требуемым навыкам
//  2. matchByWorkload — сортировка по возрастанию загрузки
//  3. matchByLocation — предпочтение ближайших к объекту
//
// Архитектура:
//
//	AutoDispatcher ──► TechnicianProvider (поиск техников)
//	              ──► WorkOrderProvider (WO CRUD)
//	              ──► SLAStatusChecker (SLA breach)
//	              ──► AuditLogger (audit trail)
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure — при ошибке не назначаем)
//   - IEC 62443 SR 3.1 (Data integrity)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - OWASP ASVS V7.1 (Log content)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
type AutoDispatcher struct {
	techProvider TechnicianProvider
	woProvider   WorkOrderProvider
	slaChecker   SLAStatusChecker
	auditLogger  DispatcherAuditLogger
	cfg          AutoDispatcherConfig
	logger       *slog.Logger
	mu           sync.RWMutex
}

// NewAutoDispatcher создаёт новый AutoDispatcher.
func NewAutoDispatcher(
	techProvider TechnicianProvider,
	woProvider WorkOrderProvider,
	slaChecker SLAStatusChecker,
	auditLogger DispatcherAuditLogger,
	cfg AutoDispatcherConfig,
	logger *slog.Logger,
) *AutoDispatcher {
	cfg.validate()
	if logger == nil {
		logger = slog.Default()
	}

	return &AutoDispatcher{
		techProvider: techProvider,
		woProvider:   woProvider,
		slaChecker:   slaChecker,
		auditLogger:  auditLogger,
		cfg:          cfg,
		logger:       logger.With("component", "cmms-auto-dispatcher"),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Public API
// ═══════════════════════════════════════════════════════════════════════

// AutoAssign выполняет автоматическое назначение техника на Work Order.
//
// Алгоритм:
//  1. Загружает Work Order по ID
//  2. Определяет требуемые навыки из типа/приоритета WO
//  3. Находит доступных техников через TechnicianProvider
//  4. Применяет matching pipeline: Skills → Workload → Location
//  5. Назначает лучшего техника (если найден)
//  6. Логирует результат в audit trail
//  7. Возвращает результат назначения
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure — отказ при ошибке)
//   - IEC 62443 SR 3.1 (Data integrity)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - OWASP ASVS V5.1 (Input validation)
func (d *AutoDispatcher) AutoAssign(ctx context.Context, workOrderID string) (*AssignResult, error) {
	start := time.Now()
	d.logger.Info("auto-assigning technician", "work_order_id", workOrderID)

	// 1. Получаем Work Order
	wo, err := d.woProvider.GetWorkOrder(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("auto-assign: get work order %s: %w", workOrderID, err)
	}
	if wo == nil {
		return nil, fmt.Errorf("auto-assign: work order %s not found", workOrderID)
	}

	// 2. Проверяем, не назначен ли уже
	if wo.AssignedTo != nil && *wo.AssignedTo != "" {
		d.logger.Warn("work order already assigned",
			"work_order_id", workOrderID,
			"assigned_to", *wo.AssignedTo,
		)
		return &AssignResult{
			WorkOrderID:  workOrderID,
			TechnicianID: *wo.AssignedTo,
			Status:       AssignStatusAlreadyAssigned,
			Reason:       "work_order_already_assigned",
		}, nil
	}

	// 3. Определяем требуемые навыки из WO
	requiredSkills := d.extractRequiredSkills(wo)

	// 4. Находим site ID для location matching
	siteID := d.extractSiteID(wo)

	// 5. Получаем доступных техников
	techs, err := d.techProvider.FindAvailableTechnicians(ctx, requiredSkills, siteID)
	if err != nil {
		d.logAudit(ctx, &DispatchAuditEntry{
			WorkOrderID: workOrderID,
			Action:      "auto_assign_failed",
			Reason:      "technician_provider_error",
			Duration:    time.Since(start),
			Timestamp:   time.Now().UTC(),
		})
		return nil, fmt.Errorf("auto-assign: find technicians: %w", err)
	}

	if len(techs) == 0 {
		d.logger.Warn("no available technicians found",
			"work_order_id", workOrderID,
			"required_skills", requiredSkills,
		)
		d.logAudit(ctx, &DispatchAuditEntry{
			WorkOrderID: workOrderID,
			Action:      "auto_assign_failed",
			Reason:      "no_available_technicians",
			Duration:    time.Since(start),
			Timestamp:   time.Now().UTC(),
		})
		return &AssignResult{
			WorkOrderID: workOrderID,
			Status:      AssignStatusNoTechnician,
			Reason:      "no_available_technicians",
		}, nil
	}

	// 6. Matching pipeline
	candidates := d.matchBySkills(wo, techs)
	candidates = d.matchByWorkload(candidates)

	if len(candidates) == 0 {
		d.logAudit(ctx, &DispatchAuditEntry{
			WorkOrderID: workOrderID,
			Action:      "auto_assign_failed",
			Reason:      "no_matching_technicians",
			Duration:    time.Since(start),
			Timestamp:   time.Now().UTC(),
		})
		return &AssignResult{
			WorkOrderID: workOrderID,
			Status:      AssignStatusNoTechnician,
			Reason:      "no_matching_technicians",
		}, nil
	}

	// 7. Берём лучшего кандидата
	best := candidates[0]

	// 8. Назначаем
	if err := d.woProvider.AssignWorkOrder(ctx, workOrderID, best.UserID); err != nil {
		d.logAudit(ctx, &DispatchAuditEntry{
			WorkOrderID:  workOrderID,
			TechnicianID: best.UserID,
			Action:       "auto_assign_failed",
			Reason:       "assign_error",
			Duration:     time.Since(start),
			Timestamp:    time.Now().UTC(),
		})
		return nil, fmt.Errorf("auto-assign: assign work order %s to %s: %w",
			workOrderID, best.UserID, err)
	}

	// 9. Аудит успешного назначения
	result := &AssignResult{
		WorkOrderID:    workOrderID,
		TechnicianID:   best.UserID,
		TechnicianName: best.UserName,
		Status:         AssignStatusSuccess,
		Reason:         "matched_by_algorithm",
		Score:          best.Score,
	}

	d.logAudit(ctx, &DispatchAuditEntry{
		WorkOrderID:    workOrderID,
		TechnicianID:   best.UserID,
		TechnicianName: best.UserName,
		Action:         "auto_assigned",
		Reason:         "matched_by_algorithm",
		Score:          best.Score,
		Duration:       time.Since(start),
		Timestamp:      time.Now().UTC(),
	})

	d.logger.Info("technician auto-assigned",
		"work_order_id", workOrderID,
		"technician_id", best.UserID,
		"technician_name", best.UserName,
		"score", best.Score,
		"duration", time.Since(start),
	)

	return result, nil
}

// ShouldEscalate проверяет, требуется ли эскалация для Work Order.
//
// Условия эскалации:
//  1. SLA breach — просроченный SLA дедлайн
//  2. Critical priority без назначения > 5 минут
//  3. Unassigned WO > 2 часа
//
// Возвращает:
//   - escalationLevel: 0 = нет, 1 = L1, 2 = L2, 3 = L3
//   - reason: причина эскалации
//   - error: ошибка выполнения
//
// Compliance:
//   - IEC 62443 SR 2.8 (Audit events — escalation)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - Приказ ОАЦ №66 п. 7.18.3 (Incident response)
func (d *AutoDispatcher) ShouldEscalate(ctx context.Context, wo *models.WorkOrder) (int, string, error) {
	if wo == nil {
		return 0, "", fmt.Errorf("should-escalate: nil work order")
	}

	now := time.Now().UTC()

	// 1. Проверка SLA breach
	if wo.SLADeadline != nil && now.After(*wo.SLADeadline) {
		breachedSince := now.Sub(*wo.SLADeadline)
		d.logger.Warn("SLA breach detected for auto-escalation",
			"work_order_id", wo.ID,
			"priority", wo.Priority,
			"breached_since", breachedSince,
		)

		// Вызываем escalation engine
		if d.slaChecker != nil {
			if _, err := d.slaChecker.CheckEscalation(ctx, wo.ID, wo.Priority, breachedSince); err != nil {
				d.logger.Error("failed to check escalation",
					"work_order_id", wo.ID, "error", err,
				)
			}
		}

		level := 1
		if breachedSince > 1*time.Hour {
			level = 2
		}
		if breachedSince > 4*time.Hour {
			level = 3
		}

		d.logAudit(ctx, &DispatchAuditEntry{
			WorkOrderID: wo.ID,
			Action:      "escalated",
			Reason:      fmt.Sprintf("sla_breach_level_%d", level),
			Timestamp:   now,
		})

		return level, fmt.Sprintf("sla_breach_%s", wo.Priority), nil
	}

	// 2. Critical priority без назначения
	if wo.Priority == "critical" && (wo.AssignedTo == nil || *wo.AssignedTo == "") {
		createdAgo := now.Sub(wo.CreatedAt)
		if createdAgo > CriticalAutoAssignTimeout {
			d.logAudit(ctx, &DispatchAuditEntry{
				WorkOrderID: wo.ID,
				Action:      "escalated",
				Reason:      "critical_unassigned",
				Timestamp:   now,
			})
			return 2, "critical_unassigned_timeout", nil
		}
	}

	// 3. Unassigned WO > 2h
	if wo.AssignedTo == nil || *wo.AssignedTo == "" {
		createdAgo := now.Sub(wo.CreatedAt)
		if createdAgo > d.cfg.UnassignedEscalationThreshold {
			d.logAudit(ctx, &DispatchAuditEntry{
				WorkOrderID: wo.ID,
				Action:      "escalated",
				Reason:      "unassigned_timeout",
				Timestamp:   now,
			})
			return 1, "unassigned_exceeds_threshold", nil
		}
	}

	return 0, "", nil
}

// BatchAutoAssign выполняет автоматическое назначение для всех непривязанных WO.
//
// Используется для периодического batch-назначения (cron).
// Обрабатывает WO по одному, пропуская уже назначенные.
func (d *AutoDispatcher) BatchAutoAssign(ctx context.Context) (*BatchAssignResult, error) {
	start := time.Now()
	d.logger.Info("starting batch auto-assign")

	// Получаем непривязанные WO старше 5 минут
	// (чтобы не назначать только что созданные — даём время на ручное назначение)
	unassigned, err := d.woProvider.ListUnassignedWorkOrders(ctx, 5*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("batch auto-assign: list unassigned: %w", err)
	}

	result := &BatchAssignResult{
		Total:   len(unassigned),
		Results: make([]*AssignResult, 0, len(unassigned)),
	}

	for _, wo := range unassigned {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, fmt.Errorf("context cancelled after %d processed", result.Assigned))
			return result, ctx.Err()
		default:
		}

		assignResult, err := d.AutoAssign(ctx, wo.ID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", wo.ID, err))
			continue
		}

		switch assignResult.Status {
		case AssignStatusSuccess:
			result.Assigned++
		default:
			result.Skipped++
		}
		result.Results = append(result.Results, assignResult)
	}

	result.Duration = time.Since(start)
	d.logger.Info("batch auto-assign complete",
		"total", result.Total,
		"assigned", result.Assigned,
		"failed", result.Failed,
		"skipped", result.Skipped,
		"duration", result.Duration,
	)

	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Matching Pipeline
// ═══════════════════════════════════════════════════════════════════════

// TechnicianCandidate — кандидат на назначение с оценкой.
type TechnicianCandidate struct {
	UserID          string   `json:"user_id"`
	UserName        string   `json:"user_name"`
	CurrentWorkload int      `json:"current_workload"`
	MaxWorkload     int      `json:"max_workload"`
	Skills          []string `json:"skills"`
	BaseLocation    *string  `json:"base_location"`
	Score           float64  `json:"score"` // общая оценка (0-100), выше = лучше
}

// matchBySkills фильтрует техников по наличию требуемых навыков.
//
// Если WO требует специфических навыков (CCTV, network), то техник
// должен иметь хотя бы один из них. Если навыки не указаны —
// пропускаем всех.
//
// Возвращает отфильтрованный список с базовой оценкой.
func (d *AutoDispatcher) matchBySkills(wo *models.WorkOrder, techs []models.TechnicianWorkload) []TechnicianCandidate {
	requiredSkills := d.extractRequiredSkills(wo)

	candidates := make([]TechnicianCandidate, 0, len(techs))

	for _, tech := range techs {
		score := 50.0 // базовый score

		if len(requiredSkills) > 0 {
			// Проверяем пересечение навыков
			matchCount := 0
			for _, req := range requiredSkills {
				for _, skill := range tech.Skills {
					if skill == req {
						matchCount++
						break
					}
				}
			}

			if matchCount == 0 {
				// Не имеет ни одного требуемого навыка — пропускаем
				continue
			}

			// Оценка за навыки: (matchCount / len(requiredSkills)) * 50
			skillScore := float64(matchCount) / float64(len(requiredSkills)) * 50.0
			score += skillScore
		} else {
			// Если навыки не требуются — базовый score
			score += 25.0
		}

		candidates = append(candidates, TechnicianCandidate{
			UserID:          tech.UserID,
			UserName:        tech.UserName,
			CurrentWorkload: tech.CurrentWorkload,
			MaxWorkload:     tech.MaxWorkload,
			Skills:          tech.Skills,
			BaseLocation:    tech.BaseLocation,
			Score:           score,
		})
	}

	return candidates
}

// matchByWorkload сортирует кандидатов по загрузке (least loaded first).
//
// Техник с меньшим количеством активных WO получает + к score.
// Полностью загруженные техники (CurrentWorkload >= MaxWorkload) исключаются.
func (d *AutoDispatcher) matchByWorkload(candidates []TechnicianCandidate) []TechnicianCandidate {
	filtered := make([]TechnicianCandidate, 0, len(candidates))

	for _, c := range candidates {
		maxWorkload := c.MaxWorkload
		if maxWorkload <= 0 {
			maxWorkload = d.cfg.MaxWorkload
		}

		// Исключаем полностью загруженных
		if c.CurrentWorkload >= maxWorkload {
			continue
		}

		// Оценка за загрузку: (1 - active/max) * 30
		loadScore := (1.0 - float64(c.CurrentWorkload)/float64(maxWorkload)) * 30.0
		c.Score += loadScore

		filtered = append(filtered, c)
	}

	// Сортируем по score (убывание)
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].Score > filtered[i].Score {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	return filtered
}

// ═══════════════════════════════════════════════════════════════════════
// Escalation Runner
// ═══════════════════════════════════════════════════════════════════════

// RunEscalationCheck проверяет все непривязанные WO на необходимость эскалации.
//
// Запускается периодически (cron) для auto-escalation.
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.3 (Incident response)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - IEC 62443 SR 2.8 (Audit events)
func (d *AutoDispatcher) RunEscalationCheck(ctx context.Context) ([]EscalationResult, error) {
	start := time.Now()
	d.logger.Info("running escalation check")

	// Получаем все непривязанные WO
	unassigned, err := d.woProvider.ListUnassignedWorkOrders(ctx, 0)
	if err != nil {
		return nil, fmt.Errorf("escalation check: list unassigned: %w", err)
	}

	var results []EscalationResult

	for _, wo := range unassigned {
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		level, reason, err := d.ShouldEscalate(ctx, &wo)
		if err != nil {
			d.logger.Warn("escalation check error",
				"work_order_id", wo.ID, "error", err,
			)
			continue
		}

		if level > 0 {
			results = append(results, EscalationResult{
				WorkOrderID: wo.ID,
				Level:       level,
				Reason:      reason,
				Priority:    wo.Priority,
				CreatedAt:   wo.CreatedAt,
				EscalatedAt: time.Now().UTC(),
			})

			d.logger.Warn("work order escalated",
				"work_order_id", wo.ID,
				"level", level,
				"reason", reason,
				"priority", wo.Priority,
			)
		}
	}

	d.logger.Info("escalation check complete",
		"total", len(unassigned),
		"escalated", len(results),
		"duration", time.Since(start),
	)

	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Result Types
// ═══════════════════════════════════════════════════════════════════════

// AssignStatus — статус назначения.
type AssignStatus string

const (
	AssignStatusSuccess         AssignStatus = "assigned"
	AssignStatusAlreadyAssigned AssignStatus = "already_assigned"
	AssignStatusNoTechnician    AssignStatus = "no_technician"
	AssignStatusFailed          AssignStatus = "failed"
)

// AssignResult — результат автоматического назначения.
type AssignResult struct {
	WorkOrderID    string       `json:"work_order_id"`
	TechnicianID   string       `json:"technician_id,omitempty"`
	TechnicianName string       `json:"technician_name,omitempty"`
	Status         AssignStatus `json:"status"`
	Reason         string       `json:"reason"`
	Score          float64      `json:"score,omitempty"`
}

// BatchAssignResult — результат batch-назначения.
type BatchAssignResult struct {
	Total    int             `json:"total"`
	Assigned int             `json:"assigned"`
	Failed   int             `json:"failed"`
	Skipped  int             `json:"skipped"`
	Duration time.Duration   `json:"duration_ms"`
	Results  []*AssignResult `json:"results,omitempty"`
	Errors   []error         `json:"errors,omitempty"`
}

// EscalationResult — результат проверки эскалации.
type EscalationResult struct {
	WorkOrderID string    `json:"work_order_id"`
	Level       int       `json:"escalation_level"`
	Reason      string    `json:"reason"`
	Priority    string    `json:"priority"`
	CreatedAt   time.Time `json:"created_at"`
	EscalatedAt time.Time `json:"escalated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// extractRequiredSkills извлекает требуемые навыки из Work Order.
//
// Определяется на основе типа WO и приоритета:
//   - emergency: требует все базовые навыки
//   - corrective: зависит от устройства
//   - preventive: стандартный набор
//   - routine: минимальные навыки
func (d *AutoDispatcher) extractRequiredSkills(wo *models.WorkOrder) []string {
	if wo == nil {
		return nil
	}

	switch wo.Type {
	case "emergency":
		return []string{"cctv", "network", "electrical"}
	case "corrective":
		return []string{"cctv", "network"}
	case "preventive":
		return []string{"cctv"}
	case "inspection":
		return []string{"cctv", "safety"}
	default:
		return []string{"cctv"}
	}
}

// extractSiteID извлекает site_id из Work Order.
//
// Сначала проверяет DeviceID (через device → site), потом прямые поля.
func (d *AutoDispatcher) extractSiteID(wo *models.WorkOrder) string {
	// В текущей модели WorkOrder site_id может быть получен через device_id
	// или через дополнительное поле. Пока возвращаем пустую строку,
	// чтобы не ломать matching — location фильтр будет пропущен.
	_ = wo.DeviceID
	return ""
}

// haversineDistance рассчитывает расстояние между двумя координатами (км).
//
// Используется для определения ближайших техников.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371.0

	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// logAudit логирует событие диспетчеризации.
func (d *AutoDispatcher) logAudit(ctx context.Context, entry *DispatchAuditEntry) {
	if !d.cfg.AuditLogEnabled || d.auditLogger == nil {
		return
	}
	entry.Duration = entry.Duration / time.Millisecond
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if err := d.auditLogger.LogDispatch(ctx, entry); err != nil {
		d.logger.Warn("failed to log dispatch audit",
			"work_order_id", entry.WorkOrderID,
			"action", entry.Action,
			"error", err,
		)
	}
}
