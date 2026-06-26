// Package sms — SMS providers для уведомлений.
//
// NOTIF-01: Поддержка SMS-шлюзов для отправки критических уведомлений.
// Текущий провайдер: RocketSMS (by kartasoft) — СМС-центр РБ (compliance).
//
// API RocketSMS: https://rocketsms.by/api
// Метод: POST /api/send
// Параметры: login, password, phone, text, sender
//
// P0-1.4: Добавлены:
//   - Rate limiting (anti-spam) — не более 10 SMS в минуту на номер
//   - Delivery tracking — счётчики sent/failed/rate_limited
//   - Email fallback — при недоступности SMS (в SLABreachNotifier)
//
// Compliance:
//   - OWASP ASVS V7.1 (Log content — номера маскируются в логах)
//   - OWASP ASVS V8 (Data Protection — credentials в env, не в коде)
//   - ISO 27001 A.9.4.3 (Password management — API credentials)
//   - ISO 27001 A.12.4.1 (Event logging — delivery tracking)
//   - СТБ 34.101.27 (Защита информации — audit trail)
//   - Приказ ОАЦ №66 п.7.18.6 (Мониторинг и реагирование — anti-spam)
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// Константы rate limiting
// ═══════════════════════════════════════════════════════════════════════

// DefaultSMSRateLimit — максимальное количество SMS в минуту на один номер.
// P0-1.4: Anti-spam мера — не более 10 SMS в минуту на получателя.
// Compliance: Приказ ОАЦ №66 п.7.18.6 (Мониторинг и реагирование)
const DefaultSMSRateLimit = 10

// DefaultSMSRateWindow — окно rate limiting (1 минута).
const DefaultSMSRateWindow = 1 * time.Minute

// ═══════════════════════════════════════════════════════════════════════
// Delivery Metrics (P0-1.4)
// ═══════════════════════════════════════════════════════════════════════

// DeliveryMetrics — счётчики доставки для мониторинга.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — delivery tracking)
//   - IEC 62443 SR 2.8 (Audit events — мониторинг канала связи)
type DeliveryMetrics struct {
	Sent        int64 `json:"sent"`
	Failed      int64 `json:"failed"`
	RateLimited int64 `json:"rate_limited"`
	TotalCost   int64 `json:"total_cost_cents"` // в копейках
}

// ═══════════════════════════════════════════════════════════════════════
// RocketSMS Configuration
// ═══════════════════════════════════════════════════════════════════════

// RocketSMSConfig — конфигурация RocketSMS провайдера.
//
// Поля получаются из env vars (НЕ из config.yaml):
//   - ROCKET_SMS_LOGIN — логин от RocketSMS
//   - ROCKET_SMS_PASSWORD — пароль от RocketSMS
//   - ROCKET_SMS_SENDER — отправитель (подпись SMS)
//   - ROCKET_SMS_API_URL — URL API (по умолчанию: https://api.rocketsms.by)
type RocketSMSConfig struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Sender   string `json:"sender"`
	APIURL   string `json:"api_url"`
}

// DefaultRocketSMSConfig — значения по умолчанию.
var DefaultRocketSMSConfig = RocketSMSConfig{
	APIURL: "https://api.rocketsms.by",
	Sender: "CCTV",
}

// ═══════════════════════════════════════════════════════════════════════
// RocketSMS Provider
// ═══════════════════════════════════════════════════════════════════════

