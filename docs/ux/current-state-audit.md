# UX Current State Audit — CCTV Intelligence Platform

**Дата:** 2026-06-20
**Статус:** Phase 0 — Research
**Версия:** 1.0

---

## 1. Список всех существующих страниц

### 1.1 Core Pages

| Страница | Файл | Статус | Описание |
|----------|------|--------|----------|
| Dashboard | [`Dashboard.tsx`](../../frontend/src/pages/Dashboard.tsx) | ✅ Хорошо | Фильтруемая сводка: устройства, тикеты, алерты. StatsCard + AlertBanner + последние тикеты. |
| Devices | [`Devices.tsx`](../../frontend/src/pages/Devices.tsx) | ✅ Хорошо | Таблица устройств с фильтрами по сайту/типу/статусу. |
| DeviceDetail | [`DeviceDetail.tsx`](../../frontend/src/pages/DeviceDetail.tsx) | ✅ Хорошо | Детальная карточка устройства: метрики, камеры, логи. |
| Sites | [`Sites.tsx`](../../frontend/src/pages/Sites.tsx) | ✅ Хорошо | Карточки сайтов с устройствами и алармами. |
| Alerts | [`Alerts.tsx`](../../frontend/src/pages/Alerts.tsx) | ✅ Хорошо | Таблица алертов с фильтрами по severity/device/времени. |
| Analytics | [`Analytics.tsx`](../../frontend/src/pages/Analytics.tsx) | ⚠️ Базовая | Графики и предиктивная аналитика. Требует улучшения визуализации. |
| Logs | [`Logs.tsx`](../../frontend/src/pages/Logs.tsx) | ✅ Хорошо | Поиск по логам с фильтрами device/level/keyword. |

### 1.2 CMMS Pages (Целевые для редизайна)

| Страница | Файл | Статус | Описание |
|----------|------|--------|----------|
| WorkOrders | [`WorkOrders.tsx`](../../frontend/src/pages/WorkOrders.tsx) | ⚠️ Требует редизайна | Таблица + фильтры + модалки. 238 строк. Таблица с кнопками Start/Complete/Cancel. Нет bulk actions, нет drag-and-drop. |
| SpareParts | [`SpareParts.tsx`](../../frontend/src/pages/SpareParts.tsx) | ⚠️ Требует редизайна | Таблица + поиск + кнопки +/-. 140 строк. Карточки (Shelf.nu style) не используются. |
| SLADashboard | [`SLADashboard.tsx`](../../frontend/src/pages/SLADashboard.tsx) | ⚠️ Требует редизайна | Две таблицы (config + compliance). 95 строк. Нет визуальных графиков, gauge-метрик. |
| MaintenanceSchedules | [`MaintenanceSchedules.tsx`](../../frontend/src/pages/MaintenanceSchedules.tsx) | ⚠️ Базовая | Календарь/список ТО. Нет календарного вида. |
| MaintenanceReports | [`MaintenanceReports.tsx`](../../frontend/src/pages/MaintenanceReports.tsx) | ⚠️ Базовая | Отчёты по ТО. Требует улучшения. |
| TechnicianDashboard | [`TechnicianDashboard.tsx`](../../frontend/src/pages/TechnicianDashboard.tsx) | ⚠️ Базовая | Дашборд техника. Требует редизайна под Snipe-IT паттерны. |

### 1.3 Admin Pages

| Страница | Файл | Статус | Описание |
|----------|------|--------|----------|
| Settings | [`Settings.tsx`](../../frontend/src/pages/Settings.tsx) | ⚠️ Требует вкладок | 953 строки. Все настройки на одной странице. Нужно разделить на вкладки: General, Services, Integrations, Security. |
| Users | [`Users.tsx`](../../frontend/src/pages/Users.tsx) | ✅ Хорошо | Управление пользователями с RBAC. |
| APIKeys | [`APIKeys.tsx`](../../frontend/src/pages/APIKeys.tsx) | ✅ Хорошо | Создание/отзыв API-ключей. |
| Notifications | [`Notifications.tsx`](../../frontend/src/pages/Notifications.tsx) | ✅ Хорошо | Push-уведомления. |
| Tickets | [`Tickets.tsx`](../../frontend/src/pages/Tickets.tsx) | ✅ Хорошо | Тикет-система. |
| TicketDetail | [`TicketDetail.tsx`](../../frontend/src/pages/TicketDetail.tsx) | ✅ Хорошо | Детали тикета с комментариями. |

### 1.4 Auth Pages

| Страница | Файл | Статус |
|----------|------|--------|
| Login | [`Login.tsx`](../../frontend/src/pages/Login.tsx) | ✅ Хорошо |
| ForgotPassword | [`ForgotPassword.tsx`](../../frontend/src/pages/ForgotPassword.tsx) | ✅ Хорошо |
| Profile | [`Profile.tsx`](../../frontend/src/pages/Profile.tsx) | ✅ Хорошо |

---

## 2. UI-атомы (Design System)

### 2.1 Существующие компоненты

