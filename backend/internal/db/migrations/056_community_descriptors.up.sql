-- +migrate Up
-- PROTO-07: Community Protocol Registry
--
-- Публичный реестр Protocol Descriptor'ов (как Docker Hub),
-- где community может публиковать и обмениваться дескрипторами
-- для различных вендоров CCTV.
--
-- Compliance:
--   - ISO 27001 A.12.4.1: Event logging
--   - IEC 62443-3-3 SR 1.1: Unique identification
--   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
--   - OWASP ASVS V5.1: Input validation

CREATE TABLE community_descriptors (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vendor      VARCHAR(200) NOT NULL,
    version     VARCHAR(50) NOT NULL,
    descriptor  JSONB NOT NULL,
    author_id   TEXT NOT NULL,
    rating      NUMERIC(3,2) NOT NULL DEFAULT 0,
    downloads   INTEGER NOT NULL DEFAULT 0,
    verified    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Один вендор = одна публикация (для простоты)
    CONSTRAINT uq_community_descriptor_vendor UNIQUE (vendor),
    -- Рейтинг от 0.00 до 5.00
    CONSTRAINT ck_community_descriptor_rating CHECK (rating >= 0 AND rating <= 5.00)
);

-- Индекс для быстрого поиска по вендору
CREATE UNIQUE INDEX idx_community_descriptors_vendor
    ON community_descriptors(vendor);

-- Индекс для поиска по рейтингу (сортировка)
CREATE INDEX idx_community_descriptors_rating
    ON community_descriptors(rating DESC);

-- Индекс для поиска по загрузкам (сортировка)
CREATE INDEX idx_community_descriptors_downloads
    ON community_descriptors(downloads DESC);

-- Индекс для verified фильтра
CREATE INDEX idx_community_descriptors_verified
    ON community_descriptors(verified)
    WHERE verified = TRUE;

-- Trigger для updated_at
CREATE OR REPLACE FUNCTION trg_community_descriptors_updated()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_community_descriptors_updated
    BEFORE UPDATE ON community_descriptors
    FOR EACH ROW
    EXECUTE FUNCTION trg_community_descriptors_updated();

-- ═══════════════════════════════════════════════════════════════════
-- Community Ratings Table
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE community_descriptor_ratings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    descriptor_id   UUID NOT NULL REFERENCES community_descriptors(id) ON DELETE CASCADE,
    user_id         TEXT NOT NULL,
    score           INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Один пользователь = одна оценка на дескриптор
    CONSTRAINT uq_descriptor_user_rating UNIQUE (descriptor_id, user_id),
    -- Оценка от 1 до 5
    CONSTRAINT ck_descriptor_rating_score CHECK (score >= 1 AND score <= 5)
);

CREATE INDEX idx_descriptor_ratings_descriptor
    ON community_descriptor_ratings(descriptor_id);

COMMENT ON TABLE community_descriptors IS
    'Публичный реестр Protocol Descriptor''ов (PROTO-07). '
    'Community может публиковать и оценивать дескрипторы для вендоров CCTV.';

COMMENT ON COLUMN community_descriptors.descriptor IS
    'JSON-дескриптор протокола (ProtocolDescriptor)';
COMMENT ON COLUMN community_descriptors.author_id IS
    'ID автора публикации (ссылка на users.id)';
COMMENT ON COLUMN community_descriptors.rating IS
    'Средний рейтинг (0.00-5.00), обновляется триггером при добавлении оценки';
COMMENT ON COLUMN community_descriptors.downloads IS
    'Счётчик скачиваний дескриптора';
COMMENT ON COLUMN community_descriptors.verified IS
    'Флаг верификации командой проекта';

COMMENT ON TABLE community_descriptor_ratings IS
    'Индивидуальные оценки пользователей для community дескрипторов';
