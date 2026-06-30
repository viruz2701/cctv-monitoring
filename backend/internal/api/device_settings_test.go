// Package api — unit tests for Device Settings HTTP handlers.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend)
//   - OWASP ASVS V5.1: Input validation
//   - OWASP ASVS V3.3: RBAC
//   - OWASP ASVS V7.1: Error handling
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gb-telemetry-collector/internal/auth"

	"github.com/go-chi/chi/v5"
)

// ── Helpers ──────────────────────────────────────────────────────────────

// addAdminContext adds admin role claims to request context for RBAC tests.
func addAdminContext(r *http.Request) *http.Request {
	claims := &auth.Claims{
		UserID: "test-admin",
		Role:   "admin",
	}
	ctx := context.WithValue(r.Context(), auth.UserContextKey, claims)
	return r.WithContext(ctx)
}

// ── Mock DeviceSettingsProvider ──────────────────────────────────────────

type mockSettingsProvider struct {
	getSettingsFunc   func(deviceID, category string) (map[string]interface{}, error)
	setSettingsFunc   func(deviceID string, settings map[string]interface{}) error
	applySettingsFunc func(deviceID string) error
}

func (m *mockSettingsProvider) GetSettings(deviceID, category string) (map[string]interface{}, error) {
	if m.getSettingsFunc != nil {
		return m.getSettingsFunc(deviceID, category)
	}
	return map[string]interface{}{"key": "value"}, nil
}

func (m *mockSettingsProvider) SetSettings(deviceID string, settings map[string]interface{}) error {
	if m.setSettingsFunc != nil {
		return m.setSettingsFunc(deviceID, settings)
	}
	return nil
}

func (m *mockSettingsProvider) ApplySettings(deviceID string) error {
	if m.applySettingsFunc != nil {
		return m.applySettingsFunc(deviceID)
	}
	return nil
}

// ── Tests: validateCategory ──────────────────────────────────────────────

func TestValidateCategory_Valid(t *testing.T) {
	valid := []string{"", "network", "video", "audio", "ptz", "storage", "alarm", "system"}
	for _, c := range valid {
		if !validateCategory(c) {
			t.Errorf("expected category %q to be valid", c)
		}
	}
}

func TestValidateCategory_Invalid(t *testing.T) {
	invalid := []string{"unknown", "security", "user", "  ", "network "}
	for _, c := range invalid {
		if validateCategory(c) {
			t.Errorf("expected category %q to be invalid", c)
		}
	}
}

// ── Tests: validateSettingsUpdate ────────────────────────────────────────

func TestValidateSettingsUpdate_Valid(t *testing.T) {
	req := &updateDeviceSettingsRequest{
		Settings: map[string]interface{}{
			"ip_address": "192.168.1.100",
			"port":       float64(554),
			"enabled":    true,
		},
	}
	err := validateSettingsUpdate(req)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateSettingsUpdate_NilSettings(t *testing.T) {
	req := &updateDeviceSettingsRequest{Settings: nil}
	err := validateSettingsUpdate(req)
	if err == nil {
		t.Error("expected error for nil settings")
	}
}

func TestValidateSettingsUpdate_EmptySettings(t *testing.T) {
	req := &updateDeviceSettingsRequest{Settings: map[string]interface{}{}}
	err := validateSettingsUpdate(req)
	if err != nil {
		t.Errorf("expected no error for empty settings map, got %v", err)
	}
}

func TestValidateSettingsUpdate_EmptyKey(t *testing.T) {
	req := &updateDeviceSettingsRequest{
		Settings: map[string]interface{}{"": "value"},
	}
	err := validateSettingsUpdate(req)
	if err == nil {
		t.Error("expected error for empty key in settings")
	}
}

// ── Tests: handleGetDeviceSettings ───────────────────────────────────────

func TestHandleGetDeviceSettings_Success(t *testing.T) {
	s := &Server{
		deviceSettingsProvider: &mockSettingsProvider{
			getSettingsFunc: func(deviceID, category string) (map[string]interface{}, error) {
				return map[string]interface{}{
					"ip_address": "192.168.1.100",
					"port":       float64(554),
				}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/settings", s.handleGetDeviceSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp getDeviceSettingsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DeviceID != "dev-1" {
		t.Errorf("expected device_id 'dev-1', got %q", resp.DeviceID)
	}
	if resp.Settings["ip_address"] != "192.168.1.100" {
		t.Errorf("expected ip_address in settings, got %v", resp.Settings)
	}
}

func TestHandleGetDeviceSettings_MissingDeviceID(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/settings", s.handleGetDeviceSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices//settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for missing device_id, got %d", w.Code)
	}
}

func TestHandleGetDeviceSettings_InvalidCategory(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/settings", s.handleGetDeviceSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/settings?category=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid category, got %d", w.Code)
	}
}

func TestHandleGetDeviceSettings_NoProvider(t *testing.T) {
	s := &Server{}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/settings", s.handleGetDeviceSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/settings", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 for missing provider, got %d", w.Code)
	}
}

func TestHandleGetDeviceSettings_WithCategory(t *testing.T) {
	var capturedCategory string
	s := &Server{
		deviceSettingsProvider: &mockSettingsProvider{
			getSettingsFunc: func(deviceID, category string) (map[string]interface{}, error) {
				capturedCategory = category
				return map[string]interface{}{"type": category}, nil
			},
		},
	}

	r := chi.NewRouter()
	r.Get("/api/v1/devices/{id}/settings", s.handleGetDeviceSettings)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/dev-1/settings?category=network", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if capturedCategory != "network" {
		t.Errorf("expected category 'network', got %q", capturedCategory)
	}
}

// ── Tests: getDeviceSettingsResponse JSON ────────────────────────────────

func TestGetDeviceSettingsResponse_JSON(t *testing.T) {
	resp := getDeviceSettingsResponse{
		DeviceID:  "dev-1",
		Category:  "network",
		Settings:  map[string]interface{}{"ip": "10.0.0.1"},
		UpdatedAt: "2026-06-30T12:00:00Z",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded["device_id"] != "dev-1" {
		t.Errorf("expected device_id 'dev-1', got %v", decoded["device_id"])
	}
	if decoded["category"] != "network" {
		t.Errorf("expected category 'network', got %v", decoded["category"])
	}
}

// ── Tests: validSettingCategories ────────────────────────────────────────

func TestValidSettingCategories_AllExpected(t *testing.T) {
	expected := []string{"network", "video", "audio", "ptz", "storage", "alarm", "system"}
	for _, cat := range expected {
		if !validSettingCategories[cat] {
			t.Errorf("expected category %q to be in whitelist", cat)
		}
	}
}

func TestValidSettingCategories_NoUnexpected(t *testing.T) {
	unexpected := []string{"", "user", "security", "admin", "password", "ssh"}
	for _, cat := range unexpected {
		if validSettingCategories[cat] {
			t.Errorf("unexpected category %q found in whitelist", cat)
		}
	}
}
