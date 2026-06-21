
# TODO.md — Task Tracker for CCTV Intelligence Platform v5.0

**Обновлено:** 2026-06-21
**Текущая фаза:** 2 (AI Intelligence & Atlas Integration)

---

## 📊 Progress Overview

| Epic | Phase | Priority | Status | Progress |
|------|-------|----------|--------|----------|
| Foundation & Analysis | Phase 0 | P0 | ✅ Done | 100% |
| Headless CMMS & Adapter | Phase 1 | P0 | ✅ Done | 100% |
| Gatekeeper Service | Phase 1.5 | P0 | ✅ Done | 100% |
| ISO 27001 Quick Wins | Phase 1.5 | P0 | ✅ Done | 100% |
| UX Refresh (Desktop) | Phase 1.5 | P1 | ✅ Done | 100% |
| Atlas CMMS Integration | Phase 2 | P1 | ✅ Done | 100% |
| AI Intelligence & TCO | Phase 2 | P1 | 🔴 Not Started | 0% |
| Universal CMMS Gateway | Phase 3 | P1 | 🔴 Not Started | 0% |
| Enterprise Scale & ISO Cert | Phase 4 | P2 | 🔴 Not Started | 0% |

---

## ✅ PHASE 0: Foundation & Analysis (DONE)

### Epic 0.1: UX Research ✅
- [x] **0.1.1** Анализ существующего UI → `docs/ux/current-state-audit.md`
- [x] **0.1.2** Адаптация Shelf.nu паттернов → `docs/ux/shelf-nu-patterns.md`
- [x] **0.1.3** Адаптация Snipe-IT паттернов → `docs/ux/snipe-it-patterns.md`
- [x] **0.1.4** Mobile UX гайдлайн → `docs/ux/mobile-current-state.md`

### Epic 0.2: ISO 27001 Gap Analysis ✅
- [x] **0.2.1** Аудит API keys → `docs/iso27001/gap-analysis.md` Gap #4
- [x] **0.2.2** Аудит Push Tokens → Gap #8
- [x] **0.2.3** Аудит Telegram Bot → Gap #14
- [x] **0.2.4** Compliance Matrix → `docs/iso27001/compliance-matrix.md`
- [x] **0.2.5** Remediation Plan → `docs/iso27001/remediation-plan.md`

### Epic 0.3: Architecture Documentation ✅
- [x] **0.3.1** ADR-001: Headless CMMS → `docs/adr/ADR-001-headless-cmms.md`
- [x] **0.3.2** ADR-002: CMMS Adapter Pattern → `docs/adr/ADR-002-cmms-adapter-pattern.md`
- [x] **0.3.3** ADR-003: Event Bus (NATS) → `docs/adr/ADR-003-event-bus.md`
- [x] **0.3.4** ADR-004: Gatekeeper Pattern → `docs/adr/ADR-004-gatekeeper-pattern.md`

---

## ✅ PHASE 1: Headless CMMS (DONE)

### Epic 1.1: CMMS Adapter Framework ✅
- [x] **1.1.1** `CMMSAdapter` interface (33 methods) → `backend/internal/cmms/adapter.go`
- [x] **1.1.2** `InternalAdapter` → `backend/internal/cmms/internal_adapter.go`
- [x] **1.1.3** `AtlasAdapter` (stub) → `backend/internal/cmms/atlas_adapter.go`
- [x] **1.1.4** `CMMSRouter` delegate → `backend/internal/cmms/adapter.go`
- [x] **1.1.5** Integration in handlers → `cmms_handlers.go`, `mobile_handlers.go`

### Epic 1.2: Gatekeeper Service ✅ (moved from Phase 1 to 1.5)
- [x] **1.2.1** Go Backend: `/api/v1/mobile/work-orders/{id}/verify`
- [x] **1.2.2** Go Backend: GPS geofence validation (Haversine distance)
- [x] **1.2.3** Go Backend: EXIF validation (time, device, gallery block)
- [x] **1.2.4** Go Backend: AI before/after (DeepSeek Vision, graceful skip)
- [x] **1.2.5** Mobile: VerificationScreen (post-PhotoCapture, pre-Signature)
- [x] **1.2.6** Mobile: Live camera only (no gallery for Gatekeeper)

