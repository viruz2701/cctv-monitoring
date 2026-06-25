# TODO.md — CCTV Health Monitor
> Living document. Roo использует этот файл как основной roadmap.
> Обновлять после завершения каждой задачи: [ ] → [x] + дата.

**Последнее обновление:** 2026-06-26
**Общая готовность:** 86%

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

---

## 🔴 P0 — Критично (Q3 2026, до 2026-09-30)

### P0-1: Security & Data Integrity

#### P0-1.1: Schema Registry Validation
- **Файлы:** `backend/internal/events/schema_registry.go`, `backend/internal/events/publisher.go`
- **Проблема:** `SchemaRegistry.Validate()` закомментирована → риск записи невалидных событий
- **Решение:** 
  - Добавить `github.com/xeipuuv/gojsonschema` в `go.mod`
  - Реализовать валидацию в `publisher.go` перед publish
  - Добавить middleware для проверки всех событий
- **Критерий приёмки:**
  - [ ] Валидация включена в production config
  - [ ] Тесты покрывают valid/invalid scenarios
  - [ ] Error logging для failed validations
- **Effort:** 2d
- **Status:** [ ]

#### P0-1.2: NATS JetStream Mandatory
- **Файлы:** `backend/config.yaml`, `backend/internal/state/state_manager.go`, `backend/main.go`
- **Проблема:** `InMemory` state manager в production не шардится между подами
- **Решение:**
  - Убрать `InMemory` fallback из production config
  - Добавить health check для NATS connectivity
  - Graceful shutdown при недоступности JetStream
- **Критерий приёмки:**
  - [ ] Production config требует JetStream
  - [ ] Startup fails если JetStream недоступен
  - [ ] `/health/ready` endpoint проверяет NATS
- **Effort:** 1d
- **Status:** [ ]

#### P0-1.3: SLA Escalation Integration
- **Файлы:** `backend/internal/sla/engine.go`, `backend/internal/notifications/sla_breach_notifier.go`
- **Проблема:** Таблицы `sla_escalation_rules` + `sla_escalation_log` есть, но не интегрированы
- **Решение:**
  - Добавить вызов `escalationNotifier.Notify()` при breach
  - Интегрировать с Telegram/SMS/Email
  - Записывать в `sla_escalation_log` для audit trail
- **Критерий приёмки:**
  - [ ] Escalation срабатывает при breach
  - [ ] Уведомления отправляются через все каналы
  - [ ] Audit log содержит все escalation events
- **Effort:** 2d
- **Status:** [ ]

#### P0-1.4: Event Schema Registry Audit
- **Файлы:** `backend/internal/events/schema_registry.go`, `backend/internal/events/publisher.go`
- **Проблема:** Нет validation при publish → могут быть записаны невалидные события
- **Решение:**
  - Добавить middleware в NATS publisher
  - Логировать все failed validations с context
  - Добавить metrics для failed/successful validations
- **Критерий приёмки:**
  - [ ] Middleware активен для всех publishers
  - [ ] Prometheus metrics для validation stats
  - [ ] Error содержит full event payload для debugging
- **Effort:** 1d
- **Status:** [ ]

---

### P0-2: UX Navigation Restructuring

#### P0-2.1: Sidebar Consolidation
- **Файлы:** `frontend/src/components/layout/Sidebar.tsx`, `frontend/src/hooks/useNavigation.ts`
- **Проблема:** 20+ пунктов первого уровня → cognitive overload
- **Решение:**
  - Группировать в 5 parents: Dashboard, Assets, Operations, Insights, Administration
  - Добавить collapsible groups
  - Сохранять expanded state в localStorage
- **Критерий приёмки:**
  - [ ] Sidebar показывает только 5 родителей
  - [ ] Дочерние пункты раскрываются по клику
  - [ ] Expanded state сохраняется между сессиями
  - [ ] Keyboard navigation работает (Arrow keys)
- **Effort:** 3d
- **Status:** [ ]

