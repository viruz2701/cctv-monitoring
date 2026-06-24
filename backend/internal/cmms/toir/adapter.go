package toir

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/models"
)

// Adapter — реализация cmms.CMMSAdapter для 1С:ТОИР REST API.
// Аутентификация: Basic Auth. Соответствует 152-ФЗ (данные в РФ).
type Adapter struct {
	client        *Client
	fallbackQueue *cmms.FallbackQueue
	logger        *slog.Logger
}

// AdapterConfig — параметры для 1С:ТОИР адаптера.
type AdapterConfig struct {
	BaseURL     string
	Username    string
	Password    string
	FallbackDir string
	Logger      *slog.Logger
}

// NewAdapter создаёт 1С:ТОИР адаптер.
func NewAdapter(cfg AdapterConfig) (*Adapter, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	client, err := NewClient(ClientConfig{
		BaseURL:  cfg.BaseURL,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("toir adapter: create client: %w", err)
	}

	fq, err := cmms.NewFallbackQueue(cfg.FallbackDir, 10, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("toir adapter: create fallback queue: %w", err)
	}

	return &Adapter{
		client:        client,
		fallbackQueue: fq,
		logger:        cfg.Logger,
	}, nil
}

// HealthCheck проверяет доступность 1С:ТОИР.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	resp, err := a.client.getRaw(ctx, pathHealth)
	if err != nil {
		return fmt.Errorf("toir health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("toir health check: HTTP %d", resp.StatusCode)
	}
	return nil
}

// SyncAsset синхронизирует устройство с 1С:ТОИР как основное средство.
func (a *Adapter) SyncAsset(ctx context.Context, deviceID string, assetData map[string]interface{}) error {
	err := a.client.post(ctx, pathAssets, assetData, nil)
	if err != nil {
		a.logger.Warn("toir: sync asset failed, enqueuing", "device_id", deviceID, "error", err)
		_ = a.fallbackQueue.Enqueue("sync_asset", map[string]interface{}{
			"device_id":  deviceID,
			"asset_data": assetData,
		})
		return nil
	}
	return nil
}

// ── Work Orders ──────────────────────────────────────────────────

func (a *Adapter) CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error {
	body := toWorkOrderTOIRBody(wo)
	err := a.client.post(ctx, pathWorkOrders, body, nil)
	if err != nil {
		a.logger.Warn("toir: create work order failed, enqueuing", "id", wo.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_wo", wo)
		return nil
	}
	return nil
}

func (a *Adapter) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	path := buildQueryPath(pathWorkOrders, filters)
	var resp toirResponse
	if err := a.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("toir: get work orders: %w", err)
	}
	var result []models.WorkOrder
	for _, raw := range resp.Data {
		result = append(result, toWorkOrder(raw))
	}
	return result, nil
}

func (a *Adapter) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	var resp toirSingleResponse
	if err := a.client.get(ctx, pathWorkOrders+"/"+id, &resp); err != nil {
		return nil, fmt.Errorf("toir: get work order %s: %w", id, err)
	}
	wo := toWorkOrder(resp.Data)
	return &wo, nil
}

