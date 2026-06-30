// Package api — маршруты для управления версиями API.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-API: API Versioning Strategy — Route Mounting
//
// Эндпоинты:
//
//	GET    /api/v1/versions              — список версий (public)
//	POST   /api/v1/versions              — новая версия (admin)
//	PUT    /api/v1/versions/{version}    — обновить метаданные (admin)
//	GET    /api/v1/changelog             — changelog (public)
//
// Соответствует:
//   - IEC 62443-3-3 SL-2 (Zone 2 — DMZ): Управление изменениями
//   - ISO 27001 A.12.4.1: Audit trail для изменений
//   - OWASP ASVS V4.1 (Access control — RBAC enforcement)
//   - OWASP ASVS V5.1 (Input validation — whitelist validation)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountVersionRoutes монтирует маршруты управления версиями API.
//
// Public endpoints (без JWT):
//
//	GET /api/v1/versions — список версий
//	GET /api/v1/changelog — changelog
//
// Admin-only (JWT + admin role):
//
//	POST /api/v1/versions               — создать версию
//	PUT  /api/v1/versions/{version}     — обновить версию
func (s *Server) mountVersionRoutes(r chi.Router) {
	// Публичные маршруты
	r.Get("/api/v1/versions", s.handleListVersions)
	r.Get("/api/v1/changelog", s.handleGetChangelog)

	// Admin-only маршруты (POST, PUT — мутации)
	r.Post("/api/v1/versions", s.handleCreateVersion)
	r.Put("/api/v1/versions/{version}", s.handleUpdateVersion)
}
