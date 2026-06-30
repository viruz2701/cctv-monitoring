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

# С coverage
npm test -- --coverage

# Storybook (интерактивная документация компонентов)
npm run storybook

# TypeScript проверка
npx tsc --noEmit
```

### E2E (Playwright)

```bash
cd frontend

# Установить браузеры (один раз)
npx playwright install

# Запустить E2E тесты
npx playwright test

# С UI режимом
npx playwright test --ui

# Accessibility тесты (axe-core)
npx playwright test --grep "a11y"
```

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

Все архитектурные решения документированы в формате ADR.

| ADR | Описание |
|-----|----------|
| [ADR-001](docs/adr/ADR-001-headless-cmms.md) | Headless CMMS Architecture |
| [ADR-002](docs/adr/ADR-002-cmms-adapter-pattern.md) | CMMS Adapter Pattern |
| [ADR-003](docs/adr/ADR-003-event-bus.md) | Event Bus (NATS) |
| [ADR-004](docs/adr/ADR-004-gatekeeper-pattern.md) | Gatekeeper Pattern |
| [ADR-005](docs/adr/ADR-005-state-management.md) | State Management |
| [ADR-006](docs/adr/ADR-006-offline-first.md) | Offline-First Architecture |
| [ADR-013](docs/adr/ADR-013-ddd-bounded-contexts.md) | DDD Bounded Contexts |
| [ADR-018](docs/adr/ADR-018-multi-region-architecture.md) | Multi-Region Architecture |

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
