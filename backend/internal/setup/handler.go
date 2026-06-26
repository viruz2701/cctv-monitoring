// Package setup — HTTP handler for Setup Wizard API.
//
// ═══════════════════════════════════════════════════════════════════════════
// P0-CE.4: Setup Wizard HTTP Handler
//
// API endpoints:
//
//	GET    /api/v1/setup/status      — статус мастера (started/completed/step)
//	POST   /api/v1/setup/start       — запуск мастера
//	GET    /api/v1/setup/regions     — список доступных регионов
//	POST   /api/v1/setup/region      — выбор региона
//	POST   /api/v1/setup/crypto      — подтверждение криптографии
//	POST   /api/v1/setup/storage     — настройка хранилища
//	POST   /api/v1/setup/admin       — создание администратора
//	POST   /api/v1/setup/network     — настройка сети
//	POST   /api/v1/setup/notifications — настройка уведомлений
//	POST   /api/v1/setup/complete    — завершение мастера
//
// ═══════════════════════════════════════════════════════════════════════════
package setup

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gb-telemetry-collector/internal/compliance"
)

// ────────────────────────────────────────────────────────────────────────────
// Handler
// ────────────────────────────────────────────────────────────────────────────

// Handler содержит HTTP handlers для Setup Wizard.
type Handler struct {
	wizard *SetupWizard
	logger *slog.Logger
}

// NewHandler создаёт новый HTTP handler для мастера.
func NewHandler(wizard *SetupWizard) *Handler {
	return &Handler{
		wizard: wizard,
		logger: wizard.logger,
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Request/Response types
// ────────────────────────────────────────────────────────────────────────────

type statusResponse struct {
	Started   bool       `json:"started"`
	Completed bool       `json:"completed"`
	Step      WizardStep `json:"step"`
	Config    any        `json:"config,omitempty"`
}

type regionRequest struct {
	Region string `json:"region"`
}

type cryptoRequest struct {
	Confirmed bool `json:"confirmed"`
}

type storageRequest struct {
	Type       string `json:"type"`
	S3Endpoint string `json:"s3_endpoint,omitempty"`
	S3Bucket   string `json:"s3_bucket,omitempty"`
	S3Region   string `json:"s3_region,omitempty"`
}

type adminRequest struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	Signature string `json:"signature,omitempty"`
}

type networkRequest struct {
	APIPort int    `json:"api_port"`
	TLSCert string `json:"tls_cert,omitempty"`
	TLSKey  string `json:"tls_key,omitempty"`
}

type notificationsRequest struct {
	TelegramToken string `json:"telegram_token,omitempty"`
	SMTPHost      string `json:"smtp_host,omitempty"`
	SMTPPort      int    `json:"smtp_port,omitempty"`
	SMTPUsername  string `json:"smtp_username,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
	Step  int    `json:"step,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Handlers
// ────────────────────────────────────────────────────────────────────────────

// HandleStatus возвращает статус мастера настройки.
func (h *Handler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{
		Started:   h.wizard.IsStarted(),
		Completed: h.wizard.IsCompleted(),
		Step:      h.wizard.CurrentStep(),
	}
	if h.wizard.IsStarted() && !h.wizard.IsCompleted() {
		resp.Config = h.wizard.Config()
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleStart запускает мастер настройки.
func (h *Handler) HandleStart(w http.ResponseWriter, r *http.Request) {
	if err := h.wizard.Start(); err != nil {
		writeError(w, http.StatusConflict, err.Error(), "SETUP_ALREADY_STARTED")
		return
	}
	writeJSON(w, http.StatusOK, statusResponse{
		Started: true,
		Step:    h.wizard.CurrentStep(),
	})
}

// HandleRegions возвращает список доступных регионов.
func (h *Handler) HandleRegions(w http.ResponseWriter, r *http.Request) {
	regions := AvailableRegions()
	writeJSON(w, http.StatusOK, map[string]any{
		"regions": regions,
	})
}

// HandleRegion выбирает регион.
func (h *Handler) HandleRegion(w http.ResponseWriter, r *http.Request) {
	var req regionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.wizard.SetRegion(req.Region); err != nil {
		code := "SETUP_ERROR"
		status := http.StatusBadRequest
		h.logger.Error("set region failed", "error", err)
		writeError(w, status, err.Error(), code)
		return
	}

	// Return crypto info for selected region
	var cryptoInfo *CryptoInfo
	for _, r := range AvailableRegions() {
		if r.Region == req.Region {
			cryptoInfo = &r.CryptoInfo
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step":        h.wizard.CurrentStep(),
		"crypto_info": cryptoInfo,
	})
}

// HandleCrypto подтверждает криптопараметры.
func (h *Handler) HandleCrypto(w http.ResponseWriter, r *http.Request) {
	var req cryptoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.wizard.ConfirmCrypto(req.Confirmed); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "CRYPTO_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step": h.wizard.CurrentStep(),
	})
}

// HandleStorage настраивает хранилище.
func (h *Handler) HandleStorage(w http.ResponseWriter, r *http.Request) {
	var req storageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.wizard.SetStorage(req.Type, req.S3Endpoint, req.S3Bucket, req.S3Region); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "STORAGE_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step": h.wizard.CurrentStep(),
	})
}

