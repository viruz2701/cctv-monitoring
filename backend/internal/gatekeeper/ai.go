package gatekeeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"gb-telemetry-collector/internal/ai"
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
	// maxImageFetchSize — максимальный размер скачиваемого изображения (50 MB).
	maxImageFetchSize int64 = 50 * 1024 * 1024
	// imageFetchTimeout — таймаут для скачивания изображения.
	imageFetchTimeout = 10 * time.Second
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

// visionGuardInstance — глобальный экземпляр VisionGuard для gatekeeper.
// Использует default config (warn mode). Strict mode включается через env.
var visionGuardInstance *ai.VisionGuard

func init() {
	cfg := ai.DefaultVisionGuardConfig()
	if os.Getenv("VISION_GUARD_STRICT") == "true" {
		cfg.StrictMode = true
	}
	visionGuardInstance = ai.NewVisionGuard(cfg, slog.Default())
}

// fetchImage скачивает изображение по URL для проверки Vision Guard.
func fetchImage(ctx context.Context, url string) ([]byte, error) {
	fetchCtx, cancel := context.WithTimeout(ctx, imageFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create fetch request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch image: status %d", resp.StatusCode)
	}

	// Ограничиваем размер
	limitedReader := io.LimitReader(resp.Body, maxImageFetchSize)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	return data, nil
}

// VerifyAI выполняет AI-сравнение фото ДО и ПОСЛЕ через DeepSeek Vision API.
// Перед отправкой проверяет фото через Vision Guard на наличие:
//   - QR-кодов и barcode (prompt injection vector)
//   - Встроенного текста (adversarial payload)
//
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

	// ═══════════════════════════════════════════════════════════════════
	// P0-CR-09: Vision Guard — проверка фото перед отправкой в AI
	// ═══════════════════════════════════════════════════════════════════
	guardWarnings := []string{}

	// Проверяем фото "до" (before)
	beforeData, err := fetchImage(ctx, req.PhotoBeforeURL)
	if err != nil {
		slog.Warn("vision guard: failed to fetch before image, skipping check",
			"url", req.PhotoBeforeURL, "error", err)
	} else {
		if result := visionGuardInstance.CheckImage(beforeData); result != nil {
			if result.Suspicious {
				slog.Warn("vision guard: suspicious before-image detected",
					"url", req.PhotoBeforeURL,
					"qr_detected", result.QRDetected,
					"text_detected", result.TextDetected,
					"warnings", result.Warnings,
				)
				guardWarnings = append(guardWarnings,
					fmt.Sprintf("before-photo: %s", result.Warnings))
			}
		}
	}

	// Проверяем фото "после" (after)
	afterData, err := fetchImage(ctx, req.PhotoAfterURL)
	if err != nil {
		slog.Warn("vision guard: failed to fetch after image, skipping check",
			"url", req.PhotoAfterURL, "error", err)
	} else {
		if result := visionGuardInstance.CheckImage(afterData); result != nil {
			if result.Suspicious {
				slog.Warn("vision guard: suspicious after-image detected",
					"url", req.PhotoAfterURL,
					"qr_detected", result.QRDetected,
					"text_detected", result.TextDetected,
					"warnings", result.Warnings,
				)
				guardWarnings = append(guardWarnings,
					fmt.Sprintf("after-photo: %s", result.Warnings))
			}
		}
	}

	// Если strict mode и фото подозрительные — reject
	if len(guardWarnings) > 0 {
		cfg := ai.DefaultVisionGuardConfig()
		if os.Getenv("VISION_GUARD_STRICT") == "true" {
			cfg.StrictMode = true
		}
		if cfg.StrictMode {
			return AIResult{
				Passed: false,
				Error:  fmt.Sprintf("vision guard rejected: %v", guardWarnings),
			}
		}
	}

	// ═══════════════════════════════════════════════════════════════════
	// Отправка в DeepSeek Vision API
	// ═══════════════════════════════════════════════════════════════════
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

	// Добавляем warning о подозрительных фото в результат (если есть)
	if len(guardWarnings) > 0 {
		if result.Error != "" {
			result.Error += "; " + fmt.Sprintf("vision guard warnings: %v", guardWarnings)
		} else {
			result.Summary += fmt.Sprintf(" | VISION GUARD: %v", guardWarnings)
		}
	}

	return result
}
