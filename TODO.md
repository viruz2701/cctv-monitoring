
# TODO.md — Task Tracker for CCTV Intelligence Platform v3.0

## 🎯 Epics Overview

| Epic | Phase | Priority | Status | Progress |
|------|-------|----------|--------|----------|
| UX Research & ISO Gap Analysis | Phase 0 | P0 | 🟡 In Progress | 50% |
| Headless CMMS & Gatekeeper | Phase 1 | P0 | 🔴 Not Started | 0% |
| AI Intelligence & TCO | Phase 2 | P1 | 🔴 Not Started | 0% |
| Universal CMMS Gateway | Phase 3 | P1 | 🔴 Not Started | 0% |
| Enterprise Scale & ISO Cert | Phase 4 | P2 | 🔴 Not Started | 0% |

---

## 📍 PHASE 0: Foundation & Analysis (Недели 1-2)

### Epic 0.1: UX Research [P0]
- [x] **0.1.1** Анализ существующего UI (WorkOrders, SpareParts, SLADashboard)
- [ ] **0.1.2** Адаптация паттернов Shelf.nu для SpareParts.tsx
- [ ] **0.1.3** Адаптация паттернов Snipe-IT для WorkOrders.tsx
- [ ] **0.1.4** UX-гайдлайн для Mobile App (Gatekeeper UI)

### Epic 0.2: ISO 27001 Gap Analysis [P0]
- [ ] **0.2.1** Аудит `apikey_handlers.go` (хеширование, хранение)
- [ ] **0.2.2** Аудит `SavePushToken` (шифрование в БД)
- [ ] **0.2.3** Аудит Telegram Bot (защита от injection)
- [ ] **0.2.4** Матрица соответствия A.5-A.18

---

## 📍 PHASE 1: Headless CMMS & Gatekeeper (Месяцы 1-2)

### Epic 1.1: CMMS Adapter Framework [P0]
- [ ] **1.1.1** Создать интерфейс `CMMSAdapter` в `backend/internal/cmms/adapter.go`
- [ ] **1.1.2** Реализовать `InternalAdapter` (обертка над существующим `repository.go`)
- [ ] **1.1.3** Реализовать `AtlasAdapter` (REST API клиент для Atlas CMMS)
- [ ] **1.1.4** Реализовать `CMMSRouter` (выбор адаптера, fallback, retry queue)
- [ ] **1.1.5** Интегрировать Router в `cmms_handlers.go` и `mobile_handlers.go`

### Epic 1.2: Gatekeeper Service [P0]
- [ ] **1.2.1** Go Backend: Эндпоинт `/api/v1/mobile/work-orders/{id}/verify`
- [ ] **1.2.2** Go Backend: Проверка GPS (геофенсинг `sites.geofence_polygon`)
- [ ] **1.2.3** Go Backend: Проверка EXIF (время, устройство, блокировка галереи)
- [ ] **1.2.4** Go Backend: AI "До/После" (интеграция с DeepSeek)
- [ ] **1.2.5** Mobile App: Добавить экран верификации в `mobile/src/screens/`
- [ ] **1.2.6** Mobile App: Интеграция камеры с проверкой EXIF/GPS

### Epic 1.3: UX Refresh (Desktop) [P1]
- [ ] **1.3.1** Обновить `SpareParts.tsx` (Shelf.nu style: карточки, индикаторы)
- [ ] **1.3.2** Обновить `WorkOrders.tsx` (Snipe-IT style: таблица, фильтры, bulk)
- [ ] **1.3.3** Обновить `SLADashboard.tsx` (визуализация TTR/TTO)
- [ ] **1.3.4** Добавить страницу `Integrations.tsx` (выбор CMMS адаптера)

### Epic 1.4: ISO 27001 Baseline [P0]
- [ ] **1.4.1** Вынос API Keys в Vault (или шифрование в БД)
- [ ] **1.4.2** Шифрование `push_token` в БД (AES-256)
- [ ] **1.4.3** Audit logging для всех операций с нарядами и верификацией
- [ ] **1.4.4** Rate limiting для `/api/v1/mobile/*`

---

## 📍 PHASE 2: AI Intelligence & TCO (Месяцы 3-5)

