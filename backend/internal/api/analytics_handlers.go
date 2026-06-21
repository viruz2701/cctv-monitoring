package api

import (
	"net/http"
	"strconv"

	"gb-telemetry-collector/internal/auth"
)

// ---------- Аналитика ----------

func (s *Server) getPredictions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	deviceID := r.URL.Query().Get("device_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if claims.Role == "owner" {
		dev, ok := s.stateManager.Get(deviceID)
		if !ok || dev.OwnerID == nil || *dev.OwnerID != claims.UserID {
			respondError(w, r, NewForbiddenError("forbidden"))
			return
		}
	}
	predictions, err := s.db.GetPredictions(deviceID, limit)
	if err != nil {
		s.logger.Error("failed to get predictions", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	jsonResponse(w, http.StatusOK, predictions)
}

// ---------- Поиск логов ----------

func (s *Server) searchLogs(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims.Role != "admin" && claims.Role != "support" {
		respondError(w, r, NewForbiddenError("forbidden"))
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	level := r.URL.Query().Get("level")
	keyword := r.URL.Query().Get("keyword")
	timeFrom := r.URL.Query().Get("time_from")
	timeTo := r.URL.Query().Get("time_to")

	logs, err := s.db.SearchLogs(deviceID, level, keyword, timeFrom, timeTo)
	if err != nil {
		s.logger.Error("failed to search logs", "error", err)
		respondError(w, r, NewInternalError("internal error", nil))
		return
	}
	jsonResponse(w, http.StatusOK, logs)
}
