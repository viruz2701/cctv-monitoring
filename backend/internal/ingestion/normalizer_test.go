// Package ingestion — unit tests for Vendor Normalizer.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5.1: Input validation (whitelist — KnownVendors)
package ingestion

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ── Tests: KnownVendors ──────────────────────────────────────────────────

func TestKnownVendors_AllRegistered(t *testing.T) {
	expected := []string{"hikvision", "dahua", "onvif", "tiandy", "uniview", "tantos"}
	for _, ev := range expected {
		found := false
		for _, kv := range KnownVendors {
			if kv == ev {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected vendor %q to be in KnownVendors", ev)
		}
	}
}

func TestKnownVendors_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool)
	for _, kv := range KnownVendors {
		if seen[kv] {
			t.Errorf("duplicate vendor %q in KnownVendors", kv)
		}
		seen[kv] = true
	}
}

// ── Tests: NewVendorNormalizer ───────────────────────────────────────────

func TestNewVendorNormalizer_AllVendorsRegistered(t *testing.T) {
	n := NewVendorNormalizer(slog.Default())
	if n == nil {
		t.Fatal("expected non-nil VendorNormalizer")
	}

	for _, vendor := range KnownVendors {
		_, ok := n.registry[strings.ToLower(vendor)]
		if !ok {
			t.Errorf("expected vendor %q to be registered", vendor)
		}
	}
}

func TestNewVendorNormalizer_NilLogger(t *testing.T) {
	n := NewVendorNormalizer(nil)
	if n == nil {
		t.Fatal("expected non-nil VendorNormalizer with nil logger")
	}
	if n.logger == nil {
		t.Error("expected non-nil logger after NewVendorNormalizer(nil)")
	}
}

// ── Tests: VendorNormalizer.Normalize ────────────────────────────────────

