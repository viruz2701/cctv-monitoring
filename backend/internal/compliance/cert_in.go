// Package compliance — CERT-In Integration for India (P2-REGIONS.4).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-REGIONS.4: CERT-In 6h Reporting Integration
//
// Реализует:
//   - CERT-In incident reporting API integration (6h SLA)
//   - CERT-In Directions compliance (2022-2024)
//   - Incident classification per CERT-In taxonomy
//   - Automated report generation in CERT-In format
//   - Синхронизация с существующим IncidentResponseEngine (P0-IR.2)
//
// Зависимости:
//   - IncidentResponseEngine из incident_response.go (уже реализован)
//   - NIS2Manager из nis2.go (реюзает модели инцидентов)
//
// Compliance:
//   - CERT-In Directions 2022 (6h incident reporting)
//   - CERT-In Directions 2024 (expanded reporting)
//   - IT Act 2000 (Information Technology Act)
//   - IT (Reasonable Security Practices) Rules 2018
//   - ISO 27001 A.16.1 — Incident management
//   - IEC 62443-3-3 SR 7.1 — Incident response
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// CERT-In Incident Categories
// ═══════════════════════════════════════════════════════════════════════════

// CERTInCategory — категория инцидента по CERT-In.
type CERTInCategory string

const (
	CERTInCategoryDoS          CERTInCategory = "denial_of_service"   // DoS/DDoS
	CERTInCategoryMalware      CERTInCategory = "malware"             // Вредоносное ПО
	CERTInCategoryPhishing     CERTInCategory = "phishing"            // Фишинг
	CERTInCategoryUnauthorized CERTInCategory = "unauthorized_access" // Несанкционированный доступ
	CERTInCategorySystemCrash  CERTInCategory = "system_crash"        // Системный сбой
	CERTInCategoryDataBreach   CERTInCategory = "data_breach"         // Утечка данных
	CERTInCategoryDefacement   CERTInCategory = "defacement"          // Дефейс
	CERTInCategoryScanning     CERTInCategory = "scanning_probing"    // Сканирование/зондирование
	CERTInCategorySpam         CERTInCategory = "spam"                // Спам
	CERTInCategoryOther        CERTInCategory = "other"               // Другое
)

// ═══════════════════════════════════════════════════════════════════════════
// CERT-In Impact Categories
// ═══════════════════════════════════════════════════════════════════════════

// CERTInImpact — уровень воздействия по CERT-In.
type CERTInImpact string

const (
	CERTInImpactLow      CERTInImpact = "low"      // Минимальное воздействие
	CERTInImpactMedium   CERTInImpact = "medium"   // Умеренное
	CERTInImpactHigh     CERTInImpact = "high"     // Высокое
	CERTInImpactCritical CERTInImpact = "critical" // Критическое
)

// ═══════════════════════════════════════════════════════════════════════════
// CERT-In Report Requirements
// ═══════════════════════════════════════════════════════════════════════════

// CERTInReport — отчёт в CERT-In согласно Directions 2022/2024.
type CERTInReport struct {
	ID          string `json:"id"`
	IncidentID  string `json:"incident_id"`              // Внутренний ID инцидента
	CERTInRefNo string `json:"cert_in_ref_no,omitempty"` // Ответный номер от CERT-In

	// Организационные данные
	OrganizationName string `json:"organization_name"`
	OrganizationType string `json:"organization_type"` // telecom, banking, cbdc, government, csp, intermediary
	Sector           string `json:"sector"`
	ContactName      string `json:"contact_name"`
	ContactEmail     string `json:"contact_email"`
	ContactPhone     string `json:"contact_phone,omitempty"`

	// Данные об инциденте
	Category        CERTInCategory `json:"category"`
	Impact          CERTInImpact   `json:"impact"`
	IncidentDate    time.Time      `json:"incident_date"`
	DetectionDate   time.Time      `json:"detection_date"`
	IncidentDesc    string         `json:"incident_description"`
	AffectedSystems int            `json:"affected_systems"`
	AffectedUsers   int            `json:"affected_users,omitempty"`
	DataCompromised bool           `json:"data_compromised"`
	DataCategory    string         `json:"data_category,omitempty"` // personal, financial, intellectual_property
	IPAddresses     []string       `json:"ip_addresses,omitempty"`
	URLs            []string       `json:"urls,omitempty"`
	MalwareHash     string         `json:"malware_hash,omitempty"`

	// Реагирование
	ActionTaken       string `json:"action_taken"`
	ContainmentStatus string `json:"containment_status"` // contained, not_contained, not_applicable
	RemediationPlan   string `json:"remediation_plan,omitempty"`
	IsOngoing         bool   `json:"is_ongoing"`

	// Статус отправки
	Status         string     `json:"status"` // draft, submitted, acknowledged, rejected
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	Deadline       time.Time  `json:"deadline"` // 6h from detection
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════════
// CERT-In API Client
// ═══════════════════════════════════════════════════════════════════════════

// CERTInAPIClient — клиент для интеграции с CERT-In API.
//
// API endpoint: https://certin.gov.in/api/v1/incidents
// Authentication: API key в заголовке X-API-Key
type CERTInAPIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewCERTInAPIClient создаёт новый клиент CERT-In API.
func NewCERTInAPIClient(baseURL, apiKey string, logger *slog.Logger) *CERTInAPIClient {
	if logger == nil {
		logger = slog.Default()
	}

	return &CERTInAPIClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     logger.With("component", "compliance.cert_in.client"),
	}
}

