
---

## 3. `TODO.md` (~350 строк)

```markdown
# TODO.md — Task Tracker for CCTV Intelligence Platform v4.0

**Обновлено:** 2026-06-21
**Текущая фаза:** 1.5 (Gatekeeper + ISO Quick Wins + UX Refresh)

---

## 📊 Progress Overview

| Epic | Phase | Priority | Status | Progress |
|------|-------|----------|--------|----------|
| Foundation & Analysis | Phase 0 | P0 | ✅ Done | 100% |
| Headless CMMS & Adapter | Phase 1 | P0 | ✅ Done | 100% |
| Gatekeeper Service | Phase 1.5 | P0 | 🔴 Not Started | 0% |
| ISO 27001 Quick Wins | Phase 1.5 | P0 | 🔴 Not Started | 0% |
| UX Refresh (Desktop) | Phase 1.5 | P1 | 🔴 Not Started | 0% |
| Atlas CMMS Integration | Phase 2 | P1 | ⚠️ Stub | 10% |
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

## ✅ PHASE 1: Headless CMMS (DONE — 75%)

### Epic 1.1: CMMS Adapter Framework ✅
- [x] **1.1.1** `CMMSAdapter` interface (33 methods) → `backend/internal/cmms/adapter.go`
- [x] **1.1.2** `InternalAdapter` → `backend/internal/cmms/internal_adapter.go`
- [x] **1.1.3** `AtlasAdapter` (stub) → `backend/internal/cmms/atlas_adapter.go`
- [x] **1.1.4** `CMMSRouter` delegate → `backend/internal/cmms/adapter.go`
- [x] **1.1.5** Integration in handlers → `cmms_handlers.go`, `mobile_handlers.go`

### Epic 1.2: Gatekeeper Service ❌ → MOVED TO PHASE 1.5
- [ ] **1.2.1** Go Backend: `/api/v1/mobile/work-orders/{id}/verify`
- [ ] **1.2.2** Go Backend: GPS geofence validation (sites.geofence_polygon)
- [ ] **1.2.3** Go Backend: EXIF validation (time, device, gallery block)
- [ ] **1.2.4** Go Backend: AI before/after (DeepSeek Vision)
- [ ] **1.2.5** Mobile: Verification screen (post-PhotoCapture, pre-Signature)
- [ ] **1.2.6** Mobile: Live camera only (no gallery for Gatekeeper)

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

## 🆕 PHASE 1.5: Gatekeeper + ISO + UX Refresh (CURRENT)

**Срок:** Месяц 3 (4 недели)
**Цель:** Завершить Phase 1, устранить критические security gaps, обновить UI

### Epic 1.5.1: Gatekeeper Service [P0] 🔴
- [ ] **1.5.1.1** Создать `backend/internal/gatekeeper/` пакет
- [ ] **1.5.1.2** Endpoint `POST /api/v1/mobile/work-orders/{id}/verify`
- [ ] **1.5.1.3** GPS: Haversine distance to site geofence
- [ ] **1.5.1.4** EXIF: timestamp validation (started_at < photo_time < now)
- [ ] **1.5.1.5** EXIF: device model matching
- [ ] **1.5.1.6** AI: DeepSeek Vision integration (before/after comparison)
- [ ] **1.5.1.7** Verification token (JWT, TTL 10min)
- [ ] **1.5.1.8** Audit log: `gatekeeper_verify` action
- [ ] **1.5.1.9** Mobile: VerificationScreen (между PhotoCapture и Signature)
- [ ] **1.5.1.10** Mobile: Block gallery upload for Gatekeeper photos
- [ ] **1.5.1.11** Update `CompleteWorkOrder` to require verification_token
- [ ] **1.5.1.12** Frontend: Verification status in WorkOrderDetail

**Файлы для создания:**
- `backend/internal/gatekeeper/verifier.go`
- `backend/internal/gatekeeper/gps.go`
- `backend/internal/gatekeeper/exif.go`
- `backend/internal/gatekeeper/ai.go`
- `mobile/src/screens/VerificationScreen.tsx`
- `mobile/src/hooks/useGatekeeper.ts`

### Epic 1.5.2: ISO 27001 Quick Wins [P0] 🔴

**QW-1: JWT Secret (10 min)**
- [ ] **1.5.2.1** `auth/jwt.go`: `getJWTSecret()` → `os.Getenv("JWT_SECRET")` с panic
- [ ] **1.5.2.2** Docker: добавить `JWT_SECRET` в docker-compose
- [ ] **1.5.2.3** Docs: обновить README с env vars

**QW-2: API Keys bcrypt (1 hour)**
- [ ] **1.5.2.4** `apikey_handlers.go`: SHA-256 → bcrypt(cost=12)
- [ ] **1.5.2.5** `apikey_middleware.go`: bcrypt.CompareHashAndPassword
- [ ] **1.5.2.6** Migration: пересоздать существующие API keys (notify users)

**QW-3: Push Token Encryption (2 hours)**
- [ ] **1.5.2.7** `crypto/aes` GCM wrapper в `internal/crypto/`
- [ ] **1.5.2.8** `cmms_repository.go`: encrypt before save, decrypt on read
- [ ] **1.5.2.9** Migration script для existing tokens

**QW-4: Config Secrets → env vars (1 hour)**
- [ ] **1.5.2.10** `config.yaml`: убрать `p2p_api_key`, FTP password, Hikvision passwords
- [ ] **1.5.2.11** `config/config.go`: читать из `os.Getenv()`
- [ ] **1.5.2.12** `.env.example` файл

**QW-5: Rate Limiting (1 hour)**
- [ ] **1.5.2.13** Добавить `github.com/go-chi/httprate` в go.mod
- [ ] **1.5.2.14** Middleware на `/api/v1/auth/login` (5 req/min)
- [ ] **1.5.2.15** Middleware на `/api/v1/mobile/*` (60 req/min)

**QW-6: Security Headers (30 min)**
- [ ] **1.5.2.16** Middleware: CSP, X-Frame-Options, X-Content-Type-Options
- [ ] **1.5.2.17** CORS: whitelist из конфига (не "*")

### Epic 1.5.3: UX Refresh (Desktop) [P1] 🔴

**1.5.3.1 WorkOrders → Snipe-IT DataGrid**
- [ ] **1.5.3.1.1** Создать `components/ui/DataGrid.tsx` (фильтры в заголовках, bulk actions)
- [ ] **1.5.3.1.2** Bulk Actions Toolbar: assign, change status, cancel, change priority
- [ ] **1.5.3.1.3** Quick Filters: My Orders, Overdue, Unassigned, Critical, Today
- [ ] **1.5.3.1.4** Inline status change (dropdown в ячейке)
- [ ] **1.5.3.1.5** Кастомизируемые колонки (visibility toggle)
- [ ] **1.5.3.1.6** Обновить `WorkOrders.tsx` на DataGrid

**1.5.3.2 SpareParts → Shelf.nu Card Grid**
- [ ] **1.5.3.2.1** Создать `components/ui/PartCard.tsx` (изображение, SKU, stock, QR)
- [ ] **1.5.3.2.2** Stock indicator: красный если `stock <= min_stock`
- [ ] **1.5.3.2.3** QR code generation (`qrcode.react`)
- [ ] **1.5.3.2.4** Grid layout (responsive: 1/2/3/4 columns)
- [ ] **1.5.3.2.5** Обновить `SpareParts.tsx` на Card Grid + toggle Table/Cards

**1.5.3.3 SLADashboard → Atlas CMMS Visuals**
- [ ] **1.5.3.3.1** Создать `components/ui/Gauge.tsx` (круговая метрика)
- [ ] **1.5.3.3.2** Создать `components/ui/SLAProgress.tsx` (progress bar с таймером)
- [ ] **1.5.3.3.3** Создать `components/ui/Timeline.tsx` (SLA breach timeline)
- [ ] **1.5.3.3.4** Обновить `SLADashboard.tsx`: gauge per priority, compliance chart

**1.5.3.4 WorkOrderDetail (новая страница)**
- [ ] **1.5.3.4.1** Создать `pages/WorkOrderDetail.tsx`
- [ ] **1.5.3.4.2** Three-column layout: Status/SLA | Checklist/Photos | Device/Parts
- [ ] **1.5.3.4.3** SLA countdown timer
- [ ] **1.5.3.4.4** Audit timeline (из audit_log)
- [ ] **1.5.3.4.5** Before/After photo comparison
- [ ] **1.5.3.4.6** Route: `/work-orders/:id`

**1.5.3.5 Settings → Tabs**
- [ ] **1.5.3.5.1** Создать `components/ui/Tabs.tsx`
- [ ] **1.5.3.5.2** Разделить `Settings.tsx` на вкладки:
  - General, Services, Integrations, Security, Notifications, Logging
- [ ] **1.5.3.5.3** Integrations tab: CMMS adapter selector (Internal/Atlas)

**1.5.3.6 Новые UI компоненты**
- [ ] **1.5.3.6.1** `components/ui/Tabs.tsx`
- [ ] **1.5.3.6.2** `components/ui/DataGrid.tsx`
- [ ] **1.5.3.6.3** `components/ui/Gauge.tsx`
- [ ] **1.5.3.6.4** `components/ui/SLAProgress.tsx`
- [ ] **1.5.3.6.5** `components/ui/Timeline.tsx`
- [ ] **1.5.3.6.6** `components/ui/QRCode.tsx`
- [ ] **1.5.3.6.7** `components/ui/FileUpload.tsx` (drag-and-drop)

---

## 📍 PHASE 2: AI Intelligence & Atlas Integration (Месяцы 4-6)

### Epic 2.1: Atlas CMMS Integration [P1]
- [ ] **2.1.1** AtlasAdapter: REST API client (OAuth2)
- [ ] **2.1.2** AtlasAdapter: CreateWorkOrder (mapping fields)
- [ ] **2.1.3** AtlasAdapter: UpdateWorkOrder (status sync)
- [ ] **2.1.4** AtlasAdapter: SyncAsset (device → CMMS asset)
- [ ] **2.1.5** Fallback queue: если Atlas недоступен → Internal DB + retry
- [ ] **2.1.6** Settings → Integrations: Atlas URL, API Key, field mapping
- [ ] **2.1.7** Health check endpoint для Atlas

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
- [ ] **4.1.3** 
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

## 🎯 Success Criteria (Phase 1.5)

- [ ] Gatekeeper блокирует закрытие наряда без GPS/EXIF/AI верификации
- [ ] JWT secret — только из env var (panic если нет)
- [ ] API keys хешируются bcrypt (не SHA-256)
- [ ] Push tokens зашифрованы AES-256-GCM
- [ ] Rate limiting на login и mobile endpoints
- [ ] WorkOrders: DataGrid с bulk actions и quick filters
- [ ] SpareParts: Card Grid с stock indicators
- [ ] SLADashboard: Gauge charts и progress bars
- [ ] WorkOrderDetail: Three-column layout с SLA timer

---

## 📈 Metrics

| Metric | Current | Target (Phase 1.5) | Target (Phase 4) |
|--------|---------|--------------------|--------------------|
| Code Coverage (Go) | ~15% | 40% | 80% |
| Code Coverage (React) | ~5% | 20% | 60% |
| API Response Time (p95) | ~200ms | <150ms | <100ms |
| ISO 27001 Gaps Open | 17 | 6 | 0 |
| Lighthouse Score | ~75 | 90+ | 95+ |
| Mobile Crash Rate | N/A | <0.1% | <0.05% |