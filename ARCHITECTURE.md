Stack
Backend: Go 1.25 (chi, pgx/v5, nats.go, telegram-bot-api)
Frontend: React 19, Vite 8, Tailwind 4, TypeScript 5.9
Mobile: React Native / Expo 52, React Query, Zustand
Edge Agent: Go 1.25 на OpenWrt (MQTT 5.0 + WireGuard)
Data: PostgreSQL + TimescaleDB
Event Bus: NATS JetStream (internal) + MQTT 5.0 (edge)
High-Level Architecture
Clients (Desktop/Mobile/Telegram)
         ↓ HTTPS/WSS
API Gateway (Go/chi) — Auth, RBAC, Rate Limit
         ↓
Core Services: Telemetry | Gatekeeper | Predictions | TCO | Voice
               Edge Agent Mgr | QR Service | WG Manager
         ↓
Integration: CMMS Router (5 adapters) | NATS | MQTT | P2P Gateway
         ↓
Data: PostgreSQL + TimescaleDB | Redis | NATS JetStream
         ↓
Edge Layer: OpenWrt Agents (Go) | WireGuard | wgdashboard
Key Decisions (ADRs)
ADR-001: Headless CMMS ✅
ADR-002: CMMS Adapter Pattern (5 adapters: Internal, Atlas, ServiceNow, Jira, Toir) ✅
ADR-003: Event Bus (NATS + MQTT) ✅
ADR-004: Gatekeeper Pattern (GPS+EXIF+AI) ✅
ADR-005: QR-паспортизация ✅ (Phase 3.5)
ADR-006: Edge Agent на OpenWrt ✅ (Phase 3.5)
ADR-007: Phase 4 Deferred ✅
Edge Agent Architecture (ADR-006)
Принцип: Разделение Control/Media plane
Control Plane (MQTT 5.0 + mTLS 1.3):
Телеметрия, heartbeat, алармы, команды
On-demand скриншоты (io.Copy, без сохранения)
Собственный CA, автовыдача сертификатов
Media Plane (WireGuard on-demand):
Доступ к роутеру + камерам + оборудованию в LAN клиента
Сессии 1-2 часа, auto-close, NetFlow-аудит
Только роли admin/support
wgdashboard для управления WG
Hardware: Wiflyer 4G (~$45) / GL-XE300 (~$65) / HDRM200 (~$90)
Size: ~5MB binary, ~30MB RAM idle
QR-паспортизация (ADR-005)
Scope: NVR, switch, rack, UPS, сервер (НЕ уличные камеры)
Payload: https://cctv.company.com/q/{base62_id}?h={hmac_16}
Lifecycle: DRAFT → PRINTED → ACTIVE → REVOKED
Print: Bluetooth thermal printer (мобильный) ИЛИ enterprise batch (A4)
CMMS Adapters
Adapter
Status
Target
InternalAdapter
✅ Production
PostgreSQL
AtlasAdapter
✅ Production
Atlas CMMS
ServiceNowAdapter
✅ Backend
ServiceNow ITSM
JiraAdapter
✅ Backend
Jira Service Mgmt
ToirAdapter
✅ Backend
1С:ТОИР (152-ФЗ)
Security (ISO 27001)
RBAC (6 ролей), TOTP 2FA, API Keys (bcrypt)
Push Tokens (AES-256-GCM), JWT Secret (env)
Audit Log (HMAC-подпись), Security Headers (CSP, HSTS)
Edge Agent: mTLS 1.3, WireGuard ChaCha20-Poly1305
QR: HMAC-SHA256, rate limiting, audit log
File Structure
backend/
├── internal/
│   ├── api/          # HTTP handlers (разбить на доменные роутеры!)
│   ├── cmms/         # 5 adapters (Internal, Atlas, SN, Jira, Toir)
│   ├── agent/        # Self-healing (decision, playbook, approval)
│   ├── edge/         # 🆕 MQTT broker client, WG manager
│   ├── qr/           # 🆕 service, hmac, base62, pdf
│   ├── wireguard/    # 🆕 manager, peer, session
│   ├── events/       # NATS publisher/subscriber
│   ├── sync/         # Conflict resolver
│   └── gatekeeper/   # GPS/EXIF/AI verification
└── migrations/       # golang-migrate files

edge-agent/           # 🆕 Go binary для OpenWrt
├── cmd/agent/
├── internal/
│   ├── discovery/    # ARP, ONVIF, SNMP, mDNS
│   ├── probe/        # SNMP, ISAPI, Dahua CGI, Modbus
│   └── mqtt/         # MQTT 5.0 client

frontend/
├── src/pages/        # + EdgeAgents.tsx, QRManagement.tsx
└── src/pages/settings/ # разбито на компоненты

mobile/
└── src/screens/      # + EdgeAgentListScreen.tsx, QRPrintScreen.tsx
Current Status
✅ Phase 0-3 Backend: Done
🔄 Quick Wins (P0): In Progress
🔄 Phase 3.5: Edge Agent + QR (Next)
⏸️ Phase 4: Deferred</content>
