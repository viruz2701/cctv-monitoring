# Архитектура CCTV Health Monitor

## Обзор

CCTV Health Monitor — платформа для мониторинга, управления и обслуживания систем видеонаблюдения (CCTV). Относится к КИИ (Критическая Информационная Инфраструктура) РБ, класс KII-2.

**Стек:** Go 1.25 + React 19 + React Native + PostgreSQL/TimescaleDB + NATS JetStream

---

## 1. Компонентная архитектура

```
┌─────────────────────────────────────────────────────────────────┐
│                      Frontend (React 19)                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐  │
│  │Dashboard │ │WorkOrders│ │ Devices  │ │ ... 40+ pages     │  │
│  └──────────┘ └──────────┘ └──────────┘ └───────────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌────────────────────────────────┐  │
│  │  Zustand  │ │ ReactQuery│ │    WebSocket (alarms)        │  │
│  └──────────┘ └──────────┘ └────────────────────────────────┘  │
└──────────────────────────┬──────────────────────────────────────┘
                           │ HTTP/mTLS
┌──────────────────────────▼──────────────────────────────────────┐
│                Backend API (Go + Chi, :8080)                    │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────────┐  │
│  │  Auth   │ │  CMMS    │ │  Events  │ │    40+ handlers   │  │
│  └─────────┘ └──────────┘ └──────────┘ └───────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Middleware: Auth, CORS, Rate Limiter, Feature Flags     │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────┬─────────────┬──────────────┬────────────────────────────┘
       │             │              │
┌──────▼──┐   ┌──────▼──────┐  ┌───▼───────────┐
│PostgreSQL│   │TimescaleDB  │  │NATS JetStream  │
│ (основная│   │(метрики,    │  │(event bus, KV) │
│  БД)     │   │ теле-      │  │                │
│          │   │ метрия)    │  │                │
└──────────┘   └─────────────┘  └────────────────┘
```

### Protocol Collectors (edge-приём)

```
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│  Dahua   │ │ Hikvision│ │  SNMP    │ │   FTP    │
│ :37777   │ │ :80/443  │ │ :161     │ │ :21      │
└──────────┘ └──────────┘ └──────────┘ └──────────┘
┌──────────┐ ┌──────────┐ ┌──────────┐
│ GB28181  │ │  TVT     │ │Hisilicon │
│ :5060    │ │ :15003   │ │ :15002   │
└──────────┘ └──────────┘ └──────────┘
```

---

## 2. Структура проекта

### Backend (`backend/`)

