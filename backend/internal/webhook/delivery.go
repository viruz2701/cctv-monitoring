// Package webhook — delivery worker for outgoing webhooks with retry + exponential backoff.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.3: Webhook Retry & Delivery Logs UI
//
// DeliveryWorker:
//   - Отправляет вебхуки с exponential backoff (10s, 30s, 90s, 270s...)
//   - Сохраняет логи доставки в БД
//   - Graceful shutdown через context
//   - Метрики: delivery_count, delivery_success, delivery_failed, retry_count
//
// Compliance:
//   - IEC 62443 SR 7.1 (Resource availability — delivery retry)
//   - ISO 27001 A.12.4.1 (Event logging — delivery audit trail)
//   - OWASP ASVS V7.1 (Error handling — no sensitive data in logs)
//
// ═══════════════════════════════════════════════════════════════════════════
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// Constants
// ────────────────────────────────────────────────────────────────────────────

const (
	// DefaultPollInterval — интервал опроса pending доставок.
	DefaultPollInterval = 5 * time.Second

	// BaseRetryDelay — базовая задержка для exponential backoff.
	BaseRetryDelay = 10 * time.Second

	// MaxRetryDelay — максимальная задержка между retry.
	MaxRetryDelay = 1 * time.Hour

	// MaxResponseBody — максимальный размер сохраняемого response body.
	MaxResponseBody = 64 * 1024 // 64KB
)

// ────────────────────────────────────────────────────────────────────────────
// Types
// ────────────────────────────────────────────────────────────────────────────

// DeliveryStore — интерфейс для хранения и загрузки delivery логов.
type DeliveryStore interface {
	GetPendingDeliveries(ctx context.Context, limit int) ([]DeliveryLog, error)
	CreateDeliveryLog(ctx context.Context, log *DeliveryLog) error
	UpdateDeliveryLog(ctx context.Context, id string, status string, responseStatus int, responseBody, errorMsg string, durationMs int, nextRetryAt *time.Time) error
	CreateWebhookEndpoint(ctx context.Context, wh *WebhookEndpoint) error
	UpdateWebhookEndpoint(ctx context.Context, id string, wh *WebhookEndpoint) error
	DeleteWebhookEndpoint(ctx context.Context, id string) error
	ListWebhookEndpoints(ctx context.Context) ([]WebhookEndpoint, error)
	GetWebhookEndpoint(ctx context.Context, id string) (*WebhookEndpoint, error)
	GetDeliveryLogs(ctx context.Context, webhookID string, limit, offset int) ([]DeliveryLog, error)
}

// WebhookEndpoint — настройки исходящего вебхука.
type WebhookEndpoint struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Secret         string   `json:"secret,omitempty"`
	Events         []string `json:"events"`
	Enabled        bool     `json:"enabled"`
	RetryCount     int      `json:"retry_count"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// DeliveryLog — запись о доставке вебхука.
type DeliveryLog struct {
	ID             string     `json:"id"`
	WebhookID      string     `json:"webhook_id"`
	EventType      string     `json:"event_type"`
	Status         string     `json:"status"`
	RequestURL     string     `json:"request_url"`
	RequestBody    string     `json:"request_body"`
	ResponseStatus int        `json:"response_status"`
	ResponseBody   string     `json:"response_body"`
	DurationMs     int        `json:"duration_ms"`
	RetryAttempt   int        `json:"retry_attempt"`
	MaxRetries     int        `json:"max_retries"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

// ────────────────────────────────────────────────────────────────────────────
// DeliveryMetrics
// ────────────────────────────────────────────────────────────────────────────

type DeliveryMetrics struct {
	deliveryCount atomic.Int64
	successCount  atomic.Int64
	failedCount   atomic.Int64
	retryCount    atomic.Int64
}

func (m *DeliveryMetrics) IncDelivery() { m.deliveryCount.Add(1) }
func (m *DeliveryMetrics) IncSuccess()  { m.successCount.Add(1) }
func (m *DeliveryMetrics) IncFailed()   { m.failedCount.Add(1) }
func (m *DeliveryMetrics) IncRetry()    { m.retryCount.Add(1) }

func (m *DeliveryMetrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"delivery_count": m.deliveryCount.Load(),
		"success_count":  m.successCount.Load(),
		"failed_count":   m.failedCount.Load(),
		"retry_count":    m.retryCount.Load(),
	}
}

// ────────────────────────────────────────────────────────────────────────────
// DeliveryWorker
// ────────────────────────────────────────────────────────────────────────────

type DeliveryWorkerConfig struct {
	PollInterval  time.Duration
	MaxConcurrent int
}

type DeliveryWorker struct {
	store   DeliveryStore
	client  *http.Client
	metrics *DeliveryMetrics
	logger  *slog.Logger
	cfg     DeliveryWorkerConfig

	mu      sync.Mutex
	started bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

func NewDeliveryWorker(store DeliveryStore, logger *slog.Logger, cfg DeliveryWorkerConfig) *DeliveryWorker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultPollInterval
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 5
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &DeliveryWorker{
		store:   store,
		client:  &http.Client{Timeout: 30 * time.Second},
		metrics: &DeliveryMetrics{},
		logger:  logger.With("component", "webhook-delivery-worker"),
		cfg:     cfg,
		stopCh:  make(chan struct{}),
	}
}

