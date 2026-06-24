// Package events — Projection Builder (CQRS read-model из Event Store).
//
// Реализует DM-1.2.3: построение read-моделей из событий Event Store.
//
// Архитектура:
//
//	Event Store (NATS JetStream + S3)
//	     ↓ replay / subscribe
//	ProjectionBuilder
//	     ↓ Handle(event)
//	┌─────────────────┬──────────────────┬──────────────────┐
//	│ WorkOrderProj.  │ SLAProj.         │ TechnicianProj.  │
//	│ (WO statuses)   │ (SLA compliance) │ (workload)       │
//	└─────────────────┴──────────────────┴──────────────────┘
//	     ↓ flush
//	Projection Store (PostgreSQL / in-memory / Redis)
//
// Compliance:
//   - CQRS pattern (Martin Fowler)
//   - Event Sourcing (immutable event log → materialized views)
//   - ISO 27001 A.12.4.1 (Event logging — replay для расследований)
//   - ISO 27001 A.12.6.1 (Capacity management — workload tracking)
//   - IEC 62443 SR 7.1 (Resource availability — SLA monitoring)
package events

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Projection interface
// ═══════════════════════════════════════════════════════════════════════

// Projection — интерфейс для read-модели (CQRS projection).
//
// Каждая проекция:
//   - Handle(event) — обрабатывает событие и обновляет read-модель
//   - Name() — возвращает имя проекции (для идентификации)
//   - Rebuild(ctx, store) — перестраивает проекцию из всех событий
//   - Snapshot() — возвращает текущее состояние проекции
type Projection interface {
	// Name возвращает уникальное имя проекции.
	Name() string

	// Handle обрабатывает событие и обновляет read-модель.
	// Возвращает error если событие не может быть обработано.
	Handle(ctx context.Context, record *EventRecord) error

	// Rebuild перестраивает проекцию из Event Store с нуля.
	Rebuild(ctx context.Context, store *EventStore) error

	// Snapshot возвращает текущее состояние проекции для сохранения.
	Snapshot() ([]byte, error)

	// Restore восстанавливает состояние проекции из снепшота.
	Restore(data []byte) error
}

// ═══════════════════════════════════════════════════════════════════════
// ProjectionBuilder — управляет lifecycle всех проекций.
// ═══════════════════════════════════════════════════════════════════════

// ProjectionBuilderConfig — конфигурация Projection Builder.
type ProjectionBuilderConfig struct {
	RebuildOnStart bool          // перестраивать проекции при старте
	FlushInterval  time.Duration // интервал сохранения снепшотов (0 = отключено)
	AutoSubscribe  bool          // автоматически подписываться на новые события
	Logger         *slog.Logger
}

// DefaultProjectionBuilderConfig возвращает конфигурацию по умолчанию.
func DefaultProjectionBuilderConfig() ProjectionBuilderConfig {
	return ProjectionBuilderConfig{
		RebuildOnStart: true,
		FlushInterval:  5 * time.Minute,
		AutoSubscribe:  true,
	}
}

// ProjectionBuilder управляет набором проекций.
//
// Responsibilities:
//   - Регистрация и управление lifecycle проекций
//   - Replay событий из Event Store для перестройки
//   - Подписка на новые события (live update)
//   - Периодическое сохранение снепшотов
//   - Graceful shutdown
type ProjectionBuilder struct {
	cfg      ProjectionBuilderConfig
	store    *EventStore
	logger   *slog.Logger

	mu        sync.RWMutex
	projections map[string]Projection

	// Snapshot persistence
	snapshotter SnapshotStore

	// Lifecycle
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// SnapshotStore — интерфейс для сохранения/восстановления снепшотов проекций.
type SnapshotStore interface {
	// SaveSnapshot сохраняет снепшот проекции.
	SaveSnapshot(ctx context.Context, name string, data []byte) error

	// LoadSnapshot загружает снепшот проекции.
	LoadSnapshot(ctx context.Context, name string) ([]byte, error)

	// DeleteSnapshot удаляет снепшот проекции.
	DeleteSnapshot(ctx context.Context, name string) error
}

// NewProjectionBuilder создаёт новый ProjectionBuilder.
func NewProjectionBuilder(store *EventStore, snapshotter SnapshotStore, cfg ProjectionBuilderConfig) *ProjectionBuilder {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 5 * time.Minute
	}

	return &ProjectionBuilder{
		cfg:         cfg,
		store:       store,
		logger:      cfg.Logger,
		projections: make(map[string]Projection),
		snapshotter: snapshotter,
		closeCh:     make(chan struct{}),
	}
}

