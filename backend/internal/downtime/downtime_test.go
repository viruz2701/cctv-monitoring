package downtime

import (
	"testing"
	"time"
)

func TestStartAndEndDowntime(t *testing.T) {
	tracker := NewTracker(nil)

	// Устанавливаем StartedAt в прошлом для корректного Duration
	pastStart := time.Now().Add(-30 * time.Minute)
	d := tracker.StartDowntime("cam-001", "Camera-1", "camera", "site-1", ReasonHardware, "Hardware failure")
	d.StartedAt = pastStart // переопределяем для теста

	if d == nil {
		t.Fatal("expected non-nil downtime")
	}
	if !d.IsActive() {
		t.Error("expected active downtime")
	}
	if d.HourlyCost != 15.0 {
		t.Errorf("expected hourly cost $15 for camera, got $%.2f", d.HourlyCost)
	}

	// Duplicate start should return same
	d2 := tracker.StartDowntime("cam-001", "Camera-1", "camera", "site-1", ReasonHardware, "")
	if d2 != d {
		t.Error("expected same downtime for duplicate start")
	}

	// End
	ended := tracker.EndDowntime("cam-001", "wo-001")
	if ended == nil {
		t.Fatal("expected ended downtime")
	}
	if ended.DurationMin < 29 || ended.DurationMin > 31 {
		t.Errorf("expected ~30 min duration, got %d min", ended.DurationMin)
	}
	if ended.TotalCost <= 0 {
		t.Errorf("expected positive cost, got $%.2f", ended.TotalCost)
	}
	if ended.WorkOrderID != "wo-001" {
		t.Errorf("expected WO ID wo-001, got %s", ended.WorkOrderID)
	}
}

func TestEndNonexistent(t *testing.T) {
	tracker := NewTracker(nil)
	d := tracker.EndDowntime("nonexistent", "")
	if d != nil {
		t.Error("expected nil for nonexistent downtime")
	}
}

func TestGetActive(t *testing.T) {
	tracker := NewTracker(nil)

	tracker.StartDowntime("cam-001", "Cam-1", "camera", "site-1", ReasonNetwork, "")
	tracker.StartDowntime("nvr-001", "NVR-1", "nvr", "site-1", ReasonPower, "")

	active := tracker.GetActive()
	if len(active) != 2 {
		t.Errorf("expected 2 active downtimes, got %d", len(active))
	}
}

func TestGetByDevice(t *testing.T) {
	tracker := NewTracker(nil)

	tracker.StartDowntime("cam-001", "Cam-1", "camera", "site-1", ReasonHardware, "")
	tracker.EndDowntime("cam-001", "")
	tracker.StartDowntime("cam-001", "Cam-1", "camera", "site-1", ReasonNetwork, "")

	history := tracker.GetByDevice("cam-001")
	if len(history) != 2 {
		t.Errorf("expected 2 entries for cam-001, got %d", len(history))
	}
}

func TestGetStats(t *testing.T) {
	tracker := NewTracker(nil)

	d := tracker.StartDowntime("cam-001", "Cam-1", "camera", "site-1", ReasonHardware, "")
	d.StartedAt = time.Now().Add(-30 * time.Minute)
	tracker.EndDowntime("cam-001", "")

	stats := tracker.GetStats()
	if stats.TotalDowntimes != 1 {
		t.Errorf("expected 1 total downtime, got %d", stats.TotalDowntimes)
	}
	if stats.TotalCost <= 0 {
		t.Errorf("expected positive total cost, got $%.2f", stats.TotalCost)
	}
	if stats.MTTR <= 0 {
		t.Errorf("expected positive MTTR, got %.1f", stats.MTTR)
	}
}

func TestCalculateTCO(t *testing.T) {
	tco := CalculateTCO("cam-001", "Camera-1", 500, 200, 150, 120, 15.0, 12)
	if tco == nil {
		t.Fatal("expected non-nil TCO")
	}
	// 120 min = 2 hours × $15/hr = $30 downtime cost
	// Total = 500 + 200 + 150 + 30 = 880
	if tco.TotalCost != 880.0 {
		t.Errorf("expected $880 total cost, got $%.2f", tco.TotalCost)
	}
	if tco.DowntimeCost != 30.0 {
		t.Errorf("expected $30 downtime cost, got $%.2f", tco.DowntimeCost)
	}
	if tco.PeriodMonths != 12 {
		t.Errorf("expected 12 months, got %d", tco.PeriodMonths)
	}
}

func TestHourlyCosts(t *testing.T) {
	if DefaultHourlyCosts["camera"] != 15.0 {
		t.Errorf("expected camera $15/hr, got $%.2f", DefaultHourlyCosts["camera"])
	}
	if DefaultHourlyCosts["server"] != 100.0 {
		t.Errorf("expected server $100/hr, got $%.2f", DefaultHourlyCosts["server"])
	}
}

func TestDowntimeSummary(t *testing.T) {
	tracker := NewTracker(nil)
	d := tracker.StartDowntime("cam-001", "Camera-1", "camera", "site-1", ReasonHardware, "Hardware failure")

	summary := d.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	t.Logf("Active summary: %s", summary)

	d.StartedAt = time.Now().Add(-30 * time.Minute)
	ended := tracker.EndDowntime("cam-001", "wo-001")
	summary = ended.Summary()
	if summary == "" {
		t.Error("expected non-empty summary after end")
	}
	t.Logf("Ended summary: %s", summary)
}

func TestMultipleDowntimes(t *testing.T) {
	tracker := NewTracker(nil)

	// Start 3
	tracker.StartDowntime("cam-001", "", "camera", "", ReasonNetwork, "")
	tracker.StartDowntime("cam-002", "", "camera", "", ReasonHardware, "")
	tracker.StartDowntime("sw-001", "", "switch", "", ReasonPower, "")

	active := tracker.GetActive()
	if len(active) != 3 {
		t.Errorf("expected 3 active, got %d", len(active))
	}

	// End 2
	tracker.EndDowntime("cam-001", "")
	tracker.EndDowntime("sw-001", "")

	active = tracker.GetActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active, got %d", len(active))
	}

	stats := tracker.GetStats()
	if stats.TotalDowntimes != 3 {
		t.Errorf("expected 3 total, got %d", stats.TotalDowntimes)
	}
}

func TestCalculateDurationAndCost(t *testing.T) {
	start := time.Now().Add(-2 * time.Hour)
	end := time.Now()

	d := &AssetDowntime{
		StartedAt:  start,
		EndedAt:    &end,
		HourlyCost: 30.0,
	}
	d.CalculateDuration()
	d.CalculateTotalCost()

	if d.DurationMin < 115 || d.DurationMin > 125 {
		t.Errorf("expected ~120 min duration, got %d", d.DurationMin)
	}
	// 2 hours × $30 = $60
	if d.TotalCost < 55 || d.TotalCost > 65 {
		t.Errorf("expected ~$60 total cost, got $%.2f", d.TotalCost)
	}
}

func TestReasons(t *testing.T) {
	reasons := []DowntimeReason{
		ReasonHardware, ReasonNetwork, ReasonPower,
		ReasonSoftware, ReasonMaintenance, ReasonUnknown,
	}
	for _, r := range reasons {
		if r == "" {
			t.Error("reason should not be empty")
		}
	}
}
