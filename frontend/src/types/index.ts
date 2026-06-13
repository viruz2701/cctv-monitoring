// Site Types
export interface Site {
    id: string;
    name: string;
    address: string;
    city: string;
    status: 'active' | 'inactive' | 'maintenance';
    lastSync: string;
}

// Тип подключения устройства
export type ConnectionType = 'ip' | 'p2p' | 'snmp' | 'syslog' | 'alarm';

// Device Types
export interface Device {
    id: string;
    name: string;
    siteId: string;
    siteName: string;
    type: 'camera' | 'nvr' | 'dvr' | 'switch';
    status: 'online' | 'offline' | 'warning';
    health: 'healthy' | 'faulty' | 'degraded';
    recordingStatus: 'recording' | 'not_recording' | 'scheduled';
    lastSeen: string;
    ipAddress: string;
    model: string;
    firmware: string;
    owner_id?: string | null;   // добавлено
    // P2P поля
    connectionType?: ConnectionType;
    p2p_brand?: string;      // бренд P2P устройства
    p2p_serial?: string;     // серийный номер для P2P
    p2p_security_code?: string;
    p2p_cloud_user?: string;
    p2p_cloud_pass?: string;
    cloud_status?: string;    // статус облачного соединения (online/offline/unknown)
    // SNMP поля
    snmp_community?: string;
    snmp_version?: 'v1' | 'v2c' | 'v3';
    // Syslog поля
    syslog_port?: number;
    // Alarm поля
    alarm_protocol?: 'http' | 'sip' | 'xml';
}

// Camera Types
export interface DeviceCamera {
    id: string;
    name: string;
    deviceId: string;
    status: 'online' | 'offline' | 'warning';
    type: 'fixed' | 'ptz' | 'dome' | 'bullet';
    resolution: string;
    channel: number;
}

// Recording Calendar Types
export type RecordingStatus = 'available' | 'missing' | 'no_data';
export interface RecordingDay {
    date: string;
    cameraId: string;
    cameraName: string;
    status: RecordingStatus;
}

// Health Timeline Types
export interface HealthTimelineEvent {
    id: string;
    deviceId: string;
    timestamp: string;
    type: 'status_change' | 'alert' | 'maintenance' | 'firmware' | 'restart';
    message: string;
    severity: 'info' | 'warning' | 'error' | 'success';
}

// Device Stats
export interface DeviceStats {
    deviceId: string;
    uptimePercent: number;
    hddFreePercent: number;
    cpuUsage: number;
    memoryUsage: number;
    temperature: number;
}

// Ticket Types
export type TicketPriority = 'critical' | 'high' | 'medium' | 'low';
export type TicketStatus = 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed';
export interface Ticket {
    id: string;
    title: string;
    description: string;
    deviceId: string;
    deviceName: string;
    siteName: string;
    priority: TicketPriority;
    status: TicketStatus;
    assignee: string;
    createdAt: string;
    updatedAt: string;
    comments?: TicketComment[];
}
export interface TicketComment {
    id: string;
    ticketId: string;
    userId: string;
    userName: string;
    userAvatar?: string;
    content: string;
    createdAt: string;
}

// User Types
export type UserRole = 'admin' | 'manager' | 'technician' | 'viewer' | 'owner' | 'support';
export type UserStatus = 'active' | 'inactive';
export interface User {
    id: string;
    name: string;
    email: string;
    role: UserRole;
    avatar: string;
    status: UserStatus;
    lastLogin: string;
    sites: string[];
}

// Dashboard Stats
export interface DashboardStats {
    totalDevices: number;
    onlineDevices: number;
    offlineDevices: number;
    healthyDevices: number;
    faultyDevices: number;
    recordingMissing: number;
    openTickets: number;
    criticalTickets: number;
    resolutionRate: number;
    avgResponseTime: number;
}

// Alert Types
export type AlertStatus = 'active' | 'acknowledged' | 'resolved';
export interface Alert {
    id: string;
    type: 'error' | 'warning' | 'info';
    status: AlertStatus;
    message: string;
    deviceId: string;
    deviceName: string;
    siteName: string;
    timestamp: string;
}

// Report Types
export interface Report {
    id: string;
    name: string;
    type: 'health' | 'uptime' | 'recording' | 'tickets' | 'custom';
    description: string;
    lastGenerated: string;
    schedule: 'daily' | 'weekly' | 'monthly' | 'on_demand';
}

// App Settings
export interface AppSettings {
    organizationName: string;
    systemEmail: string;
    timezone: string;
    dateFormat: string;
    notifications: {
        deviceOffline: boolean;
        securityAlerts: boolean;
        storageWarnings: boolean;
        dailyReports: boolean;
        mobilePush: boolean;
    };
    system: {
        healthCheckInterval: number;
        sessionTimeout: number;
        maxRecordingGap: number;
        alertThreshold: number;
    };
    security: {
        requires2FA: boolean;
        passwordPolicy: 'basic' | 'strong';
    };
}

export interface DashboardLayoutConfig {
    showStatsRow: boolean;
    showTicketStats: boolean;
    showRecentAlerts: boolean;
    showLatestTickets: boolean;
    showQuickActions: boolean;
}

// Notification Types
export interface Notification {
    id: string;
    title: string;
    message: string;
    type: 'success' | 'warning' | 'error' | 'info';
    timestamp: string;
    read: boolean;
    link?: string;
}

// P2P types
export interface P2PDevice {
    id: string;
    serial: string;
    brand: string;
    status: 'online' | 'offline' | 'unknown';
    lastSeen?: string;
    rtspUrl?: string;
}

export interface P2PRegistrationForm {
    serial: string;
    brand: string;
    securityCode: string;
    username?: string;
    password?: string;
}

export interface PTZCommand {
    command: 'left' | 'right' | 'up' | 'down' | 'zoom_in' | 'zoom_out';
    speed?: number;
}