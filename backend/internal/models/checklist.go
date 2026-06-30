// Package models — domain models for CCTV Health Monitor.
//
// P2-CHECK: Conditional Checklists (MaintainX-level)
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — checklist audit trail)
//   - ISO 27001 A.12.6 (Maintenance — structured checklists)
//   - OWASP ASVS V5.1 (Input validation — enum constraints, whitelist)
//   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)
package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// ItemType constants — whitelist for OWASP ASVS V5.1
// ═══════════════════════════════════════════════════════════════════════

type ChecklistItemType string

const (
	ItemTypeBoolean     ChecklistItemType = "boolean"
	ItemTypeText        ChecklistItemType = "text"
	ItemTypePhoto       ChecklistItemType = "photo"
	ItemTypeNumeric     ChecklistItemType = "numeric"
	ItemTypeSignature   ChecklistItemType = "signature"
	ItemTypeSelect      ChecklistItemType = "select"
	ItemTypeMultiSelect ChecklistItemType = "multi_select"
)

// ValidChecklistItemTypes для whitelist validation (OWASP ASVS V5.1)
var ValidChecklistItemTypes = []string{
	string(ItemTypeBoolean), string(ItemTypeText), string(ItemTypePhoto),
	string(ItemTypeNumeric), string(ItemTypeSignature),
	string(ItemTypeSelect), string(ItemTypeMultiSelect),
}

// ═══════════════════════════════════════════════════════════════════════
// Operator constants
// ═══════════════════════════════════════════════════════════════════════

type ConditionOperator string

const (
	OpEq  ConditionOperator = "eq"
	OpNeq ConditionOperator = "neq"
	OpGt  ConditionOperator = "gt"
	OpLt  ConditionOperator = "lt"
	OpGte ConditionOperator = "gte"
	OpLte ConditionOperator = "lte"
	OpIn  ConditionOperator = "in"
)

// ValidConditionOperators для whitelist validation
var ValidConditionOperators = []string{
	string(OpEq), string(OpNeq), string(OpGt), string(OpLt),
	string(OpGte), string(OpLte), string(OpIn),
}

// ═══════════════════════════════════════════════════════════════════════
// Checklist status constants
// ═══════════════════════════════════════════════════════════════════════

type WOChecklistStatus string

const (
	WOCStatusInProgress WOChecklistStatus = "in_progress"
	WOCStatusSubmitted  WOChecklistStatus = "submitted"
	WOCStatusVerified   WOChecklistStatus = "verified"
)

// ValidWOChecklistStatuses для whitelist validation
var ValidWOChecklistStatuses = []string{
	string(WOCStatusInProgress), string(WOCStatusSubmitted), string(WOCStatusVerified),
}

// ═══════════════════════════════════════════════════════════════════════
// ChecklistTemplate — шаблон чек-листа для типа устройства
// ═══════════════════════════════════════════════════════════════════════

