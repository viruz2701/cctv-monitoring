// ═══════════════════════════════════════════════════════════════════════
// Edge Proxy API (PROXY-01, PROXY-02, PROXY-04)
//
// API клиент для Zero-Touch Edge Proxy:
//   - HTTP прокси к камере через WireGuard VPN
//   - WebSocket SSH терминал
//   - Lazy VPN создание сессии
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5: Input validation
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════

import { request, API_BASE } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface ProxyError {
  error: {
    code: string;
    message: string;
    trace_id?: string;
  };
}

export interface VPNSessionStatus {
  id: string;
  agent_id: string;
  status: 'active' | 'expired' | 'revoked';
  expires_at: string;
  allowed_ips: string[];
}

// ─── API Methods ────────────────────────────────────────────────────

export const edgeProxyApi = {
  /**
   * Создаёт или получает активную VPN сессию для agent_id.
   * PROXY-03: Lazy VPN — сессия создаётся при первом обращении.
   *
   * @param agentId - ID edge-агента
   * @param allowedIPs - разрешённые IP/CIDR для сессии
   */
  async ensureVPNSession(agentId: string, allowedIPs: string[]): Promise<VPNSessionStatus> {
    return request<VPNSessionStatus>('/vpn/sessions', {
      method: 'POST',
      body: JSON.stringify({
        agent_id: agentId,
        allowed_ips: allowedIPs,
        duration: '1h',
      }),
    });
  },

  /**
   * Получает список активных VPN сессий.
   */
  async getVPNSessions(agentId?: string): Promise<VPNSessionStatus[]> {
    const params = agentId ? `?agent_id=${encodeURIComponent(agentId)}` : '';
    return request<VPNSessionStatus[]>(`/vpn/sessions${params}`);
  },

  /**
   * Генерирует URL для прокси HTTP-запроса к устройству.
   * PROXY-01: HTTP прокси.
   *
   * @param agentId - ID edge-агента
   * @param deviceIp - IP устройства в LAN агента
   * @param port - порт (80, 443, 8080, etc.)
   * @param path - опциональный путь
   */
  getProxyUrl(agentId: string, deviceIp: string, port: number, path?: string): string {
    const basePath = `${API_BASE}/edge/proxy/${encodeURIComponent(agentId)}/${deviceIp}:${port}`;
    if (path) {
      return `${basePath}/${path.replace(/^\//, '')}`;
    }
    return basePath;
  },

  /**
   * Генерирует URL для WebSocket SSH терминала.
   * PROXY-02: SSH через WebSocket.
   *
   * @param agentId - ID edge-агента
   * @param deviceIp - IP устройства в LAN агента
   * @param port - SSH порт (обычно 22)
   */
  getSSHWebSocketUrl(agentId: string, deviceIp: string, port: number = 22): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}${API_BASE}/edge/ssh/${encodeURIComponent(agentId)}/${deviceIp}/${port}`;
  },
};
