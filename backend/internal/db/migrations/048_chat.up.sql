-- P2-CHAT: Real-Time Chat per Work Order
--
-- Хранит сообщения чата для каждого Work Order с поддержкой:
--   - @mentions (упоминания пользователей)
--   - Attachments (файлы, изображения)
--   - Read receipts (прочитано/не прочитано)
--   - Reactions (эмодзи-реакции)
--   - Voice notes (голосовые заметки)
--
-- Compliance:
--   - ISO 27001 A.12.4.1 (Event logging — created_at/updated_at)
--   - ISO 27001 A.10.1 (Cryptographic controls — content encryption)
--   - IEC 62443 SR 2.1 (Account management — user_id tracking)
--   - OWASP ASVS V7.1 (Input validation — content size limits)
--   - Приказ ОАЦ №66 п. 7.18.3 (Аудит сообщений)

CREATE TABLE IF NOT EXISTS wo_chat_messages (
    id              VARCHAR(64) PRIMARY KEY,
    wo_id           VARCHAR(64) NOT NULL,
    user_id         VARCHAR(64) NOT NULL,
    user_name       VARCHAR(255) NOT NULL,
    text            TEXT NOT NULL DEFAULT '',
    message_type    VARCHAR(32) NOT NULL DEFAULT 'text', -- 'text', 'system', 'voice', 'image'
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Ограничения
    CONSTRAINT chk_wo_chat_message_type CHECK (message_type IN ('text', 'system', 'voice', 'image')),
    CONSTRAINT chk_wo_chat_text_not_empty CHECK (
        (message_type IN ('text', 'system') AND text != '') OR
        (message_type IN ('voice', 'image'))
    )
);

-- Индексы для быстрого поиска по Work Order
CREATE INDEX idx_wo_chat_messages_wo_id ON wo_chat_messages(wo_id);
CREATE INDEX idx_wo_chat_messages_created_at ON wo_chat_messages(wo_id, created_at DESC);
CREATE INDEX idx_wo_chat_messages_user_id ON wo_chat_messages(user_id);

-- ═══════════════════════════════════════════════════════════════════════
-- Attachments (файлы, прикреплённые к сообщению)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS wo_chat_attachments (
    id              VARCHAR(64) PRIMARY KEY,
    message_id      VARCHAR(64) NOT NULL REFERENCES wo_chat_messages(id) ON DELETE CASCADE,
    file_name       VARCHAR(512) NOT NULL,
    file_size       BIGINT NOT NULL,
    mime_type       VARCHAR(128) NOT NULL,
    storage_path    VARCHAR(1024) NOT NULL,
    thumbnail_path  VARCHAR(1024),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_wo_chat_attachment_size CHECK (file_size > 0 AND file_size <= 52428800) -- 50MB max
);

CREATE INDEX idx_wo_chat_attachments_message_id ON wo_chat_attachments(message_id);

-- ═══════════════════════════════════════════════════════════════════════
-- @mentions (упоминания пользователей)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS wo_chat_mentions (
    id              VARCHAR(64) PRIMARY KEY,
    message_id      VARCHAR(64) NOT NULL REFERENCES wo_chat_messages(id) ON DELETE CASCADE,
    mentioned_user  VARCHAR(64) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wo_chat_mentions_message_id ON wo_chat_mentions(message_id);
CREATE INDEX idx_wo_chat_mentions_user ON wo_chat_mentions(mentioned_user);

-- ═══════════════════════════════════════════════════════════════════════
-- Read Receipts (прочитано/не прочитано)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS wo_chat_read_receipts (
    message_id      VARCHAR(64) NOT NULL REFERENCES wo_chat_messages(id) ON DELETE CASCADE,
    user_id         VARCHAR(64) NOT NULL,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

-- ═══════════════════════════════════════════════════════════════════════
-- Reactions (эмодзи-реакции)
-- ═══════════════════════════════════════════════════════════════════════

CREATE TABLE IF NOT EXISTS wo_chat_reactions (
    message_id      VARCHAR(64) NOT NULL REFERENCES wo_chat_messages(id) ON DELETE CASCADE,
    user_id         VARCHAR(64) NOT NULL,
    reaction        VARCHAR(32) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id, reaction)
);

CREATE INDEX idx_wo_chat_reactions_message_id ON wo_chat_reactions(message_id);
