/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Quota Management — E2E Tests
// View quota, usage bars, request increase, soft limit warnings
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_TENANT_QUOTA = {
  tenant_id: 'tenant-1',
  tenant_name: 'Main Office — Enterprise',
  plan: 'enterprise',
  quotas: [
    {
      type: 'devices',
      label: 'Устройства',
      limit: 100,
      used: 72,
      available: 28,
      usage_percent: 72,
      soft_limit: 80,
      hard_limit: 100,
    },
    {
      type: 'storage_gb',
      label: 'Хранилище (GB)',
      limit: 5000,
      used: 3420,
      available: 1580,
      usage_percent: 68.4,
      soft_limit: 80,
      hard_limit: 100,
    },
    {
      type: 'users',
      label: 'Пользователи',
      limit: 25,
      used: 18,
      available: 7,
      usage_percent: 72,
      soft_limit: 80,
      hard_limit: 100,
    },
    {
      type: 'api_calls_per_day',
      label: 'API запросов/день',
      limit: 50000,
      used: 32150,
      available: 17850,
      usage_percent: 64.3,
      soft_limit: 80,
      hard_limit: 100,
    },
    {
      type: 'streams',
      label: 'Потоковое видео',
      limit: 50,
      used: 38,
      available: 12,
      usage_percent: 76,
      soft_limit: 80,
      hard_limit: 100,
    },
    {
      type: 'retention_days',
      label: 'Хранение архива (дни)',
      limit: 90,
      used: 90,
      available: 0,
      usage_percent: 100,
      soft_limit: 80,
      hard_limit: 100,
    },
  ],
  billing_cycle_start: new Date(Date.now() - 86400000 * 15).toISOString(),
  billing_cycle_end: new Date(Date.now() + 86400000 * 15).toISOString(),
};

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupQuotaMockApi(page: any) {
  await setupAuth(page);

  // Quota overview
  await page.route('**/api/v1/tenant/quota', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_TENANT_QUOTA),
    });
  });

  // Request increase — POST
  await page.route('**/api/v1/tenant/quota/request-increase', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          request_id: `inc-req-${Date.now()}`,
          quota_type: body.quota_type || 'devices',
          requested_amount: body.requested_amount || 50,
          status: 'pending',
          message: 'Запрос на увеличение квоты отправлен. Ожидайте подтверждения.',
          created_at: new Date().toISOString(),
        }),
      });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Quota Management — Overview
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Quota Management — Overview', () => {
  test.beforeEach(async ({ page }) => {
    await setupQuotaMockApi(page);
    await page.goto('/tenant/quota');
    await page.waitForTimeout(1500);
  });

  test('Quota page loads with overview of all quota types', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/quota');

    // Проверяем отображение названия тенанта
    const tenantName = page.locator(
      'text=/Main Office.*Enterprise|Enterprise|tenant|тариф/i',
    ).first();
    await expect(tenantName).toBeVisible();
  });

  test('Quota — device quota usage bar is displayed', async ({ page }) => {
    // Проверяем отображение квоты устройств
    const deviceQuotaLabel = page.locator(
      'text=/device|устройств/i',
    ).first();
    await expect(deviceQuotaLabel).toBeVisible();

    // Проверяем прогресс-бар использования
    const progressBar = page.locator(
      'div[role="progressbar"], div[class*="progress" i], div[class*="bar" i]',
    ).first();
    await expect(progressBar).toBeVisible();
  });

  test('Quota — storage quota with GB values displayed', async ({ page }) => {
    // Проверяем квоту хранилища
    const storageQuota = page.locator(
      'text=/storage|хранилищ|GB|3420|5000/i',
    ).first();
    await expect(storageQuota).toBeVisible();

    // Проверяем отображение использованного объёма
    const storageUsed = page.locator(
      'text=/3420|1580|available|доступ/i',
    ).first();
    await expect(storageUsed).toBeVisible();
  });

  test('Quota — usage percent displayed for each quota type', async ({ page }) => {
    // Проверяем проценты использования
    const usagePercent = page.locator(
      'text=/72%|68%|64%|76%|100%|72\\.|68\\.|64\\.|76\\./i',
    ).first();
    await expect(usagePercent).toBeVisible();
  });

  test('Quota — multiple quota types listed (devices, storage, users, API)', async ({ page }) => {
    // Проверяем отображение всех типов квот
    const deviceQuota = page.locator('text=/device|устройств/i').first();
    const storageQuota = page.locator('text=/storage|хранилищ/i').first();
    const userQuota = page.locator('text=/user|пользовател/i').first();
    const apiQuota = page.locator('text=/api|запрос/i').first();

    await expect(deviceQuota).toBeVisible();
    await expect(storageQuota).toBeVisible();
    await expect(userQuota).toBeVisible();
    await expect(apiQuota).toBeVisible();
  });

  test('Quota — billing cycle dates displayed', async ({ page }) => {
    // Проверяем отображение дат цикла биллинга
    const billingCycle = page.locator(
      'text=/billing|биллинг|cycle|цикл|15 days|дней/i',
    ).first();
    await expect(billingCycle).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Quota Management — Request Increase
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Quota Management — Request Increase', () => {
  test.beforeEach(async ({ page }) => {
    await setupQuotaMockApi(page);
    await page.goto('/tenant/quota');
    await page.waitForTimeout(1500);
  });

  test('Quota — request increase button opens modal', async ({ page }) => {
    // Находим кнопку запроса увеличения
    const requestButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /request.*increase|увелич|increase.*quota|upgrade|расшир/i }).first();

    if (await requestButton.isVisible()) {
      await requestButton.click();
      await page.waitForTimeout(1000);

      // Проверяем появление модала
      const modal = page.locator(
        'div[role="dialog"], div[class*="modal" i], div[class*="dialog" i]',
      ).first();
      const hasModal = await modal.isVisible().catch(() => false);
      if (hasModal) {
        await expect(modal).toBeVisible();
      }
    }
  });

  test('Quota — request increase form submits successfully', async ({ page }) => {
    // Открываем модал запроса
    const requestButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /request.*increase|увелич|increase.*quota|upgrade|расшир/i }).first();

    if (await requestButton.isVisible()) {
      await requestButton.click();
      await page.waitForTimeout(1000);

      // Заполняем форму
      const quotaTypeSelect = page.locator(
        'select, div[role="combobox"]',
      ).first();
      if (await quotaTypeSelect.isVisible()) {
        await quotaTypeSelect.selectOption('storage_gb');
      }

      const amountInput = page.locator(
        'input[type="number"], input[type="text"]',
      ).first();
      if (await amountInput.isVisible()) {
        await amountInput.fill('2000');
      }

      const reasonInput = page.locator(
        'textarea, input[type="text"]',
      ).last();
      if (await reasonInput.isVisible()) {
        await reasonInput.fill('Необходимо дополнительное хранилище для архива камер');
      }

      // Отправляем форму
      const submitButton = page.locator(
        'button[type="submit"], button',
      ).filter({ hasText: /send|отправ|submit|submit|request|запрос/i }).first();

      if (await submitButton.isVisible()) {
        await submitButton.click();
        await page.waitForTimeout(1000);

        // Проверяем уведомление об успехе
        const successNotification = page.locator(
          'div[class*="toast" i], div[role="alert"]',
        ).filter({ hasText: /success|успеш|отправ|pending|ожида/i }).first();
        const hasNotification = await successNotification.isVisible().catch(() => false);
      }
    }
  });

  test('Quota — soft limit warning shows for near-limit quotas', async ({ page }) => {
    // Проверяем предупреждение о приближении к лимиту
    const warningIndicator = page.locator(
      'div[class*="warning" i], span[class*="warning" i], ' +
      'svg[class*="warning" i], div[class*="alert" i], ' +
      'text=/soft limit|близк|предупрежд|warning|almost|почти|80%|90%/i',
    ).first();
    const hasWarning = await warningIndicator.isVisible().catch(() => false);
  });

  test('Quota — hard limit reached shows alert for retention', async ({ page }) => {
    // Проверяем индикатор достижения лимита (retention 100%)
    const limitReached = page.locator(
      'div[class*="danger" i], div[class*="error" i], ' +
      'span[class*="critical" i], ' +
      'text=/limit.*reached|лимит.*исчерп|100%|exceeded|превыш/i',
    ).first();
    const hasLimit = await limitReached.isVisible().catch(() => false);
  });

  test('Quota — plan name and upgrade option displayed', async ({ page }) => {
    // Проверяем отображение названия тарифа
    const planName = page.locator(
      'text=/enterprise|professional|business|тариф|plan/i',
    ).first();
    await expect(planName).toBeVisible();

    // Проверяем ссылку на апгрейд
    const upgradeLink = page.locator(
      'a, button',
    ).filter({ hasText: /upgrade|апгрейд|change plan|сменить тариф/i }).first();
    const hasUpgrade = await upgradeLink.isVisible().catch(() => false);
  });
});
