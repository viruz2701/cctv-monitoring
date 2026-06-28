Правила для Roo при работе с TODO
Перед началом задачи: Прочитать соответствующий раздел, проверить dependencies
Во время работы: Коммитить атомарно, в сообщении указывать ID задачи (например, P0-REG.1: Maintenance Regulations Data Model)
После завершения: Отметить [x] + дата, проверить критерий приёмки, обновить метрику
Если задача слишком большая: Разбить на подзадачи с суффиксами (.1, .2, ...)
Никогда не пропускать: Критерий приёмки — если он не выполнен, задача не завершена
Code review чеклист для каждой задачи:
Dark mode работает
Accessibility (WCAG 2.1 AA)
i18n ключи добавлены в locales/
Error handling реализован
Unit/integration тесты написаны
Документация обновлена
No console errors/warnings
Responsive (375px, 768px, 1440px)
<500 строк в одном файле
Regional compliance проверен (если применимо)
✅ Выполненные задачи (история для референса)
<details>
<summary>📜 Показать завершённые задачи (Q2-Q3 2026)</summary>

✅ P0-1.1: Settings.tsx разделён на 6 вкладок (120 строк)
✅ P0-CE.1: ComplianceProfile Abstraction Layer (profile.go, registry.go, providers.go + тесты)
✅ P0-CE.2: Regional Crypto Providers (belt, aes, gost, sm, provider.go)
✅ P0-CE.3: Hash & Signature Providers (hash_bash, signature_bign, password, password_migration)
✅ P0-CE.4: Setup Wizard (On-Premise) (wizard.go, SetupWizard.tsx)
✅ P0-CE.5: Tenant Compliance Profile (SaaS) (tenant/compliance.go)
✅ P0-CE.6: Data Residency Enforcement (storage/residency.go)
✅ P0-SEC.1: Schema Registry Validation (schema_registry.go, validated_publisher.go, circuit breaker)
✅ P0-SEC.2: SMS Provider Implementation (sms/rocketsms.go)
✅ P0-SEC.3: SLA Escalation Integration (sla/engine.go, notifier.go, policy.go, worker.go)
✅ P0-SEC.5: P2P Gateway Authentication (apiKeyAuth middleware)
✅ P0-SEC.10: Убран CREATE TABLE IF NOT EXISTS из миграций
✅ P0-SEC.11: Dependency Security Update (x/crypto, x/net, minio-go)
✅ P0-UX.1: AddDeviceModal Validation (react-hook-form + Zod + addDeviceSchema)
✅ P0-UX.2: Breadcrumbs для Detail Pages (Breadcrumbs.tsx, stories, aria-label)
✅ P0-MOBILE.1: Conflict Resolution UI (ConflictResolutionModal)
✅ P0-MOBILE.2: Background Sync Integration (useBackgroundSync.ts, syncService.ts, expo-background-fetch)
✅ P0-MOBILE.3: Offline Map Tile Caching
✅ P1-UX.4: Kanban Feedback & Animation (WOKanbanBoard с drag&drop, toast, optimistic update)
✅ P2-2.2: Command Palette Smart Search (Cmd+K, fuzzy matching)
✅ P2-2.3: Resource Planning Calendar (TechnicianWeek.tsx)
✅ P2-INT.2: OAuth2 для External Adapters (ServiceNow, Jira)
✅ P2-INT.3: Excel Import/Export для WO
✅ P3-1.2: JWT → HttpOnly Cookies (Secure, SameSite=Strict)
✅ P3-1.3: OpenTelemetry Integration (OTLP HTTP)
✅ P3-2.1: Materialized View Auto-Refresh (hourly cron)
✅ P3-2.2: Virtual Table Auto-Selection (>1000 rows)
✅ P3-3.3: Real-time Collaboration (WebSocket Presence Hub)
✅ Mobile: Conflict Resolution UI, Offline Map Tile Cache
✅ i18n: 15 языков (вкл. RTL арабский)
✅ Compliance: Compliance risks materialized view, Audit chain (ISO 27001 A.12.4)
</details>

