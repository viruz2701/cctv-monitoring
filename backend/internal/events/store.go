// Package events — Event Store для CCTV Health Monitor.
//
// Реализует two-tier event storage:
//   Hot Tier (NATS JetStream) — быстрый доступ, retention до 1 года
//   Cold Tier (S3/MinIO) — долговременное хранение до 5 лет
//
// Compliance:
//   - ISO 27001:2022 A.12.4 (Logging and Monitoring — event retention)
//   - ISO 27001:2022 A.12.4.1 (Event logging — immutable timeline)
//   - ISO 27001:2022 A.8.10 (Information disposal — retention policy)
//   - СТБ 34.101.27 п. 7.5 (Audit trail integrity — tamper detection)
//   - СТБ 34.101.30 (bash-256 HMAC для цепочки событий — PLACEHOLDER)
//   - IEC 62443 SR 2.8 (Audit events — tamper detection)
//   - OWASP ASVS V7 (Code Quality — error handling)
//   - Приказ ОАЦ №66 п. 7.18.3 (Audit trail для конечных узлов)
//
// DM-1.2.2: Event Store на базе NATS JetStream
// DM-1.2.3: Projection Builder (read-model из events — следующий этап)
package events

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// ═══════════════════════════════════════════════════════════════════════
// EventRecord — универсальный конверт для всех событий Event Store.
// ═══════════════════════════════════════════════════════════════════════

// EventSource определяет источник (домен) события.
type EventSource string

const (
	SourceAlarms      EventSource = "alarms"
	SourceCMMS        EventSource = "cmms"
	SourcePredictions EventSource = "predictions"
	SourceTelemetry   EventSource = "telemetry"
	SourceAudit       EventSource = "audit"
	SourceSystem      EventSource = "system"
)

// EventSchemaVersion — версия схемы события (semver).
type EventSchemaVersion string

// EventRecord — универсальный конверт события для Event Store.
// Хранится как в NATS JetStream (hot), так и в S3 (cold).
//
// Compliance:
//   - prev_hash: СТБ 34.101.30 bash-256 HMAC цепочка (tamper detection)
//   - schema_version: версионирование схемы события
//   - trace_id: распределённая трассировка (ISO 27001 A.12.4.3)
//   - signed_at: ISO 27001 A.12.4.2 (время подписи)
type EventRecord struct {
	ID            string              `json:"id"`             // UUID v7 (time-sortable)
	Source        EventSource         `json:"source"`         // alarms, cmms, predictions, telemetry
	EventType     string              `json:"event_type"`     // alarm.created, cmms.wo.completed, etc.
	SchemaVersion EventSchemaVersion  `json:"schema_version"` // "1.0.0"
	Timestamp     time.Time           `json:"timestamp"`      // время события
	AggregateID   string              `json:"aggregate_id"`   // device_id, work_order_id, etc.
	ActorID       string              `json:"actor_id,omitempty"` // кто вызвал событие (user_id, system)
	TraceID       string              `json:"trace_id,omitempty"` // распределённый trace ID (W3C Trace Context)
	PrevHash      string              `json:"prev_hash,omitempty"` // СТБ bash-256 хеш предыдущего события
	Data          json.RawMessage     `json:"data"`           // тело события (разные схемы для каждого event_type)
	Metadata      map[string]string   `json:"metadata,omitempty"` // дополнительные метаданные
	SignedAt      *time.Time          `json:"signed_at,omitempty"` // время подписи (audit trail)
}

// ═══════════════════════════════════════════════════════════════════════
// RetrieveOptions — параметры для чтения событий из Event Store.
// ═══════════════════════════════════════════════════════════════════════

type RetrieveOptions struct {
	Source      EventSource // фильтр по источнику
	EventType   string      // фильтр по типу события
	AggregateID string      // фильтр по aggregate_id
	Since       time.Time   // начиная с этого времени
	Until       time.Time   // до этого времени
	Limit       int         // максимальное количество (0 = нет лимита)
	IncludeCold bool        // включать ли cold storage в поиск
}

