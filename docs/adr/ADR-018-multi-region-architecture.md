# ADR-018: Multi-Region Geo-Redundancy Architecture

**Статус:** DRAFT  
**Дата:** 2026-06-26  
**Автор:** System Architect  

## Контекст

CCTV Health Monitor — КИИ РБ, класс KII-2. Enterprise-тенденты требуют:
- Disaster Recovery (DR) с RTO < 15 мин, RPO < 5 мин
- Data residency — данные не покидают регион (GDPR, 152-ФЗ, PDPL)
- Multi-region для глобальных клиентов (EU, CIS, MENA, SEA)

## Решение

### Модель: Active-Passive per tenant

Каждый тенант привязан к primary-региону. Вторичный регион — холодный standby с async WAL репликацией.

```
Tenant → primary_region → [EU-Central | CIS-East | MENA-Gulf | SEA-Hub]
         ↓
    TenantRegions table (tenant_id, primary_region, failover_region, status)
```

### Компоненты

#### 1. NATS JetStream — Local Raft + Async Mirror
- **In-region:** 3-node Raft cluster (consensus, exactly-once)
- **Cross-region:** Async mirror streams (KV + Object Store)
- **Failover:** Manual promotion of mirror → active
- **Latency:** Не влияет (async, batch ack)

#### 2. PostgreSQL / TimescaleDB — In-region Replicas + Async WAL DR
- **In-region:** 1 primary + 2 read replicas (hot standby)
- **Cross-region:** Async WAL streaming to DR replica
- **Failover:** `pg_promote()` + connection string switch
- **Residency:** Данные тенанта не покидают регион

#### 3. S3-Compatible Storage — CRR (Cross-Region Replication)
- **Primary:** MinIO in-region для cold storage (>30d)
- **DR:** S3 CRR to DR region bucket
- **Retention:** 7 лет (КИИ), async batch replication

### Tenant Region Mapping

```sql
CREATE TABLE tenant_regions (
    tenant_id       TEXT PRIMARY KEY,
    primary_region  TEXT NOT NULL,
    failover_region TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active', -- active, failover, migrating
    pinned_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    failover_at     TIMESTAMPTZ,
    CONSTRAINT valid_region CHECK (primary_region IN ('eu-central', 'cis-east', 'mena-gulf', 'sea-hub'))
);
```

### Health Check — Region Awareness

`/health/ready` возвращает информацию о регионе:
```json
{
  "region": "eu-central",
  "status": "ok",
  "replication_lag_ms": 150,
  "is_dr_ready": true
}
```

### Failover Process (Semi-auto)

```
1. Admin detects region outage (PagerDuty)
2. Admin confirms failover via API: POST /api/v1/admin/failover/{tenant_id}
3. System:
   a. Promotes DR NATS mirror → active
   b. Promotes DR PostgreSQL → primary
   c. Switches tenant → failover_region
   d. Updates tenant_regions.status = 'failover'
   e. Sends notification
4. Rollback: reverse process when primary recovers
```

## Последствия

### Positive
- Enterprise-grade DR (RTO < 15min, RPO < 5min)
- Data residency compliance
- Budget: ~$120-150K/year

### Negative
- Semi-auto failover (не fully automated)
- Cold DR (не active-active)
- Дополнительная сложность операций

### Constraints
- Cloud vendor: минимум 2 availability zones per region
- Cross-region latency < 150ms
- Tenant pinning — тенант не может мигрировать между регионами без downtme

## Связанные ADR
- ADR-005: State Management
- ADR-012: Security Architecture (регионы = security zones)

## Timeline
- Q3 2026: Foundation (Discovery + Topology + Sandbox)
- Q4 2026: Validation (Chaos + IaC)
- Q1 2027: Automation (Monitoring + Tooling)
- Q2 2027: Rollout (Audit + Migration)
