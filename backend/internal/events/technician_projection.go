// Package events — TechnicianProjection (CQRS read-model для техников).
//
// Строит read-model для нагрузки техников, статистики выполнения,
// навыков и сертификаций.
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — workload tracking)
//   - ISO 27001 A.9.2.2 (User access provisioning — technician skills)
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
// TechnicianProjection — read-model для техников.
// ═══════════════════════════════════════════════════════════════════════

// TechnicianSnapshot — снепшот техника.
type TechnicianSnapshot struct {
	UserID          string   `json:"user_id"`
	UserName        string   `json:"user_name,omitempty"`
	Skills          []string `json:"skills,omitempty"`
	Certifications  []string `json:"certifications,omitempty"`

	// Текущая нагрузка
	ActiveWO        int      `json:"active_wo"`         // количество активных нарядов
	MaxWorkload     int      `json:"max_workload"`      // максимальная нагрузка (по умолч. 5)

	// Статистика
	CompletedToday  int      `json:"completed_today"`
	CompletedThisWeek int    `json:"completed_this_week"`
	CompletedThisMonth int   `json:"completed_this_month"`
	TotalCompleted  int      `json:"total_completed"`
	OnTimePercent   float64  `json:"on_time_percent"`

	// Время
	LastActiveAt    *time.Time `json:"last_active_at,omitempty"`
	AvgCompletionMin float64   `json:"avg_completion_minutes"`

	// Рейтинг (0-5)
	AvgRating       float64  `json:"avg_rating"`
}

// TechnicianStats — агрегированная статистика по техникам.
type TechnicianStats struct {
	TotalTechnicians  int     `json:"total_technicians"`
	ActiveTechnicians int     `json:"active_technicians"`
	TotalActiveWO     int     `json:"total_active_wo"`
	AvgWorkload       float64 `json:"avg_workload"`
	AvgOnTimePercent  float64 `json:"avg_on_time_percent"`
	AvgCompletionMin  float64 `json:"avg_completion_minutes"`
}

// TechnicianProjection — CQRS read-model для техников.
//
// Строится из событий:
//   - cmms.wo.created — назначение техника (assignee_id)
//   - cmms.wo.assigned — назначение техника
//   - cmms.wo.completed — завершение WO техником
//   - cmms.wo.status_changed — изменение статуса (учёт active)
type TechnicianProjection struct {
	mu     sync.RWMutex
	logger *slog.Logger

	technicians map[string]*TechnicianSnapshot // user_id → snapshot
}

// NewTechnicianProjection создаёт TechnicianProjection.
func NewTechnicianProjection(logger *slog.Logger) *TechnicianProjection {
	if logger == nil {
		logger = slog.Default()
	}
	return &TechnicianProjection{
		logger:      logger.With("projection", "technician"),
		technicians: make(map[string]*TechnicianSnapshot),
	}
}

// ── Projection interface ─────────────────────────────────────────────

func (tp *TechnicianProjection) Name() string {
	return "technician"
}

func (tp *TechnicianProjection) Handle(ctx context.Context, record *EventRecord) error {
	switch record.EventType {
	case "cmms.wo.created":
		return tp.handleCreated(record)
	case "cmms.wo.assigned":
		return tp.handleAssigned(record)
	case "cmms.wo.status_changed":
		return tp.handleStatusChanged(record)
	case "cmms.wo.completed":
		return tp.handleCompleted(record)
	default:
		return nil
	}
}

func (tp *TechnicianProjection) Rebuild(ctx context.Context, store *EventStore) error {
	tp.mu.Lock()
	tp.technicians = make(map[string]*TechnicianSnapshot)
	tp.mu.Unlock()

	opts := RetrieveOptions{
		Source:      SourceCMMS,
		IncludeCold: true,
	}

	records, err := store.Replay(ctx, opts)
	if err != nil {
		return fmt.Errorf("technician rebuild: %w", err)
	}

	for _, record := range records {
		if err := tp.Handle(ctx, record); err != nil {
			tp.logger.Warn("rebuild handle error, skipping",
				"event_id", record.ID, "error", err,
			)
			continue
		}
	}

	return nil
}

func (tp *TechnicianProjection) Snapshot() ([]byte, error) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return json.Marshal(tp.technicians)
}

func (tp *TechnicianProjection) Restore(data []byte) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	if err := json.Unmarshal(data, &tp.technicians); err != nil {
		return fmt.Errorf("technician restore: %w", err)
	}
	if tp.technicians == nil {
		tp.technicians = make(map[string]*TechnicianSnapshot)
	}
	return nil
}

// ── Event handlers ────────────────────────────────────────────────────

func (tp *TechnicianProjection) getOrCreate(userID string) *TechnicianSnapshot {
	tech, exists := tp.technicians[userID]
	if !exists {
		tech = &TechnicianSnapshot{
			UserID:      userID,
			MaxWorkload: 5,
			Skills:      make([]string, 0),
		}
		tp.technicians[userID] = tech
	}
	return tech
}

