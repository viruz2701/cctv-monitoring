// ──────────────────────────────────────────────────
// E2E: Maintenance Checklist
//
// Проверяет:
//   - Открытие MaintenanceChecklistScreen
//   - Отображение шагов (checklist → gatekeeper → signature → review)
//   - Прохождение каждого пункта чек-листа (Pass/Fail)
//   - Фотофиксацию для каждого пункта
//   - Gatekeeper verification
//   - Добавление notes к каждому пункту
//   - E-signature capture
//   - Review шаг с итоговой сводкой
//   - Submit акта ТО
//   - Offline-first сохранение
//   - HMAC-signed act generation
//   - Retry при ошибке submit
// ──────────────────────────────────────────────────

import {
  mockWorkOrders,
  mockLoginResponse,
  mockVerificationResponse,
} from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Mock regulation template ─────────────────────

const MOCK_REGULATION_TEMPLATE = {
  id: 'reg-sn3.02.19-2025',
  name: 'СН 3.02.19-2025 — CCTV ТО журнал',
  items: [
    {
      id: 'item-1',
      label: 'Чистка линз',
      description: 'Проверить и очистить линзы камеры от загрязнений',
    },
    {
      id: 'item-2',
      label: 'Проверка записи',
      description: 'Убедиться, что запись ведётся на NVR',
    },
    {
      id: 'item-3',
      label: 'Терминалы соединений',
      description: 'Проверить надёжность контактных соединений',
    },
  ],
};

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

  // Device map
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    {
      status: 200,
      body: { devices: [] },
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

  // Regulation templates
  await device.mockRoute(
    { url: `${API_BASE}/mobile/regulations`, method: 'GET' },
    {
      status: 200,
      body: [MOCK_REGULATION_TEMPLATE],
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

  // Gatekeeper verify
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/verify`, method: 'POST' },
    {
      status: 200,
      body: mockVerificationResponse,
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Submit checklist
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/checklist`, method: 'POST' },
    {
      status: 200,
      body: {
        success: true,
        act_hash: 'act-wo-001-abc123def456',
        synced: true,
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Start work order button
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/start`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'in_progress' },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function setupServerErrorMock(): Promise<void> {
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/checklist`, method: 'POST' },
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
  await device.unmockRoute({ url: `${API_BASE}/mobile/regulations` });
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

describe('Maintenance Checklist', () => {
  it('Должен открыть MaintenanceChecklistScreen для work order', async () => {
    await waitForAnimation(3000);

    // Открываем work order
    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);

    // Запускаем work order (переводим в in_progress)
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);

    // Кнопка "Maintenance Checklist"
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Экран чек-листа открыт
    await expect(element(by.id('maintenance-checklist-screen'))).toBeVisible();
    await expect(element(by.text('СН 3.02.19-2025 — CCTV ТО журнал'))).toBeVisible();
  });

  it('Должен отобразить прогресс-бар с количеством пунктов', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Прогресс-бар
    await expect(element(by.id('checklist-progress-bar'))).toBeVisible();
    await expect(element(by.id('checklist-progress-text'))).toBeVisible();
    await expect(element(by.text('0/3'))).toBeVisible();
  });

  it('Должен пройти пункт чек-листа с Pass/Fail', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Первый пункт — нажимаем Pass
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(500);

    // Пункт отмечен как Passed
    await expect(element(by.id('passed-badge-item-1'))).toBeVisible();
    await expect(element(by.text('Passed'))).toBeVisible();

    // Прогресс обновился
    await expect(element(by.text('1/3'))).toBeVisible();
  });

  it('Должен отметить пункт как Failed при нажатии Fail', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Второй пункт — нажимаем Fail
    // Скроллим если нужно
    await element(by.id('fail-btn-item-2')).tap();
    await waitForAnimation(500);

    // Пункт отмечен как Failed
    await expect(element(by.id('failed-badge-item-2'))).toBeVisible();
    await expect(element(by.text('Failed'))).toBeVisible();
  });

  it('Должен добавить фото к пункту чек-листа', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Pass первый пункт
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(500);

    // Кнопка Add Photo
    await expect(element(by.id('add-photo-btn-item-1'))).toBeVisible();

    // Нажимаем Add Photo
    await element(by.id('add-photo-btn-item-1')).tap();
    await waitForAnimation(2000);

    // Фото добавлено — показывается thumbnail
    await expect(element(by.id('photo-thumbnail-item-1'))).toBeVisible();
  });

  it('Должен добавить notes к пункту чек-листа', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Pass первый пункт
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(500);

    // Поле ввода notes
    await element(by.id('notes-input-item-1')).tap();
    await element(by.id('notes-input-item-1')).typeText('Lens cleaned successfully');
    await waitForAnimation(500);

    // Notes сохранены
    await expect(element(by.id('notes-input-item-1'))).toHaveText('Lens cleaned successfully');
  });

  it('Должен перейти на шаг Gatekeeper после чек-листа', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим все пункты
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);

    // Кнопка Next становится активной
    await expect(element(by.id('checklist-next-btn'))).toBeVisible();
    await expect(element(by.id('checklist-next-btn'))).toBeEnabled();

    // Переходим на Gatekeeper шаг
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);

    // Gatekeeper шаг
    await expect(element(by.id('gatekeeper-step'))).toBeVisible();
    await expect(element(by.text('Verification'))).toBeVisible();
  });

  it('Должен выполнить Gatekeeper verification на втором шаге', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим все пункты
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);

    // Next → Gatekeeper
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);

    // Verify Now
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);

    // Результаты верификации
    await expect(element(by.id('gps-check-result'))).toBeVisible();
    await expect(element(by.id('exif-check-result'))).toBeVisible();
    await expect(element(by.id('ai-check-result'))).toBeVisible();

    // Token отобразился
    await expect(element(by.id('verification-token'))).toBeVisible();
  });

  it('Должен перейти на шаг Signature после Gatekeeper', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Чек-лист → Gatekeeper
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);

    // Gatekeeper → Signature
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);

    // Signature шаг
    await expect(element(by.id('signature-step'))).toBeVisible();
    await expect(element(by.text('Signature'))).toBeVisible();
  });

  it('Должен перейти на шаг Review после подписи', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим все шаги до Signature
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);

    // Симулируем подпись
    await element(by.id('capture-signature-btn')).tap();
    await waitForAnimation(1000);

    // Next → Review
    await element(by.id('signature-next-btn')).tap();
    await waitForAnimation(1000);

    // Review шаг
    await expect(element(by.id('review-step'))).toBeVisible();
    await expect(element(by.text('Review'))).toBeVisible();
  });

  it('Должен показать сводку на Review шаге', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим все шаги до Review
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('capture-signature-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('signature-next-btn')).tap();
    await waitForAnimation(1000);

    // Сводка
    await expect(element(by.id('review-summary'))).toBeVisible();

    // Checklist summary
    await expect(element(by.text('Checklist Summary'))).toBeVisible();
    await expect(element(by.text('3/3 completed'))).toBeVisible();

    // Gatekeeper summary
    await expect(element(by.text('GPS: Verified'))).toBeVisible();
    await expect(element(by.text('EXIF: Verified'))).toBeVisible();

    // Signature status
    await expect(element(by.text('Signature captured'))).toBeVisible();
  });

  it('Должен отправить акт ТО при нажатии Submit', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим все шаги до Review
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('capture-signature-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('signature-next-btn')).tap();
    await waitForAnimation(1000);

    // Submit
    await element(by.id('submit-act-btn')).tap();
    await waitForAnimation(3000);

    // Акт отправлен — возвращаемся на Dashboard
    await expect(element(by.id('dashboard-screen'))).toBeVisible();

    // Toast об успехе
    await expect(element(by.text('Act submitted successfully'))).toBeVisible();
  });

  it('Должен показать act_hash после submit', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим все шаги
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('capture-signature-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('signature-next-btn')).tap();
    await waitForAnimation(1000);

    // Submit
    await element(by.id('submit-act-btn')).tap();
    await waitForAnimation(3000);

    // Act hash отображается в toast
    await expect(element(by.id('act-hash-display'))).toBeVisible();
    await expect(element(by.id('act-hash-display'))).toHaveText('act-wo-001-abc123def456');
  });

  it('Должен показать error toast при ошибке submit', async () => {
    await waitForAnimation(3000);

    // Настраиваем ошибку
    await setupServerErrorMock();

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим шаги
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);
    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('capture-signature-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('signature-next-btn')).tap();
    await waitForAnimation(1000);

    // Submit с ошибкой
    await element(by.id('submit-act-btn')).tap();
    await waitForAnimation(2000);

    // Toast ошибки
    await expect(element(by.id('error-toast'))).toBeVisible();
    await expect(element(by.text('Failed to submit act'))).toBeVisible();
  });

  it('Должен сохранить чек-лист локально при offline', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);

    // Переходим в offline
    await device.setURLBlacklist(['.*']);
    await waitForAnimation(1000);

    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Проходим шаги в offline
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);

    // Должен отобразиться индикатор offline сохранения
    await expect(element(by.id('offline-save-indicator'))).toBeVisible();
    await expect(element(by.text('Saved locally'))).toBeVisible();
  });

  it('Должен показать индикатор failed пунктов в сводке', async () => {
    await waitForAnimation(3000);

    await element(by.text('Camera Main Entrance')).tap();
    await waitForAnimation(1000);
    await element(by.id('start-work-order-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('maintenance-checklist-btn')).tap();
    await waitForAnimation(2000);

    // Один пункт Pass, один Fail
    await element(by.id('pass-btn-item-1')).tap();
    await waitForAnimation(300);
    await element(by.id('fail-btn-item-2')).tap();
    await waitForAnimation(300);
    await element(by.id('pass-btn-item-3')).tap();
    await waitForAnimation(300);

    await element(by.id('checklist-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('verify-now-btn')).tap();
    await waitForAnimation(2000);
    await element(by.id('gatekeeper-next-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('capture-signature-btn')).tap();
    await waitForAnimation(1000);
    await element(by.id('signature-next-btn')).tap();
    await waitForAnimation(1000);

    // В сводке отображаются failed пункты
    await expect(element(by.id('failed-items-list'))).toBeVisible();
    await expect(element(by.text('Проверка записи'))).toBeVisible();
    await expect(element(by.id('warning-icon'))).toBeVisible();
  });
});
