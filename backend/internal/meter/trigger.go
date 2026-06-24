// Package meter — WorkOrderMeterTrigger.
//
// AH-5.3.3: Правило вида "CPU > 85°C за 10min → Создать Preventive WO".
//
// Использует cel-go (Apache 2.0) для evaluation условий.
// Временно — встроенный evaluation, cel-go будет подключён в WF-9.x.
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability)
//   - ISO 27001 A.12.6.1 (Capacity management)
package meter

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// AH-5.3.3: WorkOrderMeterTrigger
// ═══════════════════════════════════════════════════════════════════════

// TriggerConditionType — тип условия триггера.
type TriggerConditionType string

const (
	CondGreaterThan     TriggerConditionType = "gt"  // значение > порог
	CondLessThan        TriggerConditionType = "lt"  // значение < порог
	CondGreaterEqual    TriggerConditionType = "gte" // значение >= порог
	CondLessEqual       TriggerConditionType = "lte" // значение <= порог
	CondEqual           TriggerConditionType = "eq"  // значение == порог
	CondAvgOverPeriod   TriggerConditionType = "avg_over_period" // среднее за период > порог
	CondTrendUp         TriggerConditionType = "trend_up"       // тренд растёт
	CondTrendDown       TriggerConditionType = "trend_down"     // тренд падает
)

