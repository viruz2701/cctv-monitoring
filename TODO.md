Правила для Roo при работе с TODO
Перед началом задачи: Прочитать соответствующий раздел, проверить dependencies

Во время работы: Коммитить атомарно, в сообщении указывать ID задачи (например, P0-SEC.2: CSRF Protection)

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
Goal: Production-ready, unblock enterprise sales, 99.5% readiness

P0-SEC: Security Blockers
P0-SEC.1: СТБ 34.101.30 Crypto Compliance
Файлы: backend/internal/crypto/belt.go, backend/internal/crypto/bign.go, backend/internal/crypto/bash.go, backend/internal/auth/bign_jwt.go

Проблема: Используется AES-256-GCM вместо belt-gcm → запрещено для КИИ РБ

Решение:

Интегрировать github.com/bp2012/crypto

Заменить crypto/aes на belt-gcm

Заменить crypto/sha256 на bash-256

Заменить HMAC-SHA256 на bign-curve256v1

Migration script для existing encrypted data

Критерий приёмки:

belt-GCM encrypt/decrypt round-trip работает

bign-curve JWT sign/verify работает

bash-256 HMAC для audit log

Migration script без data loss

ComplianceProfile "BY" активирует СТБ providers

Unit tests coverage >90%

Security audit report готов для ОАЦ

Effort: 4 weeks (20d)

Status: [ ]

Business Impact: Разблокировать $7M КИИ РБ market

P0-SEC.2: CORS Wildcard Fix
Файлы: backend/internal/config/config.go, backend/internal/api/cors_middleware.go

Проблема: cors_allowed_origins: ["*"] → OWASP ASVS L3 V9.1 violation

Решение:

Убрать ["*"] default

Require explicit configuration

Add validation в startup: fail если empty или содержит *

Environment-specific CORS (dev: localhost, prod: production domains)

Критерий приёмки:

Production config требует explicit origins

Startup fails при wildcard или empty

Dev environment работает с localhost

Unit тесты для validation

OWASP ZAP scan: 0 CORS findings

Effort: 2 hours

Status: [ ]

P0-SEC.3: CSRF Protection + HttpOnly Cookies
Файлы: frontend/src/hooks/useAuth.ts, backend/internal/api/auth_handlers.go, backend/internal/api/csrf_middleware.go

Проблема: JWT в localStorage (XSS risk), нет CSRF токенов

Решение:

Мигрировать JWT на HttpOnly cookies (Secure, SameSite=Strict)

Добавить CSRF token в X-CSRF-Token header

Проверять на всех POST/PUT/DELETE эндпоинтах

Token rotation every 30min

Exempt safe methods (GET, HEAD, OPTIONS)

Критерий приёмки:

HttpOnly cookies работают (Secure, SameSite=Strict)

CSRF protection активна (X-CSRF-Token header)

Mobile app адаптирована (secure storage)

Token rotation every 30min

Unit тесты для CSRF middleware

E2E test: auth flow с cookies

Penetration test: CSRF attack blocked

Effort: 6 days

Status: [ ]

P0-SEC.4: Module Path Mismatch
Файлы: backend/go.mod, backend/main.go, backend/internal/**/*.go

Проблема: import "gb-telemetry-collector/internal/..." может не совпадать с module path

Решение:

Проверить go.mod module path

Синхронизировать все imports

Добавить module path validation в CI/CD

Критерий приёмки:

go build проходит без ошибок

Все imports синхронизированы

CI/CD validation работает

No import cycle errors

Effort: 1 hour

Status: [ ]

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

User testing: 5 technicians, NPS >8

Effort: 3 days

Status: [ ]

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

Effort: 3 days

Status: [ ]

P0-MOBILE.3: Offline Maps Tile Caching
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

Effort: 4 days

Status: [ ]

