Стратегический план развития (Headless CMMS Architecture)
Версия: 3.2
Дата: 2026-06-24
Статус: ACTIVE
Автор: System Architect
Горизонт планирования: 9 месяцев (Q3 2026 — Q1 2027)
Команда: 3 Senior Full-Stack + 1 ML Engineer + 1 QA
📜 Архитектурные принципы (обязательны к соблюдению)
#
Принцип
Обоснование
P1
Clean Room Implementation
Не копировать код Atlas/Grash, только паттерны. Защита от AGPL-вируса
P2
Headless CMMS
CCTV Core + pluggable CMMS Layer (Adapter Pattern)
P3
Event-Driven
NATS JetStream для всех межмодульных взаимодействий
P4
Domain-Driven Design
Bounded Contexts: Monitoring, CMMS, Assets, Workforce
P5
Permissive OSS Only
MIT/Apache 2.0 лицензии для зависимостей
P6
API-First
OpenAPI 3.1 spec до написания кода
🎯 Strategic Goals (OKR)
Objective
Key Result
Источник паттерна
O1: Unique Value Proposition
3 уникальные CCTV-only фичи в production
Наш R&D
O2: Enterprise Readiness
SLA compliance 99%+, 2 enterprise-клиента
Atlas CMMS
O3: Operational Efficiency
50% ↓ time-to-resolve, MTTR < 2h
Grash CMMS
O4: Financial Control
100% visibility TCO per device
Snipe-IT
O5: Platform Flexibility
3+ CMMS adapters (Internal/Atlas/ServiceNow)
Headless Architecture
🏛️ Epic 0: Foundation & Clean Room Setup
Цель: Заложить юридически чистый фундамент для Headless CMMS.
0.1 Legal & Compliance
F-0.1.1 IP-аудит текущего кода на наличие скопированных фрагментов 🔴 P0 · 3 SP
Tool: FOSSA / Snyk OSS
Acceptance: Отчёт + план remediation
F-0.1.2 Составить SBOM (Software Bill of Materials) 🔴 P0 · 2 SP
Acceptance: CSV со всеми зависимостями и лицензиями
F-0.1.3 Консультация с IP-юристом по Clean Room methodology 🟠 P1 · 1 SP
Acceptance: Письменное заключение
0.2 Architectural Foundation
F-0.2.1 DDD Bounded Contexts design 🏛️ ✅ Done · 5 SP
Contexts: Monitoring, CMMS, Assets, Workforce, Integration
Acceptance: Context Map + ADR-013 (см. docs/adr/ADR-013-ddd-bounded-contexts.md)
F-0.2.2 Event Schema Registry (NATS JetStream) ✅ Done · 4 SP
Tool: JSON Schema (10 built-in schemas: alarms, cmms, predictions, telemetry, audit, system)
Acceptance: SchemaRegistry в internal/events/schema_registry.go
F-0.2.3 Multi-tenancy strategy 🏛️ 🟠 P1 · 4 SP
Choice: Row-level security (RLS) PostgreSQL
Acceptance: ADR-014 + миграции
F-0.2.4 Feature Flags infrastructure ✅ Done · 3 SP
Implementation: internal/featureflag/manager.go + 14 seed flags + middleware
0.3 Developer Experience
F-0.3.1 Monorepo setup (Turborepo) 🟠 P1 · 3 SP
Packages: @cctv/core, @cctv/cmms, @cctv/web, @cctv/mobile
F-0.3.2 CI/CD Pipeline (GitHub Actions) ✅ Done · 4 SP
Implementation: .github/workflows/{ci,deploy}.yml
F-0.3.3 Local Dev Environment (Docker Compose) ✅ Done · 2 SP
Services: Postgres, TimescaleDB, NATS, Redis, MinIO — реализован в docker-compose.yml
🎯 Epic 1: Domain Model Evolution (DDD Refactoring)
Цель: Переход от плоской модели к богатой доменной модели.
1.1 Aggregates & Entities
DM-1.1.1 WorkOrderBase (abstract) ✅ Done · 3 SP
Поля: title, priority, status, assignee, dueDate, createdAt
Acceptance: models/base.go — 12 статусов, Priority, WorkOrderType
DM-1.1.2 Audit trait (createdAt/By, updatedAt/By) ✅ Done · 2 SP
Acceptance: CreatedAt/UpdatedAt/CreatedBy/UpdatedBy в моделях
DM-1.1.3 Cost (abstract) для Labor/Parts/Additional ✅ Done · 2 SP
Acceptance: CostBase + Labor + AdditionalCost в models/base.go
DM-1.1.4 State Machine (looplab/fsm — MIT) ✅ Done · 5 SP
Statuses: 12 статусов (REQUESTED → CLOSED)
Transitions: валидация через матрицу переходов
Implementation: internal/models/state_machine.go
1.2 Event Sourcing (для Work Orders)
DM-1.2.1 WorkOrderHistory — immutable timeline ✅ Done · 4 SP
Acceptance: Каждый переход сохраняется с metadata
Implementation: internal/events/store.go
DM-1.2.2 Event Store на базе NATS JetStream ✅ Done · 5 SP
Retention: 1 год hot (NATS), 5 лет cold (S3/MinIO)
Implementation: internal/events/{store,cold_storage,schema_registry}.go
DM-1.2.3 Projection Builder (read-model из events) ✅ Done · 4 SP
Implementation: internal/events/{projection,work_order_projection,sla_projection,technician_projection}.go
Projections: WorkOrder, SLA Compliance, Technician Workload
1.3 Relations & Graph
DM-1.3.1 WorkOrder ↔ Alert (Many-to-Many) ✅ Done · 2 SP
Implementation: migration 019 + API POST/DELETE/GET /work-orders/{id}/alerts
DM-1.3.2 Work Order Relations (parent/child/blocked_by/duplicate) 🟠 P1 · 4 SP
Acceptance: UI визуализация графа (react-flow)
DM-1.3.3 Soft Delete + Archive pattern ✅ Done · 2 SP
Implementation: migration 004_device_soft_delete
🎯 Epic 2: CCTV Core (Уникальное конкурентное преимущество)
Цель: Усилить monitoring — то, чего НЕТ у Atlas/Grash.
2.1 Telemetry & AI
CCTV-2.1.1 Video quality metrics (image analysis) ✅ Done · 6 SP
Метрики: blur, brightness, contrast, black screen, frozen frame, noise, blockiness
Acceptance: Go imaging (no OpenCV), 7 quality metrics, overall 0-100 score
Implementation: internal/videoq/analyzer.go
CCTV-2.1.2 XGBoost Failure Prediction (улучшение) 🔴 P0 · 4 SP
Acceptance: AUC > 0.85, объяснения через DeepSeek
CCTV-2.1.3 Root Cause Analysis engine ✅ Done · 5 SP
Алгоритм: BFS по иерархии устройств
Acceptance: "Switch-1 down → 5 cameras and 2 NVRs affected"
Implementation: internal/rca/engine.go
2.2 Protocol Support
CCTV-2.2.1 ONVIF Profile S/T full support 🟠 P1 · 6 SP
Acceptance: Auto-discovery, PTZ, recording
CCTV-2.2.2 RTSP health checker ✅ Done · 3 SP
Acceptance: TCP connect + RTSP OPTIONS/DESCRIBE telemetry, frozen stream detection, health score
Implementation: internal/rtspcheck/checker.go (без видеопотоков)
CCTV-2.2.3 Multi-vendor SDK integration (Hikvision, Dahua, Axis) ✅ Done · 8 SP
Implementation: internal/protocols/{hikvision,dahua,tvt,hisilicon}.go
2.3 Self-Healing Agent
CCTV-2.3.1 Playbook Engine (YAML-based) ✅ Done · 5 SP
Actions: reboot, SSH restart, ISAPI reset
Implementation: backend/playbooks/{camera_diagnostic,hikvision_diagnostic,reboot_camera}.yml
CCTV-2.3.2 Human-in-the-loop Approval ✅ Done · 3 SP
Acceptance: ApprovalManager в internal/agent/approval.go + Telegram интеграция
CCTV-2.3.3 Cooldown & rate limiting ✅ Done · 2 SP
🎯 Epic 3: CMMS Integration Layer (Headless Architecture)
Цель: Adapter Pattern для любой CMMS.
3.1 Core Adapter Interface
CMMS-3.1.1 CMMSAdapter interface (Go) ✅ Done · 3 SP
type CMMSAdapter interface {
    CreateWorkOrder(ctx, wo) error
    UpdateStatus(ctx, id, status) error
    GetAssetHierarchy(ctx, siteID) (Tree, error)
    // ... 30+ методов
}
Implementation: internal/cmms/adapter.go
CMMS-3.1.2 Event Dispatcher (NATS → Adapters) ✅ Done · 4 SP
Implementation: internal/cmms/dispatcher.go
CMMS-3.1.3 Retry & Dead Letter Queue ✅ Done · 3 SP
Implementation: FallbackQueue в internal/cmms/dispatcher.go
3.2 Built-in Adapters
CMMS-3.2.1 InternalAdapter (lightweight CMMS) ✅ Done · 8 SP
БД: отдельные таблицы cmms_work_orders, cmms_assets
Acceptance: Full CRUD + API — реализован в internal/db/cmms_repository.go
CMMS-3.2.2 AtlasAdapter (REST API) ✅ Done · 6 SP
Acceptance: Bi-directional sync с Atlas CMMS
Implementation: internal/cmms/atlas_adapter.go
CMMS-3.2.3 ServiceNowAdapter ✅ Done · 8 SP
Acceptance: Enterprise clients
Implementation: internal/cmms/servicenow/{adapter,client,mapper,webhook}.go
CMMS-3.2.4 JiraServiceManagementAdapter ✅ Done · 6 SP
Implementation: internal/cmms/jira/{adapter,client,mapper,webhook}.go
CMMS-3.2.5 WebhookAdapter (generic, 1С:ТОИР) ✅ Done · 3 SP
Implementation: internal/cmms/toir/{adapter,client,mapper,webhook}.go
3.3 Configuration & Routing
CMMS-3.3.1 Per-tenant adapter selection ✅ Done · 2 SP
Config: cmms.adapter: "internal" | "atlas" | "servicenow"
Implementation: internal/cmms/factory/factory.go
CMMS-3.3.2 Adapter Health Dashboard ✅ Done · 2 SP
🎯 Epic 4: Work Order Lifecycle
Цель: Enterprise WO management (вдохновлено Atlas/Grash).
4.1 Work Requests Portal
WO-4.1.1 Request entity + public submit endpoint ✅ Done · 4 SP
Acceptance: Без авторизации, с reCAPTCHA
WO-4.1.2 Approval workflow (Submit → Approve → Convert) ✅ Done · 3 SP
WO-4.1.3 QR-code на устройстве → Request Portal 🟠 P1 · 2 SP
4.2 Enhanced Management
WO-4.2.1 Bulk Actions (Snipe-IT pattern) ✅ Done · 4 SP
Implementation: backend POST /api/v1/work-orders/bulk + frontend BulkActionBar
WO-4.2.2 Quick Filters (My/Overdue/Critical) ✅ Done · 2 SP
Implementation: frontend QuickFilterBar в WorkOrders.tsx
WO-4.2.3 Inline Editing 🟠 P1 · 3 SP
WO-4.2.4 Column Filters 🟠 P1 · 3 SP
WO-4.2.5 Advanced Search (full-text + facets) 🟡 P2 · 4 SP
4.3 Three-Column Layout (Atlas pattern)
WO-4.3.1 Redesign WorkOrderDetail ✅ Done · 5 SP
Left (4/12): Status badge, Priority, Type, Live SLA Timer, Assignee, Timeline
Center (5/12): Checklist (drag&drop), Notes, Photos (annotation, before/after)
Right (3/12): Asset info, Location, Actions (start/complete/cancel), Parts
WO-4.3.2 Live SLA Timer с color-coded progress ✅ Done · 2 SP
Implementation: frontend/src/components/ui/LiveSLATimer.tsx
4.4 Time & Cost Tracking
WO-4.4.1 TimeEntry (start/stop/pause) ✅ Done · 3 SP
Implementation: migration 014 + backend API + frontend таймер в WorkOrderDetail
WO-4.4.2 Labor (hourly rate × duration) ✅ Done · 2 SP
Implementation: labor cost расчёт в time_entries + API GET /work-orders/{id}/labor-cost
WO-4.4.3 AdditionalCost (travel, subcontractor) ✅ Done · 2 SP
WO-4.4.4 Parts Consumption с cost snapshot ✅ Done · 3 SP
Implementation: POST /work-orders/{id}/parts-with-cost + cost snapshot в parts_used JSONB
WO-4.4.5 Total Cost Dashboard ✅ Done · 2 SP
4.5 Printable Work Orders
WO-4.5.1 WorkOrderPrintView component ✅ Done · 4 SP
Implementation: frontend/src/components/ui/WorkOrderPrintView.tsx
WO-4.5.2 3 шаблона (Standard/Detailed/Invoice) ✅ Done · 3 SP
WO-4.5.3 Digital Signature pad 🟡 P2 · 3 SP
WO-4.5.4 PDF Export (Puppeteer) 🟠 P1 · 4 SP
🎯 Epic 5: Asset & Location Hierarchy
Цель: Parent-child отношения для root-cause analysis.
5.1 Location Hierarchy
AH-5.1.1 parentLocation в Site entity ✅ Done · 2 SP
Tree: Building → Floor → Room → Rack — ParentLocationID в Site (migration 015)
AH-5.1.2 Location Tree View (expandable, lazy) 🟠 P1 · 4 SP
AH-5.1.3 Floor Plans с координатами 🟡 P2 · 5 SP
5.2 Asset (Device) Hierarchy
AH-5.2.1 parentDevice + type discriminator ✅ Done · 3 SP
Tree: Site → Switch → NVR → Camera — ParentDeviceID + HierarchyLevel (migration 016)
AH-5.2.2 Root Cause Analysis engine ✅ Done · 5 SP
Acceptance: Parent offline → children SUSPENDED — internal/rca/engine.go
AH-5.2.3 Asset Status lifecycle 🟠 P1 · 2 SP
5.3 Meter & Telemetry Triggers (Grash pattern)
AH-5.3.1 Meter entity ✅ Done · 3 SP
CCTV-метры: 10 видов (bitrate, CPU temp, error count, offline_ratio и др.)
AH-5.3.2 Reading table (TimescaleDB hypertable) ✅ Done · 2 SP
AH-5.3.3 WorkOrderMeterTrigger ✅ Done · 5 SP
Rule: "CPU > 85°C 10min → Create Preventive WO"
Implementation: internal/meter/{entity,trigger}.go + migration 011_meter_tables
AH-5.3.4 Meter Dashboard (time-series charts) 🟠 P1 · 4 SP
🎯 Epic 6: Advanced SLA Engine
Цель: Замена плоской SLAConfig на enterprise SLA-движок.
6.1 SLA Policy Architecture
SLA-6.1.1 SLA_Policy (Standard/Premium/24×7) ✅ Done · 2 SP
SLA-6.1.2 SLA_Matrix (Priority × Impact) ✅ Done · 3 SP
SLA-6.1.3 Business_Calendar per Site ✅ Done · 4 SP
Acceptance: Timezone, work shifts, holidays, exceptions
SLA-6.1.4 SLA_Pause_Rules (statuses для паузы) ✅ Done · 2 SP
Implementation: internal/sla/{policy,engine}.go + migration 010_sla_engine
6.2 SLA Runtime
SLA-6.2.1 SLA Calculation Service (Go worker) ✅ Done · 5 SP
Batch: every 1 min, Redis cache
SLA-6.2.2 Escalation Matrix (3 уровня) 🟠 P1 · 4 SP
SLA-6.2.3 SLA Breach alerts (email + Telegram) ✅ Done · 2 SP
Implementation: BreachCheckLoop в sla/worker.go + Telegram уведомления через telegram.Bot
6.3 SLA Analytics
SLA-6.3.1 SLA Dashboard с KPI cards ✅ Done · 3 SP
Implementation: KPI cards (Total, Within SLA, Breached, At Risk) в SLADashboard.tsx
SLA-6.3.2 Gauge chart realtime ✅ Done · 2 SP
SLA-6.3.3 SLA Compliance Report (PDF/Excel) ✅ Done · 3 SP
🎯 Epic 7: Inventory & Procurement
Цель: Полный цикл управления запчастями.
7.1 Spare Parts Enhancement
INV-7.1.1 Part Categories (иерархические) 🟠 P1 · 2 SP
INV-7.1.2 Custom Fields для Parts ✅ Done · 3 SP
Choice: JSONB (не EAV) для гибкости — реализовано
INV-7.1.3 Stock Locations (Main/Van-1/Van-2) 🟠 P1 · 3 SP
INV-7.1.4 Stock Adjustments с audit trail ✅ Done · 2 SP
Implementation: таблица stock_adjustments + audit trail + API
7.2 Vendor Management
INV-7.2.1 Vendor entity ✅ Done · 3 SP
Implementation: migration 021 + CRUD API /api/v1/vendors
INV-7.2.2 Vendor ↔ Part linkage ✅ Done · 2 SP
INV-7.2.3 Vendor Performance analytics 🟡 P2 · 3 SP
7.3 Purchase Orders (Grash pattern)
INV-7.3.1 PurchaseOrder entity ✅ Done · 4 SP
States: DRAFT → SENT → APPROVED → RECEIVED → CLOSED → CANCELLED
INV-7.3.2 PO Line Items с привязкой к Part ✅ Done · 3 SP
INV-7.3.3 Auto-PO при low-stock ✅ Done · 4 SP
INV-7.3.4 Goods Receipt (приходный ордер) ✅ Done · 3 SP
Implementation: internal/purchase/purchase.go (PO, LineItems, AutoPO, GoodsReceipt)
🎯 Epic 8: Workforce Management
Цель: Управление командой техников.
8.1 Teams & Roles
WM-8.1.1 Team entity ✅ Done · 2 SP
WM-8.1.2 Matrix RBAC ✅ Done · 5 SP
Acceptance: Role × Permission × Entity (5 roles × 9 entities)
Implementation: internal/workforce/{entity,rbac}.go
8.2 Shift Configuration
WM-8.2.1 ShiftConfiguration entity ✅ Done · 3 SP
WM-8.2.2 User ↔ Shift assignment ✅ Done · 2 SP
WM-8.2.3 On-Call rotation scheduler 🟡 P2 · 4 SP
8.3 Workload & Capacity Planning
WM-8.3.1 Workload analytics 🟠 P1 · 4 SP
WM-8.3.2 Capacity Planning view (heatmap) 🟠 P1 · 4 SP
WM-8.3.3 Smart Assignment (skills + location + workload) 🟡 P2 · 6 SP
8.4 Skills & Certifications
WM-8.4.1 Skills matrix ✅ Done · 2 SP
WM-8.4.2 Certifications с expiration ✅ Done · 2 SP
🎯 Epic 9: Automation & Workflows
Цель: Визуальный конструктор автоматизаций (Grash pattern).
9.1 Workflow Engine
WF-9.1.1 Workflow entity ✅ Done · 3 SP
WF-9.1.2 WorkflowCondition (DSL) ✅ Done · 5 SP
Choice: Built-in evaluator (CEL-ready), operators: eq/neq/gt/gte/lt/lte/contains/matches
WF-9.1.3 WorkflowAction ✅ Done · 5 SP
Actions: CREATE_WO, NOTIFY, UPDATE_STATUS, WEBHOOK, ASSIGN, ESCALATE
WF-9.1.4 Workflow Execution Engine ✅ Done · 6 SP
Implementation: internal/workflow/{entity,eval,engine}.go
9.2 Built-in Templates
WF-9.2.1 "Critical alarm → Emergency WO" ✅ Done · 2 SP
WF-9.2.2 "Low stock → Create PO" ✅ Done · 2 SP
WF-9.2.3 "Device offline > 1h → Escalate" ✅ Done · 2 SP
9.3 Webhooks (Outgoing)
WF-9.3.1 WebhookEndpoint entity 🟠 P1 · 3 SP
WF-9.3.2 Dispatcher с exponential backoff 🟠 P1 · 4 SP
WF-9.3.3 Delivery Log 🟠 P1 · 2 SP
🎯 Epic 10: Analytics & Reporting
Цель: BI-дашборды enterprise-уровня.
10.1 Asset Analytics
AN-10.1.1 MTBF by vendor/device type ✅ Done · 3 SP
Implementation: mv_device_reliability + API GET /api/v1/analytics/reliability
AN-10.1.2 MTTR by technician/team ✅ Done · 3 SP
AN-10.1.3 TCO per asset ✅ Done · 4 SP
Formula: Purchase + Labor + Parts + Downtime — реализовано в mv_tco_per_device
AN-10.1.4 Asset Overview dashboard ✅ Done · 3 SP
10.2 Work Order Analytics
AN-10.2.1 WO Aging (Incomplete by Asset/User) 🟠 P1 · 3 SP
AN-10.2.2 Costs Analysis (by week/category) 🟠 P1 · 3 SP
AN-10.2.3 Time by Week (stacked bar) 🟡 P2 · 2 SP
10.3 Downtime Tracking
AN-10.3.1 AssetDowntime entity ✅ Done · 4 SP
AN-10.3.2 Auto-downtime при AlarmEvent ✅ Done · 3 SP
AN-10.3.3 Downtime Cost calculation ✅ Done · 3 SP
Implementation: internal/downtime/downtime.go (TCO calculator, MTTR, cost per device type)
10.4 Predictive Maintenance (Phase 3)
AN-10.4.1 XGBoost model v2 🔴抛光 P0 · 3 SP
AN-10.4.2 DeepSeek AI explanations 🟠 P1 · 2 SP
AN-10.4.3 AI recommendations in WO creation 🟡 P2 · 5 SP
AN-10.4.4 Repair vs Replace analysis 🟡 P2 · 4 SP
🎯 Epic 11: UX/UI Modernization
Цель: Зрелые паттерны из Snipe-IT/Atlas.
11.1 Settings.tsx Refactor
UX-11.1.1 Разбить 953-строчный файл на 6 вкладок ✅ Done · 4 SP
Current: 167 строк, 6 вкладок (General, Notifications, Security, Services, Integrations, AtlasCMS)
UX-11.1.2 Tabbed interface с lazy loading ✅ Done · 2 SP
Implementation: frontend/src/pages/Settings.tsx + 6 компонентов в settings/
11.2 DataGrid Pattern
UX-11.2.1 Создать <DataGrid> wrapper ✅ Done · 6 SP
Implementation: DataGrid.tsx — variant, stickyHeader, emptyIcon, rowClassName, ARIA
UX-11.2.2 Применить ко всем таблицам (7+) ✅ Done · 8 SP
UX-11.2.3 Density control ✅ Done · 2 SP
UX-11.2.4 Column visibility & reordering ✅ Done · 3 SP
Features added: pagination, compact/standard/comfortable density, column drag-reorder, column resize, search, CSV export, selectable rows
11.3 Import/Export Wizard
UX-11.3.1 Универсальный Import Wizard ✅ Done · 6 SP
Steps: Upload → Preview → Match → Review → Import → Complete
UX-11.3.2 CSV/JSON support ✅ Done · 3 SP
UX-11.3.3 Export с выбором колонок ✅ Done · 3 SP
Implementation: frontend/src/components/ui/ImportWizard.tsx
11.4 Dashboard Widgets
UX-11.4.1 Technician Dashboard 🟠 P1 · 4 SP
UX-11.4.2 Manager Dashboard ✅ Done · 3 SP
UX-11.4.3 Executive Dashboard 🟡 P2 · 3 SP
🎯 Epic 12: Mobile & Offline
Цель: Полноценная работа в полях.
12.1 Mobile App Foundation
MB-12.1.1 React Native + Expo setup ✅ Done · 3 SP
MB-12.1.2 Unified Auth (JWT + refresh tokens) ✅ Done · 3 SP
MB-12.1.3 Offline-first architecture 🏛️ 🟡 P2 · 5 SP
Tool: WatermelonDB (MIT) или PowerSync
12.2 Field Features
MB-12.2.1 QR/Barcode scanner ✅ Done · 3 SP
MB-12.2.2 Photo capture + annotation ✅ Done · 4 SP
MB-12.2.3 Digital Signature 🟠 P1 · 3 SP
MB-12.2.4 GPS location verification ✅ Done · 2 SP
12.3 Offline Sync
MB-12.3.1 Local DB schema 🟡 P2 · 4 SP
MB-12.3.2 Conflict resolution strategy 🟡 P2 · 4 SP
MB-12.3.3 Background sync queue 🟡 P2 · 3 SP
🎯 Epic 13: Enterprise Integrations
Цель: Выход на enterprise-сегмент.
13.1 ITSM Integrations
INT-13.1.1 ServiceNow bi-directional sync 🟡 P2 · 8 SP
INT-13.1.2 Jira Service Desk 🟡 P2 · 6 SP
INT-13.1.3 Microsoft Teams webhook 🟡 P2 · 2 SP
13.2 API Gateway
INT-13.2.1 OpenAPI 3.1 auto-generation ✅ Done · 3 SP
Implementation: internal/api/openapi.go (42 endpoints, Swagger UI, JSON spec)
INT-13.2.2 API Key management 🟠 P1 · 3 SP
INT-13.2.3 Rate limiting per key ✅ Done · 2 SP
INT-13.2.4 GraphQL read-only endpoint 🟡 P2 · 6 SP
13.3 SSO & Identity
INT-13.3.1 SAML 2.0 support 🟡 P2 · 5 SP
INT-13.3.2 OIDC / Azure AD 🟡 P2 · 4 SP
INT-13.3.3 LDAP integration 🟠 P1 · 3 SP
📅 Phased Delivery Plan
🟢 Phase 1: Foundation (Недели 1-8, Q3 2026)
Цель: Стабильный MVP с уникальными CCTV-фичами.
CMMS-3.1.2 Event Dispatcher (NATS → Adapters) ✅ Done · 4 SP
CMMS-3.1.3 Retry & Dead Letter Queue ✅ Done · 3 SP
3.2 Built-in Adapters
CMMS-3.2.1 InternalAdapter (lightweight CMMS) ✅ Done · 8 SP
БД: отдельные таблицы cmms_work_orders, cmms_assets
Acceptance: Full CRUD + API
CMMS-3.2.2 AtlasAdapter (REST API) ✅ Done · 6 SP
Acceptance: Bi-directional sync с Atlas CMMS
CMMS-3.2.3 ServiceNowAdapter ✅ Done · 8 SP
Acceptance: Enterprise clients
CMMS-3.2.4 JiraServiceManagementAdapter ✅ Done · 6 SP
CMMS-3.2.5 WebhookAdapter (generic, 1С:ТОИР) ✅ Done · 3 SP
3.3 Configuration & Routing
CMMS-3.3.1 Per-tenant adapter selection ✅ Done · 2 SP
Config: cmms.adapter: "internal" | "atlas" | "servicenow"
CMMS-3.3.2 Adapter Health Dashboard 🟠 P1 · 2 SP
🎯 Epic 4: Work Order Lifecycle
Цель: Enterprise WO management (вдохновлено Atlas/Grash).
4.1 Work Requests Portal
WO-4.1.1 Request entity + public submit endpoint ✅ Done · 4 SP
Acceptance: Без авторизации, с reCAPTCHA
WO-4.1.2 Approval workflow (Submit → Approve → Convert) ✅ Done · 3 SP
WO-4.1.3 QR-code на устройстве → Request Portal 🟠 P1 · 2 SP
4.2 Enhanced Management
WO-4.2.1 Bulk Actions (Snipe-IT pattern) ✅ Done · 4 SP
WO-4.2.2 Quick Filters (My/Overdue/Critical) ✅ Done · 2 SP
WO-4.2.3 Inline Editing 🟠 P1 · 3 SP
WO-4.2.4 Column Filters 🟠 P1 · 3 SP
WO-4.2.5 Advanced Search (full-text + facets) 🟡 P2 · 4 SP
4.3 Three-Column Layout (Atlas pattern)
WO-4.3.1 Redesign WorkOrderDetail ✅ Done · 5 SP
Left: Status, SLA, Assignee
Center: Checklist, Notes, Photos
Right: Asset, Location, Parts
WO-4.3.2 Live SLA Timer с color-coded progress ✅ Done · 2 SP
4.4 Time & Cost Tracking
WO-4.4.1 TimeEntry (start/stop/pause) ✅ Done · 3 SP
WO-4.4.2 Labor (hourly rate × duration) ✅ Done · 2 SP
WO-4.4.3 AdditionalCost (travel, subcontractor) 🟠 P1 · 2 SP
WO-4.4.4 Parts Consumption с cost snapshot ✅ Done · 3 SP
WO-4.4.5 Total Cost Dashboard 🟠 P1 · 2 SP
4.5 Printable Work Orders
WO-4.5.1 WorkOrderPrintView component ✅ Done · 4 SP
WO-4.5.2 3 шаблона (Standard/Detailed/Invoice) 🟠 P1 · 3 SP
WO-4.5.3 Digital Signature pad 🟡 P2 · 3 SP
WO-4.5.4 PDF Export (Puppeteer) 🟠 P1 · 4 SP
Next Review: 2026-07-01 (через 1 неделю)
Owner: System Architect + Product Owner
Approval: Engineering Director + CTO
Документ является living document. Все изменения фиксируются в этом файле и в git history с указанием причины. Версионирование через semantic versioning (Major.Minor.Patch).
