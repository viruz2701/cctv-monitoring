package jira

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
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

// ServeHTTP реализует http.Handler с HMAC-SHA256 верификацией.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.logger.Error("jira webhook: read body", "error", err)
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	if !h.verifySignature(r.Header.Get("X-Hub-Signature-256"), body) {
		h.logger.Warn("jira webhook: invalid signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event jiraWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("jira webhook: unmarshal", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
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
				http.Error(w, "handler error", http.StatusInternalServerError)
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
				http.Error(w, "handler error", http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (h *WebhookHandler) verifySignature(sigHeader string, body []byte) bool {
	if h.secret == "" {
		return true
	}
	if sigHeader == "" {
		return false
	}

	// Jira использует префикс "sha256="
	sig := sigHeader
	if len(sigHeader) > 7 && sigHeader[:7] == "sha256=" {
		sig = sigHeader[7:]
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expected))
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
func WebhookVerify(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}

			sig := r.Header.Get("X-Hub-Signature-256")
			if sig == "" {
				http.Error(w, "missing signature", http.StatusUnauthorized)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "read error", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			rawSig := sig
			if len(sig) > 7 && sig[:7] == "sha256=" {
				rawSig = sig[7:]
			}

			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			expected := hex.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(rawSig), []byte(expected)) {
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}
