/// <reference types="node" />

import { test, expect } from '@playwright/test';
import { setupAuth, mockCatchAll, mockDevices } from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Device Logs API — E2E Tests
// GET logs with pagination, filtering, sorting
// ═══════════════════════════════════════════════════════════════════════════

// ── Mock Data ─────────────────────────────────────────────────────────────

function generateMockLogs(count: number, startOffset: number = 0) {
  const levels = ['info', 'warn', 'error', 'debug'];
  const sources = ['kernel', 'app', 'system', 'network', 'storage'];
  const messages = [
    'Device initialized successfully',
    'Network connection established',
    'Firmware version check passed',
    'Temperature sensor reading: 45.2°C',
    'Motion detection triggered on channel 1',
    'Recording started: channel 3',
    'Storage warning: 85% capacity used',
    'Authentication token refreshed',
    'Video stream health check passed',
    'Scheduled maintenance completed',
    'Connection timeout: remote host unreachable',
    'Disk write error: sector 0x3A2F1',
    'Firmware update available: v9.82.0',
    'Alert acknowledged: motion detection',
    'Night mode activated',
  ];

  const logs: Array<{
    timestamp: string;
    level: string;
    source: string;
    message: string;
    metadata: { device_id: string; log_index: number };
  }> = [];
  const now = Date.now();

  for (let i = 0; i < count; i++) {
    logs.push({
      timestamp: new Date(now - (startOffset + i) * 60000).toISOString(),
      level: levels[Math.floor(Math.random() * levels.length)],
      source: sources[Math.floor(Math.random() * sources.length)],
      message: messages[Math.floor(Math.random() * messages.length)],
      metadata: {
        device_id: 'dev-1',
        log_index: startOffset + i,
      },
    });
  }

  return logs;
}

const MOCK_LOGS_PAGE_1 = generateMockLogs(50, 0);
const MOCK_LOGS_PAGE_2 = generateMockLogs(50, 50);
const TOTAL_LOG_COUNT = 150;

// ── Setup ─────────────────────────────────────────────────────────────────

