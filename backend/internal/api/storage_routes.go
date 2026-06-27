// Package api — routes for Data Residency Enforcement (P0-CE.6).
//
// Маршруты защищены AuthMiddleware (вызывается из server.go).
//
// Compliance:
//   - OWASP ASVS V3 (Session management — через JWT middleware)
//   - OWASP ASVS V4 (RBAC — admin/manager/owner)
//   - ISO 27001 A.9.2 (Access control — role-based)
//   - IEC 62443-3-3 SR 5.1 (Zone-based access control)
package api

import "github.com/go-chi/chi/v5"

// mountStorageRoutes регистрирует маршруты Data Residency Enforcement.
//
// Все маршруты доступны только для аутентифицированных пользователей.
// GET /api/v1/storage/residency/violations — только admin.
//
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.9.2, IEC 62443 SR 5.1
func (s *Server) mountStorageRoutes(r chi.Router) {
	r.Route("/api/v1/storage", func(r chi.Router) {
		// Data Residency Enforcement endpoints
		r.Route("/residency", func(r chi.Router) {
			// GET /api/v1/storage/residency/status — статус data residency
			// Доступ: authenticated users
			r.Get("/status", s.handleResidencyStatus)

			// GET /api/v1/storage/residency/violations — список нарушений
			// Доступ: admin only
			r.Get("/violations", s.handleResidencyViolations)

			// POST /api/v1/storage/residency/validate — pre-flight проверка
			// Доступ: admin, manager, owner
			r.Post("/validate", s.handleValidateAccess)
		})
	})
}
