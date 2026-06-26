package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/oauth2"
)

// Adapter — реализация cmms.CMMSAdapter для Jira Cloud REST API v3.
// Work Orders = Jira Issues с типом "CCTV Work Order".
// Spare Parts = Issues с типом "Task" и лейблом "spare-part".
type Adapter struct {
	client        *Client
	fallbackQueue *cmms.FallbackQueue
	logger        *slog.Logger
}

// AdapterConfig — параметры для Jira адаптера.
type AdapterConfig struct {
	BaseURL      string
	Email        string
	APIToken     string
	ClientID     string
	ClientSecret string
	TokenURL     string
	FallbackDir  string
	Logger       *slog.Logger
	TokenStore   oauth2.TokenStore
	Metrics      *oauth2.Metrics
}

// NewAdapter creates a new JiraAdapter
func NewAdapter(cfg AdapterConfig) (*Adapter, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	client, err := NewClient(ClientConfig{
		BaseURL:      cfg.BaseURL,
		Email:        cfg.Email,
		APIToken:     cfg.APIToken,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
	}, cfg.TokenStore, cfg.Metrics, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("jira adapter: create client: %w", err)
	}

	fq, err := cmms.NewFallbackQueue(cfg.FallbackDir, 10, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("jira adapter: create fallback queue: %w", err)
	}

	return &Adapter{
		client:        client,
		fallbackQueue: fq,
		logger:        cfg.Logger,
	}, nil
}

// HealthCheck проверяет доступность Jira API.
func (a *Adapter) HealthCheck(ctx context.Context) error {
	resp, err := a.client.GetRaw(ctx, pathHealth)
	if err != nil {
		return fmt.Errorf("jira health check: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("jira health check: HTTP %d", resp.StatusCode)
	}
	return nil
}

// SyncAsset синхронизирует устройство с Jira как issue типа "Asset".
func (a *Adapter) SyncAsset(ctx context.Context, deviceID string, assetData map[string]interface{}) error {
	fields := map[string]interface{}{
		"project":     map[string]string{"key": "CCTV"},
		"summary":     fmt.Sprintf("[ASSET] %v", assetData["name"]),
		"issuetype":   map[string]string{"name": "Asset"},
		fieldDeviceID: deviceID,
	}
	body := map[string]interface{}{"fields": fields}
	err := a.client.Post(ctx, pathIssue, body, nil)
	if err != nil {
		a.logger.Warn("jira: sync asset failed, enqueuing", "device_id", deviceID, "error", err)
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
	body := toWorkOrderJiraBody(wo)
	err := a.client.Post(ctx, pathIssue, body, nil)
	if err != nil {
		a.logger.Warn("jira: create work order failed, enqueuing", "id", wo.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_wo", wo)
		return nil
	}
	return nil
}

func (a *Adapter) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
	req := jiraSearchRequest{
		JQL:        jqlSearch(filters),
		MaxResults: 100,
		Fields:     []string{"*all"},
	}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get work orders: %w", err)
	}
	var result []models.WorkOrder
	for _, issue := range resp.Issues {
		result = append(result, toWorkOrder(issue))
	}
	return result, nil
}

func (a *Adapter) GetWorkOrder(ctx context.Context, id string) (*models.WorkOrder, error) {
	var issue jiraIssue
	if err := a.client.Get(ctx, pathIssue+"/"+id, &issue); err != nil {
		return nil, fmt.Errorf("jira: get work order %s: %w", id, err)
	}
	wo := toWorkOrder(issue)
	return &wo, nil
}

func (a *Adapter) UpdateWorkOrder(ctx context.Context, id string, updates map[string]interface{}) error {
	body := map[string]interface{}{"fields": updates}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: update work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_wo", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) AssignWorkOrder(ctx context.Context, id, userID string) error {
	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"assignee": map[string]string{"id": userID},
		},
	}
	err := a.client.Put(ctx, pathIssue+"/"+id+"/assignee", body, nil)
	if err != nil {
		a.logger.Warn("jira: assign work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("assign_wo", map[string]string{"id": id, "user_id": userID})
		return nil
	}
	return nil
}