func (a *Adapter) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, pathWorkOrders+"/"+id, updates, nil)
	if err != nil {
		a.logger.Warn("toir: update work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_wo", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) AssignWorkOrder(ctx context.Context, id, userID string) error {
	body := map[string]string{"assigned_to": userID}
	err := a.client.put(ctx, pathWorkOrders+"/"+id+"/assign", body, nil)
	if err != nil {
		a.logger.Warn("toir: assign work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("assign_wo", map[string]string{"id": id, "user_id": userID})
		return nil
	}
	return nil
}

func (a *Adapter) StartWorkOrder(ctx context.Context, id string) error {
	err := a.client.post(ctx, pathWorkOrders+"/"+id+"/start", nil, nil)
	if err != nil {
		a.logger.Warn("toir: start work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("start_wo", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	body := map[string]interface{}{
		"notes":        notes,
		"photos":       photos,
		"parts":        parts,
		"completed_by": userID,
	}
	err := a.client.post(ctx, pathWorkOrders+"/"+id+"/complete", body, nil)
	if err != nil {
		a.logger.Warn("toir: complete work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_wo", body)
		return nil
	}
	return nil
}

func (a *Adapter) CancelWorkOrder(ctx context.Context, id, reason string) error {
	body := map[string]string{"reason": reason}
	err := a.client.post(ctx, pathWorkOrders+"/"+id+"/cancel", body, nil)
	if err != nil {
		a.logger.Warn("toir: cancel work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("cancel_wo", map[string]string{"id": id, "reason": reason})
		return nil
	}
	return nil
}

func (a *Adapter) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	body := map[string]interface{}{
		"part_id":  partID,
		"quantity": quantity,
		"user_id":  userID,
	}
	err := a.client.post(ctx, pathWorkOrders+"/"+workOrderID+"/parts", body, nil)
	if err != nil {
		a.logger.Warn("toir: use part in work order failed, enqueuing", "wo_id", workOrderID, "error", err)
		_ = a.fallbackQueue.Enqueue("use_part", body)
		return nil
	}
	return nil
}

// ── Spare Parts ──────────────────────────────────────────────────

func (a *Adapter) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	body := toSparePartTOIRBody(part)
	err := a.client.post(ctx, pathSpareParts, body, nil)
	if err != nil {
		a.logger.Warn("toir: create spare part failed, enqueuing", "id", part.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_part", part)
		return nil
	}
	return nil
}

func (a *Adapter) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	path := buildQueryPath(pathSpareParts, filters)
	var resp toirResponse
	if err := a.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("toir: get spare parts: %w", err)
	}
	var result []models.SparePart
	for _, raw := range resp.Data {
		result = append(result, toSparePart(raw))
	}
	return result, nil
}

func (a *Adapter) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	var resp toirSingleResponse
	if err := a.client.get(ctx, pathSpareParts+"/"+id, &resp); err != nil {
		return nil, fmt.Errorf("toir: get spare part %s: %w", id, err)
	}
	sp := toSparePart(resp.Data)
	return &sp, nil
}

func (a *Adapter) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, pathSpareParts+"/"+id, updates, nil)
	if err != nil {
		a.logger.Warn("toir: update spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_part", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteSparePart(ctx context.Context, id string) error {
	err := a.client.delete(ctx, pathSpareParts+"/"+id)
	if err != nil {
		a.logger.Warn("toir: delete spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_part", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	var resp toirResponse
	if err := a.client.get(ctx, pathSpareParts+"/low-stock", &resp); err != nil {
		return nil, fmt.Errorf("toir: get low stock parts: %w", err)
	}
	var result []models.SparePart
	for _, raw := range resp.Data {
		result = append(result, toSparePart(raw))
	}
	return result, nil
}

func (a *Adapter) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	body := map[string]int{"quantity": quantity}
	err := a.client.put(ctx, pathSpareParts+"/"+id+"/stock", body, nil)
	if err != nil {
		a.logger.Warn("toir: update spare part stock failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_stock", map[string]interface{}{"id": id, "quantity": quantity})
		return nil
	}
	return nil
}

// ── Maintenance Schedules ────────────────────────────────────────

func (a *Adapter) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	body := toMaintenanceScheduleTOIRBody(schedule)
	err := a.client.post(ctx, pathMaintenanceSchedules, body, nil)
	if err != nil {
		a.logger.Warn("toir: create schedule failed, enqueuing", "id", schedule.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_schedule", schedule)
		return nil
	}
	return nil
}

func (a *Adapter) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	path := buildQueryPath(pathMaintenanceSchedules, filters)
	var resp toirResponse
	if err := a.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("toir: get schedules: %w", err)
	}
	var result []models.MaintenanceSchedule
	for _, raw := range resp.Data {
		result = append(result, toMaintenanceSchedule(raw))
	}
	return result, nil
}

func (a *Adapter) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	var resp toirSingleResponse
	if err := a.client.get(ctx, pathMaintenanceSchedules+"/"+id, &resp); err != nil {
		return nil, fmt.Errorf("toir: get schedule %s: %w", id, err)
	}
	ms := toMaintenanceSchedule(resp.Data)
	return &ms, nil
}

