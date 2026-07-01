# Frontend (React 19 + TypeScript 5.9 + Vite 8 + TailwindCSS v4) и Mobile (React Native + Expo 52)

> **Проект:** CCTV Health Monitor  
> **Версия:** 1.0.0  
> **Дата:** 2026-07-01  
> **Compliance:** СТБ IEC 62443 SL-3, ISO 27001:2022, OWASP ASVS L3, Приказ ОАЦ №66

---

## Frontend (React 19 + TypeScript 5.9 + Vite 8 + TailwindCSS v4)

### 1. Архитектура

#### 1.1 Route-based Code Splitting (P0-CR-06)

Фронтенд построен на трёхуровневой архитектуре **lazy-loading**, где каждый уровень — отдельный chunk, загружаемый по требованию:

```
main.tsx (6KB — точка входа)
  └── AppProviders.tsx (Sentry + React Query + Theme + Auth + Toast)
       └── AppShell.tsx (Layout + BrowserRouter + все Routes)
            ├── Login, ForgotPassword — публичные
            ├── DashboardHub, Sites, Devices, WorkOrders... — защищённые
            ├── AgentDashboard, AgentDetail — EDGE-11
            ├── DescriptorEditor — PROTO-06
            ├── CommunityRegistry — PROTO-07
            └── BIQueryBuilder — P2-BI
```

**Механизм:**

- [`main.tsx`](frontend/src/main.tsx:1) — точка входа: `createRoot` + `StrictMode`, регистрация Service Worker. Всё остальное — динамический импорт через `import()`.
- [`AppProviders.tsx`](frontend/src/components/AppProviders.tsx:1) — динамический chunk с провайдерами: `SentryErrorBoundary` → `QueryClientProvider` → `ThemeProvider` → `ToastProvider` → `AuthProvider`.
- [`AppShell.tsx`](frontend/src/components/AppShell.tsx:1) — динамический chunk с `BrowserRouter`, `Layout` и всеми `Route` (45+ lazy-страниц).

**Bundle sizes (estimated):**
| Чанк | Размер |
|------|--------|
| Main (main.tsx) | ~6 KB |
| Providers (AppProviders) | ~28 KB |
| Shell + Layout | ~48 KB |
| Total initial | ~82 KB |
| Каждая страница | <100 KB |
| **Main bundle (total)** | **~195 KB** |

#### 1.2 Lazy Loading

Все 45+ страниц загружаются через `React.lazy` + `Suspense`:

```typescript
const DashboardHub = lazy(() => import('../pages/DashboardHub')
  .then((m) => ({ default: m.DashboardHub })));
```

- Каждая страница — отдельный chunk (Rolldown/Vite code splitting).
- [`PageSuspense`](frontend/src/components/layout/PageSuspense.tsx:1) — единый Fallback с skeleton-загрузкой.
- [`SkeletonPage`](frontend/src/components/layout/SkeletonPage.stories.tsx:1) — Storybook-компонент для предпросмотра skeleton-состояний.
- [`RouteErrorBoundary`](frontend/src/components/layout/RouteErrorBoundary.tsx:1) — per-route error boundary с retry.

#### 1.3 Provider Tree

```
<SentryErrorBoundary>           // Sentry + fallback UI
  <QueryClientProvider>          // @tanstack/react-query (staleTime: 30s, retry: 2)
    <ThemeProvider>              // Zustand theme store + CSS переменные
      <ToastProvider>            // Система уведомлений (Toast)
        <AuthProvider>           // JWT auth + /users/me check
          <Suspense fallback={null}>
            <AppShell />         // BrowserRouter + Routes
          </Suspense>
        </AuthProvider>
      </ToastProvider>
    </ThemeProvider>
  </QueryClientProvider>
</SentryErrorBoundary>
```

---

### 2. Технический стек

| Категория | Технология | Назначение |
|-----------|-----------|------------|
| **Ядро** | React 19 | UI-компоненты, Server Components, Actions |
| **Сборка** | Vite 8 + Rolldown | Сборка, HMR, code splitting |
| **Язык** | TypeScript 5.9 | Типизация, strict mode |
| **Стили** | TailwindCSS v4 + CSS tokens | Дизайн-система, темизация |
| **Стейт UI** | Zustand 5 | 16+ stores (client-side state) |
| **Стейт сервера** | @tanstack/react-query 5 | Кэширование API, мутации |
| **Роутинг** | react-router-dom v7 | SPA routing, useBlocker |
| **Интернационализация** | i18next 26 + react-i18next 17 | 20 языков |
| **Виртуализация** | @tanstack/react-virtual 3 | DataGrid, большие списки |
| **Формы** | react-hook-form 7 + Zod 3 | Валидация форм |
| **Чарты** | @nivo (bar, line, pie, heatmap) | Дашборды, SLA |
| **Календарь** | @schedule-x/calendar | Расписание техников |
| **DnD** | @hello-pangea/dnd | Drag-and-drop дашборды |
| **Графы** | @xyflow/react | Workflow Builder |
| **PWA** | vite-plugin-pwa + Workbox | Service Worker, offline |
| **Мониторинг** | @sentry/react 9 | Error tracking, Replay |
| **Терминал** | @xterm/xterm | Edge Agent terminal |
| **Видео** | hls.js | Видеопотоки с камер |
| **QR** | qrcode.react | QR-коды для устройств |
| **Эксель** | exceljs | Экспорт отчётов |
| **Онбординг** | react-joyride | Тур по приложению |

---

### 3. Структура компонентов

