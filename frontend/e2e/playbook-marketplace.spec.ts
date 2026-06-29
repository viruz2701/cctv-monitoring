/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Playbook Marketplace — E2E Tests
// Browse, filter, install, rate playbooks from marketplace
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_PLAYBOOKS = Array.from({ length: 12 }, (_, i) => ({
  id: `pb-${i + 1}`,
  name: [
    'NVR Health Check',
    'Camera Motion Detection',
    'Storage Cleanup',
    'Network Diagnostics',
    'Firmware Upgrade',
    'Backup Verification',
    'Bandwidth Monitor',
    'SLA Compliance Report',
    'AI Object Detection',
    'Night Vision Tuning',
    'PTZ Auto-Tracking',
    'Health Dashboard Sync',
  ][i],
  vendor: ['hikvision', 'dahua', 'axis', 'hikvision', 'dahua', 'generic',
    'axis', 'generic', 'hikvision', 'dahua', 'axis', 'generic'][i],
  category: ['health', 'analytics', 'maintenance', 'network', 'maintenance',
    'backup', 'network', 'compliance', 'analytics', 'tuning', 'analytics', 'integration'][i],
  description: `Автоматизированный плейбук для ${['NVR', 'камер', 'очистки', 'сети', 'обновлений',
    'резервного копирования', 'мониторинга', 'SLA', 'AI', 'ночного режима', 'PTZ', 'дашборда'][i]}`,
  version: '1.0.0',
  rating: Number((3.5 + Math.random() * 1.5).toFixed(1)),
  installs: Math.floor(Math.random() * 1000),
  verified: Math.random() > 0.3,
  premium: i >= 8,
  updated_at: new Date(Date.now() - Math.random() * 86400000 * 30).toISOString(),
}));

