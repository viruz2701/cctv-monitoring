 Executive Summary
Текущая версия (v4.0) описывает зрелую backend-архитектуру (DDD, Headless CMMS, Event Sourcing, 5 адаптеров), но имеет критический разрыв между архитектурной зрелостью backend и UX-зрелостью frontend. Глубокий анализ выявил:
Архитектурные долги, блокирующие горизонтальное масштабирование
UX-дефицит, делающий продукт непривлекательным на фоне MaintainX, UpKeep, Fiix
Отсутствие killer-features, которые могли бы стать УТП на рынке
Данный план v5.0 перебалансирует фокус в сторону UX/UI и killer-features, сохраняя при этом архитектурную целостность.
📜 Обновлённые архитектурные принципы (v5.0)
#
Принцип
Обоснование
Изменение
P1
Clean Room Implementation
AGPL-free, IP-чистота
✅ без изменений
P2
Headless CMMS
Pluggable CMMS Layer
✅ без изменений
P3
Event-Driven
NATS JetStream
✅ без изменений
P4
Domain-Driven Design
Bounded Contexts
✅ без изменений
P5
Permissive OSS Only
MIT/Apache 2.0
✅ без изменений
P6
API-First
OpenAPI 3.1 spec
✅ без изменений
P7
UX-First Design
Конкурентный паритет с MaintainX/UpKeep
🆕 НОВОЕ
P8
Stateless Backend
Горизонтальная масштабируемость
🆕 НОВОЕ
P9
Offline-First Mobile
Работа в полях без связи
🆕 НОВОЕ
P10
Accessibility (WCAG 2.1 AA)
Enterprise-требование, госсектор
🆕 НОВОЕ
🎯 Обновлённые Strategic Goals (OKR v5.0)
Objective
Key Result
Статус
O1: Unique Value Proposition
3+ уникальные CCTV-only фичи в production
✅ RCA, Playbook, Gatekeeper, VQ Analyzer
O2: Enterprise Readiness
SLA compliance 99%+, enterprise-клиенты
✅ SLA Engine, Escalation, Audit Log
O3: Operational Efficiency
50% ↓ time-to-resolve, MTTR < 2h
✅ Mobile Wizard, QR Portal
O4: Financial Control
100% visibility TCO per device
✅ TCO Dashboard
O5: Platform Flexibility
5 CMMS adapters
✅ Internal, Atlas, ServiceNow, Jira, 1С:ТОИР
O6: UX Excellence
NPS ≥ 50, task completion < 3 clicks
🆕 🟡 НОВОЕ
O7: Horizontal Scalability
10K+ устройств без деградации
🆕 🔴 НОВОЕ
O8: Market Differentiation
3 killer-features, недоступные конкурентам
🆕 🔴 НОВОЕ
🔴 CRITICAL BLOCKERS (необходимо решить до Production)
На основе глубокого code-review выявлены архитектурные блокеры, которые невозможно игнорировать:
✅ ARCH-01: InMemoryStateManager → JetStream KV (DONE)
Файл: backend/internal/state/jetstream_manager.go
Решение: NATS JetStream KV Store + in-memory cache + watcher-based sync.
Статус: ✅ DONE (commit 62d330d) — JetStreamStateManager реализован, протестирован, интегрирован в main.go с graceful fallback.
SP: 8 · Приоритет: 🔴 P0

