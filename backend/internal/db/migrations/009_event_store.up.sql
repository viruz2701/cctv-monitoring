-- +migrate Up
-- Migration 009: Event Store metadata table
--
-- Хранит метаданные событий для быстрого поиска без сканирования S3.
-- Сами события хранятся в NATS JetStream (hot) и S3/MinIO (cold).
--
-- Compliance:
--   ISO 27001 A.12.4.1 (Event logging)
--   СТБ 34.101.27 п. 7.5 (Audit trail integrity)
--   IEC 62443 SR 2.8 (Audit events)
--   Приказ ОАЦ №66 п. 7.18.3 (Audit trail for edge devices)

CREATE TABLE event_store_metadata (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        VARCHAR(64) NOT NULL UNIQUE,       -- UUID v7 из EventRecord.ID
    source          VARCHAR(32) NOT NULL,               -- alarms, cmms, predictions, telemetry, audit, system
    event_type      VARCHAR(128) NOT NULL,              -- alarm.created, cmms.wo.completed, etc.
    schema_version  VARCHAR(16) NOT NULL DEFAULT '1.0.0',
    aggregate_id    VARCHAR(64) NOT NULL DEFAULT '',    -- device_id, work_order_id, etc.
    actor_id        VARCHAR(64) DEFAULT '',             -- user_id или "system"
    trace_id        VARCHAR(64) DEFAULT '',             -- W3C Trace Context
    prev_hash       VARCHAR(128) DEFAULT '',            -- СТБ bash-256 HMAC цепочка
    event_timestamp TIMESTAMPTZ NOT NULL,               -- время события
    signed_at       TIMESTAMPTZ,                        -- время подписи (audit trail)
    storage_tier    VARCHAR(16) NOT NULL DEFAULT 'hot', -- 'hot' (NATS) | 'cold' (S3) | 'archived'
    s3_key          VARCHAR(512) DEFAULT '',            -- S3 key если в cold storage
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_event_store_source ON event_store_metadata (source, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_store_aggregate ON event_store_metadata (aggregate_id, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_store_type ON event_store_metadata (event_type, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_store_timestamp ON event_store_metadata (event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_event_store_tier ON event_store_metadata (storage_tier);

-- Комментарии к таблице и колонкам
COMMENT ON TABLE event_store_metadata IS 'Метаданные событий Event Store. Сами события — в NATS JetStream (hot) и S3/MinIO (cold)';
COMMENT ON COLUMN event_store_metadata.event_id IS 'UUID v7 (time-sortable) из EventRecord';
COMMENT ON COLUMN event_store_metadata.source IS 'Источник: alarms, cmms, predictions, telemetry, audit, system';
COMMENT ON COLUMN event_store_metadata.event_type IS 'Тип события: alarm.created, cmms.wo.completed и т.д.';
COMMENT ON COLUMN event_store_metadata.schema_version IS 'Версия схемы события (semver)';
COMMENT ON COLUMN event_store_metadata.aggregate_id IS 'ID агрегата: device_id, work_order_id';
COMMENT ON COLUMN event_store_metadata.actor_id IS 'Кто вызвал событие (user_id или system)';
COMMENT ON COLUMN event_store_metadata.trace_id IS 'W3C Trace Context для распределённой трассировки';
COMMENT ON COLUMN event_store_metadata.prev_hash IS 'СТБ bash-256 HMAC предыдущего события (tamper detection)';
COMMENT ON COLUMN event_store_metadata.storage_tier IS 'Уровень хранения: hot (NATS), cold (S3), archived';
COMMENT ON COLUMN event_store_metadata.s3_key IS 'S3 object key если событие в cold storage';
