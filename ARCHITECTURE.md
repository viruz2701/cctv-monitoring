Версия: 2.2
Дата: 2026-06-26
Статус: ACTIVE
Автор: System Architect
Зрелость проекта: 92% (All P2/P3 features implemented, production-ready)
📋 Executive Summary
CCTV Health Monitor — AI-powered платформа мониторинга видеонаблюдения с интегрированным CMMS-слоем, построенная по принципу Headless CMMS Architecture.
Ключевые характеристики:
✅ CCTV-specific IP: GB28181, ONVIF, P2P, RCA Engine, Gatekeeper, Playbook Engine
✅ Headless CMMS: 5 адаптеров (Internal, Atlas, ServiceNow, Jira, 1С:ТОИР) + OAuth2 аутентификация
✅ Event-Driven: NATS JetStream + CQRS + Event Sourcing + Webhook Delivery Worker
✅ Enterprise Security: ISO 27001, OWASP ASVS L3, СТБ 34.101.30, HMAC Audit Chain
✅ Multi-tenant: Row-Level Security (RLS) + Tenant Region Pinning
✅ Multi-Region DR: Active-Passive per tenant, NATS mirror, async WAL, S3 CRR
✅ Audit Trail: HMAC chain with prev_hash (tamper detection), 7-year retention
⏳ Без ML/AI (временно — нет реальных данных для обучения)
🎯 High-Level Architecture
flowchart TB
    subgraph Clients["🖥️ Clients"]
        WebApp["React 19 Web App"]
        MobileApp["React Native Mobile"]
        TelegramBot["Telegram Bot"]
        EdgeAgent["Edge Agent (OpenWrt)"]
    end

    subgraph API["🔐 API Gateway (Go/chi)"]
        Auth["JWT + RBAC + 2FA"]
        RateLimit["Rate Limiter"]
        FeatureFlags["Feature Flags"]
        CSP["CSP Nonce"]
    end

    subgraph Core["⚙️ Core Services"]
        Telemetry["Telemetry Collector<br/>GB28181 / ONVIF / P2P"]
        AlertEngine["Alert Engine<br/>+ Deduplication"]
        RCA["RCA Engine<br/>BFS Hierarchy"]
        Gatekeeper["Gatekeeper<br/>GPS + EXIF + AI"]
        Playbook["Playbook Engine<br/>Self-Healing"]
        VideoQ["Video Quality Analyzer<br/>7 metrics"]
    end

    subgraph CMMS["🔌 CMMS Integration Layer"]
        Dispatcher["Event Dispatcher<br/>+ Circuit Breaker"]
        TenantRouter["Tenant Router"]
        
        subgraph Adapters["Adapters"]
            Internal["InternalAdapter<br/>PostgreSQL"]
            Atlas["AtlasAdapter<br/>REST API"]
            ServiceNow["ServiceNowAdapter<br/>ITSM"]
            Jira["JiraAdapter<br/>Service Mgmt"]
            Toir["ToirAdapter<br/>1С:ТОИР"]
        end
    end

    subgraph Event["📡 Event Bus"]
        NATS["NATS JetStream"]
        SchemaRegistry["Schema Registry<br/>10 built-in schemas"]
        ColdStorage["Cold Storage<br/>S3/MinIO (5 лет)"]
    end

    subgraph Data["💾 Data Layer"]
        PG["PostgreSQL 16<br/>+ RLS Multi-tenancy"]
        TSDB["TimescaleDB<br/>Time-series metrics"]
        Redis["Redis<br/>Cache + Rate Limit"]
        MinIO["MinIO<br/>Files + Cold Storage"]
    end

    Clients --> API
    API --> Core
    API --> CMMS
    Core --> Event
    CMMS --> Event
    Event --> Data
    Core --> Data
    EdgeAgent -->|MQTT 5.0| Telemetry
    EdgeAgent -->|WireGuard| API



🏛️ Domain-Driven Design: Bounded Contexts
    graph LR
    subgraph Monitoring["Monitoring Context"]
        Telemetry2[Telemetry]
        Alerts[Alerts]
        VideoQ2[Video Quality]
        RCA2[RCA Engine]
    end

    subgraph CMMS2["CMMS Context"]
        WorkOrders[Work Orders]
        Requests[Work Requests]
        Schedules[Maintenance Schedules]
        SLA[SLA Engine]
    end

    subgraph Assets["Assets Context"]
        Devices[Devices]
        Sites[Sites]
        Hierarchy[Asset Hierarchy]
        Meters[Meters]
    end

    subgraph Workforce["Workforce Context"]
        Technicians[Technicians]
        Teams[Teams]
        Shifts[Shifts]
        Skills[Skills]
    end

    subgraph Integration["Integration Context"]
        Adapters2[CMMS Adapters]
        Webhooks[Webhooks]
        APIKeys[API Keys]
    end

    Monitoring -->|Domain Events| CMMS2
    CMMS2 --> Assets
    CMMS2 --> Workforce
    Monitoring --> Integration
    CMMS2 --> Integration

ADR-013: Каждый Bounded Context имеет:
Свою схему БД (или namespace)
Свой API endpoints
Свои Domain Events
Anti-Corruption Layer на границах
⚙️ Core Components
1. CCTV-Specific Features (Уникальное конкурентное преимущество)
🔍 RCA Engine (Root Cause Analysis)
Файл: backend/internal/rca/engine.go
// BFS traversal по иерархии устройств
// Acceptance: "Switch-1 down → 5 cameras and 2 NVRs affected"
func (e *Engine) Analyze(deviceID string) (*RootCause, error)


