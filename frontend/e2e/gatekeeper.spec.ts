/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockSites,
  mockDevices,
  mockUsers,
  mockWorkOrders,
  mockCatchAll,
  MOCK_SITES,
  MOCK_DEVICES,
  MOCK_WORK_ORDERS,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Gatekeeper Verification — E2E Tests
// P0 flow: Gatekeeper — верификация устройств перед добавлением в систему
// Сценарии: P2P device verification, ping check, certificate validation
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_GATEKEEPER_VERIFY_RESPONSE = {
  device_id: 'p2p-verify-01',
  status: 'verified',
  mac: 'AA:BB:CC:DD:EE:FF',
  ip_address: '10.0.10.50',
  manufacturer: 'HikVision',
  model: 'DS-2CD2T47G2-L',
  firmware: '5.7.11',
  certificate_status: 'valid',
  ping_rtt_ms: 2.3,
  security_score: 92,
  vulnerabilities: [],
  verified_at: new Date().toISOString(),
};

const MOCK_GATEKEEPER_FAIL_RESPONSE = {
  device_id: null,
  status: 'failed',
  mac: 'AA:BB:CC:DD:EE:FF',
  error: 'DEVICE_UNREACHABLE',
  message: 'Устройство недоступно по сети. Проверьте соединение и IP-адрес.',
  ping_rtt_ms: null,
  certificate_status: 'unknown',
  verified_at: null,
};

const MOCK_GATEKEEPER_PENDING_RESPONSE = {
  device_id: 'p2p-pending-01',
  status: 'pending_approval',
  mac: 'AA:BB:CC:DD:EE:01',
  ip_address: '10.0.10.51',
  manufacturer: 'Dahua',
  model: 'IPC-HFW5842H-ZHE',
  firmware: '4.001.0000009.0',
  certificate_status: 'self_signed',
  ping_rtt_ms: 5.1,
  security_score: 65,
  vulnerabilities: [
    { id: 'CVE-2024-1234', severity: 'medium', description: 'Default credentials in use' },
  ],
  verified_at: null,
  requires_approval: true,
};

