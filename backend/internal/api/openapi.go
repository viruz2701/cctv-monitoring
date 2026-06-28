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
	return []RouteMeta{
		// ── Health ─────────────────────────────────────────────────────
		{GET, "/health", "Health", "Health check", "Returns service health status", "healthCheck", false, false, false},
		{GET, "/health/ready", "Health", "Readiness probe", "Kubernetes readiness probe with dependency checks", "healthReady", false, false, false},
		{GET, "/health/dependencies", "Health", "Dependencies status", "Detailed dependency status (DB, NATS, Redis)", "healthDependencies", false, false, false},

		// ── OpenAPI ────────────────────────────────────────────────────
		{GET, "/api/v1/openapi.json", "OpenAPI", "OpenAPI spec", "OpenAPI 3.1 specification as JSON", "openapiJSON", false, false, false},
		{GET, "/api/v1/docs", "OpenAPI", "Swagger UI", "Interactive API documentation via Swagger UI", "swaggerUI", false, false, false},

		// ── Setup Wizard (P0-CE.4) ─────────────────────────────────────
		{GET, "/api/v1/setup/status", "Setup", "Setup status", "Check if setup wizard has been completed", "setupStatus", false, false, false},
		{GET, "/api/v1/setup/regions", "Setup", "List regions", "Get available compliance regions for setup", "setupRegions", false, false, false},
		{POST, "/api/v1/setup/region", "Setup", "Set region", "Set compliance region (immutable after first login)", "setupRegion", false, false, true},
		{POST, "/api/v1/setup/admin", "Setup", "Create admin", "Create initial admin user during setup", "setupAdmin", false, false, true},
		{POST, "/api/v1/setup/complete", "Setup", "Complete setup", "Finalize setup wizard configuration", "setupComplete", false, false, true},

		// ── Public Work Requests (WO-4.1.1) ──────────────────────────
		{POST, "/api/v1/public/work-requests", "Public", "Submit work request", "Submit maintenance request without authentication", "submitWorkRequest", false, false, true},

		// ── Auth ─────────────────────────────────────────────────────
		{POST, "/api/v1/auth/login", "Authentication", "User login", "Authenticate user and return JWT", "authLogin", false, false, true},
		{POST, "/api/v1/auth/refresh", "Authentication", "Refresh token", "Refresh JWT token", "authRefresh", true, false, true},
		{POST, "/api/v1/auth/logout", "Authentication", "User logout", "Invalidate session", "authLogout", true, false, false},
		{POST, "/api/v1/auth/2fa/verify", "Authentication", "Verify 2FA", "Verify TOTP 2FA code during login", "verify2FA", false, false, true},
		{POST, "/api/v1/auth/2fa/setup", "Authentication", "Setup 2FA", "Enable TOTP 2FA for current user", "setup2FA", true, false, true},
		{POST, "/api/v1/auth/2fa/disable", "Authentication", "Disable 2FA", "Disable TOTP 2FA", "disable2FA", true, false, true},
		{POST, "/api/v1/auth/webauthn/register", "Authentication", "Register WebAuthn", "Register FIDO2/WebAuthn credential", "registerWebAuthn", true, false, true},
		{POST, "/api/v1/auth/webauthn/authenticate", "Authentication", "WebAuthn auth", "Authenticate with WebAuthn credential", "authenticateWebAuthn", false, false, true},

		// ── Devices ─────────────────────────────────────────────────
		{GET, "/api/v1/devices", "Devices", "List devices", "Get paginated list of devices with filters", "listDevices", true, true, false},
		{GET, "/api/v1/devices/{id}", "Devices", "Get device", "Get device by ID", "getDevice", true, true, false},
		{POST, "/api/v1/devices", "Devices", "Create device", "Register new CCTV device", "createDevice", true, true, true},
		{PUT, "/api/v1/devices/{id}", "Devices", "Update device", "Update device attributes", "updateDevice", true, true, true},
		{DELETE, "/api/v1/devices/{id}", "Devices", "Delete device", "Soft-delete a device", "deleteDevice", true, false, false},
		{POST, "/api/v1/devices/{id}/status", "Devices", "Update status", "Update device health/status", "updateDeviceStatus", true, true, true},
		{POST, "/api/v1/devices/{id}/reboot", "Devices", "Reboot device", "Send reboot command to device", "rebootDevice", true, true, false},

		// ── Work Orders ──────────────────────────────────────────────
		{GET, "/api/v1/work-orders", "Work Orders", "List work orders", "Get paginated list of work orders", "listWorkOrders", true, true, false},
		{GET, "/api/v1/work-orders/{id}", "Work Orders", "Get work order", "Get work order by ID", "getWorkOrder", true, true, false},
		{POST, "/api/v1/work-orders", "Work Orders", "Create work order", "Create new work order", "createWorkOrder", true, true, true},
		{PUT, "/api/v1/work-orders/{id}", "Work Orders", "Update work order", "Update work order", "updateWorkOrder", true, true, true},
		{POST, "/api/v1/work-orders/{id}/start", "Work Orders", "Start work", "Start work on order", "startWorkOrder", true, true, false},
		{POST, "/api/v1/work-orders/{id}/complete", "Work Orders", "Complete work order", "Mark work order as completed", "completeWorkOrder", true, true, true},
		{POST, "/api/v1/work-orders/{id}/cancel", "Work Orders", "Cancel work order", "Cancel a work order", "cancelWorkOrder", true, true, true},
		{POST, "/api/v1/work-orders/{id}/assign", "Work Orders", "Assign technician", "Assign technician to work order", "assignWorkOrder", true, true, true},
		{GET, "/api/v1/work-orders/calendar", "Work Orders", "Calendar view", "Get work orders in calendar format", "calendarWorkOrders", true, true, false},

		// ── Sites ────────────────────────────────────────────────────
		{GET, "/api/v1/sites", "Sites", "List sites", "Get all surveillance sites", "listSites", true, true, false},
		{GET, "/api/v1/sites/{id}", "Sites", "Get site", "Get site by ID", "getSite", true, true, false},
		{POST, "/api/v1/sites", "Sites", "Create site", "Register new site", "createSite", true, true, true},
		{PUT, "/api/v1/sites/{id}", "Sites", "Update site", "Update site details", "updateSite", true, true, true},
		{DELETE, "/api/v1/sites/{id}", "Sites", "Delete site", "Delete a site", "deleteSite", true, true, false},

		// ── Spare Parts ──────────────────────────────────────────────
		{GET, "/api/v1/spare-parts", "Spare Parts", "List spare parts", "Get inventory of spare parts", "listSpareParts", true, true, false},
		{GET, "/api/v1/spare-parts/{id}", "Spare Parts", "Get spare part", "Get spare part by ID", "getSparePart", true, true, false},
		{POST, "/api/v1/spare-parts", "Spare Parts", "Create spare part", "Add new spare part to inventory", "createSparePart", true, true, true},
		{PUT, "/api/v1/spare-parts/{id}", "Spare Parts", "Update spare part", "Update spare part details", "updateSparePart", true, true, true},
		{DELETE, "/api/v1/spare-parts/{id}", "Spare Parts", "Delete spare part", "Remove spare part", "deleteSparePart", true, true, false},

		// ── Alerts ───────────────────────────────────────────────────
		{GET, "/api/v1/alerts", "Alerts", "List alerts", "Get paginated alert history", "listAlerts", true, true, false},
		{GET, "/api/v1/alerts/{id}", "Alerts", "Get alert", "Get alert details", "getAlert", true, true, false},
		{PUT, "/api/v1/alerts/{id}/acknowledge", "Alerts", "Acknowledge alert", "Acknowledge and resolve an alert", "acknowledgeAlert", true, true, false},
		{POST, "/api/v1/external/alarm", "Alerts", "External alarm", "Receive alarm from external P2P device", "externalAlarm", false, true, true},

		// ── API Keys ─────────────────────────────────────────────────
		{GET, "/api/v1/api-keys", "API Keys", "List API keys", "Get all API keys", "listAPIKeys", true, false, false},
		{POST, "/api/v1/api-keys", "API Keys", "Create API key", "Generate new API key", "createAPIKey", true, false, true},
		{DELETE, "/api/v1/api-keys/{id}", "API Keys", "Delete API key", "Revoke API key", "deleteAPIKey", true, false, false},

		// ── CMMS Integration ──────────────────────────────────────────
		{GET, "/api/v1/cmms/status", "CMMS", "CMMS adapter status", "Get status of all CMMS adapters", "cmmsStatus", true, false, false},
		{POST, "/api/v1/cmms/sync", "CMMS", "Force sync", "Force bi-directional CMMS sync", "cmmsSync", true, false, false},

		// ── SLA ──────────────────────────────────────────────────────
		{GET, "/api/v1/sla/config", "SLA", "SLA config", "Get SLA policy configuration", "getSLAConfig", true, true, false},
		{PUT, "/api/v1/sla/config/{priority}", "SLA", "Update SLA config", "Update SLA policy by priority", "updateSLAConfig", true, false, true},
		{GET, "/api/v1/sla/breaches", "SLA", "SLA breaches", "Get SLA breach history", "listSLABreaches", true, true, false},

		// ── Analytics ────────────────────────────────────────────────
		{GET, "/api/v1/analytics/dashboard", "Analytics", "Dashboard data", "Get analytics dashboard data", "getDashboard", true, true, false},
		{GET, "/api/v1/analytics/work-orders", "Analytics", "WO analytics", "Work order analytics", "getWOAnalytics", true, true, false},
		{GET, "/api/v1/analytics/sla", "Analytics", "SLA analytics", "SLA compliance analytics", "getSLAAnalytics", true, true, false},
		{GET, "/api/v1/analytics/devices", "Analytics", "Device analytics", "Device health and performance analytics", "getDeviceAnalytics", true, true, false},

		// ── Users ────────────────────────────────────────────────────
		{GET, "/api/v1/users", "Users", "List users", "Get all users", "listUsers", true, false, false},
		{POST, "/api/v1/users", "Users", "Create user", "Create new user", "createUser", true, false, true},
		{PUT, "/api/v1/users/{id}", "Users", "Update user", "Update user profile", "updateUser", true, false, true},
		{DELETE, "/api/v1/users/{id}", "Users", "Delete user", "Delete user account", "deleteUser", true, false, false},
		{GET, "/api/v1/users/me", "Users", "Current user", "Get current user profile", "getCurrentUser", true, false, false},
		{PUT, "/api/v1/users/me", "Users", "Update profile", "Update own user profile", "updateCurrentUser", true, false, true},

		// ── Sessions ─────────────────────────────────────────────────
		{GET, "/api/v1/sessions", "Sessions", "List sessions", "Get active user sessions", "listSessions", true, false, false},
		{DELETE, "/api/v1/sessions/{id}", "Sessions", "Revoke session", "Force logout a session", "revokeSession", true, false, false},

		// ── Audit ────────────────────────────────────────────────────
		{GET, "/api/v1/audit/log", "Audit", "Audit log", "Get audit trail log", "getAuditLog", true, false, false},
		{GET, "/api/v1/audit/verify", "Audit", "Verify chain", "Verify audit log chain integrity (tamper detection)", "verifyAuditChain", true, false, false},
		{GET, "/api/v1/audit/compliance", "Audit", "Compliance report", "Get compliance audit report", "getAuditCompliance", true, false, false},
		{POST, "/api/v1/audit/archive", "Audit", "Archive audit", "Archive audit logs per retention policy", "archiveAuditLogs", true, false, true},

		// ── Reports ──────────────────────────────────────────────────
		{GET, "/api/v1/reports/maintenance", "Reports", "Maintenance report", "Get maintenance report", "getMaintenanceReport", true, true, false},
		{GET, "/api/v1/reports/sla-compliance", "Reports", "SLA compliance report", "Get SLA compliance report", "getSLAComplianceReport", true, true, false},
		{GET, "/api/v1/reports/export", "Reports", "Export report", "Export report as CSV/PDF", "exportReport", true, true, false},

		// ── RCA (AI-01) ──────────────────────────────────────────────
		{GET, "/api/v1/rca/{id}", "RCA", "RCA Graph", "Get RCA visualization graph for device", "getRCAGraph", true, true, false},

		// ── WebSocket ────────────────────────────────────────────────
		{GET, "/api/v1/ws", "WebSocket", "WebSocket connection", "Real-time event stream via WebSocket", "websocketConnect", true, false, false},
		{GET, "/api/v1/ws/alarms", "WebSocket", "Alarm WebSocket", "Real-time alarm stream via WebSocket (JWT in query)", "websocketAlarms", false, false, false},

		// ── Workspace ────────────────────────────────────────────────
		{GET, "/api/v1/workspace/layout", "Workspace", "Get layout", "Get workspace dashboard layout configuration", "getWorkspaceLayout", true, false, false},
		{POST, "/api/v1/workspace/layout", "Workspace", "Save layout", "Save workspace dashboard layout", "saveWorkspaceLayout", true, false, true},

		// ── Compliance ──────────────────────────────────────────────
		{GET, "/api/v1/compliance/profile", "Compliance", "Get profile", "Get current compliance profile settings", "getComplianceProfile", true, false, false},
		{GET, "/api/v1/compliance/report", "Compliance", "Compliance report", "Generate compliance status report", "getComplianceReport", true, false, false},
		{GET, "/api/v1/compliance/regions", "Compliance", "List regions", "Get available compliance regions", "listComplianceRegions", true, false, false},
		{POST, "/api/v1/compliance/tenant", "Compliance", "Set tenant profile", "Set compliance profile for tenant (SaaS)", "setTenantCompliance", true, false, true},

		// ── GDPR (P2-EU.1) ───────────────────────────────────────────
		{POST, "/api/v1/gdpr/forget", "GDPR", "Right to be forgotten", "Request personal data deletion (GDPR)", "gdprForget", true, false, true},
		{GET, "/api/v1/gdpr/data", "GDPR", "Export personal data", "Export all personal data (GDPR Art. 20)", "gdprExportData", true, false, false},

		// ── Personal Data (P2-RU.2: 152-ФЗ) ─────────────────────────
		{POST, "/api/v1/personal-data/consent", "Personal Data", "Update consent", "Update personal data processing consent", "updateConsent", true, false, true},
		{GET, "/api/v1/personal-data/log", "Personal Data", "Processing log", "Get personal data processing log", "getProcessingLog", true, false, false},

		// ── Storage / Data Residency (P0-CE.6) ──────────────────────
		{GET, "/api/v1/storage/regions", "Storage", "Storage regions", "Get available storage regions for data residency", "listStorageRegions", true, false, false},
		{POST, "/api/v1/storage/migrate", "Storage", "Migrate data", "Migrate data between storage regions", "migrateStorageRegion", true, false, true},

		// ── Black Box (KF-15.2.4) ─────────────────────────────────────
		{GET, "/api/v1/blackbox/incidents", "Black Box", "List incidents", "Get black box incident records", "listBlackBoxIncidents", true, false, false},
		{GET, "/api/v1/blackbox/incidents/{id}", "Black Box", "Get incident", "Get black box incident details", "getBlackBoxIncident", true, false, false},

		// ── Feature Flags (F-0.2.4) ──────────────────────────────────
		{GET, "/api/v1/feature-flags", "Feature Flags", "List flags", "Get all feature flags", "listFeatureFlags", true, false, false},
		{PUT, "/api/v1/feature-flags/{key}", "Feature Flags", "Update flag", "Update a feature flag", "updateFeatureFlag", true, false, true},

		// ── Webhooks ─────────────────────────────────────────────────
		{GET, "/api/v1/webhooks", "Webhooks", "List webhooks", "Get registered webhook endpoints", "listWebhooks", true, false, false},
		{POST, "/api/v1/webhooks", "Webhooks", "Create webhook", "Register new webhook endpoint", "createWebhook", true, false, true},
		{DELETE, "/api/v1/webhooks/{id}", "Webhooks", "Delete webhook", "Remove webhook endpoint", "deleteWebhook", true, false, false},
		{POST, "/api/v1/webhooks/{id}/test", "Webhooks", "Test webhook", "Send test event to webhook endpoint", "testWebhook", true, false, false},

		// ── Camera Models (P0-9) ────────────────────────────────────
		{GET, "/api/v1/camera-models", "Camera Models", "List models", "Get camera specification database", "listCameraModels", true, true, false},
		{POST, "/api/v1/camera-models", "Camera Models", "Create model", "Add camera model to database", "createCameraModel", true, true, true},

		// ── GraphQL ──────────────────────────────────────────────────
		{POST, "/api/v1/graphql", "GraphQL", "GraphQL endpoint", "Read-only GraphQL query endpoint", "graphqlQuery", true, false, true},

		// ── Admin (Multi-Region DR, P3-1) ──────────────────────────
		{GET, "/api/v1/admin/regions", "Admin", "List regions", "Get multi-region deployment status", "adminListRegions", true, false, false},
		{POST, "/api/v1/admin/failover", "Admin", "Trigger failover", "Trigger cross-region failover", "adminTriggerFailover", true, false, true},
	}
}

// HTTP method constants
const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"
)