| Компонент | Файл | Варианты | Оценка |
|-----------|------|----------|--------|
| Card | [`Card.tsx`](../../frontend/src/components/ui/Card.tsx) | default, elevated, bordered | ✅ Полный. Есть CardHeader, CardBody, CardFooter. |
| Table | [`Table.tsx`](../../frontend/src/components/ui/Table.tsx) | sortable, expandable, loading skeleton | ✅ Полный. Есть Pagination. |
| Button | [`Button.tsx`](../../frontend/src/components/ui/Button.tsx) | primary, secondary, outline, ghost, danger | ✅ Полный. IconButton, loading state. |
| Badge | [`Badge.tsx`](../../frontend/src/components/ui/Badge.tsx) | StatusBadge, HealthBadge, PriorityBadge, TicketStatusBadge, RoleBadge | ✅ Полный. |
| Modal | [`Modal.tsx`](../../frontend/src/components/ui/Modal.tsx) | default, ConfirmModal | ✅ Полный. |
| Input | [`Input.tsx`](../../frontend/src/components/ui/Input.tsx) | SearchInput, Select, Textarea | ✅ Полный. |
| StatsCard | [`StatsCard.tsx`](../../frontend/src/components/ui/StatsCard.tsx) | default, MiniStatsCard | ✅ Хорошо. |
| Toast | [`Toast.tsx`](../../frontend/src/components/ui/Toast.tsx) | ToastProvider, useToast | ✅ Полный. |

### 2.2 Отсутствующие компоненты (нужно создать)

| Компонент | Назначение | Приоритет |
|-----------|-----------|-----------|
| Tabs | Вкладки для Settings и страниц | HIGH |
| Skeleton | Улучшенный skeleton loading | MEDIUM |
| Tooltip | Подсказки при наведении | MEDIUM |
| Dropdown Menu | Контекстное меню | MEDIUM |
| Progress Bar | Индикаторы прогресса (SLA и др.) | HIGH |
| Gauge | Круговые метрики для SLA | HIGH |
| Timeline | Хронология изменений (audit log) | MEDIUM |
| DataGrid | Таблица с inline-редактированием (Snipe-IT style) | HIGH |
| QRCode | Отображение QR-кодов для устройств | MEDIUM |
| FileUpload | Drag-and-drop загрузка файлов | MEDIUM |

---

## 3. Контексты (State Management)

| Контекст | Файл | Назначение |
|----------|------|------------|
| WorkOrdersContext | [`WorkOrdersContext.tsx`](../../frontend/src/context/WorkOrdersContext.tsx) | CRUD нарядов, start, complete, cancel |
| SparePartsContext | [`SparePartsContext.tsx`](../../frontend/src/context/SparePartsContext.tsx) | CRUD запчастей, adjustStock |
| MaintenanceContext | [`MaintenanceContext.tsx`](../../frontend/src/context/MaintenanceContext.tsx) | Графики ТО, maintenance reports |
| AlertsContext | [`AlertsContext.tsx`](../../frontend/src/context/AlertsContext.tsx) | WebSocket-алерты |
| DevicesSitesContext | [`DevicesSitesContext.tsx`](../../frontend/src/context/DevicesSitesContext.tsx) | Устройства и сайты |
| SettingsContext | [`SettingsContext.tsx`](../../frontend/src/context/SettingsContext.tsx) | Настройки, services |
| ThemeContext | [`ThemeContext.tsx`](../../frontend/src/context/ThemeContext.tsx) | Dark mode |
| NotificationsContext | [`NotificationsContext.tsx`](../../frontend/src/context/NotificationsContext.tsx) | Push-уведомления |
| ReportsContext | [`ReportsContext.tsx`](../../frontend/src/context/ReportsContext.tsx) | Отчёты |
| TicketsContext | [`TicketsContext.tsx`](../../frontend/src/context/TicketsContext.tsx) | Тикет-система |
| UsersContext | [`UsersContext.tsx`](../../frontend/src/context/UsersContext.tsx) | Управление пользователями |

---

## 4. Data Model (из [`types/index.ts`](../../frontend/src/types/index.ts))

Ключевые сущности:
- **Device** — камера/NVR/DVR/switch с connectionType (ip/p2p/snmp/gb28181/onvif)
- **Site** — площадка с адресом
- **WorkOrder** — наряд на ТО (type, status, priority, SLA deadline, checklist, photos, parts)
- **SparePart** — запчасть (SKU, stock, min_stock, location, cost)
- **MaintenanceSchedule** — график ТО (schedule_type, interval_days, next_due)
- **SLAConfig** — SLA-политики (response_time, resolution_time)
- **TechnicianSiteAssignment** — привязка техника к площадке

---

## 5. Итоговая оценка

### Страницы, требующие немедленного редизайна:
1. **WorkOrders** — применить Snipe-IT таблицы (bulk actions, inline edit, фильтры в заголовках)
2. **SpareParts** — применить Shelf.nu карточки (изображение, custom fields, QR workflow)
3. **SLADashboard** — добавить gauge-метрики, progress bars, временные ряды
4. **Settings** — разделить на вкладки (General, Services, Integrations, Security)
5. **MaintenanceSchedules** — добавить календарный вид

### Страницы, уже соответствующие стандартам:
- Dashboard, Devices, DeviceDetail, Alerts, Tickets, Users, APIKeys

### Компоненты, которые нужно создать:
- Tabs, DataGrid, Gauge, Progress, Timeline, QRCode, FileUpload