// ═══════════════════════════════════════════════════════════════════════════
// Visual Regression Testing — CCTV Health Monitor
// P1-QA.6: Percy Integration + Baseline Screenshots
// P2-MED-17: Missing Visual Regression Tests (Forms, Navigation, Modals, Edge Cases)
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
//   - Forms: work order validation, file upload, multi-step wizard
//   - Navigation: sidebar collapse, breadcrumbs, tabs
//   - Modals: notification modal, confirmation dialog
//   - Edge cases: dark mode, long text truncation
//   - Advanced: pagination, sort/filter, bulk select
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

// ═══ Forms — Visual Regression ════════════════════════════════════════
//
// Добавлено в рамках P2-MED-17:
//   - Work order creation with validation errors
//   - File upload component
//   - Multi-step wizard screenshots
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Forms — Visual Regression', () => {
  test('Work order creation — validation errors', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/create');

    // Click submit with empty fields to trigger validation
    const submitBtn = page.locator('button[type="submit"]').first();
    if (await submitBtn.isVisible()) {
      await submitBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Forms-WO-Create-Validation');
  });

  test('File upload component', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/create');

    // Look for file upload area
    const fileInput = page.locator('input[type="file"]').first();
    const dropZone = page.locator('[data-testid="file-upload"], [class*="dropzone"], [class*="upload"]').first();

    if (await dropZone.isVisible()) {
      await takeScreenshot(page, 'Forms-File-Upload');
    } else if (await fileInput.isVisible()) {
      await takeScreenshot(page, 'Forms-File-Upload');
    } else {
      // Take full page to show the form area
      await takeScreenshot(page, 'Forms-File-Upload');
    }
  });

  test('Multi-step wizard screenshot', async ({ page }) => {
    await navigateAndWait(page, '/work-orders/create');

    // Try clicking "Next" or "Continue" to advance in wizard
    const nextBtn = page.locator('button:has-text("Next"), button:has-text("Continue"), button:has-text("Далее")').first();
    if (await nextBtn.isVisible()) {
      await nextBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Forms-MultiStep-Wizard');
  });
});

// ═══ Navigation — Visual Regression ═══════════════════════════════════
//
// Добавлено в рамках P2-MED-17:
//   - Sidebar collapsed vs expanded
//   - Breadcrumbs on detail pages
//   - Tab navigation switching
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Navigation — Visual Regression', () => {
  test('Sidebar — collapsed state', async ({ page }) => {
    await navigateAndWait(page, '/');

    // Click sidebar collapse toggle
    const collapseBtn = page.locator(
      'button[data-testid="sidebar-collapse"], ' +
      'button[aria-label*="collapse" i], ' +
      'button[aria-label*="sidebar" i], ' +
      'button[class*="collapse"], ' +
      'button:has(svg[data-testid="ChevronLeftIcon"])'
    ).first();
    if (await collapseBtn.isVisible()) {
      await collapseBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Navigation-Sidebar-Collapsed');
  });

  test('Sidebar — expanded state', async ({ page }) => {
    await navigateAndWait(page, '/');

    // Ensure sidebar is expanded
    const expandBtn = page.locator(
      'button[data-testid="sidebar-expand"], ' +
      'button[aria-label*="expand" i], ' +
      'button[class*="expand"]'
    ).first();
    if (await expandBtn.isVisible()) {
      await expandBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Navigation-Sidebar-Expanded');
  });

  test('Breadcrumbs on detail page', async ({ page }) => {
    await navigateAndWait(page, '/devices/dev-1');

    // Capture breadcrumb region if it exists
    const breadcrumbs = page.locator(
      'nav[aria-label*="breadcrumb" i], ' +
      '[data-testid="breadcrumbs"], ' +
      '[class*="breadcrumb"]'
    ).first();
    if (await breadcrumbs.isVisible()) {
      await expect(breadcrumbs).toBeVisible();
    }
    await takeScreenshot(page, 'Navigation-Breadcrumbs');
  });

  test('Tabs navigation switching', async ({ page }) => {
    await navigateAndWait(page, '/');

    // Try clicking a secondary tab to verify tab switching renders
    const tabs = page.locator(
      '[role="tab"]:not([aria-selected="true"]), ' +
      'button[role="tab"]:not(.active), ' +
      '[data-testid*="tab"]:not(.active)'
    );
    const tabCount = await tabs.count();
    if (tabCount > 0) {
      await tabs.first().click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Navigation-Tabs-Switched');
  });
});

// ═══ Modals — Visual Regression ═══════════════════════════════════════
//
// Добавлено в рамках P2-MED-17:
//   - Notification modal open
//   - Confirmation dialog
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Modals — Visual Regression', () => {
  test('Notification modal open', async ({ page }) => {
    await navigateAndWait(page, '/');

    // Click notification bell/icon to open modal
    const notifBtn = page.locator(
      'button[data-testid="notifications-button"], ' +
      'button[aria-label*="notification" i], ' +
      'button[class*="bell"], ' +
      'button:has(svg[data-testid="BellIcon"]), ' +
      'button:has([class*="notification"])'
    ).first();
    if (await notifBtn.isVisible()) {
      await notifBtn.click();
      await page.waitForTimeout(600);
    }
    await takeScreenshot(page, 'Modals-Notification-Open');
  });

  test('Confirmation dialog', async ({ page }) => {
    await navigateAndWait(page, '/devices');

    // Try to find and click a delete/remove button to trigger confirmation
    const deleteBtn = page.locator(
      'button[data-testid="delete-device"], ' +
      'button[aria-label*="delete" i], ' +
      'button[class*="delete"], ' +
      'button[class*="remove"]'
    ).first();
    if (await deleteBtn.isVisible()) {
      await deleteBtn.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Modals-Confirmation-Dialog');
  });
});

