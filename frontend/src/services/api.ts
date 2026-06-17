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

export async function request<T>(
    path: string,
    options: RequestInit = {}
): Promise<T> {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json',
    };

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
            // НЕ делаем редирект, если уже на странице логина
            if (typeof window !== 'undefined' && !window.location.pathname.includes('/login')) {
                window.location.href = '/login';
            }
            throw new Error('Session expired. Please log in again.');
        }
        if (response.status === 403) {
            throw new Error('Access denied. Insufficient permissions.');
        }
        const errorText = await response.text();
        throw new Error(errorText || `Request failed with status ${response.status}`);
    }

    if (response.status === 204) {
        return null as T;
    }

    const contentType = response.headers.get('content-type');
    if (contentType && contentType.includes('application/json')) {
        return response.json();
    }

    return null as T;
}
// ─── Types ────────────────────────────────────────────────────────────



export interface User {
    id: string;
    username: string;
    name: string; // <-- УБРАЛИ ? (теперь обязательное)
    role: 'admin' | 'support' | 'owner' | 'manager' | 'technician' | 'viewer';
    owner_id?: string | null;
    created_at: string;
    avatar?: string;
    sites?: string[];
    email: string; // <-- УБРАЛИ ? (теперь обязательное)
    status?: 'active' | 'inactive';
    lastLogin?: string;
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
    // P2P fields
    p2p_brand?: string;
    p2p_serial?: string;
    cloud_status?: string;
}

export interface Alarm {
    device_id: string;
    priority: number;
    method: number;
    description: string;
    timestamp: string;
    image_path?: string;
}

export interface Prediction {
    device_id: string;
    prediction_date: string;
    failure_probability: number;
    explanation: string;
    model_version?: string;
}

export interface ParsedLog {
    device_id: string;
    log_level: string;
    event_code: number;
    message: string;
    source: string;
    timestamp: string;
    raw?: string;
}

export interface Site {
    id: string;
    name: string;
    address: string;
    city: string;
    status: 'active' | 'inactive' | 'maintenance';
    last_sync: string;
    created_at: string;
    updated_at: string;
}

export interface Ticket {
    id: string;
    title: string;
    description: string;
    device_id?: string;
    priority: string;
    status: string;
    assignee?: string;
    created_at: string;
    updated_at: string;
    comments?: TicketComment[];
}

export interface TicketComment {
    id: string;
    ticket_id: string;
    user_id?: string;
    user_name?: string;
    content: string;
    created_at: string;
}

export interface Notification {
    id: string;
    user_id: string;
    title: string;
    message: string;
    type: 'success' | 'warning' | 'error' | 'info';
    link?: string;
    read: boolean;
    created_at: string;
}

export interface Report {
    id: string;
    name: string;
    type: string;
    format: string;
    date_range?: string;
    file_url?: string;
    file_name?: string;
    size?: string;
    status: 'ready' | 'expired' | 'generating';
    generated_by?: string;
    generated_at: string;
    expires_at?: string;
}

export interface AuditLogEntry {
    id: string;
    timestamp: string;
    user_id?: string;
    action: string;
    entity_type?: string;
    entity_id?: string;
    old_value?: Record<string, any>;
    new_value?: Record<string, any>;
    ip_address?: string;
}

// ─── Services Settings Types ──────────────────────────────────────────

export interface SyslogSettings {
    enabled: boolean;
    udp_port: number;
    tcp_port: number;
}

export interface FTPSettings {
    enabled: boolean;
    port: number;
    user: string;
    password: string;
    root_path: string;
}

export interface SNMPSettings {
    enabled: boolean;
    port: number;
    community: string;
    version: 'v1' | 'v2c' | 'v3';
    user?: string;
    auth_protocol?: 'MD5' | 'SHA' | 'SHA256';
    auth_password?: string;
    priv_protocol?: 'DES' | 'AES' | 'AES192' | 'AES256';
    priv_password?: string;
}

export interface HTTPSettings {
    enabled: boolean;
    port: number;
}

export interface DahuaSettings {
    enabled: boolean;
    ports: number[];
}

export interface HisiliconSettings {
    enabled: boolean;
    port: number;
}

export interface TVTSettings {
    enabled: boolean;
    port: number;
}

