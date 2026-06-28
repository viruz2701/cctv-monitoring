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

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type OpenAIProvider struct {
	cfg OpenAIConfig
	log *slog.Logger
	hc  *http.Client
}

func NewOpenAIProvider(cfg OpenAIConfig, log *slog.Logger) *OpenAIProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com"
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o"
	}
	return &OpenAIProvider{
		cfg: cfg,
		log: log.With("provider", "openai"),
		hc:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Type() ProviderType                   { return ProviderOpenAI }
func (p *OpenAIProvider) IsAvailable(ctx context.Context) bool { return p.cfg.APIKey != "" }
func (p *OpenAIProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"}, nil
}

func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req, false)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("openai decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices")
	}
	return &ChatResponse{Content: result.Choices[0].Message.Content, Model: p.cfg.Model, Done: true}, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
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
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if len(chunk.Choices) > 0 {
				ch <- ChatResponse{Content: chunk.Choices[0].Delta.Content, Model: p.cfg.Model}
			}
		}
	}()
	return ch, nil
}

func (p *OpenAIProvider) buildBody(req ChatRequest, stream bool) map[string]interface{} {
	return map[string]interface{}{
		"model":    p.cfg.Model,
		"messages": req.Messages,
		"stream":   stream,
	}
}

func (p *OpenAIProvider) doRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai: status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
