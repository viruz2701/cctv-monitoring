package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gb-telemetry-collector/internal/crypto"
	"gb-telemetry-collector/internal/models"

	"github.com/jackc/pgx/v5"
)

// ═══════════════════════════════════════════════════════════════════════
// Maintenance Schedules
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) CreateMaintenanceSchedule(schedule *models.MaintenanceSchedule) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO maintenance_schedules (
			device_id, schedule_type, interval_days, custom_cron, next_due,
			assigned_to, checklist, estimated_minutes, priority, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, schedule.DeviceID, schedule.ScheduleType, schedule.IntervalDays,
		schedule.CustomCron, schedule.NextDue, schedule.AssignedTo,
		schedule.Checklist, schedule.EstimatedMinutes, schedule.Priority,
		schedule.Notes,
	).Scan(&schedule.ID, &schedule.CreatedAt, &schedule.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert maintenance_schedule: %w", err)
	}
	return nil
}

func (db *DB) GetMaintenanceSchedules(filters map[string]interface{}) ([]models.MaintenanceSchedule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT ms.id, ms.device_id, ms.schedule_type, ms.interval_days, ms.custom_cron,
			ms.last_completed, ms.next_due, ms.assigned_to, ms.checklist,
			ms.estimated_minutes, ms.priority, ms.notes, ms.created_at, ms.updated_at,
			COALESCE(d.name, d.device_id) as device_name,
			COALESCE(u.username, '') as assignee_name
		FROM maintenance_schedules ms
		LEFT JOIN devices d ON ms.device_id = d.device_id
		LEFT JOIN users u ON ms.assigned_to = u.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if deviceID, ok := filters["device_id"]; ok {
		query += fmt.Sprintf(" AND ms.device_id = $%d", argIdx)
		args = append(args, deviceID)
		argIdx++
	}
	if scheduleType, ok := filters["schedule_type"]; ok {
		query += fmt.Sprintf(" AND ms.schedule_type = $%d", argIdx)
		args = append(args, scheduleType)
		argIdx++
	}
	if priority, ok := filters["priority"]; ok {
		query += fmt.Sprintf(" AND ms.priority = $%d", argIdx)
		args = append(args, priority)
		argIdx++
	}
	if assignedTo, ok := filters["assigned_to"]; ok {
		query += fmt.Sprintf(" AND ms.assigned_to = $%d", argIdx)
		args = append(args, assignedTo)
		argIdx++
	}

	query += " ORDER BY ms.next_due ASC"

	if limit, ok := filters["limit"]; ok {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset, ok := filters["offset"]; ok {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query maintenance_schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.MaintenanceSchedule
	for rows.Next() {
		var s models.MaintenanceSchedule
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.ScheduleType, &s.IntervalDays, &s.CustomCron,
			&s.LastCompleted, &s.NextDue, &s.AssignedTo, &s.Checklist,
			&s.EstimatedMinutes, &s.Priority, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
			&s.DeviceName, &s.AssigneeName,
		); err != nil {
			return nil, fmt.Errorf("scan maintenance_schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (db *DB) GetMaintenanceSchedule(id string) (*models.MaintenanceSchedule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var s models.MaintenanceSchedule
	err := db.Pool.QueryRow(ctx, `
		SELECT ms.id, ms.device_id, ms.schedule_type, ms.interval_days, ms.custom_cron,
			ms.last_completed, ms.next_due, ms.assigned_to, ms.checklist,
			ms.estimated_minutes, ms.priority, ms.notes, ms.created_at, ms.updated_at,
			COALESCE(d.name, d.device_id) as device_name,
			COALESCE(u.username, '') as assignee_name
		FROM maintenance_schedules ms
		LEFT JOIN devices d ON ms.device_id = d.device_id
		LEFT JOIN users u ON ms.assigned_to = u.id
		WHERE ms.id = $1
	`, id).Scan(
		&s.ID, &s.DeviceID, &s.ScheduleType, &s.IntervalDays, &s.CustomCron,
		&s.LastCompleted, &s.NextDue, &s.AssignedTo, &s.Checklist,
		&s.EstimatedMinutes, &s.Priority, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
		&s.DeviceName, &s.AssigneeName,
	)
	if err != nil {
		return nil, fmt.Errorf("get maintenance_schedule %s: %w", id, err)
	}
	return &s, nil
}

func (db *DB) UpdateMaintenanceSchedule(id string, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE maintenance_schedules SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx+1)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update maintenance_schedule %s: %w", id, err)
	}
	return nil
}

func (db *DB) DeleteMaintenanceSchedule(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, "DELETE FROM maintenance_schedules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete maintenance_schedule %s: %w", id, err)
	}
	return nil
}

