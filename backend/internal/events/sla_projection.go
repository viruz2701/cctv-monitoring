// Package events — SLAProjection (CQRS read-model для SLA compliance).
//
// Строит read-model для мониторинга SLA:
//   - Compliance rate по приоритетам
//   - Среднее время ответа и разрешения
//   - Тренды по дням/неделям
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — SLA метрики)
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
// SLAProjection — read-model для SLA compliance.
// ═══════════════════════════════════════════════════════════════════════

// SLAEntry — запись SLA compliance для одного Work Order.
type SLAEntry struct {
	WorkOrderID       string     `json:"work_order_id"`
	Priority          string     `json:"priority"`
	CreatedAt         time.Time  `json:"created_at"`
	RespondedAt       *time.Time `json:"responded_at,omitempty"`
	ResolvedAt        *time.Time `json:"resolved_at,omitempty"`
	ResponseTimeMin   float64    `json:"response_time_minutes"`
	ResolutionTimeMin float64    `json:"resolution_time_minutes"`
	ResponseTargetMin int        `json:"response_target_minutes"`
	ResolutionTargetMin int      `json:"resolution_target_minutes"`
	ResponseBreached  bool       `json:"response_breached"`
	ResolutionBreached bool      `json:"resolution_breached"`
	EscalationLevel   int        `json:"escalation_level"` // 0, 1, 2, 3
}

// SLAComplianceStats — агрегированная статистика SLA.
type SLAComplianceStats struct {
	ByPriority map[string]*PrioritySLAStats `json:"by_priority"`
	Overall    *PrioritySLAStats            `json:"overall"`
	DailyTrend []DailySLATrend              `json:"daily_trend,omitempty"`
}

// PrioritySLAStats — статистика SLA для одного приоритета.
type PrioritySLAStats struct {
	Priority            string  `json:"priority"`
	TotalWO             int     `json:"total_wo"`
	ResponseBreached    int     `json:"response_breached"`
	ResolutionBreached  int     `json:"resolution_breached"`
	CompliancePercent   float64 `json:"compliance_percent"`
	AvgResponseTimeMin  float64 `json:"avg_response_time_minutes"`
	AvgResolutionTimeMin float64 `json:"avg_resolution_time_minutes"`
}

// DailySLATrend — дневной тренд SLA.
type DailySLATrend struct {
	Date              string  `json:"date"` // "2026-06-24"
	TotalWO           int     `json:"total_wo"`
	CompliancePercent float64 `json:"compliance_percent"`
}

// SLAProjection — CQRS read-model для SLA compliance.
//
// Строится из событий:
//   - cmms.wo.created — старт SLA таймера
//   - cmms.wo.status_changed — отслеживание времени ответа
//   - cmms.wo.completed — фиксация resolution time
type SLAProjection struct {
	mu     sync.RWMutex
	logger *slog.Logger

	entries map[string]*SLAEntry       // work_order_id → SLA entry
	stats   SLAComplianceStats
	trends  map[string]*DailySLATrend  // "2026-06-24" → trend
}

// NewSLAProjection создаёт SLAProjection.
func NewSLAProjection(logger *slog.Logger) *SLAProjection {
	if logger == nil {
		logger = slog.Default()
	}
	return &SLAProjection{
		logger:  logger.With("projection", "sla"),
		entries: make(map[string]*SLAEntry),
		stats: SLAComplianceStats{
			ByPriority: make(map[string]*PrioritySLAStats),
		},
		trends: make(map[string]*DailySLATrend),
	}
}

// ── Projection interface ─────────────────────────────────────────────

func (sp *SLAProjection) Name() string {
	return "sla"
}

func (sp *SLAProjection) Handle(ctx context.Context, record *EventRecord) error {
	switch record.EventType {
	case "cmms.wo.created":
		return sp.handleCreated(record)
	case "cmms.wo.status_changed":
		return sp.handleStatusChanged(record)
	case "cmms.wo.completed":
		return sp.handleCompleted(record)
	default:
		return nil
	}
}

func (sp *SLAProjection) Rebuild(ctx context.Context, store *EventStore) error {
	sp.mu.Lock()
	sp.entries = make(map[string]*SLAEntry)
	sp.stats = SLAComplianceStats{ByPriority: make(map[string]*PrioritySLAStats)}
	sp.trends = make(map[string]*DailySLATrend)
	sp.mu.Unlock()

	opts := RetrieveOptions{
		Source:      SourceCMMS,
		IncludeCold: true,
	}

	records, err := store.Replay(ctx, opts)
	if err != nil {
		return fmt.Errorf("sla rebuild: %w", err)
	}

	for _, record := range records {
		if err := sp.Handle(ctx, record); err != nil {
			sp.logger.Warn("rebuild handle error, skipping",
				"event_id", record.ID, "error", err,
			)
			continue
		}
	}

	return nil
}

