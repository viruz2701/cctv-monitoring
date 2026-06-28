# CCTV Health Monitor — TODO & Roadmap

## Правила для Roo

### Перед задачей
- Прочитай соответствующий раздел TODO, проверь dependencies
- Прочитай связанный ADR из `docs/adr/`
- Определи compliance-стандарты (см. матрицу стандартов)

### Во время
- Атомарные коммиты с ID задачи: `P0-CE.1: ComplianceProfile`
- Фиксируй прогресс в TODO: `⏳` → `✅ DONE`
- Формат коммита: `feat(scope): description`

### После завершения
- `[x]` + дата
- Проверь критерий приёмки — если не выполнен, задача не завершена
- Обнови метрику в Success Metrics
- Добавь commit hash

### Чеклист для каждой задачи
- [ ] Dark mode + Light mode
- [ ] WCAG 2.1 AA (aria, contrast, keyboard)
- [ ] i18n (20 языков)
- [ ] Error handling + retry
- [ ] Unit + Integration tests
- [ ] Regional compliance проверен (если применимо)
- [ ] <500 строк в одном файле
- [ ] No console errors/warnings

---

## ✅ Выполненные задачи (история для референса)

<details>
<summary>📜 Показать завершённые задачи (Q2-Q3 2026)</summary>

✅ P0-CE.1: ComplianceProfile Abstraction Layer
✅ P0-CE.2: Regional Crypto Providers (belt, aes, gost, sm)
✅ P0-CE.3: Hash & Signature Providers (hash_bash, signature_bign)
✅ P0-CE.4: Setup Wizard (On-Premise)
✅ P0-CE.5: Tenant Compliance Profile (SaaS)
✅ P0-CE.6: Data Residency Enforcement
✅ P0-SEC.1: Schema Registry Validation
✅ P0-SEC.2: SMS Provider Implementation
✅ P0-SEC.3: SLA Escalation Integration
✅ P0-SEC.5: P2P Gateway Authentication
✅ P0-SEC.10: Убран CREATE TABLE IF NOT EXISTS из миграций
✅ P0-SEC.11: Dependency Security Update
✅ P0-UX.1: AddDeviceModal Validation
✅ P0-UX.2: Breadcrumbs для Detail Pages
✅ P0-MOBILE.1: Conflict Resolution UI
✅ P0-MOBILE.2: Background Sync Integration
✅ P0-MOBILE.3: Offline Map Tile Caching
✅ P1-SEC.1: CSRF Tokens
✅ P1-SEC.2: Server-Side Validation
✅ P1-REG.5: Technician Mobile Checklist
✅ P1-REG.6: Regulatory Dashboard
✅ P1-REG.7: License Verification System
✅ P1-PERF.1: Bundle Size Reduction (17 vendor chunks)
✅ P1-PERF.2: Redis Device State Store
✅ P1-PERF.3: Graceful Shutdown
✅ P1-BACKEND.1: ActionExecutor Unit Tests
✅ P1-BACKEND.2: PlaybookRegistry Versioning
✅ P1-BACKEND.3: RCA Graph Auto-Update
✅ P1-QA.1: E2E Test Expansion (21→109)
✅ P1-QA.2: Mobile E2E Tests (86 тестов)
✅ P1-QA.3: Accessibility Testing CI (axe-core)
✅ P1-QA.4: Sentry Error Monitoring
✅ P1-QA.5: Load Testing k6
✅ P1-UX.3: Dashboard Unification
✅ P1-UX.4: Skeleton на всех страницах
✅ P1-UX.5: Unified Animations
✅ P1-UX.6: Sidebar aria-current
✅ P1-UX.7: Virtualization
✅ P1-UX.8: RCA Widget
✅ P1-UX.9: Saved Filters
✅ P1-UX.10: Bulk Operations Progress
✅ P1-ARCH.1: Context Migration to Zustand
✅ P1-ARCH.2: API Routes Organization
✅ P1-ARCH.3: OpenAPI TypeScript Generation
✅ P2-MKT.1: ГОСТ Crypto Providers (RU/KZ)
✅ P2-MKT.2: 152-ФЗ Features (RU/KZ)
✅ P2-CR.1: Regional Retention Policies
✅ P2-CR.2: Regional Compliance Reports
✅ P2-CR.3: Regional Password Policies
✅ P2-CR.4: Session & Auth Regional Policies
✅ P2-AI.1: Real ML Model Integration
✅ P2-AI.2: AI Assistant Chat
✅ P2-WF.1: Workflow Builder UI
✅ P2-WF.2: Resource Planning Calendar
✅ P2-INT.1: Webhook Builder UI
✅ P2-INT.2: OAuth2 для External Adapters
✅ P2-INT.3: Excel Import/Export для WO
✅ P2-REG.8: Regional Templates (31 регламент)
✅ P3-SEC.3: Mobile Certificate Pinning
✅ P3-DX.1: Storybook Expansion (58 stories)
✅ P3-DX.2: Onboarding Tour
✅ P3-DX.3: Help System & Glossary
✅ P3-DX.4: DEVELOPMENT.md
✅ P3-DX.5: Swagger UI
✅ P3-UI.1: Design Tokens
✅ P3-UI.2: Micro-interactions
✅ P3-UI.3: Mobile Responsiveness
✅ P3-NICE.1: Real-time Collaboration
✅ P3-NICE.2: White-label Theming
✅ Code Review Bug Fixes: 6 bugs (CardBody, IntersectionObserver, Router, LazyImage, JSX, 'j' char)
✅ POLISH: 23 задач code review (5 phases)
✅ i18n: 20 языков (+vi, id, sw, kk, uz)
</details>

