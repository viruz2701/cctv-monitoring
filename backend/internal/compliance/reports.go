// Package compliance — Regional Compliance Reports (P2-CR.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.2: Regional Compliance Reports
//
// Проблема: Нет automated compliance reporting для разных регионов.
//
// Решение:
//   - ComplianceReport — структура отчёта с gap analysis
//   - GenerateReport(region) — генерация PDF/XML отчёта
//   - GetComplianceDashboard(tenantID) — real-time dashboard data
//   - ScheduleReport(region, cronExpr) — периодическая генерация
//   - Gap analysis с remediation recommendations
//
// Compliance:
//   - ISO 27001 A.12.4 (Audit trail — отчёты с timestamp)
//   - ISO 27001 A.8.2 (Information classification — regional reports)
//   - ISO 27019 PCC.A.12 (ICS compliance reporting)
//   - IEC 62443-3-3 SR 7.1 (Resource availability reporting)
//   - СТБ 34.101.27 п. 7.3 (Документирование compliance)
//   - OWASP ASVS V7 (Log content and integrity)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ═══════════════════════════════════════════════════════════════════════════
// Compliance status constants
// ═══════════════════════════════════════════════════════════════════════════

// ComplianceStatus represents the overall compliance status for a region.
type ComplianceStatus string

const (
	StatusCompliant    ComplianceStatus = "compliant"
	StatusPartial      ComplianceStatus = "partial"
	StatusNonCompliant ComplianceStatus = "non_compliant"
	StatusNotAssessed  ComplianceStatus = "not_assessed"
)

// SeverityLevel represents the severity of a compliance gap.
type SeverityLevel string

const (
	SeverityLow      SeverityLevel = "low"
	SeverityMedium   SeverityLevel = "medium"
	SeverityHigh     SeverityLevel = "high"
	SeverityCritical SeverityLevel = "critical"
)

// ═══════════════════════════════════════════════════════════════════════════
// ReportFormat
// ═══════════════════════════════════════════════════════════════════════════

// ReportFormat represents the output format for a compliance report.
type ReportFormat string

const (
	FormatPDF ReportFormat = "pdf"
	FormatXML ReportFormat = "xml"
)

// ═══════════════════════════════════════════════════════════════════════════
// Models
// ═══════════════════════════════════════════════════════════════════════════

// ComplianceReport представляет полный compliance отчёт для региона.
type ComplianceReport struct {
	XMLName      xml.Name         `json:"-" xml:"complianceReport"`
	ID           string           `json:"id" xml:"id,attr"`
	Region       string           `json:"region" xml:"region"`
	GeneratedAt  time.Time        `json:"generated_at" xml:"generatedAt"`
	Period       ReportPeriod     `json:"period" xml:"period"`
	Status       ComplianceStatus `json:"status" xml:"status"`
	Summary      ReportSummary    `json:"summary" xml:"summary"`
	Categories   []CategoryReport `json:"categories" xml:"categories>category"`
	GapAnalysis  GapAnalysis      `json:"gap_analysis" xml:"gapAnalysis"`
	Remediations []Remediation    `json:"remediations,omitempty" xml:"remediations>remediation"`
	SignedBy     string           `json:"signed_by,omitempty" xml:"signedBy,omitempty"`
	Format       ReportFormat     `json:"format" xml:"-"`
}

// ReportPeriod представляет период отчёта.
type ReportPeriod struct {
	Start time.Time `json:"start" xml:"start"`
	End   time.Time `json:"end" xml:"end"`
}

// ReportSummary содержит сводные показатели отчёта.
type ReportSummary struct {
	TotalChecks       int     `json:"total_checks" xml:"totalChecks"`
	PassedChecks      int     `json:"passed_checks" xml:"passedChecks"`
	FailedChecks      int     `json:"failed_checks" xml:"failedChecks"`
	Warnings          int     `json:"warnings" xml:"warnings"`
	CompliancePercent float64 `json:"compliance_percent" xml:"compliancePercent"`
	TotalDevices      int     `json:"total_devices" xml:"totalDevices"`
	AtRiskDevices     int     `json:"at_risk_devices" xml:"atRiskDevices"`
	TotalExposure     float64 `json:"total_exposure" xml:"totalExposure"`
}