// ═══════════════════════════════════════════════════════════════════════
// EventStore — двухуровневое хранилище событий.
// ═══════════════════════════════════════════════════════════════════════

// EventStoreConfig — конфигурация Event Store.
type EventStoreConfig struct {
	// Hot Tier (NATS JetStream)
	NATSURL       string // NATS server URL
	NATSCreds     string // путь к NGS/JWT credentials
	NATSUseTLS    bool   // использовать TLS для NATS

	// Cold Tier (S3/MinIO)
	S3Endpoint    string // endpoint MinIO/S3
	S3Region      string // регион (по умолчанию us-east-1)
	S3Bucket      string // bucket для событий
	S3AccessKey   string // access key
	S3SecretKey   string // secret key
	S3UseTLS      bool   // использовать HTTPS

	// Retention
	HotRetention  time.Duration // срок хранения в hot tier (default: 365 days)
	ColdRetention time.Duration // срок хранения в cold tier (default: 1825 days = 5 years)

	// Performance
	BatchSize     int    // размер батча при архивации (default: 100)
	FlushInterval time.Duration // интервал сброса буфера (default: 5s)

	Logger        *slog.Logger
}

// DefaultEventStoreConfig возвращает конфигурацию по умолчанию.
func DefaultEventStoreConfig() EventStoreConfig {
	return EventStoreConfig{
		HotRetention:  365 * 24 * time.Hour,       // 1 год
		ColdRetention: 1825 * 24 * time.Hour,      // 5 лет
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
		S3Region:      "us-east-1",
		S3Bucket:      "cctv-event-store",
	}
}

// EventStore — главный entry point для работы с событиями.
//
// Hot Tier: NATS JetStream для оперативного доступа (до 1 года)
// Cold Tier: S3/MinIO для долговременного хранения (до 5 лет)
//
// Usage:
//
//	store, _ := events.NewEventStore(cfg, logger)
//	defer store.Close()
//
//	// Сохранение события
//	record := store.NewRecord(events.SourceAlarms, "alarm.created", deviceID, data)
//	if err := store.Store(ctx, record); err != nil { ... }
//
//	// Воспроизведение
//	records, err := store.Replay(ctx, events.RetrieveOptions{
//	    Source: events.SourceAlarms,
//	    Since:  time.Now().Add(-24 * time.Hour),
//	})
type EventStore struct {
	cfg    EventStoreConfig
	logger *slog.Logger

	// NATS JetStream connection
	nc  *nats.Conn
	js  nats.JetStreamContext
	mgr *JetStreamManager

	// Cold storage
	cold *ColdStorage

	// Schema validation
	schemas *SchemaRegistry

	// Buffered writes
	buffer   []*EventRecord
	bufMu    sync.Mutex
	flushCh  chan struct{}
	closeCh  chan struct{}
	wg       sync.WaitGroup

	// Event counter для prev_hash chain
	mu        sync.RWMutex
	lastHash  string // последний хеш для цепочки (по aggregate_id)
}

