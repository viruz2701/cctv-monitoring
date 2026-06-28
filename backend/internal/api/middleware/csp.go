// Package middleware — CSP nonce middleware.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"net/http"
	"os"
)

// NonceContextKey — ключ контекста для CSP nonce.
const NonceContextKey contextKey = "csp-nonce"

// contextKey — тип для ключей контекста (избегаем collision).
type contextKey string

// cspLogger — пакетный логгер для CSP nonce. По умолчанию slog.Default().
// Можно переопределить через SetCSPLogger для тестов.
var cspLogger = slog.New(slog.NewTextHandler(os.Stdout, nil))

// SetCSPLogger устанавливает логгер для CSP nonce.
// Используется в тестах для подавления логов.
func SetCSPLogger(logger *slog.Logger) {
	if logger != nil {
		cspLogger = logger
	}
}

// CSPNonceMiddleware генерирует уникальный nonce для каждого запроса,
// сохраняет его в контексте и выставляет в заголовке X-CSP-Nonce.
//
// Graceful degradation: при ошибке crypto/rand nonce будет пустым,
// но сервер продолжит работу (ADR-004).
func CSPNonceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := GenerateNonce()
		ctx := context.WithValue(r.Context(), NonceContextKey, nonce)
		w.Header().Set("X-CSP-Nonce", nonce)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// NonceFromContext извлекает CSP nonce из контекста запроса.
func NonceFromContext(ctx context.Context) string {
	nonce, _ := ctx.Value(NonceContextKey).(string)
	return nonce
}

// GenerateNonce создаёт CSP nonce (16 байт, base64).
// При ошибке crypto/rand логирует ошибку и возвращает пустой nonce.
// Graceful degradation: CSP будет менее безопасен, но сервер продолжит работу.
func GenerateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		cspLogger.Error("csp: crypto/rand.Read failed, using empty nonce",
			"error", err,
		)
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}
