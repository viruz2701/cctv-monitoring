// Package webhook — PostgreSQL реализация DeliveryStore.
//
// ═══════════════════════════════════════════════════════════════════════════
// P2-3.3: Webhook Retry & Delivery Logs UI
//
// PGDeliveryStore реализует интерфейс DeliveryStore через PostgreSQL.
// Использует таблицы: webhook_endpoints, webhook_delivery_logs
// ═══════════════════════════════════════════════════════════════════════════
package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGDeliveryStore — PostgreSQL реализация DeliveryStore.
type PGDeliveryStore struct {
	pool *pgxpool.Pool
}

// NewPGDeliveryStore создаёт новый PGDeliveryStore.
func NewPGDeliveryStore(pool *pgxpool.Pool) *PGDeliveryStore {
	return &PGDeliveryStore{pool: pool}
}

// ── Delivery Logs ──────────────────────────────────────────────────────

func (s *PGDeliveryStore) GetPendingDeliveries(ctx context.Context, limit int) ([]DeliveryLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, webhook_id, event_type, status, request_url, request_body,
		       response_status, response_body, duration_ms, retry_attempt,
		       max_retries, COALESCE(error_message, ''), next_retry_at,
		       created_at, updated_at
		FROM webhook_delivery_logs
		WHERE status = 'pending'
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending deliveries: %w", err)
	}
	defer rows.Close()

	return scanDeliveryLogs(rows)
}

func (s *PGDeliveryStore) CreateDeliveryLog(ctx context.Context, dl *DeliveryLog) error {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_delivery_logs
			(webhook_id, event_type, status, request_url, request_body,
			 response_status, response_body, duration_ms, retry_attempt,
			 max_retries, error_message, next_retry_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`, dl.WebhookID, dl.EventType, dl.Status, dl.RequestURL, dl.RequestBody,
		dl.ResponseStatus, dl.ResponseBody, dl.DurationMs, dl.RetryAttempt,
		dl.MaxRetries, dl.ErrorMessage, dl.NextRetryAt,
	).Scan(&id, &dl.CreatedAt, &dl.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create delivery log: %w", err)
	}
	dl.ID = id
	return nil
}

func (s *PGDeliveryStore) UpdateDeliveryLog(ctx context.Context, id string, status string, responseStatus int, responseBody, errorMsg string, durationMs int, nextRetryAt *time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_delivery_logs SET
			status = $1, response_status = $2, response_body = $3,
			error_message = $4, duration_ms = $5, next_retry_at = $6,
			retry_attempt = retry_attempt + 1
		WHERE id = $7
	`, status, responseStatus, responseBody, errorMsg, durationMs, nextRetryAt, id)
	if err != nil {
		return fmt.Errorf("update delivery log %s: %w", id, err)
	}
	return nil
}

func (s *PGDeliveryStore) GetDeliveryLogs(ctx context.Context, webhookID string, limit, offset int) ([]DeliveryLog, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, webhook_id, event_type, status, request_url, request_body,
		       response_status, response_body, duration_ms, retry_attempt,
		       max_retries, COALESCE(error_message, ''), next_retry_at,
		       created_at, updated_at
		FROM webhook_delivery_logs
		WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, webhookID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get delivery logs: %w", err)
	}
	defer rows.Close()

	return scanDeliveryLogs(rows)
}

// ── Webhook Endpoints ─────────────────────────────────────────────────

func (s *PGDeliveryStore) CreateWebhookEndpoint(ctx context.Context, wh *WebhookEndpoint) error {
	var id, createdAt, updatedAt string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_endpoints (name, url, secret, events, enabled, retry_count, timeout_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, wh.Name, wh.URL, wh.Secret, wh.Events, wh.Enabled, wh.RetryCount, wh.TimeoutSeconds,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("create webhook endpoint: %w", err)
	}
	wh.ID = id
	wh.CreatedAt = createdAt
	wh.UpdatedAt = updatedAt
	return nil
}

func (s *PGDeliveryStore) UpdateWebhookEndpoint(ctx context.Context, id string, wh *WebhookEndpoint) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_endpoints SET
			name = $1, url = $2, secret = $3, events = $4,
			enabled = $5, retry_count = $6, timeout_seconds = $7
		WHERE id = $8
	`, wh.Name, wh.URL, wh.Secret, wh.Events, wh.Enabled, wh.RetryCount, wh.TimeoutSeconds, id)
	if err != nil {
		return fmt.Errorf("update webhook endpoint %s: %w", id, err)
	}
	return nil
}

func (s *PGDeliveryStore) DeleteWebhookEndpoint(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM webhook_endpoints WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete webhook endpoint %s: %w", id, err)
	}
	return nil
}

func (s *PGDeliveryStore) ListWebhookEndpoints(ctx context.Context) ([]WebhookEndpoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, url, COALESCE(secret, ''), events, enabled,
		       retry_count, timeout_seconds, created_at, updated_at
		FROM webhook_endpoints
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list webhook endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []WebhookEndpoint
	for rows.Next() {
		var wh WebhookEndpoint
		if err := rows.Scan(&wh.ID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events,
			&wh.Enabled, &wh.RetryCount, &wh.TimeoutSeconds, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook endpoint: %w", err)
		}
		endpoints = append(endpoints, wh)
	}
	return endpoints, rows.Err()
}

func (s *PGDeliveryStore) GetWebhookEndpoint(ctx context.Context, id string) (*WebhookEndpoint, error) {
	var wh WebhookEndpoint
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, url, COALESCE(secret, ''), events, enabled,
		       retry_count, timeout_seconds, created_at, updated_at
		FROM webhook_endpoints
		WHERE id = $1
	`, id).Scan(&wh.ID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events,
		&wh.Enabled, &wh.RetryCount, &wh.TimeoutSeconds, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get webhook endpoint %s: %w", id, err)
	}
	return &wh, nil
}

// ── Scanner ────────────────────────────────────────────────────────────

func scanDeliveryLogs(rows pgx.Rows) ([]DeliveryLog, error) {
	var logs []DeliveryLog
	for rows.Next() {
		var dl DeliveryLog
		var nextRetryAt *time.Time
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&dl.ID, &dl.WebhookID, &dl.EventType, &dl.Status,
			&dl.RequestURL, &dl.RequestBody, &dl.ResponseStatus, &dl.ResponseBody,
			&dl.DurationMs, &dl.RetryAttempt, &dl.MaxRetries, &dl.ErrorMessage,
			&nextRetryAt, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan delivery log: %w", err)
		}

		dl.NextRetryAt = nextRetryAt
		dl.CreatedAt = createdAt.Format(time.RFC3339)
		dl.UpdatedAt = updatedAt.Format(time.RFC3339)
		logs = append(logs, dl)
	}
	if logs == nil {
		logs = []DeliveryLog{}
	}
	return logs, rows.Err()
}
