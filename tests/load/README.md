# Load Testing (k6) — CCTV Health Monitor

## Обзор

Нагрузочное тестирование с использованием [k6](https://k6.io/).

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

| Файл | Описание | Endpoint | RPS |
|------|----------|----------|-----|
| [`devices.scenario.js`](./devices.scenario.js) | GET /devices | `GET /api/v1/devices` | 1000 concurrent |
| [`work-orders.scenario.js`](./work-orders.scenario.js) | POST /work-orders | `POST /api/v1/work-orders` | 1000 concurrent |
| [`websocket.scenario.js`](./websocket.scenario.js) | WebSocket events | `ws://host/ws` | 1000 connections |
| [`smoke-test.js`](./smoke-test.js) | Smoke test (все endpoints) | CI/CD validation | 2 VUs, 30s |

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

### Вывод результатов в JSON

```bash
k6 run --out json=results.json tests/load/smoke-test.js
```

### Prometheus + Grafana

```bash
k6 run --out output-prometheus-remote tests/load/devices.scenario.js
```

## Требования

- **95th percentile** latency < 500ms для всех эндпоинтов
- **Error rate** < 1% для всех запросов
- **1000 concurrent** users для REST API
- **1000 concurrent** WebSocket соединений

## Метрики

Важные метрики k6:

| Метрика | Описание | Threshold |
|---------|----------|-----------|
| `http_req_duration` | Время выполнения запроса | p(95) < 500ms |
| `http_req_failed` | Доля неудачных запросов | rate < 0.01 |
| `checks` | Процент успешных проверок | rate > 0.99 |
| `ws_connecting` | Время установки WebSocket | p(95) < 1000ms |
| `vus` | Количество активных VUs | — |

## Compliance

- **ISO 27001 A.12.6** — Capacity Management
- **IEC 62443-3-3 SR 7.8** — Security Function Verification
- **OWASP ASVS L3 V11** — Business Logic (Rate Limiting)