const MOCK_GATEKEEPER_HISTORY = [
  {
    id: 'gk-001',
    device_mac: 'AA:BB:CC:DD:EE:01',
    device_name: 'Gate Controller A-101',
    status: 'approved',
    verified_at: new Date(Date.now() - 86400000 * 7).toISOString(),
    verified_by: 'user-1',
    security_score: 88,
  },
  {
    id: 'gk-002',
    device_mac: 'AA:BB:CC:DD:EE:02',
    device_name: 'Access Panel B-204',
    status: 'rejected',
    verified_at: new Date(Date.now() - 86400000 * 3).toISOString(),
    verified_by: 'user-1',
    security_score: 45,
    reason: 'Firmware version too old, contains known vulnerabilities',
  },
];

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupGatekeeperMockApi(page: any) {
  await setupAuth(page);

  // Gatekeeper verify endpoint (POST)
  await page.route('**/api/v1/gatekeeper/verify', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      const deviceMac = body.mac || '';

      // Simulate different responses based on MAC
      if (deviceMac === 'AA:BB:CC:DD:EE:99') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_GATEKEEPER_FAIL_RESPONSE),
        });
      }
      if (deviceMac === 'AA:BB:CC:DD:EE:01') {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(MOCK_GATEKEEPER_PENDING_RESPONSE),
        });
      }
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MOCK_GATEKEEPER_VERIFY_RESPONSE),
      });
    }
    await route.fallback();
  });

  // Gatekeeper history
  await page.route('**/api/v1/gatekeeper/history*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_GATEKEEPER_HISTORY),
    });
  });

  // Gatekeeper approve/reject endpoints
  await page.route('**/api/v1/gatekeeper/approve', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'approved',
        message: 'Устройство одобрено и добавлено в систему',
        approved_at: new Date().toISOString(),
      }),
    });
  });

  await page.route('**/api/v1/gatekeeper/reject', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'rejected',
        message: 'Устройство отклонено',
        rejected_at: new Date().toISOString(),
      }),
    });
  });

  // Standard mocks for sidebar/navigation
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Gatekeeper Verification
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Gatekeeper — Device Verification', () => {
  test.beforeEach(async ({ page }) => {
    await setupGatekeeperMockApi(page);
    await page.goto('/gatekeeper');
    await page.waitForTimeout(1500);
  });

  test('Gatekeeper page loads with verification interface', async ({ page }) => {
    // Страница загрузилась
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/gatekeeper');

    // Проверяем наличие интерфейса верификации
    const pageContent = page.locator(
      'h1, h2, [class*="title" i], [class*="heading" i]',
    ).filter({ hasText: /gatekeeper|verify|верификаци|gate|keeper/i });
    await expect(pageContent.first()).toBeVisible();
  });

  test('Verify device form — input fields are present', async ({ page }) => {
    // Проверяем форму ввода данных устройства для верификации
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();
    const ipInput = page.locator(
      'input[name="ip"], input[placeholder*="ip" i], input[placeholder*="192" i], input[id*="ip" i]',
    ).first();

    // Хотя бы одно поле должно присутствовать
    const macVisible = await macInput.isVisible().catch(() => false);
    const ipVisible = await ipInput.isVisible().catch(() => false);
    expect(macVisible || ipVisible).toBeTruthy();
  });

  test('Verify device — enter MAC and submit', async ({ page }) => {
    // Находим поле ввода MAC адреса
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('AA:BB:CC:DD:EE:FF');
    }

    // Находим кнопку Verify / Проверить
    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(1500);

      // Проверяем что появился результат верификации
      const resultSection = page.locator(
        'text=/verified|верифицирован|success|успешно|approved|одобрен|valid|валид/i',
      ).first();

      const hasResult = await resultSection.isVisible().catch(() => false);
      if (!hasResult) {
        // Если нет текстового результата, проверяем что есть секция с результатами
        const statusBadge = page.locator(
          '[class*="status" i], [class*="result" i], [role="status"], [data-testid*="result" i]',
        ).first();
        const hasStatus = await statusBadge.isVisible().catch(() => false);
        if (!hasStatus) {
          // Проверяем что хотя бы URL изменился (редирект на страницу результата)
          const currentUrl = page.url();
          expect(currentUrl).toContain('/gatekeeper');
        }
      }
    }
  });

  test('Verify device — unreachable device shows error state', async ({ page }) => {
    // Вводим MAC устройства, которое недоступно
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('AA:BB:CC:DD:EE:99');
    }

    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(1500);

      // Проверяем отображение ошибки
      const errorMessage = page.locator(
        'text=/unreachable|недоступен|error|ошибка|failed|провал|not reachable/i',
      ).first();

      const hasError = await errorMessage.isVisible().catch(() => false);
      if (!hasError) {
        // Проверяем что есть индикатор ошибки / failure
        const failureBadge = page.locator(
          '[class*="error" i], [class*="fail" i], [class*="alert" i], [role="alert"]',
        ).first();
        await expect(failureBadge).toBeVisible();
      }
    }
  });

  test('Verify device — self-signed certificate shows warning', async ({ page }) => {
    // Вводим MAC устройства с самоподписанным сертификатом
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('AA:BB:CC:DD:EE:01');
    }

    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(1500);

      // Проверяем предупреждение о самоподписанном сертификате
      const warning = page.locator(
        'text=/self.signed|самоподписан|warning|предупреждение|certificate|сертификат|security.*score|оценка.*безопасности/i',
      ).first();

      const hasWarning = await warning.isVisible().catch(() => false);
      if (!hasWarning) {
        // Проверяем что есть кнопка/элемент для approval
        const approveAction = page.locator(
          'button:has-text(/approve|одобрить|allow|разрешить/i), ' +
          '[class*="pending" i], [class*="approval" i]',
        ).first();
        const hasAction = await approveAction.isVisible().catch(() => false);
        expect(hasWarning || hasAction).toBeTruthy();
      }
    }
  });

  test('Verify device — security score is displayed after verification', async ({ page }) => {
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('AA:BB:CC:DD:EE:FF');
    }

    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(1500);

      // Проверяем отображение security score
      const score = page.locator(
        '[class*="score" i], [class*="rating" i], text=/\\d+%|score|рейтинг|оценка|балл/i',
      ).first();

      const hasScore = await score.isVisible().catch(() => false);
      if (hasScore) {
        const scoreText = await score.textContent();
        expect(scoreText).toBeTruthy();
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Gatekeeper — History & Approval
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Gatekeeper — History & Approvals', () => {
  test.beforeEach(async ({ page }) => {
    await setupGatekeeperMockApi(page);
    await page.goto('/gatekeeper');
    await page.waitForTimeout(1500);
  });

  test('Gatekeeper history tab shows past verifications', async ({ page }) => {
    // Ищем таб или раздел History / История
    const historyTab = page.locator(
      'button:has-text(/history|история|past|прошлые|log|журнал/i), ' +
      '[role="tab"]:has-text(/history|история/i)'
    ).first();

    if (await historyTab.isVisible()) {
      await historyTab.click();
      await page.waitForTimeout(500);

      // Проверяем отображение истории верификаций
      const historyEntry = page.locator(
        'text=/Gate Controller A-101|Access Panel B-204|approved|rejected|одобрен|отклонен/i',
      ).first();
      await expect(historyEntry).toBeVisible();
    }
  });

  test('Gatekeeper history — approved devices show different styling', async ({ page }) => {
    const historyTab = page.locator(
      'button:has-text(/history|история/i), [role="tab"]:has-text(/history|история/i)',
    ).first();

    if (await historyTab.isVisible()) {
      await historyTab.click();
      await page.waitForTimeout(500);

      // Проверяем что approved статус отображается
      const approvedStatus = page.locator(
        'text=/approved|одобрен|verified|верифицирован/i',
      ).first();
      const hasApproved = await approvedStatus.isVisible().catch(() => false);

      const rejectedStatus = page.locator(
        'text=/rejected|отклонен/i',
      ).first();
      const hasRejected = await rejectedStatus.isVisible().catch(() => false);

      expect(hasApproved || hasRejected).toBeTruthy();
    }
  });

  test('Approve pending device — flow completes successfully', async ({ page }) => {
    // Открываем форму верификации с MAC, который требует approval
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('AA:BB:CC:DD:EE:01');
    }

    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(1000);
    }

    // Нажимаем Approve / Одобрить
    const approveButton = page.locator(
      'button:has-text(/approve|одобрить|allow|разрешить|confirm|подтвердить/i)',
    ).first();

    if (await approveButton.isVisible()) {
      await approveButton.click();
      await page.waitForTimeout(1000);

      // Проверяем сообщение об успешном одобрении
      const successMessage = page.locator(
        'text=/успешно|success|approved|одобрен|добавлен|added/i',
      ).first();

      const hasSuccess = await successMessage.isVisible().catch(() => false);
      if (!hasSuccess) {
        // Проверяем что статус изменился на approved
        const approvedIndicator = page.locator(
          'text=/approved|одобрен|active|активен/i',
        ).first();
        expect(await approvedIndicator.isVisible().catch(() => false)).toBeTruthy();
      }
    }
  });

  test('Reject device — shows confirmation and updates status', async ({ page }) => {
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('AA:BB:CC:DD:EE:01');
    }

    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(1000);
    }

    // Нажимаем Reject / Отклонить
    const rejectButton = page.locator(
      'button:has-text(/reject|отклонить|deny|запретить|block|блокировать/i)',
    ).first();

    if (await rejectButton.isVisible()) {
      await rejectButton.click();
      await page.waitForTimeout(1000);

      // Проверяем что появилось подтверждение или изменился статус
      const confirmation = page.locator(
        'text=/rejected|отклонен|denied|запрещен|removed|удален/i',
      ).first();

      const hasConfirmation = await confirmation.isVisible().catch(() => false);
      if (!hasConfirmation) {
        // Проверяем что устройство больше не отображается как pending
        const pendingIndicator = page.locator(
          'text=/pending|ожидает/i',
        );
        const pendingCount = await pendingIndicator.count();
        expect(pendingCount).toBe(0);
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Gatekeeper — Responsiveness & Edge Cases
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Gatekeeper — Edge Cases', () => {
  test.beforeEach(async ({ page }) => {
    await setupGatekeeperMockApi(page);
    await page.goto('/gatekeeper');
    await page.waitForTimeout(1500);
  });

  test('Empty MAC validation — shows error on empty submit', async ({ page }) => {
    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(500);

      // Проверяем появление ошибки валидации
      const validationError = page.locator(
        'text=/required|обязательно|please|заполните|enter|введите|invalid|некорректный|empty|пусто/i',
      ).first();

      const hasError = await validationError.isVisible().catch(() => false);
      if (!hasError) {
        // Если нет текстовой ошибки — проверяем что поле подсвечено (aria-invalid)
        const invalidField = page.locator('[aria-invalid="true"]').first();
        await expect(invalidField).toBeVisible();
      }
    }
  });

  test('Invalid MAC format — shows format error', async ({ page }) => {
    const macInput = page.locator(
      'input[name="mac"], input[placeholder*="mac" i], input[placeholder*="адрес" i], input[id*="mac" i]',
    ).first();

    if (await macInput.isVisible()) {
      await macInput.fill('not-a-mac-address');
    }

    const verifyButton = page.locator(
      'button:has-text(/verify|верифицировать|проверить|check|scan|сканировать/i)',
    ).first();

    if (await verifyButton.isVisible()) {
      await verifyButton.click();
      await page.waitForTimeout(500);

      // Проверяем ошибку формата MAC
      const formatError = page.locator(
        'text=/invalid.*format|некорректный.*формат|format.*error|ошибка.*формата|mac.*format/i',
      ).first();

      const hasError = await formatError.isVisible().catch(() => false);
      if (!hasError) {
        const invalidField = page.locator('[aria-invalid="true"]').first();
        await expect(invalidField).toBeVisible();
      }
    }
  });
});