func TestNormalize_KnownVendor(t *testing.T) {
	n := NewVendorNormalizer(slog.Default())
	payload := json.RawMessage(`{"eventName": "motion", "channel": 1, "status": "alarm"}`)

	event, err := n.Normalize("alarm", "tiandy", payload)
	if err != nil {
		t.Fatalf("expected no error for known vendor, got %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Source != "tiandy" {
		t.Errorf("expected source 'tiandy', got %q", event.Source)
	}
}

func TestNormalize_UnknownVendor_UsesDefault(t *testing.T) {
	n := NewVendorNormalizer(slog.Default())
	payload := json.RawMessage(`{"type": "test", "severity": "low", "message": "test message"}`)

	event, err := n.Normalize("telemetry", "unknown_vendor", payload)
	if err != nil {
		t.Fatalf("expected no error for unknown vendor (uses default), got %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
	if event.Type != "test" {
		t.Errorf("expected type 'test', got %q", event.Type)
	}
	if event.Severity != "low" {
		t.Errorf("expected severity 'low', got %q", event.Severity)
	}
	if event.Message != "test message" {
		t.Errorf("expected message 'test message', got %q", event.Message)
	}
}

func TestNormalize_EmptyVendor(t *testing.T) {
	n := NewVendorNormalizer(slog.Default())
	payload := json.RawMessage(`{"type": "test"}`)

	event, err := n.Normalize("telemetry", "", payload)
	if err != nil {
		t.Fatalf("expected no error for empty vendor, got %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
}

func TestNormalize_InvalidPayload(t *testing.T) {
	n := NewVendorNormalizer(slog.Default())
	payload := json.RawMessage(`{invalid json`)

	_, err := n.Normalize("telemetry", "hikvision", payload)
	if err == nil {
		t.Error("expected error for invalid JSON payload")
	}
}

func TestNormalize_EmptyPayload(t *testing.T) {
	n := NewVendorNormalizer(slog.Default())
	payload := json.RawMessage(`{}`)

	event, err := n.Normalize("telemetry", "tiandy", payload)
	if err != nil {
		t.Fatalf("expected no error for empty payload, got %v", err)
	}
	if event == nil {
		t.Fatal("expected non-nil event")
	}
}

// ── Tests: DefaultNormalize ──────────────────────────────────────────────

func TestDefaultNormalize_Basic(t *testing.T) {
	payload := json.RawMessage(`{
		"type": "heartbeat",
		"severity": "info",
		"message": "Device is alive",
		"timestamp": "2026-06-30T12:00:00Z"
	}`)

	event, err := DefaultNormalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Type != "heartbeat" {
		t.Errorf("expected type 'heartbeat', got %q", event.Type)
	}
	if event.Severity != "info" {
		t.Errorf("expected severity 'info', got %q", event.Severity)
	}
	if event.Message != "Device is alive" {
		t.Errorf("expected message 'Device is alive', got %q", event.Message)
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.Source != "edge_unknown" {
		t.Errorf("expected source 'edge_unknown', got %q", event.Source)
	}
}

func TestDefaultNormalize_MissingTimestamp(t *testing.T) {
	payload := json.RawMessage(`{"type": "test"}`)

	event, err := DefaultNormalize("log", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.Timestamp.IsZero() {
		t.Error("expected timestamp to default to now, not zero")
	}
}

func TestDefaultNormalize_InvalidJSON(t *testing.T) {
	payload := json.RawMessage(`{invalid`)

	_, err := DefaultNormalize("telemetry", payload)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDefaultNormalize_SetsRawPayload(t *testing.T) {
	jsonStr := `{"type": "test", "value": 42}`
	payload := json.RawMessage(jsonStr)

	event, err := DefaultNormalize("telemetry", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if event.RawPayload != jsonStr {
		t.Errorf("expected raw_payload %q, got %q", jsonStr, event.RawPayload)
	}
}

// ── Tests: parseTimeOrDefault ────────────────────────────────────────────

func TestParseTimeOrDefault_RFC3339(t *testing.T) {
	expected := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	result := parseTimeOrDefault("2026-06-30T12:00:00Z", time.Time{})
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestParseTimeOrDefault_RFC3339Nano(t *testing.T) {
	expected := time.Date(2026, 6, 30, 12, 0, 0, 123456789, time.UTC)
	result := parseTimeOrDefault("2026-06-30T12:00:00.123456789Z", time.Time{})
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestParseTimeOrDefault_AlternativeFormat(t *testing.T) {
	result := parseTimeOrDefault("2026-06-30T12:00:05", time.Time{})
	if result.IsZero() {
		t.Error("expected parsed time, got zero")
	}
}

func TestParseTimeOrDefault_DateOnly(t *testing.T) {
	result := parseTimeOrDefault("2026-06-30", time.Time{})
	if result.IsZero() {
		t.Error("expected parsed date, got zero")
	}
	if result.Year() != 2026 || result.Month() != 6 || result.Day() != 30 {
		t.Errorf("expected 2026-06-30, got %v", result)
	}
}

func TestParseTimeOrDefault_Empty(t *testing.T) {
	defaultTime := time.Now()
	result := parseTimeOrDefault("", defaultTime)
	if !result.Equal(defaultTime) {
		t.Errorf("expected default time, got %v", result)
	}
}

func TestParseTimeOrDefault_Invalid(t *testing.T) {
	defaultTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	result := parseTimeOrDefault("not-a-date", defaultTime)
	if !result.Equal(defaultTime) {
		t.Errorf("expected default time on invalid input, got %v", result)
	}
}

// ── Tests: Event struct ──────────────────────────────────────────────────

func TestEvent_JSONSerialization(t *testing.T) {
	event := &models.Event{
		Type:      "motion",
		Severity:  "high",
		Source:    "tiandy",
		Message:   "Motion detected",
		Timestamp: time.Now(),
		Metrics: []models.Metric{
			{Name: "temperature", Value: 45.2, Unit: "celsius"},
		},
		Tags: map[string]string{
			"vendor": "tiandy",
			"type":   "alarm",
		},
		Metadata: map[string]string{
			"channel": "1",
		},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	var decoded models.Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal event: %v", err)
	}

	if decoded.Type != "motion" {
		t.Errorf("expected type 'motion', got %q", decoded.Type)
	}
	if len(decoded.Metrics) != 1 {
		t.Errorf("expected 1 metric, got %d", len(decoded.Metrics))
	}
	if decoded.Tags["vendor"] != "tiandy" {
		t.Errorf("expected vendor tag 'tiandy', got %q", decoded.Tags["vendor"])
	}
}
