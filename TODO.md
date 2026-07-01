# CCTV Health Monitor — TODO & Roadmap

## Правила для Roo (обязательно к исполнению)

### Перед задачей
- Прочитать связанный ADR из `docs/adr/` (ADR-027 до ADR-033)
- Проверить матрицу нормативных документов (`docs/ux/regulatory-matrix.md`)
- Убедиться, что Feature Flag зарегистрирован в `frontend/src/config/featureFlags.ts`
- Проверить, что задача не ломает существующие E2E тесты (`tests/e2e/`)

### Во время выполнения
- Атомарные коммиты с ID задачи: `UX-1.1: Unified Work Hub skeleton`
- **ЗАПРЕЩЕНО** удалять существующие роуты — только aliasing + redirect
- Все новые паттерны оборачивать в `<FeatureFlag name="..." />`
- Сохранять backward compatibility минимум 3 месяца (см. T1.5)
- Формат коммита: `feat(ux): description [UX-X.X]`

### После завершения
- Обновить метрику в Success Metrics (ниже)
- Добавить Storybook story (`frontend/src/components/**/*.stories.tsx`)
- Добавить E2E тест для critical path
- Обновить /help документацию
- Зафиксировать commit hash в этой таблице

### 🚫 Строгие запреты
- НЕ удалять `WorkOrders.tsx`, `Settings.tsx`, старые dashboard-страницы
- НЕ мигрировать глобальный state из Zustand в другой стор
- НЕ менять структуру API `/api/v1/*` (только добавлять новые эндпоинты)
- НЕ трогать backend migrations 040-056 (уже зафиксированы в проде)

---

## ✅ Выполненные задачи (история для референса)

<details>
<summary>📜 Показать завершённые задачи (Q2-Q3 2026) — 117 задач</summary>

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
✅ **Code Review Findings (61/61)**: Все P0/P1/P2 задачи закрыты
</details>

---

## 📊 Success Metrics (целевые показатели)

| Метрика | Current | Target | Δ | Ответственная задача |
|---------|---------|--------|---|---------------------|
| Time to find next WO | 8.5s | 2.3s | -73% | UX-1.1, UX-1.2 |
| Annual plan creation | 4h | 5min | -98% | UX-6.1 |
| Regulatory compliance | Manual | Auto-verified | 100% | UX-3.1 → UX-3.6 |
| TO document generation | 30min | 1 click | -99% | UX-3.5, UX-3.6 |
| Sidebar items | 30+ | 14 | -53% | UX-1.1, UX-1.6 |
| MTTA (alert ack) | 15min | 8min | -45% | UX-2.3 |
| Lighthouse Score | 87 | >95 | — | UX-7.* |
| Bundle size (gzip) | 1.62 MB | <800 KB | — | UX-7.2 |
| A11y violations | 0 critical | 0 | — | UX-8.* |

## 🚩 Feature Flag Registry

**Файл**: `frontend/src/config/featureFlags.ts`
**Middleware**: `FeatureFlagMiddleware.tsx` (уже существует)

| Flag Name | Default | Description | Задачи |
|-----------|---------|-------------|--------|
| `unified_work_hub_v2` | false | Единый хаб вместо 3 страниц WO | UX-1.2 |
| `sidebar_progressive_disclosure` | false | Сайдбар 14 пунктов с группировкой | UX-1.1, UX-1.6 |
| `command_palette_regulatory` | false | ⌘K с Regulatory Awareness | UX-5.1 |
| `to_hash_chain_signatures` | false | Крипто-цепочка ТО (СТБ 34.101.27) | UX-3.6 |
| `asset_tree_drilldown` | false | Site → Device → Component tree | UX-4.1 |
| `mobile_qr_lifecycle` | false | Onboarding → Maintenance flow | UX-4.2 |
| `print_template_builder` | false | Visual Editor для TO актов | UX-3.5 |
| `three_column_detail_layout` | false | Паттерн для WO/Device detail | UX-2.1 |
| `ai_copilot_to_journals` | false | DeepSeek для narrative (human-in-loop) | UX-3.4 |
| `role_based_home_pages` | false | Adaptive /home by role | UX-1.5 |

---

## 🛤️ Track 1: Navigation & Information Architecture (ADR-027, ADR-028)

**Цель**: Сократить когнитивную нагрузку на 53%, создать единую точку входа для работ.
**Длительность**: Week 1-2 (11 дней)