```
src/components/
├── ui/                  # 40+ базовых UI-компонентов
│   ├── Button.tsx, Input.tsx, Card.tsx, Modal.tsx
│   ├── Table.tsx, DataGrid.tsx
│   ├── Badge.tsx, Alert.tsx, Toast.tsx, Notification.tsx
│   ├── Gauge.tsx, SLAProgress.tsx, LiveSLATimer.tsx
│   ├── ProgressBar.tsx, StatsCard.tsx
│   ├── Dropdown.tsx, Tabs.tsx, Tooltip.tsx, InfoTooltip.tsx
│   ├── Breadcrumbs.tsx, Timeline.tsx
│   ├── EmptyState.tsx, Skeleton.tsx
│   ├── FileUpload.tsx, ImportWizard.tsx
│   ├── AdvancedSearch.tsx, SavedFiltersDropdown.tsx, SavedViews.tsx
│   ├── LazyImage.tsx, QRCode.tsx
│   ├── ThemeCustomizer.tsx, WhiteLabelCustomizer.tsx
│   ├── OnboardingTour.tsx, VideoTutorialCard.tsx
│   ├── CommandPalette.tsx, ShortcutsCheatsheet.tsx
│   ├── VisuallyHidden.tsx, BulkProgressModal.tsx
│   ├── MapModal.tsx, WorkOrderPrintView.tsx
│   ├── VirtualTable.tsx
│   └── index.ts (barrel export)
│
├── layout/              # Компоновка страниц
│   ├── Layout.tsx, Sidebar.tsx, Header.tsx
│   ├── ThreeColumnTemplate.tsx
│   ├── PageSuspense.tsx, SkeletonPage.tsx
│   ├── RouteErrorBoundary.tsx, ErrorBoundaryLite.tsx
│   ├── OfflineBanner.tsx
│   ├── KeyboardShortcutsHelp.tsx
│   ├── QueueModal.tsx, WorkspaceSwitcher.tsx
│   └── index.ts (barrel export)
│
├── dashboard/           # Виджеты дашборда
│   ├── DragDropDashboard.tsx
│   ├── AlertBanner.tsx
│   ├── WidgetErrorBoundary.tsx, WidgetRegistry.ts
│   └── tabs/
│       ├── OverviewTab.tsx
│       ├── MaintenanceTab.tsx
│       ├── PerformanceTab.tsx
│       └── SLAComplianceTab.tsx
│
├── auth/                # Аутентификация и RBAC
│   ├── PermissionGuard.tsx
│   ├── RoleProtectedRoute.tsx
│   └── WebAuthnSetup.tsx
│
├── work-orders/         # Work Orders (CMMS)
│   ├── WODataGrid.tsx
│   ├── WODetailInfo.tsx, WODetailTime.tsx
│   ├── WODetailParts.tsx, WODetailPhotos.tsx
│   └── PhotoAnnotation.tsx
│
├── sla/                 # SLA-совместимость
│   ├── SLABreachTimeline.tsx
│   ├── SLAGaugePanel.tsx
│   ├── SLAHeatmap.tsx
│   └── SLATrendChart.tsx
│
├── rca/                 # Root Cause Analysis
│   ├── RCAWidget.tsx
│   └── RCAGraph.tsx
│
├── agents/              # Edge Agents (EDGE-11)
│   ├── AgentStatsCard.tsx
│   ├── AgentStatusBadge.tsx
│   └── AgentTable.tsx
│
├── workflow/            # Workflow Engine
│   ├── WorkflowBuilder.tsx
│   ├── WorkflowNode.tsx
│   ├── WorkflowCELInput.tsx
│   ├── WorkflowTestPanel.tsx
│   └── WorkflowToolbar.tsx
│
├── webhooks/            # Webhook Management
│   ├── WebhookBuilder.tsx
│   ├── WebhookLogFilter.tsx
│   ├── WebhookRetryPolicy.tsx
│   ├── WebhookStatsCards.tsx
│   └── HmacVerificationHelper.tsx
│
├── descriptors/         # Protocol Descriptors (PROTO-06)
│   ├── DescriptorForm.tsx
│   ├── DescriptorPreview.tsx
│   ├── DescriptorTester.tsx
│   └── EndpointEditor.tsx
│
├── spare-parts/         # Запчасти
│   ├── PartCard.tsx
│   ├── PartHistoryTimeline.tsx
│   └── PartsGridView.tsx
│
├── molecules/           # Составные молекулы
│   ├── DateRangePicker.tsx
│   ├── PriorityPicker.tsx
│   ├── SLAProgressBar.tsx
│   └── TechnicianSelector.tsx
│
├── organisms/           # Организмы
│   ├── AssetTree.tsx
│   └── BeforeAfterSlider.tsx
│
├── reports/             # Отчёты
│   ├── ManualDownloadTab.tsx
│   ├── ReportHistoryTab.tsx
│   └── ScheduledReportsTab.tsx
│
├── chat/                # Чат
│   └── WOChat.tsx
│
├── ai/                  # AI Assistant
│   └── AIAssistantPanel.tsx
│
├── p2p/                 # P2P Gateway
│   ├── P2PRegistrationForm.tsx
│   └── PTZControls.tsx
│
├── setup/               # Установка
│   └── RegionSelector.tsx
│
├── EdgeFileManager.tsx  # Edge Agent file management
├── EdgeTerminal.tsx     # Edge Agent SSH-терминал
├── EdgeVideoPlayer.tsx  # Edge Agent видеоплеер
├── AppShell.tsx         # Shell приложения (роуты)
├── AppProviders.tsx     # Провайдеры
├── CommandPalette.tsx   # Палитра команд
├── LanguageSwitcher.tsx # Переключатель языка
├── IEModeButton.tsx     # IE Mode для интеграций
├── WireGuardConfigModal.tsx
└── DeviceActions.tsx    # Действия с устройствами
```

---

### 4. Страницы (45+, lazy-loaded)

| Страница | Роут | Назначение |
|----------|------|------------|
| [`Login`](frontend/src/pages/Login.tsx:1) | `/login` | Вход в систему |
| [`ForgotPassword`](frontend/src/pages/ForgotPassword.tsx:1) | `/forgot-password` | Восстановление пароля |
| [`DashboardHub`](frontend/src/pages/DashboardHub.tsx:1) | `/dashboard` | Главный дашборд |
| [`Sites`](frontend/src/pages/Sites.tsx:1) | `/sites` | Список объектов |
| [`SiteDetail`](frontend/src/pages/SiteDetail.tsx:1) | `/sites/:id` | Детали объекта |
| [`Devices`](frontend/src/pages/Devices.tsx:1) | `/devices` | Список устройств (камеры, edge) |
| [`DeviceDetail`](frontend/src/pages/DeviceDetail.tsx:1) | `/devices/:id` | Детали устройства |
| [`WorkOrders`](frontend/src/pages/WorkOrders.tsx:1) | `/work-orders` | Все наряды-заказы |
| [`WorkOrderDetail`](frontend/src/pages/WorkOrderDetail/WorkOrderDetail.tsx:1) | `/work-orders/:id` | Детали наряда (Info, Timeline, Photos, Parts) |
| [`Tickets`](frontend/src/pages/Tickets.tsx:1) | `/tickets` | Тикеты поддержки |
| [`TicketDetail`](frontend/src/pages/TicketDetail.tsx:1) | `/tickets/:id` | Детали тикета |
| [`Alerts`](frontend/src/pages/Alerts.tsx:1) | `/alerts` | Тревоги |
| [`Notifications`](frontend/src/pages/Notifications.tsx:1) | `/notifications` | Уведомления |
| [`Analytics`](frontend/src/pages/Analytics.tsx:1) | `/analytics` | Аналитика |
| [`AdvancedAnalytics`](frontend/src/pages/AdvancedAnalytics.tsx:1) | `/analytics/advanced` | Продвинутая аналитика |
| [`AnomalyDetection`](frontend/src/pages/AnomalyDetection.tsx:1) | `/analytics/anomalies` | Детекция аномалий |
| [`PredictiveMaintenance`](frontend/src/pages/PredictiveMaintenance.tsx:1) | `/analytics/predictive` | Предиктивное обслуживание |
| [`Reports`](frontend/src/pages/Reports.tsx:1) | `/reports` | Отчёты |
| [`CustomReports`](frontend/src/pages/CustomReports.tsx:1) | `/reports/custom` | Пользовательские отчёты |
| [`MaintenanceReports`](frontend/src/pages/MaintenanceReports.tsx:1) | `/reports/maintenance` | Отчёты по ТО |
| [`BIQueryBuilder`](frontend/src/pages/BIQueryBuilder.tsx:1) | `/analytics/bi` | Self-Service BI (P2-BI) |
| [`SLADashboard`](frontend/src/pages/SLADashboard.tsx:1) | `/sla` | SLA панель |
| [`MaintenanceSchedules`](frontend/src/pages/MaintenanceSchedules.tsx:1) | `/schedules` | Графики ТО |
| [`SpareParts`](frontend/src/pages/SpareParts.tsx:1) | `/spare-parts` | Склад запчастей |
| [`TechnicianWeek`](frontend/src/pages/TechnicianWeek.tsx:1) | `/technicians/week` | Расписание техников |
| [`WorkloadAnalytics`](frontend/src/pages/WorkloadAnalytics.tsx:1) | `/technicians/workload` | Загрузка техников |
| [`AssetOverview`](frontend/src/pages/AssetOverview.tsx:1) | `/assets` | Обзор активов |
| [`LocationTree`](frontend/src/pages/LocationTree.tsx:1) | `/locations` | Дерево локаций |
| [`MeterDashboard`](frontend/src/pages/MeterDashboard.tsx:1) | `/meters` | Панель счётчиков |
| [`TotalCostDashboard`](frontend/src/pages/TotalCostDashboard.tsx:1) | `/costs` | Панель затрат |
| [`VendorPerformance`](frontend/src/pages/VendorPerformance.tsx:1) | `/vendors` | Производительность вендоров |
| [`Logs`](frontend/src/pages/Logs.tsx:1) | `/logs` | Системные логи |
| [`AuditLog`](frontend/src/pages/AuditLog.tsx:1) | `/audit` | Audit Trail (ISO 27001 A.12.4) |
| [`BlackBox`](frontend/src/pages/BlackBox.tsx:1) | `/blackbox` | Чёрный ящик (события) |
| [`EventReplay`](frontend/src/pages/EventReplay.tsx:1) | `/replay` | Повтор событий |
| [`AgentDashboard`](frontend/src/pages/AgentDashboard.tsx:1) | `/agents` | Edge Agent дашборд (EDGE-11) |
| [`AgentDetail`](frontend/src/pages/AgentDetail.tsx:1) | `/agents/:id` | Детали Edge Agent |
| [`Settings`](frontend/src/pages/Settings.tsx:1) | `/settings` | Настройки |
| [`Profile`](frontend/src/pages/Profile.tsx:1) | `/profile` | Профиль пользователя |
| [`Users`](frontend/src/pages/Users.tsx:1) | `/admin/users` | Управление пользователями |
| [`APIKeys`](frontend/src/pages/APIKeys.tsx:1) | `/admin/api-keys` | API ключи |
| [`Webhooks`](frontend/src/pages/Webhooks.tsx:1) | `/admin/webhooks` | Webhook management |
| [`AuditLog (admin)`](frontend/src/pages/AuditLog.tsx:1) | `/admin/audit` | Audit log |
| [`DescriptorEditor`](frontend/src/pages/DescriptorEditor.tsx:1) | `/descriptors` | Редактор протоколов (PROTO-06) |
| [`CommunityRegistry`](frontend/src/pages/CommunityRegistry.tsx:1) | `/community` | Реестр сообщества (PROTO-07) |
| [`PlaybookMarketplace`](frontend/src/pages/PlaybookMarketplace.tsx:1) | `/playbooks` | Маркетплейс плейбуков |
| [`ComplianceShield`](frontend/src/pages/ComplianceShield.tsx:1) | `/compliance` | Панель комплаенса |
| [`SecurityAdvisories`](frontend/src/pages/SecurityAdvisories.tsx:1) | `/security` | Security advisories |
| [`Glossary`](frontend/src/pages/Glossary.tsx:1) | `/glossary` | Глоссарий терминов |
| [`Tutorials`](frontend/src/pages/Tutorials.tsx:1) | `/tutorials` | Обучающие материалы |
| [`OnCallSchedule`](frontend/src/pages/OnCallSchedule.tsx:1) | `/on-call` | График дежурств |
| [`WOAging`](frontend/src/pages/WOAging.tsx:1) | `/work-orders/aging` | Старение заявок |
| [`APIVersioning`](frontend/src/pages/APIVersioning.tsx:1) | `/admin/api-versioning` | Версионирование API |
| [`SetupWizard`](frontend/src/pages/SetupWizard.tsx:1) | `/setup` | Мастер установки |
| [`WorkRequestPortal`](frontend/src/pages/WorkRequestPortal.tsx:1) | `/portal` | Портал заявок (внешний) |
| [`ComplianceShield`](frontend/src/pages/ComplianceShield.tsx:1) | `/compliance` | Compliance shield |

