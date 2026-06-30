-- +migrate Up
-- EDGE-08: WireGuard On-Demand VPN Sessions
--
-- Таблица для управления временными VPN-сессиями для удалённого доступа
-- инженеров к edge-агентам через WireGuard туннель.
--
-- Compliance:
--   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 — Backend)
--   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
--   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
--   - ISO 27001 A.12.4: Audit trail (все мутации логируются)
--   - OWASP ASVS V2.1: Session management
--   - OWASP ASVS V3.3: Privilege escalation prevention

-- ═══════════════════════════════════════════════════════════════════════════
-- EDGE-08: vpn_sessions — WireGuard VPN сессии для удалённого доступа
-- ═══════════════════════════════════════════════════════════════════════════

CREATE TABLE vpn_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        VARCHAR(100) NOT NULL REFERENCES agents(id),
    engineer_id     UUID NOT NULL REFERENCES users(id),
    started_at      TIMESTAMPTZ DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    allowed_ips     INET[] NOT NULL,
    public_key      TEXT NOT NULL,
    status          VARCHAR(20) DEFAULT 'active',
    bytes_transferred BIGINT DEFAULT 0,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    closed_at       TIMESTAMPTZ,

    -- Статусы: active, expired, revoked
    CONSTRAINT ck_vpn_sessions_status CHECK (status IN ('active', 'expired', 'revoked')),
    -- expires_at должен быть в будущем при создании
    CONSTRAINT ck_vpn_sessions_expires_at CHECK (expires_at > started_at)
);

-- Индексы для быстрого поиска
CREATE INDEX idx_vpn_sessions_agent_id
    ON vpn_sessions(agent_id);

CREATE INDEX idx_vpn_sessions_engineer_id
    ON vpn_sessions(engineer_id);

CREATE INDEX idx_vpn_sessions_status
    ON vpn_sessions(status);

-- Индекс для поиска активных сессий (для auto-cleanup)
CREATE INDEX idx_vpn_sessions_active_expires
    ON vpn_sessions(expires_at)
    WHERE status = 'active';

-- Индекс для поиска по инженеру и статусу
CREATE INDEX idx_vpn_sessions_engineer_status
    ON vpn_sessions(engineer_id, status);

-- Комментарии к таблице и колонкам
COMMENT ON TABLE vpn_sessions IS 'WireGuard VPN сессии для удалённого доступа инженеров к edge-агентам. EDGE-08.';
COMMENT ON COLUMN vpn_sessions.id IS 'Уникальный идентификатор сессии';
COMMENT ON COLUMN vpn_sessions.agent_id IS 'ID edge-агента, к которому предоставлен доступ';
COMMENT ON COLUMN vpn_sessions.engineer_id IS 'ID инженера, которому предоставлен доступ';
COMMENT ON COLUMN vpn_sessions.started_at IS 'Время начала сессии';
COMMENT ON COLUMN vpn_sessions.expires_at IS 'Время истечения сессии (1-2 часа от started_at)';
COMMENT ON COLUMN vpn_sessions.allowed_ips IS 'Список разрешённых IP-адресов (LAN агента)';
COMMENT ON COLUMN vpn_sessions.public_key IS 'Публичный ключ WireGuard для пира';
COMMENT ON COLUMN vpn_sessions.status IS 'Статус сессии: active, expired, revoked';
COMMENT ON COLUMN vpn_sessions.bytes_transferred IS 'Количество переданных байт (обновляется из WG)';
COMMENT ON COLUMN vpn_sessions.closed_at IS 'Время закрытия сессии (revoke/expire)';