// RocketSMSProvider — реализация SMS провайдера через RocketSMS API
// с rate limiting и delivery tracking.
//
// API RocketSMS (json):
//
//	POST https://api.rocketsms.by/send
//	{
//	  "login": "your_login",
//	  "password": "your_password",
//	  "phone": "+375291234567",
//	  "text": "Message text",
//	  "sender": "CCTV"
//	}
//
//	Response (success):
//	{
//	  "status": "success",
//	  "id": 123456,
//	  "phone": "+375291234567",
//	  "cost": 0.05
//	}
//
//	Response (error):
//	{
//	  "status": "error",
//	  "error": "error_description"
//	}
//
// Rate limiting:
//   - Per phone number: не более DefaultSMSRateLimit SMS в минуту
//   - Счётчик автоматически очищается после окна
//
// Delivery tracking:
//   - DeliveryMetrics() возвращает счётчики sent/failed/rate_limited/cost
//   - Сброс через ResetMetrics()
type RocketSMSProvider struct {
	cfg    RocketSMSConfig
	client *http.Client
	logger *slog.Logger

	// P0-1.4: Rate limiting (anti-spam)
	rateLimit   int
	rateWindow  time.Duration
	rateMu      sync.Mutex
	rateEntries map[string][]time.Time // phone → timestamps

	// P0-1.4: Delivery tracking
	metrics DeliveryMetrics
}

// NewRocketSMSProvider создаёт новый RocketSMS провайдер.
//
// Параметры:
//   - cfg: конфигурация RocketSMS
//   - rateLimit: макс. SMS в минуту на номер (0 = DefaultSMSRateLimit)
//   - rateWindow: окно rate limiting (0 = DefaultSMSRateWindow)
//   - logger: логгер (nil = slog.Default())
func NewRocketSMSProvider(cfg RocketSMSConfig, rateLimit int, rateWindow time.Duration, logger *slog.Logger) *RocketSMSProvider {
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultRocketSMSConfig.APIURL
	}
	if cfg.Sender == "" {
		cfg.Sender = DefaultRocketSMSConfig.Sender
	}
	if rateLimit <= 0 {
		rateLimit = DefaultSMSRateLimit
	}
	if rateWindow <= 0 {
		rateWindow = DefaultSMSRateWindow
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &RocketSMSProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger:      logger.With("component", "rocketsms"),
		rateLimit:   rateLimit,
		rateWindow:  rateWindow,
		rateEntries: make(map[string][]time.Time),
	}
}

// rocketSMSRequest — структура запроса к RocketSMS API.
type rocketSMSRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Text     string `json:"text"`
	Sender   string `json:"sender"`
}

