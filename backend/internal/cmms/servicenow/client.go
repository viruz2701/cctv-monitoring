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

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// Client — HTTP-клиент ServiceNow с поддержкой OAuth2 и Basic Auth.
type Client struct {
	httpClient *http.Client
	baseURL    string
	mu         sync.RWMutex
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

// NewClient создаёт OAuth2-клиент или Basic Auth клиент.
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.InstanceURL == "" {
		return nil, fmt.Errorf("servicenow client: instance URL is required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	var httpClient *http.Client

	if cfg.ClientID != "" && cfg.ClientSecret != "" && cfg.TokenURL != "" {
		oauthCfg := clientcredentials.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			TokenURL:     cfg.TokenURL,
			Scopes:       []string{"useraccount"},
			AuthStyle:    oauth2.AuthStyleInHeader,
		}
		httpClient = oauthCfg.Client(context.Background())
		httpClient.Timeout = cfg.Timeout
	} else {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}

	return &Client{httpClient: httpClient, baseURL: cfg.InstanceURL}, nil
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

// do выполняет HTTP-запрос.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, username, password string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("servicenow request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if username != "" && password != "" {
		req.SetBasicAuth(username, password)
	}

	return c.httpClient.Do(req)
}

func (c *Client) get(ctx context.Context, path string, dest interface{}, username, password string) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil, username, password)
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

func (c *Client) post(ctx context.Context, path string, body interface{}, dest interface{}, username, password string) error {
	bodyReader, err := jsonReader(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPost, path, bodyReader, username, password)
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

func (c *Client) put(ctx context.Context, path string, body interface{}, dest interface{}, username, password string) error {
	bodyReader, err := jsonReader(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPut, path, bodyReader, username, password)
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

func (c *Client) patch(ctx context.Context, path string, body interface{}, dest interface{}, username, password string) error {
	bodyReader, err := jsonReader(body)
	if err != nil {
		return err
	}
	resp, err := c.do(ctx, http.MethodPatch, path, bodyReader, username, password)
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

func (c *Client) delete(ctx context.Context, path string, username, password string) error {
	resp, err := c.do(ctx, http.MethodDelete, path, nil, username, password)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return parseError(resp)
	}
	return nil
}

func (c *Client) getRaw(ctx context.Context, path string, username, password string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, username, password)
}

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
