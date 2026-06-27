// Package compliance — GDPR-Specific Features (P2-EU.1).
//
// Реализует:
//   - Right to be forgotten (data erasure workflow) — Art. 17
//   - Data portability exports — Art. 20
//   - Consent audit trail — Art. 7
//   - DPIA report generator — Art. 35
//   - Schrems II compliant data transfers (SCCs) — Art. 44-49
//
// Compliance:
//   - GDPR Art. 7 (Consent), Art. 17 (Right to erasure), Art. 20 (Portability)
//   - GDPR Art. 32 (Security), Art. 35 (DPIA), Art. 44-49 (Transfers)
//   - ISO 27001 A.8.2 (Classification), A.12.4 (Audit)
//   - ISO 27019 PCC.A.8, PCC.A.12
//   - СТБ 34.101.27 п. 6.2 (Политики безопасности)
//   - OWASP ASVS V8 (Data Protection), V7 (Log content)
package compliance

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// GDPR: Right to be Forgotten — Art. 17
// ═══════════════════════════════════════════════════════════════════════

// ErasureRequestStatus — статус запроса на удаление.
type ErasureRequestStatus string

const (
	ErasureStatusNew        ErasureRequestStatus = "new"
	ErasureStatusVerified   ErasureRequestStatus = "verified"    // Личность подтверждена
	ErasureStatusInProgress ErasureRequestStatus = "in_progress" // Удаление выполняется
	ErasureStatusCompleted  ErasureRequestStatus = "completed"   // Удаление завершено
	ErasureStatusRejected   ErasureRequestStatus = "rejected"    // Отклонено (Art. 17(3))
	ErasureStatusExempted   ErasureRequestStatus = "exempted"    // Исключение (legal hold)
)

// ErasureScope — область удаления.
type ErasureScope string

const (
	ErasureScopeAll         ErasureScope = "all"         // Все данные
	ErasureScopeVideo       ErasureScope = "video"       // Только видео
	ErasureScopeAnalytics   ErasureScope = "analytics"   // Только аналитика
	ErasureScopeCredentials ErasureScope = "credentials" // Только учётные данные
	ErasureScopeSpecific    ErasureScope = "specific"    // Конкретные системы
)

// ErasureRequest представляет запрос на удаление данных (Right to be Forgotten).
type ErasureRequest struct {
	ID              string               `json:"id"`
	SubjectID       string               `json:"subject_id"`
	SubjectName     string               `json:"subject_name"`
	SubjectEmail    string               `json:"subject_email"`
	Scope           ErasureScope         `json:"scope"`
	SpecificSystems []string             `json:"specific_systems,omitempty"`
	Status          ErasureRequestStatus `json:"status"`
	RejectionReason string               `json:"rejection_reason,omitempty"`
	LegalBasis      string               `json:"legal_basis,omitempty"`      // Art. 17(3) exception
	ExemptedSystems []string             `json:"exempted_systems,omitempty"` // Системы с legal hold
	RequestedAt     time.Time            `json:"requested_at"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// GDPR: Data Portability — Art. 20
// ═══════════════════════════════════════════════════════════════════════

// PortabilityFormat — формат экспорта данных.
type PortabilityFormat string

const (
	PortabilityJSON PortabilityFormat = "json" // Machine-readable JSON
	PortabilityCSV  PortabilityFormat = "csv"  // CSV with headers
)

// PortabilityExport представляет экспорт данных для портабельности.
type PortabilityExport struct {
	ID             string            `json:"id"`
	SubjectID      string            `json:"subject_id"`
	SubjectName    string            `json:"subject_name"`
	SubjectEmail   string            `json:"subject_email"`
	Format         PortabilityFormat `json:"format"`
	DataCategories []DataCategory    `json:"data_categories"`
	DataPayload    string            `json:"data_payload,omitempty"` // JSON/CSV строка
	FileSizeBytes  int64             `json:"file_size_bytes,omitempty"`
	ExpiresAt      time.Time         `json:"expires_at"` // 30 дней
	DownloadedAt   *time.Time        `json:"downloaded_at,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Expires        bool              `json:"expires"` // Флаг истечения
}

// ═══════════════════════════════════════════════════════════════════════
// GDPR: Consent Audit Trail — Art. 7
// ═══════════════════════════════════════════════════════════════════════

