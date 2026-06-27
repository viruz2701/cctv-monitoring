// Package api — OpenAPI 3.1 spec generator (INT-13.2.1).
//
// Auto-generates OpenAPI 3.1 specification from registered routes.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// OpenAPI 3.1 types
// ═══════════════════════════════════════════════════════════════════════

type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"` // "3.1.0"
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers,omitempty"`
	Paths      map[string]OpenAPIPath `json:"paths"`
	Components OpenAPIComponents      `json:"components,omitempty"`
	Tags       []OpenAPITag           `json:"tags,omitempty"`
}

type OpenAPIInfo struct {
	Title       string          `json:"title"`
	Version     string          `json:"version"`
	Description string          `json:"description,omitempty"`
	Contact     *OpenAPIContact `json:"contact,omitempty"`
}

type OpenAPIContact struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	URL   string `json:"url,omitempty"`
}

type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type OpenAPIPath map[string]OpenAPIOperation

type OpenAPIOperation struct {
	Tags        []string                   `json:"tags,omitempty"`
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	OperationID string                     `json:"operationId,omitempty"`
	Parameters  []OpenAPIParameter         `json:"parameters,omitempty"`
	RequestBody *OpenAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]OpenAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type OpenAPIParameter struct {
	Name        string        `json:"name"`
	In          string        `json:"in"` // "query", "path", "header"
	Required    bool          `json:"required,omitempty"`
	Description string        `json:"description,omitempty"`
	Schema      OpenAPISchema `json:"schema"`
}

type OpenAPIRequestBody struct {
	Required bool                        `json:"required,omitempty"`
	Content  map[string]OpenAPIMediaType `json:"content"`
}

type OpenAPIResponse struct {
	Description string                      `json:"description"`
	Content     map[string]OpenAPIMediaType `json:"content,omitempty"`
}

type OpenAPIMediaType struct {
	Schema OpenAPISchema `json:"schema,omitempty"`
}

type OpenAPISchema struct {
	Type       string                   `json:"type,omitempty"`
	Format     string                   `json:"format,omitempty"`
	Properties map[string]OpenAPISchema `json:"properties,omitempty"`
	Items      *OpenAPISchema           `json:"items,omitempty"`
	Required   []string                 `json:"required,omitempty"`
	Enum       []string                 `json:"enum,omitempty"`
	Ref        string                   `json:"$ref,omitempty"`
}

type OpenAPIComponents struct {
	Schemas         map[string]OpenAPISchema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]OpenAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type OpenAPISecurityScheme struct {
	Type   string `json:"type"`
	Scheme string `json:"scheme,omitempty"`
	In     string `json:"in,omitempty"`
	Name   string `json:"name,omitempty"`
}

type OpenAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Route metadata
// ═══════════════════════════════════════════════════════════════════════

// RouteMeta — метаданные маршрута для OpenAPI генерации.
type RouteMeta struct {
	Method      string // GET, POST, PUT, DELETE
	Path        string // /api/v1/devices/{id}
	Tag         string // Devices
	Summary     string // Get device by ID
	Description string // Full description
	OperationID string // getDeviceById
	Auth        bool   // requires JWT
	APIKey      bool   // requires API key
	Body        bool   // has request body
}

