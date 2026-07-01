// Package api — GraphQL read-only endpoint (INT-13.2.4).
//
// POST /api/v1/graphql — read-only GraphQL endpoint.
//
// Поддерживает запросы:
//   - devices { id name status device_type site_id }
//   - device(id: "xxx") { id name status }
//   - workOrders(limit: 10, status: "IN_PROGRESS") { id title status }
//   - workOrder(id: "xxx") { id title status priority }
//   - sites { id name status }
//   - technicians { user_id user_name skills }
//
// НЕ поддерживает мутации (read-only).
// Использует простой парсер без внешних зависимостей.
//
// Соответствует:
//   - OWASP ASVS V5 (Input validation — whitelist field names)
//   - OWASP ASVS V7 (Error handling — structured errors)
//   - IEC 62443 SR 7.1 (Resource availability — query limits)
//   - P1-HI-07: Query complexity analysis + max depth 5
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/models"
)

// ── Types ────────────────────────────────────────────────────────────

type graphQLRequest struct {
	Query string `json:"query"`
}

type graphQLResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []gqlError  `json:"errors,omitempty"`
}

type gqlError struct {
	Message string `json:"message"`
}

// ── Parsed Query ─────────────────────────────────────────────────────

type gqlQuery struct {
	Field        string
	Arguments    map[string]string
	Fields       []string // requested sub-fields
	Limit        int
	Complexity   int // P1-HI-07: вес запроса
	NestingDepth int // P1-HI-07: глубина вложенности
}

// P1-HI-07: Константы для анализа сложности запросов
const (
	// MaxQueryDepth — максимальная глубина вложенности GraphQL запроса.
	// Depth > 5 указывает на потенциальную атаку recursion/N+1.
	MaxQueryDepth = 5

	// MaxQueryComplexity — максимальный вес запроса.
	// Каждое поле = 1, каждый лимит в 10 элементов = +1 вес.
	MaxQueryComplexity = 50

	// BaseFieldComplexity — базовый вес одного поля.
	BaseFieldComplexity = 1

	// LimitComplexityMultiplier — множитель веса для пагинации.
	// Запрос с limit=100 будет иметь больший вес.
	LimitComplexityMultiplier = 0.1
)

// ── Handler ──────────────────────────────────────────────────────────

func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	var req graphQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeGraphQLError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Query == "" {
		writeGraphQLError(w, http.StatusBadRequest, "query is required")
		return
	}

	// Parse query
	queries := parseGQLQuery(req.Query)
	if len(queries) == 0 {
		writeGraphQLError(w, http.StatusBadRequest, "no valid queries found")
		return
	}

	// P1-HI-07: Query complexity analysis
	for _, q := range queries {
		if q.NestingDepth > MaxQueryDepth {
			writeGraphQLError(w, http.StatusBadRequest,
				fmt.Sprintf("query too deep: max depth is %d, got %d", MaxQueryDepth, q.NestingDepth))
			return
		}
		if q.Complexity > MaxQueryComplexity {
			writeGraphQLError(w, http.StatusBadRequest,
				fmt.Sprintf("query too complex: max complexity is %d, got %d", MaxQueryComplexity, q.Complexity))
			return
		}
	}

	// Execute queries
	result := make(map[string]interface{})
	var errs []gqlError

	for _, q := range queries {
		data, err := s.executeGQLQuery(r, q)
		if err != nil {
			errs = append(errs, gqlError{Message: err.Error()})
			continue
		}
		result[q.Field] = data
	}

	resp := graphQLResponse{Data: result}
	if len(errs) > 0 {
		resp.Errors = errs
	}

	jsonResponse(w, http.StatusOK, resp)
}

// ── Query Execution ──────────────────────────────────────────────────

func (s *Server) executeGQLQuery(r *http.Request, q gqlQuery) (interface{}, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	switch q.Field {
	case "device":
		return s.gqlGetDevice(r, q.Arguments["id"])
	case "devices":
		return s.gqlListDevices(r, limit, q.Fields)
	case "workOrder":
		return s.gqlGetWorkOrder(r, q.Arguments["id"])
	case "workOrders":
		return s.gqlListWorkOrders(r, limit, q.Arguments["status"], q.Fields)
	case "site":
		return s.gqlGetSite(r, q.Arguments["id"])
	case "sites":
		return s.gqlListSites(r, limit)
	case "technicians":
		return s.gqlListTechnicians(r, limit)
	case "technician":
		return s.gqlGetTechnician(r, q.Arguments["id"])
	default:
		return nil, fmt.Errorf("unknown field: %s", q.Field)
	}
}

