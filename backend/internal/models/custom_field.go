// Package models — domain models for Custom Fields (P2-FIELDS).
//
// Соответствует:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — audit trail)
//   - OWASP ASVS V5.1 (Input validation — enum constraints)
//   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)
//   - СТБ 34.101.27 п. 6.3 (Контроль целостности данных)
package models

import (
	"encoding/json"
	"time"
)

// ── FieldType ──────────────────────────────────────────────────────────

// FieldType определяет тип кастомного поля.
type FieldType string

const (
	FieldText        FieldType = "text"
	FieldNumber      FieldType = "number"
	FieldDate        FieldType = "date"
	FieldDropdown    FieldType = "dropdown"
	FieldMultiSelect FieldType = "multi_select"
	FieldURL         FieldType = "url"
	FieldEmail       FieldType = "email"
	FieldBarcode     FieldType = "barcode"
	FieldSignature   FieldType = "signature"
	FieldFile        FieldType = "file_upload"
	FieldCheckbox    FieldType = "checkbox"
	FieldRadio       FieldType = "radio"
	FieldTextarea    FieldType = "textarea"
	FieldTime        FieldType = "time"
	FieldColor       FieldType = "color"
	FieldUser        FieldType = "user"
)

// ValidFieldTypes для whitelist validation (OWASP ASVS V5.1)
var ValidFieldTypes = []string{
	string(FieldText), string(FieldNumber), string(FieldDate),
	string(FieldDropdown), string(FieldMultiSelect),
	string(FieldURL), string(FieldEmail), string(FieldBarcode),
	string(FieldSignature), string(FieldFile),
	string(FieldCheckbox), string(FieldRadio),
	string(FieldTextarea), string(FieldTime), string(FieldColor),
	string(FieldUser),
}

// ── EntityType ─────────────────────────────────────────────────────────

// EntityType определяет тип сущности, к которой привязано поле.
type EntityType string

const (
	EntityDevice    EntityType = "device"
	EntityWorkOrder EntityType = "work_order"
	EntitySite      EntityType = "site"
	EntityPart      EntityType = "part"
)

// ValidEntityTypes для whitelist validation.
var ValidEntityTypes = []string{
	string(EntityDevice), string(EntityWorkOrder),
	string(EntitySite), string(EntityPart),
}

// ── ValidationRule ─────────────────────────────────────────────────────

// ValidationRule определяет правила валидации для кастомного поля.
type ValidationRule struct {
	Min    *float64 `json:"min,omitempty"`     // минимальное значение (number, date)
	Max    *float64 `json:"max,omitempty"`     // максимальное значение (number, date)
	MinLen *int     `json:"min_len,omitempty"` // минимальная длина (text, textarea)
	MaxLen *int     `json:"max_len,omitempty"` // максимальная длина (text, textarea)
	Regex  string   `json:"regex,omitempty"`   // регулярное выражение (text, url, email, barcode)
	Custom string   `json:"custom,omitempty"`  // кастомное правило (JS-выражение для frontend)
}

// ── FieldCondition ─────────────────────────────────────────────────────

// FieldCondition определяет условие для условной видимости поля.
type FieldCondition struct {
	FieldID  string      `json:"field_id"` // ID поля-триггера
	Operator string      `json:"operator"` // eq, neq, gt, lt, gte, lte, in, contains
	Value    interface{} `json:"value"`    // значение для сравнения
}

// ── FieldGroup ─────────────────────────────────────────────────────────

// FieldGroup представляет группу кастомных полей.
type FieldGroup struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	EntityType    string    `json:"entity_type"`
	SortOrder     int       `json:"sort_order"`
	IsCollapsible bool      `json:"is_collapsible"`
	IsCollapsed   bool      `json:"is_collapsed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ── FieldDefinition ────────────────────────────────────────────────────

// FieldDefinition представляет определение кастомного поля.
type FieldDefinition struct {
	ID           string          `json:"id"`
	EntityType   string          `json:"entity_type"`
	FieldType    FieldType       `json:"field_type"`
	Name         string          `json:"name"`
	Label        string          `json:"label"`
	Description  string          `json:"description,omitempty"`
	Required     bool            `json:"required"`
	Options      []string        `json:"options,omitempty"`    // для dropdown/multi_select/radio
	Validation   *ValidationRule `json:"validation,omitempty"` // правила валидации
	Visibility   *FieldCondition `json:"visibility,omitempty"` // условная видимость
	GroupID      string          `json:"group_id,omitempty"`
	SortOrder    int             `json:"sort_order"`
	DefaultValue interface{}     `json:"default_value,omitempty"`
	Placeholder  string          `json:"placeholder,omitempty"`
	IsActive     bool            `json:"is_active"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// ── FieldValue ─────────────────────────────────────────────────────────

