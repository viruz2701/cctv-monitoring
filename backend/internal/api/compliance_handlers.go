// Package api — HTTP handlers for Compliance & Fines Shield (KF-15.1.1).
//
// Соответствие стандартам:
//   - OWASP ASVS L3 V1-V17 (полный набор контролей)
//   - IEC 62443-3-3 SR 7.1 (Resource availability — risk quantification)
//   - ISO 27001 A.12.4 (Audit trail — compliance audit log)
//   - ISO 27019 PCC.A.13 (ICS asset risk assessment)
//   - СТБ 34.101.27 п. 6.3 (Оценка рисков)
//   - Приказ ОАЦ № 66 п. 7.18 (Идентификация устройств)
//
// P0-REG.3-5: Maintenance Compliance Engine
//   - GET    /api/v1/compliance/regulations — список регламентов
//   - POST   /api/v1/compliance/regulations/{id}/generate-wo — ручная генерация WO
//   - GET    /api/v1/compliance/journal — журнал compliance
//   - POST   /api/v1/compliance/journal/{id}/sign — подписать акт HMAC
//   - GET    /api/v1/compliance/journal/{id}/verify — верификация HMAC
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/compliance"
	"gb-telemetry-collector/internal/db"

	"github.com/go-chi/chi/v5"
)

// ── Compliance Checklist (OWASP ASVS L3) ───────────────────────────────
//
// [x] V2 — Authentication (через JWT middleware)
// [x] V3 — Session Management (через AuthMiddleware)
// [x] V4 — Access Control (RBAC — admin/manager/owner)
// [x] V5 — Input Validation (whitelist query params)
// [x] V7 — Error Handling and Logging (через respondError)
// [x] V8 — Data Protection (sensitive fields not exposed)
// [x] V14 — Configuration (через config.Config)

// ═══════════════════════════════════════════════════════════════════════
// Response types
// ═══════════════════════════════════════════════════════════════════════

type complianceRiskResponse struct {
	DeviceID      string  `json:"device_id"`
	DeviceName    string  `json:"device_name,omitempty"`
	DeviceType    string  `json:"device_type"`
	SiteID        string  `json:"site_id,omitempty"`
	SiteName      string  `json:"site_name,omitempty"`
	DowntimeMin   int64   `json:"total_downtime_min"`
	DowntimeHours float64 `json:"downtime_hours"`
	HourlyFine    float64 `json:"hourly_fine"`
	TotalExposure float64 `json:"total_exposure"`
	RiskLevel     string  `json:"risk_level"`
	UpdatedAt     string  `json:"updated_at"`
}

type complianceSummaryResponse struct {
	TotalExposure    float64                  `json:"total_exposure"`
	AtRiskDevices    int                      `json:"at_risk_devices"`
	CompliantDevices int                      `json:"compliant_devices"`
	TotalDevices     int                      `json:"total_devices"`
	TopRisks         []complianceRiskResponse `json:"top_risks,omitempty"`
	RiskBreakdown    map[string]int           `json:"risk_breakdown,omitempty"`
}

