// Package downtime — Asset Downtime Tracking (AN-10.3.x).
//
// AN-10.3.1: AssetDowntime entity
// AN-10.3.2: Auto-downtime при AlarmEvent
// AN-10.3.3: Downtime Cost calculation
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - IEC 62443 SR 7.1 (Resource availability)
package downtime

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// AN-10.3.1: AssetDowntime entity
// ═══════════════════════════════════════════════════════════════════════

type DowntimeReason string

const (
	ReasonHardware  DowntimeReason = "hardware_failure"
	ReasonNetwork   DowntimeReason = "network_outage"
	ReasonPower     DowntimeReason = "power_outage"
	ReasonSoftware  DowntimeReason = "software_crash"
	ReasonMaintenance DowntimeReason = "maintenance"
	ReasonUnknown   DowntimeReason = "unknown"
)

type AssetDowntime struct {
	ID            string         `json:"id" db:"id"`
	DeviceID      string         `json:"device_id" db:"device_id" validate:"required"`
	DeviceName    string         `json:"device_name,omitempty" db:"-"`
	DeviceType    string         `json:"device_type,omitempty" db:"-"`
	SiteID        string         `json:"site_id,omitempty" db:"site_id"`
	StartedAt     time.Time      `json:"started_at" db:"started_at"`
	EndedAt       *time.Time     `json:"ended_at,omitempty" db:"ended_at"`
	DurationMin   int            `json:"duration_minutes" db:"duration_minutes"` // calculated
	Reason        DowntimeReason `json:"reason" db:"reason"`
	Description   string         `json:"description,omitempty" db:"description"`
	AlarmID       string         `json:"alarm_id,omitempty" db:"alarm_id"`          // trigger alarm
	WorkOrderID   string         `json:"work_order_id,omitempty" db:"work_order_id"` // related WO

	// Cost (AN-10.3.3)
	HourlyCost    float64        `json:"hourly_cost" db:"hourly_cost"`       // $/hour
	TotalCost     float64        `json:"total_cost" db:"total_cost"`         // calculated
	LostRevenue   float64        `json:"lost_revenue,omitempty" db:"lost_revenue"` // estimated lost revenue

	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
}

// IsActive возвращает true если downtime ещё не завершён.
func (d *AssetDowntime) IsActive() bool {
	return d.EndedAt == nil
}

// CalculateDuration пересчитывает длительность простоя.
func (d *AssetDowntime) CalculateDuration() {
	if d.EndedAt != nil {
		d.DurationMin = int(d.EndedAt.Sub(d.StartedAt).Minutes())
	}
}

// CalculateTotalCost пересчитывает общую стоимость простоя (AN-10.3.3).
func (d *AssetDowntime) CalculateTotalCost() {
	if d.HourlyCost > 0 && d.DurationMin > 0 {
		hours := float64(d.DurationMin) / 60.0
		d.TotalCost = math.Round(hours*d.HourlyCost*100) / 100
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Cost configuration per device type
// ═══════════════════════════════════════════════════════════════════════

// DefaultHourlyCosts — стоимость простоя по типам устройств ($/час).
var DefaultHourlyCosts = map[string]float64{
	"camera": 15.0,   // $15/hr per camera
	"nvr":    50.0,   // $50/hr
	"dvr":    40.0,   // $40/hr
	"switch": 30.0,   // $30/hr
	"server": 100.0,  // $100/hr
	"encoder": 25.0,  // $25/hr
	"ups":    20.0,   // $20/hr
}

// ═══════════════════════════════════════════════════════════════════════
// AN-10.3.2 + AN-10.3.3: Tracker
// ═══════════════════════════════════════════════════════════════════════

// Tracker отслеживает простои устройств.
type Tracker struct {
	mu        sync.RWMutex
	logger    *slog.Logger
	active    map[string]*AssetDowntime // device_id → active downtime
	completed []*AssetDowntime          // completed downtimes
	maxLog    int
}

// NewTracker создаёт Tracker для отслеживания простоев.
func NewTracker(logger *slog.Logger) *Tracker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Tracker{
		logger:   logger.With("component", "downtime"),
		active:   make(map[string]*AssetDowntime),
		completed: make([]*AssetDowntime, 0),
		maxLog:   10000,
	}
}

// StartDowntime начинает отслеживание простоя устройства (AN-10.3.2).
//
// Автоматически вызывается при:
//   - offline alarm от устройства
//   - heartbeat timeout (reaper)
//   - manual
func (t *Tracker) StartDowntime(deviceID, deviceName, deviceType, siteID string, reason DowntimeReason, desc string) *AssetDowntime {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Если уже есть активный downtime — не дублируем
	if existing, ok := t.active[deviceID]; ok {
		return existing
	}

	hourlyCost := DefaultHourlyCosts[deviceType]

	downtime := &AssetDowntime{
		DeviceID:    deviceID,
		DeviceName:  deviceName,
		DeviceType:  deviceType,
		SiteID:      siteID,
		StartedAt:   time.Now().UTC(),
		Reason:      reason,
		Description: desc,
		HourlyCost:  hourlyCost,
	}

	t.active[deviceID] = downtime

	t.logger.Info("downtime started",
		"device_id", deviceID,
		"device_name", deviceName,
		"reason", reason,
		"hourly_cost", hourlyCost,
	)

	return downtime
}

// EndDowntime завершает отслеживание простоя.
//
// Автоматически вызывается при:
//   - online alarm от устройства
//   - восстановлении heartbeat
//   - manual
func (t *Tracker) EndDowntime(deviceID string, workOrderID string) *AssetDowntime {
	t.mu.Lock()
	defer t.mu.Unlock()

	downtime, ok := t.active[deviceID]
	if !ok {
		return nil
	}

	now := time.Now().UTC()
	downtime.EndedAt = &now
	downtime.WorkOrderID = workOrderID
	downtime.CalculateDuration()
	downtime.CalculateTotalCost()

	// Перемещаем в completed
	delete(t.active, deviceID)
	t.completed = append(t.completed, downtime)

	// Ограничиваем размер лога
	if len(t.completed) > t.maxLog {
		t.completed = t.completed[len(t.completed)-t.maxLog:]
	}

	t.logger.Info("downtime ended",
		"device_id", deviceID,
		"duration_min", downtime.DurationMin,
		"total_cost", downtime.TotalCost,
	)

	return downtime
}

// GetActive возвращает все активные простои.
func (t *Tracker) GetActive() []*AssetDowntime {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*AssetDowntime, 0, len(t.active))
	for _, d := range t.active {
		result = append(result, d)
	}
	return result
}

