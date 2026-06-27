// Package compliance — unit tests for Regional Compliance Reports (P2-CR.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.2: Compliance Reports Tests
//
// Соответствие:
//   - ISO 27001 A.14.2 (Security testing — table-driven)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - СТБ 34.101.27 п. 7.4 (Тестирование безопасности)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// TestNewReportGenerator
// ═══════════════════════════════════════════════════════════════════════════

func TestNewReportGenerator(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	if gen == nil {
		t.Fatal("NewReportGenerator must not return nil")
	}

	if gen.registry == nil {
		t.Error("NewReportGenerator: registry should not be nil")
	}

	if gen.schedules == nil {
		t.Error("NewReportGenerator: schedules map should be initialized")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GenerateReport — PDF and XML output
// ═══════════════════════════════════════════════════════════════════════════

func TestGenerateReportPDF(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	pdfData, err := gen.GenerateReport(RegionBY, FormatPDF)
	if err != nil {
		t.Fatalf("GenerateReport(BY, PDF) error: %v", err)
	}

	if len(pdfData) == 0 {
		t.Fatal("GenerateReport(BY, PDF) returned empty data")
	}

	// Проверяем PDF signature
	if len(pdfData) < 5 || string(pdfData[:5]) != "%PDF-" {
		t.Errorf("GenerateReport(BY, PDF) output does not start with %%PDF-, got %q",
			string(pdfData[:min(20, len(pdfData))]))
	}
}

func TestGenerateReportXML(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	xmlData, err := gen.GenerateReport(RegionEU, FormatXML)
	if err != nil {
		t.Fatalf("GenerateReport(EU, XML) error: %v", err)
	}

	if len(xmlData) == 0 {
		t.Fatal("GenerateReport(EU, XML) returned empty data")
	}

	// Проверяем XML signature
	output := string(xmlData)
	if len(output) < 20 || output[:5] != "<?xml" {
		t.Errorf("GenerateReport(EU, XML) output does not start with <?xml, got %q",
			output[:min(30, len(output))])
	}

	// Проверяем регион в XML
	if !contains(output, "EU") {
		t.Error("GenerateReport(EU, XML) should contain region code EU")
	}
}

func TestGenerateReportUnsupportedFormat(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	_, err := gen.GenerateReport(RegionBY, "csv")
	if err == nil {
		t.Fatal("GenerateReport with unsupported format should return error")
	}
}

func TestGenerateReportUnknownRegion(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	// Unknown region falls back to INTL profile gracefully
	data, err := gen.GenerateReport("XX", FormatPDF)
	if err != nil {
		t.Fatalf("GenerateReport(XX, PDF) should fallback to INTL, got error: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("GenerateReport(XX, PDF) returned empty data")
	}
}

func TestGenerateReportINTL(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	pdfData, err := gen.GenerateReport(RegionINTL, FormatPDF)
	if err != nil {
		t.Fatalf("GenerateReport(INTL, PDF) error: %v", err)
	}

	if len(pdfData) == 0 {
		t.Fatal("GenerateReport(INTL, PDF) returned empty data")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetComplianceDashboard
// ═══════════════════════════════════════════════════════════════════════════

func TestGetComplianceDashboard(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	dashboard, err := gen.GetComplianceDashboard("tenant-001")
	if err != nil {
		t.Fatalf("GetComplianceDashboard error: %v", err)
	}

	if dashboard == nil {
		t.Fatal("GetComplianceDashboard returned nil")
	}

	if dashboard.TenantID != "tenant-001" {
		t.Errorf("TenantID = %q, want %q", dashboard.TenantID, "tenant-001")
	}

	if len(dashboard.Regions) == 0 {
		t.Error("GetComplianceDashboard should return at least one region")
	}

	if dashboard.GeneratedAt.IsZero() {
		t.Error("GetComplianceDashboard GeneratedAt should not be zero")
	}

	// Проверяем структуру регионов
	for _, rc := range dashboard.Regions {
		if rc.Region == "" {
			t.Error("RegionCompliance should have non-empty Region")
		}
		if rc.Score < 0 || rc.Score > 100 {
			t.Errorf("RegionCompliance Score out of range [0,100]: %f", rc.Score)
		}
	}
}

func TestGetComplianceDashboardEmptyTenant(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	_, err := gen.GetComplianceDashboard("")
	if err == nil {
		t.Fatal("GetComplianceDashboard with empty tenantID should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ScheduleReport
// ═══════════════════════════════════════════════════════════════════════════

func TestScheduleReport(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	err := gen.ScheduleReport(RegionBY, "0 0 * * 1", FormatPDF)
	if err != nil {
		t.Fatalf("ScheduleReport error: %v", err)
	}

	schedules := gen.ListSchedules()
	if len(schedules) != 1 {
		t.Errorf("ListSchedules length = %d, want 1", len(schedules))
	}

	if schedules[0].Region != RegionBY {
		t.Errorf("Schedule region = %s, want %s", schedules[0].Region, RegionBY)
	}

	if !schedules[0].Enabled {
		t.Error("Schedule should be enabled by default")
	}
}

func TestScheduleReportDuplicate(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	_ = gen.ScheduleReport(RegionBY, "0 0 * * 1", FormatPDF)
	err := gen.ScheduleReport(RegionBY, "0 0 * * 1", FormatPDF)
	if err == nil {
		t.Fatal("ScheduleReport duplicate should return error")
	}
}

func TestScheduleReportEmptyRegion(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	err := gen.ScheduleReport("", "0 0 * * 1", FormatPDF)
	if err == nil {
		t.Fatal("ScheduleReport with empty region should return error")
	}
}

func TestScheduleReportEmptyCron(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	err := gen.ScheduleReport(RegionBY, "", FormatPDF)
	if err == nil {
		t.Fatal("ScheduleReport with empty cron should return error")
	}
}

func TestUnscheduleReport(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	_ = gen.ScheduleReport(RegionBY, "0 0 * * 1", FormatPDF)
	gen.UnscheduleReport(RegionBY, FormatPDF)

	schedules := gen.ListSchedules()
	if len(schedules) != 0 {
		t.Errorf("After unschedule, ListSchedules length = %d, want 0", len(schedules))
	}
}

func TestListSchedulesOrder(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	_ = gen.ScheduleReport(RegionEU, "0 0 * * 1", FormatPDF)
	_ = gen.ScheduleReport(RegionBY, "0 0 * * 1", FormatXML)

	schedules := gen.ListSchedules()
	if len(schedules) != 2 {
		t.Fatalf("ListSchedules length = %d, want 2", len(schedules))
	}

	// Must be sorted by region
	if schedules[0].Region != RegionBY {
		t.Errorf("First schedule should be BY, got %s", schedules[0].Region)
	}
	if schedules[1].Region != RegionEU {
		t.Errorf("Second schedule should be EU, got %s", schedules[1].Region)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Gap Analysis
// ═══════════════════════════════════════════════════════════════════════════

func TestRunGapAnalysis(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	analysis, err := gen.RunGapAnalysis(RegionBY)
	if err != nil {
		t.Fatalf("RunGapAnalysis(BY) error: %v", err)
	}

	if analysis == nil {
		t.Fatal("RunGapAnalysis returned nil")
	}

	if analysis.TotalGaps == 0 {
		t.Error("RunGapAnalysis(BY) should find gaps")
	}

	// BY should have extra crypto gap
	if _, ok := analysis.BySeverity[SeverityCritical]; !ok {
		t.Error("Gap analysis should have critical severity entries")
	}
}

func TestRunGapAnalysisUnknownRegion(t *testing.T) {
	registry := RegisterBaselineProfiles(nil)
	gen := NewReportGenerator(registry, nil)

	// Unknown region falls back to INTL profile gracefully
	analysis, err := gen.RunGapAnalysis("XX")
	if err != nil {
		t.Fatalf("RunGapAnalysis(XX) should fallback to INTL, got error: %v", err)
	}

	if analysis == nil {
		t.Fatal("RunGapAnalysis returned nil")
	}

	if analysis.TotalGaps == 0 {
		t.Error("RunGapAnalysis(XX) should find gaps via INTL fallback")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ComplianceReport structure
// ═══════════════════════════════════════════════════════════════════════════

func TestComplianceReportStructure(t *testing.T) {
	now := time.Now().UTC()
	report := &ComplianceReport{
		ID:          "CR-TEST-001",
		Region:      RegionBY,
		GeneratedAt: now,
		Status:      StatusCompliant,
		Summary: ReportSummary{
			TotalChecks:       100,
			PassedChecks:      85,
			FailedChecks:      15,
			CompliancePercent: 85.0,
		},
	}

	if report.ID != "CR-TEST-001" {
		t.Errorf("Report ID = %s, want CR-TEST-001", report.ID)
	}
	if report.Status != StatusCompliant {
		t.Errorf("Report Status = %s, want compliant", report.Status)
	}
	if report.Summary.CompliancePercent != 85.0 {
		t.Errorf("CompliancePercent = %f, want 85.0", report.Summary.CompliancePercent)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ═══════════════════════════════════════════════════════════════════════════
// Error handling
// ═══════════════════════════════════════════════════════════════════════════

func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		score float64
		want  ComplianceStatus
	}{
		{95.0, StatusCompliant},
		{90.0, StatusCompliant},
		{89.99, StatusPartial},
		{75.0, StatusPartial},
		{60.0, StatusPartial},
		{59.99, StatusNonCompliant},
		{30.0, StatusNonCompliant},
		{0.0, StatusNotAssessed},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := determineOverallStatus(tt.score)
			if got != tt.want {
				t.Errorf("determineOverallStatus(%f) = %s, want %s", tt.score, got, tt.want)
			}
		})
	}
}
