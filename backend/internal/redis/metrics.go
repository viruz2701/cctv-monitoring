// ═══════════════════════════════════════════════════════════════════════════
// P1-PERF.8: Redis Metrics
//
// Метрики Redis connection pool для /metrics endpoint:
//   - PoolHits — успешное получение соединения из пула
//   - PoolMisses — пул пуст, создано новое соединение
//   - PoolTimeouts — таймаут ожидания соединения
//   - TotalConns — общее количество соединений
//   - IdleConns — количество idle соединений
//   - StaleConns — количество закрытых stale соединений
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — pool monitoring)
//   - IEC 62443 SR 7.1 (Resource availability — metrics)
// ═══════════════════════════════════════════════════════════════════════════

package redis

import (
	"sync/atomic"

	goredis "github.com/redis/go-redis/v9"
)

// Metrics — атомарные счётчики для Redis connection pool.
type Metrics struct {
	PoolHits     atomic.Int64 `json:"pool_hits"`
	PoolMisses   atomic.Int64 `json:"pool_misses"`
	PoolTimeouts atomic.Int64 `json:"pool_timeouts"`
	TotalConns   atomic.Int64 `json:"total_conns"`
	IdleConns    atomic.Int64 `json:"idle_conns"`
	StaleConns   atomic.Int64 `json:"stale_conns"`
}

// MetricsSnapshot — снимок метрик и статистики пула.
type MetricsSnapshot struct {
	// Атомарные счётчики
	PoolHits     int64 `json:"pool_hits"`
	PoolMisses   int64 `json:"pool_misses"`
	PoolTimeouts int64 `json:"pool_timeouts"`
	TotalConns   int64 `json:"total_conns"`
	IdleConns    int64 `json:"idle_conns"`
	StaleConns   int64 `json:"stale_conns"`
	// Статистика от go-redis PoolStats
	Hits           uint32 `json:"hits"`
	Misses         uint32 `json:"misses"`
	Timeouts       uint32 `json:"timeouts"`
	TotalConnsPool uint32 `json:"total_conns_pool"`
	IdleConnsPool  uint32 `json:"idle_conns_pool"`
	StaleConnsPool uint32 `json:"stale_conns_pool"`
}

// NewMetrics создаёт новый экземпляр Metrics.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// Snapshot возвращает снимок метрик со статистикой пула.
func (m *Metrics) Snapshot(client *goredis.Client) MetricsSnapshot {
	snap := MetricsSnapshot{
		PoolHits:     m.PoolHits.Load(),
		PoolMisses:   m.PoolMisses.Load(),
		PoolTimeouts: m.PoolTimeouts.Load(),
		TotalConns:   m.TotalConns.Load(),
		IdleConns:    m.IdleConns.Load(),
		StaleConns:   m.StaleConns.Load(),
	}

	if client != nil {
		stats := client.PoolStats()
		if stats != nil {
			snap.Hits = stats.Hits
			snap.Misses = stats.Misses
			snap.Timeouts = stats.Timeouts
			snap.TotalConnsPool = stats.TotalConns
			snap.IdleConnsPool = stats.IdleConns
			snap.StaleConnsPool = stats.StaleConns
		}
	}

	return snap
}

// RecordHit увеличивает счётчик успешных получений соединения.
func (m *Metrics) RecordHit() {
	m.PoolHits.Add(1)
}

// RecordMiss увеличивает счётчик созданий новых соединений.
func (m *Metrics) RecordMiss() {
	m.PoolMisses.Add(1)
}

// RecordTimeout увеличивает счётчик таймаутов.
func (m *Metrics) RecordTimeout() {
	m.PoolTimeouts.Add(1)
}

// UpdateConns обновляет счётчики соединений.
func (m *Metrics) UpdateConns(total, idle, stale int64) {
	m.TotalConns.Store(total)
	m.IdleConns.Store(idle)
	m.StaleConns.Store(stale)
}

// Reset сбрасывает все атомарные счётчики (для тестов).
func (m *Metrics) Reset() {
	m.PoolHits.Store(0)
	m.PoolMisses.Store(0)
	m.PoolTimeouts.Store(0)
	m.TotalConns.Store(0)
	m.IdleConns.Store(0)
	m.StaleConns.Store(0)
}