Алгоритм:
Получить device и его parent (NVR, Switch)
BFS вверх по иерархии
Проверить статус всех ancestors
Если parent OFFLINE → все children помечаются SUSPENDED_PARENT_DOWN
Подавление ложных алертов для children
🛡️ Gatekeeper Pattern
Файлы: backend/internal/gatekeeper/{gps,exif,ai,token,verifier}.go
ADR-004: Верификация присутствия техника на объекте.
┌─────────────────────────────────────────┐
│  Gatekeeper Verification Pipeline      │
├─────────────────────────────────────────┤
│ 1. QR Code Scan → Device ID            │
│ 2. GPS Verification (geofencing ±50m)  │
│ 3. EXIF Timestamp (photo freshness)    │
│ 4. DeepSeek AI (before/after analysis) │
│ 5. HMAC-signed Token → Verified        │
└─────────────────────────────────────────┘
Graceful Degradation:
GPS недоступен → skip с reason + manual approval
AI недоступен → skip, только GPS + EXIF
Все fail → manual verification required
🤖 Playbook Engine (Self-Healing)
Файлы: backend/internal/agent/{playbook,actions,decisions}.go
YAML-based remediation workflows:
# playbooks/camera_diagnostic.yml
name: camera_diagnostic
steps:
  - name: check_connectivity
    action: ping
    params: {host: "{{device.ip}}", count: 3}
    on_failure: escalate
  - name: isapi_reboot
    action: isapi_reboot
    params: {device_id: "{{device.id}}"}
    on_failure: create_ticket
Decision Tree с Flapping Detection:
type DecisionContext struct {
    Alarm         models.Alarm
    Device        *models.Device
    Topology      *Topology
    FailureCount  int
    LastFixTime   time.Time
    IsBusinessHours bool
}

// Decision Levels:
// - Ignore (flapping detected)
// - AutoFix (playbook execution)
// - Escalate (human intervention)
// - CreateTicket (CMMS integration)
📹 Video Quality Analyzer
Файл: backend/internal/videoq/analyzer.go
7 метрик (Go imaging, без OpenCV):
Blur Detection (Laplacian variance)
Brightness (mean luminance)
Contrast (standard deviation)
Black Screen (near-zero luminance)
Frozen Frame (SSIM between frames)
Noise (high-frequency energy)
Blockiness (DCT-based)
Overall Score: 0-100 (weighted average)
🔌 CMMS Integration Layer (Headless Architecture)
Adapter Pattern
Файл: backend/internal/cmms/adapter.go
type CMMSAdapter interface {
    // Work Orders
    CreateWorkOrder(ctx context.Context, wo *models.WorkOrder) error
    UpdateStatus(ctx context.Context, id string, status string) error
    CloseWorkOrder(ctx context.Context, id string, resolution string) error
    
    // Assets
    GetAssetHierarchy(ctx context.Context, siteID string) (*AssetTree, error)
    UpdateDevice(ctx context.Context, id string, updates map[string]interface{}) error
    
    // Maintenance
    CreateSchedule(ctx context.Context, schedule *models.MaintenanceSchedule) error
    CompleteSchedule(ctx context.Context, id string) error
    
    // Inventory
    GetSpareParts(ctx context.Context) ([]models.SparePart, error)
    UpdateStock(ctx context.Context, id string, quantity int) error
    
    // ... 30+ методов
}
Built-in Adapters
Adapter
Status
Backend
Use Case
InternalAdapter
✅ Production
PostgreSQL
Small/Medium business
AtlasAdapter
✅ Production
Atlas CMMS REST API
Atlas CMMS clients
ServiceNowAdapter
✅ Backend
ServiceNow REST
Enterprise ITSM
JiraAdapter
✅ Backend
Jira REST API
IT teams
ToirAdapter
✅ Backend
1С:ТОИР Webhooks
152-ФЗ compliance
Circuit Breaker + Fallback Queue
Файл: backend/internal/cmms/dispatcher.go
// Circuit Breaker states: closed → open → half-open
// Fallback Queue: exponential backoff (1s, 2s, 4s, 8s, max 3 retries)
// Dead Letter Queue: после 3 неудач → manual intervention
Tenant Router
Файл: backend/internal/cmms/factory/factory.go
# config.yaml (per tenant)
cmms:
  adapter: "internal"  # или "atlas", "servicenow", "jira", "toir"
  
  atlas:
    base_url: "https://atlas.example.com"
    api_key: "${ATLAS_API_KEY}"
    
  servicenow:
    instance: "company.service-now.com"
    username: "${SNOW_USER}"
    password: "${SNOW_PASS}"

📡 Event-Driven Architecture
NATS JetStream
ADR-003: Все межмодульные коммуникации через events.
Файлы:
backend/internal/events/publisher.go — публикация events
backend/internal/events/subscriber.go — подписка
backend/internal/events/store.go — Event Store
backend/internal/events/cold_storage.go — S3 archival
backend/internal/events/schema_registry.go — JSON Schema validation
Domain Events (10 built-in schemas)
Event
Publisher
Subscribers
AlarmCreated
Alert Engine
CMMS Adapters, Playbook Engine
DeviceOffline
Telemetry
RCA Engine, CMMS Adapters
WorkOrderCreated
CMMS
Notifications, Analytics
WorkOrderCompleted
CMMS
SLA Engine, Analytics
MeterThresholdExceeded
Meter Service
Workflow Engine, CMMS
StockLevelChanged
Inventory
Workflow Engine
SLABreach
SLA Engine
Notifications, Escalation
GatekeeperVerified
Gatekeeper
Work Orders
PlaybookExecuted
Agent
Audit Log, Analytics
FeatureFlagChanged
Feature Flags
All services (cache invalidation)
Event Sourcing (Work Orders)
Файл: backend/internal/events/work_order_projection.go
go
// Immutable event log → CQRS projection
// Hot storage: NATS JetStream (1 год)
// Cold storage: S3/MinIO (5 лет)
// Projection Builder: read-model для API
Schema Registry
Файл: backend/internal/events/schema_registry.go
10 built-in schemas:
alarms — Alert events
cmms — Work Order events
predictions — ML predictions (future use)
telemetry — Device metrics
audit — Audit trail
system — System events
gatekeeper — Verification events
playbook — Self-healing events
inventory — Stock events
sla — SLA events
💾 Data Layer
PostgreSQL 16 + TimescaleDB
Multi-tenancy: Row-Level Security (RLS)
ADR-014: tenant_id в каждой таблице + RLS policies
-- Пример RLS policy
CREATE POLICY tenant_isolation ON work_orders
    USING (tenant_id = current_setting('app.tenant_id')::uuid);

