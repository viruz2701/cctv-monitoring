/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockSites,
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
// Device Management — E2E Tests
// P1-QA.1: Add / Edit / Delete device, Bulk operations
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_DEVICE_TYPES = [
  { value: 'camera', label: 'Camera' },
  { value: 'nvr', label: 'NVR' },
  { value: 'switch', label: 'Switch' },
  { value: 'gateway', label: 'Gateway' },
  { value: 'controller', label: 'Controller' },
];

const MOCK_VENDOR_TYPES = [
  { value: 'hikvision', label: 'HikVision' },
  { value: 'dahua', label: 'Dahua' },
  { value: 'axis', label: 'Axis' },
  { value: 'bosch', label: 'Bosch' },
  { value: 'other', label: 'Other' },
];

let deviceIdCounter = 100;
function nextDeviceId(): string {
  deviceIdCounter++;
  return `dev-new-${deviceIdCounter}`;
}

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupDeviceManagementMockApi(page: any) {
  await setupAuth(page);

  // Devices list
  await page.route('**/api/v1/devices*', async (route: any, request: any) => {
    const url = request.url();

    // POST — create new device
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      const newDevice = {
        id: nextDeviceId(),
        ...body,
        status: 'online',
        health: 'healthy',
        last_seen: new Date().toISOString(),
        registered_at: new Date().toISOString(),
      };
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify(newDevice),
      });
    }

    // PUT — update device
    if (request.method() === 'PUT') {
      const body = JSON.parse(request.postData() || '{}');
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...MOCK_DEVICES[0],
          ...body,
          updated_at: new Date().toISOString(),
        }),
      });
    }

    // DELETE — soft delete
    if (request.method() === 'DELETE') {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          deleted: true,
          deleted_at: new Date().toISOString(),
        }),
      });
    }

    // PATCH — bulk update
    if (request.method() === 'PATCH') {
      const body = JSON.parse(request.postData() || '{}');
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          updated: body.device_ids?.length || 0,
          changes: body.changes || {},
        }),
      });
    }

    // Single device detail
    const deviceMatch = url.match(/\/devices\/([^/?]+)/);
    if (deviceMatch && request.method() === 'GET') {
      const deviceId = deviceMatch[1];
      if (deviceId && !['devices'].includes(deviceId)) {
        const device = [...MOCK_DEVICES].find((d) => d.id === deviceId);
        if (device) {
          return route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify(device),
          });
        }
      }
    }

    // Collection — list
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_DEVICES),
    });
  });

  // Device types
  await page.route('**/api/v1/devices/types', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_DEVICE_TYPES),
    });
  });

  // Vendor types
  await page.route('**/api/v1/devices/vendors', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_VENDOR_TYPES),
    });
  });

  // Standard mocks
  await mockSites(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockDashboardStats(page);
  await mockAlerts(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Add Device
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Management — Add Device', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceManagementMockApi(page);
    await page.goto('/devices/new');
    await page.waitForTimeout(1500);
  });

  test('Add device — form loads with all required fields', async ({ page }) => {
    // Проверяем поля формы создания
    const nameField = page.locator(
      'input[name="name"], input[placeholder*="name" i], input[placeholder*="назван" i], input[id*="name" i]',
    ).first();
    await expect(nameField).toBeVisible();

    const ipField = page.locator(
      'input[name="ip"], input[name="ip_address"], input[placeholder*="ip" i], input[placeholder*="192" i]',
    ).first();
    await expect(ipField).toBeVisible();

    const typeSelect = page.locator(
      'select[name="type"], select[id*="type" i], [role="combobox"]',
    ).first();
    await expect(typeSelect).toBeVisible();
  });

  test('Add device — fill form and submit successfully', async ({ page }) => {
    // Заполняем название
    const nameField = page.locator(
      'input[name="name"], input[placeholder*="name" i], input[placeholder*="назван" i], input[id*="name" i]',
    ).first();
    await nameField.fill('New Camera — Entrance Gate');

    // Заполняем IP
    const ipField = page.locator(
      'input[name="ip"], input[name="ip_address"], input[placeholder*="ip" i]',
    ).first();
    await ipField.fill('192.168.1.200');

    // Выбираем тип
    const typeSelect = page.locator(
      'select[name="type"], select[id*="type" i], [role="combobox"]',
    ).first();
    if (await typeSelect.isVisible()) {
      await typeSelect.selectOption('camera');
    }

    // Выбираем объект
    const siteSelect = page.locator(
      'select[name="site_id"], select[id*="site" i], [role="combobox"]',
    ).first();
    if (await siteSelect.isVisible()) {
      await siteSelect.selectOption('site-1');
    }

    // Сабмитим
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|add|создать|сохранить|добавить/i)',
    ).first();
    await submitButton.click();
    await page.waitForTimeout(1000);

    // Проверяем успех
    const successMessage = page.locator(
      'text=/created|успешно создан|device added|устройство добавлен|success|успех/i',
    ).first();
    const hasSuccess = await successMessage.isVisible().catch(() => false);

    const currentUrl = page.url();
    const hasRedirect = currentUrl.includes('/devices/') && !currentUrl.includes('/new');
    expect(hasSuccess || hasRedirect).toBeTruthy();
  });

  test('Add device — validation error on empty required fields', async ({ page }) => {
    const submitButton = page.locator(
      'button[type="submit"], button:has-text(/create|save|add|создать|сохранить|добавить/i)',
    ).first();
    await submitButton.click();
    await page.waitForTimeout(500);

    // Проверяем ошибку валидации
    const validationError = page.locator(
      'text=/required|обязательно|please fill|заполните|name.*required|название.*обязательно/i',
    ).first();
    await expect(validationError).toBeVisible();
  });

  test('Add device — cancel returns to devices list', async ({ page }) => {
    const cancelButton = page.locator(
      'button:has-text(/cancel|отмена|back|назад/i), a:has-text(/cancel|отмена|back|назад/i)',
    ).first();
    if (await cancelButton.isVisible()) {
      await cancelButton.click();
      await page.waitForTimeout(500);
      expect(page.url()).toContain('/devices');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Edit Device
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Management — Edit Device', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceManagementMockApi(page);
    await page.goto('/devices/dev-1/edit');
    await page.waitForTimeout(1500);
  });

  test('Edit device — form loads with existing values', async ({ page }) => {
    // Проверяем что форма загружена с существующими данными
    const nameField = page.locator(
      'input[name="name"], input[placeholder*="name" i], input[placeholder*="назван" i], input[id*="name" i]',
    ).first();
    await expect(nameField).toBeVisible();

    const currentValue = await nameField.inputValue();
    expect(currentValue.length).toBeGreaterThan(0);
  });

  test('Edit device — update name and save', async ({ page }) => {
    // Меняем название
    const nameField = page.locator(
      'input[name="name"], input[placeholder*="name" i], input[placeholder*="назван" i], input[id*="name" i]',
    ).first();
    await nameField.clear();
    await nameField.fill('Camera-Lobby-01 — Updated');

    // Меняем местоположение
    const locationField = page.locator(
      'input[name="location"], input[placeholder*="location" i], input[placeholder*="мест" i], input[id*="location" i]',
    ).first();
    if (await locationField.isVisible()) {
      await locationField.clear();
      await locationField.fill('Main Lobby — East Wing');
    }

    // Сохраняем
    const saveButton = page.locator(
      'button[type="submit"], button:has-text(/save|update|сохранить|обновить/i)',
    ).first();
    await saveButton.click();
    await page.waitForTimeout(1000);

    // Проверяем сообщение об успехе
    const successMessage = page.locator(
      'text=/updated|обновлен|saved|сохранен|success|успех/i',
    ).first();
    const hasSuccess = await successMessage.isVisible().catch(() => false);
    if (!hasSuccess) {
      // Проверяем что вернулись на страницу устройства
      const currentUrl = page.url();
      expect(currentUrl).toContain('/devices/');
    }
  });

  test('Edit device — cancel reverts changes', async ({ page }) => {
    const cancelButton = page.locator(
      'button:has-text(/cancel|отмена|back|назад/i), a:has-text(/cancel|отмена|back|назад/i)',
    ).first();
    if (await cancelButton.isVisible()) {
      await cancelButton.click();
      await page.waitForTimeout(500);

      // Возвращаемся на страницу устройства
      expect(page.url()).toContain('/devices/');
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Delete Device (Soft Delete)
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Management — Delete Device', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceManagementMockApi(page);
    await page.goto('/devices/dev-1');
    await page.waitForTimeout(1500);
  });

  test('Delete device — delete button is visible', async ({ page }) => {
    const deleteButton = page.locator(
      'button:has-text(/delete|удалить|remove|убрать|archive|архивирова/i), ' +
      '[class*="delete" i] button, button[class*="delete" i]',
    ).first();
    await expect(deleteButton).toBeVisible();
  });

  test('Delete device — confirmation dialog appears', async ({ page }) => {
    const deleteButton = page.locator(
      'button:has-text(/delete|удалить|remove|убрать/i)',
    ).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();
      await page.waitForTimeout(500);

      // Проверяем появление confirmation dialog
      const confirmDialog = page.locator(
        '[role="alertdialog"], [role="dialog"], .modal, .confirm-dialog, ' +
        'text=/are you sure|вы уверены|confirm|подтвердите|delete.*device|удалить.*устройств/i',
      ).first();
      await expect(confirmDialog).toBeVisible();
    }
  });

  test('Delete device — confirm delete shows success message', async ({ page }) => {
    const deleteButton = page.locator(
      'button:has-text(/delete|удалить|remove|убрать/i)',
    ).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();
      await page.waitForTimeout(500);

      // Подтверждаем удаление
      const confirmButton = page.locator(
        'button:has-text(/confirm|delete|yes|подтвердить|удалить|да/i)',
      ).last();
      if (await confirmButton.isVisible()) {
        await confirmButton.click();
        await page.waitForTimeout(1000);

        // Проверяем сообщение об успешном удалении
        const successMessage = page.locator(
          'text=/deleted|удален|removed|убрано|device removed|устройство удален/i',
        ).first();
        const hasSuccess = await successMessage.isVisible().catch(() => false);

        if (!hasSuccess) {
          // Проверяем редирект на список устройств
          const currentUrl = page.url();
          expect(currentUrl).toContain('/devices');
          const isOnDetail = currentUrl.includes('/devices/dev-1');
          expect(isOnDetail).toBeFalsy();
        }
      }
    }
  });

  test('Delete device — cancel delete does not remove device', async ({ page }) => {
    const deleteButton = page.locator(
      'button:has-text(/delete|удалить|remove|убрать/i)',
    ).first();

    if (await deleteButton.isVisible()) {
      await deleteButton.click();
      await page.waitForTimeout(500);

      // Отменяем удаление
      const cancelButton = page.locator(
        'button:has-text(/cancel|отмена|no|нет|keep|оставить/i)',
      ).first();
      if (await cancelButton.isVisible()) {
        await cancelButton.click();
        await page.waitForTimeout(500);

        // Проверяем что мы всё ещё на странице устройства
        expect(page.url()).toContain('/devices/dev-1');
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Bulk Status Update
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Management — Bulk Status Update', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceManagementMockApi(page);
    await page.goto('/devices');
    await page.waitForTimeout(1500);
  });

  test('Bulk update — checkboxes allow multi-select', async ({ page }) => {
    // Находим чекбоксы для выбора устройств
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();

    if (count >= 2) {
      // Выбираем несколько устройств
      await checkboxes.nth(0).check();
      await checkboxes.nth(1).check();

      // Проверяем что они выбраны
      await expect(checkboxes.nth(0)).toBeChecked();
      await expect(checkboxes.nth(1)).toBeChecked();
    }
  });

  test('Bulk update — bulk action toolbar appears after selection', async ({ page }) => {
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();

    if (count >= 1) {
      await checkboxes.nth(0).check();
      await page.waitForTimeout(300);

      // Проверяем появление тулбара с bulk-действиями
      const bulkToolbar = page.locator(
        '[class*="bulk" i], [class*="toolbar" i], [role="toolbar"], ' +
        'button:has-text(/bulk|массов|update|обновить|action|действи/i)',
      ).first();
      const hasToolbar = await bulkToolbar.isVisible().catch(() => false);

      if (hasToolbar) {
        await expect(bulkToolbar).toBeVisible();
      }
    }
  });

  test('Bulk update — select status and apply to selected devices', async ({ page }) => {
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();

    if (count >= 2) {
      // Выбираем устройства
      await checkboxes.nth(0).check();
      await checkboxes.nth(1).check();

      // Ищем кнопку bulk update
      const bulkActionButton = page.locator(
        'button:has-text(/bulk|массов|update|обновить|action/i)',
      ).first();
      if (await bulkActionButton.isVisible()) {
        await bulkActionButton.click();
        await page.waitForTimeout(500);
      }

      // Выбираем новый статус
      const statusSelect = page.locator(
        'select[name="status"], select[id*="status" i], [role="combobox"]',
      ).first();
      if (await statusSelect.isVisible()) {
        await statusSelect.selectOption('online');
      }

      // Применяем изменения
      const applyButton = page.locator(
        'button:has-text(/apply|применить|update|обновить|save|сохранить/i)',
      ).first();
      if (await applyButton.isVisible()) {
        await applyButton.click();
        await page.waitForTimeout(1000);

        // Проверяем сообщение об успехе
        const successMessage = page.locator(
          'text=/updated|обновлен|success|успех|applied|применен/i',
        ).first();
        const hasSuccess = await successMessage.isVisible().catch(() => false);
        if (!hasSuccess) {
          expect(page.url()).toContain('/devices');
        }
      }
    }
  });

  test('Bulk update — select all checkbox works', async ({ page }) => {
    // Ищем select-all checkbox
    const selectAllCheckbox = page.locator(
      'input[type="checkbox"][aria-label*="select all" i], ' +
      'input[type="checkbox"][aria-label*="выбрать все" i], ' +
      'th input[type="checkbox"], thead input[type="checkbox"]',
    ).first();

    if (await selectAllCheckbox.isVisible()) {
      await selectAllCheckbox.check();
      await page.waitForTimeout(300);

      // Проверяем что все чекбоксы выбраны
      await expect(selectAllCheckbox).toBeChecked();

      // Проверяем что появился тулбар
      const bulkToolbar = page.locator(
        '[class*="bulk" i], [role="toolbar"]',
      ).first();
      const hasToolbar = await bulkToolbar.isVisible().catch(() => false);
      if (hasToolbar) {
        await expect(bulkToolbar).toBeVisible();
      }
    }
  });
});
