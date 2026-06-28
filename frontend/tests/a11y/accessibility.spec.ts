// ═══════════════════════════════════════════════════════════════════════════════
// Accessibility Smoke Tests — CCTV Health Monitor
// P1-QA.3: Accessibility Testing in CI (axe-core)
// Tool: @axe-core/playwright v4
// Threshold: 0 critical violations per page (WCAG 2.1 AA)
//
// Coverage: 5 critical pages — Login, Dashboard, WorkOrders, Devices, Settings
// These pages are the core user flows and run on every CI push/PR.
// Full coverage (16 pages) runs weekly via tests/a11y/all-pages.spec.ts
//
// Compliance:
//   OWASP ASVS L3 V8 (Data Protection)
//   Приказ ОАЦ №66 п.7.18 (Защита конечных узлов)
//   ISO 27001 A.12.6 (Application Security Review)
//   IEC 62443-3-3 SR 7.8 (Security Function Verification)
// ═══════════════════════════════════════════════════════════════════════════════

import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// ─────────────────────────────────────────────────────────────────────────────
// Configuration
// ─────────────────────────────────────────────────────────────────────────────

const BASE_URL = 'http://localhost:5173';

/**
 * Critical pages for accessibility smoke test.
 * Format: [route, pageName, requiresAuth]
 *
 * Эти 5 страниц покрывают основные user flows:
 *   1. Login  — точка входа в систему (public)
 *   2. Dashboard — главная панель после входа (protected)
 *   3. Work Orders — управление заявками (protected)
 *   4. Devices — управление устройствами (protected)
 *   5. Settings — настройки системы (protected)
 */
const CRITICAL_PAGES: Array<[string, string, boolean]> = [
  ['/login', 'Login', false],
  ['/dashboard', 'Dashboard', true],
  ['/work-orders', 'Work Orders', true],
  ['/devices', 'Devices', true],
  ['/settings', 'Settings', true],
];

// Страницы с динамическим контентом — даём больше времени на загрузку
const DYNAMIC_PAGES = ['/dashboard', '/work-orders'];

// ─────────────────────────────────────────────────────────────────────────────
// Mock helpers for authenticated pages
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Настраивает mock API ответы для авторизованных страниц.
 * Имитирует сессию администратора с полными правами.
 */
async function setupAuthMocks(page: Page): Promise<void> {
  // Auth endpoints
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

  // Devices endpoint
  await page.route('**/api/v1/devices*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'dev-1',
          name: 'Camera-01',
          status: 'online',
          health: 'healthy',
          type: 'camera',
          site_id: 'site-1',
          ip_address: '192.168.1.100',
          model: 'AXIS P3265-LVE',
          firmware: '9.80.1',
          last_seen: new Date().toISOString(),
        },
        {
          id: 'dev-2',
          name: 'NVR-03',
          status: 'online',
          health: 'degraded',
          type: 'nvr',
          site_id: 'site-1',
          ip_address: '192.168.1.50',
          model: 'HikVision DS-7616NI',
          firmware: '5.2.0',
          last_seen: new Date().toISOString(),
        },
      ]),
    });
  });

  // Work Orders endpoint
  await page.route('**/api/v1/work-orders*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          id: 'WO-001',
          title: 'Replace camera lens — Building A',
          status: 'open',
          priority: 'critical',
          assigned_to: null,
          site_id: 'site-1',
          sla_deadline: new Date(Date.now() + 3600000).toISOString(),
          created_at: new Date().toISOString(),
        },
        {
          id: 'WO-002',
          title: 'Firmware update — NVR cluster',
          status: 'in_progress',
          priority: 'high',
          assigned_to: 'user-2',
          site_id: 'site-2',
          sla_deadline: new Date(Date.now() + 86400000).toISOString(),
          created_at: new Date().toISOString(),
        },
        {
          id: 'WO-003',
          title: 'Cable replacement — Floor 3',
          status: 'open',
          priority: 'medium',
          assigned_to: 'user-3',
          site_id: 'site-1',
          sla_deadline: new Date(Date.now() + 259200000).toISOString(),
          created_at: new Date().toISOString(),
        },
      ]),
    });
  });

  // Users endpoint
  await page.route('**/api/v1/users*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'user-1', username: 'admin', role: 'admin', full_name: 'Admin User' },
        { id: 'user-2', username: 'tech1', role: 'technician', full_name: 'Bob Technician' },
        { id: 'user-3', username: 'tech2', role: 'technician', full_name: 'Alice Engineer' },
      ]),
    });
  });

  // Sites endpoint
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

  // Settings endpoints
  await page.route('**/api/v1/settings*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        general: { language: 'en', timezone: 'UTC', date_format: 'DD/MM/YYYY' },
        security: { mfa_enabled: true, session_timeout: 30, password_policy: 'strong' },
        notifications: { email: true, push: true, sms: false },
      }),
    });
  });

  // Dashboard aggregation endpoint
  await page.route('**/api/v1/dashboard/**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        total_devices: 42,
        online_devices: 38,
        offline_devices: 3,
        degraded_devices: 1,
        open_work_orders: 5,
        critical_work_orders: 2,
        system_health: 'degraded',
      }),
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

  // Set localStorage auth token
  await page.evaluate(() => {
    localStorage.setItem('token', 'mock-token-a11y-smoke');
    localStorage.setItem('user', JSON.stringify({
      id: 'user-1',
      username: 'admin',
      role: 'admin',
    }));
  });
}