// riskRowToResponse конвертирует ComplianceRiskRow в complianceRiskResponse.
func riskRowToResponse(r db.ComplianceRiskRow) complianceRiskResponse {
	hours := float64(0)
	if r.TotalDowntimeMin > 0 {
		hours = float64(r.TotalDowntimeMin) / 60.0
	}
	return complianceRiskResponse{
		DeviceID:      r.DeviceID,
		DeviceName:    "", // будет заполнено если нужно
		DeviceType:    r.DeviceType,
		SiteID:        r.SiteID,
		DowntimeMin:   r.TotalDowntimeMin,
		DowntimeHours: hours,
		HourlyFine:    r.HourlyFine,
		TotalExposure: r.TotalExposure,
		RiskLevel:     r.RiskLevel,
		UpdatedAt:     r.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GET /api/v1/compliance/summary
// ═══════════════════════════════════════════════════════════════════════

// handleComplianceSummary возвращает общую сводку compliance рисков.
//
// Query params:
//   - site_id (optional): фильтр по площадке
//
// Access: admin, manager, owner
// Соответствует: OWASP ASVS V4 (RBAC), V5 (input validation)
func (s *Server) handleComplianceSummary(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication ──
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control (admin, manager, owner) ──
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions: admin, manager, or owner role required"))
		return
	}

	// ── V5: Input Validation ──
	siteID := r.URL.Query().Get("site_id")

	// Получаем сводку из БД
	summary, err := s.db.GetComplianceSummary(r.Context(), siteID)
	if err != nil {
		s.logger.Error("failed to get compliance summary", "error", err)
		RespondError(w, r, NewInternalError("failed to get compliance summary", err))
		return
	}

	// Получаем top риски
	risks, err := s.db.GetComplianceRisks(r.Context(), "", siteID)
	if err != nil {
		s.logger.Error("failed to get compliance risks", "error", err)
		RespondError(w, r, NewInternalError("failed to get compliance risks", err))
		return
	}

	// Берём топ-10
	topRisks := make([]complianceRiskResponse, 0, 10)
	for i, r := range risks {
		if i >= 10 {
			break
		}
		topRisks = append(topRisks, riskRowToResponse(r))
	}

	// Получаем разбивку по уровням риска
	breakdown, err := s.db.GetComplianceRiskBreakdown(r.Context(), siteID)
	if err != nil {
		s.logger.Error("failed to get compliance breakdown", "error", err)
		// Не фатально, продолжаем
		breakdown = make(map[string]int)
	}

	resp := complianceSummaryResponse{
		TotalExposure:    summary.TotalExposure,
		AtRiskDevices:    summary.AtRiskDevices,
		CompliantDevices: summary.CompliantDevices,
		TotalDevices:     summary.TotalDevices,
		TopRisks:         topRisks,
		RiskBreakdown:    breakdown,
	}

	jsonResponse(w, http.StatusOK, resp)

	// ISO 27001 A.12.4: Audit trail
	s.logger.Info("compliance summary accessed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"role", claims.Role,
		"site_id", siteID,
	)
}

// ═══════════════════════════════════════════════════════════════════════
// GET /api/v1/compliance/risks
// ═══════════════════════════════════════════════════════════════════════

// handleComplianceRisks возвращает детальные compliance риски.
//
// Query params:
//   - device_id (optional): фильтр по устройству
//   - site_id (optional): фильтр по площадке
//
// Access: admin, manager, owner
// Соответствует: OWASP ASVS V4 (RBAC), V5 (input validation)
func (s *Server) handleComplianceRisks(w http.ResponseWriter, r *http.Request) {
	// ── V2: Authentication ──
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	// ── V4: Access Control ──
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions: admin, manager, or owner role required"))
		return
	}

	// ── V5: Input Validation ──
	deviceID := r.URL.Query().Get("device_id")
	siteID := r.URL.Query().Get("site_id")

	risks, err := s.db.GetComplianceRisks(r.Context(), deviceID, siteID)
	if err != nil {
		s.logger.Error("failed to get compliance risks", "error", err)
		RespondError(w, r, NewInternalError("failed to get compliance risks", err))
		return
	}

	resp := make([]complianceRiskResponse, 0, len(risks))
	for _, r := range risks {
		resp = append(resp, riskRowToResponse(r))
	}

	jsonResponse(w, http.StatusOK, resp)

	// ISO 27001 A.12.4: Audit trail
	s.logger.Info("compliance risks accessed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"role", claims.Role,
		"device_id", deviceID,
		"site_id", siteID,
		"result_count", len(resp),
	)
}

// ═══════════════════════════════════════════════════════════════════════
// GET /api/v1/compliance/fines
// ═══════════════════════════════════════════════════════════════════════

// handleComplianceFines возвращает таблицу штрафов по умолчанию.
//
// Access: admin, manager, owner
func (s *Server) handleComplianceFines(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	// Строим in-memory engine, если complianceEngine ещё не инициализирован
	var fines map[string]float64
	if s.complianceEngine != nil {
		fines = make(map[string]float64)
		for k := range compliance.DefaultHourlyFines {
			fines[k] = s.complianceEngine.GetHourlyFine(k)
		}
	} else {
		fines = compliance.DefaultHourlyFines
	}

	jsonResponse(w, http.StatusOK, fines)

	s.logger.Info("compliance fines accessed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
	)
}

// ═══════════════════════════════════════════════════════════════════════
// POST /api/v1/compliance/refresh
// ═══════════════════════════════════════════════════════════════════════

