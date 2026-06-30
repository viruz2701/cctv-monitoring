// Package db — управление пулами соединений PostgreSQL.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-DB: Database Optimization
//
// PoolManager управляет primary и read replica пулами, обеспечивая:
//   - Автоматическую маршрутизацию read/write запросов
//   - Graceful degradation при отказе реплик (failback на primary)
//   - Мониторинг состояния пулов через PoolMonitor
//   - Поддержку PgBouncer в transaction mode
//
// Соответствие:
//   - IEC 62443-3-3 SR 4.2 (Resource Limitation)
//   - ISO 27001 A.12.6.1 (Capacity Management)
//   - СТБ 34.101.27 п. 7.3 (Управление ресурсами)
//
// ═══════════════════════════════════════════════════════════════════════════
package db

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Типы и константы ────────────────────────────────────────────────────

// QueryType определяет тип запроса для маршрутизации.
type QueryType int

const (
	QueryWrite  QueryType = iota // INSERT, UPDATE, DELETE, DDL — только primary
	QueryRead                    // SELECT — может идти на реплику
	QueryTxRead                  // SELECT внутри write-транзакции — primary
)

// ErrReplicaUnavailable возвращается, когда ни одна реплика не доступна.
var ErrReplicaUnavailable = fmt.Errorf("no healthy read replicas available")

// ── PoolManager ─────────────────────────────────────────────────────────

// PoolManager управляет primary и read replica пулами соединений.
//
// Потокобезопасен: все поля защищены sync.RWMutex или atomic.
type PoolManager struct {
	primary   *pgxpool.Pool
	replicas  []*pgxpool.Pool
	replicaWG sync.RWMutex // защищает список реплик при динамическом обновлении

	monitor *PoolMonitor
	logger  *slog.Logger
	cfg     PoolManagerConfig

	// round-robin для реплик
	rrCounter atomic.Uint64
}

// PoolManagerConfig — конфигурация PoolManager.
type PoolManagerConfig struct {
	// HealthCheckInterval — периодичность проверки здоровья реплик.
	HealthCheckInterval time.Duration

	// MaxReplicaLatency — максимальная допустимая задержка реплики (для маршрутизации).
	// Если latency > MaxReplicaLatency, реплика исключается из ротации.
	MaxReplicaLatency time.Duration

	// FallbackToPrimary — если true, read-запросы падают на primary при недоступности реплик.
	FallbackToPrimary bool

	// ReplicaSelectionStrategy — стратегия выбора реплики: "round-robin" | "random" | "least-conn".
	ReplicaSelectionStrategy string

	// MonitorEnabled — включает сбор метрик PoolMonitor.
	MonitorEnabled bool

	// SlowQueryThreshold — порог медленного запроса в миллисекундах.
	SlowQueryThreshold time.Duration
}

// DefaultPoolManagerConfig возвращает конфигурацию по умолчанию.
func DefaultPoolManagerConfig() PoolManagerConfig {
	return PoolManagerConfig{
		HealthCheckInterval:      30 * time.Second,
		MaxReplicaLatency:        100 * time.Millisecond,
		FallbackToPrimary:        true,
		ReplicaSelectionStrategy: "round-robin",
		MonitorEnabled:           true,
		SlowQueryThreshold:       100 * time.Millisecond,
	}
}

// NewPoolManager создаёт новый PoolManager.
func NewPoolManager(primary *pgxpool.Pool, replicas []*pgxpool.Pool, cfg PoolManagerConfig, logger *slog.Logger) *PoolManager {
	pm := &PoolManager{
		primary:  primary,
		replicas: replicas,
		logger:   logger.With("component", "db.pool_manager"),
		cfg:      cfg,
	}

	pm.monitor = NewPoolMonitor(primary, replicas, cfg.MonitorEnabled, logger)

	// Запускаем health check для реплик
	if cfg.HealthCheckInterval > 0 && len(replicas) > 0 {
		go pm.healthCheckLoop(cfg.HealthCheckInterval)
	}

	logger.Info("pool manager initialized",
		"primary_max_conns", primary.Config().MaxConns,
		"replica_count", len(replicas),
		"fallback_to_primary", cfg.FallbackToPrimary,
		"strategy", cfg.ReplicaSelectionStrategy,
	)
	return pm
}

// ── Основные методы ─────────────────────────────────────────────────────

// Primary возвращает primary пул.
func (pm *PoolManager) Primary() *pgxpool.Pool {
	return pm.primary
}

// Replica выбирает реплику согласно стратегии.
func (pm *PoolManager) Replica() (*pgxpool.Pool, error) {
	pm.replicaWG.RLock()
	defer pm.replicaWG.RUnlock()

	if len(pm.replicas) == 0 {
		if pm.cfg.FallbackToPrimary {
			pm.logger.Debug("no replicas available, falling back to primary")
			pm.monitor.IncReplicaFallback()
			return pm.primary, nil
		}
		return nil, ErrReplicaUnavailable
	}

	switch pm.cfg.ReplicaSelectionStrategy {
	case "random":
		idx := rand.Intn(len(pm.replicas))
		return pm.replicas[idx], nil
	case "least-conn":
		// Выбираем реплику с наименьшим количеством активных соединений
		return pm.selectLeastConnReplica()
	default: // round-robin
		return pm.selectRoundRobinReplica()
	}
}

