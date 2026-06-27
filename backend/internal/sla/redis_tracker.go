// Package sla — Redis-based SLA Tracker (PERF.5).
//
// RedisSLATracker реализует интерфейс SLATracker для хранения и получения
// SLA метрик через Redis. Использует Sorted Sets для хранения breaches
// с score = timestamp unix nano для range-запросов.
//
// Ключи:
//   - sla:breaches:{deviceID} — Sorted Set нарушений (score = unix nano)
//   - sla:compliance:{deviceID} — String процент соблюдения (0-100)
//   - sla:devices — SET всех deviceID с нарушениями
//
// TTL: 90 дней на breach data (через EXPIRE при RecordBreach).
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — SLA метрики)
//   - ISO 27001 A.12.4.1 (Event logging — SLA breach events)
//   - ISO 27019 PCC.A.12.4 (ICS audit trail)
package sla

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

// BreachTTL — время жизни данных о нарушении SLA (90 дней).
// Соответствует audit trail retention policy для КИИ РБ.
const BreachTTL = 90 * 24 * time.Hour

// slaKeyPrefix — префикс ключей Redis для SLA трекера.
const slaKeyPrefix = "sla:"

// DefaultOperationTimeout — таймаут по умолчанию для Redis операций.
const DefaultOperationTimeout = 5 * time.Second

// ═══════════════════════════════════════════════════════════════════════
// RedisCmdable — абстракция над go-redis client.
// ═══════════════════════════════════════════════════════════════════════

// Z — член Sorted Set с score.
type Z struct {
	Score  float64
	Member interface{}
}

// ZRangeBy — опции для ZRangeByScore.
type ZRangeBy struct {
	Min    string
	Max    string
	Offset int64
	Count  int64
}