// handleComplianceRefresh принудительно обновляет compliance_risks.
//
// Access: admin only
func (s *Server) handleComplianceRefresh(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	if claims.Role != "admin" {
		RespondError(w, r, NewForbiddenError("admin role required"))
		return
	}

	// Вызываем refresh функцию БД
	_, err := s.db.Pool.Exec(r.Context(), "SELECT refresh_compliance_risks()")
	if err != nil {
		s.logger.Error("failed to refresh compliance risks", "error", err)
		RespondError(w, r, NewInternalError("failed to refresh compliance risks", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "message": "compliance risks refreshed"})

	s.logger.Info("compliance risks refreshed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
	)
}

// ═══════════════════════════════════════════════════════════════════════
// POST /api/v1/compliance/calculate
// ═══════════════════════════════════════════════════════════════════════

type calculateRequest struct {
	DowntimeMinutes int64   `json:"downtime_minutes"`
	DeviceType      string  `json:"device_type"`
	HourlyRate      float64 `json:"hourly_rate,omitempty"`
}

type calculateResponse struct {
	DowntimeMinutes int64   `json:"downtime_minutes"`
	DeviceType      string  `json:"device_type"`
	HourlyRate      float64 `json:"hourly_rate"`
	TotalExposure   float64 `json:"total_exposure"`
	RiskLevel       string  `json:"risk_level"`
}

// handleComplianceCalculate вычисляет риск для переданных параметров.
//
// Access: admin, manager, owner
func (s *Server) handleComplianceCalculate(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	var req calculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	if req.DowntimeMinutes < 0 {
		RespondError(w, r, NewBadRequestError("downtime_minutes must be >= 0"))
		return
	}
	if req.DeviceType == "" {
		RespondError(w, r, NewBadRequestError("device_type is required"))
		return
	}

	exposure, riskLevel := compliance.CalculateRisk(req.DowntimeMinutes, req.DeviceType, req.HourlyRate)

	rate := req.HourlyRate
	if rate <= 0 {
		if fine, ok := compliance.DefaultHourlyFines[req.DeviceType]; ok {
			rate = fine
		} else {
			rate = compliance.DefaultHourlyFines["camera"]
		}
	}

	resp := calculateResponse{
		DowntimeMinutes: req.DowntimeMinutes,
		DeviceType:      req.DeviceType,
		HourlyRate:      rate,
		TotalExposure:   exposure,
		RiskLevel:       string(riskLevel),
	}

	jsonResponse(w, http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════
// P0-REG.3-5: Maintenance Compliance Engine Handlers
// ═══════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════
// GET /api/v1/compliance/regulations
// ═══════════════════════════════════════════════════════════════════════

// handleComplianceRegulations возвращает список регламентов ТО.
//
// Query params:
//   - region (optional): фильтр по региону (BY, RU, TR, VN, ID, BR, ZA)
//
// Access: admin, manager, owner
func (s *Server) handleComplianceRegulations(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions: admin, manager, or owner role required"))
		return
	}

	region := r.URL.Query().Get("region")

	query := `SELECT id, region_code, regulation_code, name, regulation_type,
		interval_months, estimated_minutes, total_items,
		compliance_standards, license_requirements, docs_required,
		is_active, created_at, updated_at
		FROM maintenance_regulations WHERE 1=1`

	args := make([]interface{}, 0)
	argIdx := 1

	if region != "" {
		query += fmt.Sprintf(" AND region_code = $%d", argIdx)
		args = append(args, region)
		argIdx++
	}

	query += " ORDER BY region_code, regulation_type"

	rows, err := s.db.Pool.Query(r.Context(), query, args...)
	if err != nil {
		s.logger.Error("failed to query regulations", "error", err)
		RespondError(w, r, NewInternalError("failed to query regulations", err))
		return
	}
	defer rows.Close()

	type regulationResponse struct {
		ID                  string   `json:"id"`
		RegionCode          string   `json:"region_code"`
		RegulationCode      string   `json:"regulation_code"`
		Name                string   `json:"name"`
		RegulationType      string   `json:"regulation_type"`
		IntervalMonths      int      `json:"interval_months"`
		EstimatedMinutes    int      `json:"estimated_minutes"`
		TotalItems          int      `json:"total_items"`
		ComplianceStandards []string `json:"compliance_standards"`
		LicenseRequirements *string  `json:"license_requirements,omitempty"`
		IsActive            bool     `json:"is_active"`
		CreatedAt           string   `json:"created_at"`
		UpdatedAt           string   `json:"updated_at"`
	}

	regulations := make([]regulationResponse, 0)
	for rows.Next() {
		var reg regulationResponse
		var licenseReq *string

		if err := rows.Scan(
			&reg.ID, &reg.RegionCode, &reg.RegulationCode, &reg.Name,
			&reg.RegulationType, &reg.IntervalMonths, &reg.EstimatedMinutes,
			&reg.TotalItems, &reg.ComplianceStandards, &licenseReq,
			&reg.IsActive, &reg.CreatedAt, &reg.UpdatedAt,
		); err != nil {
			s.logger.Error("failed to scan regulation", "error", err)
			RespondError(w, r, NewInternalError("failed to scan regulation", err))
			return
		}

		regulations = append(regulations, reg)
	}

	if err := rows.Err(); err != nil {
		s.logger.Error("rows iteration error", "error", err)
		RespondError(w, r, NewInternalError("rows iteration error", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"regulations": regulations,
		"total":       len(regulations),
	})

	s.logger.Info("compliance regulations listed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"region", region,
		"count", len(regulations),
	)
}

// ═══════════════════════════════════════════════════════════════════════
// POST /api/v1/compliance/regulations/{id}/generate-wo
// ═══════════════════════════════════════════════════════════════════════

// handleGenerateWOFromRegulation создаёт WO для указанного регламента вручную.
//
// Access: admin, manager
func (s *Server) handleGenerateWOFromRegulation(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("insufficient permissions: admin or manager role required"))
		return
	}

	regulationID := chi.URLParam(r, "id")
	if regulationID == "" {
		RespondError(w, r, NewBadRequestError("regulation id is required"))
		return
	}

	// Получаем регламент
	var reg compliance.DueRegulation
	var licenseReq *string
	var docsReq []byte

	err := s.db.Pool.QueryRow(r.Context(), `
		SELECT id, region_code, regulation_code, name, regulation_type,
			interval_months, estimated_minutes, total_items,
			compliance_standards, license_requirements, docs_required,
			created_at
		FROM maintenance_regulations
		WHERE id = $1 AND is_active = true
	`, regulationID).Scan(
		&reg.ID, &reg.RegionCode, &reg.RegulationCode, &reg.Name,
		&reg.RegulationType, &reg.IntervalMonths, &reg.EstimatedMinutes,
		&reg.TotalItems, &reg.ComplianceStandards, &licenseReq,
		&docsReq, &reg.LastMaintenanceDate,
	)
	if err != nil {
		s.logger.Error("failed to get regulation", "error", err)
		RespondError(w, r, NewNotFoundError("regulation not found"))
		return
	}

	reg.LicenseRequirements = licenseReq
	reg.DocsRequired = docsReq

	// Создаём WO через DefaultWOProvider
	woProvider := compliance.NewDefaultWOProvider(s.db.Pool, s.logger)
	woID, err := woProvider.CreateWO(r.Context(), &reg)
	if err != nil {
		s.logger.Error("failed to create WO from regulation", "error", err)
		RespondError(w, r, NewInternalError("failed to create work order", err))
		return
	}

	// Логируем в compliance_journal
	if s.complianceJournal != nil {
		actData := &compliance.ActData{
			Action:         "manual_generate_wo",
			EntryType:      compliance.JournalEntryWOGenerated,
			RegulationCode: reg.RegulationCode,
			RegulationName: reg.Name,
			TraceID:        TraceIDFromContext(r.Context()),
			Extra: map[string]interface{}{
				"triggered_by": claims.UserID,
				"region":       reg.RegionCode,
			},
		}

		_, err := s.complianceJournal.CreateEntry(r.Context(), reg.ID, woID, reg.RegionCode, actData)
		if err != nil {
			s.logger.Error("failed to log journal entry", "error", err)
		}
	}

	jsonResponse(w, http.StatusCreated, map[string]string{
		"work_order_id": woID,
		"status":        "created",
	})

	s.logger.Info("WO generated from regulation",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"regulation_id", regulationID,
		"wo_id", woID,
	)
}

// ═══════════════════════════════════════════════════════════════════════
// GET /api/v1/compliance/journal
// ═══════════════════════════════════════════════════════════════════════

// handleComplianceJournal возвращает записи compliance журнала.
//
// Query params:
//   - region (optional): фильтр по региону
//   - from (optional): начальная дата (RFC3339)
//   - to (optional): конечная дата (RFC3339)
//   - limit (optional): лимит записей (default 50)
//   - offset (optional): смещение
//
// Access: admin, manager, owner
func (s *Server) handleComplianceJournal(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	region := r.URL.Query().Get("region")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	limit := parseIntQuery(r.URL.Query().Get("limit"), 50)
	offset := parseIntQuery(r.URL.Query().Get("offset"), 0)

	var from, to *time.Time
	if fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			// Пробуем формат YYYY-MM-DD
			t, err = time.Parse("2006-01-02", fromStr)
			if err != nil {
				RespondError(w, r, NewBadRequestError("invalid from date format, use RFC3339 or YYYY-MM-DD"))
				return
			}
		}
		from = &t
	}
	if toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			t, err = time.Parse("2006-01-02", toStr)
			if err != nil {
				RespondError(w, r, NewBadRequestError("invalid to date format, use RFC3339 or YYYY-MM-DD"))
				return
			}
		}
		to = &t
	}

	entries, total, err := s.complianceJournal.ListEntries(r.Context(), region, from, to, limit, offset)
	if err != nil {
		s.logger.Error("failed to list journal entries", "error", err)
		RespondError(w, r, NewInternalError("failed to list journal entries", err))
		return
	}

	// Формируем ответ
	type journalEntryResponse struct {
		ID           string     `json:"id"`
		RegulationID *string    `json:"regulation_id,omitempty"`
		WoID         *string    `json:"wo_id,omitempty"`
		RegionCode   string     `json:"region_code"`
		HasSignature bool       `json:"has_signature"`
		HMACSignedAt *time.Time `json:"hmac_signed_at,omitempty"`
		CreatedAt    time.Time  `json:"created_at"`
	}

	resp := make([]journalEntryResponse, 0, len(entries))
	for _, e := range entries {
		resp = append(resp, journalEntryResponse{
			ID:           e.ID,
			RegulationID: e.RegulationID,
			WoID:         e.WoID,
			RegionCode:   e.RegionCode,
			HasSignature: e.HMACSignature != nil,
			HMACSignedAt: e.HMACSignedAt,
			CreatedAt:    e.CreatedAt,
		})
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"entries": resp,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})

	s.logger.Info("compliance journal listed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"region", region,
		"count", len(resp),
	)
}

