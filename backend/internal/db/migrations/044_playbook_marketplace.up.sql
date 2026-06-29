-- ═══════════════════════════════════════════════════════════════════════
-- P1-MARKET: Playbook Marketplace
--
-- Таблицы для публичного marketplace pre-built playbooks:
--   - playbook_marketplace — каталог плейбуков
--   - playbook_ratings     — рейтинги и отзывы (1-5 звёзд)
--   - playbook_installs    — история установок в tenant
--   - playbook_shares      — приватный обмен между tenant'ами
--
-- Compliance:
--   - ISO 27001 A.12.4 (Audit trail — created_at, updated_at)
--   - OWASP ASVS V6 (Cryptographic storage — UUID PK)
-- ═══════════════════════════════════════════════════════════════════════

-- ── Каталог плейбуков marketplace ─────────────────────────────────────
CREATE TABLE playbook_marketplace (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
    description    TEXT CHECK (char_length(description) <= 2000),
    vendor         TEXT NOT NULL CHECK (vendor IN ('hikvision', 'dahua', 'axis', 'uniview', 'generic')),
    version        TEXT NOT NULL CHECK (char_length(version) BETWEEN 1 AND 20),
    compat_matrix  TEXT[] NOT NULL DEFAULT '{}',          -- supported device models
    playbook_data  JSONB NOT NULL,                         -- полный playbook YAML/JSON
    verified       BOOLEAN NOT NULL DEFAULT false,         -- vendor-verified badge
    install_count  INT NOT NULL DEFAULT 0,
    avg_rating     NUMERIC(3,2) NOT NULL DEFAULT 0,       -- средний рейтинг
    review_count   INT NOT NULL DEFAULT 0,
    tenant_id      TEXT NOT NULL DEFAULT 'system',         -- кто опубликовал
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индекс для поиска по вендору
CREATE INDEX idx_playbook_marketplace_vendor ON playbook_marketplace(vendor);

-- Индекс для full-text поиска
CREATE INDEX idx_playbook_marketplace_search ON playbook_marketplace
    USING GIN (to_tsvector('english', name || ' ' || COALESCE(description, '')));

-- Индекс для сортировки по рейтингу
CREATE INDEX idx_playbook_marketplace_rating ON playbook_marketplace(avg_rating DESC);

-- Триггер обновления updated_at
CREATE TRIGGER trg_playbook_marketplace_updated_at
    BEFORE UPDATE ON playbook_marketplace
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ── Рейтинги и отзывы ────────────────────────────────────────────────
CREATE TABLE playbook_ratings (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    playbook_id  UUID NOT NULL REFERENCES playbook_marketplace(id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL,
    score        INT NOT NULL CHECK (score >= 1 AND score <= 5),
    review       TEXT CHECK (char_length(review) <= 2000),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(playbook_id, user_id)
);

CREATE INDEX idx_playbook_ratings_playbook ON playbook_ratings(playbook_id);

CREATE TRIGGER trg_playbook_ratings_updated_at
    BEFORE UPDATE ON playbook_ratings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ── История установок ────────────────────────────────────────────────
CREATE TABLE playbook_installs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    playbook_id  UUID NOT NULL REFERENCES playbook_marketplace(id) ON DELETE CASCADE,
    tenant_id    TEXT NOT NULL,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_playbook_installs_playbook ON playbook_installs(playbook_id);
CREATE INDEX idx_playbook_installs_tenant ON playbook_installs(tenant_id);

-- ── Приватный обмен плейбуками между tenant'ами ─────────────────────
CREATE TABLE playbook_shares (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    playbook_id   UUID NOT NULL REFERENCES playbook_marketplace(id) ON DELETE CASCADE,
    source_tenant TEXT NOT NULL,
    target_tenant TEXT NOT NULL,
    shared_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(playbook_id, target_tenant)
);

CREATE INDEX idx_playbook_shares_target ON playbook_shares(target_tenant);

-- ═══════════════════════════════════════════════════════════════════════
-- Функция пересчёта среднего рейтинга
-- ═══════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION recalc_playbook_rating()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE playbook_marketplace
    SET
        avg_rating = COALESCE(
            (SELECT ROUND(AVG(score)::numeric, 2) FROM playbook_ratings WHERE playbook_id = COALESCE(NEW.playbook_id, OLD.playbook_id)),
            0
        ),
        review_count = (
            SELECT COUNT(*) FROM playbook_ratings WHERE playbook_id = COALESCE(NEW.playbook_id, OLD.playbook_id)
        )
    WHERE id = COALESCE(NEW.playbook_id, OLD.playbook_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_recalc_playbook_rating_insert
    AFTER INSERT ON playbook_ratings
    FOR EACH ROW EXECUTE FUNCTION recalc_playbook_rating();

CREATE TRIGGER trg_recalc_playbook_rating_update
    AFTER UPDATE ON playbook_ratings
    FOR EACH ROW EXECUTE FUNCTION recalc_playbook_rating();

CREATE TRIGGER trg_recalc_playbook_rating_delete
    AFTER DELETE ON playbook_ratings
    FOR EACH ROW EXECUTE FUNCTION recalc_playbook_rating();

-- ═══════════════════════════════════════════════════════════════════════
-- Функция увеличения счётчика установок
-- ═══════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION increment_playbook_install_count()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE playbook_marketplace
    SET install_count = install_count + 1
    WHERE id = NEW.playbook_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_increment_install_count
    AFTER INSERT ON playbook_installs
    FOR EACH ROW EXECUTE FUNCTION increment_playbook_install_count();
