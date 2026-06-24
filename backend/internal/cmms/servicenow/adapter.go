package servicenow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/models"
)

// Adapter — реализация cmms.CMMSAdapter для ServiceNow Table API.
// Использует OAuth2 или Basic Auth. Поддерживает fallback-очередь.
type Adapter struct {
	client        *Client
	fallbackQueue *cmms.FallbackQueue
	username      string
	password      string
	logger        *slog.Logger
}

// AdapterConfig — параметры для ServiceNow адаптера.
type AdapterConfig struct {
	InstanceURL  string
	ClientID     string
	ClientSecret string
	TokenURL     string
	Username     string
	Password     string
	FallbackDir  string
	Logger       *slog.Logger
}

// NewAdapter создаёт ServiceNow адаптер.
func NewAdapter(cfg AdapterConfig) (*Adapter, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	client, err := NewClient(ClientConfig{
		InstanceURL:  cfg.InstanceURL,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Username:     cfg.Username,
		Password:     cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("servicenow adapter: create client: %w", err)
	}

	fq, err := cmms.NewFallbackQueue(cfg.FallbackDir, 10, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("servicenow adapter: create fallback queue: %w", err)
	}

	return &Adapter{
		client:        client,
		fallbackQueue: fq,
		username:      cfg.Username,
		password:      cfg.Password,
		logger:        cfg.Logger,
	}, nil
}

// HealthCheck проверяет доступность ServiceNow.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	resp, err := a.client.getRaw(ctx, "/api/now/table/sys_user?sysparm_limit=1", a.username, a.password)
	if err != nil {
		return fmt.Errorf("servicenow health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("servicenow health check: HTTP %d", resp.StatusCode)
	}
	return nil
}

// SyncAsset синхронизирует устройство с ServiceNow CMDB (cmdb_ci).
func (a *Adapter) SyncAsset(ctx context.Context, deviceID string, assetData map[string]interface{}) error {
	err := a.client.post(ctx, "/api/now/table/cmdb_ci", assetData, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: sync asset failed, enqueuing", "device_id", deviceID, "error", err)
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
	body := toWorkOrderSNBody(wo)
	err := a.client.post(ctx, "/api/now/table/"+TableWorkOrder, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: create work order failed, enqueuing", "id", wo.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_wo", wo)
		return nil
	}
	return nil
}

func (a *Adapter) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	path := buildTablePath(TableWorkOrder, filters)
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get work orders: %w", err)
	}
	var result []models.WorkOrder
	for _, raw := range resp.Result {
		result = append(result, toWorkOrder(raw))
	}
	return result, nil
}

func (a *Adapter) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	path := "/api/now/table/" + TableWorkOrder + "/" + id
	var resp snSingleResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get work order %s: %w", id, err)
	}
	wo := toWorkOrder(resp.Result)
	return &wo, nil
}

func (a *Adapter) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.patch(ctx, "/api/now/table/"+TableWorkOrder+"/"+id, updates, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_wo", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) AssignWorkOrder(ctx context.Context, id, userID string) error {
	body := map[string]string{"u_assigned_to": userID}
	err := a.client.patch(ctx, "/api/now/table/"+TableWorkOrder+"/"+id, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: assign work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("assign_wo", map[string]string{"id": id, "user_id": userID})
		return nil
	}
	return nil
}

