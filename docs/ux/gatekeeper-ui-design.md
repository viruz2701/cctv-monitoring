# Gatekeeper UI Design — Экран верификации

**Дата:** 2026-06-20
**Назначение:** Wireframe и спецификация экрана верификации для Mobile App
**Цель:** Предотвратить «диванное ТО» — закрытие наряда без фактического выезда

---

## 1. User Flow

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Work Order  │    │  Checklist   │    │  Gatekeeper  │    │  Complete    │
│  Detail      │───→│  Screen      │───→│  Verification│───→│  Work Order  │
│              │    │              │    │  Screen      │    │  (Success)   │
│  [Start]     │    │  [Tasks]     │    │  [GPS/EXIF]  │    │              │
│  [Photo ДО]  │    │  [Photos]    │    │  [AI Check]  │    │  [Done]      │
└──────────────┘    └──────────────┘    └──────────────┘    └──────────────┘
```

**Новый flow:**
1. Техник начинает наряд → **обязательное фото ДО**
2. Выполняет чек-лист
3. Делает фото ПОСЛЕ
4. **Экран Gatekeeper** — проверка всех условий
5. Если все ✅ → подпись → закрытие наряда

---

## 2. Wireframe экрана Gatekeeper

```
┌──────────────────────────────────────────────┐
│  ← Gatekeeper Verification                   │
├──────────────────────────────────────────────┤
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │  📍 GPS Location                     │    │
│  │  ┌────────────────────────────────┐  │    │
│  │  │ ✅ Distance to site: 23m       │  │    │
│  │  │    Accuracy: 5m                │  │    │
│  │  │    Site: "Склад №3"            │  │    │
│  │  └────────────────────────────────┘  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │  📸 Photo EXIF Validation            │    │
│  │  ┌────────────────────────────────┐  │    │
│  │  │ ✅ GPS in EXIF: 55.75, 37.62  │  │    │
│  │  │ ✅ Timestamp: 2026-06-20 14:30│  │    │
│  │  │ ✅ Device: iPhone 15 Pro      │  │    │
│  │  └────────────────────────────────┘  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │  🤖 AI Photo Comparison              │    │
│  │  ┌────────────────────────────────┐  │    │
│  │  │ ✅ Before/After match: 94%    │  │    │
│  │  │    Detected: Camera cleaned   │  │    │
│  │  │    [Before] [After]           │  │    │
│  │  └────────────────────────────────┘  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │  📝 Checklist                        │    │
│  │  ┌────────────────────────────────┐  │    │
│  │  │ ✅ 4/4 tasks completed        │  │    │
│  │  └────────────────────────────────┘  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │  ✍️ Signature                        │    │
│  │  ┌────────────────────────────────┐  │    │
│  │  │ ✅ Signature captured          │  │    │
│  │  └────────────────────────────────┘  │    │
│  └──────────────────────────────────────┘    │
│                                              │
│  ┌──────────────────────────────────────┐    │
│  │                                      │    │
│  │  All checks passed! ✅               │    │
│  │                                      │    │
│  │  [ Complete Work Order ]             │    │
│  │                                      │    │
│  └──────────────────────────────────────┘    │
│                                              │
└──────────────────────────────────────────────┘
```

---

## 3. Состояния проверок

### 3.1 GPS Location

| Состояние | Иконка | Цвет | Описание |
|-----------|--------|------|----------|
| OK | ✅ | green | Расстояние < 500м, accuracy < 50м |
| Warning | ⚠️ | yellow | Расстояние 500-1000м или accuracy 50-100м |
| Error | ❌ | red | Расстояние > 1000м или GPS недоступен |
| Pending | ⏳ | gray | Ожидание получения координат |

### 3.2 EXIF Validation

| Состояние | Иконка | Цвет | Описание |
|-----------|--------|------|----------|
| OK | ✅ | green | EXIF содержит GPS и timestamp, они валидны |
| Warning | ⚠️ | yellow | EXIF есть, но GPS отсутствует или timestamp старый |
| Error | ❌ | red | EXIF отсутствует (фото из галереи) или поддельное |
| Pending | ⏳ | gray | Ожидание анализа EXIF |

### 3.3 AI Comparison

| Состояние | Иконка | Цвет | Описание |
|-----------|--------|------|----------|
| OK | ✅ | green | Сходство > 80%, изменения детектированы |
| Warning | ⚠️ | yellow | Сходство 50-80%, возможно не тот объект |
| Error | ❌ | red | Сходство < 50% или нет фото ДО |
| Pending | ⏳ | gray | Ожидание AI-анализа |

### 3.4 Checklist

| Состояние | Иконка | Цвет | Описание |
|-----------|--------|------|----------|
| OK | ✅ | green | Все пункты выполнены |
| Incomplete | ⚠️ | yellow | Не все пункты отмечены |
| Empty | ❌ | red | Чек-лист пуст |

---

## 4. Обработка ошибок

### 4.1 GPS недоступен

```
┌──────────────────────────────────────┐
│  ⚠️ GPS Location                     │
│  ┌────────────────────────────────┐  │
│  │ ⚠️ GPS signal is weak          │  │
│  │    Accuracy: 150m (too low)    │  │
│  │                                │  │
│  │ [Retry] [Skip with approval]   │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

