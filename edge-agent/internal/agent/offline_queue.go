package agent

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// QueuedMessage represents a message stored in the offline queue.
type QueuedMessage struct {
	// Topic is the MQTT topic for this message
	Topic string `json:"topic"`
	// Payload is the message payload
	Payload []byte `json:"payload"`
	// QueuedAt is when the message was queued
	QueuedAt time.Time `json:"queued_at"`
	// RetryCount tracks delivery attempts
	RetryCount int `json:"retry_count"`
}

// OfflineQueue provides persistent message queuing using BoltDB.
// Messages are stored when MQTT is disconnected and replayed on reconnect.
//
// Compliance: Приказ ОАЦ №66 п. 7.18 — сохранность данных при потере связи
//
//	IEC 62443-3-3 SL-3 — offline resilience
type OfflineQueue struct {
	db     *bolt.DB
	logger *slog.Logger
}

const (
	queueBucket = "queue"
	maxMessages = 1000
	maxRetries  = 10
)

// NewOfflineQueue creates or opens a BoltDB-based offline queue.
func NewOfflineQueue(path string, logger *slog.Logger) *OfflineQueue {
	if path == "" {
		path = filepath.Join(os.TempDir(), "edge-agent-queue.db")
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 1 * time.Second,
	})
	if err != nil {
		logger.Warn("failed to open offline queue (non-fatal)",
			"path", path,
			"error", err,
		)
		return &OfflineQueue{
			logger: logger.With("component", "offline_queue"),
		}
	}

	// Create bucket if not exists
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(queueBucket))
		return err
	}); err != nil {
		logger.Warn("failed to create queue bucket",
			"error", err,
		)
	}

	logger.Info("offline queue initialized", "path", path)
	return &OfflineQueue{
		db:     db,
		logger: logger.With("component", "offline_queue"),
	}
}

// Enqueue adds a message to the offline queue.
func (q *OfflineQueue) Enqueue(topic string, payload []byte) error {
	if q.db == nil {
		q.logger.Warn("queue not available, dropping message", "topic", topic)
		return nil
	}

	msg := QueuedMessage{
		Topic:      topic,
		Payload:    payload,
		QueuedAt:   time.Now(),
		RetryCount: 0,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	return q.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(queueBucket))
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		// Check queue size
		stats := b.Stats()
		if stats.KeyN >= maxMessages {
			// Remove oldest message
			c := b.Cursor()
			if k, _ := c.First(); k != nil {
				if err := b.Delete(k); err != nil {
					return fmt.Errorf("delete oldest: %w", err)
				}
			}
		}

		// Use timestamp as key for ordering
		key := []byte(fmt.Sprintf("%020d", time.Now().UnixNano()))
		return b.Put(key, data)
	})
}

// DequeueAll retrieves and removes all messages from the queue.
func (q *OfflineQueue) DequeueAll() ([]QueuedMessage, error) {
	if q.db == nil {
		return nil, nil
	}

	var messages []QueuedMessage

	err := q.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(queueBucket))
		if b == nil {
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var msg QueuedMessage
			if err := json.Unmarshal(v, &msg); err != nil {
				q.logger.Warn("failed to unmarshal queued message",
					"error", err,
				)
				continue
			}

			// Skip messages that exceeded max retries
			if msg.RetryCount >= maxRetries {
				q.logger.Warn("message exceeded max retries, dropping",
					"topic", msg.Topic,
				)
				b.Delete(k)
				continue
			}

			messages = append(messages, msg)
			b.Delete(k)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("dequeue all: %w", err)
	}

	if len(messages) > 0 {
		q.logger.Info("dequeued messages for replay",
			"count", len(messages),
		)
	}

	return messages, nil
}

// Peek returns the current queue size without dequeueing.
func (q *OfflineQueue) Peek() (int, error) {
	if q.db == nil {
		return 0, nil
	}

	var count int
	err := q.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(queueBucket))
		if b == nil {
			return nil
		}
		count = b.Stats().KeyN
		return nil
	})

	return count, err
}

// Close closes the BoltDB database.
func (q *OfflineQueue) Close() error {
	if q.db == nil {
		return nil
	}
	q.logger.Info("closing offline queue")
	return q.db.Close()
}