// ── Resolvers ────────────────────────────────────────────────────────

func (s *Server) gqlGetDevice(r *http.Request, id string) (interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("device id is required")
	}
	device, err := s.deviceService.GetDevice(r.Context(), "", "", id)
	if err != nil {
		return nil, fmt.Errorf("device not found: %s", id)
	}
	return map[string]interface{}{
		"id":          device.DeviceID,
		"name":        device.Name,
		"status":      device.Status,
		"device_type": device.DeviceType,
		"site_id":     device.SiteID,
	}, nil
}

func (s *Server) gqlListDevices(r *http.Request, limit int, fields []string) (interface{}, error) {
	claims := auth.GetClaims(r)
	if claims == nil {
		return nil, fmt.Errorf("authentication required")
	}
	devices, err := s.deviceService.ListDevices(r.Context(), claims.UserID, claims.Role, models.ListDevicesFilter{PageSize: limit, Page: 1})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0)
	for _, d := range devices.Devices {
		item := map[string]interface{}{
			"id":          d.DeviceID,
			"name":        d.Name,
			"status":      string(d.Status),
			"device_type": string(d.DeviceType),
		}
		if gqlHasField(fields, "site_id") && d.SiteID != nil {
			item["site_id"] = *d.SiteID
		}
		if gqlHasField(fields, "last_seen") {
			item["last_seen"] = d.LastSeen.Format(time.RFC3339)
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Server) gqlGetWorkOrder(r *http.Request, id string) (interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("work order id is required")
	}
	wo, err := s.cmmsRouter.GetWorkOrder(r.Context(), id)
	if err != nil {
		return nil, fmt.Errorf("work order not found: %s", id)
	}
	return map[string]interface{}{
		"id":         wo.ID,
		"title":      wo.Title,
		"status":     wo.Status,
		"priority":   wo.Priority,
		"device_id":  wo.DeviceID,
		"created_at": wo.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) gqlListWorkOrders(r *http.Request, limit int, status string, fields []string) (interface{}, error) {
	filters := map[string]interface{}{"limit": limit}
	if status != "" {
		filters["status"] = status
	}
	wos, err := s.cmmsRouter.GetWorkOrders(r.Context(), filters)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(wos))
	for _, wo := range wos {
		item := map[string]interface{}{
			"id":        wo.ID,
			"title":     wo.Title,
			"status":    wo.Status,
			"priority":  wo.Priority,
			"device_id": wo.DeviceID,
		}
		if gqlHasField(fields, "assignee") {
			item["assignee"] = wo.AssigneeName
		}
		if gqlHasField(fields, "sla_status") {
			item["sla_status"] = wo.SLAStatus
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *Server) gqlGetSite(r *http.Request, id string) (interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("site id is required")
	}
	site, err := s.cmmsRouter.GetSite(r.Context(), id)
	if err != nil {
		return nil, fmt.Errorf("site not found: %s", id)
	}
	return map[string]interface{}{
		"id":     site.ID,
		"name":   site.Name,
		"status": site.Status,
	}, nil
}

func (s *Server) gqlListSites(r *http.Request, limit int) (interface{}, error) {
	sites, err := s.cmmsRouter.GetSites(r.Context(), map[string]interface{}{"limit": limit})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(sites))
	for _, site := range sites {
		result = append(result, map[string]interface{}{
			"id":     site.ID,
			"name":   site.Name,
			"status": site.Status,
		})
	}
	return result, nil
}

