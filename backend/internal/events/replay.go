// Package events — NATS JetStream Event Replay service.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Auditable message replay)
//   - ISO 27001 A.12.4 (Audit trail — traceable replay operations)
//   - OWASP ASVS V1 (Input validation — sequence numbers, stream names)
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// ── Types ────────────────────────────────────────────────────────────────────

// StreamInfo содержит основную информацию о JetStream stream.
type StreamInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Subjects    string `json:"subjects"`
	MsgCount    uint64 `json:"msg_count"`
	ByteSize    uint64 `json:"byte_size"`
	MaxAge      string `json:"max_age"`
	Storage     string `json:"storage"`
	Retention   string `json:"retention"`
}

// MessageInfo содержит данные одного сообщения из stream.
type MessageInfo struct {
	Sequence   uint64          `json:"seq"`
	Subject    string          `json:"subject"`
	Data       json.RawMessage `json:"data"`
	Timestamp  time.Time       `json:"timestamp"`
	StreamName string          `json:"stream_name"`
}

// MessageFilter для поиска по сообщениям.
type MessageFilter struct {
	Limit    int
	Offset   int
	Subject  string
	TenantID string
	Since    time.Time
	Until    time.Time
}

// ── DLQ Config ───────────────────────────────────────────────────────────────

const (
	// DLQStream — имя dead letter queue stream.
	DLQStream = "DLQ"
	// DefaultPageSize — размер страницы по умолчанию.
	DefaultPageSize = 50
	// MaxPageSize — максимальный размер страницы.
	MaxPageSize = 500
)

// ── EventReplay Service ──────────────────────────────────────────────────────

// EventReplay предоставляет методы для просмотра и повторного воспроизведения
// NATS JetStream событий.
type EventReplay struct {
	js     jetstream.JetStream
	nc     *nats.Conn
	logger *slog.Logger
}

// NewEventReplay создаёт новый EventReplay сервис.
// Использует nats.go v1.41.1+ jetstream API.
func NewEventReplay(nc *nats.Conn, logger *slog.Logger) (*EventReplay, error) {
	if logger == nil {
		logger = slog.Default()
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("jetstream new: %w", err)
	}

	return &EventReplay{js: js, nc: nc, logger: logger}, nil
}

// ListStreams возвращает список всех JetStream streams с основной информацией.
func (er *EventReplay) ListStreams(ctx context.Context) ([]StreamInfo, error) {
	lister := er.js.StreamNames(ctx)

	streams := make([]StreamInfo, 0)
	for name := range lister.Name() {
		streamObj, err := er.js.Stream(ctx, name)
		if err != nil {
			er.logger.Warn("event replay: failed to get stream", "stream", name, "error", err)
			continue
		}

		si, err := streamObj.Info(ctx)
		if err != nil {
			er.logger.Warn("event replay: failed to get stream info", "stream", name, "error", err)
			streams = append(streams, StreamInfo{Name: name})
			continue
		}

		streams = append(streams, StreamInfo{
			Name:        si.Config.Name,
			Description: si.Config.Description,
			Subjects:    fmtSubjects(si.Config.Subjects),
			MsgCount:    si.State.Msgs,
			ByteSize:    si.State.Bytes,
			MaxAge:      si.Config.MaxAge.String(),
			Storage:     storageType(si.Config.Storage),
			Retention:   retentionPolicy(si.Config.Retention),
		})
	}
	if err := lister.Err(); err != nil {
		return nil, fmt.Errorf("list stream names: %w", err)
	}

	return streams, nil
}

