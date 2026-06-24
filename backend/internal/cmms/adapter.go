// Package cmms предоставляет абстракцию для CMMS (Computerized Maintenance Management System).
// Реализует паттерн «Адаптер» с возможностью маршрутизации между InternalAdapter (БД)
// и внешними CMMS (Atlas, ServiceNow, 1С:ТОИР, Jira).
package cmms

import (
	"context"

	"gb-telemetry-collector/internal/models"
)

// CMMSAdapter определяет контракт для всех операций CMMS.
// Каждый метод принимает context.Context первым параметром для поддержки
// таймаутов, трейсинга и отмены операций.
type CMMSAdapter interface {
	// ── Work Orders ──────────────────────────────────────────────

	CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error
	GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error)
	GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error)
	UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error
	AssignWorkOrder(ctx context.Context, id, userID string) error
	StartWorkOrder(ctx context.Context, id string) error
	CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error
	CancelWorkOrder(ctx context.Context, id, reason string) error
	UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error

	// ── Spare Parts ──────────────────────────────────────────────

	CreateSparePart(ctx context.Context, part *models.SparePart) error
	GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error)
	GetSparePart(ctx context.Context, id string) (*models.SparePart, error)
	UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteSparePart(ctx context.Context, id string) error
	GetLowStockParts(ctx context.Context) ([]models.SparePart, error)
	UpdateSparePartStock(ctx context.Context, id string, quantity int) error

	// ── Maintenance Schedules ────────────────────────────────────

	CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error
	GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error)
	GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error)
	UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteMaintenanceSchedule(ctx context.Context, id string) error
	GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error)
	CompleteMaintenanceSchedule(ctx context.Context, id string) error

	// ── SLA ──────────────────────────────────────────────────────

	GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error)
	GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error)
	UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error

	// ── Technicians ──────────────────────────────────────────────

	GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error)
	GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error)
	GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error)
	UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error

	// ── Reports ──────────────────────────────────────────────────

	GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error)
	GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error)

	// ── Technician Site Assignments ──────────────────────────────

	CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error
	GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error)
	UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteTechnicianSiteAssignment(ctx context.Context, id string) error

	// ── Sites ────────────────────────────────────────────────────

	GetSites(ctx context.Context, filters map[string]interface{}) ([]models.Site, error)
	GetSite(ctx context.Context, id string) (*models.Site, error)
	CreateSite(ctx context.Context, site *models.Site) error
	UpdateSite(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteSite(ctx context.Context, id string) error

	// ── Spare Part Categories ────────────────────────────────────

	GetCategories(ctx context.Context) ([]models.SparePartCategory, error)
	CreateCategory(ctx context.Context, cat *models.SparePartCategory) error
	UpdateCategory(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteCategory(ctx context.Context, id string) error

	// ── Work Requests (WO-4.1.1) ────────────────────────────────

	CreateWorkRequest(ctx context.Context, req *models.WorkRequest) error
	GetWorkRequests(ctx context.Context, filters map[string]interface{}) ([]models.WorkRequest, error)
	GetWorkRequest(ctx context.Context, id string) (*models.WorkRequest, error)
	ApproveWorkRequest(ctx context.Context, id, approvedBy string) error
	RejectWorkRequest(ctx context.Context, id, rejectedBy, reason string) error
	ConvertWorkRequestToWO(ctx context.Context, requestID, workOrderID string) error

	// ── WorkOrder ↔ Alert (Many-to-Many) — DM-1.3.1 ────────────

	LinkAlertToWorkOrder(ctx context.Context, workOrderID, alertID, userID string) error
	UnlinkAlertFromWorkOrder(ctx context.Context, workOrderID, alertID string) error
	GetAlertsForWorkOrder(ctx context.Context, workOrderID string) ([]models.WorkOrderAlert, error)
	GetWorkOrdersForAlert(ctx context.Context, alertID string) ([]models.WorkOrderAlert, error)

	// ── Vendors (INV-7.2.1) ─────────────────────────────────────

	CreateVendor(ctx context.Context, vendor *models.Vendor) error
	GetVendors(ctx context.Context, filters map[string]interface{}) ([]models.Vendor, error)
	GetVendor(ctx context.Context, id string) (*models.Vendor, error)
	UpdateVendor(ctx context.Context, id string, updates map[string]interface{}) error
	DeleteVendor(ctx context.Context, id string) error

	// ── Mobile ───────────────────────────────────────────────────

	SavePushToken(ctx context.Context, userID, token, platform string) error
}

// CMMSRouter реализует паттерн «делегат» над CMMSAdapter.
// На текущем этапе просто проксирует все вызовы в выбранный адаптер.
// В будущем может маршрутизировать запросы между InternalAdapter и AtlasAdapter
// на основе типа устройства или других критериев.
type CMMSRouter struct {
	adapter CMMSAdapter
}

// NewCMMSRouter создаёт новый роутер с указанным адаптером.
func NewCMMSRouter(adapter CMMSAdapter) *CMMSRouter {
	return &CMMSRouter{adapter: adapter}
}

// Adapter возвращает базовый адаптер. Используется для доступа
// к специфичным методам адаптера (например, AtlasAdapter.HealthCheck).
func (r *CMMSRouter) Adapter() CMMSAdapter {
	return r.adapter
}

// ── Work Orders ──────────────────────────────────────────────────

func (r *CMMSRouter) CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error {
	return r.adapter.CreateWorkOrder(ctx, wo)
}

func (r *CMMSRouter) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	return r.adapter.GetWorkOrders(ctx, filters)
}

