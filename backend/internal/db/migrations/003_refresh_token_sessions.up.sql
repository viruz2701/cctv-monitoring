-- Migration 003: Add unique index on user_sessions.token_hash for refresh token support
-- +migrate Up

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_sessions_token_hash ON user_sessions(token_hash);
