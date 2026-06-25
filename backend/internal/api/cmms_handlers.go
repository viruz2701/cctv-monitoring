package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Maintenance Schedules Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listMaintenanceSchedules(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})

	if deviceID := r.URL.Query().Get("device_id"); deviceID != "" {
		filters["device_id"] = deviceID
	}
	if scheduleType := r.URL.Query().Get("schedule_type"); scheduleType != "" {
		filters["schedule_type"] = scheduleType
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		filters["priority"] = priority
	}
	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filters["assigned_to"] = assignedTo
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters["offset"] = o
		}
	}

	schedules, err := s.cmmsRouter.GetMaintenanceSchedules(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get maintenance schedules", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if schedules == nil {
		schedules = []models.MaintenanceSchedule{}
	}
	jsonResponse(w, http.StatusOK, schedules)
}

func (s *Server) createMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	var raw struct {
		DeviceID         string          `json:"device_id"`
		ScheduleType     string          `json:"schedule_type"`
		IntervalDays     int             `json:"interval_days"`
		CustomCron       string          `json:"custom_cron"`
		NextDue          string          `json:"next_due"`
		AssignedTo       *string         `json:"assigned_to"`
		Checklist        json.RawMessage `json:"checklist"`
		EstimatedMinutes int             `json:"estimated_minutes"`
		Priority         string          `json:"priority"`
		Notes            string          `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if raw.DeviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}
	if raw.ScheduleType == "" {
		respondError(w, r, NewBadRequestError("schedule_type is required"))
		return
	}
	if raw.NextDue == "" {
		respondError(w, r, NewBadRequestError("next_due is required"))
		return
	}

	nextDue, err := parseFutureDate(raw.NextDue, "next_due")
	if err != nil {
		// Если дата в прошлом — не ошибка, но warning (ISO 27001 A.14.2.5)
		if parsed, pErr := parseValidatedDate(raw.NextDue, "next_due"); pErr == nil && parsed.Before(time.Now().UTC()) {
			s.logger.Warn("maintenance schedule next_due is in the past",
				"next_due", raw.NextDue,
				"device_id", raw.DeviceID,
				"schedule_type", raw.ScheduleType,
			)
			// Разрешено, но с предупреждением
			nextDue = parsed
		} else {
			respondError(w, r, NewBadRequestError(err.Error()))
			return
		}
	}

	schedule := models.MaintenanceSchedule{
		DeviceID:         raw.DeviceID,
		ScheduleType:     raw.ScheduleType,
		IntervalDays:     raw.IntervalDays,
		CustomCron:       raw.CustomCron,
		NextDue:          nextDue,
		AssignedTo:       raw.AssignedTo,
		Checklist:        raw.Checklist,
		EstimatedMinutes: raw.EstimatedMinutes,
		Priority:         raw.Priority,
		Notes:            raw.Notes,
	}
	if schedule.Checklist == nil {
		schedule.Checklist = json.RawMessage("[]")
	}
	if schedule.Priority == "" {
		schedule.Priority = "medium"
	}
	if schedule.EstimatedMinutes == 0 {
		schedule.EstimatedMinutes = 30
	}

	if err := s.cmmsRouter.CreateMaintenanceSchedule(r.Context(), &schedule); err != nil {
		s.logger.Error("Failed to create maintenance schedule", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// Audit log
	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_maintenance_schedule", "maintenance_schedule", schedule.ID, nil, schedule)

	jsonResponse(w, http.StatusCreated, schedule)
}

func (s *Server) getMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	schedule, err := s.cmmsRouter.GetMaintenanceSchedule(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Schedule not found"))
		return
	}
	jsonResponse(w, http.StatusOK, schedule)
}

func (s *Server) updateMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Валидация полей
	allowedFields := map[string]bool{
		"schedule_type": true, "interval_days": true, "custom_cron": true,
		"next_due": true, "assigned_to": true, "checklist": true,
		"estimated_minutes": true, "priority": true, "notes": true,
	}
	for key := range updates {
		if !allowedFields[key] {
			delete(updates, key)
		}
	}
	if err := normalizeDateUpdate(updates, "next_due", true); err != nil {
		respondError(w, r, NewBadRequestError(err.Error()))
		return
	}

	if err := s.cmmsRouter.UpdateMaintenanceSchedule(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update maintenance schedule", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_maintenance_schedule", "maintenance_schedule", id, nil, updates)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.DeleteMaintenanceSchedule(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete maintenance schedule", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_maintenance_schedule", "maintenance_schedule", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) getDueSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.cmmsRouter.GetDueSchedules(r.Context())
	if err != nil {
		s.logger.Error("Failed to get due schedules", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if schedules == nil {
		schedules = []models.MaintenanceSchedule{}
	}
	jsonResponse(w, http.StatusOK, schedules)
}

func (s *Server) completeMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.CompleteMaintenanceSchedule(r.Context(), id); err != nil {
		s.logger.Error("Failed to complete maintenance schedule", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// Создаём work order из завершённого графика
	schedule, err := s.cmmsRouter.GetMaintenanceSchedule(r.Context(), id)
	if err == nil {
		wo := &models.WorkOrder{
			ScheduleID: &schedule.ID,
			DeviceID:   schedule.DeviceID,
			Type:       "preventive",
			Status:     "open",
			Priority:   schedule.Priority,
			AssignedTo: schedule.AssignedTo,
			Checklist:  schedule.Checklist,
			Notes:      "Auto-created from maintenance schedule",
		}
		// Устанавливаем SLA deadline
		if sla, err := s.cmmsRouter.GetSLAConfig(r.Context(), schedule.Priority); err == nil {
			deadline := time.Now().Add(time.Duration(sla.ResolutionTimeMinutes) * time.Minute)
			wo.SLADeadline = &deadline
		}
		if err := s.cmmsRouter.CreateWorkOrder(r.Context(), wo); err != nil {
			s.logger.Error("Failed to create work order from schedule", "error", err)
		}
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "complete_maintenance_schedule", "maintenance_schedule", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "completed"})
}

// ═══════════════════════════════════════════════════════════════════════
// Work Orders Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listWorkOrders(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})

	if deviceID := r.URL.Query().Get("device_id"); deviceID != "" {
		filters["device_id"] = deviceID
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}
	if woType := r.URL.Query().Get("type"); woType != "" {
		filters["type"] = woType
	}
	if priority := r.URL.Query().Get("priority"); priority != "" {
		filters["priority"] = priority
	}
	if assignedTo := r.URL.Query().Get("assigned_to"); assignedTo != "" {
		filters["assigned_to"] = assignedTo
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters["offset"] = o
		}
	}

	workOrders, err := s.cmmsRouter.GetWorkOrders(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get work orders", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if workOrders == nil {
		workOrders = []models.WorkOrder{}
	}
	jsonResponse(w, http.StatusOK, workOrders)
}

func (s *Server) createWorkOrder(w http.ResponseWriter, r *http.Request) {
	var wo models.WorkOrder
	if err := json.NewDecoder(r.Body).Decode(&wo); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if wo.DeviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}
	if wo.Type == "" {
		respondError(w, r, NewBadRequestError("type is required"))
		return
	}

	// Проверка существования device_id перед INSERT (work_orders FK fix)
	// Предотвращает 500 ошибку при FOREIGN KEY violation
	// Сначала проверяем БД, потом stateManager (in-memory device может прийти через P2P)
	ctx := r.Context()
	var devExists bool
	dbErr := s.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE device_id = $1)`, wo.DeviceID).Scan(&devExists)
	if dbErr != nil || !devExists {
		// Fallback: проверяем stateManager (устройства, зарегистрированные через P2P/GB28181)
		if _, ok := s.stateManager.Get(wo.DeviceID); !ok {
			respondError(w, r, NewBadRequestError("device_id not found: "+wo.DeviceID))
			return
		}
	}

	if wo.Checklist == nil {
		wo.Checklist = json.RawMessage("[]")
	}
	if wo.Priority == "" {
		wo.Priority = "medium"
	}
	if wo.Status == "" {
		wo.Status = "open"
	}
	if wo.SLADeadline != nil && !isFutureDate(*wo.SLADeadline) {
		respondError(w, r, NewBadRequestError("sla_deadline must be in the future"))
		return
	}

	// Устанавливаем SLA deadline
	if sla, err := s.cmmsRouter.GetSLAConfig(ctx, wo.Priority); err == nil {
		deadline := time.Now().Add(time.Duration(sla.ResolutionTimeMinutes) * time.Minute)
		wo.SLADeadline = &deadline
	}

	// Set created_by from context
	userID := getUserIDFromContext(ctx)
	wo.CreatedBy = &userID

	if err := s.cmmsRouter.CreateWorkOrder(ctx, &wo); err != nil {
		s.logger.Error("Failed to create work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	s.logAudit(userID, "create_work_order", "work_order", wo.ID, nil, wo)
	jsonResponse(w, http.StatusCreated, wo)
}

func (s *Server) getWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wo, err := s.cmmsRouter.GetWorkOrder(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Work order not found"))
		return
	}
	jsonResponse(w, http.StatusOK, wo)
}

