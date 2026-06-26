# CCTV Health Monitor — Status

## ✅ Все задачи выполнены

### Phase 2 (P2)
| Задача | Статус | Ключевые файлы |
|--------|--------|----------------|
| P2-1.1: Real ML Model Integration | ✅ Done | `internal/ml/prediction_service.go` |
| P2-1.2: AI Assistant Chat | ✅ Done | `internal/api/ai_routes.go` |
| P2-2.1: Workflow Builder UI | ✅ Done | `frontend/src/components/workflow/` |
| P2-2.3: Resource Planning Calendar | ✅ Done | `frontend/src/components/planning/` |
| P2-3.1: Webhook Builder UI | ✅ Done | `frontend/src/components/webhooks/` |
| P2-3.2: OAuth2 для External Adapters | ✅ Done | `internal/oauth2/` |
| P2-3.3: Webhook Retry & Delivery Logs | ✅ Done | `internal/webhook/delivery.go`, `pg_store.go` |

### Foundation & Fixes
| Задача | Статус | Ключевые файлы |
|--------|--------|----------------|
| SEC-02: Panic → Error | ✅ Done | `internal/auth/jwt_secret.go` |
| Map iframe: window.open → modal | ✅ Done | `frontend/src/components/ui/MapModal.tsx` |
| UX-01: Mobile Work Order Completion Wizard | ✅ Done | `mobile/src/components/CompleteWorkOrderWizard.tsx` |

### Phase 3
| Задача | Статус | Ключевые файлы |
|--------|--------|----------------|
| P3-2: Audit Trail Compliance | ✅ Done | `internal/audit/chain.go` |
| P3-1: Multi-Region Geo-Redundancy | ✅ Code-level | `internal/multiregion/region.go` |

---

## P3-2: Audit Trail Compliance (ISO 27001 A.12.4)

### DB Migration `034_audit_chain`
- `prev_hash` — HMAC предыдущей записи (tamper detection chain)
- `trace_id` — сквозной идентификатор
- `audit_log_archive` + `archive_audit_logs(N)` — 7-year retention
- `verify_audit_chain()` — проверка целостности цепочки

### Backend
- [`backend/internal/audit/chain.go`](backend/internal/audit/chain.go) — `ChainStore`: `InsertWithChain()`, `VerifyEntry()`, `GetComplianceReport()`
- [`backend/internal/api/audit_handlers.go`](backend/internal/api/audit_handlers.go) — 4 эндпоинта

### API Endpoints
| Endpoint | Описание |
|----------|----------|
| `GET /api/v1/audit/log` | Журнал с пагинацией/фильтрацией |
| `GET /api/v1/audit/verify` | Проверка HMAC + chain integrity |
| `GET /api/v1/audit/compliance` | Compliance-отчёт |
| `POST /api/v1/audit/archive` | Архивация записей (>7 лет) |

---

## P3-1: Multi-Region Geo-Redundancy

### ADR-018
[`docs/adr/ADR-018-multi-region-architecture.md`](docs/adr/ADR-018-multi-region-architecture.md)
- Active-Passive per tenant, NATS mirror, async WAL, S3 CRR
- 4 региона: EU-Central, CIS-East, MENA-Gulf, SEA-Hub

### DB Migration `035_tenant_regions`
- `tenant_regions` — привязка тенантов к регионам
- `users.region` — multi-region routing

### Multi-Region Package
[`backend/internal/multiregion/region.go`](backend/internal/multiregion/region.go)

| Компонент | Описание |
|-----------|----------|
| `PGTenantRegionStore` | PostgreSQL CRUD для tenant_regions |
| `FailoverService` | Semi-auto failover (NATS → DB → routing) |
| `NATSMirrorSetup` | Программатор NATS JetStream mirror streams |

### Admin API Endpoints
| Endpoint | Описание |
|----------|----------|
| `GET /api/v1/admin/regions` | Все tenant-region mapping |
| `GET /api/v1/admin/regions/{id}` | Region конкретного тенанта |
| `PUT /api/v1/admin/regions/{id}` | Привязка тенанта к региону |
| `POST /api/v1/admin/failover/{id}` | Execute failover |
| `POST /api/v1/admin/failover/{id}/rollback` | Rollback failover |
| `GET /api/v1/admin/dr/status` | Общий DR статус |

### Region-aware Health
- `healthResponse.Region` в `/health/live` и `/health/ready`
- `config.DeploymentRegion` — env `deployment_region`

### Осталось (требует infra-доступа)
- NATS cross-region mirror (Terraform/Helm)
- PostgreSQL WAL streaming
- S3 CRR configuration
- DR drills
- Compliance audit

---

## Verification
```bash
cd backend && go build ./...                  # ✓ OK
cd backend && go test ./internal/... -count=1  # ✓ PASS
```