### Epic 1.3: Maintenance Schedules & Cron ✅
- [x] **1.3.1** Maintenance schedules CRUD → `cmms_handlers.go`
- [x] **1.3.2** Maintenance cron (15min) → `cron/maintenance_cron.go`
- [x] **1.3.3** Auto-create work orders from due schedules
- [x] **1.3.4** SLA deadline calculation

### Epic 1.4: Technician Site Assignments ✅
- [x] **1.4.1** Migration `002_technician_site_assignments.sql`
- [x] **1.4.2** CRUD endpoints → `cmms_handlers.go`
- [x] **1.4.3** Frontend integration → `Sites.tsx`

---

## ✅ PHASE 1.5: Gatekeeper + ISO + UX Refresh (DONE)

### Epic 1.5.1: Gatekeeper Service [P0] ✅
- [x] **1.5.1.1** Создан пакет `backend/internal/gatekeeper/`
- [x] **1.5.1.2** Endpoint `POST /api/v1/mobile/work-orders/{id}/verify`
- [x] **1.5.1.3** GPS: Haversine distance to site geofence
- [x] **1.5.1.4** EXIF: timestamp validation (started_at < photo_time < now)
- [x] **1.5.1.5** EXIF: device model matching
- [x] **1.5.1.6** AI: DeepSeek Vision integration (before/after comparison, graceful skip)
- [x] **1.5.1.7** Verification token (JWT, TTL 10min)
- [x] **1.5.1.8** Audit log: `gatekeeper_verify` action
- [x] **1.5.1.9** Mobile: VerificationScreen (между PhotoCapture и Signature)
- [x] **1.5.1.10** Mobile: Block gallery upload for Gatekeeper photos
- [x] **1.5.1.11** Update `CompleteWorkOrder` to require verification_token
- [x] **1.5.1.12** Frontend: Verification status in WorkOrderDetail

### Epic 1.5.2: ISO 27001 Quick Wins [P0] ✅
- [x] **QW-1** JWT Secret: `os.Getenv("JWT_SECRET")` с panic → `auth/jwt.go`
- [x] **QW-2** API Keys: SHA-256 → bcrypt(cost=12) с prefix lookup → `apikey_handlers.go`
- [x] **QW-3** Push Tokens: AES-256-GCM encryption → `crypto/aes.go`, `cmms_repository.go`
- [x] **QW-4** Config secrets → env vars → `.env.example`
- [x] **QW-5** Rate limiting на login (5 req/min) → `server.go`
- [x] **QW-6** Security headers (CSP, X-Frame-Options, HSTS) → `server.go`

### Epic 1.5.3: UX Refresh (Desktop) [P1] ✅
- [x] **1.5.3.1** `DataGrid.tsx` (Snipe-IT style) → `components/ui/DataGrid.tsx`
- [x] **1.5.3.2** `PartCard.tsx` (Shelf.nu style) → `components/ui/PartCard.tsx`
- [x] **1.5.3.3** `Gauge.tsx` (circular metric) → `components/ui/Gauge.tsx`
- [x] **1.5.3.4** `SLAProgress.tsx` (progress bar with timer) → `components/ui/SLAProgress.tsx`
- [x] **1.5.3.5** `Timeline.tsx` (audit log) → `components/ui/Timeline.tsx`
- [x] **1.5.3.6** `Tabs.tsx` (tabbed interface) → `components/ui/Tabs.tsx`
- [x] **1.5.3.7** `QRCode.tsx` (QR code generator) → `components/ui/QRCode.tsx`
- [x] **1.5.3.8** `FileUpload.tsx` (drag-and-drop) → `components/ui/FileUpload.tsx`
- [x] **1.5.3.9** `WorkOrderDetail.tsx` (three-column layout) → `pages/WorkOrderDetail.tsx`
- [x] **1.5.3.10** `WorkOrders.tsx` → использует DataGrid
- [x] **1.5.3.11** `SpareParts.tsx` → использует PartCard (card grid)
- [x] **1.5.3.12** `SLADashboard.tsx` → использует Gauge
- [x] **1.5.3.13** `Settings.tsx` → использует Tabs

