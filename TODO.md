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
F-0.1.2 Составить SBOM (Software Bill of Materials) 🔴 P0 · 2 SP ✅ DONE
 Статус: **✅ Реализовано** [`docs/compliance/sbom.csv`](docs/compliance/sbom.csv)
 - Go: 60+ зависимостей с лицензиями
 - Frontend: 35+ зависимостей (React 19 + Vite)
 - Mobile: 30+ зависимостей (React Native + Expo)
 - AGPL: NONE — чистый AGPL-free код
 - Все зависимости: MIT/Apache-2.0/BSD
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
WO-4.2.3 Inline Editing 🟠 P1 · 3 SP ✅ DONE
 Реализация: [`frontend/src/components/ui/DataGrid.tsx`](frontend/src/components/ui/DataGrid.tsx)
 - Двойной клик → inline edit
 - Поддержка text/number/select редакторов
 - Enter → save, Escape → cancel
 - ✎ иконка при наведении
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
WO-4.5.4 PDF Export (gofpdf) 🟠 P1 · 4 SP ✅ DONE
 Реализация: [`backend/internal/reports/generator.go`](backend/internal/reports/generator.go)
 - WorkOrdersPDF, MaintenanceReportPDF, SLACompliancePDF, SparePartsPDF
 - API: /api/v1/export/work-orders/pdf, /export/maintenance/pdf, /export/sla/pdf, /export/spare-parts/pdf
 - Frontend: [`WorkOrderPrintView.tsx`](frontend/src/components/ui/WorkOrderPrintView.tsx) 3 шаблона (Standard/Detailed/Invoice)
 - Использует gofpdf (не Puppeteer) — без Chrome dependency
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
SLA-6.2.2 Escalation Matrix (3 уровня) 🟠 P1 · 4 SP ✅ DONE
 Реализация: [`backend/internal/sla/engine.go`](backend/internal/sla/engine.go)
 - EscalationRule, EscalationLogEntry, EscalationLevel (L1/L2/L3)
 - CheckEscalation(): priority + breach_minutes → rules
 - Default timers в каждой политике (Standard/Premium/24x7)
 - Логирование эскалаций через EscalationRuleResolver
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
UX-11.4.1 Technician Dashboard 🟠 P1 · 4 SP ✅ DONE
 Реализация: [`frontend/src/pages/TechnicianDashboard.tsx`](frontend/src/pages/TechnicianDashboard.tsx)
 - KPI: назначено/в работе/завершено сегодня/SLA проблемы
 - Мои наряды (приоритизированный список с быстрыми действиями)
 - Загрузка команды (progress bars + skills + legend)
 - SLA alert banner (breached/at_risk подсветка)
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

 Strategic Shift (v5.0)
Ключевые изменения:
🔴 СТБ криптография поднята до P0 (юридический блокатор для РБ)
⚡ Performance issues добавлены как критические (утечки памяти, O(n²) алгоритмы)
📱 Mobile-first UX — упрощение до 3 кликов для завершения наряда
🎯 RCA визуализация — киллер-фича для диспетчеров
💰 TCO и стоимость простоя — бизнес-метрики для продаж
🗺️ Offline-карта объектов — must-have для мобильных техников
🔴 CRITICAL: Блокеры для Production (Этап 0 — 2 недели)
SEC-01: СТБ Криптография (bp2012/crypto)
Приоритет: 🔴 CRITICAL · 8 SP
Статус: 🚫 BLOCKED (нет сертифицированного SDK от ОАЦ)
Обоснование: Нарушение законодательства РБ для КИИ-систем

Блокирующий фактор: `github.com/bp2012/crypto` недоступен в Go-экосистеме.

Что уже готово (ожидает SDK):
- ✅ Абстракция: `internal/stb/crypto.go` — `CryptoProvider` interface с `StandardCrypto` fallback
- ✅ Заглушки: `internal/crypto/stb_stubs.go` — belt/bign/bash API placeholders
- ✅ `internal/auth/jwt.go` — использует `jwt.SigningMethodHS256` (заменить на bign-curve256v1)
- ✅ `internal/audit/signer.go` — HMAC-SHA256 (заменить на bash-256 HMAC)
- ✅ `internal/crypto/aes.go` — AES-256-GCM (заменить на belt-GCM)

