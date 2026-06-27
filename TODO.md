Перед началом задачи: Прочитать соответствующий раздел, проверить зависимости
Во время работы: Коммитить атомарно, в сообщении указывать ID задачи (например, P0-CE.1: ComplianceProfile Abstraction)
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
🔴 P0 — CRITICAL (Q3 2026, до 2026-09-30)
P0-CE: Regional Compliance Engine (Стратегический приоритет)
P0-CE.1: ComplianceProfile Abstraction Layer
Файлы: backend/internal/compliance/profile.go, backend/internal/compliance/registry.go, backend/internal/compliance/providers.go
Проблема: Криптография и security policies захардкожены под РБ, блокирует выход на другие рынки
Решение:
Создать ComplianceProfile интерфейс с политиками: crypto, hash, signature, password, data_residency, retention, audit, session
Provider Registry для runtime-загрузки провайдеров по региону
Inject через DI container на основе tenant/instance config
3 baseline профиля: BY (СТБ), EU (GDPR), INTL (ISO 27001)
Критерий приёмки:
✅ ComplianceProfile interface с 8 policy методами
✅ Provider Registry с thread-safe registration
✅ Startup fails при отсутствии required provider для выбранного региона
✅ Unit тесты для profile switching (coverage >90%)
✅ Integration test: BY profile → belt-GCM, EU profile → AES-256-GCM
Effort: 5d
Status: [x] (commit be37da2)
Dependencies: P0-1.1 (Schema Registry)
P0-CE.2: Regional Crypto Providers
Файлы: backend/internal/crypto/providers/belt.go, providers/aes.go, providers/gost.go, providers/sm.go
Проблема: Только AES-256-GCM, нет belt-GCM для РБ, ГОСТ для РФ, SM для Китая
Решение:
belt-GCM провайдер (github.com/bp2012/crypto) для BY
AES-256-GCM провайдер для EU/US/INTL
GOST 28147-89 stub для RU (full impl в P2-RU)
SM4 stub для CN (full impl в P2-CN)
Automatic provider selection via ComplianceProfile
Benchmark: belt vs AES vs GOST vs SM
Критерий приёмки:
✅ belt-GCM работает для BY profile (stub через AES)
✅ AES-256-GCM fallback для INTL
✅ Benchmark: overhead <2x для всех providers (AES baseline)
✅ Graceful error если provider недоступен
✅ Unit тесты для encrypt/decrypt round-trip
✅ Performance test: 1000 ops/sec (benchmarks)
Effort: 4d
Status: [x] (commit e58cc01)
P0-CE.3: Hash & Signature Providers
Файлы: backend/internal/crypto/providers/hash_*.go, providers/signature_*.go
Проблема: Audit log HMAC, JWT signature, password hashing не адаптированы под регионы
Решение:
bash-256 provider (BY audit log HMAC)
bign-curve256v1 provider (BY JWT)
SHA-256 + ES256 provider (EU/US)
Стрибог-256 provider (RU)
Argon2id password hashing (EU) vs belt-hash+bcrypt (BY)
Runtime selection per profile
Password migration при смене profile (read-old, write-new)
Критерий приёмки:
✅ JWT sign/verify работает для обоих алгоритмов (bign, ES256)
✅ Audit log HMAC валиден для bash и SHA-256
✅ Password migration: old passwords readable, new use current profile
✅ Unit тесты coverage >90%
✅ Integration test: full auth flow с разными profiles
Effort: 4d
Status: [x] (commit e616775)
P0-CE.4: Setup Wizard (On-Premise)
Файлы: frontend/src/pages/SetupWizard.tsx, frontend/src/components/setup/RegionSelector.tsx, backend/internal/setup/handler.go, backend/internal/setup/wizard.go
Проблема: Нет UX для выбора региона при локальной установке
Решение:
7-step wizard: Region → Crypto confirmation → Storage → Admin → etc.
Region selection с detailed compliance checklist
Immutable after first login (нельзя сменить без миграции)
Digital signature админа для КИИ регионов
Compliance report generation при завершении
Критерий приёмки:
Wizard обязателен при первом запуске
Region locked после активации (cannot change without migration)
Confirmation screen с legal implications
Audit log entry для region selection
E2E test: full wizard flow
Accessibility: keyboard navigation, screen reader support
Effort: 3d
Status: [x] (commit: Setup Wizard + 4 regions + compliance report)
P0-CE.5: Tenant Compliance Profile (SaaS)
Файлы: backend/internal/tenant/compliance.go, backend/internal/db/migrations/035_tenant_compliance.sql
Проблема: Multi-tenant SaaS не поддерживает per-tenant compliance
Решение:
Поле compliance_region в tenants таблице (VARCHAR(10), NOT NULL, DEFAULT 'INTL')
Поле compliance_locked (BOOLEAN, DEFAULT false)
Mandatory при tenant creation
Immutable после first data creation
Injected в request context via TenantMiddleware
RLS policies включают compliance_region
Критерий приёмки:
Migration 035 применена без ошибок
Все downstream компоненты видят region через context
RLS policies работают для compliance_region
Admin UI для выбора region при создании tenant
Cannot change region after first data
Unit тесты для middleware injection
Effort: 3d
Status: [x] (P0-CE.5: Migration 036 + TenantComplianceStore + Admin UI + 18 тестов)
P0-CE.6: Data Residency Enforcement
Файлы: backend/internal/storage/residency.go, backend/internal/storage/s3_client.go, backend/internal/api/storage_handlers.go
Проблема: Нет technical enforcement для data residency
Решение:
Region-aware S3 endpoint selection (Минск для BY, Yandex для RU, eu-central-1 для EU, Aliyun для CN)
Cross-border transfer blocking на уровне storage API
Cold storage routing per region retention policy
Monitoring для attempted violations
Audit log для всех residency violations
Критерий приёмки:
API rejects requests с cross-region data access
Audit log для всех residency violations
Multi-region failover в пределах region only
Compliance dashboard показывает residency status
Integration test: cross-border transfer blocked
Performance: <10ms overhead per request
Effort: 4d
Status: [x] (P0-CE.6: ResidencyEnforcer + S3Client + audit violations + 27 тестов)
P0-SEC: Security & Data Integrity
P0-SEC.1: Schema Registry Validation
Файлы: backend/internal/events/schema_registry.go, backend/internal/events/publisher.go, backend/go.mod
Проблема: SchemaRegistry.Validate() закомментирована → риск записи невалидных событий
Решение:
Добавить github.com/xeipuuv/gojsonschema в go.mod
Реализовать валидацию в publisher.go перед publish
Добавить middleware для проверки всех событий
Circuit breaker при >10% failed validations
Логировать failed validations с full payload
Критерий приёмки:
Валидация включена в production config
Тесты покрывают valid/invalid scenarios
Error logging для failed validations
Circuit breaker работает при high failure rate
Performance: <5ms overhead per validation
Integration test: invalid event rejected
Effort: 2d
Status: [x] (P0-SEC.1: SchemaRegistry.Validate + gojsonschema + circuit breaker + тесты)
P0-SEC.2: SMS Provider Implementation
Файлы: backend/internal/notifications/sms/rocketsms.go, backend/internal/notifications/sms/provider.go, backend/internal/notifications/sms/rocketsms_test.go
Проблема: SMSProvider interface без реализации — заглушка
Решение:
Реализовать интеграцию с RocketSMS API (или СМС-центр РБ для compliance)
Добавить fallback на email при недоступности SMS
Delivery tracking с retry logic (3 attempts, exponential backoff)
Rate limiting для SMS (anti-spam: 10 SMS/min per user)
Audit log всех SMS отправлений
Критерий приёмки:
SMS отправляются успешно через RocketSMS
Fallback на email работает при SMS failure
Rate limiting: 10 SMS/min per user
Delivery tracking с retry logic
Audit log для всех SMS
Unit тесты coverage >90%
Integration test: SMS → email fallback
Effort: 3d
Status: [x] (P0-SEC.2: RocketSMS provider + rate limiting + delivery tracking + тесты)
P0-SEC.3: SLA Escalation Integration
Файлы: backend/internal/sla/engine.go, backend/internal/notifications/sla_breach_notifier.go, backend/internal/notifications/escalation_notifier.go
Проблема: Таблицы sla_escalation_rules + sla_escalation_log существуют, но не интегрированы
Решение:
Добавить вызов escalationNotifier.Notify() при breach
Интегрировать с Telegram/SMS/Email
Записывать в sla_escalation_log для audit trail
3 уровня escalation: team_lead → manager → director
Configurable thresholds per SLA policy
Критерий приёмки:
Escalation срабатывает при breach
Уведомления отправляются через все каналы (Telegram, SMS, Email)
Audit log содержит все escalation events
Unit тесты для escalation logic
Integration test: full escalation flow
Performance: <100ms per escalation
Effort: 2d
Status: [x] (P0-SEC.3: EscalationResolver + SLA engine + 3-level escalation + audit log)
P0-SEC.4: SLABreachNotifier Fallback
Файлы: backend/internal/notifications/sla_breach_notifier.go, backend/internal/notifications/contact_cache.go
Проблема: Зависит от UserContactProvider — нет fallback при недоступности БД
Решение:
Добавить cached contacts (Redis/memory с TTL 5min)
Fallback на default admin email
Retry logic с exponential backoff (1s, 2s, 4s, max 3 retries)
Circuit breaker при >50% failures
Alert при cache miss
Критерий приёмки:
Notifier работает при БД downtime
Cached contacts обновляются каждые 5min
Alert отправляется даже без fresh data
Circuit breaker работает при high failure rate
Unit тесты для fallback logic
Integration test: DB down → cache fallback
Effort: 2d
Status: [x] (P0-SEC.4: Contact cache TTL 5min + fallback admin email + retry backoff)
P0-UX: Critical UX Blockers
P0-UX.1: AddDeviceModal Validation
Файлы: frontend/src/components/devices/AddDeviceModal.tsx, frontend/src/lib/validations/device.ts, frontend/src/hooks/useFormValidation.ts
Проблема: Длинная форма, нет dynamic validation, нет индикации обязательных полей
Решение:
Интегрировать react-hook-form + Zod
Dynamic validation для IP, port, credentials
Conditional required fields based on connection type (P2P, ONVIF, SNMP)
Inline error messages под полями
Submit disabled при invalid form
Real-time validation (onChange/onBlur)
Критерий приёмки:
Валидация в real-time (onChange/onBlur)
Required fields помечены звёздочкой
Error messages на всех языках (i18n)
Submit disabled при invalid form
Conditional validation для connection types
Unit тесты для Zod schemas
E2E test: full form validation flow
Effort: 3d
Status: [x] (P0-UX.1: Zod schema + react-hook-form + conditional validation + i18n)
P0-UX.2: Breadcrumbs для Detail Pages
Файлы: frontend/src/components/ui/Breadcrumbs.tsx, frontend/src/pages/WorkOrderDetail.tsx, frontend/src/pages/DeviceDetail.tsx, frontend/src/pages/SiteDetail.tsx
Проблема: Отсутствие хлебных крошек в глубоких страницах
Решение:
Добавить Breadcrumbs component на все detail pages
Структура: Home > Section > Entity > Detail
Clickable breadcrumbs для navigation
Responsive на mobile (truncate middle items)
Accessibility: aria-label="Breadcrumb", keyboard navigation
Критерий приёмки:
Breadcrumbs на WorkOrderDetail, DeviceDetail, SiteDetail
Clickable navigation работает
Responsive на mobile (375px)
Accessibility: aria-label="Breadcrumb"
Keyboard navigation (Tab, Enter)
Unit тесты для Breadcrumbs component
Visual regression test
Effort: 2d
Status: [x] (P0-UX.2: Breadcrumbs компонент + WorkOrderDetail/DeviceDetail/SiteDetail + SiteDetail page)
P0-UX.3: View Mode Persistence
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/hooks/useViewMode.ts, frontend/src/store/viewModeStore.ts
Проблема: Переключение table/kanban/calendar не сохраняется
Решение:
Сохранять в URL query param: ?view=kanban
Сохранять в localStorage для persistence
Restore on page load
Default view per user role (technician → kanban, manager → table)
Bookmark с view mode работает
Критерий приёмки:
View mode сохраняется в URL
Bookmark с view mode работает
localStorage fallback
Default view per user role
Unit тесты для useViewMode hook
E2E test: view mode persistence
Effort: 1d
Status: [x] (P0-UX.3: URL query param + localStorage + role default — уже было реализовано)
P0-UX.4: Kanban Feedback & Animation
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/components/work-orders/WOKanbanBoard.tsx, frontend/src/components/ui/Toast.tsx
Проблема: При drag&drop нет анимации и toast feedback
Решение:
Добавить toast: "WO #123 moved to In Progress"
Undo button в toast (5 sec)
Smooth animation при drop (duration-200, ease-in-out)
Haptic feedback на mobile
Optimistic update с rollback при error
Критерий приёмки:
Toast появляется при status change
Undo работает (revert status)
Animation smooth (60fps)
Accessibility: aria-live="polite"
Optimistic update с rollback
Unit тесты для animation
E2E test: drag&drop with undo
Effort: 2d
Status: [x] (P0-UX.4: Toast + undo + optimistic update + GPU animations + aria-live)
P0-MOBILE: Mobile Critical Fixes
P0-MOBILE.1: Conflict Resolution UI
Файлы: mobile/src/components/ConflictResolutionModal.tsx, mobile/src/services/syncService.ts, mobile/src/store/syncStore.ts
Проблема: Только LWW (last-write-wins), нет UI для manual resolution
Решение:
Modal с diff-view: "Local vs Server"
Кнопки: Keep Local, Keep Server, Merge
Visual highlighting of changes (red = deleted, green = added)
Timestamp comparison
Conflict logged в telemetry
Merge для text fields (line-by-line)
Критерий приёмки:
Modal появляется при conflict
Diff view показывает изменения
Merge для text fields
Conflict logged в telemetry
Unit тесты для diff algorithm
E2E test: conflict resolution flow
Effort: 3d
Status: [x] (P0-MOBILE.1: ConflictResolutionModal + diff view + merge + telemetry)
P0-MOBILE.2: Background Sync Integration
Файлы: mobile/src/hooks/useBackgroundSync.ts, mobile/src/services/syncService.ts, mobile/app.json
Проблема: useBackgroundSync есть, но не интегрирован с SyncService
Решение:
Интегрировать с expo-background-fetch
Sync every 15min when app closed
Sync on network reconnect
Queue management UI
Manual sync button
Sync status indicator
Критерий приёмки:
Background sync работает (15min interval)
Queue count отображается
Manual sync button
Sync status indicator
Unit тесты для background sync
E2E test: background sync flow
Effort: 3d
Status: [x] (P0-MOBILE.2: expo-background-fetch + SyncStatusBar + manual sync + queue UI)
P0-MOBILE.3: Offline Map Tile Caching
Файлы: mobile/src/hooks/useOfflineMap.ts, mobile/src/services/tileCache.ts, mobile/src/store/deviceMapStore.ts
Проблема: useOfflineMap есть, но нет кэширования тайлов
Решение:
Кэшировать map tiles в SQLite (WatermelonDB)
Preload tiles для assigned sites
Offline map availability indicator
Tile expiration (30 days)
Cache size management (clear old tiles)
Max cache size: 500MB
Критерий приёмки:
Tiles кэшируются локально (SQLite)
Map работает offline
Cache size management (500MB limit)
Preload on site assignment
Unit тесты для tile caching
E2E test: offline map usage
Effort: 4d
Status: [x] (P0-MOBILE.3: SQLite tile cache + 500MB limit + preload metadata + CacheMetadata store)
🟡 P1 — HIGH VALUE (Q4 2026, до 2026-12-31)
P1-SEC: Security Hardening
P1-SEC.1: JWT → HttpOnly Cookies
Файлы: frontend/src/hooks/useAuth.tsx, backend/internal/api/auth_handlers.go, backend/internal/auth/jwt.go, backend/internal/auth/cookie.go, backend/internal/api/auth_routes.go, backend/internal/api/server.go, mobile/src/api/client.ts, mobile/src/api/auth.ts, backend/internal/auth/cookie_test.go
Проблема: JWT хранится в localStorage → XSS risk
Решение:
HttpOnly cookies для web (Secure, SameSite=Strict)
CSRF token в заголовке X-CSRF-Token
Mobile app: отдельный механизм (secure storage)
Token refresh endpoint
Logout clears cookie
Критерий приёмки:
HttpOnly cookies работают (Secure, SameSite=Strict)
CSRF protection активна (X-CSRF-Token header)
Mobile app адаптирована (secure storage)
Penetration test ready
Unit тесты для cookie handling
E2E test: auth flow с cookies
Effort: 6d
Status: [x] (P1-SEC.1: JWT → HttpOnly Cookies — complete)
P1-SEC.2: CSRF Tokens для Mutations
Файлы: backend/internal/api/server.go, backend/internal/api/csrf_middleware.go, frontend/src/services/api.ts
Проблема: Нет CSRF protection для state-changing операций
Решение:
Генерировать CSRF-токен при логине
Передавать в X-CSRF-Token header
Проверять на всех POST/PUT/DELETE эндпоинтах
Token rotation every 30min
Exempt safe methods (GET, HEAD, OPTIONS)
Критерий приёмки:
CSRF token генерируется при логине
Все POST/PUT/DELETE проверяют CSRF
Token rotation every 30min
Safe methods exempt
Unit тесты для CSRF middleware
Penetration test: CSRF attack blocked
Effort: 2d
Status: [x] (P1-SEC.2: CSRF Tokens — реализован в рамках P1-SEC.1)
P1-SEC.3: Server-Side Validation
Файлы: backend/internal/api/*_handlers.go, backend/internal/api/validation.go, backend/internal/api/validators/*.go
Проблема: Валидация частично реализована, нет единого подхода
Решение:
Go-валидаторы для всех эндпоинтов (go-playground/validator)
Zod-схемы на фронтенде (уже есть, но не везде)
Единый validation error format
Inline error mapping к полям формы
Rate limiting для failed validations
Критерий приёмки:
Все эндпоинты имеют валидацию
Единый validation error format
Inline error mapping к полям
Rate limiting для failed validations (20 fail/5min → 15min ban)
Unit тесты для validators
Integration test: invalid input rejected
Effort: 4d
Status: [x] (P1-SEC.3 validation.go: FieldError + domain validators + rate limiter)
P1-UX: UX Polish & Consistency
P1-UX.1: Dashboard Unification
Файлы: frontend/src/pages/DashboardHub.tsx, frontend/src/pages/ManagerDashboard.tsx, frontend/src/pages/TechnicianDashboard.tsx, frontend/src/components/dashboard/DashboardTabs.tsx
Проблема: Дублирование: Dashboard, ManagerDashboard, TechnicianDashboard, ExecutiveDashboard
Решение:
Единая страница /dashboard с role-based widgets
Widget visibility per role
Drag-and-drop customization (уже есть DragDropDashboard)
Saved layouts per user
Migration script для old URLs
Критерий приёмки:
Одна dashboard страница
Role-based default widgets
Customization сохраняется
Migration script для old URLs
Unit тесты для role-based logic
E2E test: dashboard customization
Effort: 4d
Status: [x] (P1-UX.1: DashboardHub unified + redirects + role-based tabs)
P1-UX.2: Skeleton на всех страницах
Файлы: frontend/src/components/layout/SkeletonPage.tsx, frontend/src/pages/*.tsx
Проблема: Некоторые страницы (AdvancedAnalytics, TechnicianWeek) загружаются без skeleton
Решение:
Добавить skeleton на все pages с data fetching
Skeleton per component (table, chart, cards)
Shimmer animation
Progressive loading (skeleton → partial → full)
Accessibility: aria-busy="true"
Критерий приёмки:
Skeleton на всех pages (WorkOrderDetail, DeviceDetail, TechnicianWeek, AdvancedAnalytics)
Consistent design
No layout shift
Accessibility: aria-busy="true"
Unit тесты для skeleton components
Visual regression test
Effort: 2d
Status: [x] (P1-UX.2: SkeletonDetailPage, SkeletonTechnicianWeek, SkeletonAdvancedAnalytics + 4 pages)
P1-UX.3: Unified Animations
Файлы: frontend/src/index.css, frontend/src/components/**/*.tsx
Проблема: Разные duration и easing для анимаций → inconsistent UX
Решение:
Задать единые значения: duration-200, ease-in-out
CSS variables: --animation-duration, --animation-easing
Проверить все переходы и анимации
Reduced motion support (prefers-reduced-motion)
Критерий приёмки:
Единые duration и easing
CSS variables для animations
Reduced motion support
No janky animations
Visual regression test
Effort: 1d
Status: [x] (P1-UX.3: CSS variables + reduced-motion + animations)
P1-UX.4: Sidebar aria-current
Файлы: frontend/src/components/layout/Sidebar.tsx
Проблема: Нет aria-current="page" для активных ссылок
Решение:
Добавить aria-current="page" для активной ссылки
Visual indicator (bold, color change)
Keyboard navigation (Arrow keys)
Критерий приёмки:
aria-current="page" для активной ссылки
Visual indicator
Keyboard navigation
Accessibility audit: 0 violations
Effort: 0.5d
Status: [x] (P1-UX.4: aria-current + ArrowUp/Down/Home/End navigation)
P1-UX.5: Virtualization для больших списков
Файлы: frontend/src/pages/Alerts.tsx, frontend/src/pages/Notifications.tsx, frontend/src/pages/AuditLog.tsx
Проблема: Большие списки (10k+ items) без виртуализации → slow rendering
Решение:
Использовать @tanstack/react-virtual
Auto-selection на основе rowCount > 1000
Seamless fallback
Performance monitoring
Критерий приёмки:
Virtualization для Alerts, Notifications, AuditLog
Auto-selection на основе rowCount
No UX degradation
Performance: <100ms render time
Unit тесты для virtualization
Effort: 2d
Status: [x] (P1-UX.5: @tanstack/react-virtual для Alerts/Notifications/AuditLog)
P1-UX.6: Calendar Date Mode Toggle
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/components/work-orders/WorkOrderCalendar.tsx
Проблема: Calendar показывает только deadline, не creation date
Решение:
Toggle: "Show by deadline / Show by creation date"
Color coding: deadline (red), creation (blue)
Dual date display in event details
Preference сохраняется в localStorage
Критерий приёмки:
Toggle работает
Calendar updates without reload
Preference сохраняется
Legend объясняет colors
Unit тесты для toggle logic
Effort: 2d
Status: [x] (P1-UX.6: Calendar Date Mode Toggle + useLocalStorage + 21 tests)
P1-UX.7: RCA Widget в Device Overview
Файлы: frontend/src/pages/DeviceDetail.tsx, frontend/src/components/rca/RCAWidget.tsx
Проблема: RCA граф скрыт в отдельной вкладке
Решение:
Вынести RCA summary в Overview tab
Show: root cause, blast radius, affected devices
Click to expand full RCA graph
"No RCA available" state
Real-time updates via WebSocket
Критерий приёмки:
RCA summary виден сразу
Expandable details
Real-time updates via WebSocket
Export RCA as PDF
Unit тесты для RCAWidget
Effort: 3d
Status: [x] (P1-UX.7: RCAWidget + modal + export + 10 tests)
P1-UX.8: Search Unification
Файлы: frontend/src/components/layout/Header.tsx, frontend/src/components/CommandPalette.tsx
Проблема: Дублирование: Header Search + Command Palette
Решение:
Header search открывает Command Palette
Unified search results
Recent searches shared
Keyboard shortcut: / or ⌘K
Focus management
Критерий приёмки:
Один search component
Results консистентны
Recent searches shared
Accessibility: focus management
Unit тесты для search unification
Effort: 2d
Status: [x] (P1-UX.8: Search Unification — / shortcut + already implemented)
P1-UX.9: Saved Filters
Файлы: frontend/src/components/ui/DataGrid.tsx, frontend/src/store/savedViewsStore.ts, frontend/src/components/ui/SavedFiltersDropdown.tsx
Проблема: Фильтры в DataGrid не сохраняются между сессиями
Решение:
Save filter presets (как SavedViews)
Named filters: "Critical Overdue", "My Team"
Share filters with team
Default filters per role
Export/import filters как JSON
Критерий приёмки:
Save/load filters работает
Filters в dropdown menu
Share via URL
Default filters per role
Unit тесты для filter persistence
Effort: 3d
Status: [x] (P1-UX.9: Saved Filters + export/import/URL/defaults per role + 20 tests)
P1-UX.10: Bulk Operations Progress
Файлы: frontend/src/components/ui/BulkProgressModal.tsx, frontend/src/hooks/useBulkOperations.ts
Проблема: При bulk-операциях нет прогресс-бара
Решение:
Modal с progress bar
Real-time status: "Processing 15/100..."
Cancel button
Error summary при failures
Retry failed items
WebSocket для real-time updates
Критерий приёмки:
Progress bar отображается
Real-time updates (WebSocket)
Cancel работает
Error details с retry
Unit тесты для bulk operations
E2E test: bulk operation with progress
Effort: 3d
Status: [x] (P1-UX.10: useBulkOperations + WebSocket + REST fallback + 10 tests)
P1-UX.11: Contextual Tooltips
Файлы: frontend/src/components/ui/InfoTooltip.tsx, frontend/src/pages/Glossary.tsx
Проблема: Сложные термины (MTBF, SLA, Gatekeeper) не объяснены
Решение:
Info icon (?) рядом с терминами
Tooltip с definition + link to docs
Glossary page /help/glossary
i18n для всех tooltips
Mobile-friendly (tap to show)
Критерий приёмки:
Tooltips на всех сложных терминах
Glossary page доступна
Tooltips accessible (keyboard)
Mobile-friendly (tap to show)
Unit тесты для InfoTooltip
Effort: 2d
Status: [x] (P1-UX.11: InfoTooltip + Glossary + MTBF/NVR/Health tooltips)
P1-PERF: Performance Optimization
P1-PERF.1: Bundle Size Reduction
Файлы: frontend/vite.config.ts, frontend/src/App.tsx, frontend/src/pages/*.tsx
Проблема: Bundle 2.8MB (цель <2MB)
Решение:
Lazy load FullCalendar, Recharts, XLSX
Tree-shaking для lucide-react (уже есть)
Dynamic import для heavy components
Analyze с rollup-plugin-visualizer
Route-based code splitting
Критерий приёмки:
Bundle <2MB
Lighthouse Performance >90
Initial load <3s
Route-based code splitting
Visualizer report в CI
Effort: 3d
Status: [x] (P1-PERF.1: FullCalendar lazy + XLSX lazy + manualChunks + ~1MB saving)
P1-PERF.2: Image Lazy Loading в DataGrid
Файлы: frontend/src/components/ui/DataGrid.tsx, frontend/src/components/ui/LazyImage.tsx
Проблема: Изображения в таблицах загружаются сразу
Решение:
loading="lazy" для всех images
Placeholder (blur hash)
Intersection Observer для off-screen images
Thumbnail generation на backend
WebP format (smaller size)
Критерий приёмки:
Images lazy-loaded
Placeholder отображается
No layout shift
WebP format (smaller size)
Unit тесты для LazyImage
Effort: 2d
Status: [ ]
P1-PERF.3: React Query Optimization
Файлы: frontend/src/hooks/useApiQuery.ts, frontend/src/services/*.ts
Проблема: staleTime и gcTime не оптимизированы
Решение:
Reference data (sites, users): staleTime: 5min, gcTime: 1h
Lists (devices, WOs): staleTime: 30s, gcTime: 5min
keepPreviousData для pagination
Prefetch on hover (уже есть)
Query key factory для type-safe keys
Критерий приёмки:
Optimized cache strategy
No unnecessary refetches
Smooth pagination
Network tab показывает fewer requests
Unit тесты для query optimization
Effort: 1d
Status: [ ]
P1-PERF.4: Health Checks Enhancement
Файлы: backend/internal/api/health_handlers.go, backend/internal/api/services_status.go
Проблема: Health checks базовые, нет детальных проверок
Решение:
Детальные проверки для PostgreSQL, NATS, Redis
Метрики пула соединений (active, idle, max)
Latency measurements
Circuit breaker status
JSON response с detailed status
Критерий приёмки:
Детальные проверки для всех services
Метрики пула соединений
Latency measurements
JSON response с detailed status
Unit тесты для health checks
Effort: 2d
Status: [ ]
P1-PERF.5: Redis для SLA Trackers и Device State
Файлы: backend/internal/sla/engine.go, backend/internal/state/manager.go, backend/internal/state/redis_store.go
Проблема: In-memory map для SLA trackers и device state → не шардится
Решение:
Заменить in-memory map на Redis
Fallback для NATS KV
Distributed locking
TTL для expired entries
Metrics для Redis operations
Критерий приёмки:
Redis для SLA trackers
Redis для device state
Distributed locking
TTL для expired entries
Unit тесты для Redis store
Performance test: 10k ops/sec
Effort: 3d
Status: [ ]
P1-PERF.6: Graceful Shutdown
Файлы: backend/main.go, backend/internal/**/*.go
Проблема: Нет graceful shutdown с таймаутами
Решение:
Гарантировать закрытие всех горутин за 30 секунд
Context cancellation для всех operations
Drain queues before shutdown
Close DB connections gracefully
Log shutdown progress
Критерий приёмки:
Все горутины закрываются за 30s
Context cancellation работает
Queues drained before shutdown
DB connections closed gracefully
Unit тесты для graceful shutdown
Effort: 2d
Status: [ ]
P1-QA: Testing & Quality Assurance
P1-QA.1: E2E Test Expansion
Файлы: frontend/e2e/*.spec.ts, frontend/playwright.config.ts
Проблема: Playwright покрывает только 4 сценария
Решение:
Добавить critical user journeys:
Create WO с checklist
Complete WO с photo upload
Assign technician
Export report
Register P2P device
View RCA graph
Gatekeeper verification
Mock API для isolation
Parallel execution
Критерий приёмки:
15+ e2e тестов
Coverage >80% critical paths
CI integration
Test reports в PR
Parallel execution (<10min)
Effort: 5d
Status: [x] (P1-QA.1: 67 E2E tests = 7 files + Mock API + parallel CI)
P1-QA.2: Mobile E2E Tests
Файлы: mobile/e2e/*.spec.ts, mobile/detox.config.js
Проблема: Mobile тесты практически отсутствуют
Решение:
Настроить Detox или Maestro
Тесты для offline scenarios
Sync conflict resolution
Photo upload + Gatekeeper
Push notifications
QR scanner
Критерий приёмки:
Detox/Maestro настроен
10+ e2e тестов
Offline scenarios covered
CI integration
Test reports в PR
Effort: 5d
Status: [ ]
P1-QA.3: Accessibility Testing в CI
Файлы: frontend/e2e/a11y/*.spec.ts, frontend/playwright.config.ts
Проблема: Нет автоматических a11y проверок
Решение:
Интегрировать @axe-core/playwright
Проверка всех pages
Threshold: 0 critical violations
Report в PR comments
Fail CI при violations
Критерий приёмки:
axe-core интегрирован
Все pages проверяются
CI fails при violations
Accessibility report в PR
0 critical violations
Effort: 2d
Status: [x] (P1-QA.3: @axe-core/playwright + 22 pages + 0 critical threshold)
P1-QA.4: Frontend Error Monitoring (Sentry)
Файлы: frontend/src/lib/sentry.ts, mobile/src/lib/sentry.ts, frontend/src/App.tsx
Проблема: Нет frontend monitoring ошибок
Решение:
Интегрировать Sentry SDK
Capture unhandled exceptions
User context (role, tenant)
Source maps для debugging
Alerting на critical errors
Performance monitoring
Критерий приёмки:
Sentry интегрирован (web + mobile)
Errors captured автоматически
Source maps uploaded
Alerting настроен
Performance monitoring
Unit тесты для Sentry integration
Effort: 2d
Status: [ ]
P1-QA.5: Lighthouse CI
Файлы: .github/workflows/lighthouse.yml, lighthouserc.js
Проблема: Нет автоматических performance тестов
Решение:
Lighthouse CI в PR checks
Thresholds: Performance >90, A11y >95, Best Practices >90
Regression detection
Historical trends
Fail CI при regression
Критерий приёмки:
Lighthouse CI настроен
PR checks работают
Thresholds enforced
Reports в PR comments
Historical trends
Effort: 2d
Status: [ ]
P1-QA.6: Visual Regression Testing
Файлы: frontend/e2e/visual/*.spec.ts, chromatic.yml
Проблема: Нет визуальных регрессионных тестов
Решение:
Chromatic для Storybook
Percy для e2e screenshots
Baseline screenshots
Diff detection
Review workflow
Критерий приёмки:
Chromatic/Percy настроен
Baseline screenshots созданы
CI integration
Review workflow
0 unexpected diffs
Effort: 3d
Status: [ ]
P1-QA.7: Load Testing (k6)
Файлы: tests/load/*.js, tests/load/k6.config.js
Проблема: Нет нагрузочных тестов
Решение:
Сценарии для: GET /devices, POST /work-orders, WebSocket
1000 concurrent users
95th percentile <500ms
Error rate <1%
CI integration
Критерий приёмки:
k6 сценарии написаны
1000 concurrent users
95th percentile <500ms
Error rate <1%
CI integration
Performance report
Effort: 3d
Status: [ ]
P1-QA.8: Frontend Test Coverage 80%
Файлы: frontend/src/**/*.test.tsx, frontend/vitest.config.ts
Проблема: Покрытие фронтенда ~75%, цель 80%+
Решение:
Добавить тесты для DeviceWizard, AssetTree, BeforeAfterSlider
Coverage threshold 80% в CI
Fail CI при <80%
Coverage report в PR
Критерий приёмки:
Coverage >80%
Тесты для DeviceWizard, AssetTree, BeforeAfterSlider
CI fails при <80%
Coverage report в PR
Effort: 4d
Status: [ ]
P1-BACKEND: Backend Quality
P1-BACKEND.1: ActionExecutor Unit Tests
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
P1-BACKEND.2: PlaybookRegistry Versioning
Файлы: backend/internal/playbook/registry.go, backend/internal/playbook/version.go
Проблема: Не поддерживает versioning (нет hot reload)
Решение:
Version field в playbook schema
Hot reload без restart
Rollback capability
Version history
Migration script для old playbooks
Критерий приёмки:
Versioning работает
Hot reload без downtime
Rollback работает
Migration script для old playbooks
Unit тесты для versioning
Effort: 3d
Status: [ ]
P1-BACKEND.3: CMMSIntegrator Context Timeouts
Файлы: backend/internal/cmms/integrator.go, backend/internal/cmms/adapter.go
Проблема: Context передаётся, но не проверяется для таймаутов
Решение:
Check ctx.Done() в long operations
Configurable timeouts per adapter
Graceful cancellation
Timeout metrics
Circuit breaker при timeout
Критерий приёмки:
Context cancellation работает
Timeouts configurable
No goroutine leaks
Metrics для timeout events
Unit тесты для timeout logic
Effort: 2d
Status: [ ]
P1-BACKEND.4: RCA Graph Auto-Update
Файлы: backend/internal/rca/graph_builder.go, backend/internal/rca/event_listener.go
Проблема: Граф не обновляется автоматически при добавлении/удалении устройств
Решение:
Event listener для device changes
Incremental graph updates
Cache invalidation
WebSocket notification
Performance monitoring
Критерий приёмки:
Graph updates автоматически
No full rebuild required
Cache invalidation работает
Real-time updates в UI
Unit тесты для auto-update
Effort: 3d
Status: [ ]
P1-BACKEND.5: RCA BuildFromState Accuracy
Файлы: backend/internal/rca/graph_builder.go, backend/internal/rca/validation.go
Проблема: Эвристика по IP-подсетям даёт ложные связи
Решение:
Use explicit parent-child relationships
Manual topology configuration
Validation rules
Confidence score для inferred connections
Manual override capability
Критерий приёмки:
No false positive connections
Explicit relationships used
Validation warnings
Manual override capability
Unit тесты для validation
Effort: 3d
Status: [ ]
P1-ARCH: Architecture Improvements
P1-ARCH.1: Context Migration to Zustand
Файлы: frontend/src/context/*.tsx, frontend/src/store/*.ts
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
Unit тесты для migrated stores
Effort: 4d
Status: [ ]
P1-ARCH.2: API Routes Organization
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
Update imports
Критерий приёмки:
Routes organized by domain
No breaking changes
Documentation updated
Tests pass
No import errors
Effort: 3d
Status: [ ]
P1-ARCH.3: OpenAPI TypeScript Generation
Файлы: backend/docs/openapi.yaml, frontend/src/types/api.ts, frontend/package.json
Проблема: openapi.go есть, но не используется для генерации TypeScript
Решение:
Настроить oapi-codegen или openapi-typescript
Auto-generate types из OpenAPI spec
Type-safe API client
CI validation
Update all API calls
Критерий приёмки:
Types генерируются автоматически
Type-safe API calls
CI validates spec
No manual type definitions
All API calls updated
Effort: 3d
Status: [ ]
P1-ARCH.4: Replace http.Error с respondError
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
No http.Error usage
Effort: 2d
Status: [ ]
P1-ARCH.5: Trace ID Propagation
Файлы: backend/internal/**/*.go, backend/internal/telemetry/otel.go
Проблема: trace_id propagation только в api слое
Решение:
Inject trace_id в context
Propagate через все service layers
Include в logs, metrics, events
OpenTelemetry integration
Jaeger/Zipkin visualization
Критерий приёмки:
trace_id в всех logs
Distributed tracing работает
Jaeger/Zipkin integration
Performance impact <5%
Unit тесты для trace propagation
Effort: 3d
Status: [ ]
🟢 P2 — ENTERPRISE FEATURES (Q1 2027, до 2027-03-31)
P2-CR: Compliance & Regional Expansion
P2-CR.1: Regional Retention Policies
Файлы: backend/internal/retention/policy.go, backend/internal/cron/retention_cron.go
Проблема: Retention hardcoded, не соответствует разным требованиям (BY 5 лет, EU min necessary)
Решение:
Retention policy per data type per region
Automatic lifecycle transitions (hot → cold → archive → delete)
Compliance-aware deletion (legal hold support)
Audit log для всех retention actions
Критерий приёмки:
5+ retention profiles (BY/RU/EU/US/CN)
Automated lifecycle management
Legal hold prevents deletion
Audit log для всех retention actions
Unit тесты для retention logic
Effort: 3d
Status: [x] (P2-CR.1: 5 profiles + LegalHold + lifecycle + retention_cron + 28 tests)
P2-CR.2: Regional Compliance Reports
Файлы: frontend/src/pages/compliance/ComplianceDashboard.tsx, backend/internal/compliance/reports.go
Проблема: Нет automated compliance reporting
Решение:
Dashboard с real-time compliance status per region
Auto-generated reports для регуляторов (ОАЦ, ФСТЭК, GDPR DPIA)
Gap analysis с remediation recommendations
Scheduled exports для auditors
PDF/XML exports
Критерий приёмки:
Dashboard для всех supported regions
PDF/XML exports в regulatory formats
Gap detection с severity levels
Scheduler для periodic reports
Unit тесты для report generation
Effort: 5d
Status: [x] (P2-CR.2: PDF/XML reports + ComplianceDashboard + gap analysis + 20 tests)
P2-CR.3: Regional Password Policies
Файлы: backend/internal/auth/password_policy.go, backend/internal/auth/password_validator.go
Проблема: Единая password policy для всех регионов
Решение:
Region-specific rules (BY: 12 chars + rotation 90d, EU: no forced rotation per NIST)
Complexity requirements per region
History length per region
MFA enforcement rules per region
Graceful migration для existing users
Критерий приёмки:
5 password policy profiles
Runtime enforcement
Graceful migration для existing users
Admin UI для policy customization
Unit тесты для password validation
Effort: 2d
Status: [x] (P2-CR.3: 5 password profiles + ValidatePassword + tests)
P2-CR.4: Session & Auth Regional Policies
Файлы: backend/internal/auth/session_policy.go, backend/internal/auth/session_middleware.go
Проблема: Session timeouts не адаптированы под КИИ требования
Решение:
BY: 30 min idle timeout (КИИ)
RU: 15 min (ФСТЭК)
EU/US: 8 hours
Failed login lockout per region
Concurrent session limits per region
Graceful warning before timeout
Критерий приёмки:
Policy enforcement на уровне auth middleware
Graceful warning before timeout
Admin override для экстренных случаев
Audit log для session events
Unit тесты для session policy
Effort: 2d
Status: [x] (P2-CR.4: 5 session profiles + region in JWT + admin bypass + 30+ tests)
P2-REGIONS: Regional Expansion
P2-RU.1: GOST Crypto Integration
Файлы: backend/internal/crypto/providers/gost.go, backend/internal/crypto/providers/gost_test.go
Решение:
GOST 28147-89 / Магма / Кузнечик encryption
Стрибог hash
ГОСТ Р 34.10-2012 signatures
КриптоПро HSM integration
Performance benchmarks
Критерий приёмки:
Full GOST stack working
КриптоПро integration tested
Performance benchmarks
ФСТЭК pre-certification checklist
Unit тесты для GOST crypto
Effort: 6d
Status: [ ]
P2-RU.2: 152-ФЗ Personal Data Features
Файлы: backend/internal/compliance/personal_data.go, frontend/src/pages/compliance/PersonalData.tsx
Решение:
Consent management
Data subject access requests
Automated data inventory
Roskomnadzor reporting
Data anonymization
Критерий приёмки:
Consent management UI
Data subject access requests
Automated data inventory
Roskomnadzor reporting
Unit тесты для personal data features
Effort: 5d
Status: [ ]
P2-EU.1: GDPR-Specific Features
Файлы: backend/internal/compliance/gdpr.go, frontend/src/pages/compliance/GDPR.tsx
Решение:
Right to be forgotten workflow
Data portability exports
Consent audit trail
DPIA report generator
Schrems II compliant data transfers (SCCs)
Критерий приёмки:
Right to be forgotten workflow
Data portability exports
Consent audit trail
DPIA report generator
Unit тесты для GDPR features
Effort: 5d
Status: [ ]
P2-EU.2: NIS2 Incident Reporting
Файлы: backend/internal/compliance/nis2.go, frontend/src/pages/compliance/NIS2.tsx
Решение:
Automated incident classification
24h/72h reporting templates
ENISA-format exports
Incident timeline
Критерий приёмки:
Automated incident classification
24h/72h reporting templates
ENISA-format exports
Unit тесты для NIS2 reporting
Effort: 3d
Status: [ ]
P2-CN.1: SM Crypto (国密)
Файлы: backend/internal/crypto/providers/sm.go, backend/internal/crypto/providers/sm_test.go
Решение:
SM4 encryption
SM3 hash
SM2 signatures
Local HSM integration
Performance benchmarks
Критерий приёмки:
Full SM crypto stack working
Local HSM integration
Performance benchmarks
Unit тесты для SM crypto
Effort: 6d
Status: [ ]
P2-CN.2: MLPS 2.0 Compliance
Файлы: backend/internal/compliance/mlps.go, frontend/src/pages/compliance/MLPS.tsx
Решение:
Security level classification
Audit log enhancements
Real-name verification hooks
MLPS reporting
Критерий приёмки:
Security level classification
Audit log enhancements
Real-name verification hooks
Unit тесты для MLPS compliance
Effort: 4d
Status: [ ]
P2-US.1: FIPS 140-3 Mode
Файлы: backend/internal/crypto/providers/fips.go, backend/internal/crypto/providers/fips_test.go
Решение:
FIPS-validated crypto modules
Approved algorithms only
Self-tests on startup
FIPS mode toggle
Критерий приёмки:
FIPS-validated crypto modules
Approved algorithms only
Self-tests on startup
Unit тесты для FIPS mode
Effort: 3d
Status: [ ]
P2-US.2: HIPAA Add-on (Optional)
Файлы: backend/internal/compliance/hipaa.go, frontend/src/pages/compliance/HIPAA.tsx
Решение:
PHI flagging
BAA-compliant audit logs
Breach notification workflows
HIPAA reporting
Критерий приёмки:
PHI flagging
BAA-compliant audit logs
Breach notification workflows
Unit тесты для HIPAA features
Effort: 5d
Status: [ ]
P2-US.3: SOC 2 Reporting
Файлы: backend/internal/compliance/soc2.go, frontend/src/pages/compliance/SOC2.tsx
Решение:
Trust Service Criteria mapping
Automated evidence collection
Continuous monitoring dashboards
SOC 2 reporting
Критерий приёмки:
Trust Service Criteria mapping
Automated evidence collection
Continuous monitoring dashboards
Unit тесты для SOC 2 reporting
Effort: 4d
Status: [ ]
P2-AI: Advanced Analytics & AI
P2-AI.1: Real ML Model Integration
Файлы: backend/analytics/predict.py, backend/internal/ml/prediction_service.go, backend/internal/ml/model.go
Проблема: XGBoost обучается на синтетических данных
Решение:
Train на production data из TimescaleDB
Features: offline_ratio, error_count, reboot_count, age_days
Publish predictions через NATS
Confidence score
A/B testing framework
Критерий приёмки:
Model trained на real data
Predictions >75% accuracy
NATS integration
A/B testing framework
Unit тесты для ML model
Effort: 5d
Status: [ ]
P2-AI.2: AI Assistant Chat
Файлы: frontend/src/components/ai/AIAssistantPanel.tsx, backend/internal/ai/deepseek_client.go
Проблема: Нет контекстных подсказок
Решение:
Chat-панель с DeepSeek integration
Context-aware recommendations
RCA suggestions
Natural language queries
Feedback mechanism
Критерий приёмки:
Chat panel доступен
Context-aware responses
Response time <2s
Feedback mechanism
Unit тесты для AI assistant
Effort: 4d
Status: [ ]
P2-AI.3: Predictive Maintenance Dashboard
Файлы: frontend/src/pages/PredictiveMaintenance.tsx, frontend/src/components/dashboard/PredictiveWidget.tsx
Проблема: Нет визуализации at-risk devices
Решение:
KPI cards с at-risk count
Risk distribution chart
Failure by type breakdown
Drill-down в device detail
Export to PDF/Excel
Критерий приёмки:
Dashboard показывает at-risk devices
Drill-down в device detail
Export to PDF/Excel
Email digest для managers
Unit тесты для predictive dashboard
Effort: 3d
Status: [ ]
P2-WF: Workflow & Automation
P2-WF.1: Workflow Builder UI
Файлы: frontend/src/components/workflow/WorkflowBuilder.tsx, frontend/src/components/workflow/WorkflowNode.tsx, backend/internal/workflow/engine.go
Проблема: workflow/engine.go есть, но нет UI
Решение:
React Flow для drag&drop
CEL conditions editor
Workflow testing mode
Version control
Import/export workflows
Критерий приёмки:
Drag&drop работает
CEL editor с highlighting
Test mode с mock data
Version history
Unit тесты для workflow builder
Effort: 5d
Status: [ ]
P2-WF.2: Resource Planning Calendar
Файлы: frontend/src/pages/TechnicianWeek.tsx, frontend/src/components/workforce/TechnicianCalendar.tsx, backend/internal/workforce/scheduler.go
Проблема: Нет календаря загрузки техников
Решение:
Week view с technician rows
Drag-and-drop WO assignment
Conflict detection
Availability indicators
Print-friendly view
Критерий приёмки:
Week view отображает загрузку
Drag&drop для reassignment
Conflict warnings
Print-friendly view
Unit тесты для calendar
Effort: 4d
Status: [ ]
P2-INT: Integration Ecosystem
P2-INT.1: Webhook Builder UI
Файлы: frontend/src/components/webhooks/WebhookBuilder.tsx, backend/internal/webhooks/manager.go
Проблема: webhook/verify.go есть, но нет visual builder
Решение:
Event type selector
Payload preview
Test button
Delivery logs
Webhook templates
Критерий приёмки:
Builder создает webhooks
Payload preview работает
Test mode sends mock event
Delivery logs для debugging
Unit тесты для webhook builder
Effort: 3d
Status: [ ]
P2-INT.2: OAuth2 для External Adapters
Файлы: backend/internal/cmms/servicenow/client.go, backend/internal/cmms/jira/client.go, backend/internal/oauth2/token_manager.go
Проблема: ServiceNow/Jira используют basic auth
Решение:
OAuth2 flow implementation
Token refresh logic
Secure token storage (AES-256-GCM)
Fallback to basic auth
Token rotation
Критерий приёмки:
OAuth2 flow работает
Token auto-refresh
Secure storage (encrypted)
Fallback работает
Unit тесты для OAuth2
Effort: 3d
Status: [x] (commit b30021e)
P2-INT.3: Excel Import/Export for WO
Файлы: frontend/src/pages/WorkOrders.tsx, backend/internal/reports/excel_handler.go, backend/internal/api/export_handlers.go
Проблема: Есть export_handlers.go, но нет UI-кнопки "Export all"
Решение:
Bulk export button
Import wizard для Excel
Column mapping UI
Error report для failed imports
Template download
Критерий приёмки:
Export all работает для 10k+ WO
Import wizard с preview
Column mapping с auto-detect
Error report для failed imports
Unit тесты для Excel import/export
Effort: 2d
Status: [x] (commit ae6cd90)
🔵 P3 — TECHNICAL DEBT (Q2 2027, до 2027-06-30)
P3-SEC: Security & Compliance
P3-SEC.1: belt-GCM Migration (СТБ 34.101.31)
Файлы: backend/internal/crypto/aes.go, backend/internal/crypto/belt.go, backend/internal/crypto/migration.go
Проблема: Используется AES-256-GCM, для КИИ РБ требуется belt-GCM
Решение:
Мигрировать на github.com/bp2012/crypto/belt
Backward compatibility для existing data
Migration script для encrypted data
Performance benchmarks
Security audit
Критерий приёмки:
belt-GCM используется для new data
Migration script работает
Performance benchmarks
Security audit passed
Unit тесты для migration
Effort: 4d
Status: [ ]
P3-SEC.2: JWT bign-curve256v1 Migration
Файлы: backend/internal/auth/jwt.go, backend/internal/auth/bign.go
Проблема: JWT HS256, для РБ требуется bign-curve256v1
Решение:
Мигрировать на СТБ bign-curve
Backward compatibility
Token rotation
Performance benchmarks
Критерий приёмки:
bign-curve используется
Old tokens валидны до expiry
Migration seamless
Compliance verified
Unit тесты для bign JWT
Effort: 3d
Status: [ ]
P3-SEC.3: Mobile Certificate Pinning
Файлы: mobile/src/lib/api.ts, mobile/src/lib/certificate_pinning.ts
Проблема: Нет certificate pinning
Решение:
Pin server certificates
Certificate rotation support
Fallback on cert mismatch
Security audit
Критерий приёмки:
Certificate pinning работает
MITM protection
Certificate rotation
Security audit passed
Unit тесты для certificate pinning
Effort: 2d
Status: [ ]
P3-DX: Developer Experience
P3-DX.1: Storybook Expansion
Файлы: frontend/src/components/**/*.stories.tsx, frontend/.storybook/main.js
Проблема: Storybook только для 8 из 56 компонентов
Решение:
Stories для всех UI components
Interactive examples
Accessibility notes
Design tokens documentation
Chromatic integration
Критерий приёмки:
50+ stories
All atoms/molecules covered
Interactive controls
A11y guidelines
Chromatic integration
Effort: 5d
Status: [ ]
P3-DX.2: Onboarding Tour для всех ролей
Файлы: frontend/src/components/OnboardingTour.tsx, frontend/src/store/onboardingStore.ts
Проблема: OnboardingTour только для админов
Решение:
Role-specific tours
Technician: WO creation, QR scanner
Manager: Dashboard, reports
Admin: Settings, integrations
Skip option
Критерий приёмки:
Tours для всех ролей
Contextual steps
Skip button
Completion tracking
Unit тесты для onboarding
Effort: 3d
Status: [ ]
P3-DX.3: Help System & Glossary
Файлы: frontend/src/pages/Help.tsx, frontend/src/pages/Glossary.tsx, frontend/src/components/ui/InfoTooltip.tsx
Проблема: Нет справочной системы
Решение:
/help page с FAQ
/glossary с terms
Search functionality
Video tutorials
i18n для всех content
Критерий приёмки:
Help page доступна
Glossary с 50+ terms
Search работает
i18n для всех content
Unit тесты для help system
Effort: 3d
Status: [ ]
P3-DX.4: DEVELOPMENT.md
Файлы: DEVELOPMENT.md (новый)
Проблема: Нет инструкций по локальной настройке
Решение:
Инструкции по локальной настройке
Переменные окружения
Запуск backend/frontend/mobile
Troubleshooting
Contributing guidelines
Критерий приёмки:
DEVELOPMENT.md создан
Все инструкции работают
Troubleshooting section
Contributing guidelines
Effort: 1d
Status: [ ]
P3-DX.5: Swagger UI на /api/v1/docs
Файлы: backend/internal/api/server.go, backend/internal/api/openapi.go
Проблема: ServeSwaggerUI есть, но не включён
Решение:
Включить Swagger UI на /api/v1/docs
Auto-generate из OpenAPI spec
Authentication для Swagger UI
Try it out functionality
Критерий приёмки:
Swagger UI доступен на /api/v1/docs
Auto-generated из OpenAPI
Authentication работает
Try it out functionality
Effort: 1d
Status: [ ]
P3-UI: UI/UX Polish
P3-UI.1: Design Tokens
Файлы: frontend/src/index.css, frontend/tailwind.config.js
Проблема: Нет CSS variables для design tokens
Решение:
CSS variables для всех цветов и размеров
Tailwind config с custom tokens
Dark mode tokens
Theme customizer
Критерий приёмки:
CSS variables для всех tokens
Tailwind config updated
Dark mode tokens
Theme customizer
Effort: 2d
Status: [ ]
P3-UI.2: Micro-interactions
Файлы: frontend/src/components/ui/Button.tsx, frontend/src/components/ui/Card.tsx
Проблема: Нет микроинтеракций
Решение:
Ripple-эффект для кнопок
Hover-тени для карточек
Smooth transitions
Haptic feedback на mobile
Критерий приёмки:
Ripple-эффект для кнопок
Hover-тени для карточек
Smooth transitions
Haptic feedback на mobile
Effort: 2d
Status: [ ]
P3-UI.3: Mobile Responsiveness
Файлы: mobile/src/screens/*.tsx
Проблема: ScrollView в больших списках → slow rendering
Решение:
Заменить ScrollView на FlatList в больших списках
Добавить жесты (swipe) для переключения вкладок
Optimize images
Lazy loading
Критерий приёмки:
FlatList вместо ScrollView
Swipe жесты для tabs
Optimized images
Lazy loading
Effort: 3d
Status: [ ]
P3-NICE: Nice-to-Have
P3-NICE.1: Real-time Collaboration
Файлы: frontend/src/pages/WorkOrderDetail.tsx, backend/internal/ws/hub.go
Проблема: Нет WebSocket presence indicators
Решение:
"Ivan is editing this WO" indicator
Cursor sharing
Conflict warnings
Real-time updates
Критерий приёмки:
Presence indicators работают
Real-time updates
Conflict warnings
Graceful degradation
Effort: 4d
Status: [ ]
P3-NICE.2: White-label Theming
Файлы: frontend/src/store/themeStore.ts, frontend/src/components/ui/ThemeCustomizer.tsx
Проблема: Нет white-label для enterprise
Решение:
Custom logo, colors
Per-tenant themes
CSS variables
Preview mode
Критерий приёмки:
Custom branding работает
Per-tenant themes
No code changes required
Preview mode
Effort: 3d
Status: [ ]
P3-NICE.3: Edge Agent SL-4 Security
Файлы: backend/internal/edge/agent.go, backend/internal/edge/security.go
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
Метрика
Текущее
Target (Q4 2026)
Target (Q2 2027)
Bundle Size
2.8MB
<2MB
<1.5MB
FCP
1.8s
<1.5s
<1.2s
LCP
2.5s
<2.0s
<1.5s
Lighthouse Score
87
>95
>98
E2E Coverage
4 scenarios
15+ scenarios
30+ scenarios
Mobile E2E
0%
10+ scenarios
20+ scenarios
A11y Violations
Unknown
0 critical
0 violations
Context Count
14
4
2
API Files
70+ in root
Organized by domain
Clean structure
Test Coverage (React)
75%
>80%
>85%
Test Coverage (Go)
82%
>85%
>90%
SLA Breach Detection
Manual
Automatic
100% automated
Supported Regions
1 (BY)
3 (BY, EU, INTL)
7 (+RU, CN, US, KZ)
Certifications
0
0
2-3 (ОАЦ, ISO 27001)
Compliance Reports
Manual
5 automated
15+ automated
Crypto Providers
1 (AES)
3
5+
Regional Revenue %
100% BY
70% BY / 30% INTL
40% BY / 60% global
Enterprise Deals
Pilot
2-3 signed
10+ active
🗺️ Roadmap
Q3 2026 (July-September) — P0: Critical Fixes
├─ Week 1-2: Regional Compliance Foundation
│  ├─ P0-CE.1: ComplianceProfile abstraction
│  ├─ P0-CE.2: Regional crypto providers (belt + AES)
│  ├─ P0-CE.3: Hash & signature providers
│  └─ P0-CE.4: Setup Wizard для on-premise
├─ Week 3-4: Security & Data Integrity
│  ├─ P0-SEC.1: Schema Registry Validation
│  ├─ P0-SEC.2: SMS Provider Implementation
│  ├─ P0-SEC.3: SLA Escalation Integration
│  └─ P0-SEC.4: SLABreachNotifier Fallback
├─ Week 5-6: Critical UX Blockers
│  ├─ P0-UX.1: AddDeviceModal Validation
│  ├─ P0-UX.2: Breadcrumbs для Detail Pages
│  ├─ P0-UX.3: View Mode Persistence
│  └─ P0-UX.4: Kanban Feedback & Animation
└─ Week 7-8: Mobile Critical Fixes
   ├─ P0-MOBILE.1: Conflict Resolution UI
   ├─ P0-MOBILE.2: Background Sync Integration
   └─ P0-MOBILE.3: Offline Map Tile Caching

Q4 2026 (October-December) — P1: High Value
├─ Week 9-10: Security Hardening
│  ├─ P1-SEC.1: JWT → HttpOnly Cookies
│  ├─ P1-SEC.2: CSRF Tokens для Mutations
│  └─ P1-SEC.3: Server-Side Validation
├─ Week 11-12: UX Polish & Consistency
│  ├─ P1-UX.1: Dashboard Unification
│  ├─ P1-UX.2: Skeleton на всех страницах
│  ├─ P1-UX.3: Unified Animations
│  ├─ P1-UX.4: Sidebar aria-current
│  ├─ P1-UX.5: Virtualization для больших списков
│  ├─ P1-UX.6: Calendar Date Mode Toggle
│  ├─ P1-UX.7: RCA Widget в Device Overview
│  ├─ P1-UX.8: Search Unification
│  ├─ P1-UX.9: Saved Filters
│  ├─ P1-UX.10: Bulk Operations Progress
│  └─ P1-UX.11: Contextual Tooltips
├─ Week 13-14: Performance Optimization
│  ├─ P1-PERF.1: Bundle Size Reduction
│  ├─ P1-PERF.2: Image Lazy Loading
│  ├─ P1-PERF.3: React Query Optimization
│  ├─ P1-PERF.4: Health Checks Enhancement
│  ├─ P1-PERF.5: Redis для SLA Trackers
│  └─ P1-PERF.6: Graceful Shutdown
└─ Week 15-16: Testing & QA
   ├─ P1-QA.1: E2E Test Expansion
   ├─ P1-QA.2: Mobile E2E Tests
   ├─ P1-QA.3: Accessibility Testing в CI
   ├─ P1-QA.4: Frontend Error Monitoring (Sentry)
   ├─ P1-QA.5: Lighthouse CI
   ├─ P1-QA.6: Visual Regression Testing
   ├─ P1-QA.7: Load Testing (k6)
   └─ P1-QA.8: Frontend Test Coverage 80%

Q1 2027 (January-March) — P2: Enterprise Features
├─ Compliance & Regional Expansion
│  ├─ P2-CR.1: Regional Retention Policies
│  ├─ P2-CR.2: Regional Compliance Reports
│  ├─ P2-CR.3: Regional Password Policies
│  └─ P2-CR.4: Session & Auth Regional Policies
├─ Regional Expansion
│  ├─ P2-RU.1: GOST Crypto Integration
│  ├─ P2-RU.2: 152-ФЗ Personal Data Features
│  ├─ P2-EU.1: GDPR-Specific Features
│  ├─ P2-EU.2: NIS2 Incident Reporting
│  ├─ P2-CN.1: SM Crypto (国密)
│  ├─ P2-CN.2: MLPS 2.0 Compliance
│  ├─ P2-US.1: FIPS 140-3 Mode
│  ├─ P2-US.2: HIPAA Add-on
│  └─ P2-US.3: SOC 2 Reporting
├─ Advanced Analytics & AI
│  ├─ P2-AI.1: Real ML Model Integration
│  ├─ P2-AI.2: AI Assistant Chat
│  └─ P2-AI.3: Predictive Maintenance Dashboard
├─ Workflow & Automation
│  ├─ P2-WF.1: Workflow Builder UI
│  └─ P2-WF.2: Resource Planning Calendar
└─ Integration Ecosystem
   ├─ P2-INT.1: Webhook Builder UI
   ├─ P2-INT.2: OAuth2 для External Adapters ✅ DONE
   └─ P2-INT.3: Excel Import/Export for WO ✅ DONE

Q2 2027 (April-June) — P3: Technical Debt
├─ Security & Compliance
│  ├─ P3-SEC.1: belt-GCM Migration
│  ├─ P3-SEC.2: JWT bign-curve256v1 Migration
│  └─ P3-SEC.3: Mobile Certificate Pinning
├─ Developer Experience
│  ├─ P3-DX.1: Storybook Expansion
│  ├─ P3-DX.2: Onboarding Tour для всех ролей
│  ├─ P3-DX.3: Help System & Glossary
│  ├─ P3-DX.4: DEVELOPMENT.md
│  └─ P3-DX.5: Swagger UI на /api/v1/docs
├─ UI/UX Polish
│  ├─ P3-UI.1: Design Tokens
│  ├─ P3-UI.2: Micro-interactions
│  └─ P3-UI.3: Mobile Responsiveness
└─ Nice-to-Have
   ├─ P3-NICE.1: Real-time Collaboration
   ├─ P3-NICE.2: White-label Theming
   └─ P3-NICE.3: Edge Agent SL-4 Security
Полезные ссылки
Architecture: ARCHITECTURE.md
UX Guidelines: docs/ux/ux-guideline.md
ADR Log: docs/adr/
API Docs: backend/docs/api/
Design System: frontend/.storybook/
CI/CD: .github/workflows/
Regional Compliance: docs/compliance/regional-profiles.md
Security Policy: docs/iso27001/security-policy.md
📝 История изменений
2026-06-27 — Major Update: Regional Compliance Engine
Добавлена P0-CE секция: Regional Compliance Engine (6 задач)
Интегрированы задачи из альтернативного TODO
Добавлены P2-REGIONS: Regional Expansion (RU, EU, CN, US)
Обновлены Success Metrics с regional KPIs
Добавлен roadmap для regional expansion
Общая готовность: 97.5% → Target: 99.5%
2026-06-26 — Initial TODO Creation
Создан unified TODO на основе 3 code reviews
Определены P0-P3 приоритеты
Добавлены критерии приёмки для всех задач
Установлены метрики успеха
Последний коммит: HEAD
Branch: main
Next Review: 2026-07-04
✅ Чеклист для Roo перед началом работы
Прочитать соответствующий раздел TODO
Проверить dependencies (если есть)
Создать feature branch: feature/P0-CE.1-compliance-profile
Реализовать задачу с учётом code review чеклиста
Написать тесты (unit + integration)
Обновить документацию (если применимо)
Создать PR с описанием изменений
Отметить [x] + дата в TODO после merge
Помни: Качество > Скорость. Лучше сделать 1 задачу идеально, чем 3 задачи с багами.


P0-MARKET: Market Expansion (Q3 2026 — Q1 2027)
Phase 1: СНГ Foundation (Weeks 1-10, Q3 2026)
Target: Разблокировать $122M TAM (BY + RU + KZ + UZ)
P0-MKT.1: ГОСТ Crypto Providers (RU/KZ)
Файлы: backend/internal/crypto/providers/gost.go, providers/streebog.go, providers/gost_sign.go
Бизнес-ценность: Открывает 🇷🇺 Россию ($85M) + 🇰🇿 Казахстан ($20M)
Решение:
GOST 28147-89 / Магма / Кузнечик encryption
Стрибог-256 hash (ГОСТ Р 34.11-2012)
ГОСТ Р 34.10-2012 signatures
Runtime selection через ComplianceProfile
Переиспользуется для KZ (80% требований идентичны RU)
Критерий приёмки:
GOST encrypt/decrypt round-trip работает
Стрибог hash совпадает с reference implementation
JWT sign/verify с ГОСТ Р 34.10-2012
Benchmark: overhead <3x vs AES
ComplianceProfile "RU" активирует GOST providers
Unit tests coverage >90%
Effort: 4d
Status: [ ]
Dependencies: P0-CE.1, P0-CE.2
P0-MKT.2: 152-ФЗ Features (RU/KZ Shared)
Файлы: backend/internal/compliance/personal_data.go, frontend/src/pages/compliance/PersonalData.tsx, backend/internal/api/personal_data_handlers.go
Бизнес-ценность: Обязательно для всех клиентов с ПДн (RU + KZ)
Решение:
Consent management UI (сбор, хранение, отзыв)
Data Subject Access Requests (DSAR) workflow
Automated data inventory (что хранится, где, зачем)
Роскомнадзор reporting templates
Data anonymization для analytics
Privacy policy generator
Критерий приёмки:
Consent collection на signup + settings
DSAR request через UI + email notification
Data inventory export (CSV/JSON)
Automated deletion по request
Роскомнадзор report template (PDF)
152-ФЗ compliance checklist в admin UI
Effort: 3w (15d)
Status: [ ]
Reuse: 80% кода переиспользуется для KZ
P0-MKT.3: belt-GCM + bign-curve (BY)
Файлы: backend/internal/crypto/belt.go, backend/internal/crypto/bign.go, backend/internal/auth/bign_jwt.go
Бизнес-ценность: Открывает 🇧🇾 Беларусь ($7M, но стратегически важно)
Решение:
belt-GCM (СТБ 34.101.31) для encryption
bign-curve256v1 для JWT signatures
bash-256 для audit log HMAC
Migration script для existing encrypted data
FIPS self-tests on startup
Критерий приёмки:
belt-GCM encrypt/decrypt работает
bign-curve JWT sign/verify
bash-256 HMAC для audit log
Migration script без data loss
ComplianceProfile "BY" активирует СТБ providers
Security audit report
Effort: 4w (20d)
Status: [ ]
Note: Параллельно готовить ОАЦ сертификацию (external vendor)
P0-MKT.4: ОАЦ Pre-Certification Package (BY)
Файлы: docs/compliance/oac-certification/, backend/internal/crypto/stb_test.go
Бизнес-ценность: Госконтракты в РБ (premium pricing +40%)
Решение:
Documentation package для ОАЦ
СТБ 34.101.27 audit log compliance tests
СТБ 34.101.30 crypto compliance tests
Security policy documents
Incident response procedures
Engage consulting firm (ОАЦ-approved)
Критерий приёмки:
Documentation package готов
Pre-audit self-assessment пройден
Consulting firm engaged
Certification timeline согласован
Budget approved ($15-25K)
Effort: 4w (parallel with P0-MKT.3)
Status: [ ]
P0-MKT.5: Uzbekistan Entry (Lowest Friction)
Файлы: backend/internal/compliance/uzbekistan.go, frontend/src/locales/uz/, mobile/src/i18n/uz.json
Бизнес-ценность: 🇺🇿 Узбекистан ($10M, fastest growing 25% YoY)
Решение:
Узбекский язык (кириллица + латиница)
Law "On Personal Data" compliance (procedural)
ID.UZ SSO integration (optional для enterprise)
my.gov.uz integration (для госконтрактов)
Local billing в UZS
НЕ требует крипто-сертификации!
Критерий приёмки:
Узбекский язык: 95% coverage
Cyrillic + Latin script toggle
Personal data consent UI
UZS currency formatting
Local support partner identified
3 pilot customers signed
Effort: 2w (10d)
Status: [ ]
Quick Win: Самый простой вход в СНГ
P0-MKT.6: Kazakhstan Localization
Файлы: frontend/src/locales/kk/, mobile/src/i18n/kk.json, backend/internal/compliance/kazakhstan.go
Бизнес-ценность: 🇰🇿 Казахстан ($20M, 18% YoY growth)
Решение:
Казахский язык (кириллица, transition to Latin planned)
Law "On Personal Data" compliance (reuse RU 152-ФЗ code)
eGov.kz integration (для госконтрактов)
KZ-specific data residency (AIFC exception для fintech)
Local billing в KZT
Критерий приёмки:
Казахский язык: 95% coverage
Reuse 80% 152-ФЗ code
eGov.kz SSO integration
KZT currency formatting
Data residency enforcement (KZ only)
3 pilot customers signed
Effort: 2w (10d)
Status: [ ]
Reuse: Использует P0-MKT.1 (GOST) + P0-MKT.2 (152-ФЗ)
Phase 2: Simple High-Demand Markets (Weeks 11-18, Q4 2026)
Target: Разблокировать $277M TAM (TR + BR + MX + ID + VN)
Ключевой insight: Все используют already-supported языки (tr, pt, es, id, vi — нужны только id, vi)
P0-MKT.7: Turkey Entry (Highest ROI)
Файлы: backend/internal/compliance/turkey.go, backend/internal/integrations/edevlet.go, backend/internal/integrations/kep.go
Бизнес-ценность: 🇹🇷 Турция ($42M TAM, 16% YoY)
Язык: ✅ Уже есть (tr в i18n)
Решение:
KVKK compliance (procedural, GDPR-like)
e-Devlet SSO integration (для госконтрактов)
KEP (registered email) для legal notifications
TRY currency formatting
Local data residency (Türkiye only)
НЕ требует крипто-сертификации!
Критерий приёмки:
KVKK consent management
Data subject rights workflow
e-Devlet SSO (OAuth2 flow)
KEP integration для legal docs
TRY currency + date format (dd.mm.yyyy)
Local partner identified
5 pilot customers signed
Effort: 2w (10d)
Status: [ ]
Quick Win: Язык уже есть, только procedural compliance
P0-MKT.8: Brazil Entry (Largest LATAM)
Файлы: backend/internal/compliance/brazil.go, backend/internal/integrations/govbr.go, backend/internal/integrations/pix.go
Бизнес-ценность: 🇧🇷 Бразилия ($75M TAM, 18% YoY)
Язык: ✅ Уже есть (pt в i18n, minor PT-BR adjustments needed)
Решение:
LGPD compliance (GDPR-like, reuse EU code)
Gov.br SSO integration (для госконтрактов)
PIX payment integration (optional, nice-to-have)
BRL currency formatting
ICP-Brasil digital signatures (для enterprise)
НЕ требует крипто-сертификации!
Критерий приёмки:
LGPD consent + DSAR workflow (reuse GDPR)
Gov.br OAuth2 integration
PIX webhook для payments
BRL currency + Brazilian date format
Portuguese (BR) localization polish
Local partner identified
5 pilot customers signed
Effort: 2w (10d)
Status: [ ]
Reuse: 70% GDPR code переиспользуется
P0-MKT.9: Mexico Entry (Nearshoring Boom)
Файлы: backend/internal/compliance/mexico.go, backend/internal/integrations/sat.go
Бизнес-ценность: 🇲🇽 Мексика ($50M TAM, 20% YoY, nearshoring)
Язык: ✅ Уже есть (es в i18n, minor MX Spanish adjustments)
Решение:
LFPDPPP compliance (similar to LGPD)
SAT integration (tax authority, для enterprise)
CURP validation (national ID)
MXN currency formatting
Mexican Spanish localization
НЕ требует крипто-сертификации!
Критерий приёмки:
LFPDPPP consent workflow
SAT API integration (CFDI 4.0)
CURP validation endpoint
MXN currency + Mexican format
Spanish (MX) localization polish
Local partner identified
3 pilot customers signed
Effort: 2w (10d)
Status: [ ]
P0-MKT.10: Vietnam Entry (Fastest Growing SEA)
Файлы: frontend/src/locales/vi/, mobile/src/i18n/vi.json, backend/internal/compliance/vietnam.go
Бизнес-ценность: 🇻🇳 Вьетнам ($50M TAM, 28% YoY!)
Язык: ❌ Нужен (vi) — 1 неделя effort
Решение:
Вьетнамский язык (Latin script с diacritics)
Decree 13/2023 compliance (data residency)
VNeID integration (для госконтрактов)
VND currency formatting (large numbers)
Lunar calendar support (для planning)
НЕ требует крипто-сертификации!
Критерий приёмки:
Вьетнамский язык: 95% coverage
Diacritics rendering correct
Data residency enforcement (Vietnam only)
VND currency formatting (₫ symbol, no decimals)
Lunar calendar component
FPT Software partnership (channel partner)
5 pilot customers signed
Effort: 2w (10d)
Status: [ ]
P0-MKT.11: Indonesia Entry (Largest SEA)
Файлы: frontend/src/locales/id/, mobile/src/i18n/id.json, backend/internal/compliance/indonesia.go
Бизнес-ценность: 🇮🇩 Индонезия ($65M TAM, 24% YoY)
Язык: ❌ Нужен (id) — 1 неделя effort
Решение:
Bahasa Indonesia (Latin script, simple)
UU PDP compliance (similar to GDPR)
SATUSEHAT integration (healthcare sector)
Dukcapil integration (national ID, optional)
IDR currency formatting
Halal certification tracking (food sector, nice-to-have)
НЕ требует крипто-сертификации!
Критерий приёмки:
Bahasa Indonesia: 95% coverage
UU PDP consent workflow
SATUSEHAT API integration (healthcare only)
IDR currency formatting (Rp symbol)
Telkom Indonesia partnership
5 pilot customers signed
Effort: 2w (10d)
Status: [ ]
Phase 3: Africa Expansion (Weeks 19-24, Q1 2027)
Target: Разблокировать $42M TAM (NG + KE + ZA)
Ключевой insight: English-speaking markets, только procedural compliance
P0-MKT.12: Nigeria Entry (Largest Africa)
Файлы: backend/internal/compliance/nigeria.go, backend/internal/integrations/nimc.go, backend/internal/integrations/bvn.go
Бизнес-ценность: 🇳🇬 Нигерия ($32M TAM, 30% YoY — fastest growing!)
Язык: ✅ Уже есть (en)
Решение:
NDPR compliance (similar to GDPR)
NIMC integration (National ID)
BVN integration (banking sector)
NGN currency formatting
Low-bandwidth optimization (2G/3G support)
Pidgin English option (nice-to-have)
НЕ требует крипто-сертификации!
Критерий приёмки:
NDPR consent workflow
NIMC API integration
BVN validation (banking only)
NGN currency formatting (₦ symbol)
Low-bandwidth mode (<100KB page load)
MTN partnership (telco distribution)
3 pilot customers signed
Effort: 2w (10d)
Status: [ ]
P0-MKT.13: Kenya Entry (East Africa Hub)
Файлы: frontend/src/locales/sw/, backend/internal/compliance/kenya.go, backend/internal/integrations/mpesa.go
Бизнес-ценность: 🇰🇪 Кения ($10M TAM, но East Africa hub, 28% YoY)
Язык: ❌ Нужен Swahili (sw) — 1 неделя
Решение:
Swahili language (Latin script)
DPA 2019 compliance (similar to GDPR)
M-Pesa integration (payments, CRITICAL)
eCitizen integration (government services)
KES currency formatting
НЕ требует крипто-сертификации!
Критерий приёмки:
Swahili: 95% coverage
DPA 2019 consent workflow
M-Pesa STK Push + B2C integration
eCitizen SSO (optional)
KES currency formatting
Safaricom partnership
3 pilot customers signed
Effort: 2w (10d)
Status: [ ]
P0-MKT.14: South Africa Entry (Mature Market)
Файлы: backend/internal/compliance/south_africa.go, backend/internal/integrations/sars.go
Бизнес-ценность: 🇿🇦 ЮАР ($20M TAM, 15% YoY, most mature Africa market)
Язык: ✅ Уже есть (en)
Решение:
POPIA compliance (similar to GDPR)
Cybercrimes Act 2021 compliance
MIC Policy Framework (government)
SARS integration (tax, enterprise)
ZAR currency formatting
НЕ требует крипто-сертификации!
Критерий приёмки:
POPIA consent + DSAR workflow
Cybercrimes Act incident reporting
MIC compliance checklist
SARS eFiling integration (enterprise)
ZAR currency formatting (R symbol)
Local partner identified
3 pilot customers signed
Effort: 2w (10d)
Status: [ ]
