// Package cmms — Tenant-Aware CMMS Router (CMMS-3.3.1).
//
// Per-tenant adapter selection позволяет выбирать CMMSAdapter для каждого
// tenant отдельно. Конфигурация: cmms.adapter: "internal" | "atlas" | "servicenow".
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - ISO 27001 A.9.1.2 (Access to networks — tenant separation)
//   - ISO 27001 A.15.1.1 (Supplier relationships — adapter per tenant)
//   - OWASP ASVS V2.1 (Authentication — tenant context)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
package cmms

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// TenantRouter — маршрутизация CMMS-запросов по tenant.
// ═══════════════════════════════════════════════════════════════════════

// TenantAdapterResolver определяет, какой адаптер использовать для tenant.
type TenantAdapterResolver interface {
	// ResolveAdapter возвращает имя адаптера для tenant.
	// Возвращает "internal" (по умолчанию), "atlas", "servicenow", "toir", "jira".
	ResolveAdapter(ctx context.Context, tenantID string) (string, error)
}

// TenantAdapterResolverFunc — адаптер функции для TenantAdapterResolver.
type TenantAdapterResolverFunc func(ctx context.Context, tenantID string) (string, error)

func (f TenantAdapterResolverFunc) ResolveAdapter(ctx context.Context, tenantID string) (string, error) {
	return f(ctx, tenantID)
}

// TenantRouterConfig — конфигурация TenantRouter.
type TenantRouterConfig struct {
	// DefaultAdapter — адаптер по умолчанию для tenant'ов без конфигурации.
	DefaultAdapter string `json:"default_adapter"`

	// PerTenantOverrides — явное указание адаптера для конкретных tenant'ов.
	// key = tenantID, value = adapter name.
	PerTenantOverrides map[string]string `json:"per_tenant_overrides"`
}

// TenantRouter — маршрутизатор, который выбирает CMMSAdapter на основе tenant.
//
// Архитектура:
//
//	Request (с tenantID) ─► TenantRouter ─► resolve adapter name
//	                          │
//	                          ▼
//	                    AdapterRegistry ─► CMMSAdapter
//
// Каждый tenant может использовать свой адаптер, что позволяет:
//   - Разным клиентам использовать разные CMMS (Internal, Atlas, ServiceNow)
//   - Постепенный migration между адаптерами
//   - A/B тестирование адаптеров
type TenantRouter struct {
	registry    *AdapterRegistry
	resolver    TenantAdapterResolver
	defaultName string
	overrides   map[string]string
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewTenantRouter создаёт TenantRouter.
//
// Параметры:
//   - registry: реестр адаптеров (должен быть проинициализирован)
//   - resolver: резолвер tenant → adapter name
//   - cfg: конфигурация
//   - logger: логгер
func NewTenantRouter(
	registry *AdapterRegistry,
	resolver TenantAdapterResolver,
	cfg TenantRouterConfig,
	logger *slog.Logger,
) *TenantRouter {
	if logger == nil {
		logger = slog.Default()
	}

	defaultName := cfg.DefaultAdapter
	if defaultName == "" {
		defaultName = "internal"
	}

	if cfg.PerTenantOverrides == nil {
		cfg.PerTenantOverrides = make(map[string]string)
	}

	return &TenantRouter{
		registry:    registry,
		resolver:    resolver,
		defaultName: defaultName,
		overrides:   cfg.PerTenantOverrides,
		logger:      logger.With("component", "cmms-tenant-router"),
	}
}

// GetAdapter возвращает CMMSAdapter для указанного tenant.
func (r *TenantRouter) GetAdapter(ctx context.Context, tenantID string) (CMMSAdapter, error) {
	// 1. Проверяем явные override
	r.mu.RLock()
	override, ok := r.overrides[tenantID]
	r.mu.RUnlock()

	if ok {
		adapter, err := r.registry.Get(override)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: override adapter %q: %w", tenantID, override, err)
		}
		r.logger.Debug("using overridden adapter for tenant",
			"tenant_id", tenantID, "adapter", override,
		)
		return adapter, nil
	}

	// 2. Спрашиваем resolver
	if r.resolver != nil {
		adapterName, err := r.resolver.ResolveAdapter(ctx, tenantID)
		if err != nil {
			r.logger.Warn("tenant adapter resolver failed, falling back to default",
				"tenant_id", tenantID, "error", err,
			)
			return r.getDefaultAdapter()
		}

		if adapterName != "" && adapterName != r.defaultName {
			adapter, err := r.registry.Get(adapterName)
			if err != nil {
				r.logger.Warn("resolved adapter not found, falling back to default",
					"tenant_id", tenantID, "adapter", adapterName, "error", err,
				)
				return r.getDefaultAdapter()
			}
			r.logger.Debug("resolved adapter for tenant",
				"tenant_id", tenantID, "adapter", adapterName,
			)
			return adapter, nil
		}
	}

	// 3. Default
	return r.getDefaultAdapter()
}

