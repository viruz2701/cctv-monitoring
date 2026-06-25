// Package api — routes for Compliance & Fines Shield (KF-15.1.1).
//
// Маршруты защищены AuthMiddleware (вызывается из server.go).
//
// Compliance:
//   - OWASP ASVS V3 (Session management — через JWT middleware)
//   - OWASP ASVS V4 (RBAC — admin/manager/owner)
//   - ISO 27001 A.9.2 (Access control — role-based)
//   - IEC 62443-3-3 SR 2.1 (Account management)
package api

import "github.com/go-chi/chi/v5"

// mountComplianceRoutes регистрирует маршруты Compliance & Fines Shield.
//
// Все маршруты доступны только для admin, manager, owner.
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.9.2
func (s *Server) mountComplianceRoutes(r chi.Router) {
	r.Route("/api/v1/compliance", func(r chi.Router) {
		// GET /api/v1/compliance/summary — общая сводка рисков
		r.Get("/summary", s.handleComplianceSummary)

		// GET /api/v1/compliance/risks — детальные риски (device_id, site_id)
		r.Get("/risks", s.handleComplianceRisks)

		// GET /api/v1/compliance/fines — таблица штрафов
		r.Get("/fines", s.handleComplianceFines)

		// POST /api/v1/compliance/refresh — принудительное обновление
		r.Post("/refresh", s.handleComplianceRefresh)

		// POST /api/v1/compliance/calculate — вычисление риска по параметрам
		r.Post("/calculate", s.handleComplianceCalculate)
	})
}
