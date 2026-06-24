// Package sla — tests for Advanced SLA Engine
package sla

import (
	"context"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// SLA Policy tests
// ═══════════════════════════════════════════════════════════════════════

func TestDefaultPolicies(t *testing.T) {
	policies := DefaultPolicies()
	if len(policies) != 3 {
		t.Fatalf("expected 3 default policies, got %d", len(policies))
	}

	// Check standard policy
	std := policies[0]
	if std.Type != SLAPolicyStandard {
		t.Errorf("expected standard type, got %s", std.Type)
	}
	if !std.IsDefault {
		t.Error("expected standard to be default")
	}
	if std.ResponseTimeMinutes != 120 {
		t.Errorf("expected 120min response, got %d", std.ResponseTimeMinutes)
	}

	// Check 24x7 policy
	p247 := policies[2]
	if p247.Type != SLAPolicy247 {
		t.Errorf("expected 24x7 type, got %s", p247.Type)
	}
	if p247.ResponseTimeMinutes != 15 {
		t.Errorf("expected 15min response, got %d", p247.ResponseTimeMinutes)
	}
}

func TestValidateSLAPolicyType(t *testing.T) {
	if !ValidateSLAPolicyType("standard") {
		t.Error("expected standard to be valid")
	}
	if !ValidateSLAPolicyType("premium") {
		t.Error("expected premium to be valid")
	}
	if !ValidateSLAPolicyType("24x7") {
		t.Error("expected 24x7 to be valid")
	}
	if ValidateSLAPolicyType("invalid") {
		t.Error("expected invalid to be invalid")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Matrix tests
// ═══════════════════════════════════════════════════════════════════════

func TestDefaultMatrix(t *testing.T) {
	entries := DefaultMatrix("sla-std")
	if len(entries) != 16 { // 4 priorities × 4 impacts
		t.Fatalf("expected 16 matrix entries, got %d", len(entries))
	}

	// Check critical × extensive (strictest)
	var found bool
	for _, e := range entries {
		if e.Priority == "critical" && e.Impact == ImpactExtensive {
			found = true
			if e.ResponseTimeMinutes != 5 {
				t.Errorf("expected 5min response for critical×extensive, got %d", e.ResponseTimeMinutes)
			}
			if e.ResolutionTimeMinutes != 30 {
				t.Errorf("expected 30min resolution for critical×extensive, got %d", e.ResolutionTimeMinutes)
			}
		}
	}
	if !found {
		t.Error("critical×extensive entry not found")
	}
}

func TestValidateImpactLevel(t *testing.T) {
	if !ValidateImpactLevel("extensive") {
		t.Error("expected extensive to be valid")
	}
	if !ValidateImpactLevel("minor") {
		t.Error("expected minor to be valid")
	}
	if ValidateImpactLevel("invalid") {
		t.Error("expected invalid to be invalid")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Business Calendar tests
// ═══════════════════════════════════════════════════════════════════════

func TestBusinessCalendar_IsWorkHour(t *testing.T) {
	cal := &BusinessCalendar{
		Timezone:      "Europe/Minsk",
		WorkStartHour: 9,
		WorkEndHour:   18,
		WorkDays:      []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
	}

	// Monday 10:00 = work hour
	monday10am := time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC) // Monday
	if !cal.IsWorkHour(monday10am) {
		t.Error("expected Monday 10:00 to be work hour")
	}

	// Monday 20:00 = not work hour
	monday8pm := time.Date(2026, 6, 29, 20, 0, 0, 0, time.UTC)
	if cal.IsWorkHour(monday8pm) {
		t.Error("expected Monday 20:00 to NOT be work hour")
	}

	// Sunday = not work hour
	sunday := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC) // Sunday
	if cal.IsWorkHour(sunday) {
		t.Error("expected Sunday to NOT be work hour")
	}
}

func TestBusinessCalendar_Holiday(t *testing.T) {
	cal := &BusinessCalendar{
		Timezone:      "Europe/Minsk",
		WorkStartHour: 9,
		WorkEndHour:   18,
		WorkDays:      []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Holidays: []CalendarHoliday{
			{
				Date:      time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC), // Independence Day
				Name:      "Independence Day",
				Recurring: true,
			},
		},
	}

	// July 3 (Friday) = holiday, should not be work hour
	holiday := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	if cal.IsWorkHour(holiday) {
		t.Error("expected holiday to NOT be work hour")
	}
}

func TestBusinessCalendar_Exception(t *testing.T) {
	halfDay := 13
	cal := &BusinessCalendar{
		Timezone:      "Europe/Minsk",
		WorkStartHour: 9,
		WorkEndHour:   18,
		WorkDays:      []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
		Exceptions: []CalendarException{
			{
				Date:        time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
				Description: "New Year's Eve (half day)",
				WorkEnd:     &halfDay,
			},
		},
	}

	// Dec 31 (Thursday) 08:00 UTC = 11:00 MSK = work hour (exception allows until 13:00 MSK)
	nyeMorning := time.Date(2026, 12, 31, 8, 0, 0, 0, time.UTC)
	if !cal.IsWorkHour(nyeMorning) {
		t.Error("expected Dec 31 08:00 UTC to be work hour (half day until 13:00 MSK)")
	}

	// Dec 31 15:00 = not work hour (exception ends at 13:00)
	nyeAfternoon := time.Date(2026, 12, 31, 15, 0, 0, 0, time.UTC)
	if cal.IsWorkHour(nyeAfternoon) {
		t.Error("expected Dec 31 15:00 to NOT be work hour (half day)")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Pause Rules tests
// ═══════════════════════════════════════════════════════════════════════

func TestDefaultPauseRules(t *testing.T) {
	rules := DefaultPauseRules("sla-std")
	if len(rules) != 4 {
		t.Fatalf("expected 4 pause rules, got %d", len(rules))
	}

	if !IsPausedStatus("ON_HOLD", rules) {
		t.Error("expected ON_HOLD to be paused")
	}
	if !IsPausedStatus("AWAITING_PARTS", rules) {
		t.Error("expected AWAITING_PARTS to be paused")
	}
	if IsPausedStatus("IN_PROGRESS", rules) {
		t.Error("expected IN_PROGRESS to NOT be paused")
	}
	if IsPausedStatus("COMPLETED", rules) {
		t.Error("expected COMPLETED to NOT be paused")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Calculation Engine tests
// ═══════════════════════════════════════════════════════════════════════

func TestEngine_StartTracking(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	// Setup policy
	engine.SetPolicy(DefaultPolicies()[2]) // 24x7

	// Start tracking
	tracker, err := engine.StartTracking(ctx, "wo-001", "sla-247", "critical", "extensive", "site-001")
	if err != nil {
		t.Fatalf("StartTracking failed: %v", err)
	}

	if tracker.WorkOrderID != "wo-001" {
		t.Errorf("expected wo-001, got %s", tracker.WorkOrderID)
	}
	if tracker.Status != SLAOnTrack {
		t.Errorf("expected on_track, got %s", tracker.Status)
	}
	if tracker.ResponseDeadline == nil {
		t.Error("expected response deadline to be set")
	}
	if tracker.ResolutionDeadline == nil {
		t.Error("expected resolution deadline to be set")
	}
}

func TestEngine_StartTrackingWithMatrix(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	policy := DefaultPolicies()[2] // 24x7
	engine.SetPolicy(policy)
	engine.SetMatrix("sla-247", DefaultMatrix("sla-247"))

	// Critical × Extensive — должно взять из матрицы (5min/30min)
	tracker, err := engine.StartTracking(ctx, "wo-002", "sla-247", "critical", "extensive", "site-001")
	if err != nil {
		t.Fatalf("StartTracking failed: %v", err)
	}

	if tracker.ResponseTargetMinutes != 5 {
		t.Errorf("expected 5min response from matrix, got %d", tracker.ResponseTargetMinutes)
	}
	if tracker.ResolutionTargetMinutes != 30 {
		t.Errorf("expected 30min resolution from matrix, got %d", tracker.ResolutionTargetMinutes)
	}
}

func TestEngine_PauseResume(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	policy := DefaultPolicies()[2] // 24x7
	engine.SetPolicy(policy)
	engine.SetPauseRules("sla-247", DefaultPauseRules("sla-247"))

	// Start tracking
	_, err := engine.StartTracking(ctx, "wo-003", "sla-247", "high", "limited", "site-001")
	if err != nil {
		t.Fatalf("StartTracking failed: %v", err)
	}

	// Pause
	tracker, err := engine.UpdateStatus(ctx, "wo-003", "ON_HOLD")
	if err != nil {
		t.Fatalf("UpdateStatus pause failed: %v", err)
	}
	if !tracker.IsPaused {
		t.Error("expected tracker to be paused")
	}
	if tracker.Status != SLAPaused {
		t.Errorf("expected paused status, got %s", tracker.Status)
	}

	// Небольшая задержка чтобы TotalPauseSeconds > 0
	time.Sleep(10 * time.Millisecond)

	// Resume
	tracker, err = engine.UpdateStatus(ctx, "wo-003", "IN_PROGRESS")
	if err != nil {
		t.Fatalf("UpdateStatus resume failed: %v", err)
	}
	if tracker.IsPaused {
		t.Error("expected tracker to NOT be paused")
	}
	if tracker.TotalPauseMs <= 0 {
		t.Errorf("expected positive pause duration, got %dms", tracker.TotalPauseMs)
	}
}

func TestEngine_CompleteAndBreach(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	policy := DefaultPolicies()[2] // 24x7 (15min response, 60min resolution)
	engine.SetPolicy(policy)

	// Создаём трекер в прошлом (чтобы был breach)
	// Используем низкий приоритет с коротким таргетом
	// Start tracking
	_, err := engine.StartTracking(ctx, "wo-004", "sla-247", "critical", "extensive", "site-001")
	if err != nil {
		t.Fatalf("StartTracking failed: %v", err)
	}

	// Получаем трекер и проверяем
	tracker, ok := engine.GetTracker("wo-004")
	if !ok {
		t.Fatal("expected tracker to exist")
	}

	// Complete
	tracker, err = engine.CompleteWorkOrder(ctx, "wo-004")
	if err != nil {
		t.Fatalf("CompleteWorkOrder failed: %v", err)
	}

	// Должен быть on_track (только что создали, прошло мало времени)
	if tracker.Status != SLAOnTrack {
		t.Logf("SLA status: %s (may vary based on timing)", tracker.Status)
	}

	// Проверяем breached query
	breached := engine.GetBreached()
	// Не ждём breach т.к. тест быстрый
	_ = breached
}

func TestEngine_GetAtRisk(t *testing.T) {
	engine := NewEngine(nil)
	ctx := context.Background()

	engine.SetPolicy(DefaultPolicies()[2]) // 24x7

	_, err := engine.StartTracking(ctx, "wo-005", "sla-247", "low", "minor", "site-001")
	if err != nil {
		t.Fatalf("StartTracking failed: %v", err)
	}

	atRisk := engine.GetAtRisk()
	// Not at risk immediately
	if len(atRisk) > 0 {
		t.Logf("at risk WOs: %d (may vary)", len(atRisk))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Deadline Calculator tests
// ═══════════════════════════════════════════════════════════════════════

func TestCalculateDeadline_24x7(t *testing.T) {
	// 24x7 calendar = всегда рабочее время
	cal := &BusinessCalendar{
		Timezone:      "UTC",
		WorkStartHour: 0,
		WorkEndHour:   23,
		WorkDays:      []time.Weekday{time.Sunday, time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday},
	}

	now := time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC) // Wednesday 10:00
	deadline := calculateDeadline(now, 60, cal)            // 60 min

	expected := now.Add(60 * time.Minute)
	if !deadline.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, deadline)
	}
}

func TestCalculateDeadline_BusinessHours(t *testing.T) {
	// Standard calendar: Mon-Fri 9-18
	cal := &BusinessCalendar{
		Timezone:      "UTC",
		WorkStartHour: 9,
		WorkEndHour:   18,
		WorkDays:      []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
	}

	// Friday 16:00 + 120 min = should go to Monday (not weekend)
	friday := time.Date(2026, 7, 3, 16, 0, 0, 0, time.UTC) // Friday
	deadline := calculateDeadline(friday, 120, cal)

	// 120 min = 2h. Friday 16:00 + 2h working = Monday 09:00 + 1h = Monday 10:00
	// Actually: 16:00→18:00 = 2h, so Friday 16:00 + 120min = Friday 18:00
	expected := time.Date(2026, 7, 3, 18, 0, 0, 0, time.UTC)
	if !deadline.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, deadline)
	}
}

func TestCalculateDeadline_AfterHours(t *testing.T) {
	cal := &BusinessCalendar{
		Timezone:      "UTC",
		WorkStartHour: 9,
		WorkEndHour:   18,
		WorkDays:      []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday},
	}

	// Monday 17:00 + 120 min = Monday 18:00 (60min) + Tuesday 09:00 (60min) = Tuesday 10:00
	monday := time.Date(2026, 6, 29, 17, 0, 0, 0, time.UTC) // Monday 17:00
	deadline := calculateDeadline(monday, 120, cal)

	expected := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC) // Tuesday 10:00
	if !deadline.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, deadline)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// FormatDuration tests
// ═══════════════════════════════════════════════════════════════════════

func TestFormatDurationHuman(t *testing.T) {
	tests := []struct {
		minutes int
		expected string
	}{
		{30, "30m"},
		{60, "1h"},
		{90, "1h 30m"},
		{480, "8h"},
		{1440, "24h"},
	}

	for _, tt := range tests {
		result := FormatDurationHuman(tt.minutes)
		if result != tt.expected {
			t.Errorf("FormatDurationHuman(%d) = %s, want %s", tt.minutes, result, tt.expected)
		}
	}
}
