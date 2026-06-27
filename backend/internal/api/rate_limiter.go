// Package api — rate limiter middleware.
// Соответствует: ISO 27001 A.12.1.2, OWASP ASVS V2.2.1, СТБ 34.101.27 п. 6.1
package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// rateLimiter — in-memory rate limiter per IP with automatic cleanup.
type rateLimiter struct {
	mu          sync.Mutex
	entries     map[string][]time.Time
	limit       int
	window      time.Duration
	cancel      context.CancelFunc
	activeCount atomic.Int64 // метрика: количество активных entries
}

// newRateLimiter создаёт rate limiter и запускает фоновую очистку.
func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	ctx, cancel := context.WithCancel(context.Background())
	rl := &rateLimiter{
		entries: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
		cancel:  cancel,
	}
	go rl.cleanup(ctx)
	return rl
}

// ActiveEntriesCount возвращает количество активных IP в map (для метрик).
func (rl *rateLimiter) ActiveEntriesCount() int64 {
	return rl.activeCount.Load()
}

// allow проверяет, не превышен ли лимит для данного IP.
func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	entries := rl.entries[ip]
	filtered := entries[:0]
	for _, t := range entries {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) >= rl.limit {
		rl.entries[ip] = filtered
		return false
	}

	filtered = append(filtered, now)
	rl.entries[ip] = filtered
	// Обновляем метрику при добавлении новой записи
	rl.activeCount.Store(int64(len(rl.entries)))
	return true
}

// cleanupExpired удаляет просроченные записи (публичный метод для тестов).
func (rl *rateLimiter) cleanupExpired() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)
	for ip, entries := range rl.entries {
		filtered := entries[:0]
		for _, t := range entries {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			delete(rl.entries, ip)
		} else {
			rl.entries[ip] = filtered
		}
	}
	// Обновляем метрику после очистки
	rl.activeCount.Store(int64(len(rl.entries)))
}

// cleanup периодически удаляет просроченные записи и пустые IP.
// Каждые 5 минут (ISO 27001 A.12.1.2 — resource management).
func (rl *rateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			rl.cleanupExpired()
		case <-ctx.Done():
			return
		}
	}
}

// stop останавливает фоновую очистку через context cancellation.
func (rl *rateLimiter) stop() {
	rl.cancel()
}

// extractClientIP извлекает IP клиента с учётом заголовков прокси.
func extractClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	// Убираем порт из RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// newRateLimiterMiddleware создаёт middleware с заданным лимитом и окном.
func (s *Server) newRateLimiterMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	rl := newRateLimiter(limit, window)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractClientIP(r)
			if !rl.allow(ip) {
				RespondError(w, r, NewRateLimitError("too many requests"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rateLimitMiddleware wraps the rate limiter for login endpoint (5 req/min).
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return s.newRateLimiterMiddleware(5, time.Minute)(next)
}