func (r *CMMSRouter) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	return r.adapter.GetWorkOrder(ctx, id)
}

func (r *CMMSRouter) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateWorkOrder(ctx, id, updates)
}

func (r *CMMSRouter) AssignWorkOrder(ctx context.Context, id, userID string) error {
	return r.adapter.AssignWorkOrder(ctx, id, userID)
}

func (r *CMMSRouter) StartWorkOrder(ctx context.Context, id string) error {
	return r.adapter.StartWorkOrder(ctx, id)
}

func (r *CMMSRouter) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	return r.adapter.CompleteWorkOrder(ctx, id, notes, photos, parts, userID)
}

func (r *CMMSRouter) CancelWorkOrder(ctx context.Context, id, reason string) error {
	return r.adapter.CancelWorkOrder(ctx, id, reason)
}

func (r *CMMSRouter) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	return r.adapter.UsePartInWorkOrder(ctx, workOrderID, partID, quantity, userID)
}

// ── Spare Parts ──────────────────────────────────────────────────

func (r *CMMSRouter) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	return r.adapter.CreateSparePart(ctx, part)
}

func (r *CMMSRouter) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	return r.adapter.GetSpareParts(ctx, filters)
}

func (r *CMMSRouter) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	return r.adapter.GetSparePart(ctx, id)
}

func (r *CMMSRouter) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateSparePart(ctx, id, updates)
}

func (r *CMMSRouter) DeleteSparePart(ctx context.Context, id string) error {
	return r.adapter.DeleteSparePart(ctx, id)
}

func (r *CMMSRouter) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	return r.adapter.GetLowStockParts(ctx)
}

func (r *CMMSRouter) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	return r.adapter.UpdateSparePartStock(ctx, id, quantity)
}

// ── Maintenance Schedules ────────────────────────────────────────

func (r *CMMSRouter) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	return r.adapter.CreateMaintenanceSchedule(ctx, schedule)
}

func (r *CMMSRouter) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	return r.adapter.GetMaintenanceSchedules(ctx, filters)
}

func (r *CMMSRouter) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	return r.adapter.GetMaintenanceSchedule(ctx, id)
}

func (r *CMMSRouter) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateMaintenanceSchedule(ctx, id, updates)
}

func (r *CMMSRouter) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	return r.adapter.DeleteMaintenanceSchedule(ctx, id)
}

func (r *CMMSRouter) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	return r.adapter.GetDueSchedules(ctx)
}

func (r *CMMSRouter) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	return r.adapter.CompleteMaintenanceSchedule(ctx, id)
}

// ── SLA ──────────────────────────────────────────────────────────

func (r *CMMSRouter) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	return r.adapter.GetSLAConfig(ctx, priority)
}

func (r *CMMSRouter) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	return r.adapter.GetAllSLAConfigs(ctx)
}

func (r *CMMSRouter) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	return r.adapter.UpdateSLAConfig(ctx, priority, responseTimeMinutes, resolutionTimeMinutes)
}

// ── Technicians ──────────────────────────────────────────────────

func (r *CMMSRouter) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	return r.adapter.GetTechnicianWorkload(ctx, userID)
}

func (r *CMMSRouter) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	return r.adapter.GetAllTechnicianWorkloads(ctx)
}

func (r *CMMSRouter) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	return r.adapter.GetTechnicianMonthlyStats(ctx, userID)
}

func (r *CMMSRouter) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	return r.adapter.UpdateTechnicianSkills(ctx, userID, skills, certifications)
}

// ── Reports ──────────────────────────────────────────────────────

func (r *CMMSRouter) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	return r.adapter.GetMaintenanceReport(ctx)
}

func (r *CMMSRouter) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	return r.adapter.GetSLAComplianceReport(ctx)
}

// ── Technician Site Assignments ──────────────────────────────────

func (r *CMMSRouter) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	return r.adapter.CreateTechnicianSiteAssignment(ctx, assignment)
}

func (r *CMMSRouter) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	return r.adapter.GetTechnicianSiteAssignments(ctx, filters)
}

func (r *CMMSRouter) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateTechnicianSiteAssignment(ctx, id, updates)
}

