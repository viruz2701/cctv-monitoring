// Package api — Feature Flag routes (F-0.2.4).
//
// Маршруты для управления фича-флагами.
// Защищены JWT AuthMiddleware.
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — feature gating)
//   - ISO 27001 A.12.1.2 (Change management — controlled rollout)
//   - OWASP ASVS V4.1 (Access control — JWT required)
//   - OWASP ASVS V5 (Input validation — whitelist)
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountFeatureFlagRoutes регистрирует маршруты для фича-флагов.
// Все маршруты защищены AuthMiddleware (вызывается из server.go).
func (s *Server) mountFeatureFlagRoutes(r chi.Router) {
	// GET  /api/v1/feature-flags       — список всех флагов
	// PUT  /api/v1/feature-flags/{key} — обновить флаг
	r.Get("/api/v1/feature-flags", s.handleGetFeatureFlags)
	r.Put("/api/v1/feature-flags/{key}", s.handleUpdateFeatureFlag)
}
