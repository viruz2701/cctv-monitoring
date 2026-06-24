// Package workflow — Workflow Execution Engine (WF-9.1.4).
//
// NATS-driven execution engine для автоматизаций.
//
// Алгоритм:
//  1. Получение события (NATS / webhook / cron)
//  2. Поиск Workflow с подходящим Trigger
//  3. Evaluation условий (EvaluateAll)
//  4. Выполнение действий (CREATE_WO, NOTIFY, etc.)
//  5. Логирование результата
package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// WF-9.1.4: Execution Engine
// ═══════════════════════════════════════════════════════════════════════

// EngineConfig — конфигурация Workflow Engine.
type EngineConfig struct {
	MaxConcurrent int // макс. параллельных выполнений
	Logger        *slog.Logger
}

// ExecutionEngine — главный движок выполнения workflow.
type ExecutionEngine struct {
	mu        sync.RWMutex
	logger    *slog.Logger
	workflows []*Workflow

	// Execution log
	executions []ExecutionResult
	maxLogSize int

	// Semaphore для max concurrent
	sem chan struct{}
}

// ExecutionResult — результат выполнения workflow.
type ExecutionResult struct {
	WorkflowID  string         `json:"workflow_id"`
	WorkflowName string        `json:"workflow_name"`
	TriggeredBy string         `json:"triggered_by"`
	ConditionsMet bool         `json:"conditions_met"`
	ActionsExecuted int        `json:"actions_executed"`
	Duration    time.Duration  `json:"duration"`
	Error       string         `json:"error,omitempty"`
	ExecutedAt  time.Time      `json:"executed_at"`
}

// NewEngine создаёт Workflow Execution Engine.
func NewEngine(cfg EngineConfig) *ExecutionEngine {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 10
	}

	return &ExecutionEngine{
		logger:    cfg.Logger.With("component", "workflow-engine"),
		workflows: make([]*Workflow, 0),
		executions: make([]ExecutionResult, 0),
		maxLogSize: 1000,
		sem:        make(chan struct{}, cfg.MaxConcurrent),
	}
}

// SetWorkflows устанавливает список активных workflow.
func (e *ExecutionEngine) SetWorkflows(workflows []*Workflow) {
	e.mu.Lock()
	defer e.mu.Unlock()

	active := make([]*Workflow, 0)
	for _, wf := range workflows {
		if wf.Status == WFActive {
			active = append(active, wf)
		}
	}
	e.workflows = active
	e.logger.Info("workflows loaded", "total", len(workflows), "active", len(active))
}

// HandleEvent обрабатывает входящее событие.
//
// 1. Находит Workflow с подходящим Trigger
// 2. Проверяет условия
// 3. Выполняет действия
func (e *ExecutionEngine) HandleEvent(ctx context.Context, eventSource, eventType string, eventData map[string]interface{}) []ExecutionResult {
	e.mu.RLock()
	workflows := make([]*Workflow, len(e.workflows))
	copy(workflows, e.workflows)
	e.mu.RUnlock()

	results := make([]ExecutionResult, 0)

	for _, wf := range workflows {
		// Проверка триггера
		if !e.matchesTrigger(wf, eventSource, eventType) {
			continue
		}

		// Захватываем семафор
		e.sem <- struct{}{}

		result := e.execute(wf, eventSource, eventData)
		results = append(results, result)

		<-e.sem
	}

	return results
}

// matchesTrigger проверяет, подходит ли триггер workflow под событие.
func (e *ExecutionEngine) matchesTrigger(wf *Workflow, source, eventType string) bool {
	switch wf.Trigger.Type {
	case TriggerEvent:
		return wf.Trigger.Config.EventSource == source &&
			wf.Trigger.Config.EventType == eventType
	case TriggerManual:
		return true
	default:
		return false
	}
}

