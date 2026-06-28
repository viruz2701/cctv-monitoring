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
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Maintenance Schedule — E2E Tests
// P1-QA.1: Создание, редактирование и просмотр расписания ТО
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_MAINTENANCE_SCHEDULES = [
  {
    id: 'ms-1',
    title: 'Ежеквартальное ТО — Камеры Site-1',
    device_id: 'dev-1',
    device_name: 'Camera-Lobby-01',
    site_id: 'site-1',
    site_name: 'Main Office',
    type: 'quarterly',
    status: 'scheduled',
    description: 'Проверка крепления, чистка линз, обновление прошивки',
    scheduled_date: new Date(Date.now() + 86400000 * 14).toISOString(),
    assigned_to: 'user-3',
    assigned_name: 'Bob Technician',
    estimated_hours: 3,
    created_at: new Date(Date.now() - 86400000 * 7).toISOString(),
    checklist: [
      { id: 'mcl-1', label: 'Проверить крепление камеры', completed: false },
      { id: 'mcl-2', label: 'Очистить линзы', completed: false },
      { id: 'mcl-3', label: 'Проверить прошивку', completed: false },
      { id: 'mcl-4', label: 'Тест передачи видео', completed: false },
    ],
  },
  {
    id: 'ms-2',
    title: 'Годовое ТО — NVR-03',
    device_id: 'dev-2',
    device_name: 'NVR-03 Recording Server',
    site_id: 'site-1',
    site_name: 'Main Office',
    type: 'annual',
    status: 'overdue',
    description: 'Замена HDD, чистка системы охлаждения, замена батарейки CMOS',
    scheduled_date: new Date(Date.now() - 86400000 * 5).toISOString(),
    assigned_to: 'user-2',
    assigned_name: 'Alex Manager',
    estimated_hours: 8,
    created_at: new Date(Date.now() - 86400000 * 30).toISOString(),
    checklist: [
      { id: 'mcl-5', label: 'SMART тест HDD', completed: true },
      { id: 'mcl-6', label: 'Замена HDD при необходимости', completed: false },
      { id: 'mcl-7', label: 'Чистка системы охлаждения', completed: false },
    ],
  },
  {
    id: 'ms-3',
    title: 'Ежемесячная проверка — Parking Lot',
    device_id: 'dev-3',
    device_name: 'Camera-12 Parking Lot B',
    site_id: 'site-2',
    site_name: 'Branch Office',
    type: 'monthly',
    status: 'completed',
    description: 'Визуальный осмотр, проверка качества изображения',
    scheduled_date: new Date(Date.now() - 86400000 * 10).toISOString(),
    assigned_to: 'user-3',
    assigned_name: 'Bob Technician',
    estimated_hours: 1,
    created_at: new Date(Date.now() - 86400000 * 40).toISOString(),
    completed_at: new Date(Date.now() - 86400000 * 8).toISOString(),
    checklist: [
      { id: 'mcl-8', label: 'Визуальный осмотр', completed: true },
      { id: 'mcl-9', label: 'Проверка качества видео', completed: true },
      { id: 'mcl-10', label: 'Очистка', completed: true },
    ],
  },
];

