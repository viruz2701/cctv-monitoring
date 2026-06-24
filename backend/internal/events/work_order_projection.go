// Package events — WorkOrderProjection (CQRS read-model).
//
// Строит read-model для Work Order статусов, агрегации по типам,
// времени выполнения и техникам.
//
// Compliance:
//   - CQRS pattern (разделение read/write models)
//   - ISO 27001 A.12.4.1 (Event logging — replay для аналитики)
//   - ISO 27001 A.12.6.1 (Capacity management)
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderProjection — read-model для Work Orders.
// ═══════════════════════════════════════════════════════════════════════

// WorkOrderStatusSnapshot — снепшот статуса Work Order для read-model.
type WorkOrderStatusSnapshot struct {
	WorkOrderID  string     `json:"work_order_id"`
	DeviceID     string     `json:"device_id,omitempty"`
	Status       string     `json:"status"`
	PreviousStatus string   `json:"previous_status,omitempty"`
	Priority     string     `json:"priority,omitempty"`
	AssigneeID   string     `json:"assignee_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	SLADeadline  *time.Time `json:"sla_deadline,omitempty"`
	SLAStatus    string     `json:"sla_status"` // on_track, at_risk, breached
	Duration     string     `json:"duration"`   // human-readable: "2h 15m"
	DurationMin  int        `json:"duration_minutes"`
}

// WorkOrderStats — агрегированная статистика Work Orders.
type WorkOrderStats struct {
	TotalWO        int            `json:"total_wo"`
	ByStatus       map[string]int `json:"by_status"`       // status → count
	ByPriority     map[string]int `json:"by_priority"`     // priority → count
	ByType         map[string]int `json:"by_type"`         // type → count
	ActiveCount    int            `json:"active_count"`
	OverdueCount   int            `json:"overdue_count"`
	AvgDurationMin float64        `json:"avg_duration_minutes"`
	CompletedToday int            `json:"completed_today"`
}

// WorkOrderProjection — CQRS read-model для Work Orders.
//
// Строится из событий:
//   - cmms.wo.created — создание WO
//   - cmms.wo.status_changed — изменение статуса
//   - cmms.wo.completed — завершение
//   - cmms.wo.assigned — назначение
type WorkOrderProjection struct {
	mu     sync.RWMutex
	logger *slog.Logger

	// Текущие статусы WO (work_order_id → snapshot)
	snapshots map[string]*WorkOrderStatusSnapshot

	// Агрегированная статистика
	stats WorkOrderStats

	// История изменений (для MTTR расчётов)
	completions []completionRecord
}

type completionRecord struct {
	WorkOrderID string
	CreatedAt   time.Time
	CompletedAt time.Time
	DurationMin float64
}

// NewWorkOrderProjection создаёт WorkOrderProjection.
func NewWorkOrderProjection(logger *slog.Logger) *WorkOrderProjection {
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkOrderProjection{
		logger:    logger.With("projection", "work_order"),
		snapshots: make(map[string]*WorkOrderStatusSnapshot),
		stats: WorkOrderStats{
			ByStatus:   make(map[string]int),
			ByPriority: make(map[string]int),
			ByType:     make(map[string]int),
		},
		completions: make([]completionRecord, 0),
	}
}

// ── Projection interface ─────────────────────────────────────────────

func (wp *WorkOrderProjection) Name() string {
	return "work_order"
}

func (wp *WorkOrderProjection) Handle(ctx context.Context, record *EventRecord) error {
	switch record.EventType {
	case "cmms.wo.created":
		return wp.handleCreated(record)
	case "cmms.wo.status_changed":
		return wp.handleStatusChanged(record)
	case "cmms.wo.completed":
		return wp.handleCompleted(record)
	case "cmms.wo.assigned":
		return wp.handleAssigned(record)
	default:
		// Не наш тип события — пропускаем
		return nil
	}
}

func (wp *WorkOrderProjection) Rebuild(ctx context.Context, store *EventStore) error {
	// Очищаем текущее состояние
	wp.mu.Lock()
	wp.snapshots = make(map[string]*WorkOrderStatusSnapshot)
	wp.stats = WorkOrderStats{
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
		ByType:     make(map[string]int),
	}
	wp.completions = make([]completionRecord, 0)
	wp.mu.Unlock()

	// Replay всех CMMS событий
	opts := RetrieveOptions{
		Source:      SourceCMMS,
		IncludeCold: true,
	}

	records, err := store.Replay(ctx, opts)
	if err != nil {
		return fmt.Errorf("work order rebuild: %w", err)
	}

	for _, record := range records {
		if err := wp.Handle(ctx, record); err != nil {
			wp.logger.Warn("rebuild handle error, skipping",
				"event_id", record.ID,
				"error", err,
			)
			continue
		}
	}

	wp.logger.Info("work order projection rebuilt",
		"total_wo", wp.stats.TotalWO,
		"active", wp.stats.ActiveCount,
	)
	return nil
}

func (wp *WorkOrderProjection) Snapshot() ([]byte, error) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return json.Marshal(struct {
		Snapshots  map[string]*WorkOrderStatusSnapshot `json:"snapshots"`
		Stats      WorkOrderStats                      `json:"stats"`
		Completions []completionRecord                  `json:"completions"`
	}{
		Snapshots:   wp.snapshots,
		Stats:       wp.stats,
		Completions: wp.completions,
	})
}

func (wp *WorkOrderProjection) Restore(data []byte) error {
	var state struct {
		Snapshots  map[string]*WorkOrderStatusSnapshot `json:"snapshots"`
		Stats      WorkOrderStats                      `json:"stats"`
		Completions []completionRecord                  `json:"completions"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("work order restore: %w", err)
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()
	wp.snapshots = state.Snapshots
	wp.stats = state.Stats
	wp.completions = state.Completions

	if wp.snapshots == nil {
		wp.snapshots = make(map[string]*WorkOrderStatusSnapshot)
	}
	if wp.stats.ByStatus == nil {
		wp.stats.ByStatus = make(map[string]int)
	}
	if wp.stats.ByPriority == nil {
		wp.stats.ByPriority = make(map[string]int)
	}
	if wp.stats.ByType == nil {
		wp.stats.ByType = make(map[string]int)
	}

	return nil
}

