// Package api — P2-CHECK: Conditional Checklists (MaintainX-level).
//
// Обрабатывает CRUD для шаблонов чек-листов и управление экземплярами
// чек-листов для Work Orders (start/submit/verify).
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — checklist audit trail)
//   - ISO 27001 A.12.6 (Maintenance — structured checklists)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Template Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleListTemplates возвращает список шаблонов чек-листов.
// GET /api/v1/checklists/templates?device_type=camera&active_only=true&limit=20&offset=0
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := models.TemplateListQuery{
		DeviceType: q.Get("device_type"),
		Limit:      20,
		Offset:     0,
	}

	if active := q.Get("active_only"); active == "true" {
		query.ActiveOnly = true
	}
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			query.Limit = l
		}
	}
	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			query.Offset = o
		}
	}

	templates, err := s.listChecklistTemplates(r.Context(), query)
	if err != nil {
		s.logger.Error("Failed to list checklist templates", "error", err)
		RespondError(w, r, NewInternalError("Failed to list templates", err))
		return
	}

	if templates == nil {
		templates = []models.ChecklistTemplate{}
	}
	jsonResponse(w, http.StatusOK, templates)
}

// handleGetTemplate возвращает шаблон чек-листа по ID с items.
// GET /api/v1/checklists/templates/{id}
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("template id is required"))
		return
	}

	template, err := s.getChecklistTemplate(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get checklist template", "id", id, "error", err)
		RespondError(w, r, NewNotFoundError("Template not found"))
		return
	}

	jsonResponse(w, http.StatusOK, template)
}

// handleCreateTemplate создаёт новый шаблон чек-листа.
// POST /api/v1/checklists/templates
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Валидация
	if req.Name == "" {
		RespondError(w, r, NewValidationError("name is required"))
		return
	}
	if len(req.DeviceTypes) == 0 {
		RespondError(w, r, NewValidationError("at least one device_type is required"))
		return
	}
	if req.PassThreshold < 0 || req.PassThreshold > 100 {
		req.PassThreshold = 70
	}

	template, err := s.createChecklistTemplate(r.Context(), req)
	if err != nil {
		s.logger.Error("Failed to create checklist template", "error", err)
		RespondError(w, r, NewInternalError("Failed to create template", err))
		return
	}

	// Audit log (ISO 27001 A.12.4)
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "checklist_template.created", "checklist_template", template.ID, nil, map[string]interface{}{
		"name":         template.Name,
		"device_types": template.DeviceTypes,
	})

	jsonResponse(w, http.StatusCreated, template)
}

// handleUpdateTemplate обновляет шаблон чек-листа.
// PUT /api/v1/checklists/templates/{id}
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("template id is required"))
		return
	}

	var req models.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	template, err := s.updateChecklistTemplate(r.Context(), id, req)
	if err != nil {
		s.logger.Error("Failed to update checklist template", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to update template", err))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "checklist_template.updated", "checklist_template", template.ID, nil, map[string]interface{}{
		"name": template.Name,
	})

	jsonResponse(w, http.StatusOK, template)
}

// handleDeleteTemplate удаляет шаблон чек-листа (soft delete: is_active = false).
// DELETE /api/v1/checklists/templates/{id}
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, NewValidationError("template id is required"))
		return
	}

	if err := s.deleteChecklistTemplate(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete checklist template", "id", id, "error", err)
		RespondError(w, r, NewInternalError("Failed to delete template", err))
		return
	}

	// Audit log
	userID := userIDFromCtx(r.Context())
	s.logAudit(userID, "checklist_template.deleted", "checklist_template", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Work Order Checklist Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleStartChecklist запускает чек-лист для Work Order.
// POST /api/v1/work-orders/{id}/checklist/start
func (s *Server) handleStartChecklist(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "id")
	if woID == "" {
		RespondError(w, r, NewValidationError("work order id is required"))
		return
	}

	var req models.StartChecklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}
	if req.TemplateID == "" {
		RespondError(w, r, NewValidationError("template_id is required"))
		return
	}

	userID := userIDFromCtx(r.Context())

	checklist, err := s.startWorkOrderChecklist(r.Context(), woID, req.TemplateID, userID)
	if err != nil {
		s.logger.Error("Failed to start checklist", "wo_id", woID, "error", err)
		RespondError(w, r, NewInternalError("Failed to start checklist", err))
		return
	}

	// Audit log
	s.logAudit(userID, "checklist.started", "work_order_checklist", checklist.ID, nil, map[string]interface{}{
		"work_order_id": woID,
		"template_id":   req.TemplateID,
	})

	jsonResponse(w, http.StatusCreated, checklist)
}

