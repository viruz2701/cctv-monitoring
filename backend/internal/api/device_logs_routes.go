// Package api — Device Logs Routes.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-EDGE Block 6 — API-02: Device Logs Routes
//
// Маршруты монтируются в защищённую группу (JWT required).
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone 3 (Backend)
//   - OWASP ASVS V5.1: Input validation (query params)
//   - ISO 27001 A.12.4.1: Audit logging
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountDeviceLogRoutes регистрирует маршруты для логов устройства.
//
// Маршруты:
//   GET /api/v1/devices/{id}/logs — логи устройства с фильтрацией и пагинацией
//
// Query parameters:
//   - since (RFC3339): начало периода
//   - until (RFC3339): конец периода
//   - limit (int, 1-1000): количество записей
//   - offset (int, 0-10000): смещение
func (s *Server) mountDeviceLogRoutes(r chi.Router) {
	r.Get("/api/v1/devices/{id}/logs", s.handleGetDeviceLogs)
}
