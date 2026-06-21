package servicenow

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

// ServeHTTP реализует http.Handler с HMAC-SHA256 верификацией.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		h.logger.Error("servicenow webhook: read body", "error", err)
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	if !h.verifySignature(r.Header.Get("X-SN-Signature"), body) {
		h.logger.Warn("servicenow webhook: invalid signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	var event snWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.Error("servicenow webhook: unmarshal", "error", err)
		http.Error(w, "invalid json", http.StatusBadRequest)
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
				http.Error(w, "handler error", http.StatusInternalServerError)
				return
			}
		}
	case "cmdb_ci":
		if h.onAssetUpdate != nil {
			if err := h.onAssetUpdate(event.RecordID, event.Changes); err != nil {
				h.logger.Error("servicenow webhook: onAssetUpdate", "error", err)
				http.Error(w, "handler error", http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// verifySignature проверяет HMAC-SHA256 подпись.
func (h *WebhookHandler) verifySignature(sigHeader string, body []byte) bool {
	if h.secret == "" {
		return true
	}
	if sigHeader == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sigHeader), []byte(expected))
}

// snWebhookEvent — структура входящего webhook от ServiceNow.
type snWebhookEvent struct {
	Table    string                 `json:"table"`
	RecordID string                 `json:"record_id"`
	Action   string                 `json:"action"`
	Changes  map[string]interface{} `json:"changes"`
}

// WebhookVerify — middleware для chi, проверяет HMAC-SHA256 подпись.
func WebhookVerify(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if secret == "" {
				next.ServeHTTP(w, r)
				return
			}

			sig := r.Header.Get("X-SN-Signature")
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

			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(body)
			expected := hex.EncodeToString(mac.Sum(nil))

			if !hmac.Equal([]byte(sig), []byte(expected)) {
				http.Error(w, "invalid signature", http.StatusUnauthorized)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}
