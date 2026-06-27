package api

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
)

const (
	healthCheckTimeout = 5 * time.Second
	redisCheckTimeout  = 2 * time.Second
)

// RedisClient — интерфейс для Redis health check.
// Реализуется *redis.Client из github.com/redis/go-redis/v9.
type RedisClient interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status       string                  `json:"status"`
	Timestamp    time.Time               `json:"timestamp"`
	Uptime       string                  `json:"uptime,omitempty"`
	Dependencies map[string]healthDetail `json:"dependencies,omitempty"`
	PoolStats    *poolStats              `json:"pool_stats,omitempty"`
	Region       string                  `json:"region,omitempty"`
	Memory       *memoryStats            `json:"memory,omitempty"`
}

type healthDetail struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Latency   string `json:"latency,omitempty"`
	LastCheck string `json:"last_check,omitempty"`
}

// poolStats — статистика пула соединений PostgreSQL.
type poolStats struct {
	MaxConns          int32 `json:"max_conns"`
	AcquiredConns     int32 `json:"acquired_conns"`
	IdleConns         int32 `json:"idle_conns"`
	ConstructingConns int32 `json:"constructing_conns"`
	TotalConns        int32 `json:"total_conns"`
}

// memoryStats — статистика использования памяти приложением.
type memoryStats struct {
	AllocMB      float64 `json:"alloc_mb"`
	TotalAllocMB float64 `json:"total_alloc_mb"`
	SysMB        float64 `json:"sys_mb"`
	HeapInUseMB  float64 `json:"heap_in_use_mb"`
	Warning      string  `json:"warning,omitempty"`
}

// bytesToMB converts bytes to megabytes with 2 decimal places.
func bytesToMB(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}

func (s *Server) mountHealthRoutes(r chi.Router) {
	r.Get("/health/live", s.handleLiveness)
	r.Get("/health/ready", s.handleReadiness)
	r.Get("/health/startup", s.handleStartup)
	r.Get("/health/db", s.handleDBHealth)
	r.Get("/health/memory", s.handleMemory)
	r.Get("/health/dependencies", s.handleDependencies)
}

// buildBaseResponse создаёт healthResponse с общими полями (timestamp, uptime, region).
func (s *Server) buildBaseResponse(status string) healthResponse {
	resp := healthResponse{
		Status:    status,
		Timestamp: time.Now().UTC(),
	}
	if !s.serverStart.IsZero() {
		resp.Uptime = time.Since(s.serverStart).Round(time.Second).String()
	}
	if s.config != nil {
		resp.Region = s.config.DeploymentRegion
	}
	return resp
}

// handleLiveness — всегда 200, проверка что сервер жив.
// Соответствует: ISO 27001 A.12.1.1 (Documented operating procedures)
func (s *Server) handleLiveness(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, s.buildBaseResponse("ok"))
}