---

### 5. Хуки

| Хук | Файл | Назначение |
|-----|------|------------|
| [`useUnsavedChanges`](frontend/src/hooks/useUnsavedChanges.ts:1) | P0-CR-12 | Блокировка навигации при несохранённых изменениях (`useBlocker` + `beforeunload`) |
| [`useAccessibility`](frontend/src/hooks/useAccessibility.ts:1) | A11y | Focus Trap Stack для вложенных модалок, `useSkipLink` |
| [`useBulkOperations`](frontend/src/hooks/useBulkOperations.ts:1) | Bulk | Массовые операции с прогрессом |
| [`useApiQuery`](frontend/src/hooks/useApiQuery.ts:1) | API | Типизированные запросы (обёртка над React Query) |
| [`useAuth`](frontend/src/hooks/useAuth.tsx:1) | Auth | JWT аутентификация, RBAC, session management |
| [`useKeyboardShortcuts`](frontend/src/hooks/useKeyboardShortcuts.ts:1) | UX | Глобальные хоткеи |
| [`useNavigation`](frontend/src/hooks/useNavigation.ts:1) | UX | История навигации |
| [`useLocalStorage`](frontend/src/hooks/useLocalStorage.ts:1) | Storage | Типизированное localStorage |
| [`useReducedMotion`](frontend/src/hooks/useReducedMotion.ts:1) | A11y | Предпочтения анимаций |
| [`useHapticFeedback`](frontend/src/hooks/useHapticFeedback.ts:1) | UX | Тактильная обратная связь |
| [`useRipple`](frontend/src/hooks/useRipple.tsx:1) | UX | Ripple-эффект на кнопках |
| [`useConfirmAction`](frontend/src/hooks/useConfirmAction.tsx:1) | UX | Confirm-диалоги |
| [`useFormValidation`](frontend/src/hooks/useFormValidation.ts:1) | Forms | Валидация форм (Zod) |
| [`useSearchEntities`](frontend/src/hooks/useSearchEntities.ts:1) | Search | Глобальный поиск сущностей |
| [`useTechnicianSchedule`](frontend/src/hooks/useTechnicianSchedule.ts:1) | CMMS | График техников |
| [`useWebhooks`](frontend/src/hooks/useWebhooks.ts:1) | Webhooks | Управление вебхуками |
| [`useAIAssistant`](frontend/src/hooks/useAIAssistant.ts:1) | AI | AI Assistant интеграция |

---

### 6. Стейт-менеджмент (Zustand)

**Архитектура (ARCH.1 / ADR-005):** Zustand для client-side UI state + React Query для server state.

| Store | Файл | Ключевое состояние |
|-------|------|-------------------|
| [`useAuthStore`](frontend/src/store/authStore.ts:1) | Аутентификация | `user`, `token`, `isLoading`, `hasPermission()` |
| [`useThemeStore`](frontend/src/store/themeStore.tsx:1) | Темизация | `mode` (light/dark), `accent`, тени, анимации |
| [`useSettingsStore`](frontend/src/store/settingsStore.ts:1) | Настройки | Язык, часовой пояс, формат даты, единицы |
| [`useUIStore`](frontend/src/store/uiStore.ts:1) | UI состояние | `sidebarOpen`, `commandPaletteOpen`, `modal`, `panels`, `bulkMode` |
| [`useAlertStore`](frontend/src/store/alertStore.ts:1) | Тревоги | Тост-уведомления, фильтры, выбранные ID |
| [`useNotificationStore`](frontend/src/store/notificationStore.ts:1) | Уведомления | Список, типы, фильтры |
| [`useWorkspaceStore`](frontend/src/store/workspaceStore.ts:1) | Рабочие пространства | Workspaces, виджеты дашборда, layout |
| [`useFilterStore`](frontend/src/store/filterStore.ts:1) | Фильтры | Сохранённые фильтры, saved views |
| [`useSavedViewsStore`](frontend/src/store/savedViewsStore.ts:1) | Сохранённые виды | Dashboard saved views |
| [`useReportsStore`](frontend/src/store/reportsStore.ts:1) | Отчёты | История, статус генерации, expiration sweep |
| [`useCommandPaletteStore`](frontend/src/store/commandPaletteStore.ts:1) | Палитра команд | Результаты поиска, индексы |
| [`useOnboardingStore`](frontend/src/store/onboardingStore.ts:1) | Онбординг | Прогресс тура, пропущенные шаги |
| [`useDescriptorStore`](frontend/src/store/descriptorStore.ts:1) | Редактор дескрипторов | Режим, таб, dirty-флаг, список (PROTO-06) |
| [`useCommunityRegistryStore`](frontend/src/store/communityRegistryStore.ts:1) | Реестр сообщества | Дескрипторы, пагинация, фильтры (PROTO-07) |
| [`useWorkflowStore`](frontend/src/store/workflowStore.ts:1) | Workflow Engine | Ноды, рёбра, CEL-выражения |
| [`useAgentStore`](frontend/src/store/agentStore.ts:1) | Edge Agents | Статусы, метрики, пулы (EDGE-11) |

