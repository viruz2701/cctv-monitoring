// ═══════════════════════════════════════════════════════════════════════════
// Visual Regression Testing — CCTV Health Monitor
// P1-QA.6: Percy Integration + Baseline Screenshots
//
// Использует @percy/playwright для интеграции с Percy (BrowserStack).
// Для локального запуска без Percy использует baseline screenshots.
//
// Покрытие:
//   - Dashboard (Overview, Maintenance, SLA, Performance tabs)
//   - Devices list page + Device detail
//   - AssetTree
//   - Work Orders list + creation wizard
//   - P2P Registration
//   - RCA Widget
//   - Settings page
//   - Login page
// ═══════════════════════════════════════════════════════════════════════════

import { test, expect, Page } from '@playwright/test';
import {
  setupAllMocks,
  navigateAndWait,
  MOCK_ADMIN_USER,
} from '../../frontend/e2e/shared-mocks';

// ── Percy Integration ──────────────────────────────────────────────────
// Percy SDK (optional): если установлен, используем @percy/playwright
// Иначе делаем локальные baseline screenshots для сравнения

let percySnapshot: ((page: Page, name: string) => Promise<void>) | null = null;

async function tryLoadPercy(): Promise<boolean> {
  try {
    const percyModule = await import('@percy/playwright');
    percySnapshot = percyModule.percySnapshot;
    return true;
  } catch {
    return false;
  }
}

async function takeScreenshot(
  page: Page,
  name: string,
  fullPage: boolean = true,
): Promise<void> {
  if (percySnapshot) {
    // Percy cloud snapshot
    await percySnapshot(page, name);
  } else {
    // Local baseline screenshot
    await page.screenshot({
      path: `tests/visual/baseline/${name.replace(/[^a-zA-Z0-9_-]/g, '_')}.png`,
      fullPage,
    });
  }
}

// ── Setup ──────────────────────────────────────────────────────────────

test.beforeAll(async () => {
  await tryLoadPercy();
});

test.beforeEach(async ({ page }) => {
  await setupAllMocks(page, MOCK_ADMIN_USER);
});

// ═══ Dashboard Screenshots ═════════════════════════════════════════════

test.describe('Dashboard — Visual Regression', () => {
  test('Dashboard Overview tab', async ({ page }) => {
    await navigateAndWait(page, '/');
    await takeScreenshot(page, 'Dashboard-Overview');
  });

  test('Dashboard Maintenance tab', async ({ page }) => {
    await navigateAndWait(page, '/');
    // Переключаемся на Maintenance tab
    await page.click('text=Maintenance');
    await page.waitForTimeout(1000);
    await takeScreenshot(page, 'Dashboard-Maintenance');
  });

  test('Dashboard SLA Compliance tab', async ({ page }) => {
    await navigateAndWait(page, '/');
    await page.click('text=SLA Compliance');
    await page.waitForTimeout(1000);
    await takeScreenshot(page, 'Dashboard-SLA-Compliance');
  });

  test('Dashboard Performance tab', async ({ page }) => {
    await navigateAndWait(page, '/');
    await page.click('text=Performance');
    await page.waitForTimeout(1000);
    await takeScreenshot(page, 'Dashboard-Performance');
  });
});

// ═══ Devices Screenshots ═══════════════════════════════════════════════

test.describe('Devices — Visual Regression', () => {
  test('Devices list page', async ({ page }) => {
    await navigateAndWait(page, '/devices');
    await takeScreenshot(page, 'Devices-List');
  });

  test('Devices — search active', async ({ page }) => {
    await navigateAndWait(page, '/devices');
    const searchInput = page.locator('input[type="text"]').first();
    if (await searchInput.isVisible()) {
      await searchInput.fill('Camera');
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Devices-Search-Results');
  });

  test('Device detail page', async ({ page }) => {
    await navigateAndWait(page, '/devices/dev-1');
    await takeScreenshot(page, 'Devices-Detail');
  });

  test('Device Audit Log', async ({ page }) => {
    await navigateAndWait(page, '/devices/dev-1/audit');
    await takeScreenshot(page, 'Devices-Audit-Log');
  });
});

// ═══ AssetTree Screenshots ═════════════════════════════════════════════

test.describe('AssetTree — Visual Regression', () => {
  test('AssetTree — collapsed view', async ({ page }) => {
    await navigateAndWait(page, '/assets');
    await takeScreenshot(page, 'AssetTree-Collapsed');
  });

  test('AssetTree — expanded view', async ({ page }) => {
    await navigateAndWait(page, '/assets');
    // Expand all
    const expandBtn = page.locator('button:has-text("Expand")').first();
    if (await expandBtn.isVisible()) {
      await expandBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'AssetTree-Expanded');
  });

  test('AssetTree — search results highlighted', async ({ page }) => {
    await navigateAndWait(page, '/assets');
    const searchInput = page.locator('input[type="text"]').first();
    if (await searchInput.isVisible()) {
      await searchInput.fill('Camera');
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'AssetTree-Search-Highlighted');
  });
});

// ═══ Work Orders Screenshots ═══════════════════════════════════════════

test.describe('Work Orders — Visual Regression', () => {
  test('Work Orders list', async ({ page }) => {
    await navigateAndWait(page, '/work-orders');
    await takeScreenshot(page, 'WorkOrders-List');
  });

  test('Work Order detail page', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/WO-001');
    await takeScreenshot(page, 'WorkOrders-Detail');
  });

  test('Work Order creation wizard', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/create');
    await takeScreenshot(page, 'WorkOrders-Create-Wizard');
  });

  test('Work Order Kanban board', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/kanban');
    await takeScreenshot(page, 'WorkOrders-Kanban');
  });

  test('Work Order Calendar', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/calendar');
    await takeScreenshot(page, 'WorkOrders-Calendar');
  });
});