P0-MOBILE.4: Mobile E2E Tests (Detox/Maestro)
Файлы: mobile/e2e/*.spec.ts, mobile/detox.config.js

Проблема: Mobile app не покрыт E2E тестами

Решение:

Настроить Detox или Maestro

Тесты для offline scenarios

Sync conflict resolution tests

Photo upload + Gatekeeper tests

Push notifications tests

QR scanner tests

Критерий приёмки:

Detox/Maestro настроен

20+ e2e тестов

Offline scenarios covered

CI integration

Test reports в PR

Parallel execution (<15min)

Effort: 5 days

Status: [ ]

P0-UX: Critical UX Blockers
P0-UX.1: AddDeviceModal Validation
Файлы: frontend/src/components/devices/AddDeviceModal.tsx, frontend/src/lib/validations/device.ts, frontend/src/hooks/useFormValidation.ts

Проблема: Длинная форма, нет dynamic validation

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

Effort: 3 days

Status: [ ]

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

Effort: 2 days

Status: [ ]

P0-UX.3: View Mode Persistence
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/hooks/useViewMode.ts, frontend/src/store/viewModeStore.ts

Проблема: Переключение table/kanban/calendar не сохраняется

Решение:

Сохранять в URL query param: ?view=kanban

Сохранять в localStorage для persistence

Restore on page load

Default view per user role (technician → kanban, manager → table)

Критерий приёмки:

View mode сохраняется в URL

Bookmark с view mode работает

localStorage fallback

Default view per user role

Unit тесты для useViewMode hook

E2E test: view mode persistence

Effort: 1 day

Status: [ ]

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

Effort: 2 days

Status: [ ]

P0-CE: Compliance Foundation
P0-CE.1: ComplianceProfile Abstraction Layer
Файлы: backend/internal/compliance/profile.go, backend/internal/compliance/registry.go, backend/internal/compliance/providers.go

Проблема: Криптография и security policies захардкожены под РБ

Решение:

Создать ComplianceProfile интерфейс с политиками: crypto, hash, signature, password, data_residency, retention, audit, session

Provider Registry для runtime-загрузки провайдеров по региону

Inject через DI container на основе tenant/instance config

3 baseline профиля: BY (СТБ), EU (GDPR), INTL (ISO 27001)

Критерий приёмки:

ComplianceProfile interface с 8 policy методами

Provider Registry с thread-safe registration

Startup fails при отсутствии required provider

Unit тесты для profile switching (coverage >90%)

Integration test: BY profile → belt-GCM, EU profile → AES-256-GCM

Effort: 5 days

Status: [ ]

Dependencies: P0-SEC.1 (СТБ crypto)

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

belt-GCM работает для BY profile

AES-256-GCM fallback для INTL

Benchmark: overhead <2x для всех providers

Graceful error если provider недоступен

Unit тесты для encrypt/decrypt round-trip

Performance test: 1000 ops/sec

Effort: 4 days

Status: [ ]

P0-CE.3: Setup Wizard (On-Premise)
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

Region locked после активации

Confirmation screen с legal implications

Audit log entry для region selection

E2E test: full wizard flow

Accessibility: keyboard navigation, screen reader support

Effort: 3 days

Status: [ ]

P0-CE.4: Tenant Compliance Profile (SaaS)
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

Effort: 3 days

Status: [ ]

P0-CE.5: Data Residency Enforcement
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

Effort: 4 days

Status: [ ]

P0-BACKEND: Backend Critical Gaps
P0-BACKEND.1: NATS JetStream Mandatory
Файлы: backend/config.yaml, backend/internal/state/state_manager.go, backend/main.go

Проблема: InMemory state manager в production не шардится между подами

Решение:

Убрать memory fallback из production config

Сделать JetStream mandatory

Добавить health check для NATS connectivity в /health/ready

Graceful shutdown при недоступности JetStream

Критерий приёмки:

Production config требует JetStream

Startup fails если JetStream недоступен

/health/ready endpoint проверяет NATS

Graceful shutdown работает

Unit тесты для state manager

Effort: 1 day

Status: [ ]

P0-BACKEND.2: Schema Registry Validation
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

Effort: 2 days

Status: [ ]

P0-BACKEND.3: SMS Provider Implementation
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

Effort: 3 days

Status: [ ]

P0-BACKEND.4: SLA Escalation Integration
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

Effort: 2 days

Status: [ ]

P0-BACKEND.5: SLABreachNotifier Fallback
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

Effort: 2 days

Status: [ ]

🟡 P1 — HIGH VALUE (Q4 2026, до 2026-12-31)
Goal: Close competitive gaps, parity с MaintainX/Fiix/UpKeep

P1-SEC: Security Hardening
P1-SEC.1: 2FA/WebAuthn
Файлы: backend/internal/auth/webauthn.go, frontend/src/components/auth/WebAuthnSetup.tsx

Проблема: Только TOTP 2FA (неудобно для enterprise), нет FIDO2/WebAuthn

Решение:

Добавить WebAuthn/FIDO2 support

Hardware tokens (YubiKey, Titan)

Biometric authentication (Touch ID, Face ID)

Passwordless login option

Recovery codes для backup

Критерий приёмки:

WebAuthn registration работает

Hardware tokens (YubiKey) поддерживаются

Biometric authentication (где поддерживается)

Recovery codes generated

Unit тесты для WebAuthn flow

E2E test: passwordless login

Effort: 5 days

Status: [ ]

P1-SEC.2: Data Loss Prevention (DLP)
Файлы: backend/internal/dlp/detector.go, backend/internal/api/export_handlers.go

Проблема: Нет DLP для sensitive data, можно экспортировать PII без audit trail

Решение:

PII detection в exports (emails, phones, addresses, SSN)

Automatic redaction или approval workflow

Audit log для всех exports с PII

Configurable sensitivity levels

GDPR "right to be forgotten" support

Критерий приёмки:

PII detection работает (regex + ML)

Automatic redaction или approval workflow

Audit log для всех exports с PII

Configurable sensitivity levels

Unit тесты для PII detection

Integration test: export with PII redaction

Effort: 4 days

Status: [ ]

P1-SEC.3: Rate Limiting Enhancement
Файлы: backend/internal/api/rate_limiter.go, backend/internal/redis/rate_limit.go

Проблема: In-memory rate limiter (не шардится), нет distributed rate limiting

Решение:

Мигрировать на Redis-based rate limiter

Sliding window algorithm

Per-user + per-IP limits

Circuit breaker при >1000 req/min

Configurable limits per endpoint

Критерий приёмки:

Redis-based rate limiter работает

Sliding window algorithm

Per-user + per-IP limits

Circuit breaker при high load

Unit тесты для rate limiting

Load test: 10k concurrent users

Effort: 3 days

Status: [ ]

P1-SEC.4: Secrets Rotation
Файлы: backend/internal/secrets/rotation.go, backend/internal/auth/jwt.go

Проблема: Нет automated rotation для secrets (JWT secret, API keys, HMAC key)

Решение:

Automated rotation для JWT secrets (every 90 days)

Grace period (old + new secrets valid)

Audit log для rotation events

Alerting при rotation failure

Manual rotation trigger для emergencies

Критерий приёмки:

Automated rotation every 90 days

Grace period (old + new valid)

Audit log для rotation events

Alerting при failure

Manual rotation trigger

Unit тесты для rotation logic

Effort: 3 days

Status: [ ]

P1-UX: Competitive UX Parity
P1-UX.1: WorkOrders Redesign (Snipe-IT Pattern)
Файлы: frontend/src/pages/WorkOrders.tsx, frontend/src/components/work-orders/WODataGrid.tsx

Проблема: Не применены Snipe-IT паттерны, нет bulk actions, inline edit

Решение:

Применить Snipe-IT таблицы (DataGrid)

Bulk actions toolbar (assign, change priority, close, export)

Inline status change

Quick filters (My, Overdue, Critical, Today, Unassigned)

Column resize + reorder

Export to CSV/Excel

Критерий приёмки:

DataGrid с bulk actions

Inline status change

Quick filters работают

Column resize + reorder

Export to CSV/Excel

Unit тесты для WODataGrid

E2E test: bulk operations

Effort: 4 days

Status: [ ]

P1-UX.2: SpareParts Redesign (Shelf.nu Pattern)
Файлы: frontend/src/pages/SpareParts.tsx, frontend/src/components/spare-parts/PartCard.tsx

Проблема: Не применены Shelf.nu паттерны, нет карточек с изображениями

Решение:

Применить Shelf.nu карточки

Image + QR code

Custom fields (warranty, vendor, location)

Stock level indicators (low/out)

History of movements

QR workflow (scan-to-WO, scan-to-inventory)

Критерий приёмки:

PartCard с image + QR code

Custom fields (warranty, vendor, location)

Stock level indicators

History of movements

QR workflow

Unit тесты для PartCard

E2E test: QR scan workflow

Effort: 3 days

Status: [ ]

P1-UX.3: Dashboard Unification
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

Effort: 4 days

Status: [ ]

P1-UX.4: Skeleton на всех страницах
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

Effort: 2 days

Status: [ ]

P1-UX.5: Calendar Date Mode Toggle
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

Effort: 2 days

Status: [ ]

P1-UX.6: RCA Widget в Device Overview
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

Effort: 3 days

Status: [ ]

P1-UX.7: Search Unification
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

Effort: 2 days

Status: [ ]

P1-UX.8: Saved Filters
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

Effort: 3 days

Status: [ ]

P1-UX.9: Bulk Operations Progress
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

Effort: 3 days

Status: [ ]

P1-UX.10: Contextual Tooltips
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

Effort: 2 days

Status: [ ]

P1-UX.11: Error Handling UI (NEW)
Файлы: frontend/src/components/ui/ErrorBoundary.tsx, frontend/src/hooks/useErrorHandler.ts

Проблема: Нет унифицированного UI для ошибок загрузки, нет кнопки Retry

Решение:

Добавить ErrorBoundary для всех асинхронных операций

Fallback UI с сообщением об ошибке и кнопкой Retry

Consistent design (центрированная карточка с иконкой)

Logging ошибок в Sentry

Retry with exponential backoff

Критерий приёмки:

ErrorBoundary работает на всех страницах

Retry button работает

Consistent design

Sentry logging

Unit тесты для ErrorBoundary

Effort: 2 days

Status: [ ]

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

Effort: 3 days

Status: [ ]

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

Effort: 2 days

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

Effort: 1 day

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

Effort: 2 days

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

Effort: 3 days

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

Effort: 2 days

Status: [ ]

P1-PERF.7: Performance Benchmarking Suite (NEW)
Файлы: backend/tests/benchmarks/*.go, frontend/tests/benchmarks/*.ts

Проблема: Нет регулярных бенчмарков для выявления регрессий

Решение:

Бенчмарки для критических путей: RCA engine, SLA calculation, CMMS sync, Event Store

Запуск в CI и сравнение с baseline

Alerting при >5% degradation

Исторические тренды

Критерий приёмки:

Бенчмарки написаны для всех критических компонентов

CI integration

Baseline comparison

Alerting при degradation

Performance report

Effort: 2 days

Status: [ ]

P1-PERF.8: Redis Connection Pool Optimization (NEW)
Файлы: backend/internal/redis/pool.go, backend/internal/redis/metrics.go

Проблема: Redis connection pool не оптимизирован, нет мониторинга

Решение:

Настроить pool size, timeout, idle timeout

Добавить метрики (active, idle, wait count)

Graceful handling of connection errors

Circuit breaker для Redis

Критерий приёмки:

Optimal pool settings

Метрики доступны в /metrics

Graceful degradation при Redis failure

Unit тесты для pool

Effort: 1 day

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

50+ e2e тестов

Coverage >80% critical paths

CI integration

Test reports в PR

Parallel execution (<10min)

Effort: 5 days

Status: [ ]

P1-QA.2: Accessibility Testing в CI
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

Effort: 2 days

Status: [ ]

P1-QA.3: Frontend Error Monitoring (Sentry)
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

Effort: 2 days

Status: [ ]

P1-QA.4: Lighthouse CI
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

Effort: 2 days

Status: [ ]

P1-QA.5: Load Testing (k6)
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

Effort: 3 days

Status: [ ]

P1-QA.6: Frontend Test Coverage 80%
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

Effort: 4 days

Status: [ ]

P1-QA.7: Chaos Engineering Testing (NEW)
Файлы: tests/chaos/*.js, tests/chaos/scenarios/*.js

Проблема: Нет тестов устойчивости к сбоям компонентов

Решение:

Использовать toxiproxy или chaos-mesh

Сценарии: NATS down, DB down, Redis down, high latency

Автоматическое восстановление

Метрики времени восстановления

Критерий приёмки:

Chaos-сценарии реализованы

Автоматическое восстановление

Метрики времени восстановления

CI integration (опционально)

Effort: 3 days

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

Effort: 2 days

Status: [ ]

P1-BACKEND.2: Playbook Registry Versioning
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

Effort: 3 days

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

Effort: 2 days

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

Effort: 3 days

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

Effort: 3 days

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

Effort: 4 days

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

Effort: 3 days

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

Effort: 3 days

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

Effort: 2 days

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

Effort: 3 days

Status: [ ]

🟢 P2 — ENTERPRISE FEATURES (Q1 2027, до 2027-03-31)
Goal: Competitive differentiation, enterprise sales enablers

P2-AI: Advanced Analytics & AI
P2-AI.1: Real ML Model Integration
Файлы: backend/analytics/predict.py, backend/internal/ml/prediction_service.go, backend/internal/ml/model.go

Проблема: XGBoost обучается на синтетических данных

Решение:

Train на production data из TimescaleDB

Features: offline_ratio, error_count, reboot_count, age_days, avg_alarm_priority

Publish predictions через NATS

Confidence score

A/B testing framework

Continuous learning pipeline

Критерий приёмки:

Model trained на real data (12+ months)

Predictions >75% accuracy

NATS integration

A/B testing framework

Confidence score >70%

Unit тесты для ML model

Production deployment с monitoring

Effort: 5 days

Status: [ ]

P2-AI.2: AI Assistant Chat
Файлы: frontend/src/components/ai/AIAssistantPanel.tsx, backend/internal/ai/deepseek_client.go

Проблема: Нет контекстных подсказок

Решение:

Chat-панель с DeepSeek integration

Context-aware recommendations

RCA suggestions

Natural language queries

Feedback mechanism (thumbs up/down)

Conversation history

Критерий приёмки:

Chat panel доступен

Context-aware responses

Response time <2s

Feedback mechanism

Conversation history

Unit тесты для AI assistant

Effort: 4 days

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

Email digest для managers

Recommended actions

Критерий приёмки:

Dashboard показывает at-risk devices

Drill-down в device detail

Export to PDF/Excel

Email digest для managers

Recommended actions

Unit тесты для predictive dashboard

Effort: 3 days

Status: [ ]

P2-AI.4: Anomaly Detection
Файлы: backend/internal/ml/anomaly_detection.go, backend/internal/alerts/anomaly_engine.go

Проблема: Только rule-based alerts, нет unsupervised ML

Решение:

Unsupervised ML для baseline deviation

Auto-learning baselines per device

Anomaly scoring

Integration с Alert Engine

Explainability (SHAP values)

Критерий приёмки:

Unsupervised ML работает

Auto-learning baselines

Anomaly scoring

Integration с Alert Engine

Explainability (SHAP)

Unit тесты для anomaly detection

Effort: 4 days

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

Template library

Критерий приёмки:

Drag&drop работает

CEL editor с highlighting

Test mode с mock data

Version history

Import/export

Template library

Unit тесты для workflow builder

Effort: 5 days

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

Skills matching

Критерий приёмки:

Week view отображает загрузку

Drag&drop для reassignment

Conflict warnings

Print-friendly view

Skills matching

Unit тесты для calendar

Effort: 4 days

Status: [ ]

P2-WF.3: Meter Triggers
Файлы: backend/internal/meters/trigger_engine.go, frontend/src/pages/MeterDashboard.tsx

Проблема: Нет auto-WO creation по meter thresholds

Решение:

Auto-create WO when HDD >90%, uptime >10000h, temp >80°C

Configurable thresholds per device type

Meter reading integration (SNMP, ONVIF)

Dashboard с meter trends

Alerting при approaching threshold

Критерий приёмки:

Auto-create WO по thresholds

Configurable thresholds

Meter reading integration

Dashboard с trends

Alerting при approaching

Unit тесты для trigger engine

Effort: 4 days

Status: [ ]

P2-WF.4: Conditional Checklists
Файлы: frontend/src/components/work-orders/ChecklistEditor.tsx, backend/internal/work-orders/checklist_engine.go

Проблема: Static checklists, нет dynamic fields

Решение:

Dynamic fields based on device type

Conditional logic (if PTZ → show pan/tilt/zoom tests)

Sub-items

Required/optional fields

Photo requirements per item

Критерий приёмки:

Dynamic fields based on device type

Conditional logic

Sub-items

Required/optional fields

Photo requirements

Unit тесты для checklist engine

Effort: 3 days

Status: [ ]

P2-INV: Inventory & Vendor Management
P2-INV.1: Auto Parts Deduction
Файлы: backend/internal/inventory/parts_deduction.go, frontend/src/components/work-orders/PartsUsedForm.tsx

Проблема: Manual entry для parts used, нет auto-deduction

Решение:

Auto-deduct from inventory при WO completion

Cost snapshot at time of use

Stock level validation

Auto-create reorder WO when stock < min

Audit trail для всех deductions

Критерий приёмки:

Auto-deduct при WO completion

Cost snapshot

Stock level validation

Auto-create reorder WO

Audit trail

Unit тесты для parts deduction

Effort: 3 days

Status: [ ]

P2-INV.2: Vendor Scorecards
Файлы: backend/internal/inventory/vendor_scorecard.go, frontend/src/pages/VendorScorecards.tsx

Проблема: Нет vendor performance tracking

Решение:

On-time delivery %

Cost variance (budget vs actual)

Quality score (rework rate)

MTBF/MTTR by vendor

Automated scoring

Quarterly reports

Критерий приёмки:

On-time delivery %

Cost variance

Quality score

MTBF/MTTR by vendor

Automated scoring

Quarterly reports

Unit тесты для vendor scorecards

Effort: 3 days

Status: [ ]

P2-INV.3: Lifecycle Cost Tracking
Файлы: backend/internal/inventory/lifecycle_cost.go, frontend/src/pages/AssetLifecycle.tsx

Проблема: Нет TCO (Total Cost of Ownership) tracking

Решение:

Purchase cost + maintenance + downtime = TCO

Depreciation tracking

ROI calculation

Replacement recommendations

Budget forecasting

Критерий приёмки:

TCO calculation

Depreciation tracking

ROI calculation

Replacement recommendations

Budget forecasting

Unit тесты для lifecycle cost

Effort: 3 days

Status: [ ]

P2-INV.4: Reorder Automation
Файлы: backend/internal/inventory/reorder_automation.go, frontend/src/components/inventory/ReorderWizard.tsx

Проблема: Manual reorder process

Решение:

Auto-create PO when stock < min

Vendor routing (best price, fastest delivery)

Approval workflow

PO tracking

Delivery notifications

Критерий приёмки:

Auto-create PO

Vendor routing

Approval workflow

PO tracking

Delivery notifications

Unit тесты для reorder automation

Effort: 3 days

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

Retry configuration

Критерий приёмки:

Builder создает webhooks

Payload preview работает

Test mode sends mock event

Delivery logs для debugging

Webhook templates

Retry configuration

Unit тесты для webhook builder

Effort: 3 days

Status: [ ]

P2-INT.2: Developer Portal
Файлы: frontend/src/pages/DeveloperPortal.tsx, backend/internal/api/openapi.go, docs/api/

Решение:

SDKs (Python, JavaScript, Go)

Sandbox environment

Postman collection

Rate limit documentation

Authentication guides

Code examples

Критерий приёмки:

SDKs для 3 languages

Sandbox environment

Postman collection

Rate limit docs

Authentication guides

Code examples

Effort: 5 days

Status: [ ]

P2-INT.3: API Versioning
Файлы: backend/internal/api/versioning.go, backend/internal/api/v1/*.go, backend/internal/api/v2/*.go

Решение:

Explicit versioning (v1, v2)

Deprecation policy (6 months notice)

Changelog

Migration guides

Backward compatibility

Критерий приёмки:

Explicit versioning

Deprecation policy

Changelog

Migration guides

Backward compatibility

Unit тесты для versioning

Effort: 2 days

Status: [ ]

P2-INT.4: Bi-directional Sync
Файлы: backend/internal/cmms/sync/bidirectional.go, backend/internal/cmms/sync/conflict_resolution.go

Решение:

Conflict resolution (LWW, manual merge)

Idempotency keys

Retry queues

Dead letter queue

Sync status dashboard

Критерий приёмки:

Conflict resolution

Idempotency keys

Retry queues

Dead letter queue

Sync status dashboard

Unit тесты для bi-directional sync

Effort: 4 days

Status: [ ]

P2-CR: Compliance & Regional Expansion
P2-CR.1: Regional Retention Policies
Файлы: backend/internal/retention/policy.go, backend/internal/cron/retention_cron.go

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

Effort: 3 days

Status: [ ]

P2-CR.2: Regional Compliance Reports
Файлы: frontend/src/pages/compliance/ComplianceDashboard.tsx, backend/internal/compliance/reports.go

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

Effort: 5 days

Status: [ ]

P2-CR.3: Regional Password Policies
Файлы: backend/internal/auth/password_policy.go, backend/internal/auth/password_validator.go

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

Effort: 2 days

Status: [ ]

P2-CR.4: Session & Auth Regional Policies
Файлы: backend/internal/auth/session_policy.go, backend/internal/auth/session_middleware.go

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

Effort: 2 days

Status: [ ]

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

Effort: 6 days

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

Effort: 5 days

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

Effort: 5 days

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

Effort: 3 days

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

Effort: 6 days

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

Effort: 4 days

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

Effort: 3 days

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

Effort: 5 days

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

Effort: 4 days

Status: [ ]

🔵 P3 — TECHNICAL DEBT (Q2 2027, до 2027-06-30)
Goal: Long-term maintainability, developer experience

P3-SEC: Security & Compliance
P3-SEC.1: belt-GCM Migration (СТБ 34.101.31)
Файлы: backend/internal/crypto/aes.go, backend/internal/crypto/belt.go, backend/internal/crypto/migration.go

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

Effort: 4 days

Status: [ ]

P3-SEC.2: JWT bign-curve256v1 Migration
Файлы: backend/internal/auth/jwt.go, backend/internal/auth/bign.go

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

Effort: 3 days

Status: [ ]

P3-SEC.3: Mobile Certificate Pinning
Файлы: mobile/src/lib/api.ts, mobile/src/lib/certificate_pinning.ts

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

Effort: 2 days

Status: [ ]

P3-DX: Developer Experience
P3-DX.1: Storybook Expansion
Файлы: frontend/src/components/**/*.stories.tsx, frontend/.storybook/main.js

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

