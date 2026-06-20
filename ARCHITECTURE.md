# CCTV Intelligence Platform — Architecture Document v3.0

## 1. Executive Summary

CCTV Intelligence Platform — это зрелая экосистема для мониторинга CCTV, 
управления обслуживанием и аналитики. Платформа состоит из Desktop (React), 
Mobile (React Native), Telegram Bot и P2P Gateway, объединенных Go-бэкендом.

**Ключевая архитектура:** "CMMS Router Pattern". Бэкенд абстрагирует хранение 
нарядов и активов. Клиент может использовать встроенный CMMS (PostgreSQL) 
или подключить внешний (Atlas CMMS, ServiceNow, Jira) через адаптеры.

**Принципы:**
- **Headless-ready**: CMMS — это интерфейс (Adapter), а не жесткая привязка.
- **Mobile-first**: Оптимизированные API для линейного персонала.
- **Real-time**: WebSocket для мгновенных алертов и статусов.
- **ISO 27001 compliant**: Безопасность API-ключей, Push-токенов, аудит.

---

## 2. High-Level Architecture
┌─────────────────────────────────────────────────────────────────┐
│ CLIENTS LAYER │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Desktop │ │ Mobile App │ │ Telegram Bot │ │
│ │ (React/Vite) │ │ (React Native│ │ (Notifications│ │
│ │ │ │ /Expo) │ │ & Commands) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────────────┘
│ HTTPS / WSS
▼
┌─────────────────────────────────────────────────────────────────┐
│ API GATEWAY (Go) │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ chi router + Middleware (Auth, RBAC, RateLimit, CORS) │ │
│ │ Handlers: api, mobile, cmms, telegram, apikey, ws │ │
│ └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────┐
│ CORE DOMAIN SERVICES │
│ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Telemetry │ │ CMDB │ │ Gatekeeper │ │
│ │ Collector │ │ Service │ │ Service │ │
│ │ (RTSP/SNMP/ │ │ (Devices, │ │ (GPS/EXIF/ │ │
│ │ ISAPI/SIP) │ │ Sites, QR) │ │ AI Verify) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
│ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Alarm & │ │ SLA & │ │ AI/ML │ │
│ │ State Manager│ │ Workload │ │ Service │ │
│ │ (WebSocket) │ │ Manager │ │ (XGBoost) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────┐
│ INTEGRATION LAYER │
│ │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ CMMS Router & Adapter Framework │ │
│ │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │ │
│ │ │ Internal │ │ Atlas │ │ServiceNow│ │ Jira │ │ │
│ │ │ (PgSQL) │ │ Adapter │ │ Adapter │ │ Adapter │ │ │
│ │ └──────────┘ └──────────┘ └──────────┘ └──────────┘ │ │
│ └──────────────────────────────────────────────────────────┘ │
│ │
│ ┌──────────────────────────────────────────────────────────┐ │
│ │ P2P Gateway (Go/Rust) │ │
│ │ Adapters: Hikvision, Dahua, Reolink, JFtech, Xiongmai │ │
│ └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────┐
│ DATA LAYER │
│ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────┐ │
│ │ TimescaleDB │ │ PostgreSQL │ │ Redis │ │
│ │ (telemetry, │ │ (CMDB, CMMS, │ │ (Cache, │ │
│ │ alarms, logs) │ │ Users, SLA, │ │ Sessions, │ │
│ │ │ │ API Keys) │ │ WS Hub) │ │
│ └──────────────────┘ └──────────────────┘ └──────────────┘ │
│ │
│ ┌──────────────────┐ ┌──────────────────┐ │
│ │ Object Storage │ │ Vault │ │
│ │ (Photos, Reports)│ │ (Secrets, JWT, │ │
│ │ │ │ API Keys) │ │
│ └──────────────────┘ └──────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
## 3. Domain Model (Обновлено)

