// Package api — RCA Visualization handler (AI-01).
//
// GET /api/v1/rca/{device_id} — возвращает граф для react-flow визуализации.
//
// Соответствует:
//   - AI-01: RCA Визуализация графа
//   - OWASP ASVS V4 (RBAC)
//   - OWASP ASVS V7 (Error handling)
//   - OWASP ASVS V8 (Data Protection)
package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/rca"
)

// ── API Response Types ────────────────────────────────────────────────

// RCAGraphResponse — ответ для react-flow визуализации.
type RCAGraphResponse struct {
	// Nodes — все узлы графа (для react-flow)
	Nodes []RCAGraphNode `json:"nodes"`
	// Edges — связи между узлами (для react-flow)
	Edges []RCAGraphEdge `json:"edges"`
	// RootCauseID — ID первопричины (для подсветки)
	RootCauseID string `json:"root_cause_id"`
	// FailedDeviceID — ID устройства с которого пришёл alarm
	FailedDeviceID string `json:"failed_device_id"`
	// ImpactDescription — человекочитаемое описание
	ImpactDescription string `json:"impact_description"`
	// Recommendation — рекомендация
	Recommendation string `json:"recommendation"`
	// BlastRadius — количество затронутых устройств
	BlastRadius int `json:"blast_radius"`
}

// RCAGraphNode — узел графа для react-flow.
type RCAGraphNode struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"`  // "rcaDevice" — кастомная нода
	Data  RCAGraphNodeData `json:"data"`
	Position RCAGraphPosition `json:"position"`
}

// RCAGraphNodeData — данные узла.
type RCAGraphNodeData struct {
	Label    string `json:"label"`
	DeviceType string `json:"device_type"`
	Status   string `json:"status"`
	IsRootCause bool `json:"is_root_cause"`
	IsFailed  bool   `json:"is_failed"`
	IsHealthy bool   `json:"is_healthy"`
}

// RCAGraphPosition — позиция узла на графе.
type RCAGraphPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// RCAGraphEdge — связь между узлами.
type RCAGraphEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`     // "smoothstep"
	Animated bool   `json:"animated"`
	Label    string `json:"label,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// RCA Visualization Handler
// ═══════════════════════════════════════════════════════════════════════

// handleRCAGraph возвращает данные графа для react-flow визуализации.
//
// GET /api/v1/rca/{device_id}
//
// Алгоритм:
//  1. Анализирует устройство через RCA Engine
//  2. Строит граф: root cause + путь + affected devices
//  3. Возвращает nodes + edges в формате react-flow
//
// AI-01: RCA Визуализация графа — киллер-фича для диспетчеров
func (s *Server) handleRCAGraph(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	if claims == nil {
		respondError(w, r, NewUnauthorizedError("authentication required"))
		return
	}

	deviceID := chi.URLParam(r, "id")
	if deviceID == "" {
		respondError(w, r, NewBadRequestError("device_id is required"))
		return
	}

	// ── Проверяем кэш ──
	cached, ok := s.rcaEngine.GetCachedResult(deviceID)
	var result *rca.RCAResult

	if ok && cached != nil {
		result = cached
	} else {
		// Получаем устройство из БД для инициации RCA
		device, err := s.deviceService.GetDevice(r.Context(), claims.UserID, claims.Role, deviceID)
		if err != nil {
			respondError(w, r, NewNotFoundError("device not found"))
			return
		}

		// Анализируем через RCA Engine
		event := rca.RCAEvent{
			DeviceID:  deviceID,
			EventType: "analysis",
			Severity:  "high",
		}

		// Добавляем устройство в граф если его там нет
		if _, ok := s.rcaEngine.GetGraph().GetNode(deviceID); !ok {
			parentID := ""
			if device.ParentDeviceID != nil && *device.ParentDeviceID != "" && *device.ParentDeviceID != deviceID {
				parentID = *device.ParentDeviceID
			}
			s.rcaEngine.GetGraph().AddNode(deviceID, device.Name,
				rca.DeviceType(device.DeviceType),
				parentID,
			)
			s.rcaEngine.GetGraph().UpdateStatus(deviceID, rca.DeviceStatus(device.Status))
		}

		result = s.rcaEngine.Analyze(event)
	}

	if result == nil {
		respondError(w, r, NewInternalError("rca analysis failed", nil))
		return
	}

	// ── Строим react-flow граф ──
	graph := buildRCAGraph(result)

	jsonResponse(w, http.StatusOK, graph)
}

