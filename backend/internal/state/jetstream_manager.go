// Package state — distributed device state management via NATS JetStream KV Store.
//
// ARCH-01: Решает проблему горизонтального масштабирования InMemoryStateManager.
// Вместо sync.Map in-memory (не шардится между подами K8s) используем
// NATS JetStream KV Store как source of truth.
//
// ═══════════════════════════════════════════════════════════════════════════
// Архитектура:
//
//  1. NATS JetStream KV Bucket "device_state" — source of truth
//  2. Локальный in-memory cache (sync.Map) для read-heavy операций
//  3. Event-driven инвалидация кэша через NATS Watch
//  4. Graceful degradation: при потере NATS — работаем из кэша
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — distributed state)
//   - ISO 27001 A.12.6.1 (Capacity management — horizontal scaling)
//   - ISO 27001 A.17.1.1 (Redundancy — multi-replica support)
//   - СТБ 34.101.27 п. 8.2 (Availability — отказоустойчивость)
//   - OWASP ASVS V1.8 (Architecture — stateless design)
// ═══════════════════════════════════════════════════════════════════════════
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const (
	// KVDeviceBucket — имя KV bucket для хранения состояния устройств.
	KVDeviceBucket = "device_state"

	// KVDeviceBucketTTL — время жизни записей в KV (120 суток = max heartbeat timeout).
	KVDeviceBucketTTL = 120 * 24 * time.Hour

	// kvWatchReconnectDelay — задержка перед переподключением watcher'а.
	kvWatchReconnectDelay = 5 * time.Second
)

// ═══════════════════════════════════════════════════════════════════════
// Errors
// ═══════════════════════════════════════════════════════════════════════

var (
	// ErrJetStreamNotAvailable возвращается если JetStream контекст не инициализирован.
	ErrJetStreamNotAvailable = errors.New("NATS JetStream context not available")
)

// ═══════════════════════════════════════════════════════════════════════
// JetStreamStateManager — распределённый менеджер состояния устройств.
// ═══════════════════════════════════════════════════════════════════════

// JetStreamStateManager implements DeviceStateManager using NATS JetStream KV Store.
//
// Особенности:
//   - Запись: atomically в KV bucket → инвалидация кэша
//   - Чтение: из in-memory cache (sync.Map) для производительности
//   - Watch: NATS watcher для синхронизации между подами
//   - Graceful degradation: при потере NATS — последнее известное состояние из кэша
//
// Потокобезопасность: все публичные методы thread-safe.
// sync.Map обеспечивает конкурентный доступ без глобальной блокировки.
type JetStreamStateManager struct {
	kv     nats.KeyValue
	logger *slog.Logger

	// localCache — read-through cache для быстрого чтения.
	// Инвалидируется через NATS Watch (Update/Delete события).
	localCache sync.Map // map[string]*models.Device

	// watcherCtx управляет жизнью goroutine watcher'а.
	watcherStop chan struct{}
	watcherWg   sync.WaitGroup

	// Метрики
	ops struct {
		sync.RWMutex
		totalGets    int64
		totalSets    int64
		totalDeletes int64
		cacheHits    int64
		cacheMisses  int64
		kvErrors     int64
	}
}

// NewJetStreamStateManager создаёт JetStreamStateManager.
//
// Параметры:
//   - js: NATS JetStream контекст (не может быть nil)
//   - logger: логгер (если nil — используется slog.Default())
//
// Алгоритм:
//  1. Создаёт/открывает KV bucket "device_state"
//  2. Загружает существующие записи в localCache
//  3. Запускает watcher для синхронизации между подами
//
// Graceful degradation: если KV bucket не создаётся,
// возвращаем ошибку (не паникуем).
func NewJetStreamStateManager(js nats.JetStreamContext, logger *slog.Logger) (*JetStreamStateManager, error) {
	if js == nil {
		return nil, ErrJetStreamNotAvailable
	}
	if logger == nil {
		logger = slog.Default()
	}

	// nats.KeyValueConfig — без MaxHistory (не поддерживается в nats.go v1.52)
	kv, err := js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:      KVDeviceBucket,
		Description: "Device state for CCTV Health Monitor (ARCH-01)",
		TTL:         KVDeviceBucketTTL,
		Storage:     nats.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		return nil, fmt.Errorf("create KV bucket %s: %w", KVDeviceBucket, err)
	}

	m := &JetStreamStateManager{
		kv:          kv,
		logger:      logger.With("component", "jetstream-state"),
		watcherStop: make(chan struct{}),
	}

	// 1. Загружаем существующие записи в localCache
	if err := m.loadExistingKeys(); err != nil {
		m.logger.Warn("failed to load existing keys from KV", "error", err)
		// Non-fatal: продолжаем работу, кэш заполнится при первом чтении
	}

	// 2. Запускаем watcher для синхронизации между подами
	m.watcherWg.Add(1)
	go m.watchLoop()

	m.logger.Info("JetStream state manager initialized",
		"bucket", KVDeviceBucket,
		"ttl", KVDeviceBucketTTL,
	)

	return m, nil
}

