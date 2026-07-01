-- Migration 059: Refresh Token Rotation (P1-HI-05)
--
-- Добавляет поддержку rotation refresh token с device fingerprinting,
-- token family tracking и reuse detection.
--
-- Соответствие:
--   - OWASP ASVS V3.2.2: Refresh token rotation
--   - OWASP ASVS V3.2.3: Reuse detection
--   - ISO 27001 A.9.2.1: Device/user binding
--   - Приказ ОАЦ №66 п. 7.18.1: Уникальная идентификация узлов
-- +migrate Up

-- 1. Добавляем fingerprint_hash — хеш User-Agent + IP для привязки к устройству
ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS fingerprint_hash TEXT DEFAULT '';

-- 2. Добавляем token_family — UUID семьи токенов для отслеживания всех
--    токенов, полученных от одного initial token (reuse detection)
ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS token_family UUID DEFAULT NULL;

-- 3. Добавляем is_revoked — флаг ручной/автоматической отзывы токена
--    (семейная инвалидация при reuse detection)
ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS is_revoked BOOLEAN DEFAULT FALSE;

-- 4. Индекс для быстрого поиска по семье токенов (reuse detection)
CREATE INDEX IF NOT EXISTS idx_user_sessions_token_family
    ON user_sessions(token_family);

-- 5. Индекс для поиска не отозванных токенов
CREATE INDEX IF NOT EXISTS idx_user_sessions_active
    ON user_sessions(user_id, is_revoked, expires_at)
    WHERE is_revoked = FALSE AND expires_at > NOW();

COMMENT ON COLUMN user_sessions.fingerprint_hash IS
    'P1-HI-05: SHA-256 хеш User-Agent + IP для device fingerprinting (Приказ ОАЦ №66 п.7.18.1)';

COMMENT ON COLUMN user_sessions.token_family IS
    'P1-HI-05: UUID семьи токенов. Все токены одной семьи = один initial token. '
    'При reuse старого токена — вся семья инвалидируется.';

COMMENT ON COLUMN user_sessions.is_revoked IS
    'P1-HI-05: Флаг отзыва. TRUE при reuse detection или ручной инвалидации.';
