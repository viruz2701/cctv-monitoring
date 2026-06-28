// Package api — Anomaly Detection HTTP Handlers.
//
// P2-AI.4: API handlers for anomaly detection
//   - GET  /api/v1/ai/anomalies — список аномалий
//   - POST /api/v1/ai/anomalies/feed — добавить метрику
//   - POST /api/v1/ai/anomalies/{id}/acknowledge — подтвердить
//   - POST /api/v1/ai/anomalies/{id}/resolve — разрешить
//   - GET  /api/v1/ai/anomalies/stats — статистика
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - IEC 62443 SR 3.3 (Security monitoring)
package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"gb-telemetry-collector/internal/ai"

	"github.com/go-chi/chi/v5"
)

// ─── AnomalyListResponse ──────────────────────────────────────────────────

type anomalyListResponse struct {
	Anomalies []ai.AnomalyResult `json:"anomalies"`
	Meta      anomalyListMeta    `json:"meta"`
}

type anomalyListMeta struct {
	Total int `json:"total"`
	Limit int `json:"limit"`
}

// ─── FeedMetricRequest ────────────────────────────────────────────────────

type feedMetricRequest struct {
	DeviceID   string  `json:"device_id"`
	MetricType string  `json:"metric_type"`
	Value      float64 `json:"value"`
}

// ─── Handlers ─────────────────────────────────────────────────────────────

// handleListAnomalies — GET /api/v1/ai/anomalies
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через query params)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
func (s *Server) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	if s.anomalyService == nil {
		respondJSONError(w, http.StatusServiceUnavailable, "anomaly service not available")
		return
	}

	deviceID := r.URL.Query().Get("device_id")
	metricType := r.URL.Query().Get("metric_type")
	severity := r.URL.Query().Get("severity")
	status := r.URL.Query().Get("status")
	limitStr := r.URL.Query().Get("limit")

	// Whitelist validation (OWASP ASVS V5.1)
	if metricType != "" && !isValidMetricType(metricType) {
		respondJSONError(w, http.StatusBadRequest, "invalid metric_type")
		return
	}
	if severity != "" && !isValidSeverity(severity) {
		respondJSONError(w, http.StatusBadRequest, "invalid severity")
		return
	}
	if status != "" && !isValidAnomalyStatus(status) {
		respondJSONError(w, http.StatusBadRequest, "invalid status")
		return
	}

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		} else {
			respondJSONError(w, http.StatusBadRequest, "invalid limit (1-200)")
			return
		}
	}

	anomalies := s.anomalyService.GetAllAnomalies(deviceID, metricType, severity, status, limit)

	resp := anomalyListResponse{
		Anomalies: anomalies,
		Meta: anomalyListMeta{
			Total: len(anomalies),
			Limit: limit,
		},
	}

	respondJSON(w, http.StatusOK, resp)
}

// handleFeedMetric — POST /api/v1/ai/anomalies/feed
//
// Добавляет метрику устройства для анализа аномалий.
func (s *Server) handleFeedMetric(w http.ResponseWriter, r *http.Request) {
	if s.anomalyService == nil {
		respondJSONError(w, http.StatusServiceUnavailable, "anomaly service not available")
		return
	}

	var req feedMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Input validation (OWASP ASVS V5.1)
	if req.DeviceID == "" {
		respondJSONError(w, http.StatusBadRequest, "device_id is required")
		return
	}
	if req.MetricType == "" {
		respondJSONError(w, http.StatusBadRequest, "metric_type is required")
		return
	}
	if !isValidMetricType(req.MetricType) {
		respondJSONError(w, http.StatusBadRequest, "invalid metric_type")
		return
	}

	ctx := r.Context()
	metric := ai.DeviceMetricPoint{
		DeviceID:   req.DeviceID,
		MetricType: req.MetricType,
		Value:      req.Value,
		Timestamp:  time.Now().UTC(),
	}

	anomaly := s.anomalyService.FeedMetric(ctx, metric)

	if anomaly != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "anomaly_detected",
			"anomaly": anomaly,
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAcknowledgeAnomaly — POST /api/v1/ai/anomalies/{id}/acknowledge
func (s *Server) handleAcknowledgeAnomaly(w http.ResponseWriter, r *http.Request) {
	if s.anomalyService == nil {
		respondJSONError(w, http.StatusServiceUnavailable, "anomaly service not available")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, http.StatusBadRequest, "anomaly id is required")
		return
	}

	if err := s.anomalyService.AcknowledgeAnomaly(id); err != nil {
		respondJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}

// handleResolveAnomaly — POST /api/v1/ai/anomalies/{id}/resolve
func (s *Server) handleResolveAnomaly(w http.ResponseWriter, r *http.Request) {
	if s.anomalyService == nil {
		respondJSONError(w, http.StatusServiceUnavailable, "anomaly service not available")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		respondJSONError(w, http.StatusBadRequest, "anomaly id is required")
		return
	}

	if err := s.anomalyService.ResolveAnomaly(id); err != nil {
		respondJSONError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

// handleAnomalyStats — GET /api/v1/ai/anomalies/stats
func (s *Server) handleAnomalyStats(w http.ResponseWriter, r *http.Request) {
	if s.anomalyService == nil {
		respondJSONError(w, http.StatusServiceUnavailable, "anomaly service not available")
		return
	}

	stats := s.anomalyService.Health()
	respondJSON(w, http.StatusOK, stats)
}

// ─── Validation Helpers ───────────────────────────────────────────────────

// isValidMetricType проверяет допустимый тип метрики (whitelist).
func isValidMetricType(t string) bool {
	for _, valid := range ai.ValidMetricTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// isValidSeverity проверяет допустимый уровень серьёзности.
func isValidSeverity(s string) bool {
	for _, valid := range ai.ValidSeverities {
		if s == valid {
			return true
		}
	}
	return false
}

// isValidAnomalyStatus проверяет допустимый статус.
func isValidAnomalyStatus(s string) bool {
	for _, valid := range ai.ValidAnomalyStatuses {
		if s == valid {
			return true
		}
	}
	return false
}

// ─── JSON Response helpers ────────────────────────────────────────────────

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func respondJSONError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"code":    fmt.Sprintf("ERR_%d", status),
		},
	})
}
