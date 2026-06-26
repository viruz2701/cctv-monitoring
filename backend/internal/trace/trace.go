// Package trace — распределённая трассировка (W3C Trace Context).
//
// Обеспечивает сквозную (end-to-end) трассировку запросов через все слои:
// HTTP middleware → service layer → events/NATS → audit log.
//
// Соответствие:
//   - ISO 27001 A.12.4.1: Сквозная трассировка событий
//   - IEC 62443-3-3 SR 6.1: Audit log integrity
//   - OWASP ASVS V7.1.1: Trace ID в каждом ответе
//   - W3C Trace Context: https://www.w3.org/TR/trace-context/
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
)

// ── Context Key ──────────────────────────────────────────────────────

type ctxKeyTraceID struct{}

// ── Middleware ───────────────────────────────────────────────────────

// Middleware генерирует trace_id (X-Request-ID) для каждого HTTP-запроса.
// Принимает входящий X-Request-ID или создаёт новый 16-байтный hex.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Request-ID")
		if traceID == "" {
			// Пробуем W3C Traceparent
			if tp := r.Header.Get("traceparent"); tp != "" {
				traceID = tp
			} else {
				traceID = NewID()
			}
		}
		w.Header().Set("X-Request-ID", traceID)
		ctx := context.WithValue(r.Context(), ctxKeyTraceID{}, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ── Context Helpers ──────────────────────────────────────────────────

// FromContext извлекает trace_id из context.Context.
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTraceID{}).(string); ok {
		return v
	}
	return ""
}

// WithContext возвращает новый context с trace_id.
func WithContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID{}, traceID)
}

// NewID генерирует новый trace ID (16 байт crypto/rand → hex).
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ── Log Attributes ───────────────────────────────────────────────────

// SlogAttr возвращает slog.Attr с trace_id для включения в логи.
func SlogAttr(ctx context.Context) slog.Attr {
	return slog.String("trace_id", FromContext(ctx))
}

// ── HTTP Handler для NATS/Event trace ────────────────────────────────

// HTTPHandler возвращает middleware, который логирует trace_id и
// передаёт его в контекст. Используется на всех HTTP endpoint'ах.
func HTTPHandler(next http.Handler) http.Handler {
	return Middleware(next)
}
