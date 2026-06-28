// Package compliance — NIST SP 800-53 / FedRAMP Compliance (P2-CR.4).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-CR.4: NIST SP 800-53 & FedRAMP Compliance
//
// Реализует:
//   - NIST SP 800-53 Rev. 5 control families mapping
//   - FedRAMP Rev. 5 baseline controls (Low, Moderate, High)
//   - Control implementation status tracking
//   - FedRAMP continuous monitoring
//   - POA&M (Plan of Action and Milestones) management
//   - Self-assessment scoring
//
// Compliance:
//   - NIST SP 800-53 Rev. 5 — Security and Privacy Controls
//   - FedRAMP Rev. 5 — Baseline controls (Low/Moderate/High)
//   - FIPS 199 — Security Categorization
//   - NIST SP 800-37 — Risk Management Framework (RMF)
//   - OMB M-21-31 — FedRAMP Authorization Act
//   - ISO 27001 A.5.1 — Information security policies
//   - ISO 27019 PCC.A.5 — ICS security policies
//   - IEC 62443-3-3 — IACS Security
//   - OWASP ASVS V1 (Architecture), V2 (Authentication)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// NIST SP 800-53 Control Families
// ═══════════════════════════════════════════════════════════════════════

// ControlFamily — семейство контролей NIST SP 800-53.
type ControlFamily string

const (
	ControlFamilyAC ControlFamily = "AC" // Access Control
	ControlFamilyAU ControlFamily = "AU" // Audit and Accountability
	ControlFamilyAT ControlFamily = "AT" // Awareness and Training
	ControlFamilyCM ControlFamily = "CM" // Configuration Management
	ControlFamilyCP ControlFamily = "CP" // Contingency Planning
	ControlFamilyIA ControlFamily = "IA" // Identification and Authentication
	ControlFamilyIR ControlFamily = "IR" // Incident Response
	ControlFamilyMA ControlFamily = "MA" // Maintenance
	ControlFamilyMP ControlFamily = "MP" // Media Protection
	ControlFamilyPS ControlFamily = "PS" // Personnel Security
	ControlFamilyPE ControlFamily = "PE" // Physical and Environmental Protection
	ControlFamilyPL ControlFamily = "PL" // Planning
	ControlFamilyPM ControlFamily = "PM" // Program Management
	ControlFamilyRA ControlFamily = "RA" // Risk Assessment
	ControlFamilyCA ControlFamily = "CA" // Security Assessment and Authorization
	ControlFamilySC ControlFamily = "SC" // System and Communications Protection
	ControlFamilySI ControlFamily = "SI" // System and Information Integrity
	ControlFamilySA ControlFamily = "SA" // System and Services Acquisition
	ControlFamilySR ControlFamily = "SR" // Supply Chain Risk Management
)

