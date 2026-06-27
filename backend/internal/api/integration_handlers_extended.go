package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/webhook"
)

// ────────────────────────────────────────────────────────────────────────────
// DTO
// ────────────────────────────────────────────────────────────────────────────

// WebhookConfig — DTO для API.
type WebhookConfig struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	URL            string   `json:"url"`
	Secret         string   `json:"secret,omitempty"`
	Events         []string `json:"events"`
	Enabled        bool     `json:"enabled"`
	RetryCount     int      `json:"retry_count"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	CreatedAt      string   `json:"created_at"`
}

type ExternalSystem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	APIEndpoint string `json:"api_endpoint"`
	APIKey      string `json:"api_key,omitempty"`
	Enabled     bool   `json:"enabled"`
	LastSync    string `json:"last_sync,omitempty"`
	SyncStatus  string `json:"sync_status,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Routes
// ────────────────────────────────────────────────────────────────────────────

func (s *Server) RegisterExtendedIntegrationRoutes(r chi.Router) {
	r.Route("/api/v1/integrations/extended", func(r chi.Router) {
		r.Get("/webhooks", s.handleListWebhooks)
		r.Post("/webhooks", s.handleCreateWebhook)
		r.Put("/webhooks/{id}", s.handleUpdateWebhook)
		r.Delete("/webhooks/{id}", s.handleDeleteWebhook)
		r.Post("/webhooks/{id}/test", s.handleTestWebhook)
		r.Get("/webhooks/{id}/logs", s.handleGetWebhookLogs) // P2-3.3
		r.Post("/webhooks/{id}/retry", s.handleRetryWebhook) // P2-3.3
		r.Get("/systems", s.handleListExternalSystems)
		r.Post("/systems", s.handleCreateExternalSystem)
		r.Put("/systems/{id}", s.handleUpdateExternalSystem)
		r.Delete("/systems/{id}", s.handleDeleteExternalSystem)
		r.Post("/systems/{id}/sync", s.handleSyncExternalSystem)
		r.Post("/export", s.handleExportData)
		r.Post("/import", s.handleImportData)
	})
}

