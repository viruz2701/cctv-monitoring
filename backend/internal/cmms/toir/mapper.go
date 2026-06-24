package toir

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
				"2006-01-02 15:04:05",
				"2006-01-02T15:04:05",
				"2006-01-02",
				"02.01.2006 15:04:05",
				"02.01.2006",
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

// 1C:TOIR API paths.
const (
	pathWorkOrders            = "/api/v1/work-orders"
	pathSpareParts            = "/api/v1/spare-parts"
	pathMaintenanceSchedules  = "/api/v1/maintenance/schedules"
	pathSLAConfig             = "/api/v1/sla/config"
	pathTechnicians           = "/api/v1/technicians"
	pathTechnicianAssignments = "/api/v1/technician-assignments"
	pathReports               = "/api/v1/reports"
	pathMobilePushToken       = "/api/v1/mobile/push-token"
	pathAssets                = "/api/v1/assets"
	pathHealth                = "/api/v1/health"
)

// toirResponse — стандартная обёртка ответа 1С:ТОИР.
type toirResponse struct {
	Data  []map[string]interface{} `json:"data"`
	Total int                      `json:"total"`
}

// toirSingleResponse — ответ для одного объекта.
type toirSingleResponse struct {
	Data map[string]interface{} `json:"data"`
}

// buildQueryPath добавляет query-параметры из фильтров.
func buildQueryPath(base string, filters map[string]interface{}) string {
	if len(filters) == 0 {
		return base
	}
	parts := make([]string, 0, len(filters))
	for k, v := range filters {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return base + "?" + strings.Join(parts, "&")
}

// ── Model → 1C:TOIR body mappers ─────────────────────────────────

func toWorkOrderTOIRBody(wo *models.WorkOrder) map[string]interface{} {
	body := map[string]interface{}{
		"device_id":     wo.DeviceID,
		"type":          wo.Type,
		"status":        wo.Status,
		"priority":      wo.Priority,
		"notes":         wo.Notes,
		"device_name":   wo.DeviceName,
		"assignee_name": wo.AssigneeName,
		"sla_status":    wo.SLAStatus,
	}
	if wo.ScheduleID != nil {
		body["schedule_id"] = *wo.ScheduleID
	}
	if wo.AssignedTo != nil {
		body["assigned_to"] = *wo.AssignedTo
	}
	if wo.CreatedBy != nil {
		body["created_by"] = *wo.CreatedBy
	}
	if len(wo.Checklist) > 0 {
		body["checklist"] = string(wo.Checklist)
	}
	if len(wo.Photos) > 0 {
		body["photos"] = string(wo.Photos)
	}
	if len(wo.PartsUsed) > 0 {
		body["parts_used"] = string(wo.PartsUsed)
	}
	return body
}

func toSparePartTOIRBody(sp *models.SparePart) map[string]interface{} {
	return map[string]interface{}{
		"name":               sp.Name,
		"sku":                sp.SKU,
		"category":           sp.Category,
		"stock":              sp.Stock,
		"min_stock":          sp.MinStock,
		"location":           sp.Location,
		"cost":               sp.Cost,
		"supplier":           sp.Supplier,
		"compatible_devices": sp.CompatibleDevices,
	}
}

func toMaintenanceScheduleTOIRBody(ms *models.MaintenanceSchedule) map[string]interface{} {
	body := map[string]interface{}{
		"device_id":         ms.DeviceID,
		"schedule_type":     ms.ScheduleType,
		"interval_days":     ms.IntervalDays,
		"estimated_minutes": ms.EstimatedMinutes,
		"priority":          ms.Priority,
		"notes":             ms.Notes,
		"device_name":       ms.DeviceName,
		"assignee_name":     ms.AssigneeName,
		"next_due":          ms.NextDue.Format("2006-01-02 15:04:05"),
	}
	if ms.CustomCron != "" {
		body["custom_cron"] = ms.CustomCron
	}
	if ms.AssignedTo != nil {
		body["assigned_to"] = *ms.AssignedTo
	}
	if len(ms.Checklist) > 0 {
		body["checklist"] = string(ms.Checklist)
	}
	return body
}

func toTechnicianAssignmentTOIRBody(a *models.TechnicianSiteAssignment) map[string]interface{} {
	return map[string]interface{}{
		"technician_id":   a.TechnicianID,
		"site_id":         a.SiteID,
		"is_primary":      a.IsPrimary,
		"assigned_by":     a.AssignedBy,
		"technician_name": a.TechnicianName,
		"site_name":       a.SiteName,
	}
}

// ── 1C:TOIR response → Model mappers ──────────────────────────────

func toWorkOrder(raw map[string]interface{}) models.WorkOrder {
	wo := models.WorkOrder{}
	m := newMapper(raw)
	wo.ID = m.str("id")
	wo.DeviceID = m.str("device_id")
	wo.Type = m.str("type")
	wo.Status = m.str("status")
	wo.Priority = m.str("priority")
	wo.Notes = m.str("notes")
	wo.SLAStatus = m.str("sla_status")
	wo.DeviceName = m.str("device_name")
	wo.AssigneeName = m.str("assignee_name")
	if v := m.str("schedule_id"); v != "" {
		wo.ScheduleID = &v
	}
	if v := m.str("assigned_to"); v != "" {
		wo.AssignedTo = &v
	}
	if v := m.str("created_by"); v != "" {
		wo.CreatedBy = &v
	}
	wo.CreatedAt = m.time("created_at")
	wo.UpdatedAt = m.time("updated_at")
	wo.StartedAt = m.timePtr("started_at")
	wo.CompletedAt = m.timePtr("completed_at")
	wo.SLADeadline = m.timePtr("sla_deadline")
	wo.Checklist = m.rawJSON("checklist")
	wo.Photos = m.rawJSON("photos")
	wo.PartsUsed = m.rawJSON("parts_used")
	return wo
}

func toSparePart(raw map[string]interface{}) models.SparePart {
	sp := models.SparePart{}
	m := newMapper(raw)
	sp.ID = m.str("id")
	sp.Name = m.str("name")
	sp.SKU = m.str("sku")
	sp.Category = m.str("category")
	sp.Stock = m.int("stock")
	sp.MinStock = m.int("min_stock")
	sp.Location = m.str("location")
	sp.Cost = m.float("cost")
	sp.Supplier = m.str("supplier")
	sp.CreatedAt = m.time("created_at")
	sp.UpdatedAt = m.time("updated_at")
	sp.CompatibleDevices = m.strSlice("compatible_devices")
	return sp
}

func toMaintenanceSchedule(raw map[string]interface{}) models.MaintenanceSchedule {
	ms := models.MaintenanceSchedule{}
	m := newMapper(raw)
	ms.ID = m.str("id")
	ms.DeviceID = m.str("device_id")
	ms.ScheduleType = m.str("schedule_type")
	ms.IntervalDays = m.int("interval_days")
	ms.CustomCron = m.str("custom_cron")
	ms.EstimatedMinutes = m.int("estimated_minutes")
	ms.Priority = m.str("priority")
	ms.Notes = m.str("notes")
	ms.DeviceName = m.str("device_name")
	ms.AssigneeName = m.str("assignee_name")
	if v := m.str("assigned_to"); v != "" {
		ms.AssignedTo = &v
	}
	ms.CreatedAt = m.time("created_at")
	ms.UpdatedAt = m.time("updated_at")
	ms.LastCompleted = m.timePtr("last_completed")
	ms.NextDue = m.time("next_due")
	ms.Checklist = m.rawJSON("checklist")
	return ms
}

func toSLAConfig(raw map[string]interface{}) models.SLAConfig {
	m := newMapper(raw)
	return models.SLAConfig{
		ID:                    m.str("id"),
		Priority:              m.str("priority"),
		ResponseTimeMinutes:   m.int("response_time_minutes"),
		ResolutionTimeMinutes: m.int("resolution_time_minutes"),
	}
}

func toTechnicianWorkload(raw map[string]interface{}) models.TechnicianWorkload {
	m := newMapper(raw)
	return models.TechnicianWorkload{
		UserID:          m.str("user_id"),
		UserName:        m.str("user_name"),
		CurrentWorkload: m.int("current_workload"),
		MaxWorkload:     m.int("max_workload"),
		Skills:          m.strSlice("skills"),
		BaseLocation:    strPtr(m.str("base_location")),
	}
}

// strPtr возвращает *string, или nil если строка пустая.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toTechnicianMonthlyStats(raw map[string]interface{}) models.TechnicianMonthlyStats {
	m := newMapper(raw)
	return models.TechnicianMonthlyStats{
		CompletedThisMonth: m.int("completed_this_month"),
		TotalWorkOrders:    m.int("total_work_orders"),
		OnTimePercent:      m.float("on_time_percent"),
		AvgRating:          m.float("avg_rating"),
	}
}