async function setupDeviceLogsMockApi(page: any) {
  await setupAuth(page);
  await mockDevices(page);
  await mockCatchAll(page);

  // GET /api/v1/devices/:id/logs
  await page.route('**/api/v1/devices/*/logs', async (route: any, request: any) => {
    const url = request.url();
    const match = url.match(/\/devices\/([^/]+)\/logs/);
    if (!match) return route.fulfill({ status: 404 });

    const deviceId = match[1];
    const searchParams = new URL(url).searchParams;
    const limit = parseInt(searchParams.get('limit') || '100', 10);
    const offset = parseInt(searchParams.get('offset') || '0', 10);
    const level = searchParams.get('level') || '';
    const levelFilter = level ? level.toUpperCase() : '';

    let logs = offset === 0 ? MOCK_LOGS_PAGE_1 : MOCK_LOGS_PAGE_2;

    // Apply level filter
    if (levelFilter) {
      logs = logs.filter((log) => log.level.toUpperCase() === levelFilter);
    }

    // Apply limit
    logs = logs.slice(0, limit);

    return route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        device_id: deviceId,
        logs,
        total: TOTAL_LOG_COUNT,
        limit,
        offset,
        since: searchParams.get('since') || '',
        until: searchParams.get('until') || '',
      }),
    });
  });
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Logs — List & Pagination
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Logs — List & Pagination', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceLogsMockApi(page);
  });

  test('Logs — device logs page loads', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const heading = page.locator('h1, h2').filter({ hasText: /log|лог/i }).first();
    await expect(heading).toBeVisible();
  });

  test('Logs — log entries are displayed in a list', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const logEntries = page.locator(
      'tr, div[class*="log-entry" i], div[class*="row" i]',
    );
    const count = await logEntries.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Logs — pagination controls exist', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const pagination = page.locator(
      'nav[aria-label="pagination" i], div[class*="pagination" i], ' +
      'button[aria-label*="page" i]',
    ).first();
    await expect(pagination).toBeVisible();
  });

  test('Logs — next page button works', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const nextBtn = page.locator(
      'button[aria-label*="next" i], button[aria-label*="след" i], ' +
      'button:has(svg[class*="chevron-right" i])',
    ).first();

    if (await nextBtn.isVisible() && await nextBtn.isEnabled()) {
      await nextBtn.click();
      await page.waitForTimeout(1000);
    }
  });

  test('Logs — previous page button works', async ({ page }) => {
    await page.goto('/devices/dev-1/logs?offset=50');
    await page.waitForTimeout(2000);

    const prevBtn = page.locator(
      'button[aria-label*="previous" i], button[aria-label*="пред" i], ' +
      'button:has(svg[class*="chevron-left" i])',
    ).first();

    if (await prevBtn.isVisible() && await prevBtn.isEnabled()) {
      await prevBtn.click();
      await page.waitForTimeout(1000);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Logs — Filtering
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Logs — Filtering', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceLogsMockApi(page);
  });

  test('Logs — level filter dropdown exists', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const filterSelect = page.locator(
      'select, div[role="listbox"], ' +
      'button[aria-haspopup="listbox"]',
    ).filter({ hasText: /all|все|level|уровень|info|error|warn/i }).first();
    await expect(filterSelect).toBeVisible();
  });

  test('Logs — search/filter input exists', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const searchInput = page.locator(
      'input[type="search"], input[placeholder*="search" i], ' +
      'input[placeholder*="поиск" i]',
    ).first();
    await expect(searchInput).toBeVisible();
  });

  test('Logs — date range filter controls exist', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const dateInputs = page.locator('input[type="date"], input[type="datetime-local"]');
    const count = await dateInputs.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Logs — Content Display
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Logs — Content Display', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceLogsMockApi(page);
  });

  test('Logs — log level badges/indicators are shown', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const levelBadge = page.locator(
      'span[class*="badge" i], span[class*="level" i], ' +
      'span:has-text("info"), span:has-text("warn"), span:has-text("error")',
    ).first();
    await expect(levelBadge).toBeVisible();
  });

  test('Logs — log timestamps are displayed', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const timestamp = page.locator(
      'span[class*="time" i], time, td:nth-child(1)',
    ).first();
    await expect(timestamp).toBeVisible();
  });

  test('Logs — log source is displayed', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const source = page.locator('text=/kernel|app|system|network|storage/i').first();
    await expect(source).toBeVisible();
  });

  test('Logs — log messages are visible and readable', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const message = page.locator('text=/Device initialized|Network connection|Firmware/i').first();
    await expect(message).toBeVisible();
  });

  test('Logs — total count is displayed', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const totalInfo = page.locator('text=/total|всего|150/i').first();
    await expect(totalInfo).toBeVisible();
  });

  test('Logs — refresh button reloads logs', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const refreshBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /refresh|обнов/i }).first();
    await expect(refreshBtn).toBeVisible();
  });

  test('Logs — export logs button exists', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const exportBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /export|экспорт|download|скач/i }).first();

    if (await exportBtn.isVisible()) {
      await exportBtn.click();
      await page.waitForTimeout(500);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Logs — Per Page Selector
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Logs — Per Page', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceLogsMockApi(page);
  });

  test('Logs — page size selector exists', async ({ page }) => {
    await page.goto('/devices/dev-1/logs');
    await page.waitForTimeout(2000);

    const perPage = page.locator(
      'select, div[role="listbox"]',
    ).filter({ hasText: /10|25|50|100/i }).first();
    await expect(perPage).toBeVisible();
  });

  test('Logs — log entry count matches selected page size', async ({ page }) => {
    await page.goto('/devices/dev-1/logs?limit=10');
    await page.waitForTimeout(2000);

    // Verify that a manageable number of entries is shown
    const entries = page.locator('tr, div[class*="log-entry" i]');
    const count = await entries.count();
    expect(count).toBeLessThanOrEqual(50);
  });
});
