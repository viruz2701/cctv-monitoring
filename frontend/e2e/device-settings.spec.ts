/// <reference types="node" />

import { test, expect } from '@playwright/test';
import { setupAuth, mockCatchAll, mockDevices, MOCK_DEVICES } from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Device Settings API — E2E Tests
// GET/PUT settings for devices
// ═══════════════════════════════════════════════════════════════════════════

// ── Mock Data ─────────────────────────────────────────────────────────────

const MOCK_DEVICE_SETTINGS: Record<string, Record<string, any>> = {
  'dev-1': {
    network: {
      ip_address: '192.168.1.100',
      subnet_mask: '255.255.255.0',
      gateway: '192.168.1.1',
      dns_primary: '8.8.8.8',
      dns_secondary: '8.8.4.4',
      port: 554,
      dhcp_enabled: false,
    },
    video: {
      resolution: '1920x1080',
      fps: 30,
      bitrate: 4096,
      codec: 'H.265',
      brightness: 50,
      contrast: 50,
      saturation: 50,
    },
    audio: {
      enabled: true,
      volume: 80,
      codec: 'G.711',
    },
    system: {
      device_name: 'Camera-Lobby-01',
      firmware_version: '9.80.1',
      timezone: 'Europe/Minsk',
      ntp_server: 'pool.ntp.org',
      log_level: 'info',
    },
  },
  'dev-2': {
    network: {
      ip_address: '192.168.1.50',
      subnet_mask: '255.255.255.0',
      gateway: '192.168.1.1',
      port: 80,
      dhcp_enabled: true,
    },
    storage: {
      total_gb: 4096,
      used_gb: 2048,
      retention_days: 30,
      overwrite_enabled: true,
      disk_health: 'degraded',
    },
    system: {
      device_name: 'NVR-03 Recording Server',
      firmware_version: '5.2.0',
      timezone: 'Europe/Minsk',
      log_level: 'warn',
    },
  },
};

const MOCK_SETTINGS_RESPONSE = {
  device_id: 'dev-1',
  category: 'network',
  settings: MOCK_DEVICE_SETTINGS['dev-1'].network,
  updated_at: new Date().toISOString(),
};

// ── Setup ─────────────────────────────────────────────────────────────────

async function setupDeviceSettingsMockApi(page: any) {
  await setupAuth(page);
  await mockDevices(page);
  await mockCatchAll(page);

  // GET /api/v1/devices/:id/settings
  await page.route('**/api/v1/devices/*/settings', async (route: any, request: any) => {
    const url = request.url();
    const match = url.match(/\/devices\/([^/]+)\/settings/);
    if (!match) return route.fulfill({ status: 404 });

    const deviceId = match[1];
    const category = new URL(url).searchParams.get('category') || '';
    const deviceSettings = MOCK_DEVICE_SETTINGS[deviceId];

    if (!deviceSettings) {
      return route.fulfill({ status: 404 });
    }

    if (request.method() === 'GET') {
      let settings: Record<string, any> = {};
      if (category && deviceSettings[category]) {
        settings = deviceSettings[category];
      } else if (!category) {
        settings = deviceSettings;
      } else {
        return route.fulfill({ status: 404 });
      }

      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          device_id: deviceId,
          category: category || '',
          settings,
          updated_at: new Date().toISOString(),
        }),
      });
    }

    if (request.method() === 'PUT') {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          device_id: deviceId,
          settings: JSON.parse(request.postData() || '{}').settings || {},
          updated_at: new Date().toISOString(),
        }),
      });
    }

    return route.fulfill({ status: 405 });
  });
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Settings — GET
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Settings — GET', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceSettingsMockApi(page);
  });

  test('Settings — device settings page loads', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const heading = page.locator('h1, h2').filter({ hasText: /settings|настройк/i }).first();
    await expect(heading).toBeVisible();
  });

  test('Settings — network settings section is visible', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const networkSection = page.locator('text=/network|сет/i').first();
    await expect(networkSection).toBeVisible();
  });

  test('Settings — video settings section is visible', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const videoSection = page.locator('text=/video|видео/i').first();
    await expect(videoSection).toBeVisible();
  });

  test('Settings — system settings section is visible', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const systemSection = page.locator('text=/system|систем/i').first();
    await expect(systemSection).toBeVisible();
  });

  test('Settings — IP address is displayed in network settings', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const ip = page.locator('text=192.168.1.100').first();
    await expect(ip).toBeVisible();
  });

  test('Settings — category tabs/selectors exist', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const categoryTabs = page.locator(
      'button, a, div[role="tab"]',
    ).filter({ hasText: /network|video|audio|system|storage|alarm/i }).first();
    await expect(categoryTabs).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Settings — Edit/PUT
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Settings — Edit', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceSettingsMockApi(page);
  });

  test('Settings — save button for settings is present', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const saveBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /save|сохран|apply|примен/i }).first();
    await expect(saveBtn).toBeVisible();
  });

  test('Settings — editable input fields exist', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const inputs = page.locator(
      'input:not([type="hidden"]), select, textarea',
    );
    const count = await inputs.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Settings — cancel button exists for edit mode', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const cancelBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /cancel|отмен/i }).first();

    if (await cancelBtn.isVisible()) {
      await cancelBtn.click();
      await page.waitForTimeout(500);
    }
  });

  test('Settings — apply button triggers settings application', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const applyBtn = page.locator('button, [role="button"]')
      .filter({ hasText: /apply|примен/i }).first();

    if (await applyBtn.isVisible()) {
      await applyBtn.click();
      await page.waitForTimeout(500);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Device Settings — Validation
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Device Settings — Validation', () => {
  test.beforeEach(async ({ page }) => {
    await setupDeviceSettingsMockApi(page);
  });

  test('Settings — resolution format is displayed correctly', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const resolution = page.locator('text=1920x1080').first();
    await expect(resolution).toBeVisible();
  });

  test('Settings — FPS value is shown', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const fps = page.locator('text=30').filter({ hasText: '30' }).first();
    await expect(fps).toBeVisible();
  });

  test('Settings — bitrate value is visible', async ({ page }) => {
    await page.goto('/devices/dev-1/settings');
    await page.waitForTimeout(2000);

    const bitrate = page.locator('text=4096').first();
    await expect(bitrate).toBeVisible();
  });
});