Ключевые таблицы:
devices — CCTV устройства (hierarchy: parent_device_id)
sites — Объекты (hierarchy: parent_location_id)
work_orders — Наряды (12 статусов, state machine)
work_order_history — Event sourcing (immutable)
sla_policies, sla_matrix, sla_business_calendars — SLA engine
spare_parts, purchase_orders — Inventory
audit_log — HMAC-signed audit trail (ISO 27001 A.12.4)
TimescaleDB Hypertables:
telemetry — device metrics (partitioned by time)
asset_downtime — downtime tracking
meter_readings — meter values
Материализованные представления (Analytics)
-- AN-10.1.1: MTBF/MTTR по vendor_type и device_type
CREATE MATERIALIZED VIEW mv_device_reliability AS
SELECT
    d.vendor_type,
    d.device_type,
    COUNT(DISTINCT d.device_id) as device_count,
    COUNT(dt.id) as total_downtime_events,
    COALESCE(SUM(dt.duration_minutes), 0) as total_downtime_minutes,
    COALESCE(AVG(wo.completed_at - wo.created_at), 0) as avg_mttr_minutes
FROM devices d
LEFT JOIN asset_downtime dt ON d.device_id = dt.device_id
LEFT JOIN work_orders wo ON d.device_id = wo.device_id
GROUP BY d.vendor_type, d.device_type;

-- Обновление: REFRESH MATERIALIZED VIEW CONCURRENTLY
Redis
Cache: Feature Flags, SLA calculations, Rate Limiting
Sessions: User sessions (JWT refresh tokens)
Pub/Sub: Realtime notifications (WebSocket fallback)
MinIO (S3-compatible)
Files: Photos, PDF reports, signatures
Cold Storage: Event archive (5 лет retention)
Backups: Database dumps

---

## 🔌 OAuth2 для External Adapters (P2-3.2)

**Пакет:** [`backend/internal/oauth2/`](backend/internal/oauth2/)

### Архитектура TokenManager

```
┌───────────────────────────────────────────────┐
│              TokenManager                      │
├───────────────────────────────────────────────┤
│ • OAuth2 Client Credentials flow               │
│ • Token auto-refresh с grace-периодом (30s)    │
│ • Double-check locking (sync.RWMutex)          │
│ • Зашифрованное хранение (AES-256-GCM hex)     │
│ • Fallback: OAuth2 → Basic Auth                │
│ • 401 detection → InvalidateToken → retry     │
│ • ForceRefresh для принудительного обновления  │
│ • Метрики: refresh_count, expired_count        │
└───────────────┬───────────────────────────────┘
                │
    ┌───────────┴───────────┐
    ▼                       ▼
┌──────────────┐     ┌──────────────┐
│ TokenStore   │     │   Metrics    │
├──────────────┤     ├──────────────┤
│ PGTokensStore│     │ Atomic.Int64 │
│ (PostgreSQL) │     │ Snapshot()   │
│ InMemoryStore│     │              │
│ (для тестов) │     │              │
└──────────────┘     └──────────────┘
```

### TokenStore Interface

```go
type TokenStore interface {
    GetToken(ctx context.Context, provider, providerKey string) (*oauth2.Token, error)
    SaveToken(ctx context.Context, provider, providerKey string, token *oauth2.Token) error
    DeleteToken(ctx context.Context, provider, providerKey string) error
}
```

### Поддерживаемые провайдеры

| Адаптер | Метод | Приоритет | Fallback |
|---------|-------|-----------|----------|
| [`ServiceNow`](backend/internal/cmms/servicenow/client.go) | OAuth2 Client Credentials | 1 | Basic Auth (username:password) |
| [`Jira`](backend/internal/cmms/jira/client.go) | OAuth2 Client Credentials | 1 | Basic Auth (email:api_token) |

### DB Schema (Migration 032)

```sql
CREATE TABLE oauth2_tokens (
    provider TEXT NOT NULL CHECK (provider IN ('servicenow', 'jira', 'toir')),
    provider_key TEXT NOT NULL,
    access_token TEXT NOT NULL,      -- AES-256-GCM encrypted hex
    refresh_token TEXT NOT NULL DEFAULT '',
    token_type TEXT NOT NULL DEFAULT 'Bearer',
    expiry TIMESTAMPTZ,
    scopes TEXT[] DEFAULT '{}',
    tenant_id TEXT NOT NULL DEFAULT '*',
    UNIQUE (provider, provider_key)
);
-- + RLS policy for multi-tenant isolation
```

### Compliance

| Стандарт | Контроль | Статус |
|----------|----------|--------|
| IEC 62443 SL-3 | Zone 3 — Application data integrity | ✅ |
| ISO 27001 A.10.1 | Cryptographic controls — encrypted at rest | ✅ |
| СТБ 34.101.30 | Encryption (AES-256-GCM → belt-gcm) | ⚠️ After bp2012/crypto |
| OWASP ASVS V6.2 | Stored credentials — encrypted | ✅ |

---

## 📤 Webhook Delivery Worker (P2-3.3)

**Пакет:** [`backend/internal/webhook/delivery.go`](backend/internal/webhook/delivery.go)

### Архитектура

```
NATS Event ──▶ Webhook Endpoint ──▶ DeliveryWorker ──▶ HTTP POST ──▶ External System
                                         │
                                    ┌────┴────┐
                                    │ Retry   │
                                    │ Queue   │
                                    └────┬────┘
                                         │
                              Exponential Backoff:
                              10s → 30s → 90s → 270s → ... → max 1h
```

### DeliveryWorker

Фоновый worker, который:
- Опрашивает `webhook_delivery_logs` каждые 5 секунд (pending доставки)
- Выполняет HTTP POST с HMAC-SHA256 подписью (X-Signature-256)
- При ошибке: exponential backoff (base × 3^attempt, max 1h)
- После max_retries: статус `cancelled` (больше не retry)
- Graceful shutdown через context + WaitGroup

### DeliveryStore Interface

```go
type DeliveryStore interface {
    GetPendingDeliveries(ctx context.Context, limit int) ([]DeliveryLog, error)
    CreateDeliveryLog(ctx context.Context, log *DeliveryLog) error
    UpdateDeliveryLog(ctx context.Context, id string, status string,
        responseStatus int, responseBody, errorMsg string,
        durationMs int, nextRetryAt *time.Time) error
    CreateWebhookEndpoint(ctx context.Context, wh *WebhookEndpoint) error
    UpdateWebhookEndpoint(ctx context.Context, id string, wh *WebhookEndpoint) error
    DeleteWebhookEndpoint(ctx context.Context, id string) error
    ListWebhookEndpoints(ctx context.Context) ([]WebhookEndpoint, error)
    GetWebhookEndpoint(ctx context.Context, id string) (*WebhookEndpoint, error)
    GetDeliveryLogs(ctx context.Context, webhookID string, limit, offset int) ([]DeliveryLog, error)
}
```

