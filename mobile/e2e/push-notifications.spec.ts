// ──────────────────────────────────────────────────
// E2E: Push Notifications
//
// Проверяет:
//   - Получение push-уведомлений при новом work order
//   - Получение push при изменении статуса work order
//   - Tap на уведомление → навигация на экран деталей
//   - Отображение badge count на иконке приложения
//   - Deep link из уведомления с параметрами
//   - Permission denied (отказ в разрешениях)
//   - Отображение уведомлений в NotificationCenter
//   - Очистка уведомлений
// ──────────────────────────────────────────────────

import {
  mockWorkOrders,
  mockLoginResponse,
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
      body: { devices: [] },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Push notification register endpoint
  await device.mockRoute(
    { url: `${API_BASE}/mobile/push/register`, method: 'POST' },
    {
      status: 200,
      body: { registered: true, push_token: 'expo-push-token-mock' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Work order complete (for status change notification)
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/complete`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'completed' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Work order start (for status change notification)
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'in_progress' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Notifications list endpoint
  await device.mockRoute(
    { url: `${API_BASE}/mobile/notifications`, method: 'GET' },
    {
      status: 200,
      body: [
        {
          id: 'notif-1',
          type: 'work_order_assigned',
          title: 'New Work Order: Camera Main Entrance',
          body: 'Priority: High — Please review and accept',
          work_order_id: 'wo-001',
          read: false,
          created_at: new Date().toISOString(),
        },
        {
          id: 'notif-2',
          type: 'work_order_status',
          title: 'WO-002 status changed',
          body: 'Status changed from open to in_progress',
          work_order_id: 'wo-002',
          read: false,
          created_at: new Date(Date.now() - 3600000).toISOString(),
        },
        {
          id: 'notif-3',
          type: 'sla_warning',
          title: 'SLA Warning: WO-002',
          body: 'Work order WO-002 is approaching SLA deadline',
          work_order_id: 'wo-002',
          read: true,
          created_at: new Date(Date.now() - 7200000).toISOString(),
        },
      ],
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Mark notification as read
  await device.mockRoute(
    { url: `${API_BASE}/mobile/notifications/*/read`, method: 'POST' },
    {
      status: 200,
      body: { success: true },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Clear all notifications
  await device.mockRoute(
    { url: `${API_BASE}/mobile/notifications/clear`, method: 'DELETE' },
    {
      status: 200,
      body: { success: true },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function teardownMockRoutes(): Promise<void> {
  await device.unmockRoute({ url: `${API_BASE}/mobile/auth/login` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/push/register` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/notifications` });
}

// ── Init ─────────────────────────────────────────

beforeAll(async () => {
  await device.launchApp({
    newInstance: true,
    permissions: { notifications: 'YES' },
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

describe('Push Notifications', () => {
  it('Должен зарегистрироваться на push-уведомления при старте', async () => {
    // При первом запуске приложение регистрируется на push
    await waitForAnimation(3000);

    // Индикатор push-регистрации
    await expect(element(by.id('push-status'))).toBeVisible();
    await expect(element(by.id('push-status-registered'))).toBeVisible();
    await expect(element(by.text('Notifications active'))).toBeVisible();
  });

  it('Должен показать badge count при получении нового уведомления', async () => {
    await waitForAnimation(3000);

    // Badge на иконке tab-bar
    await expect(element(by.id('notifications-badge'))).toBeVisible();
    // Unread count
    await expect(element(by.id('notifications-badge'))).toHaveText('2');
  });

  it('Должен отобразить список уведомлений в NotificationCenter', async () => {
    await waitForAnimation(3000);

    // Переходим в NotificationCenter
    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);

    // Список уведомлений
    await expect(element(by.id('notifications-list'))).toBeVisible();

    // Первое уведомление (новое)
    await expect(element(by.text('New Work Order: Camera Main Entrance'))).toBeVisible();
    await expect(element(by.text('Priority: High — Please review and accept'))).toBeVisible();

    // Второе уведомление
    await expect(element(by.text('WO-002 status changed'))).toBeVisible();

    // Прочитанное уведомление — серый стиль
    await expect(element(by.id('notification-read-SLA Warning: WO-002'))).toBeVisible();
  });

  it('Должен разделять прочитанные и непрочитанные уведомления', async () => {
    await waitForAnimation(3000);

    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);

    // Непрочитанные должны иметь индикатор "new" / жирный текст
    await expect(element(by.id('notification-unread-notif-1'))).toBeVisible();
    await expect(element(by.id('notification-unread-notif-2'))).toBeVisible();

    // Прочитанные — без индикатора
    await expect(element(by.id('notification-read-notif-3'))).toBeVisible();
  });

  it('Должен отметить уведомление как прочитанное при тапе', async () => {
    await waitForAnimation(3000);

    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);

    // Тапаем по непрочитанному
    await element(by.id('notification-unread-notif-1')).tap();
    await waitForAnimation(1000);

    // Статус изменился на прочитанное
    await expect(element(by.id('notification-read-notif-1'))).toBeVisible();
  });

  it('Должен открыть экран work order по тапу на уведомление', async () => {
    await waitForAnimation(3000);

    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);

    // Тапаем на уведомление с work_order_id
    await element(by.id('notification-notif-1')).tap();
    await waitForAnimation(2000);

    // Должны перейти на экран деталей work order
    await expect(element(by.id('work-order-detail-screen'))).toBeVisible();
    await expect(element(by.text('Camera Main Entrance'))).toBeVisible();
  });

  it('Должен обработать deep link из push-уведомления с work_order_id', async () => {
    // Симулируем deep link из push
    await device.launchApp({
      newInstance: true,
      url: 'cctv-technician://work-orders/wo-001',
    });
    await waitForAnimation(3000);

    // Должны сразу открыть экран деталей work order
    await expect(element(by.id('work-order-detail-screen'))).toBeVisible();
    await expect(element(by.text('Camera Main Entrance'))).toBeVisible();
  });

  it('Должен обработать deep link с параметром notification_id', async () => {
    await device.launchApp({
      newInstance: true,
      url: 'cctv-technician://notifications/notif-1',
    });
    await waitForAnimation(3000);

    // Должны открыть NotificationCenter с этим уведомлением
    await expect(element(by.id('notifications-list'))).toBeVisible();
    // Уведомление подсвечено
    await expect(element(by.id('notification-highlighted-notif-1'))).toBeVisible();
  });

  it('Должен показать уведомление о новом work order при логине', async () => {
    await waitForAnimation(3000);

    // После загрузки данных, если есть новые work orders — показываем toast
    await expect(element(by.id('new-work-order-toast'))).toBeVisible();
    await expect(element(by.text('New work order assigned'))).toBeVisible();
  });

  it('Должен обработать отказ в разрешении на уведомления', async () => {
    // Перезапускаем с отказом на notifications
    await device.launchApp({
      newInstance: true,
      permissions: { notifications: 'NO' },
    });
    await waitForAnimation(3000);

    // Статус push — disabled
    await expect(element(by.id('push-status'))).toBeVisible();
    await expect(element(by.id('push-status-disabled'))).toBeVisible();
    await expect(element(by.text('Notifications disabled'))).toBeVisible();

    // Кнопка "Enable Notifications" в настройках
    await expect(element(by.id('enable-notifications-btn'))).toBeVisible();
  });

  it('Должен очистить все уведомления через Clear All', async () => {
    await waitForAnimation(3000);

    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);

    // Кнопка Clear All
    await element(by.id('clear-all-notifications-btn')).tap();
    await waitForAnimation(1000);

    // Список пуст
    await expect(element(by.id('notifications-empty'))).toBeVisible();
    await expect(element(by.text('No notifications'))).toBeVisible();

    // Badge исчез
    await expect(element(by.id('notifications-badge'))).not.toBeVisible();
  });

  it('Должен показать уведомление SLA при приближении дедлайна', async () => {
    await waitForAnimation(3000);

    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);

    // SLA warning notification
    await expect(element(by.text('SLA Warning: WO-002'))).toBeVisible();
    await expect(element(by.text('is approaching SLA deadline'))).toBeVisible();
  });

  it('Должен показать индикатор синхронизации при отправке мутации после уведомления', async () => {
    await waitForAnimation(3000);

    // Переходим к work order через уведомление
    await element(by.id('notifications-tab')).tap();
    await waitForAnimation(2000);
    await element(by.id('notification-notif-1')).tap();
    await waitForAnimation(2000);

    // Offline mode — mutation
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    // Start work order
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(500);

    // Sync pending badge
    await expect(element(by.id('sync-pending-badge'))).toBeVisible();

    // Восстанавливаем сеть
    await device.setURLBlacklist([]);
    await waitForAnimation(3000);

    // Sync complete — badge gone
    await expect(element(by.id('sync-pending-badge'))).not.toBeVisible();
  });
});
