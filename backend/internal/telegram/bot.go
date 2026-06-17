package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/models"
	"gb-telemetry-collector/internal/state"
)

// Bot represents Telegram bot instance
type Bot struct {
	api          *tgbotapi.BotAPI
	db           *db.DB
	stateManager state.DeviceStateManager
	logger       *slog.Logger
	updates      tgbotapi.UpdatesChannel
}

// Config holds Telegram bot configuration
type Config struct {
	Token  string
	Logger *slog.Logger
}

// NewBot creates new Telegram bot instance
func NewBot(cfg Config, database *db.DB, stateManager state.DeviceStateManager) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	api.Debug = false

	bot := &Bot{
		api:          api,
		db:           database,
		stateManager: stateManager,
		logger:       cfg.Logger,
	}

	return bot, nil
}

// Start begins listening for updates
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("Starting Telegram bot", "bot_name", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	b.updates = b.api.GetUpdatesChan(u)

	go b.handleUpdates(ctx)

	return nil
}

// Stop gracefully stops the bot
func (b *Bot) Stop() {
	b.logger.Info("Stopping Telegram bot")
	if b.updates != nil {
		b.updates.Clear()
	}
}

// handleUpdates processes incoming updates
func (b *Bot) handleUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-b.updates:
			if update.Message != nil {
				b.handleMessage(update.Message)
			} else if update.CallbackQuery != nil {
				b.handleCallbackQuery(update.CallbackQuery)
			}
		}
	}
}

// handleMessage processes text messages
func (b *Bot) handleMessage(message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	b.logger.Debug("Received message", "chat_id", chatID, "text", text)

	// Parse command
	if !strings.HasPrefix(text, "/") {
		return
	}

	parts := strings.Fields(text)
	command := strings.ToLower(parts[0])

	switch command {
	case "/start":
		b.handleStart(chatID, parts)
	case "/status":
		b.handleStatus(chatID)
	case "/alerts":
		b.handleAlerts(chatID, parts)
	case "/tickets":
		b.handleTickets(chatID)
	case "/help":
		b.handleHelp(chatID)
	default:
		b.sendMessage(chatID, "Unknown command. Use /help to see available commands.")
	}
}

// handleStart processes /start command for account linking
func (b *Bot) handleStart(chatID int64, parts []string) {
	if len(parts) < 2 {
		b.sendMessage(chatID, "Welcome to CCTV Monitor Bot!\n\nTo link your account, use:\n/start <token>\n\nGet your token from Profile → Telegram Notifications in the web app.")
		return
	}

	token := parts[1]

	// Validate token
	userID, err := b.db.GetTelegramLinkToken(token)
	if err != nil {
		b.sendMessage(chatID, "❌ Invalid or expired token. Please generate a new one in the web app.")
		return
	}

	// Update user's telegram_chat_id
	err = b.db.UpdateTelegramChatID(userID, strconv.FormatInt(chatID, 10))
	if err != nil {
		b.logger.Error("Failed to update telegram_chat_id", "error", err, "user_id", userID)
		b.sendMessage(chatID, "❌ Failed to link account. Please try again.")
		return
	}

	// Delete used token
	_ = b.db.DeleteTelegramLinkToken(token)

	b.sendMessage(chatID, "✅ Account linked successfully!\n\nYou will now receive alerts and notifications via Telegram.\n\nUse /help to see available commands.")
}

// handleStatus processes /status command
func (b *Bot) handleStatus(chatID int64) {
	// Get user by chat_id
	user, err := b.db.GetUserByTelegramChatID(strconv.FormatInt(chatID, 10))
	if err != nil {
		b.sendMessage(chatID, "❌ Your account is not linked. Use /start <token> to link.")
		return
	}

	// Get device statistics
	devices := b.stateManager.GetAll()
	onlineCount := 0
	offlineCount := 0
	for _, dev := range devices {
		if dev.Status == models.StatusOnline {
			onlineCount++
		} else {
			offlineCount++
		}
	}

	// Get open critical tickets
	tickets, err := b.db.GetTicketsByUserID(user.ID)
	if err != nil {
		b.logger.Error("Failed to get tickets", "error", err)
	}

	criticalCount := 0
	for _, ticket := range tickets {
		if ticket.Status == "open" && ticket.Priority == "critical" {
			criticalCount++
		}
	}

	msg := fmt.Sprintf(
		"📊 *Status Report*\n\n"+
			"👤 User: %s\n"+
			"🔧 Role: %s\n\n"+
			"📹 Devices:\n"+
			"  • Online: %d\n"+
			"  • Offline: %d\n\n"+
			"🎫 Critical Tickets: %d",
		user.Username,
		user.Role,
		onlineCount,
		offlineCount,
		criticalCount,
	)

	b.sendMessage(chatID, msg)
}

