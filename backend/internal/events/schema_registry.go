// Package events — Schema Registry для Event Store.
//
// Регистрация и валидация JSON Schema для всех типов событий.
// Реализует F-0.2.2 (Event Schema Registry) из стратегического плана.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist validation)
//   - OWASP ASVS V5.3 (Input validation — structured data validation)
//   - ISO 27001 A.12.4.1 (Event logging — data quality)
//   - IEC 62443 SR 3.1 (Wireless — data integrity)
package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

// ═══════════════════════════════════════════════════════════════════════
// SchemaRegistry — реестр схем событий.
// ═══════════════════════════════════════════════════════════════════════

// SchemaDefinition описывает JSON Schema для события.
type SchemaDefinition struct {
	Source      EventSource        `json:"source"`      // alarms, cmms, etc.
	EventType   string             `json:"event_type"`  // alarm.created, cmms.wo.completed
	Version     EventSchemaVersion `json:"version"`     // "1.0.0"
	Schema      json.RawMessage    `json:"schema"`      // JSON Schema draft-07
	Description string             `json:"description"` // human-readable description
	Required    bool               `json:"required"`    // обязательна ли валидация
}

// ValidationError — ошибка валидации события.
type ValidationError struct {
	EventType string `json:"event_type"`
	Field     string `json:"field"`
	Message   string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("schema validation: %s.%s: %s", e.EventType, e.Field, e.Message)
}

// SchemaRegistry управляет реестром схем событий.
//
// Позволяет:
//   - Регистрировать JSON Schema для каждого типа событий
//   - Валидировать события перед записью в Event Store
//   - Получать схему по типу события
//   - Экспортировать все схемы для документации
type SchemaRegistry struct {
	schemas map[string]*SchemaDefinition // key: "{source}.{event_type}"
	mu      sync.RWMutex
	logger  *slog.Logger
}

// NewSchemaRegistry создаёт новый SchemaRegistry.
func NewSchemaRegistry(logger *slog.Logger) *SchemaRegistry {
	if logger == nil {
		logger = slog.Default()
	}

	reg := &SchemaRegistry{
		schemas: make(map[string]*SchemaDefinition),
		logger:  logger,
	}

	// Регистрируем встроенные схемы
	reg.registerBuiltin()

	return reg
}

// RegisterSchema регистрирует схему для типа события.
func (r *SchemaRegistry) RegisterSchema(def *SchemaDefinition) error {
	if def.Source == "" {
		return fmt.Errorf("schema source is required")
	}
	if def.EventType == "" {
		return fmt.Errorf("schema event_type is required")
	}
	if def.Version == "" {
		def.Version = "1.0.0"
	}
	if def.Schema == nil || len(def.Schema) == 0 {
		return fmt.Errorf("schema definition is required for %s.%s", def.Source, def.EventType)
	}

	// Проверяем что schema это валидный JSON
	if !json.Valid(def.Schema) {
		return fmt.Errorf("schema is not valid JSON for %s.%s", def.Source, def.EventType)
	}

	key := schemaKey(def.Source, def.EventType)

	r.mu.Lock()
	r.schemas[key] = def
	r.mu.Unlock()

	r.logger.Debug("schema registered",
		"source", def.Source,
		"event_type", def.EventType,
		"version", def.Version,
	)

	return nil
}

// GetSchema возвращает схему для типа события.
func (r *SchemaRegistry) GetSchema(source EventSource, eventType string) (*SchemaDefinition, bool) {
	key := schemaKey(source, eventType)
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.schemas[key]
	return def, ok
}

