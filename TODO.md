
---

### 2. `TODO.md` (v6.0)

```markdown
# TODO.md — Task Tracker for CCTV Intelligence Platform v6.0

**Обновлено:** 2026-06-21
**Текущая фаза:** 3 (Universal CMMS Gateway + Self-Healing)

---

## 📊 Progress Overview

| Epic | Phase | Priority | Status | Progress |
|------|-------|----------|--------|----------|
| Foundation & Analysis | 0 | P0 | ✅ Done | 100% |
| Headless CMMS & Adapter | 1 | P0 | ✅ Done | 100% |
| Gatekeeper Service | 1.5 | P0 | ✅ Done | 100% |
| ISO 27001 Quick Wins | 1.5 | P0 | ✅ Done | 100% |
| UX Refresh (Desktop) | 1.5 | P1 | ✅ Done | 100% |
| Atlas CMMS Integration | 2 | P1 | ✅ Done | 100% |
| Predictive Maintenance | 2 | P1 | ✅ Done | 100% |
| TCO Calculator | 2 | P1 | ✅ Done | 100% |
| Voice-to-Report | 2 | P1 | ✅ Done | 100% |
| Enterprise Adapters (ServiceNow/Jira/Toir) | 3 | P1 | 🔴 Not Started | 0% |
| NATS Event Bus | 3 | P1 | 🔴 Not Started | 0% |
| Agentic Self-Healing | 3 | P1 | 🔴 Not Started | 0% |
| Bi-directional ITSM Sync | 3 | P1 | 🔴 Not Started | 0% |
| Multi-tenant SaaS | 4 | P2 | 🔴 Not Started | 0% |
| AR Remote Expert | 4 | P2 | 🔴 Not Started | 0% |
| ISO 27001 Certification | 4 | P2 | 🔴 Not Started | 0% |

---

## ✅ PHASE 0-1.5 (DONE) — см. историю

Все задачи фаз 0, 1, 1.5 выполнены. См. предыдущие версии TODO.md.

---

## ✅ PHASE 2: AI Intelligence & Atlas Integration (DONE)

### Epic 2.1: Atlas CMMS Integration [P1] ✅
- [x] **2.1.1** `atlas_client.go` — OAuth2 client с auto-refresh
- [x] **2.1.2** AtlasAdapter: CreateWorkOrder с маппингом полей
- [x] **2.1.3** AtlasAdapter: UpdateWorkOrder (bi-directional sync)
- [x] **2.1.4** AtlasAdapter: SyncAsset (device → CMMS asset)
- [x] **2.1.5** `fallback_queue.go` — offline fallback + retry queue
- [x] **2.1.6** Settings → Integrations: Atlas URL, API Key, field mapping UI
- [x] **2.1.7** Health check endpoint `/api/v1/integrations/atlas/health`
- [x] **2.1.8** Webhooks от Atlas → inbound sync handler

### Epic 2.2: Predictive Maintenance [P1] ✅
- [x] **2.2.1** `predict.py` расширен: HDD, PoE, Temperature features
- [x] **2.2.2** Go Backend: `/api/v1/predictions` endpoints
- [x] **2.2.3** `predictions/predictor.go` — XGBoost wrapper
- [x] **2.2.4** CMMS Router: авто-создание PM-задач из predictions
- [x] **2.2.5** Mobile: push-уведомления о предстоящих PM
- [x] **2.2.6** Frontend: `Predictions.tsx` с DeepSeek explanations

### Epic 2.3: TCO Calculator [P1] ✅
- [x] **2.3.1** `tco/aggregator.go` — агрегация из CMMS (parts, labor, cost)
- [x] **2.3.2** `tco/calculator.go` — TCO per device/site
- [x] **2.3.3** Frontend: `TCO.tsx` страница (графики, recommendations)
- [x] **2.3.4** Replace vs Repair логика (threshold-based)

### Epic 2.4: Voice-to-Report [P1] ✅
- [x] **2.4.1** Mobile: `VoiceReportScreen.tsx` + Whisper API integration
- [x] **2.4.2** `voice/whisper_client.go` — Whisper transcription
- [x] **2.4.3** `voice/nlp_processor.go` — DeepSeek NLP → entities
- [x] **2.4.4** `voice/cmdb_updater.go` — авто-обновление CMDB
- [x] **2.4.5** Confidence-based automation (conf > 0.8 → auto, > 0.7 → propose)

### 📈 Phase 2 Success Metrics (достигнуты)
- [x] Atlas Integration Uptime: 99.2% (target 99%)
- [x] Predictive Maintenance accuracy: AUC 0.87
- [x] TCO calculation coverage: 100% devices/sites
- [x] Voice-to-Report NER accuracy: 82% (entity extraction)
- [x] Code Coverage Go: 52% (target 50%)
- [x] Code Coverage React: 32% (target 30%)

---

## 🔄 PHASE 3: Universal CMMS Gateway + Self-Healing (CURRENT)

**Срок:** Месяцы 7-9 (3 месяца)
**Цель:** Мульти-CMMS интеграция, real-time event bus, автономное восстановление

### Epic 3.1: Enterprise Adapters [P1] 🔴

**3.1.1 ServiceNowAdapter**
- [ ] Создать `backend/internal/cmms/servicenow/adapter.go`
- [ ] OAuth2 client с instance-specific endpoints
- [ ] Реализовать 33 метода интерфейса CMMSAdapter
- [ ] Интеграция с CMDB CI (Configuration Items)
- [ ] Маппинг Incident / Problem / Change records
- [ ] **API Docs:** https://developer.servicenow.com/dev.do

**3.1.2 JiraAdapter**
- [ ] Создать `backend/internal/cmms/jira/adapter.go`
- [ ] OAuth 2.0 (3LO) + API Token fallback
- [ ] Интеграция с Jira Service Management
- [ ] Маппинг Request / Incident / Asset
- [ ] Atlassian Marketplace packaging (опционально)
- [ ] **API Docs:** https://developer.atlassian.com/cloud/jira/platform/rest/v3/

**3.1.3 ToirAdapter (1С:ТОИР)**
- [ ] Создать `backend/internal/cmms/toir/adapter.go`
- [ ] 152-ФЗ compliance (персональные данные)
- [ ] Кириллица в JSON, ГОСТ форматы
- [ ] Интеграция с ТОИР API
- [ ] **API Docs:** https://toir.ru/docs/api

**3.1.4 UI для мульти-адаптеров**
- [ ] `Integrations.tsx` — drag-and-drop маппинг полей
- [ ] Per-adapter configuration (URL, auth, field mapping)
- [ ] Live test connection button
- [ ] Sync status dashboard

### Epic 3.2: NATS Event Bus [P1] 🔴

**3.2.1 Infrastructure**
- [ ] Добавить `github.com/nats-io/nats.go` в go.mod
- [ ] Docker Compose: NATS server с JetStream
- [ ] Конфигурация: URL, credentials, TLS

**3.2.2 Publisher layer**
- [ ] `backend/internal/events/publisher.go`
- [ ] Topics:
  - `alarms.{device_id}` — real-time alarms
  - `cmms.workorder.{event}` — WO lifecycle
  - `predictions.{device_id}` — new predictions
  - `telemetry.{device_id}` — telemetry stream
- [ ] Schema registry (JSON Schema / Protobuf)

**3.2.3 Subscriber layer**
- [ ] WebSocket Hub — subscribe to alarms + WO updates
- [ ] Mobile push service — push notifications
- [ ] Worker pool — async task processing
- [ ] Analytics service — real-time feature extraction

**3.2.4 JetStream для persistence**
- [ ] Persistent streams для аудита
- [ ] Replay capability для аналитики
- [ ] Message deduplication

### Epic 3.3: Agentic Self-Healing [P1] 🔴

**3.3.1 AI Agent core**
- [ ] `backend/internal/agent/` пакет
- [ ] Topology analyzer (device → switch → NVR graph)
- [ ] Decision tree: auto-fix vs human-approval vs escalate
- [ ] Playbook engine (YAML-based)

**3.3.2 Remediation actions**
- [ ] ISAPI commands через P2P Gateway
- [ ] ONVIF PTZ/reboot commands
- [ ] SNMP reset commands
- [ ] SSH-based device restart (опционально)

**3.3.3 CMMS integration**
- [ ] Авто-создание тикета при alarm
- [ ] Авто-закрытие тикета после successful self-healing
- [ ] Audit trail всех agentic действий

**3.3.4 Human-in-the-loop**
- [ ] Approval workflow для критичных действий
- [ ] Telegram bot approval messages
- [ ] Mobile push для срочных approvals
- [ ] Timeout → fallback to manual

### Epic 3.4: Bi-directional ITSM Sync [P1] 🔴

**3.4.1 Webhook handlers**
- [ ] ServiceNow scripted REST API
- [ ] Jira webhooks (issue_updated, status_changed)
- [ ] 1С:ТОИР webhook endpoints

**3.4.2 State Machine**
- [ ] Sync каждые 5 минут (cron)
- [ ] Conflict detection + resolution
- [ ] Eventual consistency guarantees

**3.4.3 Conflict Resolution**
- [ ] External-wins для статуса (ServiceNow authority)
- [ ] Local-wins для метаданных (наши fields)
- [ ] Conflict log для ручного review
- [ ] Auto-reopen closed tickets при рецидиве alarm

### Epic 3.5: ISO 27001 Medium-term [P1] 🔴
- [ ] MT-1: Security Policy (`docs/iso27001/security-policy.md`)
- [ ] MT-2: Asset Classification (`asset_class` в БД)
- [ ] MT-3: Audit Log Integrity (HMAC-подпись)
- [ ] MT-4: CORS whitelist restriction
- [ ] MT-5: Incident Response Plan
- [ ] MT-6: CI/CD Vulnerability scanning (gosec, trivy, npm audit)

---

## 📍 PHASE 4: Enterprise Scale & ISO Cert (Месяцы 10-15)

### Epic 4.1: Multi-tenant SaaS [P2]
- [ ] PostgreSQL RLS (Row-Level Security)
- [ ] Billing tiers (Community / Pro / Enterprise)
- [ ] Stripe integration
- [ ] Tenant isolation в WebSocket Hub + NATS

### Epic 4.2: AR Remote Expert [P2]
- [ ] pion/webrtc в Go Backend
- [ ] Mobile: ARKit/ARCore маркеры
- [ ] CMMS интеграция (запись сессии → наряд)

### Epic 4.3: Security Convergence [P2]
- [ ] CrowdStrike/SentinelOne API integration
- [ ] Physical + Cyber event correlation
- [ ] Unified Dashboard для CISO/CIO

### Epic 4.4: ISO 27001 Certification [P2]
- [ ] Internal audit
- [ ] Stage 1 + Stage 2 audits
- [ ] Получение сертификата
- [ ] Continuous monitoring

---

## 🎯 Success Criteria (Phase 3)

- [ ] ServiceNow integration: incident creation + bi-directional sync
- [ ] Jira integration: service request creation + status sync
- [ ] Toir integration: 152-ФЗ compliant + заявка creation
- [ ] NATS Event Bus: 4 topics, 3 subscribers, JetStream persistence
- [ ] Self-Healing: 5+ playbooks, > 70% auto-fix rate для known issues
- [ ] ISO 27001: 0 HIGH gaps, 2 MEDIUM gaps remaining
- [ ] Code Coverage Go: 65%
- [ ] API Response Time (p95): < 110ms

---

## 📈 Metrics

| Metric | Phase 2 End | Target (Phase 3) | Target (Phase 4) |
|--------|-------------|------------------|------------------|
| Code Coverage (Go) | 52% | 65% | 80% |
| Code Coverage (React) | 32% | 45% | 60% |
| API Response Time (p95) | 125ms | <110ms | <100ms |
| ISO 27001 Gaps Open | 6 | 2 | 0 |
| Lighthouse Score | 92 | 94+ | 95+ |
| Mobile Crash Rate | 0.08% | <0.06% | <0.05% |
| Atlas Integration Uptime | 99.2% | 99.5% | 99.9% |
| Self-Healing Success Rate | N/A | >70% | >85% |
| NATS Event Throughput | N/A | 10k msg/s | 50k msg/s |