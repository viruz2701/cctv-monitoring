// Package cmms — CMMSIntegrator с поддержкой context timeouts.
//
// Оборачивает CMMSAdapter в слой с:
//   - Configurable timeouts per adapter type
//   - Graceful cancellation (ctx.Done())
//   - Timeout metrics (Prometheus-ready)
//   - Structured error wrapping
//
// Compliance:
//   - IEC 62443 SR 3.1 (Data integrity — timeout control)
//   - IEC 62443 SR 7.1 (Resource availability — prevent runaway ops)
//   - ISO 27001 A.12.6.1 (Capacity management — timeout configuration)
//   - OWASP ASVS V1.8 (Malicious code — timeout on all external calls)
//   - СТБ 34.101.27 п. 7.5 (Availability control)
package cmms

import (
	"context"
	"fmt"
	"gb-telemetry-collector/internal/models"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Constants & Defaults
// ═══════════════════════════════════════════════════════════════════════

// DefaultAdapterTimeouts — таймауты по умолчанию для каждого типа адаптера.
var DefaultAdapterTimeouts = map[string]time.Duration{
	"internal":   10 * time.Second,
	"atlas":      30 * time.Second,
	"servicenow": 60 * time.Second,
	"jira":       30 * time.Second,
	"toir":       30 * time.Second,
}

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// IntegratorConfig — конфигурация CMMSIntegrator.
type IntegratorConfig struct {
	// DefaultTimeout — таймаут по умолчанию для всех операций.
	// Если не указан — 30s.
	DefaultTimeout time.Duration `json:"default_timeout"`

	// AdapterTimeouts — таймауты для конкретных адаптеров (по имени).
	// Переопределяют DefaultTimeout для указанных адаптеров.
	AdapterTimeouts map[string]time.Duration `json:"adapter_timeouts"`

	// Logger — опциональный логгер.
	Logger *slog.Logger
}

func (c *IntegratorConfig) validate() {
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 30 * time.Second
	}
	if c.AdapterTimeouts == nil {
		c.AdapterTimeouts = make(map[string]time.Duration)
	}
	// Применяем defaults для известных адаптеров
	for name, timeout := range DefaultAdapterTimeouts {
		if _, ok := c.AdapterTimeouts[name]; !ok {
			c.AdapterTimeouts[name] = timeout
		}
	}
}

// AdapterNameFunc — функция, возвращающая имя адаптера для метрик.
// По умолчанию — fmt.Sprintf("%T", adapter).
type AdapterNameFunc func(adapter CMMSAdapter) string

// IntegratorMetrics — метрики выполнения операций.
type IntegratorMetrics struct {
	TotalOps       int64
	SuccessOps     int64
	TimeoutOps     int64
	CancelOps      int64
	ErrorOps       int64
	LastOpDuration time.Duration
	LastError      string
	AdapterStats   map[string]*AdapterStats
}

// AdapterStats — статистика для конкретного адаптера.
type AdapterStats struct {
	Name            string
	TotalOps        int64
	SuccessOps      int64
	TimeoutOps      int64
	CancelOps       int64
	ErrorOps        int64
	TotalDurationMs int64
	AvgDurationMs   float64
}

// ═══════════════════════════════════════════════════════════════════════
// CMMSIntegrator
// ═══════════════════════════════════════════════════════════════════════

// CMMSIntegrator — безопасный враппер над CMMSAdapter с контролем
// таймаутов, graceful cancellation и сбором метрик.
//
// Каждая операция:
//  1. Проверяет ctx.Done() перед началом
//  2. Применяет таймаут для конкретного адаптера
//  3. Обрабатывает context.DeadlineExceeded / context.Canceled
//  4. Собирает метрики (успех/таймаут/отмена/ошибка)
type CMMSIntegrator struct {
	adapter     CMMSAdapter
	nameFn      AdapterNameFunc
	adapterName string
	timeout     time.Duration
	cfg         IntegratorConfig
	logger      *slog.Logger

	// Metrics
	totalOps   atomic.Int64
	successOps atomic.Int64
	timeoutOps atomic.Int64
	cancelOps  atomic.Int64
	errorOps   atomic.Int64
	lastErr    atomic.Value // string
	lastDur    atomic.Int64 // nanoseconds

	mu          sync.RWMutex
	adapterStat map[string]*AdapterStats
}

