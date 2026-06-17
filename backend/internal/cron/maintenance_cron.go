package cron

import (
	"context"
	"log/slog"
	"time"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
)

type MaintenanceCron struct {
	db     *db.DB
	logger *slog.Logger
}

func NewMaintenanceCron(database *db.DB, logger *slog.Logger) *MaintenanceCron {
	return &MaintenanceCron{db: database, logger: logger}
}

// Run проверяет due schedules и создаёт work orders
func (c *MaintenanceCron) Run(ctx context.Context) {
	c.logger.Info("Starting maintenance cron job")

	// 1. Получить все due schedules
	schedules, err := c.db.GetDueSchedules()
	if err != nil {
		c.logger.Error("Failed to get due schedules", "error", err)
		return
	}

	createdCount := 0
	for _, schedule := range schedules {
		// 2. Создать work order
		wo := &models.WorkOrder{
			ScheduleID: &schedule.ID,
			DeviceID:   schedule.DeviceID,
			Type:       "preventive",
			Status:     "open",
			Priority:   schedule.Priority,
			AssignedTo: schedule.AssignedTo,
			Checklist:  schedule.Checklist,
			Notes:      "Auto-created from maintenance schedule",
		}

		// Устанавливаем SLA deadline
		wo.SLADeadline = c.calculateSLADeadline(schedule.Priority)

		if err := c.db.CreateWorkOrder(wo); err != nil {
			c.logger.Error("Failed to create work order", "schedule_id", schedule.ID, "error", err)
			continue
		}

		// 3. Обновить next_due
		if err := c.db.CompleteMaintenanceSchedule(schedule.ID); err != nil {
			c.logger.Error("Failed to update schedule", "id", schedule.ID, "error", err)
		}

		createdCount++
		c.logger.Info("Created work order from schedule",
			"schedule_id", schedule.ID,
			"work_order_id", wo.ID,
			"device_id", schedule.DeviceID)
	}

	c.logger.Info("Maintenance cron job completed", "created_count", createdCount)
}

func (c *MaintenanceCron) calculateSLADeadline(priority string) *time.Time {
	sla, err := c.db.GetSLAConfig(priority)
	if err != nil {
		c.logger.Warn("Failed to get SLA config", "priority", priority, "error", err)
		return nil
	}

	deadline := time.Now().Add(time.Duration(sla.ResolutionTimeMinutes) * time.Minute)
	return &deadline
}
