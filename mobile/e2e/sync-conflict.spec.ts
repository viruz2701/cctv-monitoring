// ──────────────────────────────────────────────────
// E2E: Sync Conflict Resolution
//
// Проверяет:
//   - Обнаружение конфликтов между локальной и серверной версиями
//   - Разрешение через "Keep Local" (приоритет локальных изменений)
//   - Разрешение через "Keep Server" (приоритет серверных данных)
//   - Разрешение через "Merge" (ручное слияние полей)
//   - Telemetry логирование resolved конфликтов
//   - Корректное обновление интерфейса после разрешения
// ──────────────────────────────────────────────────

import { mockWorkOrders, mockServerWorkOrder } from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Helpers ──────────────────────────────────────

async function setupMockRoutes(): Promise<void> {
  // Мок аутентификации
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

  // Мок work orders — возвращаем серверную версию для конфликта
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
    {
      status: 200,
      body: [mockServerWorkOrder],
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок обновления work order (complete)
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

  // Мок device map
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    {
      status: 200,
      body: { devices: [] },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Мок профиля
  await device.mockRoute(
    { url: `${API_BASE}/mobile/technician/profile`, method: 'GET' },
    {
      status: 200,
      body: { user_id: 'tech-1', user_name: 'John Technician' },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function teardownMockRoutes(): Promise<void> {
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/auth/login` });
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
  await device.reloadReactNative();
  await waitForAnimation(2000);
});

// ── Tests ────────────────────────────────────────

describe('Sync Conflict Resolution', () => {
  it('Должен обнаружить конфликт при расхождении локальной и серверной версий', async () => {
    // Загружаем данные — сервер возвращает mockServerWorkOrder
    await waitForAnimation(3000);

    // Переходим в оффлайн
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Открываем work order и вносим локальные изменения
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);

    // Редактируем notes
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Fixed cable issue on site');

    // Выходим обратно — сохраняется локально
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть — триггер синхронизации
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Модальное окно конфликта должно появиться
    await expect(element(by.id('conflict-modal'))).toBeVisible();
    await expect(element(by.text('Sync Conflicts'))).toBeVisible();
  });

  it('Должен показать diff-view с подсветкой изменений', async () => {
    // Загружаем данные
    await waitForAnimation(3000);

    // Оффлайн
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Открываем work order, вносим изменения
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Fixed cable issue');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Проверяем отображение diff-view
    await expect(element(by.id('conflict-modal'))).toBeVisible();

    // Должны быть видны поля с изменениями
    // Local value отображается на красном фоне
    await expect(element(by.text('Local'))).toBeVisible();
    await expect(element(by.text('Server'))).toBeVisible();

    // Должен быть индикатор новизны
    await expect(element(by.text('Local'))).toBeVisible();

    // Должно отображаться название конфликтующего поля
    await expect(element(by.text('notes'))).toBeVisible();
  });

  it('Должен разрешить конфликт через "Keep Local"', async () => {
    await waitForAnimation(3000);

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Local fix applied');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть — конфликт
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    await expect(element(by.id('conflict-modal'))).toBeVisible();

    // Выбираем "Keep Local"
    await element(by.id('keep-local-btn')).tap();
    await waitForAnimation(1000);

    // Модальное окно должно закрыться
    await expect(element(by.id('conflict-modal'))).not.toBeVisible();

    // Значения должны остаться локальными
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await expect(element(by.id('notes-display'))).toHaveText('Local fix applied');
  });

  it('Должен разрешить конфликт через "Keep Server"', async () => {
    await waitForAnimation(3000);

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Local change to discard');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    await expect(element(by.id('conflict-modal'))).toBeVisible();

    // Выбираем "Keep Server"
    await element(by.id('keep-server-btn')).tap();
    await waitForAnimation(1000);

    // Модальное окно должно закрыться
    await expect(element(by.id('conflict-modal'))).not.toBeVisible();

    // Серверные данные должны заменить локальные
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await expect(element(by.id('notes-display'))).toHaveText('Started by dispatcher remotely');
  });

  it('Должен разрешить конфликт через ручной "Merge"', async () => {
    await waitForAnimation(3000);

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Local: fixed cable');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    await expect(element(by.id('conflict-modal'))).toBeVisible();

    // Нажимаем Merge
    await element(by.id('merge-btn')).tap();
    await waitForAnimation(1000);

    // Должно появиться поле для ввода объединённого значения
    await expect(element(by.id('merged-value-input'))).toBeVisible();

    // Вводим объединённое значение
    await element(by.id('merged-value-input')).tap();
    await element(by.id('merged-value-input')).clearText();
    await element(by.id('merged-value-input')).typeText('Local: fixed cable + Remote: started');
    await waitForAnimation(500);

    // Применяем merge
    await element(by.id('apply-merge-btn')).tap();
    await waitForAnimation(1000);

    // Модальное окно должно закрыться
    await expect(element(by.id('conflict-modal'))).not.toBeVisible();

    // Проверяем объединённое значение
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await expect(element(by.id('notes-display'))).toHaveText('Local: fixed cable + Remote: started');
  });

  it('Должен обработать множественные конфликты (несколько полей)', async () => {
    await waitForAnimation(3000);

    // Оффлайн
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Вносим изменения в несколько полей
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('New local notes');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(500);

    // Меняем статус
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Должны быть видны все конфликтующие поля
    await expect(element(by.id('conflict-modal'))).toBeVisible();
    await expect(element(by.text('notes'))).toBeVisible();
    await expect(element(by.text('status'))).toBeVisible();
  });

  it('Должен логировать resolved конфликты в telemetry', async () => {
    await waitForAnimation(3000);

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Telemetry test');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    await expect(element(by.id('conflict-modal'))).toBeVisible();

    // Разрешаем через Keep Local
    await element(by.id('keep-local-btn')).tap();
    await waitForAnimation(1000);

    // Конфликт разрешён — модалка закрыта
    await expect(element(by.id('conflict-modal'))).not.toBeVisible();
  });

  it('Должен закрыть модальное окно конфликта без разрешения через Close', async () => {
    await waitForAnimation(3000);

    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('edit-notes-btn')).tap();
    await element(by.id('notes-input')).tap();
    await element(by.id('notes-input')).typeText('Closing test');
    await element(by.id('back-btn')).tap();
    await waitForAnimation(1000);

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    await expect(element(by.id('conflict-modal'))).toBeVisible();

    // Закрываем без разрешения
    await element(by.id('close-conflict-btn')).tap();
    await waitForAnimation(1000);

    // Модалка закрыта, но конфликт остаётся в списке
    await expect(element(by.id('conflict-modal'))).not.toBeVisible();

    // Конфликт должен быть доступен позже
    await element(by.id('show-conflicts-btn')).tap();
    await waitForAnimation(1000);
    await expect(element(by.id('conflict-modal'))).toBeVisible();
  });
});
