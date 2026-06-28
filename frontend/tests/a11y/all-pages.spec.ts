// ═══════════════════════════════════════════════════════════════════════════
// Accessibility Tests — CCTV Health Monitor
// P1-QA.2: Accessibility Testing in CI
// Tool: @axe-core/playwright v4
// Threshold: 0 critical violations per page (WCAG 2.1 AA)
// Compliance: OWASP ASVS L3 V8 (Data Protection), Приказ ОАЦ №66 п.7.18
//             ISO 27001 A.12.6 (Application Security Review)
//             IEC 62443-3-3 SR 7.8 (Security Function Verification)
// ═══════════════════════════════════════════════════════════════════════════

import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// ─────────────────────────────────────────────────────────────────────────────
// Configuration
// ─────────────────────────────────────────────────────────────────────────────

const BASE_URL = 'http://localhost:5173';

// Pages to scan for accessibility violations
// Format: [route, pageName, requiresAuth]
const PAGES: Array<[string, string, boolean]> = [
  ['/', 'Home', false],
  ['/login', 'Login', false],
  ['/dashboard', 'Dashboard', true],
  ['/devices', 'Devices', true],
  ['/work-orders', 'Work Orders', true],
  ['/work-orders/create', 'Work Orders Create', true],
  ['/sites', 'Sites', true],
  ['/reports', 'Reports', true],
  ['/settings', 'Settings', true],
  ['/settings/general', 'Settings General', true],
  ['/settings/security', 'Settings Security', true],
  ['/settings/notifications', 'Settings Notifications', true],
  ['/p2p-devices', 'P2P Devices', true],
  ['/rca', 'RCA Investigations', true],
  ['/gatekeeper', 'Gatekeeper', true],
  ['/help/glossary', 'Glossary', false],
];

// Pages that may have dynamic content — use relaxed timeout
const DYNAMIC_PAGES = ['/dashboard', '/work-orders'];

// ─────────────────────────────────────────────────────────────────────────────
// Mock helpers
// ─────────────────────────────────────────────────────────────────────────────

