// Package events предоставляет NATS Event Bus для асинхронной публикации/подписки.
// Топики структурированы по доменам: alarms, cmms, predictions, telemetry.
package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// ── Topic constants ───────────────────────────────────────────────

const (
	TopicAlarms      = "alarms.%s"         // alarms.{device_id}
	TopicCMMSWO      = "cmms.workorder.%s" // cmms.workorder.{event} (created|updated|completed|cancelled)
	TopicPredictions = "predictions.%s"    // predictions.{device_id}
	TopicTelemetry   = "telemetry.%s"      // telemetry.{device_id}
)

// ── Event types ───────────────────────────────────────────────────

// AlarmEvent — событие тревоги от устройства.
type AlarmEvent struct {
	DeviceID   string    `json:"device_id"`
	DeviceName string    `json:"device_name,omitempty"`
	Type       string    `json:"type"`     // motion, tamper, video_loss, etc.
	Severity   string    `json:"severity"` // critical, high, medium, low
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	ImageURL   string    `json:"image_url,omitempty"`
}

// CMMSEvent — событие жизненного цикла заявки.
type CMMSEvent struct {
	Event       string    `json:"event"` // created, updated, assigned, started, completed, cancelled
	WorkOrderID string    `json:"work_order_id"`
	DeviceID    string    `json:"device_id,omitempty"`
	Status      string    `json:"status,omitempty"`
	AssigneeID  string    `json:"assignee_id,omitempty"`
	Priority    string    `json:"priority,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// PredictionEvent — событие предиктивного прогноза.
type PredictionEvent struct {
	DeviceID       string    `json:"device_id"`
	DeviceName     string    `json:"device_name,omitempty"`
	FailureMode    string    `json:"failure_mode"`
	Probability    float64   `json:"probability"`
	EstimatedDays  int       `json:"estimated_days"`
	Recommendation string    `json:"recommendation"`
	Timestamp      time.Time `json:"timestamp"`
}

// TelemetryEvent — событие телеметрии.
type TelemetryEvent struct {
	DeviceID   string            `json:"device_id"`
	DeviceName string            `json:"device_name,omitempty"`
	Metric     string            `json:"metric"`
	Value      float64           `json:"value"`
	Tags       map[string]string `json:"tags,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
}

// ── Publisher ─────────────────────────────────────────────────────

// Publisher публикует события в NATS.
type Publisher struct {
	conn           *nats.Conn
	js             nats.JetStreamContext
	logger         *slog.Logger
	schemaRegistry *SchemaRegistry // опциональная валидация
}

// PublisherConfig — параметры подключения.
type PublisherConfig struct {
	URL            string
	Creds          string
	UseTLS         bool
	Logger         *slog.Logger
	SchemaRegistry *SchemaRegistry // опциональный SchemaRegistry для валидации
}

// NewPublisher создаёт и подключает Publisher к NATS.
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	opts := []nats.Option{
		nats.Name("gb-telemetry-publisher"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			cfg.Logger.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			cfg.Logger.Info("nats reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			cfg.Logger.Error("nats error", "subject", sub.Subject, "error", err)
		}),
	}

	if cfg.Creds != "" {
		opts = append(opts, nats.UserCredentials(cfg.Creds))
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats jetstream: %w", err)
	}

	logger := cfg.Logger.With("component", "publisher")
	return &Publisher{
		conn:           nc,
		js:             js,
		logger:         logger,
		schemaRegistry: cfg.SchemaRegistry,
	}, nil
}

// Close закрывает соединение.
func (p *Publisher) Close() {
	p.conn.Close()
}

// PublishAlarm публикует событие тревоги.
func (p *Publisher) PublishAlarm(event AlarmEvent) error {
	return p.publishJSON(fmt.Sprintf(TopicAlarms, event.DeviceID), event)
}

// PublishCMMS публикует событие CMMS Work Order.
func (p *Publisher) PublishCMMS(event CMMSEvent) error {
	return p.publishJSON(fmt.Sprintf(TopicCMMSWO, event.Event), event)
}

// PublishPrediction публикует предиктивный прогноз.
func (p *Publisher) PublishPrediction(event PredictionEvent) error {
	return p.publishJSON(fmt.Sprintf(TopicPredictions, event.DeviceID), event)
}

// PublishTelemetry публикует телеметрию.
func (p *Publisher) PublishTelemetry(event TelemetryEvent) error {
	return p.publishJSON(fmt.Sprintf(TopicTelemetry, event.DeviceID), event)
}

// ── Validated publish (с schema validation) ───────────────────────

// PublishRecord публикует EventRecord с опциональной валидацией через SchemaRegistry.
//
// Если SchemaRegistry настроен — выполняет Validate() перед публикацией.
// При failed validation логирует полный payload на уровне WARN.
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation — whitelist validation)
//   - OWASP ASVS V5.3 (Input validation — structured data validation)
func (p *Publisher) PublishRecord(record *EventRecord) error {
	subject := subjectForRecord(record)
	return p.publishValidated(subject, record)
}

// publishValidated публикует EventRecord с валидацией (если schemaRegistry настроен).
func (p *Publisher) publishValidated(subject string, record *EventRecord) error {
	if p.schemaRegistry != nil {
		if err := p.schemaRegistry.Validate(record); err != nil {
			// Логируем failed validation с полным payload (WARN уровень)
			payloadStr := string(record.Data)
			if len(payloadStr) > 2000 {
				payloadStr = payloadStr[:2000] + "... [truncated]"
			}
			p.logger.Warn("event validation failed, publishing skipped",
				"subject", subject,
				"source", record.Source,
				"event_type", record.EventType,
				"error", err,
				"trace_id", record.TraceID,
				"payload", payloadStr,
			)
			return fmt.Errorf("publish validation: %w", err)
		}
	}
	return p.publishJSON(subject, record)
}

// JetStream возвращает JetStream context для прямых операций.
func (p *Publisher) JetStream() nats.JetStreamContext {
	return p.js
}

// Conn возвращает сырое NATS соединение.
func (p *Publisher) Conn() *nats.Conn {
	return p.conn
}

func (p *Publisher) publishJSON(subject string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := p.conn.Publish(subject, data); err != nil {
		p.logger.Error("nats publish failed", "subject", subject, "error", err)
		return fmt.Errorf("publish %s: %w", subject, err)
	}
	return nil
}
