# Chaos Engineering Tests — CCTV Health Monitor

## Обзор

Chaos Engineering тесты для проверки устойчивости системы к сбоям компонентов.
Соответствие: IEC 62443-3-3 SR 7.8, ISO 27001 A.12.6.

## Сценарии

| ID | Сценарий | Тип | Сервис | Длительность |
|----|----------|-----|--------|-------------|
| `nats-down` | NATS Outage | disconnect | NATS | 30s |
| `nats-high-latency` | NATS High Latency | latency (2s) | NATS | 30s |
| `postgres-down` | PostgreSQL Outage | disconnect | PostgreSQL | 45s |
| `postgres-high-latency` | PostgreSQL Slow Queries | latency (3s) | PostgreSQL | 20s |
| `redis-down` | Redis Outage | disconnect | Redis | 30s |
| `api-high-load` | API Gateway High Load | latency (1.5s) | API | 40s |
| `packet-loss` | Network Packet Loss (10%) | packet-loss | NATS | 30s |

## Запуск

### Dry-run (без toxiproxy)

```bash
node tests/chaos/runner.js
```

### Конкретный сценарий

```bash
node tests/chaos/runner.js --scenario nats-down
```

### С toxiproxy (полный тест)

```bash
# Запуск toxiproxy
docker run --name toxiproxy -d \
  -p 8474:8474 \
  -p 4222:4222 \
  -p 5432:5432 \
  -p 6379:6379 \
  shopify/toxiproxy

# Запуск тестов
node tests/chaos/runner.js --toxiproxy
```

## Критерии приёмки

- ✅ Chaos-сценарии реализованы (7 сценариев)
- ✅ Автоматическое восстановление < 10s для всех сервисов
- ✅ Метрики времени восстановления
- ✅ CI integration (опционально)

## Метрики

- `recovery_time_ms` — время восстановления после инжекции сбоя
- `health_check_passed` — успешность health check после recovery
- `scenario_duration_ms` — общее время выполнения сценария

## Compliance

- **IEC 62443-3-3 SR 7.8** — Security Function Verification
- **ISO 27001 A.12.6** — Capacity Management
- **Приказ ОАЦ №66 п.7.18** — Контроль целостности