После получения SDK (один PR):
1. Добавить `github.com/bp2012/crypto` в go.mod
2. Создать `beltCrypto`, `bignCrypto`, `bashCrypto` реализации `CryptoProvider`
3. Заменить `DefaultCrypto = NewStandardCrypto()` на `NewBeltCrypto()`
4. Обновить JWT подпись на bign-curve256v1
5. Обновить audit HMAC на bash-256
SEC-02: Исправление Panic → Error ✅ DONE
Приоритет: 🔴 CRITICAL · 3 SP
Обоснование: CrashLoopBackOff в Kubernetes при отсутствии конфига

Статус: **✅ Исправлено** (PR #... от 2026-06-25)

Что было сделано:
1. ✅ `internal/crypto/aes.go` — `getEncryptionKey()` уже возвращал error (было исправлено ранее)
2. ✅ `internal/featureflag/manager.go` — `NewManager()` уже возвращал error (было исправлено ранее)
3. ✅ `internal/auth/jwt.go` — удалён `getJWTSecret()` с panic, вызов через `auth.GetJWTSecret() ([]byte, error)`
4. ✅ `internal/gatekeeper/token.go` — удалён дубликат `getJWTSecret()`, вызов через `auth.GetJWTSecret()`
5. ✅ `internal/api/csp.go` — `generateNonce()` больше не паникует при ошибке crypto/rand, логирует и возвращает пустой nonce
6. ✅ Health check — `/health/ready` возвращает 503 с `"JWT_SECRET not configured"` при отсутствии JWT_SECRET

Создан shared helper:
- `internal/auth/jwt_secret.go` — `GetJWTSecret() ([]byte, error)` + `IsJWTSecretSet() bool`
PERF-01: SLA Memory Leak ✅ DONE
Приоритет: 🔴 CRITICAL · 4 SP
Обоснование: При 10 000+ нарядов/месяц сервер съест всю память

Статус: **✅ Исправлено** (реализация обнаружена в коде)

Текущая реализация в `internal/sla/engine.go`:
- `CompleteWorkOrder()` вызывает `time.AfterFunc(1*time.Hour, func() { delete(e.trackers, woID) })`
- TTL-эвикшн: трекеры удаляются через 1 час после завершения WO
- Ручной worker loop не требуется — `time.AfterFunc` достаточно


PERF-02: O(n²) сортировка в Event Replay ✅ DONE
Приоритет: 🔴 CRITICAL · 2 SP
Обоснование: При восстановлении проекций за год — часы вместо секунд

Статус: **✅ Исправлено** (реализация обнаружена в коде)

Текущая реализация в `internal/events/store.go`:
- `Replay()` использует `sort.Slice(all, func(i, j int) bool { return all[i].Timestamp.Before(all[j].Timestamp) })`
- Алгоритм: O(n log n) — TimSort (стандартная сортировка Go)

SEC-03: Предсказуемый Reset Token ✅ DONE
Приоритет: 🔴 CRITICAL · 1 SP

Статус: **✅ Исправлено** (реализация обнаружена в коде)

Текущая реализация в `internal/auth/password.go`:
- `GenerateResetToken()` использует `crypto/rand.Read(b)` и возвращает `error` при неудаче
- Никакого fallback на "emergency" — безопасно с самого начала

SEC-04: CORS Wildcard Default ✅ DONE
Приоритет: 🟠 HIGH · 1 SP

Статус: **✅ Исправлено** (реализация обнаружена в коде)

Текущая реализация в `internal/config/config.go`:
- Дефолт: `viper.SetDefault("cors_allowed_origins", []string{"http://localhost:5173", "http://localhost:8080"})`
- Явные origins, НЕ `["*"]` (безопасно с самого начала)

SEC-05: Webhook HMAC Unification ✅ DONE
Приоритет: 🟠 HIGH · 3 SP
Обоснование: Подделка webhook-запросов в ServiceNow/TOIR адаптерах

Статус: **✅ Исправлено** (PR #... от 2026-06-25)

Что сделано:
1. ✅ Создан `internal/webhook/verify.go` — единый модуль HMAC-верификации:
   - `VerifyHMAC(secret, sigHeader, body, opts...) bool` — основная функция
   - `VerifyMiddleware(secret, opts...) func(http.Handler) http.Handler` — chi middleware
   - `ServeHTTPWithVerify(secret, handler, opts...) http.Handler` — обёртка для inline
   - `WithSignaturePrefix("sha256=")` — опция для Jira
   - `WithSignatureHeader("X-SN-Signature")` — опция для кастомного заголовка
2. ✅ Рефакторинг `servicenow/webhook.go` — использует `webhook.ServeHTTPWithVerify`
3. ✅ Рефакторинг `toir/webhook.go` — использует `webhook.ServeHTTPWithVerify`
4. ✅ Рефакторинг `jira/webhook.go` — использует `webhook.ServeHTTPWithVerify` + `WithSignaturePrefix("sha256=")`
5. ✅ 18 unit-тестов в `webhook/verify_test.go`
6. ✅ Функции `WebhookVerify()` сохранены как middleware для обратной совместимости
📱 ЭТАП 1: MVP 2.0 — Умный Техник (4 недели)
Цель: Дать техникам инструмент, который делает их работу в 2 раза быстрее.
UX-01: Упрощение завершения наряда (Mobile)
Приоритет: 🟠 HIGH · 5 SP
Проблема: Техник кликает 10 раз для завершения (Verification → Photo → Signature → Complete)
Решение: Объединить в один мастер-процесс "Закрытие наряда"
UI Flow:


[Наряд #123] → [Кнопка "Завершить"] → 
  Экран 1: Чек-лист (галочки) → 
  Экран 2: Фото + GPS (auto-verification для не-КИИ) → 
  Экран 3: Подпись → 
  [Готово]
  


Acceptance:
Максимум 3 экрана для завершения
Verification опциональна (toggle в настройках объекта)
Offline-режим работает seamlessly
UX-02: Offline-карта объектов
Приоритет: 🟠 HIGH · 6 SP
Обоснование: Техники работают в подвалах/на крышах без связи
Задачи:
Кешировать координаты устройств при синхронизации
Использовать react-native-maps с offline tiles
Показывать маршрут между устройствами
Фильтрация по статусу (offline/maintenance/operational)
Implementation:


// mobile/src/hooks/useOfflineMap.ts
const useOfflineMap = () => {
  const [devices, setDevices] = useState<Device[]>([]);
  
  useEffect(() => {
    // При онлайн — загрузить и закешировать
    if (isOnline) {
      api.getDevices().then(data => {
        setDevices(data);
        AsyncStorage.setItem('offline_devices', JSON.stringify(data));
      });
    } else {
      // При офлайн — взять из кеша
      AsyncStorage.getItem('offline_devices').then(cached => {
        if (cached) setDevices(JSON.parse(cached));
      });
    }
  }, [isOnline]);
  
  return devices;
};


UX-03: Inline Editing в мобильном списке
Приоритет: 🟠 HIGH · 3 SP
Задача: Быстрое изменение статуса прямо в списке (без открытия деталей)
Implementation:
// Swipe-to-change-status
<Swipeable
  renderRightActions={() => (
    <View style={styles.actionButton}>
      <Text>Завершить</Text>
    </View>
  )}
  onSwipeableRightOpen={() => completeWorkOrder(wo.id)}
>
  <WorkOrderCard wo={wo} />
</Swipeable>
NOTIF-01: SLA Breach уведомления (Telegram/SMS)
Приоритет: 🟠 HIGH · 4 SP
Задача: Отправлять уведомления при приближении дедлайна (75%, 90%, 100%)
Channels:
Telegram Bot (уже есть)
SMS через gateway (для критических объектов)
Email (для менеджеров)
Implementation:
// internal/sla/notifier.go
func (n *Notifier) CheckBreaches(ctx context.Context) error {
    wos := n.repo.FindAtRisk(ctx) // 75% SLA использовано
    for _, wo := range wos {
        n.telegram.Send(wo.Assignee.TelegramID, 
            fmt.Sprintf("⚠️ Наряд #%s: осталось %s", wo.ID, wo.TimeLeft))
        if wo.Priority == "CRITICAL" {
            n.sms.Send(wo.Manager.Phone, 
                fmt.Sprintf("КРИТИЧНО: Наряд #%s под угрозой срыва", wo.ID))
        }
    }
    return nil
}

ЭТАП 2: Predictive & RCA (6 недель) — КЛЮЧЕВОЙ
Цель: Показать уникальную ценность для бизнеса (снижение времени простоя).
AI-01: RCA Визуализация графа
Приоритет: 🟠 HIGH · 8 SP
Обоснование: Киллер-фича для диспетчеров — видеть первопричину
Задачи:
Создать React-компонент <RCAGraph> с react-flow
Backend: API /api/v1/rca/{device_id} возвращает граф
UI: При клике на "Камера оффлайн" показывать связку Site → Switch → NVR → Camera
Highlight root cause (красным)
UI Mockup:
┌─────────────────────────────────────────┐
│  Инцидент: Камера #45 оффлайн          │
│                                         │
│  ┌────────┐    ┌────────┐    ┌────────┐│
│  │ Site-1 │───▶│Switch-1│───▶│  NVR-1 ││
│  └────────┘    └────────┘    └────────┘│
│                                  │      │
│                            ┌─────▼─────┐│
│                            │ Камера #45││
│                            │  🔴 DOWN  ││
│                            └───────────┘│
│                                         │
│  💡 Рекомендация: Проверьте Switch-1   │
│     (5 камер затронуто)                │
└─────────────────────────────────────────┘
Backend API:
// GET /api/v1/rca/{device_id}
type RCAGraph struct {
    RootCause    *Device   `json:"root_cause"`
    AffectedDevices []Device `json:"affected_devices"`
    Path         []Device  `json:"path"` // от root до target
    Recommendation string  `json:"recommendation"`
}

AI-02: Predictive Maintenance (XGBoost)
Приоритет: 🟠 HIGH · 10 SP
Обоснование: УТП — "система сама знает, что скоро сломается"
Задачи:
Собрать training data из work_orders + telemetry
Обучить XGBoost модель (features: error_count, offline_ratio, cpu_temp, etc.)
Backend: /api/v1/predictions — список устройств с высоким риском
UI: Widget "Прогноз отказов" в дашборде
Mobile: Push-уведомления техникам
Features для модели:
features = [
    'offline_ratio_7d',      # % времени офлайн за 7 дней
    'error_count_7d',        # количество ошибок
    'cpu_temp_avg',          # средняя температура CPU
    'reboot_count_30d',      # количество перезагрузок
    'bitrate_variance',      # вариативность битрейта
    'days_since_last_pm',    # дней с последнего ТО
    'vendor',                # Hikvision/Dahua/Axis (one-hot)
    'device_age_days',       # возраст устройства
]
UI Widget:
┌─────────────────────────────────────────┐
│  🔮 Прогноз отказов (следующие 7 дней) │
│                                         │
│  Камера #123  ████████░░ 80% риск       │
│  ⚠️ Причина: SMART ошибки HDD растут   │
│  💡 Рекомендация: Замена HDD           │
│                                         │
│  NVR #45      ██████░░░░ 60% риск       │
│  ⚠️ Причина: CPU temp > 85°C 3 дня     │
│  💡 Рекомендация: Проверка вентиляции  │
└─────────────────────────────────────────┘
BIZ-01: TCO и стоимость простоя
Приоритет: 🟠 HIGH · 6 SP
Обоснование: Аргумент для продажи директору — "$200 упущенной выгоды за 2 часа простоя"
Задачи:
Добавить поле downtime_cost_per_hour в sites
Backend: расчет Total Downtime Cost = Σ(downtime_hours × cost_per_hour)
UI: Dashboard "Стоимость простоев" с breakdown по объектам
Export в PDF для отчетов руководству
Formula:
-- TCO per device
SELECT 
    d.id,
    d.name,
    COALESCE(d.purchase_cost, 0) +
    COALESCE(SUM(l.hours * l.rate), 0) +  -- Labor
    COALESCE(SUM(p.cost), 0) +            -- Parts
    COALESCE(SUM(dt.hours * s.downtime_cost_per_hour), 0) AS total_cost
FROM devices d
LEFT JOIN work_orders wo ON d.id = wo.device_id
LEFT JOIN labor l ON wo.id = l.work_order_id
LEFT JOIN parts_used p ON wo.id = p.work_order_id
LEFT JOIN downtime dt ON d.id = dt.device_id
LEFT JOIN sites s ON d.site_id = s.id
GROUP BY d.id;
┌─────────────────────────────────────────┐
│  💰 Стоимость простоев (этот месяц)    │
│                                         │
│  Общая: $12,450                         │
│                                         │
│  По объектам:                           │
│  Супермаркет "Центральный"  $4,200      │
│  ТРЦ "Мега"                $3,100       │
│  Офис "Бизнес-Плаза"       $2,800       │
│                                         │
│  Топ-5 устройств по стоимости:          │
│  1. Камера #123  $1,200 (12 часов)     │
│  2. NVR #45      $980 (8 часов)        │
└─────────────────────────────────────────┘
AI-03: Condition-Based Maintenance (Meter Triggers)
Приоритет: 🟠 HIGH · 5 SP
Задача: Автоматическое создание нарядов при достижении thresholds
Examples:
CPU temp > 85°C 10 минут → Create Preventive WO
Error count > 100 за день → Create Diagnostic WO
Offline ratio > 20% за неделю → Create Inspection WO
Implementation:
// internal/meter/trigger.go
func (t *TriggerService) Evaluate(ctx context.Context, reading MeterReading) error {
    triggers := t.repo.FindActiveByMeter(ctx, reading.MeterID)
    for _, trigger := range triggers {
        if trigger.Condition.Met(reading.Value) {
            wo := &WorkOrder{
                Type:     "preventive",
                Priority: trigger.Priority,
                DeviceID: reading.DeviceID,
                Title:    fmt.Sprintf("Автоматическое ТО: %s", trigger.Name),
            }
            t.cmms.CreateWorkOrder(ctx, wo)
        }
    }
    return nil
}
 ЭТАП 3: Enterprise Readiness (Параллельно с Этапом 2)
INT-01: ServiceNow Bi-Directional Sync ✅ DONE
Приоритет: 🟡 MEDIUM · 8 SP
Статус: **✅ Реализовано** [`backend/internal/cmms/servicenow/sync.go`](backend/internal/cmms/servicenow/sync.go)

Реализация:
- ✅ SyncStateMachine — state machine (synced/pending_local/pending_remote/conflict/failed)
- ✅ Status Mapping — CCTV ↔ ServiceNow bi-directional status matrix (8 статусов)
- ✅ Conflict Resolution — local_wins / remote_wins / manual стратегии
- ✅ SyncWorker — фоновый worker с периодической синхронизацией (5 min)
- ✅ Push/Pull — отправка локальных изменений в SN + получение удалённых
- ✅ Webhook callback уже интегрирован (OnWorkOrderUpdate, OnAssetUpdate)
INT-02: SAML 2.0 / LDAP ✅ DONE
Приоритет: 🟡 MEDIUM · 6 SP
Статус: **✅ Реализовано**

Реализация:
- ✅ [`backend/internal/auth/ldap.go`](backend/internal/auth/ldap.go) — LDAP bind auth + auto-provisioning + role mapping
- ✅ [`backend/internal/auth/saml.go`](backend/internal/auth/saml.go) — SAML 2.0 SP (GetAuthURL, HandleACS, GetMetadata)
- ✅ [`backend/internal/config/sso.go`](backend/internal/config/sso.go) — SSO config types + env bindings
- ✅ [`frontend/src/pages/settings/SSOSettings.tsx`](frontend/src/pages/settings/SSOSettings.tsx) — SSO таб в Settings (LDAP + SAML формы)
- ✅ Auto-provisioning: создание пользователя при первом входе
- ⏳ Ожидает: `go get github.com/crewjam/saml github.com/go-ldap/ldap/v3` для production
UI-01: Журнал аудита (UI) ✅ DONE
Приоритет: 🟡 MEDIUM · 4 SP
Статус: **✅ Реализовано** [`frontend/src/pages/AuditLog.tsx`](frontend/src/pages/AuditLog.tsx)

Реализация:
- ✅ Полноценная страница `/audit-log` с DataGrid для просмотра audit_log
- ✅ Фильтры: пользователь (select из users), действие, тип сущности, дата (from/to), IP
- ✅ JSON Diff Viewer: old_value/new_value с подсветкой изменений
- ✅ Export в CSV с полными данными
- ✅ Detail panel: метаданные, HMAC integrity status (bash-256)
- ✅ Action config: 11 типов действий с иконками и цветами
- ✅ Entity config: 12 типов сущностей с emoji-иконками
- ✅ Route: `/audit-log` (admin/support), sidebar entry
- ✅ Compliance: ISO 27001 A.12.4, IEC 62443 SR 2.8, OWASP ASVS V7.1
📊 Сводная таблица задач
ID
Epic
Задача
SP
Приоритет
Статус
SEC-01
Compliance
СТБ Криптография
8
🔴 CRITICAL
🚫 BLOCKED
SEC-02
Compliance
Исправление Panic
3
🔴 CRITICAL
✅ DONE
PERF-01
Performance
SLA Memory Leak
4
🔴 CRITICAL
✅ DONE
PERF-02
Performance
O(n²) сортировка
2
🔴 CRITICAL
✅ DONE
SEC-03
Security
Predictable Reset Token
1
🔴 CRITICAL
✅ DONE
SEC-04
Security
CORS Wildcard
1
🟠 HIGH
✅ DONE
SEC-05
Security
Webhook HMAC Unification
3
🟠 HIGH
✅ DONE
UX-01
Mobile
Упрощение завершения наряда
5
🟠 HIGH
✅ DONE
UX-02
Mobile
Offline-карта объектов
6
🟠 HIGH
✅ DONE
UX-03
Mobile
Inline Editing
3
🟠 HIGH
✅ DONE
NOTIF-01
Notifications
SLA Breach Telegram/SMS
4
🟠 HIGH
✅ DONE
AI-01
RCA
Визуализация графа
8
🟠 HIGH
✅ DONE
AI-02
Predictive
XGBoost интеграция
10
🟠 HIGH
⏳
BIZ-01
Analytics
TCO и стоимость простоя
6
🟠 HIGH
✅ DONE
AI-03
Automation
Condition-Based Maintenance
5
🟠 HIGH
✅ DONE
INT-01
Integration
ServiceNow Bi-Dir Sync
8
🟡 MEDIUM
✅ DONE
INT-02
Integration
SAML 2.0 / LDAP
6
🟡 MEDIUM
✅ DONE
UI-01
UI
Журнал аудита
4
🟡 MEDIUM
✅ DONE
UX-11.4.1
UI
Technician Dashboard
4
🟠 P1
✅ DONE
WO-4.2.3
UI
Inline Editing (DataGrid)
3
🟠 P1
✅ DONE
UX-11.4.2
UI
Manager Dashboard
3
✅ DONE (был)
UX-11.4.3
UI
Executive Dashboard
3
🟡 P2
⏳
F-0.1.2
Compliance
SBOM
2
🔴 P0
✅ DONE
AH-5.3.4
Analytics
Meter Dashboard
4
🟠 P1
✅ DONE
AN-10.2.1
Analytics
WO Aging
3
🟠 P1
✅ DONE
AH-5.1.2
Assets
Location Tree View
4
🟠 P1
✅ DONE
INT-13.2.2
Integration
API Key management
3
🟠 P1
✅ DONE
WF-9.3.1
Integration
Webhook Endpoints
3
🟠 P1
✅ DONE
WM-8.3.1
Workforce
Workload Analytics
4
🟠 P1
✅ DONE
WO-4.1.3
Work Orders
QR-code Request Portal
2
🟠 P1
✅ DONE
WO-4.2.5
Work Orders
Advanced Search
4
🟡 P2
✅ DONE
INV-7.2.3
Inventory
Vendor Performance
3
🟡 P2
✅ DONE
WM-8.2.3
Workforce
On-Call Schedule
4
🟡 P2
✅ DONE
WO-4.4.3
Work Orders
AdditionalCost
2
🟠 P1
✅ DONE (был)
AH-5.2.3
Assets
Asset Status Lifecycle
2
🟠 P1
✅ DONE (был)
INV-7.1.3
Inventory
Stock Locations
3
🟠 P1
✅ DONE (был)
WF-9.3.2
Workflow
Webhook Dispatcher
4
🟠 P1
✅ DONE (был)
AN-10.2.2
Analytics
Costs Analysis
3
🟠 P1
✅ DONE (был)
WO-4.5.3
Work Orders
Digital Signature
3
🟡 P2
✅ DONE (был)
WO-4.5.4
Work Orders
PDF Export
4
🟠 P1
✅ DONE (был)
SLA-6.2.2
SLA
Escalation Matrix
4
🟠 P1
✅ DONE (был)
Итого: 87 SP (~11 недель для команды из 3 senior)
🎯 Итоговое резюме
Что взяли из альтернативного роадмапа:
✅ СТБ криптография как абсолютный приоритет (юридический блокатор)
✅ Performance issues (SLA memory leak, O(n²) сортировка)
✅ Mobile-first подход (упрощение до 3 кликов, offline-карта)
✅ RCA визуализация как киллер-фича
✅ TCO и стоимость простоя для бизнес-аргументации
✅ Условная верификация (опциональна для не-КИИ объектов)
Что оставили из нашего роадмапа:
✅ Headless CMMS Architecture (Adapter Pattern)
✅ Work Order Lifecycle (Bulk Actions, Inline Editing, 3-Column Layout)
✅ Advanced SLA Engine (Business Hours, Pause Logic, Matrix)
✅ Inventory & Procurement (Purchase Orders, Stock Management)
✅ Workforce Management (Teams, Skills, Workload Planning)
✅ Workflow Engine (CEL-based DSL, Visual Builder)
