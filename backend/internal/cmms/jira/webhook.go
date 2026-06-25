package jira

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gb-telemetry-collector/internal/webhook"
)

// WebhookHandler обрабатывает входящие webhook-уведомления от Jira.
type WebhookHandler struct {
	secret            string
	logger            *slog.Logger
	onWorkOrderUpdate func(workOrderID string, changes map[string]interface{}) error
	onAssetUpdate     func(assetID string, changes map[string]interface{}) error
}

// NewWebhookHandler создаёт обработчик webhook'ов Jira.
func NewWebhookHandler(secret string, logger *slog.Logger) *WebhookHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &WebhookHandler{secret: secret, logger: logger}
}

// OnWorkOrderUpdate регистрирует колбэк для обновлений WorkOrder.
func (h *WebhookHandler) OnWorkOrderUpdate(fn func(workOrderID string, changes map[string]interface{}) error) {
	h.onWorkOrderUpdate = fn
}

// OnAssetUpdate регистрирует колбэк для обновлений Asset.
func (h *WebhookHandler) OnAssetUpdate(fn func(assetID string, changes map[string]interface{}) error) {
	h.onAssetUpdate = fn
}

// ServeHTTP реализует http.Handler с HMAC-SHA256 верификацией через единый webhook пакет.
// Jira использует заголовок X-Hub-Signature-256 с префиксом "sha256=".
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := webhook.ServeHTTPWithVerify(h.secret, h.handleVerified,
		webhook.WithSignatureHeader("X-Hub-Signature-256"),
		webhook.WithSignaturePrefix("sha256="),
		webhook.WithLogger(h.logger),
	)
	handler.ServeHTTP(w, r)
}

// handleVerified вызывается после успешной HMAC-верификации.
func (h *WebhookHandler) handleVerified(w http.ResponseWriter, r *http.Request, body []byte) {
	var event jiraWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("jira webhook: unmarshal", "error", err)
		webhook.JSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	h.logger.Info("jira webhook received",
		"issue_key", event.Issue.Key,
		"event_type", event.WebhookEvent,
		"issue_type", event.Issue.Fields.IssueType.Name,
	)

	switch event.Issue.Fields.IssueType.Name {
	case "CCTV Work Order":
		if h.onWorkOrderUpdate != nil {
			changes := map[string]interface{}{
				"status":      event.Issue.Fields.Status.Name,
				"priority":    event.Issue.Fields.Priority.Name,
				"assignee":    event.Issue.Fields.Assignee,
				"resolution":  event.Issue.Fields.Resolution,
				"summary":     event.Issue.Fields.Summary,
				"description": event.Issue.Fields.Description,
			}
			if err := h.onWorkOrderUpdate(event.Issue.Key, changes); err != nil {
				h.logger.Error("jira webhook: onWorkOrderUpdate", "error", err)
				webhook.JSONError(w, http.StatusInternalServerError, "handler error")
				return
			}
		}
	case "Asset":
		if h.onAssetUpdate != nil {
			changes := map[string]interface{}{
				"status":  event.Issue.Fields.Status.Name,
				"summary": event.Issue.Fields.Summary,
			}
			if err := h.onAssetUpdate(event.Issue.Key, changes); err != nil {
				h.logger.Error("jira webhook: onAssetUpdate", "error", err)
				webhook.JSONError(w, http.StatusInternalServerError, "handler error")
				return
			}
		}
	}

	webhook.JSONOK(w)
}

// jiraWebhookEvent — структура входящего webhook от Jira.
type jiraWebhookEvent struct {
	WebhookEvent string           `json:"webhookEvent"`
	Issue        jiraWebhookIssue `json:"issue"`
}

// jiraWebhookIssue — упрощённая структура issue для webhook.
type jiraWebhookIssue struct {
	Key    string                 `json:"key"`
	Fields jiraWebhookIssueFields `json:"fields"`
}

// jiraWebhookIssueFields — типизированные поля webhook-issue.
type jiraWebhookIssueFields struct {
	Summary     string                 `json:"summary"`
	Description string                 `json:"description"`
	IssueType   struct{ Name string }  `json:"issuetype"`
	Status      struct{ Name string }  `json:"status"`
	Priority    struct{ Name string }  `json:"priority"`
	Assignee    map[string]interface{} `json:"assignee"`
	Resolution  map[string]interface{} `json:"resolution"`
}

// WebhookVerify — middleware для chi, проверяет Jira HMAC-SHA256 подпись.
// Использует единый webhook.VerifyMiddleware с префиксом "sha256=".
func WebhookVerify(secret string) func(http.Handler) http.Handler {
	return webhook.VerifyMiddleware(secret,
		webhook.WithSignatureHeader("X-Hub-Signature-256"),
		webhook.WithSignaturePrefix("sha256="),
	)
}