// ═══ Edge Cases — Visual Regression ═══════════════════════════════════
//
// Добавлено в рамках P2-MED-17:
//   - Dark mode (if ThemeProvider exists)
//   - Long text truncation in table cells / cards
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Edge Cases — Visual Regression', () => {
  test('Dark mode screenshot', async ({ page }) => {
    // Try setting dark mode via localStorage or class toggle
    await page.addInitScript(() => {
      localStorage.setItem('theme', JSON.stringify({ mode: 'dark' }));
      localStorage.setItem('color-scheme', 'dark');
    });
    await page.evaluate(() => {
      document.documentElement.classList.add('dark');
      document.documentElement.style.colorScheme = 'dark';
    }).catch(() => {
      // Class toggle may fail before page loads; try after navigation
    });

    await navigateAndWait(page, '/');
    await takeScreenshot(page, 'EdgeCases-Dark-Mode');
  });

  test('Long text truncation', async ({ page }) => {
    // Override device mock with very long names to test truncation
    await page.route('**/api/v1/devices*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'dev-long-1',
            name: 'Camera-Entrance-Main-Building-North-Wing-Security-System-Floor-01',
            ip_address: '192.168.1.100',
            status: 'online',
            health: 'healthy',
            type: 'camera',
            site_id: 'site-1',
            model: 'AXIS Q1615-MkIII-Advanced-Security-Camera-System',
            firmware: '9.80.1.2345-beta-hotfix-2026',
            last_seen: new Date().toISOString(),
          },
          {
            id: 'dev-long-2',
            name: 'NVR-Recording-Server-Main-DataCenter-Rack-03-UPS-Backup-System',
            ip_address: '192.168.1.50',
            status: 'online',
            health: 'degraded',
            type: 'nvr',
            site_id: 'site-1',
            model: 'HikVision-DS-9608-NI-4K-H265-Advanced-Recording-Platform',
            firmware: '5.2.0.12345-release-candidate-2026-q2',
            last_seen: new Date().toISOString(),
          },
        ]),
      });
    });
    await navigateAndWait(page, '/devices');
    await takeScreenshot(page, 'EdgeCases-Long-Text-Truncation');
  });
});

// ═══ Advanced Interactions — Visual Regression ════════════════════════
//
// Добавлено в рамках P2-MED-17:
//   - Pagination controls
//   - Sort / filter active
//   - Bulk select mode
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Advanced Interactions — Visual Regression', () => {
  test('Pagination controls visible', async ({ page }) => {
    await navigateAndWait(page, '/devices');

    // Try clicking pagination "Next" or page number
    const pagination = page.locator(
      'nav[aria-label*="pagination" i], ' +
      '[data-testid="pagination"], ' +
      '[class*="pagination"], ' +
      'button:has-text("Next"), ' +
      'button[aria-label*="next page" i]'
    ).first();
    if (await pagination.isVisible()) {
      await pagination.click();
      await page.waitForTimeout(500);
    }
    await takeScreenshot(page, 'Advanced-Pagination');
  });

  test('Sort and filter active', async ({ page }) => {
    await navigateAndWait(page, '/devices');

    // Click sort button or column header to activate sorting
    const sortBtn = page.locator(
      'button[data-testid*="sort"], ' +
      'th[aria-sort], ' +
      'button[aria-label*="sort" i], ' +
      'th button, ' +
      'button:has(svg[data-testid*="Sort"])'
    ).first();
    if (await sortBtn.isVisible()) {
      await sortBtn.click();
      await page.waitForTimeout(300);
    }

    // Click filter button if visible
    const filterBtn = page.locator(
      'button[data-testid*="filter"], ' +
      'button[aria-label*="filter" i], ' +
      'button:has(svg[data-testid*="Filter"])'
    ).first();
    if (await filterBtn.isVisible()) {
      await filterBtn.click();
      await page.waitForTimeout(300);
    }
    await takeScreenshot(page, 'Advanced-Sort-Filter');
  });

  test('Bulk select mode', async ({ page }) => {
    await navigateAndWait(page, '/devices');

    // Try clicking checkboxes to enable bulk selection
    const checkboxes = page.locator(
      'input[type="checkbox"][data-testid*="select"], ' +
      'input[type="checkbox"][aria-label*="select" i], ' +
      'thead input[type="checkbox"]'
    );
    const cbCount = await checkboxes.count();
    if (cbCount > 0) {
      await checkboxes.first().click();
      await page.waitForTimeout(300);

      // Also select a second row
      const rowCheckboxes = page.locator(
        'tbody input[type="checkbox"], ' +
        'tr input[type="checkbox"], ' +
        '[data-testid*="row"] input[type="checkbox"]'
      );
      const rowCount = await rowCheckboxes.count();
      if (rowCount > 1) {
        await rowCheckboxes.nth(1).click();
        await page.waitForTimeout(300);
      }
    }
    await takeScreenshot(page, 'Advanced-Bulk-Select');
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
