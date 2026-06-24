package models

import (
	"encoding/json"
	"time"
)

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

// WorkOrder — наряд на работу
type WorkOrder struct {
	ID          string          `json:"id" db:"id"`
	ScheduleID  *string         `json:"schedule_id,omitempty" db:"schedule_id"`
	DeviceID    string          `json:"device_id" db:"device_id"`
	Type        string          `json:"type" db:"type"`     // preventive, corrective, emergency
	Status      string          `json:"status" db:"status"` // open, in_progress, completed, cancelled
	Priority    string          `json:"priority" db:"priority"`
	AssignedTo  *string         `json:"assigned_to,omitempty" db:"assigned_to"`
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

	// Denormalized
	DeviceName   string `json:"device_name,omitempty"`
	AssigneeName string `json:"assignee_name,omitempty"`
	SLAStatus    string `json:"sla_status,omitempty"` // on_track, at_risk, breached
}

// PartUsage — использование запчасти в наряде
type PartUsage struct {
	PartID   string `json:"part_id"`
	Quantity int    `json:"quantity"`
}

// SparePart — запчасть
type SparePart struct {
	ID                string    `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	SKU               string    `json:"sku" db:"sku"`
	Category          string    `json:"category,omitempty" db:"category"`
	Stock             int       `json:"stock" db:"stock"`
	MinStock          int       `json:"min_stock" db:"min_stock"`
	Location          string    `json:"location,omitempty" db:"location"`
	CompatibleDevices []string  `json:"compatible_devices" db:"compatible_devices"`
	Cost              float64   `json:"cost" db:"cost"`
	Supplier          string    `json:"supplier,omitempty" db:"supplier"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
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
	BaseLocation    string   `json:"base_location"`
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

// TechnicianMonthlyStats — статистика техника за текущий месяц
type TechnicianMonthlyStats struct {
	CompletedThisMonth int     `json:"completed_this_month"`
	TotalWorkOrders    int     `json:"total_work_orders"`
	OnTimePercent      float64 `json:"on_time_percent"`
	AvgRating          float64 `json:"avg_rating"`
}

// Site — объект (площадка) видеонаблюдения
type Site struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Address      string    `json:"address,omitempty" db:"address"`
	City         string    `json:"city,omitempty" db:"city"`
	Organization string    `json:"organization,omitempty" db:"organization"`
	Latitude     float64   `json:"latitude,omitempty" db:"latitude"`
	Longitude    float64   `json:"longitude,omitempty" db:"longitude"`
	Status       string    `json:"status" db:"status"`
	LastSync     *time.Time `json:"last_sync,omitempty" db:"last_sync"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
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
