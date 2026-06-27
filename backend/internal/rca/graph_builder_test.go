// Package rca — tests for GraphBuilder
package rca

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
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

// ═══════════════════════════════════════════════════════════════════════
// BACKEND.5: BuildFromState — Accuracy Tests
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_BuildFromState_DeepHierarchy_Accuracy(t *testing.T) {
	// Build 5-level deep hierarchy, verify all relationships
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{ID: "lvl1", Name: "Level-1", Type: DeviceTypeSite, Status: StatusOnline},
		{ID: "lvl2", Name: "Level-2", Type: DeviceTypeSwitch, ParentID: "lvl1", Status: StatusOnline},
		{ID: "lvl3", Name: "Level-3", Type: DeviceTypeNVR, ParentID: "lvl2", Status: StatusOnline},
		{ID: "lvl4", Name: "Level-4", Type: DeviceTypeEncoder, ParentID: "lvl3", Status: StatusDegraded},
		{ID: "lvl5", Name: "Level-5", Type: DeviceTypeCamera, ParentID: "lvl4", Status: StatusOffline},
	}

	added, err := builder.BuildFromState(devices)
	if err != nil {
		t.Fatalf("BuildFromState error: %v", err)
	}
	if added != 5 {
		t.Errorf("expected 5 devices added, got %d", added)
	}
	if graph.NodeCount() != 5 {
		t.Errorf("expected 5 nodes, got %d", graph.NodeCount())
	}

	// Verify root
	if len(graph.roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(graph.roots))
	}
	if graph.roots[0] != "lvl1" {
		t.Errorf("expected root lvl1, got %s", graph.roots[0])
	}

	// Verify chain: lvl5 → lvl4 → lvl3 → lvl2 → lvl1
	expectedChain := []struct {
		id       string
		parentID string
	}{
		{"lvl5", "lvl4"},
		{"lvl4", "lvl3"},
		{"lvl3", "lvl2"},
		{"lvl2", "lvl1"},
		{"lvl1", ""},
	}
	for _, ec := range expectedChain {
		node, ok := graph.GetNode(ec.id)
		if !ok {
			t.Fatalf("node %q not found", ec.id)
		}
		if node.ParentID != ec.parentID {
			t.Errorf("node %q: expected parent %q, got %q", ec.id, ec.parentID, node.ParentID)
		}
	}

	// Verify ancestors
	ancestors := graph.GetAncestors("lvl5")
	if len(ancestors) != 4 {
		t.Errorf("expected 4 ancestors for lvl5, got %d", len(ancestors))
	}

	// Verify descendants from root
	descendants := graph.GetAllDescendants("lvl1")
	if len(descendants) != 4 {
		t.Errorf("expected 4 descendants from root, got %d", len(descendants))
	}

	// Verify GraphVersion incremented
	if builder.GraphVersion() != 1 {
		t.Errorf("expected version 1, got %d", builder.GraphVersion())
	}
}

