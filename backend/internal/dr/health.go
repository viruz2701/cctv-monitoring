// Package dr — Disaster Recovery automation для CCTV Health Monitor.
//
// ═══════════════════════════════════════════════════════════════════════════════
// P3-DR: Disaster Recovery Automation
//
// Содержит:
//   - HealthMonitor — автоматические health checks (30s интервал)
//   - FailoverManager — auto-failover с admin confirm
//   - DrillRunner — quarterly drill automation
//   - RTO/RPO трекинг
//
// Compliance:
//   - ISO 27001 A.17.1 (Business continuity — DR)
//   - IEC 62443-3-3 SR 7.1 (Resource availability)
//   - Приказ ОАЦ №66 п. 7.18 (Конечные узлы КИИ)
//   - СТБ 34.101.27 п. 7.1 (Мониторинг доступности)
//
// ═══════════════════════════════════════════════════════════════════════════════
package dr

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// ──────────────────────────────────────────────────────────────────────────────
// Constants
// ──────────────────────────────────────────────────────────────────────────────

const (
	// DefaultCheckInterval — интервал health checks (30s по заданию).
	DefaultCheckInterval = 30 * time.Second

	// HealthCheckTimeout — таймаут для каждого individual check.
	HealthCheckTimeout = 5 * time.Second

	// FailoverThreshold — количество последовательных failures перед auto-failover.
	FailoverThreshold = 3

	// RecoveryThreshold — количество последовательных успешных checks для recovery.
	RecoveryThreshold = 3

	// MaxRTO — максимальный Recovery Time Objective (15 мин для КИИ-2).
	MaxRTO = 15 * time.Minute

	// MaxRPO — максимальный Recovery Point Objective (5 мин для КИИ-2).
	MaxRPO = 5 * time.Minute

	// RetentionDays — срок хранения истории health checks.
	RetentionDays = 365
)

// Регионы для DR.
const (
	RegionPrimary   = "primary"
	RegionSecondary = "secondary"
	RegionDR        = "dr"
)

// ──────────────────────────────────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────────────────────────────────

// HealthMonitor выполняет периодические health checks всех зависимостей.
//
// Соответствует:
//   - IEC 62443-3-3 SR 7.1: Continuous health monitoring
//   - ISO 27001 A.17.1: Business continuity monitoring
//   - Приказ ОАЦ №66 п. 7.18.1: Мониторинг конечных узлов
type HealthMonitor struct {
	mu            sync.RWMutex
	db            *pgxpool.Pool
	nats          *nats.Conn
	redis         *redis.Client
	logger        *slog.Logger
	interval      time.Duration
	status        HealthStatus
	history       []HealthRecord
	failureCount  int
	recoveryCount int
	stopCh        chan struct{}
	running       bool
	startTime     time.Time

	// Callbacks для оповещения о смене статуса.
	onStatusChange func(HealthStatus, HealthStatus)
}

// HealthStatus — текущий статус health check.
type HealthStatus struct {
	Region             string          `json:"region"`
	DB                 ComponentStatus `json:"db"`
	NATS               ComponentStatus `json:"nats"`
	Redis              ComponentStatus `json:"redis"`
	LastCheck          time.Time       `json:"last_check"`
	FailoverInProgress bool            `json:"failover_in_progress"`
	Overall            string          `json:"overall"` // "healthy" | "degraded" | "unavailable"
	Uptime             time.Duration   `json:"uptime"`
}

// ComponentStatus — статус отдельного компонента.
type ComponentStatus struct {
	Healthy   bool      `json:"healthy"`
	Latency   string    `json:"latency,omitempty"`
	Error     string    `json:"error,omitempty"`
	LastCheck time.Time `json:"last_check"`
}

// HealthRecord — запись health check для истории.
type HealthRecord struct {
	Timestamp    time.Time     `json:"timestamp"`
	DB           bool          `json:"db"`
	NATS         bool          `json:"nats"`
	Redis        bool          `json:"redis"`
	Overall      string        `json:"overall"`
	CheckLatency time.Duration `json:"check_latency"`
}

// HealthConfig — конфигурация HealthMonitor.
type HealthConfig struct {
	Region          string        `json:"region"`
	CheckInterval   time.Duration `json:"check_interval"`
	FailoverThresh  int           `json:"failover_threshold"`
	RecoveryThresh  int           `json:"recovery_threshold"`
	EnableAutoFail  bool          `json:"enable_auto_failover"`
	RetentionPeriod time.Duration `json:"retention_period"`
}

