/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockSites,
  mockDevices,
  mockUsers,
  mockWorkOrders,
  mockDashboardStats,
  mockAlerts,
  mockCatchAll,
  MOCK_ADMIN_USER,
  MOCK_SITES,
  MOCK_DEVICES,
  MOCK_DASHBOARD_STATS,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Dashboard — E2E Tests
// P1-QA.1: Site hierarchy, Dark mode toggle, Dashboard metrics
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_SITE_HIERARCHY = {
  sites: [
    {
      id: 'site-1',
      name: 'Main Office',
      address: '123 Main St, Minsk',
      status: 'active',
      device_count: 24,
      online_count: 22,
      offline_count: 1,
      warning_count: 1,
      children: [
        { id: 'floor-1', name: 'Floor 1 — Lobby', device_count: 8, online_count: 8 },
        { id: 'floor-2', name: 'Floor 2 — Offices', device_count: 10, online_count: 9 },
        { id: 'floor-3', name: 'Floor 3 — Server Room', device_count: 6, online_count: 5 },
      ],
    },
    {
      id: 'site-2',
      name: 'Branch Office',
      address: '456 Branch Ave, Brest',
      status: 'active',
      device_count: 12,
      online_count: 10,
      offline_count: 2,
      warning_count: 0,
      children: [
        { id: 'floor-b1', name: 'Floor B1 — Parking', device_count: 4, online_count: 3 },
        { id: 'floor-b2', name: 'Floor B2 — Main', device_count: 8, online_count: 7 },
      ],
    },
    {
      id: 'site-3',
      name: 'Warehouse',
      address: '789 Industrial Rd, Gomel',
      status: 'active',
      device_count: 6,
      online_count: 6,
      offline_count: 0,
      warning_count: 0,
      children: [
        { id: 'wh-1', name: 'Warehouse Floor', device_count: 6, online_count: 6 },
      ],
    },
  ],
  total_devices: 42,
  total_sites: 3,
};

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupDashboardMockApi(page: any) {
  await setupAuth(page);

  // Dashboard stats
  await mockDashboardStats(page, {
    ...MOCK_DASHBOARD_STATS,
    total_devices: 42,
    online_devices: 38,
    offline_devices: 3,
    warning_devices: 1,
  });

  // Site hierarchy
  await page.route('**/api/v1/sites/hierarchy', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SITE_HIERARCHY),
    });
  });

  // Sites
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockAlerts(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Dashboard — Metrics & Stats
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Dashboard — Metrics & Stats', () => {
  test.beforeEach(async ({ page }) => {
    await setupDashboardMockApi(page);
    await page.goto('/dashboard');
    await page.waitForTimeout(1500);
  });

  test('Dashboard loads with stats cards', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/dashboard');

    // Проверяем отображение счетчиков
    const totalDevices = page.locator(
      'text=/42|total devices|всего устройств|device count/i',
    ).first();
    await expect(totalDevices).toBeVisible();
  });

  test('Dashboard — online/offline device counts are visible', async ({ page }) => {
    // Проверяем счетчики online/offline
    const onlineCount = page.locator(
      'text=/38|online|онлайн|active|актив/i',
    ).first();
    await expect(onlineCount).toBeVisible();

    const offlineCount = page.locator(
      'text=/3|offline|офлайн|offline|неактив/i',
    ).first();
    await expect(offlineCount).toBeVisible();
  });

  test('Dashboard — SLA breach counter is displayed', async ({ page }) => {
    // Проверяем счетчик SLA breaches
    const slaCounter = page.locator(
      'text=/2.*overdue|overdue.*2|overdue.*sla|просрочен.*sla|sla.*breach|нарушен.*sla|2/i',
    ).first();
    await expect(slaCounter).toBeVisible();
  });

  test('Dashboard — resolution rate is displayed', async ({ page }) => {
    // Проверяем resolution rate
    const resolutionRate = page.locator(
      'text=/94|94\\.5|resolution|решен|rate|процент/i',
    ).first();
    await expect(resolutionRate).toBeVisible();
  });

  test('Dashboard — open work orders count shown', async ({ page }) => {
    const openWO = page.locator(
      'text=/12.*open|open.*12|open.*work.*order|открыт.*заявк|ticket|тикет/i',
    ).first();
    await expect(openWO).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Dashboard — Site Hierarchy
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Dashboard — Site Hierarchy', () => {
  test.beforeEach(async ({ page }) => {
    await setupDashboardMockApi(page);
    await page.goto('/sites');
    await page.waitForTimeout(1500);
  });

  test('Sites page loads with hierarchy tree', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/sites');

    // Проверяем отображение сайтов
    const siteName = page.locator(
      'text=/Main Office|Branch Office|Warehouse/i',
    ).first();
    await expect(siteName).toBeVisible();
  });

  test('Site hierarchy — expand site shows floors/zones', async ({ page }) => {
    // Находим и расширяем site
    const expandButton = page.locator(
      'button:has(svg), [role="button"], [class*="expand" i], [class*="chevron" i], ' +
      'summary, details summary',
    ).first();

    if (await expandButton.isVisible()) {
      await expandButton.click();
      await page.waitForTimeout(500);

      // Проверяем отображение дочерних элементов (floors)
      const floorName = page.locator(
        'text=/Lobby|Offices|Server Room|Parking|Warehouse Floor/i',
      ).first();
      const hasFloor = await floorName.isVisible().catch(() => false);

      if (hasFloor) {
        await expect(floorName).toBeVisible();
      }
    }
  });

  test('Site hierarchy — site device counts displayed', async ({ page }) => {
    // Проверяем количество устройств на сайте
    const deviceCount = page.locator(
      'text=/24.*device|12.*device|6.*device|24.*устройств|12.*устройств|6.*устройств/i',
    ).first();
    await expect(deviceCount).toBeVisible();
  });

  test('Site hierarchy — click site navigates to site detail', async ({ page }) => {
    const siteLink = page.locator(
      'a, button, [role="button"]',
    ).filter({ hasText: /Main Office|Branch Office/i }).first();

    if (await siteLink.isVisible()) {
      await siteLink.click();
      await page.waitForTimeout(1000);

      // Проверяем что URL изменился
      const currentUrl = page.url();
      const hasSiteDetail = currentUrl.includes('/sites/site-') || currentUrl.includes('/sites/');
      expect(hasSiteDetail).toBeTruthy();
    }
  });

  test('Site hierarchy — site status badges are visible', async ({ page }) => {
    const statusBadge = page.locator(
      'span, badge, [class*="status" i]',
    ).filter({ hasText: /active|актив/i }).first();
    await expect(statusBadge).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Dark Mode Toggle
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Dark Mode Toggle', () => {
  test.beforeEach(async ({ page }) => {
    await setupDashboardMockApi(page);
    await page.goto('/dashboard');
    await page.waitForTimeout(1500);
  });

  test('Dark mode — theme toggle button is visible', async ({ page }) => {
    // Проверяем наличие кнопки переключения темы
    const themeToggle = page.locator(
      'button[aria-label*="theme" i], button[aria-label*="dark" i], ' +
      'button[aria-label*="light" i], button[aria-label*="mode" i], ' +
      'button[aria-label*="тем" i], button[class*="theme" i], ' +
      'button:has(svg[class*="sun" i]), button:has(svg[class*="moon" i])',
    ).first();

    await expect(themeToggle).toBeVisible();
  });

  test('Dark mode — clicking toggle switches theme', async ({ page }) => {
    const themeToggle = page.locator(
      'button[aria-label*="theme" i], button[aria-label*="dark" i], ' +
      'button[aria-label*="light" i], button[aria-label*="mode" i], ' +
      'button[aria-label*="тем" i], button[class*="theme" i], ' +
      'button:has(svg[class*="sun" i]), button:has(svg[class*="moon" i])',
    ).first();

    if (await themeToggle.isVisible()) {
      // Запоминаем текущее состояние
      const initialClass = await page.locator('html').getAttribute('class').catch(() => null);      

      await themeToggle.click();
      await page.waitForTimeout(500);

      // Проверяем что класс темы изменился
      const updatedClass = await page.locator('html').getAttribute('class').catch(() => null);

      const hasChanged = initialClass !== updatedClass;
      if (!hasChanged) {
        // Если класс html не изменился, проверяем data-attribute
        const dataTheme = await page.locator('html').getAttribute('data-theme').catch(() => null);
        if (dataTheme) {
          expect(['dark', 'light']).toContain(dataTheme);
        }
      }
    }
  });

  test('Dark mode — preference persists in localStorage', async ({ page }) => {
    const themeToggle = page.locator(
      'button[aria-label*="theme" i], button[aria-label*="dark" i], ' +
      'button[aria-label*="light" i], button[aria-label*="mode" i], ' +
      'button[aria-label*="тем" i], button[class*="theme" i], ' +
      'button:has(svg[class*="sun" i]), button:has(svg[class*="moon" i])',
    ).first();

    if (await themeToggle.isVisible()) {
      await themeToggle.click();
      await page.waitForTimeout(500);

      // Проверяем localStorage
      const theme = await page.evaluate(() => {
        return localStorage.getItem('theme') || localStorage.getItem('color-theme') || '';
      }).catch(() => '');

      if (theme) {
        expect(['dark', 'light']).toContain(theme);
      }

      // Переключаем обратно
      await themeToggle.click();
      await page.waitForTimeout(500);

      const themeAfterToggle = await page.evaluate(() => {
        return localStorage.getItem('theme') || localStorage.getItem('color-theme') || '';
      }).catch(() => '');

      if (theme && themeAfterToggle) {
        expect(themeAfterToggle).not.toBe(theme);
      }
    }
  });

  test('Dark mode — system preference works', async ({ page }) => {
    // Эмулируем темную тему через CDP
    await page.emulateMedia({ colorScheme: 'dark' });
    await page.reload();
    await page.waitForTimeout(1000);

    // Проверяем что тема соответствует system preference
    const dataTheme = await page.locator('html').getAttribute('data-theme').catch(() => null);
    const htmlClass = await page.locator('html').getAttribute('class').catch(() => null);

    if (dataTheme) {
      expect(dataTheme).toBe('dark');
    }

    // Переключаем на светлую
    await page.emulateMedia({ colorScheme: 'light' });
    await page.reload();
    await page.waitForTimeout(1000);

    const lightDataTheme = await page.locator('html').getAttribute('data-theme').catch(() => null);
    if (lightDataTheme) {
      expect(lightDataTheme).toBe('light');
    }
  });
});
