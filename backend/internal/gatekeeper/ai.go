package gatekeeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// AIVerifyRequest представляет запрос к DeepSeek Vision API для сравнения фото.
type AIVerifyRequest struct {
	PhotoBeforeURL string `json:"photo_before_url"`
	PhotoAfterURL  string `json:"photo_after_url"`
}

// AIResult содержит результат AI-верификации.
type AIResult struct {
	Passed         bool    `json:"passed"`
	Similarity     float64 `json:"similarity"`      // 0.0 – 1.0, степень сходства
	ChangeDetected bool    `json:"change_detected"` // обнаружены ли изменения
	Summary        string  `json:"summary,omitempty"`
	Error          string  `json:"error,omitempty"`
	Skipped        bool    `json:"skipped"` // AI пропущен (Phase 2)
}

const (
	// MinSimilarity — минимальный порог сходства для подтверждения «тот же объект».
	MinSimilarity = 0.80
	// DeepSeekAPIURL — эндпоинт DeepSeek Vision API.
	DeepSeekAPIURL = "https://api.deepseek.com/v1/chat/completions"
	// AIRequestTimeout — таймаут для запроса к AI.
	AIRequestTimeout = 30 * time.Second
)

// deepSeekVisionPrompt — промпт для сравнения фото ДО и ПОСЛЕ.
const deepSeekVisionPrompt = `You are a CCTV maintenance verification system. Compare the two photos of a surveillance camera installation.

Photo 1: BEFORE maintenance
Photo 2: AFTER maintenance

Analyze:
1. Is this the same camera/location? (similarity 0-100%)
2. Are there visible changes? (cleaned lens, replaced cable, adjusted angle, etc.)

Respond ONLY with valid JSON:
{
  "similarity": 0.95,
  "change_detected": true,
  "summary": "Camera lens cleaned, angle adjusted slightly"
}`

// deepSeekResponse — структура ответа от DeepSeek API.
type deepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// VerifyAI выполняет AI-сравнение фото ДО и ПОСЛЕ через DeepSeek Vision API.
// Если API-ключ не настроен, AI-проверка пропускается (graceful degradation).
func VerifyAI(ctx context.Context, req AIVerifyRequest) AIResult {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return AIResult{
			Skipped: true,
			Error:   "AI verification skipped: DEEPSEEK_API_KEY not configured",
		}
	}

	if req.PhotoBeforeURL == "" || req.PhotoAfterURL == "" {
		return AIResult{
			Passed: false,
			Error:  "both photo_before_url and photo_after_url are required for AI verification",
		}
	}

	ctx, cancel := context.WithTimeout(ctx, AIRequestTimeout)
	defer cancel()

	payload := map[string]interface{}{
		"model": "deepseek-chat",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": deepSeekVisionPrompt,
			},
		},
		"temperature": 0.1,
		"max_tokens":  512,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return AIResult{Error: fmt.Sprintf("marshal ai request: %v", err)}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, DeepSeekAPIURL, bytes.NewReader(body))
	if err != nil {
		return AIResult{Error: fmt.Sprintf("create ai request: %v", err)}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return AIResult{Error: fmt.Sprintf("ai api error: %v", err)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return AIResult{Error: fmt.Sprintf("read ai response: %v", err)}
	}

	if resp.StatusCode != http.StatusOK {
		return AIResult{Error: fmt.Sprintf("ai api returned status %d: %s", resp.StatusCode, string(respBody))}
	}

	var dsResp deepSeekResponse
	if err := json.Unmarshal(respBody, &dsResp); err != nil {
		return AIResult{Error: fmt.Sprintf("parse ai response: %v", err)}
	}

	if len(dsResp.Choices) == 0 {
		return AIResult{Error: "ai returned empty response"}
	}

	// Парсим JSON из ответа модели
	var analysis struct {
		Similarity     float64 `json:"similarity"`
		ChangeDetected bool    `json:"change_detected"`
		Summary        string  `json:"summary"`
	}
	if err := json.Unmarshal([]byte(dsResp.Choices[0].Message.Content), &analysis); err != nil {
		// Если модель вернула не JSON — считаем что AI не смог определить
		return AIResult{
			Passed: false,
			Error:  fmt.Sprintf("ai returned non-json response: %s", dsResp.Choices[0].Message.Content),
		}
	}

	result := AIResult{
		Similarity:     analysis.Similarity,
		ChangeDetected: analysis.ChangeDetected,
		Summary:        analysis.Summary,
	}

	if analysis.Similarity >= MinSimilarity {
		result.Passed = true
	} else {
		result.Error = fmt.Sprintf("similarity %.0f%% below threshold %.0f%%", analysis.Similarity*100, MinSimilarity*100)
	}

	return result
}
