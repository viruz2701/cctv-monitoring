const API_BASE = '/api/v1'; 

let authToken: string | null = localStorage.getItem('token');

export function setAuthToken(token: string | null) {
    authToken = token;
    if (token) {
        localStorage.setItem('token', token);
    } else {
        localStorage.removeItem('token');
    }
}

async function request<T>(
    path: string,
    options: RequestInit = {}
): Promise<T> {
    // Используем Record<string, string> для динамических ключей
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };
    // Объединяем с переданными заголовками
    if (options.headers) {
        Object.assign(headers, options.headers as Record<string, string>);
    }
    if (authToken) {
        headers['Authorization'] = `Bearer ${authToken}`;
    }

    const response = await fetch(`${API_BASE}${path}`, {
        ...options,
        headers,
    });

    if (!response.ok) {
        if (response.status === 401) {
            setAuthToken(null);
            if (typeof window !== 'undefined') {
                window.location.href = '/login';
            }
            throw new Error('Session expired. Please log in again.');
        }
        const errorText = await response.text();
        throw new Error(errorText || `Request failed with status ${response.status}`);
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        return response.json();
    }
    return null as T;
}

// Типы для API
export interface User {
    id: string;
    username: string;
    role: 'admin' | 'support' | 'owner';
    owner_id?: string | null;
    created_at: string;
    avatar?: string;
    sites?: string[];
}

export interface Device {
    device_id: string;
    owner_id?: string | null;
    name?: string;
    location?: string;
    vendor_type?: string;
    status: string;
    last_seen: string;
    registered_at: string;
    user_agent?: string;
}

export interface Alarm {
    device_id: string;
    priority: number;
    method: number;
    description: string;
    timestamp: string;
}

export interface Prediction {
    device_id: string;
    prediction_date: string;
    failure_probability: number;
    explanation: string;
}

export interface ParsedLog {
    device_id: string;
    log_level: string;
    event_code: number;
    message: string;
    source: string;
    timestamp: string;
}

// API методы
export const api = {
    // Аутентификация
    async login(username: string, password: string): Promise<{ token: string; user: User }> {
        const data = await request<{ token: string; user: User }>('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });
        if (data.token) {
            setAuthToken(data.token);
        }
        return data;
    },

    async getCurrentUser(): Promise<User> {
        return request<User>('/users/me');
    },

    logout(): void {
        setAuthToken(null);
    },

    // Устройства
    async getDevices(): Promise<Device[]> {
        return request<Device[]>('/devices');
    },

    async getDevice(deviceId: string): Promise<Device> {
        return request<Device>(`/devices/${deviceId}`);
    },

    async getDeviceStatus(deviceId: string): Promise<{ device_id: string; status: string; last_seen: string }> {
        return request<{ device_id: string; status: string; last_seen: string }>(`/devices/${deviceId}/status`);
    },

    // Алермы
    async getAlarms(deviceId?: string): Promise<Alarm[]> {
        const query = deviceId ? `?device_id=${deviceId}` : '';
        return request<Alarm[]>(`/alarms${query}`);
    },

    // Прогнозы аналитики
    async getPredictions(deviceId?: string): Promise<Prediction[]> {
    const query = deviceId ? `?device_id=${deviceId}` : '';
    const data = await request<Prediction[] | null>(`/analytics/predictions${query}`);
    return data || [];
},

    // Логи
    async searchLogs(params: {
        device_id?: string;
        level?: string;
        keyword?: string;
        time_from?: string;
        time_to?: string;
    }): Promise<ParsedLog[]> {
        const query = new URLSearchParams();
        if (params.device_id) query.append('device_id', params.device_id);
        if (params.level) query.append('level', params.level);
        if (params.keyword) query.append('keyword', params.keyword);
        if (params.time_from) query.append('time_from', params.time_from);
        if (params.time_to) query.append('time_to', params.time_to);
        const url = `/logs/search?${query.toString()}`;
        return request<ParsedLog[]>(url);
    },
};