// Package rca — Graph Builder для RCA Engine (CCTV-2.1.3).
//
// Предоставляет:
//   - BuildFromState: построение графа из явных parent-child отношений
//   - Incremental updates: добавление/удаление/обновление узлов
//   - Event listener: реакция на изменения устройств
//   - Validation: проверка циклов, корректности parentID, обязательных полей
//   - Cache invalidation: автоматическая очистка кэша RCAEngine
//   - WebSocket notification: оповещение подписчиков об изменениях
//
// Compliance:
//   - CCTV Core IP (уникальная фича)
//   - IEC 62443 SR 7.1 (Resource availability — incremental updates)
//   - ISO 27001 A.12.6.1 (Capacity management — topology validation)
//   - ISO 27019 PCC.A.12.6 (ICS topology management)
package rca

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

// DeviceState — состояние устройства для построения графа.
type DeviceState struct {
	ID       string       `json:"id" validate:"required"`
	Name     string       `json:"name" validate:"required"`
	Type     DeviceType   `json:"type" validate:"required"`
	Status   DeviceStatus `json:"status"`
	Location string       `json:"location,omitempty"`
	SiteID   string       `json:"site_id,omitempty"`
	ParentID string       `json:"parent_id,omitempty"` // ID родительского узла (пусто = корень)
}

// TopologyConfig — конфигурация топологии для ручного построения.
type TopologyConfig struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description,omitempty" yaml:"description"`
	Devices     []DeviceState  `json:"devices" yaml:"devices"`
	Links       []TopologyLink `json:"links,omitempty" yaml:"links"`
}

// TopologyLink — явная связь между устройствами.
// Используется, когда parent-child не выражен через ParentID.
type TopologyLink struct {
	ParentID string `json:"parent_id" yaml:"parent_id" validate:"required"`
	ChildID  string `json:"child_id" yaml:"child_id" validate:"required"`
	Relation string `json:"relation,omitempty" yaml:"relation"` // "uplink", "downlink", "redundant"
}

// GraphChangeType — тип изменения графа.
type GraphChangeType string

const (
	ChangeNodeAdded    GraphChangeType = "node_added"
	ChangeNodeUpdated  GraphChangeType = "node_updated"
	ChangeNodeRemoved  GraphChangeType = "node_removed"
	ChangeLinkAdded    GraphChangeType = "link_added"
	ChangeLinkRemoved  GraphChangeType = "link_removed"
	ChangeGraphRebuilt GraphChangeType = "graph_rebuilt"
)

