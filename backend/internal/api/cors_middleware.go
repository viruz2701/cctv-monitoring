// Package api — CORS middleware with OWASP ASVS L3 compliance.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-SEC.2: CORS Wildcard Fix
//
// OWASP ASVS L3 V9.1 (V13.4): ЗАПРЕЩЕНО использовать wildcard "*" в production.
// Требование ISO 27001 A.13.2: Only explicitly whitelisted origins.
//
// Правила:
//  1. Wildcard "*" вызывает fatal error при старте (не fallback)
//  2. Empty origins вызывают fatal error (кроме debug mode)
//  3. Development: localhost origins разрешены автоматически
//  4. Production: только явно указанные origins из конфига
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"fmt"
	"strings"

	"github.com/go-chi/cors"
)

// ────────────────────────────────────────────────────────────────────────────
// CORS validation
// ────────────────────────────────────────────────────────────────────────────

// ValidateCORSOrigins проверяет CORS origins на соответствие OWASP ASVS L3.
//
// Возвращает ошибку если:
//   - Список пуст (не в debug mode)
//   - Содержит wildcard "*"
//
// Для debug mode разрешены localhost origins без явной конфигурации.
// Для production требуется явная конфигурация.
func ValidateCORSOrigins(origins []string, debug bool) error {
	if len(origins) == 0 {
		if debug {
			return nil // debug mode: разрешаем пустой список (будет localhost default)
		}
		return fmt.Errorf("CORS: cors_allowed_origins is empty — " +
			"требуется явная конфигурация для production (OWASP ASVS V13.4)")
	}

	for _, origin := range origins {
		if origin == "*" {
			return fmt.Errorf("CORS: wildcard origin '*' detected — " +
				"ЗАПРЕЩЕНО для production (OWASP ASVS V9.1, ISO 27001 A.13.2)")
		}
	}

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// CORS options factory
// ────────────────────────────────────────────────────────────────────────────

// DefaultAllowedOrigins — безопасные дефолты для development.
var DefaultAllowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
	"http://localhost:8080",
	"http://127.0.0.1:3000",
	"http://127.0.0.1:5173",
	"http://127.0.0.1:8080",
}

// isLocalhostOrigin проверяет, является ли origin localhost/127.0.0.1 адресом.
func isLocalhostOrigin(origin string) bool {
	return strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "https://localhost") ||
		strings.HasPrefix(origin, "http://127.0.0.1") ||
		strings.HasPrefix(origin, "https://127.0.0.1")
}

// NewCORSHandler создаёт CORS middleware handler с валидацией.
//
// В debug mode: если origins пуст, используются DefaultAllowedOrigins.
// В production: origins ДОЛЖНЫ быть явно указаны в конфиге.
func NewCORSHandler(origins []string, debug bool) (cors.Options, error) {
	if err := ValidateCORSOrigins(origins, debug); err != nil {
		return cors.Options{}, err
	}

	allowedOrigins := origins
	if len(allowedOrigins) == 0 && debug {
		allowedOrigins = DefaultAllowedOrigins
	}

	return cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID", "X-Trace-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           300, // 5 minutes, ISO 27001 A.13.2 recommendation
	}, nil
}
