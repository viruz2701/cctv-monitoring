// Package state — Redis-based Device State Store (P1-PERF.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-PERF.2: Redis для SLA Trackers и Device State
//
// Заменяет in-memory map на Redis:
//   - Distributed locking для конкурентного доступа
//   - TTL для автоматической очистки stale state
//   - Атомарные операции (INCR, EXPIRE)
//   - Pub/Sub для real-time уведомлений об изменениях
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — distributed state)
//   - ISO 27001 A.12.4.1 (Event logging — state changes)
//   - ISO 27019 PCC.A.12.4 (ICS audit trail)
//
// ═══════════════════════════════════════════════════════════════════════════
package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// DefaultDeviceStateTTL — TTL для состояния устройства (30 мин без heartbeat).
	DefaultDeviceStateTTL = 30 * time.Minute

	// DefaultLockTTL — TTL для distributed lock (10 сек).
	DefaultLockTTL = 10 * time.Second

	// StateChannel — Redis Pub/Sub канал для изменений состояния.
	StateChannel = "device:state:changes"
)

// ────────────────────────────────────────────────────────────────────────────
// RedisDeviceStore
// ────────────────────────────────────────────────────────────────────────────

// RedisDeviceStore реализует DeviceStateManager через Redis.
//
// Ключи:
//   - device:state:{deviceID} — Hash с полями состояния (JSON)
//   - device:lock:{deviceID} — String для distributed lock (NX + EXPIRE)
//   - device:online — Set всех online deviceID
//   - device:offline — Set всех offline deviceID
type RedisDeviceStore struct {
	client  *redis.Client
	ttl     time.Duration
	lockTTL time.Duration
	logger  *slog.Logger
	pubsub  *redis.PubSub
	subs    map[string][]chan DeviceStateChange
	subsMu  sync.RWMutex
	closeCh chan struct{}
}

// DeviceStateChange — событие изменения состояния устройства.
type DeviceStateChange struct {
	DeviceID  string       `json:"device_id"`
	OldStatus DeviceStatus `json:"old_status,omitempty"`
	NewStatus DeviceStatus `json:"new_status"`
	Timestamp time.Time    `json:"timestamp"`
}

// DeviceStatus — статус устройства.
type DeviceStatus string

const (
	DeviceOnline  DeviceStatus = "online"
	DeviceOffline DeviceStatus = "offline"
	DeviceWarning DeviceStatus = "warning"
	DeviceUnknown DeviceStatus = "unknown"
)

// RedisStoreOption — функциональная опция для RedisDeviceStore.
type RedisStoreOption func(*RedisDeviceStore)

// WithTTL устанавливает TTL для состояния устройства.
func WithTTL(ttl time.Duration) RedisStoreOption {
	return func(s *RedisDeviceStore) {
		if ttl > 0 {
			s.ttl = ttl
		}
	}
}

// WithLockTTL устанавливает TTL для distributed lock.
func WithLockTTL(ttl time.Duration) RedisStoreOption {
	return func(s *RedisDeviceStore) {
		if ttl > 0 {
			s.lockTTL = ttl
		}
	}
}

// WithLogger устанавливает логгер.
func WithLogger(logger *slog.Logger) RedisStoreOption {
	return func(s *RedisDeviceStore) {
		s.logger = logger
	}
}

// NewRedisDeviceStore создаёт новый RedisDeviceStore.
func NewRedisDeviceStore(client *redis.Client, opts ...RedisStoreOption) *RedisDeviceStore {
	s := &RedisDeviceStore{
		client:  client,
		ttl:     DefaultDeviceStateTTL,
		lockTTL: DefaultLockTTL,
		logger:  slog.Default().With("component", "state.redis_store"),
		subs:    make(map[string][]chan DeviceStateChange),
		closeCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Запускаем Pub/Sub listener
	s.pubsub = s.client.Subscribe(context.Background(), StateChannel)
	go s.listenPubSub()

	return s
}

// ────────────────────────────────────────────────────────────────────────────
// DeviceStateManager interface implementation
// ────────────────────────────────────────────────────────────────────────────

// SetDeviceState сохраняет состояние устройства в Redis.
func (s *RedisDeviceStore) SetDeviceState(ctx context.Context, deviceID string, state interface{}) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal device %s state: %w", deviceID, err)
	}

	key := fmt.Sprintf("device:state:%s", deviceID)
	if err := s.client.Set(ctx, key, data, s.ttl).Err(); err != nil {
		return fmt.Errorf("set device %s state: %w", deviceID, err)
	}

	// Добавляем в online set
	s.client.SAdd(ctx, "device:online", deviceID)

	return nil
}

// GetDeviceState возвращает состояние устройства из Redis.
func (s *RedisDeviceStore) GetDeviceState(ctx context.Context, deviceID string, target interface{}) error {
	key := fmt.Sprintf("device:state:%s", deviceID)
	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("device %s state not found", deviceID)
		}
		return fmt.Errorf("get device %s state: %w", deviceID, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("unmarshal device %s state: %w", deviceID, err)
	}

	return nil
}

