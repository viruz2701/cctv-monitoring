# CCTV Health Monitor — Development Guide

## Overview

CCTV Health Monitor — система мониторинга CCTV оборудования с поддержкой реального времени, P2P-шлюзов, интеграцией с CMMS, офлайн-режимом и криптографией по стандартам Республики Беларусь (СТБ 34.101.30, СТБ 34.101.27).

Стек: Go 1.25 (backend) + React 19 (frontend) + React Native / Expo 52 (mobile) + PostgreSQL / TimescaleDB + NATS.

---

## Prerequisites

| Компонент         | Версия       | Примечание                     |
|-------------------|--------------|---------------------------------|
| Go                | 1.25+        | `go version`                   |
| Node.js           | 22+          | `node --version`               |
| npm               | 10+          | `npm --version`                |
| PostgreSQL        | 16+          | С TimescaleDB extension        |
| NATS Server       | 2.10+        | `nats-server -v`               |
| Docker            | любая        | Опционально (тестконтейнеры)   |

---

## Environment Variables

Файл: [`backend/.env.example`](backend/.env.example)

Все переменные окружения дублируются в конфигурационном файле [`backend/config.yaml`](backend/config.yaml). Приоритет: ENV > config.yaml > default.

### Database

| Переменная      | Описание                                         | По умолчанию         |
|-----------------|--------------------------------------------------|----------------------|
| `DB_HOST`       | Хост PostgreSQL                                  | `localhost`          |
| `DB_USER`       | Пользователь БД                                  | `gb_user`            |
| `DB_PASSWORD`   | Пароль пользователя БД                           | `gb_password`        |
| `DB_NAME`       | Имя базы данных                                  | `gb_telemetry`       |
| `DATABASE_URL`  | Полная строка подключения (альтернатива)         | —                    |

### NATS

Переменные окружения для NATS задаются в [`backend/config.yaml`](backend/config.yaml) (секция `nats`):

| Поле конфига       | Переменная         | Описание                     | По умолчанию               |
|--------------------|--------------------|------------------------------|---------------------------|
| `nats_url`         | `GB_NATS_URL`      | Адрес NATS сервера           | `nats://localhost:4222`   |
| `nats_required`    | `GB_NATS_REQUIRED` | NATS обязателен для старта   | `true`                    |
| `use_nats_kv`      | `GB_USE_NATS_KV`   | Использовать JetStream KV    | `true`                    |
| `nats_creds`       | `GB_NATS_CREDS`    | Путь к credentials файлу     | —                         |
| `nats_tls`         | `GB_NATS_TLS`      | TLS для NATS                 | `false`                   |

> **Для разработки:** установите `GB_NATS_REQUIRED=false` и `GB_USE_NATS_KV=false` если NATS не запущен.

### Auth / JWT

| Переменная              | Описание                                                              | По умолчанию                          |
|-------------------------|-----------------------------------------------------------------------|---------------------------------------|
| `JWT_SECRET`            | (Legacy) Секрет для refresh token хеширования                        | `change-me-to-a-random-64-char-string` |
| `BIGN_PRIVATE_KEY`      | PEM-encoded ECDSA P-256 приватный ключ для подписи JWT (P3-SEC.2)    | Автогенерация (dev)                   |
| `BIGN_PRIVATE_KEY_FILE` | Путь к PEM-файлу с ECDSA P-256 ключом (альтернатива `BIGN_PRIVATE_KEY`) | —                                   |

> **P3-SEC.2:** JWT подписываются ECDSA P-256 (bign-curve256v1 / ES256) вместо HMAC-SHA256.
> В production **обязательно** укажите `BIGN_PRIVATE_KEY` или `BIGN_PRIVATE_KEY_FILE`.
> В development режиме ключ генерируется автоматически при старте.

### P2P Gateway

| Переменная             | Описание                | По умолчанию                       |
|------------------------|-------------------------|------------------------------------|
| `GB_P2P_GATEWAY_URL`   | URL P2P Gateway сервиса | `http://localhost:8082`            |
| `GB_P2P_API_KEY`       | API ключ для P2P Gateway| `change-me`                        |

### CMMS Integration

| Переменная          | Описание                                    | По умолчанию |
|---------------------|---------------------------------------------|--------------|
| `GB_CMMS_ADAPTER`   | Тип адаптера: `internal` (по умолчанию) или `atlas` | `internal` |
| `GB_ATLAS_URL`      | URL Atlas CMMS (если выбран `atlas`)        | —            |
| `GB_ATLAS_API_KEY`  | API ключ Atlas CMMS                         | —            |

### Telegram Bot

| Переменная             | Описание                          | По умолчанию |
|------------------------|-----------------------------------|--------------|
| `GB_TELEGRAM_ENABLED`  | Включить Telegram-бота            | `false`      |
| `GB_TELEGRAM_TOKEN`    | Токен Telegram-бота               | —            |

### Storage / Encryption

| Переменная                  | Описание                                         | По умолчанию |
|-----------------------------|--------------------------------------------------|--------------|
| `PUSH_TOKEN_ENCRYPTION_KEY` | AES-256 ключ для шифрования push-токенов (64 hex символа) | —     |

### Audit

| Переменная           | Описание                                                   | По умолчанию |
|----------------------|------------------------------------------------------------|--------------|
| `GB_AUDIT_HMAC_KEY`  | Ключ HMAC-подписи audit_log (ISO 27001 A.12.4, СТБ 34.101.30). Минимум 32 байта | `change-me-to-a-random-string-at-least-32-bytes` |

### Admin Password (первый запуск)

| Переменная          | Описание                                      | По умолчанию |
|---------------------|-----------------------------------------------|--------------|
| `GB_ADMIN_PASSWORD` | Пароль администратора при seed БД. Если не задан — генерируется случайный 32-символьный | — |

### AI / Gatekeeper

| Переменная          | Описание                    | По умолчанию                     |
|---------------------|-----------------------------|----------------------------------|
| `DEEPSEEK_API_KEY`  | API ключ DeepSeek           | —                                |
| `DEEPSEEK_API_URL`  | URL DeepSeek API            | `https://api.deepseek.com/v1`    |

### Vision Guard

| Переменная                | Описание                                           | По умолчанию |
|---------------------------|----------------------------------------------------|--------------|
| `VISION_GUARD_STRICT`     | Строгий режим Vision Guard (блокировка при сомнениях) | `false`      |

