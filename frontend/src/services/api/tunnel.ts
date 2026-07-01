// ═══════════════════════════════════════════════════════════════════════
// Tunnel API (UX-2.4: Secure Tunnel Integration)
//
// API для remote troubleshooting через SSH/HTTPS proxy.
// One-time token с TTL 1h, auto-disconnect 30min inactivity.
//
// Endpoints:
//   POST   /devices/{id}/tunnel/token  — создать tunnel token
//   GET    /devices/{id}/tunnel/status  — статус активного tunnel
//   DELETE /devices/{id}/tunnel         — закрыть tunnel принудительно
//   GET    /devices/{id}/tunnel/log     — audit log подключений
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation (Zone 3 → Zone 2)
//   - OWASP ASVS L3 V3.3: One-time token
//   - ISO 27001 A.12.4: Audit trail
//   - Приказ ОАЦ №66 п.7.18.2: mTLS 1.3 для всех соединений
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export type TunnelProtocol = 'ssh' | 'https';

export interface TunnelTokenResponse {
  /** UUID tunnel сессии */
  tunnel_id: string;
  /** Одноразовый токен для подключения */
  token: string;
  /** URL для подключения (SSH или HTTPS) */
  tunnel_url: string;
  /** Протокол */
  protocol: TunnelProtocol;
  /** Когда expires (ISO 8601) */
  expires_at: string;
  /** TTL в секундах */
  ttl_seconds: number;
}

export interface TunnelStatus {
  tunnel_id: string;
  device_id: string;
  status: 'inactive' | 'active' | 'expired' | 'revoked';
  protocol: TunnelProtocol;
  tunnel_url?: string;
  expires_at?: string;
  /** Время неактивности в секундах */
  idle_seconds?: number;
  /** Максимальное время неактивности */
  max_idle_seconds: number;
  /** Количество подключений */
  connection_count: number;
  created_at: string;
}

export interface TunnelLogEntry {
  id: string;
  tunnel_id: string;
  action: 'created' | 'connected' | 'disconnected' | 'expired' | 'revoked';
  protocol: TunnelProtocol;
  remote_ip: string;
  user_agent?: string;
  created_at: string;
  metadata?: Record<string, string>;
}

export interface CreateTunnelRequest {
  protocol: TunnelProtocol;
  /** Разрешённые IP/CIDR для доступа (опционально) */
  allowed_ips?: string[];
}

// ─── API Methods ────────────────────────────────────────────────────

export const tunnelApi = {
  /**
   * Создать одноразовый tunnel token для удалённого доступа.
   * UX-2.4: Генерируется при нажатии кнопки "Generate Tunnel".
   *
   * @param deviceId - ID устройства
   * @param protocol - 'ssh' | 'https'
   * @param allowedIPs - опционально, CIDR для ограничения доступа
   */
  async createToken(
    deviceId: string,
    protocol: TunnelProtocol = 'ssh',
    allowedIPs?: string[],
  ): Promise<TunnelTokenResponse> {
    const body: CreateTunnelRequest = { protocol };
    if (allowedIPs && allowedIPs.length > 0) {
      body.allowed_ips = allowedIPs;
    }
    return request<TunnelTokenResponse>(`/devices/${encodeURIComponent(deviceId)}/tunnel/token`, {
      method: 'POST',
      body: JSON.stringify(body),
    });
  },

  /**
   * Получить статус активного tunnel для устройства.
   */
  async getStatus(deviceId: string): Promise<TunnelStatus> {
    return request<TunnelStatus>(`/devices/${encodeURIComponent(deviceId)}/tunnel/status`);
  },

  /**
   * Принудительно закрыть tunnel сессию.
   */
  async revoke(deviceId: string): Promise<void> {
    await request<void>(`/devices/${encodeURIComponent(deviceId)}/tunnel`, {
      method: 'DELETE',
    });
  },

  /**
   * Получить audit log подключений для tunnel.
   */
  async getLog(deviceId: string): Promise<TunnelLogEntry[]> {
    return request<TunnelLogEntry[]>(`/devices/${encodeURIComponent(deviceId)}/tunnel/log`);
  },
};
