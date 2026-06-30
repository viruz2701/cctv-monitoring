-- +migrate Up
-- P2-API: API Versioning Strategy
--
-- Таблица метаданных версий API для URL-based (/api/v1/) и
-- header-based (X-API-Version) версионирования.
--
-- Поддерживает:
--   - Регистрация новых версий API
--   - Deprecation с sunset date
--   - Changelog для каждой версии
--   - Audit trail для изменений (ISO 27001 A.12.4)
--
-- Compliance:
--   - IEC 62443-3-3 SL-2 (Zone 2 — DMZ): Управление изменениями API
--   - ISO 27001 A.12.4.1: Event logging — audit trail
--   - OWASP ASVS V2.1.1: Версионирование API
--   - Приказ ОАЦ №66 п. 7.18.3: Аудит операций

-- +migrate Up

-- ═══════════════════════════════════════════════════════════════════════
-- 1. API Versions Registry
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE api_versions (
    version         TEXT PRIMARY KEY,              -- 'v1', 'v2'
    released_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deprecated_at   TIMESTAMPTZ,                   -- когда объявлена deprecated
    sunset_at       TIMESTAMPTZ,                   -- когда версия будет выключена
    changelog       TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_api_version_format CHECK (
        version ~ '^v[0-9]+$'
    ),
    CONSTRAINT chk_api_sunset_after_deprecated CHECK (
        sunset_at IS NULL
        OR (deprecated_at IS NOT NULL AND sunset_at >= deprecated_at)
    )
);

COMMENT ON TABLE api_versions IS
    'P2-API: Реестр версий API. Содержит метаданные о версиях, '
    'даты deprecation/sunset, changelog. '
    'Соответствует IEC 62443 SL-2, OWASP ASVS V2.1.1';

-- Индекс для поиска активных версий
CREATE INDEX idx_api_versions_active
    ON api_versions(version)
    WHERE deprecated_at IS NULL;

-- Индекс для поиска sunset-версий по дате
CREATE INDEX idx_api_versions_sunset
    ON api_versions(sunset_at)
    WHERE sunset_at IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════
-- 2. Changelog Entries
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE api_changelog (
    id              VARCHAR(64) PRIMARY KEY DEFAULT gen_random_uuid()::text,
    version         TEXT NOT NULL REFERENCES api_versions(version) ON DELETE CASCADE,
    change_date     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    change_text     TEXT NOT NULL,
    is_breaking     BOOLEAN NOT NULL DEFAULT FALSE,
    jira_ref        TEXT NOT NULL DEFAULT '',
    created_by      TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_changelog_version
    ON api_changelog(version, change_date DESC);

COMMENT ON TABLE api_changelog IS
    'P2-API: Changelog записей для каждой версии API. '
    'Содержит описание изменений, breaking changes, ссылки на JIRA.';

-- ═══════════════════════════════════════════════════════════════════════
-- 3. Seed data — v1 по умолчанию
-- ═══════════════════════════════════════════════════════════════════════

INSERT INTO api_versions (version, released_at, changelog)
VALUES ('v1', '2026-01-15T00:00:00Z', 'Initial release v1')
ON CONFLICT (version) DO NOTHING;

INSERT INTO api_changelog (version, change_date, change_text, is_breaking, created_by)
VALUES ('v1', '2026-01-15T00:00:00Z', 'Initial release', FALSE, 'system')
ON CONFLICT DO NOTHING;