---

### 7. Безопасность

#### 7.1 CSP (Content Security Policy)

CSP-заголовки настраиваются через Vite plugin (`vite.config.ts`):

```http
Content-Security-Policy: default-src 'self';
  script-src 'self' 'strict-dynamic';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: blob:;
  connect-src 'self' wss:.api.example.com;
  frame-ancestors 'none';
  form-action 'self';
  base-uri 'self';
```

#### 7.2 Sentry Rate Limiting

- [`sentry.ts`](frontend/src/lib/sentry.ts:1) — инициализация Sentry
- `tracesSampleRate`: 0.0 (DEV) / 0.2 (PROD)
- `replaysSessionSampleRate`: 0.0 (DEV) / 0.1 (PROD)
- `replaysOnErrorSampleRate`: 0.0 (DEV) / 1.0 (PROD)

#### 7.3 Trace ID (только DEV)

Trace ID отображается только в development-среде для отладки. В production заменяется на общий ID инцидента.

#### 7.4 ARIA и A11y

- [`Input`](frontend/src/components/ui/Input.tsx:1) — `aria-label`, `aria-describedby`, `aria-invalid`
- [`Select`](frontend/src/components/ui/Dropdown.tsx:1) — `aria-expanded`, `aria-activedescendant`
- [`Badge`](frontend/src/components/ui/Badge.tsx:1) — `role="status"`
- [`Toast`](frontend/src/components/ui/Toast.tsx:1) — `role="alert"`, `aria-live="polite"`
- [`Notification`](frontend/src/components/ui/Notification.tsx:1) — `role="region"`, `aria-label`
- [`Modal`](frontend/src/components/ui/Modal.tsx:1) — `role="dialog"`, `aria-modal`, Focus Trap
- [`ProgressBar`](frontend/src/components/ui/ProgressBar.tsx:1) — `role="progressbar"`, `aria-valuenow`

#### 7.5 Focus Trap Stack

[`useAccessibility`](frontend/src/hooks/useAccessibility.ts:1) управляет стеком Focus Trap для вложенных модалок, предотвращая потерю фокуса при каскадных модальных окнах.

#### 7.6 Permission Guard

- [`PermissionGuard`](frontend/src/components/auth/PermissionGuard.tsx:1) — компонент-обёртка для RBAC
- [`RoleProtectedRoute`](frontend/src/components/auth/RoleProtectedRoute.tsx:1) — роут-гард
- Роли: `admin`, `support`, `owner`, `manager`, `technician`, `viewer`

---

### 8. Тестирование

#### 8.1 Unit-тесты (Vitest)

| Тест | Файл | Компонент |
|------|------|-----------|
| Badge | [`Badge.test.tsx`](frontend/src/components/ui/__tests__/Badge.test.tsx:1) | Badge |
| Breadcrumbs | [`Breadcrumbs.test.tsx`](frontend/src/components/ui/__tests__/Breadcrumbs.test.tsx:1) | Breadcrumbs |
| Button | [`Button.test.tsx`](frontend/src/components/ui/__tests__/Button.test.tsx:1) | Button |
| DataGrid | [`DataGrid.test.tsx`](frontend/src/components/ui/__tests__/DataGrid.test.tsx:1) | DataGrid |
| Dropdown | [`Dropdown.test.tsx`](frontend/src/components/ui/__tests__/Dropdown.test.tsx:1) | Dropdown |
| EmptyState | [`EmptyState.test.tsx`](frontend/src/components/ui/__tests__/EmptyState.test.tsx:1) | EmptyState |
| LazyImage | [`LazyImage.test.tsx`](frontend/src/components/ui/__tests__/LazyImage.test.tsx:1) | LazyImage |
| Modal | [`Modal.test.tsx`](frontend/src/components/ui/__tests__/Modal.test.tsx:1) | Modal |
| ProgressBar | [`ProgressBar.test.tsx`](frontend/src/components/ui/__tests__/ProgressBar.test.tsx:1) | ProgressBar |
| Skeleton | [`Skeleton.test.tsx`](frontend/src/components/ui/__tests__/Skeleton.test.tsx:1) | Skeleton |
| StatsCard | [`StatsCard.test.tsx`](frontend/src/components/ui/__tests__/StatsCard.test.tsx:1) | StatsCard |
| Tabs | [`Tabs.test.tsx`](frontend/src/components/ui/__tests__/Tabs.test.tsx:1) | Tabs |
| Tooltip | [`Tooltip.test.tsx`](frontend/src/components/ui/__tests__/Tooltip.test.tsx:1) | Tooltip |
| AssetTree | [`AssetTree.test.tsx`](frontend/src/components/organisms/__tests__/AssetTree.test.tsx:1) | AssetTree |
| BeforeAfterSlider | [`BeforeAfterSlider.test.tsx`](frontend/src/components/organisms/__tests__/BeforeAfterSlider.test.tsx:1) | BeforeAfterSlider |
| AgentStatsCard | [`AgentStatsCard.test.tsx`](frontend/src/components/agents/__tests__/AgentStatsCard.test.tsx:1) | AgentStatsCard |
| AgentStatusBadge | [`AgentStatusBadge.test.tsx`](frontend/src/components/agents/__tests__/AgentStatusBadge.test.tsx:1) | AgentStatusBadge |
| AgentTable | [`AgentTable.test.tsx`](frontend/src/components/agents/__tests__/AgentTable.test.tsx:1) | AgentTable |
| RCAWidget | [`RCAWidget.test.tsx`](frontend/src/components/rca/__tests__/RCAWidget.test.tsx:1) | RCAWidget |

#### 8.2 Storybook (9+ stories)

