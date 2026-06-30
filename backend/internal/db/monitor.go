// Package db — мониторинг пулов соединений PostgreSQL.
//
// ═══════════════════════════════════════════════════════════════════════════
// P3-DB: Database Optimization — PoolMonitor
//
// PoolMonitor собирает метрики пулов соединений и экспортирует их
// для Prometheus. Метрики включают активные/idle соединения, ожидание,
// latency, и статистику по репликам.
//
// Соответствие:
//   - ISO 27001 A.12.6.1 (Capacity Management)
//   - IEC 62443-3-3 SR 4.2 (Resource Limitation)
//   - IEC 62443-3-3 SR 7.2 (Performance Monitoring)
//
// ═══════════════════════════════════════════════════════════════════════════
package db

import (
	"log/slog"
	"math"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── PoolMonitor ─────────────────────────────────────────────────────────

// PoolMonitor собирает и экспортирует метрики пулов соединений.
//
// Потокобезопасен: все счётчики — atomic, статистика читается через Stat().
type PoolMonitor struct {
	primary  *pgxpool.Pool
	replicas []*pgxpool.Pool

	enabled bool
	logger  *slog.Logger

	// ── Счётчики (atomic) ────────────────────────────────────────────

	// Активные/idle соединения — агрегированные по всем пулам
	activeConns atomic.Int64
	idleConns   atomic.Int64

	// Статистика ожидания
	waitCount    atomic.Int64
	waitDuration atomic.Int64 // наносекунды, для точности

	// Пиковые значения
	maxConnsReached atomic.Int32

	// Счётчики событий
	totalAcquired      atomic.Uint64
	totalReleased      atomic.Uint64
	replicaFallbacks   atomic.Uint64 // сколько раз упали на primary
	replicaDegradation atomic.Uint64 // сколько раз реплика была недоступна
	slowQueryCount     atomic.Uint64 // количество медленных запросов

	// Latency гистограмма (p50, p95, p99)
	latencyP50 atomic.Int64
	latencyP95 atomic.Int64
	latencyP99 atomic.Int64
}

// PoolStats — снапшот метрик на момент запроса.
type PoolStats struct {
	// Primary
	PrimaryMaxConns     int32 `json:"primary_max_conns"`
	PrimaryAcquired     int32 `json:"primary_acquired"`
	PrimaryConstructing int32 `json:"primary_constructing"`
	PrimaryIdle         int32 `json:"primary_idle"`

	// Replicas (агрегированные)
	ReplicaCount    int   `json:"replica_count"`
	ReplicaMaxConns int32 `json:"replica_max_conns"`
	ReplicaAcquired int32 `json:"replica_acquired"`
	ReplicaIdle     int32 `json:"replica_idle"`

	// Глобальные
	ActiveConns         int64   `json:"active_conns"`
	IdleConns           int64   `json:"idle_conns"`
	WaitCount           int64   `json:"wait_count"`
	WaitDurationMs      float64 `json:"wait_duration_ms"`
	MaxConnsReached     int32   `json:"max_conns_reached"`
	TotalAcquired       uint64  `json:"total_acquired"`
	TotalReleased       uint64  `json:"total_released"`
	ReplicaFallbacks    uint64  `json:"replica_fallbacks"`
	ReplicaDegradations uint64  `json:"replica_degradations"`
	SlowQueryCount      uint64  `json:"slow_query_count"`

	// Latency (миллисекунды)
	LatencyP50Ms float64 `json:"latency_p50_ms"`
	LatencyP95Ms float64 `json:"latency_p95_ms"`
	LatencyP99Ms float64 `json:"latency_p99_ms"`

	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// NewPoolMonitor создаёт новый PoolMonitor.
func NewPoolMonitor(primary *pgxpool.Pool, replicas []*pgxpool.Pool, enabled bool, logger *slog.Logger) *PoolMonitor {
	pm := &PoolMonitor{
		primary:  primary,
		replicas: replicas,
		enabled:  enabled,
		logger:   logger.With("component", "db.pool_monitor"),
	}

	// Запускаем сбор метрик
	if enabled {
		go pm.collectLoop()
	}

	return pm
}

// ── Сбор метрик ─────────────────────────────────────────────────────────

// collectLoop периодически собирает метрики из пулов.
func (pm *PoolMonitor) collectLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pm.collect()
	}
}

