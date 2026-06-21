package cmms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FallbackQueueEntry — запись в очереди отложенной синхронизации.
type FallbackQueueEntry struct {
	ID        string          `json:"id"`
	Method    string          `json:"method"` // create_wo, update_wo, sync_asset, etc.
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	Retries   int             `json:"retries"`
	LastError string          `json:"last_error,omitempty"`
}

// FallbackQueue — персистентная очередь для операций, которые не удалось
// отправить в Atlas CMMS из-за недоступности API. Операции сохраняются
// на диск и повторяются при восстановлении связи.
type FallbackQueue struct {
	dir        string
	mu         sync.Mutex
	logger     *slog.Logger
	maxRetries int
}

// NewFallbackQueue создаёт новую fallback-очередь.
// dir — директория для хранения файлов очереди.
func NewFallbackQueue(dir string, maxRetries int, logger *slog.Logger) (*FallbackQueue, error) {
	if dir == "" {
		dir = "/var/lib/gb-telemetry/fallback"
	}
	if maxRetries <= 0 {
		maxRetries = 10
	}

	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("fallback queue: failed to create dir %s: %w", dir, err)
	}

	return &FallbackQueue{
		dir:        dir,
		logger:     logger,
		maxRetries: maxRetries,
	}, nil
}

// Enqueue добавляет операцию в очередь отложенной синхронизации.
func (fq *FallbackQueue) Enqueue(method string, payload interface{}) error {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("fallback: marshal payload: %w", err)
	}

	entry := FallbackQueueEntry{
		ID:        fmt.Sprintf("%s_%d", method, time.Now().UnixNano()),
		Method:    method,
		Payload:   payloadJSON,
		CreatedAt: time.Now(),
		Retries:   0,
	}

	entryPath := filepath.Join(fq.dir, entry.ID+".json")
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("fallback: marshal entry: %w", err)
	}

	if err := os.WriteFile(entryPath, data, 0640); err != nil {
		return fmt.Errorf("fallback: write entry: %w", err)
	}

	fq.logger.Info("fallback: enqueued operation", "method", method, "id", entry.ID)
	return nil
}

// Pending возвращает все записи, ожидающие обработки.
func (fq *FallbackQueue) Pending() ([]FallbackQueueEntry, error) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	entries, err := os.ReadDir(fq.dir)
	if err != nil {
		return nil, fmt.Errorf("fallback: read dir: %w", err)
	}

	var result []FallbackQueueEntry
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		entryPath := filepath.Join(fq.dir, e.Name())
		data, err := os.ReadFile(entryPath)
		if err != nil {
			fq.logger.Warn("fallback: failed to read entry", "file", e.Name(), "error", err)
			continue
		}
		var entry FallbackQueueEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			fq.logger.Warn("fallback: failed to parse entry", "file", e.Name(), "error", err)
			continue
		}
		result = append(result, entry)
	}
	return result, nil
}

// MarkRetry увеличивает счётчик попыток для записи.
func (fq *FallbackQueue) MarkRetry(entryID string, lastError string) error {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	entryPath := filepath.Join(fq.dir, entryID+".json")
	data, err := os.ReadFile(entryPath)
	if err != nil {
		return fmt.Errorf("fallback: read entry %s: %w", entryID, err)
	}

	var entry FallbackQueueEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("fallback: parse entry %s: %w", entryID, err)
	}

	entry.Retries++
	entry.LastError = lastError

	if entry.Retries >= fq.maxRetries {
		fq.logger.Error("fallback: max retries exceeded, removing entry",
			"id", entryID, "method", entry.Method, "retries", entry.Retries)
		return os.Remove(entryPath)
	}

	newData, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("fallback: marshal entry %s: %w", entryID, err)
	}

	return os.WriteFile(entryPath, newData, 0640)
}

// Remove удаляет успешно обработанную запись из очереди.
func (fq *FallbackQueue) Remove(entryID string) error {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	entryPath := filepath.Join(fq.dir, entryID+".json")
	if err := os.Remove(entryPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("fallback: remove entry %s: %w", entryID, err)
	}
	return nil
}

// Len возвращает количество записей в очереди.
func (fq *FallbackQueue) Len() (int, error) {
	entries, err := fq.Pending()
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// RetryFunc — функция, вызываемая для повторной отправки записи.
// Возвращает ошибку, если операция не удалась.
type RetryFunc func(ctx context.Context, entry FallbackQueueEntry) error

// RetryAll пытается повторно отправить все записи из очереди.
func (fq *FallbackQueue) RetryAll(ctx context.Context, fn RetryFunc) (success, failed int) {
	entries, err := fq.Pending()
	if err != nil {
		fq.logger.Error("fallback: failed to list pending entries", "error", err)
		return 0, 0
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return success, failed
		default:
		}

		if err := fn(ctx, entry); err != nil {
			failed++
			_ = fq.MarkRetry(entry.ID, err.Error())
			fq.logger.Warn("fallback: retry failed",
				"id", entry.ID, "method", entry.Method, "error", err)
		} else {
			success++
			_ = fq.Remove(entry.ID)
			fq.logger.Info("fallback: retry succeeded",
				"id", entry.ID, "method", entry.Method)
		}
	}
	return success, failed
}