// AdapterName возвращает имя адаптера для tenant (без создания адаптера).
func (r *TenantRouter) AdapterName(ctx context.Context, tenantID string) string {
	r.mu.RLock()
	override, ok := r.overrides[tenantID]
	r.mu.RUnlock()

	if ok {
		return override
	}

	if r.resolver != nil {
		name, err := r.resolver.ResolveAdapter(ctx, tenantID)
		if err == nil && name != "" {
			return name
		}
	}

	return r.defaultName
}

// SetOverride устанавливает override адаптера для tenant.
func (r *TenantRouter) SetOverride(tenantID, adapterName string) error {
	if err := r.registry.Validate(adapterName); err != nil {
		return fmt.Errorf("set override: %w", err)
	}

	r.mu.Lock()
	r.overrides[tenantID] = adapterName
	r.mu.Unlock()

	r.logger.Info("tenant adapter override set",
		"tenant_id", tenantID, "adapter", adapterName,
	)
	return nil
}

// RemoveOverride удаляет override адаптера для tenant.
func (r *TenantRouter) RemoveOverride(tenantID string) {
	r.mu.Lock()
	delete(r.overrides, tenantID)
	r.mu.Unlock()

	r.logger.Info("tenant adapter override removed", "tenant_id", tenantID)
}

// ListOverrides возвращает все override адаптеров.
func (r *TenantRouter) ListOverrides() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string, len(r.overrides))
	for k, v := range r.overrides {
		result[k] = v
	}
	return result
}

func (r *TenantRouter) getDefaultAdapter() (CMMSAdapter, error) {
	adapter, err := r.registry.Get(r.defaultName)
	if err != nil {
		return nil, fmt.Errorf("default adapter %q: %w", r.defaultName, err)
	}
	return adapter, nil
}

// ═══════════════════════════════════════════════════════════════════════
// AdapterRegistry — реестр CMMSAdapter
// ═══════════════════════════════════════════════════════════════════════

// AdapterRegistry — потокобезопасный реестр CMMSAdapter.
type AdapterRegistry struct {
	adapters map[string]CMMSAdapter
	mu       sync.RWMutex
	logger   *slog.Logger
}

// NewAdapterRegistry создаёт реестр адаптеров.
func NewAdapterRegistry(logger *slog.Logger) *AdapterRegistry {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdapterRegistry{
		adapters: make(map[string]CMMSAdapter),
		logger:   logger.With("component", "cmms-adapter-registry"),
	}
}

// Register регистрирует адаптер под именем.
func (r *AdapterRegistry) Register(name string, adapter CMMSAdapter) error {
	if name == "" {
		return fmt.Errorf("adapter name is required")
	}
	if adapter == nil {
		return fmt.Errorf("adapter %q is nil", name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.adapters[name]; exists {
		r.logger.Warn("overwriting existing adapter", "name", name)
	}

	r.adapters[name] = adapter
	r.logger.Info("adapter registered", "name", name)
	return nil
}

// Get возвращает адаптер по имени.
func (r *AdapterRegistry) Get(name string) (CMMSAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	adapter, ok := r.adapters[name]
	if !ok {
		return nil, fmt.Errorf("adapter %q not found", name)
	}
	return adapter, nil
}

// Validate проверяет, что адаптер с таким именем зарегистрирован.
func (r *AdapterRegistry) Validate(name string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.adapters[name]; !ok {
		return fmt.Errorf("adapter %q not registered", name)
	}
	return nil
}

// List возвращает имена всех зарегистрированных адаптеров.
func (r *AdapterRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.adapters))
	for name := range r.adapters {
		names = append(names, name)
	}
	return names
}

// Remove удаляет адаптер из реестра.
func (r *AdapterRegistry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.adapters, name)
	r.logger.Info("adapter removed", "name", name)
}

