// ──────────────────────────────────────────────────
// E2E: Photo Annotation Extended
//
// Проверяет:
//   - Создание, редактирование, удаление аннотаций
//   - Multiple annotation types (blur, arrow, highlight, text)
//   - Смена цвета и толщины линии
//   - Полный цикл: create → edit → export
// ──────────────────────────────────────────────────

import { mockWorkOrders } from './helpers/mockData';
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
      body: {
        token: 'mock-token',
        refresh_token: 'mock-refresh',
        user: { id: 'tech-1', username: 'johntech', role: 'technician' },
      },
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

  // Photo upload
  await device.mockRoute(
    { url: `${API_BASE}/mobile/photos/cam-101`, method: 'GET' },
    {
      status: 200,
      body: {
        photo_url: 'https://storage.example.com/photos/cam-101-extended.jpg',
        metadata: { width: 1920, height: 1080, format: 'jpeg', size_bytes: 2_400_000 },
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Export
  await device.mockRoute(
    { url: `${API_BASE}/mobile/photos/export`, method: 'POST' },
    {
      status: 200,
      body: { export_url: 'https://storage.example.com/exports/annotated-extended.png', file_size_bytes: 3_100_000, format: 'png' },
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

  // Work order complete
  await device.mockRoute(
    { url: `${API_BASE}/mobile/work-orders/*/complete`, method: 'POST' },
    {
      status: 200,
      body: { ...mockWorkOrders[0], status: 'completed' },
      headers: { 'Content-Type': 'application/json' },
    },
  );
}

async function teardownMockRoutes(): Promise<void> {
  await device.unmockRoute({ url: `${API_BASE}/mobile/auth/login` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/work-orders` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/photos/cam-101` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/photos/export` });
}

async function openPhotoAnnotation(): Promise<void> {
  await waitForAnimation(3000);
  await element(by.text('Camera Main Entrance')).tap();
  await waitForAnimation(1000);
  await element(by.id('add-photo-btn')).tap();
  await waitForAnimation(1000);
  await expect(element(by.id('photo-preview'))).toBeVisible();
  await element(by.id('annotate-btn')).tap();
  await waitForAnimation(2000);
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

describe('Photo Annotation — Extended', () => {
  // ── 1. Create multiple annotation types ──

  it('Должен создать blur, arrow и highlight на одном фото', async () => {
    await openPhotoAnnotation();

    // Blur
    await element(by.id('tool-blur')).tap();
    await waitForAnimation(500);
    await expect(element(by.id('tool-blur-active'))).toBeVisible();
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);
    await expect(element(by.id('annotation-layer-blur'))).toBeVisible();

    // Arrow
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('down', 'slow', 0.2);
    await waitForAnimation(1000);
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();

    // Highlight
    await element(by.id('tool-highlight')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('left', 'slow', 0.3);
    await waitForAnimation(1000);
    await expect(element(by.id('annotation-layer-highlight'))).toBeVisible();

    // Все три слоя
    await expect(element(by.id('annotation-count'))).toHaveText('3');
  });

  // ── 2. Delete annotation by undo ──

  it('Должен удалить аннотацию через Undo', async () => {
    await openPhotoAnnotation();

    // Draw arrow
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);
    await expect(element(by.id('annotation-count'))).toHaveText('1');

    // Delete via Undo
    await element(by.id('tool-undo')).tap();
    await waitForAnimation(1000);
    await expect(element(by.id('annotation-layer-arrow'))).not.toBeVisible();
    await expect(element(by.id('annotation-count'))).toHaveText('0');
  });

  // ── 3. Edit stroke color ──

  it('Должен изменить цвет обводки через Color Picker', async () => {
    await openPhotoAnnotation();

    // Open color picker
    await element(by.id('tool-color-picker')).tap();
    await waitForAnimation(500);

    // Color palette visible
    await expect(element(by.id('color-palette'))).toBeVisible();

    // Select red color
    await element(by.id('color-red')).tap();
    await waitForAnimation(500);

    // Active color indicator shows red
    await expect(element(by.id('active-color-indicator'))).toBeVisible();
  });

  // ── 4. Change stroke width ──

  it('Должен изменить толщину линии через Stroke Width Selector', async () => {
    await openPhotoAnnotation();

    // Open stroke width selector
    await element(by.id('tool-stroke-width')).tap();
    await waitForAnimation(500);

    // Stroke width options visible
    await expect(element(by.id('stroke-width-picker'))).toBeVisible();

    // Select thick line
    await element(by.id('stroke-width-thick')).tap();
    await waitForAnimation(500);

    // Selection confirmed
    await expect(element(by.id('stroke-width-thick-active'))).toBeVisible();
  });

  // ── 5. Full cycle: create → edit → export ──

  it('Должен выполнить полный цикл: создать аннотации → экспорт', async () => {
    await openPhotoAnnotation();

    // Create blur
    await element(by.id('tool-blur')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);

    // Create arrow
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('down', 'slow', 0.2);
    await waitForAnimation(1000);

    // Счётчик показывает 2
    await expect(element(by.id('annotation-count'))).toHaveText('2');

    // Export
    await element(by.id('export-btn')).tap();
    await waitForAnimation(2000);
    await expect(element(by.id('export-modal'))).toBeVisible();

    // Select PNG format
    await element(by.text('PNG')).tap();
    await waitForAnimation(500);

    // Confirm export
    await element(by.id('export-confirm-btn')).tap();
    await waitForAnimation(3000);

    // Export complete
    await expect(element(by.id('export-complete'))).toBeVisible();
  });

  // ── 6. Layer visibility toggle ──

  it('Должен переключить видимость слоя', async () => {
    await openPhotoAnnotation();

    // Draw multiple annotations
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);

    await element(by.id('tool-highlight')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('left', 'slow', 0.3);
    await waitForAnimation(1000);

    // Open layers panel
    await element(by.id('show-layers-btn')).tap();
    await waitForAnimation(1000);
    await expect(element(by.id('layer-panel'))).toBeVisible();

    // Toggle first layer visibility
    await element(by.id('layer-toggle-0')).tap();
    await waitForAnimation(500);

    // Layer hidden
    await expect(element(by.id('annotation-layer-arrow'))).not.toBeVisible();

    // Toggle back
    await element(by.id('layer-toggle-0')).tap();
    await waitForAnimation(500);

    // Layer visible again
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();
  });

  // ── 7. Annotation with measurement tool ──

  it('Должен использовать Measurement Tool для измерения', async () => {
    await openPhotoAnnotation();

    // Select measurement tool
    await element(by.id('tool-measure')).tap();
    await waitForAnimation(500);
    await expect(element(by.id('tool-measure-active'))).toBeVisible();

    // Draw measurement line
    await element(by.id('annotation-canvas')).swipe('down', 'slow', 0.3);
    await waitForAnimation(1000);

    // Measurement values displayed
    await expect(element(by.id('measurement-value-px'))).toBeVisible();
    await expect(element(by.id('measurement-value-px'))).not.toHaveText('');
  });
});
