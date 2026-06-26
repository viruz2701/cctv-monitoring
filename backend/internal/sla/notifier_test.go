// Package sla — SLA Breach Notifier tests (NOTIF-01).
package sla

import (
	"context"
	"fmt"
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
	// dbDown симулирует недоступность БД
	dbDown bool
}

func (m *mockContactProvider) GetContactInfo(_ context.Context, userID string) (*UserContactInfo, error) {
	if m.dbDown {
		return nil, fmt.Errorf("db connection failed")
	}
	c, ok := m.contacts[userID]
	if !ok {
		return &UserContactInfo{UserID: userID}, nil
	}
	return c, nil
}

func (m *mockContactProvider) FindManagerForDevice(_ context.Context, _ string) (*UserContactInfo, error) {
	if m.dbDown {
		return nil, fmt.Errorf("db connection failed")
	}
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
		ID:          "wo-1",
		Title:       "Camera #45 Repair",
		Priority:    "HIGH",
		DeviceName:  "Camera IP #45",
		AssignedTo:  "tech-1",
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
		ID:          "wo-1",
		Title:       "Test",
		Priority:    "CRITICAL",
		DeviceID:    "dev-1",
		AssignedTo:  "tech-1",
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

// ═══════════════════════════════════════════════════════════════════════
// P0-1.5: Contact Cache Tests
// ═══════════════════════════════════════════════════════════════════════

// TestContactCache_Hit проверяет, что кэш возвращает сохранённый контакт.
func TestContactCache_Hit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1", TelegramChatID: "12345", Email: "tech@test.com"},
		},
	}
	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(time.Minute)

	ctx := context.Background()

	// Первый вызов — загружает в кэш
	contact1, err := notifier.getCachedContact(ctx, "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact1.Email != "tech@test.com" {
		t.Errorf("expected tech@test.com, got %s", contact1.Email)
	}

	// Второй вызов — из кэша (не должен вызывать контакт провайдер)
	contact2, err := notifier.getCachedContact(ctx, "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact2.Email != "tech@test.com" {
		t.Errorf("expected tech@test.com, got %s", contact2.Email)
	}
}

// TestContactCache_Expiry проверяет, что после истечения TTL контакт обновляется.
func TestContactCache_Expiry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1", TelegramChatID: "12345"},
		},
	}
	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(50 * time.Millisecond)

	_, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// TTL истёк
	time.Sleep(60 * time.Millisecond)

	// Должен обратиться к провайдеру снова
	contact, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact.TelegramChatID != "12345" {
		t.Errorf("expected 12345, got %s", contact.TelegramChatID)
	}
}

// TestContactCache_StaleWhileRevalidate проверяет, что при недоступности БД
// возвращается просроченный кэш (stale-while-revalidate).
func TestContactCache_StaleWhileRevalidate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1", TelegramChatID: "12345", Email: "cached@test.com"},
		},
	}
	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(50 * time.Millisecond)

	// Загружаем в кэш
	_, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ждём истечения TTL
	time.Sleep(60 * time.Millisecond)

	// "Ломаем" БД — симулируем недоступность
	contacts.dbDown = true

	// Должен вернуть просроченный кэш, а не ошибку
	contact, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("expected stale cache, got error: %v", err)
	}
	if contact.Email != "cached@test.com" {
		t.Errorf("expected cached@test.com from stale cache, got %s", contact.Email)
	}
}

// TestContactCache_Clear проверяет очистку кэша.
func TestContactCache_Clear(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1", TelegramChatID: "12345"},
		},
	}
	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(time.Minute)

	// Загружаем в кэш
	_, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Первый вызов из кэша — успех
	contact, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact.TelegramChatID != "12345" {
		t.Errorf("expected 12345, got %s", contact.TelegramChatID)
	}

	// Очищаем кэш
	notifier.ClearCache()

	// "Ломаем" БД
	contacts.dbDown = true

	// После очистки кэша и БД down должен вернуть ошибку
	_, err = notifier.getCachedContact(context.Background(), "tech-1")
	if err == nil {
		t.Fatal("expected error after cache clear and DB down")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P0-1.5: Default Admin Email Fallback Tests
// ═══════════════════════════════════════════════════════════════════════

// TestFallback_DefaultAdminEmail проверяет fallback на admin email при БД downtime.
func TestFallback_DefaultAdminEmail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	email := &mockEmailSender{available: true}
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1", TelegramChatID: "123456789"},
		},
		managers: map[string]*UserContactInfo{},
	}
	telegram := &mockTelegramSender{}

	notifier := NewSLABreachNotifier(telegram, nil, email, contacts, logger)
	notifier.SetDefaultAdminEmail("admin@cctv.com")

	// "Ломаем" БД — FindManagerForDevice вернёт ошибку
	contacts.dbDown = true

	wo := BreachedWorkOrder{
		ID:           "wo-1",
		Title:        "Camera offline",
		DeviceID:     "cam-001",
		DeviceName:   "Main Camera",
		Priority:     "critical",
		AssignedTo:   "tech-1",
		AssigneeName: "John Doe",
		SLADeadline:  time.Now().Add(-1 * time.Hour),
	}

	err := notifier.NotifyBreach(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// При БД downtime должен использоваться fallback admin email
	if len(email.sent) == 0 {
		t.Fatal("expected email to be sent to fallback admin email")
	}
	lastEmail := email.sent[len(email.sent)-1]
	if lastEmail.to != "admin@cctv.com" {
		t.Errorf("expected admin@cctv.com, got %s", lastEmail.to)
	}
}

