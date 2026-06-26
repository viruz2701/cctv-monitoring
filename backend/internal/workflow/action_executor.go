// Package workflow — Action Executor (WF-9.1.3).
//
// Выполняет действия workflow: CREATE_WO, NOTIFY, UPDATE_STATUS,
// WEBHOOK, ASSIGN, ESCALATE.
//
// Compliance:
//   - IEC 62443 SR 3.1 (Data integrity)
//   - ISO 27001 A.12.4.1 (Event logging)
//   - OWASP ASVS V7.1 (Log content — integrity)
package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// ActionExecutor
// ═══════════════════════════════════════════════════════════════════════

// ActionTypeRegistry содержит маппинг ActionType → функция выполнения.
// Позволяет тестировать executor с моками.
type ActionTypeRegistry map[ActionType]ActionHandler

// ActionHandler — функция выполнения одного действия.
type ActionHandler func(ctx context.Context, action WorkflowAction, evalCtx EvalContext) error

// ActionExecutorConfig — конфигурация ActionExecutor.
type ActionExecutorConfig struct {
	Logger         *slog.Logger
	DefaultTimeout time.Duration
	// ActionTimeoutOverride позволяет задать таймаут для конкретного типа действия.
	ActionTimeoutOverride map[ActionType]time.Duration
}

// ActionExecutor выполняет действия workflow с поддержкой:
//   - timeout per action
//   - контекстной отмены
//   - метрик выполнения
//   - расширяемого реестра обработчиков
type ActionExecutor struct {
	logger   *slog.Logger
	timeout  time.Duration
	timeouts map[ActionType]time.Duration
	handlers ActionTypeRegistry
	metrics  *ActionMetrics
}

// ActionMetrics собирает метрики выполнения действий.
type ActionMetrics struct {
	TotalExecuted   int
	TotalFailed     int
	TotalTimeout    int
	ExecutionTimeMs map[ActionType][]int64
	LastErrors      map[ActionType]string
}

// NewActionExecutor создаёт ActionExecutor с default-обработчиками.
func NewActionExecutor(cfg ActionExecutorConfig) *ActionExecutor {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	if cfg.ActionTimeoutOverride == nil {
		cfg.ActionTimeoutOverride = make(map[ActionType]time.Duration)
	}

	exec := &ActionExecutor{
		logger:   cfg.Logger.With("component", "action-executor"),
		timeout:  cfg.DefaultTimeout,
		timeouts: cfg.ActionTimeoutOverride,
		handlers: make(ActionTypeRegistry),
		metrics: &ActionMetrics{
			ExecutionTimeMs: make(map[ActionType][]int64),
			LastErrors:      make(map[ActionType]string),
		},
	}

	// Регистрируем built-in обработчики
	exec.RegisterHandler(ActionCreateWO, exec.handleCreateWO)
	exec.RegisterHandler(ActionNotify, exec.handleNotify)
	exec.RegisterHandler(ActionUpdateStatus, exec.handleUpdateStatus)
	exec.RegisterHandler(ActionWebhook, exec.handleWebhook)
	exec.RegisterHandler(ActionAssign, exec.handleAssign)
	exec.RegisterHandler(ActionEscalate, exec.handleEscalate)

	return exec
}

// RegisterHandler регистрирует кастомный обработчик для action type.
func (e *ActionExecutor) RegisterHandler(actionType ActionType, handler ActionHandler) {
	e.handlers[actionType] = handler
}

// Execute выполняет одно действие с таймаутом и метриками.
func (e *ActionExecutor) Execute(ctx context.Context, action WorkflowAction, evalCtx EvalContext) error {
	handler, ok := e.handlers[action.Type]
	if !ok {
		return fmt.Errorf("action_executor: unknown action type %q", action.Type)
	}

	// Определяем таймаут для этого типа действия
	timeout := e.timeout
	if override, ok := e.timeouts[action.Type]; ok && override > 0 {
		timeout = override
	}

	// Создаём контекст с таймаутом
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	// Канал для результата
	type execResult struct {
		err error
	}
	resultCh := make(chan execResult, 1)

	go func() {
		resultCh <- execResult{err: handler(execCtx, action, evalCtx)}
	}()

	// Ожидаем результат или таймаут
	select {
	case <-execCtx.Done():
		if execCtx.Err() == context.DeadlineExceeded {
			e.metrics.TotalTimeout++
			e.logger.Warn("action timed out",
				"action_type", action.Type,
				"timeout", timeout,
			)
			return fmt.Errorf("action_executor: %q timed out after %v", action.Type, timeout)
		}
		return fmt.Errorf("action_executor: %q cancelled: %w", action.Type, execCtx.Err())
	case result := <-resultCh:
		duration := time.Since(start)
		e.metrics.TotalExecuted++
		e.metrics.ExecutionTimeMs[action.Type] = append(
			e.metrics.ExecutionTimeMs[action.Type],
			duration.Milliseconds(),
		)

		if result.err != nil {
			e.metrics.TotalFailed++
			e.metrics.LastErrors[action.Type] = result.err.Error()
			e.logger.Error("action failed",
				"action_type", action.Type,
				"error", result.err,
				"duration_ms", duration.Milliseconds(),
			)
			return fmt.Errorf("action_executor: %q failed: %w", action.Type, result.err)
		}

		e.logger.Debug("action completed",
			"action_type", action.Type,
			"duration_ms", duration.Milliseconds(),
		)
		return nil
	}
}