// RegisterProjection регистрирует проекцию.
func (pb *ProjectionBuilder) RegisterProjection(p Projection) error {
	name := p.Name()
	if name == "" {
		return fmt.Errorf("projection name is required")
	}

	pb.mu.Lock()
	defer pb.mu.Unlock()

	if _, exists := pb.projections[name]; exists {
		return fmt.Errorf("projection %q already registered", name)
	}

	pb.projections[name] = p
	pb.logger.Info("projection registered", "name", name)
	return nil
}

// Start запускает все зарегистрированные проекции.
//
// Порядок:
//  1. Восстановление из снепшотов (если есть)
//  2. Перестройка из Event Store (если RebuildOnStart)
//  3. Запуск фонового сохранения снепшотов
func (pb *ProjectionBuilder) Start(ctx context.Context) error {
	pb.mu.RLock()
	names := make([]string, 0, len(pb.projections))
	for name := range pb.projections {
		names = append(names, name)
	}
	pb.mu.RUnlock()

	if len(names) == 0 {
		pb.logger.Warn("projection builder started with no projections")
		return nil
	}

	pb.logger.Info("projection builder starting...", "count", len(names))

	for _, name := range names {
		pb.mu.RLock()
		p := pb.projections[name]
		pb.mu.RUnlock()

		// 1. Восстановление из снепшота
		if pb.snapshotter != nil {
			data, err := pb.snapshotter.LoadSnapshot(ctx, name)
			if err == nil && len(data) > 0 {
				if err := p.Restore(data); err != nil {
					pb.logger.Warn("projection restore failed, will rebuild",
						"name", name, "error", err,
					)
				} else {
					pb.logger.Info("projection restored from snapshot", "name", name)
					continue
				}
			}
		}

		// 2. Перестройка из Event Store
		if pb.cfg.RebuildOnStart {
			pb.logger.Info("rebuilding projection from events", "name", name)
			if err := p.Rebuild(ctx, pb.store); err != nil {
				return fmt.Errorf("rebuild projection %q: %w", name, err)
			}
			pb.logger.Info("projection rebuilt", "name", name)
		}
	}

	// 3. Запуск фонового сохранения снепшотов
	if pb.cfg.FlushInterval > 0 && pb.snapshotter != nil {
		pb.wg.Add(1)
		go pb.snapshotLoop(ctx)
	}

	pb.logger.Info("projection builder started", "projections", len(names))
	return nil
}

// HandleEvent передаёт событие всем зарегистрированным проекциям.
func (pb *ProjectionBuilder) HandleEvent(ctx context.Context, record *EventRecord) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	for name, p := range pb.projections {
		if err := p.Handle(ctx, record); err != nil {
			pb.logger.Error("projection handle failed",
				"projection", name,
				"event_id", record.ID,
				"event_type", record.EventType,
				"error", err,
			)
		}
	}
}

// GetProjection возвращает проекцию по имени.
func (pb *ProjectionBuilder) GetProjection(name string) (Projection, bool) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	p, ok := pb.projections[name]
	return p, ok
}

// ListProjections возвращает список всех зарегистрированных проекций.
func (pb *ProjectionBuilder) ListProjections() []string {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	names := make([]string, 0, len(pb.projections))
	for name := range pb.projections {
		names = append(names, name)
	}
	return names
}

// ── Snapshot persistence ──────────────────────────────────────────────

