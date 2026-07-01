-- Migration 059: Rollback Refresh Token Rotation
-- +migrate Down

DROP INDEX IF EXISTS idx_user_sessions_active;
DROP INDEX IF EXISTS idx_user_sessions_token_family;

ALTER TABLE user_sessions DROP COLUMN IF EXISTS fingerprint_hash;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS token_family;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS is_revoked;
