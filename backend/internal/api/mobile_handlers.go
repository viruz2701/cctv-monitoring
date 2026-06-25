package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/gatekeeper"
	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Mobile-Specific API Handlers
// Оптимизированные для мобильных клиентов (компактные ответы, batch-операции)
// ═══════════════════════════════════════════════════════════════════════

// ---------- Mobile Work Orders ----------

// listMobileWorkOrders возвращает компактный список нарядов для текущего техника
func (s *Server) listMobileWorkOrders(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	filters := map[string]interface{}{
		"assigned_to": claims.UserID,
		"limit":       50,
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filters["status"] = status
	}

	workOrders, err := s.cmmsRouter.GetWorkOrders(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get mobile work orders", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if workOrders == nil {
		workOrders = []models.WorkOrder{}
	}

	// Компактный ответ для мобильного — убираем лишние поля
	type mobileWorkOrder struct {
		ID          string          `json:"id"`
		DeviceID    string          `json:"device_id"`
		DeviceName  string          `json:"device_name"`
		Type        string          `json:"type"`
		Status      string          `json:"status"`
		Priority    string          `json:"priority"`
		SLADeadline *string         `json:"sla_deadline,omitempty"`
		Checklist   json.RawMessage `json:"checklist"`
		StartedAt   *string         `json:"started_at,omitempty"`
		CompletedAt *string         `json:"completed_at,omitempty"`
		Notes       string          `json:"notes,omitempty"`
		PhotosCount int             `json:"photos_count"`
		CreatedAt   string          `json:"created_at"`
		SiteName    string          `json:"site_name,omitempty"`
	}

	result := make([]mobileWorkOrder, 0, len(workOrders))
	for _, wo := range workOrders {
		var slaDeadline *string
		if wo.SLADeadline != nil {
			s := wo.SLADeadline.Format(time.RFC3339)
			slaDeadline = &s
		}
		var startedAt *string
		if wo.StartedAt != nil {
			s := wo.StartedAt.Format(time.RFC3339)
			startedAt = &s
		}
		var completedAt *string
		if wo.CompletedAt != nil {
			s := wo.CompletedAt.Format(time.RFC3339)
			completedAt = &s
		}

		photosCount := 0
		if wo.Photos != nil {
			var photos []string
			if err := json.Unmarshal(wo.Photos, &photos); err == nil {
				photosCount = len(photos)
			}
		}

		result = append(result, mobileWorkOrder{
			ID:          wo.ID,
			DeviceID:    wo.DeviceID,
			DeviceName:  wo.DeviceName,
			Type:        wo.Type,
			Status:      wo.Status,
			Priority:    wo.Priority,
			SLADeadline: slaDeadline,
			Checklist:   wo.Checklist,
			StartedAt:   startedAt,
			CompletedAt: completedAt,
			Notes:       wo.Notes,
			PhotosCount: photosCount,
			CreatedAt:   wo.CreatedAt.Format(time.RFC3339),
			SiteName:    getSiteName(wo),
		})
	}

	jsonResponse(w, http.StatusOK, result)
}