// collect собирает текущие метрики из pgxpool.Stat.
func (pm *PoolMonitor) collect() {
	if !pm.enabled {
		return
	}

	// Primary статистика
	primaryStat := pm.primary.Stat()
	pm.activeConns.Store(int64(primaryStat.AcquiredConns()))
	pm.idleConns.Store(int64(primaryStat.IdleConns()))
	pm.waitCount.Store(int64(primaryStat.TotalConns()))           // совместимость
	pm.waitDuration.Store(int64(primaryStat.EmptyAcquireCount())) // совместимость

	// Проверка достижения максимума
	maxConns := primaryStat.MaxConns()
	if primaryStat.AcquiredConns() >= maxConns {
		pm.maxConnsReached.Store(maxConns)
	}

	// Агрегируем реплики
	var totalReplicaAcquired, totalReplicaIdle int32
	for _, replica := range pm.replicas {
		stat := replica.Stat()
		totalReplicaAcquired += stat.AcquiredConns()
		totalReplicaIdle += stat.IdleConns()
	}
}

// SetReplicas обновляет список реплик для мониторинга.
func (pm *PoolMonitor) SetReplicas(replicas []*pgxpool.Pool) {
	pm.replicas = replicas
}

// ── Инкременты (вызываются из PoolManager) ──────────────────────────────

// IncAcquired увеличивает счётчик взятых соединений.
func (pm *PoolMonitor) IncAcquired() {
	if !pm.enabled {
		return
	}
	pm.totalAcquired.Add(1)
}

// IncReleased увеличивает счётчик освобождённых соединений.
func (pm *PoolMonitor) IncReleased() {
	if !pm.enabled {
		return
	}
	pm.totalReleased.Add(1)
}

// IncReplicaFallback увеличивает счётчик fallback'ов на primary.
func (pm *PoolMonitor) IncReplicaFallback() {
	if !pm.enabled {
		return
	}
	pm.replicaFallbacks.Add(1)
}

// IncReplicaDegradation увеличивает счётчик деградаций реплик.
func (pm *PoolMonitor) IncReplicaDegradation() {
	if !pm.enabled {
		return
	}
	pm.replicaDegradation.Add(1)
}

// IncSlowQuery увеличивает счётчик медленных запросов.
func (pm *PoolMonitor) IncSlowQuery() {
	if !pm.enabled {
		return
	}
	pm.slowQueryCount.Add(1)
}

// RecordLatency записывает latency запроса для гистограммы.
func (pm *PoolMonitor) RecordLatency(d time.Duration) {
	if !pm.enabled {
		return
	}

	ms := float64(d.Microseconds()) / 1000.0

	// Экспоненциальное скользящее среднее для p50/p95/p99
	// Используем weight 0.1 для новых значений
	const weight = 0.1

	p50 := float64(pm.latencyP50.Load())
	p95 := float64(pm.latencyP95.Load())
	p99 := float64(pm.latencyP99.Load())

	if p50 == 0 {
		pm.latencyP50.Store(int64(ms * 1000))
		pm.latencyP95.Store(int64(ms * 1000))
		pm.latencyP99.Store(int64(ms * 1000))
		return
	}

	pm.latencyP50.Store(int64((p50*(1-weight) + ms*1000*weight)))
	pm.latencyP95.Store(int64((p95*(1-weight) + ms*1000*weight)))
	pm.latencyP99.Store(int64((p99*(1-weight) + ms*1000*weight)))
}

// ── Получение статистики ───────────────────────────────────────────────