// GenerateOpenAPI создаёт OpenAPI 3.1 spec из метаданных маршрутов.
func GenerateOpenAPI(routes []RouteMeta, baseURL, version string) *OpenAPISpec {
	spec := &OpenAPISpec{
		OpenAPI: "3.1.0",
		Info: OpenAPIInfo{
			Title:       "CCTV Health Monitor API",
			Version:     version,
			Description: "REST API for CCTV Health Monitor platform. Provides device monitoring, alerting, CMMS work order management, and analytics.",
			Contact: &OpenAPIContact{
				Name: "CCTV Health Monitor Team",
				URL:  "https://cctv-monitor.company.com",
			},
		},
		Servers: []OpenAPIServer{
			{URL: baseURL, Description: "API Server"},
		},
		Paths: make(map[string]OpenAPIPath),
		Tags:  make([]OpenAPITag, 0),
		Components: OpenAPIComponents{
			Schemas:         DefaultSchemas(),
			SecuritySchemes: DefaultSecuritySchemes(),
		},
	}

	// Группируем по path
	pathMap := make(map[string]map[string]RouteMeta)
	tagSet := make(map[string]string)

	for _, route := range routes {
		if pathMap[route.Path] == nil {
			pathMap[route.Path] = make(map[string]RouteMeta)
		}
		pathMap[route.Path][route.Method] = route
		if route.Tag != "" {
			tagSet[route.Tag] = route.Description
		}
	}

	// Сортируем теги
	for name, desc := range tagSet {
		spec.Tags = append(spec.Tags, OpenAPITag{Name: name, Description: desc})
	}
	sort.Slice(spec.Tags, func(i, j int) bool {
		return spec.Tags[i].Name < spec.Tags[j].Name
	})

	// Строим Paths
	paths := make([]string, 0, len(pathMap))
	for p := range pathMap {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, path := range paths {
		methods := pathMap[path]
		openAPIPath := make(OpenAPIPath)

		for method, route := range methods {
			op := buildOperation(route)
			openAPIPath[strings.ToLower(method)] = op
		}

		spec.Paths[path] = openAPIPath
	}

	return spec
}

// buildOperation строит OpenAPIOperation из RouteMeta.
func buildOperation(route RouteMeta) OpenAPIOperation {
	op := OpenAPIOperation{
		Tags:        []string{},
		Summary:     route.Summary,
		Description: route.Description,
		OperationID: route.OperationID,
		Responses:   DefaultResponses(),
	}

	if route.Tag != "" {
		op.Tags = append(op.Tags, route.Tag)
	}

	// Path parameters
	for _, segment := range strings.Split(route.Path, "/") {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			name := strings.Trim(segment, "{}")
			op.Parameters = append(op.Parameters, OpenAPIParameter{
				Name:        name,
				In:          "path",
				Required:    true,
				Description: fmt.Sprintf("%s ID", name),
				Schema:      OpenAPISchema{Type: "string"},
			})
		}
	}

	// Request body
	if route.Body {
		op.RequestBody = &OpenAPIRequestBody{
			Required: true,
			Content: map[string]OpenAPIMediaType{
				"application/json": {
					Schema: OpenAPISchema{Type: "object"},
				},
			},
		}
	}

	// Security
	if route.Auth {
		op.Security = []map[string][]string{
			{"bearerAuth": {}},
		}
	}
	if route.APIKey {
		op.Security = []map[string][]string{
			{"apiKey": {}},
		}
	}

	return op
}

// ═══════════════════════════════════════════════════════════════════════
// Defaults
// ═══════════════════════════════════════════════════════════════════════

func DefaultSecuritySchemes() map[string]OpenAPISecurityScheme {
	return map[string]OpenAPISecurityScheme{
		"bearerAuth": {
			Type:   "http",
			Scheme: "bearer",
			In:     "header",
			Name:   "Authorization",
		},
		"apiKey": {
			Type: "apiKey",
			In:   "header",
			Name: "X-API-Key",
		},
	}
}

func DefaultResponses() map[string]OpenAPIResponse {
	return map[string]OpenAPIResponse{
		"200": {Description: "Successful operation"},
		"400": {Description: "Bad request"},
		"401": {Description: "Unauthorized"},
		"403": {Description: "Forbidden"},
		"404": {Description: "Not found"},
		"429": {Description: "Too many requests (rate limit)"},
		"500": {Description: "Internal server error"},
	}
}

