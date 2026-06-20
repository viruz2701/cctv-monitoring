package cmms

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ErrNotImplemented возвращается AtlasAdapter для методов, которые ещё
// не реализованы в интеграции с внешним CMMS API.
var ErrNotImplemented = errors.New("atlas adapter: method not implemented")

// AtlasAdapter — реализация CMMSAdapter для внешнего CMMS API (Atlas).
// На текущем этапе все методы возвращают ErrNotImplemented — это задел
// на будущую интеграцию с внешней системой управления ТО.
type AtlasAdapter struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAtlasAdapter создаёт новый экземпляр AtlasAdapter.
func NewAtlasAdapter(baseURL, apiKey string) *AtlasAdapter {
	return &AtlasAdapter{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthCheck проверяет доступность внешнего CMMS API.
func (a *AtlasAdapter) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("atlas health check: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("atlas health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("atlas health check: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// ── Work Orders ──────────────────────────────────────────────────

func (a *AtlasAdapter) CreateWorkOrder(_ context.Context, _ *models.WorkOrder) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) GetWorkOrders(_ context.Context, _ map[string]interface{}) ([]models.WorkOrder, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetWorkOrder(_ context.Context, _ string) (*models.WorkOrder, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateWorkOrder(_ context.Context, _ string, _ map[string]interface{}) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) AssignWorkOrder(_ context.Context, _, _ string) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) StartWorkOrder(_ context.Context, _ string) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) CompleteWorkOrder(_ context.Context, _ string, _ string, _ []string, _ []models.PartUsage, _ string) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) CancelWorkOrder(_ context.Context, _, _ string) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) UsePartInWorkOrder(_ context.Context, _, _ string, _ int, _ string) error {
	return ErrNotImplemented
}

// ── Spare Parts ──────────────────────────────────────────────────

func (a *AtlasAdapter) CreateSparePart(_ context.Context, _ *models.SparePart) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) GetSpareParts(_ context.Context, _ map[string]interface{}) ([]models.SparePart, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetSparePart(_ context.Context, _ string) (*models.SparePart, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateSparePart(_ context.Context, _ string, _ map[string]interface{}) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) DeleteSparePart(_ context.Context, _ string) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) GetLowStockParts(_ context.Context) ([]models.SparePart, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateSparePartStock(_ context.Context, _ string, _ int) error {
	return ErrNotImplemented
}

// ── Maintenance Schedules ────────────────────────────────────────

func (a *AtlasAdapter) CreateMaintenanceSchedule(_ context.Context, _ *models.MaintenanceSchedule) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) GetMaintenanceSchedules(_ context.Context, _ map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetMaintenanceSchedule(_ context.Context, _ string) (*models.MaintenanceSchedule, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateMaintenanceSchedule(_ context.Context, _ string, _ map[string]interface{}) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) DeleteMaintenanceSchedule(_ context.Context, _ string) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) GetDueSchedules(_ context.Context) ([]models.MaintenanceSchedule, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) CompleteMaintenanceSchedule(_ context.Context, _ string) error {
	return ErrNotImplemented
}

// ── SLA ──────────────────────────────────────────────────────────

func (a *AtlasAdapter) GetSLAConfig(_ context.Context, _ string) (*models.SLAConfig, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetAllSLAConfigs(_ context.Context) ([]models.SLAConfig, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateSLAConfig(_ context.Context, _ string, _, _ int) error {
	return ErrNotImplemented
}

// ── Technicians ──────────────────────────────────────────────────

func (a *AtlasAdapter) GetTechnicianWorkload(_ context.Context, _ string) (*models.TechnicianWorkload, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetAllTechnicianWorkloads(_ context.Context) ([]models.TechnicianWorkload, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetTechnicianMonthlyStats(_ context.Context, _ string) (*models.TechnicianMonthlyStats, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateTechnicianSkills(_ context.Context, _ string, _ []string, _ []string) error {
	return ErrNotImplemented
}

// ── Reports ──────────────────────────────────────────────────────

func (a *AtlasAdapter) GetMaintenanceReport(_ context.Context) ([]models.MaintenanceReport, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) GetSLAComplianceReport(_ context.Context) ([]models.SLAComplianceReport, error) {
	return nil, ErrNotImplemented
}

// ── Technician Site Assignments ──────────────────────────────────

func (a *AtlasAdapter) CreateTechnicianSiteAssignment(_ context.Context, _ *models.TechnicianSiteAssignment) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) GetTechnicianSiteAssignments(_ context.Context, _ map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	return nil, ErrNotImplemented
}

func (a *AtlasAdapter) UpdateTechnicianSiteAssignment(_ context.Context, _ string, _ map[string]interface{}) error {
	return ErrNotImplemented
}

func (a *AtlasAdapter) DeleteTechnicianSiteAssignment(_ context.Context, _ string) error {
	return ErrNotImplemented
}

// ── Mobile ───────────────────────────────────────────────────────

func (a *AtlasAdapter) SavePushToken(_ context.Context, _, _, _ string) error {
	return ErrNotImplemented
}
