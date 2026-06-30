-- P2-CHECK: Conditional Checklists (MaintainX-level)
--
-- Система шаблонов чек-листов с поддержкой:
--   - depends_on/operator/value (conditional logic)
--   - Sub-items (children)
--   - Scoring с passing threshold
--   - Mandatory/optional items
--   - Templates per device type (camera, nvr, dvr, etc.)
--
-- Compliance:
--   - IEC 62443 SL-3 (Zone 3 — Application integrity)
--   - ISO 27001 A.12.4.1 (Event logging — checklist audit trail)
--   - ISO 27001 A.12.6 (Maintenance — structured checklists)
--   - OWASP ASVS V5.1 (Input validation — enum constraints)
--   - Приказ ОАЦ №66 п. 7.18.3 (Аудит операций)

-- ═══════════════════════════════════════════════════════════════════════
-- 1. Checklist Templates
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS checklist_templates (
    id              VARCHAR(64) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    device_types    TEXT[] NOT NULL DEFAULT '{}',       -- camera, nvr, dvr, etc
    pass_threshold  INT NOT NULL DEFAULT 70,            -- % threshold to pass (0-100)
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_checklist_template_pass_threshold CHECK (pass_threshold >= 0 AND pass_threshold <= 100)
);

CREATE INDEX idx_checklist_templates_device_types ON checklist_templates USING GIN(device_types);
CREATE INDEX idx_checklist_templates_active ON checklist_templates(is_active);

-- ═══════════════════════════════════════════════════════════════════════
-- 2. Checklist Items (hierarchical, with condition support)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS checklist_items (
    id              VARCHAR(64) PRIMARY KEY,
    template_id     VARCHAR(64) NOT NULL REFERENCES checklist_templates(id) ON DELETE CASCADE,
    parent_id       VARCHAR(64) REFERENCES checklist_items(id) ON DELETE CASCADE,  -- NULL = root item
    label           TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    item_type       VARCHAR(32) NOT NULL DEFAULT 'boolean',
        -- boolean, text, photo, numeric, signature, select, multi_select
    mandatory       BOOLEAN NOT NULL DEFAULT TRUE,
    score           INT NOT NULL DEFAULT 0,             -- points for this item (0 = no score)
    sort_order      INT NOT NULL DEFAULT 0,
    options         JSONB DEFAULT NULL,                 -- for select/multi_select: ["opt1","opt2"]
    validation_min  FLOAT DEFAULT NULL,                 -- min value for numeric type
    validation_max  FLOAT DEFAULT NULL,                 -- max value for numeric type
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_checklist_item_type CHECK (
        item_type IN ('boolean', 'text', 'photo', 'numeric', 'signature', 'select', 'multi_select')
    ),
    CONSTRAINT chk_checklist_item_score CHECK (score >= 0)
);

CREATE INDEX idx_checklist_items_template_id ON checklist_items(template_id);
CREATE INDEX idx_checklist_items_parent_id ON checklist_items(parent_id);

-- ═══════════════════════════════════════════════════════════════════════
-- 3. Checklist Conditions (depends_on logic)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS checklist_conditions (
    id              VARCHAR(64) PRIMARY KEY,
    item_id         VARCHAR(64) NOT NULL REFERENCES checklist_items(id) ON DELETE CASCADE,
    field_id        VARCHAR(64) NOT NULL,               -- ссылается на checklist_items.id (поле-триггер)
    operator        VARCHAR(16) NOT NULL,                -- eq, neq, gt, lt, gte, lte, in
    value           TEXT NOT NULL,                       -- serialized as text (JSON for 'in' operator)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_checklist_condition_operator CHECK (
        operator IN ('eq', 'neq', 'gt', 'lt', 'gte', 'lte', 'in')
    )
);

CREATE INDEX idx_checklist_conditions_item_id ON checklist_conditions(item_id);
CREATE INDEX idx_checklist_conditions_field_id ON checklist_conditions(field_id);

-- ═══════════════════════════════════════════════════════════════════════
-- 4. Work Order Checklists (started/submitted instances)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS work_order_checklists (
    id              VARCHAR(64) PRIMARY KEY,
    work_order_id   VARCHAR(64) NOT NULL,
    template_id     VARCHAR(64) NOT NULL REFERENCES checklist_templates(id) ON DELETE RESTRICT,
    status          VARCHAR(32) NOT NULL DEFAULT 'in_progress',
        -- in_progress, submitted, verified
    total_score     INT NOT NULL DEFAULT 0,
    max_score       INT NOT NULL DEFAULT 0,
    score_percent   FLOAT NOT NULL DEFAULT 0,
    passed          BOOLEAN NOT NULL DEFAULT FALSE,
    started_by      VARCHAR(64) NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    submitted_by    VARCHAR(64) DEFAULT NULL,
    submitted_at    TIMESTAMPTZ DEFAULT NULL,
    verified_by     VARCHAR(64) DEFAULT NULL,
    verified_at     TIMESTAMPTZ DEFAULT NULL,
    notes           TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_wo_checklist_status CHECK (
        status IN ('in_progress', 'submitted', 'verified')
    ),
    CONSTRAINT chk_wo_checklist_score_percent CHECK (
        score_percent >= 0 AND score_percent <= 100
    )
);

CREATE INDEX idx_wo_checklists_work_order_id ON work_order_checklists(work_order_id);
CREATE INDEX idx_wo_checklists_template_id ON work_order_checklists(template_id);
CREATE INDEX idx_wo_checklists_status ON work_order_checklists(status);

-- ═══════════════════════════════════════════════════════════════════════
-- 5. Work Order Checklist Responses
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS work_order_checklist_responses (
    id              VARCHAR(64) PRIMARY KEY,
    checklist_id    VARCHAR(64) NOT NULL REFERENCES work_order_checklists(id) ON DELETE CASCADE,
    item_id         VARCHAR(64) NOT NULL REFERENCES checklist_items(id) ON DELETE RESTRICT,
    value           TEXT NOT NULL DEFAULT '',            -- 'true', 'false', text, photo_url, numeric, signature_data
    photo_url       VARCHAR(1024) DEFAULT NULL,          -- for photo type items
    skipped         BOOLEAN NOT NULL DEFAULT FALSE,      -- true if hidden by condition
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Уникальность: один ответ на item в рамках одного запуска чек-листа
    CONSTRAINT uq_wo_checklist_response UNIQUE (checklist_id, item_id)
);

CREATE INDEX idx_wo_checklist_responses_checklist_id ON work_order_checklist_responses(checklist_id);
CREATE INDEX idx_wo_checklist_responses_item_id ON work_order_checklist_responses(item_id);

-- ═══════════════════════════════════════════════════════════════════════
-- 6. Checklist Scores (audit trail for scoring history)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS checklist_scores (
    id              VARCHAR(64) PRIMARY KEY,
    checklist_id    VARCHAR(64) NOT NULL REFERENCES work_order_checklists(id) ON DELETE CASCADE,
    item_id         VARCHAR(64) NOT NULL REFERENCES checklist_items(id) ON DELETE RESTRICT,
    score           INT NOT NULL DEFAULT 0,
    max_score       INT NOT NULL DEFAULT 0,
    scored_by       VARCHAR(64) NOT NULL,
    scored_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes           TEXT NOT NULL DEFAULT '',

    CONSTRAINT chk_checklist_score_non_negative CHECK (score >= 0 AND max_score >= 0)
);

CREATE INDEX idx_checklist_scores_checklist_id ON checklist_scores(checklist_id);
CREATE INDEX idx_checklist_scores_item_id ON checklist_scores(item_id);
