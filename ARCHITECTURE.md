# CCTV Intelligence Platform — Architecture Document v6.0

**Дата обновления:** 2026-06-21
**Статус:** Phase 3 in progress (Universal CMMS Gateway + Self-Healing)
**Версия:** 6.0 (post-Phase 2)

---

## 1. Executive Summary

CCTV Intelligence Platform — зрелая экосистема enterprise-класса для мониторинга CCTV, предиктивного обслуживания и ИТ/ИБ-конвергенции.

**Текущий стек:**
- **Backend:** Go 1.25 (chi, pgx/v5, gorilla/websocket, telegram-bot-api)
- **Frontend:** React 19, Vite 8, Tailwind 4, TypeScript 5.9, i18next
- **Mobile:** React Native / Expo 52, React Query, Zustand
- **P2P Gateway:** Go 1.25 + бинарные адаптеры (neolink, dh-p2p)
- **Analytics:** Python 3.11 (XGBoost, pandas, psycopg2)
- **AI/ML:** XGBoost (predictions), DeepSeek Vision (gatekeeper), Whisper + DeepSeek NLP (voice)
- **Data:** PostgreSQL + TimescaleDB (hypertables)

**Ключевые архитектурные решения:**
- **ADR-001:** Headless CMMS ✅
- **ADR-002:** CMMS Adapter Pattern ✅ (InternalAdapter + AtlasAdapter + Router)
- **ADR-003:** Event Bus (NATS) — Phase 3
- **ADR-004:** Gatekeeper Pattern ✅ (GPS/EXIF/AI)

**Выполненные фазы:**
- ✅ Phase 0: Foundation & Analysis
- ✅ Phase 1: Headless CMMS (CMMS Router, Maintenance Schedules)
- ✅ Phase 1.5: Gatekeeper + ISO Quick Wins + UX Refresh
- ✅ Phase 2: Atlas Integration + Predictive Maintenance + TCO + Voice-to-Report

**В процессе:**
- 🔄 Phase 3: ServiceNow / Jira / 1С:ТОИР адаптеры, NATS Event Bus, Agentic Self-Healing

---

## 2. High-Level Architecture


┌─────────────────────────────────────────────────────────────────────┐
│ CLIENTS LAYER │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Desktop │ │ Mobile App │ │ Telegram Bot │ │
│ │ (React/Vite) │ │ (Expo/RN) │ │ (Commands) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
│ HTTPS / WSS
▼
┌─────────────────────────────────────────────────────────────────────┐
│ API GATEWAY (Go/chi) │
│ Middleware: SecurityHeaders ✅, RateLimiter ✅, Auth, RBAC, CORS │
│ Handlers: api, mobile, cmms, gatekeeper ✅, telegram, ws, p2p, │
│ atlas_sync ✅, predictions ✅, tco ✅, voice ✅ │
└─────────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────┐
│ CORE DOMAIN SERVICES │
│ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ Telemetry │ │ Gatekeeper │ │ Predictions │ │
│ │ Collector │ │ Service ✅ │ │ Service ✅ │ │
│ │ (8 protos) │ │ (GPS/EXIF/AI)│ │ (XGBoost) │ │
│ └──────────────┘ └──────────────┘ └──────────────┘ │
│ │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ │
│ │ TCO │ │ Voice │ │ Alarm & │ │
│ │ Calculator ✅│ │ NLP ✅ │ │ State Mgr │ │
│ │ (per device) │ │ (Whisper+NLP)│ │ (WebSocket) │ │
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
│ │ │ ✅ │ │ ✅ │ │ Phase 3 │ │ Phase 3 │ │ │
│ │ └──────────┘ └──────────┘ └──────────┘ └──────────┘ │ │
│ │ ┌──────────┐ ┌──────────────┐ │ │
│ │ │ Toir │ │ Fallback │ │ │
│ │ │ Adapter │ │ Queue ✅ │ │ │
│ │ │ Phase 3 │ │ (offline) │ │ │
│ │ └──────────┘ └──────────────┘ │ │
│ └────────────────────────────────────────────────────────────┘ │
│ │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ Event Bus (NATS) — Phase 3 ⚠️ │ │
│ │ Topics: alarms.{device_id}, cmms.workorder.*, predictions │ │
│ └────────────────────────────────────────────────────────────┘ │
│ │
│ ┌────────────────────────────────────────────────────────────┐ │
│ │ P2P Gateway (Go + binaries) ✅ │ │
│ │ Adapters: Hikvision, Reolink, Dahua, Xiongmai │ │
│ └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
│
▼
┌─────────────────────────────────────────────────────────────────────┐
│ DATA LAYER │
│ ┌──────────────────┐ ┌──────────────────┐ ┌──────────────┐ │
│ │ TimescaleDB │ │ PostgreSQL │ │ Redis │ │
│ │ (telemetry, │ │ (CMDB, CMMS, │ │ (Cache, │ │
│ │ alarms, logs, │ │ Users, SLA, │ │ Sessions, │ │
│ │ predictions ✅)│ │ API Keys) │ │ NATS ⚠️) │ │
│ └──────────────────┘ └──────────────────┘ └──────────────┘ │
└─────────────────────────────────────────────────────────────────────┘


