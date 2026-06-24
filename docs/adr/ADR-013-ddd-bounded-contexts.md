# ADR-013: Domain-Driven Design Bounded Contexts

## Статус
ACCEPTED (2026-06-24)

## Контекст
CCTV Health Monitor вырос из монолитного прототипа в платформу с мониторингом, CMMS, asset management и workforce management. Для масштабирования требуется чёткое разделение на Bounded Contexts по DDD.

## Решение

### Bounded Contexts Map

```
┌─────────────────────────────────────────────────────────┐
│                    CCTV Health Monitor                    │
│                                                          │
│  ┌──────────────┐  ┌────────────┐  ┌──────────────────┐ │
│  │  Monitoring  │  │   CMMS     │  │     Assets       │ │
│  │  (CCTV Core) │──│ (Work Mgmt)│──│ (Device/Site)    │ │
│  │              │  │            │  │                  │ │
│  │• Telemetry   │  │• WorkOrders│  │• Device Registry │ │
│  │• Alerts      │  │• Schedules │  │• Site Hierarchy  │ │
│  │• VideoQ      │  │• SLA Engine│  │• Parent/Child    │ │
│  │• RCA Engine  │  │• Cost Track│  │• Location Tree   │ │
│  │• Predictions │  │• Print/PDF │  │• Asset Lifecycle │ │
│  └──────┬───────┘  └─────┬──────┘  └────────┬─────────┘ │
│         │                │                   │           │
│         ▼                ▼                   ▼           │
│  ┌──────────────┐  ┌────────────┐  ┌──────────────────┐ │
│  │  Workforce   │  │ Inventory  │  │   Integration    │ │
│  │  (Tech Mgmt) │  │ (SpareParts)│  │   (CMMS Adapter) │ │
│  │              │  │            │  │                  │ │
│  │• Technicians │  │• Parts     │  │• Atlas Adapter   │ │
│  │• Skills/Cert │  │• Stock     │  │• ServiceNow      │ │
│  │• Shifts      │  │• Vendors   │  │• Jira            │ │
│  │• Workload    │  │• PO        │  │• 1C:ТОИР         │ │
│  └──────────────┘  └────────────┘  └──────────────────┘ │
│                                                          │
│  ┌──────────────────────────────────────────────────────┐│
│  │               Shared Kernel                          ││
│  │  • models (base entities, value objects)             ││
│  │  • events (NATS JetStream schema registry)           ││
│  │  • audit (ISO 27001 A.12.4 HMAC chain)               ││
│  │  • auth (JWT + RBAC)                                 ││
│  │  • featureflag                                       ││
│  └──────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────┘
```

### Взаимодействие между контекстами

| Source Context | Event | Target Context | Канал |
|---|---|---|---|
| Monitoring | `AlertCreated` | CMMS | NATS |
| Monitoring | `DeviceOffline` | Assets | NATS |
| CMMS | `WorkOrderCompleted` | Assets | NATS |
| CMMS | `WorkOrderCreated` | Workforce | NATS |
| Inventory | `LowStockDetected` | CMMS | NATS |
| Assets | `DeviceStatusChanged` | Monitoring | NATS |
| Integration | `CMMSSyncCompleted` | CMMS | NATS |

### Агрегаты

| Контекст | Агрегат | Root Entity | Value Objects |
|---|---|---|---|
| Monitoring | Alarm | Alarm.ID | Priority, Method |
| Monitoring | Prediction | Prediction.DeviceID | FailureProbability |
| CMMS | WorkOrder | WorkOrder.ID | Status, Priority, CostBase |
| CMMS | MaintenanceSchedule | Schedule.ID | Interval, NextDue |
| CMMS | SLA Policy | SLAPolicy.ID | ResponseTime, Calendar |
| Assets | Device | Device.ID | Location, VendorType |
| Assets | Site | Site.ID | ParentLocation, Address |
| Workforce | Technician | User.ID | Skills, Certifications |
| Workforce | Shift | Shift.ID | Schedule, Assignment |
| Inventory | SparePart | Part.ID | Stock, Cost, Category |
| Inventory | PurchaseOrder | PO.ID | LineItem, Vendor, Status |

### Anti-Corruption Layer (ACL)

CMMS Integration Layer выступает как ACL между нашим CMMS Bounded Context и внешними системами (Atlas, ServiceNow, Jira, 1C:ТОИР). Каждый адаптер:

1. Транслирует внешние API в наш внутренний `CMMSAdapter` interface
2. Маппит внешние статусы/приоритеты в наши внутренние enum
3. Использует Fallback Queue для off-line tolerance (CMMS-3.1.3)

### Принципы

1. **Persistence ignorance**: Каждый контекст использует свой репозиторий
2. **Eventual consistency**: Межконтекстная коммуникация через NATS
3. **Shared Kernel**: Только базовые модели (base.go) и event schema
4. **Context isolation**: Нет прямых вызовов repository другого контекста

## Compliance

- IEC 62443-3-3 SR 1.1 (Defense in depth — изоляция контекстов)
- ISO 27001 A.8 (Asset management — контекст Assets)
- ISO 27001 A.12.4 (Event logging — межконтекстные события)
- ISO 27001 A.9.2 (Access control — RBAC per context)

## Последствия

Positive:
- Чёткое разделение ответственности
- Каждый контекст можно масштабировать независимо
- Новые CMMS адаптеры не затрагивают CCTV Core

Negative:
- Нужно следить за разрастанием Shared Kernel
- Eventual consistency требует обработки конфликтов
- Дополнительная сложность при кросс-контекстных запросах

## References
- Eric Evans, "Domain-Driven Design" (2003)
- Vaughn Vernon, "Implementing Domain-Driven Design"
- Martin Fowler, "BoundedContext" (martinfowler.com)
