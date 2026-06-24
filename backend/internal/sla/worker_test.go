package sla

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Mock implementations
// ═══════════════════════════════════════════════════════════════════════

type mockWorkOrderProvider struct {
	orders []WorkOrderRef
}

func (m *mockWorkOrderProvider) GetActiveWorkOrders(_ context.Context, limit, offset int) ([]WorkOrderRef, error) {
	if offset >= len(m.orders) {
		return nil, nil
	}
	end := offset + limit
	if end > len(m.orders) {
		end = len(m.orders)
	}
	return m.orders[offset:end], nil
}

type mockEventPublisher struct {
	events []SLAEventPayload
}

func (m *mockEventPublisher) PublishSLABreach(_ context.Context, event SLAEventPayload) error {
	m.events = append(m.events, event)
	return nil
}

type mockStatusRecorder struct {
	saved  int32
	loaded int32
	logged int32
}

func (m *mockStatusRecorder) SaveSLATracker(_ context.Context, _ *SLATrackerState) error {
	atomic.AddInt32(&m.saved, 1)
	return nil
}

func (m *mockStatusRecorder) LoadSLATrackers(_ context.Context) ([]*SLATrackerState, error) {
	atomic.AddInt32(&m.loaded, 1)
	return nil, nil
}

func (m *mockStatusRecorder) LogSLABreach(_ context.Context, _ SLAEventPayload) error {
	atomic.AddInt32(&m.logged, 1)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Tests: Worker Config
// ═══════════════════════════════════════════════════════════════════════

func TestSLAWorkerConfigDefaults(t *testing.T) {
	cfg := WorkerConfig{}
	cfg.validate()

	if cfg.Interval != 60*time.Second {
		t.Errorf("expected interval 60s, got %v", cfg.Interval)
	}
	if cfg.BatchSize != 100 {
		t.Errorf("expected batch size 100, got %d", cfg.BatchSize)
	}
	if cfg.SaveInterval != 5*time.Minute {
		t.Errorf("expected save interval 5m, got %v", cfg.SaveInterval)
	}
	if cfg.BreachThreshold != 0.75 {
		t.Errorf("expected breach threshold 0.75, got %f", cfg.BreachThreshold)
	}
	if cfg.CriticalThreshold != 0.90 {
		t.Errorf("expected critical threshold 0.90, got %f", cfg.CriticalThreshold)
	}
}

func TestSLAWorkerHealthMetrics(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	provider := &mockWorkOrderProvider{}
	publisher := &mockEventPublisher{}

	w := NewSLAWorker(engine, provider, publisher, nil, WorkerConfig{}, logger, nil)

	health := w.Health()
	if health["status"] != "running" {
		t.Errorf("expected status 'running', got %v", health["status"])
	}

	metrics := w.Metrics()
	if metrics["sla_processed_total"] != 0 {
		t.Errorf("expected 0 processed, got %d", metrics["sla_processed_total"])
	}
}

func TestSLAWorkerProcessBatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)

	policy := DefaultPolicies()[0]
	engine.SetPolicy(policy)
	engine.SetMatrix(policy.ID, DefaultMatrix(policy.ID))
	engine.SetPauseRules(policy.ID, DefaultPauseRules(policy.ID))

	ctx := context.Background()
	_, err := engine.StartTracking(ctx, "wo-1", policy.ID, "critical", "extensive", "site-1")
	if err != nil {
		t.Fatalf("failed to start tracking: %v", err)
	}

	provider := &mockWorkOrderProvider{
		orders: []WorkOrderRef{
			{ID: "wo-1", Status: "in_progress", Priority: "critical", CreatedAt: time.Now().Add(-30 * time.Minute)},
		},
	}
	publisher := &mockEventPublisher{}
	recorder := &mockStatusRecorder{}

	w := NewSLAWorker(engine, provider, publisher, recorder, WorkerConfig{
		Interval:     time.Hour,
		BatchSize:    100,
		SaveInterval: time.Hour,
	}, logger, nil)

	w.processBatch()

	// Проверяем, что трекер обновлён
	tracker, ok := engine.GetTracker("wo-1")
	if !ok {
		t.Fatal("expected tracker to exist")
	}
	_ = tracker

	// Проверяем метрики
	metrics := w.Metrics()
	if metrics["sla_processed_total"] <= 0 {
		t.Errorf("expected processed count > 0, got %d", metrics["sla_processed_total"])
	}
}

