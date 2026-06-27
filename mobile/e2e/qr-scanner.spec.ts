// ──────────────────────────────────────────────────
// E2E: QR Scanner
//
// Проверяет:
//   - Открытие QRScannerScreen
//   - Разрешение на камеру
//   - Сканирование QR-кода с asset tag
//   - Распознавание формата cctv-asset JSON
//   - Навигация на устройство после сканирования
//   - Обработка невалидного QR-кода
//   - Повторное сканирование после успеха
//   - Ручной ввод ID устройства
//   - Flash toggle
// ──────────────────────────────────────────────────

import { mockLoginResponse } from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Моковые QR-данные ────────────────────────────

const MOCK_VALID_QR_DATA = JSON.stringify({
  type: 'cctv-asset',
  model: 'DS-2CD2386G2-I',
  vendor: 'hikvision',
  ip: '192.168.1.100',
  mac: 'AA:BB:CC:DD:EE:FF',
  generated: '2026-06-25T12:00:00.000Z',
});

const MOCK_INVALID_QR_DATA = 'not-a-valid-qr-code';

const MOCK_WORK_ORDER_QR = JSON.stringify({
  type: 'cctv-work-order',
  work_order_id: 'wo-001',
  device_id: 'cam-101',
  site: 'Facility A',
});

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
      body: [],
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Device detail
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/cam-101`, method: 'GET' },
    {
      status: 200,
      body: {
        device_id: 'cam-101',
        name: 'Camera Main Entrance',
        model: 'DS-2CD2386G2-I',
        vendor: 'hikvision',
        ip_address: '192.168.1.100',
        mac_address: 'AA:BB:CC:DD:EE:FF',
        status: 'ONLINE',
        health: 'healthy',
        site_name: 'Facility A',
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
}

async function teardownMockRoutes(): Promise<void> {
  await device.unmockRoute({ url: `${API_BASE}/mobile/auth/login` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/devices/cam-101` });
}

// ── Init ─────────────────────────────────────────