// NewCMMSIntegrator создаёт CMMSIntegrator.
//
// Параметры:
//   - adapter: CMMSAdapter для выполнения операций
//   - cfg: конфигурация (таймауты, логгер)
//   - nameFn: опциональная функция для определения имени адаптера
func NewCMMSIntegrator(adapter CMMSAdapter, cfg IntegratorConfig, nameFn AdapterNameFunc) *CMMSIntegrator {
	cfg.validate()
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if nameFn == nil {
		nameFn = func(a CMMSAdapter) string {
			return fmt.Sprintf("%T", a)
		}
	}

	adapterName := nameFn(adapter)
	timeout := cfg.DefaultTimeout
	if t, ok := cfg.AdapterTimeouts[adapterName]; ok && t > 0 {
		timeout = t
	}

	integrator := &CMMSIntegrator{
		adapter:     adapter,
		nameFn:      nameFn,
		adapterName: adapterName,
		timeout:     timeout,
		cfg:         cfg,
		logger:      cfg.Logger.With("component", "cmms-integrator", "adapter", adapterName),
		adapterStat: make(map[string]*AdapterStats),
	}

	return integrator
}

// AdapterName возвращает имя адаптера (для метрик и логирования).
func (i *CMMSIntegrator) AdapterName() string {
	return i.adapterName
}

// ═══════════════════════════════════════════════════════════════════════
// Core Execution
// ═══════════════════════════════════════════════════════════════════════

