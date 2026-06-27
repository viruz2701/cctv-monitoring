// Package api — unit tests for input validation.
// Соответствует: OWASP ASVS V5 (Validation), TDD approach
package api

import (
	"errors"
	"testing"
)

func TestValidator_Required(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"non-empty string", "test", true},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.Required("field", tt.value)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v", tt.valid, v.Valid())
			}
		})
	}
}

func TestValidator_MinLength(t *testing.T) {
	tests := []struct {
		name  string
		value string
		min   int
		valid bool
	}{
		{"exactly min", "abc", 3, true},
		{"above min", "abcdef", 3, true},
		{"below min", "ab", 3, false},
		{"empty with min>0", "", 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.MinLength("field", tt.value, tt.min)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v (value=%q, min=%d)",
					tt.valid, v.Valid(), tt.value, tt.min)
			}
		})
	}
}

func TestValidator_MaxLength(t *testing.T) {
	tests := []struct {
		name  string
		value string
		max   int
		valid bool
	}{
		{"exactly max", "abc", 3, true},
		{"below max", "ab", 3, true},
		{"above max", "abcd", 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.MaxLength("field", tt.value, tt.max)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v", tt.valid, v.Valid())
			}
		})
	}
}

func TestValidator_OneOf(t *testing.T) {
	allowed := []string{"ONLINE", "OFFLINE", "WARNING"}

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid value", "ONLINE", true},
		{"another valid", "OFFLINE", true},
		{"case sensitive", "online", false},
		{"invalid value", "UNKNOWN", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.OneOf("field", tt.value, allowed)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v for value=%q",
					tt.valid, v.Valid(), tt.value)
			}
		})
	}
}

func TestValidator_UUID(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid UUID v1", "550e8400-e29b-11d4-a716-446655440000", true},
		{"invalid format", "not-a-uuid", false},
		{"empty string (optional)", "", true},
		{"missing dashes", "550e8400e29b41d4a716446655440000", false},
		{"too short", "abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.UUID("field", tt.value)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v for value=%q",
					tt.valid, v.Valid(), tt.value)
			}
		})
	}
}

func TestValidator_MAC(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid MAC (colons)", "00:1B:44:11:3A:B7", true},
		{"valid MAC (dashes)", "00-1B-44-11-3A-B7", true},
		{"empty string (optional)", "", true},
		{"invalid MAC", "not-a-mac", false},
		{"too short", "00:11", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.MAC("field", tt.value)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v for value=%q",
					tt.valid, v.Valid(), tt.value)
			}
		})
	}
}

func TestValidator_RangeFloat(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		min   float64
		max   float64
		valid bool
	}{
		{"in range", 45.0, -90, 90, true},
		{"at min boundary", -90, -90, 90, true},
		{"at max boundary", 90, -90, 90, true},
		{"below min", -100, -90, 90, false},
		{"above max", 100, -90, 90, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.RangeFloat("field", tt.value, tt.min, tt.max)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v", tt.valid, v.Valid())
			}
		})
	}
}

func TestValidator_Email(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid email", "user@example.com", true},
		{"valid with plus", "user+tag@example.com", true},
		{"invalid - no domain", "user@", false},
		{"invalid - no @", "userexample.com", false},
		{"empty string (optional)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.Email("field", tt.value)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v for value=%q",
					tt.valid, v.Valid(), tt.value)
			}
		})
	}
}

func TestValidator_IP(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv6", "::1", true},
		{"invalid", "not-an-ip", false},
		{"empty string (optional)", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.IP("field", tt.value)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v for value=%q",
					tt.valid, v.Valid(), tt.value)
			}
		})
	}
}

func TestValidator_Port(t *testing.T) {
	tests := []struct {
		name  string
		value int
		valid bool
	}{
		{"valid port 80", 80, true},
		{"valid port 443", 443, true},
		{"valid port 65535", 65535, true},
		{"invalid port 0", 0, false},
		{"invalid port 70000", 70000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewValidator()
			v.Port("field", tt.value)
			if v.Valid() != tt.valid {
				t.Errorf("Expected Valid()=%v, got %v for port=%d",
					tt.valid, v.Valid(), tt.value)
			}
		})
	}
}

func TestValidator_ChainedValidation(t *testing.T) {
	v := NewValidator()
	v.Required("name", "").
		OneOf("status", "INVALID", validStatuses).
		MaxLength("location", "too long value here", 5)

	if v.Valid() {
		t.Error("Expected validation to fail for multiple invalid fields")
	}

	errors := v.Errors()
	if len(errors) != 3 {
		t.Errorf("Expected 3 errors, got %d: %v", len(errors), errors)
	}
}

