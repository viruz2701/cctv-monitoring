// Package workflow — Workflow Engine (WF-9.x).
//
// Визуальный конструктор автоматизаций с CEL-based DSL.
//
// WF-9.1.1: Workflow entity
// WF-9.1.2: WorkflowCondition (DSL)
// WF-9.1.3: WorkflowAction
// WF-9.1.4: Workflow Execution Engine (NATS-driven)
//
// Compliance:
//   - Apache 2.0: cel-go для evaluation
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - IEC 62443 SR 7.1 (Resource availability)
package workflow

import (
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// WF-9.1.1: Workflow entity
// ═══════════════════════════════════════════════════════════════════════

type WorkflowStatus string

const (
	WFActive   WorkflowStatus = "ACTIVE"
	WFInactive WorkflowStatus = "INACTIVE"
	WFDraft    WorkflowStatus = "DRAFT"
)

type Workflow struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name" validate:"required,max=200"`
	Description string         `json:"description,omitempty" db:"description"`
	Status      WorkflowStatus `json:"status" db:"status"`
	Trigger     WorkflowTrigger `json:"trigger" db:"trigger"`         // что запускает workflow
	Conditions  []WorkflowCondition `json:"conditions" db:"conditions"` // условия (AND)
	Actions     []WorkflowAction    `json:"actions" db:"actions"`       // действия при выполнении условий
	CreatedBy   string         `json:"created_by" db:"created_by"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Trigger
// ═══════════════════════════════════════════════════════════════════════

type WorkflowTrigger struct {
	Type    TriggerType `json:"type"`
	Config  TriggerConfig `json:"config"`
}

type TriggerType string

const (
	TriggerEvent  TriggerType = "event"   // NATS event
	TriggerCron   TriggerType = "cron"    // Scheduled (cron expression)
	TriggerManual TriggerType = "manual"  // Manual trigger
	TriggerMeter  TriggerType = "meter"   // Meter threshold exceeded
)

type TriggerConfig struct {
	// Event trigger
	EventSource string `json:"event_source,omitempty"` // "alarms", "cmms", "predictions"
	EventType   string `json:"event_type,omitempty"`   // "alarm.created", "cmms.wo.completed"

	// Cron trigger
	CronExpr    string `json:"cron_expr,omitempty"`    // "0 */1 * * *"

	// Meter trigger
	MeterKind   string `json:"meter_kind,omitempty"`
	MeterThreshold float64 `json:"meter_threshold,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// WF-9.1.2: WorkflowCondition (CEL-based DSL)
// ═══════════════════════════════════════════════════════════════════════

type WorkflowCondition struct {
	Field    string      `json:"field"`              // "event.severity", "event.value", "work_order.priority"
	Operator ConditionOp `json:"operator"`            // "eq", "neq", "gt", "gte", "lt", "lte", "contains", "matches"
	Value    interface{} `json:"value"`               // сравниваемое значение
}

type ConditionOp string

const (
	OpEQ       ConditionOp = "eq"
	OpNEQ      ConditionOp = "neq"
	OpGT       ConditionOp = "gt"
	OpGTE      ConditionOp = "gte"
	OpLT       ConditionOp = "lt"
	OpLTE      ConditionOp = "lte"
	OpContains ConditionOp = "contains"
	OpMatches  ConditionOp = "matches" // regex
)

// ═══════════════════════════════════════════════════════════════════════
// WF-9.1.3: WorkflowAction
// ═══════════════════════════════════════════════════════════════════════

type WorkflowAction struct {
	Type   ActionType    `json:"type"`
	Params ActionParams  `json:"params"`
}

type ActionType string

const (
	ActionCreateWO    ActionType = "CREATE_WO"
	ActionNotify      ActionType = "NOTIFY"
	ActionUpdateStatus ActionType = "UPDATE_STATUS"
	ActionWebhook     ActionType = "WEBHOOK"
	ActionAssign      ActionType = "ASSIGN"
	ActionEscalate    ActionType = "ESCALATE"
)

type ActionParams struct {
	// CREATE_WO
	WorkOrderType string `json:"work_order_type,omitempty"`
	Priority      string `json:"priority,omitempty"`
	TitleTemplate string `json:"title_template,omitempty"`
	DescTemplate  string `json:"desc_template,omitempty"`

	// NOTIFY
	Channel      string   `json:"channel,omitempty"`       // "telegram", "email", "sms"
	Recipients   []string `json:"recipients,omitempty"`
	MessageTemplate string `json:"message_template,omitempty"`

	// UPDATE_STATUS
	TargetStatus string `json:"target_status,omitempty"`

	// WEBHOOK
	WebhookURL  string `json:"webhook_url,omitempty"`
	WebhookMethod string `json:"webhook_method,omitempty"`
	WebhookBody    string `json:"webhook_body,omitempty"`

	// ASSIGN
	AssigneeID string `json:"assignee_id,omitempty"`

	// ESCALATE
	EscalateTo    string `json:"escalate_to,omitempty"` // role or user_id
	EscalateAfter int    `json:"escalate_after_minutes,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Built-in Templates (WF-9.2.x)
// ═══════════════════════════════════════════════════════════════════════

// DefaultTemplates возвращает встроенные шаблоны workflow.
func DefaultTemplates() []*Workflow {
	return []*Workflow{
		{
			Name:        "Critical alarm → Emergency WO",
			Description: "Automatically create emergency work order when critical alarm received",
			Status:      WFActive,
			Trigger: WorkflowTrigger{
				Type: TriggerEvent,
				Config: TriggerConfig{
					EventSource: "alarms",
					EventType:   "alarm.created",
				},
			},
			Conditions: []WorkflowCondition{
				{Field: "event.severity", Operator: OpEQ, Value: "critical"},
			},
			Actions: []WorkflowAction{
				{
					Type: ActionCreateWO,
					Params: ActionParams{
						WorkOrderType: "emergency",
						Priority:      "critical",
						TitleTemplate:  "Emergency: {event.message} on {event.device_name}",
						DescTemplate:   "Critical alarm received: {event.message}\nDevice: {event.device_name}\nSeverity: {event.severity}",
					},
				},
				{
					Type: ActionNotify,
					Params: ActionParams{
						Channel:         "telegram",
						MessageTemplate: "🚨 CRITICAL: {event.message} on {event.device_name}",
					},
				},
			},
		},
		{
			Name:        "Low stock → Create PO",
			Description: "Auto-create purchase order when spare part stock is low",
			Status:      WFActive,
			Trigger: WorkflowTrigger{
				Type: TriggerEvent,
				Config: TriggerConfig{
					EventSource: "cmms",
					EventType:   "spare_part.low_stock",
				},
			},
			Conditions: []WorkflowCondition{
				{Field: "part.stock", Operator: OpLTE, Value: "part.min_stock"},
			},
			Actions: []WorkflowAction{
				{
					Type: ActionCreateWO,
					Params: ActionParams{
						WorkOrderType: "preventive",
						Priority:      "medium",
						TitleTemplate:  "Restock: {part.name} - low stock ({part.stock}/{part.min_stock})",
					},
				},
				{
					Type: ActionNotify,
					Params: ActionParams{
						Channel:         "telegram",
						MessageTemplate: "📦 Low stock: {part.name} ({part.stock} remaining, min {part.min_stock})",
						Recipients:      []string{"inventory-manager"},
					},
				},
			},
		},
		{
			Name:        "Device offline > 1h → Escalate",
			Description: "Escalate to manager if device is offline for more than 1 hour",
			Status:      WFActive,
			Trigger: WorkflowTrigger{
				Type: TriggerEvent,
				Config: TriggerConfig{
					EventSource: "alarms",
					EventType:   "device.offline",
				},
			},
			Conditions: []WorkflowCondition{
				{Field: "event.duration_minutes", Operator: OpGT, Value: float64(60)},
				{Field: "device.asset_class", Operator: OpEQ, Value: "critical"},
			},
			Actions: []WorkflowAction{
				{
					Type: ActionEscalate,
					Params: ActionParams{
						EscalateTo:    "manager",
						EscalateAfter: 60,
					},
				},
				{
					Type: ActionNotify,
					Params: ActionParams{
						Channel:         "telegram",
						MessageTemplate: "⏰ CRITICAL DEVICE OFFLINE: {device.name} offline for {event.duration_minutes} minutes",
					},
				},
			},
		},
	}
}
