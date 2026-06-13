import type { Site, Device, Ticket, User, DashboardStats, Alert, Report, DeviceCamera, RecordingDay, HealthTimelineEvent, DeviceStats } from '../types';

// Sites Data
export const sites: Site[] = [
    {
        id: 'site-001',
        name: 'Downtown Mall',
        address: '123 Main Street',
        city: 'New York',
        status: 'active',
        lastSync: '2026-02-09T14:00:00Z',
    },
    {
        id: 'site-002',
        name: 'Central Bank HQ',
        address: '456 Finance Ave',
        city: 'Chicago',
        status: 'active',
        lastSync: '2026-02-09T14:05:00Z',
    },
    {
        id: 'site-003',
        name: 'Airport Terminal B',
        address: '789 Airport Blvd',
        city: 'Los Angeles',
        status: 'active',
        lastSync: '2026-02-09T13:55:00Z',
    },
    {
        id: 'site-004',
        name: 'Harbor Warehouse',
        address: '321 Port Street',
        city: 'Seattle',
        status: 'maintenance',
        lastSync: '2026-02-09T12:30:00Z',
    },
    {
        id: 'site-005',
        name: 'Tech Campus Alpha',
        address: '555 Innovation Dr',
        city: 'San Francisco',
        status: 'active',
        lastSync: '2026-02-09T14:10:00Z',
    },
];

// Devices Data
export const devices: Device[] = [
    {
        id: 'dev-001',
        name: 'Entrance Cam 1',
        siteId: 'site-001',
        siteName: 'Downtown Mall',
        type: 'camera',
        status: 'online',
        health: 'healthy',
        recordingStatus: 'recording',
        lastSeen: '2026-02-09T14:15:00Z',
        ipAddress: '192.168.1.101',
        model: 'Hikvision DS-2CD2143G2',
        firmware: 'v5.7.1',
    },
    {
        id: 'dev-002',
        name: 'Parking Lot NVR',
        siteId: 'site-001',
        siteName: 'Downtown Mall',
        type: 'nvr',
        status: 'online',
        health: 'healthy',
        recordingStatus: 'recording',
        lastSeen: '2026-02-09T14:14:00Z',
        ipAddress: '192.168.1.102',
        model: 'Hikvision DS-7616NI',
        firmware: 'v4.52.105',
    },
    {
        id: 'dev-003',
        name: 'Lobby Cam 2',
        siteId: 'site-002',
        siteName: 'Central Bank HQ',
        type: 'camera',
        status: 'offline',
        health: 'faulty',
        recordingStatus: 'not_recording',
        lastSeen: '2026-02-09T10:30:00Z',
        ipAddress: '192.168.2.103',
        model: 'Axis P3245-V',
        firmware: 'v10.12.114',
    },
    {
        id: 'dev-004',
        name: 'Server Room Cam',
        siteId: 'site-002',
        siteName: 'Central Bank HQ',
        type: 'camera',
        status: 'online',
        health: 'degraded',
        recordingStatus: 'recording',
        lastSeen: '2026-02-09T14:10:00Z',
        ipAddress: '192.168.2.104',
        model: 'Dahua IPC-HFW2831S',
        firmware: 'v2.820.0',
    },
    {
        id: 'dev-005',
        name: 'Terminal Gate A1',
        siteId: 'site-003',
        siteName: 'Airport Terminal B',
        type: 'camera',
        status: 'online',
        health: 'healthy',
        recordingStatus: 'recording',
        lastSeen: '2026-02-09T14:15:00Z',
        ipAddress: '192.168.3.105',
        model: 'Hanwha XNV-8082R',
        firmware: 'v1.41.02',
    },
    {
        id: 'dev-006',
        name: 'Baggage Claim DVR',
        siteId: 'site-003',
        siteName: 'Airport Terminal B',
        type: 'dvr',
        status: 'warning',
        health: 'degraded',
        recordingStatus: 'scheduled',
        lastSeen: '2026-02-09T14:00:00Z',
        ipAddress: '192.168.3.106',
        model: 'Hikvision DS-7332HUHI',
        firmware: 'v4.40.105',
    },
    {
        id: 'dev-007',
        name: 'Warehouse Entry',
        siteId: 'site-004',
        siteName: 'Harbor Warehouse',
        type: 'camera',
        status: 'offline',
        health: 'faulty',
        recordingStatus: 'not_recording',
        lastSeen: '2026-02-08T23:45:00Z',
        ipAddress: '192.168.4.107',
        model: 'Vivotek IB9387-HT',
        firmware: 'v0205a',
    },
    {
        id: 'dev-008',
        name: 'Loading Dock Cam',
        siteId: 'site-004',
        siteName: 'Harbor Warehouse',
        type: 'camera',
        status: 'online',
        health: 'healthy',
        recordingStatus: 'recording',
        lastSeen: '2026-02-09T14:12:00Z',
        ipAddress: '192.168.4.108',
        model: 'Bosch NBE-6502-AL',
        firmware: 'v7.82',
    },
];

