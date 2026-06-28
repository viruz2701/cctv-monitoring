// Package api — CORS middleware tests.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-SEC.2: CORS Wildcard Fix — Unit Tests
//
// Coverage:
//   - ValidateCORSOrigins: wildcard, empty, valid origins
//   - NewCORSHandler: debug vs production
//   - isLocalhostOrigin: edge cases
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"testing"

	apimw "gb-telemetry-collector/internal/api/middleware"
)

func TestValidateCORSOrigins_RejectsWildcard(t *testing.T) {
	t.Parallel()

	err := apimw.ValidateCORSOrigins([]string{"*"}, false)
	if err == nil {
		t.Fatal("expected error for wildcard origin, got nil")
	}
	if err.Error() != "CORS: wildcard origin '*' detected — ЗАПРЕЩЕНО для production (OWASP ASVS V9.1, ISO 27001 A.13.2)" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateCORSOrigins_RejectsWildcardInList(t *testing.T) {
	t.Parallel()

	err := apimw.ValidateCORSOrigins([]string{"https://example.com", "*", "http://localhost:3000"}, false)
	if err == nil {
		t.Fatal("expected error when wildcard is in the list")
	}
}

func TestValidateCORSOrigins_RejectsEmptyProduction(t *testing.T) {
	t.Parallel()

	err := apimw.ValidateCORSOrigins([]string{}, false)
	if err == nil {
		t.Fatal("expected error for empty origins in production")
	}
	if err.Error() != "CORS: cors_allowed_origins is empty — требуется явная конфигурация для production (OWASP ASVS V13.4)" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateCORSOrigins_AllowsEmptyDebug(t *testing.T) {
	t.Parallel()

	err := apimw.ValidateCORSOrigins([]string{}, true)
	if err != nil {
		t.Fatalf("expected no error for empty origins in debug mode: %v", err)
	}
}

func TestValidateCORSOrigins_AllowsValidOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origins []string
	}{
		{"single origin", []string{"https://app.example.com"}},
		{"multiple origins", []string{"https://app.example.com", "https://admin.example.com"}},
		{"localhost dev", []string{"http://localhost:5173"}},
		{"localhost all", []string{"http://localhost:3000", "http://localhost:5173", "http://localhost:8080"}},
		{"production domains", []string{"https://cctv.example.com", "https://api.cctv.example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := apimw.ValidateCORSOrigins(tt.origins, false)
			if err != nil {
				t.Fatalf("unexpected error for valid origins %v: %v", tt.origins, err)
			}
		})
	}
}

func TestValidateCORSOrigins_RejectsWildcardInMixedList(t *testing.T) {
	t.Parallel()

	err := apimw.ValidateCORSOrigins([]string{"https://valid.com", "*", "https://other.com"}, true)
	if err == nil {
		t.Fatal("expected error even in debug mode for wildcard")
	}
}

func TestNewCORSHandler_ProductionWithExplicitOrigins(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{"https://app.example.com"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(opts.AllowedOrigins) != 1 {
		t.Fatalf("expected 1 origin, got %d: %v", len(opts.AllowedOrigins), opts.AllowedOrigins)
	}
	if opts.AllowedOrigins[0] != "https://app.example.com" {
		t.Fatalf("expected https://app.example.com, got %s", opts.AllowedOrigins[0])
	}
}

func TestNewCORSHandler_ProductionFailsOnEmpty(t *testing.T) {
	t.Parallel()

	_, err := apimw.NewCORSHandler([]string{}, false)
	if err == nil {
		t.Fatal("expected error for production with empty origins")
	}
}

func TestNewCORSHandler_ProductionFailsOnWildcard(t *testing.T) {
	t.Parallel()

	_, err := apimw.NewCORSHandler([]string{"*"}, false)
	if err == nil {
		t.Fatal("expected error for production with wildcard")
	}
}

func TestNewCORSHandler_DebugWithEmptyOrigins(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{}, true)
	if err != nil {
		t.Fatalf("unexpected error in debug mode: %v", err)
	}

	if len(opts.AllowedOrigins) == 0 {
		t.Fatal("expected default origins in debug mode")
	}

	// Проверяем, что все дефолтные origins — localhost
	for _, origin := range opts.AllowedOrigins {
		if !apimw.IsLocalhostOrigin(origin) {
			t.Fatalf("expected localhost origin, got %s", origin)
		}
	}
}

func TestNewCORSHandler_DebugWithCustomOrigins(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{"https://dev.example.com"}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(opts.AllowedOrigins) != 1 {
		t.Fatalf("expected 1 origin, got %d", len(opts.AllowedOrigins))
	}
	if opts.AllowedOrigins[0] != "https://dev.example.com" {
		t.Fatalf("expected https://dev.example.com, got %s", opts.AllowedOrigins[0])
	}
}

func TestIsLocalhostOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		origin   string
		expected bool
	}{
		{"http://localhost:3000", true},
		{"http://localhost:5173", true},
		{"https://localhost:443", true},
		{"http://127.0.0.1:8080", true},
		{"https://127.0.0.1:8443", true},
		{"http://localhost", true},
		{"https://app.example.com", false},
		{"http://192.168.1.1:3000", false},
		{"https://staging.example.com", false},
		{"http://10.0.0.1:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			result := apimw.IsLocalhostOrigin(tt.origin)
			if result != tt.expected {
				t.Fatalf("IsLocalhostOrigin(%q) = %v, want %v", tt.origin, result, tt.expected)
			}
		})
	}
}

func TestNewCORSHandler_AllowCredentials(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{"https://app.example.com"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !opts.AllowCredentials {
		t.Fatal("expected AllowCredentials to be true (required for HttpOnly cookies)")
	}
}

func TestNewCORSHandler_AllowedMethods(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{"https://app.example.com"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMethods := map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "OPTIONS": true, "PATCH": true}
	for _, m := range opts.AllowedMethods {
		if !expectedMethods[m] {
			t.Fatalf("unexpected method: %s", m)
		}
	}
}

func TestNewCORSHandler_CSRFHeaderAllowed(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{"https://app.example.com"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foundXCSRF := false
	for _, h := range opts.AllowedHeaders {
		if h == "X-CSRF-Token" {
			foundXCSRF = true
			break
		}
	}
	if !foundXCSRF {
		t.Fatal("expected X-CSRF-Token in AllowedHeaders (required for CSRF protection)")
	}
}

func TestNewCORSHandler_MaxAge(t *testing.T) {
	t.Parallel()

	opts, err := apimw.NewCORSHandler([]string{"https://app.example.com"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.MaxAge != 300 {
		t.Fatalf("expected MaxAge 300, got %d", opts.MaxAge)
	}
}