### UX-1.1 · Sidebar Progressive Disclosure 🔴 CRITICAL
**Effort**: 3d | **Feature Flag**: `sidebar_progressive_disclosure`
**Файлы**:
- `frontend/src/components/Sidebar.tsx` (существует, >500 строк)
- `frontend/src/hooks/useNavigation.ts`
- `frontend/src/config/sidebarGroups.ts` (новый)

**Проблема**: 30+ пунктов в сайдбаре → пользователь тратит до 30s на поиск

**Решение**: Группировка по 5 доменам:
- **Operations**: My Work, Team, Requests
- **Assets**: Sites, Devices, Spare Parts
- **Analytics**: Dashboards, BI, Custom Reports
- **Governance**: Compliance, Audit, Regulatory Calendar
- **Admin**: Users, Integrations, API

**Безопасный переход**:
- Старый sidebar рендерится если `featureFlags.sidebar_progressive_disclosure === false`
- Использовать `useFeatureFlag('sidebar_progressive_disclosure')`
- Collapse state хранится в `localStorage` (не Zustand!)

**Acceptance**:
- 14 видимых пунктов + 5 групп-аккордеонов
- Keyboard navigation (↑↓, Enter, Esc)
- ARIA `role="navigation"`, `aria-expanded` для групп
- Mobile: burger menu с теми же 14 пунктами
- E2E: `tests/e2e/sidebar-progressive.spec.ts`
- **Compliance**: ISO 9241-110 (Principle of minimal cognitive load)

### UX-1.2 · Unified Work Hub (ADR-028) 🔴 CRITICAL
**Effort**: 3d | **Feature Flag**: `unified_work_hub_v2`
**Файлы**:
- `frontend/src/pages/UnifiedWorkHub.tsx` (новый)
- `frontend/src/components/work-hub/WorkHubTabs.tsx` (новый)
- `frontend/src/components/work-hub/QuickFilters.tsx` (новый)
- `frontend/src/routes/workHubRoutes.ts` (новый, aliasing)

**Проблема**: 3 разные страницы (WorkOrders.tsx, Tickets.tsx, Requests.tsx) — дублирование логики

**Решение**: Одна страница с 3 табами:
```
┌─ Unified Work Hub ─────────────────────┐
│ [My Tasks] [Team] [Requests]           │
│ ┌─ Quick Filters ────────────────────┐ │
│ │ [Overdue] [Critical] [Unassigned]  │ │
│ └────────────────────────────────────┘ │
│ ┌─ DataGrid (WODataGrid.tsx) ────────┐ │
│ │ ... существующий компонент ...     │ │
│ └────────────────────────────────────┘ │
└────────────────────────────────────────┘
```

**Безопасный переход (Route Aliasing)**:
- `/work-orders` → redirect на `/hub?tab=tasks`
- `/tickets` → redirect на `/hub?tab=requests`
- Использовать `Navigate` из react-router с `replace={true}`
- Сохранить старые роуты 3 месяца (зафиксировать в `docs/migration/deprecated-routes.md`)

**State Management**:
- Активный таб → URL searchParams (`?tab=tasks&filter=critical`)
- Фильтры → React Query params (не Zustand!)
- Выделение строк → локальное состояние компонента

**Acceptance**:
- Один GET `/api/v1/work-orders?scope=mine|team|requests` вместо 3 запросов
- Quick Filters обновляют грид без рефреша страницы
- URL шаринг: скопировал ссылку → открыл тот же вид
- Bulk actions toolbar работает во всех табах
- E2E: `tests/e2e/unified-work-hub.spec.ts`
- **Compliance**: ISO 9241-110 (Principle of single entry point)

### UX-1.3 · Route Aliasing Middleware 🟡 HIGH
**Effort**: 1d
**Файлы**:
- `frontend/src/middleware/routeAliasing.ts` (новый)
- `frontend/src/config/routeAliases.ts` (новый)
- `frontend/src/main.tsx` (подключить middleware)

**Решение**: Декларативный маппинг старых URL на новые:
- `/work-orders` → `/hub?tab=tasks`
- `/tickets` → `/hub?tab=requests`
- `/devices/:id` → `/assets/devices/:id`
- `/settings` → `/admin/settings`

**Acceptance**:
- Redirect с HTTP 301 (не 302) для SEO
- `window.history.replaceState` вместо `pushState`
- Логирование в Sentry: `route_alias_used: {from, to, user_id}`
- **Compliance**: NIST SP 800-53 SI-12 (Information Retention)

