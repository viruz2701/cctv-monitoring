# TODO.md — CCTV Health Monitor
> Living document. Roo использует этот файл как основной roadmap.
> Обновлять после завершения каждой задачи: [ ] → [x] + дата.
> Последнее обновление: 2026-06-25

---

## 🔴 P0 — Критично (Q3 2026, до 2026-09-30)

### P0-1: Разделить Settings.tsx на 6 вкладок
- [ ] **P0-1.1** Проанализировать текущий `frontend/src/pages/Settings.tsx` (953 строки), выделить логические блоки
- [ ] **P0-1.2** Создать компоненты вкладок:
  - `frontend/src/pages/settings/GeneralSettings.tsx`
  - `frontend/src/pages/settings/ServicesSettings.tsx`
  - `frontend/src/pages/settings/IntegrationsSettings.tsx` (ServiceNow, 1С:ТОИР, Jira)
  - `frontend/src/pages/settings/SecuritySettings.tsx` (SSO/LDAP/SAML уже существует как `SSOSettings.tsx` — интегрировать)
  - `frontend/src/pages/settings/NotificationsSettings.tsx`
  - `frontend/src/pages/settings/LoggingSettings.tsx`
- [ ] **P0-1.3** Создать `frontend/src/components/ui/Tabs.tsx` (атом дизайн-системы, если ещё нет)
- [ ] **P0-1.4** Добавить RBAC-контроль доступа к вкладкам (admin-only для Security/Integrations)
- [ ] **P0-1.5** Обновить роут: `/settings` → `/settings/:tab` с deep linking
- [ ] **P0-1.6** Убедиться что Settings.tsx < 200 строк после рефакторинга
- **Критерий приёмки:** Каждая вкладка — отдельный файл <300 строк, RBAC работает, deep linking сохраняет активную вкладку

### P0-2: Редизайн WorkOrders (Snipe-IT паттерн)
- [ ] **P0-2.1** Создать `frontend/src/components/ui/ProgressBar.tsx` — переиспользуемый атом
- [ ] **P0-2.2** Создать `frontend/src/components/ui/Breadcrumbs.tsx` — атом навигации
- [ ] **P0-2.3** Доработать `frontend/src/components/ui/DataGrid.tsx`:
  - Добавить multi-select checkboxes (колонка с чекбоксами)
  - Добавить bulk action toolbar (массовый assign / cancel / priority change)
  - Добавить inline status change (dropdown в ячейке)
  - Интегрировать `@tanstack/react-virtual` для виртуализации строк (уже в package.json!)
- [ ] **P0-2.4** Создать `frontend/src/components/work-orders/QuickFilters.tsx`:
  - Чипы: "My Orders", "Overdue", "Unassigned", "Critical", "All"
  - Счётчики бейджей на каждом чипе
  - Sync с URL search params для shareable links
- [ ] **P0-2.5** Создать `frontend/src/components/work-orders/WOKanbanBoard.tsx`:
  - Колонки: New → Assigned → In Progress → Completed → Cancelled
  - Drag-and-drop (использовать `@dnd-kit/core` или `react-beautiful-dnd`)
  - Карточка WO: title, device, priority badge, SLA progress bar, assignee avatar
  - Toggle view: Table ↔ Kanban (иконка-переключатель)
- [ ] **P0-2.6** Обновить `frontend/src/pages/WorkOrders.tsx`:
  - Интегрировать QuickFilters сверху
  - Интегрировать обновлённый DataGrid + Kanban toggle
  - Bulk action toolbar
- **Критерий приёмки:** Bulk actions работают (assign 10+ WO), Kanban drag-and-drop, QuickFilters с URL sync

### P0-3: Редизайн SpareParts (Shelf.nu паттерн)
- [ ] **P0-3.1** Создать `frontend/src/components/spare-parts/PartCard.tsx`:
  - Изображение запчасти (placeholder если нет)
  - Название, SKU, категория
  - Stock уровень с цветовой индикацией: 🟢 OK / 🟡 Low / 🔴 Out of stock
  - QR-код кнопка (использовать уже установленный `qrcode`)
