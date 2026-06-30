// Package api — Credential Management Routes.
//
// ═══════════════════════════════════════════════════════════════════════════
// CRED-03: API Routes for Credential Management
// CRED-05: Automatic Credential Rotation
//
// Маршруты монтируются в защищённую группу (JWT required).
// Только admin может управлять credentials.
//
// Compliance:
//   - OWASP ASVS V3.3: RBAC (admin only)
//   - OWASP ASVS V5.1: Input validation
//   - ISO 27001 A.9.2.3: Privileged access management
//   - ISO 27001 A.12.4.1: Audit logging
//   - IEC 62443-3-3 SR 2.2: Password management
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountCredentialRoutes регистрирует маршруты для управления credentials.
// Все маршруты требуют JWT аутентификации и роли admin.
//
// Маршруты:
//
//	POST   /api/v1/devices/{id}/credentials          — создать credentials
//	GET    /api/v1/devices/{id}/credentials          — получить credentials
//	PUT    /api/v1/devices/{id}/credentials          — обновить credentials
//	DELETE /api/v1/devices/{id}/credentials          — удалить credentials
//
// CRED-05 (Automatic Rotation):
//
//	POST   /api/v1/devices/{id}/credentials/rotate   — принудительная ротация
//	GET    /api/v1/credentials/expiring              — список истекающих credentials
func (s *Server) mountCredentialRoutes(r chi.Router) {
	// CRED-03: Basic CRUD for credentials
	r.Route("/api/v1/devices/{id}/credentials", func(r chi.Router) {
		// POST — создать credentials
		r.Post("/", s.handleStoreCredentials)

		// GET — получить credentials (password маскируется)
		r.Get("/", s.handleGetCredentials)

		// PUT — обновить credentials
		r.Put("/", s.handleRotateCredentials)

		// DELETE — удалить credentials
		r.Delete("/", s.handleDeleteCredentials)

		// CRED-05: Принудительная ротация пароля на устройстве
		r.Post("/rotate", s.handleRotateDeviceCredentials)
	})

	// CRED-05: Список credentials с истекающим сроком
	r.Get("/api/v1/credentials/expiring", s.handleListExpiringCredentials)
}