// GraphChangeEvent — событие изменения графа.
type GraphChangeEvent struct {
	Type      GraphChangeType `json:"type"`
	NodeID    string          `json:"node_id,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Details   string          `json:"details,omitempty"`
}

// GraphChangeHandler — callback для уведомления об изменениях графа.
// Может использоваться для WebSocket рассылки.
type GraphChangeHandler func(event GraphChangeEvent)

// ValidationError — ошибка валидации топологии.
type ValidationError struct {
	Message string
	Errors  []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("topology validation: %s (%d errors)", e.Message, len(e.Errors))
}

// ═══════════════════════════════════════════════════════════════════════
// GraphBuilder
// ═══════════════════════════════════════════════════════════════════════

// GraphBuilderConfig — конфигурация GraphBuilder.
type GraphBuilderConfig struct {
	Logger           *slog.Logger
	ChangeHandler    GraphChangeHandler // опциональный callback для уведомлений
	AutoInvalidate   bool               // автоматически инвалидировать кэш RCAEngine при изменениях
	StrictValidation bool               // строгая валидация (ошибка = не строим граф)
}

// GraphBuilder строит и обновляет DeviceGraph из различных источников.
type GraphBuilder struct {
	graph  *DeviceGraph
	engine *RCAEngine // опциональная ссылка для cache invalidation
	cfg    GraphBuilderConfig
	logger *slog.Logger
	mu     sync.RWMutex

	// Индексы для быстрого поиска
	parentIndex   map[string]string   // childID → parentID
	childrenIndex map[string][]string // parentID → []childID

	// Счётчик версий графа (для отслеживания изменений)
	graphVersion int64

	// Auto-refresh (BACKEND.4)
	autoRefreshCancel context.CancelFunc
}

// NewGraphBuilder создаёт GraphBuilder.
func NewGraphBuilder(graph *DeviceGraph, engine *RCAEngine, cfg GraphBuilderConfig) *GraphBuilder {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &GraphBuilder{
		graph:         graph,
		engine:        engine,
		cfg:           cfg,
		logger:        cfg.Logger.With("component", "rca-graph-builder"),
		parentIndex:   make(map[string]string),
		childrenIndex: make(map[string][]string),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P1-4.5: BuildFromState — точное построение из parent-child отношений
// ═══════════════════════════════════════════════════════════════════════

// BuildFromState строит граф из массива DeviceState.
//
// Использует явные parent-child отношения (ParentID).
// Перед построением выполняет валидацию:
//   - Обязательные поля (ID, Name, Type)
//   - Отсутствие циклов
//   - Родитель существует (если ParentID указан)
//   - Нет дубликатов ID
//
// Возвращает количество добавленных узлов и список ошибок валидации.
func (b *GraphBuilder) BuildFromState(devices []DeviceState) (int, *ValidationError) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// ── Шаг 1: Валидация ────────────────────────────────────────────
	if err := b.validateDevices(devices); err != nil {
		return 0, err
	}

	// ── Шаг 2: Очищаем граф и индексы ───────────────────────────────
	// Очищаем граф напрямую (мы в том же пакете)
	b.graph.mu.Lock()
	b.graph.nodes = make(map[string]*DeviceNode)
	b.graph.roots = make([]string, 0)
	b.graph.mu.Unlock()
	b.parentIndex = make(map[string]string)
	b.childrenIndex = make(map[string][]string)

	// ── Шаг 3: Строим parentIndex ────────────────────────────────────
	for _, d := range devices {
		if d.ParentID != "" {
			b.parentIndex[d.ID] = d.ParentID
			b.childrenIndex[d.ParentID] = append(b.childrenIndex[d.ParentID], d.ID)
		}
	}

	// ── Шаг 4: Добавляем узлы в граф ─────────────────────────────────
	// Сначала корневые узлы (ParentID = ""), потом остальные
	added := 0

	// 4a: Корневые узлы
	for _, d := range devices {
		if d.ParentID == "" {
			b.graph.AddNode(d.ID, d.Name, d.Type, "")
			if node, ok := b.graph.GetNode(d.ID); ok {
				node.Status = d.Status
				node.Location = d.Location
				node.SiteID = d.SiteID
			}
			added++
		}
	}

	// 4b: Дочерние узлы (сортировка по глубине не требуется,
	//     т.к. DeviceGraph.AddNode поддерживает добавление в несуществующего parent)
	for _, d := range devices {
		if d.ParentID != "" {
			b.graph.AddNode(d.ID, d.Name, d.Type, d.ParentID)
			if node, ok := b.graph.GetNode(d.ID); ok {
				node.Status = d.Status
				node.Location = d.Location
				node.SiteID = d.SiteID
			}
			added++
		}
	}

	b.graphVersion++
	b.logger.Info("graph built from state",
		"devices", len(devices),
		"added", added,
		"version", b.graphVersion,
	)

	// ── Шаг 5: Инвалидация кэша ──────────────────────────────────────
	b.invalidateCache()

	// ── Шаг 6: Уведомление ───────────────────────────────────────────
	b.notifyChange(GraphChangeEvent{
		Type:      ChangeGraphRebuilt,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("graph rebuilt with %d devices", added),
	})

	return added, nil
}

// ═══════════════════════════════════════════════════════════════════════
// DeviceStateProvider — функция получения состояния устройств
// ═══════════════════════════════════════════════════════════════════════

// DeviceStateProvider — функция для получения текущего состояния устройств.
// Используется в StartAutoRefresh для периодического обновления графа.
type DeviceStateProvider func(ctx context.Context) ([]DeviceState, error)

// ═══════════════════════════════════════════════════════════════════════
// P1-4.6: Auto-Refresh (BACKEND.4)
// ═══════════════════════════════════════════════════════════════════════

// StartAutoRefresh запускает автоматическое обновление графа.
//
// provider вызывается с интервалом interval для получения текущего состояния
// устройств. Каждый вызов provider → BuildFromState для полной перестройки графа.
//
// Параметры:
//   - ctx: контекст для graceful shutdown (ctx.Done → остановка)
//   - interval: интервал между вызовами provider
//   - provider: функция получения DeviceState
//
// Возвращает канал ошибок (буферизованный, размер 10).
// Ошибки от provider и BuildFromState отправляются в errCh.
// Канал закрывается при остановке.
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — мониторинг топологии)
//   - ISO 27001 A.12.6.1 (Capacity management)
//   - ISO 27019 PCC.A.12.6 (ICS topology management)
func (b *GraphBuilder) StartAutoRefresh(ctx context.Context, interval time.Duration, provider DeviceStateProvider) <-chan error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Если уже запущен — останавливаем предыдущий
	if b.autoRefreshCancel != nil {
		b.autoRefreshCancel()
	}

	// Создаём дочерний контекст для управления жизненным циклом
	refreshCtx, cancel := context.WithCancel(ctx)
	b.autoRefreshCancel = cancel

	errCh := make(chan error, 10)
	go b.autoRefreshLoop(refreshCtx, interval, provider, errCh)

	b.logger.Info("auto-refresh started",
		"interval", interval,
	)
	return errCh
}

// StopAutoRefresh останавливает автоматическое обновление графа.
// Безопасен для многократного вызова (идемпотентен).
func (b *GraphBuilder) StopAutoRefresh() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.autoRefreshCancel != nil {
		b.autoRefreshCancel()
		b.autoRefreshCancel = nil
		b.logger.Info("auto-refresh stopped")
	}
}

// autoRefreshLoop — основной цикл автообновления.
// Запускается в отдельной горутине через StartAutoRefresh.
func (b *GraphBuilder) autoRefreshLoop(ctx context.Context, interval time.Duration, provider DeviceStateProvider, errCh chan<- error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Немедленный первый рефреш
	b.refreshFromProvider(ctx, provider, errCh)
	if ctx.Err() != nil {
		close(errCh)
		return
	}

	for {
		select {
		case <-ctx.Done():
			close(errCh)
			return
		case <-ticker.C:
			b.refreshFromProvider(ctx, provider, errCh)
			if ctx.Err() != nil {
				close(errCh)
				return
			}
		}
	}
}

// refreshFromProvider вызывает provider и строит граф через BuildFromState.
func (b *GraphBuilder) refreshFromProvider(ctx context.Context, provider DeviceStateProvider, errCh chan<- error) {
	if ctx.Err() != nil {
		return
	}

	devices, err := provider(ctx)
	if err != nil {
		select {
		case errCh <- fmt.Errorf("auto-refresh provider: %w", err):
		default:
			b.logger.Warn("auto-refresh error channel full, dropping provider error")
		}
		return
	}

	if _, err := b.BuildFromState(devices); err != nil {
		select {
		case errCh <- fmt.Errorf("auto-refresh build: %w", err):
		default:
			b.logger.Warn("auto-refresh error channel full, dropping build error")
		}
	}
}

// BuildFromTopology строит граф из TopologyConfig.
// Поддерживает как parent-child отношения, так и явные связи (Links).
func (b *GraphBuilder) BuildFromTopology(cfg TopologyConfig) (int, *ValidationError) {
	// Собираем все DeviceState
	devices := cfg.Devices

	// Добавляем явные связи из Links
	linkIndex := make(map[string]bool) // childID → already linked
	for _, link := range cfg.Links {
		if link.ChildID == "" || link.ParentID == "" {
			continue
		}
		// Ищем DeviceState для child и устанавливаем ParentID
		for i, d := range devices {
			if d.ID == link.ChildID {
				devices[i].ParentID = link.ParentID
				linkIndex[link.ChildID] = true
				break
			}
		}
	}

	return b.BuildFromState(devices)
}

// ═══════════════════════════════════════════════════════════════════════
// P1-4.4: Incremental Updates
// ═══════════════════════════════════════════════════════════════════════

// AddNode добавляет один узел в граф.
// Если узел уже существует — обновляет его.
func (b *GraphBuilder) AddNode(state DeviceState) error {
	if state.ID == "" {
		return fmt.Errorf("graph_builder: device ID is required")
	}
	if state.Name == "" {
		return fmt.Errorf("graph_builder: device %q name is required", state.ID)
	}
	if state.Type == "" {
		return fmt.Errorf("graph_builder: device %q type is required", state.ID)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Проверка на цикл
	if state.ParentID != "" {
		if b.wouldCreateCycle(state.ID, state.ParentID) {
			return fmt.Errorf("graph_builder: adding %q as child of %q would create a cycle", state.ID, state.ParentID)
		}
	}

	existing, exists := b.graph.GetNode(state.ID)
	if exists {
		// Обновление существующего узла
		oldParent := existing.ParentID
		b.graph.AddNode(state.ID, state.Name, state.Type, state.ParentID)
		if node, ok := b.graph.GetNode(state.ID); ok {
			node.Status = state.Status
			node.Location = state.Location
			node.SiteID = state.SiteID
		}

		// Обновляем индексы при смене родителя
		if oldParent != state.ParentID {
			b.removeFromIndex(oldParent, state.ID)
			if state.ParentID != "" {
				b.parentIndex[state.ID] = state.ParentID
				b.childrenIndex[state.ParentID] = append(b.childrenIndex[state.ParentID], state.ID)
			} else {
				delete(b.parentIndex, state.ID)
			}
		}

		b.graphVersion++
		b.invalidateCache()
		b.notifyChange(GraphChangeEvent{
			Type:      ChangeNodeUpdated,
			NodeID:    state.ID,
			Timestamp: time.Now(),
			Details:   fmt.Sprintf("node %q updated (type=%s, status=%s)", state.Name, state.Type, state.Status),
		})

		b.logger.Debug("node updated", "id", state.ID, "name", state.Name)
		return nil
	}

	// Добавление нового узла
	b.graph.AddNode(state.ID, state.Name, state.Type, state.ParentID)
	if node, ok := b.graph.GetNode(state.ID); ok {
		node.Status = state.Status
		node.Location = state.Location
		node.SiteID = state.SiteID
	}

	if state.ParentID != "" {
		b.parentIndex[state.ID] = state.ParentID
		b.childrenIndex[state.ParentID] = append(b.childrenIndex[state.ParentID], state.ID)
	}

	b.graphVersion++
	b.invalidateCache()
	b.notifyChange(GraphChangeEvent{
		Type:      ChangeNodeAdded,
		NodeID:    state.ID,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("node %q added (type=%s, parent=%s)", state.Name, state.Type, state.ParentID),
	})

	b.logger.Debug("node added", "id", state.ID, "name", state.Name, "parent", state.ParentID)
	return nil
}

// RemoveNode удаляет узел из графа.
// Если removeChildren = true — удаляет всех потомков рекурсивно.
func (b *GraphBuilder) RemoveNode(deviceID string, removeChildren bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	node, ok := b.graph.GetNode(deviceID)
	if !ok {
		return fmt.Errorf("graph_builder: node %q not found", deviceID)
	}

	// Если нужно — рекурсивно удаляем всех потомков
	if removeChildren {
		descendants := b.graph.GetAllDescendants(deviceID)
		for _, desc := range descendants {
			b.graph.mu.Lock()
			delete(b.graph.nodes, desc.ID)
			b.graph.mu.Unlock()
			delete(b.parentIndex, desc.ID)
			// Удаляем из childrenIndex родителя
			if parentID, ok := b.parentIndex[desc.ID]; ok {
				b.removeFromIndex(parentID, desc.ID)
			}
		}
	}

	// Удаляем сам узел
	b.graph.mu.Lock()
	delete(b.graph.nodes, deviceID)
	b.graph.mu.Unlock()

	// Удаляем из roots если был корнем
	for i, root := range b.graph.roots {
		if root == deviceID {
			b.graph.roots = append(b.graph.roots[:i], b.graph.roots[i+1:]...)
			break
		}
	}

	delete(b.parentIndex, deviceID)
	b.removeFromIndex(node.ParentID, deviceID)

	// Удаляем из childrenIndex других узлов
	delete(b.childrenIndex, deviceID)

	b.graphVersion++
	b.invalidateCache()
	b.notifyChange(GraphChangeEvent{
		Type:      ChangeNodeRemoved,
		NodeID:    deviceID,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("node %q removed (children=%v)", node.Name, removeChildren),
	})

	b.logger.Info("node removed", "id", deviceID, "name", node.Name)
	return nil
}

// UpdateNodeStatus обновляет статус устройства. Если статус изменился —
// инвалидирует кэш и уведомляет подписчиков.
func (b *GraphBuilder) UpdateNodeStatus(deviceID string, status DeviceStatus) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	node, ok := b.graph.GetNode(deviceID)
	if !ok {
		return false
	}

	if node.Status == status {
		return false // статус не изменился
	}

	oldStatus := node.Status
	node.Status = status

	b.graphVersion++
	b.invalidateCache()
	b.notifyChange(GraphChangeEvent{
		Type:      ChangeNodeUpdated,
		NodeID:    deviceID,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("status changed: %s → %s", oldStatus, status),
	})

	b.logger.Debug("node status updated",
		"id", deviceID,
		"old_status", oldStatus,
		"new_status", status,
	)
	return true
}

// ═══════════════════════════════════════════════════════════════════════
// Event Listener (P1-4.4)
// ═══════════════════════════════════════════════════════════════════════

// OnDeviceChange подписывает callback на изменения устройств.
// Возвращает функцию для отписки.
func (b *GraphBuilder) OnDeviceChange(handler GraphChangeHandler) func() {
	b.mu.Lock()
	defer b.mu.Unlock()

	oldHandler := b.cfg.ChangeHandler
	b.cfg.ChangeHandler = func(event GraphChangeEvent) {
		if oldHandler != nil {
			oldHandler(event)
		}
		handler(event)
	}

	// Возвращаем функцию отписки
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		b.cfg.ChangeHandler = oldHandler
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Cache Invalidation
// ═══════════════════════════════════════════════════════════════════════

// InvalidateCache очищает кэш RCAEngine для указанного устройства.
func (b *GraphBuilder) InvalidateCache(deviceID string) {
	if b.engine != nil {
		b.engine.InvalidateCache(deviceID)
	}
}

// InvalidateAllCache очищает весь кэш RCAEngine.
func (b *GraphBuilder) InvalidateAllCache() {
	if b.engine != nil {
		b.engine.ClearCache()
	}
}

// GraphVersion возвращает текущую версию графа.
func (b *GraphBuilder) GraphVersion() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.graphVersion
}

// ═══════════════════════════════════════════════════════════════════════
// Validation (P1-4.5)
// ═══════════════════════════════════════════════════════════════════════

// validateDevices проверяет массив устройств на корректность.
func (b *GraphBuilder) validateDevices(devices []DeviceState) *ValidationError {
	var errs []string
	idSet := make(map[string]bool)

	for _, d := range devices {
		// Обязательные поля
		if d.ID == "" {
			errs = append(errs, "device with empty ID")
			continue
		}
		if d.Name == "" {
			errs = append(errs, fmt.Sprintf("device %q: name is required", d.ID))
		}
		if d.Type == "" {
			errs = append(errs, fmt.Sprintf("device %q: type is required", d.ID))
		}
		if _, ok := validDeviceTypes[d.Type]; !ok && d.Type != "" {
			errs = append(errs, fmt.Sprintf("device %q: invalid type %q", d.ID, d.Type))
		}

		// Дубликаты ID
		if idSet[d.ID] {
			errs = append(errs, fmt.Sprintf("duplicate device ID: %q", d.ID))
		}
		idSet[d.ID] = true
	}

	// Проверка: родитель существует
	for _, d := range devices {
		if d.ParentID != "" && !idSet[d.ParentID] {
			errs = append(errs, fmt.Sprintf("device %q: parent %q not found in devices", d.ID, d.ParentID))
		}
	}

	// Проверка циклов
	visited := make(map[string]bool)
	for _, d := range devices {
		if !visited[d.ID] {
			if hasCycle(d.ID, devices, visited, make(map[string]bool)) {
				errs = append(errs, fmt.Sprintf("cycle detected involving device %q", d.ID))
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return &ValidationError{
		Message: fmt.Sprintf("%d validation errors", len(errs)),
		Errors:  errs,
	}
}

// wouldCreateCycle проверяет, создаст ли добавление parentID цикл.
func (b *GraphBuilder) wouldCreateCycle(childID, parentID string) bool {
	// BFS от parentID вверх: если встретим childID — цикл
	current := parentID
	for current != "" {
		if current == childID {
			return true
		}
		if node, ok := b.graph.GetNode(current); ok {
			current = node.ParentID
		} else {
			current = b.parentIndex[current]
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

func (b *GraphBuilder) invalidateCache() {
	if b.cfg.AutoInvalidate && b.engine != nil {
		b.engine.ClearCache()
	}
}

func (b *GraphBuilder) notifyChange(event GraphChangeEvent) {
	if b.cfg.ChangeHandler != nil {
		b.cfg.ChangeHandler(event)
	}
}

func (b *GraphBuilder) removeFromIndex(parentID, childID string) {
	if parentID == "" {
		return
	}
	children := b.childrenIndex[parentID]
	for i, id := range children {
		if id == childID {
			b.childrenIndex[parentID] = append(children[:i], children[i+1:]...)
			break
		}
	}
}

// validDeviceTypes — множество допустимых типов устройств.
var validDeviceTypes = map[DeviceType]bool{
	DeviceTypeSite:    true,
	DeviceTypeSwitch:  true,
	DeviceTypeNVR:     true,
	DeviceTypeDVR:     true,
	DeviceTypeCamera:  true,
	DeviceTypeEncoder: true,
	DeviceTypeServer:  true,
	DeviceTypeUPS:     true,
	DeviceTypeRack:    true,
}

// hasCycle — DFS проверка цикла в графе parent-child.
func hasCycle(nodeID string, devices []DeviceState, visited, path map[string]bool) bool {
	visited[nodeID] = true
	path[nodeID] = true

	// Находим ParentID для nodeID
	var parentID string
	for _, d := range devices {
		if d.ID == nodeID {
			parentID = d.ParentID
			break
		}
	}

	if parentID != "" {
		if path[parentID] {
			return true // найден цикл
		}
		if !visited[parentID] {
			if hasCycle(parentID, devices, visited, path) {
				return true
			}
		}
	}

	path[nodeID] = false
	return false
}