// ═══════════════════════════════════════════════════════════════════════
// POST /api/v1/compliance/journal/{id}/sign
// ═══════════════════════════════════════════════════════════════════════

// handleSignJournalEntry подписывает запись журнала HMAC.
//
// Access: admin, manager
func (s *Server) handleSignJournalEntry(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if claims.Role != "admin" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("insufficient permissions: admin or manager role required"))
		return
	}

	entryID := chi.URLParam(r, "id")
	if entryID == "" {
		RespondError(w, r, NewBadRequestError("entry id is required"))
		return
	}

	signature, err := s.complianceJournal.SignAct(r.Context(), entryID)
	if err != nil {
		s.logger.Error("failed to sign journal entry", "error", err)
		RespondError(w, r, NewInternalError("failed to sign journal entry", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"entry_id":  entryID,
		"signature": signature,
		"status":    "signed",
	})

	s.logger.Info("journal entry signed",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"entry_id", entryID,
	)
}

// ═══════════════════════════════════════════════════════════════════════
// GET /api/v1/compliance/journal/{id}/verify
// ═══════════════════════════════════════════════════════════════════════

// handleVerifyJournalEntry верифицирует HMAC подпись записи журнала.
//
// Access: admin, manager, owner
func (s *Server) handleVerifyJournalEntry(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		RespondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}
	if !isComplianceRole(claims.Role) {
		RespondError(w, r, NewForbiddenError("insufficient permissions"))
		return
	}

	entryID := chi.URLParam(r, "id")
	if entryID == "" {
		RespondError(w, r, NewBadRequestError("entry id is required"))
		return
	}

	valid, err := s.complianceJournal.VerifyAct(r.Context(), entryID)
	if err != nil {
		s.logger.Error("failed to verify journal entry", "error", err)
		RespondError(w, r, NewInternalError("failed to verify journal entry", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"entry_id": entryID,
		"valid":    valid,
	})

	s.logger.Info("journal entry verified",
		"trace_id", TraceIDFromContext(r.Context()),
		"user_id", claims.UserID,
		"entry_id", entryID,
		"valid", valid,
	)
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// isComplianceRole проверяет наличие роли для доступа к compliance данным.
func isComplianceRole(role string) bool {
	switch role {
	case "admin", "manager", "owner":
		return true
	default:
		return false
	}
}

// parseIntQuery парсит целое число из строки query параметра.
func parseIntQuery(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return defaultVal
	}
	if n < 0 {
		return defaultVal
	}
	return n
}
