// Package sla — Advanced SLA Engine для CCTV Health Monitor.
//
// Заменяет плоскую SLAConfig на enterprise SLA-движок:
//   - SLA Policy (Standard/Premium/24×7)
//   - SLA Matrix (Priority × Impact)
//   - Business Calendar (timezone, shifts, holidays)
//   - SLA Pause Rules (statuses для паузы таймера)
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — SLA метрики)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - ISO 27001 A.12.4.1 (Audit — SLA breach events)
//   - Приказ ОАЦ №66 п. 7.18.3 (SLA для edge devices)
package sla

import (
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.1.1: SLA Policy
// ═══════════════════════════════════════════════════════════════════════

// SLAPolicyType определяет тип SLA политики.
type SLAPolicyType string

const (
	SLAPolicyStandard SLAPolicyType = "standard" // Business hours, next business day
	SLAPolicyPremium  SLAPolicyType = "premium"  // Extended hours, faster response
	SLAPolicy247     SLAPolicyType = "24x7"     // Round-the-clock, immediate response
)

// ValidSLAPolicyTypes для whitelist validation (OWASP ASVS V5.1).
var ValidSLAPolicyTypes = []string{
	string(SLAPolicyStandard),
	string(SLAPolicyPremium),
	string(SLAPolicy247),
}

// SLAPolicy — политика обслуживания (Standard/Premium/24×7).
//
// Определяет базовые параметры SLA для клиента/сайта.
type SLAPolicy struct {
	ID          string        `json:"id" db:"id"`
	Name        string        `json:"name" db:"name" validate:"required,max=100"`
	Type        SLAPolicyType `json:"type" db:"type" validate:"required,oneof=standard premium 24x7"`
	Description string        `json:"description,omitempty" db:"description" validate:"max=500"`
	IsDefault   bool          `json:"is_default" db:"is_default"`

	// Business hours (для standard/premium)
	WorkStartHour int `json:"work_start_hour" db:"work_start_hour" validate:"min=0,max=23"`   // 9 = 09:00
	WorkEndHour   int `json:"work_end_hour" db:"work_end_hour" validate:"min=0,max=23"`       // 18 = 18:00
	WorkDays      []time.Weekday `json:"work_days" db:"work_days"`                            // Monday-Friday

	// SLA targets (в рабочих часах)
	ResponseTimeMinutes   int `json:"response_time_minutes" db:"response_time_minutes" validate:"min=1"`
	ResolutionTimeMinutes int `json:"resolution_time_minutes" db:"resolution_time_minutes" validate:"min=1"`

	// Escalation
	Escalation1AfterMinutes int `json:"escalation_1_after_minutes" db:"escalation_1_after_minutes"`
	Escalation2AfterMinutes int `json:"escalation_2_after_minutes" db:"escalation_2_after_minutes"`
	Escalation3AfterMinutes int `json:"escalation_3_after_minutes" db:"escalation_3_after_minutes"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// DefaultPolicies возвращает 3 стандартные SLA политики.
func DefaultPolicies() []*SLAPolicy {
	return []*SLAPolicy{
		{
			ID:                    "sla-std",
			Name:                  "Standard",
			Type:                  SLAPolicyStandard,
			Description:           "Business hours support, next business day resolution",
			IsDefault:             true,
			WorkStartHour:         9,
			WorkEndHour:           18,
			WorkDays:              []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
			ResponseTimeMinutes:   120,  // 2h response (business)
			ResolutionTimeMinutes: 960,  // 16h resolution (2 business days)
			Escalation1AfterMinutes: 240,  // 4h → L1
			Escalation2AfterMinutes: 480,  // 8h → L2
			Escalation3AfterMinutes: 1440, // 24h → L3
		},
		{
			ID:                    "sla-prem",
			Name:                  "Premium",
			Type:                  SLAPolicyPremium,
			Description:           "Extended hours support, faster resolution",
			IsDefault:             false,
			WorkStartHour:         7,
			WorkEndHour:           22,
			WorkDays:              []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday},
			ResponseTimeMinutes:   30,   // 30min response
			ResolutionTimeMinutes: 240,  // 4h resolution
			Escalation1AfterMinutes: 60,   // 1h → L1
			Escalation2AfterMinutes: 180,  // 3h → L2
			Escalation3AfterMinutes: 480,  // 8h → L3
		},
		{
			ID:                    "sla-247",
			Name:                  "24×7",
			Type:                  SLAPolicy247,
			Description:           "Round-the-clock support, immediate response",
			IsDefault:             false,
			WorkStartHour:         0,
			WorkEndHour:           23,
			WorkDays:              []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday},
			ResponseTimeMinutes:   15,   // 15min response
			ResolutionTimeMinutes: 60,   // 1h resolution
			Escalation1AfterMinutes: 30,   // 30min → L1
			Escalation2AfterMinutes: 90,   // 1.5h → L2
			Escalation3AfterMinutes: 180,  // 3h → L3
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.1.2: SLA Matrix (Priority × Impact)
// ═══════════════════════════════════════════════════════════════════════

// ImpactLevel определяет уровень влияния инцидента.
type ImpactLevel string

const (
	ImpactExtensive ImpactLevel = "extensive" // Множество устройств/клиентов
	ImpactSignificant ImpactLevel = "significant" // Один клиент, несколько устройств
	ImpactLimited   ImpactLevel = "limited"    // Одно устройство
	ImpactMinor     ImpactLevel = "minor"      // Некритичная проблема
)

// ValidImpactLevels для whitelist validation.
var ValidImpactLevels = []string{
	string(ImpactExtensive),
	string(ImpactSignificant),
	string(ImpactLimited),
	string(ImpactMinor),
}

// SLAMatrixEntry — запись матрицы SLA: Priority × Impact → целевое время.
//
// Пример:
//
//	Priority: critical, Impact: extensive → Response: 5min, Resolution: 30min
//	Priority: low, Impact: minor → Response: 4h, Resolution: 24h
type SLAMatrixEntry struct {
	ID                    string       `json:"id" db:"id"`
	PolicyID              string       `json:"policy_id" db:"policy_id" validate:"required"`
	Priority              string       `json:"priority" db:"priority" validate:"required,oneof=critical high medium low"`
	Impact                ImpactLevel  `json:"impact" db:"impact" validate:"required,oneof=extensive significant limited minor"`
	ResponseTimeMinutes   int          `json:"response_time_minutes" db:"response_time_minutes" validate:"min=1"`
	ResolutionTimeMinutes int          `json:"resolution_time_minutes" db:"resolution_time_minutes" validate:"min=1"`
	Escalation1Minutes    int          `json:"escalation_1_minutes" db:"escalation_1_minutes"`
	Escalation2Minutes    int          `json:"escalation_2_minutes" db:"escalation_2_minutes"`
	Escalation3Minutes    int          `json:"escalation_3_minutes" db:"escalation_3_minutes"`
}

// DefaultMatrix возвращает матрицу SLA по умолчанию для заданной политики.
func DefaultMatrix(policyID string) []*SLAMatrixEntry {
	entries := make([]*SLAMatrixEntry, 0)

	priorities := []string{"critical", "high", "medium", "low"}
	impacts := []ImpactLevel{ImpactExtensive, ImpactSignificant, ImpactLimited, ImpactMinor}

	// Critical × Extensive = самый строгий → самый мягкий
	targets := map[string]map[ImpactLevel]struct {
		Resp, Res int
		E1, E2, E3 int
	}{
		"critical": {
			ImpactExtensive:    {5, 30, 10, 20, 45},
			ImpactSignificant:  {10, 60, 20, 40, 90},
			ImpactLimited:      {15, 120, 30, 60, 180},
			ImpactMinor:        {30, 240, 60, 120, 360},
		},
		"high": {
			ImpactExtensive:    {15, 60, 30, 60, 120},
			ImpactSignificant:  {30, 120, 60, 120, 240},
			ImpactLimited:      {45, 240, 90, 180, 360},
			ImpactMinor:        {60, 480, 120, 240, 480},
		},
		"medium": {
			ImpactExtensive:    {30, 120, 60, 120, 240},
			ImpactSignificant:  {60, 240, 120, 240, 480},
			ImpactLimited:      {90, 480, 180, 360, 720},
			ImpactMinor:        {120, 960, 240, 480, 1440},
		},
		"low": {
			ImpactExtensive:    {60, 240, 120, 240, 480},
			ImpactSignificant:  {120, 480, 240, 480, 960},
			ImpactLimited:      {240, 960, 480, 720, 1440},
			ImpactMinor:        {480, 2880, 720, 1440, 4320},
		},
	}

	for _, p := range priorities {
		for _, imp := range impacts {
			t := targets[p][imp]
			entries = append(entries, &SLAMatrixEntry{
				PolicyID:              policyID,
				Priority:              p,
				Impact:                imp,
				ResponseTimeMinutes:   t.Resp,
				ResolutionTimeMinutes: t.Res,
				Escalation1Minutes:    t.E1,
				Escalation2Minutes:    t.E2,
				Escalation3Minutes:    t.E3,
			})
		}
	}

	return entries
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.1.3: Business Calendar
// ═══════════════════════════════════════════════════════════════════════

// BusinessCalendar определяет рабочий календарь для сайта.
//
// Учитывает:
//   - Часовой пояс
//   - Рабочие смены
//   - Праздники (национальные + корпоративные)
//   - Исключения (специальные даты)
type BusinessCalendar struct {
	ID        string `json:"id" db:"id"`
	SiteID    string `json:"site_id" db:"site_id" validate:"required"`
	Name      string `json:"name" db:"name" validate:"required,max=100"`
	Timezone  string `json:"timezone" db:"timezone" validate:"required"` // IANA: "Europe/Minsk", "Asia/Tashkent"

	// Weekly schedule
	WorkStartHour int            `json:"work_start_hour" db:"work_start_hour" validate:"min=0,max=23"`
	WorkEndHour   int            `json:"work_end_hour" db:"work_end_hour" validate:"min=0,max=23"`
	WorkDays      []time.Weekday `json:"work_days" db:"work_days"` // дни недели

	// Holidays (даты, когда сервис не работает)
	Holidays []CalendarHoliday `json:"holidays,omitempty"`
	// Exceptions (специальные даты с другим расписанием)
	Exceptions []CalendarException `json:"exceptions,omitempty"`

	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CalendarHoliday — праздничный день.
type CalendarHoliday struct {
	Date        time.Time `json:"date"`        // дата праздника
	Name        string    `json:"name"`         // "New Year", "Christmas"
	Recurring   bool      `json:"recurring"`    // ежегодный?
	HalfDay     bool      `json:"half_day"`     // сокращённый день?
}

// CalendarException — исключение из расписания.
type CalendarException struct {
	Date        time.Time `json:"date"`         // дата
	Description string    `json:"description"`  // "Maintenance window", "Emergency"
	WorkStart   *int      `json:"work_start,omitempty"` // null = не работает
	WorkEnd     *int      `json:"work_end,omitempty"`
}

// IsWorkHour проверяет, является ли указанное время рабочим.
func (bc *BusinessCalendar) IsWorkHour(t time.Time) bool {
	loc, err := time.LoadLocation(bc.Timezone)
	if err != nil {
		return false
	}
	local := t.In(loc)

	// Проверка праздников
	for _, h := range bc.Holidays {
		if isSameDay(local, h.Date) {
			return h.HalfDay && local.Hour() < bc.WorkEndHour-4
		}
	}

	// Проверка исключений
	for _, e := range bc.Exceptions {
		if isSameDay(local, e.Date) {
			start := bc.WorkStartHour
			end := bc.WorkEndHour
			if e.WorkStart != nil {
				start = *e.WorkStart
			}
			if e.WorkEnd != nil {
				end = *e.WorkEnd
			}
			// Если start == end — нерабочий день
			if start >= end {
				return false
			}
			return local.Hour() >= start && local.Hour() < end
		}
	}

	// Проверка дня недели
	isWorkDay := false
	for _, d := range bc.WorkDays {
		if local.Weekday() == d {
			isWorkDay = true
			break
		}
	}
	if !isWorkDay {
		return false
	}

	// Проверка часов
	return local.Hour() >= bc.WorkStartHour && local.Hour() < bc.WorkEndHour
}

// NextWorkStart возвращает следующее рабочее время.
func (bc *BusinessCalendar) NextWorkStart(from time.Time) time.Time {
	loc, _ := time.LoadLocation(bc.Timezone)
	local := from.In(loc)

	// Пробуем каждый час в течение следующих 7 дней
	for i := 0; i < 168; i++ { // 7 дней × 24ч
		candidate := local.Add(time.Duration(i) * time.Hour)
		if bc.IsWorkHour(candidate) {
			return candidate
		}
	}
	return local.Add(7 * 24 * time.Hour)
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.1.4: Pause Rules
// ═══════════════════════════════════════════════════════════════════════

// SLAPauseRule определяет при каком статусе Work Order ставится на паузу.
//
// Когда WO в paused status — SLA таймер не тикает.
// Это справедливо для: ожидания запчастей, ожидания вендора, ожидания клиента.
type SLAPauseRule struct {
	ID          string `json:"id" db:"id"`
	PolicyID    string `json:"policy_id" db:"policy_id" validate:"required"`
	Status      string `json:"status" db:"status" validate:"required"` // WorkOrder status
	Description string `json:"description,omitempty" db:"description"`
	IsActive    bool   `json:"is_active" db:"is_active"`
}

// DefaultPauseRules возвращает правила паузы по умолчанию.
func DefaultPauseRules(policyID string) []*SLAPauseRule {
	return []*SLAPauseRule{
		{PolicyID: policyID, Status: "ON_HOLD", Description: "WO on hold by dispatcher", IsActive: true},
		{PolicyID: policyID, Status: "AWAITING_PARTS", Description: "Waiting for spare parts", IsActive: true},
		{PolicyID: policyID, Status: "AWAITING_VENDOR", Description: "Waiting for vendor", IsActive: true},
		{PolicyID: policyID, Status: "AWAITING_CLIENT", Description: "Waiting for client response", IsActive: true},
	}
}

// IsPausedStatus проверяет, является ли статус паузой для SLA.
func IsPausedStatus(status string, rules []*SLAPauseRule) bool {
	for _, r := range rules {
		if r.Status == status && r.IsActive {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func isSameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// ValidateSLAPolicyType проверяет тип SLA политики.
func ValidateSLAPolicyType(t string) bool {
	for _, valid := range ValidSLAPolicyTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// ValidateImpactLevel проверяет уровень влияния.
func ValidateImpactLevel(level string) bool {
	for _, valid := range ValidImpactLevels {
		if level == valid {
			return true
		}
	}
	return false
}

// FormatDurationHuman форматирует длительность в человекочитаемый вид.
func FormatDurationHuman(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}