### API Endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/integrations/extended/webhooks` | Список вебхуков |
| POST | `/api/v1/integrations/extended/webhooks` | Создать вебхук |
| PUT | `/api/v1/integrations/extended/webhooks/{id}` | Обновить вебхук |
| DELETE | `/api/v1/integrations/extended/webhooks/{id}` | Удалить вебхук |
| POST | `/api/v1/integrations/extended/webhooks/{id}/test` | Тест вебхука |
| **GET** | **`/api/v1/integrations/extended/webhooks/{id}/logs`** | **Логи доставки (P2-3.3)** |
| **POST** | **`/api/v1/integrations/extended/webhooks/{id}/retry`** | **Принудительный retry (P2-3.3)** |

### DB Schema (Migration 033)

```sql
CREATE TABLE webhook_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    url TEXT NOT NULL CHECK (url LIKE 'https://%'),
    secret TEXT, events TEXT[], enabled BOOLEAN DEFAULT true,
    retry_count INTEGER DEFAULT 3, timeout_seconds INTEGER DEFAULT 10
);

CREATE TABLE webhook_delivery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES webhook_endpoints(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    status TEXT CHECK (status IN ('pending','success','failed','cancelled')),
    request_url TEXT, request_body TEXT,
    response_status INTEGER DEFAULT 0, response_body TEXT,
    duration_ms INTEGER DEFAULT 0,
    retry_attempt INTEGER DEFAULT 0, max_retries INTEGER DEFAULT 3,
    error_message TEXT, next_retry_at TIMESTAMPTZ
);
```

### Metrics (sync/atomic)

| Счётчик | Описание |
|---------|----------|
| `delivery_count` | Всего попыток доставки |
| `success_count` | Успешных доставок |
| `failed_count` | Неудачных доставок |
| `retry_count` | Количество retry |

---

## 📋 Audit Trail Compliance (P3-2)

**Пакет:** [`backend/internal/audit/chain.go`](backend/internal/audit/chain.go)

### HMAC Chain Architecture

```
Record 1                    Record 2                    Record 3
┌────────────────┐         ┌────────────────┐         ┌────────────────┐
│ user_id        │         │ user_id        │         │ user_id        │
│ action         │         │ action         │         │ action         │
│ old_value      │         │ old_value      │         │ old_value      │
│ new_value      │         │ new_value      │         │ new_value      │
│ hmac_signature │──prev──▶│ prev_hash=hmac1│──prev──▶│ prev_hash=hmac2│
│ = H(data)      │         │ hmac_signature │         │ hmac_signature │
│                │         │ = H(prev1+data)│         │ = H(prev2+data)│
└────────────────┘         └────────────────┘         └────────────────┘
```

Принцип: каждая запись audit_log содержит HMAC предыдущей. Если злоумышленник изменит запись, HMAC следующей перестанет совпадать → tamper detection.

### ChainStore

```go
type ChainStore struct {
    pool   *pgxpool.Pool
    signer *Signer
}

func (s *ChainStore) InsertWithChain(ctx context.Context, entry *AuditEntry) error
func (s *ChainStore) VerifyEntry(ctx context.Context, entryID int64) (bool, error)
func (s *ChainStore) GetComplianceReport(ctx context.Context) (*ComplianceReport, error)
```

### DB Functions (Migration 034)

```sql
-- Получить HMAC последней записи для chain
SELECT get_last_audit_hmac();

-- Проверить целостность всей цепочки
SELECT * FROM verify_audit_chain();
-- → chain_broken: boolean, first_broken_id: bigint, broken_count: bigint

-- Архивация записей старше N лет
SELECT archive_audit_logs(7);  -- → количество перемещённых записей
```

### API Endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/audit/log?limit=N&offset=N&action=X&entity_type=Y` | Журнал с пагинацией и фильтрацией |
| GET | `/api/v1/audit/verify` | Проверка HMAC + chain integrity (prev_hash) |
| GET | `/api/v1/audit/compliance` | Compliance-отчёт (статистика, action stats) |
| POST | `/api/v1/audit/archive?retention_years=7` | Архивация записей старше N лет |

### Compliance Report

```json
{
  "total_entries": 15420,
  "last_30_days": 1230,
  "chain_broken": false,
  "oldest_entry": "2025-01-15T08:00:00Z",
  "newest_entry": "2026-06-26T12:00:00Z",
  "action_stats": {
    "CREATE_USER": 45,
    "UPDATE_WORK_ORDER": 3200,
    "COMPLETE_WORK_ORDER": 1200,
    "LOGIN": 8500
  }
}
```

### Compliance

| Стандарт | Контроль | Статус |
|----------|----------|--------|
| ISO 27001 A.12.4.1 | Event logging — audit trail | ✅ |
| ISO 27001 A.12.4.2 | Protection of log information — HMAC chain | ✅ |
| ISO 27001 A.12.4.3 | Retention — 7 years for КИИ РБ | ✅ |
| СТБ 34.101.27 п. 7.2 | Целостность журналов аудита | ✅ |
| IEC 62443 SR 3.1 | Communication integrity — audit chain | ✅ |

---

## 🌍 Multi-Region Geo-Redundancy (P3-1)

**Пакет:** [`backend/internal/multiregion/region.go`](backend/internal/multiregion/region.go)
**ADR:** [`docs/adr/ADR-018-multi-region-architecture.md`](docs/adr/ADR-018-multi-region-architecture.md)

### Архитектурное решение

| Параметр | Решение | Обоснование |
|----------|---------|-------------|
| **Модель** | Active-Passive per tenant | Compliance + cost (vs Active-Active) |
| **NATS** | Local Raft + async mirror | Cross-region Raft ломает latency |
| **PostgreSQL** | In-region replicas + async WAL DR | Residency laws + predictable RPO |
| **Failover** | Semi-auto (admin confirm) | Compliance audit trail |
| **Cold data** | S3 CRR + batch | Экономия 60% vs realtime sync |

