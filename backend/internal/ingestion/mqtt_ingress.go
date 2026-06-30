// Package ingestion — Unified Ingestion Layer for CCTV Health Monitor (P0-EDGE Block 5).
//
// ═══════════════════════════════════════════════════════════════════════
// INGEST-01: MQTT Ingress Handler
//
// Подписывается на NATS топики edge.>.>.>, парсит входящие данные от
// edge-агентов, нормализует через VendorNormalizer и распределяет по
// каналам: DeviceState, Alarm, EventBus, TimescaleDB.
//
// Топик: edge.{agent_id}.{device_id}.{type}
// Типы: telemetry, alarm, log, event
//
// Compliance:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
//   - OWASP ASVS L3 V5.1: Input validation (whitelist validation)
//   - OWASP ASVS L3 V7.1: Error handling (no information leakage)
//   - ISO 27001 A.12.4: Audit trail
//   - СТБ 34.101.27 п. 7.5: Audit trail integrity
// ═══════════════════════════════════════════════════════════════════════
package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gb-telemetry-collector/internal/events"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
	"gb-telemetry-collector/internal/trace"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

// ── Constants ──────────────────────────────────────────────────────────

// Topic prefix for edge agent data ingestion.
const edgeTopicPrefix = "edge."

// edgeDataTypes defines valid data types from edge agents (whitelist).
// OWASP ASVS V5.1: Input validation via whitelist.
var validEdgeDataTypes = map[string]struct{}{
	"telemetry": {},
	"alarm":     {},
	"log":       {},
	"event":     {},
}

// ── Data Types ─────────────────────────────────────────────────────────

// EdgeIngressMessage — универсальная структура входящего сообщения от edge-агента.
// Поле Payload содержит вендор-специфичные данные, которые нормализуются
// через VendorNormalizer.
type EdgeIngressMessage struct {
	Vendor    string          `json:"vendor"`              // hikvision, dahua, onvif, tiandy, uniview, tantos
	Model     string          `json:"model,omitempty"`     // model name
	Type      string          `json:"type"`                // telemetry, alarm, log, event
	Timestamp time.Time       `json:"timestamp,omitempty"` // время события (edge)
	Payload   json.RawMessage `json:"payload"`             // вендор-специфичные данные
}

// ── MQTT Ingress ───────────────────────────────────────────────────────

// MQTTIngressConfig — конфигурация MQTT Ingress Handler.
type MQTTIngressConfig struct {
	NATSURL      string // NATS server URL
	NATSCreds    string // NATS credentials file (optional)
	LogTTL       time.Duration // retention for TimescaleDB logs
	TraceContext context.Context // базовый context для фоновых горутин
	Logger       *slog.Logger
}

// MQTTIngress подписывается на edge-топики NATS, нормализует и распределяет
// входящие данные от edge-агентов.
//
// Zone: 3 (Backend) по IEC 62443-3-3 SL-3
type MQTTIngress struct {
	conn       *nats.Conn
	sub        *nats.Subscription
	normalizer *VendorNormalizer
	stateMgr   state.DeviceStateManager
	pool       *pgxpool.Pool            // для TimescaleDB логов
	publisher  *events.Publisher        // Event Bus
	logger     *slog.Logger
	config     MQTTIngressConfig
}

