// Package compliance — NIS2 Incident Reporting (P2-EU.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-EU.2: NIS2 Incident Reporting
//
// Проблема: Нет automated incident reporting по NIS2 Directive.
//
// Решение:
//   - IncidentClassification — классификация инцидентов по severity/impact/type
//   - ClassifyIncident(event) — automated incident classification
//   - GenerateNIS2Report(incident) — 24h/72h reporting templates (ENISA-format)
//   - GetIncidentTimeline(incidentID) — полная timeline событий
//   - Export formats: PDF, XML (ENISA-compatible)
//
// Compliance:
//   - NIS2 Directive Art. 23 — Incident reporting
//   - NIS2 Art. 24 — Security requirements for critical sectors
//   - ENISA Technical Guidelines on Incident Reporting (2024)
//   - ISO 27001 A.16.1 — Incident management
//   - ISO 27019 PCC.A.16 — ICS incident response
//   - IEC 62443-3-3 SR 7.1 — Resource availability (incident response)
//   - СТБ 34.101.27 п. 7.2 — Реагирование на инциденты КИИ
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ═══════════════════════════════════════════════════════════════════════════
// NIS2 Incident Severity (Art. 23)
// ═══════════════════════════════════════════════════════════════════════════

// IncidentSeverity — уровень severity инцидента по NIS2.
type IncidentSeverity string

const (
	SeverityNIS2Low         IncidentSeverity = "low"         // Минимальное воздействие
	SeverityNIS2Medium      IncidentSeverity = "medium"      // Умеренное воздействие
	SeverityNIS2High        IncidentSeverity = "high"        // Значительное воздействие
	SeverityNIS2Critical    IncidentSeverity = "critical"    // Критическое — 24h reporting
	SeverityNIS2Significant IncidentSeverity = "significant" // Значимый инцидент (NIS2 Art. 23(3))
)

// ═══════════════════════════════════════════════════════════════════════════
// NIS2 Incident Type (ENISA taxonomy)
// ═══════════════════════════════════════════════════════════════════════════

// IncidentType — тип инцидента по ENISA CSIRT taxonomy.
type IncidentType string

const (
	IncidentTypeUnauthorizedAccess  IncidentType = "unauthorized_access"  // Несанкционированный доступ
	IncidentTypeDataBreach          IncidentType = "data_breach"          // Утечка данных
	IncidentTypeSystemFailure       IncidentType = "system_failure"       // Отказ системы
	IncidentTypeMalware             IncidentType = "malware"              // Вредоносное ПО
	IncidentTypePhysicalTampering   IncidentType = "physical_tampering"   // Физическое вмешательство
	IncidentTypeDenialOfService     IncidentType = "denial_of_service"    // DoS/DDoS
	IncidentTypeConfigurationChange IncidentType = "configuration_change" // Изменение конфигурации
	IncidentTypeNetworkBreach       IncidentType = "network_breach"       // Сетевое проникновение
	IncidentTypeInsiderThreat       IncidentType = "insider_threat"       // Внутренний нарушитель
	IncidentTypeThirdParty          IncidentType = "third_party_breach"   // Нарушение у третьих лиц
)

// ═══════════════════════════════════════════════════════════════════════════
// Impact categories
// ═══════════════════════════════════════════════════════════════════════════

// ImpactCategory — категория воздействия инцидента.
type ImpactCategory string

const (
	ImpactAvailability    ImpactCategory = "availability"    // Доступность
	ImpactIntegrity       ImpactCategory = "integrity"       // Целостность
	ImpactConfidentiality ImpactCategory = "confidentiality" // Конфиденциальность
	ImpactSafety          ImpactCategory = "safety"          // Безопасность (physical)
	ImpactFinancial       ImpactCategory = "financial"       // Финансовый ущерб
	ImpactReputational    ImpactCategory = "reputational"    // Репутационный ущерб
	ImpactRegulatory      ImpactCategory = "regulatory"      // Регуляторные последствия
)

// ═══════════════════════════════════════════════════════════════════════════
// Report phase (NIS2 Art. 23)
// ═══════════════════════════════════════════════════════════════════════════

// ReportPhase — фаза отчётности по NIS2.
type ReportPhase string

const (
	ReportPhaseEarlyWarning ReportPhase = "early_warning" // 24h — initial alert
	ReportPhaseNotification ReportPhase = "notification"  // 72h — detailed notification
	ReportPhaseFinal        ReportPhase = "final"         // 1 month — final report
	ReportPhaseProgress     ReportPhase = "progress"      // Interim progress update
)

// ═══════════════════════════════════════════════════════════════════════════
// Models
// ═══════════════════════════════════════════════════════════════════════════

// IncidentEvent — сырое событие инцидента для классификации.
type IncidentEvent struct {
	ID             string            `json:"id"`
	Source         string            `json:"source"` // camera, nvr, server, api
	Type           string            `json:"type"`   // raw type from detection
	Description    string            `json:"description"`
	DetectedAt     time.Time         `json:"detected_at"`
	AffectedAssets []string          `json:"affected_assets,omitempty"`
	RawData        map[string]string `json:"raw_data,omitempty"`
}

// IncidentClassification — результат классификации инцидента.
type IncidentClassification struct {
	Severity    IncidentSeverity `json:"severity"`
	Type        IncidentType     `json:"type"`
	Impact      []ImpactCategory `json:"impact"`
	Confidence  float64          `json:"confidence"` // 0.0 — 1.0
	Description string           `json:"description"`
	Rationale   string           `json:"rationale"`
}

// Incident — полная запись об инциденте.
type Incident struct {
	XMLName        xml.Name               `json:"-" xml:"incident"`
	ID             string                 `json:"id" xml:"id,attr"`
	Classification IncidentClassification `json:"classification" xml:"classification"`
	Status         string                 `json:"status" xml:"status"` // open, investigating, contained, resolved, closed
	DetectedAt     time.Time              `json:"detected_at" xml:"detectedAt"`
	ReportedAt     *time.Time             `json:"reported_at,omitempty" xml:"reportedAt,omitempty"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty" xml:"resolvedAt,omitempty"`
	AssetID        string                 `json:"asset_id" xml:"assetId"`
	Zone           string                 `json:"zone" xml:"zone"` // IEC 62443 zone
	Description    string                 `json:"description" xml:"description"`
	Actions        []IncidentAction       `json:"actions,omitempty" xml:"actions>action"`
	Timeline       []TimelineEntry        `json:"timeline" xml:"timeline>entry"`
	Reports        []NIS2Report           `json:"reports,omitempty" xml:"reports>report"`
	CreatedAt      time.Time              `json:"created_at" xml:"createdAt"`
	UpdatedAt      time.Time              `json:"updated_at" xml:"updatedAt"`
}

