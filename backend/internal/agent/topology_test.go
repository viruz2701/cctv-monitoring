package agent

import (
	"log/slog"
	"testing"

	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
)

func TestNewTopologyGraph(t *testing.T) {
	g := NewTopologyGraph()
	if g == nil {
		t.Fatal("NewTopologyGraph returned nil")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected 0 nodes, got %d", g.NodeCount())
	}
}

func TestTopologyAddNode(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode(TopologyNode{
		DeviceID:   "cam-001",
		DeviceName: "Front Gate",
		VendorType: "Hikvision",
		IP:         "192.168.1.10",
		DeviceType: "camera",
	})
	if g.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", g.NodeCount())
	}
	n, ok := g.GetNode("cam-001")
	if !ok {
		t.Fatal("GetNode returned false for existing node")
	}
	if n.DeviceName != "Front Gate" {
		t.Errorf("expected 'Front Gate', got %q", n.DeviceName)
	}
	if n.IP != "192.168.1.10" {
		t.Errorf("expected '192.168.1.10', got %q", n.IP)
	}
}

func TestTopologyGetNodeMissing(t *testing.T) {
	g := NewTopologyGraph()
	_, ok := g.GetNode("nonexistent")
	if ok {
		t.Error("GetNode should return false for missing node")
	}
}

func TestTopologyAddEdge(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode(TopologyNode{DeviceID: "cam-001", DeviceType: "camera", IP: "192.168.1.10"})
	g.AddNode(TopologyNode{DeviceID: "sw-001", DeviceType: "switch", IP: "192.168.1.1"})
	g.AddEdge(TopologyEdge{From: "cam-001", To: "sw-001", Type: "uplink"})

	neighbors := g.GetNeighbors("cam-001")
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor, got %d", len(neighbors))
	}
	if neighbors[0] != "sw-001" {
		t.Errorf("expected neighbor 'sw-001', got %q", neighbors[0])
	}

	// Проверяем обратную связь (ненаправленный граф)
	revNeighbors := g.GetNeighbors("sw-001")
	if len(revNeighbors) != 1 {
		t.Fatalf("expected 1 reverse neighbor, got %d", len(revNeighbors))
	}
	if revNeighbors[0] != "cam-001" {
		t.Errorf("expected reverse neighbor 'cam-001', got %q", revNeighbors[0])
	}
}

func TestTopologyGetUpstreamSwitch(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode(TopologyNode{DeviceID: "cam-001", DeviceType: "camera", IP: "192.168.1.10"})
	g.AddNode(TopologyNode{DeviceID: "sw-001", DeviceType: "switch", IP: "192.168.1.1"})
	g.AddNode(TopologyNode{DeviceID: "nvr-001", DeviceType: "nvr", IP: "192.168.1.100"})
	g.AddEdge(TopologyEdge{From: "cam-001", To: "sw-001", Type: "uplink"})
	g.AddEdge(TopologyEdge{From: "sw-001", To: "nvr-001", Type: "uplink"})

	up := g.GetUpstreamSwitch("cam-001")
	if up == nil {
		t.Fatal("GetUpstreamSwitch returned nil")
	}
	if up.DeviceType != "switch" {
		t.Errorf("expected switch, got %q", up.DeviceType)
	}
	if up.DeviceID != "sw-001" {
		t.Errorf("expected 'sw-001', got %q", up.DeviceID)
	}
}

func TestTopologyGetUpstreamSwitchNone(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode(TopologyNode{DeviceID: "cam-001", DeviceType: "camera", IP: "192.168.1.10"})

	up := g.GetUpstreamSwitch("cam-001")
	if up != nil {
		t.Error("GetUpstreamSwitch should return nil when no switch exists")
	}
}

func TestTopologyGetDownstreamCameras(t *testing.T) {
	g := NewTopologyGraph()
	g.AddNode(TopologyNode{DeviceID: "cam-001", DeviceType: "camera", IP: "192.168.1.10"})
	g.AddNode(TopologyNode{DeviceID: "cam-002", DeviceType: "camera", IP: "192.168.1.11"})
	g.AddNode(TopologyNode{DeviceID: "cam-003", DeviceType: "camera", IP: "192.168.1.12"})
	g.AddNode(TopologyNode{DeviceID: "sw-001", DeviceType: "switch", IP: "192.168.1.1"})
	g.AddEdge(TopologyEdge{From: "cam-001", To: "sw-001", Type: "poe"})
	g.AddEdge(TopologyEdge{From: "cam-002", To: "sw-001", Type: "poe"})
	g.AddEdge(TopologyEdge{From: "cam-003", To: "sw-001", Type: "poe"})

	cameras := g.GetDownstreamCameras("sw-001")
	if len(cameras) != 3 {
		t.Fatalf("expected 3 downstream cameras, got %d", len(cameras))
	}
}

