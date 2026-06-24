// Package recaptcha — валидация Google reCAPTCHA v2/v3 токенов.
//
// Используется для публичного submit endpoint (WO-4.1.1) без авторизации.
//
// Compliance:
//   - OWASP ASVS V3.1 (Session management — bot detection)
//   - ISO 27001 A.9.2.1 (User registration — anti-automation)
//   - СТБ 34.101.27 п. 6.3 (Защита от автоматизированных атак)
package recaptcha

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ── Constants ──────────────────────────────────────────────────────

const (
	// VerifyURL — Google reCAPTCHA API endpoint.
	VerifyURL = "https://www.google.com/recaptcha/api/siteverify"

	// DefaultTimeout — таймаут HTTP запроса к Google.
	DefaultTimeout = 5 * time.Second
)

// ── Types ──────────────────────────────────────────────────────────

// Config — конфигурация reCAPTCHA.
type Config struct {
	// SecretKey — секретный ключ reCAPTCHA (серверный).
	SecretKey string `json:"secret_key"`

	// SiteKey — публичный ключ reCAPTCHA (для клиента).
	SiteKey string `json:"site_key"`

	// MinScore — минимальный score для reCAPTCHA v3 (0.0-1.0).
	// По умолчанию 0.5. Для v2 игнорируется.
	MinScore float32 `json:"min_score"`

	// Enabled — включает/выключает валидацию.
	Enabled bool `json:"enabled"`

	// HTTPClient — кастомный HTTP клиент (опционально).
	HTTPClient *http.Client `json:"-"`
}

// VerifyResponse — ответ от Google reCAPTCHA API.
type VerifyResponse struct {
	Success     bool     `json:"success"`
	Score       float32  `json:"score,omitempty"`       // v3 only
	Action      string   `json:"action,omitempty"`      // v3 only
	ChallengeTS string   `json:"challenge_ts,omitempty"` // timestamp
	Hostname    string   `json:"hostname,omitempty"`
	ErrorCodes  []string `json:"error-codes,omitempty"`
}

// Validator — валидатор reCAPTCHA токенов.
type Validator struct {
	cfg    Config
	client *http.Client
}

// NewValidator создаёт новый валидатор reCAPTCHA.
func NewValidator(cfg Config) *Validator {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: DefaultTimeout}
	}
	if cfg.MinScore == 0 {
		cfg.MinScore = 0.5
	}
	return &Validator{
		cfg:    cfg,
		client: cfg.HTTPClient,
	}
}

// Verify проверяет токен reCAPTCHA.
//
// Если валидация отключена (Enabled=false) — всегда успех.
// Возвращает ошибку если токен невалиден или score ниже порога (для v3).
func (v *Validator) Verify(ctx context.Context, token string) error {
	if !v.cfg.Enabled {
		return nil
	}

	if token == "" {
		return fmt.Errorf("recaptcha: token is required")
	}

	resp, err := v.verifyToken(ctx, token)
	if err != nil {
		return fmt.Errorf("recaptcha: verify: %w", err)
	}

	if !resp.Success {
		if len(resp.ErrorCodes) > 0 {
			return fmt.Errorf("recaptcha: %s", strings.Join(resp.ErrorCodes, ", "))
		}
		return fmt.Errorf("recaptcha: verification failed")
	}

	// Для reCAPTCHA v3 проверяем score
	if resp.Score > 0 && resp.Score < v.cfg.MinScore {
		return fmt.Errorf("recaptcha: score too low (%.2f < %.2f)", resp.Score, v.cfg.MinScore)
	}

	return nil
}

// SiteKey возвращает публичный ключ для клиентской стороны.
func (v *Validator) SiteKey() string {
	return v.cfg.SiteKey
}

// IsEnabled возвращает статус валидации.
func (v *Validator) IsEnabled() bool {
	return v.cfg.Enabled
}

func (v *Validator) verifyToken(ctx context.Context, token string) (*VerifyResponse, error) {
	data := url.Values{
		"secret": {v.cfg.SecretKey},
		"response": {token},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, VerifyURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var verifyResp VerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &verifyResp, nil
}
