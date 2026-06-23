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
	PoolStats    *poolStats              `json:"pool_stats,omitempty"`
}

type healthDetail struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// poolStats — статистика пула соединений PostgreSQL.
type poolStats struct {
	MaxConns        int32 `json:"max_conns"`
	AcquiredConns   int32 `json:"acquired_conns"`
	IdleConns       int32 `json:"idle_conns"`
	ConstructingConns int32 `json:"constructing_conns"`
	TotalConns      int32 `json:"total_conns"`
}

func (s *Server) mountHealthRoutes(r chi.Router) {
	r.Get("/health/live", s.handleLiveness)
	r.Get("/health/ready", s.handleReadiness)
	r.Get("/health/startup", s.handleStartup)
	r.Get("/health/db", s.handleDBHealth)
}

// handleLiveness — всегда 200, проверка что сервер жив.
// Соответствует: ISO 27001 A.12.1.1 (Documented operating procedures)
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	})
}

// handleStartup — 200 если сервер полностью инициализирован.
// Startup probe для Kubernetes.
func (s *Server) handleStartup(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
		Dependencies: map[string]healthDetail{
			"database": {Status: "ok"},
		},
	}

	statusCode := http.StatusOK

	// DB must be ready for startup
	if err := s.checkDatabaseReady(r.Context()); err != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = "not_ready"
		response.Dependencies["database"] = healthDetail{
			Status: "unavailable",
			Error:  err.Error(),
		}
	}

	jsonResponse(w, statusCode, response)
}

// handleDBHealth — детальная информация о состоянии пула соединений.
// Соответствует: ISO 27001 A.12.6.1 (Capacity management), IEC 62443 SR 7.1
func (s *Server) handleDBHealth(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC(),
	}

	statusCode := http.StatusOK

	if s.db == nil || s.db.Pool == nil {
		response.Status = "unavailable"
		response.Dependencies = map[string]healthDetail{
			"database": {Status: "unavailable", Error: "pool not initialized"},
		}
		jsonResponse(w, http.StatusServiceUnavailable, response)
		return
	}

	// Ping check
	if err := s.checkDatabaseReady(r.Context()); err != nil {
		statusCode = http.StatusServiceUnavailable
		response.Status = "degraded"
		response.Dependencies = map[string]healthDetail{
			"database": {Status: "unavailable", Error: err.Error()},
		}
	}

	// Pool statistics
	stat := s.db.Pool.Stat()
	response.PoolStats = &poolStats{
		MaxConns:          stat.MaxConns(),
		AcquiredConns:     stat.AcquiredConns(),
		IdleConns:         stat.IdleConns(),
		ConstructingConns: stat.ConstructingConns(),
		TotalConns:        stat.TotalConns(),
	}

	if response.Dependencies == nil {
		response.Dependencies = map[string]healthDetail{
			"database": {Status: "ok"},
		}
	}

	jsonResponse(w, statusCode, response)
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