func (s *Server) updateWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	allowedFields := map[string]bool{
		"type": true, "status": true, "priority": true, "assigned_to": true,
		"sla_deadline": true, "checklist": true, "notes": true,
	}
	for key := range updates {
		if !allowedFields[key] {
			delete(updates, key)
		}
	}
	if err := normalizeDateUpdate(updates, "sla_deadline", true); err != nil {
		respondError(w, r, NewBadRequestError(err.Error()))
		return
	}

	if err := s.cmmsRouter.UpdateWorkOrder(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_work_order", "work_order", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.UpdateWorkOrder(r.Context(), id, map[string]interface{}{"status": "cancelled"}); err != nil {
		s.logger.Error("Failed to delete work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_work_order", "work_order", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Bulk Actions (WO-4.2.1) ─────────────────────────────────────────

// bulkBulkWorkOrdersRequest — запрос на массовую операцию.
type bulkWorkOrdersRequest struct {
	Action string   `json:"action"` // "status_change", "assign", "delete", "priority_change"
	IDs    []string `json:"ids"`
	Value  string   `json:"value"` // новый статус / user_id / priority
}

// handleBulkWorkOrders выполняет массовые операции над Work Orders.
//
// Соответствует:
//   - ISO 27001 A.12.4.1 (Event logging)
//   - OWASP ASVS V5.1 (Input validation)
//   - IEC 62443 SR 3.1 (Data integrity)
func (s *Server) handleBulkWorkOrders(w http.ResponseWriter, r *http.Request) {
	var req bulkWorkOrdersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Whitelist validation: action
	validActions := map[string]bool{
		"status_change": true, "assign": true,
		"delete": true, "priority_change": true,
	}
	if !validActions[req.Action] {
		respondError(w, r, NewBadRequestError("Unsupported action: "+req.Action))
		return
	}

	if len(req.IDs) == 0 {
		respondError(w, r, NewBadRequestError("ids must be a non-empty array"))
		return
	}

	if len(req.IDs) > 100 {
		respondError(w, r, NewBadRequestError("max 100 ids per bulk request"))
		return
	}

	results, err := s.db.BulkWorkOrders(db.BulkActionType(req.Action), req.IDs, req.Value)
	if err != nil {
		s.logger.Error("Bulk action failed", "action", req.Action, "error", err)
		respondError(w, r, NewInternalError("bulk operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "bulk_"+req.Action, "work_order", "", nil, map[string]interface{}{
		"ids":     req.IDs,
		"value":   req.Value,
		"results": results,
	})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   len(results),
		"success": countSuccess(results),
		"failed":  countFailed(results),
	})
}

func countSuccess(results []db.BulkActionResult) int {
	count := 0
	for _, r := range results {
		if r.Status == "success" {
			count++
		}
	}
	return count
}

func countFailed(results []db.BulkActionResult) int {
	count := 0
	for _, r := range results {
		if r.Status == "error" {
			count++
		}
	}
	return count
}

func (s *Server) assignWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		respondError(w, r, NewBadRequestError("user_id is required"))
		return
	}

	if err := s.cmmsRouter.AssignWorkOrder(r.Context(), id, req.UserID); err != nil {
		s.logger.Error("Failed to assign work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "assign_work_order", "work_order", id, nil, map[string]string{"assigned_to": req.UserID})
	jsonResponse(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (s *Server) startWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.StartWorkOrder(r.Context(), id); err != nil {
		s.logger.Error("Failed to start work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "start_work_order", "work_order", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) completeWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Notes  string             `json:"notes"`
		Photos []string           `json:"photos"`
		Parts  []models.PartUsage `json:"parts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	userID := getUserIDFromContext(r.Context())
	if err := s.cmmsRouter.CompleteWorkOrder(r.Context(), id, req.Notes, req.Photos, req.Parts, userID); err != nil {
		s.logger.Error("Failed to complete work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	s.logAudit(userID, "complete_work_order", "work_order", id, nil, req)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "completed"})
}

func (s *Server) cancelWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Reason = "No reason provided"
	}

	if err := s.cmmsRouter.CancelWorkOrder(r.Context(), id, req.Reason); err != nil {
		s.logger.Error("Failed to cancel work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "cancel_work_order", "work_order", id, nil, map[string]string{"reason": req.Reason})
	jsonResponse(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (s *Server) uploadWorkOrderPhotos(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, r, NewBadRequestError("Failed to parse multipart form"))
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		respondError(w, r, NewBadRequestError("No photos provided"))
		return
	}

	var photoURLs []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			continue
		}
		defer file.Close()

		// Save to images directory
		filename := generateFilename(fileHeader.Filename)
		dst := s.imagesDir + "/" + filename

		if err := saveUploadedFile(file, dst); err != nil {
			s.logger.Error("Failed to save photo", "error", err)
			continue
		}

		photoURLs = append(photoURLs, "/api/v1/images/"+filename)
	}

	// Update work order photos
	wo, err := s.cmmsRouter.GetWorkOrder(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Work order not found"))
		return
	}

	var existingPhotos []string
	if wo.Photos != nil {
		json.Unmarshal(wo.Photos, &existingPhotos)
	}
	existingPhotos = append(existingPhotos, photoURLs...)

	photosJSON, _ := json.Marshal(existingPhotos)
	if err := s.cmmsRouter.UpdateWorkOrder(r.Context(), id, map[string]interface{}{"photos": photosJSON}); err != nil {
		s.logger.Error("Failed to update work order photos", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{"photos": existingPhotos})
}

func (s *Server) addWorkOrderParts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Parts []models.PartUsage `json:"parts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Parts) == 0 {
		respondError(w, r, NewBadRequestError("parts array is required"))
		return
	}

	userID := getUserIDFromContext(r.Context())
	for _, part := range req.Parts {
		if err := s.cmmsRouter.UsePartInWorkOrder(r.Context(), id, part.PartID, part.Quantity, userID); err != nil {
			s.logger.Error("Failed to use part in work order", "error", err)
			respondError(w, r, NewInternalError("operation failed", err))
			return
		}
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "parts_added"})
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Parts Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listSpareParts(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})

	if category := r.URL.Query().Get("category"); category != "" {
		filters["category"] = category
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters["offset"] = o
		}
	}

	parts, err := s.cmmsRouter.GetSpareParts(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get spare parts", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if parts == nil {
		parts = []models.SparePart{}
	}
	jsonResponse(w, http.StatusOK, parts)
}