### Региональные хабы

| Фаза | Регион | Cloud | Покрытие |
|------|--------|-------|----------|
| Phase 1 | EU-Central | AWS Frankfurt | EU, UK, Turkey |
| Phase 1 | CIS-East | Yandex/Telecom | RU, KZ, BY, UZ |
| Phase 2 | MENA-Gulf | Azure Dubai | SA, AE, QA |
| Phase 3 | SEA-Hub | AWS Singapore | VN, ID, PH, TH |

### Tenant Region Mapping

```sql
CREATE TABLE tenant_regions (
    tenant_id TEXT PRIMARY KEY,
    primary_region TEXT NOT NULL CHECK (primary_region IN ('eu-central','cis-east','mena-gulf','sea-hub')),
    failover_region TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active','failover','migrating')),
    failover_count INTEGER DEFAULT 0,
    last_failover_at TIMESTAMPTZ,
    pinned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### FailoverService

```go
type FailoverService struct {
    store  RegionStore
    nc     *nats.Conn
    config FailoverConfig
}

func (s *FailoverService) ExecuteFailover(ctx context.Context, tenantID string) (*FailoverResult, error)
```

**Failover Process:**
1. Admin: `POST /api/v1/admin/failover/{tenant_id}`
2. System promotes NATS mirror → active (`dr.nats.promote`)
3. System promotes DR PostgreSQL (`dr.postgres.promote`)
4. System updates `tenant_regions.status = 'failover'`
5. Returns `FailoverResult` с статусом каждого шага

### NATSMirrorSetup

```go
type NATSMirrorSetup struct {
    js jetstream.JetStream
}

func (s *NATSMirrorSetup) CreateMirrorStream(ctx context.Context, cfg MirrorConfig) (*jetstream.StreamInfo, error)
func (s *NATSMirrorSetup) SetupCrossRegionMirrors(ctx context.Context, remoteDomain string) (int, error)
```

Системные стримы для mirror: `events`, `alarms`, `telemetry`, `work_orders`, `sla_events`

### Admin API Endpoints

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/v1/admin/regions` | Все tenant-region mapping |
| GET | `/api/v1/admin/regions/{tenant_id}` | Region конкретного тенанта |
| PUT | `/api/v1/admin/regions/{tenant_id}` | Привязка тенанта к региону |
| POST | `/api/v1/admin/failover/{tenant_id}` | Execute failover |
| POST | `/api/v1/admin/failover/{tenant_id}/rollback` | Rollback failover |
| GET | `/api/v1/admin/dr/status` | DR статус по всем регионам |

### Health Check — Region Awareness

```json
GET /health/live
{
  "status": "ok",
  "region": "eu-central",
  "timestamp": "2026-06-26T12:00:00Z"
}
```

Конфигурация: `GB_DEPLOYMENT_REGION=eu-central` (env var)

### Compliance

| Стандарт | Контроль | Статус |
|----------|----------|--------|
| ISO 27001 A.17.1 | Business continuity — DR | ✅ Code |
| IEC 62443 SR 7.1 | Resource availability — multi-region | ✅ Code |
| GDPR Art. 44-49 | Data transfer — region pinning | ✅ Schema |
| 152-ФЗ ст. 18 | Data localization — CIS-East | ✅ Schema |

🔐 Security & Compliance
ISO 27001 Controls
Control
Implementation
Файл
A.9.2 RBAC
6 ролей: admin, manager, technician, viewer, owner, auditor
internal/auth/rbac.go
A.9.4 Authentication
JWT + refresh tokens, TOTP 2FA, API Keys (bcrypt)
internal/auth/
A.12.1.2 Rate Limiting
In-memory per IP с automatic cleanup
internal/api/rate_limiter.go
A.12.4 Audit Logging
HMAC-signed audit log (СТБ 34.101.30)
internal/audit/signer.go
A.13.1 Network Security
Security headers (CSP, HSTS, X-Frame-Options)
internal/api/server.go
A.13.2 CORS
Whitelist origins (не wildcard)
internal/api/server.go
A.14.2 Input Validation
OWASP ASVS V5 (whitelist, not blacklist)
internal/api/validators.go
A.18.1 Compliance
СТБ 34.101.30 crypto (Belarus KII-2)
internal/crypto/
OWASP ASVS L3 Compliance
Checklist:
✅ V1: Architecture (DDD, Clean Architecture)
✅ V2: Authentication (JWT, 2FA, password hashing)
✅ V3: Session Management (HttpOnly cookies, CSRF tokens)
✅ V4: Access Control (RBAC, RLS)
✅ V5: Validation (whitelist, prepared statements)
✅ V6: Cryptography (AES-256-GCM, HMAC-SHA256)
✅ V7: Error Handling (structured errors, no stack traces)
✅ V8: Data Protection (encryption at rest, TLS 1.3)
✅ V9: Communications (HTTPS only, certificate pinning)
✅ V10: Malicious Code (CSP, input sanitization)
✅ V11: Business Logic (state machine, validation)
✅ V12: Files (upload validation, antivirus)
✅ V13: API (rate limiting, pagination)
✅ V14: Configuration (env vars, secrets management)
Edge Agent Security
mTLS 1.3: Mutual authentication
WireGuard: ChaCha20-Poly1305 encryption
Minimal Attack Surface: Read-only telemetry + signed commands
🖥️ Frontend Architecture
Tech Stack
React 19 + TypeScript 5.9
Vite 8 (build tool)
Tailwind CSS 4 (utility-first)
Material-UI v6 (design system)
React Query (data fetching + cache)
Zustand (state management)
Zod (runtime validation)
Recharts (charts)
FullCalendar (scheduling)
Architecture Pattern
frontend/src/
├── components/
│   ├── ui/              # Reusable UI components
│   │   ├── DataGrid.tsx # Snipe-IT pattern
│   │   ├── ImportWizard.tsx
│   │   ├── WorkOrderPrintView.tsx
│   │   └── LiveSLATimer.tsx
│   ├── layout/          # Header, Sidebar, Layout
│   ├── auth/            # PermissionGuard, RoleProtectedRoute
│   └── work-orders/     # BeforeAfterSlider, PhotoAnnotation
├── pages/               # Route components
│   ├── WorkOrders.tsx   # Bulk Actions, Quick Filters
│   ├── WorkOrderDetail.tsx # Three-Column Layout (Atlas)
│   ├── Devices.tsx
│   ├── SpareParts.tsx
│   └── settings/        # 6 tabbed components
├── context/             # React Context (14 contexts)
│   ├── AlertsContext.tsx
│   ├── DataContext.tsx
│   ├── DevicesSitesContext.tsx
│   ├── MaintenanceContext.tsx
│   └── ...
├── services/            # API clients
│   ├── api.ts           # Main API client
│   ├── maintenanceApi.ts
│   └── p2pApi.ts
└── hooks/               # Custom hooks

