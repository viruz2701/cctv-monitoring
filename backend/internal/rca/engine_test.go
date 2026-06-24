// Package rca — tests for Root Cause Analysis engine
package rca

import (
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════
// Device Graph tests
// ═══════════════════════════════════════════════════════════════════════

func TestDeviceGraph_AddAndGet(t *testing.T) {
	g := NewDeviceGraph()

	g.AddNode("site-1", "Main Office", DeviceTypeSite, "")
	g.AddNode("sw-1", "Switch-1", DeviceTypeSwitch, "site-1")
	g.AddNode("nvr-1", "NVR-1", DeviceTypeNVR, "sw-1")
	g.AddNode("cam-1", "Camera-1", DeviceTypeCamera, "nvr-1")
	g.AddNode("cam-2", "Camera-2", DeviceTypeCamera, "nvr-1")

	if g.NodeCount() != 5 {
		t.Errorf("expected 5 nodes, got %d", g.NodeCount())
	}

	// GetNode
	node, ok := g.GetNode("nvr-1")
	if !ok {
		t.Fatal("expected nvr-1 to exist")
	}
	if node.Name != "NVR-1" {
		t.Errorf("expected NVR-1, got %s", node.Name)
	}
	if node.ParentID != "sw-1" {
		t.Errorf("expected parent sw-1, got %s", node.ParentID)
	}
}

func TestDeviceGraph_Hierarchy(t *testing.T) {
	g := buildTestGraph()

	// Site → Switch → NVR → Camera
	swChildren := g.GetChildren("site-1")
	if len(swChildren) != 1 {
		t.Errorf("expected 1 child of site-1 (sw-1), got %d", len(swChildren))
	}

	nvrChildren := g.GetChildren("sw-1")
	if len(nvrChildren) != 2 {
		t.Errorf("expected 2 children of sw-1 (nvr-1, nvr-2), got %d", len(nvrChildren))
	}

	camChildren := g.GetChildren("nvr-1")
	if len(camChildren) != 3 {
		t.Errorf("expected 3 children of nvr-1, got %d", len(camChildren))
	}
}

func TestDeviceGraph_Ancestors(t *testing.T) {
	g := buildTestGraph()

	// Camera-1 → NVR-1 → Switch-1 → Site-1
	ancestors := g.GetAncestors("cam-1")
	if len(ancestors) != 3 {
		t.Fatalf("expected 3 ancestors, got %d", len(ancestors))
	}
	if ancestors[0].ID != "nvr-1" {
		t.Errorf("expected first ancestor nvr-1, got %s", ancestors[0].ID)
	}
	if ancestors[1].ID != "sw-1" {
		t.Errorf("expected second ancestor sw-1, got %s", ancestors[1].ID)
	}
	if ancestors[2].ID != "site-1" {
		t.Errorf("expected third ancestor site-1, got %s", ancestors[2].ID)
	}
}

func TestDeviceGraph_Descendants(t *testing.T) {
	g := buildTestGraph()

	// Switch-1 → 2 NVRs + 5 cameras
	desc := g.GetAllDescendants("sw-1")
	if len(desc) != 7 { // nvr-1, nvr-2, cam-1..cam-5
		t.Fatalf("expected 7 descendants of sw-1, got %d", len(desc))
	}

	// NVR-1 → 3 cameras
	desc = g.GetAllDescendants("nvr-1")
	if len(desc) != 3 {
		t.Fatalf("expected 3 descendants of nvr-1, got %d", len(desc))
	}

	// Camera has no descendants
	desc = g.GetAllDescendants("cam-1")
	if len(desc) != 0 {
		t.Errorf("expected 0 descendants of cam-1, got %d", len(desc))
	}
}

func TestDeviceGraph_Parent(t *testing.T) {
	g := buildTestGraph()

	parent := g.GetParent("cam-1")
	if parent == nil {
		t.Fatal("expected parent for cam-1")
	}
	if parent.ID != "nvr-1" {
		t.Errorf("expected parent nvr-1, got %s", parent.ID)
	}

	// Root has no parent
	parent = g.GetParent("site-1")
	if parent != nil {
		t.Error("expected nil parent for root")
	}
}

func TestDeviceGraph_UpdateStatus(t *testing.T) {
	g := buildTestGraph()

	g.UpdateStatus("sw-1", StatusOffline)

	node, _ := g.GetNode("sw-1")
	if node.Status != StatusOffline {
		t.Errorf("expected OFFLINE, got %s", node.Status)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// RCA Engine tests
// ═══════════════════════════════════════════════════════════════════════

func TestRCAEngine_DeviceIsRootCause(t *testing.T) {
	g := buildTestGraph()
	engine := NewRCAEngine(g, nil)

	// Camera-1 goes offline but its parents are online
	// → Camera-1 is the root cause
	g.UpdateStatus("cam-1", StatusOffline)

	result := engine.Analyze(RCAEvent{
		DeviceID:  "cam-1",
		EventType: "offline",
		Severity:  "high",
	})

	if !result.IsRootCause {
		t.Error("expected camera to be root cause (parents are online)")
	}
	if result.RootCause.ID != "cam-1" {
		t.Errorf("expected root cause cam-1, got %s", result.RootCause.ID)
	}
	if result.BlastRadius != 0 {
		t.Errorf("expected 0 blast radius for leaf device, got %d", result.BlastRadius)
	}
}

func TestRCAEngine_ParentIsRootCause(t *testing.T) {
	g := buildTestGraph()
	engine := NewRCAEngine(g, nil)

	// Switch-1 goes offline → all cameras/NVRs under it are affected
	g.UpdateStatus("sw-1", StatusOffline)
	g.UpdateStatus("cam-1", StatusOffline)

	result := engine.Analyze(RCAEvent{
		DeviceID:  "cam-1",
		EventType: "offline",
		Severity:  "critical",
	})

	if result.IsRootCause {
		t.Error("expected camera NOT to be root cause (switch is offline)")
	}
	if result.RootCause.ID != "sw-1" {
		t.Errorf("expected root cause sw-1, got %s", result.RootCause.ID)
	}
	if result.BlastRadius == 0 {
		t.Error("expected positive blast radius")
	}
	if result.BlastRadiusByType == nil {
		t.Fatal("expected blast radius by type")
	}
	if result.BlastRadiusByType["camera"] != 5 {
		t.Errorf("expected 5 cameras affected, got %d", result.BlastRadiusByType["camera"])
	}
	if result.BlastRadiusByType["nvr"] != 2 {
		t.Errorf("expected 2 NVRs affected, got %d", result.BlastRadiusByType["nvr"])
	}
}

func TestRCAEngine_ImpactDescription(t *testing.T) {
	g := buildTestGraph()
	engine := NewRCAEngine(g, nil)

	g.UpdateStatus("sw-1", StatusOffline)
	g.UpdateStatus("cam-1", StatusOffline)

	result := engine.Analyze(RCAEvent{
		DeviceID:  "cam-1",
		EventType: "offline",
		Severity:  "critical",
	})

	if result.ImpactDescription == "" {
		t.Fatal("expected non-empty impact description")
	}

	if result.Confidence < 0.5 || result.Confidence > 1.0 {
		t.Errorf("expected confidence 0.5-1.0, got %f", result.Confidence)
	}
}

func TestRCAEngine_NoAffectedDevices(t *testing.T) {
	g := NewDeviceGraph()
	g.AddNode("cam-1", "Standalone Camera", DeviceTypeCamera, "")

	engine := NewRCAEngine(g, nil)

	g.UpdateStatus("cam-1", StatusOffline)

	result := engine.Analyze(RCAEvent{
		DeviceID:  "cam-1",
		EventType: "offline",
	})

	if !result.IsRootCause {
		t.Error("expected standalone camera to be root cause")
	}
	if result.BlastRadius != 0 {
		t.Errorf("expected 0 blast radius, got %d", result.BlastRadius)
	}
	if result.ImpactDescription == "" {
		t.Error("expected non-empty impact description")
	}
}

func TestRCAEngine_DeepHierarchy(t *testing.T) {
	// Site → Rack → Switch → NVR → Camera
	g := NewDeviceGraph()
	g.AddNode("site-1", "Data Center", DeviceTypeSite, "")
	g.AddNode("rack-1", "Rack-A", DeviceTypeRack, "site-1")
	g.AddNode("sw-1", "Core Switch", DeviceTypeSwitch, "rack-1")
	g.AddNode("nvr-1", "NVR-1", DeviceTypeNVR, "sw-1")
	g.AddNode("cam-1", "Camera-1", DeviceTypeCamera, "nvr-1")

	engine := NewRCAEngine(g, nil)

	// Rack fails → everything under it goes down
	g.UpdateStatus("rack-1", StatusOffline)
	g.UpdateStatus("cam-1", StatusOffline)

	result := engine.Analyze(RCAEvent{
		DeviceID:  "cam-1",
		EventType: "offline",
		Severity:  "critical",
	})

	if result.RootCause.ID != "rack-1" {
		t.Errorf("expected root cause rack-1 (deep hierarchy), got %s", result.RootCause.ID)
	}
}

func TestRCAEngine_Cache(t *testing.T) {
	g := buildTestGraph()
	engine := NewRCAEngine(g, nil)

	g.UpdateStatus("cam-1", StatusOffline)
	result := engine.Analyze(RCAEvent{DeviceID: "cam-1", EventType: "offline"})

	// Check cache
	cached, ok := engine.GetCachedResult("cam-1")
	if !ok {
		t.Error("expected cached result")
	}
	if cached.RootCause.ID != result.RootCause.ID {
		t.Error("cached result mismatch")
	}

	// Invalidate
	engine.InvalidateCache("cam-1")
	_, ok = engine.GetCachedResult("cam-1")
	if ok {
		t.Error("expected no cached result after invalidation")
	}
}

func TestRCAEngine_UnknownDevice(t *testing.T) {
	g := NewDeviceGraph()
	engine := NewRCAEngine(g, nil)

	result := engine.Analyze(RCAEvent{
		DeviceID:  "unknown-device",
		EventType: "offline",
	})

	if result.Confidence != 0.0 {
		t.Errorf("expected 0 confidence for unknown device, got %f", result.Confidence)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func buildTestGraph() *DeviceGraph {
	g := NewDeviceGraph()

	// Site → Switch → NVR → Camera
	g.AddNode("site-1", "Main Office", DeviceTypeSite, "")
	g.AddNode("sw-1", "Switch-1", DeviceTypeSwitch, "site-1")
	g.AddNode("nvr-1", "NVR-1", DeviceTypeNVR, "sw-1")
	g.AddNode("nvr-2", "NVR-2", DeviceTypeNVR, "sw-1")
	g.AddNode("cam-1", "Camera-1", DeviceTypeCamera, "nvr-1")
	g.AddNode("cam-2", "Camera-2", DeviceTypeCamera, "nvr-1")
	g.AddNode("cam-3", "Camera-3", DeviceTypeCamera, "nvr-1")
	g.AddNode("cam-4", "Camera-4", DeviceTypeCamera, "nvr-2")
	g.AddNode("cam-5", "Camera-5", DeviceTypeCamera, "nvr-2")

	return g
}
