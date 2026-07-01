package events

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// ── JetStream stream names ────────────────────────────────────────

const (
	StreamAlarms      = "ALARMS"
	StreamCMMS        = "CMMS_WORKORDERS"
	StreamPredictions = "PREDICTIONS"
	StreamTelemetry   = "TELEMETRY"
	StreamAudit       = "AUDIT"
)

// ── JetStream Manager ─────────────────────────────────────────────

// JetStreamManager управляет persistent streams и consumer'ами.
type JetStreamManager struct {
	js     nats.JetStreamContext
	logger *slog.Logger
}

// NewJetStreamManager создаёт менеджер JetStream.
func NewJetStreamManager(js nats.JetStreamContext, logger *slog.Logger) *JetStreamManager {
	if logger == nil {
		logger = slog.Default()
	}
	return &JetStreamManager{js: js, logger: logger}
}

// InitStreams создаёт все persistent streams, если они ещё не существуют.
func (m *JetStreamManager) InitStreams() error {
	streams := []*nats.StreamConfig{
		{
			Name:        StreamAlarms,
			Description: "Real-time alarm events",
			Subjects:    []string{"alarms.>"},
			Retention:   nats.InterestPolicy,
			MaxAge:      7 * 24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		},
		{
			Name:        StreamCMMS,
			Description: "CMMS Work Order lifecycle events",
			Subjects:    []string{"cmms.workorder.>"},
			Retention:   nats.InterestPolicy,
			MaxAge:      30 * 24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		},
		{
			Name:        StreamPredictions,
			Description: "Predictive maintenance predictions",
			Subjects:    []string{"predictions.>"},
			Retention:   nats.InterestPolicy,
			MaxAge:      30 * 24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		},
		{
			Name:        StreamTelemetry,
			Description: "Device telemetry stream",
			Subjects:    []string{"telemetry.>"},
			Retention:   nats.LimitsPolicy,
			MaxAge:      24 * time.Hour,
			MaxMsgs:     10_000_000,
			MaxBytes:    5 * 1024 * 1024 * 1024, // 5 GB
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld,
		},
		{
			Name:        StreamAudit,
			Description: "Audit trail for all events",
			Subjects:    []string{"alarms.>", "cmms.workorder.>", "predictions.>", "telemetry.>"},
			Retention:   nats.LimitsPolicy,
			MaxAge:      365 * 24 * time.Hour,
			MaxMsgs:     50_000_000,              // P1-HI-04: лимит сообщений
			MaxBytes:    10 * 1024 * 1024 * 1024, // P1-HI-04: 10 GB лимит
			Storage:     nats.FileStorage,
			Replicas:    1,
			Discard:     nats.DiscardOld, // P1-HI-04: discard old вместо new
			AllowDirect: true,
		},
	}

	for _, cfg := range streams {
		_, err := m.js.AddStream(cfg)
		if err != nil {
			m.logger.Warn("jetstream add stream", "name", cfg.Name, "error", err)
		} else {
			m.logger.Info("jetstream stream created", "name", cfg.Name, "subjects", cfg.Subjects)
		}
	}
	return nil
}

// CreateDurableConsumer создаёт durable consumer для replay.
func (m *JetStreamManager) CreateDurableConsumer(stream, durable, filterSubject string) (*nats.Subscription, error) {
	sub, err := m.js.PullSubscribe(
		filterSubject,
		durable,
		nats.BindStream(stream),
		nats.AckExplicit(),
		nats.MaxDeliver(3),
		nats.AckWait(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("create durable consumer %s on %s: %w", durable, stream, err)
	}
	return sub, nil
}

// ReplayStream воспроизводит сообщения из стрима с заданного времени.
func (m *JetStreamManager) ReplayStream(stream string, since time.Duration, handler func(msg *nats.Msg) error) error {
	sub, err := m.js.PullSubscribe(
		">",
		"replay-"+stream,
		nats.BindStream(stream),
	)
	if err != nil {
		return fmt.Errorf("replay subscribe %s: %w", stream, err)
	}
	defer sub.Unsubscribe()

	startTime := time.Now().Add(-since)
	for {
		msgs, err := sub.Fetch(100, nats.MaxWait(5*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				break
			}
			return fmt.Errorf("replay fetch %s: %w", stream, err)
		}
		for _, msg := range msgs {
			meta, _ := msg.Metadata()
			if meta != nil && meta.Timestamp.Before(startTime) {
				_ = msg.Ack()
				continue
			}
			if err := handler(msg); err != nil {
				m.logger.Error("replay handler error", "stream", stream, "error", err)
				_ = msg.Nak()
			} else {
				_ = msg.Ack()
			}
		}
	}
	return nil
}

// StreamInfo возвращает информацию о стриме.
func (m *JetStreamManager) StreamInfo(stream string) (*nats.StreamInfo, error) {
	return m.js.StreamInfo(stream)
}

// PurgeStream очищает стрим.
func (m *JetStreamManager) PurgeStream(stream string) error {
	return m.js.PurgeStream(stream)
}

// DeleteStream удаляет стрим.
func (m *JetStreamManager) DeleteStream(stream string) error {
	return m.js.DeleteStream(stream)
}