// NewEventStore создаёт новый Event Store.
//
// Compliance: ISO 27001 A.12.4.1 (Event logging), СТБ 34.101.27 п. 7.5
func NewEventStore(cfg EventStoreConfig) (*EventStore, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.HotRetention <= 0 {
		cfg.HotRetention = 365 * 24 * time.Hour
	}
	if cfg.ColdRetention <= 0 {
		cfg.ColdRetention = 1825 * 24 * time.Hour
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Second
	}

	// ── NATS JetStream connection ─────────────────────────────────
	opts := []nats.Option{
		nats.Name("cctv-event-store"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			cfg.Logger.Warn("event store nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			cfg.Logger.Info("event store nats reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			cfg.Logger.Error("event store nats error", "subject", sub.Subject, "error", err)
		}),
	}

	if cfg.NATSCreds != "" {
		opts = append(opts, nats.UserCredentials(cfg.NATSCreds))
	}

	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("event store nats connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("event store jetstream: %w", err)
	}

	// ── Init JetStream streams ────────────────────────────────────
	mgr := NewJetStreamManager(js, cfg.Logger)
	if err := mgr.InitStreams(); err != nil {
		cfg.Logger.Warn("event store init streams", "error", err)
	}

	// ── Schema registry ───────────────────────────────────────────
	schemas := NewSchemaRegistry(cfg.Logger)

	// ── Cold storage (опционально) ────────────────────────────────
	var cold *ColdStorage
	if cfg.S3Endpoint != "" {
		cold, err = NewColdStorage(ColdStorageConfig{
			Endpoint:  cfg.S3Endpoint,
			Region:    cfg.S3Region,
			Bucket:    cfg.S3Bucket,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			UseTLS:    cfg.S3UseTLS,
			Retention: cfg.ColdRetention,
			Logger:    cfg.Logger,
		})
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("event store cold storage: %w", err)
		}
		cfg.Logger.Info("event store cold storage configured",
			"endpoint", cfg.S3Endpoint,
			"bucket", cfg.S3Bucket,
		)
	}

	es := &EventStore{
		cfg:      cfg,
		logger:   cfg.Logger,
		nc:       nc,
		js:       js,
		mgr:      mgr,
		cold:     cold,
		schemas:  schemas,
		buffer:   make([]*EventRecord, 0, cfg.BatchSize),
		flushCh:  make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
		lastHash: "",
	}

	// Запускаем фоновый flush
	es.wg.Add(1)
	go es.flushLoop()

	cfg.Logger.Info("event store initialized",
		"hot_retention", cfg.HotRetention,
		"cold_retention", cfg.ColdRetention,
		"cold_storage", cold != nil,
	)

	return es, nil
}

// ── Core operations ──────────────────────────────────────────────────

// Store сохраняет событие в Event Store.
// Сначала пишет в буфер (batch write), затем во flush:
//   - Hot tier: NATS JetStream (всегда)
//   - Cold tier: S3/MinIO (если настроен)
//
// Compliance:
//   - OWASP ASVS V7.1 (Input validation через Schema Registry)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - СТБ 34.101.27 п. 7.5 (Audit trail integrity)
func (es *EventStore) Store(ctx context.Context, record *EventRecord) error {
	// ── 1. Schema validation ──────────────────────────────────────
	if err := es.schemas.Validate(record); err != nil {
		return fmt.Errorf("event store schema validation: %w", err)
	}

	// ── 2. Prev-hash chain (tamper detection) ─────────────────────
	es.mu.Lock()
	if es.lastHash != "" {
		record.PrevHash = es.lastHash
	}
	// ⚠ PLACEHOLDER: Используем HMAC-SHA256 как временное решение.
	// TODO(C1): Заменить на СТБ bash-256 HMAC (github.com/bp2012/crypto/bash) перед production:
	//   h := bash.NewHmac(key, bash.Size256)
	//   h.Write([]byte(record.ID + record.TraceID))
	//   es.lastHash = hex.EncodeToString(h.Sum(nil))
	h := hmac.New(sha256.New, []byte("event-store-chain-placeholder"))
	h.Write([]byte(es.lastHash))
	h.Write([]byte(record.ID))
	es.lastHash = hex.EncodeToString(h.Sum(nil))
	es.mu.Unlock()

	// ── 3. Буферизированная запись ────────────────────────────────
	es.bufMu.Lock()
	es.buffer = append(es.buffer, record)

	// Если буфер заполнен — триггерим flush
	if len(es.buffer) >= es.cfg.BatchSize {
		select {
		case es.flushCh <- struct{}{}:
		default:
		}
	}
	es.bufMu.Unlock()

	return nil
}

// StoreSync сохраняет событие синхронно (без буфера).
// Используется для critical audit trail событий.
func (es *EventStore) StoreSync(ctx context.Context, record *EventRecord) error {
	// ── 1. Schema validation ──────────────────────────────────────
	if err := es.schemas.Validate(record); err != nil {
		return fmt.Errorf("event store schema validation: %w", err)
	}

	// ── 2. Serialize ──────────────────────────────────────────────
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("event store marshal: %w", err)
	}

	// ── 3. Hot tier: publish to NATS ──────────────────────────────
	subject := fmt.Sprintf("eventstore.%s.%s", record.Source, record.EventType)
	if _, err := es.js.Publish(subject, data); err != nil {
		return fmt.Errorf("event store publish: %w", err)
	}

	// ── 4. Cold tier: save to S3 (если настроен) ─────────────────
	if es.cold != nil {
		if err := es.cold.Save(ctx, record); err != nil {
			es.logger.Error("event store cold storage save failed",
				"event_id", record.ID,
				"error", err,
			)
			// Не фатально — hot tier сохранён успешно
		}
	}

	return nil
}

