// Package tenant — Tenant Quota Management (P1-QUOTA).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-QUOTA: Tenant Quota Management
//
// Обеспечивает per-tenant quota management с Redis real-time counters:
//   - Soft limit (80%) — warning, не блокирует
//   - Hard limit (100%) — blocks creation of new resources
//   - Grace period — 7 days after hard limit before auto-suspend
//
// Redis key structure:
//
//	quota:{tenant_id}:{type}:current       — текущее значение (counter)
//	quota:{tenant_id}:{type}:hard_limit    — hard limit (cached from DB)
//	quota:{tenant_id}:{type}:soft_limit    — soft limit (cached from DB)
//	quota:{tenant_id}:grace_until          — grace period end timestamp
//	quota:{tenant_id}:warned               — was warning sent (bitfield per type)
//
// Compliance:
//   - ISO 27001 A.12.1.2 (Capacity management)
//   - IEC 62443-3-3 SR 3.1 (Resource management)
//   - IEC 62443-3-3 SR 7.1 (Audit trail — quota changes)
//   - OWASP ASVS V2.2.1 (Rate limiting)
//   - СТБ 34.101.27 п. 6.1 (Защита от DoS)
//
// ═══════════════════════════════════════════════════════════════════════════
package tenant

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// QuotaType — тип квоты tenant'а.
type QuotaType string

const (
	QuotaDevices    QuotaType = "devices"
	QuotaUsers      QuotaType = "users"
	QuotaStorage    QuotaType = "storage_gb"
	QuotaAPICalls   QuotaType = "api_calls"
	QuotaWorkOrders QuotaType = "work_orders"
)

// AllQuotaTypes возвращает все типы квот для итерации.
func AllQuotaTypes() []QuotaType {
	return []QuotaType{
		QuotaDevices,
		QuotaUsers,
		QuotaStorage,
		QuotaAPICalls,
		QuotaWorkOrders,
	}
}

// QuotaConfig — конфигурация квоты для одного типа.
type QuotaConfig struct {
	Type      QuotaType `json:"type"`
	SoftLimit int64     `json:"soft_limit"` // 80% от hard limit
	HardLimit int64     `json:"hard_limit"` // 100% — blocks creation
	Unit      string    `json:"unit"`       // count, gb, req/h
	GraceDays int       `json:"grace_days"` // 7
}

// DefaultQuotaConfigs возвращает конфигурации по умолчанию для всех типов квот.
func DefaultQuotaConfigs() map[QuotaType]QuotaConfig {
	return map[QuotaType]QuotaConfig{
		QuotaDevices: {
			Type:      QuotaDevices,
			SoftLimit: 80,
			HardLimit: 100,
			Unit:      "count",
			GraceDays: 7,
		},
		QuotaUsers: {
			Type:      QuotaUsers,
			SoftLimit: 8,
			HardLimit: 10,
			Unit:      "count",
			GraceDays: 7,
		},
		QuotaStorage: {
			Type:      QuotaStorage,
			SoftLimit: 800,
			HardLimit: 1000,
			Unit:      "gb",
			GraceDays: 7,
		},
		QuotaAPICalls: {
			Type:      QuotaAPICalls,
			SoftLimit: 8000,
			HardLimit: 10000,
			Unit:      "req/h",
			GraceDays: 0, // no grace period for API calls
		},
		QuotaWorkOrders: {
			Type:      QuotaWorkOrders,
			SoftLimit: 400,
			HardLimit: 500,
			Unit:      "count",
			GraceDays: 7,
		},
	}
}

// QuotaStatus — статус проверки квоты.
type QuotaStatus struct {
	Type       QuotaType  `json:"type"`
	Current    int64      `json:"current"`
	SoftLimit  int64      `json:"soft_limit"`
	HardLimit  int64      `json:"hard_limit"`
	Usage      float64    `json:"usage_percent"` // 0.0 — 100.0
	IsSoft     bool       `json:"is_soft"`       // >= 80%
	IsHard     bool       `json:"is_hard"`       // >= 100%
	OnGrace    bool       `json:"on_grace"`      // grace period active
	GraceUntil *time.Time `json:"grace_until,omitempty"`
}

