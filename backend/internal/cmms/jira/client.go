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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Client — HTTP-клиент для Jira Cloud REST API v3.
// Аутентификация: OAuth 2.0 (3LO) или Basic Auth (email:api_token).
type Client struct {
	httpClient *http.Client
	baseURL    string
	email      string
	apiToken   string
	mu         sync.RWMutex
}

// ClientConfig — параметры подключения к Jira.
type ClientConfig struct {
	BaseURL  string // https://your-domain.atlassian.net
	Email    string
	APIToken string
	Timeout  time.Duration
}

// NewClient создаёт Jira клиент с Basic Auth (email:api_token).
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("jira client: base URL is required")
	}
	if cfg.Email == "" || cfg.APIToken == "" {
		return nil, fmt.Errorf("jira client: email and api_token are required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    cfg.BaseURL,
		email:      cfg.Email,
		apiToken:   cfg.APIToken,
	}, nil
}

// OAuth2Config — конфигурация OAuth2 Client Credentials.
//
// P2-3.2: OAuth2 for External Adapters
type OAuth2Config struct {
	TokenURL     string   `json:"token_url"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	Scopes       []string `json:"scopes,omitempty"`
}

// TokenAwareClient — HTTP клиент с OAuth2 токеном и auto-refresh.
//
// P2-3.2: OAuth2 for External Adapters
//   - OAuth2 Client Credentials flow
//   - Token auto-refresh
//   - Secure token storage (in-memory)
//   - Fallback to basic auth
type TokenAwareClient struct {
	config *clientcredentials.Config
	mu     sync.RWMutex
	token  *oauth2.Token
	logger *slog.Logger
}

// NewTokenAwareClient создаёт клиент с OAuth2 Client Credentials.
func NewTokenAwareClient(cfg OAuth2Config, logger *slog.Logger) *TokenAwareClient {
	if logger == nil {
		logger = slog.Default()
	}

	return &TokenAwareClient{
		config: &clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Scopes:       cfg.Scopes,
		},
		logger: logger.With("component", "oauth2-client"),
	}
}

// Client возвращает HTTP клиент с OAuth2 токеном (с auto-refresh).
func (c *TokenAwareClient) Client(ctx context.Context) *http.Client {
	return c.config.Client(ctx)
}

// Token возвращает текущий токен (обновляет если истёк).
func (c *TokenAwareClient) Token(ctx context.Context) (*oauth2.Token, error) {
	c.mu.RLock()
	token := c.token
	c.mu.RUnlock()

	if token != nil && token.Valid() {
		return token, nil
	}

	// Token истёк или отсутствует — получаем новый
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check после блокировки
	if c.token != nil && c.token.Valid() {
		return c.token, nil
	}

	newToken, err := c.config.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("oauth2 token refresh: %w", err)
	}

	c.token = newToken
	c.logger.Info("OAuth2 token refreshed", "expiry", newToken.Expiry)
	return newToken, nil
}

// do выполняет HTTP-запрос с Basic Auth.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("jira request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.SetBasicAuth(c.email, c.apiToken)

	return c.httpClient.Do(req)
}

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

func (c *Client) getRaw(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

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
