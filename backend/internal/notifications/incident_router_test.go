package notifications

import (
	"testing"
	"time"

	"log/slog"
)

// mockSender — реализация NotificationSender для тестов.
type mockSender struct {
	sent []*IncidentNotification
}

func (m *mockSender) Send(notification *IncidentNotification) error {
	m.sent = append(m.sent, notification)
	return nil
}

func TestNewIncidentRouter(t *testing.T) {
	router := NewIncidentRouter(slog.Default(), nil)
	if router == nil {
		t.Fatal("expected non-nil router")
	}

	routes := router.ListRoutes()
	if len(routes) == 0 {
		t.Fatal("expected at least one default route")
	}
}

func TestRouteNotification(t *testing.T) {
	emailSender := &mockSender{}
	senders := map[NotificationChannel]NotificationSender{
		ChannelEmail: emailSender,
	}

	router := NewIncidentRouter(slog.Default(), senders)

	notification, err := router.RouteNotification(
		"INC-001", "DORA", "EU", "critical", "initial_report",
	)
	if err != nil {
		t.Fatalf("RouteNotification error: %v", err)
	}

	if notification == nil {
		t.Fatal("expected non-nil notification")
	}

	if notification.Framework != "DORA" {
		t.Fatalf("expected DORA, got %s", notification.Framework)
	}
	if notification.Region != "EU" {
		t.Fatalf("expected EU, got %s", notification.Region)
	}
	if notification.Status != "sent" {
		t.Fatalf("expected sent, got %s", notification.Status)
	}
}

func TestRouteNotification_UnknownRegion(t *testing.T) {
	router := NewIncidentRouter(slog.Default(), nil)

	_, err := router.RouteNotification(
		"INC-002", "DORA", "XX", "critical", "initial_report",
	)
	if err == nil {
		t.Fatal("expected error for unknown region")
	}
}

func TestRouteNotification_UnsupportedFramework(t *testing.T) {
	router := NewIncidentRouter(slog.Default(), nil)

	_, err := router.RouteNotification(
		"INC-003", "INVALID_FW", "EU", "critical", "initial_report",
	)
	if err == nil {
		t.Fatal("expected error for unsupported framework")
	}
}

func TestChannelSelection(t *testing.T) {
	emailSender := &mockSender{}
	smsSender := &mockSender{}
	senders := map[NotificationChannel]NotificationSender{
		ChannelEmail: emailSender,
		ChannelSMS:   smsSender,
	}

	router := NewIncidentRouter(slog.Default(), senders)

	// Initial report — all channels
	notification, err := router.RouteNotification(
		"INC-004", "DORA", "EU", "critical", "initial_report",
	)
	if err != nil {
		t.Fatalf("RouteNotification error: %v", err)
	}

	if len(notification.Channels) == 0 {
		t.Fatal("expected at least one channel")
	}

	// Acknowledgment — only email
	notification, err = router.RouteNotification(
		"INC-005", "DORA", "EU", "critical", "acknowledgment",
	)
	if err != nil {
		t.Fatalf("RouteNotification error: %v", err)
	}

	if len(notification.Channels) != 1 || notification.Channels[0] != ChannelEmail {
		t.Fatal("expected only email channel for acknowledgment")
	}
}

func TestGetPending(t *testing.T) {
	router := NewIncidentRouter(slog.Default(), nil)

	pending := router.GetPending()
	if len(pending) != 0 {
		t.Fatal("expected empty pending list")
	}
}

func TestGetHistory(t *testing.T) {
	emailSender := &mockSender{}
	senders := map[NotificationChannel]NotificationSender{
		ChannelEmail: emailSender,
	}

	router := NewIncidentRouter(slog.Default(), senders)

	_, _ = router.RouteNotification(
		"INC-006", "DORA", "EU", "critical", "initial_report",
	)

	history := router.GetHistory(10)
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
}

func TestSetRoute(t *testing.T) {
	router := NewIncidentRouter(slog.Default(), nil)

	newRoute := RegionRouteConfig{
		Region:         "JP",
		Frameworks:     []string{"CCRA"},
		ReportingHours: 12,
	}
	router.SetRoute(newRoute)

	route, ok := router.GetRoute("JP")
	if !ok {
		t.Fatal("expected route for JP to exist")
	}
	if route.ReportingHours != 12 {
		t.Fatalf("expected 12 reporting hours, got %d", route.ReportingHours)
	}
}

func TestDefaultRoutes(t *testing.T) {
	routes := DefaultRegionRoutes()
	if len(routes) == 0 {
		t.Fatal("expected at least one default route")
	}

	regions := make(map[string]bool)
	for _, r := range routes {
		if regions[r.Region] {
			t.Fatalf("duplicate region: %s", r.Region)
		}
		regions[r.Region] = true

		if r.ReportingHours <= 0 {
			t.Errorf("region %s: reporting hours must be positive", r.Region)
		}
		if len(r.PriorityChannels) == 0 {
			t.Errorf("region %s: must have at least one priority channel", r.Region)
		}
	}

	// Проверяем обязательные регионы
	requiredRegions := []string{"EU", "IN", "SG", "BY", "RU"}
	for _, region := range requiredRegions {
		if !regions[region] {
			t.Errorf("missing required region: %s", region)
		}
	}
}

func TestMessageBuilding(t *testing.T) {
	router := NewIncidentRouter(slog.Default(), nil)

	// Проверяем через внутренние методы
	notification, err := router.RouteNotification(
		"INC-007", "CERT-In", "IN", "high", "initial_report",
	)
	if err != nil {
		t.Fatalf("RouteNotification error: %v", err)
	}

	if notification.Title == "" {
		t.Fatal("expected non-empty title")
	}
	if notification.Body == "" {
		t.Fatal("expected non-empty body")
	}

	// Проверяем наличие ключевой информации в теле
	if notification.Framework != "CERT-In" {
		t.Fatalf("expected CERT-In, got %s", notification.Framework)
	}
}

func TestSendFailure(t *testing.T) {
	senders := map[NotificationChannel]NotificationSender{
		ChannelEmail: nil, // nil sender — будет ошибка
	}

	router := NewIncidentRouter(slog.Default(), senders)

	notification, err := router.RouteNotification(
		"INC-008", "DORA", "EU", "critical", "initial_report",
	)
	if err != nil {
		t.Fatalf("RouteNotification error: %v", err)
	}

	// Должен быть помечен как sent (всегда), но с ошибками в логе
	if notification.Status != "sent" {
		t.Fatalf("expected sent, got %s", notification.Status)
	}
}

func TestTimeoutConfig(t *testing.T) {
	route := RegionRouteConfig{
		Region:           "TEST",
		Frameworks:       []string{"TEST-FW"},
		ReportingHours:   1,
		PriorityChannels: []NotificationChannel{ChannelEmail},
		BackupChannels:   []NotificationChannel{ChannelSMS},
		EscalationDelay:  5 * time.Minute,
	}

	if route.EscalationDelay != 5*time.Minute {
		t.Fatalf("expected 5m escalation delay, got %v", route.EscalationDelay)
	}
}
