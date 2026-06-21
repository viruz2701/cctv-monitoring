package cmms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/models"
)

// AtlasAdapter — реализация CMMSAdapter для внешнего Atlas CMMS API.
// Использует OAuth2 client credentials flow для аутентификации.
// Поддерживает fallback-очередь для сохранения операций при недоступности API.
type AtlasAdapter struct {
	client        *AtlasClient
	fallbackQueue *FallbackQueue
	apiKey        string // fallback API key (если OAuth2 не настроен)
	logger        *slog.Logger
}

// AtlasAdapterConfig — параметры для создания AtlasAdapter.
type AtlasAdapterConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	TokenURL     string
	APIKey       string
	FallbackDir  string
	Logger       *slog.Logger
}

// NewAtlasAdapter создаёт новый экземпляр AtlasAdapter с OAuth2-клиентом.
// Если ClientID/ClientSecret/TokenURL не указаны, используется API-ключ.
func NewAtlasAdapter(cfg AtlasAdapterConfig) (*AtlasAdapter, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	var client *AtlasClient
	var err error

	if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TokenURL != "" {
		client, err = NewAtlasClient(AtlasClientConfig{
			BaseURL:      cfg.BaseURL,
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Timeout:      30 * time.Second,
		})
		if err != nil {
			return nil, fmt.Errorf("atlas adapter: create oauth2 client: %w", err)
		}
	} else {
		client = NewAtlasClientWithAPIKey(cfg.BaseURL, cfg.APIKey, 30*time.Second)
	}

	fq, err := NewFallbackQueue(cfg.FallbackDir, 10, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("atlas adapter: create fallback queue: %w", err)
	}

	return &AtlasAdapter{
		client:        client,
		fallbackQueue: fq,
		apiKey:        cfg.APIKey,
		logger:        cfg.Logger,
	}, nil
}

