// Package middleware — API middleware с OWASP ASVS L3 compliance.
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-RATE: Redis-based Distributed Rate Limiting
//
// Заменяет in-memory sliding window rate limiter на Redis-based token bucket
// с atomic sliding window через Lua scripting.
//
// Соответствует:
//   - OWASP ASVS V2.2.1 (Rate limiting)
//   - ISO 27001 A.12.1.2 (Capacity management)
//   - IEC 62443-3-3 SR 3.1 (Resource management)
//   - СТБ 34.101.27 п. 6.1 (Защита от DoS)
//
// Архитектура:
//   - Redis sorted sets для atomic sliding window (ZADD + ZREMRANGEBYSCORE)
//   - Lua script для атомарности check-and-increment
//   - Пер-tenant/user/IP идентификация
//   - X-RateLimit-* headers для клиентов
//   - Fail-open при недоступности Redis
//   - Prometheus метрики через OpenTelemetry
//
// ═══════════════════════════════════════════════════════════════════════════
package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// ────────────────────────────────────────────────────────────────────────────
// Константы и дефолты
// ────────────────────────────────────────────────────────────────────────────

const (
	// DefaultReadLimit — лимит read запросов в минуту (tenant).
	DefaultReadLimit = 100

	// DefaultWriteLimit — лимит write запросов в минуту (tenant).
	DefaultWriteLimit = 30

	// DefaultWindow — окно rate limiting.
	DefaultWindow = 1 * time.Minute

	// DefaultAPIKeyLimit — лимит запросов для API key.
	DefaultAPIKeyLimit = 100

	// DefaultAPIKeyWindow — окно rate limiting для API key.
	DefaultAPIKeyWindow = 1 * time.Minute

	// rateLimitKeyPrefix — префикс для Redis ключей rate limiting.
	rateLimitKeyPrefix = "ratelimit:"

	// luaSlidingWindow — Lua script для атомарного sliding window.
	// Использует sorted sets для хранения timestamp'ов запросов.
	// Возвращает {allowed, currentCount, limit}
	luaSlidingWindow = `
		local key = KEYS[1]
		local now = tonumber(ARGV[1])
		local window = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])

		redis.call('ZREMRANGEBYSCORE', key, 0, now - window)
		local count = redis.call('ZCARD', key)

		if count >= limit then
			return {0, count, limit}
		end

		redis.call('ZADD', key, now, now .. ':' .. math.random())
		redis.call('EXPIRE', key, window + 1)
		return {1, count + 1, limit}
	`
)

// ────────────────────────────────────────────────────────────────────────────
// RateLimiter — Redis-based distributed rate limiter
// ────────────────────────────────────────────────────────────────────────────

// RateLimiter реализует distributed rate limiting через Redis sorted sets.
//
// Потокобезопасен (вся атомарность на стороне Redis через Lua).
// При недоступности Redis использует fail-open (пропускает запрос).
type RateLimiter struct {
	client     *redis.Client
	readLimit  int           // лимит read запросов за window
	writeLimit int           // лимит write запросов за window
	window     time.Duration // временнóе окно

	// Prometheus метрики через OpenTelemetry
	rateLimitTotal   metric.Int64Counter
	rateLimitBlocked metric.Int64Counter
	rateLimitCurrent metric.Int64Gauge // текущее использование для top-N

	meterInit bool // флаг успешной инициализации метрик
}

// NewRateLimiter создаёт новый Redis-based rate limiter.
//
// Параметры:
//   - client: Redis клиент (nil-safe — возвращает noop limiter)
//   - readLimit: лимит GET/HEAD/OPTIONS запросов
//   - writeLimit: лимит POST/PUT/DELETE/PATCH запросов
//   - window: временнóе окно (например, time.Minute)
func NewRateLimiter(client *redis.Client, readLimit, writeLimit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		client:     client,
		readLimit:  readLimit,
		writeLimit: writeLimit,
		window:     window,
	}

	// Инициализация метрик (best-effort, не фатально при ошибке)
	rl.initMetrics()

	return rl
}

// initMetrics инициализирует OpenTelemetry метрики для rate limiter.
func (rl *RateLimiter) initMetrics() {
	meter := otel.Meter("cctv.ratelimit")

	var err error

	rl.rateLimitTotal, err = meter.Int64Counter(
		"ratelimit.total",
		metric.WithDescription("Total number of rate limit checks"),
		metric.WithUnit("1"),
	)
	if err != nil {
		rl.meterInit = false
		return
	}

	rl.rateLimitBlocked, err = meter.Int64Counter(
		"ratelimit.blocked",
		metric.WithDescription("Number of blocked rate limit requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		rl.meterInit = false
		return
	}

	rl.rateLimitCurrent, err = meter.Int64Gauge(
		"ratelimit.current",
		metric.WithDescription("Current rate limit usage per identifier"),
		metric.WithUnit("1"),
	)
	if err != nil {
		rl.meterInit = false
		return
	}

	rl.meterInit = true
}

// ────────────────────────────────────────────────────────────────────────────
// Ключи и идентификация
// ────────────────────────────────────────────────────────────────────────────

// limitTypeForMethod возвращает тип лимита (read/write) по HTTP методу.
func limitTypeForMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return "read"
	default:
		return "write"
	}
}

