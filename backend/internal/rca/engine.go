// Package rca — Root Cause Analysis engine для CCTV Health Monitor.
//
// CCTV-2.1.3: BFS по иерархии устройств для определения первопричины сбоя.
//
// Алгоритм:
//   1. При получении alarm/offline события — BFS вверх по топологии
//   2. Если parent (switch/NVR) тоже offline — он и есть root cause
//   3. Если parent online — проблема в самом устройстве
//   4. После определения root cause — BFS вниз для blast radius
//
// Пример:
//   "Switch-1 down → 16 cameras suspended"
//   "NVR-3 offline → 64 cameras degraded"
//   "Power supply failed on Floor-2 → 8 cameras + 2 switches affected"
//
// Compliance:
//   - CCTV Core IP (уникальная фича)
//   - IEC 62443 SR 7.1 (Resource availability)
//   - ISO 27001 A.12.6.1 (Capacity management)
package rca

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// DeviceType — тип устройства в иерархии.
type DeviceType string

const (
	DeviceTypeSite   DeviceType = "site"
	DeviceTypeSwitch DeviceType = "switch"
	DeviceTypeNVR    DeviceType = "nvr"
	DeviceTypeDVR    DeviceType = "dvr"
	DeviceTypeCamera DeviceType = "camera"
	DeviceTypeEncoder DeviceType = "encoder"
	DeviceTypeServer DeviceType = "server"
	DeviceTypeUPS    DeviceType = "ups"
	DeviceTypeRack   DeviceType = "rack"
)

// DeviceStatus — статус устройства.
type DeviceStatus string

const (
	StatusOnline  DeviceStatus = "ONLINE"
	StatusOffline DeviceStatus = "OFFLINE"
	StatusWarning DeviceStatus = "WARNING"
	StatusSuspended DeviceStatus = "SUSPENDED"
	StatusDegraded DeviceStatus = "DEGRADED"
	StatusUnknown DeviceStatus = "UNKNOWN"
)

// ═══════════════════════════════════════════════════════════════════════
// Device Graph
// ═══════════════════════════════════════════════════════════════════════

// DeviceNode — узел графа устройств.
type DeviceNode struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Type       DeviceType   `json:"type"`
	Status     DeviceStatus `json:"status"`
	Location   string       `json:"location,omitempty"`
	SiteID     string       `json:"site_id,omitempty"`
	ParentID   string       `json:"parent_id,omitempty"` // родительский узел (uplink)
	Children   []string     `json:"children,omitempty"`   // дочерние узлы
}

// DeviceGraph — ориентированный граф иерархии устройств (rooted tree).
//
// Иерархия:
//
//	Site
//	  ├── Switch-1
//	  │    ├── NVR-1
//	  │    │    ├── Camera-1
//	  │    │    ├── Camera-2
//	  │    │    └── Camera-3
//	  │    └── NVR-2
//	  │         ├── Camera-4
//	  │         └── Camera-5
//	  ├── Switch-2
//	  │    ├── Camera-6
//	  │    └── Camera-7
//	  └── Server-1 (VMS)
type DeviceGraph struct {
	mu    sync.RWMutex
	nodes map[string]*DeviceNode // deviceID → node
	roots []string               // корневые узлы (sites)
}

// NewDeviceGraph создаёт пустой граф устройств.
func NewDeviceGraph() *DeviceGraph {
	return &DeviceGraph{
		nodes: make(map[string]*DeviceNode),
		roots: make([]string, 0),
	}
}

// AddNode добавляет устройство в граф.
func (g *DeviceGraph) AddNode(id, name string, dtype DeviceType, parentID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	node, exists := g.nodes[id]
	if exists {
		node.Name = name
		node.Type = dtype
		node.ParentID = parentID
		return
	}

	node = &DeviceNode{
		ID:       id,
		Name:     name,
		Type:     dtype,
		Status:   StatusUnknown,
		ParentID: parentID,
		Children: make([]string, 0),
	}
	g.nodes[id] = node

	// Добавляем себя в children родителя
	if parentID != "" {
		parent, ok := g.nodes[parentID]
		if ok {
			parent.Children = append(parent.Children, id)
		}
	} else {
		// Корневой узел
		g.roots = append(g.roots, id)
	}
}

