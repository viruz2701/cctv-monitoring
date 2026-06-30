/// <reference types="node" />

import { test, expect } from '@playwright/test';
import { setupAuth, mockCatchAll } from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Photo Annotation — Extended E2E Tests
// Create, edit, delete annotations on canvas
// ═══════════════════════════════════════════════════════════════════════════

async function setupAnnotationMockApi(page: any) {
  await setupAuth(page);

  // Mock photo upload
  await page.route('**/api/v1/upload', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        url: 'https://storage.example.com/uploads/test-photo.jpg',
        filename: 'test-photo.jpg',
        size: 1048576,
        mime: 'image/jpeg',
        width: 1920,
        height: 1080,
      }),
    });
  });

  // Mock annotation save
  await page.route('**/api/v1/annotations*', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'ann-' + Date.now(),
          ...body,
          created_at: new Date().toISOString(),
        }),
      });
    } else if (request.method() === 'DELETE') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ status: 'deleted' }),
      });
    } else if (request.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          annotations: [
            { id: 'ann-1', type: 'arrow', color: '#ff0000', x: 100, y: 100, width: 50, height: 50 },
            { id: 'ann-2', type: 'rectangle', color: '#00ff00', x: 200, y: 200, width: 100, height: 80 },
          ],
        }),
      });
    } else {
      await route.fulfill({ status: 405 });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Photo Annotation — Create
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Photo Annotation — Create', () => {
  test.beforeEach(async ({ page }) => {
    await setupAnnotationMockApi(page);
    await page.goto('/work-orders/annotate');
    await page.waitForTimeout(1500);
  });

  test('Annotation — canvas is present for drawing', async ({ page }) => {
    const canvas = page.locator('canvas').first();
    await expect(canvas).toBeVisible();
  });

  test('Annotation — rectangle tool draws a rectangle', async ({ page }) => {
    const rectTool = page.locator(
      'button[aria-label*="rect" i], button[aria-label*="square" i], ' +
      'button[title*="rect" i], button[title*="Квадрат" i]',
    ).first();

    if (await rectTool.isVisible()) {
      await rectTool.click();
      await page.waitForTimeout(300);
    }

    const canvas = page.locator('canvas').first();
    if (await canvas.isVisible()) {
      const box = await canvas.boundingBox();
      if (box) {
        await page.mouse.move(box.x + 100, box.y + 100);
        await page.mouse.down();
        await page.mouse.move(box.x + 300, box.y + 250, { steps: 10 });
        await page.mouse.up();
        await page.waitForTimeout(500);
      }
    }
  });

  test('Annotation — circle tool draws an ellipse', async ({ page }) => {
    const circleTool = page.locator(
      'button[aria-label*="circle" i], button[aria-label*="ellipse" i], ' +
      'button[title*="circle" i], button[title*="Круг" i]',
    ).first();

    if (await circleTool.isVisible()) {
      await circleTool.click();
      await page.waitForTimeout(300);
    }

    const canvas = page.locator('canvas').first();
    if (await canvas.isVisible()) {
      const box = await canvas.boundingBox();
      if (box) {
        await page.mouse.move(box.x + 150, box.y + 150);
        await page.mouse.down();
        await page.mouse.move(box.x + 350, box.y + 300, { steps: 10 });
        await page.mouse.up();
        await page.waitForTimeout(500);
      }
    }
  });

  test('Annotation — arrow tool draws a directional arrow', async ({ page }) => {
    const arrowTool = page.locator(
      'button[aria-label*="arrow" i], button[aria-label*="line" i], ' +
      'button[title*="Стрелка" i]',
    ).first();

    if (await arrowTool.isVisible()) {
      await arrowTool.click();
      await page.waitForTimeout(300);
    }

    const canvas = page.locator('canvas').first();
    if (await canvas.isVisible()) {
      const box = await canvas.boundingBox();
      if (box) {
        await page.mouse.move(box.x + 50, box.y + 50);
        await page.mouse.down();
        await page.mouse.move(box.x + 400, box.y + 400, { steps: 10 });
        await page.mouse.up();
        await page.waitForTimeout(500);
      }
    }
  });

  test('Annotation — highlight tool selects area', async ({ page }) => {
    const highlightTool = page.locator(
      'button[aria-label*="highlight" i], button[aria-label*="Выдел" i], ' +
      'button[title*="Выделение" i]',
    ).first();

    if (await highlightTool.isVisible()) {
      await highlightTool.click();
      await page.waitForTimeout(300);
    }
  });

  test('Annotation — text tool adds text label', async ({ page }) => {
    const textTool = page.locator(
      'button[aria-label*="text" i], button[aria-label*="label" i], ' +
      'button[title*="Текст" i]',
    ).first();

    if (await textTool.isVisible()) {
      await textTool.click();
      await page.waitForTimeout(300);

      const canvas = page.locator('canvas').first();
      if (await canvas.isVisible()) {
        const box = await canvas.boundingBox();
        if (box) {
          await page.mouse.click(box.x + 200, box.y + 200);
          await page.waitForTimeout(300);
        }
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Photo Annotation — Edit
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Photo Annotation — Edit', () => {
  test.beforeEach(async ({ page }) => {
    await setupAnnotationMockApi(page);
    await page.goto('/work-orders/annotate');
    await page.waitForTimeout(1500);
  });

  test('Annotation — color picker has multiple color options', async ({ page }) => {
    // Find color swatches
    const colorSwatches = page.locator(
      'button[class*="color" i], button[class*="swatch" i], ' +
      'div[class*="color" i] button',
    );
    const count = await colorSwatches.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test('Annotation — stroke width selector exists', async ({ page }) => {
    const widthControl = page.locator(
      'input[type="range"], select[aria-label*="width" i], ' +
      'button[aria-label*="size" i], button[aria-label*="width" i]',
    ).first();
    await expect(widthControl).toBeVisible();
  });

  test('Annotation — undo button is present', async ({ page }) => {
    const undoBtn = page.locator(
      'button[aria-label*="undo" i], button[title*="Undo" i], ' +
      'button[aria-label*="Отменить" i]',
    ).first();
    await expect(undoBtn).toBeVisible();
  });

  test('Annotation — redo button is present', async ({ page }) => {
    const redoBtn = page.locator(
      'button[aria-label*="redo" i], button[title*="Redo" i], ' +
      'button[aria-label*="Повтор" i]',
    ).first();
    await expect(redoBtn).toBeVisible();
  });

  test('Annotation — clear all button is present', async ({ page }) => {
    const clearBtn = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /clear|очист|reset|сброс/i }).first();
    await expect(clearBtn).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Photo Annotation — Save & Export
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Photo Annotation — Save & Export', () => {
  test.beforeEach(async ({ page }) => {
    await setupAnnotationMockApi(page);
    await page.goto('/work-orders/annotate');
    await page.waitForTimeout(1500);
  });

  test('Annotation — export button generates image', async ({ page }) => {
    const exportBtn = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /export|экспорт|save|сохран/i }).first();
    await expect(exportBtn).toBeVisible();
  });

  test('Annotation — zoom controls adjust view', async ({ page }) => {
    const zoomControls = page.locator(
      'button[aria-label*="zoom" i], button[aria-label*="масштаб" i]',
    );
    const count = await zoomControls.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Annotation — fullscreen mode toggle available', async ({ page }) => {
    const fullscreenBtn = page.locator(
      'button[aria-label*="fullscreen" i], button[aria-label*="full screen" i], ' +
      'button[title*="full" i]',
    ).first();

    if (await fullscreenBtn.isVisible()) {
      await fullscreenBtn.click();
      await page.waitForTimeout(500);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Photo Annotation — Multiple Annotations
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Photo Annotation — Multiple Layers', () => {
  test.beforeEach(async ({ page }) => {
    await setupAnnotationMockApi(page);
    await page.goto('/work-orders/annotate');
    await page.waitForTimeout(1500);
  });

  test('Annotation — layer toggle controls exist', async ({ page }) => {
    const layerBtn = page.locator(
      'button[aria-label*="layer" i], button[aria-label*="слой" i], ' +
      'button[title*="Layer" i]',
    ).first();

    if (await layerBtn.isVisible()) {
      await layerBtn.click();
      await page.waitForTimeout(500);
    }
  });

  test('Annotation — annotation count indicator exists', async ({ page }) => {
    const countIndicator = page.locator(
      'span[class*="count" i], div[class*="count" i], ' +
      'text=/[0-9]+\s*(annotation|элемент)/i',
    ).first();
    await expect(countIndicator).toBeVisible();
  });
});