// ═══ P2P Screenshots ══════════════════════════════════════════════════

test.describe('P2P Gateway — Visual Regression', () => {
  test('P2P Registration form', async ({ page }) => {
    await navigateAndWait(page, '/p2p/register');
    await takeScreenshot(page, 'P2P-Registration-Form');
  });

  test('P2P Devices list', async ({ page }) => {
    await navigateAndWait(page, '/p2p');
    await takeScreenshot(page, 'P2P-Devices-List');
  });
});

// ═══ RCA Screenshots ═══════════════════════════════════════════════════

test.describe('RCA — Visual Regression', () => {
  test('RCA Widget', async ({ page }) => {
    await navigateAndWait(page, '/rca');
    await takeScreenshot(page, 'RCA-Widget');
  });

  test('RCA Investigation detail with graph', async ({ page }) => {
    await navigateAndWait(page, '/rca/rca-1');
    await takeScreenshot(page, 'RCA-Detail-Graph');
  });
});

// ═══ Settings Screenshots ══════════════════════════════════════════════

test.describe('Settings — Visual Regression', () => {
  test('Settings page', async ({ page }) => {
    await navigateAndWait(page, '/settings');
    await takeScreenshot(page, 'Settings-General');
  });
});

// ═══ Login Screenshots ═════════════════════════════════════════════════

test.describe('Login — Visual Regression', () => {
  test('Login page', async ({ page }) => {
    await navigateAndWait(page, '/login');
    await takeScreenshot(page, 'Login-Page');
  });

  test('Login — error state', async ({ page }) => {
    await navigateAndWait(page, '/login');
    // Submit empty form to trigger validation
    const submitBtn = page.locator('button[type="submit"]').first();
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Login-Validation-Error');
  });
});

// ═══ Empty/Error States ════════════════════════════════════════════════

test.describe('Empty/Error States — Visual Regression', () => {
  test('Empty state — no devices', async ({ page }) => {
    // Override device mock with empty array
    await page.route('**/api/v1/devices*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });
    await navigateAndWait(page, '/devices');
    await takeScreenshot(page, 'EmptyState-No-Devices');
  });

  test('Error state — API failure', async ({ page }) => {
    await page.route('**/api/v1/dashboard/stats', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });
    await navigateAndWait(page, '/');
    await takeScreenshot(page, 'ErrorState-API-Failure');
  });

  test('Loading state — skeleton', async ({ page }) => {
    // Delay API response to show loading skeleton
    await page.route('**/api/v1/dashboard/stats', async (route) => {
      await new Promise((r) => setTimeout(r, 3000));
      await route.continue();
    });
    await page.goto('/');
    // Take screenshot during loading
    await page.waitForTimeout(500);
    await takeScreenshot(page, 'LoadingState-Skeleton');
  });
});

// ═══ Responsive Screenshots ════════════════════════════════════════════

test.describe('Responsive — Visual Regression', () => {
  test('Mobile viewport — Dashboard', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await navigateAndWait(page, '/');
    await takeScreenshot(page, 'Responsive-Mobile-Dashboard');
  });

  test('Tablet viewport — Dashboard', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 });
    await navigateAndWait(page, '/');
    await takeScreenshot(page, 'Responsive-Tablet-Dashboard');
  });

  test('Mobile viewport — Devices list', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await navigateAndWait(page, '/devices');
    await takeScreenshot(page, 'Responsive-Mobile-Devices');
  });
});
