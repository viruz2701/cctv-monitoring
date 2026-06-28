// Package api — централизованная обработка HTTP-запросов.
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-ARCH.2: API Routes Organization
//
// server.go      — Server struct, конструктор, Start/Stop, вспомогательные
// router.go      — MountRoutes() — единый роутер с chi Router Groups
// response.go    — общие типы ответов (traceID, APIError, RespondError)
// middleware/    — standalone middleware (CORS, CSP, rate limiter, validation)
//
// Доменные роуты вынесены в *_routes.go файлы:
//
//	auth_routes.go, device_routes.go, cmms_routes.go, agent_routes.go,
//	integration_routes.go, compliance_routes.go, blackbox_routes.go,
//	camera_routes.go, storage_routes.go, featureflag_routes.go,
//	ai_routes.go, admin_handlers.go, tenant_compliance_routes.go
//
// Соответствует:
//   - OWASP ASVS L3 V1-V17
//   - ISO 27001 A.5-A.18
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
//
// ═══════════════════════════════════════════════════════════════════════════
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

	"gb-telemetry-collector/internal/ai"
	apimw "gb-telemetry-collector/internal/api/middleware"
	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/blackbox"
	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/cmms/factory"
	"gb-telemetry-collector/internal/compliance"
	"gb-telemetry-collector/internal/config"
	"gb-telemetry-collector/internal/db"
	"gb-telemetry-collector/internal/featureflag"
	"gb-telemetry-collector/internal/multiregion"
	"gb-telemetry-collector/internal/rca"
	"gb-telemetry-collector/internal/recaptcha"
	"gb-telemetry-collector/internal/service"
	"gb-telemetry-collector/internal/sip"
	"gb-telemetry-collector/internal/state"
	"gb-telemetry-collector/internal/storage"
	syncengine "gb-telemetry-collector/internal/sync"
	"gb-telemetry-collector/internal/telegram"
	"gb-telemetry-collector/internal/tenant"
	"gb-telemetry-collector/internal/webhook"
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

	// P2-AI.4: Anomaly Detection Service
	anomalyService *ai.AnomalyService

	// CMMS adapter — абстракция над Internal/Atlas CMMS
	cmmsRouter *cmms.CMMSRouter

	// Bi-directional ITSM sync engine
	syncEngine *syncengine.SyncEngine

	// Audit log HMAC signer (ISO 27001 A.12.4)
	auditSigner *audit.Signer
	// P3-2: Audit Chain Store (HMAC chain + compliance)
	auditChainStore *audit.ChainStore

	// P2P gateway integration
	p2pGatewayURL string
	p2pAPIKey     string
	httpClient    *http.Client

	// NATS connection for health checks
	natsConn     *nats.Conn
	natsRequired bool // если true — NATS unavailable = service unavailable

	// Device service with audit trail (ISO 27001 A.12.4)
	deviceService *service.DeviceService

	// Feature Flag manager (F-0.2.4)
	featureFlags *featureflag.Manager

	// reCAPTCHA validator for public work request submission (WO-4.1.1)
	recaptchaValidator *recaptcha.Validator

	// RCA Engine (CCTV-2.1.3, AI-01)
	rcaEngine *rca.RCAEngine

	// Compliance Engine (KF-15.1.1)
	complianceEngine *compliance.Engine

	// Black Box Incident Recorder (KF-15.2.4)
	blackboxRecorder *blackbox.Recorder

	// Auto-dispatcher service (P1-6)
	autoDispatcher *cmms.AutoDispatcher

	// Dispatch rules engine (P1-6)
	ruleEngine *cmms.RuleEngine

	// P2-3.3: Webhook delivery worker and store
	webhookStore   webhook.DeliveryStore
	deliveryWorker *webhook.DeliveryWorker

	// P3-1: Multi-Region Geo-Redundancy
	regionStore     multiregion.RegionStore
	failoverService *multiregion.FailoverService

	// P0-CE.5: Tenant Compliance Profile (SaaS)
	tenantComplianceStore *tenant.TenantComplianceStore
	complianceRegistry    *compliance.ProfileRegistry

	// P0-CE.6: Data Residency Enforcement
	storageEnforcer *storage.ResidencyEnforcer

	// P2-RU.2: 152-ФЗ Personal Data Manager
	personalDataManager *compliance.PersonalDataManager

	// P2-EU.1: GDPR Manager
	gdprManager *compliance.GDPRManager

	// Redis client for health checks and caching
	redisClient RedisClient

	// Server start time for uptime tracking (PERF.4)
	serverStart time.Time

	// P0-N1: SBOM (Software Bill of Materials) Provider
	sbomProvider *SBOMProvider
}