func (sp *SLAProjection) Snapshot() ([]byte, error) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return json.Marshal(struct {
		Entries map[string]*SLAEntry     `json:"entries"`
		Stats   SLAComplianceStats       `json:"stats"`
		Trends  map[string]*DailySLATrend `json:"trends"`
	}{
		Entries: sp.entries,
		Stats:   sp.stats,
		Trends:  sp.trends,
	})
}

func (sp *SLAProjection) Restore(data []byte) error {
	var state struct {
		Entries map[string]*SLAEntry      `json:"entries"`
		Stats   SLAComplianceStats        `json:"stats"`
		Trends  map[string]*DailySLATrend `json:"trends"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("sla restore: %w", err)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.entries = state.Entries
	sp.stats = state.Stats
	sp.trends = state.Trends

	if sp.entries == nil {
		sp.entries = make(map[string]*SLAEntry)
	}
	if sp.stats.ByPriority == nil {
		sp.stats.ByPriority = make(map[string]*PrioritySLAStats)
	}
	if sp.trends == nil {
		sp.trends = make(map[string]*DailySLATrend)
	}

	return nil
}

// ── Event handlers ────────────────────────────────────────────────────

func (sp *SLAProjection) handleCreated(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		Priority    string `json:"priority"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal sla.created: %w", err)
	}

	// SLA targets по приоритету
	respTarget, resTarget := slaTargets(data.Priority)

	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.entries[data.WorkOrderID] = &SLAEntry{
		WorkOrderID:        data.WorkOrderID,
		Priority:           data.Priority,
		CreatedAt:          record.Timestamp,
		ResponseTargetMin:  respTarget,
		ResolutionTargetMin: resTarget,
	}

	return nil
}

