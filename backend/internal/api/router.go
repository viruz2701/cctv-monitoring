// Package api — единый роутер API с доменной группировкой.
//
// ═══════════════════════════════════════════════════════════════════════════
// P1-ARCH.2: API Routes Organization
//
// Роуты сгруппированы по доменам с использованием chi Router Groups.
// Каждый домен имеет свой файл *_routes.go с методами монтирования.
//
// Домены:
//   - Health    → mountHealthRoutes()     (health_handlers.go)
//   - Auth      → mountAuthRoutes()       (auth_routes.go)
//   - Protected → mountProtectedAuthRoutes() (auth_routes.go)
//   - Devices   → mountDeviceRoutes()     (device_routes.go)
//   - CMMS      → mountCMMSRoutes()       (cmms_routes.go)
//   - Agent     → mountAgentRoutes()      (agent_routes.go)
//   - Integration → mountIntegrationRoutes() (integration_routes.go)
//   - Compliance → mountComplianceRoutes()  (compliance_routes.go)
//   - BlackBox  → mountBlackBoxRoutes()   (blackbox_routes.go)
//   - Storage   → mountStorageRoutes()    (storage_routes.go)
//   - Feature Flags → mountFeatureFlagRoutes() (featureflag_routes.go)
//   - Camera Models → mountCameraModelRoutes() (camera_routes.go)
//   - AI        → mountAIRoutes()         (ai_routes.go)
//   - Admin     → mountAdminRoutes()      (admin_handlers.go)
//   - Tenant Compliance → mountTenantComplianceRoutes() (tenant_compliance_routes.go)
//
// Соответствует:
//   - OWASP ASVS L3 V1-V17 (полный спектр контролей)
//   - ISO 27001 A.9.2 (RBAC), A.12.4 (Audit)
//   - IEC 62443-3-3 SR 1.1 (Defense in depth)
//
// ═══════════════════════════════════════════════════════════════════════════
package api

import (
	"context"
	"time"

	"github.com/go-chi/chi/v5"

	"gb-telemetry-collector/internal/ai"
	apimw "gb-telemetry-collector/internal/api/middleware"
	"gb-telemetry-collector/internal/auth"
	"gb-telemetry-collector/internal/blackbox"
	"gb-telemetry-collector/internal/cmms"
	"gb-telemetry-collector/internal/compliance"
	"gb-telemetry-collector/internal/multiregion"
	"gb-telemetry-collector/internal/service"
	"gb-telemetry-collector/internal/setup"
	"gb-telemetry-collector/internal/tenant"
	"gb-telemetry-collector/internal/webhook"
)