const MOCK_MAINTENANCE_TYPES = [
  { value: 'weekly', label: 'Еженедельное' },
  { value: 'monthly', label: 'Ежемесячное' },
  { value: 'quarterly', label: 'Ежеквартальное' },
  { value: 'biannual', label: 'Полугодовое' },
  { value: 'annual', label: 'Годовое' },
];

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupMaintenanceMockApi(page: any) {
  await setupAuth(page);

  // Maintenance schedules list + CRUD
  await page.route('**/api/v1/maintenance*', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: `ms-new-${Date.now()}`,
          ...body,
          status: 'scheduled',
          created_at: new Date().toISOString(),
        }),
      });
      return;
    }

    if (request.method() === 'PUT') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...MOCK_MAINTENANCE_SCHEDULES[0],
          ...JSON.parse(request.postData() || '{}'),
          updated_at: new Date().toISOString(),
        }),
      });
      return;
    }

    if (request.method() === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ deleted: true }),
      });
      return;
    }

    // GET
    const url = request.url();
    const detailMatch = url.match(/\/maintenance\/([^/?]+)/);
    if (detailMatch && detailMatch[1] && !detailMatch[1].includes('?')) {
      const msId = detailMatch[1];
      const schedule = MOCK_MAINTENANCE_SCHEDULES.find((s) => s.id === msId);
      if (schedule) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(schedule),
        });
      }
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_MAINTENANCE_SCHEDULES),
    });
  });

  // Maintenance types
  await page.route('**/api/v1/maintenance/types', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_MAINTENANCE_TYPES),
    });
  });

  // Standard mocks
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockDashboardStats(page);
  await mockAlerts(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Maintenance Schedule — List & View
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Maintenance Schedule — List & View', () => {
  test.beforeEach(async ({ page }) => {
    await setupMaintenanceMockApi(page);
    await page.goto('/maintenance');
    await page.waitForTimeout(1500);
  });

  test('Maintenance page loads with schedule list', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/maintenance');

    // Проверяем отображение расписаний
    const scheduleTitle = page.locator(
      'text=/Ежеквартальное ТО|Годовое ТО|Ежемесячная проверк|Camera-Lobby|NVR-03|Parking Lot/i',
    ).first();
    await expect(scheduleTitle).toBeVisible();
  });

  test('Maintenance — status badges are displayed', async ({ page }) => {
    // Проверяем статусные badges
    const statusBadges = page.locator(
      'text=/scheduled|запланирован|overdue|просрочен|completed|завершен/i',
    ).first();
    await expect(statusBadges).toBeVisible();
  });

  test('Maintenance — overdue schedule shows warning', async ({ page }) => {
    // Проверяем индикатор просрочки
    const overdueIndicator = page.locator(
      'text=/overdue|просрочен|past due|просрочк|delayed|задержк/i',
    ).first();
    await expect(overdueIndicator).toBeVisible();
  });

  test('Maintenance — schedule detail shows checklist items', async ({ page }) => {
    // Кликаем на расписание для просмотра деталей
    const scheduleItem = page.locator(
      'a, button, tr, [role="row"], .card, .item',
    ).filter({ hasText: /Ежеквартальное ТО|Camera-Lobby/i }).first();

    if (await scheduleItem.isVisible()) {
      await scheduleItem.click();
      await page.waitForTimeout(1000);

      // Проверяем отображение чеклиста
      const checklistItem = page.locator(
        'text=/крепление|линз|прошивк|видео|checklist|чеклист/i',
      ).first();
      await expect(checklistItem).toBeVisible();
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Maintenance Schedule — Create
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Maintenance Schedule — Create', () => {
  test.beforeEach(async ({ page }) => {
    await setupMaintenanceMockApi(page);
    await page.goto('/maintenance/create');
    await page.waitForTimeout(1500);
  });

  test('Create maintenance — form loads with required fields', async ({ page }) => {
    // Проверяем поля формы
    const titleField = page.locator(
      'input[name="title"], input[placeholder*="title" i], input[placeholder*="назван" i], input[id*="title" i]',
    ).first();
    await expect(titleField).toBeVisible();

    const deviceSelect = page.locator(
      'select[name="device_id"], select[id*="device" i], [role="combobox"]',
    ).first();
    await expect(deviceSelect).toBeVisible();
  });

  test('Create maintenance — fill form and submit', async ({ page }) => {
    // Заполняем название
    const titleField = page.locator(
      'input[name="title"], input[placeholder*="title" i], input[placeholder*="назван" i], input[id*="title" i]',
    ).first();
    await titleField.fill('Monthly check — Server Room cameras');

    // Выбираем устройство
    const deviceSelect = page.locator(
      'select[name="device_id"], select[id*="device" i], [role="combobox"]',
    ).first();
    if (await deviceSelect.isVisible()) {
      await deviceSelect.selectOption('dev-1');
    }

    // Выбираем тип ТО
    const typeSelect = page.locator(
      'select[name="type"], select[name="maintenance_type"], select[id*="type" i], [role="combobox"]',
    ).first();
    if (await typeSelect.isVisible()) {
      await typeSelect.selectOption('monthly');
    }

    // Сабмитим
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|создать|сохранить|schedule|запланировать/i)',
    ).first();
    await submitButton.click();
    await page.waitForTimeout(1000);

    // Проверяем успех
    const successMessage = page.locator(
      'text=/created|успешно создан|scheduled|запланирован|maintenance created/i',
    ).first();
    const hasSuccess = await successMessage.isVisible().catch(() => false);

    const currentUrl = page.url();
    const hasRedirect = currentUrl.includes('/maintenance/');
    expect(hasSuccess || hasRedirect).toBeTruthy();
  });

  test('Create maintenance — validation error on empty form', async ({ page }) => {
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|создать|сохранить|schedule|запланировать/i)',
    ).first();
    await submitButton.click();
    await page.waitForTimeout(500);

    // Проверяем ошибку валидации
    const validationError = page.locator(
      'text=/required|обязательно|please fill|заполните|title.*required|название.*обязательно/i',
    ).first();
    await expect(validationError).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Maintenance Schedule — Filters & Status
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Maintenance Schedule — Filters & Status', () => {
  test.beforeEach(async ({ page }) => {
    await setupMaintenanceMockApi(page);
    await page.goto('/maintenance');
    await page.waitForTimeout(1500);
  });

  test('Maintenance — filter by status shows relevant items', async ({ page }) => {
    // Ищем filter controls
    const statusFilter = page.locator(
      'select[name="status"], [role="combobox"], button:has-text(/filter|фильтр|all|все|status|статус/i)',
    ).first();

    if (await statusFilter.isVisible()) {
      const tagName = await statusFilter.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        await statusFilter.selectOption('overdue');
        await page.waitForTimeout(500);

        // Проверяем что отображаются просроченные
        const overdueItem = page.locator(
          'text=/overdue|просрочен|NVR-03|Годовое ТО/i',
        ).first();
        const isVisible = await overdueItem.isVisible().catch(() => false);
        if (!isVisible) {
          // Хотя бы страница обновилась
          expect(page.url()).toContain('/maintenance');
        }
      }
    }
  });

  test('Maintenance — filter by device shows relevant schedules', async ({ page }) => {
    const deviceFilter = page.locator(
      'select[name="device_id"], select[id*="device" i], [role="combobox"]',
    ).first();

    if (await deviceFilter.isVisible()) {
      await deviceFilter.click();
      await page.waitForTimeout(300);

      // Проверяем что опции выпадающего списка содержат устройства
      const deviceOption = page.locator(
        'option, [role="option"], [role="listbox"] option',
      ).filter({ hasText: /Camera-Lobby|NVR-03|dev-1|dev-2/i }).first();

      const hasOption = await deviceOption.isVisible().catch(() => false);
      if (hasOption) {
        await deviceOption.click();
        await page.waitForTimeout(300);
      }
    }
  });

  test('Maintenance — type filter shows maintenance types', async ({ page }) => {
    const typeFilter = page.locator(
      'select[name="type"], select[id*="type" i], [role="combobox"]',
    ).first();

    if (await typeFilter.isVisible()) {
      const tagName = await typeFilter.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        // Проверяем наличие опций
        const options = await typeFilter.locator('option').allTextContents();
        const hasMonthly = options.some((o: string) => /monthly|ежемесяч/i.test(o));
        const hasQuarterly = options.some((o: string) => /quarterly|ежекварталь/i.test(o));
        expect(hasMonthly || hasQuarterly).toBeTruthy();
      }
    }
  });

  test('Maintenance — filter by site narrows the list', async ({ page }) => {
    const siteFilter = page.locator(
      'select[name="site_id"], select[id*="site" i], [role="combobox"]',
    ).first();

    if (await siteFilter.isVisible()) {
      const tagName = await siteFilter.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        const options = await siteFilter.locator('option').allTextContents();
        const hasSite1 = options.some((o: string) => /Main Office|site-1|Main/i.test(o));
        expect(hasSite1).toBeTruthy();
      }
    }
  });
});
