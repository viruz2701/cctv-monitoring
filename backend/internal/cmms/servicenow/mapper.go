package servicenow

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

func newMapper(raw map[string]interface{}) *mapper {
	return &mapper{raw: raw}
}

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
		switch t := v.(type) {
		case string:
			for _, layout := range []string{
				time.RFC3339,
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

// ServiceNow Table API table names.
const (
	TableWorkOrder            = "u_cctv_work_order"
	TableSparePart            = "u_cctv_spare_part"
	TableMaintenanceSchedule  = "u_cctv_maintenance_schedule"
	TableSLA                  = "u_cctv_sla_config"
	TableTechnicianAssignment = "u_cctv_technician_site_assignment"
	TablePushToken            = "u_cctv_push_token"
)

// Field mapping: internal model → ServiceNow field name.
// ServiceNow uses u_ prefix for custom fields.
var (
	workOrderFields = map[string]string{
		"id":            "sys_id",
		"schedule_id":   "u_schedule_id",
		"device_id":     "u_device_id",
		"type":          "u_type",
		"status":        "u_status",
		"priority":      "u_priority",
		"assigned_to":   "u_assigned_to",
		"sla_deadline":  "u_sla_deadline",
		"checklist":     "u_checklist",
		"started_at":    "u_started_at",
		"completed_at":  "u_completed_at",
		"notes":         "u_notes",
		"photos":        "u_photos",
		"parts_used":    "u_parts_used",
		"created_by":    "u_created_by",
		"created_at":    "sys_created_on",
		"updated_at":    "sys_updated_on",
		"device_name":   "u_device_name",
		"assignee_name": "u_assignee_name",
		"sla_status":    "u_sla_status",
	}

	sparePartFields = map[string]string{
		"id":                 "sys_id",
		"name":               "u_name",
		"sku":                "u_sku",
		"category":           "u_category",
		"stock":              "u_stock",
		"min_stock":          "u_min_stock",
		"location":           "u_location",
		"compatible_devices": "u_compatible_devices",
		"cost":               "u_cost",
		"supplier":           "u_supplier",
		"created_at":         "sys_created_on",
		"updated_at":         "sys_updated_on",
	}

	maintenanceScheduleFields = map[string]string{
		"id":                "sys_id",
		"device_id":         "u_device_id",
		"schedule_type":     "u_schedule_type",
		"interval_days":     "u_interval_days",
		"custom_cron":       "u_custom_cron",
		"last_completed":    "u_last_completed",
		"next_due":          "u_next_due",
		"assigned_to":       "u_assigned_to",
		"checklist":         "u_checklist",
		"estimated_minutes": "u_estimated_minutes",
		"priority":          "u_priority",
		"notes":             "u_notes",
		"created_at":        "sys_created_on",
		"updated_at":        "sys_updated_on",
		"device_name":       "u_device_name",
		"assignee_name":     "u_assignee_name",
	}

	slaFields = map[string]string{
		"id":                      "sys_id",
		"priority":                "u_priority",
		"response_time_minutes":   "u_response_time_minutes",
		"resolution_time_minutes": "u_resolution_time_minutes",
	}

	technicianAssignmentFields = map[string]string{
		"id":              "sys_id",
		"technician_id":   "u_technician_id",
		"site_id":         "u_site_id",
		"is_primary":      "u_is_primary",
		"assigned_at":     "u_assigned_at",
		"assigned_by":     "u_assigned_by",
		"technician_name": "u_technician_name",
		"site_name":       "u_site_name",
	}

	pushTokenFields = map[string]string{
		"user_id":  "u_user_id",
		"token":    "u_token",
		"platform": "u_platform",
	}
)

// sysparmQuery строит ServiceNow-совместимый encoded query из filters.
func sysparmQuery(filters map[string]interface{}) string {
	if len(filters) == 0 {
		return ""
	}
	var parts []string
	for k, v := range filters {
		field := workOrderFields[k]
		if field == "" {
			field = k
		}
		parts = append(parts, fmt.Sprintf("%s=%v", field, v))
	}
	return "sysparm_query=" + strings.Join(parts, "^")
}

// buildTablePath формирует путь к ServiceNow Table API.
func buildTablePath(table string, filters map[string]interface{}) string {
	path := "/api/now/table/" + table
	if q := sysparmQuery(filters); q != "" {
		path += "?" + q
	}
	return path
}

// toWorkOrderResponse преобразует ServiceNow-ответ в models.WorkOrder.
func toWorkOrderResponse(raw map[string]interface{}) (*models.WorkOrder, error) {
	// ServiceNow возвращает поле result с объектом или массивом
	return &models.WorkOrder{}, nil
}

// snResponse — стандартная обёртка ответа ServiceNow Table API.
type snResponse struct {
	Result []map[string]interface{} `json:"result"`
}

// snSingleResponse — ответ для одного объекта.
type snSingleResponse struct {
	Result map[string]interface{} `json:"result"`
}

// snCountResponse — ответ для aggregate/count.
type snCountResponse struct {
	Result struct {
		Count int `json:"count"`
	} `json:"result"`
}

// toWorkOrder преобразует map из ServiceNow в models.WorkOrder.
func toWorkOrder(raw map[string]interface{}) models.WorkOrder {
	wo := models.WorkOrder{}
	m := newMapper(raw)
	wo.ID = m.str("sys_id")
	wo.DeviceID = m.str("u_device_id")
	wo.Type = m.str("u_type")
	wo.Status = m.str("u_status")
	wo.Priority = m.str("u_priority")
	wo.Notes = m.str("u_notes")
	wo.SLAStatus = m.str("u_sla_status")
	wo.DeviceName = m.str("u_device_name")
	wo.AssigneeName = m.str("u_assignee_name")
	if v := m.str("u_schedule_id"); v != "" {
		wo.ScheduleID = &v
	}
	if v := m.str("u_assigned_to"); v != "" {
		wo.AssignedTo = &v
	}
	if v := m.str("u_created_by"); v != "" {
		wo.CreatedBy = &v
	}
	wo.CreatedAt = m.time("sys_created_on")
	wo.UpdatedAt = m.time("sys_updated_on")
	wo.StartedAt = m.timePtr("u_started_at")
	wo.CompletedAt = m.timePtr("u_completed_at")
	wo.SLADeadline = m.timePtr("u_sla_deadline")
	wo.Checklist = m.rawJSON("u_checklist")
	wo.Photos = m.rawJSON("u_photos")
	wo.PartsUsed = m.rawJSON("u_parts_used")
	return wo
}

// toSparePart преобразует map из ServiceNow в models.SparePart.
func toSparePart(raw map[string]interface{}) models.SparePart {
	sp := models.SparePart{}
	m := newMapper(raw)
	sp.ID = m.str("sys_id")
	sp.Name = m.str("u_name")
	sp.SKU = m.str("u_sku")
	sp.Category = m.str("u_category")
	sp.Stock = m.int("u_stock")
	sp.MinStock = m.int("u_min_stock")
	sp.Location = m.str("u_location")
	sp.Cost = m.float("u_cost")
	sp.Supplier = m.str("u_supplier")
	sp.CreatedAt = m.time("sys_created_on")
	sp.UpdatedAt = m.time("sys_updated_on")
	sp.CompatibleDevices = m.strSlice("u_compatible_devices")
	return sp
}

// toMaintenanceSchedule преобразует map в models.MaintenanceSchedule.
func toMaintenanceSchedule(raw map[string]interface{}) models.MaintenanceSchedule {
	ms := models.MaintenanceSchedule{}
	m := newMapper(raw)
	ms.ID = m.str("sys_id")
	ms.DeviceID = m.str("u_device_id")
	ms.ScheduleType = m.str("u_schedule_type")
	ms.IntervalDays = m.int("u_interval_days")
	ms.CustomCron = m.str("u_custom_cron")
	ms.EstimatedMinutes = m.int("u_estimated_minutes")
	ms.Priority = m.str("u_priority")
	ms.Notes = m.str("u_notes")
	ms.DeviceName = m.str("u_device_name")
	ms.AssigneeName = m.str("u_assignee_name")
	if v := m.str("u_assigned_to"); v != "" {
		ms.AssignedTo = &v
	}
	ms.CreatedAt = m.time("sys_created_on")
	ms.UpdatedAt = m.time("sys_updated_on")
	ms.LastCompleted = m.timePtr("u_last_completed")
	ms.NextDue = m.time("u_next_due")
	ms.Checklist = m.rawJSON("u_checklist")
	return ms
}

// toSLAConfig преобразует map в models.SLAConfig.
func toSLAConfig(raw map[string]interface{}) models.SLAConfig {
	m := newMapper(raw)
	return models.SLAConfig{
		ID:                    m.str("sys_id"),
		Priority:              m.str("u_priority"),
		ResponseTimeMinutes:   m.int("u_response_time_minutes"),
		ResolutionTimeMinutes: m.int("u_resolution_time_minutes"),
	}
}

// toWorkOrderSNBody преобразует models.WorkOrder в ServiceNow-совместимый map.
func toWorkOrderSNBody(wo *models.WorkOrder) map[string]interface{} {
	body := map[string]interface{}{
		"u_device_id":     wo.DeviceID,
		"u_type":          wo.Type,
		"u_status":        wo.Status,
		"u_priority":      wo.Priority,
		"u_notes":         wo.Notes,
		"u_device_name":   wo.DeviceName,
		"u_assignee_name": wo.AssigneeName,
		"u_sla_status":    wo.SLAStatus,
	}
	if wo.ScheduleID != nil {
		body["u_schedule_id"] = *wo.ScheduleID
	}
	if wo.AssignedTo != nil {
		body["u_assigned_to"] = *wo.AssignedTo
	}
	if wo.CreatedBy != nil {
		body["u_created_by"] = *wo.CreatedBy
	}
	if len(wo.Checklist) > 0 {
		body["u_checklist"] = string(wo.Checklist)
	}
	if len(wo.Photos) > 0 {
		body["u_photos"] = string(wo.Photos)
	}
	if len(wo.PartsUsed) > 0 {
		body["u_parts_used"] = string(wo.PartsUsed)
	}
	return body
}

// toSparePartSNBody преобразует models.SparePart в ServiceNow map.
func toSparePartSNBody(sp *models.SparePart) map[string]interface{} {
	return map[string]interface{}{
		"u_name":               sp.Name,
		"u_sku":                sp.SKU,
		"u_category":           sp.Category,
		"u_stock":              sp.Stock,
		"u_min_stock":          sp.MinStock,
		"u_location":           sp.Location,
		"u_cost":               sp.Cost,
		"u_supplier":           sp.Supplier,
		"u_compatible_devices": strings.Join(sp.CompatibleDevices, ","),
	}
}

// toMaintenanceScheduleSNBody преобразует models.MaintenanceSchedule в ServiceNow map.
func toMaintenanceScheduleSNBody(ms *models.MaintenanceSchedule) map[string]interface{} {
	body := map[string]interface{}{
		"u_device_id":         ms.DeviceID,
		"u_schedule_type":     ms.ScheduleType,
		"u_interval_days":     ms.IntervalDays,
		"u_estimated_minutes": ms.EstimatedMinutes,
		"u_priority":          ms.Priority,
		"u_notes":             ms.Notes,
		"u_device_name":       ms.DeviceName,
		"u_assignee_name":     ms.AssigneeName,
		"u_next_due":          ms.NextDue.Format("2006-01-02 15:04:05"),
	}
	if ms.CustomCron != "" {
		body["u_custom_cron"] = ms.CustomCron
	}
	if ms.AssignedTo != nil {
		body["u_assigned_to"] = *ms.AssignedTo
	}
	if len(ms.Checklist) > 0 {
		body["u_checklist"] = string(ms.Checklist)
	}
	return body
}

// toTechnicianAssignmentSNBody преобразует models.TechnicianSiteAssignment в ServiceNow map.
func toTechnicianAssignmentSNBody(a *models.TechnicianSiteAssignment) map[string]interface{} {
	return map[string]interface{}{
		"u_technician_id":   a.TechnicianID,
		"u_site_id":         a.SiteID,
		"u_is_primary":      a.IsPrimary,
		"u_assigned_by":     a.AssignedBy,
		"u_technician_name": a.TechnicianName,
		"u_site_name":       a.SiteName,
	}
}

// toTechnicianAssignment преобразует map в models.TechnicianSiteAssignment.
func toTechnicianAssignment(raw map[string]interface{}) models.TechnicianSiteAssignment {
	m := newMapper(raw)
	return models.TechnicianSiteAssignment{
		ID:             m.str("sys_id"),
		TechnicianID:   m.str("u_technician_id"),
		SiteID:         m.str("u_site_id"),
		IsPrimary:      m.bool("u_is_primary"),
		AssignedBy:     m.str("u_assigned_by"),
		TechnicianName: m.str("u_technician_name"),
		SiteName:       m.str("u_site_name"),
		AssignedAt:     m.time("sys_created_on"),
	}
}

// toTechnicianWorkload преобразует ServiceNow-ответ в models.TechnicianWorkload.
func toTechnicianWorkload(raw map[string]interface{}) models.TechnicianWorkload {
	m := newMapper(raw)
	return models.TechnicianWorkload{
		UserID:          m.str("u_user_id"),
		UserName:        m.str("u_user_name"),
		CurrentWorkload: m.int("u_current_workload"),
		MaxWorkload:     m.int("u_max_workload"),
		Skills:          m.strSlice("u_skills"),
		BaseLocation:    m.str("u_base_location"),
	}
}

// toTechnicianMonthlyStats преобразует ServiceNow-ответ в models.TechnicianMonthlyStats.
func toTechnicianMonthlyStats(raw map[string]interface{}) models.TechnicianMonthlyStats {
	m := newMapper(raw)
	return models.TechnicianMonthlyStats{
		CompletedThisMonth: m.int("u_completed_this_month"),
		TotalWorkOrders:    m.int("u_total_work_orders"),
		OnTimePercent:      m.float("u_on_time_percent"),
		AvgRating:          m.float("u_avg_rating"),
	}
}

// toMaintenanceReport преобразует ServiceNow-ответ в models.MaintenanceReport.
func toMaintenanceReport(raw map[string]interface{}) models.MaintenanceReport {
	m := newMapper(raw)
	return models.MaintenanceReport{
		DeviceID:        m.str("u_device_id"),
		DeviceName:      m.str("u_device_name"),
		MTBF:            m.float("u_mtbf_hours"),
		MTTR:            m.float("u_mttr_minutes"),
		TotalWorkOrders: m.int("u_total_work_orders"),
		CompletedCount:  m.int("u_completed_count"),
		OverdueCount:    m.int("u_overdue_count"),
		TotalCost:       m.float("u_total_cost"),
	}
}

// toSLAComplianceReport преобразует ServiceNow-ответ в models.SLAComplianceReport.
func toSLAComplianceReport(raw map[string]interface{}) models.SLAComplianceReport {
	m := newMapper(raw)
	return models.SLAComplianceReport{
		Priority:          m.str("u_priority"),
		TotalWorkOrders:   m.int("u_total_work_orders"),
		WithinSLA:         m.int("u_within_sla"),
		BreachedSLA:       m.int("u_breached_sla"),
		CompliancePercent: m.float("u_compliance_percent"),
		AvgResponseTime:   m.float("u_avg_response_minutes"),
		AvgResolutionTime: m.float("u_avg_resolution_minutes"),
	}
}
