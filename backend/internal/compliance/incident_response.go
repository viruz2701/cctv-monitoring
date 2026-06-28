// Package compliance — Multi-Tier Incident Response Engine (P0-N3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-N3: Multi-Tier Incident Response Engine
//
// Проблема: Разные регионы требуют разные сроки reporting:
//   - India (CERT-In): 6h
//   - EU (DORA/NIS2): 4h
//   - Singapore (CSA): 2h
//   - Belarus (ОАЦ): 24h
//   - Russia (ФСТЭК): 24h
//
// Решение:
//   - IncidentClassificationEngine — multi-region классификация
//   - Multi-tier routing per region с SLA таймерами
//   - Automated report generation per regulator format
//   - Legal hold + evidence preservation
//   - Escalation matrix per region
//
// Compliance:
//   - EU DORA (Digital Operational Resilience Act)
//   - EU NIS2 Directive Art. 23
//   - India CERT-In Directions (6h reporting)
//   - Singapore CSA (2h reporting)
//   - ISO 27001 A.16.1 (Incident Management)
//   - IEC 62443-3-3 SR 7.1 (Incident Response)
//   - СТБ 34.101.27 п. 7.2 (Реагирование на инциденты КИИ)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// Regulatory Frameworks
// ═══════════════════════════════════════════════════════════════════════════

// RegulatoryFramework идентифицирует регуляторную рамку.
type RegulatoryFramework string

const (
	FrameworkNIS2   RegulatoryFramework = "NIS2"    // EU NIS2 Directive
	FrameworkDORA   RegulatoryFramework = "DORA"    // EU Digital Operational Resilience Act
	FrameworkCERTIn RegulatoryFramework = "CERT-In" // India CERT-In
	FrameworkCSA    RegulatoryFramework = "CSA"     // Singapore CSA
	FrameworkOAC    RegulatoryFramework = "ОАЦ"     // Belarus ОАЦ
	FrameworkFSTEK  RegulatoryFramework = "ФСТЭК"   // Russia ФСТЭК
	FrameworkCCRA   RegulatoryFramework = "CCRA"    // EU Cyber Resilience Act
)

// RegulatoryFrameworkConfig содержит настройки regulator框架.
type RegulatoryFrameworkConfig struct {
	Framework        RegulatoryFramework `json:"framework"`
	Region           string              `json:"region"`
	ReportingHours   int                 `json:"reporting_hours"` // Макс. часов на reporting
	SeverityLevel    IncidentSeverity    `json:"severity_level"`  // Минимальный severity для reporting
	Formats          []string            `json:"formats"`         // Поддерживаемые форматы
	RequireLegalHold bool                `json:"require_legal_hold"`
	RequireDPIA      bool                `json:"require_dpia"`
}

