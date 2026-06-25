// Package api — недостающие HTTP-обработчики для фронтенда.
//
// Соответствует:
//   - OWASP ASVS L3 V1-V17 (полный спектр контролей)
//   - ISO 27001 A.9.2 (RBAC), A.12.4 (Audit)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
package api

import (
	"net/http"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
)

// handleListAlarms возвращает список тревог.
// GET /api/v1/alarms?device_id=xxx
// Если device_id не указан — возвращает все тревоги.
// При ошибке БД возвращает пустой массив (graceful degradation).
func (s *Server) handleListAlarms(w http.ResponseWriter, r *http.Request) {
	deviceID := r.URL.Query().Get("device_id")

	alarms, err := s.db.GetAlarms(deviceID)
	if err != nil {
		s.logger.Warn("GetAlarms query failed, returning empty array", "error", err)
		jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}

	if alarms == nil {
		alarms = []models.Alarm{} // заменяем nil на пустой массив
	}
	jsonResponse(w, http.StatusOK, alarms)
}

// handleListTickets возвращает список тикетов.
// GET /api/v1/tickets
// При ошибке БД возвращает пустой массив (graceful degradation).
func (s *Server) handleListTickets(w http.ResponseWriter, r *http.Request) {
	tickets, err := s.db.GetTickets()
	if err != nil {
		s.logger.Warn("GetTickets query failed, returning empty array", "error", err)
		jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}

	if tickets == nil {
		tickets = []db.Ticket{} // заменяем nil на пустой массив
	}
	jsonResponse(w, http.StatusOK, tickets)
}

// handleListNotifications возвращает список уведомлений.
// GET /api/v1/notifications
// При ошибке БД возвращает пустой массив (graceful degradation).
func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	notifications, err := s.db.GetNotifications()
	if err != nil {
		s.logger.Warn("GetNotifications query failed, returning empty array", "error", err)
		jsonResponse(w, http.StatusOK, []interface{}{})
		return
	}

	if notifications == nil {
		notifications = []db.Notification{} // заменяем nil на пустой массив
	}
	jsonResponse(w, http.StatusOK, notifications)
}