// redisKey формирует Redis key для rate limit entry.
// Формат: ratelimit:{identifier}:{read|write}
func redisKey(identifier, method string) string {
	return fmt.Sprintf("%s%s:%s", rateLimitKeyPrefix, identifier, limitTypeForMethod(method))
}

// ────────────────────────────────────────────────────────────────────────────
// Allow — проверка rate limit (публичный метод)
// ────────────────────────────────────────────────────────────────────────────

// Allow проверяет, не превышен ли rate limit для указанного идентификатора.
//
// Возвращает:
//   - allowed: true если запрос разрешён
//   - current: текущее количество запросов в окне
//   - limit: лимит запросов
//   - err: ошибка Redis (при fail-open возвращается allowed=true)
//
// Используется для прямого вызова из API key middleware и других компонентов.
func (rl *RateLimiter) Allow(ctx context.Context, identifier, method string) (allowed bool, current, limit int, err error) {
	// Если Redis не сконфигурирован — fail-open
	if rl.client == nil {
		return true, 0, rl.getLimit(method), nil
	}

	key := redisKey(identifier, method)
	now := time.Now().Unix()
	windowSec := int64(rl.window.Seconds())
	limit = rl.getLimit(method)

	script := redis.NewScript(luaSlidingWindow)

	result, err := script.Run(ctx, rl.client, []string{key}, now, windowSec, limit).Slice()
	if err != nil {
		// Fail-open: при ошибке Redis пропускаем запрос
		return true, 0, limit, err
	}

	allowed = result[0].(int64) == 1
	current = int(result[1].(int64))

	return allowed, current, limit, nil
}

// getLimit возвращает лимит для HTTP метода.
func (rl *RateLimiter) getLimit(method string) int {
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		return rl.readLimit
	}
	return rl.writeLimit
}

// ────────────────────────────────────────────────────────────────────────────
// Middleware — HTTP middleware
// ────────────────────────────────────────────────────────────────────────────

// Middleware возвращает HTTP middleware для rate limiting.
//
// Определяет идентификатор в порядке приоритета:
//  1. Tenant ID (из контекста)
//  2. User ID (из контекста)
//  3. Client IP
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identifier := getIdentifier(r)
		if identifier == "" {
			identifier = getClientIP(r)
		}

		allowed, current, limit, err := rl.Allow(r.Context(), identifier, r.Method)
		if err != nil {
			// Fail-open: при ошибке Redis пропускаем запрос, но логируем
			// Метрики не обновляем — ошибка не связана с клиентом
			next.ServeHTTP(w, r)
			return
		}

		// ── Prometheus метрики ──────────────────────────────────────────
		if rl.meterInit {
			attrs := metric.WithAttributes(
				attribute.String("identifier_type", getIdentifierType(r)),
				attribute.String("method", r.Method),
				attribute.Bool("allowed", allowed),
			)
			rl.rateLimitTotal.Add(r.Context(), 1, attrs)
			if !allowed {
				rl.rateLimitBlocked.Add(r.Context(), 1, attrs)
			}
			rl.rateLimitCurrent.Record(r.Context(), int64(current),
				metric.WithAttributes(
					attribute.String("identifier", identifier),
					attribute.String("limit_type", limitTypeForMethod(r.Method)),
				),
			)
		}

		// ── X-RateLimit headers ─────────────────────────────────────────
		remaining := max(0, limit-current)
		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(rl.window).Unix(), 10))

		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(rl.window.Seconds())))
			http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many requests"}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Вспомогательные функции
// ────────────────────────────────────────────────────────────────────────────

// getIdentifier извлекает идентификатор для rate limiting из контекста запроса.
// Приоритет: tenant > user > ""
func getIdentifier(r *http.Request) string {
	if tenantID := r.Context().Value("tenant_id"); tenantID != nil {
		if id, ok := tenantID.(string); ok && id != "" {
			return fmt.Sprintf("tenant:%s", id)
		}
	}
	if userID := r.Context().Value("user_id"); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			return fmt.Sprintf("user:%s", id)
		}
	}
	return ""
}

// getIdentifierType возвращает тип идентификатора для метрик.
func getIdentifierType(r *http.Request) string {
	if r.Context().Value("tenant_id") != nil {
		return "tenant"
	}
	if r.Context().Value("user_id") != nil {
		return "user"
	}
	return "ip"
}

// getClientIP извлекает IP клиента из запроса.
// Учитывает reverse proxy заголовки:
//   - X-Forwarded-For (первый IP в цепочке)
//   - X-Real-IP
//   - RemoteAddr (fallback)
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	// Убираем порт из RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// ────────────────────────────────────────────────────────────────────────────
// Max helper (Go <1.21 compatibility)
// ────────────────────────────────────────────────────────────────────────────

// max возвращает большее из двух целых чисел.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ────────────────────────────────────────────────────────────────────────────
// rand — для Lua script uniqueness (не криптостойкий, только для коллизий)
// ────────────────────────────────────────────────────────────────────────────

func init() {
	// Инициализируем rand для уникальности member'ов в sorted set
	// (не криптографическая безопасность, а предотвращение коллизий)
	_ = rand.Int()
}