// getMobileWorkOrder возвращает детали наряда для мобильного
func (s *Server) getMobileWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wo, err := s.cmmsRouter.GetWorkOrder(r.Context(), id)
	if err != nil {
		respondError(w, r, NewNotFoundError("Work order not found"))
		return
	}

	// Декодируем photos и parts_used для удобства мобильного клиента
	var photos []string
	if wo.Photos != nil {
		json.Unmarshal(wo.Photos, &photos)
	}
	if photos == nil {
		photos = []string{}
	}

	var partsUsed []models.PartUsage
	if wo.PartsUsed != nil {
		json.Unmarshal(wo.PartsUsed, &partsUsed)
	}
	if partsUsed == nil {
		partsUsed = []models.PartUsage{}
	}

	var checklist []models.ChecklistItem
	if wo.Checklist != nil {
		json.Unmarshal(wo.Checklist, &checklist)
	}
	if checklist == nil {
		checklist = []models.ChecklistItem{}
	}

	type mobileWorkOrderDetail struct {
		ID           string                 `json:"id"`
		ScheduleID   *string                `json:"schedule_id,omitempty"`
		DeviceID     string                 `json:"device_id"`
		DeviceName   string                 `json:"device_name"`
		SiteName     string                 `json:"site_name,omitempty"`
		Type         string                 `json:"type"`
		Status       string                 `json:"status"`
		Priority     string                 `json:"priority"`
		AssignedTo   *string                `json:"assigned_to,omitempty"`
		SLADeadline  *string                `json:"sla_deadline,omitempty"`
		Checklist    []models.ChecklistItem `json:"checklist"`
		StartedAt    *string                `json:"started_at,omitempty"`
		CompletedAt  *string                `json:"completed_at,omitempty"`
		Notes        string                 `json:"notes,omitempty"`
		Photos       []string               `json:"photos"`
		PartsUsed    []models.PartUsage     `json:"parts_used"`
		CreatedAt    string                 `json:"created_at"`
		UpdatedAt    string                 `json:"updated_at"`
		AssigneeName string                 `json:"assignee_name,omitempty"`
		SLAStatus    string                 `json:"sla_status,omitempty"`
	}

	var slaDeadline *string
	if wo.SLADeadline != nil {
		s := wo.SLADeadline.Format(time.RFC3339)
		slaDeadline = &s
	}
	var startedAt *string
	if wo.StartedAt != nil {
		s := wo.StartedAt.Format(time.RFC3339)
		startedAt = &s
	}
	var completedAt *string
	if wo.CompletedAt != nil {
		s := wo.CompletedAt.Format(time.RFC3339)
		completedAt = &s
	}

	jsonResponse(w, http.StatusOK, mobileWorkOrderDetail{
		ID:           wo.ID,
		ScheduleID:   wo.ScheduleID,
		DeviceID:     wo.DeviceID,
		DeviceName:   wo.DeviceName,
		SiteName:     getSiteName(*wo),
		Type:         wo.Type,
		Status:       wo.Status,
		Priority:     wo.Priority,
		AssignedTo:   wo.AssignedTo,
		SLADeadline:  slaDeadline,
		Checklist:    checklist,
		StartedAt:    startedAt,
		CompletedAt:  completedAt,
		Notes:        wo.Notes,
		Photos:       photos,
		PartsUsed:    partsUsed,
		CreatedAt:    wo.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    wo.UpdatedAt.Format(time.RFC3339),
		AssigneeName: wo.AssigneeName,
		SLAStatus:    wo.SLAStatus,
	})
}

// startMobileWorkOrder — начать выполнение наряда
func (s *Server) startMobileWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.cmmsRouter.StartWorkOrder(r.Context(), id); err != nil {
		s.logger.Error("Failed to start mobile work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	userID := getUserIDFromContext(r.Context())
	s.logAudit(userID, "mobile_start_work_order", "work_order", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "started"})
}

// completeMobileWorkOrder — завершить наряд с расширенным payload.
// Требует verification_token, полученный через POST /verify.
func (s *Server) completeMobileWorkOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req struct {
		Notes             string                 `json:"notes"`
		Checklist         []models.ChecklistItem `json:"checklist"`
		Photos            []string               `json:"photos"`
		PartsUsed         []models.PartUsage     `json:"parts_used"`
		Signature         *string                `json:"signature"`
		VerificationToken string                 `json:"verification_token"`
		Location          *struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// Gatekeeper: проверяем verification token
	if req.VerificationToken == "" {
		respondError(w, r, NewBadRequestError("verification_token is required. Call POST /verify first."))
		return
	}

	vClaims, err := gatekeeper.ValidateVerificationToken(req.VerificationToken)
	if err != nil {
		respondError(w, r, NewUnauthorizedError("invalid or expired verification_token"))
		return
	}

	if vClaims.WorkOrderID != id {
		respondError(w, r, NewBadRequestError("verification_token does not match this work order"))
		return
	}

	// Сохраняем чек-лист в work order
	checklistJSON, _ := json.Marshal(req.Checklist)
	if err := s.cmmsRouter.UpdateWorkOrder(r.Context(), id, map[string]interface{}{"checklist": checklistJSON}); err != nil {
		s.logger.Warn("Failed to update checklist", "error", err)
	}

	// Сохраняем signature как заметку (или в отдельное поле, если нужно)
	notes := req.Notes
	if req.Signature != nil && *req.Signature != "" && *req.Signature != "skipped" {
		// Сохраняем signature base64 в notes с префиксом
		if notes != "" {
			notes += "\n\n[SIGNATURE:" + (*req.Signature)[:50] + "...]"
		} else {
			notes = "[SIGNATURE:" + (*req.Signature)[:50] + "...]"
		}
	}

	// Добавляем информацию о верификации в notes
	notes += "\n\n[VERIFIED: GPS=" + boolToStr(vClaims.GPSPassed) +
		" EXIF=" + boolToStr(vClaims.EXIFPassed) +
		" AI=" + boolToStr(vClaims.AIPassed) +
		" GPS_SKIPPED=" + boolToStr(vClaims.GPSSkipped) + "]"

	// Сохраняем location
	if req.Location != nil {
		locJSON, _ := json.Marshal(req.Location)
		_ = s.cmmsRouter.UpdateWorkOrder(r.Context(), id, map[string]interface{}{"notes": notes, "location": locJSON})
	}

	userID := getUserIDFromContext(r.Context())
	if err := s.cmmsRouter.CompleteWorkOrder(r.Context(), id, notes, req.Photos, req.PartsUsed, userID); err != nil {
		s.logger.Error("Failed to complete mobile work order", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	s.logAudit(userID, "mobile_complete_work_order", "work_order", id, nil, req)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "completed"})
}