func (a *Adapter) StartWorkOrder(ctx context.Context, id string) error {
	transitionID := internalStatusToJiraTransition("in_progress")
	if transitionID == "" {
		return fmt.Errorf("jira: no transition for in_progress")
	}
	body := jiraTransition{Transition: struct {
		ID string `json:"id"`
	}{ID: transitionID}}
	err := a.client.Post(ctx, pathIssue+"/"+id+"/transitions", body, nil)
	if err != nil {
		a.logger.Warn("jira: start work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("start_wo", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) CompleteWorkOrder(ctx context.Context, id, notes string, photos []string, parts []models.PartUsage, userID string) error {
	transitionID := internalStatusToJiraTransition("completed")
	if transitionID == "" {
		return fmt.Errorf("jira: no transition for completed")
	}
	body := jiraTransition{Transition: struct {
		ID string `json:"id"`
	}{ID: transitionID}}
	err := a.client.Post(ctx, pathIssue+"/"+id+"/transitions", body, nil)
	if err != nil {
		a.logger.Warn("jira: complete work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_wo", map[string]interface{}{
			"id": id, "notes": notes, "photos": photos, "parts": parts, "completed_by": userID,
		})
		return nil
	}

	// Добавляем комментарий с деталями завершения
	commentBody := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []map[string]interface{}{
				{
					"type": "paragraph",
					"content": []map[string]interface{}{
						{"type": "text", "text": fmt.Sprintf("Completed by %s. Notes: %s", userID, notes)},
					},
				},
			},
		},
	}
	_ = a.client.Post(ctx, pathIssue+"/"+id+"/comment", commentBody, nil)
	return nil
}

func (a *Adapter) CancelWorkOrder(ctx context.Context, id, reason string) error {
	transitionID := internalStatusToJiraTransition("cancelled")
	if transitionID == "" {
		return fmt.Errorf("jira: no transition for cancelled")
	}
	body := jiraTransition{Transition: struct {
		ID string `json:"id"`
	}{ID: transitionID}}
	err := a.client.Post(ctx, pathIssue+"/"+id+"/transitions", body, nil)
	if err != nil {
		a.logger.Warn("jira: cancel work order failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("cancel_wo", map[string]string{"id": id, "reason": reason})
		return nil
	}
	return nil
}

func (a *Adapter) UsePartInWorkOrder(ctx context.Context, workOrderID, partID string, quantity int, userID string) error {
	commentBody := map[string]interface{}{
		"body": map[string]interface{}{
			"type":    "doc",
			"version": 1,
			"content": []map[string]interface{}{
				{
					"type": "paragraph",
					"content": []map[string]interface{}{
						{"type": "text", "text": fmt.Sprintf("Part used: %s × %d by %s", partID, quantity, userID)},
					},
				},
			},
		},
	}
	err := a.client.Post(ctx, pathIssue+"/"+workOrderID+"/comment", commentBody, nil)
	if err != nil {
		a.logger.Warn("jira: use part in work order failed, enqueuing", "wo_id", workOrderID, "error", err)
		_ = a.fallbackQueue.Enqueue("use_part", map[string]interface{}{
			"wo_id": workOrderID, "part_id": partID, "quantity": quantity, "user_id": userID,
		})
		return nil
	}
	return nil
}

// ── Spare Parts ──────────────────────────────────────────────────

func (a *Adapter) CreateSparePart(ctx context.Context, part *models.SparePart) error {
	body := toSparePartJiraBody(part)
	err := a.client.Post(ctx, pathIssue, body, nil)
	if err != nil {
		a.logger.Warn("jira: create spare part failed, enqueuing", "id", part.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_part", part)
		return nil
	}
	return nil
}

func (a *Adapter) GetSpareParts(ctx context.Context, filters map[string]interface{}) ([]models.SparePart, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = spare-part"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get spare parts: %w", err)
	}
	var result []models.SparePart
	for _, issue := range resp.Issues {
		result = append(result, toSparePart(issue))
	}
	return result, nil
}

func (a *Adapter) GetSparePart(ctx context.Context, id string) (*models.SparePart, error) {
	var issue jiraIssue
	if err := a.client.Get(ctx, pathIssue+"/"+id, &issue); err != nil {
		return nil, fmt.Errorf("jira: get spare part %s: %w", id, err)
	}
	sp := toSparePart(issue)
	return &sp, nil
}

