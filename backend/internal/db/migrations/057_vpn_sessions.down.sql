-- +migrate Down
-- EDGE-08: WireGuard On-Demand VPN Sessions — откат

DROP INDEX IF EXISTS idx_vpn_sessions_engineer_status;
DROP INDEX IF EXISTS idx_vpn_sessions_active_expires;
DROP INDEX IF EXISTS idx_vpn_sessions_status;
DROP INDEX IF EXISTS idx_vpn_sessions_engineer_id;
DROP INDEX IF EXISTS idx_vpn_sessions_agent_id;

DROP TABLE IF EXISTS vpn_sessions;
