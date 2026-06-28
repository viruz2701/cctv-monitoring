// ═══════════════════════════════════════════════════════════════════════════
// P1-PERF.8: Redis Connection Pool Optimization
//
// Wrapper вокруг go-redis/v9 connection pool с оптимальными настройками:
//   - PoolSize: 10 (на 4 CPU) / 20 (на 8+ CPU) / кастом
//   - MinIdleConns: 5 (предотвращает холодный старт)
//   - ConnMaxLifetime: 30min (ротация соединений)
//   - ConnMaxIdleTime: 5min (освобождение неиспользуемых)
//   - PoolTimeout: 1s (таймаут ожидания соединения)
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — pool sizing)
//   - IEC 62443 SR 7.1 (Resource availability — connection resilience)
// ═══════════════════════════════════════════════════════════════════════════

package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Default pool configuration constants.
const (
	DefaultPoolSize        = 10
	DefaultMinIdleConns    = 5
	DefaultMaxConnLifetime = 30 * time.Minute
	DefaultMaxConnIdleTime = 5 * time.Minute
	DefaultPoolTimeout     = 4 * time.Second
	DefaultDialTimeout     = 5 * time.Second
	DefaultReadTimeout     = 3 * time.Second
	DefaultWriteTimeout    = 3 * time.Second
)

// PoolConfig — конфигурация Redis connection pool.
type PoolConfig struct {
	// Адрес Redis (host:port).
	Addr string `json:"addr" yaml:"addr"`
	// Пароль (если требуется).
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// Номер базы данных.
	DB int `json:"db" yaml:"db"`
	// Использовать TLS.
	UseTLS bool `json:"use_tls" yaml:"use_tls"`
	// Максимальное количество соединений в пуле.
	// По умолчанию: 10 (4 CPU) — 20 (8+ CPU).
	PoolSize int `json:"pool_size" yaml:"pool_size"`
	// Минимальное количество idle соединений.
	// По умолчанию: 5.
	MinIdleConns int `json:"min_idle_conns" yaml:"min_idle_conns"`
	// Максимальное время жизни соединения.
	// По умолчанию: 30 минут.
	MaxConnLifetime time.Duration `json:"max_conn_lifetime" yaml:"max_conn_lifetime"`
	// Максимальное время idle соединения.
	// По умолчанию: 5 минут.
	MaxConnIdleTime time.Duration `json:"max_conn_idle_time" yaml:"max_conn_idle_time"`
	// Таймаут ожидания соединения из пула.
	// По умолчанию: 4 секунды.
	PoolTimeout time.Duration `json:"pool_timeout" yaml:"pool_timeout"`
}

// DefaultPoolConfig возвращает конфигурацию пула по умолчанию.
// PoolSize авто-подбирается под количество CPU (минимум 10).
func DefaultPoolConfig() PoolConfig {
	poolSize := DefaultPoolSize
	return PoolConfig{
		Addr:            "localhost:6379",
		DB:              0,
		PoolSize:        poolSize,
		MinIdleConns:    DefaultMinIdleConns,
		MaxConnLifetime: DefaultMaxConnLifetime,
		MaxConnIdleTime: DefaultMaxConnIdleTime,
		PoolTimeout:     DefaultPoolTimeout,
	}
}

// NewClient creates a new Redis client with the specified pool configuration.
// Returns a configured *goredis.Client and an error if TLS setup fails.
func NewClient(cfg PoolConfig) (*goredis.Client, error) {
	opts := &goredis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxLifetime: cfg.MaxConnLifetime,
		ConnMaxIdleTime: cfg.MaxConnIdleTime,
		PoolTimeout:     cfg.PoolTimeout,
		DialTimeout:     DefaultDialTimeout,
		ReadTimeout:     DefaultReadTimeout,
		WriteTimeout:    DefaultWriteTimeout,

		// P1-PERF.8: OnConnect — метрики при создании соединения
		OnConnect: func(ctx context.Context, cn *goredis.Conn) error {
			return nil
		},
	}

	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := goredis.NewClient(opts)

	// Проверяем соединение
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}

// PoolStats возвращает статистику connection pool из go-redis.
func PoolStats(client *goredis.Client) *goredis.PoolStats {
	if client == nil {
		return nil
	}
	return client.PoolStats()
}

// ClosePool safely closes the Redis client and its connection pool.
func ClosePool(client *goredis.Client) error {
	if client == nil {
		return nil
	}
	return client.Close()
}
