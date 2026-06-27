// Package api — Work Request handlers (WO-4.1.1).
//
// Public submit endpoint (без авторизации, с reCAPTCHA) + protected
// approval workflow (list, approve, reject, convert).
//
// Compliance:
//   - OWASP ASVS V1.1 (Input validation)
//   - OWASP ASVS V3.1 (Session management — reCAPTCHA)
//   - OWASP ASVS V5.3 (Input validation — structured data)
//   - ISO 27001 A.9.2.1 (User registration — external request)
//   - ISO 27001 A.14.2.1 (Service delivery — request portal)
//   - IEC 62443 SR 2.1 (Account management — request workflow)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/models"
)

// ── Constants ──────────────────────────────────────────────────────

const (
	// MaxWorkRequestTitle — максимальная длина заголовка заявки.
	MaxWorkRequestTitle = 500
	// MaxWorkRequestDesc — максимальная длина описания.
	MaxWorkRequestDesc = 5000
	// MaxWorkRequestName — максимальная длина имени заявителя.
	MaxWorkRequestName = 200
	// MaxWorkRequestEmail — максимальная длина email.
	MaxWorkRequestEmail = 200
	// MaxWorkRequestPhone — максимальная длина телефона.
	MaxWorkRequestPhone = 50
)

// ═══════════════════════════════════════════════════════════════════════
// Public endpoint — без авторизации
// ═══════════════════════════════════════════════════════════════════════

// submitWorkRequest — публичный endpoint для подачи заявки (без JWT).
//
// POST /api/v1/public/work-requests
//
// Body:
//
//	{
//	  "title": "Camera offline",
//	  "description": "Camera 101 is offline since 2 hours",
//	  "device_id": "uuid",
//	  "site_id": "uuid",
//	  "priority": "high",
//	  "type": "corrective",
//	  "requester_name": "John Doe",
//	  "requester_email": "john@example.com",
//	  "requester_phone": "+1234567890",
//	  "captcha_token": "reCAPTCHA token"
//	}
//
// Rate limiting: 10 req/min/IP (настраивается через middleware).
func (s *Server) submitWorkRequest(w http.ResponseWriter, r *http.Request) {
	var req models.WorkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	// ── Валидация reCAPTCHA ─────────────────────────────────────
	if err := s.recaptchaValidator.Verify(r.Context(), req.CaptchaToken); err != nil {
		s.logger.Warn("recaptcha verification failed", "error", err)
		RespondError(w, r, NewBadRequestError("CAPTCHA verification failed"))
		return
	}

	// ── Валидация полей (OWASP ASVS V5.1 — whitelist validation) ─

	// Title
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		RespondError(w, r, NewBadRequestError("title is required"))
		return
	}
	if len(req.Title) > MaxWorkRequestTitle {
		req.Title = req.Title[:MaxWorkRequestTitle]
	}

	// Description
	req.Description = strings.TrimSpace(req.Description)
	if len(req.Description) > MaxWorkRequestDesc {
		req.Description = req.Description[:MaxWorkRequestDesc]
	}

	// Requester name
	req.RequesterName = strings.TrimSpace(req.RequesterName)
	if req.RequesterName == "" {
		RespondError(w, r, NewBadRequestError("requester_name is required"))
		return
	}
	if len(req.RequesterName) > MaxWorkRequestName {
		req.RequesterName = req.RequesterName[:MaxWorkRequestName]
	}

	// Requester email
	req.RequesterEmail = strings.TrimSpace(req.RequesterEmail)
	if req.RequesterEmail == "" {
		RespondError(w, r, NewBadRequestError("requester_email is required"))
		return
	}
	if len(req.RequesterEmail) > MaxWorkRequestEmail {
		req.RequesterEmail = req.RequesterEmail[:MaxWorkRequestEmail]
	}
	if !isValidEmail(req.RequesterEmail) {
		RespondError(w, r, NewBadRequestError("invalid requester_email format"))
		return
	}

	// Requester phone (опционально)
	req.RequesterPhone = strings.TrimSpace(req.RequesterPhone)
	if len(req.RequesterPhone) > MaxWorkRequestPhone {
		req.RequesterPhone = req.RequesterPhone[:MaxWorkRequestPhone]
	}

	// Priority
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if !models.ValidWorkRequestPriority(req.Priority) {
		RespondError(w, r, NewBadRequestError(
			fmt.Sprintf("invalid priority: %s (must be: critical, high, medium, low)", req.Priority),
		))
		return
	}

	// Type
	if req.Type == "" {
		req.Type = "corrective"
	}
	if !models.ValidWorkRequestType(req.Type) {
		RespondError(w, r, NewBadRequestError(
			fmt.Sprintf("invalid type: %s (must be: corrective, preventive, emergency, routine, inspection)", req.Type),
		))
		return
	}

	// ── Метаданные ──────────────────────────────────────────────
	req.Status = models.WorkRequestSubmitted
	req.SourceIP = r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		req.SourceIP = strings.Split(forwarded, ",")[0]
	}
	req.UserAgent = r.UserAgent()

	// ── Сохранение ──────────────────────────────────────────────
	if err := s.cmmsRouter.CreateWorkRequest(r.Context(), &req); err != nil {
		s.logger.Error("Failed to create work request", "error", err)
		RespondError(w, r, NewInternalError("Failed to submit request", err))
		return
	}

	// Audit log
	s.logAudit("public", "create_work_request", "work_request", req.ID, nil, map[string]string{
		"title":          req.Title,
		"requester_name": req.RequesterName,
		"requester_email": req.RequesterEmail,
		"priority":       req.Priority,
		"type":           req.Type,
	})

	jsonResponse(w, http.StatusCreated, req)
}

