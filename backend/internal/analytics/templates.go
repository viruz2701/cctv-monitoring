// Package analytics — default BI query templates for CCTV Health Monitor.
//
// P2-BI: Self-Service Analytics Templates
//
// Каждый шаблон содержит:
//   - SQL: параметризованный запрос (подзапрос для внешней GROUP BY)
//   - Dimensions: поля для группировки (измерения)
//   - Measures: поля для агрегации (метрики)
//   - DateField: колонка для фильтра по времени
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — MTTR/MTBF метрики)
//   - ISO 27001 A.12.6.1 (Capacity management — reliability metrics)
//   - IEC 62443 SR 7.1 (Resource availability — device uptime)
//   - IEC 62443 SR 7.3 (Auditability — work order tracking)
package analytics

// DefaultTemplates возвращает стандартные BI-шаблоны для CCTV Health Monitor.
func DefaultTemplates() []QueryTemplate {
	return []QueryTemplate{
		mttrByDevice(),
		mtbfByDevice(),
		deviceUptime(),
		workOrderSummary(),
		tcoByDevice(),
		downtimeBySite(),
		alarmByType(),
		healthByVendor(),
	}
}

// mttrByDevice — Mean Time To Repair по устройствам.
//
// Используется для: SLA compliance, reliability reporting
// Поля: device_id, device_name, vendor_type, device_type, technician_name
// Метрики: avg_mttr_min, max_mttr_min, min_mttr_min, wo_count
func mttrByDevice() QueryTemplate {
	return QueryTemplate{
		ID:          "mttr_by_device",
		Name:        "MTTR by Device",
		Description: "Mean Time To Repair (minutes) grouped by device — ключевой SLA-показатель для CMMS",
		SQL: `SELECT
			wo.device_id,
			d.name AS device_name,
			d.vendor_type,
			d.device_type,
			COALESCE(t.full_name, 'Unassigned') AS technician_name,
			EXTRACT(EPOCH FROM (wo.resolved_at - wo.started_at)) / 60 AS resolution_time_min
		FROM work_orders wo
		JOIN devices d ON d.device_id = wo.device_id
		LEFT JOIN technicians t ON t.id = wo.technician_id
		WHERE wo.status = 'resolved'
		  AND wo.started_at IS NOT NULL
		  AND wo.resolved_at IS NOT NULL`,
		Dimensions: []Field{
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "device_type", Label: "Device Type", Type: "string"},
			{Key: "technician_name", Label: "Technician", Type: "string"},
		},
		Measures: []Field{
			{Key: "resolution_time_min", Label: "Resolution Time (min)", Type: "number"},
			{Key: "avg_mttr_min", Label: "Avg MTTR (min)", Type: "number", AggFunction: "AVG", SQLExpr: "resolution_time_min"},
			{Key: "max_mttr_min", Label: "Max MTTR (min)", Type: "number", AggFunction: "MAX", SQLExpr: "resolution_time_min"},
			{Key: "min_mttr_min", Label: "Min MTTR (min)", Type: "number", AggFunction: "MIN", SQLExpr: "resolution_time_min"},
			{Key: "wo_count", Label: "Work Orders", Type: "number", AggFunction: "COUNT", SQLExpr: "1"},
		},
		DateField: "wo.resolved_at",
	}
}

// mtbfByDevice — Mean Time Between Failures по устройствам.
func mtbfByDevice() QueryTemplate {
	return QueryTemplate{
		ID:          "mtbf_by_device",
		Name:        "MTBF by Device",
		Description: "Mean Time Between Failures (hours) — надёжность оборудования",
		SQL: `SELECT
			d.device_id,
			d.name AS device_name,
			d.vendor_type,
			d.device_type,
			EXTRACT(EPOCH FROM (d.last_seen - d.registered_at)) / 3600 AS uptime_hours,
			(SELECT COUNT(*) FROM work_orders wo2 WHERE wo2.device_id = d.device_id AND wo2.category = 'breakdown') AS failure_count
		FROM devices d
		WHERE d.last_seen IS NOT NULL
		  AND d.registered_at IS NOT NULL`,
		Dimensions: []Field{
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "device_type", Label: "Device Type", Type: "string"},
		},
		Measures: []Field{
			{Key: "uptime_hours", Label: "Uptime (hours)", Type: "number"},
			{Key: "failure_count", Label: "Failures", Type: "number"},
			{Key: "avg_uptime_hours", Label: "Avg Uptime (hours)", Type: "number", AggFunction: "AVG", SQLExpr: "uptime_hours"},
		},
		DateField: "d.registered_at",
	}
}

