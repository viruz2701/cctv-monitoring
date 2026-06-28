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
	"time"
)

type OllamaConfig struct {
	BaseURL string
	Model   string
}

type OllamaProvider struct {
	cfg OllamaConfig
	log *slog.Logger
	hc  *http.Client
}

func NewOllamaProvider(cfg OllamaConfig, log *slog.Logger) *OllamaProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}
	if cfg.Model == "" {
		cfg.Model = "llama3"
	}
	return &OllamaProvider{
		cfg: cfg,
		log: log.With("provider", "ollama"),
		hc:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (p *OllamaProvider) Type() ProviderType { return ProviderOllama }
func (p *OllamaProvider) IsAvailable(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", p.cfg.BaseURL+"/api/tags", nil)
	resp, err := p.hc.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

func (p *OllamaProvider) Models(ctx context.Context) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", p.cfg.BaseURL+"/api/tags", nil)
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama tags: %w", err)
	}
	defer resp.Body.Close()
	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}
	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}
	return models, nil
}

func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	body := p.buildBody(req, false)
	resp, err := p.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Done bool `json:"done"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}
	return &ChatResponse{Content: result.Message.Content, Model: p.cfg.Model, Done: result.Done}, nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
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
			if line == "" {
				continue
			}
			var chunk struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done bool `json:"done"`
			}
			if err := json.Unmarshal([]byte(line), &chunk); err != nil {
				continue
			}
			ch <- ChatResponse{Content: chunk.Message.Content, Model: p.cfg.Model, Done: chunk.Done}
			if chunk.Done {
				return
			}
		}
	}()
	return ch, nil
}

func (p *OllamaProvider) buildBody(req ChatRequest, stream bool) map[string]interface{} {
	ollamaMsgs := make([]map[string]interface{}, len(req.Messages))
	for i, m := range req.Messages {
		ollamaMsgs[i] = map[string]interface{}{"role": m.Role, "content": m.Content}
	}
	return map[string]interface{}{
		"model":    p.cfg.Model,
		"messages": ollamaMsgs,
		"stream":   stream,
	}
}

func (p *OllamaProvider) doRequest(ctx context.Context, body map[string]interface{}) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.BaseURL+"/api/chat", bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: status %d: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}
