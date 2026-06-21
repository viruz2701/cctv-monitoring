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
	respondJSON(w, http.StatusOK, schedules)
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

	// Парсим дату в нескольких форматах (RFC3339, YYYY-MM-DD)
	nextDue, err := parseFlexibleDate(raw.NextDue)
	if err != nil {
		respondError(w, r, NewBadRequestError("invalid next_due format, use RFC3339 or YYYY-MM-DD"))
		return
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

	respondJSON(w, http.StatusCreated, schedule)
}

func (s *Server) getMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	schedule, err := s.cmmsRouter.GetMaintenanceSchedule(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Schedule not found"))
		return
	}
	respondJSON(w, http.StatusOK, schedule)
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

	if err := s.cmmsRouter.UpdateMaintenanceSchedule(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update maintenance schedule", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_maintenance_schedule", "maintenance_schedule", id, nil, updates)

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
	respondJSON(w, http.StatusOK, schedules)
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "completed"})
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
	respondJSON(w, http.StatusOK, workOrders)
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
	if wo.Checklist == nil {
		wo.Checklist = json.RawMessage("[]")
	}
	if wo.Priority == "" {
		wo.Priority = "medium"
	}
	if wo.Status == "" {
		wo.Status = "open"
	}

	// Устанавливаем SLA deadline
	if sla, err := s.cmmsRouter.GetSLAConfig(r.Context(), wo.Priority); err == nil {
		deadline := time.Now().Add(time.Duration(sla.ResolutionTimeMinutes) * time.Minute)
		wo.SLADeadline = &deadline
	}

	// Set created_by from context
	userID := getUserIDFromContext(r.Context())
	wo.CreatedBy = &userID

	if err := s.cmmsRouter.CreateWorkOrder(r.Context(), &wo); err != nil {
		s.logger.Error("Failed to create work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	s.logAudit(userID, "create_work_order", "work_order", wo.ID, nil, wo)
	respondJSON(w, http.StatusCreated, wo)
}

func (s *Server) getWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wo, err := s.cmmsRouter.GetWorkOrder(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Work order not found"))
		return
	}
	respondJSON(w, http.StatusOK, wo)
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

	if err := s.cmmsRouter.UpdateWorkOrder(r.Context(), id, updates); err != nil {
		s.logger.Error("Failed to update work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_work_order", "work_order", id, nil, updates)
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "started"})
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "completed"})
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
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

	respondJSON(w, http.StatusOK, map[string]interface{}{"photos": existingPhotos})
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "parts_added"})
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
	respondJSON(w, http.StatusOK, parts)
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
	respondJSON(w, http.StatusCreated, part)
}

func (s *Server) getSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	part, err := s.cmmsRouter.GetSparePart(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Spare part not found"))
		return
	}
	respondJSON(w, http.StatusOK, part)
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
		"cost": true, "supplier": true,
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
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
	respondJSON(w, http.StatusOK, parts)
}

func (s *Server) adjustSparePartStock(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if err := s.cmmsRouter.UpdateSparePartStock(r.Context(), id, req.Quantity); err != nil {
		s.logger.Error("Failed to adjust spare part stock", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "adjust_spare_part_stock", "spare_part", id, nil, map[string]int{"quantity": req.Quantity})
	respondJSON(w, http.StatusOK, map[string]string{"status": "adjusted"})
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
	respondJSON(w, http.StatusOK, workloads)
}

func (s *Server) getTechnicianWorkload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workload, err := s.cmmsRouter.GetTechnicianWorkload(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Technician not found"))
		return
	}
	respondJSON(w, http.StatusOK, workload)
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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
	respondJSON(w, http.StatusOK, configs)
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
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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
	respondJSON(w, http.StatusOK, report)
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
	respondJSON(w, http.StatusOK, report)
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
	respondJSON(w, http.StatusOK, assignments)
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

	respondJSON(w, http.StatusCreated, assignment)
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ═══════════════════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════════════════

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return "system"
}

// parseFlexibleDate парсит дату в форматах RFC3339, YYYY-MM-DD, YYYY-MM-DDTHH:MM:SS
func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func (s *Server) logAudit(userID, action, entityType, entityID string, oldValue, newValue interface{}) {
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
