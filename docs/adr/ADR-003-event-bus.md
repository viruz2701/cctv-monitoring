# ADR-003: Event Bus Architecture

**Дата:** 2026-06-20
**Статус:** Accepted
**Автор:** Architecture Team

---

## Context

CCTV Intelligence Platform состоит из нескольких сервисов:
- **Backend** (Go) — API, CMMS, аутентификация
- **P2P Gateway** (Go + Rust) — подключение к камерам через P2P
- **WebSocket Hub** — real-time алерты
- **Worker** (Go) — фоновая обработка (SLA monitoring, maintenance scheduling)
- **Mobile App** (React Native) — push-уведомления

Необходим механизм коммуникации между сервисами для:
- Алертов (камера → backend → WebSocket → UI)
- Событий CMMS (work order created → push-уведомление технику)
- Gatekeeper верификации (mobile → backend → AI сервис)
- SLA monitoring (cron → проверка дедлайнов → алерты)

---

## Decision

**Phase 1-3 (Текущая): NATS**
**Phase 4 (High-load): Kafka**

### NATS

```
┌──────────┐    ┌──────────┐    ┌──────────┐
│ Backend  │    │  Worker  │    │ P2P GW   │
└────┬─────┘    └────┬─────┘    └────┬─────┘
     │               │               │
     └───────┬───────┴───────┬───────┘
             │               │
        ┌────▼───────────────▼────┐
        │       NATS Server       │
        │  (JetStream optional)   │
        └──────────┬──────────────┘
                   │
        ┌──────────▼──────────┐
        │   WebSocket Hub     │
        │   (real-time push)  │
        └─────────────────────┘
```

### Топики NATS

| Топик | Publisher | Subscribers | Назначение |
|-------|-----------|-------------|------------|
| `alarms.{device_id}` | Backend | WS Hub | Новый аларм от устройства |
| `cmms.workorder.created` | Backend | Worker, Mobile | Создан наряд → push + SLA мониторинг |
| `cmms.workorder.completed` | Backend | Worker | Закрыт наряд → обновление статистики |
| `cmms.sla.breached` | Worker | Backend, Mobile | SLA breached → алерт |
| `gatekeeper.verify` | Mobile | Backend | Запрос верификации |
| `gatekeeper.result` | Backend | Mobile | Результат верификации |
| `devices.status.{id}` | P2P GW | Backend, WS Hub | Статус устройства изменился |

### Почему NATS, а не Kafka (Phase 1-3)

| Критерий | NATS | Kafka |
|----------|------|-------|
| Сложность развёртывания | 1 бинарник, 0 зависимостей | ZooKeeper/KRaft, брокеры |
| Потребление ресурсов | Минимальное (50MB RAM) | Значительное (2GB+ RAM) |
| Скорость | До 10M msg/sec | До 1M msg/sec |
| At-least-once delivery | Через JetStream | Built-in |
| Persistence | JetStream (опционально) | Всегда |
| Подходит для | Текущий масштаб (сотни камер) | Тысячи камер, enterprise |

**Вывод:** NATS покрывает потребности Phase 1-3 с минимальным operational overhead. Kafka будет рассмотрен в Phase 4 при росте до тысяч камер.

---

## Consequences

### Плюсы
- **Low latency:** NATS — один из самых быстрых message brokers
- **Simple:** Один процесс, не требует ZooKeeper
- **Go-native:** Хорошая библиотека `nats.go`
- **Gradual:** JetStream можно включить позже для persistence

### Минусы
- **No persistence by default:** Нужно включать JetStream для гарантированной доставки
- **Migration path:** При переходе на Kafka нужно будет переписать publishers/subscribers
- **Less ecosystem:** Меньше tooling чем у Kafka (Kafka Connect, KSQL, etc.)

---

## Alternatives Considered

### Альтернатива 1: RabbitMQ
**Отклонено:** Сложнее в эксплуатации, требует Erlang.

### Альтернатива 2: Redis Pub/Sub
**Отклонено:** Нет гарантий доставки, нет persistence.

### Альтернатива 3: Прямые HTTP-вызовы
**Отклонено:** Tight coupling, нет асинхронности, сложно масштабировать.

---

## Implementation Plan

1. Добавить `nats.go` в Go модуль
2. Создать `internal/nats/` пакет с publisher/subscriber
3. Интегрировать в сервер при старте
4. Обновить WebSocket Hub для подписки на NATS

---

## References
- [NATS Documentation](https://docs.nats.io/)
- [NATS vs Kafka](https://docs.nats.io/compare-nats)
- [nats.go](https://github.com/nats-io/nats.go)