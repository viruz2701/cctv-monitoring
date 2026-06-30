// Package api — Device Settings Routes.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-EDGE Block 6 — API-01: Device Settings Routes
//
// Маршруты монтируются в защищённую группу (JWT required).
// Только admin может мутировать настройки (PUT, POST).
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend) — все маршруты
//   - OWASP ASVS V3.3: RBAC (admin only для мутаций)
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.9.2.3: Privileged access management
//   - ISO 27001 A.12.4.1: Audit logging
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountDeviceSettingsRoutes регистрирует маршруты для управления настройками устройства.
//
// Маршруты:
//   GET  /api/v1/devices/{id}/settings       — получить настройки (viewer+)
//   PUT  /api/v1/devices/{id}/settings       — обновить настройки (admin only)
//   POST /api/v1/devices/{id}/settings/apply — применить настройки (admin only)
//
// RBAC:
//   - GET: доступен всем аутентифицированным пользователям
//   - PUT, POST: только admin/superadmin
func (s *Server) mountDeviceSettingsRoutes(r chi.Router) {
	// GET — получить настройки (доступен всем)
	r.Get("/api/v1/devices/{id}/settings", s.handleGetDeviceSettings)

	// PUT — обновить настройки (admin only, проверяется в handler)
	r.Put("/api/v1/devices/{id}/settings", s.handleUpdateDeviceSettings)

	// POST — применить настройки (admin only, проверяется в handler)
	r.Post("/api/v1/devices/{id}/settings/apply", s.handleApplyDeviceSettings)
}
