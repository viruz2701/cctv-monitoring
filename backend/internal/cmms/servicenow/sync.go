// Package servicenow — Bi-Directional Sync State Machine.
//
// INT-01: ServiceNow Bi-Directional Sync.
//
// Обеспечивает двунаправленную синхронизацию между CCTV Health Monitor
// и ServiceNow с использованием state machine + conflict resolution.
//
// ── Архитектура ───────────────────────────────────────────────────────
//
//	CCTV Local DB  ←→  SyncStateMachine  ←→  ServiceNow Instance
//	                        ↕
//	                   Conflict Resolver
//	                   (last-write-wins)
//
// ── Состояния синхронизации ──────────────────────────────────────────
//
//	synced       — данные согласованы
//	pending_local  — изменения только в CCTV (ожидают отправки в SN)
//	pending_remote — изменения только в SN (ожидают получения в CCTV)
//	conflict     — изменения в обеих системах (требуется resolution)
//	failed       — ошибка синхронизации (требуется retry)
//
// ── Соответствие стандартам ──────────────────────────────────────────
//
//	ISO 27001 A.12.4.1 (Event logging — все синхронизации логируются)
//	ISO 27001 A.12.6.1 (Capacity management — sync queue)
//	IEC 62443 SR 7.1 (Resource availability — graceful degradation)
//	OWASP ASVS V7.1 (Log content — no sensitive data in sync logs)
//
// ═══════════════════════════════════════════════════════════════════════
package servicenow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ── Sync Status ───────────────────────────────────────────────────────

// SyncStatus — состояние синхронизации записи.
type SyncStatus string

const (
	SyncStatusSynced        SyncStatus = "synced"
	SyncStatusPendingLocal  SyncStatus = "pending_local"
	SyncStatusPendingRemote SyncStatus = "pending_remote"
	SyncStatusConflict      SyncStatus = "conflict"
	SyncStatusFailed        SyncStatus = "failed"
)

// ── Entity Types ──────────────────────────────────────────────────────

// SyncEntityType — тип синхронизируемой сущности.
type SyncEntityType string

const (
	SyncEntityWorkOrder SyncEntityType = "work_order"
	SyncEntityAsset     SyncEntityType = "asset"
	SyncEntityPart      SyncEntityType = "spare_part"
	SyncEntitySchedule  SyncEntityType = "maintenance_schedule"
)

// ── Status Mapping ────────────────────────────────────────────────────

// StatusMapping — матрица маппинга статусов CCTV ↔ ServiceNow.
//
// INT-01: Bi-directional status mapping.
// Ключ: CCTV status, Значение: ServiceNow status.
var cctvToSNStatus = map[string]string{
	"REQUESTED":   "u_requested",
	"APPROVED":    "u_approved",
	"ASSIGNED":    "u_assigned",
	"IN_PROGRESS": "in_progress",
	"ON_HOLD":     "on_hold",
	"COMPLETED":   "completed",
	"CANCELLED":   "cancelled",
	"CLOSED":      "closed",
}

// snToCCTVStatus — обратный маппинг.
var snToCCTVStatus = map[string]string{
	"u_requested": "REQUESTED",
	"u_approved":  "APPROVED",
	"u_assigned":  "ASSIGNED",
	"in_progress": "IN_PROGRESS",
	"on_hold":     "ON_HOLD",
	"completed":   "COMPLETED",
	"cancelled":   "CANCELLED",
	"closed":      "CLOSED",
}

// ── Sync Record ───────────────────────────────────────────────────────

