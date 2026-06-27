import { test, expect, type Page, type Route } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// ═══════════════════════════════════════════════════════════════════════════
// Accessibility (a11y) Tests — All Pages
// P1-QA.3: Automated a11y checks with @axe-core/playwright
// Compliance: OWASP ASVS L3 (V1-V5), Приказ ОАЦ №66 п.7.18
// Threshold: 0 critical violations per page (HARD FAIL в CI)
// ═══════════════════════════════════════════════════════════════════════════

// ───────────────────────────────────────────────────────────────────────────
// Types
// ───────────────────────────────────────────────────────────────────────────

interface MockUser {
  id: string;
  username: string;
  role: string;
  name: string;
  email: string;
  avatar: string;
  sites: string[];
}

// ───────────────────────────────────────────────────────────────────────────
// Mock data
// ───────────────────────────────────────────────────────────────────────────

const MOCK_USER: MockUser = {
  id: 'user-1',
  username: 'admin',
  name: 'Admin User',
  role: 'admin',
  email: 'admin@cctv.local',
  avatar: '',
  sites: ['site-1', 'site-2'],
};

const MOCK_DASHBOARD_STATS = {
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

const MOCK_DEVICES = [
  { device_id: 'dev-1', name: 'Camera-Lobby-01', location: 'Main Lobby', vendor_type: 'hikvision', status: 'online', last_seen: new Date().toISOString(), registered_at: new Date(Date.now() - 86400000 * 90).toISOString() },
  { device_id: 'dev-2', name: 'Camera-Parking-A', location: 'Parking Lot A', vendor_type: 'dahua', status: 'offline', last_seen: new Date(Date.now() - 7200000).toISOString(), registered_at: new Date(Date.now() - 86400000 * 180).toISOString() },
  { device_id: 'dev-3', name: 'NVR-01', location: 'Server Room', vendor_type: 'hikvision', status: 'online', last_seen: new Date().toISOString(), registered_at: new Date(Date.now() - 86400000 * 365).toISOString() },
];

const MOCK_SITES = [
  { id: 'site-1', name: 'Main Office', address: '123 Main St', city: 'Minsk', status: 'active', last_sync: new Date().toISOString(), created_at: new Date(Date.now() - 86400000 * 365).toISOString(), updated_at: new Date().toISOString() },
  { id: 'site-2', name: 'Branch Office', address: '456 Branch Ave', city: 'Brest', status: 'active', last_sync: new Date().toISOString(), created_at: new Date(Date.now() - 86400000 * 180).toISOString(), updated_at: new Date().toISOString() },
];

const MOCK_WORK_ORDERS = [
  { id: 'wo-1', title: 'Replace Camera Lens', site_id: 'site-1', work_type: 'maintenance', priority: 'high', status: 'in_progress', description: 'Camera-Lobby-01 lens damaged', scheduled_date: new Date(Date.now() + 86400000).toISOString(), assigned_to: 'tech-1', estimated_hours: 2, created_at: new Date().toISOString() },
  { id: 'wo-2', title: 'Firmware Update NVR-01', site_id: 'site-1', work_type: 'maintenance', priority: 'medium', status: 'open', description: 'Scheduled firmware update', scheduled_date: new Date(Date.now() + 86400000 * 3).toISOString(), assigned_to: 'tech-2', estimated_hours: 1, created_at: new Date().toISOString() },
];

const MOCK_REPORTS = [
  { id: 'rpt-1', name: 'Daily Report', type: 'daily', format: 'pdf', status: 'ready', generated_by: 'admin', generated_at: new Date(Date.now() - 86400000).toISOString(), expires_at: new Date(Date.now() + 86400000 * 7).toISOString() },
  { id: 'rpt-2', name: 'Weekly Summary', type: 'weekly', format: 'xlsx', status: 'ready', generated_by: 'admin', generated_at: new Date(Date.now() - 604800000).toISOString(), expires_at: new Date(Date.now() + 86400000 * 7).toISOString() },
];

const MOCK_SETTINGS = {
  services_syslog: { enabled: true, udp_port: 514, tcp_port: 514 },
  services_ftp: { enabled: true, port: 21, user: 'cctv', password: '', root_path: '/var/ftp' },
  services_snmp: { enabled: true, port: 161, community: 'public', version: 'v2c' },
  services_http: { enabled: true, port: 80 },
  services_dahua: { enabled: true, ports: [37777, 37778] },
  services_hisilicon: { enabled: true, port: 9000 },
  services_tvt: { enabled: true, port: 34567 },
};

const MOCK_P2P_DEVICES = [
  { id: 'p2p-1', name: 'Gate Controller A-101', mac: 'AA:BB:CC:DD:EE:01', status: 'online', ip_address: '10.0.1.10', firmware: '2.3.1', last_seen: new Date().toISOString(), registered_at: new Date(Date.now() - 86400000 * 30).toISOString() },
  { id: 'p2p-2', name: 'Access Panel B-204', mac: 'AA:BB:CC:DD:EE:02', status: 'offline', ip_address: '10.0.2.20', firmware: '2.1.0', last_seen: new Date(Date.now() - 7200000).toISOString(), registered_at: new Date(Date.now() - 86400000 * 60).toISOString() },
];

const MOCK_RCA_LIST = [
  { id: 'rca-1', title: 'Camera-12 Parking Lot B — Power Loss', device_id: 'dev-3', status: 'open', severity: 'critical', detected_at: new Date(Date.now() - 86400000).toISOString(), resolved_at: null },
  { id: 'rca-2', title: 'NVR-03 — Disk Failure', device_id: 'dev-2', status: 'resolved', severity: 'high', detected_at: new Date(Date.now() - 604800000).toISOString(), resolved_at: new Date(Date.now() - 432000000).toISOString() },
];

const MOCK_USERS = [
  { id: 'user-1', username: 'admin', role: 'admin', full_name: 'Admin User' },
  { id: 'user-2', username: 'tech1', role: 'technician', full_name: 'Bob Technician' },
];

const MOCK_ALERTS = [
  { id: 'alert-1', severity: 'critical', message: 'NVR-03 disk failure imminent', device_id: 'dev-2', status: 'active', created_at: new Date().toISOString() },
  { id: 'alert-2', severity: 'warning', message: 'Camera-12 offline > 24h', device_id: 'dev-3', status: 'active', created_at: new Date(Date.now() - 3600000).toISOString() },
];

// ───────────────────────────────────────────────────────────────────────────
// Helpers
// ───────────────────────────────────────────────────────────────────────────

/**
 * Устанавливает общие API моки для аутентификации.
 * Все protected pages требуют валидного /auth/me.
 */
async function setupAuthMock(page: Page) {
  await page.route('**/api/v1/auth/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USER),
    });
  });
  await page.route('**/api/v1/users/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USER),
    });
  });
  await page.evaluate(() => localStorage.setItem('token', 'mock-token-for-a11y'));
}