- [ ] **P0-3.2** Создать `frontend/src/components/spare-parts/PartsGridView.tsx`:
  - Toggle: Table ↔ Grid (карточки)
  - Grid: responsive 2/3/4 колонки
  - Low-stock визуальный акцент (красная рамка + иконка ⚠️)
- [ ] **P0-3.3** Добавить bulk operations в Parts DataGrid:
  - Mass update stock
  - Mass change location
  - Export selected
- [ ] **P0-3.4** Создать `frontend/src/components/spare-parts/PartHistoryTimeline.tsx`:
  - История перемещений и использований запчасти
  - Привязка к WorkOrders
- [ ] **P0-3.5** Обновить `frontend/src/pages/SpareParts.tsx`:
  - Интегрировать Grid/Table toggle
  - Интегрировать PartCard
  - Добавить "Low Stock" quick filter
- **Критерий приёмки:** Card view с фото и QR, low-stock индикаторы, bulk stock update

### P0-4: Редизайн SLADashboard
- [ ] **P0-4.1** Создать `frontend/src/components/ui/Gauge.tsx`:
  - Круговая диаграмма (SVG-based, без тяжёлых chart-библиотек)
  - Props: value (0-100), thresholds (green/yellow/red), label, size
  - Анимация при mount
- [ ] **P0-4.2** Создать `frontend/src/components/sla/SLAGaugePanel.tsx`:
  - 4 gauge: Overall Compliance %, MTTR Compliance %, Preventive Compliance %, Emergency Response %
  - Цветовая индикация: 🟢 ≥95% / 🟡 80-94% / 🟠 60-79% / 🔴 <60%
- [ ] **P0-4.3** Создать `frontend/src/components/sla/SLAHeatmap.tsx`:
  - Строки: Sites, Колонки: месяцы/недели
  - Цвет ячейки: compliance % (green → red)
  - Tooltip с деталями при hover
- [ ] **P0-4.4** Создать `frontend/src/components/sla/SLATrendChart.tsx`:
  - Line chart: SLA compliance за 30/90/180 дней
  - Использовать recharts (уже в dependencies)
  - Target line (95%) как reference
- [ ] **P0-4.5** Создать `frontend/src/components/sla/SLABreachTimeline.tsx`:
  - Список breach-событий: когда, какое устройство, какой SLA, насколько просрочен
  - Фильтр по severity
- [ ] **P0-4.6** Обновить `frontend/src/pages/SLADashboard.tsx`:
  - Top: SLAGaugePanel (4 метрики)
  - Middle left: SLATrendChart, Middle right: SLAHeatmap
  - Bottom: SLABreachTimeline
  - Убрать "голые таблицы" как основной view
- **Критерий приёмки:** 4 gauge-метрики сверху, heatmap по сайтам, trend-график, breach-timeline

### P0-5: Создать AuditTimeline organism
- [ ] **P0-5.1** Создать `frontend/src/components/ui/Timeline.tsx` (если нет или доработать существующий):
  - Вертикальная timeline с иконками по типу события
  - Поддержка: status_change, note, photo, part_used, assignment, system
  - Diff-view для изменений (old → new с подсветкой)
  - Expandable details
- [ ] **P0-5.2** Создать `frontend/src/components/work-orders/WOAuditLog.tsx`:
  - Полная история изменений WO
  - Фильтр по типу события
  - Экспорт audit log (CSV)
- [ ] **P0-5.3** Создать `frontend/src/components/devices/DeviceAuditLog.tsx`:
  - История изменений устройства
  - Привязка к WO и maintenance events