// DefaultHealthConfig возвращает конфигурацию по умолчанию.
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		Region:          RegionPrimary,
		CheckInterval:   DefaultCheckInterval,
		FailoverThresh:  FailoverThreshold,
		RecoveryThresh:  RecoveryThreshold,
		EnableAutoFail:  false, // По умолчанию — manual confirm
		RetentionPeriod: RetentionDays * 24 * time.Hour,
	}
}

// Store — интерфейс для персистентного хранения DR данных.
type Store interface {
	SaveHealthRecord(ctx context.Context, record *HealthRecord) error
	GetHealthHistory(ctx context.Context, limit int) ([]HealthRecord, error)
	SaveFailoverEvent(ctx context.Context, event *FailoverEvent) error
	GetFailoverHistory(ctx context.Context, limit int) ([]FailoverEvent, error)
	SaveDrillReport(ctx context.Context, report *DrillReport) error
	GetDrillHistory(ctx context.Context, limit int) ([]DrillReport, error)
}

// ──────────────────────────────────────────────────────────────────────────────
// Constructor
// ──────────────────────────────────────────────────────────────────────────────

// NewHealthMonitor создаёт новый HealthMonitor.
func NewHealthMonitor(
	db *pgxpool.Pool,
	nats *nats.Conn,
	redis *redis.Client,
	cfg HealthConfig,
	logger *slog.Logger,
) *HealthMonitor {
	if logger == nil {
		logger = slog.Default()
	}

	hm := &HealthMonitor{
		db:       db,
		nats:     nats,
		redis:    redis,
		logger:   logger.With("component", "dr.health-monitor"),
		interval: cfg.CheckInterval,
		status: HealthStatus{
			Region:  cfg.Region,
			Overall: "unknown",
		},
		history:   make([]HealthRecord, 0, 100),
		stopCh:    make(chan struct{}),
		startTime: time.Now(),
	}

	hm.status.DB = ComponentStatus{LastCheck: time.Now()}
	hm.status.NATS = ComponentStatus{LastCheck: time.Now()}
	hm.status.Redis = ComponentStatus{LastCheck: time.Now()}

	return hm
}

// OnStatusChange устанавливает callback при смене статуса.
func (hm *HealthMonitor) OnStatusChange(fn func(HealthStatus, HealthStatus)) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.onStatusChange = fn
}

// ──────────────────────────────────────────────────────────────────────────────
// Lifecycle
// ──────────────────────────────────────────────────────────────────────────────

// Start запускает периодические health checks.
func (hm *HealthMonitor) Start(ctx context.Context) {
	hm.mu.Lock()
	if hm.running {
		hm.mu.Unlock()
		return
	}
	hm.running = true
	hm.startTime = time.Now()
	hm.mu.Unlock()

	hm.logger.Info("DR health monitor started",
		"interval", hm.interval,
	)

	// Немедленный первый check.
	hm.runCheck(ctx)

	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hm.runCheck(ctx)
		case <-hm.stopCh:
			hm.logger.Info("DR health monitor stopped")
			return
		case <-ctx.Done():
			hm.logger.Info("DR health monitor context cancelled")
			return
		}
	}
}