func (a *Adapter) StartWorkOrder(ctx context.Context, id string) error {
	body := map[string]interface{}{
		"u_status":     "in_progress",
		"u_started_at": "now",
	}
	err := a.client.patch(ctx, "/api/now/table/"+TableWorkOrder+"/"+id, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: start work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("start_wo", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	body := map[string]interface{}{
		"u_status":       "completed",
		"u_notes":        notes,
		"u_photos":       photos,
		"u_parts_used":   parts,
		"u_completed_by": userID,
		"u_completed_at": "now",
	}
	err := a.client.patch(ctx, "/api/now/table/"+TableWorkOrder+"/"+id, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: complete work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_wo", body)
		return nil
	}
	return nil
}

func (a *Adapter) CancelWorkOrder(ctx context.Context, id, reason string) error {
	body := map[string]string{
		"u_status": "cancelled",
		"u_notes":  reason,
	}
	err := a.client.patch(ctx, "/api/now/table/"+TableWorkOrder+"/"+id, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: cancel work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("cancel_wo", map[string]string{"id": id, "reason": reason})
		return nil
	}
	return nil
}

func (a *Adapter) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	body := map[string]interface{}{
		"u_part_id":  partID,
		"u_quantity": quantity,
		"u_user_id":  userID,
	}
	err := a.client.post(ctx, "/api/now/table/"+TableWorkOrder+"/"+workOrderID+"/parts", body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: use part in work order failed, enqueuing", "wo_id", workOrderID, "error", err)
		_ = a.fallbackQueue.Enqueue("use_part", body)
		return nil
	}
	return nil
}

// ── Spare Parts ──────────────────────────────────────────────────

func (a *Adapter) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	body := toSparePartSNBody(part)
	err := a.client.post(ctx, "/api/now/table/"+TableSparePart, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: create spare part failed, enqueuing", "id", part.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_part", part)
		return nil
	}
	return nil
}

func (a *Adapter) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	path := buildTablePath(TableSparePart, filters)
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get spare parts: %w", err)
	}
	var result []models.SparePart
	for _, raw := range resp.Result {
		result = append(result, toSparePart(raw))
	}
	return result, nil
}

func (a *Adapter) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	path := "/api/now/table/" + TableSparePart + "/" + id
	var resp snSingleResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get spare part %s: %w", id, err)
	}
	sp := toSparePart(resp.Result)
	return &sp, nil
}

func (a *Adapter) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.patch(ctx, "/api/now/table/"+TableSparePart+"/"+id, updates, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_part", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteSparePart(ctx context.Context, id string) error {
	err := a.client.delete(ctx, "/api/now/table/"+TableSparePart+"/"+id, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: delete spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_part", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	path := "/api/now/table/" + TableSparePart + "?sysparm_query=u_stock<=u_min_stock"
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get low stock parts: %w", err)
	}
	var result []models.SparePart
	for _, raw := range resp.Result {
		result = append(result, toSparePart(raw))
	}
	return result, nil
}

func (a *Adapter) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	body := map[string]int{"u_stock": quantity}
	err := a.client.patch(ctx, "/api/now/table/"+TableSparePart+"/"+id, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update spare part stock failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_stock", map[string]interface{}{"id": id, "quantity": quantity})
		return nil
	}
	return nil
}

// ── Maintenance Schedules ────────────────────────────────────────

func (a *Adapter) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	body := toMaintenanceScheduleSNBody(schedule)
	err := a.client.post(ctx, "/api/now/table/"+TableMaintenanceSchedule, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: create schedule failed, enqueuing", "id", schedule.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_schedule", schedule)
		return nil
	}
	return nil
}

func (a *Adapter) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	path := buildTablePath(TableMaintenanceSchedule, filters)
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get schedules: %w", err)
	}
	var result []models.MaintenanceSchedule
	for _, raw := range resp.Result {
		result = append(result, toMaintenanceSchedule(raw))
	}
	return result, nil
}

func (a *Adapter) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	path := "/api/now/table/" + TableMaintenanceSchedule + "/" + id
	var resp snSingleResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get schedule %s: %w", id, err)
	}
	ms := toMaintenanceSchedule(resp.Result)
	return &ms, nil
}

func (a *Adapter) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.patch(ctx, "/api/now/table/"+TableMaintenanceSchedule+"/"+id, updates, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_schedule", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	err := a.client.delete(ctx, "/api/now/table/"+TableMaintenanceSchedule+"/"+id, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: delete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	path := "/api/now/table/" + TableMaintenanceSchedule + "?sysparm_query=u_next_due<=javascript:gs.now()"
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get due schedules: %w", err)
	}
	var result []models.MaintenanceSchedule
	for _, raw := range resp.Result {
		result = append(result, toMaintenanceSchedule(raw))
	}
	return result, nil
}