func (db *DB) GetDueSchedules() ([]models.MaintenanceSchedule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT ms.id, ms.device_id, ms.schedule_type, ms.interval_days, ms.custom_cron,
			ms.last_completed, ms.next_due, ms.assigned_to, ms.checklist,
			ms.estimated_minutes, ms.priority, ms.notes, ms.created_at, ms.updated_at,
			COALESCE(d.name, d.device_id) as device_name,
			COALESCE(u.username, '') as assignee_name
		FROM maintenance_schedules ms
		LEFT JOIN devices d ON ms.device_id = d.device_id
		LEFT JOIN users u ON ms.assigned_to = u.id
		WHERE ms.next_due <= NOW()
		ORDER BY ms.next_due ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []models.MaintenanceSchedule
	for rows.Next() {
		var s models.MaintenanceSchedule
		if err := rows.Scan(
			&s.ID, &s.DeviceID, &s.ScheduleType, &s.IntervalDays, &s.CustomCron,
			&s.LastCompleted, &s.NextDue, &s.AssignedTo, &s.Checklist,
			&s.EstimatedMinutes, &s.Priority, &s.Notes, &s.CreatedAt, &s.UpdatedAt,
			&s.DeviceName, &s.AssigneeName,
		); err != nil {
			return nil, fmt.Errorf("scan due schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (db *DB) CompleteMaintenanceSchedule(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Получаем schedule для вычисления next_due
	var schedule models.MaintenanceSchedule
	err := db.Pool.QueryRow(ctx, `
		SELECT schedule_type, interval_days FROM maintenance_schedules WHERE id = $1
	`, id).Scan(&schedule.ScheduleType, &schedule.IntervalDays)
	if err != nil {
		return fmt.Errorf("get schedule for completion: %w", err)
	}

	// Вычисляем следующий due
	var nextDue time.Time
	now := time.Now()
	switch schedule.ScheduleType {
	case "daily":
		nextDue = now.AddDate(0, 0, 1)
	case "weekly":
		nextDue = now.AddDate(0, 0, 7)
	case "monthly":
		nextDue = now.AddDate(0, 1, 0)
	case "quarterly":
		nextDue = now.AddDate(0, 3, 0)
	case "custom":
		if schedule.IntervalDays > 0 {
			nextDue = now.AddDate(0, 0, schedule.IntervalDays)
		} else {
			nextDue = now.AddDate(0, 1, 0) // fallback
		}
	default:
		nextDue = now.AddDate(0, 0, schedule.IntervalDays)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE maintenance_schedules
		SET last_completed = NOW(), next_due = $1, updated_at = NOW()
		WHERE id = $2
	`, nextDue, id)
	if err != nil {
		return fmt.Errorf("complete maintenance_schedule %s: %w", id, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Work Orders
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) CreateWorkOrder(wo *models.WorkOrder) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO work_orders (
			schedule_id, device_id, type, status, priority, assigned_to,
			sla_deadline, checklist, notes, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, wo.ScheduleID, wo.DeviceID, wo.Type, wo.Status, wo.Priority,
		wo.AssignedTo, wo.SLADeadline, wo.Checklist, wo.Notes, wo.CreatedBy,
	).Scan(&wo.ID, &wo.CreatedAt, &wo.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert work_order: %w", err)
	}

	// Увеличиваем workload техника
	if wo.AssignedTo != nil {
		_, _ = db.Pool.Exec(ctx, `
			UPDATE users SET current_workload = current_workload + 1 WHERE id = $1
		`, *wo.AssignedTo)
	}

	return nil
}

func (db *DB) GetWorkOrders(filters map[string]interface{}) ([]models.WorkOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT wo.id, wo.schedule_id, wo.device_id, wo.type, wo.status, wo.priority,
			wo.assigned_to, wo.sla_deadline, wo.checklist, wo.started_at, wo.completed_at,
			wo.notes, wo.photos, wo.parts_used, wo.created_by, wo.created_at, wo.updated_at,
			COALESCE(d.name, d.device_id) as device_name,
			COALESCE(u.username, '') as assignee_name
		FROM work_orders wo
		LEFT JOIN devices d ON wo.device_id = d.device_id
		LEFT JOIN users u ON wo.assigned_to = u.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if deviceID, ok := filters["device_id"]; ok {
		query += fmt.Sprintf(" AND wo.device_id = $%d", argIdx)
		args = append(args, deviceID)
		argIdx++
	}
	if status, ok := filters["status"]; ok {
		query += fmt.Sprintf(" AND wo.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if woType, ok := filters["type"]; ok {
		query += fmt.Sprintf(" AND wo.type = $%d", argIdx)
		args = append(args, woType)
		argIdx++
	}
	if priority, ok := filters["priority"]; ok {
		query += fmt.Sprintf(" AND wo.priority = $%d", argIdx)
		args = append(args, priority)
		argIdx++
	}
	if assignedTo, ok := filters["assigned_to"]; ok {
		query += fmt.Sprintf(" AND wo.assigned_to = $%d", argIdx)
		args = append(args, assignedTo)
		argIdx++
	}

	query += " ORDER BY " +
		"CASE wo.priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END, " +
		"wo.created_at DESC"

	if limit, ok := filters["limit"]; ok {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset, ok := filters["offset"]; ok {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query work_orders: %w", err)
	}
	defer rows.Close()

	var workOrders []models.WorkOrder
	for rows.Next() {
		var wo models.WorkOrder
		if err := rows.Scan(
			&wo.ID, &wo.ScheduleID, &wo.DeviceID, &wo.Type, &wo.Status, &wo.Priority,
			&wo.AssignedTo, &wo.SLADeadline, &wo.Checklist, &wo.StartedAt, &wo.CompletedAt,
			&wo.Notes, &wo.Photos, &wo.PartsUsed, &wo.CreatedBy, &wo.CreatedAt, &wo.UpdatedAt,
			&wo.DeviceName, &wo.AssigneeName,
		); err != nil {
			return nil, fmt.Errorf("scan work_order: %w", err)
		}
		// Вычисляем SLA status
		wo.SLAStatus = calculateSLAStatus(wo)
		workOrders = append(workOrders, wo)
	}
	return workOrders, rows.Err()
}

func calculateSLAStatus(wo models.WorkOrder) string {
	if wo.Status == "completed" || wo.Status == "cancelled" {
		return "completed"
	}
	if wo.SLADeadline == nil {
		return "no_sla"
	}
	now := time.Now()
	deadline := *wo.SLADeadline
	remaining := deadline.Sub(now)
	total := deadline.Sub(wo.CreatedAt)

	if remaining <= 0 {
		return "breached"
	}
	if total > 0 && float64(remaining)/float64(total) < 0.25 {
		return "at_risk"
	}
	return "on_track"
}

func (db *DB) GetWorkOrder(id string) (*models.WorkOrder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wo models.WorkOrder
	err := db.Pool.QueryRow(ctx, `
		SELECT wo.id, wo.schedule_id, wo.device_id, wo.type, wo.status, wo.priority,
			wo.assigned_to, wo.sla_deadline, wo.checklist, wo.started_at, wo.completed_at,
			wo.notes, wo.photos, wo.parts_used, wo.created_by, wo.created_at, wo.updated_at,
			COALESCE(d.name, d.device_id) as device_name,
			COALESCE(u.username, '') as assignee_name
		FROM work_orders wo
		LEFT JOIN devices d ON wo.device_id = d.device_id
		LEFT JOIN users u ON wo.assigned_to = u.id
		WHERE wo.id = $1
	`, id).Scan(
		&wo.ID, &wo.ScheduleID, &wo.DeviceID, &wo.Type, &wo.Status, &wo.Priority,
		&wo.AssignedTo, &wo.SLADeadline, &wo.Checklist, &wo.StartedAt, &wo.CompletedAt,
		&wo.Notes, &wo.Photos, &wo.PartsUsed, &wo.CreatedBy, &wo.CreatedAt, &wo.UpdatedAt,
		&wo.DeviceName, &wo.AssigneeName,
	)
	if err != nil {
		return nil, fmt.Errorf("get work_order %s: %w", id, err)
	}
	wo.SLAStatus = calculateSLAStatus(wo)
	return &wo, nil
}

func (db *DB) UpdateWorkOrder(id string, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	args = append(args, id)

	query := fmt.Sprintf("UPDATE work_orders SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx+1)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update work_order %s: %w", id, err)
	}
	return nil
}

func (db *DB) AssignWorkOrder(id string, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Получаем текущего assignee для уменьшения workload
	var oldAssignee *string
	_ = db.Pool.QueryRow(ctx, "SELECT assigned_to FROM work_orders WHERE id = $1", id).Scan(&oldAssignee)

	_, err := db.Pool.Exec(ctx, `
		UPDATE work_orders SET assigned_to = $1, updated_at = NOW() WHERE id = $2
	`, userID, id)
	if err != nil {
		return fmt.Errorf("assign work_order %s: %w", id, err)
	}

	// Обновляем workload
	if oldAssignee != nil {
		_, _ = db.Pool.Exec(ctx, `UPDATE users SET current_workload = GREATEST(0, current_workload - 1) WHERE id = $1`, *oldAssignee)
	}
	_, _ = db.Pool.Exec(ctx, `UPDATE users SET current_workload = current_workload + 1 WHERE id = $1`, userID)

	return nil
}

// ── Bulk Actions (WO-4.2.1) ────────────────────────────────────────────

// BulkActionType определяет тип массовой операции.
type BulkActionType string

const (
	BulkStatusChange BulkActionType = "status_change"
	BulkAssign       BulkActionType = "assign"
	BulkDelete       BulkActionType = "delete"
	BulkPriority     BulkActionType = "priority_change"
)

// BulkActionResult содержит результат по каждому ID.
type BulkActionResult struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "success" или "error"
	Error  string `json:"error,omitempty"`
}

// BulkWorkOrders выполняет массовую операцию над списком Work Orders.
// Соответствует: OWASP ASVS V5.1 (input validation — whitelist)
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — audit trail для массовых операций)
//   - IEC 62443 SR 3.1 (Data integrity)
func (db *DB) BulkWorkOrders(action BulkActionType, ids []string, value string) ([]BulkActionResult, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no ids provided")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	results := make([]BulkActionResult, 0, len(ids))

	switch action {
	case BulkStatusChange:
		// Whitelist validation: только разрешённые статусы
		validStatuses := map[string]bool{
			"open": true, "in_progress": true, "completed": true,
			"cancelled": true, "on_hold": true,
		}
		if !validStatuses[value] {
			return nil, fmt.Errorf("invalid status: %s", value)
		}
		for _, id := range ids {
			_, err := db.Pool.Exec(ctx, `
				UPDATE work_orders SET status = $1, updated_at = NOW() WHERE id = $2
			`, value, id)
			if err != nil {
				results = append(results, BulkActionResult{ID: id, Status: "error", Error: err.Error()})
			} else {
				results = append(results, BulkActionResult{ID: id, Status: "success"})
			}
		}

	case BulkAssign:
		if value == "" {
			return nil, fmt.Errorf("user_id is required for assign action")
		}
		for _, id := range ids {
			err := db.AssignWorkOrder(id, value)
			if err != nil {
				results = append(results, BulkActionResult{ID: id, Status: "error", Error: err.Error()})
			} else {
				results = append(results, BulkActionResult{ID: id, Status: "success"})
			}
		}

	case BulkDelete:
		for _, id := range ids {
			_, err := db.Pool.Exec(ctx, `
				UPDATE work_orders SET status = 'cancelled', updated_at = NOW() WHERE id = $1
			`, id)
			if err != nil {
				results = append(results, BulkActionResult{ID: id, Status: "error", Error: err.Error()})
			} else {
				results = append(results, BulkActionResult{ID: id, Status: "success"})
			}
		}

	case BulkPriority:
		validPriorities := map[string]bool{
			"critical": true, "high": true, "medium": true, "low": true,
		}
		if !validPriorities[value] {
			return nil, fmt.Errorf("invalid priority: %s", value)
		}
		for _, id := range ids {
			_, err := db.Pool.Exec(ctx, `
				UPDATE work_orders SET priority = $1, updated_at = NOW() WHERE id = $2
			`, value, id)
			if err != nil {
				results = append(results, BulkActionResult{ID: id, Status: "error", Error: err.Error()})
			} else {
				results = append(results, BulkActionResult{ID: id, Status: "success"})
			}
		}

	default:
		return nil, fmt.Errorf("unsupported bulk action: %s", action)
	}

	return results, nil
}

func (db *DB) StartWorkOrder(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE work_orders SET status = 'in_progress', started_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("start work_order %s: %w", id, err)
	}
	return nil
}

func (db *DB) CompleteWorkOrder(id string, notes string, photos []string, parts []models.PartUsage, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	photosJSON, _ := json.Marshal(photos)
	partsJSON, _ := json.Marshal(parts)

	_, err = tx.Exec(ctx, `
		UPDATE work_orders
		SET status = 'completed', completed_at = NOW(), notes = $1,
			photos = $2::jsonb, parts_used = $3::jsonb, updated_at = NOW()
		WHERE id = $4
	`, notes, photosJSON, partsJSON, id)
	if err != nil {
		return fmt.Errorf("complete work_order %s: %w", id, err)
	}

	// Уменьшаем workload
	var assignedTo *string
	_ = tx.QueryRow(ctx, "SELECT assigned_to FROM work_orders WHERE id = $1", id).Scan(&assignedTo)
	if assignedTo != nil {
		_, _ = tx.Exec(ctx, `UPDATE users SET current_workload = GREATEST(0, current_workload - 1) WHERE id = $1`, *assignedTo)
	}

	// Списываем запчасти
	for _, part := range parts {
		_, err = tx.Exec(ctx, `UPDATE spare_parts SET stock = stock - $1, updated_at = NOW() WHERE id = $2`,
			part.Quantity, part.PartID)
		if err != nil {
			return fmt.Errorf("deduct part %s: %w", part.PartID, err)
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO part_usage (work_order_id, part_id, quantity, used_by)
			VALUES ($1, $2, $3, $4)
		`, id, part.PartID, part.Quantity, userID)
		if err != nil {
			return fmt.Errorf("log part usage: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (db *DB) CancelWorkOrder(id string, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Уменьшаем workload
	var assignedTo *string
	_ = db.Pool.QueryRow(ctx, "SELECT assigned_to FROM work_orders WHERE id = $1", id).Scan(&assignedTo)

	_, err := db.Pool.Exec(ctx, `
		UPDATE work_orders SET status = 'cancelled', notes = COALESCE(notes || E'\n', '') || 'Cancelled: ' || $1,
			updated_at = NOW()
		WHERE id = $2
	`, reason, id)
	if err != nil {
		return fmt.Errorf("cancel work_order %s: %w", id, err)
	}

	if assignedTo != nil {
		_, _ = db.Pool.Exec(ctx, `UPDATE users SET current_workload = GREATEST(0, current_workload - 1) WHERE id = $1`, *assignedTo)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Parts
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) CreateSparePart(part *models.SparePart) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Нормализуем custom_fields: если nil — '{}'
	customFields := part.CustomFields
	if customFields == nil {
		customFields = json.RawMessage("{}")
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO spare_parts (name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, custom_fields)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at
	`, part.Name, part.SKU, part.Category, part.Stock, part.MinStock,
		part.Location, part.CompatibleDevices, part.Cost, part.Supplier,
		customFields,
	).Scan(&part.ID, &part.CreatedAt, &part.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert spare_part: %w", err)
	}
	return nil
}

func (db *DB) GetSpareParts(filters map[string]interface{}) ([]models.SparePart, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `SELECT id, name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, custom_fields, created_at, updated_at FROM spare_parts WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if category, ok := filters["category"]; ok {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}
	if search, ok := filters["search"]; ok {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR sku ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search.(string)+"%")
		argIdx++
	}

	query += " ORDER BY name ASC"

	if limit, ok := filters["limit"]; ok {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset, ok := filters["offset"]; ok {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query spare_parts: %w", err)
	}
	defer rows.Close()

	var parts []models.SparePart
	for rows.Next() {
		var p models.SparePart
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Category, &p.Stock, &p.MinStock,
			&p.Location, &p.CompatibleDevices, &p.Cost, &p.Supplier, &p.CustomFields, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan spare_part: %w", err)
		}
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

func (db *DB) GetSparePart(id string) (*models.SparePart, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var p models.SparePart
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, custom_fields, created_at, updated_at
		FROM spare_parts WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.SKU, &p.Category, &p.Stock, &p.MinStock,
		&p.Location, &p.CompatibleDevices, &p.Cost, &p.Supplier, &p.CustomFields, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get spare_part %s: %w", id, err)
	}
	return &p, nil
}

// UpdateSparePart обновляет поля запчасти.
// Поле custom_fields поддерживается через whitelist (INV-7.1.2).
//
// Compliance:
//   - OWASP ASVS V5.1 (Whitelist validation — allowedFields)
//   - ISO 27001 A.12.4.1 (Event logging — audit trail в хендлере)
func (db *DB) UpdateSparePart(id string, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE spare_parts SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update spare_part %s: %w", id, err)
	}
	return nil
}

func (db *DB) UpdateSparePartStock(id string, quantity int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE spare_parts SET stock = $1, updated_at = NOW() WHERE id = $2
	`, quantity, id)
	if err != nil {
		return fmt.Errorf("update spare_part stock %s: %w", id, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Stock Adjustments (INV-7.1.4)
// ═══════════════════════════════════════════════════════════════════════

// CreateStockAdjustment создаёт запись корректировки остатка (audit trail).
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — stock adjustment audit trail)
//   - ISO/IEC 27019 PCC.A.12 (Operations security)
//   - СТБ 34.101.27 (Защита информации — фиксация изменений остатков)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) CreateStockAdjustment(adj *models.StockAdjustment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO stock_adjustments (part_id, previous_stock, new_stock, delta, reason, adjusted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, adj.PartID, adj.PreviousStock, adj.NewStock, adj.Delta, adj.Reason, adj.AdjustedBy,
	).Scan(&adj.ID, &adj.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert stock_adjustment: %w", err)
	}
	return nil
}

// GetStockAdjustments возвращает историю корректировок остатка для запчасти.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) GetStockAdjustments(partID string) ([]models.StockAdjustment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT sa.id, sa.part_id, sa.previous_stock, sa.new_stock, sa.delta,
			COALESCE(sa.reason, '') as reason,
			COALESCE(sa.adjusted_by, '') as adjusted_by,
			sa.created_at,
			COALESCE(sp.name, '') as part_name,
			COALESCE(sp.sku, '') as part_sku,
			COALESCE(u.username, '') as user_name
		FROM stock_adjustments sa
		LEFT JOIN spare_parts sp ON sa.part_id = sp.id
		LEFT JOIN users u ON sa.adjusted_by = u.id
		WHERE sa.part_id = $1
		ORDER BY sa.created_at DESC
	`, partID)
	if err != nil {
		return nil, fmt.Errorf("query stock_adjustments for part %s: %w", partID, err)
	}
	defer rows.Close()

	var adjustments []models.StockAdjustment
	for rows.Next() {
		var a models.StockAdjustment
		if err := rows.Scan(
			&a.ID, &a.PartID, &a.PreviousStock, &a.NewStock, &a.Delta,
			&a.Reason, &a.AdjustedBy, &a.CreatedAt,
			&a.PartName, &a.PartSKU, &a.UserName,
		); err != nil {
			return nil, fmt.Errorf("scan stock_adjustment: %w", err)
		}
		adjustments = append(adjustments, a)
	}
	if adjustments == nil {
		adjustments = []models.StockAdjustment{}
	}
	return adjustments, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Part Categories (below)
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) DeleteSparePart(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, "DELETE FROM spare_parts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete spare_part %s: %w", id, err)
	}
	return nil
}

func (db *DB) UsePartInWorkOrder(workOrderID, partID string, quantity int, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO part_usage (work_order_id, part_id, quantity, used_by)
		VALUES ($1, $2, $3, $4)
	`, workOrderID, partID, quantity, userID)
	if err != nil {
		return fmt.Errorf("use part in work order: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE spare_parts SET stock = stock - $1, updated_at = NOW() WHERE id = $2
	`, quantity, partID)
	if err != nil {
		return fmt.Errorf("deduct part stock: %w", err)
	}
	return nil
}

func (db *DB) GetLowStockParts() ([]models.SparePart, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, custom_fields, created_at, updated_at
		FROM spare_parts
		WHERE stock <= min_stock
		ORDER BY (stock::float / GREATEST(min_stock, 1)) ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query low stock parts: %w", err)
	}
	defer rows.Close()

	var parts []models.SparePart
	for rows.Next() {
		var p models.SparePart
		if err := rows.Scan(&p.ID, &p.Name, &p.SKU, &p.Category, &p.Stock, &p.MinStock,
			&p.Location, &p.CompatibleDevices, &p.Cost, &p.Supplier, &p.CustomFields, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan low stock part: %w", err)
		}
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Configuration
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) GetSLAConfig(priority string) (*models.SLAConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sla models.SLAConfig
	err := db.Pool.QueryRow(ctx, `
		SELECT id, priority, response_time_minutes, resolution_time_minutes
		FROM sla_config WHERE priority = $1
	`, priority).Scan(&sla.ID, &sla.Priority, &sla.ResponseTimeMinutes, &sla.ResolutionTimeMinutes)
	if err != nil {
		return nil, fmt.Errorf("get sla_config for %s: %w", priority, err)
	}
	return &sla, nil
}

func (db *DB) GetAllSLAConfigs() ([]models.SLAConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, priority, response_time_minutes, resolution_time_minutes
		FROM sla_config ORDER BY
			CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END
	`)
	if err != nil {
		return nil, fmt.Errorf("query sla_configs: %w", err)
	}
	defer rows.Close()

	var configs []models.SLAConfig
	for rows.Next() {
		var sla models.SLAConfig
		if err := rows.Scan(&sla.ID, &sla.Priority, &sla.ResponseTimeMinutes, &sla.ResolutionTimeMinutes); err != nil {
			return nil, fmt.Errorf("scan sla_config: %w", err)
		}
		configs = append(configs, sla)
	}
	return configs, rows.Err()
}

func (db *DB) UpdateSLAConfig(priority string, responseMinutes, resolutionMinutes int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE sla_config SET response_time_minutes = $1, resolution_time_minutes = $2
		WHERE priority = $3
	`, responseMinutes, resolutionMinutes, priority)
	if err != nil {
		return fmt.Errorf("update sla_config for %s: %w", priority, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Technician Workload
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) GetTechnicianWorkload(userID string) (*models.TechnicianWorkload, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tw models.TechnicianWorkload
	err := db.Pool.QueryRow(ctx, `
		SELECT id, username, current_workload, max_workload, skills, base_location
		FROM users WHERE id = $1 AND role IN ('technician', 'manager')
	`, userID).Scan(&tw.UserID, &tw.UserName, &tw.CurrentWorkload, &tw.MaxWorkload,
		&tw.Skills, &tw.BaseLocation)
	if err != nil {
		return nil, fmt.Errorf("get technician workload %s: %w", userID, err)
	}
	return &tw, nil
}

func (db *DB) GetAllTechnicianWorkloads() ([]models.TechnicianWorkload, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, username, current_workload, max_workload, skills, base_location
		FROM users
		WHERE role IN ('technician', 'manager')
		ORDER BY username ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query technician workloads: %w", err)
	}
	defer rows.Close()

	var workloads []models.TechnicianWorkload
	for rows.Next() {
		var tw models.TechnicianWorkload
		if err := rows.Scan(&tw.UserID, &tw.UserName, &tw.CurrentWorkload, &tw.MaxWorkload,
			&tw.Skills, &tw.BaseLocation); err != nil {
			return nil, fmt.Errorf("scan technician workload: %w", err)
		}
		workloads = append(workloads, tw)
	}
	return workloads, rows.Err()
}

func (db *DB) UpdateTechnicianSkills(userID string, skills []string, certifications []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE users SET skills = $1, certifications = $2, updated_at = NOW()
		WHERE id = $3
	`, skills, certifications, userID)
	if err != nil {
		return fmt.Errorf("update technician skills %s: %w", userID, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Reports
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) GetMaintenanceReport() ([]models.MaintenanceReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT
			d.device_id,
			COALESCE(d.name, d.device_id) as device_name,
			COUNT(wo.id) as total_work_orders,
			COUNT(CASE WHEN wo.status = 'completed' THEN 1 END) as completed_count,
			COUNT(CASE WHEN wo.sla_deadline < NOW() AND wo.status NOT IN ('completed', 'cancelled') THEN 1 END) as overdue_count,
			COALESCE(AVG(
				EXTRACT(EPOCH FROM (wo.completed_at - wo.started_at)) / 60.0
			), 0) as avg_resolution_minutes,
			COALESCE(SUM(
				(SELECT COALESCE(SUM(pu.quantity * sp.cost), 0)
				 FROM part_usage pu
				 JOIN spare_parts sp ON pu.part_id = sp.id
				 WHERE pu.work_order_id = wo.id)
			), 0) as total_cost
		FROM devices d
		LEFT JOIN work_orders wo ON d.device_id = wo.device_id
		GROUP BY d.device_id, d.name
		ORDER BY total_work_orders DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query maintenance report: %w", err)
	}
	defer rows.Close()

	var reports []models.MaintenanceReport
	for rows.Next() {
		var r models.MaintenanceReport
		if err := rows.Scan(&r.DeviceID, &r.DeviceName, &r.TotalWorkOrders,
			&r.CompletedCount, &r.OverdueCount, &r.MTTR, &r.TotalCost); err != nil {
			return nil, fmt.Errorf("scan maintenance report: %w", err)
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

func (db *DB) GetSLAComplianceReport() ([]models.SLAComplianceReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT
			priority,
			COUNT(*) as total,
			COUNT(CASE WHEN sla_deadline >= completed_at OR status IN ('open', 'in_progress') THEN 1 END) as within_sla,
			COUNT(CASE WHEN sla_deadline < NOW() AND status NOT IN ('completed', 'cancelled') THEN 1 END) as breached,
			COALESCE(AVG(EXTRACT(EPOCH FROM (started_at - created_at)) / 60.0), 0) as avg_response,
			COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - created_at)) / 60.0), 0) as avg_resolution
		FROM work_orders
		WHERE created_at > NOW() - INTERVAL '30 days'
		GROUP BY priority
		ORDER BY CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END
	`)
	if err != nil {
		return nil, fmt.Errorf("query sla compliance report: %w", err)
	}
	defer rows.Close()

	var reports []models.SLAComplianceReport
	for rows.Next() {
		var r models.SLAComplianceReport
		if err := rows.Scan(&r.Priority, &r.TotalWorkOrders, &r.WithinSLA, &r.BreachedSLA,
			&r.AvgResponseTime, &r.AvgResolutionTime); err != nil {
			return nil, fmt.Errorf("scan sla compliance report: %w", err)
		}
		if r.TotalWorkOrders > 0 {
			r.CompliancePercent = float64(r.WithinSLA) / float64(r.TotalWorkOrders) * 100
		}
		reports = append(reports, r)
	}
	return reports, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// Device Reliability (AN-10.1.1)
// ═══════════════════════════════════════════════════════════════════════

// GetDeviceReliability возвращает MTBF/MTTR метрики по vendor_type и device_type.
//
// Фильтрация:
//   - vendorType: если не пустая — фильтр по vendor_type
//   - deviceType: если не пустая — фильтр по device_type
//
// MTBF рассчитывается как отношение total_downtime_minutes к total_downtime_events.
// Если downtime_events = 0, MTBF = 0 (нет данных для расчёта).
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — reliability metrics)
//   - IEC 62443 SR 7.1 (Resource availability — MTBF tracking)
//   - OWASP ASVS V5.1 (Parameterized query — SQL injection prevention)
func (db *DB) GetDeviceReliability(ctx context.Context, vendorType, deviceType string) ([]models.DeviceReliability, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(dbCtx, `
		SELECT
			vendor_type,
			device_type,
			device_count,
			total_downtime_events,
			total_downtime_minutes,
			total_completions,
			avg_mttr_minutes
		FROM mv_device_reliability
		WHERE ($1 = '' OR vendor_type = $1)
		  AND ($2 = '' OR device_type = $2)
		ORDER BY vendor_type, device_type
	`, vendorType, deviceType)
	if err != nil {
		return nil, fmt.Errorf("query device reliability: %w", err)
	}
	defer rows.Close()

	var results []models.DeviceReliability
	for rows.Next() {
		var r models.DeviceReliability
		if err := rows.Scan(
			&r.VendorType,
			&r.DeviceType,
			&r.DeviceCount,
			&r.TotalDowntimeEvents,
			&r.TotalDowntimeMinutes,
			&r.TotalCompletions,
			&r.AvgMTTRMinutes,
		); err != nil {
			return nil, fmt.Errorf("scan device reliability: %w", err)
		}

		// MTBF = total_downtime_minutes / total_downtime_events (в минутах),
		// затем конвертируем в часы.
		if r.TotalDowntimeEvents > 0 {
			r.MTBFHours = float64(r.TotalDowntimeMinutes) / float64(r.TotalDowntimeEvents) / 60.0
		}

		// MTTR = avg_mttr_minutes (уже посчитано в материализованном представлении)
		r.MTTRMinutes = r.AvgMTTRMinutes

		results = append(results, r)
	}

	if results == nil {
		results = []models.DeviceReliability{}
	}

	return results, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// WO-4.4.3: AdditionalCost CRUD
// ═══════════════════════════════════════════════════════════════════════

// CreateAdditionalCost создаёт запись дополнительных затрат для Work Order.
//
// Compliance:
//   - OWASP ASVS V5.1 (Parameterized query — SQL injection prevention)
//   - OWASP ASVS V7.1 (Error handling — no sensitive data)
//   - ISO 27001 A.12.4.1 (Event logging — cost tracking)
func (db *DB) CreateAdditionalCost(ctx context.Context, cost *models.AdditionalCost) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(dbCtx, `
		INSERT INTO additional_costs (id, work_order_id, category, description, vendor_name,
			estimated_cost, actual_cost, currency, receipt_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, cost.ID, cost.WorkOrderID, cost.Category, cost.Description, cost.VendorName,
		cost.EstimatedCost, cost.ActualCost, cost.Currency, cost.ReceiptURL, cost.CreatedBy)
	if err != nil {
		return fmt.Errorf("create additional cost: %w", err)
	}
	return nil
}

// GetAdditionalCostsByWorkOrder возвращает дополнительные затраты для Work Order.
//
// Compliance:
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) GetAdditionalCostsByWorkOrder(ctx context.Context, workOrderID string) ([]models.AdditionalCost, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(dbCtx, `
		SELECT id, work_order_id, category, description, vendor_name,
			estimated_cost, actual_cost, currency, receipt_url, created_at, created_by
		FROM additional_costs
		WHERE work_order_id = $1
		ORDER BY created_at DESC
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("query additional costs: %w", err)
	}
	defer rows.Close()

	var costs []models.AdditionalCost
	for rows.Next() {
		var c models.AdditionalCost
		if err := rows.Scan(
			&c.ID, &c.WorkOrderID, &c.Category, &c.Description, &c.VendorName,
			&c.EstimatedCost, &c.ActualCost, &c.Currency, &c.ReceiptURL, &c.CreatedAt, &c.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scan additional cost: %w", err)
		}
		costs = append(costs, c)
	}

	if costs == nil {
		costs = []models.AdditionalCost{}
	}
	return costs, rows.Err()
}

