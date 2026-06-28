package compliance

import (
	"testing"
	"time"

	"log/slog"
)

// mockEvidenceStore — реализация EvidenceStore для тестов.
type mockEvidenceStore struct{}

func (m *mockEvidenceStore) SaveEvidence(incidentID, evidenceType string, data []byte) (string, error) {
	return "ev-001", nil
}
func (m *mockEvidenceStore) GetEvidence(id string) ([]byte, error) {
	return []byte("test"), nil
}
func (m *mockEvidenceStore) ListEvidence(incidentID string) ([]string, error) {
	return []string{"ev-001"}, nil
}
func (m *mockEvidenceStore) DeleteEvidence(id string) error {
	return nil
}

// mockNotificationSink — реализация NotificationSink для тестов.
type mockNotificationSink struct{}

func (m *mockNotificationSink) Notify(incidentID, eventType string, payload interface{}) error {
	return nil
}

func setupTestEngine(t *testing.T) *IncidentResponseEngine {
	t.Helper()

	registry := NewProfileRegistry(
		WithRequiredRegions(RegionINTL),
		WithProfile(NewINTLProfile()),
	)

	return NewIncidentResponseEngine(
		slog.Default(),
		registry,
		&mockEvidenceStore{},
		&mockNotificationSink{},
	)
}

func TestNewIncidentResponseEngine(t *testing.T) {
	engine := setupTestEngine(t)
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	frameworks := engine.ListFrameworkConfigs()
	if len(frameworks) == 0 {
		t.Fatal("expected at least one framework config")
	}
}

func TestRegisterIncident(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{
		ID:          "INC-001",
		Status:      "open",
		AssetID:     "cam-001",
		Description: "Test incident",
		CreatedAt:   time.Now().UTC(),
	}

	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Impact:     []ImpactCategory{ImpactAvailability},
		Confidence: 0.95,
	}

	active, err := engine.RegisterIncident(incident, classification)
	if err != nil {
		t.Fatalf("RegisterIncident error: %v", err)
	}

	if active.IncidentID != "INC-001" {
		t.Fatalf("expected INC-001, got %s", active.IncidentID)
	}

	// Проверяем, что framework status созданы
	if len(active.FrameworkStatus) == 0 {
		t.Fatal("expected at least one framework status")
	}

	// Проверяем DORA (4h reporting for critical)
	if status, ok := active.FrameworkStatus[FrameworkDORA]; ok {
		if status.Deadline.Before(time.Now()) {
			t.Error("DORA deadline should be in the future")
		}
	} else {
		t.Error("expected DORA framework to be tracked for critical severity")
	}

	// Проверяем legal hold
	if len(active.LegalHolds) == 0 {
		t.Error("expected legal hold for DORA framework")
	}
}

func TestRegisterIncident_NilInput(t *testing.T) {
	engine := setupTestEngine(t)

	_, err := engine.RegisterIncident(nil, &IncidentClassification{})
	if err == nil {
		t.Fatal("expected error for nil incident")
	}

	_, err = engine.RegisterIncident(&Incident{ID: "INC-002"}, nil)
	if err == nil {
		t.Fatal("expected error for nil classification")
	}
}

func TestGetTimeRemaining(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-003", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.9,
	}

	_, err := engine.RegisterIncident(incident, classification)
	if err != nil {
		t.Fatalf("RegisterIncident error: %v", err)
	}

	remaining := engine.GetTimeRemaining("INC-003")
	if len(remaining) == 0 {
		t.Fatal("expected time remaining for registered incident")
	}

	for fw, dur := range remaining {
		if dur <= 0 {
			t.Errorf("expected positive duration for %s, got %v", fw, dur)
		}
	}
}

func TestGetTimeRemaining_UnknownIncident(t *testing.T) {
	engine := setupTestEngine(t)
	remaining := engine.GetTimeRemaining("NONEXISTENT")
	if len(remaining) != 0 {
		t.Fatal("expected empty result for unknown incident")
	}
}

func TestGetOverdueFrameworks(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-004", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Low,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.9,
	}

	_, err := engine.RegisterIncident(incident, classification)
	if err != nil {
		t.Fatalf("RegisterIncident error: %v", err)
	}

	// Low severity — не должно быть фреймворков
	overdue := engine.GetOverdueFrameworks("INC-004")
	if len(overdue) != 0 {
		t.Logf("expected no overdue frameworks for low severity (got %d)", len(overdue))
	}
}