// ControlFamilyInfo — информация о семействе контролей.
type ControlFamilyInfo struct {
	Family       ControlFamily `json:"family"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	ControlCount int           `json:"control_count"`
}

// ControlFamilies — полный список семейств контролей NIST SP 800-53 Rev. 5.
var ControlFamilies = map[ControlFamily]ControlFamilyInfo{
	ControlFamilyAC: {
		Family: ControlFamilyAC, Name: "Access Control",
		Description:  "Access control policies and procedures",
		ControlCount: 29,
	},
	ControlFamilyAU: {
		Family: ControlFamilyAU, Name: "Audit and Accountability",
		Description:  "Audit record generation, protection, and review",
		ControlCount: 19,
	},
	ControlFamilyAT: {
		Family: ControlFamilyAT, Name: "Awareness and Training",
		Description:  "Security awareness and role-based training",
		ControlCount: 6,
	},
	ControlFamilyCM: {
		Family: ControlFamilyCM, Name: "Configuration Management",
		Description:  "Baseline configurations and change control",
		ControlCount: 16,
	},
	ControlFamilyCP: {
		Family: ControlFamilyCP, Name: "Contingency Planning",
		Description:  "Business continuity and disaster recovery",
		ControlCount: 15,
	},
	ControlFamilyIA: {
		Family: ControlFamilyIA, Name: "Identification and Authentication",
		Description:  "User identification, authentication, and credential management",
		ControlCount: 14,
	},
	ControlFamilyIR: {
		Family: ControlFamilyIR, Name: "Incident Response",
		Description:  "Incident handling, monitoring, and reporting",
		ControlCount: 11,
	},
	ControlFamilyMA: {
		Family: ControlFamilyMA, Name: "Maintenance",
		Description:  "System maintenance and remote maintenance",
		ControlCount: 8,
	},
	ControlFamilyMP: {
		Family: ControlFamilyMP, Name: "Media Protection",
		Description:  "Media access, marking, and sanitization",
		ControlCount: 8,
	},
	ControlFamilyPS: {
		Family: ControlFamilyPS, Name: "Personnel Security",
		Description:  "Personnel screening and termination",
		ControlCount: 4,
	},
	ControlFamilyPE: {
		Family: ControlFamilyPE, Name: "Physical and Environmental Protection",
		Description:  "Physical access controls and environmental safeguards",
		ControlCount: 18,
	},
	ControlFamilyPL: {
		Family: ControlFamilyPL, Name: "Planning",
		Description:  "Security planning and system security plans",
		ControlCount: 6,
	},
	ControlFamilyPM: {
		Family: ControlFamilyPM, Name: "Program Management",
		Description:  "Information security program management",
		ControlCount: 16,
	},
	ControlFamilyRA: {
		Family: ControlFamilyRA, Name: "Risk Assessment",
		Description:  "Risk assessment and vulnerability scanning",
		ControlCount: 8,
	},
	ControlFamilyCA: {
		Family: ControlFamilyCA, Name: "Security Assessment and Authorization",
		Description:  "Security assessments, continuous monitoring, and authorizations",
		ControlCount: 10,
	},
	ControlFamilySC: {
		Family: ControlFamilySC, Name: "System and Communications Protection",
		Description:  "Cryptography, boundary protection, and transmission integrity",
		ControlCount: 42,
	},
	ControlFamilySI: {
		Family: ControlFamilySI, Name: "System and Information Integrity",
		Description:  "Flaw remediation, malicious code protection, and monitoring",
		ControlCount: 25,
	},
	ControlFamilySA: {
		Family: ControlFamilySA, Name: "System and Services Acquisition",
		Description:  "Allocation of resources and system development lifecycle",
		ControlCount: 22,
	},
	ControlFamilySR: {
		Family: ControlFamilySR, Name: "Supply Chain Risk Management",
		Description:  "Supply chain security and vendor assessments",
		ControlCount: 9,
	},
}

// ═══════════════════════════════════════════════════════════════════════
// FedRAMP Baseline Levels
// ═══════════════════════════════════════════════════════════════════════

// FedRAMPSecurityLevel — уровень безопасности FedRAMP.
type FedRAMPSecurityLevel string

const (
	FedRAMPLow      FedRAMPSecurityLevel = "low"      // FIPS 199 Low
	FedRAMPModerate FedRAMPSecurityLevel = "moderate" // FIPS 199 Moderate
	FedRAMPHigh     FedRAMPSecurityLevel = "high"     // FIPS 199 High
)

// FIPS199Category — категория безопасности по FIPS 199.
type FIPS199Category string

const (
	FIPS199Confidentiality FIPS199Category = "confidentiality"
	FIPS199Integrity       FIPS199Category = "integrity"
	FIPS199Availability    FIPS199Category = "availability"
)

// FIPS199ImpactLevel — уровень воздействия по FIPS 199.
type FIPS199ImpactLevel string

const (
	FIPS199LowImpact      FIPS199ImpactLevel = "low"
	FIPS199ModerateImpact FIPS199ImpactLevel = "moderate"
	FIPS199HighImpact     FIPS199ImpactLevel = "high"
)

// FIPS199Categorization — категоризация системы по FIPS 199.
type FIPS199Categorization struct {
	SystemName        string                     `json:"system_name"`
	SystemDescription string                     `json:"system_description"`
	Confidentiality   FIPS199ImpactLevel         `json:"confidentiality"`
	Integrity         FIPS199ImpactLevel         `json:"integrity"`
	Availability      FIPS199ImpactLevel         `json:"availability"`
	OverallLevel      FedRAMPSecurityLevel       `json:"overall_level"`
	Rationale         map[FIPS199Category]string `json:"rationale,omitempty"`
	AssessedBy        string                     `json:"assessed_by"`
	AssessmentDate    time.Time                  `json:"assessment_date"`
}

// FedRAMPBaseline определяет, какие контроли входят в базовый набор.
type FedRAMPBaseline struct {
	Level           FedRAMPSecurityLevel `json:"level"`
	ControlCount    int                  `json:"control_count"`
	ControlFamilies []ControlFamily      `json:"control_families"`
}

// FedRAMPBaselines — базовые наборы контролей по уровням FedRAMP.
var FedRAMPBaselines = map[FedRAMPSecurityLevel]FedRAMPBaseline{
	FedRAMPLow: {
		Level:        FedRAMPLow,
		ControlCount: 125,
		ControlFamilies: []ControlFamily{
			ControlFamilyAC, ControlFamilyAU, ControlFamilyAT,
			ControlFamilyCM, ControlFamilyCP, ControlFamilyIA,
			ControlFamilyIR, ControlFamilyMA, ControlFamilyMP,
			ControlFamilyPE, ControlFamilyPL, ControlFamilyRA,
			ControlFamilyCA, ControlFamilySC, ControlFamilySI,
			ControlFamilySA,
		},
	},
	FedRAMPModerate: {
		Level:        FedRAMPModerate,
		ControlCount: 325,
		ControlFamilies: []ControlFamily{
			ControlFamilyAC, ControlFamilyAU, ControlFamilyAT,
			ControlFamilyCM, ControlFamilyCP, ControlFamilyIA,
			ControlFamilyIR, ControlFamilyMA, ControlFamilyMP,
			ControlFamilyPS, ControlFamilyPE, ControlFamilyPL,
			ControlFamilyRA, ControlFamilyCA, ControlFamilySC,
			ControlFamilySI, ControlFamilySA,
		},
	},
	FedRAMPHigh: {
		Level:        FedRAMPHigh,
		ControlCount: 421,
		ControlFamilies: []ControlFamily{
			ControlFamilyAC, ControlFamilyAU, ControlFamilyAT,
			ControlFamilyCM, ControlFamilyCP, ControlFamilyIA,
			ControlFamilyIR, ControlFamilyMA, ControlFamilyMP,
			ControlFamilyPS, ControlFamilyPE, ControlFamilyPL,
			ControlFamilyPM, ControlFamilyRA, ControlFamilyCA,
			ControlFamilySC, ControlFamilySI, ControlFamilySA,
			ControlFamilySR,
		},
	},
}

// ═══════════════════════════════════════════════════════════════════════
// Control Implementation Status
// ═══════════════════════════════════════════════════════════════════════

// ControlStatus — статус реализации контроля.
type ControlStatus string

const (
	ControlStatusNotImplemented ControlStatus = "not_implemented"
	ControlStatusPlanned        ControlStatus = "planned"
	ControlStatusPartial        ControlStatus = "partial"
	ControlStatusImplemented    ControlStatus = "implemented"
	ControlStatusTested         ControlStatus = "tested"
	ControlStatusCertified      ControlStatus = "certified"
	ControlStatusNotApplicable  ControlStatus = "not_applicable"
)

// NISTControl представляет один контроль NIST SP 800-53.
type NISTControl struct {
	ControlID      string                 `json:"control_id"` // e.g., AC-1, AU-2
	Family         ControlFamily          `json:"family"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Supplemental   string                 `json:"supplemental_guidance,omitempty"`
	Priority       string                 `json:"priority"` // P0, P1, P2
	FedRAMPLevels  []FedRAMPSecurityLevel `json:"fedramp_levels"`
	Status         ControlStatus          `json:"status"`
	Implementation string                 `json:"implementation_details,omitempty"`
	TestProcedure  string                 `json:"test_procedure,omitempty"`
	LastAssessment *time.Time             `json:"last_assessment,omitempty"`
	AssessedBy     string                 `json:"assessed_by,omitempty"`
	POAMItemID     string                 `json:"poam_item_id,omitempty"`
	Notes          string                 `json:"notes,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// POA&M (Plan of Action and Milestones)
// ═══════════════════════════════════════════════════════════════════════

// POAMPriority — приоритет POA&M элемента.
type POAMPriority string

const (
	POAMPriorityCritical POAMPriority = "critical"
	POAMPriorityHigh     POAMPriority = "high"
	POAMPriorityMedium   POAMPriority = "medium"
	POAMPriorityLow      POAMPriority = "low"
)

// POAMStatus — статус POA&M элемента.
type POAMStatus string

const (
	POAMStatusOpen            POAMStatus = "open"
	POAMStatusInProgress      POAMStatus = "in_progress"
	POAMStatusCompleted       POAMStatus = "completed"
	POAMStatusClosed          POAMStatus = "closed"
	POAMStatusVendorDependent POAMStatus = "vendor_dependent"
)

// POAMItem — элемент плана действий и контрольных точек.
type POAMItem struct {
	ID               string          `json:"id"`
	ControlID        string          `json:"control_id"`
	Family           ControlFamily   `json:"family"`
	Weakness         string          `json:"weakness"`
	Description      string          `json:"description"`
	RootCause        string          `json:"root_cause,omitempty"`
	Priority         POAMPriority    `json:"priority"`
	Status           POAMStatus      `json:"status"`
	Remediation      string          `json:"remediation"`
	ResponsibleParty string          `json:"responsible_party"`
	TargetDate       time.Time       `json:"target_date"`
	ClosureDate      *time.Time      `json:"closure_date,omitempty"`
	EstimatedCost    float64         `json:"estimated_cost,omitempty"`
	Milestones       []POAMMilestone `json:"milestones,omitempty"`
	Notes            string          `json:"notes,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// POAMMilestone — контрольная точка POA&M.
type POAMMilestone struct {
	ID             string     `json:"id"`
	POAMItemID     string     `json:"poam_item_id"`
	Description    string     `json:"description"`
	TargetDate     time.Time  `json:"target_date"`
	CompletionDate *time.Time `json:"completion_date,omitempty"`
	Status         POAMStatus `json:"status"`
	Deliverable    string     `json:"deliverable,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Self-Assessment Scoring
// ═══════════════════════════════════════════════════════════════════════

// AssessmentScore — результат самооценки.
type AssessmentScore struct {
	Family         ControlFamily `json:"family"`
	FamilyName     string        `json:"family_name"`
	TotalControls  int           `json:"total_controls"`
	Implemented    int           `json:"implemented"`
	Tested         int           `json:"tested"`
	Partial        int           `json:"partial"`
	NotImplemented int           `json:"not_implemented"`
	NotApplicable  int           `json:"not_applicable"`
	CompliancePct  float64       `json:"compliance_percentage"`
}

// SelfAssessmentReport — полный отчёт самооценки.
type SelfAssessmentReport struct {
	SystemName       string                 `json:"system_name"`
	SystemID         string                 `json:"system_id"`
	FIPSCategory     *FIPS199Categorization `json:"fips_category,omitempty"`
	FedRAMPLevel     FedRAMPSecurityLevel   `json:"fedramp_level"`
	OverallScore     float64                `json:"overall_compliance_percentage"`
	FamilyScores     []AssessmentScore      `json:"family_scores"`
	OpenPOAMs        int                    `json:"open_poams"`
	CriticalFindings int                    `json:"critical_findings"`
	AssessedBy       string                 `json:"assessed_by"`
	AssessmentDate   time.Time              `json:"assessment_date"`
	NextAssessment   time.Time              `json:"next_assessment_date"`
	CreatedAt        time.Time              `json:"created_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// NISTManager — управление NIST SP 800-53 / FedRAMP compliance
// ═══════════════════════════════════════════════════════════════════════

// NISTStore — интерфейс для хранения NIST-данных.
type NISTStore interface {
	// Controls
	SaveControl(ctx interface{}, control *NISTControl) error
	GetControl(ctx interface{}, controlID string) (*NISTControl, error)
	ListControls(ctx interface{}, family ControlFamily) ([]*NISTControl, error)
	ListAllControls(ctx interface{}) ([]*NISTControl, error)
	UpdateControlStatus(ctx interface{}, controlID string, status ControlStatus, implementation string) error

	// POA&M
	SavePOAMItem(ctx interface{}, item *POAMItem) error
	GetPOAMItem(ctx interface{}, id string) (*POAMItem, error)
	ListPOAMItems(ctx interface{}, status POAMStatus) ([]*POAMItem, error)
	ListAllPOAMItems(ctx interface{}) ([]*POAMItem, error)
	UpdatePOAMStatus(ctx interface{}, id string, status POAMStatus) error

	// FIPS 199
	SaveFIPSCategorization(ctx interface{}, cat *FIPS199Categorization) error
	GetFIPSCategorization(ctx interface{}, systemName string) (*FIPS199Categorization, error)
}

// NISTManager — бизнес-логика управления NIST SP 800-53 / FedRAMP compliance.
type NISTManager struct {
	store  NISTStore
	logger *slog.Logger
	mu     sync.RWMutex

	// Baseline controls cache
	baselineMap map[FedRAMPSecurityLevel][]string
}

// NewNISTManager создаёт новый NISTManager.
func NewNISTManager(store NISTStore, logger *slog.Logger) *NISTManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &NISTManager{
		store:  store,
		logger: logger.With("component", "compliance.nist"),
		baselineMap: map[FedRAMPSecurityLevel][]string{
			FedRAMPLow:      generateBaselineControls(FedRAMPLow),
			FedRAMPModerate: generateBaselineControls(FedRAMPModerate),
			FedRAMPHigh:     generateBaselineControls(FedRAMPHigh),
		},
	}
}

// ── FIPS 199 Categorization ────────────────────────────────────────────

// CreateFIPSCategorization создаёт категоризацию системы по FIPS 199.
func (m *NISTManager) CreateFIPSCategorization(systemName, systemDescription string,
	confidentiality, integrity, availability FIPS199ImpactLevel,
	rationale map[FIPS199Category]string, assessedBy string) (*FIPS199Categorization, error) {

	if systemName == "" {
		return nil, fmt.Errorf("nist: system_name is required")
	}

	overall := m.determineOverallLevel(confidentiality, integrity, availability)

	cat := &FIPS199Categorization{
		SystemName:        systemName,
		SystemDescription: systemDescription,
		Confidentiality:   confidentiality,
		Integrity:         integrity,
		Availability:      availability,
		OverallLevel:      overall,
		Rationale:         rationale,
		AssessedBy:        assessedBy,
		AssessmentDate:    time.Now().UTC(),
	}

	if err := m.store.SaveFIPSCategorization(nil, cat); err != nil {
		return nil, fmt.Errorf("nist: save FIPS categorization: %w", err)
	}

	m.logger.Info("FIPS 199 categorization created",
		"system", systemName,
		"overall_level", overall,
	)

	return cat, nil
}

// determineOverallLevel определяет общий уровень системы по FIPS 199.
func (m *NISTManager) determineOverallLevel(confidentiality, integrity, availability FIPS199ImpactLevel) FedRAMPSecurityLevel {
	levels := map[FIPS199ImpactLevel]int{
		FIPS199LowImpact:      1,
		FIPS199ModerateImpact: 2,
		FIPS199HighImpact:     3,
	}

	maxLevel := 0
	for _, l := range []FIPS199ImpactLevel{confidentiality, integrity, availability} {
		if v, ok := levels[l]; ok && v > maxLevel {
			maxLevel = v
		}
	}

	switch maxLevel {
	case 1:
		return FedRAMPLow
	case 2:
		return FedRAMPModerate
	case 3:
		return FedRAMPHigh
	default:
		return FedRAMPLow
	}
}

// ── Control Management ─────────────────────────────────────────────────

// RegisterControl регистрирует контроль NIST SP 800-53.
func (m *NISTManager) RegisterControl(controlID string, family ControlFamily, name, description string,
	priority string, fedrampLevels []FedRAMPSecurityLevel) (*NISTControl, error) {

	if controlID == "" {
		return nil, fmt.Errorf("nist: control_id is required")
	}
	if family == "" {
		return nil, fmt.Errorf("nist: family is required")
	}
	if name == "" {
		return nil, fmt.Errorf("nist: name is required")
	}

	control := &NISTControl{
		ControlID:     controlID,
		Family:        family,
		Name:          name,
		Description:   description,
		Priority:      priority,
		FedRAMPLevels: fedrampLevels,
		Status:        ControlStatusNotImplemented,
	}

	if err := m.store.SaveControl(nil, control); err != nil {
		return nil, fmt.Errorf("nist: save control: %w", err)
	}

	m.logger.Info("NIST control registered",
		"control_id", controlID,
		"family", family,
	)

	return control, nil
}

// UpdateControlImplementation обновляет статус реализации контроля.
func (m *NISTManager) UpdateControlImplementation(controlID string, status ControlStatus, implementation string) error {
	if controlID == "" {
		return fmt.Errorf("nist: control_id is required")
	}
	if status == "" {
		return fmt.Errorf("nist: status is required")
	}

	return m.store.UpdateControlStatus(nil, controlID, status, implementation)
}

// GetControl возвращает контроль по ID.
func (m *NISTManager) GetControl(controlID string) (*NISTControl, error) {
	return m.store.GetControl(nil, controlID)
}

// ListControlsByFamily возвращает все контроли семейства.
func (m *NISTManager) ListControlsByFamily(family ControlFamily) ([]*NISTControl, error) {
	return m.store.ListControls(nil, family)
}

// ListAllControls возвращает все зарегистрированные контроли.
func (m *NISTManager) ListAllControls() ([]*NISTControl, error) {
	return m.store.ListAllControls(nil)
}

// ── POA&M Management ──────────────────────────────────────────────────

// CreatePOAMItem создаёт элемент POA&M.
func (m *NISTManager) CreatePOAMItem(controlID string, family ControlFamily, weakness, description,
	rootCause string, priority POAMPriority, remediation, responsibleParty string,
	targetDate time.Time, estimatedCost float64) (*POAMItem, error) {

	if controlID == "" {
		return nil, fmt.Errorf("nist: control_id is required")
	}
	if weakness == "" {
		return nil, fmt.Errorf("nist: weakness is required")
	}
	if remediation == "" {
		return nil, fmt.Errorf("nist: remediation is required")
	}
	if responsibleParty == "" {
		return nil, fmt.Errorf("nist: responsible_party is required")
	}

	now := time.Now().UTC()
	item := &POAMItem{
		ID:               generateNISTID("poam"),
		ControlID:        controlID,
		Family:           family,
		Weakness:         weakness,
		Description:      description,
		RootCause:        rootCause,
		Priority:         priority,
		Status:           POAMStatusOpen,
		Remediation:      remediation,
		ResponsibleParty: responsibleParty,
		TargetDate:       targetDate,
		EstimatedCost:    estimatedCost,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := m.store.SavePOAMItem(nil, item); err != nil {
		return nil, fmt.Errorf("nist: save POA&M item: %w", err)
	}

	m.logger.Info("POA&M item created",
		"poam_id", item.ID,
		"control_id", controlID,
		"priority", priority,
	)

	return item, nil
}

// UpdatePOAMStatus обновляет статус POA&M элемента.
func (m *NISTManager) UpdatePOAMStatus(poamID string, status POAMStatus) error {
	if poamID == "" {
		return fmt.Errorf("nist: poam_id is required")
	}
	if status == "" {
		return fmt.Errorf("nist: status is required")
	}

	return m.store.UpdatePOAMStatus(nil, poamID, status)
}

// AddPOAMMilestone добавляет контрольную точку к POA&M.
func (m *NISTManager) AddPOAMMilestone(poamItemID, description string, targetDate time.Time, deliverable string) (*POAMMilestone, error) {
	if poamItemID == "" {
		return nil, fmt.Errorf("nist: poam_item_id is required")
	}
	if description == "" {
		return nil, fmt.Errorf("nist: description is required")
	}

	milestone := &POAMMilestone{
		ID:          generateNISTID("ms"),
		POAMItemID:  poamItemID,
		Description: description,
		TargetDate:  targetDate,
		Status:      POAMStatusOpen,
		Deliverable: deliverable,
	}

	m.logger.Info("POA&M milestone added",
		"poam_id", poamItemID,
		"milestone_id", milestone.ID,
		"target_date", targetDate,
	)

	return milestone, nil
}

// ListOpenPOAMItems возвращает все открытые POA&M элементы.
func (m *NISTManager) ListOpenPOAMItems() ([]*POAMItem, error) {
	return m.store.ListPOAMItems(nil, POAMStatusOpen)
}

// ListAllPOAMItems возвращает все POA&M элементы.
func (m *NISTManager) ListAllPOAMItems() ([]*POAMItem, error) {
	return m.store.ListAllPOAMItems(nil)
}

// ── Self-Assessment ───────────────────────────────────────────────────

// RunSelfAssessment выполняет самооценку по NIST SP 800-53.
func (m *NISTManager) RunSelfAssessment(systemName, systemID string, fedrampLevel FedRAMPSecurityLevel,
	assessedBy string) (*SelfAssessmentReport, error) {

	if systemName == "" {
		return nil, fmt.Errorf("nist: system_name is required")
	}

	allControls, err := m.store.ListAllControls(nil)
	if err != nil {
		return nil, fmt.Errorf("nist: list controls for assessment: %w", err)
	}

	if len(allControls) == 0 {
		now := time.Now().UTC()
		report := &SelfAssessmentReport{
			SystemName:     systemName,
			SystemID:       systemID,
			FedRAMPLevel:   fedrampLevel,
			OverallScore:   0,
			FamilyScores:   make([]AssessmentScore, 0),
			OpenPOAMs:      0,
			AssessedBy:     assessedBy,
			AssessmentDate: now,
			NextAssessment: now.AddDate(0, 3, 0),
			CreatedAt:      now,
		}
		return report, nil
	}

	familyMap := make(map[ControlFamily][]*NISTControl)
	for _, c := range allControls {
		familyMap[c.Family] = append(familyMap[c.Family], c)
	}

	familyScores := make([]AssessmentScore, 0, len(familyMap))
	totalCompliance := 0.0
	totalControls := 0
	openPOAMs := 0
	criticalFindings := 0

	for family, controls := range familyMap {
		score := m.scoreFamily(family, controls)
		familyScores = append(familyScores, *score)
		totalControls += score.TotalControls
		familyCompliance := float64(score.Implemented)*100.0 + float64(score.Partial)*50.0
		if score.TotalControls-score.NotApplicable > 0 {
			totalCompliance += familyCompliance
		}
	}

	poamItems, _ := m.store.ListAllPOAMItems(nil)
	for _, item := range poamItems {
		if item.Status == POAMStatusOpen || item.Status == POAMStatusInProgress {
			openPOAMs++
			if item.Priority == POAMPriorityCritical {
				criticalFindings++
			}
		}
	}

	now := time.Now().UTC()
	overallScore := 0.0
	if totalControls > 0 {
		overallScore = totalCompliance / float64(totalControls)
	}

	report := &SelfAssessmentReport{
		SystemName:       systemName,
		SystemID:         systemID,
		FedRAMPLevel:     fedrampLevel,
		OverallScore:     overallScore,
		FamilyScores:     familyScores,
		OpenPOAMs:        openPOAMs,
		CriticalFindings: criticalFindings,
		AssessedBy:       assessedBy,
		AssessmentDate:   now,
		NextAssessment:   now.AddDate(0, 3, 0),
		CreatedAt:        now,
	}

	m.logger.Info("NIST self-assessment completed",
		"system", systemName,
		"overall_score", overallScore,
		"families", len(familyScores),
		"open_poams", openPOAMs,
	)

	return report, nil
}

// scoreFamily оценивает одно семейство контролей.
func (m *NISTManager) scoreFamily(family ControlFamily, controls []*NISTControl) *AssessmentScore {
	info := ControlFamilies[family]
	score := &AssessmentScore{
		Family:        family,
		FamilyName:    info.Name,
		TotalControls: len(controls),
	}

	for _, c := range controls {
		switch c.Status {
		case ControlStatusImplemented, ControlStatusTested, ControlStatusCertified:
			score.Implemented++
		case ControlStatusPartial:
			score.Partial++
		case ControlStatusNotImplemented, ControlStatusPlanned:
			score.NotImplemented++
		case ControlStatusNotApplicable:
			score.NotApplicable++
		}
	}

	applicable := score.TotalControls - score.NotApplicable
	if applicable > 0 {
		score.CompliancePct = float64(score.Implemented+score.Tested) / float64(applicable) * 100.0
	}

	return score
}

// ── Continuous Monitoring ─────────────────────────────────────────────

// ContinuousMonitoringResult — результат непрерывного мониторинга.
type ContinuousMonitoringResult struct {
	SystemName    string               `json:"system_name"`
	SystemID      string               `json:"system_id"`
	FedRAMPLevel  FedRAMPSecurityLevel `json:"fedramp_level"`
	ScanDate      time.Time            `json:"scan_date"`
	ScanType      string               `json:"scan_type"`
	Findings      int                  `json:"findings"`
	CriticalCount int                  `json:"critical_count"`
	HighCount     int                  `json:"high_count"`
	MediumCount   int                  `json:"medium_count"`
	LowCount      int                  `json:"low_count"`
	Compliant     bool                 `json:"compliant"`
	ScanTool      string               `json:"scan_tool,omitempty"`
	RawData       string               `json:"raw_data,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
}

// RecordContinuousMonitoring фиксирует результат мониторинга.
func (m *NISTManager) RecordContinuousMonitoring(systemName, systemID string,
	fedrampLevel FedRAMPSecurityLevel, scanType, scanTool string,
	critical, high, medium, low int) *ContinuousMonitoringResult {

	result := &ContinuousMonitoringResult{
		SystemName:    systemName,
		SystemID:      systemID,
		FedRAMPLevel:  fedrampLevel,
		ScanDate:      time.Now().UTC(),
		ScanType:      scanType,
		Findings:      critical + high + medium + low,
		CriticalCount: critical,
		HighCount:     high,
		MediumCount:   medium,
		LowCount:      low,
		Compliant:     critical == 0 && high == 0,
		ScanTool:      scanTool,
		CreatedAt:     time.Now().UTC(),
	}

	m.logger.Info("continuous monitoring recorded",
		"system", systemName,
		"scan_type", scanType,
		"findings", result.Findings,
		"compliant", result.Compliant,
	)

	return result
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// generateBaselineControls генерирует список control ID для уровня FedRAMP.
func generateBaselineControls(level FedRAMPSecurityLevel) []string {
	baseline := map[FedRAMPSecurityLevel][]string{
		FedRAMPLow: {
			"AC-1", "AC-2", "AC-3", "AC-4", "AC-5", "AC-6", "AC-7",
			"AU-1", "AU-2", "AU-3", "AU-4", "AU-6", "AU-8", "AU-12",
			"AT-1", "AT-2", "AT-3",
			"CM-1", "CM-2", "CM-3", "CM-6", "CM-7", "CM-8",
			"CP-1", "CP-2", "CP-3", "CP-9", "CP-10",
			"IA-1", "IA-2", "IA-3", "IA-4", "IA-5", "IA-6", "IA-7", "IA-8",
			"IR-1", "IR-2", "IR-4", "IR-5", "IR-6", "IR-7",
			"MA-1", "MA-2", "MA-3", "MA-4",
			"MP-1", "MP-2", "MP-3",
			"PE-1", "PE-2", "PE-3", "PE-5", "PE-6", "PE-8",
			"PL-1", "PL-2", "PL-4", "PL-8",
			"RA-1", "RA-2", "RA-3", "RA-5",
			"CA-1", "CA-2", "CA-3", "CA-5", "CA-6", "CA-7", "CA-9",
			"SC-1", "SC-2", "SC-5", "SC-7", "SC-8", "SC-12", "SC-13",
			"SI-1", "SI-2", "SI-3", "SI-4", "SI-5", "SI-7", "SI-10", "SI-12",
			"SA-1", "SA-2", "SA-3",
		},
		FedRAMPModerate: {
			"AC-1", "AC-2", "AC-3", "AC-4", "AC-5", "AC-6", "AC-7", "AC-8",
			"AC-10", "AC-11", "AC-14", "AC-17", "AC-18", "AC-19", "AC-20", "AC-22",
			"AU-1", "AU-2", "AU-3", "AU-4", "AU-5", "AU-6", "AU-7", "AU-8",
			"AU-9", "AU-11", "AU-12",
			"AT-1", "AT-2", "AT-3", "AT-4",
			"CM-1", "CM-2", "CM-3", "CM-4", "CM-5", "CM-6", "CM-7", "CM-8", "CM-9",
			"CP-1", "CP-2", "CP-3", "CP-4", "CP-6", "CP-7", "CP-8", "CP-9", "CP-10",
			"IA-1", "IA-2", "IA-3", "IA-4", "IA-5", "IA-6", "IA-7", "IA-8",
			"IR-1", "IR-2", "IR-3", "IR-4", "IR-5", "IR-6", "IR-7", "IR-8",
			"MA-1", "MA-2", "MA-3", "MA-4", "MA-5", "MA-6",
			"MP-1", "MP-2", "MP-3", "MP-4", "MP-5", "MP-6", "MP-7", "MP-8",
			"PS-1", "PS-2", "PS-3", "PS-4",
			"PE-1", "PE-2", "PE-3", "PE-5", "PE-6", "PE-8", "PE-9", "PE-10",
			"PE-11", "PE-12", "PE-13", "PE-15",
			"PL-1", "PL-2", "PL-4", "PL-8",
			"RA-1", "RA-2", "RA-3", "RA-5",
			"CA-1", "CA-2", "CA-3", "CA-5", "CA-6", "CA-7", "CA-8", "CA-9",
			"SC-1", "SC-2", "SC-4", "SC-5", "SC-7", "SC-8", "SC-10", "SC-12",
			"SC-13", "SC-15", "SC-18", "SC-20", "SC-21", "SC-22", "SC-23", "SC-28",
			"SI-1", "SI-2", "SI-3", "SI-4", "SI-5", "SI-6", "SI-7", "SI-8",
			"SI-10", "SI-11", "SI-12", "SI-16",
			"SA-1", "SA-2", "SA-3", "SA-4", "SA-5", "SA-8", "SA-9", "SA-10",
		},
		FedRAMPHigh: {
			"AC-1", "AC-2", "AC-3", "AC-4", "AC-5", "AC-6", "AC-7", "AC-8",
			"AC-10", "AC-11", "AC-14", "AC-16", "AC-17", "AC-18", "AC-19", "AC-20",
			"AC-21", "AC-22", "AC-24", "AC-25",
			"AU-1", "AU-2", "AU-3", "AU-4", "AU-5", "AU-6", "AU-7", "AU-8",
			"AU-9", "AU-10", "AU-11", "AU-12", "AU-13", "AU-14", "AU-16",
			"AT-1", "AT-2", "AT-3", "AT-4",
			"CM-1", "CM-2", "CM-3", "CM-4", "CM-5", "CM-6", "CM-7", "CM-8", "CM-9",
			"CM-10", "CM-11", "CM-12",
			"CP-1", "CP-2", "CP-3", "CP-4", "CP-6", "CP-7", "CP-8", "CP-9", "CP-10",
			"IA-1", "IA-2", "IA-3", "IA-4", "IA-5", "IA-6", "IA-7", "IA-8",
			"IA-9", "IA-11", "IA-12",
			"IR-1", "IR-2", "IR-3", "IR-4", "IR-5", "IR-6", "IR-7", "IR-8", "IR-9", "IR-10",
			"MA-1", "MA-2", "MA-3", "MA-4", "MA-5", "MA-6",
			"MP-1", "MP-2", "MP-3", "MP-4", "MP-5", "MP-6", "MP-7", "MP-8",
			"PS-1", "PS-2", "PS-3", "PS-4",
			"PE-1", "PE-2", "PE-3", "PE-5", "PE-6", "PE-8", "PE-9", "PE-10",
			"PE-11", "PE-12", "PE-13", "PE-14", "PE-15", "PE-16", "PE-17", "PE-18",
			"PL-1", "PL-2", "PL-4", "PL-8", "PL-9",
			"PM-1", "PM-2", "PM-3", "PM-4", "PM-5", "PM-6", "PM-7", "PM-8", "PM-9",
			"PM-10", "PM-11", "PM-12", "PM-13", "PM-14", "PM-15", "PM-16",
			"RA-1", "RA-2", "RA-3", "RA-5", "RA-6", "RA-7", "RA-8", "RA-9",
			"CA-1", "CA-2", "CA-3", "CA-5", "CA-6", "CA-7", "CA-8", "CA-9",
			"SC-1", "SC-2", "SC-3", "SC-4", "SC-5", "SC-6", "SC-7", "SC-8", "SC-10",
			"SC-12", "SC-13", "SC-15", "SC-17", "SC-18", "SC-20", "SC-21", "SC-22",
			"SC-23", "SC-24", "SC-28", "SC-29", "SC-30", "SC-32", "SC-34", "SC-39",
			"SI-1", "SI-2", "SI-3", "SI-4", "SI-5", "SI-6", "SI-7", "SI-8",
			"SI-10", "SI-11", "SI-12", "SI-13", "SI-14", "SI-15", "SI-16", "SI-17",
			"SA-1", "SA-2", "SA-3", "SA-4", "SA-5", "SA-8", "SA-9", "SA-10",
			"SA-11", "SA-15", "SA-16", "SA-17", "SA-19", "SA-21", "SA-22",
			"SR-1", "SR-2", "SR-3", "SR-4", "SR-5", "SR-6", "SR-7", "SR-8", "SR-9",
		},
	}

	if controls, ok := baseline[level]; ok {
		return controls
	}
	return []string{}
}

// generateNISTID генерирует ID с префиксом.
func generateNISTID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, generateID()[3:])
}
