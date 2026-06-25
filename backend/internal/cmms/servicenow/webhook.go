package servicenow

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gb-telemetry-collector/internal/webhook"
)

// WebhookHandler обрабатывает входящие webhook-уведомления от ServiceNow.
type WebhookHandler struct {
	secret            string
	logger            *slog.Logger
	onWorkOrderUpdate func(workOrderID string, changes map[string]interface{}) error
	onAssetUpdate     func(assetID string, changes map[string]interface{}) error
}

// NewWebhookHandler создаёт обработчик webhook'ов.
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

// OnAssetUpdate регистрирует колбэк для обновлений CMDB Asset.
func (h *WebhookHandler) OnAssetUpdate(fn func(assetID string, changes map[string]interface{}) error) {
	h.onAssetUpdate = fn
}

// ServeHTTP реализует http.Handler с HMAC-SHA256 верификацией через единый webhook пакет.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler := webhook.ServeHTTPWithVerify(h.secret, h.handleVerified,
		webhook.WithSignatureHeader("X-SN-Signature"),
		webhook.WithLogger(h.logger),
	)
	handler.ServeHTTP(w, r)
}

// handleVerified вызывается после успешной HMAC-верификации.
func (h *WebhookHandler) handleVerified(w http.ResponseWriter, r *http.Request, body []byte) {
	var event snWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("servicenow webhook: unmarshal", "error", err)
		webhook.JSONError(w, http.StatusBadRequest, "invalid json")
		return
	}

	h.logger.Info("servicenow webhook received",
		"table", event.Table,
		"record_id", event.RecordID,
		"action", event.Action,
	)

	switch event.Table {
	case TableWorkOrder:
		if h.onWorkOrderUpdate != nil {
			if err := h.onWorkOrderUpdate(event.RecordID, event.Changes); err != nil {
				h.logger.Error("servicenow webhook: onWorkOrderUpdate", "error", err)
				webhook.JSONError(w, http.StatusInternalServerError, "handler error")
				return
			}
		}
	case "cmdb_ci":
		if h.onAssetUpdate != nil {
			if err := h.onAssetUpdate(event.RecordID, event.Changes); err != nil {
				h.logger.Error("servicenow webhook: onAssetUpdate", "error", err)
				webhook.JSONError(w, http.StatusInternalServerError, "handler error")
				return
			}
		}
	}

	webhook.JSONOK(w)
}

// snWebhookEvent — структура входящего webhook от ServiceNow.
type snWebhookEvent struct {
	Table    string                 `json:"table"`
	RecordID string                 `json:"record_id"`
	Action   string                 `json:"action"`
	Changes  map[string]interface{} `json:"changes"`
}

// WebhookVerify — middleware для chi, проверяет HMAC-SHA256 подпись.
// Использует единый webhook.VerifyMiddleware.
func WebhookVerify(secret string) func(http.Handler) http.Handler {
	return webhook.VerifyMiddleware(secret,
		webhook.WithSignatureHeader("X-SN-Signature"),
	)
}