// securityHeadersMiddleware добавляет security headers ко всем ответам.
// Соответствует: OWASP ASVS V5.3.3, ISO 27001 A.13.2.3, СТБ 34.101.27 п. 6.3
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем nonce из контекста (устанавливается CSPNonceMiddleware)
		nonce := apimw.NonceFromContext(r.Context())

		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(self)")

		// CSP with nonce (OWASP ASVS V5.3.3)
		// strict-dynamic отключает fallback к 'self' в старых браузерах — это нормально
		// unpkg.com — CDN для Swagger UI (P3-DX.5: /api/v1/docs)
		// ⚠ 'unsafe-inline' ЗАПРЕЩЁН для OWASP ASVS L3 — используем nonce для стилей
		csp := fmt.Sprintf(
			"default-src 'self'; "+
				"script-src 'self' 'nonce-%s' 'strict-dynamic'; "+
				"style-src 'self' 'nonce-%s' https://fonts.googleapis.com https://unpkg.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data: https:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'",
			nonce, nonce,
		)
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

// NewServer создаёт новый экземпляр HTTP-сервера с настроенным роутером.
// Роуты монтируются через MountRoutes() с chi Router Groups.
func NewServer(addr string, stateMgr state.DeviceStateManager, logger *slog.Logger, database *db.DB, imagesDir string, cfg *config.Config, sipHandler *sip.SIPHandler, syncEng *syncengine.SyncEngine) *Server {
	// P0-N1: SBOM Provider (загружается из директории sbom/ при старте)
	// В production SBOM генерируется в CI/CD и копируется в sbom/ директорию.
	sbomProvider := NewSBOMProvider("./sbom", "unknown", "0.0.0-dev")
	r := chi.NewRouter()

	// TraceID — must be first for audit trail
	r.Use(TraceIDMiddleware)

	// CSP nonce generation (for HTML pages)
	r.Use(apimw.CSPNonceMiddleware)

	// Security headers
	r.Use(securityHeadersMiddleware)

	// CORS middleware (P0-SEC.2: OWASP ASVS L3 V9.1 compliance)
	// ISO 27001 A.13.2: только явно указанные origins, без wildcard.
	corsOpts, err := apimw.NewCORSHandler(cfg.CORSAllowedOrigins, cfg.Debug)
	if err != nil {
		logger.Error("CORS configuration rejected — startup failed",
			"error", err,
			"action", "fix cors_allowed_origins in config or environment",
		)
		panic(fmt.Sprintf("CORS validation failed: %v", err))
	}
	r.Use(cors.Handler(corsOpts))

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
		sbomProvider:       sbomProvider,
		serverStart:        time.Now(),
	}

	// Инициализация сервисов
	s.initServices()

	// WebSocket hub
	go s.wsHub.Run()

	// Монтирование всех маршрутов
	s.MountRoutes(r)

	// Таймауты HTTP-сервера — предотвращают зависание соединений
	// при медленных запросах к БД или атаках slowloris.
	// ReadHeaderTimeout должен быть меньше ReadTimeout для защиты заголовков.
	// WriteTimeout = Max(readTimeout, maxExpectedResponseTime) — даём время на ответ.
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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

// SetRedisClient устанавливает Redis клиент для health checks.
// Если установлен, будет проверяться в readiness и dependencies probes.
func (s *Server) SetRedisClient(client RedisClient) {
	s.redisClient = client
}

// SetNATSConn устанавливает NATS соединение для health checks.
// natsRequired указывает, обязателен ли NATS для readiness probe.
func (s *Server) SetNATSConn(conn *nats.Conn, natsRequired bool) {
	s.natsConn = conn
	s.natsRequired = natsRequired
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

// ═══════════════════════════════════════════════════════════════════════
// P3-DX.5: OpenAPI 3.1 + Swagger UI Handlers
// ═══════════════════════════════════════════════════════════════════════

// handleOpenAPIJSON serves the OpenAPI 3.1 specification as JSON.
// Endpoint: GET /api/v1/openapi.json
func (s *Server) handleOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	baseURL := fmt.Sprintf("%s://%s", schemeFromRequest(r), r.Host)
	ServeOpenAPIJSON(w, r, DefaultRoutes(), baseURL, "0.0.0-dev")
}

// handleSwaggerUI serves the Swagger UI HTML page.
// Endpoint: GET /api/v1/docs
func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	nonce := apimw.NonceFromContext(r.Context())
	ServeSwaggerUI(w, r, nonce)
}

// schemeFromRequest determines the HTTP scheme from the request.
func schemeFromRequest(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if fwd := r.Header.Get("X-Forwarded-Proto"); fwd != "" {
		return fwd
	}
	return "http"
}