| Story | Файл |
|-------|------|
| Button | [`Button.stories.tsx`](frontend/src/components/ui/Button.stories.tsx:1) |
| Input | [`Input.stories.tsx`](frontend/src/components/ui/Input.stories.tsx:1) |
| Table | [`Table.stories.tsx`](frontend/src/components/ui/Table.stories.tsx:1) |
| DataGrid | [`DataGrid.stories.tsx`](frontend/src/components/ui/DataGrid.stories.tsx:1) |
| Gauge | [`Gauge.stories.tsx`](frontend/src/components/ui/Gauge.stories.tsx:1) |
| Notification | [`Notification.stories.tsx`](frontend/src/components/ui/Notification.stories.tsx:1) |
| InfoTooltip | [`InfoTooltip.stories.tsx`](frontend/src/components/ui/InfoTooltip.stories.tsx:1) |
| SLAProgress | [`SLAProgress.stories.tsx`](frontend/src/components/ui/SLAProgress.stories.tsx:1) |
| BulkProgressModal | [`BulkProgressModal.stories.tsx`](frontend/src/components/ui/BulkProgressModal.stories.tsx:1) |
| Toast | [`Toast.stories.tsx`](frontend/src/components/ui/Toast.stories.tsx:1) |
| Modal | [`Modal.stories.tsx`](frontend/src/components/ui/Modal.stories.tsx:1) |
| Badge | [`Badge.stories.tsx`](frontend/src/components/ui/Badge.stories.tsx:1) |
| Dropdown | [`Dropdown.stories.tsx`](frontend/src/components/ui/Dropdown.stories.tsx:1) |
| Tabs | [`Tabs.stories.tsx`](frontend/src/components/ui/Tabs.stories.tsx:1) |
| ProgressBar | [`ProgressBar.stories.tsx`](frontend/src/components/ui/ProgressBar.stories.tsx:1) |
| Tooltip | [`Tooltip.stories.tsx`](frontend/src/components/ui/Tooltip.stories.tsx:1) |
| EmptyState | [`EmptyState.stories.tsx`](frontend/src/components/ui/EmptyState.stories.tsx:1) |
| Skeleton | [`Skeleton.stories.tsx`](frontend/src/components/ui/Skeleton.stories.tsx:1) |
| VisuallyHidden | [`VisuallyHidden.stories.tsx`](frontend/src/components/ui/VisuallyHidden.stories.tsx:1) |
| StatsCard | [`StatsCard.stories.tsx`](frontend/src/components/ui/StatsCard.stories.tsx:1) |
| ThemeCustomizer | [`ThemeCustomizer.stories.tsx`](frontend/src/components/ui/ThemeCustomizer.stories.tsx:1) |
| LazyImage | [`LazyImage.stories.tsx`](frontend/src/components/ui/LazyImage.stories.tsx:1) |
| SavedViews | [`SavedViews.stories.tsx`](frontend/src/components/ui/SavedViews.stories.tsx:1) |
| VideoTutorialCard | [`VideoTutorialCard.stories.tsx`](frontend/src/components/ui/VideoTutorialCard.stories.tsx:1) |
| ShortcutsCheatsheet | [`ShortcutsCheatsheet.stories.tsx`](frontend/src/components/ui/ShortcutsCheatsheet.stories.tsx:1) |
| SavedFiltersDropdown | [`SavedFiltersDropdown.stories.tsx`](frontend/src/components/ui/SavedFiltersDropdown.stories.tsx:1) |
| QRCode | [`QRCode.stories.tsx`](frontend/src/components/ui/QRCode.stories.tsx:1) |
| LiveSLATimer | [`LiveSLATimer.stories.tsx`](frontend/src/components/ui/LiveSLATimer.stories.tsx:1) |
| SLAProgressBar | [`SLAProgressBar.stories.tsx`](frontend/src/components/molecules/SLAProgressBar.stories.tsx:1) |
| DateRangePicker | [`DateRangePicker.stories.tsx`](frontend/src/components/molecules/DateRangePicker.stories.tsx:1) |
| PriorityPicker | [`PriorityPicker.stories.tsx`](frontend/src/components/molecules/PriorityPicker.stories.tsx:1) |
| TechnicianSelector | [`TechnicianSelector.stories.tsx`](frontend/src/components/molecules/TechnicianSelector.stories.tsx:1) |
| AssetTree | [`AssetTree.stories.tsx`](frontend/src/components/organisms/AssetTree.stories.tsx:1) |
| BeforeAfterSlider | [`BeforeAfterSlider.stories.tsx`](frontend/src/components/organisms/BeforeAfterSlider.stories.tsx:1) |
| RCAGraph | [`RCAGraph.stories.tsx`](frontend/src/components/rca/RCAGraph.stories.tsx:1) |
| RCAWidget | [`RCAWidget.stories.tsx`](frontend/src/components/rca/RCAWidget.stories.tsx:1) |
| SLABreachTimeline | [`SLABreachTimeline.stories.tsx`](frontend/src/components/sla/SLABreachTimeline.stories.tsx:1) |
| SLAGaugePanel | [`SLAGaugePanel.stories.tsx`](frontend/src/components/sla/SLAGaugePanel.stories.tsx:1) |
| SLAHeatmap | [`SLAHeatmap.stories.tsx`](frontend/src/components/sla/SLAHeatmap.stories.tsx:1) |
| SLATrendChart | [`SLATrendChart.stories.tsx`](frontend/src/components/sla/SLATrendChart.stories.tsx:1) |
| AlertBanner | [`AlertBanner.stories.tsx`](frontend/src/components/dashboard/AlertBanner.stories.tsx:1) |
| DragDropDashboard | [`DragDropDashboard.stories.tsx`](frontend/src/components/dashboard/DragDropDashboard.stories.tsx:1) |
| LanguageSwitcher | [`LanguageSwitcher.stories.tsx`](frontend/src/components/LanguageSwitcher.stories.tsx:1) |
| ErrorBoundaryLite | [`ErrorBoundaryLite.stories.tsx`](frontend/src/components/ErrorBoundaryLite.stories.tsx:1) |
| PermissionGuard | [`PermissionGuard.stories.tsx`](frontend/src/components/auth/PermissionGuard.stories.tsx:1) |
| RoleProtectedRoute | [`RoleProtectedRoute.stories.tsx`](frontend/src/components/auth/RoleProtectedRoute.stories.tsx:1) |
| WebAuthnSetup | [`WebAuthnSetup.stories.tsx`](frontend/src/components/auth/WebAuthnSetup.stories.tsx:1) |
| WOChat | [`WOChat.stories.tsx`](frontend/src/components/chat/WOChat.stories.tsx:1) |
| PhotoAnnotation | [`PhotoAnnotation.stories.tsx`](frontend/src/components/work-orders/PhotoAnnotation.stories.tsx:1) |
| P2PRegistrationForm | [`P2PRegistrationForm.stories.tsx`](frontend/src/components/p2p/P2PRegistrationForm.stories.tsx:1) |
| PTZControls | [`PTZControls.stories.tsx`](frontend/src/components/p2p/PTZControls.stories.tsx:1) |
| AIAssistantPanel | [`AIAssistantPanel.stories.tsx`](frontend/src/components/ai/AIAssistantPanel.stories.tsx:1) |
| Header | [`Header.stories.tsx`](frontend/src/components/layout/Header.stories.tsx:1) |
| Sidebar | [`Sidebar.stories.tsx`](frontend/src/components/layout/Sidebar.stories.tsx:1) |
| OfflineBanner | [`OfflineBanner.stories.tsx`](frontend/src/components/layout/OfflineBanner.stories.tsx:1) |
| PageSuspense | [`PageSuspense.stories.tsx`](frontend/src/components/layout/PageSuspense.stories.tsx:1) |
| RouteErrorBoundary | [`RouteErrorBoundary.stories.tsx`](frontend/src/components/layout/RouteErrorBoundary.stories.tsx:1) |
| SkeletonPage | [`SkeletonPage.stories.tsx`](frontend/src/components/layout/SkeletonPage.stories.tsx:1) |
| WorkspaceSwitcher | [`WorkspaceSwitcher.stories.tsx`](frontend/src/components/layout/WorkspaceSwitcher.stories.tsx:1) |
| WebhookBuilder | [`WebhookBuilder.stories.tsx`](frontend/src/components/webhooks/WebhookBuilder.stories.tsx:1) |
| WebhookLogFilter | [`WebhookLogFilter.stories.tsx`](frontend/src/components/webhooks/WebhookLogFilter.stories.tsx:1) |
| WebhookRetryPolicy | [`WebhookRetryPolicy.stories.tsx`](frontend/src/components/webhooks/WebhookRetryPolicy.stories.tsx:1) |
| WebhookStatsCards | [`WebhookStatsCards.stories.tsx`](frontend/src/components/webhooks/WebhookStatsCards.stories.tsx:1) |
| APIVersioning | [`APIVersioning.stories.tsx`](frontend/src/pages/APIVersioning.stories.tsx:1) |
| EventReplay | [`EventReplay.stories.tsx`](frontend/src/pages/EventReplay.stories.tsx:1) |
| PlaybookMarketplace | [`PlaybookMarketplace.stories.tsx`](frontend/src/pages/PlaybookMarketplace.stories.tsx:1) |

