# Архитектура CCTV Health Monitor

## Обзор

CCTV Health Monitor — платформа для мониторинга, управления и обслуживания систем видеонаблюдения (CCTV). Относится к КИИ (Критическая Информационная Инфраструктура) РБ, класс KII-2.

**Стек:** Go 1.25 + React 19 + React Native + PostgreSQL/TimescaleDB + NATS JetStream

---

## 1. Компонентная архитектура

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      Frontend (React 19 + Vite 8)                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
│  │Dashboard │ │WorkOrders│ │ Devices  │ │ ... 40+  │ │  СSP Headers  │  │
│  │          │ │          │ │          │ │  pages   │ │ (vite-plugin) │  │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └───────────────┘  │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────────────────┐  │
│  │  Zustand  │ │ReactQuery│ │WebSocket │ │ useFocusTrap (nested mod.)│  │
│  │   v5     │ │   v5     │ │(alarms)  │ │ WCAG 2.1 AA + Focus Trap  │  │
│  └──────────┘ └──────────┘ └──────────┘ └────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  PWA (vite-plugin-pwa) — Service Worker + Offline Caching       │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└──────────────────────────┬───────────────────────────────────────────────┘
                           │ HTTP/mTLS 1.3
┌──────────────────────────▼───────────────────────────────────────────────┐
│                Backend API (Go + Chi, :8080)                             │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌────────────────┐  │
│  │  Auth   │ │  CMMS    │ │  Events  │ │   AI/ML  │ │    40+         │  │
│  │ JWT Rot.│ │          │ │          │ │ Vision   │ │   handlers     │  │
│  │ Refresh │ │          │ │          │ │ Guard    │ │                │  │
│  └─────────┘ └──────────┘ └──────────┘ └──────────┘ └────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  Middleware: Auth, CORS, Rate Limiter, Feature Flags, CSP        │   │
│  │  Audit Trail (HMAC bash-256), RLS (Row-Level Security)           │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│  ┌───────────────┐ ┌──────────────────┐ ┌───────────────────────────┐   │
│  │  Edge Module  │ │  Compliance     │ │  Telegram Vault           │   │
│  │  PQ Hybrid    │ │  Engine         │ │  (Vault/Env Token Prov.)  │   │
│  │  WireGuard    │ │  (GDPR/NIS2/КИИ)│ │                          │   │
│  └───────────────┘ └──────────────────┘ └───────────────────────────┘   │
└──────┬─────────────┬──────────────┬──────────────────┬──────────────────┘
       │             │              │                  │
┌──────▼──┐   ┌──────▼──────┐  ┌───▼───────────┐  ┌──▼──────────────────┐
│PostgreSQL│   │TimescaleDB  │  │NATS JetStream  │  │  ML Prediction     │
│ (основная│   │(метрики,    │  │(event bus, KV, │  │  Queue (WorkQueue) │
│  БД)     │   │ телеметрия) │  │ WorkQueue)     │  │  JetStream →       │
│          │   │             │  │                │  │  Python Workers    │
│   + RLS  │   │             │  │                │  │                   │
└──────────┘   └─────────────┘  └────────────────┘  └───────────────────┘
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

### Edge Agent (Zone 5 — IoT/Edge)

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Edge Agent (Go)                                │
│  ┌──────────────────┐ ┌──────────────────┐ ┌─────────────────────┐  │
│  │  OTA Updater     │ │  WireGuard VPN   │ │  Protocol Proxies   │  │
│  │  Dual-boot A/B   │ │  X25519 + ML-KEM │ │  HTTP, SSH, Proxy   │  │
│  │  Ed25519 Signed  │ │  (PQ Hybrid)     │ │                    │  │
│  └──────────────────┘ └──────────────────┘ └─────────────────────┘  │
│  ┌──────────────────┐ ┌──────────────────┐                          │
│  │  Lazy VPN        │ │  Security Monitor│                          │
│  │  (on-demand conn)│ │  Tamper Detection│                          │
│  └──────────────────┘ └──────────────────┘                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Mobile (React Native + Expo 52)