// handleSubmitChecklist сабмитит заполненный чек-лист.
// POST /api/v1/work-orders/{id}/checklist/submit
func (s *Server) handleSubmitChecklist(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "id")
	if woID == "" {
		RespondError(w, r, NewValidationError("work order id is required"))
		return
	}

	var req models.SubmitChecklistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}
	if len(req.Responses) == 0 {
		RespondError(w, r, NewValidationError("at least one response is required"))
		return
	}

	userID := userIDFromCtx(r.Context())

	summary, err := s.submitWorkOrderChecklist(r.Context(), woID, req, userID)
	if err != nil {
		s.logger.Error("Failed to submit checklist", "wo_id", woID, "error", err)
		RespondError(w, r, NewInternalError("Failed to submit checklist", err))
		return
	}

	// Audit log
	s.logAudit(userID, "checklist.submitted", "work_order_checklist", summary.ID, nil, map[string]interface{}{
		"work_order_id":   woID,
		"score_percent":   summary.ScorePercent,
		"passed":          summary.Passed,
		"total_items":     summary.TotalItems,
		"completed_items": summary.CompletedItems,
	})

	jsonResponse(w, http.StatusOK, summary)
}

// handleGetWorkOrderChecklist возвращает текущий чек-лист для Work Order.
// GET /api/v1/work-orders/{id}/checklist
func (s *Server) handleGetWorkOrderChecklist(w http.ResponseWriter, r *http.Request) {
	woID := chi.URLParam(r, "id")
	if woID == "" {
		RespondError(w, r, NewValidationError("work order id is required"))
		return
	}

	checklist, err := s.getWorkOrderChecklist(r.Context(), woID)
	if err != nil {
		s.logger.Error("Failed to get work order checklist", "wo_id", woID, "error", err)
		RespondError(w, r, NewNotFoundError("Checklist not found for this work order"))
		return
	}

	jsonResponse(w, http.StatusOK, checklist)
}

// ═══════════════════════════════════════════════════════════════════════
// Repository methods
// ═══════════════════════════════════════════════════════════════════════

// listChecklistTemplates получает список шаблонов из БД.
func (s *Server) listChecklistTemplates(ctx context.Context, query models.TemplateListQuery) ([]models.ChecklistTemplate, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	sql := `SELECT id, name, description, device_types, pass_threshold, is_active, created_at, updated_at
		FROM checklist_templates WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if query.DeviceType != "" {
		sql += ` AND $` + strconv.Itoa(argIdx) + ` = ANY(device_types)`
		args = append(args, query.DeviceType)
		argIdx++
	}
	if query.ActiveOnly {
		sql += ` AND is_active = TRUE`
	}
	sql += ` ORDER BY name ASC`
	sql += ` LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, query.Limit, query.Offset)

	rows, err := s.db.Pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []models.ChecklistTemplate
	for rows.Next() {
		var t models.ChecklistTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.DeviceTypes, &t.PassThreshold, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}

	return templates, nil
}

// getChecklistTemplate получает шаблон с items из БД.
func (s *Server) getChecklistTemplate(ctx context.Context, id string) (*models.ChecklistTemplate, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	var t models.ChecklistTemplate
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id, name, description, device_types, pass_threshold, is_active, created_at, updated_at
		FROM checklist_templates WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Description, &t.DeviceTypes, &t.PassThreshold, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Load items
	items, err := s.loadChecklistItems(ctx, t.ID)
	if err != nil {
		return nil, err
	}
	t.Items = items

	return &t, nil
}

// loadChecklistItems загружает иерархию элементов чек-листа.
func (s *Server) loadChecklistItems(ctx context.Context, templateID string) ([]models.ChecklistItem, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, template_id, parent_id, label, description, item_type, mandatory,
			score, sort_order, options, validation_min, validation_max, created_at, updated_at
		FROM checklist_items
		WHERE template_id = $1
		ORDER BY sort_order ASC, label ASC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allItems []models.ChecklistItem
	for rows.Next() {
		var item models.ChecklistItem
		var parentID *string
		if err := rows.Scan(
			&item.ID, &item.TemplateID, &parentID, &item.Label, &item.Description,
			&item.ItemType, &item.Mandatory, &item.Score, &item.SortOrder,
			&item.Options, &item.ValidationMin, &item.ValidationMax,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.ParentID = parentID

		// Load condition for this item
		cond, err := s.loadItemCondition(ctx, item.ID)
		if err != nil {
			return nil, err
		}
		item.DependsOn = cond

		allItems = append(allItems, item)
	}

	return buildItemTree(allItems), nil
}

// loadItemCondition загружает условие для элемента.
func (s *Server) loadItemCondition(ctx context.Context, itemID string) (*models.Condition, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, nil
	}

	var fieldID, operator, value string
	err := s.db.Pool.QueryRow(ctx, `
		SELECT field_id, operator, value FROM checklist_conditions WHERE item_id = $1 LIMIT 1
	`, itemID).Scan(&fieldID, &operator, &value)
	if err != nil {
		return nil, nil // no condition
	}

	var val interface{}
	val = value
	if operator == "in" {
		var arr []interface{}
		if json.Unmarshal([]byte(value), &arr) == nil {
			val = arr
		}
	}

	return &models.Condition{
		FieldID:  fieldID,
		Operator: operator,
		Value:    val,
	}, nil
}

