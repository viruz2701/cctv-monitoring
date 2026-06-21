// Package sync реализует bi-directional ITSM синхронизацию между CMMS и локальной БД.
// Включает: приём webhook-уведомлений, State Machine с cron-синхронизацией,
// и систему разрешения конфликтов (external-wins для статуса, local-wins для метаданных).
package sync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"gb-telemetry-collector/internal/db"
)

// SyncEngine — центральный компонент bi-directional синхронизации.
// Обрабатывает входящие webhook-уведомления и выполняет периодическую cron-синхронизацию.
type SyncEngine struct {
	db           *db.DB
	logger       *slog.Logger
	conflictRes  *ConflictResolver
	mu           sync.RWMutex
	syncInterval time.Duration
	stopCh       chan struct{}

	// Webhook secrets
	snSecret   string
	jiraSecret string
	toirSecret string
}

// NewSyncEngine создаёт новый движок синхронизации.
func NewSyncEngine(database *db.DB, logger *slog.Logger, snSecret, jiraSecret, toirSecret string, syncInterval time.Duration) *SyncEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncEngine{
		db:           database,
		logger:       logger,
		conflictRes:  NewConflictResolver(logger),
		syncInterval: syncInterval,
		stopCh:       make(chan struct{}),
		snSecret:     snSecret,
		jiraSecret:   jiraSecret,
		toirSecret:   toirSecret,
	}
}

// Start запускает периодическую cron-синхронизацию.
func (e *SyncEngine) Start(ctx context.Context) {
	e.logger.Info("sync engine: starting periodic sync", "interval", e.syncInterval)
	go func() {
		// Первый запуск через 30 секунд после старта
		select {
		case <-time.After(30 * time.Second):
			e.RunSync(ctx)
		case <-e.stopCh:
			return
		}

		ticker := time.NewTicker(e.syncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				e.RunSync(ctx)
			case <-e.stopCh:
				return
			}
		}
	}()
}

// Stop останавливает периодическую синхронизацию.
func (e *SyncEngine) Stop() {
	close(e.stopCh)
}

// RunSync выполняет полный цикл синхронизации:
// 1. Получает все WorkOrders из локальной БД со статусом, отличным от completed/cancelled
// 2. Проверяет external_changed_at — если внешняя система обновила запись позже локальной
// 3. Разрешает конфликты через ConflictResolver
// 4. Обновляет локальную БД
func (e *SyncEngine) RunSync(ctx context.Context) {
	e.logger.Info("sync engine: starting sync cycle")

	// Получаем все активные WorkOrders
	allOrders, err := e.db.GetWorkOrders(map[string]interface{}{})
	if err != nil {
		e.logger.Error("sync engine: failed to get work orders", "error", err)
		return
	}

	syncedCount := 0
	conflictCount := 0

	for _, wo := range allOrders {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Проверяем, есть ли внешние изменения
		extStatusDB, err := e.db.GetExternalWorkOrderStatus(ctx, wo.ID)
		if err != nil {
			continue
		}
		if extStatusDB == nil {
			continue
		}

		// Проверяем, была ли внешняя система обновлена позже локальной
		if extStatusDB.ExternalChangedAt.After(wo.UpdatedAt) {
			resolved := e.conflictRes.ResolveWorkOrder(ctx, &wo, extStatusDB, e.db)
			if resolved.ConflictDetected {
				conflictCount++
				e.logger.Warn("sync engine: conflict resolved",
					"work_order_id", wo.ID,
					"resolution", resolved.Resolution,
					"external_status", extStatusDB.Status,
					"local_status", wo.Status,
				)
			}
			syncedCount++
		}
	}

	e.logger.Info("sync engine: sync cycle completed",
		"synced", syncedCount,
		"conflicts", conflictCount,
		"total_active", len(allOrders),
	)
}

// ── Webhook Handlers ──────────────────────────────────────────────────