// Tickets Data
const baseTickets: Ticket[] = [
    {
        id: 'TKT-001',
        title: 'Camera offline - Lobby Cam 2',
        description: 'Camera went offline and is not responding to ping. Needs physical inspection.',
        deviceId: 'dev-003',
        deviceName: 'Lobby Cam 2',
        siteName: 'Central Bank HQ',
        priority: 'critical',
        status: 'open',
        assignee: 'John Smith',
        createdAt: '2026-02-09T10:35:00Z',
        updatedAt: '2026-02-09T11:00:00Z',
        comments: [
            {
                id: 'cm-001',
                ticketId: 'TKT-001',
                userId: 'user-001',
                userName: 'John Smith',
                content: 'Scheduled technician visit for tomorrow.',
                createdAt: '2026-02-09T10:45:00Z',
            }
        ]
    },
    {
        id: 'TKT-002',
        title: 'Recording gap detected',
        description: 'DVR showing 2-hour recording gap from 02:00 to 04:00 AM.',
        deviceId: 'dev-006',
        deviceName: 'Baggage Claim DVR',
        siteName: 'Airport Terminal B',
        priority: 'high',
        status: 'in_progress',
        assignee: 'Sarah Johnson',
        createdAt: '2026-02-09T08:00:00Z',
        updatedAt: '2026-02-09T12:30:00Z',
    },
    {
        id: 'TKT-003',
        title: 'Firmware update required',
        description: 'Server room camera firmware is outdated and needs security patch.',
        deviceId: 'dev-004',
        deviceName: 'Server Room Cam',
        siteName: 'Central Bank HQ',
        priority: 'medium',
        status: 'pending',
        assignee: 'Mike Wilson',
        createdAt: '2026-02-08T16:00:00Z',
        updatedAt: '2026-02-09T09:00:00Z',
    },
    {
        id: 'TKT-004',
        title: 'Warehouse camera down',
        description: 'Entry camera not responding since last night. Possible power issue.',
        deviceId: 'dev-007',
        deviceName: 'Warehouse Entry',
        siteName: 'Harbor Warehouse',
        priority: 'critical',
        status: 'open',
        assignee: 'John Smith',
        createdAt: '2026-02-09T06:00:00Z',
        updatedAt: '2026-02-09T06:00:00Z',
    },
    {
        id: 'TKT-005',
        title: 'Low disk space warning',
        description: 'NVR storage at 85% capacity. Consider archiving or expanding.',
        deviceId: 'dev-002',
        deviceName: 'Parking Lot NVR',
        siteName: 'Downtown Mall',
        priority: 'low',
        status: 'open',
        assignee: 'Sarah Johnson',
        createdAt: '2026-02-07T14:00:00Z',
        updatedAt: '2026-02-07T14:00:00Z',
    },
];

const seedRandom = (seed: number) => {
    let s = seed;
    return () => {
        s = (s * 16807 + 0) % 2147483647;
        return s / 2147483647;
    };
};

const randTicket = seedRandom(12345);