func TestGraphBuilder_BuildFromState_LargeGraph_Accuracy(t *testing.T) {
	// Build 100+ node graph, verify counts and relationships
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})
	builder.cfg.StrictValidation = true

	const numSites = 3
	const numSwitchesPerSite = 4
	const numNVRsPerSwitch = 2
	const numCamerasPerNVR = 8

	var devices []DeviceState
	totalExpected := 0

	for si := 1; si <= numSites; si++ {
		siteID := fmt.Sprintf("site-%d", si)
		devices = append(devices, DeviceState{
			ID: siteID, Name: siteID, Type: DeviceTypeSite, Status: StatusOnline,
		})
		totalExpected++

		for swi := 1; swi <= numSwitchesPerSite; swi++ {
			swID := fmt.Sprintf("sw-%d-%d", si, swi)
			devices = append(devices, DeviceState{
				ID: swID, Name: swID, Type: DeviceTypeSwitch,
				ParentID: siteID, Status: StatusOnline,
			})
			totalExpected++

			for nvri := 1; nvri <= numNVRsPerSwitch; nvri++ {
				nvrID := fmt.Sprintf("nvr-%d-%d-%d", si, swi, nvri)
				devices = append(devices, DeviceState{
					ID: nvrID, Name: nvrID, Type: DeviceTypeNVR,
					ParentID: swID, Status: StatusOnline,
				})
				totalExpected++

				for ci := 1; ci <= numCamerasPerNVR; ci++ {
					camID := fmt.Sprintf("cam-%d-%d-%d-%d", si, swi, nvri, ci)
					devices = append(devices, DeviceState{
						ID: camID, Name: camID, Type: DeviceTypeCamera,
						ParentID: nvrID, Status: StatusOnline,
					})
					totalExpected++
				}
			}
		}
	}

	added, err := builder.BuildFromState(devices)
	if err != nil {
		t.Fatalf("BuildFromState error: %v", err)
	}
	if added != totalExpected {
		t.Errorf("expected %d devices, got %d", totalExpected, added)
	}
	if graph.NodeCount() != totalExpected {
		t.Errorf("expected %d nodes, got %d", totalExpected, graph.NodeCount())
	}

	// Verify root count
	if len(graph.roots) != numSites {
		t.Errorf("expected %d roots, got %d", numSites, len(graph.roots))
	}

	// Verify camera under specific path
	expectedCamID := "cam-2-3-1-4"
	if _, ok := graph.GetNode(expectedCamID); !ok {
		t.Fatalf("expected camera %q not found", expectedCamID)
	}
	parent := graph.GetParent(expectedCamID)
	if parent == nil || parent.ID != "nvr-2-3-1" {
		t.Errorf("expected parent nvr-2-3-1, got %v", parent)
	}

	// Verify ancestors of deep camera
	ancestors := graph.GetAncestors(expectedCamID)
	if len(ancestors) != 3 { // nvr → switch → site
		t.Errorf("expected 3 ancestors for camera, got %d", len(ancestors))
	}

	// Verify version
	if builder.GraphVersion() != 1 {
		t.Errorf("expected version 1, got %d", builder.GraphVersion())
	}
}

func TestGraphBuilder_BuildFromState_StatusAndMetadata_Preserved(t *testing.T) {
	// Verify Location, SiteID, Status are properly set
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{
			ID: "site-1", Name: "Main Office", Type: DeviceTypeSite,
			Status: StatusOnline, Location: "Minsk, Belarus", SiteID: "site-1",
		},
		{
			ID: "cam-1", Name: "Entrance Camera", Type: DeviceTypeCamera,
			Status: StatusOffline, Location: "Building A, Floor 1", SiteID: "site-1",
			ParentID: "site-1",
		},
		{
			ID: "cam-2", Name: "Parking Camera", Type: DeviceTypeCamera,
			Status: StatusDegraded, Location: "Parking Lot A", SiteID: "site-1",
			ParentID: "site-1",
		},
	}

	added, err := builder.BuildFromState(devices)
	if err != nil {
		t.Fatalf("BuildFromState error: %v", err)
	}
	if added != 3 {
		t.Errorf("expected 3 devices, got %d", added)
	}

	// Verify metadata for each node
	tests := []struct {
		id       string
		status   DeviceStatus
		location string
		siteID   string
	}{
		{"site-1", StatusOnline, "Minsk, Belarus", "site-1"},
		{"cam-1", StatusOffline, "Building A, Floor 1", "site-1"},
		{"cam-2", StatusDegraded, "Parking Lot A", "site-1"},
	}

	for _, tt := range tests {
		node, ok := graph.GetNode(tt.id)
		if !ok {
			t.Fatalf("node %q not found", tt.id)
		}
		if node.Status != tt.status {
			t.Errorf("node %q: expected status %q, got %q", tt.id, tt.status, node.Status)
		}
		if node.Location != tt.location {
			t.Errorf("node %q: expected location %q, got %q", tt.id, tt.location, node.Location)
		}
		if node.SiteID != tt.siteID {
			t.Errorf("node %q: expected siteID %q, got %q", tt.id, tt.siteID, node.SiteID)
		}
	}
}

