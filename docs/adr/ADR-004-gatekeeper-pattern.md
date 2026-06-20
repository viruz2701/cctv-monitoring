# ADR-004: Gatekeeper Pattern

**Дата:** 2026-06-20
**Статус:** Proposed
**Автор:** Architecture Team

---

## Context

В системах технического обслуживания существует проблема «диванного ТО» (couch maintenance) — когда техник закрывает наряд, не выезжая на объект. Это критично для CCTV-инфраструктуры, где камеры расположены на удалённых объектах.

Текущий процесс закрытия наряда в Mobile App:
1. Чек-лист ✅
2. Фото ✅
3. Подпись ✅
4. Закрытие наряда

**Проблема:** Нет верификации, что техник действительно находится на объекте и фото сделаны на месте.

---

## Decision

Внедряем **Gatekeeper Service** — сервис верификации, который проверяет 3 условия перед закрытием наряда:

```
┌──────────────────────────────────────────────────────────┐
│                    Gatekeeper Service                     │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │ GPS Verify   │  │ EXIF Verify  │  │ AI Verify    │   │
│  │              │  │              │  │              │   │
│  │ Distance to  │  │ GPS in EXIF  │  │ Before/After │   │
│  │ site < 500m  │  │ Timestamp OK │  │ similarity   │   │
│  │ Accuracy<50m │  │ Device match │  │ > 80%        │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
│         │                 │                 │            │
│         └─────────┬───────┴─────────┬───────┘            │
│                   │                 │                    │
│              ┌────▼─────────────────▼────┐               │
│              │   Verification Result     │               │
│              │   ALL CHECKS PASSED       │               │
│              └───────────────────────────┘               │
└──────────────────────────────────────────────────────────┘
```

### Три уровня верификации

#### 1. GPS Verification
- **Вход:** Координаты техника + координаты site
- **Проверка:** Haversine distance < 500 метров
- **Проверка:** GPS accuracy < 50 метров
- **Проверка:** Timestamp GPS в пределах ±5 минут

#### 2. EXIF Verification
- **Вход:** EXIF-метаданные фото (GPS, DateTimeOriginal, Make, Model)
- **Проверка:** GPS в EXIF соответствует координатам site
- **Проверка:** DateTimeOriginal в пределах ±1 часа
- **Проверка:** EXIF не пустой (фото не из галереи)

#### 3. AI Verification (Phase 2+)
- **Вход:** Фото ДО и ПОСЛЕ выполнения работ
- **Проверка:** Сходство > 80% (тот же объект)
- **Проверка:** Детектированы изменения (камера очищена, кабель заменён)

### API Endpoint

**POST** `/api/v1/mobile/work-orders/{id}/verify`

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

### Graceful Degradation

Если GPS недоступен (подвал, плохая погода):
- Показать warning
- Разрешить skip с комментарием
- Записать в audit log

Если EXIF отсутствует:
- Заблокировать закрытие
- Потребовать новое фото через камеру (не галерею)

---

## Consequences

### Плюсы
- **100% верификация:** Каждый наряд проверяется по 3 критериям
- **Audit trail:** Все результаты верификации сохраняются
- **Mobile-first:** Проверки происходят на устройстве + сервере
- **Graceful:** GPS можно пропустить с обоснованием

### Минусы
- **Зависимость от GPS:** В помещении сигнал слабый
- **EXIF не всегда:** Некоторые камеры не пишут EXIF
- **AI требует GPU:** Для Phase 2 нужен GPU-сервер
- **User experience:** Дополнительный шаг перед закрытием

---

## Alternatives Considered

### Альтернатива 1: Только GPS
**Отклонено:** GPS можно подделать (mock location).

### Альтернатива 2: Только фото
**Отклонено:** Фото можно сделать заранее.

### Альтернатива 3: QR-код на объекте
**Частично принято:** QR-сканер уже есть, но это дополнительный фактор, не замена Gatekeeper.

### Альтернатива 4: Биометрия (face ID)
**Отклонено:** Избыточно для текущего этапа.

---

## Implementation Plan

### Phase 1 (Mobile + Backend)
1. EXIF-извлечение в Mobile App (через `exif-js` или `expo-image-manipulator`)
2. GPS-верификация на backend
3. Gatekeeper экран в Mobile App
4. API endpoint `POST /verify`

### Phase 2 (AI)
5. Before/After фото сравнение
6. AI-модель для детекции изменений
7. Интеграция с GPU-сервером

---

## References
- [Gatekeeper UI Design](../ux/gatekeeper-ui-design.md)
- [Mobile Current State](../ux/mobile-current-state.md)
- [Mobile API: completeMobileWorkOrder](../../backend/internal/api/mobile_handlers.go)