#### P0-2.2: Role-Based Navigation Filtering
- **Файлы:** `frontend/src/hooks/useNavigation.ts`, `frontend/src/store/authStore.ts`
- **Проблема:** Technician видит те же пункты, что и Admin
- **Решение:**
  - Создать `useNavigation(role)` хук
  - Фильтровать `allNavItems` по ролям
  - Добавить `NavItemPermission` интерфейс
- **Критерий приёмки:**
  - [ ] Technician не видит Administration
  - [ ] Viewer видит только read-only пункты
  - [ ] Admin видит всё
  - [ ] Unit тесты для всех ролей
- **Effort:** 2d
- **Status:** [ ]

#### P0-2.3: Quick Access Bar
- **Файлы:** `frontend/src/components/layout/Sidebar.tsx`, `frontend/src/store/workspaceStore.ts`
- **Проблема:** Частые действия требуют 3+ кликов
- **Решение:**
  - Закрепить 3-4 пункта сверху sidebar
  - Позволить пользователю кастомизировать
  - Сохранять в `user_preferences`
- **Критерий приёмки:**
  - [ ] Quick access bar виден всегда
  - [ ] Drag-and-drop для reordering
  - [ ] Settings для добавления/удаления пунктов
  - [ ] Сохраняется в backend
- **Effort:** 1d
- **Status:** [ ]

---

## 🎨 P1 — High Priority (Q4 2026, до 2026-12-31)

### P1-1: Dashboard Consolidation

#### P1-1.1: Unified Dashboard Hub
- **Файлы:** `frontend/src/pages/Dashboard.tsx`, `frontend/src/components/dashboard/DashboardTabs.tsx`
- **Проблема:** 5+ разрозненных дашбордов → дублирование метрик
- **Решение:**
  - Создать единую страницу `/dashboard`
  - Добавить tabs: Overview, SLA & Compliance, Performance, Maintenance
  - Lazy-load widgets per tab
- **Критерий приёмки:**
  - [ ] Одна страница вместо 5
  - [ ] Tabs переключаются без reload
  - [ ] URL sync: `/dashboard?view=sla`
  - [ ] Loading skeleton per widget
- **Effort:** 4d
- **Status:** [ ]

#### P1-1.2: Role-Based Default Views
- **Файлы:** `frontend/src/pages/Dashboard.tsx`, `frontend/src/store/workspaceStore.ts`
- **Проблема:** Пользователи переключаются между дашбордами
- **Решение:**
  - Auto-detect role → set default tab
  - Technician → "My Work"
  - Manager → "Overview"
  - Admin → "System Health"
- **Критерий приёмки:**
  - [ ] Default view зависит от роли
  - [ ] Пользователь может override
  - [ ] Выбор сохраняется в profile
  - [ ] A/B тестирование для optimization
- **Effort:** 2d
- **Status:** [ ]

#### P1-1.3: Widget Registry & Saved Views
- **Файлы:** `frontend/src/components/dashboard/WidgetRegistry.ts`, `frontend/src/store/savedViewsStore.ts`
- **Проблема:** Нет возможности сохранить custom layout
- **Решение:**
  - Создать `WidgetRegistry` с metadata
  - Добавить `SavedViews` store
  - Сохранять в `user_preferences` + localStorage fallback
- **Критерий приёмки:**
  - [ ] Пользователь может сохранить layout
  - [ ] Saved views доступны в dropdown
  - [ ] Share view с другими пользователями
  - [ ] Import/export view как JSON
- **Effort:** 3d
- **Status:** [ ]

#### P1-1.4: Dashboard Multi-Device Sync
- **Файлы:** `frontend/src/store/workspaceStore.ts`, `backend/internal/api/workspace_handlers.go`
- **Проблема:** Layout сохраняется только в localStorage → не sync между устройствами
- **Решение:**
  - Сохранять layout в БД с привязкой к `workspace_id`
  - Sync при login на новом устройстве
  - Conflict resolution: last-write-wins
- **Критерий приёмки:**
  - [ ] Layout sync между desktop и tablet
  - [ ] Conflict resolution UI при необходимости
  - [ ] Offline queue для layout changes
  - [ ] Version history для layout
- **Effort:** 2d
- **Status:** [ ]

---