#### 8.3 Visual Regression (Playwright)

14 visual regression тестов:

```bash
npm run test:visual        # Playwright visual tests
npm run test:a11y          # Accessibility (axe-core)
npm run test:a11y:ci       # A11y CI (forbid-only)
```

#### 8.4 A11y axe-core CI

- [`playwright-report/`](frontend/playwright-report/) — отчёты о визуальных регрессиях
- A11y smoke tests: `npm run test:a11y:smoke`
- axe-core интеграция в CI pipeline

---

### 9. Интернационализация (i18n)

**20 языков:**

| Язык | Код | Тип |
|------|-----|-----|
| 🇬🇧 Английский | `en` | Статический (default) |
| 🇷🇺 Русский | `ru` | Статический (fallback) |
| 🇧🇾 Белорусский | `be` | Статический |
| 🇩🇪 Немецкий | `de` | Lazy |
| 🇫🇷 Французский | `fr` | Lazy |
| 🇪🇸 Испанский | `es` | Lazy |
| 🇮🇹 Итальянский | `it` | Lazy |
| 🇵🇱 Польский | `pl` | Lazy |
| 🇹🇷 Турецкий | `tr` | Lazy |
| 🇺🇦 Украинский | `uk` | Lazy |
| 🇰🇿 Казахский | `kk` | Lazy |
| 🇺🇿 Узбекский | `uz` | Lazy |
| 🇻🇳 Вьетнамский | `vi` | Lazy |
| 🇮🇩 Индонезийский | `id` | Lazy |
| 🇯🇵 Японский | `ja` | Lazy |
| 🇰🇷 Корейский | `ko` | Lazy |
| 🇨🇳 Китайский | `zh` | Lazy |
| 🇸🇦 Арабский | `ar` | Lazy |
| 🇸🇪 Шведский | `sw` | Lazy |
| 🇵🇹 Португальский | `pt` | Lazy |

**Механизм загрузки** ([`i18n.ts`](frontend/src/i18n.ts:1)):

- `en`, `ru`, `be` — статически загружаются при инициализации
- Остальные 17 языков — lazy-loaded при переключении (`i18n.on('languageChanged')`)

---

### 10. Стилизация и Дизайн-система

- **TailwindCSS v4** — основа CSS фреймворка
- [`tokens.css`](frontend/src/styles/tokens.css:1) — CSS custom properties (цвета, тени, шрифты)
- [`animations.css`](frontend/src/styles/animations.css:1) — кастомные анимации
- [`a11y.css`](frontend/src/styles/a11y.css:1) — accessibility стили (focus, reduced-motion)
- [`print.css`](frontend/src/styles/print.css:1) — стили для печати work orders
- [`safelist.css`](frontend/src/styles/safelist.css:1) — safelist для динамических классов Tailwind
- [`ThemeProvider`](frontend/src/store/ThemeProvider.tsx:1) — Zustand-based темизация (light/dark, accent)
- [`ThemeCustomizer`](frontend/src/components/ui/ThemeCustomizer.tsx:1) — кастомизация темы пользователем
- [`WhiteLabelCustomizer`](frontend/src/components/ui/WhiteLabelCustomizer.tsx:1) — white-label для OEM

---

### 11. PWA и Offline

- **Service Worker** ([`sw.js`](frontend/public/sw.js:1)) — кэширование статики через Workbox
- **Offline fallback** ([`offline.html`](frontend/public/offline.html:1)) — страница при отсутствии соединения
- **vite-plugin-pwa** — генерация manifest, precaching

---

## Mobile (React Native + Expo 52)

### 1. Технический стек

| Категория | Технология | Назначение |
|-----------|-----------|------------|
| **Ядро** | React Native 0.76 + Expo 52 | Кроссплатформенное мобильное приложение |
| **Язык** | TypeScript 5.3 | Типизация |
| **Навигация** | @react-navigation/native 7 + bottom-tabs + native-stack | Навигация |
| **Офлайн-БД** | @nozbe/watermelondb 0.28 | Reactive offline storage |
| **Локальное SQLite** | expo-sqlite | Нижнеуровневая БД |
| **Стейт** | Zustand 5 | Client-side state |
| **Серверный стейт** | @tanstack/react-query 5 | Кэширование API |
| **HTTP** | axios | API-клиент |
| **Карты** | react-native-maps 1.18 | Карта объектов |
| **Камера** | expo-camera 16 | QR-сканирование |
| **Баркод** | expo-barcode-scanner 13 | Сканер QR/баркодов |
| **Биометрия** | expo-secure-store 14 | Безопасное хранение токенов |
| **Геолокация** | expo-location 18 | GPS-трекинг |
| **Уведомления** | expo-notifications 0.29 | Push-уведомления |
| **Фоновая синхр.** | expo-background-fetch + expo-task-manager | Background sync |
| **Файлы** | expo-file-system 18 | Работа с файлами |
| **Изображения** | expo-image-picker 16 | Фото с камеры/галереи |
| **Аудио** | expo-av 15 | Голосовые заметки |
| **Подпись** | react-native-signature-canvas 4 | Электронная подпись |
| **WebView** | react-native-webview 13 | Просмотр отчётов |
| **Мониторинг** | @sentry/react-native 6 | Error tracking |
| **Скриншоты** | react-native-view-shot 3.8 | Фиксация подписи |

---

### 2. Офлайн-архитектура

#### 2.1 WatermelonDB как Single Source of Truth

[`schema.ts`](mobile/src/database/schema.ts:1) определяет 4 таблицы:

```typescript
tables: [
  work_orders — type, status, priority, checklist, photos, parts_used, notes, sla
  devices     — name, device_type, status, coordinates, health
  sites       — name, coordinates, address, timezone
  pending_mutations — offline mutation queue (background sync)
]
```

Маппинг WatermelonDB моделей: [`models.ts`](mobile/src/database/models.ts:1).

#### 2.2 Sync: Push/Pull с 3-Way Merge (P0-CR-05)

[`differentialSync.ts`](mobile/src/services/differentialSync.ts:1) — полный sync cycle:

```
1. Pull — fetchDiff() с сервера
2. Apply — применить remote изменения в локальный SQLite
3. Collect — собрать локальные pending мутации
4. Resolve — 3-way merge (P0-CR-05)
5. Push — applyChanges() на сервер
```

**3-Way Merge Policy:**

| Тип поля | Authority | Поведение |
|----------|-----------|-----------|
| `status`, `priority`, `sla_deadline`, `sla_status`, `assigned_to` | Server | Всегда принимается серверная версия |
| `notes`, `parts_used`, `checklist`, `photos` | Client | Сохраняется локальная версия; если обе стороны изменили → Conflict Resolution UI |
| `device_id`, `type`, `created_by`, `created_at` | Immutable | Не участвуют в merge |

#### 2.3 Offline Queue

[`syncStore.ts`](mobile/src/store/syncStore.ts:1) управляет очередью:

- `SyncAction` — типы: `complete_work_order`, `start_work_order`, `checklist_update`, `checklist_complete`
- `PendingMutation` — очередь с `retryCount`
- Интеграция с WatermelonDB (`pending_mutations` table)

#### 2.4 Conflict Resolution UI

[`ConflictResolutionModal`](mobile/src/components/ConflictResolutionModal.tsx:1) — UI для ручного разрешения конфликтов:

- Показывает diff между локальной и серверной версией
- Выбор: `useLocal | useRemote | merge`

#### 2.5 Background Sync

