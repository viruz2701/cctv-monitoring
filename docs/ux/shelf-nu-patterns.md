# Shelf.nu UX Patterns Reference

**Дата:** 2026-06-20
**Референс:** [Shelf.nu](https://shelf.nu) — Open Source Asset Management

---

## 1. Обзор Shelf.nu

Shelf.nu — open-source система управления активами (Asset Management), построенная на React + Remix. Ключевые UX-паттерны, релевантные для нашего CMMS:

### 1.1 Карточки активов (Asset Cards)

**Паттерн:** Каждый актив представлен карточкой с:
- Изображением/иконкой устройства
- Названием и тегами
- Статусом (Available, Checked Out, Maintenance)
- Местоположением
- QR-кодом для быстрого сканирования
- Кастомными полями (custom fields)

**Применение в нашем проекте:**
- [`SpareParts.tsx`](../../frontend/src/pages/SpareParts.tsx) — сейчас таблица, переделать на карточки с изображением, SKU, stock level, QR-кодом
- [`Devices.tsx`](../../frontend/src/pages/Devices.tsx) — добавить QR-код для каждого устройства

### 1.2 QR Workflow

**Паттерн:** Shelf.nu использует QR-коды для:
- Быстрой идентификации актива (сканирование → мгновенный переход к карточке)
- Check-in/Check-out процесса
- Инвентаризации

**Применение:**
- Mobile App: уже есть [`QRScannerScreen.tsx`](../../mobile/src/screens/QRScannerScreen.tsx)
- Desktop: добавить генерацию QR-кодов для SpareParts и Devices
- Work Orders: сканирование QR устройства → автоматическое создание наряда

### 1.3 Custom Fields System

**Паттерн:** Shelf.nu позволяет добавлять произвольные поля к активам:
- Типы полей: Text, Number, Date, Boolean, Select, Multi-select
- Визуально отображаются в карточке актива

**Применение:**
- SpareParts: custom fields для специфичных атрибутов (voltage, connector type, compatibility)
- Devices: custom fields для метаданных

### 1.4 Dashboard Cards Layout

**Паттерн:** Shelf.nu использует masonry/grid layout с карточками:
- Карточка «Total Assets» с иконкой
- Карточка «Checked Out» с предупреждением
- Карточка «Maintenance» с индикатором
- Карточка «Recent Activity» с timeline

**Применение:**
- [`Dashboard.tsx`](../../frontend/src/pages/Dashboard.tsx) — уже использует StatsCard, но можно добавить больше CMMS-метрик
- [`TechnicianDashboard.tsx`](../../frontend/src/pages/TechnicianDashboard.tsx) — переделать под карточки

### 1.5 Booking / Reservation System

**Паттерн:** Shelf.nu имеет систему бронирования активов с календарём

**Применение:**
- Maintenance Schedules: календарный вид для планирования ТО

---

## 2. Цветовая схема Shelf.nu

| Элемент | Цвет |
|---------|------|
| Primary | `#2563EB` (blue-600) |
| Success | `#16A34A` (green-600) |
| Warning | `#F59E0B` (amber-500) |
| Danger | `#DC2626` (red-600) |
| Background | `#F8FAFC` (slate-50) |
| Card | `#FFFFFF` white |

**Совместимость с нашим проектом:** Полная. Уже используем blue-600 как акцентный.

---

## 3. Рекомендованные изменения

### 3.1 SpareParts → Shelf.nu Card Grid

**Текущее состояние:** Таблица с колонками name, SKU, category, stock, cost, location, actions.

**Целевое состояние:**
```
┌──────────────────────────────────────────────────────────────┐
│  [Search]                                          [+ Add]   │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ [IMG]        │  │ [IMG]        │  │ [IMG]        │          │
│  │ PoE Switch   │  │ IR Illumin. │  │ BNC Cable    │          │
│  │ SKU: PS-001  │  │ SKU: IR-002 │  │ SKU: BC-003 │          │
│  │ 📦 5 / min 2 │  │ 📦 12/min 5 │  │ ⚠ 2 / min 10│          │
│  │ $129.99      │  │ $49.99      │  │ $5.99       │          │
│  │ Location: WH │  │ Location: WH │  │ Location: S1│          │
│  │ [QR] [Edit]  │  │ [QR] [Edit]  │  │ [QR] [Edit] │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└──────────────────────────────────────────────────────────────┘
```

### 3.2 Devices → QR Code Generation

Добавить кнопку «Generate QR» в [`DeviceDetail.tsx`](../../frontend/src/pages/DeviceDetail.tsx) и [`Devices.tsx`](../../frontend/src/pages/Devices.tsx), генерирующую QR-код с URL на устройство.

### 3.3 Custom Fields для SpareParts

Добавить поддержку custom fields в модель SparePart:
```typescript
interface SparePart {
  // ... existing fields
  custom_fields?: Record<string, string | number | boolean>;
}
```

---

## 4. Ссылки

- [Shelf.nu GitHub](https://github.com/Shelf-nu/shelf.nu)
- [Shelf.nu Demo](https://demo.shelf.nu)
- [Shelf.nu Design System](https://www.shelf.nu/design)