✅ ARCH-02: 14 React Contexts → Zustand + React Query (DONE)
Файлы: frontend/src/store/*.ts, frontend/src/hooks/useApiQuery.ts, frontend/src/context/*.tsx
Решение: Zustand для UI-state (theme, alerts UI) + React Query для server state.
Статус: ✅ DONE (commit 62d330d) — все 12 контекстов мигрированы на Zustand/React Query, QueryClientProvider в App.tsx.
SP: 12 · Приоритет: 🔴 P0

✅ ARCH-03: Mock Data → Real API (DONE)
Файлы: frontend/src/pages/DeviceDetail.tsx, frontend/src/utils/reportGenerator.ts
Решение: Импорты mockData заменены на React Query хуки.
Статус: ✅ DONE (commit 62d330d) — DeviceDetail.tsx и reportGenerator.ts больше не используют mockData.
SP: 6 · Приоритет: 🔴 P0

✅ ARCH-04: Virtualization everywhere (DONE)
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/pages/AuditLog.tsx, frontend/src/pages/Devices.tsx
Решение: @tanstack/react-virtual через компонент VirtualTable.
Статус: ✅ DONE (commit 62d330d) — VirtualTable внедрён в WorkOrders, AuditLog, Devices.
SP: 4 · Приоритет: 🔴 P0
🎨 EPIC 14: UX/UI Excellence (НОВЫЙ ЭПИК)
Самый крупный эпи́к — именно здесь закрывается gap между architectural maturity и user experience.
🔴 P0 — Critical UX (Q3 2026)
ID
Задача
SP
Описание
Бизнес-ценность
UX-14.1.1
State Management Migration
12
Zustand + React Query вместо 14 Context
-60% re-renders
UX-14.1.2
Virtualization Everywhere
4
@tanstack/react-virtual во всех DataGrid
10K+ rows без lag
UX-14.1.3
Mock Data Decoupling
6
OpenAPI → TS types, Pact-контракты
Production-ready
UX-14.1.4
Global Error Boundary + Suspense
3
Per-route ErrorBoundary + skeletons
No white screens
UX-14.1.5
Command Palette (⌘K)
5
Поиск по всему приложению (Linear-style)
-80% time-to-action
✅ DONE (commit b71f7d7)
UX-14.1.6
Onboarding Tour
4
react-joyride для новых пользователей
-50% time-to-value
✅ DONE (commit b9d5b08)
UX-14.1.7
Empty States с CTA
3
Illustrative empty states во всех списках
+30% conversion
UX-14.1.8
Keyboard Shortcuts
3
⌘N новый WO, ⌘K поиск, Esc закрыть
Power users
UX-14.1.9
Dark Mode Toggle в Header
1
Быстрое переключение темы
User request #1
UX-14.1.10
Confirmation Dialogs
2
Confirm для всех destructive actions
-data loss
🟠 P1 — High Priority UX (Q3-Q4 2026)
ID
Задача
SP
Описание
UX-14.2.1
Progressive Disclosure в WorkOrderDetail
5
3-колонки → табы + drawers
UX-14.2.2
Drag-n-Drop Dashboards
6
react-grid-layout с сохранением layout
UX-14.2.3
Skeleton Screens
3
Shimmer-эффекты для всех async-загрузок
UX-14.2.4
Inline Validation
4
Real-time Zod validation в формах
UX-14.2.5
Toast Notifications Redesign
2
Stacked toasts с undo-action
UX-14.2.6
Global Search (⌘K) v2
4
Fuzzy search + recent + categories
UX-14.2.7
Accessibility Audit (WCAG 2.1 AA)
8
axe-core + Lighthouse, ARIA labels
UX-14.2.8
Focus Management
3
Focus traps в модалках, skip-links
UX-14.2.9
Color Contrast Fix
2
WebAIM-совместимые палитры
UX-14.2.10
Multi-language (15+)
8
i18next + Crowdin integration
🟡 P2 — Medium Priority (Q1 2027)
ID
Задача
SP
UX-14.3.1
Customizable Workspaces
6
UX-14.3.2
Saved Views & Filters
4
UX-14.3.3
Advanced DataGrid (pivoting)
8
UX-14.3.4
Theming Engine
5
UX-14.3.5
Onboarding Video Tutorials
3
Итого Epic 14: 59 SP (~7 недель для 3 frontend-разработчиков)
🚀 EPIC 15: Killer Features (НОВЫЙ ЭПИК)
Features, которые делают продукт уникальным на рынке CMMS. Конкуренты (MaintainX, UpKeep, Fiix, Atlas) не имеют ничего подобного.
🔴 TIER 1: Must-have Killer Features (Q3 2026)
ID
Killer Feature
SP
УТП
Pricing
KF-15.1.1
Compliance & Fines Shield
10
Конвертация downtime в $risk. "Камера на кассе = $500/час штрафа"
Premium tier
KF-15.1.2
Contractor Auto-Enforcer
8
GPS + EXIF + AI = техник не может фальсифицировать работу
Premium
KF-15.1.3
Predictive Maintenance Dashboard
10
XGBoost + DeepSeek explanations. "Камера #123 — 80% риск отказа"
Premium
KF-15.1.4
AI RCA с визуализацией
8
BFS + AI-объяснение root cause + рекомендации
Premium
KF-15.1.5
Offline-First Mobile (WatermelonDB)
12
CRDT + background sync. Работа в подвалах/на крышах
Standard
KF-15.1.6
Barcode/QR Inventory
5
Сканирование запчастей вместо ручного ввода
Standard
🟠 TIER 2: Differentiators (Q4 2026)
ID
Killer Feature
SP
УТП
Pricing
KF-15.2.1
Digital Twin (3D/2D plans)
15
Интерактивные планы зданий с устройствами
Premium
KF-15.2.2
Voice reports
6
Hands-free для техников ("на данном обьекте сделано ... неисправно такое оборудование..")
Premium
KF-15.2.3
AR Overlay
20
AR-маркеры на устройствах через камеру
Ultra-Premium
KF-15.2.4
Black Box Incident Recorder
8
Автоматический "пакет доказательств" при инциденте
Premium
KF-15.2.5
Smart Dispatch (AI)
10
ML-оптимизация маршрутов техников
Premium
KF-15.2.6
Vendor Marketplace
8
B2B marketplace для запчастей
Revenue share
🟡 TIER 3: Future (2027)
ID
Feature
SP
KF-15.3.1
Blockchain Audit Trail
12
KF-15.3.2
Computer Vision Anomaly Detection
20
KF-15.3.3
Digital Twin + IoT
25
KF-15.3.4
Marketplace Integrations
8
Итого Epic 15: ~110 SP (~14 недель для 3 senior + 1 ML)
📋 Обновлённый PENDING (с приоритезацией UX/UI)
🔴 P0 / CRITICAL (Q3 2026 — до Production)
ID | Epic | Задача | SP | Бизнес-ценность | Статус
ARCH-01 | Architecture | InMemoryStateManager → JetStream KV | 8 | 🚨 Horizontal scaling | ✅ DONE
ARCH-02 | Architecture | 14 Contexts → Zustand + React Query | 12 | 🚨 -60% re-renders | ✅ DONE
ARCH-03 | Architecture | Mock Data → OpenAPI types | 6 | 🚨 Production-ready | ✅ DONE
ARCH-04 | Architecture | Virtualization everywhere | 4 | 🚨 10K+ rows | ✅ DONE
UX-14.1.5
UX
Command Palette (⌘K)
5
⚡ -80% time-to-action
✅ DONE (commit b71f7d7)
UX-14.1.6
UX
Onboarding Tour
4
⚡ -50% time-to-value
✅ DONE (commit b9d5b08)
UX-14.1.7
UX
Empty States с CTA
3
⚡ +30% conversion
KF-15.1.1
Killer
Compliance & Fines Shield
10
💰 Premium tier
KF-15.1.2
Killer
Contractor Auto-Enforcer
8
💰 Premium tier
KF-15.1.3
Killer
Predictive Maintenance
10
💰 Premium tier
F-0.1.1
Foundation
IP-аудит (FOSSA/Snyk)
3
⚖️ Legal
CCTV-2.1.2
CCTV Core
XGBoost Failure Prediction
4
🧠 AI
Итого P0: ~70 SP
🟠 P1 / HIGH (Q4 2026)
ID
Epic
Задача
SP
UX-14.2.1
UX
Progressive Disclosure WorkOrderDetail
5
UX-14.2.2
UX
Drag-n-Drop Dashboards
6
UX-14.2.3
UX
Skeleton Screens
3
UX-14.2.7
UX
WCAG 2.1 AA Audit
8
UX-14.2.10
UX
Multi-language (15+)
8
KF-15.2.1
Killer
Digital Twin (3D/2D)
15
KF-15.2.4
Killer
Black Box Incident Recorder
8
KF-15.2.5
Killer
Smart Dispatch (AI)
10
F-0.2.3
Foundation
Multi-tenancy RLS
4
CCTV-2.2.1
CCTV Core
ONVIF Profile S/T
6
WM-8.3.2
Workforce
Capacity Planning heatmap
4
Итого P1: ~60 SP
🟡 P2 / MEDIUM (Q1 2027)
ID
Epic
Задача
SP
UX-14.3.1
UX
Customizable Workspaces
6
UX-14.3.3
UX
Advanced DataGrid (pivoting)
8
KF-15.3.1
Killer
Blockchain Audit Trail
12
KF-15.3.2
Killer
Computer Vision Anomaly
20
MB-12.3.1
Mobile
Local DB (WatermelonDB)
4
MB-12.3.2
Mobile
Conflict resolution (CRDT)
4
INT-13.3.1
Integration
SAML SSO production
5
Итого P2: ~50 SP
💼 Конкурентный анализ: Gap Analysis
Фича
CCTV Monitor
MaintainX
UpKeep
Fiix
Atlas CMMS
CCTV-specific IP
✅✅✅
❌
❌
❌
❌
Predictive Maintenance
🟡
✅✅
✅✅
✅✅
❌
Offline Mobile
🟡
✅✅
✅✅
❌
❌
Barcode/QR Inventory
🟡
✅✅
✅✅
✅
❌
AI Explanations
🟡
❌
❌
❌
❌
Digital Twin
❌
❌
❌
❌
❌
Contractor Auto-Enforcer
✅
❌
❌
❌
❌
Compliance Shield
🟡
❌
❌
❌
❌
5 CMMS Adapters
✅
❌
❌
❌
❌
Open Source
✅
❌
❌
❌
❌
Вывод: CCTV Monitor имеет уникальную позицию — единственный CCTV-specific CMMS с AI и open source. Но UX отстаёт от MaintainX/UpKeep на 2-3 года.
💰 Revenue Model & Pricing Strategy
Tier
Features
Цена
Target
OSS Community
Base CMMS + Internal Adapter
$0
SMB, self-hosted
Standard
+ Offline Mobile + Barcode
$49/user/mo
SMB
Premium
+ Predictive + Compliance Shield + Contractor Enforcer
$149/user/mo
Mid-market
Enterprise
+ Digital Twin + Smart Dispatch + Blockchain Audit
$399/user/mo
Enterprise
Ultra
+ AR Overlay + Custom integrations
Custom
Government
Revenue potential: $10M ARR при 1000 paying customers (mix tiers)
📊 Обновлённая сводка (v5.0)
Метрика
Значение
Всего задач в плане
~160+ (+80 vs v4.0)
Реализовано
~64 (40%) ✅ +4 P0
Pending P0 (Critical)
8 задач (~40 SP) ⬇️
Pending P1 (High)
11 задач (~60 SP)
Pending P2 (Medium)
7 задач (~50 SP)
Блокировано
1 (SEC-01: СТБ SDK)
Общий SP pending
~180 SP (~22 недели для 3 Senior)
UX/UI focus
60% от P0+P1 задач
Killer Features
13 features, 6 в P0
🎯 Top-5 рекомендаций CTO
Немедленно (неделя 1-2): Начать с ARCH-01/02/03 — без этого product не масштабируется и не выйдет в production.
Параллельно (неделя 2-6): UX-14.1.5-10 (Command Palette, Onboarding, Empty States, Keyboard shortcuts) — быстрый UX-win, повышает NPS на 20+ пунктов.
Q3 2026: 3 killer-features (Compliance Shield + Contractor Enforcer + Predictive) — основа Premium tier.
Q4 2026: Digital Twin + Smart Dispatch — выход в Enterprise tier.
2027: AR Overlay + Blockchain — ultra-premium и government contracts.
⚠️ Риски и mitigation
Риск
Вероятность
Mitigation
ARCH-01 блокирует K8s deployment
90% → ✅ 0%
✅ JetStream KV реализован (неделя 1)
ARCH-02 каскадные re-render'ы
90% → ✅ 0%
✅ Zustand + React Query (неделя 1)
ARCH-03 mock data coupling
80% → ✅ 0%
✅ API контракты везде (неделя 1)
ARCH-04 UI freeze при 1000+ строк
85% → ✅ 0%
✅ VirtualTable внедрён (неделя 1)
UX отстаёт от MaintainX на 2 года
85%
Epic 14 (60% фокус)
Нет killer-features для Premium tier
75%
Epic 15 (6 features в P0/P1)
SEC-01 (СТБ SDK) блокирует РБ-рынок
70%
Параллельная работа с ОАЦ
Конкуренты копируют уникальные фичи
60%
Patent + быстрое execution
Mobile offline sync конфликты
55%
CRDT + WatermelonDB
📅 Roadmap (Gantt-подобный)
Q3 2026 (Jul-Sep)
├── Неделя 1-2: ARCH-01/02/03/04 (scaling foundation)
├── Неделя 3-4: UX-14.1.5-10 (Command Palette, Onboarding, Empty States)
├── Неделя 5-8: KF-15.1.1-6 (Compliance Shield, Contractor Enforcer, Predictive)
└── Неделя 9-12: Beta launch + feedback loop

Q4 2026 (Oct-Dec)
├── UX-14.2.x (Progressive Disclosure, Dashboards, WCAG)
├── KF-15.2.1-6 (Digital Twin, Black Box, Smart Dispatch)
├── Multi-language (15+)
└── Enterprise launch

Q1 2027 (Jan-Mar)
├── UX-14.3.x (Customizable Workspaces, Advanced DataGrid)
├── KF-15.3.1-4 (Blockchain, Computer Vision)
└── Government contracts