- [ ] **P0-5.4** Интегрировать AuditLog в WorkOrderDetail (отдельная вкладка или панель)
- [ ] **P0-5.5** Интегрировать AuditLog в DeviceDetail (отдельная вкладка)
- **Критерий приёмки:** Timeline показывает все изменения с diff-view, фильтр и экспорт работают


### P0-6: Calendar View для WorkOrders (HubEx pattern)
- [ ] Создать `frontend/src/components/work-orders/WorkOrderCalendar.tsx`
- [ ] Интегрировать FullCalendar (уже в deps!)
- [ ] Drag-and-drop для изменения сроков/исполнителей
- [ ] Визуализация загрузки техников
- [ ] Toggle: Table ↔ Calendar ↔ Kanban

### P0-7: QR Scanner в mobile app (HubEx pattern)
- [ ] Создать `mobile/src/screens/QRScannerScreen.tsx`
- [ ] Использовать `expo-camera` для сканирования
- [ ] Deep link на DeviceDetail / Create WO
- [ ] Batch QR generation для инвентаризации

### P0-8: Электронная подпись (HubEx pattern)
- [ ] Создать `mobile/src/screens/SignatureScreen.tsx`
- [ ] Использовать `react-native-signature-canvas`
- [ ] Сохранение подписи в WO (base64)
- [ ] Интеграция с Gatekeeper verification

### P0-9: Camera Specs Database Integration
- [ ] Импортировать `cameras.json` из cctv-camera-database в PostgreSQL reference: https://github.com/viruz2701/cctv-camera-database
- [ ] Создать API endpoint `/api/v1/camera-models/{brand}/{model}`
- [ ] Интегрировать в Device Creation Wizard (автозаполнение)
- [ ] Добавить compatibility checker (PoE, protocols)

### P1-6: Auto-dispatcher Service (HubEx pattern)
- [ ] Создать `backend/internal/cmms/dispatcher.go`
- [ ] Алгоритм: skills + workload + location matching
- [ ] Auto-escalation при просрочке SLA
- [ ] Rules engine для custom logic

---

## 🟠 P1 — Важно (Q4 2026, до 2026-12-31)

### P1-1: Трёхколоночный layout для WorkOrderDetail (Atlas CMMS паттерн)
- [ ] **P1-1.1** Создать `frontend/src/components/layout/ThreeColumnTemplate.tsx`:
  - Left (25%): Metadata, Status, Priority, SLA Timer, Timeline
  - Center (50%): Checklist, Notes, Photos, Before/After
  - Right (25%): Device Info, Parts Used, Labor, Related WOs
  - Responsive: на mobile — single column с accordion
- [ ] **P1-1.2** Создать `frontend/src/components/work-orders/SLATimer.tsx`:
  - Countdown timer до SLA deadline
  - Цветовая индикация: 🟢 on track / 🟡 at risk / 🔴 breached
  - Пульсация при <1 часа до breach
- [ ] **P1-1.3** Рефакторинг `WorkOrderDetail.tsx`:
  - Заменить текущий layout на ThreeColumnTemplate
  - Интегрировать WODetailHeader (уже существует, sticky)
  - Интегрировать WODetailInfo, WODetailParts, WODetailPhotos, WODetailTime, WODetailTimeline (все уже существуют — переместить в колонки)
  - Добавить SLATimer в левую колонку
- **Критерий приёмки:** 3-колоночный layout, responsive, SLA timer с countdown

### P1-2: Design System v2 — недостающие атомы и молекулы
- [ ] **P1-2.1** Создать `frontend/src/components/ui/Tabs.tsx` (если не создан в P0-1.3)
- [ ] **P1-2.2** Создать `frontend/src/components/ui/Tooltip.tsx`
- [ ] **P1-2.3** Создать `frontend/src/components/ui/Dropdown.tsx` (с keyboard navigation)
- [ ] **P1-2.4** Создать `frontend/src/components/ui/Skeleton.tsx` (loading states)
- [ ] **P1-2.5** Создать `frontend/src/components/ui/EmptyState.tsx` (illustrated empty states)
- [ ] **P1-2.6** Создать молекулу `frontend/src/components/molecules/SLAProgressBar.tsx`:
  - Linear progress bar с цветовой индикацией
  - Показывает: elapsed / remaining / total
  - Текст: "2ч 15м осталось" или "Просрочен на 45м"
