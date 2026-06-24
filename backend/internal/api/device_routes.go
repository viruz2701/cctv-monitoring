// Package api — Device domain routes: CRUD, status, images, analytics, logs, audit.
//
// Соответствует:
//   - OWASP ASVS L3 V1-V17 (полный спектр контролей)
//   - ISO 27001 A.9.2 (RBAC), A.12.4 (Audit)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountDeviceRoutes регистрирует device-маршруты.
// Маршруты защищены AuthMiddleware (вызывается из server.go).
func (s *Server) mountDeviceRoutes(r chi.Router) {
	// ── CRUD (с OWASP ASVS L3) ───────────────────────────────────────
	// [x] V4 — Access Control (RBAC в хендлерах)
	// [x] V5 — Validation (whitelist через Validator)
	// [x] V7 — Error Handling (через respondError)
	r.Post("/api/v1/devices", s.handleCreateDevice)         // C — Create
	r.Get("/api/v1/devices", s.handleListDevices)           // R — List (с пагинацией)
	r.Get("/api/v1/devices/{id}", s.handleGetDevice)        // R — Read
	r.Put("/api/v1/devices/{id}", s.handleUpdateDevice)     // U — Update
	r.Delete("/api/v1/devices/{id}", s.handleDeleteDevice)  // D — Delete (soft)
	r.Post("/api/v1/devices/{id}/restore", s.handleRestoreDevice) // Restore

	// ── Status ───────────────────────────────────────────────────────
	r.Get("/api/v1/devices/{id}/status", s.getDeviceStatus)

	// ── Изображения ──────────────────────────────────────────────────
	r.Get("/api/v1/images/{filename}", s.getImage)
	r.Get("/api/v1/images/device/{deviceId}", s.listDeviceImages)

	// ── Аналитика ────────────────────────────────────────────────────
	r.Get("/api/v1/analytics/predictions", s.getPredictions)
	r.Get("/api/v1/analytics/reliability", s.getReliability)
	r.Get("/api/v1/analytics/tco", s.getTCOPerDevice)
	r.Get("/api/v1/analytics/wo-costs", s.getWorkOrderCosts) // WO-4.4.5

	// ── Логи ─────────────────────────────────────────────────────────
	r.Get("/api/v1/logs/search", s.searchLogs)

	// ── Audit (ISO 27001 A.12.4) ─────────────────────────────────────
	r.Get("/api/v1/audit/verify", s.handleAuditVerify)
}
