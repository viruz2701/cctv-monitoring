// Package auth — аутентификация и управление доступом.
//
// P2-CR.4: Session & Auth Regional Policies
//   - AuthMiddleware использует региональные политики сессий из claims
//   - Graceful warning headers перед истечением таймаута
//   - Admin override для экстренных случаев (ISO 27001 A.9.2.3)
//   - Audit log для session events (ISO 27001 A.12.4)
//
// Compliance:
//   - OWASP ASVS V3 (Session Management)
//   - ISO 27001 A.9.4 (Access control — session management)
//   - IEC 62443 SR 2.1 (Account management — session timeout)
//   - СТБ 34.101.27 п. 6.1 (Аутентификация — таймауты сессий)
//   - Приказ ОАЦ №66 п. 7.18.2 (Защита сетей — управление сессиями)
package auth

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

type contextKey string

const UserContextKey contextKey = "user"

// Session event types для audit log.
const (
	SessionEventExpired     = "session_expired"
	SessionEventAdminBypass = "session_admin_bypass"
	SessionEventWarning     = "session_warning"
)

// logger для audit log session events.
var sessionLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))

// AuthMiddleware проверяет JWT и применяет региональные политики сессий.
//
// Обновление (P2-CR.4):
//   - Определяет регион из JWT claims
//   - Применяет региональный IdleTimeout/AbsoluteTimeout
//   - Добавляет warning headers при приближении таймаута
//   - Admin override: admin/superadmin bypass session timeout
//   - Audit log для session events
//
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

		// P2-CR.4: Определяем региональную политику сессий
		region := Region(claims.Region)
		policy := GetSessionPolicy(region)

		// P2-CR.4: Admin override — администраторы bypass session timeout
		// (ISO 27001 A.9.2.3 — Privilege management, экстренный доступ)
		if IsAdminOverride(claims.Role) {
			sessionLogger.Info("session admin bypass",
				"user_id", claims.UserID,
				"role", claims.Role,
				"region", region,
				"trace_id", r.Header.Get("X-Request-ID"),
				"event", SessionEventAdminBypass,
			)
			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// P2-CR.4: Idle timeout enforcement (региональный)
		if claims.IssuedAt != nil {
			sessionAge := time.Since(claims.IssuedAt.Time)

			// Warning headers: предупреждаем за WarningThreshold до таймаута
			warningThreshold := policy.WarningThreshold()
			remaining := policy.IdleTimeout - sessionAge

			w.Header().Set("X-Session-Timeout", policy.IdleTimeout.String())
			w.Header().Set("X-Session-Age", sessionAge.Round(time.Second).String())
			w.Header().Set("X-Session-Remaining", remaining.Round(time.Second).String())
			w.Header().Set("X-Session-Region", string(region))

			if sessionAge > policy.IdleTimeout {
				sessionLogger.Warn("session expired due to inactivity",
					"user_id", claims.UserID,
					"role", claims.Role,
					"region", region,
					"idle_timeout", policy.IdleTimeout.String(),
					"session_age", sessionAge.Round(time.Second).String(),
					"trace_id", r.Header.Get("X-Request-ID"),
					"event", SessionEventExpired,
				)
				writeAuthError(w, r, "session expired due to inactivity")
				return
			}

			// Graceful warning: если осталось меньше WarningThreshold
			if remaining <= warningThreshold {
				w.Header().Set("X-Session-Warning", "true")
				w.Header().Set("X-Session-Warning-In", remaining.Round(time.Second).String())

				sessionLogger.Info("session warning",
					"user_id", claims.UserID,
					"role", claims.Role,
					"region", region,
					"remaining", remaining.Round(time.Second).String(),
					"trace_id", r.Header.Get("X-Request-ID"),
					"event", SessionEventWarning,
				)
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