| Директория | Назначение | Ключевые файлы |
|-----------|-----------|----------------|
| [`cmd/`](backend/cmd/) | Точки входа | `migrate/main.go` — миграции БД |
| [`main.go`](backend/main.go) | Точка входа приложения | Инициализация всех сервисов, graceful shutdown |
| [`internal/api/`](backend/internal/api/) | HTTP-слой (Chi router) | `server.go`, `auth_handlers.go`, `device_crud_handlers.go`, ... 40+ handler files |
| [`internal/config/`](backend/internal/config/) | Конфигурация (Viper) | `config.go` — все параметры из config.yaml + env |
| [`internal/db/`](backend/internal/db/) | Слой БД (pgx/v5) | `db.go`, `repository.go`, `migrate.go`, `rls.go` |
| [`internal/auth/`](backend/internal/auth/) | Аутентификация | `jwt.go`, `middleware.go`, `password.go`, `session_policy.go`, `ldap.go`, `saml.go` |
| [`internal/events/`](backend/internal/events/) | Event Store + NATS | `store.go`, `publisher.go`, `subscriber.go`, `projection.go` |
| [`internal/crypto/`](backend/internal/crypto/) | Криптография | `providers/` — AES, belt, gost, sm, hash_bash, signature_bign |
| [`internal/sla/`](backend/internal/sla/) | SLA engine | `engine.go`, `policy.go`, `notifier.go`, `worker.go` |
| [`internal/compliance/`](backend/internal/compliance/) | Compliance engine | `engine.go`, `profile.go`, `providers.go`, `gdpr.go`, `nis2.go` |
| [`internal/cmms/`](backend/internal/cmms/) | CMMS адаптеры | `adapter.go`, `jira/`, `servicenow/`, `toir/`, `atlas_adapter.go` |
| [`internal/protocols/`](backend/internal/protocols/) | Протоколы CCTV | `dahua.go`, `hikvision.go`, `onvif.go`, `snmp.go`, `ftp.go` |
| [`internal/rca/`](backend/internal/rca/) | Root Cause Analysis | `engine.go`, `graph_builder.go` |
| [`internal/audit/`](backend/internal/audit/) | Audit trail | `chain.go`, `signer.go` — tamper-proof лог |
| [`internal/state/`](backend/internal/state/) | State manager | `manager.go`, `jetstream_manager.go` — in-memory или NATS KV |
| [`internal/sync/`](backend/internal/sync/) | ITSM sync engine | `sync.go`, `conflict.go` |
| [`internal/gatekeeper/`](backend/internal/gatekeeper/) | Gatekeeper (AI) | `verifier.go`, `ai.go`, `exif.go`, `gps.go` |
| [`internal/worker/`](backend/internal/worker/) | Worker pool | `pool.go` |
| [`internal/webhook/`](backend/internal/webhook/) | Webhook delivery | `verify.go`, `delivery.go`, `pg_store.go` |

### Frontend (`frontend/`)

| Директория | Назначение |
|-----------|-----------|
| [`src/pages/`](frontend/src/pages/) | 40+ страниц (lazy-loaded) |
| [`src/components/`](frontend/src/components/) | UI компоненты (ui/), бизнес-компоненты (work-orders/, sla/, dashboard/) |
| [`src/hooks/`](frontend/src/hooks/) | React hooks (useApiQuery, useAuth, useBulkOperations) |
| [`src/services/`](frontend/src/services/) | API клиенты (axios), WebSocket |
| [`src/store/`](frontend/src/store/) | Zustand stores (auth, theme, settings, ui) |
| [`src/context/`](frontend/src/context/) | React context (Theme, Settings, Reports) |
| [`src/types/`](frontend/src/types/) | TypeScript типы (api.ts, index.ts, workflow.ts, p2p.ts) |
| [`src/lib/`](frontend/src/lib/) | Утилиты (sentry.ts, validations/, deepseek.ts) |
| [`src/locales/`](frontend/src/locales/) | i18n (12 языков) |

### P2P Gateway (`p2p-gateway/`)

Отдельный шлюз для P2P-подключения к камерам (Dahua, Hikvision, Reolink и др.).

---

## 3. Зоны безопасности (IEC 62443)

| Зона | Компоненты | Уровень безопасности |
|------|-----------|---------------------|
| Zone 1 (Enterprise) | Frontend, Public API | SL-1 |
| Zone 2 (DMZ) | API Gateway, Rate Limiter | SL-2 |
| Zone 3 (Application) | Backend, CMMS, NATS | SL-3 |
| Zone 4 (Data) | PostgreSQL, TimescaleDB | SL-3 |
| Zone 5 (Edge) | Edge Agent (отложен) | SL-4 |

Conduits между зонами: только **mTLS 1.3**.

---

## 4. Ключевые потоки данных

### 4.1 Приём телеметрии
```
Camera/SNMP → Protocol Handler → State Manager → DB Writer → PostgreSQL
                                → NATS Event → Event Store
                                → WebSocket → Frontend
```

### 4.2 Work Order lifecycle
```
Frontend → API → SLA Engine → CMMS Adapter → External CMMS
         → Event Store (audit trail)
         → Telegram/Email notification
```

