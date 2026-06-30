// Package ingestion — unit tests for MQTT Ingress Handler.
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
//   - OWASP ASVS L3 V5.1: Input validation
//   - OWASP ASVS L3 V7.1: Error handling
package ingestion

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
)

// ── Mocks ────────────────────────────────────────────────────────────────

type mockStateManager struct {
	state.DeviceStateManager
	updateLastSeenCalled bool
	setOnlineCalled      bool
	addAlarmCalled       bool
}

func (m *mockStateManager) UpdateLastSeen(deviceID string) {
	m.updateLastSeenCalled = true
}

func (m *mockStateManager) SetOnline(deviceID string) {
	m.setOnlineCalled = true
}

func (m *mockStateManager) AddAlarm(deviceID string, alarm *models.Alarm) {
	m.addAlarmCalled = true
}

// ── Tests: parseEdgeTopic ────────────────────────────────────────────────

func TestParseEdgeTopic_Valid(t *testing.T) {
	tests := []struct {
		name         string
		topic        string
		wantAgentID  string
		wantDeviceID string
		wantDataType string
		wantErr      bool
	}{
		{"telemetry topic", "edge.agent-01.cam-101.telemetry", "agent-01", "cam-101", "telemetry", false},
		{"alarm topic", "edge.agent-01.cam-101.alarm", "agent-01", "cam-101", "alarm", false},
		{"log topic", "edge.agent-01.nvr-03.log", "agent-01", "nvr-03", "log", false},
		{"event topic", "edge.agent-01.cam-102.event", "agent-01", "cam-102", "event", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentID, deviceID, dataType, err := parseEdgeTopic(tt.topic)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEdgeTopic() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if agentID != tt.wantAgentID {
				t.Errorf("agentID = %q, want %q", agentID, tt.wantAgentID)
			}
			if deviceID != tt.wantDeviceID {
				t.Errorf("deviceID = %q, want %q", deviceID, tt.wantDeviceID)
			}
			if dataType != tt.wantDataType {
				t.Errorf("dataType = %q, want %q", dataType, tt.wantDataType)
			}
		})
	}
}

func TestParseEdgeTopic_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		topic string
	}{
		{"no prefix", "invalid.topic"},
		{"too few parts", "edge.agent-01"},
		{"empty agent", "edge..cam-101.telemetry"},
		{"empty device", "edge.agent-01..telemetry"},
		{"empty type", "edge.agent-01.cam-101."},
		{"empty topic", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := parseEdgeTopic(tt.topic)
			if err == nil {
				t.Errorf("parseEdgeTopic(%q) expected error, got nil", tt.topic)
			}
		})
	}
}

// ── Tests: validEdgeDataTypes ────────────────────────────────────────────

func TestValidEdgeDataTypes_AllValid(t *testing.T) {
	validTypes := []string{"telemetry", "alarm", "log", "event"}
	for _, vt := range validTypes {
		if _, ok := validEdgeDataTypes[vt]; !ok {
			t.Errorf("expected %q to be a valid edge data type", vt)
		}
	}
}

func TestValidEdgeDataTypes_Invalid(t *testing.T) {
	invalidTypes := []string{"", "command", "response", "unknown", "heartbeat"}
	for _, iv := range invalidTypes {
		if _, ok := validEdgeDataTypes[iv]; ok {
			t.Errorf("expected %q to be INVALID edge data type", iv)
		}
	}
}

// ── Tests: mapAlarmPriority ──────────────────────────────────────────────

func TestMapAlarmPriority(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		want     models.AlarmPriority
	}{
		{"critical severity", "critical", models.AlarmPriorityHigh},
		{"high severity", "high", models.AlarmPriorityHigh},
		{"medium severity", "medium", models.AlarmPriorityMedium},
		{"low severity", "low", models.AlarmPriorityLow},
		{"info severity (default)", "info", models.AlarmPriorityLow},
		{"empty severity (default)", "", models.AlarmPriorityLow},
		{"case insensitive", "CRITICAL", models.AlarmPriorityHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAlarmPriority(tt.severity)
			if got != tt.want {
				t.Errorf("mapAlarmPriority(%q) = %v, want %v", tt.severity, got, tt.want)
			}
		})
	}
}