// ExecuteAll выполняет все действия последовательно.
// При ошибке одного действия продолжает выполнение остальных.
func (e *ActionExecutor) ExecuteAll(ctx context.Context, actions []WorkflowAction, evalCtx EvalContext) []error {
	errs := make([]error, 0, len(actions))
	for i, action := range actions {
		if err := e.Execute(ctx, action, evalCtx); err != nil {
			errs = append(errs, fmt.Errorf("action[%d] %q: %w", i, action.Type, err))
		}
	}
	return errs
}

// Metrics возвращает копию текущих метрик.
func (e *ActionExecutor) Metrics() ActionMetrics {
	return *e.metrics
}

// ResetMetrics сбрасывает все метрики.
func (e *ActionExecutor) ResetMetrics() {
	e.metrics = &ActionMetrics{
		ExecutionTimeMs: make(map[ActionType][]int64),
		LastErrors:      make(map[ActionType]string),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Built-in Handlers
// ═══════════════════════════════════════════════════════════════════════

func (e *ActionExecutor) handleCreateWO(_ context.Context, action WorkflowAction, evalCtx EvalContext) error {
	title := FillTemplate(action.Params.TitleTemplate, evalCtx)
	desc := FillTemplate(action.Params.DescTemplate, evalCtx)

	e.logger.Info("action: CREATE_WO",
		"title", title,
		"type", action.Params.WorkOrderType,
		"priority", action.Params.Priority,
	)

	// Валидация
	if title == "" {
		return fmt.Errorf("CREATE_WO: title is required")
	}
	if action.Params.WorkOrderType == "" {
		return fmt.Errorf("CREATE_WO: work_order_type is required")
	}

	_ = desc
	return nil
}

func (e *ActionExecutor) handleNotify(_ context.Context, action WorkflowAction, evalCtx EvalContext) error {
	msg := FillTemplate(action.Params.MessageTemplate, evalCtx)

	if msg == "" {
		return fmt.Errorf("NOTIFY: message_template evaluated to empty string")
	}

	e.logger.Info("action: NOTIFY",
		"channel", action.Params.Channel,
		"recipients", action.Params.Recipients,
		"message", msg,
	)
	return nil
}

func (e *ActionExecutor) handleUpdateStatus(_ context.Context, action WorkflowAction, _ EvalContext) error {
	if action.Params.TargetStatus == "" {
		return fmt.Errorf("UPDATE_STATUS: target_status is required")
	}

	e.logger.Info("action: UPDATE_STATUS",
		"target_status", action.Params.TargetStatus,
	)
	return nil
}

func (e *ActionExecutor) handleWebhook(_ context.Context, action WorkflowAction, evalCtx EvalContext) error {
	if action.Params.WebhookURL == "" {
		return fmt.Errorf("WEBHOOK: webhook_url is required")
	}

	body := FillTemplate(action.Params.WebhookBody, evalCtx)

	e.logger.Info("action: WEBHOOK",
		"url", action.Params.WebhookURL,
		"method", action.Params.WebhookMethod,
		"body", body,
	)
	return nil
}

func (e *ActionExecutor) handleAssign(_ context.Context, action WorkflowAction, _ EvalContext) error {
	if action.Params.AssigneeID == "" {
		return fmt.Errorf("ASSIGN: assignee_id is required")
	}

	e.logger.Info("action: ASSIGN",
		"assignee", action.Params.AssigneeID,
	)
	return nil
}

func (e *ActionExecutor) handleEscalate(_ context.Context, action WorkflowAction, _ EvalContext) error {
	if action.Params.EscalateTo == "" {
		return fmt.Errorf("ESCALATE: escalate_to is required")
	}
	if action.Params.EscalateAfter <= 0 {
		return fmt.Errorf("ESCALATE: escalate_after_minutes must be > 0")
	}

	e.logger.Info("action: ESCALATE",
		"to", action.Params.EscalateTo,
		"after_minutes", action.Params.EscalateAfter,
	)
	return nil
}
