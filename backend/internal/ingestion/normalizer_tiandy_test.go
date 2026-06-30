// Package ingestion — unit tests for Tiandy Vendor Normalizer.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5.1: Input validation
package ingestion

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── Tests: TiandyNormalizer.Normalize ────────────────────────────────────

func TestTiandyNormalize_AlarmEvent(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{
		"eventName": "MotionDetect",
		"channel": 1,
		"status": "alarm",
		"startTime": "2026-06-30T12:00:00Z",
		"description": "Motion detected on channel 1"
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if !strings.EqualFold(event.Type, "MotionDetect") {
		t.Errorf("expected type 'MotionDetect', got %q", event.Type)
	}
	if event.Severity != "high" {
		t.Errorf("expected severity 'high' for alarm status, got %q", event.Severity)
	}
	if event.Source != "tiandy" {
		t.Errorf("expected source 'tiandy', got %q", event.Source)
	}
	if event.Message != "Motion detected on channel 1" {
		t.Errorf("expected message, got %q", event.Message)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Metadata["channel"] != "1" {
		t.Errorf("expected channel metadata '1', got %q", event.Metadata["channel"])
	}
	if event.Metadata["status"] != "alarm" {
		t.Errorf("expected status metadata 'alarm', got %q", event.Metadata["status"])
	}
}

func TestTiandyNormalize_TelemetryWithMetrics(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{
		"eventName": "TemperatureReading",
		"channel": 2,
		"status": "normal",
		"temperature": 45.2,
		"deviceTemp": 42.5,
		"cpuRate": 65.3,
		"memoryRate": 72.1,
		"netSpeed": 1000000
	}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Severity != "info" {
		t.Errorf("expected severity 'info' for normal status, got %q", event.Severity)
	}

	// Check all 5 metrics
	expectedMetrics := map[string]float64{
		"temperature":   45.2,
		"device_temp":   42.5,
		"cpu_usage":     65.3,
		"memory_usage":  72.1,
		"network_speed": 1000000,
	}

	if len(event.Metrics) != len(expectedMetrics) {
		t.Errorf("expected %d metrics, got %d", len(expectedMetrics), len(event.Metrics))
	}

	for _, m := range event.Metrics {
		expectedValue, ok := expectedMetrics[m.Name]
		if !ok {
			t.Errorf("unexpected metric: %s", m.Name)
			continue
		}
		if m.Value != expectedValue {
			t.Errorf("metric %s: expected %f, got %f", m.Name, expectedValue, m.Value)
		}
	}
}

func TestTiandyNormalize_NoMetrics(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{
		"eventName": "SystemStart",
		"channel": 1,
		"status": "normal"
	}`)

	event, err := n.Normalize("event", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Source != "tiandy" {
		t.Errorf("expected source 'tiandy', got %q", event.Source)
	}
	if len(event.Metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(event.Metrics))
	}
}

func TestTiandyNormalize_InvalidJSON(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{invalid`)

	_, err := n.Normalize("alarm", payload)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTiandyNormalize_EmptyPayload(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error for empty payload, got %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Source != "tiandy" {
		t.Errorf("expected source 'tiandy', got %q", event.Source)
	}
}

func TestTiandyNormalize_NormalStatus(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{
		"eventName": "DiskCheck",
		"channel": 1,
		"status": "normal"
	}`)

	event, err := n.Normalize("log", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Severity != "info" {
		t.Errorf("expected severity 'info' for normal status, got %q", event.Severity)
	}
}

func TestTiandyNormalize_PartialMetrics(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{
		"eventName": "Performance",
		"channel": 1,
		"status": "normal",
		"cpuRate": 50.0,
		"memoryRate": 60.0
	}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(event.Metrics) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(event.Metrics))
	}

	// Verify cpu metric
	if event.Metrics[0].Name != "cpu_usage" || event.Metrics[0].Value != 50.0 {
		t.Errorf("expected cpu_usage=50, got %s=%f", event.Metrics[0].Name, event.Metrics[0].Value)
	}
}

func TestTiandyNormalize_Tags(t *testing.T) {
	n := &TiandyNormalizer{}
	payload := json.RawMessage(`{"eventName": "Test", "channel": 1, "status": "normal"}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Tags == nil {
		t.Fatal("expected non-nil tags")
	}
	if event.Tags["vendor"] != "tiandy" {
		t.Errorf("expected vendor tag 'tiandy', got %q", event.Tags["vendor"])
	}
	if event.Tags["type"] != "telemetry" {
		t.Errorf("expected type tag 'telemetry', got %q", event.Tags["type"])
	}
}
