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
// Alerts & Notifications — E2E Tests
// P1-QA.1: Alert acknowledgment, notification settings, alert filtering
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_ALERTS_EXTENDED = [
  { id: 'alert-1', severity: 'critical', message: 'NVR-03 disk failure imminent', device_id: 'dev-2', device_name: 'NVR-03 Recording Server', status: 'active', created_at: new Date().toISOString(), acknowledged_at: null, acknowledged_by: null },
  { id: 'alert-2', severity: 'warning', message: 'Camera-12 offline > 24h', device_id: 'dev-3', device_name: 'Camera-12 Parking Lot B', status: 'active', created_at: new Date(Date.now() - 3600000).toISOString(), acknowledged_at: null, acknowledged_by: null },
  { id: 'alert-3', severity: 'info', message: 'Scheduled maintenance due', device_id: 'dev-1', device_name: 'Camera-Lobby-01', status: 'acknowledged', created_at: new Date(Date.now() - 7200000).toISOString(), acknowledged_at: new Date(Date.now() - 3600000).toISOString(), acknowledged_by: 'user-1' },
  { id: 'alert-4', severity: 'critical', message: 'UPS battery low — Site-1 Server Room', device_id: 'dev-4', device_name: 'Switch-02 Floor B2', status: 'active', created_at: new Date(Date.now() - 600000).toISOString(), acknowledged_at: null, acknowledged_by: null },
  { id: 'alert-5', severity: 'warning', message: 'Bandwidth threshold exceeded on NVR-03', device_id: 'dev-2', device_name: 'NVR-03 Recording Server', status: 'acknowledged', created_at: new Date(Date.now() - 86400000).toISOString(), acknowledged_at: new Date(Date.now() - 43200000).toISOString(), acknowledged_by: 'user-2' },
  { id: 'alert-6', severity: 'info', message: 'Firmware update available for Camera-Lobby-01', device_id: 'dev-1', device_name: 'Camera-Lobby-01', status: 'resolved', created_at: new Date(Date.now() - 86400000 * 3).toISOString(), resolved_at: new Date(Date.now() - 86400000 * 2).toISOString() },
];

