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

type DeepSeekConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type DeepSeekProvider struct {
	cfg DeepSeekConfig
	log *slog.Logger
	hc  *http.Client
}

func NewDeepSeekProvider(cfg DeepSeekConfig, log *slog.Logger) *DeepSeekProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}
	return &DeepSeekProvider{
		cfg: cfg,
		log: log.With("provider", "deepseek"),
		hc:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *DeepSeekProvider) Type() ProviderType { return ProviderDeepSeek }

func (p *DeepSeekProvider) IsAvailable(ctx context.Context) bool {
	return p.cfg.APIKey != ""
}

func (p *DeepSeekProvider) Models(ctx context.Context) ([]string, error) {
	return []string{"deepseek-chat", "deepseek-reasoner"}, nil
}

func (p *DeepSeekProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req)
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
		return nil, fmt.Errorf("deepseek decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("deepseek: no choices")
	}
	return &ChatResponse{Content: result.Choices[0].Message.Content, Model: p.cfg.Model, Done: true}, nil
}

func (p *DeepSeekProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	body := p.buildBody(req)
	body["stream"] = true
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

func (p *DeepSeekProvider) buildBody(req ChatRequest) map[string]interface{} {
	return map[string]interface{}{
		"model":    p.cfg.Model,
		"messages": req.Messages,
	}
}

func (p *DeepSeekProvider) doRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("deepseek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deepseek: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("deepseek: status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