### P1-2: Error Handling Strategy

#### P1-2.1: Unified Error Boundary System
- **Файлы:** `frontend/src/components/layout/RouteErrorBoundary.tsx`, `frontend/src/components/dashboard/WidgetErrorBoundary.tsx`
- **Проблема:** Нет global error handling → crash всей страницы
- **Решение:**
  - Добавить `RouteErrorBoundary` для pages
  - Добавить `WidgetErrorBoundary` для widgets
  - Интегрировать с Sentry/OTel
- **Критерий приёмки:**
  - [ ] Crash widget не ломает весь dashboard
  - [ ] Error boundary показывает retry button
  - [ ] Error context отправляется в telemetry
  - [ ] User-friendly error messages
- **Effort:** 2d
- **Status:** [ ]

#### P1-2.2: API Error Mapper
- **Файлы:** `frontend/src/services/apiErrorMapper.ts`, `frontend/src/services/api.ts`
- **Проблема:** Разные форматы ошибок от backend → inconsistent UX
- **Решение:**
  - Создать `apiErrorMapper` → `{ type, message, field?, retryable, action? }`
  - Unified error format для всех endpoints
  - Automatic retry для retryable errors
- **Критерий приёмки:**
  - [ ] Все API errors проходят через mapper
  - [ ] Inline errors для form fields
  - [ ] Toast для global errors
  - [ ] Retry logic для 5xx и network errors
- **Effort:** 2d
- **Status:** [ ]

#### P1-2.3: Offline Queue UI
- **Файлы:** `frontend/src/components/layout/OfflineBanner.tsx`, `frontend/src/components/layout/QueueModal.tsx`
- **Проблема:** Нет visual indicator offline mode + queue count
- **Решение:**
  - Persistent banner: "Offline mode. 3 operations queued."
  - Queue modal с conflict resolution
  - Exponential backoff visualization
- **Критерий приёмки:**
  - [ ] Banner появляется при offline
  - [ ] Queue count обновляется в real-time
  - [ ] Modal показывает все pending operations
  - [ ] Manual retry button для failed operations
- **Effort:** 3d
- **Status:** [ ]

#### P1-2.4: Business Rule Validation
- **Файлы:** `frontend/src/lib/businessRuleValidator.ts`, `frontend/src/components/work-orders/WorkOrderForm.tsx`
- **Проблема:** Нет client-side validation для бизнес-правил
- **Решение:**
  - Добавить `BusinessRuleValidator` с inline warnings
  - Disabled submit при violations
  - Tooltips с explanation
- **Критерий приёмки:**
  - [ ] SLA pause warning при status change
  - [ ] Gatekeeper fail warning
  - [ ] Checklist incomplete warning
  - [ ] Inventory shortage warning
- **Effort:** 2d
- **Status:** [ ]

---

### P1-3: Mobile Experience Enhancement

#### P1-3.1: Conflict Resolution UI
- **Файлы:** `mobile/src/components/ConflictResolutionModal.tsx`, `mobile/src/services/syncService.ts`
- **Проблема:** Нет UI для offline sync conflicts
- **Решение:**
  - Modal с diff-view: "Local vs Server"
  - Кнопки: Keep Local, Keep Server, Merge
  - Visual diff highlighting
- **Критерий приёмки:**
  - [ ] Modal появляется при conflict
  - [ ] Diff view показывает изменения
  - [ ] Merge работает для text fields
  - [ ] Conflict logged в telemetry
- **Effort:** 3d
- **Status:** [ ]

#### P1-3.2: BeforeAfterSlider in Mobile
- **Файлы:** `mobile/src/screens/WorkOrderDetailScreen.tsx`, `mobile/src/components/BeforeAfterSlider.tsx`
- **Проблема:** Mobile app не использует BeforeAfterSlider для Gatekeeper
- **Решение:**
  - Добавить `BeforeAfterSlider` в verification flow
  - Pinch-to-zoom для photos
  - Annotation tools (arrows, circles)
- **Критерий приёмки:**
  - [ ] Slider работает в Gatekeeper flow
  - [ ] Zoom и pan работают smoothly
  - [ ] Annotations сохраняются
  - [ ] Export annotated photo