// execute выполняет один workflow.
func (e *ExecutionEngine) execute(wf *Workflow, source string, eventData map[string]interface{}) ExecutionResult {
	start := time.Now()
	result := ExecutionResult{
		WorkflowID:   wf.ID,
		WorkflowName: wf.Name,
		TriggeredBy:  source,
		ExecutedAt:   start,
	}

	e.logger.Info("executing workflow",
		"workflow", wf.Name,
		"trigger", source,
		"conditions", len(wf.Conditions),
	)

	// Строим контекст
	ctx := EvalContext{
		"event": eventData,
	}

	// Проверка условий
	if len(wf.Conditions) > 0 {
		ok, err := EvaluateAll(wf.Conditions, ctx)
		if err != nil {
			result.Error = fmt.Sprintf("condition eval error: %v", err)
			result.Duration = time.Since(start)
			e.logExecution(result)
			return result
		}
		if !ok {
			result.ConditionsMet = false
			result.Duration = time.Since(start)
			e.logger.Debug("conditions not met", "workflow", wf.Name)
			e.logExecution(result)
			return result
		}
	}

	result.ConditionsMet = true

	// Выполнение действий
	for i, action := range wf.Actions {
		if err := e.executeAction(action, ctx); err != nil {
			e.logger.Error("action failed",
				"workflow", wf.Name,
				"action_index", i,
				"action_type", action.Type,
				"error", err,
			)
			result.Error = fmt.Sprintf("action %d (%s): %v", i, action.Type, err)
		} else {
			result.ActionsExecuted++
		}
	}

	result.Duration = time.Since(start)
	e.logExecution(result)

	e.logger.Info("workflow completed",
		"workflow", wf.Name,
		"actions", result.ActionsExecuted,
		"duration", result.Duration,
	)

	return result
}

// executeAction выполняет одно действие workflow.
func (e *ExecutionEngine) executeAction(action WorkflowAction, ctx EvalContext) error {
	switch action.Type {
	case ActionCreateWO:
		return e.actionCreateWO(action, ctx)
	case ActionNotify:
		return e.actionNotify(action, ctx)
	case ActionUpdateStatus:
		return e.actionUpdateStatus(action, ctx)
	case ActionWebhook:
		return e.actionWebhook(action, ctx)
	case ActionAssign:
		return e.actionAssign(action, ctx)
	case ActionEscalate:
		return e.actionEscalate(action, ctx)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

func (e *ExecutionEngine) actionCreateWO(action WorkflowAction, ctx EvalContext) error {
	title := FillTemplate(action.Params.TitleTemplate, ctx)
	desc := FillTemplate(action.Params.DescTemplate, ctx)

	e.logger.Info("action: CREATE_WO",
		"title", title,
		"type", action.Params.WorkOrderType,
		"priority", action.Params.Priority,
	)

	// Здесь будет вызов CMMSAdapter.CreateWorkOrder
	_ = desc
	return nil
}

func (e *ExecutionEngine) actionNotify(action WorkflowAction, ctx EvalContext) error {
	msg := FillTemplate(action.Params.MessageTemplate, ctx)

	e.logger.Info("action: NOTIFY",
		"channel", action.Params.Channel,
		"recipients", action.Params.Recipients,
		"message", msg,
	)

	// Здесь будет вызов Telegram/Email/SMS
	return nil
}

func (e *ExecutionEngine) actionUpdateStatus(action WorkflowAction, ctx EvalContext) error {
	e.logger.Info("action: UPDATE_STATUS",
		"target_status", action.Params.TargetStatus,
	)
	return nil
}

func (e *ExecutionEngine) actionWebhook(action WorkflowAction, ctx EvalContext) error {
	body := FillTemplate(action.Params.WebhookBody, ctx)

	e.logger.Info("action: WEBHOOK",
		"url", action.Params.WebhookURL,
		"method", action.Params.WebhookMethod,
		"body", body,
	)

	// Здесь будет HTTP вызов
	return nil
}

func (e *ExecutionEngine) actionAssign(action WorkflowAction, ctx EvalContext) error {
	e.logger.Info("action: ASSIGN",
		"assignee", action.Params.AssigneeID,
	)
	return nil
}

func (e *ExecutionEngine) actionEscalate(action WorkflowAction, ctx EvalContext) error {
	e.logger.Info("action: ESCALATE",
		"to", action.Params.EscalateTo,
		"after_minutes", action.Params.EscalateAfter,
	)
	return nil
}

// logExecution логирует результат выполнения.
func (e *ExecutionEngine) logExecution(result ExecutionResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.executions = append(e.executions, result)
	if len(e.executions) > e.maxLogSize {
		e.executions = e.executions[len(e.executions)-e.maxLogSize:]
	}
}

// GetExecutionLog возвращает лог выполненных workflow.
func (e *ExecutionEngine) GetExecutionLog() []ExecutionResult {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]ExecutionResult, len(e.executions))
	copy(result, e.executions)
	return result
}