// DefaultFrameworkConfigs возвращает конфигурации по умолчанию для всех регионов.
func DefaultFrameworkConfigs() []RegulatoryFrameworkConfig {
	return []RegulatoryFrameworkConfig{
		{
			Framework:        FrameworkDORA,
			Region:           RegionEU,
			ReportingHours:   4,
			SeverityLevel:    SeverityNIS2Significant,
			Formats:          []string{"json", "xml"},
			RequireLegalHold: true,
			RequireDPIA:      false,
		},
		{
			Framework:        FrameworkNIS2,
			Region:           RegionEU,
			ReportingHours:   24,
			SeverityLevel:    SeverityNIS2Medium,
			Formats:          []string{"json", "xml", "pdf"},
			RequireLegalHold: true,
			RequireDPIA:      false,
		},
		{
			Framework:        FrameworkCERTIn,
			Region:           "IN",
			ReportingHours:   6,
			SeverityLevel:    SeverityNIS2Medium,
			Formats:          []string{"json", "xml"},
			RequireLegalHold: true,
			RequireDPIA:      true,
		},
		{
			Framework:        FrameworkCSA,
			Region:           "SG",
			ReportingHours:   2,
			SeverityLevel:    SeverityNIS2Medium,
			Formats:          []string{"json"},
			RequireLegalHold: true,
			RequireDPIA:      false,
		},
		{
			Framework:        FrameworkOAC,
			Region:           RegionBY,
			ReportingHours:   24,
			SeverityLevel:    SeverityNIS2Medium,
			Formats:          []string{"xml", "pdf"},
			RequireLegalHold: true,
			RequireDPIA:      true,
		},
		{
			Framework:        FrameworkFSTEK,
			Region:           RegionRU,
			ReportingHours:   24,
			SeverityLevel:    SeverityNIS2High,
			Formats:          []string{"xml"},
			RequireLegalHold: true,
			RequireDPIA:      false,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Incident Response Engine
// ═══════════════════════════════════════════════════════════════════════════

// IncidentResponseEngine — multi-tier incident response engine.
//
// Управляет:
//   - Multi-region classification and routing
//   - SLA timer management per regulatory framework
//   - Legal hold and evidence preservation
//   - Escalation matrix
//   - Automated report generation
type IncidentResponseEngine struct {
	mu               sync.RWMutex
	logger           *slog.Logger
	frameworks       []RegulatoryFrameworkConfig
	activeIncidents  map[string]*ActiveIncident
	registry         *ProfileRegistry
	evidenceStore    EvidenceStore
	notificationSink NotificationSink
}

// ActiveIncident представляет активный инцидент с multi-region трекингом.
type ActiveIncident struct {
	IncidentID      string                                         `json:"incident_id"`
	Classification  *IncidentClassification                        `json:"classification"`
	Incident        *Incident                                      `json:"incident"`
	CreatedAt       time.Time                                      `json:"created_at"`
	FrameworkStatus map[RegulatoryFramework]*FrameworkReportStatus `json:"framework_status"`
	LegalHolds      []LegalHold                                    `json:"legal_holds,omitempty"`
	Escalations     []EscalationEntry                              `json:"escalations,omitempty"`
}

// FrameworkReportStatus — статус reporting для конкретного фреймворка.
type FrameworkReportStatus struct {
	Framework     RegulatoryFramework `json:"framework"`
	Deadline      time.Time           `json:"deadline"`
	InitialReport bool                `json:"initial_report_sent"`
	FinalReport   bool                `json:"final_report_sent"`
	ReminderSent  bool                `json:"reminder_sent"`
	Acknowledged  bool                `json:"acknowledged"`
}

// LegalHold представляет юридическое удержание доказательств.
type LegalHold struct {
	ID            string    `json:"id"`
	IncidentID    string    `json:"incident_id"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	EvidenceTypes []string  `json:"evidence_types"`
	Status        string    `json:"status"` // active, released, expired
	CreatedBy     string    `json:"created_by"`
	Reason        string    `json:"reason"`
}

// EscalationEntry — запись об эскалации инцидента.
type EscalationEntry struct {
	ID          string     `json:"id"`
	IncidentID  string     `json:"incident_id"`
	Level       int        `json:"level"` // 1, 2, 3
	AssignedTo  string     `json:"assigned_to"`
	EscalatedAt time.Time  `json:"escalated_at"`
	Reason      string     `json:"reason"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// IncidentReport — готовый отчёт для регулятора.
type IncidentReport struct {
	Framework   RegulatoryFramework `json:"framework"`
	Format      string              `json:"format"`
	IncidentID  string              `json:"incident_id"`
	GeneratedAt time.Time           `json:"generated_at"`
	ReportBody  []byte              `json:"report_body"`
	ContentType string              `json:"content_type"`
}

// ═══════════════════════════════════════════════════════════════════════════
// Storage Interfaces
// ═══════════════════════════════════════════════════════════════════════════

// EvidenceStore — интерфейс для хранения доказательств.
type EvidenceStore interface {
	// SaveEvidence сохраняет доказательство.
	SaveEvidence(incidentID string, evidenceType string, data []byte) (string, error)
	// GetEvidence возвращает доказательство по ID.
	GetEvidence(id string) ([]byte, error)
	// ListEvidence возвращает список ID доказательств для инцидента.
	ListEvidence(incidentID string) ([]string, error)
	// DeleteEvidence удаляет доказательство.
	DeleteEvidence(id string) error
}

// NotificationSink — интерфейс для отправки уведомлений.
type NotificationSink interface {
	// Notify отправляет уведомление о событии.
	Notify(incidentID string, eventType string, payload interface{}) error
}

// ═══════════════════════════════════════════════════════════════════════════
// Constructor
// ═══════════════════════════════════════════════════════════════════════════

// NewIncidentResponseEngine создаёт новый IncidentResponseEngine.
func NewIncidentResponseEngine(
	logger *slog.Logger,
	registry *ProfileRegistry,
	evidenceStore EvidenceStore,
	notificationSink NotificationSink,
	frameworks ...RegulatoryFrameworkConfig,
) *IncidentResponseEngine {
	if logger == nil {
		logger = slog.Default().With("component", "compliance.incident_response")
	}

	cfg := DefaultFrameworkConfigs()
	if len(frameworks) > 0 {
		cfg = frameworks
	}

	return &IncidentResponseEngine{
		logger:           logger,
		frameworks:       cfg,
		activeIncidents:  make(map[string]*ActiveIncident),
		registry:         registry,
		evidenceStore:    evidenceStore,
		notificationSink: notificationSink,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Incident Registration & Tracking
// ═══════════════════════════════════════════════════════════════════════════

// RegisterIncident регистрирует инцидент и запускает multi-region трекинг.
func (e *IncidentResponseEngine) RegisterIncident(incident *Incident, classification *IncidentClassification) (*ActiveIncident, error) {
	if incident == nil || classification == nil {
		return nil, fmt.Errorf("incident_response: incident and classification are required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	active := &ActiveIncident{
		IncidentID:      incident.ID,
		Classification:  classification,
		Incident:        incident,
		CreatedAt:       time.Now().UTC(),
		FrameworkStatus: make(map[RegulatoryFramework]*FrameworkReportStatus),
		LegalHolds:      make([]LegalHold, 0),
		Escalations:     make([]EscalationEntry, 0),
	}

	// Определяем подходящие фреймворки
	for _, fw := range e.frameworks {
		if !e.isFrameworkApplicable(fw, classification) {
			continue
		}

		deadline := time.Now().UTC().Add(time.Duration(fw.ReportingHours) * time.Hour)
		active.FrameworkStatus[fw.Framework] = &FrameworkReportStatus{
			Framework: fw.Framework,
			Deadline:  deadline,
		}

		e.logger.Info("incident_response: framework tracking started",
			"incident_id", incident.ID,
			"framework", fw.Framework,
			"deadline", deadline.Format(time.RFC3339),
		)

		// Legal hold если требуется
		if fw.RequireLegalHold {
			hold := e.createLegalHold(incident.ID, fw)
			active.LegalHolds = append(active.LegalHolds, hold)
			e.logger.Info("incident_response: legal hold created",
				"incident_id", incident.ID,
				"framework", fw.Framework,
				"hold_id", hold.ID,
			)
		}
	}

	e.activeIncidents[incident.ID] = active
	return active, nil
}

// isFrameworkApplicable проверяет, применим ли фреймворк к данному инциденту.
func (e *IncidentResponseEngine) isFrameworkApplicable(fw RegulatoryFrameworkConfig, class *IncidentClassification) bool {
	// Проверяем по severity
	sevScore := severityScore(class.Severity)
	fwScore := severityScore(fw.SeverityLevel)
	return sevScore >= fwScore
}

// severityScore возвращает числовой score для severity.
func severityScore(s IncidentSeverity) int {
	switch s {
	case SeverityNIS2Low:
		return 1
	case SeverityNIS2Medium:
		return 2
	case SeverityNIS2High:
		return 3
	case SeverityNIS2Critical, SeverityNIS2Significant:
		return 4
	default:
		return 0
	}
}

// createLegalHold создаёт юридическое удержание доказательств.
func (e *IncidentResponseEngine) createLegalHold(incidentID string, fw RegulatoryFrameworkConfig) LegalHold {
	return LegalHold{
		ID:            fmt.Sprintf("LH-%s-%s", incidentID, time.Now().Format("20060102150405")),
		IncidentID:    incidentID,
		CreatedAt:     time.Now().UTC(),
		ExpiresAt:     time.Now().UTC().AddDate(0, 6, 0), // 6 месяцев
		EvidenceTypes: []string{"logs", "video_archive", "system_state", "network_capture"},
		Status:        "active",
		CreatedBy:     "incident_response_engine",
		Reason:        fmt.Sprintf("Legal hold required by %s (%s)", fw.Framework, fw.Region),
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// SLA Timer Management
// ═══════════════════════════════════════════════════════════════════════════

// GetOverdueFrameworks возвращает фреймворки, по которым deadline просрочен.
func (e *IncidentResponseEngine) GetOverdueFrameworks(incidentID string) []RegulatoryFramework {
	e.mu.RLock()
	active, ok := e.activeIncidents[incidentID]
	e.mu.RUnlock()

	if !ok {
		return nil
	}

	now := time.Now().UTC()
	var overdue []RegulatoryFramework

	for fw, status := range active.FrameworkStatus {
		if status.InitialReport {
			continue
		}
		if now.After(status.Deadline) {
			overdue = append(overdue, fw)
		}
	}

	return overdue
}

// GetTimeRemaining возвращает оставшееся время для каждого фреймворка.
func (e *IncidentResponseEngine) GetTimeRemaining(incidentID string) map[RegulatoryFramework]time.Duration {
	e.mu.RLock()
	active, ok := e.activeIncidents[incidentID]
	e.mu.RUnlock()

	result := make(map[RegulatoryFramework]time.Duration)
	if !ok {
		return result
	}

	for fw, status := range active.FrameworkStatus {
		if status.InitialReport {
			result[fw] = 0
			continue
		}
		remaining := time.Until(status.Deadline)
		if remaining < 0 {
			remaining = 0
		}
		result[fw] = remaining
	}

	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// Escalation Management
// ═══════════════════════════════════════════════════════════════════════════

// Escalate инкрементирует уровень эскалации для инцидента.
func (e *IncidentResponseEngine) Escalate(incidentID string, reason string) (*EscalationEntry, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	active, ok := e.activeIncidents[incidentID]
	if !ok {
		return nil, fmt.Errorf("incident_response: incident %s not found", incidentID)
	}

	currentLevel := len(active.Escalations) + 1
	if currentLevel > 3 {
		return nil, fmt.Errorf("incident_response: max escalation level reached for %s", incidentID)
	}

	entry := &EscalationEntry{
		ID:          fmt.Sprintf("ESC-%s-L%d", incidentID, currentLevel),
		IncidentID:  incidentID,
		Level:       currentLevel,
		AssignedTo:  escalationTarget(currentLevel),
		EscalatedAt: time.Now().UTC(),
		Reason:      reason,
	}

	active.Escalations = append(active.Escalations, *entry)

	e.logger.Warn("incident_response: escalation triggered",
		"incident_id", incidentID,
		"level", currentLevel,
		"assigned_to", entry.AssignedTo,
		"reason", reason,
	)

	return entry, nil
}

// escalationTarget возвращает цель эскалации для уровня.
func escalationTarget(level int) string {
	switch level {
	case 1:
		return "security_analyst"
	case 2:
		return "security_manager"
	case 3:
		return "ciso"
	default:
		return "unknown"
	}
}

// GetEscalationMatrix возвращает матрицу эскалации для инцидента.
func (e *IncidentResponseEngine) GetEscalationMatrix(incidentID string) []EscalationEntry {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if active, ok := e.activeIncidents[incidentID]; ok {
		result := make([]EscalationEntry, len(active.Escalations))
		copy(result, active.Escalations)
		return result
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Report Generation
// ═══════════════════════════════════════════════════════════════════════════

// GenerateRegulatoryReport генерирует отчёт для указанного регулятора.
func (e *IncidentResponseEngine) GenerateRegulatoryReport(incidentID string, framework RegulatoryFramework, format string) (*IncidentReport, error) {
	e.mu.RLock()
	active, ok := e.activeIncidents[incidentID]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("incident_response: incident %s not found", incidentID)
	}

	fwConfig := e.findFrameworkConfig(framework)
	if fwConfig == nil {
		return nil, fmt.Errorf("incident_response: framework %s not configured", framework)
	}

	// Проверяем поддерживаемость формата
	formatSupported := false
	for _, f := range fwConfig.Formats {
		if f == format {
			formatSupported = true
			break
		}
	}
	if !formatSupported {
		return nil, fmt.Errorf("incident_response: format %s not supported for %s", format, framework)
	}

	report := e.buildReport(active, fwConfig, format)

	// Отмечаем initial report как отправленный
	e.mu.Lock()
	if status, ok := active.FrameworkStatus[framework]; ok {
		if !status.InitialReport {
			status.InitialReport = true
		}
	}
	e.mu.Unlock()

	return report, nil
}

// findFrameworkConfig ищет конфигурацию фреймворка.
func (e *IncidentResponseEngine) findFrameworkConfig(framework RegulatoryFramework) *RegulatoryFrameworkConfig {
	for _, fw := range e.frameworks {
		if fw.Framework == framework {
			return &fw
		}
	}
	return nil
}

// buildReport строит отчёт для регулятора.
func (e *IncidentResponseEngine) buildReport(active *ActiveIncident, fwConfig *RegulatoryFrameworkConfig, format string) *IncidentReport {
	var body []byte
	var contentType string

	switch format {
	case "json":
		body = e.buildJSONReport(active, fwConfig)
		contentType = "application/json"
	case "xml":
		body = e.buildXMLReport(active, fwConfig)
		contentType = "application/xml"
	case "pdf":
		body = e.buildPDFReport(active, fwConfig)
		contentType = "application/pdf"
	default:
		body = e.buildJSONReport(active, fwConfig)
		contentType = "application/json"
	}

	return &IncidentReport{
		Framework:   fwConfig.Framework,
		Format:      format,
		IncidentID:  active.IncidentID,
		GeneratedAt: time.Now().UTC(),
		ReportBody:  body,
		ContentType: contentType,
	}
}

// buildJSONReport строит JSON отчёт.
func (e *IncidentResponseEngine) buildJSONReport(active *ActiveIncident, fwConfig *RegulatoryFrameworkConfig) []byte {
	// В production — полный structured report
	report := fmt.Sprintf(`{
	 "framework": "%s",
	 "region": "%s",
	 "incident_id": "%s",
	 "severity": "%s",
	 "reporting_deadline_hours": %d,
	 "generated_at": "%s",
	 "incident_type": "%s",
	 "affected_devices": %d,
	 "status": "initial_report"
}`,
		fwConfig.Framework,
		fwConfig.Region,
		active.IncidentID,
		active.Classification.Severity,
		fwConfig.ReportingHours,
		time.Now().UTC().Format(time.RFC3339),
		active.Classification.Type,
		countIncidentDevices(active.Incident),
	)

	return []byte(report)
}

// buildXMLReport строит XML отчёт.
func (e *IncidentResponseEngine) buildXMLReport(active *ActiveIncident, fwConfig *RegulatoryFrameworkConfig) []byte {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<incidentReport>
	 <framework>%s</framework>
	 <region>%s</region>
	 <incidentId>%s</incidentId>
	 <severity>%s</severity>
	 <reportingDeadlineHours>%d</reportingDeadlineHours>
	 <generatedAt>%s</generatedAt>
	 <incidentType>%s</incidentType>
	 <status>initial_report</status>
</incidentReport>`,
		fwConfig.Framework,
		fwConfig.Region,
		active.IncidentID,
		active.Classification.Severity,
		fwConfig.ReportingHours,
		time.Now().UTC().Format(time.RFC3339),
		active.Classification.Type,
	)

	return []byte(xml)
}

// buildPDFReport генерирует PDF отчёт (placeholder).
func (e *IncidentResponseEngine) buildPDFReport(active *ActiveIncident, fwConfig *RegulatoryFrameworkConfig) []byte {
	// В production — полноценный PDF через jung-kurt/gofpdf
	content := fmt.Sprintf("Incident Report: %s\nFramework: %s\nRegion: %s\nSeverity: %s",
		active.IncidentID, fwConfig.Framework, fwConfig.Region, active.Classification.Severity)
	return []byte(content)
}

func countIncidentDevices(incident *Incident) int {
	if incident == nil {
		return 0
	}
	// Считаем количество уникальных asset IDs из Actions
	deviceSet := make(map[string]bool)
	if incident.AssetID != "" {
		deviceSet[incident.AssetID] = true
	}
	return len(deviceSet)
}

// ═══════════════════════════════════════════════════════════════════════════
// Evidence Preservation
// ═══════════════════════════════════════════════════════════════════════════

// PreserveEvidence сохраняет доказательство по инциденту.
func (e *IncidentResponseEngine) PreserveEvidence(incidentID string, evidenceType string, data []byte) (string, error) {
	if e.evidenceStore == nil {
		return "", fmt.Errorf("incident_response: evidence store not configured")
	}

	id, err := e.evidenceStore.SaveEvidence(incidentID, evidenceType, data)
	if err != nil {
		return "", fmt.Errorf("incident_response: failed to preserve evidence: %w", err)
	}

	e.logger.Info("incident_response: evidence preserved",
		"incident_id", incidentID,
		"evidence_type", evidenceType,
		"evidence_id", id,
	)

	return id, nil
}

// GetActiveLegalHolds возвращает активные legal holds для инцидента.
func (e *IncidentResponseEngine) GetActiveLegalHolds(incidentID string) []LegalHold {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if active, ok := e.activeIncidents[incidentID]; ok {
		result := make([]LegalHold, 0)
		for _, hold := range active.LegalHolds {
			if hold.Status == "active" {
				result = append(result, hold)
			}
		}
		return result
	}
	return nil
}

// ReleaseLegalHold освобождает legal hold.
func (e *IncidentResponseEngine) ReleaseLegalHold(holdID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, active := range e.activeIncidents {
		for i, hold := range active.LegalHolds {
			if hold.ID == holdID {
				active.LegalHolds[i].Status = "released"
				e.logger.Info("incident_response: legal hold released",
					"hold_id", holdID,
					"incident_id", active.IncidentID,
				)
				return nil
			}
		}
	}

	return fmt.Errorf("incident_response: legal hold %s not found", holdID)
}

// ═══════════════════════════════════════════════════════════════════════════
// Query Methods
// ═══════════════════════════════════════════════════════════════════════════

// GetActiveIncident возвращает активный инцидент по ID.
func (e *IncidentResponseEngine) GetActiveIncident(incidentID string) *ActiveIncident {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.activeIncidents[incidentID]
}

// ListActiveIncidents возвращает список активных инцидентов.
func (e *IncidentResponseEngine) ListActiveIncidents() []*ActiveIncident {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]*ActiveIncident, 0, len(e.activeIncidents))
	for _, inc := range e.activeIncidents {
		result = append(result, inc)
	}

	// Сортируем по дате создания (новые сверху)
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// ListFrameworkConfigs возвращает список настроенных фреймворков.
func (e *IncidentResponseEngine) ListFrameworkConfigs() []RegulatoryFrameworkConfig {
	result := make([]RegulatoryFrameworkConfig, len(e.frameworks))
	copy(result, e.frameworks)
	return result
}

// ═══════════════════════════════════════════════════════════════════════════
// Notification Routing
// ═══════════════════════════════════════════════════════════════════════════

// NotifyFramework отправляет уведомление для указанного фреймворка.
func (e *IncidentResponseEngine) NotifyFramework(incidentID string, framework RegulatoryFramework, eventType string) error {
	if e.notificationSink == nil {
		return fmt.Errorf("incident_response: notification sink not configured")
	}

	e.mu.RLock()
	active, ok := e.activeIncidents[incidentID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("incident_response: incident %s not found", incidentID)
	}

	payload := map[string]interface{}{
		"incident_id": incidentID,
		"framework":   string(framework),
		"event_type":  eventType,
		"severity":    active.Classification.Severity,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"deadlines":   e.GetTimeRemaining(incidentID),
	}

	return e.notificationSink.Notify(incidentID, fmt.Sprintf("incident.%s.%s", strings.ToLower(string(framework)), eventType), payload)
}

// RemindPending напоминает о pending reports для всех активных инцидентов.
func (e *IncidentResponseEngine) RemindPending() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	var reminded []string
	now := time.Now().UTC()

	for id, active := range e.activeIncidents {
		for fw, status := range active.FrameworkStatus {
			if status.InitialReport || status.ReminderSent {
				continue
			}

			remaining := time.Until(status.Deadline)
			// Отправляем reminder за 30% до дедлайна
			totalDuration := status.Deadline.Sub(active.CreatedAt)
			reminderThreshold := totalDuration - time.Duration(float64(totalDuration)*0.3)

			if now.After(active.CreatedAt.Add(reminderThreshold)) {
				status.ReminderSent = true
				reminded = append(reminded, fmt.Sprintf("%s/%s", id, fw))

				e.logger.Warn("incident_response: deadline reminder",
					"incident_id", id,
					"framework", fw,
					"remaining_hours", remaining.Hours(),
				)
			}
		}
	}

	return reminded
}