Effort: 5 days

Status: [ ]

P3-DX.2: Onboarding Tour для всех ролей
Файлы: frontend/src/components/OnboardingTour.tsx, frontend/src/store/onboardingStore.ts

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

Effort: 3 days

Status: [ ]

P3-DX.3: Help System & Glossary
Файлы: frontend/src/pages/Help.tsx, frontend/src/pages/Glossary.tsx, frontend/src/components/ui/InfoTooltip.tsx

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

Effort: 3 days

Status: [ ]

P3-DX.4: DEVELOPMENT.md
Файлы: DEVELOPMENT.md (новый)

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

Effort: 1 day

Status: [ ]

P3-DX.5: Swagger UI на /api/v1/docs
Файлы: backend/internal/api/server.go, backend/internal/api/openapi.go

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

Effort: 1 day

Status: [ ]

P3-UI: UI/UX Polish
P3-UI.1: Design Tokens
Файлы: frontend/src/index.css, frontend/tailwind.config.js

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

Effort: 2 days

Status: [ ]

P3-UI.2: Micro-interactions
Файлы: frontend/src/components/ui/Button.tsx, frontend/src/components/ui/Card.tsx

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

Effort: 2 days

Status: [ ]

P3-UI.3: Mobile Responsiveness
Файлы: mobile/src/screens/*.tsx

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

Effort: 3 days

Status: [ ]

P3-NICE: Nice-to-Have
P3-NICE.1: Real-time Collaboration
Файлы: frontend/src/pages/WorkOrderDetail.tsx, backend/internal/ws/hub.go

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

Effort: 4 days

Status: [ ]

P3-NICE.2: White-label Theming
Файлы: frontend/src/store/themeStore.ts, frontend/src/components/ui/ThemeCustomizer.tsx

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

Effort: 3 days

Status: [ ]

P3-NICE.3: Edge Agent SL-4 Security
Файлы: backend/internal/edge/agent.go, backend/internal/edge/security.go

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

Effort: 5 days

Status: [ ]

📊 Success Metrics
Metric	Current	Target (Q4 2026)	Target (Q2 2027)
Overall Readiness	99%	99.5%	99.8%
Security Score	8.8/10	9.5/10	9.8/10
OWASP ASVS L3	88%	95%	100%
ISO 27001 controls	92%	98%	100%
СТБ 34.101.30	80%	100%	100%
Backend test coverage	82%	85%	90%
Frontend test coverage	75%	85%	90%
E2E tests (web)	21	50+	80+
E2E tests (mobile)	0	20+	30+
A11y violations	Unknown	0 critical	0 violations
Bundle size	2.8MB	<2MB	<1.5MB
Lighthouse score	87	>95	>98
FCP	1.8s	<1.5s	<1.2s
LCP	2.5s	<2.0s	<1.5s
Storybook coverage	14%	50%	90%+
Supported Regions	1 (BY)	3 (BY, EU, INTL)	7 (+RU, CN, US, KZ)
Certifications	0	0	2-3 (ОАЦ, ISO 27001)
Active Markets	1	4	12
ARR	Pilot	$2M	$8M
🗺️ Roadmap
Q3 2026 (July-September) — P0: Critical Fixes (10 weeks)
Week 1-2: P0-SEC.1 (СТБ Crypto), P0-SEC.2 (CORS), P0-SEC.3 (CSRF start), P0-SEC.4 (Module path)

Week 3-4: P0-SEC.1 (finish), P0-SEC.3 (finish), P0-MOBILE.1 (Conflict UI), P0-MOBILE.2 (Background sync)

Week 5-6: P0-MOBILE.3 (Offline maps), P0-MOBILE.4 (Mobile E2E start), P0-UX.1 (AddDeviceModal), P0-UX.2 (Breadcrumbs)

Week 7-8: P0-MOBILE.4 (finish), P0-UX.3 (View mode), P0-UX.4 (Kanban feedback), P0-CE.1 (ComplianceProfile), P0-CE.2 (Crypto providers)

Week 9-10: P0-CE.3 (Setup Wizard), P0-CE.4 (Tenant compliance), P0-CE.5 (Data residency), P0-BACKEND.1 (NATS mandatory), P0-BACKEND.2 (Schema validation), P0-BACKEND.3 (SMS), P0-BACKEND.4 (SLA escalation), P0-BACKEND.5 (Notifier fallback)

Q4 2026 (October-December) — P1: Close Competitive Gaps (12 weeks)
Week 11-12: P1-SEC.1 (2FA/WebAuthn), P1-SEC.2 (DLP), P1-SEC.3 (Rate limiting), P1-SEC.4 (Secrets rotation)

Week 13-14: P1-UX.1 (WorkOrders redesign), P1-UX.2 (SpareParts redesign), P1-UX.3 (Dashboard unification), P1-UX.4 (Skeleton)

Week 15-16: P1-UX.5 (Calendar toggle), P1-UX.6 (RCA widget), P1-UX.7 (Search unification), P1-UX.8 (Saved filters), P1-UX.9 (Bulk progress), P1-UX.10 (Tooltips), P1-UX.11 (Error handling)

Week 17-18: P1-PERF.1 (Bundle), P1-PERF.2 (Image lazy), P1-PERF.3 (React Query), P1-PERF.4 (Health checks), P1-PERF.5 (Redis), P1-PERF.6 (Graceful shutdown), P1-PERF.7 (Benchmarks), P1-PERF.8 (Redis pool)

Week 19-20: P1-QA.1 (E2E expansion), P1-QA.2 (A11y CI), P1-QA.3 (Sentry), P1-QA.4 (Lighthouse CI), P1-QA.5 (Load testing), P1-QA.6 (Coverage 80%), P1-QA.7 (Chaos)

Week 21-22: P1-BACKEND.1-5 (Backend quality) + P1-ARCH.1-5 (Architecture)

Q1 2027 (January-March) — P2: Enterprise Features (12 weeks)
Week 23-24: P2-AI.1 (ML model), P2-AI.2 (AI chat), P2-AI.3 (Predictive dashboard), P2-AI.4 (Anomaly detection)

Week 25-26: P2-WF.1 (Workflow builder), P2-WF.2 (Resource calendar), P2-WF.3 (Meter triggers), P2-WF.4 (Conditional checklists)

Week 27-28: P2-INV.1 (Auto parts), P2-INV.2 (Vendor scorecards), P2-INV.3 (Lifecycle cost), P2-INV.4 (Reorder automation)

Week 29-30: P2-INT.1 (Webhook builder), P2-INT.2 (Developer portal), P2-INT.3 (API versioning), P2-INT.4 (Bi-directional sync)

Week 31-32: P2-CR.1-4 (Regional policies) + P2-REGIONS (RU, EU, CN, US tasks)

Q2 2027 (April-June) — P3: Technical Debt (8 weeks)
Week 33-34: P3-SEC.1 (belt-GCM), P3-SEC.2 (bign JWT), P3-SEC.3 (Mobile pinning)

Week 35-36: P3-DX.1 (Storybook), P3-DX.2 (Onboarding), P3-DX.3 (Help), P3-DX.4 (DEVELOPMENT.md), P3-DX.5 (Swagger UI)

Week 37-38: P3-UI.1 (Design tokens), P3-UI.2 (Micro-interactions), P3-UI.3 (Mobile responsiveness)

Week 39-40: P3-NICE.1 (Real-time collab), P3-NICE.2 (White-label), P3-NICE.3 (Edge agent)

📝 История изменений
Дата	Версия	Изменения
2026-06-28	v2.0	Интеграция выводов комплексного Code Review (DevSecOps, QA, UX/UI). Добавлены задачи: P1-UX.11 (Error Handling UI), P1-PERF.7 (Benchmark Suite), P1-PERF.8 (Redis Connection Pool), P1-QA.7 (Chaos Engineering). Уточнены критерии приёмки и зависимости.
2026-06-27	v1.0	Первоначальная версия от пользователя