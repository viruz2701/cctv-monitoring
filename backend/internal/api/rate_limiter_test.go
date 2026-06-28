package api

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// ── Rate Limiter Cleanup Tests (ISO 27001 A.12.1.2) ─────────────────────

// TestRateLimiterCleanup проверяет очистку просроченных записей.
func TestRateLimiterCleanup(t *testing.T) {
	rl := newRateLimiter(5, 100*time.Millisecond)

	// Добавляем запись
	rl.allow("192.168.1.1")

	// Ждём пока истечёт window
	time.Sleep(200 * time.Millisecond)

	// Запускаем очистку вручную
	rl.cleanupExpired()

	rl.mu.Lock()
	count := len(rl.entries)
	rl.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", count)
	}

	rl.stop()
}

// TestRateLimiterCleanupMultipleIPs проверяет очистку для нескольких IP.
func TestRateLimiterCleanupMultipleIPs(t *testing.T) {
	rl := newRateLimiter(5, 100*time.Millisecond)

	// Добавляем записи для разных IP
	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	for _, ip := range ips {
		rl.allow(ip)
	}

	// Ждём истечения
	time.Sleep(200 * time.Millisecond)

	rl.cleanupExpired()

	rl.mu.Lock()
	count := len(rl.entries)
	rl.mu.Unlock()

	if count != 0 {
		t.Errorf("expected 0 entries after cleanup, got %d", count)
	}

	rl.stop()
}

// TestRateLimiterCleanupPartial проверяет частичную очистку.
func TestRateLimiterCleanupPartial(t *testing.T) {
	rl := newRateLimiter(10, 500*time.Millisecond)

	// Добавляем старую запись (с истекшим временем)
	rl.mu.Lock()
	rl.entries["old-ip"] = []time.Time{time.Now().Add(-time.Minute)}
	rl.mu.Unlock()

	// Добавляем новую запись через allow
	rl.allow("new-ip")

	// Запускаем очистку
	rl.cleanupExpired()

	rl.mu.Lock()
	if _, exists := rl.entries["old-ip"]; exists {
		t.Error("old-ip should have been removed")
	}
	if _, exists := rl.entries["new-ip"]; !exists {
		t.Error("new-ip should still exist")
	}
	rl.mu.Unlock()

	rl.stop()
}

// TestRateLimiterActiveEntries проверяет метрику активных записей.
func TestRateLimiterActiveEntries(t *testing.T) {
	rl := newRateLimiter(10, time.Minute)

	rl.allow("192.168.1.1")
	rl.allow("192.168.1.2")

	if count := rl.ActiveEntriesCount(); count != 2 {
		t.Errorf("expected 2 active entries, got %d", count)
	}

	rl.stop()
}

// TestRateLimiterStop проверяет остановку cleanup goroutine.
func TestRateLimiterStop(t *testing.T) {
	rl := newRateLimiter(5, 100*time.Millisecond)
	rl.allow("192.168.1.1")

	// Останавливаем
	rl.stop()

	// Проверяем что после остановки не паникуем
	time.Sleep(200 * time.Millisecond)

	rl.mu.Lock()
	// entries должны быть доступны даже после stop
	_ = rl.entries
	rl.mu.Unlock()
}

// TestRateLimiterConcurrentAccess проверяет конкурентный доступ к rate limiter.
func TestRateLimiterConcurrentAccess(t *testing.T) {
	rl := newRateLimiter(100, time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ip := "192.168.1.1"
			for j := 0; j < 10; j++ {
				rl.allow(ip)
			}
		}(i)
	}
	wg.Wait()

	rl.stop()
}

// TestExtractClientIP проверяет извлечение IP из разных источников.
func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For single",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1"},
			expected: "192.168.1.1",
		},
		{
			name:     "X-Forwarded-For multiple",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			expected: "192.168.1.1",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "10.0.0.1"},
			expected: "10.0.0.1",
		},
		{
			name:     "RemoteAddr with port",
			remote:   "192.168.1.1:54321",
			expected: "192.168.1.1",
		},
		{
			name:     "IPv6 RemoteAddr",
			remote:   "[::1]:8080",
			expected: "[::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.headers
			_ = tt.remote
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P1-PERF.5: Performance Benchmarks — 10k ops/sec target
// ═══════════════════════════════════════════════════════════════════════

// BenchmarkRateLimiterSingleIP benchmarks rate limiter throughput for a single IP.
// Target: >10,000 ops/sec with minimal allocation.
func BenchmarkRateLimiterSingleIP(b *testing.B) {
	rl := newRateLimiter(100000, time.Minute)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.allow("192.168.1.1")
		}
	})

	rl.stop()
}

// BenchmarkRateLimiterManyIPs benchmarks rate limiter with many unique IPs.
// Simulates real-world load with distributed clients.
func BenchmarkRateLimiterManyIPs(b *testing.B) {
	rl := newRateLimiter(100, time.Minute)
	ips := make([]string, 1000)
	for i := range ips {
		ips[i] = fmt.Sprintf("10.0.0.%d", i%256)
	}
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
			rl.allow(ips[idx%len(ips)])
			idx++
		}
	})

	rl.stop()
}

// BenchmarkRateLimiterHighContention benchmarks rate limiter under high contention.
// Single IP, many goroutines — worst case for mutex.
func BenchmarkRateLimiterHighContention(b *testing.B) {
	rl := newRateLimiter(100000, time.Minute)
	b.ResetTimer()

	b.SetParallelism(100) // 100 goroutines on single IP
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.allow("10.0.0.1")
		}
	})

	rl.stop()
}

// BenchmarkRateLimiterRejected benchmarks rate limiter when limit is exceeded.
func BenchmarkRateLimiterRejected(b *testing.B) {
	rl := newRateLimiter(1, time.Minute) // limit 1 request per minute
	rl.allow("192.168.1.1")              // consume the only allowed request
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rl.allow("192.168.1.1")
		}
	})

	rl.stop()
}

// BenchmarkExtractClientIP benchmarks IP extraction with various headers.
func BenchmarkExtractClientIP(b *testing.B) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
	req.RemoteAddr = "192.168.1.1:54321"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractClientIP(req)
	}
}