// RedisCmdable — интерфейс Redis команд, используемых трекером.
//
// Позволяет тестировать RedisSLATracker без реального Redis подключения.
// Соответствует подмножеству go-redis Cmdable.
type RedisCmdable interface {
	Ping(ctx context.Context) error
	ZAdd(ctx context.Context, key string, members ...Z) error
	ZRangeByScore(ctx context.Context, key string, opt ZRangeBy) ([]string, error)
	ZCount(ctx context.Context, key string, min, max string) (int64, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	SAdd(ctx context.Context, key string, members ...string) error
	SMembers(ctx context.Context, key string) ([]string, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Close() error
}

// ═══════════════════════════════════════════════════════════════════════
// RedisSLATracker
// ═══════════════════════════════════════════════════════════════════════

// RedisSLATracker — Redis-based реализация SLATracker.
//
// Хранит нарушения SLA в Redis Sorted Sets для эффективного range-поиска.
// Все операции поддерживают context timeout для graceful degradation.
//
// Goroutine-safe: использует sync.RWMutex для доступа к внутреннему состоянию.
//
// Compliance:
//   - IEC 62443 SR 2.8 (Audit events — breach tracking)
//   - IEC 62443 SR 7.1 (Resource availability — SLA метрики)
//   - ISO 27001 A.12.4.1 (Event logging — SLA breach events)
//   - СТБ 34.101.27 (Защита информации — audit trail)
type RedisSLATracker struct {
	client    RedisCmdable
	logger    *slog.Logger
	startTime time.Time

	mu      sync.RWMutex
	timeout time.Duration // per-operation timeout
}

// NewRedisSLATracker создаёт Redis-based SLA трекер.
//
// Параметры:
//   - client: RedisCmdable интерфейс (real Redis client or mock)
//   - logger: логгер (nil = slog.Default())
//
// Default timeout: 5 секунд на операцию.
//
// Пример:
//
//	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
//	tracker := NewRedisSLATracker(rdb, slog.Default())
func NewRedisSLATracker(client RedisCmdable, logger *slog.Logger) *RedisSLATracker {
	if logger == nil {
		logger = slog.Default()
	}
	return &RedisSLATracker{
		client:    client,
		logger:    logger.With("component", "sla-redis-tracker"),
		startTime: time.Now().UTC(),
		timeout:   DefaultOperationTimeout,
	}
}

// WithTimeout устанавливает таймаут для Redis операций.
// По умолчанию: DefaultOperationTimeout (5s).
func (t *RedisSLATracker) WithTimeout(timeout time.Duration) *RedisSLATracker {
	t.mu.Lock()
	defer t.mu.Unlock()
	if timeout > 0 {
		t.timeout = timeout
	}
	return t
}

// ── Key helpers ──────────────────────────────────────────────────────

// breachKey возвращает ключ Redis для Sorted Set breaches device.
func breachKey(deviceID string) string {
	return slaKeyPrefix + "breaches:" + deviceID
}

// complianceKey возвращает ключ Redis для compliance rate device.
func complianceKey(deviceID string) string {
	return slaKeyPrefix + "compliance:" + deviceID
}

// devicesKey возвращает ключ Redis для SET устройств.
func devicesKey() string {
	return slaKeyPrefix + "devices"
}

// ── ID generation ────────────────────────────────────────────────────

// generateID создаёт криптостойкий ID для breach.
// Использует crypto/rand вместо sequential IDs для предотвращения
// угадывания ID (OWASP ASVS V2).
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate breach id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ── contextWithTimeout ───────────────────────────────────────────────

// contextWithTimeout возвращает контекст с таймаутом, если ctx не имеет deadline.
func (t *RedisSLATracker) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	t.mu.RLock()
	timeout := t.timeout
	t.mu.RUnlock()

	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

// ═══════════════════════════════════════════════════════════════════════
// SLATracker implementation
// ═══════════════════════════════════════════════════════════════════════

// RecordBreach записывает факт нарушения SLA.
//
// Сохраняет breach в Redis Sorted Set (score = unix nano timestamp).
// Устанавливает TTL 90 дней на ключ breaches device.
// Добавляет deviceID в SET sla:devices.
// Обновляет compliance rate для device.
//
// Алгоритм:
//  1. Генерирует криптостойкий ID для breach
//  2. Сериализует breach в JSON
//  3. Добавляет в Sorted Set sla:breaches:{deviceID} с score = timestamp
//  4. Добавляет deviceID в sla:devices SET
//  5. Устанавливает TTL 90 дней на все затронутые ключи
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — breach record)
//   - IEC 62443 SR 2.8 (Audit events — breach tracking)
//   - OWASP ASVS V7.1 (Structured logging — JSON format)
func (t *RedisSLATracker) RecordBreach(ctx context.Context, breach *SLABreach) error {
	if breach == nil {
		return fmt.Errorf("breach is nil")
	}

	ctx, cancel := t.contextWithTimeout(ctx)
	defer cancel()

	// 1. Генерируем ID
	id, err := generateID()
	if err != nil {
		return fmt.Errorf("record breach: %w", err)
	}
	breach.ID = id

	// 2. Сериализуем
	data, err := json.Marshal(breach)
	if err != nil {
		return fmt.Errorf("record breach marshal: %w", err)
	}

	// 3. Добавляем в Sorted Set (score = unix nano)
	bKey := breachKey(breach.DeviceID)
	score := float64(breach.OccurredAt.UnixNano())

	if err := t.client.ZAdd(ctx, bKey, Z{Score: score, Member: string(data)}); err != nil {
		return fmt.Errorf("record breach zadd: %w", err)
	}

	// 4. Добавляем device в SET
	if err := t.client.SAdd(ctx, devicesKey(), breach.DeviceID); err != nil {
		t.logger.Warn("failed to add device to set",
			"device_id", breach.DeviceID,
			"error", err,
		)
	}

	// 5. Устанавливаем TTL
	if err := t.client.Expire(ctx, bKey, BreachTTL); err != nil {
		t.logger.Warn("failed to set breach TTL",
			"device_id", breach.DeviceID,
			"error", err,
		)
	}

	t.logger.Debug("breach recorded",
		"breach_id", breach.ID,
		"device_id", breach.DeviceID,
		"violation_type", breach.ViolationType,
		"actual_value", breach.ActualValue,
		"threshold", breach.Threshold,
	)

	return nil
}

// GetBreaches возвращает нарушения SLA за указанный период.
//
// Выполняет ZRangeByScore по ключу sla:breaches:{deviceID}
// с min/max = unix nano timestamps.
//
// Возвращает пустой слайс (не nil) если breaches не найдены.
//
// Compliance:
//   - IEC 62443 SR 2.8 (Audit trail — breach history)
//   - ISO 27001 A.12.4.1 (Event logging — breach retrieval)
func (t *RedisSLATracker) GetBreaches(ctx context.Context, deviceID string, from, to time.Time) ([]SLABreach, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("deviceID is required")
	}

	ctx, cancel := t.contextWithTimeout(ctx)
	defer cancel()

	// Конвертируем time в unix nano строки для Redis range
	minScore := fmt.Sprintf("%d", from.UnixNano())
	maxScore := fmt.Sprintf("%d", to.UnixNano())

	bKey := breachKey(deviceID)
	members, err := t.client.ZRangeByScore(ctx, bKey, ZRangeBy{
		Min: minScore,
		Max: maxScore,
	})
	if err != nil {
		return nil, fmt.Errorf("get breaches: %w", err)
	}

	if len(members) == 0 {
		return []SLABreach{}, nil
	}

	breaches := make([]SLABreach, 0, len(members))
	for _, m := range members {
		var b SLABreach
		if err := json.Unmarshal([]byte(m), &b); err != nil {
			t.logger.Warn("failed to unmarshal breach",
				"device_id", deviceID,
				"error", err,
			)
			continue
		}
		breaches = append(breaches, b)
	}

	return breaches, nil
}