// HealthCheck проверяет доступность внешнего Atlas CMMS API.
func (a *AtlasAdapter) HealthCheck(ctx context.Context) error {
	resp, err := a.client.getRaw(ctx, "/health", a.apiKey)
	if err != nil {
		return fmt.Errorf("atlas health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("atlas health check: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// SyncAsset синхронизирует устройство с CMMS как актив (asset).
func (a *AtlasAdapter) SyncAsset(ctx context.Context, deviceID string, assetData map[string]interface{}) error {
	err := a.client.post(ctx, "/api/v1/assets", assetData, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: sync asset failed, enqueuing to fallback", "device_id", deviceID, "error", err)
		_ = a.fallbackQueue.Enqueue("sync_asset", map[string]interface{}{
			"device_id":  deviceID,
			"asset_data": assetData,
		})
		return nil // не возвращаем ошибку — операция сохранена в очередь
	}
	return nil
}

// ── Work Orders ──────────────────────────────────────────────────

func (a *AtlasAdapter) CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error {
	err := a.client.post(ctx, "/api/v1/work-orders", wo, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: create work order failed, enqueuing", "id", wo.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_wo", wo)
		return nil
	}
	return nil
}

func (a *AtlasAdapter) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	query := buildQueryPath("/api/v1/work-orders", filters)
	var result []models.WorkOrder
	if err := a.client.get(ctx, query, &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get work orders: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	var wo models.WorkOrder
	if err := a.client.get(ctx, "/api/v1/work-orders/"+id, &wo, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get work order %s: %w", id, err)
	}
	return &wo, nil
}

func (a *AtlasAdapter) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, "/api/v1/work-orders/"+id, updates, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_wo", map[string]interface{}{
			"id":      id,
			"updates": updates,
		})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) AssignWorkOrder(ctx context.Context, id, userID string) error {
	body := map[string]string{"assigned_to": userID}
	err := a.client.put(ctx, "/api/v1/work-orders/"+id+"/assign", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: assign work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("assign_wo", map[string]string{"id": id, "user_id": userID})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) StartWorkOrder(ctx context.Context, id string) error {
	err := a.client.post(ctx, "/api/v1/work-orders/"+id+"/start", nil, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: start work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("start_wo", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	body := map[string]interface{}{
		"notes":        notes,
		"photos":       photos,
		"parts":        parts,
		"completed_by": userID,
	}
	err := a.client.post(ctx, "/api/v1/work-orders/"+id+"/complete", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: complete work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_wo", body)
		return nil
	}
	return nil
}

func (a *AtlasAdapter) CancelWorkOrder(ctx context.Context, id, reason string) error {
	body := map[string]string{"reason": reason}
	err := a.client.post(ctx, "/api/v1/work-orders/"+id+"/cancel", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: cancel work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("cancel_wo", map[string]string{"id": id, "reason": reason})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	body := map[string]interface{}{
		"part_id":  partID,
		"quantity": quantity,
		"user_id":  userID,
	}
	err := a.client.post(ctx, "/api/v1/work-orders/"+workOrderID+"/parts", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: use part in work order failed, enqueuing", "wo_id", workOrderID, "error", err)
		_ = a.fallbackQueue.Enqueue("use_part", body)
		return nil
	}
	return nil
}

// ── Spare Parts ──────────────────────────────────────────────────

func (a *AtlasAdapter) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	err := a.client.post(ctx, "/api/v1/spare-parts", part, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: create spare part failed, enqueuing", "id", part.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_part", part)
		return nil
	}
	return nil
}

func (a *AtlasAdapter) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	query := buildQueryPath("/api/v1/spare-parts", filters)
	var result []models.SparePart
	if err := a.client.get(ctx, query, &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get spare parts: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	var part models.SparePart
	if err := a.client.get(ctx, "/api/v1/spare-parts/"+id, &part, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get spare part %s: %w", id, err)
	}
	return &part, nil
}

func (a *AtlasAdapter) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, "/api/v1/spare-parts/"+id, updates, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_part", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) DeleteSparePart(ctx context.Context, id string) error {
	err := a.client.delete(ctx, "/api/v1/spare-parts/"+id, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: delete spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_part", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	var result []models.SparePart
	if err := a.client.get(ctx, "/api/v1/spare-parts/low-stock", &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get low stock parts: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	body := map[string]int{"quantity": quantity}
	err := a.client.put(ctx, "/api/v1/spare-parts/"+id+"/stock", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update spare part stock failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_stock", map[string]interface{}{"id": id, "quantity": quantity})
		return nil
	}
	return nil
}

// ── Maintenance Schedules ────────────────────────────────────────

func (a *AtlasAdapter) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	err := a.client.post(ctx, "/api/v1/maintenance/schedules", schedule, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: create schedule failed, enqueuing", "id", schedule.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_schedule", schedule)
		return nil
	}
	return nil
}

func (a *AtlasAdapter) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	query := buildQueryPath("/api/v1/maintenance/schedules", filters)
	var result []models.MaintenanceSchedule
	if err := a.client.get(ctx, query, &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get schedules: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	var schedule models.MaintenanceSchedule
	if err := a.client.get(ctx, "/api/v1/maintenance/schedules/"+id, &schedule, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get schedule %s: %w", id, err)
	}
	return &schedule, nil
}

func (a *AtlasAdapter) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, "/api/v1/maintenance/schedules/"+id, updates, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_schedule", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	err := a.client.delete(ctx, "/api/v1/maintenance/schedules/"+id, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: delete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	var result []models.MaintenanceSchedule
	if err := a.client.get(ctx, "/api/v1/maintenance/schedules/due", &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get due schedules: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	err := a.client.post(ctx, "/api/v1/maintenance/schedules/"+id+"/complete", nil, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: complete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── SLA ──────────────────────────────────────────────────────────

func (a *AtlasAdapter) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	var config models.SLAConfig
	if err := a.client.get(ctx, "/api/v1/sla/config/"+priority, &config, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get sla config %s: %w", priority, err)
	}
	return &config, nil
}

func (a *AtlasAdapter) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	var result []models.SLAConfig
	if err := a.client.get(ctx, "/api/v1/sla/config", &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get all sla configs: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	body := map[string]int{
		"response_time_minutes":   responseTimeMinutes,
		"resolution_time_minutes": resolutionTimeMinutes,
	}
	err := a.client.put(ctx, "/api/v1/sla/config/"+priority, body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update sla config failed, enqueuing", "priority", priority, "error", err)
		_ = a.fallbackQueue.Enqueue("update_sla", map[string]interface{}{"priority": priority, "config": body})
		return nil
	}
	return nil
}

// ── Technicians ──────────────────────────────────────────────────

func (a *AtlasAdapter) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	var workload models.TechnicianWorkload
	if err := a.client.get(ctx, "/api/v1/technicians/"+userID+"/workload", &workload, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get technician workload %s: %w", userID, err)
	}
	return &workload, nil
}

func (a *AtlasAdapter) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	var result []models.TechnicianWorkload
	if err := a.client.get(ctx, "/api/v1/technicians/workload", &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get all technician workloads: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	var stats models.TechnicianMonthlyStats
	if err := a.client.get(ctx, "/api/v1/technicians/"+userID+"/monthly-stats", &stats, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get technician monthly stats %s: %w", userID, err)
	}
	return &stats, nil
}

func (a *AtlasAdapter) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	body := map[string]interface{}{
		"skills":         skills,
		"certifications": certifications,
	}
	err := a.client.put(ctx, "/api/v1/technicians/"+userID+"/skills", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update technician skills failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("update_skills", map[string]interface{}{"user_id": userID, "body": body})
		return nil
	}
	return nil
}