func (a *Adapter) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, pathMaintenanceSchedules+"/"+id, updates, nil)
	if err != nil {
		a.logger.Warn("toir: update schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_schedule", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	err := a.client.delete(ctx, pathMaintenanceSchedules+"/"+id)
	if err != nil {
		a.logger.Warn("toir: delete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	var resp toirResponse
	if err := a.client.get(ctx, pathMaintenanceSchedules+"/due", &resp); err != nil {
		return nil, fmt.Errorf("toir: get due schedules: %w", err)
	}
	var result []models.MaintenanceSchedule
	for _, raw := range resp.Data {
		result = append(result, toMaintenanceSchedule(raw))
	}
	return result, nil
}

func (a *Adapter) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	err := a.client.post(ctx, pathMaintenanceSchedules+"/"+id+"/complete", nil, nil)
	if err != nil {
		a.logger.Warn("toir: complete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── SLA ──────────────────────────────────────────────────────────

func (a *Adapter) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	var resp toirSingleResponse
	if err := a.client.get(ctx, pathSLAConfig+"/"+priority, &resp); err != nil {
		return nil, fmt.Errorf("toir: get sla config %s: %w", priority, err)
	}
	cfg := toSLAConfig(resp.Data)
	return &cfg, nil
}

func (a *Adapter) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	var resp toirResponse
	if err := a.client.get(ctx, pathSLAConfig, &resp); err != nil {
		return nil, fmt.Errorf("toir: get all sla configs: %w", err)
	}
	var result []models.SLAConfig
	for _, raw := range resp.Data {
		result = append(result, toSLAConfig(raw))
	}
	return result, nil
}

func (a *Adapter) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	body := map[string]int{
		"response_time_minutes":   responseTimeMinutes,
		"resolution_time_minutes": resolutionTimeMinutes,
	}
	err := a.client.put(ctx, pathSLAConfig+"/"+priority, body, nil)
	if err != nil {
		a.logger.Warn("toir: update sla config failed, enqueuing", "priority", priority, "error", err)
		_ = a.fallbackQueue.Enqueue("update_sla", map[string]interface{}{"priority": priority, "config": body})
		return nil
	}
	return nil
}

// ── Technicians ──────────────────────────────────────────────────

func (a *Adapter) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	var resp toirSingleResponse
	if err := a.client.get(ctx, pathTechnicians+"/"+userID+"/workload", &resp); err != nil {
		return nil, fmt.Errorf("toir: get technician workload %s: %w", userID, err)
	}
	wl := toTechnicianWorkload(resp.Data)
	return &wl, nil
}

func (a *Adapter) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	var resp toirResponse
	if err := a.client.get(ctx, pathTechnicians+"/workload", &resp); err != nil {
		return nil, fmt.Errorf("toir: get all technician workloads: %w", err)
	}
	var result []models.TechnicianWorkload
	for _, raw := range resp.Data {
		result = append(result, toTechnicianWorkload(raw))
	}
	return result, nil
}

func (a *Adapter) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	var resp toirSingleResponse
	if err := a.client.get(ctx, pathTechnicians+"/"+userID+"/monthly-stats", &resp); err != nil {
		return nil, fmt.Errorf("toir: get technician monthly stats %s: %w", userID, err)
	}
	stats := toTechnicianMonthlyStats(resp.Data)
	return &stats, nil
}

func (a *Adapter) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	body := map[string]interface{}{
		"skills":         skills,
		"certifications": certifications,
	}
	err := a.client.put(ctx, pathTechnicians+"/"+userID+"/skills", body, nil)
	if err != nil {
		a.logger.Warn("toir: update technician skills failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("update_skills", map[string]interface{}{"user_id": userID, "body": body})
		return nil
	}
	return nil
}

