import { test, expect, type Page, type Route } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════════
// Visual Regression Tests — CCTV Health Monitor
// P0 flow: visual regression snapshots for critical pages and components
// ═══════════════════════════════════════════════════════════════════════════

// ── Mock Data ───────────────────────────────────────────────────────────────

const MOCK_USER = {
  id: 'user-1',
  username: 'admin',
  role: 'admin' as const,
  name: 'Admin User',
  email: 'admin@cctv.local',
  sites: ['site-1', 'site-2'],
};

const MOCK_SITES = [
  { id: 'site-1', name: 'Main Office' },
  { id: 'site-2', name: 'Branch Office' },
];

const MOCK_DEVICES = [
  {
    id: 'dev-1',
    name: 'Camera-01 Main Entrance',
    ip_address: '192.168.1.100',
    status: 'online',
    health: 'healthy',
    type: 'camera',
    site_id: 'site-1',
    model: 'AXIS Q1615',
    firmware: '9.80.1',
    last_seen: new Date().toISOString(),
  },
  {
    id: 'dev-2',
    name: 'NVR-03 Recording Server',
    ip_address: '192.168.1.50',
    status: 'online',
    health: 'degraded',
    type: 'nvr',
    site_id: 'site-1',
    model: 'HikVision DS-9608',
    firmware: '5.2.0',
    last_seen: new Date().toISOString(),
  },
  {
    id: 'dev-3',
    name: 'Camera-12 Parking Lot B',
    ip_address: '192.168.2.20',
    status: 'offline',
    health: 'faulty',
    type: 'camera',
    site_id: 'site-2',
    model: 'Dahua IPC-HFW',
    firmware: '3.2.1',
    last_seen: new Date(Date.now() - 86_400_000).toISOString(),
  },
];

const MOCK_WORK_ORDERS = [
  {
    id: 'WO-001',
    title: 'Replace camera lens',
    status: 'open',
    priority: 'critical',
    assigned_to: 'user-1',
    site_id: 'site-1',
    sla_deadline: new Date(Date.now() + 3_600_000).toISOString(),
    created_at: new Date().toISOString(),
  },
  {
    id: 'WO-002',
    title: 'Firmware update NVR-03',
    status: 'in_progress',
    priority: 'high',
    assigned_to: null,
    site_id: 'site-2',
    sla_deadline: new Date(Date.now() - 3_600_000).toISOString(),
    created_at: new Date().toISOString(),
  },
  {
    id: 'WO-003',
    title: 'Cable replacement floor B2',
    status: 'completed',
    priority: 'medium',
    assigned_to: 'user-2',
    site_id: 'site-1',
    sla_deadline: new Date(Date.now() + 86_400_000).toISOString(),
    created_at: new Date().toISOString(),
  },
];

const MOCK_USERS = [
  { id: 'user-1', username: 'admin', role: 'admin' },
  { id: 'user-2', username: 'tech1', role: 'technician' },
];

const MOCK_DASHBOARD_STATS = {
  total_devices: 42,
  online_devices: 38,
  offline_devices: 4,
  active_alerts: 7,
  open_work_orders: 12,
  overdue_sla: 2,
};

// ── Shared Helpers ──────────────────────────────────────────────────────────

/**
 * Mock authentication endpoint so every protected page thinks we're logged in.
 * Must be called in each test's beforeEach or at the top of the page setup.
 */
async function setupAuth(page: Page): Promise<void> {
  await page.route('**/api/v1/auth/me', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_USER),
    });
  });

  await page.route('**/api/v1/auth/session', async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ valid: true }),
    });
  });
}

/**
 * Generic data-router: catches any unmatched /api/ call and returns an empty
 * array to prevent 404s from breaking page rendering.
 */
async function setupFallbackRoutes(page: Page): Promise<void> {
  await page.route('**/api/v1/**', async (route: Route) => {
    const url = route.request().url();
    // Only fulfil GET requests we haven't explicitly mocked
    if (route.request().method() === 'GET') {
      // Let already-fulfilled routes pass through
      const existing = await page.context().route;
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    } else {
      await route.continue();
    }
  });
}