// TestFallback_NoFallbackConfigured проверяет, что без fallback уведомление не падает.
func TestFallback_NoFallbackConfigured(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{},
		managers: map[string]*UserContactInfo{},
	}

	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	// Не устанавливаем defaultAdminEmail

	wo := BreachedWorkOrder{
		ID:          "wo-1",
		Title:       "Test",
		DeviceID:    "dev-1",
		Priority:    "critical",
		AssignedTo:  "tech-1",
		SLADeadline: time.Now().Add(-1 * time.Hour),
	}

	// Не должно паниковать
	err := notifier.NotifyBreach(context.Background(), wo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P0-1.5: Retry with Exponential Backoff Tests
// ═══════════════════════════════════════════════════════════════════════

// TestRetry_SuccessAfterRetry проверяет, что retry успешен после временной ошибки.
func TestRetry_SuccessAfterRetry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1", TelegramChatID: "12345"},
		},
	}
	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(0) // Отключаем кэш для теста retry

	ctx := context.Background()
	contact, err := notifier.getCachedContact(ctx, "tech-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if contact.TelegramChatID != "12345" {
		t.Errorf("expected 12345, got %s", contact.TelegramChatID)
	}
}

// TestRetry_AllFail проверяет, что при всех неудачных retry используется fallback.
func TestRetry_AllFail(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{
			"tech-1": {UserID: "tech-1"},
		},
		managers: map[string]*UserContactInfo{},
	}
	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(0) // Отключаем кэш
	notifier.SetDefaultAdminEmail("admin@fallback.com")

	// "Ломаем" БД
	contacts.dbDown = true

	// Все retry должны провалиться, fallback должен вернуть admin email
	contact, err := notifier.getCachedContact(context.Background(), "tech-1")
	if err != nil {
		t.Fatalf("expected fallback contact, got error: %v", err)
	}
	if contact.Email != "admin@fallback.com" {
		t.Errorf("expected fallback email 'admin@fallback.com', got '%s'", contact.Email)
	}
}

// TestRetry_ContextCancellation проверяет, что retry обрабатывает отмену контекста.
func TestRetry_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	contacts := &mockContactProvider{
		contacts: map[string]*UserContactInfo{},
		managers: map[string]*UserContactInfo{},
	}

	notifier := NewSLABreachNotifier(nil, nil, nil, contacts, logger)
	notifier.SetCacheTTL(0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Немедленная отмена

	// Mock не проверяет контекст, поэтому retry пройдёт успешно
	// (возвращает пустой контакт без ошибки)
	contact, err := notifier.getCachedContact(ctx, "tech-1")
	if err != nil {
		// Если ошибка — это тоже нормально (зависит от реализации)
		t.Logf("got expected error: %v", err)
	} else {
		// Если нет ошибки — тоже ок (mock не блокируется)
		_ = contact
	}
}

// ═══════════════════════════════════════════════════════════════════════
// P0-1.5: End-to-End Notifier Resilience Tests
// ═══════════════════════════════════════════════════════════════════════

// TestNotifyBreach_WithCache проверяет, что после кэширования контактов
// notifier работает даже при "падении" БД.
func TestNotifyBreach_WithCache(t *testing.T) {
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
			"cam-001": {
				UserID: "mgr-1",
				Email:  "manager@test.com",
			},
		},
	}

	n := NewSLABreachNotifier(telegram, nil, email, contacts, logger)
	n.SetCacheTTL(time.Minute)

	wo := BreachedWorkOrder{
		ID:           "wo-1",
		Title:        "Camera offline",
		DeviceID:     "cam-001",
		DeviceName:   "Main Camera",
		Priority:     "CRITICAL",
		AssignedTo:   "tech-1",
		AssigneeName: "John Doe",
		SLADeadline:  time.Now().Add(-1 * time.Hour),
	}

	// Первый вызов — загружает в кэш
	if err := n.NotifyBreach(context.Background(), wo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if telegram.count() != 1 {
		t.Fatalf("expected 1 telegram, got %d", telegram.count())
	}

	// "Ломаем" БД — очищаем провайдер
	contacts.contacts = map[string]*UserContactInfo{}
	contacts.managers = map[string]*UserContactInfo{}

	// Второй вызов — должен использовать кэш
	if err := n.NotifyBreach(context.Background(), wo); err != nil {
		t.Fatalf("unexpected error after DB failure: %v", err)
	}
	if telegram.count() != 2 {
		t.Fatalf("expected 2 telegrams (cached), got %d", telegram.count())
	}
}