func (a *Adapter) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	body := map[string]string{"u_last_completed": "now"}
	err := a.client.patch(ctx, "/api/now/table/"+TableMaintenanceSchedule+"/"+id, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: complete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── SLA ──────────────────────────────────────────────────────────

func (a *Adapter) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	path := "/api/now/table/" + TableSLA + "?sysparm_query=u_priority=" + priority
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get sla config %s: %w", priority, err)
	}
	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("servicenow: sla config not found for priority %s", priority)
	}
	cfg := toSLAConfig(resp.Result[0])
	return &cfg, nil
}

func (a *Adapter) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	path := "/api/now/table/" + TableSLA
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get all sla configs: %w", err)
	}
	var result []models.SLAConfig
	for _, raw := range resp.Result {
		result = append(result, toSLAConfig(raw))
	}
	return result, nil
}

func (a *Adapter) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	body := map[string]int{
		"u_response_time_minutes":   responseTimeMinutes,
		"u_resolution_time_minutes": resolutionTimeMinutes,
	}
	err := a.client.patch(ctx, "/api/now/table/"+TableSLA+"?sysparm_query=u_priority="+priority, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update sla config failed, enqueuing", "priority", priority, "error", err)
		_ = a.fallbackQueue.Enqueue("update_sla", map[string]interface{}{"priority": priority, "config": body})
		return nil
	}
	return nil
}

// ── Technicians ──────────────────────────────────────────────────

func (a *Adapter) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	path := "/api/now/table/sys_user?sysparm_query=sys_id=" + userID
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get technician workload %s: %w", userID, err)
	}
	if len(resp.Result) == 0 {
		return nil, fmt.Errorf("servicenow: technician %s not found", userID)
	}
	wl := toTechnicianWorkload(resp.Result[0])
	return &wl, nil
}

func (a *Adapter) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	path := "/api/now/table/sys_user?sysparm_query=u_is_technician=true"
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get all technician workloads: %w", err)
	}
	var result []models.TechnicianWorkload
	for _, raw := range resp.Result {
		result = append(result, toTechnicianWorkload(raw))
	}
	return result, nil
}

func (a *Adapter) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	path := "/api/now/table/" + TableWorkOrder + "?sysparm_query=u_assigned_to=" + userID + "^u_completed_atONLast 30 days@javascript:gs.beginningOfLast30Days()@javascript:gs.endOfLast30Days()"
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get technician monthly stats %s: %w", userID, err)
	}
	stats := models.TechnicianMonthlyStats{
		CompletedThisMonth: len(resp.Result),
		TotalWorkOrders:    len(resp.Result),
	}
	return &stats, nil
}

func (a *Adapter) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	body := map[string]interface{}{
		"u_skills":         skills,
		"u_certifications": certifications,
	}
	err := a.client.patch(ctx, "/api/now/table/sys_user/"+userID, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update technician skills failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("update_skills", map[string]interface{}{"user_id": userID, "body": body})
		return nil
	}
	return nil
}

// ── Reports ──────────────────────────────────────────────────────

func (a *Adapter) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	path := "/api/now/table/" + TableWorkOrder + "?sysparm_query=u_status=completed"
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get maintenance report: %w", err)
	}
	var result []models.MaintenanceReport
	for _, raw := range resp.Result {
		result = append(result, toMaintenanceReport(raw))
	}
	return result, nil
}

func (a *Adapter) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	path := "/api/now/table/" + TableSLA
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get sla compliance report: %w", err)
	}
	var result []models.SLAComplianceReport
	for _, raw := range resp.Result {
		result = append(result, toSLAComplianceReport(raw))
	}
	return result, nil
}

// ── Technician Site Assignments ──────────────────────────────────

