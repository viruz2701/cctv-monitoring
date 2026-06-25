// Package auth — Tenant-aware middleware (F-0.2.3).
//
// TenantMiddleware извлекает tenantID из JWT Claims и устанавливает
// его в контекст запроса для последующего использования RLS-политиками.
//
// Compliance:
//   - IEC 62443 SR 2.1 (Account management — tenant isolation)
//   - IEC 62443 SR 5.1 (Network segmentation — tenant data isolation)
//   - ISO 27001 A.9.1.2 (Access to networks — tenant separation)
//   - ISO 27001 A.9.2.3 (Privilege management — admin bypass)
//   - OWASP ASVS V2.1 (Authentication — tenant context)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
package auth

import (
	"context"
	"net/http"
)

// TenantIDKey — ключ для tenantID в контексте запроса.
const TenantIDKey contextKey = "tenant_id"

// TenantRoleKey — ключ для роли в контексте запроса (для RLS).
const TenantRoleKey contextKey = "tenant_role"

// TenantMiddleware извлекает tenantID из JWT Claims и проверяет доступ.
//
// ДОЛЖЕН быть установлен после AuthMiddleware (чтобы Claims были доступны).
//
// Admin bypass: если role == "admin", tenantID устанавливается в "*",
// что позволяет RLS-политикам видеть данные всех tenant'ов.
//
// Соответствует: ISO 27001 A.9.2.3, IEC 62443 SR 2.1
func TenantMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)
		if claims == nil {
			// AuthMiddleware должен отработать раньше, но на всякий случай
			writeAuthError(w, r, "unauthorized: missing claims")
			return
		}

		tenantID := claims.TenantID
		role := claims.Role

		// Admin bypass: admin видит все tenant'ы
		if role == "admin" {
			tenantID = "*"
		}

		ctx := context.WithValue(r.Context(), TenantIDKey, tenantID)
		ctx = context.WithValue(ctx, TenantRoleKey, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetTenantID извлекает tenantID из контекста запроса.
// Возвращает пустую строку, если tenantID не установлен.
func GetTenantID(r *http.Request) string {
	if v := r.Context().Value(TenantIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetTenantRole извлекает роль из tenant-контекста.
func GetTenantRole(r *http.Request) string {
	if v := r.Context().Value(TenantRoleKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ContextWithTenantID добавляет tenantID в context (для тестов и system context).
func ContextWithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, TenantIDKey, tenantID)
}

// ContextWithTenantRole добавляет role в context.
func ContextWithTenantRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, TenantRoleKey, role)
}
