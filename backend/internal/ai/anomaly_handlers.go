// Package ai — Anomaly Detection HTTP Handlers.
//
// P2-AI.4: API endpoint for anomaly detection
//   - GET  /api/v1/ai/anomalies — список аномалий
//   - POST /api/v1/ai/anomalies/:id/acknowledge — подтвердить
//   - POST /api/v1/ai/anomalies/:id/resolve — разрешить
//   - POST /api/v1/ai/anomalies/feed — добавить метрику (для агентов)
//   - GET  /api/v1/ai/anomalies/stats — статистика детектора
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// ─── HTTP Types ───────────────────────────────────────────────────────────

// AnomalyListResponse — ответ со списком аномалий.
type AnomalyListResponse struct {
	Anomalies []AnomalyResult `json:"anomalies"`
	Meta      AnomalyListMeta `json:"meta"`
}

// AnomalyListMeta — метаданные списка.
type AnomalyListMeta struct {
	Total int `json:"total"`
	Limit int `json:"limit"`
}

// FeedMetricRequest — запрос на добавление метрики.
type FeedMetricRequest struct {
	DeviceID   string  `json:"device_id"`
	MetricType string  `json:"metric_type"`
	Value      float64 `json:"value"`
}

// ─── RegisterAnomalyRoutes ────────────────────────────────────────────────

// RegisterAnomalyRoutes регистрирует маршруты аномалий на роутере.
func RegisterAnomalyRoutes(r chi.Router, service *AnomalyService, logger interface{}) {
	r.Get("/api/v1/ai/anomalies", handleListAnomalies(service))
	r.Post("/api/v1/ai/anomalies/feed", handleFeedMetric(service, logger))
	r.Post("/api/v1/ai/anomalies/{id}/acknowledge", handleAcknowledgeAnomaly(service))
	r.Post("/api/v1/ai/anomalies/{id}/resolve", handleResolveAnomaly(service))
	r.Get("/api/v1/ai/anomalies/stats", handleAnomalyStats(service))
}

// ─── Handlers ─────────────────────────────────────────────────────────────

// handleListAnomalies — GET /api/v1/ai/anomalies
//
// Query params:
//   - device_id (optional): фильтр по устройству
//   - metric_type (optional): фильтр по типу метрики
//   - severity (optional): фильтр по серьёзности
//   - status (optional): фильтр по статусу
//   - limit (optional, default 50): лимит записей
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через query params)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
func handleListAnomalies(service *AnomalyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := r.URL.Query().Get("device_id")
		metricType := r.URL.Query().Get("metric_type")
		severity := r.URL.Query().Get("severity")
		status := r.URL.Query().Get("status")
		limitStr := r.URL.Query().Get("limit")

		// Валидация metric_type
		if metricType != "" && !isValidMetricType(metricType) {
			writeJSONError(w, http.StatusBadRequest, "invalid metric_type: "+metricType)
			return
		}

		// Валидация severity
		if severity != "" && !isValidSeverity(severity) {
			writeJSONError(w, http.StatusBadRequest, "invalid severity: "+severity)
			return
		}

		// Валидация status
		if status != "" && !isValidAnomalyStatus(status) {
			writeJSONError(w, http.StatusBadRequest, "invalid status: "+status)
			return
		}

		// Лимит
		limit := 50
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
				limit = l
			} else {
				writeJSONError(w, http.StatusBadRequest, "invalid limit (1-200)")
				return
			}
		}

		anomalies := service.GetAllAnomalies(deviceID, metricType, severity, status, limit)

		resp := AnomalyListResponse{
			Anomalies: anomalies,
			Meta: AnomalyListMeta{
				Total: len(anomalies),
				Limit: limit,
			},
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

// handleFeedMetric — POST /api/v1/ai/anomalies/feed
//
// Добавляет метрику устройства для анализа.
// Если метрика является аномалией, возвращает её.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — JSON schema validation)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - IEC 62443 SR 3.3 (Security monitoring)
func handleFeedMetric(service *AnomalyService, logger interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req FeedMetricRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		// Валидация
		if req.DeviceID == "" {
			writeJSONError(w, http.StatusBadRequest, "device_id is required")
			return
		}
		if req.MetricType == "" {
			writeJSONError(w, http.StatusBadRequest, "metric_type is required")
			return
		}
		if !isValidMetricType(req.MetricType) {
			writeJSONError(w, http.StatusBadRequest, "invalid metric_type: "+req.MetricType)
			return
		}

		ctx := r.Context()
		metric := DeviceMetricPoint{
			DeviceID:   req.DeviceID,
			MetricType: req.MetricType,
			Value:      req.Value,
			Timestamp:  time.Now().UTC(),
		}

		anomaly := service.FeedMetric(ctx, metric)

		if anomaly != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":  "anomaly_detected",
				"anomaly": anomaly,
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	}
}

// handleAcknowledgeAnomaly — POST /api/v1/ai/anomalies/{id}/acknowledge
func handleAcknowledgeAnomaly(service *AnomalyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "anomaly id is required")
			return
		}

		if err := service.AcknowledgeAnomaly(id); err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "acknowledged",
		})
	}
}

// handleResolveAnomaly — POST /api/v1/ai/anomalies/{id}/resolve
func handleResolveAnomaly(service *AnomalyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			writeJSONError(w, http.StatusBadRequest, "anomaly id is required")
			return
		}

		if err := service.ResolveAnomaly(id); err != nil {
			writeJSONError(w, http.StatusNotFound, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "resolved",
		})
	}
}

// handleAnomalyStats — GET /api/v1/ai/anomalies/stats
//
// Возвращает статистику детектора аномалий.
func handleAnomalyStats(service *AnomalyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := service.Health()
		writeJSON(w, http.StatusOK, stats)
	}
}

// ─── Validation helpers ───────────────────────────────────────────────────

func isValidMetricType(t string) bool {
	for _, valid := range ValidMetricTypes {
		if t == valid {
			return true
		}
	}
	return false
}

func isValidSeverity(s string) bool {
	for _, valid := range ValidSeverities {
		if s == valid {
			return true
		}
	}
	return false
}

func isValidAnomalyStatus(s string) bool {
	for _, valid := range ValidAnomalyStatuses {
		if s == valid {
			return true
		}
	}
	return false
}

// ─── Response helpers ─────────────────────────────────────────────────────

// writeJSON отправляет JSON-ответ.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// writeJSONError отправляет JSON-ответ с ошибкой.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"code":    fmt.Sprintf("ERR_%d", status),
		},
	})
}

// ─── Ensure slog import ───────────────────────────────────────────────────

var _ = slog.String

// ContextKey для traceID.
type contextKey string

const traceIDKey contextKey = "trace_id"

// WithTraceIDContext добавляет traceID в контекст.
func WithTraceIDContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID извлекает traceID из контекста.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok {
		return id
	}
	if id, ok := ctx.Value("trace_id").(string); ok {
		return id
	}
	return "unknown"
}

// Ensure imports are used
var _ = strings.TrimSpace