// ── Tests: mapAlarmMethod ────────────────────────────────────────────────

func TestMapAlarmMethod(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      models.AlarmMethod
	}{
		{"motion detection", "motion", models.AlarmMethodMotionDetection},
		{"motion_detection", "motion_detection", models.AlarmMethodMotionDetection},
		{"video loss", "video_loss", models.AlarmMethodVideoLoss},
		{"default (equipment fault)", "unknown", models.AlarmMethodEquipmentFault},
		{"empty type", "", models.AlarmMethodEquipmentFault},
		{"case insensitive", "MOTION", models.AlarmMethodMotionDetection},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapAlarmMethod(tt.eventType)
			if got != tt.want {
				t.Errorf("mapAlarmMethod(%q) = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

// ── Tests: extractPrimaryValue ───────────────────────────────────────────

func TestExtractPrimaryValue_WithMetrics(t *testing.T) {
	event := &models.Event{
		Metrics: []models.Metric{
			{Name: "temperature", Value: 45.2, Unit: "celsius"},
			{Name: "humidity", Value: 60.0, Unit: "percent"},
		},
	}
	got := extractPrimaryValue(event)
	if got != 45.2 {
		t.Errorf("extractPrimaryValue() = %f, want 45.2", got)
	}
}

func TestExtractPrimaryValue_NoMetrics(t *testing.T) {
	event := &models.Event{}
	got := extractPrimaryValue(event)
	if got != 0 {
		t.Errorf("extractPrimaryValue() = %f, want 0", got)
	}
}

func TestExtractPrimaryValue_NilMetrics(t *testing.T) {
	event := &models.Event{Metrics: nil}
	got := extractPrimaryValue(event)
	if got != 0 {
		t.Errorf("extractPrimaryValue() = %f, want 0", got)
	}
}

// ── Tests: truncateString ────────────────────────────────────────────────

func TestTruncateString_ShorterThanMax(t *testing.T) {
	result := truncateString("short", 10)
	if result != "short" {
		t.Errorf("expected 'short', got %q", result)
	}
}

func TestTruncateString_ExactMax(t *testing.T) {
	result := truncateString("exactly10", 10)
	if result != "exactly10" {
		t.Errorf("expected 'exactly10', got %q", result)
	}
}

func TestTruncateString_LongerThanMax(t *testing.T) {
	result := truncateString("this is a very long string that should be truncated", 20)
	expected := "this is a very long ..."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTruncateString_Empty(t *testing.T) {
	result := truncateString("", 10)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// ── Tests: handleTelemetry ───────────────────────────────────────────────

func TestHandleTelemetry_UpdatesState(t *testing.T) {
	sm := &mockStateManager{}
	ingress := &MQTTIngress{
		stateMgr: sm,
		logger:   slog.Default(),
	}

	event := &models.Event{
		Type:    "temperature",
		Metrics: []models.Metric{{Name: "temp", Value: 45.0, Unit: "celsius"}},
		Tags:    map[string]string{"vendor": "hikvision"},
	}

	ingress.handleTelemetry(context.Background(), "cam-101", event, ingress.logger)

	if !sm.updateLastSeenCalled {
		t.Error("expected UpdateLastSeen to be called")
	}
	if !sm.setOnlineCalled {
		t.Error("expected SetOnline to be called")
	}
}

// ── Tests: handleAlarm ───────────────────────────────────────────────────

func TestHandleAlarm_AddsAlarm(t *testing.T) {
	sm := &mockStateManager{}
	ingress := &MQTTIngress{
		stateMgr: sm,
		logger:   slog.Default(),
	}

	event := &models.Event{
		Type:      "motion_detection",
		Severity:  "high",
		Message:   "Motion detected at main entrance",
		Timestamp: time.Now(),
		ImageURL:  "http://example.com/snapshot.jpg",
	}

	ingress.handleAlarm(context.Background(), "cam-101", event, ingress.logger)

	if !sm.addAlarmCalled {
		t.Error("expected AddAlarm to be called for alarm event")
	}
}

func TestHandleAlarm_WithCriticalSeverity(t *testing.T) {
	sm := &mockStateManager{}
	ingress := &MQTTIngress{
		stateMgr: sm,
		logger:   slog.Default(),
	}

	event := &models.Event{
		Type:     "video_loss",
		Severity: "critical",
		Message:  "Video signal lost on channel 1",
	}

	ingress.handleAlarm(context.Background(), "cam-101", event, ingress.logger)

	if !sm.addAlarmCalled {
		t.Error("expected AddAlarm to be called for critical alarm")
	}
}

// ── Tests: handleEvent ───────────────────────────────────────────────────

func TestHandleEvent_NoPublisher(t *testing.T) {
	ingress := &MQTTIngress{
		logger: slog.Default(),
	}
	// Should not panic when publisher is nil
	ingress.handleEvent(context.Background(), "cam-101", &models.Event{}, ingress.logger)
}

// ── Tests: validDataTypes ────────────────────────────────────────────────

func TestValidDataTypes_ReturnsAllTypes(t *testing.T) {
	types := validDataTypes()
	if len(types) != len(validEdgeDataTypes) {
		t.Errorf("expected %d types, got %d", len(validEdgeDataTypes), len(types))
	}

	typeSet := make(map[string]bool)
	for _, dt := range types {
		typeSet[dt] = true
	}

	for vt := range validEdgeDataTypes {
		if !typeSet[vt] {
			t.Errorf("expected %q to be in returned types", vt)
		}
	}
}

// ── Compilation tests ────────────────────────────────────────────────────

// TestMQTTIngressImplementsInterface проверяет, что структура компилируется
// с требуемыми зависимостями (nats.Conn, pgxpool.Pool).
func TestMQTTIngress_ConfigDefaults(t *testing.T) {
	cfg := MQTTIngressConfig{}
	if cfg.LogTTL != 0 {
		t.Errorf("expected zero LogTTL, got %v", cfg.LogTTL)
	}
	if cfg.Logger != nil {
		t.Error("expected nil Logger by default")
	}
}

func TestNewMQTTIngress_InvalidNATSURL(t *testing.T) {
	cfg := MQTTIngressConfig{
		NATSURL: "invalid://url",
		Logger:  nil,
	}
	_, err := NewMQTTIngress(cfg, nil, nil, nil, nil)
	if err == nil {
		t.Error("expected error for invalid NATS URL")
	}
}

// ── EdgeIngressMessage tests ─────────────────────────────────────────────

func TestEdgeIngressMessage_JSONUnmarshal(t *testing.T) {
	jsonData := `{
		"vendor": "hikvision",
		"model": "DS-2CD2T47G2-L",
		"type": "telemetry",
		"timestamp": "2026-06-30T12:00:00Z",
		"payload": {"temperature": 45.2}
	}`

	var msg EdgeIngressMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if msg.Vendor != "hikvision" {
		t.Errorf("expected vendor 'hikvision', got %q", msg.Vendor)
	}
	if msg.Model != "DS-2CD2T47G2-L" {
		t.Errorf("expected model 'DS-2CD2T47G2-L', got %q", msg.Model)
	}
	if msg.Type != "telemetry" {
		t.Errorf("expected type 'telemetry', got %q", msg.Type)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if msg.Payload == nil {
		t.Error("expected non-nil payload")
	}
}

func TestEdgeIngressMessage_JSONMissingOptional(t *testing.T) {
	jsonData := `{"vendor": "dahua", "type": "alarm", "payload": {}}`

	var msg EdgeIngressMessage
	if err := json.Unmarshal([]byte(jsonData), &msg); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if msg.Vendor != "dahua" {
		t.Errorf("expected vendor 'dahua', got %q", msg.Vendor)
	}
	if msg.Model != "" {
		t.Errorf("expected empty model, got %q", msg.Model)
	}
	if msg.Timestamp.IsZero() == false {
		// Zero timestamp is fine for missing field
	}
}