func (a *Adapter) UpdateSparePart(ctx context.Context, id string, updates map[string]interface{}) error {
	body := map[string]interface{}{"fields": updates}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: update spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_part", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteSparePart(ctx context.Context, id string) error {
	err := a.client.Delete(ctx, pathIssue+"/"+id)
	if err != nil {
		a.logger.Warn("jira: delete spare part failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_part", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) GetLowStockParts(ctx context.Context) ([]models.SparePart, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = spare-part AND \"Stock\" <= \"Min Stock\""
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get low stock parts: %w", err)
	}
	var result []models.SparePart
	for _, issue := range resp.Issues {
		result = append(result, toSparePart(issue))
	}
	return result, nil
}

func (a *Adapter) UpdateSparePartStock(ctx context.Context, id string, quantity int) error {
	body := map[string]interface{}{"fields": map[string]interface{}{fieldStock: quantity}}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: update spare part stock failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_stock", map[string]interface{}{"id": id, "quantity": quantity})
		return nil
	}
	return nil
}

// ── Maintenance Schedules ────────────────────────────────────────

func (a *Adapter) CreateMaintenanceSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error {
	body := toMaintenanceScheduleJiraBody(schedule)
	err := a.client.Post(ctx, pathIssue, body, nil)
	if err != nil {
		a.logger.Warn("jira: create schedule failed, enqueuing", "id", schedule.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_schedule", schedule)
		return nil
	}
	return nil
}

func (a *Adapter) GetMaintenanceSchedules(ctx context.Context, filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = schedule"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get schedules: %w", err)
	}
	var result []models.MaintenanceSchedule
	for _, issue := range resp.Issues {
		result = append(result, toMaintenanceSchedule(issue))
	}
	return result, nil
}

func (a *Adapter) GetMaintenanceSchedule(ctx context.Context, id string) (*models.MaintenanceSchedule, error) {
	var issue jiraIssue
	if err := a.client.Get(ctx, pathIssue+"/"+id, &issue); err != nil {
		return nil, fmt.Errorf("jira: get schedule %s: %w", id, err)
	}
	ms := toMaintenanceSchedule(issue)
	return &ms, nil
}