// CategoryReport содержит compliance статус по категории.
type CategoryReport struct {
	Name         string           `json:"name" xml:"name,attr"`
	Status       ComplianceStatus `json:"status" xml:"status"`
	Score        float64          `json:"score" xml:"score"`
	ChecksPassed int              `json:"checks_passed" xml:"checksPassed"`
	ChecksTotal  int              `json:"checks_total" xml:"checksTotal"`
	Description  string           `json:"description,omitempty" xml:"description,omitempty"`
}

// GapAnalysis содержит анализ несоответствий.
type GapAnalysis struct {
	TotalGaps  int                   `json:"total_gaps" xml:"totalGaps"`
	BySeverity map[SeverityLevel]int `json:"by_severity" xml:"-"`
	Gaps       []GapItem             `json:"gaps,omitempty" xml:"gaps>gap"`
}

// GapItem представляет одно несоответствие.
type GapItem struct {
	ID          string        `json:"id" xml:"id,attr"`
	Category    string        `json:"category" xml:"category"`
	Title       string        `json:"title" xml:"title"`
	Description string        `json:"description" xml:"description"`
	Severity    SeverityLevel `json:"severity" xml:"severity"`
	Status      string        `json:"status" xml:"status"`
	DetectedAt  time.Time     `json:"detected_at" xml:"detectedAt"`
}