// ── Event handlers ────────────────────────────────────────────────────

func (wp *WorkOrderProjection) handleCreated(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		DeviceID    string `json:"device_id"`
		Title       string `json:"title"`
		Type        string `json:"type"`
		Priority    string `json:"priority"`
		AssigneeID  string `json:"assignee_id,omitempty"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal cmms.wo.created: %w", err)
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	wp.snapshots[data.WorkOrderID] = &WorkOrderStatusSnapshot{
		WorkOrderID:  data.WorkOrderID,
		DeviceID:     data.DeviceID,
		Status:       "REQUESTED",
		Priority:     data.Priority,
		AssigneeID:   data.AssigneeID,
		CreatedAt:    record.Timestamp,
		UpdatedAt:    record.Timestamp,
		SLAStatus:    "on_track",
	}

	wp.stats.TotalWO++
	wp.stats.ByStatus["REQUESTED"]++
	wp.stats.ByPriority[data.Priority]++
	wp.stats.ByType[data.Type]++
	wp.stats.ActiveCount++

	return nil
}

func (wp *WorkOrderProjection) handleStatusChanged(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		FromStatus  string `json:"from_status"`
		ToStatus    string `json:"to_status"`
		ChangedBy   string `json:"changed_by,omitempty"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal cmms.wo.status_changed: %w", err)
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	snap, exists := wp.snapshots[data.WorkOrderID]
	if !exists {
		// Создаём запись если ещё нет (возможно событие created было раньше)
		snap = &WorkOrderStatusSnapshot{
			WorkOrderID: data.WorkOrderID,
			Status:      data.FromStatus,
		}
		wp.snapshots[data.WorkOrderID] = snap
	}

	// Обновляем статистику статусов
	wp.stats.ByStatus[data.FromStatus]--
	if wp.stats.ByStatus[data.FromStatus] <= 0 {
		delete(wp.stats.ByStatus, data.FromStatus)
	}
	wp.stats.ByStatus[data.ToStatus]++

	// Обновляем снепшот
	snap.PreviousStatus = snap.Status
	snap.Status = data.ToStatus
	snap.UpdatedAt = record.Timestamp

	// Если WO завершён или отклонён — уменьшаем active count
	if data.ToStatus == "COMPLETED" || data.ToStatus == "CLOSED" || data.ToStatus == "REJECTED" {
		wp.stats.ActiveCount--
	}

	// Проверка overdue (если есть SLA deadline и WO не закрыта)
	if snap.SLADeadline != nil && data.ToStatus != "CLOSED" && data.ToStatus != "REJECTED" {
		if record.Timestamp.After(*snap.SLADeadline) {
			snap.SLAStatus = "breached"
		} else if record.Timestamp.Add(30*time.Minute).After(*snap.SLADeadline) {
			snap.SLAStatus = "at_risk"
		}
	}

	return nil
}

