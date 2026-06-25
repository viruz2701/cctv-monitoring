# Стратегический план развития (Headless CMMS Architecture)

**Версия:** 4.0 · **Дата:** 2026-06-25 · **Статус:** ACTIVE
**Автор:** System Architect
**Зрелость проекта:** ~85% (Production-ready, большинство enterprise features реализованы)

---

## 📜 Архитектурные принципы

| # | Принцип | Обоснование |
|---|---------|-------------|
| P1 | Clean Room Implementation | Не копировать код Atlas/Grash, только паттерны. AGPL-free |
| P2 | Headless CMMS | CCTV Core + pluggable CMMS Layer (Adapter Pattern) |
| P3 | Event-Driven | NATS JetStream для всех межмодульных взаимодействий |
| P4 | Domain-Driven Design | Bounded Contexts: Monitoring, CMMS, Assets, Workforce |
| P5 | Permissive OSS Only | MIT/Apache 2.0 лицензии для зависимостей |
| P6 | API-First | OpenAPI 3.1 spec до написания кода |

---

## 🎯 Strategic Goals (OKR)

| Objective | Key Result | Статус |
|-----------|------------|--------|
| O1: Unique Value Proposition | 3+ уникальные CCTV-only фичи в production | ✅ RCA, Playbook, Gatekeeper, VQ Analyzer |
| O2: Enterprise Readiness | SLA compliance 99%+, enterprise-клиенты | ✅ SLA Engine, Escalation, Audit Log |
| O3: Operational Efficiency | 50% ↓ time-to-resolve, MTTR < 2h | ✅ Mobile Wizard, QR Portal, Smart Dispatch |
| O4: Financial Control | 100% visibility TCO per device | ✅ TCO Dashboard, Cost Analysis |
| O5: Platform Flexibility | 5 CMMS adapters | ✅ Internal, Atlas, ServiceNow, Jira, 1С:ТОИР |

---

## ✅ РЕАЛИЗОВАНО (исторически)

### Epic 0: Foundation & Clean Room
| Задача | SP | Описание |
|--------|----|----------|
| F-0.1.2 SBOM | 2 | CSV со всеми зависимостями, лицензиями — AGPL-free |
| F-0.2.1 DDD Bounded Contexts | 5 | ADR-013: Monitoring, CMMS, Assets, Workforce, Integration |
| F-0.2.2 Event Schema Registry | 4 | 10 JSON schemas в NATS JetStream |
| F-0.2.4 Feature Flags | 3 | 14 seed flags + middleware |
| F-0.3.2 CI/CD Pipeline | 4 | GitHub Actions: lint, test, build, security scan, deploy |
| F-0.3.3 Docker Compose | 2 | Postgres, TimescaleDB, NATS, Redis, MinIO |

### Epic 1: Domain Model Evolution
| Задача | SP | Описание |
|--------|----|----------|
| DM-1.1.1 WorkOrderBase | 3 | 12 статусов, Priority, WorkOrderType |
| DM-1.1.2 Audit trait | 2 | CreatedAt/UpdatedAt/CreatedBy/UpdatedBy |
| DM-1.1.3 Cost (abstract) | 2 | Labor, Parts, Additional Cost |
| DM-1.1.4 State Machine | 5 | looplab/fsm, 12 статусов, матрица переходов |
| DM-1.2.1 WorkOrderHistory | 4 | Immutable timeline событий |
| DM-1.2.2 Event Store | 5 | NATS JetStream + S3 Cold Storage |
| DM-1.2.3 Projection Builder | 4 | WorkOrder, SLA Compliance, Technician Workload |
| DM-1.3.1 WO ↔ Alert M2M | 2 | Migration 019 + API |
| DM-1.3.3 Soft Delete | 2 | Migration 004 |

### Epic 2: CCTV Core
| Задача | SP | Описание |
|--------|----|----------|
| CCTV-2.1.1 Video Quality Metrics | 6 | 7 метрик (blur, brightness, frozen frame...) |
| CCTV-2.1.3 RCA Engine | 5 | BFS по иерархии устройств |
| CCTV-2.2.2 RTSP Health Checker | 3 | TCP + RTSP OPTIONS/DESCRIBE |
| CCTV-2.2.3 Multi-vendor SDK | 8 | Hikvision, Dahua, Axis, TVT, Hisilicon |
| CCTV-2.3.1 Playbook Engine | 5 | YAML-based self-healing |
| CCTV-2.3.2 Human-in-the-loop | 3 | ApprovalManager + Telegram |
| CCTV-2.3.3 Cooldown & Rate Limiting | 2 | |

