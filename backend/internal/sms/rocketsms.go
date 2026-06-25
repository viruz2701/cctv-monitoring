// Package sms — SMS providers для уведомлений.
//
// NOTIF-01: Поддержка SMS-шлюзов для отправки критических уведомлений.
// Текущий провайдер: RocketSMS (by kartasoft)
//
// API RocketSMS: https://rocketsms.by/api
// Метод: POST /api/send
// Параметры: login, password, phone, text, sender
//
// Compliance:
//   - OWASP ASVS V7.1 (Log content — номера маскируются в логах)
//   - OWASP ASVS V8 (Data Protection — credentials в env, не в коде)
//   - ISO 27001 A.9.4.3 (Password management — API credentials)
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
	"time"
)

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

// RocketSMSProvider — реализация SMS провайдера через RocketSMS API.
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
type RocketSMSProvider struct {
	cfg    RocketSMSConfig
	client *http.Client
	logger *slog.Logger
}

// NewRocketSMSProvider создаёт новый RocketSMS провайдер.
func NewRocketSMSProvider(cfg RocketSMSConfig, logger *slog.Logger) *RocketSMSProvider {
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultRocketSMSConfig.APIURL
	}
	if cfg.Sender == "" {
		cfg.Sender = DefaultRocketSMSConfig.Sender
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &RocketSMSProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger.With("component", "rocketsms"),
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

// SendSMS отправляет SMS через RocketSMS API.
//
// Соответствует:
//   - OWASP ASVS V7.1 (Error handling — ошибки API не содержат sensitive data)
//   - OWASP ASVS V8 (Data Protection — credentials не в логах)
//   - ISO 27001 A.12.4.1 (Event logging — логируются только статусы)
func (p *RocketSMSProvider) SendSMS(ctx context.Context, phoneNumber, message string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("rocketsms: not configured (login/password empty)")
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
		return fmt.Errorf("rocketsms: marshal request: %w", err)
	}

	apiURL, err := url.JoinPath(p.cfg.APIURL, "/send")
	if err != nil {
		return fmt.Errorf("rocketsms: invalid api url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("rocketsms: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("rocketsms: send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("rocketsms: read response: %w", err)
	}

	var smsResp rocketSMSResponse
	if err := json.Unmarshal(respBody, &smsResp); err != nil {
		return fmt.Errorf("rocketsms: parse response: %w", err)
	}

	if smsResp.Status != "success" {
		errMsg := smsResp.Error
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("rocketsms: api error: %s", errMsg)
	}

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