// deviceUptime — аптайм устройств с фильтром по статусу.
func deviceUptime() QueryTemplate {
	return QueryTemplate{
		ID:          "device_uptime",
		Name:        "Device Uptime",
		Description: "Текущий аптайм устройств и статус здоровья по сайтам",
		SQL: `SELECT
			d.device_id,
			d.name AS device_name,
			d.site_id,
			s.name AS site_name,
			d.status,
			d.health,
			d.vendor_type,
			d.device_type,
			d.last_seen,
			CASE
				WHEN d.last_seen IS NULL THEN 0
				WHEN d.last_seen < NOW() - INTERVAL '24 hours' THEN 0
				ELSE EXTRACT(EPOCH FROM (NOW() - d.last_seen)) / 3600
			END AS hours_since_last_seen
		FROM devices d
		LEFT JOIN sites s ON s.id = d.site_id`,
		Dimensions: []Field{
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
			{Key: "site_id", Label: "Site ID", Type: "string"},
			{Key: "site_name", Label: "Site Name", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
			{Key: "health", Label: "Health", Type: "string"},
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "device_type", Label: "Device Type", Type: "string"},
		},
		Measures: []Field{
			{Key: "hours_since_last_seen", Label: "Hours Since Last Seen", Type: "number"},
			{Key: "device_count", Label: "Device Count", Type: "number", AggFunction: "COUNT", SQLExpr: "1"},
			{Key: "avg_hours_since_last_seen", Label: "Avg Hours Since Last Seen", Type: "number", AggFunction: "AVG", SQLExpr: "hours_since_last_seen"},
		},
		DateField: "d.last_seen",
	}
}