### Epic 3: CMMS Integration Layer
| Задача | SP | Описание |
|--------|----|----------|
| CMMS-3.1.1 CMMSAdapter interface | 3 | 30+ методов |
| CMMS-3.1.2 Event Dispatcher | 4 | NATS → Adapters |
| CMMS-3.1.3 Retry & DLQ | 3 | FallbackQueue |
| CMMS-3.2.1 InternalAdapter | 8 | PostgreSQL CMMS |
| CMMS-3.2.2 AtlasAdapter | 6 | REST API |
| CMMS-3.2.3 ServiceNowAdapter | 8 | Enterprise ITSM |
| CMMS-3.2.4 JiraAdapter | 6 | Service Management |
| CMMS-3.2.5 ToirAdapter | 3 | 1С:ТОИР Webhooks |
| CMMS-3.3.1 Tenant Router | 2 | Per-tenant adapter selection |
| CMMS-3.3.2 Adapter Health Dashboard | 2 | |

### Epic 4: Work Order Lifecycle
| Задача | SP | Описание |
|--------|----|----------|
| WO-4.1.1 Request Portal | 4 | Публичный submit + reCAPTCHA |
| WO-4.1.2 Approval Workflow | 3 | Submit → Approve → Convert |
| WO-4.1.3 QR-code Request Portal | 2 | 📱 Публичная страница заявок |
| WO-4.2.1 Bulk Actions | 4 | Snipe-IT pattern |
| WO-4.2.2 Quick Filters | 2 | My/Overdue/Critical |
| WO-4.2.3 Inline Editing | 3 | DataGrid double-click edit |
| WO-4.2.5 Advanced Search | 4 | Full-text + facets + saved searches |
| WO-4.3.1 Three-Column Layout | 5 | Atlas CMMS pattern |
| WO-4.3.2 Live SLA Timer | 2 | Color-coded progress |
| WO-4.4.1 TimeEntry | 3 | Start/stop/pause |
| WO-4.4.2 Labor Cost | 2 | Hourly rate × duration |
| WO-4.4.3 AdditionalCost | 2 | Travel, subcontractor |
| WO-4.4.4 Parts Consumption | 3 | Cost snapshot |
| WO-4.4.5 Total Cost Dashboard | 2 | |
| WO-4.5.1 WorkOrderPrintView | 4 | React print component |
| WO-4.5.2 3 Templates | 3 | Standard, Detailed, Invoice |
| WO-4.5.3 Digital Signature | 3 | react-native-signature-canvas (mobile) |
| WO-4.5.4 PDF Export | 4 | gofpdf генератор |

### Epic 5: Asset & Location Hierarchy
| Задача | SP | Описание |
|--------|----|----------|
| AH-5.1.1 parentLocation | 2 | Site hierarchy (migration 015) |
| AH-5.1.2 Location Tree View | 4 | Expandable tree с устройствами |
| AH-5.2.1 parentDevice | 3 | Device hierarchy (migration 016) |
| AH-5.2.2 RCA Engine | 5 | Parent offline → children SUSPENDED |
| AH-5.3.1 Meter entity | 3 | 12 CCTV-метрик |
| AH-5.3.2 Reading table | 2 | TimescaleDB hypertable |
| AH-5.3.3 WorkOrderMeterTrigger | 5 | CPU >85°C → Preventive WO |
| AH-5.3.4 Meter Dashboard | 4 | Time-series charts (Recharts) |

### Epic 6: Advanced SLA Engine
| Задача | SP | Описание |
|--------|----|----------|
| SLA-6.1.1 SLA Policy | 2 | Standard/Premium/24×7 |
| SLA-6.1.2 SLA Matrix | 3 | Priority × Impact |
| SLA-6.1.3 Business Calendar | 4 | Timezone, shifts, holidays |
| SLA-6.1.4 Pause Rules | 2 | ON_HOLD/AWAITING_* паузы |
| SLA-6.2.1 SLA Calculation | 5 | Go worker, 1min batch |
| SLA-6.2.2 Escalation Matrix | 4 | 3 уровня (L1/L2/L3) |
| SLA-6.2.3 Breach Alerts | 2 | Telegram + Email |
| SLA-6.3.1 SLA Dashboard | 3 | KPI cards |
| SLA-6.3.2 Gauge Chart | 2 | |
| SLA-6.3.3 SLA Compliance Report | 3 | PDF/Excel |