// snapshotLoop периодически сохраняет снепшоты проекций.
func (pb *ProjectionBuilder) snapshotLoop(ctx context.Context) {
	defer pb.wg.Done()

	ticker := time.NewTicker(pb.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pb.closeCh:
			pb.saveAllSnapshots(context.Background())
			return
		case <-ticker.C:
			pb.saveAllSnapshots(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (pb *ProjectionBuilder) saveAllSnapshots(ctx context.Context) {
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	for name, p := range pb.projections {
		data, err := p.Snapshot()
		if err != nil {
			pb.logger.Error("projection snapshot failed",
				"name", name, "error", err,
			)
			continue
		}

		if pb.snapshotter != nil {
			if err := pb.snapshotter.SaveSnapshot(ctx, name, data); err != nil {
				pb.logger.Error("projection snapshot save failed",
					"name", name, "error", err,
				)
			}
		}
	}
}

// ── Lifecycle ─────────────────────────────────────────────────────────

// Close выполняет graceful shutdown Projection Builder.
func (pb *ProjectionBuilder) Close() error {
	pb.logger.Info("projection builder shutting down...")

	close(pb.closeCh)
	pb.wg.Wait()

	// Финальное сохранение снепшотов (уже выполнено в snapshotLoop)
	pb.logger.Info("projection builder shut down")
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// InMemorySnapshotStore — in-memory реализация SnapshotStore для dev.
// ═══════════════════════════════════════════════════════════════════════

// InMemorySnapshotStore хранит снепшоты в памяти.
// Для production использовать PostgreSQL или Redis.
type InMemorySnapshotStore struct {
	mu    sync.RWMutex
	data  map[string][]byte
}

// NewInMemorySnapshotStore создаёт in-memory snapshot store.
func NewInMemorySnapshotStore() *InMemorySnapshotStore {
	return &InMemorySnapshotStore{
		data: make(map[string][]byte),
	}
}

func (s *InMemorySnapshotStore) SaveSnapshot(_ context.Context, name string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[name] = data
	return nil
}

func (s *InMemorySnapshotStore) LoadSnapshot(_ context.Context, name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.data[name]
	if !ok {
		return nil, fmt.Errorf("snapshot %q not found", name)
	}
	return data, nil
}

func (s *InMemorySnapshotStore) DeleteSnapshot(_ context.Context, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, name)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// BaseProjection — вспомогательный базовый тип для проекций.
// ═══════════════════════════════════════════════════════════════════════

// BaseProjection содержит общую логику для проекций.
type BaseProjection struct {
	name    string
	store   *EventStore
	logger  *slog.Logger
	handler func(ctx context.Context, record *EventRecord) error
}

// NewBaseProjection создаёт базовую проекцию.
func NewBaseProjection(name string, store *EventStore, handler func(ctx context.Context, record *EventRecord) error) *BaseProjection {
	return &BaseProjection{
		name:    name,
		store:   store,
		logger:  slog.Default().With("projection", name),
		handler: handler,
	}
}

func (bp *BaseProjection) Name() string {
	return bp.name
}

func (bp *BaseProjection) Handle(ctx context.Context, record *EventRecord) error {
	return bp.handler(ctx, record)
}

func (bp *BaseProjection) Rebuild(ctx context.Context, store *EventStore) error {
	// Перестройка: replay всех событий из Event Store
	opts := RetrieveOptions{
		IncludeCold: true,
		Limit:       0, // без лимита
	}

	records, err := store.Replay(ctx, opts)
	if err != nil {
		return fmt.Errorf("rebuild replay: %w", err)
	}

	bp.logger.Info("rebuilding projection", "total_events", len(records))

	for _, record := range records {
		if err := bp.Handle(ctx, record); err != nil {
			bp.logger.Warn("rebuild handle failed, skipping",
				"event_id", record.ID,
				"error", err,
			)
			continue
		}
	}

	bp.logger.Info("projection rebuild complete", "events_processed", len(records))
	return nil
}

// Snapshot возвращает ошибку — BaseProjection не хранит состояние.
// Конкретные проекции должны реализовать свой Snapshot/Restore.
func (bp *BaseProjection) Snapshot() ([]byte, error) {
	return nil, fmt.Errorf("projection %q: Snapshot not implemented — override in concrete projection", bp.name)
}

// Restore возвращает ошибку — BaseProjection не хранит состояние.
func (bp *BaseProjection) Restore(data []byte) error {
	return fmt.Errorf("projection %q: Restore not implemented — override in concrete projection", bp.name)
}
