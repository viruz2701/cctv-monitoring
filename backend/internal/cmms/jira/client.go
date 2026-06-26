// Package jira предоставляет HTTP-клиент для Jira Cloud REST API v3.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.2: OAuth2 for External Adapters
//
// Поддерживает:
//   - OAuth2 Client Credentials flow (через TokenManager)
//   - Token auto-refresh с grace-периодом
//   - Зашифрованное хранение токенов в БД
//   - Fallback на Basic Auth (email:api_token) если OAuth2 не настроен
//   - Метрики: token_refresh_count, token_expired_count
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application)
//   - ISO 27001 A.9.4.2 (Authentication)
//   - OWASP ASVS V2.1 (Credential storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"gb-telemetry-collector/internal/oauth2"
)

// ────────────────────────────────────────────────────────────────────────────
// Client
// ────────────────────────────────────────────────────────────────────────────

// Client — HTTP-клиент для Jira Cloud REST API v3.
// Аутентификация: OAuth 2.0 (Client Credentials) или Basic Auth (email:api_token).
type Client struct {
	httpClient *http.Client
	baseURL    string
	email      string
	apiToken   string
	tokenMgr   *oauth2.TokenManager
	mu         sync.RWMutex
	logger     *slog.Logger
}

// ClientConfig — параметры подключения к Jira.
type ClientConfig struct {
	BaseURL      string // https://your-domain.atlassian.net
	Email        string
	APIToken     string
	ClientID     string // OAuth2
	ClientSecret string // OAuth2
	TokenURL     string // OAuth2
	Timeout      time.Duration
}

// NewClient создаёт Jira клиент с OAuth2 или Basic Auth.
//
// Приоритет аутентификации:
//  1. OAuth2 Client Credentials (если указаны ClientID, ClientSecret, TokenURL)
//  2. Basic Auth (email:api_token)
//  3. Ошибка, если не указано ничего
func NewClient(cfg ClientConfig, tokenStore oauth2.TokenStore, metrics *oauth2.Metrics, logger *slog.Logger) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("jira client: base URL is required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}

	httpClient := &http.Client{Timeout: cfg.Timeout}

	var tokenMgr *oauth2.TokenManager

	// Приоритет 1: OAuth2 Client Credentials
	if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TokenURL != "" {
		if metrics == nil {
			metrics = oauth2.NewMetrics()
		}

		tokenCfg := &oauth2.TokenManagerConfig{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Scopes:       []string{"read:jira-work", "write:jira-work", "offline_access"},
		}

		tokenMgr = oauth2.NewTokenManager(
			tokenCfg, tokenStore, metrics, logger,
			"jira", cfg.BaseURL,
		)
		logger.Info("jira client: OAuth2 configured",
			"base_url", cfg.BaseURL,
			"token_url", cfg.TokenURL,
		)
	} else if cfg.Email != "" && cfg.APIToken != "" {
		// Приоритет 2: Basic Auth
		logger.Info("jira client: Basic Auth configured (OAuth2 not available)",
			"base_url", cfg.BaseURL,
			"email", cfg.Email,
		)
	} else {
		return nil, fmt.Errorf("jira client: no authentication method configured; provide OAuth2 (ClientID, ClientSecret, TokenURL) or Basic Auth (Email, APIToken)")
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    cfg.BaseURL,
		email:      cfg.Email,
		apiToken:   cfg.APIToken,
		tokenMgr:   tokenMgr,
		logger:     logger.With("component", "jira-client", "base_url", cfg.BaseURL),
	}, nil
}

// ── Публичные методы ─────────────────────────────────────────────────────

// Get выполняет GET-запрос с OAuth2 или Basic Auth.
func (c *Client) Get(ctx context.Context, path string, dest interface{}) error {
	return c.get(ctx, path, dest)
}

// Post выполняет POST-запрос с OAuth2 или Basic Auth.
func (c *Client) Post(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return c.post(ctx, path, body, dest)
}

// Put выполняет PUT-запрос с OAuth2 или Basic Auth.
func (c *Client) Put(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return c.put(ctx, path, body, dest)
}

// Delete выполняет DELETE-запрос с OAuth2 или Basic Auth.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.delete(ctx, path)
}

// GetRaw выполняет GET-запрос и возвращает сырой *http.Response.
func (c *Client) GetRaw(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// ForceTokenRefresh принудительно обновляет OAuth2 токен.
func (c *Client) ForceTokenRefresh(ctx context.Context) error {
	if c.tokenMgr == nil {
		return fmt.Errorf("jira: OAuth2 not configured")
	}
	_, err := c.tokenMgr.ForceRefresh(ctx)
	return err
}

// InvalidateToken инвалидирует текущий токен (при 401 от сервера).
func (c *Client) InvalidateToken(ctx context.Context) {
	if c.tokenMgr != nil {
		c.tokenMgr.InvalidateToken(ctx)
	}
}

// OAuth2Metrics возвращает метрики OAuth2 (может быть nil если OAuth2 не настроен).
func (c *Client) OAuth2Metrics() *oauth2.Metrics {
	if c.tokenMgr == nil {
		return nil
	}
	return c.tokenMgr.Metrics()
}

// ── Внутренние HTTP методы ──────────────────────────────────────────────

// do выполняет HTTP-запрос с аутентификацией.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("jira request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Выбираем метод аутентификации
	if err := c.authenticateRequest(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jira: %w", err)
	}

	// Если 401 и OAuth2 — инвалидируем токен для следующего запроса
	if resp.StatusCode == http.StatusUnauthorized && c.tokenMgr != nil {
		c.logger.Warn("jira returned 401, invalidating token")
		c.tokenMgr.InvalidateToken(ctx)
	}

	return resp, nil
}

// authenticateRequest добавляет аутентификацию к запросу.
// Приоритет: OAuth2 → Basic Auth.
func (c *Client) authenticateRequest(req *http.Request) error {
	if c.tokenMgr != nil && c.tokenMgr.IsConfigured() {
		token, err := c.tokenMgr.GetToken(req.Context())
		if err != nil {
			return fmt.Errorf("jira: oauth2 auth: %w", err)
		}
		token.SetAuthHeader(req)
		return nil
	}

	if c.email != "" && c.apiToken != "" {
		req.SetBasicAuth(c.email, c.apiToken)
		return nil
	}

	return fmt.Errorf("jira: no authentication method available")
}

// ── CRUD методы ──────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, dest interface{}) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string, body interface{}, dest interface{}) error {
	bodyReader, err := jsonReader(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPost, path, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

func (c *Client) put(ctx context.Context, path string, body interface{}, dest interface{}) error {
	bodyReader, err := jsonReader(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPut, path, bodyReader)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

func (c *Client) delete(ctx context.Context, path string) error {
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────

func jsonReader(v interface{}) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("jira: marshal body: %w", err)
	}
	return bytes.NewReader(data), nil
}

func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("jira: HTTP %d: %s", resp.StatusCode, string(body))
}