// MountRoutes монтирует все API маршруты на предоставленный chi роутер.
//
// Разделение на:
//   - Публичные (без JWT): health, auth/login, refresh, setup wizard, OpenAPI, SBOM
//   - Защищённые (JWT): все остальные
//   - API key: внешние alarm webhook
//   - ITSM Webhooks: ServiceNow, Jira, 1C:TOIR (HMAC, rate-limited)
func (s *Server) MountRoutes(r chi.Router) {
	// ── Публичные маршруты (без JWT) ─────────────────────────────────
	s.mountHealthRoutes(r)
	s.mountAuthRoutes(r)

	// ═════════════════════════════════════════════════════════════════
	// P0-N1: SBOM (Software Bill of Materials) — public endpoints
	//   GET /api/v1/sbom          — список доступных форматов SBOM
	//   GET /api/v1/sbom/{format} — SBOM в указанном формате (JSON)
	//   GET /api/v1/sbom/{format}/raw — "сырой" SBOM (для инструментов)
	// ═════════════════════════════════════════════════════════════════
	s.mountSBOMRoutes(r)

	// ═════════════════════════════════════════════════════════════════
	// P0-N2: Well-Known URIs (RFC 8615, RFC 9116) — public endpoints
	//   GET /.well-known/security.txt     — Vulnerability Disclosure Policy
	//   GET /.well-known/security-policy  — HTML version of SECURITY.md
	// ═════════════════════════════════════════════════════════════════
	s.mountWellKnownRoutes(r)

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
	if s.config.HTTPXMLEnabled {
		r.Post("/api/v1/external/alarm/xml", s.handleExternalAlarmXML)
	}
	if s.config.VigiEnabled {
		r.Post("/api/v1/external/alarm/vigi", s.handleExternalAlarmVigi)
	}

	// ── Setup Wizard (P0-CE.4: On-Premise, без JWT) ──────────────────
	s.mountSetupWizardRoutes(r)

	// ── Защищённые маршруты (JWT) ────────────────────────────────────
	// P1-SEC.1: CookieAuthMiddleware + AuthMiddleware для поддержки
	// HttpOnly cookies (веб) и Authorization header (API/mobile).
	// CSRFMiddleware для защиты state-changing методов.
	r.Group(func(r chi.Router) {
		r.Use(auth.CookieAuthMiddleware)
		r.Use(auth.AuthMiddleware)
		r.Use(auth.CSRFMiddleware)
		r.Use(auth.TenantMiddleware)

		// P1-RATE: Distributed Rate Limiting (Redis-based)
		// Применяется ко всем защищённым маршрутам после аутентификации.
		// Лимиты: 100 read/30 write per minute per tenant/user.
		// Соответствует: OWASP ASVS V2.2.1, ISO 27001 A.12.1.2
		if s.rateLimitRedis != nil {
			rateLimiter := apimw.NewRateLimiter(s.rateLimitRedis, 100, 30, time.Minute)
			r.Use(rateLimiter.Middleware)
		}

		// P0-CE.5: Tenant Compliance Middleware (injects compliance profile into context)
		if s.tenantComplianceStore != nil {
			r.Use(s.tenantComplianceStore.Middleware)
		}

		// P1-QUOTA: Tenant Quota Middleware
		// Проверяет квоты на мутирующих запросах (POST, PUT, DELETE, PATCH).
		// Soft limit (80%) → X-Quota-Warning header
		// Hard limit (100%) → HTTP 429 (если не на grace period)
		if s.tenantQuotaManager != nil {
			r.Use(apimw.QuotaMiddleware(s.tenantQuotaManager, ""))
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

		// Integration domain (Atlas CMMS, webhooks)
		s.mountIntegrationRoutes(r)

		// CMMS domain (maintenance, work orders, spare parts, SLA, mobile)
		s.mountCMMSRoutes(r)

		// Feature Flag domain (F-0.2.4)
		s.mountFeatureFlagRoutes(r)

		// P3-1: Admin routes (multi-region DR, users, settings, audit)
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

		// P1-REPLAY: NATS JetStream Event Replay UI
		if s.eventReplay != nil {
			s.mountEventReplayRoutes(r)
		}

		// GraphQL read-only endpoint (INT-13.2.4)
		s.mountGraphQLRoute(r)

		// P1-QUOTA: Tenant Quota Management
		//   GET    /api/v1/tenant/quota           — текущее использование
		//   GET    /api/v1/tenant/quota/history   — история изменений
		//   PUT    /api/v1/tenant/quota           — обновить лимиты (admin)
		//   POST   /api/v1/tenant/quota/increase  — запрос на увеличение
		if s.tenantQuotaManager != nil {
			s.mountTenantQuotaRoutes(r)
		}
	})

	// ── External API key auth ────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(s.APIKeyMiddleware)
		r.Post("/api/v1/external/alarm", s.handleExternalAlarm)
	})

	// ── ITSM Webhooks (HMAC, rate-limited) ───────────────────────────
	s.mountWebhookRoutes(r)
}

// mountSetupWizardRoutes монтирует маршруты мастера установки.
func (s *Server) mountSetupWizardRoutes(r chi.Router) {
	// Публичные endpoint'ы для первоначальной настройки:
	//   - Статус мастера (GET /api/v1/setup/status)
	//   - Список регионов (GET /api/v1/setup/regions)
	//   - Все шаги мастера (POST /api/v1/setup/*)
	// Доступны только до завершения setup. После — регион locked.

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

	// Tenant compliance store initialization
	if s.db != nil && s.db.Pool != nil && s.complianceRegistry != nil {
		s.tenantComplianceStore = tenant.NewTenantComplianceStore(s.db.Pool, s.complianceRegistry)
	}
}

