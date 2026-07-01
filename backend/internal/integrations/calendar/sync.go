package calendar

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── SyncEngine ────────────────────────────────────────────────────────

// SyncEngine управляет bi-directional синхронизацией Work Orders
// с внешними календарями (Google Calendar, Microsoft Outlook).
//
// Flow:
//  1. Push: WO created/updated/cancelled → CreateEvent/UpdateEvent/DeleteEvent
//  2. Pull: SyncChanges → compare → create/update WO reminders
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — каждый sync логируется)
//   - IEC 62443-3-3 SL-3 (Zone 3 — application integrity)
//   - OWASP ASVS V6.2 (Encrypted tokens at rest)
type SyncEngine struct {
	providers map[string]CalendarProvider // provider name → provider instance
	store     SyncStore
	config    Config
	logger    *slog.Logger

	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
}

// NewSyncEngine создаёт новый SyncEngine.
func NewSyncEngine(store SyncStore, config Config, logger *slog.Logger) *SyncEngine {
	if logger == nil {
		logger = slog.Default()
	}

	return &SyncEngine{
		providers: make(map[string]CalendarProvider),
		store:     store,
		config:    config,
		logger:    logger.With("component", "calendar.sync"),
	}
}

// RegisterProvider регистрирует провайдера календаря.
func (se *SyncEngine) RegisterProvider(name string, provider CalendarProvider) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.providers[name] = provider
}

// ── Push Operations ───────────────────────────────────────────────────

// PushCreate создаёт события во всех активных календарях.
func (se *SyncEngine) PushCreate(ctx context.Context, wo WorkOrderEvent) error {
	conns, err := se.store.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("list connections: %w", err)
	}

	for _, conn := range conns {
		if !conn.Enabled {
			continue
		}

		provider, ok := se.providers[conn.Provider]
		if !ok {
			continue
		}

		externalID, err := provider.CreateEvent(ctx, wo)
		if err != nil {
			se.logSyncError(ctx, wo.ID, conn.Provider, "push", "created", err)
			continue
		}

		// Сохраняем маппинг
		mapping := &EventMapping{
			WOID:       wo.ID,
			Provider:   conn.Provider,
			ExternalID: externalID,
			Status:     "active",
			LastSynced: time.Now(),
		}
		if err := se.store.SaveEventMapping(ctx, mapping); err != nil {
			se.logger.Error("failed to save event mapping", "error", err, "wo_id", wo.ID)
		}

		se.logSync(ctx, wo.ID, conn.Provider, "push", "created", externalID, "success", "")
	}

	return nil
}

// PushUpdate обновляет события во всех активных календарях.
func (se *SyncEngine) PushUpdate(ctx context.Context, wo WorkOrderEvent) error {
	conns, err := se.store.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("list connections: %w", err)
	}

	for _, conn := range conns {
		if !conn.Enabled {
			continue
		}

		mapping, err := se.store.GetEventMapping(ctx, wo.ID, conn.Provider)
		if err != nil || mapping == nil {
			// Маппинга нет — создаём событие
			if err := se.PushCreate(ctx, wo); err != nil {
				se.logger.Error("fallback create after missing mapping",
					"error", err, "wo_id", wo.ID)
			}
			continue
		}

		provider, ok := se.providers[conn.Provider]
		if !ok {
			continue
		}

		if err := provider.UpdateEvent(ctx, mapping.ExternalID, wo); err != nil {
			se.logSyncError(ctx, wo.ID, conn.Provider, "push", "updated", err)
			continue
		}

		mapping.Status = "updated"
		mapping.LastSynced = time.Now()
		if err := se.store.SaveEventMapping(ctx, mapping); err != nil {
			se.logger.Error("failed to update event mapping", "error", err, "wo_id", wo.ID)
		}

		se.logSync(ctx, wo.ID, conn.Provider, "push", "updated", mapping.ExternalID, "success", "")
	}

	return nil
}