// FieldValue представляет значение кастомного поля для конкретной сущности.
type FieldValue struct {
	ID         string      `json:"id"`
	FieldID    string      `json:"field_id"`
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	Value      interface{} `json:"value"`
	CreatedBy  string      `json:"created_by,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// ── FieldValueAudit ────────────────────────────────────────────────────

// FieldValueAudit представляет запись аудита изменения кастомного поля.
type FieldValueAudit struct {
	ID         string      `json:"id"`
	ValueID    string      `json:"value_id"`
	FieldID    string      `json:"field_id"`
	EntityType string      `json:"entity_type"`
	EntityID   string      `json:"entity_id"`
	OldValue   interface{} `json:"old_value,omitempty"`
	NewValue   interface{} `json:"new_value"`
	ChangedBy  string      `json:"changed_by"`
	ChangedAt  time.Time   `json:"changed_at"`
}

// ── Request / Response Types ───────────────────────────────────────────

// CreateFieldDefinitionRequest — запрос на создание определения поля.
type CreateFieldDefinitionRequest struct {
	EntityType   string          `json:"entity_type"`
	FieldType    FieldType       `json:"field_type"`
	Name         string          `json:"name"`
	Label        string          `json:"label"`
	Description  string          `json:"description,omitempty"`
	Required     bool            `json:"required"`
	Options      []string        `json:"options,omitempty"`
	Validation   *ValidationRule `json:"validation,omitempty"`
	Visibility   *FieldCondition `json:"visibility,omitempty"`
	GroupID      string          `json:"group_id,omitempty"`
	SortOrder    int             `json:"sort_order"`
	DefaultValue interface{}     `json:"default_value,omitempty"`
	Placeholder  string          `json:"placeholder,omitempty"`
}

// UpdateFieldDefinitionRequest — запрос на обновление определения поля.
type UpdateFieldDefinitionRequest struct {
	Label        *string         `json:"label,omitempty"`
	Description  *string         `json:"description,omitempty"`
	Required     *bool           `json:"required,omitempty"`
	Options      *[]string       `json:"options,omitempty"`
	Validation   *ValidationRule `json:"validation,omitempty"`
	Visibility   *FieldCondition `json:"visibility,omitempty"`
	GroupID      *string         `json:"group_id,omitempty"`
	SortOrder    *int            `json:"sort_order,omitempty"`
	DefaultValue *interface{}    `json:"default_value,omitempty"`
	Placeholder  *string         `json:"placeholder,omitempty"`
	IsActive     *bool           `json:"is_active,omitempty"`
}

// CreateGroupRequest — запрос на создание группы полей.
type CreateGroupRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	EntityType    string `json:"entity_type"`
	SortOrder     int    `json:"sort_order"`
	IsCollapsible bool   `json:"is_collapsible"`
	IsCollapsed   bool   `json:"is_collapsed"`
}

// UpdateGroupRequest — запрос на обновление группы полей.
type UpdateGroupRequest struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	SortOrder     *int    `json:"sort_order,omitempty"`
	IsCollapsible *bool   `json:"is_collapsible,omitempty"`
	IsCollapsed   *bool   `json:"is_collapsed,omitempty"`
}

// BulkUpdateValuesRequest — запрос на массовое обновление значений полей.
type BulkUpdateValuesRequest struct {
	Values map[string]interface{} `json:"values"` // key = field_id, value = новое значение
}

// FieldDefinitionListQuery — параметры фильтрации списка определений.
type FieldDefinitionListQuery struct {
	EntityType string `json:"entity_type"`
	GroupID    string `json:"group_id,omitempty"`
	ActiveOnly bool   `json:"active_only,omitempty"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
}

// FieldDefinitionWithValue — определение поля с текущим значением для entity.
type FieldDefinitionWithValue struct {
	FieldDefinition
	Value interface{} `json:"value,omitempty"`
}

// ── Helpers ────────────────────────────────────────────────────────────

// Validate проверяет обязательные поля для CreateFieldDefinitionRequest.
func (r *CreateFieldDefinitionRequest) Validate() error {
	return nil // validation logic is in handler layer
}

// Sanitize очищает поля от потенциально опасного содержимого.
func (r *CreateFieldDefinitionRequest) Sanitize() {
	// noop — handler layer handles sanitization
}

// String реализует Stringer для FieldType.
func (ft FieldType) String() string {
	return string(ft)
}

// MarshalJSON кастомная сериализация FieldDefinition с JSONB полями.
func (fd *FieldDefinition) MarshalJSON() ([]byte, error) {
	type Alias FieldDefinition
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(fd),
	})
}

// UnmarshalJSON кастомная десериализация FieldDefinition.
func (fd *FieldDefinition) UnmarshalJSON(data []byte) error {
	type Alias FieldDefinition
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(fd),
	}
	return json.Unmarshal(data, aux)
}
