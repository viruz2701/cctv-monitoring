package events

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// ── Subscriber ────────────────────────────────────────────────────

// Subscriber подписывается на NATS топики и диспетчеризует события через worker pool.
type Subscriber struct {
	conn    *nats.Conn
	js      nats.JetStreamContext
	logger  *slog.Logger
	workers int
	subs    []*nats.Subscription
	mu      sync.Mutex
	wg      sync.WaitGroup
	msgCh   chan *nats.Msg
	stopCh  chan struct{}

	// Callbacks
	onAlarm      func(AlarmEvent)
	onCMMS       func(CMMSEvent)
	onPrediction func(PredictionEvent)
	onTelemetry  func(TelemetryEvent)
}

// SubscriberConfig — параметры для Subscriber.
type SubscriberConfig struct {
	URL     string
	Creds   string
	UseTLS  bool
	Workers int
	Logger  *slog.Logger
}

// NewSubscriber создаёт Subscriber с worker pool.
func NewSubscriber(cfg SubscriberConfig) (*Subscriber, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}

	opts := []nats.Option{
		nats.Name("gb-telemetry-subscriber"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			cfg.Logger.Warn("nats subscriber disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			cfg.Logger.Info("nats subscriber reconnected", "url", nc.ConnectedUrl())
		}),
	}

	if cfg.Creds != "" {
		opts = append(opts, nats.UserCredentials(cfg.Creds))
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("nats subscriber connect: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("nats subscriber jetstream: %w", err)
	}

	s := &Subscriber{
		conn:    nc,
		js:      js,
		logger:  cfg.Logger,
		workers: cfg.Workers,
		msgCh:   make(chan *nats.Msg, 1024),
		stopCh:  make(chan struct{}),
	}

	s.startWorkers()
	return s, nil
}

// OnAlarm регистрирует обработчик тревог.
func (s *Subscriber) OnAlarm(fn func(AlarmEvent)) {
	s.onAlarm = fn
}

// OnCMMS регистрирует обработчик CMMS событий.
func (s *Subscriber) OnCMMS(fn func(CMMSEvent)) {
	s.onCMMS = fn
}

// OnPrediction регистрирует обработчик предиктивных прогнозов.
func (s *Subscriber) OnPrediction(fn func(PredictionEvent)) {
	s.onPrediction = fn
}

// OnTelemetry регистрирует обработчик телеметрии.
func (s *Subscriber) OnTelemetry(fn func(TelemetryEvent)) {
	s.onTelemetry = fn
}

// SubscribeAll подписывается на все шаблоны топиков.
func (s *Subscriber) SubscribeAll() error {
	patterns := []string{
		"alarms.>",
		"cmms.workorder.>",
		"predictions.>",
		"telemetry.>",
	}
	for _, p := range patterns {
		sub, err := s.conn.ChanSubscribe(p, s.msgCh)
		if err != nil {
			return fmt.Errorf("subscribe %s: %w", p, err)
		}
		s.mu.Lock()
		s.subs = append(s.subs, sub)
		s.mu.Unlock()
		s.logger.Info("nats subscribed", "pattern", p)
	}
	return nil
}

// SubscribeCustom подписывается на конкретный subject.
func (s *Subscriber) SubscribeCustom(subject string) error {
	sub, err := s.conn.ChanSubscribe(subject, s.msgCh)
	if err != nil {
		return fmt.Errorf("subscribe %s: %w", subject, err)
	}
	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()
	return nil
}

// Close отписывается и закрывает соединение.
func (s *Subscriber) Close() {
	close(s.stopCh)
	s.wg.Wait()

	s.mu.Lock()
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
	s.mu.Unlock()

	s.conn.Close()
}

// JetStream возвращает JetStream context.
func (s *Subscriber) JetStream() nats.JetStreamContext {
	return s.js
}

// ── Worker pool ───────────────────────────────────────────────────

func (s *Subscriber) startWorkers() {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
}

func (s *Subscriber) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case <-s.stopCh:
			return
		case msg := <-s.msgCh:
			s.dispatch(msg)
		}
	}
}

func (s *Subscriber) dispatch(msg *nats.Msg) {
	subject := msg.Subject

	switch {
	case matchSubject(subject, "alarms."):
		if s.onAlarm == nil {
			return
		}
		var event AlarmEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("nats unmarshal alarm", "subject", subject, "error", err)
			return
		}
		s.onAlarm(event)

	case matchSubject(subject, "cmms.workorder."):
		if s.onCMMS == nil {
			return
		}
		var event CMMSEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("nats unmarshal cmms", "subject", subject, "error", err)
			return
		}
		s.onCMMS(event)

	case matchSubject(subject, "predictions."):
		if s.onPrediction == nil {
			return
		}
		var event PredictionEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("nats unmarshal prediction", "subject", subject, "error", err)
			return
		}
		s.onPrediction(event)

	case matchSubject(subject, "telemetry."):
		if s.onTelemetry == nil {
			return
		}
		var event TelemetryEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			s.logger.Error("nats unmarshal telemetry", "subject", subject, "error", err)
			return
		}
		s.onTelemetry(event)
	}
}

func matchSubject(subject, prefix string) bool {
	return len(subject) >= len(prefix) && subject[:len(prefix)] == prefix
}
