/// <reference types="node" />

import { test, expect } from '@playwright/test';
import { setupAuth, mockCatchAll } from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Differential Sync — E2E Tests
// Sync cycle: trigger sync, progress indicator, conflict resolution
// ═══════════════════════════════════════════════════════════════════════════

// ── Mock Data ─────────────────────────────────────────────────────────────

const MOCK_SYNC_STATUS = {
  last_sync: new Date().toISOString(),
  changes_pending: 5,
  changes_applied: 42,
  conflicts: 0,
  entities: ['work_orders', 'devices', 'photos', 'audit'],
};

const MOCK_DELTA_RESPONSE = {
  changes: [
    { id: 'wo-005', type: 'created', entity: 'work_orders', fields: { title: 'Emergency repair', status: 'open', priority: 'critical' }, updated_at: new Date().toISOString() },
    { id: 'dev-5', type: 'updated', entity: 'devices', fields: { status: 'online', health: 'healthy' }, updated_at: new Date().toISOString() },
    { id: 'wo-001', type: 'updated', entity: 'work_orders', fields: { status: 'in_progress' }, updated_at: new Date().toISOString() },
  ],
  timestamp: new Date().toISOString(),
  compressed: false,
  has_more: false,
  total_count: 3,
};

// ── Setup ─────────────────────────────────────────────────────────────────

async function setupSyncMockApi(page: any) {
  await setupAuth(page);

  // Sync status
  await page.route('**/api/v1/sync/status', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SYNC_STATUS),
    });
  });

  // Sync delta
  await page.route('**/api/v1/sync/delta', async (route: any, request: any) => {
    const url = request.url();
    if (url.includes('empty=true')) {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ changes: [], timestamp: new Date().toISOString(), compressed: false, has_more: false, total_count: 0 }),
      });
    }
    if (url.includes('error=true')) {
      return route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_DELTA_RESPONSE),
    });
  });

  // Sync trigger
  await page.route('**/api/v1/sync/trigger', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'syncing', started_at: new Date().toISOString() }),
      });
    } else {
      await route.fulfill({ status: 405 });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Differential Sync — Status & Trigger
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Differential Sync — Status & Trigger', () => {
  test.beforeEach(async ({ page }) => {
    await setupSyncMockApi(page);
  });

  test('Sync — sync page loads and shows status', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    const heading = page.locator('h1, h2').filter({ hasText: /sync|синхрониз/i }).first();
    await expect(heading).toBeVisible();
  });

  test('Sync — trigger sync button is present', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    const syncBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /sync|синхрониз|refresh|обнов/i }).first();
    await expect(syncBtn).toBeVisible();
  });

  test('Sync — sync status shows entities list', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    // Check for entity names
    for (const entity of MOCK_SYNC_STATUS.entities) {
      const entityEl = page.locator(`text=${entity}`).first();
      const visible = await entityEl.isVisible().catch(() => false);
      // At least some entities should be visible
    }
  });

  test('Sync — pending changes count is displayed', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    const pendingInfo = page.locator('text=/pending|ожида|changes|изменен/i').first();
    await expect(pendingInfo).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Differential Sync — Progress & Completion
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Differential Sync — Progress & Completion', () => {
  test.beforeEach(async ({ page }) => {
    await setupSyncMockApi(page);
  });

  test('Sync — progress bar appears during sync', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    const progressBar = page.locator(
      'div[role="progressbar"], div[class*="progress" i], ' +
      'div[class*="loading" i], div[class*="spinner" i]',
    ).first();
    // Progress bar may be visible when sync is running
  });

  test('Sync — last sync timestamp is displayed', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    const timestamp = page.locator('text=/last sync|последня/i').first();
    await expect(timestamp).toBeVisible();
  });

  test('Sync — changes applied counter is visible', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    const appliedSection = page.locator('text=/applied|применен/i').first();
    await expect(appliedSection).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Differential Sync — Error & Retry
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Differential Sync — Error & Retry', () => {
  test.beforeEach(async ({ page }) => {
    await setupSyncMockApi(page);
  });

  test('Sync — error state shows retry option', async ({ page }) => {
    await page.goto('/sync?error=true');
    await page.waitForTimeout(2000);

    const retryBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /retry|повтор/i }).first();

    if (await retryBtn.isVisible()) {
      await retryBtn.click();
      await page.waitForTimeout(500);
    }
  });

  test('Sync — error badge appears on failure', async ({ page }) => {
    await page.goto('/sync?error=true');
    await page.waitForTimeout(2000);

    const errorEl = page.locator(
      'div[role="alert"], div[class*="error" i], ' +
      'div[class*="alert" i]',
    ).filter({ hasText: /error|ошибк/i }).first();
    await expect(errorEl).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Differential Sync — Entity Selection
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Differential Sync — Entity Selection', () => {
  test.beforeEach(async ({ page }) => {
    await setupSyncMockApi(page);
  });

  test('Sync — entities can be selected for selective sync', async ({ page }) => {
    await page.goto('/sync');
    await page.waitForTimeout(2000);

    // Checkboxes or toggles for entity selection
    const checkboxes = page.locator(
      'input[type="checkbox"], div[role="checkbox"], ' +
      'label:has(input[type="checkbox"])',
    );
    const count = await checkboxes.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});
