# TODO.md — CCTV Health Monitor
> Living document. Roo использует этот файл как основной roadmap.
> Обновлять после завершения каждой задачи: [ ] → [x] + дата.

**Последнее обновление:** 2026-06-26
**Общая готовность:** 99%

---

## 📋 Правила для Roo при работе с TODO

1. **Перед началом задачи:** Прочитать соответствующий раздел, проверить зависимости
2. **Во время работы:** Коммитить атомарно, в сообщении указывать ID задачи
3. **После завершения:** Отметить [x] + дата, проверить критерий приёмки, обновить метрику
4. **Если задача слишком большая:** Разбить на подзадачи с суффиксами (.1, .2, ...)
5. **Никогда не пропускать:** Критерий приёмки — если он не выполнен, задача не завершена
6. **Code review чеклист для каждой задачи:**
   - [ ] Dark mode работает
   - [ ] Accessibility (WCAG 2.1 AA)
   - [ ] i18n ключи добавлены в locales/
   - [ ] Error handling реализован
   - [ ] Unit/integration тесты написаны
   - [ ] Документация обновлена

Executive Summary
Ключевые findings из 3 reviews:
✅ Архитектура: World-class (DDD, Event-Driven, Headless CMMS) — 9.5/10
⚠️ UX/UI: Требует полировки (навигация, дашборды, forms) — 8/10
🔴 QA Gaps: Недостаточное покрытие e2e, нет a11y в CI, нет frontend monitoring — 7/10
⚠️ Performance: Bundle 2.8MB, LCP 2.5s, Lighthouse 87 — требует оптимизации
Приоритеты:
🔴 P0 (Blockers): Security gaps, data integrity, critical UX blockers
🟡 P1 (High Value): UX polish, performance optimization, testing coverage
🟢 P2 (Enterprise): Advanced features, integrations, AI/ML
🔵 P3 (Tech Debt): Refactoring, documentation, nice-to-have
🔴 P0 — CRITICAL (Q3 2026, до 2026-09-30)
P0-1: Security & Data Integrity
P0-1.1: Schema Registry Validation ✅ DONE
Status: [x] 2026-06-26
Effort: 2d
Impact: Data integrity, compliance
P0-1.2: NATS JetStream Mandatory ✅ DONE
Status: [x] 2026-06-26
Effort: 1d
Impact: Horizontal scaling, data consistency
P0-1.3: SLA Escalation Integration
Файлы: backend/internal/sla/engine.go, backend/internal/sla/worker.go, backend/internal/sla/notifier.go
Проблема: Таблицы sla_escalation_rules + sla_escalation_log существуют, но не интегрированы
Решение:
Добавить вызов w.notifier.NotifyBreach() в checkBreachedSLAs() при breach
Интегрирован multi-channel notifier (Telegram/SMS/Email) через SLABreachNotifier
Audit trail: escalation логируется в sla_escalation_log через CheckEscalation()
Fixed: isCriticalPriority() — case-insensitive сравнение приоритетов
Критерий приёмки:
✅ Escalation срабатывает при breach
✅ Уведомления отправляются через все каналы
✅ Audit log содержит все escalation events
✅ Unit тесты для escalation logic (3 новых теста)
Effort: 2d
Status: [x] 2026-06-26
P0-1.4: SMS Provider Implementation
Файлы: backend/internal/sms/rocketsms.go, backend/internal/sms/rocketsms_test.go, backend/internal/sla/notifier.go
Проблема: SMSProvider interface без реализации — заглушка
Решение:
✅ RocketSMSProvider реализован (СМС-центр РБ, compliance РБ)
✅ Rate limiting — не более 10 SMS в минуту на номер (anti-spam)
✅ Delivery tracking — счётчики sent/failed/rate_limited/cost
✅ Email fallback — при недоступности SMS отправляется email (в SLABreachNotifier)
✅ 15 unit тестов для RocketSMS provider (rate limit, delivery, config)
Критерий приёмки:
✅ SMS отправляются успешно (RocketSMS API)
✅ Fallback на email работает (SMS fail → email)
✅ Rate limiting для SMS (anti-spam, per phone number)
✅ Delivery tracking (Prometheus-ready метрики)
Effort: 3d
Status: [x] 2026-06-26
P0-1.5: SLABreachNotifier Fallback
Файлы: backend/internal/sla/notifier.go, backend/internal/sla/notifier_test.go
Проблема: Зависит от UserContactProvider — нет fallback при недоступности БД
Решение:
✅ Contact cache (in-memory sync.Map, TTL 5min) — stale-while-revalidate
✅ Fallback на default admin email при БД downtime
✅ Retry logic с exponential backoff (3 попытки, 100ms/200ms/400ms)
✅ 10 unit тестов: cache hit/expiry/stale-while-revalidate/clear, fallback, retry
Критерий приёмки:
✅ Notifier работает при БД downtime (stale cache + fallback email)
✅ Cached contacts обновляются каждые 5min (TTL)
✅ Alert отправляется даже без fresh data (fallback admin email)
Effort: 2d
Status: [x] 2026-06-26
P0-2: Critical UX Blockers
P0-2.1: AddDeviceModal Validation
Файлы: frontend/src/components/devices/AddDeviceModal.tsx
Проблема: Длинная форма, нет dynamic validation, нет индикации обязательных полей
Решение:
Интегрировать react-hook-form + Zod
Dynamic validation для IP, port, credentials
Conditional required fields based on connection type
Inline error messages под полями
Критерий приёмки:
Валидация в real-time (onChange/onBlur)
Required fields помечены звёздочкой
Error messages на всех языках (i18n)
Submit disabled при invalid form
Effort: 3d
Status: [ ]
P0-2.2: Breadcrumbs для Detail Pages
Файлы: frontend/src/components/ui/Breadcrumbs.tsx, frontend/src/pages/WorkOrderDetail.tsx, frontend/src/pages/DeviceDetail.tsx
Проблема: Отсутствие хлебных крошек в глубоких страницах
Решение:
Добавить Breadcrumbs component на все detail pages
Структура: Home > Section > Entity > Detail
Clickable breadcrumbs для navigation
Критерий приёмки:
Breadcrumbs на WorkOrderDetail, DeviceDetail, SiteDetail
Clickable navigation работает
Responsive на mobile
Accessibility: aria-label="Breadcrumb"
Effort: 2d
Status: [ ]
P0-2.3: View Mode Persistence
Файлы: frontend/src/pages/WorkOrders.tsx
Проблема: Переключение table/kanban/calendar не сохраняется
Решение:
Сохранять в URL query param: ?view=kanban
Сохранять в localStorage для persistence
Restore on page load
Критерий приёмки:
View mode сохраняется в URL
Bookmark с view mode работает
localStorage fallback
Default view per user role
Effort: 1d
Status: [ ]
P0-2.4: Kanban Feedback & Animation
Файлы: frontend/src/pages/WorkOrders.tsx
Проблема: При drag&drop нет анимации и toast feedback
Решение:
Добавить toast: "WO #123 moved to In Progress"
Undo button в toast (5 sec)
Smooth animation при drop
Haptic feedback на mobile
Критерий приёмки:
Toast появляется при status change
Undo работает (revert status)
Animation smooth (60fps)
Accessibility: aria-live="polite"
Effort: 2d
Status: [ ]
P0-3: Mobile Critical Fixes
P0-3.1: Conflict Resolution UI
Файлы: mobile/src/components/ConflictResolutionModal.tsx
Проблема: Только LWW (last-write-wins), нет UI для manual resolution
Решение:
Modal с diff-view: "Local vs Server"
Кнопки: Keep Local, Keep Server, Merge
Visual highlighting of changes
Timestamp comparison
Критерий приёмки:
Modal появляется при conflict
Diff view показывает изменения
Merge для text fields
Conflict logged в telemetry
Effort: 3d
Status: [ ]
P0-3.2: Background Sync Integration
Файлы: mobile/src/hooks/useBackgroundSync.ts, mobile/src/services/syncService.ts
Проблема: useBackgroundSync есть, но не интегрирован с SyncService
Решение:
Интегрировать с expo-background-fetch
Sync every 15min when app closed
Sync on network reconnect
Queue management UI
Критерий приёмки:
Background sync работает
Queue count отображается
Manual sync button
Sync status indicator
Effort: 3d
Status: [ ]
P0-3.3: Offline Map Tile Caching
Файлы: mobile/src/hooks/useOfflineMap.ts
Проблема: useOfflineMap есть, но нет кэширования тайлов
Решение:
Кэшировать map tiles в SQLite
Preload tiles для assigned sites
Offline map availability indicator
Tile expiration (30 days)
Критерий приёмки:
Tiles кэшируются локально
Map работает offline
Cache size management (clear old)
Preload on site assignment
Effort: 4d
Status: [ ]
🟡 P1 — HIGH VALUE (Q4 2026, до 2026-12-31)
P1-1: UX Polish & Consistency
P1-1.1: Dashboard Unification
Файлы: frontend/src/pages/DashboardHub.tsx, frontend/src/pages/ManagerDashboard.tsx, frontend/src/pages/TechnicianDashboard.tsx
Проблема: Дублирование: Dashboard, ManagerDashboard, TechnicianDashboard, ExecutiveDashboard
Решение:
Единая страница /dashboard с role-based widgets
Widget visibility per role
Drag-and-drop customization (уже есть DragDropDashboard)
Saved layouts per user
Критерий приёмки:
Одна dashboard страница
Role-based default widgets
Customization сохраняется
Migration script для old URLs
Effort: 4d
Status: [ ]
P1-1.2: Calendar Date Mode Toggle
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/components/work-orders/WorkOrderCalendar.tsx
Проблема: Calendar показывает только deadline, не creation date
Решение:
Toggle: "Show by deadline / Show by creation date"
Color coding: deadline (red), creation (blue)
Dual date display in event details
Критерий приёмки:
Toggle работает
Calendar updates without reload
Preference сохраняется
Legend объясняет colors
Effort: 2d
Status: [ ]
P1-1.3: RCA Widget в Device Overview
Файлы: frontend/src/pages/DeviceDetail.tsx
Проблема: RCA граф скрыт в отдельной вкладке
Решение:
Вынести RCA summary в Overview tab
Show: root cause, blast radius, affected devices
Click to expand full RCA graph
"No RCA available" state
Критерий приёмки:
RCA summary виден сразу
Expandable details
Real-time updates via WebSocket
Export RCA as PDF
Effort: 3d
Status: [ ]
P1-1.4: Search Unification
Файлы: frontend/src/components/layout/Header.tsx, frontend/src/components/CommandPalette.tsx
Проблема: Дублирование: Header Search + Command Palette
Решение:
Header search открывает Command Palette
Unified search results
Recent searches shared
Keyboard shortcut: / or ⌘K
Критерий приёмки:
Один search component
Results консистентны
Recent searches shared
Accessibility: focus management
Effort: 2d
Status: [ ]
P1-1.5: Saved Filters
Файлы: frontend/src/components/ui/DataGrid.tsx, frontend/src/store/savedViewsStore.ts
Проблема: Фильтры в DataGrid не сохраняются между сессиями
Решение:
Save filter presets (как SavedViews)
Named filters: "Critical Overdue", "My Team"
Share filters with team
Default filters per role
Критерий приёмки:
Save/load filters работает
Filters в dropdown menu
Share via URL
Default filters per role
Effort: 3d
Status: [ ]
P1-1.6: Bulk Operations Progress
Файлы: frontend/src/components/ui/BulkProgressModal.tsx
Проблема: При bulk-операциях нет прогресс-бара
Решение:
Modal с progress bar
Real-time status: "Processing 15/100..."
Cancel button
Error summary при failures
Retry failed items
Критерий приёмки:
Progress bar отображается
Real-time updates (WebSocket)
Cancel работает
Error details с retry
Effort: 3d
Status: [ ]
P1-1.7: Contextual Tooltips
Файлы: frontend/src/components/ui/InfoTooltip.tsx
Проблема: Сложные термины (MTBF, SLA, Gatekeeper) не объяснены
Решение:
Info icon (?) рядом с терминами
Tooltip с definition + link to docs
Glossary page /help/glossary
i18n для всех tooltips
Критерий приёмки:
Tooltips на всех сложных терминах
Glossary page доступна
Tooltips accessible (keyboard)
Mobile-friendly (tap to show)
Effort: 2d
Status: [ ]
P1-2: Performance Optimization
P1-2.1: Bundle Size Reduction
Файлы: frontend/vite.config.ts, frontend/src/App.tsx
Проблема: Bundle 2.8MB (цель <2MB)
Решение:
Lazy load FullCalendar, Recharts, XLSX
Tree-shaking для lucide-react (уже есть)
Dynamic import для heavy components
Analyze с rollup-plugin-visualizer
Критерий приёмки:
Bundle <2MB
Lighthouse Performance >90
Initial load <3s
Route-based code splitting
Effort: 3d
Status: [ ]
P1-2.2: Image Lazy Loading в DataGrid
Файлы: frontend/src/components/ui/DataGrid.tsx
Проблема: Изображения в таблицах загружаются сразу
Решение:
loading="lazy" для всех images
Placeholder (blur hash)
Intersection Observer для off-screen images
Thumbnail generation на backend
Критерий приёмки:
Images lazy-loaded
Placeholder отображается
No layout shift
WebP format (smaller size)
Effort: 2d
Status: [ ]
P1-2.3: React Query Optimization
Файлы: frontend/src/hooks/useApiQuery.ts
Проблема: staleTime и gcTime не оптимизированы
Решение:
Reference data (sites, users): staleTime: 5min, gcTime: 1h
Lists (devices, WOs): staleTime: 30s, gcTime: 5min
keepPreviousData для pagination
Prefetch on hover (уже есть)
Критерий приёмки:
Optimized cache strategy
No unnecessary refetches
Smooth pagination
Network tab показывает fewer requests
Effort: 1d
Status: [ ]
P1-2.4: Skeleton на всех страницах
Файлы: frontend/src/components/layout/SkeletonPage.tsx
Проблема: Некоторые страницы (AdvancedAnalytics) загружаются без skeleton
Решение:
Добавить skeleton на все pages с data fetching
Skeleton per component (table, chart, cards)
Shimmer animation
Progressive loading (skeleton → partial → full)
Критерий приёмки:
Skeleton на всех pages
Consistent design
No layout shift
Accessibility: aria-busy="true"
Effort: 2d
Status: [ ]
P1-3: Testing & Quality Assurance
P1-3.1: E2E Test Expansion
Файлы: frontend/tests/e2e/*.spec.ts
Проблема: Playwright покрывает только 4 сценария
Решение:
Добавить critical user journeys:
Create WO с checklist
Complete WO с photo upload
Assign technician
Export report
Register P2P device
View RCA graph
Mock API для isolation
Parallel execution
Критерий приёмки:
15+ e2e тестов
Coverage >80% critical paths
CI integration
Test reports в PR
Effort: 5d
Status: [ ]
P1-3.2: Mobile E2E Tests
Файлы: mobile/e2e/*.spec.ts
Проблема: Mobile тесты практически отсутствуют
Решение:
Настроить Detox или Maestro
Тесты для offline scenarios
Sync conflict resolution
Photo upload + Gatekeeper
Push notifications
Критерий приёмки:
Detox/Maestro настроен
10+ e2e тестов
Offline scenarios covered
CI integration
Effort: 5d
Status: [ ]
P1-3.3: Accessibility Testing в CI
Файлы: frontend/tests/a11y/*.spec.ts, playwright.config.ts
Проблема: Нет автоматических a11y проверок
Решение:
Интегрировать @axe-core/playwright
Проверка всех pages
Threshold: 0 critical violations
Report в PR comments
Критерий приёмки:
axe-core интегрирован
Все pages проверяются
CI fails при violations
Accessibility report в PR
Effort: 2d
Status: [ ]
P1-3.4: Frontend Error Monitoring (Sentry)
Файлы: frontend/src/lib/sentry.ts, mobile/src/lib/sentry.ts
Проблема: Нет frontend monitoring ошибок
Решение:
Интегрировать Sentry SDK
Capture unhandled exceptions
User context (role, tenant)
Source maps для debugging
Alerting на critical errors
Критерий приёмки:
Sentry интегрирован (web + mobile)
Errors captured автоматически
Source maps uploaded
Alerting настроен
Effort: 2d
Status: [ ]
P1-3.5: Lighthouse CI
Файлы: .github/workflows/lighthouse.yml, lighthouserc.js
Проблема: Нет автоматических performance тестов
Решение:
Lighthouse CI в PR checks
Thresholds: Performance >90, A11y >95
Regression detection
Historical trends
Критерий приёмки:
Lighthouse CI настроен
PR checks работают
Thresholds enforced
Reports в PR comments
Effort: 2d
Status: [ ]
P1-3.6: Visual Regression Testing
Файлы: frontend/tests/visual/*.spec.ts
Проблема: Нет визуальных регрессионных тестов
Решение:
Chromatic для Storybook
Percy для e2e screenshots
Baseline screenshots
Diff detection
Критерий приёмки:
Chromatic/Percy настроен
Baseline screenshots созданы
CI integration
Review workflow
Effort: 3d
Status: [ ]
P1-4: Backend Quality
P1-4.1: ActionExecutor Unit Tests
Файлы: backend/internal/workflow/action_executor_test.go
Проблема: Нет unit-тестов для ActionExecutor (только интеграционные)
Решение:
Table-driven tests для всех action types
Mock dependencies
Edge cases coverage
Benchmark tests
Критерий приёмки:
Unit тесты написаны
Coverage >90%
Edge cases covered
Benchmarks added
Effort: 2d
Status: [ ]
P1-4.2: PlaybookRegistry Versioning
Файлы: backend/internal/playbook/registry.go
Проблема: Не поддерживает versioning (нет hot reload)
Решение:
Version field в playbook schema
Hot reload без restart
Rollback capability
Version history
Критерий приёмки:
Versioning работает
Hot reload без downtime
Rollback работает
Migration script для old playbooks
Effort: 3d
Status: [ ]
P1-4.3: CMMSIntegrator Context Timeouts
Файлы: backend/internal/cmms/integrator.go
Проблема: Context передаётся, но не проверяется для таймаутов
Решение:
Check ctx.Done() в long operations
Configurable timeouts per adapter
Graceful cancellation
Timeout metrics
Критерий приёмки:
Context cancellation работает
Timeouts configurable
No goroutine leaks
Metrics для timeout events
Effort: 2d
Status: [ ]
P1-4.4: RCA Graph Auto-Update
Файлы: backend/internal/rca/graph_builder.go
Проблема: Граф не обновляется автоматически при добавлении/удалении устройств
Решение:
Event listener для device changes
Incremental graph updates
Cache invalidation
WebSocket notification
Критерий приёмки:
Graph updates автоматически
No full rebuild required
Cache invalidation работает
Real-time updates в UI
Effort: 3d
Status: [ ]
P1-4.5: RCA BuildFromState Accuracy
Файлы: backend/internal/rca/graph_builder.go
Проблема: Эвристика по IP-подсетям даёт ложные связи
Решение:
Use explicit parent-child relationships
Manual topology configuration
ML-based inference (future)
Validation rules
Критерий приёмки:
No false positive connections
Explicit relationships used
Validation warnings
Manual override capability
Effort: 3d
Status: [ ]
P1-5: Architecture Improvements
P1-5.1: Context Migration to Zustand
Файлы: frontend/src/context/*.tsx
Проблема: 14 React Context провайдеров — performance bottleneck
Решение:
Мигрировать DevicesSitesContext, MaintenanceContext на React Query
Мигрировать AlertsContext на Zustand
Оставить только auth, theme contexts
Benchmark before/after
Критерий приёмки:
Context count: 14 → 4
No performance regression
All features work
Benchmark показывает improvement
Effort: 4d
Status: [ ]
P1-5.2: API Routes Organization
Файлы: backend/internal/api/*.go
Проблема: 70+ файлов в internal/api/ — сложно navigate
Решение:
Группировать по доменам:
api/work_orders/
api/devices/
api/auth/
api/cmms/
Domain-based routing
Shared middleware
Критерий приёмки:
Routes organized by domain
No breaking changes
Documentation updated
Tests pass
Effort: 3d
Status: [ ]
P1-5.3: OpenAPI TypeScript Generation
Файлы: backend/docs/openapi.yaml, frontend/src/types/api.ts
Проблема: openapi.go есть, но не используется для генерации TypeScript
Решение:
Настроить oapi-codegen или openapi-typescript
Auto-generate types из OpenAPI spec
Type-safe API client
CI validation
Критерий приёмки:
Types генерируются автоматически
Type-safe API calls
CI validates spec
No manual type definitions
Effort: 3d
Status: [ ]
P1-5.4: Replace http.Error с respondError
Файлы: backend/internal/api/**/*.go
Проблема: В некоторых handlers используется http.Error вместо respondError
Решение:
Глобальный replace (уже есть скрипт replace_http_error.py)
Code review для всех handlers
Linter rule для предотвращения
Documentation
Критерий приёмки:
Все handlers используют respondError
Linter rule добавлен
Consistent error format
Trace ID в всех errors
Effort: 2d
Status: [ ]
P1-5.5: Trace ID Propagation
Файлы: backend/internal/**/*.go
Проблема: trace_id propagation только в api слое
Решение:
Inject trace_id в context
Propagate через все service layers
Include в logs, metrics, events
OpenTelemetry integration
Критерий приёмки:
trace_id в всех logs
Distributed tracing работает
Jaeger/Zipkin integration
Performance impact <5%
Effort: 3d
Status: [ ]
🟢 P2 — ENTERPRISE FEATURES (Q1 2027, до 2027-03-31)
P2-1: Advanced Analytics & AI
P2-1.1: Real ML Model Integration
Файлы: backend/analytics/predict.py, backend/internal/ml/prediction_service.go
Проблема: XGBoost обучается на синтетических данных
Решение:
Train на production data из TimescaleDB
Features: offline_ratio, error_count, reboot_count, age_days
Publish predictions через NATS
Confidence score
Критерий приёмки:
Model trained на real data
Predictions >75% accuracy
NATS integration
A/B testing framework
Effort: 5d
Status: [ ]
P2-1.2: AI Assistant Chat
Файлы: frontend/src/components/ai/AIAssistantPanel.tsx
Проблема: Нет контекстных подсказок
Решение:
DeepSeek integration
Context-aware recommendations
RCA suggestions
Natural language queries
Критерий приёмки:
Chat panel доступен
Context-aware responses
Response time <2s
Feedback mechanism
Effort: 4d
Status: [ ]
P2-2: Workflow & Automation
P2-2.1: Workflow Builder UI
Файлы: frontend/src/components/workflow/WorkflowBuilder.tsx
Проблема: workflow/engine.go есть, но нет UI
Решение:
React Flow для drag&drop
CEL conditions editor
Workflow testing mode
Version control
Критерий приёмки:
Drag&drop работает
CEL editor с highlighting
Test mode с mock data
Version history
Effort: 5d
Status: [ ]
P2-2.2: Resource Planning Calendar
Файлы: frontend/src/pages/TechnicianWeek.tsx
Проблема: Нет календаря загрузки техников
Решение:
Week view с technician rows
Drag-and-drop WO assignment
Conflict detection
Availability indicators
Критерий приёмки:
Week view отображает загрузку
Drag&drop для reassignment
Conflict warnings
Print-friendly view
Effort: 4d
Status: [ ]
P2-3: Integration Ecosystem
P2-3.1: Webhook Builder UI
Файлы: frontend/src/components/webhooks/WebhookBuilder.tsx
Проблема: webhook/verify.go есть, но нет visual builder
Решение:
Event type selector
Payload preview
Test button
Delivery logs
Критерий приёмки:
Builder создает webhooks
Payload preview работает
Test mode sends mock event
Delivery logs для debugging
Effort: 3d
Status: [ ]
P2-3.2: OAuth2 для External Adapters
Файлы: backend/internal/cmms/servicenow/client.go, backend/internal/cmms/jira/client.go
Проблема: ServiceNow/Jira используют basic auth
Решение:
OAuth2 flow implementation
Token refresh logic
Secure token storage
Fallback to basic auth
Критерий приёмки:
OAuth2 flow работает
Token auto-refresh
Secure storage (encrypted)
Fallback работает
Effort: 3d
Status: [ ]
🔵 P3 — TECHNICAL DEBT (Q2 2027, до 2027-06-30)
P3-1: Security & Compliance
P3-1.1: belt-GCM Migration (СТБ 34.101.31)
Файлы: backend/internal/crypto/aes.go, backend/internal/crypto/belt.go
Проблема: AES-256-GCM, для КИИ РБ требуется belt-GCM
Решение:
Мигрировать на github.com/bp2012/crypto/belt
Backward compatibility
Migration script
Критерий приёмки:
belt-GCM для new data
Migration script работает
Performance benchmarks
Security audit passed
Effort: 4d
Status: [ ]
P3-1.2: JWT bign-curve256v1 Migration
Файлы: backend/internal/auth/jwt.go
Проблема: JWT HS256, для РБ требуется bign-curve256v1
Решение:
Мигрировать на СТБ bign-curve
Backward compatibility
Token rotation
Критерий приёмки:
bign-curve используется
Old tokens валидны до expiry
Migration seamless
Compliance verified
Effort: 3d
Status: [ ]
P3-1.3: Mobile Certificate Pinning
Файлы: mobile/src/lib/api.ts
Проблема: Нет certificate pinning
Решение:
Pin server certificates
Certificate rotation support
Fallback on cert mismatch
Критерий приёмки:
Certificate pinning работает
MITM protection
Certificate rotation
Security audit passed
Effort: 2d
Status: [ ]
P3-2: Developer Experience
P3-2.1: Storybook Expansion
Файлы: frontend/src/components/**/*.stories.tsx
Проблема: Storybook только для 8 из 56 компонентов
Решение:
Stories для всех UI components
Interactive examples
Accessibility notes
Design tokens documentation
Критерий приёмки:
50+ stories
All atoms/molecules covered
Interactive controls
A11y guidelines
Effort: 5d
Status: [ ]
P3-2.2: Onboarding Tour для всех ролей
Файлы: frontend/src/components/OnboardingTour.tsx
Проблема: OnboardingTour только для админов
Решение:
Role-specific tours
Technician: WO creation, QR scanner
Manager: Dashboard, reports
Admin: Settings, integrations
Критерий приёмки:
Tours для всех ролей
Contextual steps
Skip option
Completion tracking
Effort: 3d
Status: [ ]
P3-2.3: Help System & Glossary
Файлы: frontend/src/pages/Help.tsx, frontend/src/pages/Glossary.tsx
Проблема: Нет справочной системы
Решение:
/help page с FAQ
/glossary с terms
Search functionality
Video tutorials
Критерий приёмки:
Help page доступна
Glossary с 50+ terms
Search работает
i18n для всех content
Effort: 3d
Status: [ ]
P3-3: Nice-to-Have
P3-3.1: Real-time Collaboration
Файлы: frontend/src/pages/WorkOrderDetail.tsx, backend/internal/ws/hub.go
Проблема: Нет WebSocket presence indicators
Решение:
"Ivan is editing this WO" indicator
Cursor sharing
Conflict warnings
Критерий приёмки:
Presence indicators работают
Real-time updates
Conflict warnings
Graceful degradation
Effort: 4d
Status: [ ]
P3-3.2: White-label Theming
Файлы: frontend/src/store/themeStore.ts
Проблема: Нет white-label для enterprise
Решение:
Custom logo, colors
Per-tenant themes
CSS variables
Критерий приёмки:
Custom branding работает
Per-tenant themes
No code changes required
Preview mode
Effort: 3d
Status: [ ]
P3-3.3: Edge Agent SL-4 Security
Файлы: backend/internal/edge/agent.go
Проблема: Edge Agent требует SL-4 (secure boot, mTLS, tamper detection)
Решение:
Secure boot verification
mTLS для всех communications
Tamper detection
Hardware security module
Критерий приёмки:
Secure boot работает
mTLS enforced
Tamper detection active
Security certification
Effort: 5d
Status: [ ]
📊 Success Metrics
Metric
Current
Target (Q4 2026)
Measurement
Bundle Size
2.8MB
<2MB
rollup-plugin-visualizer
FCP
1.8s
<1.5s
Lighthouse
LCP
2.5s
<2.0s
Lighthouse
Lighthouse Score
87
>95
Lighthouse CI
E2E Coverage
4 scenarios
15+ scenarios
Playwright
Mobile E2E
0%
10+ scenarios
Detox/Maestro
A11y Violations
Unknown
0 critical
axe-core
Context Count
14
4
Code analysis
API Files
70+ in root
Organized by domain
Structure review
Test Coverage (React)
75%
>80%
Vitest
SLA Breach Detection
Manual
Automatic
SLA logs

Перед началом задачи: Прочитать соответствующий раздел, проверить зависимости
Во время работы: Коммитить атомарно, в сообщении указывать ID задачи (например, P0-1.3: SLA Escalation Integration)
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
🔗 Полезные ссылки
Architecture: ARCHITECTURE.md
UX Guidelines: docs/ux/ux-guideline.md
ADR Log: docs/adr/
API Docs: backend/docs/api/
Design System: frontend/.storybook/
CI/CD: .github/workflows/
Code Reviews: plans/code-review-*.md
Последний коммит: HEAD
Branch: main
Next Review: 2026-07-03
📈 История изменений
2026-06-26 — Unified TODO Creation
Объединены findings из 3 code reviews
Создана единая структура P0-P3
Добавлены критерии приёмки для всех задач
Установлены метрики успеха
Определён roadmap на Q3 2026 - Q2 2027
Impact: Готовность 96% → Target 99% за 4 квартала