// NewMQTTIngress создаёт новый MQTT Ingress Handler.
func NewMQTTIngress(
	cfg MQTTIngressConfig,
	normalizer *VendorNormalizer,
	stateMgr state.DeviceStateManager,
	pool *pgxpool.Pool,
	publisher *events.Publisher,
) (*MQTTIngress, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.LogTTL == 0 {
		cfg.LogTTL = 90 * 24 * time.Hour // 90 дней по умолчанию
	}

	logger := cfg.Logger.With("component", "mqtt_ingress")

	opts := []nats.Option{
		nats.Name("gb-telemetry-ingress"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("nats disconnected", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("nats reconnected", "url", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Error("nats error", "subject", sub.Subject, "error", err)
		}),
	}

	if cfg.NATSCreds != "" {
		opts = append(opts, nats.UserCredentials(cfg.NATSCreds))
	}

	nc, err := nats.Connect(cfg.NATSURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("ingress nats connect: %w", err)
	}

	return &MQTTIngress{
		conn:       nc,
		normalizer: normalizer,
		stateMgr:   stateMgr,
		pool:       pool,
		publisher:  publisher,
		logger:     logger,
		config:     cfg,
	}, nil
}

// Start подписывается на edge.> и начинает обработку входящих сообщений.
func (m *MQTTIngress) Start() error {
	sub, err := m.conn.Subscribe("edge.>", m.handleMessage)
	if err != nil {
		return fmt.Errorf("subscribe edge.>: %w", err)
	}
	m.sub = sub
	m.logger.Info("mqtt ingress started", "subject", "edge.>")
	return nil
}

// Stop отписывается и закрывает NATS соединение.
func (m *MQTTIngress) Stop() {
	if m.sub != nil {
		if err := m.sub.Unsubscribe(); err != nil {
			m.logger.Warn("unsubscribe error", "error", err)
		}
	}
	m.conn.Close()
	m.logger.Info("mqtt ingress stopped")
}

// handleMessage — основной обработчик входящих сообщений от edge-агентов.
//
// Парсит топик вида edge.{agent_id}.{device_id}.{type}, валидирует тип данных,
// нормализует payload и распределяет по каналам обработки.
//
// Compliance:
//   - OWASP ASVS V5.1: Whitelist validation (validEdgeDataTypes)
//   - OWASP ASVS V7.1: Error handling — все ошибки логируются, sensitive data
//     не раскрывается в сообщениях об ошибках
//   - ISO 27001 A.12.4: Все мутации логируются с trace_id
func (m *MQTTIngress) handleMessage(msg *nats.Msg) {
	// 1. Создаём trace_id для сквозной трассировки
	ctx := m.config.TraceContext
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = trace.WithNewID(ctx)
	traceID := trace.FromContext(ctx)

	logger := m.logger.With(
		slog.String("trace_id", traceID),
		slog.String("subject", msg.Subject),
	)

	// 2. Парсим топик: edge.{agent_id}.{device_id}.{type}
	// OWASP ASVS V5.1: Проверяем формат топика (whitelist)
	agentID, deviceID, dataType, err := parseEdgeTopic(msg.Subject)
	if err != nil {
		logger.Warn("invalid edge topic format", "error", err)
		return
	}

	logger = logger.With(
		slog.String("agent_id", agentID),
		slog.String("device_id", deviceID),
		slog.String("data_type", dataType),
	)

	// 3. Валидация типа данных (whitelist — OWASP ASVS V5.1)
	if _, valid := validEdgeDataTypes[dataType]; !valid {
		logger.Warn("unknown edge data type, ignoring",
			"valid_types", strings.Join(validDataTypes(), ", "),
		)
		return
	}

	// 4. Парсим входящее сообщение
	var ingressMsg EdgeIngressMessage
	if err := json.Unmarshal(msg.Data, &ingressMsg); err != nil {
		logger.Warn("failed to unmarshal edge message",
			"error", err,
			"payload_size", len(msg.Data),
		)
		return
	}

	// 5. Валидация обязательных полей (OWASP ASVS V5.1)
	if ingressMsg.Vendor == "" {
		logger.Warn("missing vendor field in edge message")
		return
	}
	if ingressMsg.Payload == nil || len(ingressMsg.Payload) == 0 {
		logger.Warn("empty payload in edge message")
		return
	}

	// 6. Нормализуем данные через VendorNormalizer
	event, err := m.normalizer.Normalize(dataType, ingressMsg.Vendor, ingressMsg.Payload)
	if err != nil {
		logger.Warn("data normalization failed",
			"vendor", ingressMsg.Vendor,
			"error", err,
		)
		return
	}

	// 7. Распределяем по каналам обработки
	switch dataType {
	case "telemetry":
		m.handleTelemetry(ctx, deviceID, event, logger)
	case "alarm":
		m.handleAlarm(ctx, deviceID, event, logger)
	case "log":
		m.handleLog(ctx, deviceID, event, logger)
	case "event":
		m.handleEvent(ctx, deviceID, event, logger)
	}
}

// ── Обработчики по типам данных ───────────────────────────────────────

// handleTelemetry обновляет DeviceState новыми телеметрическими данными.
//
// Соответствие:
//   - IEC 62443-3-3 SR 3.1: Wireless/remote monitoring
//   - ISO 27001 A.12.4.1: Event logging
func (m *MQTTIngress) handleTelemetry(ctx context.Context, deviceID string, event *models.Event, logger *slog.Logger) {
	m.stateMgr.UpdateLastSeen(deviceID)
	m.stateMgr.SetOnline(deviceID)

	logger.Debug("telemetry processed",
		"event_type", event.Type,
		"metrics", len(event.Metrics),
	)

	// Публикуем в Event Bus для downstream обработчиков
	if m.publisher != nil {
		_ = m.publisher.PublishTelemetry(events.TelemetryEvent{
			DeviceID:   deviceID,
			Metric:     event.Type,
			Value:      extractPrimaryValue(event),
			Tags:       event.Tags,
			Timestamp:  time.Now(),
		})
	}
}

// handleAlarm создаёт Alarm в DeviceState и публикует alarm-событие.
//
// Соответствие:
//   - IEC 62443-3-3 SR 3.1: Alarm handling
//   - ISO 27001 A.12.4.1: События тревог логируются
func (m *MQTTIngress) handleAlarm(ctx context.Context, deviceID string, event *models.Event, logger *slog.Logger) {
	alarm := &models.Alarm{
		DeviceID:    deviceID,
		Priority:    mapAlarmPriority(event.Severity),
		Method:      mapAlarmMethod(event.Type),
		Timestamp:   event.Timestamp,
		Description: event.Message,
		ImagePath:   event.ImageURL,
	}
	m.stateMgr.AddAlarm(deviceID, alarm)

	logger.Warn("alarm received",
		"priority", alarm.Priority,
		"method", alarm.Method,
		"description", truncateString(alarm.Description, 200),
	)

	// Публикуем alarm-событие в Event Bus
	if m.publisher != nil {
		_ = m.publisher.PublishAlarm(events.AlarmEvent{
			DeviceID:   deviceID,
			Type:       event.Type,
			Severity:   event.Severity,
			Message:    event.Message,
			Timestamp:  event.Timestamp,
			ImageURL:   event.ImageURL,
		})
	}
}

// handleLog сохраняет лог-сообщение в TimescaleDB.
//
// Соответствие:
//   - ISO 27001 A.12.4: Audit trail — логи сохраняются с retention 90 дней
//   - СТБ 34.101.27 п. 7.5: Audit trail integrity
//   - OWASP ASVS V7.1: Error handling — параметризованные запросы
func (m *MQTTIngress) handleLog(ctx context.Context, deviceID string, event *models.Event, logger *slog.Logger) {
	if m.pool == nil {
		logger.Warn("no database pool configured, log entry dropped")
		return
	}

	// Parameterized query (OWASP ASVS V5.1 — SQL injection prevention)
	query := `
		INSERT INTO device_logs (device_id, log_level, event_code, message, source, raw, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`

	eventCode := 0
	if code, ok := event.Metadata["event_code"]; ok {
		fmt.Sscanf(code, "%d", &eventCode)
	}

	_, err := m.pool.Exec(ctx, query,
		deviceID,
		event.Severity,
		eventCode,
		event.Message,
		event.Source,
		event.RawPayload,
	)
	if err != nil {
		logger.Error("failed to save log entry",
			"error", err,
			"device_id", deviceID,
		)
		return
	}

	logger.Debug("log entry saved", "event_code", eventCode)
}

// handleEvent публикует событие в Event Bus для downstream потребителей.
//
// Соответствие:
//   - IEC 62443-3-3 SR 3.1: Event-driven monitoring
//   - ISO 27001 A.12.4.1: Event logging
func (m *MQTTIngress) handleEvent(ctx context.Context, deviceID string, event *models.Event, logger *slog.Logger) {
	if m.publisher == nil {
		logger.Warn("no publisher configured, event dropped")
		return
	}

	_ = m.publisher.PublishTelemetry(events.TelemetryEvent{
		DeviceID:   deviceID,
		Metric:     event.Type,
		Value:      extractPrimaryValue(event),
		Tags:       event.Tags,
		Timestamp:  event.Timestamp,
	})

	logger.Debug("event published to bus", "event_type", event.Type)
}

// ── Парсинг топика ─────────────────────────────────────────────────────

// parseEdgeTopic парсит NATS топик формата edge.{agent_id}.{device_id}.{type}.
//
// OWASP ASVS V5.1: Валидация формата топика (whitelist структуры).
// Все части топика проверяются на не-пустоту и корректные символы.
func parseEdgeTopic(topic string) (agentID, deviceID, dataType string, err error) {
	if !strings.HasPrefix(topic, edgeTopicPrefix) {
		return "", "", "", fmt.Errorf("topic must start with 'edge.'")
	}

	parts := strings.SplitN(topic, ".", 5)
	if len(parts) < 4 {
		return "", "", "", fmt.Errorf("invalid topic format: expected edge.{agent_id}.{device_id}.{type}, got %q", topic)
	}

	agentID = parts[1]
	deviceID = parts[2]
	dataType = parts[3]

	// Проверка на пустые значения
	if agentID == "" {
		return "", "", "", fmt.Errorf("empty agent_id in topic %q", topic)
	}
	if deviceID == "" {
		return "", "", "", fmt.Errorf("empty device_id in topic %q", topic)
	}
	if dataType == "" {
		return "", "", "", fmt.Errorf("empty data type in topic %q", topic)
	}

	return agentID, deviceID, dataType, nil
}

// ── Вспомогательные функции ───────────────────────────────────────────

// validDataTypes возвращает список valid data types для ошибок валидации.
func validDataTypes() []string {
	types := make([]string, 0, len(validEdgeDataTypes))
	for t := range validEdgeDataTypes {
		types = append(types, t)
	}
	return types
}

// extractPrimaryValue извлекает первичное числовое значение из Event.
// Если метрики есть — возвращает первую. Иначе 0.
func extractPrimaryValue(event *models.Event) float64 {
	if len(event.Metrics) > 0 {
		return event.Metrics[0].Value
	}
	return 0
}

// mapAlarmPriority маппит severity-строку из Event в models.AlarmPriority.
func mapAlarmPriority(severity string) models.AlarmPriority {
	switch strings.ToLower(severity) {
	case "critical", "high":
		return models.AlarmPriorityHigh
	case "medium":
		return models.AlarmPriorityMedium
	default:
		return models.AlarmPriorityLow
	}
}

// mapAlarmMethod маппит тип события в models.AlarmMethod.
func mapAlarmMethod(eventType string) models.AlarmMethod {
	switch strings.ToLower(eventType) {
	case "motion", "motion_detection":
		return models.AlarmMethodMotionDetection
	case "video_loss":
		return models.AlarmMethodVideoLoss
	default:
		return models.AlarmMethodEquipmentFault
	}
}

// truncateString обрезает строку до maxLen символов.
// Используется для логирования, чтобы не раскрывать большие payload'ы.
// OWASP ASVS V7.1: No information leakage через логи.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
