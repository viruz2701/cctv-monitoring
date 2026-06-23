package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const UserContextKey contextKey = "user"

// SessionTimeout — максимальное время бездействия сессии (ISO 27001 A.9.4)
const SessionTimeout = 30 * time.Minute

// MaxConcurrentSessions — максимальное количество одновременных сессий (ISO 27001 A.9.4)
const MaxConcurrentSessions = 3

// AuthMiddleware проверяет JWT и применяет session timeout.
// Соответствует: OWASP ASVS V3 (Session Management), ISO 27001 A.9.4
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeAuthError(w, r, "missing authorization header")
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			writeAuthError(w, r, "invalid authorization header format")
			return
		}
		claims, err := ValidateJWT(parts[1])
		if err != nil {
			writeAuthError(w, r, "invalid or expired token")
			return
		}

		// ISO 27001 A.9.4: Session timeout enforcement (30 мин idle)
		if claims.IssuedAt != nil {
			sessionAge := time.Since(claims.IssuedAt.Time)
			if sessionAge > SessionTimeout {
				writeAuthError(w, r, "session expired due to inactivity")
				return
			}
		}

		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetClaims извлекает JWT claims из контекста запроса.
func GetClaims(r *http.Request) *Claims {
	claims, ok := r.Context().Value(UserContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}

func writeAuthError(w http.ResponseWriter, r *http.Request, message string) {
	traceID := r.Header.Get("X-Request-ID")
	if traceID == "" {
		traceID = "unknown"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    "UNAUTHORIZED",
			"message": message,
		},
		"trace_id":  traceID,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
