// Package sms — RocketSMS Provider Tests.
//
// P0-1.4: Тесты для SMS провайдера с rate limiting и delivery tracking.
//
// Compliance:
//   - OWASP ASVS V7.1 (Log content — номера маскируются)
//   - ISO 27001 A.12.4.1 (Event logging — delivery tracking)
//   - Приказ ОАЦ №66 п.7.18.6 (Anti-spam — rate limiting)
package sms

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Config & Initialization Tests
// ═══════════════════════════════════════════════════════════════════════

func TestNewRocketSMSProvider_Defaults(t *testing.T) {
	cfg := RocketSMSConfig{
		Login:    "test-login",
		Password: "test-pass",
	}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	if p.cfg.APIURL != "https://api.rocketsms.by" {
		t.Errorf("expected default api url, got %s", p.cfg.APIURL)
	}
	if p.cfg.Sender != "CCTV" {
		t.Errorf("expected default sender CCTV, got %s", p.cfg.Sender)
	}
	if p.rateLimit != DefaultSMSRateLimit {
		t.Errorf("expected default rate limit %d, got %d", DefaultSMSRateLimit, p.rateLimit)
	}
	if p.rateWindow != DefaultSMSRateWindow {
		t.Errorf("expected default rate window %v, got %v", DefaultSMSRateWindow, p.rateWindow)
	}
}

func TestNewRocketSMSProvider_CustomConfig(t *testing.T) {
	cfg := RocketSMSConfig{
		Login:    "custom-login",
		Password: "custom-pass",
		Sender:   "ALERT",
		APIURL:   "https://custom.api.rocketsms.by",
	}
	p := NewRocketSMSProvider(cfg, 5, 30*time.Second, nil)

	if p.cfg.Sender != "ALERT" {
		t.Errorf("expected custom sender ALERT, got %s", p.cfg.Sender)
	}
	if p.rateLimit != 5 {
		t.Errorf("expected custom rate limit 5, got %d", p.rateLimit)
	}
	if p.rateWindow != 30*time.Second {
		t.Errorf("expected custom rate window 30s, got %v", p.rateWindow)
	}
}