```
┌──────────────────────────────────────────────────────────────────────┐
│                   Mobile App (React Native + Expo 52)                │
│  ┌──────────────┐ ┌──────────────────┐ ┌──────────────────────────┐  │
│  │ WatermelonDB │ │  Background Sync │ │  40+ Screens             │  │
│  │ (SQLite ORM) │ │  (offline queue) │ │                         │  │
│  └──────────────┘ └──────────────────┘ └──────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Gatekeeper API Client — Photo Upload + Vision Guard Check   │   │
│  └──────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
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
| [`internal/db/`](backend/internal/db/) | Слой БД (pgx/v5) | `db.go`, `repository.go`, `migrate.go`, `rls.go`, `pool.go`, `monitor.go`, `slow_query.go` |
| [`internal/auth/`](backend/internal/auth/) | Аутентификация | `jwt.go`, `middleware.go`, `password.go`, `session_policy.go`, `ldap.go`, `saml.go`, **`refresh_token.go`** — JWT Rotation + Reuse Detection + Fingerprint |
| [`internal/events/`](backend/internal/events/) | Event Store + NATS | `store.go`, `publisher.go`, `subscriber.go`, `projection.go`, `replay.go`, `report_queue.go` |
| [`internal/crypto/`](backend/internal/crypto/) | Криптография | `providers/` — AES, belt, gost, sm, hash_bash, signature_bign |
| [`internal/sla/`](backend/internal/sla/) | SLA engine | `engine.go`, `policy.go`, `notifier.go`, `worker.go` |
| [`internal/compliance/`](backend/internal/compliance/) | Compliance engine | `engine.go`, `profile.go`, `providers.go`, `gdpr.go`, `nis2.go`, `regulatory_cron.go`, `electronic_journal.go`, `incident_response.go`, `eu_cra.go`, `cert_in.go` |
| [`internal/cmms/`](backend/internal/cmms/) | CMMS адаптеры | `adapter.go`, `jira/`, `servicenow/`, `toir/`, `atlas_adapter.go` |
| [`internal/protocols/`](backend/internal/protocols/) | Протоколы CCTV | `dahua.go`, `hikvision.go`, `onvif.go`, `snmp.go`, `ftp.go` |
| [`internal/rca/`](backend/internal/rca/) | Root Cause Analysis | `engine.go`, `graph_builder.go` |
| [`internal/audit/`](backend/internal/audit/) | Audit trail | `chain.go`, `signer.go` — tamper-proof лог с HMAC bash-256 |
| [`internal/state/`](backend/internal/state/) | State manager | `manager.go`, `jetstream_manager.go` — Redis или NATS KV |
| [`internal/sync/`](backend/internal/sync/) | ITSM sync engine | `sync.go`, `conflict.go` |
| [`internal/gatekeeper/`](backend/internal/gatekeeper/) | Gatekeeper (AI) | `verifier.go`, `ai.go`, `exif.go`, `gps.go` |
| [`internal/ai/`](backend/internal/ai/) | **AI/ML модули (NEW)** | **`vision_guard.go`** — QR/text detection для защиты от prompt injection; `deepseek.go`, `claude.go`, `ollama.go`, `openai.go`, `vllm.go` — AI провайдеры; `anomaly_detector.go`, `anomaly_service.go` |
| [`internal/ml/`](backend/internal/ml/) | **ML Prediction (NEW)** | **`prediction_queue.go`** — NATS JetStream WorkQueue для предсказания отказов; `prediction_service.go` — сервисный слой; `config.go` — ML конфигурация |
| [`internal/worker/`](backend/internal/worker/) | Worker pool | `pool.go`, `quota.go` |
| [`internal/webhook/`](backend/internal/webhook/) | Webhook delivery | `verify.go`, `delivery.go`, `pg_store.go` |
| [`internal/tenant/`](backend/internal/tenant/) | Tenant management | `quota.go`, `branding.go` |
| [`internal/playbook/`](backend/internal/playbook/) | Playbook marketplace | `marketplace.go` |
| [`internal/dr/`](backend/internal/dr/) | Disaster Recovery | `health.go`, `failover.go`, `drills.go` |
| [`internal/integrations/`](backend/internal/integrations/) | External integrations | `calendar/google.go`, `calendar/outlook.go`, `calendar/sync.go`, `oauth2/` |
| [`internal/edge/`](backend/internal/edge/) | **Edge/VPN модули (NEW)** | **`pq_hybrid.go`** — Post-Quantum Hybrid (X25519 + ML-KEM); `wireguard_server.go`, `wg_config_generator.go`, `vpn_session_manager.go`, `vpn_session_config.go`, `lazy_vpn.go`, `http_proxy.go`, `ssh_proxy.go`, `agent.go`, `security.go` |
| [`internal/telegram/`](backend/internal/telegram/) | **Telegram Bot (NEW)** | **`token_provider.go`** — Vault/Env Token Provider с rotation; `bot.go` — Telegram bot |

### Frontend (`frontend/`)

| Директория | Назначение |
|-----------|-----------|
| [`src/pages/`](frontend/src/pages/) | 40+ страниц (lazy-loaded) |
| [`src/components/`](frontend/src/components/) | UI компоненты (ui/), бизнес-компоненты (work-orders/, sla/, dashboard/) |
| [`src/hooks/`](frontend/src/hooks/) | React hooks — **`useAccessibility.ts` (NEW)** — Focus Trap Stack для вложенных модалок, WCAG 2.1 AA; `useApiQuery`, `useAuth`, `useBulkOperations` |
| [`src/services/`](frontend/src/services/) | API клиенты (axios), WebSocket |
| [`src/store/`](frontend/src/store/) | Zustand stores (auth, theme, settings, ui) |
| [`src/context/`](frontend/src/context/) | React context (Theme, Settings, Reports) |
| [`src/types/`](frontend/src/types/) | TypeScript типы (api.ts, index.ts, workflow.ts, p2p.ts) |
| [`src/lib/`](frontend/src/lib/) | Утилиты (sentry.ts, validations/, deepseek.ts) |
| [`src/locales/`](frontend/src/locales/) | i18n (12 языков) |
| **`vite.config.ts`** (NEW) | **CSP Headers Plugin** — `Content-Security-Policy` без `unsafe-inline` в production; PWA конфигурация; оптимизация сборки |

### Mobile (`mobile/`)

| Директория | Назначение |
|-----------|-----------|
| [`src/database/`](mobile/src/database/) | **WatermelonDB (NEW)** — `schema.ts`, `models.ts`, `index.ts` — reactive SQLite ORM с offline-очередью мутаций; 4 модели: WorkOrder, Device, Site, PendingMutation |
| [`src/api/`](mobile/src/api/) | API клиенты — `gatekeeper.ts`, `sync.ts` |
| [`src/hooks/`](mobile/src/hooks/) | React hooks — `useBackgroundSync.ts` |

### Edge Agent (`edge-agent/`)

| Директория | Назначение |
|-----------|-----------|
| [`internal/agent/`](edge-agent/internal/agent/) | **OTA Updater (NEW)** — `ota.go` — dual-boot A/B с Ed25519 подписью, атомарный symlink, health check с auto-rollback |
| [`internal/agent/`](edge-agent/internal/agent/) | Proxy и VPN (существующие): `vpn.go`, `config.go`, `proxy.go` |

### P2P Gateway (`p2p-gateway/`)

Отдельный шлюз для P2P-подключения к камерам (Dahua, Hikvision, Reolink и др.).

### CI/CD (`.github/workflows/`)

| Файл | Назначение |
|------|-----------|
| **`sbom.yml` (NEW)** | **SBOM Generation** — CycloneDX v1.6 для Go/npm/Expo; VEX (OpenVEX) с osv-scanner; EU CRA + US EO 14028 compliance |

---

## 3. Зоны безопасности (IEC 62443)

| Зона | Компоненты | Уровень безопасности | Новые модули |
|------|-----------|---------------------|--------------|
| Zone 1 (Enterprise) | Frontend, Public API | SL-1 | CSP Headers, Focus Trap Stack, PWA |
| Zone 2 (DMZ) | API Gateway, Rate Limiter | SL-2 | — |
| Zone 3 (Application) | Backend, CMMS, NATS | SL-3 | Vision Guard, JWT Refresh Rotation, Prediction Queue, Compliance Engine, Telegram Vault |
| Zone 4 (Data) | PostgreSQL, TimescaleDB | SL-3 | RLS, Audit Trail (bash-256 HMAC) |
| Zone 5 (Edge) | Edge Agent | SL-4 | **OTA A/B (SL-4)**, **WireGuard PQ Hybrid (SL-4)**, Ed25519 signature, Lazy VPN |

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

### 4.4 Vision Guard — Prompt Injection Protection (NEW)
```
                      ┌─────────────────────────────┐
                      │     Vision Guard (P0-CR-09) │