func (w *DeliveryWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.started {
		w.mu.Unlock()
		return
	}
	w.started = true
	w.mu.Unlock()

	w.logger.Info("webhook delivery worker started",
		"poll_interval", w.cfg.PollInterval,
		"max_concurrent", w.cfg.MaxConcurrent,
	)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processPending(ctx)
		case <-w.stopCh:
			w.logger.Info("webhook delivery worker stopping")
			w.wg.Wait()
			return
		case <-ctx.Done():
			w.logger.Info("webhook delivery worker stopped via context")
			w.wg.Wait()
			return
		}
	}
}

func (w *DeliveryWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		close(w.stopCh)
		w.started = false
	}
}

func (w *DeliveryWorker) Metrics() *DeliveryMetrics {
	return w.metrics
}

func (w *DeliveryWorker) Deliver(ctx context.Context, wh *WebhookEndpoint, eventType string, payload interface{}) (*DeliveryLog, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("webhook: marshal payload: %w", err)
	}

	entry := &DeliveryLog{
		WebhookID:    wh.ID,
		EventType:    eventType,
		Status:       "pending",
		RequestURL:   wh.URL,
		RequestBody:  string(bodyBytes),
		MaxRetries:   wh.RetryCount,
		RetryAttempt: 0,
	}

	if err := w.store.CreateDeliveryLog(ctx, entry); err != nil {
		return nil, fmt.Errorf("webhook: create delivery log: %w", err)
	}

	w.performDelivery(ctx, entry, wh.Secret)
	return entry, nil
}

// ── Internal ─────────────────────────────────────────────────────────────

func (w *DeliveryWorker) processPending(ctx context.Context) {
	pending, err := w.store.GetPendingDeliveries(ctx, w.cfg.MaxConcurrent)
	if err != nil {
		w.logger.Error("failed to fetch pending deliveries", "error", err)
		return
	}

	for i := range pending {
		w.wg.Add(1)
		go func(dl *DeliveryLog) {
			defer w.wg.Done()

			endpoint, err := w.store.GetWebhookEndpoint(ctx, dl.WebhookID)
			if err != nil {
				w.logger.Error("failed to get webhook endpoint", "id", dl.WebhookID, "error", err)
				return
			}

			w.performDelivery(ctx, dl, endpoint.Secret)
		}(&pending[i])
	}
}

func (w *DeliveryWorker) performDelivery(ctx context.Context, dl *DeliveryLog, secret string) {
	w.metrics.IncDelivery()

	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dl.RequestURL, bytes.NewReader([]byte(dl.RequestBody)))
	if err != nil {
		w.updateFailed(ctx, dl, 0, "", fmt.Sprintf("create request: %v", err), start)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CCTV-Health-Monitor-Webhook/1.0")

	if secret != "" {
		sig := hex.EncodeToString(hmacSignature(secret, []byte(dl.RequestBody)))
		req.Header.Set("X-Signature-256", sig)
	}

	resp, err := w.client.Do(req)
	durationMs := int(time.Since(start).Milliseconds())

	if err != nil {
		w.updateFailed(ctx, dl, 0, "", fmt.Sprintf("http request failed: %v", err), start)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBody))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.metrics.IncSuccess()
		if err := w.store.UpdateDeliveryLog(ctx, dl.ID, "success",
			resp.StatusCode, string(respBody), "", durationMs, nil); err != nil {
			w.logger.Error("failed to update delivery log", "id", dl.ID, "error", err)
		}

		w.logger.Info("webhook delivered successfully",
			"id", dl.ID, "url", dl.RequestURL,
			"status", resp.StatusCode, "duration_ms", durationMs,
		)
	} else {
		w.metrics.IncFailed()

		dl.RetryAttempt++
		nextRetry := w.scheduleRetry(dl)
		errMsg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, truncateString(string(respBody), 256))

		status := "failed"
		if nextRetry == nil {
			status = "cancelled"
		}

		if err := w.store.UpdateDeliveryLog(ctx, dl.ID, status,
			resp.StatusCode, string(respBody), errMsg, durationMs, nextRetry); err != nil {
			w.logger.Error("failed to update delivery log", "id", dl.ID, "error", err)
		}

		w.logger.Warn("webhook delivery failed",
			"id", dl.ID, "url", dl.RequestURL,
			"status", resp.StatusCode, "attempt", dl.RetryAttempt,
			"next_retry", nextRetry,
		)
	}
}

func (w *DeliveryWorker) updateFailed(ctx context.Context, dl *DeliveryLog, statusCode int, respBody, errMsg string, start time.Time) {
	w.metrics.IncFailed()

	dl.RetryAttempt++
	durationMs := int(time.Since(start).Milliseconds())
	nextRetry := w.scheduleRetry(dl)

	status := "failed"
	if nextRetry == nil {
		status = "cancelled"
	}

	if err := w.store.UpdateDeliveryLog(ctx, dl.ID, status,
		statusCode, respBody, errMsg, durationMs, nextRetry); err != nil {
		w.logger.Error("failed to update delivery log", "id", dl.ID, "error", err)
	}
}

func (w *DeliveryWorker) scheduleRetry(dl *DeliveryLog) *time.Time {
	if dl.RetryAttempt >= dl.MaxRetries {
		return nil
	}

	w.metrics.IncRetry()

	delay := float64(BaseRetryDelay) * math.Pow(3, float64(dl.RetryAttempt-1))
	if delay > float64(MaxRetryDelay) {
		delay = float64(MaxRetryDelay)
	}

	next := time.Now().Add(time.Duration(delay))
	return &next
}

func hmacSignature(secret string, body []byte) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return mac.Sum(nil)
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
