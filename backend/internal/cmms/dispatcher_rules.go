// Package cmms — Dispatch Rules Engine.
//
// RuleEngine реализует систему правил для автоматической диспетчеризации.
// Правила состоят из условий (conditions) и действий (actions).
//
// Пример правила:
//
//	rule = {
//	  name: "Critical priority auto-assign",
//	  conditions: [
//	    {field: "priority", operator: "eq", value: "critical"},
//	    {field: "assigned_to", operator: "eq", value: ""}
//	  ],
//	  action: {type: "assign_to_team", params: {team: "emergency"}}
//	}
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure — при ошибке парсинга правила пропускаются)
//   - IEC 62443 SR 3.1 (Data integrity — условия не содержат SQL)
//   - ISO 27001 A.12.4.1 (Event logging — каждое срабатывание правила)
//   - OWASP ASVS V5.1 (Input validation — whitelist операторов)
//   - OWASP ASVS V7.1 (Log content — структурированные логи)
//   - СТБ 34.101.27 п. 7.2 (Audit trail)
package cmms

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gb-telemetry-collector/internal/models"
)

// ═══════════════════════════════════════════════════════════════════════
// Rule Types
// ═══════════════════════════════════════════════════════════════════════

// ConditionOperator — оператор сравнения для условия.
type ConditionOperator string

const (
	OpEquals    ConditionOperator = "eq"
	OpNotEquals ConditionOperator = "ne"
	OpContains  ConditionOperator = "contains"
	OpGreater   ConditionOperator = "gt"
	OpLess      ConditionOperator = "lt"
	OpIn        ConditionOperator = "in"
	OpNotEmpty  ConditionOperator = "not_empty"
	OpIsEmpty   ConditionOperator = "is_empty"
)

// ValidOperators — whitelist валидных операторов (OWASP ASVS V5.1).
var ValidOperators = map[ConditionOperator]bool{
	OpEquals:    true,
	OpNotEquals: true,
	OpContains:  true,
	OpGreater:   true,
	OpLess:      true,
	OpIn:        true,
	OpNotEmpty:  true,
	OpIsEmpty:   true,
}

// ActionType — тип действия правила.
type ActionType string

const (
	ActionAssignToTeam      ActionType = "assign_to_team"
	ActionEscalateToManager ActionType = "escalate_to_manager"
	ActionNotify            ActionType = "notify"
	ActionSetPriority       ActionType = "set_priority"
	ActionAutoAssign        ActionType = "auto_assign"
	ActionSetSLADeadline    ActionType = "set_sla_deadline"
)

// ValidActionTypes — whitelist валидных действий.
var ValidActionTypes = map[ActionType]bool{
	ActionAssignToTeam:      true,
	ActionEscalateToManager: true,
	ActionNotify:            true,
	ActionSetPriority:       true,
	ActionAutoAssign:        true,
	ActionSetSLADeadline:    true,
}

// ═══════════════════════════════════════════════════════════════════════
// DispatchRule — правило диспетчеризации
// ═══════════════════════════════════════════════════════════════════════

// Condition — условие правила.
//
// Поле field может быть: priority, status, type, assigned_to, device_id, sla_status.
// Оператор operator: eq, ne, contains, gt, lt, in, not_empty, is_empty.
// Значение value зависит от оператора.
//
// Compliance:
//   - OWASP ASVS V5.1 (Whitelist validation — field и operator)
//   - OWASP ASVS V5.3 (Output encoding — value не содержит SQL)
type Condition struct {
	Field    string            `json:"field" validate:"required"`
	Operator ConditionOperator `json:"operator" validate:"required"`
	Value    string            `json:"value"`
}

// Action — действие правила.
type Action struct {
	Type   ActionType             `json:"type" validate:"required"`
	Params map[string]interface{} `json:"params,omitempty"`
}