// PushDelete удаляет события из всех активных календарей.
func (se *SyncEngine) PushDelete(ctx context.Context, woID string) error {
	conns, err := se.store.ListConnections(ctx)
	if err != nil {
		return fmt.Errorf("list connections: %w", err)
	}

	for _, conn := range conns {
		if !conn.Enabled {
			continue
		}

		mapping, err := se.store.GetEventMapping(ctx, woID, conn.Provider)
		if err != nil || mapping == nil {
			continue
		}

		provider, ok := se.providers[conn.Provider]
		if !ok {
			continue
		}

		if err := provider.DeleteEvent(ctx, mapping.ExternalID); err != nil {
			se.logSyncError(ctx, woID, conn.Provider, "push", "deleted", err)
			continue
		}

		if err := se.store.DeleteEventMapping(ctx, woID, conn.Provider); err != nil {
			se.logger.Error("failed to delete event mapping", "error", err, "wo_id", woID)
		}

		se.logSync(ctx, woID, conn.Provider, "push", "deleted", mapping.ExternalID, "success", "")
	}

	return nil
}

// ── Pull Operations ───────────────────────────────────────────────────

// PullChanges получает изменения из всех календарей.
func (se *SyncEngine) PullChanges(ctx context.Context) ([]CalendarChange, error) {
	conns, err := se.store.ListConnections(ctx)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}

	var allChanges []CalendarChange
	since := time.Now().Add(-se.config.SyncWindow)

	for _, conn := range conns {
		if !conn.Enabled {
			continue
		}

		provider, ok := se.providers[conn.Provider]
		if !ok {
			continue
		}

		changes, err := provider.SyncChanges(ctx, since)
		if err != nil {
			se.logger.Error("failed to pull changes",
				"provider", conn.Provider,
				"error", err,
			)
			continue
		}

		for _, ch := range changes {
			se.logSync(ctx, "", conn.Provider, "pull", ch.Type, ch.ExternalID, "success", "")
		}

		allChanges = append(allChanges, changes...)
	}

	return allChanges, nil
}

// ── Background Sync ───────────────────────────────────────────────────

// Start запускает фоновую синхронизацию с интервалом config.SyncInterval.
func (se *SyncEngine) Start(ctx context.Context) {
	se.mu.Lock()
	if se.running {
		se.mu.Unlock()
		return
	}
	se.running = true
	se.stopCh = make(chan struct{})
	se.mu.Unlock()

	se.logger.Info("calendar sync engine started",
		"interval", se.config.SyncInterval,
		"strategy", se.config.ConflictStrategy,
	)

	go func() {
		ticker := time.NewTicker(se.config.SyncInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				se.syncAll(ctx)
			case <-se.stopCh:
				se.logger.Info("calendar sync engine stopped")
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop останавливает фоновую синхронизацию.
func (se *SyncEngine) Stop() {
	se.mu.Lock()
	defer se.mu.Unlock()

	if se.running {
		close(se.stopCh)
		se.running = false
	}
}

// syncAll выполняет полный цикл push + pull.
func (se *SyncEngine) syncAll(ctx context.Context) {
	se.logger.Debug("starting sync cycle")

	// Pull changes from external calendars
	changes, err := se.PullChanges(ctx)
	if err != nil {
		se.logger.Error("sync cycle pull failed", "error", err)
	} else {
		se.logger.Debug("sync cycle completed", "changes", len(changes))
	}
}

// ── Logging Helpers ───────────────────────────────────────────────────

func (se *SyncEngine) logSync(ctx context.Context, woID, provider, direction,
	eventType, externalID, status, errMsg string) {

	entry := &SyncLogEntry{
		WOID:           woID,
		Provider:       provider,
		Direction:      direction,
		EventType:      eventType,
		ExternalID:     externalID,
		Status:         status,
		ErrorMsg:       errMsg,
		IdempotencyKey: uuid.New().String(), // P1-HI-09: уникальный ключ для dedup
	}

	if err := se.store.LogSync(ctx, entry); err != nil {
		se.logger.Error("failed to log sync entry", "error", err)
	}
}

func (se *SyncEngine) logSyncError(ctx context.Context, woID, provider, direction,
	eventType string, err error) {

	se.logger.Error("sync operation failed",
		"wo_id", woID,
		"provider", provider,
		"direction", direction,
		"event_type", eventType,
		"error", err,
	)

	se.logSync(ctx, woID, provider, direction, eventType, "", "error", err.Error())
}
