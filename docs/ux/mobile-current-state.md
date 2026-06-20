# Mobile App Current State Audit

**Дата:** 2026-06-20
**Приложение:** React Native (Expo) — CCTV Intelligence Platform Mobile
**Целевая аудитория:** Техники, выезжающие на объекты

---

## 1. Структура навигации

```
AppNavigator
├── LoginScreen (auth gate)
└── MainTabs (Bottom Tab Navigator)
    ├── Dashboard (Мои задания)
    │   └── WorkOrderDetail → Checklist → PhotoCapture → Signature
    └── Profile (Профиль)
```

Дополнительные экраны (stack):
- QRScanner — сканирование QR-кода устройства
- WorkOrderDetail — детальная карточка наряда
- Checklist — чек-лист выполнения
- PhotoCapture — фотофиксация
- Signature — подпись клиента

---

## 2. Список экранов

### 2.1 LoginScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`LoginScreen.tsx`](../../mobile/src/screens/LoginScreen.tsx) |
| Статус | ✅ Хорошо |
| Функции | JWT-аутентификация, сохранение токена в SecureStore, offline-режим |

### 2.2 DashboardScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`DashboardScreen.tsx`](../../mobile/src/screens/DashboardScreen.tsx) |
| Статус | ✅ Хорошо |
| Функции | Список назначенных нарядов, фильтр по статусу, pull-to-refresh |
| Данные | `GET /api/v1/mobile/work-orders` |

### 2.3 WorkOrderDetailScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`WorkOrderDetailScreen.tsx`](../../mobile/src/screens/WorkOrderDetailScreen.tsx) |
| Статус | ⚠️ Требует доработки |
| Функции | Детали наряда, статус, SLA, кнопки Start/Checklist |
| Отсутствует | GPS-верификация, EXIF-проверка, Before/After сравнение |

### 2.4 ChecklistScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`ChecklistScreen.tsx`](../../mobile/src/screens/ChecklistScreen.tsx) |
| Статус | ✅ Хорошо |
| Функции | Чек-лист с чекбоксами, прогресс выполнения |

### 2.5 PhotoCaptureScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`PhotoCaptureScreen.tsx`](../../mobile/src/screens/PhotoCaptureScreen.tsx) |
| Статус | ⚠️ Требует доработки |
| Функции | Камера + галерея, загрузка фото |
| Отсутствует | EXIF-метаданные (GPS-координаты из фото), проверка timestamp |
| Есть | `useLocation()` hook — получает GPS, но не привязывает к фото |

### 2.6 SignatureScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`SignatureScreen.tsx`](../../mobile/src/screens/SignatureScreen.tsx) |
| Статус | ✅ Хорошо |
| Функции | Canvas для подписи, сохранение в base64 |

### 2.7 QRScannerScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`QRScannerScreen.tsx`](../../mobile/src/screens/QRScannerScreen.tsx) |
| Статус | ✅ Хорошо |
| Функции | Сканирование QR-кода устройства |

### 2.8 ProfileScreen

| Параметр | Значение |
|----------|----------|
| Файл | [`ProfileScreen.tsx`](../../mobile/src/screens/ProfileScreen.tsx) |
| Статус | ✅ Хорошо |
| Функции | Профиль техника, статистика, выход |

---

## 3. Как сейчас происходит закрытие наряда

Текущий flow:
```
1. Техник открывает наряд → кнопка "Start"
2. Выполняет чек-лист → ChecklistScreen
3. Делает фото → PhotoCaptureScreen
4. Получает подпись → SignatureScreen
5. Нажимает "Complete" → POST /api/v1/mobile/work-orders/{id}/complete
```

**Что отправляется в completeMobileWorkOrder:**
- `checklist` — выполненные пункты
- `photos` — URL загруженных фото
- `signature` — base64 подпись
- `notes` — заметки
- `location` — GPS (если есть)
- `parts_used` — использованные запчасти

---

## 4. Где не хватает верификации

### 4.1 GPS Verification

**Проблема:** `useLocation()` hook получает координаты, но они не верифицируются:
- Нет сравнения с координатами объекта (site)
- Нет geofence — техник может закрыть наряд из любого места
- Нет проверки точности GPS (accuracy)

**Решение:** Gatekeeper должно проверять:
- Расстояние от координат техника до координат объекта < 500м
- GPS accuracy < 50м
- Timestamp GPS в пределах ±5 минут

### 4.2 EXIF Verification

**Проблема:** Фото делаются через `expo-image-picker`, но EXIF-метаданные не извлекаются:
- Нет проверки, что фото сделано ТОЛЬКО ЧТО (не из галереи)
- Нет проверки GPS-координат в EXIF фото
- Нет проверки, что фото соответствует устройству

**Решение:**
- Извлекать EXIF (GPS, DateTimeOriginal) через `exif-js` или `expo-image-manipulator`
- Сравнивать EXIF.GPS с координатами объекта
- Проверять, что фото сделано в течение последнего часа

### 4.3 Before/After Photo Comparison

**Проблема:** Нет фото ДО начала работ — только ПОСЛЕ. Невозможно сравнить.

**Решение:**
- Требовать фото ДО (при старте наряда)
- Требовать фото ПОСЛЕ (при завершении)
- AI-сравнение на backend

### 4.4 Отсутствует экран верификации

**Проблема:** Нет единого экрана, показывающего статус всех проверок перед закрытием.

**Решение:** Создать Gatekeeper экран (см. [`gatekeeper-ui-design.md`](gatekeeper-ui-design.md)).

---

## 5. Компоненты Mobile App

| Компонент | Файл | Статус |
|-----------|------|--------|
| StatusBadge | [`StatusBadge.tsx`](../../mobile/src/components/StatusBadge.tsx) | ✅ |
| WorkOrderCard | [`WorkOrderCard.tsx`](../../mobile/src/components/WorkOrderCard.tsx) | ✅ |
| OfflineIndicator | [`OfflineIndicator.tsx`](../../mobile/src/components/OfflineIndicator.tsx) | ✅ |

---

## 6. State Management

| Store | Файл | Назначение |
|-------|------|------------|
| authStore | [`authStore.ts`](../../mobile/src/store/authStore.ts) | JWT, пользователь |
| workOrderStore | [`workOrderStore.ts`](../../mobile/src/store/workOrderStore.ts) | Кэш нарядов |
| syncStore | [`syncStore.ts`](../../mobile/src/store/syncStore.ts) | Offline очередь |

---

## 7. Выводы

### Что уже хорошо:
- Offline-режим с синхронизацией
- Полный flow закрытия наряда (checklist + фото + подпись)
- QR-сканер для идентификации устройства
- React Query для кэширования

### Что нужно добавить (Phase 1):
1. **Gatekeeper экран** — единая точка верификации перед закрытием
2. **EXIF-извлечение** — GPS и timestamp из фото
3. **Geofence-проверка** — сравнение координат техника с объектом
4. **Before/After фото** — обязательное фото ДО старта
5. **AI-верификация** — backend-проверка фото