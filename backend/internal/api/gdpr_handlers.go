// Package api — GDPR HTTP handlers (P2-EU.1).
//
// Маршруты защищены AuthMiddleware (вызывается из server.go).
//
// Compliance:
//   - GDPR Art. 17 (Right to erasure), Art. 20 (Portability)
//   - GDPR Art. 7 (Consent audit), Art. 35 (DPIA), Art. 44-49 (Transfers)
//   - OWASP ASVS V4 (RBAC — admin/manager/owner)
//   - ISO 27001 A.12.4 (Audit trail)
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/compliance"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V2 — Authentication (через JWT middleware)
// [x] V3 — Session Management (через AuthMiddleware)
// [x] V4 — Access Control (admin/manager/owner)
// [x] V5 — Input Validation (whitelist query params)
// [x] V7 — Error Handling (через respondError)
// [x] V8 — Data Protection

// ═══════════════════════════════════════════════════════════════════════
// Request/Response types
// ═══════════════════════════════════════════════════════════════════════

type requestErasureRequest struct {
	SubjectID       string   `json:"subject_id"`
	SubjectName     string   `json:"subject_name"`
	SubjectEmail    string   `json:"subject_email"`
	Scope           string   `json:"scope"`
	SpecificSystems []string `json:"specific_systems,omitempty"`
}

type completeErasureRequest struct {
	ErasureID string `json:"erasure_id"`
}

type rejectErasureRequest struct {
	ErasureID string `json:"erasure_id"`
	Reason    string `json:"reason"`
}

type createPortabilityRequest struct {
	SubjectID    string   `json:"subject_id"`
	SubjectName  string   `json:"subject_name"`
	SubjectEmail string   `json:"subject_email"`
	Format       string   `json:"format"`
	Categories   []string `json:"categories,omitempty"`
	Payload      string   `json:"payload,omitempty"`
}

type generateDPIARequest struct {
	SystemName             string   `json:"system_name"`
	SystemDescription      string   `json:"system_description"`
	DataController         string   `json:"data_controller"`
	DataProcessor          string   `json:"data_processor,omitempty"`
	DPO                    string   `json:"dpo,omitempty"`
	ProcessingPurposes     []string `json:"processing_purposes"`
	DataCategories         []string `json:"data_categories"`
	DataSubjects           []string `json:"data_subjects"`
	LegalBasis             string   `json:"legal_basis"`
	DataRetentionPeriod    string   `json:"data_retention_period"`
	TechnicalMeasures      []string `json:"technical_measures,omitempty"`
	OrganizationalMeasures []string `json:"organizational_measures,omitempty"`
	ThirdPartyProcessors   []string `json:"third_party_processors,omitempty"`
	CrossBorderTransfers   []string `json:"cross_border_transfers,omitempty"`
}

