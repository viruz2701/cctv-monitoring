// Package events — NATS JetStream consumer для асинхронной генерации отчётов.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Queue-based processing with retries)
//   - ISO 27001 A.12.4 (Audit trail — traceable task queue)
//   - OWASP ASVS V1 (Input validation — JSON schema enforcement)
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ── Report Task ────────────────────────────────────────────────────────────

// ReportTask описывает задачу для асинхронной генерации отчёта.
type ReportTask struct {
	ReportID  string `json:"report_id"`
	Type      string `json:"type"`   // maintenance, sla, tco
	Format    string `json:"format"` // pdf, excel
	TenantID  string `json:"tenant_id"`
	CreatedAt string `json:"created_at"`
}

// Validate проверяет обязательные поля задачи.
func (t *ReportTask) Validate() error {
	if t.ReportID == "" {
		return fmt.Errorf("report_id is required")
	}
	if t.Type == "" {
		return fmt.Errorf("type is required")
	}
	if t.Format == "" {
		return fmt.Errorf("format is required")
	}
	return nil
}

// ── Stream Config ──────────────────────────────────────────────────────────

const (
	// StreamReports — имя JetStream стрима для отчётов.
	StreamReports = "REPORTS"

	// SubjectReportGenerate — subject для публикации задач генерации отчётов.
	SubjectReportGenerate = "report.generate"

	// ConsumerReportWorker — имя durable consumer для worker'а отчётов.
	ConsumerReportWorker = "report-worker"

	// MaxReportDeliver — максимальное количество попыток доставки.
	MaxReportDeliver = 3

	// ReportStreamMaxAge — время жизни сообщения в стриме.
	ReportStreamMaxAge = 24 * time.Hour
)

// ── Report Queue ───────────────────────────────────────────────────────────

// ReportQueue управляет очередью задач генерации отчётов через NATS JetStream.
//
// Использует WorkQueuePolicy (auto-delete после Ack) для предотвращения
// повторной обработки завершённых задач.
type ReportQueue struct {
	js     jetstream.JetStream
	logger *slog.Logger
}

// NewReportQueue создаёт новый ReportQueue с JetStream стримом.
func NewReportQueue(nc *nats.Conn, logger *slog.Logger) (*ReportQueue, error) {
	if logger == nil {
		logger = slog.Default()
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      StreamReports,
		Subjects:  []string{SubjectReportGenerate},
		Retention: jetstream.WorkQueuePolicy, // auto-delete after ack
		MaxAge:    ReportStreamMaxAge,
		Storage:   jetstream.FileStorage,
	})
	if err != nil && !isStreamAlreadyExists(err) {
		return nil, fmt.Errorf("create stream %s: %w", StreamReports, err)
	}

	return &ReportQueue{js: js, logger: logger}, nil
}

// Publish ставит задачу в очередь генерации отчётов.
func (q *ReportQueue) Publish(ctx context.Context, task ReportTask) error {
	if err := task.Validate(); err != nil {
		return fmt.Errorf("validate task: %w", err)
	}

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}

	_, err = q.js.Publish(ctx, SubjectReportGenerate, data)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	q.logger.Debug("report task published",
		"report_id", task.ReportID,
		"type", task.Type,
		"format", task.Format,
	)
	return nil
}

// Consume запускает consumer для обработки задач генерации отчётов.
// Блокируется до отмены контекста. При ошибке обработки задача
// автоматически повторяется (до MaxReportDeliver раз).
func (q *ReportQueue) Consume(ctx context.Context, handler func(context.Context, ReportTask) error) error {
	cons, err := q.js.CreateOrUpdateConsumer(ctx, StreamReports, jetstream.ConsumerConfig{
		Name:          ConsumerReportWorker,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    MaxReportDeliver,
	})
	if err != nil {
		return fmt.Errorf("create consumer %s: %w", ConsumerReportWorker, err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		var task ReportTask
		if err := json.Unmarshal(msg.Data(), &task); err != nil {
			q.logger.Error("report queue: failed to unmarshal task",
				"error", err,
				"data_size", len(msg.Data()),
			)
			msg.Nak() // retry
			return
		}

		q.logger.Info("report queue: processing task",
			"report_id", task.ReportID,
			"type", task.Type,
			"format", task.Format,
		)

		if err := handler(ctx, task); err != nil {
			q.logger.Error("report queue: generation failed",
				"report_id", task.ReportID,
				"type", task.Type,
				"error", err,
			)
			msg.Nak() // retry (up to MaxReportDeliver)
			return
		}

		msg.Ack()
		q.logger.Info("report queue: task completed",
			"report_id", task.ReportID,
			"type", task.Type,
		)
	})
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	q.logger.Info("report queue consumer started",
		"stream", StreamReports,
		"consumer", ConsumerReportWorker,
		"max_deliver", MaxReportDeliver,
	)

	<-ctx.Done()
	cc.Stop()
	q.logger.Info("report queue consumer stopped")
	return nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

// isStreamAlreadyExists проверяет, является ли ошибка "stream already exists".
func isStreamAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "stream name already in use") ||
		strings.Contains(msg, "already exists")
}