func (a *Adapter) UpdateMaintenanceSchedule(ctx context.Context, id string, updates map[string]interface{}) error {
	body := map[string]interface{}{"fields": updates}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: update schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_schedule", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteMaintenanceSchedule(ctx context.Context, id string) error {
	err := a.client.Delete(ctx, pathIssue+"/"+id)
	if err != nil {
		a.logger.Warn("jira: delete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

func (a *Adapter) GetDueSchedules(ctx context.Context) ([]models.MaintenanceSchedule, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = schedule AND duedate <= now()"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get due schedules: %w", err)
	}
	var result []models.MaintenanceSchedule
	for _, issue := range resp.Issues {
		result = append(result, toMaintenanceSchedule(issue))
	}
	return result, nil
}

func (a *Adapter) CompleteMaintenanceSchedule(ctx context.Context, id string) error {
	body := map[string]interface{}{"fields": map[string]interface{}{fieldLastCompleted: "now"}}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: complete schedule failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("complete_schedule", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── SLA ──────────────────────────────────────────────────────────

func (a *Adapter) GetSLAConfig(ctx context.Context, priority string) (*models.SLAConfig, error) {
	jql := fmt.Sprintf("project = CCTV AND issuetype = Task AND labels = sla AND priority = \"%s\"", priority)
	req := jiraSearchRequest{JQL: jql, MaxResults: 1, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get sla config %s: %w", priority, err)
	}
	if len(resp.Issues) == 0 {
		return nil, fmt.Errorf("jira: sla config not found for priority %s", priority)
	}
	cfg := toSLAConfig(resp.Issues[0])
	return &cfg, nil
}

func (a *Adapter) GetAllSLAConfigs(ctx context.Context) ([]models.SLAConfig, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = sla"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get all sla configs: %w", err)
	}
	var result []models.SLAConfig
	for _, issue := range resp.Issues {
		result = append(result, toSLAConfig(issue))
	}
	return result, nil
}

func (a *Adapter) UpdateSLAConfig(ctx context.Context, priority string, responseTimeMinutes, resolutionTimeMinutes int) error {
	jql := fmt.Sprintf("project = CCTV AND issuetype = Task AND labels = sla AND priority = \"%s\"", priority)
	req := jiraSearchRequest{JQL: jql, MaxResults: 1, Fields: []string{"id"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return fmt.Errorf("jira: find sla config for update: %w", err)
	}
	if len(resp.Issues) == 0 {
		return fmt.Errorf("jira: sla config not found for priority %s", priority)
	}
	id := resp.Issues[0].ID
	body := map[string]interface{}{
		"fields": map[string]interface{}{
			fieldResponseTime:   responseTimeMinutes,
			fieldResolutionTime: resolutionTimeMinutes,
		},
	}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: update sla config failed, enqueuing", "priority", priority, "error", err)
		_ = a.fallbackQueue.Enqueue("update_sla", map[string]interface{}{"priority": priority, "config": body})
		return nil
	}
	return nil
}

// ── Technicians ──────────────────────────────────────────────────

func (a *Adapter) GetTechnicianWorkload(ctx context.Context, userID string) (*models.TechnicianWorkload, error) {
	var user map[string]interface{}
	if err := a.client.Get(ctx, pathUsers+"?accountId="+userID, &user); err != nil {
		return nil, fmt.Errorf("jira: get technician workload %s: %w", userID, err)
	}
	m := newMapper(user)
	loc := m.str(fieldBaseLocation)
	return &models.TechnicianWorkload{
		UserID:          m.str("accountId"),
		UserName:        m.str("displayName"),
		CurrentWorkload: m.int(fieldCurrentWorkload),
		MaxWorkload:     m.int(fieldMaxWorkload),
		Skills:          m.strSlice(fieldSkills),
		BaseLocation:    &loc,
	}, nil
}

func (a *Adapter) GetAllTechnicianWorkloads(ctx context.Context) ([]models.TechnicianWorkload, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = technician"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get all technician workloads: %w", err)
	}
	var result []models.TechnicianWorkload
	for _, issue := range resp.Issues {
		result = append(result, toTechnicianWorkload(issue))
	}
	return result, nil
}

func (a *Adapter) GetTechnicianMonthlyStats(ctx context.Context, userID string) (*models.TechnicianMonthlyStats, error) {
	jql := fmt.Sprintf("project = CCTV AND assignee = \"%s\" AND resolved >= -30d", userID)
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get technician monthly stats %s: %w", userID, err)
	}
	return &models.TechnicianMonthlyStats{
		CompletedThisMonth: len(resp.Issues),
		TotalWorkOrders:    len(resp.Issues),
	}, nil
}

func (a *Adapter) UpdateTechnicianSkills(ctx context.Context, userID string, skills []string, certifications []string) error {
	body := map[string]interface{}{
		"fields": map[string]interface{}{
			fieldSkills:         skills,
			fieldCertifications: certifications,
		},
	}
	err := a.client.Put(ctx, pathIssue+"/"+userID, body, nil)
	if err != nil {
		a.logger.Warn("jira: update technician skills failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("update_skills", map[string]interface{}{"user_id": userID, "body": body})
		return nil
	}
	return nil
}

// ── Reports ──────────────────────────────────────────────────────

func (a *Adapter) GetMaintenanceReport(ctx context.Context) ([]models.MaintenanceReport, error) {
	jql := "project = CCTV AND issuetype = \"CCTV Work Order\" AND status = Done"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get maintenance report: %w", err)
	}
	var result []models.MaintenanceReport
	for _, issue := range resp.Issues {
		result = append(result, toMaintenanceReport(issue))
	}
	return result, nil
}

func (a *Adapter) GetSLAComplianceReport(ctx context.Context) ([]models.SLAComplianceReport, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = sla"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get sla compliance report: %w", err)
	}
	var result []models.SLAComplianceReport
	for _, issue := range resp.Issues {
		result = append(result, toSLAComplianceReport(issue))
	}
	return result, nil
}

// ── Technician Site Assignments ──────────────────────────────────