// Remediation представляет рекомендацию по устранению.
type Remediation struct {
	GapID      string `json:"gap_id" xml:"gapId,attr"`
	Action     string `json:"action" xml:"action"`
	Priority   string `json:"priority" xml:"priority"`
	TargetDate string `json:"target_date,omitempty" xml:"targetDate,omitempty"`
	AssignedTo string `json:"assigned_to,omitempty" xml:"assignedTo,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════
// Dashboard models
// ═══════════════════════════════════════════════════════════════════════════

// ComplianceDashboard представляет real-time dashboard данные.
type ComplianceDashboard struct {
	TenantID      string               `json:"tenant_id"`
	OverallStatus ComplianceStatus     `json:"overall_status"`
	OverallScore  float64              `json:"overall_score"`
	Regions       []RegionCompliance   `json:"regions"`
	RecentGaps    []GapItem            `json:"recent_gaps,omitempty"`
	RiskSummary   DashboardRiskSummary `json:"risk_summary"`
	GeneratedAt   time.Time            `json:"generated_at"`
}

// RegionCompliance содержит compliance статус для одного региона.
type RegionCompliance struct {
	Region    string           `json:"region"`
	Status    ComplianceStatus `json:"status"`
	Score     float64          `json:"score"`
	Devices   int              `json:"devices"`
	Exposure  float64          `json:"exposure"`
	Gaps      int              `json:"gaps"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// DashboardRiskSummary содержит сводку рисков для dashboard.
type DashboardRiskSummary struct {
	TotalExposure     float64        `json:"total_exposure"`
	AtRiskDevices     int            `json:"at_risk_devices"`
	CompliantDevices  int            `json:"compliant_devices"`
	TotalDevices      int            `json:"total_devices"`
	SeverityBreakdown map[string]int `json:"severity_breakdown"`
}

// ═══════════════════════════════════════════════════════════════════════════
// ReportGenerator
// ═══════════════════════════════════════════════════════════════════════════

// ReportGenerator генерирует compliance отчёты.
type ReportGenerator struct {
	mu        sync.RWMutex
	logger    *slog.Logger
	registry  *ProfileRegistry
	schedules map[string]*ReportSchedule
}

// ReportSchedule представляет запланированный отчёт.
type ReportSchedule struct {
	Region   string       `json:"region"`
	CronExpr string       `json:"cron_expr"`
	Format   ReportFormat `json:"format"`
	LastRun  time.Time    `json:"last_run"`
	NextRun  time.Time    `json:"next_run"`
	Enabled  bool         `json:"enabled"`
}

// NewReportGenerator создаёт новый ReportGenerator.
func NewReportGenerator(registry *ProfileRegistry, logger *slog.Logger) *ReportGenerator {
	if logger == nil {
		logger = slog.Default()
	}
	return &ReportGenerator{
		logger:    logger.With("component", "compliance.reports"),
		registry:  registry,
		schedules: make(map[string]*ReportSchedule),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GenerateReport — генерация отчёта
// ═══════════════════════════════════════════════════════════════════════════

// GenerateReport создаёт compliance отчёт для указанного региона.
//
// Параметры:
//   - region: код региона (BY, EU, INTL, и т.д.)
//   - format: формат отчёта (pdf, xml)
//
// Возвращает:
//   - []byte: сгенерированный отчёт
//   - error: ошибка генерации
func (g *ReportGenerator) GenerateReport(region string, format ReportFormat) ([]byte, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Проверяем, что профиль для региона существует
	_, err := g.registry.Get(region)
	if err != nil {
		return nil, fmt.Errorf("generate report: %w", err)
	}

	g.logger.Info("generating compliance report",
		"region", region,
		"format", format,
	)

	// Собираем данные для отчёта
	report := g.buildComplianceReport(region, format)

	switch format {
	case FormatPDF:
		return g.renderPDF(report)
	case FormatXML:
		return g.renderXML(report)
	default:
		return nil, fmt.Errorf("generate report: unsupported format: %s", format)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// GetComplianceDashboard — real-time dashboard data
// ═══════════════════════════════════════════════════════════════════════════

// GetComplianceDashboard возвращает real-time compliance данные для тенанта.
func (g *ReportGenerator) GetComplianceDashboard(tenantID string) (*ComplianceDashboard, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("compliance dashboard: tenantID cannot be empty")
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	regions := g.registry.List()
	regionCompliance := make([]RegionCompliance, 0, len(regions))
	totalScore := 0.0
	totalDevices := 0
	totalExposure := 0.0
	atRisk := 0
	compliant := 0

	for _, region := range regions {
		rc := g.buildRegionCompliance(region)
		regionCompliance = append(regionCompliance, rc)
		totalScore += rc.Score
		totalDevices += rc.Devices
		totalExposure += rc.Exposure
		atRisk += rc.Gaps
	}

	compliant = totalDevices - atRisk
	if compliant < 0 {
		compliant = 0
	}

	avgScore := 0.0
	if len(regions) > 0 {
		avgScore = math.Round(totalScore/float64(len(regions))*100) / 100
	}

	overallStatus := determineOverallStatus(avgScore)

	dashboard := &ComplianceDashboard{
		TenantID:      tenantID,
		OverallStatus: overallStatus,
		OverallScore:  avgScore,
		Regions:       regionCompliance,
		RiskSummary: DashboardRiskSummary{
			TotalExposure:    math.Round(totalExposure*100) / 100,
			AtRiskDevices:    atRisk,
			CompliantDevices: compliant,
			TotalDevices:     totalDevices,
			SeverityBreakdown: map[string]int{
				"low":      0,
				"medium":   0,
				"high":     0,
				"critical": 0,
			},
		},
		GeneratedAt: time.Now().UTC(),
	}

	g.logger.Info("compliance dashboard generated",
		"tenant", tenantID,
		"status", overallStatus,
		"score", avgScore,
		"regions", len(regions),
	)

	return dashboard, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ScheduleReport — периодическая генерация отчётов
// ═══════════════════════════════════════════════════════════════════════════

// ScheduleReport настраивает периодическую генерацию отчёта для региона.
//
// Параметры:
//   - region: код региона
//   - cronExpr: cron-выражение (например, "0 0 * * 1" — каждый понедельник)
//   - format: формат отчёта
func (g *ReportGenerator) ScheduleReport(region string, cronExpr string, format ReportFormat) error {
	if region == "" {
		return fmt.Errorf("schedule report: region cannot be empty")
	}
	if cronExpr == "" {
		return fmt.Errorf("schedule report: cron expression cannot be empty")
	}

	// Проверяем регион
	if _, err := g.registry.Get(region); err != nil {
		return fmt.Errorf("schedule report: %w", err)
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	key := region + "_" + string(format)
	if _, exists := g.schedules[key]; exists {
		return fmt.Errorf("schedule report: already scheduled for region %s format %s", region, format)
	}

	g.schedules[key] = &ReportSchedule{
		Region:   region,
		CronExpr: cronExpr,
		Format:   format,
		Enabled:  true,
	}

	g.logger.Info("report scheduled",
		"region", region,
		"cron", cronExpr,
		"format", format,
	)

	return nil
}

// UnscheduleReport отменяет запланированную генерацию отчёта.
func (g *ReportGenerator) UnscheduleReport(region string, format ReportFormat) {
	g.mu.Lock()
	defer g.mu.Unlock()

	key := region + "_" + string(format)
	delete(g.schedules, key)

	g.logger.Info("report unscheduled",
		"region", region,
		"format", format,
	)
}

// ListSchedules возвращает список всех запланированных отчётов.
func (g *ReportGenerator) ListSchedules() []*ReportSchedule {
	g.mu.RLock()
	defer g.mu.RUnlock()

	schedules := make([]*ReportSchedule, 0, len(g.schedules))
	for _, s := range g.schedules {
		schedules = append(schedules, s)
	}
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].Region < schedules[j].Region
	})
	return schedules
}

// ═══════════════════════════════════════════════════════════════════════════
// Gap Analysis
// ═══════════════════════════════════════════════════════════════════════════

// RunGapAnalysis выполняет gap analysis для указанного региона.
func (g *ReportGenerator) RunGapAnalysis(region string) (*GapAnalysis, error) {
	if _, err := g.registry.Get(region); err != nil {
		return nil, fmt.Errorf("gap analysis: %w", err)
	}

	gaps := g.identifyGaps(region)

	bySeverity := map[SeverityLevel]int{
		SeverityLow:      0,
		SeverityMedium:   0,
		SeverityHigh:     0,
		SeverityCritical: 0,
	}
	for _, gap := range gaps {
		bySeverity[gap.Severity]++
	}

	analysis := &GapAnalysis{
		TotalGaps:  len(gaps),
		BySeverity: bySeverity,
		Gaps:       gaps,
	}

	g.logger.Info("gap analysis completed",
		"region", region,
		"total_gaps", analysis.TotalGaps,
	)

	return analysis, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Internal — report building
// ═══════════════════════════════════════════════════════════════════════════

// buildComplianceReport собирает ComplianceReport из registry данных.
func (g *ReportGenerator) buildComplianceReport(region string, format ReportFormat) *ComplianceReport {
	profile, _ := g.registry.Get(region)

	now := time.Now().UTC()
	categories := g.buildCategoryReports(region)
	gaps := g.identifyGaps(region)

	summary := g.calculateSummary(categories)
	remediations := g.buildRemediations(gaps)

	report := &ComplianceReport{
		ID:          fmt.Sprintf("CR-%s-%s", region, now.Format("20060102-150405")),
		Region:      region,
		GeneratedAt: now,
		Period: ReportPeriod{
			Start: now.AddDate(0, -1, 0), // last month
			End:   now,
		},
		Status:     determineOverallStatus(summary.CompliancePercent),
		Summary:    summary,
		Categories: categories,
		GapAnalysis: GapAnalysis{
			TotalGaps:  len(gaps),
			BySeverity: countBySeverity(gaps),
			Gaps:       gaps,
		},
		Remediations: remediations,
		Format:       format,
	}

	if profile != nil {
		report.SignedBy = fmt.Sprintf("Compliance Profile: %s (%s)", profile.Name(), profile.Region())
	}

	return report
}

// buildCategoryReports создаёт отчёты по категориям compliance.
func (g *ReportGenerator) buildCategoryReports(region string) []CategoryReport {
	// Базовые категории compliance проверки
	categories := []struct {
		name         string
		description  string
		checksTotal  int
		checksPassed int
	}{
		{"cryptography", "Криптографическая защита (шифрование, хеширование, подписи)", 8, 6},
		{"authentication", "Аутентификация и управление доступом", 10, 8},
		{"audit_logging", "Аудит и логирование", 6, 5},
		{"data_protection", "Защита данных и data residency", 7, 6},
		{"network_security", "Сетевая безопасность и сегментация", 9, 7},
		{"incident_response", "Реагирование на инциденты", 5, 4},
		{"physical_security", "Физическая безопасность (IEC 62443)", 4, 4},
		{"retention", "Хранение и архивирование данных", 6, 5},
	}

	reports := make([]CategoryReport, 0, len(categories))
	for _, cat := range categories {
		score := 0.0
		if cat.checksTotal > 0 {
			score = math.Round(float64(cat.checksPassed)/float64(cat.checksTotal)*100*100) / 100
		}

		status := StatusCompliant
		switch {
		case score >= 90:
			status = StatusCompliant
		case score >= 60:
			status = StatusPartial
		default:
			status = StatusNonCompliant
		}

		reports = append(reports, CategoryReport{
			Name:         cat.name,
			Status:       status,
			Score:        score,
			ChecksPassed: cat.checksPassed,
			ChecksTotal:  cat.checksTotal,
			Description:  cat.description,
		})
	}

	return reports
}

// calculateSummary вычисляет сводные показатели из категорий.
func (g *ReportGenerator) calculateSummary(categories []CategoryReport) ReportSummary {
	summary := ReportSummary{}

	for _, cat := range categories {
		summary.TotalChecks += cat.ChecksTotal
		summary.PassedChecks += cat.ChecksPassed
		summary.FailedChecks += cat.ChecksTotal - cat.ChecksPassed
	}

	if summary.TotalChecks > 0 {
		summary.CompliancePercent = math.Round(
			float64(summary.PassedChecks)/float64(summary.TotalChecks)*100*100) / 100
	}

	summary.TotalDevices = 150      // mock — в production из БД
	summary.AtRiskDevices = 12      // mock
	summary.TotalExposure = 4850.00 // mock

	return summary
}

// identifyGaps определяет gaps для региона.
func (g *ReportGenerator) identifyGaps(region string) []GapItem {
	// Симулированные gaps — в production из БД/engine
	baseGaps := []GapItem{
		{
			ID:          "GAP-CRYPTO-001",
			Category:    "cryptography",
			Title:       "TLS 1.2 detected on Zone 3 endpoints",
			Description: "Обнаружены соединения с TLS 1.2 вместо TLS 1.3 на внутренних endpoints",
			Severity:    SeverityHigh,
			Status:      "open",
			DetectedAt:  time.Now().AddDate(0, 0, -5),
		},
		{
			ID:          "GAP-AUTH-001",
			Category:    "authentication",
			Title:       "MFA not enforced for non-admin users",
			Description: "Многофакторная аутентификация не обязательна для пользователей без роли admin",
			Severity:    SeverityMedium,
			Status:      "open",
			DetectedAt:  time.Now().AddDate(0, 0, -10),
		},
		{
			ID:          "GAP-AUDIT-001",
			Category:    "audit_logging",
			Title:       "Audit log retention below regional requirement",
			Description: "Текущий retention 180 дней, требуется 365 дней для региона",
			Severity:    SeverityMedium,
			Status:      "open",
			DetectedAt:  time.Now().AddDate(0, -1, 0),
		},
		{
			ID:          "GAP-DATA-001",
			Category:    "data_protection",
			Title:       "Cross-border data transfer without SCC",
			Description: "Обнаружена передача данных в регион без Standard Contractual Clauses",
			Severity:    SeverityHigh,
			Status:      "in_progress",
			DetectedAt:  time.Now().AddDate(0, 0, -20),
		},
		{
			ID:          "GAP-NET-001",
			Category:    "network_security",
			Title:       "Zone 2 to Zone 3 conduit without mTLS",
			Description: "Обнаружен conduit между DMZ и Application без взаимной TLS аутентификации",
			Severity:    SeverityCritical,
			Status:      "open",
			DetectedAt:  time.Now().AddDate(0, 0, -3),
		},
	}

	// Регион-специфичные adjustments
	switch region {
	case RegionBY:
		// Добавляем BY-specific gaps
		baseGaps = append(baseGaps, GapItem{
			ID:          "GAP-CRYPTO-002",
			Category:    "cryptography",
			Title:       "Non-СТБ cryptographic algorithms in use",
			Description: "Обнаружено использование AES-256 вместо belt-gcm (СТБ 34.101.30)",
			Severity:    SeverityCritical,
			Status:      "open",
			DetectedAt:  time.Now().AddDate(0, 0, -7),
		})
	case RegionEU:
		baseGaps = append(baseGaps, GapItem{
			ID:          "GAP-DATA-002",
			Category:    "data_protection",
			Title:       "GDPR Art. 17 right to erasure not implemented",
			Description: "Отсутствует механизм полного удаления данных по запросу субъекта",
			Severity:    SeverityHigh,
			Status:      "open",
			DetectedAt:  time.Now().AddDate(0, -2, 0),
		})
	}

	return baseGaps
}

// buildRemediations создаёт рекомендации по устранению gaps.
func (g *ReportGenerator) buildRemediations(gaps []GapItem) []Remediation {
	remediations := make([]Remediation, 0, len(gaps))
	for _, gap := range gaps {
		remediations = append(remediations, Remediation{
			GapID:    gap.ID,
			Action:   g.generateRemediationAction(gap),
			Priority: string(gap.Severity),
		})
	}
	return remediations
}

// generateRemediationAction генерирует текст remediation.
func (g *ReportGenerator) generateRemediationAction(gap GapItem) string {
	actions := map[string]string{
		"GAP-CRYPTO-001": "Обновить конфигурацию TLS до версии 1.3 на всех Zone 3 endpoints",
		"GAP-CRYPTO-002": "Мигрировать шифрование с AES-256-GCM на belt-gcm (СТБ 34.101.31)",
		"GAP-AUTH-001":   "Включить обязательную MFA для всех пользователей (TOTP или FIDO2)",
		"GAP-AUDIT-001":  "Увеличить retention audit логов до 365 дней, настроить автоматическую ротацию",
		"GAP-DATA-001":   "Заключить Standard Contractual Clauses (SCC) для трансграничной передачи данных",
		"GAP-DATA-002":   "Реализовать механизм GDPR Art. 17 (right to erasure) с подтверждением удаления",
		"GAP-NET-001":    "Настроить mTLS 1.3 для всех conduit между Zone 2 и Zone 3",
	}

	if action, ok := actions[gap.ID]; ok {
		return action
	}
	return fmt.Sprintf("Провести аудит и устранить несоответствие: %s", gap.Title)
}

// buildRegionCompliance строит данные compliance для региона.
func (g *ReportGenerator) buildRegionCompliance(region string) RegionCompliance {
	profile, _ := g.registry.Get(region)
	gaps := g.identifyGaps(region)

	score := 85.0 // mock score
	if region == RegionBY {
		score = 72.0 // BY typically stricter
	}

	status := StatusCompliant
	switch {
	case score >= 90:
		status = StatusCompliant
	case score >= 60:
		status = StatusPartial
	default:
		status = StatusNonCompliant
	}

	rc := RegionCompliance{
		Region:    region,
		Status:    status,
		Score:     score,
		Devices:   150,     // mock
		Exposure:  4850.00, // mock
		Gaps:      len(gaps),
		UpdatedAt: time.Now().UTC(),
	}

	if profile != nil {
		_ = profile.Name() // available for future enrichment
	}

	return rc
}

// ═══════════════════════════════════════════════════════════════════════════
// PDF Rendering
// ═══════════════════════════════════════════════════════════════════════════

// renderPDF генерирует PDF отчёт.
func (g *ReportGenerator) renderPDF(report *ComplianceReport) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	// Header
	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(190, 15, "Compliance Report", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Report metadata
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(190, 6, fmt.Sprintf("Report ID: %s", report.ID), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Region: %s", report.Region), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Generated: %s", report.GeneratedAt.Format(time.RFC1123)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Period: %s - %s",
		report.Period.Start.Format("2006-01-02"),
		report.Period.End.Format("2006-01-02")), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Status: %s", report.Status), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Summary section
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, "Executive Summary", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(190, 6, fmt.Sprintf("Compliance Score: %.1f%%", report.Summary.CompliancePercent), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Passed: %d/%d checks", report.Summary.PassedChecks, report.Summary.TotalChecks), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Failed: %d checks", report.Summary.FailedChecks), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Total Exposure: $%.2f", report.Summary.TotalExposure), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Categories table
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, "Compliance Categories", "", 1, "L", false, 0, "")

	// Table header
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(50, 7, "Category", "1", 0, "L", true, 0, "")
	pdf.CellFormat(20, 7, "Status", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Score", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Passed", "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 7, "Description", "1", 1, "L", true, 0, "")

	// Table rows
	pdf.SetFont("Helvetica", "", 8)
	for _, cat := range report.Categories {
		pdf.CellFormat(50, 6, cat.Name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(20, 6, string(cat.Status), "1", 0, "C", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%.1f%%", cat.Score), "1", 0, "C", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%d/%d", cat.ChecksPassed, cat.ChecksTotal), "1", 0, "C", false, 0, "")
		pdf.CellFormat(70, 6, truncate(cat.Description, 50), "1", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// Gap Analysis
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, fmt.Sprintf("Gap Analysis (%d gaps found)", report.GapAnalysis.TotalGaps), "", 1, "L", false, 0, "")

	if len(report.GapAnalysis.Gaps) > 0 {
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(8, 7, "#", "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, "Category", "1", 0, "L", true, 0, "")
		pdf.CellFormat(55, 7, "Title", "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, 7, "Severity", "1", 0, "C", true, 0, "")
		pdf.CellFormat(72, 7, "Action Required", "1", 1, "L", true, 0, "")

		pdf.SetFont("Helvetica", "", 8)
		for i, gap := range report.GapAnalysis.Gaps {
			action := g.generateRemediationAction(gap)
			pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 6, gap.Category, "1", 0, "L", false, 0, "")
			pdf.CellFormat(55, 6, truncate(gap.Title, 35), "1", 0, "L", false, 0, "")
			pdf.CellFormat(20, 6, string(gap.Severity), "1", 0, "C", false, 0, "")
			pdf.CellFormat(72, 6, truncate(action, 45), "1", 1, "L", false, 0, "")
		}
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(190, 5, fmt.Sprintf("Signed by: %s", report.SignedBy), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, "This report is auto-generated. For verification, contact compliance@cctv-monitor.io", "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("render PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// XML Rendering
// ═══════════════════════════════════════════════════════════════════════════

// renderXML генерирует XML отчёт.
func (g *ReportGenerator) renderXML(report *ComplianceReport) ([]byte, error) {
	output, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render XML: %w", err)
	}

	// Добавляем XML header
	header := []byte(xml.Header)
	return append(header, output...), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// determineOverallStatus определяет общий статус на основе процента compliance.
func determineOverallStatus(score float64) ComplianceStatus {
	switch {
	case score >= 90:
		return StatusCompliant
	case score >= 60:
		return StatusPartial
	case score > 0:
		return StatusNonCompliant
	default:
		return StatusNotAssessed
	}
}

// countBySeverity группирует gaps по severity.
func countBySeverity(gaps []GapItem) map[SeverityLevel]int {
	result := map[SeverityLevel]int{
		SeverityLow:      0,
		SeverityMedium:   0,
		SeverityHigh:     0,
		SeverityCritical: 0,
	}
	for _, g := range gaps {
		result[g.Severity]++
	}
	return result
}

// truncate обрезает строку до указанной длины.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// Обрезаем по границе слов
	trimmed := s[:maxLen]
	if idx := strings.LastIndex(trimmed, " "); idx > 0 {
		trimmed = trimmed[:idx]
	}
	return trimmed + "..."
}
