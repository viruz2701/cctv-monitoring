package cmms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

// AtlasClient — OAuth2 HTTP-клиент для взаимодействия с внешним Atlas CMMS API.
// Использует client credentials grant для автоматического получения и
// обновления access token.
type AtlasClient struct {
	httpClient *http.Client
	baseURL    string
	mu         sync.RWMutex
}

// AtlasClientConfig содержит параметры для создания AtlasClient.
type AtlasClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	TokenURL     string
	Timeout      time.Duration
}

// NewAtlasClient создаёт OAuth2-клиент для Atlas CMMS API.
// Использует client credentials flow для автоматического управления токенами.
func NewAtlasClient(cfg AtlasClientConfig) (*AtlasClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("atlas client: base URL is required")
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" || cfg.TokenURL == "" {
		return nil, fmt.Errorf("atlas client: client_id, client_secret and token_url are required for OAuth2")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	oauthCfg := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Scopes:       []string{"cmms:read", "cmms:write"},
	}

	ctx := context.Background()
	httpClient := oauthCfg.Client(ctx)
	httpClient.Timeout = cfg.Timeout

	return &AtlasClient{
		httpClient: httpClient,
		baseURL:    cfg.BaseURL,
	}, nil
}

// NewAtlasClientWithAPIKey создаёт клиент, использующий статический API-ключ
// вместо OAuth2. Это fallback-режим для обратной совместимости.
func NewAtlasClientWithAPIKey(baseURL, apiKey string, timeout time.Duration) *AtlasClient {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &AtlasClient{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
	}
}

// do выполняет HTTP-запрос с автоматическим OAuth2-токеном (если настроен)
// или API-ключом (если передан).
func (c *AtlasClient) do(ctx context.Context, method, path string, body io.Reader, apiKey string) (*http.Response, error) {
	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("atlas request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return c.httpClient.Do(req)
}

// get выполняет GET-запрос и декодирует JSON-ответ в dest.
func (c *AtlasClient) get(ctx context.Context, path string, dest interface{}, apiKey string) error {
	resp, err := c.do(ctx, http.MethodGet, path, nil, apiKey)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// post выполняет POST-запрос с JSON-телом.
func (c *AtlasClient) post(ctx context.Context, path string, body interface{}, dest interface{}, apiKey string) error {
	bodyReader, err := c.jsonReader(body)
	if err != nil {
		return err
	}

	resp, err := c.do(ctx, http.MethodPost, path, bodyReader, apiKey)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// put выполняет PUT-запрос с JSON-телом.
func (c *AtlasClient) put(ctx context.Context, path string, body interface{}, dest interface{}, apiKey string) error {
	bodyReader, err := c.jsonReader(body)
	if err != nil {
		return err
	}

	resp, err := c.do(ctx, http.MethodPut, path, bodyReader, apiKey)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

// delete выполняет DELETE-запрос.
func (c *AtlasClient) delete(ctx context.Context, path string, apiKey string) error {
	resp, err := c.do(ctx, http.MethodDelete, path, nil, apiKey)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.parseError(resp)
	}
	return nil
}

// getRaw выполняет GET и возвращает сырой ответ.
func (c *AtlasClient) getRaw(ctx context.Context, path string, apiKey string) (*http.Response, error) {
	resp, err := c.do(ctx, http.MethodGet, path, nil, apiKey)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, c.parseError(resp)
	}
	return resp, nil
}

// jsonReader сериализует объект в io.Reader для HTTP-тела.
func (c *AtlasClient) jsonReader(v interface{}) (io.Reader, error) {
	if v == nil {
		return nil, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("atlas marshal: %w", err)
	}
	return io.NopCloser(bytesReader(data)), nil
}

// parseError читает тело ошибки из ответа API.
func (c *AtlasClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("atlas api error %d: %s", resp.StatusCode, string(body))
}

// bytesReader — вспомогательный тип для превращения []byte в io.ReadCloser
// без импорта bytes (используем strings).
type bytesReader []byte

func (b bytesReader) Read(p []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, io.EOF
	}
	n = copy(p, b)
	return n, nil
}

func (b bytesReader) Close() error { return nil }
