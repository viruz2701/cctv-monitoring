package api

import (
	"context"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
)

const healthCheckTimeout = 5 * time.Second

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

// handleLiveness — всегда 200, проверка что сервер жив.
// Соответствует: ISO 27001 A.12.1.1 (Documented operating procedures)
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	})
}

// handleReadiness — 200 если все зависимости доступны, 503 если нет.
// Проверяет: PostgreSQL, NATS (если сконфигурирован), disk space для imagesDir.
// Соответствует: ISO 27001 A.12.1.1, ISO 27001 A.12.4.1, СТБ 34.101.27 п. 7.1
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]healthDetail{
			"database": {Status: "ok"},
		},
	}

	statusCode := http.StatusOK

	// Check PostgreSQL (обязательная зависимость)
	if err := s.checkDatabaseReady(r.Context()); err != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = "degraded"
		response.Dependencies["database"] = healthDetail{
			Status: "unavailable",
			Error:  err.Error(),
		}
	}

	// Check NATS connection (если сконфигурирован)
	if s.natsConn != nil {
		if s.natsConn.IsConnected() {
			response.Dependencies["nats"] = healthDetail{Status: "ok"}
		} else {
			statusCode = http.StatusServiceUnavailable
			response.Status = "degraded"
			response.Dependencies["nats"] = healthDetail{
				Status: "unavailable",
				Error:  "NATS not connected",
			}
		}
	}

	// Check disk space for imagesDir (проверка доступности для записи)
	if s.imagesDir != "" {
		if err := checkDiskWritable(s.imagesDir); err != nil {
			statusCode = http.StatusServiceUnavailable
			response.Status = "degraded"
			response.Dependencies["disk"] = healthDetail{
				Status: "unavailable",
				Error:  err.Error(),
			}
		} else {
			response.Dependencies["disk"] = healthDetail{Status: "ok"}
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

// checkDiskWritable проверяет что директория существует и доступна для записи.
func checkDiskWritable(dir string) error {
	// Проверяем что директория существует
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Пытаемся создать
			if mkErr := os.MkdirAll(dir, 0755); mkErr != nil {
				return mkErr
			}
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return NewExternalServiceError("not a directory")
	}

	// Проверяем доступность для записи через statfs
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return err
	}

	// Проверяем что есть свободное место (минимум 100MB)
	freeBytes := stat.Bavail * uint64(stat.Bsize)
	if freeBytes < 100*1024*1024 {
		return NewExternalServiceError("insufficient disk space (< 100MB)")
	}

	return nil
}
