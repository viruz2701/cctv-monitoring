package api

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apimw "gb-telemetry-collector/internal/api/middleware"
)

func init() {
	// Подавляем логи CSP в тестах
	apimw.SetCSPLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// ── Nonce Generation Tests (OWASP ASVS V5.3.3) ──────────────────────────

// TestGenerateNonceLength проверяет длину nonce (16 байт → 24 символа base64).
func TestGenerateNonceLength(t *testing.T) {
	nonce := apimw.GenerateNonce()
	if nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	if len(nonce) != 24 {
		t.Errorf("expected nonce length 24 (16 bytes base64), got %d: %q", len(nonce), nonce)
	}
}

// TestGenerateNonceBase64 проверяет что nonce — валидный base64.
func TestGenerateNonceBase64(t *testing.T) {
	nonce := apimw.GenerateNonce()
	if nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	decoded, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		t.Errorf("nonce is not valid base64: %v", err)
	}
	if len(decoded) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(decoded))
	}
}

// TestGenerateNonceUniqueness проверяет уникальность nonce.
func TestGenerateNonceUniqueness(t *testing.T) {
	nonces := make(map[string]bool)
	for i := 0; i < 100; i++ {
		nonce := apimw.GenerateNonce()
		if nonce == "" {
			t.Fatal("unexpected empty nonce")
		}
		if nonces[nonce] {
			t.Errorf("duplicate nonce after %d iterations: %s", i, nonce)
		}
		nonces[nonce] = true
	}
}

// TestGenerateNonceCSPFormat проверяет что nonce пригоден для CSP.
func TestGenerateNonceCSPFormat(t *testing.T) {
	nonce := apimw.GenerateNonce()
	if nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	// CSP nonce должен содержать только base64 символы
	for _, c := range nonce {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=", c) {
			t.Errorf("nonce contains invalid CSP character: %c", c)
		}
	}
}

// ── CSP Middleware Tests ────────────────────────────────────────────────

// TestCSPNonceMiddleware проверяет, что middleware устанавливает заголовок.
func TestCSPNonceMiddleware(t *testing.T) {
	handler := apimw.CSPNonceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := apimw.NonceFromContext(r.Context())
		if nonce == "" {
			t.Error("nonce not found in context")
		}
		if len(nonce) != 24 {
			t.Errorf("unexpected nonce length: %d", len(nonce))
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cspNonce := rec.Header().Get("X-CSP-Nonce")
	if cspNonce == "" {
		t.Error("X-CSP-Nonce header not set")
	}
	if len(cspNonce) != 24 {
		t.Errorf("unexpected X-CSP-Nonce length: %d", len(cspNonce))
	}
}

// TestCSPNonceMiddlewarePerRequest проверяет, что nonce уникален на каждый запрос.
func TestCSPNonceMiddlewarePerRequest(t *testing.T) {
	var nonce1, nonce2 string

	handler := apimw.CSPNonceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := apimw.NonceFromContext(r.Context())
		if nonce1 == "" {
			nonce1 = n
		} else {
			nonce2 = n
		}
	}))

	req1 := httptest.NewRequest("GET", "/", nil)
	req2 := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req1)
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if nonce1 == nonce2 {
		t.Error("nonces should be unique per request")
	}
}

// TestSecurityHeadersCSP проверяет, что CSP header присутствует и содержит nonce.
func TestSecurityHeadersCSP(t *testing.T) {
	handler := apimw.CSPNonceMiddleware(securityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header not set")
	}

	// Проверяем обязательные директивы
	checks := []struct {
		name   string
		expect string
	}{
		{"default-src", "default-src 'self'"},
		{"script-src", "script-src 'self' 'nonce-"},
		{"strict-dynamic", "'strict-dynamic'"},
		{"frame-ancestors", "frame-ancestors 'none'"},
		{"base-uri", "base-uri 'self'"},
		{"form-action", "form-action 'self'"},
	}

	for _, check := range checks {
		if !strings.Contains(csp, check.expect) {
			t.Errorf("CSP missing %s: expected %q in %q", check.name, check.expect, csp)
		}
	}

	// Проверяем что нет 'unsafe-inline' в script-src
	if strings.Contains(csp, "script-src") && strings.Contains(csp, "'unsafe-inline'") {
		// Получаем часть script-src
		parts := strings.Split(csp, ";")
		for _, part := range parts {
			if strings.Contains(part, "script-src") && strings.Contains(part, "'unsafe-inline'") {
				t.Error("script-src should not contain 'unsafe-inline'")
			}
		}
	}
}