---

## 🔴 P0 — CRITICAL BLOCKERS (Q3 2026, до 2026-09-30)

### P0-PDF: Server-Side PDF Generation (Гибридный подход)
**Файлы**: `backend/internal/reports/generator.go`, `frontend/src/styles/print.css`
**Контекст**: Удалить jsPDF/html2canvas с фронта (-280KB), генерировать PDF на Go-сервере
**Effort**: 8d | **Статус**: [ ]

- [ ] **P0-PDF.1**: CSS `@media print` framework для quick preview
- [ ] **P0-PDF.2**: Server-side generation с HMAC signatures + QR verification
- [ ] **P0-PDF.3**: Report Generation Queue (NATS + async jobs)
- [ ] **P0-PDF.4**: Удалить jspdf/html2canvas из bundle
- [ ] **P0-PDF.5**: Regional templates (ОАЦ, ФСТЭК, МЧС РК, KVKK)

### P0-SBOM: Supply Chain Security (EU CRA blocker)
**Файлы**: `.github/workflows/sbom.yml`, `backend/sbom.json`
**Контекст**: EU CRA (Dec 2027) и US EO 14028 требуют SBOM при продаже ПО.
**Effort**: 3d | **Статус**: ⏳ Частично DONE (commit 6fb4d13)

- [x] **P0-SBOM.1**: CycloneDX/SPDX auto-generation в CI ✅
- [x] **P0-SBOM.2**: `/.well-known/security.txt` (RFC 9116) ✅
- [x] **P0-SBOM.3**: Security advisories page + RSS ✅
- [ ] **P0-SBOM.4**: CNA application (CVE Numbering Authority)

### P0-IR: Multi-Tier Incident Response
**Файлы**: `backend/internal/compliance/incident_response.go`
**Контекст**: Разные регионы требуют разные сроки reporting
**Effort**: 5d | **Статус**: ⏳ Частично DONE (commit 3aa2677)

- [x] **P0-IR.1**: Classification engine (NIS2/DORA/CERT-In) ✅
- [x] **P0-IR.2**: 6h CERT-In reporting (India) ✅
- [x] **P0-IR.3**: 4h DORA reporting (EU) ✅
- [x] **P0-IR.4**: Evidence preservation (immutable snapshots) ✅

### P0-REG: Maintenance Compliance Engine
**Файлы**: `backend/internal/db/migrations/040_maintenance_regulations.up.sql`
**Контекст**: Регуляторные требования к ТО систем БЖиО (BY, RU, KZ, TR, VN, ID, BR)
**Effort**: 12d | **Статус**: [ ]

- [ ] **P0-REG.1**: Data model: regulations, checklists, journals, acts, licenses
- [ ] **P0-REG.2**: Pre-loaded templates (BY, RU, KZ, TR, VN, ID, BR)
- [ ] **P0-REG.3**: Auto-generation WO из compliance schedules
- [ ] **P0-REG.4**: Electronic journal + HMAC-signed acts
- [ ] **P0-REG.5**: Mobile checklist с offline-first

### P0-CLEANUP: Remove Legacy Dependencies
**Файлы**: `frontend/package.json`, `frontend/vite.config.ts`
**Контекст**: После миграций остались упоминания старых пакетов
**Effort**: 3d | **Статус**: ⏳ Частично

