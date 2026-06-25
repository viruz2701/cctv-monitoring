// Package api — routes for Black Box Incident Recorder (KF-15.2.4).
//
// Маршруты защищены AuthMiddleware (вызывается из server.go).
//
// Compliance:
//   - OWASP ASVS V3 (Session management — через JWT middleware)
//   - OWASP ASVS V4 (RBAC — admin/support)
//   - ISO 27001 A.9.2 (Access control — role-based)
//   - IEC 62443-3-3 SR 7.1 (Resource availability — evidence collection)
package api

import "github.com/go-chi/chi/v5"

// mountBlackBoxRoutes регистрирует маршруты Black Box Incident Recorder.
//
// Все маршруты доступны только для admin, support.
// Соответствует: OWASP ASVS V4 (RBAC), ISO 27001 A.9.2
func (s *Server) mountBlackBoxRoutes(r chi.Router) {
	r.Route("/api/v1/blackbox", func(r chi.Router) {
		// POST /api/v1/blackbox/trigger — ручной вызов сбора доказательств
		r.Post("/trigger", s.handleTriggerIncident)

		// GET /api/v1/blackbox/reports — список отчётов
		r.Get("/reports", s.handleListReports)

		// GET /api/v1/blackbox/reports/{id} — детальный отчёт
		r.Get("/reports/{id}", s.handleGetReport)

		// GET /api/v1/blackbox/reports/{id}/export — экспорт отчёта
		r.Get("/reports/{id}/export", s.handleExportReport)

		// DELETE /api/v1/blackbox/reports/{id} — удаление (admin only)
		r.Delete("/reports/{id}", s.handleDeleteReport)
	})
}
