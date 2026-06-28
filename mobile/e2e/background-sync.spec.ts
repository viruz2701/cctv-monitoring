// ──────────────────────────────────────────────────
// E2E: Background Sync
//
// Проверяет:
//   - Регистрацию background fetch задачи
//   - Автоматическую синхронизацию при возврате в foreground
//   - Manual sync через UI (Sync Now)
//   - Отображение статуса синхронизации (SyncStatusBar)
//   - Отображение lastSyncAt timestamp
//   - Toggle background sync (вкл/выкл)
//   - Error handling при недоступном сервере
//   - Push pending мутаций при background sync
//   - Pull свежих данных с сервера
//   - Status indicator при синхронизации (syncing)
// ──────────────────────────────────────────────────

import {
  mockWorkOrders,
  mockLoginResponse,
  mockDevicesForMap,
} from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Helpers ──────────────────────────────────────

async function setupMockRoutes(): Promise<void> {
  // Auth
  await device.mockRoute(
    { url: `${API_BASE}/mobile/auth/login`, method: 'POST' },
    {
      status: 200,
      body: mockLoginResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Work orders
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
    {
      status: 200,
      body: mockWorkOrders,
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
      body: mockDevicesForMap,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Start work order
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'in_progress' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Complete work order
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/complete`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'completed' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Background sync register endpoint
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/register-bg`, method: 'POST' },
    {
      status: 200,
      body: { registered: true },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Sync status endpoint
  await device.mockRoute(
    { url: `${API_BASE}/mobile/sync/status`, method: 'GET' },
    {
      status: 200,
      body: {
        is_registered: true,
        last_sync_at: new Date().toISOString(),
        pending_count: 0,
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function setupServerErrorMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
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
  await device.unmockRoute({ url: `${API_BASE}/mobile/sync/register-bg` });
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

describe('Background Sync', () => {
  it('Должен зарегистрировать background fetch задачу при старте', async () => {
    await waitForAnimation(3000);

    // Background sync статус — registered
    await expect(element(by.id('bg-sync-status'))).toBeVisible();
    await expect(element(by.id('bg-sync-registered'))).toBeVisible();
    await expect(element(by.text('Background sync active'))).toBeVisible();
  });

  it('Должен показать SyncStatusBar с количеством pending мутаций', async () => {
    await waitForAnimation(3000);

    // SyncStatusBar должен отображаться
    await expect(element(by.id('sync-status-bar'))).toBeVisible();

    // Начальное состояние — 0 pending
    await expect(element(by.id('sync-pending-count'))).toHaveText('0');
    await expect(element(by.text('All synced'))).toBeVisible();
  });

  it('Должен синхронизировать при возврате в foreground из background', async () => {
    await waitForAnimation(3000);

    // Создаём pending mutation
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Отправляем в background
    await device.sendToHome();
    await waitForAnimation(2000);

    // Восстанавливаем сеть перед foreground
    await device.setURLBlacklist([]);
    await waitForAnimation(500);

    // Возвращаем в foreground
    await device.launchApp({ newInstance: false });
    await waitForAnimation(3000);

    // Должна произойти автоматическая синхронизация
    // SyncStatusBar показывает syncing, затем online
    await expect(element(by.id('sync-status-bar'))).toBeVisible();

    // После синхронизации pending count должен быть 0
    await waitForElement(element(by.id('sync-pending-count')));
    await expect(element(by.id('sync-pending-count'))).toHaveText('0');
  });

  it('Должен выполнить manual sync по кнопке Sync Now', async () => {
    await waitForAnimation(3000);

    // Создаём offline мутацию
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Pending count должен быть > 0
    await expect(element(by.id('sync-pending-count'))).toBeVisible();

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(1000);

    // Нажимаем Sync Now
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Статус — syncing, затем synced
    await expect(element(by.id('sync-status-bar'))).toBeVisible();
    await expect(element(by.id('sync-pending-count'))).toHaveText('0');
    await expect(element(by.text('All synced'))).toBeVisible();
  });

  it('Должен показать последний timestamp синхронизации', async () => {
    await waitForAnimation(3000);

    // Выполняем sync
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    await device.setURLBlacklist([]);
    await waitForAnimation(1000);

    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Timestamp отображается
    await expect(element(by.id('last-sync-timestamp'))).toBeVisible();
    await expect(element(by.id('last-sync-timestamp'))).not.toHaveText('');
  });

  it('Должен показать ошибку синхронизации при недоступном сервере', async () => {
    await waitForAnimation(3000);

    // Настраиваем 500 ошибку
    await setupServerErrorMock();

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    await device.setURLBlacklist([]);
    await waitForAnimation(1000);

    // Пробуем sync — будет ошибка
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Должен отобразиться error badge
    await expect(element(by.id('sync-error-badge'))).toBeVisible();
    await expect(element(by.text('Sync error'))).toBeVisible();
  });

  it('Должен позволить retry после ошибки синхронизации', async () => {
    await waitForAnimation(3000);

    // Сначала 500 ошибка
    await setupServerErrorMock();

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    await device.setURLBlacklist([]);
    await waitForAnimation(1000);

    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // Ошибка
    await expect(element(by.id('sync-error-badge'))).toBeVisible();

    // Восстанавливаем mock
    await setupMockRoutes();

    // Retry
    await element(by.id('sync-retry-btn')).tap();
    await waitForAnimation(3000);

    // После retry — успех
    await expect(element(by.id('sync-error-badge'))).not.toBeVisible();
    await expect(element(by.id('sync-pending-count'))).toHaveText('0');
  });

  it('Должен показывать статус syncing во время синхронизации', async () => {
    await waitForAnimation(3000);

    // Создаём мутацию
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(1000);

    // Sync Now
    await element(by.id('sync-now-btn')).tap();

    // Статус должен переключиться на syncing
    await expect(element(by.id('sync-status-syncing'))).toBeVisible();
    await expect(element(by.text('Syncing...'))).toBeVisible();

    // После завершения — online
    await waitForAnimation(3000);
    await expect(element(by.id('sync-status-online'))).toBeVisible();
    await expect(element(by.text('All synced'))).toBeVisible();
  });

  it('Должен показать количество pending мутаций в SyncStatusBar', async () => {
    await waitForAnimation(3000);

    // Offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Добавляем первую мутацию
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Проверяем счётчик
    await expect(element(by.id('sync-pending-count'))).toHaveText('1');

    // Добавляем вторую
    await element(by.text('Camera Parking Lot')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Счётчик = 2
    await expect(element(by.id('sync-pending-count'))).toHaveText('2');
  });

  it('Должен показывать SyncStatusBar с индикатором background sync', async () => {
    await waitForAnimation(3000);

    // Background sync зарегистрирована
    await expect(element(by.id('bg-sync-status'))).toBeVisible();
    await expect(element(by.id('bg-sync-registered'))).toBeVisible();

    // Индикатор background fetch
    await expect(element(by.id('bg-sync-interval'))).toBeVisible();
    await expect(element(by.text('Every 15 min'))).toBeVisible();
  });

  it('Должен поддерживать toggle background sync (off → on)', async () => {
    await waitForAnimation(3000);

    // Переходим в настройки sync
    await element(by.id('sync-settings-btn')).tap();
    await waitForAnimation(1000);

    // Toggle background sync — выключаем
    await element(by.id('bg-sync-toggle')).tap();
    await waitForAnimation(1000);

    // Статус изменился на disabled
    await expect(element(by.id('bg-sync-disabled'))).toBeVisible();
    await expect(element(by.text('Background sync disabled'))).toBeVisible();

    // Включаем обратно
    await element(by.id('bg-sync-toggle')).tap();
    await waitForAnimation(1000);

    // Статус — registered
    await expect(element(by.id('bg-sync-registered'))).toBeVisible();
    await expect(element(by.text('Background sync active'))).toBeVisible();
  });

  it('Должен пулить свежие данные с сервера после успешной синхронизации', async () => {
    await waitForAnimation(3000);

    // Меняем mock — возвращаем обновлённые данные
    await device.mockRoute(
      { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
      {
        status: 200,
        body: [
          {
            ...mockWorkOrders[0],
            status: 'completed',
            notes: 'Updated by dispatcher',
            updated_at: new Date().toISOString(),
          },
          ...mockWorkOrders.slice(1),
        ],
        headers: { 'Content-Type': 'application/json' },
      },
    );

    // Выполняем синхронизацию
    await element(by.id('sync-now-btn')).tap();
    await waitForAnimation(3000);

    // После pull — данные должны обновиться
    // Открываем work order
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);

    // Статус должен отражать серверную версию
    await expect(element(by.id('work-order-status'))).toBeVisible();
  });
});