const generateTickets = (): Ticket[] => {
    const list = [...baseTickets];
    const today = new Date('2026-02-09T12:00:00Z');

    // Add ~200 tickets spanning the past 365 days
    for (let i = 1; i <= 200; i++) {
        const d = new Date(today);
        d.setDate(d.getDate() - Math.floor(randTicket() * 365));
        d.setHours(Math.floor(randTicket() * 24));
        const dtStr = d.toISOString();

        const device = devices[Math.floor(randTicket() * devices.length)];
        const priorities: ('low' | 'medium' | 'high' | 'critical')[] = ['low', 'medium', 'high', 'critical'];
        const statuses: ('open' | 'in_progress' | 'resolved' | 'closed' | 'pending')[] = ['resolved', 'closed', 'resolved', 'closed', 'open', 'in_progress', 'pending'];

        list.push({
            id: `TKT-GEN-${i.toString().padStart(3, '0')}`,
            title: `Autogenerated Issue for ${device.name}`,
            description: 'This ticket was autogenerated for 1-year historical reporting test.',
            deviceId: device.id,
            deviceName: device.name,
            siteName: device.siteName,
            priority: priorities[Math.floor(randTicket() * priorities.length)],
            status: statuses[Math.floor(randTicket() * statuses.length)],
            assignee: ['John Smith', 'Sarah Johnson', 'Mike Wilson'][Math.floor(randTicket() * 3)],
            createdAt: dtStr,
            updatedAt: dtStr,
        });
    }
    return list.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime());
};

export const tickets: Ticket[] = generateTickets();

// Users Data
export const users: User[] = [
    {
        id: 'user-001',
        name: 'John Smith',
        email: 'john.smith@company.com',
        role: 'admin',
        avatar: 'JS',
        status: 'active',
        lastLogin: '2026-02-09T14:00:00Z',
        sites: ['site-001', 'site-002', 'site-003', 'site-004', 'site-005'],
    },
    {
        id: 'user-002',
        name: 'Sarah Johnson',
        email: 'sarah.johnson@company.com',
        role: 'manager',
        avatar: 'SJ',
        status: 'active',
        lastLogin: '2026-02-09T13:30:00Z',
        sites: ['site-001', 'site-003'],
    },
    {
        id: 'user-003',
        name: 'Mike Wilson',
        email: 'mike.wilson@company.com',
        role: 'technician',
        avatar: 'MW',
        status: 'active',
        lastLogin: '2026-02-09T12:00:00Z',
        sites: ['site-002', 'site-004'],
    },
    {
        id: 'user-004',
        name: 'Emily Davis',
        email: 'emily.davis@company.com',
        role: 'viewer',
        avatar: 'ED',
        status: 'inactive',
        lastLogin: '2026-02-01T09:00:00Z',
        sites: ['site-005'],
    },
];

// Dashboard Stats
export const dashboardStats: DashboardStats = {
    totalDevices: 248,
    onlineDevices: 233,
    offlineDevices: 15,
    healthyDevices: 220,
    faultyDevices: 12,
    recordingMissing: 8,
    openTickets: 23,
    criticalTickets: 5,
    resolutionRate: 94,
    avgResponseTime: 2.4,
};

// Recent Alerts
const baseAlerts: Alert[] = [
    {
        id: 'alert-001',
        type: 'error',
        message: 'Camera went offline',
        deviceId: 'dev-003',
        deviceName: 'Lobby Cam 2',
        siteName: 'Central Bank HQ',
        status: 'active',
        timestamp: '2026-02-09T10:30:00Z',
    },
    {
        id: 'alert-002',
        type: 'warning',
        message: 'Recording gap detected (2 hours)',
        deviceId: 'dev-006',
        deviceName: 'Baggage Claim DVR',
        siteName: 'Airport Terminal B',
        status: 'acknowledged',
        timestamp: '2026-02-09T08:00:00Z',
    },
    {
        id: 'alert-003',
        type: 'error',
        message: 'Device not responding',
        deviceId: 'dev-007',
        deviceName: 'Warehouse Entry',
        siteName: 'Harbor Warehouse',
        status: 'active',
        timestamp: '2026-02-09T06:00:00Z',
    },
    {
        id: 'alert-004',
        type: 'warning',
        message: 'Storage capacity at 85%',
        deviceId: 'dev-002',
        deviceName: 'Parking Lot NVR',
        siteName: 'Downtown Mall',
        status: 'resolved',
        timestamp: '2026-02-07T14:00:00Z',
    },
    {
        id: 'alert-005',
        type: 'info',
        message: 'Firmware update available',
        deviceId: 'dev-004',
        deviceName: 'Server Room Cam',
        siteName: 'Central Bank HQ',
        status: 'resolved',
        timestamp: '2026-02-08T16:00:00Z',
    },
];