- [ ] **P1-2.7** Создать молекулу `frontend/src/components/molecules/PriorityPicker.tsx`:
  - Visual picker: Critical 🔴 / High 🟠 / Medium 🟡 / Low 🟢
  - Keyboard accessible
- [ ] **P1-2.8** Создать молекулу `frontend/src/components/molecules/TechnicianSelector.tsx`:
  - Combobox с аватарами
  - Показывает: имя, роль, текущая загрузка (workload)
  - Group by team
- [ ] **P1-2.9** Создать молекулу `frontend/src/components/molecules/DateRangePicker.tsx`:
  - Пресеты: Today, Last 7 days, Last 30 days, This month, Custom
  - Calendar popup
  - Использовать `date-fns` (уже в deps)
- [ ] **P1-2.10** Создать organism `frontend/src/components/organisms/BeforeAfterSlider.tsx`:
  - Сравнение фото до/после (Gatekeeper integration)
  - Draggable divider
- **Критерий приёмки:** Все компоненты в Storybook, WCAG AA, dark mode, documented props

### P1-3: Performance Optimization
- [ ] **P1-3.1** Code splitting — `React.lazy` + `Suspense` на каждый роут:
  - Проверить `frontend/src/App.tsx` или router config
  - Каждая page = lazy import
  - Skeleton loading state для каждого Suspense boundary
- [ ] **P1-3.2** Image optimization pipeline:
  - Добавить `vite-imagetools` или `sharp` в build
  - Конвертация в WebP/AVIF
  - Responsive images (srcset) для DeviceDetail фото
- [ ] **P1-3.3** Bundle analysis в CI:
  - Добавить `rollup-plugin-visualizer`
  - Budget: initial JS < 200KB gzipped, per-route < 50KB
  - Fail CI если budget превышен
- [ ] **P1-3.4** Memoization audit:
  - Пройтись по DataGrid строкам — добавить `React.memo`
  - Проверить `useMemo` для тяжёлых вычислений (TCO, SLA)
  - Проверить `useCallback` для event handlers в списках
- [ ] **P1-3.5** React Query prefetch:
  - Prefetch detail page data на hover (link prefetch)
  - Stale time tuning для разных entities
- **Критерий приёмки:** Lighthouse Performance > 90, initial bundle < 200KB gzip

### P1-4: Accessibility CI
- [ ] **P1-4.1** Интегрировать `@axe-core/playwright` в e2e тесты
- [ ] **P1-4.2** Автоматические color-blind simulation тесты (daltonize)
- [ ] **P1-4.3** Создать `docs/ux/keyboard-navigation-map.md`
- [ ] **P1-4.4** Аудит reduced motion support (`prefers-reduced-motion`)
- **Критерий приёмки:** 0 critical axe violations в CI, keyboard nav map задокументирован

### P1-5: State Management Cleanup
- [ ] **P1-5.1** Зафиксировать в ADR: "Только TailwindCSS, без Material-UI"
  - Проверить `package.json` — удалить `@mui/*` если присутствует
  - Проверить импорты во всех файлах
- [ ] **P1-5.2** Миграция Context → React Query + Zustand:
  - `DevicesSitesContext` → React Query `useDevices()`, `useSites()`
  - `AlertsContext` → React Query `useAlerts()` + WebSocket subscription
  - `TicketsContext` → React Query `useTickets()`
  - `WorkOrdersContext` → React Query `useWorkOrders()`
  - `SparePartsContext` → React Query `useSpareParts()`
  - Оставить Context только для: Theme, Auth (session), UI state
