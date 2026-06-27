/// <reference types="node" />

// ═══════════════════════════════════════════════════════════════════════════
// Shared Mock API Module — E2E Tests
// P1-QA.1: Isolation via Mock API — переиспользуемые моки для всех E2E тестов
// ═══════════════════════════════════════════════════════════════════════════

import type { Page, Route } from '@playwright/test';

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

export interface MockUser {
  id: string;
  username: string;
  role: 'admin' | 'manager' | 'technician' | 'support' | 'owner';
  name?: string;
  email?: string;
  sites?: string[];
  avatar?: string;
}

export interface MockSite {
  id: string;
  name: string;
  address?: string;
  city?: string;
  status?: string;
  last_sync?: string;
  created_at?: string;
  updated_at?: string;
}

export interface MockDevice {
  id: string;
  name: string;
  ip_address: string;
  status: 'online' | 'offline' | 'warning';
  health: 'healthy' | 'degraded' | 'faulty';
  type: 'camera' | 'nvr' | 'switch' | 'gateway' | 'controller';
  site_id: string;
  model?: string;
  firmware?: string;
  last_seen: string;
  vendor_type?: string;
  location?: string;
  registered_at?: string;
}

export interface MockWorkOrder {
  id: string;
  title: string;
  status: 'open' | 'in_progress' | 'completed' | 'cancelled';
  priority: 'critical' | 'high' | 'medium' | 'low';
  assigned_to: string | null;
  site_id: string;
  site_name?: string;
  description?: string;
  work_type?: string;
  sla_deadline: string;
  created_at: string;
  created_by?: string;
  checklists?: MockChecklistItem[];
  photos?: string[];
  estimated_hours?: number;
  scheduled_date?: string;
}

export interface MockChecklistItem {
  id: string;
  label: string;
  required: boolean;
  completed: boolean;
}

export interface MockReport {
  id: string;
  title: string;
  type: 'daily' | 'weekly' | 'monthly' | 'incident';
  format: 'pdf' | 'xlsx' | 'csv';
  status: 'ready' | 'generating' | 'failed';
  created_at: string;
  url: string | null;
  generated_by?: string;
  expires_at?: string;
}

export interface MockP2PDevice {
  id: string;
  name: string;
  mac: string;
  status: 'online' | 'offline' | 'pending';
  ip_address: string;
  firmware: string;
  last_seen: string | null;
  registered_at: string;
}

export interface MockRCAInvestigation {
  id: string;
  title: string;
  device_id: string;
  status: 'open' | 'in_progress' | 'resolved';
  severity: 'critical' | 'high' | 'medium' | 'low';
  detected_at: string;
  resolved_at: string | null;
}

export interface MockDashboardStats {
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  warning_devices?: number;
  open_tickets?: number;
  critical_tickets?: number;
  resolution_rate?: number;
  avg_response_time_hours?: number;
  active_alerts?: number;
  open_work_orders?: number;
  overdue_sla?: number;
}

// ─────────────────────────────────────────────────────────────────────────────
// Default Mock Data
// ─────────────────────────────────────────────────────────────────────────────

export const MOCK_ADMIN_USER: MockUser = {
  id: 'user-1',
  username: 'admin',
  role: 'admin',
  name: 'Admin User',
  email: 'admin@cctv.local',
  sites: ['site-1', 'site-2'],
};

export const MOCK_MANAGER_USER: MockUser = {
  id: 'user-2',
  username: 'manager',
  role: 'manager',
  name: 'Alex Manager',
  email: 'manager@cctv.local',
  sites: ['site-1'],
};

export const MOCK_TECHNICIAN_USER: MockUser = {
  id: 'user-3',
  username: 'tech1',
  role: 'technician',
  name: 'Bob Technician',
  email: 'tech1@cctv.local',
  sites: ['site-1', 'site-2'],
};

export const MOCK_SITES: MockSite[] = [
  { id: 'site-1', name: 'Main Office', address: '123 Main St', city: 'Minsk', status: 'active', last_sync: new Date().toISOString(), created_at: new Date(Date.now() - 86400000 * 365).toISOString(), updated_at: new Date().toISOString() },
  { id: 'site-2', name: 'Branch Office', address: '456 Branch Ave', city: 'Brest', status: 'active', last_sync: new Date().toISOString(), created_at: new Date(Date.now() - 86400000 * 180).toISOString(), updated_at: new Date().toISOString() },
  { id: 'site-3', name: 'Warehouse', address: '789 Industrial Rd', city: 'Gomel', status: 'active', last_sync: new Date().toISOString(), created_at: new Date(Date.now() - 86400000 * 90).toISOString(), updated_at: new Date().toISOString() },
];