### Epic 2.1: Predictive Maintenance [P1]
- [ ] **2.1.1** Расширить `predict.py`: HDD, PoE, Temperature
- [ ] **2.1.2** Go Backend: Эндпоинт `/api/v1/predictions`
- [ ] **2.1.3** CMMS Router: Авто-создание PM-задач (Preventive Maintenance)
- [ ] **2.1.4** Mobile App: Уведомления о предстоящих PM-задачах

### Epic 2.2: TCO Calculator [P1]
- [ ] **2.2.1** Go Backend: Агрегация данных из CMMS (запчасти, часы)
- [ ] **2.2.2** Go Backend: Расчет TCO per device/site
- [ ] **2.2.3** Desktop: Страница `TCO.tsx` (графики, рекомендации)

### Epic 2.3: Voice-to-Report [P1]
- [ ] **2.3.1** Mobile App: Интеграция Whisper API (или on-device STT)
- [ ] **2.3.2** Go Backend: NLP обработка (DeepSeek) -> извлечение сущностей
- [ ] **2.3.3** Авто-обновление CMDB из голосовых отчетов

---

## 📍 PHASE 3: Universal CMMS Gateway (Месяцы 6-9)

### Epic 3.1: Enterprise Adapters [P1]
- [ ] **3.1.1** `ServiceNowAdapter` (SOAP/REST, CMDB CI, Incident/Problem)
- [ ] **3.1.2** `JiraAdapter` (REST v3, Jira Service Management)
- [ ] **3.1.3** `ToirAdapter` (1С:ТОИР REST API, 152-ФЗ)
- [ ] **3.1.4** UI маппинга полей (drag-and-drop в `Integrations.tsx`)

### Epic 3.2: Agentic Self-Healing [P1]
- [ ] **3.2.1** AI Agent: Диагностика (анализ топологии)
- [ ] **3.2.2** AI Agent: Remediation (ISAPI/ONVIF команды через P2P Gateway)
- [ ] **3.2.3** CMMS Router: Авто-закрытие тикетов после успешного self-healing

### Epic 3.3: Bi-directional ITSM [P1]
- [ ] **3.3.1** Webhooks от ServiceNow/Jira -> Go Backend
- [ ] **3.3.2** State Machine: Синхронизация статусов (каждые 5 минут)
- [ ] **3.3.3** Conflict Resolution: Авто-переоткрытие тикетов

---

## 📍 PHASE 4: Enterprise Scale & ISO Cert (Месяцы 10-15)

### Epic 4.1: Multi-tenant SaaS [P2]
- [ ] **4.1.1** Row-level security (PostgreSQL RLS)
- [ ] **4.1.2** Billing tiers (Community/Pro/Enterprise)
- [ ] **4.1.3** Stripe integration

### Epic 4.2: AR Remote Expert [P2]
- [ ] **4.2.1** WebRTC (pion/webrtc) в Go Backend
- [ ] **4.2.2** Mobile App: AR-маркеры (React Native ARKit/ARCore)
- [ ] **4.2.3** Интеграция с CMMS (запись сессии -> наряд)

### Epic 4.3: Security Convergence [P2]
- [ ] **4.3.1** Интеграция с CrowdStrike/SentinelOne API
- [ ] **4.3.2** Корреляция Physical + Cyber events
- [ ] **4.3.3** Unified Dashboard для CISO/CIO

### Epic 4.4: ISO 27001 Certification [P2]
- [ ] **4.4.1** Internal audit
- [ ] **4.4.2** Stage 1 + Stage 2 audit
- [ ] **4.4.3** Получение сертификата

---

## 📊 Priority Definitions
- **P0**: Critical — блокирует релиз, must-have.
- **P1**: High — важно для конкурентоспособности.
- **P2**: Medium — nice-to-have, для лидерства на рынке.

## 🎯 Success Criteria (Phase 1)
- [ ] Atlas CMMS integration работает (создание/обновление нарядов).
- [ ] Gatekeeper блокирует закрытие наряда без GPS/EXIF/AI верификации.
- [ ] UX Refresh завершен (WorkOrders, SpareParts соответствуют референсам).
- [ ] ISO 27001 Baseline: API Keys и Push Tokens зашифрованы, audit log работает.