// execute выполняет операцию с контролем таймаута.
//
// Алгоритм:
//  1. Проверка ctx.Done() — если контекст уже отменён, возвращаем ошибку
//  2. Создание контекста с таймаутом для адаптера
//  3. Запуск операции
//  4. Обработка результата: успех / таймаут / отмена / ошибка
//  5. Сбор метрик
func (i *CMMSIntegrator) execute(ctx context.Context, opName string, fn func(context.Context) error) error {
	i.totalOps.Add(1)

	// ── Шаг 1: Проверка ctx.Done() ──────────────────────────────────
	select {
	case <-ctx.Done():
		i.cancelOps.Add(1)
		i.logger.Warn("operation cancelled before start",
			"operation", opName,
			"error", ctx.Err(),
		)
		return fmt.Errorf("cmms_integrator: %s: %w", opName, ctx.Err())
	default:
	}

	// ── Шаг 2: Контекст с таймаутом ─────────────────────────────────
	opCtx, cancel := context.WithTimeout(ctx, i.timeout)
	defer cancel()

	start := time.Now()

	// ── Шаг 3: Запуск операции ──────────────────────────────────────
	err := fn(opCtx)

	// ── Шаг 4: Обработка результата ─────────────────────────────────
	duration := time.Since(start)
	i.lastDur.Store(duration.Nanoseconds())

	switch {
	case err == nil:
		i.successOps.Add(1)
		i.logger.Debug("operation completed",
			"operation", opName,
			"duration_ms", duration.Milliseconds(),
		)

	case isContextDeadlineExceeded(err, opCtx):
		i.timeoutOps.Add(1)
		i.lastErr.Store(err.Error())
		i.logger.Warn("operation timed out",
			"operation", opName,
			"timeout", i.timeout,
			"duration_ms", duration.Milliseconds(),
		)
		return fmt.Errorf("cmms_integrator: %s timed out after %v: %w",
			opName, i.timeout, err)

	case isContextCanceled(err, opCtx):
		i.cancelOps.Add(1)
		i.lastErr.Store(err.Error())
		i.logger.Warn("operation cancelled",
			"operation", opName,
			"duration_ms", duration.Milliseconds(),
		)
		return fmt.Errorf("cmms_integrator: %s cancelled: %w", opName, err)

	default:
		i.errorOps.Add(1)
		i.lastErr.Store(err.Error())
		i.logger.Error("operation failed",
			"operation", opName,
			"error", err,
			"duration_ms", duration.Milliseconds(),
		)
		return fmt.Errorf("cmms_integrator: %s: %w", opName, err)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Delegated CMMSAdapter Methods
// ═══════════════════════════════════════════════════════════════════════

// ── Work Orders ──────────────────────────────────────────────────────

func (i *CMMSIntegrator) CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error {
	return i.execute(ctx, "CreateWorkOrder", func(ctx context.Context) error {
		return i.adapter.CreateWorkOrder(ctx, wo)
	})
}

func (i *CMMSIntegrator) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	var result []models.WorkOrder
	err := i.execute(ctx, "GetWorkOrders", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetWorkOrders(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	var result *models.WorkOrder
	err := i.execute(ctx, "GetWorkOrder", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetWorkOrder(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateWorkOrder", func(ctx context.Context) error {
		return i.adapter.UpdateWorkOrder(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) AssignWorkOrder(ctx context.Context, id, userID string) error {
	return i.execute(ctx, "AssignWorkOrder", func(ctx context.Context) error {
		return i.adapter.AssignWorkOrder(ctx, id, userID)
	})
}

func (i *CMMSIntegrator) StartWorkOrder(ctx context.Context, id string) error {
	return i.execute(ctx, "StartWorkOrder", func(ctx context.Context) error {
		return i.adapter.StartWorkOrder(ctx, id)
	})
}

func (i *CMMSIntegrator) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	return i.execute(ctx, "CompleteWorkOrder", func(ctx context.Context) error {
		return i.adapter.CompleteWorkOrder(ctx, id, notes, photos, parts, userID)
	})
}

func (i *CMMSIntegrator) CancelWorkOrder(ctx context.Context, id, reason string) error {
	return i.execute(ctx, "CancelWorkOrder", func(ctx context.Context) error {
		return i.adapter.CancelWorkOrder(ctx, id, reason)
	})
}

func (i *CMMSIntegrator) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	return i.execute(ctx, "UsePartInWorkOrder", func(ctx context.Context) error {
		return i.adapter.UsePartInWorkOrder(ctx, workOrderID, partID, quantity, userID)
	})
}

// ── Spare Parts ──────────────────────────────────────────────────────

func (i *CMMSIntegrator) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	return i.execute(ctx, "CreateSparePart", func(ctx context.Context) error {
		return i.adapter.CreateSparePart(ctx, part)
	})
}

func (i *CMMSIntegrator) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	var result []models.SparePart
	err := i.execute(ctx, "GetSpareParts", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetSpareParts(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	var result *models.SparePart
	err := i.execute(ctx, "GetSparePart", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetSparePart(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateSparePart", func(ctx context.Context) error {
		return i.adapter.UpdateSparePart(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) DeleteSparePart(ctx context.Context, id string) error {
	return i.execute(ctx, "DeleteSparePart", func(ctx context.Context) error {
		return i.adapter.DeleteSparePart(ctx, id)
	})
}

func (i *CMMSIntegrator) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	var result []models.SparePart
	err := i.execute(ctx, "GetLowStockParts", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetLowStockParts(ctx)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	return i.execute(ctx, "UpdateSparePartStock", func(ctx context.Context) error {
		return i.adapter.UpdateSparePartStock(ctx, id, quantity)
	})
}

// ── Maintenance Schedules ────────────────────────────────────────────

func (i *CMMSIntegrator) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	return i.execute(ctx, "CreateMaintenanceSchedule", func(ctx context.Context) error {
		return i.adapter.CreateMaintenanceSchedule(ctx, schedule)
	})
}

func (i *CMMSIntegrator) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	var result []models.MaintenanceSchedule
	err := i.execute(ctx, "GetMaintenanceSchedules", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetMaintenanceSchedules(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	var result *models.MaintenanceSchedule
	err := i.execute(ctx, "GetMaintenanceSchedule", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetMaintenanceSchedule(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateMaintenanceSchedule", func(ctx context.Context) error {
		return i.adapter.UpdateMaintenanceSchedule(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	return i.execute(ctx, "DeleteMaintenanceSchedule", func(ctx context.Context) error {
		return i.adapter.DeleteMaintenanceSchedule(ctx, id)
	})
}

func (i *CMMSIntegrator) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	var result []models.MaintenanceSchedule
	err := i.execute(ctx, "GetDueSchedules", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetDueSchedules(ctx)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	return i.execute(ctx, "CompleteMaintenanceSchedule", func(ctx context.Context) error {
		return i.adapter.CompleteMaintenanceSchedule(ctx, id)
	})
}

// ── SLA ──────────────────────────────────────────────────────────────

func (i *CMMSIntegrator) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	var result *models.SLAConfig
	err := i.execute(ctx, "GetSLAConfig", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetSLAConfig(ctx, priority)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	var result []models.SLAConfig
	err := i.execute(ctx, "GetAllSLAConfigs", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetAllSLAConfigs(ctx)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	return i.execute(ctx, "UpdateSLAConfig", func(ctx context.Context) error {
		return i.adapter.UpdateSLAConfig(ctx, priority, responseTimeMinutes, resolutionTimeMinutes)
	})
}

// ── Technicians ──────────────────────────────────────────────────────

func (i *CMMSIntegrator) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	var result *models.TechnicianWorkload
	err := i.execute(ctx, "GetTechnicianWorkload", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetTechnicianWorkload(ctx, userID)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	var result []models.TechnicianWorkload
	err := i.execute(ctx, "GetAllTechnicianWorkloads", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetAllTechnicianWorkloads(ctx)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	var result *models.TechnicianMonthlyStats
	err := i.execute(ctx, "GetTechnicianMonthlyStats", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetTechnicianMonthlyStats(ctx, userID)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	return i.execute(ctx, "UpdateTechnicianSkills", func(ctx context.Context) error {
		return i.adapter.UpdateTechnicianSkills(ctx, userID, skills, certifications)
	})
}

// ── Reports ──────────────────────────────────────────────────────────

func (i *CMMSIntegrator) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	var result []models.MaintenanceReport
	err := i.execute(ctx, "GetMaintenanceReport", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetMaintenanceReport(ctx)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	var result []models.SLAComplianceReport
	err := i.execute(ctx, "GetSLAComplianceReport", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetSLAComplianceReport(ctx)
		return innerErr
	})
	return result, err
}

// ── Technician Site Assignments ──────────────────────────────────────

func (i *CMMSIntegrator) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	return i.execute(ctx, "CreateTechnicianSiteAssignment", func(ctx context.Context) error {
		return i.adapter.CreateTechnicianSiteAssignment(ctx, assignment)
	})
}

func (i *CMMSIntegrator) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	var result []models.TechnicianSiteAssignment
	err := i.execute(ctx, "GetTechnicianSiteAssignments", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetTechnicianSiteAssignments(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateTechnicianSiteAssignment", func(ctx context.Context) error {
		return i.adapter.UpdateTechnicianSiteAssignment(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	return i.execute(ctx, "DeleteTechnicianSiteAssignment", func(ctx context.Context) error {
		return i.adapter.DeleteTechnicianSiteAssignment(ctx, id)
	})
}

// ── Sites ────────────────────────────────────────────────────────────

func (i *CMMSIntegrator) GetSites(ctx context.Context, filters map[string]interface{}) ([]models.Site, error) {
	var result []models.Site
	err := i.execute(ctx, "GetSites", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetSites(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetSite(ctx context.Context, id string) (*models.Site, error) {
	var result *models.Site
	err := i.execute(ctx, "GetSite", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetSite(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) CreateSite(ctx context.Context, site *models.Site) error {
	return i.execute(ctx, "CreateSite", func(ctx context.Context) error {
		return i.adapter.CreateSite(ctx, site)
	})
}

func (i *CMMSIntegrator) UpdateSite(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateSite", func(ctx context.Context) error {
		return i.adapter.UpdateSite(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) DeleteSite(ctx context.Context, id string) error {
	return i.execute(ctx, "DeleteSite", func(ctx context.Context) error {
		return i.adapter.DeleteSite(ctx, id)
	})
}

// ── Spare Part Categories ────────────────────────────────────────────

func (i *CMMSIntegrator) GetCategories(ctx context.Context) ([]models.SparePartCategory, error) {
	var result []models.SparePartCategory
	err := i.execute(ctx, "GetCategories", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetCategories(ctx)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) CreateCategory(ctx context.Context, cat *models.SparePartCategory) error {
	return i.execute(ctx, "CreateCategory", func(ctx context.Context) error {
		return i.adapter.CreateCategory(ctx, cat)
	})
}

func (i *CMMSIntegrator) UpdateCategory(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateCategory", func(ctx context.Context) error {
		return i.adapter.UpdateCategory(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) DeleteCategory(ctx context.Context, id string) error {
	return i.execute(ctx, "DeleteCategory", func(ctx context.Context) error {
		return i.adapter.DeleteCategory(ctx, id)
	})
}

// ── Work Requests ────────────────────────────────────────────────────

func (i *CMMSIntegrator) CreateWorkRequest(ctx context.Context, req *models.WorkRequest) error {
	return i.execute(ctx, "CreateWorkRequest", func(ctx context.Context) error {
		return i.adapter.CreateWorkRequest(ctx, req)
	})
}

func (i *CMMSIntegrator) GetWorkRequests(ctx context.Context, filters map[string]interface{}) ([]models.WorkRequest, error) {
	var result []models.WorkRequest
	err := i.execute(ctx, "GetWorkRequests", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetWorkRequests(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetWorkRequest(ctx context.Context, id string) (*models.WorkRequest, error) {
	var result *models.WorkRequest
	err := i.execute(ctx, "GetWorkRequest", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetWorkRequest(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) ApproveWorkRequest(ctx context.Context, id, approvedBy string) error {
	return i.execute(ctx, "ApproveWorkRequest", func(ctx context.Context) error {
		return i.adapter.ApproveWorkRequest(ctx, id, approvedBy)
	})
}

func (i *CMMSIntegrator) RejectWorkRequest(ctx context.Context, id, rejectedBy, reason string) error {
	return i.execute(ctx, "RejectWorkRequest", func(ctx context.Context) error {
		return i.adapter.RejectWorkRequest(ctx, id, rejectedBy, reason)
	})
}

func (i *CMMSIntegrator) ConvertWorkRequestToWO(ctx context.Context, requestID, workOrderID string) error {
	return i.execute(ctx, "ConvertWorkRequestToWO", func(ctx context.Context) error {
		return i.adapter.ConvertWorkRequestToWO(ctx, requestID, workOrderID)
	})
}

// ── WorkOrder ↔ Alert ────────────────────────────────────────────────

func (i *CMMSIntegrator) LinkAlertToWorkOrder(ctx context.Context, workOrderID, alertID, userID string) error {
	return i.execute(ctx, "LinkAlertToWorkOrder", func(ctx context.Context) error {
		return i.adapter.LinkAlertToWorkOrder(ctx, workOrderID, alertID, userID)
	})
}

func (i *CMMSIntegrator) UnlinkAlertFromWorkOrder(ctx context.Context, workOrderID, alertID string) error {
	return i.execute(ctx, "UnlinkAlertFromWorkOrder", func(ctx context.Context) error {
		return i.adapter.UnlinkAlertFromWorkOrder(ctx, workOrderID, alertID)
	})
}

func (i *CMMSIntegrator) GetAlertsForWorkOrder(ctx context.Context, workOrderID string) ([]models.WorkOrderAlert, error) {
	var result []models.WorkOrderAlert
	err := i.execute(ctx, "GetAlertsForWorkOrder", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetAlertsForWorkOrder(ctx, workOrderID)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetWorkOrdersForAlert(ctx context.Context, alertID string) ([]models.WorkOrderAlert, error) {
	var result []models.WorkOrderAlert
	err := i.execute(ctx, "GetWorkOrdersForAlert", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetWorkOrdersForAlert(ctx, alertID)
		return innerErr
	})
	return result, err
}

// ── Vendors ──────────────────────────────────────────────────────────

func (i *CMMSIntegrator) CreateVendor(ctx context.Context, vendor *models.Vendor) error {
	return i.execute(ctx, "CreateVendor", func(ctx context.Context) error {
		return i.adapter.CreateVendor(ctx, vendor)
	})
}

func (i *CMMSIntegrator) GetVendors(ctx context.Context, filters map[string]interface{}) ([]models.Vendor, error) {
	var result []models.Vendor
	err := i.execute(ctx, "GetVendors", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetVendors(ctx, filters)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) GetVendor(ctx context.Context, id string) (*models.Vendor, error) {
	var result *models.Vendor
	err := i.execute(ctx, "GetVendor", func(ctx context.Context) error {
		var innerErr error
		result, innerErr = i.adapter.GetVendor(ctx, id)
		return innerErr
	})
	return result, err
}

func (i *CMMSIntegrator) UpdateVendor(ctx context.Context, id string, updates map[string]interface{}) error {
	return i.execute(ctx, "UpdateVendor", func(ctx context.Context) error {
		return i.adapter.UpdateVendor(ctx, id, updates)
	})
}

func (i *CMMSIntegrator) DeleteVendor(ctx context.Context, id string) error {
	return i.execute(ctx, "DeleteVendor", func(ctx context.Context) error {
		return i.adapter.DeleteVendor(ctx, id)
	})
}

// ── Mobile ───────────────────────────────────────────────────────────

func (i *CMMSIntegrator) SavePushToken(ctx context.Context, userID, token, platform string) error {
	return i.execute(ctx, "SavePushToken", func(ctx context.Context) error {
		return i.adapter.SavePushToken(ctx, userID, token, platform)
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Metrics
// ═══════════════════════════════════════════════════════════════════════

// Metrics возвращает копию текущих метрик.
func (i *CMMSIntegrator) Metrics() IntegratorMetrics {
	lastErrStr := ""
	if v := i.lastErr.Load(); v != nil {
		lastErrStr = v.(string)
	}

	return IntegratorMetrics{
		TotalOps:       i.totalOps.Load(),
		SuccessOps:     i.successOps.Load(),
		TimeoutOps:     i.timeoutOps.Load(),
		CancelOps:      i.cancelOps.Load(),
		ErrorOps:       i.errorOps.Load(),
		LastOpDuration: time.Duration(i.lastDur.Load()),
		LastError:      lastErrStr,
	}
}

// ResetMetrics сбрасывает все метрики.
func (i *CMMSIntegrator) ResetMetrics() {
	i.totalOps.Store(0)
	i.successOps.Store(0)
	i.timeoutOps.Store(0)
	i.cancelOps.Store(0)
	i.errorOps.Store(0)
	i.lastDur.Store(0)
	i.lastErr.Store("")
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// isContextDeadlineExceeded проверяет, является ли ошибка результатом
// превышения таймаута (нашего или родительского контекста).
func isContextDeadlineExceeded(err error, opCtx context.Context) bool {
	if err == context.DeadlineExceeded {
		return true
	}
	if opCtx.Err() == context.DeadlineExceeded {
		return true
	}
	return false
}

// isContextCanceled проверяет, является ли ошибка результатом отмены
// контекста (нашей или родительской).
func isContextCanceled(err error, opCtx context.Context) bool {
	if err == context.Canceled {
		return true
	}
	if opCtx.Err() == context.Canceled {
		return true
	}
	return false
}
