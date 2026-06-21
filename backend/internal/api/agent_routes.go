// Package api — Agent/External routes: P2P, GB28181, alarms, WebSocket, external API key.
package api

import (
	"github.com/go-chi/chi/v5"
)

// mountAgentRoutes регистрирует P2P, GB28181, WebSocket маршруты.
func (s *Server) mountAgentRoutes(r chi.Router) {
	// P2P management
	r.Get("/api/v1/p2p/devices", s.listP2PDevices)
	r.Post("/api/v1/p2p/devices", s.registerP2PDevice)
	r.Get("/api/v1/p2p/status/{id}", s.getP2PDeviceStatus)
	r.Post("/api/v1/p2p/command/{id}", s.sendP2PCommand)
	r.Get("/api/v1/p2p/snapshot/{id}", s.getP2PSnapshot)

	// GB28181
	r.Post("/api/v1/gb28181/catalog/{id}", s.requestCatalog)
	r.Post("/api/v1/gb28181/ptz/{id}", s.sendPTZCommand)

	// WebSocket real-time alarms
	r.Get("/api/v1/ws/alarms", s.handleWebSocket)
}

// mountExternalAlarmRoutes регистрирует публичные alarm-эндпоинты.
func (s *Server) mountExternalAlarmRoutes(r chi.Router) {
	// P2P alarm (protected by API key, not JWT)
	r.Post("/api/v1/external/alarm/p2p", s.handleP2PAlarm)

	// Protected external alarm (JWT)
	r.Post("/api/v1/external/alarm", s.handleExternalAlarm)
}
