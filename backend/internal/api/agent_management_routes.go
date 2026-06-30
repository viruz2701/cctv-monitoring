// Package api — Agent Management Routes.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-EDGE Block 6 — API-03: Agent Management Routes
//
// Маршруты монтируются в защищённую группу (JWT required).
// Только admin может мутировать агентов (POST, DELETE).
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend), SR 1.1 (Defense in depth)
//   - OWASP ASVS V3.3: RBAC (admin for mutations)
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.9.2.3: Privileged access management
//   - ISO 27001 A.12.4.1: Audit logging
//   - Приказ ОАЦ №66 п. 7.18: mTLS для agent communication
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountAgentManagementRoutes регистрирует маршруты для управления агентами.
//
// Маршруты:
//   GET    /api/v1/agents              — список всех агентов
//   GET    /api/v1/agents/{id}         — детали агента
//   POST   /api/v1/agents/{id}/command — отправить команду агенту (admin only)
//   DELETE /api/v1/agents/{id}         — удалить агента (admin only)
//
// RBAC:
//   - GET: доступен всем аутентифицированным пользователям
//   - POST, DELETE: только admin/superadmin (проверяется в handler)
func (s *Server) mountAgentManagementRoutes(r chi.Router) {
	// GET — список агентов
	r.Get("/api/v1/agents", s.handleListAgents)

	// GET — детали агента
	r.Get("/api/v1/agents/{id}", s.handleGetAgent)

	// POST — отправить команду (admin only)
	r.Post("/api/v1/agents/{id}/command", s.handleSendAgentCommand)

	// DELETE — удалить агента (admin only)
	r.Delete("/api/v1/agents/{id}", s.handleDeleteAgent)
}