- **Effort:** 2d
- **Status:** [ ]

#### P1-3.3: Push Notifications for SLA Breach
- **Файлы:** `mobile/src/utils/notifications.ts`, `backend/internal/notifications/sla_breach_notifier.go`
- **Проблема:** Нет push-уведомлений для критических событий
- **Решение:**
  - Интегрировать `expo-notifications`
  - Deep linking в notification
  - User preferences для notification types
- **Критерий приёмки:**
  - [ ] Push приходит при SLA breach
  - [ ] Tap открывает WorkOrderDetail
  - [ ] User can disable specific types
  - [ ] Badge count на app icon
- **Effort:** 2d
- **Status:** [ ]

---

### P1-4: Performance & Accessibility

#### P1-4.1: Heavy Component Memoization
- **Файлы:** `frontend/src/pages/WorkOrderCalendar.tsx`, `frontend/src/components/ui/DataGrid.tsx`
- **Проблема:** `WorkOrderCalendar` (FullCalendar) перерендеривается часто
- **Решение:**
  - Обернуть в `React.memo`
  - `useMemo` для props
  - `useCallback` для handlers
- **Критерий приёмки:**
  - [ ] React DevTools Profiler показывает <5 re-renders
  - [ ] No performance regression
  - [ ] Unit тесты проходят
  - [ ] Lighthouse score >90
- **Effort:** 1d
- **Status:** [ ]

#### P1-4.2: Critical Error aria-live
- **Файлы:** `frontend/src/hooks/useAccessibility.ts`, `frontend/src/components/ui/Toast.tsx`
- **Проблема:** Нет `aria-live="assertive"` для критических ошибок
- **Решение:**
  - Добавить в `announce()`: `aria-live="assertive"` для ошибок
  - `aria-live="polite"` для warnings
  - Focus management при error
- **Критерий приёмки:**
  - [ ] Screen reader читает ошибки сразу
  - [ ] Focus переходит на error summary
  - [ ] Color contrast >4.5:1
  - [ ] WCAG 2.1 AA compliance
- **Effort:** 1d
- **Status:** [ ]

#### P1-4.3: Toast Deduplication
- **Файлы:** `frontend/src/store/alertStore.ts`, `frontend/src/components/ui/Toast.tsx`
- **Проблема:** Одинаковые toast-сообщения дублируются
- **Решение:**
  - Deduplication по `title + message`
  - Counter для одинаковых toasts
  - Collapse после 3 одинаковых
- **Критерий приёмки:**
  - [ ] Одинаковые toasts не дублируются
  - [ ] Counter показывает количество
  - [ ] Collapse работает после 3
  - [ ] Unit тесты для deduplication
- **Effort:** 1d
- **Status:** [ ]

---

## 🏢 P2 — Enterprise Features (Q1 2027, до 2027-03-31)

### P2-1: Advanced Analytics & AI

#### P2-1.1: Real ML Model Integration
- **Файлы:** `backend/analytics/predict.py`, `backend/internal/ml/prediction_service.go`
- **Проблема:** Python-скрипты обучаются на синтетических данных
- **Решение:**
  - Интегрировать XGBoost на реальных данных
  - Publish predictions через NATS
  - Add prediction confidence score
- **Критерий приёмки:**
  - [ ] Model обучена на production data
  - [ ] Predictions публикуются в NATS
  - [ ] Confidence score >75%
  - [ ] A/B testing для model validation
- **Effort:** 5d
- **Status:** [ ]

#### P2-1.2: AI Assistant in UI
- **Файлы:** `frontend/src/components/ai/AIAssistantPanel.tsx`, `backend/internal/ai/deepseek_client.go`
- **Проблема:** Нет контекстных подсказок для техников
- **Решение:**
  - Chat-панель с DeepSeek integration
  - Context-aware recommendations
  - RCA suggestions
- **Критерий приёмки:**
  - [ ] Panel доступен во всех work orders
  - [ ] Recommendations релевантны контексту
  - [ ] Response time <2s
  - [ ] User feedback mechanism