// ── Reports ──────────────────────────────────────────────────────

func (a *Adapter) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	var resp toirResponse
	if err := a.client.get(ctx, pathReports+"/maintenance", &resp); err != nil {
		return nil, fmt.Errorf("toir: get maintenance report: %w", err)
	}
	var result []models.MaintenanceReport
	for _, raw := range resp.Data {
		result = append(result, toMaintenanceReport(raw))
	}
	return result, nil
}

func (a *Adapter) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	var resp toirResponse
	if err := a.client.get(ctx, pathReports+"/sla-compliance", &resp); err != nil {
		return nil, fmt.Errorf("toir: get sla compliance report: %w", err)
	}
	var result []models.SLAComplianceReport
	for _, raw := range resp.Data {
		result = append(result, toSLAComplianceReport(raw))
	}
	return result, nil
}

// ── Technician Site Assignments ──────────────────────────────────

func (a *Adapter) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	body := toTechnicianAssignmentTOIRBody(assignment)
	err := a.client.post(ctx, pathTechnicianAssignments, body, nil)
	if err != nil {
		a.logger.Warn("toir: create assignment failed, enqueuing", "id", assignment.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_assignment", assignment)
		return nil
	}
	return nil
}

func (a *Adapter) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	path := buildQueryPath(pathTechnicianAssignments, filters)
	var resp toirResponse
	if err := a.client.get(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("toir: get assignments: %w", err)
	}
	var result []models.TechnicianSiteAssignment
	for _, raw := range resp.Data {
		result = append(result, toTechnicianAssignment(raw))
	}
	return result, nil
}