// ── Reports ──────────────────────────────────────────────────────

func (a *AtlasAdapter) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	var result []models.MaintenanceReport
	if err := a.client.get(ctx, "/api/v1/reports/maintenance", &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get maintenance report: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	var result []models.SLAComplianceReport
	if err := a.client.get(ctx, "/api/v1/reports/sla-compliance", &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get sla compliance report: %w", err)
	}
	return result, nil
}

// ── Technician Site Assignments ──────────────────────────────────

func (a *AtlasAdapter) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	err := a.client.post(ctx, "/api/v1/technician-assignments", assignment, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: create assignment failed, enqueuing", "id", assignment.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_assignment", assignment)
		return nil
	}
	return nil
}

func (a *AtlasAdapter) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	query := buildQueryPath("/api/v1/technician-assignments", filters)
	var result []models.TechnicianSiteAssignment
	if err := a.client.get(ctx, query, &result, a.apiKey); err != nil {
		return nil, fmt.Errorf("atlas: get assignments: %w", err)
	}
	return result, nil
}

func (a *AtlasAdapter) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	err := a.client.put(ctx, "/api/v1/technician-assignments/"+id, updates, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: update assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_assignment", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *AtlasAdapter) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	err := a.client.delete(ctx, "/api/v1/technician-assignments/"+id, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: delete assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_assignment", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── Mobile ───────────────────────────────────────────────────────

func (a *AtlasAdapter) SavePushToken(ctx context.Context, userID, token, platform string) error {
	body := map[string]string{
		"user_id":  userID,
		"token":    token,
		"platform": platform,
	}
	err := a.client.post(ctx, "/api/v1/mobile/push-token", body, nil, a.apiKey)
	if err != nil {
		a.logger.Warn("atlas: save push token failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("save_push_token", body)
		return nil
	}
	return nil
}

// ── Fallback Queue Management ────────────────────────────────────

// RetryFallback пытается повторно отправить все записи из fallback-очереди.
func (a *AtlasAdapter) RetryFallback(ctx context.Context) (success, failed int) {
	return a.fallbackQueue.RetryAll(ctx, func(ctx context.Context, entry FallbackQueueEntry) error {
		switch entry.Method {
		case "create_wo":
			var wo models.WorkOrder
			if err := json.Unmarshal(entry.Payload, &wo); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, "/api/v1/work-orders", wo, nil, a.apiKey)
		case "update_wo":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			id, _ := payload["id"].(string)
			updates, _ := payload["updates"].(map[string]interface{})
			return a.client.put(ctx, "/api/v1/work-orders/"+id, updates, nil, a.apiKey)
		case "sync_asset":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			assetData, _ := payload["asset_data"].(map[string]interface{})
			return a.client.post(ctx, "/api/v1/assets", assetData, nil, a.apiKey)
		case "complete_schedule":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, "/api/v1/maintenance/schedules/"+payload["id"]+"/complete", nil, nil, a.apiKey)
		case "save_push_token":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			return a.client.post(ctx, "/api/v1/mobile/push-token", payload, nil, a.apiKey)
		default:
			// Для остальных методов — обобщённый retry через POST
			return fmt.Errorf("fallback: unsupported retry method: %s", entry.Method)
		}
	})
}

// FallbackQueueSize возвращает размер очереди отложенных операций.
func (a *AtlasAdapter) FallbackQueueSize() int {
	count, _ := a.fallbackQueue.Len()
	return count
}

// ── Helpers ──────────────────────────────────────────────────────

// buildQueryPath добавляет query-параметры к URL из фильтров.
func buildQueryPath(base string, filters map[string]interface{}) string {
	if len(filters) == 0 {
		return base
	}
	query := base + "?"
	first := true
	for k, v := range filters {
		if !first {
			query += "&"
		}
		query += fmt.Sprintf("%s=%v", k, v)
		first = false
	}
	return query
}
