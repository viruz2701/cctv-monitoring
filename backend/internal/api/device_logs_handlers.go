// Package api — Device Logs HTTP handlers.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-EDGE Block 6 — API-02: Device Logs Endpoints
//
// Endpoints:
//   GET /api/v1/devices/{id}/logs?since=2026-06-01T00:00:00Z&limit=100
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend), SR 3.1 (Resource management)
//   - OWASP ASVS V5.1: Input validation (whitelist — query params)
//   - OWASP ASVS V7.1: Error handling (no information leakage)
//   - ISO 27001 A.12.4.1: Event logging
//   - ISO 27001 A.12.6.1: Capacity management (pagination limits)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/respond"
)

// ────────────────────────────────────────────────────────────────────────────
// DeviceLogProvider — интерфейс для получения логов устройства
// через VendorDevice.GetLogs().
// ────────────────────────────────────────────────────────────────────────────

// DeviceLogProvider определяет контракт для получения логов устройства.
// Реализуется через DeviceFactory, который создаёт VendorDevice для конкретного вендора.
type DeviceLogProvider interface {
	// GetLogs возвращает логи устройства с фильтрацией по времени и пагинацией.
	// Параметры:
	//   - deviceID: ID устройства
	//   - since: начало периода (RFC3339)
	//   - until: конец периода (RFC3339, empty = now)
	//   - limit: максимальное количество записей (1-1000)
	//   - offset: смещение для пагинации
	GetLogs(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error)
}

// DeviceLogEntry — одна запись лога устройства.
type DeviceLogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Source      string                 `json:"source,omitempty"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Request/Response DTOs
// ────────────────────────────────────────────────────────────────────────────

// getDeviceLogsResponse — ответ для GET logs.
type getDeviceLogsResponse struct {
	DeviceID string           `json:"device_id"`
	Logs     []DeviceLogEntry `json:"logs"`
	Total    int              `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
	Since    string           `json:"since,omitempty"`
	Until    string           `json:"until,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// maxLogLimit — максимальный лимит записей лога за один запрос.
	// Соответствует: ISO 27001 A.12.6.1 (Capacity management)
	maxLogLimit = 1000

	// defaultLogLimit — лимит по умолчанию.
	defaultLogLimit = 100

	// maxLogOffset — максимальное смещение для пагинации (защита от deep pagination).
	maxLogOffset = 10000
)

// ────────────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────────────

// handleGetDeviceLogs возвращает логи устройства (GET).
//
// Query parameters:
//   - since (optional, RFC3339): начало периода
//   - until (optional, RFC3339): конец периода (по умолчанию now)
//   - limit (optional, 1-1000): количество записей (по умолчанию 100)
//   - offset (optional, 0-10000): смещение для пагинации
//
// Compliance:
//   - OWASP ASVS V5.1: Input validation (query parameters)
//   - IEC 62443-3-3 SR 3.1: Resource management (pagination limits)
//   - ISO 27001 A.12.6.1: Capacity management
func (s *Server) handleGetDeviceLogs(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respond.RespondError(w, r, respond.NewBadRequestError("device_id is required"))
		return
	}

	// ── Parse query parameters ─────────────────────────────────────────

	// since (optional)
	var since time.Time
	sinceStr := r.URL.Query().Get("since")
	if sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			respond.RespondError(w, r, respond.NewValidationError(
				"invalid 'since' format, expected RFC3339 (e.g., 2026-06-01T00:00:00Z)"))
			return
		}
	}

	// until (optional, default = now)
	until := time.Now().UTC()
	untilStr := r.URL.Query().Get("until")
	if untilStr != "" {
		var err error
		until, err = time.Parse(time.RFC3339, untilStr)
		if err != nil {
			respond.RespondError(w, r, respond.NewValidationError(
				"invalid 'until' format, expected RFC3339 (e.g., 2026-06-01T00:00:00Z)"))
			return
		}
	}

	// Если указан until, проверяем что он не в будущем
	if until.After(time.Now().UTC()) {
		until = time.Now().UTC()
	}

	// Если указаны both since и until, проверяем что since < until
	if !since.IsZero() && since.After(until) {
		respond.RespondError(w, r, respond.NewValidationError("'since' must be before 'until'"))
		return
	}

	// limit (optional, 1-1000, default 100)
	limit := defaultLogLimit
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err != nil || parsed < 1 || parsed > maxLogLimit {
			respond.RespondError(w, r, respond.NewValidationError(
				fmt.Sprintf("'limit' must be between 1 and %d", maxLogLimit)))
			return
		}
		limit = parsed
	}

	// offset (optional, 0-10000, default 0)
	offset := 0
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		parsed, err := strconv.Atoi(offsetStr)
		if err != nil || parsed < 0 || parsed > maxLogOffset {
			respond.RespondError(w, r, respond.NewValidationError(
				fmt.Sprintf("'offset' must be between 0 and %d", maxLogOffset)))
			return
		}
		offset = parsed
	}

	// ── Get logs ────────────────────────────────────────────────────────
	if s.deviceLogProvider == nil {
		respond.RespondError(w, r, respond.NewInternalError("device log provider not available", nil))
		return
	}

	logs, err := s.deviceLogProvider.GetLogs(deviceID, since, until, limit, offset)
	if err != nil {
		respond.RespondError(w, r, respond.NewInternalError("failed to get device logs", err))
		return
	}

	jsonResponse(w, http.StatusOK, getDeviceLogsResponse{
		DeviceID: deviceID,
		Logs:     logs,
		Total:    len(logs),
		Limit:    limit,
		Offset:   offset,
		Since:    formatTimePtr(since),
		Until:    formatTimePtr(until),
	})
}

// formatTimePtr форматирует time.Time в RFC3339 или возвращает пустую строку для zero value.
func formatTimePtr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