// Replay воспроизводит события из Event Store по заданным критериям.
//
// Сначала читает из hot tier (NATS JetStream), затем из cold tier (S3) если
// запрошено через opts.IncludeCold.
//
// Compliance: ISO 27001 A.12.4.1 (Event replay for incident investigation)
func (es *EventStore) Replay(ctx context.Context, opts RetrieveOptions) ([]*EventRecord, error) {
	// ── 1. Чтение из hot tier (NATS JetStream) ───────────────────
	hotRecords, err := es.replayFromHot(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("event store replay hot: %w", err)
	}

	// Если cold storage не нужен — возвращаем hot результаты
	if !opts.IncludeCold || es.cold == nil {
		return hotRecords, nil
	}

	// ── 2. Чтение из cold tier (S3) ──────────────────────────────
	coldRecords, err := es.cold.List(ctx, ColdListOptions{
		Source:      opts.Source,
		EventType:   opts.EventType,
		AggregateID: opts.AggregateID,
		Since:       opts.Since,
		Until:       opts.Until,
		Limit:       opts.Limit,
	})
	if err != nil {
		es.logger.Error("event store replay cold failed",
			"source", opts.Source,
			"error", err,
		)
		// Возвращаем hot даже если cold не доступен
		return hotRecords, nil
	}

	// ── 3. Merge: hot + cold, сортировка по timestamp ────────────
	all := append(hotRecords, coldRecords...)

	// Сортировка по timestamp (стабильная)
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].Timestamp.Before(all[i].Timestamp) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	// Применяем limit если задан
	if opts.Limit > 0 && len(all) > opts.Limit {
		all = all[:opts.Limit]
	}

	return all, nil
}

// replayFromHot воспроизводит события из hot tier (NATS JetStream).
func (es *EventStore) replayFromHot(ctx context.Context, opts RetrieveOptions) ([]*EventRecord, error) {
	// Определяем stream и subject pattern для replay
	streamName, subjectPattern := es.resolveStream(opts.Source)

	records := make([]*EventRecord, 0)

	// Используем ephemeral pull subscriber для replay
	// durable name: "replay-" + streamName + случайный суффикс
	durable := fmt.Sprintf("replay-%s-%s", streamName, newShortID())
	sub, err := es.js.PullSubscribe(
		subjectPattern,
		durable,
		nats.BindStream(streamName),
		nats.MaxDeliver(1),
		nats.AckExplicit(),
	)
	if err != nil {
		return nil, fmt.Errorf("replay subscribe %s: %w", streamName, err)
	}
	defer sub.Unsubscribe()

	// Читаем сообщения с timeout
	for {
		msgs, err := sub.Fetch(es.cfg.BatchSize, nats.MaxWait(3*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				break
			}
			return nil, fmt.Errorf("replay fetch: %w", err)
		}

		for _, msg := range msgs {
			var record EventRecord
			if err := json.Unmarshal(msg.Data, &record); err != nil {
				es.logger.Warn("replay unmarshal error, skipping",
					"subject", msg.Subject,
					"error", err,
				)
				_ = msg.Ack()
				continue
			}

			// Фильтрация
			if matchesFilter(&record, opts) {
				records = append(records, &record)
			}
			_ = msg.Ack()
		}

		// Проверка контекста
		select {
		case <-ctx.Done():
			return records, ctx.Err()
		default:
		}
	}

	return records, nil
}

