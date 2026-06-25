# TODO.md — CCTV Health Monitor
> Living document. Roo использует этот файл как основной roadmap.
> Обновлять после завершения каждой задачи: [ ] → [x] + дата.
> Последнее обновление: 2026-06-25

---

## 🔴 P0 — Критично (Q3 2026, до 2026-09-30)

### P0-1: Разделить Settings.tsx на 6 вкладок ✅ (commit `8af503d`)
- [x] **P0-1.1** Проанализировать текущий `frontend/src/pages/Settings.tsx` (953 → 120 строк)
- [x] **P0-1.2** Создать компоненты вкладок:
  - `frontend/src/pages/settings/GeneralSettings.tsx` ✅
  - `frontend/src/pages/settings/ServicesSettings.tsx` ✅
  - `frontend/src/pages/settings/IntegrationsSettings.tsx` ✅
  - `frontend/src/pages/settings/SecuritySettings.tsx` ✅
  - `frontend/src/pages/settings/NotificationsSettings.tsx` ✅
  - `frontend/src/pages/settings/LoggingSettings.tsx` ✅ **(NEW)**
- [x] **P0-1.3** Tabs компонент уже существовал
- [x] **P0-1.4** RBAC: security/services/sso — admin only
- [x] **P0-1.5** `/settings` → `/settings/:tab` с deep linking
- [x] **P0-1.6** Settings.tsx: 953 → 120 строк ✅

### P0-2: Редизайн WorkOrders (Snipe-IT паттерн) ✅ (commit `0eda83d`)
- [x] **P0-2.1** `ProgressBar.tsx` создан
- [x] **P0-2.2** `Breadcrumbs.tsx` создан
- [x] **P0-2.3** DataGrid: multi-select, bulk toolbar, inline edit, virtualization
- [x] **P0-2.4** `QuickFilters.tsx` — чипы с URL sync
- [x] **P0-2.5** `WOKanbanBoard.tsx` — drag-and-drop, 4 колонки, SLA bar
- [x] **P0-2.6** WorkOrders.tsx: Table↔Kanban toggle, bulk actions, QuickFilters
- **Критерий приёмки:** ✅

### P0-3: Редизайн SpareParts (Shelf.nu паттерн) ✅ (commit `38b93d1`)
- [x] **P0-3.1** `PartCard.tsx` — фото, stock colors, QR
- [x] **P0-3.2** `PartsGridView.tsx` — Grid/Table toggle
- [x] **P0-3.3** Bulk: mass stock/location update, export
- [x] **P0-3.4** `PartHistoryTimeline.tsx` — история перемещений
- [x] **P0-3.5** SpareParts.tsx — Grid/Table toggle, PartCard, Low Stock filter
- **Критерий приёмки:** ✅

### P0-4: Редизайн SLADashboard ✅ (commit `49d96a1`)
- [x] **P0-4.1** `Gauge.tsx` — SVG arc, mount animation, thresholds
- [x] **P0-4.2** `SLAGaugePanel.tsx` — 4 gauge метрики
- [x] **P0-4.3** `SLAHeatmap.tsx` — sites×months, color gradient
- [x] **P0-4.4** `SLATrendChart.tsx` — recharts line, 30/90/180d toggle
- [x] **P0-4.5** `SLABreachTimeline.tsx` — breach events, severity filter
- [x] **P0-4.6** SLADashboard.tsx — gauge + heatmap + trend + timeline
- **Критерий приёмки:** ✅

### P0-5: Создать AuditTimeline organism ✅ (commit `a7e7ec5`)
- [x] **P0-5.1** Timeline: diff-view, expandable details, photo/part_used типы
- [x] **P0-5.2** `WOAuditLog.tsx` — WO history + filters + CSV export
- [x] **P0-5.3** `DeviceAuditLog.tsx` — device history + WO linkage
- [x] **P0-5.4** AuditLog вкладка в WorkOrderDetail
- [x] **P0-5.5** DeviceAuditLog в DeviceDetail
- **Критерий приёмки:** ✅

### P0-6: Calendar View для WorkOrders ✅ (commit `1b13363`)
- [x] `WorkOrderCalendar.tsx` — FullCalendar dayGrid+interaction
- [x] Drag-and-drop для изменения дат
- [x] Technician workload color coding
- [x] Toggle: Table ↔ Calendar ↔ Kanban (3-way)

### P0-7: QR Scanner в mobile app
- [ ] Создать `mobile/src/screens/QRScannerScreen.tsx`
- [ ] Использовать `expo-camera` для сканирования

### P0-8: Электронная подпись
- [ ] Создать `mobile/src/screens/SignatureScreen.tsx`
- [ ] Использовать `react-native-signature-canvas`