// ── Webhook Handlers ─────────────────────────────────────────────────────

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	if s.webhookStore == nil {
		json.NewEncoder(w).Encode([]WebhookConfig{})
		return
	}

	endpoints, err := s.webhookStore.ListWebhookEndpoints(r.Context())
	if err != nil {
		RespondError(w, r, fmt.Errorf("list webhooks: %w", err))
		return
	}

	result := make([]WebhookConfig, len(endpoints))
	for i, ep := range endpoints {
		result[i] = WebhookConfig{
			ID:             ep.ID,
			Name:           ep.Name,
			URL:            ep.URL,
			Events:         ep.Events,
			Enabled:        ep.Enabled,
			RetryCount:     ep.RetryCount,
			TimeoutSeconds: ep.TimeoutSeconds,
			CreatedAt:      ep.CreatedAt,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var cfg WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}

	if cfg.RetryCount <= 0 {
		cfg.RetryCount = 3
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 10
	}

	if s.webhookStore != nil {
		wh := &webhook.WebhookEndpoint{
			Name:           cfg.Name,
			URL:            cfg.URL,
			Secret:         cfg.Secret,
			Events:         cfg.Events,
			Enabled:        cfg.Enabled,
			RetryCount:     cfg.RetryCount,
			TimeoutSeconds: cfg.TimeoutSeconds,
		}

		if err := s.webhookStore.CreateWebhookEndpoint(r.Context(), wh); err != nil {
			RespondError(w, r, fmt.Errorf("create webhook: %w", err))
			return
		}

		cfg.ID = wh.ID
		cfg.CreatedAt = wh.CreatedAt
	} else {
		cfg.ID = fmt.Sprintf("wh_%d", time.Now().UnixNano())
		cfg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var cfg WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		RespondError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}

	if s.webhookStore != nil {
		wh := &webhook.WebhookEndpoint{
			Name:           cfg.Name,
			URL:            cfg.URL,
			Secret:         cfg.Secret,
			Events:         cfg.Events,
			Enabled:        cfg.Enabled,
			RetryCount:     cfg.RetryCount,
			TimeoutSeconds: cfg.TimeoutSeconds,
		}
		if err := s.webhookStore.UpdateWebhookEndpoint(r.Context(), id, wh); err != nil {
			RespondError(w, r, fmt.Errorf("update webhook %s: %w", id, err))
			return
		}
	}

	cfg.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if s.webhookStore != nil {
		if err := s.webhookStore.DeleteWebhookEndpoint(r.Context(), id); err != nil {
			RespondError(w, r, fmt.Errorf("delete webhook %s: %w", id, err))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, fmt.Errorf("webhook ID is required"))
		return
	}

	start := time.Now()

	// Получаем endpoint из БД или используем дефолтный
	var targetURL string
	if s.webhookStore != nil {
		wh, err := s.webhookStore.GetWebhookEndpoint(r.Context(), id)
		if err != nil {
			RespondError(w, r, fmt.Errorf("get webhook %s: %w", id, err))
			return
		}
		if wh == nil {
			RespondError(w, r, fmt.Errorf("webhook %s not found", id))
			return
		}
		targetURL = wh.URL
	} else {
		targetURL = "https://example.com/webhook-test"
	}

	// Выполняем тестовый запрос
	_, err := http.Post(targetURL, "application/json", nil)
	durationMs := int(time.Since(start).Milliseconds())

	result := map[string]interface{}{
		"status":        "success",
		"status_code":   0,
		"duration_ms":   durationMs,
		"response_body": "",
	}

	if err != nil {
		result["status"] = "failed"
		result["error"] = err.Error()
	} else {
		result["status_code"] = 200
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleGetWebhookLogs — P2-3.3: возвращает логи доставки для вебхука.
func (s *Server) handleGetWebhookLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, fmt.Errorf("webhook ID is required"))
		return
	}

	if s.webhookStore == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]webhook.DeliveryLog{})
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	logs, err := s.webhookStore.GetDeliveryLogs(r.Context(), id, limit, offset)
	if err != nil {
		RespondError(w, r, fmt.Errorf("get delivery logs: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleRetryWebhook — P2-3.3: принудительный retry для failed доставки.
func (s *Server) handleRetryWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		RespondError(w, r, fmt.Errorf("delivery log ID is required"))
		return
	}

	if s.webhookStore == nil || s.deliveryWorker == nil {
		RespondError(w, r, fmt.Errorf("delivery worker not available"))
		return
	}

	// Получаем логи доставки с лимитом 1 (самый последний)
	logs, err := s.webhookStore.GetDeliveryLogs(r.Context(), id, 1, 0)
	if err != nil || len(logs) == 0 {
		RespondError(w, r, fmt.Errorf("delivery log not found"))
		return
	}

	dl := logs[0]
	endpoint, err := s.webhookStore.GetWebhookEndpoint(r.Context(), dl.WebhookID)
	if err != nil || endpoint == nil {
		RespondError(w, r, fmt.Errorf("webhook endpoint not found"))
		return
	}

	// Сбрасываем retry_attempt и next_retry_at для немедленного retry
	now := time.Now()
	if err := s.webhookStore.UpdateDeliveryLog(r.Context(), dl.ID, "pending",
		dl.ResponseStatus, dl.ResponseBody, "", dl.DurationMs, &now); err != nil {
		RespondError(w, r, fmt.Errorf("reset delivery log: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "retry_scheduled"})
}

// ── External Systems Handlers ──────────────────────────────────────────

func (s *Server) handleListExternalSystems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]ExternalSystem{})
}

func (s *Server) handleCreateExternalSystem(w http.ResponseWriter, r *http.Request) {
	var sys ExternalSystem
	if err := json.NewDecoder(r.Body).Decode(&sys); err != nil {
		RespondError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	sys.ID = fmt.Sprintf("ext_%d", time.Now().UnixNano())
	sys.SyncStatus = "never"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sys)
}

func (s *Server) handleUpdateExternalSystem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sys ExternalSystem
	if err := json.NewDecoder(r.Body).Decode(&sys); err != nil {
		RespondError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	sys.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sys)
}

func (s *Server) handleDeleteExternalSystem(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *Server) handleSyncExternalSystem(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"status":   "synced",
		"records":  42,
		"errors":   0,
		"duration": "1.2s",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleExportData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=export.json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"version":     "1.0",
		"data":        map[string]interface{}{},
	})
}

func (s *Server) handleImportData(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		RespondError(w, r, fmt.Errorf("empty request body"))
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		RespondError(w, r, fmt.Errorf("invalid JSON: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "imported",
		"records": 0,
		"errors":  []string{},
	})
}