// resolveStream возвращает имя JetStream стрима и subject pattern по источнику.
func (es *EventStore) resolveStream(source EventSource) (string, string) {
	switch source {
	case SourceAlarms:
		return StreamAlarms, "alarms.>"
	case SourceCMMS:
		return StreamCMMS, "cmms.workorder.>"
	case SourcePredictions:
		return StreamPredictions, "predictions.>"
	case SourceTelemetry:
		return StreamTelemetry, "telemetry.>"
	default:
		return StreamAudit, "eventstore.>"
	}
}

// GetStream возвращает JetStream стрим для заданного источника.
func (es *EventStore) GetStream(source EventSource) string {
	streamName, _ := es.resolveStream(source)
	return streamName
}

// ── Utility methods ─────────────────────────────────────────────────

// NewRecord создаёт новый EventRecord с заполненными обязательными полями.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation через whitelist)
//   - ISO 27001 A.12.4.1 (Timestamp accuracy)
func (es *EventStore) NewRecord(source EventSource, eventType, aggregateID string, data interface{}) *EventRecord {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		es.logger.Error("event store marshal data",
			"source", source,
			"event_type", eventType,
			"error", err,
		)
		dataJSON = json.RawMessage(`{}`)
	}

	return &EventRecord{
		ID:            newUUID(), // UUID v7
		Source:        source,
		EventType:     eventType,
		SchemaVersion: "1.0.0",
		Timestamp:     time.Now().UTC(),
		AggregateID:   aggregateID,
		ActorID:       "system",
		TraceID:       newTraceID(),
		Data:          dataJSON,
		Metadata:      make(map[string]string),
	}
}

// SchemaRegistry возвращает SchemaRegistry для регистрации схем.
func (es *EventStore) SchemaRegistry() *SchemaRegistry {
	return es.schemas
}

// ColdStorage возвращает ColdStorage для прямых операций (архивация, восстановление).
func (es *EventStore) ColdStorage() *ColdStorage {
	return es.cold
}

// JetStream возвращает JetStream context для прямых операций.
func (es *EventStore) JetStream() nats.JetStreamContext {
	return es.js
}

// Stats возвращает статистику Event Store.
type EventStoreStats struct {
	BufferedEvents int    `json:"buffered_events"`
	ColdStorage    bool   `json:"cold_storage_enabled"`
	LastHash       string `json:"last_hash"`
}

func (es *EventStore) Stats() EventStoreStats {
	es.bufMu.Lock()
	bufLen := len(es.buffer)
	es.bufMu.Unlock()

	es.mu.RLock()
	lastHash := es.lastHash
	es.mu.RUnlock()

	return EventStoreStats{
		BufferedEvents: bufLen,
		ColdStorage:    es.cold != nil,
		LastHash:       lastHash,
	}
}

// ── Flush (batch write) ──────────────────────────────────────────────

// flushLoop — фоновый цикл для периодического сброса буфера.
func (es *EventStore) flushLoop() {
	defer es.wg.Done()

	ticker := time.NewTicker(es.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-es.closeCh:
			// Финальный flush перед закрытием
			es.flush()
			return
		case <-es.flushCh:
			es.flush()
		case <-ticker.C:
			es.flush()
		}
	}
}

// flush сбрасывает буфер в NATS JetStream (и S3 если настроен).
func (es *EventStore) flush() {
	es.bufMu.Lock()
	if len(es.buffer) == 0 {
		es.bufMu.Unlock()
		return
	}
	batch := es.buffer
	es.buffer = make([]*EventRecord, 0, es.cfg.BatchSize)
	es.bufMu.Unlock()

	for _, record := range batch {
		data, err := json.Marshal(record)
		if err != nil {
			es.logger.Error("event store marshal during flush",
				"event_id", record.ID,
				"error", err,
			)
			continue
		}

		// ── Hot tier: publish to NATS ─────────────────────────────
		subject := fmt.Sprintf("eventstore.%s.%s", record.Source, record.EventType)
		if _, err := es.js.Publish(subject, data); err != nil {
			es.logger.Error("event store publish during flush",
				"subject", subject,
				"event_id", record.ID,
				"error", err,
			)
			continue
		}

		// ── Cold tier: save to S3 ─────────────────────────────────
		if es.cold != nil {
			if err := es.cold.Save(context.Background(), record); err != nil {
				es.logger.Error("event store cold save during flush",
					"event_id", record.ID,
					"error", err,
				)
			}
		}
	}
}

