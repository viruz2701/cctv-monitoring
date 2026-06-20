# ADR-001: Headless CMMS Pattern

**Дата:** 2026-06-20
**Статус:** Accepted
**Автор:** Architecture Team

---

## Context

CCTV Intelligence Platform требует системы управления техническим обслуживанием (CMMS) для:
- Создания и отслеживания нарядов на ТО (Work Orders)
- Управления складом запчастей (Spare Parts)
- Планирования профилактического обслуживания (Maintenance Schedules)
- SLA-контроля и отчётности

У нас уже есть работающая Internal CMMS (таблицы в БД, API-эндпоинты, UI). В будущем может потребоваться интеграция с внешними CMMS (Atlas, ServiceNow, SAP).

**Проблема:** Если мы захардкодим логику CMMS в handlers, миграция на внешнюю систему потребует полного переписывания.

---

## Decision

Используем **Headless CMMS Pattern**:

1. Определяем интерфейс [`CMMSAdapter`](../../backend/internal/cmms/adapter.go) с 33 методами (1:1 к существующим методам `db.DB`)
2. Создаём [`InternalAdapter`](../../backend/internal/cmms/internal_adapter.go) — обёртку над существующей БД
3. Создаём [`AtlasAdapter`](../../backend/internal/cmms/atlas_adapter.go) — заглушку для будущей интеграции
4. [`CMMSRouter`](../../backend/internal/cmms/adapter.go) — делегат, который можно расширить для маршрутизации между адаптерами

**Выбор адаптера** происходит через конфиг:
```yaml
cmms_adapter: "internal"  # или "atlas"
```

---

## Consequences

### Плюсы
- **Быстрый TTM:** Internal CMMS работает сразу, без переписывания
- **Гибкость:** Добавление нового CMMS = новый адаптер, handlers не меняются
- **Тестируемость:** Можно мокать интерфейс для unit-тестов
- **Миграция:** Переход на Atlas = смена одной строки в конфиге

### Минусы
- **Дублирование сигнатур:** 33 метода в интерфейсе = 33 метода в каждом адаптере
- **Контекст в интерфейсе:** `context.Context` добавлен «на будущее», InternalAdapter его игнорирует
- **Синхронизация:** При использовании AtlasAdapter нужна синхронизация данных между системами

---

## Alternatives Considered

### Альтернатива 1: Писать CMMS с нуля
**Отклонено:** Слишком долго, не даёт преимуществ перед InternalAdapter.

### Альтернатива 2: Использовать только Atlas CMMS
**Отклонено:** Зависимость от внешнего API, нет offline-режима для техников.

### Альтернатива 3: Прямые вызовы `s.db.*` в handlers (статус-кво)
**Отклонено:** Невозможно переключиться на внешний CMMS без переписывания handlers.

---

## References
- [CMMSAdapter interface](../../backend/internal/cmms/adapter.go)
- [InternalAdapter](../../backend/internal/cmms/internal_adapter.go)
- [AtlasAdapter](../../backend/internal/cmms/atlas_adapter.go)
- [CMMSRouter](../../backend/internal/cmms/adapter.go)