func (wp *WorkOrderProjection) handleCompleted(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		CompletedBy string `json:"completed_by"`
		Notes       string `json:"notes,omitempty"`
		ActualCost  float64 `json:"actual_cost,omitempty"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal cmms.wo.completed: %w", err)
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	snap, exists := wp.snapshots[data.WorkOrderID]
	if !exists {
		return fmt.Errorf("work order %s not found for completion", data.WorkOrderID)
	}

	now := record.Timestamp
	snap.CompletedAt = &now
	snap.Status = "COMPLETED"
	snap.UpdatedAt = now

	// Расчёт длительности
	duration := now.Sub(snap.CreatedAt)
	snap.DurationMin = int(duration.Minutes())
	snap.Duration = formatDuration(duration)

	// Статистика завершений
	wp.stats.CompletedToday++
	wp.stats.ByStatus["COMPLETED"]++
	wp.stats.ActiveCount--

	// Для MTTR
	wp.completions = append(wp.completions, completionRecord{
		WorkOrderID: data.WorkOrderID,
		CreatedAt:   snap.CreatedAt,
		CompletedAt: now,
		DurationMin: duration.Minutes(),
	})

	// Средняя длительность
	total := 0.0
	for _, c := range wp.completions {
		total += c.DurationMin
	}
	if len(wp.completions) > 0 {
		wp.stats.AvgDurationMin = total / float64(len(wp.completions))
	}

	return nil
}

func (wp *WorkOrderProjection) handleAssigned(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		AssigneeID  string `json:"assignee_id"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal cmms.wo.assigned: %w", err)
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	if snap, exists := wp.snapshots[data.WorkOrderID]; exists {
		snap.AssigneeID = data.AssigneeID
		snap.UpdatedAt = record.Timestamp
	}

	return nil
}

// ── Query methods ─────────────────────────────────────────────────────

// GetSnapshot возвращает снепшот Work Order по ID.
func (wp *WorkOrderProjection) GetSnapshot(workOrderID string) (*WorkOrderStatusSnapshot, bool) {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	snap, ok := wp.snapshots[workOrderID]
	return snap, ok
}

// GetStats возвращает агрегированную статистику.
func (wp *WorkOrderProjection) GetStats() WorkOrderStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	return wp.stats
}

// GetByStatus возвращает Work Orders по статусу.
func (wp *WorkOrderProjection) GetByStatus(status string) []*WorkOrderStatusSnapshot {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	result := make([]*WorkOrderStatusSnapshot, 0)
	for _, snap := range wp.snapshots {
		if snap.Status == status {
			result = append(result, snap)
		}
	}
	return result
}

// GetActive возвращает активные Work Orders (не COMPLETED/CLOSED/REJECTED).
func (wp *WorkOrderProjection) GetActive() []*WorkOrderStatusSnapshot {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	result := make([]*WorkOrderStatusSnapshot, 0)
	for _, snap := range wp.snapshots {
		switch snap.Status {
		case "COMPLETED", "CLOSED", "REJECTED", "VERIFIED":
			continue
		}
		result = append(result, snap)
	}
	return result
}

// GetOverdue возвращает Work Orders с просроченным SLA.
func (wp *WorkOrderProjection) GetOverdue() []*WorkOrderStatusSnapshot {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	result := make([]*WorkOrderStatusSnapshot, 0)
	for _, snap := range wp.snapshots {
		if snap.SLAStatus == "breached" {
			result = append(result, snap)
		}
	}
	return result
}

// GetMTTR возвращает среднее время восстановления (Mean Time To Repair).
func (wp *WorkOrderProjection) GetMTTR() float64 {
	wp.mu.RLock()
	defer wp.mu.RUnlock()

	if len(wp.completions) == 0 {
		return 0
	}
	return wp.stats.AvgDurationMin
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