Photo Upload ────────►│                             │
                      │  1. Size validation         │
                      │  2. Image decode (JPEG/PNG) │
                      │  3. QR/Barcode detection    │──► gozxing
                      │     (QR, DataMatrix, Aztec) │   (pure Go)
                      │  4. Text region detection   │──► Sobel edge
                      │     (edge density analysis) │    detection
                      │  5. Decision:               │
                      │     • Strict mode → reject  │
                      │     • Warn mode → log only  │
                      └─────────────┬───────────────┘
                                    │
                         ┌──────────▼──────────┐
                         │  Если rejected:     │
                         │  • 422 Unprocessable │
                         │  • Audit log        │
                         │  • Warnings в ответе │
                         └─────────────────────┘
                                    │
                         ┌──────────▼──────────┐
                         │  Если passed:        │
                         │  → AI Provider       │
                         │    (DeepSeek/Claude/ │
                         │     OpenAI/Ollama)   │
                         └─────────────────────┘
```

**Compliance:** OWASP ASVS V5.1 (Input validation — image content), IEC 62443 SR 3.3 (Security monitoring), Приказ ОАЦ №66 п. 7.18.2 (Контроль целостности данных).

### 4.5 JWT Refresh Token Rotation (NEW)
```
                  ┌───────────────────────────────────┐
                  │       Refresh Token Rotation       │
                  │          (P1-HI-05)                │
                  │                                    │