// rocketSMSResponse — структура ответа от RocketSMS API.
type rocketSMSResponse struct {
	Status string  `json:"status"`
	ID     int64   `json:"id,omitempty"`
	Phone  string  `json:"phone,omitempty"`
	Cost   float64 `json:"cost,omitempty"`
	Error  string  `json:"error,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════
// Rate Limiting (P0-1.4)
// ═══════════════════════════════════════════════════════════════════════

// allow проверяет rate limit для указанного номера телефона.
// Возвращает true, если SMS разрешена.
//
// Алгоритм:
//  1. Очищает просроченные записи (старше rateWindow)
//  2. Считает количество SMS в текущем окне
//  3. Если превышен лимит — возвращает false
//  4. Если лимит не превышен — добавляет запись и возвращает true
func (p *RocketSMSProvider) allow(phone string) bool {
	p.rateMu.Lock()
	defer p.rateMu.Unlock()

	now := time.Now()
	windowStart := now.Add(-p.rateWindow)

	// Очищаем просроченные записи
	entries := p.rateEntries[phone]
	var active []time.Time
	for _, t := range entries {
		if t.After(windowStart) {
			active = append(active, t)
		}
	}

	if len(active) >= p.rateLimit {
		p.rateEntries[phone] = active
		return false
	}

	active = append(active, now)
	p.rateEntries[phone] = active
	return true
}

// ═══════════════════════════════════════════════════════════════════════
// SMS Delivery
// ═══════════════════════════════════════════════════════════════════════

// SendSMS отправляет SMS через RocketSMS API с rate limiting.
//
// P0-1.4:
//   - Rate limiting: не более DefaultSMSRateLimit SMS в минуту на номер
//   - Delivery tracking: обновляет счётчики sent/failed/rate_limited
//
// Соответствует:
//   - OWASP ASVS V7.1 (Error handling — ошибки API не содержат sensitive data)
//   - OWASP ASVS V8 (Data Protection — credentials не в логах)
//   - ISO 27001 A.12.4.1 (Event logging — логируются только статусы)
//   - Приказ ОАЦ №66 п.7.18.6 (Anti-spam — rate limiting)
func (p *RocketSMSProvider) SendSMS(ctx context.Context, phoneNumber, message string) error {
	if !p.IsAvailable() {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: not configured (login/password empty)")
	}

	// P0-1.4: Rate limiting check
	if !p.allow(phoneNumber) {
		atomic.AddInt64(&p.metrics.RateLimited, 1)
		return fmt.Errorf("rocketsms: rate limit exceeded for %s", maskPhone(phoneNumber))
	}

	reqBody := rocketSMSRequest{
		Login:    p.cfg.Login,
		Password: p.cfg.Password,
		Phone:    phoneNumber,
		Text:     message,
		Sender:   p.cfg.Sender,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: marshal request: %w", err)
	}

	apiURL, err := url.JoinPath(p.cfg.APIURL, "/send")
	if err != nil {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: invalid api url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: read response: %w", err)
	}

	var smsResp rocketSMSResponse
	if err := json.Unmarshal(respBody, &smsResp); err != nil {
		atomic.AddInt64(&p.metrics.Failed, 1)
		return fmt.Errorf("rocketsms: parse response: %w", err)
	}

	if smsResp.Status != "success" {
		atomic.AddInt64(&p.metrics.Failed, 1)
		errMsg := smsResp.Error
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("rocketsms: api error: %s", errMsg)
	}

	// P0-1.4: Delivery tracking
	atomic.AddInt64(&p.metrics.Sent, 1)
	atomic.AddInt64(&p.metrics.TotalCost, int64(smsResp.Cost*100)) // рубли → копейки

	p.logger.Info("SMS sent successfully",
		"id", smsResp.ID,
		"cost", smsResp.Cost,
	)

	return nil
}

// IsAvailable проверяет, настроен ли RocketSMS провайдер.
func (p *RocketSMSProvider) IsAvailable() bool {
	return p.cfg.Login != "" && p.cfg.Password != ""
}

// GetConfig возвращает текущую конфигурацию (без пароля).
func (p *RocketSMSProvider) GetConfig() RocketSMSConfig {
	return RocketSMSConfig{
		Login:  p.cfg.Login,
		Sender: p.cfg.Sender,
		APIURL: p.cfg.APIURL,
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Delivery Metrics (P0-1.4)
// ═══════════════════════════════════════════════════════════════════════

// DeliveryMetrics возвращает текущие счётчики доставки.
//
// Используется для Prometheus метрик и мониторинга.
// Compliance: ISO 27001 A.12.4.1 (Event logging)
func (p *RocketSMSProvider) DeliveryMetrics() DeliveryMetrics {
	return DeliveryMetrics{
		Sent:        atomic.LoadInt64(&p.metrics.Sent),
		Failed:      atomic.LoadInt64(&p.metrics.Failed),
		RateLimited: atomic.LoadInt64(&p.metrics.RateLimited),
		TotalCost:   atomic.LoadInt64(&p.metrics.TotalCost),
	}
}

// ResetMetrics сбрасывает все счётчики доставки.
func (p *RocketSMSProvider) ResetMetrics() {
	atomic.StoreInt64(&p.metrics.Sent, 0)
	atomic.StoreInt64(&p.metrics.Failed, 0)
	atomic.StoreInt64(&p.metrics.RateLimited, 0)
	atomic.StoreInt64(&p.metrics.TotalCost, 0)
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// maskPhone маскирует номер телефона для логов (OWASP ASVS V7.1).
func maskPhone(phone string) string {
	if len(phone) < 5 {
		return "***"
	}
	if len(phone) < 6 {
		return phone[:3] + "***" + phone[len(phone)-2:]
	}
	return phone[:4] + "***" + phone[len(phone)-2:]
}