func TestGraphBuilder_BuildFromState_Rebuild_ReplacesAll(t *testing.T) {
	// Rebuild should completely replace old graph
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	// First build
	devices1 := []DeviceState{
		{ID: "site-1", Name: "Old Site", Type: DeviceTypeSite, Status: StatusOnline},
		{ID: "cam-1", Name: "Old Camera", Type: DeviceTypeCamera, ParentID: "site-1", Status: StatusOnline},
	}
	added1, err := builder.BuildFromState(devices1)
	if err != nil {
		t.Fatalf("first BuildFromState error: %v", err)
	}
	if added1 != 2 {
		t.Errorf("expected 2 devices in first build, got %d", added1)
	}

	// Second build with completely different topology
	devices2 := []DeviceState{
		{ID: "site-2", Name: "New Site", Type: DeviceTypeSite, Status: StatusOnline},
		{ID: "nvr-2", Name: "New NVR", Type: DeviceTypeNVR, ParentID: "site-2", Status: StatusOnline},
		{ID: "cam-2", Name: "New Camera", Type: DeviceTypeCamera, ParentID: "nvr-2", Status: StatusOnline},
	}
	added2, err := builder.BuildFromState(devices2)
	if err != nil {
		t.Fatalf("second BuildFromState error: %v", err)
	}
	if added2 != 3 {
		t.Errorf("expected 3 devices in second build, got %d", added2)
	}

	// Old nodes should be gone
	if _, ok := graph.GetNode("site-1"); ok {
		t.Error("old node site-1 should not exist after rebuild")
	}
	if _, ok := graph.GetNode("cam-1"); ok {
		t.Error("old node cam-1 should not exist after rebuild")
	}

	// New nodes should exist
	if _, ok := graph.GetNode("site-2"); !ok {
		t.Error("new node site-2 should exist after rebuild")
	}
	if _, ok := graph.GetNode("cam-2"); !ok {
		t.Error("new node cam-2 should exist after rebuild")
	}

	// Total count should match new topology
	if graph.NodeCount() != 3 {
		t.Errorf("expected 3 nodes after rebuild, got %d", graph.NodeCount())
	}

	// Root should be new site
	if len(graph.roots) != 1 || graph.roots[0] != "site-2" {
		t.Errorf("expected root site-2 after rebuild, got %v", graph.roots)
	}

	// Parent relationship should be from new topology
	parent := graph.GetParent("cam-2")
	if parent == nil || parent.ID != "nvr-2" {
		t.Errorf("expected cam-2 parent to be nvr-2, got %v", parent)
	}

	// GraphVersion should increment on rebuild
	if builder.GraphVersion() != 2 {
		t.Errorf("expected version 2 after rebuild, got %d", builder.GraphVersion())
	}
}

func TestGraphBuilder_BuildFromState_EmptyDevices(t *testing.T) {
	// Empty array should produce empty graph
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	added, err := builder.BuildFromState([]DeviceState{})
	if err != nil {
		t.Fatalf("BuildFromState error: %v", err)
	}
	if added != 0 {
		t.Errorf("expected 0 devices, got %d", added)
	}
	if graph.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", graph.NodeCount())
	}
	if len(graph.roots) != 0 {
		t.Errorf("expected 0 roots, got %d", len(graph.roots))
	}
}

