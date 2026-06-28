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
	"gb-telemetry-collector/internal/audit"
	"gb-telemetry-collector/internal/auth"
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
	"gb-telemetry-collector/internal/setup"
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
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(self)")

		// CSP with nonce (OWASP ASVS V5.3.3)
		// strict-dynamic отключает fallback к 'self' в старых браузерах — это нормально
		// unpkg.com — CDN для Swagger UI (P3-DX.5: /api/v1/docs)
		csp := fmt.Sprintf(
			"default-src 'self'; "+
				"script-src 'self' 'nonce-%s' 'strict-dynamic' https://unpkg.com; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://unpkg.com; "+
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

	// CORS middleware (P0-SEC.2: OWASP ASVS L3 V9.1 compliance)
	// ISO 27001 A.13.2: только явно указанные origins, без wildcard.
	corsOpts, err := NewCORSHandler(cfg.CORSAllowedOrigins, cfg.Debug)
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
		serverStart:        time.Now(),
	}

	// ── P2-AI.4: Anomaly Detection Service ─────────────────────────
	{
		anomalyCfg := ai.DefaultAnomalyConfig()
		var anomalyBroadcaster ai.Broadcaster
		if s.wsHub != nil {
			anomalyBroadcaster = s.wsHub
		}
		anomalyService, err := ai.NewAnomalyService(anomalyCfg, s.natsConn, anomalyBroadcaster, logger)
		if err != nil {
			logger.Warn("anomaly service init warning", "error", err)
		} else {
			s.anomalyService = anomalyService
			logger.Info("anomaly detection service initialized",
				"z_score_threshold", anomalyCfg.ZScoreThreshold,
				"ma_window", anomalyCfg.MovingAverageWindow,
			)
		}
	}

	// ── Device Service ────────────────────────────────────────────────
	s.deviceService = service.NewDeviceService(database, s.auditSigner, logger)

	// ── Compliance Engine (KF-15.1.1) ─────────────────────────────────
	s.complianceEngine = compliance.NewEngine(nil, logger, nil)

	// ── P2-RU.2: 152-ФЗ Personal Data Manager ─────────────────────────
	pdStore := compliance.NewMemoryPersonalDataStore(logger)
	s.personalDataManager = compliance.NewPersonalDataManager(pdStore, logger)

	// ── P2-EU.1: GDPR Manager ─────────────────────────────────────────
	gdprStore := compliance.NewMemoryGDPRStore(logger)
	s.gdprManager = compliance.NewGDPRManager(gdprStore, logger)

	// ── Black Box Incident Recorder (KF-15.2.4) ───────────────────────
	bbRepo := blackbox.NewDBRepository(database.Pool, logger)
	s.blackboxRecorder = blackbox.NewRecorder(bbRepo, database, nil, logger)

	// ── Auto-dispatcher Service (P1-6) ────────────────────────────────
	// WorkOrderProvider инициализируется nil, так как требуется адаптация
	// к существующему CMMSAdapter. Будет подключён через SetWorkOrderProvider
	// при инициализации CMMS адаптера.
	s.autoDispatcher = cmms.NewAutoDispatcher(
		nil, // TechnicianProvider — будет подключён при инициализации workforce
		nil, // WorkOrderProvider — будет подключён при инициализации CMMS
		nil, // SLAStatusChecker — будет подключён при инициализации SLA
		cmms.DispatcherAuditLoggerFunc(func(ctx context.Context, entry *cmms.DispatchAuditEntry) error {
			userID := "system"
			s.logAudit(userID, entry.Action, "dispatch", entry.WorkOrderID, nil, entry)
			return nil
		}),
		cmms.DefaultAutoDispatcherConfig,
		logger,
	)

	// ── Dispatch Rules Engine (P1-6) ──────────────────────────────────
	s.ruleEngine = cmms.NewRuleEngine(logger)

	// ── P2-3.3: Webhook Delivery Worker ─────────────────────────────
	if database != nil && database.Pool != nil {
		s.webhookStore = webhook.NewPGDeliveryStore(database.Pool)
		s.deliveryWorker = webhook.NewDeliveryWorker(
			s.webhookStore, logger,
			webhook.DeliveryWorkerConfig{
				PollInterval:  5 * time.Second,
				MaxConcurrent: 5,
			},
		)
		go s.deliveryWorker.Start(context.Background())
	}

	// ── P3-1: Multi-Region Geo-Redundancy ──────────────────────────
	if database != nil && database.Pool != nil {
		s.regionStore = multiregion.NewPGTenantRegionStore(database.Pool)
		s.failoverService = multiregion.NewFailoverService(
			s.regionStore, s.natsConn,
			multiregion.FailoverConfig{
				NATSMirrorDomain: cfg.DeploymentRegion + "-dr.example.com",
			},
			logger,
		)
	}

	go s.wsHub.Run()

	// ── Публичные маршруты (без JWT) ─────────────────────────────────
	s.mountHealthRoutes(r)
	s.mountAuthRoutes(r)

	// ═════════════════════════════════════════════════════════════════
	// P3-DX.5: OpenAPI 3.1 + Swagger UI (без JWT)
	//   GET /api/v1/openapi.json — OpenAPI spec (JSON)
	//   GET /api/v1/docs         — Swagger UI (HTML)
	// ═════════════════════════════════════════════════════════════════
	r.Get("/api/v1/openapi.json", s.handleOpenAPIJSON)
	r.Get("/api/v1/docs", s.handleSwaggerUI)

	// Публичный endpoint для подачи заявок (WO-4.1.1)
	// Rate limit: 10 req/min/IP
	r.Group(func(r chi.Router) {
		r.Use(s.workRequestRateLimiter)
		r.Post("/api/v1/public/work-requests", s.submitWorkRequest)
	})

	// External alarm routes (P2P alarm with API key, etc.)
	s.mountExternalAlarmRoutes(r)

	// WebSocket для alarm (JWT в query-параметре, НЕ в Authorization header)
	// Браузерный WebSocket API не поддерживает кастомные заголовки,
	// поэтому маршрут НЕ может быть под AuthMiddleware.
	// handleWebSocket сам валидирует JWT из ?token=...
	r.Get("/api/v1/ws/alarms", s.handleWebSocket)

	// Legacy XML/Vigi alarm endpoints
	if cfg.HTTPXMLEnabled {
		r.Post("/api/v1/external/alarm/xml", s.handleExternalAlarmXML)
	}
	if cfg.VigiEnabled {
		r.Post("/api/v1/external/alarm/vigi", s.handleExternalAlarmVigi)
	}

	// ── Setup Wizard (P0-CE.4: On-Premise, без JWT) ──────────────────
	// Публичные endpoint'ы для первоначальной настройки:
	//   - Статус мастера (GET /api/v1/setup/status)
	//   - Список регионов (GET /api/v1/setup/regions)
	//   - Все шаги мастера (POST /api/v1/setup/*)
	// Доступны только до завершения setup. После — регион locked.
	{
		// Создаём compliance registry с BY, RU, EU, INTL профилями
		registry := compliance.NewProfileRegistry(
			compliance.WithRequiredRegions(compliance.RegionINTL),
			compliance.WithProfile(compliance.NewBYProfile()),
			compliance.WithProfile(compliance.NewRUProfile()),
			compliance.WithProfile(compliance.NewEUProfile()),
			compliance.WithProfile(compliance.NewINTLProfile()),
		)
		wizard := setup.NewSetupWizard(registry,
			setup.WithLogger(s.logger.With("component", "setup.wizard")),
			setup.WithSetupCompleteHandler(func(cfg *setup.SetupConfig) error {
				s.logger.Info("setup wizard completed",
					"region", cfg.Region,
					"admin", cfg.AdminUsername,
				)
				return nil
			}),
		)
		setup.RegisterRoutes(r, wizard)

		// P0-CE.5: Tenant Compliance Profile — registry shared with API
		s.complianceRegistry = registry
	}

	// P0-CE.5: Tenant Compliance Profile
	if database != nil && database.Pool != nil && s.complianceRegistry != nil {
		s.tenantComplianceStore = tenant.NewTenantComplianceStore(database.Pool, s.complianceRegistry)
	}

	// ── Защищённые маршруты (JWT) ────────────────────────────────────
	// P1-SEC.1: CookieAuthMiddleware + AuthMiddleware для поддержки
	// HttpOnly cookies (веб) и Authorization header (API/mobile).
	// CSRFMiddleware для защиты state-changing методов.
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieAuthMiddleware)
		r.Use(auth.AuthMiddleware)
		r.Use(auth.CSRFMiddleware)
		r.Use(auth.TenantMiddleware)

		// P0-CE.5: Tenant Compliance Middleware (injects compliance profile into context)
		if s.tenantComplianceStore != nil {
			r.Use(s.tenantComplianceStore.Middleware)
		}

		// Auth domain (users, sessions, 2FA, Telegram, API keys, settings)
		s.mountProtectedAuthRoutes(r)

		// Device domain (devices, images, analytics, logs, audit)
		s.mountDeviceRoutes(r)

		// Audit domain (P3-2: compliance, chain verification, reporting)
		r.Get("/api/v1/audit/log", s.handleListAuditLog)
		r.Get("/api/v1/audit/verify", s.handleAuditVerify)
		r.Get("/api/v1/audit/compliance", s.handleAuditCompliance)
		r.Post("/api/v1/audit/archive", s.handleAuditArchive)

		// Agent domain (P2P, GB28181, WebSocket, external alarms)
		s.mountAgentRoutes(r)
		r.Post("/api/v1/external/alarm", s.handleExternalAlarm)

		// Integration domain (Atlas CMMS)
		s.mountIntegrationRoutes(r)

		// CMMS domain (maintenance, work orders, spare parts, SLA, mobile)
		s.mountCMMSRoutes(r)

		// Feature Flag domain (F-0.2.4)
		s.mountFeatureFlagRoutes(r)

		// P3-1: Admin routes (multi-region DR)
		s.mountAdminRoutes(r)

		// P0-CE.5: Tenant Compliance Profile (admin routes)
		if s.tenantComplianceStore != nil {
			s.mountTenantComplianceRoutes(r)
		}

		// P0-CE.6: Data Residency Enforcement
		s.mountStorageRoutes(r)

		// Camera Specs Database (P0-9)
		s.mountCameraModelRoutes(r)

		// Workspace Dashboard Multi-Device Sync (P1-1.4)
		r.Get("/api/v1/workspace/layout", s.handleGetLayout)
		r.Post("/api/v1/workspace/layout", s.handleSaveLayout)

		// AI Assistant Chat (P2-1.2)
		s.mountAIRoutes(r)

		// Compliance & Fines Shield (KF-15.1.1)
		s.mountComplianceRoutes(r)

		// Black Box Incident Recorder (KF-15.2.4)
		s.mountBlackBoxRoutes(r)

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
	nonce := NonceFromContext(r.Context())
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