---

## 🔄 PHASE 2: AI Intelligence & Atlas Integration (CURRENT)

**Срок:** Месяцы 4-6
**Цель:** Интеграция с Atlas CMMS, предиктивная аналитика, TCO калькулятор, голосовые отчёты

### Epic 2.1: Atlas CMMS Integration [P1] ✅ Done
- [x] **2.1.1** AtlasAdapter: REST API client (OAuth2) → `backend/internal/cmms/atlas_client.go`
- [x] **2.1.2** AtlasAdapter: CreateWorkOrder, UpdateWorkOrder, все 33 метода → `backend/internal/cmms/atlas_adapter.go`
- [x] **2.1.3** AtlasAdapter: WorkOrder status sync (assign/start/complete/cancel)
- [x] **2.1.4** AtlasAdapter: SyncAsset (device → CMMS asset) → `POST /api/v1/atlas/sync-asset/{deviceId}`
- [x] **2.1.5** Fallback queue: персистентная очередь на ФС + retry → `backend/internal/cmms/fallback_queue.go`
- [x] **2.1.6** Settings → Integrations: AtlasCMSPanel с health-check, fallback queue, конфигурацией → `frontend/src/pages/Settings.tsx`
- [x] **2.1.7** Health check endpoint → `GET /api/v1/atlas/health`
- [x] **2.1.8** Config: OAuth2 client credentials (client_id, client_secret, token_url) → `backend/internal/config/config.go`
- [x] **2.1.9** Frontend API: atlasHealthCheck, atlasFallbackStatus, atlasRetryFallback, atlasSyncAsset → `frontend/src/services/api.ts`

### Epic 2.2: Predictive Maintenance [P1]
- [ ] **2.2.1** Расширить `predict.py`: HDD, PoE, Temperature features
- [ ] **2.2.2** Go Backend: `/api/v1/predictions` endpoint
- [ ] **2.2.3** CMMS Router: авто-создание PM-задач из predictions
- [ ] **2.2.4** Mobile: push-уведомления о предстоящих PM
- [ ] **2.2.5** Frontend: Predictions page с explanations

### Epic 2.3: TCO Calculator [P1]
- [ ] **2.3.1** Go Backend: агрегация из CMMS (запчасти, часы, cost)
- [ ] **2.3.2** Go Backend: расчёт TCO per device/site
- [ ] **2.3.3** Frontend: `TCO.tsx` страница (графики, рекомендации)
- [ ] **2.3.4** Replace vs Repair рекомендации

### Epic 2.4: Voice-to-Report [P1]
- [ ] **2.4.1** Mobile: Whisper API integration (или on-device STT)
- [ ] **2.4.2** Go Backend: NLP обработка (DeepSeek) → entity extraction
- [ ] **2.4.3** Авто-обновление CMDB из голосовых отчётов
- [ ] **2.4.4** Voice recording UI в Mobile

---

## 📍 PHASE 3: Universal CMMS Gateway (Месяцы 7-9)

### Epic 3.1: Enterprise Adapters [P1]
- [ ] **3.1.1** `ServiceNowAdapter` (SOAP/REST, CMDB CI, Incident/Problem)
- [ ] **3.1.2** `JiraAdapter` (REST v3, Jira Service Management)
- [ ] **3.1.3** `ToirAdapter` (1С:ТОИР REST API, 152-ФЗ)
- [ ] **3.1.4** UI маппинга полей (drag-and-drop в Integrations)

