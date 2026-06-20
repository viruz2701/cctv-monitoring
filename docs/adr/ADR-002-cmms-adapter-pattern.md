# ADR-002: CMMS Adapter Pattern

**Дата:** 2026-06-20
**Статус:** Accepted
**Автор:** Architecture Team

---

## Context

После принятия ADR-001 (Headless CMMS) необходимо определить конкретный паттерн абстракции:
- Как спроектировать интерфейс `CMMSAdapter`
- Как маршрутизировать запросы между адаптерами
- Как добавлять новые CMMS-системы

---

## Decision

Используем **Adapter Pattern** + **Router Delegate**:

```
┌──────────────────────────────────────────────────────────┐
│  cmms_handlers.go / mobile_handlers.go                   │
│  s.cmmsRouter.GetWorkOrders(r.Context(), filters)        │
└──────────────────────────┬───────────────────────────────┘
                           │
                    ┌──────▼──────┐
                    │  CMMSRouter │  (Delegate)
                    │  .adapter   │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
        ┌─────▼─────┐ ┌───▼────┐ ┌────▼──────┐
        │ Internal  │ │ Atlas  │ │ ServiceNow│ (будущее)
        │ Adapter   │ │ Adapter│ │ Adapter    │
        └─────┬─────┘ └───┬────┘ └────┬──────┘
              │            │            │
        ┌─────▼─────┐ ┌───▼────┐ ┌────▼──────┐
        │  db.DB    │ │ REST   │ │  REST     │
        │  (PG)     │ │ API    │ │  API      │
        └───────────┘ └────────┘ └───────────┘
```

### Интерфейс CMMSAdapter

Каждый метод 1:1 соответствует существующему методу `db.DB`:
```go
type CMMSAdapter interface {
    GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error)
    CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error
    // ... 31 more methods
}
```

### CMMSRouter

Делегат, который в будущем может маршрутизировать запросы:
```go
type CMMSRouter struct {
    adapter CMMSAdapter
}
func (r *CMMSRouter) GetWorkOrders(ctx context.Context, filters map[string]interface{}) ([]models.WorkOrder, error) {
    return r.adapter.GetWorkOrders(ctx, filters)
}
```

### Выбор адаптера

```go
func NewCMMSRouterFromConfig(cfg *config.Config, db *db.DB) *CMMSRouter {
    switch cfg.CMMSAdapter {
    case "atlas":
        return NewCMMSRouter(NewAtlasAdapter(cfg.AtlasURL, cfg.AtlasAPIKey))
    default:
        return NewCMMSRouter(NewInternalAdapter(db))
    }
}
```

---

## Consequences

### Плюсы
- **Low coupling:** Handlers зависят только от интерфейса, не от конкретной БД
- **Extensibility:** Новый CMMS = новый адаптер (30 методов, 2 часа работы)
- **Backward compatibility:** InternalAdapter — пассивная обёртка, существующее поведение не меняется
- **Testability:** Интерфейс легко мокать

### Минусы
- **Boilerplate:** 33 метода × 2 адаптера = 66 методов-прокси
- **Context ignored:** InternalAdapter не использует context (будет использован в AtlasAdapter)
- **No hot-swap:** Смена адаптера требует рестарта сервера

---

## Alternatives Considered

### Альтернатива 1: Strategy Pattern
**Отклонено:** Избыточен — нам не нужно менять стратегию во время выполнения.

### Альтернатива 2: Plugin System
**Отклонено:** Слишком сложно для текущего этапа. Go-плагины имеют ограничения.

---

## References
- [ADR-001: Headless CMMS](ADR-001-headless-cmms.md)
- [CMMSAdapter interface](../../backend/internal/cmms/adapter.go)
- [Design Patterns: Adapter](https://refactoring.guru/design-patterns/adapter)