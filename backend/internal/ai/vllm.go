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

type VLLMConfig struct {
	BaseURL string
	Model   string
}

type VLLMProvider struct {
	cfg VLLMConfig
	log *slog.Logger
	hc  *http.Client
}

func NewVLLMProvider(cfg VLLMConfig, log *slog.Logger) *VLLMProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8000"
	}
	if cfg.Model == "" {
		cfg.Model = "mistral"
	}
	return &VLLMProvider{
		cfg: cfg,
		log: log.With("provider", "vllm"),
		hc:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *VLLMProvider) Type() ProviderType { return ProviderVLLM }
func (p *VLLMProvider) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", p.cfg.BaseURL+"/health", nil)
	resp, err := p.hc.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (p *VLLMProvider) Models(ctx context.Context) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", p.cfg.BaseURL+"/v1/models", nil)
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vllm models: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("vllm decode: %w", err)
	}
	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}
	return models, nil
}

func (p *VLLMProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
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
		return nil, fmt.Errorf("vllm decode: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("vllm: no choices")
	}
	return &ChatResponse{Content: result.Choices[0].Message.Content, Model: p.cfg.Model, Done: true}, nil
}

func (p *VLLMProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
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

func (p *VLLMProvider) buildBody(req ChatRequest, stream bool) map[string]interface{} {
	return map[string]interface{}{
		"model":    p.cfg.Model,
		"messages": req.Messages,
		"stream":   stream,
	}
}

func (p *VLLMProvider) doRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("vllm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vllm: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("vllm: status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
