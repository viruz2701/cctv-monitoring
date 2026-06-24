-- +migrate Up
-- 013_work_requests.up.sql — WorkRequest entity (WO-4.1.1)
--
-- Публичная заявка на выполнение работ. Submit без авторизации, с reCAPTCHA.
-- После одобрения конвертируется в WorkOrder.
--
-- Compliance:
--   - ISO 27001 A.9.2.1 (User registration — external request)
--   - ISO 27001 A.14.2.1 (Service delivery — request portal)
--   - IEC 62443 SR 2.1 (Account management — request workflow)
--   - OWASP ASVS V1.1 (Input validation)
--   - СТБ 34.101.27 п. 6.2 (Разграничение доступа)

CREATE TABLE work_requests (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Основные поля
    title           VARCHAR(500) NOT NULL,
    description     TEXT,

    -- Связанные сущности (TEXT PK — совместимость с 001_initial_schema)
    device_id       TEXT REFERENCES devices(device_id) ON DELETE SET NULL,
    site_id         TEXT REFERENCES sites(id) ON DELETE SET NULL,

    -- Приоритет и тип
    priority        VARCHAR(20) NOT NULL DEFAULT 'medium'
                    CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    type            VARCHAR(20) NOT NULL DEFAULT 'corrective'
                    CHECK (type IN ('corrective', 'preventive', 'emergency', 'routine', 'inspection')),

    -- Контактные данные заявителя
    requester_name  VARCHAR(200) NOT NULL,
    requester_email VARCHAR(200) NOT NULL,
    requester_phone VARCHAR(50),

    -- Статус
    status          VARCHAR(20) NOT NULL DEFAULT 'submitted'
                    CHECK (status IN ('submitted', 'approved', 'converted', 'rejected', 'cancelled')),

    -- Approval
    approved_by     TEXT REFERENCES users(id) ON DELETE SET NULL,
    approved_at     TIMESTAMPTZ,
    rejected_by     TEXT REFERENCES users(id) ON DELETE SET NULL,
    rejected_at     TIMESTAMPTZ,
    rejection_reason TEXT,

    -- Конвертация в WorkOrder
    converted_work_order_id TEXT REFERENCES work_orders(id) ON DELETE SET NULL,
    converted_at            TIMESTAMPTZ,

    -- Метаданные
    source_ip       INET,
    user_agent      TEXT
);

-- Индексы для производительности
CREATE INDEX idx_work_requests_status ON work_requests(status);
CREATE INDEX idx_work_requests_created_at ON work_requests(created_at DESC);
CREATE INDEX idx_work_requests_device_id ON work_requests(device_id);
CREATE INDEX idx_work_requests_requester_email ON work_requests(requester_email);
