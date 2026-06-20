package cmms

import (
	"context"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
)

// InternalAdapter — реализация CMMSAdapter, делегирующая все вызовы
// напрямую в существующий слой db.DB. Это паттерн Headless CMMS:
// существующая Internal CMMS используется как адаптер без переписывания.
type InternalAdapter struct {
	db *db.DB
}

// NewInternalAdapter создаёт адаптер, оборачивающий экземпляр db.DB.
func NewInternalAdapter(database *db.DB) *InternalAdapter {
	return &InternalAdapter{db: database}
}

// ── Work Orders ──────────────────────────────────────────────────

func (a *InternalAdapter) CreateWorkOrder(_ context.Context, wo *models.WorkOrder) error {
	return a.db.CreateWorkOrder(wo)
}

func (a *InternalAdapter) GetWorkOrders(_ context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	return a.db.GetWorkOrders(filters)
}

func (a *InternalAdapter) GetWorkOrder(_ context.Context, id string) (*models.WorkOrder, error) {
	return a.db.GetWorkOrder(id)
}

func (a *InternalAdapter) UpdateWorkOrder(_ context.Context, id string, updates map[string]interface{}) error {
	return a.db.UpdateWorkOrder(id, updates)
}

func (a *InternalAdapter) AssignWorkOrder(_ context.Context, id, userID string) error {
	return a.db.AssignWorkOrder(id, userID)
}

func (a *InternalAdapter) StartWorkOrder(_ context.Context, id string) error {
	return a.db.StartWorkOrder(id)
}

func (a *InternalAdapter) CompleteWorkOrder(_ context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	return a.db.CompleteWorkOrder(id, notes, photos, parts, userID)
}

func (a *InternalAdapter) CancelWorkOrder(_ context.Context, id, reason string) error {
	return a.db.CancelWorkOrder(id, reason)
}

func (a *InternalAdapter) UsePartInWorkOrder(_ context.Context, workOrderID, partID string, quantity int, userID string) error {
	return a.db.UsePartInWorkOrder(workOrderID, partID, quantity, userID)
}

// ── Spare Parts ──────────────────────────────────────────────────

func (a *InternalAdapter) CreateSparePart(_ context.Context, part *models.SparePart) error {
	return a.db.CreateSparePart(part)
}

func (a *InternalAdapter) GetSpareParts(_ context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	return a.db.GetSpareParts(filters)
}

func (a *InternalAdapter) GetSparePart(_ context.Context, id string) (*models.SparePart, error) {
	return a.db.GetSparePart(id)
}

func (a *InternalAdapter) UpdateSparePart(_ context.Context, id string, updates map[string]interface{}) error {
	return a.db.UpdateSparePart(id, updates)
}

func (a *InternalAdapter) DeleteSparePart(_ context.Context, id string) error {
	return a.db.DeleteSparePart(id)
}

func (a *InternalAdapter) GetLowStockParts(_ context.Context) ([]models.SparePart, error) {
	return a.db.GetLowStockParts()
}

func (a *InternalAdapter) UpdateSparePartStock(_ context.Context, id string, quantity int) error {
	return a.db.UpdateSparePartStock(id, quantity)
}

// ── Maintenance Schedules ────────────────────────────────────────

func (a *InternalAdapter) CreateMaintenanceSchedule(_ context.Context, schedule *models.MaintenanceSchedule) error {
	return a.db.CreateMaintenanceSchedule(schedule)
}

func (a *InternalAdapter) GetMaintenanceSchedules(_ context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	return a.db.GetMaintenanceSchedules(filters)
}

func (a *InternalAdapter) GetMaintenanceSchedule(_ context.Context, id string) (*models.MaintenanceSchedule, error) {
	return a.db.GetMaintenanceSchedule(id)
}

func (a *InternalAdapter) UpdateMaintenanceSchedule(_ context.Context, id string, updates map[string]interface{}) error {
	return a.db.UpdateMaintenanceSchedule(id, updates)
}

func (a *InternalAdapter) DeleteMaintenanceSchedule(_ context.Context, id string) error {
	return a.db.DeleteMaintenanceSchedule(id)
}

func (a *InternalAdapter) GetDueSchedules(_ context.Context) ([]models.MaintenanceSchedule, error) {
	return a.db.GetDueSchedules()
}

func (a *InternalAdapter) CompleteMaintenanceSchedule(_ context.Context, id string) error {
	return a.db.CompleteMaintenanceSchedule(id)
}

// ── SLA ──────────────────────────────────────────────────────────

func (a *InternalAdapter) GetSLAConfig(_ context.Context, priority string) (*models.SLAConfig, error) {
	return a.db.GetSLAConfig(priority)
}

func (a *InternalAdapter) GetAllSLAConfigs(_ context.Context) ([]models.SLAConfig, error) {
	return a.db.GetAllSLAConfigs()
}

func (a *InternalAdapter) UpdateSLAConfig(_ context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	return a.db.UpdateSLAConfig(priority, responseTimeMinutes, resolutionTimeMinutes)
}

// ── Technicians ──────────────────────────────────────────────────

func (a *InternalAdapter) GetTechnicianWorkload(_ context.Context, userID string) (*models.TechnicianWorkload, error) {
	return a.db.GetTechnicianWorkload(userID)
}

func (a *InternalAdapter) GetAllTechnicianWorkloads(_ context.Context) ([]models.TechnicianWorkload, error) {
	return a.db.GetAllTechnicianWorkloads()
}

func (a *InternalAdapter) GetTechnicianMonthlyStats(_ context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	return a.db.GetTechnicianMonthlyStats(userID)
}

func (a *InternalAdapter) UpdateTechnicianSkills(_ context.Context, userID string, skills []string, certifications []string) error {
	return a.db.UpdateTechnicianSkills(userID, skills, certifications)
}

// ── Reports ──────────────────────────────────────────────────────

func (a *InternalAdapter) GetMaintenanceReport(_ context.Context) ([]models.MaintenanceReport, error) {
	return a.db.GetMaintenanceReport()
}

func (a *InternalAdapter) GetSLAComplianceReport(_ context.Context) ([]models.SLAComplianceReport, error) {
	return a.db.GetSLAComplianceReport()
}

// ── Technician Site Assignments ──────────────────────────────────

func (a *InternalAdapter) CreateTechnicianSiteAssignment(_ context.Context, assignment *models.TechnicianSiteAssignment) error {
	return a.db.CreateTechnicianSiteAssignment(assignment)
}

func (a *InternalAdapter) GetTechnicianSiteAssignments(_ context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	return a.db.GetTechnicianSiteAssignments(filters)
}

func (a *InternalAdapter) UpdateTechnicianSiteAssignment(_ context.Context, id string, updates map[string]interface{}) error {
	return a.db.UpdateTechnicianSiteAssignment(id, updates)
}

func (a *InternalAdapter) DeleteTechnicianSiteAssignment(_ context.Context, id string) error {
	return a.db.DeleteTechnicianSiteAssignment(id)
}

// ── Mobile ───────────────────────────────────────────────────────

func (a *InternalAdapter) SavePushToken(_ context.Context, userID, token, platform string) error {
	return a.db.SavePushToken(userID, token, platform)
}