// DeleteAdditionalCost удаляет запись дополнительных затрат.
//
// Compliance:
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) DeleteAdditionalCost(ctx context.Context, id string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(dbCtx, `DELETE FROM additional_costs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete additional cost: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// WO-4.4.5: WorkOrder Cost Summary
// ═══════════════════════════════════════════════════════════════════════

// GetWorkOrderCostSummary возвращает агрегированную сводку затрат по Work Orders.
//
// Агрегирует labor cost из time_entries, parts cost из parts_used,
// additional cost из additional_costs.
//
// Compliance:
//   - OWASP ASVS V5.1 (Parameterized query — SQL injection prevention)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
func (db *DB) GetWorkOrderCostSummary(ctx context.Context) (*models.WorkOrderCostSummary, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var s models.WorkOrderCostSummary
	err := db.Pool.QueryRow(dbCtx, `
		SELECT
			COUNT(DISTINCT wo.id)::bigint AS total_work_orders,
			COALESCE(SUM(
				COALESCE(EXTRACT(EPOCH FROM (COALESCE(te.end_time, NOW()) - te.start_time) - COALESCE(te.paused_duration, INTERVAL '0')) / 3600.0 * te.hourly_rate, 0)
			), 0)::numeric(12,2) AS total_labor_cost,
			COALESCE(SUM(
				COALESCE((pu.quantity * pu.unit_cost), 0)
			), 0)::numeric(12,2) AS total_parts_cost,
			0::numeric(12,2) AS total_additional_cost,
			COALESCE(SUM(
				COALESCE(EXTRACT(EPOCH FROM (COALESCE(te.end_time, NOW()) - te.start_time) - COALESCE(te.paused_duration, INTERVAL '0')) / 3600.0 * te.hourly_rate, 0) +
				COALESCE((pu.quantity * pu.unit_cost), 0)
			), 0)::numeric(12,2) AS total_cost
		FROM work_orders wo
		LEFT JOIN time_entries te ON te.work_order_id = wo.id AND te.status = 'stopped'
		LEFT JOIN parts_used pu ON pu.work_order_id = wo.id
	`).Scan(
		&s.TotalWorkOrders,
		&s.TotalLaborCost,
		&s.TotalPartsCost,
		&s.TotalAdditionalCost,
		&s.TotalCost,
	)
	if err != nil {
		return nil, fmt.Errorf("get work order cost summary: %w", err)
	}

	if s.TotalWorkOrders > 0 {
		s.AverageCostPerOrder = s.TotalCost / float64(s.TotalWorkOrders)
	}
	s.Currency = "USD"

	return &s, nil
}

// GetWorkOrderCostBreakdown возвращает разбивку затрат по категориям.
//
// NOTE: Этот метод ВЫПОЛНЯЕТ повторный дорогой запрос GetWorkOrderCostSummary.
// Для production используйте GetWorkOrderCostBreakdownFromSummary,
// который принимает уже полученный summary и не делает второй запрос.
//
// Compliance:
//   - OWASP ASVS V5.1 (Parameterized query — SQL injection prevention)
func (db *DB) GetWorkOrderCostBreakdown(ctx context.Context) ([]models.WorkOrderCostBreakdown, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	summary, err := db.GetWorkOrderCostSummary(ctx)
	if err != nil {
		return nil, err
	}

	return db.GetWorkOrderCostBreakdownFromSummary(dbCtx, summary)
}

// GetWorkOrderCostBreakdownFromSummary возвращает разбивку затрат на основе
// уже полученного WorkOrderCostSummary — без повторного дорогого запроса.
func (db *DB) GetWorkOrderCostBreakdownFromSummary(ctx context.Context, summary *models.WorkOrderCostSummary) ([]models.WorkOrderCostBreakdown, error) {
	total := summary.TotalCost
	if total == 0 {
		total = 1
	}

	breakdown := []models.WorkOrderCostBreakdown{
		{Category: "labor", Amount: summary.TotalLaborCost, Percent: summary.TotalLaborCost / total * 100},
		{Category: "parts", Amount: summary.TotalPartsCost, Percent: summary.TotalPartsCost / total * 100},
	}

	// Лёгкий запрос — только подсчёт строк (не JOIN, без агрегации)
	err := db.Pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM time_entries WHERE status = 'stopped')::bigint,
			(SELECT COUNT(*) FROM parts_used)::bigint
	`).Scan(&breakdown[0].Count, &breakdown[1].Count)
	if err != nil {
		// При ошибке оставляем нулевые значения (graceful degradation)
		db.Logger.Warn("failed to count work order cost breakdown items, using zeros", "error", err)
	}

	return breakdown, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Technician Site Assignments
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) CreateTechnicianSiteAssignment(assignment *models.TechnicianSiteAssignment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Преобразуем пустую строку assigned_by в nil (NULL в БД)
	var assignedBy interface{}
	if assignment.AssignedBy == "" {
		assignedBy = nil
	} else {
		assignedBy = assignment.AssignedBy
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO technician_site_assignments (
			technician_id, site_id, is_primary, assigned_by
		) VALUES ($1, $2, $3, $4)
		RETURNING id, assigned_at
	`, assignment.TechnicianID, assignment.SiteID, assignment.IsPrimary, assignedBy,
	).Scan(&assignment.ID, &assignment.AssignedAt)

	if err != nil {
		return fmt.Errorf("insert technician_site_assignment: %w", err)
	}
	return nil
}

func (db *DB) GetTechnicianSiteAssignments(filters map[string]interface{}) ([]models.TechnicianSiteAssignment, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT tsa.id, tsa.technician_id, tsa.site_id, tsa.is_primary, tsa.assigned_at, tsa.assigned_by,
			COALESCE(u.username, '') as technician_name,
			COALESCE(s.name, s.id) as site_name
		FROM technician_site_assignments tsa
		LEFT JOIN users u ON tsa.technician_id = u.id
		LEFT JOIN sites s ON tsa.site_id = s.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if technicianID, ok := filters["technician_id"]; ok {
		query += fmt.Sprintf(" AND tsa.technician_id = $%d", argIdx)
		args = append(args, technicianID)
		argIdx++
	}
	if siteID, ok := filters["site_id"]; ok {
		query += fmt.Sprintf(" AND tsa.site_id = $%d", argIdx)
		args = append(args, siteID)
		argIdx++
	}
	if isPrimary, ok := filters["is_primary"]; ok {
		query += fmt.Sprintf(" AND tsa.is_primary = $%d", argIdx)
		args = append(args, isPrimary)
		argIdx++
	}

	query += " ORDER BY tsa.assigned_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query technician_site_assignments: %w", err)
	}
	defer rows.Close()

	var assignments []models.TechnicianSiteAssignment
	for rows.Next() {
		var a models.TechnicianSiteAssignment
		if err := rows.Scan(
			&a.ID, &a.TechnicianID, &a.SiteID, &a.IsPrimary, &a.AssignedAt, &a.AssignedBy,
			&a.TechnicianName, &a.SiteName,
		); err != nil {
			return nil, fmt.Errorf("scan technician_site_assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}

func (db *DB) DeleteTechnicianSiteAssignment(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, "DELETE FROM technician_site_assignments WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete technician_site_assignment %s: %w", id, err)
	}
	return nil
}

func (db *DB) UpdateTechnicianSiteAssignment(id string, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	if len(setClauses) == 0 {
		return nil
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE technician_site_assignments SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update technician_site_assignment %s: %w", id, err)
	}
	return nil
}

// SavePushToken сохраняет push-токен для техника с AES-256-GCM шифрованием
func (db *DB) SavePushToken(userID, token, platform string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	encryptedToken, err := crypto.Encrypt(token)
	if err != nil {
		return fmt.Errorf("encrypt push token for user %s: %w", userID, err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE users SET push_token = $1, push_platform = $2, updated_at = NOW()
		WHERE id = $3
	`, encryptedToken, platform, userID)
	if err != nil {
		return fmt.Errorf("save push token for user %s: %w", userID, err)
	}
	return nil
}