### UX-1.4 · Breadcrumbs Enhancement 🟡 HIGH
**Effort**: 1d
**Файлы**:
- `frontend/src/components/Breadcrumbs.tsx` (существует)
- `frontend/src/hooks/useBreadcrumbs.ts` (существует)

**Решение**: Динамическая генерация из routeAliases + текущего контекста
- Support для deep links: Assets / Sites / Main Office / Cameras / Cam-001
- Clickable breadcrumbs с searchParams preservation

**Acceptance**:
- ARIA `nav aria-label="Breadcrumb"` + `ol` + `li`
- Последний элемент не кликабл (текущая страница)
- Keyboard: Tab → Tab → Enter для навигации

### UX-1.5 · Role-Based Home Pages (T1.5) 🟡 HIGH
**Effort**: 2d | **Feature Flag**: `role_based_home_pages`
**Файлы**:
- `frontend/src/pages/DashboardHub.tsx` (существует)
- `frontend/src/components/home/TechnicianHome.tsx` (новый)
- `frontend/src/components/home/ManagerHome.tsx` (новый)
- `frontend/src/components/home/AdminHome.tsx` (новый)

**Решение**: Один `/home` endpoint с адаптивным контентом по `user.role`:
- **Technician**: My Tasks (Today), Overdue, Quick QR scan
- **Manager**: Team Heatmap, SLA Breach risk, Approvals pending
- **Admin**: System Health, Compliance Status, Audit Alerts

**Acceptance**:
- 3 роли видят релевантный контент
- Один GET `/api/v1/home` endpoint с role в JWT
- Skeleton loader пока грузятся role-specific widgets
- E2E: `tests/e2e/role-based-home.spec.ts`
- **Compliance**: ISO 27001 A.9.2 (User access management)

### UX-1.6 · Sidebar Keyboard Navigation & A11y 🟡 HIGH
**Effort**: 1d
**Файлы**: `frontend/src/components/Sidebar.tsx`

**Acceptance**:
- `role="navigation"`, `aria-label="Main navigation"`
- Skip link: "Skip to main content"
- Focus visible ring (Tailwind `focus-visible:ring-2`)
- Arrow keys для навигации внутри группы
- Home/End для перехода к первому/последнему пункту
- CI Gate: axe-core в Playwright — 0 violations в sidebar
- **Compliance**: WCAG 2.1 AA, EN 301 549

---

## 🛠️ Track 2: Device Operations Center (ADR-029)

**Цель**: Создать unified operations UX, которого нет у MaintainX, Fiix, IBM Maximo.
**Длительность**: Week 3-4 (14 дней)

### UX-2.1 · Three-Column Layout Pattern 🟡 HIGH
**Effort**: 2d | **Feature Flag**: `three_column_detail_layout`
**Файлы**:
- `frontend/src/components/layouts/ThreeColumnLayout.tsx` (новый)
- `frontend/src/pages/WorkOrderDetail/WorkOrderDetail.tsx` (существует)
- `frontend/src/pages/DeviceDetail/DeviceDetail.tsx` (существует)

**Решение**: Универсальный 3-колоночный layout:
```
┌─ Left (240px) ─┬─ Center (flex) ─┬─ Right (320px) ─┐
│ Breadcrumbs    │ Tabs:           │ Metadata:       │
│ Status badge   │ - Overview      │ - SLA timer     │
│ Quick actions  │ - Live View     │ - Assignee      │
│                │ - History       │ - Audit Log     │
│                │ - Documents     │ - Actions       │
└────────────────┴─────────────────┴─────────────────┘
```

**Безопасный переход**:
- Сначала внедрить в DeviceDetail (менее критичная страница)
- После 2 недель stability → перенести в WorkOrderDetail
- Использовать `featureFlags.three_column_detail_layout`

### UX-2.2 · Device Live View Tab 🟡 HIGH
**Effort**: 3d
**Файлы**:
- `frontend/src/components/device/LiveViewTab.tsx` (новый)
- `frontend/src/services/webrtc.ts` (существует)
- `frontend/src/services/p2pApi.ts` (существует)

**Решение**: Вкладка "Live View" в DeviceDetail с:
- WebRTC/HLS стрим с камеры
- PTZ controls (если поддерживается)
- Snapshot button
- Recording trigger