func DefaultSchemas() map[string]OpenAPISchema {
	return map[string]OpenAPISchema{
		"Error": {
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"error": {
					Type: "object",
					Properties: map[string]OpenAPISchema{
						"code":    {Type: "string"},
						"message": {Type: "string"},
					},
				},
				"trace_id":  {Type: "string"},
				"timestamp": {Type: "string", Format: "date-time"},
			},
		},
		"Pagination": {
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"total":       {Type: "integer"},
				"page":        {Type: "integer"},
				"page_size":   {Type: "integer"},
				"total_pages": {Type: "integer"},
			},
		},
		"Device": {
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"device_id":   {Type: "string"},
				"name":        {Type: "string"},
				"status":      {Type: "string", Enum: []string{"ONLINE", "OFFLINE", "WARNING"}},
				"device_type": {Type: "string", Enum: []string{"camera", "nvr", "dvr", "switch"}},
				"health":      {Type: "string", Enum: []string{"healthy", "faulty", "degraded"}},
			},
		},
		"WorkOrder": {
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"id":        {Type: "string"},
				"title":     {Type: "string"},
				"type":      {Type: "string"},
				"status":    {Type: "string"},
				"priority":  {Type: "string"},
				"device_id": {Type: "string"},
			},
		},
		"APIKey": {
			Type: "object",
			Properties: map[string]OpenAPISchema{
				"id":         {Type: "string"},
				"name":       {Type: "string"},
				"key_prefix": {Type: "string"},
				"role":       {Type: "string"},
				"expires_at": {Type: "string", Format: "date-time"},
				"created_at": {Type: "string", Format: "date-time"},
			},
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════
// HTTP Handler
// ═══════════════════════════════════════════════════════════════════════

// ServeOpenAPIJSON serves the OpenAPI spec as JSON.
func ServeOpenAPIJSON(w http.ResponseWriter, r *http.Request, routes []RouteMeta, baseURL, version string) {
	spec := GenerateOpenAPI(routes, baseURL, version)
	data, _ := json.MarshalIndent(spec, "", "  ")

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// ServeSwaggerUI serves the Swagger UI HTML page.
// nonce — CSP nonce для inline-скрипта (OWASP ASVS V5.3.3).
func ServeSwaggerUI(w http.ResponseWriter, r *http.Request, nonce string) {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>CCTV Health Monitor API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>body { margin: 0; background: #f8fafc; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script nonce="%s">
    SwaggerUIBundle({
      url: '/api/v1/openapi.json',
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [SwaggerUIBundle.presets.apis],
      layout: "BaseLayout",
    });
  </script>
</body>
</html>`, nonce)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}

// DefaultRoutes возвращает метаданные всех API маршрутов.
func DefaultRoutes() []RouteMeta {
	now := time.Now().UTC().Format(time.RFC3339)
	_ = now

	return []RouteMeta{
		// Health
		{GET, "/health", "Health", "Health check", "Returns service health status", "healthCheck", false, false, false},

		// Auth
		{POST, "/api/v1/auth/login", "Authentication", "User login", "Authenticate user and return JWT", "authLogin", false, false, true},
		{POST, "/api/v1/auth/refresh", "Authentication", "Refresh token", "Refresh JWT token", "authRefresh", true, false, true},
		{POST, "/api/v1/auth/logout", "Authentication", "User logout", "Invalidate session", "authLogout", true, false, false},

		// Devices
		{GET, "/api/v1/devices", "Devices", "List devices", "Get paginated list of devices with filters", "listDevices", true, true, false},
		{GET, "/api/v1/devices/{id}", "Devices", "Get device", "Get device by ID", "getDevice", true, true, false},
		{POST, "/api/v1/devices", "Devices", "Create device", "Register new CCTV device", "createDevice", true, true, true},
		{PUT, "/api/v1/devices/{id}", "Devices", "Update device", "Update device attributes", "updateDevice", true, true, true},
		{DELETE, "/api/v1/devices/{id}", "Devices", "Delete device", "Soft-delete a device", "deleteDevice", true, false, false},

		// Work Orders
		{GET, "/api/v1/work-orders", "Work Orders", "List work orders", "Get paginated list of work orders", "listWorkOrders", true, true, false},
		{GET, "/api/v1/work-orders/{id}", "Work Orders", "Get work order", "Get work order by ID", "getWorkOrder", true, true, false},
		{POST, "/api/v1/work-orders", "Work Orders", "Create work order", "Create new work order", "createWorkOrder", true, true, true},
		{PUT, "/api/v1/work-orders/{id}", "Work Orders", "Update work order", "Update work order", "updateWorkOrder", true, true, true},
		{POST, "/api/v1/work-orders/{id}/start", "Work Orders", "Start work", "Start work on order", "startWorkOrder", true, true, false},
		{POST, "/api/v1/work-orders/{id}/complete", "Work Orders", "Complete work order", "Mark work order as completed", "completeWorkOrder", true, true, true},
		{POST, "/api/v1/work-orders/{id}/cancel", "Work Orders", "Cancel work order", "Cancel a work order", "cancelWorkOrder", true, true, true},

		// Sites
		{GET, "/api/v1/sites", "Sites", "List sites", "Get all surveillance sites", "listSites", true, true, false},
		{GET, "/api/v1/sites/{id}", "Sites", "Get site", "Get site by ID", "getSite", true, true, false},
		{POST, "/api/v1/sites", "Sites", "Create site", "Register new site", "createSite", true, true, true},
		{PUT, "/api/v1/sites/{id}", "Sites", "Update site", "Update site details", "updateSite", true, true, true},
		{DELETE, "/api/v1/sites/{id}", "Sites", "Delete site", "Delete a site", "deleteSite", true, true, false},

		// Spare Parts
		{GET, "/api/v1/spare-parts", "Spare Parts", "List spare parts", "Get inventory of spare parts", "listSpareParts", true, true, false},
		{GET, "/api/v1/spare-parts/{id}", "Spare Parts", "Get spare part", "Get spare part by ID", "getSparePart", true, true, false},
		{POST, "/api/v1/spare-parts", "Spare Parts", "Create spare part", "Add new spare part to inventory", "createSparePart", true, true, true},
		{PUT, "/api/v1/spare-parts/{id}", "Spare Parts", "Update spare part", "Update spare part details", "updateSparePart", true, true, true},
		{DELETE, "/api/v1/spare-parts/{id}", "Spare Parts", "Delete spare part", "Remove spare part", "deleteSparePart", true, true, false},

		// Alerts
		{GET, "/api/v1/alerts", "Alerts", "List alerts", "Get paginated alert history", "listAlerts", true, true, false},
		{GET, "/api/v1/alerts/{id}", "Alerts", "Get alert", "Get alert details", "getAlert", true, true, false},

		// API Keys
		{GET, "/api/v1/api-keys", "API Keys", "List API keys", "Get all API keys", "listAPIKeys", true, false, false},
		{POST, "/api/v1/api-keys", "API Keys", "Create API key", "Generate new API key", "createAPIKey", true, false, true},
		{DELETE, "/api/v1/api-keys/{id}", "API Keys", "Delete API key", "Revoke API key", "deleteAPIKey", true, false, false},

		// CMMS Integration
		{GET, "/api/v1/cmms/status", "CMMS", "CMMS adapter status", "Get status of all CMMS adapters", "cmmsStatus", true, false, false},
		{POST, "/api/v1/cmms/sync", "CMMS", "Force sync", "Force bi-directional CMMS sync", "cmmsSync", true, false, false},

		// SLA
		{GET, "/api/v1/sla/config", "SLA", "SLA config", "Get SLA policy configuration", "getSLAConfig", true, true, false},
		{PUT, "/api/v1/sla/config/{priority}", "SLA", "Update SLA config", "Update SLA policy by priority", "updateSLAConfig", true, false, true},

		// Analytics
		{GET, "/api/v1/analytics/dashboard", "Analytics", "Dashboard data", "Get analytics dashboard data", "getDashboard", true, true, false},
		{GET, "/api/v1/analytics/work-orders", "Analytics", "WO analytics", "Work order analytics", "getWOAnalytics", true, true, false},
		{GET, "/api/v1/analytics/sla", "Analytics", "SLA analytics", "SLA compliance analytics", "getSLAAnalytics", true, true, false},

		// Users
		{GET, "/api/v1/users", "Users", "List users", "Get all users", "listUsers", true, false, false},
		{POST, "/api/v1/users", "Users", "Create user", "Create new user", "createUser", true, false, true},
		{PUT, "/api/v1/users/{id}", "Users", "Update user", "Update user profile", "updateUser", true, false, true},
		{DELETE, "/api/v1/users/{id}", "Users", "Delete user", "Delete user account", "deleteUser", true, false, false},

		// Sessions
		{GET, "/api/v1/sessions", "Sessions", "List sessions", "Get active user sessions", "listSessions", true, false, false},
		{DELETE, "/api/v1/sessions/{id}", "Sessions", "Revoke session", "Force logout a session", "revokeSession", true, false, false},

		// Audit
		{GET, "/api/v1/audit/log", "Audit", "Audit log", "Get audit trail log", "getAuditLog", true, false, false},

		// Reports
		{GET, "/api/v1/reports/maintenance", "Reports", "Maintenance report", "Get maintenance report", "getMaintenanceReport", true, true, false},
		{GET, "/api/v1/reports/sla-compliance", "Reports", "SLA compliance report", "Get SLA compliance report", "getSLAComplianceReport", true, true, false},
		{GET, "/api/v1/reports/export", "Reports", "Export report", "Export report as CSV/PDF", "exportReport", true, true, false},

		// RCA (AI-01)
		{GET, "/api/v1/rca/{id}", "RCA", "RCA Graph", "Get RCA visualization graph for device", "getRCAGraph", true, true, false},

		// WebSocket
		{GET, "/api/v1/ws", "WebSocket", "WebSocket connection", "Real-time event stream via WebSocket", "websocketConnect", true, false, false},
	}
}

// HTTP method constants
const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)
