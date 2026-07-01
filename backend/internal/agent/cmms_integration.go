// Package agent — CMMS integration: auto-ticket creation/closure, audit trail.
package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/models"
)

// CMMSIntegrator — интеграция Self-Healing Agent с CMMS.
type CMMSIntegrator struct {
	adapter cmms.CMMSAdapter
	logger  *slog.Logger

	// Маппинг deviceID → ticketID для отслеживания открытых тикетов
	ticketMap map[string]string // deviceID → workOrderID
	mu        sync.RWMutex      // защита ticketMap от concurrent map read/write (P0-CR-03)
}

// NewCMMSIntegrator создаёт новый интегратор.
func NewCMMSIntegrator(adapter cmms.CMMSAdapter, logger *slog.Logger) *CMMSIntegrator {
	if logger == nil {
		logger = slog.Default()
	}
	return &CMMSIntegrator{
		adapter:   adapter,
		logger:    logger,
		ticketMap: make(map[string]string),
	}
}

// AutoCreateTicket создаёт тикет при тревоге с таймаутом 30 секунд.
func (ci *CMMSIntegrator) AutoCreateTicket(ctx context.Context, deviceID, deviceName, alarmType, severity, description string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	priority := mapSeverityToPriority(severity)

	wo := &models.WorkOrder{
		DeviceID:  deviceID,
		Type:      "corrective",
		Priority:  priority,
		Status:    "open",
		Notes:     fmt.Sprintf("[Self-Healing][%s] %s severity alarm on %s (%s). Details: %s", alarmType, severity, deviceName, deviceID, description),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := ci.adapter.CreateWorkOrder(ctx, wo); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			ci.logger.Warn("auto-create ticket timed out", "device", deviceID, "timeout", "30s")
		} else {
			ci.logger.Error("auto-create ticket failed", "device", deviceID, "error", err)
		}
		return "", fmt.Errorf("create ticket: %w", err)
	}

	ci.mu.Lock()
	ci.ticketMap[deviceID] = wo.ID
	ci.mu.Unlock()

	ci.logger.Info("auto-ticket created",
		"device_id", deviceID,
		"ticket_id", wo.ID,
		"priority", priority,
		"type", alarmType,
	)

	return wo.ID, nil
}

// AutoCloseTicket закрывает тикет после успешного self-healing с таймаутом 30 секунд.
func (ci *CMMSIntegrator) AutoCloseTicket(ctx context.Context, deviceID, ticketID, resolution string) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if ticketID == "" {
		// Пробуем найти по deviceID
		var ok bool
		ci.mu.RLock()
		ticketID, ok = ci.ticketMap[deviceID]
		ci.mu.RUnlock()
		if !ok {
			ci.logger.Warn("no ticket found for device", "device_id", deviceID)
			return nil
		}
	}

	updates := map[string]interface{}{
		"status":    "completed",
		"completed": true,
		"notes":     fmt.Sprintf("[Self-Healing] %s\nAuto-closed by agent after successful remediation.", resolution),
	}

	updates["completed_at"] = time.Now()
	if err := ci.adapter.UpdateWorkOrder(ctx, ticketID, updates); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			ci.logger.Warn("auto-close ticket timed out", "device_id", deviceID, "ticket_id", ticketID, "timeout", "30s")
		} else {
			ci.logger.Error("auto-close ticket failed", "ticket_id", ticketID, "error", err)
		}
		return fmt.Errorf("close ticket %s: %w", ticketID, err)
	}

	ci.mu.Lock()
	delete(ci.ticketMap, deviceID)
	ci.mu.Unlock()

	ci.logger.Info("auto-ticket closed", "device_id", deviceID, "ticket_id", ticketID)
	return nil
}

// AddAuditNote добавляет audit-заметку к существующему тикету с таймаутом 15 секунд.
func (ci *CMMSIntegrator) AddAuditNote(ctx context.Context, ticketID, action, details string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	note := fmt.Sprintf("[%s] %s: %s", time.Now().Format(time.RFC3339), action, details)
	updates := map[string]interface{}{
		"notes": note,
	}
	if err := ci.adapter.UpdateWorkOrder(ctx, ticketID, updates); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			ci.logger.Warn("add audit note timed out", "ticket_id", ticketID, "timeout", "15s")
		} else {
			ci.logger.Error("add audit note failed", "ticket_id", ticketID, "error", err)
		}
		return fmt.Errorf("add audit note to %s: %w", ticketID, err)
	}
	return nil
}

// GetTicketForDevice возвращает ID тикета для устройства.
func (ci *CMMSIntegrator) GetTicketForDevice(deviceID string) (string, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	id, ok := ci.ticketMap[deviceID]
	return id, ok
}

// ── Helpers ────────────────────────────────────────────────────────

func mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "P1"
	case "high":
		return "P2"
	case "medium":
		return "P3"
	default:
		return "P4"
	}
}