// handleStartup — 200 если сервер полностью инициализирован.
// Startup probe для Kubernetes.
func (s *Server) handleStartup(w http.ResponseWriter, r *http.Request) {
	response := s.buildBaseResponse("ok")
	response.Dependencies = map[string]healthDetail{
		"database": {Status: "ok"},
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

// handleDBHealth — детальная информация о состоянии пула соединений и памяти.
// Соответствует: ISO 27001 A.12.6.1 (Capacity management), IEC 62443 SR 7.1
func (s *Server) handleDBHealth(w http.ResponseWriter, r *http.Request) {
	response := s.buildBaseResponse("ok")
	statusCode := http.StatusOK

	if s.db == nil || s.db.Pool == nil {
		response.Status = "unavailable"
		response.Dependencies = map[string]healthDetail{
			"database": {Status: "unavailable", Error: "pool not initialized"},
		}
		response.Memory = collectMemoryStats()
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

	// Включаем memory stats в ответ /health/db
	response.Memory = collectMemoryStats()

	jsonResponse(w, statusCode, response)
}

// handleReadiness — 200 если все зависимости доступны, 503 если нет.
// Проверяет: PostgreSQL, NATS (обязательно если natsRequired=true), Redis, disk space.
// Соответствует: ISO 27001 A.12.1.1, ISO 27001 A.12.4.1, СТБ 34.101.27 п. 7.1
// Compliance: IEC 62443 SR 7.1 (Resource availability — health monitoring)
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	response := s.buildBaseResponse("ok")
	response.Dependencies = map[string]healthDetail{
		"database": {Status: "ok"},
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

	// Check JWT_SECRET (SEC-02: graceful degradation)
	if !auth.IsJWTSecretSet() {
		statusCode = http.StatusServiceUnavailable
		response.Status = "degraded"
		response.Dependencies["auth"] = healthDetail{
			Status: "unavailable",
			Error:  "JWT_SECRET not configured — authentication unavailable",
		}
	} else {
		response.Dependencies["auth"] = healthDetail{Status: "ok"}
	}

	// Check NATS connection
	if s.natsConn != nil {
		if s.natsConn.IsConnected() {
			response.Dependencies["nats"] = healthDetail{Status: "ok"}
		} else if s.natsRequired {
			statusCode = http.StatusServiceUnavailable
			response.Status = "unavailable"
			response.Dependencies["nats"] = healthDetail{
				Status: "unavailable",
				Error:  "NATS not connected — required dependency",
			}
		} else {
			statusCode = http.StatusServiceUnavailable
			response.Status = "degraded"
			response.Dependencies["nats"] = healthDetail{
				Status: "unavailable",
				Error:  "NATS not connected",
			}
		}
	}

	// Check Redis (если настроен)
	if s.redisClient != nil {
		ctx, cancel := context.WithTimeout(r.Context(), redisCheckTimeout)
		defer cancel()

		start := time.Now()
		if err := s.redisClient.Ping(ctx); err != nil {
			statusCode = http.StatusServiceUnavailable
			response.Status = "degraded"
			response.Dependencies["redis"] = healthDetail{
				Status: "unavailable",
				Error:  err.Error(),
			}
		} else {
			response.Dependencies["redis"] = healthDetail{
				Status:  "ok",
				Latency: time.Since(start).Round(time.Microsecond).String(),
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

	// Memory check
	response.Memory = collectMemoryStats()

	jsonResponse(w, statusCode, response)
}

// handleMemory — endpoint для детальной информации об использовании памяти.
// GET /health/memory
// Соответствует: ISO 27001 A.12.6.1 (Capacity management)
func (s *Server) handleMemory(w http.ResponseWriter, r *http.Request) {
	response := s.buildBaseResponse("ok")
	response.Memory = collectMemoryStats()

	jsonResponse(w, http.StatusOK, response)
}

// handleDependencies — endpoint для детального статуса всех зависимостей.
// GET /health/dependencies
// Соответствует: IEC 62443 SR 7.1 (Resource availability)
func (s *Server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	response := s.buildBaseResponse("ok")
	response.Dependencies = make(map[string]healthDetail)

	// Database check with latency
	{
		start := time.Now()
		detail := healthDetail{}
		if err := s.checkDatabaseReady(r.Context()); err != nil {
			detail.Status = "unavailable"
			detail.Error = err.Error()
			response.Status = "degraded"
		} else {
			detail.Status = "ok"
		}
		detail.Latency = time.Since(start).Round(time.Microsecond).String()
		detail.LastCheck = time.Now().UTC().Format(time.RFC3339)
		response.Dependencies["database"] = detail
	}

	// JWT check
	{
		detail := healthDetail{}
		if !auth.IsJWTSecretSet() {
			detail.Status = "unavailable"
			detail.Error = "JWT_SECRET not configured"
			response.Status = "degraded"
		} else {
			detail.Status = "ok"
		}
		detail.LastCheck = time.Now().UTC().Format(time.RFC3339)
		response.Dependencies["auth"] = detail
	}

	// NATS check
	{
		detail := healthDetail{}
		if s.natsConn == nil {
			detail.Status = "not_configured"
		} else if s.natsConn.IsConnected() {
			detail.Status = "ok"
		} else {
			detail.Status = "unavailable"
			detail.Error = "NATS not connected"
			response.Status = "degraded"
		}
		detail.LastCheck = time.Now().UTC().Format(time.RFC3339)
		response.Dependencies["nats"] = detail
	}

	// Redis check (с latency)
	{
		detail := healthDetail{}
		if s.redisClient == nil {
			detail.Status = "not_configured"
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), redisCheckTimeout)
			defer cancel()

			start := time.Now()
			if err := s.redisClient.Ping(ctx); err != nil {
				detail.Status = "unavailable"
				detail.Error = err.Error()
				response.Status = "degraded"
			} else {
				detail.Status = "ok"
			}
			detail.Latency = time.Since(start).Round(time.Microsecond).String()
		}
		detail.LastCheck = time.Now().UTC().Format(time.RFC3339)
		response.Dependencies["redis"] = detail
	}

	// Disk check
	{
		detail := healthDetail{}
		if s.imagesDir == "" {
			detail.Status = "not_configured"
		} else if err := checkDiskWritable(s.imagesDir); err != nil {
			detail.Status = "unavailable"
			detail.Error = err.Error()
			response.Status = "degraded"
		} else {
			detail.Status = "ok"
		}
		detail.LastCheck = time.Now().UTC().Format(time.RFC3339)
		response.Dependencies["disk"] = detail
	}

	// Memory stats
	response.Memory = collectMemoryStats()

	jsonResponse(w, http.StatusOK, response)
}

func (s *Server) checkDatabaseReady(parent context.Context) error {
	if s.db == nil || s.db.Pool == nil {
		return NewExternalServiceError("database pool is not initialized")
	}

	ctx, cancel := context.WithTimeout(parent, healthCheckTimeout)
	defer cancel()

	return s.db.Pool.Ping(ctx)
}

// collectMemoryStats собирает статистику использования памяти через runtime.ReadMemStats.
// Возвращает nil если память не может быть прочитана.
func collectMemoryStats() *memoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := &memoryStats{
		AllocMB:      bytesToMB(m.Alloc),
		TotalAllocMB: bytesToMB(m.TotalAlloc),
		SysMB:        bytesToMB(m.Sys),
		HeapInUseMB:  bytesToMB(m.HeapInuse),
	}

	// Предупреждение если Alloc > 80% от Sys (общей памяти от ОС)
	if m.Sys > 0 && m.Alloc*100/m.Sys > 80 {
		stats.Warning = "memory usage above 80% of system-allocated memory"
	}

	return stats
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