func (sp *SLAProjection) handleStatusChanged(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
		ToStatus    string `json:"to_status"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal sla.status_changed: %w", err)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	entry, exists := sp.entries[data.WorkOrderID]
	if !exists {
		return nil
	}

	// Первый переход из REQUESTED → APPROVED/OPEN = response time
	if entry.RespondedAt == nil && data.ToStatus != "REQUESTED" && data.ToStatus != "REJECTED" {
		now := record.Timestamp
		entry.RespondedAt = &now
		entry.ResponseTimeMin = now.Sub(entry.CreatedAt).Minutes()
		entry.ResponseBreached = entry.ResponseTimeMin > float64(entry.ResponseTargetMin)
	}

	// Эскалация по времени
	elapsed := record.Timestamp.Sub(entry.CreatedAt).Minutes()
	switch {
	case elapsed > float64(entry.ResolutionTargetMin)*1.5:
		entry.EscalationLevel = 3
	case elapsed > float64(entry.ResolutionTargetMin)*1.2:
		entry.EscalationLevel = 2
	case elapsed > float64(entry.ResolutionTargetMin)*0.8:
		entry.EscalationLevel = 1
	}

	return nil
}

func (sp *SLAProjection) handleCompleted(record *EventRecord) error {
	var data struct {
		WorkOrderID string `json:"work_order_id"`
	}
	if err := json.Unmarshal(record.Data, &data); err != nil {
		return fmt.Errorf("unmarshal sla.completed: %w", err)
	}

	sp.mu.Lock()
	defer sp.mu.Unlock()

	entry, exists := sp.entries[data.WorkOrderID]
	if !exists {
		return fmt.Errorf("sla entry %s not found", data.WorkOrderID)
	}

	now := record.Timestamp
	entry.ResolvedAt = &now
	entry.ResolutionTimeMin = now.Sub(entry.CreatedAt).Minutes()
	entry.ResolutionBreached = entry.ResolutionTimeMin > float64(entry.ResolutionTargetMin)

	// Обновляем статистику
	sp.updateStats(entry)

	// Дневной тренд
	dateKey := now.Format("2006-01-02")
	trend, exists := sp.trends[dateKey]
	if !exists {
		trend = &DailySLATrend{Date: dateKey}
		sp.trends[dateKey] = trend
	}
	trend.TotalWO++
	if !entry.ResponseBreached && !entry.ResolutionBreached {
		trend.CompliancePercent = float64(trend.TotalWO) // все пока compliant
	}
	// Пересчёт compliance процента
	compliant := 0
	for _, e := range sp.entries {
		if e.ResolvedAt != nil && !e.ResponseBreached && !e.ResolutionBreached {
			compliant++
		}
	}
	if trend.TotalWO > 0 {
		trend.CompliancePercent = float64(compliant) / float64(trend.TotalWO) * 100
	}

	return nil
}

// ── Stats ─────────────────────────────────────────────────────────────

func (sp *SLAProjection) updateStats(entry *SLAEntry) {
	// По приоритету
	ps, exists := sp.stats.ByPriority[entry.Priority]
	if !exists {
		ps = &PrioritySLAStats{Priority: entry.Priority}
		sp.stats.ByPriority[entry.Priority] = ps
	}

	ps.TotalWO++
	if entry.ResponseBreached {
		ps.ResponseBreached++
	}
	if entry.ResolutionBreached {
		ps.ResolutionBreached++
	}

	ps.AvgResponseTimeMin = ((ps.AvgResponseTimeMin * float64(ps.TotalWO-1)) + entry.ResponseTimeMin) / float64(ps.TotalWO)
	ps.AvgResolutionTimeMin = ((ps.AvgResolutionTimeMin * float64(ps.TotalWO-1)) + entry.ResolutionTimeMin) / float64(ps.TotalWO)

	// Simplified: each WO has response + resolution checks
	totalChecks := ps.TotalWO * 2
	totalBreaches := ps.ResponseBreached + ps.ResolutionBreached
	if totalChecks > 0 {
		ps.CompliancePercent = float64(totalChecks-totalBreaches) / float64(totalChecks) * 100
	}

	// Overall stats
	totalWO := 0
	totalRespBreach := 0
	totalResBreach := 0
	totalRespTime := 0.0
	totalResTime := 0.0

	for _, p := range sp.stats.ByPriority {
		totalWO += p.TotalWO
		totalRespBreach += p.ResponseBreached
		totalResBreach += p.ResolutionBreached
		totalRespTime += p.AvgResponseTimeMin * float64(p.TotalWO)
		totalResTime += p.AvgResolutionTimeMin * float64(p.TotalWO)
	}

	if totalWO > 0 {
		totalChecks := totalWO * 2
		totalBreaches := totalRespBreach + totalResBreach
		if sp.stats.Overall == nil {
			sp.stats.Overall = &PrioritySLAStats{Priority: "overall"}
		}
		sp.stats.Overall.TotalWO = totalWO
		sp.stats.Overall.ResponseBreached = totalRespBreach
		sp.stats.Overall.ResolutionBreached = totalResBreach
		sp.stats.Overall.CompliancePercent = float64(totalChecks-totalBreaches) / float64(totalChecks) * 100
		sp.stats.Overall.AvgResponseTimeMin = totalRespTime / float64(totalWO)
		sp.stats.Overall.AvgResolutionTimeMin = totalResTime / float64(totalWO)
	}
}

// ── Query methods ─────────────────────────────────────────────────────

// GetSLAEntry возвращает SLA запись для Work Order.
func (sp *SLAProjection) GetSLAEntry(workOrderID string) (*SLAEntry, bool) {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	e, ok := sp.entries[workOrderID]
	return e, ok
}

// GetStats возвращает агрегированную SLA статистику.
func (sp *SLAProjection) GetStats() SLAComplianceStats {
	sp.mu.RLock()
	defer sp.mu.RUnlock()
	return sp.stats
}

// GetBreached возвращает Work Orders с нарушением SLA.
func (sp *SLAProjection) GetBreached() []*SLAEntry {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	result := make([]*SLAEntry, 0)
	for _, e := range sp.entries {
		if e.ResponseBreached || e.ResolutionBreached {
			result = append(result, e)
		}
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Targets by priority
// ═══════════════════════════════════════════════════════════════════════

func slaTargets(priority string) (responseMinutes, resolutionMinutes int) {
	switch priority {
	case "critical":
		return 15, 60   // 15min response, 1h resolution
	case "high":
		return 30, 240  // 30min response, 4h resolution
	case "medium":
		return 60, 480  // 1h response, 8h resolution
	case "low":
		return 120, 960 // 2h response, 16h resolution
	default:
		return 60, 480
	}
}