// Validate проверяет событие на соответствие зарегистрированной схеме.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist)
//   - OWASP ASVS V5.3 (Input validation — structured data)
//   - ISO 27001 A.12.4.1 (Event logging — data quality enforcement)
//
// Если схема не найдена:
//   - Если Required == true: возвращаем ошибку
//   - Если Required == false (default): пропускаем валидацию (log warn)
func (r *SchemaRegistry) Validate(record *EventRecord) error {
	def, ok := r.GetSchema(record.Source, record.EventType)
	if !ok {
		// Схема не зарегистрирована
		r.logger.Warn("schema not found for event",
			"source", record.Source,
			"event_type", record.EventType,
		)
		return nil
	}

	// Базовая валидация обязательных полей EventRecord
	if record.ID == "" {
		return &ValidationError{
			EventType: record.EventType,
			Field:     "id",
			Message:   "event ID is required",
		}
	}
	if record.Timestamp.IsZero() {
		return &ValidationError{
			EventType: record.EventType,
			Field:     "timestamp",
			Message:   "event timestamp is required",
		}
	}
	if record.Data == nil || len(record.Data) == 0 {
		return &ValidationError{
			EventType: record.EventType,
			Field:     "data",
			Message:   "event data is required",
		}
	}

	// Валидация Data по JSON Schema (gojsonschema)
	schemaLoader := gojsonschema.NewStringLoader(string(def.Schema))
	docLoader := gojsonschema.NewBytesLoader(record.Data)

	result, err := gojsonschema.Validate(schemaLoader, docLoader)
	if err != nil {
		return fmt.Errorf("schema validation error for %s.%s: %w",
			record.Source, record.EventType, err)
	}

	if !result.Valid() {
		// Собираем все ошибки валидации
		var errs []string
		for _, desc := range result.Errors() {
			errs = append(errs, desc.String())
		}
		return &ValidationError{
			EventType: record.EventType,
			Field:     "data",
			Message:   fmt.Sprintf("schema validation failed: %s", strings.Join(errs, "; ")),
		}
	}

	return nil
}