- [x] **P0-CLEANUP.1**: Удалить xlsx/recharts/@fullcalendar из package.json ✅
- [x] **P0-CLEANUP.2**: Проверить vendor-pdf, vendor-calendar chunks ✅
- [ ] **P0-CLEANUP.3**: Удалить html2canvas если не используется
- [ ] **P0-CLEANUP.4**: Проверить jspdf — оставить (dynamic import, нужен для совместимости)

---

## 🟡 P1 — HIGH VALUE (Q4 2026)

### P1-PERF-BUNDLE: Bundle Size Optimization ✅ ВСЁ DONE

| Чанк | Размер | gzip | Статус |
|------|--------|------|--------|
| `vendor-schedule-x` | 167.78 KB | 41.92 KB | ✅ Schedule-X |
| `vendor-nivo` | 386.89 KB | 122.76 KB | ✅ Nivo |
| `vendor-excel` (ExcelJS) | 929.91 KB | 256.48 KB | ✅ MIT license |
| `vendor-pdf` (jsPDF) | 558.06 KB | 162.75 KB | ✅ lazy-loaded |
| `vendor-other` | 397.58 KB | 130.44 KB | ⚠️ |
| `index` (main) | 612.19 KB | 161.69 KB | ⚠️ |
| **Precache total** | **5015.31 KB** | — | **Target: <2MB** |

- [x] **Quick Wins**: lazy-load jsPDF, react-joyride, react-datepicker (commit `b01ef28`)
- [x] **BUNDLE.1**: FullCalendar (~328KB) → Schedule-X (~168KB) (commit `8eccc81`)
- [x] **BUNDLE.2**: Recharts (~440KB) → Nivo (~387KB) (commit `5f78b99`)
- [x] **BUNDLE.3**: xlsx/SheetJS (~425KB, Pro license) → ExcelJS (~930KB, MIT) (commit `45cdd63`)

### P1-OPT: Bundle Micro-Optimizations
**Effort**: 6.5d | **Статус**: [ ]

- [ ] **P1-OPT.1**: lucide-react tree-shaking (централизованный Icons.tsx вместо разрозненных импортов)
- [ ] **P1-OPT.2**: `@xyflow/react` lazy-load (только на странице WorkflowBuilder)
- [ ] **P1-OPT.3**: i18n lazy-load per language (не грузить все 20 языков сразу)
- [ ] **P1-OPT.4**: `react-grid-layout` lazy-load (только на DashboardHub)

### P1-QUOTA: SaaS Protection (Tenant Quota Management)
**Файлы**: `backend/internal/tenant/quota.go`, `backend/internal/db/migrations/043_tenant_quotas.sql`
**Проблема**: Нет ограничений на ресурсы tenant → risk of abuse в SaaS
**Effort**: 4d | **Статус**: [ ]

- Quotas: devices, users, storage, API calls, work orders
- Usage tracking (Redis real-time counters)
- Soft limit (80% warning) + Hard limit (100% block)
- Over-quota grace period (7 дней)
- Admin UI для quota management + usage dashboard

### P1-MARKET: Playbook Marketplace
**Файлы**: `frontend/src/pages/PlaybookMarketplace.tsx`, `backend/internal/playbook/marketplace.go`
**Effort**: 5d | **Статус**: [ ]

- Public marketplace с pre-built playbooks (Hikvision, Dahua, Axis, Uniview)
- Rating + review system, version compatibility matrix
- One-click install, private sharing между tenants
- Vendor-verified badges

### P1-CALENDAR: External Calendar Sync (Google + Outlook)
**Файлы**: `backend/internal/integrations/calendar/google.go`, `calendar/outlook.go`
**Effort**: 5d | **Статус**: [ ]

- Google Calendar API + Microsoft Graph API (OAuth2)
- Auto-create events при WO assignment
- Auto-update при status change / reschedule
- Bi-directional sync + conflict detection

### P1-PHOTO: Advanced Photo Annotation
**Файлы**: `frontend/src/components/PhotoAnnotation.tsx`, `mobile/src/components/PhotoAnnotation.tsx`
**Effort**: 4d | **Статус**: [ ]

- Freehand drawing, text labels, measurement tool
- Blur/Redact sensitive areas (faces, license plates)
- Layer management + export annotated image
- Annotation history per photo

### P1-SYNC: Differential Sync для Mobile
**Файлы**: `mobile/src/services/differentialSync.ts`, `backend/internal/api/sync/diff.go`
**Effort**: 5d | **Статус**: [ ]