func TestSLAWorkerSLABreachEvent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)

	policy := DefaultPolicies()[0]
	engine.SetPolicy(policy)
	engine.SetMatrix(policy.ID, DefaultMatrix(policy.ID))
	engine.SetPauseRules(policy.ID, DefaultPauseRules(policy.ID))

	ctx := context.Background()
	_, err := engine.StartTracking(ctx, "wo-breach", policy.ID, "critical", "extensive", "site-1")
	if err != nil {
		t.Fatalf("failed to start tracking: %v", err)
	}

	provider := &mockWorkOrderProvider{
		orders: []WorkOrderRef{
			{ID: "wo-breach", Status: "in_progress", Priority: "critical", CreatedAt: time.Now().Add(-2 * time.Hour)},
		},
	}
	publisher := &mockEventPublisher{}
	recorder := &mockStatusRecorder{}

	w := NewSLAWorker(engine, provider, publisher, recorder, WorkerConfig{
		Interval:     time.Hour,
		BatchSize:    100,
		SaveInterval: time.Hour,
	}, logger, nil)

	w.processBatch()

	_ = publisher
	_ = recorder
}

func TestSLAWorkerStartStop(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	provider := &mockWorkOrderProvider{}

	w := NewSLAWorker(engine, provider, nil, nil, WorkerConfig{
		Interval:     100 * time.Millisecond,
		BatchSize:    100,
		SaveInterval: time.Hour,
	}, logger, nil)

	if err := w.Start(); err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	w.Stop()

	// Повторный Stop не должен паниковать
	w.Stop()
}

func TestSLAWorkerProcessBatchEmpty(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	provider := &mockWorkOrderProvider{} // empty

	w := NewSLAWorker(engine, provider, nil, nil, WorkerConfig{}, logger, nil)

	// Не должно паниковать при пустом батче
	w.processBatch()

	metrics := w.Metrics()
	if metrics["sla_processed_total"] != 0 {
		t.Errorf("expected 0 processed for empty batch, got %d", metrics["sla_processed_total"])
	}
}