func toMaintenanceReport(raw map[string]interface{}) models.MaintenanceReport {
	m := newMapper(raw)
	return models.MaintenanceReport{
		DeviceID:        m.str("device_id"),
		DeviceName:      m.str("device_name"),
		MTBF:            m.float("mtbf_hours"),
		MTTR:            m.float("mttr_minutes"),
		TotalWorkOrders: m.int("total_work_orders"),
		CompletedCount:  m.int("completed_count"),
		OverdueCount:    m.int("overdue_count"),
		TotalCost:       m.float("total_cost"),
	}
}

func toSLAComplianceReport(raw map[string]interface{}) models.SLAComplianceReport {
	m := newMapper(raw)
	return models.SLAComplianceReport{
		Priority:          m.str("priority"),
		TotalWorkOrders:   m.int("total_work_orders"),
		WithinSLA:         m.int("within_sla"),
		BreachedSLA:       m.int("breached_sla"),
		CompliancePercent: m.float("compliance_percent"),
		AvgResponseTime:   m.float("avg_response_minutes"),
		AvgResolutionTime: m.float("avg_resolution_minutes"),
	}
}

func toTechnicianAssignment(raw map[string]interface{}) models.TechnicianSiteAssignment {
	m := newMapper(raw)
	return models.TechnicianSiteAssignment{
		ID:             m.str("id"),
		TechnicianID:   m.str("technician_id"),
		SiteID:         m.str("site_id"),
		IsPrimary:      m.bool("is_primary"),
		AssignedBy:     m.str("assigned_by"),
		TechnicianName: m.str("technician_name"),
		SiteName:       m.str("site_name"),
		AssignedAt:     m.time("created_at"),
	}
}