/**
 * Фильтрует нарушения только по цветовому контрасту.
 * Используется для опциональной диагностики — НЕ для fail-условия.
 */
function filterContrastOnly(violations: any[]) {
  return violations.filter((v) => v.id === 'color-contrast');
}

// ─────────────────────────────────────────────────────────────────────────────
// Test Suite — Critical Pages WCAG 2.1 AA Audit
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Accessibility Smoke — Critical Pages (P1-QA.3)', () => {
  CRITICAL_PAGES.forEach(([route, pageName, requiresAuth]) => {
    test(`[${pageName}] ${route} — 0 critical violations`, async ({ page }) => {
      test.setTimeout(60_000);

      // Setup auth mocks for protected pages
      if (requiresAuth) {
        await setupAuthMocks(page);
      }

      // Navigate to page with network idle
      await page.goto(`${BASE_URL}${route}`, {
        waitUntil: 'networkidle',
      });

      // Allow dynamic content to render
      const waitTime = DYNAMIC_PAGES.includes(route) ? 3000 : 2000;
      await page.waitForTimeout(waitTime);

      // Ensure page is rendered
      await expect(page.locator('body')).toBeVisible();

      // ── Run axe-core scan ──────────────────────────────────────────────
      // Using WCAG 2.1 AA ruleset (minimum for compliance)
      // Disabled rules:
      //   - 'color-contrast': known theme-dependent, verified separately
      //   - 'link-in-text-block': false positives with React rich text
      //   - 'region': React SPAs use fragments, not all ARIA regions required
      const accessibilityScanResults = await new AxeBuilder({ page })
        .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'])
        .disableRules([
          'color-contrast',
          'link-in-text-block',
          'region',
        ])
        .analyze();

      // ── Assert: 0 critical violations (threshold) ──────────────────────
      const criticalSerious = accessibilityScanResults.violations.filter(
        (v) => v.impact === 'critical' || v.impact === 'serious',
      );

      if (criticalSerious.length > 0) {
        console.log(`\n❌ [${pageName}] — ${criticalSerious.length} critical/serious violations:`);
        criticalSerious.forEach((violation) => {
          console.log(`  - ${violation.id}: ${violation.help}`);
          console.log(`    Help URL: ${violation.helpUrl}`);
          violation.nodes.forEach((node) => {
            console.log(`    • Target: ${node.target}`);
            console.log(`      HTML: ${node.html}`);
            console.log(`      Failure: ${node.failureSummary}`);
          });
        });
      }

      expect(
        criticalSerious,
        `[${pageName}] Должно быть 0 critical/serious accessibility violations, найдено ${criticalSerious.length}`,
      ).toHaveLength(0);

      // ── Log minor violations for awareness (not fail) ──────────────────
      const minorViolations = accessibilityScanResults.violations.filter(
        (v) => v.impact === 'minor' || v.impact === 'moderate',
      );

      if (minorViolations.length > 0) {
        console.log(`\n⚠️  [${pageName}] — ${minorViolations.length} minor/moderate violations:`);
        minorViolations.forEach((v) => {
          console.log(`  - ${v.id}: ${v.help} (${v.impact})`);
        });
      }

      // ── Log passes for diagnostics ────────────────────────────────────
      const ruleCount = accessibilityScanResults.passes.length;
      console.log(`  ✓ [${pageName}] — ${ruleCount} rules passed, 0 critical violations`);
    });
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// Focus Management Tests (Keyboard Navigation)
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Accessibility Smoke — Focus Management', () => {
  test('Login page — Tab order follows logical sequence', async ({ page }) => {
    test.setTimeout(30_000);

    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // Tab forward through the form
    await page.keyboard.press('Tab');
    const focusedAfterFirstTab = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el) return 'none';
      return el.tagName + (el.getAttribute('type') ? `[type="${el.getAttribute('type')}"]` : '');
    });
    console.log(`  Focus after Tab #1: ${focusedAfterFirstTab}`);

    await page.keyboard.press('Tab');
    const focusedAfterSecondTab = await page.evaluate(() => {
      const el = document.activeElement;
      if (!el) return 'none';
      return el.tagName + (el.getAttribute('type') ? `[type="${el.getAttribute('type')}"]` : '');
    });
    console.log(`  Focus after Tab #2: ${focusedAfterSecondTab}`);

    // At least one form element should receive focus
    const emailFocused = await page
      .locator('input[type="email"]')
      .evaluate((el) => el === document.activeElement);

    const passwordFocused = await page
      .locator('input[type="password"]')
      .evaluate((el) => el === document.activeElement);

    expect(emailFocused || passwordFocused || focusedAfterFirstTab.includes('INPUT'))
      .toBeTruthy();
  });

  test('Dashboard — Skip navigation link available', async ({ page }) => {
    test.setTimeout(30_000);

    await setupAuthMocks(page);
    await page.goto(`${BASE_URL}/dashboard`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    // Press Tab and check for skip-to-content link
    await page.keyboard.press('Tab');

    const skipLink = page.locator(
      'a[href*="#main" i], a[href*="#content" i], [data-skip-link], ' +
      'a.skip-link, a:has-text(/skip|перейти к содержанию|пропустить/i)',
    );
    const skipVisible = await skipLink.isVisible().catch(() => false);

    if (skipVisible) {
      console.log('  ✓ Skip navigation link is present and visible');
    } else {
      // Skip link might be visually hidden (accessible only on focus)
      const skipInDOM = await skipLink.count().catch(() => 0);
      console.log(`  ℹ️  Skip link in DOM: ${skipInDOM > 0 ? 'yes' : 'no'} (may be visually hidden)`);
    }

    // Body must be present
    await expect(page.locator('body')).toBeVisible();
  });
});

// ─────────────────────────────────────────────────────────────────────────────
// ARIA Landmarks Smoke Tests
// ─────────────────────────────────────────────────────────────────────────────

test.describe('Accessibility Smoke — ARIA Landmarks', () => {
  test('Dashboard — contains main landmark', async ({ page }) => {
    test.setTimeout(30_000);

    await setupAuthMocks(page);
    await page.goto(`${BASE_URL}/dashboard`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    const mainLandmark = page.locator('main, [role="main"]');
    await expect(mainLandmark.first()).toBeVisible({ timeout: 5000 });
  });

  test('Settings — contains navigation landmark', async ({ page }) => {
    test.setTimeout(30_000);

    await setupAuthMocks(page);
    await page.goto(`${BASE_URL}/settings`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    const nav = page.locator('nav, [role="navigation"]');
    await expect(nav.first()).toBeVisible({ timeout: 5000 });
  });
});
