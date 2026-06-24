// Package models — domain models for CCTV Health Monitor.
// Compliance: СТБ 34.101.27 (Защита информации), ISO 27001 A.12.4 (Audit),
// IEC 62443 SL-3 (Zone 3 — Backend), OWASP ASVS V5 (Validation)
//
// Этот файл содержит абстрактные базовые типы для доменной модели,
// реализующие паттерн Go embedding (композиция) взамен классического наследования.
// Референс: com.grash.model.abstracts.WorkOrderBase (Grash CMMS)
package models

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// WorkOrder Status Constants (12 статусов по Grash CMMS)
// ═══════════════════════════════════════════════════════════════════════

type WorkOrderStatus string

const (
	StatusRequested      WorkOrderStatus = "REQUESTED"
	StatusApproved       WorkOrderStatus = "APPROVED"
	StatusOpen           WorkOrderStatus = "OPEN"
	StatusInProgress     WorkOrderStatus = "IN_PROGRESS"
	StatusOnHold         WorkOrderStatus = "ON_HOLD"
	StatusAwaitingParts  WorkOrderStatus = "AWAITING_PARTS"
	StatusAwaitingVendor WorkOrderStatus = "AWAITING_VENDOR"
	StatusAwaitingClient WorkOrderStatus = "AWAITING_CLIENT"
	StatusCompleted      WorkOrderStatus = "COMPLETED"
	StatusVerified       WorkOrderStatus = "VERIFIED"
	StatusClosed         WorkOrderStatus = "CLOSED"
	StatusRejected       WorkOrderStatus = "REJECTED"
)

// ValidWorkOrderStatuses для whitelist validation (OWASP ASVS V5.1)
var ValidWorkOrderStatuses = []string{
	string(StatusRequested),
	string(StatusApproved),
	string(StatusOpen),
	string(StatusInProgress),
	string(StatusOnHold),
	string(StatusAwaitingParts),
	string(StatusAwaitingVendor),
	string(StatusAwaitingClient),
	string(StatusCompleted),
	string(StatusVerified),
	string(StatusClosed),
	string(StatusRejected),
}

// ═══════════════════════════════════════════════════════════════════════
// Priority Constants
// ═══════════════════════════════════════════════════════════════════════

type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityMedium   Priority = "medium"
	PriorityLow      Priority = "low"
)

// ValidPriorities для whitelist validation (OWASP ASVS V5.1)
var ValidPriorities = []string{
	string(PriorityCritical),
	string(PriorityHigh),
	string(PriorityMedium),
	string(PriorityLow),
}

// ═══════════════════════════════════════════════════════════════════════
// Work Order Type Constants
// ═══════════════════════════════════════════════════════════════════════

type WorkOrderType string

const (
	TypePreventive WorkOrderType = "preventive"
	TypeCorrective WorkOrderType = "corrective"
	TypeEmergency  WorkOrderType = "emergency"
	TypeRoutine    WorkOrderType = "routine"
	TypeInspection WorkOrderType = "inspection"
)

// ValidWorkOrderTypes для whitelist validation (OWASP ASVS V5.1)
var ValidWorkOrderTypes = []string{
	string(TypePreventive),
	string(TypeCorrective),
	string(TypeEmergency),
	string(TypeRoutine),
	string(TypeInspection),
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderBase — абстрактная базовая сущность для WorkOrder,
// PreventiveMaintenance и Request (Grash CMMS pattern).
//
// Compliance: OWASP ASVS V5.1 (whitelist validation через validate теги),
// ISO 27001 A.8 (Asset Management)
//
// Ref: com.grash.model.abstracts.WorkOrderBase
// ═══════════════════════════════════════════════════════════════════════

// WorkOrderBase содержит общие поля для всех типов рабочих назначений.
// Используется через Go embedding (композицию).
type WorkOrderBase struct {
	Title    string          `json:"title" db:"title" validate:"required,min=1,max=500"`
	Priority Priority        `json:"priority" db:"priority" validate:"required,oneof=critical high medium low"`
	Status   WorkOrderStatus `json:"status" db:"status" validate:"required"`
	Assignee *string         `json:"assignee,omitempty" db:"assigned_to" validate:"omitempty,uuid"`
	DueDate  *time.Time      `json:"due_date,omitempty" db:"due_date"`
}

// ═══════════════════════════════════════════════════════════════════════
// AuditBase — базовый audit trail для всех сущностей.
//
// Compliance: ISO 27001 A.12.4 (Audit Logging),
// СТБ 34.101.27 п. 7.3 (Контроль доступа и аудит),
// IEC 62443 SR 2.8 (Audit Events)
//
// Ref: com.grash.model.abstracts.Audit
// ═══════════════════════════════════════════════════════════════════════

type AuditBase struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy *string   `json:"created_by,omitempty" db:"created_by" validate:"omitempty,uuid"`
	UpdatedBy *string   `json:"updated_by,omitempty" db:"updated_by" validate:"omitempty,uuid"`
}

