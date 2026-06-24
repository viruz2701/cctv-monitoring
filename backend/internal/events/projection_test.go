// Package events — tests for Projection Builder
//
// Compliance:
//   - CQRS pattern validation
//   - ISO 27001 A.12.4.1 (Event replay correctness)
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// ProjectionBuilder tests
// ═══════════════════════════════════════════════════════════════════════

func TestProjectionBuilder_RegisterAndList(t *testing.T) {
	pb := NewProjectionBuilder(nil, NewInMemorySnapshotStore(), DefaultProjectionBuilderConfig())

	wp := NewWorkOrderProjection(nil)
	sla := NewSLAProjection(nil)
	tech := NewTechnicianProjection(nil)

	if err := pb.RegisterProjection(wp); err != nil {
		t.Fatalf("RegisterProjection failed: %v", err)
	}
	if err := pb.RegisterProjection(sla); err != nil {
		t.Fatalf("RegisterProjection failed: %v", err)
	}
	if err := pb.RegisterProjection(tech); err != nil {
		t.Fatalf("RegisterProjection failed: %v", err)
	}

	// Duplicate registration
	err := pb.RegisterProjection(wp)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}

	names := pb.ListProjections()
	if len(names) != 3 {
		t.Errorf("expected 3 projections, got %d", len(names))
	}

	// GetProjection
	p, ok := pb.GetProjection("work_order")
	if !ok {
		t.Error("expected to find work_order projection")
	}
	if p.Name() != "work_order" {
		t.Errorf("expected name work_order, got %s", p.Name())
	}
}

func TestProjectionBuilder_HandleEvent(t *testing.T) {
	pb := NewProjectionBuilder(nil, NewInMemorySnapshotStore(), ProjectionBuilderConfig{
		RebuildOnStart: false,
		FlushInterval:  0,
		AutoSubscribe:  false,
		Logger:         slog.Default(),
	})

	wp := NewWorkOrderProjection(nil)
	_ = pb.RegisterProjection(wp)

	// Создаём событие WO creation
	now := time.Now()
	record := &EventRecord{
		ID:        "test-001",
		Source:    SourceCMMS,
		EventType: "cmms.wo.created",
		Timestamp: now,
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001",
			"device_id":     "dev-001",
			"title":         "Test WO",
			"type":          "corrective",
			"priority":      "high",
		}),
	}

	pb.HandleEvent(context.Background(), record)

	// Проверяем что проекция обновилась
	snap, ok := wp.GetSnapshot("wo-001")
	if !ok {
		t.Fatal("expected snapshot for wo-001")
	}
	if snap.Status != "REQUESTED" {
		t.Errorf("expected REQUESTED, got %s", snap.Status)
	}
	if snap.Priority != "high" {
		t.Errorf("expected high priority, got %s", snap.Priority)
	}

	stats := wp.GetStats()
	if stats.TotalWO != 1 {
		t.Errorf("expected 1 total WO, got %d", stats.TotalWO)
	}
	if stats.ActiveCount != 1 {
		t.Errorf("expected 1 active WO, got %d", stats.ActiveCount)
	}
}

