package jira

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gb-telemetry-collector/internal/models"
)

// mapper — utility для безопасного извлечения полей из map[string]interface{}.
type mapper struct {
	raw map[string]interface{}
}

func newMapper(raw map[string]interface{}) *mapper { return &mapper{raw: raw} }
func (m *mapper) str(key string) string {
	if v, ok := m.raw[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}
func (m *mapper) int(key string) int {
	if v, ok := m.raw[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case json.Number:
			i, _ := n.Int64()
			return int(i)
		case string:
			i, _ := strconv.Atoi(n)
			return i
		}
	}
	return 0
}
func (m *mapper) float(key string) float64 {
	if v, ok := m.raw[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case json.Number:
			f, _ := n.Float64()
			return f
		case string:
			f, _ := strconv.ParseFloat(n, 64)
			return f
		}
	}
	return 0
}
func (m *mapper) bool(key string) bool {
	if v, ok := m.raw[key]; ok {
		switch b := v.(type) {
		case bool:
			return b
		case string:
			return b == "true" || b == "1"
		}
	}
	return false
}
func (m *mapper) time(key string) time.Time {
	if v, ok := m.raw[key]; ok {
		if t, ok := v.(string); ok {
			for _, layout := range []string{
				time.RFC3339,
				"2006-01-02T15:04:05.000-0700",
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05",
				"2006-01-02",
			} {
				if parsed, err := time.Parse(layout, t); err == nil {
					return parsed
				}
			}
		}
	}
	return time.Time{}
}
func (m *mapper) timePtr(key string) *time.Time {
	t := m.time(key)
	if t.IsZero() {
		return nil
	}
	return &t
}
func (m *mapper) rawJSON(key string) json.RawMessage {
	if v, ok := m.raw[key]; ok {
		switch r := v.(type) {
		case string:
			return json.RawMessage(r)
		default:
			data, _ := json.Marshal(r)
			return data
		}
	}
	return nil
}
func (m *mapper) strSlice(key string) []string {
	if v, ok := m.raw[key]; ok {
		switch s := v.(type) {
		case string:
			if s == "" {
				return nil
			}
			return strings.Split(s, ",")
		case []interface{}:
			var result []string
			for _, item := range s {
				result = append(result, fmt.Sprintf("%v", item))
			}
			return result
		}
	}
	return nil
}

// Jira API paths. Work Orders = Jira Issues (custom issue type: "CCTV Work Order").
const (
	pathIssue   = "/rest/api/3/issue"
	pathSearch  = "/rest/api/3/search"
	pathMyself  = "/rest/api/3/myself"
	pathUsers   = "/rest/api/3/user"
	pathProject = "/rest/api/3/project"
	pathHealth  = "/rest/api/3/myself"
)

// Jira custom field IDs (настраиваются в админке Jira).
const (
	fieldDeviceID          = "customfield_10001"
	fieldScheduleID        = "customfield_10002"
	fieldWorkOrderType     = "customfield_10003"
	fieldSLADeadline       = "customfield_10004"
	fieldChecklist         = "customfield_10005"
	fieldPhotos            = "customfield_10006"
	fieldPartsUsed         = "customfield_10007"
	fieldSLAStatus         = "customfield_10008"
	fieldDeviceName        = "customfield_10009"
	fieldAssigneeName      = "customfield_10010"
	fieldCreatedBy         = "customfield_10011"
	fieldCompletedBy       = "customfield_10012"
	fieldStock             = "customfield_10020"
	fieldMinStock          = "customfield_10021"
	fieldLocation          = "customfield_10022"
	fieldCost              = "customfield_10023"
	fieldSupplier          = "customfield_10024"
	fieldCompatibleDevices = "customfield_10025"
	fieldIntervalDays      = "customfield_10030"
	fieldCustomCron        = "customfield_10031"
	fieldEstimatedMinutes  = "customfield_10032"
	fieldNextDue           = "customfield_10033"
	fieldLastCompleted     = "customfield_10034"
	fieldScheduleType      = "customfield_10035"
	fieldResponseTime      = "customfield_10040"
	fieldResolutionTime    = "customfield_10041"
	fieldIsPrimary         = "customfield_10050"
	fieldTechnicianID      = "customfield_10051"
	fieldSiteID            = "customfield_10052"
	fieldAssignedBy        = "customfield_10053"
	fieldSkills            = "customfield_10060"
	fieldCertifications    = "customfield_10061"
	fieldCurrentWorkload   = "customfield_10062"
	fieldMaxWorkload       = "customfield_10063"
	fieldBaseLocation      = "customfield_10064"
	fieldPushToken         = "customfield_10070"
	fieldPushPlatform      = "customfield_10071"
)