### 3.1 Core Entities
**Device**: id, name, type, vendor, model, serial, site_id, custom_fields (JSONB), qr_code, status.
**Site / Location**: Hierarchy (Site -> Building -> Floor -> Rack). GPS, geofence.
**Alarm**: id, device_id, type, severity, status, cmms_ticket_id.
**WorkOrder**: id, type, device_id, assignee_id, status, priority, sla_deadline, cmms_external_id, verification (GPS/EXIF/AI).
**SparePart**: id, name, qty, min_stock, cost, location.
**Technician**: id, user_id, skills, base_location, current_workload, push_token (encrypted).
**SLAConfig**: priority, response_time, resolution_time.
**APIKey**: id, name, hash, permissions, owner_id, expires_at.

### 3.2 CMMS Adapter Interface
```go
type CMMSAdapter interface {
    CreateWorkOrder(ctx, event) (*WorkOrder, error)
    UpdateWorkOrder(ctx, id, update) error
    SyncAsset(ctx, device) error
    GetTCOData(ctx, assetID) (*TCOData, error)
    HealthCheck(ctx) error
}
Реализации: InternalAdapter (работает с db/repository.go), AtlasAdapter, ServiceNowAdapter.
4. UX Architecture (Референсы для существующего UI)
4.1 Desktop (React)
Применяем паттерны Shelf.nu/Snipe-IT к существующим страницам:
WorkOrders.tsx: Таблица с фильтрами, bulk actions, статус-бейджи (Snipe-IT style).
SpareParts.tsx: Карточки с thumbnail, qty indicator, min_stock alert (Shelf.nu style).
SLADashboard.tsx: Таймлайны, gauge charts для SLA compliance.
4.2 Mobile (React Native)
Экран техника: Крупные кнопки, offline-first, чек-листы с фото.
Gatekeeper UI: Экран верификации: GPS статус, EXIF проверка, кнопка "AI Check".
QR Scan: Интеграция с камерой, переход в карточку актива.
5. ISO 27001 Compliance Matrix (Ключевые контролы)
A.9 Access Control: RBAC (6 ролей), PermissionGuard, RoleProtectedRoute.
A.10 Cryptography: API Keys в Vault, Push Tokens шифруются в БД (SavePushToken).
A.12.4 Logging: Audit log для всех действий техников (закрытие нарядов, верификация).
A.13 Communications: TLS 1.3, mTLS для P2P Gateway.
A.14 Development: SAST/DAST, secure coding для Telegram Bot (защита от injection).
6. Integration Architecture
6.1 CMMS Router
Маршрутизирует запросы из cmms_handlers.go и mobile_handlers.go в нужный адаптер.
Поддерживает Fallback: если Atlas недоступен, пишет во Internal DB и ставит в очередь.
6.2 P2P Gateway
Микросервис для проброса P2P-потоков.
Go: API, device manager, adapters (Hikvision, Dahua).
Rust: Высокопроизводительные бинарники (neolink, dh-p2p).
6.3 Telegram Bot
Канал для уведомлений и быстрых команд (подтверждение алертов, статусы).
Интеграция через telegram_handlers.go.
7. Technology Stack
Backend: Go 1.25 (chi, pgx, NATS, WebSocket).
Frontend: React 19, Vite 8, Tailwind 4, TypeScript 5.9.
Mobile: React Native / Expo, TypeScript, React Navigation.
P2P Gateway: Go + Rust (neolink, dh-p2p).
Data: TimescaleDB, PostgreSQL, Redis.
Infrastructure: Docker, Kubernetes, Vault.

8. Roadmap Alignment
Фаза
Срок
Ключевые deliverables
Phase 1
Месяцы 1-2
Atlas Adapter, Gatekeeper (Mobile+Go), UX Refresh (Web)
Phase 2
Месяцы 3-5
AI Predictive, TCO, Voice-to-Report
Phase 3
Месяцы 6-9
ServiceNow/Jira Adapters, Self-Healing
Phase 4
Месяцы 10-15
SaaS, AR, ISO Certification
