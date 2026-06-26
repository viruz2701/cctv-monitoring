// ──────────────────────────────────────────────────
// E2E: Offline Scenarios
//
// Проверяет:
//   - Работу приложения без сети (offline mode)
//   - Кэширование загруженных work orders
//   - Постановку мутаций в очередь синхронизации
//   - Автоматическую синхронизацию при восстановлении сети
//   - Отображение OfflineIndicator
// ──────────────────────────────────────────────────

import {
  mockWorkOrders,
  mockDevicesForMap,
  mockLoginResponse,
} from './helpers/mockData';
import { waitForElement, waitForAnimation, tapAndWait } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';
const API_ENDPOINTS = [
  `${API_BASE}/mobile/auth/login`,
  `${API_BASE}/mobile/work-orders`,
  `${API_BASE}/mobile/devices/map`,
  `${API_BASE}/mobile/technician/profile`,
];

// ── Helpers ──────────────────────────────────────

/**
 * Настройка mock-маршрутов для offline-тестов.
 * device.mockRoute() перехватывает запросы на уровне native-сети.
 */
async function setupMockRoutes(): Promise<void> {
  // Мок аутентификации
  await device.mockRoute(
    { url: `${API_BASE}/mobile/auth/login`, method: 'POST' },
    {
      status: 200,
      body: mockLoginResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок work orders
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
    {
      status: 200,
      body: mockWorkOrders,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок device map
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    {
      status: 200,
      body: mockDevicesForMap,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок профиля
  await device.mockRoute(
    { url: `${API_BASE}/mobile/technician/profile`, method: 'GET' },
    {
      status: 200,
      body: {
        user_id: 'tech-1',
        user_name: 'John Technician',
        current_workload: 3,
        max_workload: 8,
        skills: ['cctv', 'networking'],
        base_location: 'Minsk, Belarus',
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок завершения work order
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/complete`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'completed' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок старта work order
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'in_progress' },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

/**
 * Очистка mock-маршрутов.
 */
async function teardownMockRoutes(): Promise<void> {
  for (const endpoint of API_ENDPOINTS) {
    try {
      await device.unmockRoute({ url: endpoint });
    } catch {
      // Игнорируем, если маршрут не был замокан
    }
  }
}

// ── Init ─────────────────────────────────────────

beforeAll(async () => {
  await device.launchApp({
    newInstance: true,
    permissions: { location: 'always', camera: 'YES' },
  });
  await setupMockRoutes();
});

afterAll(async () => {
  await teardownMockRoutes();
});

beforeEach(async () => {
  // Перезапускаем с чистым состоянием
  await device.reloadReactNative();
  await waitForAnimation(2000);
});

// ── Tests ────────────────────────────────────────

describe('Offline Scenarios', () => {
  it('Должен показать индикатор оффлайн режима при отсутствии сети', async () => {
    // Симулируем потерю сети через blacklist — все API запросы будут блокироваться
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Проверяем, что OfflineIndicator отображается
    await waitForElement(element(by.id('offline-indicator')));
    await expect(element(by.id('offline-indicator'))).toBeVisible();

    // Текст оффлайн статуса
    await expect(element(by.text('No internet connection'))).toBeVisible();
    await expect(element(by.text('Offline'))).toBeVisible();

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
  });

  it('Должен кэшировать work orders локально при онлайн загрузке', async () => {
    // Даём время на загрузку данных и кэширование
    await waitForAnimation(3000);

    // Переходим в offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Work orders должны быть видны из кэша
    await expect(element(by.text('Camera Main Entrance'))).toBeVisible();
    await expect(element(by.text('Camera Parking Lot'))).toBeVisible();
  });

  it('Должен ставить мутации в очередь при оффлайн статусе', async () => {
    // Загружаем данные
    await waitForAnimation(3000);

    // Переходим в offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Открываем work order
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);

    // Пытаемся сменить статус на "in_progress"
    await element(by.id('start-work-order-btn')).tap();

    // Валидация: mutation добавлена в очередь
    // Иконка/бадж pending sync должен появиться
    await expect(element(by.id('sync-pending-badge'))).toBeVisible();
    await expect(element(by.text('1 pending'))).toBeVisible();
  });

  it('Должен автоматически синхронизировать очередь при восстановлении сети', async () => {
    // Загружаем данные
    await waitForAnimation(3000);

    // Переходим в offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Создаём mutation (start work order)
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Проверяем pending badge
    await expect(element(by.id('sync-pending-badge'))).toBeVisible();

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // После синхронизации pending badge должен исчезнуть
    await expect(element(by.id('sync-pending-badge'))).not.toBeVisible();

    // Индикатор сети показывает online
    await expect(element(by.id('online-indicator'))).toBeVisible();
    await expect(element(by.text('Online'))).toBeVisible();
  });

  it('Должен показывать ошибку синхронизации при недоступном сервере', async () => {
    // Загружаем данные
    await waitForAnimation(3000);

    // Симулируем недоступность сервера (500 ошибка на complete)
    await device.mockRoute(
      { url: `${API_BASE}/mobile/work-orders/*/complete`, method: 'POST' },
      {
        status: 500,
        body: { error: 'Internal Server Error' },
        headers: { 'Content-Type': 'application/json' },
      },
    );

    // Создаём mutation
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Должен отобразиться индикатор ошибки синхронизации
    await expect(element(by.id('sync-error-badge'))).toBeVisible();
    await expect(element(by.text('Sync error'))).toBeVisible();
  });

  it('Должен корректно отображать количество pending мутаций', async () => {
    // Загружаем данные
    await waitForAnimation(3000);

    // Offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Добавляем несколько мутаций
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Проверяем счётчик
    await expect(element(by.id('sync-pending-count'))).toHaveText('1');

    // Добавляем ещё одну
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);
    await element(by.text('Camera Parking Lot')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Счётчик должен быть 2
    await expect(element(by.id('sync-pending-count'))).toHaveText('2');
  });

  it('Должен синхронизировать при возврате в foreground', async () => {
    // Загружаем данные
    await waitForAnimation(3000);

    // Offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Добавляем mutation
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Отправляем в background и возвращаем
    await device.sendToHome();
    await waitForAnimation(2000);
    await device.launchApp({ newInstance: false });
    await waitForAnimation(3000);

    // После возвращения — всё ещё offline, pending сохраняется
    await expect(element(by.id('sync-pending-badge'))).toBeVisible();

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Должна произойти синхронизация
    await expect(element(by.id('sync-pending-badge'))).not.toBeVisible();
  });

  it('Должен показывать DiffView при синхронизации с конфликтом', async () => {
    // Загружаем начальные данные
    await waitForAnimation(3000);

    // Offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Меняем work order локально
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Fixed cable issue on site');

    // Восстанавливаем сеть — будет конфликт, т.к. серверная версия отличается
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Должно появиться модальное окно конфликта
    await expect(element(by.id('conflict-modal'))).toBeVisible();
    await expect(element(by.text('Sync Conflicts'))).toBeVisible();
  });
});
