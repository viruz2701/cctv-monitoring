# CCTV Health Monitor — System Architecture

## 🎯 Overview

Enterprise-grade CCTV monitoring platform для health monitoring, predictive maintenance и multi-vendor device management. Поддерживает **15+ вендоров** через прямые протоколы и **P2P/Push модели** для устройств за NAT.

**Текущий статус**: Production-ready MVP с полным управлением пользователями, 10+ протокольными обработчиками, P2P gateway и ML-прогнозированием отказов.

## 🏗️ System Architecture
┌

─────────────────────────────────────────────────────────────────────────────┐
│ FRONTEND (React 19) │
│ Vite 8 + TypeScript 5.9 + Tailwind 4 + i18next (RU/EN) │
│ Context API (domain-separated) + React Router 7 + Lucide Icons │
└────────────────────────────┬────────────────────────────────────────────────┘
│ HTTP/REST + WebSocket (planned)
▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ API GATEWAY (Go 1.25) │
│ chi router + JWT auth + CORS + Rate Limiting (planned) │
│ /api/v1/* endpoints │
└──┬──────────┬──────────┬──────────┬──────────┬─────────────────────────────┘
│ │ │ │ │
▼ ▼ ▼ ▼ ▼
┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────────┐
│Users │ │Device│ │Alarms│ │Reports│ │P2P Gateway│
│CRUD │ │State │ │& Logs│ │& Audit│ │ (Go) │
└──┬───┘ └──┬───┘ └──┬───┘ └──┬───┘ └────┬─────┘
│ │ │ │ │
└─────────┴─────────┴─────────┴────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ DATABASE (PostgreSQL 16 + TimescaleDB) │
│ Hypertables: telemetry, alarms, parsed_logs, predictions │
│ Retention: 30d (telemetry/logs), 90d (alarms), 365d (predictions) │
└─────────────────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ ANALYTICS PIPELINE (Python 3.11) │
│ pandas + XGBoost + scikit-learn + DeepSeek API (optional) │
│ ETL → Feature Engineering → ML Prediction → DB Write │
└─────────────────────────────────────────────────────────────────────────────┘
12345
backend/
├── main.go # Bootstrap, graceful shutdown, Reaper
├── config.yaml # YAML config + ENV overrides
├── internal/
│ ├── api/
│ │ └── server.go # chi router, all handlers, CORS, auth middleware
│ ├── auth/
│ │ ├── jwt.go # JWT generation/validation (ENV-based secret)
│ │ ├── middleware.go # Auth middleware + GetClaims helper
│ │ └── password.go # bcrypt hashing (cost=10)
│ ├── config/
│ │ └── config.go # Viper-based config (YAML + ENV bindings)
│ ├── db/
│ │ ├── db.go # pgx pool, schema migrations, system_settings
│ │ └── repository.go # CRUD for devices, users, telemetry, alarms
│ ├── logging/
│ │ └── logger.go # slog + lumberjack rotation
│ ├── logserver/
│ │ └── server.go # Syslog UDP/TCP receiver + HTTP log endpoint
│ ├── models/
│ │ └── device.go # Domain models (Device, Alarm, User, ParsedLog)
│ ├── protocols/
│ │ ├── manager.go # ProtocolManager (Register, StartAll, StopAll)
│ │ ├── dahua.go # Dahua private protocol (0x12 0x34 header)
│ │ ├── hisilicon.go # Hisilicon binary with embedded JSON
│ │ ├── tvt.go # TVT XML/JSON hybrid
│ │ ├── ftp.go # FTP server (goftp.io) for snapshots
│ │ ├── snmp.go # SNMP trap receiver (gosnmp)
│ │ └── hikvision.go # Hikvision ISAPI (Digest auth, multipart)
│ ├── sip/
│ │ ├── handler.go # SIP message parser, REGISTER/MESSAGE/NOTIFY
│ │ └── gb28181.go # GB28181 20-digit device ID parser
│ ├── state/
│ │ └── manager.go # In-memory state (sync.Map)
│ └── worker/
│ └── pool.go # Buffered job queue (1000 capacity)


### Protocol Handlers

| Protocol | Mode | Port | Description |
|----------|------|------|-------------|
| **SIP GB28181** | Push | 5060 UDP/TCP | Chinese national standard, Hikvision/Dahua NVRs |
| **Dahua Private** | Push | 37777-37778 TCP | Proprietary binary (0x12 0x34 header) |
| **Hisilicon** | Push | 15002 TCP | Binary with embedded JSON |
| **TVT** | Push | 15003 TCP | XML/JSON hybrid protocol |
| **FTP** | Pull | 2121 TCP | Snapshot/log file uploads |
| **SNMP** | Push | 162 UDP | Trap receiver (v1/v2c/v3) |
| **Syslog** | Push | 515 UDP/TCP | Standard syslog (RFC 3164/5424) |
| **Hikvision ISAPI** | Pull | 80/443 HTTP | Digest auth, multipart stream |
| **ONVIF** | Pull | 80/443 HTTP | Profile S/T (planned Q3 2026) |

### Device State Management

```go
type DeviceStateManager interface {
    Get(deviceID string) (*Device, bool)
    Set(device *Device)
    Delete(deviceID string)
    GetAll() map[string]*Device
    UpdateLastSeen(deviceID string)
    SetOnline(deviceID string)
    SetOffline(deviceID string)
    AddAlarm(deviceID string, alarm *Alarm)
}

Реализация: sync.Map для конкурентного доступа, асинхронная запись в БД через буферизованный канал (1000 ёмкость).
Database Schema
Users & RBAC:

Database Schema
Users & RBAC:
sql 
users (id, username, password_hash, role, owner_id, email, avatar, status, last_login)
user_sessions (id, user_id, token_hash, ip_address, user_agent, expires_at)
audit_log (id, timestamp, user_id, action, entity_type, entity_id, old_value, new_value)

Devices & Telemetry:
sql
devices (device_id, owner_id, site_id, name, vendor_type, status, health, 
         connection_type, gb28181_device_id, p2p_brand, p2p_serial, ...)
telemetry (time, device_id, status, last_seen, heartbeat_interval) [hypertable]
sites (id, name, address, city, status, last_sync)

Alarms & Logs:
sql
alarms (id, time, device_id, priority, method, description, image_path, status) [hypertable]
parsed_logs (id, time, device_id, log_level, event_code, message, source, raw) [hypertable]

Analytics:
sql
predictions (id, device_id, prediction_date, failure_probability, explanation, model_version) [hypertable]
Settings (JSONB):
sql
system_settings (key, value JSONB, description, updated_by, updated_at)
-- Keys: services_syslog, services_gb28181, services_p2p_gateway, services_dahua, ...
🌐 P2P Gateway Architecture (Go 1.25)
Purpose
Прокси-слой для CCTV устройств за NAT/файрволом, которые не могут принимать входящие соединения. Поддерживает 4 основных P2P вендора.
p2p-gateway/
├── cmd/p2p-gateway/
│   ├── main.go              # Bootstrap, adapter registration
│   ├── api.go               # HTTP API (chi router)
│   ├── config.go            # YAML config loader
│   └── device_manager.go    # Device lifecycle, port allocation
├── pkg/
│   ├── adapters/
│   │   ├── adapter.go       # DeviceAdapter interface
│   │   ├── hikvision.go     # Hikvision P2P (via HikConnect)
│   │   ├── dahua.go         # Dahua P2P (via Python script)
│   │   ├── reolink.go       # Reolink P2P (via neolink binary)
│   │   └── xiongmai.go      # Xiongmai/Jftech P2P (native Go)
│   ├── hikp2p/              # Hikvision P2P protocol implementation
│   └── jftech/              # Jftech API client
└── internal/models/
    └── device.go            # Device model

Supported Vendors
Vendor
P2P Method
Binary/Script
Status
Hikvision
HikConnect
Native Go (hikp2p)
✅ Production
Dahua
gDMSS/DMSS
Python (dh-p2p)
✅ Production
Reolink
Reolink P2P
Rust (neolink)
✅ Production
Xiongmai
XMEye
Native Go (jftech)
✅ Production
EZVIZ
HikConnect
Via Hikvision adapter
✅ Production
TP-Link Tapo
Tapo Care
Planned
🔄 Q3 2026
Uniview
UniConnect
Planned
🔄 Q3 2026
Lorex
gDMSS
Via Dahua adapter
✅ Production
Swann
XMEye
Via Xiongmai adapter
✅ Production
API Endpoints
POST /p2p/register          # Register new P2P device
GET  /p2p/devices           # List all P2P devices
GET  /p2p/status/{id}       # Get device status + RTSP URL
POST /p2p/command/{id}      # Send PTZ command (Xiongmai only)
GET  /p2p/snapshot/{id}     # Get JPEG snapshot (Xiongmai only)
GET  /p2p/logs/{id}         # Get device logs (Xiongmai only)

🎨 Frontend Architecture (React 19)
State Management
Domain-separated Context API (без монолитного состояния):
typescript
<ThemeProvider>           // Dark/light/system mode
  <ToastProvider>         // Global toast notifications
    <AuthProvider>        // JWT token, user profile
      <SettingsProvider>  // System settings, services config
        <UsersProvider>   // User CRUD (admin only)
          <DevicesSitesProvider>   // Devices + Sites
            <TicketsProvider>      // Ticket lifecycle
              <AlertsProvider>     // Alarm management
                <NotificationsProvider>  // User notifications
                  <ReportsProvider>      // Report generation + history
UI Components
components/
├── ui/                  # Atomic: Card, Button, Badge, Modal, Table, Input, Toast
├── auth/                # PermissionGuard, RoleProtectedRoute
├── layout/              # Sidebar, Header, Layout
├── dashboard/           # AlertBanner, StatsCard
├── p2p/                 # P2PRegistrationForm, PTZControls
└── reports/             # ManualDownloadTab, ScheduledReportsTab, ReportHistoryTab
RBAC Model
6 ролей с иерархическими правами:
Role
Permissions
admin
Полный доступ ко всем функциям
manager
Управление командами, эскалация заявок, расширенные отчёты
technician
Конфигурация устройств, решение заявок, журналы обслуживания
viewer
Только чтение: панели, базовые отчёты, публичные статусы
owner
Только свои устройства (multi-tenancy)
support
Логи, аналитика, управление заявками
i18n
Default language: Russian (ru)
Fallback: ru
Supported: en, ru
Все строки UI через t('key')
Переводы в src/i18n.ts
📊 Analytics Pipeline (Python 3.11)
ML Model
Алгоритм: XGBoost binary classifier
Признаки (30-дневное окно):
offline_ratio — % времени, когда устройство было offline
error_count — общее количество ERROR-level логов
reboot_count — перезагрузки устройства (alarm method=6)
age_days — возраст устройства в днях
avg_alarm_priority — средний приоритет тревог
last_error_code — последний код ошибки
Выход: failure_probability (0.0–1.0) + explanation (через DeepSeek API, опционально)
ETL Process
python
# analytics/etl.py
1. Запрос telemetry, parsed_logs, alarms за последние 30 дней
2. Агрегация признаков по устройству
3. Обработка NULL (COALESCE to 0)
4. Возврат DataFrame для предсказания
🔒 Security Model
Authentication
JWT (HS256) с 24h expiry
Secret из ENV: JWT_SECRET
Token в localStorage
Header: Authorization: Bearer <token>
Password Security
bcrypt hashing (cost=10)
Min length: 6 chars (basic), 8+ chars with symbol (strong)
Смена пароля требует проверки текущего
Админ может сбрасывать пароли (логируется в audit_log)
RBAC Enforcement
Backend: auth.GetClaims(r).Role проверка в каждом handler
Frontend: <PermissionGuard> и <RoleProtectedRoute> компоненты
Audit Logging
Все критические действия логируются в audit_log:
User CRUD (CREATE_USER, UPDATE_USER, DELETE_USER)
Password changes (CHANGE_PASSWORD, RESET_PASSWORD)
Settings updates (UPDATE_SERVICES_SETTINGS)
Device modifications
🚀 Deployment
Current (Single-Node)
bash
# Backend
cd backend
export JWT_SECRET="..."
export DB_HOST="127.0.0.1"
go run main.go

# Frontend
cd frontend
npm run dev

# P2P Gateway
cd p2p-gateway
go run cmd/p2p-gateway/main.go

# Analytics (cron daily)
cd analytics
python predict.py

📈 Roadmap (Q2–Q4 2026)
✅ Q2 2026 — Completed
User management (CRUD, password change, admin reset)
Role-based access control (6 roles)
Audit logging for user actions
P2P Gateway UI integration (settings page)
GB28181 SIP server (full implementation)
Multi-vendor protocol support (Dahua, Hisilicon, TVT, SNMP)
🔄 Q2 2026 — In Progress
Rate limiting для /api/v1/auth/login
CORS hardening (restrict AllowedOrigins in production)
Health check endpoint (GET /health для Kubernetes probes)
WebSocket для real-time alarm push
Session management UI (view active sessions, revoke)
📋 Q3 2026 — Planned
2FA для admin accounts (TOTP)
API key management (create/revoke keys)
TP-Link Tapo P2P adapter
Uniview UniConnect adapter
MQTT adapter для IoT sensors
Prometheus metrics export (/metrics)
Grafana dashboards (pre-built templates)
Webhook receiver (generic HTTP POST)
Email-to-ticket converter (IMAP)
Slack/Telegram webhook notifications
Firmware update detection (SNMP OID polling)
🎯 Q4 2026 — Planned
Redis для кэширования
NATS/Kafka event bus
Multi-tenancy (partition by tenant_id)
Mobile app (React Native)
Video playback (HLS.js for RTSP→HLS)
ONVIF Profile S/T support
Elasticsearch для log aggregation
Jira/ServiceNow ticket sync
🔮 2027 — Long-term
Computer vision (object detection)
Edge computing (analytics on NVR/DVR)
Anomaly detection (Isolation Forest)
Natural language queries
GDPR compliance
SOC 2 Type II certification
📊 Performance Targets
Metric
Target
Current
API response time (p95)
< 200ms
~150ms
Device state update latency
< 1s
~500ms
Alarm processing throughput
10k/sec
~5k/sec
Database query time (p95)
< 100ms
~80ms
Frontend LCP
< 2.5s
~2.0s
P2P tunnel setup time
< 5s
~3s
💾 Disaster Recovery
Backup Strategy
Database: Daily pg_dump + WAL archiving (Point-in-Time Recovery)
Config: Git version control (все YAML файлы)
Media: S3-compatible storage для snapshots (lifecycle: 30 days)
RTO/RPO
RTO (Recovery Time Objective): 1 hour
RPO (Recovery Point Objective): 15 minutes (WAL archive interval)
📄 License
Proprietary — All rights reserved.