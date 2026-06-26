// Package rca — tests for GraphBuilder
package rca

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// BuildFromState
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_BuildFromState(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{ID: "site-1", Name: "Main Office", Type: DeviceTypeSite, Status: StatusOnline},
		{ID: "sw-1", Name: "Switch-1", Type: DeviceTypeSwitch, ParentID: "site-1", Status: StatusOnline},
		{ID: "nvr-1", Name: "NVR-1", Type: DeviceTypeNVR, ParentID: "sw-1", Status: StatusOnline},
		{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, ParentID: "nvr-1", Status: StatusOnline},
	}

	added, err := builder.BuildFromState(devices)
	if err != nil {
		t.Fatalf("BuildFromState error: %v", err)
	}
	if added != 4 {
		t.Errorf("expected 4 devices added, got %d", added)
	}
	if graph.NodeCount() != 4 {
		t.Errorf("expected 4 nodes in graph, got %d", graph.NodeCount())
	}

	// Check parent-child relationships
	parent := graph.GetParent("cam-1")
	if parent == nil || parent.ID != "nvr-1" {
		t.Errorf("expected cam-1 parent to be nvr-1, got %v", parent)
	}
}

func TestGraphBuilder_BuildFromState_DuplicateID(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera},
		{ID: "cam-1", Name: "Camera-1 Duplicate", Type: DeviceTypeCamera},
	}

	_, err := builder.BuildFromState(devices)
	if err == nil {
		t.Error("expected validation error for duplicate IDs")
	}
}

func TestGraphBuilder_BuildFromState_MissingParent(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, ParentID: "nonexistent"},
	}

	_, err := builder.BuildFromState(devices)
	if err == nil {
		t.Error("expected validation error for missing parent")
	}
}

func TestGraphBuilder_BuildFromState_Cycle(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	// A → B → C → A creates a cycle
	devices := []DeviceState{
		{ID: "a", Name: "A", Type: DeviceTypeSite, ParentID: "c"},
		{ID: "b", Name: "B", Type: DeviceTypeSwitch, ParentID: "a"},
		{ID: "c", Name: "C", Type: DeviceTypeNVR, ParentID: "b"},
	}

	_, err := builder.BuildFromState(devices)
	if err == nil {
		t.Error("expected validation error for cycle detection")
	}
}

func TestGraphBuilder_BuildFromState_InvalidType(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{ID: "dev-1", Name: "Unknown Device", Type: "invalid_type"},
	}

	_, err := builder.BuildFromState(devices)
	if err == nil {
		t.Error("expected validation error for invalid device type")
	}
}