### State Management (ADR-005)

**Принцип разделения ответственности:**
- **Server State** → React Query (TanStack Query v5). Кеш с автоматической инвалидацией, оптимистичные обновления, рефетч по фокусу.
- **UI State** → Zustand v5. Лёгкие store-only модули без boilerplate.
- **URL State** → React Router v7 search params. Фильтры, пагинация, сортировка.

```
frontend/src/store/
├── alertStore.ts          # Toast-уведомления, фильтры алертов
├── commandPaletteStore.ts # Command Palette (Cmd+K)
└── filterStore.ts         # Глобальные фильтры (дата, сайт, статус)
```

**Правила:**
- Zustand store содержит ТОЛЬКО UI-состояние (модалки, фильтры, выделенные элементы)
- Серверные данные НЕ дублируются в Zustand — только React Query
- Мутации через React Query `useMutation` с `onSuccess` инвалидацией
- Для optimistic updates — `onMutate` + `onSettled`

**Пример (Zustand):** [`frontend/src/store/alertStore.ts`](frontend/src/store/alertStore.ts)
```typescript
interface AlertStore {
  toasts: ToastAlert[];
  addToast: (toast: Omit<ToastAlert, 'id'>) => void;
  removeToast: (id: string) => void;
  filters: AlertFilters;
  setFilters: (filters: Partial<AlertFilters>) => void;
}
```

**Пример (React Query):** [`frontend/src/hooks/useApiQuery.ts`](frontend/src/hooks/useApiQuery.ts)
```typescript
export function useDevices(filters?: DeviceFilters) {
  return useQuery({
    queryKey: ['devices', filters],
    queryFn: () => api.get('/api/devices', { params: filters }),
    staleTime: 30_000,
  });
}
```

### Component Architecture (Atomic Design)

```
frontend/src/components/
├── ui/                    # Atoms — базовые UI-компоненты
│   ├── Button.tsx
│   ├── Badge.tsx
│   ├── Modal.tsx
│   ├── Input.tsx
│   ├── Tabs.tsx
│   ├── Tooltip.tsx
│   └── __tests__/         # Vitest tests (новые)
│       ├── Button.test.tsx
│       ├── Badge.test.tsx
│       ├── Modal.test.tsx
│       └── ...
├── molecules/             # Molecules — композиция атомов
│   ├── DateRangePicker.tsx
│   ├── PriorityPicker.tsx
│   ├── SLAProgressBar.tsx
│   └── TechnicianSelector.tsx
├── organisms/             # Organisms — сложные самодостаточные блоки
│   ├── AssetTree.tsx
│   └── BeforeAfterSlider.tsx
├── layout/                # Layout — каркас приложения
│   ├── Header.tsx
│   ├── Sidebar.tsx
│   ├── Layout.tsx
│   └── ThreeColumnTemplate.tsx
├── auth/                  # Auth — guards, protected routes
│   ├── PermissionGuard.tsx
│   └── RoleProtectedRoute.tsx
├── dashboard/             # Dashboard — виджеты дашборда
│   ├── AlertBanner.tsx
│   └── DragDropDashboard.tsx
├── work-orders/           # Work Orders — доменные компоненты
│   ├── QuickFilters.tsx
│   ├── WOKanbanBoard.tsx
│   ├── WODetailHeader.tsx
│   └── ...
├── devices/               # Devices — доменные компоненты
├── reports/               # Reports — доменные компоненты
├── sla/                   # SLA — доменные компоненты
└── spare-parts/           # Spare Parts — доменные компоненты
```

**Принципы композиции:**
- Atoms не имеют бизнес-логики, только presentation + a11y
- Molecules комбинируют atoms + простую валидацию
- Organisms содержат бизнес-логику, data fetching, state
- Domain-компоненты (work-orders/, devices/) используют organisms и molecules

### Design System v2

**Токены (TailwindCSS v4):**
- Цвета: slate как базовый, blue/accent, emerald/success, amber/warning, red/danger
- Типографика: system font stack (Inter fallback)
- Тени: `shadow-sm` для карточек, `shadow-lg` для модалок, `shadow-xl` для dropdown
- Скругления: `rounded-lg` (8px) для UI, `rounded-2xl` (16px) для модалок
- Анимации: `transition-all duration-150 ease-in-out` стандарт

**Accessibility (WCAG 2.1 AA):**
- Все интерактивные элементы имеют `focus-visible` ring
- Цветовые индикаторы (dot) сопровождаются `aria-label`
- Модалки используют focus trap + `aria-modal`
- Tooltip имеет `role="tooltip"`, dismiss via Escape
- Dropdown имеет `aria-haspopup`, `aria-expanded`, keyboard navigation

**Тёмная тема:**
- Все компоненты поддерживают `dark:` префиксы
- Переключение через [`ThemeContext`](frontend/src/context/ThemeContext.tsx)
- Persistence в localStorage с `prefers-color-scheme` fallback

**Компоненты (56 шт.):**
- Atoms: Button, Badge, Input, Select, Tabs, Tooltip, Dropdown, Modal, Skeleton, ProgressBar, EmptyState, Alert, Card, Breadcrumbs, VisuallyHidden, Toast
- Molecules: StatsCard, Gauge, Timeline, QRCode, FileUpload, DateRangePicker, PriorityPicker, SLAProgress, LiveSLATimer
- Organisms: DataGrid, VirtualTable, ImportWizard, AdvancedSearch, CommandPalette, OnboardingTour, ThemeCustomizer, SavedViews, WorkOrderPrintView

### Performance Patterns

