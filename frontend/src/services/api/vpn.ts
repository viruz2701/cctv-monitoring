// ═══════════════════════════════════════════════════════════════════════
// VPN API Service (SELFSERV-02, EDGE-08)
//
// API для self-service скачивания WireGuard конфигураций.
// Инженер может скачать .conf файл для своей сессии.
//
// Endpoint:
//   GET /api/v1/vpn/sessions/{id}/config — скачать WG .conf файл
//
// Соответствие:
//   - OWASP ASVS L3 V3.3: Privilege escalation prevention
//   - IEC 62443-3-3 SR 2.1: Authorisation enforcement
// ═══════════════════════════════════════════════════════════════════════

import { request, requestBlob } from './client';
import { API_BASE } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface VPNSession {
    id: string;
    agent_id: string;
    engineer_id: string;
    started_at: string;
    expires_at: string;
    allowed_ips: string[];
    public_key: string;
    status: 'active' | 'expired' | 'revoked';
    bytes_transferred: number;
    created_at: string;
    closed_at?: string | null;
}

export interface WGClientConfig {
    interface: {
        private_key: string;
        address: string;
        dns?: string[];
    };
    peer: {
        public_key: string;
        allowed_ips: string[];
        endpoint: string;
        persistent_keepalive: number;
    };
}

// ─── API Methods ───────────────────────────────────────────────────

/**
 * Скачать WireGuard .conf файл для активной VPN сессии.
 * SELFSERV-02: Self-service с проверкой ownership.
 *
 * @param sessionId - UUID сессии
 * @returns Promise с текстом .conf файла
 * @throws Error если сессия не принадлежит текущему пользователю
 */
export async function downloadVPNConfig(sessionId: string): Promise<string> {
    const blob = await requestBlob(`/vpn/sessions/${sessionId}/config`);
    return await blob.text();
}

/**
 * Скачать WireGuard .conf файл и скачать его через браузер.
 * SELFSERV-02: Альтернативный метод через создание ссылки.
 */
export async function downloadVPNConfigFile(sessionId: string): Promise<void> {
    const response = await fetch(`${API_BASE}/vpn/sessions/${sessionId}/config`, {
        credentials: 'include',
        headers: {
            'Accept': 'application/x-wireguard-config',
        },
    });

    if (!response.ok) {
        if (response.status === 403) {
            throw new Error('Access denied. You are not the owner of this session.');
        }
        if (response.status === 400) {
            const text = await response.text();
            let msg = 'Session is not active';
            try {
                const parsed = JSON.parse(text);
                msg = parsed?.error?.message || msg;
            } catch { /* ignore */ }
            throw new Error(msg);
        }
        throw new Error(`Failed to download config: ${response.statusText}`);
    }

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `cctv-vpn-${sessionId.slice(0, 8)}.conf`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    window.URL.revokeObjectURL(url);
}

/**
 * Получить список VPN сессий текущего пользователя.
 */
export async function listVPNSessions(): Promise<VPNSession[]> {
    return request<VPNSession[]>('/vpn/sessions');
}

/**
 * Получить детали VPN сессии.
 */
export async function getVPNSession(sessionId: string): Promise<VPNSession> {
    return request<VPNSession>(`/vpn/sessions/${sessionId}`);
}
