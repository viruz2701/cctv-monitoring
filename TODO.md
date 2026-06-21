🚨 Quick Wins (P0) — 2 недели
Security
QW-1.1 Заменить sshpass → golang.org/x/crypto/ssh (backend/internal/agent/actions.go)
QW-1.2 CSP nonce вместо 'unsafe-inline' (backend/internal/api/server.go)
QW-1.3 Убрать секреты из p2p-gateway/config.yaml → env vars
QW-1.4 Rate limiting на mobile + webhook endpoints
Architecture
QW-2.1 Разбить server.go на доменные роутеры (8 часов)
QW-2.2 Подключить golang-migrate вместо initSchema() (6 часов)
QW-2.3 Централизованная обработка ошибок respondError() (4 часа)
QW-2.4 Graceful Shutdown P2P Gateway (2 часа)
Mobile
QW-3.1 Refresh Token flow (5 часов)
QW-3.2 Параллельная загрузка фото (3 часа)
QW-3.3 Реальный EXIF из JPEG (3 часа)
Performance
QW-4.1 Excel/PDF генерация на Backend (8 часов)
QW-4.2 Разбить Settings.tsx на компоненты (4 часа)
QW-4.3 Виртуализация списков @tanstack/react-virtual (3 часа)
Infrastructure
QW-5.1 Dockerfile Backend multi-stage (2 часа)
QW-5.2 Единый docker-compose.yml (4 часа)
QW-5.3 Интеграционные тесты testcontainers-go (8 часов)
🆕 Phase 3.5 — Edge Agent + QR (3 месяца)
Epic 3.5.1: Edge Agent (OpenWrt)
Hardware: Wiflyer 4G (~$45) / GL-XE300 (~$65) / HDRM200 (~$90)
Stack: Go 1.25, ~5MB binary, ~30MB RAM idle
Protocols: ARP, ONVIF WS-Discovery, SNMP, mDNS, ISAPI, Dahua CGI, Modbus
Control Plane (MQTT 5.0 + mTLS 1.3)
edge-agent/cmd/agent/main.go — entry point
edge-agent/internal/discovery/ — ARP, ONVIF, SNMP, mDNS
edge-agent/internal/probe/ — SNMP v2c/v3, ISAPI, Dahua CGI, Modbus
edge-agent/internal/mqtt/ — MQTT 5.0 client (mTLS 1.3)
Backend: backend/internal/mqtt/broker.go — MQTT subscriber
Topics: edge/{agent_id}/telemetry|command|alarm
Собственный CA для автовыдачи сертификатов
QoS 1 для команд, QoS 0 для телеметрии
Media Plane (WireGuard on-demand)
Backend: backend/internal/wireguard/manager.go
Интеграция с wgdashboard (веб-UI для WG)
On-demand туннели: 1-2 часа, auto-close
Доступ: роутер + камеры + оборудование в LAN клиента
NetFlow-аудит сессии
Только роли admin / support
UI
Frontend: EdgeAgents.tsx, EdgeAgentDetail.tsx, DebugSession.tsx
Mobile: EdgeAgentListScreen.tsx, DebugSessionScreen.tsx
Backend API: /api/v1/edge/agents, /api/v1/edge/sessions
Deploy
edge-agent/install.sh — установка на OpenWrt
Auto-update через MQTT (OTA)
Telegram-бот для управления агентами
Epic 3.5.2: QR-паспортизация
Scope: NVR, switch, rack, UPS, сервер (НЕ уличные камеры)
Print: Bluetooth thermal printer (мобильный) ИЛИ enterprise batch (A4)
Backend
backend/internal/qr/service.go — генерация QR payload
backend/internal/qr/hmac.go — HMAC-SHA256 подпись
backend/internal/qr/base62.go — компактное кодирование UUID
Payload: https://cctv.company.com/q/{base62_id}?h={hmac_16}
Lifecycle: DRAFT → PRINTED → ACTIVE → REVOKED
Миграция: backend/migrations/003_qr_codes.sql
Таблицы: qr_codes, qr_scans, qr_print_jobs
API: POST /api/v1/qr/generate, GET /api/v1/qr/{id}, POST /api/v1/qr/{id}/scan
PDF-генерация: backend/internal/qr/pdf.go (gofpdf)
Макет наклейки 30×30 мм, A4-лист с сеткой 21 наклеек
Mobile
Расширить QRScannerScreen.tsx — сканирование + валидация
Offline-кэш QR (SQLite, TTL 24 часа)
После сканирования → карточка объекта
QRPrintScreen.tsx — печать через Bluetooth thermal printer
Библиотеки: react-native-ble-plx + react-native-print
Frontend
QRManagement.tsx — управление QR
Массовая генерация (batch PDF)
История печати, статистика сканирований
Security
HMAC-SHA256 подпись payload
Rate limiting: 100 req/min/IP
Whitelist доменов
Audit log всех сканирований
Автоматический отзыв при компрометации
Epic 3.5.3: UI для CMMS-адаптеров
frontend/src/pages/settings/IntegrationsSettings.tsx
Вкладки: Atlas / ServiceNow / Jira / 1С:ТОИР
Per-adapter configuration (URL, auth, field mapping)
Live test connection button
Sync status dashboard
⏸️ Phase 4 (Deferred)
Отложена до завершения Quick Wins + Phase 3.5
Multi-tenant SaaS (PostgreSQL RLS, Stripe)
AR Remote Expert (pion/webrtc)
ISO 27001 Certification
📈 Metrics
Metric
Target (Phase 3.5)
Code Coverage (Go)
70%
Code Coverage (React)
50%
API Response Time (p95)
<100ms
Edge Agents Deployed
5+
QR Objects Covered
95%
QR Scan Time
<2s
🎯 Success Criteria (Phase 3.5)
Все Quick Wins выполнены
Edge Agent: 5+ объектов, телеметрия real-time
WireGuard: on-demand сессии с аудитом, wgdashboard UI
QR: 95% объектов с QR, печать через BT printer + enterprise batch
QR Scan: <2 сек от сканирования до карточки
CMMS UI: ServiceNow/Jira/TOIR конфигурируются через UI</content>