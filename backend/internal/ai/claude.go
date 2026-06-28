package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type ClaudeConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type ClaudeProvider struct {
	cfg ClaudeConfig
	log *slog.Logger
	hc  *http.Client
}

func NewClaudeProvider(cfg ClaudeConfig, log *slog.Logger) *ClaudeProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}
	return &ClaudeProvider{
		cfg: cfg,
		log: log.With("provider", "claude"),
		hc:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *ClaudeProvider) Type() ProviderType                   { return ProviderClaude }
func (p *ClaudeProvider) IsAvailable(ctx context.Context) bool { return p.cfg.APIKey != "" }
func (p *ClaudeProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"claude-sonnet-4-20250514", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229"}, nil
}

func (p *ClaudeProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req, false)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("claude decode: %w", err)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("claude: no content")
	}
	return &ChatResponse{Content: result.Content[0].Text, Model: p.cfg.Model, Done: true}, nil
}

func (p *ClaudeProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	body := p.buildBody(req, true)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	ch := make(chan ChatResponse, 64)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- ChatResponse{Done: true}
				return
			}
			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}
			if event.Type == "content_block_delta" {
				ch <- ChatResponse{Content: event.Delta.Text, Model: p.cfg.Model}
			}
		}
	}()
	return ch, nil
}

func (p *ClaudeProvider) buildBody(req ChatRequest, stream bool) map[string]interface{} {
	systemMsg := ""
	messages := make([]map[string]interface{}, 0)
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemMsg = m.Content
		} else {
			messages = append(messages, map[string]interface{}{
				"role":    m.Role,
				"content": m.Content,
			})
		}
	}
	body := map[string]interface{}{
		"model":      p.cfg.Model,
		"messages":   messages,
		"max_tokens": 4096,
	}
	if stream {
		body["stream"] = true
	}
	if systemMsg != "" {
		body["system"] = systemMsg
	}
	return body
}

func (p *ClaudeProvider) doRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/v1/messages", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("claude request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("claude: status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