const randAlert = seedRandom(54321);

const generateAlerts = (): Alert[] => {
    const list = [...baseAlerts];
    const today = new Date('2026-02-09T12:00:00Z');

    // Add ~250 alerts spanning the past 365 days
    for (let i = 1; i <= 250; i++) {
        const d = new Date(today);
        d.setDate(d.getDate() - Math.floor(randAlert() * 365));
        d.setHours(Math.floor(randAlert() * 24));
        const dtStr = d.toISOString();

        const device = devices[Math.floor(randAlert() * devices.length)];
        const types: ('error' | 'warning' | 'info')[] = ['error', 'warning', 'info'];
        const statuses: ('active' | 'acknowledged' | 'resolved')[] = ['resolved', 'resolved', 'active', 'acknowledged'];
        const msgs = ['Motion event out of hours', 'SMART self-test warning', 'Connection unstable', 'Video signal lost temporarily', 'High CPU utilization'];

        list.push({
            id: `alert-GEN-${i.toString().padStart(3, '0')}`,
            type: types[Math.floor(randAlert() * types.length)],
            message: msgs[Math.floor(randAlert() * msgs.length)],
            deviceId: device.id,
            deviceName: device.name,
            siteName: device.siteName,
            status: statuses[Math.floor(randAlert() * statuses.length)],
            timestamp: dtStr,
        });
    }
    return list.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
};

export const alerts: Alert[] = generateAlerts();

// Reports Data
const baseReports: Report[] = [
    {
        id: 'report-001',
        name: 'Daily Health Summary',
        type: 'health',
        description: 'Overview of all device health status and issues encountered.',
        lastGenerated: '2026-02-09T06:00:00Z',
        schedule: 'daily',
    },
    {
        id: 'report-002',
        name: 'Weekly Uptime Report',
        type: 'uptime',
        description: 'Detailed uptime statistics for all devices across sites.',
        lastGenerated: '2026-02-03T00:00:00Z',
        schedule: 'weekly',
    },
    {
        id: 'report-003',
        name: 'Recording Compliance',
        type: 'recording',
        description: 'Analysis of recording gaps and storage utilization.',
        lastGenerated: '2026-02-09T07:00:00Z',
        schedule: 'daily',
    },
    {
        id: 'report-004',
        name: 'Monthly Ticket Analysis',
        type: 'tickets',
        description: 'Summary of tickets, resolution times, and trends.',
        lastGenerated: '2026-02-01T00:00:00Z',
        schedule: 'monthly',
    },
];

const randReport = seedRandom(98765);

const generateReports = (): Report[] => {
    const list = [...baseReports];
    const today = new Date('2026-02-09T12:00:00Z');

    // Add ~50 historical reports spanning 365 days
    for (let i = 1; i <= 50; i++) {
        const d = new Date(today);
        d.setDate(d.getDate() - Math.floor(randReport() * 365));
        d.setHours(Math.floor(randReport() * 24));
        const dtStr = d.toISOString();

        const types: ('health' | 'uptime' | 'recording' | 'tickets')[] = ['health', 'uptime', 'recording', 'tickets'];
        const typeObj = types[Math.floor(randReport() * types.length)];

        list.push({
            id: `report-GEN-${i.toString().padStart(3, '0')}`,
            name: `Generated ${typeObj} Archive Report`,
            type: typeObj,
            description: `On-demand archive generation for ${typeObj}.`,
            lastGenerated: dtStr,
            schedule: 'on_demand',
        });
    }
    return list.sort((a, b) => new Date(b.lastGenerated).getTime() - new Date(a.lastGenerated).getTime());
};

