// ──────────────────────────────────────────────────
// E2E: Differential Sync
//
// Проверяет:
//   - Применение delta-патчей (ChangeEntry) в локальный кэш
//   - Триггер differential sync при переходе offline→online
//   - Sync progress bar / spinner во время применения delta
//   - Conflict resolution modal при конфликтах после sync
//   - Selective sync (выбор сущностей для синхронизации)
//   - Background sync retry при сетевой ошибке
//   - Badge/notification о завершении синхронизации
// ──────────────────────────────────────────────────

import { mockWorkOrders } from './helpers/mockData';
import {
  waitForElement,
  waitForAnimation,
} from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Mock Data: Delta Patches ─────────────────────

/**
 * Мок-данные для delta-патчей, соответствующие структуре DeltaResponse
 * из differentialSync.ts:
 *   - ChangeEntry[] с type: 'created' | 'updated' | 'deleted'
 *   - Поля fields — только changed поля (partial update)
 *   - entity: 'work_orders' | 'devices' | 'photos' | 'audit'
 */
const mockDeltaResponse = {
  changes: [
    {
      id: 'wo-003',
      type: 'created' as const,
      entity: 'work_orders',
      fields: {
        device_id: 'cam-103',
        device_name: 'Camera Warehouse',
        site_name: 'Facility B',
        type: 'preventive',
        status: 'open',
        priority: 'medium',
        assigned_to: 'tech-1',
        notes: 'Scheduled monthly check',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      updated_at: new Date().toISOString(),
    },
    {
      id: 'wo-001',
      type: 'updated' as const,
      entity: 'work_orders',
      fields: {
        status: 'in_progress',
        notes: 'Dispatched via delta sync',
        updated_at: new Date().toISOString(),
      },
      updated_at: new Date().toISOString(),
    },
    {
      id: 'cam-102',
      type: 'updated' as const,
      entity: 'devices',
      fields: {
        name: 'Camera Parking Lot',
        status: 'ONLINE',
        health: 'healthy',
        updated_at: new Date().toISOString(),
      },
      updated_at: new Date().toISOString(),
    },
  ],
  timestamp: new Date().toISOString(),
  compressed: false,
  entity: 'work_orders',
  has_more: false,
  total_count: 3,
};

/** Мок для sync status endpoint */
const mockSyncStatusResponse = {
  bandwidth_usage_bytes: 42_000,
  last_sync: {
    work_orders: new Date(Date.now() - 60_000).toISOString(),
    devices: new Date(Date.now() - 120_000).toISOString(),
    photos: new Date(Date.now() - 300_000).toISOString(),
    audit: new Date(Date.now() - 600_000).toISOString(),
  },
  total_syncs: 12,
  total_changes: 47,
};

/** Мок для серверной версии work order (для конфликта) */
const mockServerUpdatedWorkOrder = {
  ...mockWorkOrders[0],
  status: 'completed',
  notes: 'Completed by dispatcher remotely',
  updated_at: new Date().toISOString(),
};

// ── Helpers ──────────────────────────────────────

async function setupMockRoutes(): Promise<void> {
  // Auth
  await device.mockRoute(
    { url: `${API_BASE}/mobile/auth/login`, method: 'POST' },
    {
      status: 200,
      body: {
        token: 'mock-token',
        refresh_token: 'mock-refresh',
        user: { id: 'tech-1', username: 'johntech', role: 'technician' },
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Work orders (initial list)
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
    {
      status: 200,
      body: mockWorkOrders,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Differential sync delta endpoint
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
    {
      status: 200,
      body: mockDeltaResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Sync status
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/status`, method: 'GET' },
    {
      status: 200,
      body: mockSyncStatusResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Selective sync (by entity)
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta?entity=work_orders`, method: 'GET' },
    {
      status: 200,
      body: {
        ...mockDeltaResponse,
        entity: 'work_orders',
        changes: mockDeltaResponse.changes.filter(
          (c) => c.entity === 'work_orders',
        ),
        total_count: 2,
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Profile
  await device.mockRoute(
    { url: `${API_BASE}/mobile/technician/profile`, method: 'GET' },
    {
      status: 200,
      body: { user_id: 'tech-1', user_name: 'John Technician' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Device map
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    {
      status: 200,
      body: { devices: [] },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Work order start
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'in_progress' },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

/** Переключить mock для delta response — пустой патч (нет изменений) */
async function setupEmptyDeltaMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
    {
      status: 200,
      body: {
        changes: [],
        timestamp: new Date().toISOString(),
        compressed: false,
        entity: 'work_orders',
        has_more: false,
        total_count: 0,
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

/** Переключить mock для delta с серверным конфликтом */
async function setupConflictDeltaMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
    {
      status: 200,
      body: {
        changes: [
          {
            id: 'wo-001',
            type: 'updated',
            entity: 'work_orders',
            fields: {
              status: 'completed',
              notes: 'Completed by dispatcher remotely',
              updated_at: new Date().toISOString(),
            },
            updated_at: new Date().toISOString(),
          },
        ],
        timestamp: new Date().toISOString(),
        compressed: false,
        entity: 'work_orders',
        has_more: false,
        total_count: 1,
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

/** Переключить mock для сетевой ошибки (500) */
async function setupNetworkErrorMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/delta`, method: 'GET' },
    {
      status: 500,
      body: { error: 'Internal Server Error' },
      headers: { 'Content-Type': 'application/json' },
    },
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

describe('Differential Sync', () => {
  // ── 1. Differential sync applies delta patches correctly ──

  it('Должен применить delta-патчи при синхронизации', async () => {
    await waitForAnimation(3000);

    // Выполняем differential sync
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // После применения delta: новый work order "Camera Warehouse" должен появиться
    await expect(element(by.text('Camera Warehouse'))).toBeVisible();

    // Существующий work order "Camera Main Entrance" обновлён статусом
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);

    // Статус изменился на "in_progress" (из delta-патча)
    await expect(element(by.id('work-order-status'))).toBeVisible();
    // Заметка от диспетчера через delta sync
    await expect(element(by.text('Dispatched via delta sync'))).toBeVisible();
  });

  // ── 2. Offline→online transition triggers differential sync ──

  it('Должен запустить differential sync при переходе offline→online', async () => {
    await waitForAnimation(3000);

    // Переходим в offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Симулируем мутацию в offline (меняем notes)
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Offline change pending sync');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Pending mutation сохранилась
    await expect(element(by.id('sync-pending-badge'))).toBeVisible();

    // Восстанавливаем сеть — триггер differential sync
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // После sync — pending badge исчезает
    await expect(element(by.id('sync-pending-badge'))).not.toBeVisible();

    // Данные с сервера (delta) применились: новый work order "Camera Warehouse"
    await expect(element(by.text('Camera Warehouse'))).toBeVisible();
  });

  // ── 3. Sync progress indicator during delta application ──

  it('Должен показывать ProgressBar/спиннер во время применения delta', async () => {
    await waitForAnimation(3000);

    // Запускаем sync
    await element(by.id('sync-now-btn')).tap();

    // Во время синхронизации отображается индикатор прогресса
    await expect(element(by.id('sync-progress-bar'))).toBeVisible();

    // Текст прогресса: "Applying delta patches..." или "Syncing..."
    await expect(element(by.text('Applying delta patches...'))).toBeVisible();

    // Прогресс для каждой сущности (work_orders, devices, photos, audit)
    await expect(element(by.id('sync-progress-work_orders'))).toBeVisible();

    // После завершения — индикатор скрывается
    await waitForAnimation(4000);
    await expect(element(by.id('sync-progress-bar'))).not.toBeVisible();
  });

  // ── 4. Conflict resolution modal appears on sync conflicts ──

  it('Должен показать Conflict Modal при конфликте после sync', async () => {
    await waitForAnimation(3000);

    // Меняем локально work order (имитируем конфликт)
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Local technician note');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Настраиваем mock с конфликтующей серверной версией
    await setupConflictDeltaMock();

    // Восстанавливаем сеть — триггер sync
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Должна появиться модалка конфликта
    await expect(element(by.id('conflict-modal'))).toBeVisible();
    await expect(element(by.text('Sync Conflicts'))).toBeVisible();

    // Показывается разница между локальной и серверной версией
    await expect(element(by.text('Local'))).toBeVisible();
    await expect(element(by.text('Server'))).toBeVisible();

    // Восстанавливаем исходный mock
    await setupMockRoutes();
  });

  // ── 5. Selective sync (choose which items to sync) ──

  it('Должен поддерживать selective sync (выбор сущностей)', async () => {
    await waitForAnimation(3000);

    // Открываем настройки sync
    await element(by.id('sync-settings-btn')).tap();
    await waitForAnimation(1000);

    // Показывается список сущностей для выбора
    await expect(element(by.id('sync-entity-picker'))).toBeVisible();
    await expect(element(by.text('work_orders'))).toBeVisible();
    await expect(element(by.text('devices'))).toBeVisible();
    await expect(element(by.text('photos'))).toBeVisible();
    await expect(element(by.text('audit'))).toBeVisible();

    // Выбираем только work_orders
    await element(by.text('work_orders')).tap();
    await waitForAnimation(500);

    // Запускаем selective sync
    await element(by.id('sync-selective-btn')).tap();
    await waitForAnimation(3000);

    // Прогресс отображается только для work_orders
    await expect(element(by.id('sync-progress-work_orders'))).toBeVisible();

    // Другие сущности не синхронизируются
    await expect(element(by.id('sync-progress-devices'))).not.toBeVisible();

    // После завершения — данные work_orders обновлены
    await waitForAnimation(2000);
    await expect(element(by.text('Camera Warehouse'))).toBeVisible();
  });

  // ── 6. Background sync retry on network failure ──

  it('Должен retry background sync при сетевой ошибке', async () => {
    await waitForAnimation(3000);

    // Настраиваем сетевую ошибку
    await setupNetworkErrorMock();

    // Пытаемся выполнить sync
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Появляется error badge
    await expect(element(by.id('sync-error-badge'))).toBeVisible();
    await expect(element(by.text('Sync error'))).toBeVisible();

    // Кнопка retry отображается
    await expect(element(by.id('sync-retry-btn'))).toBeVisible();

    // Восстанавливаем mock
    await setupMockRoutes();

    // Retry — нажимаем кнопку повтора
    await element(by.id('sync-retry-btn')).tap();
    await waitForAnimation(3000);

    // После retry — успех, error badge исчезает
    await expect(element(by.id('sync-error-badge'))).not.toBeVisible();

    // Данные применились
    await expect(element(by.text('Camera Warehouse'))).toBeVisible();
  });

  // ── 7. Sync completion badge/notification ──

  it('Должен показать badge/notification о завершении sync', async () => {
    await waitForAnimation(3000);

    // Устанавливаем пустой delta (нет новых изменений)
    await setupEmptyDeltaMock();

    // Выполняем sync
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // После завершения появляется нотификация
    await expect(element(by.id('sync-complete-badge'))).toBeVisible();

    // Текст: "Sync complete" или "All synced"
    await expect(element(by.text('All synced'))).toBeVisible();

    // Badge содержит количество применённых изменений (0 — пустой delta)
    await expect(element(by.id('sync-changes-count'))).toHaveText('0');

    // Восстанавливаем полный mock
    await setupMockRoutes();

    // Повторный sync с данными
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Badge показывает количество изменений
    await expect(element(by.id('sync-complete-badge'))).toBeVisible();
    await expect(element(by.id('sync-changes-count')))
      .toHaveText('3'); // 3 change entries в mockDeltaResponse
  });
});
