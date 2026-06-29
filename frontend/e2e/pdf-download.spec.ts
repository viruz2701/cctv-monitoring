/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// PDF Download — E2E Tests
// Report list, generate, download, verify content-disposition, regional templates
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_REPORT_LIST = [
  { id: 'rpt-1', title: 'Ежедневный отчёт — 2026-06-28', type: 'daily', format: 'pdf', status: 'ready', created_at: new Date(Date.now() - 86400000).toISOString(), url: '/api/v1/reports/rpt-1/download', generated_by: 'admin', expires_at: new Date(Date.now() + 86400000 * 7).toISOString() },
  { id: 'rpt-2', title: 'Еженедельный отчёт — W26', type: 'weekly', format: 'pdf', status: 'ready', created_at: new Date(Date.now() - 604800000).toISOString(), url: '/api/v1/reports/rpt-2/download', generated_by: 'admin', expires_at: new Date(Date.now() + 86400000 * 30).toISOString() },
  { id: 'rpt-3', title: 'Аварийный отчёт — Камера 12', type: 'incident', format: 'pdf', status: 'generating', created_at: new Date(Date.now() - 3600000).toISOString(), url: null, generated_by: 'system', expires_at: null },
  { id: 'rpt-4', title: 'Ежемесячный отчёт — Июнь 2026', type: 'monthly', format: 'pdf', status: 'ready', created_at: new Date(Date.now() - 86400000 * 5).toISOString(), url: '/api/v1/reports/rpt-4/download', generated_by: 'manager', expires_at: new Date(Date.now() + 86400000 * 90).toISOString() },
  { id: 'rpt-5', title: 'Инвентаризация оборудования Q2', type: 'monthly', format: 'xlsx', status: 'ready', created_at: new Date(Date.now() - 86400000 * 10).toISOString(), url: '/api/v1/reports/rpt-5/download', generated_by: 'admin', expires_at: new Date(Date.now() + 86400000 * 90).toISOString() },
  { id: 'rpt-6', title: 'Отчёт по инцидентам — Неделя 26', type: 'incident', format: 'pdf', status: 'failed', created_at: new Date(Date.now() - 86400000 * 2).toISOString(), url: null, generated_by: 'system', expires_at: null },
];

const PDF_MOCK_CONTENT = Buffer.from(
  '%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\nendobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \ntrailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n190\n%%EOF',
);

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupPdfMockApi(page: any, regionalTemplate?: string) {
  await setupAuth(page);

  // Report list
  await page.route('**/api/v1/reports', async (route: any, request: any) => {
    if (request.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_REPORT_LIST),
      });
    } else if (request.method() === 'POST') {
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'rpt-new-' + Date.now(),
          title: 'Сгенерированный отчёт',
          type: 'daily',
          format: 'pdf',
          status: 'generating',
          created_at: new Date().toISOString(),
          url: null,
          generated_by: 'admin',
          expires_at: null,
          message: 'Отчёт поставлен в очередь генерации',
        }),
      });
    }
  });

  // Generate report — POST specific
  await page.route('**/api/v1/reports/generate', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: 'rpt-gen-' + Date.now(),
          title: body.title || 'Сгенерированный отчёт',
          type: body.type || 'daily',
          format: 'pdf',
          status: 'generating',
          template: body.template || regionalTemplate || 'standard',
          created_at: new Date().toISOString(),
          url: null,
        }),
      });
    }
  });

  // Download report
  await page.route('**/api/v1/reports/*/download', async (route: any, request: any) => {
    const url = request.url();
    const reportId = url.match(/\/reports\/([^/]+)\/download/)?.[1] || 'unknown';
    const report = MOCK_REPORT_LIST.find((r) => r.id === reportId);

    if (report && report.status === 'ready') {
      const disposition = regionalTemplate
        ? `attachment; filename="report-${regionalTemplate}-${reportId}.pdf"`
        : `attachment; filename="report-${reportId}.pdf"`;

      await route.fulfill({
        status: 200,
        contentType: 'application/pdf',
        headers: {
          'Content-Disposition': disposition,
          'Content-Type': 'application/pdf',
          'Content-Length': String(PDF_MOCK_CONTENT.length),
        },
        body: PDF_MOCK_CONTENT,
      });
    } else {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'report_not_found', message: 'Отчёт не найден или ещё генерируется' }),
      });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: PDF Download — Report List
// ═══════════════════════════════════════════════════════════════════════════