export const reports: Report[] = generateReports();

// Current User (for header)
export const currentUser: User = users[0];

// Device Cameras Data
export const deviceCameras: DeviceCamera[] = [
    // Downtown Mall - dev-001 (Entrance Cam 1)
    { id: 'cam-001', name: 'Front Entrance', deviceId: 'dev-001', status: 'online', type: 'dome', resolution: '4K (3840×2160)', channel: 1 },
    { id: 'cam-002', name: 'Side Entrance', deviceId: 'dev-001', status: 'online', type: 'bullet', resolution: '1080p (1920×1080)', channel: 2 },
    // Downtown Mall - dev-002 (Parking Lot NVR)
    { id: 'cam-003', name: 'Parking Level 1', deviceId: 'dev-002', status: 'online', type: 'fixed', resolution: '4K (3840×2160)', channel: 1 },
    { id: 'cam-004', name: 'Parking Level 2', deviceId: 'dev-002', status: 'online', type: 'fixed', resolution: '1080p (1920×1080)', channel: 2 },
    { id: 'cam-005', name: 'Parking Entrance', deviceId: 'dev-002', status: 'online', type: 'bullet', resolution: '4K (3840×2160)', channel: 3 },
    { id: 'cam-006', name: 'Parking Exit', deviceId: 'dev-002', status: 'warning', type: 'bullet', resolution: '1080p (1920×1080)', channel: 4 },
    // Central Bank HQ - dev-003 (Lobby Cam 2 - offline)
    { id: 'cam-007', name: 'Main Lobby', deviceId: 'dev-003', status: 'offline', type: 'ptz', resolution: '4K (3840×2160)', channel: 1 },
    { id: 'cam-008', name: 'Reception Desk', deviceId: 'dev-003', status: 'offline', type: 'dome', resolution: '1080p (1920×1080)', channel: 2 },
    // Central Bank HQ - dev-004 (Server Room Cam - degraded)
    { id: 'cam-009', name: 'Server Rack A', deviceId: 'dev-004', status: 'online', type: 'fixed', resolution: '1080p (1920×1080)', channel: 1 },
    { id: 'cam-010', name: 'Server Rack B', deviceId: 'dev-004', status: 'warning', type: 'fixed', resolution: '1080p (1920×1080)', channel: 2 },
    { id: 'cam-011', name: 'Server Room Entry', deviceId: 'dev-004', status: 'online', type: 'dome', resolution: '4K (3840×2160)', channel: 3 },
    // Airport Terminal B - dev-005
    { id: 'cam-012', name: 'Gate A1 Boarding', deviceId: 'dev-005', status: 'online', type: 'ptz', resolution: '4K (3840×2160)', channel: 1 },
    { id: 'cam-013', name: 'Gate A1 Corridor', deviceId: 'dev-005', status: 'online', type: 'bullet', resolution: '1080p (1920×1080)', channel: 2 },
    // Airport Terminal B - dev-006 (Baggage Claim DVR - warning)
    { id: 'cam-014', name: 'Carousel 1', deviceId: 'dev-006', status: 'online', type: 'ptz', resolution: '1080p (1920×1080)', channel: 1 },
    { id: 'cam-015', name: 'Carousel 2', deviceId: 'dev-006', status: 'warning', type: 'ptz', resolution: '1080p (1920×1080)', channel: 2 },
    { id: 'cam-016', name: 'Exit Door', deviceId: 'dev-006', status: 'online', type: 'bullet', resolution: '720p (1280×720)', channel: 3 },
    // Harbor Warehouse - dev-007 (offline)
    { id: 'cam-017', name: 'Warehouse Gate', deviceId: 'dev-007', status: 'offline', type: 'bullet', resolution: '1080p (1920×1080)', channel: 1 },
    { id: 'cam-018', name: 'Loading Bay', deviceId: 'dev-007', status: 'offline', type: 'fixed', resolution: '1080p (1920×1080)', channel: 2 },
    // Harbor Warehouse - dev-008
    { id: 'cam-019', name: 'Dock Area 1', deviceId: 'dev-008', status: 'online', type: 'ptz', resolution: '4K (3840×2160)', channel: 1 },
    { id: 'cam-020', name: 'Dock Area 2', deviceId: 'dev-008', status: 'online', type: 'fixed', resolution: '1080p (1920×1080)', channel: 2 },
];