func (r *CMMSRouter) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	return r.adapter.DeleteTechnicianSiteAssignment(ctx, id)
}

// ── Sites ────────────────────────────────────────────────────────

func (r *CMMSRouter) GetSites(ctx context.Context, filters map[string]interface{}) ([]models.Site, error) {
	return r.adapter.GetSites(ctx, filters)
}

func (r *CMMSRouter) GetSite(ctx context.Context, id string) (*models.Site, error) {
	return r.adapter.GetSite(ctx, id)
}

func (r *CMMSRouter) CreateSite(ctx context.Context, site *models.Site) error {
	return r.adapter.CreateSite(ctx, site)
}

func (r *CMMSRouter) UpdateSite(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateSite(ctx, id, updates)
}

func (r *CMMSRouter) DeleteSite(ctx context.Context, id string) error {
	return r.adapter.DeleteSite(ctx, id)
}

// ── Spare Part Categories ────────────────────────────────────────

func (r *CMMSRouter) GetCategories(ctx context.Context) ([]models.SparePartCategory, error) {
	return r.adapter.GetCategories(ctx)
}

func (r *CMMSRouter) CreateCategory(ctx context.Context, cat *models.SparePartCategory) error {
	return r.adapter.CreateCategory(ctx, cat)
}

func (r *CMMSRouter) UpdateCategory(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateCategory(ctx, id, updates)
}

func (r *CMMSRouter) DeleteCategory(ctx context.Context, id string) error {
	return r.adapter.DeleteCategory(ctx, id)
}

// ── Work Requests (WO-4.1.1) ────────────────────────────────────

func (r *CMMSRouter) CreateWorkRequest(ctx context.Context, req *models.WorkRequest) error {
	return r.adapter.CreateWorkRequest(ctx, req)
}

func (r *CMMSRouter) GetWorkRequests(ctx context.Context, filters map[string]interface{}) ([]models.WorkRequest, error) {
	return r.adapter.GetWorkRequests(ctx, filters)
}

func (r *CMMSRouter) GetWorkRequest(ctx context.Context, id string) (*models.WorkRequest, error) {
	return r.adapter.GetWorkRequest(ctx, id)
}

func (r *CMMSRouter) ApproveWorkRequest(ctx context.Context, id, approvedBy string) error {
	return r.adapter.ApproveWorkRequest(ctx, id, approvedBy)
}

func (r *CMMSRouter) RejectWorkRequest(ctx context.Context, id, rejectedBy, reason string) error {
	return r.adapter.RejectWorkRequest(ctx, id, rejectedBy, reason)
}

func (r *CMMSRouter) ConvertWorkRequestToWO(ctx context.Context, requestID, workOrderID string) error {
	return r.adapter.ConvertWorkRequestToWO(ctx, requestID, workOrderID)
}

// ── WorkOrder ↔ Alert (Many-to-Many) — DM-1.3.1 ────────────────

func (r *CMMSRouter) LinkAlertToWorkOrder(ctx context.Context, workOrderID, alertID, userID string) error {
	return r.adapter.LinkAlertToWorkOrder(ctx, workOrderID, alertID, userID)
}

func (r *CMMSRouter) UnlinkAlertFromWorkOrder(ctx context.Context, workOrderID, alertID string) error {
	return r.adapter.UnlinkAlertFromWorkOrder(ctx, workOrderID, alertID)
}

func (r *CMMSRouter) GetAlertsForWorkOrder(ctx context.Context, workOrderID string) ([]models.WorkOrderAlert, error) {
	return r.adapter.GetAlertsForWorkOrder(ctx, workOrderID)
}

func (r *CMMSRouter) GetWorkOrdersForAlert(ctx context.Context, alertID string) ([]models.WorkOrderAlert, error) {
	return r.adapter.GetWorkOrdersForAlert(ctx, alertID)
}

// ── Vendors (INV-7.2.1) ──────────────────────────────────────────

func (r *CMMSRouter) CreateVendor(ctx context.Context, vendor *models.Vendor) error {
	return r.adapter.CreateVendor(ctx, vendor)
}

func (r *CMMSRouter) GetVendors(ctx context.Context, filters map[string]interface{}) ([]models.Vendor, error) {
	return r.adapter.GetVendors(ctx, filters)
}

func (r *CMMSRouter) GetVendor(ctx context.Context, id string) (*models.Vendor, error) {
	return r.adapter.GetVendor(ctx, id)
}

func (r *CMMSRouter) UpdateVendor(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.adapter.UpdateVendor(ctx, id, updates)
}

func (r *CMMSRouter) DeleteVendor(ctx context.Context, id string) error {
	return r.adapter.DeleteVendor(ctx, id)
}

// ── Mobile ───────────────────────────────────────────────────────

func (r *CMMSRouter) SavePushToken(ctx context.Context, userID, token, platform string) error {
	return r.adapter.SavePushToken(ctx, userID, token, platform)
}
