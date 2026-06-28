// Package notifications — Multi-Tier Incident Notification Router (P0-N3).
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-N3: Multi-Tier Incident Response Engine — Notification Router
//
// Маршрутизирует уведомления об инцидентах по:
//   - Регионам (EU, BY, RU, IN, SG)
//   - Уровням эскалации (L1, L2, L3)
//   - Каналам (email, SMS, webhook, telegram)
//   - SLA таймерам (4h, 6h, 2h, 24h)
//
// Compliance:
//   - EU DORA Art. 11 — ICT incident notification
//   - NIS2 Art. 23 — Incident notification
//   - India CERT-In — 6h reporting
//   - Singapore CSA — 2h reporting
//
// ═══════════════════════════════════════════════════════════════════════════
package notifications

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════
// Channel Types
// ═══════════════════════════════════════════════════════════════════════════

// NotificationChannel — канал доставки уведомления.
type NotificationChannel string

const (
	ChannelEmail    NotificationChannel = "email"
	ChannelSMS      NotificationChannel = "sms"
	ChannelWebhook  NotificationChannel = "webhook"
	ChannelTelegram NotificationChannel = "telegram"
	ChannelSlack    NotificationChannel = "slack"
	ChannelPush     NotificationChannel = "push" // Mobile push
)

// ═══════════════════════════════════════════════════════════════════════════
// Incident Notification
// ═══════════════════════════════════════════════════════════════════════════

// IncidentNotification — уведомление об инциденте.
type IncidentNotification struct {
	ID         string                `json:"id"`
	IncidentID string                `json:"incident_id"`
	Framework  string                `json:"framework"`
	Region     string                `json:"region"`
	Severity   string                `json:"severity"`
	EventType  string                `json:"event_type"` // initial_report, reminder, escalation, deadline_missed
	Channels   []NotificationChannel `json:"channels"`
	Title      string                `json:"title"`
	Body       string                `json:"body"`
	CreatedAt  time.Time             `json:"created_at"`
	SentAt     *time.Time            `json:"sent_at,omitempty"`
	Status     string                `json:"status"` // pending, sent, failed
	RetryCount int                   `json:"retry_count"`
}

// NotificationSender — интерфейс отправителя уведомлений.
type NotificationSender interface {
	Send(notification *IncidentNotification) error
}

// ═══════════════════════════════════════════════════════════════════════════
// Region Routing Config
// ═══════════════════════════════════════════════════════════════════════════

// RegionRouteConfig — конфигурация маршрутизации для региона.
type RegionRouteConfig struct {
	Region           string                `json:"region"`
	Frameworks       []string              `json:"frameworks"`
	ReportingHours   int                   `json:"reporting_hours"`
	PriorityChannels []NotificationChannel `json:"priority_channels"`
	BackupChannels   []NotificationChannel `json:"backup_channels"`
	EscalationDelay  time.Duration         `json:"escalation_delay"` // Задержка до эскалации
}