func (a *Adapter) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, pathTechnicianAssignments+"/"+id, updates, nil)
	if err != nil {
		a.logger.Warn("toir: update assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_assignment", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	err := a.client.delete(ctx, pathTechnicianAssignments+"/"+id)
	if err != nil {
		a.logger.Warn("toir: delete assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_assignment", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── Mobile ───────────────────────────────────────────────────────

func (a *Adapter) SavePushToken(ctx context.Context, userID, token, platform string) error {
	body := map[string]string{
		"user_id":  userID,
		"token":    token,
		"platform": platform,
	}
	err := a.client.post(ctx, pathMobilePushToken, body, nil)
	if err != nil {
		a.logger.Warn("toir: save push token failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("save_push_token", body)
		return nil
	}
	return nil
}

// ── Fallback Queue Management ────────────────────────────────────

// RetryFallback повторяет все операции из fallback-очереди.
func (a *Adapter) RetryFallback(ctx context.Context) (success, failed int) {
	return a.fallbackQueue.RetryAll(ctx, func(ctx context.Context, entry cmms.FallbackQueueEntry) error {
		switch entry.Method {
		case "create_wo":
			var wo models.WorkOrder
			if err := json.Unmarshal(entry.Payload, &wo); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, pathWorkOrders, toWorkOrderTOIRBody(&wo), nil)
		case "update_wo":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			id, _ := payload["id"].(string)
			updates, _ := payload["updates"].(map[string]interface{})
			return a.client.put(ctx, pathWorkOrders+"/"+id, updates, nil)
		case "sync_asset":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			assetData, _ := payload["asset_data"].(map[string]interface{})
			return a.client.post(ctx, pathAssets, assetData, nil)
		case "complete_schedule":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, pathMaintenanceSchedules+"/"+payload["id"]+"/complete", nil, nil)
		case "save_push_token":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, pathMobilePushToken, payload, nil)
		default:
			return fmt.Errorf("fallback: unsupported retry method: %s", entry.Method)
		}
	})
}

// FallbackQueueSize возвращает размер очереди.
func (a *Adapter) FallbackQueueSize() int {
	count, _ := a.fallbackQueue.Len()
	return count
}

// ── Sites ────────────────────────────────────────────────────────

func (a *Adapter) GetSites(_ context.Context, _ map[string]interface{}) ([]models.Site, error) {
	return nil, fmt.Errorf("get sites not implemented for TOIR adapter")
}

func (a *Adapter) GetSite(_ context.Context, _ string) (*models.Site, error) {
	return nil, fmt.Errorf("get site not implemented for TOIR adapter")
}

func (a *Adapter) CreateSite(_ context.Context, _ *models.Site) error {
	return fmt.Errorf("create site not implemented for TOIR adapter")
}

func (a *Adapter) UpdateSite(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("update site not implemented for TOIR adapter")
}

func (a *Adapter) DeleteSite(_ context.Context, _ string) error {
	return fmt.Errorf("delete site not implemented for TOIR adapter")
}

// ── Spare Part Categories ────────────────────────────────────────

func (a *Adapter) GetCategories(_ context.Context) ([]models.SparePartCategory, error) {
	return nil, fmt.Errorf("get categories not implemented for TOIR adapter")
}

func (a *Adapter) CreateCategory(_ context.Context, _ *models.SparePartCategory) error {
	return fmt.Errorf("create category not implemented for TOIR adapter")
}

func (a *Adapter) UpdateCategory(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("update category not implemented for TOIR adapter")
}

func (a *Adapter) DeleteCategory(_ context.Context, _ string) error {
	return fmt.Errorf("delete category not implemented for TOIR adapter")
}

// ── Work Requests (not supported) ───────────────────────────────

func (a *Adapter) CreateWorkRequest(_ context.Context, _ *models.WorkRequest) error {
	return fmt.Errorf("work requests not supported for TOIR adapter")
}

func (a *Adapter) GetWorkRequests(_ context.Context, _ map[string]interface{}) ([]models.WorkRequest, error) {
	return nil, fmt.Errorf("work requests not supported for TOIR adapter")
}

func (a *Adapter) GetWorkRequest(_ context.Context, _ string) (*models.WorkRequest, error) {
	return nil, fmt.Errorf("work requests not supported for TOIR adapter")
}

func (a *Adapter) ApproveWorkRequest(_ context.Context, _, _ string) error {
	return fmt.Errorf("work requests not supported for TOIR adapter")
}

func (a *Adapter) RejectWorkRequest(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("work requests not supported for TOIR adapter")
}

func (a *Adapter) ConvertWorkRequestToWO(_ context.Context, _, _ string) error {
	return fmt.Errorf("work requests not supported for TOIR adapter")
}

// ── WorkOrder ↔ Alert (DM-1.3.1 — not supported for external CMMS) ─

func (a *Adapter) LinkAlertToWorkOrder(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("work order alerts not supported for TOIR adapter")
}

func (a *Adapter) UnlinkAlertFromWorkOrder(_ context.Context, _, _ string) error {
	return fmt.Errorf("work order alerts not supported for TOIR adapter")
}

func (a *Adapter) GetAlertsForWorkOrder(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, fmt.Errorf("work order alerts not supported for TOIR adapter")
}

func (a *Adapter) GetWorkOrdersForAlert(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, fmt.Errorf("work order alerts not supported for TOIR adapter")
}

// ── Vendors (INV-7.2.1 — not supported for external CMMS) ────────

func (a *Adapter) CreateVendor(_ context.Context, _ *models.Vendor) error {
	return fmt.Errorf("vendors not supported for TOIR adapter")
}

func (a *Adapter) GetVendors(_ context.Context, _ map[string]interface{}) ([]models.Vendor, error) {
	return nil, fmt.Errorf("vendors not supported for TOIR adapter")
}

func (a *Adapter) GetVendor(_ context.Context, _ string) (*models.Vendor, error) {
	return nil, fmt.Errorf("vendors not supported for TOIR adapter")
}

func (a *Adapter) UpdateVendor(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("vendors not supported for TOIR adapter")
}

func (a *Adapter) DeleteVendor(_ context.Context, _ string) error {
	return fmt.Errorf("vendors not supported for TOIR adapter")
}