// UpdateStatus обновляет статус устройства.
func (g *DeviceGraph) UpdateStatus(deviceID string, status DeviceStatus) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if node, ok := g.nodes[deviceID]; ok {
		node.Status = status
	}
}

// GetNode возвращает узел по ID.
func (g *DeviceGraph) GetNode(deviceID string) (*DeviceNode, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	n, ok := g.nodes[deviceID]
	return n, ok
}

// GetChildren возвращает прямых потомков узла.
func (g *DeviceGraph) GetChildren(deviceID string) []*DeviceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, ok := g.nodes[deviceID]
	if !ok {
		return nil
	}

	result := make([]*DeviceNode, 0, len(node.Children))
	for _, childID := range node.Children {
		if child, ok := g.nodes[childID]; ok {
			result = append(result, child)
		}
	}
	return result
}

// GetParent возвращает родительский узел.
func (g *DeviceGraph) GetParent(deviceID string) *DeviceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, ok := g.nodes[deviceID]
	if !ok || node.ParentID == "" {
		return nil
	}
	parent, ok := g.nodes[node.ParentID]
	if !ok {
		return nil
	}
	return parent
}

// GetAncestors возвращает всех предков узла (от родителя до корня).
func (g *DeviceGraph) GetAncestors(deviceID string) []*DeviceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var ancestors []*DeviceNode
	current := deviceID

	for {
		node, ok := g.nodes[current]
		if !ok || node.ParentID == "" {
			break
		}
		parent, ok := g.nodes[node.ParentID]
		if !ok {
			break
		}
		ancestors = append(ancestors, parent)
		current = node.ParentID
	}

	return ancestors
}

// GetAllDescendants возвращает ВСЕХ потомков узла (рекурсивно).
func (g *DeviceGraph) GetAllDescendants(deviceID string) []*DeviceNode {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var descendants []*DeviceNode
	queue := []string{deviceID}
	visited[deviceID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		node, ok := g.nodes[current]
		if ok && current != deviceID {
			descendants = append(descendants, node)
		}

		if !ok {
			continue
		}
		for _, childID := range node.Children {
			if !visited[childID] {
				visited[childID] = true
				queue = append(queue, childID)
			}
		}
	}

	return descendants
}

// NodeCount возвращает количество узлов.
func (g *DeviceGraph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// ═══════════════════════════════════════════════════════════════════════
// RCA Engine
// ═══════════════════════════════════════════════════════════════════════

// RCAEngine — движок Root Cause Analysis.
//
// Использует DeviceGraph для анализа сбоев:
//   - BFS вверх: поиск первопричины (первый offline parent)
//   - BFS вниз: расчёт blast radius (сколько устройств затронуто)
//   - Impact Report: человекочитаемый отчёт
type RCAEngine struct {
	graph  *DeviceGraph
	logger *slog.Logger

	// Кэш результатов RCA (device_id → последний анализ)
	mu     sync.RWMutex
	cache  map[string]*RCAResult
}

// RCAEvent — входное событие для RCA.
type RCAEvent struct {
	DeviceID    string    `json:"device_id"`
	EventType   string    `json:"event_type"`   // "alarm", "offline", "warning", "suspended"
	Severity    string    `json:"severity"`     // "critical", "high", "medium", "low"
	Timestamp   time.Time `json:"timestamp"`
}

// RCAResult — результат Root Cause Analysis.
type RCAResult struct {
	Event              RCAEvent        `json:"event"`
	RootCause          *DeviceNode     `json:"root_cause,omitempty"`    // первопричина
	FailedDevice       *DeviceNode     `json:"failed_device"`          // устройство, с которого пришёл alarm
	AffectedDevices    []*DeviceNode   `json:"affected_devices"`       // все затронутые устройства (включая root cause)
	BlastRadius        int             `json:"blast_radius"`            // количество затронутых
	BlastRadiusByType  map[string]int  `json:"blast_radius_by_type"`   // по типам: {"camera": 16, "nvr": 2}
	ImpactDescription  string          `json:"impact_description"`     // "Switch-1 down → 16 cameras suspended"
	IsRootCause        bool            `json:"is_root_cause"`          // true если проблемное устройство = первопричина
	Confidence         float64         `json:"confidence"`              // 0.0 - 1.0
	AnalyzedAt         time.Time       `json:"analyzed_at"`
}

// NewRCAEngine создаёт RCA Engine.
func NewRCAEngine(graph *DeviceGraph, logger *slog.Logger) *RCAEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &RCAEngine{
		graph:  graph,
		logger: logger.With("component", "rca"),
		cache:  make(map[string]*RCAResult),
	}
}