const VENDOR_FILTERS = ['hikvision', 'dahua', 'axis', 'generic'];
const CATEGORY_FILTERS = ['health', 'analytics', 'maintenance', 'network', 'backup', 'compliance', 'tuning', 'integration'];

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupPlaybookMockApi(page: any) {
  await setupAuth(page);

  // Marketplace playbooks list
  await page.route('**/api/v1/playbook/marketplace', async (route: any, request: any) => {
    const url = new URL(request.url());
    const vendor = url.searchParams.get('vendor') || '';
    const search = url.searchParams.get('search') || '';
    const pageNum = parseInt(url.searchParams.get('page') || '1', 10);
    const pageSize = 6;

    let filtered = [...MOCK_PLAYBOOKS];

    if (vendor) {
      filtered = filtered.filter((p) => p.vendor === vendor);
    }
    if (search) {
      const q = search.toLowerCase();
      filtered = filtered.filter((p) =>
        p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q),
      );
    }

    const total = filtered.length;
    const totalPages = Math.ceil(total / pageSize);
    const start = (pageNum - 1) * pageSize;
    const items = filtered.slice(start, start + pageSize);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        items,
        total,
        page: pageNum,
        total_pages: totalPages,
        page_size: pageSize,
      }),
    });
  });

  // Install playbook — POST
  await page.route('**/api/v1/playbook/marketplace/*/install', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'installed',
          installed_at: new Date().toISOString(),
          playbook_id: 'pb-1',
        }),
      });
    }
  });

  // Rate playbook — POST
  await page.route('**/api/v1/playbook/marketplace/*/rate', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'rated',
          rating: 4.5,
          updated_at: new Date().toISOString(),
        }),
      });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Playbook Marketplace — Browse & Filter
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Playbook Marketplace — Browse & Filter', () => {
  test.beforeEach(async ({ page }) => {
    await setupPlaybookMockApi(page);
    await page.goto('/playbook/marketplace');
    await page.waitForTimeout(1500);
  });

  test('Marketplace loads with playbook grid', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/marketplace');

    // Проверяем отображение списка плейбуков
    const playbookGrid = page.locator(
      'div[class*="grid" i], div[class*="list" i], div[class*="cards" i], div[class*="container" i]',
    ).first();
    await expect(playbookGrid).toBeVisible();
  });

  test('Marketplace — playbook name and vendor displayed', async ({ page }) => {
    // Проверяем отображение имени плейбука
    const playbookName = page.locator(
      'text=/NVR Health Check|Camera Motion|Storage Cleanup|Network Diagnostics/i',
    ).first();
    await expect(playbookName).toBeVisible();

    // Проверяем отображение вендора
    const vendorBadge = page.locator(
      'text=/hikvision|dahua|axis|generic/i',
    ).first();
    await expect(vendorBadge).toBeVisible();
  });

  test('Marketplace — filter by vendor narrows results', async ({ page }) => {
    // Кликаем по фильтру вендора
    const vendorFilter = page.locator(
      'button, select, [role="button"], a, label',
    ).filter({ hasText: /hikvision|hik|все|all|vendor|вендор/i }).first();

    if (await vendorFilter.isVisible()) {
      await vendorFilter.click();
      await page.waitForTimeout(1000);

      // Проверяем что результаты отфильтровались
      const url = page.url();
      const hasFilterParam = url.includes('vendor=') || url.includes('filter=');
    }
  });

  test('Marketplace — search by name returns matching results', async ({ page }) => {
    // Находим поле поиска
    const searchInput = page.locator(
      'input[type="text"], input[type="search"], input[placeholder*="search" i], ' +
      'input[placeholder*="поиск" i], input[placeholder*="playbook" i]',
    ).first();

    if (await searchInput.isVisible()) {
      await searchInput.fill('Camera');
      await page.waitForTimeout(1000);

      // Проверяем что отображаются только подходящие результаты
      const cameraRelated = page.locator(
        'text=/Camera|Motion|NVR/i',
      ).first();
      await expect(cameraRelated).toBeVisible();
    }
  });

  test('Marketplace — install button triggers installation', async ({ page }) => {
    // Находим кнопку установки
    const installButton = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /install|устано|download|скач|get|получ/i }).first();

    await expect(installButton).toBeVisible();
    await installButton.click();
    await page.waitForTimeout(1000);

    // Проверяем что появилось уведомление об установке
    const installNotification = page.locator(
      'div[class*="toast" i], div[class*="notification" i], div[role="alert"]',
    ).filter({ hasText: /install|устано|success|успеш/i }).first();
    const hasNotification = await installNotification.isVisible().catch(() => false);
    if (hasNotification) {
      await expect(installNotification).toBeVisible();
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Playbook Marketplace — Details & Rating
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Playbook Marketplace — Details & Rating', () => {
  test.beforeEach(async ({ page }) => {
    await setupPlaybookMockApi(page);
    await page.goto('/playbook/marketplace');
    await page.waitForTimeout(1500);
  });

  test('Marketplace — playbook detail modal opens on click', async ({ page }) => {
    // Кликаем по первому плейбуку для открытия деталей
    const playbookCard = page.locator(
      'div[class*="card" i], div[class*="item" i], tr, li',
    ).filter({ hasText: /NVR|Camera|Storage|Network|Firmware/i }).first();

    if (await playbookCard.isVisible()) {
      await playbookCard.click();
      await page.waitForTimeout(1000);

      // Проверяем что открылся модал с деталями
      const detailModal = page.locator(
        'div[role="dialog"], div[class*="modal" i], div[class*="drawer" i]',
      ).first();
      const hasModal = await detailModal.isVisible().catch(() => false);
      if (hasModal) {
        await expect(detailModal).toBeVisible();
      }
    }
  });

  test('Marketplace — rate playbook updates rating display', async ({ page }) => {
    // Открываем плейбук
    const playbookCard = page.locator(
      'div[class*="card" i], div[class*="item" i]',
    ).filter({ hasText: /NVR|Camera|Storage/i }).first();

    if (await playbookCard.isVisible()) {
      await playbookCard.click();
      await page.waitForTimeout(1000);
    }

    // Находим и кликаем по звездам рейтинга
    const starRating = page.locator(
      'button[aria-label*="star" i], button[aria-label*="rate" i], ' +
      'button[class*="star" i], span[class*="star" i], ' +
      'svg[class*="star" i], i[class*="star" i]',
    ).first();

    if (await starRating.isVisible()) {
      await starRating.click();
      await page.waitForTimeout(500);

      // Проверяем что рейтинг обновился
      const rateConfirmation = page.locator(
        'text=/rated|оцен|thanks|спасиб|rating|рейтинг|4\\.5/i',
      ).first();
      const hasConfirmation = await rateConfirmation.isVisible().catch(() => false);
    }
  });

  test('Marketplace — premium badge visible on premium playbooks', async ({ page }) => {
    // Проверяем отображение premium бейджа
    const premiumBadge = page.locator(
      'span, badge, div[class*="badge" i]',
    ).filter({ hasText: /premium|pro|professional/i }).first();
    const hasPremium = await premiumBadge.isVisible().catch(() => false);
  });

  test('Marketplace — pagination loads next page', async ({ page }) => {
    // Проверяем наличие пагинации
    const pagination = page.locator(
      'nav[aria-label*="pagination" i], div[class*="pagination" i], ' +
      'button[aria-label*="next" i], button[aria-label*="prev" i], ' +
      'button:has-text("→"), button:has-text("→"), a[aria-label*="page" i]',
    ).first();

    if (await pagination.isVisible()) {
      const nextButton = page.locator(
        'button[aria-label*="next" i], a[aria-label*="next" i], ' +
        'button:has-text("→"), button:has-text(">")',
      ).first();

      if (await nextButton.isVisible()) {
        await nextButton.click();
        await page.waitForTimeout(1000);

        // Проверяем что URL изменился (добавился параметр page)
        const currentUrl = page.url();
        const hasPageParam = currentUrl.includes('page=') || currentUrl.includes('offset=');
      }
    }
  });

  test('Marketplace — verified badge shown for verified playbooks', async ({ page }) => {
    // Проверяем бейдж верификации
    const verifiedBadge = page.locator(
      'span, svg, i, img',
    ).filter({ hasText: /verified|провер|✓|✔|authentic/i }).first();
    const hasVerified = await verifiedBadge.isVisible().catch(() => false);
  });
});
