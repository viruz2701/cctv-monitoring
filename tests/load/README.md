# Load Testing (k6) — CCTV Health Monitor

## Обзор

Нагрузочное тестирование с использованием [k6](https://k6.io/) для CCTV Health Monitor (КИИ РБ, класс KII-2).

**Цель:** Проверка производительности API при 1000 concurrent пользователей, WebSocket — 1000 concurrent соединений.

**Compliance:**
- ISO 27001 A.12.6 — Capacity Management
- IEC 62443-3-3 SR 7.8 — Security Function Verification
- OWASP ASVS L3 V11 — Business Logic (Rate Limiting)
- OWASP ASVS L3 V13 — API & Web Service (WebSocket)

## Установка k6

```bash
# Linux (Debian/Ubuntu)
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# macOS
brew install k6

# Docker
docker run -i grafana/k6 run - <script.js
```

## Сценарии

### Smoke Test (CI/CD)

| Файл | Описание | VUs | Длительность |
|------|----------|-----|--------------|
| [`smoke-test.js`](./smoke-test.js) | Быстрая проверка всех критических endpoint'ов | 2 | 30s |

**Покрываемые endpoint'ы:**
- `GET /health/live`, `GET /health/ready`
- `GET /api/v1/users/me`
- `GET /api/v1/devices`, `GET /api/v1/devices/{id}`
- `GET /api/v1/work-orders`
- `GET /api/v1/sites`
- `GET /api/v1/alarms`
- `GET /api/v1/compliance/summary`
- `GET /api/v1/audit/log`
- `GET /api/v1/notifications`
- `GET /api/v1/tickets`

### Load Test (1000 concurrent)

| Файл | Описание | Endpoint | Длительность |
|------|----------|----------|--------------|
| [`devices.scenario.js`](./devices.scenario.js) | GET /devices (пагинация, фильтры, детали, статус) | `GET /api/v1/devices` | 60s ramp-up, 180s stay, 30s ramp-down |
| [`work-orders.scenario.js`](./work-orders.scenario.js) | CRUD /work-orders (создание, статус-транзишны, список) | `POST /api/v1/work-orders` | 60s ramp-up, 180s stay, 30s ramp-down |
| [`websocket.scenario.js`](./websocket.scenario.js) | WebSocket events (подписки, heartbeat, reconnect) | `ws://host/ws/alarms` | 60s ramp-up, 120s stay, 30s ramp-down |

## Пороги производительности (Thresholds)

### Глобальные (P1-QA.7)

| Метрика | Threshold | Описание |
|---------|-----------|----------|
| `http_req_duration` | `p(95) < 500ms` | 95% запросов быстрее 500ms |
| `http_req_duration` | `p(99) < 1000ms` | 99% запросов быстрее 1s |
| `http_req_failed` | `rate < 0.01` | Ошибок менее 1% |
| `checks` | `rate > 0.99` | Успешных проверок более 99% |

### Per-endpoint (Devices)

| Endpoint | Thresholds |
|----------|------------|
| `devices.list` | `p(95) < 500ms`, `p(99) < 800ms` |
| `devices.detail` | `p(95) < 300ms`, `p(99) < 600ms` |
| `devices.status` | `p(95) < 200ms`, `p(99) < 400ms` |
| `devices.filter` | `p(95) < 500ms`, `p(99) < 1000ms` |
| `devices.search` | `p(95) < 500ms`, `p(99) < 1000ms` |

### Per-endpoint (Work Orders)

| Endpoint | Thresholds |
|----------|------------|
| `work-orders.create` | `p(95) < 500ms`, `p(99) < 1000ms` |
| `work-orders.assign` | `p(95) < 400ms`, `p(99) < 800ms` |
| `work-orders.start` | `p(95) < 300ms`, `p(99) < 600ms` |
| `work-orders.complete` | `p(95) < 300ms`, `p(99) < 600ms` |
| `work-orders.cancel` | `p(95) < 300ms`, `p(99) < 600ms` |
| `work-orders.list` | `p(95) < 500ms`, `p(99) < 1000ms` |
| `work-orders.get` | `p(95) < 300ms`, `p(99) < 600ms` |

### Per-endpoint (WebSocket)

| Endpoint | Thresholds |
|----------|------------|
| `websocket.events` | `ws_connecting p(95) < 1000ms` |
| `websocket.reconnect` | `ws_connecting p(95) < 2000ms` |
| WebSocket session | `ws_session_duration p(95) > 10000ms` |
| Heartbeat | `latency < 500ms` |

## Запуск

### Smoke test (CI/CD)

```bash
k6 run tests/load/smoke-test.js
```

### Полный load test

```bash
# Devices
k6 run tests/load/devices.scenario.js

# Work Orders
k6 run tests/load/work-orders.scenario.js

# WebSocket
k6 run tests/load/websocket.scenario.js
```

### С переменными окружения

```bash
k6 run \
  -e BASE_URL=https://staging.example.com \
  -e AUTH_TOKEN=eyJhbGciOi... \
  -e WS_URL=wss://staging.example.com/ws \
  tests/load/devices.scenario.js
```

### Параллельный запуск всех сценариев

```bash
# Установка параллельного раннера
npm install -g @grafana/k6-parallel

# Запуск всех сценариев
k6-parallel \
  tests/load/devices.scenario.js \
  tests/load/work-orders.scenario.js \
  tests/load/websocket.scenario.js
```

### Вывод результатов в JSON

```bash
k6 run --out json=results.json tests/load/smoke-test.js

# Анализ результатов
cat results.json | jq '. | select(.type=="Point") | .metric == "http_req_duration"'
```

### Prometheus + Grafana

```bash
# Удалённый вывод в Prometheus
k6 run \
  --out output-prometheus-remote \
  tests/load/devices.scenario.js

# Loki для логов
k6 run \
  --out output-prometheus-remote \
  --out output-loki \
  tests/load/devices.scenario.js
```

## Архитектура сценариев

### devices.scenario.js

```
GET /devices (page, page_size)  ─────────────────────┐
  ├── 30%: GET /devices (status, device_type)         │ Фильтрация
  ├── Если есть данные:                               │
  │   ├── GET /devices/:id                            │ Детали (случайное устройство)
  │   └── 30%: GET /devices/:id/status                │ Статус
  └── 10%: GET /devices (search)                      │ Поиск
```

### work-orders.scenario.js

```
POST /work-orders ───────────────────────────────────┐
  ├── Успех:                                          │
  │   ├── GET /work-orders/:id                        │ Верификация
  │   └── 50%: Статус-транзишн                        │
  │       ├── POST /work-orders/:id/assign            │
  │       ├── POST /work-orders/:id/start             │
  │       └── 80%: POST /work-orders/:id/complete     │
  │           └── 20%: POST /work-orders/:id/cancel   │
  └── 20%: GET /work-orders (status, priority, ...)   │ Список с фильтрацией
```

### websocket.scenario.js

```
ws.connect /ws/alarms ───────────────────────────────┐
  ├── on open: subscribe (device.events, alerts, ...) │
  ├── on message:                                     │
  │   ├── heartbeat → check timestamp + latency       │
  │   ├── subscribed → check channels + client_id     │
  │   ├── device.event → check device_id, timestamp   │
  │   ├── alert → check device_id, severity           │
  │   └── work-order.update → check id, status        │
  ├── ping каждые 30s                                 │
  ├── re-subscribe каждые 60s                         │
  └── session: 20-60s                                 │
  └── При неудаче: реконнект (до 3 попыток)           │
```

## Метрики

### Основные метрики k6

| Метрика | Тип | Описание |
|---------|-----|----------|
| `http_req_duration` | Trend | Время выполнения HTTP-запроса |
| `http_req_failed` | Rate | Доля неудачных запросов |
| `http_req_sending` | Trend | Время отправки запроса |
| `http_req_waiting` | Trend | Время ожидания (TTFB) |
| `http_req_receiving` | Trend | Время получения ответа |
| `checks` | Rate | Процент успешных проверок |
| `vus` | Gauge | Количество активных VUs |
| `vus_max` | Gauge | Максимальное количество VUs |
| `iterations` | Counter | Количество итераций |
| `data_received` | Counter | Получено данных |
| `data_sent` | Counter | Отправлено данных |

### WebSocket метрики

| Метрика | Тип | Описание |
|---------|-----|----------|
| `ws_connecting` | Trend | Время установки WebSocket-соединения |
| `ws_session_duration` | Trend | Длительность WebSocket-сессии |
| `ws_msgs_received` | Counter | Количество полученных сообщений |
| `ws_msgs_sent` | Counter | Количество отправленных сообщений |

## Анализ результатов

### Быстрый анализ через jq

```bash
# Средняя длительность запросов
cat results.json | jq 'select(.type=="Point" and .metric=="http_req_duration") | .data.value' | awk '{sum+=$1; n++} END {print sum/n}'

# 95-й перцентиль
cat results.json | jq 'select(.type=="Point" and .metric=="http_req_duration") | .data.value' | sort -n | awk '{a[NR]=$1} END {print a[int(NR*0.95)]}'

# Количество ошибок
cat results.json | jq 'select(.type=="Point" and .metric=="http_req_failed" and .data.value > 0)'
```

### HTML отчёт

```bash
k6 run --out html=report.html tests/load/devices.scenario.js
```

## Требования (P1-QA.5)

- **95th percentile** latency < 500ms для всех эндпоинтов
- **99th percentile** latency < 1000ms для всех эндпоинтов
- **Error rate** < 1% для всех запросов
- **Checks rate** > 99% успешных проверок
- **WebSocket connecting** p(95) < 1s
- **WebSocket reconnect** p(95) < 2s
- **1000 concurrent** users для REST API
- **1000 concurrent** WebSocket соединений
- **Ramp up** за 60s (плавный старт)
- **Think time** 1-5s (имитация реального пользователя)

## Troubleshooting

### WebSocket таймауты

Если WebSocket соединения не устанавливаются:

```bash
# Увеличить таймаут
k6 run -e WS_URL=ws://host:8080/ws --compatibility-mode=extended tests/load/websocket.scenario.js

# Проверить WSS (если требуется TLS)
k6 run -e WS_URL=wss://staging.example.com/ws tests/load/websocket.scenario.js --insecure-skip-tls-verify
```

### Rate Limiting (OWASP ASVS V11)

Если нагрузочный тест упирается в rate limiter:

```bash
# Увеличить лимиты для тестового пользователя
# Или использовать service account без rate limiting
k6 run -e AUTH_TOKEN=test_admin_token tests/load/devices.scenario.js
```

### Недостаточно памяти

```bash
# Ограничить использование памяти
k6 run --max-execution-duration=5m tests/load/devices.scenario.js
```

## Интеграция с CI/CD

### GitHub Actions

```yaml
- name: k6 Smoke Test
  run: |
    docker run -i grafana/k6 run \
      -e BASE_URL=${{ env.BASE_URL }} \
      -e AUTH_TOKEN=${{ secrets.AUTH_TOKEN }} \
      - < tests/load/smoke-test.js
```

### GitLab CI

```yaml
k6-load-test:
  image: grafana/k6:latest
  script:
    - k6 run tests/load/devices.scenario.js
  variables:
    BASE_URL: "https://staging.example.com"
```

## Compliance Notes

- **ISO 27001 A.12.6**: Capacity Management — проверка, что система выдерживает нагрузку
- **IEC 62443-3-3 SR 7.8**: Security Function Verification — каждый VU независим, уникальная идентификация
- **OWASP ASVS L3 V11**: Business Logic — rate limiting не ломает бизнес-логику
- **OWASP ASVS L3 V13**: API & Web Service — WebSocket валидация сообщений и авторизация
- **СТБ 34.101.27**: Защита информации — проверка целостности и доступности
- **Приказ ОАЦ №66 п.7.18**: Контроль нагрузки на конечные узлы
