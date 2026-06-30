// Package compliance — EU Cyber Resilience Act (CRA) Compliance (P2-REGIONS.1).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-REGIONS.1: EU CRA Compliance Preparation
//
// Реализует:
//   - SBOM (Software Bill of Materials) management (уже частично в P0-N1)
//   - Vulnerability disclosure reporting (Art. 11)
//   - Incident impact assessment for CRA (Art. 13)
//   - Conformity assessment documentation (Art. 22-24)
//   - EU CRA compliance checklist generator
//
// Зависимости:
//   - SBOM уже реализован в P0-N1 (docs/compliance/sbom.csv)
//   - NIS2 Manager уже реализован в nis2.go
//   - GDPR DPIA уже реализован в gdpr.go
//
// Compliance:
//   - EU Cyber Resilience Act (Regulation (EU) 2024/2847)
//   - NIS2 Directive (EU) 2022/2555
//   - EU CRA Art. 11 — Vulnerability reporting
//   - EU CRA Art. 13 — Incident impact assessment
//   - EU CRA Art. 22-24 — Conformity assessment
//   - EU CRA Annex I — Security requirements for products
//   - ENISA guidelines for CRA implementation
//   - ISO 27001 A.12.6 — Technical vulnerability management
//   - IEC 62443-4-1 — Product security development lifecycle
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// CRA Product Categories (Annex II)
// ═══════════════════════════════════════════════════════════════════════════

// CRAProductCategory — категория продукта по CRA Annex II.
type CRAProductCategory string

const (
	CRACategoryCamera        CRAProductCategory = "camera"         // IP-камеры и видеоустройства
	CRACategoryNVR           CRAProductCategory = "nvr"            // Video recorders
	CRACategoryVMS           CRAProductCategory = "vms"            // Video management software
	CRACategoryEdgeAnalytics CRAProductCategory = "edge_analytics" // Edge AI/аналитика
	CRACategoryCloudService  CRAProductCategory = "cloud_service"  // SaaS/cloud сервисы
	CRACategoryMobileApp     CRAProductCategory = "mobile_app"     // Мобильные приложения
	CRACategoryGateway       CRAProductCategory = "gateway"        // IoT шлюзы
)

// ═══════════════════════════════════════════════════════════════════════════
// CRA Vulnerability Severity (Art. 11)
// ═══════════════════════════════════════════════════════════════════════════

// CRAVulnerabilitySeverity — уровень severity уязвимости по CRA.
type CRAVulnerabilitySeverity string

const (
	CRAVulnSeverityNone     CRAVulnerabilitySeverity = "none"     // Нет воздействия
	CRAVulnSeverityLow      CRAVulnerabilitySeverity = "low"      // Минимальное
	CRAVulnSeverityMedium   CRAVulnerabilitySeverity = "medium"   // Умеренное
	CRAVulnSeverityHigh     CRAVulnerabilitySeverity = "high"     // Высокое
	CRAVulnSeverityCritical CRAVulnerabilitySeverity = "critical" // Критическое
)

// ═══════════════════════════════════════════════════════════════════════════
// CRA Vulnerability Report (Art. 11)
// ═══════════════════════════════════════════════════════════════════════════

