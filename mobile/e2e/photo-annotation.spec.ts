// ──────────────────────────────────────────────────
// E2E: Photo Annotation (Blur, Measurement, Export)
//
// Проверяет:
//   - Canvas для аннотации загружается с фото
//   - Инструмент "Blur" скрывает выделенную область
//   - Инструмент "Measurement" показывает px/mm значения
//   - Инструменты "Arrow" и "Highlight" работают
//   - Undo/Redo аннотаций
//   - Экспорт создаёт shareable файл
//   - Множественные аннотации накладываются корректно
// ──────────────────────────────────────────────────

import { mockWorkOrders } from './helpers/mockData';
import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

/** Mock URL фото для аннотации — симулирует загрузку изображения с сервера */
const MOCK_PHOTO_URL = 'https://storage.example.com/photos/cam-101-2026-06-29.jpg';

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

  // Mock фото для загрузки на canvas
  await device.mockRoute(
    { url: `${API_BASE}/mobile/photos/cam-101`, method: 'GET' },
    {
      status: 200,
      body: {
        photo_url: MOCK_PHOTO_URL,
        metadata: {
          width: 1920,
          height: 1080,
          format: 'jpeg',
          size_bytes: 2_400_000,
        },
      },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Mock для экспорта аннотированного файла
  await device.mockRoute(
    { url: `${API_BASE}/mobile/photos/export`, method: 'POST' },
    {
      status: 200,
      body: {
        export_url: 'https://storage.example.com/exports/annotated-cam-101.png',
        file_size_bytes: 3_100_000,
        format: 'png',
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

/**
 * Вспомогательный шаг: открыть work order с photo, перейти в annotation.
 * Используется в нескольких тестах.
 */
async function openPhotoAnnotation(): Promise<void> {
  await waitForAnimation(3000);

  // Открываем work order "Camera Main Entrance"
  await element(by.text('Camera Main Entrance')).tap();
  await waitForAnimation(1000);

  // Добавляем фото (кнопка capture)
  await element(by.id('add-photo-btn')).tap();
  await waitForAnimation(1000);

  // Mock: фото загружено, показываем preview
  await expect(element(by.id('photo-preview'))).toBeVisible();

  // Открываем annotation canvas
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

describe('Photo Annotation', () => {
  // ── 1. Photo annotation canvas loads with image ──

  it('Должен загрузить canvas для аннотации с изображением', async () => {
    await openPhotoAnnotation();

    // Canvas загружен с фото
    await expect(element(by.id('annotation-canvas'))).toBeVisible();
    await expect(element(by.id('annotation-image'))).toBeVisible();

    // Панель инструментов отображается
    await expect(element(by.id('annotation-toolbar'))).toBeVisible();

    // Доступные инструменты
    await expect(element(by.id('tool-blur'))).toBeVisible();
    await expect(element(by.id('tool-measure'))).toBeVisible();
    await expect(element(by.id('tool-arrow'))).toBeVisible();
    await expect(element(by.id('tool-highlight'))).toBeVisible();
    await expect(element(by.id('tool-undo'))).toBeVisible();
    await expect(element(by.id('tool-redo'))).toBeVisible();
  });

  // ── 2. Blur tool redacts selected area ──

  it('Должен скрыть (blur) выбранную область на фото', async () => {
    await openPhotoAnnotation();

    // Выбираем инструмент Blur
    await element(by.id('tool-blur')).tap();
    await waitForAnimation(500);

    // Инструмент активирован — показывается подсветка активного инструмента
    await expect(element(by.id('tool-blur-active'))).toBeVisible();

    // Выделяем область для blur (проводим пальцем по canvas)
    // Симуляция: тап и drag по canvas
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);

    // После применения blur — область должна быть скрыта
    // Индикатор наличия blur-аннотации
    await expect(element(by.id('annotation-layer-blur'))).toBeVisible();

    // Количество аннотаций на слое
    await expect(element(by.id('annotation-count'))).toHaveText('1');
  });

  // ── 3. Measurement tool shows pixel/mm values ──

  it('Должен показать px/mm значения через Measurement Tool', async () => {
    await openPhotoAnnotation();

    // Выбираем инструмент Measurement
    await element(by.id('tool-measure')).tap();
    await waitForAnimation(500);

    // Инструмент активен
    await expect(element(by.id('tool-measure-active'))).toBeVisible();

    // Проводим линию для измерения
    await element(by.id('annotation-canvas')).swipe('down', 'slow', 0.2);
    await waitForAnimation(1000);

    // Появляется значение измерения в пикселях
    await expect(element(by.id('measurement-value-px'))).toBeVisible();
    await expect(element(by.id('measurement-value-px'))).not.toHaveText('');

    // Значение в миллиметрах (при известном scale)
    await expect(element(by.id('measurement-value-mm'))).toBeVisible();
    await expect(element(by.id('measurement-value-mm'))).not.toHaveText('');

    // Индикатор единицы измерения
    await expect(element(by.text('px'))).toBeVisible();
    await expect(element(by.text('mm'))).toBeVisible();
  });

  // ── 4. Arrow/highlight tools work ──

  it('Должен нарисовать стрелку и выделение через инструменты Arrow/Highlight', async () => {
    await openPhotoAnnotation();

    // ── Arrow tool ──
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);

    // Инструмент активен
    await expect(element(by.id('tool-arrow-active'))).toBeVisible();

    // Рисуем стрелку на canvas
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.4);
    await waitForAnimation(1000);

    // Стрелка отобразилась
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();

    // Счётчик аннотаций: 1
    await expect(element(by.id('annotation-count'))).toHaveText('1');

    // ── Highlight tool ──
    await element(by.id('tool-highlight')).tap();
    await waitForAnimation(500);

    // Инструмент активен
    await expect(element(by.id('tool-highlight-active'))).toBeVisible();

    // Выделяем область (swipe)
    await element(by.id('annotation-canvas')).swipe('left', 'slow', 0.3);
    await waitForAnimation(1000);

    // Highlight отобразился
    await expect(element(by.id('annotation-layer-highlight'))).toBeVisible();

    // Счётчик аннотаций: 2
    await expect(element(by.id('annotation-count'))).toHaveText('2');
  });

  // ── 5. Undo/redo annotation changes ──

  it('Должен отменить и вернуть аннотации через Undo/Redo', async () => {
    await openPhotoAnnotation();

    // Рисуем первую аннотацию (стрелка)
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.4);
    await waitForAnimation(1000);

    // Рисуем вторую аннотацию (highlight)
    await element(by.id('tool-highlight')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('left', 'slow', 0.3);
    await waitForAnimation(1000);

    // Счётчик: 2
    await expect(element(by.id('annotation-count'))).toHaveText('2');

    // ── Undo: отменяем последнюю аннотацию (highlight) ──
    await element(by.id('tool-undo')).tap();
    await waitForAnimation(1000);

    // Highlight должен исчезнуть
    await expect(element(by.id('annotation-layer-highlight'))).not.toBeVisible();

    // Стрелка осталась
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();

    // Счётчик: 1
    await expect(element(by.id('annotation-count'))).toHaveText('1');

    // ── Undo: отменяем стрелку ──
    await element(by.id('tool-undo')).tap();
    await waitForAnimation(1000);

    // Стрелка тоже исчезла
    await expect(element(by.id('annotation-layer-arrow'))).not.toBeVisible();

    // Счётчик: 0
    await expect(element(by.id('annotation-count'))).toHaveText('0');

    // ── Redo: возвращаем стрелку ──
    await element(by.id('tool-redo')).tap();
    await waitForAnimation(1000);

    // Стрелка снова отображается
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();

    // Счётчик: 1
    await expect(element(by.id('annotation-count'))).toHaveText('1');

    // ── Redo: возвращаем highlight ──
    await element(by.id('tool-redo')).tap();
    await waitForAnimation(1000);

    // Highlight снова отображается
    await expect(element(by.id('annotation-layer-highlight'))).toBeVisible();

    // Счётчик: 2
    await expect(element(by.id('annotation-count'))).toHaveText('2');
  });

  // ── 6. Export creates shareable file ──

  it('Должен экспортировать аннотированное фото в shareable файл', async () => {
    await openPhotoAnnotation();

    // Добавляем пару аннотаций перед экспортом
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);

    await element(by.id('tool-highlight')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('left', 'slow', 0.3);
    await waitForAnimation(1000);

    // Нажимаем Export
    await element(by.id('export-btn')).tap();
    await waitForAnimation(2000);

    // Модальное окно экспорта с опциями
    await expect(element(by.id('export-modal'))).toBeVisible();

    // Опции: формат файла, качество
    await expect(element(by.id('export-format-picker'))).toBeVisible();
    await expect(element(by.text('PNG'))).toBeVisible();
    await expect(element(by.text('JPEG'))).toBeVisible();

    // Выбираем формат PNG
    await element(by.text('PNG')).tap();
    await waitForAnimation(500);

    // Подтверждаем экспорт
    await element(by.id('export-confirm-btn')).tap();
    await waitForAnimation(3000);

    // Прогресс экспорта
    await expect(element(by.id('export-progress'))).toBeVisible();
    await waitForAnimation(2000);

    // Экспорт завершён — показывается share sheet
    await expect(element(by.id('export-complete'))).toBeVisible();
    await expect(element(by.text('Export complete'))).toBeVisible();

    // Кнопка Share
    await expect(element(by.id('share-btn'))).toBeVisible();
  });

  // ── 7. Multiple annotations layer correctly ──

  it('Должен корректно наложить множественные аннотации (blur + arrow + highlight)', async () => {
    await openPhotoAnnotation();

    // ── 1. Blur область ──
    await element(by.id('tool-blur')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('right', 'slow', 0.3);
    await waitForAnimation(1000);

    // ── 2. Стрелка ──
    await element(by.id('tool-arrow')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('down', 'slow', 0.2);
    await waitForAnimation(1000);

    // ── 3. Highlight ──
    await element(by.id('tool-highlight')).tap();
    await waitForAnimation(500);
    await element(by.id('annotation-canvas')).swipe('left', 'slow', 0.3);
    await waitForAnimation(1000);

    // Все три слоя отображаются
    await expect(element(by.id('annotation-layer-blur'))).toBeVisible();
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();
    await expect(element(by.id('annotation-layer-highlight'))).toBeVisible();

    // Счётчик: 3
    await expect(element(by.id('annotation-count'))).toHaveText('3');

    // Список слоёв (layer stack) доступен
    await element(by.id('show-layers-btn')).tap();
    await waitForAnimation(1000);

    // Layer panel отображает все слои по порядку
    await expect(element(by.id('layer-panel'))).toBeVisible();
    await expect(element(by.id('layer-item-0'))).toBeVisible(); // blur (первый)
    await expect(element(by.id('layer-item-1'))).toBeVisible(); // arrow
    await expect(element(by.id('layer-item-2'))).toBeVisible(); // highlight (последний)

    // Можно переключить видимость слоя
    await element(by.id('layer-toggle-1')).tap(); // скрываем arrow
    await waitForAnimation(500);
    await expect(element(by.id('annotation-layer-arrow'))).not.toBeVisible();

    // Показываем обратно
    await element(by.id('layer-toggle-1')).tap();
    await waitForAnimation(500);
    await expect(element(by.id('annotation-layer-arrow'))).toBeVisible();
  });
});