// ═══════════════════════════════════════════════════════════════════════
// SoftDeleteMixin — паттерн soft delete + archive.
//
// Compliance: ISO 27001 A.8.10 (Information disposal),
// ISO 27001 A.12.4.1 (Event logging — retention)
// ═══════════════════════════════════════════════════════════════════════

type SoftDeleteMixin struct {
	DeletedAt  *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	ArchivedAt *time.Time `json:"archived_at,omitempty" db:"archived_at"`
	DeletedBy  *string    `json:"deleted_by,omitempty" db:"deleted_by" validate:"omitempty,uuid"`
}

// IsDeleted возвращает true если сущность помечена как удалённая.
func (s *SoftDeleteMixin) IsDeleted() bool {
	return s.DeletedAt != nil && !s.DeletedAt.IsZero()
}

// IsArchived возвращает true если сущность архивирована.
func (s *SoftDeleteMixin) IsArchived() bool {
	return s.ArchivedAt != nil && !s.ArchivedAt.IsZero()
}

// ═══════════════════════════════════════════════════════════════════════
// CostBase — абстрактная базовая стоимость.
//
// Compliance: ISO 27001 A.12.6 (Technical vulnerability management — cost tracking)
// Ref: com.grash.model.abstracts.Cost
// ═══════════════════════════════════════════════════════════════════════

