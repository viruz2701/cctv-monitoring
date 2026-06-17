package db

import (
	"context"
	"encoding/json"
	"fmt"
	"gb-telemetry-collector/internal/models"
	"strings"
	"time"
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

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO spare_parts (name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, part.Name, part.SKU, part.Category, part.Stock, part.MinStock,
		part.Location, part.CompatibleDevices, part.Cost, part.Supplier,
	).Scan(&part.ID, &part.CreatedAt, &part.UpdatedAt)

	if err != nil {
		return fmt.Errorf("insert spare_part: %w", err)
	}
	return nil
}

func (db *DB) GetSpareParts(filters map[string]interface{}) ([]models.SparePart, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `SELECT id, name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, created_at, updated_at FROM spare_parts WHERE 1=1`
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
			&p.Location, &p.CompatibleDevices, &p.Cost, &p.Supplier, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
		SELECT id, name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, created_at, updated_at
		FROM spare_parts WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.SKU, &p.Category, &p.Stock, &p.MinStock,
		&p.Location, &p.CompatibleDevices, &p.Cost, &p.Supplier, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get spare_part %s: %w", id, err)
	}
	return &p, nil
}

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
		SELECT id, name, sku, category, stock, min_stock, location, compatible_devices, cost, supplier, created_at, updated_at
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
			&p.Location, &p.CompatibleDevices, &p.Cost, &p.Supplier, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
// Technician Site Assignments
// ═══════════════════════════════════════════════════════════════════════

func (db *DB) CreateTechnicianSiteAssignment(assignment *models.TechnicianSiteAssignment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := db.Pool.QueryRow(ctx, `
		INSERT INTO technician_site_assignments (
			technician_id, site_id, is_primary, assigned_by
		) VALUES ($1, $2, $3, $4)
		RETURNING id, assigned_at
	`, assignment.TechnicianID, assignment.SiteID, assignment.IsPrimary, assignment.AssignedBy,
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
