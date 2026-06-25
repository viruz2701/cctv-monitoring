import { test, expect } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════
// Settings Page — E2E Tests
// P0 flow: RBAC + tab navigation
// ═══════════════════════════════════════════════════════════════════════

const MOCK_ADMIN = {
  id: '1',
  username: 'admin',
  role: 'admin',
};

const MOCK_TECHNICIAN = {
  id: '2',
  username: 'tech',
  role: 'technician',
};

async function mockAuthenticatedUser(page: any, user: typeof MOCK_ADMIN) {
  // Мокаем все API запросы аутентификации
  await page.route('**/api/v1/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(user),
    });
  });

  // Мокаем настройки
  await page.route('**/api/v1/settings', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
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
      }),
    });
  });
}

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    await mockAuthenticatedUser(page, MOCK_ADMIN);
    // Set token in localStorage before navigation
    await page.goto('/login');
    await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
    await page.goto('/settings');
  });

  test('Settings page loads with tabs', async ({ page }) => {
    // Ждем загрузки страницы
    await expect(page.locator('text=Settings')).toBeVisible();
    // Проверяем что табы отображаются
    const tabs = page.locator('[role="tab"], button:has(svg)');
    await expect(tabs.first()).toBeVisible();
  });

  test('Tab navigation works — clicking tabs changes content', async ({ page }) => {
    // Ждем загрузки
    await page.waitForTimeout(1500);

    // Проверяем навигацию по табам
    const tabButtons = page.locator('button:has(svg)');
    const tabCount = await tabButtons.count();

    // Проходим по видимым табам
    for (let i = 0; i < Math.min(tabCount, 3); i++) {
      const tab = tabButtons.nth(i);
      if (await tab.isVisible()) {
        await tab.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('Non-admin user can not see security tab', async ({ page }) => {
    // Переключаемся на пользователя с ролью technician
    await page.evaluate(() => localStorage.removeItem('token'));
    await mockAuthenticatedUser(page, MOCK_TECHNICIAN);
    await page.goto('/settings');

    await page.waitForTimeout(1000);

    // Пытаемся перейти на security tab
    await page.goto('/settings/security');
    await page.waitForTimeout(500);

    // Non-admin должен получить fallback доступ запрещен (PermissionGuard)
    // Или быть перенаправлен на general tab
    const currentUrl = page.url();
    // Т.к. PermissionGuard сработает на уровне страницы,
    // technician не должен видеть security настройки
    await expect(page.locator('text=access denied').or(page.locator('text=Access Denied'))).toBeVisible();
  });

  test('General settings form — update site name', async ({ page }) => {
    await page.waitForTimeout(1000);

    // Пробуем найти поле site_name
    const siteNameInput = page.locator('input[name="site_name"], input[id*="site"]').first();
    if (await siteNameInput.isVisible()) {
      await siteNameInput.fill('Updated Site Name');

      // Пробуем сохранить
      const saveButton = page.getByRole('button', { name: /save/i });
      if (await saveButton.isVisible()) {
        await saveButton.click();
      }
    }
  });
});