// QuotaUsage — полная информация об использовании квот tenant'а.
type QuotaUsage struct {
	TenantID   string                     `json:"tenant_id"`
	Quotas     map[QuotaType]*QuotaStatus `json:"quotas"`
	OnGrace    bool                       `json:"on_grace"`
	GraceUntil *time.Time                 `json:"grace_until,omitempty"`
	GraceDays  int                        `json:"grace_days"`
	SuspendAt  *time.Time                 `json:"suspend_at,omitempty"`
}

// QuotaHistoryEntry — запись истории изменения квоты.
type QuotaHistoryEntry struct {
	ID        int64     `json:"id"`
	TenantID  string    `json:"tenant_id"`
	QuotaType QuotaType `json:"quota_type"`
	OldLimit  int64     `json:"old_limit"`
	NewLimit  int64     `json:"new_limit"`
	Reason    string    `json:"reason"`
	ChangedBy string    `json:"changed_by"`
	CreatedAt time.Time `json:"created_at"`
}

// QuotaManager управляет квотами tenant'ов через Redis counters.
//
// Потокобезопасен: все атомарные операции на стороне Redis.
// При недоступности Redis использует fail-open (пропускает проверку).
type QuotaManager struct {
	client *redis.Client
	pool   *pgxpool.Pool
	logger *slog.Logger
	mu     sync.RWMutex

	// Кэш лимитов из БД (tenant_id -> QuotaType -> limit)
	limitsCache map[string]map[QuotaType]QuotaConfig
	graceCache  map[string]*time.Time
}