### Telegram Token Provider

| Переменная                     | Описание                                              | По умолчанию |
|--------------------------------|-------------------------------------------------------|--------------|
| `GB_TELEGRAM_TOKEN_PROVIDER`   | Провайдер Telegram токена: `env` или `vault`          | `env`        |

### WireGuard

| Переменная            | Описание                                | По умолчанию |
|-----------------------|-----------------------------------------|--------------|
| `GB_WG_PSK_ENABLED`   | Включить WireGuard PSK для P2P шлюзов   | `false`      |

### JWT Rotation

| Переменная                  | Описание                                              | По умолчанию |
|-----------------------------|-------------------------------------------------------|--------------|
| `GB_JWT_ROTATION_ENABLED`   | Автоматическая ротация refresh токенов (P3-SEC.5)    | `false`      |

### ML Prediction Queue (NATS WorkQueue)

| Переменная                      | Описание                                                  | По умолчанию |
|---------------------------------|-----------------------------------------------------------|--------------|
| `GB_PREDICTION_QUEUE_ENABLED`   | Включить NATS WorkQueue для ML предсказаний               | `false`      |

### CSP Reporting

| Переменная             | Описание                                               | По умолчанию |
|------------------------|--------------------------------------------------------|--------------|
| `GB_CSP_REPORT_URI`    | Endpoint для CSP violation reports (Content Security Policy) | —            |

---

## Quick Start

### 1. Clone & Install

```bash
git clone <repo>
cd cctv-monitoring
```

### 2. Database Setup

```bash
# Создать базу данных
createdb gb_telemetry

# Запустить миграции (автоматически при старте backend)
cd backend && go run .
```

> **Dirty state recovery:** Если миграции в dirty state (например, после изменения файла миграции):
> ```bash
> FORCE_MIGRATION_VERSION=auto go run .
> ```
> Это автоматически определит последнюю версию (36) и восстановит состояние.

> **Требование:** PostgreSQL должен быть установлен с расширением TimescaleDB. Убедитесь, что `psql` доступен в PATH.

### 3. Backend

```bash
cd backend

# Настроить окружение
cp .env.example .env   # отредактируйте под своё окружение

# Установить зависимости
go mod download

# Запустить API сервер
go run ./cmd/api
```

Сервер будет доступен на `http://localhost:8080` (по умолчанию). Hot-reload через Air: `air`.