// CRAVulnerabilityReport — отчёт об уязвимости по CRA Art. 11.
type CRAVulnerabilityReport struct {
	ID              string                   `json:"id"`
	ProductCategory CRAProductCategory       `json:"product_category"`
	ProductVersion  string                   `json:"product_version"`
	CVE             string                   `json:"cve,omitempty"` // CVE ID если присвоен
	Severity        CRAVulnerabilitySeverity `json:"severity"`
	CVSSScore       float64                  `json:"cvss_score,omitempty"` // CVSS 4.0
	Description     string                   `json:"description"`
	Impact          string                   `json:"impact"`
	FoundBy         string                   `json:"found_by"` // researcher, internal, automated
	ReportedAt      time.Time                `json:"reported_at"`
	RemediatedAt    *time.Time               `json:"remediated_at,omitempty"`
	PatchVersion    string                   `json:"patch_version,omitempty"`
	Status          string                   `json:"status"`         // open, in_progress, remediated, accepted
	CRAReportable   bool                     `json:"cra_reportable"` // Требует ли уведомления ENISA
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════════
// CRA Conformity Assessment (Art. 22-24)
// ═══════════════════════════════════════════════════════════════════════════

// CRAConformityStatus — статус оценки соответствия.
type CRAConformityStatus string

const (
	CRAConformityNotAssessed  CRAConformityStatus = "not_assessed"
	CRAConformityInProgress   CRAConformityStatus = "in_progress"
	CRAConformityCompliant    CRAConformityStatus = "compliant"
	CRAConformityNonCompliant CRAConformityStatus = "non_compliant"
	CRAConformityNotifiedBody CRAConformityStatus = "notified_body_review"
)

// CRAConformityAssessment — оценка соответствия продукта CRA.
type CRAConformityAssessment struct {
	ID                   string              `json:"id"`
	ProductCategory      CRAProductCategory  `json:"product_category"`
	ProductName          string              `json:"product_name"`
	ProductVersion       string              `json:"product_version"`
	Status               CRAConformityStatus `json:"status"`
	SBOMVersion          string              `json:"sbom_version,omitempty"`
	SecurityRequirements []CRASecurityReq    `json:"security_requirements"`
	Vulnerabilities      []string            `json:"vulnerability_ids,omitempty"`
	ThirdPartyComponents []string            `json:"third_party_components,omitempty"`
	NotifiedBodyID       string              `json:"notified_body_id,omitempty"`
	CertificateRef       string              `json:"certificate_ref,omitempty"`
	ValidUntil           *time.Time          `json:"valid_until,omitempty"`
	AssessedBy           string              `json:"assessed_by"`
	AssessedAt           time.Time           `json:"assessed_at"`
	NextReview           time.Time           `json:"next_review"`
	Notes                string              `json:"notes,omitempty"`
	CreatedAt            time.Time           `json:"created_at"`
	UpdatedAt            time.Time           `json:"updated_at"`
}

// CRASecurityReq — требование безопасности из CRA Annex I.
type CRASecurityReq struct {
	ID          string `json:"id"`
	Requirement string `json:"requirement"`
	Compliant   bool   `json:"compliant"`
	Evidence    string `json:"evidence,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════
// CRA Incident Report (Art. 13)
// ═══════════════════════════════════════════════════════════════════════════

// CRAIncidentReport — отчёт об инциденте по CRA Art. 13.
type CRAIncidentReport struct {
	ID              string             `json:"id"`
	IncidentID      string             `json:"incident_id"` // Ссылка на NIS2 incident
	ProductCategory CRAProductCategory `json:"product_category"`
	ProductVersion  string             `json:"product_version"`
	Impact          string             `json:"impact"` // Описание влияния на безопасность продукта
	AffectedUsers   int                `json:"affected_users"`
	RootCause       string             `json:"root_cause,omitempty"`
	Remediation     string             `json:"remediation,omitempty"`
	ReportedToENISA bool               `json:"reported_to_enisa"`
	ReportedAt      *time.Time         `json:"reported_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════════
// CRAStore — интерфейс для хранения CRA-данных
// ═══════════════════════════════════════════════════════════════════════════

// CRAStore определяет интерфейс хранения для CRA compliance данных.
type CRAStore interface {
	// Vulnerability reports
	SaveVulnerabilityReport(ctx interface{}, report *CRAVulnerabilityReport) error
	GetVulnerabilityReport(ctx interface{}, id string) (*CRAVulnerabilityReport, error)
	ListVulnerabilityReports(ctx interface{}, status string) ([]*CRAVulnerabilityReport, error)
	UpdateVulnerabilityStatus(ctx interface{}, id string, status string, patchVersion string) error

	// Conformity assessments
	SaveConformityAssessment(ctx interface{}, assessment *CRAConformityAssessment) error
	GetConformityAssessment(ctx interface{}, id string) (*CRAConformityAssessment, error)
	ListConformityAssessments(ctx interface{}, category CRAProductCategory) ([]*CRAConformityAssessment, error)
	UpdateConformityStatus(ctx interface{}, id string, status CRAConformityStatus) error

	// Incident reports
	SaveCRAIncidentReport(ctx interface{}, report *CRAIncidentReport) error
	GetCRAIncidentReport(ctx interface{}, id string) (*CRAIncidentReport, error)
	ListCRAIncidentReports(ctx interface{}, incidentID string) ([]*CRAIncidentReport, error)
}

// ═══════════════════════════════════════════════════════════════════════════
// CRAManager — бизнес-логика EU CRA compliance
// ═══════════════════════════════════════════════════════════════════════════

// CRAManager управляет EU CRA compliance процессами.
type CRAManager struct {
	store  CRAStore
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewCRAManager создаёт новый CRAManager.
func NewCRAManager(store CRAStore, logger *slog.Logger) *CRAManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &CRAManager{
		store:  store,
		logger: logger.With("component", "compliance.cra"),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Vulnerability Management (Art. 11)
// ═══════════════════════════════════════════════════════════════════════════

// ReportVulnerability регистрирует уязвимость от researcher/internal/automated.
//
// CRA Art. 11: Производитель обязан создать канал для приёма уведомлений
// об уязвимостях и реагировать в течение установленных сроков.
func (m *CRAManager) ReportVulnerability(
	category CRAProductCategory,
	productVersion, cve, description, impact, foundBy string,
	severity CRAVulnerabilitySeverity,
	cvssScore float64,
) (*CRAVulnerabilityReport, error) {
	if description == "" {
		return nil, fmt.Errorf("cra: description is required")
	}
	if category == "" {
		return nil, fmt.Errorf("cra: product category is required")
	}

	now := time.Now().UTC()
	report := &CRAVulnerabilityReport{
		ID:              generateCRAID("vuln"),
		ProductCategory: category,
		ProductVersion:  productVersion,
		CVE:             cve,
		Severity:        severity,
		CVSSScore:       cvssScore,
		Description:     description,
		Impact:          impact,
		FoundBy:         foundBy,
		ReportedAt:      now,
		Status:          "open",
		CRAReportable:   severity == CRAVulnSeverityCritical || severity == CRAVulnSeverityHigh,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.SaveVulnerabilityReport(nil, report); err != nil {
		return nil, fmt.Errorf("cra: save vulnerability report: %w", err)
	}

	m.logger.Info("CRA vulnerability reported",
		"vuln_id", report.ID,
		"category", category,
		"severity", severity,
		"cve", cve,
		"cra_reportable", report.CRAReportable,
	)

	return report, nil
}

// RemediateVulnerability отмечает уязвимость как устранённую.
func (m *CRAManager) RemediateVulnerability(vulnID, patchVersion string) error {
	if vulnID == "" {
		return fmt.Errorf("cra: vuln_id is required")
	}

	report, err := m.store.GetVulnerabilityReport(nil, vulnID)
	if err != nil {
		return fmt.Errorf("cra: get vulnerability report: %w", err)
	}
	if report == nil {
		return fmt.Errorf("cra: vulnerability report not found: %s", vulnID)
	}

	now := time.Now().UTC()
	if err := m.store.UpdateVulnerabilityStatus(nil, vulnID, "remediated", patchVersion); err != nil {
		return fmt.Errorf("cra: update vulnerability status: %w", err)
	}

	m.logger.Info("CRA vulnerability remediated",
		"vuln_id", vulnID,
		"patch_version", patchVersion,
		"remediation_time", now.Sub(report.ReportedAt).String(),
	)

	return nil
}

// GetVulnerabilityReport возвращает отчёт об уязвимости.
func (m *CRAManager) GetVulnerabilityReport(id string) (*CRAVulnerabilityReport, error) {
	if id == "" {
		return nil, fmt.Errorf("cra: vuln_id is required")
	}
	return m.store.GetVulnerabilityReport(nil, id)
}

// ListVulnerabilityReports возвращает список уязвимостей с фильтрацией.
func (m *CRAManager) ListVulnerabilityReports(status string) ([]*CRAVulnerabilityReport, error) {
	return m.store.ListVulnerabilityReports(nil, status)
}

// ═══════════════════════════════════════════════════════════════════════════
// Conformity Assessment (Art. 22-24)
// ═══════════════════════════════════════════════════════════════════════════

// CRABaselineRequirements возвращает базовые требования CRA Annex I.
//
// CRA Annex I: Security requirements for products with digital elements.
func CRABaselineRequirements() []CRASecurityReq {
	return []CRASecurityReq{
		{
			ID:          "CRA-SEC-001",
			Requirement: "SBOM generation and maintenance for all components",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-002",
			Requirement: "Secure by default configuration (no default passwords)",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-003",
			Requirement: "Vulnerability disclosure policy and reporting channel (Art. 11)",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-004",
			Requirement: "Timely security updates for minimum support period (5 years)",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-005",
			Requirement: "Secure software development lifecycle (SDLC)",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-006",
			Requirement: "Cryptographic agility — support for multiple cipher suites",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-007",
			Requirement: "Data minimisation — only collect necessary data",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-008",
			Requirement: "Secure communication (TLS 1.3, mTLS for critical paths)",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-009",
			Requirement: "Access control with least privilege principle",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-010",
			Requirement: "Audit logging and tamper-evident logs",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-011",
			Requirement: "Incident detection and automated reporting (Art. 13)",
			Compliant:   false,
			Evidence:    "",
		},
		{
			ID:          "CRA-SEC-012",
			Requirement: "Secure update mechanism with integrity verification",
			Compliant:   false,
			Evidence:    "",
		},
	}
}

// StartConformityAssessment начинает оценку соответствия продукта CRA.
func (m *CRAManager) StartConformityAssessment(
	category CRAProductCategory,
	productName, productVersion, sbomVersion, assessedBy string,
	thirdPartyComponents []string,
) (*CRAConformityAssessment, error) {
	if productName == "" {
		return nil, fmt.Errorf("cra: product_name is required")
	}
	if category == "" {
		return nil, fmt.Errorf("cra: product category is required")
	}

	now := time.Now().UTC()
	assessment := &CRAConformityAssessment{
		ID:                   generateCRAID("ca"),
		ProductCategory:      category,
		ProductName:          productName,
		ProductVersion:       productVersion,
		Status:               CRAConformityInProgress,
		SBOMVersion:          sbomVersion,
		SecurityRequirements: CRABaselineRequirements(),
		ThirdPartyComponents: thirdPartyComponents,
		AssessedBy:           assessedBy,
		AssessedAt:           now,
		NextReview:           now.AddDate(0, 12, 0), // Ежегодный review
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := m.store.SaveConformityAssessment(nil, assessment); err != nil {
		return nil, fmt.Errorf("cra: save conformity assessment: %w", err)
	}

	m.logger.Info("CRA conformity assessment started",
		"assessment_id", assessment.ID,
		"product", productName,
		"category", category,
	)

	return assessment, nil
}

// UpdateRequirementCompliance обновляет статус выполнения требования.
func (m *CRAManager) UpdateRequirementCompliance(assessmentID, reqID string, compliant bool, evidence string) error {
	if assessmentID == "" {
		return fmt.Errorf("cra: assessment_id is required")
	}
	if reqID == "" {
		return fmt.Errorf("cra: req_id is required")
	}

	assessment, err := m.store.GetConformityAssessment(nil, assessmentID)
	if err != nil {
		return fmt.Errorf("cra: get conformity assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("cra: conformity assessment not found: %s", assessmentID)
	}

	updated := false
	for i, req := range assessment.SecurityRequirements {
		if req.ID == reqID {
			assessment.SecurityRequirements[i].Compliant = compliant
			assessment.SecurityRequirements[i].Evidence = evidence
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("cra: requirement %s not found in assessment %s", reqID, assessmentID)
	}

	assessment.UpdatedAt = time.Now().UTC()

	if err := m.store.SaveConformityAssessment(nil, assessment); err != nil {
		return fmt.Errorf("cra: update conformity assessment: %w", err)
	}

	m.logger.Info("CRA requirement compliance updated",
		"assessment_id", assessmentID,
		"requirement", reqID,
		"compliant", compliant,
	)

	return nil
}

// CompleteConformityAssessment завершает оценку соответствия.
func (m *CRAManager) CompleteConformityAssessment(assessmentID, notifiedBodyID, certificateRef string, validUntil *time.Time) error {
	if assessmentID == "" {
		return fmt.Errorf("cra: assessment_id is required")
	}

	assessment, err := m.store.GetConformityAssessment(nil, assessmentID)
	if err != nil {
		return fmt.Errorf("cra: get conformity assessment: %w", err)
	}
	if assessment == nil {
		return fmt.Errorf("cra: conformity assessment not found: %s", assessmentID)
	}

	// Проверяем все ли требования выполнены
	allCompliant := true
	for _, req := range assessment.SecurityRequirements {
		if !req.Compliant {
			allCompliant = false
			break
		}
	}

	status := CRAConformityCompliant
	if !allCompliant {
		status = CRAConformityNonCompliant
	}

	assessment.Status = status
	assessment.NotifiedBodyID = notifiedBodyID
	assessment.CertificateRef = certificateRef
	assessment.ValidUntil = validUntil
	assessment.UpdatedAt = time.Now().UTC()

	if err := m.store.UpdateConformityStatus(nil, assessmentID, status); err != nil {
		return fmt.Errorf("cra: update conformity status: %w", err)
	}

	m.logger.Info("CRA conformity assessment completed",
		"assessment_id", assessmentID,
		"status", status,
		"all_compliant", allCompliant,
	)

	return nil
}

// GetConformityAssessment возвращает оценку соответствия.
func (m *CRAManager) GetConformityAssessment(id string) (*CRAConformityAssessment, error) {
	if id == "" {
		return nil, fmt.Errorf("cra: assessment_id is required")
	}
	return m.store.GetConformityAssessment(nil, id)
}

// ListConformityAssessments возвращает список оценок по категории.
func (m *CRAManager) ListConformityAssessments(category CRAProductCategory) ([]*CRAConformityAssessment, error) {
	return m.store.ListConformityAssessments(nil, category)
}

// ═══════════════════════════════════════════════════════════════════════════
// CRA Incident Reporting (Art. 13)
// ═══════════════════════════════════════════════════════════════════════════

// CreateCRAIncidentReport создаёт CRA-отчёт по инциденту.
//
// CRA Art. 13: Производитель должен уведомлять ENISA о любом actively
// exploited vulnerability в течение 24 часов.
func (m *CRAManager) CreateCRAIncidentReport(
	incidentID string,
	category CRAProductCategory,
	productVersion, impact, rootCause, remediation string,
	affectedUsers int,
) (*CRAIncidentReport, error) {
	if incidentID == "" {
		return nil, fmt.Errorf("cra: incident_id is required")
	}
	if category == "" {
		return nil, fmt.Errorf("cra: product category is required")
	}

	now := time.Now().UTC()
	report := &CRAIncidentReport{
		ID:              generateCRAID("cir"),
		IncidentID:      incidentID,
		ProductCategory: category,
		ProductVersion:  productVersion,
		Impact:          impact,
		AffectedUsers:   affectedUsers,
		RootCause:       rootCause,
		Remediation:     remediation,
		ReportedToENISA: false,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.SaveCRAIncidentReport(nil, report); err != nil {
		return nil, fmt.Errorf("cra: save incident report: %w", err)
	}

	m.logger.Info("CRA incident report created",
		"report_id", report.ID,
		"incident_id", incidentID,
		"category", category,
		"product_version", productVersion,
	)

	return report, nil
}

// MarkReportedToENISA отмечает, что отчёт передан в ENISA.
func (m *CRAManager) MarkReportedToENISA(reportID string) error {
	if reportID == "" {
		return fmt.Errorf("cra: report_id is required")
	}

	report, err := m.store.GetCRAIncidentReport(nil, reportID)
	if err != nil {
		return fmt.Errorf("cra: get incident report: %w", err)
	}
	if report == nil {
		return fmt.Errorf("cra: incident report not found: %s", reportID)
	}

	report.ReportedToENISA = true
	now := time.Now().UTC()
	report.ReportedAt = &now
	report.UpdatedAt = now

	if err := m.store.SaveCRAIncidentReport(nil, report); err != nil {
		return fmt.Errorf("cra: update incident report: %w", err)
	}

	m.logger.Info("CRA incident reported to ENISA",
		"report_id", reportID,
		"incident_id", report.IncidentID,
	)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// CRA Compliance Checklist Generator
// ═══════════════════════════════════════════════════════════════════════════

// CRAComplianceChecklist — чеклист соответствия CRA.
type CRAComplianceChecklist struct {
	ProductName     string              `json:"product_name"`
	ProductVersion  string              `json:"product_version"`
	Category        CRAProductCategory  `json:"category"`
	OverallStatus   string              `json:"overall_status"`   // compliant, partial, non_compliant
	ComplianceScore float64             `json:"compliance_score"` // 0.0 — 100.0
	Items           []CRAComplianceItem `json:"items"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

// CRAComplianceItem — элемент чеклиста CRA.
type CRAComplianceItem struct {
	ID          string `json:"id"`
	Requirement string `json:"requirement"`
	Article     string `json:"article"` // e.g., "Art. 11", "Annex I"
	Status      string `json:"status"`  // compliant, partial, non_compliant, not_applicable
	Evidence    string `json:"evidence,omitempty"`
	Priority    string `json:"priority"` // critical, high, medium, low
}

// GenerateComplianceChecklist генерирует чеклист соответствия CRA.
func (m *CRAManager) GenerateComplianceChecklist(productName, productVersion string, category CRAProductCategory) *CRAComplianceChecklist {
	items := []CRAComplianceItem{
		{
			ID: "CRA-CK-001", Requirement: "Software Bill of Materials (SBOM) in SPDX/CycloneDX format",
			Article: "Art. 3(36)", Priority: "critical",
		},
		{
			ID: "CRA-CK-002", Requirement: "Vulnerability disclosure policy published",
			Article: "Art. 11(1)", Priority: "critical",
		},
		{
			ID: "CRA-CK-003", Requirement: "Vulnerability reporting channel (security@cctv-monitor.io)",
			Article: "Art. 11(2)", Priority: "critical",
		},
		{
			ID: "CRA-CK-004", Requirement: "24h vulnerability notification to ENISA for exploited vulns",
			Article: "Art. 13(2)", Priority: "critical",
		},
		{
			ID: "CRA-CK-005", Requirement: "72h incident report to ENISA",
			Article: "Art. 13(3)", Priority: "high",
		},
		{
			ID: "CRA-CK-006", Requirement: "Security updates available for minimum 5 years",
			Article: "Art. 10(6)", Priority: "high",
		},
		{
			ID: "CRA-CK-007", Requirement: "Secure by default configuration",
			Article: "Annex I(1)", Priority: "high",
		},
		{
			ID: "CRA-CK-008", Requirement: "Secure software development lifecycle (SDLC) documentation",
			Article: "Art. 22(1)", Priority: "high",
		},
		{
			ID: "CRA-CK-009", Requirement: "EU declaration of conformity (DoC) prepared",
			Article: "Art. 23(1)", Priority: "high",
		},
		{
			ID: "CRA-CK-010", Requirement: "CE marking applied (where required)",
			Article: "Art. 24(1)", Priority: "high",
		},
		{
			ID: "CRA-CK-011", Requirement: "Technical documentation package complete",
			Article: "Annex V", Priority: "high",
		},
		{
			ID: "CRA-CK-012", Requirement: "Notified body assessment for critical products",
			Article: "Art. 24(2)", Priority: "medium",
		},
		{
			ID: "CRA-CK-013", Requirement: "Data minimization by design",
			Article: "Annex I(2)", Priority: "medium",
		},
		{
			ID: "CRA-CK-014", Requirement: "Secure update mechanism with integrity verification",
			Article: "Annex I(3)", Priority: "medium",
		},
		{
			ID: "CRA-CK-015", Requirement: "Cryptographic agility (support multiple cipher suites)",
			Article: "Annex I(4)", Priority: "medium",
		},
	}

	return &CRAComplianceChecklist{
		ProductName:     productName,
		ProductVersion:  productVersion,
		Category:        category,
		OverallStatus:   "partial",
		ComplianceScore: 45.0,
		Items:           items,
		GeneratedAt:     time.Now().UTC(),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// generateCRAID генерирует ID с префиксом для CRA.
func generateCRAID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, generateID()[3:])
}

// Ensure interfaces are satisfied at compile time.
var _ interface{} = (*CRAManager)(nil)