// workOrderSummary — сводка по Work Orders.
func workOrderSummary() QueryTemplate {
	return QueryTemplate{
		ID:          "work_order_summary",
		Name:        "Work Order Summary",
		Description: "Агрегированная сводка по Work Orders: статусы, приоритеты, категории",
		SQL: `SELECT
			wo.id AS wo_id,
			wo.device_id,
			d.name AS device_name,
			d.site_id,
			s.name AS site_name,
			wo.status,
			wo.priority,
			wo.category,
			wo.technician_id,
			COALESCE(t.full_name, 'Unassigned') AS technician_name,
			EXTRACT(EPOCH FROM (COALESCE(wo.resolved_at, NOW()) - wo.created_at)) / 3600 AS age_hours,
			wo.total_cost
		FROM work_orders wo
		JOIN devices d ON d.device_id = wo.device_id
		LEFT JOIN sites s ON s.id = d.site_id
		LEFT JOIN technicians t ON t.id = wo.technician_id`,
		Dimensions: []Field{
			{Key: "wo_id", Label: "Work Order ID", Type: "string"},
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
			{Key: "site_id", Label: "Site ID", Type: "string"},
			{Key: "site_name", Label: "Site Name", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
			{Key: "priority", Label: "Priority", Type: "string"},
			{Key: "category", Label: "Category", Type: "string"},
			{Key: "technician_name", Label: "Technician", Type: "string"},
		},
		Measures: []Field{
			{Key: "age_hours", Label: "Age (hours)", Type: "number"},
			{Key: "total_cost", Label: "Total Cost", Type: "number"},
			{Key: "wo_count", Label: "Work Orders", Type: "number", AggFunction: "COUNT", SQLExpr: "1"},
			{Key: "total_cost_sum", Label: "Total Cost Sum", Type: "number", AggFunction: "SUM", SQLExpr: "total_cost"},
			{Key: "avg_cost", Label: "Avg Cost", Type: "number", AggFunction: "AVG", SQLExpr: "total_cost"},
			{Key: "max_age_hours", Label: "Max Age (hours)", Type: "number", AggFunction: "MAX", SQLExpr: "age_hours"},
		},
		DateField: "wo.created_at",
	}
}

// tcoByDevice — Total Cost of Ownership по устройствам.
func tcoByDevice() QueryTemplate {
	return QueryTemplate{
		ID:          "tco_by_device",
		Name:        "TCO by Device",
		Description: "Total Cost of Ownership: purchase + labor + parts + downtime",
		SQL: `SELECT
			d.device_id,
			d.name AS device_name,
			d.vendor_type,
			d.device_type,
			d.site_id,
			s.name AS site_name,
			COALESCE(tco.purchase_cost, 0) AS purchase_cost,
			COALESCE(tco.labor_cost, 0) AS labor_cost,
			COALESCE(tco.parts_cost, 0) AS parts_cost,
			COALESCE(tco.downtime_cost, 0) AS downtime_cost,
			COALESCE(tco.purchase_cost, 0) + COALESCE(tco.labor_cost, 0) + COALESCE(tco.parts_cost, 0) + COALESCE(tco.downtime_cost, 0) AS tco
		FROM devices d
		LEFT JOIN sites s ON s.id = d.site_id
		LEFT JOIN mv_tco_per_device tco ON tco.device_id = d.device_id`,
		Dimensions: []Field{
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "device_type", Label: "Device Type", Type: "string"},
			{Key: "site_id", Label: "Site ID", Type: "string"},
			{Key: "site_name", Label: "Site Name", Type: "string"},
		},
		Measures: []Field{
			{Key: "purchase_cost", Label: "Purchase Cost", Type: "number"},
			{Key: "labor_cost", Label: "Labor Cost", Type: "number"},
			{Key: "parts_cost", Label: "Parts Cost", Type: "number"},
			{Key: "downtime_cost", Label: "Downtime Cost", Type: "number"},
			{Key: "tco", Label: "Total TCO", Type: "number"},
			{Key: "avg_tco", Label: "Avg TCO", Type: "number", AggFunction: "AVG", SQLExpr: "tco"},
			{Key: "total_tco", Label: "Total TCO Sum", Type: "number", AggFunction: "SUM", SQLExpr: "tco"},
		},
		DateField: "d.registered_at",
	}
}

// downtimeBySite — стоимость простоев по объектам.
func downtimeBySite() QueryTemplate {
	return QueryTemplate{
		ID:          "downtime_by_site",
		Name:        "Downtime Cost by Site",
		Description: "Стоимость простоев с группировкой по объектам (BIZ-01)",
		SQL: `SELECT
			d.site_id,
			s.name AS site_name,
			d.device_id,
			d.name AS device_name,
			EXTRACT(EPOCH FROM (COALESCE(wo.resolved_at, NOW()) - wo.started_at)) / 3600 AS downtime_hours,
			COALESCE(wo.total_cost, 0) AS downtime_cost,
			COALESCE(s.downtime_cost_per_hour, 50) AS cost_per_hour
		FROM work_orders wo
		JOIN devices d ON d.device_id = wo.device_id
		LEFT JOIN sites s ON s.id = d.site_id
		WHERE wo.category IN ('breakdown', 'emergency')
		  AND wo.started_at IS NOT NULL`,
		Dimensions: []Field{
			{Key: "site_id", Label: "Site ID", Type: "string"},
			{Key: "site_name", Label: "Site Name", Type: "string"},
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
		},
		Measures: []Field{
			{Key: "downtime_hours", Label: "Downtime (hours)", Type: "number"},
			{Key: "downtime_cost", Label: "Downtime Cost", Type: "number"},
			{Key: "cost_per_hour", Label: "Cost per Hour", Type: "number"},
			{Key: "total_downtime_hours", Label: "Total Downtime (hours)", Type: "number", AggFunction: "SUM", SQLExpr: "downtime_hours"},
			{Key: "total_downtime_cost", Label: "Total Downtime Cost", Type: "number", AggFunction: "SUM", SQLExpr: "downtime_cost"},
			{Key: "avg_downtime_hours", Label: "Avg Downtime (hours)", Type: "number", AggFunction: "AVG", SQLExpr: "downtime_hours"},
		},
		DateField: "wo.started_at",
	}
}