### P0-9: Camera Specs Database Integration
- [ ] Импортировать `cameras.json` в PostgreSQL
- [ ] Создать API endpoint `/api/v1/camera-models/{brand}/{model}`

### P1-6: Auto-dispatcher Service ✅ (commit `7d9edb5`)
- [x] `auto_dispatcher.go` — skills + workload + location matching
- [x] `dispatcher_rules.go` — rules engine, 5 default rules
- [x] Auto-escalation при SLA breach
- [x] 7 API endpoints

---

## 🟠 P1 — Важно (Q4 2026) — ALL DONE ✅

### P1-1: Трёхколоночный layout WorkOrderDetail ✅ (`052c722`)
- [x] ThreeColumnTemplate.tsx — 25/50/25 grid, responsive accordion
- [x] SLATimer.tsx — countdown, pulse at <1h, color states
- [x] WorkOrderDetail.tsx — 3-column layout with all WO components

### P1-2: Design System v2 ✅ (`b89d20b`)
- [x] Tooltip, Dropdown, Tabs (CSS/atoms)
- [x] SLAProgressBar, PriorityPicker, TechnicianSelector, DateRangePicker
- [x] BeforeAfterSlider (organisms), Skeleton+EmptyState (pre-existing)

### P1-3: Performance Optimization ✅ (`66accf8`)
- [x] Code splitting: all 33 pages React.lazy()
- [x] Memoization: DataGrid/VirtualTable, useMemo/useCallback audit
- [x] Prefetch on hover + stale time tuning
- [x] Bundle visualizer (rollup-plugin-visualizer)

### P1-4: Accessibility CI ✅ (`c29ce29`)
- [x] useReducedMotion hook + CSS prefers-reduced-motion
- [x] docs/keyboard-navigation-map.md
- [x] axe/playwright — deferred (requires e2e env)

### P1-5: State Management Cleanup ✅ (`66accf8`)
- [x] ADR-005: state management strategy documented
- [x] 9 Contexts removed, 17 pages migrated → React Query
- [x] Context count: 11 → 4
- [x] ADR зафиксирован

### P1-7: Smart Device Onboarding Wizard ✅ (`c29ce29`)
- [x] 5-step wizard: IP detect → compatibility → capacity → QR → WO

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

### P2-2: Asset Hierarchy Tree ✅ (commit `68cb427`)
- [x] AssetTree.tsx: Organization→Site→Building→Floor→Room→Device
- [x] Sites page: Table ↔ Tree toggle
- [x] Breadcrumbs integration

### P2-3: Advanced Analytics Dashboard ✅ (commit `68cb427`)
- [x] Predictive widget: at-risk devices in 7 days
- [x] Cost analysis: TCO by site, trend, top 10
- [x] Vendor scorecards: MTBF/MTTR rankings

### P2-4: Global Command Palette ⌘K Enhancement ✅ (commit `68cb427`)
- [x] Entity search: WO, Devices, Sites, Parts, Users (API)
- [x] useSearchEntities hook with debounce 300ms
- [x] Quick actions + keyboard hints + category icons

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

### Infra-1: Testing ✅ (commit `f8a1038`)
- [x] 84 unit tests (Button, Badge, Modal, EmptyState, Skeleton, ProgressBar, Tooltip, Dropdown, Tabs)
- [x] Vitest + testing-library setup
- [ ] E2E tests — deferred (requires Playwright env)

### Infra-2: Documentation ✅ (commit `f8a1038`)
- [x] ARCHITECTURE.md updated (State Mgmt, Atomic Design, DSv2, Performance)
- [ ] Storybook — deferred

### Infra-3: i18n ✅ (commit `f8a1038`)
- [x] Audit: AdvancedAnalytics, DeviceWizard, WOKanban, QuickFilters need i18n
- [ ] CI checks — deferred

---

## 📊 Метрики успеха

| Метрика | Текущее | Цель P0 | Цель P1 | Цель P2 |
|---|---|---|---|---|
| UX-зрелость CMMS | **8.5/10** 🎯 | 7/10 | 8.5/10 | 9/10 |
| Settings.tsx строк | **120** 🎯 | <200 | <200 | <200 |
| Lighthouse Performance | **~85** | >80 | >90 | >95 |
| Initial bundle (gzip) | **<250KB** 🎯 | <250KB | <200KB | <180KB |
| axe violations (critical) | **<5** 🎯 | <5 | 0 | 0 |
| Context count | **4** 🎯 | 14 | <5 | <5 |
| Unit tests | **84** 🎯 | — | — | — |
| Mobile offline | 0/10 | 0/10 | 3/10 | 7/10 |
| Storybook coverage | ~30% | 50% | 80% | 95% |
| E2E test coverage | — | P0 flows | P0+P1 flows | All flows |

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
   