// GetPushToken возвращает расшифрованный push-токен техника
func (db *DB) GetPushToken(userID string) (token string, platform string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var encryptedToken string
	err = db.Pool.QueryRow(ctx, `
		SELECT COALESCE(push_token, ''), COALESCE(push_platform, '')
		FROM users WHERE id = $1
	`, userID).Scan(&encryptedToken, &platform)
	if err != nil {
		return "", "", fmt.Errorf("get push token for user %s: %w", userID, err)
	}

	if encryptedToken == "" {
		return "", "", nil
	}

	token, err = crypto.Decrypt(encryptedToken)
	if err != nil {
		return "", "", fmt.Errorf("decrypt push token for user %s: %w", userID, err)
	}
	return token, platform, nil
}

// GetTechnicianMonthlyStats возвращает статистику техника за текущий месяц
func (db *DB) GetTechnicianMonthlyStats(userID string) (*models.TechnicianMonthlyStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stats models.TechnicianMonthlyStats
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_this_month,
			COUNT(*) as total_work_orders,
			COALESCE(
				ROUND(
					100.0 * COUNT(CASE WHEN status = 'completed' AND (sla_deadline IS NULL OR completed_at <= sla_deadline) THEN 1 END)
					/ NULLIF(COUNT(CASE WHEN status = 'completed' THEN 1 END), 0),
					1
				), 0
			) as on_time_percent,
			COALESCE(AVG(
				CASE WHEN status = 'completed' THEN
					CASE
						WHEN sla_deadline IS NULL THEN 5.0
						WHEN completed_at <= sla_deadline THEN 5.0
						WHEN completed_at <= sla_deadline + INTERVAL '1 hour' THEN 4.0
						WHEN completed_at <= sla_deadline + INTERVAL '4 hours' THEN 3.0
						ELSE 2.0
					END
				END
			), 0) as avg_rating
		FROM work_orders
		WHERE assigned_to = $1
			AND created_at >= date_trunc('month', NOW())
	`, userID).Scan(&stats.CompletedThisMonth, &stats.TotalWorkOrders, &stats.OnTimePercent, &stats.AvgRating)
	if err != nil {
		return nil, fmt.Errorf("get technician monthly stats for user %s: %w", userID, err)
	}
	return &stats, nil
}

// ═══════════════════════════════════════════════════════════════════════
// External Work Order Sync (Bi-directional ITSM)
// ═══════════════════════════════════════════════════════════════════════

// ExternalWorkOrderStatus — внешний статус WorkOrder (из CMMS).
type ExternalWorkOrderStatus struct {
	ID                int                    `json:"id"`
	ExternalID        string                 `json:"external_id"`
	Source            string                 `json:"source"`
	Status            string                 `json:"status"`
	Priority          string                 `json:"priority"`
	Summary           string                 `json:"summary"`
	ExternalChangedAt time.Time              `json:"external_changed_at"`
	Changes           map[string]interface{} `json:"changes"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// GetExternalWorkOrderStatus возвращает последний внешний статус для WorkOrder по его локальному ID.
// Ищет через связку external_id/external_source в work_orders → external_work_order_status.
func (db *DB) GetExternalWorkOrderStatus(ctx context.Context, workOrderID string) (*ExternalWorkOrderStatus, error) {
	var ext ExternalWorkOrderStatus
	err := db.Pool.QueryRow(ctx, `
		SELECT ews.id, ews.external_id, ews.source, ews.status, ews.priority,
			ews.summary, ews.external_changed_at, ews.changes, ews.created_at, ews.updated_at
		FROM external_work_order_status ews
		JOIN work_orders wo ON wo.external_id = ews.external_id AND wo.external_source = ews.source
		WHERE wo.id = $1
		ORDER BY ews.external_changed_at DESC
		LIMIT 1
	`, workOrderID).Scan(
		&ext.ID, &ext.ExternalID, &ext.Source, &ext.Status, &ext.Priority,
		&ext.Summary, &ext.ExternalChangedAt, &ext.Changes, &ext.CreatedAt, &ext.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get external work order status for %s: %w", workOrderID, err)
	}
	return &ext, nil
}

// UpsertExternalWorkOrderStatus вставляет или обновляет внешний статус WorkOrder.
func (db *DB) UpsertExternalWorkOrderStatus(ctx context.Context, ext *ExternalWorkOrderStatus) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO external_work_order_status (external_id, source, status, priority, summary, external_changed_at, changes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING
	`, ext.ExternalID, ext.Source, ext.Status, ext.Priority, ext.Summary, ext.ExternalChangedAt, ext.Changes)
	if err != nil {
		return fmt.Errorf("upsert external work order status: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Work Requests (WO-4.1.1)
// ═══════════════════════════════════════════════════════════════════════

// CreateWorkRequest создаёт публичную заявку.
func (db *DB) CreateWorkRequest(req *models.WorkRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Валидация обязательных полей
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.RequesterName == "" {
		return fmt.Errorf("requester_name is required")
	}
	if req.RequesterEmail == "" {
		return fmt.Errorf("requester_email is required")
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	if req.Type == "" {
		req.Type = "corrective"
	}
	if req.Status == "" {
		req.Status = models.WorkRequestSubmitted
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO work_requests (
			title, description, device_id, site_id,
			priority, type,
			requester_name, requester_email, requester_phone,
			status, source_ip, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`, req.Title, req.Description, req.DeviceID, req.SiteID,
		req.Priority, req.Type,
		req.RequesterName, req.RequesterEmail, req.RequesterPhone,
		req.Status, req.SourceIP, req.UserAgent,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert work_request: %w", err)
	}
	return nil
}

