// Package api — 152-ФЗ Personal Data HTTP handlers (P2-RU.2).
//
// Маршруты защищены AuthMiddleware (вызывается из server.go).
//
// Compliance:
//   - 152-ФЗ ст. 9 (согласие), ст. 14 (доступ к ПД), ст. 21 (блокировка)
//   - OWASP ASVS V4 (RBAC — admin/manager/owner)
//   - ISO 27001 A.12.4 (Audit trail)
//   - СТБ 34.101.27 п. 6.2 (Политики безопасности)
package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/compliance"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V2 — Authentication (через JWT middleware)
// [x] V3 — Session Management (через AuthMiddleware)
// [x] V4 — Access Control (admin/manager/owner)
// [x] V5 — Input Validation (whitelist query params)
// [x] V7 — Error Handling and Logging (через respondError)
// [x] V8 — Data Protection (не раскрываем sensitive fields)

// ═══════════════════════════════════════════════════════════════════════
// Request/Response types
// ═══════════════════════════════════════════════════════════════════════

type grantConsentRequest struct {
	SubjectID     string `json:"subject_id"`
	SubjectName   string `json:"subject_name"`
	Purpose       string `json:"purpose"`
	Source        string `json:"source"`
	ExpiresInDays int    `json:"expires_in_days,omitempty"`
}

type revokeConsentRequest struct {
	ConsentID string `json:"consent_id"`
}

type submitDSARRequest struct {
	SubjectID    string `json:"subject_id"`
	SubjectName  string `json:"subject_name"`
	SubjectEmail string `json:"subject_email"`
	SubjectPhone string `json:"subject_phone,omitempty"`
	RequestType  string `json:"request_type"`
	Description  string `json:"description"`
}

type fulfillDSARRequest struct {
	DSARID       string `json:"dsar_id"`
	ResponseData string `json:"response_data"`
}

type rejectDSARRequest struct {
	DSARID string `json:"dsar_id"`
	Reason string `json:"reason"`
}

type registerInventoryItemRequest struct {
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	DataFields      []string `json:"data_fields"`
	StorageLocation string   `json:"storage_location"`
	Purpose         string   `json:"purpose"`
	RetentionDays   int      `json:"retention_days"`
	LegalBasis      string   `json:"legal_basis"`
}

type roskomnadzorReportRequest struct {
	OperatorName    string `json:"operator_name"`
	OperatorINN     string `json:"operator_inn"`
	OperatorAddress string `json:"operator_address"`
	SubjectCount    int    `json:"subject_count"`
}

// ═══════════════════════════════════════════════════════════════════════
// Consent Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleGrantConsent — POST /api/v1/compliance/personal-data/consent
//
// Access: admin, manager, owner
func (s *Server) handleGrantConsent(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	var req grantConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.SubjectID == "" || req.Purpose == "" {
		RespondError(w, r, NewValidationError("subject_id and purpose are required"))
		return
	}

	record, err := s.personalDataManager.GrantConsent(
		req.SubjectID, req.SubjectName,
		compliance.ConsentPurpose(req.Purpose),
		req.Source, req.ExpiresInDays,
	)
	if err != nil {
		s.logger.Error("failed to grant consent", "error", err)
		RespondError(w, r, NewInternalError("failed to grant consent", err))
		return
	}

	jsonResponse(w, http.StatusCreated, record)

	s.logger.Info("consent granted",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"consent_id", record.ID,
	)
}