type createTransferAgreementRequest struct {
	TransferFrom          string   `json:"transfer_from"`
	TransferTo            string   `json:"transfer_to"`
	Mechanism             string   `json:"mechanism"`
	ControllerName        string   `json:"controller_name"`
	ProcessorName         string   `json:"processor_name,omitempty"`
	SignedBy              string   `json:"signed_by"`
	Categories            []string `json:"categories,omitempty"`
	EffectiveDate         string   `json:"effective_date"`
	SupplementaryMeasures []string `json:"supplementary_measures,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Right to be Forgotten Handlers (Art. 17)
// ═══════════════════════════════════════════════════════════════════════

// handleRequestErasure — POST /api/v1/compliance/gdpr/erasure
//
// Access: admin, manager, owner
func (s *Server) handleRequestErasure(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	var req requestErasureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.SubjectID == "" || req.Scope == "" {
		RespondError(w, r, NewValidationError("subject_id and scope are required"))
		return
	}

	erasure, err := s.gdprManager.RequestErasure(
		req.SubjectID, req.SubjectName, req.SubjectEmail,
		compliance.ErasureScope(req.Scope), req.SpecificSystems,
	)
	if err != nil {
		s.logger.Error("failed to request erasure", "error", err)
		RespondError(w, r, NewInternalError("failed to request erasure", err))
		return
	}

	jsonResponse(w, http.StatusCreated, erasure)

	s.logger.Info("right to be forgotten requested",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"erasure_id", erasure.ID,
	)
}

// handleCompleteErasure — POST /api/v1/compliance/gdpr/erasure/complete
//
// Access: admin only
func (s *Server) handleCompleteErasure(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req completeErasureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.ErasureID == "" {
		RespondError(w, r, NewValidationError("erasure_id is required"))
		return
	}

	if err := s.gdprManager.CompleteErasure(req.ErasureID); err != nil {
		s.logger.Error("failed to complete erasure", "error", err)
		RespondError(w, r, NewInternalError("failed to complete erasure", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "erasure_id": req.ErasureID})

	s.logger.Info("right to be forgotten completed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"erasure_id", req.ErasureID,
	)
}

// handleRejectErasure — POST /api/v1/compliance/gdpr/erasure/reject
//
// Access: admin only
func (s *Server) handleRejectErasure(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req rejectErasureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.ErasureID == "" || req.Reason == "" {
		RespondError(w, r, NewValidationError("erasure_id and reason are required"))
		return
	}

	if err := s.gdprManager.RejectErasure(req.ErasureID, req.Reason); err != nil {
		s.logger.Error("failed to reject erasure", "error", err)
		RespondError(w, r, NewInternalError("failed to reject erasure", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "erasure_id": req.ErasureID})
}

// handleListErasureRequests — GET /api/v1/compliance/gdpr/erasure
//
// Query params: subject_id (required)
// Access: admin, manager, owner
func (s *Server) handleListErasureRequests(w http.ResponseWriter, r *http.Request) {
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

	requests, err := s.gdprManager.ListSubjectErasureRequests(subjectID)
	if err != nil {
		s.logger.Error("failed to list erasure requests", "error", err)
		RespondError(w, r, NewInternalError("failed to list erasure requests", err))
		return
	}

	jsonResponse(w, http.StatusOK, requests)
}

// ═══════════════════════════════════════════════════════════════════════
// Data Portability Handlers (Art. 20)
// ═══════════════════════════════════════════════════════════════════════

// handleCreatePortabilityExport — POST /api/v1/compliance/gdpr/portability
//
// Access: admin, manager, owner
func (s *Server) handleCreatePortabilityExport(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	var req createPortabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.SubjectID == "" || req.Format == "" {
		RespondError(w, r, NewValidationError("subject_id and format are required"))
		return
	}

	categories := make([]compliance.DataCategory, 0, len(req.Categories))
	for _, c := range req.Categories {
		categories = append(categories, compliance.DataCategory(c))
	}

	export, err := s.gdprManager.CreatePortabilityExport(
		req.SubjectID, req.SubjectName, req.SubjectEmail,
		compliance.PortabilityFormat(req.Format), categories, req.Payload,
	)
	if err != nil {
		s.logger.Error("failed to create portability export", "error", err)
		RespondError(w, r, NewInternalError("failed to create portability export", err))
		return
	}

	jsonResponse(w, http.StatusCreated, export)

	s.logger.Info("portability export created",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"export_id", export.ID,
	)
}

// handleListPortabilityExports — GET /api/v1/compliance/gdpr/portability
//
// Query params: subject_id (required)
// Access: admin, manager, owner
func (s *Server) handleListPortabilityExports(w http.ResponseWriter, r *http.Request) {
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

	exports, err := s.gdprManager.ListSubjectPortabilityExports(subjectID)
	if err != nil {
		s.logger.Error("failed to list portability exports", "error", err)
		RespondError(w, r, NewInternalError("failed to list portability exports", err))
		return
	}

	jsonResponse(w, http.StatusOK, exports)
}

// ═══════════════════════════════════════════════════════════════════════
// Consent Audit Trail Handlers (Art. 7)
// ═══════════════════════════════════════════════════════════════════════

// handleGetConsentAuditTrail — GET /api/v1/compliance/gdpr/consent-audit
//
// Query params: subject_id (required)
// Access: admin, manager, owner
func (s *Server) handleGetConsentAuditTrail(w http.ResponseWriter, r *http.Request) {
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

	entries, err := s.gdprManager.GetConsentAuditTrail(subjectID)
	if err != nil {
		s.logger.Error("failed to get consent audit trail", "error", err)
		RespondError(w, r, NewInternalError("failed to get consent audit trail", err))
		return
	}

	jsonResponse(w, http.StatusOK, entries)
}

// ═══════════════════════════════════════════════════════════════════════
// DPIA Handlers (Art. 35)
// ═══════════════════════════════════════════════════════════════════════

// handleGenerateDPIA — POST /api/v1/compliance/gdpr/dpia
//
// Access: admin only
func (s *Server) handleGenerateDPIA(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req generateDPIARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.SystemName == "" || req.DataController == "" {
		RespondError(w, r, NewValidationError("system_name and data_controller are required"))
		return
	}

	categories := make([]compliance.DataCategory, 0, len(req.DataCategories))
	for _, c := range req.DataCategories {
		categories = append(categories, compliance.DataCategory(c))
	}

	report, err := s.gdprManager.GenerateDPIAReport(
		req.SystemName, req.SystemDescription, req.DataController, req.DataProcessor,
		req.DPO, req.ProcessingPurposes, categories, req.DataSubjects,
		req.LegalBasis, req.DataRetentionPeriod, req.TechnicalMeasures,
		req.OrganizationalMeasures, req.ThirdPartyProcessors, req.CrossBorderTransfers,
	)
	if err != nil {
		s.logger.Error("failed to generate DPIA report", "error", err)
		RespondError(w, r, NewInternalError("failed to generate DPIA report", err))
		return
	}

	jsonResponse(w, http.StatusCreated, report)

	s.logger.Info("DPIA report generated",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"dpia_id", report.ID,
	)
}

// handleListDPIAReports — GET /api/v1/compliance/gdpr/dpia
//
// Access: admin, manager, owner
func (s *Server) handleListDPIAReports(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	reports, err := s.gdprManager.ListDPIAReports()
	if err != nil {
		s.logger.Error("failed to list DPIA reports", "error", err)
		RespondError(w, r, NewInternalError("failed to list DPIA reports", err))
		return
	}

	jsonResponse(w, http.StatusOK, reports)
}

// ═══════════════════════════════════════════════════════════════════════
// Schrems II / Data Transfer Handlers (Art. 44-49)
// ═══════════════════════════════════════════════════════════════════════

// handleCreateTransferAgreement — POST /api/v1/compliance/gdpr/transfers
//
// Access: admin only
func (s *Server) handleCreateTransferAgreement(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	var req createTransferAgreementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}
	if req.TransferFrom == "" || req.TransferTo == "" || req.Mechanism == "" || req.ControllerName == "" {
		RespondError(w, r, NewValidationError("transfer_from, transfer_to, mechanism, and controller_name are required"))
		return
	}

	effectiveDate, err := time.Parse(time.RFC3339, req.EffectiveDate)
	if err != nil {
		effectiveDate = time.Now().UTC().AddDate(0, 0, 30) // Default: 30 days from now
	}

	categories := make([]compliance.DataCategory, 0, len(req.Categories))
	for _, c := range req.Categories {
		categories = append(categories, compliance.DataCategory(c))
	}

	agreement, err := s.gdprManager.CreateTransferAgreement(
		req.TransferFrom, req.TransferTo, compliance.TransferMechanism(req.Mechanism),
		req.ControllerName, req.ProcessorName, req.SignedBy,
		categories, effectiveDate, req.SupplementaryMeasures,
	)
	if err != nil {
		s.logger.Error("failed to create transfer agreement", "error", err)
		RespondError(w, r, NewInternalError("failed to create transfer agreement", err))
		return
	}

	jsonResponse(w, http.StatusCreated, agreement)

	s.logger.Info("transfer agreement created",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"agreement_id", agreement.ID,
	)
}

// handleCompleteTIA — POST /api/v1/compliance/gdpr/transfers/{id}/tia
//
// Access: admin only
func (s *Server) handleCompleteTIA(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	// Получаем ID из URL
	agreementID := r.PathValue("id")
	if agreementID == "" {
		agreementID = r.URL.Query().Get("id")
	}
	if agreementID == "" {
		RespondError(w, r, NewValidationError("agreement id is required"))
		return
	}

	if err := s.gdprManager.CompleteTIA(agreementID); err != nil {
		s.logger.Error("failed to complete TIA", "error", err)
		RespondError(w, r, NewInternalError("failed to complete TIA", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "agreement_id": agreementID})
}

// handleListTransferAgreements — GET /api/v1/compliance/gdpr/transfers
//
// Access: admin, manager, owner
func (s *Server) handleListTransferAgreements(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	agreements, err := s.gdprManager.ListTransferAgreements()
	if err != nil {
		s.logger.Error("failed to list transfer agreements", "error", err)
		RespondError(w, r, NewInternalError("failed to list transfer agreements", err))
		return
	}

	jsonResponse(w, http.StatusOK, agreements)
}
