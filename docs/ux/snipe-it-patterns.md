# Snipe-IT UX Patterns Reference

**Дата:** 2026-06-20
**Референс:** [Snipe-IT](https://snipeitapp.com) — Open Source IT Asset Management

---

## 1. Обзор Snipe-IT

Snipe-IT — наиболее зрелая open-source система IT Asset Management. Построена на Laravel + Bootstrap. Ключевые UX-паттерны для CMMS:

### 1.1 Таблицы с фильтрами в заголовках (DataGrid Pattern)

**Паттерн:** Таблицы Snipe-IT имеют:
- Фильтры прямо в заголовках колонок (выпадающие списки, поиск)
- Сортировку по клику на заголовок
- Bulk actions (чекбоксы слева → массовые действия в toolbar)
- Кастомизируемые колонки (пользователь выбирает видимые колонки)
- Inline-редактирование некоторых полей
- Экспорт в CSV/PDF

**Применение в нашем проекте:**
- [`WorkOrders.tsx`](../../frontend/src/pages/WorkOrders.tsx) — добавить bulk actions (массовый assign, cancel, change priority), фильтры в заголовках
- [`SpareParts.tsx`](../../frontend/src/pages/SpareParts.tsx) — bulk actions (массовое списание, перемещение)
- [`Users.tsx`](../../frontend/src/pages/Users.tsx) — bulk actions (массовая блокировка, смена роли)

### 1.2 Audit Log (Timeline View)

**Паттерн:** Snipe-IT показывает полную историю изменений каждого актива:
- Хронологическая лента (timeline)
- Кто, когда, что изменил (old_value → new_value)
- Фильтрация по типу действия
- Экспорт audit log

**Применение:**
- У нас уже есть `audit_log` таблица и `s.logAudit()` метод
- Нужно создать страницу/компонент AuditLog для просмотра истории WorkOrders и SpareParts
- Добавить вкладку «History» в карточку каждого устройства

### 1.3 Bulk Actions Toolbar

**Паттерн:**
```
┌──────────────────────────────────────────────────────────────┐
│  [☐] 3 selected  │  [Assign] [Change Status] [Delete] [..]  │
├──────────────────────────────────────────────────────────────┤
│  ☐ │ Device │ Type │ Priority │ Status │ Assigned │ SLA    │
│  ☐ │ Cam-01 │ rep. │ high     │ open   │ Ivan     │ 2h     │
│  ☑ │ Cam-02 │ prev. │ medium   │ open   │ -        │ 24h    │
│  ☑ │ NVR-01 │ rep. │ critical │ open   │ -        │ 1h     │
│  ☑ │ Switch │ prev. │ low      │ open   │ -        │ 72h    │
└──────────────────────────────────────────────────────────────┘
```

**Применение:** Все CMMS-таблицы должны поддерживать bulk selection.

### 1.4 Dashboard Widgets

**Паттерн:** Snipe-IT dashboard:
- Метрики в верхней части (Total Assets, Deployed, Ready to Deploy, Pending)
- График «Assets by Status» (pie/donut)
- График «Recent Activity» (timeline)
- Быстрые ссылки на частые действия

**Применение:**
- [`Dashboard.tsx`](../../frontend/src/pages/Dashboard.tsx) — добавить CMMS-виджеты (Open Work Orders, SLA Breaches Today, Low Stock Parts)
- [`TechnicianDashboard.tsx`](../../frontend/src/pages/TechnicianDashboard.tsx) — персонализированные виджеты

### 1.5 Advanced Search

**Паттерн:** Snipe-IT имеет мощный поиск:
- Полнотекстовый поиск по всем полям
- Сохранение поисковых запросов
- Предустановленные фильтры (Assigned to me, Overdue, etc.)

**Применение:**
- Добавить Quick Filters в WorkOrders: «My Work Orders», «Overdue», «Unassigned», «Today»

---

## 2. Цветовая схема Snipe-IT

| Элемент | Цвет |
|---------|------|
| Primary | `#337AB7` (blue) |
| Success | `#5CB85C` (green) |
| Info | `#5BC0DE` (cyan) |
| Warning | `#F0AD4E` (orange) |
| Danger | `#D9534F` (red) |
| Table header | `#F5F5F5` (light gray) |

---

## 3. Рекомендованные изменения

### 3.1 WorkOrders → Snipe-IT DataGrid

**Текущее состояние:** Простая таблица с фильтрами вверху страницы.

**Целевое состояние:**
- Фильтры в заголовках колонок
- Bulk actions с toolbar
- Inline status change (выпадающий список в ячейке)
- Кастомизируемые колонки
- Quick Filters: «My Orders», «Overdue», «Critical»

### 3.2 Audit Log Component

Создать компонент [`AuditLog.tsx`](../../frontend/src/components/) для отображения:
```
┌──────────────────────────────────────────────────────┐
│  Audit Log — Work Order #WO-2026-001                 │
│                                                      │
│  ● 2026-06-20 14:30 — Ivan Petrov                    │
│  │  Changed status: open → in_progress               │
│  │                                                   │
│  ● 2026-06-20 10:15 — System                         │
│  │  Created work order (priority: high)              │
│  │                                                   │
│  ● 2026-06-19 18:00 — Admin                          │
│     Assigned to: - → Ivan Petrov                     │
└──────────────────────────────────────────────────────┘
```

### 3.3 Quick Filters

Добавить в [`WorkOrdersContext.tsx`](../../frontend/src/context/WorkOrdersContext.tsx):
```typescript
type QuickFilter = 'all' | 'mine' | 'overdue' | 'unassigned' | 'critical' | 'today';
```

---

## 4. Ссылки

- [Snipe-IT GitHub](https://github.com/snipe/snipe-it)
- [Snipe-IT Demo](https://demo.snipeitapp.com)
- [Snipe-IT Docs](https://snipe-it.readme.io/docs)