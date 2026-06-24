// Package models — WorkRequest entity (WO-4.1.1).
//
// WorkRequest — публичная заявка на создание WorkOrder от внешнего пользователя.
// Submission без авторизации, с reCAPTCHA. После одобрения конвертируется в WorkOrder.
//
// Compliance:
//   - OWASP ASVS V1.1 (Input validation — whitelist)
//   - OWASP ASVS V3.1 (Session management — reCAPTCHA)
//   - ISO 27001 A.9.2.1 (User registration — external request)
//   - ISO 27001 A.14.2.1 (Service delivery — request portal)
//   - IEC 62443 SR 2.1 (Account management — request workflow)
//   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)
package models

import (
	"time"
)

// WorkRequestStatus — статусы заявки.
type WorkRequestStatus string

const (
	WorkRequestSubmitted  WorkRequestStatus = "submitted"   // подана, ожидает подтверждения
	WorkRequestApproved   WorkRequestStatus = "approved"    // одобрена
	WorkRequestConverted  WorkRequestStatus = "converted"   // конвертирована в WorkOrder
	WorkRequestRejected   WorkRequestStatus = "rejected"    // отклонена
	WorkRequestCancelled  WorkRequestStatus = "cancelled"   // отозвана заявителем
)

// WorkRequest — публичная заявка на выполнение работ.
//
// Отличие от WorkOrder: WorkRequest не требует авторизации, создаётся
// через публичный endpoint и проходит через approval workflow.
type WorkRequest struct {
	ID        string    `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// ── Основные поля ───────────────────────────────────────────
	Title       string `json:"title" db:"title"`
	Description string `json:"description,omitempty" db:"description"`

	// ── Связанные сущности ──────────────────────────────────────
	DeviceID  string `json:"device_id,omitempty" db:"device_id"`
	DeviceName string `json:"device_name,omitempty" db:"-"` // denormalized
	SiteID    string `json:"site_id,omitempty" db:"site_id"`
	SiteName  string `json:"site_name,omitempty" db:"-"` // denormalized

	// ── Приоритет и тип ─────────────────────────────────────────
	Priority string `json:"priority" db:"priority"` // critical, high, medium, low
	Type     string `json:"type" db:"type"`          // corrective, preventive, emergency, routine, inspection

	// ── Контактные данные заявителя ──────────────────────────────
	RequesterName  string `json:"requester_name" db:"requester_name"`
	RequesterEmail string `json:"requester_email" db:"requester_email"`
	RequesterPhone string `json:"requester_phone,omitempty" db:"requester_phone"`

	// ── Статус и workflow ────────────────────────────────────────
	Status    WorkRequestStatus `json:"status" db:"status"`
	ApprovedBy *string          `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt *time.Time       `json:"approved_at,omitempty" db:"approved_at"`
	RejectedBy *string          `json:"rejected_by,omitempty" db:"rejected_by"`
	RejectedAt *time.Time       `json:"rejected_at,omitempty" db:"rejected_at"`
	RejectionReason string     `json:"rejection_reason,omitempty" db:"rejection_reason"`

	// ── Связь с WorkOrder ────────────────────────────────────────
	ConvertedWorkOrderID *string `json:"converted_work_order_id,omitempty" db:"converted_work_order_id"`
	ConvertedAt          *time.Time `json:"converted_at,omitempty" db:"converted_at"`

	// ── reCAPTCHA ────────────────────────────────────────────────
	CaptchaToken string `json:"captcha_token,omitempty" db:"-"` // только для submit

	// ── Метаданные ──────────────────────────────────────────────
	SourceIP  string `json:"source_ip,omitempty" db:"source_ip"`
	UserAgent string `json:"user_agent,omitempty" db:"user_agent"`
}

// WorkRequestStatuses — список всех статусов заявки для валидации.
var WorkRequestStatuses = []WorkRequestStatus{
	WorkRequestSubmitted,
	WorkRequestApproved,
	WorkRequestConverted,
	WorkRequestRejected,
	WorkRequestCancelled,
}

// ValidWorkRequestStatus проверяет, является ли статус допустимым.
func ValidWorkRequestStatus(s string) bool {
	for _, status := range WorkRequestStatuses {
		if string(status) == s {
			return true
		}
	}
	return false
}

// WorkRequestPriorities — список приоритетов.
var WorkRequestPriorities = []string{"critical", "high", "medium", "low"}

// ValidWorkRequestPriority проверяет приоритет.
func ValidWorkRequestPriority(s string) bool {
	for _, p := range WorkRequestPriorities {
		if p == s {
			return true
		}
	}
	return false
}

// WorkRequestTypes — список типов.
var WorkRequestTypes = []string{"corrective", "preventive", "emergency", "routine", "inspection"}

// ValidWorkRequestType проверяет тип.
func ValidWorkRequestType(s string) bool {
	for _, t := range WorkRequestTypes {
		if t == s {
			return true
		}
	}
	return false
}

// ── Approval Workflow ──────────────────────────────────────────────

// WorkRequestTransition проверяет допустимость перехода статусов.
func WorkRequestTransition(from, to WorkRequestStatus) bool {
	transitions := map[WorkRequestStatus][]WorkRequestStatus{
		WorkRequestSubmitted: {WorkRequestApproved, WorkRequestRejected, WorkRequestCancelled},
		WorkRequestApproved:  {WorkRequestConverted, WorkRequestCancelled},
		WorkRequestConverted: {}, // терминальный
		WorkRequestRejected:  {}, // терминальный
		WorkRequestCancelled: {}, // терминальный
	}
	allowed, ok := transitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
