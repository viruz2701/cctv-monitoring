// Package events — тесты для ValidatedPublisher и JSON Schema validation.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist validation testing)
//   - OWASP ASVS V5.3 (Input validation — structured data validation testing)
//   - ISO 27001 A.12.4.1 (Event logging — data quality enforcement testing)
package events

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: JSON Schema Validation (расширенные)
// ═══════════════════════════════════════════════════════════════════════

func TestSchemaRegistry_ValidateJSONSchema_AlarmCreated(t *testing.T) {
	r := NewSchemaRegistry(nil)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid alarm with all fields",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","device_name":"Camera-1","type":"motion","severity":"high","message":"Motion detected","image_url":"https://example.com/snapshot.jpg"}`,
			wantErr: false,
		},
		{
			name:    "valid alarm minimal fields",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"tamper","severity":"critical","message":"Tamper alert"}`,
			wantErr: false,
		},
		{
			name:    "missing required device_id",
			data:    `{"type":"motion","severity":"high","message":"test"}`,
			wantErr: true,
		},
		{
			name:    "missing required type",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","severity":"high","message":"test"}`,
			wantErr: true,
		},
		{
			name:    "missing required severity",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","message":"test"}`,
			wantErr: true,
		},
		{
			name:    "missing required message",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high"}`,
			wantErr: true,
		},
		{
			name:    "invalid severity enum value",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"unknown","message":"test"}`,
			wantErr: true,
		},
		{
			name:    "invalid type enum value",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"nonexistent","severity":"high","message":"test"}`,
			wantErr: true,
		},
		{
			name:    "severity as number instead of string",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":123,"message":"test"}`,
			wantErr: true,
		},
		{
			name:    "device_id not a string",
			data:    `{"device_id":12345,"type":"motion","severity":"high","message":"test"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourceAlarms,
				EventType: "alarm.created",
				Timestamp: time.Now(),
				Data:      json.RawMessage(tt.data),
			}
			err := r.Validate(record)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaRegistry_ValidateJSONSchema_CMMSWorkOrder(t *testing.T) {
	r := NewSchemaRegistry(nil)

	type testCase struct {
		name      string
		eventType string
		data      string
		wantErr   bool
	}

	tests := []testCase{
		{
			name:      "valid work order created",
			eventType: "cmms.wo.created",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","device_id":"0190abcd-1234-7000-8000-000000000002","title":"Replace faulty camera","type":"corrective","priority":"critical"}`,
			wantErr:   false,
		},
		{
			name:      "valid work order with assignee",
			eventType: "cmms.wo.created",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","device_id":"0190abcd-1234-7000-8000-000000000002","title":"Routine inspection","type":"routine","priority":"low","assignee_id":"0190abcd-1234-7000-8000-000000000004"}`,
			wantErr:   false,
		},
		{
			name:      "missing work_order_id",
			eventType: "cmms.wo.created",
			data:      `{"device_id":"0190abcd-1234-7000-8000-000000000002","title":"Test","type":"preventive","priority":"medium"}`,
			wantErr:   true,
		},
		{
			name:      "missing device_id",
			eventType: "cmms.wo.created",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","title":"Test","type":"preventive","priority":"medium"}`,
			wantErr:   true,
		},
		{
			name:      "missing title",
			eventType: "cmms.wo.created",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","device_id":"0190abcd-1234-7000-8000-000000000002","type":"preventive","priority":"medium"}`,
			wantErr:   true,
		},
		{
			name:      "invalid type enum",
			eventType: "cmms.wo.created",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","device_id":"0190abcd-1234-7000-8000-000000000002","title":"Test","type":"invalid_type","priority":"medium"}`,
			wantErr:   true,
		},
		{
			name:      "invalid priority enum",
			eventType: "cmms.wo.created",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","device_id":"0190abcd-1234-7000-8000-000000000002","title":"Test","type":"preventive","priority":"urgent"}`,
			wantErr:   true,
		},
		{
			name:      "valid work order completed",
			eventType: "cmms.wo.completed",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","completed_by":"0190abcd-1234-7000-8000-000000000004","notes":"Replaced successfully","actual_cost":150.50}`,
			wantErr:   false,
		},
		{
			name:      "work order completed missing completed_by",
			eventType: "cmms.wo.completed",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","notes":"Done"}`,
			wantErr:   true,
		},
		{
			name:      "work order completed negative cost",
			eventType: "cmms.wo.completed",
			data:      `{"work_order_id":"0190abcd-1234-7000-8000-000000000003","completed_by":"0190abcd-1234-7000-8000-000000000004","actual_cost":-10}`,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourceCMMS,
				EventType: tt.eventType,
				Timestamp: time.Now(),
				Data:      json.RawMessage(tt.data),
			}
			err := r.Validate(record)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaRegistry_ValidateJSONSchema_Predictions(t *testing.T) {
	r := NewSchemaRegistry(nil)

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid prediction",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure","probability":0.85,"estimated_days":30,"recommendation":"Replace disk"}`,
			wantErr: false,
		},
		{
			name:    "valid prediction minimal",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"overheating","probability":0.42}`,
			wantErr: false,
		},
		{
			name:    "missing device_id",
			data:    `{"failure_mode":"disk_failure","probability":0.85}`,
			wantErr: true,
		},
		{
			name:    "missing failure_mode",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","probability":0.85}`,
			wantErr: true,
		},
		{
			name:    "missing probability",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure"}`,
			wantErr: true,
		},
		{
			name:    "probability over 1.0",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure","probability":1.5}`,
			wantErr: true,
		},
		{
			name:    "probability negative",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure","probability":-0.1}`,
			wantErr: true,
		},
		{
			name:    "estimated_days negative",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure","probability":0.5,"estimated_days":-1}`,
			wantErr: true,
		},
		{
			name:    "probability as string instead of number",
			data:    `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure","probability":"high"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourcePredictions,
				EventType: "prediction.created",
				Timestamp: time.Now(),
				Data:      json.RawMessage(tt.data),
			}
			err := r.Validate(record)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: ValidationStats
// ═══════════════════════════════════════════════════════════════════════

func TestValidationStats_Snapshot(t *testing.T) {
	stats := &ValidationStats{}
	stats.TotalValidations.Add(100)
	stats.ValidEvents.Add(80)
	stats.InvalidEvents.Add(15)
	stats.SchemaNotFound.Add(3)
	stats.ValidationErrors.Add(2)

	snapshot := stats.Snapshot()

	if snapshot["total_validations"] != 100 {
		t.Errorf("expected 100 total_validations, got %d", snapshot["total_validations"])
	}
	if snapshot["valid_events"] != 80 {
		t.Errorf("expected 80 valid_events, got %d", snapshot["valid_events"])
	}
	if snapshot["invalid_events"] != 15 {
		t.Errorf("expected 15 invalid_events, got %d", snapshot["invalid_events"])
	}
	if snapshot["schema_not_found"] != 3 {
		t.Errorf("expected 3 schema_not_found, got %d", snapshot["schema_not_found"])
	}
	if snapshot["validation_errors"] != 2 {
		t.Errorf("expected 2 validation_errors, got %d", snapshot["validation_errors"])
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: Helper functions
// ═══════════════════════════════════════════════════════════════════════

func TestEventToRecord(t *testing.T) {
	// Override timeNow для детерминированного теста
	fixedTime := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	timeNow = func() time.Time { return fixedTime }
	defer func() { timeNow = func() time.Time { return time.Now().UTC() } }()

	alarm := AlarmEvent{
		DeviceID: "0190abcd-1234-7000-8000-000000000001",
		Type:     "motion",
		Severity: "high",
		Message:  "Motion detected",
	}

	record := eventToRecord(SourceAlarms, "alarm.created", alarm)

	if record.Source != SourceAlarms {
		t.Errorf("expected SourceAlarms, got %s", record.Source)
	}
	if record.EventType != "alarm.created" {
		t.Errorf("expected alarm.created, got %s", record.EventType)
	}
	if record.SchemaVersion != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", record.SchemaVersion)
	}
	if !record.Timestamp.Equal(fixedTime) {
		t.Errorf("expected %v, got %v", fixedTime, record.Timestamp)
	}
	if record.Data == nil {
		t.Fatal("expected non-nil data")
	}

	// Проверяем что данные корректно сериализовались
	var parsed map[string]interface{}
	if err := json.Unmarshal(record.Data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal data: %v", err)
	}
	if parsed["device_id"] != "0190abcd-1234-7000-8000-000000000001" {
		t.Errorf("expected device_id in data, got %v", parsed["device_id"])
	}
}

func TestSubjectForRecord(t *testing.T) {
	tests := []struct {
		name     string
		record   *EventRecord
		expected string
	}{
		{
			name: "alarm event",
			record: &EventRecord{
				Source:      SourceAlarms,
				AggregateID: "device-001",
			},
			expected: "alarms.device-001",
		},
		{
			name: "cmms event",
			record: &EventRecord{
				Source:    SourceCMMS,
				EventType: "cmms.wo.created",
			},
			expected: "cmms.workorder.cmms.wo.created",
		},
		{
			name: "prediction event",
			record: &EventRecord{
				Source:      SourcePredictions,
				AggregateID: "device-002",
			},
			expected: "predictions.device-002",
		},
		{
			name: "telemetry event",
			record: &EventRecord{
				Source:      SourceTelemetry,
				AggregateID: "device-003",
			},
			expected: "telemetry.device-003",
		},
		{
			name: "unknown source",
			record: &EventRecord{
				Source:    SourceSystem,
				EventType: "system.startup",
			},
			expected: "events.system.system.startup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := subjectForRecord(tt.record)
			if result != tt.expected {
				t.Errorf("subjectForRecord() = %s, want %s", result, tt.expected)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: ValidatedPublisher (без реального NATS)
// ═══════════════════════════════════════════════════════════════════════

func TestNewValidatedPublisher(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	registry := NewSchemaRegistry(nil)

	// Без Publisher — только проверяем создание структуры
	vp := &ValidatedPublisher{
		registry: registry,
		logger:   logger,
		stats:    &ValidationStats{},
		enabled:  true,
	}

	if vp.Registry() != registry {
		t.Error("Registry() should return the same registry")
	}
	if vp.Stats() == nil {
		t.Error("Stats() should not return nil")
	}

	// Проверяем SetEnabled
	vp.SetEnabled(false)
	vp.SetEnabled(true)
}

func TestValidatedPublisher_StatsSnapshot(t *testing.T) {
	vp := &ValidatedPublisher{
		stats: &ValidationStats{},
	}

	vp.stats.TotalValidations.Add(42)
	vp.stats.ValidEvents.Add(30)
	vp.stats.InvalidEvents.Add(10)
	vp.stats.SchemaNotFound.Add(2)

	snapshot := vp.Stats().Snapshot()
	if snapshot["total_validations"] != 42 {
		t.Errorf("expected 42, got %d", snapshot["total_validations"])
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Compliance Tests: JSON Schema Validation
// ═══════════════════════════════════════════════════════════════════════

// TestCompliance_SchemaValidationAllTypes проверяет что все типы событий
// имеют корректные JSON Schema определения.
func TestCompliance_SchemaValidationAllTypes(t *testing.T) {
	r := NewSchemaRegistry(nil)

	// Проверяем что для каждой required схемы есть валидный пример данных
	validSamples := map[string]string{
		"alarms.alarm.created":           `{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high","message":"test"}`,
		"alarms.alarm.resolved":          `{"alarm_id":"0190abcd-1234-7000-8000-000000000003","resolved_by":"tech-001","resolution":"Fixed"}`,
		"cmms.cmms.wo.created":           `{"work_order_id":"0190abcd-1234-7000-8000-000000000004","device_id":"0190abcd-1234-7000-8000-000000000002","title":"Test","type":"preventive","priority":"medium"}`,
		"cmms.cmms.wo.completed":         `{"work_order_id":"0190abcd-1234-7000-8000-000000000004","completed_by":"0190abcd-1234-7000-8000-000000000005"}`,
		"cmms.cmms.wo.status_changed":    `{"work_order_id":"0190abcd-1234-7000-8000-000000000004","from_status":"open","to_status":"in_progress"}`,
		"predictions.prediction.created": `{"device_id":"0190abcd-1234-7000-8000-000000000002","failure_mode":"disk_failure","probability":0.85}`,
		"audit.audit.access":             `{"user_id":"0190abcd-1234-7000-8000-000000000005","action":"login","resource":"/api/devices"}`,
	}

	for _, def := range r.ListSchemas() {
		key := string(def.Source) + "." + def.EventType
		sample, ok := validSamples[key]
		if !ok {
			continue
		}

		record := &EventRecord{
			ID:        "0190abcd-1234-7000-8000-000000000001",
			Source:    def.Source,
			EventType: def.EventType,
			Timestamp: time.Now(),
			Data:      json.RawMessage(sample),
		}

		if err := r.Validate(record); err != nil {
			t.Errorf("compliance: %s validation failed with valid sample: %v", key, err)
		}
	}
}
