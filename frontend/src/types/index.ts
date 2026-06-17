// ═══════════════════════════════════════════════════════════════════════
// Site Types
// ═══════════════════════════════════════════════════════════════════════

export interface Site {
    id: string;
    name: string;
    address: string;
    city: string;
    status: 'active' | 'inactive' | 'maintenance';
    lastSync: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Connection & Protocol Types
// ═══════════════════════════════════════════════════════════════════════

export type ConnectionType = 'ip' | 'p2p' | 'snmp' | 'syslog' | 'alarm' | 'gb28181' | 'onvif';

// GB28181 DeviceID Structure (20 digits per GB/T 28181-2016 standard)
export interface GB28181DeviceIDInfo {
    raw: string;
    typeCode: string;      // 2 digits: device type (11=DVR, 34=IPC, etc.)
    regionCode: string;    // 4 digits: administrative region
    industryCode: string;  // 4 digits: industry code
    networkCode: string;   // 2 digits: network type
    seqNumber: string;     // 8 digits: sequence number
    isValid: boolean;
}

export type GB28181DeviceType = 
    | 'dvr' | 'nvr' | 'hvr' | 'encoder'           // 11-14
    | 'platform' | 'gateway' | 'client'            // 20-22
    | 'ipc' | 'ipc_hd' | 'ipc_hf'                  // 34-36
    | 'alarm_controller' | 'access_control'        // 41-42
    | 'decoder' | 'matrix'                         // 51-52
    | 'unknown';

// ═══════════════════════════════════════════════════════════════════════
// Device Types
// ═══════════════════════════════════════════════════════════════════════

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
    owner_id?: string | null;
    
    // Connection type
    connectionType?: ConnectionType;
    
    // P2P fields (Dahua, Hikvision, Reolink, Xiongmai, EZVIZ)
    p2p_brand?: string;
    p2p_serial?: string;
    p2p_security_code?: string;
    p2p_cloud_user?: string;
    p2p_cloud_pass?: string;
    cloud_status?: 'online' | 'offline' | 'unknown';
    
    // SNMP fields
    snmp_community?: string;
    snmp_version?: 'v1' | 'v2c' | 'v3';
    snmp_user?: string;           // SNMPv3
    snmp_auth_protocol?: string;  // SNMPv3: SHA, MD5
    snmp_auth_password?: string;  // SNMPv3
    snmp_priv_protocol?: string;  // SNMPv3: AES, DES
    snmp_priv_password?: string;  // SNMPv3
    
    // Syslog fields
    syslog_port?: number;
    syslog_protocol?: 'udp' | 'tcp';
    
    // Alarm/Event fields
    alarm_protocol?: 'http' | 'sip' | 'xml' | 'mqtt';
    alarm_webhook_url?: string;
    
    // GB28181 fields (NEW)
    gb28181_device_id?: string;           // 20-digit GB28181 ID
    gb28181_device_type?: GB28181DeviceType;
    gb28181_parent_id?: string;           // Parent NVR/Platform ID
    gb28181_sip_port?: number;            // Device SIP port
    gb28181_realm?: string;               // SIP realm
    gb28181_register_expires?: number;    // Registration expiry (sec)
    gb28181_last_register?: string;       // Last REGISTER timestamp
    gb28181_channel_count?: number;       // Number of child channels (for NVR)
    gb28181_sub_devices?: string[];       // Child device IDs (for NVR)
    
    // ONVIF fields
    onvif_url?: string;
    onvif_username?: string;
    onvif_password?: string;
    onvif_profiles?: string[];
    
    // Extended metadata
    manufacturer?: string;
    serial_number?: string;
    mac_address?: string;
    location_description?: string;
    tags?: string[];
    
    // Performance metrics (latest snapshot)
    metrics?: DeviceMetrics;
}

