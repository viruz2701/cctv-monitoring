-- ═══════════════════════════════════════════════════════════════════════
-- P1-MARKET: Playbook Marketplace — Down Migration
-- ═══════════════════════════════════════════════════════════════════════

-- +migrate Down

DROP TRIGGER IF EXISTS trg_increment_install_count ON playbook_installs;
DROP FUNCTION IF EXISTS increment_playbook_install_count();

DROP TRIGGER IF EXISTS trg_recalc_playbook_rating_delete ON playbook_ratings;
DROP TRIGGER IF EXISTS trg_recalc_playbook_rating_update ON playbook_ratings;
DROP TRIGGER IF EXISTS trg_recalc_playbook_rating_insert ON playbook_ratings;
DROP FUNCTION IF EXISTS recalc_playbook_rating();

DROP TABLE IF EXISTS playbook_shares;
DROP TABLE IF EXISTS playbook_installs;
DROP TABLE IF EXISTS playbook_ratings;
DROP TABLE IF EXISTS playbook_marketplace;
