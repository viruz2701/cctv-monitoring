// Package api — AI Assistant Chat proxy (DeepSeek SSE streaming).
//
// P2-1.2: AI Assistant Chat
//   - Backend proxy для DeepSeek API (API key не светится на клиенте)
//   - Server-Sent Events streaming ответов
//   - Контекст: current page, device_id, wo_id, tenant
//   - Feedback: thumbs up/down (сохраняется в audit_log)
//
// Compliance:
//   - OWASP ASVS V3 (Session Management — JWT)
//   - OWASP ASVS V5 (Input Validation — whitelist)
//   - IEC 62443 SR 7.1 (Resource availability — timeout controls)
//   - СТБ 34.101.27 (Audit trail — feedback logging)
//   - ISO 27001 A.12.4.1 (Event logging)
package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	deepSeekChatURL    = "https://api.deepseek.com/v1/chat/completions"
	aiChatTimeout      = 60 * time.Second
	aiChatModel        = "deepseek-chat"
	maxMessageLength   = 4096
	maxHistoryMessages = 20
)

// ─── Types ────────────────────────────────────────────────────────────

// AIChatRequest — запрос от клиента к backend proxy.
type AIChatRequest struct {
	Message string        `json:"message"`
	Context AIChatContext `json:"context"`
}

// AIChatContext — контекст текущей страницы.
type AIChatContext struct {
	CurrentPage string `json:"current_page,omitempty"`
	DeviceID    string `json:"device_id,omitempty"`
	WorkOrderID string `json:"wo_id,omitempty"`
	SiteID      string `json:"site_id,omitempty"`
}

// AIChatMessage — сообщение в истории чата.
type AIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AIChatHistoryRequest — запрос с историей для потокового ответа.
type AIChatHistoryRequest struct {
	Message string          `json:"message"`
	Context AIChatContext   `json:"context"`
	History []AIChatMessage `json:"history"`
}

// AIFeedbackRequest — запрос обратной связи.
type AIFeedbackRequest struct {
	MessageID string `json:"message_id"`
	Score     string `json:"score"` // "like" | "dislike"
}

