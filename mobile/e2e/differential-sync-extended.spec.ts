// ──────────────────────────────────────────────────
// E2E: Differential Sync Extended
//
// Проверяет:
//   - Полный sync cycle с progress bar
//   - Sync with large delta (multiple entities)
//   - Selective sync with filter by entity type
//   - Auto-retry on network error
//   - Sync badge showing change count
//   - Partial sync (only work orders)
// ──────────────────────────────────────────────────

import { mockWorkOrders } from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Mock Data: Large Delta ───────────────────────

const mockLargeDeltaResponse = {
  changes: [
    { id: 'wo-010', type: 'created', entity: 'work_orders', fields: {
      device_id: 'cam-110', device_name: 'Camera Loading Dock', site_name: 'Facility B',
      type: 'corrective', status: 'open', priority: 'medium',
      notes: 'Created via sync', created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
    }, updated_at: new Date().toISOString() },
    { id: 'wo-011', type: 'created', entity: 'work_orders', fields: {
      device_id: 'cam-111', device_name: 'Camera Server Room', site_name: 'Facility A',
      type: 'preventive', status: 'open', priority: 'low',
      notes: 'Quarterly inspection', created_at: new Date().toISOString(), updated_at: new Date().toISOString(),
    }, updated_at: new Date().toISOString() },
    { id: 'cam-101', type: 'updated', entity: 'devices', fields: {
      name: 'Camera Main Entrance', status: 'ONLINE', health: 'healthy', updated_at: new Date().toISOString(),
    }, updated_at: new Date().toISOString() },
    { id: 'cam-102', type: 'updated', entity: 'devices', fields: {
      name: 'Camera Parking Lot', status: 'DEGRADED', health: 'degraded', updated_at: new Date().toISOString(),
    }, updated_at: new Date().toISOString() },
    { id: 'ph-001', type: 'deleted', entity: 'photos', fields: {}, updated_at: new Date().toISOString() },
  ],
  timestamp: new Date().toISOString(),
  compressed: false,
  entity: 'work_orders',
  has_more: false,
  total_count: 5,
};

const mockSyncStatusResponse = {
  bandwidth_usage_bytes: 128_000,
  last_sync: {
    work_orders: new Date(Date.now() - 30_000).toISOString(),
    devices: new Date(Date.now() - 60_000).toISOString(),
    photos: new Date(Date.now() - 120_000).toISOString(),
    audit: new Date(Date.now() - 300_000).toISOString(),
  },
  total_syncs: 24,
  total_changes: 156,
};

// ── Helpers ──────────────────────────────────────

async function setupMockRoutes(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/auth/login`, method: 'POST' },
    { status: 200, body: { token: 'mock-token', refresh_token: 'mock-refresh', user: { id: 'tech-1', username: 'johntech', role: 'technician' } }, headers: { 'Content-Type': 'application/json' } },
  );
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
    { status: 200, body: mockWorkOrders, headers: { 'Content-Type': 'application/json' } },
  );
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
    { status: 200, body: mockLargeDeltaResponse, headers: { 'Content-Type': 'application/json' } },
  );
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/status`, method: 'GET' },
    { status: 200, body: mockSyncStatusResponse, headers: { 'Content-Type': 'application/json' } },
  );
  await device.mockRoute(
    { url: `${API_BASE}/mobile/technician/profile`, method: 'GET' },
    { status: 200, body: { user_id: 'tech-1', user_name: 'John Technician' }, headers: { 'Content-Type': 'application/json' } },
  );
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    { status: 200, body: { devices: [] }, headers: { 'Content-Type': 'application/json' } },
  );
}

async function setupNetworkErrorMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
    { status: 500, body: { error: 'Internal Server Error' }, headers: { 'Content-Type': 'application/json' } },
  );
}

async function teardownMockRoutes(): Promise<void> {
  await device.unmockRoute({ url: `${API_BASE}/mobile/auth/login` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/sync/delta` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/sync/status` });
}

// ── Init ─────────────────────────────────────────

beforeAll(async () => {
  await device.launchApp({
    newInstance: true,
    permissions: { location: 'always' },
  });
  await setupMockRoutes();
});

afterAll(async () => {
  await teardownMockRoutes();
});

beforeEach(async () => {
  await device.reloadReactNative();
  await waitForAnimation(2000);
});

// ── Tests ────────────────────────────────────────

