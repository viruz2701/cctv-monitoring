package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Compliance & Fines Shield (KF-15.1.1)
// ═══════════════════════════════════════════════════════════════════════

// ComplianceRiskRow представляет строку compliance_risks из БД.
type ComplianceRiskRow struct {
	DeviceID         string    `db:"device_id"`
	SiteID           string    `db:"site_id"`
	DeviceType       string    `db:"device_type"`
	TotalDowntimeMin int64     `db:"total_downtime_min"`
	HourlyFine       float64   `db:"hourly_fine"`
	TotalExposure    float64   `db:"total_exposure"`
	RiskLevel        string    `db:"risk_level"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// ComplianceSummaryRow представляет агрегированную сводку из БД.
type ComplianceSummaryRow struct {
	TotalExposure    float64 `db:"total_exposure"`
	AtRiskDevices    int     `db:"at_risk_devices"`
	CompliantDevices int     `db:"compliant_devices"`
	TotalDevices     int     `db:"total_devices"`
}

// GetComplianceRisks возвращает все compliance риски с опциональной фильтрацией.
// Соответствует: OWASP ASVS V5.2 (parameterized queries), ISO 27001 A.12.4 (audit).
func (db *DB) GetComplianceRisks(ctx context.Context, deviceID, siteID string) ([]ComplianceRiskRow, error) {
	query := `
		SELECT device_id, COALESCE(site_id, '') as site_id, device_type,
		       total_downtime_min, hourly_fine, total_exposure, risk_level, updated_at
		FROM compliance_risks
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argIdx := 1

	if deviceID != "" {
		query += fmt.Sprintf(" AND device_id = $%d", argIdx)
		args = append(args, deviceID)
		argIdx++
	}
	if siteID != "" {
		query += fmt.Sprintf(" AND site_id = $%d", argIdx)
		args = append(args, siteID)
		argIdx++
	}

	query += " ORDER BY total_exposure DESC"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query compliance_risks: %w", err)
	}
	defer rows.Close()

	var risks []ComplianceRiskRow
	for rows.Next() {
		var r ComplianceRiskRow
		if err := rows.Scan(&r.DeviceID, &r.SiteID, &r.DeviceType,
			&r.TotalDowntimeMin, &r.HourlyFine, &r.TotalExposure, &r.RiskLevel, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan compliance_risk: %w", err)
		}
		risks = append(risks, r)
	}
	return risks, rows.Err()
}

// GetComplianceSummary возвращает агрегированную сводку compliance рисков.
func (db *DB) GetComplianceSummary(ctx context.Context, siteID string) (*ComplianceSummaryRow, error) {
	query := `
		SELECT
			COALESCE(SUM(total_exposure), 0) as total_exposure,
			COUNT(*) FILTER (WHERE total_downtime_min >= 60 AND total_exposure > 0)::int as at_risk_devices,
			COUNT(*) FILTER (WHERE total_downtime_min < 60 OR total_exposure = 0)::int as compliant_devices,
			COUNT(*)::int as total_devices
		FROM compliance_risks
	`
	args := make([]interface{}, 0)
	argIdx := 1

	if siteID != "" {
		query += fmt.Sprintf(" WHERE site_id = $%d", argIdx)
		args = append(args, siteID)
	}

	var summary ComplianceSummaryRow
	err := db.Pool.QueryRow(ctx, query, args...).Scan(
		&summary.TotalExposure,
		&summary.AtRiskDevices,
		&summary.CompliantDevices,
		&summary.TotalDevices,
	)
	if err != nil {
		return nil, fmt.Errorf("query compliance summary: %w", err)
	}
	return &summary, nil
}

// GetComplianceRiskBreakdown возвращает разбивку по уровням риска.
func (db *DB) GetComplianceRiskBreakdown(ctx context.Context, siteID string) (map[string]int, error) {
	query := `
		SELECT risk_level, COUNT(*)::int as cnt
		FROM compliance_risks
	`
	args := make([]interface{}, 0)
	argIdx := 1

	if siteID != "" {
		query += fmt.Sprintf(" WHERE site_id = $%d", argIdx)
		args = append(args, siteID)
	}

	query += " GROUP BY risk_level ORDER BY risk_level"

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query compliance breakdown: %w", err)
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scan compliance breakdown: %w", err)
		}
		breakdown[level] = count
	}
	return breakdown, rows.Err()
}

// LogComplianceAudit записывает audit-log для compliance (ISO 27001 A.12.4).
func (db *DB) LogComplianceAudit(ctx context.Context, deviceID, siteID string, totalExposure float64, riskLevel, traceID string, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal compliance audit details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO compliance_audit_log (device_id, site_id, total_exposure, risk_level, details, trace_id)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, deviceID, siteID, totalExposure, riskLevel, detailsJSON, traceID)
	if err != nil {
		return fmt.Errorf("insert compliance audit: %w", err)
	}
	return nil
}