- Delta sync (only changed fields)
- Change tracking via updated_at + field-level diff
- Compression (gzip/brotli), bandwidth monitoring
- Partial sync priority (WO status > photos > audit)

### P1-RATE: Rate Limiting Middleware
**Файлы**: `backend/internal/api/rate_limiter.go`, `backend/internal/api/middleware/ratelimit.go`
**Effort**: 3d | **Статус**: [ ]

- Token bucket per tenant/user (Redis-based)
- Configurable limits: read 100/min, write 30/min
- X-RateLimit-* headers, 429 Retry-After
- Prometheus metrics

### P1-REPLAY: Event Replay UI
**Файлы**: `frontend/src/pages/EventReplay.tsx`, `backend/internal/events/replay.go`
**Effort**: 4d | **Статус**: [ ]

- Event browser (filter by type, tenant, date)
- JSON payload viewer, replay capability
- Dead letter queue viewer
- Event flow visualization (Sankey diagram)

### P1-QA: Testing Expansion
**Effort**: 10d | **Статус**: [ ]

- [ ] **P1-QA.1**: E2E: 109 → 150 scenarios
- [ ] **P1-QA.2**: Mobile E2E: 86 → 100 tests
- [ ] **P1-QA.3**: Go coverage: 85% → 90%
- [ ] **P1-QA.4**: Frontend coverage: 82% → 85%

---

## 🟢 P2 — STRATEGIC (Q1 2027)

### P2-BI: Embedded Self-Service Analytics
**Файлы**: `frontend/src/pages/CustomReports.tsx`, `backend/internal/analytics/query_builder.go`
**Effort**: 6d | **Статус**: [ ]

- Visual query builder (drag-and-drop dimensions + measures)
- Pre-built SQL templates (MTTR, MTBF, first-time fix rate, cost per WO)
- Custom charts + saved reports + scheduled delivery
- Export: PDF, Excel, CSV, PNG

### P2-CHAT: Real-Time Collaboration
**Файлы**: `frontend/src/components/chat/WOChat.tsx`, `backend/internal/ws/chat.go`
**Effort**: 5d | **Статус**: [ ]

- WebSocket chat per WO: text, photo, voice note, checklist reference
- @mentions + push notifications, reactions, read receipts
- Searchable history, offline queue
- **P2-CHAT.3 (NEW)**: Voice-to-text notes (Web Speech API + expo-speech)

### P2-CHECK: Conditional Checklists (MaintainX-level)
**Файлы**: `frontend/src/components/checklists/ConditionalChecklist.tsx`, `backend/internal/models/checklist.go`
**Effort**: 4d | **Статус**: [ ]

- depends_on/operator/value conditions, dynamic show/hide
- Sub-items, scoring, mandatory vs optional
- Conditional required photos, templates per device type

### P2-FIELDS: Custom Fields Advanced (Shelf.nu-level)
**Файлы**: `frontend/src/components/custom-fields/FieldBuilder.tsx`, `backend/internal/models/custom_field.go`
**Effort**: 6d | **Статус**: [ ]

- 15+ field types: text, number, date, dropdown, multi-select, URL, email, barcode, signature, file upload
- Validation rules, conditional visibility, field groups
- Bulk apply, REST API, drag-and-drop ordering

### P2-API: API Versioning Strategy
**Файлы**: `backend/internal/api/versioning.go`, `backend/internal/api/v1/`, `backend/internal/api/v2/`
**Effort**: 3d | **Статус**: [ ]

- URL-based (/api/v1/, /api/v2/) + header-based (X-API-Version)
- Deprecation policy (6 months notice + Sunset header)
- API changelog + migration guides + backward compat tests

### P2-REGIONS: Regional Expansion
**Effort**: 24d | **Статус**: [ ]

- [ ] **P2-REGIONS.1**: EU: GDPR + NIS2 + CRA preparation
- [ ] **P2-REGIONS.2**: US: NERC CIP gap analysis
- [ ] **P2-REGIONS.3**: China: SM crypto + MLPS 2.0
- [ ] **P2-REGIONS.4**: India: CERT-In 6h reporting
- [ ] **P2-REGIONS.5**: Market entries (TR, BR, MX, VN, ID, NG, KE, ZA)

---

## 🔵 P3 — POLISH & DEBT (Q2 2027)

### P3-MONITOR: Observability Stack
**Файлы**: `infra/grafana/dashboards/`, `infra/prometheus/rules/`
**Effort**: 4d | **Статус**: [ ]

