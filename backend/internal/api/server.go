package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/cmms/factory"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/sip"
	"gb-telemetry-collector/internal/state"
	syncengine "gb-telemetry-collector/internal/sync"
	"gb-telemetry-collector/internal/telegram"
	"gb-telemetry-collector/internal/ws"
)

// mustNewAuditSigner создаёт Signer и паникует при ошибке (фатально для старта).
func mustNewAuditSigner(key string, logger *slog.Logger) *audit.Signer {
	signer, err := audit.NewSigner(key)
	if err != nil {
		logger.Error("invalid audit HMAC key, refusing to start", "error", err)
		fmt.Fprintf(os.Stderr, "FATAL: %v\n", err)
		os.Exit(1)
	}
	return signer
}

// Server объединяет все зависимости HTTP-сервера.
type Server struct {
	stateManager state.DeviceStateManager
	logger       *slog.Logger
	db           *db.DB
	httpServer   *http.Server
	imagesDir    string
	config       *config.Config
	sipHandler   *sip.SIPHandler
	wsHub        *ws.Hub
	telegramBot  *telegram.Bot

	// CMMS adapter — абстракция над Internal/Atlas CMMS
	cmmsRouter *cmms.CMMSRouter

	// Bi-directional ITSM sync engine
	syncEngine *syncengine.SyncEngine

	// Audit log HMAC signer (ISO 27001 A.12.4)
	auditSigner *audit.Signer

	// P2P gateway integration
	p2pGatewayURL string
	p2pAPIKey     string
	httpClient    *http.Client
}

// securityHeadersMiddleware добавляет security headers ко всем ответам.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

// NewServer создаёт новый экземпляр HTTP-сервера с настроенным роутером.
// Роуты разбиты на доменные файлы: auth_routes, cmms_routes, device_routes, agent_routes, integration_routes.
func NewServer(addr string, stateMgr state.DeviceStateManager, logger *slog.Logger, database *db.DB, imagesDir string, cfg *config.Config, sipHandler *sip.SIPHandler, syncEng *syncengine.SyncEngine) *Server {
	r := chi.NewRouter()

	// TraceID — must be first for audit trail
	r.Use(TraceIDMiddleware)

	// CSP nonce generation (for HTML pages)
	r.Use(CSPNonceMiddleware)

	// Security headers
	r.Use(securityHeadersMiddleware)

	// CORS middleware (ISO 27001 A.13.2 — whitelist, не wildcard)
	allowedOrigins := cfg.CORSAllowedOrigins
	if len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
		allowedOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Инициализация CMMS Router
	cmmsRouter := factory.NewCMMSRouterFromConfig(cfg, database)

	s := &Server{
		stateManager:  stateMgr,
		logger:        logger,
		db:            database,
		imagesDir:     imagesDir,
		config:        cfg,
		sipHandler:    sipHandler,
		wsHub:         ws.NewHub(),
		cmmsRouter:    cmmsRouter,
		syncEngine:    syncEng,
		auditSigner:   mustNewAuditSigner(cfg.AuditHMACKey, logger),
		p2pGatewayURL: cfg.P2PGatewayURL,
		p2pAPIKey:     cfg.P2PAPIKey,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}
	go s.wsHub.Run()

	// ── Публичные маршруты (без JWT) ─────────────────────────────────
	s.mountHealthRoutes(r)
	s.mountAuthRoutes(r)

	// External alarm routes (P2P alarm with API key, etc.)
	s.mountExternalAlarmRoutes(r)

	// Legacy XML/Vigi alarm endpoints
	if cfg.HTTPXMLEnabled {
		r.Post("/api/v1/external/alarm/xml", s.handleExternalAlarmXML)
	}
	if cfg.VigiEnabled {
		r.Post("/api/v1/external/alarm/vigi", s.handleExternalAlarmVigi)
	}

	// ── Защищённые маршруты (JWT) ────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)

		// Auth domain (users, sessions, 2FA, Telegram, API keys, settings)
		s.mountProtectedAuthRoutes(r)

		// Device domain (devices, images, analytics, logs, audit)
		s.mountDeviceRoutes(r)

		// Agent domain (P2P, GB28181, WebSocket, external alarms)
		s.mountAgentRoutes(r)
		r.Post("/api/v1/external/alarm", s.handleExternalAlarm)

		// Integration domain (Atlas CMMS)
		s.mountIntegrationRoutes(r)

		// CMMS domain (maintenance, work orders, spare parts, SLA, mobile)
		s.mountCMMSRoutes(r)
	})

	// ── External API key auth ────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(s.APIKeyMiddleware)
		r.Post("/api/v1/external/alarm", s.handleExternalAlarm)
	})

	// ── ITSM Webhooks (HMAC, rate-limited) ───────────────────────────
	s.mountWebhookRoutes(r)

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: r,
	}
	return s
}

// Start запускает HTTP-сервер.
func (s *Server) Start() error {
	s.logger.Info("API server started", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Stop gracefully останавливает HTTP-сервер, давая активным запросам завершиться.
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// SetTelegramBot устанавливает экземпляр Telegram-бота для сервера.
func (s *Server) SetTelegramBot(bot *telegram.Bot) {
	s.telegramBot = bot
}

// ---------- Вспомогательные ----------

// jsonResponse отправляет JSON-ответ с заданным статус-кодом.
func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
