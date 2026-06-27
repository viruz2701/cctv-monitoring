// Package api — Feature Flag middleware (F-0.2.4).
//
// FeatureFlagMiddleware проверяет фича-флаг перед обработкой запроса.
// Если флаг disabled → HTTP 503 Service Unavailable.
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — feature gating)
//   - IEC 62443-3-3 SR 7.1 (Resource availability — feature control)
//   - ISO 27001 A.12.1.2 (Change management — controlled rollout)
//   - OWASP ASVS V1.1 (Architecture — feature flags как security control)
package api

import (
	"net/http"
)

// FeatureFlagMiddleware возвращает middleware которая проверяет фича-флаг.
// Если флаг disabled → 503 Service Unavailable.
// Если флаг не найден → 503 (fail-secure).
//
// Использование:
//
//	r.Group(func(r chi.Router) {
//	    r.Use(api.FeatureFlagMiddleware("cmms.adapter.atlas"))
//	    r.Post("/api/v1/atlas/sync", s.handleAtlasSync)
//	})
func (s *Server) FeatureFlagMiddleware(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.featureFlags.IsEnabled(key) {
				w.Header().Set("Retry-After", "3600")
				RespondError(w, r, &APIError{
					Status:  http.StatusServiceUnavailable,
					Code:    "FEATURE_DISABLED",
					Message: "Feature '" + key + "' is disabled",
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
