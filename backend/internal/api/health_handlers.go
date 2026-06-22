package api

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const healthCheckTimeout = 2 * time.Second

type healthResponse struct {
	Status       string                  `json:"status"`
	Timestamp    time.Time               `json:"timestamp"`
	Dependencies map[string]healthDetail `json:"dependencies,omitempty"`
}

type healthDetail struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func (s *Server) mountHealthRoutes(r chi.Router) {
	r.Get("/health/live", s.handleLiveness)
	r.Get("/health/ready", s.handleReadiness)
}

func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]healthDetail{
			"database": {Status: "ok"},
		},
	}

	statusCode := http.StatusOK
	if err := s.checkDatabaseReady(r.Context()); err != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = "degraded"
		response.Dependencies["database"] = healthDetail{
			Status: "unavailable",
			Error:  err.Error(),
		}
	}

	jsonResponse(w, statusCode, response)
}

func (s *Server) checkDatabaseReady(parent context.Context) error {
	if s.db == nil || s.db.Pool == nil {
		return NewExternalServiceError("database pool is not initialized")
	}

	ctx, cancel := context.WithTimeout(parent, healthCheckTimeout)
	defer cancel()

	return s.db.Pool.Ping(ctx)
}