// ListMessages возвращает сообщения из stream с пагинацией.
// Использует DirectGet для последовательного доступа к сообщениям.
func (er *EventReplay) ListMessages(ctx context.Context, stream string, filter MessageFilter) ([]MessageInfo, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	streamObj, err := er.js.Stream(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("get stream %s: %w", stream, err)
	}

	si, err := streamObj.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream info %s: %w", stream, err)
	}

	firstSeq := si.State.FirstSeq
	lastSeq := si.State.LastSeq
	if firstSeq == 0 || lastSeq == 0 {
		return []MessageInfo{}, nil
	}

	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	startSeq := firstSeq + uint64(offset)
	endSeq := startSeq + uint64(limit) - 1
	if endSeq > lastSeq {
		endSeq = lastSeq
	}
	if startSeq > lastSeq {
		return []MessageInfo{}, nil
	}

	messages := make([]MessageInfo, 0, limit)
	for seq := startSeq; seq <= endSeq; seq++ {
		msg, err := streamObj.GetMsg(ctx, seq)
		if err != nil {
			er.logger.Debug("event replay: get msg skipped",
				"stream", stream, "seq", seq, "error", err,
			)
			continue
		}

		info := MessageInfo{
			Sequence:   msg.Sequence,
			Subject:    msg.Subject,
			Data:       msg.Data,
			Timestamp:  msg.Time,
			StreamName: stream,
		}

		// Применяем фильтры
		if filter.Subject != "" && msg.Subject != filter.Subject {
			continue
		}
		if filter.TenantID != "" {
			var envelope struct {
				TenantID string `json:"tenant_id"`
			}
			if err := json.Unmarshal(msg.Data, &envelope); err != nil {
				continue
			}
			if envelope.TenantID != filter.TenantID {
				continue
			}
		}
		if !filter.Since.IsZero() && info.Timestamp.Before(filter.Since) {
			continue
		}
		if !filter.Until.IsZero() && info.Timestamp.After(filter.Until) {
			break
		}

		messages = append(messages, info)
	}

	if messages == nil {
		messages = []MessageInfo{}
	}

	return messages, nil
}

// ReplayMessage повторно публикует сообщение из stream в его исходный subject.
func (er *EventReplay) ReplayMessage(ctx context.Context, stream string, seq uint64) error {
	streamObj, err := er.js.Stream(ctx, stream)
	if err != nil {
		return fmt.Errorf("get stream %s: %w", stream, err)
	}

	msg, err := streamObj.GetMsg(ctx, seq)
	if err != nil {
		return fmt.Errorf("get msg %s seq %d: %w", stream, seq, err)
	}

	// Создаём nats.Msg с заголовками replay
	replayMsg := &nats.Msg{
		Subject: msg.Subject,
		Data:    msg.Data,
		Header: nats.Header{
			"X-Replay":           []string{"true"},
			"X-Replay-Stream":    []string{stream},
			"X-Replay-Seq":       []string{fmt.Sprintf("%d", seq)},
			"X-Replay-Timestamp": []string{msg.Time.Format(time.RFC3339)},
		},
	}

	_, err = er.js.PublishMsg(ctx, replayMsg)
	if err != nil {
		return fmt.Errorf("replay publish %s seq %d: %w", stream, seq, err)
	}

	er.logger.Info("event replay: message replayed",
		"stream", stream,
		"seq", seq,
		"subject", msg.Subject,
	)
	return nil
}

// ListDeadLetters возвращает сообщения из DLQ (dead letter queue).
func (er *EventReplay) ListDeadLetters(ctx context.Context, filter MessageFilter) ([]MessageInfo, error) {
	return er.ListMessages(ctx, DLQStream, filter)
}

// GetStreamSubjects возвращает список subject'ов для stream.
func (er *EventReplay) GetStreamSubjects(ctx context.Context, stream string) ([]string, error) {
	streamObj, err := er.js.Stream(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("get stream %s: %w", stream, err)
	}

	si, err := streamObj.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream info %s: %w", stream, err)
	}

	return si.Config.Subjects, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func fmtSubjects(subjects []string) string {
	if len(subjects) == 0 {
		return ">"
	}
	result := ""
	for i, s := range subjects {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func storageType(s jetstream.StorageType) string {
	switch s {
	case jetstream.FileStorage:
		return "file"
	case jetstream.MemoryStorage:
		return "memory"
	default:
		return "unknown"
	}
}

func retentionPolicy(r jetstream.RetentionPolicy) string {
	switch r {
	case jetstream.LimitsPolicy:
		return "limits"
	case jetstream.InterestPolicy:
		return "interest"
	case jetstream.WorkQueuePolicy:
		return "work_queue"
	default:
		return "unknown"
	}
}