Client ──POST────►│  1. Validate old refresh token     │
/refresh          │  2. Check expiry                   │
                  │  3. REUSE DETECTION:               │
                  │     • Token уже revoked?           │──► Revoke entire
                  │     • → Revoke token family        │    token family
                  │  4. Fingerprint match:             │
                  │     • SHA-256(User-Agent + IP)    │
                  │     • Mismatch → 401 (possible     │
                  │       token theft)                 │
                  │  5. Revoke old session             │
                  │  6. Generate new opaque token       │
                  │     (crypto/rand, 32 bytes)        │
                  │  7. Store new session (same family) │
                  │  8. Return new refresh token        │
                  │     + new access token (JWT bign)  │
                  └─────────────┬─────────────────────┘
                                │
                     ┌──────────▼──────────┐
                     │  Response:          │
                     │  • HttpOnly cookie  │──► refresh_token
                     │  • JSON body        │──► access_token
                     │  • X-Fingerprint    │──► fingerprint hash
                     └─────────────────────┘
```

**Compliance:** OWASP ASVS V3.2.2 (Rotation), V3.2.3 (Reuse detection), V3.2.4 (Device binding), Приказ ОАЦ №66 п. 7.18.1 (Уникальная идентификация).

### 4.6 Prediction Queue — ML WorkQueue via NATS (NEW)
```
                     ┌────────────────────────────────────┐
                     │      Prediction Queue (P0-CR-04)   │
                     │                                    │
Backend Cron/Scheduler                                │
  └─► PublishTask(device_id, model_variant, trace_id)  │
       └─► NATS JetStream Stream (WorkQueuePolicy) ────┤
                     │   Subject: predictions.>          │
                     │   Retention: WorkQueue (auto-del) │
                     │   MaxAge: 7 days                  │
                     │   Storage: File                   │
                     └──────────────┬─────────────────────┘
                                    │ Consume (MaxAckPending backpressure)
                         ┌──────────▼──────────┐
                         │  Python Worker       │
                         │  predict_worker.py   │
                         │                      │
                         │  Ack → complete      │
                         │  Nak → retry (max 5) │
                         └─────────────────────┘
