// Package compliance — unit tests for NIS2 Incident Reporting (P2-EU.2).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-EU.2: NIS2 Incident Reporting Tests
//
// Соответствие:
//   - ISO 27001 A.16.1 (Incident management testing)
//   - IEC 62443 SR 3.1 (Boundary testing)
//   - NIS2 Art. 23 (Incident reporting compliance)
//   - СТБ 34.101.27 п. 7.4 (Тестирование безопасности)
//
// ═══════════════════════════════════════════════════════════════════════════
package compliance

import (
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// Mock NIS2Store
// ═══════════════════════════════════════════════════════════════════════════

// mockNIS2Store implements NIS2Store for testing.
type mockNIS2Store struct {
	incidents map[string]*Incident
	reports   map[string]*NIS2Report
	timelines map[string][]*TimelineEntry
}

func newMockNIS2Store() *mockNIS2Store {
	return &mockNIS2Store{
		incidents: make(map[string]*Incident),
		reports:   make(map[string]*NIS2Report),
		timelines: make(map[string][]*TimelineEntry),
	}
}

func (m *mockNIS2Store) SaveIncident(_ interface{}, incident *Incident) error {
	m.incidents[incident.ID] = incident
	return nil
}

func (m *mockNIS2Store) GetIncident(_ interface{}, id string) (*Incident, error) {
	incident, ok := m.incidents[id]
	if !ok {
		return nil, nil
	}
	return incident, nil
}

func (m *mockNIS2Store) ListIncidents(_ interface{}, severity IncidentSeverity, limit, offset int) ([]*Incident, error) {
	result := make([]*Incident, 0)
	for _, inc := range m.incidents {
		if severity == "" || inc.Classification.Severity == severity {
			result = append(result, inc)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockNIS2Store) UpdateIncident(_ interface{}, incident *Incident) error {
	m.incidents[incident.ID] = incident
	return nil
}

func (m *mockNIS2Store) SaveReport(_ interface{}, report *NIS2Report) error {
	m.reports[report.ID] = report
	return nil
}

func (m *mockNIS2Store) GetReport(_ interface{}, id string) (*NIS2Report, error) {
	report, ok := m.reports[id]
	if !ok {
		return nil, nil
	}
	return report, nil
}

func (m *mockNIS2Store) ListReports(_ interface{}, incidentID string) ([]*NIS2Report, error) {
	result := make([]*NIS2Report, 0)
	for _, report := range m.reports {
		if report.IncidentID == incidentID {
			result = append(result, report)
		}
	}
	return result, nil
}

func (m *mockNIS2Store) AddTimelineEntry(_ interface{}, incidentID string, entry *TimelineEntry) error {
	m.timelines[incidentID] = append(m.timelines[incidentID], entry)
	return nil
}

func (m *mockNIS2Store) GetTimeline(_ interface{}, incidentID string) ([]*TimelineEntry, error) {
	return m.timelines[incidentID], nil
}

// ═══════════════════════════════════════════════════════════════════════════
// TestNewNIS2Manager
// ═══════════════════════════════════════════════════════════════════════════

func TestNewNIS2Manager(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	if mgr == nil {
		t.Fatal("NewNIS2Manager must not return nil")
	}

	if mgr.store == nil {
		t.Error("NewNIS2Manager: store should not be nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestClassifyIncident
// ═══════════════════════════════════════════════════════════════════════════

func TestClassifyIncidentUnauthorizedAccess(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:             "evt-001",
		Source:         "server",
		Type:           "unauthorized_access",
		Description:    "Unauthorized login attempt detected on admin panel",
		DetectedAt:     time.Now().UTC(),
		AffectedAssets: []string{"admin-panel", "auth-service"},
	}

	class := mgr.ClassifyIncident(event)
	if class == nil {
		t.Fatal("ClassifyIncident returned nil")
	}

	if class.Type != IncidentTypeUnauthorizedAccess {
		t.Errorf("Type = %s, want %s", class.Type, IncidentTypeUnauthorizedAccess)
	}

	if class.Confidence <= 0 || class.Confidence > 1.0 {
		t.Errorf("Confidence out of range (0,1]: %f", class.Confidence)
	}

	if class.Severity == "" {
		t.Error("Severity should not be empty")
	}

	if len(class.Impact) == 0 {
		t.Error("Impact should not be empty")
	}

	if class.Rationale == "" {
		t.Error("Rationale should not be empty")
	}
}

func TestClassifyIncidentNilEvent(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	class := mgr.ClassifyIncident(nil)
	if class == nil {
		t.Fatal("ClassifyIncident(nil) should return non-nil classification")
	}

	if class.Severity != SeverityNIS2Low {
		t.Errorf("Nil event severity = %s, want %s", class.Severity, SeverityNIS2Low)
	}

	if class.Confidence != 0.0 {
		t.Errorf("Nil event confidence = %f, want 0.0", class.Confidence)
	}
}

func TestClassifyIncidentSystemFailure(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:          "evt-002",
		Source:      "camera",
		Type:        "offline",
		Description: "Camera NVR-03 went offline",
		DetectedAt:  time.Now().UTC(),
	}

	class := mgr.ClassifyIncident(event)
	if class == nil {
		t.Fatal("ClassifyIncident returned nil")
	}

	if class.Type != IncidentTypeSystemFailure {
		t.Errorf("Camera offline type = %s, want %s", class.Type, IncidentTypeSystemFailure)
	}
}

func TestClassifyIncidentDataBreach(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:          "evt-003",
		Source:      "server",
		Type:        "data_breach",
		Description: "Video archive data leak detected",
		DetectedAt:  time.Now().UTC(),
		AffectedAssets: []string{"archive-server", "db-01", "db-02", "nvr-01", "nvr-02", "nvr-03",
			"nvr-04", "nvr-05", "nvr-06", "nvr-07", "nvr-08"},
	}

	class := mgr.ClassifyIncident(event)
	if class == nil {
		t.Fatal("ClassifyIncident returned nil")
	}

	if class.Type != IncidentTypeDataBreach {
		t.Errorf("Type = %s, want %s", class.Type, IncidentTypeDataBreach)
	}

	// Data breach with many assets should be critical
	if class.Severity != SeverityNIS2Critical {
		t.Errorf("Major data breach severity = %s, want %s", class.Severity, SeverityNIS2Critical)
	}

	// Should include confidentiality impact
	hasConfidentiality := false
	for _, impact := range class.Impact {
		if impact == ImpactConfidentiality {
			hasConfidentiality = true
			break
		}
	}
	if !hasConfidentiality {
		t.Error("Data breach should have confidentiality impact")
	}
}

func TestClassifyIncidentPhysicalTampering(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:             "evt-004",
		Source:         "camera",
		Type:           "tamper",
		Description:    "Physical tamper detected on camera CAM-12",
		DetectedAt:     time.Now().UTC(),
		AffectedAssets: []string{"CAM-12"},
	}

	class := mgr.ClassifyIncident(event)
	if class == nil {
		t.Fatal("ClassifyIncident returned nil")
	}

	if class.Type != IncidentTypePhysicalTampering {
		t.Errorf("Type = %s, want %s", class.Type, IncidentTypePhysicalTampering)
	}

	// Physical tampering should have safety impact
	hasSafety := false
	for _, impact := range class.Impact {
		if impact == ImpactSafety {
			hasSafety = true
			break
		}
	}
	if !hasSafety {
		t.Error("Physical tampering should have safety impact")
	}
}

func TestClassifyIncidentDDoS(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:             "evt-005",
		Source:         "api",
		Type:           "ddos_flood",
		Description:    "DDoS attack detected on API gateway",
		DetectedAt:     time.Now().UTC(),
		AffectedAssets: []string{"api-gateway", "rate-limiter"},
	}

	class := mgr.ClassifyIncident(event)
	if class == nil {
		t.Fatal("ClassifyIncident returned nil")
	}

	if class.Type != IncidentTypeDenialOfService {
		t.Errorf("Type = %s, want %s", class.Type, IncidentTypeDenialOfService)
	}

	// DDoS should have availability impact
	hasAvailability := false
	for _, impact := range class.Impact {
		if impact == ImpactAvailability {
			hasAvailability = true
			break
		}
	}
	if !hasAvailability {
		t.Error("DDoS should have availability impact")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestClassifyIncident — table-driven tests
// ═══════════════════════════════════════════════════════════════════════════

func TestClassifyIncidentTableDriven(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	tests := []struct {
		name     string
		event    *IncidentEvent
		wantType IncidentType
	}{
		{
			name: "unauthorized access on server",
			event: &IncidentEvent{
				Source: "server", Type: "unauthorized", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeUnauthorizedAccess,
		},
		{
			name: "malware detected",
			event: &IncidentEvent{
				Source: "server", Type: "ransomware", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeMalware,
		},
		{
			name: "configuration change",
			event: &IncidentEvent{
				Source: "nvr", Type: "misconfig", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeConfigurationChange,
		},
		{
			name: "network breach",
			event: &IncidentEvent{
				Source: "server", Type: "port_scan", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeNetworkBreach,
		},
		{
			name: "insider threat",
			event: &IncidentEvent{
				Source: "server", Type: "insider", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeInsiderThreat,
		},
		{
			name: "third party breach",
			event: &IncidentEvent{
				Source: "api", Type: "third_party", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeThirdParty,
		},
		{
			name: "default system failure",
			event: &IncidentEvent{
				Source: "unknown", Type: "unknown", Description: "test",
				DetectedAt: time.Now().UTC(),
			},
			wantType: IncidentTypeSystemFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class := mgr.ClassifyIncident(tt.event)
			if class == nil {
				t.Fatal("ClassifyIncident returned nil")
			}
			if class.Type != tt.wantType {
				t.Errorf("Type = %s, want %s", class.Type, tt.wantType)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestCreateIncident
// ═══════════════════════════════════════════════════════════════════════════

func TestCreateIncident(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:             "evt-001",
		Source:         "server",
		Type:           "unauthorized_access",
		Description:    "Unauthorized access detected on admin panel",
		DetectedAt:     time.Now().UTC(),
		AffectedAssets: []string{"admin-panel"},
	}

	incident, err := mgr.CreateIncident(event)
	if err != nil {
		t.Fatalf("CreateIncident error: %v", err)
	}

	if incident == nil {
		t.Fatal("CreateIncident returned nil")
	}

	if incident.ID == "" {
		t.Error("Incident ID should not be empty")
	}

	if incident.Status != "open" {
		t.Errorf("Status = %s, want open", incident.Status)
	}

	if len(incident.Timeline) == 0 {
		t.Error("Incident should have at least one timeline entry")
	}

	if incident.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	if incident.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestCreateIncidentNilEvent(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	_, err := mgr.CreateIncident(nil)
	if err == nil {
		t.Fatal("CreateIncident with nil event should return error")
	}
}

func TestCreateIncidentCriticalGeneratesEarlyWarning(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:          "evt-critical",
		Source:      "server",
		Type:        "data_breach",
		Description: "Massive data breach detected",
		DetectedAt:  time.Now().UTC(),
		AffectedAssets: []string{"db-01", "db-02", "db-03", "db-04", "db-05",
			"nvr-01", "nvr-02", "nvr-03", "nvr-04", "nvr-05", "nvr-06"},
	}

	incident, err := mgr.CreateIncident(event)
	if err != nil {
		t.Fatalf("CreateIncident error: %v", err)
	}

	if incident.Classification.Severity != SeverityNIS2Critical {
		t.Fatalf("Severity = %s, want %s", incident.Classification.Severity, SeverityNIS2Critical)
	}

	// Critical incidents should have early warning report
	if len(incident.Reports) == 0 {
		t.Error("Critical incident should have early warning report")
	}

	if len(incident.Reports) > 0 && incident.Reports[0].Phase != ReportPhaseEarlyWarning {
		t.Errorf("Report phase = %s, want %s", incident.Reports[0].Phase, ReportPhaseEarlyWarning)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestGenerateNIS2Report
// ═══════════════════════════════════════════════════════════════════════════

func TestGenerateNIS2ReportNilIncident(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	_, err := mgr.GenerateNIS2Report(nil, ReportPhaseNotification, "xml")
	if err == nil {
		t.Fatal("GenerateNIS2Report with nil incident should return error")
	}
}

func TestGenerateNIS2ReportXML(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	incident := &Incident{
		ID: "inc-test-001",
		Classification: IncidentClassification{
			Severity:    SeverityNIS2High,
			Type:        IncidentTypeUnauthorizedAccess,
			Impact:      []ImpactCategory{ImpactConfidentiality, ImpactAvailability},
			Confidence:  0.92,
			Description: "Unauthorized access detected",
			Rationale:   "Test rationale",
		},
		Status:     "investigating",
		DetectedAt: time.Now().UTC().Add(-2 * time.Hour),
		UpdatedAt:  time.Now().UTC(),
		Zone:       "Zone 3 (Application)",
		Timeline: []TimelineEntry{
			{
				ID:          "tl-001",
				Timestamp:   time.Now().UTC().Add(-2 * time.Hour),
				Event:       "incident_detected",
				Source:      "auth_service",
				Description: "Anomalous login pattern detected",
			},
		},
		CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}

	xmlData, err := mgr.GenerateNIS2Report(incident, ReportPhaseNotification, "xml")
	if err != nil {
		t.Fatalf("GenerateNIS2Report error: %v", err)
	}

	if len(xmlData) == 0 {
		t.Fatal("GenerateNIS2Report returned empty data")
	}

	output := string(xmlData)
	if len(output) < 20 || output[:5] != "<?xml" {
		t.Errorf("Output should start with <?xml, got %q", output[:min(30, len(output))])
	}

	if !contains(output, "NIS2") && !contains(output, "nis2Report") {
		t.Error("XML output should contain NIS2 report elements")
	}
}

func TestGenerateNIS2ReportPDF(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	incident := &Incident{
		ID: "inc-test-002",
		Classification: IncidentClassification{
			Severity:    SeverityNIS2Significant,
			Type:        IncidentTypeDataBreach,
			Impact:      []ImpactCategory{ImpactConfidentiality, ImpactRegulatory, ImpactFinancial},
			Confidence:  0.95,
			Description: "Data breach detected in video archive",
			Rationale:   "Significant incident requiring NIS2 reporting",
		},
		Status:      "investigating",
		DetectedAt:  time.Now().UTC().Add(-6 * time.Hour),
		UpdatedAt:   time.Now().UTC(),
		Zone:        "Zone 4 (Data)",
		Description: "Video archive data potentially compromised",
		Actions: []IncidentAction{
			{
				ID:          "act-001",
				Action:      "Isolate affected storage",
				Owner:       "security-team",
				Status:      "completed",
				PerformedAt: time.Now().UTC().Add(-4 * time.Hour),
				Notes:       "Archive server isolated from network",
			},
		},
		Timeline: []TimelineEntry{
			{
				ID:          "tl-001",
				Timestamp:   time.Now().UTC().Add(-6 * time.Hour),
				Event:       "incident_detected",
				Source:      "ids_system",
				Description: "Data exfiltration detected",
			},
			{
				ID:          "tl-002",
				Timestamp:   time.Now().UTC().Add(-5 * time.Hour),
				Event:       "ioc_collected",
				Source:      "forensic_tool",
				Description: "Indicator of compromise: unusual outbound traffic",
			},
		},
		CreatedAt: time.Now().UTC().Add(-6 * time.Hour),
	}

	pdfData, err := mgr.GenerateNIS2Report(incident, ReportPhaseEarlyWarning, "pdf")
	if err != nil {
		t.Fatalf("GenerateNIS2Report(PDF) error: %v", err)
	}

	if len(pdfData) == 0 {
		t.Fatal("GenerateNIS2Report(PDF) returned empty data")
	}

	// Check PDF signature
	if len(pdfData) < 5 || string(pdfData[:5]) != "%PDF-" {
		t.Errorf("Output should start with %%PDF-, got %q", string(pdfData[:min(20, len(pdfData))]))
	}
}

func TestGenerateNIS2ReportUnsupportedFormat(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	incident := &Incident{
		ID: "inc-test-003",
		Classification: IncidentClassification{
			Severity: SeverityNIS2Low,
			Type:     IncidentTypeSystemFailure,
			Impact:   []ImpactCategory{ImpactAvailability},
		},
		Status:     "open",
		DetectedAt: time.Now().UTC(),
		Timeline:   []TimelineEntry{},
		CreatedAt:  time.Now().UTC(),
	}

	_, err := mgr.GenerateNIS2Report(incident, ReportPhaseNotification, "csv")
	if err == nil {
		t.Fatal("GenerateNIS2Report with unsupported format should return error")
	}
}

func TestGenerateNIS2ReportPhaseDeadlines(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	incident := &Incident{
		ID: "inc-test-004",
		Classification: IncidentClassification{
			Severity: SeverityNIS2Medium,
			Type:     IncidentTypeConfigurationChange,
			Impact:   []ImpactCategory{ImpactIntegrity},
		},
		Status:     "resolved",
		DetectedAt: time.Now().UTC(),
		Timeline:   []TimelineEntry{},
		CreatedAt:  time.Now().UTC(),
	}

	deadline := mgr.calculateDeadline(incident.DetectedAt, ReportPhaseEarlyWarning)
	expected := incident.DetectedAt.Add(24 * time.Hour)
	if !deadline.Equal(expected) {
		t.Errorf("Early warning deadline = %v, want %v", deadline, expected)
	}

	deadline = mgr.calculateDeadline(incident.DetectedAt, ReportPhaseNotification)
	expected = incident.DetectedAt.Add(72 * time.Hour)
	if !deadline.Equal(expected) {
		t.Errorf("Notification deadline = %v, want %v", deadline, expected)
	}

	deadline = mgr.calculateDeadline(incident.DetectedAt, ReportPhaseFinal)
	expected = incident.DetectedAt.Add(30 * 24 * time.Hour)
	if !deadline.Equal(expected) {
		t.Errorf("Final deadline = %v, want %v", deadline, expected)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestGetIncidentTimeline
// ═══════════════════════════════════════════════════════════════════════════

func TestGetIncidentTimeline(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	now := time.Now().UTC()

	// Add timeline entries
	entry1 := &TimelineEntry{
		Timestamp:   now.Add(-1 * time.Hour),
		Event:       "incident_detected",
		Source:      "detector",
		Description: "First detection",
	}

	entry2 := &TimelineEntry{
		Timestamp:   now.Add(-30 * time.Minute),
		Event:       "analysis_started",
		Source:      "analyst",
		Description: "Manual analysis initiated",
	}

	_ = mgr.AddTimelineEntry("inc-test-tl", entry1)
	_ = mgr.AddTimelineEntry("inc-test-tl", entry2)

	timeline, err := mgr.GetIncidentTimeline("inc-test-tl")
	if err != nil {
		t.Fatalf("GetIncidentTimeline error: %v", err)
	}

	if len(timeline) != 2 {
		t.Errorf("Timeline length = %d, want 2", len(timeline))
	}

	// Should be sorted by timestamp
	if len(timeline) == 2 {
		if timeline[0].Timestamp.After(timeline[1].Timestamp) {
			t.Error("Timeline should be sorted by timestamp ascending")
		}
	}
}

func TestGetIncidentTimelineEmptyID(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	_, err := mgr.GetIncidentTimeline("")
	if err == nil {
		t.Fatal("GetIncidentTimeline with empty ID should return error")
	}
}

func TestAddTimelineEntry(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	entry := &TimelineEntry{
		Event:       "test_event",
		Source:      "test",
		Description: "Test timeline entry",
	}

	err := mgr.AddTimelineEntry("inc-test-ae", entry)
	if err != nil {
		t.Fatalf("AddTimelineEntry error: %v", err)
	}

	if entry.ID == "" {
		t.Error("Timeline entry ID should be set after AddTimelineEntry")
	}

	if entry.Timestamp.IsZero() {
		t.Error("Timeline entry Timestamp should be set")
	}
}

func TestAddTimelineEntryNilEntry(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	err := mgr.AddTimelineEntry("inc-test-nil", nil)
	if err == nil {
		t.Fatal("AddTimelineEntry with nil entry should return error")
	}
}

func TestAddTimelineEntryEmptyID(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	err := mgr.AddTimelineEntry("", &TimelineEntry{Event: "test"})
	if err == nil {
		t.Fatal("AddTimelineEntry with empty incident ID should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestUpdateIncidentStatus
// ═══════════════════════════════════════════════════════════════════════════

func TestUpdateIncidentStatus(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	event := &IncidentEvent{
		ID:          "evt-status",
		Source:      "server",
		Type:        "unauthorized",
		Description: "test",
		DetectedAt:  time.Now().UTC(),
	}

	incident, err := mgr.CreateIncident(event)
	if err != nil {
		t.Fatalf("CreateIncident error: %v", err)
	}

	err = mgr.UpdateIncidentStatus(incident.ID, "investigating")
	if err != nil {
		t.Fatalf("UpdateIncidentStatus error: %v", err)
	}

	updated, err := mgr.GetIncident(incident.ID)
	if err != nil {
		t.Fatalf("GetIncident error: %v", err)
	}

	if updated.Status != "investigating" {
		t.Errorf("Status = %s, want investigating", updated.Status)
	}

	// Resolve
	err = mgr.UpdateIncidentStatus(incident.ID, "resolved")
	if err != nil {
		t.Fatalf("UpdateIncidentStatus(resolved) error: %v", err)
	}

	updated, _ = mgr.GetIncident(incident.ID)
	if updated.ResolvedAt == nil {
		t.Error("ResolvedAt should be set when incident is resolved")
	}
}

func TestUpdateIncidentStatusEmptyID(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	err := mgr.UpdateIncidentStatus("", "resolved")
	if err == nil {
		t.Fatal("UpdateIncidentStatus with empty ID should return error")
	}
}

func TestUpdateIncidentStatusEmptyStatus(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	err := mgr.UpdateIncidentStatus("inc-test", "")
	if err == nil {
		t.Fatal("UpdateIncidentStatus with empty status should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestIncidentSeverityFromString
// ═══════════════════════════════════════════════════════════════════════════

func TestIncidentSeverityFromString(t *testing.T) {
	tests := []struct {
		input string
		want  IncidentSeverity
	}{
		{"low", SeverityNIS2Low},
		{"medium", SeverityNIS2Medium},
		{"high", SeverityNIS2High},
		{"significant", SeverityNIS2Significant},
		{"critical", SeverityNIS2Critical},
		{"LOW", SeverityNIS2Low},
		{"CRITICAL", SeverityNIS2Critical},
		{"unknown", SeverityNIS2Low},
		{"", SeverityNIS2Low},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IncidentSeverityFromString(tt.input)
			if got != tt.want {
				t.Errorf("IncidentSeverityFromString(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestIncidentTypeFromString
// ═══════════════════════════════════════════════════════════════════════════

func TestIncidentTypeFromString(t *testing.T) {
	tests := []struct {
		input string
		want  IncidentType
	}{
		{"unauthorized_access", IncidentTypeUnauthorizedAccess},
		{"unauthorized", IncidentTypeUnauthorizedAccess},
		{"data_breach", IncidentTypeDataBreach},
		{"breach", IncidentTypeDataBreach},
		{"system_failure", IncidentTypeSystemFailure},
		{"malware", IncidentTypeMalware},
		{"ransomware", IncidentTypeMalware},
		{"physical_tampering", IncidentTypePhysicalTampering},
		{"dos", IncidentTypeDenialOfService},
		{"ddos", IncidentTypeDenialOfService},
		{"misconfig", IncidentTypeConfigurationChange},
		{"network_breach", IncidentTypeNetworkBreach},
		{"insider_threat", IncidentTypeInsiderThreat},
		{"third_party_breach", IncidentTypeThirdParty},
		{"unknown_type", IncidentTypeSystemFailure},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IncidentTypeFromString(tt.input)
			if got != tt.want {
				t.Errorf("IncidentTypeFromString(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestImpactCategories
// ═══════════════════════════════════════════════════════════════════════════

func TestIncidentClassificationImpact(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	// Test that different event types produce different impact categories
	events := []struct {
		name       string
		event      *IncidentEvent
		wantImpact ImpactCategory
	}{
		{
			name: "offline camera",
			event: &IncidentEvent{
				Source: "camera", Type: "offline", Description: "camera offline",
				DetectedAt: time.Now().UTC(),
			},
			wantImpact: ImpactAvailability,
		},
		{
			name: "data leak",
			event: &IncidentEvent{
				Source: "server", Type: "data_leak", Description: "data leak",
				DetectedAt: time.Now().UTC(),
			},
			wantImpact: ImpactConfidentiality,
		},
		{
			name: "config change",
			event: &IncidentEvent{
				Source: "nvr", Type: "misconfig", Description: "misconfiguration",
				DetectedAt: time.Now().UTC(),
			},
			wantImpact: ImpactIntegrity,
		},
		{
			name: "physical tamper",
			event: &IncidentEvent{
				Source: "camera", Type: "tamper", Description: "tamper",
				DetectedAt: time.Now().UTC(),
			},
			wantImpact: ImpactSafety,
		},
	}

	for _, tt := range events {
		t.Run(tt.name, func(t *testing.T) {
			class := mgr.ClassifyIncident(tt.event)
			if class == nil {
				t.Fatal("ClassifyIncident returned nil")
			}

			found := false
			for _, impact := range class.Impact {
				if impact == tt.wantImpact {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Impact should contain %s, got %v", tt.wantImpact, class.Impact)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestListIncidents
// ═══════════════════════════════════════════════════════════════════════════

func TestListIncidents(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	// Create a few incidents
	events := []*IncidentEvent{
		{Source: "server", Type: "unauthorized", Description: "auth", DetectedAt: time.Now().UTC()},
		{Source: "camera", Type: "offline", Description: "offline", DetectedAt: time.Now().UTC()},
		{Source: "server", Type: "data_breach", Description: "breach", DetectedAt: time.Now().UTC(),
			AffectedAssets: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}},
	}

	for _, evt := range events {
		_, err := mgr.CreateIncident(evt)
		if err != nil {
			t.Fatalf("CreateIncident error: %v", err)
		}
	}

	// List all
	all, err := mgr.ListIncidents("", 50, 0)
	if err != nil {
		t.Fatalf("ListIncidents error: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("ListIncidents length = %d, want 3", len(all))
	}

	// List with severity filter
	critical, err := mgr.ListIncidents(SeverityNIS2Critical, 50, 0)
	if err != nil {
		t.Fatalf("ListIncidents(critical) error: %v", err)
	}

	if len(critical) == 0 {
		t.Error("Should have at least one critical incident (data breach)")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestGetIncident
// ═══════════════════════════════════════════════════════════════════════════

func TestGetIncident(t *testing.T) {
	store := newMockNIS2Store()
	mgr := NewNIS2Manager(store, nil)

	_, err := mgr.GetIncident("")
	if err == nil {
		t.Fatal("GetIncident with empty ID should return error")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// TestMapSourceToZone
// ═══════════════════════════════════════════════════════════════════════════

func TestMapSourceToZone(t *testing.T) {
	mgr := NewNIS2Manager(newMockNIS2Store(), nil)

	tests := []struct {
		source string
		want   string
	}{
		{"camera", "Zone 5 (Edge)"},
		{"nvr", "Zone 4 (Data)"},
		{"server", "Zone 3 (Application)"},
		{"api", "Zone 2 (DMZ)"},
		{"db", "Zone 4 (Data)"},
		{"nats", "Zone 3 (Application)"},
		{"edge", "Zone 5 (Edge)"},
		{"unknown", "Zone 3 (Application)"},
		{"CAMERA", "Zone 5 (Edge)"},
	}

	for _, tt := range tests {
		t.Run(tt.source, func(t *testing.T) {
			got := mgr.mapSourceToZone(tt.source)
			if got != tt.want {
				t.Errorf("mapSourceToZone(%q) = %s, want %s", tt.source, got, tt.want)
			}
		})
	}
}
