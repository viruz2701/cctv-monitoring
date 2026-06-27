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
	"strings"
)

// ── Context Key ──────────────────────────────────────────────────────

type ctxKeyTraceID struct{}

// ── Middleware ───────────────────────────────────────────────────────

// Middleware генерирует trace_id (X-Request-ID) для каждого HTTP-запроса.
// Совместим с Chi router. Порядок определения trace ID:
//  1. X-Request-ID header (наивысший приоритет)
//  2. traceparent header (W3C Trace Context)
//  3. Новый сгенерированный ID (fallback)
//
// Устанавливает заголовок ответа X-Request-ID.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := extractTraceID(r)
		w.Header().Set("X-Request-ID", traceID)
		ctx := context.WithValue(r.Context(), ctxKeyTraceID{}, traceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractTraceID извлекает trace ID из HTTP-запроса в порядке приоритета:
//
//  1. X-Request-ID header (наивысший приоритет)
//  2. traceparent header (W3C Trace Context), извлекает только trace-id
//  3. Новый сгенерированный ID (fallback)
func extractTraceID(r *http.Request) string {
	// Приоритет 1: X-Request-ID
	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	// Приоритет 2: traceparent (W3C Trace Context)
	if tp := r.Header.Get("traceparent"); tp != "" {
		if id := parseTraceparent(tp); id != "" {
			return id
		}
	}
	// Приоритет 3: новый ID
	return NewID()
}

// parseTraceparent извлекает trace-id из W3C traceparent header.
//
// Формат: version-traceid-parentid-flags
// Пример: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
//   - version:    2 hex символа
//   - trace-id:   32 hex символа (16 байт) — извлекается
//   - parent-id:  16 hex символа
//   - trace-flags: 2 hex символа
//
// Возвращает пустую строку при невалидном формате.
func parseTraceparent(tp string) string {
	parts := strings.SplitN(tp, "-", 4)
	if len(parts) != 4 {
		return ""
	}
	traceID := parts[1]
	if len(traceID) != 32 {
		return ""
	}
	if _, err := hex.DecodeString(traceID); err != nil {
		return ""
	}
	return traceID
}

// ── Context Helpers ──────────────────────────────────────────────────

// FromContext извлекает trace_id из context.Context.
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyTraceID{}).(string); ok {
		return v
	}
	return ""
}

// FromContextOrDefault извлекает trace_id или возвращает "unknown".
// Используется в сервисном слое и фоновых горутинах, где trace ID
// может отсутствовать, но лог должен быть структурированным.
func FromContextOrDefault(ctx context.Context) string {
	if id := FromContext(ctx); id != "" {
		return id
	}
	return "unknown"
}

// WithContext возвращает новый context с trace_id.
func WithContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID{}, traceID)
}

// WithNewID возвращает новый context со свежим trace_id.
// Используется для фоновых горутин (background goroutines),
// NATS-обработчиков и планировщиков, где нужно продолжить
// цепочку трассировки с новым ID.
func WithNewID(ctx context.Context) context.Context {
	return WithContext(ctx, NewID())
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

// LogAttrs возвращает []slog.Attr с trace_id для включения
// в структурированные логи. Удобство для множественных аттрибутов:
//
//	slog.Info("msg", trace.LogAttrs(ctx)...)
func LogAttrs(ctx context.Context) []slog.Attr {
	return []slog.Attr{SlogAttr(ctx)}
}

// ── HTTP Extraction (non-middleware) ─────────────────────────────────

// ExtractFromHTTP извлекает trace ID из HTTP-запроса и возвращает
// context с ним. Полезно для сценариев без middleware:
//   - WebSocket upgrade
//   - Server-Sent Events (SSE)
//   - Тестирование
//   - Кастомные обработчики
func ExtractFromHTTP(r *http.Request) context.Context {
	traceID := extractTraceID(r)
	return WithContext(r.Context(), traceID)
}

// ── HTTP Handler для NATS/Event trace ────────────────────────────────

// HTTPHandler возвращает middleware, который логирует trace_id и
// передаёт его в контекст. Используется на всех HTTP endpoint'ах.
func HTTPHandler(next http.Handler) http.Handler {
	return Middleware(next)
}