---

## 3. Domain Model (обновлено)

### 3.1 Core Entities

| Entity | Таблица | Статус | Описание |
|--------|---------|--------|----------|
| Device | devices | ✅ | + lat/lng/geofence |
| Site | sites | ✅ | + geofence_polygon |
| WorkOrder | work_orders | ✅ | + verification_token, atlas_external_id |
| SparePart | spare_parts | ✅ | + sku, min_stock, tco_contribution |
| MaintenanceSchedule | maintenance_schedules | ✅ | + auto_create_from_prediction |
| APIKey | api_keys | ✅ | bcrypt hash + prefix |
| Prediction | predictions (hypertable) | ✅ | + ttf (time-to-failure), features |
| TCORecord | tco_records ✅ | NEW | device_id, capex, opex, mttr, mtbf |
| VoiceReport | voice_reports ✅ | NEW | user_id, audio_path, transcript, entities (JSONB) |
| AtlasSyncState | atlas_sync_state ✅ | NEW | last_sync, pending_changes, conflict_log |

### 3.2 CMMS Adapter Interface (ADR-002) — 33 метода

```go
type CMMSAdapter interface {
    // Work Orders, Spare Parts, Maintenance, SLA, Technicians, Reports
    // Total: 33 methods
}
Реализации:
Adapter
Статус
Target System
API Docs
InternalAdapter
✅ Production
PostgreSQL
Internal
AtlasAdapter
✅ Production (Phase 2)
Atlas CMMS
Maintenancex/Atlas REST API
ServiceNowAdapter
🔄 Phase 3
ServiceNow ITSM
ServiceNow REST API
JiraAdapter
🔄 Phase 3
Jira Service Mgmt
Jira REST API v3
ToirAdapter
🔄 Phase 3
1С:ТОИР (152-ФЗ)
ТОИР API Docs
3.3 Gatekeeper Pattern (ADR-004) — ✅
Mobile → POST /verify → GPS+EXIF+AI → verification_token (JWT, 10 min TTL)
Mobile → POST /complete + token → validate → complete WO
3.4 Predictive Maintenance Pipeline (Phase 2 ✅)
TimescaleDB telemetry → ETL (Python) → XGBoost model
                                              ↓
                              predictions table (TTF, failure_prob)
                                              ↓
                    ┌────────────────────────┼────────────────────┐
                    ▼                        ▼                    ▼
            Frontend Predictions    Auto-create PM WO    Mobile push
            (DeepSeek explain)      (CMMS Router)        notifications

3.5 TCO Calculator Pipeline (Phase 2 ✅)
CMMS data (spare parts + labor hours + cost)
            ↓
    Aggregator per device/site
            ↓
    CapEx + OpEx + MTTR + MTBF
            ↓
    Replace vs Repair recommendation (threshold-based)
            ↓
    Frontend TCO.tsx (charts + recommendations)
3.6 Voice-to-Report Pipeline (Phase 2 ✅)
Mobile VoiceRecorder → audio blob → POST /voice/reports
                                         ↓
                              Whisper API → transcript
                                         ↓
                              DeepSeek NLP → entities (JSONB):
                                  {device_id, site_id, 
                                   issue_type, parts_used,
                                   confidence_score}
                                         ↓
                    ┌────────────────────┼────────────────────┐
                    ▼                    ▼                    ▼
            CMDB auto-update    WorkOrder creation    Audit log
            (if conf > 0.8)     (if conf > 0.7)       (all reports)
4. Security Architecture (ISO 27001)
4.1 Реализовано ✅
Control
Реализация
A.9.1 RBAC
6 ролей, PermissionGuard
A.10.1 TOTP 2FA
RFC 6238
A.10.2 API Keys
bcrypt(cost=12) + prefix
A.10.3 Push Tokens
AES-256-GCM
A.10.4 JWT Secret
env var (panic if missing)
A.12.4 Audit Log
все write-операции
A.13.2 Security Headers
CSP, X-Frame-Options, HSTS
A.13.3 Rate Limiting
login + mobile endpoints
4.2 Pending (Phase 3-4)
Gap
Phase
Redis sessions
Phase 3
Vault integration
Phase 3
HMAC audit log integrity
Phase 3
JWT → HttpOnly Cookies + CSRF
Phase 3
CI/CD vulnerability scanning
Phase 3
ISO 27001 Stage 1 + Stage 2 audit
Phase 4
5. Integration Architecture
5.1 CMMS Router (ADR-002)
Handler → CMMSRouter.adapter
                    ↓
        ┌───────────┼───────────┬──────────┬──────────┐
        ▼           ▼           ▼          ▼          ▼
   Internal    Atlas (✅)  ServiceNow  Jira        Toir
   Adapter     OAuth2      Phase 3    Phase 3    Phase 3
                    ↓
            Fallback Queue (✅)
                    ↓
        Internal DB + async retry

5.2 Atlas Integration Details (Phase 2 ✅)
Auth: OAuth2 с automatic token refresh
Sync: Bi-directional webhook-driven
Fallback: Internal DB + async queue если Atlas недоступен
Conflict Resolution: Atlas-wins для статуса, наш-wins для локальных полей
Health Check: /api/v1/integrations/atlas/health
5.3 NATS Event Bus (Phase 3)
Publisher:
  - alarms.{device_id}
  - cmms.workorder.{created|updated|completed}
  - predictions.{device_id}
  - telemetry.{device_id}

Subscribers:
  - WebSocket Hub (desktop)
  - Mobile push service
  - Worker (async tasks)
  - Analytics service
5.4 Agentic Self-Healing (Phase 3)
Alarm → AI Agent (topology analysis)
              ↓
      Decision Tree:
        ├── Auto-fix (ISAPI/ONVIF via P2P)
        ├── Human-approval required
        └── Escalate to CMMS
              ↓
      CMMS Router: auto-close ticket on success
6. Roadmap (Updated)
Phase
Срок
Статус
Deliverables
Phase 0
Недели 1-2
✅ Done
UX Research, ISO Gap
Phase 1
Месяцы 1-2
✅ Done
CMMS Router, Maintenance
Phase 1.5
Месяц 3
✅ Done
Gatekeeper, ISO Quick Wins, UX
Phase 2
Месяцы 4-6
✅ Done
Atlas, Predictions, TCO, Voice
Phase 3
Месяцы 7-9
🔄 Current
ServiceNow/Jira/Toir, NATS, Self-Healing
Phase 4
Месяцы 10-15
Pending
SaaS, AR, ISO Certification
7. API Adapter Reference (для разработчиков)
Atlas CMMS (✅ Production)
Docs: https://docs.atlas-cmms.com/api
Auth: OAuth2 (client_credentials)
Rate Limit: 100 req/min
Key Endpoints:
POST /api/v2/workorders — create WO
PATCH /api/v2/workorders/{id} — update WO
GET /api/v2/assets — sync assets
POST /webhooks/cmms — inbound sync
ServiceNow (Phase 3)
Docs: https://developer.servicenow.com/dev.do
Auth: OAuth2 + Basic Auth fallback
Rate Limit: 1000 req/min (instance-dependent)
Key Endpoints:
POST /api/now/table/incident — create incident
PATCH /api/now/table/incident/{sys_id}
GET /api/now/cmdb/instance/{class} — CMDB sync
Webhooks: Scripted REST API для inbound sync
Jira Service Management (Phase 3)
Docs: https://developer.atlassian.com/cloud/jira/platform/rest/v3/
Auth: OAuth 2.0 (3LO) или API Token
Rate Limit: зависит от плана Atlassian
Key Endpoints:
POST /rest/api/3/issue — create issue
PUT /rest/api/3/issue/{id} — update
GET /rest/servicedeskapi/request/{id} — service request
1С:ТОИР (Phase 3, РФ-специфика)
Docs: https://toir.ru/docs/api
Auth: Basic Auth + 152-ФЗ compliance
Rate Limit: 60 req/min
Key Endpoints:
POST /hs/TOIR_API/v1/requests — заявка
GET /hs/TOIR_API/v1/equipment — оборудование
Особенности: кириллица в JSON, ГОСТ форматы дат
8. File Structure (актуальная, ~230 файлов)
├── backend/
│   ├── internal/
│   │   ├── api/          # + atlas, predictions, tco, voice handlers
│   │   ├── cmms/         # + atlas_client.go, atlas_sync.go, fallback_queue.go
│   │   ├── crypto/       # AES-256-GCM ✅
│   │   ├── gatekeeper/   # ✅ production-ready
│   │   ├── predictions/  # ✅ XGBoost wrapper
│   │   ├── tco/          # ✅ calculator + aggregator
│   │   ├── voice/        # ✅ Whisper + DeepSeek NLP
│   │   ├── protocols/    # 8 protocols
│   │   ├── sip/          # GB28181
│   │   └── ws/           # WebSocket hub
│   └── analytics/        # Python ML
├── frontend/
│   ├── src/pages/        # + TCO.tsx, Predictions.tsx, Integrations.tsx
│   └── src/components/ui/
├── mobile/
│   └── src/screens/      # + VoiceReportScreen.tsx
└── p2p-gateway/

