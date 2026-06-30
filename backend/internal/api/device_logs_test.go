// Package api — unit tests for Device Logs HTTP handlers.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend)
//   - OWASP ASVS V5.1: Input validation
//   - OWASP ASVS V7.1: Error handling
//   - ISO 27001 A.12.6.1: Capacity management (pagination limits)
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// ── Mock DeviceLogProvider ───────────────────────────────────────────────

type mockLogProvider struct {
	getLogsFunc func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error)
}

func (m *mockLogProvider) GetLogs(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
	if m.getLogsFunc != nil {
		return m.getLogsFunc(deviceID, since, until, limit, offset)
	}
	return []DeviceLogEntry{}, nil
}

// ── Tests: handleGetDeviceLogs ───────────────────────────────────────────

func TestHandleGetDeviceLogs_Success(t *testing.T) {
	logs := []DeviceLogEntry{
		{Timestamp: time.Now(), Level: "info", Source: "kernel", Message: "Device initialized"},
		{Timestamp: time.Now(), Level: "warn", Source: "app", Message: "CPU temperature high"},
	}

	s := &Server{
		deviceLogProvider: &mockLogProvider{
			getLogsFunc: func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
				return logs, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp getDeviceLogsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DeviceID != "dev-1" {
		t.Errorf("expected device_id 'dev-1', got %q", resp.DeviceID)
	}
	if len(resp.Logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(resp.Logs))
	}
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
}

func TestHandleGetDeviceLogs_MissingDeviceID(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices//logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_WithLimit(t *testing.T) {
	var capturedLimit int
	s := &Server{
		deviceLogProvider: &mockLogProvider{
			getLogsFunc: func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
				capturedLimit = limit
				return []DeviceLogEntry{}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?limit=50", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if capturedLimit != 50 {
		t.Errorf("expected limit 50, got %d", capturedLimit)
	}
}

func TestHandleGetDeviceLogs_InvalidLimit(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?limit=-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for negative limit, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_ExcessiveLimit(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?limit=9999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for excessive limit, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_WithOffset(t *testing.T) {
	var capturedOffset int
	s := &Server{
		deviceLogProvider: &mockLogProvider{
			getLogsFunc: func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
				capturedOffset = offset
				return []DeviceLogEntry{}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?offset=100", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if capturedOffset != 100 {
		t.Errorf("expected offset 100, got %d", capturedOffset)
	}
}

func TestHandleGetDeviceLogs_NegativeOffset(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?offset=-5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for negative offset, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_ExcessiveOffset(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?offset=99999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for excessive offset, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_DefaultValues(t *testing.T) {
	var capturedLimit, capturedOffset int
	s := &Server{
		deviceLogProvider: &mockLogProvider{
			getLogsFunc: func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
				capturedLimit = limit
				capturedOffset = offset
				return []DeviceLogEntry{}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if capturedLimit != 100 {
		t.Errorf("expected default limit 100, got %d", capturedLimit)
	}
	if capturedOffset != 0 {
		t.Errorf("expected default offset 0, got %d", capturedOffset)
	}
}

func TestHandleGetDeviceLogs_NoProvider(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_WithSince(t *testing.T) {
	var capturedSince time.Time
	s := &Server{
		deviceLogProvider: &mockLogProvider{
			getLogsFunc: func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
				capturedSince = since
				return []DeviceLogEntry{}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?since=2026-06-01T00:00:00Z", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if capturedSince.IsZero() {
		t.Error("expected non-zero since time")
	}
}

func TestHandleGetDeviceLogs_InvalidSince(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?since=not-a-date", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid since, got %d", w.Code)
	}
}

func TestHandleGetDeviceLogs_WithUntil(t *testing.T) {
	var capturedUntil time.Time
	s := &Server{
		deviceLogProvider: &mockLogProvider{
			getLogsFunc: func(deviceID string, since, until time.Time, limit, offset int) ([]DeviceLogEntry, error) {
				capturedUntil = until
				return []DeviceLogEntry{}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?until=2026-07-01T00:00:00Z", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if capturedUntil.IsZero() {
		t.Error("expected non-zero until time")
	}
}

func TestHandleGetDeviceLogs_InvalidUntil(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/logs?until=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid until, got %d", w.Code)
	}
}

// ── Tests: formatTimePtr ─────────────────────────────────────────────────

func TestFormatTimePtr_Zero(t *testing.T) {
	result := formatTimePtr(time.Time{})
	if result != "" {
		t.Errorf("expected empty string for zero time, got %q", result)
	}
}

func TestFormatTimePtr_NonZero(t *testing.T) {
	ts := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	result := formatTimePtr(ts)
	expected := "2026-06-30T12:00:00Z"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// ── Tests: DeviceLogEntry ────────────────────────────────────────────────

func TestDeviceLogEntry_JSON(t *testing.T) {
	entry := DeviceLogEntry{
		Timestamp: time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC),
		Level:     "error",
		Source:    "kernel",
		Message:   "Disk failure imminent",
		Metadata:  map[string]interface{}{"disk_id": "sda1"},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded DeviceLogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Level != "error" {
		t.Errorf("expected level 'error', got %q", decoded.Level)
	}
	if decoded.Message != "Disk failure imminent" {
		t.Errorf("expected message, got %q", decoded.Message)
	}
}