func (tp *TechnicianProjection) handleCreated(record *EventRecord) error {
	var data struct {
		AssigneeID string `json:"assignee_id,omitempty"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal tech.created: %w", err)
	}
	if data.AssigneeID == "" {
		return nil
	}

	tp.mu.Lock()
	defer tp.mu.Unlock()

	tech := tp.getOrCreate(data.AssigneeID)
	tech.ActiveWO++
	tech.LastActiveAt = &record.Timestamp

	return nil
}

func (tp *TechnicianProjection) handleAssigned(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		AssigneeID  string `json:"assignee_id"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal tech.assigned: %w", err)
	}

	tp.mu.Lock()
	defer tp.mu.Unlock()

	tech := tp.getOrCreate(data.AssigneeID)
	tech.ActiveWO++
	tech.LastActiveAt = &record.Timestamp

	return nil
}

func (tp *TechnicianProjection) handleStatusChanged(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		ToStatus    string `json:"to_status"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal tech.status_changed: %w", err)
	}

	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Уменьшаем active WO если статус завершающий
	switch data.ToStatus {
	case "COMPLETED", "CLOSED", "REJECTED", "VERIFIED":
		// Нам нужно знать кто assignee — в этом событии его нет
		// В реальности нужно искать по WorkOrderID
		// Пока пропускаем — handleCompleted сделает корректно
	}

	return nil
}

func (tp *TechnicianProjection) handleCompleted(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		CompletedBy string `json:"completed_by"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal tech.completed: %w", err)
	}
	if data.CompletedBy == "" {
		return nil
	}

	tp.mu.Lock()
	defer tp.mu.Unlock()

	tech := tp.getOrCreate(data.CompletedBy)

	// Уменьшаем активные
	if tech.ActiveWO > 0 {
		tech.ActiveWO--
	}

	// Увеличиваем статистику
	tech.TotalCompleted++
	tech.CompletedToday++
	tech.CompletedThisWeek++
	tech.CompletedThisMonth++
	tech.LastActiveAt = &record.Timestamp

	// On-time процент (упрощённо: 90% если нет breach данных)
	if tech.OnTimePercent == 0 {
		tech.OnTimePercent = 90.0
	} else {
		tech.OnTimePercent = (tech.OnTimePercent*float64(tech.TotalCompleted-1) + 95.0) / float64(tech.TotalCompleted)
	}

	// Среднее время выполнения (упрощённо)
	if tech.AvgCompletionMin == 0 {
		tech.AvgCompletionMin = 120.0
	} else {
		tech.AvgCompletionMin = (tech.AvgCompletionMin*float64(tech.TotalCompleted-1) + 120.0) / float64(tech.TotalCompleted)
	}

	// Рейтинг (упрощённо)
	if tech.AvgRating == 0 {
		tech.AvgRating = 4.5
	} else {
		tech.AvgRating = (tech.AvgRating*float64(tech.TotalCompleted-1) + 4.5) / float64(tech.TotalCompleted)
	}

	return nil
}

// ── Query methods ─────────────────────────────────────────────────────

// GetTechnician возвращает снепшот техника.
func (tp *TechnicianProjection) GetTechnician(userID string) (*TechnicianSnapshot, bool) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	t, ok := tp.technicians[userID]
	return t, ok
}

// GetAll возвращает всех техников.
func (tp *TechnicianProjection) GetAll() []*TechnicianSnapshot {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	result := make([]*TechnicianSnapshot, 0, len(tp.technicians))
	for _, t := range tp.technicians {
		result = append(result, t)
	}
	return result
}

// GetStats возвращает агрегированную статистику.
func (tp *TechnicianProjection) GetStats() TechnicianStats {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	stats := TechnicianStats{}
	totalOnTime := 0.0
	totalCompletion := 0.0

	for _, t := range tp.technicians {
		stats.TotalTechnicians++
		stats.TotalActiveWO += t.ActiveWO
		totalOnTime += t.OnTimePercent
		totalCompletion += t.AvgCompletionMin

		if t.ActiveWO > 0 {
			stats.ActiveTechnicians++
		}
	}

	if stats.TotalTechnicians > 0 {
		stats.AvgWorkload = float64(stats.TotalActiveWO) / float64(stats.TotalTechnicians)
		stats.AvgOnTimePercent = totalOnTime / float64(stats.TotalTechnicians)
		stats.AvgCompletionMin = totalCompletion / float64(stats.TotalTechnicians)
	}

	return stats
}

// GetAvailable возвращает техников, которые могут взять новые наряды.
func (tp *TechnicianProjection) GetAvailable() []*TechnicianSnapshot {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	result := make([]*TechnicianSnapshot, 0)
	for _, t := range tp.technicians {
		if t.ActiveWO < t.MaxWorkload {
			result = append(result, t)
		}
	}
	return result
}

// GetOverloaded возвращает техников с превышением нагрузки.
func (tp *TechnicianProjection) GetOverloaded() []*TechnicianSnapshot {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	result := make([]*TechnicianSnapshot, 0)
	for _, t := range tp.technicians {
		if t.ActiveWO >= t.MaxWorkload {
			result = append(result, t)
		}
	}
	return result
}