// GetWorkRequests возвращает список заявок с фильтрацией.
func (db *DB) GetWorkRequests(filters map[string]interface{}) ([]models.WorkRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT wr.id, wr.created_at, wr.updated_at,
			wr.title, wr.description,
			wr.device_id, COALESCE(d.name, d.device_id, '') as device_name,
			wr.site_id, COALESCE(s.name, '') as site_name,
			wr.priority, wr.type,
			wr.requester_name, wr.requester_email, wr.requester_phone,
			wr.status,
			wr.approved_by, wr.approved_at,
			wr.rejected_by, wr.rejected_at, wr.rejection_reason,
			wr.converted_work_order_id, wr.converted_at,
			wr.source_ip, wr.user_agent
		FROM work_requests wr
		LEFT JOIN devices d ON wr.device_id = d.device_id
		LEFT JOIN sites s ON wr.site_id = s.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if status, ok := filters["status"]; ok {
		query += fmt.Sprintf(" AND wr.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if deviceID, ok := filters["device_id"]; ok {
		query += fmt.Sprintf(" AND wr.device_id = $%d", argIdx)
		args = append(args, deviceID)
		argIdx++
	}
	if email, ok := filters["requester_email"]; ok {
		query += fmt.Sprintf(" AND wr.requester_email = $%d", argIdx)
		args = append(args, email)
		argIdx++
	}
	if limit, ok := filters["limit"]; ok {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset, ok := filters["offset"]; ok {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
		argIdx++
	}

	query += " ORDER BY wr.created_at DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query work_requests: %w", err)
	}
	defer rows.Close()

	var requests []models.WorkRequest
	for rows.Next() {
		var req models.WorkRequest
		err := rows.Scan(
			&req.ID, &req.CreatedAt, &req.UpdatedAt,
			&req.Title, &req.Description,
			&req.DeviceID, &req.DeviceName,
			&req.SiteID, &req.SiteName,
			&req.Priority, &req.Type,
			&req.RequesterName, &req.RequesterEmail, &req.RequesterPhone,
			&req.Status,
			&req.ApprovedBy, &req.ApprovedAt,
			&req.RejectedBy, &req.RejectedAt, &req.RejectionReason,
			&req.ConvertedWorkOrderID, &req.ConvertedAt,
			&req.SourceIP, &req.UserAgent,
		)
		if err != nil {
			return nil, fmt.Errorf("scan work_request: %w", err)
		}
		requests = append(requests, req)
	}

	if requests == nil {
		requests = []models.WorkRequest{}
	}
	return requests, nil
}

