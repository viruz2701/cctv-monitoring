// Package models — Annotation models for P1-PHOTO (Photo Annotation).
//
// Обеспечивает хранение элементов аннотации (стрелки, круги, текст и т.д.)
// в формате JSONB для каждого фото work order.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — JSON schema validation на уровне handler)
//   - ISO 27001 A.12.4 (Audit trail — created_at/updated_at + audit_log)
//   - IEC 62443 SL-3 (Zone 3 — Application security)
//   - СТБ 34.101.27 п. 6.2 (Контроль целостности данных)
package models

import (
	"encoding/json"
	"time"
)

// AnnotationElement представляет один элемент аннотации на фото.
// Хранится как JSONB в таблице work_order_annotations.
type AnnotationElement struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"` // arrow, freehand, text, highlight, circle, blur, measurement
	Color       string          `json:"color"`
	StrokeWidth int             `json:"strokeWidth"`
	Points      json.RawMessage `json:"points,omitempty"`   // для freehand
	Start       json.RawMessage `json:"start,omitempty"`    // Point {x, y}
	End         json.RawMessage `json:"end,omitempty"`      // Point {x, y}
	Center      json.RawMessage `json:"center,omitempty"`   // Point {x, y}
	Position    json.RawMessage `json:"position,omitempty"` // Point {x, y} для text
	Text        string          `json:"text,omitempty"`     // для text
	FontSize    int             `json:"fontSize,omitempty"` // для text
	Radius      float64         `json:"radius,omitempty"`   // для circle
	LengthPx    float64         `json:"lengthPx,omitempty"` // для measurement
}

// WorkOrderAnnotation — аннотация для конкретного фото в work order.
//
// Хранит JSONB-массив элементов аннотации, привязанных к URL фото.
// Один work order может иметь множество аннотаций (по одной на фото).
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — created_at/updated_at)
//   - OWASP ASVS V5.1 (Input validation — Elements валидируется на уровне handler)
type WorkOrderAnnotation struct {
	ID          string              `json:"id" db:"id"`
	WorkOrderID string              `json:"work_order_id" db:"work_order_id"`
	PhotoURL    string              `json:"photo_url" db:"photo_url"`
	Elements    []AnnotationElement `json:"elements" db:"elements"`
	CreatedBy   string              `json:"created_by" db:"created_by"`
	CreatedAt   time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" db:"updated_at"`
}

// AnnotationSaveRequest — тело запроса для создания/обновления аннотации.
type AnnotationSaveRequest struct {
	Elements []AnnotationElement `json:"elements"`
}

// Validate проверяет корректность структуры запроса.
// Соответствует: OWASP ASVS V5.1 (Input validation — whitelist approach)
func (r *AnnotationSaveRequest) Validate() []string {
	var errs []string

	if len(r.Elements) == 0 {
		errs = append(errs, "elements: must contain at least one element")
	}

	validTypes := map[string]bool{
		"arrow": true, "freehand": true, "text": true,
		"highlight": true, "circle": true, "blur": true, "measurement": true,
	}

	for i, el := range r.Elements {
		if el.ID == "" {
			errs = append(errs, fieldError("elements", i, "id", "is required"))
		}
		if !validTypes[el.Type] {
			errs = append(errs, fieldError("elements", i, "type", "invalid type: "+el.Type))
		}
		if el.Color == "" {
			errs = append(errs, fieldError("elements", i, "color", "is required"))
		}
		if el.StrokeWidth <= 0 || el.StrokeWidth > 20 {
			errs = append(errs, fieldError("elements", i, "strokeWidth", "must be between 1 and 20"))
		}
	}

	// Max 500 элементов для предотвращения DoS
	if len(r.Elements) > 500 {
		errs = append(errs, "elements: maximum 500 elements allowed")
	}

	return errs
}

func fieldError(prefix string, index int, field, msg string) string {
	return prefix + "[" + itoa(index) + "]." + field + ": " + msg
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [12]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