// handleAlerts processes /alerts command
func (b *Bot) handleAlerts(chatID int64, parts []string) {
	user, err := b.db.GetUserByTelegramChatID(strconv.FormatInt(chatID, 10))
	if err != nil {
		b.sendMessage(chatID, "❌ Your account is not linked. Use /start <token> to link.")
		return
	}

	if len(parts) < 2 {
		status := "disabled"
		if user.TelegramAlerts {
			status = "enabled"
		}
		b.sendMessage(chatID, fmt.Sprintf("🔔 Alerts are currently %s.\n\nUse:\n/alerts on - enable alerts\n/alerts off - disable alerts", status))
		return
	}

	action := strings.ToLower(parts[1])
	var enable bool

	switch action {
	case "on":
		enable = true
	case "off":
		enable = false
	default:
		b.sendMessage(chatID, "❌ Invalid option. Use: /alerts on or /alerts off")
		return
	}

	err = b.db.UpdateTelegramSettings(user.ID, enable, user.Telegram2FA)
	if err != nil {
		b.logger.Error("Failed to update telegram settings", "error", err)
		b.sendMessage(chatID, "❌ Failed to update settings. Please try again.")
		return
	}

	if enable {
		b.sendMessage(chatID, "✅ Alerts enabled. You will now receive alarm notifications.")
	} else {
		b.sendMessage(chatID, "✅ Alerts disabled.")
	}
}

// handleTickets processes /tickets command
func (b *Bot) handleTickets(chatID int64) {
	user, err := b.db.GetUserByTelegramChatID(strconv.FormatInt(chatID, 10))
	if err != nil {
		b.sendMessage(chatID, "❌ Your account is not linked. Use /start <token> to link.")
		return
	}

	tickets, err := b.db.GetTicketsByUserID(user.ID)
	if err != nil {
		b.logger.Error("Failed to get tickets", "error", err)
		b.sendMessage(chatID, "❌ Failed to fetch tickets.")
		return
	}

	if len(tickets) == 0 {
		b.sendMessage(chatID, "📭 No open tickets assigned to you.")
		return
	}

	// Show first 5 tickets with inline buttons
	msg := "🎫 *Your Open Tickets*\n\n"
	var buttons []tgbotapi.InlineKeyboardButton

	count := 0
	for _, ticket := range tickets {
		if ticket.Status != "open" || count >= 5 {
			break
		}
		msg += fmt.Sprintf("%d. [%s] %s\n", count+1, ticket.Priority, ticket.Title)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("Ack #%s", ticket.ID[:8]),
			fmt.Sprintf("ack_ticket:%s", ticket.ID),
		))
		count++
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(buttons...),
	)

	b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

// handleHelp processes /help command
func (b *Bot) handleHelp(chatID int64) {
	msg := "🤖 *CCTV Monitor Bot Commands*\n\n" +
		"/start <token> - Link your account\n" +
		"/status - View system status\n" +
		"/alerts on|off - Manage alert notifications\n" +
		"/tickets - View your open tickets\n" +
		"/help - Show this help message\n\n" +
		"🔐 *Security Features:*\n" +
		"• Telegram as 2FA method\n" +
		"• Login authorization via Telegram\n\n" +
		"Configure these features in Profile → Telegram Notifications"

	b.sendMessage(chatID, msg)
}

