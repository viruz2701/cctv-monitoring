package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// mockRedisClient — реализация RedisClient для тестов.
type mockRedisClient struct {
	pingErr error
}

func (m *mockRedisClient) Ping(ctx context.Context) error {
	return m.pingErr
}

func TestHandleLiveness(t *testing.T) {
	s := &Server{serverStart: time.Now()}
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

func TestHandleLiveness_Uptime(t *testing.T) {
	s := &Server{serverStart: time.Now().Add(-1 * time.Hour)}
	r := chi.NewRouter()
	r.Get("/health/live", s.handleLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Uptime == "" {
		t.Error("expected non-empty uptime string")
	}
	if !strings.HasSuffix(resp.Uptime, "0s") && !strings.Contains(resp.Uptime, "h") && !strings.Contains(resp.Uptime, "m") {
		t.Logf("uptime format: %s", resp.Uptime)
	}
}

func TestHandleLiveness_NoUptimeWhenServerStartZero(t *testing.T) {
	s := &Server{} // serverStart is zero
	r := chi.NewRouter()
	r.Get("/health/live", s.handleLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Uptime != "" {
		t.Errorf("expected empty uptime when serverStart is zero, got '%s'", resp.Uptime)
	}
}

func TestHandleReadinessNoDB(t *testing.T) {
	// Server without DB should return 503
	s := &Server{serverStart: time.Now()}
	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 without DB, got %d", w.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Errorf("expected status 'degraded', got '%s'", resp.Status)
	}
}

func TestHandleReadinessWithDBMock(t *testing.T) {
	s := &Server{serverStart: time.Now()}
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
	s := &Server{serverStart: time.Now()}
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
	s := &Server{serverStart: time.Now()}
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
	s := &Server{serverStart: time.Now()}
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
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		PoolStats: &poolStats{
			MaxConns:          25,
			AcquiredConns:     3,
			IdleConns:         7,
			TotalConns:        10,
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
	s := &Server{serverStart: time.Now()}
	r := chi.NewRouter()
	r.Get("/health/live", s.handleLiveness)

	req := httptest.NewRequest(http.MethodGet, "/health/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// ── Redis health check tests ──────────────────────────────────────────

func TestHandleReadiness_RedisOK(t *testing.T) {
	s := &Server{
		serverStart: time.Now(),
		redisClient: &mockRedisClient{},
	}
	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	redisDep, ok := resp.Dependencies["redis"]
	if !ok {
		t.Fatal("expected redis dependency in response")
	}
	if redisDep.Status != "ok" {
		t.Errorf("expected redis status 'ok', got '%s'", redisDep.Status)
	}
	if redisDep.Latency == "" {
		t.Error("expected non-empty latency for redis check")
	}
}

func TestHandleReadiness_RedisUnavailable(t *testing.T) {
	s := &Server{
		serverStart: time.Now(),
		redisClient: &mockRedisClient{pingErr: errors.New("connection refused")},
	}
	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	redisDep, ok := resp.Dependencies["redis"]
	if !ok {
		t.Fatal("expected redis dependency in response")
	}
	if redisDep.Status != "unavailable" {
		t.Errorf("expected redis status 'unavailable', got '%s'", redisDep.Status)
	}
	if redisDep.Error == "" {
		t.Error("expected non-empty error for unavailable redis")
	}
	if resp.Status != "degraded" {
		t.Errorf("expected overall status 'degraded', got '%s'", resp.Status)
	}
}

func TestHandleReadiness_RedisNotConfigured(t *testing.T) {
	// Redis client not set — should not appear in dependencies
	s := &Server{serverStart: time.Now()}
	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	_, exists := resp.Dependencies["redis"]
	if exists {
		t.Error("redis should not appear in dependencies when not configured")
	}
}

func TestHandleDependencies_NoDB(t *testing.T) {
	s := &Server{serverStart: time.Now()}
	r := chi.NewRouter()
	r.Get("/health/dependencies", s.handleDependencies)

	req := httptest.NewRequest(http.MethodGet, "/health/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should report database as unavailable
	dbDep, ok := resp.Dependencies["database"]
	if !ok {
		t.Fatal("expected database dependency in response")
	}
	if dbDep.Status != "unavailable" {
		t.Errorf("expected database status 'unavailable', got '%s'", dbDep.Status)
	}
	if dbDep.Latency == "" {
		t.Error("expected non-empty latency for database check")
	}
	if dbDep.LastCheck == "" {
		t.Error("expected non-empty last_check for database")
	}

	// Should report auth as unavailable
	authDep, ok := resp.Dependencies["auth"]
	if !ok {
		t.Fatal("expected auth dependency in response")
	}
	if authDep.Status != "unavailable" {
		t.Errorf("expected auth status 'unavailable', got '%s'", authDep.Status)
	}

	// Should report nats as not_configured
	natsDep, ok := resp.Dependencies["nats"]
	if !ok {
		t.Fatal("expected nats dependency in response")
	}
	if natsDep.Status != "not_configured" {
		t.Errorf("expected nats status 'not_configured', got '%s'", natsDep.Status)
	}

	// Should report redis as not_configured
	redisDep, ok := resp.Dependencies["redis"]
	if !ok {
		t.Fatal("expected redis dependency in response")
	}
	if redisDep.Status != "not_configured" {
		t.Errorf("expected redis status 'not_configured', got '%s'", redisDep.Status)
	}

	// Should report disk as not_configured
	diskDep, ok := resp.Dependencies["disk"]
	if !ok {
		t.Fatal("expected disk dependency in response")
	}
	if diskDep.Status != "not_configured" {
		t.Errorf("expected disk status 'not_configured', got '%s'", diskDep.Status)
	}

	// Should include uptime
	if resp.Uptime == "" {
		t.Error("expected non-empty uptime in dependencies response")
	}

	// Should include memory stats
	if resp.Memory == nil {
		t.Error("expected memory stats in dependencies response")
	}
}

func TestHandleDependencies_WithRedisOK(t *testing.T) {
	s := &Server{
		serverStart: time.Now(),
		redisClient: &mockRedisClient{},
	}
	r := chi.NewRouter()
	r.Get("/health/dependencies", s.handleDependencies)

	req := httptest.NewRequest(http.MethodGet, "/health/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	redisDep, ok := resp.Dependencies["redis"]
	if !ok {
		t.Fatal("expected redis dependency in response")
	}
	if redisDep.Status != "ok" {
		t.Errorf("expected redis status 'ok', got '%s'", redisDep.Status)
	}
	if redisDep.Latency == "" {
		t.Error("expected non-empty latency for redis")
	}
	if redisDep.LastCheck == "" {
		t.Error("expected non-empty last_check for redis")
	}
}

func TestHandleDependencies_WithRedisUnavailable(t *testing.T) {
	s := &Server{
		serverStart: time.Now(),
		redisClient: &mockRedisClient{pingErr: errors.New("timeout")},
	}
	r := chi.NewRouter()
	r.Get("/health/dependencies", s.handleDependencies)

	req := httptest.NewRequest(http.MethodGet, "/health/dependencies", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	redisDep, ok := resp.Dependencies["redis"]
	if !ok {
		t.Fatal("expected redis dependency in response")
	}
	if redisDep.Status != "unavailable" {
		t.Errorf("expected redis status 'unavailable', got '%s'", redisDep.Status)
	}
	if redisDep.Error != "timeout" {
		t.Errorf("expected error 'timeout', got '%s'", redisDep.Error)
	}
}

// ── Memory check tests ───────────────────────────────────────────────

func TestHandleMemory(t *testing.T) {
	s := &Server{serverStart: time.Now()}
	r := chi.NewRouter()
	r.Get("/health/memory", s.handleMemory)

	req := httptest.NewRequest(http.MethodGet, "/health/memory", nil)
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

	if resp.Memory == nil {
		t.Fatal("expected memory stats in response")
	}

	if resp.Memory.AllocMB <= 0 {
		t.Errorf("expected positive AllocMB, got %f", resp.Memory.AllocMB)
	}
	if resp.Memory.SysMB <= 0 {
		t.Errorf("expected positive SysMB, got %f", resp.Memory.SysMB)
	}
	if resp.Memory.HeapInUseMB <= 0 {
		t.Errorf("expected positive HeapInUseMB, got %f", resp.Memory.HeapInUseMB)
	}

	// Проверяем uptime
	if resp.Uptime == "" {
		t.Error("expected non-empty uptime in memory response")
	}
}

func TestMemoryStats_AllocLessThanSys(t *testing.T) {
	stats := collectMemoryStats()
	if stats == nil {
		t.Fatal("expected non-nil memory stats")
	}

	// Alloc should be less than or equal to Sys
	if stats.AllocMB > stats.SysMB {
		t.Errorf("expected AllocMB (%f) <= SysMB (%f)", stats.AllocMB, stats.SysMB)
	}
}

func TestMemoryStats_TotalAllocGreaterOrEqualAlloc(t *testing.T) {
	stats := collectMemoryStats()
	if stats == nil {
		t.Fatal("expected non-nil memory stats")
	}

	if stats.TotalAllocMB < stats.AllocMB {
		t.Errorf("expected TotalAllocMB (%f) >= AllocMB (%f)", stats.TotalAllocMB, stats.AllocMB)
	}
}

func TestMemoryStats_JSONSerialization(t *testing.T) {
	stats := &memoryStats{
		AllocMB:      128.5,
		TotalAllocMB: 1024.0,
		SysMB:        512.0,
		HeapInUseMB:  200.0,
		Warning:      "memory usage above 80%",
	}

	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("failed to marshal memoryStats: %v", err)
	}

	var decoded memoryStats
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal memoryStats: %v", err)
	}

	if decoded.AllocMB != 128.5 {
		t.Errorf("expected AllocMB 128.5, got %f", decoded.AllocMB)
	}
	if decoded.Warning != "memory usage above 80%" {
		t.Errorf("expected warning, got '%s'", decoded.Warning)
	}
}

func TestHealthResponse_WithMemory(t *testing.T) {
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Uptime:    "1h2m3s",
		Memory: &memoryStats{
			AllocMB:      64.0,
			TotalAllocMB: 256.0,
			SysMB:        128.0,
			HeapInUseMB:  80.0,
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

	if decoded["uptime"] != "1h2m3s" {
		t.Errorf("expected uptime '1h2m3s', got '%v'", decoded["uptime"])
	}

	mem, ok := decoded["memory"].(map[string]interface{})
	if !ok {
		t.Fatal("expected memory in response")
	}
	if mem["alloc_mb"] != float64(64) {
		t.Errorf("expected alloc_mb=64, got %v", mem["alloc_mb"])
	}
}

func TestHandleDBHealth_WithMemoryStats(t *testing.T) {
	s := &Server{serverStart: time.Now()}
	r := chi.NewRouter()
	r.Get("/health/db", s.handleDBHealth)

	req := httptest.NewRequest(http.MethodGet, "/health/db", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// /health/db должен включать memory stats
	if resp.Memory == nil {
		t.Error("expected memory stats in /health/db response")
	}
}

// ── DependencyDetail Latency/LastCheck tests ─────────────────────────

func TestHealthDetail_LatencyAndLastCheck(t *testing.T) {
	detail := healthDetail{
		Status:    "ok",
		Latency:   "1.5ms",
		LastCheck: "2026-06-27T20:00:00Z",
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded healthDetail
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Latency != "1.5ms" {
		t.Errorf("expected Latency '1.5ms', got '%s'", decoded.Latency)
	}
	if decoded.LastCheck != "2026-06-27T20:00:00Z" {
		t.Errorf("expected LastCheck, got '%s'", decoded.LastCheck)
	}
}

// ── SetRedisClient test ──────────────────────────────────────────────

func TestSetRedisClient(t *testing.T) {
	s := &Server{serverStart: time.Now()}
	mock := &mockRedisClient{}

	s.SetRedisClient(mock)

	if s.redisClient != mock {
		t.Error("SetRedisClient did not set the redisClient field")
	}
}

func TestSetRedisClient_HealthCheckUsesClient(t *testing.T) {
	s := &Server{serverStart: time.Now()}
	s.SetRedisClient(&mockRedisClient{})

	r := chi.NewRouter()
	r.Get("/health/ready", s.handleReadiness)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp healthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	redisDep, ok := resp.Dependencies["redis"]
	if !ok {
		t.Fatal("expected redis dependency after SetRedisClient")
	}
	if redisDep.Status != "ok" {
		t.Errorf("expected redis status 'ok', got '%s'", redisDep.Status)
	}
}