export interface DeviceMetrics {
    cpuUsage?: number;        // 0-100
    memoryUsage?: number;     // 0-100
    diskUsage?: number;       // 0-100
    temperature?: number;     // Celsius
    networkLatency?: number;  // ms
    packetLoss?: number;      // 0-100
    uptime?: number;          // seconds
    lastMetricsUpdate?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Camera Types
// ═══════════════════════════════════════════════════════════════════════

export interface DeviceCamera {
    id: string;
    name: string;
    deviceId: string;
    status: 'online' | 'offline' | 'warning';
    type: 'fixed' | 'ptz' | 'dome' | 'bullet';
    resolution: string;
    channel: number;
    // GB28181 channel-specific
    gb28181_channel_id?: string;
    gb28181_stream_url?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Recording Calendar Types
// ═══════════════════════════════════════════════════════════════════════

export type RecordingStatus = 'available' | 'missing' | 'no_data';

export interface RecordingDay {
    date: string;
    cameraId: string;
    cameraName: string;
    status: RecordingStatus;
}

// ═══════════════════════════════════════════════════════════════════════
// Health Timeline Types
// ═══════════════════════════════════════════════════════════════════════

export interface HealthTimelineEvent {
    id: string;
    deviceId: string;
    timestamp: string;
    type: 'status_change' | 'alert' | 'maintenance' | 'firmware' | 'restart';
    message: string;
    severity: 'info' | 'warning' | 'error' | 'success';
}

// ═══════════════════════════════════════════════════════════════════════
// Device Stats
// ═══════════════════════════════════════════════════════════════════════

export interface DeviceStats {
    deviceId: string;
    uptimePercent: number;
    hddFreePercent: number;
    cpuUsage: number;
    memoryUsage: number;
    temperature: number;
}

// ═══════════════════════════════════════════════════════════════════════
// Ticket Types
// ═══════════════════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════════════════
// User Types (Extended)
// ═══════════════════════════════════════════════════════════════════════

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
    // Extended fields
    phone?: string;
    department?: string;
    timezone?: string;
    language?: string;
    twoFactorEnabled?: boolean;
    passwordLastChanged?: string;
    failedLoginAttempts?: number;
    lockedUntil?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Dashboard Stats
// ═══════════════════════════════════════════════════════════════════════

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
    // Extended stats
    totalSites?: number;
    gb28181Devices?: number;
    p2pDevices?: number;
    activeAlarms?: number;
    predictionsRun?: number;
}

// ═══════════════════════════════════════════════════════════════════════
// Alert Types
// ═══════════════════════════════════════════════════════════════════════

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
    // Extended
    priority?: 'critical' | 'high' | 'medium' | 'low';
    source?: string;  // syslog, snmp, gb28181, etc.
    acknowledgedBy?: string;
    resolvedBy?: string;
    resolvedAt?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Report Types
// ═══════════════════════════════════════════════════════════════════════

export interface Report {
    id: string;
    name: string;
    type: 'health' | 'uptime' | 'recording' | 'tickets' | 'custom' | 'gb28181_compliance' | 'p2p_connectivity';
    description: string;
    lastGenerated: string;
    schedule: 'daily' | 'weekly' | 'monthly' | 'on_demand';
}

// ═══════════════════════════════════════════════════════════════════════
// App Settings (Core)
// ═══════════════════════════════════════════════════════════════════════

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

// ═══════════════════════════════════════════════════════════════════════
// Notification Types
// ═══════════════════════════════════════════════════════════════════════

export interface Notification {
    id: string;
    title: string;
    message: string;
    type: 'success' | 'warning' | 'error' | 'info';
    timestamp: string;
    read: boolean;
    link?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// P2P Types
// ═══════════════════════════════════════════════════════════════════════

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
    command: 'left' | 'right' | 'up' | 'down' | 'zoom_in' | 'zoom_out' | 'stop';
    speed?: number;
}

// ═══════════════════════════════════════════════════════════════════════
// Services Settings (Network Protocols) - NEW
// ═══════════════════════════════════════════════════════════════════════

export interface SyslogSettings {
    enabled: boolean;
    udp_port: number;
    tcp_port: number;
    max_message_size?: number;
    parse_vendor?: boolean;
}

export interface FTPSettings {
    enabled: boolean;
    port: number;
    user: string;
    password: string;
    root_path: string;
    passive_mode?: boolean;
    passive_port_range?: string;  // e.g., "50000-50100"
}

export interface SNMPSettings {
    enabled: boolean;
    port: number;
    community: string;
    version: 'v1' | 'v2c' | 'v3';
    // SNMPv3 fields
    user?: string;
    auth_protocol?: 'MD5' | 'SHA' | 'SHA256';
    auth_password?: string;
    priv_protocol?: 'DES' | 'AES' | 'AES192' | 'AES256';
    priv_password?: string;
}

export interface HTTPSettings {
    enabled: boolean;
    port: number;
    require_auth?: boolean;
    auth_token?: string;
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
    host: string;
    port: number;
    server_id: string;        // 20-digit server DeviceID
    server_ip: string;        // Public IP for Contact header
    realm: string;            // SIP domain
    auth_enabled: boolean;    // SIP Digest Authentication
    auth_user: string;
    auth_password: string;
    auto_catalog: boolean;    // Auto-request catalog on REGISTER
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

// ═══════════════════════════════════════════════════════════════════════
// Audit Log Types - NEW
// ═══════════════════════════════════════════════════════════════════════

export type AuditAction = 
    | 'create' | 'update' | 'delete' 
    | 'login' | 'logout' | 'password_change'
    | 'settings_change' | 'service_restart'
    | 'device_register' | 'device_unregister';

export interface AuditLogEntry {
    id: string;
    timestamp: string;
    user_id?: string;
    user_name?: string;
    action: AuditAction;
    entity_type: string;      // 'device', 'user', 'site', 'settings', etc.
    entity_id?: string;
    old_value?: Record<string, any>;
    new_value?: Record<string, any>;
    ip_address?: string;
    user_agent?: string;
    details?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// API Key Types - NEW
// ═══════════════════════════════════════════════════════════════════════

export interface APIKey {
    id: string;
    name: string;
    key?: string;            // Only shown on creation
    permissions: string[];
    expires_at?: string;
    last_used_at?: string;
    created_at: string;
    created_by?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Analytics / Prediction Types
// ═══════════════════════════════════════════════════════════════════════

export interface Prediction {
    device_id: string;
    prediction_date: string;
    failure_probability: number;
    explanation: string;
    model_version?: string;
    expected_remaining_hours?: number;
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

// ═══════════════════════════════════════════════════════════════════════
// GB28181 Catalog Item (from NVR response) - NEW
// ═══════════════════════════════════════════════════════════════════════

export interface GB28181CatalogItem {
    device_id: string;
    name: string;
    manufacturer: string;
    model: string;
    owner?: string;
    civil_code?: string;
    address?: string;
    parental: number;
    parent_id?: string;
    safety_way?: number;
    register_way?: number;
    secrecy?: number;
    ip_address?: string;
    port?: number;
    status: 'ON' | 'OFF' | 'VLOST' | 'FAULT';
}

// ═══════════════════════════════════════════════════════════════════════
// System Status Types - NEW
// ═══════════════════════════════════════════════════════════════════════

export interface SystemStatus {
    version: string;
    uptime: number;          // seconds
    cpu_usage: number;
    memory_usage: number;
    disk_usage: number;
    active_connections: number;
    protocols_status: Record<string, {
        enabled: boolean;
        running: boolean;
        port: number;
        connections: number;
    }>;
    database: {
        connected: boolean;
        size_mb: number;
        tables_count: number;
    };
}

// ═══════════════════════════════════════════════════════════════════════
// Helper Types
// ═══════════════════════════════════════════════════════════════════════

export type SortDirection = 'asc' | 'desc';

export interface PaginationParams {
    page: number;
    pageSize: number;
    sortBy?: string;
    sortDirection?: SortDirection;
}

export interface PaginatedResponse<T> {
    data: T[];
    total: number;
    page: number;
    pageSize: number;
    totalPages: number;
}