- Grafana dashboards: System Health, SLA, API Performance, NATS Events, DB Queries
- Prometheus alerting: error rate, slow queries, NATS lag, disk
- SLO/SLI tracking + error budget
- Public status page для SaaS

### P3-DR: Disaster Recovery Automation
**Файлы**: `infra/dr/failover.sh`, `infra/dr/runbook.md`, `backend/internal/dr/health.go`
**Effort**: 5d | **Статус**: [ ]

- Automated health checks (30s interval)
- Auto-failover с admin confirmation
- DR drill automation (quarterly)
- RTO/RPO monitoring dashboard

### P3-DB: Database Optimization
**Файлы**: `backend/internal/db/pool.go`, `backend/config.yaml`
**Effort**: 3d | **Статус**: [ ]

- PgBouncer (transaction mode), read replicas routing
- Pool monitoring (active, idle, wait), slow query detection
- Query plan analysis, index recommendations

### P3-AR: AR-Assisted Maintenance (R&D)
**Файлы**: `mobile/src/screens/ARMaintenance.tsx`, `mobile/src/components/AROverlay.tsx`
**Effort**: 12d | **Статус**: [ ]

- ARKit/ARCore overlay для equipment identification
- QR scan → AR overlay с device info
- Virtual navigation arrows
- AR checklist overlay + photo capture

### P3-WL: White-Label Theming
**Файлы**: `frontend/src/store/whiteLabelStore.ts`, `frontend/src/components/WhiteLabelConfigurator.tsx`
**Effort**: 4d | **Статус**: [ ]

- Per-tenant logo + colors + custom domain (CNAME)
- Branded emails + PDFs
- Preview mode

### P3-DX: Developer Experience
**Effort**: 9d | **Статус**: [ ]

- [ ] **P3-DX.1**: Storybook: 58 → 80+ stories
- [ ] **P3-DX.2**: Glossary: 30 → 50+ терминов
- [ ] **P3-DX.3**: DEVELOPMENT.md + Swagger UI (обновить)

### P3-CERT: Certifications (External process)
**Статус**: External

- [ ] **P3-CERT.1**: ISO 27001 + SOC 2 (6 months, ~$60K)
- [ ] **P3-CERT.2**: ОАЦ РБ (8 weeks, ~$25K)
- [ ] **P3-CERT.3**: ФСТЭК РФ (12 weeks, ~$40K)
- [ ] **P3-CERT.4**: EU CRA notified body assessment

---

## 📊 Success Metrics

| Метрика | Current | Q4 2026 Target | Q2 2027 Target |
|---------|---------|----------------|----------------|
| Bundle Size (precache) | 5.02 MB | <2 MB | <1.5 MB |
| Bundle gzip | 1.62 MB | <800 KB | <600 KB |
| Lighthouse Score | 87 | >95 | >98 |
| Go Coverage | 85% | 90% | 95% |
| Frontend Coverage | 82% | 85% | 90% |
| E2E Scenarios | 109 | 150 | 200 |
| Mobile E2E | 86 | 100 | 120 |
| A11y Violations | 0 critical | 0 | 0 |
| SBOM | CycloneDX ✅ | + VEX | Automated |
| Supported Regions | 10 | 12 | 15+ |
| Certifications | 0 | ISO 27001 prep | 2-3 active |
| ML Accuracy | Synthetic | 75% production | 85% |
| Enterprise Playbooks | 3 | 20+ | 50+ |
| RTO/RPO | Manual | <15min / <5min | Automated |

---

## 📚 Приоритизационные правила

### Правило 1: Language-First
Если язык уже есть в i18n (tr, pt, es, ar) → market entry 2 недели вместо 3.
✅ TR, BR, MX: приоритет выше
⚠️ VN, ID, KE: +1 неделя на localization

### Правило 2: Procedural Before Crypto
Procedural compliance (consent, DSAR, reports) всегда перед крипто-сертификацией.
Crypto только для BY, RU, KZ (3 из 14 рынков)
11 рынков работают на INTL profile (AES-256-GCM)

### Правило 3: Reuse Matrix
**152-ФЗ (RU)** → переиспользуется для: Kazakhstan (80%), Uzbekistan (60%), Kyrgyzstan (90%), Armenia (70%)
**GDPR (EU)** → переиспользуется для: Turkey/KVKK (80%), Brazil/LGPD (85%), Indonesia/UU PDP (75%), South Africa/POPIA (80%), Nigeria/NDPR (70%), Kenya/DPA (75%)

### Правило 4: Partner-First Entry
Для каждого рынка сначала найти local partner. Без partner → отложить market entry.

