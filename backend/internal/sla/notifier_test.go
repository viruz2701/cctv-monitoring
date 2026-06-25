// Package sla — SLA Breach Notifier tests (NOTIF-01).
package sla

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Mocks
// ═══════════════════════════════════════════════════════════════════════

type mockTelegramSender struct {
	mu       sync.Mutex
	messages []struct {
		chatID int64
		text   string
	}
}

func (m *mockTelegramSender) SendTextMessage(chatID int64, text string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, struct {
		chatID int64
		text   string
	}{chatID, text})
}

func (m *mockTelegramSender) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

type mockSMSProvider struct {
	mu        sync.Mutex
	sent      []string
	available bool
}

func (m *mockSMSProvider) SendSMS(_ context.Context, phoneNumber, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, phoneNumber+":"+message)
	return nil
}

func (m *mockSMSProvider) IsAvailable() bool {
	return m.available
}

type mockEmailSender struct {
	mu        sync.Mutex
	sent      []struct{ to, subject, body string }
	available bool
}

func (m *mockEmailSender) SendEmail(_ context.Context, to, subject, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, struct{ to, subject, body string }{to, subject, body})
	return nil
}

func (m *mockEmailSender) IsAvailable() bool {
	return m.available
}

type mockContactProvider struct {
	contacts map[string]*UserContactInfo
	managers map[string]*UserContactInfo
}

func (m *mockContactProvider) GetContactInfo(_ context.Context, userID string) (*UserContactInfo, error) {
	c, ok := m.contacts[userID]
	if !ok {
		return &UserContactInfo{UserID: userID}, nil
	}
	return c, nil
}

func (m *mockContactProvider) FindManagerForDevice(_ context.Context, _ string) (*UserContactInfo, error) {
	for _, mgr := range m.managers {
		return mgr, nil
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════
// Tests
// ═══════════════════════════════════════════════════════════════════════

func TestNewSLABreachNotifier_NilLogger(t *testing.T) {
	n := NewSLABreachNotifier(nil, nil, nil, nil, nil)
	if n == nil {
		t.Fatal("expected non-nil notifier")
	}
	if n.logger == nil {
		t.Fatal("expected default logger")
	}
}

func TestNotifyAtRisk_NoTelegram(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	n := NewSLABreachNotifier(nil, nil, nil, &mockContactProvider{}, logger)

	wo := BreachedWorkOrder{
		ID:       "wo-1",
		Title:    "Test Work Order",
		Priority: "HIGH",
		DeviceID: "dev-1",
	}

	err := n.NotifyAtRisk(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotifyAtRisk_SendsTelegram(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	telegram := &mockTelegramSender{}
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {
				UserID:         "tech-1",
				TelegramChatID: "123456789",
			},
		},
	}

	n := NewSLABreachNotifier(telegram, nil, nil, contacts, logger)

	wo := BreachedWorkOrder{
		ID:         "wo-1",
		Title:      "Camera #45 Repair",
		Priority:   "HIGH",
		DeviceName: "Camera IP #45",
		AssignedTo: "tech-1",
		SLADeadline: time.Now().Add(30 * time.Minute),
	}

	err := n.NotifyAtRisk(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if telegram.count() != 1 {
		t.Fatalf("expected 1 telegram message, got %d", telegram.count())
	}
}

func TestNotifyAtRisk_NoAssignee(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	telegram := &mockTelegramSender{}

	n := NewSLABreachNotifier(telegram, nil, nil, &mockContactProvider{}, logger)

	wo := BreachedWorkOrder{
		ID:       "wo-1",
		Title:    "Test",
		Priority: "MEDIUM",
	}

	err := n.NotifyAtRisk(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if telegram.count() != 0 {
		t.Fatalf("expected 0 messages for unassigned WO, got %d", telegram.count())
	}
}

func TestNotifyCritical_SendsTelegramAndEmail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	telegram := &mockTelegramSender{}
	email := &mockEmailSender{available: true}
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {
				UserID:         "tech-1",
				TelegramChatID: "123456789",
			},
		},
		managers: map[string]*UserContactInfo{
			"mgr-1": {
				UserID: "mgr-1",
				Email:  "manager@example.com",
			},
		},
	}

	n := NewSLABreachNotifier(telegram, nil, email, contacts, logger)

	wo := BreachedWorkOrder{
		ID:           "wo-1",
		Title:        "Critical Camera Repair",
		Priority:     "CRITICAL",
		DeviceName:   "Camera #1",
		DeviceID:     "dev-1",
		AssignedTo:   "tech-1",
		AssigneeName: "Ivan Petrov",
		SLADeadline:  time.Now().Add(10 * time.Minute),
	}

	err := n.NotifyCritical(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if telegram.count() != 1 {
		t.Fatalf("expected 1 telegram message, got %d", telegram.count())
	}

	email.mu.Lock()
	emailCount := len(email.sent)
	email.mu.Unlock()
	if emailCount != 1 {
		t.Fatalf("expected 1 email, got %d", emailCount)
	}
}

func TestNotifyBreach_SendsAllChannels(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	telegram := &mockTelegramSender{}
	sms := &mockSMSProvider{available: true}
	email := &mockEmailSender{available: true}
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {
				UserID:         "tech-1",
				TelegramChatID: "123456789",
			},
		},
		managers: map[string]*UserContactInfo{
			"mgr-1": {
				UserID:      "mgr-1",
				Email:       "manager@example.com",
				PhoneNumber: "+375291234567",
			},
		},
	}

	n := NewSLABreachNotifier(telegram, sms, email, contacts, logger)

	wo := BreachedWorkOrder{
		ID:           "wo-1",
		Title:        "NVR #12 Failure",
		Priority:     "CRITICAL",
		DeviceName:   "NVR #12",
		DeviceID:     "dev-nvr-12",
		AssignedTo:   "tech-1",
		AssigneeName: "Ivan Petrov",
		SLADeadline:  time.Now().Add(-1 * time.Hour),
	}

	err := n.NotifyBreach(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if telegram.count() != 1 {
		t.Fatalf("expected 1 telegram message, got %d", telegram.count())
	}

	sms.mu.Lock()
	smsCount := len(sms.sent)
	sms.mu.Unlock()
	if smsCount != 1 {
		t.Fatalf("expected 1 SMS, got %d", smsCount)
	}

	email.mu.Lock()
	emailCount := len(email.sent)
	email.mu.Unlock()
	if emailCount != 1 {
		t.Fatalf("expected 1 email, got %d", emailCount)
	}
}

func TestNotifyBreach_LowPriority_NoSMS(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	telegram := &mockTelegramSender{}
	sms := &mockSMSProvider{available: true}
	email := &mockEmailSender{available: true}
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {
				UserID:         "tech-1",
				TelegramChatID: "123456789",
			},
		},
		managers: map[string]*UserContactInfo{
			"mgr-1": {
				UserID:      "mgr-1",
				Email:       "manager@example.com",
				PhoneNumber: "+375291234567",
			},
		},
	}

	n := NewSLABreachNotifier(telegram, sms, email, contacts, logger)

	wo := BreachedWorkOrder{
		ID:           "wo-1",
		Title:        "Low Priority Task",
		Priority:     "LOW",
		DeviceName:   "Camera #1",
		DeviceID:     "dev-1",
		AssignedTo:   "tech-1",
		AssigneeName: "Ivan Petrov",
		SLADeadline:  time.Now().Add(-30 * time.Minute),
	}

	err := n.NotifyBreach(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SMS только для CRITICAL/HIGH
	sms.mu.Lock()
	smsCount := len(sms.sent)
	sms.mu.Unlock()
	if smsCount != 0 {
		t.Fatalf("expected 0 SMS for LOW priority, got %d", smsCount)
	}

	// Email должен быть отправлен
	email.mu.Lock()
	emailCount := len(email.sent)
	email.mu.Unlock()
	if emailCount != 1 {
		t.Fatalf("expected 1 email for LOW priority, got %d", emailCount)
	}
}