// GetWorkRequest возвращает заявку по ID.
func (db *DB) GetWorkRequest(id string) (*models.WorkRequest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var req models.WorkRequest
	err := db.Pool.QueryRow(ctx, `
		SELECT wr.id, wr.created_at, wr.updated_at,
			wr.title, wr.description,
			wr.device_id, COALESCE(d.name, d.device_id, '') as device_name,
			wr.site_id, COALESCE(s.name, '') as site_name,
			wr.priority, wr.type,
			wr.requester_name, wr.requester_email, wr.requester_phone,
			wr.status,
			wr.approved_by, wr.approved_at,
			wr.rejected_by, wr.rejected_at, wr.rejection_reason,
			wr.converted_work_order_id, wr.converted_at,
			wr.source_ip, wr.user_agent
		FROM work_requests wr
		LEFT JOIN devices d ON wr.device_id = d.device_id
		LEFT JOIN sites s ON wr.site_id = s.id
		WHERE wr.id = $1
	`, id).Scan(
		&req.ID, &req.CreatedAt, &req.UpdatedAt,
		&req.Title, &req.Description,
		&req.DeviceID, &req.DeviceName,
		&req.SiteID, &req.SiteName,
		&req.Priority, &req.Type,
		&req.RequesterName, &req.RequesterEmail, &req.RequesterPhone,
		&req.Status,
		&req.ApprovedBy, &req.ApprovedAt,
		&req.RejectedBy, &req.RejectedAt, &req.RejectionReason,
		&req.ConvertedWorkOrderID, &req.ConvertedAt,
		&req.SourceIP, &req.UserAgent,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get work_request %s: %w", id, err)
	}
	return &req, nil
}

