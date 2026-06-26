-- +migrate Down
-- Откат P2-3.2: OAuth2 for External Adapters
DROP TRIGGER IF EXISTS trg_oauth2_tokens_updated_at ON oauth2_tokens;
DROP FUNCTION IF EXISTS update_oauth2_tokens_updated_at();
DROP TABLE IF EXISTS oauth2_tokens;