### Epic 7: Inventory & Procurement
| Задача | SP | Описание |
|--------|----|----------|
| INV-7.1.2 Custom Fields | 3 | JSONB |
| INV-7.1.4 Stock Adjustments | 2 | Audit trail |
| INV-7.2.1 Vendor entity | 3 | CRUD API |
| INV-7.2.2 Vendor ↔ Part | 2 | Linkage |
| INV-7.2.3 Vendor Performance | 3 | Аналитика (rating, delivery, cost) |
| INV-7.3.1 PurchaseOrder | 4 | 6 статусов |
| INV-7.3.2 PO Line Items | 3 | |
| INV-7.3.3 Auto-PO | 4 | Low-stock trigger |
| INV-7.3.4 Goods Receipt | 3 | |

### Epic 8: Workforce Management
| Задача | SP | Описание |
|--------|----|----------|
| WM-8.1.1 Team entity | 2 | |
| WM-8.1.2 Matrix RBAC | 5 | 5 ролей × 9 сущностей |
| WM-8.2.1 ShiftConfiguration | 3 | |
| WM-8.2.2 User ↔ Shift | 2 | |
| WM-8.2.3 On-Call Schedule | 4 | Недельный график дежурств |
| WM-8.3.1 Workload Analytics | 4 | Heatmap + bar charts |
| WM-8.4.1 Skills matrix | 2 | |
| WM-8.4.2 Certifications | 2 | |

### Epic 9: Automation & Workflows
| Задача | SP | Описание |
|--------|----|----------|
| WF-9.1.1 Workflow entity | 3 | |
| WF-9.1.2 WorkflowCondition DSL | 5 | Built-in evaluator |
| WF-9.1.3 WorkflowAction | 5 | CREATE_WO, NOTIFY, WEBHOOK... |
| WF-9.1.4 Execution Engine | 6 | |
| WF-9.2.1 Critical Alarm → WO | 2 | Built-in template |
| WF-9.2.2 Low Stock → PO | 2 | Built-in template |
| WF-9.2.3 Offline >1h → Escalate | 2 | Built-in template |
| WF-9.3.1 Webhook Endpoints | 3 | CRUD + test + events management |

### Epic 10: Analytics & Reporting
| Задача | SP | Описание |
|--------|----|----------|
| AN-10.1.1 MTBF | 3 | По vendor/device type |
| AN-10.1.2 MTTR | 3 | По technician/team |
| AN-10.1.3 TCO per asset | 4 | Purchase + Labor + Parts + Downtime |
| AN-10.1.4 Asset Overview | 3 | Dashboard |
| AN-10.3.1 AssetDowntime | 4 | Entity |
| AN-10.3.2 Auto-downtime | 3 | AlarmEvent → downtime |
| AN-10.3.3 Downtime Cost | 3 | TCO calculator |
| AN-10.2.1 WO Aging | 3 | Buckets: <1h ... >7d + charts |
| AN-10.2.2 Costs Analysis | 3 | В TotalCostDashboard |

### Epic 11: UX/UI Modernization
| Задача | SP | Описание |
|--------|----|----------|
| UX-11.1.1 Settings Refactor | 4 | 6 вкладок |
| UX-11.1.2 Tabbed Layout | 2 | |
| UX-11.2.1 DataGrid | 6 | Density, columns, resize, search, CSV |
| UX-11.2.2 Apply to 7+ tables | 8 | |
| UX-11.2.3 Density control | 2 | |
| UX-11.2.4 Column visibility | 3 | |
| UX-11.3.1 Import Wizard | 6 | 6-step wizard |
| UX-11.3.2 CSV/JSON | 3 | |
| UX-11.3.3 Export | 3 | |
| UX-11.4.1 Technician Dashboard | 4 | KPI + мои наряды + workload |
| UX-11.4.2 Manager Dashboard | 3 | |
| UX-11.4.3 Executive Dashboard | 3 | uptime, SLA, costs, trends |

### Epic 12: Mobile & Offline
| Задача | SP | Описание |
|--------|----|----------|
| MB-12.1.1 RN + Expo | 3 | Setup |
| MB-12.1.2 Auth | 3 | JWT + refresh |
| MB-12.2.1 QR Scanner | 3 | |
| MB-12.2.2 Photo Capture | 4 | |
| MB-12.2.3 Digital Signature | 3 | SignatureCanvas |
| MB-12.2.4 GPS Verification | 2 | |

### Epic 13: Enterprise Integrations
| Задача | SP | Описание |
|--------|----|----------|
| INT-13.2.1 OpenAPI 3.1 | 3 | 42 endpoints, Swagger UI |
| INT-13.2.2 API Key Management | 3 | Create/revoke/copy/list |
| INT-13.2.3 Rate Limiting | 2 | Per-key |
| INT-13.2.4 GraphQL Endpoint | 6 | Read-only, devices/WOs/sites/techs |

