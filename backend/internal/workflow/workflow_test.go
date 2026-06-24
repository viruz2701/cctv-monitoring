package workflow

import (
	"context"
	"testing"
)

func TestEvaluate_EQ(t *testing.T) {
	ok, err := Evaluate(WorkflowCondition{Field: "severity", Operator: OpEQ, Value: "critical"}, EvalContext{
		"severity": "critical",
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !ok {
		t.Error("expected critical == critical")
	}
}

func TestEvaluate_GT(t *testing.T) {
	ok, err := Evaluate(WorkflowCondition{Field: "value", Operator: OpGT, Value: float64(50)}, EvalContext{
		"value": 75.0,
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !ok {
		t.Error("expected 75 > 50")
	}

	ok, err = Evaluate(WorkflowCondition{Field: "value", Operator: OpGT, Value: float64(100)}, EvalContext{
		"value": 75.0,
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if ok {
		t.Error("expected 75 NOT > 100")
	}
}

func TestEvaluate_NestedField(t *testing.T) {
	ok, err := Evaluate(WorkflowCondition{Field: "event.severity", Operator: OpEQ, Value: "critical"}, EvalContext{
		"event": map[string]interface{}{
			"severity": "critical",
			"type":     "alarm.created",
		},
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !ok {
		t.Error("expected event.severity == critical")
	}
}

func TestEvaluateAll_AllConditions(t *testing.T) {
	ctx := EvalContext{
		"event": map[string]interface{}{
			"severity": "critical",
			"type":     "alarm.created",
		},
		"device": map[string]interface{}{
			"asset_class": "critical",
		},
	}

	conditions := []WorkflowCondition{
		{Field: "event.severity", Operator: OpEQ, Value: "critical"},
		{Field: "event.type", Operator: OpEQ, Value: "alarm.created"},
		{Field: "device.asset_class", Operator: OpEQ, Value: "critical"},
	}

	ok, err := EvaluateAll(conditions, ctx)
	if err != nil {
		t.Fatalf("EvaluateAll failed: %v", err)
	}
	if !ok {
		t.Error("expected all conditions to pass")
	}

	// One condition fails
	conditions[0].Value = "low"
	ok, err = EvaluateAll(conditions, ctx)
	if err != nil {
		t.Fatalf("EvaluateAll failed: %v", err)
	}
	if ok {
		t.Error("expected conditions to fail")
	}
}

func TestFillTemplate(t *testing.T) {
	ctx := EvalContext{
		"event": map[string]interface{}{
			"message":     "Motion detected",
			"device_name": "Camera-1",
			"severity":    "critical",
		},
	}

	tpl := "🚨 {event.severity}: {event.message} on {event.device_name}"
	result := FillTemplate(tpl, ctx)

	expected := "🚨 critical: Motion detected on Camera-1"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestFillTemplate_UnknownField(t *testing.T) {
	ctx := EvalContext{"event": map[string]interface{}{"message": "test"}}
	result := FillTemplate("{event.message} on {event.unknown}", ctx)
	expected := "test on <unknown>"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

func TestDefaultTemplates(t *testing.T) {
	templates := DefaultTemplates()
	if len(templates) != 3 {
		t.Fatalf("expected 3 default templates, got %d", len(templates))
	}

	// Check critical alarm template
	critical := templates[0]
	if critical.Name != "Critical alarm → Emergency WO" {
		t.Errorf("expected critical alarm template, got %s", critical.Name)
	}
	if len(critical.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(critical.Actions))
	}
	if critical.Actions[0].Type != ActionCreateWO {
		t.Errorf("expected CREATE_WO action, got %s", critical.Actions[0].Type)
	}
}

func TestEngine_HandleEvent(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	engine.SetWorkflows(DefaultTemplates())

	ctx := context.Background()
	eventData := map[string]interface{}{
		"severity":    "critical",
		"message":     "Motion detected at entrance",
		"device_name": "Camera-1",
		"type":        "alarm.created",
	}

	results := engine.HandleEvent(ctx, "alarms", "alarm.created", eventData)

	if len(results) == 0 {
		t.Fatal("expected at least one workflow execution")
	}

	// Critical alarm template should fire
	for _, r := range results {
		if r.WorkflowName == "Critical alarm → Emergency WO" {
			if !r.ConditionsMet {
				t.Error("expected conditions to be met for critical alarm")
			}
			if r.ActionsExecuted == 0 {
				t.Error("expected at least one action executed")
			}
		}
	}
}

func TestEngine_NonMatchingEvent(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	engine.SetWorkflows(DefaultTemplates())

	ctx := context.Background()
	eventData := map[string]interface{}{
		"severity": "low",
		"message":  "Info alert",
		"type":     "alarm.created",
	}

	results := engine.HandleEvent(ctx, "alarms", "alarm.created", eventData)

	// Critical alarm template should NOT fire (severity != critical)
	for _, r := range results {
		if r.WorkflowName == "Critical alarm → Emergency WO" && r.ConditionsMet {
			t.Error("expected conditions NOT to be met for non-critical alarm")
		}
	}
}

func TestEngine_ExecutionLog(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	engine.SetWorkflows(DefaultTemplates())

	ctx := context.Background()
	engine.HandleEvent(ctx, "alarms", "alarm.created", map[string]interface{}{
		"severity": "critical",
		"message":  "Test",
		"type":     "alarm.created",
	})

	log := engine.GetExecutionLog()
	if len(log) == 0 {
		t.Error("expected execution log entries")
	}
}

func TestContains_String(t *testing.T) {
	ok, err := Evaluate(WorkflowCondition{Field: "message", Operator: OpContains, Value: "Motion"}, EvalContext{
		"message": "Motion detected at entrance",
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !ok {
		t.Error("expected 'Motion detected' to contain 'motion'")
	}
}

func TestMatches_Regex(t *testing.T) {
	ok, err := Evaluate(WorkflowCondition{Field: "device_id", Operator: OpMatches, Value: "^cam-.*"}, EvalContext{
		"device_id": "cam-001",
	})
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !ok {
		t.Error("expected cam-001 to match ^cam-.*")
	}
}

func TestResolveField_NotFound(t *testing.T) {
	_, err := resolveField("nonexistent.field", EvalContext{
		"event": map[string]interface{}{"type": "test"},
	})
	if err == nil {
		t.Error("expected error for nonexistent field")
	}
}
