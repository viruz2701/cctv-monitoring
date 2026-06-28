// Package api — unit tests for Server-Side Validation (P1-SEC.2).
package api

import (
	"gb-telemetry-collector/internal/respond"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// ═══════════════════════════════════════════════════════════════════════════
// Custom validators
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateDeviceType(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid camera", "camera", true},
		{"valid nvr", "nvr", true},
		{"valid dvr", "dvr", true},
		{"valid switch", "switch", true},
		{"invalid type", "printer", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.value, "device_type")
			if tt.valid && err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid, got valid")
			}
		})
	}
}

func TestValidateConnType(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid ip", "ip", true},
		{"valid p2p", "p2p", true},
		{"valid onvif", "onvif", true},
		{"invalid", "bluetooth", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.value, "conn_type")
			if tt.valid && err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid, got valid")
			}
		})
	}
}

func TestValidateWOStatus(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"open", "open", true},
		{"completed", "completed", true},
		{"invalid", "deleted", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.value, "wo_status")
			if tt.valid && err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid, got valid")
			}
		})
	}
}

func TestValidateIPWithPort(t *testing.T) {
	v := GetValidator()

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid ipv4", "192.168.1.1:8080", true},
		{"valid ipv6", "[::1]:443", true},
		{"empty", "", true},
		{"missing port", "192.168.1.1", false},
		{"missing ip", ":8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Var(tt.value, "ip_with_port")
			if tt.valid && err != nil {
				t.Errorf("expected valid, got: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid, got valid")
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetValidator singleton
// ═══════════════════════════════════════════════════════════════════════════

func TestGetValidator_Singleton(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()

	if v1 != v2 {
		t.Error("GetValidator must return the same instance")
	}
}

func TestGetValidator_HasCustomValidators(t *testing.T) {
	v := GetValidator()

	// Проверяем что кастомные валидаторы работают
	customTests := []struct {
		tag   string
		value string
		valid bool
	}{
		{"device_type", "camera", true},
		{"conn_type", "ip", true},
		{"wo_status", "open", true},
		{"wo_priority", "critical", true},
		{"health", "healthy", true},
		{"asset_class", "critical", true},
		{"ip_with_port", "192.168.1.1:8080", true},
	}

	for _, tt := range customTests {
		t.Run(tt.tag, func(t *testing.T) {
			err := v.Var(tt.value, tt.tag)
			if tt.valid && err != nil {
				t.Errorf("expected valid for %s=%s, got: %v", tt.tag, tt.value, err)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Struct validation
// ═══════════════════════════════════════════════════════════════════════════

type testCreateDeviceRequest struct {
	Name       string `json:"name" validate:"required,min=1,max=100"`
	DeviceType string `json:"device_type" validate:"required,device_type"`
	ConnType   string `json:"conn_type" validate:"required,conn_type"`
	IPPort     string `json:"ip_port" validate:"omitempty,ip_with_port"`
	MAC        string `json:"mac" validate:"omitempty,mac"`
}

func TestStructValidation_Valid(t *testing.T) {
	v := GetValidator()

	req := testCreateDeviceRequest{
		Name:       "Camera-1",
		DeviceType: "camera",
		ConnType:   "ip",
		IPPort:     "192.168.1.100:554",
		MAC:        "00:1B:44:11:3A:B7",
	}

	err := v.Struct(req)
	if err != nil {
		t.Errorf("expected valid struct, got: %v", err)
	}
}

func TestStructValidation_Invalid(t *testing.T) {
	v := GetValidator()

	req := testCreateDeviceRequest{
		Name:       "",
		DeviceType: "printer",
		ConnType:   "bluetooth",
	}

	err := v.Struct(req)
	if err == nil {
		t.Fatal("expected validation errors")
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// Проверяем что получили 3 ошибки
	if len(validationErrs) < 3 {
		t.Errorf("expected at least 3 validation errors, got %d: %v", len(validationErrs), validationErrs)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ValidateRequest function
// ═══════════════════════════════════════════════════════════════════════════

func TestValidateRequest_ValidJSON(t *testing.T) {
	body := `{"name":"Camera-1","device_type":"camera","conn_type":"ip"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var req testCreateDeviceRequest
	err := ValidateRequest(r, &req)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if req.Name != "Camera-1" {
		t.Errorf("expected name Camera-1, got %s", req.Name)
	}
}

func TestValidateRequest_InvalidJSON(t *testing.T) {
	body := `{"name":"Camera-1","device_type":"printer","conn_type":"bluetooth"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var req testCreateDeviceRequest
	err := ValidateRequest(r, &req)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRequest_MalformedJSON(t *testing.T) {
	body := `{"name": invalid json}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var req testCreateDeviceRequest
	err := ValidateRequest(r, &req)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestValidateRequest_EmptyBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/devices", nil)
	r.Header.Set("Content-Type", "application/json")

	var req testCreateDeviceRequest
	err := ValidateRequest(r, &req)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestValidateRequest_DisallowUnknownFields(t *testing.T) {
	body := `{"name":"Camera-1","device_type":"camera","conn_type":"ip","unknown_field":"value"}`
	r := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	var req testCreateDeviceRequest
	err := ValidateRequest(r, &req, WithDisallowUnknownFields())
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Error conversion tests
// ═══════════════════════════════════════════════════════════════════════════

func TestConvertValidationErrors_Format(t *testing.T) {
	v := GetValidator()

	req := testCreateDeviceRequest{
		Name:       "",
		DeviceType: "invalid",
		ConnType:   "invalid",
	}

	err := v.Struct(req)
	if err == nil {
		t.Fatal("expected validation error")
	}

	apiErr := convertValidationErrors(err)
	if apiErr == nil {
		t.Fatal("expected APIError")
	}

	// Проверяем что это *respond.APIError
	apiError, ok := apiErr.(*respond.APIError) //nolint:errorlint
	if !ok {
		t.Fatalf("expected *respond.APIError, got %T", apiErr)
	}

	if apiError.Status != 422 {
		t.Errorf("expected status 422, got %d", apiError.Status)
	}

	if apiError.Code != "VALIDATION_ERROR" {
		t.Errorf("expected code VALIDATION_ERROR, got %s", apiError.Code)
	}

	if apiError.Details == nil {
		t.Fatal("expected details to be set")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// toSnakeCase helper
// ═══════════════════════════════════════════════════════════════════════════

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"DeviceType", "device_type"},
		{"ConnType", "conn_type"},
		{"IPAddress", "ip_address"},
		{"MACAddress", "mac_address"},
		{"Name", "name"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
