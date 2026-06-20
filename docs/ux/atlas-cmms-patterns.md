# Atlas CMMS UX Patterns Reference

**Дата:** 2026-06-20
**Референс:** Atlas CMMS — Enterprise Computerized Maintenance Management System

---

## 1. Обзор Atlas CMMS

Atlas CMMS — enterprise-система управления ТО. Ключевые UX-паттерны, релевантные для нашего проекта:

### 1.1 Work Order Layout (Three-Column)

**Паттерн:** Atlas использует трёхколоночный layout для наряда:
```
┌──────────────┬──────────────────────────────┬──────────────┐
│  LEFT PANEL  │       MAIN CONTENT           │ RIGHT PANEL  │
│              │                              │              │
│  Status      │  Checklist                   │  Asset Info  │
│  Priority    │  ☐ Task 1                    │  Location    │
│  Assigned    │  ☑ Task 2 (completed)        │  Last TO     │
│  SLA Timer   │  ☐ Task 3                    │  Warranty    │
│              │                              │              │
│  Attachments │  Notes & Photos              │  Parts Used  │
│  (photos)    │  [Photo 1] [Photo 2]         │  • Part A x2 │
│              │                              │  • Part B x1 │
│  Timeline    │  [Add Photo] [Add Note]      │              │
│  (activity)  │                              │  Labor       │
│              │                              │  Duration    │
└──────────────┴──────────────────────────────┴──────────────┘
```

**Применение:**
- [`WorkOrderDetail`](../../frontend/src/pages/) — создать новую страницу с трёхколоночным layout
- [`WorkOrderDetailScreen.tsx`](../../mobile/src/screens/WorkOrderDetailScreen.tsx) — адаптировать для mobile (вертикальный scroll)

### 1.2 Task Checklists with Progress

**Паттерн:** Atlas показывает прогресс-бар выполнения чеклиста:
```
┌────────────────────────────────────────────┐
│  Checklist Progress                        │
│  ████████████████░░░░░░░░ 75% (3/4 tasks)  │
│                                            │
│  ☑ Check power supply                      │
│  ☑ Clean lens                              │
│  ☑ Verify network connectivity             │
│  ☐ Update firmware                         │
└────────────────────────────────────────────┘
```

**Применение:**
- Уже есть в [`ChecklistScreen.tsx`](../../mobile/src/screens/ChecklistScreen.tsx)
- Добавить progress bar в Desktop WorkOrderDetail

### 1.3 Mobile-First Verification

**Паттерн:** Atlas mobile app требует:
- **GPS-координаты** при закрытии наряда (гео-привязка к объекту)
- **Фото ДО и ПОСЛЕ** (before/after comparison)
- **Подпись клиента** (signature capture)
- **QR-код объекта** (подтверждение, что техник на месте)

**Применение:**
- Уже есть: PhotoCapture, Signature, QRScanner
- Нужно добавить: GPS verification, Before/After comparison, Gatekeeper экран

### 1.4 SLA Visual Indicators

**Паттерн:** Atlas использует цветовые индикаторы SLA:
- 🟢 Green: > 50% времени осталось
- 🟡 Yellow: 25-50% времени осталось
- 🟠 Orange: < 25% времени осталось
- 🔴 Red: SLA breached

С визуальным таймером обратного отсчёта.

**Применение:**
- Уже есть в [`WorkOrders.tsx`](../../frontend/src/pages/WorkOrders.tsx) (SLA иконки)
- Улучшить: добавить progress bar с оставшимся временем

### 1.5 Asset Hierarchy Tree

**Паттерн:** Atlas показывает иерархию активов:
```
📁 Site A
  ├─ 📁 Building 1
  │   ├─ 📹 Camera 1 (Front Entrance)
  │   ├─ 📹 Camera 2 (Parking)
  │   └─ 🖥 NVR 1
  └─ 📁 Building 2
      ├─ 📹 Camera 3 (Warehouse)
      └─ 🔌 Switch 1
```

**Применение:**
- [`Sites.tsx`](../../frontend/src/pages/Sites.tsx) — добавить tree view устройств
- Полезно для навигации при создании Work Order

---

## 2. Ключевые отличия Atlas CMMS

| Паттерн | Shelf.nu | Snipe-IT | Atlas CMMS |
|---------|----------|----------|------------|
| Карточки активов | ✅ | ❌ | ❌ |
| Таблицы с фильтрами | ❌ | ✅ | ✅ |
| Трёхколоночный layout | ❌ | ❌ | ✅ |
| GPS верификация | ❌ | ❌ | ✅ |
| Before/After фото | ❌ | ❌ | ✅ |
| SLA с таймером | ❌ | ❌ | ✅ |
| Иерархия активов | ❌ | ❌ | ✅ |

---

## 3. Рекомендованные изменения

### 3.1 WorkOrderDetail → Three-Column Layout

Создать новую страницу [`WorkOrderDetail.tsx`](../../frontend/src/pages/WorkOrderDetail.tsx):
- Левая панель: Status, Priority, Assigned, SLA Timer, Timeline
- Центр: Checklist с progress, Notes, Photos
- Правая панель: Device Info, Parts Used, Labor

### 3.2 SLA Progress Bar

Добавить компонент `SLAProgress`:
```typescript
interface SLAProgressProps {
  deadline: Date;
  createdAt: Date;
  status: 'on_track' | 'at_risk' | 'breached';
}
```

### 3.3 Before/After Photo Comparison

Для Gatekeeper: компонент сравнения фото ДО и ПОСЛЕ выполнения работ.

---

## 4. Ссылки

- [Atlas CMMS](https://www.atlascmms.com)
- [Atlas CMMS Features](https://www.atlascmms.com/features)