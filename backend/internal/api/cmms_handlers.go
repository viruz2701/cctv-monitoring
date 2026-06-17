package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

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

	schedules, err := s.db.GetMaintenanceSchedules(filters)
	if err != nil {
		s.logger.Error("Failed to get maintenance schedules", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if schedules == nil {
		schedules = []models.MaintenanceSchedule{}
	}
	respondJSON(w, http.StatusOK, schedules)
}

func (s *Server) createMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	var schedule models.MaintenanceSchedule
	if err := json.NewDecoder(r.Body).Decode(&schedule); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if schedule.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if schedule.ScheduleType == "" {
		http.Error(w, "schedule_type is required", http.StatusBadRequest)
		return
	}
	if schedule.NextDue.IsZero() {
		http.Error(w, "next_due is required", http.StatusBadRequest)
		return
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

	if err := s.db.CreateMaintenanceSchedule(&schedule); err != nil {
		s.logger.Error("Failed to create maintenance schedule", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_maintenance_schedule", "maintenance_schedule", schedule.ID, nil, schedule)

	respondJSON(w, http.StatusCreated, schedule)
}

func (s *Server) getMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	schedule, err := s.db.GetMaintenanceSchedule(id)
	if err != nil {
		http.Error(w, "Schedule not found", http.StatusNotFound)
		return
	}
	respondJSON(w, http.StatusOK, schedule)
}

func (s *Server) updateMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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

	if err := s.db.UpdateMaintenanceSchedule(id, updates); err != nil {
		s.logger.Error("Failed to update maintenance schedule", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_maintenance_schedule", "maintenance_schedule", id, nil, updates)

	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.DeleteMaintenanceSchedule(id); err != nil {
		s.logger.Error("Failed to delete maintenance schedule", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_maintenance_schedule", "maintenance_schedule", id, nil, nil)

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) getDueSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.db.GetDueSchedules()
	if err != nil {
		s.logger.Error("Failed to get due schedules", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if schedules == nil {
		schedules = []models.MaintenanceSchedule{}
	}
	respondJSON(w, http.StatusOK, schedules)
}

func (s *Server) completeMaintenanceSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.CompleteMaintenanceSchedule(id); err != nil {
		s.logger.Error("Failed to complete maintenance schedule", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаём work order из завершённого графика
	schedule, err := s.db.GetMaintenanceSchedule(id)
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
		if sla, err := s.db.GetSLAConfig(schedule.Priority); err == nil {
			deadline := time.Now().Add(time.Duration(sla.ResolutionTimeMinutes) * time.Minute)
			wo.SLADeadline = &deadline
		}
		if err := s.db.CreateWorkOrder(wo); err != nil {
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

	workOrders, err := s.db.GetWorkOrders(filters)
	if err != nil {
		s.logger.Error("Failed to get work orders", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if wo.DeviceID == "" {
		http.Error(w, "device_id is required", http.StatusBadRequest)
		return
	}
	if wo.Type == "" {
		http.Error(w, "type is required", http.StatusBadRequest)
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
	if sla, err := s.db.GetSLAConfig(wo.Priority); err == nil {
		deadline := time.Now().Add(time.Duration(sla.ResolutionTimeMinutes) * time.Minute)
		wo.SLADeadline = &deadline
	}

	// Set created_by from context
	userID := getUserIDFromContext(r.Context())
	wo.CreatedBy = &userID

	if err := s.db.CreateWorkOrder(&wo); err != nil {
		s.logger.Error("Failed to create work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.logAudit(userID, "create_work_order", "work_order", wo.ID, nil, wo)
	respondJSON(w, http.StatusCreated, wo)
}

func (s *Server) getWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wo, err := s.db.GetWorkOrder(id)
	if err != nil {
		http.Error(w, "Work order not found", http.StatusNotFound)
		return
	}
	respondJSON(w, http.StatusOK, wo)
}

func (s *Server) updateWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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

	if err := s.db.UpdateWorkOrder(id, updates); err != nil {
		s.logger.Error("Failed to update work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_work_order", "work_order", id, nil, updates)
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.UpdateWorkOrder(id, map[string]interface{}{"status": "cancelled"}); err != nil {
		s.logger.Error("Failed to delete work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	if err := s.db.AssignWorkOrder(id, req.UserID); err != nil {
		s.logger.Error("Failed to assign work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "assign_work_order", "work_order", id, nil, map[string]string{"assigned_to": req.UserID})
	respondJSON(w, http.StatusOK, map[string]string{"status": "assigned"})
}

func (s *Server) startWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.StartWorkOrder(id); err != nil {
		s.logger.Error("Failed to start work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r.Context())
	if err := s.db.CompleteWorkOrder(id, req.Notes, req.Photos, req.Parts, userID); err != nil {
		s.logger.Error("Failed to complete work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	if err := s.db.CancelWorkOrder(id, req.Reason); err != nil {
		s.logger.Error("Failed to cancel work order", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["photos"]
	if len(files) == 0 {
		http.Error(w, "No photos provided", http.StatusBadRequest)
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
	wo, err := s.db.GetWorkOrder(id)
	if err != nil {
		http.Error(w, "Work order not found", http.StatusNotFound)
		return
	}

	var existingPhotos []string
	if wo.Photos != nil {
		json.Unmarshal(wo.Photos, &existingPhotos)
	}
	existingPhotos = append(existingPhotos, photoURLs...)

	photosJSON, _ := json.Marshal(existingPhotos)
	if err := s.db.UpdateWorkOrder(id, map[string]interface{}{"photos": photosJSON}); err != nil {
		s.logger.Error("Failed to update work order photos", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "parts array is required", http.StatusBadRequest)
		return
	}

	userID := getUserIDFromContext(r.Context())
	for _, part := range req.Parts {
		if err := s.db.UsePartInWorkOrder(id, part.PartID, part.Quantity, userID); err != nil {
			s.logger.Error("Failed to use part in work order", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

	parts, err := s.db.GetSpareParts(filters)
	if err != nil {
		s.logger.Error("Failed to get spare parts", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if part.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	if err := s.db.CreateSparePart(&part); err != nil {
		s.logger.Error("Failed to create spare part", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "create_spare_part", "spare_part", part.ID, nil, part)
	respondJSON(w, http.StatusCreated, part)
}

func (s *Server) getSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	part, err := s.db.GetSparePart(id)
	if err != nil {
		http.Error(w, "Spare part not found", http.StatusNotFound)
		return
	}
	respondJSON(w, http.StatusOK, part)
}

func (s *Server) updateSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
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

	if err := s.db.UpdateSparePart(id, updates); err != nil {
		s.logger.Error("Failed to update spare part", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_spare_part", "spare_part", id, nil, updates)
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) deleteSparePart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.DeleteSparePart(id); err != nil {
		s.logger.Error("Failed to delete spare part", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "delete_spare_part", "spare_part", id, nil, nil)
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) getLowStockParts(w http.ResponseWriter, r *http.Request) {
	parts, err := s.db.GetLowStockParts()
	if err != nil {
		s.logger.Error("Failed to get low stock parts", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateSparePartStock(id, req.Quantity); err != nil {
		s.logger.Error("Failed to adjust spare part stock", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	workloads, err := s.db.GetAllTechnicianWorkloads()
	if err != nil {
		s.logger.Error("Failed to get technician workloads", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if workloads == nil {
		workloads = []models.TechnicianWorkload{}
	}
	respondJSON(w, http.StatusOK, workloads)
}

func (s *Server) getTechnicianWorkload(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	workload, err := s.db.GetTechnicianWorkload(id)
	if err != nil {
		http.Error(w, "Technician not found", http.StatusNotFound)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateTechnicianSkills(id, req.Skills, req.Certifications); err != nil {
		s.logger.Error("Failed to update technician skills", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	configs, err := s.db.GetAllSLAConfigs()
	if err != nil {
		s.logger.Error("Failed to get SLA configs", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResponseTimeMinutes <= 0 || req.ResolutionTimeMinutes <= 0 {
		http.Error(w, "Times must be positive", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateSLAConfig(priority, req.ResponseTimeMinutes, req.ResolutionTimeMinutes); err != nil {
		s.logger.Error("Failed to update SLA config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "update_sla_config", "sla_config", priority, nil, req)
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) getMaintenanceReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.db.GetMaintenanceReport()
	if err != nil {
		s.logger.Error("Failed to get maintenance report", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if report == nil {
		report = []models.MaintenanceReport{}
	}
	respondJSON(w, http.StatusOK, report)
}

func (s *Server) getSLAComplianceReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.db.GetSLAComplianceReport()
	if err != nil {
		s.logger.Error("Failed to get SLA compliance report", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	assignments, err := s.db.GetTechnicianSiteAssignments(filters)
	if err != nil {
		s.logger.Error("Failed to get technician site assignments", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if assignment.TechnicianID == "" {
		http.Error(w, "technician_id is required", http.StatusBadRequest)
		return
	}
	if assignment.SiteID == "" {
		http.Error(w, "site_id is required", http.StatusBadRequest)
		return
	}

	// Get assigned_by from context
	claims := auth.GetClaims(r)
	if claims != nil {
		assignment.AssignedBy = claims.UserID
	}

	if err := s.db.CreateTechnicianSiteAssignment(&assignment); err != nil {
		s.logger.Error("Failed to create technician site assignment", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Only allow updating is_primary field
	allowedUpdates := map[string]interface{}{}
	if isPrimary, ok := updates["is_primary"]; ok {
		allowedUpdates["is_primary"] = isPrimary
	}

	if len(allowedUpdates) == 0 {
		http.Error(w, "No valid fields to update", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateTechnicianSiteAssignment(id, allowedUpdates); err != nil {
		s.logger.Error("Failed to update technician site assignment", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteTechnicianSiteAssignment(id); err != nil {
		s.logger.Error("Failed to delete technician site assignment", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func (s *Server) logAudit(userID, action, entityType, entityID string, oldValue, newValue interface{}) {
	var oldJSON, newJSON []byte
	if oldValue != nil {
		oldJSON, _ = json.Marshal(oldValue)
	}
	if newValue != nil {
		newJSON, _ = json.Marshal(newValue)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO audit_log (user_id, action, entity_type, entity_id, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb)
	`, userID, action, entityType, entityID, oldJSON, newJSON)
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