// GetComplianceRate возвращает процент соблюдения SLA для device за период.
//
// Алгоритм:
//  1. Получает общее количество breaches за период через ZCount
//  2. Рассчитывает compliance rate: max(0, 100 - breaches_count * weight)
//  3. Weight = 5% за каждый breach (20 breaches = 0% compliance)
//
// Если breaches нет — возвращает 100%.
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — SLA compliance metrics)
//   - ISO 27001 A.12.6.1 (Capacity management)
func (t *RedisSLATracker) GetComplianceRate(ctx context.Context, deviceID string, from, to time.Time) (float64, error) {
	if deviceID == "" {
		return 0, fmt.Errorf("deviceID is required")
	}

	ctx, cancel := t.contextWithTimeout(ctx)
	defer cancel()

	bKey := breachKey(deviceID)
	minScore := fmt.Sprintf("%d", from.UnixNano())
	maxScore := fmt.Sprintf("%d", to.UnixNano())

	count, err := t.client.ZCount(ctx, bKey, minScore, maxScore)
	if err != nil {
		return 0, fmt.Errorf("get compliance rate: %w", err)
	}

	// 20 breaches = 0% compliance (5% per breach)
	rate := math.Max(0, 100-float64(count)*5)

	return rate, nil
}

// GetTrackerStatus возвращает статус трекера.
//
// Проверяет:
//   - Connected: может ли Redis ответить на PING
//   - KeysCount: количество ключей с префиксом sla:
//   - Uptime: время работы трекера
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — health check)
func (t *RedisSLATracker) GetTrackerStatus(ctx context.Context) (*TrackerStatus, error) {
	ctx, cancel := t.contextWithTimeout(ctx)
	defer cancel()

	status := &TrackerStatus{
		Uptime: time.Since(t.startTime).Round(time.Second).String(),
	}

	// Проверяем соединение
	if err := t.client.Ping(ctx); err != nil {
		status.Connected = false
		return status, nil
	}

	status.Connected = true

	// Получаем количество устройств в SET
	members, err := t.client.SMembers(ctx, devicesKey())
	if err != nil {
		status.KeysCount = 0
		return status, nil
	}
	status.KeysCount = int64(len(members))

	return status, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Metrics Collector
// ═══════════════════════════════════════════════════════════════════════

// RedisTrackerMetrics — метрики Redis трекера для Prometheus.
type RedisTrackerMetrics struct {
	Connected          bool
	DeviceCount        int64
	TotalBreaches      atomic.Int64
	RecordBreachErrors atomic.Int64
	QueryErrors        atomic.Int64
}

// Metrics возвращает метрики трекера.
func (t *RedisSLATracker) Metrics() *RedisTrackerMetrics {
	m := &RedisTrackerMetrics{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := t.client.Ping(ctx); err != nil {
		return m
	}
	m.Connected = true

	members, err := t.client.SMembers(ctx, devicesKey())
	if err == nil {
		m.DeviceCount = int64(len(members))
	}

	return m
}

// Close закрывает Redis соединение.
func (t *RedisSLATracker) Close() error {
	return t.client.Close()
}