// DispatchRule — правило для автоматической диспетчеризации.
//
// Правило состоит из:
//   - ID: уникальный идентификатор
//   - Name: человекочитаемое имя
//   - Description: описание (что делает правило)
//   - Enabled: активно ли правило
//   - Priority: приоритет правила (меньше = выше приоритет)
//   - Conditions: список условий (AND — все должны совпасть)
//   - Action: действие при совпадении условий
//   - CreatedAt/UpdatedAt: временные метки
//
// Compliance:
//   - IEC 62443 SR 3.1 (Data integrity — rule audit trail)
//   - ISO 27001 A.12.4.1 (Event logging — rule changes)
//   - OWASP ASVS V5.1 (Input validation — whitelist)
type DispatchRule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name" validate:"required,max=200"`
	Description string      `json:"description,omitempty" validate:"max=1000"`
	Enabled     bool        `json:"enabled"`
	Priority    int         `json:"priority"` // 1 = highest
	Conditions  []Condition `json:"conditions" validate:"required,min=1"`
	Action      Action      `json:"action" validate:"required"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Default Rules
// ═══════════════════════════════════════════════════════════════════════

// DefaultDispatchRules возвращает правила диспетчеризации по умолчанию.
func DefaultDispatchRules() []DispatchRule {
	now := time.Now().UTC()
	return []DispatchRule{
		{
			ID:          "rule-critical-assign",
			Name:        "Critical WO — auto assign",
			Description: "Critical priority Work Orders автоматически назначаются на доступного техника",
			Enabled:     true,
			Priority:    1,
			Conditions: []Condition{
				{Field: "priority", Operator: OpEquals, Value: "critical"},
				{Field: "assigned_to", Operator: OpIsEmpty, Value: ""},
			},
			Action: Action{
				Type: ActionAutoAssign,
				Params: map[string]interface{}{
					"max_delay_minutes": 5,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "rule-sla-breach-escalation",
			Name:        "SLA breach — escalate to manager",
			Description: "При нарушении SLA дедлайна — эскалация на manager",
			Enabled:     true,
			Priority:    2,
			Conditions: []Condition{
				{Field: "sla_status", Operator: OpEquals, Value: "breached"},
			},
			Action: Action{
				Type: ActionEscalateToManager,
				Params: map[string]interface{}{
					"level":   1,
					"channel": "telegram",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "rule-unassigned-escalation",
			Name:        "Unassigned > 2h — escalate",
			Description: "Непривязанные Work Orders старше 2 часов эскалируются",
			Enabled:     true,
			Priority:    3,
			Conditions: []Condition{
				{Field: "assigned_to", Operator: OpIsEmpty, Value: ""},
				{Field: "age_minutes", Operator: OpGreater, Value: "120"},
			},
			Action: Action{
				Type: ActionNotify,
				Params: map[string]interface{}{
					"role":    "dispatcher",
					"channel": "telegram",
					"message": "Work Order {{id}} не назначен более 2 часов",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "rule-emergency-team",
			Name:        "Emergency — assign to emergency team",
			Description: "Emergency Work Orders назначаются на emergency team",
			Enabled:     true,
			Priority:    4,
			Conditions: []Condition{
				{Field: "type", Operator: OpEquals, Value: "emergency"},
			},
			Action: Action{
				Type: ActionAssignToTeam,
				Params: map[string]interface{}{
					"team": "emergency",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:          "rule-high-priority-notify",
			Name:        "High priority — notify dispatcher",
			Description: "High priority WO уведомляет диспетчера",
			Enabled:     true,
			Priority:    5,
			Conditions: []Condition{
				{Field: "priority", Operator: OpEquals, Value: "high"},
				{Field: "assigned_to", Operator: OpIsEmpty, Value: ""},
			},
			Action: Action{
				Type: ActionNotify,
				Params: map[string]interface{}{
					"role":    "dispatcher",
					"channel": "telegram",
					"message": "High priority WO {{id}} ожидает назначения",
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════
// RuleEngine
// ═══════════════════════════════════════════════════════════════════════

// RuleEngine — движок правил диспетчеризации.
//
// Оценивает Work Order по всем активным правилам и возвращает список действий.
// Правила сортируются по приоритету (меньше = выше).
// При срабатывании правила, его действие добавляется в результат.
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure — при ошибке парсинга условия пропускаем)
//   - IEC 62443 SR 3.1 (Data integrity — условия без side effects)
//   - ISO 27001 A.12.4.1 (Event logging — каждое срабатывание)
type RuleEngine struct {
	mu     sync.RWMutex
	rules  []DispatchRule
	logger *slog.Logger
}

// NewRuleEngine создаёт новый RuleEngine с правилами по умолчанию.
func NewRuleEngine(logger *slog.Logger) *RuleEngine {
	if logger == nil {
		logger = slog.Default()
	}

	return &RuleEngine{
		rules:  DefaultDispatchRules(),
		logger: logger.With("component", "cmms-rule-engine"),
	}
}

// NewRuleEngineWithRules создаёт RuleEngine с указанными правилами.
func NewRuleEngineWithRules(rules []DispatchRule, logger *slog.Logger) *RuleEngine {
	if logger == nil {
		logger = slog.Default()
	}

	if rules == nil {
		rules = DefaultDispatchRules()
	}

	return &RuleEngine{
		rules:  rules,
		logger: logger.With("component", "cmms-rule-engine"),
	}
}

// ── Rule management ──────────────────────────────────────────────────

// GetRules возвращает копию всех правил.
func (e *RuleEngine) GetRules() []DispatchRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]DispatchRule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

// GetRule возвращает правило по ID.
func (e *RuleEngine) GetRule(id string) (*DispatchRule, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, r := range e.rules {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("rule %s not found", id)
}

// AddRule добавляет новое правило.
//
// Валидация:
//   - ID не должен быть пустым
//   - Имя не должно быть пустым
//   - Должно быть хотя бы одно условие
//   - Действие должно быть валидным
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation)
//   - ISO 27001 A.12.4.1 (Event logging)
func (e *RuleEngine) AddRule(rule DispatchRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule id is required")
	}
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	if !ValidActionTypes[rule.Action.Type] {
		return fmt.Errorf("invalid action type: %s", rule.Action.Type)
	}
	for _, c := range rule.Conditions {
		if !ValidOperators[c.Operator] {
			return fmt.Errorf("invalid operator: %s", c.Operator)
		}
	}

	now := time.Now().UTC()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	e.mu.Lock()
	defer e.mu.Unlock()

	// Проверяем дубликаты
	for _, r := range e.rules {
		if r.ID == rule.ID {
			return fmt.Errorf("rule with id %s already exists", rule.ID)
		}
	}

	e.rules = append(e.rules, rule)
	e.sortRules()

	e.logger.Info("rule added", "rule_id", rule.ID, "rule_name", rule.Name)
	return nil
}

// UpdateRule обновляет существующее правило.
func (e *RuleEngine) UpdateRule(id string, updates map[string]interface{}) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == id {
			// Применяем обновления
			if name, ok := updates["name"]; ok {
				if nameStr, ok := name.(string); ok {
					e.rules[i].Name = nameStr
				}
			}
			if desc, ok := updates["description"]; ok {
				if descStr, ok := desc.(string); ok {
					e.rules[i].Description = descStr
				}
			}
			if enabled, ok := updates["enabled"]; ok {
				if enabledBool, ok := enabled.(bool); ok {
					e.rules[i].Enabled = enabledBool
				}
			}
			if priority, ok := updates["priority"]; ok {
				if priorityFloat, ok := priority.(float64); ok {
					e.rules[i].Priority = int(priorityFloat)
				}
			}
			if conditions, ok := updates["conditions"]; ok {
				if condBytes, err := json.Marshal(conditions); err == nil {
					var conds []Condition
					if json.Unmarshal(condBytes, &conds) == nil {
						e.rules[i].Conditions = conds
					}
				}
			}
			if action, ok := updates["action"]; ok {
				if actBytes, err := json.Marshal(action); err == nil {
					var act Action
					if json.Unmarshal(actBytes, &act) == nil {
						e.rules[i].Action = act
					}
				}
			}

			e.rules[i].UpdatedAt = time.Now().UTC()
			e.sortRules()

			e.logger.Info("rule updated", "rule_id", id)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", id)
}

// DeleteRule удаляет правило по ID.
func (e *RuleEngine) DeleteRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, r := range e.rules {
		if r.ID == id {
			e.rules = append(e.rules[:i], e.rules[i+1:]...)
			e.logger.Info("rule deleted", "rule_id", id)
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", id)
}

// ── Evaluation ───────────────────────────────────────────────────────

// Evaluate оценивает Work Order по всем активным правилам.
//
// Возвращает список действий для выполнения.
// Правила сортируются по приоритету (меньше = выше).
// Если ни одно правило не сработало — возвращает пустой список.
//
// Compliance:
//   - IEC 62443 SR 7.1 (Fail Secure — ошибки не прерывают评估цию)
//   - OWASP ASVS V7.1 (Log content — без sensitive data)
func (e *RuleEngine) Evaluate(wo *models.WorkOrder) []RuleActionResult {
	if wo == nil {
		e.logger.Warn("evaluate called with nil work order")
		return nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	var results []RuleActionResult

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		// Проверяем все условия (AND)
		matched := true
		for _, cond := range rule.Conditions {
			if !e.evaluateCondition(wo, cond) {
				matched = false
				break
			}
		}

		if matched {
			result := RuleActionResult{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Action:    rule.Action,
				MatchedAt: time.Now().UTC(),
			}
			results = append(results, result)

			e.logger.Info("rule matched",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"work_order_id", wo.ID,
				"action_type", rule.Action.Type,
			)
		}
	}

	return results
}

// evaluateCondition оценивает одно условие для Work Order.
//
// Возвращает true если условие выполняется.
// При неизвестном поле или операторе — false (fail secure).
func (e *RuleEngine) evaluateCondition(wo *models.WorkOrder, cond Condition) bool {
	// Получаем значение поля из Work Order
	fieldValue := e.getFieldValue(wo, cond.Field)

	switch cond.Operator {
	case OpEquals:
		return fieldValue == cond.Value
	case OpNotEquals:
		return fieldValue != cond.Value
	case OpContains:
		return strings.Contains(fieldValue, cond.Value)
	case OpGreater:
		return compareNumeric(fieldValue, cond.Value) > 0
	case OpLess:
		return compareNumeric(fieldValue, cond.Value) < 0
	case OpIn:
		values := strings.Split(cond.Value, ",")
		for _, v := range values {
			if strings.TrimSpace(v) == fieldValue {
				return true
			}
		}
		return false
	case OpNotEmpty:
		return fieldValue != ""
	case OpIsEmpty:
		return fieldValue == ""
	default:
		e.logger.Warn("unknown operator in rule condition",
			"operator", cond.Operator,
			"field", cond.Field,
		)
		return false
	}
}

// getFieldValue извлекает значение поля из Work Order.
//
// Поддерживаемые поля:
//   - priority, status, type, assigned_to, sla_status, device_id
//   - title, notes, id
//   - age_minutes — вычисляемое поле (возраст WO в минутах)
func (e *RuleEngine) getFieldValue(wo *models.WorkOrder, field string) string {
	switch field {
	case "id":
		return wo.ID
	case "priority":
		return wo.Priority
	case "status":
		return wo.Status
	case "type":
		return wo.Type
	case "assigned_to":
		if wo.AssignedTo != nil {
			return *wo.AssignedTo
		}
		return ""
	case "sla_status":
		return wo.SLAStatus
	case "device_id":
		return wo.DeviceID
	case "title":
		return wo.Title
	case "notes":
		return wo.Notes
	case "age_minutes":
		age := time.Since(wo.CreatedAt).Minutes()
		return fmt.Sprintf("%.0f", age)
	default:
		e.logger.Warn("unknown field in rule condition",
			"field", field,
		)
		return ""
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Result Types
// ═══════════════════════════════════════════════════════════════════════

// RuleActionResult — результат срабатывания правила.
type RuleActionResult struct {
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	Action    Action    `json:"action"`
	MatchedAt time.Time `json:"matched_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// sortRules сортирует правила по приоритету (меньше = выше).
func (e *RuleEngine) sortRules() {
	for i := 0; i < len(e.rules); i++ {
		for j := i + 1; j < len(e.rules); j++ {
			if e.rules[j].Priority < e.rules[i].Priority {
				e.rules[i], e.rules[j] = e.rules[j], e.rules[i]
			}
		}
	}
}

// compareNumeric сравнивает два числовых строковых значения.
// Возвращает: -1 (a < b), 0 (a == b), 1 (a > b).
func compareNumeric(a, b string) int {
	var aVal, bVal float64
	if _, err := fmt.Sscanf(a, "%f", &aVal); err != nil {
		return 0
	}
	if _, err := fmt.Sscanf(b, "%f", &bVal); err != nil {
		return 0
	}
	if aVal < bVal {
		return -1
	}
	if aVal > bVal {
		return 1
	}
	return 0
}
