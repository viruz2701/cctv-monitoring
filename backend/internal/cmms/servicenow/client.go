// Package servicenow предоставляет HTTP-клиент для ServiceNow API.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.2: OAuth2 for External Adapters
//
// Поддерживает:
//   - OAuth2 Client Credentials flow (через TokenManager)
//   - Token auto-refresh с grace-периодом
//   - Зашифрованное хранение токенов в БД
//   - Fallback на Basic Auth если OAuth2 не настроен
//   - Метрики: token_refresh_count, token_expired_count
//
// Compliance:
//   - IEC 62443 SL-3 (Zone 3 — Application)
//   - ISO 27001 A.9.4.2 (Authentication)
//   - OWASP ASVS V2.1 (Credential storage)
//
// ═══════════════════════════════════════════════════════════════════════════
package servicenow

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

// Client — HTTP-клиент ServiceNow с поддержкой OAuth2 и Basic Auth.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	tokenMgr   *oauth2.TokenManager
	mu         sync.RWMutex
	logger     *slog.Logger
}

// ClientConfig — параметры подключения к ServiceNow.
type ClientConfig struct {
	InstanceURL  string
	ClientID     string
	ClientSecret string
	TokenURL     string
	Username     string
	Password     string
	Timeout      time.Duration
}

// NewClient создаёт ServiceNow клиент с OAuth2 или Basic Auth.
//
// Приоритет аутентификации:
//  1. OAuth2 Client Credentials (если указаны ClientID, ClientSecret, TokenURL)
//  2. Basic Auth (если указаны Username, Password)
//  3. Ошибка, если не указано ничего
func NewClient(cfg ClientConfig, tokenStore oauth2.TokenStore, metrics *oauth2.Metrics, logger *slog.Logger) (*Client, error) {
	if cfg.InstanceURL == "" {
		return nil, fmt.Errorf("servicenow client: instance URL is required")
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
			Scopes:       []string{"useraccount"},
		}

		tokenMgr = oauth2.NewTokenManager(
			tokenCfg, tokenStore, metrics, logger,
			"servicenow", cfg.InstanceURL,
		)
		logger.Info("servicenow client: OAuth2 configured",
			"instance", cfg.InstanceURL,
			"token_url", cfg.TokenURL,
		)
	} else if cfg.Username != "" && cfg.Password != "" {
		// Приоритет 2: Basic Auth
		logger.Info("servicenow client: Basic Auth configured (OAuth2 not available)",
			"instance", cfg.InstanceURL,
			"username", cfg.Username,
		)
	} else {
		return nil, fmt.Errorf("servicenow client: no authentication method configured; provide OAuth2 (ClientID, ClientSecret, TokenURL) or Basic Auth (Username, Password)")
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    cfg.InstanceURL,
		username:   cfg.Username,
		password:   cfg.Password,
		tokenMgr:   tokenMgr,
		logger:     logger.With("component", "servicenow-client", "instance", cfg.InstanceURL),
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

// Patch выполняет PATCH-запрос с OAuth2 или Basic Auth.
func (c *Client) Patch(ctx context.Context, path string, body interface{}, dest interface{}) error {
	return c.patch(ctx, path, body, dest)
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
		return fmt.Errorf("servicenow: OAuth2 not configured")
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
		return nil, fmt.Errorf("servicenow request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Выбираем метод аутентификации
	if _, err := c.authenticateRequest(req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("servicenow: %w", err)
	}

	// Если 401 и OAuth2 — инвалидируем токен для следующего запроса
	if resp.StatusCode == http.StatusUnauthorized && c.tokenMgr != nil {
		c.logger.Warn("servicenow returned 401, invalidating token")
		c.tokenMgr.InvalidateToken(ctx)
	}

	return resp, nil
}

// authenticateRequest добавляет аутентификацию к запросу.
// Приоритет: OAuth2 → Basic Auth.
func (c *Client) authenticateRequest(req *http.Request) (bool, error) {
	if c.tokenMgr != nil && c.tokenMgr.IsConfigured() {
		token, err := c.tokenMgr.GetToken(req.Context())
		if err != nil {
			return false, fmt.Errorf("servicenow: oauth2 auth: %w", err)
		}
		token.SetAuthHeader(req)
		return true, nil
	}

	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
		return true, nil
	}

	return false, fmt.Errorf("servicenow: no authentication method available")
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

func (c *Client) patch(ctx context.Context, path string, body interface{}, dest interface{}) error {
	bodyReader, err := jsonReader(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPatch, path, bodyReader)
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
		return nil, fmt.Errorf("servicenow: marshal body: %w", err)
	}
	return bytes.NewReader(data), nil
}

func parseError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("servicenow: HTTP %d: %s", resp.StatusCode, string(body))
}
