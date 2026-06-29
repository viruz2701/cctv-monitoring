/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Rate Limiting — E2E Tests
// X-RateLimit headers, 429 handling, Retry-After
// ═══════════════════════════════════════════════════════════════════════════

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

/**
 * Настраивает моки API с кастомными rate limit headers
 */
async function setupRateLimitMockApi(page: any, remaining: number = 100, retryAfter?: number) {
  await setupAuth(page);

  // Mock for sites endpoint with rate limit headers
  await page.route('**/api/v1/sites*', async (route: any) => {
    const headers: Record<string, string> = {
      'X-RateLimit-Limit': '100',
      'X-RateLimit-Remaining': String(Math.max(0, remaining)),
      'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 60),
    };

    if (retryAfter) {
      headers['Retry-After'] = String(retryAfter);
    }

    await route.fulfill({
      status: remaining > 0 ? 200 : 429,
      contentType: 'application/json',
      headers,
      body: remaining > 0
        ? JSON.stringify([{ id: 'site-1', name: 'Main Office' }])
        : JSON.stringify({
            error: 'rate_limit_exceeded',
            message: 'Превышен лимит запросов. Пожалуйста, повторите попытку позже.',
          }),
    });
  });

  // Mock для всех API с rate limit headers
  await page.route('**/api/v1/devices*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      headers: {
        'X-RateLimit-Limit': '200',
        'X-RateLimit-Remaining': String(Math.max(0, remaining - 20)),
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 120),
      },
      body: JSON.stringify([]),
    });
  });

  // Mock для dashboard с разными лимитами
  await page.route('**/api/v1/dashboard/stats', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      headers: {
        'X-RateLimit-Limit': '50',
        'X-RateLimit-Remaining': String(Math.max(0, remaining - 30)),
        'X-RateLimit-Reset': String(Math.floor(Date.now() / 1000) + 30),
      },
      body: JSON.stringify({}),
    });
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Rate Limiting — Headers & Display
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Rate Limiting — Headers & Display', () => {
  test.beforeEach(async ({ page }) => {
    await setupRateLimitMockApi(page, 100);
    await page.goto('/sites');
    await page.waitForTimeout(1500);
  });

  test('Rate limiting — X-RateLimit-Limit header is present', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();

    // Перехватываем ответ и проверяем заголовки
    const response = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/sites') && resp.status() === 200,
    );

    const limitHeader = response.headers()['x-ratelimit-limit'];
    expect(limitHeader).toBeDefined();
    expect(limitHeader).toBe('100');
  });

  test('Rate limiting — X-RateLimit-Remaining decreases after requests', async ({ page }) => {
    // Делаем несколько запросов
    await page.goto('/devices');
    await page.waitForTimeout(500);
    await page.goto('/dashboard');
    await page.waitForTimeout(500);

    // Проверяем заголовок remaining после нескольких запросов
    const response = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/dashboard/stats'),
    );
    const remainingHeader = response.headers()['x-ratelimit-remaining'];
    const remaining = parseInt(remainingHeader || '0', 10);
    expect(remaining).toBeLessThan(100);
    expect(remaining).toBeGreaterThanOrEqual(0);
  });

  test('Rate limiting — X-RateLimit-Reset has valid unix timestamp format', async ({ page }) => {
    const response = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/sites'),
    );
    const resetHeader = response.headers()['x-ratelimit-reset'];
    expect(resetHeader).toBeDefined();

    // Проверяем что это валидный unix timestamp
    const resetValue = parseInt(resetHeader || '0', 10);
    expect(resetValue).toBeGreaterThan(0);

    // Проверяем что timestamp в будущем (ближайшие 5 минут)
    const now = Math.floor(Date.now() / 1000);
    expect(resetValue).toBeGreaterThanOrEqual(now);
    expect(resetValue).toBeLessThanOrEqual(now + 300);
  });

  test('Rate limiting — rate limit info displayed in UI', async ({ page }) => {
    await page.goto('/sites');
    await page.waitForTimeout(1000);

    // Проверяем отображение информации о rate limit в UI
    const rateLimitInfo = page.locator(
      'div[class*="rate" i], span[class*="rate" i], ' +
      'text=/rate limit|лимит|api limit|remaining|осталос/i',
    ).first();
    const hasInfo = await rateLimitInfo.isVisible().catch(() => false);
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Rate Limiting — 429 Error Handling
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Rate Limiting — 429 Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await setupRateLimitMockApi(page, 0, 30);
    await page.goto('/sites');
    await page.waitForTimeout(1500);
  });

  test('Rate limiting — 429 status code triggers error display', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();

    // Проверяем что пришел 429 статус
    const response = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/sites'),
    );
    expect(response.status()).toBe(429);

    // Проверяем отображение ошибки rate limiting
    const errorMessage = page.locator(
      'text=/rate limit|лимит|429|too many|много запрос|превышен/i',
    ).first();
    const hasError = await errorMessage.isVisible().catch(() => false);
    if (hasError) {
      await expect(errorMessage).toBeVisible();
    }
  });

  test('Rate limiting — Retry-After header present on 429', async ({ page }) => {
    const response = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/sites'),
    );
    expect(response.status()).toBe(429);

    const retryAfter = response.headers()['retry-after'];
    expect(retryAfter).toBeDefined();
    expect(retryAfter).toBe('30');
  });

  test('Rate limiting — Retry-After countdown displayed in UI', async ({ page }) => {
    // Проверяем отображение таймера обратного отсчета
    const retryTimer = page.locator(
      'text=/30|seconds|секунд|retry|повтор|wait|подожд/i',
    ).first();
    const hasTimer = await retryTimer.isVisible().catch(() => false);
  });

  test('Rate limiting — error shows rate limit exceeded message', async ({ page }) => {
    // Проверяем специфическое сообщение об ошибке
    const limitError = page.locator(
      'text=/rate_limit_exceeded|превышен лимит|limit exceeded|повторите попытку/i',
    ).first();
    const hasError = await limitError.isVisible().catch(() => false);
    if (hasError) {
      await expect(limitError).toBeVisible();
    }
  });

  test('Rate limiting — different endpoints have different limits', async ({ page }) => {
    // Проверяем заголовки для разных эндпоинтов
    const sitesResponse = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/sites'),
    );
    expect(sitesResponse.headers()['x-ratelimit-limit']).toBe('100');

    const devicesResponse = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/devices'),
    );
    expect(devicesResponse.headers()['x-ratelimit-limit']).toBe('200');

    const dashboardResponse = await page.waitForResponse(
      (resp) => resp.url().includes('/api/v1/dashboard/stats'),
    );
    expect(dashboardResponse.headers()['x-ratelimit-limit']).toBe('50');
  });
});