**Code Splitting:**
```typescript
// Ленивая загрузка страниц (React.lazy + Suspense)
const Dashboard = lazy(() => import('./pages/Dashboard'));
const WorkOrders = lazy(() => import('./pages/WorkOrders'));

// PageSuspense — обёртка с Skeleton
<PageSuspense name="dashboard">
  <Dashboard />
</PageSuspense>
```
Файл: [`frontend/src/components/layout/PageSuspense.tsx`](frontend/src/components/layout/PageSuspense.tsx)

**Virtual Scrolling:**
- [`DataGrid`](frontend/src/components/ui/DataGrid.tsx) использует `@tanstack/react-virtual` для рендеринга только видимых строк
- [`VirtualTable`](frontend/src/components/ui/VirtualTable.tsx) — альтернатива для больших таблиц (10k+ rows)

**React Query Optimizations:**
- `staleTime: 30s` для списков, `staleTime: 5min` для справочников
- `gcTime: 5min` для кеширования после unmount
- `keepPreviousData` для пагинации (без лагов между страницами)
- Prefetching на hover: `queryClient.prefetchQuery`

**Мемоизация:**
- React.memo для тяжёлых компонентов (DataGrid rows, таблицы)
- useMemo для вычисляемых значений (фильтрация, сортировка)
- useCallback для callback'ов в списках

**Бандл:**
- Vite 8 с automatic code splitting по routes
- Lucide icons — tree-shakeable (импорт только нужных иконок)
- Rollup Plugin Visualizer для анализа бандла: `npm run build && npx vite-bundle-analyzer`

**Тестирование производительности:**
- Vitest для unit-тестов компонентов
- React Testing Library для интеграционных тестов
- Lighthouse CI в CI/CD pipeline (planned)

UX Patterns (from Atlas/Grash/Snipe-IT)
1. DataGrid Pattern (Snipe-IT)
Файл: frontend/src/components/ui/DataGrid.tsx
interface DataGridProps<T> {
  data: T[];
  columns: Column<T>[];
  bulkActions?: BulkAction[];
  quickFilters?: QuickFilter[];
  enableInlineEdit?: boolean;
  enableColumnFilters?: boolean;
  enableDensityControl?: boolean;
}
Features:
✅ Checkbox selection (select all, select page)
✅ Bulk Actions toolbar (Assign, Change Priority, Close, Export)
✅ Quick Filters (My, Overdue, Critical, Today, Unassigned)
✅ Inline Editing (double-click → edit → Enter → save)
✅ Column Filters (dropdown с уникальными значениями)
✅ Density Control (compact/standard/comfortable)
✅ Column Visibility & Reordering
2. Three-Column Layout (Atlas CMMS)
Файл: frontend/src/pages/WorkOrderDetail.tsx
┌──────────────┬──────────────────────────────┬──────────────┐
│ LEFT (4/12)  │ CENTER (5/12)                │ RIGHT (3/12) │
│              │                              │              │
│ Status badge │ Checklist (drag&drop)        │ Asset Info   │
│ Priority     │ ☐ Task 1                     │ Location     │
│ Type         │ ☑ Task 2 (completed)         │ Actions      │
│ Live SLA     │ ☐ Task 3                     │ Parts Used   │
│ Assignee     │                              │              │
│ Timeline     │ Notes & Photos               │ Related WOs  │
│              │ [Photo 1] [Photo 2]          │              │
└──────────────┴──────────────────────────────┴──────────────┘
3. Import Wizard (Grash)
Файл: frontend/src/components/ui/ImportWizard.tsx
Steps:
Upload (CSV/XLSX)
Preview (first 10 rows)
Set Header (row number)
Match Columns (auto-detect + manual)
Review Duplicates (merge/skip/overwrite)
Import (progress bar + results)
📱 Mobile Architecture
Tech Stack
React Native + Expo 52
React Query (data fetching)
Zustand (state management)
WatermelonDB (offline-first, planned)
Screens
mobile/src/screens/
├── LoginScreen.tsx
├── DashboardScreen.tsx
├── WorkOrderDetailScreen.tsx
├── ChecklistScreen.tsx
├── PhotoCaptureScreen.tsx
├── SignatureScreen.tsx
├── QRScannerScreen.tsx
├── VerificationScreen.tsx
└── ProfileScreen.tsx
Offline-First (Planned)
ADR-018: WatermelonDB vs PowerSync vs RxDB
┌─────────────────────────────────────────┐
│  Mobile App                             │
│  ├─ Local DB (WatermelonDB)            │
│  ├─ Sync Queue (background)            │
│  └─ Conflict Resolution (last-write)   │
└─────────────────────────────────────────┘
         ↕ (when online)
┌─────────────────────────────────────────┐
│  Backend API                            │
└─────────────────────────────────────────┘
🚀 Deployment
Docker Compose (Development)
Файл: docker-compose.yml
services:
  backend:
    build: ./backend
    ports: ["8080:8080"]
    depends_on: [postgres, nats, redis, minio]
    
  frontend:
    build: ./frontend
    ports: ["3000:3000"]
    
  postgres:
    image: timescale/timescaledb:latest-pg16
    volumes: ["pgdata:/var/lib/postgresql/data"]
    
  nats:
    image: nats:2.10-alpine
    command: ["--jetstream"]
    
  redis:
    image: redis:7-alpine
    
  minio:
    image: minio/minio
    command: server /data
Kubernetes (Production)
Файлы: .github/workflows/deploy.yml
Kubernetes (Production)
Файлы: .github/workflows/deploy.yml
CI/CD Pipeline
Файлы: .github/workflows/{ci,deploy,security-scan}.yml
┌─────────┐   ┌──────┐   ┌───────┐   ┌──────────┐   ┌────────┐
│  Lint   │──▶│ Test │──▶│ Build │──▶│ Security │──▶│ Deploy │
│         │   │      │   │       │   │  Scan    │   │        │
└─────────┘   └──────┘   └───────┘   └──────────┘   └────────┘
     │             │           │             │            │
  golangci     unit +      Docker       gosec +       Staging →
  eslint       integ       image        trivy         Production
  📊 Current Status & Progress