// boolToStr converts bool to "true"/"false" string.
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// uploadMobileWorkOrderPhoto — загрузка одного фото через multipart
func (s *Server) uploadMobileWorkOrderPhoto(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondError(w, r, NewBadRequestError("Failed to parse form"))
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		respondError(w, r, NewBadRequestError("Failed to get file"))
		return
	}
	defer file.Close()

	filename := generateFilename(header.Filename)
	dst := s.imagesDir + "/" + filename

	if err := saveUploadedFile(file, dst); err != nil {
		s.logger.Error("Failed to save mobile photo", "error", err)
		respondError(w, r, NewInternalError("Failed to save file", nil))
		return
	}

	photoURL := "/api/v1/images/" + filename

	// Обновляем work order photos
	wo, err := s.cmmsRouter.GetWorkOrder(r.Context(), workOrderID)
	if err != nil {
		respondError(w, r, NewNotFoundError("Work order not found"))
		return
	}

	var existingPhotos []string
	if wo.Photos != nil {
		json.Unmarshal(wo.Photos, &existingPhotos)
	}
	existingPhotos = append(existingPhotos, photoURL)

	photosJSON, _ := json.Marshal(existingPhotos)
	if err := s.cmmsRouter.UpdateWorkOrder(r.Context(), workOrderID, map[string]interface{}{"photos": photosJSON}); err != nil {
		s.logger.Error("Failed to update work order photos", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"url":      photoURL,
		"filename": filename,
	})
}

// ---------- Mobile Push Token ----------