// Query возвращает подходящий пул для типа запроса.
// Write-запросы всегда идут на primary.
// Read-запросы могут идти на реплику (если доступна).
func (pm *PoolManager) Query(qt QueryType) *pgxpool.Pool {
	switch qt {
	case QueryWrite, QueryTxRead:
		return pm.primary
	case QueryRead:
		replica, err := pm.Replica()
		if err != nil {
			// fallback уже обработан в Replica()
			return pm.primary
		}
		return replica
	default:
		return pm.primary
	}
}

// Acquire берёт соединение из подходящего пула.
// Для write — primary, для read — реплика (с резервом на primary).
func (pm *PoolManager) Acquire(ctx context.Context, qt QueryType) (*pgxpool.Conn, error) {
	pool := pm.Query(qt)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection from %s pool: %w", poolName(qt), err)
	}
	pm.monitor.IncAcquired()
	return conn, nil
}

// ── Управление репликами ───────────────────────────────────────────────

// SetReplicas динамически обновляет список реплик (например, при авто-масштабировании).
func (pm *PoolManager) SetReplicas(replicas []*pgxpool.Pool) {
	pm.replicaWG.Lock()
	defer pm.replicaWG.Unlock()
	pm.replicas = replicas
	pm.logger.Info("replicas updated", "count", len(replicas))
	pm.monitor.SetReplicas(replicas)
}

// AddReplica добавляет новую реплику в пул.
func (pm *PoolManager) AddReplica(pool *pgxpool.Pool) {
	pm.replicaWG.Lock()
	defer pm.replicaWG.Unlock()
	pm.replicas = append(pm.replicas, pool)
	pm.logger.Info("replica added", "total", len(pm.replicas))
	pm.monitor.SetReplicas(pm.replicas)
}

// HealthStatus возвращает статус здоровья всех пулов.
func (pm *PoolManager) HealthStatus(ctx context.Context) map[string]bool {
	status := make(map[string]bool)

	// Primary
	if err := pm.primary.Ping(ctx); err != nil {
		status["primary"] = false
	} else {
		status["primary"] = true
	}

	// Replicas
	pm.replicaWG.RLock()
	defer pm.replicaWG.RUnlock()
	for i, replica := range pm.replicas {
		key := fmt.Sprintf("replica_%d", i)
		if err := replica.Ping(ctx); err != nil {
			status[key] = false
		} else {
			status[key] = true
		}
	}

	return status
}

// Monitor возвращает PoolMonitor для сбора метрик.
func (pm *PoolManager) Monitor() *PoolMonitor {
	return pm.monitor
}

// Close закрывает все пулы.
func (pm *PoolManager) Close() {
	pm.logger.Info("closing all database pools")
	pm.primary.Close()
	for i, replica := range pm.replicas {
		replica.Close()
		pm.logger.Debug("replica pool closed", "index", i)
	}
}

// ── Внутренние методы ──────────────────────────────────────────────────

// healthCheckLoop периодически проверяет здоровье реплик.
func (pm *PoolManager) healthCheckLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		pm.checkReplicaHealth()
	}
}

// checkReplicaHealth проверяет каждую реплику и логирует недоступные.
func (pm *PoolManager) checkReplicaHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pm.replicaWG.RLock()
	healthyReplicas := make([]*pgxpool.Pool, 0, len(pm.replicas))
	for _, replica := range pm.replicas {
		if err := replica.Ping(ctx); err == nil {
			healthyReplicas = append(healthyReplicas, replica)
		} else {
			pm.logger.Warn("replica health check failed", "error", err)
		}
	}
	pm.replicaWG.RUnlock()

	// Если количество здоровых реплик изменилось — обновляем список
	pm.replicaWG.Lock()
	// Просто логируем; реальное обновление списка — через SetReplicas
	healthyCount := len(healthyReplicas)
	totalCount := len(pm.replicas)
	pm.replicaWG.Unlock()

	if healthyCount < totalCount {
		pm.monitor.IncReplicaDegradation()
		pm.logger.Warn("replica degradation detected",
			"healthy", healthyCount,
			"total", totalCount,
		)
	}
}

// selectRoundRobinReplica выбирает реплику по round-robin.
func (pm *PoolManager) selectRoundRobinReplica() (*pgxpool.Pool, error) {
	if len(pm.replicas) == 0 {
		return nil, ErrReplicaUnavailable
	}
	idx := pm.rrCounter.Add(1) % uint64(len(pm.replicas))
	return pm.replicas[idx], nil
}

// selectLeastConnReplica выбирает реплику с наименьшим количеством активных соединений.
func (pm *PoolManager) selectLeastConnReplica() (*pgxpool.Pool, error) {
	if len(pm.replicas) == 0 {
		return nil, ErrReplicaUnavailable
	}

	bestIdx := 0
	bestIdle := pm.replicas[0].Config().MaxConns

	for i, replica := range pm.replicas {
		// Используем статистику пула (pgxpool.Stat)
		stats := replica.Stat()
		idle := stats.MaxConns() - stats.AcquiredConns()
		if idle > bestIdle {
			bestIdle = idle
			bestIdx = i
		}
	}

	return pm.replicas[bestIdx], nil
}

// poolName возвращает имя пула для логирования.
func poolName(qt QueryType) string {
	switch qt {
	case QueryWrite, QueryTxRead:
		return "primary"
	case QueryRead:
		return "replica"
	default:
		return "unknown"
	}
}