- [ ] **P1-5.3** Убрать дублирование: если данные в React Query — не дублировать в Context
- [ ] **P1-5.4** Создать `frontend/src/domains/cmms/` и `frontend/src/domains/monitoring/`:
  - Перенести domain-specific hooks, components, types
  - Feature-sliced design для CMMS-модуля
- **Критерий приёмки:** < 5 Context'ов, ADR зафиксирован, domain folders созданы

### P1-7: Smart Device Onboarding Wizard
- [ ] Шаг 1: IP → auto-detect model
- [ ] Шаг 2: Compatibility check
- [ ] Шаг 3: Capacity calculation
- [ ] Шаг 4: QR code generation
- [ ] Шаг 5: Create WorkOrder

---

## 🟡 P2 — Желательно (Q1 2027, до 2027-03-31)

### P2-1: Mobile Offline-First
- [ ] **P2-1.1** Архитектурное решение: WatermelonDB vs PowerSync vs RxDB
  - Написать ADR с анализом
  - Учитывать: React Native + Expo 52, конфликт resolution, attachment sync
- [ ] **P2-1.2** Service Worker для PWA:
  - Cache-first для статики
  - Network-first для API
  - Offline fallback page
- [ ] **P2-1.3** Background sync:
  - Queue для offline WO updates
  - Conflict resolution strategy (last-write-wins + manual merge)
  - Visual indicator: online/offline/syncing
- [ ] **P2-1.4** QR scanner integration:
  - `expo-camera` для сканирования QR устройств/запчастей
  - Deep link на DeviceDetail / PartDetail
- [ ] **P2-1.5** Photo annotation tools:
  - Drawing на фото (стрелки, текст, highlights)
  - Использовать существующий `PhotoAnnotation.tsx` как базу
- **Критерий приёмки:** WO creation/editing работает offline, sync при reconnect

### P2-2: Asset Hierarchy Tree
- [ ] **P2-2.1** Создать `frontend/src/components/organisms/AssetTree.tsx`:
  - Иерархия: Site → Building → Floor → Room → Device
  - Drag-and-drop для перемещения
  - Expand/collapse с lazy loading детей
  - Search/filter внутри дерева
- [ ] **P2-2.2** Интегрировать в Sites page как alternative view
- [ ] **P2-2.3** Breadcrumbs на основе позиции в иерархии
- **Критерий приёмки:** Дерево рендерит 1000+ узлов с virtualization, search < 100ms

### P2-3: Advanced Analytics Dashboard
- [ ] **P2-3.1** Predictive maintenance widget:
  - "Устройства, требующие внимания в следующие 7 дней"
  - На основе ML-моделей из `backend/analytics/predict.py`
- [ ] **P2-3.2** Cost analysis dashboard:
  - TCO breakdown по сайтам/типам устройств
  - Интегрировать данные из `mv_tco_per_device` materialized view
- [ ] **P2-3.3** Vendor performance scorecards:
  - Рейтинг производителей по MTBF/MTTR
  - Данные из `mv_device_reliability` materialized view
- **Критерий приёмки:** Дашборд загружается < 2s, данные актуальны (materialized view refresh)

### P2-4: Global Command Palette ⌘K Enhancement
- [ ] **P2-4.1** Расширить существующий CommandPalette:
  - Поиск по WO, Devices, Sites, Parts, Users
  - Quick actions: "Create WO", "Go to Settings", "Switch Site"
  - Recent items
  - Keyboard hints
- [ ] **P2-4.2** Категоризация результатов с иконками
- **Критерий приёмки:** Поиск < 50ms, все entities индексируются, fuzzy matching

---

## 🟢 P3 — Nice-to-Have (Q2 2027, до 2027-06-30)

### P3-1: AI-ассистент в UI
- [ ] **P3-1.1** Chat-панель с DeepSeek integration
- [ ] **P3-1.2** Контекстные подсказки: "Похожие WO", "Рекомендуемые запчасти"
- [ ] **P3-1.3** Natural language поиск: "покажи все просроченные наряды на cameras в Минске"