// registerMobilePushToken — регистрация push-токена для техника
func (s *Server) registerMobilePushToken(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if req.Token == "" {
		respondError(w, r, NewBadRequestError("token is required"))
		return
	}

	if req.Platform == "" {
		req.Platform = "unknown"
	}

	// Сохраняем push-токен в БД
	if err := s.cmmsRouter.SavePushToken(r.Context(), claims.UserID, req.Token, req.Platform); err != nil {
		s.logger.Error("Failed to save push token", "error", err)
		respondError(w, r, NewInternalError("operation failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "registered"})
}

// ---------- Mobile Devices (Offline Map: UX-02) ----------

// MobileDeviceMapData — лёгкая структура для карты устройств
// Содержит только поля, необходимые для отображения на карте (OWASP ASVS V8 — Data Protection)
type MobileDeviceMapData struct {
	DeviceID   string  `json:"device_id"`
	Name       string  `json:"name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Status     string  `json:"status"`
	DeviceType string  `json:"device_type"`
	SiteName   string  `json:"site_name,omitempty"`
	Health     string  `json:"health"`
}

// listMobileDevices возвращает компактный список устройств с координатами для карты.
// GET /api/v1/mobile/devices
// Соответствует:
//   - OWASP ASVS V4 (RBAC — только свои устройства)
//   - OWASP ASVS V5 (Whitelist validation)
//   - OWASP ASVS V7 (Error handling — no information leakage)
//   - OWASP ASVS V8 (Data Protection — только необходимые поля)
//   - ISO 27001 A.12.6.1 (Capacity management — лимит страницы)
func (s *Server) listMobileDevices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// Фильтры (V5 — whitelist validation)
	statusFilter := r.URL.Query().Get("status")
	deviceTypeFilter := r.URL.Query().Get("device_type")
	siteID := r.URL.Query().Get("site_id")
	search := r.URL.Query().Get("search")

	// Валидация статуса
	if statusFilter != "" {
		valid := false
		for _, s := range validStatuses {
			if s == statusFilter {
				valid = true
				break
			}
		}
		if !valid {
			respondError(w, r, NewValidationError("invalid status: must be ONLINE, OFFLINE, or WARNING"))
			return
		}
	}

	// Валидация device_type
	if deviceTypeFilter != "" {
		valid := false
		for _, dt := range validDeviceTypes {
			if dt == deviceTypeFilter {
				valid = true
				break
			}
		}
		if !valid {
			respondError(w, r, NewValidationError("invalid device_type: must be camera, nvr, dvr, or switch"))
			return
		}
	}

	filter := models.ListDevicesFilter{
		Page:       1,
		PageSize:   500,
		Status:     statusFilter,
		DeviceType: deviceTypeFilter,
		SiteID:     siteID,
		Search:     search,
	}

	result, err := s.deviceService.ListDevices(r.Context(), claims.UserID, claims.Role, filter)
	if err != nil {
		respondError(w, r, NewInternalError("failed to list devices", err))
		return
	}

	// Маппинг в компактный формат для карты (V8 — только необходимые поля)
	devices := make([]MobileDeviceMapData, 0, len(result.Devices))
	for _, d := range result.Devices {
		fullDevice, err := s.deviceService.GetDevice(r.Context(), claims.UserID, claims.Role, d.DeviceID)
		if err != nil || fullDevice == nil {
			continue
		}
		// Пропускаем устройства без координат
		if fullDevice.Latitude == 0 && fullDevice.Longitude == 0 {
			continue
		}

		devices = append(devices, MobileDeviceMapData{
			DeviceID:   fullDevice.DeviceID,
			Name:       fullDevice.Name,
			Latitude:   fullDevice.Latitude,
			Longitude:  fullDevice.Longitude,
			Status:     string(fullDevice.Status),
			DeviceType: string(fullDevice.DeviceType),
			SiteName:   fullDevice.Location,
			Health:     string(fullDevice.Health),
		})
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"devices": devices,
		"total":   len(devices),
	})
}

// ---------- Mobile Technician Profile ----------

// getMobileTechnicianProfile возвращает профиль текущего техника
func (s *Server) getMobileTechnicianProfile(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	workload, err := s.cmmsRouter.GetTechnicianWorkload(r.Context(), claims.UserID)
	if err != nil {
		// Возвращаем базовый профиль если workload не найден
		user, err := s.db.GetUserByID(claims.UserID)
		if err != nil {
			respondError(w, r, NewNotFoundError("User not found"))
			return
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"user_id":          user.ID,
			"user_name":        user.Username,
			"current_workload": 0,
			"max_workload":     10,
			"skills":           []string{},
			"base_location":    "",
		})
		return
	}

	jsonResponse(w, http.StatusOK, workload)
}

// getMobileTechnicianStats возвращает статистику техника за месяц
func (s *Server) getMobileTechnicianStats(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("unauthorized"))
		return
	}

	// Получаем статистику из БД
	stats, err := s.cmmsRouter.GetTechnicianMonthlyStats(r.Context(), claims.UserID)
	if err != nil {
		s.logger.Error("Failed to get technician stats", "error", err)
		// Возвращаем нулевую статистику
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"completed_this_month": 0,
			"total_work_orders":    0,
			"on_time_percent":      0,
			"avg_rating":           0,
		})
		return
	}

	jsonResponse(w, http.StatusOK, stats)
}

// ---------- Helpers ----------

func getSiteName(wo models.WorkOrder) string {
	// Заглушка: в будущем будет запрашивать имя объекта из БД
	_ = wo
	return ""
}