func (a *Adapter) CreateTechnicianSiteAssignment(ctx context.Context, assignment *models.TechnicianSiteAssignment) error {
	body := toTechnicianAssignmentJiraBody(assignment)
	err := a.client.Post(ctx, pathIssue, body, nil)
	if err != nil {
		a.logger.Warn("jira: create assignment failed, enqueuing", "id", assignment.ID, "error", err)
		_ = a.fallbackQueue.Enqueue("create_assignment", assignment)
		return nil
	}
	return nil
}

func (a *Adapter) GetTechnicianSiteAssignments(ctx context.Context, filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	jql := "project = CCTV AND issuetype = Task AND labels = assignment"
	req := jiraSearchRequest{JQL: jql, MaxResults: 100, Fields: []string{"*all"}}
	var resp jiraSearchResponse
	if err := a.client.Post(ctx, pathSearch, req, &resp); err != nil {
		return nil, fmt.Errorf("jira: get assignments: %w", err)
	}
	var result []models.TechnicianSiteAssignment
	for _, issue := range resp.Issues {
		result = append(result, toTechnicianAssignment(issue))
	}
	return result, nil
}

func (a *Adapter) UpdateTechnicianSiteAssignment(ctx context.Context, id string, updates map[string]interface{}) error {
	body := map[string]interface{}{"fields": updates}
	err := a.client.Put(ctx, pathIssue+"/"+id, body, nil)
	if err != nil {
		a.logger.Warn("jira: update assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("update_assignment", map[string]interface{}{"id": id, "updates": updates})
		return nil
	}
	return nil
}

func (a *Adapter) DeleteTechnicianSiteAssignment(ctx context.Context, id string) error {
	err := a.client.Delete(ctx, pathIssue+"/"+id)
	if err != nil {
		a.logger.Warn("jira: delete assignment failed, enqueuing", "id", id, "error", err)
		_ = a.fallbackQueue.Enqueue("delete_assignment", map[string]string{"id": id})
		return nil
	}
	return nil
}

// ── Mobile ───────────────────────────────────────────────────────

func (a *Adapter) SavePushToken(ctx context.Context, userID, token, platform string) error {
	body := map[string]interface{}{
		"fields": map[string]interface{}{
			"project":         map[string]string{"key": "CCTV"},
			"summary":         fmt.Sprintf("[PUSH] %s", userID),
			"issuetype":       map[string]string{"name": "Task"},
			"labels":          []string{"push-token"},
			fieldPushToken:    token,
			fieldPushPlatform: platform,
		},
	}
	err := a.client.Post(ctx, pathIssue, body, nil)
	if err != nil {
		a.logger.Warn("jira: save push token failed, enqueuing", "user_id", userID, "error", err)
		_ = a.fallbackQueue.Enqueue("save_push_token", map[string]string{"user_id": userID, "token": token, "platform": platform})
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
			return a.client.Post(ctx, pathIssue, toWorkOrderJiraBody(&wo), nil)
		case "update_wo":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			id, _ := payload["id"].(string)
			updates, _ := payload["updates"].(map[string]interface{})
			return a.client.Put(ctx, pathIssue+"/"+id, map[string]interface{}{"fields": updates}, nil)
		case "sync_asset":
			var payload map[string]interface{}
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			assetData, _ := payload["asset_data"].(map[string]interface{})
			fields := map[string]interface{}{
				"project":   map[string]string{"key": "CCTV"},
				"summary":   fmt.Sprintf("[ASSET] %v", assetData["name"]),
				"issuetype": map[string]string{"name": "Asset"},
			}
			return a.client.Post(ctx, pathIssue, map[string]interface{}{"fields": fields}, nil)
		case "complete_schedule":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			body := map[string]interface{}{"fields": map[string]interface{}{fieldLastCompleted: "now"}}
			return a.client.Put(ctx, pathIssue+"/"+payload["id"], body, nil)
		case "save_push_token":
			var payload map[string]string
			if err := json.Unmarshal(entry.Payload, &payload); err != nil {
				return fmt.Errorf("unmarshal: %w", err)
			}
			body := map[string]interface{}{
				"fields": map[string]interface{}{
					"project":         map[string]string{"key": "CCTV"},
					"summary":         fmt.Sprintf("[PUSH] %s", payload["user_id"]),
					"issuetype":       map[string]string{"name": "Task"},
					"labels":          []string{"push-token"},
					fieldPushToken:    payload["token"],
					fieldPushPlatform: payload["platform"],
				},
			}
			return a.client.Post(ctx, pathIssue, body, nil)
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
	return nil, fmt.Errorf("get sites not implemented for Jira adapter")
}

func (a *Adapter) GetSite(_ context.Context, _ string) (*models.Site, error) {
	return nil, fmt.Errorf("get site not implemented for Jira adapter")
}

func (a *Adapter) CreateSite(_ context.Context, _ *models.Site) error {
	return fmt.Errorf("create site not implemented for Jira adapter")
}

func (a *Adapter) UpdateSite(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("update site not implemented for Jira adapter")
}

func (a *Adapter) DeleteSite(_ context.Context, _ string) error {
	return fmt.Errorf("delete site not implemented for Jira adapter")
}

// ── Spare Part Categories ────────────────────────────────────────

func (a *Adapter) GetCategories(_ context.Context) ([]models.SparePartCategory, error) {
	return nil, fmt.Errorf("get categories not implemented for Jira adapter")
}

func (a *Adapter) CreateCategory(_ context.Context, _ *models.SparePartCategory) error {
	return fmt.Errorf("create category not implemented for Jira adapter")
}

func (a *Adapter) UpdateCategory(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("update category not implemented for Jira adapter")
}

func (a *Adapter) DeleteCategory(_ context.Context, _ string) error {
	return fmt.Errorf("delete category not implemented for Jira adapter")
}

// ── Work Requests (not supported) ───────────────────────────────

func (a *Adapter) CreateWorkRequest(_ context.Context, _ *models.WorkRequest) error {
	return fmt.Errorf("work requests not supported for Jira adapter")
}

func (a *Adapter) GetWorkRequests(_ context.Context, _ map[string]interface{}) ([]models.WorkRequest, error) {
	return nil, fmt.Errorf("work requests not supported for Jira adapter")
}

func (a *Adapter) GetWorkRequest(_ context.Context, _ string) (*models.WorkRequest, error) {
	return nil, fmt.Errorf("work requests not supported for Jira adapter")
}

func (a *Adapter) ApproveWorkRequest(_ context.Context, _, _ string) error {
	return fmt.Errorf("work requests not supported for Jira adapter")
}

func (a *Adapter) RejectWorkRequest(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("work requests not supported for Jira adapter")
}

func (a *Adapter) ConvertWorkRequestToWO(_ context.Context, _, _ string) error {
	return fmt.Errorf("work requests not supported for Jira adapter")
}

// ── WorkOrder ↔ Alert (DM-1.3.1 — not supported for external CMMS) ─

func (a *Adapter) LinkAlertToWorkOrder(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("work order alerts not supported for Jira adapter")
}

func (a *Adapter) UnlinkAlertFromWorkOrder(_ context.Context, _, _ string) error {
	return fmt.Errorf("work order alerts not supported for Jira adapter")
}

func (a *Adapter) GetAlertsForWorkOrder(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, fmt.Errorf("work order alerts not supported for Jira adapter")
}

func (a *Adapter) GetWorkOrdersForAlert(_ context.Context, _ string) ([]models.WorkOrderAlert, error) {
	return nil, fmt.Errorf("work order alerts not supported for Jira adapter")
}

// ── Vendors (INV-7.2.1 — not supported for external CMMS) ────────

func (a *Adapter) CreateVendor(_ context.Context, _ *models.Vendor) error {
	return fmt.Errorf("vendors not supported for Jira adapter")
}

func (a *Adapter) GetVendors(_ context.Context, _ map[string]interface{}) ([]models.Vendor, error) {
	return nil, fmt.Errorf("vendors not supported for Jira adapter")
}

func (a *Adapter) GetVendor(_ context.Context, _ string) (*models.Vendor, error) {
	return nil, fmt.Errorf("vendors not supported for Jira adapter")
}

func (a *Adapter) UpdateVendor(_ context.Context, _ string, _ map[string]interface{}) error {
	return fmt.Errorf("vendors not supported for Jira adapter")
}

func (a *Adapter) DeleteVendor(_ context.Context, _ string) error {
	return fmt.Errorf("vendors not supported for Jira adapter")
}
