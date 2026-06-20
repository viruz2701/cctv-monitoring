# UX Guideline — CCTV Intelligence Platform

**Дата:** 2026-06-20
**Версия:** 1.0
**Назначение:** Итоговый гайдлайн по применению UX-паттернов к страницам проекта

---

## 1. Источники паттернов

| Источник | Сильные стороны | Применяем к |
|----------|-----------------|-------------|
| **Shelf.nu** | Карточки активов, QR workflow, custom fields | SpareParts, Devices |
| **Snipe-IT** | Таблицы с фильтрами, bulk actions, audit log, quick filters | WorkOrders, Users, Tickets |
| **Atlas CMMS** | Трёхколоночный layout, SLA visual, GPS verification, before/after | WorkOrderDetail, SLADashboard, Gatekeeper |

---

## 2. Цветовая схема

| Токен | Light Mode | Dark Mode | Tailwind |
|-------|-----------|-----------|----------|
| Primary | `#3B82F6` | `#3B82F6` | `blue-600` |
| Primary Hover | `#2563EB` | `#60A5FA` | `blue-700` / `blue-400` |
| Success | `#16A34A` | `#22C55E` | `green-600` / `green-500` |
| Warning | `#F59E0B` | `#FBBF24` | `amber-500` / `amber-400` |
| Danger | `#DC2626` | `#EF4444` | `red-600` / `red-500` |
| Info | `#0EA5E9` | `#38BDF8` | `sky-500` / `sky-400` |
| Neutral | `#64748B` | `#94A3B8` | `slate-500` / `slate-400` |
| Background | `#F8FAFC` | `#0F172A` | `slate-50` / `slate-900` |
| Surface | `#FFFFFF` | `#1E293B` | `white` / `slate-800` |
| Border | `#E2E8F0` | `#334155` | `slate-200` / `slate-700` |
| Text Primary | `#0F172A` | `#F8FAFC` | `slate-900` / `slate-50` |
| Text Secondary | `#475569` | `#CBD5E1` | `slate-600` / `slate-300` |

**Правило:** Всегда используем Tailwind `dark:` префикс для dark mode. Акцентный цвет `blue-600` — неизменен в обоих режимах.

---

## 3. Типографика

| Уровень | Тег | Размер | Weight | Line Height |
|---------|-----|--------|--------|-------------|
| H1 | `<h1>` | `text-2xl` (24px) | `font-bold` (700) | `leading-tight` |
| H2 | `<h2>` | `text-xl` (20px) | `font-semibold` (600) | `leading-snug` |
| H3 | `<h3>` | `text-lg` (18px) | `font-semibold` (600) | `leading-snug` |
| Body | `<p>` | `text-sm` (14px) | `font-normal` (400) | `leading-relaxed` |
| Small | `<small>` | `text-xs` (12px) | `font-normal` (400) | `leading-normal` |
| Table Header | `<th>` | `text-xs` (12px) | `font-semibold` (600) | `uppercase tracking-wider` |

---

## 4. Spacing

| Токен | Значение | Tailwind | Применение |
|-------|----------|----------|------------|
| xs | 4px | `p-1` / `gap-1` | Иконки в кнопках |
| sm | 8px | `p-2` / `gap-2` | Внутренние отступы |
| md | 16px | `p-4` / `gap-4` | Стандартный отступ |
| lg | 24px | `p-6` / `gap-6` | Отступы между секциями |
| xl | 32px | `p-8` / `gap-8` | Страничные отступы |
| Page | 24px | `p-6` | Стандартный отступ страницы |

---

## 5. Shadows

| Токен | Tailwind | Применение |
|-------|----------|------------|
| Card | `shadow-sm` | Базовые карточки |
| Elevated | `shadow-lg` | Модальные окна, dropdown |
| None | `shadow-none` | Плоские элементы |

---

## 6. Матрица применения паттернов

| Страница | Shelf.nu | Snipe-IT | Atlas CMMS | Приоритет |
|----------|----------|----------|------------|-----------|
| **WorkOrders** | — | DataGrid, Bulk Actions, Quick Filters | — | P0 |
| **WorkOrderDetail** (новая) | — | Audit Log | Three-column, SLA Timer, Checklist | P0 |
| **SpareParts** | Card Grid, QR, Custom Fields | Bulk Actions | — | P0 |
| **SLADashboard** | — | — | SLA Visual, Gauge, Progress | P1 |
| **MaintenanceSchedules** | Calendar View | — | Asset Hierarchy | P1 |
| **TechnicianDashboard** | Dashboard Cards | Dashboard Widgets | — | P1 |
| **Settings** | — | — | — (Tabs layout) | P1 |
| **Devices** | QR Generation | — | Asset Tree | P2 |
| **DeviceDetail** | — | Audit Log | — | P2 |
| **Dashboard** | Stats Cards | CMMS Widgets | — | P2 |

---

## 7. Компоненты для создания

| Компонент | Источник паттерна | Приоритет | Назначение |
|-----------|-------------------|-----------|------------|
| DataGrid | Snipe-IT | P0 | Таблица с bulk actions, фильтрами в заголовках |
| PartCard | Shelf.nu | P0 | Карточка запчасти с изображением, QR, stock |
| SLAProgress | Atlas CMMS | P0 | Progress bar с таймером SLA |
| Gauge | Atlas CMMS | P1 | Круговая метрика compliance % |
| AuditTimeline | Snipe-IT | P1 | Хронология изменений |
| QuickFilters | Snipe-IT | P1 | Быстрые фильтры (My, Overdue, etc.) |
| QRCode | Shelf.nu | P1 | Генерация и отображение QR |
| Tabs | Custom | P1 | Вкладки для Settings |
| FileUpload | Custom | P2 | Drag-and-drop загрузка |
| BeforeAfter | Atlas CMMS | P2 | Сравнение фото ДО/ПОСЛЕ |

---

## 8. Разработка Settings по вкладкам

Страница [`Settings.tsx`](../../frontend/src/pages/Settings.tsx) (953 строки) должна быть разделена на вкладки:

| Вкладка | Содержание | Доступ |
|---------|-----------|--------|
| **General** | Site name, timezone, language, date format | Admin |
| **Services** | P2P, GB28181, SNMP, FTP, Hikvision, Dahua, Hisilicon, TVT | Admin |
| **Integrations** | CMMS Adapter (internal/atlas), Atlas URL, Atlas API Key | Admin |
| **Security** | JWT secret, API key settings, session timeout, 2FA | Admin |
| **Notifications** | Email, Telegram, Push | Admin |
| **Logging** | Log file, rotation, level | Admin |

**Правило:** Все настройки адаптеров, API и прочие системные настройки доступны **только администраторам** на странице Settings.

---

## 9. Правила code review

- Все новые компоненты должны поддерживать `dark:` режим
- Все тексты через `useTranslation()` (i18n)
- Все состояния: loading, empty, error, success
- Все кликабельные элементы: `cursor-pointer`, `hover:`, `focus:ring-2`
- Все таблицы: `aria-label`, `role` для accessibility