### UX-2.3 · Alert Center with MTTA Optimization 🟡 HIGH
**Effort**: 3d
**Файлы**:
- `frontend/src/components/alerts/AlertCenter.tsx` (новый)
- `frontend/src/hooks/useAlerts.ts` (новый, на React Query)
- `frontend/src/services/websocket.ts` (существует)

**Решение**:
- Единый entry point для всех алертов
- Keyboard shortcut `A` для быстрого acknowledge
- Bulk ack для алертов одного типа
- Auto-ack для known false-positives

### UX-2.4 · Secure Tunnel Integration (T3.2) 🔴 CRITICAL
**Effort**: 3d
**Файлы**:
- `frontend/src/components/device/SecureTunnel.tsx` (новый)
- `frontend/src/services/p2pApi.ts` (существует)

**Решение**: В DeviceDetail вкладка "Tunnel" с:
- SSH/HTTPS proxy через backend
- One-time token с TTL 1h
- Audit log каждого подключения

**Compliance**: СТБ 34.101.27, 187-ФЗ (КИИ)

### UX-2.5 · Device History Timeline 🟢 MEDIUM
**Effort**: 2d
**Файлы**:
- `frontend/src/components/device/HistoryTimeline.tsx` (новый)
- `frontend/src/services/eventsApi.ts` (существует)

---

## 🔐 Track 3: TO Lifecycle & Compliance Automation

**Цель**: Получить главный competitive moat — compliance-автоматизацию с крипто-гарантиями.
**Длительность**: Week 5-6 (14 дней)
**Ключевые ADR**: ADR-030, ADR-032, ADR-033

### UX-3.1 · TO Journals with Regulatory Templates (T4.1) 🔴 CRITICAL
**Effort**: 3d
**Файлы**:
- `frontend/src/pages/TOJournals.tsx` (новый)
- `frontend/src/components/to-journals/JournalList.tsx` (новый)
- `frontend/src/components/to-journals/TemplateSelector.tsx` (новый)

**Решение**: Список журналов с фильтром по region_code, auto-fill применимых шаблонов, preview перед генерацией (PDF).

**Compliance**: СН 3.02.19-2025, РД 25.964-90

### UX-3.2 · Auto-fill при закрытии WorkOrder (T4.2) 🔴 CRITICAL
**Effort**: 3d | **Feature Flag**: `to_auto_generation`
**Файлы**:
- `frontend/src/pages/WorkOrderDetail/WorkOrderDetail.tsx` (существует)
- `frontend/src/components/work-orders/WOCompletionFlow.tsx` (новый)

**Решение**: Автоматическое создание записей во всех applicable TO-журналах при статусе "completed".

**Compliance**: ISO 27001 A.12.4.1

### UX-3.3 · TO Document Preview & Editing 🟡 HIGH
**Effort**: 2d
**Файлы**:
- `frontend/src/components/to-documents/DocumentPreview.tsx` (новый)
- `frontend/src/components/to-documents/FieldEditor.tsx` (новый)

### UX-3.4 · AI Copilot for TO Journals (Human-in-Loop) 🟡 HIGH
**Effort**: 3d | **Feature Flag**: `ai_copilot_to_journals`
**Файлы**:
- `frontend/src/components/to-journals/AICopilot.tsx` (новый)
- `frontend/src/services/aiApi.ts` (существует)

**Решение**: Паттерн Copilot — AI предлагает narrative, пользователь применяет (accept/reject/edit).

**Compliance**: EU AI Act (Human oversight requirement), ISO/IEC 42001

### UX-3.5 · Print Template Visual Editor (T4.3) 🔴 CRITICAL
**Effort**: 4d | **Feature Flag**: `print_template_builder`
**Файлы**:
- `frontend/src/pages/PrintTemplateBuilder.tsx` (новый)
- `frontend/src/components/print-builder/TemplateCanvas.tsx` (новый)
- `frontend/src/components/print-builder/PropertiesPanel.tsx` (новый)
- `frontend/src/components/print-builder/BlockLibrary.tsx` (новый)

**Решение**: Visual editor для TO актов (референс: Canva, Figma):
- Drag-n-drop блоки (text, table, image, signature, QR)
- Properties panel для каждого блока
- Preview в реальном времени

### UX-3.6 · Hash-Chain Digital Signatures (ADR-032, T4.5) 🔴 CRITICAL
**Effort**: 3d | **Feature Flag**: `to_hash_chain_signatures`
**Файлы**:
- `frontend/src/components/signatures/HashChainSignature.tsx` (новый)
- `frontend/src/components/signatures/ChainVerifier.tsx` (новый)
- `frontend/src/services/signatureApi.ts` (новый)

