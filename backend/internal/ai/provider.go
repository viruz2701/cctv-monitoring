// Package ai — Multi-Provider AI Chat (P2-AI).
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-AI: Multi-Provider AI Assistant
//
// Поддержка нескольких AI провайдеров:
//   - DeepSeek (существующий)
//   - OpenAI (GPT-4, GPT-3.5)
//   - Anthropic Claude
//   - Локальные модели через Ollama/vLLM
//
// Provider auto-detection + runtime switching.
// ═══════════════════════════════════════════════════════════════════════════
package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// ProviderType — тип AI провайдера.
type ProviderType string

const (
	ProviderDeepSeek ProviderType = "deepseek"
	ProviderOpenAI   ProviderType = "openai"
	ProviderClaude   ProviderType = "claude"
	ProviderOllama   ProviderType = "ollama"
	ProviderVLLM     ProviderType = "vllm"
)

// ChatMessage — сообщение чата.
type ChatMessage struct {
	Role    string `json:"role"` // "user", "assistant", "system"
	Content string `json:"content"`
}

// ChatRequest — запрос к AI провайдеру.
type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// ChatResponse — ответ от AI провайдера.
type ChatResponse struct {
	Content string `json:"content"`
	Model   string `json:"model"`
	Done    bool   `json:"done"`
}

// Provider — интерфейс AI провайдера.
type Provider interface {
	// Type возвращает тип провайдера.
	Type() ProviderType

	// Chat отправляет запрос и возвращает ответ.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream отправляет запрос и возвращает streaming ответ.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error)

	// Models возвращает список доступных моделей.
	Models(ctx context.Context) ([]string, error)

	// IsAvailable проверяет доступность провайдера.
	IsAvailable(ctx context.Context) bool
}

// ────────────────────────────────────────────────────────────────────────────
// Provider Registry
// ────────────────────────────────────────────────────────────────────────────

// ProviderConfig — конфигурация провайдера.
type ProviderConfig struct {
	Type    ProviderType `json:"type"`
	APIKey  string       `json:"api_key,omitempty"`
	BaseURL string       `json:"base_url,omitempty"`
	Model   string       `json:"model,omitempty"`
	Order   int          `json:"order"` // приоритет автоматического выбора
}

// Registry — реестр AI провайдеров.
type Registry struct {
	mu        sync.RWMutex
	providers map[ProviderType]Provider
	order     []ProviderType // порядок auto-detection
	cfg       []ProviderConfig
	log       *slog.Logger
}

// ErrNoProviderAvailable — ни один провайдер не доступен.
var ErrNoProviderAvailable = errors.New("no AI provider available")

// NewRegistry создаёт реестр провайдеров.
func NewRegistry(configs []ProviderConfig, log *slog.Logger) (*Registry, error) {
	if log == nil {
		log = slog.Default()
	}
	r := &Registry{
		providers: make(map[ProviderType]Provider),
		cfg:       configs,
		log:       log.With("component", "ai-registry"),
	}

	for _, cfg := range configs {
		p, err := r.createProvider(cfg)
		if err != nil {
			log.Warn("failed to create AI provider", "type", cfg.Type, "error", err)
			continue
		}
		r.providers[cfg.Type] = p
		r.order = append(r.order, cfg.Type)
	}

	if len(r.providers) == 0 {
		return nil, ErrNoProviderAvailable
	}

	return r, nil
}

// createProvider создаёт провайдера по конфигурации.
func (r *Registry) createProvider(cfg ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case ProviderDeepSeek:
		return NewDeepSeekProvider(DeepSeekConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, r.log), nil
	case ProviderOpenAI:
		return NewOpenAIProvider(OpenAIConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, r.log), nil
	case ProviderClaude:
		return NewClaudeProvider(ClaudeConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, r.log), nil
	case ProviderOllama:
		return NewOllamaProvider(OllamaConfig{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, r.log), nil
	case ProviderVLLM:
		return NewVLLMProvider(VLLMConfig{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		}, r.log), nil
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}

// Get возвращает провайдера по типу.
func (r *Registry) Get(pt ProviderType) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[pt]
	return p, ok
}

// GetAvailable находит первый доступный провайдер (по порядку конфигурации).
func (r *Registry) GetAvailable(ctx context.Context) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, pt := range r.order {
		if p, ok := r.providers[pt]; ok {
			if p.IsAvailable(ctx) {
				return p, nil
			}
		}
	}
	return nil, ErrNoProviderAvailable
}

// List возвращает список зарегистрированных провайдеров.
func (r *Registry) List() []ProviderType {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]ProviderType, len(r.order))
	copy(types, r.order)
	return types
}

// Chat отправляет запрос первому доступному провайдеру.
func (r *Registry) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	p, err := r.GetAvailable(ctx)
	if err != nil {
		return nil, err
	}
	r.log.Debug("ai chat", "provider", p.Type(), "model", req.Model)
	return p.Chat(ctx, req)
}

// ChatStream отправляет streaming запрос первому доступному провайдеру.
func (r *Registry) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatResponse, error) {
	p, err := r.GetAvailable(ctx)
	if err != nil {
		return nil, err
	}
	return p.ChatStream(ctx, req)
}

// ────────────────────────────────────────────────────────────────────────────
// Streaming helpers
// ────────────────────────────────────────────────────────────────────────────

// StreamToWriter читает streaming response и пишет в writer.
func StreamToWriter(ctx context.Context, stream <-chan ChatResponse, w io.Writer) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case resp, ok := <-stream:
			if !ok {
				return nil
			}
			if _, err := io.WriteString(w, resp.Content); err != nil {
				return err
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}