type ChecklistTemplate struct {
	ID            string          `json:"id" db:"id"`
	Name          string          `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description   string          `json:"description" db:"description" validate:"max=5000"`
	DeviceTypes   []string        `json:"device_types" db:"device_types"` // camera, nvr, dvr, etc
	PassThreshold int             `json:"pass_threshold" db:"pass_threshold" validate:"min=0,max=100"`
	IsActive      bool            `json:"is_active" db:"is_active"`
	Items         []ChecklistItem `json:"items,omitempty"` // populated on GET by ID
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// ChecklistItem — элемент чек-листа (поддерживает иерархию и условия)
// ═══════════════════════════════════════════════════════════════════════

type ChecklistItem struct {
	ID            string          `json:"id" db:"id"`
	TemplateID    string          `json:"template_id" db:"template_id" validate:"required"`
	ParentID      *string         `json:"parent_id,omitempty" db:"parent_id"`
	Label         string          `json:"label" db:"label" validate:"required,min=1,max=500"`
	Description   string          `json:"description" db:"description" validate:"max=2000"`
	ItemType      string          `json:"item_type" db:"item_type" validate:"required,oneof=boolean text photo numeric signature select multi_select"`
	Mandatory     bool            `json:"mandatory" db:"mandatory"`
	Score         int             `json:"score" db:"score" validate:"min=0"`
	SortOrder     int             `json:"sort_order" db:"sort_order"`
	Options       json.RawMessage `json:"options,omitempty" db:"options"`               // for select/multi_select
	ValidationMin *float64        `json:"validation_min,omitempty" db:"validation_min"` // for numeric
	ValidationMax *float64        `json:"validation_max,omitempty" db:"validation_max"` // for numeric
	DependsOn     *Condition      `json:"depends_on,omitempty"`                         // conditional logic
	Children      []ChecklistItem `json:"children,omitempty"`                           // sub-items
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Condition — условие для depends_on (show/hide logic)
// ═══════════════════════════════════════════════════════════════════════

type Condition struct {
	FieldID  string      `json:"field_id"`
	Operator string      `json:"operator" validate:"required,oneof=eq neq gt lt gte lte in"`
	Value    interface{} `json:"value"`
}

// Evaluate проверяет, выполняется ли условие для данного значения.
// Используется для динамического show/hide элементов чек-листа.
func (c *Condition) Evaluate(actualValue interface{}) bool {
	if c == nil {
		return true
	}

	switch c.Operator {
	case "eq":
		return compareEq(actualValue, c.Value)
	case "neq":
		return !compareEq(actualValue, c.Value)
	case "gt":
		return compareNumeric(actualValue, c.Value) > 0
	case "lt":
		return compareNumeric(actualValue, c.Value) < 0
	case "gte":
		return compareNumeric(actualValue, c.Value) >= 0
	case "lte":
		return compareNumeric(actualValue, c.Value) <= 0
	case "in":
		return compareIn(actualValue, c.Value)
	default:
		return true
	}
}

// compareEq сравнивает два значения на равенство.
func compareEq(a, b interface{}) bool {
	aStr, aOK := toString(a)
	bStr, bOK := toString(b)
	if aOK && bOK {
		return aStr == bStr
	}
	return a == b
}

// compareNumeric сравнивает два числовых значения.
// Возвращает -1, 0, 1 для <, ==, >.
func compareNumeric(a, b interface{}) int {
	aFloat, aOK := toFloat64(a)
	bFloat, bOK := toFloat64(b)
	if aOK && bOK {
		if aFloat < bFloat {
			return -1
		}
		if aFloat > bFloat {
			return 1
		}
		return 0
	}
	return 0
}

// compareIn проверяет, входит ли значение в список (JSON array).
func compareIn(actualValue, possibleValues interface{}) bool {
	actualStr, ok := toString(actualValue)
	if !ok {
		return false
	}

	switch v := possibleValues.(type) {
	case []interface{}:
		for _, item := range v {
			if itemStr, ok := toString(item); ok && itemStr == actualStr {
				return true
			}
		}
	case string:
		// Try parsing as JSON array
		var arr []interface{}
		if err := json.Unmarshal([]byte(v), &arr); err == nil {
			for _, item := range arr {
				if itemStr, ok := toString(item); ok && itemStr == actualStr {
					return true
				}
			}
		}
	}
	return false
}

// toString пытается конвертировать значение в строку.
func toString(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	switch val := v.(type) {
	case string:
		return val, true
	case bool:
		if val {
			return "true", true
		}
		return "false", true
	case float64:
		return json.Number(json.Number(formatFloat(val))).String(), true
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "", false
		}
		return string(b), true
	}
}

// toFloat64 пытается конвертировать значение в float64.
func toFloat64(v interface{}) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}

// formatFloat форматирует float64 без лишних нулей.
func formatFloat(f float64) string {
	if f == float64(int(f)) {
		return fmt.Sprintf("%d", int(f))
	}
	return fmt.Sprintf("%g", f)
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderChecklist — запущенный экземпляр чек-листа для Work Order
// ═══════════════════════════════════════════════════════════════════════

type WorkOrderChecklist struct {
	ID           string              `json:"id" db:"id"`
	WorkOrderID  string              `json:"work_order_id" db:"work_order_id" validate:"required"`
	TemplateID   string              `json:"template_id" db:"template_id" validate:"required"`
	Status       string              `json:"status" db:"status" validate:"required,oneof=in_progress submitted verified"`
	TotalScore   int                 `json:"total_score" db:"total_score"`
	MaxScore     int                 `json:"max_score" db:"max_score"`
	ScorePercent float64             `json:"score_percent" db:"score_percent"`
	Passed       bool                `json:"passed" db:"passed"`
	StartedBy    string              `json:"started_by" db:"started_by" validate:"required"`
	StartedAt    time.Time           `json:"started_at" db:"started_at"`
	SubmittedBy  *string             `json:"submitted_by,omitempty" db:"submitted_by"`
	SubmittedAt  *time.Time          `json:"submitted_at,omitempty" db:"submitted_at"`
	VerifiedBy   *string             `json:"verified_by,omitempty" db:"verified_by"`
	VerifiedAt   *time.Time          `json:"verified_at,omitempty" db:"verified_at"`
	Notes        string              `json:"notes" db:"notes" validate:"max=5000"`
	Responses    []ChecklistResponse `json:"responses,omitempty"` // populated on GET
	CreatedAt    time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at" db:"updated_at"`

	// Denormalized для UI
	TemplateName string `json:"template_name,omitempty"`
	DeviceName   string `json:"device_name,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// ChecklistResponse — ответ на элемент чек-листа
// ═══════════════════════════════════════════════════════════════════════

type ChecklistResponse struct {
	ID          string    `json:"id" db:"id"`
	ChecklistID string    `json:"checklist_id" db:"checklist_id" validate:"required"`
	ItemID      string    `json:"item_id" db:"item_id" validate:"required"`
	Value       string    `json:"value" db:"value"` // 'true', 'false', text, photo_url, etc
	PhotoURL    *string   `json:"photo_url,omitempty" db:"photo_url"`
	Skipped     bool      `json:"skipped" db:"skipped"` // true if hidden by condition
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// ChecklistScore — запись оценки по элементу чек-листа
// ═══════════════════════════════════════════════════════════════════════

type ChecklistScore struct {
	ID          string    `json:"id" db:"id"`
	ChecklistID string    `json:"checklist_id" db:"checklist_id" validate:"required"`
	ItemID      string    `json:"item_id" db:"item_id" validate:"required"`
	Score       int       `json:"score" db:"score" validate:"min=0"`
	MaxScore    int       `json:"max_score" db:"max_score" validate:"min=0"`
	ScoredBy    string    `json:"scored_by" db:"scored_by" validate:"required"`
	ScoredAt    time.Time `json:"scored_at" db:"scored_at"`
	Notes       string    `json:"notes" db:"notes" validate:"max=2000"`
}

// ═══════════════════════════════════════════════════════════════════════
// Request DTOs
// ═══════════════════════════════════════════════════════════════════════

// CreateTemplateRequest — DTO для создания шаблона чек-листа.
type CreateTemplateRequest struct {
	Name          string          `json:"name" validate:"required,min=1,max=255"`
	Description   string          `json:"description,omitempty" validate:"max=5000"`
	DeviceTypes   []string        `json:"device_types" validate:"required,min=1"`
	PassThreshold int             `json:"pass_threshold" validate:"min=0,max=100"`
	Items         []ChecklistItem `json:"items,omitempty"`
}

// StartChecklistRequest — DTO для старта чек-листа по Work Order.
type StartChecklistRequest struct {
	TemplateID string `json:"template_id" validate:"required"`
}

// SubmitChecklistRequest — DTO для сабмита чек-листа.
type SubmitChecklistRequest struct {
	Responses []SubmitItemResponse `json:"responses" validate:"required,min=1"`
	Notes     string               `json:"notes,omitempty" validate:"max=5000"`
}

// SubmitItemResponse — ответ на один элемент при сабмите.
type SubmitItemResponse struct {
	ItemID   string  `json:"item_id" validate:"required"`
	Value    string  `json:"value"`
	PhotoURL *string `json:"photo_url,omitempty"`
	Skipped  bool    `json:"skipped"`
}

// TemplateListQuery — параметры для GET /api/v1/checklists/templates
type TemplateListQuery struct {
	DeviceType string `json:"device_type,omitempty"`
	ActiveOnly bool   `json:"active_only,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// ChecklistSummary — агрегированная сводка по чек-листу.
type ChecklistSummary struct {
	ID             string  `json:"id"`
	WorkOrderID    string  `json:"work_order_id"`
	TemplateName   string  `json:"template_name"`
	Status         string  `json:"status"`
	ScorePercent   float64 `json:"score_percent"`
	Passed         bool    `json:"passed"`
	TotalItems     int     `json:"total_items"`
	CompletedItems int     `json:"completed_items"`
	SkippedItems   int     `json:"skipped_items"`
	StartedBy      string  `json:"started_by"`
	StartedAt      string  `json:"started_at"`
	SubmittedBy    *string `json:"submitted_by,omitempty"`
	SubmittedAt    *string `json:"submitted_at,omitempty"`
}