### 4.2 EXIF невалиден

```
┌──────────────────────────────────────┐
│  ❌ Photo EXIF Validation             │
│  ┌────────────────────────────────┐  │
│  │ ❌ No EXIF data in photo       │  │
│  │    Photo may be from gallery   │  │
│  │                                │  │
│  │ [Take new photo]               │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

### 4.3 AI comparison failed

```
┌──────────────────────────────────────┐
│  ❌ AI Photo Comparison              │
│  ┌────────────────────────────────┐  │
│  │ ❌ Before/After mismatch: 12%  │  │
│  │    Different location detected │  │
│  │                                │  │
│  │ [Retake photos] [Report issue] │  │
│  └────────────────────────────────┘  │
└──────────────────────────────────────┘
```

---

## 5. API Endpoint

**POST** `/api/v1/mobile/work-orders/{id}/verify`

Request:
```json
{
  "gps": {
    "latitude": 55.751244,
    "longitude": 37.618423,
    "accuracy": 5.0,
    "timestamp": "2026-06-20T14:30:00Z"
  },
  "photo_exif": {
    "gps_latitude": 55.751244,
    "gps_longitude": 37.618423,
    "date_time_original": "2026-06-20T14:30:00Z",
    "make": "Apple",
    "model": "iPhone 15 Pro"
  },
  "photo_before_url": "https://...",
  "photo_after_url": "https://...",
  "checklist_completed": true,
  "signature": "base64..."
}
```

Response:
```json
{
  "verified": true,
  "checks": {
    "gps": {"passed": true, "distance_m": 23, "accuracy_m": 5},
    "exif": {"passed": true, "has_gps": true, "timestamp_valid": true},
    "ai": {"passed": true, "similarity": 0.94, "changes_detected": ["camera_cleaned"]},
    "checklist": {"passed": true, "completed": 4, "total": 4},
    "signature": {"passed": true}
  }
}
```

---

## 6. Компоненты для создания

| Компонент | Приоритет | Назначение |
|-----------|-----------|------------|
| VerificationCheck | P0 | Row с иконкой, названием и статусом проверки |
| GPSStatusCard | P0 | Карточка статуса GPS |
| EXIFStatusCard | P0 | Карточка статуса EXIF |
| AICheckCard | P1 | Карточка результатов AI |
| GatekeeperScreen | P0 | Экран верификации |
| BeforeAfterSlider | P1 | Слайдер сравнения фото ДО/ПОСЛЕ |

---

## 7. Навигационные изменения

Добавить в [`AppNavigator.tsx`](../../mobile/src/navigation/AppNavigator.tsx):

```typescript
<Stack.Screen
  name="Gatekeeper"
  component={GatekeeperScreen}
  options={{ title: 'Верификация' }}
/>
```

Flow: `WorkOrderDetail → Checklist → PhotoCapture → Gatekeeper → Signature → Complete`