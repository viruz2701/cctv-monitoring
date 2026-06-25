// Package sla — SLA Breach Notifier (NOTIF-01).
//
// Multi-channel notifier для SLA breach уведомлений:
//   - Telegram: для техников (уже есть через telegram.Bot)
//   - SMS: для критических объектов (через SMS gateway interface)
//   - Email: для менеджеров (через Email sender interface)
//
// Уровни оповещения:
//   75% дедлайна (at_risk) → Telegram предупреждение
//   90% дедлайна (critical) → Telegram + Email менеджеру
//   100% дедлайна (breach) → Telegram + SMS (для critical) + Email
//
// Compliance:
//   - ISO 27001 A.12.4.1 (Event logging)
//   - IEC 62443 SR 2.8 (Audit events)
//   - ISO 27019 PCC.A.12.4 (ICS event logging)
//   - OWASP ASVS V7.1 (Log content — no sensitive data leakage)
//   - Приказ ОАЦ №66 п.7.18.6 (Мониторинг и реагирование)
package sla

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════
// NOTIF-01: SLA Breach Notification Channels
// ═══════════════════════════════════════════════════════════════════════

// NotificationLevel — уровень оповещения.
type NotificationLevel int

const (
	// NotificationAtRisk — 75% дедлайна (предупреждение)
	NotificationAtRisk NotificationLevel = iota
	// NotificationCritical — 90% дедлайна (критическое предупреждение)
	NotificationCritical
	// NotificationBreach — 100% дедлайна (нарушение)
	NotificationBreach
)

func (l NotificationLevel) String() string {
	switch l {
	case NotificationAtRisk:
		return "at_risk"
	case NotificationCritical:
		return "critical"
	case NotificationBreach:
		return "breach"
	default:
		return "unknown"
	}
}

// ═══════════════════════════════════════════════════════════════════════
// SMS Gateway Interface
// ═══════════════════════════════════════════════════════════════════════

// SMSProvider — интерфейс для отправки SMS.
// TODO: Реализовать через SMS-шлюз (например, Twilio, ClickSend, или СМС-центр РБ)
type SMSProvider interface {
	// SendSMS отправляет SMS-сообщение.
	SendSMS(ctx context.Context, phoneNumber, message string) error
	// IsAvailable проверяет доступность SMS-шлюза.
	IsAvailable() bool
}

// ═══════════════════════════════════════════════════════════════════════
// Email Sender Interface
// ═══════════════════════════════════════════════════════════════════════

// EmailSender — интерфейс для отправки Email.
// TODO: Реализовать через SMTP или email-сервис (SendGrid, AWS SES, или свой SMTP)
type EmailSender interface {
	// SendEmail отправляет email-сообщение.
	SendEmail(ctx context.Context, to, subject, body string) error
	// IsAvailable проверяет доступность email-сервиса.
	IsAvailable() bool
}

// ═══════════════════════════════════════════════════════════════════════
// User Contact Repository
// ═══════════════════════════════════════════════════════════════════════

// UserContactInfo — контактная информация пользователя для уведомлений.
type UserContactInfo struct {
	UserID        string `json:"user_id"`
	TelegramChatID string `json:"telegram_chat_id,omitempty"`
	PhoneNumber   string `json:"phone_number,omitempty"`
	Email         string `json:"email,omitempty"`
	Role          string `json:"role"`
}

// UserContactProvider — интерфейс для получения контактной информации.
type UserContactProvider interface {
	// GetContactInfo возвращает контактную информацию пользователя.
	GetContactInfo(ctx context.Context, userID string) (*UserContactInfo, error)
	// FindManagerForDevice находит менеджера, ответственного за устройство.
	FindManagerForDevice(ctx context.Context, deviceID string) (*UserContactInfo, error)
}

// ═══════════════════════════════════════════════════════════════════════
// SLA Notifier
// ═══════════════════════════════════════════════════════════════════════

// SLABreachNotifier — multi-channel notifier для SLA breach уведомлений.
//
// NOTIF-01: Отправляет уведомления при приближении дедлайна:
//   - 75% → at_risk: Telegram предупреждение технику
//   - 90% → critical: Telegram технику + Email менеджеру
//   - 100% → breach: Telegram технику + SMS (для critical) + Email менеджеру
//
// Compliance:
//   - ISO 27001 A.12.4.1: Все уведомления логируются
//   - IEC 62443 SR 2.8: Audit trail для критических событий
//   - OWASP ASVS V7.1: Сообщения не содержат sensitive data
type SLABreachNotifier struct {
	telegram    TelegramSender
	sms         SMSProvider
	email       EmailSender
	contacts    UserContactProvider
	logger      *slog.Logger
}