export const MOCK_DEVICES: MockDevice[] = [
  { id: 'dev-1', name: 'Camera-Lobby-01', ip_address: '192.168.1.100', status: 'online', health: 'healthy', type: 'camera', site_id: 'site-1', model: 'AXIS Q1615', firmware: '9.80.1', last_seen: new Date().toISOString(), vendor_type: 'hikvision', location: 'Main Lobby', registered_at: new Date(Date.now() - 86400000 * 90).toISOString() },
  { id: 'dev-2', name: 'NVR-03 Recording Server', ip_address: '192.168.1.50', status: 'online', health: 'degraded', type: 'nvr', site_id: 'site-1', model: 'HikVision DS-9608', firmware: '5.2.0', last_seen: new Date().toISOString(), vendor_type: 'hikvision', location: 'Server Room', registered_at: new Date(Date.now() - 86400000 * 365).toISOString() },
  { id: 'dev-3', name: 'Camera-12 Parking Lot B', ip_address: '192.168.2.20', status: 'offline', health: 'faulty', type: 'camera', site_id: 'site-2', model: 'Dahua IPC-HFW', firmware: '3.2.1', last_seen: new Date(Date.now() - 86400000).toISOString(), vendor_type: 'dahua', location: 'Parking Lot B', registered_at: new Date(Date.now() - 86400000 * 180).toISOString() },
  { id: 'dev-4', name: 'Switch-02 Floor B2', ip_address: '192.168.3.10', status: 'online', health: 'healthy', type: 'switch', site_id: 'site-2', model: 'Cisco Catalyst 2960', firmware: '15.2(7)', last_seen: new Date().toISOString() },
];

export const MOCK_WORK_ORDERS: MockWorkOrder[] = [
  { id: 'WO-001', title: 'Replace camera lens', status: 'open', priority: 'critical', assigned_to: null, site_id: 'site-1', description: 'Camera at main entrance has cracked lens', sla_deadline: new Date(Date.now() + 3600000).toISOString(), created_at: new Date().toISOString() },
  { id: 'WO-002', title: 'Firmware update NVR-03', status: 'in_progress', priority: 'high', assigned_to: 'user-2', site_id: 'site-1', sla_deadline: new Date(Date.now() - 3600000).toISOString(), created_at: new Date().toISOString() },
  { id: 'WO-003', title: 'Cable replacement floor B2', status: 'completed', priority: 'medium', assigned_to: 'user-2', site_id: 'site-2', sla_deadline: new Date(Date.now() + 86400000).toISOString(), created_at: new Date().toISOString() },
  { id: 'WO-004', title: 'Emergency camera repair', status: 'open', priority: 'critical', assigned_to: null, site_id: 'site-3', sla_deadline: new Date(Date.now() + 1800000).toISOString(), created_at: new Date().toISOString() },
];

export const MOCK_USERS = [
  { id: 'user-1', username: 'admin', role: 'admin', full_name: 'Admin User' },
  { id: 'user-2', username: 'manager', role: 'manager', full_name: 'Alex Manager' },
  { id: 'user-3', username: 'tech1', role: 'technician', full_name: 'Bob Technician' },
  { id: 'user-4', username: 'tech2', role: 'technician', full_name: 'Carol Engineer' },
];

export const MOCK_REPORTS: MockReport[] = [
  { id: 'rpt-1', title: 'Ежедневный отчёт — 2026-06-25', type: 'daily', format: 'pdf', status: 'ready', created_at: new Date(Date.now() - 86400000).toISOString(), url: '/api/v1/reports/rpt-1/download' },
  { id: 'rpt-2', title: 'Еженедельный отчёт — W26', type: 'weekly', format: 'xlsx', status: 'ready', created_at: new Date(Date.now() - 604800000).toISOString(), url: '/api/v1/reports/rpt-2/download' },
  { id: 'rpt-3', title: 'Аварийный отчёт — 2026-06-24', type: 'incident', format: 'pdf', status: 'generating', created_at: new Date(Date.now() - 172800000).toISOString(), url: null },
];

export const MOCK_P2P_DEVICES: MockP2PDevice[] = [
  { id: 'p2p-1', name: 'Gate Controller A-101', mac: 'AA:BB:CC:DD:EE:01', status: 'online', ip_address: '10.0.1.10', firmware: '2.3.1', last_seen: new Date().toISOString(), registered_at: new Date(Date.now() - 86400000 * 30).toISOString() },
  { id: 'p2p-2', name: 'Access Panel B-204', mac: 'AA:BB:CC:DD:EE:02', status: 'offline', ip_address: '10.0.2.20', firmware: '2.1.0', last_seen: new Date(Date.now() - 7200000).toISOString(), registered_at: new Date(Date.now() - 86400000 * 60).toISOString() },
];