- **Effort:** 4d
- **Status:** [ ]

#### P2-1.3: Predictive Maintenance Dashboard
- **Файлы:** `frontend/src/pages/PredictiveMaintenance.tsx`, `frontend/src/components/dashboard/PredictiveWidget.tsx`
- **Проблема:** Нет визуализации at-risk devices
- **Решение:**
  - KPI cards с at-risk count
  - Risk distribution chart
  - Failure by type breakdown
- **Критерий приёмки:**
  - [ ] Dashboard показывает at-risk devices
  - [ ] Drill-down в device detail
  - [ ] Export to PDF/Excel
  - [ ] Email digest для managers
- **Effort:** 3d
- **Status:** [ ]

---

### P2-2: Workflow & Automation

#### P2-2.1: Workflow Builder UI
- **Файлы:** `frontend/src/components/workflow/WorkflowBuilder.tsx`, `backend/internal/workflow/engine.go`
- **Проблема:** `workflow/engine.go` есть, но нет UI-конструктора
- **Решение:**
  - React Flow для drag&drop
  - CEL conditions editor
  - Workflow testing mode
- **Критерий приёмки:**
  - [ ] Drag&drop nodes работают
  - [ ] CEL editor с syntax highlighting
  - [ ] Test mode с mock data
  - [ ] Version control для workflows
- **Effort:** 5d
- **Status:** [ ]

#### P2-2.2: Smart Command Palette Search
- **Файлы:** `frontend/src/components/CommandPalette.tsx`, `backend/internal/api/search_handlers.go`
- **Проблема:** Поиск только по entities, не по тексту WO
- **Решение:**
  - Полнотекстовый поиск через `pg_trgm`
  - Поиск по заголовку, описанию, серийным номерам
  - Fuzzy matching для typos
- **Критерий приёмки:**
  - [ ] Поиск по тексту WO работает
  - [ ] Fuzzy matching для typos
  - [ ] Results ranked по relevance
  - [ ] Search analytics для optimization
- **Effort:** 3d
- **Status:** [ ]

#### P2-2.3: Resource Planning Calendar
- **Файлы:** `frontend/src/pages/TechnicianWeek.tsx`, `backend/internal/workforce/scheduler.go`
- **Проблема:** Нет календаря загрузки техников
- **Решение:**
  - Week view с technician rows
  - Drag-and-drop WO assignment
  - Conflict detection
- **Критерий приёмки:**
  - [ ] Week view показывает загрузку
  - [ ] Drag&drop для reassignment
  - [ ] Conflict warning при overlap
  - [ ] Print-friendly view
- **Effort:** 4d
- **Status:** [ ]

---

### P2-3: Integration Ecosystem

#### P2-3.1: Webhook Builder UI
- **Файлы:** `frontend/src/components/webhooks/WebhookBuilder.tsx`, `backend/internal/webhooks/manager.go`
- **Проблема:** Есть `webhook/verify.go`, но нет visual builder
- **Решение:**
  - Event type selector
  - Payload preview
  - Test button с mock event
- **Критерий приёмки:**
  - [ ] Builder создает webhooks
  - [ ] Payload preview работает
  - [ ] Test mode отправляет mock event
  - [ ] Delivery logs для debugging
- **Effort:** 3d
- **Status:** [ ]

#### P2-3.2: OAuth2 for External Adapters
- **Файлы:** `backend/internal/cmms/servicenow/client.go`, `backend/internal/cmms/jira/client.go`
- **Проблема:** ServiceNow/Jira адаптеры используют basic auth
- **Решение:**
  - OAuth2 flow implementation
  - Token refresh logic
  - Secure token storage
- **Критерий приёмки:**
  - [ ] OAuth2 flow работает
  - [ ] Token auto-refresh
  - [ ] Secure storage (encrypted)
  - [ ] Fallback to basic auth
- **Effort:** 3d
- **Status:** [ ]

#### P2-3.3: Excel Import/Export for WO
- **Файлы:** `frontend/src/pages/WorkOrders.tsx`, `backend/internal/reports/excel_handler.go`
- **Проблема:** Есть `export_handlers.go`, но нет UI-кнопки "Export all"
- **Решение:**
  - Bulk export button
  - Import wizard для Excel
  - Column mapping UI