describe('Differential Sync — Extended', () => {
  // ── 1. Full sync cycle with progress ──

  it('Должен выполнить полный sync cycle с progress bar', async () => {
    await waitForAnimation(3000);

    // Trigger sync
    await element(by.id('sync-now-btn')).tap();

    // Progress bar visible during sync
    await expect(element(by.id('sync-progress-bar'))).toBeVisible();

    // Progress text
    await expect(element(by.text('Applying delta patches...'))).toBeVisible();

    // Individual entity progress
    await expect(element(by.id('sync-progress-work_orders'))).toBeVisible();
    await expect(element(by.id('sync-progress-devices'))).toBeVisible();

    // Wait for completion
    await waitForAnimation(4000);

    // Progress bar hidden after completion
    await expect(element(by.id('sync-progress-bar'))).not.toBeVisible();

    // Sync complete badge visible
    await expect(element(by.id('sync-complete-badge'))).toBeVisible();
  });

  // ── 2. Sync with large delta (5 changes) ──

  it('Должен применить large delta (5 изменений) и отобразить счётчик', async () => {
    await waitForAnimation(3000);

    // Trigger sync
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(4000);

    // Sync complete badge shows 5 changes
    await expect(element(by.id('sync-complete-badge'))).toBeVisible();
    await expect(element(by.id('sync-changes-count'))).toHaveText('5');
  });

  // ── 3. Sync status shows bandwidth usage ──

  it('Должен отобразить использование Bandwidth в статусе sync', async () => {
    await waitForAnimation(3000);

    // Open sync settings
    await element(by.id('sync-settings-btn')).tap();
    await waitForAnimation(1000);

    // Bandwidth usage displayed
    await expect(element(by.id('sync-bandwidth-usage'))).toBeVisible();
    const bandwidthText = await element(by.id('sync-bandwidth-usage')).text();
    expect(bandwidthText).not.toBe('');
  });

  // ── 4. Retry after network error ──

  it('Должен retry sync при сетевой ошибке и восстановиться', async () => {
    await waitForAnimation(3000);

    // Switch to error mock
    await setupNetworkErrorMock();

    // Trigger sync
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Error badge visible
    await expect(element(by.id('sync-error-badge'))).toBeVisible();
    await expect(element(by.id('sync-retry-btn'))).toBeVisible();

    // Restore mock
    await setupMockRoutes();

    // Retry
    await element(by.id('sync-retry-btn')).tap();
    await waitForAnimation(4000);

    // Error cleared, success
    await expect(element(by.id('sync-error-badge'))).not.toBeVisible();
    await expect(element(by.id('sync-complete-badge'))).toBeVisible();
  });

  // ── 5. Selective sync by entity ──

  it('Должен выполнить selective sync только для work_orders', async () => {
    await waitForAnimation(3000);

    // Open sync settings
    await element(by.id('sync-settings-btn')).tap();
    await waitForAnimation(1000);

    // Select only work_orders
    await element(by.text('work_orders')).tap();
    await waitForAnimation(500);

    // Start selective sync
    await element(by.id('sync-selective-btn')).tap();
    await waitForAnimation(3000);

    // Only work_orders progress visible
    await expect(element(by.id('sync-progress-work_orders'))).toBeVisible();

    // Other entities not synced
    await expect(element(by.id('sync-progress-devices'))).not.toBeVisible();
  });

  // ── 6. Sync with zero changes ──

  it('Должен показать 0 изменений при пустой дельте', async () => {
    // Setup empty delta
    await device.mockRoute(
      { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
      { status: 200, body: { changes: [], timestamp: new Date().toISOString(), compressed: false, entity: 'work_orders', has_more: false, total_count: 0 }, headers: { 'Content-Type': 'application/json' } },
    );

    await waitForAnimation(3000);

    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    await expect(element(by.id('sync-complete-badge'))).toBeVisible();
    await expect(element(by.id('sync-changes-count'))).toHaveText('0');

    // Restore mock
    await setupMockRoutes();
  });

  // ── 7. Last sync timestamps per entity ──

  it('Должен отобразить время последней синхронизации для каждой сущности', async () => {
    await waitForAnimation(3000);

    // Open sync settings
    await element(by.id('sync-settings-btn')).tap();
    await waitForAnimation(1000);

    // Last sync timestamps for each entity
    await expect(element(by.id('sync-last-work_orders'))).toBeVisible();
    await expect(element(by.id('sync-last-devices'))).toBeVisible();
    await expect(element(by.id('sync-last-photos'))).toBeVisible();
  });
});