export const MOCK_RCA_LIST: MockRCAInvestigation[] = [
  { id: 'rca-1', title: 'Camera-12 Parking Lot B — Power Loss', device_id: 'dev-3', status: 'open', severity: 'critical', detected_at: new Date(Date.now() - 86400000).toISOString(), resolved_at: null },
  { id: 'rca-2', title: 'NVR-03 — Disk Failure', device_id: 'dev-2', status: 'resolved', severity: 'high', detected_at: new Date(Date.now() - 604800000).toISOString(), resolved_at: new Date(Date.now() - 432000000).toISOString() },
];

export const MOCK_DASHBOARD_STATS: MockDashboardStats = {
  total_devices: 42,
  online_devices: 38,
  offline_devices: 3,
  warning_devices: 1,
  open_tickets: 7,
  critical_tickets: 2,
  resolution_rate: 94.5,
  avg_response_time_hours: 1.8,
  active_alerts: 5,
  open_work_orders: 12,
  overdue_sla: 2,
};

export const MOCK_SETTINGS = {
  site_name: 'Test Site',
  language: 'en',
  timezone: 'UTC',
  date_format: 'YYYY-MM-DD',
  security: {
    password_policy: 'basic',
    session_timeout: 30,
    max_login_attempts: 5,
  },
  notifications: {
    email_alerts: true,
    sms_alerts: false,
    push_alerts: true,
  },
  system: {
    log_retention_days: 30,
    max_devices: 100,
  },
};

// ─────────────────────────────────────────────────────────────────────────────
// Mock Setup Functions
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Устанавливает мок для /auth/me — основной эндпоинт аутентификации.
 * Все protected pages требуют этого мока.
 */
export async function mockAuthMe(page: Page, user: MockUser = MOCK_ADMIN_USER): Promise<void> {
  await page.route('**/api/v1/auth/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(user),
    });
  });

  await page.route('**/api/v1/users/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(user),
    });
  });
}

/**
 * Устанавливает токен в localStorage для имитации аутентификации.
 * Должен вызываться ПОСЛЕ page.goto (на странице login или после navigate).
 */
export async function setToken(page: Page, token: string = 'mock-token-e2e'): Promise<void> {
  await page.evaluate((t: string) => localStorage.setItem('token', t), token);
}

/**
 * Быстрая настройка аутентификации для protected pages.
 * Комбинирует mockAuthMe + setToken.
 */
export async function setupAuth(page: Page, user: MockUser = MOCK_ADMIN_USER): Promise<void> {
  await mockAuthMe(page, user);
  await page.evaluate((userData: MockUser) => {
    localStorage.setItem('token', 'mock-token-e2e');
    localStorage.setItem('user', JSON.stringify(userData));
  }, user);
}

/**
 * Мокает эндпоинт /sites
 */
export async function mockSites(page: Page, sites: MockSite[] = MOCK_SITES): Promise<void> {
  await page.route('**/api/v1/sites*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(sites),
    });
  });
}

/**
 * Мокает эндпоинт /devices
 */
export async function mockDevices(page: Page, devices: MockDevice[] = MOCK_DEVICES): Promise<void> {
  await page.route('**/api/v1/devices*', async (route: Route, request) => {
    const url = request.url();
    // Single device detail
    const deviceMatch = url.match(/\/devices\/([^/?]+)/);
    if (deviceMatch && request.method() === 'GET') {
      const deviceId = deviceMatch[1];
      // Skip if it's the collection endpoint
      if (deviceId && !['devices'].includes(deviceId)) {
        const device = devices.find((d) => d.id === deviceId);
        if (device) {
          return route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(device),
          });
        }
      }
    }
    // Collection endpoint
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(devices),
    });
  });
}

/**
 * Мокает эндпоинт /users
 */
export async function mockUsers(page: Page, users: typeof MOCK_USERS = MOCK_USERS): Promise<void> {
  await page.route('**/api/v1/users*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(users),
    });
  });
}

/**
 * Мокает эндпоинт /work-orders
 * Поддерживает GET (list) и POST (create).
 */