// GetByDevice возвращает историю простоев для устройства.
func (t *Tracker) GetByDevice(deviceID string) []*AssetDowntime {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]*AssetDowntime, 0)

	// Check active
	if d, ok := t.active[deviceID]; ok {
		result = append(result, d)
	}

	// Check completed
	for _, d := range t.completed {
		if d.DeviceID == deviceID {
			result = append(result, d)
		}
	}

	return result
}

// GetStats возвращает статистику по простоям.
type DowntimeStats struct {
	TotalDowntimes    int              `json:"total_downtimes"`
	ActiveDowntimes   int              `json:"active_downtimes"`
	TotalDurationMin  int              `json:"total_duration_minutes"`
	TotalCost         float64          `json:"total_cost"`
	AvgDurationMin    float64          `json:"avg_duration_minutes"`
	MTTR              float64          `json:"mttr_minutes"` // Mean Time To Repair
	ByDevice          map[string]int   `json:"by_device,omitempty"`
	ByReason          map[string]int   `json:"by_reason,omitempty"`
}

func (t *Tracker) GetStats() DowntimeStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := DowntimeStats{
		ActiveDowntimes: len(t.active),
		ByDevice:        make(map[string]int),
		ByReason:        make(map[string]int),
	}

	totalDuration := 0
	completedCount := 0

	// Active downtimes
	for _, d := range t.active {
		stats.TotalDowntimes++
		stats.ByDevice[d.DeviceID]++
		stats.ByReason[string(d.Reason)]++
	}

	// Completed downtimes
	for _, d := range t.completed {
		stats.TotalDowntimes++
		totalDuration += d.DurationMin
		stats.TotalCost += d.TotalCost
		stats.ByDevice[d.DeviceID]++
		stats.ByReason[string(d.Reason)]++
		if d.DurationMin > 0 {
			completedCount++
		}
	}

	if completedCount > 0 {
		stats.AvgDurationMin = float64(totalDuration) / float64(completedCount)
		stats.MTTR = stats.AvgDurationMin
	}

	stats.TotalCost = math.Round(stats.TotalCost*100) / 100

	return stats
}

// ═══════════════════════════════════════════════════════════════════════
// AN-10.3.3: Cost calculation helpers
// ═══════════════════════════════════════════════════════════════════════

// CalculateTCO рассчитывает Total Cost of Ownership для устройства.
// Formula: Purchase + (Labor + Parts + Downtime) over period
type TCO struct {
	DeviceID        string  `json:"device_id"`
	DeviceName      string  `json:"device_name"`
	PurchaseCost    float64 `json:"purchase_cost"`
	LaborCost       float64 `json:"labor_cost"`
	PartsCost       float64 `json:"parts_cost"`
	DowntimeCost    float64 `json:"downtime_cost"`
	TotalCost       float64 `json:"total_cost"`
	PeriodMonths    int     `json:"period_months"`
}

// CalculateTCO рассчитывает TCO.
func CalculateTCO(deviceID, deviceName string, purchaseCost, laborCost, partsCost float64, downtimeMinutes int, hourlyRate float64, periodMonths int) *TCO {
	downtimeHours := float64(downtimeMinutes) / 60.0
	downtimeCost := math.Round(downtimeHours*hourlyRate*100) / 100
	total := purchaseCost + laborCost + partsCost + downtimeCost

	return &TCO{
		DeviceID:     deviceID,
		DeviceName:   deviceName,
		PurchaseCost: purchaseCost,
		LaborCost:    laborCost,
		PartsCost:    partsCost,
		DowntimeCost: downtimeCost,
		TotalCost:    total,
		PeriodMonths: periodMonths,
	}
}

// Summary возвращает текстовую сводку по downtime.
func (d *AssetDowntime) Summary() string {
	if d.EndedAt == nil {
		return fmt.Sprintf("%s (%s): active since %s (running %d min, $%.2f/hr)",
			d.DeviceName, d.Reason,
			d.StartedAt.Format("15:04"), int(time.Since(d.StartedAt).Minutes()),
			d.HourlyCost,
		)
	}
	return fmt.Sprintf("%s (%s): %d min downtime, cost $%.2f",
		d.DeviceName, d.Reason, d.DurationMin, d.TotalCost)
}