```

**Compliance:** IEC 62443-3-3 SR 3.1 (Queue-based processing with retries), ISO 27001 A.12.4 (Audit trail), OWASP ASVS L3 V1 (Input validation).

### 4.7 Edge Agent OTA Update (NEW)
```
                     ┌───────────────────────────────────────┐
                     │       OTA Update (Edge Agent)         │
                     │                                       │
  ┌──────────┐      │  1. Check version GET /api/v1/edge/   │
  │ Backend  │◄─────│     /version                           │
  │ OTA      │      │  2. Download binary + .sig             │
  │ Server   │──────►     to inactive slot (A/B)             │
  └──────────┘      │  3. Verify Ed25519 signature            │
                     │  4. Atomic symlink switch              │
                     │  5. systemctl restart edge-agent       │
                     │  6. Health check (30s timeout)         │
                     │     └─ Failure → auto-rollback        │
                     │        symlink → restart → health     │
                     └───────────────────────────────────────┘

  Файловая система:
  /usr/local/bin/
  ├── edge-agent.a      ← Slot A (binary)
  ├── edge-agent.b      ← Slot B (binary)
  ├── edge-agent        ← Symlink → active slot
  └── edge-agent.bak    ← Previous version (backup)
```

**Compliance:** IEC 62443-3-3 SL-3 (Signed firmware), Приказ ОАЦ №66 п. 7.18.3 (Контроль целостности — Ed25519), п. 7.18.5 (Управление обновлениями — атомарное обновление с rollback), OWASP ASVS V12 (File integrity).

### 4.8 WireGuard Post-Quantum Hybrid Key Exchange (NEW)
```
                    ┌─────────────────────────────────────┐
                    │   PQ Hybrid Key Exchange (P1-HI-06) │
                    │                                     │
  VPN Session       │  Гибридный session key:            │
  Init ────────────►│  KDF(X25519_shared || MLKEM_shared)│
                    │                                     │
                    │  • X25519 — классический ECDH       │
                    │  • ML-KEM-768 (placeholder) —       │
                    │    пост-квантовая KEM               │
                    │  • 1184-byte PQ public key          │
                    │    (CSPRNG до HW HSMS)             │
                    │  • CNSA 2.0 совместимость           │
                    └─────────────────────────────────────┘
```

**Compliance:** IEC 62443-3-3 SR 4.2 (Key generation), Приказ ОАЦ №66 п. 7.18.2 (Криптографическая защита каналов), CNSA 2.0 (Hybrid X25519 + ML-KEM).

### 4.9 Telegram Vault Token Provider (NEW)
```
                    ┌──────────────────────────────────────┐
                    │   TokenProvider (P2-MED-04)           │
                    │                                      │
  Bot Startup ─────►│  GetToken(ctx)                       │
                    │                                      │
                    │  ┌─ VaultEnabled? ── YES ──► Read    │
                    │  │                        Vault      │
                    │  │                        Secret     │
                    │  │                        (path)     │
                    │  └─ NO / Error ──────────► Env       │
                    │                           Fallback   │
                    │                           (env var)  │
                    │                                      │
                    │  Поддержка rotation:                 │
                    │  GetToken() вызывается при каждом    │
                    │  переподключении бота                │
                    └──────────────────────────────────────┘