// deepSeekMessage — сообщение для DeepSeek API.
type deepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// deepSeekRequest — запрос к DeepSeek Chat API.
type deepSeekRequest struct {
	Model       string            `json:"model"`
	Messages    []deepSeekMessage `json:"messages"`
	Stream      bool              `json:"stream"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
}

// deepSeekStreamChunk — чанк SSE ответа от DeepSeek.
type deepSeekStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// aiSystemPrompt — системный промпт для AI Assistant.
const aiSystemPrompt = `You are an AI assistant for CCTV Health Monitor, a surveillance system management platform. You help technicians, operators, and administrators with:

1. **Device Diagnostics**: Analyzing camera health, connectivity issues, video quality problems
2. **Work Order Management**: Explaining maintenance procedures, SLA policies, repair workflows
3. **Root Cause Analysis**: Suggesting possible causes for device failures based on symptoms
4. **Compliance Guidance**: Answering questions about security standards (IEC 62443, ISO 27001, OAC #66)
5. **System Operations**: Guiding users through common tasks and troubleshooting

Rules:
- Be concise and technical. Use Markdown formatting for clarity.
- When suggesting RCA, provide a structured analysis with possible causes ordered by probability.
- When discussing work orders, reference the WO ID if provided in context.
- When discussing devices, reference the device ID if provided.
- If you don't know something, say so — do not hallucinate.
- Never reveal API keys, credentials, or internal configuration.
- Always prioritize security and compliance in recommendations.
- Current context (page, device, work order) is provided when available — use it to give relevant answers.`

// ─── Routes ───────────────────────────────────────────────────────────

// mountAIRoutes регистрирует AI Assistant маршруты в защищённой группе.
func (s *Server) mountAIRoutes(r chi.Router) {
	r.Post("/api/v1/ai/chat", s.handleAIChat)         // Streaming chat
	r.Post("/api/v1/ai/feedback", s.handleAIFeedback) // Feedback
}

// ─── Handlers ─────────────────────────────────────────────────────────

// handleAIChat обрабатывает чат-запрос и стримит ответ через SSE.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — message length check)
//   - OWASP ASVS V7.1 (Error handling — no information leakage)
//   - IEC 62443 SR 7.1 (Timeout — 60s max)
func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Parse request
	var req AIChatHistoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request body: "+err.Error()))
		return
	}

	// 2. Validate input (OWASP ASVS V5.1 — whitelist validation)
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		respondError(w, r, NewValidationError("message cannot be empty"))
		return
	}
	if len(req.Message) > maxMessageLength {
		respondError(w, r, NewValidationError(fmt.Sprintf("message too long (max %d characters)", maxMessageLength)))
		return
	}
	if len(req.History) > maxHistoryMessages {
		req.History = req.History[len(req.History)-maxHistoryMessages:]
	}

	// 3. Get API key from config
	apiKey := s.config.DeepSeekAPIKey
	if apiKey == "" {
		respondError(w, r, NewExternalServiceError("AI service not configured"))
		return
	}

	// 4. Build system prompt with context
	systemPrompt := aiSystemPrompt
	if req.Context.CurrentPage != "" || req.Context.DeviceID != "" || req.Context.WorkOrderID != "" {
		systemPrompt += "\n\n## Current Context\n"
		if req.Context.CurrentPage != "" {
			systemPrompt += fmt.Sprintf("- Current page: %s\n", req.Context.CurrentPage)
		}
		if req.Context.DeviceID != "" {
			systemPrompt += fmt.Sprintf("- Device ID: %s\n", req.Context.DeviceID)
		}
		if req.Context.WorkOrderID != "" {
			systemPrompt += fmt.Sprintf("- Work Order ID: %s\n", req.Context.WorkOrderID)
		}
		if req.Context.SiteID != "" {
			systemPrompt += fmt.Sprintf("- Site ID: %s\n", req.Context.SiteID)
		}
		systemPrompt += "\nUse this context to provide relevant assistance."
	}

	// 5. Build messages array
	messages := []deepSeekMessage{
		{Role: "system", Content: systemPrompt},
	}
	for _, h := range req.History {
		messages = append(messages, deepSeekMessage{Role: h.Role, Content: h.Content})
	}
	messages = append(messages, deepSeekMessage{Role: "user", Content: req.Message})

	// 6. Call DeepSeek API with streaming
	s.streamDeepSeekResponse(ctx, w, apiKey, messages)
}

// streamDeepSeekResponse вызывает DeepSeek API и стримит ответ через SSE.
func (s *Server) streamDeepSeekResponse(ctx context.Context, w http.ResponseWriter, apiKey string, messages []deepSeekMessage) {
	dsReq := deepSeekRequest{
		Model:       aiChatModel,
		Messages:    messages,
		Stream:      true,
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	body, err := json.Marshal(dsReq)
	if err != nil {
		s.writeSSEError(w, "failed to process request")
		return
	}

	// Create HTTP request to DeepSeek
	dsHTTPReq, err := http.NewRequestWithContext(ctx, http.MethodPost, deepSeekChatURL, bytes.NewReader(body))
	if err != nil {
		s.writeSSEError(w, "failed to create AI request")
		return
	}
	dsHTTPReq.Header.Set("Content-Type", "application/json")
	dsHTTPReq.Header.Set("Authorization", "Bearer "+apiKey)
	dsHTTPReq.Header.Set("Accept", "text/event-stream")

	// Send request
	client := &http.Client{Timeout: aiChatTimeout}
	dsResp, err := client.Do(dsHTTPReq)
	if err != nil {
		s.writeSSEError(w, "AI service unavailable")
		return
	}
	defer dsResp.Body.Close()

	if dsResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(dsResp.Body)
		s.logger.Error("DeepSeek API error",
			"status", dsResp.StatusCode,
			"body", string(respBody),
		)
		s.writeSSEError(w, "AI service error")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeSSEError(w, "streaming not supported")
		return
	}

	// Stream response
	scanner := bufio.NewScanner(dsResp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var fullResponse strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and SSE comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data: "data: {...}"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// DeepSeek sends "data: [DONE]" when complete
		if data == "[DONE]" {
			break
		}

		// Parse JSON chunk
		var chunk deepSeekStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		content := chunk.Choices[0].Delta.Content
		if content == "" {
			continue
		}

		fullResponse.WriteString(content)

		// Send SSE event to client
		sseEvent, _ := json.Marshal(map[string]string{
			"type":    "chunk",
			"content": content,
		})
		fmt.Fprintf(w, "data: %s\n\n", sseEvent)
		flusher.Flush()
	}

	if err := scanner.Err(); err != nil {
		s.logger.Error("SSE stream error", "error", err)
	}

	// Send completion event
	completionEvent, _ := json.Marshal(map[string]string{
		"type":    "done",
		"content": fullResponse.String(),
	})
	fmt.Fprintf(w, "data: %s\n\n", completionEvent)
	flusher.Flush()
}

// handleAIFeedback сохраняет обратную связь пользователя.
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging — feedback in audit trail)
//   - СТБ 34.101.27 (Audit trail)
func (s *Server) handleAIFeedback(w http.ResponseWriter, r *http.Request) {
	var req AIFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, r, NewBadRequestError("invalid request body: "+err.Error()))
		return
	}

	// Validate score
	if req.Score != "like" && req.Score != "dislike" {
		respondError(w, r, NewValidationError("invalid score value, must be 'like' or 'dislike'"))
		return
	}

	userID := getUserIDFromContext(r.Context())

	// Log feedback to audit trail
	s.logAudit(userID, "ai_feedback", "ai_assistant", req.MessageID, map[string]interface{}{
		"score": req.Score,
	}, nil)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// writeSSEError отправляет SSE событие с ошибкой.
func (s *Server) writeSSEError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	errEvent, _ := json.Marshal(map[string]string{
		"type":    "error",
		"content": message,
	})
	fmt.Fprintf(w, "data: %s\n\n", errEvent)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