// Device Stats
export const deviceStatsData: DeviceStats[] = [
    { deviceId: 'dev-001', uptimePercent: 99.8, hddFreePercent: 42, cpuUsage: 35, memoryUsage: 61, temperature: 42 },
    { deviceId: 'dev-002', uptimePercent: 99.5, hddFreePercent: 15, cpuUsage: 72, memoryUsage: 78, temperature: 55 },
    { deviceId: 'dev-003', uptimePercent: 62.3, hddFreePercent: 88, cpuUsage: 0, memoryUsage: 0, temperature: 22 },
    { deviceId: 'dev-004', uptimePercent: 94.1, hddFreePercent: 53, cpuUsage: 45, memoryUsage: 67, temperature: 48 },
    { deviceId: 'dev-005', uptimePercent: 99.9, hddFreePercent: 67, cpuUsage: 28, memoryUsage: 44, temperature: 38 },
    { deviceId: 'dev-006', uptimePercent: 87.6, hddFreePercent: 31, cpuUsage: 55, memoryUsage: 71, temperature: 51 },
    { deviceId: 'dev-007', uptimePercent: 23.4, hddFreePercent: 91, cpuUsage: 0, memoryUsage: 0, temperature: 20 },
    { deviceId: 'dev-008', uptimePercent: 98.7, hddFreePercent: 58, cpuUsage: 31, memoryUsage: 52, temperature: 40 },
];