```

**Compliance:** IEC 62443-3-3 SR 4.2 (Централизованное управление секретами), ISO 27001 A.9.4.3, OWASP ASVS V2.10, Приказ ОАЦ №66 п. 7.18.4 (Защита credentials).

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
| [ADR-018](docs/adr/ADR-018-multi-region-architecture.md) | Multi-Region Geo-Redundancy (Active-Passive per tenant) |
| **P0-CR-09** | **Vision Guard — DeepSeek Vision Prompt Injection Protection** — QR/barcode + text region detection через gozxing и Sobel edge analysis |
| **P0-CR-04** | **Prediction Queue — NATS JetStream WorkQueue** — замена subprocess на очередь с backpressure (MaxAckPending) и graceful shutdown |
| **P1-HI-05** | **Refresh Token Rotation** — opaque токены (32 байта crypto/rand) + rotation + reuse detection + device fingerprint (SHA-256) |
| **P1-HI-06** | **Post-Quantum Hybrid Key Exchange** — X25519 + ML-KEM-768 для WireGuard VPN сессий (CNSA 2.0) |
| **P2-MED-04** | **Telegram Token Provider** — Vault + env fallback с поддержкой rotation без рестарта бота |
| **P2-MED-14** | **Focus Trap Stack** — глобальный стек вложенных focus trap'ов для модальных окон (WCAG 2.4.3) |
| **P2-MED-03** | **SBOM + VEX CI** — CycloneDX v1.6 SBOM генерация для Go/npm/Expo + OpenVEX vulnerability disclosure |
| **P2-MED-21** | **CSP Headers Plugin** — Content-Security-Policy без `unsafe-inline` в production (OWASP ASVS V5.3.3) |
| P0-PDF | Server-side PDF с HMAC+QR |
| P0-REG | Maintenance Compliance Engine |
| P1-RATE | Redis distributed rate limiting |
| P1-QUOTA | Tenant quota management |
| P1-REPLAY | NATS JetStream Event Replay |
| P1-MARKET | Playbook Marketplace |
| P1-CALENDAR | Google + Outlook Calendar Sync |
| P2-BI | Self-Service Analytics query builder |
| P2-CHAT | WebSocket Chat per Work Order |
| P2-API | API Versioning Strategy |
| P3-DR | Disaster Recovery Automation |

---

## 6. Compliance & Security

Проект соответствует:

| Стандарт | Область | Новые меры |
|----------|---------|------------|
| **СТБ IEC 62443** | Industrial Automation Security (SL-1..SL-4) | Vision Guard (SR 3.3), PQ Hybrid (SR 4.2), OTA (SL-3 signed firmware), JWT Rotation (SR 2.1), Prediction Queue (SR 3.1) |
| **ISO/IEC 27001:2022** | ISMS (A.5-A.18) | A.12.4 (Audit trail — все новые модули), A.9.2.1 (Device binding), A.9.4.3 (Secrets), A.15.1 (Supply chain — SBOM) |
| **ISO/IEC 27019** | ICS/SCADA Security | Расширено на Edge Zone (A.11) |
| **СТБ 34.101.30** | Криптография (belt/bign/bash) | JWT c подписью bign, HMAC bash-256 для audit trail |
| **СТБ 34.101.27** | Защита информации | Vision Guard (целостность данных), OTA (контроль целостности) |
| **OWASP ASVS L3** | Application Security | V3.2 (Session management — rotation, reuse, fingerprint); V5.1 (Image content validation); V5.3.3 (CSP); V12 (File integrity — OTA); V2.10 (Secrets) |
| **Приказ ОАЦ №66** | Защита конечных узлов и сетей | п. 7.18.1 (JWT fingerprint), п. 7.18.2 (PQ hybrid, mTLS), п. 7.18.3 (OTA Ed25519, Vision Guard), п. 7.18.4 (Telegram Vault), п. 7.18.5 (OTA atomar + rollback), п. 7.18.6 (Health check) |
| **EU CRA (Dec 2027)** | Cyber Resilience Act | SBOM + VEX в CI, supply chain compliance |
| **US EO 14028** | Supply Chain Security | SBOM generation, vulnerability scanning |
| **CNSA 2.0** | Post-Quantum Cryptography | Hybrid X25519 + ML-KEM-768 |

### Ключевые механизмы безопасности:

- **Audit trail** с HMAC-подписью (bash-256) и chain of hashes
- **Row-Level Security (RLS)** для multi-tenant изоляции
- **Rate limiting** (Redis-based distributed, token bucket, 100 read/min, 30 write/min)
- **CORS validation**, **CSP headers** (strict-dynamic, без unsafe-inline в production), X-API-Version versioning
- **API Versioning** (URL-based + SunSet headers, deprecation policy)
- **HttpOnly cookies** + CSRF protection
- **СТБ-совместимые криптопровайдеры** (belt-GCM, bign, bash)
- **JWT Refresh Token Rotation** с reuse detection и device fingerprint
- **Vision Guard** — защита AI от prompt injection через QR/barcode и text detection
- **OTA с Ed25519 signature** — защита целостности прошивки Edge Agent
- **Post-Quantum Hybrid Key Exchange** — X25519 + ML-KEM для WireGuard
- **Telegram Vault Token Provider** — централизованное управление секретами
- **SBOM + VEX CI** — прозрачность цепочки поставок

---

## 7. Зависимости

### Backend (Go)
- **Chi** (`go-chi/chi/v5`) — HTTP router
- **pgx/v5** (`jackc/pgx`) — PostgreSQL driver
- **NATS** (`nats-io/nats.go`) — Event Bus + KV Store + JetStream WorkQueue
- **Viper** (`spf13/viper`) — Configuration
- **Chi CORS** (`go-chi/cors`) — CORS middleware
- **golang-migrate** — Database migrations
- **jwt/v5** (`golang-jwt/jwt`) — JWT tokens
- **excelize/v2** (`xuri/excelize`) — Excel export
- **gojsonschema** (`xeipuuv/gojsonschema`) — Schema validation
- **gozxing** (`makiuchi-d/gozxing`) — QR/Barcode detection (Vision Guard)
- **uuid** (`google/uuid`) — UUID generation (Refresh token family)
- **ed25519** (stdlib) — Ed25519 signature verification (OTA)
- **crypto/rand** (stdlib) — CSPRNG (Refresh tokens, PQ keys)

### Frontend (React 19)
- **React Router v7** — Routing
- **TanStack React Query v5** — Server state
- **Zustand v5** — Client state
- **TailwindCSS v4** — Styling
- **i18next** — Internationalization (20 languages, 17 lazy-loaded)
- **Nivo** — Charts (replaced Recharts)
- **Schedule-X** — Calendar views (replaced FullCalendar)
- **Sentry** — Error monitoring
- **ExcelJS** — Excel export (replaced xlsx/SheetJS)
- **Lazy-loaded**: jsPDF (server-side), react-joyride, react-datepicker
- **Centralized Icons**: `Icons.tsx` (89 lucide-react icons)
- **React Hook Form + Zod** — Form validation
- **Vite PWA Plugin** — Service Worker + Offline caching
- **Vite Image Tools** — WebP auto-conversion (sharp)

### Mobile (React Native + Expo 52)
- **WatermelonDB** (`@nozbe/watermelondb`) — Reactive SQLite ORM с JSI
- **Expo SQLite** — SQLite adapter

### Edge Agent (Go)
- **WireGuard** — VPN tunnel
- **Ed25519** (stdlib) — OTA signature verification
- **mTLS 1.3** — Communication with Backend

### CI/CD
- **cyclonedx-gomod** — Go SBOM generation
- **@cyclonedx/bom** — npm/Expo SBOM generation
- **osv-scanner** — Vulnerability scanning
- **vexctl** (OpenVEX) — VEX statement generation

---

## 8. Покрытие тестами

| Компонент | Unit | Integration | E2E | Security/Compliance |
|-----------|------|-------------|-----|---------------------|
| Backend (Go) | **90%** | testcontainers-go | — | Vision Guard (fuzzing + boundary), JWT Rotation (reuse scenarios), Prediction Queue (backpressure) |
| Frontend (TS) | **85%** | Vitest (292 теста) | Playwright (**150 сценариев**) | Focus Trap (nested modals), CSP headers |
| Mobile (RN) | — | — | Detox (**100 тестов**) | WatermelonDB offline sync |
| Edge Agent | — | — | — | OTA update (A/B switch, rollback), Ed25519 verification |

---

## 9. Convention rules

- Go: `gofmt`, `golangci-lint`, table-driven tests
- TypeScript: ESLint strict, functional components
- SQL: snake_case, индексы, транзакции, golang-migrate (запрещён `CREATE TABLE IF NOT EXISTS`)
- Файлы >500 строк — разбивать
- Хардкод секретов запрещён — только env/config
- **Compliance-first development**: перед написанием кода — compliance-check по матрице стандартов
- **СТБ-криптография в production**: запрещены AES/RSA/ECDSA/SHA — только belt/bign/bash (исключения: TLS 1.3, JWT bign, bcrypt fallback)