// ConsentAuditEntry — запись аудита согласия.
type ConsentAuditEntry struct {
	ID          string         `json:"id"`
	SubjectID   string         `json:"subject_id"`
	Action      string         `json:"action"` // granted, revoked, expired, updated
	Purpose     ConsentPurpose `json:"purpose"`
	OldStatus   ConsentStatus  `json:"old_status"`
	NewStatus   ConsentStatus  `json:"new_status"`
	ChangedBy   string         `json:"changed_by"` // user_id или system
	SourceIP    string         `json:"source_ip,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
	Timestamp   time.Time      `json:"timestamp"`
	ConsentID   string         `json:"consent_id"`
	DocumentRef string         `json:"document_ref,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// GDPR: DPIA Report — Art. 35
// ═══════════════════════════════════════════════════════════════════════

// RiskLevel — уровень риска для DPIA.
type DPIARiskLevel string

const (
	DPIARiskLow      DPIARiskLevel = "low"
	DPIARiskMedium   DPIARiskLevel = "medium"
	DPIARiskHigh     DPIARiskLevel = "high"
	DPIARiskCritical DPIARiskLevel = "critical"
)

// DPIAReport — отчёт об оценке воздействия на защиту данных.
type DPIAReport struct {
	ID                     string         `json:"id"`
	SystemName             string         `json:"system_name"`
	SystemDescription      string         `json:"system_description"`
	DataController         string         `json:"data_controller"`
	DataProcessor          string         `json:"data_processor,omitempty"`
	DPO                    string         `json:"dpo,omitempty"` // Data Protection Officer
	ProcessingPurposes     []string       `json:"processing_purposes"`
	DataCategories         []DataCategory `json:"data_categories"`
	DataSubjects           []string       `json:"data_subjects"` // employee, customer, visitor
	LegalBasis             string         `json:"legal_basis"`
	DataRetentionPeriod    string         `json:"data_retention_period"`
	TechnicalMeasures      []string       `json:"technical_measures"`
	OrganizationalMeasures []string       `json:"organizational_measures"`
	ThirdPartyProcessors   []string       `json:"third_party_processors"`
	CrossBorderTransfers   []string       `json:"cross_border_transfers"`
	RiskLevel              DPIARiskLevel  `json:"risk_level"`
	RiskAssessment         string         `json:"risk_assessment"`
	MitigationMeasures     []string       `json:"mitigation_measures"`
	ResidualRiskLevel      DPIARiskLevel  `json:"residual_risk_level"`
	DPIARequired           bool           `json:"dpia_required"`
	DPOReviewed            bool           `json:"dpo_reviewed"`
	DPOReviewDate          *time.Time     `json:"dpo_review_date,omitempty"`
	ApprovedBy             string         `json:"approved_by,omitempty"`
	ApprovedAt             *time.Time     `json:"approved_at,omitempty"`
	ReviewDate             time.Time      `json:"review_date"` // Next review
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// GDPR: Schrems II — SCCs (Art. 44-49)
// ═══════════════════════════════════════════════════════════════════════

// SCCStatus — статус Standard Contractual Clauses.
type SCCStatus string

const (
	SCCStatusActive      SCCStatus = "active"
	SCCStatusExpired     SCCStatus = "expired"
	SCCStatusRevoked     SCCStatus = "revoked"
	SCCStatusNegotiating SCCStatus = "negotiating"
)

// TransferMechanism — механизм трансграничной передачи.
type TransferMechanism string

const (
	TransferSCC        TransferMechanism = "scc"        // Standard Contractual Clauses
	TransferBCR        TransferMechanism = "bcr"        // Binding Corporate Rules
	TransferAdequacy   TransferMechanism = "adequacy"   // Adequacy decision
	TransferDerogation TransferMechanism = "derogation" // Derogation Art. 49
)

// DataTransferAgreement — соглашение о трансграничной передаче данных.
type DataTransferAgreement struct {
	ID                    string            `json:"id"`
	TransferFrom          string            `json:"transfer_from"` // EU
	TransferTo            string            `json:"transfer_to"`   // Third country
	Mechanism             TransferMechanism `json:"mechanism"`
	SCCStatus             SCCStatus         `json:"scc_status"`
	ControllerName        string            `json:"controller_name"`
	ProcessorName         string            `json:"processor_name,omitempty"`
	DataCategories        []DataCategory    `json:"data_categories"`
	TransferBasis         string            `json:"transfer_basis"` // Legal basis
	TIACompleted          bool              `json:"tia_completed"`  // Transfer Impact Assessment
	TIADate               *time.Time        `json:"tia_date,omitempty"`
	SupplementaryMeasures []string          `json:"supplementary_measures,omitempty"`
	EffectiveDate         time.Time         `json:"effective_date"`
	ExpiryDate            *time.Time        `json:"expiry_date,omitempty"`
	SignedBy              string            `json:"signed_by"`
	DocumentRef           string            `json:"document_ref,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}

// ═══════════════════════════════════════════════════════════════════════
// GDPRManager — бизнес-логика GDPR compliance
// ═══════════════════════════════════════════════════════════════════════

// GDPRStore — интерфейс для хранения GDPR-данных.
type GDPRStore interface {
	// Erasure
	SaveErasureRequest(ctx interface{}, req *ErasureRequest) error
	GetErasureRequest(ctx interface{}, id string) (*ErasureRequest, error)
	ListErasureRequests(ctx interface{}, subjectID string) ([]*ErasureRequest, error)
	UpdateErasureStatus(ctx interface{}, id string, status ErasureRequestStatus, rejectionReason string) error

	// Portability
	SavePortabilityExport(ctx interface{}, exp *PortabilityExport) error
	GetPortabilityExport(ctx interface{}, id string) (*PortabilityExport, error)
	ListPortabilityExports(ctx interface{}, subjectID string) ([]*PortabilityExport, error)

	// Consent Audit
	SaveConsentAuditEntry(ctx interface{}, entry *ConsentAuditEntry) error
	ListConsentAuditEntries(ctx interface{}, subjectID string) ([]*ConsentAuditEntry, error)

	// DPIA
	SaveDPIAReport(ctx interface{}, report *DPIAReport) error
	GetDPIAReport(ctx interface{}, id string) (*DPIAReport, error)
	ListDPIAReports(ctx interface{}) ([]*DPIAReport, error)

	// Data Transfers
	SaveTransferAgreement(ctx interface{}, agreement *DataTransferAgreement) error
	GetTransferAgreement(ctx interface{}, id string) (*DataTransferAgreement, error)
	ListTransferAgreements(ctx interface{}) ([]*DataTransferAgreement, error)
}

// GDPRManager — бизнес-логика управления GDPR compliance.
type GDPRManager struct {
	store  GDPRStore
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewGDPRManager создаёт новый GDPRManager.
func NewGDPRManager(store GDPRStore, logger *slog.Logger) *GDPRManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &GDPRManager{
		store:  store,
		logger: logger.With("component", "compliance.gdpr"),
	}
}

// ── Right to be Forgotten (Art. 17) ───────────────────────────────

// RequestErasure создаёт запрос на удаление данных.
func (m *GDPRManager) RequestErasure(subjectID, subjectName, subjectEmail string, scope ErasureScope, specificSystems []string) (*ErasureRequest, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("gdpr: subject_id is required")
	}
	if scope == "" {
		return nil, fmt.Errorf("gdpr: scope is required")
	}

	now := time.Now().UTC()
	req := &ErasureRequest{
		ID:              generateGDPRID("er"),
		SubjectID:       subjectID,
		SubjectName:     subjectName,
		SubjectEmail:    subjectEmail,
		Scope:           scope,
		SpecificSystems: specificSystems,
		Status:          ErasureStatusNew,
		RequestedAt:     now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := m.store.SaveErasureRequest(nil, req); err != nil {
		return nil, fmt.Errorf("gdpr: save erasure request: %w", err)
	}

	m.logger.Info("right to be forgotten requested",
		"erasure_id", req.ID,
		"subject_id", subjectID,
		"scope", scope,
	)

	return req, nil
}

// CompleteErasure завершает удаление данных.
func (m *GDPRManager) CompleteErasure(erasureID string) error {
	if erasureID == "" {
		return fmt.Errorf("gdpr: erasure_id is required")
	}

	req, err := m.store.GetErasureRequest(nil, erasureID)
	if err != nil {
		return fmt.Errorf("gdpr: get erasure request: %w", err)
	}
	if req == nil {
		return fmt.Errorf("gdpr: erasure request not found: %s", erasureID)
	}

	if err := m.store.UpdateErasureStatus(nil, erasureID, ErasureStatusCompleted, ""); err != nil {
		return fmt.Errorf("gdpr: complete erasure: %w", err)
	}

	m.logger.Info("right to be forgotten completed",
		"erasure_id", erasureID,
		"subject_id", req.SubjectID,
	)

	return nil
}

// RejectErasure отклоняет запрос на удаление (Art. 17(3) exceptions).
func (m *GDPRManager) RejectErasure(erasureID, reason string) error {
	if erasureID == "" {
		return fmt.Errorf("gdpr: erasure_id is required")
	}
	if reason == "" {
		return fmt.Errorf("gdpr: rejection reason is required")
	}

	return m.store.UpdateErasureStatus(nil, erasureID, ErasureStatusRejected, reason)
}

// GetErasureRequest возвращает запрос на удаление.
func (m *GDPRManager) GetErasureRequest(id string) (*ErasureRequest, error) {
	return m.store.GetErasureRequest(nil, id)
}

// ListSubjectErasureRequests возвращает все запросы на удаление субъекта.
func (m *GDPRManager) ListSubjectErasureRequests(subjectID string) ([]*ErasureRequest, error) {
	return m.store.ListErasureRequests(nil, subjectID)
}

// ── Data Portability (Art. 20) ────────────────────────────────────

// CreatePortabilityExport создаёт экспорт данных для портабельности.
func (m *GDPRManager) CreatePortabilityExport(subjectID, subjectName, subjectEmail string, format PortabilityFormat, categories []DataCategory, payload string) (*PortabilityExport, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("gdpr: subject_id is required")
	}
	if format == "" {
		return nil, fmt.Errorf("gdpr: format is required")
	}

	now := time.Now().UTC()
	exp := &PortabilityExport{
		ID:             generateGDPRID("pe"),
		SubjectID:      subjectID,
		SubjectName:    subjectName,
		SubjectEmail:   subjectEmail,
		Format:         format,
		DataCategories: categories,
		DataPayload:    payload,
		FileSizeBytes:  int64(len(payload)),
		ExpiresAt:      now.AddDate(0, 0, 30), // 30 дней по GDPR
		CreatedAt:      now,
		Expires:        true,
	}

	if err := m.store.SavePortabilityExport(nil, exp); err != nil {
		return nil, fmt.Errorf("gdpr: save portability export: %w", err)
	}

	m.logger.Info("portability export created",
		"export_id", exp.ID,
		"subject_id", subjectID,
		"format", format,
		"size", exp.FileSizeBytes,
	)

	return exp, nil
}

// GetPortabilityExport возвращает экспорт данных.
func (m *GDPRManager) GetPortabilityExport(id string) (*PortabilityExport, error) {
	return m.store.GetPortabilityExport(nil, id)
}

// ListSubjectPortabilityExports возвращает все экспорты субъекта.
func (m *GDPRManager) ListSubjectPortabilityExports(subjectID string) ([]*PortabilityExport, error) {
	return m.store.ListPortabilityExports(nil, subjectID)
}

// ── Consent Audit Trail (Art. 7) ──────────────────────────────────

// RecordConsentAudit фиксирует изменение согласия в аудите.
func (m *GDPRManager) RecordConsentAudit(subjectID string, action string, purpose ConsentPurpose,
	oldStatus, newStatus ConsentStatus, changedBy, sourceIP, userAgent, consentID string) *ConsentAuditEntry {

	entry := &ConsentAuditEntry{
		ID:        generateGDPRID("ca"),
		SubjectID: subjectID,
		Action:    action,
		Purpose:   purpose,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		ChangedBy: changedBy,
		SourceIP:  sourceIP,
		UserAgent: userAgent,
		Timestamp: time.Now().UTC(),
		ConsentID: consentID,
	}

	// Non-blocking save — логируем ошибку, не прерываем операцию
	if err := m.store.SaveConsentAuditEntry(nil, entry); err != nil {
		m.logger.Error("failed to save consent audit entry", "error", err)
	}

	return entry
}

// GetConsentAuditTrail возвращает историю изменений согласий субъекта.
func (m *GDPRManager) GetConsentAuditTrail(subjectID string) ([]*ConsentAuditEntry, error) {
	return m.store.ListConsentAuditEntries(nil, subjectID)
}

// ── DPIA Report Generator (Art. 35) ───────────────────────────────

// GenerateDPIAReport создаёт отчёт DPIA.
func (m *GDPRManager) GenerateDPIAReport(systemName, systemDescription, dataController, dataProcessor, dpo string,
	purposes []string, categories []DataCategory, subjects []string, legalBasis, retentionPeriod string,
	techMeasures, orgMeasures []string, thirdParties, crossBorderTransfers []string) (*DPIAReport, error) {

	if systemName == "" {
		return nil, fmt.Errorf("gdpr: system_name is required")
	}
	if dataController == "" {
		return nil, fmt.Errorf("gdpr: data_controller is required")
	}

	now := time.Now().UTC()
	riskLevel := m.assessRiskLevel(categories, purposes, crossBorderTransfers)

	report := &DPIAReport{
		ID:                     generateGDPRID("dpia"),
		SystemName:             systemName,
		SystemDescription:      systemDescription,
		DataController:         dataController,
		DataProcessor:          dataProcessor,
		DPO:                    dpo,
		ProcessingPurposes:     purposes,
		DataCategories:         categories,
		DataSubjects:           subjects,
		LegalBasis:             legalBasis,
		DataRetentionPeriod:    retentionPeriod,
		TechnicalMeasures:      techMeasures,
		OrganizationalMeasures: orgMeasures,
		ThirdPartyProcessors:   thirdParties,
		CrossBorderTransfers:   crossBorderTransfers,
		RiskLevel:              riskLevel,
		RiskAssessment:         m.generateRiskAssessment(riskLevel, categories, purposes),
		MitigationMeasures:     m.generateMitigationMeasures(riskLevel, categories),
		ResidualRiskLevel:      m.calculateResidualRisk(riskLevel),
		DPIARequired:           riskLevel >= DPIARiskHigh,
		DPOReviewed:            false,
		ReviewDate:             now.AddDate(0, 6, 0), // Review every 6 months
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := m.store.SaveDPIAReport(nil, report); err != nil {
		return nil, fmt.Errorf("gdpr: save DPIA report: %w", err)
	}

	m.logger.Info("DPIA report generated",
		"dpia_id", report.ID,
		"system", systemName,
		"risk_level", riskLevel,
	)

	return report, nil
}

// GetDPIAReport возвращает DPIA отчёт.
func (m *GDPRManager) GetDPIAReport(id string) (*DPIAReport, error) {
	return m.store.GetDPIAReport(nil, id)
}

// ListDPIAReports возвращает все DPIA отчёты.
func (m *GDPRManager) ListDPIAReports() ([]*DPIAReport, error) {
	return m.store.ListDPIAReports(nil)
}

// assessRiskLevel определяет уровень риска на основе категорий данных и целей.
func (m *GDPRManager) assessRiskLevel(categories []DataCategory, purposes []string, crossBorderTransfers []string) DPIARiskLevel {
	hasBiometric := false
	hasLocation := false
	for _, c := range categories {
		if c == DataCategoryBiometric {
			hasBiometric = true
		}
		if c == DataCategoryLocation {
			hasLocation = true
		}
	}

	if hasBiometric && len(crossBorderTransfers) > 0 {
		return DPIARiskCritical
	}
	if hasBiometric || (hasLocation && len(crossBorderTransfers) > 0) {
		return DPIARiskHigh
	}
	if hasLocation || len(crossBorderTransfers) > 0 {
		return DPIARiskMedium
	}
	return DPIARiskLow
}

// generateRiskAssessment генерирует описание риска.
func (m *GDPRManager) generateRiskAssessment(level DPIARiskLevel, categories []DataCategory, purposes []string) string {
	switch level {
	case DPIARiskCritical:
		return "Критический риск: обработка биометрических данных с трансграничной передачей. " +
			"Требуется обязательное проведение DPIA и предварительное согласование с надзорным органом."
	case DPIARiskHigh:
		return "Высокий риск: обработка биометрических или геолокационных данных. " +
			"Требуется DPIA, оценка соразмерности и меры минимизации риска."
	case DPIARiskMedium:
		return "Средний риск: обработка геолокационных данных или трансграничная передача. " +
			"Рекомендуется DPIA и дополнительные меры защиты."
	default:
		return "Низкий риск: обработка стандартных категорий данных без трансграничной передачи. " +
			"DPIA не требуется, но рекомендуется документирование."
	}
}

// generateMitigationMeasures генерирует меры минимизации риска.
func (m *GDPRManager) generateMitigationMeasures(level DPIARiskLevel, categories []DataCategory) []string {
	measures := []string{
		"Data encryption at rest (AES-256-GCM)",
		"Data encryption in transit (TLS 1.3)",
		"Access control (RBAC with least privilege)",
		"Audit logging (ISO 27001 A.12.4)",
	}

	if level >= DPIARiskHigh {
		measures = append(measures,
			"Pseudonymization/anonymization",
			"Data minimization review",
			"Regular security assessments",
		)
	}

	if level == DPIARiskCritical {
		measures = append(measures,
			"Prior consultation with supervisory authority",
			"Independent DPIA review by DPO",
			"Additional technical measures (PETs)",
			"Data Protection by Design review",
		)
	}

	hasBiometric := false
	for _, c := range categories {
		if c == DataCategoryBiometric {
			hasBiometric = true
			break
		}
	}
	if hasBiometric {
		measures = append(measures, "Biometric data-specific protection (liveness detection, anti-spoofing)")
	}

	return measures
}

// calculateResidualRisk вычисляет остаточный риск после мер минимизации.
func (m *GDPRManager) calculateResidualRisk(initial DPIARiskLevel) DPIARiskLevel {
	switch initial {
	case DPIARiskCritical:
		return DPIARiskHigh // После мер — снижаем до высокого
	case DPIARiskHigh:
		return DPIARiskMedium
	case DPIARiskMedium:
		return DPIARiskLow
	default:
		return DPIARiskLow
	}
}

// ── Schrems II: Data Transfers (Art. 44-49) ───────────────────────

// CreateTransferAgreement создаёт соглашение о трансграничной передаче.
func (m *GDPRManager) CreateTransferAgreement(transferFrom, transferTo string, mechanism TransferMechanism,
	controllerName, processorName, signedBy string, categories []DataCategory,
	effectiveDate time.Time, supplementaryMeasures []string) (*DataTransferAgreement, error) {

	if transferFrom == "" || transferTo == "" {
		return nil, fmt.Errorf("gdpr: transfer_from and transfer_to are required")
	}
	if mechanism == "" {
		return nil, fmt.Errorf("gdpr: mechanism is required")
	}
	if controllerName == "" {
		return nil, fmt.Errorf("gdpr: controller_name is required")
	}

	now := time.Now().UTC()
	agreement := &DataTransferAgreement{
		ID:                    generateGDPRID("scc"),
		TransferFrom:          transferFrom,
		TransferTo:            transferTo,
		Mechanism:             mechanism,
		SCCStatus:             SCCStatusNegotiating,
		ControllerName:        controllerName,
		ProcessorName:         processorName,
		DataCategories:        categories,
		TransferBasis:         fmt.Sprintf("SCCs (%s) per GDPR Art. 46", mechanism),
		TIACompleted:          false,
		SupplementaryMeasures: supplementaryMeasures,
		EffectiveDate:         effectiveDate,
		SignedBy:              signedBy,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	if err := m.store.SaveTransferAgreement(nil, agreement); err != nil {
		return nil, fmt.Errorf("gdpr: save transfer agreement: %w", err)
	}

	m.logger.Info("data transfer agreement created",
		"agreement_id", agreement.ID,
		"from", transferFrom,
		"to", transferTo,
		"mechanism", mechanism,
	)

	return agreement, nil
}

// CompleteTIA отмечает Transfer Impact Assessment как выполненный.
func (m *GDPRManager) CompleteTIA(agreementID string) error {
	agreement, err := m.store.GetTransferAgreement(nil, agreementID)
	if err != nil {
		return fmt.Errorf("gdpr: get transfer agreement: %w", err)
	}
	if agreement == nil {
		return fmt.Errorf("gdpr: transfer agreement not found: %s", agreementID)
	}

	agreement.TIACompleted = true
	now := time.Now().UTC()
	agreement.TIADate = &now
	agreement.SCCStatus = SCCStatusActive
	agreement.UpdatedAt = now

	// В реальной системе здесь было бы сохранение через store
	// Для MVP используем SaveTransferAgreement
	if err := m.store.SaveTransferAgreement(nil, agreement); err != nil {
		return fmt.Errorf("gdpr: update transfer agreement: %w", err)
	}

	m.logger.Info("TIA completed for transfer agreement",
		"agreement_id", agreementID,
	)

	return nil
}

// GetTransferAgreement возвращает соглашение о передаче.
func (m *GDPRManager) GetTransferAgreement(id string) (*DataTransferAgreement, error) {
	return m.store.GetTransferAgreement(nil, id)
}

// ListTransferAgreements возвращает все соглашения о передаче.
func (m *GDPRManager) ListTransferAgreements() ([]*DataTransferAgreement, error) {
	return m.store.ListTransferAgreements(nil)
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// generateGDPRID генерирует ID с префиксом.
func generateGDPRID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, generateID()[3:]) // Убираем "pd_" из generateID
}