func TestInferDeviceType(t *testing.T) {
	tests := []struct {
		deviceID   string
		vendorType string
		expected   string
	}{
		{"snmp_switch_01", "", "switch"},
		{"switch_core", "", "switch"},
		{"nvr_main", "", "nvr"},
		{"dvr_backup", "", "nvr"},
		{"hikvision_front", "", "camera"},
		{"dahua_gate", "", "camera"},
		{"vigi_wall", "", "camera"},
		{"snmp_cam_01", "", "camera"},
		{"camera_unknown", "", "camera"},
		{"unknown_device", "Hikvision", "camera"},
		{"unknown_device", "Dahua", "camera"},
		{"unknown_device", "Dahua/Intelbras", "camera"},
		{"unknown_device", "Generic", "camera"},
	}

	for _, tt := range tests {
		result := inferDeviceType(tt.deviceID, tt.vendorType)
		if result != tt.expected {
			t.Errorf("inferDeviceType(%q, %q) = %q, want %q",
				tt.deviceID, tt.vendorType, result, tt.expected)
		}
	}
}

func TestContainsPrefix(t *testing.T) {
	tests := []struct {
		s, prefix string
		expected  bool
	}{
		{"hikvision_front", "hikvision_", true},
		{"hikvision_front", "dahua_", false},
		{"short", "longer_prefix", false},
		{"exact", "exact", true},
		{"", "prefix", false},
		{"prefix", "", true},
	}

	for _, tt := range tests {
		result := containsPrefix(tt.s, tt.prefix)
		if result != tt.expected {
			t.Errorf("containsPrefix(%q, %q) = %v, want %v",
				tt.s, tt.prefix, result, tt.expected)
		}
	}
}

func TestSameSubnet(t *testing.T) {
	tests := []struct {
		ip1, ip2 string
		expected bool
	}{
		{"192.168.1.10", "192.168.1.1", true},
		{"192.168.1.10", "192.168.2.1", false},
		{"10.0.0.5", "10.0.0.1", true},
		{"", "192.168.1.1", false},
		{"192.168.1.1", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		result := sameSubnet(tt.ip1, tt.ip2)
		if result != tt.expected {
			t.Errorf("sameSubnet(%q, %q) = %v, want %v",
				tt.ip1, tt.ip2, result, tt.expected)
		}
	}
}

func TestFirstThreeOctets(t *testing.T) {
	tests := []struct {
		ip       string
		expected string
	}{
		{"192.168.1.10", "192.168.1"},
		{"10.0.0.5", "10.0.0"},
		{"invalid", "invalid"},
		{"1.2", "1.2"},
	}

	for _, tt := range tests {
		result := firstThreeOctets(tt.ip)
		if result != tt.expected {
			t.Errorf("firstThreeOctets(%q) = %q, want %q",
				tt.ip, result, tt.expected)
		}
	}
}

func TestBuildFromState(t *testing.T) {
	mgr := state.NewInMemoryStateManager()
	mgr.Set(&models.Device{
		DeviceID:   "hikvision_gate",
		Name:       "Gate Camera",
		VendorType: "Hikvision",
		Location:   "192.168.1.10",
		Status:     models.StatusOnline,
	})
	mgr.Set(&models.Device{
		DeviceID:   "snmp_switch_01",
		Name:       "Core Switch",
		VendorType: "Generic",
		Location:   "192.168.1.1",
		Status:     models.StatusOnline,
	})
	mgr.Set(&models.Device{
		DeviceID:   "dahua_wall",
		Name:       "Wall Camera",
		VendorType: "Dahua",
		Location:   "192.168.2.10",
		Status:     models.StatusOnline,
	})

	logger := slog.Default()
	g := BuildFromState(mgr, logger)

	if g.NodeCount() != 3 {
		t.Errorf("expected 3 nodes, got %d", g.NodeCount())
	}

	// Камера в подсети со свитчом должна быть связана
	n, ok := g.GetNode("hikvision_gate")
	if !ok {
		t.Fatal("hikvision_gate not found")
	}
	if n.DeviceType != "camera" {
		t.Errorf("expected camera, got %q", n.DeviceType)
	}

	// Камера в другой подсети не должна быть связана со свитчом
	neighbors := g.GetNeighbors("dahua_wall")
	if len(neighbors) != 0 {
		t.Errorf("expected 0 neighbors for dahua_wall (different subnet), got %d", len(neighbors))
	}

	// Камера в той же подсети должна быть связана
	neighbors = g.GetNeighbors("hikvision_gate")
	if len(neighbors) != 1 {
		t.Errorf("expected 1 neighbor for hikvision_gate, got %d", len(neighbors))
	}
}