func TestProjectionBuilder_StatusTransition(t *testing.T) {
	pb := NewProjectionBuilder(nil, NewInMemorySnapshotStore(), ProjectionBuilderConfig{
		RebuildOnStart: false,
		FlushInterval:  0,
		Logger:         slog.Default(),
	})

	wp := NewWorkOrderProjection(nil)
	_ = pb.RegisterProjection(wp)

	ctx := context.Background()
	now := time.Now()

	// 1. Create WO
	pb.HandleEvent(ctx, &EventRecord{
		ID:        "e1", Source: SourceCMMS, EventType: "cmms.wo.created",
		Timestamp: now, AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "device_id": "dev-001",
			"title": "Test", "type": "corrective", "priority": "critical",
		}),
	})

	// 2. Status change: REQUESTED → IN_PROGRESS
	pb.HandleEvent(ctx, &EventRecord{
		ID: "e2", Source: SourceCMMS, EventType: "cmms.wo.status_changed",
		Timestamp: now.Add(5 * time.Minute), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "from_status": "REQUESTED", "to_status": "IN_PROGRESS",
		}),
	})

	snap, _ := wp.GetSnapshot("wo-001")
	if snap.Status != "IN_PROGRESS" {
		t.Errorf("expected IN_PROGRESS, got %s", snap.Status)
	}

	// 3. Complete
	pb.HandleEvent(ctx, &EventRecord{
		ID: "e3", Source: SourceCMMS, EventType: "cmms.wo.completed",
		Timestamp: now.Add(2 * time.Hour), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "completed_by": "tech-001",
		}),
	})

	snap, _ = wp.GetSnapshot("wo-001")
	if snap.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", snap.Status)
	}
	if snap.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	stats := wp.GetStats()
	if stats.CompletedToday != 1 {
		t.Errorf("expected 1 completed today, got %d", stats.CompletedToday)
	}
	if stats.ActiveCount != 0 {
		t.Errorf("expected 0 active, got %d", stats.ActiveCount)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// WorkOrderProjection tests
// ═══════════════════════════════════════════════════════════════════════

func TestWorkOrderProjection_QueryMethods(t *testing.T) {
	wp := NewWorkOrderProjection(nil)
	ctx := context.Background()
	now := time.Now()

	// Create 3 WOs
	for i := 1; i <= 3; i++ {
		woID := fmt.Sprintf("wo-%03d", i)
		wp.Handle(ctx, &EventRecord{
			ID: "e"+woID, Source: SourceCMMS, EventType: "cmms.wo.created",
			Timestamp: now, AggregateID: woID,
			Data: mustMarshal(map[string]interface{}{
				"work_order_id": woID, "device_id": "dev-001",
				"title": "WO "+woID, "type": "corrective", "priority": "medium",
			}),
		})
	}

	// Complete one
	wp.Handle(ctx, &EventRecord{
		ID: "e-complete", Source: SourceCMMS, EventType: "cmms.wo.completed",
		Timestamp: now.Add(1 * time.Hour), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "completed_by": "tech-001",
		}),
	})

	// GetActive should return 2 (wo-002, wo-003)
	active := wp.GetActive()
	if len(active) != 2 {
		t.Errorf("expected 2 active WO, got %d", len(active))
	}

	// GetByStatus should return 1 COMPLETED
	completed := wp.GetByStatus("COMPLETED")
	if len(completed) != 1 {
		t.Errorf("expected 1 COMPLETED, got %d", len(completed))
	}

	// MTTR
	mttr := wp.GetMTTR()
	if mttr <= 0 {
		t.Errorf("expected positive MTTR, got %f", mttr)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SLAProjection tests
// ═══════════════════════════════════════════════════════════════════════

func TestSLAProjection_Basic(t *testing.T) {
	sp := NewSLAProjection(nil)
	ctx := context.Background()
	now := time.Now()

	// Create critical WO
	sp.Handle(ctx, &EventRecord{
		ID: "e1", Source: SourceCMMS, EventType: "cmms.wo.created",
		Timestamp: now, AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "priority": "critical",
		}),
	})

	// Status change (response)
	sp.Handle(ctx, &EventRecord{
		ID: "e2", Source: SourceCMMS, EventType: "cmms.wo.status_changed",
		Timestamp: now.Add(10 * time.Minute), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "from_status": "REQUESTED", "to_status": "IN_PROGRESS",
		}),
	})

	// Complete within SLA (critical = 60 min resolution)
	sp.Handle(ctx, &EventRecord{
		ID: "e3", Source: SourceCMMS, EventType: "cmms.wo.completed",
		Timestamp: now.Add(30 * time.Minute), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001",
		}),
	})

	entry, ok := sp.GetSLAEntry("wo-001")
	if !ok {
		t.Fatal("expected SLA entry")
	}
	if entry.ResponseBreached {
		t.Error("response should not be breached (10min < 15min target)")
	}
	if entry.ResolutionBreached {
		t.Error("resolution should not be breached (30min < 60min target)")
	}

	stats := sp.GetStats()
	if stats.Overall == nil {
		t.Fatal("expected overall stats")
	}
	if stats.Overall.CompliancePercent != 100 {
		t.Errorf("expected 100%% compliance, got %.1f%%", stats.Overall.CompliancePercent)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TechnicianProjection tests
// ═══════════════════════════════════════════════════════════════════════

func TestTechnicianProjection_Basic(t *testing.T) {
	tp := NewTechnicianProjection(nil)
	ctx := context.Background()
	now := time.Now()

	// Assign WO to technician
	tp.Handle(ctx, &EventRecord{
		ID: "e1", Source: SourceCMMS, EventType: "cmms.wo.assigned",
		Timestamp: now, AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "assignee_id": "tech-001",
		}),
	})

	tech, ok := tp.GetTechnician("tech-001")
	if !ok {
		t.Fatal("expected technician")
	}
	if tech.ActiveWO != 1 {
		t.Errorf("expected 1 active WO, got %d", tech.ActiveWO)
	}

	// Complete WO
	tp.Handle(ctx, &EventRecord{
		ID: "e2", Source: SourceCMMS, EventType: "cmms.wo.completed",
		Timestamp: now.Add(2 * time.Hour), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "completed_by": "tech-001",
		}),
	})

	tech, _ = tp.GetTechnician("tech-001")
	if tech.ActiveWO != 0 {
		t.Errorf("expected 0 active WO after completion, got %d", tech.ActiveWO)
	}
	if tech.TotalCompleted != 1 {
		t.Errorf("expected 1 total completed, got %d", tech.TotalCompleted)
	}

	stats := tp.GetStats()
	if stats.TotalTechnicians != 1 {
		t.Errorf("expected 1 technician, got %d", stats.TotalTechnicians)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Snapshot tests
// ═══════════════════════════════════════════════════════════════════════

func TestInMemorySnapshotStore(t *testing.T) {
	store := NewInMemorySnapshotStore()
	ctx := context.Background()

	// Save
	err := store.SaveSnapshot(ctx, "test-proj", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Load
	data, err := store.LoadSnapshot(ctx, "test-proj")
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	if string(data) != `{"key":"value"}` {
		t.Errorf("expected json, got %s", string(data))
	}

	// Load non-existent
	_, err = store.LoadSnapshot(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent snapshot")
	}

	// Delete
	err = store.DeleteSnapshot(ctx, "test-proj")
	if err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}
	_, err = store.LoadSnapshot(ctx, "test-proj")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestWorkOrderProjection_SnapshotRoundtrip(t *testing.T) {
	wp := NewWorkOrderProjection(nil)
	ctx := context.Background()

	wp.Handle(ctx, &EventRecord{
		ID: "e1", Source: SourceCMMS, EventType: "cmms.wo.created",
		Timestamp: time.Now(), AggregateID: "wo-001",
		Data: mustMarshal(map[string]interface{}{
			"work_order_id": "wo-001", "device_id": "dev-001",
			"title": "Test", "type": "corrective", "priority": "high",
		}),
	})

	// Snapshot
	data, err := wp.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot failed: %v", err)
	}

	// Create new projection and restore
	wp2 := NewWorkOrderProjection(nil)
	if err := wp2.Restore(data); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	snap, ok := wp2.GetSnapshot("wo-001")
	if !ok {
		t.Fatal("expected snapshot after restore")
	}
	if snap.Priority != "high" {
		t.Errorf("expected high priority, got %s", snap.Priority)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func mustMarshal(v interface{}) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}

