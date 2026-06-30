// Package ingestion — unit tests for Uniview Vendor Normalizer.
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

// ── Tests: UniviewNormalizer.Normalize ───────────────────────────────────

func TestUniviewNormalize_BasicEvent(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{
		"eventCode": "VIDEO_LOSS",
		"eventDesc": "Video loss on channel 1",
		"channelId": 1,
		"alarmInput": 2,
		"startTime": "2026-06-30T12:00:00Z"
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if !strings.EqualFold(event.Type, "VIDEO_LOSS") {
		t.Errorf("expected type 'VIDEO_LOSS', got %q", event.Type)
	}
	if event.Source != "uniview" {
		t.Errorf("expected source 'uniview', got %q", event.Source)
	}
	if event.Message != "Video loss on channel 1" {
		t.Errorf("expected message, got %q", event.Message)
	}
	if event.Severity != "medium" {
		t.Errorf("expected severity 'medium', got %q", event.Severity)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Metadata["channel_id"] != "1" {
		t.Errorf("expected channel_id '1', got %q", event.Metadata["channel_id"])
	}
	if event.Metadata["alarm_input"] != "2" {
		t.Errorf("expected alarm_input '2', got %q", event.Metadata["alarm_input"])
	}
}

func TestUniviewNormalize_TelemetryWithMetrics(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{
		"eventCode": "PERFORMANCE",
		"eventDesc": "Performance metrics",
		"channelId": 1,
		"startTime": "2026-06-30T12:00:00Z",
		"temperature": 38.5,
		"cpuLoad": 55.2,
		"memLoad": 70.1,
		"netLoad": 500000,
		"diskLoad": 45.8
	}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedMetrics := map[string]float64{
		"temperature":  38.5,
		"cpu_usage":    55.2,
		"memory_usage": 70.1,
		"network_load": 500000,
		"disk_usage":   45.8,
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

func TestUniviewNormalize_PartialMetrics(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{
		"eventCode": "CPU_HIGH",
		"eventDesc": "CPU usage high",
		"channelId": 1,
		"startTime": "2026-06-30T12:00:00Z",
		"cpuLoad": 90.5
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(event.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(event.Metrics))
	}
	if event.Metrics[0].Name != "cpu_usage" {
		t.Errorf("expected cpu_usage, got %s", event.Metrics[0].Name)
	}
	if event.Metrics[0].Value != 90.5 {
		t.Errorf("expected 90.5, got %f", event.Metrics[0].Value)
	}
}

func TestUniviewNormalize_NoMetrics(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{
		"eventCode": "SYSTEM_START",
		"eventDesc": "System started",
		"channelId": 1,
		"startTime": "2026-06-30T12:00:00Z"
	}`)

	event, err := n.Normalize("event", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(event.Metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(event.Metrics))
	}
}

func TestUniviewNormalize_InvalidJSON(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{invalid`)

	_, err := n.Normalize("alarm", payload)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUniviewNormalize_EmptyPayload(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error for empty payload, got %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Source != "uniview" {
		t.Errorf("expected source 'uniview', got %q", event.Source)
	}
}

func TestUniviewNormalize_DefaultAlarmInput(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{
		"eventCode": "MOTION",
		"eventDesc": "Motion detected",
		"channelId": 2,
		"startTime": "2026-06-30T12:00:00Z"
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Metadata["alarm_input"] != "0" {
		t.Errorf("expected alarm_input '0' for missing field, got %q", event.Metadata["alarm_input"])
	}
}

func TestUniviewNormalize_Tags(t *testing.T) {
	n := &UniviewNormalizer{}
	payload := json.RawMessage(`{"eventCode": "TEST", "eventDesc": "test", "channelId": 1, "startTime": "2026-06-30T12:00:00Z"}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Tags["vendor"] != "uniview" {
		t.Errorf("expected vendor tag 'uniview', got %q", event.Tags["vendor"])
	}
	if event.Tags["type"] != "telemetry" {
		t.Errorf("expected type tag 'telemetry', got %q", event.Tags["type"])
	}
}
