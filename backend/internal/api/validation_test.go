// Package api — unit tests for input validation.
// Соответствует: OWASP ASVS V5 (Validation), TDD approach
package api

import (
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