type CostBase struct {
	EstimatedCost float64 `json:"estimated_cost" db:"estimated_cost" validate:"min=0"`
	ActualCost    float64 `json:"actual_cost" db:"actual_cost" validate:"min=0"`
	Currency      string  `json:"currency" db:"currency" validate:"omitempty,len=3"` // ISO 4217: USD, EUR, BYN
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderHistory — immutable timeline событий WorkOrder.
//
// Compliance: ISO 27001 A.12.4.1 (Event logging),
// СТБ 34.101.27 п. 7.5 (Audit trail integrity),
// IEC 62443 SR 2.8 (Audit events — tamper detection)
//
// Ref: com.grash.model.WorkOrderHistoryShowDTO
// ═══════════════════════════════════════════════════════════════════════

type WorkOrderHistory struct {
	ID          string          `json:"id" db:"id"`
	WorkOrderID string          `json:"work_order_id" db:"work_order_id" validate:"required,uuid"`
	FromStatus  WorkOrderStatus `json:"from_status" db:"from_status"`
	ToStatus    WorkOrderStatus `json:"to_status" db:"to_status"`
	ChangedBy   string          `json:"changed_by" db:"changed_by" validate:"required,uuid"`
	Comment     string          `json:"comment,omitempty" db:"comment" validate:"max=2000"`
	ChangedAt   time.Time       `json:"changed_at" db:"changed_at"`
	PrevHash    string          `json:"prev_hash" db:"prev_hash"` // СТБ bash-256 HMAC цепочка
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderRelation — связи между WorkOrder (parent/child, blocked_by и т.д.)
//
// Ref: com.grash.model.WorkOrderRelation (DUPLICATE_OF, BLOCKED_BY, SPLIT_TO)
// ═══════════════════════════════════════════════════════════════════════

type WorkOrderRelationType string

const (
	RelationParentChild WorkOrderRelationType = "PARENT_CHILD"
	RelationBlockedBy   WorkOrderRelationType = "BLOCKED_BY"
	RelationDuplicateOf WorkOrderRelationType = "DUPLICATE_OF"
	RelationSplitTo     WorkOrderRelationType = "SPLIT_TO"
	RelationRelatedTo   WorkOrderRelationType = "RELATED_TO"
)

type WorkOrderRelation struct {
	ID           string                `json:"id" db:"id"`
	SourceWOID   string                `json:"source_wo_id" db:"source_wo_id" validate:"required,uuid"`
	TargetWOID   string                `json:"target_wo_id" db:"target_wo_id" validate:"required,uuid"`
	RelationType WorkOrderRelationType `json:"relation_type" db:"relation_type"`
	CreatedAt    time.Time             `json:"created_at" db:"created_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Request — заявка от пользователя/клиента (Request Portal).
//
// Compliance: OWASP ASVS V5 (Input validation), ISO 27001 A.9.2 (User access)
// Ref: com.grash.model.Request, RequestPortal
// ═══════════════════════════════════════════════════════════════════════

type Request struct {
	WorkOrderBase   // title, priority, status, assignee, dueDate
	AuditBase       // created_at, updated_at, created_by, updated_by
	SoftDeleteMixin // deleted_at, archived_at

	ID            string  `json:"id" db:"id"`
	DeviceID      string  `json:"device_id" db:"device_id" validate:"required,uuid"`
	SiteID        *string `json:"site_id,omitempty" db:"site_id" validate:"omitempty,uuid"`
	Description   string  `json:"description" db:"description" validate:"max=5000"`
	ContactName   string  `json:"contact_name" db:"contact_name" validate:"max=255"`
	ContactEmail  string  `json:"contact_email" db:"contact_email" validate:"omitempty,email"`
	ContactPhone  string  `json:"contact_phone" db:"contact_phone" validate:"omitempty,max=20"`
	Source        string  `json:"source" db:"source"` // portal, email, phone, telegram, api
	ConvertedToWO *string `json:"converted_to_wo,omitempty" db:"converted_to_wo" validate:"omitempty,uuid"`

	// Denormalized для UI
	DeviceName string `json:"device_name,omitempty"`
	SiteName   string `json:"site_name,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// TimeEntry — запись времени техника (WO-4.4.1).
//
// Compliance: ISO 27001 A.9.4 (Access control — user activity logging),
// ISO 27001 A.12.4.1 (Event logging)
// Ref: com.grash.model.AdditionalTime, Labor tracking
// ═══════════════════════════════════════════════════════════════════════

type TimeEntry struct {
	ID             string     `json:"id" db:"id"`
	WorkOrderID    string     `json:"work_order_id" db:"work_order_id" validate:"required,uuid"`
	UserID         string     `json:"user_id" db:"user_id" validate:"required,uuid"`
	StartTime      time.Time  `json:"start_time" db:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty" db:"end_time"`
	PausedDuration int64      `json:"paused_duration_seconds" db:"paused_duration"` // сумма пауз в секундах
	Status         string     `json:"status" db:"status"`                           // running, paused, stopped
	Notes          string     `json:"notes,omitempty" db:"notes" validate:"max=1000"`
	HourlyRate     float64    `json:"hourly_rate" db:"hourly_rate"` // WO-4.4.2: ставка на момент старта
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`

	// Denormalized
	UserName string `json:"user_name,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// LaborCost — расчёт стоимости труда (WO-4.4.2).
// Рассчитывается как сумма (длительность × hourly_rate) по всем time_entries.
//
// Compliance: ISO 27001 A.9.4 (Activity logging)
// ═══════════════════════════════════════════════════════════════════════

type LaborCost struct {
	WorkOrderID  string  `json:"work_order_id"`
	TotalSeconds int64   `json:"total_seconds"`
	TotalHours   float64 `json:"total_hours"`
	HourlyRate   float64 `json:"hourly_rate"`
	TotalCost    float64 `json:"total_cost"`
	Currency     string  `json:"currency"`
}

// ═══════════════════════════════════════════════════════════════════════
// Labor — трудозатраты с почасовой ставкой (legacy).
//
// Compliance: ISO 27001 A.9.4 (Activity logging)
// Ref: com.grash.model.Labor
// ═══════════════════════════════════════════════════════════════════════

type Labor struct {
	CostBase               // estimated_cost, actual_cost, currency
	ID           string    `json:"id" db:"id"`
	WorkOrderID  string    `json:"work_order_id" db:"work_order_id" validate:"required,uuid"`
	TechnicianID string    `json:"technician_id" db:"technician_id" validate:"required,uuid"`
	HourlyRate   float64   `json:"hourly_rate" db:"hourly_rate" validate:"min=0"`
	HoursWorked  float64   `json:"hours_worked" db:"hours_worked" validate:"min=0"`
	Description  string    `json:"description,omitempty" db:"description" validate:"max=1000"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// AdditionalCost — дополнительные затраты (travel, subcontractor, permits).
//
// Ref: com.grash.model.AdditionalCost
// ═══════════════════════════════════════════════════════════════════════

type AdditionalCost struct {
	CostBase              // estimated_cost, actual_cost, currency
	ID          string    `json:"id" db:"id"`
	WorkOrderID string    `json:"work_order_id" db:"work_order_id" validate:"required,uuid"`
	Category    string    `json:"category" db:"category" validate:"required,oneof=travel subcontractor permit equipment other"`
	Description string    `json:"description,omitempty" db:"description" validate:"max=1000"`
	VendorName  string    `json:"vendor_name,omitempty" db:"vendor_name" validate:"max=255"`
	ReceiptURL  string    `json:"receipt_url,omitempty" db:"receipt_url" validate:"omitempty,url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	CreatedBy   *string   `json:"created_by,omitempty" db:"created_by" validate:"omitempty,uuid"`
}

// ═══════════════════════════════════════════════════════════════════════
// ValidAdditionalCostCategories для whitelist validation (OWASP ASVS V5.1)
// ═══════════════════════════════════════════════════════════════════════

var ValidAdditionalCostCategories = []string{
	"travel", "subcontractor", "permit", "equipment", "other",
}

// ═══════════════════════════════════════════════════════════════════════
// FeatureFlag — фича-флаг для F-0.2.4 Feature Flags infrastructure.
//
// Compliance:
//   - IEC 62443-3-3 SR 1.1 (Defense in depth — feature gating)
//   - ISO 27001 A.12.1.2 (Change management — controlled rollout)
//   - ISO/IEC 27019 PCC.A.12 (Change management for ICS)
//   - СТБ 34.101.27 (Защита информации — контроль доступа к функциям)
//   - OWASP ASVS V1.1 (Architecture — feature flags как security control)
// ═══════════════════════════════════════════════════════════════════════

type FeatureFlag struct {
	Key         string    `json:"key" db:"key" validate:"required,max=255"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	Description string    `json:"description" db:"description" validate:"max=1000"`
	TenantID    string    `json:"tenant_id" db:"tenant_id" validate:"max=255"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