### 4.3 Gatekeeper (AI-верификация фото)
```
Mobile App → Upload Photo → Gatekeeper AI (DeepSeek) → Exif проверка
                                                      → GPS проверка
                                                      → Audit log
```

---

## 5. Ключевые архитектурные решения (ADR)

| ADR | Решение |
|-----|---------|
| [ADR-001](docs/adr/ADR-001-headless-cmms.md) | Headless CMMS — отдельный слой адаптеров |
| [ADR-002](docs/adr/ADR-002-cmms-adapter-pattern.md) | CMMS Adapter Pattern для множества провайдеров |
| [ADR-003](docs/adr/ADR-003-event-bus.md) | NATS JetStream как единая событийная шина |
| [ADR-004](docs/adr/ADR-004-gatekeeper-pattern.md) | Gatekeeper для верификации полевых данных |
| [ADR-005](docs/adr/ADR-005-state-management.md) | Zustand для клиентского состояния |
| [ADR-006](docs/adr/ADR-006-offline-first.md) | Offline-first для мобильных техников |
| [ADR-013](docs/adr/ADR-013-ddd-bounded-contexts.md) | DDD Bounded Contexts |

---

## 6. Compliance & Security

Проект соответствует:

| Стандарт | Область |
|----------|---------|
| **СТБ IEC 62443** | Industrial Automation Security (SL-1..SL-4) |
| **ISO/IEC 27001:2022** | ISMS (A.5-A.18) |
| **ISO/IEC 27019** | ICS/SCADA Security |
| **СТБ 34.101.30** | Криптография (belt/bign/bash) |
| **СТБ 34.101.27** | Защита информации |
| **OWASP ASVS L3** | Application Security |
| **Приказ ОАЦ №66** | Защита конечных узлов и сетей |

Ключевые механизмы:
- Audit trail с HMAC-подписью (bash-256) и chain of hashes
- Row-Level Security (RLS) для multi-tenant изоляции
- Rate limiting (in-memory, login: 5/min, API: 100/min)
- CORS validation (запрет wildcard в production)
- HttpOnly cookies + CSRF protection
- СТБ-совместимые криптопровайдеры (belt-GCM, bign, bash)

---

## 7. Зависимости

### Backend (Go)
- **Chi** (`go-chi/chi/v5`) — HTTP router
- **pgx/v5** (`jackc/pgx`) — PostgreSQL driver
- **NATS** (`nats-io/nats.go`) — Event Bus + KV Store
- **Viper** (`spf13/viper`) — Configuration
- **Chi CORS** (`go-chi/cors`) — CORS middleware
- **golang-migrate** — Database migrations
- **jwt/v5** (`golang-jwt/jwt`) — JWT tokens
- **excelize/v2** (`xuri/excelize`) — Excel export
- **gojsonschema** (`xeipuuv/gojsonschema`) — Schema validation

### Frontend (React 19)
- **React Router v7** — Routing
- **TanStack React Query v5** — Server state
- **Zustand v5** — Client state
- **TailwindCSS v4** — Styling
- **i18next** — Internationalization (12 languages)
- **Recharts** — Charts
- **FullCalendar** — Calendar views
- **Sentry** — Error monitoring
- **React Hook Form + Zod** — Form validation

---

## 8. Покрытие тестами

| Компонент | Unit | Integration | E2E |
|-----------|------|-------------|-----|
| Backend (Go) | 70%+ | testcontainers-go | — |
| Frontend (TS) | 75%+ | Vitest | Playwright (4 сценария) |
| Mobile (RN) | — | — | Detox (4 сценария) |

---

## 9. Convention rules

- Go: `gofmt`, `golangci-lint`, table-driven tests
- TypeScript: ESLint strict, functional components
- SQL: snake_case, индексы, транзакции, golang-migrate (запрещён `CREATE TABLE IF NOT EXISTS`)
- Файлы >500 строк — разбивать
- Хардкод секретов запрещён — только env/config