// DefaultRegionRoutes возвращает конфигурации маршрутизации по умолчанию.
func DefaultRegionRoutes() []RegionRouteConfig {
	return []RegionRouteConfig{
		{
			Region:           "EU",
			Frameworks:       []string{"DORA", "NIS2"},
			ReportingHours:   4,
			PriorityChannels: []NotificationChannel{ChannelEmail, ChannelWebhook},
			BackupChannels:   []NotificationChannel{ChannelSMS, ChannelTelegram},
			EscalationDelay:  30 * time.Minute,
		},
		{
			Region:           "IN",
			Frameworks:       []string{"CERT-In"},
			ReportingHours:   6,
			PriorityChannels: []NotificationChannel{ChannelEmail, ChannelSMS},
			BackupChannels:   []NotificationChannel{ChannelTelegram, ChannelPush},
			EscalationDelay:  15 * time.Minute,
		},
		{
			Region:           "SG",
			Frameworks:       []string{"CSA"},
			ReportingHours:   2,
			PriorityChannels: []NotificationChannel{ChannelEmail, ChannelSMS, ChannelWebhook},
			BackupChannels:   []NotificationChannel{ChannelTelegram, ChannelPush},
			EscalationDelay:  10 * time.Minute,
		},
		{
			Region:           "BY",
			Frameworks:       []string{"ОАЦ"},
			ReportingHours:   24,
			PriorityChannels: []NotificationChannel{ChannelEmail, ChannelWebhook},
			BackupChannels:   []NotificationChannel{ChannelTelegram},
			EscalationDelay:  60 * time.Minute,
		},
		{
			Region:           "RU",
			Frameworks:       []string{"ФСТЭК"},
			ReportingHours:   24,
			PriorityChannels: []NotificationChannel{ChannelEmail, ChannelWebhook},
			BackupChannels:   []NotificationChannel{ChannelTelegram},
			EscalationDelay:  60 * time.Minute,
		},
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Incident Router
// ═══════════════════════════════════════════════════════════════════════════

// IncidentRouter — multi-tier incident notification router.
type IncidentRouter struct {
	mu         sync.RWMutex
	logger     *slog.Logger
	routes     map[string]RegionRouteConfig // key = region
	senders    map[NotificationChannel]NotificationSender
	pending    []*IncidentNotification
	history    []*IncidentNotification
	maxRetries int
}

// NewIncidentRouter создаёт новый IncidentRouter.
func NewIncidentRouter(logger *slog.Logger, senders map[NotificationChannel]NotificationSender, routes ...RegionRouteConfig) *IncidentRouter {
	if logger == nil {
		logger = slog.Default().With("component", "notifications.incident_router")
	}

	routeMap := make(map[string]RegionRouteConfig)
	if len(routes) > 0 {
		for _, r := range routes {
			routeMap[r.Region] = r
		}
	} else {
		for _, r := range DefaultRegionRoutes() {
			routeMap[r.Region] = r
		}
	}

	return &IncidentRouter{
		logger:     logger,
		routes:     routeMap,
		senders:    senders,
		pending:    make([]*IncidentNotification, 0),
		history:    make([]*IncidentNotification, 0),
		maxRetries: 3,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Route & Send
// ═══════════════════════════════════════════════════════════════════════════

// RouteNotification маршрутизирует уведомление согласно региональной конфигурации.
func (r *IncidentRouter) RouteNotification(incidentID, framework, region, severity, eventType string) (*IncidentNotification, error) {
	route, ok := r.routes[region]
	if !ok {
		return nil, fmt.Errorf("incident_router: no route configured for region %s", region)
	}

	// Проверяем, что framework поддерживается для данного региона
	frameworkSupported := false
	for _, fw := range route.Frameworks {
		if fw == framework {
			frameworkSupported = true
			break
		}
	}
	if !frameworkSupported {
		return nil, fmt.Errorf("incident_router: framework %s not supported for region %s", framework, region)
	}

	channels := r.selectChannels(route, eventType)

	notification := &IncidentNotification{
		ID:         fmt.Sprintf("NTF-%s-%s", incidentID, time.Now().Format("20060102150405")),
		IncidentID: incidentID,
		Framework:  framework,
		Region:     region,
		Severity:   severity,
		EventType:  eventType,
		Channels:   channels,
		Title:      r.buildTitle(framework, severity, eventType),
		Body:       r.buildBody(incidentID, framework, region, severity, eventType, route.ReportingHours),
		CreatedAt:  time.Now().UTC(),
		Status:     "pending",
		RetryCount: 0,
	}

	r.mu.Lock()
	r.pending = append(r.pending, notification)
	r.mu.Unlock()

	r.logger.Info("incident_router: notification created",
		"notification_id", notification.ID,
		"incident_id", incidentID,
		"framework", framework,
		"region", region,
		"channels", channels,
	)

	// Отправляем немедленно
	r.sendNotification(notification)

	return notification, nil
}

// selectChannels выбирает каналы уведомления на основе eventType.
func (r *IncidentRouter) selectChannels(route RegionRouteConfig, eventType string) []NotificationChannel {
	switch eventType {
	case "initial_report", "deadline_missed":
		// Критические события — все каналы
		return append(route.PriorityChannels, route.BackupChannels...)
	case "reminder":
		// Напоминания — только priority
		return route.PriorityChannels
	case "escalation":
		// Эскалация — все каналы
		return append(route.PriorityChannels, route.BackupChannels...)
	case "acknowledgment":
		// Подтверждение — только email
		return []NotificationChannel{ChannelEmail}
	default:
		return route.PriorityChannels
	}
}

// sendNotification отправляет уведомление через все выбранные каналы.
func (r *IncidentRouter) sendNotification(notification *IncidentNotification) {
	for _, channel := range notification.Channels {
		sender, ok := r.senders[channel]
		if !ok || sender == nil {
			r.logger.Warn("incident_router: no sender for channel",
				"notification_id", notification.ID,
				"channel", channel,
			)
			continue
		}

		if err := sender.Send(notification); err != nil {
			r.logger.Error("incident_router: failed to send notification",
				"notification_id", notification.ID,
				"channel", channel,
				"error", err,
			)
			r.handleFailure(notification, channel, err)
		} else {
			r.logger.Info("incident_router: notification sent",
				"notification_id", notification.ID,
				"channel", channel,
			)
		}
	}

	now := time.Now().UTC()
	notification.SentAt = &now
	notification.Status = "sent"

	r.mu.Lock()
	r.history = append(r.history, notification)
	r.mu.Unlock()
}

// handleFailure обрабатывает неудачную отправку.
func (r *IncidentRouter) handleFailure(notification *IncidentNotification, channel NotificationChannel, err error) {
	notification.RetryCount++
	notification.Status = "failed"

	r.logger.Error("incident_router: notification failed",
		"notification_id", notification.ID,
		"channel", channel,
		"retry", notification.RetryCount,
		"max_retries", r.maxRetries,
		"error", err,
	)
}

// ═══════════════════════════════════════════════════════════════════════════
// Message Building
// ═══════════════════════════════════════════════════════════════════════════

func (r *IncidentRouter) buildTitle(framework, severity, eventType string) string {
	eventLabels := map[string]string{
		"initial_report":  "🚨 Initial Incident Report Required",
		"reminder":        "⏰ Incident Report Reminder",
		"escalation":      "🔴 Incident Escalation",
		"deadline_missed": "⚠️ CRITICAL: Reporting Deadline Missed",
		"acknowledgment":  "✅ Incident Report Acknowledged",
	}

	label, ok := eventLabels[eventType]
	if !ok {
		label = fmt.Sprintf("Incident Notification: %s", eventType)
	}

	return fmt.Sprintf("[%s] [%s] %s", framework, severity, label)
}

func (r *IncidentRouter) buildBody(incidentID, framework, region, severity, eventType string, reportingHours int) string {
	return fmt.Sprintf(`Incident ID: %s
Framework: %s
Region: %s
Severity: %s
Event: %s
Reporting Deadline: %d hours

Please submit the incident report to the regulatory authority within the required timeframe.

For more details, visit: https://cctv-monitor.io/incidents/%s
Contact: security@gb-telemetry.com`,
		incidentID, framework, region, severity, eventType, reportingHours, incidentID)
}

// ═══════════════════════════════════════════════════════════════════════════
// Query Methods
// ═══════════════════════════════════════════════════════════════════════════

// GetPending возвращает список ожидающих отправки уведомлений.
func (r *IncidentRouter) GetPending() []*IncidentNotification {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*IncidentNotification, 0)
	for _, n := range r.pending {
		if n.Status == "pending" {
			result = append(result, n)
		}
	}
	return result
}

// GetHistory возвращает историю отправленных уведомлений.
func (r *IncidentRouter) GetHistory(limit int) []*IncidentNotification {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 || limit > len(r.history) {
		limit = len(r.history)
	}

	// Возвращаем последние N записей
	start := len(r.history) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*IncidentNotification, limit)
	copy(result, r.history[start:])
	return result
}

// GetRoute возвращает конфигурацию маршрута для региона.
func (r *IncidentRouter) GetRoute(region string) (RegionRouteConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	route, ok := r.routes[region]
	return route, ok
}

// SetRoute устанавливает/обновляет конфигурацию маршрута для региона.
func (r *IncidentRouter) SetRoute(route RegionRouteConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[route.Region] = route
}

// ListRoutes возвращает все зарегистрированные маршруты.
func (r *IncidentRouter) ListRoutes() []RegionRouteConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]RegionRouteConfig, 0, len(r.routes))
	for _, route := range r.routes {
		result = append(result, route)
	}
	return result
}