// ListSchemas возвращает все зарегистрированные схемы.
func (r *SchemaRegistry) ListSchemas() []*SchemaDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SchemaDefinition, 0, len(r.schemas))
	for _, def := range r.schemas {
		result = append(result, def)
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════
// Built-in schemas
// ═══════════════════════════════════════════════════════════════════════

func (r *SchemaRegistry) registerBuiltin() {
	// ── Alarm events ──────────────────────────────────────────────
	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceAlarms,
		EventType:   "alarm.created",
		Version:     "1.0.0",
		Description: "Событие тревоги от устройства видеонаблюдения",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["device_id", "type", "severity", "message"],
			"properties": {
				"device_id":  {"type": "string", "format": "uuid"},
				"device_name": {"type": "string"},
				"type":       {"type": "string", "enum": ["motion", "tamper", "video_loss", "line_cross", "intrusion", "defocus", "scene_change", "equipment_fault", "other"]},
				"severity":   {"type": "string", "enum": ["critical", "high", "medium", "low"]},
				"message":    {"type": "string", "maxLength": 2000},
				"image_url":  {"type": "string", "format": "uri"}
			}
		}`),
		Required: true,
	})

	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceAlarms,
		EventType:   "alarm.resolved",
		Version:     "1.0.0",
		Description: "Событие снятия тревоги",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["alarm_id", "resolved_by", "resolution"],
			"properties": {
				"alarm_id":    {"type": "string", "format": "uuid"},
				"resolved_by": {"type": "string"},
				"resolution":  {"type": "string", "maxLength": 2000},
				"auto_resolved": {"type": "boolean"}
			}
		}`),
		Required: true,
	})

	// ── CMMS Work Order events ───────────────────────────────────
	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceCMMS,
		EventType:   "cmms.wo.created",
		Version:     "1.0.0",
		Description: "Создание наряда на работу",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["work_order_id", "device_id", "title", "type", "priority"],
			"properties": {
				"work_order_id": {"type": "string", "format": "uuid"},
				"device_id":     {"type": "string", "format": "uuid"},
				"title":         {"type": "string", "maxLength": 500},
				"type":          {"type": "string", "enum": ["preventive", "corrective", "emergency", "routine", "inspection"]},
				"priority":      {"type": "string", "enum": ["critical", "high", "medium", "low"]},
				"assignee_id":   {"type": "string", "format": "uuid"}
			}
		}`),
		Required: true,
	})

	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceCMMS,
		EventType:   "cmms.wo.completed",
		Version:     "1.0.0",
		Description: "Завершение наряда",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["work_order_id", "completed_by"],
			"properties": {
				"work_order_id": {"type": "string", "format": "uuid"},
				"completed_by":  {"type": "string", "format": "uuid"},
				"notes":         {"type": "string", "maxLength": 5000},
				"actual_cost":   {"type": "number", "minimum": 0}
			}
		}`),
		Required: true,
	})

	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceCMMS,
		EventType:   "cmms.wo.status_changed",
		Version:     "1.0.0",
		Description: "Изменение статуса наряда",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["work_order_id", "from_status", "to_status"],
			"properties": {
				"work_order_id": {"type": "string", "format": "uuid"},
				"from_status":   {"type": "string"},
				"to_status":     {"type": "string"},
				"changed_by":    {"type": "string", "format": "uuid"}
			}
		}`),
		Required: true,
	})

	// ── Prediction events ─────────────────────────────────────────
	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourcePredictions,
		EventType:   "prediction.created",
		Version:     "1.0.0",
		Description: "Предиктивный прогноз отказов",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["device_id", "failure_mode", "probability"],
			"properties": {
				"device_id":      {"type": "string", "format": "uuid"},
				"device_name":    {"type": "string"},
				"failure_mode":   {"type": "string"},
				"probability":    {"type": "number", "minimum": 0, "maximum": 1},
				"estimated_days": {"type": "integer", "minimum": 0},
				"recommendation": {"type": "string", "maxLength": 2000}
			}
		}`),
		Required: true,
	})

	// ── Telemetry events ──────────────────────────────────────────
	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceTelemetry,
		EventType:   "telemetry.metric",
		Version:     "1.0.0",
		Description: "Метрика телеметрии устройства",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["device_id", "metric", "value"],
			"properties": {
				"device_id": {"type": "string", "format": "uuid"},
				"metric":    {"type": "string"},
				"value":     {"type": "number"},
				"tags":      {"type": "object"},
				"unit":      {"type": "string"}
			}
		}`),
		Required: false, // телеметрия может быть в любом формате
	})

	// ── Audit events ──────────────────────────────────────────────
	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceAudit,
		EventType:   "audit.access",
		Version:     "1.0.0",
		Description: "Событие доступа к системе",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["user_id", "action", "resource"],
			"properties": {
				"user_id":     {"type": "string", "format": "uuid"},
				"action":      {"type": "string", "enum": ["login", "logout", "create", "read", "update", "delete", "export"]},
				"resource":    {"type": "string"},
				"resource_id": {"type": "string"},
				"ip_address":  {"type": "string", "format": "ipv4"},
				"user_agent":  {"type": "string", "maxLength": 500}
			}
		}`),
		Required: true,
	})

	// ── System events ─────────────────────────────────────────────
	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceSystem,
		EventType:   "system.startup",
		Version:     "1.0.0",
		Description: "Запуск системы",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"version":      {"type": "string"},
				"hostname":     {"type": "string"},
				"go_version":   {"type": "string"},
				"uptime_sec":   {"type": "integer"}
			}
		}`),
		Required: false,
	})

	_ = r.RegisterSchema(&SchemaDefinition{
		Source:      SourceSystem,
		EventType:   "system.shutdown",
		Version:     "1.0.0",
		Description: "Остановка системы",
		Schema: json.RawMessage(`{
			"type": "object",
			"required": ["reason"],
			"properties": {
				"reason":     {"type": "string"},
				"uptime_sec": {"type": "integer"},
				"signal":     {"type": "string"}
			}
		}`),
		Required: false,
	})
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func schemaKey(source EventSource, eventType string) string {
	return fmt.Sprintf("%s.%s", source, eventType)
}