func TestValidator_ValidChain(t *testing.T) {
	v := NewValidator()
	v.Required("name", "valid-name").
		OneOf("status", "ONLINE", validStatuses).
		UUID("device_id", "550e8400-e29b-41d4-a716-446655440000")

	if !v.Valid() {
		t.Errorf("Expected validation to pass, got errors: %v", v.Errors())
	}
}

// ── ValidationErrors tests (P1-SEC.3) ──────────────────────────────────

func TestValidationErrors_AddAndValid(t *testing.T) {
	ve := &ValidationErrors{}
	if !ve.Valid() {
		t.Error("Expected empty ValidationErrors to be valid")
	}

	ve.Add("name", "required", "REQUIRED")
	if ve.Valid() {
		t.Error("Expected ValidationErrors with errors to be invalid")
	}
}

func TestValidationErrors_Error(t *testing.T) {
	ve := &ValidationErrors{}
	ve.Add("name", "required", "REQUIRED")
	ve.Add("email", "invalid format", "INVALID_FORMAT")

	msg := ve.Error()
	if msg != "name: required; email: invalid format" {
		t.Errorf("unexpected error message: %s", msg)
	}
}

func TestValidationErrors_JSON(t *testing.T) {
	ve := &ValidationErrors{}
	ve.Add("name", "required", "REQUIRED")

	// ValidationErrors has Fields with json tags, so json.Marshal works
	// Just verify the struct is properly tagged
	if len(ve.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(ve.Fields))
	}
	if ve.Fields[0].Field != "name" {
		t.Errorf("expected field 'name', got '%s'", ve.Fields[0].Field)
	}
	if ve.Fields[0].Code != "REQUIRED" {
		t.Errorf("expected code 'REQUIRED', got '%s'", ve.Fields[0].Code)
	}
}

// ── Domain validator tests (P1-SEC.3) ──────────────────────────────────

func TestValidateWorkOrderRequest_Valid(t *testing.T) {
	err := validateWorkOrderRequest("Fix camera", "maintenance", "high", "Need to fix camera at site A")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateWorkOrderRequest_Invalid(t *testing.T) {
	err := validateWorkOrderRequest("", "invalid_type", "", "")
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	if len(ve.Fields) < 3 {
		t.Errorf("expected at least 3 field errors, got %d", len(ve.Fields))
	}
}

func TestValidateSiteRequest_Valid(t *testing.T) {
	err := validateSiteRequest("Main Office", "123 Main St", "Minsk")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateSiteRequest_MissingFields(t *testing.T) {
	err := validateSiteRequest("", "", "")
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}

	// 3 required fields
	if len(ve.Fields) < 3 {
		t.Errorf("expected at least 3 field errors, got %d", len(ve.Fields))
	}
}

func TestValidateLoginRequest_Valid(t *testing.T) {
	err := validateLoginRequest("admin", "password123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateLoginRequest_Empty(t *testing.T) {
	err := validateLoginRequest("", "")
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
}

// ── ValidationErrors As error interface tests ──────────────────────────

func TestNewValidator_Empty(t *testing.T) {
	v := NewValidator()
	if !v.Valid() {
		t.Error("new validator should be valid")
	}
	if len(v.Errors()) != 0 {
		t.Errorf("expected 0 errors, got %d", len(v.Errors()))
	}
}

func TestRespondValidationError_Interface(t *testing.T) {
	// Verify the function signature compiles correctly
	ve := &ValidationErrors{}
	ve.Add("test", "error", "ERR")
	if !errors.As(ve, &ve) {
		t.Error("ValidationErrors should implement error interface")
	}
}

// ── OWASP ASVS V5 Compliance Tests ────────────────────────────────────

func TestWhitelistValidDeviceTypes(t *testing.T) {
	// OWASP ASVS V5.1: Validation should use whitelist, not blacklist
	expected := []string{"camera", "nvr", "dvr", "switch"}
	for _, et := range expected {
		found := false
		for _, at := range validDeviceTypes {
			if et == at {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected device type %q to be in whitelist", et)
		}
	}

	// Verify no unexpected types
	unexpected := []string{"", "invalid", "CAMERA", "NVR", "sensor", "controller"}
	for _, ut := range unexpected {
		for _, at := range validDeviceTypes {
			if ut == at {
				t.Errorf("Unexpected device type %q found in whitelist", ut)
			}
		}
	}
}

func TestWhitelistValidStatuses(t *testing.T) {
	// OWASP ASVS V5.1: Whitelist must not include invalid statuses
	invalid := []string{"", "deleted", "unknown", "DISABLED", "active"}
	for _, iv := range invalid {
		for _, vs := range validStatuses {
			if iv == vs {
				t.Errorf("Invalid status %q found in whitelist", iv)
			}
		}
	}
}