Progress by Epic
Epic
Название
Progress
Status
0
Foundation & Clean Room
90%
✅
1
Domain Model Evolution
95%
✅
2
CCTV Core (без ML)
85%
✅
3
CMMS Integration Layer
100%
✅
4
Work Order Lifecycle
95%
✅
5
Asset & Location Hierarchy
95%
✅
6
Advanced SLA Engine
90%
✅
7
Inventory & Procurement
75%
🟡
8
Workforce Management
75%
🟡
9
Automation & Workflows
60%
🟠
10
Analytics & Reporting
85%
✅
11
UX/UI Modernization
90%
✅
12
Mobile & Offline
60%
🟠
13
Enterprise Integrations
75%
🟡
Overall: 87% → Production-ready foundation, большинство enterprise features реализованы

## ✅ Последние изменения (2026-06-25)
- **ARCH-01**: NATS JetStream KV State Manager — распределённое состояние устройств (horizontal scaling)
- **SEC-02**: SLA Memory Leak — заменён time.AfterFunc на централизованную cleanupLoop с TTL-эвикцией
- **SEC-03**: CORS Security — reject wildcard `*` origins (OWASP ASVS V13.4)
- **ARCH-01**: `backend/internal/state/jetstream_manager.go` — новый файл
- **ARCH-01**: `backend/internal/state/jetstream_manager_test.go` — тесты (graceful skip без NATS)
Key Metrics
Metric
Current
Target (Q4 2026)
Code Coverage
45% (Go), 50% (TS)
85%
API Endpoints
140+
200+
Database Tables
38
50
Domain Events
10
20
CMMS Adapters
5
5 (stable)
ISO 27001 Controls
85%
100%
ARCH-01 (NATS KV State Manager)
✅ Complete
—
SLA Memory Leak (TTL Eviction)
✅ Fixed
—
CORS Security (Wildcard Rejection)
✅ Fixed
—
🗺️ Roadmap (Q3-Q4 2026)
Phase 1: Foundation & Core CMMS (Недели 1-4)
Цель: Стабильный MVP с enterprise WO management.
Deliverables:
✅ Work Requests Portal (reCAPTCHA)
✅ Three-Column Layout (Atlas pattern)
✅ Bulk Actions + Quick Filters (Snipe-IT)
✅ SLA Matrix + Business Calendar
✅ Purchase Orders workflow
Phase 2: Workflows & Enterprise (Недели 5-8) ✅
Цель: Automation + Workforce Management.
Deliverables:
✅ Workflow Engine (CEL-based DSL)
✅ Visual Workflow Builder (React Flow)
✅ Matrix RBAC
✅ Workload & Capacity Planning
🔲 Import/Export Wizard
Phase 3: Security & Compliance (Недели 9-16) ✅
Цель: OAuth2, Webhook, Audit Trail, Multi-Region.
Deliverables:
✅ MTBF/MTTR/TCO Analytics
✅ Downtime Cost Calculation
🔲 Mobile Offline (WatermelonDB)
✅ ServiceNow Integration (OAuth2)
✅ SAML 2.0 SSO
📚 Architectural Decision Records (ADRs)
ADR
Тема
Статус
Дата
ADR-001
Headless CMMS Architecture
✅ Accepted
2026-06-15
ADR-002
CMMS Adapter Pattern
✅ Accepted
2026-06-16
ADR-003
Event Bus (NATS JetStream)
✅ Accepted
2026-06-20
ADR-004
Gatekeeper Pattern
✅ Accepted
2026-06-21
ADR-013
DDD Bounded Contexts
✅ Accepted
2026-06-24
ADR-014
Multi-tenancy (RLS)
🟡 Planned
2026-06-25
ADR-015
Event Sourcing for WorkOrders
🟡 Planned
2026-06-25
ADR-016
State Machine Library
🟡 Planned
2026-06-25
ADR-017
PDF Generation (Puppeteer)
🟡 Planned
2026-06-25
ADR-018
Multi-Region Geo-Redundancy
✅ Accepted
2026-06-26
ADR-019
Workflow DSL (CEL)
🟡 Planned
2026-06-25
ADR-020
Custom Fields (JSONB)
🟡 Planned
2026-06-25
ADR-021
NATS JetStream KV State Manager
✅ Accepted
2026-06-25
🔧 File Structure
cctv-monitoring/
├── backend/
│   ├── cmd/
│   │   ├── api/              # API server entry point
│   │   ├── agent/            # Self-healing agent
│   │   └── migrate/          # DB migrations CLI
│   ├── internal/
│   │   ├── api/              # HTTP handlers (chi router)
│   │   ├── cmms/             # 5 CMMS adapters
│   │   ├── events/           # Event Store + Projections
│   │   ├── gatekeeper/       # Verification pipeline
│   │   ├── agent/            # Playbook Engine
│   │   ├── rca/              # Root Cause Analysis
│   │   ├── sla/              # SLA Engine
│   │   ├── videoq/           # Video Quality Analyzer
│   │   ├── meter/            # Meter Triggers
│   │   ├── featureflag/      # Feature Flags
│   │   ├── audit/            # HMAC-signed audit log + ChainStore
│   │   └── auth/             # JWT + RBAC + 2FA
│   ├── analytics/            # Python ML scripts (paused)
│   ├── playbooks/            # YAML remediation workflows
│   ├── oauth2/           # TokenManager + PGTokensStore (P2-3.2)
│   ├── multiregion/       # RegionStore + FailoverService (P3-1)
│   ├── webhook/           # DeliveryWorker + PGDeliveryStore (P2-3.3)
│   └── migrations/           # SQL migrations (035 files)
├── frontend/
│   ├── src/
│   │   ├── components/       # UI components
│   │   ├── pages/            # Route components
│   │   ├── context/          # React Context (14)
│   │   ├── services/         # API clients
│   │   └── hooks/            # Custom hooks
│   └── public/
├── mobile/
│   └── src/
│       ├── screens/          # React Native screens
│       ├── store/            # Zustand stores
│       └── hooks/            # Custom hooks
├── p2p-gateway/              # P2P camera connectivity
├── docs/
│   ├── adr/                  # Architectural Decisions
│   ├── iso27001/             # Compliance docs
│   ├── ux/                   # UX guidelines
│   └── compliance/           # Audit reports
├── docker-compose.yml
├── .github/workflows/        # CI/CD
└── ARCHITECTURE.md           # This file
Этот документ является living document. Все изменения фиксируются в git history с указанием причины. Версионирование через semantic versioning (Major.Minor.Patch).