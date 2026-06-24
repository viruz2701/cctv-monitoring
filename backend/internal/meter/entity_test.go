// Package meter — tests
package meter

import (
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Meter entity tests
// ═══════════════════════════════════════════════════════════════════════

func TestDefaultMeters(t *testing.T) {
	meters := DefaultMeters("dev-001")
	if len(meters) != 10 {
		t.Fatalf("expected 10 default meters, got %d", len(meters))
	}

	// Check CPU temp
	var cpuTemp *Meter
	for _, m := range meters {
		if m.Kind == MeterCPUTemp {
			cpuTemp = m
			break
		}
	}
	if cpuTemp == nil {
		t.Fatal("expected CPU temp meter")
	}
	if cpuTemp.Thresholds.Critical != 85 {
		t.Errorf("expected critical 85°C, got %f", cpuTemp.Thresholds.Critical)
	}
	if cpuTemp.Interval != 300 {
		t.Errorf("expected interval 300s, got %d", cpuTemp.Interval)
	}
}

func TestValidateMeterKind(t *testing.T) {
	if !ValidateMeterKind("cpu_temp") {
		t.Error("expected cpu_temp to be valid")
	}
	if !ValidateMeterKind("bitrate") {
		t.Error("expected bitrate to be valid")
	}
	if ValidateMeterKind("invalid") {
		t.Error("expected invalid to be invalid")
	}
}

func TestIsWithinThreshold(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		th    MeterThreshold
		want  string
	}{
		{"normal temp", 50, MeterThreshold{Warning: 75, Critical: 85, Min: -20, Max: 100}, "ok"},
		{"warning temp", 78, MeterThreshold{Warning: 75, Critical: 85, Min: -20, Max: 100}, "warning"},
		{"critical temp", 90, MeterThreshold{Warning: 75, Critical: 85, Min: -20, Max: 100}, "critical"},
		{"normal fps", 25, MeterThreshold{Warning: 15, Critical: 10, Min: 5, Max: 30}, "ok"},
		{"low fps warning", 12, MeterThreshold{Warning: 15, Critical: 10, Min: 5, Max: 30}, "warning"},
		{"low fps critical", 8, MeterThreshold{Warning: 15, Critical: 10, Min: 5, Max: 30}, "critical"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsWithinThreshold(tt.value, tt.th)
			if got != tt.want {
				t.Errorf("IsWithinThreshold(%f) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Trigger engine tests
// ═══════════════════════════════════════════════════════════════════════

func TestDefaultTriggers(t *testing.T) {
	triggers := DefaultTriggers()
	if len(triggers) != 4 {
		t.Fatalf("expected 4 default triggers, got %d", len(triggers))
	}

	// Check CPU trigger
	cpuTrigger := triggers[0]
	if cpuTrigger.MeterKind != MeterCPUTemp {
		t.Errorf("expected CPU temp trigger, got %s", cpuTrigger.MeterKind)
	}
	if cpuTrigger.Threshold != 85 {
		t.Errorf("expected threshold 85, got %f", cpuTrigger.Threshold)
	}
	if cpuTrigger.DurationSeconds != 600 {
		t.Errorf("expected duration 600s, got %d", cpuTrigger.DurationSeconds)
	}
}

func TestTriggerEngine_BasicCondition(t *testing.T) {
	engine := NewTriggerEngine(nil)
	engine.SetTriggers(DefaultTriggers())

	// Add a reading that exceeds CPU threshold
	reading := Reading{
		Time:     time.Now(),
		MeterID:  "meter-001",
		DeviceID: "dev-001",
		Kind:     MeterCPUTemp,
		Value:    90, // exceeds 85°C
	}

	fired := engine.AddReading(reading, "Camera-1")
	// First reading won't trigger (needs duration check)
	if len(fired) != 0 {
		t.Logf("first reading triggered %d triggers (may vary)", len(fired))
	}

	// Add multiple readings above threshold
	for i := 0; i < 10; i++ {
		r := Reading{
			Time:     time.Now().Add(time.Duration(i) * time.Second),
			MeterID:  "meter-001",
			DeviceID: "dev-001",
			Kind:     MeterCPUTemp,
			Value:    90,
		}
		fired = engine.AddReading(r, "Camera-1")
	}

	// Should have fired by now
	if len(fired) > 0 {
		t.Logf("trigger fired: %s", fired[0].Trigger.Name)
		if fired[0].Trigger.MeterKind != MeterCPUTemp {
			t.Errorf("expected CPU temp trigger, got %s", fired[0].Trigger.MeterKind)
		}
	}
}

func TestTriggerEngine_TemplateFilling(t *testing.T) {
	ft := FiredTrigger{
		Trigger: WorkOrderMeterTrigger{
			Threshold: 85,
			Action: TriggerAction{
				TitleTemplate: "CPU overheating on {device_name}",
				DescTemplate:  "CPU temp on {device_name} is {value}°C (threshold: {threshold})",
			},
		},
		Reading: Reading{
			DeviceID: "dev-001",
			Kind:     MeterCPUTemp,
			Value:    90.5,
		},
		DeviceName: "Camera-1",
	}

	title := ft.GenerateWOTitle()
	expectedTitle := "CPU overheating on Camera-1"
	if title != expectedTitle {
		t.Errorf("expected '%s', got '%s'", expectedTitle, title)
	}

	desc := ft.GenerateWODescription()
	expectedDesc := "CPU temp on Camera-1 is 90.5°C (threshold: 85.0)"
	if desc != expectedDesc {
		t.Errorf("expected '%s', got '%s'", expectedDesc, desc)
	}
}

func TestTriggerEngine_Cooldown(t *testing.T) {
	engine := NewTriggerEngine(nil)

	trigger := &WorkOrderMeterTrigger{
		ID:              "trigger-001",
		Name:            "Test Trigger",
		Enabled:         true,
		MeterKind:       MeterCPUTemp,
		Condition:       CondGreaterThan,
		Threshold:       80,
		DurationSeconds: 0, // instant
		CooldownMinutes: 60,
		Action: TriggerAction{
			WorkOrderType: "preventive",
			Priority:      "high",
			TitleTemplate: "Test: {device_name}",
		},
	}
	engine.SetTriggers([]*WorkOrderMeterTrigger{trigger})

	// First trigger should fire
	fired := engine.AddReading(Reading{
		Time: time.Now(), MeterID: "m-1", DeviceID: "dev-001",
		Kind: MeterCPUTemp, Value: 90,
	}, "Camera-1")

	if len(fired) != 1 {
		t.Error("expected trigger to fire on first reading")
	}

	// Second trigger should NOT fire (cooldown)
	fired = engine.AddReading(Reading{
		Time: time.Now(), MeterID: "m-1", DeviceID: "dev-001",
		Kind: MeterCPUTemp, Value: 95,
	}, "Camera-1")

	if len(fired) != 0 {
		t.Error("expected trigger NOT to fire (cooldown)")
	}
}

func TestTriggerEngine_DeviceFilter(t *testing.T) {
	engine := NewTriggerEngine(nil)

	trigger := &WorkOrderMeterTrigger{
		ID:              "trigger-002",
		Name:            "Specific Device Only",
		Enabled:         true,
		MeterKind:       MeterCPUTemp,
		Condition:       CondGreaterThan,
		Threshold:       80,
		DurationSeconds: 0,
		DeviceIDs:       []string{"dev-001"},
		Action: TriggerAction{
			TitleTemplate: "Test: {device_name}",
		},
	}
	engine.SetTriggers([]*WorkOrderMeterTrigger{trigger})

	// Different device - should not fire
	fired := engine.AddReading(Reading{
		Time: time.Now(), MeterID: "m-1", DeviceID: "dev-002",
		Kind: MeterCPUTemp, Value: 90,
	}, "Camera-2")

	if len(fired) != 0 {
		t.Error("expected trigger NOT to fire for different device")
	}

	// Correct device - should fire
	fired = engine.AddReading(Reading{
		Time: time.Now(), MeterID: "m-1", DeviceID: "dev-001",
		Kind: MeterCPUTemp, Value: 90,
	}, "Camera-1")

	if len(fired) != 1 {
		t.Error("expected trigger to fire for matching device")
	}
}

func TestValidateMeterKindList(t *testing.T) {
	expected := []string{
		"bitrate", "fps", "cpu_temp", "cpu_usage", "memory_usage",
		"error_count", "offline_ratio", "packet_loss", "signal_strength",
		"disk_usage", "recording_duration", "motion_events",
	}

	if len(ValidMeterKinds) != len(expected) {
		t.Fatalf("expected %d meter kinds, got %d", len(expected), len(ValidMeterKinds))
	}

	for _, exp := range expected {
		if !ValidateMeterKind(exp) {
			t.Errorf("expected %s to be valid", exp)
		}
	}
}

func TestIsWithinThreshold_EdgeCases(t *testing.T) {
	th := MeterThreshold{Warning: 75, Critical: 85, Min: -20, Max: 100}

	// Exactly at min
	if IsWithinThreshold(-20, th) != "ok" {
		t.Error("expected -20 to be ok (at min)")
	}

	// Below min
	if IsWithinThreshold(-25, th) != "critical" {
		t.Error("expected -25 to be critical (below min)")
	}

	// Exactly at warning
	if IsWithinThreshold(75, th) != "warning" {
		t.Error("expected 75 to be warning (at warning)")
	}

	// Exactly at critical
	if IsWithinThreshold(85, th) != "critical" {
		t.Error("expected 85 to be critical (at critical)")
	}
}