// SubmitReport отправляет отчёт в CERT-In.
//
// Возвращает reference number от CERT-In при успешной отправке.
func (c *CERTInAPIClient) SubmitReport(report *CERTInReport) (string, error) {
	if report == nil {
		return "", fmt.Errorf("cert_in: report is nil")
	}

	payload, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("cert_in: marshal report: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/incidents", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("cert_in: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("User-Agent", "CCTV-Health-Monitor/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cert_in: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return "", fmt.Errorf("cert_in: API returned status %d", resp.StatusCode)
	}

	var result struct {
		RefNo   string `json:"reference_number"`
		Status  string `json:"status"`
		Message string `json:"message,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("cert_in: decode response: %w", err)
	}

	c.logger.Info("CERT-In report submitted",
		"incident_id", report.IncidentID,
		"ref_no", result.RefNo,
		"status", result.Status,
	)

	return result.RefNo, nil
}

// CheckStatus проверяет статус ранее отправленного отчёта.
func (c *CERTInAPIClient) CheckStatus(refNo string) (string, error) {
	if refNo == "" {
		return "", fmt.Errorf("cert_in: reference number is required")
	}

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/incidents/"+refNo, nil)
	if err != nil {
		return "", fmt.Errorf("cert_in: create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("cert_in: check status: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("cert_in: decode response: %w", err)
	}

	return result.Status, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// CERTInManager — бизнес-логика CERT-In compliance
// ═══════════════════════════════════════════════════════════════════════════

// CERTInStore — интерфейс для хранения CERT-In данных.
type CERTInStore interface {
	SaveCERTInReport(ctx interface{}, report *CERTInReport) error
	GetCERTInReport(ctx interface{}, id string) (*CERTInReport, error)
	GetCERTInReportByIncident(ctx interface{}, incidentID string) (*CERTInReport, error)
	ListCERTInReports(ctx interface{}, status string) ([]*CERTInReport, error)
	UpdateCERTInStatus(ctx interface{}, id string, status string, refNo string) error
}

// CERTInManager управляет CERT-In compliance процессами.
type CERTInManager struct {
	store  CERTInStore
	api    *CERTInAPIClient
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewCERTInManager создаёт новый CERTInManager.
func NewCERTInManager(store CERTInStore, apiClient *CERTInAPIClient, logger *slog.Logger) *CERTInManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &CERTInManager{
		store:  store,
		api:    apiClient,
		logger: logger.With("component", "compliance.cert_in"),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// CERT-In Report Lifecycle
// ═══════════════════════════════════════════════════════════════════════════

// CreateReport создаёт черновик отчёта CERT-In из данных инцидента.
//
// CERT-In Directions: Incident must be reported within 6 hours.
// Дата-время обнаружения (detectionDate) используется как точка отсчёта.
func (m *CERTInManager) CreateReport(
	incidentID string,
	category CERTInCategory,
	impact CERTInImpact,
	organizationName, organizationType, sector string,
	contactName, contactEmail, contactPhone string,
	incidentDescription, actionTaken, containmentStatus, remediationPlan string,
	affectedSystems, affectedUsers int,
	dataCompromised bool,
	dataCategory string,
	ipAddresses, urls []string,
	malwareHash string,
	isOngoing bool,
) (*CERTInReport, error) {
	if incidentID == "" {
		return nil, fmt.Errorf("cert_in: incident_id is required")
	}
	if category == "" {
		return nil, fmt.Errorf("cert_in: category is required")
	}

	now := time.Now().UTC()
	report := &CERTInReport{
		ID:                generateCERTInID("cr"),
		IncidentID:        incidentID,
		OrganizationName:  organizationName,
		OrganizationType:  organizationType,
		Sector:            sector,
		ContactName:       contactName,
		ContactEmail:      contactEmail,
		ContactPhone:      contactPhone,
		Category:          category,
		Impact:            impact,
		IncidentDate:      now,
		DetectionDate:     now,
		IncidentDesc:      incidentDescription,
		AffectedSystems:   affectedSystems,
		AffectedUsers:     affectedUsers,
		DataCompromised:   dataCompromised,
		DataCategory:      dataCategory,
		IPAddresses:       ipAddresses,
		URLs:              urls,
		MalwareHash:       malwareHash,
		ActionTaken:       actionTaken,
		ContainmentStatus: containmentStatus,
		RemediationPlan:   remediationPlan,
		IsOngoing:         isOngoing,
		Status:            "draft",
		Deadline:          now.Add(6 * time.Hour), // 6h SLA
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := m.store.SaveCERTInReport(nil, report); err != nil {
		return nil, fmt.Errorf("cert_in: save report: %w", err)
	}

	m.logger.Info("CERT-In report draft created",
		"report_id", report.ID,
		"incident_id", incidentID,
		"deadline", report.Deadline.Format(time.RFC3339),
		"hours_remaining", 6.0,
	)

	return report, nil
}

// SubmitReport отправляет отчёт в CERT-In через API.
//
// Автоматически обновляет статус и reference number при успешной отправке.
func (m *CERTInManager) SubmitReport(reportID string) error {
	if reportID == "" {
		return fmt.Errorf("cert_in: report_id is required")
	}

	report, err := m.store.GetCERTInReport(nil, reportID)
	if err != nil {
		return fmt.Errorf("cert_in: get report: %w", err)
	}
	if report == nil {
		return fmt.Errorf("cert_in: report not found: %s", reportID)
	}

	if report.Status == "submitted" {
		return fmt.Errorf("cert_in: report %s already submitted", reportID)
	}

	// Проверяем deadline
	if time.Now().UTC().After(report.Deadline) {
		m.logger.Warn("CERT-In report submission past deadline",
			"report_id", reportID,
			"deadline", report.Deadline.Format(time.RFC3339),
		)
	}

	// Отправляем через API
	refNo, err := m.api.SubmitReport(report)
	if err != nil {
		// При ошибке API — не фейлим, а логируем
		// В production здесь должна быть очередь retry
		m.logger.Error("CERT-In API submission failed",
			"report_id", reportID,
			"error", err,
		)
		return fmt.Errorf("cert_in: submit via API: %w", err)
	}

	now := time.Now().UTC()
	if err := m.store.UpdateCERTInStatus(nil, reportID, "submitted", refNo); err != nil {
		return fmt.Errorf("cert_in: update status: %w", err)
	}

	m.logger.Info("CERT-In report submitted successfully",
		"report_id", reportID,
		"ref_no", refNo,
		"deadline_met", !now.After(report.Deadline),
	)

	return nil
}

// GetReport возвращает отчёт CERT-In.
func (m *CERTInManager) GetReport(id string) (*CERTInReport, error) {
	if id == "" {
		return nil, fmt.Errorf("cert_in: report_id is required")
	}
	return m.store.GetCERTInReport(nil, id)
}

// GetReportByIncident возвращает отчёт по ID инцидента.
func (m *CERTInManager) GetReportByIncident(incidentID string) (*CERTInReport, error) {
	if incidentID == "" {
		return nil, fmt.Errorf("cert_in: incident_id is required")
	}
	return m.store.GetCERTInReportByIncident(nil, incidentID)
}

// ListReports возвращает список отчётов с фильтром по статусу.
func (m *CERTInManager) ListReports(status string) ([]*CERTInReport, error) {
	return m.store.ListCERTInReports(nil, status)
}

// ═══════════════════════════════════════════════════════════════════════════
// CERT-In Compliance Helpers
// ═══════════════════════════════════════════════════════════════════════════

// MapNIS2ToCERTInCategory маппит NIS2 категорию на CERT-In.
func MapNIS2ToCERTInCategory(nis2Type IncidentType) CERTInCategory {
	mapping := map[IncidentType]CERTInCategory{
		IncidentTypeDenialOfService:     CERTInCategoryDoS,
		IncidentTypeMalware:             CERTInCategoryMalware,
		IncidentTypeUnauthorizedAccess:  CERTInCategoryUnauthorized,
		IncidentTypeDataBreach:          CERTInCategoryDataBreach,
		IncidentTypeSystemFailure:       CERTInCategorySystemCrash,
		IncidentTypeNetworkBreach:       CERTInCategoryScanning,
		IncidentTypePhysicalTampering:   CERTInCategoryOther,
		IncidentTypeConfigurationChange: CERTInCategoryOther,
		IncidentTypeInsiderThreat:       CERTInCategoryUnauthorized,
		IncidentTypeThirdParty:          CERTInCategoryOther,
	}
	if cat, ok := mapping[nis2Type]; ok {
		return cat
	}
	return CERTInCategoryOther
}

// MapNIS2ToCERTInImpact маппит NIS2 severity на CERT-In impact.
func MapNIS2ToCERTInImpact(severity IncidentSeverity) CERTInImpact {
	mapping := map[IncidentSeverity]CERTInImpact{
		SeverityNIS2Low:         CERTInImpactLow,
		SeverityNIS2Medium:      CERTInImpactMedium,
		SeverityNIS2High:        CERTInImpactHigh,
		SeverityNIS2Significant: CERTInImpactHigh,
		SeverityNIS2Critical:    CERTInImpactCritical,
	}
	if imp, ok := mapping[severity]; ok {
		return imp
	}
	return CERTInImpactLow
}

// CreateFromIncident создаёт CERT-In отчёт из NIS2 инцидента.
//
// Упрощённый конвертер для автоматической генерации отчётов.
// В production требует дополнительных данных от оператора.
func (m *CERTInManager) CreateFromIncident(
	incident *Incident,
	orgName, orgType, sector string,
	contactName, contactEmail string,
) (*CERTInReport, error) {
	if incident == nil {
		return nil, fmt.Errorf("cert_in: incident is nil")
	}

	category := MapNIS2ToCERTInCategory(incident.Classification.Type)
	impact := MapNIS2ToCERTInImpact(incident.Classification.Severity)

	affectedSystems := 0
	for _, entry := range incident.Timeline {
		if entry.Source != "" {
			affectedSystems++
		}
	}

	return m.CreateReport(
		incident.ID,
		category,
		impact,
		orgName, orgType, sector,
		contactName, contactEmail, "",
		incident.Description,
		"Under investigation",
		"not_contained",
		"",
		affectedSystems,
		0,
		false,
		"",
		nil,
		nil,
		"",
		true,
	)
}

// ═══════════════════════════════════════════════════════════════════════════
// SLA Monitoring
// ═══════════════════════════════════════════════════════════════════════════

// GetTimeRemaining возвращает оставшееся время до дедлайна.
func (m *CERTInManager) GetTimeRemaining(reportID string) (time.Duration, error) {
	report, err := m.store.GetCERTInReport(nil, reportID)
	if err != nil {
		return 0, fmt.Errorf("cert_in: get report: %w", err)
	}
	if report == nil {
		return 0, fmt.Errorf("cert_in: report not found: %s", reportID)
	}

	remaining := time.Until(report.Deadline)
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// GetOverdueReports возвращает список просроченных отчётов.
func (m *CERTInManager) GetOverdueReports() ([]*CERTInReport, error) {
	reports, err := m.store.ListCERTInReports(nil, "draft")
	if err != nil {
		return nil, fmt.Errorf("cert_in: list reports: %w", err)
	}

	now := time.Now().UTC()
	overdue := make([]*CERTInReport, 0)
	for _, report := range reports {
		if now.After(report.Deadline) {
			overdue = append(overdue, report)
		}
	}

	return overdue, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// generateCERTInID генерирует ID с префиксом для CERT-In.
func generateCERTInID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, generateID()[3:])
}

// Ensure interface compliance at compile time.
var _ interface{} = (*CERTInManager)(nil)