// buildRCAGraph строит react-flow граф из RCA результата.
func buildRCAGraph(result *rca.RCAResult) *RCAGraphResponse {
	nodes := make([]RCAGraphNode, 0)
	edges := make([]RCAGraphEdge, 0)
	added := make(map[string]bool)

	// ── Уровни для позиционирования ──
	// Корень (root cause) и affected выстраиваются сверху вниз
	// root → children → grandchildren
	type levelNode struct {
		node   *rca.DeviceNode
		level  int
		offset int
	}

	// Собираем все affected устройства
	allDevices := make(map[string]*rca.DeviceNode)

	// Root cause
	if result.RootCause != nil {
		allDevices[result.RootCause.ID] = result.RootCause
	}

	// Failed device (если отличается от root cause)
	if result.FailedDevice != nil {
		allDevices[result.FailedDevice.ID] = result.FailedDevice
	}

	// Affected devices
	for _, d := range result.AffectedDevices {
		allDevices[d.ID] = d
	}

	// ── Строим уровни (BFS от root cause) ──
	levels := make([][]*rca.DeviceNode, 0)
	if result.RootCause != nil {
		queue := []*rca.DeviceNode{result.RootCause}
		visited := make(map[string]bool)
		visited[result.RootCause.ID] = true

		for len(queue) > 0 {
			levelSize := len(queue)
			level := make([]*rca.DeviceNode, 0)

			for i := 0; i < levelSize; i++ {
				node := queue[0]
				queue = queue[1:]
				level = append(level, node)

				// Добавляем детей
				for _, childID := range node.Children {
					if !visited[childID] {
						if child, ok := allDevices[childID]; ok {
							visited[childID] = true
							queue = append(queue, child)
						}
					}
				}
			}

			if len(level) > 0 {
				levels = append(levels, level)
			}
		}
	}

	// Если уровней нет (одиночное устройство) — создаём один уровень
	if len(levels) == 0 && result.FailedDevice != nil {
		levels = [][]*rca.DeviceNode{{result.FailedDevice}}
	}

	// ── Позиционирование узлов ──
	const (
		levelHeight = 120
		nodeWidth   = 200
		horizMargin = 40
		vertMargin  = 60
	)

	for levelIdx, level := range levels {
		startX := float64(len(level)) * (nodeWidth+horizMargin) / -2
		for nodeIdx, node := range level {
			x := startX + float64(nodeIdx)*(nodeWidth+horizMargin)
			y := float64(levelIdx) * (levelHeight + vertMargin)

			isRootCause := result.RootCause != nil && node.ID == result.RootCause.ID
			isFailed := result.FailedDevice != nil && node.ID == result.FailedDevice.ID
			isHealthy := node.Status == rca.StatusOnline

			nodes = append(nodes, RCAGraphNode{
				ID:   node.ID,
				Type: "rcaDevice",
				Data: RCAGraphNodeData{
					Label:       node.Name,
					DeviceType:  string(node.Type),
					Status:      string(node.Status),
					IsRootCause: isRootCause,
					IsFailed:    isFailed && !isRootCause,
					IsHealthy:   isHealthy,
				},
				Position: RCAGraphPosition{X: x, Y: y},
			})
			added[node.ID] = true

			// ── Edge от родителя ──
			if node.ParentID != "" && added[node.ParentID] {
				edges = append(edges, RCAGraphEdge{
					ID:       "e-" + node.ParentID + "-" + node.ID,
					Source:   node.ParentID,
					Target:   node.ID,
					Type:     "smoothstep",
					Animated: isRootCause,
					Label:    "",
				})
			}
		}
	}

	// Если устройство не было добавлено ни на один уровень — добавляем отдельно
	if result.FailedDevice != nil && !added[result.FailedDevice.ID] {
		nodes = append(nodes, RCAGraphNode{
			ID:   result.FailedDevice.ID,
			Type: "rcaDevice",
			Data: RCAGraphNodeData{
				Label:    result.FailedDevice.Name,
				DeviceType: string(result.FailedDevice.Type),
				Status:   string(result.FailedDevice.Status),
				IsFailed: true,
			},
			Position: RCAGraphPosition{X: 0, Y: float64(len(levels)) * (levelHeight + vertMargin)},
		})
	}

	// Recommendation
	recommendation := buildRecommendation(result)

	return &RCAGraphResponse{
		Nodes:             nodes,
		Edges:             edges,
		RootCauseID:       getID(result.RootCause),
		FailedDeviceID:    getID(result.FailedDevice),
		ImpactDescription: result.ImpactDescription,
		Recommendation:    recommendation,
		BlastRadius:       result.BlastRadius,
	}
}

func getID(node *rca.DeviceNode) string {
	if node == nil {
		return ""
	}
	return node.ID
}

// buildRecommendation формирует рекомендацию на основе RCA.
func buildRecommendation(result *rca.RCAResult) string {
	if result.RootCause == nil {
		return "Не удалось определить причину. Проверьте устройство вручную."
	}

	if result.IsRootCause {
		return fmt.Sprintf("Проблема в самом устройстве %s (%s). Проверьте подключение и питание.",
			result.RootCause.Name, result.RootCause.Status)
	}

	// Проблема в parent-устройстве
	rec := fmt.Sprintf("Проверьте %s (%s) — это первопричина. Затронуто %d устройств.",
		result.RootCause.Name, result.RootCause.Status, result.BlastRadius)

	if result.BlastRadiusByType != nil {
		if cams, ok := result.BlastRadiusByType["camera"]; ok && cams > 0 {
			rec += fmt.Sprintf(" Из них %d камер.", cams)
		}
	}

	return rec
}
