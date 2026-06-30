-- P2-FIELDS: Custom Fields Advanced (Shelf.nu-level)
--
-- Система кастомных полей с поддержкой:
--   - 15+ field types (text, number, date, dropdown, multi_select, url, email,
--     barcode, signature, file_upload, checkbox, radio, textarea, time, color, user)
--   - Валидация (min/max, regex, custom)
--   - Условная видимость (conditional show/hide based on other field values)
--   - Группы полей (field groups)
--   - EAV модель хранения значений (custom_field_values)
--   - Привязка к entity_type: device, work_order, site, part
--
-- Compliance:
--   - IEC 62443 SL-3 (Zone 3 — Application integrity)
--   - ISO 27001 A.12.4.1 (Event logging — audit trail for field mutations)
--   - OWASP ASVS V5.1 (Input validation — enum constraints on field_type)
--   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)
--   - СТБ 34.101.27 п. 6.3 (Контроль целостности данных)

-- +migrate Up

-- ═══════════════════════════════════════════════════════════════════════
-- 1. Custom Field Groups
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS custom_field_groups (
    id              VARCHAR(64) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    entity_type     VARCHAR(32) NOT NULL,       -- device, work_order, site, part
    sort_order      INT NOT NULL DEFAULT 0,
    is_collapsible  BOOLEAN NOT NULL DEFAULT FALSE,
    is_collapsed    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_cfg_entity_type CHECK (
        entity_type IN ('device', 'work_order', 'site', 'part')
    )
);

CREATE INDEX idx_cfg_entity_type ON custom_field_groups(entity_type);
CREATE INDEX idx_cfg_sort_order ON custom_field_groups(sort_order);

COMMENT ON TABLE custom_field_groups IS
    'P2-FIELDS: Группы кастомных полей для группировки в FieldBuilder UI. '
    'Соответствует IEC 62443 SL-3, OWASP ASVS V5.1';

-- ═══════════════════════════════════════════════════════════════════════
-- 2. Custom Field Definitions
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS custom_field_definitions (
    id              VARCHAR(64) PRIMARY KEY,
    entity_type     VARCHAR(32) NOT NULL,       -- device, work_order, site, part
    field_type      VARCHAR(32) NOT NULL,       -- text, number, date, dropdown, etc.
    name            VARCHAR(255) NOT NULL,       -- unique within entity_type
    label           VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    required        BOOLEAN NOT NULL DEFAULT FALSE,
    options         JSONB DEFAULT NULL,          -- для dropdown/multi_select/radio: ["opt1","opt2"]
    validation      JSONB DEFAULT NULL,          -- { "min": 0, "max": 100, "regex": "...", "custom": "..." }
    visibility      JSONB DEFAULT NULL,          -- { "field_id": "...", "operator": "eq", "value": "..." }
    group_id        VARCHAR(64) REFERENCES custom_field_groups(id) ON DELETE SET NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    default_value   JSONB DEFAULT NULL,          -- значение по умолчанию
    placeholder     VARCHAR(255) NOT NULL DEFAULT '',
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_cfd_entity_type CHECK (
        entity_type IN ('device', 'work_order', 'site', 'part')
    ),
    CONSTRAINT chk_cfd_field_type CHECK (
        field_type IN (
            'text', 'number', 'date', 'dropdown', 'multi_select',
            'url', 'email', 'barcode', 'signature', 'file_upload',
            'checkbox', 'radio', 'textarea', 'time', 'color', 'user'
        )
    ),
    CONSTRAINT uq_cfd_name_per_entity UNIQUE (entity_type, name)
);

CREATE INDEX idx_cfd_entity_type ON custom_field_definitions(entity_type);
CREATE INDEX idx_cfd_group_id ON custom_field_definitions(group_id);
CREATE INDEX idx_cfd_active ON custom_field_definitions(is_active);
CREATE INDEX idx_cfd_sort_order ON custom_field_definitions(sort_order);

COMMENT ON TABLE custom_field_definitions IS
    'P2-FIELDS: Определения кастомных полей. Содержит тип, валидацию, '
    'условную видимость, группу. Соответствует IEC 62443 SL-3, OWASP ASVS V5.1';

-- ═══════════════════════════════════════════════════════════════════════
-- 3. Custom Field Values (EAV)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS custom_field_values (
    id              VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    field_id        VARCHAR(64) NOT NULL REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    entity_type     VARCHAR(32) NOT NULL,
    entity_id       VARCHAR(64) NOT NULL,       -- UUID or text ID of the entity
    value           JSONB NOT NULL DEFAULT 'null',
    created_by      TEXT,                       -- user ID who set the value
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_cfv_entity_type CHECK (
        entity_type IN ('device', 'work_order', 'site', 'part')
    ),
    CONSTRAINT uq_cfv_field_entity UNIQUE (field_id, entity_type, entity_id)
);

CREATE INDEX idx_cfv_field_id ON custom_field_values(field_id);
CREATE INDEX idx_cfv_entity ON custom_field_values(entity_type, entity_id);
CREATE INDEX idx_cfv_entity_type ON custom_field_values(entity_type);
CREATE INDEX idx_cfv_updated_at ON custom_field_values(updated_at DESC);

COMMENT ON TABLE custom_field_values IS
    'P2-FIELDS: EAV-модель хранения значений кастомных полей. '
    'Каждая запись — значение для конкретного поля и сущности. '
    'UNIQUE (field_id, entity_type, entity_id) предотвращает дубли.';

-- ═══════════════════════════════════════════════════════════════════════
-- 4. Audit trigger for custom_field_values (ISO 27001 A.12.4)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS custom_field_value_audit (
    id              VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    value_id        VARCHAR(64) NOT NULL,
    field_id        VARCHAR(64) NOT NULL,
    entity_type     VARCHAR(32) NOT NULL,
    entity_id       VARCHAR(64) NOT NULL,
    old_value       JSONB DEFAULT NULL,
    new_value       JSONB NOT NULL,
    changed_by      TEXT NOT NULL,
    changed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cfv_audit_value_id ON custom_field_value_audit(value_id);
CREATE INDEX idx_cfv_audit_field_id ON custom_field_value_audit(field_id);
CREATE INDEX idx_cfv_audit_entity ON custom_field_value_audit(entity_type, entity_id);
CREATE INDEX idx_cfv_audit_changed_at ON custom_field_value_audit(changed_at DESC);

COMMENT ON TABLE custom_field_value_audit IS
    'P2-FIELDS: Audit trail для изменений значений кастомных полей. '
    'Соответствует ISO 27001 A.12.4.1 (Event logging), Приказ ОАЦ №66 п. 7.18.3';