/**
 * AxeBuilder + analyze с жёсткой проверкой на critical violations.
 * P1-QA.3: FAIL в CI при любых critical violations (HARD FAIL).
 * Threshold: 0 critical violations per page.
 */
async function runAxeCheck(page: Page, pageName: string) {
  const builder = new AxeBuilder({ page });
  const results = await builder.analyze();

  const criticalViolations = results.violations.filter(
    (v) => v.impact === 'critical',
  );

  if (criticalViolations.length > 0) {
    console.error(`[a11y ❌] ${pageName} — ${criticalViolations.length} critical violation(s) found`);
    for (const violation of criticalViolations) {
      console.error(`  - ${violation.id}: ${violation.description}`);
      for (const node of violation.nodes) {
        console.error(`    → ${node.target.join(', ')}`);
      }
    }
  }

  expect(
    criticalViolations,
    `${pageName}: ${criticalViolations.length} critical a11y violation(s) found. Threshold: 0`,
  ).toHaveLength(0);

  console.log(`[a11y ✅] ${pageName} — 0 critical violations`);
}

/**
 * Универсальный catch-all для незамоканных API запросов.
 */
async function setupCatchAllMock(page: Page) {
  await page.route('**/api/v1/**', async (route: Route) => {
    const url = route.request().url();
    if (
      url.includes('/auth/me') ||
      url.includes('/users/me') ||
      url.includes('/dashboard/stats') ||
      url.includes('/devices') ||
      url.includes('/sites') ||
      url.includes('/work-orders') ||
      url.includes('/reports') ||
      url.includes('/settings') ||
      url.includes('/p2p') ||
      url.includes('/rca') ||
      url.includes('/alerts') ||
      url.includes('/users') ||
      url.includes('/analytics') ||
      url.includes('/logs') ||
      url.includes('/audit') ||
      url.includes('/profile') ||
      url.includes('/notifications') ||
      url.includes('/tickets') ||
      url.includes('/sla') ||
      url.includes('/maintenance') ||
      url.includes('/spare-parts') ||
      url.includes('/tutorials')
    ) {
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
 * Переход на страницу с ожиданием полной загрузки.
 */
async function goToPage(page: Page, path: string) {
  await page.goto(path);
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(500);
}

// ───────────────────────────────────────────────────────────────────────────
// Page configurations for a11y testing
// Каждая страница имеет свои необходимые API моки
// ───────────────────────────────────────────────────────────────────────────

interface PageTestConfig {
  path: string;
  name: string;
  setupMocks?: (page: Page) => Promise<void>;
}

const PUBLIC_PAGES: PageTestConfig[] = [
  { path: '/login', name: 'Login' },
  { path: '/forgot-password', name: 'Forgot Password' },
];

const PROTECTED_PAGES: PageTestConfig[] = [
  {
    path: '/dashboard',
    name: 'Dashboard',
    setupMocks: async (page) => {
      await page.route('**/api/v1/dashboard/stats', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_DASHBOARD_STATS) });
      });
      await page.route('**/api/v1/devices*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_DEVICES) });
      });
    },
  },
  {
    path: '/work-orders',
    name: 'Work Orders',
    setupMocks: async (page) => {
      await page.route('**/api/v1/work-orders*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_WORK_ORDERS) });
      });
      await page.route('**/api/v1/sites*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SITES) });
      });
    },
  },
  {
    path: '/devices',
    name: 'Devices',
    setupMocks: async (page) => {
      await page.route('**/api/v1/devices*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_DEVICES) });
      });
      await page.route('**/api/v1/sites*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SITES) });
      });
    },
  },
  {
    path: '/reports',
    name: 'Reports',
    setupMocks: async (page) => {
      await page.route('**/api/v1/reports*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_REPORTS) });
      });
      await page.route('**/api/v1/sites*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SITES) });
      });
    },
  },
  {
    path: '/settings',
    name: 'Settings',
    setupMocks: async (page) => {
      await page.route('**/api/v1/settings/services', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SETTINGS) });
      });
      await page.route('**/api/v1/settings/services/status', async (route: Route) => {
        await route.fulfill({
          status: 200, contentType: 'application/json', body: JSON.stringify({
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
    },
  },
  {
    path: '/p2p-devices',
    name: 'P2P Devices',
    setupMocks: async (page) => {
      await page.route('**/api/v1/p2p-devices*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_P2P_DEVICES) });
      });
      await page.route('**/api/v1/sites*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SITES) });
      });
    },
  },
  {
    path: '/rca',
    name: 'RCA Investigations',
    setupMocks: async (page) => {
      await page.route('**/api/v1/rca*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_RCA_LIST) });
      });
      await page.route('**/api/v1/devices*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_DEVICES) });
      });
      await page.route('**/api/v1/sites*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SITES) });
      });
    },
  },
  {
    path: '/alerts',
    name: 'Alerts',
    setupMocks: async (page) => {
      await page.route('**/api/v1/alerts*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_ALERTS) });
      });
    },
  },
  {
    path: '/notifications',
    name: 'Notifications',
    setupMocks: async (page) => {
      await page.route('**/api/v1/notifications*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
      });
    },
  },
  {
    path: '/profile',
    name: 'Profile',
    setupMocks: async (page) => {
      await page.route('**/api/v1/users/*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USER) });
      });
    },
  },
  {
    path: '/tickets',
    name: 'Tickets',
    setupMocks: async (page) => {
      await page.route('**/api/v1/tickets*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
      });
    },
  },
  {
    path: '/sites',
    name: 'Sites',
    setupMocks: async (page) => {
      await page.route('**/api/v1/sites*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_SITES) });
      });
    },
  },
  {
    path: '/users',
    name: 'Users Management',
    setupMocks: async (page) => {
      await page.route('**/api/v1/users*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(MOCK_USERS) });
      });
    },
  },
  {
    path: '/analytics',
    name: 'Analytics',
    setupMocks: async (page) => {
      await page.route('**/api/v1/analytics*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
      });
    },
  },
  {
    path: '/logs',
    name: 'Logs',
    setupMocks: async (page) => {
      await page.route('**/api/v1/logs*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
      });
    },
  },
  {
    path: '/audit-log',
    name: 'Audit Log',
    setupMocks: async (page) => {
      await page.route('**/api/v1/audit*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
      });
    },
  },
  {
    path: '/compliance-shield',
    name: 'Compliance Shield',
    setupMocks: async (page) => {
      await page.route('**/api/v1/compliance*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ status: 'compliant' }) });
      });
    },
  },
  {
    path: '/tutorials',
    name: 'Tutorials',
  },
  {
    path: '/maintenance',
    name: 'Maintenance Schedules',
    setupMocks: async (page) => {
      await page.route('**/api/v1/maintenance*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
      });
    },
  },
  {
    path: '/spare-parts',
    name: 'Spare Parts',
    setupMocks: async (page) => {
      await page.route('**/api/v1/spare-parts*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify([]) });
      });
    },
  },
  {
    path: '/sla',
    name: 'SLA Dashboard',
    setupMocks: async (page) => {
      await page.route('**/api/v1/sla*', async (route: Route) => {
        await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) });
      });
    },
  },
  {
    path: '/glossary',
    name: 'Glossary',
  },
];

// ───────────────────────────────────────────────────────────────────────────
// Test Suite: Accessibility — All Pages (critical violations only)
// Каждая страница тестируется изолированно с собственными моками.
// P1-QA.3: Threshold = 0 critical violations, HARD FAIL в CI.
// ───────────────────────────────────────────────────────────────────────────

test.describe('Accessibility — All Pages (critical violations only)', () => {
  // ─── Public Pages ─────────────────────────────────────────────────────

  test.describe('Public Pages', () => {
    for (const { path, name } of PUBLIC_PAGES) {
      test(`${name} (${path}) has no critical a11y violations`, async ({ page }) => {
        await goToPage(page, path);
        await runAxeCheck(page, path);
      });
    }
  });

  // ─── Protected Pages (admin role) ─────────────────────────────────────

  test.describe('Protected Pages (admin role)', () => {
    for (const { path, name, setupMocks } of PROTECTED_PAGES) {
      test(`${name} (${path}) has no critical a11y violations`, async ({ page }) => {
        await setupAuthMock(page);
        if (setupMocks) {
          await setupMocks(page);
        }
        await setupCatchAllMock(page);
        await goToPage(page, path);
        await runAxeCheck(page, path);
      });
    }
  });
});