### Правило 5: Maintenance Compliance = Differentiator
РД 25.964-90 automation — 0 конкурентов имеют digital журнал с HMAC-signature.
Gatekeeper + regulatory act — уникальное сочетание (GPS + AI + e-signature = юридически значимый акт).

---

## 🔗 Полезные ссылки

| Ресурс | Путь |
|--------|------|
| Architecture | `ARCHITECTURE.md` |
| UX Guidelines | `docs/ux/ux-guideline.md` |
| ADR Log | `docs/adr/` |
| API Docs | `backend/docs/api/` |
| Design System | `frontend/.storybook/` |
| CI/CD | `.github/workflows/` |
| Security Policy | `docs/iso27001/security-policy.md` |

---

## 📝 История изменений

**2026-06-28 — P1-PERF-BUNDLE: Bundle Size Optimization**
✅ Quick Wins: lazy-load jsPDF, react-joyride, react-datepicker (commit `b01ef28`)
✅ BUNDLE.1: FullCalendar → Schedule-X (commit `8eccc81`)
✅ BUNDLE.2: Recharts → Nivo (commit `5f78b99`)
✅ BUNDLE.3: xlsx/SheetJS → ExcelJS, MIT (commit `45cdd63`)
✅ 9 vendor chunks оптимизированы, license risk снят (GPL+Pro→MIT)

**2026-06-28 — Unified TODO создан**
✅ Merge текущего TODO с новой структурой задач
✅ P0-PDF, P0-CLEANUP, P1-OPT добавлены
✅ Все выполненные задачи сохранены в истории
✅ Success Metrics обновлены

---

## 📚 Appendix: Матрица нормативных документов

| Регион | Документ | Сфера | Периодичность ТО | Retention |
|--------|----------|-------|------------------|-----------|
| 🇧🇾 BY | СН 3.02.19-2025 | CCTV | 1/3/12 мес | 10 лет |
| 🇧🇾 BY | ТКП 472-2013 | ОПС | 1/6/12 мес | 10 лет |
| 🇷🇺 RU | РД 25.964-90 | АУПТ, ОПС | 1/6/12 мес | 10 лет |
| 🇷🇺 RU | РД 009-01-96 | Пожарная автоматика | 1/6/12 мес | 10 лет |
| 🇷🇺 RU | РД 009-02-96 | ТО и ППР | 1/6/12 мес | 10 лет |
| 🇷🇺 RU | ГОСТ Р 51558-2014 | CCTV | 1/3/12 мес | 10 лет |
| 🇰🇿 KZ | Приказ МЧС №55 | Пожарная автоматика | 1/3/12 мес | 10 лет |
| 🇰🇿 KZ | СТ РК ГОСТ Р 50776-2010 | Тревожная сигнализация | 1/3/12 мес | 10 лет |
| 🇹🇷 TR | KVKK №6698 | CCTV + ПДн | 3/12 мес | 5 лет |
| 🇹🇷 TR | TS EN 62676 | CCTV | 3/12 мес | 5 лет |
| 🇻🇳 VN | TCVN 11930:2017 | ИБ | 3/12 мес | 5 лет |
| 🇮🇩 ID | SNI 27001 | ISMS | 3/12 мес | 5 лет |
| 🇧🇷 BR | ABNT NBR series | CCTV + ОПС | 3/6/12 мес | 5 лет |
| 🇿🇦 ZA | SANS 10160-4 | Безопасность | 3/6/12 мес | 5 лет |
| 🇰🇪 KE | KS 2110-4/5:2009 | CCTV | 3/6/12 мес | 5 лет |

---

## 💼 Бизнес-ценность

| Возможность | TAM | Pricing Premium |
|-------------|-----|-----------------|
| Compliance Automation для КИИ РБ | $7M | +40% (enterprise tier) |
| МЧС лицензия для RU/KZ | $105M | +30% (certified vendor) |
| KVKK compliance для TR | $42M | +25% (legal protection) |
| LGPD/POPIA compliance | $115M | +20% (risk mitigation) |

**Competitive Moat**: РД 25.964-90 automation — 0 конкурентов имеют digital журнал с HMAC-signature.
Gatekeeper + regulatory act — уникальное сочетание (GPS + AI + e-signature = юридически значимый акт).

**Bottom line**: Добавление MaintenanceComplianceProfile превращает систему из "CCTV monitoring tool" в enterprise compliance platform с юридически значимой документацией, готовой для предъявления регуляторам. Суммарный TAM: **$461M**.
