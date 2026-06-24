-- +migrate Up
-- Migration 012: Workforce Management
--
-- WM-8.1.1: teams
-- WM-8.1.2: matrix RBAC (application-level, not DB)
-- WM-8.2.1: shift_configurations
-- WM-8.2.2: user_shift_assignments
-- WM-8.4.1: skills + user_skills
-- WM-8.4.2: certifications + user_certifications

CREATE TABLE teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    description TEXT DEFAULT '',
    lead_id     VARCHAR(64),
    member_ids  JSONB DEFAULT '[]'::jsonb,
    site_ids    JSONB DEFAULT '[]'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE shift_configurations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    type            VARCHAR(16) NOT NULL DEFAULT 'day',
    start_hour      INTEGER NOT NULL,
    end_hour        INTEGER NOT NULL,
    work_days       INTEGER[] NOT NULL DEFAULT '{1,2,3,4,5}',
    timezone        VARCHAR(64) NOT NULL DEFAULT 'UTC',
    site_id         VARCHAR(64) NOT NULL DEFAULT '',
    max_team_size   INTEGER NOT NULL DEFAULT 10,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_shift_assignments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(64) NOT NULL,
    shift_id    UUID NOT NULL REFERENCES shift_configurations(id) ON DELETE CASCADE,
    team_id     UUID REFERENCES teams(id),
    valid_from  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    is_primary  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(32) NOT NULL DEFAULT 'cctv',
    description TEXT DEFAULT ''
);

CREATE TABLE user_skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     VARCHAR(64) NOT NULL,
    skill_id    UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    level       INTEGER NOT NULL DEFAULT 1 CHECK (level >= 1 AND level <= 5),
    verified_by VARCHAR(64),
    verified_at TIMESTAMPTZ,
    UNIQUE (user_id, skill_id)
);

CREATE TABLE certifications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(200) NOT NULL,
    issuer              VARCHAR(100) NOT NULL DEFAULT '',
    category            VARCHAR(32) DEFAULT '',
    expires_after_days  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE user_certifications (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           VARCHAR(64) NOT NULL,
    certification_id  UUID NOT NULL REFERENCES certifications(id) ON DELETE CASCADE,
    obtained_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at        TIMESTAMPTZ,
    verified_by       VARCHAR(64),
    UNIQUE (user_id, certification_id)
);

COMMENT ON TABLE teams IS 'WM-8.1.1: Бригады техников';
COMMENT ON TABLE shift_configurations IS 'WM-8.2.1: Конфигурации смен';
COMMENT ON TABLE user_shift_assignments IS 'WM-8.2.2: Назначение техников на смены';
COMMENT ON TABLE skills IS 'WM-8.4.1: Навыки';
COMMENT ON TABLE user_skills IS 'WM-8.4.1: Навыки пользователей';
COMMENT ON TABLE certifications IS 'WM-8.4.2: Сертификации';
COMMENT ON TABLE user_certifications IS 'WM-8.4.2: Сертификации пользователей';