🔴 P0 — CRITICAL (Q3 2026, до 2026-09-30)
✅ **Все P0 задачи выполнены** (см. историю выше)

P0-CE: Regional Compliance Engine (Стратегический приоритет)
P0-CE.1: ComplianceProfile Abstraction Layer ✅ DONE (2026-06-28)
P0-CE.2: Regional Crypto Providers ✅ DONE (2026-06-28)
P0-CE.3: Hash & Signature Providers ✅ DONE (2026-06-28)
P0-CE.4: Setup Wizard (On-Premise) ✅ DONE (2026-06-28)
P0-CE.5: Tenant Compliance Profile (SaaS) ✅ DONE (2026-06-28)
P0-CE.6: Data Residency Enforcement ✅ DONE (2026-06-28)

P0-REG: Regional Maintenance Compliance Engine ⭐ NEW
P0-REG.1: Maintenance Regulations Data Model — DEPENDS ON P0-CE.1, P0-CE.5 ✅
P0-REG.2: Work Order Auto-Generation from Regulations
P0-REG.3: Electronic Journal & Act Generation
P0-REG.4: Pre-loaded Regional Templates (BY, RU, KZ)

P0-SEC: Security & Data Integrity
P0-SEC.1: Schema Registry Validation ✅ DONE (2026-06-28)
P0-SEC.2: SMS Provider Implementation ✅ DONE (2026-06-28)
P0-SEC.3: SLA Escalation Integration ✅ DONE (2026-06-28)

P0-UX: Critical UX Blockers
P0-UX.1: AddDeviceModal Validation ✅ DONE (2026-06-28)
P0-UX.2: Breadcrumbs для Detail Pages ✅ DONE (2026-06-28)

P0-MOBILE: Mobile Critical Fixes
P0-MOBILE.1: Conflict Resolution UI ✅ DONE (2026-06-28)
P0-MOBILE.2: Background Sync Integration ✅ DONE (2026-06-28)
P0-MOBILE.3: Offline Map Tile Caching ✅ DONE (2026-06-28)