// Stop останавливает health checks.
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	if hm.running {
		close(hm.stopCh)
		hm.running = false
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Health Check Execution
// ──────────────────────────────────────────────────────────────────────────────

// runCheck выполняет один цикл health checks.
func (hm *HealthMonitor) runCheck(ctx context.Context) {
	start := time.Now()

	prevStatus := hm.GetStatus()

	dbStatus := hm.checkDB(ctx)
	natsStatus := hm.checkNATS(ctx)
	redisStatus := hm.checkRedis(ctx)

	overall := "healthy"
	if !dbStatus.Healthy || !natsStatus.Healthy || !redisStatus.Healthy {
		overall = "degraded"
	}
	if !dbStatus.Healthy {
		overall = "unavailable"
	}

	hm.mu.Lock()
	hm.status.DB = dbStatus
	hm.status.NATS = natsStatus
	hm.status.Redis = redisStatus
	hm.status.LastCheck = time.Now()
	hm.status.Overall = overall
	hm.status.Uptime = time.Since(hm.startTime)

	record := HealthRecord{
		Timestamp:    time.Now(),
		DB:           dbStatus.Healthy,
		NATS:         natsStatus.Healthy,
		Redis:        redisStatus.Healthy,
		Overall:      overall,
		CheckLatency: time.Since(start),
	}

	hm.history = append(hm.history, record)
	if len(hm.history) > 1000 {
		hm.history = hm.history[900:]
	}

	// Обновляем счётчики failure/recovery.
	if overall != "healthy" {
		hm.failureCount++
		hm.recoveryCount = 0
	} else {
		hm.recoveryCount++
		if hm.recoveryCount >= RecoveryThreshold {
			hm.failureCount = 0
		}
	}

	currentStatus := hm.status
	hm.mu.Unlock()

	// Callback при смене статуса.
	if prevStatus.Overall != currentStatus.Overall {
		hm.logger.Warn("DR health status changed",
			"from", prevStatus.Overall,
			"to", currentStatus.Overall,
			"db", dbStatus.Healthy,
			"nats", natsStatus.Healthy,
			"redis", redisStatus.Healthy,
		)
		if hm.onStatusChange != nil {
			hm.onStatusChange(prevStatus, currentStatus)
		}
	}

	hm.logger.Debug("DR health check completed",
		"overall", overall,
		"latency", time.Since(start),
	)
}

// checkDB проверяет доступность PostgreSQL.
func (hm *HealthMonitor) checkDB(ctx context.Context) ComponentStatus {
	start := time.Now()
	status := ComponentStatus{LastCheck: time.Now()}

	if hm.db == nil {
		status.Error = "db pool not initialized"
		return status
	}

	checkCtx, cancel := context.WithTimeout(ctx, HealthCheckTimeout)
	defer cancel()

	if err := hm.db.Ping(checkCtx); err != nil {
		status.Error = fmt.Sprintf("db ping failed: %v", err)
		return status
	}

	status.Healthy = true
	status.Latency = time.Since(start).Round(time.Microsecond).String()
	return status
}

// checkNATS проверяет доступность NATS.
func (hm *HealthMonitor) checkNATS(ctx context.Context) ComponentStatus {
	start := time.Now()
	status := ComponentStatus{LastCheck: time.Now()}

	if hm.nats == nil {
		status.Error = "nats not initialized"
		return status
	}

	if !hm.nats.IsConnected() {
		status.Error = "nats not connected"
		return status
	}

	// Проверяем через flush с таймаутом.
	if err := hm.nats.FlushTimeout(HealthCheckTimeout); err != nil {
		status.Error = fmt.Sprintf("nats flush failed: %v", err)
		return status
	}

	status.Healthy = true
	status.Latency = time.Since(start).Round(time.Microsecond).String()
	return status
}

// checkRedis проверяет доступность Redis.
func (hm *HealthMonitor) checkRedis(ctx context.Context) ComponentStatus {
	start := time.Now()
	status := ComponentStatus{LastCheck: time.Now()}

	if hm.redis == nil {
		status.Error = "redis not initialized"
		return status
	}

	checkCtx, cancel := context.WithTimeout(ctx, HealthCheckTimeout)
	defer cancel()

	if err := hm.redis.Ping(checkCtx).Err(); err != nil {
		status.Error = fmt.Sprintf("redis ping failed: %v", err)
		return status
	}

	status.Healthy = true
	status.Latency = time.Since(start).Round(time.Microsecond).String()
	return status
}

// ──────────────────────────────────────────────────────────────────────────────
// Getters
// ──────────────────────────────────────────────────────────────────────────────

// GetStatus возвращает текущий статус health check.
func (hm *HealthMonitor) GetStatus() HealthStatus {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.status
}

// GetHistory возвращает историю health checks.
func (hm *HealthMonitor) GetHistory(n int) []HealthRecord {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if n <= 0 || n > len(hm.history) {
		n = len(hm.history)
	}
	result := make([]HealthRecord, n)
	copy(result, hm.history[len(hm.history)-n:])
	return result
}

// ShouldFailover проверяет, нужно ли инициировать failover.
func (hm *HealthMonitor) ShouldFailover() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.failureCount >= FailoverThreshold
}

// IsRecovered проверяет, восстановилась ли система.
func (hm *HealthMonitor) IsRecovered() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.recoveryCount >= RecoveryThreshold
}

// SetFailoverInProgress устанавливает флаг failover.
func (hm *HealthMonitor) SetFailoverInProgress(inProgress bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.status.FailoverInProgress = inProgress
}