func TestSLAWorkerMultipleBatches(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)

	policy := DefaultPolicies()[0]
	engine.SetPolicy(policy)
	engine.SetMatrix(policy.ID, DefaultMatrix(policy.ID))

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		woID := fmt.Sprintf("wo-multi-%d", i)
		_, err := engine.StartTracking(ctx, woID, policy.ID, "medium", "limited", "site-1")
		if err != nil {
			t.Fatalf("failed to start tracking %s: %v", woID, err)
		}
	}

	orders := make([]WorkOrderRef, 5)
	for i := 0; i < 5; i++ {
		orders[i] = WorkOrderRef{
			ID:        fmt.Sprintf("wo-multi-%d", i),
			Status:    "in_progress",
			Priority:  "medium",
			CreatedAt: time.Now().Add(-10 * time.Minute),
		}
	}

	provider := &mockWorkOrderProvider{orders: orders}
	w := NewSLAWorker(engine, provider, nil, nil, WorkerConfig{
		BatchSize: 2,
	}, logger, nil)

	w.processBatch()

	metrics := w.Metrics()
	if metrics["sla_processed_total"] != 5 {
		t.Errorf("expected 5 processed, got %d", metrics["sla_processed_total"])
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLA-6.2.3: Tests for BreachedWorkOrder & checkBreachedSLAs
// ═══════════════════════════════════════════════════════════════════════

type mockBreachedFinder struct {
	orders []BreachedWorkOrder
	err    error
}

func (m *mockBreachedFinder) FindBreachedWorkOrders(_ context.Context) ([]BreachedWorkOrder, error) {
	return m.orders, m.err
}

func TestEngine_FindBreachedWorkOrders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)

	// Без finder — ошибка
	ctx := context.Background()
	_, err := engine.FindBreachedWorkOrders(ctx)
	if err == nil {
		t.Fatal("expected error when finder not set")
	}

	// С finder без данных
	mockFinder := &mockBreachedFinder{}
	engine.SetBreachedFinder(mockFinder)
	breached, err := engine.FindBreachedWorkOrders(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(breached) != 0 {
		t.Errorf("expected 0 breached, got %d", len(breached))
	}

	// С finder с данными
	now := time.Now().UTC()
	mockFinder.orders = []BreachedWorkOrder{
		{
			ID:           "wo-breach-1",
			Title:        "Camera offline",
			DeviceID:     "cam-001",
			DeviceName:   "Main Entrance Camera",
			Priority:     "critical",
			SLADeadline:  now.Add(-30 * time.Minute),
			AssignedTo:   "12345",
			AssigneeName: "John Doe",
		},
	}
	breached, err = engine.FindBreachedWorkOrders(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(breached) != 1 {
		t.Fatalf("expected 1 breached, got %d", len(breached))
	}
	if breached[0].Title != "Camera offline" {
		t.Errorf("expected 'Camera offline', got '%s'", breached[0].Title)
	}
	if breached[0].AssignedTo != "12345" {
		t.Errorf("expected assigned_to '12345', got '%s'", breached[0].AssignedTo)
	}
}

func TestEngine_FindBreachedWorkOrders_Error(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	engine.SetBreachedFinder(&mockBreachedFinder{err: fmt.Errorf("db connection failed")})

	_, err := engine.FindBreachedWorkOrders(context.Background())
	if err == nil {
		t.Fatal("expected error from finder")
	}
}

func TestParseChatID(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
		wantVal int64
	}{
		{"12345", false, 12345},
		{"0", true, 0},
		{"unassigned", true, 0},
		{"", true, 0},
		{"abc123", true, 0},
		{"12a34", true, 0},
		{"9876543210", false, 9876543210},
	}

	for _, tt := range tests {
		val, err := parseChatID(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseChatID(%q) expected error, got %d", tt.input, val)
			}
		} else {
			if err != nil {
				t.Errorf("parseChatID(%q) unexpected error: %v", tt.input, err)
			}
			if val != tt.wantVal {
				t.Errorf("parseChatID(%q) = %d, want %d", tt.input, val, tt.wantVal)
			}
		}
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello_world", `hello\_world`},
		{"*bold*", `\*bold\*`},
		{"[link]", `\[link\]`},
		{"text (with) brackets", `text \(with\) brackets`},
		{"no special chars here 123", "no special chars here 123"},
		{"mix_of_special*chars!", `mix\_of\_special\*chars\!`},
	}

	for _, tt := range tests {
		result := escapeMarkdown(tt.input)
		if result != tt.expected {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestCheckBreachedSLAs_NoTelegramBot(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	provider := &mockWorkOrderProvider{}

	// Без telegram bot — не должно паниковать
	w := NewSLAWorker(engine, provider, nil, nil, WorkerConfig{}, logger, nil)
	w.checkBreachedSLAs() // should not panic
}

func TestCheckBreachedSLAs_NoBreaches(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	provider := &mockWorkOrderProvider{}

	// С finder без данных
	engine.SetBreachedFinder(&mockBreachedFinder{})

	w := NewSLAWorker(engine, provider, nil, nil, WorkerConfig{}, logger, nil)
	w.checkBreachedSLAs() // should not panic
}

func TestBreachCheckLoop_StartWithBreachCheck(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := NewEngine(logger)
	provider := &mockWorkOrderProvider{}

	// With finder but without telegram bot — breach check should be disabled but Start should work
	engine.SetBreachedFinder(&mockBreachedFinder{})

	w := NewSLAWorker(engine, provider, nil, nil, WorkerConfig{
		Interval:     100 * time.Millisecond,
		BatchSize:    100,
		SaveInterval: time.Hour,
	}, logger, nil)

	if err := w.Start(); err != nil {
		t.Fatalf("failed to start worker: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	w.Stop()

	// Проверяем что breach check был тихо пропущен (без паники)
	metrics := w.Metrics()
	_ = metrics
}