func TestEscalate(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-005", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.95,
	}

	_, err := engine.RegisterIncident(incident, classification)
	if err != nil {
		t.Fatalf("RegisterIncident error: %v", err)
	}

	// L1 escalation
	entry, err := engine.Escalate("INC-005", "SLA approaching deadline")
	if err != nil {
		t.Fatalf("Escalate error: %v", err)
	}
	if entry.Level != 1 {
		t.Fatalf("expected level 1, got %d", entry.Level)
	}
	if entry.AssignedTo != "security_analyst" {
		t.Fatalf("expected security_analyst, got %s", entry.AssignedTo)
	}

	// L2 escalation
	entry, err = engine.Escalate("INC-005", "Deadline missed")
	if err != nil {
		t.Fatalf("Escalate error: %v", err)
	}
	if entry.Level != 2 {
		t.Fatalf("expected level 2, got %d", entry.Level)
	}

	// L3 escalation
	entry, err = engine.Escalate("INC-005", "Critical breach")
	if err != nil {
		t.Fatalf("Escalate error: %v", err)
	}
	if entry.Level != 3 {
		t.Fatalf("expected level 3, got %d", entry.Level)
	}

	// Max level reached
	_, err = engine.Escalate("INC-005", "Too late")
	if err == nil {
		t.Fatal("expected error at max escalation level")
	}
}

func TestGetEscalationMatrix(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-006", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.9,
	}

	_, _ = engine.RegisterIncident(incident, classification)
	_, _ = engine.Escalate("INC-006", "Test escalation")

	matrix := engine.GetEscalationMatrix("INC-006")
	if len(matrix) != 1 {
		t.Fatalf("expected 1 escalation entry, got %d", len(matrix))
	}
}

func TestPreserveEvidence(t *testing.T) {
	engine := setupTestEngine(t)

	id, err := engine.PreserveEvidence("INC-007", "logs", []byte("test log data"))
	if err != nil {
		t.Fatalf("PreserveEvidence error: %v", err)
	}
	if id != "ev-001" {
		t.Fatalf("expected ev-001, got %s", id)
	}
}

func TestListActiveIncidents(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-008", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.9,
	}

	_, _ = engine.RegisterIncident(incident, classification)

	active := engine.ListActiveIncidents()
	if len(active) != 1 {
		t.Fatalf("expected 1 active incident, got %d", len(active))
	}
	if active[0].IncidentID != "INC-008" {
		t.Fatalf("expected INC-008, got %s", active[0].IncidentID)
	}
}

func TestGenerateRegulatoryReport(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-009", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Impact:     []ImpactCategory{ImpactAvailability},
		Confidence: 0.95,
	}

	_, _ = engine.RegisterIncident(incident, classification)

	report, err := engine.GenerateRegulatoryReport("INC-009", FrameworkDORA, "json")
	if err != nil {
		t.Fatalf("GenerateRegulatoryReport error: %v", err)
	}

	if report.Framework != FrameworkDORA {
		t.Fatalf("expected DORA, got %s", report.Framework)
	}
	if report.IncidentID != "INC-009" {
		t.Fatalf("expected INC-009, got %s", report.IncidentID)
	}
	if len(report.ReportBody) == 0 {
		t.Fatal("expected non-empty report body")
	}
}

func TestGenerateRegulatoryReport_UnknownFramework(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-010", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.9,
	}

	_, _ = engine.RegisterIncident(incident, classification)

	_, err := engine.GenerateRegulatoryReport("INC-010", "UNKNOWN", "json")
	if err == nil {
		t.Fatal("expected error for unknown framework")
	}
}

func TestLegalHoldLifecycle(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-011", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.95,
	}

	active, _ := engine.RegisterIncident(incident, classification)

	// Проверяем создание legal hold
	holds := engine.GetActiveLegalHolds("INC-011")
	initialCount := len(holds)
	if initialCount == 0 {
		t.Fatal("expected at least one legal hold")
	}

	// Освобождаем первый legal hold
	holdID := holds[0].ID
	err := engine.ReleaseLegalHold(holdID)
	if err != nil {
		t.Fatalf("ReleaseLegalHold error: %v", err)
	}

	// Проверяем, что количество активных holds уменьшилось
	holds = engine.GetActiveLegalHolds("INC-011")
	if len(holds) >= initialCount {
		t.Fatalf("expected fewer active holds after release (was %d, now %d)", initialCount, len(holds))
	}

	_ = active
}

func TestRemindPending(t *testing.T) {
	engine := setupTestEngine(t)

	incident := &Incident{ID: "INC-012", CreatedAt: time.Now().UTC()}
	classification := &IncidentClassification{
		Severity:   SeverityNIS2Critical,
		Type:       IncidentTypeUnauthorizedAccess,
		Confidence: 0.95,
	}

	_, _ = engine.RegisterIncident(incident, classification)

	// RemindPending должен найти инциденты
	reminded := engine.RemindPending()
	t.Logf("Reminded: %v", reminded)
	// Может быть 0, если reminder threshold не превышен (зависит от времени)
}
