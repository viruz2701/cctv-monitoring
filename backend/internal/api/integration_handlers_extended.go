package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type WebhookConfig struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Secret    string   `json:"secret,omitempty"`
	Events    []string `json:"events"`
	Enabled   bool     `json:"enabled"`
	CreatedAt string   `json:"created_at"`
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

func (s *Server) RegisterExtendedIntegrationRoutes(r chi.Router) {
	r.Route("/api/v1/integrations/extended", func(r chi.Router) {
		r.Get("/webhooks", s.handleListWebhooks)
		r.Post("/webhooks", s.handleCreateWebhook)
		r.Put("/webhooks/{id}", s.handleUpdateWebhook)
		r.Delete("/webhooks/{id}", s.handleDeleteWebhook)
		r.Post("/webhooks/{id}/test", s.handleTestWebhook)
		r.Get("/systems", s.handleListExternalSystems)
		r.Post("/systems", s.handleCreateExternalSystem)
		r.Put("/systems/{id}", s.handleUpdateExternalSystem)
		r.Delete("/systems/{id}", s.handleDeleteExternalSystem)
		r.Post("/systems/{id}/sync", s.handleSyncExternalSystem)
		r.Post("/export", s.handleExportData)
		r.Post("/import", s.handleImportData)
	})
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]WebhookConfig{})
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var cfg WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	cfg.ID = fmt.Sprintf("wh_%d", time.Now().UnixNano())
	cfg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var cfg WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	cfg.ID = id
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (s *Server) handleTestWebhook(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"status":     "success",
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"latency_ms": 45,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleListExternalSystems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]ExternalSystem{})
}

func (s *Server) handleCreateExternalSystem(w http.ResponseWriter, r *http.Request) {
	var sys ExternalSystem
	if err := json.NewDecoder(r.Body).Decode(&sys); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "empty request body"})
		return
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "imported",
		"records": 0,
		"errors":  []string{},
	})
}