// ═══════════════════════════════════════════════════════════════════════
// TenantRouterWrapper — обёртка CMMSAdapter с tenant-контекстом.
// ═══════════════════════════════════════════════════════════════════════

// TenantRouterWrapper реализует CMMSAdapter и делегирует вызовы
// в TenantRouter для выбора адаптера на основе tenantID из контекста.
//
// TenantID извлекается из context через middleware (см. internal/api/apikey_middleware.go).
type TenantRouterWrapper struct {
	router *TenantRouter
	logger *slog.Logger
}

// NewTenantRouterWrapper создаёт обёртку для TenantRouter.
func NewTenantRouterWrapper(router *TenantRouter, logger *slog.Logger) *TenantRouterWrapper {
	if logger == nil {
		logger = slog.Default()
	}
	return &TenantRouterWrapper{
		router: router,
		logger: logger.With("component", "cmms-tenant-wrapper"),
	}
}

// resolveAdapter извлекает tenantID из контекста и получает адаптер.
func (w *TenantRouterWrapper) resolveAdapter(ctx context.Context) (CMMSAdapter, error) {
	tenantID := TenantIDFromContext(ctx)
	if tenantID == "" {
		// Fallback для запросов без tenant (system context)
		return w.router.getDefaultAdapter()
	}
	return w.router.GetAdapter(ctx, tenantID)
}

// ── Work Orders ──────────────────────────────────────────────────

func (w *TenantRouterWrapper) CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CreateWorkOrder(ctx, wo)
}

func (w *TenantRouterWrapper) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetWorkOrders(ctx, filters)
}

func (w *TenantRouterWrapper) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetWorkOrder(ctx, id)
}

func (w *TenantRouterWrapper) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateWorkOrder(ctx, id, updates)
}

func (w *TenantRouterWrapper) AssignWorkOrder(ctx context.Context, id, userID string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.AssignWorkOrder(ctx, id, userID)
}

func (w *TenantRouterWrapper) StartWorkOrder(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.StartWorkOrder(ctx, id)
}

func (w *TenantRouterWrapper) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CompleteWorkOrder(ctx, id, notes, photos, parts, userID)
}

func (w *TenantRouterWrapper) CancelWorkOrder(ctx context.Context, id, reason string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CancelWorkOrder(ctx, id, reason)
}

func (w *TenantRouterWrapper) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UsePartInWorkOrder(ctx, workOrderID, partID, quantity, userID)
}

// ── Spare Parts ──────────────────────────────────────────────────

func (w *TenantRouterWrapper) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CreateSparePart(ctx, part)
}

func (w *TenantRouterWrapper) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetSpareParts(ctx, filters)
}

func (w *TenantRouterWrapper) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetSparePart(ctx, id)
}

func (w *TenantRouterWrapper) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateSparePart(ctx, id, updates)
}

func (w *TenantRouterWrapper) DeleteSparePart(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.DeleteSparePart(ctx, id)
}

func (w *TenantRouterWrapper) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetLowStockParts(ctx)
}

func (w *TenantRouterWrapper) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateSparePartStock(ctx, id, quantity)
}

// ── Maintenance Schedules ────────────────────────────────────────

func (w *TenantRouterWrapper) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CreateMaintenanceSchedule(ctx, schedule)
}

func (w *TenantRouterWrapper) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetMaintenanceSchedules(ctx, filters)
}

func (w *TenantRouterWrapper) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetMaintenanceSchedule(ctx, id)
}

func (w *TenantRouterWrapper) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateMaintenanceSchedule(ctx, id, updates)
}

func (w *TenantRouterWrapper) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.DeleteMaintenanceSchedule(ctx, id)
}

func (w *TenantRouterWrapper) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetDueSchedules(ctx)
}

func (w *TenantRouterWrapper) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CompleteMaintenanceSchedule(ctx, id)
}

// ── SLA ──────────────────────────────────────────────────────────

func (w *TenantRouterWrapper) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetSLAConfig(ctx, priority)
}

func (w *TenantRouterWrapper) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetAllSLAConfigs(ctx)
}

func (w *TenantRouterWrapper) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateSLAConfig(ctx, priority, responseTimeMinutes, resolutionTimeMinutes)
}

// ── Technicians ──────────────────────────────────────────────────

