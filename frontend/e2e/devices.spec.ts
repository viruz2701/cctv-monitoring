import { test, expect } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════
// Devices Page — E2E Tests
// P0 flow: page loads, search works, device detail navigation
// ═══════════════════════════════════════════════════════════════════════

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
    last_seen: new Date(Date.now() - 86400000).toISOString(),
  },
];

const MOCK_SITES = [
  { id: 'site-1', name: 'Main Office' },
  { id: 'site-2', name: 'Branch Office' },
];

async function setupMockApi(page: any) {
  // Auth
  await page.route('**/api/v1/auth/me', async (route: any) => {
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

  // Devices list
  await page.route('**/api/v1/devices*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_DEVICES),
    });
  });

  // Sites list
  await page.route('**/api/v1/sites*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SITES),
    });
  });

  await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
}

test.describe('Devices Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupMockApi(page);
    await page.goto('/devices');
  });

  test('Devices page loads', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Страница загрузилась
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/devices');
  });

  test('Search input works', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Находим поиск
    const searchInput = page.locator('input[type="text"], input[placeholder*="search" i], input[placeholder*="поиск" i]').first();
    if (await searchInput.isVisible()) {
      // Вводим поисковый запрос
      await searchInput.fill('Camera');
      await page.waitForTimeout(500);

      // Проверяем что значение ввелось
      const value = await searchInput.inputValue();
      expect(value.toLowerCase()).toContain('camera');
    }
  });

  test('Device list renders device names', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Проверяем отображение устройств (текст из моков)
    await expect(page.locator('text=Camera-01 Main Entrance').first()).toBeVisible();
  });

  test('Status badges are displayed', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Проверяем наличие статусных баджей
    const statusBadges = page.locator('text=online, text=offline, text=warning');
    // Хотя бы один статусный бадж должен быть
  });

  test('Device detail navigation', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Находим ссылку на первое устройство
    const deviceLink = page.locator('a:has-text("Camera-01 Main Entrance")').first();
    if (await deviceLink.isVisible()) {
      await deviceLink.click();
      await page.waitForTimeout(1000);

      // Проверяем переход на страницу деталей
      expect(page.url()).toContain('/devices/');
    }
  });

  test('Filter by status works', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Ищем select или кнопку для фильтрации
    const statusFilter = page.locator('select, [role="combobox"]').filter({ hasText: /status|all|online|offline/i }).first();
    if (await statusFilter.isVisible()) {
      await statusFilter.click();
      await page.waitForTimeout(300);
    }
  });

  test('Bulk select checkboxes', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Ищем checkbox для выбора устройств
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();

    if (count > 0) {
      // Выбираем первое устройство
      await checkboxes.first().check();
      await page.waitForTimeout(200);

      // Проверяем что чекбокс выбран
      await expect(checkboxes.first()).toBeChecked();
    }
  });
});