// buildItemTree строит иерархию элементов (parent → children).
func buildItemTree(items []models.ChecklistItem) []models.ChecklistItem {
	itemMap := make(map[string]*models.ChecklistItem)
	var roots []models.ChecklistItem

	for i := range items {
		item := items[i]
		item.Children = []models.ChecklistItem{}
		itemMap[item.ID] = &items[i]
	}

	for i := range items {
		item := &items[i]
		if item.ParentID != nil {
			if parent, ok := itemMap[*item.ParentID]; ok {
				parent.Children = append(parent.Children, *item)
			}
		} else {
			roots = append(roots, *item)
		}
	}

	return roots
}

// createChecklistTemplate создаёт шаблон с элементами.
func (s *Server) createChecklistTemplate(ctx context.Context, req models.CreateTemplateRequest) (*models.ChecklistTemplate, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	templateID := generateID()

	_, err = tx.Exec(ctx, `
		INSERT INTO checklist_templates (id, name, description, device_types, pass_threshold, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, TRUE, $6, $6)
	`, templateID, req.Name, req.Description, req.DeviceTypes, req.PassThreshold, now)
	if err != nil {
		return nil, err
	}

	// Save items if provided
	for i, item := range req.Items {
		if err := s.saveChecklistItemTx(ctx, tx, templateID, nil, item, i, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.getChecklistTemplate(ctx, templateID)
}

// saveChecklistItemTx сохраняет элемент чек-листа в транзакции.
func (s *Server) saveChecklistItemTx(ctx context.Context, tx pgx.Tx, templateID string, parentID *string, item models.ChecklistItem, sortOrder int, now time.Time) error {
	itemID := generateID()

	opts := item.Options
	if opts == nil {
		opts = []byte("null")
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO checklist_items (id, template_id, parent_id, label, description, item_type,
			mandatory, score, sort_order, options, validation_min, validation_max, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)
	`, itemID, templateID, parentID, item.Label, item.Description, item.ItemType,
		item.Mandatory, item.Score, sortOrder, opts, item.ValidationMin, item.ValidationMax, now)
	if err != nil {
		return err
	}

	// Save condition
	if item.DependsOn != nil {
		valStr := fmt.Sprintf("%v", item.DependsOn.Value)
		if item.DependsOn.Operator == "in" {
			if b, err := json.Marshal(item.DependsOn.Value); err == nil {
				valStr = string(b)
			}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO checklist_conditions (id, item_id, field_id, operator, value, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, generateID(), itemID, item.DependsOn.FieldID, item.DependsOn.Operator, valStr, now)
		if err != nil {
			return err
		}
	}

	// Save children
	for i, child := range item.Children {
		parent := itemID
		if err := s.saveChecklistItemTx(ctx, tx, templateID, &parent, child, i, now); err != nil {
			return err
		}
	}

	return nil
}

// updateChecklistTemplate обновляет шаблон (пересоздаёт items).
func (s *Server) updateChecklistTemplate(ctx context.Context, id string, req models.CreateTemplateRequest) (*models.ChecklistTemplate, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()

	// Update template
	result, err := tx.Exec(ctx, `
		UPDATE checklist_templates SET
			name = $1, description = $2, device_types = $3, pass_threshold = $4,
			is_active = TRUE, updated_at = $5
		WHERE id = $6
	`, req.Name, req.Description, req.DeviceTypes, req.PassThreshold, now, id)
	if err != nil {
		return nil, err
	}
	if result.RowsAffected() == 0 {
		return nil, fmt.Errorf("template not found")
	}

	// Delete old items (cascades to conditions via FK)
	_, err = tx.Exec(ctx, `DELETE FROM checklist_items WHERE template_id = $1`, id)
	if err != nil {
		return nil, err
	}

	// Re-create items
	for i, item := range req.Items {
		if err := s.saveChecklistItemTx(ctx, tx, id, nil, item, i, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.getChecklistTemplate(ctx, id)
}

// deleteChecklistTemplate мягко удаляет шаблон.
func (s *Server) deleteChecklistTemplate(ctx context.Context, id string) error {
	if s.db == nil || s.db.Pool == nil {
		return fmt.Errorf("database not available")
	}

	_, err := s.db.Pool.Exec(ctx, `
		UPDATE checklist_templates SET is_active = FALSE, updated_at = $1 WHERE id = $2
	`, time.Now().UTC(), id)
	return err
}

// startWorkOrderChecklist создаёт экземпляр чек-листа для Work Order.
func (s *Server) startWorkOrderChecklist(ctx context.Context, woID, templateID, userID string) (*models.WorkOrderChecklist, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	now := time.Now().UTC()
	clID := generateID()

	// Load template to calculate max score
	template, err := s.getChecklistTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	maxScore := calculateMaxScore(template.Items)

	_, err = s.db.Pool.Exec(ctx, `
		INSERT INTO work_order_checklists (id, work_order_id, template_id, status,
			total_score, max_score, score_percent, passed, started_by, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'in_progress', 0, $4, 0, FALSE, $5, $6, $6, $6)
	`, clID, woID, templateID, maxScore, userID, now)
	if err != nil {
		return nil, err
	}

	return &models.WorkOrderChecklist{
		ID:          clID,
		WorkOrderID: woID,
		TemplateID:  templateID,
		Status:      string(models.WOCStatusInProgress),
		MaxScore:    maxScore,
		StartedBy:   userID,
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// calculateMaxScore подсчитывает максимально возможный score.
func calculateMaxScore(items []models.ChecklistItem) int {
	total := 0
	for _, item := range items {
		total += item.Score
		total += calculateMaxScore(item.Children)
	}
	return total
}

// submitWorkOrderChecklist сабмитит чек-лист и рассчитывает score.
func (s *Server) submitWorkOrderChecklist(ctx context.Context, woID string, req models.SubmitChecklistRequest, userID string) (*models.ChecklistSummary, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Find the active checklist for this WO
	var clID, templateID string
	var maxScore int
	err = tx.QueryRow(ctx, `
		SELECT id, template_id, max_score FROM work_order_checklists
		WHERE work_order_id = $1 AND status = 'in_progress'
		ORDER BY started_at DESC LIMIT 1
	`, woID).Scan(&clID, &templateID, &maxScore)
	if err != nil {
		return nil, fmt.Errorf("no active checklist found for work order %s", woID)
	}

	// Load template items for validation
	template, err := s.getChecklistTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Build item lookup map
	itemMap := buildItemMap(template.Items)

	// Save responses and calculate scores
	totalScore := 0
	completedItems := 0
	skippedItems := 0
	now := time.Now().UTC()

	for _, resp := range req.Responses {
		item, exists := itemMap[resp.ItemID]
		if !exists {
			continue
		}

		// Check if item should be skipped based on conditions
		isSkipped := resp.Skipped
		if item.DependsOn != nil && !resp.Skipped {
			// Look up the trigger field's value in this submission
			triggerValue := findResponseValue(req.Responses, item.DependsOn.FieldID)
			if !item.DependsOn.Evaluate(triggerValue) {
				isSkipped = true
			}
		}

		if isSkipped {
			skippedItems++
		} else {
			completedItems++
			if item.Score > 0 && isPositiveResponse(resp.Value) {
				totalScore += item.Score
			}
		}

		// Save response
		_, err = tx.Exec(ctx, `
			INSERT INTO work_order_checklist_responses (id, checklist_id, item_id, value, photo_url, skipped, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
			ON CONFLICT (checklist_id, item_id) DO UPDATE SET
				value = EXCLUDED.value, photo_url = EXCLUDED.photo_url,
				skipped = EXCLUDED.skipped, updated_at = EXCLUDED.updated_at
		`, generateID(), clID, resp.ItemID, resp.Value, resp.PhotoURL, isSkipped, now)
		if err != nil {
			return nil, err
		}

		// Save score record
		if item.Score > 0 {
			itemScore := 0
			if !isSkipped && isPositiveResponse(resp.Value) {
				itemScore = item.Score
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO checklist_scores (id, checklist_id, item_id, score, max_score, scored_by, scored_at, notes)
				VALUES ($1, $2, $3, $4, $5, $6, $7, '')
			`, generateID(), clID, resp.ItemID, itemScore, item.Score, userID, now)
			if err != nil {
				return nil, err
			}
		}
	}

	// Calculate score percentage
	scorePercent := 0.0
	if maxScore > 0 {
		scorePercent = float64(totalScore) / float64(maxScore) * 100
	}
	passed := scorePercent >= float64(template.PassThreshold)

	// Update checklist
	_, err = tx.Exec(ctx, `
		UPDATE work_order_checklists SET
			status = 'submitted', total_score = $1, score_percent = $2,
			passed = $3, submitted_by = $4, submitted_at = $5, updated_at = $5
		WHERE id = $6
	`, totalScore, scorePercent, passed, userID, now, clID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &models.ChecklistSummary{
		ID:             clID,
		WorkOrderID:    woID,
		TemplateName:   template.Name,
		Status:         string(models.WOCStatusSubmitted),
		ScorePercent:   scorePercent,
		Passed:         passed,
		TotalItems:     len(itemMap),
		CompletedItems: completedItems,
		SkippedItems:   skippedItems,
		StartedBy:      userID,
		StartedAt:      now.Format(time.RFC3339),
		SubmittedBy:    &userID,
		SubmittedAt:    strPtr(now.Format(time.RFC3339)),
	}, nil
}

// getWorkOrderChecklist возвращает активный чек-лист для Work Order.
func (s *Server) getWorkOrderChecklist(ctx context.Context, woID string) (*models.WorkOrderChecklist, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, fmt.Errorf("database not available")
	}

	var cl models.WorkOrderChecklist
	err := s.db.Pool.QueryRow(ctx, `
		SELECT wc.id, wc.work_order_id, wc.template_id, wc.status,
			wc.total_score, wc.max_score, wc.score_percent, wc.passed,
			wc.started_by, wc.started_at, wc.submitted_by, wc.submitted_at,
			wc.verified_by, wc.verified_at, wc.notes, wc.created_at, wc.updated_at,
			ct.name as template_name
		FROM work_order_checklists wc
		JOIN checklist_templates ct ON ct.id = wc.template_id
		WHERE wc.work_order_id = $1
		ORDER BY wc.started_at DESC LIMIT 1
	`, woID).Scan(
		&cl.ID, &cl.WorkOrderID, &cl.TemplateID, &cl.Status,
		&cl.TotalScore, &cl.MaxScore, &cl.ScorePercent, &cl.Passed,
		&cl.StartedBy, &cl.StartedAt, &cl.SubmittedBy, &cl.SubmittedAt,
		&cl.VerifiedBy, &cl.VerifiedAt, &cl.Notes, &cl.CreatedAt, &cl.UpdatedAt,
		&cl.TemplateName,
	)
	if err != nil {
		return nil, err
	}

	// Load responses
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id, checklist_id, item_id, value, photo_url, skipped, created_at, updated_at
		FROM work_order_checklist_responses
		WHERE checklist_id = $1
	`, cl.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var resp models.ChecklistResponse
		if err := rows.Scan(&resp.ID, &resp.ChecklistID, &resp.ItemID,
			&resp.Value, &resp.PhotoURL, &resp.Skipped, &resp.CreatedAt, &resp.UpdatedAt); err != nil {
			return nil, err
		}
		cl.Responses = append(cl.Responses, resp)
	}

	return &cl, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// userIDFromCtx извлекает user_id из контекста запроса.
func userIDFromCtx(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "system"
}

// buildItemMap строит map[id]item для быстрого lookup.
func buildItemMap(items []models.ChecklistItem) map[string]models.ChecklistItem {
	m := make(map[string]models.ChecklistItem)
	for _, item := range items {
		m[item.ID] = item
		for _, child := range item.Children {
			m[child.ID] = child
		}
	}
	return m
}

// findResponseValue ищет значение ответа по item_id.
func findResponseValue(responses []models.SubmitItemResponse, itemID string) interface{} {
	for _, r := range responses {
		if r.ItemID == itemID {
			return r.Value
		}
	}
	return nil
}

// isPositiveResponse определяет, является ли ответ положительным (score).
func isPositiveResponse(value string) bool {
	switch value {
	case "true", "yes", "pass", "1":
		return true
	default:
		return false
	}
}

// generateID генерирует уникальный ID.
func generateID() string {
	return fmt.Sprintf("chk_%d", time.Now().UnixNano())
}

// strPtr возвращает указатель на строку.
func strPtr(s string) *string {
	return &s
}
