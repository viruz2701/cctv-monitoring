package models

import (
	"encoding/json"
	"time"
)

// WorkOrderAlert — связь WorkOrder ↔ Alert (Many-to-Many).
//
// Позволяет привязывать неограниченное количество алертов к наряду
// и наоборот — один алерт может быть связан с несколькими нарядами.
// alert_id — TEXT без FK, так как алерты могут поступать из внешних систем.
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application security)
//   - ISO 27001 A.12.4.1 (Event logging — linked_at audit trail)
//   - СТБ 34.101.27 (Защита информации — связь инцидентов)
//
// Ref: DM-1.3.1
type WorkOrderAlert struct {
	WorkOrderID string    `json:"work_order_id" db:"work_order_id"`
	AlertID     string    `json:"alert_id" db:"alert_id"`
	LinkedAt    time.Time `json:"linked_at" db:"linked_at"`
	LinkedBy    string    `json:"linked_by,omitempty" db:"linked_by"`
}

// TechnicianSiteAssignment — закрепление техника за объектом
type TechnicianSiteAssignment struct {
	ID           string    `json:"id" db:"id"`
	TechnicianID string    `json:"technician_id" db:"technician_id"`
	SiteID       string    `json:"site_id" db:"site_id"`
	IsPrimary    bool      `json:"is_primary" db:"is_primary"`
	AssignedAt   time.Time `json:"assigned_at" db:"assigned_at"`
	AssignedBy   string    `json:"assigned_by" db:"assigned_by"`

	// Denormalized
	TechnicianName string `json:"technician_name,omitempty"`
	SiteName       string `json:"site_name,omitempty"`
}

// MaintenanceSchedule — график планового ТО
type MaintenanceSchedule struct {
	ID               string          `json:"id" db:"id"`
	DeviceID         string          `json:"device_id" db:"device_id"`
	ScheduleType     string          `json:"schedule_type" db:"schedule_type"` // daily, weekly, monthly, quarterly, custom
	IntervalDays     int             `json:"interval_days" db:"interval_days"`
	CustomCron       string          `json:"custom_cron,omitempty" db:"custom_cron"`
	LastCompleted    *time.Time      `json:"last_completed,omitempty" db:"last_completed"`
	NextDue          time.Time       `json:"next_due" db:"next_due"`
	AssignedTo       *string         `json:"assigned_to,omitempty" db:"assigned_to"`
	Checklist        json.RawMessage `json:"checklist" db:"checklist"` // [{task: string, completed: bool}]
	EstimatedMinutes int             `json:"estimated_minutes" db:"estimated_minutes"`
	Priority         string          `json:"priority" db:"priority"`
	Notes            string          `json:"notes,omitempty" db:"notes"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`

	// Denormalized для UI
	DeviceName   string `json:"device_name,omitempty"`
	AssigneeName string `json:"assignee_name,omitempty"`
}

// ChecklistItem — элемент чек-листа
type ChecklistItem struct {
	Task      string `json:"task"`
	Completed bool   `json:"completed"`
}

// WorkOrder — наряд на работу.
// Сохраняет обратную совместимость с существующим кодом.
// Новые сущности (Request, PreventiveMaintenance) используют полноценный
// Go embedding от WorkOrderBase.
//
// Compliance: IEC 62443 SL-3, ISO 27001 A.12.4, СТБ 34.101.27, OWASP ASVS V5
// Ref: com.grash.model.WorkOrder
type WorkOrder struct {
	ID          string          `json:"id" db:"id"`
	ScheduleID  *string         `json:"schedule_id,omitempty" db:"schedule_id"`
	DeviceID    string          `json:"device_id" db:"device_id"`
	Title       string          `json:"title,omitempty" db:"title"`
	Type        string          `json:"type" db:"type"`     // preventive, corrective, emergency
	Status      string          `json:"status" db:"status"` // open, in_progress, completed, cancelled
	Priority    string          `json:"priority" db:"priority"`
	AssignedTo  *string         `json:"assigned_to,omitempty" db:"assigned_to"`
	DueDate     *time.Time      `json:"due_date,omitempty" db:"due_date"`
	SLADeadline *time.Time      `json:"sla_deadline,omitempty" db:"sla_deadline"`
	Checklist   json.RawMessage `json:"checklist" db:"checklist"`
	StartedAt   *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	Notes       string          `json:"notes,omitempty" db:"notes"`
	Photos      json.RawMessage `json:"photos" db:"photos"` // []string URLs
	PartsUsed   json.RawMessage `json:"parts_used" db:"parts_used"`
	CreatedBy   *string         `json:"created_by,omitempty" db:"created_by"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	DeletedAt   *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`

	// Denormalized
	DeviceName   string `json:"device_name,omitempty"`
	AssigneeName string `json:"assignee_name,omitempty"`
	SLAStatus    string `json:"sla_status,omitempty"` // on_track, at_risk, breached
}

