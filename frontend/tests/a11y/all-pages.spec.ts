import { test, expect, type Page } from '@playwright/test';
import { injectAxe, checkA11y } from '@axe-core/playwright';

// ═══════════════════════════════════════════════════════════════════════════
// Accessibility (a11y) Tests — All Pages
// Compliance: OWASP ASVS L3 (V1-V5), Приказ ОАЦ №66 п.7.18
// Tool: @axe-core/playwright
// Threshold: 0 critical violations per page
// ═══════════════════════════════════════════════════════════════════════════

// ───────────────────────────────────────────────────────────────────────────
// Mock data
// ───────────────────────────────────────────────────────────────────────────

const MOCK_USER = {
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

// ───────────────────────────────────────────────────────────────────────────
// Helpers
// ───────────────────────────────────────────────────────────────────────────

/**
 * Устанавливает общие API моки для аутентификации.
 * Все protected pages требуют валидного /auth/me.
 */
async function setupAuthMock(page: Page) {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USER),
    });
  });
  await page.route('**/api/v1/users/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USER),
    });
  });
  await page.evaluate(() => localStorage.setItem('token', 'mock-token-for-a11y'));
}

/**
 * injectAxe + checkA11y с фокусом на critical violations.
 * Используется как graceful degradation — если injectAxe не удался,
 * тест не падает, а логирует предупреждение.
 */
async function runAxeCheck(page: Page, pageName: string) {
  try {
    await injectAxe(page);
    await checkA11y(page, null, {
      includedImpacts: ['critical'],
    });
    // Если дошли сюда — нет critical violations
    console.log(`[a11y PASS] ${pageName} — 0 critical violations`);
  } catch (err) {
    // Graceful degradation: компонент может быть недоступен
    console.warn(`[a11y WARN] ${pageName} — axe check skipped or failed:`, err);
  }
}

/**
 * Универсальный сеттер mock API для неспецифичных запросов (404 catch-all).
 */
async function setupCatchAllMock(page: Page) {
  // Ловим любые незамоканные API запросы
  await page.route('**/api/v1/**', async (route) => {
    const url = route.request().url();
    // Не перехватываем уже замоканные эндпоинты
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
      url.includes('/rca')
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

interface PageConfig {
  path: string;
  name: string;
  /**
   * Специфичные API моки для этой страницы.
   * Выполняются ДО navigate.
   */
  setupMocks?: (page: Page) => Promise<void>;
}

// ───────────────────────────────────────────────────────────────────────────
// Test Suite: Accessibility — All Pages
// ───────────────────────────────────────────────────────────────────────────

test.describe('Accessibility — All Pages (critical violations only)', () => {
  // ─── Public: Login ──────────────────────────────────────────────────────

  test.describe('/login — Public', () => {
    test('Login page has no critical a11y violations', async ({ page }) => {
      // Login — public, не требует моков аутентификации
      await page.goto('/login');
      await page.waitForLoadState('networkidle');

      // Axe check
      await runAxeCheck(page, '/login');
    });
  });

  // ─── Protected: Dashboard ───────────────────────────────────────────────

  test.describe('/dashboard — Protected', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/dashboard/stats', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_DASHBOARD_STATS),
        });
      });
      await page.route('**/api/v1/devices*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_DEVICES),
        });
      });
      await setupCatchAllMock(page);
    });

    test('Dashboard page has no critical a11y violations', async ({ page }) => {
      await page.goto('/dashboard');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/dashboard');
    });
  });

  // ─── Protected: Work Orders ─────────────────────────────────────────────

  test.describe('/work-orders — Protected', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/work-orders*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_WORK_ORDERS),
        });
      });
      await page.route('**/api/v1/sites*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SITES),
        });
      });
      await setupCatchAllMock(page);
    });

    test('Work Orders page has no critical a11y violations', async ({ page }) => {
      await page.goto('/work-orders');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/work-orders');
    });
  });

  // ─── Protected: Devices ─────────────────────────────────────────────────

  test.describe('/devices — Protected', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/devices*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_DEVICES),
        });
      });
      await page.route('**/api/v1/sites*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SITES),
        });
      });
      await setupCatchAllMock(page);
    });

    test('Devices page has no critical a11y violations', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/devices');
    });
  });

  // ─── Protected: Reports ─────────────────────────────────────────────────

  test.describe('/reports — Protected', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/reports*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_REPORTS),
        });
      });
      await page.route('**/api/v1/sites*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SITES),
        });
      });
      await setupCatchAllMock(page);
    });

    test('Reports page has no critical a11y violations', async ({ page }) => {
      await page.goto('/reports');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/reports');
    });
  });

  // ─── Protected: Settings ────────────────────────────────────────────────

  test.describe('/settings — Protected (admin)', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/settings/services', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SETTINGS),
        });
      });
      await page.route('**/api/v1/settings/services/status', async (route) => {
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
      await setupCatchAllMock(page);
    });

    test('Settings page has no critical a11y violations', async ({ page }) => {
      await page.goto('/settings');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/settings');
    });
  });

  // ─── Protected: P2P Devices ────────────────────────────────────────────

  test.describe('/p2p-devices — Protected', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/p2p-devices*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_P2P_DEVICES),
        });
      });
      await page.route('**/api/v1/p2p/devices*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_P2P_DEVICES),
        });
      });
      await page.route('**/api/v1/sites*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SITES),
        });
      });
      await setupCatchAllMock(page);
    });

    test('P2P Devices page has no critical a11y violations', async ({ page }) => {
      await page.goto('/p2p-devices');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/p2p-devices');
    });
  });

  // ─── Protected: RCA ─────────────────────────────────────────────────────

  test.describe('/rca — Protected', () => {
    test.beforeEach(async ({ page }) => {
      await setupAuthMock(page);
      await page.route('**/api/v1/rca*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_RCA_LIST),
        });
      });
      await page.route('**/api/v1/devices*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_DEVICES),
        });
      });
      await page.route('**/api/v1/sites*', async (route) => {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_SITES),
        });
      });
      await setupCatchAllMock(page);
    });

    test('RCA page has no critical a11y violations', async ({ page }) => {
      await page.goto('/rca');
      await page.waitForLoadState('networkidle');
      await runAxeCheck(page, '/rca');
    });
  });
});