- **Критерий приёмки:**
  - [ ] Export all работает для 10k+ WO
  - [ ] Import wizard с preview
  - [ ] Column mapping с auto-detect
  - [ ] Error report для failed imports
- **Effort:** 2d
- **Status:** [ ]

---

## 🔧 P3 — Technical Debt (Q2 2027, до 2027-06-30)

### P3-1: Security & Compliance

#### P3-1.1: belt-GCM Migration (СТБ 34.101.31)
- **Файлы:** `backend/internal/crypto/aes.go`, `backend/internal/crypto/belt.go`
- **Проблема:** Используется AES-256-GCM, для КИИ РБ требуется belt-GCM
- **Решение:**
  - Мигрировать на `github.com/bp2012/crypto/belt`
  - Backward compatibility для existing data
  - Migration script для encrypted data
- **Критерий приёмки:**
  - [ ] belt-GCM используется для new data
  - [ ] Migration script работает
  - [ ] Performance benchmarks
  - [ ] Security audit passed
- **Effort:** 4d
- **Status:** [ ]

#### P3-1.2: JWT → HttpOnly Cookies
- **Файлы:** `backend/internal/auth/jwt.go`, `frontend/src/services/auth.ts`, `mobile/src/services/auth.ts`
- **Проблема:** JWT хранится в localStorage → XSS risk
- **Решение:**
  - HttpOnly cookies для web
  - Secure flag + SameSite
  - CSRF tokens
- **Критерий приёмки:**
  - [ ] HttpOnly cookies работают
  - [ ] CSRF protection активна
  - [ ] Mobile app адаптирована
  - [ ] Penetration test passed
- **Effort:** 6d
- **Status:** [ ]

#### P3-1.3: OpenTelemetry Integration
- **Файлы:** `backend/internal/telemetry/otel.go`, `frontend/src/lib/telemetry.ts`
- **Проблема:** Нет distributed tracing
- **Решение:**
  - OpenTelemetry SDK
  - Trace context propagation
  - Jaeger/Zipkin integration
- **Критерий приёмки:**
  - [ ] Traces отправляются в collector
  - [ ] Trace ID в logs
  - [ ] Distributed tracing работает
  - [ ] Performance impact <5%
- **Effort:** 3d
- **Status:** [ ]

---

### P3-2: Performance & Scalability

#### P3-2.1: Materialized View Auto-Refresh
- **Файлы:** `backend/internal/maintenance/cron.go`, `backend/migrations/*.sql`
- **Проблема:** `mv_device_reliability` + `mv_tco_per_device` обновляются вручную
- **Решение:**
  - Cron job для `REFRESH MATERIALIZED VIEW CONCURRENTLY`
  - Staleness monitoring
  - Alert при refresh failure
- **Критерий приёмки:**
  - [ ] Cron job запускается hourly
  - [ ] Refresh занимает <5min
  - [ ] Alert при failure
  - [ ] Staleness metric в Prometheus
- **Effort:** 2d
- **Status:** [ ]

#### P3-2.2: Virtual Table Auto-Selection
- **Файлы:** `frontend/src/components/ui/Table.tsx`, `frontend/src/components/ui/VirtualTable.tsx`
- **Проблема:** `VirtualTable` используется вручную
- **Решение:**
  - Auto-selection на основе `rowCount > 1000`
  - Seamless fallback
  - Performance monitoring
- **Критерий приёмки:**
  - [ ] Auto-selection работает
  - [ ] No UX degradation
  - [ ] Performance metrics logged
  - [ ] Unit тесты для обоих режимов
- **Effort:** 2d
- **Status:** [ ]

#### P3-2.3: Bundle Size Optimization
- **Файлы:** `frontend/vite.config.ts`, `frontend/rollup.config.js`
- **Проблема:** Нет анализа chunk sizes
- **Решение:**
  - `rollup-plugin-visualizer` в CI
  - Alert если chunk > 500KB
  - Code splitting optimization