**Решение**: Каждая подпись ТО включает hash предыдущего ТО этого же устройства. Делает историю ТО криптографически защищенной.

**Compliance**: СТБ 34.101.27, 187-ФЗ (КИИ), EU eIDAS

### UX-3.7 · Regulatory Checklist Enforcement (T4.6) 🔴 CRITICAL
**Effort**: 2d
**Файлы**:
- `frontend/src/components/work-orders/RegulatoryGatekeeper.tsx` (новый)
- `frontend/src/pages/WorkOrderDetail/WOCompletionFlow.tsx`

---

## 📱 Track 4: Mobile QR Flow & Maintenance Calendar (ADR-031, ADR-033)

**Цель**: Решить проблему "первого впечатления" и автоматизировать рутинную работу.
**Длительность**: Week 7-8 (14 дней)

### UX-4.1 · Asset Tree Drill-down 🟡 HIGH
**Effort**: 4d | **Feature Flag**: `asset_tree_drilldown`
**Файлы**:
- `frontend/src/components/assets/AssetTree.tsx` (Storybook существует)
- `frontend/src/pages/AssetExplorer.tsx` (новый)
- `frontend/src/components/assets/TreeBreadcrumbs.tsx` (новый)

### UX-4.2 · QR Mobile Flow (ADR-031, T5.1) 🔴 CRITICAL
**Effort**: 4d | **Feature Flag**: `mobile_qr_lifecycle`
**Файлы**:
- `mobile/src/screens/QRScannerScreen.tsx` (существует)
- `mobile/src/screens/DeviceOnboarding.tsx` (новый)
- `mobile/src/services/qrLifecycle.ts` (новый)

**Решение**: Full lifecycle через QR:
- Onboarding: Scan QR → Auto-create device → Assign to site
- Maintenance: Scan QR → Open WO → Fill checklist → Sign → Generate TO
- Verification: Scan QR → View history → Verify hash-chain

**Compliance**: ISO 27001 A.8.2.3

### UX-4.3 · Maintenance Calendar UI 🟡 HIGH
**Effort**: 3d
**Файлы**:
- `frontend/src/pages/MaintenanceCalendar.tsx` (существует, рефактор)
- `frontend/src/components/calendar/ScheduleEvent.tsx` (новый)
- `frontend/src/components/calendar/ConflictDetector.tsx` (новый)

### UX-4.4 · Schedule Builder (T6.1) 🟡 HIGH
**Effort**: 4d
**Файлы**:
- `frontend/src/components/schedule/ScheduleBuilder.tsx` (новый)
- `frontend/src/components/schedule/RuleEditor.tsx` (новый)

---

## ⌨️ Track 5: Contextual Command Palette (ADR-029)

**Цель**: Создать domain-specific ⌘K, которого нет у Linear/Notion/Figma.
**Длительность**: Week 3 (3 дня, параллельно с Track 2)

### UX-5.1 · Command Palette with Regulatory Awareness 🔴 CRITICAL
**Effort**: 3d | **Feature Flag**: `command_palette_regulatory`
**Файлы**:
- `frontend/src/components/CommandPalette.tsx` (существует, рефактор)
- `frontend/src/services/commandIndex.ts` (новый)
- `frontend/src/hooks/useCommandPalette.ts` (существует)

**Решение**: Contextual actions на основе текущей страницы, роли пользователя, региона.

**Compliance**: ISO 9241-110 (Principle of efficiency)

---

## ⚡ Track 6: Performance & A11y Gates

**Цель**: Сохранить и улучшить текущие показатели (Lighthouse 87 → 95+).
**Длительность**: Постоянно, во время всех треков

### UX-7.1 · Bundle Size Optimization 🟡 HIGH
**Effort**: 2d
**Файлы**: `frontend/vite.config.ts`, `frontend/package.json`

### UX-7.2 · Image Optimization Pipeline 🟢 MEDIUM
**Effort**: 2d
**Файлы**: `frontend/src/components/Image.tsx`, `frontend/vite.config.ts`

### UX-8.1 · A11y Audit & CI Gate 🔴 CRITICAL
**Effort**: 2d
**Файлы**: `tests/e2e/a11y.spec.ts` (новый), `.github/workflows/a11y.yml` (новый)