func (w *TenantRouterWrapper) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetTechnicianWorkload(ctx, userID)
}

func (w *TenantRouterWrapper) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetAllTechnicianWorkloads(ctx)
}

func (w *TenantRouterWrapper) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetTechnicianMonthlyStats(ctx, userID)
}

func (w *TenantRouterWrapper) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateTechnicianSkills(ctx, userID, skills, certifications)
}

// ── Reports ──────────────────────────────────────────────────────

func (w *TenantRouterWrapper) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetMaintenanceReport(ctx)
}

func (w *TenantRouterWrapper) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetSLAComplianceReport(ctx)
}

// ── Technician Site Assignments ──────────────────────────────────

func (w *TenantRouterWrapper) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CreateTechnicianSiteAssignment(ctx, assignment)
}

func (w *TenantRouterWrapper) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetTechnicianSiteAssignments(ctx, filters)
}

func (w *TenantRouterWrapper) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateTechnicianSiteAssignment(ctx, id, updates)
}

func (w *TenantRouterWrapper) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.DeleteTechnicianSiteAssignment(ctx, id)
}

// ── Sites ────────────────────────────────────────────────────────

func (w *TenantRouterWrapper) GetSites(ctx context.Context, filters map[string]interface{}) ([]models.Site, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetSites(ctx, filters)
}

func (w *TenantRouterWrapper) GetSite(ctx context.Context, id string) (*models.Site, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetSite(ctx, id)
}

func (w *TenantRouterWrapper) CreateSite(ctx context.Context, site *models.Site) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CreateSite(ctx, site)
}

func (w *TenantRouterWrapper) UpdateSite(ctx context.Context, id string, updates map[string]interface{}) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateSite(ctx, id, updates)
}

func (w *TenantRouterWrapper) DeleteSite(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.DeleteSite(ctx, id)
}

// ── Spare Part Categories ────────────────────────────────────────

func (w *TenantRouterWrapper) GetCategories(ctx context.Context) ([]models.SparePartCategory, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetCategories(ctx)
}

func (w *TenantRouterWrapper) CreateCategory(ctx context.Context, cat *models.SparePartCategory) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.CreateCategory(ctx, cat)
}

func (w *TenantRouterWrapper) UpdateCategory(ctx context.Context, id string, updates map[string]interface{}) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UpdateCategory(ctx, id, updates)
}

func (w *TenantRouterWrapper) DeleteCategory(ctx context.Context, id string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.DeleteCategory(ctx, id)
}

// ── Mobile ───────────────────────────────────────────────────────

// ── WorkOrder ↔ Alert (Many-to-Many) — DM-1.3.1 ────────────────

func (w *TenantRouterWrapper) LinkAlertToWorkOrder(ctx context.Context, workOrderID, alertID, userID string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.LinkAlertToWorkOrder(ctx, workOrderID, alertID, userID)
}

func (w *TenantRouterWrapper) UnlinkAlertFromWorkOrder(ctx context.Context, workOrderID, alertID string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.UnlinkAlertFromWorkOrder(ctx, workOrderID, alertID)
}

func (w *TenantRouterWrapper) GetAlertsForWorkOrder(ctx context.Context, workOrderID string) ([]models.WorkOrderAlert, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetAlertsForWorkOrder(ctx, workOrderID)
}

func (w *TenantRouterWrapper) GetWorkOrdersForAlert(ctx context.Context, alertID string) ([]models.WorkOrderAlert, error) {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return nil, err
	}
	return adapter.GetWorkOrdersForAlert(ctx, alertID)
}

func (w *TenantRouterWrapper) SavePushToken(ctx context.Context, userID, token, platform string) error {
	adapter, err := w.resolveAdapter(ctx)
	if err != nil {
		return err
	}
	return adapter.SavePushToken(ctx, userID, token, platform)
}

// ═══════════════════════════════════════════════════════════════════════
// Context helpers
// ═══════════════════════════════════════════════════════════════════════

// contextKey — тип для ключей контекста (чтобы избежать коллизий).
type contextKey string

const (
	// TenantIDKey — ключ для tenantID в context.
	TenantIDKey contextKey = "tenant_id"
)

// ContextWithTenantID добавляет tenantID в context.
func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// TenantIDFromContext извлекает tenantID из context.
func TenantIDFromContext(ctx context.Context) string {
	if v := ctx.Value(TenantIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