**Swagger UI / OpenAPI:**
- OpenAPI spec (JSON): [`http://localhost:8080/api/v1/openapi.json`](http://localhost:8080/api/v1/openapi.json)
- Swagger UI (HTML): [`http://localhost:8080/api/v1/docs`](http://localhost:8080/api/v1/docs)

### 4. Frontend

```bash
cd frontend

# Установить зависимости
npm install

# Настроить окружение (опционально)
cp .env.example .env.local

# Запустить dev-сервер
npm run dev
```

Dev-сервер будет доступен на `http://localhost:5173`.

### 5. Mobile (React Native + Expo)

```bash
cd mobile

# Установить зависимости
npm install

# Запустить Expo
npx expo start
```

После запуска отсканируйте QR-код в приложении Expo Go или нажмите `a` для Android / `i` для iOS симулятора.

### 6. P2P Gateway (опционально)

```bash
cd p2p-gateway
go build -o p2p-gateway .
./p2p-gateway
```

P2P Gateway будет доступен на `http://localhost:8082`.

---

## Project Structure

```
cctv-monitoring/
├── backend/                  # Go API сервер
│   ├── cmd/
│   │   ├── api/              # Точка входа API сервера
│   │   └── migrate/          # Мигратор БД
│   ├── internal/
│   │   ├── api/              # HTTP хендлеры, middleware, роутинг, OpenAPI
│   │   ├── auth/             # Аутентификация: JWT (ES256), WebAuthn, 2FA, RBAC
│   │   ├── cmms/             # CMMS адаптеры (Internal, Atlas)
│   │   ├── compliance/       # Compliance профили (BY, EU, INTL, RU, CN)
│   │   ├── config/           # Конфигурация (Viper)
│   │   ├── crypto/           # Криптопровайдеры (belt, bign, bash stubs)
│   │   │   └── providers/    # Regional providers: AES, Belt, GOST, SM, bign ECDSA
│   │   ├── db/               # Слой базы данных (pgx/v5, миграции)
│   │   ├── events/           # NATS события, Schema Registry
│   │   ├── gatekeeper/       # Gatekeeper token (верификация перед CompleteWO)
│   │   ├── notifications/    # Уведомления (Telegram, SMS, Email)
│   │   ├── rca/              # Root Cause Analysis граф
│   │   ├── sla/              # SLA engine, escalation
│   │   ├── stb/              # СТБ CryptoProvider interface
│   │   ├── tenant/           # Multi-tenant, RLS, compliance per tenant
│   │   └── webhook/          # Webhook delivery worker
│   ├── migrations/           # SQL миграции (golang-migrate)
│   ├── config.yaml           # Конфигурация приложения
│   └── Dockerfile
│
├── frontend/                 # React 19 + Vite + TailwindCSS v4
│   ├── src/
│   │   ├── components/       # UI компоненты (ui/, dashboard/, layout/, molecules/)
│   │   ├── hooks/            # React hooks
│   │   ├── pages/            # Страницы приложения
│   │   ├── services/         # API клиенты
│   │   ├── store/            # Управление состоянием (Zustand)
│   │   └── utils/            # Утилиты
│   └── package.json
│
├── mobile/                   # React Native + Expo 52
│   ├── src/
│   │   ├── components/       # UI компоненты
│   │   ├── hooks/            # React hooks (офлайн, геолокация, синхронизация)
│   │   ├── services/         # Сервисы (офлайн-хранилище, синхронизация, кэш тайлов)
│   │   └── store/            # Состояние (Zustand)
│   └── package.json
│
├── p2p-gateway/              # P2P Gateway сервис (Go)
│
├── docs/                     # Документация
│   ├── adr/                  # Architecture Decision Records
│   ├── compliance/           # Compliance отчёты
│   ├── iso27001/             # ISO 27001 документация
│   └── accessibility/        # Accessibility audit
│
├── tests/                    # Интеграционные/E2E тесты
│   ├── load/                 # k6 нагрузочные тесты
│   └── chaos/                # Chaos engineering тесты
│
└── plans/                    # Планы разработки
```

---

## Testing

### Backend

```bash
cd backend

# Все тесты
go test ./... -v

# С coverage отчётом
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# Интеграционные тесты (требуют testcontainers-go)
go test ./... -tags=integration -v

# Benchmark тесты
go test ./... -bench=. -benchmem

# Compliance тесты (СТБ криптография)
go test ./internal/crypto/... -v
```

### Frontend

```bash
cd frontend

# Unit тесты (vitest)
npm test

# С coverage (требование: ≥ 85%)
npm run test:coverage

# Storybook (интерактивная документация компонентов)
npm run storybook

# TypeScript проверка
npx tsc --noEmit
```

### Coverage Targets

| Уровень | Компонент   | Минимум | Текущий  | Инструмент       |
|---------|-------------|---------|----------|------------------|
| Unit    | Backend     | 80%     | 78%      | `go test -cover` |
| Unit    | Frontend    | 85%     | 82%      | Vitest           |
| Integration | Backend | 70%     | 65%      | testcontainers-go|
| E2E     | Frontend    | 60%     | 55%      | Playwright       |
| Visual  | Frontend    | 90%     | 87%      | Playwright + snap|

Команда `npm run test:coverage` запускает Vitest с флагом `--coverage` и проверяет порог 85%.
При падении ниже порога — CI падает с ошибкой. Конфигурация в [`frontend/vitest.config.ts`](frontend/vitest.config.ts).

### Unit-тесты (Go Table-Driven)

Все Go-тесты следуют паттерну **table-driven tests**:

```go
func TestValidateToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        secret  string
        wantErr bool
    }{
        {name: "valid token", token: validToken, secret: "secret", wantErr: false},
        {name: "expired token", token: expiredToken, secret: "secret", wantErr: true},
        {name: "wrong signature", token: wrongSig, secret: "wrong", wantErr: true},
        {name: "empty token", token: "", secret: "secret", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := ValidateToken(tt.token, tt.secret)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateToken() error = %v, wantErr = %v", err, tt.wantErr)
            }
        })
    }
}
```

Обязательные кейсы в каждом table-driven тесте:
- **Happy path** — успешный сценарий
- **Error path** — ожидаемая ошибка
- **Boundary** — граничные значения (пустая строка, nil, max int)
- **Security** — попытка инъекции, подделки, подбора

### Integration-тесты (testcontainers-go)

Интеграционные тесты используют `testcontainers-go` для поднятия PostgreSQL + NATS в Docker:

```bash
cd backend

# Интеграционные тесты (требуют Docker)
go test ./... -tags=integration -v

# С coverage
go test ./... -tags=integration -coverprofile=coverage.out
```

Пример структуры:

```go
func TestDeviceRepository(t *testing.T) {
    ctx := context.Background()
    pgContainer, err := postgres.RunContainer(ctx,
        postgres.WithDatabase("gb_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
    )
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)

    dbURL, _ := pgContainer.ConnectionString(ctx)
    pool, _ := pgxpool.New(ctx, dbURL)
    defer pool.Close()

    repo := NewDeviceRepository(pool)
    // ... table-driven тесты
}
```

### E2E (Playwright)

```bash
# Из корня проекта (E2E тесты всего стека)
cd tests && npx playwright test

# Из frontend (только frontend E2E)
cd frontend && npx playwright test

# С UI режимом
npx playwright test --ui

# Accessibility тесты (axe-core)
npx playwright test --grep "a11y"

# Visual regression тесты (скриншоты)
npx playwright test --grep "visual"
```

E2E тесты расположены в двух местах:

| Директория       | Назначение                                  | Конфиг                                    |
|------------------|---------------------------------------------|-------------------------------------------|
| `frontend/e2e/`  | Frontend-only E2E (мокированный API)        | [`frontend/playwright.config.ts`](frontend/playwright.config.ts) |
| `tests/`         | Full-stack E2E (реальный backend + БД)      | [`tests/playwright.config.ts`](tests/playwright.config.ts) |

### Visual Regression (Скриншоты)

Playwright настроен на автоматическое создание скриншотов при каждом E2E прогоне:

```bash
# Обновить baseline-скриншоты
npx playwright test --update-snapshots

# Сравнить с baseline
npx playwright test --grep "visual"
```

Baseline-скриншоты хранятся в `frontend/e2e/snapshots/` и `tests/snapshots/`.
При изменении UI — запустите `--update-snapshots` и проверьте diff в отчёте Playwright.

**Порог чувствительности:** 0.1% пикселей (конфигурация в `playwright.config.ts`).
Отчёт visual regression: [`playwright-report-visual/`](playwright-report-visual/index.html)

### Accessibility (a11y)

Каждый компонент проходит проверку **axe-core**:

```bash
# Запуск всех a11y тестов
npx playwright test --grep "a11y"

# Проверка контрастности (WCAG AA/AAA)
npm run check-contrast
```

Результаты a11y-тестов:
- Лог: `playwright-report-a11y/`
- Violations группируются по правилам WCAG 2.1 AA
- Блокирующие violations (critical/serious) → падение CI

**WCAG Contrast Check** (`npm run check-contrast`) проверяет все CSS-переменные и таблицы цветов на соответствие WCAG 2.1 AA (4.5:1 для текста, 3:1 для крупных элементов). Конфигурация в [`lighthouserc.js`](lighthouserc.js).

### Load Testing (k6)

```bash
cd tests/load

# Установить k6 (Ubuntu)
sudo apt install k6

# Smoke test
k6 run smoke-test.js

# Нагрузочный тест устройств
k6 run devices.scenario.js

# Нагрузочный тест WebSocket
k6 run websocket.scenario.js
```

### Chaos Engineering

```bash
cd tests/chaos

# Dry-run режим (без toxiproxy)
node runner.js --dry-run

# Полный прогон с toxiproxy
node runner.js
```

---

## Code Quality

### Backend

```bash
cd backend

# Lint (golangci-lint)
golangci-lint run ./...

# Форматирование
gofmt -w .

# Проверка на уязвимости
go vet ./...

# Проверка на наличие crypto/aes (запрещено для BY profile)
grep -r "crypto/aes" --include="*.go" internal/crypto/ || echo "OK: no crypto/aes in crypto layer"
```

### Frontend

```bash
cd frontend

# TypeScript проверка
npx tsc --noEmit

# ESLint
npm run lint

# Форматирование (Prettier)
npx prettier --check .
```

---

## CI/CD Pipeline

CI/CD пайплайн настроен через **GitHub Actions**. Файлы в [`.github/`](.github/).

### Workflow: CI

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Backend lint
        run: cd backend && golangci-lint run ./...
      - name: Frontend lint
        run: cd frontend && npm run lint
      - name: TypeScript check
        run: cd frontend && npx tsc --noEmit

  test:
    needs: lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Backend unit tests
        run: cd backend && go test ./... -v -coverprofile=coverage.out
      - name: Frontend unit tests + coverage
        run: cd frontend && npm run test:coverage
      - name: Security scan (gosec)
        run: cd backend && gosec ./...

  sbom:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Generate SBOM (Syft)
        run: syft dir:./backend -o spdx-json > sbom-backend.json
      - name: Upload SBOM
        uses: actions/upload-artifact@v4
        with:
          name: sbom
          path: sbom-*.json

  contrast:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: WCAG Contrast Check
        run: cd frontend && npm run check-contrast

  storybook:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Storybook
        run: cd frontend && npm run storybook:build
      - name: Deploy to Chromatic
        uses: chromaui/action@v1
        with:
          projectToken: ${{ secrets.CHROMATIC_PROJECT_TOKEN }}
```

### Workflow: Deploy

```yaml
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker images
        run: |
          docker build -t ghcr.io/org/cctv-backend:latest ./backend
          docker build -t ghcr.io/org/cctv-frontend:latest ./frontend
      - name: Push to registry
        run: |
          docker push ghcr.io/org/cctv-backend:latest
          docker push ghcr.io/org/cctv-frontend:latest

  e2e:
    needs: build
    runs-on: ubuntu-latest
    services:
      postgres:
        image: timescale/timescaledb:latest-pg16
        env:
          POSTGRES_DB: gb_test
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
        ports:
          - 5432:5432
      nats:
        image: nats:latest
        ports:
          - 4222:4222
    steps:
      - uses: actions/checkout@v4
      - name: E2E tests
        run: cd tests && npx playwright test
```

### Этапы CI/CD

| Этап       | Команда / Действие                          | Проверка                    |
|------------|---------------------------------------------|-----------------------------|
| Lint       | `golangci-lint`, `eslint`, `tsc --noEmit`   | Синтаксис, типы, стиль      |
| Test       | `go test`, `npm run test:coverage`          | Unit, coverage ≥ 85%        |
| Security   | `gosec`, `npm audit`                        | Уязвимости, SAST            |
| SBOM       | `syft` (SPDX JSON)                          | Software Bill of Materials  |
| Contrast   | `npm run check-contrast`                    | WCAG 2.1 AA compliance      |
| Storybook  | `chromaui/action` → Chromatic               | Visual review               |
| Build      | `docker build` → `docker push`              | Container images            |
| E2E        | `npx playwright test`                       | Full-stack E2E              |

### Локальный прогон CI

```bash
# Запустить все проверки локально (имитация CI)
cd backend && golangci-lint run ./... && go test ./... -v
cd frontend && npm run lint && npm run test:coverage && npx tsc --noEmit
cd tests && npx playwright test
```

---

## Security

Система реализует **Defense-in-Depth** по зонам безопасности IEC 62443.

### Content Security Policy (CSP)

CSP заголовки настраиваются в [`backend/config.yaml`](backend/config.yaml):

```yaml
security:
  csp:
    default-src: "'self'"
    script-src: "'self' 'strict-dynamic'"
    style-src: "'self' 'unsafe-inline'"
    img-src: "'self' data: blob:"
    connect-src: "'self' ws: wss:"
    report-uri: ${GB_CSP_REPORT_URI}
    report-to: csp-endpoint
```

**Запрещено:** `'unsafe-inline'` в `script-src` для production.
Нарушения CSP отправляются на `GB_CSP_REPORT_URI` для мониторинга.

### CORS

```yaml
# config.yaml
cors:
  allowed_origins:
    - http://localhost:5173       # dev frontend
    - https://app.cctv-monitor.io # production
  allowed_methods: [GET, POST, PUT, DELETE, PATCH]
  allowed_headers: [Authorization, Content-Type, X-Trace-ID]
  max_age: 300
```

### Rate Limiting

Rate limiting реализован на уровне API Gateway (Zone 2):

| Эндпоинт            | Лимит         | Window | Последствия         |
|---------------------|---------------|--------|---------------------|
| `POST /auth/login`  | 5 запросов    | 1 мин  | 429 + блок 5 мин    |
| `POST /auth/2fa`    | 3 запроса     | 1 мин  | 429 + блок 15 мин   |
| `GET /api/v1/*`     | 100 запросов  | 1 мин  | 429 + Retry-After   |
| `POST /api/v1/*`    | 30 запросов   | 1 мин  | 429 + Retry-After   |

### JWT Rotation (P3-SEC.5)

При включении `GB_JWT_ROTATION_ENABLED`:

1. Access token (ES256): 15 мин
2. Refresh token (ES256): 7 дней
3. Refresh token rotation: каждый refresh выдаёт новый refresh token
4. Refresh token reuse detection: при повторном использовании — отзыв всех токенов
5. История токенов хранится в `token_family` таблице с `prev_hash` chain

```go
type TokenFamily struct {
    ID           uuid.UUID
    UserID       uuid.UUID
    RefreshHash  string // bash-256(refresh_token + secret)
    PrevHash     string // предыдущий refresh_hash (chain)
    CreatedAt    time.Time
    RevokedAt    *time.Time
}
```

### Vision Guard

`VISION_GUARD_STRICT=true` включает строгий режим:

- ML-модель проверяет каждый кадр перед обработкой
- При confidence < 0.9 — кадр отклоняется
- При обнаружении аномалии (подмена видео, артефакты) — срабатывает alert
- Все события логируются в `audit_log` с меткой `vision_guard`

### Audit Trail (ISO 27001 A.12.4)

Каждая мутация данных логируется:

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trace_id    UUID NOT NULL,
    entity_type VARCHAR(50) NOT NULL,    -- device, work_order, user
    entity_id   UUID NOT NULL,
    action      VARCHAR(20) NOT NULL,     -- create, update, delete
    old_values  JSONB,
    new_values  JSONB,
    actor_id    UUID NOT NULL,
    hmac        BYTEA NOT NULL,          -- bash-256(trace_id || entity_id || action || old_values || new_values || GB_AUDIT_HMAC_KEY)
    prev_hash   BYTEA,                   -- bash-256(previous row hmac)
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

- Retention: 7 лет (КИИ РБ, ISO 27001 A.12.4)
- Подпись HMAC гарантирует tamper detection
- Цепочка `prev_hash` предотвращает удаление строк

### Row-Level Security (RLS)

Multi-tenant изоляция через RLS в PostgreSQL:

```sql
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON devices
    USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE POLICY admin_access ON devices
    USING (current_setting('app.current_role') = 'admin');
```

Каждый запрос устанавливает `app.current_tenant_id` через middleware.

---

## Storybook

Storybook используется для интерактивной документации UI компонентов.

Текущее покрытие: **80 stories** для всех ключевых компонентов UI, включая:
- **UI Kit**: Button, Card, Modal, Table, Tabs, Badge, Toast, Tooltip и др.
- **Layout**: Header, Sidebar, OfflineBanner, PageSuspense, RouteErrorBoundary
- **Dashboard**: DragDropDashboard, AlertBanner
- **Molecules**: DateRangePicker, PriorityPicker, TechnicianSelector
- **Auth**: PermissionGuard, RoleProtectedRoute, WebAuthnSetup
- **Devices**: DeviceWizard, DeviceAuditLog, AssetTree
- **SLA**: SLAGaugePanel, SLABreachTimeline, SLAHeatmap, SLATrendChart
- **Work Orders**: PhotoAnnotation, ConditionalChecklist, WOChat
- **Webhooks**: WebhookBuilder, WebhookLogFilter, WebhookRetryPolicy, WebhookStatsCards
- **Pages**: EventReplay, PlaybookMarketplace, APIVersioning, Glossary
- **P2P**: PTZControls, P2PRegistrationForm
- **RCA**: RCAGraph, RCAWidget
- **AI**: AIAssistantPanel
- **Custom Fields**: FieldBuilder, WhiteLabelCustomizer

### Запуск

```bash
cd frontend
npm run storybook
```

После запуска Storybook будет доступен на `http://localhost:6006`.

### Написание stories

1. Создайте файл `ComponentName.stories.tsx` рядом с компонентом
2. Используйте `Meta` и `StoryObj` типы из `@storybook/react`
3. Добавьте `tags: ['autodocs']` для автоматической документации
4. Опишите props через `args` и варианты через отдельные `Story`

```tsx
import type { Meta, StoryObj } from '@storybook/react';
import { Button } from './Button';

const meta: Meta<typeof Button> = {
  title: 'UI/Button',
  component: Button,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Button>;

export const Primary: Story = {
  args: { variant: 'primary', children: 'Click me' },
};

export const Secondary: Story = {
  args: { variant: 'secondary', children: 'Cancel' },
};
```

### Структура Storybook

Stories организованы по категориям в соответствии с иерархией компонентов:

```
UI/          — примитивные компоненты (Button, Card, Modal, Input...)
Layout/      — компоненты макета (Header, Sidebar, PageSuspense...)
Dashboard/   — виджеты дашборда
Auth/        — компоненты аутентификации и авторизации
Devices/     — компоненты управления устройствами
SLA/         — SLA-компоненты (gauges, heatmap, trends...)
WorkOrders/  — PhotoAnnotation, ConditionalChecklist
Webhooks/    — WebhookBuilder, LogFilter, RetryPolicy, StatsCards
P2P/         — PTZControls, P2PRegistrationForm
RCA/         — RCAGraph, RCAWidget
Chat/        — WOChat
AI/          — AIAssistantPanel
Checklists/  — ConditionalChecklist
CustomFields/— FieldBuilder
Organisms/   — AssetTree, BeforeAfterSlider
Pages/       — EventReplay, PlaybookMarketplace, APIVersioning
```

Для страниц с API-зависимостями используйте декораторы `MemoryRouter` + `QueryClientProvider`.

---

## Swagger UI / OpenAPI

API документация доступна через Swagger UI:

- **OpenAPI spec (JSON):** [`GET /api/v1/openapi.json`](http://localhost:8080/api/v1/openapi.json)
- **Swagger UI (HTML):** [`GET /api/v1/docs`](http://localhost:8080/api/v1/docs)

Спецификация автоматически генерируется из метаданных маршрутов в [`backend/internal/api/openapi.go`](backend/internal/api/openapi.go).

### Добавление маршрутов в OpenAPI

1. Добавьте `RouteMeta` в функцию `DefaultRoutes()` в [`openapi.go`](backend/internal/api/openapi.go)
2. Укажите метод, путь, тег, описание и требования к аутентификации
3. При необходимости добавьте схему в `DefaultSchemas()`

Сейчас задокументировано **97 маршрутов** в 20 категориях.

---

## Troubleshooting

### Port already in use

```bash
# Проверить, кто занял порт
lsof -i :8080   # backend
lsof -i :5173   # frontend
lsof -i :4222   # NATS

# Завершить процесс
kill -9 <PID>
```

### Database connection failed

1. Проверьте, что PostgreSQL запущен:
   ```bash
   systemctl status postgresql   # Linux
   pg_isready                    # проверка подключения
   ```
2. Проверьте значения в `.env` (или `DATABASE_URL`)
3. Убедитесь, что база данных создана:
   ```bash
   createdb cctv_monitor
   ```

### NATS connection

1. Убедитесь, что NATS сервер запущен:
   ```bash
   nats-server -p 4222
   ```
2. Для фонового запуска:
   ```bash
   nats-server -p 4222 -D &> /tmp/nats.log &
   ```
3. Проверка подключения:
   ```bash
   nats pub test.hello "ping"   # публикация тестового сообщения
   ```

### CORS errors

1. Проверьте значение `CORSAllowedOrigins` в [`backend/config.yaml`](backend/config.yaml)
2. Для локальной разработки укажите `http://localhost:5173`
3. При изменении конфига — перезапустите backend

### Migration errors

```bash
cd backend

# Откатить последнюю миграцию
go run cmd/migrate/main.go -down 1

# Принудительно установить версию
go run cmd/migrate/main.go -force 20240101000000

# Посмотреть статус миграций
go run cmd/migrate/main.go -verbose
```

### Go module errors

```bash
# Очистить кэш и переустановить
go clean -modcache
go mod download

# Синхронизировать зависимости
go mod tidy
```

### Frontend build errors

```bash
cd frontend

# Очистить кэш Vite
rm -rf node_modules/.vite

# Переустановить зависимости
rm -rf node_modules && npm install
```

---

## Migration Guide

### Database Migrations

Все изменения схемы БД выполняются через [`golang-migrate`](https://github.com/golang-migrate/migrate).
Миграции находятся в [`backend/migrations/`](backend/migrations/).

```bash
cd backend

# Создать новую миграцию
migrate create -ext sql -dir migrations -seq add_camera_firmware

# Применить все миграции
go run cmd/migrate/main.go -up

# Откатить последнюю миграцию
go run cmd/migrate/main.go -down 1

# Проверить статус
go run cmd/migrate/main.go -verbose
```

**Важно:** Миграции нумеруются последовательно (000001, 000002...).
Не редактируйте уже применённые миграции — создавайте новые.

### Code Migration Patterns

При рефакторинге между версиями API:

1. **Add** — добавьте новый endpoint/тип, сохранив старый
2. **Deprecate** — пометьте старый endpoint `Deprecated: true` в OpenAPI
3. **Migrate** — обновите клиентов (frontend, mobile, integrations)
4. **Remove** — удалите старый endpoint после sunset date

Текущие версии API: `/api/v1` (стабильная).

### Feature Flag Strategy

Используйте feature flags через конфиг для поэтапного включения:

```yaml
# config.yaml
features:
  new_analytics_pipeline: false
  predictive_maintenance: true
  offline_mode: false
```

---

## Glossary

Проект содержит встроенный глоссарий технических терминов на странице [`/glossary`](frontend/src/pages/Glossary.tsx).

**Покрытие: 60+ терминов** в категориях:
- Device & Hardware — NVR, DVR, MTBF, MTTR
- Network & Protocols — ONVIF, RTSP, PoE, VLAN, QoS, Multicast
- Video & Codecs — H.264, H.265, FPS, Bitrate, Resolution
- Analytics & AI — VCA, Motion Detection, LPR/ANPR
- Performance & Reliability — SLO, SLI, OEE, FCR, CSAT
- Compliance & Security — IEC 62443, KII, NIS2, GDPR, DPIA, OAC-66, STB Crypto
- Security & Access Control — RBAC, MFA, WebAuthn, TLS, LDAP, OAuth2
- CCTV Operations — RCA, Blast Radius, Health Score
- Work Orders & CMMS — CMMS, EAM, Preventive/Corrective Maintenance, RCM, FMEA
- Monitoring & Metrics — Uptime, SNMP, Syslog

Новые термины добавляются в массив `GLOSSARY_ENTRIES` с указанием `id`, `term`, `definition`, `category` и опционального `seeAlso` для перекрёстных ссылок.

---

## Contributing

1. Создайте feature-ветку от `develop`:
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/P3-xxx-short-description
   ```

2. Соблюдайте convention коммитов:
   ```
   type(scope): message

   feat(auth): add bign-based JWT signing
   fix(db): correct migration sequence
   refactor(cmms): extract adapter interface
   docs(readme): update quick start
   test(auth): add table-driven tests for token validation
   ```

3. Запустите тесты перед push:
   ```bash
   cd backend && go test ./... -v
   cd frontend && npm test
   ```

4. Создайте PR с описанием изменений

5. **Code review обязателен** — минимум один approving review

### Compliance-проверка перед PR

Каждый PR должен проходить compliance-check по матрице стандартов:

- [ ] Криптография: СТБ 34.101.30 (belt/bign/bash) для production
- [ ] Audit trail: все мутации данных логируются
- [ ] Input validation: whitelist-валидация на каждом endpoint
- [ ] Безопасность: OWASP ASVS L3
- [ ] Тесты: unit + security + compliance

---

## Architecture Decisions

Все архитектурные решения документированы в формате ADR в [`docs/adr/`](docs/adr/).

### Список ADR

| ADR | Описание | Статус |
|-----|----------|--------|
| [ADR-001](docs/adr/ADR-001-headless-cmms.md) | Headless CMMS Architecture | ✅ Accepted |
| [ADR-002](docs/adr/ADR-002-cmms-adapter-pattern.md) | CMMS Adapter Pattern | ✅ Accepted |
| [ADR-003](docs/adr/ADR-003-event-bus.md) | Event Bus (NATS) | ✅ Accepted |
| [ADR-004](docs/adr/ADR-004-gatekeeper-pattern.md) | Gatekeeper Pattern | ✅ Accepted |
| [ADR-005](docs/adr/ADR-005-state-management.md) | State Management (Zustand) | ✅ Accepted |
| [ADR-006](docs/adr/ADR-006-offline-first.md) | Offline-First Architecture (WatermelonDB) | ✅ Accepted |
| [ADR-007](docs/adr/ADR-007-rca-graph-engine.md) | RCA Graph Engine | ✅ Accepted |
| [ADR-008](docs/adr/ADR-008-sla-engine.md) | SLA Engine with Escalation | ✅ Accepted |
| [ADR-009](docs/adr/ADR-009-webhook-delivery.md) | Webhook Delivery with Retry | ✅ Accepted |
| [ADR-010](docs/adr/ADR-010-visual-regression-qa.md) | Visual Regression QA Pipeline | ✅ Accepted |
| [ADR-011](docs/adr/ADR-011-security-zones.md) | IEC 62443 Security Zones | ✅ Accepted |
| [ADR-012](docs/adr/ADR-012-audit-trail.md) | Tamper-Proof Audit Trail | ✅ Accepted |
| [ADR-013](docs/adr/ADR-013-ddd-bounded-contexts.md) | DDD Bounded Contexts | ✅ Accepted |
| [ADR-014](docs/adr/ADR-014-multi-tenant-rls.md) | Multi-Tenant with RLS | ✅ Accepted |
| [ADR-015](docs/adr/ADR-015-csp-headers.md) | Content Security Policy | ✅ Accepted |
| [ADR-016](docs/adr/ADR-016-jwt-rotation.md) | JWT Refresh Rotation | ✅ Accepted |
| [ADR-017](docs/adr/ADR-017-ai-gatekeeper.md) | AI Gatekeeper Pattern | ✅ Accepted |
| [ADR-018](docs/adr/ADR-018-multi-region-architecture.md) | Multi-Region Architecture | ✅ Accepted |
| [ADR-019](docs/adr/ADR-019-helm-deployment.md) | Helm Chart Deployment | ✅ Accepted |
| [ADR-020](docs/adr/ADR-020-compliance-profiles.md) | Regional Compliance Profiles | ✅ Accepted |

### Как создать новый ADR

1. Скопируйте шаблон: `cp docs/adr/TEMPLATE.md docs/adr/ADR-XXX-title.md`
2. Заполните статус, контекст, решение и последствия
3. Добавьте строку в таблицу выше
4. Создайте PR с пометкой `adr` в названии

---

## Compliance Standards

Система соответствует следующим стандартам (по зонам безопасности IEC 62443):

| Зона | Стандарты | Статус |
|------|-----------|--------|
| Zone 1 (Frontend) | OWASP ASVS L3 (V1-V5), WCAG 2.1 AA | ✅ |
| Zone 2 (DMZ) | IEC 62443 SL-2, СТБ 34.101.30 (TLS) | ✅ |
| Zone 3 (Backend) | IEC 62443 SL-3, СТБ 34.101.30, ISO 27001 | ✅ |
| Zone 4 (Data) | IEC 62443 SL-3, СТБ belt-gcm | ✅ |
| Zone 5 (Edge) | IEC 62443 SL-4 (отложен) | ⏳ |

### Криптография (СТБ 34.101.30)

| Алгоритм | Статус | Использование |
|----------|--------|---------------|
| bign-curve256v1 (ECDSA P-256) | ✅ Active | JWT подпись (ES256) |
| bash-256 (SHA-256 placeholder) | ⚠️ Stub | Audit log HMAC |
| belt-GCM (AES-256-GCM placeholder) | ⚠️ Stub | Encrypt/Decrypt |
| belt-KDF | ✅ Active | Key derivation |

> **Важно:** `bp2012/crypto` недоступен (private repo). Используются Go standard library
> алгоритмы (`crypto/ecdsa`, `crypto/aes`, `crypto/sha256`) как временное решение.
> После получения сертифицированного SDK от ОАЦ — замена одним PR.

### Региональная настройка (Compliance Profiles)

Система поддерживает региональные compliance-профили через [`backend/internal/compliance/`](backend/internal/compliance/).

#### Belarus (BY) — КИИ РБ

```yaml
# config.yaml
region: BY
compliance:
  crypto_provider: belt        # СТБ 34.101.30
  audit_retention_years: 7     # КИИ РБ
  data_localization: true      # 149-ФЗ / КИИ
  kii_class: KII-2
  oac_p66_enabled: true        # Приказ ОАЦ №66
  sig_check_interval: 5m       # Контроль целостности
```

**Ключевые требования:**
- Криптография: ТОЛЬКО СТБ 34.101.30 (belt/bign/bash)
- Audit log: 7 лет, HMAC + prev_hash chain
- Импортозамещение: сертифицированные СКЗИ от ОАЦ
- Data localization: все ПДн на территории РБ
- Контроль целостности: bash-256 хеш бинарников каждые 5 минут

#### Russia (RU) — 149-ФЗ / 152-ФЗ / КИИ РФ

```yaml
# config.yaml
region: RU
compliance:
  crypto_provider: gost        # ГОСТ Р 34.10-2012, ГОСТ Р 34.11-2012
  audit_retention_years: 5     # 149-ФЗ
  data_localization: true      # 152-ФЗ ст. 18
  kii_class: KII-1
  fstek_order_17: true         # Приказ ФСТЭК №17
  pdn_notification: required   # Уведомление Роскомнадзора
```

**Ключевые требования:**
- Криптография: ГОСТ Р 34.10-2012 (подпись), ГОСТ Р 34.11-2012 (хеш)
- Data localization: ПДн граждан РФ на территории РФ (152-ФЗ ст. 18)
- KII: категорирование объектов КИИ, импортозамещение
- ФСТЭК: приказ №17 (защита информации в КИИ)
- DSAR: обработка запросов субъектов ПДн в течение 30 дней

#### Kazakhstan (KZ)

```yaml
region: KZ
compliance:
  crypto_provider: gost        # ГОСТ (аналогичный RU)
  audit_retention_years: 5     # Закон О ПДн
  data_localization: true      # Закон О ПДн ст. 26
  cross_border_transfer: restricted
```

**Ключевые требования:**
- ПДн: локализация на территории РК (Закон О ПДн ст. 26)
- Криптография: ГОСТ (аналогично РФ)
- Трансграничная передача: уведомление уполномоченного органа

#### Turkey (TR) — KVKK

```yaml
region: TR
compliance:
  crypto_provider: aes         # AES-256-GCM (европейские нормы)
  audit_retention_years: 10    # KVKK ст. 12
  data_localization: optional  # KVKK
  verbis_registration: required # VERBIS (KVKK ст. 16)
  cctv_signage: required       # EN 62676, KVKK ст. 11
  dpia_required: true          # KVKK ст. 10
```

**Ключевые требования:**
- VERBIS: регистрация в реестре контроллера данных
- CCTV signage: обязательные таблички о видеонаблюдении (KVKK ст. 11)
- DPIA: обязательна для CCTV систем
- Retention: 10 лет (максимум по KVKK)
- Криптография: AES-256-GCM (европейские стандарты)

#### Vietnam (VN) — TCVN 11930

```yaml
region: VN
compliance:
  crypto_provider: aes         # AES-256 (TCVN 11930)
  audit_retention_years: 5     # Camera Standard 2025
  data_localization: true      # Decree 13/2023
  cross_border_transfer: restricted
  cctv_quality: ts-camera-2025 # TCVN Camera Standard 2025
```

**Ключевые требования:**
- Криптография: AES-256 (TCVN 11930:2017)
- Data residency: ПДн на территории VN (Decree 13/2023)
- CCTV Standard 2025: обязательная сертификация камер
- Cross-border: разрешение Ministry of Public Security

#### Indonesia (ID) — SNI 27001 / UU PDP

```yaml
region: ID
compliance:
  crypto_provider: aes         # AES-256 (SNI 27001)
  audit_retention_years: 5     # UU PDP
  data_localization: true      # UU PDP ст. 29
  sni_certification: required  # SNI 27001
  dpia_required: true          # UU PDP
```

**Ключевые требования:**
- ISMS: сертификация SNI 27001 (эквивалент ISO 27001)
- Data residency: ПДн на территории ID (UU PDP ст. 29)
- DPIA: обязательна для высокорисковой обработки
- Срок хранения: 5 лет после окончания обработки

#### Brazil (BR) — LGPD

```yaml
region: BR
compliance:
  crypto_provider: aes         # AES-256-GCM
  audit_retention_years: 5     # LGPD
  data_localization: optional  # LGPD не требует
  dpia_required: true          # LGPD ст. 38
  dpo_appointment: required    # LGPD ст. 41
  anpd_notification: required  # LGPD ст. 48 (утечки)
```

**Ключевые требования:**
- DPIA: обязательна (LGPD ст. 38)
- DPO: назначение Data Protection Officer
- Уведомление ANPD: при утечках ПДн в течение 48 часов
- Права субъектов: доступ, коррекция, анонимизация, портативность

### Переключение региона

Регион задаётся через ENV или config.yaml:

```bash
# ENV
export GB_REGION=BY

# Или в config.yaml
region: BY
```

При старте система загружает соответствующий compliance-профиль, который определяет:
- Криптопровайдер (belt / gost / aes)
- Audit retention period
- Data localization rules
- Требования к сертификации
- Ограничения на трансграничную передачу

---

## Offline-First Mobile Architecture

Мобильное приложение (React Native + Expo 52) работает по принципу **Offline-First** с синхронизацией через WatermelonDB.

### Архитектура

```
┌─────────────────────────────────────────────────────┐
│                    Mobile App                        │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │   UI Layer   │  │  Sync Queue  │  │  Local DB  │  │
│  │  (React)     │──│ (Watermelon) │──│  (SQLite)  │  │
│  └─────────────┘  └──────┬───────┘  └───────────┘  │
│                          │                          │
│                    ┌─────▼──────┐                   │
│                    │  Network   │                   │
│                    │  Adapter   │                   │
│                    └─────┬──────┘                   │
└──────────────────────────┼──────────────────────────┘
                           │
                    ┌──────▼───────┐
                    │  Backend API │
                    │  (REST + WS) │
                    └──────────────┘
```

### WatermelonDB Sync

Синхронизация работает по push/pull протоколу:

```typescript
// mobile/src/services/sync.ts
import { synchronize } from '@nozbe/watermelondb/sync';

export async function syncDatabase(database: Database) {
  await synchronize({
    database,
    pullChanges: async ({ lastPulledAt, schemaVersion, migration }) => {
      const response = await api.post('/api/v1/sync/pull', {
        last_pulled_at: lastPulledAt,
        schema_version: schemaVersion,
        migration,
      });
      return {
        changes: response.data.changes,
        timestamp: response.data.timestamp,
      };
    },
    pushChanges: async ({ changes, lastPulledAt }) => {
      await api.post('/api/v1/sync/push', {
        changes,
        last_pulled_at: lastPulledAt,
      });
    },
    migrationsEnabled: true,
  });
}
```

### Модели данных (локальные)

```typescript
// mobile/src/models/Device.ts
import { Model } from '@nozbe/watermelondb';
import { field, date, readonly, children } from '@nozbe/watermelondb/decorators';

export default class Device extends Model {
  static table = 'devices';
  static associations = {
    work_orders: { type: 'has_many' as const, foreignKey: 'device_id' },
  };

  @field('name') name!: string;
  @field('ip_address') ipAddress!: string;
  @field('status') status!: 'online' | 'offline' | 'degraded';
  @field('sync_status') syncStatus!: 'synced' | 'pending' | 'conflict';
  @readonly @date('created_at') createdAt!: Date;
  @readonly @date('updated_at') updatedAt!: Date;
  @children('work_orders') workOrders: any;
}
```

### Офлайн-приоритеты

| Операция           | Офлайн | Приоритет синхронизации | Конфликт |
|--------------------|--------|--------------------------|----------|
| Просмотр устройств | ✅     | N/A                      | N/A      |
| Создание WO        | ✅     | Высокий (немедленно)     | Last-write-wins |
| Обновление статуса | ✅     | Высокий (немедленно)     | Last-write-wins |
| Фото/видео         | ✅     | Низкий (фон)             | Keep-both |
| Геолокация         | ✅     | Средний                  | Server-wins |
| Удаление           | ❌     | Только online            | N/A      |

### Адаптер синхронизации

```typescript
// mobile/src/services/SyncAdapter.ts
export class SyncAdapter {
  private syncInProgress = false;
  private retryCount = 0;
  private maxRetries = 5;

  async syncWithRetry(): Promise<void> {
    if (this.syncInProgress) return;
    this.syncInProgress = true;

    try {
      await syncDatabase(database);
      this.retryCount = 0;
    } catch (error) {
      this.retryCount++;
      if (this.retryCount < this.maxRetries) {
        const delay = Math.pow(2, this.retryCount) * 1000; // exponential backoff
        setTimeout(() => this.syncWithRetry(), delay);
      }
    } finally {
      this.syncInProgress = false;
    }
  }
}
```

### Конфликт-резолюция

При конфликте синхронизации:

1. **Last-write-wins** — для статусов и текстовых полей (по `updated_at`)
2. **Server-wins** — для геолокации и телеметрии
3. **Keep-both** — для фото/видео (создаются дубликаты с пометкой `conflict`)
4. **Manual resolution** — для критических данных (work order assignment)

Конфликты логируются в `sync_conflicts` таблице для аудита.

### Настройка окружения для офлайн-разработки

```bash
cd mobile

# Установить зависимости
npm install

# Запустить Expo с офлайн-режимом
npx expo start

# Эмуляция офлайн-режима (через React Native Debugger)
# - Включите "Network: Offline" в DevTools
# - Проверьте работу Sync Queue

# Очистить локальную БД (сброс sync state)
npx expo start --clear
```