beforeAll(async () => {
  await device.launchApp({
    newInstance: true,
    permissions: { camera: 'YES' },
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

describe('QR Scanner', () => {
  it('Должен открыть QRScannerScreen через bottom tab', async () => {
    await waitForAnimation(3000);

    // Тапаем на иконку QR Scanner в tab-bar
    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Экран сканера открыт
    await expect(element(by.id('qr-scanner-screen'))).toBeVisible();
    await expect(element(by.id('scanner-viewport'))).toBeVisible();
  });

  it('Должен показать инструкцию на экране сканера', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Инструкция
    await expect(element(by.text('Scan asset QR code'))).toBeVisible();
    await expect(element(by.text('Point the camera at the asset tag')))
      .toBeVisible();
  });

  it('Должен обработать сканирование валидного cctv-asset QR', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Симулируем сканирование QR-кода через Device API
    // В Detox: device.mockShallowCopy не поддерживает камеру,
    // поэтому используем device.matchQRCode() (условный API)
    // Если Detox не поддерживает — вводим данные вручную
    await device.matchQRCode?.({
      data: MOCK_VALID_QR_DATA,
      type: 'qr',
    });
    await waitForAnimation(2000);

    // Распознанное устройство
    await expect(element(by.id('scanned-device-info'))).toBeVisible();
    await expect(element(by.text('DS-2CD2386G2-I'))).toBeVisible();
    await expect(element(by.text('hikvision'))).toBeVisible();
    await expect(element(by.text('192.168.1.100'))).toBeVisible();
    await expect(element(by.text('AA:BB:CC:DD:EE:FF'))).toBeVisible();
  });

  it('Должен показать кнопку "View Device" после успешного сканирования', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Сканируем
    await device.matchQRCode?.({
      data: MOCK_VALID_QR_DATA,
      type: 'qr',
    });
    await waitForAnimation(2000);

    // Кнопка для перехода к устройству
    await expect(element(by.id('view-device-btn'))).toBeVisible();
    await expect(element(by.text('View Device'))).toBeVisible();
  });

  it('Должен перейти на экран устройства по кнопке View Device', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    await device.matchQRCode?.({
      data: MOCK_VALID_QR_DATA,
      type: 'qr',
    });
    await waitForAnimation(2000);

    // Переход
    await element(by.id('view-device-btn')).tap();
    await waitForAnimation(2000);

    // Экран устройства
    await expect(element(by.id('device-detail-screen'))).toBeVisible();
    await expect(element(by.text('Camera Main Entrance'))).toBeVisible();
  });

  it('Должен показать ошибку для невалидного QR-кода', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Сканируем невалидный QR
    await device.matchQRCode?.({
      data: MOCK_INVALID_QR_DATA,
      type: 'qr',
    });
    await waitForAnimation(1000);

    // Ошибка
    await expect(element(by.id('qr-scan-error'))).toBeVisible();
    await expect(element(by.text('Invalid QR code'))).toBeVisible();
    await expect(element(by.text('Format not recognized'))).toBeVisible();
  });

  it('Должен предложить повторное сканирование после ошибки', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Невалидный QR
    await device.matchQRCode?.({
      data: MOCK_INVALID_QR_DATA,
      type: 'qr',
    });
    await waitForAnimation(1000);

    // Кнопка "Scan Again"
    await expect(element(by.id('scan-again-btn'))).toBeVisible();

    // Тапаем Scan Again
    await element(by.id('scan-again-btn')).tap();
    await waitForAnimation(1000);

    // Сканнер снова активен
    await expect(element(by.id('scanner-viewport'))).toBeVisible();
    // Ошибка скрыта
    await expect(element(by.id('qr-scan-error'))).not.toBeVisible();
  });

  it('Должен обработать work order QR-код', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Сканируем work order QR
    await device.matchQRCode?.({
      data: MOCK_WORK_ORDER_QR,
      type: 'qr',
    });
    await waitForAnimation(2000);

    // Распознан work order
    await expect(element(by.id('scanned-workorder-info'))).toBeVisible();
    await expect(element(by.text('Work Order: WO-001'))).toBeVisible();
    await expect(element(by.text('Facility A'))).toBeVisible();

    // Кнопка перехода
    await expect(element(by.id('view-workorder-btn'))).toBeVisible();
  });

  it('Должен позволить ручной ввод ID устройства', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Переключаемся на ручной ввод
    await element(by.id('manual-input-toggle')).tap();
    await waitForAnimation(500);

    // Поле ввода
    await expect(element(by.id('device-id-input'))).toBeVisible();

    // Вводим ID
    await element(by.id('device-id-input')).tap();
    await element(by.id('device-id-input')).typeText('cam-101');

    // Submit
    await element(by.id('search-device-btn')).tap();
    await waitForAnimation(2000);

    // Должны перейти на устройство
    await expect(element(by.id('device-detail-screen'))).toBeVisible();
  });

  it('Должен показать ошибку при ручном вводе несуществующего ID', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    await element(by.id('manual-input-toggle')).tap();
    await waitForAnimation(500);

    // Невалидный ID
    await element(by.id('device-id-input')).tap();
    await element(by.id('device-id-input')).typeText('nonexistent-device');

    // Submit
    await element(by.id('search-device-btn')).tap();
    await waitForAnimation(1000);

    // Ошибка
    await expect(element(by.id('device-not-found-error'))).toBeVisible();
    await expect(element(by.text('Device not found'))).toBeVisible();
  });

  it('Должен поддерживать flash toggle на камере', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Flash toggle
    await element(by.id('flash-toggle-btn')).tap();
    await waitForAnimation(500);

    // Flash icon изменилась
    await expect(element(by.id('flash-on-icon'))).toBeVisible();

    // Выключаем
    await element(by.id('flash-toggle-btn')).tap();
    await waitForAnimation(500);

    await expect(element(by.id('flash-off-icon'))).toBeVisible();
  });

  it('Должен закрыть сканер по кнопке Close', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Close
    await element(by.id('close-scanner-btn')).tap();
    await waitForAnimation(1000);

    // Вернулись на Dashboard
    await expect(element(by.id('dashboard-screen'))).toBeVisible();
  });

  it('Должен сохранять историю сканирований', async () => {
    await waitForAnimation(3000);

    await element(by.id('qr-scanner-tab')).tap();
    await waitForAnimation(2000);

    // Сканируем валидный QR
    await device.matchQRCode?.({
      data: MOCK_VALID_QR_DATA,
      type: 'qr',
    });
    await waitForAnimation(1500);

    // Возвращаемся на сканер
    await element(by.id('scan-again-btn')).tap();
    await waitForAnimation(500);

    // Открываем историю
    await element(by.id('scan-history-btn')).tap();
    await waitForAnimation(1000);

    // История содержит предыдущее сканирование
    await expect(element(by.id('scan-history-list'))).toBeVisible();
    await expect(element(by.text('DS-2CD2386G2-I'))).toBeVisible();
    await expect(element(by.text('192.168.1.100'))).toBeVisible();
  });
});