// Stats возвращает текущий снапшот метрик.
func (pm *PoolMonitor) Stats() *PoolStats {
	if !pm.enabled || pm.primary == nil {
		return &PoolStats{Timestamp: time.Now()}
	}

	primaryStat := pm.primary.Stat()

	// Агрегируем реплики
	var totalReplicaMaxConns, totalReplicaAcquired, totalReplicaIdle int32
	for _, replica := range pm.replicas {
		stat := replica.Stat()
		totalReplicaMaxConns += stat.MaxConns()
		totalReplicaAcquired += stat.AcquiredConns()
		totalReplicaIdle += stat.IdleConns()
	}

	return &PoolStats{
		PrimaryMaxConns:     primaryStat.MaxConns(),
		PrimaryAcquired:     primaryStat.AcquiredConns(),
		PrimaryConstructing: primaryStat.ConstructingConns(),
		PrimaryIdle:         primaryStat.IdleConns(),

		ReplicaCount:    len(pm.replicas),
		ReplicaMaxConns: totalReplicaMaxConns,
		ReplicaAcquired: totalReplicaAcquired,
		ReplicaIdle:     totalReplicaIdle,

		ActiveConns:         pm.activeConns.Load(),
		IdleConns:           pm.idleConns.Load(),
		WaitCount:           pm.waitCount.Load(),
		WaitDurationMs:      float64(pm.waitDuration.Load()) / 1_000_000,
		MaxConnsReached:     pm.maxConnsReached.Load(),
		TotalAcquired:       pm.totalAcquired.Load(),
		TotalReleased:       pm.totalReleased.Load(),
		ReplicaFallbacks:    pm.replicaFallbacks.Load(),
		ReplicaDegradations: pm.replicaDegradation.Load(),
		SlowQueryCount:      pm.slowQueryCount.Load(),

		LatencyP50Ms: float64(pm.latencyP50.Load()) / 1000,
		LatencyP95Ms: float64(pm.latencyP95.Load()) / 1000,
		LatencyP99Ms: float64(pm.latencyP99.Load()) / 1000,

		Timestamp: time.Now().UTC(),
	}
}

// Enabled возвращает true если мониторинг включён.
func (pm *PoolMonitor) Enabled() bool {
	return pm.enabled
}

// ── Prometheus экспорт (интеграция) ────────────────────────────────────

// PrometheusMetrics возвращает метрики в формате, совместимом с Prometheus.
// Для интеграции с существующим Prometheus-экспортером.
func (pm *PoolMonitor) PrometheusMetrics() map[string]float64 {
	stats := pm.Stats()
	return map[string]float64{
		"db_primary_max_conns":      float64(stats.PrimaryMaxConns),
		"db_primary_acquired_conns": float64(stats.PrimaryAcquired),
		"db_primary_idle_conns":     float64(stats.PrimaryIdle),
		"db_replica_count":          float64(stats.ReplicaCount),
		"db_replica_acquired_conns": float64(stats.ReplicaAcquired),
		"db_replica_idle_conns":     float64(stats.ReplicaIdle),
		"db_active_conns":           float64(stats.ActiveConns),
		"db_idle_conns":             float64(stats.IdleConns),
		"db_wait_count":             float64(stats.WaitCount),
		"db_wait_duration_ms":       stats.WaitDurationMs,
		"db_replica_fallbacks":      float64(stats.ReplicaFallbacks),
		"db_replica_degradations":   float64(stats.ReplicaDegradations),
		"db_slow_query_count":       float64(stats.SlowQueryCount),
		"db_latency_p50_ms":         stats.LatencyP50Ms,
		"db_latency_p95_ms":         stats.LatencyP95Ms,
		"db_latency_p99_ms":         stats.LatencyP99Ms,
	}
}

// ── Утилиты ─────────────────────────────────────────────────────────────

// ExponentialMovingAverage вычисляет экспоненциальное скользящее среднее.
// Используется для сглаживания метрик latency.
func ExponentialMovingAverage(current, newValue, alpha float64) float64 {
	if math.IsNaN(current) || current == 0 {
		return newValue
	}
	return current*(1-alpha) + newValue*alpha
}