func TestGraphBuilder_BuildFromState_SingleRoot(t *testing.T) {
	// Single root node — no parent
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	devices := []DeviceState{
		{ID: "site-1", Name: "Single Site", Type: DeviceTypeSite, Status: StatusOnline},
	}

	added, err := builder.BuildFromState(devices)
	if err != nil {
		t.Fatalf("BuildFromState error: %v", err)
	}
	if added != 1 {
		t.Errorf("expected 1 device, got %d", added)
	}
	if graph.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", graph.NodeCount())
	}
	if len(graph.roots) != 1 || graph.roots[0] != "site-1" {
		t.Errorf("expected root site-1, got %v", graph.roots)
	}

	node, ok := graph.GetNode("site-1")
	if !ok {
		t.Fatal("node site-1 not found")
	}
	if node.ParentID != "" {
		t.Errorf("expected empty ParentID for root, got %q", node.ParentID)
	}
	if node.Status != StatusOnline {
		t.Errorf("expected StatusOnline, got %q", node.Status)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BACKEND.5: AddNode — Cycle Prevention Deep Chain
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_AddNode_CyclePrevention_DeepChain(t *testing.T) {
	// Deep cycle detection across many nodes
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	// Build chain: a → b → c → d → e → f
	nodes := []DeviceState{
		{ID: "a", Name: "A", Type: DeviceTypeSite},
		{ID: "b", Name: "B", Type: DeviceTypeSwitch, ParentID: "a"},
		{ID: "c", Name: "C", Type: DeviceTypeNVR, ParentID: "b"},
		{ID: "d", Name: "D", Type: DeviceTypeEncoder, ParentID: "c"},
		{ID: "e", Name: "E", Type: DeviceTypeCamera, ParentID: "d"},
		{ID: "f", Name: "F", Type: DeviceTypeCamera, ParentID: "e"},
	}
	for _, n := range nodes {
		if err := builder.AddNode(n); err != nil {
			t.Fatalf("AddNode(%q) error: %v", n.ID, err)
		}
	}

	// Try: a → f (would create a→b→c→d→e→f→a cycle)
	err := builder.AddNode(DeviceState{ID: "a", Name: "A", Type: DeviceTypeSite, ParentID: "f"})
	if err == nil {
		t.Error("expected cycle detection error when making 'a' child of 'f'")
	}

	// Try: c → a (re-parenting c from b to a — breaks old b→c link, no cycle)
	err = builder.AddNode(DeviceState{ID: "c", Name: "C", Type: DeviceTypeNVR, ParentID: "a"})
	if err != nil {
		t.Errorf("re-parenting c under a should not create a cycle (old b→c link is broken), got: %v", err)
	}

	// Verify c's parent is now a
	parent := graph.GetParent("c")
	if parent == nil || parent.ID != "a" {
		t.Errorf("expected c parent to be a after re-parenting, got %v", parent)
	}

	// Re-add d→e→f chain under re-parented c
	_ = builder.AddNode(DeviceState{ID: "d", Name: "D", Type: DeviceTypeEncoder, ParentID: "c"})
	_ = builder.AddNode(DeviceState{ID: "e", Name: "E", Type: DeviceTypeCamera, ParentID: "d"})
	_ = builder.AddNode(DeviceState{ID: "f", Name: "F", Type: DeviceTypeCamera, ParentID: "e"})

	// Try: f → b (re-parenting f from e to b — breaks old e→f link, no cycle)
	err = builder.AddNode(DeviceState{ID: "f", Name: "F", Type: DeviceTypeCamera, ParentID: "b"})
	if err != nil {
		t.Errorf("re-parenting f under b should not create a cycle (old e→f link is broken), got: %v", err)
	}

	// Verify f's parent is now b
	parent = graph.GetParent("f")
	if parent == nil || parent.ID != "b" {
		t.Errorf("expected f parent to be b after re-parenting, got %v", parent)
	}

	// Now try: b → f (would create cycle: b→? and f→b)
	// Note: this is safe because wouldCreateCycle checks ancestors of new parent.
	// b's ancestors: a → "" (no f), so no cycle detected — but it IS semantically a
	// cycle in the directed graph sense if we consider the OLD children view.
	// The actual parent chain is clean: b→a→"", f→b→a→""
	// This test documents the behavior: AddNode re-parents safely.
	err = builder.AddNode(DeviceState{ID: "b", Name: "B", Type: DeviceTypeSwitch, ParentID: "f"})
	if err == nil {
		t.Error("expected cycle detection error when making 'b' child of 'f' (b→a and f→b creates f→b→a→...→f loop)")
	}

	// Valid operation: add new node g as child of f
	err = builder.AddNode(DeviceState{ID: "g", Name: "G", Type: DeviceTypeCamera, ParentID: "f"})
	if err != nil {
		t.Errorf("expected no error for valid add, got: %v", err)
	}

	if graph.NodeCount() != 7 {
		t.Errorf("expected 7 nodes, got %d", graph.NodeCount())
	}
}

// ═══════════════════════════════════════════════════════════════════════
// BACKEND.4: Auto-Refresh Tests
// ═══════════════════════════════════════════════════════════════════════

func TestGraphBuilder_AutoRefresh_Basic(t *testing.T) {
	// Test auto-refresh with mock provider that changes state over time
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	var mu sync.Mutex
	callCount := 0
	provider := func(ctx context.Context) ([]DeviceState, error) {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		return []DeviceState{
			{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, Status: StatusOnline},
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	errCh := builder.StartAutoRefresh(ctx, 50*time.Millisecond, DeviceStateProvider(provider))

	// Wait for context to expire
	<-ctx.Done()

	// Allow some time for final refresh
	time.Sleep(20 * time.Millisecond)

	// Stop the refresh
	builder.StopAutoRefresh()

	// Verify errors channel is closed
	_, ok := <-errCh
	if ok {
		// Drain any remaining errors
		for range errCh {
		}
	}

	mu.Lock()
	gotCalls := callCount
	mu.Unlock()

	if gotCalls < 2 {
		t.Errorf("expected at least 2 provider calls (1 immediate + 1 ticker), got %d", gotCalls)
	}

	// Verify graph was populated
	if graph.NodeCount() != 1 {
		t.Errorf("expected 1 node in graph, got %d", graph.NodeCount())
	}

	node, ok := graph.GetNode("cam-1")
	if !ok {
		t.Fatal("expected cam-1 to exist")
	}
	if node.Status != StatusOnline {
		t.Errorf("expected StatusOnline, got %q", node.Status)
	}
}

func TestGraphBuilder_AutoRefresh_ContextCancel(t *testing.T) {
	// Test graceful stop via context cancellation
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	provider := func(ctx context.Context) ([]DeviceState, error) {
		return []DeviceState{
			{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, Status: StatusOnline},
		}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	errCh := builder.StartAutoRefresh(ctx, 1*time.Hour, DeviceStateProvider(provider))

	// Cancel immediately
	cancel()

	// errCh should close
	_, ok := <-errCh
	if ok {
		t.Error("expected errCh to be closed after context cancellation")
	}

	// Verify StopAutoRefresh is idempotent after cancel
	builder.StopAutoRefresh()
	builder.StopAutoRefresh()
}

func TestGraphBuilder_AutoRefresh_ProviderError(t *testing.T) {
	// Test error propagation
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	provider := func(ctx context.Context) ([]DeviceState, error) {
		return nil, fmt.Errorf("database connection refused")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	errCh := builder.StartAutoRefresh(ctx, 30*time.Millisecond, DeviceStateProvider(provider))

	// Collect errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	if len(errors) == 0 {
		t.Error("expected at least 1 error from provider")
	}

	// Verify first error mentions provider
	if len(errors) > 0 {
		if errors[0].Error() != "auto-refresh provider: database connection refused" {
			t.Errorf("unexpected error message: %v", errors[0])
		}
	}

	// Verify graph is empty (BuildFromState was never called with valid data)
	if graph.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", graph.NodeCount())
	}
}

func TestGraphBuilder_AutoRefresh_StateChange(t *testing.T) {
	// Test that graph reflects changing state from provider
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	var mu sync.Mutex
	currentState := StatusOnline
	provider := func(ctx context.Context) ([]DeviceState, error) {
		mu.Lock()
		status := currentState
		mu.Unlock()
		return []DeviceState{
			{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, Status: status},
			{ID: "cam-2", Name: "Camera-2", Type: DeviceTypeCamera, Status: StatusOnline},
		}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	errCh := builder.StartAutoRefresh(ctx, 50*time.Millisecond, DeviceStateProvider(provider))

	// Let first refresh happen
	time.Sleep(30 * time.Millisecond)

	// Change state
	mu.Lock()
	currentState = StatusOffline
	mu.Unlock()

	// Wait for second refresh
	time.Sleep(100 * time.Millisecond)

	// Drain errCh
	go func() {
		for range errCh {
		}
	}()

	// Wait for context timeout
	<-ctx.Done()
	builder.StopAutoRefresh()

	// Verify final state
	node, ok := graph.GetNode("cam-1")
	if !ok {
		t.Fatal("expected cam-1 to exist")
	}
	if node.Status != StatusOffline {
		t.Errorf("expected StatusOffline after state change, got %q", node.Status)
	}
}

func TestGraphBuilder_AutoRefresh_Restart(t *testing.T) {
	// Test StartAutoRefresh after StopAutoRefresh
	graph := NewDeviceGraph()
	builder := NewGraphBuilder(graph, nil, GraphBuilderConfig{})

	provider1 := func(ctx context.Context) ([]DeviceState, error) {
		return []DeviceState{
			{ID: "cam-1", Name: "Camera-1", Type: DeviceTypeCamera, Status: StatusOnline},
		}, nil
	}

	provider2 := func(ctx context.Context) ([]DeviceState, error) {
		return []DeviceState{
			{ID: "cam-2", Name: "Camera-2", Type: DeviceTypeCamera, Status: StatusOnline},
		}, nil
	}

	ctx1, cancel1 := context.WithCancel(context.Background())
	errCh1 := builder.StartAutoRefresh(ctx1, 1*time.Hour, DeviceStateProvider(provider1))
	cancel1()
	<-errCh1

	// Start new refresh with different provider
	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()

	errCh2 := builder.StartAutoRefresh(ctx2, 30*time.Millisecond, DeviceStateProvider(provider2))
	<-ctx2.Done()
	builder.StopAutoRefresh()

	// Drain errCh2
	for range errCh2 {
	}

	// Verify graph has cam-2 (from second provider), not cam-1
	if graph.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", graph.NodeCount())
	}
	_, ok := graph.GetNode("cam-2")
	if !ok {
		t.Error("expected cam-2 to exist after restart")
	}
	_, ok = graph.GetNode("cam-1")
	if ok {
		t.Error("cam-1 should not exist after restart")
	}
}