// DeleteDeviceState удаляет состояние устройства.
func (s *RedisDeviceStore) DeleteDeviceState(ctx context.Context, deviceID string) error {
	key := fmt.Sprintf("device:state:%s", deviceID)
	pipe := s.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, "device:online", deviceID)
	pipe.SAdd(ctx, "device:offline", deviceID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete device %s state: %w", deviceID, err)
	}

	// Публикуем изменение
	s.publishChange(ctx, DeviceStateChange{
		DeviceID:  deviceID,
		NewStatus: DeviceOffline,
		Timestamp: time.Now().UTC(),
	})

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Distributed locking
// ────────────────────────────────────────────────────────────────────────────

// AcquireLock пытается получить distributed lock для устройства.
// Возвращает true если блокировка получена.
func (s *RedisDeviceStore) AcquireLock(ctx context.Context, deviceID string) (bool, error) {
	key := fmt.Sprintf("device:lock:%s", deviceID)
	ok, err := s.client.SetNX(ctx, key, "1", s.lockTTL).Result()
	if err != nil {
		return false, fmt.Errorf("acquire lock for %s: %w", deviceID, err)
	}
	return ok, nil
}

// ReleaseLock освобождает distributed lock.
func (s *RedisDeviceStore) ReleaseLock(ctx context.Context, deviceID string) error {
	key := fmt.Sprintf("device:lock:%s", deviceID)
	return s.client.Del(ctx, key).Err()
}

// ────────────────────────────────────────────────────────────────────────────
// Online/Offline tracking
// ────────────────────────────────────────────────────────────────────────────

// GetOnlineDevices возвращает список online устройств.
func (s *RedisDeviceStore) GetOnlineDevices(ctx context.Context) ([]string, error) {
	return s.client.SMembers(ctx, "device:online").Result()
}

// GetOfflineDevices возвращает список offline устройств.
func (s *RedisDeviceStore) GetOfflineDevices(ctx context.Context) ([]string, error) {
	return s.client.SMembers(ctx, "device:offline").Result()
}

// MarkOffline помечает устройство как offline.
func (s *RedisDeviceStore) MarkOffline(ctx context.Context, deviceID string) error {
	pipe := s.client.Pipeline()
	pipe.SRem(ctx, "device:online", deviceID)
	pipe.SAdd(ctx, "device:offline", deviceID)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark device %s offline: %w", deviceID, err)
	}

	s.publishChange(ctx, DeviceStateChange{
		DeviceID:  deviceID,
		NewStatus: DeviceOffline,
		Timestamp: time.Now().UTC(),
	})

	return nil
}

// ────────────────────────────────────────────────────────────────────────────
// Atomic operations
// ────────────────────────────────────────────────────────────────────────────

// IncrementCounter атомарно увеличивает счётчик для устройства.
func (s *RedisDeviceStore) IncrementCounter(ctx context.Context, deviceID, counter string) (int64, error) {
	key := fmt.Sprintf("device:counter:%s:%s", deviceID, counter)
	val, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("increment counter %s for %s: %w", counter, deviceID, err)
	}

	// Обновляем TTL
	s.client.Expire(ctx, key, s.ttl)

	return val, nil
}

// ────────────────────────────────────────────────────────────────────────────
// Pub/Sub for real-time updates
// ────────────────────────────────────────────────────────────────────────────

// Subscribe подписывается на изменения состояния устройства.
// Возвращает канал для получения событий.
func (s *RedisDeviceStore) Subscribe(deviceID string) <-chan DeviceStateChange {
	ch := make(chan DeviceStateChange, 100)
	s.subsMu.Lock()
	s.subs[deviceID] = append(s.subs[deviceID], ch)
	s.subsMu.Unlock()
	return ch
}

// Unsubscribe отписывается от изменений.
func (s *RedisDeviceStore) Unsubscribe(deviceID string, ch <-chan DeviceStateChange) {
	s.subsMu.Lock()
	defer s.subsMu.Unlock()
	subs := s.subs[deviceID]
	for i, sub := range subs {
		if sub == ch {
			s.subs[deviceID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}

// publishChange публикует изменение в Redis Pub/Sub и локальным подписчикам.
func (s *RedisDeviceStore) publishChange(ctx context.Context, change DeviceStateChange) {
	data, err := json.Marshal(change)
	if err != nil {
		s.logger.Error("marshal state change", "error", err)
		return
	}

	// Redis Pub/Sub
	if err := s.client.Publish(ctx, StateChannel, data).Err(); err != nil {
		s.logger.Error("publish state change", "error", err)
	}

	// Локальные подписчики
	s.subsMu.RLock()
	subs := s.subs[change.DeviceID]
	s.subsMu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- change:
		default:
			// Канал переполнен — пропускаем
		}
	}
}

// listenPubSub слушает Redis Pub/Sub сообщения.
func (s *RedisDeviceStore) listenPubSub() {
	ch := s.pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			var change DeviceStateChange
			if err := json.Unmarshal([]byte(msg.Payload), &change); err != nil {
				s.logger.Error("unmarshal pubsub message", "error", err)
				continue
			}

			// Рассылаем локальным подписчикам
			s.subsMu.RLock()
			subs := s.subs[change.DeviceID]
			s.subsMu.RUnlock()

			for _, sub := range subs {
				select {
				case sub <- change:
				default:
				}
			}

		case <-s.closeCh:
			return
		}
	}
}

// Close закрывает RedisDeviceStore.
func (s *RedisDeviceStore) Close() error {
	close(s.closeCh)
	if s.pubsub != nil {
		return s.pubsub.Close()
	}
	return nil
}
