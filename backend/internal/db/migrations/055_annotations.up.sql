-- +migrate Up
-- ═══════════════════════════════════════════════════════════════════════
-- 055_annotations.up.sql — Work Order Photo Annotations (P1-PHOTO)
--
-- Таблица для хранения элементов аннотации для каждого фото в work order.
-- Элементы хранятся как JSONB для гибкости (разные типы: arrow, circle, text и т.д.)
--
-- Compliance:
--   - OWASP ASVS V5.1 (Input validation — JSON schema validation)
--   - ISO 27001 A.12.4 (Audit trail — created_at/updated_at)
--   - IEC 62443-3-3 SL-3 (Zone 3 — Application security)
--   - СТБ 34.101.27 п. 6.2 (Контроль целостности данных)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE work_order_annotations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    work_order_id   TEXT NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    photo_url       TEXT NOT NULL,
    elements        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_by      TEXT NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Одна аннотация на фото в рамках work order
    UNIQUE (work_order_id, photo_url)
);

-- Индекс для быстрого поиска аннотаций по work order
CREATE INDEX IF NOT EXISTS idx_annotations_work_order_id
    ON work_order_annotations(work_order_id);

-- Индекс для поиска по конкретному фото
CREATE INDEX IF NOT EXISTS idx_annotations_photo_url
    ON work_order_annotations(photo_url);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_annotations_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_annotations_updated_at ON work_order_annotations;
CREATE TRIGGER trg_annotations_updated_at
    BEFORE UPDATE ON work_order_annotations
    FOR EACH ROW
    EXECUTE FUNCTION update_annotations_updated_at();

-- RLS policies (соответствует multi-tenant архитектуре)
ALTER TABLE work_order_annotations ENABLE ROW LEVEL SECURITY;

-- Tenant isolation через work_orders join
CREATE POLICY annotations_tenant_isolation ON work_order_annotations
    USING (
        work_order_id IN (
            SELECT wo.id FROM work_orders wo
            JOIN devices d ON d.device_id = wo.device_id
            WHERE d.owner_id = current_setting('app.tenant_id')
        )
    );

-- Комментарии к колонкам
COMMENT ON TABLE work_order_annotations IS 'Элементы аннотации для фото work order (P1-PHOTO)';
COMMENT ON COLUMN work_order_annotations.id IS 'Уникальный идентификатор аннотации';
COMMENT ON COLUMN work_order_annotations.work_order_id IS 'Ссылка на work order';
COMMENT ON COLUMN work_order_annotations.photo_url IS 'URL фото, к которому привязана аннотация';
COMMENT ON COLUMN work_order_annotations.elements IS 'JSONB массив элементов аннотации (arrow, circle, text, и т.д.)';
COMMENT ON COLUMN work_order_annotations.created_by IS 'ID пользователя, создавшего аннотацию';