func (s *Server) gqlListTechnicians(r *http.Request, limit int) (interface{}, error) {
	techs, err := s.cmmsRouter.GetAllTechnicianWorkloads(r.Context())
	if err != nil {
		return nil, err
	}
	result := make([]map[string]interface{}, 0, len(techs))
	for _, t := range techs {
		result = append(result, map[string]interface{}{
			"user_id":   t.UserID,
			"user_name": t.UserName,
			"skills":    t.Skills,
			"location":  t.BaseLocation,
		})
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *Server) gqlGetTechnician(r *http.Request, id string) (interface{}, error) {
	if id == "" {
		return nil, fmt.Errorf("technician id is required")
	}
	t, err := s.cmmsRouter.GetTechnicianWorkload(r.Context(), id)
	if err != nil {
		return nil, fmt.Errorf("technician not found: %s", id)
	}
	return map[string]interface{}{
		"user_id":   t.UserID,
		"user_name": t.UserName,
		"skills":    t.Skills,
		"location":  t.BaseLocation,
	}, nil
}

// ── Query Parser ─────────────────────────────────────────────────────

func parseGQLQuery(query string) []gqlQuery {
	var queries []gqlQuery

	// Remove newlines and extra spaces
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")

	// Find all top-level queries: fieldName(args) { fields }
	for {
		query = strings.TrimSpace(query)
		if query == "" {
			break
		}

		// Read field name
		end := strings.IndexAny(query, "( {")
		if end < 0 {
			break
		}

		field := strings.TrimSpace(query[:end])
		query = strings.TrimSpace(query[end:])

		q := gqlQuery{Field: field, Limit: 20}

		// Parse arguments: (arg1: "val1", arg2: "val2")
		if strings.HasPrefix(query, "(") {
			closeParen := indexOfMatching(query, '(', ')')
			if closeParen < 0 {
				break
			}
			argsStr := query[1:closeParen]
			q.Arguments = parseGQLArgs(argsStr)

			// Parse limit
			if l, ok := q.Arguments["limit"]; ok {
				fmt.Sscanf(l, "%d", &q.Limit)
			}

			query = strings.TrimSpace(query[closeParen+1:])
		}

		// Parse fields: { field1 field2 }
		if strings.HasPrefix(query, "{") {
			closeBrace := indexOfMatching(query, '{', '}')
			if closeBrace < 0 {
				break
			}
			fieldsStr := query[1:closeBrace]
			q.Fields = strings.Fields(fieldsStr)

			// P1-HI-07: Calculate nesting depth based on nested braces
			q.NestingDepth = calculateNestingDepth(fieldsStr)

			query = strings.TrimSpace(query[closeBrace+1:])
		}

		// P1-HI-07: Calculate total query complexity
		q.Complexity = calculateQueryComplexity(&q)

		queries = append(queries, q)

		// Handle comma or end
		if strings.HasPrefix(query, ",") {
			query = strings.TrimSpace(query[1:])
		}
	}

	return queries
}

// P1-HI-07: calculateNestingDepth вычисляет максимальную глубину вложенности
// полей в GraphQL запросе. Считает уровень по фигурным скобкам {}.
func calculateNestingDepth(fieldsStr string) int {
	maxDepth := 0
	currentDepth := 0
	for _, ch := range fieldsStr {
		switch ch {
		case '{':
			currentDepth++
			if currentDepth > maxDepth {
				maxDepth = currentDepth
			}
		case '}':
			if currentDepth > 0 {
				currentDepth--
			}
		}
	}
	return maxDepth
}

// P1-HI-07: calculateQueryComplexity вычисляет общий вес запроса.
//
// Формула: BaseFieldComplexity * len(Fields) + LimitComplexityMultiplier * limit
// Вес полей списка умножается на лимит пагинации, так как каждое поле
// будет возвращено для каждого элемента списка.
//
// Compliance:
//   - IEC 62443 SR 7.1: Resource availability — предотвращение DoS через сложные запросы
//   - OWASP ASVS V5.3.1: Input validation — ограничение сложности запросов
func calculateQueryComplexity(q *gqlQuery) int {
	fieldCount := len(q.Fields)
	if fieldCount == 0 {
		fieldCount = 1 // Минимум 1 поле (сам запрос)
	}

	// Базовая сложность: количество запрашиваемых полей
	complexity := BaseFieldComplexity * fieldCount

	// P1-HI-07: Добавляем вес пагинации (поле * лимит)
	// Запрос с limit=100 возвращает до 100 записей → высокая стоимость
	if q.Limit > 0 {
		complexity += int(float64(fieldCount) * float64(q.Limit) * LimitComplexityMultiplier)
	}

	// Добавляем вес вложенности (глубина * 2)
	complexity += q.NestingDepth * 2

	return complexity
}

func parseGQLArgs(argsStr string) map[string]string {
	args := make(map[string]string)
	argsStr = strings.TrimSpace(argsStr)
	if argsStr == "" {
		return args
	}

	pairs := splitArgs(argsStr)
	for _, pair := range pairs {
		eq := strings.IndexByte(pair, ':')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:eq])
		val := strings.TrimSpace(pair[eq+1:])
		val = strings.Trim(val, "\"")
		args[key] = val
	}
	return args
}

func splitArgs(s string) []string {
	var result []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				result = append(result, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

func indexOfMatching(s string, open, close byte) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func gqlHasField(arr []string, s string) bool {
	for _, a := range arr {
		if a == s {
			return true
		}
	}
	return false
}

// ── Helpers ──────────────────────────────────────────────────────────

func writeGraphQLError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(graphQLResponse{
		Errors: []gqlError{{Message: message}},
	})
}
