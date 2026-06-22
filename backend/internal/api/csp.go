// Package api — CSP nonce middleware.
package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// NonceContextKey — ключ контекста для CSP nonce.
const NonceContextKey contextKey = "csp-nonce"

// CSPNonceMiddleware генерирует уникальный nonce для каждого запроса,
// сохраняет его в контексте и выставляет в заголовке X-CSP-Nonce.
func CSPNonceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := generateNonce()
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

// generateNonce создаёт криптографически безопасный nonce (16 байт, base64).
func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback: panic недопустим, возвращаем пустую строку
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}