func TestGraphBuilder_BuildFromState_MissingRequired(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	tests := []struct {
		name    string
		devices []DeviceState
	}{
		{"missing ID", []DeviceState{{Name: "NoID", Type: DeviceTypeCamera}}},
		{"missing Name", []DeviceState{{ID: "no-name", Type: DeviceTypeCamera}}},
		{"missing Type", []DeviceState{{ID: "no-type", Name: "NoType"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := builder.BuildFromState(tt.devices)
			if err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Incremental Updates
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_AddNode(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	err := builder.AddNode(DeviceState{
		ID:   "cam-1",
		Name: "Camera-1",
		Type: DeviceTypeCamera,
	})
	if err != nil {
		t.Fatalf("AddNode error: %v", err)
	}

	node, ok := graph.GetNode("cam-1")
	if !ok {
		t.Fatal("expected cam-1 to exist")
	}
	if node.Name != "Camera-1" {
		t.Errorf("expected Camera-1, got %s", node.Name)
	}
}

func TestGraphBuilder_AddNode_Validation(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	tests := []struct {
		name  string
		state DeviceState
	}{
		{"empty ID", DeviceState{Name: "Test", Type: DeviceTypeCamera}},
		{"empty Name", DeviceState{ID: "test", Type: DeviceTypeCamera}},
		{"empty Type", DeviceState{ID: "test", Name: "Test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := builder.AddNode(tt.state); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestGraphBuilder_AddNode_Update(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	_ = builder.AddNode(DeviceState{
		ID:     "cam-1",
		Name:   "Camera-1",
		Type:   DeviceTypeCamera,
		Status: StatusOnline,
	})

	// Update
	err := builder.AddNode(DeviceState{
		ID:     "cam-1",
		Name:   "Camera-1 Updated",
		Type:   DeviceTypeCamera,
		Status: StatusOffline,
	})
	if err != nil {
		t.Fatalf("AddNode update error: %v", err)
	}

	node, _ := graph.GetNode("cam-1")
	if node.Name != "Camera-1 Updated" {
		t.Errorf("expected updated name, got %s", node.Name)
	}
}

func TestGraphBuilder_RemoveNode(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	_ = builder.AddNode(DeviceState{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera})
	_ = builder.AddNode(DeviceState{ID: "cam-2", Name: "Camera-2", Type: DeviceTypeCamera})

	err := builder.RemoveNode("cam-1", false)
	if err != nil {
		t.Fatalf("RemoveNode error: %v", err)
	}

	if graph.NodeCount() != 1 {
		t.Errorf("expected 1 node after removal, got %d", graph.NodeCount())
	}
}

func TestGraphBuilder_RemoveNode_WithChildren(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	_ = builder.AddNode(DeviceState{ID: "sw-1", Name: "Switch", Type: DeviceTypeSwitch})
	_ = builder.AddNode(DeviceState{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, ParentID: "sw-1"})
	_ = builder.AddNode(DeviceState{ID: "cam-2", Name: "Camera-2", Type: DeviceTypeCamera, ParentID: "sw-1"})

	err := builder.RemoveNode("sw-1", true)
	if err != nil {
		t.Fatalf("RemoveNode error: %v", err)
	}

	if graph.NodeCount() != 0 {
		t.Errorf("expected 0 nodes after recursive removal, got %d", graph.NodeCount())
	}
}

func TestGraphBuilder_RemoveNode_NotFound(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	err := builder.RemoveNode("nonexistent", false)
	if err == nil {
		t.Error("expected error for removing non-existent node")
	}
}

func TestGraphBuilder_UpdateNodeStatus(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	_ = builder.AddNode(DeviceState{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, Status: StatusOnline})

	changed := builder.UpdateNodeStatus("cam-1", StatusOffline)
	if !changed {
		t.Error("expected status change detected")
	}

	node, _ := graph.GetNode("cam-1")
	if node.Status != StatusOffline {
		t.Errorf("expected OFFLINE, got %s", node.Status)
	}

	// Same status — should return false
	changed = builder.UpdateNodeStatus("cam-1", StatusOffline)
	if changed {
		t.Error("expected no change for same status")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// TopologyConfig
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_BuildFromTopology(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	cfg := TopologyConfig{
		Name: "Test Topology",
		Devices: []DeviceState{
			{ID: "site-1", Name: "Site", Type: DeviceTypeSite},
			{ID: "sw-1", Name: "Switch", Type: DeviceTypeSwitch},
			{ID: "cam-1", Name: "Camera", Type: DeviceTypeCamera},
		},
		Links: []TopologyLink{
			{ParentID: "site-1", ChildID: "sw-1", Relation: "uplink"},
			{ParentID: "sw-1", ChildID: "cam-1", Relation: "downlink"},
		},
	}

	added, err := builder.BuildFromTopology(cfg)
	if err != nil {
		t.Fatalf("BuildFromTopology error: %v", err)
	}
	if added != 3 {
		t.Errorf("expected 3 devices, got %d", added)
	}

	parent := graph.GetParent("cam-1")
	if parent == nil || parent.ID != "sw-1" {
		t.Errorf("expected cam-1 parent sw-1, got %v", parent)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Event Listener / Cache Invalidation
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_OnDeviceChange(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	events := make([]GraphChangeEvent, 0)
	unsubscribe := builder.OnDeviceChange(func(event GraphChangeEvent) {
		events = append(events, event)
	})

	_ = builder.AddNode(DeviceState{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != ChangeNodeAdded {
		t.Errorf("expected ChangeNodeAdded, got %s", events[0].Type)
	}
	if events[0].NodeID != "cam-1" {
		t.Errorf("expected NodeID cam-1, got %s", events[0].NodeID)
	}

	// Отписка
	unsubscribe()

	_ = builder.AddNode(DeviceState{ID: "cam-2", Name: "Camera-2", Type: DeviceTypeCamera})
	if len(events) != 1 {
		t.Errorf("expected no new events after unsubscribe, got %d", len(events))
	}
}

func TestGraphBuilder_GraphVersion(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	if builder.GraphVersion() != 0 {
		t.Errorf("expected version 0 for empty graph, got %d", builder.GraphVersion())
	}

	_ = builder.AddNode(DeviceState{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera})

	if builder.GraphVersion() != 1 {
		t.Errorf("expected version 1 after add, got %d", builder.GraphVersion())
	}
}

func TestGraphBuilder_CyclePrevention(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	_ = builder.AddNode(DeviceState{ID: "a", Name: "A", Type: DeviceTypeSite})
	_ = builder.AddNode(DeviceState{ID: "b", Name: "B", Type: DeviceTypeSwitch, ParentID: "a"})
	_ = builder.AddNode(DeviceState{ID: "c", Name: "C", Type: DeviceTypeNVR, ParentID: "b"})

	// Try to make A child of C — would create a cycle
	err := builder.AddNode(DeviceState{ID: "a", Name: "A", Type: DeviceTypeSite, ParentID: "c"})
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BuildFromTopology — edge cases
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_BuildFromTopology_Empty(t *testing.T) {
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	added, err := builder.BuildFromTopology(TopologyConfig{})
	if err != nil {
		t.Fatalf("BuildFromTopology error: %v", err)
	}
	if added != 0 {
		t.Errorf("expected 0 devices, got %d", added)
	}
}