### UX-8.2 · Keyboard Navigation Audit 🟡 HIGH
**Effort**: 2d

---

## 🧪 Cross-Cutting Concerns

| # | Концерн | Описание |
|---|---------|----------|
| CC-1 | Storybook-Driven Development | Любой новый компонент сначала пишется в Storybook. Coverage >80%. |
| CC-2 | Visual Regression Testing | Playwright snapshots. Diff threshold 0.1%. CI gate. |
| CC-3 | Feature Flag Middleware | Все новые паттерны оборачивать в `<FeatureFlag>`. Centralized management. |
| CC-4 | Error Boundary Strategy | Granular boundaries для каждого трека. Sentry + trace_id. |
| CC-5 | i18n Coverage | RU/EN/BE минимум. 100% coverage для новых компонентов. |

---

## 📅 Timeline & Milestones

| Week | Milestone | Deliverables | Gate |
|------|-----------|-------------|------|
| W1-2 | 🎯 Navigation & IA | UX-1.1 → UX-1.6 | Sidebar 30→14, Unified Hub live |
| W3-4 | 🔧 Device Operations | UX-2.1 → UX-2.5, UX-5.1 | 3-column layout, MTTA -45% |
| W5-6 | 🔐 TO Compliance | UX-3.1 → UX-3.7 | Hash-chain live, 100% regulatory |
| W7-8 | 📱 Mobile & Calendar | UX-4.1 → UX-4.4 | QR lifecycle, Schedule builder |
| W9+ | ⚡ Performance & Polish | UX-7.*, UX-8.* | Lighthouse 95+, 0 a11y violations |

## 🎯 Top-5 Must-Do First (высший ROI)

| # | Задача | Effort | Impact | Status |
|---|--------|--------|--------|--------|
| 1 | **UX-1.2** Unified Work Hub | 3d | -66% confusion для техников | ⏳ Ready to start |
| 2 | **UX-3.2** Auto-fill TO Journals | 3d | Киллер-фича для B2G | ⏳ Ready to start |
| 3 | **UX-4.2** Device Onboarding (QR) | 4d | Решает проблему "первого впечатления" | ⏳ Ready to start |
| 4 | **UX-2.4** Secure Tunnel | 3d | Критично для remote troubleshooting | ⏳ Ready to start |
| 5 | **UX-4.4** Schedule Builder | 4d | Автоматизация рутинной работы | ⏳ Ready to start |

---

## 📚 Связанные артефакты

- **ADR**: `docs/adr/ADR-027.md` → `ADR-033.md`
- **UX Guidelines**: `docs/ux/ux-guideline.md`
- **Regulatory Matrix**: `docs/ux/regulatory-matrix.md`
- **Architecture**: `ARCHITECTURE.md`
- **Security Policy**: `SECURITY.md`
- **Existing TODO (технический)**: `TODO.md` (61/61 ✅)
- **Plans**: `plans/`

---

## ✅ Definition of Done (для каждой задачи)

- [ ] Dark mode + Light mode работают
- [ ] Mobile responsive (< 768px)
- [ ] Keyboard navigation + ARIA labels
- [ ] i18n: RU/EN/BE
- [ ] Unit тесты coverage > 80%
- [ ] E2E тест для critical path
- [ ] Audit log запись для state-changing операций
- [ ] Error boundary с trace_id
- [ ] Sentry error tracking
- [ ] Documentation в /help
- [ ] Storybook story (для UI компонентов)
- [ ] Feature flag (если применимо)
- [ ] Route aliasing (если меняется URL structure)
- [ ] Backward compatibility (3 месяца для breaking changes)
- [ ] Lighthouse score не ухудшился
- [ ] A11y: 0 critical/serious violations
- [ ] Bundle size не увеличился >5%

---

## 🚨 Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Breaking existing bookmarks | Route aliasing + 3-month redirect |
| User confusion with new UI | Feature flags + gradual rollout (10% → 50% → 100%) |
| Performance regression | Lighthouse CI gate + bundle analyzer |
| Compliance violation | Regulatory checklist enforcement + audit log |
| Mobile offline sync conflicts | Differential sync (3-way merge, уже реализовано) |
| API breaking changes | Versioning (backend/internal/api/versioning.go уже есть) |

**Последнее обновление**: 2026-07-02
**Ответственный архитектор**: System Architect (UX/UI specialization)
**Следующий review**: 2026-07-09 (после Week 1-2)