export interface SIPSettings {
    enabled: boolean;
    port: number;
    host: string;
}

export interface GB28181Settings {
    enabled: boolean;
    port: number;
    host: string;
    server_id: string;
    server_ip: string;
    realm: string;
    auth_enabled: boolean;
    auth_user: string;
    auth_password: string;
    auto_catalog: boolean;
    auto_device_info: boolean;
    keepalive_interval: number;
    keepalive_timeout: number;
    max_sub_channels: number;
    log_sip_messages: boolean;
}

export interface P2PGatewaySettings {
    url: string;
    api_key: string;
    enabled?: boolean;
}

export interface ServicesSettings {
    services_syslog: SyslogSettings;
    services_ftp: FTPSettings;
    services_snmp: SNMPSettings;
    services_http: HTTPSettings;
    services_dahua: DahuaSettings;
    services_hisilicon: HisiliconSettings;
    services_tvt: TVTSettings;
    services_sip: SIPSettings;
    services_gb28181?: GB28181Settings;
    services_p2p_gateway: P2PGatewaySettings;
}

// ─── Technician Site Assignments ──────────────────────────────────────

export interface TechnicianSiteAssignment {
    id: string;
    technician_id: string;
    site_id: string;
    is_primary: boolean;
    assigned_at: string;
    assigned_by: string;
    technician_name?: string;
    site_name?: string;
}

// ─── Dashboard Stats ──────────────────────────────────────────────────

export interface DashboardStats {
    total_devices: number;
    online_devices: number;
    offline_devices: number;
    warning_devices: number;
    open_tickets: number;
    critical_tickets: number;
    resolution_rate: number;
    avg_response_time_hours: number;
}

// ─── API Methods ──────────────────────────────────────────────────────