// Health Timeline Events
export const healthTimelineEvents: HealthTimelineEvent[] = [
    // dev-001
    { id: 'evt-001', deviceId: 'dev-001', timestamp: '2026-02-09T14:00:00Z', type: 'status_change', message: 'Device came online', severity: 'success' },
    { id: 'evt-002', deviceId: 'dev-001', timestamp: '2026-02-09T06:12:00Z', type: 'maintenance', message: 'Scheduled maintenance completed', severity: 'info' },
    { id: 'evt-003', deviceId: 'dev-001', timestamp: '2026-02-08T22:45:00Z', type: 'restart', message: 'Device restarted automatically', severity: 'warning' },
    // dev-002
    { id: 'evt-004', deviceId: 'dev-002', timestamp: '2026-02-09T14:05:00Z', type: 'alert', message: 'Storage capacity at 85% — consider archiving', severity: 'warning' },
    { id: 'evt-005', deviceId: 'dev-002', timestamp: '2026-02-09T08:00:00Z', type: 'status_change', message: 'Device came online', severity: 'success' },
    { id: 'evt-006', deviceId: 'dev-002', timestamp: '2026-02-07T14:00:00Z', type: 'alert', message: 'HDD health check: SMART warning detected', severity: 'error' },
    // dev-003
    { id: 'evt-007', deviceId: 'dev-003', timestamp: '2026-02-09T10:30:00Z', type: 'status_change', message: 'Device went offline — connection lost', severity: 'error' },
    { id: 'evt-008', deviceId: 'dev-003', timestamp: '2026-02-09T10:25:00Z', type: 'alert', message: 'High packet loss detected (42%)', severity: 'warning' },
    { id: 'evt-009', deviceId: 'dev-003', timestamp: '2026-02-08T16:00:00Z', type: 'firmware', message: 'Firmware update v10.12.114 installed', severity: 'info' },
    { id: 'evt-010', deviceId: 'dev-003', timestamp: '2026-02-07T09:00:00Z', type: 'status_change', message: 'Device came online', severity: 'success' },
    // dev-004
    { id: 'evt-011', deviceId: 'dev-004', timestamp: '2026-02-09T14:10:00Z', type: 'alert', message: 'Image quality degraded — sensor cleaning required', severity: 'warning' },
    { id: 'evt-012', deviceId: 'dev-004', timestamp: '2026-02-08T16:00:00Z', type: 'firmware', message: 'Firmware update available: v2.821.0', severity: 'info' },
    { id: 'evt-013', deviceId: 'dev-004', timestamp: '2026-02-07T11:30:00Z', type: 'restart', message: 'Manual restart performed by admin', severity: 'info' },
    // dev-005
    { id: 'evt-014', deviceId: 'dev-005', timestamp: '2026-02-09T14:15:00Z', type: 'status_change', message: 'All systems operational', severity: 'success' },
    { id: 'evt-015', deviceId: 'dev-005', timestamp: '2026-02-06T03:00:00Z', type: 'maintenance', message: 'Scheduled firmware update completed', severity: 'info' },
    // dev-006
    { id: 'evt-016', deviceId: 'dev-006', timestamp: '2026-02-09T14:00:00Z', type: 'alert', message: 'Recording gap detected (02:00–04:00 AM)', severity: 'error' },
    { id: 'evt-017', deviceId: 'dev-006', timestamp: '2026-02-09T08:00:00Z', type: 'status_change', message: 'Device status changed to Warning', severity: 'warning' },
    { id: 'evt-018', deviceId: 'dev-006', timestamp: '2026-02-08T20:00:00Z', type: 'restart', message: 'DVR rebooted due to high temperature', severity: 'error' },
    // dev-007
    { id: 'evt-019', deviceId: 'dev-007', timestamp: '2026-02-08T23:45:00Z', type: 'status_change', message: 'Device went offline — power failure suspected', severity: 'error' },
    { id: 'evt-020', deviceId: 'dev-007', timestamp: '2026-02-08T18:00:00Z', type: 'alert', message: 'Network timeout (5 min)', severity: 'warning' },
    { id: 'evt-021', deviceId: 'dev-007', timestamp: '2026-02-07T10:00:00Z', type: 'maintenance', message: 'Lens cleaning performed', severity: 'info' },
    // dev-008
    { id: 'evt-022', deviceId: 'dev-008', timestamp: '2026-02-09T14:12:00Z', type: 'status_change', message: 'Device online — all checks passed', severity: 'success' },
    { id: 'evt-023', deviceId: 'dev-008', timestamp: '2026-02-08T06:00:00Z', type: 'firmware', message: 'Firmware v7.82 installed successfully', severity: 'info' },
];

// Generate 1-year (365-day) recording calendar for a device's cameras
export function generateRecordingCalendar(deviceId: string): RecordingDay[] {
    // seedRandom is now defined globally for all generators
    const cameras = deviceCameras.filter(c => c.deviceId === deviceId);
    const result: RecordingDay[] = [];
    const today = new Date('2026-02-09');

    cameras.forEach((camera, camIdx) => {
        const numId = parseInt(camera.id.split('-')[1] || '0', 10);
        const rand = seedRandom(numId * 12345 + 1);
        for (let i = 364; i >= 0; i--) {
            const date = new Date(today);
            date.setDate(date.getDate() - i);
            const dateStr = date.toISOString().split('T')[0];

            let status: RecordingDay['status'];
            const r = rand();

            if (camera.status === 'offline') {
                // Offline cameras: more missing/no_data in recent days
                if (i < 7) status = 'no_data';
                else if (r < 0.5) status = 'available';
                else if (r < 0.85) status = 'missing';
                else status = 'no_data';
            } else if (camera.status === 'warning') {
                // Warning cameras: occasional gaps
                if (r < 0.75) status = 'available';
                else if (r < 0.92) status = 'missing';
                else status = 'no_data';
            } else {
                // Online cameras: mostly available
                if (r < 0.88) status = 'available';
                else if (r < 0.96) status = 'missing';
                else status = 'no_data';
            }

            result.push({ date: dateStr, cameraId: camera.id, cameraName: camera.name, status });
        }
    });

    return result;
}
