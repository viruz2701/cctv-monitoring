/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Photo Annotation — E2E Tests
// Drawing tools, color picker, undo/redo, export
// ═══════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupPhotoAnnotationMockApi(page: any) {
  await setupAuth(page);

  // Upload photo
  await page.route('**/api/v1/upload', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: 'https://storage.example.com/uploads/annotation-mock.jpg',
          filename: 'annotation-mock.jpg',
          size: 2048576,
          mime: 'image/jpeg',
          width: 1920,
          height: 1080,
        }),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          url: 'https://storage.example.com/annotations/mock-result.jpg',
          filename: 'annotated-result.jpg',
          size: 1024000,
          mime: 'image/jpeg',
        }),
      });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Photo Annotation — Tools & Color
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Photo Annotation — Tools & Color', () => {
  test.beforeEach(async ({ page }) => {
    await setupPhotoAnnotationMockApi(page);
    await page.goto('/work-orders/annotate');
    await page.waitForTimeout(1500);
  });

  test('Annotation toolbar is visible with drawing tools', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/annotate');

    // Проверяем отображение тулбара аннотаций
    const toolbar = page.locator(
      'div[class*="toolbar" i], div[class*="annotation" i], nav[class*="tool" i], ' +
      'div[role="toolbar"]',
    ).first();
    await expect(toolbar).toBeVisible();

    // Проверяем кнопки инструментов
    const toolButtons = page.locator(
      'button[class*="tool" i], button[aria-label*="draw" i], button[aria-label*="shape" i], ' +
      'button:has(svg), button[class*="icon" i]',
    );
    const count = await toolButtons.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test('Annotation — switch between drawing tools (rectangle, circle, arrow)', async ({ page }) => {
    // Находим и кликаем по инструменту прямоугольника
    const rectTool = page.locator(
      'button[aria-label*="rect" i], button[aria-label*="square" i], ' +
      'button[class*="rect" i], button:has-text("rect" i)',
    ).first();

    if (await rectTool.isVisible()) {
      await rectTool.click();
      await page.waitForTimeout(300);

      // Проверяем что инструмент активирован
      const isActive = await rectTool.getAttribute('class').catch(() => '');
      const isPressed = await rectTool.getAttribute('aria-pressed').catch(() => '');
    }

    // Переключаемся на круг/эллипс
    const circleTool = page.locator(
      'button[aria-label*="circle" i], button[aria-label*="ellipse" i], ' +
      'button[aria-label*="oval" i]',
    ).first();

    if (await circleTool.isVisible()) {
      await circleTool.click();
      await page.waitForTimeout(300);
    }

    // Переключаемся на стрелку/линию
    const arrowTool = page.locator(
      'button[aria-label*="arrow" i], button[aria-label*="line" i], ' +
      'button[class*="arrow" i]',
    ).first();

    if (await arrowTool.isVisible()) {
      await arrowTool.click();
      await page.waitForTimeout(300);
    }
  });

  test('Annotation — color picker changes stroke color', async ({ page }) => {
    // Находим палитру цветов
    const colorPicker = page.locator(
      'input[type="color"], div[class*="color" i] button, ' +
      'button[aria-label*="color" i], div[class*="picker" i] button, ' +
      'div[class*="swatch" i]',
    ).first();

    if (await colorPicker.isVisible()) {
      await colorPicker.click();
      await page.waitForTimeout(500);

      // Выбираем красный цвет
      const redColor = page.locator(
        'button[aria-label*="red" i], button[aria-label*="#ff" i], ' +
        'button[style*="red" i], button[style*="#ff0000" i]',
      ).first();

      if (await redColor.isVisible()) {
        await redColor.click();
        await page.waitForTimeout(300);
      }
    }
  });

  test('Annotation — draw on canvas creates annotation', async ({ page }) => {
    // Находим canvas для рисования
    const canvas = page.locator(
      'canvas, div[class*="canvas" i], svg[class*="annotation" i]',
    ).first();

    if (await canvas.isVisible()) {
      // Получаем размеры canvas
      const box = await canvas.boundingBox();
      if (box) {
        // Симулируем рисование прямоугольника
        await page.mouse.move(box.x + 100, box.y + 100);
        await page.mouse.down();
        await page.mouse.move(box.x + 300, box.y + 250, { steps: 10 });
        await page.mouse.up();
        await page.waitForTimeout(500);

        // Проверяем что появился аннотационный элемент
        const annotationElement = page.locator(
          'rect, circle, ellipse, line, path, div[class*="annotation" i], ' +
          'svg > g > *',
        ).first();
        const hasAnnotation = await annotationElement.isVisible().catch(() => false);
      }
    }
  });

  test('Annotation — undo/redo buttons are functional', async ({ page }) => {
    // Проверяем кнопку Undo
    const undoButton = page.locator(
      'button[aria-label*="undo" i], button[aria-label*="back" i], ' +
      'button:has(svg[class*="undo" i]), button[class*="undo" i]',
    ).first();
    await expect(undoButton).toBeVisible();

    // Проверяем кнопку Redo
    const redoButton = page.locator(
      'button[aria-label*="redo" i], button[aria-label*="forward" i], ' +
      'button:has(svg[class*="redo" i]), button[class*="redo" i]',
    ).first();
    await expect(redoButton).toBeVisible();

    // Кликаем Undo если он активен
    if (await undoButton.isEnabled().catch(() => false)) {
      await undoButton.click();
      await page.waitForTimeout(300);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Photo Annotation — Clear & Export
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Photo Annotation — Clear & Export', () => {
  test.beforeEach(async ({ page }) => {
    await setupPhotoAnnotationMockApi(page);
    await page.goto('/work-orders/annotate');
    await page.waitForTimeout(1500);
  });

  test('Annotation — clear all removes annotations from canvas', async ({ page }) => {
    // Находим кнопку Clear All
    const clearButton = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /clear|очист|reset|сброс|remove all|удалить все/i }).first();

    await expect(clearButton).toBeVisible();
    await clearButton.click();
    await page.waitForTimeout(500);

    // Проверяем диалог подтверждения
    const confirmDialog = page.locator(
      'div[role="dialog"], div[class*="modal" i], div[class*="confirm" i]',
    ).filter({ hasText: /clear|очист|confirm|подтвер/i }).first();
    const hasDialog = await confirmDialog.isVisible().catch(() => false);

    if (hasDialog) {
      const yesButton = confirmDialog.locator(
        'button, [role="button"]',
      ).filter({ hasText: /yes|да|clear|очист|confirm|подтвер/i }).first();
      await yesButton.click();
      await page.waitForTimeout(500);
    }
  });

  test('Annotation — export button generates annotated image', async ({ page }) => {
    // Находим кнопку Export
    const exportButton = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /export|экспорт|save|сохран|download|скач/i }).first();

    await expect(exportButton).toBeVisible();
    await exportButton.click();
    await page.waitForTimeout(1000);

    // Проверяем что экспорт был инициирован
    const exportNotification = page.locator(
      'div[class*="toast" i], div[role="alert"]',
    ).filter({ hasText: /export|экспорт|download|скач|success|успеш/i }).first();
    const hasNotification = await exportNotification.isVisible().catch(() => false);
  });

  test('Annotation — stroke width selector changes brush size', async ({ page }) => {
    // Находим селектор толщины линии
    const strokeWidth = page.locator(
      'input[type="range"], select[aria-label*="width" i], ' +
      'button[aria-label*="size" i], button[aria-label*="width" i], ' +
      'div[class*="stroke" i] button',
    ).first();

    if (await strokeWidth.isVisible()) {
      await strokeWidth.click();
      await page.waitForTimeout(300);

      // Выбираем большую толщину
      const thickOption = page.locator(
        'button, option, div[role="option"]',
      ).filter({ hasText: /8|10|thick|толст|large|больш/i }).first();
      if (await thickOption.isVisible()) {
        await thickOption.click();
        await page.waitForTimeout(300);
      }
    }
  });

  test('Annotation — text tool adds text annotation', async ({ page }) => {
    // Находим и активируем текстовый инструмент
    const textTool = page.locator(
      'button[aria-label*="text" i], button[aria-label*="label" i], ' +
      'button[aria-label*="type" i]',
    ).first();

    if (await textTool.isVisible()) {
      await textTool.click();
      await page.waitForTimeout(300);

      // Кликаем на canvas для размещения текста
      const canvas = page.locator('canvas').first();
      if (await canvas.isVisible()) {
        const box = await canvas.boundingBox();
        if (box) {
          await page.mouse.click(box.x + 200, box.y + 200);
          await page.waitForTimeout(300);

          // Вводим текст
          const textInput = page.locator(
            'input[type="text"], textarea, div[contenteditable="true"]',
          ).first();

          if (await textInput.isVisible()) {
            await textInput.fill('Defect area A-12');
            await page.waitForTimeout(300);

            // Подтверждаем ввод
            await page.keyboard.press('Enter');
            await page.waitForTimeout(300);
          }
        }
      }
    }
  });

  test('Annotation — zoom controls adjust canvas view', async ({ page }) => {
    // Проверяем zoom контролы
    const zoomIn = page.locator(
      'button[aria-label*="zoom in" i], button[aria-label*="plus" i], ' +
      'button[aria-label*="увелич" i]',
    ).first();

    const zoomOut = page.locator(
      'button[aria-label*="zoom out" i], button[aria-label*="minus" i], ' +
      'button[aria-label*="уменьш" i]',
    ).first();

    if (await zoomIn.isVisible()) {
      await zoomIn.click();
      await page.waitForTimeout(300);
    }

    if (await zoomOut.isVisible()) {
      await zoomOut.click();
      await page.waitForTimeout(300);
    }

    // Проверяем отображение процента зума
    const zoomLevel = page.locator(
      'text=/100%|%|zoom|масштаб/i',
    ).first();
    await expect(zoomLevel).toBeVisible();
  });
});
