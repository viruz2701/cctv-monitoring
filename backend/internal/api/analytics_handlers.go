package api

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"

	"gb-telemetry-collector/internal/analytics"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/models"
)

// ---------- Аналитика ----------

func (s *Server) getPredictions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	deviceID := r.URL.Query().Get("device_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if claims.Role == "owner" {
		dev, ok := s.stateManager.Get(deviceID)
		if !ok || dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			RespondError(w, r, NewForbiddenError("forbidden"))
			return
		}
	}
	predictions, err := s.db.GetPredictions(deviceID, limit)
	if err != nil {
		s.logger.Error("failed to get predictions", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}
	jsonResponse(w, http.StatusOK, predictions)
}

// ---------- Поиск логов ----------

func (s *Server) searchLogs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	level := r.URL.Query().Get("level")
	keyword := r.URL.Query().Get("keyword")
	timeFrom := r.URL.Query().Get("time_from")
	timeTo := r.URL.Query().Get("time_to")

	logs, err := s.db.SearchLogs(deviceID, level, keyword, timeFrom, timeTo)
	if err != nil {
		s.logger.Error("failed to search logs", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}
	jsonResponse(w, http.StatusOK, logs)
}

// ---------- Reliability Metrics (AN-10.1.1) ----------

// getReliability возвращает MTBF/MTTR метрики по vendor_type и device_type.
//
// Эндпоинт: GET /api/v1/analytics/reliability
// Параметры:
//   - vendor_type (optional): фильтр по вендору
//   - device_type (optional): фильтр по типу устройства
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через query params)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - ISO 27001 A.12.6.1 (Capacity management — reliability metrics)
func (s *Server) getReliability(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	vendorType := r.URL.Query().Get("vendor_type")
	deviceType := r.URL.Query().Get("device_type")

	results, err := s.db.GetDeviceReliability(r.Context(), vendorType, deviceType)
	if err != nil {
		s.logger.Error("failed to get device reliability", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// Конвертируем в response DTO
	type reliabilityResponse struct {
		VendorType           string  `json:"vendor_type"`
		DeviceType           string  `json:"device_type"`
		DeviceCount          int64   `json:"device_count"`
		MTBFHours            float64 `json:"mtbf_hours"`
		MTTRMinutes          float64 `json:"mttr_minutes"`
		TotalDowntimeMinutes int64   `json:"total_downtime_minutes"`
		TotalCompletions     int64   `json:"total_completions"`
	}

	response := make([]reliabilityResponse, 0, len(results))
	for _, r := range results {
		response = append(response, reliabilityResponse{
			VendorType:           r.VendorType,
			DeviceType:           r.DeviceType,
			DeviceCount:          r.DeviceCount,
			MTBFHours:            r.MTBFHours,
			MTTRMinutes:          r.MTTRMinutes,
			TotalDowntimeMinutes: r.TotalDowntimeMinutes,
			TotalCompletions:     r.TotalCompletions,
		})
	}

	jsonResponse(w, http.StatusOK, response)
}

// ---------- TCO Per Device (AN-10.1.3) ----------

// getTCOPerDevice возвращает TCO (Total Cost of Ownership) per device.
//
// Эндпоинт: GET /api/v1/analytics/tco
// Параметры:
//   - vendor_type (optional): фильтр по вендору
//   - device_type (optional): фильтр по типу устройства
//   - device_id (optional): фильтр по ID устройства
//   - limit (optional): лимит записей (default: 50, max: 500)
//   - offset (optional): смещение
//
// TCO = Purchase + Labor + Parts + Downtime
// Данные берутся из mv_tco_per_device (материализованное представление,
// обновляется через REFRESH MATERIALIZED VIEW).
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через query params)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
//   - IEC 62443 SR 7.1 (Resource availability — asset TCO)
func (s *Server) getTCOPerDevice(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	// Парсим параметры фильтрации
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}
	offset := 0
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filter := models.TCOFilter{
		VendorType: r.URL.Query().Get("vendor_type"),
		DeviceType: r.URL.Query().Get("device_type"),
		DeviceID:   r.URL.Query().Get("device_id"),
		Limit:      limit,
		Offset:     offset,
	}

	results, err := s.db.GetTCOPerDevice(r.Context(), filter)
	if err != nil {
		s.logger.Error("failed to get TCO per device", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	jsonResponse(w, http.StatusOK, results)
}

// ---------- Downtime Cost by Site (BIZ-01) ----------

// getDowntimeCostsBySite возвращает стоимость простоев с группировкой по объектам.
// GET /api/v1/analytics/downtime-costs
//
// BIZ-01: TCO и стоимость простоя — аргумент для продажи директору.
// Формула: Total Downtime Cost = Σ(downtime_hours × cost_per_hour)
//
// Compliance:
//   - OWASP ASVS V4.1 (RBAC — admin/manager/support only)
//   - OWASP ASVS V7.1 (Error handling)
//   - ISO 27001 A.12.6.1 (Cost tracking)
func (s *Server) getDowntimeCostsBySite(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	// Используем существующий TCO per device с группировкой
	tcoResults, err := s.db.GetTCOPerDevice(r.Context(), models.TCOFilter{Limit: 500})
	if err != nil {
		s.logger.Error("failed to get downtime costs", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	var totalDowntimeCost float64
	totalDevices := len(tcoResults)
	devicesWithDowntime := 0
	topByDowntime := make([]map[string]interface{}, 0)

	for _, d := range tcoResults {
		totalDowntimeCost += d.TotalDowntimeCost
		if d.TotalDowntimeCost > 0 {
			devicesWithDowntime++
			topByDowntime = append(topByDowntime, map[string]interface{}{
				"device_id":       d.DeviceID,
				"device_name":     d.DeviceName,
				"device_type":     d.DeviceType,
				"downtime_cost":   d.TotalDowntimeCost,
				"downtime_events": d.TotalDowntimeEvents,
				"tco":             d.TCO,
			})
		}
	}

	// Сортируем по убыванию downtime cost
	sort.Slice(topByDowntime, func(i, j int) bool {
		ci := topByDowntime[i]["downtime_cost"].(float64)
		cj := topByDowntime[j]["downtime_cost"].(float64)
		return ci > cj
	})

	// Лимитируем топ-20
	if len(topByDowntime) > 20 {
		topByDowntime = topByDowntime[:20]
	}

	result := map[string]interface{}{
		"total_downtime_cost":   totalDowntimeCost,
		"total_devices":         totalDevices,
		"devices_with_downtime": devicesWithDowntime,
		"top_by_downtime_cost":  topByDowntime,
	}

	jsonResponse(w, http.StatusOK, result)
}

// ---------- Work Order Cost Summary (WO-4.4.5) ----------

// getWorkOrderCosts возвращает агрегированную сводку затрат по Work Orders.
//
// Эндпоинт: GET /api/v1/analytics/wo-costs
//
// Compliance:
//   - OWASP ASVS V4.1 (RBAC — admin/manager/support only)
//   - OWASP ASVS V5.1 (Input validation — no user input)
//   - OWASP ASVS V7.1 (Structured response — no sensitive data)
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
func (s *Server) getWorkOrderCosts(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	summary, err := s.db.GetWorkOrderCostSummary(r.Context())
	if err != nil {
		s.logger.Error("failed to get work order cost summary", "error", err)
		RespondError(w, r, NewInternalError("internal error", nil))
		return
	}

	// Используем GetWorkOrderCostBreakdownFromSummary — не делаем повторный
	// дорогой запрос GetWorkOrderCostSummary (см. WO-4.4.5)
	breakdown, err := s.db.GetWorkOrderCostBreakdownFromSummary(r.Context(), summary)
	if err != nil {
		s.logger.Warn("failed to get work order cost breakdown", "error", err)
		breakdown = []models.WorkOrderCostBreakdown{}
	}

	type costResponse struct {
		Summary   models.WorkOrderCostSummary     `json:"summary"`
		Breakdown []models.WorkOrderCostBreakdown `json:"breakdown"`
	}

	jsonResponse(w, http.StatusOK, costResponse{
		Summary:   *summary,
		Breakdown: breakdown,
	})
}

// getAnalyticsCost возвращает аналитику затрат (заглушка).
// TODO: реализовать полноценный запрос к БД.
func (s *Server) getAnalyticsCost(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, []interface{}{})
}

// getAnalyticsCostTrend возвращает тренд затрат (заглушка).
// TODO: реализовать полноценный запрос к БД.
func (s *Server) getAnalyticsCostTrend(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, []interface{}{})
}

// getAnalyticsCostTop возвращает топ дорогих устройств (заглушка).
// TODO: реализовать полноценный запрос к БД.
func (s *Server) getAnalyticsCostTop(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, []interface{}{})
}

// ═══════════════════════════════════════════════════════════════════════════
// P2-BI: Self-Service Analytics Query Builder Handlers
// ═══════════════════════════════════════════════════════════════════════════

// handleBIGetTemplates возвращает список доступных BI-шаблонов.
//
// Эндпоинт: GET /api/v1/analytics/bi/templates
//
// Ответ: массив QueryTemplate (ID, Name, Description, Dimensions, Measures, DateField)
//
// Compliance:
//   - OWASP ASVS V4.1 (RBAC — admin/manager/support/owner могут смотреть)
//   - OWASP ASVS V7.1 (Error handling — structured response, no sensitive data)
//   - ISO 27001 A.12.6.1 (Capacity management — analytics templates)
func (s *Server) handleBIGetTemplates(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	templates := s.queryBuilder.GetTemplates()
	jsonResponse(w, http.StatusOK, templates)
}

// handleBIExecuteQuery выполняет BI-запрос на основе шаблона и параметров.
//
// Эндпоинт: POST /api/v1/analytics/bi/query
//
//	Body: QueryParams {
//	  template_id: string (required)
//	  dimensions: string[] (optional — выбранные поля для GROUP BY)
//	  measures: string[] (optional — выбранные метрики)
//	  filters: FilterCondition[] (optional — условия)
//	  time_from: string (optional — RFC3339)
//	  time_to: string (optional — RFC3339)
//	  limit: int (optional)
//	  offset: int (optional)
//	  order_by: string (optional)
//	  order_dir: string (optional — "asc" | "desc")
//	}
//
// Ответ: QueryResult { columns: string[], rows: any[][], total: int, took: string }
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — JSON body validation)
//   - OWASP ASVS V5.3 (Output encoding — JSON encoder, no raw HTML)
//   - OWASP ASVS V6.2 (SQL injection prevention — parameterized queries)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - ISO 27001 A.12.4.1 (Event logging — audit trail)
//   - IEC 62443 SR 3.1 (Input validation — field whitelist)
func (s *Server) handleBIExecuteQuery(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" && claims.Role != "manager" {
		RespondError(w, r, NewForbiddenError("forbidden"))
		return
	}

	var params analytics.QueryParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		RespondError(w, r, NewValidationError("invalid JSON body: "+err.Error()))
		return
	}

	// Валидация: template_id обязателен
	if params.TemplateID == "" {
		RespondError(w, r, NewValidationError("template_id is required"))
		return
	}

	// Валидация: хотя бы одна dimension или measure
	if len(params.Dimensions) == 0 && len(params.Measures) == 0 {
		RespondError(w, r, NewValidationError("at least one dimension or measure is required"))
		return
	}

	// Валидация: limit не более 1000
	if params.Limit <= 0 || params.Limit > 1000 {
		params.Limit = 100
	}

	// Выполнение запроса
	result, err := s.queryBuilder.Execute(r.Context(), params)
	if err != nil {
		s.logger.Error("P2-BI: query execution failed",
			"template_id", params.TemplateID,
			"user_id", claims.UserID,
			"error", err,
		)

		// Безопасная обработка ошибок — без раскрытия SQL/schema details
		switch err.(type) {
		case *analytics.ValidationError:
			RespondError(w, r, NewValidationError(err.Error()))
		case *analytics.TimeoutError:
			RespondError(w, r, NewInternalError("query timed out", nil))
		default:
			RespondError(w, r, NewInternalError("query execution failed", nil))
		}
		return
	}

	s.logger.Info("P2-BI: query executed",
		"template_id", params.TemplateID,
		"user_id", claims.UserID,
		"rows", result.Total,
		"took", result.Took,
	)

	jsonResponse(w, http.StatusOK, result)
}