// WorkOrderCreateRequest — DTO для создания WorkOrder (с whitelist validation).
// Compliance: OWASP ASVS V5.1 (Input validation)
type WorkOrderCreateRequest struct {
	ScheduleID  *string    `json:"schedule_id,omitempty" validate:"omitempty,uuid"`
	DeviceID    string     `json:"device_id" validate:"required,uuid"`
	Title       string     `json:"title" validate:"required,min=1,max=500"`
	Type        string     `json:"type" validate:"required,oneof=preventive corrective emergency routine inspection"`
	Priority    string     `json:"priority" validate:"required,oneof=critical high medium low"`
	AssignedTo  *string    `json:"assigned_to,omitempty" validate:"omitempty,uuid"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	SLADeadline *time.Time `json:"sla_deadline,omitempty"`
	Notes       string     `json:"notes,omitempty" validate:"max=5000"`
}

// PreventiveMaintenance — плановое ТО (Preventive Maintenance).
// Embed WorkOrderBase для общих полей title/priority/status/assignee/dueDate.
//
// Compliance: ISO 27001 A.12.6 (Maintenance), IEC 62443 SR 7.1 (Resource availability)
// Ref: com.grash.model.PreventiveMaintenance
type PreventiveMaintenance struct {
	WorkOrderBase // title, priority, status, assignee, dueDate
	AuditBase     // created_at, updated_at, created_by, updated_by
	SoftDeleteMixin

	ID               string          `json:"id" db:"id"`
	DeviceID         string          `json:"device_id" db:"device_id" validate:"required,uuid"`
	ScheduleType     string          `json:"schedule_type" db:"schedule_type" validate:"required,oneof=daily weekly monthly quarterly custom"`
	IntervalDays     int             `json:"interval_days" db:"interval_days" validate:"min=0"`
	CustomCron       string          `json:"custom_cron,omitempty" db:"custom_cron"`
	LastCompleted    *time.Time      `json:"last_completed,omitempty" db:"last_completed"`
	NextDue          time.Time       `json:"next_due" db:"next_due"`
	Checklist        json.RawMessage `json:"checklist" db:"checklist"`
	EstimatedMinutes int             `json:"estimated_minutes" db:"estimated_minutes" validate:"min=1"`
	Notes            string          `json:"notes,omitempty" db:"notes" validate:"max=5000"`

	// Denormalized для UI
	DeviceName   string `json:"device_name,omitempty"`
	AssigneeName string `json:"assignee_name,omitempty"`
}

// PartsConsumption — списание запчастей с фиксацией цены на момент списания.
// Цена фиксируется (snapshot) — не пересчитывается при изменении цены запчасти.
//
// Compliance: ISO 27001 A.12.6 (Cost tracking audit trail)
// Ref: com.grash.model.PartQuantity, PartQuantityPatchDTO
type PartsConsumption struct {
	CostBase // estimated_cost, actual_cost, currency

	ID          string    `json:"id" db:"id"`
	WorkOrderID string    `json:"work_order_id" db:"work_order_id" validate:"required,uuid"`
	PartID      string    `json:"part_id" db:"part_id" validate:"required,uuid"`
	Quantity    int       `json:"quantity" db:"quantity" validate:"required,min=1"`
	UnitPrice   float64   `json:"unit_price" db:"unit_price" validate:"min=0"` // snapshot at consumption time
	TotalPrice  float64   `json:"total_price" db:"total_price" validate:"min=0"`
	UsedBy      *string   `json:"used_by,omitempty" db:"used_by" validate:"omitempty,uuid"`
	UsedAt      time.Time `json:"used_at" db:"used_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`

	// Denormalized
	PartName string `json:"part_name,omitempty"`
	PartSKU  string `json:"part_sku,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

// PartUsage — использование запчасти в наряде (legacy, использовать PartsConsumption).
type PartUsage struct {
	PartID   string `json:"part_id"`
	Quantity int    `json:"quantity"`
}

// SparePart — запчасть
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — stock tracking)
//   - OWASP ASVS V6 (Stored cryptography — JSONB custom_fields)
//
// Ref: INV-7.1.2 (Custom Fields JSONB), INV-7.2.1 (Vendor reference)
type SparePart struct {
	ID                string          `json:"id" db:"id"`
	Name              string          `json:"name" db:"name"`
	SKU               string          `json:"sku" db:"sku"`
	Category          string          `json:"category,omitempty" db:"category"`
	Stock             int             `json:"stock" db:"stock"`
	MinStock          int             `json:"min_stock" db:"min_stock"`
	Location          string          `json:"location,omitempty" db:"location"`
	CompatibleDevices []string        `json:"compatible_devices" db:"compatible_devices"`
	Cost              float64         `json:"cost" db:"cost"`
	Supplier          string          `json:"supplier,omitempty" db:"supplier"`
	VendorID          *string         `json:"vendor_id,omitempty" db:"vendor_id"`
	CustomFields      json.RawMessage `json:"custom_fields,omitempty" db:"custom_fields"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// StockAdjustment — запись корректировки остатка запчасти (audit trail).
//
// Фиксирует мутацию stock: previous_stock, new_stock, delta, reason.
// Каждая запись подписывается через audit_log (ISO 27001 A.12.4).
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.4.1 (Event logging — stock adjustment audit trail)
//   - ISO/IEC 27019 PCC.A.12 (Operations security — inventory changes)
//   - СТБ 34.101.27 (Защита информации — audit trail для складских операций)
//   - OWASP ASVS V7.1 (Structured audit log)
//
// Ref: INV-7.1.4
type StockAdjustment struct {
	ID            string    `json:"id" db:"id"`
	PartID        string    `json:"part_id" db:"part_id"`
	PreviousStock int       `json:"previous_stock" db:"previous_stock"`
	NewStock      int       `json:"new_stock" db:"new_stock"`
	Delta         int       `json:"delta" db:"delta"`
	Reason        string    `json:"reason,omitempty" db:"reason"`
	AdjustedBy    string    `json:"adjusted_by,omitempty" db:"adjusted_by"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`

	// Denormalized для UI
	PartName string `json:"part_name,omitempty"`
	PartSKU  string `json:"part_sku,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

// SLAConfig — конфигурация SLA
type SLAConfig struct {
	ID                    string `json:"id" db:"id"`
	Priority              string `json:"priority" db:"priority"`
	ResponseTimeMinutes   int    `json:"response_time_minutes" db:"response_time_minutes"`
	ResolutionTimeMinutes int    `json:"resolution_time_minutes" db:"resolution_time_minutes"`
}

// TechnicianWorkload — нагрузка техника
type TechnicianWorkload struct {
	UserID          string   `json:"user_id"`
	UserName        string   `json:"user_name"`
	CurrentWorkload int      `json:"current_workload"`
	MaxWorkload     int      `json:"max_workload"`
	Skills          []string `json:"skills"`
	BaseLocation    *string  `json:"base_location"`
}

// MaintenanceReport — отчёт по обслуживанию
type MaintenanceReport struct {
	DeviceID        string  `json:"device_id"`
	DeviceName      string  `json:"device_name"`
	MTBF            float64 `json:"mtbf_hours"`   // Mean Time Between Failures
	MTTR            float64 `json:"mttr_minutes"` // Mean Time To Repair
	TotalWorkOrders int     `json:"total_work_orders"`
	CompletedCount  int     `json:"completed_count"`
	OverdueCount    int     `json:"overdue_count"`
	TotalCost       float64 `json:"total_cost"`
}

// SLAComplianceReport — отчёт по соблюдению SLA
type SLAComplianceReport struct {
	Priority          string  `json:"priority"`
	TotalWorkOrders   int     `json:"total_work_orders"`
	WithinSLA         int     `json:"within_sla"`
	BreachedSLA       int     `json:"breached_sla"`
	CompliancePercent float64 `json:"compliance_percent"`
	AvgResponseTime   float64 `json:"avg_response_minutes"`
	AvgResolutionTime float64 `json:"avg_resolution_minutes"`
}

// DeviceReliability — метрики надёжности устройства (AN-10.1.1).
//
// MTBF (Mean Time Between Failures) — среднее время между отказами в часах.
// Рассчитывается как: (общее время работы) / (количество отказов).
// MTTR (Mean Time To Repair) — среднее время восстановления в минутах.
// Берётся из mv_device_reliability.avg_mttr_minutes.
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — reliability metrics)
//   - IEC 62443 SR 7.1 (Resource availability — MTBF tracking)
//   - СТБ 34.101.27 п. 7.3 (Анализ защищённости)
type DeviceReliability struct {
	VendorType           string  `json:"vendor_type"`
	DeviceType           string  `json:"device_type"`
	DeviceCount          int64   `json:"device_count"`
	TotalDowntimeEvents  int64   `json:"total_downtime_events"`
	TotalDowntimeMinutes int64   `json:"total_downtime_minutes"`
	TotalCompletions     int64   `json:"total_completions"`
	AvgMTTRMinutes       float64 `json:"avg_mttr_minutes"`
	MTBFHours            float64 `json:"mtbf_hours"`
	MTTRMinutes          float64 `json:"mttr_minutes"`
}

// TechnicianMonthlyStats — статистика техника за текущий месяц
type TechnicianMonthlyStats struct {
	CompletedThisMonth int     `json:"completed_this_month"`
	TotalWorkOrders    int     `json:"total_work_orders"`
	OnTimePercent      float64 `json:"on_time_percent"`
	AvgRating          float64 `json:"avg_rating"`
}

// Site — объект (площадка) видеонаблюдения.
// Поддерживает иерархию локаций: Building → Floor → Room → Rack.
//
// Compliance: IEC 62443 SL-3 (Zone 3 — Asset management)
type Site struct {
	ID               string     `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	Address          string     `json:"address,omitempty" db:"address"`
	City             string     `json:"city,omitempty" db:"city"`
	Organization     string     `json:"organization,omitempty" db:"organization"`
	Latitude         float64    `json:"latitude,omitempty" db:"latitude"`
	Longitude        float64    `json:"longitude,omitempty" db:"longitude"`
	Status           string     `json:"status" db:"status"`
	ParentLocationID *string    `json:"parent_location_id,omitempty" db:"parent_location_id"`
	LocationType     string     `json:"location_type" db:"location_type"` // building, floor, room, rack
	LastSync         *time.Time `json:"last_sync,omitempty" db:"last_sync"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

// SparePartCategory — категория запчасти
type SparePartCategory struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	Color       string    `json:"color,omitempty" db:"color"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.2.2: Escalation Models
// ═══════════════════════════════════════════════════════════════════════

// EscalationRule — правило эскалации SLA breach.
//
// Определяет кому и когда отправлять уведомление при нарушении SLA.
// 3 уровня эскалации: L1 (manager), L2 (director), L3 (emergency).
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — escalation notifications)
//   - IEC 62443 SR 2.8 (Audit events — escalation tracking)
//   - OWASP ASVS V7.1 (Structured log content)
//   - Приказ ОАЦ №66 п. 7.18.3 (Incident response)
type EscalationRule struct {
	ID                    string    `json:"id" db:"id"`
	Priority              string    `json:"priority" db:"priority"`
	EscalationLevel       int       `json:"escalation_level" db:"escalation_level"`
	BreachMinutes         int       `json:"breach_minutes" db:"breach_minutes"`
	NotifyRole            string    `json:"notify_role" db:"notify_role"`
	NotifyChannel         string    `json:"notify_channel" db:"notify_channel"`
	RepeatIntervalMinutes int       `json:"repeat_interval_minutes" db:"repeat_interval_minutes"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// EscalationLogEntry — запись в журнале эскалаций.
//
// Фиксирует факт отправки уведомления, подтверждение получения
// и заметки по разрешению инцидента.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — full audit trail)
//   - IEC 62443 SR 2.8 (Audit events — escalation chain)
//   - СТБ 34.101.27 (Защита информации — audit log)
type EscalationLogEntry struct {
	ID              string     `json:"id" db:"id"`
	WorkOrderID     string     `json:"work_order_id" db:"work_order_id"`
	EscalationLevel int        `json:"escalation_level" db:"escalation_level"`
	RuleID          string     `json:"rule_id" db:"rule_id"`
	NotifiedAt      time.Time  `json:"notified_at" db:"notified_at"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty" db:"acknowledged_at"`
	AcknowledgedBy  *string    `json:"acknowledged_by,omitempty" db:"acknowledged_by"`
	ResolutionNotes string     `json:"resolution_notes,omitempty" db:"resolution_notes"`
}

// ═══════════════════════════════════════════════════════════════════════
// INV-7.2.1: Vendor entity
// ═══════════════════════════════════════════════════════════════════════

// Vendor — поставщик (поставщик запчастей, оборудования и услуг).
//
// Хранит контактную информацию и статус поставщика.
// Может быть связан с запчастями (SparePart.VendorID).
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.15.1.1 (Supplier security policy — vendor management)
//   - ISO/IEC 27019 PCC.A.5 (Supply chain management)
//   - СТБ 34.101.27 (Защита информации — управление поставщиками)
//   - OWASP ASVS V5.1 (Whitelist validation через status CHECK)
//
// Ref: INV-7.2.1
type Vendor struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	ContactPerson string    `json:"contact_person,omitempty" db:"contact_person"`
	Email         string    `json:"email,omitempty" db:"email"`
	Phone         string    `json:"phone,omitempty" db:"phone"`
	Address       string    `json:"address,omitempty" db:"address"`
	Website       string    `json:"website,omitempty" db:"website"`
	Notes         string    `json:"notes,omitempty" db:"notes"`
	Status        string    `json:"status" db:"status"` // active, inactive
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// VendorCreateRequest — DTO для создания Vendor (с whitelist validation).
// Compliance: OWASP ASVS V5.1 (Input validation)
type VendorCreateRequest struct {
	Name          string `json:"name" validate:"required,min=1,max=255"`
	ContactPerson string `json:"contact_person,omitempty" validate:"max=255"`
	Email         string `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Phone         string `json:"phone,omitempty" validate:"max=50"`
	Address       string `json:"address,omitempty" validate:"max=500"`
	Website       string `json:"website,omitempty" validate:"omitempty,url,max=500"`
	Notes         string `json:"notes,omitempty" validate:"max=5000"`
	Status        string `json:"status,omitempty" validate:"omitempty,oneof=active inactive"`
}

// ═══════════════════════════════════════════════════════════════════════
// AN-10.1.3: TCO (Total Cost of Ownership) per device
// ═══════════════════════════════════════════════════════════════════════

// TCOPerDevice — Total Cost of Ownership per device.
//
// Формула: TCO = Purchase + Labor + Parts + Downtime
// Данные берутся из материализованного представления mv_tco_per_device.
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
//   - IEC 62443 SR 7.1 (Resource availability — asset TCO)
//   - ISO/IEC 27019 PCC.A.10 (Cost management for ICS assets)
//   - СТБ 34.101.27 (Защита информации — учёт стоимости активов)
//
// Ref: AN-10.1.3
type TCOPerDevice struct {
	DeviceID            string  `json:"device_id" db:"device_id"`
	DeviceName          string  `json:"device_name" db:"device_name"`
	VendorType          string  `json:"vendor_type" db:"vendor_type"`
	DeviceType          string  `json:"device_type" db:"device_type"`
	Manufacturer        string  `json:"manufacturer" db:"manufacturer"`
	TotalPurchaseCost   float64 `json:"total_purchase_cost" db:"total_purchase_cost"`
	TotalLaborCost      float64 `json:"total_labor_cost" db:"total_labor_cost"`
	TotalPartsCost      float64 `json:"total_parts_cost" db:"total_parts_cost"`
	TotalDowntimeCost   float64 `json:"total_downtime_cost" db:"total_downtime_cost"`
	TCO                 float64 `json:"tco" db:"tco"`
	TotalWorkOrders     int64   `json:"total_work_orders" db:"total_work_orders"`
	TotalDowntimeEvents int64   `json:"total_downtime_events" db:"total_downtime_events"`
}

// ═══════════════════════════════════════════════════════════════════════
// WO-4.4.5: WorkOrderCostSummary — сводка затрат по Work Orders
// ═══════════════════════════════════════════════════════════════════════

// WorkOrderCostSummary — агрегированная сводка затрат по Work Orders.
//
// Compliance:
//   - ISO 27001 A.12.6.1 (Capacity management — cost tracking)
//   - IEC 62443 SR 7.1 (Resource availability)
//   - OWASP ASVS V7.1 (Structured response — no sensitive data)
type WorkOrderCostSummary struct {
	TotalWorkOrders     int64   `json:"total_work_orders" db:"total_work_orders"`
	TotalLaborCost      float64 `json:"total_labor_cost" db:"total_labor_cost"`
	TotalPartsCost      float64 `json:"total_parts_cost" db:"total_parts_cost"`
	TotalAdditionalCost float64 `json:"total_additional_cost" db:"total_additional_cost"`
	TotalCost           float64 `json:"total_cost" db:"total_cost"`
	AverageCostPerOrder float64 `json:"avg_cost_per_order" db:"avg_cost_per_order"`
	Currency            string  `json:"currency"`
}

// WorkOrderCostBreakdown — детальная разбивка затрат по категориям.
//
// Compliance:
//   - OWASP ASVS V5.1 (Whitelist validation — category enum)
type WorkOrderCostBreakdown struct {
	Category string  `json:"category"` // labor, parts, additional
	Amount   float64 `json:"amount"`
	Count    int64   `json:"count"`
	Percent  float64 `json:"percent"`
}

// TCOFilter — параметры фильтрации для GetTCOPerDevice.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist через query params)
//   - OWASP ASVS V5.3 (Output encoding — поля не содержат SQL)
type TCOFilter struct {
	VendorType string `json:"vendor_type,omitempty"`
	DeviceType string `json:"device_type,omitempty"`
	DeviceID   string `json:"device_id,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}