func TestNewRocketSMSProvider_NilLogger(t *testing.T) {
	cfg := RocketSMSConfig{Login: "test", Password: "test"}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)
	if p.logger == nil {
		t.Fatal("expected default logger when nil provided")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// IsAvailable Tests
// ═══════════════════════════════════════════════════════════════════════

func TestIsAvailable_Configured(t *testing.T) {
	cfg := RocketSMSConfig{Login: "test-login", Password: "test-pass"}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	if !p.IsAvailable() {
		t.Error("expected provider to be available with login/password")
	}
}

func TestIsAvailable_NotConfigured(t *testing.T) {
	cfg := RocketSMSConfig{}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	if p.IsAvailable() {
		t.Error("expected provider to NOT be available without login/password")
	}
}

func TestIsAvailable_OnlyLogin(t *testing.T) {
	cfg := RocketSMSConfig{Login: "test-login"}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	if p.IsAvailable() {
		t.Error("expected provider to NOT be available without password")
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SendSMS Tests (без реального HTTP)
// ═══════════════════════════════════════════════════════════════════════

func TestSendSMS_NotConfigured(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := RocketSMSConfig{}
	p := NewRocketSMSProvider(cfg, 0, 0, logger)

	err := p.SendSMS(context.Background(), "+375291234567", "test message")
	if err == nil {
		t.Fatal("expected error when provider not configured")
	}

	// Проверяем delivery tracking
	metrics := p.DeliveryMetrics()
	if metrics.Failed != 1 {
		t.Errorf("expected 1 failed delivery, got %d", metrics.Failed)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Rate Limiting Tests (P0-1.4)
// ═══════════════════════════════════════════════════════════════════════

func TestRateLimit_AllowsWithinLimit(t *testing.T) {
	p := &RocketSMSProvider{
		rateLimit:   5,
		rateWindow:  time.Minute,
		rateEntries: make(map[string][]time.Time),
	}

	// Должны пройти 5 SMS
	for i := 0; i < 5; i++ {
		if !p.allow("+375291234567") {
			t.Errorf("expected SMS %d to be allowed", i+1)
		}
	}
}

func TestRateLimit_BlocksExcess(t *testing.T) {
	p := &RocketSMSProvider{
		rateLimit:   3,
		rateWindow:  time.Minute,
		rateEntries: make(map[string][]time.Time),
	}

	// Первые 3 должны пройти
	for i := 0; i < 3; i++ {
		if !p.allow("+375291234567") {
			t.Errorf("expected SMS %d to be allowed", i+1)
		}
	}

	// 4-я должна быть заблокирована
	if p.allow("+375291234567") {
		t.Error("expected 4th SMS to be rate limited")
	}
}

func TestRateLimit_DifferentPhones(t *testing.T) {
	p := &RocketSMSProvider{
		rateLimit:   2,
		rateWindow:  time.Minute,
		rateEntries: make(map[string][]time.Time),
	}

	// 2 SMS на один номер
	p.allow("+375291111111")
	p.allow("+375291111111")
	if p.allow("+375291111111") {
		t.Error("expected 3rd SMS to phone1 to be rate limited")
	}

	// Другой номер не должен быть затронут
	if !p.allow("+375292222222") {
		t.Error("expected SMS to phone2 to be allowed")
	}
}

func TestRateLimit_WindowExpiration(t *testing.T) {
	p := &RocketSMSProvider{
		rateLimit:   1,
		rateWindow:  50 * time.Millisecond,
		rateEntries: make(map[string][]time.Time),
	}

	// Первая SMS проходит
	if !p.allow("+375291234567") {
		t.Error("expected 1st SMS to be allowed")
	}

	// Вторая блокируется
	if p.allow("+375291234567") {
		t.Error("expected 2nd SMS to be rate limited")
	}

	// Ждём истечения окна
	time.Sleep(60 * time.Millisecond)

	// Третья должна пройти
	if !p.allow("+375291234567") {
		t.Error("expected SMS after window expiration to be allowed")
	}
}

func TestRateLimit_SendSMSWithRateLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := RocketSMSConfig{Login: "test-login", Password: "test-pass"}
	p := NewRocketSMSProvider(cfg, 2, time.Minute, logger)

	// Провайдер настроен, но HTTP запрос не может быть выполнен
	// Rate limiting применяется ДО HTTP запроса
	// Должен вернуть rate limit error после превышения лимита

	// Первый SendSMS — должен упасть на HTTP (not actual API), не на rate limit
	err1 := p.SendSMS(context.Background(), "+375291234567", "test 1")
	if err1 == nil {
		t.Fatal("expected error (no real API)")
	}
	_ = err1

	// Второй — тоже упадёт на HTTP
	err2 := p.SendSMS(context.Background(), "+375291234567", "test 2")
	if err2 == nil {
		t.Fatal("expected error (no real API)")
	}
	_ = err2

	// Третий — должен быть rate limited (2 SMS в минуту)
	err3 := p.SendSMS(context.Background(), "+375291234567", "test 3")
	if err3 == nil {
		t.Fatal("expected rate limit error")
	}

	metrics := p.DeliveryMetrics()
	if metrics.RateLimited != 1 {
		t.Errorf("expected 1 rate limited, got %d", metrics.RateLimited)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Delivery Metrics Tests (P0-1.4)
// ═══════════════════════════════════════════════════════════════════════

func TestDeliveryMetrics_Initial(t *testing.T) {
	cfg := RocketSMSConfig{Login: "test", Password: "test"}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	metrics := p.DeliveryMetrics()
	if metrics.Sent != 0 {
		t.Errorf("expected 0 sent, got %d", metrics.Sent)
	}
	if metrics.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", metrics.Failed)
	}
	if metrics.RateLimited != 0 {
		t.Errorf("expected 0 rate limited, got %d", metrics.RateLimited)
	}
	if metrics.TotalCost != 0 {
		t.Errorf("expected 0 total cost, got %d", metrics.TotalCost)
	}
}

func TestDeliveryMetrics_Reset(t *testing.T) {
	cfg := RocketSMSConfig{Login: "test", Password: "test"}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	// Симулируем failed
	p.SendSMS(context.Background(), "+375291234567", "test")
	metrics := p.DeliveryMetrics()
	if metrics.Failed == 0 {
		t.Error("expected at least 1 failed after SendSMS")
	}

	p.ResetMetrics()
	metrics = p.DeliveryMetrics()
	if metrics.Failed != 0 {
		t.Errorf("expected 0 failed after reset, got %d", metrics.Failed)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// GetConfig Tests
// ═══════════════════════════════════════════════════════════════════════

func TestGetConfig_NoPasswordLeak(t *testing.T) {
	cfg := RocketSMSConfig{
		Login:    "test-login",
		Password: "supersecret",
		Sender:   "CCTV",
		APIURL:   "https://api.rocketsms.by",
	}
	p := NewRocketSMSProvider(cfg, 0, 0, nil)

	config := p.GetConfig()
	if config.Password != "" {
		t.Error("expected password to be empty in GetConfig (security)")
	}
	if config.Login != "test-login" {
		t.Errorf("expected login to be preserved, got %s", config.Login)
	}
}