// WorkOrderMeterTrigger — правило: условие по метрике → создание WO.
//
// Пример:
//
//	MeterKind: cpu_temp
//	Condition: gt
//	Threshold: 85
//	Duration:  10m (условие должно выполняться 10 минут)
//	Action:
//	  WorkOrderType: preventive
//	  Priority:      high
//	  TitleTemplate: "CPU temperature high on {device_name}"
//	  Description:   "CPU temperature on {device_name} is {value}°C (threshold: 85°C)"
type WorkOrderMeterTrigger struct {
	ID        string              `json:"id" db:"id"`
	Name      string              `json:"name" db:"name" validate:"required,max=100"`
	Enabled   bool                `json:"enabled" db:"enabled"`
	MeterKind MeterKind           `json:"meter_kind" db:"meter_kind" validate:"required"`
	Condition TriggerConditionType `json:"condition" db:"condition" validate:"required"`
	Threshold float64             `json:"threshold" db:"threshold"`

	// Длительность: условие должно выполняться столько времени
	DurationSeconds int `json:"duration_seconds" db:"duration_seconds" validate:"min=0"`

	// Фильтр по устройствам (пусто = все устройства с этим MeterKind)
	DeviceIDs []string `json:"device_ids,omitempty" db:"device_ids"`

	// Cooldown: не создавать повторный WO раньше этого времени
	CooldownMinutes int `json:"cooldown_minutes" db:"cooldown_minutes"`

	// Действие: создание WO
	Action TriggerAction `json:"action" db:"action"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TriggerAction — действие при срабатывании триггера.
type TriggerAction struct {
	WorkOrderType  string `json:"work_order_type"`  // preventive, corrective, emergency
	Priority       string `json:"priority"`          // critical, high, medium, low
	TitleTemplate  string `json:"title_template"`
	DescTemplate   string `json:"desc_template"`
	AssigneeID     string `json:"assignee_id,omitempty"`
	AutoApprove    bool   `json:"auto_approve"`      // создавать WO сразу (без approval)
}

// DefaultTriggers возвращает стандартные триггеры для CCTV.
func DefaultTriggers() []*WorkOrderMeterTrigger {
	return []*WorkOrderMeterTrigger{
		{
			Name:            "CPU Overheating",
			Enabled:         true,
			MeterKind:       MeterCPUTemp,
			Condition:       CondGreaterThan,
			Threshold:       85,
			DurationSeconds: 600, // 10 минут
			CooldownMinutes: 120, // 2 часа
			Action: TriggerAction{
				WorkOrderType: "preventive",
				Priority:      "high",
				TitleTemplate:  "CPU overheating on {device_name}",
				DescTemplate:   "CPU temperature on {device_name} is {value}°C (threshold: 85°C). Needs immediate inspection.",
				AutoApprove:    true,
			},
		},
		{
			Name:            "High Packet Loss",
			Enabled:         true,
			MeterKind:       MeterPacketLoss,
			Condition:       CondGreaterThan,
			Threshold:       5,
			DurationSeconds: 300, // 5 минут
			CooldownMinutes: 60,
			Action: TriggerAction{
				WorkOrderType: "corrective",
				Priority:      "high",
				TitleTemplate:  "High packet loss on {device_name}",
				DescTemplate:   "Packet loss on {device_name} is {value}% (threshold: 5%). Check network connection.",
				AutoApprove:    true,
			},
		},
		{
			Name:            "Low Frame Rate",
			Enabled:         true,
			MeterKind:       MeterFPS,
			Condition:       CondLessThan,
			Threshold:       10,
			DurationSeconds: 600,
			CooldownMinutes: 120,
			Action: TriggerAction{
				WorkOrderType: "corrective",
				Priority:      "medium",
				TitleTemplate:  "Low frame rate on {device_name}",
				DescTemplate:   "Frame rate on {device_name} is {value} FPS (threshold: 10 FPS). May indicate network or hardware issue.",
				AutoApprove:    false,
			},
		},
		{
			Name:            "NVR Disk Almost Full",
			Enabled:         true,
			MeterKind:       MeterDiskUsage,
			Condition:       CondGreaterThan,
			Threshold:       90,
			DurationSeconds: 1800, // 30 минут
			CooldownMinutes: 1440, // 24 часа
			Action: TriggerAction{
				WorkOrderType: "preventive",
				Priority:      "high",
				TitleTemplate:  "NVR disk almost full on {device_name}",
				DescTemplate:   "NVR disk usage is {value}% (threshold: 90%). Cleanup or expand storage needed.",
				AutoApprove:    true,
			},
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Trigger Engine
// ═══════════════════════════════════════════════════════════════════════

// TriggerEngine — движок проверки условий триггеров.
type TriggerEngine struct {
	mu       sync.RWMutex
	logger   *slog.Logger
	triggers []*WorkOrderMeterTrigger

	// Cooldown tracking: trigger_id → last_fired_at
	cooldowns map[string]time.Time

	// История значений для avg_over_period: meter_id → [{time, value}]
	history map[string][]Reading
	historyMax int
}

// NewTriggerEngine создаёт TriggerEngine.
func NewTriggerEngine(logger *slog.Logger) *TriggerEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &TriggerEngine{
		logger:      logger.With("component", "meter-trigger"),
		cooldowns:   make(map[string]time.Time),
		history:     make(map[string][]Reading),
		historyMax:  100, // хранить до 100 последних значений
	}
}

// SetTriggers устанавливает список активных триггеров.
func (te *TriggerEngine) SetTriggers(triggers []*WorkOrderMeterTrigger) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.triggers = triggers
}

// AddReading добавляет новое показание и проверяет триггеры.
//
// Возвращает список триггеров, которые сработали.
func (te *TriggerEngine) AddReading(reading Reading, deviceName string) []FiredTrigger {
	te.mu.Lock()

	// Добавляем в историю
	key := fmt.Sprintf("%s:%s", reading.MeterID, reading.Kind)
	te.history[key] = append(te.history[key], reading)
	if len(te.history[key]) > te.historyMax {
		te.history[key] = te.history[key][len(te.history[key])-te.historyMax:]
	}

	// Достаём историю для evaluateCondition (уже под write lock)
	history := te.history[key]

	// Проверяем триггеры
	fired := make([]FiredTrigger, 0)
	for _, t := range te.triggers {
		if !t.Enabled {
			continue
		}
		if string(t.MeterKind) != string(reading.Kind) {
			continue
		}

		// Фильтр по устройствам
		if len(t.DeviceIDs) > 0 {
			matched := false
			for _, did := range t.DeviceIDs {
				if did == reading.DeviceID {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// Cooldown check
		if lastFired, ok := te.cooldowns[t.ID]; ok {
			if time.Since(lastFired).Minutes() < float64(t.CooldownMinutes) {
				continue
			}
		}

		// Condition check (передаём history напрямую, без повторного взятия блокировки)
		if te.evaluateConditionWithHistory(t, reading, deviceName, history) {
			fired = append(fired, FiredTrigger{
				Trigger:     *t,
				Reading:     reading,
				DeviceName:  deviceName,
				FiredAt:     time.Now(),
			})
			te.cooldowns[t.ID] = time.Now()
		}
	}
	te.mu.Unlock()

	return fired
}

// evaluateCondition проверяет условие триггера.
func (te *TriggerEngine) evaluateCondition(t *WorkOrderMeterTrigger, r Reading, deviceName string) bool {
	readings := te.getHistory(r.MeterID, r.Kind)

	switch t.Condition {
	case CondGreaterThan:
		if t.DurationSeconds <= 0 {
			return r.Value > t.Threshold
		}
		return te.allAboveThreshold(readings, t.Threshold, t.DurationSeconds)

	case CondLessThan:
		if t.DurationSeconds <= 0 {
			return r.Value < t.Threshold
		}
		return te.allBelowThreshold(readings, t.Threshold, t.DurationSeconds)

	case CondGreaterEqual:
		return r.Value >= t.Threshold

	case CondLessEqual:
		return r.Value <= t.Threshold

	case CondEqual:
		return r.Value == t.Threshold

	case CondAvgOverPeriod:
		avg := te.averageOverPeriod(readings, t.DurationSeconds)
		return avg > t.Threshold

	case CondTrendUp:
		return te.isTrendingUp(readings, 5) // последние 5 значений

	case CondTrendDown:
		return te.isTrendingDown(readings, 5)
	}

	return false
}

// evaluateConditionWithHistory проверяет условие триггера с переданной историей (без блокировки).
func (te *TriggerEngine) evaluateConditionWithHistory(t *WorkOrderMeterTrigger, r Reading, deviceName string, readings []Reading) bool {
	switch t.Condition {
	case CondGreaterThan:
		if t.DurationSeconds <= 0 {
			return r.Value > t.Threshold
		}
		return te.allAboveThreshold(readings, t.Threshold, t.DurationSeconds)

	case CondLessThan:
		if t.DurationSeconds <= 0 {
			return r.Value < t.Threshold
		}
		return te.allBelowThreshold(readings, t.Threshold, t.DurationSeconds)

	case CondGreaterEqual:
		return r.Value >= t.Threshold

	case CondLessEqual:
		return r.Value <= t.Threshold

	case CondEqual:
		return r.Value == t.Threshold

	case CondAvgOverPeriod:
		avg := te.averageOverPeriod(readings, t.DurationSeconds)
		return avg > t.Threshold

	case CondTrendUp:
		return te.isTrendingUp(readings, 5)

	case CondTrendDown:
		return te.isTrendingDown(readings, 5)
	}

	return false
}

// getHistory возвращает историю показаний для метрики.
// Внимание: требует внешней блокировки te.mu (RLock или Lock).
func (te *TriggerEngine) getHistory(meterID string, kind MeterKind) []Reading {
	key := fmt.Sprintf("%s:%s", meterID, kind)
	if h, ok := te.history[key]; ok {
		return h
	}
	return nil
}

// allAboveThreshold проверяет, что все показания за период выше порога.
func (te *TriggerEngine) allAboveThreshold(readings []Reading, threshold float64, durationSec int) bool {
	if len(readings) == 0 {
		return false
	}

	cutoff := time.Now().Add(-time.Duration(durationSec) * time.Second)
	count := 0
	total := 0

	for _, r := range readings {
		if r.Time.After(cutoff) {
			total++
			if r.Value > threshold {
				count++
			}
		}
	}

	if total == 0 {
		return false
	}
	// 80%+ показаний должны быть выше порога
	return float64(count)/float64(total) >= 0.8
}

// allBelowThreshold проверяет, что все показания за период ниже порога.
func (te *TriggerEngine) allBelowThreshold(readings []Reading, threshold float64, durationSec int) bool {
	if len(readings) == 0 {
		return false
	}

	cutoff := time.Now().Add(-time.Duration(durationSec) * time.Second)
	count := 0
	total := 0

	for _, r := range readings {
		if r.Time.After(cutoff) {
			total++
			if r.Value < threshold {
				count++
			}
		}
	}

	if total == 0 {
		return false
	}
	return float64(count)/float64(total) >= 0.8
}

// averageOverPeriod вычисляет среднее за период.
func (te *TriggerEngine) averageOverPeriod(readings []Reading, durationSec int) float64 {
	if len(readings) == 0 {
		return 0
	}

	cutoff := time.Now().Add(-time.Duration(durationSec) * time.Second)
	sum := 0.0
	count := 0

	for _, r := range readings {
		if r.Time.After(cutoff) {
			sum += r.Value
			count++
		}
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// isTrendingUp проверяет восходящий тренд (последние N значений).
func (te *TriggerEngine) isTrendingUp(readings []Reading, n int) bool {
	if len(readings) < n {
		return false
	}

	recent := readings[len(readings)-n:]
	upCount := 0
	for i := 1; i < len(recent); i++ {
		if recent[i].Value > recent[i-1].Value {
			upCount++
		}
	}
	return float64(upCount)/float64(len(recent)-1) >= 0.6
}

// isTrendingDown проверяет нисходящий тренд (последние N значений).
func (te *TriggerEngine) isTrendingDown(readings []Reading, n int) bool {
	if len(readings) < n {
		return false
	}

	recent := readings[len(readings)-n:]
	downCount := 0
	for i := 1; i < len(recent); i++ {
		if recent[i].Value < recent[i-1].Value {
			downCount++
		}
	}
	return float64(downCount)/float64(len(recent)-1) >= 0.6
}

// ═══════════════════════════════════════════════════════════════════════
// FiredTrigger — результат срабатывания триггера.
// ═══════════════════════════════════════════════════════════════════════

// FiredTrigger — информация о сработавшем триггере.
type FiredTrigger struct {
	Trigger    WorkOrderMeterTrigger `json:"trigger"`
	Reading    Reading               `json:"reading"`
	DeviceName string                `json:"device_name"`
	FiredAt    time.Time             `json:"fired_at"`
}

// GenerateWOTitle генерирует заголовок WO из шаблона.
func (ft *FiredTrigger) GenerateWOTitle() string {
	return fillTemplate(ft.Trigger.Action.TitleTemplate, ft)
}

// GenerateWODescription генерирует описание WO из шаблона.
func (ft *FiredTrigger) GenerateWODescription() string {
	return fillTemplate(ft.Trigger.Action.DescTemplate, ft)
}

func fillTemplate(tpl string, ft *FiredTrigger) string {
	result := tpl
	replacements := map[string]string{
		"{device_name}": ft.DeviceName,
		"{device_id}":   ft.Reading.DeviceID,
		"{value}":       fmt.Sprintf("%.1f", ft.Reading.Value),
		"{threshold}":   fmt.Sprintf("%.1f", ft.Trigger.Threshold),
		"{meter_kind}":  string(ft.Reading.Kind),
		"{time}":        ft.Reading.Time.Format("2006-01-02 15:04:05"),
	}

	for k, v := range replacements {
		result = replaceAll(result, k, v)
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := s
	for {
		idx := indexOf(result, old)
		if idx < 0 {
			break
		}
		result = result[:idx] + new + result[idx+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
