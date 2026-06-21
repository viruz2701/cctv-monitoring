# CCTV Intelligence Platform — Architecture Document v5.0

**Дата обновления:** 2026-06-21
**Статус:** Phase 2 in progress (AI Intelligence & Atlas Integration)
**Версия:** 5.0 (post-Phase 1.5)

---

## 1. Executive Summary

CCTV Intelligence Platform — зрелая экосистема для мониторинга CCTV, управления обслуживанием и предиктивной аналитики.

**Текущий стек:**
- **Backend:** Go 1.25 (chi, pgx/v5, gorilla/websocket, telegram-bot-api)
- **Frontend:** React 19, Vite 8, Tailwind 4, TypeScript 5.9, i18next
- **Mobile:** React Native / Expo 52, React Query, Zustand
- **P2P Gateway:** Go 1.25 + бинарные адаптеры (neolink, dh-p2p)
- **Analytics:** Python 3.11 (XGBoost, pandas, psycopg2)
- **Data:** PostgreSQL + TimescaleDB (hypertables)

**Ключевые архитектурные решения (приняты):**
- **ADR-001:** Headless CMMS — CMMS как интерфейс, не как жёсткая привязка
- **ADR-002:** CMMS Adapter Pattern — InternalAdapter + AtlasAdapter + Router
- **ADR-003:** Event Bus — NATS (Phase 1-3), Kafka (Phase 4)
- **ADR-004:** Gatekeeper Pattern — GPS/EXIF/AI верификация закрытия нарядов ✅ **РЕАЛИЗОВАНО**

**Реализовано сверх roadmap:**
- GB/T 28181 SIP-сервер (полный стек: REGISTER, MESSAGE, Catalog, PTZ)
- 7 приватных протоколов CCTV (Dahua, Hisilicon, TVT, Hikvision ISAPI, FTP, SNMP, Syslog)
- Telegram Bot (account linking, 2FA login, alarm notifications)
- Mobile App с offline-first синхронизацией
- P2P Gateway для 4 брендов (Hikvision, Reolink, Dahua, Xiongmai/Jftech)

---

## 2. High-Level Architecture

┌─────────────────────────────────────────────────────────────────────┐
│ CLIENTS LAYER │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Desktop │ │ Mobile App │ │ Telegram Bot │ │
│ │ (React/Vite) │ │ (Expo/RN) │ │ (Commands & │ │
│ │ │ │ │ │ Alerts) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
│ HTTPS / WSS
▼
┌─────────────────────────────────────────────────────────────────────┐
│ API GATEWAY (Go/chi) │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Middleware: Auth(JWT), RBAC, CORS, Logger, Recoverer, │ │
│ │ RateLimiter, SecurityHeaders ✅ │ │
│ │ Handlers: api, mobile, cmms, telegram, apikey, ws, p2p, │ │
│ │ gatekeeper ✅ │ │
│ │ Protocols: SIP/GB28181, Dahua, Hisilicon, TVT, FTP, SNMP │ │
│ └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────┐
│ CORE DOMAIN SERVICES │
│ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Telemetry │ │ CMDB │ │ Gatekeeper │ │
│ │ Collector │ │ Service │ │ Service │ ✅ DONE │
│ │ (RTSP/SNMP/ │ │ (Devices, │ │ (GPS/EXIF/ │ │
│ │ ISAPI/SIP/ │ │ Sites, QR) │ │ AI Verify) │ │
│ │ Dahua/FTP) │ │ │ │ │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
│ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Alarm & │ │ SLA & │ │ AI/ML │ │
│ │ State Mgr │ │ Workload │ │ Service │ │
│ │ (WebSocket) │ │ Manager │ │ (XGBoost) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────┐
│ INTEGRATION LAYER │
│ │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ CMMS Router & Adapter Framework │ │
│ │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │ │
│ │ │ Internal │ │ Atlas │ │ServiceNow│ │ Jira │ │ │
│ │ │ (PgSQL) │ │ Adapter │ │ Adapter │ │ Adapter │ │ │
│ │ │ ✅ │ │ ⚠️ Stub │ │ Phase 3 │ │ Phase 3 │ │ │
│ │ └──────────┘ └──────────┘ └──────────┘ └──────────┘ │ │
│ └────────────────────────────────────────────────────────────┘ │
│ │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ P2P Gateway (Go + binaries) │ │
│ │ Adapters: Hikvision ✅, Reolink ✅, Dahua ✅, Xiongmai ✅ │ │
│ └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────┐
│ DATA LAYER │
│ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────┐ │
│ │ TimescaleDB │ │ PostgreSQL │ │ Redis │ │
│ │ (telemetry, │ │ (CMDB, CMMS, │ │ (Cache, │ │
│ │ alarms, logs, │ │ Users, SLA, │ │ Sessions) │ │
│ │ predictions) │ │ API Keys) │ │ ⚠️ Pending │ │
│ └──────────────────┘ └──────────────────┘ └──────────────┘ │
│ │
│ ┌──────────────────┐ ┌──────────────────┐ │
│ │ Object Storage │ │ Vault │ │
│ │ (Photos, │ │ (Secrets, JWT) │ │
│ │ Reports) │ │ ⚠️ Pending │ │
│ └──────────────────┘ └──────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘

---

## 3. Domain Model (обновлено)

### 3.1 Core Entities

| Entity | Таблица | Статус | Описание |
|--------|---------|--------|----------|
| **Device** | `devices` | ✅ | id, name, type, vendor, site_id, qr_code, GB28181 fields, P2P fields, ONVIF fields, **latitude/longitude/geofence** ✅ |
| **Site** | `sites` | ✅ | Hierarchy (Site → Building → Floor → Rack), GPS, geofence |
| **Alarm** | `alarms` (hypertable) | ✅ | device_id, type, severity, status, image_path |
| **WorkOrder** | `work_orders` | ✅ | type, status, priority, SLA deadline, checklist (JSONB), photos, parts_used, **verification_token** ✅ |
| **SparePart** | `spare_parts` | ✅ | name, sku, stock, min_stock, cost, location |
| **MaintenanceSchedule** | `maintenance_schedules` | ✅ | schedule_type, interval_days, next_due, checklist |
| **Technician** | `users` + columns | ✅ | skills, workload, **push_token (AES-256-GCM)** ✅, certifications |
| **SLAConfig** | `sla_config` | ✅ | priority → response_time, resolution_time |
| **APIKey** | `api_keys` | ✅ | hash (**bcrypt** ✅), **key_prefix** ✅, permissions, expires_at |
| **TechnicianSiteAssignment** | `technician_site_assignments` | ✅ | technician_id, site_id, is_primary |
| **AuditLog** | `audit_log` | ✅ | user_id, action, entity, old/new JSONB |
| **UserSession** | `user_sessions` | ✅ | token_hash, ip, user_agent, expires_at |
| **Prediction** | `predictions` (hypertable) | ✅ | failure_probability, explanation |
| **TelegramLinkToken** | `telegram_link_tokens` | ✅ | token, user_id, expires_at |

### 3.2 CMMS Adapter Interface (ADR-002)

