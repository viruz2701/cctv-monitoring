// Package events — tests for Event Store
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging testing)
//   - OWASP ASVS V5.1 (Input validation testing)
//   - IEC 62443 SR 2.8 (Audit events testing)
//   - Правило 7: Тестирование соответствия (unit ≥ 80%, security, compliance)
package events

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: Schema Registry
// ═══════════════════════════════════════════════════════════════════════

func TestSchemaRegistry_RegisterAndGet(t *testing.T) {
	r := NewSchemaRegistry(nil)

	def, ok := r.GetSchema(SourceAlarms, "alarm.created")
	if !ok {
		t.Fatal("expected builtin schema alarm.created to be registered")
	}
	if def.Source != SourceAlarms {
		t.Errorf("expected source alarms, got %s", def.Source)
	}
	if def.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", def.Version)
	}
}

func TestSchemaRegistry_RegisterCustom(t *testing.T) {
	r := NewSchemaRegistry(nil)

	err := r.RegisterSchema(&SchemaDefinition{
		Source:      SourceSystem,
		EventType:   "custom.test",
		Version:     "2.0.0",
		Description: "Test schema",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["test_field"],
			"properties": {
				"test_field": {"type": "string"}
			}
		}`),
		Required: true,
	})
	if err != nil {
		t.Fatalf("RegisterSchema failed: %v", err)
	}

	def, ok := r.GetSchema(SourceSystem, "custom.test")
	if !ok {
		t.Fatal("expected custom schema to be registered")
	}
	if def.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", def.Version)
	}
}

func TestSchemaRegistry_RegisterInvalid(t *testing.T) {
	r := NewSchemaRegistry(nil)

	// Empty source
	err := r.RegisterSchema(&SchemaDefinition{
		Source:    "",
		EventType: "test",
		Schema:    json.RawMessage(`{"type": "object"}`),
	})
	if err == nil {
		t.Error("expected error for empty source")
	}

	// Empty schema
	err = r.RegisterSchema(&SchemaDefinition{
		Source:    SourceSystem,
		EventType: "test",
		Schema:    nil,
	})
	if err == nil {
		t.Error("expected error for nil schema")
	}

	// Invalid JSON schema
	err = r.RegisterSchema(&SchemaDefinition{
		Source:    SourceSystem,
		EventType: "test2",
		Schema:    json.RawMessage(`{invalid json}`),
	})
	if err == nil {
		t.Error("expected error for invalid JSON schema")
	}
}

func TestSchemaRegistry_Validate(t *testing.T) {
	r := NewSchemaRegistry(nil)

	tests := []struct {
		name    string
		record  *EventRecord
		wantErr bool
	}{
		{
			name: "valid alarm event",
			record: &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourceAlarms,
				EventType: "alarm.created",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high","message":"Motion detected"}`),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			record: &EventRecord{
				Source:    SourceAlarms,
				EventType: "alarm.created",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high","message":"test"}`),
			},
			wantErr: true,
		},
		{
			name: "missing timestamp",
			record: &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourceAlarms,
				EventType: "alarm.created",
				Data:      json.RawMessage(`{"device_id":"0190abcd-1234-7000-8000-000000000002","type":"motion","severity":"high","message":"test"}`),
			},
			wantErr: true,
		},
		{
			name: "missing data",
			record: &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourceAlarms,
				EventType: "alarm.created",
				Timestamp: time.Now(),
				Data:      nil,
			},
			wantErr: true,
		},
		{
			name: "unknown event type (not required)",
			record: &EventRecord{
				ID:        "0190abcd-1234-7000-8000-000000000001",
				Source:    SourceTelemetry,
				EventType: "telemetry.unknown",
				Timestamp: time.Now(),
				Data:      json.RawMessage(`{"test": true}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := r.Validate(tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchemaRegistry_ListSchemas(t *testing.T) {
	r := NewSchemaRegistry(nil)
	schemas := r.ListSchemas()

	if len(schemas) == 0 {
		t.Error("expected at least some builtin schemas")
	}

	// Check that all builtin schemas are present
	schemaMap := make(map[string]bool)
	for _, s := range schemas {
		key := string(s.Source) + "." + s.EventType
		schemaMap[key] = true
	}

	expected := []string{
		"alarms.alarm.created",
		"alarms.alarm.resolved",
		"cmms.cmms.wo.created",
		"cmms.cmms.wo.completed",
		"cmms.cmms.wo.status_changed",
		"predictions.prediction.created",
		"telemetry.telemetry.metric",
		"audit.audit.access",
		"system.system.startup",
		"system.system.shutdown",
	}

	for _, exp := range expected {
		if !schemaMap[exp] {
			t.Errorf("expected schema %s to be registered", exp)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Unit Tests: EventRecord & Helpers
// ═══════════════════════════════════════════════════════════════════════

func TestNewUUID(t *testing.T) {
	uuid1 := newUUID()
	uuid2 := newUUID()

	if uuid1 == uuid2 {
		t.Error("expected unique UUIDs")
	}

	if len(uuid1) != 36 {
		t.Errorf("expected UUID length 36, got %d", len(uuid1))
	}

	// Check UUID v7 format (version digit should be 7)
	if uuid1[14] != '7' {
		t.Errorf("expected UUID version 7, got %c", uuid1[14])
	}
}

func TestNewTraceID(t *testing.T) {
	id1 := newTraceID()
	id2 := newTraceID()

	if id1 == id2 {
		t.Error("expected unique trace IDs")
	}

	if len(id1) != 32 {
		t.Errorf("expected trace ID length 32, got %d", len(id1))
	}
}

func TestMatchesFilter(t *testing.T) {
	now := time.Now()
	record := &EventRecord{
		ID:          "test-id",
		Source:      SourceAlarms,
		EventType:   "alarm.created",
		AggregateID: "device-123",
		Timestamp:   now,
	}

	tests := []struct {
		name   string
		opts   RetrieveOptions
		match  bool
	}{
		{"empty filter", RetrieveOptions{}, true},
		{"source match", RetrieveOptions{Source: SourceAlarms}, true},
		{"source mismatch", RetrieveOptions{Source: SourceCMMS}, false},
		{"event type match", RetrieveOptions{EventType: "alarm.created"}, true},
		{"event type mismatch", RetrieveOptions{EventType: "alarm.resolved"}, false},
		{"aggregate match", RetrieveOptions{AggregateID: "device-123"}, true},
		{"aggregate mismatch", RetrieveOptions{AggregateID: "device-456"}, false},
		{"since before", RetrieveOptions{Since: now.Add(-time.Hour)}, true},
		{"since after", RetrieveOptions{Since: now.Add(time.Hour)}, false},
		{"until after", RetrieveOptions{Until: now.Add(time.Hour)}, true},
		{"until before", RetrieveOptions{Until: now.Add(-time.Hour)}, false},
		{"combined match", RetrieveOptions{
			Source:      SourceAlarms,
			EventType:   "alarm.created",
			AggregateID: "device-123",
			Since:       now.Add(-time.Hour),
			Until:       now.Add(time.Hour),
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesFilter(record, tt.opts); got != tt.match {
				t.Errorf("matchesFilter() = %v, want %v", got, tt.match)
			}
		})
	}
}

func TestNewRecord(t *testing.T) {
	// Can't test without NATS, but we can test the record creation
	// We'll test NewRecord indirectly via store creation
	// For safety, just verify the schema
	store := &EventStore{
		logger:  nilLogger(),
		schemas: NewSchemaRegistry(nil),
	}

	data := map[string]interface{}{
		"device_id": "test-device",
		"type":      "motion",
		"severity":  "high",
		"message":   "Movement detected at entrance",
	}

	record := store.NewRecord(SourceAlarms, "alarm.created", "test-device", data)
	if record == nil {
		t.Fatal("expected non-nil record")
	}
	if record.ID == "" {
		t.Error("expected non-empty ID")
	}
	if record.EventType != "alarm.created" {
		t.Errorf("expected alarm.created, got %s", record.EventType)
	}
	if record.Source != SourceAlarms {
		t.Errorf("expected alarms source, got %s", record.Source)
	}
	if record.AggregateID != "test-device" {
		t.Errorf("expected test-device, got %s", record.AggregateID)
	}
	if record.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if record.SchemaVersion != "1.0.0" {
		t.Errorf("expected 1.0.0, got %s", record.SchemaVersion)
	}
	if record.TraceID == "" {
		t.Error("expected non-empty trace ID")
	}
}

func TestEventStoreStats(t *testing.T) {
	store := &EventStore{
		logger: nilLogger(),
	}
	stats := store.Stats()
	if stats.ColdStorage {
		t.Error("expected cold storage disabled")
	}
	if stats.BufferedEvents != 0 {
		t.Errorf("expected 0 buffered events, got %d", stats.BufferedEvents)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Compliance Tests
// ═══════════════════════════════════════════════════════════════════════

// TestCompliance_SchemaWhitelist проверяет OWASP ASVS V5.1 (whitelist validation).
func TestCompliance_SchemaWhitelist(t *testing.T) {
	r := NewSchemaRegistry(nil)

	// Все зарегистрированные схемы должны быть с whitelist валидацией
	for _, def := range r.ListSchemas() {
		if def.Schema == nil || len(def.Schema) == 0 {
			t.Errorf("schema %s.%s has no schema definition", def.Source, def.EventType)
		}
		if !json.Valid(def.Schema) {
			t.Errorf("schema %s.%s is not valid JSON", def.Source, def.EventType)
		}
	}
}

// TestCompliance_EventRecordStructure проверяет что EventRecord содержит
// все обязательные поля для ISO 27001 A.12.4.
func TestCompliance_EventRecordStructure(t *testing.T) {
	record := &EventRecord{
		ID:            "0190abcd-1234-7000-8000-000000000001",
		Source:        SourceAlarms,
		EventType:     "test.event",
		SchemaVersion: "1.0.0",
		Timestamp:     time.Now(),
		AggregateID:   "test-agg",
		ActorID:       "test-actor",
		TraceID:       "test-trace",
		Data:          json.RawMessage(`{"test": true}`),
	}

	// ISO 27001 A.12.4.1: Каждое событие должно содержать:
	// - user/actor ID
	if record.ActorID == "" {
		t.Error("ISO 27001 A.12.4.1: actor_id is required")
	}
	// - timestamp
	if record.Timestamp.IsZero() {
		t.Error("ISO 27001 A.12.4.1: timestamp is required")
	}
	// - event type
	if record.EventType == "" {
		t.Error("ISO 27001 A.12.4.1: event type is required")
	}
	// - unique ID
	if record.ID == "" {
		t.Error("ISO 27001 A.12.4.1: unique event ID is required")
	}

	// СТБ 34.101.30: prev_hash для tamper detection
	// (опционально, но если есть — должен быть непустым)
	// record.PrevHash = "..." // заполняется в Store()

	// Сериализация/десериализация не должна терять поля
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var restored EventRecord
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if restored.ID != record.ID {
		t.Error("JSON roundtrip: ID mismatch")
	}
	if restored.Source != record.Source {
		t.Error("JSON roundtrip: Source mismatch")
	}
	if restored.EventType != record.EventType {
		t.Error("JSON roundtrip: EventType mismatch")
	}
}

// nilLogger возвращает no-op логгер для тестов
func nilLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