export async function mockWorkOrders(
  page: Page,
  orders: MockWorkOrder[] = MOCK_WORK_ORDERS,
): Promise<void> {
  await page.route('**/api/v1/work-orders*', async (route: Route, request) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      const newOrder: MockWorkOrder = {
        id: `WO-NEW-${Date.now()}`,
        ...body,
        status: 'open',
        created_at: new Date().toISOString(),
        sla_deadline: body.sla_deadline || new Date(Date.now() + 86400000).toISOString(),
      };
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(newOrder),
      });
    }

    // GET — check for individual detail
    const url = request.url();
    const detailMatch = url.match(/\/work-orders\/([^/?]+)/);
    if (detailMatch && detailMatch[1] && !detailMatch[1].includes('?')) {
      const woId = detailMatch[1];
      const order = orders.find((o) => o.id === woId);
      if (order) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(order),
        });
      }
    }

    // GET — list
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(orders),
    });
  });

  // Mock assign endpoint
  await page.route('**/api/v1/work-orders/*/assign', async (route: Route, request) => {
    const body = JSON.parse(request.postData() || '{}');
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        assigned_to: body.technician_id || body.user_id,
        assigned_at: new Date().toISOString(),
      }),
    });
  });

  // Mock complete endpoint
  await page.route('**/api/v1/work-orders/*/complete', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'completed',
        completed_at: new Date().toISOString(),
      }),
    });
  });
}

/**
 * Мокает эндпоинт /dashboard/stats
 */
export async function mockDashboardStats(
  page: Page,
  stats: MockDashboardStats = MOCK_DASHBOARD_STATS,
): Promise<void> {
  await page.route('**/api/v1/dashboard/stats', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(stats),
    });
  });
}

/**
 * Мокает эндпоинт /alerts
 */
export async function mockAlerts(page: Page): Promise<void> {
  await page.route('**/api/v1/alerts*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'alert-1', severity: 'critical', message: 'NVR-03 disk failure imminent', device_id: 'dev-2', status: 'active', created_at: new Date().toISOString() },
        { id: 'alert-2', severity: 'warning', message: 'Camera-12 offline > 24h', device_id: 'dev-3', status: 'active', created_at: new Date(Date.now() - 3600000).toISOString() },
        { id: 'alert-3', severity: 'info', message: 'Scheduled maintenance due', device_id: 'dev-1', status: 'acknowledged', created_at: new Date(Date.now() - 7200000).toISOString() },
      ]),
    });
  });
}

/**
 * Мокает эндпоинт /reports
 * Поддерживает GET (list), POST (generate) и download.
 */
export async function mockReports(
  page: Page,
  reports: MockReport[] = MOCK_REPORTS,
): Promise<void> {
  await page.route('**/api/v1/reports*', async (route: Route, request) => {
    if (request.method() === 'POST') {
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: `rpt-new-${Date.now()}`,
          title: 'Сгенерированный отчёт',
          status: 'generating',
          created_at: new Date().toISOString(),
          url: null,
        }),
      });
    }

    const url = request.url();
    if (url.includes('/download')) {
      return route.fulfill({
        status: 200,
        contentType: 'application/octet-stream',
        headers: { 'Content-Disposition': 'attachment; filename="report.pdf"' },
        body: Buffer.from('%PDF-1.4 mock pdf content'),
      });
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(reports),
    });
  });
}

/**
 * Мокает эндпоинт /p2p-devices
 * Поддерживает GET (list) и POST (register).
 * Для POST поддерживает ответ 409 (duplicate MAC).
 */
export async function mockP2PDevices(
  page: Page,
  devices: MockP2PDevice[] = MOCK_P2P_DEVICES,
): Promise<void> {
  await page.route('**/api/v1/p2p-devices*', async (route: Route, request) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      // Check for duplicate MAC
      const existingMac = devices.find((d) => d.mac === body.mac);
      if (existingMac) {
        return route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({
            code: 'DUPLICATE_MAC',
            message: 'Устройство с таким MAC-адресом уже зарегистрировано',
          }),
        });
      }
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: `p2p-new-${Date.now()}`,
          ...body,
          status: 'pending',
          last_seen: null,
          registered_at: new Date().toISOString(),
          message: 'Устройство успешно зарегистрировано',
        }),
      });
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(devices),
    });
  });
}

/**
 * Мокает эндпоинт /rca (Root Cause Analysis)
 */