```go
// backend/internal/cmms/adapter.go
type CMMSAdapter interface {
    // Work Orders (8 methods)
    CreateWorkOrder(ctx, wo) error
    GetWorkOrders(ctx, filters) ([]WorkOrder, error)
    // ... AssignWorkOrder, StartWorkOrder, CompleteWorkOrder, CancelWorkOrder
    
    // Spare Parts (7 methods)
    CreateSparePart, GetSpareParts, UpdateSparePart, DeleteSparePart, ...
    
    // Maintenance Schedules (7 methods)
    CreateMaintenanceSchedule, GetDueSchedules, CompleteMaintenanceSchedule, ...
    
    // SLA, Technicians, Reports, Site Assignments, Mobile
    // Total: 33 methods
}
Реализации:
InternalAdapter ✅ — обёртка над db.DB, production-ready
AtlasAdapter ⚠️ — все методы возвращают ErrNotImplemented (задел на Phase 2)
ServiceNowAdapter, JiraAdapter — planned for Phase 3
3.3 Gatekeeper Pattern (ADR-004) — ✅ РЕАЛИЗОВАНО
Mobile App                    Backend                      AI Service
    │                            │                             │
    │  POST /verify              │                             │
    │  {photo, GPS, EXIF}        │                             │
    ├───────────────────────────►│                             │
    │                            │  1. GPS geofence check ✅   │
    │                            │  2. EXIF time/device ✅     │
    │                            │  3. POST /gatekeeper/ai     │
    │                            ├────────────────────────────►│
    │                            │                             │ DeepSeek: before/after
    │                            │◄────────────────────────────┤
    │                            │  4. Generate verify_token ✅│
    │◄───────────────────────────┤                             │
    │  {verification_token}      │                             │
    │                            │                             │
    │  POST /complete            │                             │
    │  {token, notes, photos}    │                             │
    ├───────────────────────────►│                             │
    │                            │  Validate token → complete ✅│
    Реализованные компоненты:
gatekeeper/verifier.go — оркестратор верификации
gatekeeper/gps.go — Haversine distance, geofence validation
gatekeeper/exif.go — EXIF metadata validation (timestamp, GPS match)
gatekeeper/ai.go — DeepSeek Vision API integration (graceful skip if not configured)
gatekeeper/token.go — JWT verification token (10 min TTL)
api/gatekeeper_handler.go — HTTP endpoint
Mobile: VerificationScreen.tsx, GPSStatus.tsx, EXIFStatus.tsx, AIScore.tsx
4. Protocol Architecture (реализовано сверх roadmap)
4.1 GB/T 28181 (China National Standard)
Полная реализация SIP-сервера:
REGISTER — регистрация устройств, NAT traversal
MESSAGE — Keepalive, Alarm, MobilePosition
Catalog — запрос каталога NVR → авто-регистрация child devices
DeviceInfo — авто-запрос manufacturer/model/firmware
PTZ — команды управления (Direction + Zoom)
GB2312/GBK — декодирование китайской кодировки
DeviceID parsing — 20-значный код (type/region/manufacturer/serial)
4.2 Приватные протоколы
Протокол
Порт
Статус
Особенности
Dahua
37777, 37778
✅
Binary header 0x12 0x34, key=value payload
Hisilicon
15002
✅
JSON в бинарных данных, hex→IP конвертация
TVT
15003
✅
XML/JSON fallback, ASCII-поиск
Hikvision ISAPI
HTTP pull
✅
Multipart streaming + Raw TCP fallback
FTP
2121
✅
Приём snapshot, авто-регистрация
SNMP traps
162
✅
v1/v2c, OID-based vendor detection
Syslog
1514 UDP/TCP
✅
Эвристический парсер (Hikvision/Dahua)
4.3 P2P Gateway
Микросервис на Go 1.25 с бинарными адаптерами:
Hikvision — EZVIZ/Hik-Connect cloud proxy
Reolink — neolink (Rust binary)
Dahua — dh-p2p (Python script)
Xiongmai/Jftech — nat traversal через JftechWS API
5. Security Architecture (ISO 27001 Status)
5.1 Реализовано ✅
Control
Реализация
Файл
A.9.1 RBAC
6 ролей: admin, support, owner, manager, technician, viewer
auth/middleware.go
A.9.2 User Registration
CreateUser с хешированием bcrypt
api/server.go
A.9.4 Password Policy
Min 6 chars (basic), 8+symbol (strong)
Settings.tsx
A.10.1 TOTP 2FA
RFC 6238, Google Authenticator
api/server.go, auth/jwt.go
A.10.2 API Keys
bcrypt (cost=12) ✅ с prefix lookup
apikey_handlers.go
A.10.3 Push Tokens
AES-256-GCM шифрование ✅
crypto/aes.go
A.10.4 JWT Secret
env var (panic if missing) ✅
auth/jwt.go
A.12.4 Audit Log
Все CMMS-операции логируются
cmms_handlers.go
A.13.1 TLS
Termination на reverse proxy
infra
A.13.2 Security Headers
CSP, X-Frame-Options, HSTS, Referrer-Policy ✅
server.go
A.13.3 Rate Limiting
In-memory rate limiter на login ✅
server.go
A.14.2 Input Validation
chi URL params, JSON decode
все handlers
5.2 Pending ⚠️
Gap
Severity
Remediation
Redis for sessions/caching
MEDIUM
Добавить Redis для session storage
Vault integration
MEDIUM
HashiCorp Vault для secrets
CORS restriction
MEDIUM
Config-based allowed origins
Audit Log Integrity (HMAC)
MEDIUM
HMAC-подпись для audit-записей
JWT → HttpOnly Cookies
MEDIUM
Перевести на HttpOnly cookies с CSRF
Vulnerability scanning CI/CD
MEDIUM
gosec, trivy, npm audit в GitHub Actions
6. Mobile Architecture
6.1 Screens & Navigation (обновлено)
LoginScreen
    └── MainTabs
        ├── DashboardScreen (WorkOrderCard list)
        │   └── WorkOrderDetailScreen
        │       ├── ChecklistScreen (progress bar)
        │       ├── PhotoCaptureScreen (camera + GPS)
        │       ├── VerificationScreen ✅ (GPS/EXIF/AI checks)
        │       ├── SignatureScreen (accepts verification_token)
        │       └── QRScannerScreen (expo-barcode-scanner)
        └── ProfileScreen (stats, skills, logout)
        6.2 Offline-First Architecture
        ┌─────────────────────────────────────────────┐
│  UI Layer (React Query)                     │
│  ├── useQuery → cached data                 │
│  └── useMutation → optimistic updates       │
├─────────────────────────────────────────────┤
│  State Layer (Zustand)                      │
│  ├── authStore — token, user                │
│  ├── workOrderStore — cached Map            │
│  └── syncStore — offline queue              │
├─────────────────────────────────────────────┤
│  Sync Layer                                 │
│  ├── AppState listener (background→active)  │
│  ├── AsyncStorage persistence               │
│  └── Retry logic (3 attempts, then drop)    │
└─────────────────────────────────────────────┘
7. Data Layer
7.1 TimescaleDB Hypertables
Table
Partition Key
Retention
telemetry
time
30 days
alarms
time
90 days
parsed_logs
time
30 days
predictions
prediction_date
365 days
7.2 PostgreSQL Tables (23 total)
Core: users, sites, devices, tickets, ticket_comments, notifications, reports
CMMS: work_orders, maintenance_schedules, spare_parts, part_usage, sla_config
Auth: api_keys, user_sessions, telegram_link_tokens, telegram_login_codes, password_reset_tokens
Meta: system_settings, audit_log, technician_site_assignments
8. Integration Architecture
8.1 CMMS Router
// backend/internal/cmms/adapter.go
func NewCMMSRouterFromConfig(cfg *config.Config, db *db.DB) *CMMSRouter {
    switch cfg.CMMSAdapter {
    case "atlas":
        return NewCMMSRouter(NewAtlasAdapter(cfg.AtlasURL, cfg.AtlasAPIKey))
    default:
        return NewCMMSRouter(NewInternalAdapter(db))
    }
}
Текущее поведение: Все запросы идут в InternalAdapter (PostgreSQL). AtlasAdapter — stub.
Planned (Phase 2-3): Fallback queue, bi-directional sync, conflict resolution.
8.2 WebSocket Hub

Backend ──alarm──► ws.Hub ──broadcast──► All connected clients
                                             │
                                             ├── Desktop (AlertsContext)
                                             ├── Mobile (future)
                                             └── Telegram Bot (future)
                                             9. Roadmap (Updated)
Phase
Срок
Статус
Ключевые deliverables
Phase 0
Недели 1-2
✅ Done
UX Research, ISO Gap Analysis
Phase 1
Месяцы 1-2
✅ Done
CMMS Router, Maintenance Schedules, Technician Assignments
Phase 1.5
Месяц 3
✅ Done
Gatekeeper ✅, ISO Quick Wins ✅, UX Refresh ✅
Phase 2
Месяцы 4-6
🔄 In Progress
AI Predictive, TCO, Voice-to-Report, Atlas integration
Phase 3
Месяцы 7-9
Pending
ServiceNow/Jira Adapters, Self-Healing, NATS Event Bus
Phase 4
Месяцы 10-15
Pending
SaaS Multi-tenant, AR Remote Expert, ISO Certification
10. Technology Stack (полный)
Layer
Technology
Version
Backend
Go
1.25
Router
chi/v5
5.2.1
Database
pgx/v5
5.10.0
WebSocket
gorilla/websocket
1.5.3
JWT
golang-jwt/v5
5.3.1
TOTP
pquerna/otp
1.5.0
Config
spf13/viper
1.21.0
Logging
slog + lumberjack
—
Frontend
React
19.2.0
Build
Vite
8.0.16
CSS
Tailwind
4.1.18
i18n
i18next
26.3.1
Charts
FullCalendar
6.1.20
Mobile
React Native
0.76.0
Mobile FW
Expo
52.0
State (mobile)
Zustand
5.0
Data (mobile)
React Query
5.60
Analytics
Python
3.11
ML
XGBoost
2.0+
LLM
DeepSeek API
—
11. File Structure (актуальная, ~210 файлов)
смотри /home/viruz/cctv-monitoring/project-structure.txt
