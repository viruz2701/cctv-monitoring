package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

// setupSBOMTest создаёт временную директорию с тестовыми SBOM файлами.
func setupSBOMTest(t *testing.T) (*SBOMProvider, string) {
	t.Helper()

	dir := t.TempDir()

	// CycloneDX SBOM
	cdxSBOM := map[string]interface{}{
		"bomFormat":   "CycloneDX",
		"specVersion": "1.6",
		"version":     1,
		"metadata": map[string]interface{}{
			"component": map[string]interface{}{
				"name":    "gb-telemetry-collector",
				"version": "0.0.0-dev",
				"type":    "application",
			},
		},
		"components": []map[string]interface{}{
			{
				"name":    "github.com/go-chi/chi/v5",
				"version": "5.2.1",
				"type":    "library",
			},
		},
	}
	cdxData, _ := json.Marshal(cdxSBOM)
	_ = os.WriteFile(filepath.Join(dir, "sbom.cyclonedx.json"), cdxData, 0644)

	provider := NewSBOMProvider(dir, "2026-06-28T20:00:00Z", "0.0.0-dev")
	return provider, dir
}

// mountSBOMRoutes монтирует SBOM маршруты на chi router.
func mountSBOMRoutes(provider *SBOMProvider) chi.Router {
	r := chi.NewRouter()
	r.Get("/api/v1/sbom", provider.HandleListSBOM)
	r.Get("/api/v1/sbom/{format}", provider.HandleGetSBOM)
	r.Get("/api/v1/sbom/{format}/raw", provider.HandleGetSBOMRaw)
	return r
}

func TestSBOMProvider_HandleListSBOM(t *testing.T) {
	provider, _ := setupSBOMTest(t)
	r := mountSBOMRoutes(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SBOMListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Available) == 0 {
		t.Fatal("expected at least one SBOM format")
	}
	if resp.Default != SBOMFormatCycloneDX {
		t.Fatalf("expected default=cyclonedx, got %s", resp.Default)
	}
}

func TestSBOMProvider_HandleGetSBOM_Default(t *testing.T) {
	provider, _ := setupSBOMTest(t)
	r := mountSBOMRoutes(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/cyclonedx", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp SBOMResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Format != SBOMFormatCycloneDX {
		t.Fatalf("expected format=cyclonedx, got %s", resp.Format)
	}
	if resp.VersionInfo.Version != "0.0.0-dev" {
		t.Fatalf("expected version=0.0.0-dev, got %s", resp.VersionInfo.Version)
	}
	if resp.VersionInfo.BuildTime != "2026-06-28T20:00:00Z" {
		t.Fatalf("unexpected build_time: %s", resp.VersionInfo.BuildTime)
	}
}

func TestSBOMProvider_HandleGetSBOM_UnsupportedFormat(t *testing.T) {
	provider, _ := setupSBOMTest(t)
	r := mountSBOMRoutes(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/invalid", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSBOMProvider_HandleGetSBOM_NotFound(t *testing.T) {
	provider, dir := setupSBOMTest(t)
	// Удаляем cyclonedx файл, оставляем только spdx (которого нет)
	_ = os.Remove(filepath.Join(dir, "sbom.cyclonedx.json"))
	r := mountSBOMRoutes(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/spdx", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSBOMProvider_HandleGetSBOMRaw(t *testing.T) {
	provider, _ := setupSBOMTest(t)
	r := mountSBOMRoutes(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/cyclonedx/raw", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/vnd.cyclonedx+json" {
		t.Fatalf("expected Content-Type=application/vnd.cyclonedx+json, got %s", ct)
	}

	// Verify it's valid JSON
	var raw json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}
}

func TestSBOMProvider_Cache(t *testing.T) {
	provider, dir := setupSBOMTest(t)
	r := mountSBOMRoutes(provider)

	// First load - cache miss
	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/cyclonedx", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on first load, got %d", rec.Code)
	}

	// Delete the underlying file
	_ = os.Remove(filepath.Join(dir, "sbom.cyclonedx.json"))

	// Second load - should still work from cache
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/cyclonedx", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 from cache, got %d", rec2.Code)
	}
}

func TestSBOMProvider_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	provider := NewSBOMProvider(dir, "", "test")
	r := mountSBOMRoutes(provider)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sbom/cyclonedx", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing SBOM, got %d", rec.Code)
	}
}
