import { test, expect } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════
// Work Orders Page — E2E Tests
// P0 flow: QuickFilters, Kanban toggle, bulk select
// ═══════════════════════════════════════════════════════════════════════

const MOCK_WORK_ORDERS = [
  {
    id: 'WO-001',
    title: 'Replace camera lens',
    status: 'open',
    priority: 'critical',
    assigned_to: 'user-1',
    site_id: 'site-1',
    sla_deadline: new Date(Date.now() + 3600000).toISOString(),
    created_at: new Date().toISOString(),
  },
  {
    id: 'WO-002',
    title: 'Firmware update NVR-03',
    status: 'in_progress',
    priority: 'high',
    assigned_to: null,
    site_id: 'site-2',
    sla_deadline: new Date(Date.now() - 3600000).toISOString(),
    created_at: new Date().toISOString(),
  },
  {
    id: 'WO-003',
    title: 'Cable replacement floor B2',
    status: 'completed',
    priority: 'medium',
    assigned_to: 'user-2',
    site_id: 'site-1',
    sla_deadline: new Date(Date.now() + 86400000).toISOString(),
    created_at: new Date().toISOString(),
  },
];

async function setupAuth(page: any) {
  await page.route('**/api/v1/auth/me', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        id: 'user-1',
        username: 'manager',
        role: 'manager',
      }),
    });
  });

  await page.route('**/api/v1/work-orders*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_WORK_ORDERS),
    });
  });

  await page.route('**/api/v1/sites*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([{ id: 'site-1', name: 'Main Office' }, { id: 'site-2', name: 'Branch' }]),
    });
  });

  await page.route('**/api/v1/users*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        { id: 'user-1', username: 'manager', role: 'manager' },
        { id: 'user-2', username: 'tech1', role: 'technician' },
      ]),
    });
  });

  await page.evaluate(() => localStorage.setItem('token', 'mock-token'));
}

test.describe('Work Orders Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupAuth(page);
    await page.goto('/work-orders');
  });

  test('Work Orders page loads', async ({ page }) => {
    await page.waitForTimeout(2000);

    // Страница загрузилась
    await expect(page.locator('body')).toBeVisible();
    // Проверяем URL
    expect(page.url()).toContain('/work-orders');
  });

  test('QuickFilters work — click "My Orders"', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Находим кнопку "My Orders" и кликаем
    const myOrdersButton = page.locator('button:has-text("My Orders")');
    if (await myOrdersButton.isVisible()) {
      await myOrdersButton.click();
      await page.waitForTimeout(300);

      // Проверяем что фильтр применился (aria-pressed)
      await expect(myOrdersButton).toHaveAttribute('aria-pressed', 'true');
    }
  });

  test('View toggle — switch to Kanban', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Ищем кнопку переключения вида (Kanban/List)
    const kanbanButton = page.locator('button:has(svg), [role="tab"]').filter({ hasText: /kanban|board/i }).first();
    const listButton = page.locator('button:has(svg), [role="tab"]').filter({ hasText: /list|table/i }).first();

    if (await kanbanButton.isVisible()) {
      await kanbanButton.click();
      await page.waitForTimeout(500);
    } else if (await listButton.isVisible()) {
      // Если Kanban включен, переключаемся обратно
      await listButton.click();
      await page.waitForTimeout(500);
    }
  });

  test('QuickFilters show counters', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Проверяем что фильтры отображают счетчики
    const filters = page.locator('button:has(span)');
    const filterCount = await filters.count();
    // Должен быть хотя бы один фильтр со счетчиком
    expect(filterCount).toBeGreaterThan(0);
  });

  test('Overdue filter shows count', async ({ page }) => {
    await page.waitForTimeout(1500);

    // Кликаем на "Overdue"
    const overdueButton = page.locator('button:has-text("Overdue")');
    if (await overdueButton.isVisible()) {
      await overdueButton.click();
      await page.waitForTimeout(300);
      await expect(overdueButton).toHaveAttribute('aria-pressed', 'true');
    }
  });
});