// ═══════════════════════════════════════════════════════════════════════
// DeviceStateManager interface implementation
// ═══════════════════════════════════════════════════════════════════════

// Get возвращает устройство из локального кэша.
// Если в кэше нет — пытается прочитать из KV (cache miss recovery).
func (m *JetStreamStateManager) Get(deviceID string) (*models.Device, bool) {
	m.ops.Lock()
	m.ops.totalGets++
	m.ops.Unlock()

	// 1. Пытаемся прочитать из кэша
	if val, ok := m.localCache.Load(deviceID); ok {
		m.ops.Lock()
		m.ops.cacheHits++
		m.ops.Unlock()
		return val.(*models.Device), true
	}

	m.ops.Lock()
	m.ops.cacheMisses++
	m.ops.Unlock()

	// 2. Cache miss: читаем из KV (recovery)
	entry, err := m.kv.Get(deviceID)
	if err != nil {
		return nil, false
	}

	dev, err := decodeDevice(entry.Value())
	if err != nil {
		m.logger.Warn("failed to decode device from KV", "device_id", deviceID, "error", err)
		return nil, false
	}

	// Обновляем кэш
	m.localCache.Store(deviceID, dev)
	return dev, true
}

// Set сохраняет устройство в KV и обновляет локальный кэш.
// Атомарная операция: запись в KV → обновление кэша.
func (m *JetStreamStateManager) Set(device *models.Device) {
	m.ops.Lock()
	m.ops.totalSets++
	m.ops.Unlock()

	data, err := encodeDevice(device)
	if err != nil {
		m.logger.Error("failed to encode device", "device_id", device.DeviceID, "error", err)
		return
	}

	// Записываем в KV (атомарно)
	if _, err := m.kv.Put(device.DeviceID, data); err != nil {
		m.logger.Error("failed to put device in KV", "device_id", device.DeviceID, "error", err)
		m.ops.Lock()
		m.ops.kvErrors++
		m.ops.Unlock()
		return
	}

	// Обновляем локальный кэш
	m.localCache.Store(device.DeviceID, device)
}

// Delete удаляет устройство из KV и локального кэша.
func (m *JetStreamStateManager) Delete(deviceID string) {
	m.ops.Lock()
	m.ops.totalDeletes++
	m.ops.Unlock()

	// Удаляем из KV
	if err := m.kv.Delete(deviceID); err != nil {
		m.logger.Warn("failed to delete device from KV", "device_id", deviceID, "error", err)
		m.ops.Lock()
		m.ops.kvErrors++
		m.ops.Unlock()
	}

	// Удаляем из кэша (даже если KV delete не удался — данные могли уже исчезнуть)
	m.localCache.Delete(deviceID)
}

// GetAll возвращает все устройства из локального кэша.
// Не гарантирует полную актуальность (eventual consistency).
func (m *JetStreamStateManager) GetAll() map[string]*models.Device {
	result := make(map[string]*models.Device)
	m.localCache.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*models.Device)
		return true
	})
	return result
}

// UpdateLastSeen обновляет время последней активности устройства.
func (m *JetStreamStateManager) UpdateLastSeen(deviceID string) {
	if dev, ok := m.Get(deviceID); ok {
		dev.LastSeen = time.Now()
		m.Set(dev)
	}
}

// SetOnline устанавливает статус устройства в ONLINE.
func (m *JetStreamStateManager) SetOnline(deviceID string) {
	if dev, ok := m.Get(deviceID); ok {
		if dev.Status != models.StatusOnline {
			dev.Status = models.StatusOnline
			dev.LastSeen = time.Now()
			m.Set(dev)
		}
	}
}

// SetOffline устанавливает статус устройства в OFFLINE.
func (m *JetStreamStateManager) SetOffline(deviceID string) {
	if dev, ok := m.Get(deviceID); ok {
		if dev.Status != models.StatusOffline {
			dev.Status = models.StatusOffline
			dev.LastSeen = time.Now()
			m.Set(dev)
		}
	}
}

