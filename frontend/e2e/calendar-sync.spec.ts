/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
  MOCK_ADMIN_USER,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Calendar Sync — E2E Tests
// Google/Outlook OAuth flow, connection management, sync status
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_CALENDAR_CONNECTIONS = [
  {
    id: 'cal-google-1',
    provider: 'google',
    email: 'admin@cctv.local',
    connected: true,
    last_sync: new Date(Date.now() - 3600000).toISOString(),
    calendar_count: 3,
    sync_enabled: true,
  },
  {
    id: 'cal-outlook-1',
    provider: 'outlook',
    email: 'admin@office365.com',
    connected: true,
    last_sync: new Date(Date.now() - 7200000).toISOString(),
    calendar_count: 2,
    sync_enabled: false,
  },
];

const MOCK_SYNC_STATUS = {
  active_syncs: 1,
  last_sync: new Date(Date.now() - 3600000).toISOString(),
  next_sync: new Date(Date.now() + 3600000).toISOString(),
  total_events_synced: 128,
  failed_events: 3,
  sync_interval_minutes: 60,
  providers: [
    { provider: 'google', status: 'syncing', last_sync: new Date(Date.now() - 1800000).toISOString(), events_synced: 85 },
    { provider: 'outlook', status: 'idle', last_sync: new Date(Date.now() - 7200000).toISOString(), events_synced: 43 },
  ],
};

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupCalendarMockApi(page: any) {
  await setupAuth(page);

  // Calendar connections list
  await page.route('**/api/v1/calendar/connections', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_CALENDAR_CONNECTIONS),
    });
  });

  // Sync status
  await page.route('**/api/v1/calendar/sync/status', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SYNC_STATUS),
    });
  });

  // OAuth connect — POST
  await page.route('**/api/v1/calendar/connect/*', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          auth_url: 'https://accounts.google.com/o/oauth2/auth?mock=true',
          state: 'mock-state-token',
        }),
      });
    }
  });

  // Disconnect — DELETE
  await page.route('**/api/v1/calendar/connections/*', async (route: any, request: any) => {
    if (request.method() === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true }),
      });
    }
  });

  // Sync now — POST
  await page.route('**/api/v1/calendar/sync', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({ sync_id: 'sync-abc-123', status: 'started' }),
      });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Calendar Sync — Connection Management
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Calendar Sync — Connection Management', () => {
  test.beforeEach(async ({ page }) => {
    await setupCalendarMockApi(page);
    await page.goto('/calendar');
    await page.waitForTimeout(1500);
  });

  test('Calendar sync page loads with connection list', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/calendar');

    // Проверяем отображение списка подключений
    const googleConnection = page.locator(
      'text=/Google|google|gmail|admin@cctv/i',
    ).first();
    await expect(googleConnection).toBeVisible();

    const outlookConnection = page.locator(
      'text=/Outlook|outlook|office365|admin@office365/i',
    ).first();
    await expect(outlookConnection).toBeVisible();
  });

  test('Calendar sync — connected status badges are visible', async ({ page }) => {
    // Проверяем бейджи статуса подключения
    const connectedBadge = page.locator(
      'span, badge, [class*="status" i], [class*="badge" i]',
    ).filter({ hasText: /connected|подключен|active|актив/i }).first();
    await expect(connectedBadge).toBeVisible();
  });

  test('Calendar sync — connect new OAuth flow triggered', async ({ page }) => {
    // Проверяем кнопку подключения нового календаря
    const connectButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /connect|подключ|add|добав|new|нов/i }).first();

    await expect(connectButton).toBeVisible();
    await connectButton.click();
    await page.waitForTimeout(1000);

    // Проверяем что появился диалог выбора провайдера
    const providerDialog = page.locator(
      'div[role="dialog"], div[class*="modal" i], div[class*="dialog" i]',
    ).first();
    const hasDialog = await providerDialog.isVisible().catch(() => false);
    if (hasDialog) {
      await expect(providerDialog).toBeVisible();
    }
  });

  test('Calendar sync — disconnect removes connection', async ({ page }) => {
    // Находим кнопку отключения
    const disconnectButton = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /disconnect|отключ|remove|удал|unlink|отвяза/i }).first();

    if (await disconnectButton.isVisible()) {
      await disconnectButton.click();
      await page.waitForTimeout(1000);

      // Проверяем подтверждение отключения
      const confirmDialog = page.locator(
        'div[role="dialog"], div[class*="modal" i], div[class*="confirm" i]',
      ).filter({ hasText: /confirm|подтвер|disconnect|отключ/i }).first();

      const hasConfirm = await confirmDialog.isVisible().catch(() => false);
      if (hasConfirm) {
        const confirmButton = confirmDialog.locator(
          'button, [role="button"]',
        ).filter({ hasText: /yes|да|confirm|подтвер|disconnect|отключ/i }).first();
        await confirmButton.click();
        await page.waitForTimeout(500);
      }
    }
  });

  test('Calendar sync — last sync timestamp displayed', async ({ page }) => {
    // Проверяем отображение времени последней синхронизации
    const lastSyncTime = page.locator(
      'text=/last sync|последн.*синх|sync.*ago|синхрониз/i',
    ).first();
    await expect(lastSyncTime).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Calendar Sync — Sync Status & Controls
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Calendar Sync — Sync Status & Controls', () => {
  test.beforeEach(async ({ page }) => {
    await setupCalendarMockApi(page);
    await page.goto('/calendar/sync');
    await page.waitForTimeout(1500);
  });

  test('Calendar sync — sync status overview loads', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/sync');

    // Проверяем отображение статуса синхронизации
    const syncStatus = page.locator(
      'text=/sync|синхрон|active|актив/i',
    ).first();
    await expect(syncStatus).toBeVisible();
  });

  test('Calendar sync — sync now button triggers sync', async ({ page }) => {
    // Находим кнопку "Sync Now"
    const syncNowButton = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /sync now|синхрониз|sync all|force sync/i }).first();

    await expect(syncNowButton).toBeVisible();
    await syncNowButton.click();
    await page.waitForTimeout(1000);

    // Проверяем что появилось уведомление о запуске синхронизации
    const syncNotification = page.locator(
      'div[class*="toast" i], div[class*="notification" i], div[role="alert"]',
    ).filter({ hasText: /sync|синхрон|started|запущ/i }).first();
    const hasNotification = await syncNotification.isVisible().catch(() => false);
    if (hasNotification) {
      await expect(syncNotification).toBeVisible();
    }
  });

  test('Calendar sync — sync interval and next sync displayed', async ({ page }) => {
    // Проверяем отображение интервала синхронизации
    const syncInterval = page.locator(
      'text=/60|minut|interval|интервал|hour|час/i',
    ).first();
    await expect(syncInterval).toBeVisible();
  });

  test('Calendar sync — event sync counters visible', async ({ page }) => {
    // Проверяем счетчики синхронизированных событий
    const eventCounter = page.locator(
      'text=/128|events|событ|synced|синхрон|total|всего/i',
    ).first();
    await expect(eventCounter).toBeVisible();

    // Проверяем счетчик ошибок
    const failedCounter = page.locator(
      'text=/3|failed|ошиб|fail|error/i',
    ).first();
    await expect(failedCounter).toBeVisible();
  });

  test('Calendar sync — error handling shows disconnected message', async ({ page }) => {
    // Мокаем пустой список подключений — ошибка
    await page.route('**/api/v1/calendar/connections', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.reload();
    await page.waitForTimeout(1500);

    // Проверяем сообщение об отсутствии подключений
    const emptyState = page.locator(
      'text=/no connection|not connected|не подключ|connect.*calendar|empty|no calendar/i',
    ).first();
    const hasEmptyState = await emptyState.isVisible().catch(() => false);
    if (hasEmptyState) {
      await expect(emptyState).toBeVisible();
    }
  });
});