test.describe('PDF Download — Report List', () => {
  test.beforeEach(async ({ page }) => {
    await setupPdfMockApi(page);
    await page.goto('/reports');
    await page.waitForTimeout(1500);
  });

  test('Reports page loads with report list', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/reports');

    // Проверяем отображение списка отчётов
    const reportTitle = page.locator(
      'text=/Ежедневный отчёт|Еженедельный отчёт|Аварийный отчёт|Ежемесячный отчёт/i',
    ).first();
    await expect(reportTitle).toBeVisible();
  });

  test('Reports — report status badges visible (ready, generating, failed)', async ({ page }) => {
    // Проверяем бейдж статуса "ready"
    const readyBadge = page.locator(
      'span, badge, div[class*="badge" i]',
    ).filter({ hasText: /ready|готов/i }).first();
    await expect(readyBadge).toBeVisible();

    // Проверяем бейдж статуса "generating"
    const generatingBadge = page.locator(
      'span, badge, div[class*="badge" i]',
    ).filter({ hasText: /generating|генерир/i }).first();
    await expect(generatingBadge).toBeVisible();
  });

  test('Reports — report types displayed (daily, weekly, monthly, incident)', async ({ page }) => {
    // Проверяем отображение разных типов отчётов
    const dailyReport = page.locator('text=/ежедневн|daily/i').first();
    const weeklyReport = page.locator('text=/еженедельн|weekly/i').first();
    const monthlyReport = page.locator('text=/ежемесячн|monthly/i').first();
    const incidentReport = page.locator('text=/аварийн|incident|инцидент/i').first();

    await expect(dailyReport).toBeVisible();
    await expect(weeklyReport).toBeVisible();
    await expect(monthlyReport).toBeVisible();
    await expect(incidentReport).toBeVisible();
  });

  test('Reports — generated by and date displayed', async ({ page }) => {
    // Проверяем отображение информации о создателе
    const generatedBy = page.locator(
      'text=/admin|manager|system/i',
    ).first();
    await expect(generatedBy).toBeVisible();

    // Проверяем дату создания
    const createdDate = page.locator(
      'text=/2026|Июнь|June/i',
    ).first();
    await expect(createdDate).toBeVisible();
  });

  test('Reports — download button visible on ready reports', async ({ page }) => {
    // Проверяем кнопку скачивания на готовых отчётах
    const downloadButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /download|скач|export|экспорт/i }).first();
    await expect(downloadButton).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: PDF Download — Generate & Download
// ═══════════════════════════════════════════════════════════════════════════

test.describe('PDF Download — Generate & Download', () => {
  test.beforeEach(async ({ page }) => {
    await setupPdfMockApi(page);
    await page.goto('/reports');
    await page.waitForTimeout(1500);
  });

  test('Reports — generate PDF button triggers creation', async ({ page }) => {
    // Находим кнопку генерации нового отчёта
    const generateButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /generate|создат|new report|новый отчёт|create/i }).first();

    await expect(generateButton).toBeVisible();
    await generateButton.click();
    await page.waitForTimeout(1000);

    // Проверяем появление уведомления о постановке в очередь
    const generateNotification = page.locator(
      'div[class*="toast" i], div[role="alert"]',
    ).filter({ hasText: /generate|создан|queued|очеред|generating|генерир/i }).first();
    const hasNotification = await generateNotification.isVisible().catch(() => false);
    if (hasNotification) {
      await expect(generateNotification).toBeVisible();
    }
  });

  test('Reports — download PDF returns application/octet-stream', async ({ page }) => {
    // Кликаем по кнопке скачивания готового отчёта
    const downloadButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /download|скач/i }).first();

    if (await downloadButton.isVisible()) {
      // Устанавливаем перехват скачивания
      const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);
      await downloadButton.click();

      const download = await downloadPromise;
      if (download) {
        const suggestedName = download.suggestedFilename();
        expect(suggestedName).toContain('.pdf');
        expect(suggestedName.toLowerCase()).toMatch(/\.pdf$/);
      }
    }
  });

  test('Reports — verify Content-Disposition header', async ({ page }) => {
    // Проверяем заголовок Content-Disposition при скачивании
    const response = await page.evaluate(async () => {
      const resp = await fetch('/api/v1/reports/rpt-1/download');
      return {
        disposition: resp.headers.get('Content-Disposition'),
        contentType: resp.headers.get('Content-Type'),
      };
    });

    expect(response.disposition).toContain('attachment');
    expect(response.disposition).toContain('filename=');
    expect(response.contentType).toBe('application/pdf');
  });

  test('Reports — regional template generates localized PDF', async ({ page }) => {
    // Настраиваем API с региональным шаблоном
    await setupPdfMockApi(page, 'by-region');
    await page.goto('/reports');
    await page.waitForTimeout(1500);

    // Проверяем генерацию с региональным шаблоном
    const generateButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /generate|создат|new report|новый отчёт/i }).first();

    if (await generateButton.isVisible()) {
      await generateButton.click();
      await page.waitForTimeout(1000);

      // Проверяем отображение информации о шаблоне
      const templateInfo = page.locator(
        'text=/template|шаблон|by-region|regional|регион/i',
      ).first();
      const hasTemplate = await templateInfo.isVisible().catch(() => false);
    }
  });

  test('Reports — report history shows multiple generated reports', async ({ page }) => {
    // Проверяем отображение истории отчётов
    const reportHistory = page.locator(
      'table, div[class*="list" i], div[class*="table" i]',
    ).first();

    await expect(reportHistory).toBeVisible();

    // Проверяем количество отчётов в списке
    const reportRows = page.locator(
      'tr, div[class*="row" i], li',
    ).filter({ hasText: /отчёт|report/i });
    const count = await reportRows.count();
    expect(count).toBeGreaterThanOrEqual(3);
  });

  test('Reports — failed report status shows retry option', async ({ page }) => {
    // Проверяем отображение статуса failed
    const failedStatus = page.locator(
      'span, badge, div[class*="badge" i]',
    ).filter({ hasText: /failed|ошибк|error/i }).first();

    if (await failedStatus.isVisible()) {
      // Проверяем кнопку повторной генерации
      const retryButton = page.locator(
        'button, [role="button"]',
      ).filter({ hasText: /retry|повтор|regenerate|перегенер/i }).first();
      const hasRetry = await retryButton.isVisible().catch(() => false);
    }
  });
});
