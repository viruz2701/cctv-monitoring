// Package ingestion — unit tests for Tantos Vendor Normalizer.
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

// ── Tests: TantosNormalizer.Normalize ────────────────────────────────────

func TestTantosNormalize_CriticalAlarm(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{
		"eventType": "MOTION_DETECTED",
		"deviceName": "Camera-12",
		"deviceId": "cam-12",
		"severity": "critical",
		"description": "Motion detected in restricted area",
		"source": "video_analytics",
		"timestamp": "2026-06-30T12:00:00Z",
		"imageUrl": "http://example.com/snapshot.jpg"
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}

	if !strings.EqualFold(event.Type, "MOTION_DETECTED") {
		t.Errorf("expected type 'MOTION_DETECTED', got %q", event.Type)
	}
	if event.Severity != "critical" {
		t.Errorf("expected severity 'critical', got %q", event.Severity)
	}
	if event.Source != "tantos" {
		t.Errorf("expected source 'tantos', got %q", event.Source)
	}
	if event.Message != "Motion detected in restricted area" {
		t.Errorf("expected message, got %q", event.Message)
	}
	if event.ImageURL != "http://example.com/snapshot.jpg" {
		t.Errorf("expected image URL, got %q", event.ImageURL)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Metadata["device_name"] != "Camera-12" {
		t.Errorf("expected device_name 'Camera-12', got %q", event.Metadata["device_name"])
	}
	if event.Metadata["device_id"] != "cam-12" {
		t.Errorf("expected device_id 'cam-12', got %q", event.Metadata["device_id"])
	}
	if event.Metadata["source"] != "video_analytics" {
		t.Errorf("expected source metadata 'video_analytics', got %q", event.Metadata["source"])
	}
}

func TestTantosNormalize_TelemetryWithMetrics(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{
		"eventType": "PERFORMANCE",
		"deviceName": "NVR-03",
		"deviceId": "nvr-03",
		"severity": "info",
		"description": "Performance metrics",
		"source": "system",
		"temperature": 42.0,
		"cpu": 55.3,
		"memory": 72.8,
		"network": 1000000,
		"storage": 65.4
	}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedMetrics := map[string]float64{
		"temperature":   42.0,
		"cpu_usage":     55.3,
		"memory_usage":  72.8,
		"network_usage": 1000000,
		"storage_usage": 65.4,
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

func TestTantosNormalize_WarningSeverity(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{
		"eventType": "TEMP_HIGH",
		"deviceName": "Camera-12",
		"deviceId": "cam-12",
		"severity": "warning",
		"description": "Temperature above threshold",
		"source": "sensor",
		"temperature": 48.0
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Severity != "warning" {
		t.Errorf("expected severity 'warning', got %q", event.Severity)
	}
	if len(event.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(event.Metrics))
	}
	if event.Metrics[0].Name != "temperature" {
		t.Errorf("expected temperature metric, got %s", event.Metrics[0].Name)
	}
}

func TestTantosNormalize_PartialMetrics(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{
		"eventType": "CPU_HIGH",
		"deviceName": "NVR-03",
		"deviceId": "nvr-03",
		"severity": "warning",
		"description": "CPU usage high",
		"source": "system",
		"cpu": 88.5
	}`)

	event, err := n.Normalize("alarm", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(event.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(event.Metrics))
	}
	if event.Metrics[0].Value != 88.5 {
		t.Errorf("expected 88.5, got %f", event.Metrics[0].Value)
	}
}

func TestTantosNormalize_NoMetrics(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{
		"eventType": "SYSTEM_RESTART",
		"deviceName": "NVR-03",
		"deviceId": "nvr-03",
		"severity": "info",
		"description": "System restarted",
		"source": "system"
	}`)

	event, err := n.Normalize("event", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(event.Metrics) != 0 {
		t.Errorf("expected 0 metrics, got %d", len(event.Metrics))
	}
}

func TestTantosNormalize_InvalidJSON(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{invalid`)

	_, err := n.Normalize("alarm", payload)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestTantosNormalize_EmptyPayload(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error for empty payload, got %v", err)
	}

	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Source != "tantos" {
		t.Errorf("expected source 'tantos', got %q", event.Source)
	}
}

func TestTantosNormalize_Tags(t *testing.T) {
	n := &TantosNormalizer{}
	payload := json.RawMessage(`{"eventType": "TEST", "deviceName": "Test", "deviceId": "test", "severity": "info", "description": "test", "source": "test"}`)

	event, err := n.Normalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Tags["vendor"] != "tantos" {
		t.Errorf("expected vendor tag 'tantos', got %q", event.Tags["vendor"])
	}
	if event.Tags["type"] != "telemetry" {
		t.Errorf("expected type tag 'telemetry', got %q", event.Tags["type"])
	}
}