// ServiceNowWebhookHandler возвращает http.Handler для ServiceNow webhook.
func (e *SyncEngine) ServiceNowWebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			e.logger.Error("sync: sn webhook read body", "error", err)
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		if !e.verifyHMAC(e.snSecret, r.Header.Get("X-SN-Signature"), body) {
			e.logger.Warn("sync: sn webhook invalid signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		var event struct {
			Table    string                 `json:"table"`
			RecordID string                 `json:"record_id"`
			Action   string                 `json:"action"`
			Changes  map[string]interface{} `json:"changes"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			e.logger.Error("sync: sn webhook unmarshal", "error", err)
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		e.logger.Info("sync: sn webhook received",
			"table", event.Table,
			"record_id", event.RecordID,
			"action", event.Action,
		)

		if event.Table == "u_cctv_work_order" || event.Table == "x_gb_cctv_work_order" {
			if err := e.handleExternalWorkOrderUpdate(r.Context(), "servicenow", event.RecordID, event.Changes); err != nil {
				e.logger.Error("sync: sn handle work order update", "error", err)
				http.Error(w, "handler error", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

// JiraWebhookHandler возвращает http.Handler для Jira webhook.
func (e *SyncEngine) JiraWebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			e.logger.Error("sync: jira webhook read body", "error", err)
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		if !e.verifyHMAC(e.jiraSecret, r.Header.Get("X-Hub-Signature-256"), body) {
			e.logger.Warn("sync: jira webhook invalid signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		var event struct {
			WebhookEvent string `json:"webhookEvent"`
			Issue        struct {
				Key    string `json:"key"`
				Fields struct {
					Summary     string `json:"summary"`
					Description string `json:"description"`
					IssueType   struct {
						Name string `json:"name"`
					} `json:"issuetype"`
					Status struct {
						Name string `json:"name"`
					} `json:"status"`
					Priority struct {
						Name string `json:"name"`
					} `json:"priority"`
				} `json:"fields"`
			} `json:"issue"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			e.logger.Error("sync: jira webhook unmarshal", "error", err)
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		e.logger.Info("sync: jira webhook received",
			"issue_key", event.Issue.Key,
			"event_type", event.WebhookEvent,
		)

		changes := map[string]interface{}{
			"status":      event.Issue.Fields.Status.Name,
			"priority":    event.Issue.Fields.Priority.Name,
			"summary":     event.Issue.Fields.Summary,
			"description": event.Issue.Fields.Description,
		}

		if err := e.handleExternalWorkOrderUpdate(r.Context(), "jira", event.Issue.Key, changes); err != nil {
			e.logger.Error("sync: jira handle work order update", "error", err)
			http.Error(w, "handler error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

// TOIRWebhookHandler возвращает http.Handler для 1С:ТОИР webhook.
func (e *SyncEngine) TOIRWebhookHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			e.logger.Error("sync: toir webhook read body", "error", err)
			http.Error(w, "read error", http.StatusBadRequest)
			return
		}

		if !e.verifyHMAC(e.toirSecret, r.Header.Get("X-TOIR-Signature"), body) {
			e.logger.Warn("sync: toir webhook invalid signature")
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		var event struct {
			Entity   string                 `json:"entity"`
			RecordID string                 `json:"record_id"`
			Action   string                 `json:"action"`
			Changes  map[string]interface{} `json:"changes"`
		}
		if err := json.Unmarshal(body, &event); err != nil {
			e.logger.Error("sync: toir webhook unmarshal", "error", err)
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		e.logger.Info("sync: toir webhook received",
			"entity", event.Entity,
			"record_id", event.RecordID,
			"action", event.Action,
		)

		if event.Entity == "work_order" {
			if err := e.handleExternalWorkOrderUpdate(r.Context(), "toir", event.RecordID, event.Changes); err != nil {
				e.logger.Error("sync: toir handle work order update", "error", err)
				http.Error(w, "handler error", http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
}

// ── Internal Methods ──────────────────────────────────────────────────

// handleExternalWorkOrderUpdate обрабатывает обновление WorkOrder из внешней CMMS.
// Сохраняет внешний статус в таблицу external_work_order_status и разрешает конфликты.
func (e *SyncEngine) handleExternalWorkOrderUpdate(ctx context.Context, source, externalID string, changes map[string]interface{}) error {
	extStatus := &db.ExternalWorkOrderStatus{
		ExternalID:        externalID,
		Source:            source,
		ExternalChangedAt: time.Now(),
		Changes:           changes,
	}

	if status, ok := changes["status"].(string); ok {
		extStatus.Status = status
	}
	if priority, ok := changes["priority"].(string); ok {
		extStatus.Priority = priority
	}
	if summary, ok := changes["summary"].(string); ok {
		extStatus.Summary = summary
	}

	// Сохраняем внешний статус
	if err := e.db.UpsertExternalWorkOrderStatus(ctx, extStatus); err != nil {
		return fmt.Errorf("upsert external status: %w", err)
	}

	// Пытаемся найти локальный WorkOrder, связанный с этим внешним ID
	localWO, err := e.db.GetWorkOrderByExternalID(ctx, source, externalID)
	if err != nil || localWO == nil {
		// WorkOrder ещё не синхронизирован — создаём при необходимости
		e.logger.Info("sync: external work order not yet mapped locally",
			"source", source, "external_id", externalID)
		return nil
	}

	// Разрешаем конфликт
	resolved := e.conflictRes.ResolveWorkOrder(ctx, localWO, extStatus, e.db)
	if resolved.ConflictDetected {
		e.logger.Warn("sync: conflict detected and resolved",
			"work_order_id", localWO.ID,
			"source", source,
			"resolution", resolved.Resolution,
		)
	}

	return nil
}

// verifyHMAC проверяет HMAC-SHA256 подпись с поддержкой префикса "sha256=".
func (e *SyncEngine) verifyHMAC(secret, sigHeader string, body []byte) bool {
	if secret == "" {
		return true
	}
	if sigHeader == "" {
		return false
	}

	sig := sigHeader
	if len(sigHeader) > 7 && sigHeader[:7] == "sha256=" {
		sig = sigHeader[7:]
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
}

// ConflictResolution — результат разрешения конфликта.
type ConflictResolution struct {
	WorkOrderID      string                 `json:"work_order_id"`
	ConflictDetected bool                   `json:"conflict_detected"`
	Resolution       string                 `json:"resolution"` // external_wins, local_wins, merged
	AppliedChanges   map[string]interface{} `json:"applied_changes"`
	ConflictLogEntry string                 `json:"conflict_log_entry"`
}