// initServices инициализирует сервисы, требующие время на старте.
// Вызывается из NewServer после создания Server.
func (s *Server) initServices() {
	// ── P2-AI.4: Anomaly Detection Service ─────────────────────────
	s.initAnomalyService()

	// ── Device Service ────────────────────────────────────────────────
	s.deviceService = service.NewDeviceService(s.db, s.auditSigner, s.logger)

	// ── Compliance Engine (KF-15.1.1) ─────────────────────────────────
	s.complianceEngine = compliance.NewEngine(nil, s.logger, nil)

	// ── P2-RU.2: 152-ФЗ Personal Data Manager ─────────────────────────
	pdStore := compliance.NewMemoryPersonalDataStore(s.logger)
	s.personalDataManager = compliance.NewPersonalDataManager(pdStore, s.logger)

	// ── P2-EU.1: GDPR Manager ─────────────────────────────────────────
	gdprStore := compliance.NewMemoryGDPRStore(s.logger)
	s.gdprManager = compliance.NewGDPRManager(gdprStore, s.logger)

	// ── Black Box Incident Recorder (KF-15.2.4) ───────────────────────
	bbRepo := blackbox.NewDBRepository(s.db.Pool, s.logger)
	s.blackboxRecorder = blackbox.NewRecorder(bbRepo, s.db, nil, s.logger)

	// ── Auto-dispatcher Service (P1-6) ────────────────────────────────
	s.initAutoDispatcher()

	// ── Dispatch Rules Engine (P1-6) ──────────────────────────────────
	s.ruleEngine = cmms.NewRuleEngine(s.logger)

	// ── P2-3.3: Webhook Delivery Worker ─────────────────────────────
	s.initWebhookWorker()

	// ── P3-1: Multi-Region Geo-Redundancy ──────────────────────────
	s.initMultiRegion()
}

// initAnomalyService инициализирует сервис обнаружения аномалий.
func (s *Server) initAnomalyService() {
	anomalyCfg := ai.DefaultAnomalyConfig()
	var anomalyBroadcaster ai.Broadcaster
	if s.wsHub != nil {
		anomalyBroadcaster = s.wsHub
	}
	anomalyService, err := ai.NewAnomalyService(anomalyCfg, s.natsConn, anomalyBroadcaster, s.logger)
	if err != nil {
		s.logger.Warn("anomaly service init warning", "error", err)
	} else {
		s.anomalyService = anomalyService
		s.logger.Info("anomaly detection service initialized",
			"z_score_threshold", anomalyCfg.ZScoreThreshold,
			"ma_window", anomalyCfg.MovingAverageWindow,
		)
	}
}

// initAutoDispatcher инициализирует автоматический диспетчер.
func (s *Server) initAutoDispatcher() {
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
		s.logger,
	)
}

// initWebhookWorker инициализирует webhook delivery worker.
func (s *Server) initWebhookWorker() {
	if s.db != nil && s.db.Pool != nil {
		s.webhookStore = webhook.NewPGDeliveryStore(s.db.Pool)
		s.deliveryWorker = webhook.NewDeliveryWorker(
			s.webhookStore, s.logger,
			webhook.DeliveryWorkerConfig{
				PollInterval:  5 * time.Second,
				MaxConcurrent: 5,
			},
		)
		go s.deliveryWorker.Start(context.Background())
	}
}

// initMultiRegion инициализирует multi-region geo-redundancy.
func (s *Server) initMultiRegion() {
	if s.db != nil && s.db.Pool != nil {
		s.regionStore = multiregion.NewPGTenantRegionStore(s.db.Pool)
		s.failoverService = multiregion.NewFailoverService(
			s.regionStore, s.natsConn,
			multiregion.FailoverConfig{
				NATSMirrorDomain: s.config.DeploymentRegion + "-dr.example.com",
			},
			s.logger,
		)
	}
}