// AddAlarm добавляет тревогу к устройству.
func (m *JetStreamStateManager) AddAlarm(deviceID string, alarm *models.Alarm) {
	if dev, ok := m.Get(deviceID); ok {
		dev.LastAlarm = alarm
		m.Set(dev)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Watcher — синхронизация между подами
// ═══════════════════════════════════════════════════════════════════════

// watchLoop слушает изменения в KV bucket и синхронизирует локальный кэш.
//
// Обрабатывает события:
//   - KeyValuePut: обновляет кэш
//   - KeyValueDelete: удаляет из кэша
//   - KeyValuePurge: очищает кэш
//
// При ошибке watcher'а — переподключается с экспоненциальной задержкой.
func (m *JetStreamStateManager) watchLoop() {
	defer m.watcherWg.Done()

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		select {
		case <-m.watcherStop:
			m.logger.Debug("watcher stopped")
			return
		default:
		}

		watcher, err := m.kv.WatchAll()
		if err != nil {
			m.logger.Warn("failed to start KV watcher, retrying", "error", err)
			select {
			case <-m.watcherStop:
				return
			case <-time.After(backoff):
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		backoff = time.Second // reset on successful connect

		m.processWatcherEvents(watcher)
	}
}

// processWatcherEvents обрабатывает события от watcher'а.
func (m *JetStreamStateManager) processWatcherEvents(watcher nats.KeyWatcher) {
	defer watcher.Stop()

	for {
		select {
		case <-m.watcherStop:
			return
		case entry, ok := <-watcher.Updates():
			if !ok {
				// Канал закрыт — нужно пересоздать watcher
				m.logger.Warn("KV watcher channel closed, reconnecting")
				return
			}
			if entry == nil {
				// Начальная загрузка завершена
				continue
			}

			m.handleKVEntry(entry)
		}
	}
}

// handleKVEntry обрабатывает одно событие из KV bucket.
func (m *JetStreamStateManager) handleKVEntry(entry nats.KeyValueEntry) {
	switch entry.Operation() {
	case nats.KeyValuePut:
		dev, err := decodeDevice(entry.Value())
		if err != nil {
			m.logger.Warn("failed to decode device from KV watch",
				"device_id", entry.Key(), "error", err,
			)
			return
		}
		m.localCache.Store(entry.Key(), dev)

	case nats.KeyValueDelete, nats.KeyValuePurge:
		m.localCache.Delete(entry.Key())
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Initial load
// ═══════════════════════════════════════════════════════════════════════

// loadExistingKeys загружает существующие записи из KV в localCache при старте.
func (m *JetStreamStateManager) loadExistingKeys() error {
	keys, err := m.kv.Keys()
	if err != nil {
		return fmt.Errorf("list KV keys: %w", err)
	}

	for _, key := range keys {
		entry, err := m.kv.Get(key)
		if err != nil {
			m.logger.Warn("failed to get key from KV during load", "key", key, "error", err)
			continue
		}

		dev, err := decodeDevice(entry.Value())
		if err != nil {
			m.logger.Warn("failed to decode device during load", "key", key, "error", err)
			continue
		}

		m.localCache.Store(key, dev)
	}

	m.logger.Info("loaded existing devices from KV", "count", len(keys))
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Shutdown
// ═══════════════════════════════════════════════════════════════════════

// Stop останавливает watcher и освобождает ресурсы.
func (m *JetStreamStateManager) Stop() {
	close(m.watcherStop)
	m.watcherWg.Wait()
	m.logger.Info("JetStream state manager stopped")
}

// ═══════════════════════════════════════════════════════════════════════
// Metrics
// ═══════════════════════════════════════════════════════════════════════

// Metrics возвращает метрики производительности.
// Используется для health checks и мониторинга.
func (m *JetStreamStateManager) Metrics() map[string]int64 {
	m.ops.RLock()
	defer m.ops.RUnlock()

	return map[string]int64{
		"total_gets":    m.ops.totalGets,
		"total_sets":    m.ops.totalSets,
		"total_deletes": m.ops.totalDeletes,
		"cache_hits":    m.ops.cacheHits,
		"cache_misses":  m.ops.cacheMisses,
		"kv_errors":     m.ops.kvErrors,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Serialization helpers
// ═══════════════════════════════════════════════════════════════════════

// encodeDevice сериализует Device в JSON.
func encodeDevice(dev *models.Device) ([]byte, error) {
	// Используем json.Marshal (без compression — KV сам управляет хранением)
	return json.Marshal(dev)
}

// decodeDevice десериализует Device из JSON.
func decodeDevice(data []byte) (*models.Device, error) {
	var dev models.Device
	if err := json.Unmarshal(data, &dev); err != nil {
		return nil, fmt.Errorf("decode device: %w", err)
	}
	return &dev, nil
}
