# P3-DR: Disaster Recovery Runbook

> **CCTV Health Monitor — КИИ РБ, класс KII-2**
>
> **Версия:** 1.0.0
> **Обновлён:** 2026-06-30
> **Владелец:** Platform Engineering
> **SLA:** RTO ≤ 15 мин, RPO ≤ 5 мин

---

## Содержание

1. [Введение](#1-введение)
2. [Архитектура DR](#2-архитектура-dr)
3. [Health Monitoring](#3-health-monitoring)
4. [Процедура Failover](#4-процедура-failover)
5. [Процедура Failback](#5-процедура-failback)
6. [Quarterly Drills](#6-quarterly-drills)
7. [RTO/RPO Dashboard](#7-rtorpo-dashboard)
8. [Compliance Checklist](#8-compliance-checklist)

---

## 1. Введение

### 1.1 Цель

Обеспечение непрерывности работы CCTV Health Monitor при отказе основного региона.

### 1.2 Область применения

Настоящий runbook описывает действия для:

- Автоматического обнаружения отказа компонентов (health checks, 30s)
- Semi-auto failover с подтверждением администратора
- DNS failover, DB promotion, NATS stream handover
- Quarterly DR drills
- Возврата к нормальной работе (failback)

### 1.3 Регионы

| Регион | Роль | Кластер | DNS Prefix |
|--------|------|---------|------------|
| `eu-central` | Primary | Frankfurt | `api.cctv.example.com` |
| `cis-east` | DR | Moscow | `dr-api.cctv.example.com` |
| `mena-gulf` | DR | Dubai | `dr-mena.cctv.example.com` |

### 1.4 Ключевые метрики

| Метрика | Цель | Критичность |
|---------|------|-------------|
| RTO (Recovery Time Objective) | ≤ 15 мин | КИИ-2 |
| RPO (Recovery Point Objective) | ≤ 5 мин | КИИ-2 |
| Health Check Interval | 30 с | P3-DR |
| Failover Detection | 3 последовательных failure | P3-DR |

---

## 2. Архитектура DR

```
┌──────────────────────────┐     ┌──────────────────────────┐
│     eu-central (Primary)  │     │     cis-east (DR)         │
│                          │     │                          │
│  ┌────────────────────┐  │     │  ┌────────────────────┐  │
│  │   API Gateway       │  │     │  │   API Gateway       │  │
│  │   (Active)          │──┼─────┼─>│   (Standby)         │  │
│  └────────────────────┘  │     │  └────────────────────┘  │
│  ┌────────────────────┐  │     │  ┌────────────────────┐  │
│  │   PostgreSQL        │  │     │  │   PostgreSQL        │  │
│  │   (Primary)         │──┼─────┼─>│   (Replica/DR)     │  │
│  └────────────────────┘  │     │  └────────────────────┘  │
│  ┌────────────────────┐  │     │  ┌────────────────────┐  │
│  │   NATS JetStream    │  │     │  │   NATS JetStream    │  │
│  │   (Active)          │──┼─────┼─>│   (Mirror)         │  │
│  └────────────────────┘  │     │  └────────────────────┘  │
│  ┌────────────────────┐  │     │  ┌────────────────────┐  │
│  │   Redis             │  │     │  │   Redis             │  │
│  │   (Primary)         │──┼─────┼─>│   (Replica)        │  │
│  └────────────────────┘  │     │  └────────────────────┘  │
└──────────────────────────┘     └──────────────────────────┘
         │                              │
         └────────── DNS ───────────────┘
                    api.cctv.example.com
```

---

## 3. Health Monitoring

### 3.1 Компоненты мониторинга

| Компонент | Проверка | Таймаут | Интервал |
|-----------|----------|---------|----------|
| PostgreSQL | `SELECT 1` (ping) | 5s | 30s |
| NATS | `FlushTimeout()` | 5s | 30s |
| Redis | `PING` | 5s | 30s |

### 3.2 Статусы

| Статус | Описание | Действие |
|--------|----------|----------|
| `healthy` | Все компоненты работают | — |
| `degraded` | 1+ компонент недоступен, но система работает | Проверить логи |
| `unavailable` | PostgreSQL недоступен | Инициировать failover |

### 3.3 Пороги failover

- **3 последовательных failure** = инициация failover (pending)
- **Admin confirm** обязателен для production
- **Auto-failover** enabled только для non-production сред

### 3.4 API

```bash
# Статус health checks
curl -X GET https://api.cctv.example.com/api/v1/dr/health

# Пример ответа
{
  "status": {
    "region": "eu-central",
    "db": {"healthy": true, "latency": "1.2ms"},
    "nats": {"healthy": true, "latency": "0.8ms"},
    "redis": {"healthy": true, "latency": "0.5ms"},
    "overall": "healthy"
  },
  "metrics": {
    "uptime_seconds": 86400,
    "check_count": 2880,
    "rto_compliance": true,
    "rpo_compliance": true
  }
}
```

---

## 4. Процедура Failover

### 4.1 Когда выполнять failover

- PostgreSQL primary недоступен > 30 секунд
- NATS cluster недоступен > 30 секунд
- Оба компонента (DB + NATS) недоступны одновременно
- По решению администратора (manual)

### 4.2 Подтверждение failover

#### Шаг 1: Проверка статуса

```bash
# Проверить текущий статус
curl -X GET https://api.cctv.example.com/api/v1/dr/health | jq .
```

#### Шаг 2: Инициация failover

```bash
# POST с телом запроса
curl -X POST https://api.cctv.example.com/api/v1/dr/failover \
  -H "Content-Type: application/json" \
  -d '{"reason": "db_unavailable", "tenant_id": "prod-001"}'
```

#### Шаг 3: Подтверждение (admin)

```bash
# Получить event_id из ответа, затем подтвердить
curl -X POST https://api.cctv.example.com/api/v1/dr/failover/{event_id}/approve \
  -H "Content-Type: application/json" \
  -d '{"approved_by": "admin@cctv.com"}'
```

### 4.3 Автоматические шаги failover

После подтверждения выполняются:

1. **DNS failover** — обновление DNS A-записей на IP DR региона
2. **DB promotion** — promotion DR PostgreSQL до primary
3. **NATS handover** — mirror → active stream promotion
4. **Post-failover health check** — проверка всех компонентов

### 4.4 Проверка после failover

```bash
# Проверить health в DR регионе
curl -X GET https://dr-api.cctv.example.com/api/v1/dr/health

# Проверить историю failover
curl -X GET https://dr-api.cctv.example.com/api/v1/dr/history
```

### 4.5 Если failover не удался

```bash
# Проверить причину ошибки
curl -X GET https://api.cctv.example.com/api/v1/dr/history | jq '.failover_history[-1]'

# Выполнить rollback
curl -X POST https://api.cctv.example.com/api/v1/dr/failover/{event_id}/rollback \
  -H "Content-Type: application/json" \
  -d '{"reason": "db_promotion_failed"}'

# Если API недоступен — ручной rollback через infra/dr/failover.sh
./infra/dr/failover.sh --from cis-east --to eu-central
```

### 4.6 Ручной DNS failover

```bash
# Cloudflare
./infra/dr/failover.sh \
  --from eu-central \
  --to cis-east \
  --provider cloudflare

# AWS Route53
./infra/dr/failover.sh \
  --from eu-central \
  --to cis-east \
  --provider route53

# Dry-run (без изменений)
./infra/dr/failover.sh \
  --from eu-central \
  --to cis-east \
  --dry-run
```

---

## 5. Процедура Failback

### 5.1 Подготовка

Перед failback убедитесь:

- [ ] Исходный primary регион полностью восстановлен
- [ ] Репликация данных работает (replication lag < RPO)
- [ ] Все компоненты healthy (health check status = "healthy")
- [ ] Проведён drill на failback

### 5.2 Шаги failback

1. **Остановить запись в DR регионе**
2. **Завершить синхронизацию** данных DR → Primary
3. **Promote Primary** PostgreSQL обратно
4. **Переключить NATS** mirror обратно
5. **Обновить DNS** на original primary IP
6. **Проверить health**

### 5.3 Failback через API

```bash
# Инициировать failback (используется стандартный failover, reversed)
curl -X POST https://dr-api.cctv.example.com/api/v1/dr/failover \
  -H "Content-Type: application/json" \
  -d '{"reason": "failback_after_recovery"}'
```

---

## 6. Quarterly Drills

### 6.1 График drills

| Тип | Периодичность | Описание | RTO impact |
|-----|---------------|----------|------------|
| DNS | Ежемесячно | DNS failover test | None (dry-run) |
| DB | Ежеквартально | DR DB promotion test | Read replica only |
| NATS | Ежеквартально | NATS mirror handover | None (mirror test) |
| Full | Раз в полгода | Полный failover simulation | Read-only mode |

### 6.2 Запуск drill

```bash
# DNS drill
curl -X POST https://api.cctv.example.com/api/v1/dr/drill \
  -H "Content-Type: application/json" \
  -d '{"type": "dns"}'

# DB drill
curl -X POST https://api.cctv.example.com/api/v1/dr/drill \
  -H "Content-Type: application/json" \
  -d '{"type": "db"}'

# Full drill
curl -X POST https://api.cctv.example.com/api/v1/dr/drill \
  -H "Content-Type: application/json" \
  -d '{"type": "full"}'

# Проверить активный drill
curl -X GET https://api.cctv.example.com/api/v1/dr/drill/active
```

### 6.3 Чек-лист drill

#### DNS Drill
- [ ] Проверка DNS A/AAAA записей DR региона
- [ ] Проверка DNS propagation
- [ ] Dry-run DNS failover скрипта
- [ ] Измерение времени propagation

#### DB Drill
- [ ] Проверка подключения к DR PostgreSQL
- [ ] Измерение replication lag (RPO)
- [ ] Симуляция promotion DR → primary
- [ ] Проверка read-only режима

#### Full Drill
- [ ] DNS failover simulation
- [ ] DB promotion simulation
- [ ] NATS stream handover simulation
- [ ] RPO verification (max 5 min)
- [ ] Post-drill health check

### 6.4 Критерии прохождения

- [ ] Все check items passed
- [ ] RTO < 15 минут
- [ ] RPO < 5 минут
- [ ] Health после drill = "healthy"
- [ ] Нет потери данных

---

## 7. RTO/RPO Dashboard

### 7.1 Метрики

```bash
# Получить метрики через API
curl -X GET https://api.cctv.example.com/api/v1/dr/health | jq '.metrics'

{
  "uptime_seconds": 604800,
  "check_count": 20160,
  "failure_rate": 0.001,
  "rto_compliance": true,
  "rpo_compliance": true
}
```

### 7.2 Целевые показатели

| Показатель | Target | Warning | Critical |
|-----------|--------|---------|----------|
| RTO | < 15 min | > 10 min | > 15 min |
| RPO | < 5 min | > 3 min | > 5 min |
| Uptime | > 99.9% | < 99.9% | < 99.5% |
| Failure rate | < 0.1% | > 0.1% | > 1% |

---

## 8. Compliance Checklist

### ISO 27001

- [ ] **A.17.1.1** — DR policy defined and documented
- [ ] **A.17.1.2** — DR procedures implemented and tested
- [ ] **A.17.1.3** — Quarterly drills conducted
- [ ] **A.12.4.1** — All failover events logged with trace ID
- [ ] **A.12.6.1** — Capacity monitoring in DR

### IEC 62443-3-3

- [ ] **SR 7.1** — Continuous health monitoring (30s interval)
- [ ] **SR 7.2** — Periodic DR testing (quarterly drills)
- [ ] **SR 7.3** — Failover mechanism with admin approval

### Приказ ОАЦ №66

- [ ] **п. 7.18.1** — Мониторинг конечных узлов (health checks)
- [ ] **п. 7.18.2** — Резервирование каналов связи (DNS failover)
- [ ] **п. 7.18.5** — Периодическое тестирование (quarterly drills)

### GDPR

- [ ] **Art. 32** — Security of processing (DR for personal data)
- [ ] **Art. 35** — DPIA includes DR scenarios
- [ ] **Art. 44-49** — Data transfer compliance during failover

---

## Приложение A: Быстрые команды

```bash
# ── Health ────────────────────────────────────────
curl -X GET https://api.cctv.example.com/api/v1/dr/health

# ── Failover ──────────────────────────────────────
curl -X POST https://api.cctv.example.com/api/v1/dr/failover \
  -H "Content-Type: application/json" \
  -d '{"reason": "emergency"}'

# ── Подтверждение failover ────────────────────────
curl -X POST https://api.cctv.example.com/api/v1/dr/failover/{id}/approve

# ── Отклонение failover ───────────────────────────
curl -X POST https://api.cctv.example.com/api/v1/dr/failover/{id}/reject \
  -d '{"reason": "false_positive"}'

# ── История ───────────────────────────────────────
curl -X GET https://api.cctv.example.com/api/v1/dr/history

# ── Drills ────────────────────────────────────────
curl -X POST https://api.cctv.example.com/api/v1/dr/drill \
  -d '{"type": "dns"}'

curl -X POST https://api.cctv.example.com/api/v1/dr/drill \
  -d '{"type": "full"}'

curl -X GET https://api.cctv.example.com/api/v1/dr/drill/active
```

## Приложение B: Переменные окружения

| Переменная | Описание | Обязательно |
|-----------|----------|-------------|
| `DNS_PROVIDER` | DNS провайдер (cloudflare/route53/generic) | Да |
| `CLOUDFLARE_ZONE_ID` | Cloudflare Zone ID | Для Cloudflare |
| `CLOUDFLARE_API_TOKEN` | Cloudflare API Token | Для Cloudflare |
| `ROUTE53_ZONE_ID` | Route53 Hosted Zone ID | Для Route53 |
| `AWS_PROFILE` | AWS CLI Profile | Для Route53 |
| `DNS_API_URL` | Generic DNS API URL | Для generic |
| `DNS_API_KEY` | Generic DNS API Key | Для generic |