// SyncRecord — запись состояния синхронизации.
type SyncRecord struct {
	ID            string          `json:"id" db:"id"`
	EntityType    SyncEntityType  `json:"entity_type" db:"entity_type"`
	EntityID      string          `json:"entity_id" db:"entity_id"`
	RemoteID      string          `json:"remote_id" db:"remote_id"` // ServiceNow sys_id
	Status        SyncStatus      `json:"status" db:"status"`
	LocalVersion  int64           `json:"local_version" db:"local_version"`
	RemoteVersion int64           `json:"remote_version" db:"remote_version"`
	LastSyncAt    *time.Time      `json:"last_sync_at" db:"last_sync_at"`
	LastError     string          `json:"last_error,omitempty" db:"last_error"`
	ConflictData  json.RawMessage `json:"conflict_data,omitempty" db:"conflict_data"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// ── Sync State Machine ───────────────────────────────────────────────

// SyncStateMachine — state machine для bi-directional sync.
type SyncStateMachine struct {
	mu      sync.RWMutex
	logger  *slog.Logger
	adapter *Adapter
	client  *Client
	config  SyncConfig

	// In-memory sync records (в production — БД)
	records map[string]*SyncRecord
}

// SyncConfig — конфигурация синхронизации.
type SyncConfig struct {
	// SyncInterval — интервал между циклами синхронизации
	SyncInterval time.Duration `json:"sync_interval"`
	// MaxRetries — максимальное количество попыток
	MaxRetries int `json:"max_retries"`
	// RetryDelay — задержка между попытками
	RetryDelay time.Duration `json:"retry_delay"`
	// ConflictStrategy — стратегия разрешения конфликтов
	ConflictStrategy string `json:"conflict_strategy"` // "local_wins" | "remote_wins" | "manual"
}

// DefaultSyncConfig — конфигурация по умолчанию.
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		SyncInterval:     5 * time.Minute,
		MaxRetries:       3,
		RetryDelay:       30 * time.Second,
		ConflictStrategy: "remote_wins",
	}
}

// NewSyncStateMachine создаёт SyncStateMachine.
func NewSyncStateMachine(adapter *Adapter, client *Client, logger *slog.Logger, config SyncConfig) *SyncStateMachine {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncStateMachine{
		logger:  logger.With("component", "sn-sync"),
		adapter: adapter,
		client:  client,
		records: make(map[string]*SyncRecord),
		config:  config,
	}
}

// ── Core Sync Operations ─────────────────────────────────────────────

// MarkLocalChange отмечает, что запись изменена локально (CCTV).
//
// Вызывается при создании/обновлении WO/Asset в CCTV.
func (sm *SyncStateMachine) MarkLocalChange(ctx context.Context, entityType SyncEntityType, entityID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	key := sm.recordKey(entityType, entityID)
	record, ok := sm.records[key]
	if !ok {
		record = &SyncRecord{
			EntityType: entityType,
			EntityID:   entityID,
			Status:     SyncStatusPendingLocal,
		}
		sm.records[key] = record
	}

	switch record.Status {
	case SyncStatusSynced:
		record.Status = SyncStatusPendingLocal
	case SyncStatusPendingRemote:
		// Изменения в обеих системах → conflict
		record.Status = SyncStatusConflict
		record.ConflictData = sm.captureConflictData(entityType, entityID)
	case SyncStatusConflict, SyncStatusFailed:
		// Уже в конфликте/ошибке — не меняем
		return nil
	}

	record.LocalVersion++
	record.UpdatedAt = time.Now()
	sm.logger.Info("sync: local change marked",
		"type", entityType, "entity_id", entityID,
		"status", record.Status, "version", record.LocalVersion,
	)

	return nil
}

// MarkRemoteChange отмечает, что запись изменена удалённо (ServiceNow).
//
// Вызывается из webhook при получении уведомления от ServiceNow.
func (sm *SyncStateMachine) MarkRemoteChange(ctx context.Context, entityType SyncEntityType, remoteID string, changes map[string]interface{}) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Ищем запись по remote_id
	var record *SyncRecord
	for _, r := range sm.records {
		if r.RemoteID == remoteID && r.EntityType == entityType {
			record = r
			break
		}
	}

	if record == nil {
		// Новая запись из ServiceNow — нужно создать локально
		sm.logger.Info("sync: new remote entity, will create locally",
			"type", entityType, "remote_id", remoteID,
		)
		return nil
	}

	switch record.Status {
	case SyncStatusSynced:
		record.Status = SyncStatusPendingRemote
	case SyncStatusPendingLocal:
		// Изменения в обеих системах → conflict
		record.Status = SyncStatusConflict
		record.ConflictData = sm.captureConflictData(record.EntityType, record.EntityID)
	case SyncStatusConflict, SyncStatusFailed:
		return nil
	}

	record.RemoteVersion++
	record.UpdatedAt = time.Now()
	sm.logger.Info("sync: remote change marked",
		"type", entityType, "remote_id", remoteID,
		"status", record.Status, "version", record.RemoteVersion,
		"changes", len(changes),
	)

	return nil
}

// ── Sync Execution ────────────────────────────────────────────────────

// SyncPending отправляет ожидающие локальные изменения в ServiceNow
// и забирает ожидающие удалённые изменения из ServiceNow.
func (sm *SyncStateMachine) SyncPending(ctx context.Context) (synced, failed int, err error) {
	sm.mu.Lock()
	// Копируем список записей для синхронизации
	pendingLocal := make([]*SyncRecord, 0)
	pendingRemote := make([]*SyncRecord, 0)
	conflicts := make([]*SyncRecord, 0)

	for _, record := range sm.records {
		switch record.Status {
		case SyncStatusPendingLocal:
			pendingLocal = append(pendingLocal, record)
		case SyncStatusPendingRemote:
			pendingRemote = append(pendingRemote, record)
		case SyncStatusConflict:
			conflicts = append(conflicts, record)
		}
	}
	sm.mu.Unlock()

	// Этап 1: Отправляем локальные изменения в ServiceNow
	for _, record := range pendingLocal {
		if err := sm.pushToRemote(ctx, record); err != nil {
			sm.mu.Lock()
			record.Status = SyncStatusFailed
			record.LastError = err.Error()
			sm.mu.Unlock()
			failed++
			sm.logger.Error("sync: push to remote failed",
				"type", record.EntityType, "entity_id", record.EntityID, "error", err,
			)
		} else {
			sm.mu.Lock()
			record.Status = SyncStatusSynced
			now := time.Now()
			record.LastSyncAt = &now
			sm.mu.Unlock()
			synced++
		}
	}

	// Этап 2: Забираем удалённые изменения из ServiceNow
	for _, record := range pendingRemote {
		if err := sm.pullFromRemote(ctx, record); err != nil {
			sm.mu.Lock()
			record.Status = SyncStatusFailed
			record.LastError = err.Error()
			sm.mu.Unlock()
			failed++
			sm.logger.Error("sync: pull from remote failed",
				"type", record.EntityType, "remote_id", record.RemoteID, "error", err,
			)
		} else {
			sm.mu.Lock()
			record.Status = SyncStatusSynced
			now := time.Now()
			record.LastSyncAt = &now
			sm.mu.Unlock()
			synced++
		}
	}

	// Этап 3: Разрешаем конфликты
	for _, record := range conflicts {
		sm.resolveConflict(ctx, record)
	}

	return synced, failed, nil
}

// ── Push / Pull ───────────────────────────────────────────────────────

func (sm *SyncStateMachine) pushToRemote(ctx context.Context, record *SyncRecord) error {
	switch record.EntityType {
	case SyncEntityWorkOrder:
		// Получаем WO из локальной БД
		wo, err := sm.adapter.GetWorkOrder(ctx, record.EntityID)
		if err != nil {
			return fmt.Errorf("get local work order: %w", err)
		}

		// Маппинг статуса
		snStatus, ok := cctvToSNStatus[wo.Status]
		if !ok {
			snStatus = wo.Status
		}

		// Отправляем в ServiceNow
		body := toWorkOrderSNBody(wo)
		body["u_status"] = snStatus

		if record.RemoteID != "" {
			// UPDATE существующей записи
			return sm.client.Patch(ctx,
				"/api/now/table/"+TableWorkOrder+"/"+record.RemoteID,
				body, nil)
		}

		// CREATE новой записи
		var result map[string]interface{}
		if err := sm.client.Post(ctx,
			"/api/now/table/"+TableWorkOrder,
			body, &result); err != nil {
			return err
		}
		// Сохраняем remote_id из ответа ServiceNow
		if result != nil {
			if r, ok := result["result"].(map[string]interface{}); ok {
				if sysID, ok := r["sys_id"].(string); ok {
					sm.mu.Lock()
					record.RemoteID = sysID
					sm.mu.Unlock()
				}
			}
		}

	case SyncEntityAsset:
		body := map[string]interface{}{
			"name":        record.EntityID,
			"u_device_id": record.EntityID,
		}
		if record.RemoteID != "" {
			return sm.client.Patch(ctx,
				"/api/now/table/cmdb_ci/"+record.RemoteID,
				body, nil)
		}
		return sm.client.Post(ctx, "/api/now/table/cmdb_ci", body, nil)

	default:
		return fmt.Errorf("unsupported entity type for push: %s", record.EntityType)
	}

	return nil
}

func (sm *SyncStateMachine) pullFromRemote(ctx context.Context, record *SyncRecord) error {
	switch record.EntityType {
	case SyncEntityWorkOrder:
		// Получаем WO из ServiceNow
		wo, err := sm.adapter.GetWorkOrder(ctx, record.RemoteID)
		if err != nil {
			return fmt.Errorf("get remote work order: %w", err)
		}

		// Маппинг статуса обратно
		if cctvStatus, ok := snToCCTVStatus[wo.Status]; ok {
			wo.Status = cctvStatus
		}

		// Обновляем локальную запись
		if err := sm.adapter.UpdateWorkOrder(ctx, record.EntityID, map[string]interface{}{
			"status":  wo.Status,
			"u_notes": wo.Notes,
			"u_type":  wo.Type,
		}); err != nil {
			return fmt.Errorf("update local work order: %w", err)
		}

	default:
		return fmt.Errorf("unsupported entity type for pull: %s", record.EntityType)
	}

	return nil
}

// ── Conflict Resolution ──────────────────────────────────────────────

func (sm *SyncStateMachine) resolveConflict(ctx context.Context, record *SyncRecord) {
	sm.mu.Lock()
	strategy := sm.config.ConflictStrategy
	sm.mu.Unlock()

	switch strategy {
	case "local_wins":
		sm.logger.Info("sync: conflict resolved — local wins",
			"type", record.EntityType, "entity_id", record.EntityID,
		)
		record.Status = SyncStatusPendingLocal

	case "remote_wins":
		sm.logger.Info("sync: conflict resolved — remote wins",
			"type", record.EntityType, "entity_id", record.EntityID,
		)
		record.Status = SyncStatusPendingRemote

	case "manual":
		sm.logger.Warn("sync: conflict requires manual resolution",
			"type", record.EntityType, "entity_id", record.EntityID,
		)
		// Остаётся в статусе conflict
		// TODO: Отправить уведомление администратору

	default:
		sm.logger.Warn("sync: unknown conflict strategy, using remote_wins",
			"strategy", strategy,
		)
		record.Status = SyncStatusPendingRemote
	}
}

// ── Sync Stats ────────────────────────────────────────────────────────

// SyncStats — статистика синхронизации.
type SyncStats struct {
	TotalRecords int                    `json:"total_records"`
	ByStatus     map[SyncStatus]int     `json:"by_status"`
	ByEntity     map[SyncEntityType]int `json:"by_entity"`
	LastSyncAt   *time.Time             `json:"last_sync_at"`
	FailedCount  int                    `json:"failed_count"`
}

// GetStats возвращает статистику синхронизации.
func (sm *SyncStateMachine) GetStats() SyncStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := SyncStats{
		TotalRecords: len(sm.records),
		ByStatus:     make(map[SyncStatus]int),
		ByEntity:     make(map[SyncEntityType]int),
	}

	for _, record := range sm.records {
		stats.ByStatus[record.Status]++
		stats.ByEntity[record.EntityType]++
		if record.Status == SyncStatusFailed {
			stats.FailedCount++
		}
		if stats.LastSyncAt == nil || (record.LastSyncAt != nil && record.LastSyncAt.After(*stats.LastSyncAt)) {
			stats.LastSyncAt = record.LastSyncAt
		}
	}

	return stats
}

// ── Sync Worker ───────────────────────────────────────────────────────

// SyncWorker — периодический worker для синхронизации.
type SyncWorker struct {
	sm     *SyncStateMachine
	logger *slog.Logger
	ticker *time.Ticker
	stopCh chan struct{}
}

// NewSyncWorker создаёт фоновый worker.
func NewSyncWorker(sm *SyncStateMachine, logger *slog.Logger) *SyncWorker {
	return &SyncWorker{
		sm:     sm,
		logger: logger.With("component", "sn-sync-worker"),
		stopCh: make(chan struct{}),
	}
}

// Start запускает циклическую синхронизацию.
func (w *SyncWorker) Start(ctx context.Context) {
	w.ticker = time.NewTicker(w.sm.config.SyncInterval)
	w.logger.Info("sync worker started", "interval", w.sm.config.SyncInterval)

	for {
		select {
		case <-w.ticker.C:
			synced, failed, err := w.sm.SyncPending(ctx)
			if err != nil {
				w.logger.Error("sync worker error", "error", err)
			}
			if synced > 0 || failed > 0 {
				w.logger.Info("sync cycle complete",
					"synced", synced, "failed", failed,
				)
			}
		case <-w.stopCh:
			w.ticker.Stop()
			w.logger.Info("sync worker stopped")
			return
		}
	}
}

// Stop останавливает worker.
func (w *SyncWorker) Stop() {
	close(w.stopCh)
}

// ── Helpers ───────────────────────────────────────────────────────────

func (sm *SyncStateMachine) recordKey(entityType SyncEntityType, entityID string) string {
	return string(entityType) + ":" + entityID
}

func (sm *SyncStateMachine) captureConflictData(entityType SyncEntityType, entityID string) json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"type":        entityType,
		"entity_id":   entityID,
		"captured_at": time.Now().UTC().Format(time.RFC3339),
	})
	return data
}