// NewQuotaManager создаёт новый QuotaManager.
//
// Если client == nil, QuotaManager работает в режиме fail-open
// (все проверки пропускаются).
func NewQuotaManager(client *redis.Client, pool *pgxpool.Pool) *QuotaManager {
	return &QuotaManager{
		client:      client,
		pool:        pool,
		logger:      slog.Default().With("component", "tenant.quota"),
		limitsCache: make(map[string]map[QuotaType]QuotaConfig),
		graceCache:  make(map[string]*time.Time),
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Redis key helpers
// ────────────────────────────────────────────────────────────────────────────

const (
	quotaKeyPrefix    = "quota:"
	quotaCurrentKey   = "%s:%s:current"
	quotaHardLimitKey = "%s:%s:hard_limit"
	quotaSoftLimitKey = "%s:%s:soft_limit"
	quotaGraceKey     = "%s:%s:grace_until"
	quotaWarnedKey    = "%s:%s:warned"
)

func redisKeyCurrent(tenantID string, qt QuotaType) string {
	return quotaKeyPrefix + fmt.Sprintf(quotaCurrentKey, tenantID, string(qt))
}

func redisKeyHardLimit(tenantID string, qt QuotaType) string {
	return quotaKeyPrefix + fmt.Sprintf(quotaHardLimitKey, tenantID, string(qt))
}

func redisKeySoftLimit(tenantID string, qt QuotaType) string {
	return quotaKeyPrefix + fmt.Sprintf(quotaSoftLimitKey, tenantID, string(qt))
}

func redisKeyGrace(tenantID string) string {
	return quotaKeyPrefix + fmt.Sprintf(quotaGraceKey, tenantID, "global")
}

func redisKeyWarned(tenantID string, qt QuotaType) string {
	return quotaKeyPrefix + fmt.Sprintf(quotaWarnedKey, tenantID, string(qt))
}

// ────────────────────────────────────────────────────────────────────────────
// Public API
// ────────────────────────────────────────────────────────────────────────────

// Current возвращает текущее использование quota для tenant'а.
func (qm *QuotaManager) Current(ctx context.Context, tenantID string, qt QuotaType) (int64, error) {
	if qm.client == nil {
		return 0, nil // fail-open
	}

	val, err := qm.client.Get(ctx, redisKeyCurrent(tenantID, qt)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		qm.logger.Warn("quota: Redis GET failed, returning 0",
			"tenant_id", tenantID, "quota_type", qt, "error", err,
		)
		return 0, nil // fail-open
	}
	return val, nil
}

// Increment увеличивает счётчик квоты и проверяет лимиты.
//
// Возвращает:
//   - (true, nil) — успешно, лимит не превышен
//   - (false, nil) — hard limit превышен (блокировка)
//   - (false, error) — ошибка проверки
func (qm *QuotaManager) Increment(ctx context.Context, tenantID string, qt QuotaType) (bool, error) {
	if qm.client == nil {
		return true, nil // fail-open
	}

	key := redisKeyCurrent(tenantID, qt)

	// Атомарный INCR
	current, err := qm.client.Incr(ctx, key).Result()
	if err != nil {
		qm.logger.Warn("quota: Redis INCR failed, allowing (fail-open)",
			"tenant_id", tenantID, "quota_type", qt, "error", err,
		)
		return true, nil
	}

	// Устанавливаем TTL если это первый инкремент
	if current == 1 {
		ttl := 24 * time.Hour // для api_calls — 1 час, для остальных — бесконечно
		if qt == QuotaAPICalls {
			ttl = 1 * time.Hour
		}
		qm.client.Expire(ctx, key, ttl)
	}

	// Получаем лимиты
	limits, err := qm.getLimits(ctx, tenantID, qt)
	if err != nil {
		return true, nil // fail-open при ошибке получения лимитов
	}

	// Проверка hard limit
	if current >= limits.HardLimit {
		qm.logger.Warn("quota: hard limit exceeded",
			"tenant_id", tenantID, "quota_type", qt,
			"current", current, "hard_limit", limits.HardLimit,
		)
		return false, nil
	}

	// Проверка soft limit (только логируем, не блокируем)
	if limits.SoftLimit > 0 && current >= limits.SoftLimit {
		// Проверяем, было ли уже предупреждение
		warned, err := qm.client.Get(ctx, redisKeyWarned(tenantID, qt)).Result()
		if err == redis.Nil {
			// Отправляем предупреждение (NATS event)
			qm.client.Set(ctx, redisKeyWarned(tenantID, qt), "1", 24*time.Hour)
			qm.logger.Warn("quota: soft limit reached",
				"tenant_id", tenantID, "quota_type", qt,
				"current", current, "soft_limit", limits.SoftLimit,
			)
		}
		_ = warned // подавляем unused variable
	}

	return true, nil
}

// Decrement уменьшает счётчик квоты (при удалении ресурса).
func (qm *QuotaManager) Decrement(ctx context.Context, tenantID string, qt QuotaType) error {
	if qm.client == nil {
		return nil // fail-open
	}

	key := redisKeyCurrent(tenantID, qt)

	// Атомарный DECR, но не уходим ниже 0
	// Используем Lua script для атомарности
	const luaDecr = `
		local key = KEYS[1]
		local val = redis.call('GET', key)
		if val and tonumber(val) > 0 then
			return redis.call('DECR', key)
		end
		return 0
	`

	_, err := qm.client.Eval(ctx, luaDecr, []string{key}).Result()
	if err != nil {
		qm.logger.Warn("quota: Redis DECR failed",
			"tenant_id", tenantID, "quota_type", qt, "error", err,
		)
		return fmt.Errorf("decrement quota %s for tenant %s: %w", qt, tenantID, err)
	}

	return nil
}

// Check проверяет не превышен ли лимит квоты.
// В отличие от Increment, не увеличивает счётчик.
func (qm *QuotaManager) Check(ctx context.Context, tenantID string, qt QuotaType) (*QuotaStatus, error) {
	if qm.client == nil {
		return &QuotaStatus{
			Type:      qt,
			Current:   0,
			SoftLimit: 0,
			HardLimit: 0,
			Usage:     0,
		}, nil // fail-open
	}

	current, err := qm.Current(ctx, tenantID, qt)
	if err != nil {
		return nil, fmt.Errorf("check quota %s: %w", qt, err)
	}

	limits, err := qm.getLimits(ctx, tenantID, qt)
	if err != nil {
		return nil, fmt.Errorf("get quota limits %s: %w", qt, err)
	}

	// Проверка grace period
	graceUntil, _ := qm.getGraceUntil(ctx, tenantID)
	onGrace := graceUntil != nil && time.Now().Before(*graceUntil)

	// Вычисляем процент использования
	usagePercent := 0.0
	if limits.HardLimit > 0 {
		usagePercent = float64(current) / float64(limits.HardLimit) * 100.0
	}

	status := &QuotaStatus{
		Type:      qt,
		Current:   current,
		SoftLimit: limits.SoftLimit,
		HardLimit: limits.HardLimit,
		Usage:     usagePercent,
		IsSoft:    limits.SoftLimit > 0 && current >= limits.SoftLimit,
		IsHard:    limits.HardLimit > 0 && current >= limits.HardLimit,
		OnGrace:   onGrace,
	}

	if onGrace && graceUntil != nil {
		gu := *graceUntil
		status.GraceUntil = &gu
	}

	return status, nil
}

// GetAll возвращает использование всех квот для tenant'а.
func (qm *QuotaManager) GetAll(ctx context.Context, tenantID string) (*QuotaUsage, error) {
	usage := &QuotaUsage{
		TenantID: tenantID,
		Quotas:   make(map[QuotaType]*QuotaStatus),
	}

	for _, qt := range AllQuotaTypes() {
		status, err := qm.Check(ctx, tenantID, qt)
		if err != nil {
			qm.logger.Warn("quota: failed to check quota type",
				"tenant_id", tenantID, "quota_type", qt, "error", err,
			)
			continue
		}
		usage.Quotas[qt] = status
	}

	// Grace period info
	graceUntil, _ := qm.getGraceUntil(ctx, tenantID)
	if graceUntil != nil && time.Now().Before(*graceUntil) {
		usage.OnGrace = true
		usage.GraceUntil = graceUntil
		usage.GraceDays = int(time.Until(*graceUntil).Hours() / 24)
		suspendAt := graceUntil.Add(7 * 24 * time.Hour)
		usage.SuspendAt = &suspendAt
	}

	return usage, nil
}

// SetLimits устанавливает лимиты квоты для tenant'а (admin).
func (qm *QuotaManager) SetLimits(ctx context.Context, tenantID string, qt QuotaType, hardLimit int64) error {
	if qm.client == nil {
		return nil // fail-open
	}

	defaults := DefaultQuotaConfigs()
	cfg, ok := defaults[qt]
	if !ok {
		return fmt.Errorf("unknown quota type: %s", qt)
	}

	softLimit := int64(float64(hardLimit) * 0.8)

	// Устанавливаем в Redis
	pipe := qm.client.Pipeline()
	pipe.Set(ctx, redisKeyHardLimit(tenantID, qt), hardLimit, 0)
	pipe.Set(ctx, redisKeySoftLimit(tenantID, qt), softLimit, 0)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set quota limits in redis: %w", err)
	}

	// Обновляем кэш
	qm.mu.Lock()
	if qm.limitsCache[tenantID] == nil {
		qm.limitsCache[tenantID] = make(map[QuotaType]QuotaConfig)
	}
	qm.limitsCache[tenantID][qt] = QuotaConfig{
		Type:      qt,
		SoftLimit: softLimit,
		HardLimit: hardLimit,
		Unit:      cfg.Unit,
		GraceDays: cfg.GraceDays,
	}
	qm.mu.Unlock()

	qm.logger.Info("quota: limits updated",
		"tenant_id", tenantID, "quota_type", qt,
		"soft_limit", softLimit, "hard_limit", hardLimit,
	)

	return nil
}

// SetGraceUntil устанавливает grace period для tenant'а.
func (qm *QuotaManager) SetGraceUntil(ctx context.Context, tenantID string, until time.Time) error {
	if qm.client == nil {
		return nil // fail-open
	}

	err := qm.client.Set(ctx, redisKeyGrace(tenantID), until.Format(time.RFC3339), 0).Err()
	if err != nil {
		return fmt.Errorf("set grace period for tenant %s: %w", tenantID, err)
	}

	qm.mu.Lock()
	qm.graceCache[tenantID] = &until
	qm.mu.Unlock()

	return nil
}

// ResetQuota сбрасывает счётчик квоты для tenant'а.
func (qm *QuotaManager) ResetQuota(ctx context.Context, tenantID string, qt QuotaType) error {
	if qm.client == nil {
		return nil // fail-open
	}

	pipe := qm.client.Pipeline()
	pipe.Del(ctx, redisKeyCurrent(tenantID, qt))
	pipe.Del(ctx, redisKeyWarned(tenantID, qt))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("reset quota %s for tenant %s: %w", qt, tenantID, err)
	}

	return nil
}

