// Package api — Calendar Sync HTTP handlers (P1-CALENDAR).
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/integrations/calendar"
)

// ── Calendar Sync Handler ─────────────────────────────────────────────

// CalendarHandler группирует HTTP-обработчики для Calendar Sync.
type CalendarHandler struct {
	syncEngine *calendar.SyncEngine
	store      calendar.SyncStore
}

// NewCalendarHandler создаёт новый CalendarHandler.
func NewCalendarHandler(syncEngine *calendar.SyncEngine, store calendar.SyncStore) *CalendarHandler {
	return &CalendarHandler{
		syncEngine: syncEngine,
		store:      store,
	}
}

// ── Provider List ─────────────────────────────────────────────────────

// handleListProviders возвращает список доступных календарь-провайдеров.
//
// GET /api/v1/integrations/calendar/providers
func (h *CalendarHandler) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providers := []map[string]interface{}{
		{
			"id":          "google",
			"name":        "Google Calendar",
			"description": "Google Calendar integration via OAuth2",
			"enabled":     true,
		},
		{
			"id":          "outlook",
			"name":        "Microsoft Outlook",
			"description": "Microsoft 365 Calendar integration via OAuth2",
			"enabled":     true,
		},
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

// ── OAuth2 Connect ────────────────────────────────────────────────────

// handleConnect инициирует OAuth2 авторизацию для провайдера.
//
// POST /api/v1/integrations/calendar/{provider}/connect
//
// Body: { "redirect_url": "https://..." }
// Response: { "auth_url": "https://oauth2.provider.com/auth?..." }
func (h *CalendarHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider != "google" && provider != "outlook" {
		RespondError(w, r, NewBadRequestError("unsupported provider: "+provider))
		return
	}

	var req struct {
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, r, NewBadRequestError("invalid request body"))
		return
	}

	if req.RedirectURL == "" {
		RespondError(w, r, NewBadRequestError("redirect_url is required"))
		return
	}

	// В production здесь генерируется state + PKCE challenge и возвращается auth URL
	// state сохраняется в сессии/кэше для верификации callback.
	//
	// TODO: Реализовать полноценный OAuth2 flow с PKCE
	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"auth_url":      "https://accounts.google.com/o/oauth2/auth?...", // placeholder
		"provider":      provider,
		"state":         "generated-state-token",
		"pkce_required": true,
	})
}

// ── OAuth2 Disconnect ─────────────────────────────────────────────────

// handleDisconnect отключает провайдера и удаляет токены.
//
// POST /api/v1/integrations/calendar/{provider}/disconnect
func (h *CalendarHandler) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider != "google" && provider != "outlook" {
		RespondError(w, r, NewBadRequestError("unsupported provider: "+provider))
		return
	}

	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		RespondError(w, r, NewUnauthorizedError("user not authenticated"))
		return
	}

	conn, err := h.store.GetConnection(r.Context(), provider, userID)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to get connection", err))
		return
	}
	if conn == nil {
		RespondError(w, r, NewNotFoundError("no connection found for provider"))
		return
	}

	if err := h.store.DeleteConnection(r.Context(), conn.ID); err != nil {
		RespondError(w, r, NewInternalError("failed to disconnect", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":   "disconnected",
		"provider": provider,
	})
}

// ── Connection Status ─────────────────────────────────────────────────

// handleStatus возвращает статус подключения для провайдера.
//
// GET /api/v1/integrations/calendar/{provider}/status
func (h *CalendarHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if provider != "google" && provider != "outlook" {
		RespondError(w, r, NewBadRequestError("unsupported provider: "+provider))
		return
	}

	userID := r.Context().Value("user_id").(string)
	if userID == "" {
		RespondError(w, r, NewUnauthorizedError("user not authenticated"))
		return
	}

	conn, err := h.store.GetConnection(r.Context(), provider, userID)
	if err != nil {
		RespondError(w, r, NewInternalError("failed to get connection", err))
		return
	}

	if conn == nil {
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"provider":  provider,
			"connected": false,
			"calendar":  nil,
		})
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"provider":    provider,
		"connected":   conn.Enabled,
		"calendar_id": conn.CalendarID,
		"created_at":  conn.CreatedAt,
		"updated_at":  conn.UpdatedAt,
	})
}

// ── Manual Sync ───────────────────────────────────────────────────────

// handleSync запускает ручную синхронизацию.
//
// POST /api/v1/integrations/calendar/sync
func (h *CalendarHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	changes, err := h.syncEngine.PullChanges(r.Context())
	if err != nil {
		RespondError(w, r, NewInternalError("sync failed", err))
		return
	}

	jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "completed",
		"changes": len(changes),
	})
}

// ── OAuth2 Callback ───────────────────────────────────────────────────

// handleCallback обрабатывает OAuth2 callback от провайдера.
//
// GET /api/v1/integrations/calendar/{provider}/callback?code=...&state=...
func (h *CalendarHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" {
		RespondError(w, r, NewBadRequestError("authorization code is required"))
		return
	}

	_ = state // В production верифицируется с сохранённым значением

	// В production здесь обмениваем code на токены через OAuth2 провайдера
	// и сохраняем в calendar_connections.
	//
	// TODO: Реализовать полноценный OAuth2 callback handler с PKCE
	_ = provider

	jsonResponse(w, http.StatusOK, map[string]string{
		"status":   "connected",
		"provider": provider,
	})
}