func (a *Adapter) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	body := toTechnicianAssignmentSNBody(assignment)
	err := a.client.post(ctx, "/api/now/table/"+TableTechnicianAssignment, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: create assignment failed, enqueuing", "id", assignment.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_assignment", assignment)
		return nil
	}
	return nil
}

func (a *Adapter) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	path := buildTablePath(TableTechnicianAssignment, filters)
	var resp snResponse
	if err := a.client.get(ctx, path, &resp, a.username, a.password); err != nil {
		return nil, fmt.Errorf("servicenow: get assignments: %w", err)
	}
	var result []models.TechnicianSiteAssignment
	for _, raw := range resp.Result {
		result = append(result, toTechnicianAssignment(raw))
	}
	return result, nil
}

func (a *Adapter) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.patch(ctx, "/api/now/table/"+TableTechnicianAssignment+"/"+id, updates, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: update assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_assignment", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	err := a.client.delete(ctx, "/api/now/table/"+TableTechnicianAssignment+"/"+id, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: delete assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_assignment", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── Mobile ───────────────────────────────────────────────────────

func (a *Adapter) SavePushToken(ctx context.Context, userID, token, platform string) error {
	body := map[string]string{
		"u_user_id":  userID,
		"u_token":    token,
		"u_platform": platform,
	}
	err := a.client.post(ctx, "/api/now/table/"+TablePushToken, body, nil, a.username, a.password)
	if err != nil {
		a.logger.Warn("servicenow: save push token failed, enqueuing", "user_id", userID, "error", err)
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
			return a.client.post(ctx, "/api/now/table/"+TableWorkOrder, toWorkOrderSNBody(&wo), nil, a.username, a.password)
		case "update_wo":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			id, _ := payload["id"].(string)
			updates, _ := payload["updates"].(map[string]interface{})
			return a.client.patch(ctx, "/api/now/table/"+TableWorkOrder+"/"+id, updates, nil, a.username, a.password)
		case "sync_asset":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			assetData, _ := payload["asset_data"].(map[string]interface{})
			return a.client.post(ctx, "/api/now/table/cmdb_ci", assetData, nil, a.username, a.password)
		case "complete_schedule":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			body := map[string]string{"u_last_completed": "now"}
			return a.client.patch(ctx, "/api/now/table/"+TableMaintenanceSchedule+"/"+payload["id"], body, nil, a.username, a.password)
		case "save_push_token":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, "/api/now/table/"+TablePushToken, payload, nil, a.username, a.password)
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

func (a *Adapter) GetSites(_ context.Context) ([]models.Site, error) {
	return nil, fmt.Errorf("get sites not implemented for ServiceNow adapter")
}

func (a *Adapter) GetSite(_ context.Context, _ string) (*models.Site, error) {
	return nil, fmt.Errorf("get site not implemented for ServiceNow adapter")
}

func (a *Adapter) CreateSite(_ context.Context, _ *models.Site) error {
	return fmt.Errorf("create site not implemented for ServiceNow adapter")
}

func (a *Adapter) UpdateSite(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("update site not implemented for ServiceNow adapter")
}

func (a *Adapter) DeleteSite(_ context.Context, _ string) error {
	return fmt.Errorf("delete site not implemented for ServiceNow adapter")
}

// ── Spare Part Categories ────────────────────────────────────────

func (a *Adapter) GetCategories(_ context.Context) ([]models.SparePartCategory, error) {
	return nil, fmt.Errorf("get categories not implemented for ServiceNow adapter")
}

func (a *Adapter) CreateCategory(_ context.Context, _ *models.SparePartCategory) error {
	return fmt.Errorf("create category not implemented for ServiceNow adapter")
}

func (a *Adapter) UpdateCategory(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("update category not implemented for ServiceNow adapter")
}

func (a *Adapter) DeleteCategory(_ context.Context, _ string) error {
	return fmt.Errorf("delete category not implemented for ServiceNow adapter")
}
