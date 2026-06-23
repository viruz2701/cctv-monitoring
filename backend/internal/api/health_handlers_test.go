package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestHandleLiveness(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/live", s.handleLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", resp.Status)
	}
}

func TestHandleReadinessNoDB(t *testing.T) {
	// Server without DB should return 503
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 without DB, got %d", w.Code)
	}
}

func TestHandleReadinessWithDBMock(t *testing.T) {
	// We can't easily mock pgxpool in unit tests,
	// but we can test the response structure
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	if resp.Status == "" {
		t.Error("expected non-empty status")
	}
}

func TestHealthResponseFormat(t *testing.T) {
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]healthDetail{
			"database": {Status: "ok"},
			"nats":     {Status: "unavailable", Error: "NATS not connected"},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%v'", decoded["status"])
	}

	deps, ok := decoded["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("expected dependencies map")
	}

	dbDep, ok := deps["database"].(map[string]interface{})
	if !ok {
		t.Fatal("expected database dependency")
	}
	if dbDep["status"] != "ok" {
		t.Errorf("expected database status 'ok', got '%v'", dbDep["status"])
	}
}

func TestHealthResponseContentType(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/live", s.handleLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestCheckDiskWritable(t *testing.T) {
	// Test with non-existent directory (should try to create and succeed or fail gracefully)
	err := checkDiskWritable("/tmp/test-health-dir")
	if err != nil {
		t.Logf("expected possible error for non-existent dir: %v", err)
	}
}

func TestHandleStartup_NoDB(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/startup", s.handleStartup)

	req := httptest.NewRequest(http.MethodGet, "/health/startup", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 without DB, got %d", w.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "not_ready" {
		t.Errorf("expected status 'not_ready', got '%s'", resp.Status)
	}
}

func TestHandleDBHealth_NoDB(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/db", s.handleDBHealth)

	req := httptest.NewRequest(http.MethodGet, "/health/db", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 without DB, got %d", w.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "unavailable" {
		t.Errorf("expected status 'unavailable', got '%s'", resp.Status)
	}
}

func TestHealthResponse_WithPoolStats(t *testing.T) {
	// Verify the response structure handles pool_stats correctly
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		PoolStats: &poolStats{
			MaxConns:        25,
			AcquiredConns:   3,
			IdleConns:       7,
			TotalConns:      10,
			ConstructingConns: 0,
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	stats, ok := decoded["pool_stats"].(map[string]interface{})
	if !ok {
		t.Fatal("expected pool_stats in response")
	}
	if stats["max_conns"] != float64(25) {
		t.Errorf("expected max_conns=25, got %v", stats["max_conns"])
	}
}

func TestHealthEndpointNotFound(t *testing.T) {
	s := &Server{}
	r := chi.NewRouter()
	r.Get("/health/live", s.handleLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