// handleRevokeConsent — POST /api/v1/compliance/personal-data/consent/revoke
//
// Access: admin, manager, owner
func (s *Server) handleRevokeConsent(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	var req revokeConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.ConsentID == "" {
		RespondError(w, r, NewValidationError("consent_id is required"))
		return
	}

	if err := s.personalDataManager.RevokeConsent(req.ConsentID); err != nil {
		s.logger.Error("failed to revoke consent", "error", err)
		RespondError(w, r, NewInternalError("failed to revoke consent", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "consent_id": req.ConsentID})

	s.logger.Info("consent revoked",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"consent_id", req.ConsentID,
	)
}

// handleListConsents — GET /api/v1/compliance/personal-data/consent
//
// Query params: subject_id (optional)
// Access: admin, manager, owner
func (s *Server) handleListConsents(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	subjectID := r.URL.Query().Get("subject_id")

	var consents interface{}
	var err error
	if subjectID != "" {
		consents, err = s.personalDataManager.ListSubjectConsents(subjectID)
	} else {
		// Если нет subject_id — не возвращаем все конфиденциально
		RespondError(w, r, NewValidationError("subject_id query parameter is required"))
		return
	}
	if err != nil {
		s.logger.Error("failed to list consents", "error", err)
		RespondError(w, r, NewInternalError("failed to list consents", err))
		return
	}

	jsonResponse(w, http.StatusOK, consents)
}

// ═══════════════════════════════════════════════════════════════════════
// DSAR Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleSubmitDSAR — POST /api/v1/compliance/personal-data/dsar
//
// Access: admin, manager, owner
func (s *Server) handleSubmitDSAR(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	var req submitDSARRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.SubjectID == "" || req.RequestType == "" {
		RespondError(w, r, NewValidationError("subject_id and request_type are required"))
		return
	}

	dsar, err := s.personalDataManager.SubmitDSAR(
		req.SubjectID, req.SubjectName, req.SubjectEmail,
		req.SubjectPhone, req.RequestType, req.Description,
	)
	if err != nil {
		s.logger.Error("failed to submit DSAR", "error", err)
		RespondError(w, r, NewInternalError("failed to submit DSAR", err))
		return
	}

	jsonResponse(w, http.StatusCreated, dsar)

	s.logger.Info("DSAR submitted",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"dsar_id", dsar.ID,
	)
}

// handleFulfillDSAR — POST /api/v1/compliance/personal-data/dsar/fulfill
//
// Access: admin only
func (s *Server) handleFulfillDSAR(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req fulfillDSARRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.DSARID == "" {
		RespondError(w, r, NewValidationError("dsar_id is required"))
		return
	}

	if err := s.personalDataManager.FulfillDSAR(req.DSARID, req.ResponseData); err != nil {
		s.logger.Error("failed to fulfill DSAR", "error", err)
		RespondError(w, r, NewInternalError("failed to fulfill DSAR", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "dsar_id": req.DSARID})

	s.logger.Info("DSAR fulfilled",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"dsar_id", req.DSARID,
	)
}

// handleRejectDSAR — POST /api/v1/compliance/personal-data/dsar/reject
//
// Access: admin only
func (s *Server) handleRejectDSAR(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req rejectDSARRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.DSARID == "" || req.Reason == "" {
		RespondError(w, r, NewValidationError("dsar_id and reason are required"))
		return
	}

	if err := s.personalDataManager.RejectDSAR(req.DSARID, req.Reason); err != nil {
		s.logger.Error("failed to reject DSAR", "error", err)
		RespondError(w, r, NewInternalError("failed to reject DSAR", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "dsar_id": req.DSARID})

	s.logger.Info("DSAR rejected",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"dsar_id", req.DSARID,
	)
}

// handleListDSARs — GET /api/v1/compliance/personal-data/dsar
//
// Query params: subject_id (optional)
// Access: admin, manager, owner
func (s *Server) handleListDSARs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	subjectID := r.URL.Query().Get("subject_id")
	if subjectID == "" {
		RespondError(w, r, NewValidationError("subject_id query parameter is required"))
		return
	}

	dsars, err := s.personalDataManager.ListSubjectDSARs(subjectID)
	if err != nil {
		s.logger.Error("failed to list DSARs", "error", err)
		RespondError(w, r, NewInternalError("failed to list DSARs", err))
		return
	}

	jsonResponse(w, http.StatusOK, dsars)
}

// ═══════════════════════════════════════════════════════════════════════
// Data Inventory Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleRegisterInventoryItem — POST /api/v1/compliance/personal-data/inventory
//
// Access: admin only
func (s *Server) handleRegisterInventoryItem(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req registerInventoryItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.Category == "" || len(req.DataFields) == 0 {
		RespondError(w, r, NewValidationError("category and data_fields are required"))
		return
	}

	item, err := s.personalDataManager.RegisterInventoryItem(
		compliance.DataCategory(req.Category), req.Description, req.DataFields,
		req.StorageLocation, compliance.ConsentPurpose(req.Purpose),
		req.RetentionDays, req.LegalBasis,
	)
	if err != nil {
		s.logger.Error("failed to register inventory item", "error", err)
		RespondError(w, r, NewInternalError("failed to register inventory item", err))
		return
	}

	jsonResponse(w, http.StatusCreated, item)

	s.logger.Info("inventory item registered",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"item_id", item.ID,
	)
}

// handleGetInventory — GET /api/v1/compliance/personal-data/inventory
//
// Access: admin, manager, owner
func (s *Server) handleGetInventory(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	anonymizeParam := r.URL.Query().Get("anonymize")

	items, err := s.personalDataManager.GetInventory()
	if err != nil {
		s.logger.Error("failed to get inventory", "error", err)
		RespondError(w, r, NewInternalError("failed to get inventory", err))
		return
	}

	if anonymizeParam == "true" {
		items = s.personalDataManager.AnonymizeData(items)
	}

	jsonResponse(w, http.StatusOK, items)
}

// ═══════════════════════════════════════════════════════════════════════
// Роскомнадзор Report Handler
// ═══════════════════════════════════════════════════════════════════════

// handleGenerateRoskomnadzorReport — POST /api/v1/compliance/personal-data/report/rkn
//
// Access: admin only
func (s *Server) handleGenerateRoskomnadzorReport(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req roskomnadzorReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.OperatorName == "" || req.OperatorINN == "" {
		RespondError(w, r, NewValidationError("operator_name and operator_inn are required"))
		return
	}

	report, err := s.personalDataManager.GenerateRoskomnadzorReport(
		req.OperatorName, req.OperatorINN, req.OperatorAddress, req.SubjectCount,
	)
	if err != nil {
		s.logger.Error("failed to generate Роскомнадзор report", "error", err)
		RespondError(w, r, NewInternalError("failed to generate report", err))
		return
	}

	jsonResponse(w, http.StatusOK, report)

	s.logger.Info("Роскомнадзор report generated",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"operator", req.OperatorName,
	)
}

// handleExportInventory — GET /api/v1/compliance/personal-data/inventory/export
//
// Query params: format (csv/json, default json), anonymize (true/false)
// Access: admin, manager, owner
func (s *Server) handleExportInventory(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	anonymize := r.URL.Query().Get("anonymize") == "true"

	items, err := s.personalDataManager.GetInventory()
	if err != nil {
		s.logger.Error("failed to get inventory for export", "error", err)
		RespondError(w, r, NewInternalError("failed to export inventory", err))
		return
	}

	if anonymize {
		items = s.personalDataManager.AnonymizeData(items)
	}

	switch format {
	case "csv":
		exportInventoryCSV(w, items)
	default:
		jsonResponse(w, http.StatusOK, items)
	}
}

// exportInventoryCSV экспортирует inventory в CSV.
func exportInventoryCSV(w http.ResponseWriter, items []*compliance.DataInventoryItem) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=data_inventory.csv")

	// Пишем BOM для Excel
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	// Заголовки
	w.Write([]byte("ID,Category,Description,Fields,Storage,Purpose,RetentionDays,Anonymized,Encrypted,LegalBasis\n"))

	for _, item := range items {
		fields := ""
		for i, f := range item.DataFields {
			if i > 0 {
				fields += "; "
			}
			fields += f
		}
		line := strconv.Quote(item.ID) + "," +
			strconv.Quote(string(item.Category)) + "," +
			strconv.Quote(item.Description) + "," +
			strconv.Quote(fields) + "," +
			strconv.Quote(item.StorageLocation) + "," +
			strconv.Quote(string(item.Purpose)) + "," +
			strconv.Itoa(item.RetentionDays) + "," +
			strconv.FormatBool(item.Anonymized) + "," +
			strconv.FormatBool(item.Encrypted) + "," +
			strconv.Quote(item.LegalBasis) + "\n"
		w.Write([]byte(line))
	}
}