🟡 P1 — HIGH VALUE (Q4 2026, до 2026-12-31)
P1-SEC: Security Hardening
P1-SEC.1: CSRF Tokens для Mutations
Файлы: backend/internal/api/csrf_middleware.go
Решение: CSRF token в X-CSRF-Token, token rotation 30min
Effort: 2d
Status: [-]
P1-SEC.2: Server-Side Validation (Go-validators)
Файлы: backend/internal/api/validation.go
Решение: go-playground/validator для всех эндпоинтов, единый error format
Effort: 4d
Status: [ ]
P1-REG: Regional Maintenance Expansion ⭐ NEW
P1-REG.5: Technician Mobile Checklist
Файлы: mobile/src/screens/MaintenanceChecklistScreen.tsx
Решение:
Offline-first checklist (WatermelonDB)
Photo evidence per checklist item
Gatekeeper verification mandatory
E-signature capture
Auto-generate act на device
Критерий приёмки:
Offline checklist работает
Photo per item
E-signature captured
Effort: 4d
Status: [ ]
P1-REG.6: Regulatory Dashboard
Файлы: frontend/src/pages/RegulatoryCompliance.tsx
Решение:
Widget в Compliance Shield
KPI: upcoming TO, overdue, completed
Retention status (archive / hot / cold)
License expiration alerts
Regional compliance score
Effort: 3d
Status: [ ]
P1-REG.7: License Verification System
Файлы: backend/internal/compliance/license_verifier.go
Решение:
Auto-check license expiration
Alert 30 дней до expiration
Block WO assignment to unlicensed vendors
Integration с government registries (где есть API)
Effort: 3d
Status: [ ]
P1-UX: UX Polish & Consistency
P1-UX.3: Dashboard Unification
Файлы: frontend/src/pages/DashboardHub.tsx
Решение: Единая /dashboard с role-based widgets, Saved layouts
Effort: 4d
Status: [ ]
P1-UX.4: Skeleton на всех страницах
Файлы: frontend/src/components/layout/SkeletonPage.tsx
Решение: Добавить на WorkOrderDetail, DeviceDetail, TechnicianWeek, AdvancedAnalytics
Effort: 2d
Status: [ ]
P1-UX.5: Unified Animations
Файлы: frontend/src/index.css
Решение: CSS variables --animation-duration, --animation-easing, reduced motion
Effort: 1d
Status: [ ]
P1-UX.6: Sidebar aria-current
Файлы: frontend/src/components/layout/Sidebar.tsx
Решение: aria-current="page" для активной ссылки
Effort: 0.5d
Status: [ ]
P1-UX.7: Virtualization для больших списков
Файлы: Alerts.tsx, Notifications.tsx, AuditLog.tsx
Решение: @tanstack/react-virtual, auto-selection на основе rowCount
Effort: 2d
Status: [ ]
P1-UX.8: RCA Widget в Device Overview
Файлы: frontend/src/components/rca/RCAWidget.tsx
Решение: RCA summary в Overview tab, expandable graph, real-time updates
Effort: 3d
Status: [ ]
P1-UX.9: Saved Filters в DataGrid
Файлы: frontend/src/store/savedViewsStore.ts
Решение: Named filters, share via URL, default per role
Effort: 3d
Status: [ ]
P1-UX.10: Bulk Operations Progress
Файлы: frontend/src/components/ui/BulkProgressModal.tsx
Решение: Modal с progress bar, WebSocket updates, Cancel, Retry failed
Effort: 3d
Status: [ ]
P1-PERF: Performance Optimization
P1-PERF.1: Bundle Size Reduction (2.8MB → <2MB)
Файлы: frontend/vite.config.ts
Решение: Lazy load FullCalendar, Recharts, XLSX; route-based code splitting
Effort: 3d
Status: [ ]
P1-PERF.2: Redis для SLA Trackers и Device State
Файлы: backend/internal/state/redis_store.go
Решение: Replace in-memory map, distributed locking, TTL
Effort: 3d
Status: [ ]
P1-PERF.3: Graceful Shutdown с таймаутами
Файлы: backend/main.go
Решение: 30s timeout, context cancellation, drain queues
Effort: 2d
Status: [ ]
P1-QA: Testing & Quality Assurance
P1-QA.1: E2E Test Expansion (21 → 50+)
Файлы: frontend/e2e/*.spec.ts
Сценарии: Create WO с checklist, Complete с photo, Assign, Export, Gatekeeper, RCA view
Effort: 5d
Status: [ ]
P1-QA.2: Mobile E2E Tests (Detox/Maestro)
Файлы: mobile/e2e/*.spec.ts
Сценарии: Offline scenarios, Sync conflicts, Photo upload + Gatekeeper, Push
Effort: 5d
Status: [ ]
P1-QA.3: Accessibility Testing в CI (axe-core)
Файлы: playwright.config.ts
Решение: @axe-core/playwright, threshold: 0 critical violations
Effort: 2d
Status: [ ]
P1-QA.4: Frontend Error Monitoring (Sentry)
Файлы: frontend/src/lib/sentry.ts
Решение: Sentry SDK, source maps, user context, alerting
Effort: 2d
Status: [ ]
P1-QA.5: Load Testing (k6)
Файлы: tests/load/*.js (уже есть базовые сценарии)
Сценарии: GET /devices, POST /work-orders, WebSocket (1000 concurrent)
Effort: 3d
Status: [ ]
P1-BACKEND: Backend Quality
P1-BACKEND.1: ActionExecutor Unit Tests
Файлы: backend/internal/workflow/action_executor_test.go
Решение: Table-driven tests для всех action types
Effort: 2d
Status: [ ]
P1-BACKEND.2: PlaybookRegistry Versioning
Файлы: backend/internal/playbook/registry.go (уже есть hot reload)
Решение: Добавить version field, rollback, version history
Effort: 3d
Status: [ ]
P1-BACKEND.3: RCA Graph Auto-Update
Файлы: backend/internal/rca/graph_builder.go
Решение: Event listener для device changes, incremental updates
Effort: 3d
Status: [ ]
P1-ARCH: Architecture Improvements
P1-ARCH.1: Context Migration to Zustand (14 → 4)
Файлы: frontend/src/context/*.tsx, frontend/src/store/*.ts
Решение: Мигрировать DevicesSitesContext, MaintenanceContext, AlertsContext
Effort: 4d
Status: [ ]
P1-ARCH.2: API Routes Organization
Файлы: backend/internal/api/*.go (70+ файлов)
Решение: Группировать по доменам: work_orders/, devices/, auth/, cmms/
Effort: 3d
Status: [ ]
P1-ARCH.3: OpenAPI TypeScript Generation
Файлы: frontend/src/types/api.ts
Решение: oapi-codegen, type-safe API client, CI validation
Effort: 3d
Status: [ ]
🟢 P2 — ENTERPRISE FEATURES (Q1 2027, до 2027-03-31)
P2-MARKET: Regional Expansion ⭐ NEW
Стратегия: Использовать 15 языков i18n + ComplianceProfile для быстрого входа на рынки
Цель: $461M TAM за 9 месяцев, $6-10M ARR в Year 1
Phase 1: СНГ Foundation (Weeks 1-10)
P2-MKT.1: ГОСТ Crypto Providers (RU/KZ)
Файлы: backend/internal/crypto/providers/gost.go
TAM: $85M (RU) + $20M (KZ)
Решение: GOST 28147-89, Стрибог-256, ГОСТ Р 34.10-2012
Effort: 4d
Status: [ ]
P2-MKT.2: 152-ФЗ Features (RU/KZ shared)
Файлы: backend/internal/compliance/personal_data.go
Решение: Consent management, DSAR workflow, Data inventory, Роскомнадзор reports
Reuse: 80% для KZ, 60% для UZ
Effort: 3w (15d)
Status: [ ]
P2-MKT.3: belt-GCM + bign-curve (BY)
Файлы: backend/internal/crypto/belt.go, bign.go
Решение: belt-GCM, bign-curve256v1 для JWT, bash-256 для audit
Effort: 4w (20d)
Status: [ ]
P2-MKT.4: ОАЦ Pre-Certification Package (BY)
Файлы: docs/compliance/oac-certification/
Решение: Documentation package + СТБ compliance tests + consulting engagement
Budget: $15-25K
Effort: 4w (parallel)
Status: [ ]
P2-MKT.5: Uzbekistan Entry (Lowest Friction)
Файлы: frontend/src/locales/uz/, backend/internal/compliance/uzbekistan.go
Язык: Нужен (uz) — 1 неделя
Решение: Law "On Personal Data" (procedural), ID.UZ SSO (optional), my.gov.uz, UZS billing
Crypto: НЕ требуется!
Effort: 2w
Status: [ ]
P2-MKT.6: Kazakhstan Localization
Файлы: frontend/src/locales/kk/
Решение: Казахский язык, reuse 152-ФЗ code, eGov.kz SSO, KZT billing
Effort: 2w
Status: [ ]
Phase 2: Simple High-Demand Markets (Weeks 11-18)
P2-MKT.7: Turkey Entry (Highest ROI)
Язык: ✅ Уже есть (tr)
TAM: $42M
Решение: KVKK compliance, e-Devlet SSO, KEP (registered email), TRY billing
Crypto: НЕ требуется!
Effort: 2w
Status: [ ]
P2-MKT.8: Brazil Entry (Largest LATAM)
Язык: ✅ Уже есть (pt, minor PT-BR polish)
TAM: $75M
Решение: LGPD compliance (reuse 70% GDPR code), Gov.br SSO, PIX, BRL billing
Crypto: НЕ требуется!
Effort: 2w
Status: [ ]
P2-MKT.9: Mexico Entry (Nearshoring Boom)
Язык: ✅ Уже есть (es)
TAM: $50M
Решение: LFPDPPP, SAT integration, CURP, MXN billing
Effort: 2w
Status: [ ]
P2-MKT.10: Vietnam Entry (Fastest Growing 28% YoY)
Язык: ❌ Нужен (vi) — 1 неделя
TAM: $50M
Решение: Вьетнамский язык, Decree 13/2023 (data residency), VND billing
Effort: 2w
Status: [ ]
P2-MKT.11: Indonesia Entry (Largest SEA)
Язык: ❌ Нужен (id) — 1 неделя
TAM: $65M
Решение: Bahasa Indonesia, UU PDP, SATUSEHAT (health), IDR billing
Effort: 2w
Status: [ ]
Phase 3: Africa Expansion (Weeks 19-24)
P2-MKT.12: Nigeria Entry
Язык: ✅ Уже есть (en)
TAM: $32M (30% YoY — fastest growing)
Решение: NDPR, NIMC, BVN, NGN billing, low-bandwidth mode
Effort: 2w
Status: [ ]
P2-MKT.13: Kenya Entry (East Africa Hub)
Язык: ❌ Нужен Swahili (sw)
TAM: $10M (но East Africa hub)
Решение: DPA 2019, M-Pesa integration (CRITICAL), eCitizen, KES billing
Effort: 2w
Status: [ ]
P2-MKT.14: South Africa Entry
Язык: ✅ Уже есть (en)
TAM: $20M
Решение: POPIA, Cybercrimes Act, SARS, ZAR billing
Effort: 2w
Status: [ ]
P2-REG: Advanced Maintenance Templates ⭐ NEW
P2-REG.8: Templates для TR, VN, ID, BR, ZA
Решение:
TR: KVKK + TS EN 62676 (CCTV specific)
VN: TCVN 11930:2017 + Camera Standard 2025
ID: SNI 27001 + UU PDP
BR: ABNT NBR + LGPD
ZA: SANS + POPIA
Effort: 6d
Status: [ ]
P2-CR: Compliance Features
P2-CR.1: Regional Retention Policies
Решение: Per-region retention (BY 5y, EU min necessary, CN 6m)
Effort: 3d
Status: [ ]
P2-CR.2: Regional Compliance Reports
Решение: PDF/XML для ОАЦ, ФСТЭК, GDPR DPIA, NIS2
Effort: 5d
Status: [ ]
P2-CR.3: Regional Password Policies
Решение: BY: 12 chars + 90d rotation; EU: NIST (no forced rotation)
Effort: 2d
Status: [ ]
P2-CR.4: Session & Auth Regional Policies
Решение: BY: 30 min timeout (КИИ); RU: 15 min (ФСТЭК); EU/US: 8h
Effort: 2d
Status: [ ]
P2-AI: Advanced Analytics & AI
P2-AI.1: Real ML Model Integration
Файлы: backend/analytics/predict.py
Решение: XGBoost на real TimescaleDB data, NATS publishing, confidence score
Effort: 5d
Status: [ ]
P2-AI.2: AI Assistant Chat
Файлы: frontend/src/components/ai/AIAssistantPanel.tsx
Решение: DeepSeek integration, context-aware recommendations, RCA suggestions
Effort: 4d
Status: [ ]
P2-WF: Workflow & Automation
P2-WF.1: Workflow Builder UI
Файлы: frontend/src/components/workflow/WorkflowBuilder.tsx
Решение: React Flow, CEL conditions editor, testing mode, version control
Effort: 5d
Status: [ ]
P2-WF.2: Resource Planning Calendar ✅ DONE
Status: [x] TechnicianWeek.tsx с drag-and-drop
P2-INT: Integration Ecosystem
P2-INT.1: Webhook Builder UI ✅ DONE
Status: [x] WebhookBuilder.tsx
P2-INT.2: OAuth2 для External Adapters ✅ DONE
Status: [x] ServiceNow, Jira с encrypted storage
P2-INT.3: Excel Import/Export для WO ✅ DONE
Status: [x] Export handlers
🔵 P3 — TECHNICAL DEBT (Q2 2027, до 2027-06-30)
P3-SEC: Security & Compliance
P3-SEC.1: belt-GCM Migration (СТБ 34.101.31)
Решение: Migration script, backward compatibility, security audit
Effort: 4d
Status: [ ]
P3-SEC.2: JWT bign-curve256v1 Migration
Решение: СТБ bign-curve, token rotation
Effort: 3d
Status: [ ]
P3-SEC.3: Mobile Certificate Pinning
Решение: Pin server certificates, rotation support
Effort: 2d
Status: [ ]
P3-DX: Developer Experience
P3-DX.1: Storybook Expansion (8 → 50+ stories)
Файлы: frontend/src/components/**/*.stories.tsx
Приоритет: DataGrid, AssetTree, WorkOrderPrintView
Effort: 5d
Status: [ ]
P3-DX.2: Onboarding Tour для всех ролей
Решение: Role-specific tours (Technician, Manager, Admin)
Effort: 3d
Status: [ ]
P3-DX.3: Help System & Glossary
Файлы: frontend/src/pages/Help.tsx, Glossary.tsx
Решение: FAQ, 50+ terms, search, video tutorials, i18n
Effort: 3d
Status: [ ]
P3-DX.4: DEVELOPMENT.md
Решение: Local setup, env vars, run instructions, troubleshooting
Effort: 1d
Status: [ ]
P3-DX.5: Swagger UI на /api/v1/docs
Решение: Auto-generate из OpenAPI, authentication, Try it out
Effort: 1d
Status: [ ]
P3-UI: UI/UX Polish
P3-UI.1: Design Tokens (CSS variables)
Решение: CSS variables для цветов, spacing, typography, dark mode tokens
Effort: 2d
Status: [ ]
P3-UI.2: Micro-interactions
Решение: Ripple для кнопок, hover-тени для карточек, haptic feedback
Effort: 2d
Status: [ ]
P3-UI.3: Mobile Responsiveness (FlatList вместо ScrollView)
Файлы: mobile/src/screens/*.tsx
Решение: FlatList для больших списков, swipe жесты, lazy loading
Effort: 3d
Status: [ ]
P3-NICE: Nice-to-Have
P3-NICE.1: Real-time Collaboration ✅ DONE
Status: [x] WebSocket Presence Hub
P3-NICE.2: White-label Theming
Решение: Custom logo/colors per tenant, CSS variables
Effort: 3d
Status: [ ]
P3-NICE.3: Edge Agent SL-4 Security
Решение: Secure boot, mTLS, tamper detection, HSM
Effort: 5d
Status: [ ]
📊 Success Metrics
Метрика
Текущее
Target (Q4 2026)
Target (Q2 2027)
Bundle Size
2.8MB
<2MB
<1.5MB
Lighthouse Score
87
>95
>98
E2E Coverage
21 scenarios
50+
80+
Mobile E2E
0
20+
50+
A11y Violations
Unknown
0 critical
0 violations
Context Count
14
4
2
Test Coverage (React)
75%
>80%
>85%
Test Coverage (Go)
85%
>88%
>90%
Supported Regions
1 (BY)
3 (BY, EU, INTL)
14 (+RU, KZ, UZ, TR, BR, MX, VN, ID, NG, KE, ZA, AE)
Maintenance Regulations
0
3 (BY, RU, KZ)
10+
Active Markets
1
4
14
Regional Revenue %
100% BY
30% BY / 70% INTL
40% BY / 60% global
ARR from New Markets
$0
$3M
$6-10M


Приоритизационные правила для Roo
Правило 1: Language-First
Если язык уже есть в i18n (tr, pt, es, ar) → market entry 2 недели вместо 3.
✅ TR, BR, MX: приоритет выше
⚠️ VN, ID, KE: +1 неделя на localization
Правило 2: Procedural Before Crypto
Procedural compliance (consent, DSAR, reports) всегда перед крипто-сертификацией.
Crypto только для BY, RU, KZ (3 из 14 рынков)
11 рынков работают на INTL profile (AES-256-GCM)
Правило 3: Reuse Matrix
152-ФЗ (RU) → переиспользуется для:
  ├─ Kazakhstan (80%)
  ├─ Uzbekistan (60%)
  ├─ Kyrgyzstan (90%)
  └─ Armenia (70%)

GDPR (EU) → переиспользуется для:
  ├─ Turkey/KVKK (80%)
  ├─ Brazil/LGPD (85%)
  ├─ Indonesia/UU PDP (75%)
  ├─ South Africa/POPIA (80%)
  ├─ Nigeria/NDPR (70%)
  └─ Kenya/DPA (75%)

  Правило 4: Partner-First Entry
Для каждого рынка сначала найти local partner:
СНГ: System integrators с госсвязями
Turkey: KVKK consulting firms
Brazil: Totvs ecosystem
SEA: Telkom (ID), FPT (VN)
Africa: MTN (NG), Safaricom (KE), Vodacom (ZA)
Без partner → отложить market entry.
Правило 5: Maintenance Compliance = Differentiator
РД 25.964-90 automation — 0 конкурентов имеют digital журнал с HMAC-signature.
Gatekeeper + regulatory act — уникальное сочетание (GPS + AI + e-signature = юридически значимый акт).
🔗 Полезные ссылки
Architecture: ARCHITECTURE.md
UX Guidelines: docs/ux/ux-guideline.md
ADR Log: docs/adr/
API Docs: backend/docs/api/
Design System: frontend/.storybook/
CI/CD: .github/workflows/
Regional Compliance: docs/compliance/regional-profiles.md (создать)
Maintenance Regulations: docs/compliance/maintenance-regulations.md (создать)
Security Policy: docs/iso27001/security-policy.md
📝 История изменений
2026-06-28 — Major Update: Все P0 задачи отмечены как DONE, начато выполнение P1
✅ Все P0-CE задачи (1-6) проверены и подтверждены как реализованные
✅ Все P0-SEC задачи (1-3) проверены и подтверждены как реализованные
✅ Все P0-UX задачи (1-2) проверены и подтверждены как реализованные
✅ Все P0-MOBILE задачи (1-3) проверены и подтверждены как реализованные
➡️ Начало P1-SEC.1: CSRF Tokens для Mutations
2026-06-28 — Major Update: Regional Maintenance Compliance Engine
✅ Добавлена P0-REG секция: Maintenance Regulations Data Model (7 таблиц)
✅ Добавлена P1-REG секция: Mobile Checklist, Regulatory Dashboard, License Verification
✅ Добавлена P2-REG секция: Templates для TR, VN, ID, BR, ZA
✅ Добавлена P2-MARKET секция: 14 regional expansion tasks (СНГ + Simple + Africa)
✅ Интеграция с существующими компонентами:
MaintenanceCron → regulatory_cron.go
audit/chain.go (HMAC) → signed journals
Gatekeeper → evidence для regulatory acts
15 языков i18n → 9 рынков с existing localization
✅ Нормативная база покрыта:
🇧🇾 СН 3.02.19-2025 (вводится 24.09.2025)
🇷🇺 РД 25.964-90, РД 009-01-96, РД 009-02-96, РД 78.145-93
🇰🇿 Приказ МЧС №55 (с 01.02.2026 — лицензия обязательна)
🇹🇷 KVKK, 🇻🇳 TCVN 11930, 🇮🇩 SNI 27001, 🇧🇷 ABNT NBR, 🇿🇦 SANS
✅ Добавлены success metrics для maintenance compliance
✅ Обновлён roadmap с учётом regional expansion
2026-06-27 — Previous Update
✅ Выполнены: P0-SEC.5 (P2P Gateway), P0-SEC.10-11, P3-1.2-3, P3-2.1-2, P3-3.3
✅ Добавлена P0-MARKET секция (Regional Compliance Engine)
✅ Интеграция ComplianceProfile с P0-MKT задачами
2026-06-26 — Initial TODO Creation
Создан unified TODO на основе 3 code reviews
Определены P0-P3 приоритеты
✅ Чеклист для Roo перед началом работы
Прочитать соответствующий раздел TODO
Проверить dependencies (если есть)
Создать feature branch: feature/P0-REG.1-maintenance-regulations
Реализовать задачу с учётом code review чеклиста
Написать тесты (unit + integration)
Обновить документацию (если применимо)
Создать PR с описанием изменений
Отметить [x] + дата в TODO после merge
Последний коммит: HEAD
Branch: main
Next Review: 2026-07-05
📚 Appendix: Матрица нормативных документов
Регион
Документ
Сфера
Периодичность ТО
Retention
🇧🇾 BY
СН 3.02.19-2025
CCTV
1/3/12 мес
10 лет
🇧🇾 BY
ТКП 472-2013
ОПС
1/6/12 мес
10 лет
🇷🇺 RU
РД 25.964-90
АУПТ, ОПС
1/6/12 мес
10 лет
🇷🇺 RU
РД 009-01-96
Пожарная автоматика
1/6/12 мес
10 лет
🇷🇺 RU
РД 009-02-96
ТО и ППР
1/6/12 мес
10 лет
🇷🇺 RU
РД 78.145-93
ОПС монтаж
1/6/12 мес
10 лет
🇷🇺 RU
ГОСТ Р 51558-2014
CCTV
1/3/12 мес
10 лет
🇰🇿 KZ
Приказ МЧС №55
Пожарная автоматика
1/3/12 мес
10 лет
🇰🇿 KZ
СТ РК ГОСТ Р 50776-2010
Тревожная сигнализация
1/3/12 мес
10 лет
🇰🇿 KZ
Закон РК «Об охранной деятельности»
Все
1/3/12 мес
10 лет
🇹🇷 TR
KVKK №6698
CCTV + ПДн
3/12 мес
5 лет
🇹🇷 TR
TS EN 62676
CCTV
3/12 мес
5 лет
🇻🇳 VN
TCVN 11930:2017
ИБ
3/12 мес
5 лет
🇻🇳 VN
Camera Standard (15.02.2025)
CCTV
3/12 мес
5 лет
🇮🇩 ID
SNI 27001
ISMS
3/12 мес
5 лет
🇮🇩 ID
UU PDP (2022)
ПДн
3/12 мес
5 лет
🇧🇷 BR
ABNT NBR series
CCTV + ОПС
3/6/12 мес
5 лет
🇧🇷 BR
LGPD
ПДн
3/6/12 мес
5 лет
🇿🇦 ZA
SANS 10160-4
Безопасность объектов
3/6/12 мес
5 лет
🇿🇦 ZA
POPIA
ПДн
3/6/12 мес
5 лет
🇰🇪 KE
KS 2110-4/5:2009
CCTV
3/6/12 мес
5 лет
🇰🇪 KE
DPA 2019
ПДн
3/6/12 мес
5 лет
💼 Бизнес-ценность
Revenue Impact
Возможность
TAM
Pricing Premium
Compliance Automation для КИИ РБ
$7M
+40% (enterprise tier)
МЧС лицензия для RU/KZ
$105M
+30% (certified vendor)
KVKK compliance для TR
$42M
+25% (legal protection)
LGPD/POPIA compliance
$115M
+20% (risk mitigation)
Competitive Moat
РД 25.964-90 automation — 0 конкурентов имеют digital версию журнала с HMAC-signature
Gatekeeper + regulatory act — уникальное сочетание (GPS + AI + e-signature = юридически значимый акт)
Multi-region compliance engine — только enterprise-вендоры (ServiceNow) имеют подобное, но не для CCTV
Risk Mitigation
Риск
Последствие
Наша защита
Штрафы МЧС РК
до $50K + уголовная ответственность
License verification + automated ТО
ОАЦ РБ несоответствие
Запрет на КИИ работы
Pre-audit compliance reports
KVKK штрафы
€15M или 2.5% оборота
Privacy signage + VERBIS automation
LGPD/POPIA lawsuits
до 2% оборота
Automated DSAR + retention
Bottom line: Добавление MaintenanceComplianceProfile превращает систему из "CCTV monitoring tool" в enterprise compliance platform с юридически значимой документацией, готовой для предъявления регуляторам. Это ключевой differentiator для КИИ-сектора и enterprise-сделок на 14+ рынках с суммарным TAM $461M.
Следующий шаг для Roo: Начать с P0-REG.1: Maintenance Regulations Data Model (5 дней) — это создаст фундамент для всех остальных задач блока P0-REG и P2-REG.