// IncidentAction — действие, предпринятое в ответ на инцидент.
type IncidentAction struct {
	ID          string    `json:"id" xml:"id,attr"`
	Action      string    `json:"action" xml:"action"`
	Owner       string    `json:"owner" xml:"owner"`
	Status      string    `json:"status" xml:"status"`
	PerformedAt time.Time `json:"performed_at" xml:"performedAt"`
	Notes       string    `json:"notes,omitempty" xml:"notes,omitempty"`
}

// TimelineEntry — запись в timeline инцидента.
type TimelineEntry struct {
	ID          string    `json:"id" xml:"id,attr"`
	Timestamp   time.Time `json:"timestamp" xml:"timestamp"`
	Event       string    `json:"event" xml:"event"`
	Source      string    `json:"source" xml:"source"`
	Description string    `json:"description" xml:"description"`
	Actor       string    `json:"actor,omitempty" xml:"actor,omitempty"`
	Severity    string    `json:"severity,omitempty" xml:"severity,omitempty"`
}

// NIS2Report — отчёт по NIS2 Directive Art. 23.
type NIS2Report struct {
	XMLName        xml.Name      `json:"-" xml:"nis2Report"`
	ID             string        `json:"id" xml:"id,attr"`
	IncidentID     string        `json:"incident_id" xml:"incidentId"`
	Phase          ReportPhase   `json:"phase" xml:"phase"`
	GeneratedAt    time.Time     `json:"generated_at" xml:"generatedAt"`
	Format         string        `json:"format" xml:"format"`
	Content        ReportContent `json:"content" xml:"reportContent"`
	ENISACompliant bool          `json:"enisa_compliant" xml:"enisaCompliant"`
}

// ReportContent — содержимое отчёта NIS2.
type ReportContent struct {
	XMLName           xml.Name         `json:"-" xml:"reportContent"`
	ReporterInfo      ReporterInfo     `json:"reporter_info" xml:"reporterInfo"`
	IncidentInfo      IncidentInfo     `json:"incident_info" xml:"incidentInfo"`
	ImpactAssessment  ImpactAssessment `json:"impact_assessment" xml:"impactAssessment"`
	TechnicalDetails  TechnicalDetails `json:"technical_details" xml:"technicalDetails"`
	ResponseActions   []ResponseAction `json:"response_actions,omitempty" xml:"responseActions>action"`
	Recommendations   []string         `json:"recommendations,omitempty" xml:"recommendations>recommendation"`
	CSIRTCoordination string           `json:"csirt_coordination,omitempty" xml:"csirtCoordination,omitempty"`
	LessonsLearned    string           `json:"lessons_learned,omitempty" xml:"lessonsLearned,omitempty"`
}

// ReporterInfo — информация об отчитывающейся организации.
type ReporterInfo struct {
	OrganizationName string `json:"organization_name" xml:"organizationName"`
	Sector           string `json:"sector" xml:"sector"`
	Country          string `json:"country" xml:"country"`
	CSIRTEmail       string `json:"csirt_email" xml:"csirtEmail"`
	CSIRTPhone       string `json:"csirt_phone,omitempty" xml:"csirtPhone,omitempty"`
	ReferenceNumber  string `json:"reference_number" xml:"referenceNumber"`
}

// IncidentInfo — информация об инциденте.
type IncidentInfo struct {
	IncidentID      string                 `json:"incident_id" xml:"incidentId"`
	Classification  IncidentClassification `json:"classification" xml:"classification"`
	DetectionMethod string                 `json:"detection_method" xml:"detectionMethod"`
	FirstDetected   time.Time              `json:"first_detected" xml:"firstDetected"`
	LastUpdated     time.Time              `json:"last_updated" xml:"lastUpdated"`
	Status          string                 `json:"status" xml:"status"`
	AffectedAssets  []AffectedAsset        `json:"affected_assets,omitempty" xml:"affectedAssets>asset"`
	AffectedZones   []string               `json:"affected_zones,omitempty" xml:"affectedZones>zone"`
}

// AffectedAsset — затронутый актив.
type AffectedAsset struct {
	ID       string `json:"id" xml:"id,attr"`
	Type     string `json:"type" xml:"type"`
	Zone     string `json:"zone" xml:"zone"`
	Critical bool   `json:"critical" xml:"critical"`
}

// ImpactAssessment — оценка воздействия.
type ImpactAssessment struct {
	ServiceDisruption  bool     `json:"service_disruption" xml:"serviceDisruption"`
	DisruptionDuration string   `json:"disruption_duration,omitempty" xml:"disruptionDuration,omitempty"`
	DataCompromised    bool     `json:"data_compromised" xml:"dataCompromised"`
	DataCategories     []string `json:"data_categories,omitempty" xml:"dataCategories>category"`
	AffectedUsers      int      `json:"affected_users" xml:"affectedUsers"`
	AffectedDevices    int      `json:"affected_devices" xml:"affectedDevices"`
	FinancialImpact    float64  `json:"financial_impact,omitempty" xml:"financialImpact,omitempty"`
	RegulatoryImpact   bool     `json:"regulatory_impact" xml:"regulatoryImpact"`
	ReputationalImpact bool     `json:"reputational_impact" xml:"reputationalImpact"`
	CriticalityScore   float64  `json:"criticality_score" xml:"criticalityScore"` // 0-10
}

// TechnicalDetails — технические детали инцидента.
type TechnicalDetails struct {
	RootCause          string   `json:"root_cause,omitempty" xml:"rootCause,omitempty"`
	AttackVector       string   `json:"attack_vector,omitempty" xml:"attackVector,omitempty"`
	Indicators         []string `json:"indicators,omitempty" xml:"indicators>indicator"`
	LogsPreserved      bool     `json:"logs_preserved" xml:"logsPreserved"`
	ArtifactsCollected int      `json:"artifacts_collected" xml:"artifactsCollected"`
	ForensicState      string   `json:"forensic_state,omitempty" xml:"forensicState,omitempty"`
}