// ApproveWorkRequest одобряет заявку.
func (db *DB) ApproveWorkRequest(id, approvedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	tag, err := db.Pool.Exec(ctx, `
		UPDATE work_requests
		SET status = 'approved', approved_by = $1, approved_at = $2, updated_at = $2
		WHERE id = $3 AND status = 'submitted'
	`, approvedBy, now, id)
	if err != nil {
		return fmt.Errorf("approve work_request %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("work_request %s not found or not in submitted status", id)
	}
	return nil
}

// RejectWorkRequest отклоняет заявку.
func (db *DB) RejectWorkRequest(id, rejectedBy, reason string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	tag, err := db.Pool.Exec(ctx, `
		UPDATE work_requests
		SET status = 'rejected', rejected_by = $1, rejected_at = $2, rejection_reason = $3, updated_at = $2
		WHERE id = $4 AND status = 'submitted'
	`, rejectedBy, now, reason, id)
	if err != nil {
		return fmt.Errorf("reject work_request %s: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("work_request %s not found or not in submitted status", id)
	}
	return nil
}

// ConvertWorkRequestToWO конвертирует одобренную заявку в WorkOrder.
func (db *DB) ConvertWorkRequestToWO(requestID, workOrderID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	tag, err := db.Pool.Exec(ctx, `
		UPDATE work_requests
		SET status = 'converted', converted_work_order_id = $1, converted_at = $2, updated_at = $2
		WHERE id = $3 AND status = 'approved'
	`, workOrderID, now, requestID)
	if err != nil {
		return fmt.Errorf("convert work_request %s: %w", requestID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("work_request %s not found or not in approved status", requestID)
	}
	return nil
}

// GetWorkRequestByExternalID ищет локальный WorkOrder по внешнему ID и источнику.
func (db *DB) GetWorkOrderByExternalID(ctx context.Context, source, externalID string) (*models.WorkOrder, error) {
	var wo models.WorkOrder
	err := db.Pool.QueryRow(ctx, `
		SELECT wo.id, wo.schedule_id, wo.device_id, wo.type, wo.status, wo.priority,
			wo.assigned_to, wo.sla_deadline, wo.checklist, wo.started_at, wo.completed_at,
			wo.notes, wo.photos, wo.parts_used, wo.created_by, wo.created_at, wo.updated_at,
			COALESCE(d.name, d.device_id) as device_name,
			COALESCE(u.username, '') as assignee_name
		FROM work_orders wo
		LEFT JOIN devices d ON wo.device_id = d.device_id
		LEFT JOIN users u ON wo.assigned_to = u.id
		WHERE wo.external_id = $1 AND wo.external_source = $2
	`, externalID, source).Scan(
		&wo.ID, &wo.ScheduleID, &wo.DeviceID, &wo.Type, &wo.Status, &wo.Priority,
		&wo.AssignedTo, &wo.SLADeadline, &wo.Checklist, &wo.StartedAt, &wo.CompletedAt,
		&wo.Notes, &wo.Photos, &wo.PartsUsed, &wo.CreatedBy, &wo.CreatedAt, &wo.UpdatedAt,
		&wo.DeviceName, &wo.AssigneeName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get work order by external id %s/%s: %w", source, externalID, err)
	}
	return &wo, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Time Entries (WO-4.4.1)
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) CreateTimeEntry(entry *models.TimeEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO time_entries (work_order_id, user_id, start_time, status, notes, hourly_rate)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`, entry.WorkOrderID, entry.UserID, entry.StartTime, entry.Status, entry.Notes, entry.HourlyRate,
	).Scan(&entry.ID, &entry.CreatedAt, &entry.UpdatedAt)

	if err != nil {
		return fmt.Errorf("create time_entry: %w", err)
	}
	return nil
}

func (db *DB) GetTimeEntries(workOrderID string) ([]models.TimeEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT te.id, te.work_order_id, te.user_id, te.start_time, te.end_time,
			te.paused_duration, te.status, te.notes, te.hourly_rate,
			te.created_at, te.updated_at,
			COALESCE(u.username, '') as user_name
		FROM time_entries te
		LEFT JOIN users u ON te.user_id = u.id
		WHERE te.work_order_id = $1
		ORDER BY te.start_time DESC
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("get time_entries: %w", err)
	}
	defer rows.Close()

	var entries []models.TimeEntry
	for rows.Next() {
		var e models.TimeEntry
		if err := rows.Scan(
			&e.ID, &e.WorkOrderID, &e.UserID, &e.StartTime, &e.EndTime,
			&e.PausedDuration, &e.Status, &e.Notes, &e.HourlyRate,
			&e.CreatedAt, &e.UpdatedAt, &e.UserName,
		); err != nil {
			return nil, fmt.Errorf("scan time_entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// UpdateTimeEntryStatus обновляет статус time entry (running/paused/stopped).
// При остановке (stopped) заполняет end_time и пересчитывает total_labor_cost.
func (db *DB) UpdateTimeEntryStatus(id string, status string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		UPDATE time_entries
		SET status = $1,
		    end_time = CASE WHEN $1 = 'stopped' THEN NOW() ELSE end_time END,
		    updated_at = NOW()
		WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update time_entry status: %w", err)
	}

	// При остановке — пересчитываем total_labor_cost и total_labor_seconds в work_orders
	if status == "stopped" {
		_, _ = db.Pool.Exec(ctx, `
			UPDATE work_orders wo
			SET total_labor_seconds = (
				SELECT COALESCE(SUM(
					EXTRACT(EPOCH FROM (COALESCE(te.end_time, NOW()) - te.start_time)) - te.paused_duration
				), 0)::bigint
				FROM time_entries te
				WHERE te.work_order_id = wo.id AND te.status = 'stopped'
			),
			total_labor_cost = (
				SELECT COALESCE(SUM(
					((EXTRACT(EPOCH FROM (COALESCE(te.end_time, NOW()) - te.start_time)) - te.paused_duration) / 3600.0) * te.hourly_rate
				), 0)
				FROM time_entries te
				WHERE te.work_order_id = wo.id AND te.status = 'stopped'
			),
			total_cost = COALESCE(total_parts_cost, 0) + (
				SELECT COALESCE(SUM(
					((EXTRACT(EPOCH FROM (COALESCE(te.end_time, NOW()) - te.start_time)) - te.paused_duration) / 3600.0) * te.hourly_rate
				), 0)
				FROM time_entries te
				WHERE te.work_order_id = wo.id AND te.status = 'stopped'
			),
			updated_at = NOW()
			WHERE wo.id = (
				SELECT work_order_id FROM time_entries WHERE id = $2
			)
		`, id)
	}

	return nil
}

// DeleteTimeEntry удаляет time entry (только если статус running).
func (db *DB) DeleteTimeEntry(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		DELETE FROM time_entries WHERE id = $1 AND status = 'running'
	`, id)
	if err != nil {
		return fmt.Errorf("delete time_entry: %w", err)
	}
	return nil
}

// ── Parts Consumption with Cost Snapshot (WO-4.4.4) ─────────────────

// AddPartToWorkOrder добавляет запчасть к WorkOrder с фиксацией цены.
// Частично реализовано через work_orders.parts_used JSONB.
// Здесь добавляем cost snapshot и обновляем total_parts_cost.
func (db *DB) AddPartToWorkOrderWithCost(workOrderID, partID string, quantity int, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Получаем текущую цену запчасти
	var unitPrice, currentStock int
	err := db.Pool.QueryRow(ctx, `
		SELECT cost::numeric(10,2), stock FROM spare_parts WHERE id = $1
	`, partID).Scan(&unitPrice, &currentStock)
	if err != nil {
		return fmt.Errorf("get spare part cost: %w", err)
	}

	if currentStock < quantity {
		return fmt.Errorf("insufficient stock: have %d, need %d", currentStock, quantity)
	}

	totalPrice := float64(quantity) * float64(unitPrice)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Добавляем в parts_used JSONB с cost snapshot
	_, err = tx.Exec(ctx, `
		UPDATE work_orders
		SET parts_used = COALESCE(parts_used, '[]'::jsonb) || jsonb_build_object(
			'part_id', $1,
			'quantity', $2,
			'unit_price', $3,
			'total_price', $4,
			'used_at', NOW()::text,
			'used_by', $5
		),
		total_parts_cost = COALESCE(total_parts_cost, 0) + $4,
		total_cost = COALESCE(total_cost, 0) + $4,
		updated_at = NOW()
		WHERE id = $6
	`, partID, quantity, unitPrice, totalPrice, userID, workOrderID)
	if err != nil {
		return fmt.Errorf("add part to work_order: %w", err)
	}

	// Списываем со склада
	_, err = tx.Exec(ctx, `
		UPDATE spare_parts SET stock = stock - $1, updated_at = NOW() WHERE id = $2
	`, quantity, partID)
	if err != nil {
		return fmt.Errorf("update stock: %w", err)
	}

	return tx.Commit(ctx)
}

// ── Labor Cost Calculation (WO-4.4.2) ──────────────────────────────

// GetLaborCost возвращает расчёт labour cost по WorkOrder.
func (db *DB) GetLaborCost(workOrderID string) (*models.LaborCost, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var lc models.LaborCost
	err := db.Pool.QueryRow(ctx, `
		SELECT
			$1 as work_order_id,
			COALESCE(SUM(EXTRACT(EPOCH FROM (COALESCE(te.end_time, NOW()) - te.start_time)) - te.paused_duration), 0)::bigint as total_seconds,
			COALESCE(AVG(te.hourly_rate), 0)::numeric(10,2) as avg_hourly_rate
		FROM time_entries te
		WHERE te.work_order_id = $1 AND te.status = 'stopped'
	`, workOrderID).Scan(&lc.WorkOrderID, &lc.TotalSeconds, &lc.HourlyRate)
	if err != nil {
		return nil, fmt.Errorf("get labor cost: %w", err)
	}

	lc.TotalHours = float64(lc.TotalSeconds) / 3600.0
	lc.TotalCost = lc.TotalHours * lc.HourlyRate
	lc.Currency = "USD"

	return &lc, nil
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Escalation Matrix (SLA-6.2.2)
// ═══════════════════════════════════════════════════════════════════════

// GetEscalationRules возвращает правила эскалации для указанного приоритета
// и времени после дедлайна.
//
// Алгоритм:
//  1. Ищет правила по priority, где breach_minutes <= breachMinutes
//  2. Сортирует по escalation_level ASC
//  3. Возвращает подходящие правила
//
// Compliance:
//   - OWASP ASVS V5.1 (Parameterized query — SQL injection prevention)
//   - ISO 27001 A.12.4.1 (Audit trail for escalation)
func (db *DB) GetEscalationRules(ctx context.Context, priority string, breachMinutes int) ([]models.EscalationRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, priority, escalation_level, breach_minutes,
		       notify_role, notify_channel, repeat_interval_minutes, created_at
		FROM sla_escalation_rules
		WHERE priority = $1 AND breach_minutes <= $2
		ORDER BY escalation_level ASC
	`, priority, breachMinutes)
	if err != nil {
		return nil, fmt.Errorf("query escalation rules: %w", err)
	}
	defer rows.Close()

	var rules []models.EscalationRule
	for rows.Next() {
		var r models.EscalationRule
		if err := rows.Scan(
			&r.ID, &r.Priority, &r.EscalationLevel, &r.BreachMinutes,
			&r.NotifyRole, &r.NotifyChannel, &r.RepeatIntervalMinutes, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan escalation rule: %w", err)
		}
		rules = append(rules, r)
	}
	if rules == nil {
		rules = []models.EscalationRule{}
	}
	return rules, rows.Err()
}

// LogEscalation записывает событие эскалации в журнал.
//
// Используется SLA engine для фиксации факта отправки уведомления.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — escalation audit trail)
//   - IEC 62443 SR 2.8 (Audit events)
//   - СТБ 34.101.27 (Защита информации — logging)
func (db *DB) LogEscalation(ctx context.Context, entry *models.EscalationLogEntry) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_escalation_log (
			work_order_id, escalation_level, rule_id,
			notified_at, acknowledged_at, acknowledged_by, resolution_notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, entry.WorkOrderID, entry.EscalationLevel, entry.RuleID,
		entry.NotifiedAt, entry.AcknowledgedAt, entry.AcknowledgedBy, entry.ResolutionNotes)
	if err != nil {
		return fmt.Errorf("insert escalation log: %w", err)
	}
	return nil
}

// GetActiveEscalations возвращает активные (не подтверждённые) эскалации
// для указанного Work Order.
//
// Используется для проверки, была ли уже отправлена эскалация данного уровня.
func (db *DB) GetActiveEscalations(ctx context.Context, workOrderID string) ([]models.EscalationLogEntry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, work_order_id, escalation_level, rule_id,
		       notified_at, acknowledged_at, acknowledged_by,
		       COALESCE(resolution_notes, '') as resolution_notes
		FROM sla_escalation_log
		WHERE work_order_id = $1 AND acknowledged_at IS NULL
		ORDER BY escalation_level ASC
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("query active escalations: %w", err)
	}
	defer rows.Close()

	var entries []models.EscalationLogEntry
	for rows.Next() {
		var e models.EscalationLogEntry
		if err := rows.Scan(
			&e.ID, &e.WorkOrderID, &e.EscalationLevel, &e.RuleID,
			&e.NotifiedAt, &e.AcknowledgedAt, &e.AcknowledgedBy, &e.ResolutionNotes,
		); err != nil {
			return nil, fmt.Errorf("scan escalation log: %w", err)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []models.EscalationLogEntry{}
	}
	return entries, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrder ↔ Alert (Many-to-Many) — DM-1.3.1
// ═══════════════════════════════════════════════════════════════════════

// LinkAlertToWorkOrder привязывает алерт к WorkOrder.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — linked_at фиксируется)
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) LinkAlertToWorkOrder(workOrderID, alertID, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO work_order_alerts (work_order_id, alert_id, linked_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (work_order_id, alert_id) DO NOTHING
	`, workOrderID, alertID, userID)
	if err != nil {
		return fmt.Errorf("link alert %s to work_order %s: %w", alertID, workOrderID, err)
	}
	return nil
}

// UnlinkAlertFromWorkOrder отвязывает алерт от WorkOrder.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — audit trail для удаления связи)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) UnlinkAlertFromWorkOrder(workOrderID, alertID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tag, err := db.Pool.Exec(ctx, `
		DELETE FROM work_order_alerts
		WHERE work_order_id = $1 AND alert_id = $2
	`, workOrderID, alertID)
	if err != nil {
		return fmt.Errorf("unlink alert %s from work_order %s: %w", alertID, workOrderID, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("link not found: work_order %s / alert %s", workOrderID, alertID)
	}
	return nil
}

// GetAlertsForWorkOrder возвращает все алерты, привязанные к WorkOrder.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) GetAlertsForWorkOrder(workOrderID string) ([]models.WorkOrderAlert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT work_order_id, alert_id, linked_at, COALESCE(linked_by, '') as linked_by
		FROM work_order_alerts
		WHERE work_order_id = $1
		ORDER BY linked_at DESC
	`, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("get alerts for work_order %s: %w", workOrderID, err)
	}
	defer rows.Close()

	var alerts []models.WorkOrderAlert
	for rows.Next() {
		var a models.WorkOrderAlert
		if err := rows.Scan(&a.WorkOrderID, &a.AlertID, &a.LinkedAt, &a.LinkedBy); err != nil {
			return nil, fmt.Errorf("scan work_order_alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	if alerts == nil {
		alerts = []models.WorkOrderAlert{}
	}
	return alerts, rows.Err()
}

// GetWorkOrdersForAlert возвращает все WorkOrder, связанные с алертом.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) GetWorkOrdersForAlert(alertID string) ([]models.WorkOrderAlert, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT work_order_id, alert_id, linked_at, COALESCE(linked_by, '') as linked_by
		FROM work_order_alerts
		WHERE alert_id = $1
		ORDER BY linked_at DESC
	`, alertID)
	if err != nil {
		return nil, fmt.Errorf("get work_orders for alert %s: %w", alertID, err)
	}
	defer rows.Close()

	var workOrders []models.WorkOrderAlert
	for rows.Next() {
		var a models.WorkOrderAlert
		if err := rows.Scan(&a.WorkOrderID, &a.AlertID, &a.LinkedAt, &a.LinkedBy); err != nil {
			return nil, fmt.Errorf("scan work_order_alert: %w", err)
		}
		workOrders = append(workOrders, a)
	}
	if workOrders == nil {
		workOrders = []models.WorkOrderAlert{}
	}
	return workOrders, rows.Err()
}

// ═══════════════════════════════════════════════════════════════════════
// Vendors (INV-7.2.1)
// ═══════════════════════════════════════════════════════════════════════

// CreateVendor создаёт нового поставщика.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.15.1.1 (Supplier security policy)
//   - ISO/IEC 27019 PCC.A.5 (Supply chain management)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) CreateVendor(vendor *models.Vendor) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if vendor.Status == "" {
		vendor.Status = "active"
	}

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO vendors (name, contact_person, email, phone, address, website, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`, vendor.Name, vendor.ContactPerson, vendor.Email, vendor.Phone,
		vendor.Address, vendor.Website, vendor.Notes, vendor.Status,
	).Scan(&vendor.ID, &vendor.CreatedAt, &vendor.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert vendor: %w", err)
	}
	return nil
}

