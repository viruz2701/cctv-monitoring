// ──────────────────────────────────────────────────
// E2E: Photo Upload with Gatekeeper Verification
//
// Проверяет:
//   - Загрузку фото через камеру/галерею
//   - Проверку GPS-координат (geofence verification)
//   - Проверку EXIF-данных (timestamp, camera info)
//   - AI сравнение "before/after" фото
//   - Отображение статусов GPS/EXIF/Similarity
//   - Error handling при failed verification
//   - Retry механизм при ошибке загрузки
// ──────────────────────────────────────────────────

import {
  mockWorkOrders,
  mockLoginResponse,
  mockVerificationResponse,
  mockVerificationGpsFailed,
  mockVerificationExifFailed,
} from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Helpers ──────────────────────────────────────

async function setupMockRoutes(): Promise<void> {
  // Auth mock
  await device.mockRoute(
    { url: `${API_BASE}/mobile/auth/login`, method: 'POST' },
    {
      status: 200,
      body: mockLoginResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Work orders mock
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders`, method: 'GET' },
    {
      status: 200,
      body: mockWorkOrders,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Device map mock
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    {
      status: 200,
      body: { devices: [] },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Profile mock
  await device.mockRoute(
    { url: `${API_BASE}/mobile/technician/profile`, method: 'GET' },
    {
      status: 200,
      body: { user_id: 'tech-1', user_name: 'John Technician' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Photo upload mock — success
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/photos`, method: 'POST' },
    {
      status: 200,
      body: { url: 'https://cdn.cctv-monitoring.com/photos/wo-001/photo_001.jpg' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Start work order mock
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'in_progress' },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Complete work order mock (default — success)
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/complete`, method: 'POST' },
    {
      status: 200,
      body: {
        ...mockWorkOrders[0],
        status: 'completed',
        verification: mockVerificationResponse,
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Gatekeeper verify mock (default — success)
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/verify`, method: 'POST' },
    {
      status: 200,
      body: mockVerificationResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function setupGpsFailMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/verify`, method: 'POST' },
    {
      status: 200,
      body: mockVerificationGpsFailed,
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function setupExifFailMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/verify`, method: 'POST' },
    {
      status: 200,
      body: mockVerificationExifFailed,
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function setupServerErrorMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/verify`, method: 'POST' },
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
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders/*/photos` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders/*/verify` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders/*/complete` });
}

// ── Init ─────────────────────────────────────────

beforeAll(async () => {
  await device.launchApp({
    newInstance: true,
    permissions: { location: 'always', camera: 'YES', photo: 'YES' },
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

describe('Photo Upload with Gatekeeper', () => {
  it('Должен загрузить фото через камеру при завершении work order', async () => {
    await waitForAnimation(3000);

    // Открываем work order
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);

    // Нажимаем Complete
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);

    // Открывается CompleteWorkOrderWizard
    await expect(element(by.id('complete-wizard'))).toBeVisible();

    // Нажимаем "Take Photo"
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Фото загружено — проверяем статус загрузки
    await expect(element(by.id('photo-upload-progress'))).toBeVisible();
    await expect(element(by.id('photo-upload-success'))).toBeVisible();
  });

  it('Должен отобразить статус GPS-проверки после загрузки фото', async () => {
    await waitForAnimation(3000);

    // Настраиваем успешный GPS
    await setupMockRoutes();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Проверяем GPS статус в verification секции
    await expect(element(by.id('gps-status'))).toBeVisible();
    await expect(element(by.id('gps-status-pass'))).toBeVisible();
    await expect(element(by.text('GPS: Verified'))).toBeVisible();
    await expect(element(by.text('Distance: 2.5m'))).toBeVisible();
  });

  it('Должен показать ошибку GPS при выходе за геозону', async () => {
    await waitForAnimation(3000);

    await setupGpsFailMock();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Проверяем GPS статус — failed
    await expect(element(by.id('gps-status'))).toBeVisible();
    await expect(element(by.id('gps-status-fail'))).toBeVisible();
    await expect(element(by.text('GPS: Failed'))).toBeVisible();
    await expect(element(by.text('Outside geofence (150m)'))).toBeVisible();
  });

  it('Должен отобразить статус EXIF-проверки после загрузки фото', async () => {
    await waitForAnimation(3000);

    await setupMockRoutes();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Проверяем EXIF статус
    await expect(element(by.id('exif-status'))).toBeVisible();
    await expect(element(by.id('exif-status-pass'))).toBeVisible();
    await expect(element(by.text('EXIF: Verified'))).toBeVisible();
  });

  it('Должен показать ошибку EXIF при отсутствии метаданных', async () => {
    await waitForAnimation(3000);

    await setupExifFailMock();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Проверяем EXIF статус — failed
    await expect(element(by.id('exif-status'))).toBeVisible();
    await expect(element(by.id('exif-status-fail'))).toBeVisible();
    await expect(element(by.text('EXIF: Failed'))).toBeVisible();
    await expect(element(by.text('EXIF data missing'))).toBeVisible();
  });

  it('Должен отобразить AI similarity score после загрузки фото', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Проверяем AI Score
    await expect(element(by.id('ai-score'))).toBeVisible();
    await expect(element(by.id('ai-similarity-bar'))).toBeVisible();
    await expect(element(by.text('97%'))).toBeVisible();
    await expect(element(by.text('Change detected'))).toBeVisible();
  });

  it('Должен показать все статусы Gatekeeper в сводке (GPS + EXIF + AI)', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Сводка verification
    await expect(element(by.id('gatekeeper-summary'))).toBeVisible();

    // Все три статуса
    await expect(element(by.id('gps-status'))).toBeVisible();
    await expect(element(by.id('exif-status'))).toBeVisible();
    await expect(element(by.id('ai-score'))).toBeVisible();

    // Общий статус — Passed
    await expect(element(by.id('verification-passed-badge'))).toBeVisible();
    await expect(element(by.text('Verification Passed'))).toBeVisible();
  });

  it('Должен показать сводку с ошибками при failed verification', async () => {
    await waitForAnimation(3000);

    await setupExifFailMock();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Общий статус — Failed
    await expect(element(by.id('verification-failed-badge'))).toBeVisible();
    await expect(element(by.text('Verification Failed'))).toBeVisible();

    // Список причин отказа
    await expect(element(by.id('fail-reasons-list'))).toBeVisible();
    await expect(element(by.text('EXIF data missing'))).toBeVisible();
  });

  it('Должен отправить verification request на сервер при нажатии Submit', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Заполняем чеклист
    await element(by.id('checklist-item-0')).tap();
    await element(by.id('checklist-item-1')).tap();
    await waitForAnimation(500);

    // Добавляем notes
    await element(by.id('complete-notes-input')).tap();
    await element(by.id('complete-notes-input')).typeText('All checks completed');

    // Нажимаем Submit
    await element(by.id('submit-verification-btn')).tap();
    await waitForAnimation(3000);

    // Должны вернуться на Dashboard
    await expect(element(by.id('dashboard-screen'))).toBeVisible();
  });

  it('Должен показать error toast при недоступности сервера', async () => {
    await waitForAnimation(3000);

    await setupServerErrorMock();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Заполняем и submit
    await element(by.id('submit-verification-btn')).tap();
    await waitForAnimation(2000);

    // Должен появиться toast с ошибкой
    await expect(element(by.id('error-toast'))).toBeVisible();
    await expect(element(by.text('Verification failed'))).toBeVisible();
  });

  it('Должен позволить retry после ошибки verification', async () => {
    await waitForAnimation(3000);

    // Сначала настраиваем ошибку
    await setupServerErrorMock();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    await element(by.id('submit-verification-btn')).tap();
    await waitForAnimation(2000);

    // Toast с ошибкой
    await expect(element(by.id('error-toast'))).toBeVisible();

    // Меняем mock на успешный
    await setupMockRoutes();

    // Нажимаем Retry
    await element(by.id('retry-btn')).tap();
    await waitForAnimation(3000);

    // Должны вернуться на Dashboard
    await expect(element(by.id('dashboard-screen'))).toBeVisible();
  });

  it('Должен показать verification_token после успешной проверки', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);

    // Проверяем token
    await expect(element(by.id('verification-token'))).toBeVisible();
    await expect(element(by.id('verification-token'))).toHaveText('Token: vrf_tkn_test_token');
  });

  it('Должен загрузить фото из галереи (не только камера)', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);

    // Выбираем фото из галереи
    await element(by.id('choose-from-gallery-btn')).tap();
    await waitForAnimation(2000);

    // Фото загружено
    await expect(element(by.id('photo-upload-success'))).toBeVisible();
  });

  it('Должен загрузить несколько фото для before/after сравнения', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('complete-work-order-btn')).tap();
    await waitForAnimation(2000);

    // Загружаем "before" фото
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);
    await expect(element(by.id('photo-upload-success'))).toBeVisible();

    // Загружаем "after" фото
    await element(by.id('take-photo-btn')).tap();
    await waitForAnimation(2000);
    await expect(element(by.id('photo-upload-success'))).toBeVisible();

    // Должны быть видны оба фото
    await expect(element(by.id('photo-before-preview'))).toBeVisible();
    await expect(element(by.id('photo-after-preview'))).toBeVisible();
  });
});