- [`BackgroundSyncApp`](mobile/src/components/BackgroundSyncApp.tsx:1) — компонент фоновой синхронизации
- [`useBackgroundSync`](mobile/src/hooks/useBackgroundSync.ts:1) — хук для `expo-background-fetch` + `expo-task-manager`
- [`useOfflineSync`](mobile/src/hooks/useOfflineSync.ts:1) — хук управления синхронизацией

---

### 3. Структура приложения

```
mobile/
├── App.tsx                        # Точка входа
├── src/
│   ├── api/
│   │   ├── auth.ts                # Auth API (login, refresh, logout)
│   │   ├── client.ts              # Axios client с перехватчиками
│   │   ├── devices.ts             # Devices API
│   │   ├── gatekeeper.ts          # Gatekeeper check-in API
│   │   ├── sync.ts                # Sync API (fetchDiff, applyChanges)
│   │   └── workOrders.ts          # Work Orders API
│   │
│   ├── components/
│   │   ├── AIScore.tsx            # AI-based diagnostic score
│   │   ├── BackgroundSyncApp.tsx   # Background sync manager
│   │   ├── CompleteWorkOrderWizard.tsx  # Мастер завершения
│   │   ├── ConflictResolutionModal.tsx  # UI разрешения конфликтов
│   │   ├── EXIFStatus.tsx         # EXIF статус фото
│   │   ├── GPSStatus.tsx          # GPS статус
│   │   ├── OfflineIndicator.tsx   # Индикатор офлайн-режима
│   │   ├── PhotoAnnotation.tsx    # Аннотация фото
│   │   ├── StatusBadge.tsx        # Бейдж статуса
│   │   ├── SwipeableCard.tsx      # Swipeable карточка
│   │   ├── SyncStatusBar.tsx      # Статус-бар синхронизации
│   │   ├── SyncStatusIndicator.tsx # Индикатор синхронизации
│   │   ├── VoiceNoteRecorder.tsx  # Запись голосовых заметок
│   │   └── WorkOrderCard.tsx      # Карточка work order
│   │   └── checklist/
│   │       ├── ChecklistItem.tsx          # Элемент чек-листа
│   │       └── RegulatoryChecklist.tsx     # Регуляторный чек-лист
│   │
│   ├── database/
│   │   ├── index.ts               # WatermelonDB database init
│   │   ├── models.ts              # WatermelonDB модели (WorkOrder, Device, Site)
│   │   └── schema.ts              # WatermelonDB schema
│   │
│   ├── hooks/
│   │   ├── useBackgroundSync.ts   # Background sync hook
│   │   ├── useGatekeeper.ts       # Gatekeeper hook
│   │   ├── useLocation.ts         # GPS location hook
│   │   ├── useOfflineMap.ts       # Offline map tiles hook
│   │   ├── useOfflineSync.ts      # Offline sync management
│   │   └── useWorkOrders.ts       # Work Orders hook
│   │
│   ├── lib/
│   │   └── sentry.ts              # Sentry initialization
│   │
│   ├── navigation/
│   │   └── AppNavigator.tsx       # Root navigator (tabs + stack)
│   │
│   ├── screens/
│   │   ├── DashboardScreen.tsx    # Список заданий
│   │   ├── LoginScreen.tsx        # Вход
│   │   ├── MaintenanceChecklistScreen.tsx  # Чек-лист ТО
│   │   ├── MapScreen.tsx          # Карта объектов
│   │   ├── ProfileScreen.tsx      # Профиль
│   │   ├── QRScannerScreen.tsx    # QR-сканер
│   │   ├── SignatureScreen.tsx    # Электронная подпись
│   │   └── WorkOrderDetailScreen.tsx  # Детали наряда
│   │
│   ├── services/
│   │   ├── certificatePinning.ts  # Certificate pinning (OWASP V9.1)
│   │   ├── differentialSync.ts   # Differential sync (3-way merge)
│   │   ├── offlineStorage.ts     # SQLite offline CRUD
│   │   ├── syncService.ts        # Sync orchestration
│   │   └── tileCache.ts          # LRU map tile cache
│   │
│   ├── store/
│   │   ├── authStore.ts           # Auth state
│   │   ├── deviceMapStore.ts      # Device map state
│   │   ├── syncStore.ts           # Sync state (576 lines)
│   │   └── workOrderStore.ts      # Work order state
│   │
│   ├── types/
│   │   └── index.ts               # TypeScript types
│   │
│   └── utils/
│       ├── dateHelpers.ts         # Date formatting
│       ├── i18n.ts                # Mobile i18n
│       ├── notifications.ts       # Push notification helpers
│       └── storage.ts             # SecureStore wrapper
```

---

### 4. Экраны (Screens)

| Экран | Файл | Назначение |
|-------|------|------------|
| [`LoginScreen`](mobile/src/screens/LoginScreen.tsx:1) | Вход | Аутентификация (JWT + биометрия) |
| [`DashboardScreen`](mobile/src/screens/DashboardScreen.tsx:1) | Dashboard | Список активных заданий техника |
| [`WorkOrderDetailScreen`](mobile/src/screens/WorkOrderDetailScreen.tsx:1) | Детали наряда | Просмотр и выполнение work order |
| [`MapScreen`](mobile/src/screens/MapScreen.tsx:1) | Карта | Карта объектов с устройствами |
| [`QRScannerScreen`](mobile/src/screens/QRScannerScreen.tsx:1) | QR-сканер | Сканирование QR устройств |
| [`SignatureScreen`](mobile/src/screens/SignatureScreen.tsx:1) | Подпись | Электронная подпись по завершении |
| [`ProfileScreen`](mobile/src/screens/ProfileScreen.tsx:1) | Профиль | Настройки профиля техника |
| [`MaintenanceChecklistScreen`](mobile/src/screens/MaintenanceChecklistScreen.tsx:1) | Чек-лист | Регуляторный чек-лист ТО |

#### 4.1 Навигация

[`AppNavigator.tsx`](mobile/src/navigation/AppNavigator.tsx:1) — 2 уровня:

1. **Bottom Tab Navigator** — 3 вкладки:
   - `Dashboard` — Мои задания
   - `Map` — Карта объектов
   - `Profile` — Профиль

2. **Native Stack Navigator** — модальные/стековые экраны:
   - `Login` (если не аутентифицирован)
   - `WorkOrderDetail` (из Dashboard)
   - `QRScanner` (из Dashboard)
   - `Signature` (из WorkOrderDetail)
   - `CompleteWorkOrder` (wizard)

#### 4.2 MapScreen — Кэш тайлов (LRU)

[`tileCache.ts`](mobile/src/services/tileCache.ts:1) — LRU-кэш для офлайн-карт:

- Кэширование тайлов в файловой системе (expo-file-system)
- LRU eviction policy
- Tile pre-fetching для зон обслуживания

---

### 5. Безопасность (Mobile)

#### 5.1 Certificate Pinning (OWASP ASVS V9.1)

[`certificatePinning.ts`](mobile/src/services/certificatePinning.ts:1):

- Хранение fingerprint'ов сертификатов в SecureStore
- Проверка SPKI fingerprint при подключении
- Grace period для rotation (7 дней)
- Audit log нарушений (pin violations)
- Максимум 100 записей нарушений в логе

**Compliance:**
- OWASP ASVS V9.1 (Certificate Pinning)
- OWASP ASVS V14.2 (Mobile endpoint security)
- IEC 62443 SR 2.1 (Account management — secure communication)
- Приказ ОАЦ №66 п. 7.18.2 (mTLS)

#### 5.2 Биометрия

- [`authStore.ts`](mobile/src/store/authStore.ts:1) — интеграция с `expo-secure-store`
- Биометрическая аутентификация при повторном входе
- SecureStore для хранения токенов (не AsyncStorage)

#### 5.3 Secure Token Storage