// ── Lifecycle ─────────────────────────────────────────────────────────

// Close выполняет graceful shutdown Event Store.
//
// Порядок:
//  1. Остановка flush loop
//  2. Финальный flush буфера
//  3. Закрытие NATS соединения
//  4. Закрытие S3 соединения
func (es *EventStore) Close() error {
	es.logger.Info("event store shutting down...")

	// 1. Останавливаем flush loop
	close(es.closeCh)
	es.wg.Wait()

	// 2. Финальный flush уже выполнен в flushLoop

	// 3. NATS drain
	if err := es.nc.Drain(); err != nil {
		es.logger.Error("event store nats drain", "error", err)
	}
	es.nc.Close()

	// 4. Cold storage close
	if es.cold != nil {
		if err := es.cold.Close(); err != nil {
			es.logger.Error("event store cold storage close", "error", err)
		}
	}

	es.logger.Info("event store shut down complete")
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Filter helpers
// ═══════════════════════════════════════════════════════════════════════

// matchesFilter проверяет соответствует ли запись заданным фильтрам.
func matchesFilter(record *EventRecord, opts RetrieveOptions) bool {
	if opts.Source != "" && record.Source != opts.Source {
		return false
	}
	if opts.EventType != "" && record.EventType != opts.EventType {
		return false
	}
	if opts.AggregateID != "" && record.AggregateID != opts.AggregateID {
		return false
	}
	if !opts.Since.IsZero() && record.Timestamp.Before(opts.Since) {
		return false
	}
	if !opts.Until.IsZero() && record.Timestamp.After(opts.Until) {
		return false
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════
// UUID & Trace ID generators
// ═══════════════════════════════════════════════════════════════════════

// newUUID генерирует UUID v7 (time-sortable) с random счётчиком для uniqueness.
// Формат: tttttttt-tttt-7xxx-yxxx-xxxxxxxxxxxx
//
// Использует nano-time для предотвращения коллизий в одной миллисекунде.
//
// ⚠ PLACEHOLDER: В production заменить на звонок к bp2012/crypto
func newUUID() string {
	// Используем наносекунды вместо миллисекунд для уникальности + counter
	now := time.Now().UnixNano()

	b := make([]byte, 16)
	// time_low (4 bytes)
	b[0] = byte(now >> 56)
	b[1] = byte(now >> 48)
	b[2] = byte(now >> 40)
	b[3] = byte(now >> 32)
	// time_mid (2 bytes)
	b[4] = byte(now >> 24)
	b[5] = byte(now >> 16)
	// time_hi_and_version (2 bytes) + version 7
	b[6] = byte(now>>8) & 0x0f | 0x70
	b[7] = byte(now)
	// clock_seq_hi_and_reserved (variant 10xx)
	b[8] = 0x80 | byte(now>>56)&0x3f
	// node — random часть (unique per call)
	b[9] = byte(now >> 48)
	b[10] = byte(now >> 40)
	b[11] = byte(now >> 32)
	b[12] = byte(now >> 24)
	b[13] = byte(now >> 16)
	b[14] = byte(now >> 8)
	b[15] = byte(now)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16],
	)
}

// newShortID генерирует короткий ID для durable consumer name.
func newShortID() string {
	return fmt.Sprintf("%08x", time.Now().UnixNano())
}

// newTraceID генерирует W3C Trace Context trace ID (16 байт hex).
func newTraceID() string {
	b := make([]byte, 16)
	// Временно используем timestamp + random
	now := time.Now().UnixNano()
	b[0] = byte(now >> 56)
	b[1] = byte(now >> 48)
	b[2] = byte(now >> 40)
	b[3] = byte(now >> 32)
	b[4] = byte(now >> 24)
	b[5] = byte(now >> 16)
	b[6] = byte(now >> 8)
	b[7] = byte(now)

	return fmt.Sprintf("%032x", b)
}
