import { P2PDevice, P2PRegistrationForm, PTZCommand } from '../types';

const API_BASE = import.meta.env.VITE_API_URL || '/api/v1';

const getAuthToken = (): string | null => localStorage.getItem('token');

type RequestOptions = RequestInit & { responseType?: 'blob' };

async function request<T>(
    path: string,
    options: RequestOptions = {}
): Promise<T> {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
        ...(options.headers as Record<string, string>),
    };
    const token = getAuthToken();
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }
    const response = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers,
    });
    if (!response.ok) {
        if (response.status === 401) {
            window.location.href = '/login';
        }
        const errorText = await response.text();
        throw new Error(errorText || `Request failed with status ${response.status}`);
    }
    if (options.responseType === 'blob') {
        return (await response.blob()) as T;
    }
    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        return response.json();
    }
    return null as T;
}

export const p2pApi = {
    getDevices: () => request<P2PDevice[]>('/p2p/devices'),
    register: (data: P2PRegistrationForm) => request<void>('/p2p/devices', { method: 'POST', body: JSON.stringify(data) }),
    getStatus: (deviceId: string) => request<{ status: string; rtsp_url: string }>(`/p2p/status/${deviceId}`),
    sendCommand: (deviceId: string, command: PTZCommand) => request<void>(`/p2p/command/${deviceId}`, { method: 'POST', body: JSON.stringify(command) }),
    getSnapshot: (deviceId: string) => request<Blob>(`/p2p/snapshot/${deviceId}`, { responseType: 'blob' }),
};