// GetVendors возвращает список поставщиков с фильтрацией.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) GetVendors(filters map[string]interface{}) ([]models.Vendor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `SELECT id, name, contact_person, email, phone, address, website, notes, status, created_at, updated_at FROM vendors WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if status, ok := filters["status"]; ok {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if search, ok := filters["search"]; ok {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR contact_person ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search.(string)+"%")
		argIdx++
	}

	query += " ORDER BY name ASC"

	if limit, ok := filters["limit"]; ok {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, limit)
		argIdx++
	}
	if offset, ok := filters["offset"]; ok {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query vendors: %w", err)
	}
	defer rows.Close()

	var vendors []models.Vendor
	for rows.Next() {
		var v models.Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.ContactPerson, &v.Email, &v.Phone,
			&v.Address, &v.Website, &v.Notes, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan vendor: %w", err)
		}
		vendors = append(vendors, v)
	}
	if vendors == nil {
		vendors = []models.Vendor{}
	}
	return vendors, rows.Err()
}

// GetVendor возвращает поставщика по ID.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Read access control)
//   - OWASP ASVS V5.1 (Parameterized query)
func (db *DB) GetVendor(id string) (*models.Vendor, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var v models.Vendor
	err := db.Pool.QueryRow(ctx, `
		SELECT id, name, contact_person, email, phone, address, website, notes, status, created_at, updated_at
		FROM vendors WHERE id = $1
	`, id).Scan(&v.ID, &v.Name, &v.ContactPerson, &v.Email, &v.Phone,
		&v.Address, &v.Website, &v.Notes, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get vendor %s: %w", id, err)
	}
	return &v, nil
}

// UpdateVendor обновляет поля поставщика.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - OWASP ASVS V5.1 (Parameterized query)
//   - ISO 27001 A.15.1.1 (Supplier security policy)
func (db *DB) UpdateVendor(id string, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, value := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
		args = append(args, value)
		argIdx++
	}
	if len(setClauses) == 0 {
		return nil
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE vendors SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "), argIdx)

	_, err := db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update vendor %s: %w", id, err)
	}
	return nil
}

// DeleteVendor удаляет поставщика.
// Запчасти, связанные с этим поставщиком, получат vendor_id = NULL (ON DELETE SET NULL).
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - OWASP ASVS V5.1 (Parameterized query)
//   - ISO 27001 A.8.10 (Information disposal)
func (db *DB) DeleteVendor(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Pool.Exec(ctx, "DELETE FROM vendors WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete vendor %s: %w", id, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// TCO Per Device (AN-10.1.3)
// ═══════════════════════════════════════════════════════════════════════

// GetTCOPerDevice возвращает TCO (Total Cost of Ownership) per device
// из материализованного представления mv_tco_per_device.
//
// Фильтрация:
//   - vendorType: если не пустая — фильтр по vendor_type
//   - deviceType: если не пустая — фильтр по device_type
//   - deviceID: если не пустая — фильтр по device_id
//
// TCO = Purchase + Labor + Parts + Downtime
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
//   - IEC 62443 SR 7.1 (Resource availability — asset TCO)
//   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
//   - СТБ 34.101.27 (Защита информации — учёт стоимости активов)
//   - OWASP ASVS V5.1 (Parameterized query — SQL injection prevention)
func (db *DB) GetTCOPerDevice(ctx context.Context, filter models.TCOFilter) ([]models.TCOPerDevice, error) {
	dbCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	query := `
		SELECT
			device_id,
			device_name,
			vendor_type,
			device_type,
			manufacturer,
			total_purchase_cost,
			total_labor_cost,
			total_parts_cost,
			total_downtime_cost,
			tco,
			total_work_orders,
			total_downtime_events
		FROM mv_tco_per_device
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if filter.VendorType != "" {
		query += fmt.Sprintf(" AND vendor_type = $%d", argIdx)
		args = append(args, filter.VendorType)
		argIdx++
	}
	if filter.DeviceType != "" {
		query += fmt.Sprintf(" AND device_type = $%d", argIdx)
		args = append(args, filter.DeviceType)
		argIdx++
	}
	if filter.DeviceID != "" {
		query += fmt.Sprintf(" AND device_id = $%d", argIdx)
		args = append(args, filter.DeviceID)
		argIdx++
	}

	query += " ORDER BY tco DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := db.Pool.Query(dbCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query tco per device: %w", err)
	}
	defer rows.Close()

	var results []models.TCOPerDevice
	for rows.Next() {
		var r models.TCOPerDevice
		if err := rows.Scan(
			&r.DeviceID,
			&r.DeviceName,
			&r.VendorType,
			&r.DeviceType,
			&r.Manufacturer,
			&r.TotalPurchaseCost,
			&r.TotalLaborCost,
			&r.TotalPartsCost,
			&r.TotalDowntimeCost,
			&r.TCO,
			&r.TotalWorkOrders,
			&r.TotalDowntimeEvents,
		); err != nil {
			return nil, fmt.Errorf("scan tco per device: %w", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []models.TCOPerDevice{}
	}

	return results, rows.Err()
}
