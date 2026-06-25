// Package email — Email sending providers.
//
// NOTIF-01: Поддержка отправки email через SMTP для уведомлений менеджеров.
//
// Compliance:
//   - OWASP ASVS V7.1 (Log content — email адреса маскируются)
//   - OWASP ASVS V8 (Data Protection — credentials в env, не в коде)
//   - ISO 27001 A.9.4.3 (Password management — SMTP credentials)
package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// SMTPConfig — конфигурация SMTP провайдера.
// Поля получаются из env vars: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	From     string `json:"from"`
}

// SMTPProvider — реализация отправки email через SMTP.
type SMTPProvider struct {
	cfg    SMTPConfig
	logger *slog.Logger
}

func NewSMTPProvider(cfg SMTPConfig, logger *slog.Logger) *SMTPProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &SMTPProvider{cfg: cfg, logger: logger.With("component", "smtp")}
}

func (p *SMTPProvider) IsAvailable() bool {
	return p.cfg.Host != "" && p.cfg.User != "" && p.cfg.Password != ""
}

func (p *SMTPProvider) SendEmail(ctx context.Context, to, subject, body string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("smtp: not configured")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	addr := fmt.Sprintf("%s:%d", p.cfg.Host, p.cfg.Port)
	auth := smtp.PlainAuth("", p.cfg.User, p.cfg.Password, p.cfg.Host)

	msg := p.buildMessage(to, subject, body)

	if err := smtp.SendMail(addr, auth, p.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp: send email: %w", err)
	}

	p.logger.Info("email sent", "to", maskEmail(to), "subject", subject)
	return nil
}

func (p *SMTPProvider) buildMessage(to, subject, body string) string {
	headers := make(map[string]string)
	headers["From"] = p.cfg.From
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=\"UTF-8\""

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return msg.String()
}

func maskEmail(email string) string {
	at := strings.Index(email, "@")
	if at < 2 {
		return "***@***"
	}
	return email[:2] + "***@" + email[at+1:]
}
