package api

import (
	"encoding/json"
	"net/http"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/ws"
)

// handleWebSocket handles WebSocket connections for real-time alarm notifications.
// JWT token is passed via query parameter ?token=...
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondError(w, r, NewUnauthorizedError("token required"))
		return
	}

	claims, err := auth.ValidateJWT(token)
	if err != nil {
		respondError(w, r, NewUnauthorizedError("invalid token"))
		return
	}

	s.logger.Info("WebSocket client connected", "user_id", claims.UserID, "username", claims.Username)

	_, err = ws.ServeWs(s.wsHub, w, r)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", "error", err, "user_id", claims.UserID)
		return
	}
}

// BroadcastAlarm sends an alarm to all connected WebSocket clients.
func (s *Server) BroadcastAlarm(alarm *models.Alarm) {
	if s.wsHub == nil {
		return
	}

	data, err := json.Marshal(map[string]interface{}{
		"type":  "alarm",
		"alarm": alarm,
	})
	if err != nil {
		s.logger.Error("Failed to marshal alarm for WebSocket", "error", err)
		return
	}

	s.wsHub.Broadcast(data)
}