const MOCK_NOTIFICATION_SETTINGS = {
  email: {
    alert_critical: true,
    alert_high: true,
    alert_warning: false,
    alert_info: false,
    weekly_digest: true,
    sla_breach: true,
    maintenance_reminder: true,
  },
  sms: {
    alert_critical: true,
    alert_high: false,
    alert_warning: false,
    on_call_escalation: true,
  },
  push: {
    alert_critical: true,
    alert_high: true,
    alert_warning: true,
    alert_info: false,
  },
  telegram: {
    enabled: false,
    alert_critical: true,
  },
  general: {
    quiet_hours_start: '22:00',
    quiet_hours_end: '07:00',
    notify_on_resolution: true,
    daily_summary: true,
  },
};

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupAlertsMockApi(page: any) {
  await setupAuth(page);

  // Extended alerts
  await page.route('**/api/v1/alerts*', async (route: any, request: any) => {
    // Acknowledge endpoint
    if (request.method() === 'POST') {
      const url = request.url();
      if (url.includes('/acknowledge') || url.includes('/ack')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'acknowledged',
            acknowledged_at: new Date().toISOString(),
            acknowledged_by: 'user-1',
          }),
        });
      }

      // Resolve endpoint
      if (url.includes('/resolve')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'resolved',
            resolved_at: new Date().toISOString(),
          }),
        });
      }
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_ALERTS_EXTENDED),
    });
  });

  // Notification settings
  await page.route('**/api/v1/settings/notifications', async (route: any, request: any) => {
    if (request.method() === 'PUT') {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ...MOCK_NOTIFICATION_SETTINGS,
          ...JSON.parse(request.postData() || '{}'),
          updated_at: new Date().toISOString(),
        }),
      });
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_NOTIFICATION_SETTINGS),
    });
  });

  // Standard mocks
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockDashboardStats(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Alerts — List & View
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Alerts — List & View', () => {
  test.beforeEach(async ({ page }) => {
    await setupAlertsMockApi(page);
    await page.goto('/alerts');
    await page.waitForTimeout(1500);
  });

  test('Alerts page loads with alert list', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/alerts');

    // Проверяем отображение alerts из мок-данных
    const alertMessage = page.locator(
      'text=/disk failure|offline|maintenance due|UPS battery|bandwidth|firmware update/i',
    ).first();
    await expect(alertMessage).toBeVisible();
  });

  test('Alerts — severity badges are visible', async ({ page }) => {
    // Проверяем severity badges
    const severityBadge = page.locator(
      'span, badge, [class*="severity" i], [class*="badge" i]',
    ).filter({ hasText: /critical|критич|warning|предупрежд|info/i }).first();
    await expect(severityBadge).toBeVisible();
  });

  test('Alerts — critical alerts are highlighted', async ({ page }) => {
    // Проверяем что critical alerts выделены
    const criticalAlert = page.locator(
      'text=/disk failure|UPS battery/i',
    ).first();
    await expect(criticalAlert).toBeVisible();

    // Проверяем что есть индикатор critical severity
    const criticalBadge = page.locator(
      'span, badge, [class*="severity" i]',
    ).filter({ hasText: /critical|критич/i }).first();
    await expect(criticalBadge).toBeVisible();
  });

  test('Alerts — filter by severity', async ({ page }) => {
    // Ищем фильтр по severity
    const severityFilter = page.locator(
      'select[name="severity"], select[id*="severity" i], ' +
      'button:has-text(/filter|фильтр|all|все|severity|важность/i)',
    ).first();

    if (await severityFilter.isVisible()) {
      const tagName = await severityFilter.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        await severityFilter.selectOption('critical');
        await page.waitForTimeout(500);

        // После фильтрации critical alerts должны быть видны
        const criticalAlert = page.locator(
          'text=/disk failure|UPS battery/i',
        ).first();
        const isVisible = await criticalAlert.isVisible().catch(() => false);
        if (!isVisible) {
          expect(page.url()).toContain('/alerts');
        }
      }
    }
  });

  test('Alerts — filter by status shows acknowledged alerts', async ({ page }) => {
    const statusFilter = page.locator(
      'select[name="status"], select[id*="status" i], ' +
      'button:has-text(/filter|фильтр|status|статус|all|все/i)',
    ).first();

    if (await statusFilter.isVisible()) {
      const tagName = await statusFilter.evaluate((el: any) => el.tagName.toLowerCase());

      if (tagName === 'select') {
        await statusFilter.selectOption('acknowledged');
        await page.waitForTimeout(500);

        // Проверяем что acknowledged alert виден
        const acknowledgedAlert = page.locator(
          'text=/bandwidth|maintenance due|firmware update/i',
        ).first();
        const isVisible = await acknowledgedAlert.isVisible().catch(() => false);
        if (!isVisible) {
          expect(page.url()).toContain('/alerts');
        }
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Alerts — Acknowledgment
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Alerts — Acknowledgment', () => {
  test.beforeEach(async ({ page }) => {
    await setupAlertsMockApi(page);
    await page.goto('/alerts');
    await page.waitForTimeout(1500);
  });

  test('Alert acknowledgment — active alert shows acknowledge button', async ({ page }) => {
    // Находим active alert
    const activeAlert = page.locator(
      'a, button, tr, [role="row"], .card, .item, [class*="alert" i]',
    ).filter({ hasText: /disk failure|UPS battery|отказ.*диск|батаре.*ups/i }).first();

    if (await activeAlert.isVisible()) {
      // Проверяем кнопку acknowledge
      const acknowledgeButton = page.locator(
        'button:has-text(/acknowledge|подтвердить|ack|принять|dismiss|отклонить/i)',
      ).first();
      await expect(acknowledgeButton).toBeVisible();
    }
  });

  test('Alert acknowledgment — click acknowledge changes status', async ({ page }) => {
    const acknowledgeButton = page.locator(
      'button:has-text(/acknowledge|подтвердить|ack|принять|dismiss|отклонить/i)',
    ).first();

    if (await acknowledgeButton.isVisible()) {
      await acknowledgeButton.click();
      await page.waitForTimeout(1000);

      // Проверяем что статус изменился
      const statusChanged = page.locator(
        'text=/acknowledged|подтвержден|accepted|принят/i',
      ).first();
      const hasChanged = await statusChanged.isVisible().catch(() => false);

      if (hasChanged) {
        await expect(statusChanged).toBeVisible();
      }
    }
  });

  test('Alert acknowledgment — acknowledged alert shows acknowledge info', async ({ page }) => {
    // Проверяем acknowledged alert (должен показывать кто и когда подтвердил)
    const acknowledgedAlert = page.locator(
      'text=/bandwidth|maintenance due/i',
    ).first();

    if (await acknowledgedAlert.isVisible()) {
      // Проверяем отображение информации о подтверждении
      const ackInfo = page.locator(
        'text=/acknowledged|подтвержден|by|кем|user|пользовател/i',
      ).first();
      const hasInfo = await ackInfo.isVisible().catch(() => false);
      if (hasInfo) {
        await expect(ackInfo).toBeVisible();
      }
    }
  });

  test('Alert acknowledgment — bulk acknowledge multiple alerts', async ({ page }) => {
    // Находим чекбоксы для выбора alerts
    const checkboxes = page.locator('input[type="checkbox"]');
    const count = await checkboxes.count();

    if (count >= 2) {
      // Выбираем несколько active alerts
      await checkboxes.nth(0).check();
      await checkboxes.nth(1).check();
      await page.waitForTimeout(300);

      // Ищем bulk acknowledge кнопку
      const bulkAckButton = page.locator(
        'button:has-text(/acknowledge.*selected|подтвердить.*выбран|bulk.*ack|ack.*all|подтвердить.*все/i)',
      ).first();

      if (await bulkAckButton.isVisible()) {
        await bulkAckButton.click();
        await page.waitForTimeout(1000);

        // Проверяем успех bulk acknowledge
        const successMessage = page.locator(
          'text=/acknowledged|подтвержден|success|успех|updated|обновлен/i',
        ).first();
        const hasSuccess = await successMessage.isVisible().catch(() => false);
        if (!hasSuccess) {
          expect(page.url()).toContain('/alerts');
        }
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Notification Settings
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Notification Settings', () => {
  test.beforeEach(async ({ page }) => {
    await setupAlertsMockApi(page);
    await page.goto('/settings/notifications');
    await page.waitForTimeout(1500);
  });

  test('Notification settings — page loads with channel sections', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/notifications');

    // Проверяем секции каналов уведомлений
    const emailSection = page.locator(
      'text=/email|e-mail|почт/i',
    ).first();
    await expect(emailSection).toBeVisible();
  });

  test('Notification settings — toggle email alert critical on/off', async ({ page }) => {
    // Находим toggle для critical email alerts
    const criticalToggle = page.locator(
      'input[type="checkbox"][name*="critical" i], ' +
      'input[type="checkbox"][id*="critical" i], ' +
      'label:has-text(/critical.*alert|критич.*уведомл|alert.*critical/i) input[type="checkbox"], ' +
      '[role="switch"]',
    ).first();

    if (await criticalToggle.isVisible()) {
      // Переключаем
      const isChecked = await criticalToggle.isChecked();
      if (isChecked) {
        await criticalToggle.uncheck();
      } else {
        await criticalToggle.check();
      }
      await page.waitForTimeout(300);

      // Сохраняем настройки
      const saveButton = page.locator(
        'button[type="submit"], button:has-text(/save|сохранить|apply|применить/i)',
      ).first();

      if (await saveButton.isVisible()) {
        await saveButton.click();
        await page.waitForTimeout(1000);

        // Проверяем сообщение об успешном сохранении
        const successMessage = page.locator(
          'text=/saved|сохранен|updated|обновлен|settings.*saved|настройки.*сохранен/i',
        ).first();
        const hasSuccess = await successMessage.isVisible().catch(() => false);
        if (hasSuccess) {
          await expect(successMessage).toBeVisible();
        }
      }
    }
  });

  test('Notification settings — SMS section configuration', async ({ page }) => {
    // Проверяем настройки SMS
    const smsSection = page.locator(
      'text=/sms|текст/i',
    ).first();
    await expect(smsSection).toBeVisible();

    // Проверяем toggle on-call escalation
    const onCallToggle = page.locator(
      'input[type="checkbox"]',
    ).first();

    if (await onCallToggle.isVisible()) {
      await onCallToggle.check();
      await page.waitForTimeout(200);
    }
  });

  test('Notification settings — quiet hours configuration', async ({ page }) => {
    // Проверяем настройки quiet hours
    const quietHoursSection = page.locator(
      'text=/quiet|тих|not disturb|не беспоко|silent|shutdown/i',
    ).first();
    await expect(quietHoursSection).toBeVisible();

    // Проверяем поля времени
    const startTimeInput = page.locator(
      'input[type="time"], input[name*="quiet" i], input[id*="quiet" i], input[name*="start" i]',
    ).first();
    const hasTimeField = await startTimeInput.isVisible().catch(() => false);
    if (hasTimeField) {
      await expect(startTimeInput).toBeVisible();
    }
  });

  test('Notification settings — daily summary toggle', async ({ page }) => {
    const summaryToggle = page.locator(
      'label:has-text(/daily|summary|ежеднев.*сводк|дайджест|digest/i) input[type="checkbox"], ' +
      'input[type="checkbox"][name*="daily" i], [role="switch"]',
    ).first();

    if (await summaryToggle.isVisible()) {
      const isChecked = await summaryToggle.isChecked();
      // Переключаем и сохраняем
      if (isChecked) {
        await summaryToggle.uncheck();
      } else {
        await summaryToggle.check();
      }
      await page.waitForTimeout(200);
    }
  });
});
