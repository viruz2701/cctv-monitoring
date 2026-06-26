// Package workflow — ActionExecutor unit tests
package workflow

import (
	"context"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Action Executor — Constructor & Defaults
// ═══════════════════════════════════════════════════════════════════════

func TestNewActionExecutor_Defaults(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})

	if exec.timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", exec.timeout)
	}
	if len(exec.handlers) != 6 {
		t.Errorf("expected 6 built-in handlers, got %d", len(exec.handlers))
	}
}

func TestNewActionExecutor_CustomTimeout(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{
		DefaultTimeout: 10 * time.Second,
	})

	if exec.timeout != 10*time.Second {
		t.Errorf("expected custom timeout 10s, got %v", exec.timeout)
	}
}

func TestNewActionExecutor_ActionTimeoutOverride(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{
		DefaultTimeout: 30 * time.Second,
		ActionTimeoutOverride: map[ActionType]time.Duration{
			ActionWebhook: 60 * time.Second,
		},
	})

	if exec.timeouts[ActionWebhook] != 60*time.Second {
		t.Errorf("expected webhook timeout 60s, got %v", exec.timeouts[ActionWebhook])
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Action Executor — Table-driven tests for all action types
// ═══════════════════════════════════════════════════════════════════════

func TestActionExecutor_Execute_AllTypes(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx := context.Background()
	evalCtx := EvalContext{
		"event": map[string]interface{}{
			"severity":    "critical",
			"message":     "Motion detected",
			"device_name": "Camera-1",
		},
	}

	tests := []struct {
		name    string
		action  WorkflowAction
		wantErr bool
	}{
		{
			name: "CREATE_WO — valid",
			action: WorkflowAction{
				Type: ActionCreateWO,
				Params: ActionParams{
					WorkOrderType: "emergency",
					Priority:      "critical",
					TitleTemplate: "Alarm: {event.message}",
					DescTemplate:  "Device: {event.device_name}",
				},
			},
			wantErr: false,
		},
		{
			name: "CREATE_WO — missing title",
			action: WorkflowAction{
				Type: ActionCreateWO,
				Params: ActionParams{
					WorkOrderType: "emergency",
				},
			},
			wantErr: true,
		},
		{
			name: "CREATE_WO — missing type",
			action: WorkflowAction{
				Type: ActionCreateWO,
				Params: ActionParams{
					TitleTemplate: "Test",
				},
			},
			wantErr: true,
		},
		{
			name: "NOTIFY — valid",
			action: WorkflowAction{
				Type: ActionNotify,
				Params: ActionParams{
					Channel:         "telegram",
					Recipients:      []string{"admin"},
					MessageTemplate: "Alert: {event.severity}",
				},
			},
			wantErr: false,
		},
		{
			name: "NOTIFY — empty message",
			action: WorkflowAction{
				Type:   ActionNotify,
				Params: ActionParams{},
			},
			wantErr: true,
		},
		{
			name: "UPDATE_STATUS — valid",
			action: WorkflowAction{
				Type: ActionUpdateStatus,
				Params: ActionParams{
					TargetStatus: "in_progress",
				},
			},
			wantErr: false,
		},
		{
			name: "UPDATE_STATUS — missing target",
			action: WorkflowAction{
				Type:   ActionUpdateStatus,
				Params: ActionParams{},
			},
			wantErr: true,
		},
		{
			name: "WEBHOOK — valid",
			action: WorkflowAction{
				Type: ActionWebhook,
				Params: ActionParams{
					WebhookURL:  "https://hooks.example.com/alert",
					WebhookBody: `{"msg": "{event.message}"}`,
				},
			},
			wantErr: false,
		},
		{
			name: "WEBHOOK — missing URL",
			action: WorkflowAction{
				Type:   ActionWebhook,
				Params: ActionParams{},
			},
			wantErr: true,
		},
		{
			name: "ASSIGN — valid",
			action: WorkflowAction{
				Type: ActionAssign,
				Params: ActionParams{
					AssigneeID: "user-42",
				},
			},
			wantErr: false,
		},
		{
			name: "ASSIGN — missing assignee",
			action: WorkflowAction{
				Type:   ActionAssign,
				Params: ActionParams{},
			},
			wantErr: true,
		},
		{
			name: "ESCALATE — valid",
			action: WorkflowAction{
				Type: ActionEscalate,
				Params: ActionParams{
					EscalateTo:    "manager",
					EscalateAfter: 60,
				},
			},
			wantErr: false,
		},
		{
			name: "ESCALATE — missing escalate_to",
			action: WorkflowAction{
				Type: ActionEscalate,
				Params: ActionParams{
					EscalateAfter: 60,
				},
			},
			wantErr: true,
		},
		{
			name: "ESCALATE — zero minutes",
			action: WorkflowAction{
				Type: ActionEscalate,
				Params: ActionParams{
					EscalateTo:    "manager",
					EscalateAfter: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "UNKNOWN — action type",
			action: WorkflowAction{
				Type:   "UNKNOWN_TYPE",
				Params: ActionParams{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := exec.Execute(ctx, tt.action, evalCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// ExecuteAll — error collection
// ═══════════════════════════════════════════════════════════════════════

func TestActionExecutor_ExecuteAll_ContinuesOnError(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx := context.Background()
	evalCtx := EvalContext{}

	actions := []WorkflowAction{
		{Type: ActionNotify, Params: ActionParams{MessageTemplate: "ok"}},
		{Type: ActionNotify, Params: ActionParams{}}, // empty → error
		{Type: ActionUpdateStatus, Params: ActionParams{TargetStatus: "done"}},
	}

	errs := exec.ExecuteAll(ctx, actions, evalCtx)
	if len(errs) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(errs), errs)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Timeout tests
// ═══════════════════════════════════════════════════════════════════════

func TestActionExecutor_Timeout(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{
		DefaultTimeout: 1 * time.Millisecond,
	})
	ctx := context.Background()

	// Регистрируем "медленный" обработчик
	exec.RegisterHandler("SLOW_ACTION", func(_ context.Context, _ WorkflowAction, _ EvalContext) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	err := exec.Execute(ctx, WorkflowAction{Type: "SLOW_ACTION"}, EvalContext{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestActionExecutor_ContextCancellation(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // сразу отменяем

	err := exec.Execute(ctx, WorkflowAction{
		Type:   ActionNotify,
		Params: ActionParams{MessageTemplate: "test"},
	}, EvalContext{})
	if err == nil {
		t.Error("expected cancellation error")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Custom handler registration
// ═══════════════════════════════════════════════════════════════════════

func TestActionExecutor_RegisterHandler(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})

	customCalled := false
	exec.RegisterHandler("CUSTOM", func(_ context.Context, _ WorkflowAction, _ EvalContext) error {
		customCalled = true
		return nil
	})

	err := exec.Execute(context.Background(), WorkflowAction{Type: "CUSTOM"}, EvalContext{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !customCalled {
		t.Error("custom handler was not called")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Metrics tests
// ═══════════════════════════════════════════════════════════════════════

func TestActionExecutor_Metrics(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx := context.Background()
	evalCtx := EvalContext{}

	// Выполняем несколько действий
	_ = exec.Execute(ctx, WorkflowAction{
		Type:   ActionNotify,
		Params: ActionParams{MessageTemplate: "test1"},
	}, evalCtx)

	_ = exec.Execute(ctx, WorkflowAction{
		Type:   ActionUpdateStatus,
		Params: ActionParams{TargetStatus: "done"},
	}, evalCtx)

	// Ошибочное действие
	_ = exec.Execute(ctx, WorkflowAction{
		Type:   ActionNotify,
		Params: ActionParams{},
	}, evalCtx)

	metrics := exec.Metrics()
	if metrics.TotalExecuted != 3 {
		t.Errorf("expected 3 executed, got %d", metrics.TotalExecuted)
	}
	if metrics.TotalFailed != 1 {
		t.Errorf("expected 1 failed, got %d", metrics.TotalFailed)
	}
	if _, ok := metrics.LastErrors[ActionNotify]; !ok {
		t.Error("expected LastErrors for NOTIFY")
	}
}

func TestActionExecutor_ResetMetrics(t *testing.T) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx := context.Background()

	_ = exec.Execute(ctx, WorkflowAction{
		Type:   ActionNotify,
		Params: ActionParams{MessageTemplate: "test"},
	}, EvalContext{})

	exec.ResetMetrics()
	metrics := exec.Metrics()
	if metrics.TotalExecuted != 0 {
		t.Errorf("expected 0 after reset, got %d", metrics.TotalExecuted)
	}
	if len(metrics.LastErrors) != 0 {
		t.Errorf("expected empty LastErrors after reset, got %d", len(metrics.LastErrors))
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Benchmarks
// ═══════════════════════════════════════════════════════════════════════

func BenchmarkActionExecutor_Execute(b *testing.B) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx := context.Background()
	evalCtx := EvalContext{"event": map[string]interface{}{"message": "test"}}

	action := WorkflowAction{
		Type: ActionNotify,
		Params: ActionParams{
			Channel:         "telegram",
			MessageTemplate: "Benchmark: {event.message}",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = exec.Execute(ctx, action, evalCtx)
	}
}

func BenchmarkActionExecutor_ExecuteAll(b *testing.B) {
	exec := NewActionExecutor(ActionExecutorConfig{})
	ctx := context.Background()
	evalCtx := EvalContext{}

	actions := []WorkflowAction{
		{Type: ActionNotify, Params: ActionParams{MessageTemplate: "msg1"}},
		{Type: ActionUpdateStatus, Params: ActionParams{TargetStatus: "done"}},
		{Type: ActionAssign, Params: ActionParams{AssigneeID: "user-1"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = exec.ExecuteAll(ctx, actions, evalCtx)
	}
}