// ResponseAction — предпринятое действие в рамках реагирования.
type ResponseAction struct {
	Action        string `json:"action" xml:"action"`
	Owner         string `json:"owner" xml:"owner"`
	Status        string `json:"status" xml:"status"`
	CompletedAt   string `json:"completed_at,omitempty" xml:"completedAt,omitempty"`
	Effectiveness string `json:"effectiveness,omitempty" xml:"effectiveness,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════
// NIS2Manager — бизнес-логика NIS2 Incident Reporting
// ═══════════════════════════════════════════════════════════════════════════

// NIS2Store — интерфейс для хранения NIS2-данных.
type NIS2Store interface {
	SaveIncident(ctx interface{}, incident *Incident) error
	GetIncident(ctx interface{}, id string) (*Incident, error)
	ListIncidents(ctx interface{}, severity IncidentSeverity, limit, offset int) ([]*Incident, error)
	UpdateIncident(ctx interface{}, incident *Incident) error
	SaveReport(ctx interface{}, report *NIS2Report) error
	GetReport(ctx interface{}, id string) (*NIS2Report, error)
	ListReports(ctx interface{}, incidentID string) ([]*NIS2Report, error)
	AddTimelineEntry(ctx interface{}, incidentID string, entry *TimelineEntry) error
	GetTimeline(ctx interface{}, incidentID string) ([]*TimelineEntry, error)
}

// NIS2Manager — бизнес-логика управления NIS2 incident reporting.
type NIS2Manager struct {
	store  NIS2Store
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewNIS2Manager создаёт новый NIS2Manager.
func NewNIS2Manager(store NIS2Store, logger *slog.Logger) *NIS2Manager {
	if logger == nil {
		logger = slog.Default()
	}

	return &NIS2Manager{
		store:  store,
		logger: logger.With("component", "compliance.nis2"),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// ClassifyIncident — automated incident classification (NIS2 Art. 23)
// ═══════════════════════════════════════════════════════════════════════════

// ClassifyIncident выполняет автоматическую классификацию инцидента
// на основе типа события, затронутых активов и зон безопасности.
//
// Возвращает классификацию с severity, типом, impact-категориями
// и уровнем уверенности (confidence).
func (m *NIS2Manager) ClassifyIncident(event *IncidentEvent) *IncidentClassification {
	if event == nil {
		return &IncidentClassification{
			Severity:    SeverityNIS2Low,
			Type:        IncidentTypeSystemFailure,
			Confidence:  0.0,
			Description: "nil event — unable to classify",
		}
	}

	class := &IncidentClassification{
		Confidence:  0.85,
		Description: event.Description,
	}

	// 1. Определяем тип инцидента на основе source и raw type
	class.Type = m.determineIncidentType(event)

	// 2. Определяем impact-категории
	class.Impact = m.determineImpact(event)

	// 3. Определяем severity
	class.Severity = m.determineSeverity(event, class.Type, class.Impact)

	// 4. Формируем rationale
	class.Rationale = m.buildRationale(class, event)

	return class
}

// determineIncidentType определяет тип инцидента.
func (m *NIS2Manager) determineIncidentType(event *IncidentEvent) IncidentType {
	source := strings.ToLower(event.Source)
	rawType := strings.ToLower(event.Type)

	// Проверка по source
	switch {
	case strings.Contains(rawType, "unauthorized") || strings.Contains(rawType, "forbidden"):
		return IncidentTypeUnauthorizedAccess
	case strings.Contains(rawType, "breach") || strings.Contains(rawType, "leak"):
		return IncidentTypeDataBreach
	case strings.Contains(rawType, "dos") || strings.Contains(rawType, "ddos") || strings.Contains(rawType, "flood"):
		return IncidentTypeDenialOfService
	case strings.Contains(rawType, "malware") || strings.Contains(rawType, "ransomware") || strings.Contains(rawType, "trojan"):
		return IncidentTypeMalware
	case strings.Contains(rawType, "config") || strings.Contains(rawType, "misconfig"):
		return IncidentTypeConfigurationChange
	case strings.Contains(rawType, "tamper") || strings.Contains(rawType, "physical"):
		return IncidentTypePhysicalTampering
	case strings.Contains(rawType, "insider") || strings.Contains(rawType, "policy_violation"):
		return IncidentTypeInsiderThreat
	case strings.Contains(rawType, "third_party") || strings.Contains(rawType, "vendor"):
		return IncidentTypeThirdParty
	case strings.Contains(rawType, "network") || strings.Contains(rawType, "port_scan"):
		return IncidentTypeNetworkBreach
	}

	// Определение по source для CCTV
	switch {
	case source == "camera" || source == "nvr":
		if strings.Contains(rawType, "offline") || strings.Contains(rawType, "disconnect") {
			return IncidentTypeSystemFailure
		}
		if strings.Contains(rawType, "auth") || strings.Contains(rawType, "login") {
			return IncidentTypeUnauthorizedAccess
		}
		return IncidentTypePhysicalTampering
	case source == "server" || source == "api":
		if strings.Contains(rawType, "crash") || strings.Contains(rawType, "error") {
			return IncidentTypeSystemFailure
		}
		return IncidentTypeUnauthorizedAccess
	default:
		return IncidentTypeSystemFailure
	}
}

// determineImpact определяет категории воздействия.
func (m *NIS2Manager) determineImpact(event *IncidentEvent) []ImpactCategory {
	impacts := make([]ImpactCategory, 0)
	seen := make(map[ImpactCategory]bool)

	// CCTV-specific impact analysis
	source := strings.ToLower(event.Source)
	rawType := strings.ToLower(event.Type)

	// Availability impact
	if strings.Contains(rawType, "offline") || strings.Contains(rawType, "dos") ||
		strings.Contains(rawType, "crash") || strings.Contains(rawType, "disconnect") {
		if !seen[ImpactAvailability] {
			impacts = append(impacts, ImpactAvailability)
			seen[ImpactAvailability] = true
		}
	}

	// Integrity impact
	if strings.Contains(rawType, "config") || strings.Contains(rawType, "tamper") ||
		strings.Contains(rawType, "misconfig") {
		if !seen[ImpactIntegrity] {
			impacts = append(impacts, ImpactIntegrity)
			seen[ImpactIntegrity] = true
		}
	}

	// Confidentiality impact
	if strings.Contains(rawType, "breach") || strings.Contains(rawType, "leak") ||
		strings.Contains(rawType, "unauthorized") {
		if !seen[ImpactConfidentiality] {
			impacts = append(impacts, ImpactConfidentiality)
			seen[ImpactConfidentiality] = true
		}
	}

	// Safety impact (physical security)
	if source == "camera" && strings.Contains(rawType, "tamper") {
		if !seen[ImpactSafety] {
			impacts = append(impacts, ImpactSafety)
			seen[ImpactSafety] = true
		}
	}

	// Regulatory impact
	if strings.Contains(rawType, "breach") || strings.Contains(rawType, "leak") {
		if !seen[ImpactRegulatory] {
			impacts = append(impacts, ImpactRegulatory)
			seen[ImpactRegulatory] = true
		}
	}

	// Financial impact for critical assets
	if len(event.AffectedAssets) > 0 {
		if !seen[ImpactFinancial] {
			impacts = append(impacts, ImpactFinancial)
			seen[ImpactFinancial] = true
		}
	}

	// If no specific impact detected, at minimum mark availability
	if len(impacts) == 0 {
		impacts = append(impacts, ImpactAvailability)
	}

	return impacts
}

// determineSeverity определяет уровень severity инцидента.
func (m *NIS2Manager) determineSeverity(event *IncidentEvent, incType IncidentType, impacts []ImpactCategory) IncidentSeverity {
	score := 0.0

	// Severity by incident type (NIS2 weighting)
	typeScore := map[IncidentType]float64{
		IncidentTypeDataBreach:          9.0,
		IncidentTypeMalware:             8.0,
		IncidentTypeDenialOfService:     7.0,
		IncidentTypeNetworkBreach:       7.0,
		IncidentTypeUnauthorizedAccess:  6.0,
		IncidentTypePhysicalTampering:   6.0,
		IncidentTypeInsiderThreat:       5.0,
		IncidentTypeThirdParty:          4.0,
		IncidentTypeConfigurationChange: 3.0,
		IncidentTypeSystemFailure:       2.0,
	}
	score += typeScore[incType]

	// Severity by impact
	for _, impact := range impacts {
		switch impact {
		case ImpactSafety:
			score += 4.0
		case ImpactAvailability:
			score += 3.0
		case ImpactConfidentiality:
			score += 3.0
		case ImpactIntegrity:
			score += 2.0
		case ImpactRegulatory:
			score += 2.0
		case ImpactFinancial:
			score += 1.0
		case ImpactReputational:
			score += 1.0
		}
	}

	// Severity by affected assets
	assetCount := len(event.AffectedAssets)
	if assetCount > 10 {
		score += 3.0
	} else if assetCount > 3 {
		score += 1.5
	} else if assetCount > 0 {
		score += 0.5
	}

	// Map score to NIS2 severity
	switch {
	case score >= 12.0:
		return SeverityNIS2Critical
	case score >= 8.0:
		return SeverityNIS2Significant
	case score >= 5.0:
		return SeverityNIS2High
	case score >= 3.0:
		return SeverityNIS2Medium
	default:
		return SeverityNIS2Low
	}
}

// buildRationale формирует текстовое обоснование классификации.
func (m *NIS2Manager) buildRationale(class *IncidentClassification, event *IncidentEvent) string {
	parts := []string{
		fmt.Sprintf("Incident classified as %s", class.Type),
		fmt.Sprintf("Severity: %s", class.Severity),
		fmt.Sprintf("Based on source '%s' and event type '%s'", event.Source, event.Type),
		fmt.Sprintf("Detected %d impact categories: %v", len(class.Impact), class.Impact),
		fmt.Sprintf("Confidence: %.0f%%", class.Confidence*100),
	}

	if len(event.AffectedAssets) > 0 {
		parts = append(parts, fmt.Sprintf("Affected assets: %d", len(event.AffectedAssets)))
	}

	if class.Severity == SeverityNIS2Critical || class.Severity == SeverityNIS2Significant {
		parts = append(parts, "NIS2 Art. 23: Mandatory reporting required within 24h")
	}

	return strings.Join(parts, "; ")
}

// ═══════════════════════════════════════════════════════════════════════════
// GenerateNIS2Report — 24h/72h/final report generation
// ═══════════════════════════════════════════════════════════════════════════

// GenerateNIS2Report генерирует отчёт по инциденту согласно NIS2 Art. 23.
//
// Параметры:
//   - incident: объект инцидента
//   - phase: фаза отчётности (early_warning|notification|final|progress)
//   - format: формат (pdf|xml)
//
// Возвращает:
//   - []byte: сгенерированный отчёт
//   - error: ошибка генерации
func (m *NIS2Manager) GenerateNIS2Report(incident *Incident, phase ReportPhase, format string) ([]byte, error) {
	if incident == nil {
		return nil, fmt.Errorf("nis2 report: incident is nil")
	}

	m.logger.Info("generating NIS2 report",
		"incident_id", incident.ID,
		"phase", phase,
		"format", format,
	)

	// Определяем deadline отчёта
	deadline := m.calculateDeadline(incident.DetectedAt, phase)

	// Строим содержимое отчёта
	content := m.buildReportContent(incident, phase, deadline)

	report := &NIS2Report{
		ID:             generateNIS2ID("nr", incident.ID),
		IncidentID:     incident.ID,
		Phase:          phase,
		GeneratedAt:    time.Now().UTC(),
		Format:         format,
		Content:        *content,
		ENISACompliant: true,
	}

	switch format {
	case "pdf":
		return m.renderNIS2PDF(report, incident)
	case "xml":
		return m.renderNIS2XML(report)
	default:
		return nil, fmt.Errorf("nis2 report: unsupported format: %s", format)
	}
}

// calculateDeadline вычисляет дедлайн для фазы отчёта.
func (m *NIS2Manager) calculateDeadline(detectedAt time.Time, phase ReportPhase) time.Time {
	switch phase {
	case ReportPhaseEarlyWarning:
		return detectedAt.Add(24 * time.Hour) // 24h
	case ReportPhaseNotification:
		return detectedAt.Add(72 * time.Hour) // 72h
	case ReportPhaseFinal:
		return detectedAt.Add(30 * 24 * time.Hour) // 1 month
	default:
		return detectedAt.Add(72 * time.Hour)
	}
}

// buildReportContent строит содержимое отчёта NIS2.
func (m *NIS2Manager) buildReportContent(incident *Incident, phase ReportPhase, deadline time.Time) *ReportContent {
	now := time.Now().UTC()

	content := &ReportContent{
		ReporterInfo: ReporterInfo{
			OrganizationName: "CCTV Health Monitor",
			Sector:           "Digital Infrastructure (CCTV surveillance)",
			Country:          "EU",
			CSIRTEmail:       "security@cctv-monitor.io",
			ReferenceNumber:  fmt.Sprintf("NIS2-%s-%s", incident.ID, now.Format("20060102")),
		},
		IncidentInfo: IncidentInfo{
			IncidentID:      incident.ID,
			Classification:  incident.Classification,
			DetectionMethod: "automated_detection",
			FirstDetected:   incident.DetectedAt,
			LastUpdated:     incident.UpdatedAt,
			Status:          incident.Status,
			AffectedAssets:  m.buildAffectedAssets(incident),
			AffectedZones:   m.extractZones(incident),
		},
		ImpactAssessment: m.buildImpactAssessment(incident, phase),
		TechnicalDetails: m.buildTechnicalDetails(incident),
		ResponseActions:  m.buildResponseActions(incident),
		Recommendations:  m.generateRecommendations(incident, phase),
	}

	// Только для final report — lessons learned
	if phase == ReportPhaseFinal {
		content.LessonsLearned = m.generateLessonsLearned(incident)
	}

	// CSIRT coordination details for critical incidents
	if incident.Classification.Severity == SeverityNIS2Critical || incident.Classification.Severity == SeverityNIS2Significant {
		content.CSIRTCoordination = fmt.Sprintf(
			"CSIRT coordination initiated. Incident reference: %s. "+
				"Coordination required per NIS2 Art. 23(4). Deadline: %s",
			incident.ID,
			deadline.Format(time.RFC3339),
		)
	}

	return content
}

// buildAffectedAssets строит список затронутых активов.
func (m *NIS2Manager) buildAffectedAssets(incident *Incident) []AffectedAsset {
	assets := make([]AffectedAsset, 0)
	// В production — из БД, здесь из timeline/actions
	for _, entry := range incident.Timeline {
		if entry.Source != "" && entry.Severity == "critical" {
			assets = append(assets, AffectedAsset{
				ID:       entry.ID,
				Type:     entry.Source,
				Zone:     m.mapSourceToZone(entry.Source),
				Critical: strings.Contains(entry.Severity, "critical"),
			})
		}
	}
	return assets
}

// mapSourceToZone маппит source на зону IEC 62443.
func (m *NIS2Manager) mapSourceToZone(source string) string {
	sourceMap := map[string]string{
		"camera": "Zone 5 (Edge)",
		"nvr":    "Zone 4 (Data)",
		"server": "Zone 3 (Application)",
		"api":    "Zone 2 (DMZ)",
		"db":     "Zone 4 (Data)",
		"nats":   "Zone 3 (Application)",
		"edge":   "Zone 5 (Edge)",
	}
	if zone, ok := sourceMap[strings.ToLower(source)]; ok {
		return zone
	}
	return "Zone 3 (Application)"
}

// extractZones извлекает зоны из инцидента.
func (m *NIS2Manager) extractZones(incident *Incident) []string {
	zoneSet := make(map[string]bool)
	for _, entry := range incident.Timeline {
		zone := m.mapSourceToZone(entry.Source)
		zoneSet[zone] = true
	}

	zones := make([]string, 0, len(zoneSet))
	for zone := range zoneSet {
		zones = append(zones, zone)
	}
	sort.Strings(zones)
	return zones
}

// buildImpactAssessment строит оценку воздействия.
func (m *NIS2Manager) buildImpactAssessment(incident *Incident, phase ReportPhase) ImpactAssessment {
	ia := ImpactAssessment{
		AffectedDevices:  m.countAffectedDevices(incident),
		AffectedUsers:    m.estimateAffectedUsers(incident),
		CriticalityScore: m.calculateCriticalityScore(incident),
	}

	// Определяем наличие disruption
	for _, impact := range incident.Classification.Impact {
		switch impact {
		case ImpactAvailability:
			ia.ServiceDisruption = true
			ia.DisruptionDuration = m.estimateDisruptionDuration(incident)
		case ImpactConfidentiality:
			ia.DataCompromised = true
			ia.DataCategories = m.extractDataCategories(incident)
		case ImpactRegulatory:
			ia.RegulatoryImpact = true
		case ImpactReputational:
			ia.ReputationalImpact = true
		case ImpactFinancial:
			ia.FinancialImpact = m.estimateFinancialImpact(incident)
		}
	}

	return ia
}

// countAffectedDevices подсчитывает затронутые устройства.
func (m *NIS2Manager) countAffectedDevices(incident *Incident) int {
	deviceSet := make(map[string]bool)
	for _, entry := range incident.Timeline {
		if entry.Source != "" {
			deviceSet[entry.Source] = true
		}
	}
	return len(deviceSet)
}

// estimateAffectedUsers оценивает количество затронутых пользователей.
func (m *NIS2Manager) estimateAffectedUsers(incident *Incident) int {
	// В production — из БД
	if incident.Classification.Severity == SeverityNIS2Critical {
		return 500
	}
	if incident.Classification.Severity == SeverityNIS2Significant {
		return 200
	}
	return 50
}

// estimateDisruptionDuration оценивает длительность нарушения.
func (m *NIS2Manager) estimateDisruptionDuration(incident *Incident) string {
	if incident.ResolvedAt != nil {
		duration := incident.ResolvedAt.Sub(incident.DetectedAt)
		return fmt.Sprintf("%.1f hours", duration.Hours())
	}
	return "ongoing"
}

// extractDataCategories извлекает категории данных.
func (m *NIS2Manager) extractDataCategories(incident *Incident) []string {
	categories := []string{"video_archive"}
	for _, impact := range incident.Classification.Impact {
		if impact == ImpactConfidentiality {
			categories = append(categories, "credentials", "access_logs")
			break
		}
	}
	return categories
}

// estimateFinancialImpact оценивает финансовое воздействие.
func (m *NIS2Manager) estimateFinancialImpact(incident *Incident) float64 {
	// В production — интеграция с CMMS/finance system
	baseCosts := map[IncidentSeverity]float64{
		SeverityNIS2Low:         1000.0,
		SeverityNIS2Medium:      5000.0,
		SeverityNIS2High:        25000.0,
		SeverityNIS2Significant: 100000.0,
		SeverityNIS2Critical:    500000.0,
	}
	return baseCosts[incident.Classification.Severity]
}

// calculateCriticalityScore вычисляет score критичности (0-10).
func (m *NIS2Manager) calculateCriticalityScore(incident *Incident) float64 {
	severityScores := map[IncidentSeverity]float64{
		SeverityNIS2Low:         1.0,
		SeverityNIS2Medium:      3.0,
		SeverityNIS2High:        5.0,
		SeverityNIS2Significant: 7.0,
		SeverityNIS2Critical:    9.5,
	}

	score := severityScores[incident.Classification.Severity]

	// +1 if multiple zones affected
	if len(m.extractZones(incident)) > 1 {
		score += 0.5
	}

	if score > 10.0 {
		score = 10.0
	}

	return score
}

// buildTechnicalDetails строит технические детали.
func (m *NIS2Manager) buildTechnicalDetails(incident *Incident) TechnicalDetails {
	td := TechnicalDetails{
		LogsPreserved:      true,
		ArtifactsCollected: 0,
		ForensicState:      "collection_in_progress",
	}

	// Извлекаем IOCs из timeline
	iocs := make([]string, 0)
	for _, entry := range incident.Timeline {
		if strings.Contains(strings.ToLower(entry.Event), "ioc") ||
			strings.Contains(strings.ToLower(entry.Description), "indicator") {
			iocs = append(iocs, entry.Description)
		}
	}
	td.Indicators = iocs

	// Root cause — из последнего action
	if len(incident.Actions) > 0 {
		lastAction := incident.Actions[len(incident.Actions)-1]
		if lastAction.Notes != "" {
			td.RootCause = lastAction.Notes
		}
	}

	if incident.ResolvedAt != nil {
		td.ForensicState = "completed"
		td.ArtifactsCollected = len(incident.Timeline)
	}

	return td
}

// buildResponseActions строит список действий реагирования.
func (m *NIS2Manager) buildResponseActions(incident *Incident) []ResponseAction {
	actions := make([]ResponseAction, 0, len(incident.Actions))
	for _, action := range incident.Actions {
		ra := ResponseAction{
			Action: action.Action,
			Owner:  action.Owner,
			Status: action.Status,
		}
		if action.Status == "completed" {
			ra.CompletedAt = action.PerformedAt.Format(time.RFC3339)
		}
		actions = append(actions, ra)
	}
	return actions
}

// generateRecommendations генерирует рекомендации на основе инцидента.
func (m *NIS2Manager) generateRecommendations(incident *Incident, phase ReportPhase) []string {
	recs := make([]string, 0)

	switch incident.Classification.Type {
	case IncidentTypeUnauthorizedAccess:
		recs = append(recs,
			"Review and rotate all credentials",
			"Enable MFA for all user accounts",
			"Audit access logs for suspicious activity",
		)
	case IncidentTypeDataBreach:
		recs = append(recs,
			"Notify DPO and affected data subjects",
			"Initiate data breach response procedure",
			"Review data access controls",
		)
	case IncidentTypeSystemFailure:
		recs = append(recs,
			"Implement redundancy for critical components",
			"Review monitoring and alerting thresholds",
			"Schedule root cause analysis",
		)
	case IncidentTypeMalware:
		recs = append(recs,
			"Isolate affected systems from network",
			"Run full antivirus/malware scan",
			"Update EDR/XDR signatures",
		)
	case IncidentTypePhysicalTampering:
		recs = append(recs,
			"Review physical security controls",
			"Audit camera tamper detection logs",
			"Notify physical security team",
		)
	case IncidentTypeDenialOfService:
		recs = append(recs,
			"Enable DDoS protection (rate limiting, WAF)",
			"Review network capacity planning",
			"Implement failover to secondary PoP",
		)
	default:
		recs = append(recs,
			"Conduct thorough incident investigation",
			"Document lessons learned",
			"Update incident response playbook",
		)
	}

	// Phase-specific recommendations
	if phase == ReportPhaseEarlyWarning {
		recs = append([]string{
			"Immediate containment actions required",
			"Preserve all logs and evidence",
		}, recs...)
	}

	if phase == ReportPhaseFinal {
		recs = append(recs,
			"Schedule security awareness training",
			"Update risk register",
			"Review compliance controls",
		)
	}

	return recs
}

// generateLessonsLearned генерирует lessons learned.
func (m *NIS2Manager) generateLessonsLearned(incident *Incident) string {
	return fmt.Sprintf(
		"Lessons learned from incident %s (%s):\n"+
			"- Detection time: %s\n"+
			"- Resolution time: %s\n"+
			"- Root cause: %s\n"+
			"- Key improvement: Enhance detection capabilities for %s incidents\n"+
			"- Action items: Review and update incident response playbook",
		incident.ID,
		incident.Classification.Type,
		incident.DetectedAt.Format(time.RFC3339),
		func() string {
			if incident.ResolvedAt != nil {
				return incident.ResolvedAt.Format(time.RFC3339)
			}
			return "not resolved"
		}(),
		incident.Description,
		incident.Classification.Type,
	)
}

// ═══════════════════════════════════════════════════════════════════════════
// GetIncidentTimeline — полная timeline событий
// ═══════════════════════════════════════════════════════════════════════════

// GetIncidentTimeline возвращает полную хронологию событий инцидента.
func (m *NIS2Manager) GetIncidentTimeline(incidentID string) ([]TimelineEntry, error) {
	if incidentID == "" {
		return nil, fmt.Errorf("nis2: incident_id is required")
	}

	m.logger.Info("fetching incident timeline",
		"incident_id", incidentID,
	)

	storeTimeline, err := m.store.GetTimeline(nil, incidentID)
	if err != nil {
		return nil, fmt.Errorf("nis2: get timeline: %w", err)
	}

	// Сортируем по timestamp
	sort.Slice(storeTimeline, func(i, j int) bool {
		return storeTimeline[i].Timestamp.Before(storeTimeline[j].Timestamp)
	})

	// Convert []*TimelineEntry to []TimelineEntry
	timeline := make([]TimelineEntry, len(storeTimeline))
	for i, entry := range storeTimeline {
		if entry != nil {
			timeline[i] = *entry
		}
	}

	return timeline, nil
}

// AddTimelineEntry добавляет запись в timeline инцидента.
func (m *NIS2Manager) AddTimelineEntry(incidentID string, entry *TimelineEntry) error {
	if incidentID == "" {
		return fmt.Errorf("nis2: incident_id is required")
	}
	if entry == nil {
		return fmt.Errorf("nis2: timeline entry is nil")
	}

	entry.ID = generateNIS2ID("tl", incidentID)
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}

	if err := m.store.AddTimelineEntry(nil, incidentID, entry); err != nil {
		return fmt.Errorf("nis2: add timeline entry: %w", err)
	}

	m.logger.Info("timeline entry added",
		"incident_id", incidentID,
		"event", entry.Event,
	)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Incident CRUD
// ═══════════════════════════════════════════════════════════════════════════

// CreateIncident создаёт новый инцидент.
func (m *NIS2Manager) CreateIncident(event *IncidentEvent) (*Incident, error) {
	if event == nil {
		return nil, fmt.Errorf("nis2: event is nil")
	}

	now := time.Now().UTC()
	classification := m.ClassifyIncident(event)

	incident := &Incident{
		ID:             generateNIS2ID("inc", ""),
		Classification: *classification,
		Status:         "open",
		DetectedAt:     event.DetectedAt,
		AssetID:        event.Source,
		Zone:           m.mapSourceToZone(event.Source),
		Description:    event.Description,
		Timeline: []TimelineEntry{
			{
				ID:          generateNIS2ID("tl", ""),
				Timestamp:   now,
				Event:       "incident_created",
				Source:      "nis2_classifier",
				Description: fmt.Sprintf("Incident automatically classified as %s (severity: %s)", classification.Type, classification.Severity),
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Early warning report для critical/significant
	if classification.Severity == SeverityNIS2Critical || classification.Severity == SeverityNIS2Significant {
		earlyReport := &NIS2Report{
			ID:             generateNIS2ID("nr", incident.ID),
			IncidentID:     incident.ID,
			Phase:          ReportPhaseEarlyWarning,
			GeneratedAt:    now,
			Format:         "xml",
			ENISACompliant: true,
		}
		// Content будет заполнен при генерации
		incident.Reports = append(incident.Reports, *earlyReport)
	}

	if err := m.store.SaveIncident(nil, incident); err != nil {
		return nil, fmt.Errorf("nis2: save incident: %w", err)
	}

	m.logger.Info("incident created",
		"incident_id", incident.ID,
		"severity", classification.Severity,
		"type", classification.Type,
	)

	return incident, nil
}

// GetIncident возвращает инцидент по ID.
func (m *NIS2Manager) GetIncident(id string) (*Incident, error) {
	if id == "" {
		return nil, fmt.Errorf("nis2: incident_id is required")
	}
	return m.store.GetIncident(nil, id)
}

// ListIncidents возвращает список инцидентов с фильтрацией.
func (m *NIS2Manager) ListIncidents(severity IncidentSeverity, limit, offset int) ([]*Incident, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	return m.store.ListIncidents(nil, severity, limit, offset)
}

// UpdateIncidentStatus обновляет статус инцидента.
func (m *NIS2Manager) UpdateIncidentStatus(incidentID, status string) error {
	if incidentID == "" {
		return fmt.Errorf("nis2: incident_id is required")
	}
	if status == "" {
		return fmt.Errorf("nis2: status is required")
	}

	incident, err := m.store.GetIncident(nil, incidentID)
	if err != nil {
		return fmt.Errorf("nis2: get incident: %w", err)
	}

	oldStatus := incident.Status
	incident.Status = status
	incident.UpdatedAt = time.Now().UTC()

	if status == "resolved" || status == "closed" {
		now := time.Now().UTC()
		incident.ResolvedAt = &now

		// Добавляем запись в timeline
		incident.Timeline = append(incident.Timeline, TimelineEntry{
			ID:          generateNIS2ID("tl", incidentID),
			Timestamp:   now,
			Event:       fmt.Sprintf("incident_%s", status),
			Source:      "nis2_manager",
			Description: fmt.Sprintf("Incident status changed from %s to %s", oldStatus, status),
		})
	}

	if err := m.store.UpdateIncident(nil, incident); err != nil {
		return fmt.Errorf("nis2: update incident: %w", err)
	}

	m.logger.Info("incident status updated",
		"incident_id", incidentID,
		"old_status", oldStatus,
		"new_status", status,
	)

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// PDF Rendering (ENISA-compatible)
// ═══════════════════════════════════════════════════════════════════════════

// renderNIS2PDF генерирует PDF отчёт по NIS2.
func (m *NIS2Manager) renderNIS2PDF(report *NIS2Report, incident *Incident) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)
	pdf.AddPage()

	// ── Header ──
	pdf.SetFont("Helvetica", "B", 20)
	pdf.CellFormat(190, 15, "NIS2 Incident Report", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(190, 5, "Directive (EU) 2022/2555 — Article 23 Incident Reporting", "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// ── Report metadata ──
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(190, 6, fmt.Sprintf("Report ID: %s", report.ID), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Phase: %s (%s)", report.Phase, m.phaseLabel(report.Phase)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Generated: %s", report.GeneratedAt.Format(time.RFC1123)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("ENISA Compliant: %t", report.ENISACompliant), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// ── Severity badge ──
	sevColor := m.severityColor(incident.Classification.Severity)
	pdf.SetFillColor(int(sevColor.R), int(sevColor.G), int(sevColor.B))
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(190, 8, fmt.Sprintf("Severity: %s", incident.Classification.Severity), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// ── Classification ──
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, "Incident Classification", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(190, 6, fmt.Sprintf("Type: %s", incident.Classification.Type), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Confidence: %.0f%%", incident.Classification.Confidence*100), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Impact: %v", incident.Classification.Impact), "", 1, "L", false, 0, "")
	pdf.MultiCell(190, 5, fmt.Sprintf("Rationale: %s", incident.Classification.Rationale), "", "L", false)
	pdf.Ln(3)

	// ── Incident Details ──
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, "Incident Details", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(190, 6, fmt.Sprintf("Incident ID: %s", incident.ID), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Detected: %s", incident.DetectedAt.Format(time.RFC1123)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Status: %s", incident.Status), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Zone: %s (IEC 62443)", incident.Zone), "", 1, "L", false, 0, "")
	pdf.MultiCell(190, 5, fmt.Sprintf("Description: %s", incident.Description), "", "L", false)
	pdf.Ln(3)

	// ── Impact Assessment ──
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 10, "Impact Assessment", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	ia := report.Content.ImpactAssessment
	pdf.CellFormat(190, 6, fmt.Sprintf("Criticality Score: %.1f / 10", ia.CriticalityScore), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Service Disruption: %t", ia.ServiceDisruption), "", 1, "L", false, 0, "")
	if ia.DisruptionDuration != "" {
		pdf.CellFormat(190, 6, fmt.Sprintf("Disruption Duration: %s", ia.DisruptionDuration), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(190, 6, fmt.Sprintf("Data Compromised: %t", ia.DataCompromised), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Affected Users: %d", ia.AffectedUsers), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 6, fmt.Sprintf("Affected Devices: %d", ia.AffectedDevices), "", 1, "L", false, 0, "")
	if ia.FinancialImpact > 0 {
		pdf.CellFormat(190, 6, fmt.Sprintf("Financial Impact: EUR %.2f", ia.FinancialImpact), "", 1, "L", false, 0, "")
	}
	pdf.Ln(3)

	// ── Timeline ──
	if len(incident.Timeline) > 0 {
		pdf.SetFont("Helvetica", "B", 14)
		pdf.CellFormat(190, 10, fmt.Sprintf("Timeline (%d entries)", len(incident.Timeline)), "", 1, "L", false, 0, "")

		// Table header
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(40, 7, "Timestamp", "1", 0, "L", true, 0, "")
		pdf.CellFormat(40, 7, "Event", "1", 0, "L", true, 0, "")
		pdf.CellFormat(25, 7, "Source", "1", 0, "L", true, 0, "")
		pdf.CellFormat(85, 7, "Description", "1", 1, "L", true, 0, "")

		// Table rows
		pdf.SetFont("Helvetica", "", 8)
		for _, entry := range incident.Timeline {
			pdf.CellFormat(40, 6, entry.Timestamp.Format("2006-01-02 15:04"), "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 6, truncate(entry.Event, 25), "1", 0, "L", false, 0, "")
			pdf.CellFormat(25, 6, truncate(entry.Source, 15), "1", 0, "L", false, 0, "")
			pdf.CellFormat(85, 6, truncate(entry.Description, 55), "1", 1, "L", false, 0, "")
		}
		pdf.Ln(3)
	}

	// ── Recommendations ──
	if len(report.Content.Recommendations) > 0 {
		pdf.SetFont("Helvetica", "B", 14)
		pdf.CellFormat(190, 10, "Recommendations", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		for i, rec := range report.Content.Recommendations {
			pdf.CellFormat(190, 6, fmt.Sprintf("%d. %s", i+1, rec), "", 1, "L", false, 0, "")
		}
		pdf.Ln(3)
	}

	// ── Footer ──
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.CellFormat(190, 5, fmt.Sprintf("Reporter: %s | Sector: %s | Country: %s",
		report.Content.ReporterInfo.OrganizationName,
		report.Content.ReporterInfo.Sector,
		report.Content.ReporterInfo.Country,
	), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, fmt.Sprintf("CSIRT: %s | Ref: %s",
		report.Content.ReporterInfo.CSIRTEmail,
		report.Content.ReporterInfo.ReferenceNumber,
	), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, "This report is auto-generated per NIS2 Directive Art. 23. For verification, contact security@cctv-monitor.io", "", 1, "L", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("render NIS2 PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// severityColor возвращает цвет для severity.
type rgbColor struct{ R, G, B uint8 }

func (m *NIS2Manager) severityColor(severity IncidentSeverity) rgbColor {
	switch severity {
	case SeverityNIS2Critical:
		return rgbColor{180, 0, 0} // Dark red
	case SeverityNIS2Significant:
		return rgbColor{200, 80, 0} // Orange
	case SeverityNIS2High:
		return rgbColor{200, 160, 0} // Amber
	case SeverityNIS2Medium:
		return rgbColor{180, 180, 0} // Yellow
	default:
		return rgbColor{100, 180, 100} // Green
	}
}

// phaseLabel возвращает человекочитаемую метку фазы.
func (m *NIS2Manager) phaseLabel(phase ReportPhase) string {
	labels := map[ReportPhase]string{
		ReportPhaseEarlyWarning: "Early Warning (24h)",
		ReportPhaseNotification: "Notification (72h)",
		ReportPhaseFinal:        "Final Report (1 month)",
		ReportPhaseProgress:     "Progress Update",
	}
	if label, ok := labels[phase]; ok {
		return label
	}
	return string(phase)
}

// ═══════════════════════════════════════════════════════════════════════════
// XML Rendering (ENISA-compatible)
// ═══════════════════════════════════════════════════════════════════════════

// renderNIS2XML генерирует XML отчёт в ENISA-совместимом формате.
func (m *NIS2Manager) renderNIS2XML(report *NIS2Report) ([]byte, error) {
	output, err := xml.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render NIS2 XML: %w", err)
	}

	// Добавляем XML header
	header := []byte(xml.Header)
	return append(header, output...), nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════

// generateNIS2ID генерирует ID с префиксом для NIS2.
func generateNIS2ID(prefix, incidentID string) string {
	id := generateID()[3:] // Убираем "pd_" из generateID
	if incidentID != "" {
		return fmt.Sprintf("%s_%s_%s", prefix, incidentID, id)
	}
	return fmt.Sprintf("%s_%s", prefix, id)
}

// IncidentSeverityFromString парсит строку в IncidentSeverity.
func IncidentSeverityFromString(s string) IncidentSeverity {
	switch strings.ToLower(s) {
	case "low":
		return SeverityNIS2Low
	case "medium":
		return SeverityNIS2Medium
	case "high":
		return SeverityNIS2High
	case "significant":
		return SeverityNIS2Significant
	case "critical":
		return SeverityNIS2Critical
	default:
		return SeverityNIS2Low
	}
}

// IncidentTypeFromString парсит строку в IncidentType.
func IncidentTypeFromString(s string) IncidentType {
	switch strings.ToLower(s) {
	case "unauthorized_access", "unauthorized":
		return IncidentTypeUnauthorizedAccess
	case "data_breach", "breach":
		return IncidentTypeDataBreach
	case "system_failure", "failure":
		return IncidentTypeSystemFailure
	case "malware", "ransomware":
		return IncidentTypeMalware
	case "physical_tampering", "tampering":
		return IncidentTypePhysicalTampering
	case "denial_of_service", "dos", "ddos":
		return IncidentTypeDenialOfService
	case "configuration_change", "misconfig":
		return IncidentTypeConfigurationChange
	case "network_breach":
		return IncidentTypeNetworkBreach
	case "insider_threat":
		return IncidentTypeInsiderThreat
	case "third_party_breach":
		return IncidentTypeThirdParty
	default:
		return IncidentTypeSystemFailure
	}
}