func TestNotifyBreach_NoManager_NoSMSNoEmail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	telegram := &mockTelegramSender{}
	sms := &mockSMSProvider{available: true}
	email := &mockEmailSender{available: true}
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {
				UserID:         "tech-1",
				TelegramChatID: "123456789",
			},
		},
		// No managers configured
		managers: map[string]*UserContactInfo{},
	}

	n := NewSLABreachNotifier(telegram, sms, email, contacts, logger)

	wo := BreachedWorkOrder{
		ID:         "wo-1",
		Title:      "Test",
		Priority:   "CRITICAL",
		DeviceID:   "dev-1",
		AssignedTo: "tech-1",
		SLADeadline: time.Now().Add(-1 * time.Hour),
	}

	err := n.NotifyBreach(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Telegram должен быть отправлен
	if telegram.count() != 1 {
		t.Fatalf("expected 1 telegram, got %d", telegram.count())
	}

	// SMS и Email не должны быть отправлены (нет менеджера)
	sms.mu.Lock()
	smsCount := len(sms.sent)
	sms.mu.Unlock()
	if smsCount != 0 {
		t.Fatalf("expected 0 SMS without manager, got %d", smsCount)
	}

	email.mu.Lock()
	emailCount := len(email.sent)
	email.mu.Unlock()
	if emailCount != 0 {
		t.Fatalf("expected 0 email without manager, got %d", emailCount)
	}
}

func TestFormatDuration_Hours(t *testing.T) {
	d := 2*time.Hour + 30*time.Minute
	result := formatDuration(d)
	expected := "2ч 30мин"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestFormatDuration_Minutes(t *testing.T) {
	d := 45 * time.Minute
	result := formatDuration(d)
	expected := "45 мин"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"+375291234567", "+375***67"},
		{"12345", "123***45"},
		{"12", "***"},
	}

	for _, tt := range tests {
		result := maskPhone(tt.input)
		if result != tt.expected {
			t.Errorf("maskPhone(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ivan@example.com", "iv***@example.com"},
		{"a@b.com", "***@***"},
		{"ab@c.d", "ab***@c.d"},
	}

	for _, tt := range tests {
		result := maskEmail(tt.input)
		if result != tt.expected {
			t.Errorf("maskEmail(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
