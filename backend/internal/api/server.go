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
	"github.com/nats-io/nats.go"

	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/featureflag"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/cmms/factory"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/rca"
	"gb-telemetry-collector/internal/recaptcha"
	"gb-telemetry-collector/internal/service"
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

	// NATS connection for health checks
	natsConn *nats.Conn

	// Device service with audit trail (ISO 27001 A.12.4)
	deviceService *service.DeviceService

	// Feature Flag manager (F-0.2.4)
	featureFlags *featureflag.Manager

	// reCAPTCHA validator for public work request submission (WO-4.1.1)
	recaptchaValidator *recaptcha.Validator

	// RCA Engine (CCTV-2.1.3, AI-01)
	rcaEngine *rca.RCAEngine
}

// securityHeadersMiddleware добавляет security headers ко всем ответам.
// Соответствует: OWASP ASVS V5.3.3, ISO 27001 A.13.2.3, СТБ 34.101.27 п. 6.3
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем nonce из контекста (устанавливается CSPNonceMiddleware)
		nonce := NonceFromContext(r.Context())

		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// CSP with nonce (OWASP ASVS V5.3.3)
		// strict-dynamic отключает fallback к 'self' в старых браузерах — это нормально
		csp := fmt.Sprintf(
			"default-src 'self'; "+
				"script-src 'self' 'nonce-%s' 'strict-dynamic'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data: https:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'",
			nonce,
		)
		w.Header().Set("Content-Security-Policy", csp)
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
	// OWASP ASVS V13.4: ЗАПРЕЩЕНО использовать wildcard "*" в production.
	allowedOrigins := cfg.CORSAllowedOrigins
	for _, origin := range allowedOrigins {
		if origin == "*" {
			// Wildcard origin — security violation (OWASP ASVS V13.4)
			// Не fallback, а reject: если в конфиге "*" — используем default whitelist
			logger.Warn("CORS wildcard origin '*' detected and rejected! Falling back to safe defaults.",
				"action", "using localhost defaults per OWASP ASVS V13.4",
			)
			allowedOrigins = []string{"http://localhost:3000", "http://localhost:5173"}
			break
		}
	}
	if len(allowedOrigins) == 0 {
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

	// Инициализация reCAPTCHA валидатора (WO-4.1.1)
	recaptchaValidator := recaptcha.NewValidator(recaptcha.Config{
		SecretKey: cfg.RecaptchaSecretKey,
		SiteKey:   cfg.RecaptchaSiteKey,
		MinScore:  0.5,
		Enabled:   cfg.RecaptchaEnabled,
	})

	s := &Server{
		stateManager:       stateMgr,
		logger:             logger,
		db:                 database,
		imagesDir:          imagesDir,
		config:             cfg,
		sipHandler:         sipHandler,
		wsHub:              ws.NewHub(),
		cmmsRouter:         cmmsRouter,
		syncEngine:         syncEng,
		auditSigner:        mustNewAuditSigner(cfg.AuditHMACKey, logger),
		p2pGatewayURL:      cfg.P2PGatewayURL,
		p2pAPIKey:          cfg.P2PAPIKey,
		httpClient:         &http.Client{Timeout: 30 * time.Second},
		recaptchaValidator: recaptchaValidator,
	}
	// ── Device Service ────────────────────────────────────────────────
	s.deviceService = service.NewDeviceService(database, s.auditSigner, logger)

	go s.wsHub.Run()

	// ── Публичные маршруты (без JWT) ─────────────────────────────────
	s.mountHealthRoutes(r)
	s.mountAuthRoutes(r)

	// Публичный endpoint для подачи заявок (WO-4.1.1)
	// Rate limit: 10 req/min/IP
	r.Group(func(r chi.Router) {
		r.Use(s.workRequestRateLimiter)
		r.Post("/api/v1/public/work-requests", s.submitWorkRequest)
	})

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

		// Feature Flag domain (F-0.2.4)
		s.mountFeatureFlagRoutes(r)

		// GraphQL read-only endpoint (INT-13.2.4)
		s.mountGraphQLRoute(r)
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

// SetNATSConn устанавливает NATS соединение для health checks.
func (s *Server) SetNATSConn(conn *nats.Conn) {
	s.natsConn = conn
}

// SetFeatureFlagsManager устанавливает Feature Flag менеджер (F-0.2.4).
func (s *Server) SetFeatureFlagsManager(ff *featureflag.Manager) {
	s.featureFlags = ff
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