// TelegramSender — интерфейс для отправки Telegram сообщений.
type TelegramSender interface {
	SendTextMessage(chatID int64, text string)
}

// NewSLABreachNotifier создаёт новый SLA Breach Notifier.
//
// Все провайдеры опциональны (nil = канал отключён).
// Graceful degradation: если канал недоступен — уведомление отправляется
// через доступные каналы с логированием ошибки.
func NewSLABreachNotifier(
	telegram TelegramSender,
	sms SMSProvider,
	email EmailSender,
	contacts UserContactProvider,
	logger *slog.Logger,
) *SLABreachNotifier {
	if logger == nil {
		logger = slog.Default()
	}

	return &SLABreachNotifier{
		telegram: telegram,
		sms:      sms,
		email:    email,
		contacts: contacts,
		logger:   logger.With("component", "sla-notifier"),
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Notification Methods
// ═══════════════════════════════════════════════════════════════════════

// NotifyAtRisk отправляет предупреждение при 75% использованного SLA.
//
// Канал: Telegram технику
func (n *SLABreachNotifier) NotifyAtRisk(ctx context.Context, wo BreachedWorkOrder) error {
	n.logger.Info("SLA at risk notification",
		"work_order", wo.ID,
		"priority", wo.Priority,
		"deadline", wo.SLADeadline,
	)

	if n.telegram == nil {
		return nil
	}

	if wo.AssignedTo == "" {
		n.logger.Debug("cannot send at-risk notification: no assignee",
			"work_order", wo.ID,
		)
		return nil
	}

	contact, err := n.contacts.GetContactInfo(ctx, wo.AssignedTo)
	if err != nil {
		return fmt.Errorf("get contact info for at-risk: %w", err)
	}

	if contact.TelegramChatID == "" {
		n.logger.Debug("cannot send at-risk notification: no telegram chat",
			"user_id", wo.AssignedTo,
		)
		return nil
	}

	timeLeft := time.Until(wo.SLADeadline).Round(time.Minute)

	msg := fmt.Sprintf(
		"⚠️ *SLA на исходе!*\n\n"+
			"📋 Наряд: *%s*\n"+
			"🏷 Приоритет: %s\n"+
			"⏰ Осталось: *%s*\n"+
			"📹 Устройство: %s\n\n"+
			"❗️ Использовано 75%% времени. Примите меры.",
		escapeMarkdown(wo.Title),
		escapeMarkdown(wo.Priority),
		escapeMarkdown(formatDuration(timeLeft)),
		escapeMarkdown(wo.DeviceName),
	)

	chatID, err := parseChatID(contact.TelegramChatID)
	if err != nil {
		n.logger.Warn("invalid telegram chat_id for at-risk notification",
			"work_order", wo.ID,
			"error", err,
		)
		return nil
	}

	n.telegram.SendTextMessage(chatID, msg)
	return nil
}

// NotifyCritical отправляет критическое предупреждение при 90% SLA.
//
// Каналы:
//   - Telegram технику
//   - Email менеджеру (если настроен)
func (n *SLABreachNotifier) NotifyCritical(ctx context.Context, wo BreachedWorkOrder) error {
	n.logger.Warn("SLA critical notification",
		"work_order", wo.ID,
		"priority", wo.Priority,
		"deadline", wo.SLADeadline,
	)

	// Telegram технику
	if n.telegram != nil && wo.AssignedTo != "" {
		n.sendTelegramCritical(ctx, wo)
	}

	// Email менеджеру
	if n.email != nil && n.email.IsAvailable() {
		n.sendEmailCritical(ctx, wo)
	}

	return nil
}

// NotifyBreach отправляет уведомление о нарушении SLA.
//
// Каналы:
//   - Telegram технику
//   - SMS менеджеру (для CRITICAL/HIGH приоритетов)
//   - Email менеджеру
func (n *SLABreachNotifier) NotifyBreach(ctx context.Context, wo BreachedWorkOrder) error {
	n.logger.Error("SLA breach notification",
		"work_order", wo.ID,
		"priority", wo.Priority,
		"deadline", wo.SLADeadline,
	)

	// Telegram технику
	if n.telegram != nil && wo.AssignedTo != "" {
		n.sendTelegramBreach(ctx, wo)
	}

	// Находим менеджера для эскалации
	manager, err := n.contacts.FindManagerForDevice(ctx, wo.DeviceID)
	if err != nil {
		n.logger.Debug("no manager found for device escalation",
			"device_id", wo.DeviceID,
			"error", err,
		)
		// Не фатально — продолжаем с другими каналами
	}

	if manager == nil {
		return nil
	}

	// SMS для критических объектов (только CRITICAL/HIGH priority)
	if n.sms != nil && n.sms.IsAvailable() && (wo.Priority == "CRITICAL" || wo.Priority == "HIGH") {
		n.sendSMSBreach(ctx, wo, manager)
	}

	// Email менеджеру
	if n.email != nil && n.email.IsAvailable() {
		n.sendEmailBreach(ctx, wo, manager)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════
// Internal senders
// ═══════════════════════════════════════════════════════════════════════

func (n *SLABreachNotifier) sendTelegramCritical(ctx context.Context, wo BreachedWorkOrder) {
	contact, err := n.contacts.GetContactInfo(ctx, wo.AssignedTo)
	if err != nil {
		n.logger.Warn("failed to get contact for critical notification",
			"work_order", wo.ID, "error", err,
		)
		return
	}

	if contact.TelegramChatID == "" {
		return
	}

	timeLeft := time.Until(wo.SLADeadline).Round(time.Minute)

	msg := fmt.Sprintf(
		"🔴 *SLA: КРИТИЧЕСКИЙ УРОВЕНЬ!*\n\n"+
			"📋 Наряд: *%s*\n"+
			"🏷 Приоритет: %s\n"+
			"⏰ Осталось: *%s*\n"+
			"📹 Устройство: %s\n\n"+
			"🚨 Использовано 90%% времени. Требуется немедленное вмешательство!",
		escapeMarkdown(wo.Title),
		escapeMarkdown(wo.Priority),
		escapeMarkdown(formatDuration(timeLeft)),
		escapeMarkdown(wo.DeviceName),
	)

	chatID, err := parseChatID(contact.TelegramChatID)
	if err != nil {
		return
	}

	n.telegram.SendTextMessage(chatID, msg)
}

func (n *SLABreachNotifier) sendTelegramBreach(ctx context.Context, wo BreachedWorkOrder) {
	contact, err := n.contacts.GetContactInfo(ctx, wo.AssignedTo)
	if err != nil {
		n.logger.Warn("failed to get contact for breach notification",
			"work_order", wo.ID, "error", err,
		)
		return
	}

	if contact.TelegramChatID == "" {
		return
	}

	overdue := time.Since(wo.SLADeadline).Round(time.Minute)

	msg := fmt.Sprintf(
		"❌ *SLA НАРУШЕНО!*\n\n"+
			"📋 Наряд: *%s*\n"+
			"🏷 Приоритет: %s\n"+
			"⏰ Просрочено: *%s*\n"+
			"📹 Устройство: %s\n"+
			"🆔 ID: `%s`\n\n"+
			"🔔 Требуется эскалация!",
		escapeMarkdown(wo.Title),
		escapeMarkdown(wo.Priority),
		escapeMarkdown(formatDuration(overdue)),
		escapeMarkdown(wo.DeviceName),
		shortID(wo.ID),
	)

	chatID, err := parseChatID(contact.TelegramChatID)
	if err != nil {
		return
	}

	n.telegram.SendTextMessage(chatID, msg)
}

func (n *SLABreachNotifier) sendSMSBreach(ctx context.Context, wo BreachedWorkOrder, manager *UserContactInfo) {
	if manager.PhoneNumber == "" {
		n.logger.Debug("cannot send SMS breach: manager has no phone",
			"work_order", wo.ID,
		)
		return
	}

	overdue := time.Since(wo.SLADeadline).Round(time.Hour)

	msg := fmt.Sprintf(
		"[CCTV Monitor] SLA BREACH: %s (%s). Priority: %s. Overdue: %s. Device: %s",
		wo.Title,
		shortID(wo.ID),
		wo.Priority,
		formatDuration(overdue),
		wo.DeviceName,
	)

	if err := n.sms.SendSMS(ctx, manager.PhoneNumber, msg); err != nil {
		n.logger.Error("failed to send SMS breach notification",
			"work_order", wo.ID,
			"phone", maskPhone(manager.PhoneNumber),
			"error", err,
		)
	}
}

func (n *SLABreachNotifier) sendEmailBreach(ctx context.Context, wo BreachedWorkOrder, manager *UserContactInfo) {
	if manager.Email == "" {
		n.logger.Debug("cannot send email breach: manager has no email",
			"work_order", wo.ID,
		)
		return
	}

	overdue := time.Since(wo.SLADeadline).Round(time.Hour)

	subject := fmt.Sprintf("[SLA BREACH] %s - %s", wo.Priority, wo.Title)

	body := fmt.Sprintf(
		"Уважаемый менеджер,\n\n"+
			"Обнаружено нарушение SLA по наряду:\n\n"+
			"Наряд: %s\n"+
			"ID: %s\n"+
			"Приоритет: %s\n"+
			"Устройство: %s\n"+
			"Дедлайн: %s\n"+
			"Просрочено: %s\n\n"+
			"Техник: %s\n\n"+
			"Требуется ваше вмешательство.\n\n"+
			"С уважением,\n"+
			"CCTV Health Monitor",
		wo.Title,
		wo.ID,
		wo.Priority,
		wo.DeviceName,
		wo.SLADeadline.Format("2006-01-02 15:04 MST"),
		formatDuration(overdue),
		wo.AssigneeName,
	)

	if err := n.email.SendEmail(ctx, manager.Email, subject, body); err != nil {
		n.logger.Error("failed to send email breach notification",
			"work_order", wo.ID,
			"email", maskEmail(manager.Email),
			"error", err,
		)
	}
}

func (n *SLABreachNotifier) sendEmailCritical(ctx context.Context, wo BreachedWorkOrder) {
	manager, err := n.contacts.FindManagerForDevice(ctx, wo.DeviceID)
	if err != nil || manager == nil || manager.Email == "" {
		return
	}

	timeLeft := time.Until(wo.SLADeadline).Round(time.Minute)

	subject := fmt.Sprintf("[SLA CRITICAL] %s - %s", wo.Priority, wo.Title)

	body := fmt.Sprintf(
		"Уважаемый менеджер,\n\n"+
			"SLA по наряду на критическом уровне (90%% времени использовано):\n\n"+
			"Наряд: %s\n"+
			"ID: %s\n"+
			"Приоритет: %s\n"+
			"Устройство: %s\n"+
			"Дедлайн: %s\n"+
			"Осталось: %s\n\n"+
			"Техник: %s\n\n"+
			"Просьба принять меры.\n\n"+
			"С уважением,\n"+
			"CCTV Health Monitor",
		wo.Title,
		wo.ID,
		wo.Priority,
		wo.DeviceName,
		wo.SLADeadline.Format("2006-01-02 15:04 MST"),
		formatDuration(timeLeft),
		wo.AssigneeName,
	)

	if err := n.email.SendEmail(ctx, manager.Email, subject, body); err != nil {
		n.logger.Error("failed to send email critical notification",
			"work_order", wo.ID,
			"error", err,
		)
	}
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

// formatDuration форматирует duration в человекочитаемый вид.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dч %dмин", hours, minutes)
	}
	return fmt.Sprintf("%d мин", minutes)
}

// maskPhone маскирует номер телефона для логов (OWASP ASVS V7.1).
func maskPhone(phone string) string {
	if len(phone) < 5 {
		return "***"
	}
	if len(phone) < 6 {
		return phone[:3] + "***" + phone[len(phone)-2:]
	}
	return phone[:4] + "***" + phone[len(phone)-2:]
}

// shortID возвращает первые 8 символов ID (или весь ID, если короче).
func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// maskEmail маскирует email для логов (OWASP ASVS V7.1).
func maskEmail(email string) string {
	at := -1
	for i, ch := range email {
		if ch == '@' {
			at = i
			break
		}
	}
	if at < 2 {
		return "***@***"
	}
	return email[:2] + "***@" + email[at+1:]
}
