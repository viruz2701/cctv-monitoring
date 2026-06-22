-- Migration 003: Remove unique index on user_sessions.token_hash
-- +migrate Down

DROP INDEX IF EXISTS idx_user_sessions_token_hash;
