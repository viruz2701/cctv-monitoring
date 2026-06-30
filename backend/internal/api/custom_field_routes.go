// Package api — P2-FIELDS: Custom Fields Advanced routes.
package api

import "github.com/go-chi/chi/v5"

// mountCustomFieldRoutes монтирует все маршруты для кастомных полей.
//
// Маршруты:
//   - GET    /api/v1/custom-fields/definitions               — список определений
//   - GET    /api/v1/custom-fields/definitions/{id}          — определение по ID
//   - POST   /api/v1/custom-fields/definitions               — создать определение
//   - PUT    /api/v1/custom-fields/definitions/{id}          — обновить определение
//   - DELETE /api/v1/custom-fields/definitions/{id}          — удалить определение
//   - GET    /api/v1/custom-fields/groups                    — список групп
//   - GET    /api/v1/custom-fields/groups/{id}               — группа по ID
//   - POST   /api/v1/custom-fields/groups                    — создать группу
//   - PUT    /api/v1/custom-fields/groups/{id}               — обновить группу
//   - DELETE /api/v1/custom-fields/groups/{id}               — удалить группу
//   - GET    /api/v1/custom-fields/values/{entity_type}/{entity_id}  — значения для сущности
//   - PUT    /api/v1/custom-fields/values/{entity_type}/{entity_id}  — массовое обновление значений
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — audit trail)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
func (s *Server) mountCustomFieldRoutes(r chi.Router) {
	// Field Definitions CRUD
	r.Get("/api/v1/custom-fields/definitions", s.handleListFieldDefinitions)
	r.Get("/api/v1/custom-fields/definitions/{id}", s.handleGetFieldDefinition)
	r.Post("/api/v1/custom-fields/definitions", s.handleCreateFieldDefinition)
	r.Put("/api/v1/custom-fields/definitions/{id}", s.handleUpdateFieldDefinition)
	r.Delete("/api/v1/custom-fields/definitions/{id}", s.handleDeleteFieldDefinition)

	// Field Groups CRUD
	r.Get("/api/v1/custom-fields/groups", s.handleListFieldGroups)
	r.Get("/api/v1/custom-fields/groups/{id}", s.handleGetFieldGroup)
	r.Post("/api/v1/custom-fields/groups", s.handleCreateFieldGroup)
	r.Put("/api/v1/custom-fields/groups/{id}", s.handleUpdateFieldGroup)
	r.Delete("/api/v1/custom-fields/groups/{id}", s.handleDeleteFieldGroup)

	// Field Values (EAV)
	r.Get("/api/v1/custom-fields/values/{entity_type}/{entity_id}", s.handleGetFieldValues)
	r.Put("/api/v1/custom-fields/values/{entity_type}/{entity_id}", s.handleBulkUpdateFieldValues)
}