export const api = {
    // ── Authentication ─────────────────────────────────────────────────

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


   

    // ── Devices ────────────────────────────────────────────────────────

    async getDevices(): Promise<Device[]> {
        return request<Device[]>('/devices');
    },

    async getDevice(deviceId: string): Promise<Device> {
        return request<Device>(`/devices/${deviceId}`);
    },

    async getDeviceStatus(deviceId: string): Promise<{ device_id: string; status: string; last_seen: string }> {
        return request<{ device_id: string; status: string; last_seen: string }>(`/devices/${deviceId}/status`);
    },

    async createDevice(device: Partial<Device>): Promise<Device> {
        return request<Device>('/devices', {
            method: 'POST',
            body: JSON.stringify(device),
        });
    },

    async updateDevice(deviceId: string, updates: Partial<Device>): Promise<Device> {
        return request<Device>(`/devices/${deviceId}`, {
            method: 'PUT',
            body: JSON.stringify(updates),
        });
    },

    async deleteDevice(deviceId: string): Promise<void> {
        await request<void>(`/devices/${deviceId}`, {
            method: 'DELETE',
        });
    },

    async getDeviceImages(deviceId: string): Promise<string[]> {
        return request<string[]>(`/images/device/${deviceId}`);
    },

    // ── Alarms ─────────────────────────────────────────────────────────

    async getAlarms(deviceId?: string): Promise<Alarm[]> {
        const query = deviceId ? `?device_id=${deviceId}` : '';
        return request<Alarm[]>(`/alarms${query}`);
    },

    async acknowledgeAlarm(alarmId: string): Promise<void> {
        await request<void>(`/alarms/${alarmId}/acknowledge`, {
            method: 'POST',
        });
    },

    async resolveAlarm(alarmId: string): Promise<void> {
        await request<void>(`/alarms/${alarmId}/resolve`, {
            method: 'POST',
        });
    },

    async deleteAlarm(alarmId: string): Promise<void> {
        await request<void>(`/alarms/${alarmId}`, {
            method: 'DELETE',
        });
    },

    // ── Analytics / Predictions ────────────────────────────────────────

    async getPredictions(deviceId?: string, limit?: number): Promise<Prediction[]> {
        const params = new URLSearchParams();
        if (deviceId) params.append('device_id', deviceId);
        if (limit) params.append('limit', String(limit));
        const query = params.toString() ? `?${params.toString()}` : '';
        const data = await request<Prediction[] | null>(`/analytics/predictions${query}`);
        return data || [];
    },

    async triggerPredictionRun(): Promise<{ status: string }> {
        return request<{ status: string }>('/analytics/predictions/run', {
            method: 'POST',
        });
    },

    // ── Logs ───────────────────────────────────────────────────────────

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

    // ── Sites ──────────────────────────────────────────────────────────

    async getSites(): Promise<Site[]> {
        return request<Site[]>('/sites');
    },

    async getSite(siteId: string): Promise<Site> {
        return request<Site>(`/sites/${siteId}`);
    },

    async createSite(site: Partial<Site>): Promise<Site> {
        return request<Site>('/sites', {
            method: 'POST',
            body: JSON.stringify(site),
        });
    },

    async updateSite(siteId: string, updates: Partial<Site>): Promise<Site> {
        return request<Site>(`/sites/${siteId}`, {
            method: 'PUT',
            body: JSON.stringify(updates),
        });
    },

    async deleteSite(siteId: string): Promise<void> {
        await request<void>(`/sites/${siteId}`, {
            method: 'DELETE',
        });
    },

    // ── Tickets ────────────────────────────────────────────────────────

    async getTickets(): Promise<Ticket[]> {
        return request<Ticket[]>('/tickets');
    },

    async getTicket(ticketId: string): Promise<Ticket> {
        return request<Ticket>(`/tickets/${ticketId}`);
    },

    async createTicket(ticket: Partial<Ticket>): Promise<Ticket> {
        return request<Ticket>('/tickets', {
            method: 'POST',
            body: JSON.stringify(ticket),
        });
    },

    async updateTicket(ticketId: string, updates: Partial<Ticket>): Promise<Ticket> {
        return request<Ticket>(`/tickets/${ticketId}`, {
            method: 'PUT',
            body: JSON.stringify(updates),
        });
    },

    async deleteTicket(ticketId: string): Promise<void> {
        await request<void>(`/tickets/${ticketId}`, {
            method: 'DELETE',
        });
    },

    async addTicketComment(ticketId: string, content: string): Promise<TicketComment> {
        return request<TicketComment>(`/tickets/${ticketId}/comments`, {
            method: 'POST',
            body: JSON.stringify({ content }),
        });
    },

    // ── Users ──────────────────────────────────────────────────────────

    async getUsers(): Promise<User[]> {
        return request<User[]>('/users');
    },

    async getUser(userId: string): Promise<User> {
        return request<User>(`/users/${userId}`);
    },

    async createUser(user: { username: string; password: string; role: string; email?: string }): Promise<User> {
        return request<User>('/users', {
            method: 'POST',
            body: JSON.stringify(user),
        });
    },

    async updateUser(userId: string, updates: Partial<User>): Promise<User> {
        return request<User>(`/users/${userId}`, {
            method: 'PUT',
            body: JSON.stringify(updates),
        });
    },

    async deleteUser(userId: string): Promise<void> {
        await request<void>(`/users/${userId}`, {
            method: 'DELETE',
        });
    },

    async changePassword(currentPassword: string, newPassword: string): Promise<void> {
        await request<void>('/users/me/password', {
            method: 'PUT',
            body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
        });
    },

    async resetUserPassword(userId: string, newPassword: string): Promise<void> {
        await request<void>(`/users/${userId}/reset-password`, {
            method: 'PUT',
            body: JSON.stringify({ new_password: newPassword }),
        });
    },

    // ── Session Management ─────────────────────────────────────────────
    async getSessions(): Promise<any[]> {
        return request<any[]>('/sessions');
    },

    async revokeSession(sessionId: string): Promise<void> {
        await request<void>(`/sessions/${sessionId}`, {
            method: 'DELETE',
        });
    },

    async revokeAllOtherSessions(currentSessionId: string): Promise<void> {
        await request<void>('/sessions/revoke-all', {
            method: 'POST',
            body: JSON.stringify({ current_session_id: currentSessionId }),
        });
    },

    // ── 2FA (TOTP) ─────────────────────────────────────────────────────
    async setup2FA(): Promise<{ secret: string; uri: string }> {
        return request<{ secret: string; uri: string }>('/users/me/2fa/setup', {
            method: 'POST',
        });
    },

    async verify2FA(code: string): Promise<void> {
        await request<void>('/users/me/2fa/verify', {
            method: 'POST',
            body: JSON.stringify({ code }),
        });
    },

    async disable2FA(password: string): Promise<void> {
        await request<void>('/users/me/2fa/disable', {
            method: 'POST',
            body: JSON.stringify({ password }),
        });
    },

    async login2FA(sessionToken: string, code: string): Promise<{ token: string; user: User }> {
        const data = await request<{ token: string; user: User }>('/auth/login/2fa', {
            method: 'POST',
            body: JSON.stringify({ session_token: sessionToken, code }),
        });
        if (data.token) {
            setAuthToken(data.token);
        }
        return data;
    },

    // ── Notifications ──────────────────────────────────────────────────

    async getNotifications(): Promise<Notification[]> {
        return request<Notification[]>('/notifications');
    },

    async markNotificationRead(notificationId: string): Promise<void> {
        await request<void>(`/notifications/${notificationId}/read`, {
            method: 'POST',
        });
    },

    async markAllNotificationsRead(): Promise<void> {
        await request<void>('/notifications/read-all', {
            method: 'POST',
        });
    },

    async deleteNotification(notificationId: string): Promise<void> {
        await request<void>(`/notifications/${notificationId}`, {
            method: 'DELETE',
        });
    },

    async deleteNotifications(ids: string[]): Promise<void> {
        await request<void>('/notifications/bulk-delete', {
            method: 'POST',
            body: JSON.stringify({ ids }),
        });
    },

    // ── Reports ────────────────────────────────────────────────────────

    async getReports(): Promise<Report[]> {
        return request<Report[]>('/reports');
    },

    async generateReport(params: {
        type: string;
        format: string;
        date_range: string;
        filters?: Record<string, any>;
    }): Promise<Report> {
        return request<Report>('/reports/generate', {
            method: 'POST',
            body: JSON.stringify(params),
        });
    },

    async getReportFile(reportId: string): Promise<Blob> {
        const headers: Record<string, string> = {};
        if (authToken) {
            headers['Authorization'] = `Bearer ${authToken}`;
        }
        const response = await fetch(`${API_BASE}/reports/${reportId}/download`, { headers });
        if (!response.ok) throw new Error('Failed to download report');
        return response.blob();
    },

    async deleteReport(reportId: string): Promise<void> {
        await request<void>(`/reports/${reportId}`, {
            method: 'DELETE',
        });
    },

    // ── Dashboard ──────────────────────────────────────────────────────

    async getDashboardStats(): Promise<DashboardStats> {
        return request<DashboardStats>('/dashboard/stats');
    },

    // ── Audit Log ──────────────────────────────────────────────────────

    async getAuditLog(params?: {
        user_id?: string;
        action?: string;
        entity_type?: string;
        time_from?: string;
        time_to?: string;
        limit?: number;
    }): Promise<AuditLogEntry[]> {
        const query = new URLSearchParams();
        if (params?.user_id) query.append('user_id', params.user_id);
        if (params?.action) query.append('action', params.action);
        if (params?.entity_type) query.append('entity_type', params.entity_type);
        if (params?.time_from) query.append('time_from', params.time_from);
        if (params?.time_to) query.append('time_to', params.time_to);
        if (params?.limit) query.append('limit', String(params.limit));
        const url = `/audit-log?${query.toString()}`;
        return request<AuditLogEntry[]>(url);
    },

    // ── Services Settings (Network Protocols) ──────────────────────────

    async getServicesSettings(): Promise<ServicesSettings> {
        return request<ServicesSettings>('/settings/services');
    },

    async updateServicesSettings(settings: Partial<ServicesSettings>): Promise<{ status: string; restarted: string[] }> {
        return request<{ status: string; restarted: string[] }>('/settings/services', {
            method: 'PUT',
            body: JSON.stringify(settings),
        });
    },

    // ── P2P Gateway Management ─────────────────────────────────────────

    async listP2PDevices(): Promise<any[]> {
        return request<any[]>('/p2p/devices');
    },

    async registerP2PDevice(data: {
        brand: string;
        serial: string;
        username?: string;
        password?: string;
        security_code?: string;
        ip_address?: string;
    }): Promise<any> {
        return request<any>('/p2p/devices', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async getP2PDeviceStatus(deviceId: string): Promise<{ device_id: string; status: string; rtsp_url: string }> {
        return request<{ device_id: string; status: string; rtsp_url: string }>(`/p2p/status/${deviceId}`);
    },

    async sendP2PCommand(deviceId: string, command: { command: string; speed?: number }): Promise<void> {
        await request<void>(`/p2p/command/${deviceId}`, {
            method: 'POST',
            body: JSON.stringify(command),
        });
    },

    async getP2PSnapshot(deviceId: string): Promise<Blob> {
        const headers: Record<string, string> = {};
        if (authToken) {
            headers['Authorization'] = `Bearer ${authToken}`;
        }
        const response = await fetch(`${API_BASE}/p2p/snapshot/${deviceId}`, { headers });
        if (!response.ok) throw new Error('Failed to get snapshot');
        return response.blob();
    },

    // ── External Alarms (for integrations) ─────────────────────────────

    async sendExternalAlarm(alarm: {
        device_id: string;
        event_type: string;
        priority: number;
        method: number;
        description: string;
        timestamp?: string;
    }): Promise<void> {
        await request<void>('/external/alarm', {
            method: 'POST',
            body: JSON.stringify(alarm),
        });
    },

    // ── API Keys Management ────────────────────────────────────────────

    async getAPIKeys(): Promise<any[]> {
        return request<any[]>('/api-keys');
    },

    async createAPIKey(data: { name: string; permissions: string[]; expires_at?: string }): Promise<any> {
        return request<any>('/api-keys', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async revokeAPIKey(id: string): Promise<void> {
        await request<void>(`/api-keys/${id}`, {
            method: 'DELETE',
        });
    },

    // ── Telegram Integration ───────────────────────────────────────────

    async generateTelegramLink(): Promise<{ token: string; expires_at: string }> {
        return request<{ token: string; expires_at: string }>('/users/me/telegram/generate-link', {
            method: 'POST',
        });
    },

    async getTelegramStatus(): Promise<{ linked: boolean; alerts: boolean; tfa: boolean }> {
        return request<{ linked: boolean; alerts: boolean; tfa: boolean }>('/users/me/telegram/status');
    },

    async updateTelegramSettings(settings: { alerts: boolean; tfa: boolean }): Promise<void> {
        await request<void>('/users/me/telegram/settings', {
            method: 'POST',
            body: JSON.stringify(settings),
        });
    },

    async requestTelegramLoginCode(username: string): Promise<{ message: string; code: string }> {
        return request<{ message: string; code: string }>('/auth/telegram/request-code', {
            method: 'POST',
            body: JSON.stringify({ username }),
        });
    },

    async verifyTelegramLogin(username: string, code: string): Promise<{ token: string; user: any }> {
        return request<{ token: string; user: any }>('/auth/telegram/verify', {
            method: 'POST',
            body: JSON.stringify({ username, code }),
        });
    },

    // ── Technician Site Assignments ────────────────────────────────────

    async getTechnicianSiteAssignments(filters?: { technician_id?: string; site_id?: string; is_primary?: boolean }): Promise<TechnicianSiteAssignment[]> {
        const params = new URLSearchParams();
        if (filters?.technician_id) params.append('technician_id', filters.technician_id);
        if (filters?.site_id) params.append('site_id', filters.site_id);
        if (filters?.is_primary !== undefined) params.append('is_primary', filters.is_primary.toString());
        const query = params.toString() ? `?${params.toString()}` : '';
        return request<TechnicianSiteAssignment[]>(`/technician-assignments${query}`);
    },

    async createTechnicianSiteAssignment(data: { technician_id: string; site_id: string; is_primary?: boolean }): Promise<TechnicianSiteAssignment> {
        return request<TechnicianSiteAssignment>('/technician-assignments', {
            method: 'POST',
            body: JSON.stringify(data),
        });
    },

    async updateTechnicianSiteAssignment(id: string, data: { is_primary?: boolean }): Promise<void> {
        await request<void>(`/technician-assignments/${id}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        });
    },

    async deleteTechnicianSiteAssignment(id: string): Promise<void> {
        await request<void>(`/technician-assignments/${id}`, {
            method: 'DELETE',
        });
    },
};