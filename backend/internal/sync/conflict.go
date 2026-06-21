package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
)

// ConflictResolver реализует правила разрешения конфликтов:
// - external-wins для статуса (внешняя CMMS — источник истины для статуса)
// - local-wins для метаданных (локальная БД — источник истины для описания, приоритета)
// - auto-reopen: если локальный тикет закрыт, а внешний переоткрыт — переоткрываем локальный
type ConflictResolver struct {
	logger *slog.Logger
}

// NewConflictResolver создаёт новый резолвер конфликтов.
func NewConflictResolver(logger *slog.Logger) *ConflictResolver {
	if logger == nil {
		logger = slog.Default()
	}
	return &ConflictResolver{logger: logger}
}

// ResolveWorkOrder разрешает конфликт между локальным WorkOrder и внешним статусом.
// Правила (Epic 3.4.3):
//  1. external-wins для статуса — статус всегда берётся из внешней CMMS
//  2. local-wins для метаданных — локальные поля (notes, checklist, photos) не перезаписываются
//  3. auto-reopen: если локальный статус completed/cancelled, а внешний — open/in_progress,
//     переоткрываем локальный тикет
//  4. Конфликт логируется в audit_log
func (r *ConflictResolver) ResolveWorkOrder(ctx context.Context, localWO *models.WorkOrder, extStatus *db.ExternalWorkOrderStatus, database *db.DB) *ConflictResolution {
	resolution := &ConflictResolution{
		WorkOrderID:      localWO.ID,
		ConflictDetected: false,
		Resolution:       "none",
		AppliedChanges:   make(map[string]interface{}),
	}

	// Нормализуем статусы
	extStatusNorm := normalizeStatus(extStatus.Status)
	localStatusNorm := normalizeStatus(localWO.Status)

	// Если статусы совпадают — конфликта нет
	if extStatusNorm == localStatusNorm {
		return resolution
	}

	// Конфликт обнаружен
	resolution.ConflictDetected = true

	// Правило 3: auto-reopen — локальный закрыт, внешний открыт
	if isTerminalStatus(localStatusNorm) && !isTerminalStatus(extStatusNorm) {
		r.logger.Info("conflict: auto-reopen closed ticket",
			"work_order_id", localWO.ID,
			"local_status", localWO.Status,
			"external_status", extStatus.Status,
			"source", extStatus.Source,
		)

		// Переоткрываем локальный WorkOrder
		if err := r.reopenWorkOrder(ctx, database, localWO.ID, extStatus); err != nil {
			r.logger.Error("conflict: failed to reopen work order",
				"work_order_id", localWO.ID,
				"error", err,
			)
			resolution.Resolution = "error"
			resolution.ConflictLogEntry = fmt.Sprintf(
				"Auto-reopen failed: local=%s, external=%s (source: %s), error: %v",
				localWO.Status, extStatus.Status, extStatus.Source, err,
			)
			return resolution
		}

		resolution.Resolution = "external_wins_reopen"
		resolution.AppliedChanges["status"] = extStatusNorm
		resolution.AppliedChanges["action"] = "reopened"
		resolution.ConflictLogEntry = fmt.Sprintf(
			"Auto-reopen: local was %s, external is %s (source: %s). Ticket reopened.",
			localWO.Status, extStatus.Status, extStatus.Source,
		)

		// Логируем в audit_log
		r.logConflict(ctx, database, localWO.ID, extStatus.Source, "reopen", localWO.Status, extStatus.Status)
		return resolution
	}

	// Правило 1: external-wins для статуса
	if extStatusNorm != localStatusNorm {
		r.logger.Info("conflict: external-wins for status",
			"work_order_id", localWO.ID,
			"local_status", localWO.Status,
			"external_status", extStatus.Status,
			"source", extStatus.Source,
		)

		// Обновляем статус в локальной БД
		if err := r.updateWorkOrderStatus(ctx, database, localWO.ID, extStatus); err != nil {
			r.logger.Error("conflict: failed to update work order status",
				"work_order_id", localWO.ID,
				"error", err,
			)
			resolution.Resolution = "error"
			resolution.ConflictLogEntry = fmt.Sprintf(
				"External-wins status update failed: %s -> %s, error: %v",
				localWO.Status, extStatus.Status, err,
			)
			return resolution
		}

		resolution.Resolution = "external_wins"
		resolution.AppliedChanges["status"] = extStatusNorm
		resolution.AppliedChanges["old_status"] = localWO.Status
		resolution.ConflictLogEntry = fmt.Sprintf(
			"External-wins: status %s -> %s (source: %s)",
			localWO.Status, extStatus.Status, extStatus.Source,
		)

		// Логируем в audit_log
		r.logConflict(ctx, database, localWO.ID, extStatus.Source, "status_update", localWO.Status, extStatus.Status)
	}

	return resolution
}

// reopenWorkOrder переоткрывает закрытый/отменённый WorkOrder.
func (r *ConflictResolver) reopenWorkOrder(ctx context.Context, database *db.DB, workOrderID string, extStatus *db.ExternalWorkOrderStatus) error {
	now := time.Now()
	_, err := database.Pool.Exec(ctx, `
		UPDATE work_orders
		SET status = $1,
		    completed_at = NULL,
		    started_at = CASE WHEN status = 'completed' THEN $2 ELSE started_at END,
		    notes = COALESCE(notes, '') || E'\n[Auto-reopen] ' || $3,
		    updated_at = $4
		WHERE id = $5
	`, extStatus.Status, &now, fmt.Sprintf("Reopened via %s sync: external status changed to '%s'", extStatus.Source, extStatus.Status), now, workOrderID)
	return err
}

// updateWorkOrderStatus обновляет только статус WorkOrder (external-wins).
func (r *ConflictResolver) updateWorkOrderStatus(ctx context.Context, database *db.DB, workOrderID string, extStatus *db.ExternalWorkOrderStatus) error {
	now := time.Now()
	_, err := database.Pool.Exec(ctx, `
		UPDATE work_orders
		SET status = $1,
		    updated_at = $2
		WHERE id = $3
	`, extStatus.Status, now, workOrderID)
	return err
}

// logConflict записывает разрешение конфликта в audit_log.
func (r *ConflictResolver) logConflict(ctx context.Context, database *db.DB, workOrderID, source, action, oldStatus, newStatus string) {
	oldVal, _ := json.Marshal(map[string]string{"status": oldStatus})
	newVal, _ := json.Marshal(map[string]string{"status": newStatus, "source": source, "action": action})

	_, err := database.Pool.Exec(ctx, `
		INSERT INTO audit_log (user_id, action, entity_type, entity_id, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, "system", "sync_conflict_resolved", "work_order", workOrderID, oldVal, newVal)
	if err != nil {
		r.logger.Error("conflict: failed to write audit log", "error", err)
	}
}

// normalizeStatus приводит статус к одному из четырёх стандартных значений.
func normalizeStatus(status string) string {
	switch status {
	case "open", "new", "pending", "submitted", "queued":
		return "open"
	case "in_progress", "in progress", "progress", "active", "assigned", "working":
		return "in_progress"
	case "completed", "complete", "done", "resolved", "closed", "finished":
		return "completed"
	case "cancelled", "canceled", "rejected", "void":
		return "cancelled"
	default:
		return status
	}
}

// isTerminalStatus возвращает true, если статус завершающий (completed или cancelled).
func isTerminalStatus(status string) bool {
	return status == "completed" || status == "cancelled"
}