- **Критерий приёмки:**
  - [ ] Visualizer в CI pipeline
  - [ ] Alert для large chunks
  - [ ] Bundle size <2MB
  - [ ] Lighthouse performance >90
- **Effort:** 1d
- **Status:** [ ]

---

### P3-3: Developer Experience

#### P3-3.1: Onboarding Tour Role Adaptation
- **Файлы:** `frontend/src/components/OnboardingTour.tsx`, `frontend/src/store/authStore.ts`
- **Проблема:** `react-joyride` шаги статичны
- **Решение:**
  - Адаптировать шаги под роль
  - Conditional steps
  - Skip option для experienced users
- **Критерий приёмки:**
  - [ ] Technician видит только relevant steps
  - [ ] Admin видит все steps
  - [ ] Skip button работает
  - [ ] Tour completion tracked
- **Effort:** 2d
- **Status:** [ ]

#### P3-3.2: Power User Keyboard Shortcuts
- **Файлы:** `frontend/src/hooks/useKeyboardShortcuts.ts`, `frontend/src/components/layout/Layout.tsx`
- **Проблема:** Нет горячих клавиш для быстрого переключения WO
- **Решение:**
  - `Alt+1..9` для 9 последних WO
  - `/` для focus на search
  - `?` для shortcut help
- **Критерий приёмки:**
  - [ ] Shortcuts работают globally
  - [ ] Help modal с all shortcuts
  - [ ] Customizable shortcuts
  - [ ] No conflicts с browser shortcuts
- **Effort:** 2d
- **Status:** [ ]

#### P3-3.3: Real-time Collaboration
- **Файлы:** `frontend/src/pages/WorkOrderDetail.tsx`, `backend/internal/ws/hub.go`
- **Проблема:** Нет WebSocket presence indicators
- **Решение:**
  - WebSocket для presence
  - "Ivan is editing this WO" indicator
  - Conflict warning при concurrent edit
- **Критерий приёмки:**
  - [ ] Presence indicator работает
  - [ ] Real-time updates
  - [ ] Conflict warning
  - [ ] Graceful degradation при WS failure
- **Effort:** 4d
- **Status:** [ ]

---

## 📊 Метрики успеха

| Метрика | Текущее | Target (Q4 2026) | Измерение |
|---------|---------|------------------|-----------|
| **Time-to-Task** | ~30s | ~18s (-40%) | User testing, session recordings |
| **Navigation Clicks** | 3-5 | 1-2 (-60%) | Analytics, heatmaps |
| **Dashboard Load Time** | 2.5s | <1.5s (-40%) | Lighthouse, WebPageTest |
| **Error Recovery Time** | ~2min | ~30s (-75%) | Support tickets, user feedback |
| **Mobile Offline Success Rate** | 85% | 95% (+10%) | Sync logs, conflict resolution rate |
| **SLA Breach Detection** | Manual | Automatic (100%) | SLA logs, escalation notifications |
| **AI Prediction Accuracy** | N/A | >75% | ML model validation |
| **Bundle Size** | 2.8MB | <2MB | rollup-plugin-visualizer |
| **Lighthouse Score** | 87 | >95 | Lighthouse CI |
| **WCAG Compliance** | 90% | 100% | axe-core, manual audit |

---

## 📝 История изменений

### 2026-06-26 — Initial TODO Creation
- Создан unified TODO на основе двух code reviews
- Определены P0-P3 приоритеты
- Добавлены критерии приёмки для всех задач
- Установлены метрики успеха

### [Дата] — [Описание изменения]
- [Детали изменения]
- [Связанные коммиты]
- [Impact на метрики]

---

## 🔗 Полезные ссылки

- **Architecture:** `ARCHITECTURE.md`
- **UX Guidelines:** `docs/ux/ux-guideline.md`
- **ADR Log:** `docs/adr/`
- **API Docs:** `backend/docs/api/`
- **Design System:** `frontend/.storybook/`
- **CI/CD:** `.github/workflows/`

---

**Последний коммит:** `HEAD`
**Branch:** `main`
**Next Review:** 2026-07-03