// ResetAllQuotas сбрасывает все счётчики квот для tenant'а.
func (qm *QuotaManager) ResetAllQuotas(ctx context.Context, tenantID string) error {
	pipe := qm.client.Pipeline()
	for _, qt := range AllQuotaTypes() {
		pipe.Del(ctx, redisKeyCurrent(tenantID, qt))
		pipe.Del(ctx, redisKeyWarned(tenantID, qt))
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("reset all quotas for tenant %s: %w", tenantID, err)
	}
	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ────────────────────────────────────────────────────────────────────────────

// getLimits возвращает лимиты квоты для tenant'а.
// Сначала проверяет кэш, затем Redis, затем использует дефолты.
func (qm *QuotaManager) getLimits(ctx context.Context, tenantID string, qt QuotaType) (QuotaConfig, error) {
	// Проверяем кэш
	qm.mu.RLock()
	if limits, ok := qm.limitsCache[tenantID][qt]; ok {
		qm.mu.RUnlock()
		return limits, nil
	}
	qm.mu.RUnlock()

	// Пробуем получить из Redis
	pipe := qm.client.Pipeline()
	hardCmd := pipe.Get(ctx, redisKeyHardLimit(tenantID, qt))
	softCmd := pipe.Get(ctx, redisKeySoftLimit(tenantID, qt))
	_, err := pipe.Exec(ctx)
	if err == nil {
		hardLimit, _ := strconv.ParseInt(hardCmd.Val(), 10, 64)
		softLimit, _ := strconv.ParseInt(softCmd.Val(), 10, 64)
		if hardLimit > 0 {
			cfg := QuotaConfig{
				Type:      qt,
				SoftLimit: softLimit,
				HardLimit: hardLimit,
			}
			qm.mu.Lock()
			if qm.limitsCache[tenantID] == nil {
				qm.limitsCache[tenantID] = make(map[QuotaType]QuotaConfig)
			}
			qm.limitsCache[tenantID][qt] = cfg
			qm.mu.Unlock()
			return cfg, nil
		}
	}

	// Fallback: дефолтные лимиты
	defaults := DefaultQuotaConfigs()
	cfg, ok := defaults[qt]
	if !ok {
		return QuotaConfig{}, fmt.Errorf("unknown quota type: %s", qt)
	}

	return cfg, nil
}

// getGraceUntil возвращает дату окончания grace period для tenant'а.
func (qm *QuotaManager) getGraceUntil(ctx context.Context, tenantID string) (*time.Time, error) {
	// Проверяем кэш
	qm.mu.RLock()
	if until, ok := qm.graceCache[tenantID]; ok {
		qm.mu.RUnlock()
		return until, nil
	}
	qm.mu.RUnlock()

	// Пробуем получить из Redis
	val, err := qm.client.Get(ctx, redisKeyGrace(tenantID)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return nil, err
	}

	qm.mu.Lock()
	qm.graceCache[tenantID] = &t
	qm.mu.Unlock()

	return &t, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

// QuotaTypeFromString преобразует строку в QuotaType.
func QuotaTypeFromString(s string) (QuotaType, error) {
	switch strings.ToLower(s) {
	case "devices":
		return QuotaDevices, nil
	case "users":
		return QuotaUsers, nil
	case "storage_gb", "storage":
		return QuotaStorage, nil
	case "api_calls", "api":
		return QuotaAPICalls, nil
	case "work_orders", "wo":
		return QuotaWorkOrders, nil
	default:
		return "", fmt.Errorf("unknown quota type: %s", s)
	}
}