### Epic 3.2: Agentic Self-Healing [P1]
- [ ] **3.2.1** AI Agent: диагностика (анализ топологии)
- [ ] **3.2.2** AI Agent: remediation (ISAPI/ONVIF через P2P Gateway)
- [ ] **3.2.3** CMMS Router: авто-закрытие тикетов после self-healing
- [ ] **3.2.4** Human-in-the-loop approval для критичных действий

### Epic 3.3: NATS Event Bus [P1]
- [ ] **3.3.1** Добавить `nats.go` в backend
- [ ] **3.3.2** Publisher: `alarms.{device_id}`, `cmms.workorder.*`
- [ ] **3.3.3** Subscriber: WebSocket Hub, Mobile push, Worker
- [ ] **3.3.4** JetStream для persistence (optional)

### Epic 3.4: Bi-directional ITSM Sync [P1]
- [ ] **3.4.1** Webhooks от ServiceNow/Jira → Go Backend
- [ ] **3.4.2** State Machine: синхронизация статусов (каждые 5 мин)
- [ ] **3.4.3** Conflict Resolution: авто-переоткрытие тикетов

---

## 📍 PHASE 4: Enterprise Scale & ISO Cert (Месяцы 10-15)

### Epic 4.1: Multi-tenant SaaS [P2]
- [ ] **4.1.1** Row-level security (PostgreSQL RLS)
- [ ] **4.1.2** Billing tiers (Community/Pro/Enterprise)
- [ ] **4.1.3** Stripe integration
- [ ] **4.1.4** Tenant isolation в WebSocket Hub

### Epic 4.2: AR Remote Expert [P2]
- [ ] **4.2.1** WebRTC (pion/webrtc) в Go Backend
- [ ] **4.2.2** Mobile: AR-маркеры (React Native ARKit/ARCore)
- [ ] **4.2.3** Интеграция с CMMS (запись сессии → наряд)

### Epic 4.3: Security Convergence [P2]
- [ ] **4.3.1** CrowdStrike/SentinelOne API integration
- [ ] **4.3.2** Корреляция Physical + Cyber events
- [ ] **4.3.3** Unified Dashboard для CISO/CIO

### Epic 4.4: ISO 27001 Certification [P2]
- [ ] **4.4.1** Internal audit
- [ ] **4.4.2** Stage 1 + Stage 2 audit
- [ ] **4.4.3** Получение сертификата
- [ ] **4.4.4** Continuous monitoring

---

## 📊 Priority Definitions

| Priority | Описание | SLA |
|----------|----------|-----|
| **P0** | Critical — блокирует релиз, security vulnerability | Fix within 24h |
| **P1** | High — важно для конкурентоспособности | Fix within 1 week |
| **P2** | Medium — nice-to-have, для лидерства на рынке | Fix within 1 month |

---

## 🎯 Success Criteria (Phase 2)

- [ ] Atlas CMMS integration работает (создание/обновление нарядов)
- [ ] Predictive Maintenance: авто-создание PM-задач из XGBoost predictions
- [ ] TCO Calculator: расчёт стоимости владения per device/site
- [ ] Voice-to-Report: голосовые отчёты через Whisper API
- [ ] Frontend: Predictions page с explanations от DeepSeek

---

## 📈 Metrics

| Metric | Current | Target (Phase 2) | Target (Phase 4) |
|--------|---------|--------------------|--------------------|
| Code Coverage (Go) | ~15% | 50% | 80% |
| Code Coverage (React) | ~5% | 30% | 60% |
| API Response Time (p95) | ~200ms | <120ms | <100ms |
| ISO 27001 Gaps Open | 6 | 3 | 0 |
| Lighthouse Score | ~75 | 92+ | 95+ |
| Mobile Crash Rate | N/A | <0.1% | <0.05% |
| Atlas Integration Uptime | N/A | 99% | 99.9% |