- [`storage.ts`](mobile/src/utils/storage.ts:1) — обёртка над `expo-secure-store`
- `setToken()`, `getToken()`, `removeToken()`
- `setRefreshToken()`, `getRefreshToken()`, `removeRefreshToken()`

---

### 6. Тестирование (Mobile)

#### 6.1 E2E Тесты (Detox + Jest)

| Тест | Файл |
|------|------|
| Agent Dashboard | [`agent-dashboard.spec.ts`](mobile/e2e/agent-dashboard.spec.ts:1) |
| Background Sync | [`background-sync.spec.ts`](mobile/e2e/background-sync.spec.ts:1) |
| Differential Sync | [`differential-sync.spec.ts`](mobile/e2e/differential-sync.spec.ts:1) |
| Differential Sync Extended | [`differential-sync-extended.spec.ts`](mobile/e2e/differential-sync-extended.spec.ts:1) |
| E-Signature | [`e-signature.spec.ts`](mobile/e2e/e-signature.spec.ts:1) |
| Maintenance Checklist | [`maintenance-checklist.spec.ts`](mobile/e2e/maintenance-checklist.spec.ts:1) |
| Offline Mode | [`offline.spec.ts`](mobile/e2e/offline.spec.ts:1) |
| Photo Annotation | [`photo-annotation.spec.ts`](mobile/e2e/photo-annotation.spec.ts:1) |
| Photo Annotation Extended | [`photo-annotation-extended.spec.ts`](mobile/e2e/photo-annotation-extended.spec.ts:1) |
| Photo Gatekeeper | [`photo-gatekeeper.spec.ts`](mobile/e2e/photo-gatekeeper.spec.ts:1) |
| Push Notifications | [`push-notifications.spec.ts`](mobile/e2e/push-notifications.spec.ts:1) |
| QR Scanner | [`qr-scanner.spec.ts`](mobile/e2e/qr-scanner.spec.ts:1) |
| Sync Conflict | [`sync-conflict.spec.ts`](mobile/e2e/sync-conflict.spec.ts:1) |

#### 6.2 Helpers

| Файл | Назначение |
|------|------------|
| [`mockData.ts`](mobile/e2e/helpers/mockData.ts:1) | Мок-данные для тестов |
| [`testUtils.ts`](mobile/e2e/helpers/testUtils.ts:1) | Утилиты для тестов |
| [`ArtifactPathBuilder.js`](mobile/e2e/helpers/ArtifactPathBuilder.js:1) | Построение путей для артефактов |

---

### 7. API Слой

#### 7.1 Frontend API Services

| Сервис | Файл | Назначение |
|--------|------|------------|
| API Client | [`api.ts`](frontend/src/services/api.ts:1) | Axios instance с перехватчиками |
| Error Mapper | [`apiErrorMapper.ts`](frontend/src/services/apiErrorMapper.ts:1) | Маппинг ошибок API |
| Chat API | [`chatApi.ts`](frontend/src/services/chatApi.ts:1) | Чат по work orders |
| Checklist API | [`checklistApi.ts`](frontend/src/services/checklistApi.ts:1) | Чек-листы ТО |
| Maintenance API | [`maintenanceApi.ts`](frontend/src/services/maintenanceApi.ts:1) | CMMS API |
| P2P API | [`p2pApi.ts`](frontend/src/services/p2pApi.ts:1) | P2P Gateway API |
| Spare Parts API | [`sparePartsApi.ts`](frontend/src/services/sparePartsApi.ts:1) | API запчастей |
| WebSocket | [`websocket.ts`](frontend/src/services/websocket.ts:1) | Real-time уведомления |
| Work Orders API | [`workOrdersApi.ts`](frontend/src/services/workOrdersApi.ts:1) | Work Orders CRUD |

#### 7.2 Mobile API Clients

| Сервис | Файл | Назначение |
|--------|------|------------|
| Auth API | [`auth.ts`](mobile/src/api/auth.ts:1) | Login, refresh, logout |
| Client | [`client.ts`](mobile/src/api/client.ts:1) | Axios + certificate pinning |
| Devices API | [`devices.ts`](mobile/src/api/devices.ts:1) | Device catalog |
| Gatekeeper API | [`gatekeeper.ts`](mobile/src/api/gatekeeper.ts:1) | Check-in/out |
| Sync API | [`sync.ts`](mobile/src/api/sync.ts:1) | Differential sync |
| Work Orders API | [`workOrders.ts`](mobile/src/api/workOrders.ts:1) | WO CRUD |

---

### 8. Compliance Matrix

| Стандарт | Frontend | Mobile |
|----------|----------|--------|
| **СТБ IEC 62443 SL-3** | RBAC (SR 3.1), Input validation (SR 3.4), Session mgmt (SR 1.1) | mTLS (SR 2.1), Queue processing (SR 3.1) |
| **ISO 27001 A.12.4** | Audit trail, Sentry logging | Audit trail (differentialSync), pending_mutations |
| **OWASP ASVS L3** | V1-V17 (Input, Output, SQLi, Access, Session, Crypto, Errors) | V9.1 (Cert pinning), V14.2 (Mobile security) |
| **Приказ ОАЦ №66** | п. 7.18.1 (ID), п. 7.18.2 (mTLS) | п. 7.18.2 (mTLS), п. 7.18.3 (Integrity) |
| **СТБ 34.101.27** | Защита информации при передаче | Защита данных на устройстве |
| **ГОСТ/СТБ крипто** | TLS 1.3 (belt-gcm/bign) — на уровне API Gateway | Certificate pinning, SecureStore |

---

### 9. Ключевые ADR

| ADR | Описание |
|-----|----------|
| [`ADR-005`](docs/adr/ADR-005-state-management.md) | State management: Zustand + React Query |
| [`ADR-006`](docs/adr/ADR-006-offline-first.md) | Offline-first архитектура (WatermelonDB, 3-way sync) |

---

### 10. Скрипты

#### Frontend

```bash
npm run dev                 # Vite dev server
npm run build               # TypeScript check + production build
npm run preview             # Preview production build
npm run storybook           # Storybook dev server
npm run test:unit           # Vitest unit tests
npm run test:coverage       # Vitest with coverage
npm run test:e2e            # Playwright E2E
npm run test:a11y           # Accessibility tests
npm run test:visual         # Visual regression tests
npm run generate:api         # OpenAPI → TypeScript types
```

#### Mobile

```bash
npm start                   # Expo dev server
npm run android             # Android build
npm run ios                 # iOS build
npm run web                 # Web (Expo)
npm test                    # Jest tests
npm run lint                # ESLint check
```

---

### 11. Международные версии

Приложение поддерживает **20 языков**:

- **Default:** Английский (en), Русский (ru), Белорусский (be) — статическая загрузка
- **Lazy-loaded:** Немецкий, Французский, Испанский, Итальянский, Польский, Турецкий, Украинский, Казахский, Узбекский, Вьетнамский, Индонезийский, Японский, Корейский, Китайский, Арабский, Шведский, Португальский

Локализованные regulatory requirements учитывают:
- 🇧🇾 **РБ** — СТБ 34.101.27, СТБ 34.101.30, Приказ ОАЦ №66
- 🇷🇺 **РФ** — 152-ФЗ, 149-ФЗ, Приказ ФСТЭК №17
- 🇪🇺 **EU** — GDPR Art. 35 (DPIA), NIS2, EN 62676
- 🇹🇷 **Турция** — KVKK №6698, TS EN 62676
- 🇻🇳 **Вьетнам** — TCVN 11930:2017, Decree 13/2023
- 🇮🇩 **Индонезия** — UU PDP, SNI 27001
- 🇿🇦 **ЮАР** — POPIA, SANS 10160-4
- 🇧🇷 **Бразилия** — LGPD
- 🇲🇽 **Мексика** — LFPDPPP