// HandleAdmin создаёт учётную запись администратора.
func (h *Handler) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	var req adminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.wizard.SetAdmin(req.Username, req.Email, req.Signature); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "ADMIN_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step": h.wizard.CurrentStep(),
	})
}

// HandleNetwork настраивает сеть.
func (h *Handler) HandleNetwork(w http.ResponseWriter, r *http.Request) {
	var req networkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.wizard.SetNetwork(req.APIPort, req.TLSCert, req.TLSKey); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "NETWORK_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step": h.wizard.CurrentStep(),
	})
}

// HandleNotifications настраивает уведомления.
func (h *Handler) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	var req notificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if err := h.wizard.SetNotifications(req.TelegramToken, req.SMTPHost, req.SMTPPort, req.SMTPUsername); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "NOTIFICATIONS_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"step": h.wizard.CurrentStep(),
	})
}

// HandleComplete завершает мастер настройки.
func (h *Handler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	if err := h.wizard.Complete(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "COMPLETE_ERROR")
		return
	}

	config := h.wizard.Config()
	writeJSON(w, http.StatusOK, map[string]any{
		"completed":         true,
		"region":            config.Region,
		"region_locked":     config.RegionLocked,
		"completed_at":      config.CompletedAt,
		"compliance_report": config.ComplianceReport,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// Routes registration
// ────────────────────────────────────────────────────────────────────────────

// RegisterRoutes регистрирует маршруты Setup Wizard API.
func RegisterRoutes(mux interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
}, wizard *SetupWizard) {
	h := NewHandler(wizard)

	mux.Get("/api/v1/setup/status", h.HandleStatus)
	mux.Get("/api/v1/setup/regions", h.HandleRegions)
	mux.Post("/api/v1/setup/start", h.HandleStart)
	mux.Post("/api/v1/setup/region", h.HandleRegion)
	mux.Post("/api/v1/setup/crypto", h.HandleCrypto)
	mux.Post("/api/v1/setup/storage", h.HandleStorage)
	mux.Post("/api/v1/setup/admin", h.HandleAdmin)
	mux.Post("/api/v1/setup/network", h.HandleNetwork)
	mux.Post("/api/v1/setup/notifications", h.HandleNotifications)
	mux.Post("/api/v1/setup/complete", h.HandleComplete)
}

// ────────────────────────────────────────────────────────────────────────────
// Response helpers
// ────────────────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, errorResponse{
		Error: msg,
		Code:  code,
	})
}

// ────────────────────────────────────────────────────────────────────────────
// NewWizardFromRegistry — convenience constructor
// ────────────────────────────────────────────────────────────────────────────

// NewWizardFromRegistry создаёт SetupWizard из compliance реестра.
func NewWizardFromRegistry(registry *compliance.ProfileRegistry, opts ...WizardOption) *SetupWizard {
	return NewSetupWizard(registry, opts...)
}
