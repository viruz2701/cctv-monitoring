-- +migrate Up
-- Migration 030: Dashboard Multi-Device Sync (P1-1.4)
--
-- Сохраняет layout дашборда для синхронизации между устройствами
-- пользователя. Использует last-write-wins conflict resolution.
--
-- Compliance:
--   - IEC 62443-3-3 SR 2.1: User account management — привязка к user_id
--   - ISO 27001 A.12.4: Audit trail — updated_at для отслеживания изменений
--   - ISO 27019 PCC.A.12: ICS audit trail
--   - OWASP ASVS V3.3: Session management — данные привязаны к пользователю
--   - Приказ ОАЦ № 66 п. 7.18.2: Идентификация пользователей

-- ═══════════════════════════════════════════════════════════════════
-- 1. Таблица workspace_layouts
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS workspace_layouts (
    user_id     TEXT NOT NULL,
    tab_id      TEXT NOT NULL DEFAULT 'overview',
    layout      JSONB NOT NULL DEFAULT '[]'::jsonb,
    visible_widgets TEXT[] NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, tab_id)
);

COMMENT ON TABLE workspace_layouts IS
    'P1-1.4: Dashboard Multi-Device Sync. Хранит layout дашборда '
    'с привязкой к user_id для синхронизации между устройствами. '
    'Conflict resolution: last-write-wins.';

COMMENT ON COLUMN workspace_layouts.layout IS
    'JSONB массив виджетов с их позициями (x, y, w, h)';
COMMENT ON COLUMN workspace_layouts.visible_widgets IS
    'Массив ID видимых виджетов на дашборде';

-- ═══════════════════════════════════════════════════════════════════
-- 2. Индексы для быстрого поиска
-- ═══════════════════════════════════════════════════════════════════

CREATE INDEX IF NOT EXISTS idx_workspace_layouts_user
    ON workspace_layouts(user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_layouts_updated
    ON workspace_layouts(updated_at DESC);