export async function mockRCA(
  page: Page,
  investigations: MockRCAInvestigation[] = MOCK_RCA_LIST,
): Promise<void> {
  await page.route('**/api/v1/rca*', async (route: Route, request) => {
    const url = request.url();

    // Single RCA investigation with graph
    const detailMatch = url.match(/\/rca\/([^/?]+)/);
    if (detailMatch && detailMatch[1]) {
      const rcaId = detailMatch[1];
      const investigation = investigations.find((i) => i.id === rcaId);

      if (url.includes('/graph')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            root_cause: {
              id: 'dev-3',
              name: 'Camera-12 Parking Lot B',
              type: 'camera',
              status: 'offline',
              health: 'faulty',
              failure_type: 'power_loss',
              detected_at: new Date(Date.now() - 86400000).toISOString(),
            },
            affected_devices: [
              { id: 'dev-4', name: 'NVR-03 Recording Server', type: 'nvr', impact: 'degraded', relation: 'upstream' },
              { id: 'dev-5', name: 'Switch-02 Floor B2', type: 'switch', impact: 'affected', relation: 'network_parent' },
            ],
            timeline: [
              { event: 'power_loss', timestamp: new Date(Date.now() - 86400000).toISOString(), severity: 'critical' },
              { event: 'connection_lost', timestamp: new Date(Date.now() - 86400000 + 60000).toISOString(), severity: 'critical' },
              { event: 'alert_triggered', timestamp: new Date(Date.now() - 86400000 + 120000).toISOString(), severity: 'high' },
            ],
            recommendations: [
              'Проверить питание на панели P-12 (Floor B2)',
              'Заменить блок питания камеры Camera-12',
              'Проверить состояние UPS в серверной B2',
            ],
          }),
        });
      }

      if (investigation) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(investigation),
        });
      }
    }

    // List all investigations
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(investigations),
    });
  });
}

/**
 * Мокает эндпоинт /upload (фото/файлы)
 */
export async function mockUpload(page: Page): Promise<void> {
  await page.route('**/api/v1/upload*', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        url: 'https://storage.example.com/uploads/mock-file.jpg',
        filename: 'mock-file.jpg',
        size: 1024,
        mime: 'image/jpeg',
      }),
    });
  });
}

/**
 * Мокает эндпоинт /settings
 */
export async function mockSettings(page: Page): Promise<void> {
  await page.route('**/api/v1/settings', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SETTINGS),
    });
  });

  await page.route('**/api/v1/settings/services', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        services_syslog: { enabled: true, udp_port: 514, tcp_port: 514 },
        services_ftp: { enabled: true, port: 21 },
        services_snmp: { enabled: true, port: 161, community: 'public', version: 'v2c' },
        services_http: { enabled: true, port: 80 },
        services_dahua: { enabled: true, ports: [37777, 37778] },
        services_hisilicon: { enabled: true, port: 9000 },
        services_tvt: { enabled: true, port: 34567 },
      }),
    });
  });

  await page.route('**/api/v1/settings/services/status', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        services: {
          syslog: { status: 'running', port: 514 },
          ftp: { status: 'running', port: 21 },
          snmp: { status: 'running', port: 161 },
          http: { status: 'running', port: 80 },
          p2p_gateway: { status: 'running', port: 8082 },
        },
      }),
    });
  });
}

/**
 * Catch-all для незамоканных API запросов.
 * Предотвращает 404 ошибки при рендеринге страниц.
 */
export async function mockCatchAll(page: Page): Promise<void> {
  await page.route('**/api/v1/**', async (route: Route) => {
    const url = route.request().url();
    const skipPaths = [
      '/auth/me', '/users/me', '/dashboard/stats',
      '/devices', '/sites', '/work-orders',
      '/reports', '/settings', '/p2p',
      '/rca', '/upload', '/alerts', '/users',
    ];
    const shouldSkip = skipPaths.some((p) => url.includes(p));
    if (shouldSkip) {
      return route.fallback();
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({}),
    });
  });
}

/**
 * Полная настройка всех базовых моков для protected pages.
 * Вызывается в beforeEach для тестов, которым нужен полный набор данных.
 */
export async function setupAllMocks(page: Page, user: MockUser = MOCK_ADMIN_USER): Promise<void> {
  await setupAuth(page, user);
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockDashboardStats(page);
  await mockAlerts(page);
  await mockReports(page);
  await mockP2PDevices(page);
  await mockRCA(page);
  await mockUpload(page);
  await mockSettings(page);
  await mockCatchAll(page);
}

/**
 * Переход на страницу с ожиданием загрузки.
 * Используется как унифицированный navigate для E2E тестов.
 */
export async function navigateAndWait(
  page: Page,
  url: string,
  waitMs: number = 1500,
): Promise<void> {
  await page.goto(url);
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(waitMs);
}
