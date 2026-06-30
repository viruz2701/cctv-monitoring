// Package api — P2-CHECK: Conditional Checklists routes.
package api

import "github.com/go-chi/chi/v5"

// mountChecklistRoutes монтирует все маршруты для чек-листов.
//
// Маршруты:
//   - GET    /api/v1/checklists/templates                  — список шаблонов
//   - GET    /api/v1/checklists/templates/{id}             — шаблон с items
//   - POST   /api/v1/checklists/templates                  — создать шаблон
//   - PUT    /api/v1/checklists/templates/{id}             — обновить шаблон
//   - DELETE /api/v1/checklists/templates/{id}             — удалить шаблон
//   - GET    /api/v1/work-orders/{id}/checklist            — текущий чек-лист
//   - POST   /api/v1/work-orders/{id}/checklist/start      — запустить чек-лист
//   - POST   /api/v1/work-orders/{id}/checklist/submit     — сабмит чек-листа
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - OWASP ASVS V5.1 (Input validation)
func (s *Server) mountChecklistRoutes(r chi.Router) {
	// Checklist Templates CRUD
	r.Get("/api/v1/checklists/templates", s.handleListTemplates)
	r.Get("/api/v1/checklists/templates/{id}", s.handleGetTemplate)
	r.Post("/api/v1/checklists/templates", s.handleCreateTemplate)
	r.Put("/api/v1/checklists/templates/{id}", s.handleUpdateTemplate)
	r.Delete("/api/v1/checklists/templates/{id}", s.handleDeleteTemplate)

	// Work Order Checklist lifecycle
	r.Get("/api/v1/work-orders/{id}/checklist", s.handleGetWorkOrderChecklist)
	r.Post("/api/v1/work-orders/{id}/checklist/start", s.handleStartChecklist)
	r.Post("/api/v1/work-orders/{id}/checklist/submit", s.handleSubmitChecklist)
}