// ═══════════════════════════════════════════════════════════════════════
// Protected endpoints — требуют JWT
// ═══════════════════════════════════════════════════════════════════════

// listWorkRequests — список заявок (с фильтрацией).
func (s *Server) listWorkRequests(w http.ResponseWriter, r *http.Request) {
	filters := make(map[string]interface{})

	if status := r.URL.Query().Get("status"); status != "" {
		if !models.ValidWorkRequestStatus(status) {
			RespondError(w, r, NewBadRequestError("invalid status"))
			return
		}
		filters["status"] = status
	}
	if deviceID := r.URL.Query().Get("device_id"); deviceID != "" {
		filters["device_id"] = deviceID
	}
	if email := r.URL.Query().Get("requester_email"); email != "" {
		filters["requester_email"] = email
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filters["limit"] = l
		}
	}
	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters["offset"] = o
		}
	}

	requests, err := s.cmmsRouter.GetWorkRequests(r.Context(), filters)
	if err != nil {
		s.logger.Error("Failed to get work requests", "error", err)
		RespondError(w, r, NewInternalError("operation failed", err))
		return
	}

	if requests == nil {
		requests = []models.WorkRequest{}
	}
	jsonResponse(w, http.StatusOK, requests)
}

// getWorkRequest — детали заявки.
func (s *Server) getWorkRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	req, err := s.cmmsRouter.GetWorkRequest(r.Context(), id)
	if err != nil {
		RespondError(w, r, NewNotFoundError("Work request not found"))
		return
	}
	jsonResponse(w, http.StatusOK, req)
}

// approveWorkRequest — одобрение заявки.
func (s *Server) approveWorkRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := getUserIDFromContext(r.Context())

	if err := s.cmmsRouter.ApproveWorkRequest(r.Context(), id, userID); err != nil {
		s.logger.Error("Failed to approve work request", "error", err)
		RespondError(w, r, NewBadRequestError(err.Error()))
		return
	}

	s.logAudit(userID, "approve_work_request", "work_request", id, nil, nil)
	jsonResponse(w, http.StatusOK, map[string]string{"status": "approved"})
}

// rejectWorkRequest — отклонение заявки.
func (s *Server) rejectWorkRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.Reason = "No reason provided"
	}

	userID := getUserIDFromContext(r.Context())
	if err := s.cmmsRouter.RejectWorkRequest(r.Context(), id, userID, body.Reason); err != nil {
		s.logger.Error("Failed to reject work request", "error", err)
		RespondError(w, r, NewBadRequestError(err.Error()))
		return
	}

	s.logAudit(userID, "reject_work_request", "work_request", id, nil, map[string]string{"reason": body.Reason})
	jsonResponse(w, http.StatusOK, map[string]string{"status": "rejected"})
}

// convertWorkRequestToWO — конвертация одобренной заявки в WorkOrder.
func (s *Server) convertWorkRequestToWO(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Получаем заявку
	req, err := s.cmmsRouter.GetWorkRequest(r.Context(), id)
	if err != nil || req == nil {
		RespondError(w, r, NewNotFoundError("Work request not found"))
		return
	}

	if req.Status != models.WorkRequestApproved {
		RespondError(w, r, NewBadRequestError("Work request must be approved before conversion"))
		return
	}

	// Создаём WorkOrder из заявки
	userID := getUserIDFromContext(r.Context())
	wo := &models.WorkOrder{
		Title:     req.Title,
		DeviceID:  req.DeviceID,
		Type:      req.Type,
		Priority:  req.Priority,
		Status:    "open",
		Notes:     fmt.Sprintf("Converted from work request %s\n\n%s\n\nRequester: %s (%s, %s)", req.ID, req.Description, req.RequesterName, req.RequesterEmail, req.RequesterPhone),
		CreatedBy: &userID,
	}

	if err := s.cmmsRouter.CreateWorkOrder(r.Context(), wo); err != nil {
		s.logger.Error("Failed to create work order from request", "error", err)
		RespondError(w, r, NewInternalError("Failed to create work order", err))
		return
	}

	// Обновляем заявку — связываем с WorkOrder
	if err := s.cmmsRouter.ConvertWorkRequestToWO(r.Context(), id, wo.ID); err != nil {
		s.logger.Error("Failed to convert work request", "error", err)
		RespondError(w, r, NewInternalError("Failed to convert request", err))
		return
	}

	s.logAudit(userID, "convert_work_request", "work_request", id, nil, map[string]string{
		"work_order_id": wo.ID,
	})

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":         "converted",
		"work_order_id":  wo.ID,
		"work_order":     wo,
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// isValidEmail — простая валидация email (OWASP ASVS V5.1 — whitelist format).
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || at >= len(email)-4 {
		return false
	}
	local := email[:at]
	domain := email[at+1:]

	if len(local) == 0 || len(domain) < 3 {
		return false
	}
	if !strings.Contains(domain, ".") {
		return false
	}
	// Проверяем что нет запрещённых символов
	for _, c := range email {
		if c > 127 {
			return false // только ASCII
		}
	}
	return true
}

// ── Rate limiter для public endpoint ───────────────────────────────

// workRequestRateLimiter — лимит 10 запросов в минуту на IP для public endpoint.
func (s *Server) workRequestRateLimiter(next http.Handler) http.Handler {
	return s.newRateLimiterMiddleware(10, time.Minute)(next)
}