// Analyze анализирует событие и определяет root cause.
//
// Алгоритм:
//  1. Получаем устройство из графа
//  2. BFS вверх по родителям:
//     a. Если parent OFFLINE → parent = root cause, рекурсивно проверяем его parent
//     b. Если parent ONLINE → текущий узел = root cause
//  3. BFS вниз от root cause → blast radius
//  4. Формируем Impact Report
func (r *RCAEngine) Analyze(event RCAEvent) *RCAResult {
	r.logger.Info("rca analyzing",
		"device_id", event.DeviceID,
		"event_type", event.EventType,
	)

	device, ok := r.graph.GetNode(event.DeviceID)
	if !ok {
		r.logger.Warn("rca device not found in graph", "device_id", event.DeviceID)
		return &RCAResult{
			Event:        event,
			FailedDevice: &DeviceNode{ID: event.DeviceID, Name: event.DeviceID, Status: StatusUnknown},
			Confidence:   0.0,
			AnalyzedAt:   time.Now(),
		}
	}

	// ── Шаг 1: BFS вверх — поиск root cause ─────────────────────
	rootCause, path := r.findRootCause(device)
	isRootCause := rootCause.ID == event.DeviceID

	// ── Шаг 2: BFS вниз — расчёт blast radius ───────────────────
	affected := r.graph.GetAllDescendants(rootCause.ID)

	// ── Шаг 3: Статистика по типам ───────────────────────────────
	blastByType := make(map[string]int)
	for _, a := range affected {
		blastByType[string(a.Type)]++
	}

	// ── Шаг 4: Impact description ────────────────────────────────
	desc := r.generateImpactDescription(rootCause, affected, blastByType)

	// Confidence
	confidence := 0.9
	if len(path) > 1 {
		// Если root cause через несколько hop'ов — уверенность чуть ниже
		confidence = 0.9 - float64(len(path)-1)*0.05
		if confidence < 0.5 {
			confidence = 0.5
		}
	}

	result := &RCAResult{
		Event:             event,
		RootCause:         rootCause,
		FailedDevice:      device,
		AffectedDevices:   affected,
		BlastRadius:       len(affected),
		BlastRadiusByType: blastByType,
		ImpactDescription: desc,
		IsRootCause:       isRootCause,
		Confidence:        confidence,
		AnalyzedAt:        time.Now(),
	}

	// Кэшируем
	r.mu.Lock()
	r.cache[event.DeviceID] = result
	r.mu.Unlock()

	r.logger.Info("rca complete",
		"device_id", event.DeviceID,
		"root_cause", rootCause.ID,
		"blast_radius", result.BlastRadius,
		"description", desc,
	)

	return result
}