/**
 * Navigate and stabilise the page before taking a screenshot.
 * Always use `networkidle` + a small cooldown so async renders finish.
 */
async function navigateAndStabilise(
  page: Page,
  url: string,
  stabiliseMs = 500,
): Promise<void> {
  await page.goto(url);
  await page.waitForLoadState('networkidle');
  await page.waitForTimeout(stabiliseMs);
}

// ── Screenshot Options ──────────────────────────────────────────────────────

const FULL_PAGE_OPTS = {
  fullPage: true as const,
  animations: 'disabled' as const,
  caret: 'hide' as const,
};

const COMPONENT_OPTS = {
  fullPage: false as const,
  animations: 'disabled' as const,
  caret: 'hide' as const,
};

const DIFF_OPTS = {
  maxDiffPixels: 100,
  maxDiffPixelRatio: 0.01,
};

// ═══════════════════════════════════════════════════════════════════════════
// Login Page
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Login page — visual regression', () => {
  test('full page screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/login');

    await expect(page).toHaveScreenshot('login-full-page.png', {
      ...FULL_PAGE_OPTS,
      ...DIFF_OPTS,
    });
  });

  test('login form with validation error', async ({ page }) => {
    await navigateAndStabilise(page, '/login');

    // Trigger HTML5 validation by clicking submit with empty fields
    await page.getByRole('button', { name: /sign in|login|войти/i }).click();
    await page.waitForTimeout(300);

    await expect(page).toHaveScreenshot('login-validation-error.png', {
      ...FULL_PAGE_OPTS,
      ...DIFF_OPTS,
    });
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Dashboard Page
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Dashboard page — visual regression', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    // Mock dashboard data
    await page.route('**/api/v1/dashboard/stats', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_DASHBOARD_STATS),
      });
    });

    await page.route('**/api/v1/devices*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_DEVICES),
      });
    });

    await page.route('**/api/v1/work-orders*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_WORK_ORDERS),
      });
    });

    await page.route('**/api/v1/sites*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SITES),
      });
    });

    await page.route('**/api/v1/alerts*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 'alert-1', severity: 'critical', message: 'NVR-03 disk failure imminent', device_id: 'dev-2' },
          { id: 'alert-2', severity: 'warning', message: 'Camera-12 offline > 24h', device_id: 'dev-3' },
        ]),
      });
    });
  });

  test('full page screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/dashboard', 1_000);

    await expect(page).toHaveScreenshot('dashboard-full-page.png', {
      ...FULL_PAGE_OPTS,
      ...DIFF_OPTS,
    });
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Work Orders Page
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Work Orders page — visual regression', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    await page.route('**/api/v1/work-orders*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_WORK_ORDERS),
      });
    });

    await page.route('**/api/v1/sites*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SITES),
      });
    });

    await page.route('**/api/v1/users*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_USERS),
      });
    });
  });

  test('full page screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/work-orders', 1_000);

    await expect(page).toHaveScreenshot('work-orders-full-page.png', {
      ...FULL_PAGE_OPTS,
      ...DIFF_OPTS,
    });
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Devices Page
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Devices page — visual regression', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    await page.route('**/api/v1/devices*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_DEVICES),
      });
    });

    await page.route('**/api/v1/sites*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SITES),
      });
    });
  });

  test('full page screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/devices', 1_000);

    await expect(page).toHaveScreenshot('devices-full-page.png', {
      ...FULL_PAGE_OPTS,
      ...DIFF_OPTS,
    });
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Component-Level Screenshots
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Component visual regression', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    await page.route('**/api/v1/devices*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_DEVICES),
      });
    });

    await page.route('**/api/v1/sites*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SITES),
      });
    });
  });

  // ── Modal Component ───────────────────────────────────────────────────

  test('modal component screenshot', async ({ page }) => {
    // Navigate to devices page where "Add Device" button triggers a modal
    await navigateAndStabilise(page, '/devices', 500);

    // Click the "Add Device" button to open the modal
    const addButton = page.locator('button', { hasText: /add device|add|create/i }).first();
    if (await addButton.isVisible()) {
      await addButton.click();
      await page.waitForTimeout(500); // Wait for modal transition
    }

    await expect(page).toHaveScreenshot('component-modal.png', {
      ...COMPONENT_OPTS,
      ...DIFF_OPTS,
    });
  });

  // ── Table Component ───────────────────────────────────────────────────

  test('table component screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/devices', 1_000);

    // Locate the data table region
    const table = page.locator('table, [role="grid"], [data-testid="data-table"]').first();
    await expect(table).toBeVisible({ timeout: 5_000 });

    await expect(table).toHaveScreenshot('component-table.png', {
      ...COMPONENT_OPTS,
      ...DIFF_OPTS,
    });
  });

  // ── Device Card Component ─────────────────────────────────────────────

  test('device card component screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/devices', 1_000);

    // Capture the first device card / row in the list
    const deviceCard = page
      .locator('[data-testid="device-card"], [class*="card"], tr[data-device-id]')
      .first();

    await expect(deviceCard).toBeVisible({ timeout: 5_000 });

    await expect(deviceCard).toHaveScreenshot('component-device-card.png', {
      ...COMPONENT_OPTS,
      ...DIFF_OPTS,
    });
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Dark Mode Screenshots
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Dark mode visual regression', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);

    // Mock all dashboard data
    await page.route('**/api/v1/dashboard/stats', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_DASHBOARD_STATS),
      });
    });

    await page.route('**/api/v1/devices*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_DEVICES),
      });
    });

    await page.route('**/api/v1/work-orders*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_WORK_ORDERS),
      });
    });

    await page.route('**/api/v1/sites*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_SITES),
      });
    });

    await page.route('**/api/v1/alerts*', async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { id: 'alert-1', severity: 'critical', message: 'NVR-03 disk failure imminent', device_id: 'dev-2' },
        ]),
      });
    });

    // Enable dark mode via localStorage (adjust key to match app's theme store)
    await page.addInitScript(() => {
      localStorage.setItem('theme', JSON.stringify({ mode: 'dark' }));
    });
  });

  test('dashboard dark mode screenshot', async ({ page }) => {
    await navigateAndStabilise(page, '/dashboard', 1_000);

    await expect(page).toHaveScreenshot('dark-dashboard-full-page.png', {
      ...FULL_PAGE_OPTS,
      ...DIFF_OPTS,
    });
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Graceful Degradation — Skip if no baseline exists
// ═══════════════════════════════════════════════════════════════════════════

/**
 * Playwright's `toHaveScreenshot` automatically creates a baseline on first run.
 * If the snapshot directory is empty or the snapshot file does not exist,
 * the test will fail with "Screenshot comparison failed: A snapshot doesn't exist".
 *
 * To handle this gracefully, we check for the baseline and skip if missing.
 * Uncomment the helper below and use `testOrSkip` in place of `test` to enable.
 *
 * NOTE: On CI, always run once with `--update-snapshots` to create baselines.
 */

/*
import { existsSync } from 'fs';
import { resolve } from 'path';

function getSnapshotPath(page: Page, name: string): string {
  const config = page.context()._config || {};
  const snapshotDir = config.snapshotDir || '__snapshots__';
  const projectName = config.project?.name || 'chromium';
  return resolve(snapshotDir, projectName, name);
}

function testOrSkip(
  description: string,
  snapshotName: string,
  fn: (page: Page) => Promise<void>,
) {
  const snapshotPath = getSnapshotPath(...);
  if (!existsSync(snapshotPath)) {
    test.skip(description, () => {
      console.warn(`[visual-regression] Baseline missing: ${snapshotPath}. Run with --update-snapshots to create.`);
    });
    return;
  }
  test(description, fn);
}
*/
