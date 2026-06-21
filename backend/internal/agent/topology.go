// Package agent реализует Self-Healing Agent для автоматического восстановления устройств.
// Topology — граф топологии устройств (device → switch → NVR).
package agent

import (
	"log/slog"
	"sync"

	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
)

// TopologyNode представляет узел в графе топологии.
type TopologyNode struct {
	DeviceID   string
	DeviceName string
	VendorType string
	IP         string
	DeviceType string // camera, nvr, switch, encoder
	Status     models.DeviceStatus
}

// TopologyEdge представляет связь между устройствами.
type TopologyEdge struct {
	From string
	To   string
	Type string // uplink, cascade, poe
}

// TopologyGraph — ориентированный граф топологии CCTV-инфраструктуры.
type TopologyGraph struct {
	mu    sync.RWMutex
	nodes map[string]*TopologyNode
	edges []TopologyEdge
	adj   map[string][]string // adjacency list: deviceID → neighborIDs
}

// NewTopologyGraph создаёт пустой граф.
func NewTopologyGraph() *TopologyGraph {
	return &TopologyGraph{
		nodes: make(map[string]*TopologyNode),
		adj:   make(map[string][]string),
	}
}

// AddNode добавляет устройство в граф.
func (g *TopologyGraph) AddNode(node TopologyNode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[node.DeviceID] = &node
	if g.adj[node.DeviceID] == nil {
		g.adj[node.DeviceID] = make([]string, 0)
	}
}

// AddEdge добавляет связь.
func (g *TopologyGraph) AddEdge(edge TopologyEdge) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges = append(g.edges, edge)
	g.adj[edge.From] = append(g.adj[edge.From], edge.To)
	g.adj[edge.To] = append(g.adj[edge.To], edge.From) // ненаправленный граф
}

// GetNode возвращает узел по ID.
func (g *TopologyGraph) GetNode(deviceID string) (*TopologyNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[deviceID]
	return n, ok
}

// GetNeighbors возвращает соседей устройства.
func (g *TopologyGraph) GetNeighbors(deviceID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.adj[deviceID]
}

// GetUpstreamSwitch находит ближайший свитч/NVR вверх по топологии.
func (g *TopologyGraph) GetUpstreamSwitch(deviceID string) *TopologyNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	queue := []string{deviceID}
	visited[deviceID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, neighbor := range g.adj[current] {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			node, ok := g.nodes[neighbor]
			if !ok {
				continue
			}
			if node.DeviceType == "switch" || node.DeviceType == "nvr" {
				return node
			}
			queue = append(queue, neighbor)
		}
	}
	return nil
}

// GetDownstreamCameras возвращает все камеры за свитчом/NVR.
func (g *TopologyGraph) GetDownstreamCameras(deviceID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var cameras []string
	queue := []string{deviceID}
	visited[deviceID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		node, ok := g.nodes[current]
		if ok && node.DeviceType == "camera" && current != deviceID {
			cameras = append(cameras, current)
		}

		for _, neighbor := range g.adj[current] {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			queue = append(queue, neighbor)
		}
	}
	return cameras
}

// NodeCount возвращает количество узлов.
func (g *TopologyGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// BuildFromState строит граф из DeviceStateManager.
// Определяет тип устройства по префиксу ID и VendorType.
func BuildFromState(stateMgr state.DeviceStateManager, logger *slog.Logger) *TopologyGraph {
	g := NewTopologyGraph()
	devices := stateMgr.GetAll()

	for id, dev := range devices {
		deviceType := inferDeviceType(id, dev.VendorType)
		node := TopologyNode{
			DeviceID:   id,
			DeviceName: dev.Name,
			VendorType: dev.VendorType,
			IP:         dev.Location,
			DeviceType: deviceType,
			Status:     dev.Status,
		}
		g.AddNode(node)

		// Связываем камеры с NVR/switches по общему префиксу локации
		if deviceType == "camera" {
			for otherID, otherDev := range devices {
				if otherID == id {
					continue
				}
				otherType := inferDeviceType(otherID, otherDev.VendorType)
				if otherType == "switch" || otherType == "nvr" {
					// Эвристика: камеры и свитчи в одной подсети связаны
					if sameSubnet(dev.Location, otherDev.Location) {
						g.AddEdge(TopologyEdge{
							From: id,
							To:   otherID,
							Type: "uplink",
						})
					}
				}
			}
		}
	}

	logger.Info("topology built", "nodes", len(g.nodes), "edges", len(g.edges))
	return g
}

// inferDeviceType определяет тип устройства по ID и vendor.
func inferDeviceType(deviceID, vendorType string) string {
	// По префиксу ID
	switch {
	case containsPrefix(deviceID, "snmp_switch_"), containsPrefix(deviceID, "switch_"):
		return "switch"
	case containsPrefix(deviceID, "nvr_"), containsPrefix(deviceID, "dvr_"):
		return "nvr"
	case containsPrefix(deviceID, "hikvision_"), containsPrefix(deviceID, "dahua_"),
		containsPrefix(deviceID, "vigi_"), containsPrefix(deviceID, "snmp_"),
		containsPrefix(deviceID, "camera_"):
		return "camera"
	case vendorType == "Hikvision" || vendorType == "Dahua" || vendorType == "Dahua/Intelbras":
		return "camera"
	}
	return "camera" // по умолчанию считаем камерой
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func sameSubnet(ip1, ip2 string) bool {
	if ip1 == "" || ip2 == "" {
		return false
	}
	// Простая эвристика: первые 3 октета совпадают
	return firstThreeOctets(ip1) == firstThreeOctets(ip2)
}

func firstThreeOctets(ip string) string {
	dots := 0
	for i, c := range ip {
		if c == '.' {
			dots++
			if dots == 3 {
				return ip[:i]
			}
		}
	}
	return ip
}