// findRootCause ищет первопричину: BFS вверх по иерархии до первого offline предка.
//
// Алгоритм:
//  1. Собираем ВСЕХ предков устройства
//  2. Ищем среди них первого (сверху вниз) с offline/suspended статусом
//  3. Если нашли — он root cause
//  4. Если не нашли — само устройство root cause
func (r *RCAEngine) findRootCause(device *DeviceNode) (rootCause *DeviceNode, path []*DeviceNode) {
	// Собираем всех предков
	ancestors := r.graph.GetAncestors(device.ID)

	// Ищем первого offline предка (снизу вверх: от ближайшего к корню)
	// ancestors[0] = parent, ancestors[1] = grandparent, ...
	for _, ancestor := range ancestors {
		path = append(path, ancestor)
		if ancestor.Status == StatusOffline || ancestor.Status == StatusSuspended {
			return ancestor, path
		}
	}

	// Никто из предков не offline — само устройство root cause
	return device, path
}

// generateImpactDescription формирует человекочитаемое описание.
//
// Примеры:
//   - "Switch-1 (192.168.1.1) is offline → 16 cameras, 2 NVRs affected"
//   - "NVR-3 degraded → 48 cameras with degraded streaming"
//   - "Power failure on Rack-A → 2 switches, 4 NVRs, 64 cameras offline"
func (r *RCAEngine) generateImpactDescription(rootCause *DeviceNode, affected []*DeviceNode, byType map[string]int) string {
	if rootCause == nil {
		return "Unknown root cause"
	}

	// Сортируем типы для консистентного вывода
	types := make([]string, 0, len(byType))
	for t := range byType {
		types = append(types, t)
	}
	sort.Strings(types)

	// Строим описание
	parts := make([]string, 0)
	for _, t := range types {
		count := byType[t]
		if count > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", count, pluralize(t, count)))
		}
	}

	// Убираем сам root cause из подсчёта
	selfCount := byType[string(rootCause.Type)]
	if selfCount > 0 && len(affected) > 0 {
		// Уже учтён
	}

	location := ""
	if rootCause.Location != "" {
		location = fmt.Sprintf(" (%s)", rootCause.Location)
	}

	if len(parts) == 0 {
		return fmt.Sprintf("%s%s is %s → no downstream devices affected",
			rootCause.Name, location, rootCause.Status)
	}

	return fmt.Sprintf("%s%s is %s → %s affected",
		rootCause.Name, location, rootCause.Status, joinParts(parts))
}

// ═══════════════════════════════════════════════════════════════════════
// Query Methods
// ═══════════════════════════════════════════════════════════════════════

// GetCachedResult возвращает кэшированный результат RCA.
func (r *RCAEngine) GetCachedResult(deviceID string) (*RCAResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, ok := r.cache[deviceID]
	return res, ok
}

// GetGraph возвращает граф устройств.
func (r *RCAEngine) GetGraph() *DeviceGraph {
	return r.graph
}

// InvalidateCache очищает кэш для устройства.
func (r *RCAEngine) InvalidateCache(deviceID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cache, deviceID)
}

// ClearCache очищает весь кэш.
func (r *RCAEngine) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]*RCAResult)
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func pluralize(word string, count int) string {
	if count == 1 {
		return word
	}
	// Простое правило для английского (для impact description)
	switch word {
	case "camera":
		return "cameras"
	case "switch":
		return "switches"
	case "nvr":
		return "NVRs"
	case "dvr":
		return "DVRs"
	case "encoder":
		return "encoders"
	case "server":
		return "servers"
	case "ups":
		return "UPSes"
	case "rack":
		return "racks"
	case "site":
		return "sites"
	default:
		return word + "s"
	}
}

func joinParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	// O(n) вместо O(n²): strings.Builder исключает переаллокации при конкатенации
	var b strings.Builder
	totalLen := len(parts) * 2 // резервируем под разделители
	for _, p := range parts {
		totalLen += len(p)
	}
	b.Grow(totalLen)

	for i, p := range parts {
		if i == len(parts)-1 {
			b.WriteString(" and ")
			b.WriteString(p)
		} else if i > 0 {
			b.WriteString(", ")
			b.WriteString(p)
		} else {
			b.WriteString(p)
		}
	}
	return b.String()
}