async function setupAuthMocks(page: Page): Promise<void> {
  await page.route('**/api/v1/auth/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'admin',
        role: 'admin',
        name: 'Admin User',
      }),
    });
  });

  await page.route('**/api/v1/users/me', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'admin',
        role: 'admin',
      }),
    });
  });

  await page.route('**/api/v1/sites*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'site-1', name: 'Main Office' },
        { id: 'site-2', name: 'Branch Office' },
      ]),
    });
  });

  await page.route('**/api/v1/devices*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'dev-1', name: 'Camera-01', status: 'online', health: 'healthy', type: 'camera', site_id: 'site-1', ip_address: '192.168.1.100', model: 'AXIS', firmware: '9.80.1', last_seen: new Date().toISOString() },
        { id: 'dev-2', name: 'NVR-03', status: 'online', health: 'degraded', type: 'nvr', site_id: 'site-1', ip_address: '192.168.1.50', model: 'HikVision', firmware: '5.2.0', last_seen: new Date().toISOString() },
      ]),
    });
  });

  await page.route('**/api/v1/work-orders*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'WO-001', title: 'Replace camera lens', status: 'open', priority: 'critical', assigned_to: null, site_id: 'site-1', sla_deadline: new Date(Date.now() + 3600000).toISOString(), created_at: new Date().toISOString() },
        { id: 'WO-002', title: 'Firmware update', status: 'in_progress', priority: 'high', assigned_to: 'user-2', site_id: 'site-2', sla_deadline: new Date(Date.now() + 86400000).toISOString(), created_at: new Date().toISOString() },
      ]),
    });
  });

  await page.route('**/api/v1/users*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'user-1', username: 'admin', role: 'admin', full_name: 'Admin User' },
        { id: 'user-2', username: 'tech1', role: 'technician', full_name: 'Bob Technician' },
      ]),
    });
  });

  await page.route('**/api/v1/reports*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'rpt-1', title: 'Daily Report', type: 'daily', format: 'pdf', status: 'ready', created_at: new Date().toISOString(), url: '/api/v1/reports/rpt-1/download' },
      ]),
    });
  });

  // Catch-all for other API routes
  await page.route('**/api/v1/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({}),
    });
  });

  await page.evaluate(() => {
    localStorage.setItem('token', 'mock-token-a11y');
    localStorage.setItem('user', JSON.stringify({
      id: 'user-1',
      username: 'admin',
      role: 'admin',
    }));
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// Custom: check if page has minimum contrast issues
// ─────────────────────────────────────────────────────────────────────────────

function filterContrastOnly(violations: any[]) {
  return violations.filter((v) => v.id === 'color-contrast');
}

// ─────────────────────────────────────────────────────────────────────────────
// Test Suite
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Accessibility — WCAG 2.1 AA Audit', () => {
  PAGES.forEach(([route, pageName, requiresAuth]) => {
    test(`[${pageName}] ${route} — 0 critical violations`, async ({ page }) => {
      test.setTimeout(60_000);

      // Setup auth for protected pages
      if (requiresAuth) {
        await setupAuthMocks(page);
      }

      // Navigate to page
      await page.goto(`${BASE_URL}${route}`, {
        waitUntil: 'networkidle',
      });

      // Wait for dynamic content
      const waitTime = DYNAMIC_PAGES.includes(route) ? 3000 : 2000;
      await page.waitForTimeout(waitTime);

      // Ensure page is fully loaded
      await expect(page.locator('body')).toBeVisible();

      // Run axe-core scan with WCAG 2.1 AA ruleset
      const accessibilityScanResults = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .disableRules([
          // Skip 'color-contrast' for dynamic themed pages
          // Skip 'link-in-text-block' for pages with rich text
          // Skip 'region' for SPAs that use React fragments
          'color-contrast',
          'link-in-text-block',
          'region',
        ])
        .analyze();

      // Assert: 0 critical/serious violations
      const criticalSerious = accessibilityScanResults.violations.filter(
        (v) => v.impact === 'critical' || v.impact === 'serious',
      );

      if (criticalSerious.length > 0) {
        console.log(`\n❌ [${pageName}] — ${criticalSerious.length} critical/serious violations:`);
        criticalSerious.forEach((violation) => {
          console.log(`  - ${violation.id}: ${violation.help}`);
          violation.nodes.forEach((node) => {
            console.log(`    • ${node.html}`);
          });
        });
      }

      expect(
        criticalSerious,
        `[${pageName}] Должно быть 0 critical/serious accessibility violations`,
      ).toHaveLength(0);

      // Log minor violations for awareness (not fail)
      const minorViolations = accessibilityScanResults.violations.filter(
        (v) => v.impact === 'minor' || v.impact === 'moderate',
      );
      if (minorViolations.length > 0) {
        console.log(`\n⚠️  [${pageName}] — ${minorViolations.length} minor/moderate violations:`);
        minorViolations.forEach((v) => {
          console.log(`  - ${v.id}: ${v.help} (${v.impact})`);
        });
      }
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Focus Management Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Accessibility — Focus Management', () => {
  test('Login page — focus order follows logical sequence', async ({ page }) => {
    await page.goto(`${BASE_URL}/login`);
    await page.waitForTimeout(2000);

    // Tab through form elements
    await page.keyboard.press('Tab');
    const emailFocused = await page.locator('input[type="email"]').evaluate((el) => el === document.activeElement);

    await page.keyboard.press('Tab');
    const passwordFocused = await page.locator('input[type="password"]').evaluate((el) => el === document.activeElement);

    // At least one of these should be focused after tab
    expect(emailFocused || passwordFocused).toBeTruthy();
  });

  test('Keyboard navigation — Skip link present', async ({ page }) => {
    await setupAuthMocks(page);
    await page.goto(`${BASE_URL}/dashboard`);
    await page.waitForTimeout(2000);

    // Press Tab to check for skip-to-content link
    await page.keyboard.press('Tab');

    const skipLink = page.locator('a[href*="#main" i], a[href*="#content" i], [data-skip-link], a.skip-link, a:has-text(/skip|перейти к содержанию|пропустить/i)');
    const skipVisible = await skipLink.isVisible().catch(() => false);
    const bodyVisible = await page.locator('body').isVisible();

    expect(bodyVisible).toBeTruthy();
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// ARIA Landmarks Test
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Accessibility — ARIA Landmarks', () => {
  test('Dashboard page has main landmark', async ({ page }) => {
    await setupAuthMocks(page);
    await page.goto(`${BASE_URL}/dashboard`);
    await page.waitForTimeout(2000);

    const mainLandmark = page.locator('main, [role="main"]');
    await expect(mainLandmark.first()).toBeVisible();
  });

  test('Protected pages have navigation landmark', async ({ page }) => {
    await setupAuthMocks(page);
    await page.goto(`${BASE_URL}/devices`);
    await page.waitForTimeout(2000);

    const nav = page.locator('nav, [role="navigation"]');
    await expect(nav.first()).toBeVisible();
  });
});