### P3-2: Real-time Collaboration
- [ ] **P3-2.1** WebSocket для совместного редактирования WO
- [ ] **P3-2.2** Presence indicators ("Техник Иванов сейчас просматривает этот WO")
- [ ] **P3-2.3** Real-time обновления в Kanban board

### P3-3: White-label Theming
- [ ] **P3-3.1** CSS custom properties для enterprise-клиентов
- [ ] **P3-3.2** Custom logo, colors, favicon per tenant
- [ ] **P3-3.3** Branding в PDF-отчётах (ReportGenerator)

### P3-4: Voice Commands
- [ ] **P3-4.1** Speech-to-text для создания заметок в WO (hands-free для техников)
- [ ] **P3-4.2** Voice status update: "Наряд 1234 завершён"

---

## 📐 Инфраструктурные задачи (параллельно)

### Infra-1: Testing
- [ ] **Infra-1.1** Unit tests для всех новых UI-компонентов (Vitest + React Testing Library)
- [ ] **Infra-1.2** E2E tests для P0 flows (Playwright):
  - WO creation flow
  - Bulk actions flow
  - SLA dashboard load
  - Settings tab navigation
- [ ] **Infra-1.3** Visual regression tests (Chromatic или Percy)

### Infra-2: Documentation
- [ ] **Infra-2.1** Storybook для всех атомов/молекул/организмов
- [ ] **Infra-2.2** Обновить `ARCHITECTURE.md` после рефакторинга state management
- [ ] **Infra-2.3** UX-документация: user flows для Technician, Manager, Admin
- [ ] **Infra-2.4** Обновить `.clinerules` с новыми правилами для CMMS-домена

### Infra-3: i18n
- [ ] **Infra-3.1** Аудит: все новые строки добавлены в 17 языков
- [ ] **Infra-3.2** Автоматическая проверка: CI fail если есть untranslated keys
- [ ] **Infra-3.3** Fallback chain: current lang → English → hardcoded default

---

## 📊 Метрики успеха

| Метрика | Текущее | Цель P0 | Цель P1 | Цель P2 |
|---|---|---|---|---|
| UX-зрелость CMMS | 5/10 | 7/10 | 8.5/10 | 9/10 |
| Settings.tsx строк | 953 | <200 | <200 | <200 |
| Lighthouse Performance | ~70 | >80 | >90 | >95 |
| Initial bundle (gzip) | ? | <250KB | <200KB | <180KB |
| axe violations (critical) | ? | <5 | 0 | 0 |
| Context count | 14 | 14 | <5 | <5 |
| Mobile offline | 0/10 | 0/10 | 3/10 | 7/10 |
| Storybook coverage | ~30% | 50% | 80% | 95% |
| E2E test coverage | ? | P0 flows | P0+P1 flows | All flows |

---

## 📝 Правила для Roo при работе с TODO

1. **Перед началом задачи:** Прочитать соответствующий раздел, проверить зависимости (другие задачи которые должны быть завершены)
2. **Во время работы:** Коммитить атомарно, в сообщении указывать ID задачи (например: `P0-1.3: create Tabs atom component`)
3. **После завершения:** Отметить [x] + дата, проверить критерий приёмки, обновить метрику
4. **Если задача слишком большая:** Разбить на подзадачи с суффиксами (.1, .2, ...)
5. **Никогда не пропускать:** Критерий приёмки — если он не выполнен, задача не завершена
6. **Code review чеклист для каждой задачи:**
   - [ ] Dark mode работает
   - [ ] i18n: все строки через t()
   - [ ] WCAG AA: keyboard accessible, aria-labels
   - [ ] Responsive: проверено на 375px, 768px, 1440px
   - [ ] Нет console errors/warnings
   - [ ] <500 строк в одном файле
   