// ═══════════════════════════════════════════════════════════════════════════
// Package api — VPN Session Routes (EDGE-08)
//
// Маршруты для управления WireGuard VPN сессиями.
// Все маршруты защищены JWT аутентификацией (монтируются внутри protected group).
//
// Mutations (POST) проверяют RBAC admin/support внутри хендлеров.
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
//   - OWASP ASVS V3.3: Privilege escalation prevention
// ═══════════════════════════════════════════════════════════════════════════

package api

import (
	"github.com/go-chi/chi/v5"
)

// mountVPNSessionRoutes регистрирует маршруты VPN сессий.
//
// Все эндпоинты требуют JWT (монтируются внутри protected group).
// Мутации дополнительно проверяют RBAC внутри хендлеров.
func (s *Server) mountVPNSessionRoutes(r chi.Router) {
	// EDGE-08: WireGuard VPN Session Management
	// POST   /api/v1/vpn/sessions           — создать сессию (admin/support only)
	// GET    /api/v1/vpn/sessions           — список сессий
	// GET    /api/v1/vpn/sessions/{id}      — детали сессии
	// POST   /api/v1/vpn/sessions/{id}/revoke — закрыть сессию (admin/support only)
	// GET    /api/v1/vpn/sessions/{id}/config — скачать WG .conf файл (SELFSERV-02, self-service с ownership check)

	r.Post("/api/v1/vpn/sessions", s.handleCreateVPNSession)
	r.Get("/api/v1/vpn/sessions", s.handleListVPNSessions)
	r.Get("/api/v1/vpn/sessions/{id}", s.handleGetVPNSession)
	r.Post("/api/v1/vpn/sessions/{id}/revoke", s.handleRevokeVPNSession)
	// SELFSERV-02: Self-service скачивание WG конфига с проверкой владельца
	r.Get("/api/v1/vpn/sessions/{id}/config", s.handleSelfServiceGetVPNSessionConfig)
}
