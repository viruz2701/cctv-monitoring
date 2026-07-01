// Package api — TO Journal API handlers (UX-3.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// UX-3.2: Auto-fill TO Journals при закрытии WorkOrder
//
// Feature Flag: to_auto_generation (default: false)
//
// Endpoints:
//
//	POST /api/v1/work-orders/{id}/to-journal/auto-fill
//	  — Создать авто-заполненные записи TO-журнала при закрытии WO
//	GET  /api/v1/work-orders/{id}/to-journal
//	  — Список TO-журналов для WorkOrder
//	PUT  /api/v1/to-journal/{id}
//	  — Обновить required поля TO-журнала (manual input)
//	GET  /api/v1/work-orders/{id}/to-journal/check
//	  — Проверка regulatory checklist перед закрытием
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — все endpoints требуют JWT)
//   - OWASP ASVS L3 V1-V17 (полный спектр контролей)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - OWASP ASVS V7.1 (Error handling — no stack traces)
//   - ISO 27001 A.12.4 (Audit trail)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/compliance"
)

// ═══════════════════════════════════════════════════════════════════════════
// handleAutoFillTOJournal — POST /api/v1/work-orders/{id}/to-journal/auto-fill
// ═══════════════════════════════════════════════════════════════════════════

// handleAutoFillTOJournal создаёт авто-заполненные записи в TO-журнале
// при закрытии Work Order (статус "completed").
//
// Pre-fill: device, date, technician, location, time (из work_order)
// Required fields (manual): checklist_notes, defects, customer_signature
//
// Feature Flag: to_auto_generation — если disabled → 503.
func (s *Server) handleAutoFillTOJournal(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	if workOrderID == "" {
		RespondError(w, r, NewValidationError("work order id is required"))
		return
	}

	// Проверяем feature flag (fail-secure)
	if !s.featureFlags.IsEnabled("to_auto_generation") {
		RespondError(w, r, &APIError{
			Status:  http.StatusServiceUnavailable,
			Code:    "FEATURE_DISABLED",
			Message: "Feature 'to_auto_generation' is disabled",
		})
		return
	}

	var req struct {
		DeviceID       string     `json:"device_id"`
		TechnicianID   string     `json:"technician_id,omitempty"`
		TechnicianName string     `json:"technician_name,omitempty"`
		SiteName       string     `json:"site_name,omitempty"`
		StartedAt      *time.Time `json:"started_at,omitempty"`
		CompletedAt    time.Time  `json:"completed_at"`
		DurationMin    int        `json:"duration_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	if req.DeviceID == "" {
		RespondError(w, r, NewValidationError("device_id is required"))
		return
	}

	if req.CompletedAt.IsZero() {
		req.CompletedAt = time.Now().UTC()
	}

	createReq := &compliance.TOJournalCreateRequest{
		WorkOrderID:    workOrderID,
		DeviceID:       req.DeviceID,
		TechnicianID:   req.TechnicianID,
		TechnicianName: req.TechnicianName,
		SiteName:       req.SiteName,
		StartedAt:      req.StartedAt,
		CompletedAt:    req.CompletedAt,
		DurationMin:    req.DurationMin,
	}

	entries, err := s.toJournalService.CreateAutoFilledEntries(r.Context(), createReq)
	if err != nil {
		s.logger.Error("Failed to auto-fill TO journal",
			"work_order_id", workOrderID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("failed to auto-fill TO journal", err))
		return
	}

	userID := getRequestUserID(r)
	s.logAudit(userID, "auto_fill_to_journal", "work_order", workOrderID, nil, map[string]interface{}{
		"entries_count": len(entries),
		"device_id":     req.DeviceID,
	})

	jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"entries": entries,
		"total":   len(entries),
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// handleListTOJournal — GET /api/v1/work-orders/{id}/to-journal
// ═══════════════════════════════════════════════════════════════════════════

// handleListTOJournal возвращает список записей TO-журнала для WorkOrder.
func (s *Server) handleListTOJournal(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	if workOrderID == "" {
		RespondError(w, r, NewValidationError("work order id is required"))
		return
	}

	summary, err := s.toJournalService.GetEntriesByWorkOrder(r.Context(), workOrderID)
	if err != nil {
		s.logger.Error("Failed to list TO journal entries",
			"work_order_id", workOrderID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("failed to list TO journal entries", err))
		return
	}

	jsonResponse(w, http.StatusOK, summary)
}

// ═══════════════════════════════════════════════════════════════════════════
// handleUpdateTOJournal — PUT /api/v1/to-journal/{id}
// ═══════════════════════════════════════════════════════════════════════════

// handleUpdateTOJournal обновляет required поля TO-журнала (manual input).
// После заполнения всех required полей запись считается is_completed = true.
func (s *Server) handleUpdateTOJournal(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")
	if entryID == "" {
		RespondError(w, r, NewValidationError("entry id is required"))
		return
	}

	var req compliance.TOJournalUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("Invalid request body"))
		return
	}

	entry, err := s.toJournalService.UpdateRequiredFields(r.Context(), entryID, &req)
	if err != nil {
		s.logger.Error("Failed to update TO journal entry",
			"entry_id", entryID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("failed to update TO journal entry", err))
		return
	}

	userID := getRequestUserID(r)
	s.logAudit(userID, "update_to_journal", "to_journal", entryID, nil, map[string]interface{}{
		"is_completed": entry.IsCompleted,
	})

	jsonResponse(w, http.StatusOK, entry)
}

// ═══════════════════════════════════════════════════════════════════════════
// handleCheckTOJournal — GET /api/v1/work-orders/{id}/to-journal/check
// ═══════════════════════════════════════════════════════════════════════════

// handleCheckTOJournal проверяет regulatory checklist перед закрытием WO.
// Возвращает статус: все ли required поля заполнены.
func (s *Server) handleCheckTOJournal(w http.ResponseWriter, r *http.Request) {
	workOrderID := chi.URLParam(r, "id")
	if workOrderID == "" {
		RespondError(w, r, NewValidationError("work order id is required"))
		return
	}

	result, err := s.toJournalService.CheckRegulatoryCompliance(r.Context(), workOrderID)
	if err != nil {
		s.logger.Error("Failed to check TO journal compliance",
			"work_order_id", workOrderID,
			"error", err,
		)
		RespondError(w, r, NewInternalError("failed to check TO journal compliance", err))
		return
	}

	jsonResponse(w, http.StatusOK, result)
}