// handleCallbackQuery processes inline keyboard callbacks
func (b *Bot) handleCallbackQuery(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data

	b.logger.Debug("Received callback", "chat_id", chatID, "data", data)

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return
	}

	action := parts[0]
	id := parts[1]

	switch action {
	case "ack_ticket":
		b.handleAcknowledgeTicket(chatID, id, query.From.ID)
	}

	// Answer callback query
	callback := tgbotapi.NewCallback(query.ID, "Processing...")
	b.api.Send(callback)
}

// handleAcknowledgeTicket acknowledges a ticket
func (b *Bot) handleAcknowledgeTicket(chatID int64, ticketID string, telegramUserID int64) {
	user, err := b.db.GetUserByTelegramChatID(strconv.FormatInt(telegramUserID, 10))
	if err != nil {
		b.sendMessage(chatID, "❌ Your account is not linked.")
		return
	}

	err = b.db.UpdateTicketStatus(ticketID, "acknowledged", user.ID)
	if err != nil {
		b.logger.Error("Failed to acknowledge ticket", "error", err, "ticket_id", ticketID)
		b.sendMessage(chatID, "❌ Failed to acknowledge ticket.")
		return
	}

	b.sendMessage(chatID, fmt.Sprintf("✅ Ticket %s acknowledged.", ticketID[:8]))
}

// sendMessage sends a text message
func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}

// sendMessageWithKeyboard sends a message with inline keyboard
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

// SendAlarmNotification sends alarm notification to user
func (b *Bot) SendAlarmNotification(userID string, alarm *models.Alarm) error {
	user, err := b.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.TelegramAlerts || user.TelegramChatID == "" {
		return nil
	}

	chatID, err := strconv.ParseInt(user.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}

	msg := fmt.Sprintf(
		"🚨 *ALARM NOTIFICATION*\n\n"+
			"📹 Device: %s\n"+
			"⚠️ Priority: %s\n"+
			"🔧 Method: %s\n"+
			"📝 Description: %s\n"+
			"🕐 Time: %s",
		alarm.DeviceID,
		alarm.Priority,
		alarm.Method,
		alarm.Description,
		alarm.Timestamp.Format(time.RFC3339),
	)

	b.sendMessage(chatID, msg)
	return nil
}

// Send2FACode sends 2FA code to user via Telegram
func (b *Bot) Send2FACode(userID, code string) error {
	user, err := b.db.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if !user.Telegram2FA || user.TelegramChatID == "" {
		return fmt.Errorf("telegram 2fa not enabled for user")
	}

	chatID, err := strconv.ParseInt(user.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id: %w", err)
	}

	msg := fmt.Sprintf(
		"🔐 *Login Authorization*\n\n"+
			"Your verification code: *%s*\n\n"+
			"⚠️ Do not share this code with anyone.\n"+
			"Code expires in 5 minutes.",
		code,
	)

	b.sendMessage(chatID, msg)
	return nil
}

// GenerateLoginCode generates and sends login code for Telegram-based auth
func (b *Bot) GenerateLoginCode(chatID int64) (string, error) {
	user, err := b.db.GetUserByTelegramChatID(strconv.FormatInt(chatID, 10))
	if err != nil {
		return "", fmt.Errorf("account not linked")
	}

	return b.GenerateLoginCodeByUserID(user.ID)
}

// GenerateLoginCodeByUserID generates and sends login code by user ID
func (b *Bot) GenerateLoginCodeByUserID(userID string) (string, error) {
	user, err := b.db.GetUserByID(userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	if user.TelegramChatID == "" {
		return "", fmt.Errorf("telegram not linked")
	}

	chatID, err := strconv.ParseInt(user.TelegramChatID, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid chat_id: %w", err)
	}

	// Generate 6-digit code
	code := fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)

	// Save code to database (expires in 5 minutes)
	expiresAt := time.Now().Add(5 * time.Minute)
	err = b.db.SaveTelegramLoginCode(user.ID, code, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to save code: %w", err)
	}

	// Send code to user
	msg := fmt.Sprintf(
		"🔑 *Login Code*\n\n"+
			"Your login code: *%s*\n\n"+
			"Use this code to log in to the web app.\n"+
			"Code expires in 5 minutes.",
		code,
	)

	b.sendMessage(chatID, msg)
	return code, nil
}