// alarmByType — аналитика тревог по типам.
func alarmByType() QueryTemplate {
	return QueryTemplate{
		ID:          "alarm_by_type",
		Name:        "Alarms by Type",
		Description: "Распределение тревог по типам и устройствам за период",
		SQL: `SELECT
			a.event_type,
			a.severity,
			a.device_id,
			d.name AS device_name,
			d.vendor_type,
			d.site_id,
			s.name AS site_name,
			a.message,
			a.acknowledged,
			a.acknowledged_by
		FROM alarms a
		JOIN devices d ON d.device_id = a.device_id
		LEFT JOIN sites s ON s.id = d.site_id`,
		Dimensions: []Field{
			{Key: "event_type", Label: "Event Type", Type: "string"},
			{Key: "severity", Label: "Severity", Type: "string"},
			{Key: "device_id", Label: "Device ID", Type: "string"},
			{Key: "device_name", Label: "Device Name", Type: "string"},
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "site_id", Label: "Site ID", Type: "string"},
			{Key: "site_name", Label: "Site Name", Type: "string"},
			{Key: "acknowledged", Label: "Acknowledged", Type: "boolean"},
			{Key: "acknowledged_by", Label: "Acknowledged By", Type: "string"},
		},
		Measures: []Field{
			{Key: "alarm_count", Label: "Alarm Count", Type: "number", AggFunction: "COUNT", SQLExpr: "1"},
			{Key: "unacknowledged_count", Label: "Unacknowledged", Type: "number", AggFunction: "COUNT", SQLExpr: "CASE WHEN a.acknowledged = false THEN 1 END"},
		},
		DateField: "a.created_at",
	}
}

// healthByVendor — статистика здоровья устройств по вендорам.
func healthByVendor() QueryTemplate {
	return QueryTemplate{
		ID:          "health_by_vendor",
		Name:        "Health by Vendor",
		Description: "Статус здоровья устройств в разрезе производителей",
		SQL: `SELECT
			d.vendor_type,
			d.device_type,
			d.status,
			d.health,
			d.site_id,
			s.name AS site_name
		FROM devices d
		LEFT JOIN sites s ON s.id = d.site_id`,
		Dimensions: []Field{
			{Key: "vendor_type", Label: "Vendor", Type: "string"},
			{Key: "device_type", Label: "Device Type", Type: "string"},
			{Key: "status", Label: "Status", Type: "string"},
			{Key: "health", Label: "Health", Type: "string"},
			{Key: "site_id", Label: "Site ID", Type: "string"},
			{Key: "site_name", Label: "Site Name", Type: "string"},
		},
		Measures: []Field{
			{Key: "device_count", Label: "Device Count", Type: "number", AggFunction: "COUNT", SQLExpr: "1"},
			{Key: "online_count", Label: "Online", Type: "number", AggFunction: "COUNT", SQLExpr: "CASE WHEN d.status = 'online' THEN 1 END"},
			{Key: "offline_count", Label: "Offline", Type: "number", AggFunction: "COUNT", SQLExpr: "CASE WHEN d.status = 'offline' THEN 1 END"},
			{Key: "healthy_count", Label: "Healthy", Type: "number", AggFunction: "COUNT", SQLExpr: "CASE WHEN d.health = 'good' THEN 1 END"},
			{Key: "warning_count", Label: "Warning", Type: "number", AggFunction: "COUNT", SQLExpr: "CASE WHEN d.health = 'warning' THEN 1 END"},
			{Key: "critical_count", Label: "Critical", Type: "number", AggFunction: "COUNT", SQLExpr: "CASE WHEN d.health = 'critical' THEN 1 END"},
		},
		DateField: "d.last_seen",
	}
}