func (s *Server) createSparePart(w http.ResponseWriter, r *http.Request) {
	var part models.SparePart
	if err := json.NewDecoder(r.Body).Decode(&part); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if part.Name == "" {
		respondError(w, r, NewBadRequestError("name is required"))
		return
	}

	if err := s.cmmsRouter.CreateSparePart(r.Context(), &part); err != nil {
		s.logger.Error("Failed to create spare part", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_spare_part", "spare_part", part.ID, nil, part)
	jsonResponse(w, http.StatusCreated, part)
}

func (s *Server) getSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	part, err := s.cmmsRouter.GetSparePart(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Spare part not found"))
		return
	}
	jsonResponse(w, http.StatusOK, part)
}

func (s *Server) updateSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	allowedFields := map[string]bool{
		"name": true, "sku": true, "category": true, "stock": true,
		"min_stock": true, "location": true, "compatible_devices": true,
		"cost": true, "supplier": true, "custom_fields": true,
	}
	for key := range updates {
		if !allowedFields[key] {
			delete(updates, key)
		}
	}

	if err := s.cmmsRouter.UpdateSparePart(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update spare part", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_spare_part", "spare_part", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.DeleteSparePart(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete spare part", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_spare_part", "spare_part", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) getLowStockParts(w http.ResponseWriter, r *http.Request) {
	parts, err := s.cmmsRouter.GetLowStockParts(r.Context())
	if err != nil {
		s.logger.Error("Failed to get low stock parts", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if parts == nil {
		parts = []models.SparePart{}
	}
	jsonResponse(w, http.StatusOK, parts)
}

// adjustSparePartStock корректирует остаток запчасти с audit trail (INV-7.1.4).
//
// Читает текущий остаток, обновляет его, создаёт запись в stock_adjustments.
// Все изменения логируются в audit_log с HMAC-подписью.
//
// POST /api/v1/spare-parts/{id}/adjust
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — stock adjustment audit trail)
//   - ISO/IEC 27019 PCC.A.12 (Operations security — inventory changes)
//   - СТБ 34.101.27 (Защита информации — фиксация складских операций)
//   - OWASP ASVS V5.1 (Input validation — whitelist reason)
func (s *Server) adjustSparePartStock(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Quantity int    `json:"quantity"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if req.Quantity < 0 {
		respondError(w, r, NewBadRequestError("quantity must be >= 0"))
		return
	}

	userID := getUserIDFromContext(r.Context())

	// Получаем текущий spare part для previous_stock
	part, err := s.cmmsRouter.GetSparePart(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get spare part for adjustment", "id", id, "error", err)
		respondError(w, r, NewNotFoundError("Spare part not found"))
		return
	}

	previousStock := part.Stock
	delta := req.Quantity - previousStock

	// Обновляем остаток
	if err := s.cmmsRouter.UpdateSparePartStock(r.Context(), id, req.Quantity); err != nil {
		s.logger.Error("Failed to adjust spare part stock", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// Создаём запись в stock_adjustments (audit trail)
	adj := &models.StockAdjustment{
		PartID:        id,
		PreviousStock: previousStock,
		NewStock:      req.Quantity,
		Delta:         delta,
		Reason:        req.Reason,
		AdjustedBy:    userID,
	}
	if err := s.db.CreateStockAdjustment(adj); err != nil {
		s.logger.Error("Failed to create stock adjustment record", "error", err)
		// Не возвращаем ошибку — stock уже обновлён, только audit запись не создалась
	}

	// Audit log (ISO 27001 A.12.4)
	s.logAudit(userID, "adjust_spare_part_stock", "spare_part", id, nil, map[string]interface{}{
		"previous_stock": previousStock,
		"new_stock":      req.Quantity,
		"delta":          delta,
		"reason":         req.Reason,
		"adjustment_id":  adj.ID,
	})

	jsonResponse(w, http.StatusOK, map[string]string{"status": "adjusted"})
}

// listSparePartStockAdjustments возвращает историю корректировок остатка запчасти (INV-7.1.4).
//
// GET /api/v1/spare-parts/{id}/adjustments
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Parameterized query — в DB слое)
func (s *Server) listSparePartStockAdjustments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	adjustments, err := s.db.GetStockAdjustments(id)
	if err != nil {
		s.logger.Error("Failed to get stock adjustments", "part_id", id, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if adjustments == nil {
		adjustments = []models.StockAdjustment{}
	}
	jsonResponse(w, http.StatusOK, adjustments)
}

// ═══════════════════════════════════════════════════════════════════════
// Technician Management Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) getAllTechnicianWorkloads(w http.ResponseWriter, r *http.Request) {
	workloads, err := s.cmmsRouter.GetAllTechnicianWorkloads(r.Context())
	if err != nil {
		s.logger.Error("Failed to get technician workloads", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if workloads == nil {
		workloads = []models.TechnicianWorkload{}
	}
	jsonResponse(w, http.StatusOK, workloads)
}

func (s *Server) getTechnicianWorkload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workload, err := s.cmmsRouter.GetTechnicianWorkload(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Technician not found"))
		return
	}
	jsonResponse(w, http.StatusOK, workload)
}

func (s *Server) updateTechnicianSkills(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Skills         []string `json:"skills"`
		Certifications []string `json:"certifications"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if err := s.cmmsRouter.UpdateTechnicianSkills(r.Context(), id, req.Skills, req.Certifications); err != nil {
		s.logger.Error("Failed to update technician skills", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_technician_skills", "user", id, nil, req)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// ═══════════════════════════════════════════════════════════════════════
// SLA & Reports Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) getSLAConfig(w http.ResponseWriter, r *http.Request) {
	configs, err := s.cmmsRouter.GetAllSLAConfigs(r.Context())
	if err != nil {
		s.logger.Error("Failed to get SLA configs", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if configs == nil {
		configs = []models.SLAConfig{}
	}
	jsonResponse(w, http.StatusOK, configs)
}

func (s *Server) updateSLAConfig(w http.ResponseWriter, r *http.Request) {
	priority := chi.URLParam(r, "priority")

	var req struct {
		ResponseTimeMinutes   int `json:"response_time_minutes"`
		ResolutionTimeMinutes int `json:"resolution_time_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if req.ResponseTimeMinutes <= 0 || req.ResolutionTimeMinutes <= 0 {
		respondError(w, r, NewBadRequestError("Times must be positive"))
		return
	}

	if err := s.cmmsRouter.UpdateSLAConfig(r.Context(), priority, req.ResponseTimeMinutes, req.ResolutionTimeMinutes); err != nil {
		s.logger.Error("Failed to update SLA config", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_sla_config", "sla_config", priority, nil, req)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) getMaintenanceReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.cmmsRouter.GetMaintenanceReport(r.Context())
	if err != nil {
		s.logger.Error("Failed to get maintenance report", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if report == nil {
		report = []models.MaintenanceReport{}
	}
	jsonResponse(w, http.StatusOK, report)
}

func (s *Server) getSLAComplianceReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.cmmsRouter.GetSLAComplianceReport(r.Context())
	if err != nil {
		s.logger.Error("Failed to get SLA compliance report", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if report == nil {
		report = []models.SLAComplianceReport{}
	}
	jsonResponse(w, http.StatusOK, report)
}

// ═══════════════════════════════════════════════════════════════════════
// Technician Site Assignments Handlers
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listTechnicianSiteAssignments(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})

	if technicianID := r.URL.Query().Get("technician_id"); technicianID != "" {
		filters["technician_id"] = technicianID
	}
	if siteID := r.URL.Query().Get("site_id"); siteID != "" {
		filters["site_id"] = siteID
	}
	if isPrimary := r.URL.Query().Get("is_primary"); isPrimary != "" {
		filters["is_primary"] = isPrimary == "true"
	}

	assignments, err := s.cmmsRouter.GetTechnicianSiteAssignments(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get technician site assignments", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if assignments == nil {
		assignments = []models.TechnicianSiteAssignment{}
	}
	jsonResponse(w, http.StatusOK, assignments)
}

func (s *Server) createTechnicianSiteAssignment(w http.ResponseWriter, r *http.Request) {
	var assignment models.TechnicianSiteAssignment
	if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if assignment.TechnicianID == "" {
		respondError(w, r, NewBadRequestError("technician_id is required"))
		return
	}
	if assignment.SiteID == "" {
		respondError(w, r, NewBadRequestError("site_id is required"))
		return
	}

	// Get assigned_by from context; если claims нет — оставляем nil, а не пустую строку
	claims := auth.GetClaims(r)
	if claims != nil && claims.UserID != "" {
		assignment.AssignedBy = claims.UserID
	} else {
		assignment.AssignedBy = "" // БД-метод должен обработать пустую строку как NULL
	}

	if err := s.cmmsRouter.CreateTechnicianSiteAssignment(r.Context(), &assignment); err != nil {
		s.logger.Error("Failed to create technician site assignment", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// Audit log
	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_technician_site_assignment", "technician_site_assignment", assignment.ID, nil, assignment)

	jsonResponse(w, http.StatusCreated, assignment)
}

func (s *Server) updateTechnicianSiteAssignment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Only allow updating is_primary field
	allowedUpdates := map[string]interface{}{}
	if isPrimary, ok := updates["is_primary"]; ok {
		allowedUpdates["is_primary"] = isPrimary
	}

	if len(allowedUpdates) == 0 {
		respondError(w, r, NewBadRequestError("No valid fields to update"))
		return
	}

	if err := s.cmmsRouter.UpdateTechnicianSiteAssignment(r.Context(), id, allowedUpdates); err != nil {
		s.logger.Error("Failed to update technician site assignment", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// Audit log
	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_technician_site_assignment", "technician_site_assignment", id, nil, allowedUpdates)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteTechnicianSiteAssignment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	if err := s.cmmsRouter.DeleteTechnicianSiteAssignment(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete technician site assignment", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	// Audit log
	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_technician_site_assignment", "technician_site_assignment", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════════════════

func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "system"
}

func normalizeDateUpdate(updates map[string]interface{}, field string, requireFuture bool) error {
	value, ok := updates[field]
	if !ok || value == nil {
		return nil
	}

	raw, ok := value.(string)
	if !ok {
		return fmt.Errorf("%s must be a string", field)
	}

	parsed, err := parseValidatedDate(raw, field)
	if err != nil {
		return err
	}
	if requireFuture && !isFutureDate(parsed) {
		return fmt.Errorf("%s must be in the future", field)
	}

	updates[field] = parsed
	return nil
}

func parseFutureDate(value, field string) (time.Time, error) {
	parsed, err := parseValidatedDate(value, field)
	if err != nil {
		return time.Time{}, err
	}
	if !isFutureDate(parsed) {
		return time.Time{}, fmt.Errorf("%s must be in the future", field)
	}
	return parsed, nil
}

// parseValidatedDate парсит дату и проверяет диапазон year >= 2020 && year <= 2035.
// Соответствует: OWASP ASVS V5.1.1 (Input validation), ISO 27001 A.14.2.5, СТБ 34.101.27 п. 6.2
func parseValidatedDate(value, field string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("%s is required", field)
	}
	if len(value) > 64 {
		return time.Time{}, fmt.Errorf("%s is too long", field)
	}

	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			if parsed.IsZero() {
				return time.Time{}, fmt.Errorf("%s must be a valid date", field)
			}
			// Валидация диапазона года (OWASP ASVS V5.1.1)
			year := parsed.Year()
			if year < 2020 || year > 2035 {
				return time.Time{}, fmt.Errorf("%s year %d is out of range (must be 2020-2035)", field, year)
			}
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("%s must use RFC3339 or YYYY-MM-DD format", field)
}

func isFutureDate(value time.Time) bool {
	return value.After(time.Now().UTC())
}

func (s *Server) logAudit(userID, action, entityType, entityID string, oldValue, newValue interface{}) {
	// Если userID пустой или не найден — используем 'system' (audit_log_user_id_fkey fix)
	if userID == "" {
		userID = "system"
	}

	var oldJSON, newJSON []byte
	if oldValue != nil {
		oldJSON, _ = json.Marshal(oldValue)
	}
	if newValue != nil {
		newJSON, _ = json.Marshal(newValue)
	}

	// HMAC-подпись для целостности журнала аудита (ISO 27001 A.12.4)
	var hmacSig string
	if s.auditSigner != nil {
		data := audit.SignAuditEntry(userID, action, entityType, entityID, oldJSON, newJSON)
		hmacSig = s.auditSigner.Sign(data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO audit_log (user_id, action, entity_type, entity_id, old_value, new_value, hmac_signature)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
	`, userID, action, entityType, entityID, oldJSON, newJSON, hmacSig)
	if err != nil {
		s.logger.Error("Failed to log audit", "error", err)
	}
}

func generateFilename(original string) string {
	return strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + original
}

func saveUploadedFile(src io.Reader, dst string) error {
	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// ═══════════════════════════════════════════════════════════════════════
// Time Entry Handlers (WO-4.4.1)
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listTimeEntries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	entries, err := s.db.GetTimeEntries(id)
	if err != nil {
		s.logger.Error("Failed to get time entries", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	if entries == nil {
		entries = []models.TimeEntry{}
	}
	jsonResponse(w, http.StatusOK, entries)
}

func (s *Server) createTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	var req struct {
		Notes      string  `json:"notes"`
		HourlyRate float64 `json:"hourly_rate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	entry := &models.TimeEntry{
		WorkOrderID: id,
		UserID:      userID,
		StartTime:   time.Now(),
		Status:      "running",
		Notes:       req.Notes,
		HourlyRate:  req.HourlyRate,
	}

	if err := s.db.CreateTimeEntry(entry); err != nil {
		s.logger.Error("Failed to create time entry", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusCreated, entry)
}

func (s *Server) pauseTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.handleTimeEntryStatusChange(w, r, id, "paused")
}

func (s *Server) resumeTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.handleTimeEntryStatusChange(w, r, id, "running")
}

func (s *Server) stopTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.handleTimeEntryStatusChange(w, r, id, "stopped")
}

func (s *Server) handleTimeEntryStatusChange(w http.ResponseWriter, r *http.Request, id, status string) {
	if err := s.db.UpdateTimeEntryStatus(id, status); err != nil {
		s.logger.Error("Failed to update time entry", "id", id, "status", status, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": status})
}

func (s *Server) deleteTimeEntry(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.db.DeleteTimeEntry(id); err != nil {
		s.logger.Error("Failed to delete time entry", "id", id, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Labor Cost (WO-4.4.2) ──────────────────────────────────────────

func (s *Server) getLaborCost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	lc, err := s.db.GetLaborCost(id)
	if err != nil {
		s.logger.Error("Failed to get labor cost", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}
	jsonResponse(w, http.StatusOK, lc)
}

// ── Parts Consumption with Cost Snapshot (WO-4.4.4) ────────────────

func (s *Server) addPartWithCost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	var req struct {
		PartID   string `json:"part_id"`
		Quantity int    `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if req.PartID == "" || req.Quantity <= 0 {
		respondError(w, r, NewBadRequestError("part_id and quantity > 0 are required"))
		return
	}

	if err := s.db.AddPartToWorkOrderWithCost(id, req.PartID, req.Quantity, userID); err != nil {
		s.logger.Error("Failed to add part with cost", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "added"})
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrder ↔ Alert Handlers (DM-1.3.1)
// ═══════════════════════════════════════════════════════════════════════

// linkAlertToWorkOrder привязывает алерт к WorkOrder.
//
// POST /api/v1/work-orders/{id}/alerts
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Audit trail for linking)
//   - OWASP ASVS V5.1 (Input validation)
func (s *Server) linkAlertToWorkOrder(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	var req struct {
		AlertID string `json:"alert_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}
	if req.AlertID == "" {
		respondError(w, r, NewBadRequestError("alert_id is required"))
		return
	}

	if err := s.cmmsRouter.LinkAlertToWorkOrder(r.Context(), workOrderID, req.AlertID, userID); err != nil {
		s.logger.Error("Failed to link alert to work order", "work_order_id", workOrderID, "alert_id", req.AlertID, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	s.logAudit(userID, "link_alert_to_work_order", "work_order_alert", workOrderID, nil, map[string]string{
		"alert_id": req.AlertID,
	})

	jsonResponse(w, http.StatusOK, map[string]string{"status": "linked"})
}

// unlinkAlertFromWorkOrder отвязывает алерт от WorkOrder.
//
// DELETE /api/v1/work-orders/{id}/alerts/{alertId}
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Audit trail for unlinking)
//   - OWASP ASVS V5.1 (Input validation)
func (s *Server) unlinkAlertFromWorkOrder(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	alertID := chi.URLParam(r, "alertId")

	if alertID == "" {
		respondError(w, r, NewBadRequestError("alert_id is required"))
		return
	}

	if err := s.cmmsRouter.UnlinkAlertFromWorkOrder(r.Context(), workOrderID, alertID); err != nil {
		s.logger.Error("Failed to unlink alert from work order", "work_order_id", workOrderID, "alert_id", alertID, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "unlink_alert_from_work_order", "work_order_alert", workOrderID, nil, map[string]string{
		"alert_id": alertID,
	})

	jsonResponse(w, http.StatusOK, map[string]string{"status": "unlinked"})
}

// listAlertsForWorkOrder возвращает список алертов, привязанных к WorkOrder.
//
// GET /api/v1/work-orders/{id}/alerts
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Input validation)
func (s *Server) listAlertsForWorkOrder(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")

	alerts, err := s.cmmsRouter.GetAlertsForWorkOrder(r.Context(), workOrderID)
	if err != nil {
		s.logger.Error("Failed to get alerts for work order", "work_order_id", workOrderID, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, alerts)
}

// ═══════════════════════════════════════════════════════════════════════
// Vendor Handlers (INV-7.2.1)
// ═══════════════════════════════════════════════════════════════════════

func (s *Server) listVendors(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filters["offset"] = o
		}
	}

	vendors, err := s.cmmsRouter.GetVendors(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get vendors", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if vendors == nil {
		vendors = []models.Vendor{}
	}
	jsonResponse(w, http.StatusOK, vendors)
}

func (s *Server) createVendor(w http.ResponseWriter, r *http.Request) {
	var vendor models.Vendor
	if err := json.NewDecoder(r.Body).Decode(&vendor); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if vendor.Name == "" {
		respondError(w, r, NewBadRequestError("name is required"))
		return
	}

	if vendor.Status == "" {
		vendor.Status = "active"
	}

	if err := s.cmmsRouter.CreateVendor(r.Context(), &vendor); err != nil {
		s.logger.Error("Failed to create vendor", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_vendor", "vendor", vendor.ID, nil, vendor)
	jsonResponse(w, http.StatusCreated, vendor)
}

func (s *Server) getVendor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	vendor, err := s.cmmsRouter.GetVendor(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Vendor not found"))
		return
	}
	if vendor == nil {
		respondError(w, r, NewNotFoundError("Vendor not found"))
		return
	}
	jsonResponse(w, http.StatusOK, vendor)
}

func (s *Server) updateVendor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	allowedFields := map[string]bool{
		"name": true, "contact_person": true, "email": true,
		"phone": true, "address": true, "website": true,
		"notes": true, "status": true,
	}
	for key := range updates {
		if !allowedFields[key] {
			delete(updates, key)
		}
	}

	if err := s.cmmsRouter.UpdateVendor(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update vendor", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_vendor", "vendor", id, nil, updates)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteVendor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.DeleteVendor(r.Context(), id); err != nil {
		s.logger.Error("Failed to delete vendor", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_vendor", "vendor", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// WO-4.4.3: AdditionalCost (travel, subcontractor, permits)
// ═══════════════════════════════════════════════════════════════════════

// listAdditionalCosts возвращает дополнительные затраты для Work Order.
//
// GET /api/v1/work-orders/{id}/additional-costs
//
// Compliance:
//   - OWASP ASVS V4.1 (RBAC — admin/manager/technician)
//   - OWASP ASVS V5.1 (Path parameter validation)
func (s *Server) listAdditionalCosts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	costs, err := s.db.GetAdditionalCostsByWorkOrder(r.Context(), id)
	if err != nil {
		s.logger.Error("Failed to get additional costs", "work_order_id", id, "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, costs)
}

// createAdditionalCost создаёт запись дополнительных затрат.
//
// POST /api/v1/work-orders/{id}/additional-costs
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через category enum)
//   - OWASP ASVS V7.1 (Error handling — no sensitive data)
func (s *Server) createAdditionalCost(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	var req struct {
		Category      string  `json:"category"`
		Description   string  `json:"description"`
		VendorName    string  `json:"vendor_name"`
		EstimatedCost float64 `json:"estimated_cost"`
		ActualCost    float64 `json:"actual_cost"`
		Currency      string  `json:"currency"`
		ReceiptURL    string  `json:"receipt_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Whitelist validation (OWASP ASVS V5.1)
	validCategory := false
	for _, c := range models.ValidAdditionalCostCategories {
		if req.Category == c {
			validCategory = true
			break
		}
	}
	if !validCategory {
		respondError(w, r, NewBadRequestError("Invalid category: must be one of travel, subcontractor, permit, equipment, other"))
		return
	}

	cost := &models.AdditionalCost{
		CostBase: models.CostBase{
			EstimatedCost: req.EstimatedCost,
			ActualCost:    req.ActualCost,
			Currency:      req.Currency,
		},
		ID:          fmt.Sprintf("ac_%s_%d", id, time.Now().UnixNano()),
		WorkOrderID: id,
		Category:    req.Category,
		Description: req.Description,
		VendorName:  req.VendorName,
		ReceiptURL:  req.ReceiptURL,
		CreatedBy:   &userID,
	}

	if err := s.db.CreateAdditionalCost(r.Context(), cost); err != nil {
		s.logger.Error("Failed to create additional cost", "error", err)
		respondError(w, r, NewInternalError("operation failed", nil))
		return
	}

	s.logAudit(userID, "create_additional_cost", "work_order", id, nil, map[string]interface{}{
		"category": req.Category,
		"cost":     req.ActualCost,
	})
	jsonResponse(w, http.StatusCreated, cost)
}

// deleteAdditionalCost удаляет запись дополнительных затрат.
//
// DELETE /api/v1/additional-costs/{id}
//
// Compliance:
//   - OWASP ASVS V4.1 (RBAC — admin only for deletion)
func (s *Server) deleteAdditionalCost(w http.ResponseWriter, r *http.Request) {
	costID := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	if err := s.db.DeleteAdditionalCost(r.Context(), costID); err != nil {
		s.logger.Error("Failed to delete additional cost", "id", costID, "error", err)
		respondError(w, r, NewInternalError("operation failed", nil))
		return
	}

	s.logAudit(userID, "delete_additional_cost", "additional_cost", costID, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Auto-dispatcher Handlers (P1-6)
// ═══════════════════════════════════════════════════════════════════════

// handleAutoAssign запускает автоматическое назначение техника на Work Order.
//
// POST /api/v1/dispatcher/auto-assign/{workOrderId}
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - OWASP ASVS V5.1 (Path parameter validation)
func (s *Server) handleAutoAssign(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "workOrderId")
	if workOrderID == "" {
		respondError(w, r, NewBadRequestError("workOrderId is required"))
		return
	}

	if s.autoDispatcher == nil {
		respondError(w, r, NewInternalError("auto-dispatcher not initialized", nil))
		return
	}

	result, err := s.autoDispatcher.AutoAssign(r.Context(), workOrderID)
	if err != nil {
		s.logger.Error("auto-assign failed", "work_order_id", workOrderID, "error", err)
		respondError(w, r, NewInternalError("auto-assign operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "auto_assign", "work_order", workOrderID, nil, result)

	status := http.StatusOK
	if result.Status == "no_technician" {
		status = http.StatusAccepted
	}
	jsonResponse(w, status, result)
}

// handleListDispatchRules возвращает список правил диспетчеризации.
//
// GET /api/v1/dispatcher/rules
func (s *Server) handleListDispatchRules(w http.ResponseWriter, r *http.Request) {
	if s.ruleEngine == nil {
		respondError(w, r, NewInternalError("rule engine not initialized", nil))
		return
	}

	rules := s.ruleEngine.GetRules()
	if rules == nil {
		rules = []cmms.DispatchRule{}
	}
	jsonResponse(w, http.StatusOK, rules)
}

// handleCreateDispatchRule создаёт новое правило диспетчеризации.
//
// POST /api/v1/dispatcher/rules
func (s *Server) handleCreateDispatchRule(w http.ResponseWriter, r *http.Request) {
	if s.ruleEngine == nil {
		respondError(w, r, NewInternalError("rule engine not initialized", nil))
		return
	}

	var rule cmms.DispatchRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if err := s.ruleEngine.AddRule(rule); err != nil {
		s.logger.Error("failed to create dispatch rule", "error", err)
		respondError(w, r, NewBadRequestError(err.Error()))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_dispatch_rule", "dispatch_rule", rule.ID, nil, rule)

	jsonResponse(w, http.StatusCreated, rule)
}

// handleBatchAutoAssign запускает batch-назначение для всех непривязанных WO.
//
// POST /api/v1/dispatcher/batch-assign
func (s *Server) handleBatchAutoAssign(w http.ResponseWriter, r *http.Request) {
	if s.autoDispatcher == nil {
		respondError(w, r, NewInternalError("auto-dispatcher not initialized", nil))
		return
	}

	result, err := s.autoDispatcher.BatchAutoAssign(r.Context())
	if err != nil {
		s.logger.Error("batch auto-assign failed", "error", err)
		respondError(w, r, NewInternalError("batch auto-assign failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "batch_auto_assign", "work_order", "", nil, map[string]interface{}{
		"total":    result.Total,
		"assigned": result.Assigned,
		"failed":   result.Failed,
		"skipped":  result.Skipped,
	})

	jsonResponse(w, http.StatusOK, result)
}

// handleRunEscalationCheck запускает проверку эскалации для всех непривязанных WO.
//
// POST /api/v1/dispatcher/escalation-check
//
// Compliance:
//   - Приказ ОАЦ №66 п. 7.18.3 (Incident response)
func (s *Server) handleRunEscalationCheck(w http.ResponseWriter, r *http.Request) {
	if s.autoDispatcher == nil {
		respondError(w, r, NewInternalError("auto-dispatcher not initialized", nil))
		return
	}

	results, err := s.autoDispatcher.RunEscalationCheck(r.Context())
	if err != nil {
		s.logger.Error("escalation check failed", "error", err)
		respondError(w, r, NewInternalError("escalation check failed", err))
		return
	}

	if results == nil {
		results = []cmms.EscalationResult{}
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "escalation_check", "work_order", "", nil, map[string]interface{}{
		"escalated": len(results),
	})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"escalated": len(results),
		"results":   results,
	})
}

// handleUpdateDispatchRule обновляет существующее правило.
//
// PUT /api/v1/dispatcher/rules/{id}
func (s *Server) handleUpdateDispatchRule(w http.ResponseWriter, r *http.Request) {
	if s.ruleEngine == nil {
		respondError(w, r, NewInternalError("rule engine not initialized", nil))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if err := s.ruleEngine.UpdateRule(id, updates); err != nil {
		s.logger.Error("failed to update dispatch rule", "id", id, "error", err)
		respondError(w, r, NewBadRequestError(err.Error()))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_dispatch_rule", "dispatch_rule", id, nil, updates)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleDeleteDispatchRule удаляет правило.
//
// DELETE /api/v1/dispatcher/rules/{id}
func (s *Server) handleDeleteDispatchRule(w http.ResponseWriter, r *http.Request) {
	if s.ruleEngine == nil {
		respondError(w, r, NewInternalError("rule engine not initialized", nil))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		respondError(w, r, NewBadRequestError("id is required"))
		return
	}

	if err := s.ruleEngine.DeleteRule(id); err != nil {
		s.logger.Error("failed to delete dispatch rule", "id", id, "error", err)
		respondError(w, r, NewNotFoundError("Rule not found"))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_dispatch_rule", "dispatch_rule", id, nil, nil)

	jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}
