// Package api — Device domain routes: device CRUD, status, images, analytics, logs.
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountDeviceRoutes регистрирует device-маршруты.
func (s *Server) mountDeviceRoutes(r chi.Router) {
	r.Get("/api/v1/devices", s.listDevices)
	r.Get("/api/v1/devices/{id}", s.getDevice)
	r.Get("/api/v1/devices/{id}/status", s.getDeviceStatus)

	// Изображения
	r.Get("/api/v1/images/{filename}", s.getImage)
	r.Get("/api/v1/images/device/{deviceId}", s.listDeviceImages)

	// Аналитика
	r.Get("/api/v1/analytics/predictions", s.getPredictions)
	r.Get("/api/v1/logs/search", s.searchLogs)

	// Audit (ISO 27001)
	r.Get("/api/v1/audit/verify", s.handleAuditVerify)
}
