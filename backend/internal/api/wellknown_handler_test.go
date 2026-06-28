package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestWellKnownHandler_HandleSecurityTxt_Default(t *testing.T) {
	handler := NewWellKnownHandler("")
	r := chi.NewRouter()
	r.Get("/.well-known/security.txt", handler.HandleSecurityTxt)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("expected text/plain content type, got %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "security@gb-telemetry.com") {
		t.Fatal("expected security contact email in response")
	}
	if !strings.Contains(body, "RFC 9116") {
		t.Fatal("expected RFC 9116 reference in response")
	}
}

func TestWellKnownHandler_HandleSecurityTxt_FromFile(t *testing.T) {
	dir := t.TempDir()
	customContent := "Contact: mailto:custom@example.com\nExpires: 2028-01-01T00:00:00Z\n"
	txtPath := filepath.Join(dir, "security.txt")
	_ = os.WriteFile(txtPath, []byte(customContent), 0644)

	handler := NewWellKnownHandler(txtPath)
	r := chi.NewRouter()
	r.Get("/.well-known/security.txt", handler.HandleSecurityTxt)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "custom@example.com") {
		t.Fatal("expected custom contact email from file")
	}
}

func TestWellKnownHandler_HandleSecurityTxt_AccessControl(t *testing.T) {
	handler := NewWellKnownHandler("")
	r := chi.NewRouter()
	r.Get("/.well-known/security.txt", handler.HandleSecurityTxt)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	origin := rec.Header().Get("Access-Control-Allow-Origin")
	if origin != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin: *, got %s", origin)
	}
}

func TestWellKnownHandler_HandleSecurityPolicy(t *testing.T) {
	handler := NewWellKnownHandler("")
	r := chi.NewRouter()
	r.Get("/.well-known/security-policy", handler.HandleSecurityPolicy)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/security-policy", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html content type, got %s", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Vulnerability Disclosure Program") {
		t.Fatal("expected VDP text in HTML response")
	}
	if !strings.Contains(body, "security@gb-telemetry.com") {
		t.Fatal("expected security contact email in HTML response")
	}
}

func TestWellKnownHandler_FileNotFound_Fallback(t *testing.T) {
	// Путь к несуществующему файлу — должен упасть на fallback
	handler := NewWellKnownHandler("/nonexistent/path/security.txt")
	r := chi.NewRouter()
	r.Get("/.well-known/security.txt", handler.HandleSecurityTxt)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/security.txt", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (fallback), got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "security@gb-telemetry.com") {
		t.Fatal("expected fallback to default security.txt content")
	}
}