### Cross-Cutting (Security & Compliance)
| Задача | SP | Описание |
|--------|----|----------|
| SEC-02 Fix Panics | 3 | JWT, CSP, Health Check 503 |
| SEC-03 Reset Token | 1 | crypto/rand |
| SEC-04 CORS | 1 | Whitelist |
| SEC-05 Webhook HMAC | 3 | Unified module |
| PERF-01 SLA Memory Leak | 4 | TTL eviction |
| PERF-02 O(n²) Sort | 2 | O(n log n) TimSort |
| UX-01 Complete Wizard | 5 | 3-click mobile complete |
| UX-02 Offline Map | 6 | AsyncStorage caching |
| UX-03 Inline Swipe | 3 | SwipeableCard |
| NOTIF-01 SLA Notifier | 4 | Telegram/SMS/Email |
| AI-01 RCA Graph | 8 | SVG визуализация |
| AI-03 Meter Triggers | 5 | Condition-Based Maintenance |
| BIZ-01 TCO Dashboard | 6 | PDF report + KPI |
| UI-01 Audit Log | 4 | JSON diff + HMAC + CSV |
| INT-01 SN Bi-Dir Sync | 8 | State machine + conflict resolution |
| INT-02 SAML/LDAP | 6 | LDAP bind + SAML SP + SSO UI |

---

## 📋 PENDING (нереализованные задачи)

### 🔴 P0 / CRITICAL
| ID | Задача | SP | Описание |
|----|--------|----|----------|
| F-0.1.1 | IP-аудит кода | 3 | FOSSA/Snyk, remediation plan |
| CCTV-2.1.2 | XGBoost Failure Prediction | 4 | Требует реальных данных для обучения |

### 🟠 P1 / HIGH
| ID | Epic | Задача | SP | Примечание |
|----|------|--------|----|-----------|
| F-0.1.3 | Foundation | Консультация с IP-юристом | 1 | Clean Room methodology |
| F-0.2.3 | Foundation | Multi-tenancy RLS | 4 | ADR-014 + миграции |
| F-0.3.1 | Foundation | Monorepo Turborepo | 3 | @cctv/core, @cctv/web, @cctv/mobile |
| DM-1.3.2 | Domain | WO Relations graph | 4 | React-flow визуализация |
| CCTV-2.2.1 | CCTV Core | ONVIF Profile S/T | 6 | Auto-discovery, PTZ, recording |
| WO-4.2.4 | WO | Column Filters | 3 | DataGrid column filtering |
| AH-5.2.3 | Assets | Asset Status lifecycle | 2 | |
| INV-7.1.3 | Inventory | Stock Locations | 3 | Main/Van-1/Van-2 |
| WM-8.3.2 | Workforce | Capacity Planning heatmap | 4 | |
| WF-9.3.3 | Workflow | Webhook Delivery Log | 2 | |
| AN-10.4.2 | Analytics | DeepSeek AI explanations | 2 | |

### 🟡 P2 / MEDIUM
| ID | Epic | Задача | SP | Примечание |
|----|------|--------|----|-----------|
| AH-5.1.3 | Assets | Floor Plans | 5 | С координатами |
| AN-10.2.3 | Analytics | Time by Week bar | 2 | |
| AN-10.4.3 | Analytics | AI Recommendations | 5 | В создание WO |
| AN-10.4.4 | Analytics | Repair vs Replace | 4 | |
| WM-8.3.3 | Workforce | Smart Assignment | 6 | Skills + location + workload |
| MB-12.3.1 | Mobile | Local DB schema | 4 | WatermelonDB |
| MB-12.3.2 | Mobile | Conflict resolution | 4 | |
| MB-12.3.3 | Mobile | Background sync | 3 | |
| INT-13.3.1 | Integration | SAML SSO IdP | 5 | Crewjam/saml production |

---

## 📊 Сводка

| Метрика | Значение |
|---------|----------|
| **Всего задач в плане** | ~80+ |
| **Реализовано** | ~60 (85%) |
| **Pending** | ~20 |
| **Блокировано** | 1 (SEC-01: СТБ SDK) |
| **Общий SP** | ~87 SP (~11 недель для 3 Senior) |
| **Фактически выполнено** | ~60+ SP ✅ |

**Обновление:** 2026-06-25 — массовое обновление (25 задач за сессию)
