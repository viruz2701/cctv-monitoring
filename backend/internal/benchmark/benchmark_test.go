// ═══════════════════════════════════════════════════════════════════════════
// P1-PERF.7: Performance Benchmarking Suite
//
// Benchmarks for critical paths:
//   - JSON serialization/deserialization
//   - Health check response building
//   - Memory stats collection
//   - Circuit breaker operations
//
// Запуск: go test -bench=. -benchmem ./internal/benchmark/
// ═══════════════════════════════════════════════════════════════════════════

package benchmark

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"
)

// ── Test types matching health_handlers.go structures ──────────────────────

type healthDetail struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Latency   string `json:"latency,omitempty"`
	LastCheck string `json:"last_check,omitempty"`
}

type poolStats struct {
	MaxConns          int32 `json:"max_conns"`
	AcquiredConns     int32 `json:"acquired_conns"`
	IdleConns         int32 `json:"idle_conns"`
	ConstructingConns int32 `json:"constructing_conns"`
	TotalConns        int32 `json:"total_conns"`
}

type memoryStats struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	HeapInUseMB  float64 `json:"heap_in_use_mb"`
	Warning      string  `json:"warning,omitempty"`
}

type circuitBreakerStatus struct {
	State   string `json:"state"`
	Counter int    `json:"counter,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type healthResponse struct {
	Status         string                  `json:"status"`
	Timestamp      time.Time               `json:"timestamp"`
	Uptime         string                  `json:"uptime,omitempty"`
	Dependencies   map[string]healthDetail `json:"dependencies,omitempty"`
	PoolStats      *poolStats              `json:"pool_stats,omitempty"`
	Region         string                  `json:"region,omitempty"`
	Memory         *memoryStats            `json:"memory,omitempty"`
	CircuitBreaker *circuitBreakerStatus   `json:"circuit_breaker,omitempty"`
}

// makeFullHealthResponse creates a health response with all fields populated.
func makeFullHealthResponse() healthResponse {
	return healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Uptime:    "72h30m15s",
		Region:    "BY",
		Dependencies: map[string]healthDetail{
			"database": {Status: "ok", Latency: "1.2ms", LastCheck: "2026-06-28T09:00:00Z"},
			"nats":     {Status: "ok", Latency: "0.5ms", LastCheck: "2026-06-28T09:00:00Z"},
			"redis":    {Status: "ok", Latency: "0.3ms", LastCheck: "2026-06-28T09:00:00Z"},
			"auth":     {Status: "ok", LastCheck: "2026-06-28T09:00:00Z"},
			"disk":     {Status: "ok", LastCheck: "2026-06-28T09:00:00Z"},
		},
		PoolStats: &poolStats{
			MaxConns:      25,
			AcquiredConns: 3,
			IdleConns:     7,
			TotalConns:    10,
		},
		Memory: &memoryStats{
			AllocMB:      128.5,
			TotalAllocMB: 1024.0,
			SysMB:        512.0,
			HeapInUseMB:  200.0,
		},
		CircuitBreaker: &circuitBreakerStatus{
			State: "closed",
		},
	}
}

// ── JSON Serialization Benchmarks ─────────────────────────────────────────

// BenchmarkJSONMarshalHealthResponse benchmarks JSON marshaling of a full health response.
func BenchmarkJSONMarshalHealthResponse(b *testing.B) {
	resp := makeFullHealthResponse()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resp)
	}
}

// BenchmarkJSONUnmarshalHealthResponse benchmarks JSON unmarshaling of a health response.
func BenchmarkJSONUnmarshalHealthResponse(b *testing.B) {
	resp := makeFullHealthResponse()
	data, _ := json.Marshal(resp)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var decoded healthResponse
		_ = json.Unmarshal(data, &decoded)
	}
}

// ── Health Response Building Benchmarks ────────────────────────────────────

// BenchmarkBuildHealthResponse benchmarks constructing a full health response.
func BenchmarkBuildHealthResponse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = makeFullHealthResponse()
	}
}

// BenchmarkBuildHealthResponseMinimal benchmarks constructing a minimal health response.
func BenchmarkBuildHealthResponseMinimal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = healthResponse{
			Status:    "ok",
			Timestamp: time.Now().UTC(),
		}
	}
}

// ── Memory Stats Benchmarks ───────────────────────────────────────────────

// BenchmarkCollectMemoryStats benchmarks runtime memory stats collection.
func BenchmarkCollectMemoryStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Simulate collectMemoryStats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		_ = &memoryStats{
			AllocMB:      bytesToMB(m.Alloc),
			TotalAllocMB: bytesToMB(m.TotalAlloc),
			SysMB:        bytesToMB(m.Sys),
			HeapInUseMB:  bytesToMB(m.HeapInuse),
		}
	}
}

// ── Circuit Breaker Benchmarks ────────────────────────────────────────────

// BenchmarkCircuitBreakerStatus benchmarks circuit breaker operations.
func BenchmarkCircuitBreakerStatus(b *testing.B) {
	cb := &circuitBreakerStatus{State: "closed"}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cb.State = "closed"
		cb.Counter = i
		_ = cb.State
	}
}

// ── Map Operations Benchmarks ─────────────────────────────────────────────

// BenchmarkDependencyMapInsert benchmarks inserting into a dependency map.
func BenchmarkDependencyMapInsert(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m := make(map[string]healthDetail, 5)
		m["database"] = healthDetail{Status: "ok"}
		m["nats"] = healthDetail{Status: "ok"}
		m["redis"] = healthDetail{Status: "ok"}
		m["auth"] = healthDetail{Status: "ok"}
		m["disk"] = healthDetail{Status: "ok"}
		_ = m
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

func bytesToMB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}