// jqlSearch строит JQL из фильтров.
func jqlSearch(filters map[string]interface{}) string {
	if len(filters) == 0 {
		return "project = CCTV"
	}
	parts := []string{"project = CCTV"}
	for k, v := range filters {
		switch k {
		case "status":
			parts = append(parts, fmt.Sprintf("status = \"%v\"", v))
		case "type":
			parts = append(parts, fmt.Sprintf("issuetype = \"%v\"", v))
		case "priority":
			parts = append(parts, fmt.Sprintf("priority = \"%v\"", v))
		case "device_id":
			parts = append(parts, fmt.Sprintf("\"%s\" ~ \"%v\"", fieldDeviceID, v))
		case "assigned_to":
			parts = append(parts, fmt.Sprintf("assignee = \"%v\"", v))
		}
	}
	return strings.Join(parts, " AND ")
}

// jiraSearchRequest — тело запроса к Jira Search API.
type jiraSearchRequest struct {
	JQL        string   `json:"jql"`
	StartAt    int      `json:"startAt"`
	MaxResults int      `json:"maxResults"`
	Fields     []string `json:"fields"`
}

// jiraSearchResponse — ответ от Jira Search API.
type jiraSearchResponse struct {
	Issues []jiraIssue `json:"issues"`
	Total  int         `json:"total"`
}

// jiraIssue — Jira issue.
type jiraIssue struct {
	ID     string                 `json:"id"`
	Key    string                 `json:"key"`
	Fields map[string]interface{} `json:"fields"`
}

// jiraIssueFields — поля Jira issue, извлекаемые из Fields.
type jiraIssueFields struct {
	raw map[string]interface{}
}

func (f *jiraIssueFields) get(key string) interface{} {
	if f.raw == nil {
		return nil
	}
	return f.raw[key]
}

func (f *jiraIssueFields) getStr(key string) string {
	if v := f.get(key); v != nil {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// jiraTransition — структура для смены статуса.
type jiraTransition struct {
	Transition struct {
		ID string `json:"id"`
	} `json:"transition"`
}

// ── Model → Jira body mappers ────────────────────────────────────

func toWorkOrderJiraBody(wo *models.WorkOrder) map[string]interface{} {
	fields := map[string]interface{}{
		"project":          map[string]string{"key": "CCTV"},
		"summary":          fmt.Sprintf("[%s] %s - %s", wo.Type, wo.DeviceName, wo.ID),
		"description":      wo.Notes,
		"issuetype":        map[string]string{"name": "CCTV Work Order"},
		fieldDeviceID:      wo.DeviceID,
		fieldWorkOrderType: wo.Type,
		fieldDeviceName:    wo.DeviceName,
		fieldAssigneeName:  wo.AssigneeName,
		fieldSLAStatus:     wo.SLAStatus,
	}
	if wo.ScheduleID != nil {
		fields[fieldScheduleID] = *wo.ScheduleID
	}
	if wo.AssignedTo != nil {
		fields["assignee"] = map[string]string{"id": *wo.AssignedTo}
	}
	if wo.CreatedBy != nil {
		fields[fieldCreatedBy] = *wo.CreatedBy
	}
	if len(wo.Checklist) > 0 {
		fields[fieldChecklist] = string(wo.Checklist)
	}
	if len(wo.Photos) > 0 {
		fields[fieldPhotos] = string(wo.Photos)
	}
	if len(wo.PartsUsed) > 0 {
		fields[fieldPartsUsed] = string(wo.PartsUsed)
	}
	if wo.Priority != "" {
		fields["priority"] = map[string]string{"name": wo.Priority}
	}
	return map[string]interface{}{"fields": fields}
}

func toSparePartJiraBody(sp *models.SparePart) map[string]interface{} {
	fields := map[string]interface{}{
		"project":              map[string]string{"key": "CCTV"},
		"summary":              fmt.Sprintf("[SPARE] %s - %s", sp.Name, sp.SKU),
		"description":          sp.Category,
		"issuetype":            map[string]string{"name": "Task"},
		fieldStock:             sp.Stock,
		fieldMinStock:          sp.MinStock,
		fieldLocation:          sp.Location,
		fieldCost:              sp.Cost,
		fieldSupplier:          sp.Supplier,
		fieldCompatibleDevices: strings.Join(sp.CompatibleDevices, ","),
	}
	return map[string]interface{}{"fields": fields}
}

func toMaintenanceScheduleJiraBody(ms *models.MaintenanceSchedule) map[string]interface{} {
	fields := map[string]interface{}{
		"project":             map[string]string{"key": "CCTV"},
		"summary":             fmt.Sprintf("[SCHEDULE] %s", ms.DeviceName),
		"description":         ms.Notes,
		"issuetype":           map[string]string{"name": "Task"},
		fieldDeviceID:         ms.DeviceID,
		fieldScheduleType:     ms.ScheduleType,
		fieldIntervalDays:     ms.IntervalDays,
		fieldEstimatedMinutes: ms.EstimatedMinutes,
		fieldDeviceName:       ms.DeviceName,
		fieldAssigneeName:     ms.AssigneeName,
		fieldNextDue:          ms.NextDue.Format("2006-01-02 15:04:05"),
	}
	if ms.CustomCron != "" {
		fields[fieldCustomCron] = ms.CustomCron
	}
	if ms.AssignedTo != nil {
		fields["assignee"] = map[string]string{"id": *ms.AssignedTo}
	}
	if len(ms.Checklist) > 0 {
		fields[fieldChecklist] = string(ms.Checklist)
	}
	if ms.Priority != "" {
		fields["priority"] = map[string]string{"name": ms.Priority}
	}
	return map[string]interface{}{"fields": fields}
}

func toTechnicianAssignmentJiraBody(a *models.TechnicianSiteAssignment) map[string]interface{} {
	fields := map[string]interface{}{
		"project":         map[string]string{"key": "CCTV"},
		"summary":         fmt.Sprintf("[ASSIGN] %s → %s", a.TechnicianName, a.SiteName),
		"issuetype":       map[string]string{"name": "Task"},
		fieldTechnicianID: a.TechnicianID,
		fieldSiteID:       a.SiteID,
		fieldIsPrimary:    a.IsPrimary,
		fieldAssignedBy:   a.AssignedBy,
	}
	return map[string]interface{}{"fields": fields}
}

// ── Jira response → Model mappers ────────────────────────────────

func toWorkOrder(issue jiraIssue) models.WorkOrder {
	wo := models.WorkOrder{}
	raw := issue.Fields
	m := newMapper(raw)
	wo.ID = issue.Key
	wo.DeviceID = m.str(fieldDeviceID)
	wo.Type = m.str(fieldWorkOrderType)
	wo.Notes = m.str("description")
	wo.SLAStatus = m.str(fieldSLAStatus)
	wo.DeviceName = m.str(fieldDeviceName)
	wo.AssigneeName = m.str(fieldAssigneeName)
	wo.Priority = m.str("priority")

	if v := m.str("status"); v != "" {
		wo.Status = jiraStatusToInternal(v)
	}

	if v := m.str(fieldScheduleID); v != "" {
		wo.ScheduleID = &v
	}
	if v := m.str(fieldCreatedBy); v != "" {
		wo.CreatedBy = &v
	}
	if assignee, ok := raw["assignee"].(map[string]interface{}); ok {
		if id, ok := assignee["accountId"].(string); ok {
			wo.AssignedTo = &id
		}
	}

	wo.CreatedAt = m.time("created")
	wo.UpdatedAt = m.time("updated")
	wo.StartedAt = m.timePtr("customfield_10003")
	wo.CompletedAt = m.timePtr("resolutiondate")
	wo.SLADeadline = m.timePtr("duedate")
	wo.Checklist = m.rawJSON(fieldChecklist)
	wo.Photos = m.rawJSON(fieldPhotos)
	wo.PartsUsed = m.rawJSON(fieldPartsUsed)
	return wo
}

func toSparePart(issue jiraIssue) models.SparePart {
	sp := models.SparePart{}
	raw := issue.Fields
	m := newMapper(raw)
	sp.ID = issue.Key
	sp.Name = m.str("summary")
	sp.SKU = m.str("description")
	sp.Category = m.str("description")
	sp.Stock = m.int(fieldStock)
	sp.MinStock = m.int(fieldMinStock)
	sp.Location = m.str(fieldLocation)
	sp.Cost = m.float(fieldCost)
	sp.Supplier = m.str(fieldSupplier)
	sp.CreatedAt = m.time("created")
	sp.UpdatedAt = m.time("updated")
	sp.CompatibleDevices = m.strSlice(fieldCompatibleDevices)
	return sp
}

func toMaintenanceSchedule(issue jiraIssue) models.MaintenanceSchedule {
	ms := models.MaintenanceSchedule{}
	raw := issue.Fields
	m := newMapper(raw)
	ms.ID = issue.Key
	ms.DeviceID = m.str(fieldDeviceID)
	ms.ScheduleType = m.str(fieldScheduleType)
	ms.IntervalDays = m.int(fieldIntervalDays)
	ms.CustomCron = m.str(fieldCustomCron)
	ms.EstimatedMinutes = m.int(fieldEstimatedMinutes)
	ms.Notes = m.str("description")
	ms.DeviceName = m.str(fieldDeviceName)
	ms.AssigneeName = m.str(fieldAssigneeName)
	ms.Priority = m.str("priority")

	if v := m.str(fieldLastCompleted); v != "" {
		t := m.time(fieldLastCompleted)
		ms.LastCompleted = &t
	}
	ms.NextDue = m.time(fieldNextDue)
	ms.CreatedAt = m.time("created")
	ms.UpdatedAt = m.time("updated")
	if assignee, ok := raw["assignee"].(map[string]interface{}); ok {
		if id, ok := assignee["accountId"].(string); ok {
			ms.AssignedTo = &id
		}
	}
	ms.Checklist = m.rawJSON(fieldChecklist)
	return ms
}

func toSLAConfig(issue jiraIssue) models.SLAConfig {
	raw := issue.Fields
	m := newMapper(raw)
	return models.SLAConfig{
		ID:                    issue.Key,
		Priority:              m.str("priority"),
		ResponseTimeMinutes:   m.int(fieldResponseTime),
		ResolutionTimeMinutes: m.int(fieldResolutionTime),
	}
}

func toTechnicianWorkload(issue jiraIssue) models.TechnicianWorkload {
	raw := issue.Fields
	m := newMapper(raw)
	return models.TechnicianWorkload{
		UserID:          issue.Key,
		UserName:        m.str("summary"),
		CurrentWorkload: m.int(fieldCurrentWorkload),
		MaxWorkload:     m.int(fieldMaxWorkload),
		Skills:          m.strSlice(fieldSkills),
		BaseLocation:    strPtr(m.str(fieldBaseLocation)),
	}
}

// strPtr возвращает *string, или nil если строка пустая.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toTechnicianMonthlyStats(issue jiraIssue) models.TechnicianMonthlyStats {
	raw := issue.Fields
	m := newMapper(raw)
	return models.TechnicianMonthlyStats{
		CompletedThisMonth: m.int("customfield_10001"),
		TotalWorkOrders:    m.int("customfield_10002"),
		OnTimePercent:      m.float("customfield_10003"),
		AvgRating:          m.float("customfield_10004"),
	}
}

func toMaintenanceReport(issue jiraIssue) models.MaintenanceReport {
	raw := issue.Fields
	m := newMapper(raw)
	return models.MaintenanceReport{
		DeviceID:        m.str(fieldDeviceID),
		DeviceName:      m.str(fieldDeviceName),
		MTBF:            m.float("customfield_10005"),
		MTTR:            m.float("customfield_10006"),
		TotalWorkOrders: m.int("customfield_10007"),
		CompletedCount:  m.int("customfield_10008"),
		OverdueCount:    m.int("customfield_10009"),
		TotalCost:       m.float(fieldCost),
	}
}

func toSLAComplianceReport(issue jiraIssue) models.SLAComplianceReport {
	raw := issue.Fields
	m := newMapper(raw)
	return models.SLAComplianceReport{
		Priority:          m.str("priority"),
		TotalWorkOrders:   m.int("customfield_10010"),
		WithinSLA:         m.int("customfield_10011"),
		BreachedSLA:       m.int("customfield_10012"),
		CompliancePercent: m.float("customfield_10013"),
		AvgResponseTime:   m.float(fieldResponseTime),
		AvgResolutionTime: m.float(fieldResolutionTime),
	}
}

func toTechnicianAssignment(issue jiraIssue) models.TechnicianSiteAssignment {
	raw := issue.Fields
	m := newMapper(raw)
	return models.TechnicianSiteAssignment{
		ID:             issue.Key,
		TechnicianID:   m.str(fieldTechnicianID),
		SiteID:         m.str(fieldSiteID),
		IsPrimary:      m.bool(fieldIsPrimary),
		AssignedBy:     m.str(fieldAssignedBy),
		TechnicianName: m.str("summary"),
		SiteName:       m.str("description"),
		AssignedAt:     m.time("created"),
	}
}

// jiraStatusToInternal конвертирует статус Jira → внутренний статус.
func jiraStatusToInternal(jiraStatus string) string {
	mapping := map[string]string{
		"To Do":       "open",
		"In Progress": "in_progress",
		"Done":        "completed",
		"Cancelled":   "cancelled",
		"Open":        "open",
		"Closed":      "completed",
		"Resolved":    "completed",
		"Reopened":    "open",
	}
	if s, ok := mapping[jiraStatus]; ok {
		return s
	}
	return strings.ToLower(jiraStatus)
}

// internalStatusToJiraTransition возвращает ID перехода для смены статуса.
func internalStatusToJiraTransition(status string) string {
	mapping := map[string]string{
		"in_progress": "4", // Start Progress
		"completed":   "2", // Done
		"cancelled":   "5", // Cancel
		"open":        "3", // Reopen
	}
	if s, ok := mapping[status]; ok {
		return s
	}
	return ""
}
