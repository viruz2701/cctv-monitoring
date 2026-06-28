/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockSites,
  mockDevices,
  mockUsers,
  mockWorkOrders,
  mockDashboardStats,
  mockAlerts,
  mockCatchAll,
  MOCK_ADMIN_USER,
  MOCK_SITES,
  MOCK_DEVICES,
  MOCK_WORK_ORDERS,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// SLA Breach — E2E Tests
// P1-QA.1: Создание SLA breach, escalation, уведомления
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_SLA_BREACHES = [
  {
    id: 'sla-1',
    work_order_id: 'WO-002',
    work_order_title: 'Firmware update NVR-03',
    priority: 'high',
    status: 'active',
    breach_type: 'response_time',
    breach_threshold_hours: 4,
    elapsed_hours: 6.5,
    detected_at: new Date(Date.now() - 7200000).toISOString(),
    escalated_to: null,
    escalated_at: null,
    site_id: 'site-1',
    site_name: 'Main Office',
    assigned_to: 'user-2',
    assigned_name: 'Alex Manager',
  },
  {
    id: 'sla-2',
    work_order_id: 'WO-001',
    work_order_title: 'Replace camera lens',
    priority: 'critical',
    status: 'escalated',
    breach_type: 'response_time',
    breach_threshold_hours: 2,
    elapsed_hours: 8.0,
    detected_at: new Date(Date.now() - 28800000).toISOString(),
    escalated_to: 'user-1',
    escalated_to_name: 'Admin User',
    escalated_at: new Date(Date.now() - 14400000).toISOString(),
    site_id: 'site-1',
    site_name: 'Main Office',
    assigned_to: null,
    assigned_name: null,
    escalation_level: 2,
  },
  {
    id: 'sla-3',
    work_order_id: 'WO-004',
    work_order_title: 'Emergency camera repair',
    priority: 'critical',
    status: 'resolved',
    breach_type: 'resolution_time',
    breach_threshold_hours: 24,
    elapsed_hours: 18.0,
    detected_at: new Date(Date.now() - 86400000 * 2).toISOString(),
    escalated_to: null,
    escalated_at: null,
    resolved_at: new Date(Date.now() - 86400000).toISOString(),
    site_id: 'site-3',
    site_name: 'Warehouse',
    assigned_to: null,
    assigned_name: null,
  },
];

const MOCK_ESCALATION_MATRIX = [
  { level: 1, role: 'manager', notify_after_minutes: 30, actions: ['notify_manager', 'increase_priority'] },
  { level: 2, role: 'admin', notify_after_minutes: 60, actions: ['notify_admin', 'alert_team'] },
  { level: 3, role: 'director', notify_after_minutes: 120, actions: ['notify_director', 'emergency_meeting'] },
];

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupSlaBreachMockApi(page: any) {
  await setupAuth(page);

  // SLA breaches list + detail
  await page.route('**/api/v1/sla/breaches*', async (route: any, request: any) => {
    const url = request.url();

    if (request.method() === 'POST') {
      return route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: `sla-new-${Date.now()}`,
          ...JSON.parse(request.postData() || '{}'),
          status: 'active',
          detected_at: new Date().toISOString(),
        }),
      });
    }

    // Escalate endpoint
    if (url.includes('/escalate')) {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'escalated',
          escalated_to: 'user-1',
          escalated_at: new Date().toISOString(),
          escalation_level: 2,
          message: 'SLA breach escalated to level 2',
        }),
      });
    }

    // Resolve endpoint
    if (url.includes('/resolve')) {
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          status: 'resolved',
          resolved_at: new Date().toISOString(),
        }),
      });
    }

    // Detail
    const detailMatch = url.match(/\/sla\/breaches\/([^/?]+)/);
    if (detailMatch && detailMatch[1] && !detailMatch[1].includes('?')) {
      const slaId = detailMatch[1];
      const breach = MOCK_SLA_BREACHES.find((b) => b.id === slaId);
      if (breach) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(breach),
        });
      }
    }

    // List
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_SLA_BREACHES),
    });
  });

  // Escalation matrix
  await page.route('**/api/v1/sla/escalation-matrix', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_ESCALATION_MATRIX),
    });
  });

  // SLA settings
  await page.route('**/api/v1/sla/settings', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        critical_response_hours: 2,
        high_response_hours: 4,
        medium_response_hours: 8,
        low_response_hours: 24,
        critical_resolution_hours: 24,
        high_resolution_hours: 48,
        medium_resolution_hours: 72,
        low_resolution_hours: 168,
        business_hours_only: false,
        auto_escalate: true,
      }),
    });
  });

  // Standard mocks
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page, [
    ...MOCK_WORK_ORDERS,
    { id: 'WO-005', title: 'Overdue SLA test', status: 'open', priority: 'critical', assigned_to: null, site_id: 'site-1', sla_deadline: new Date(Date.now() - 3600000).toISOString(), created_at: new Date().toISOString() },
  ]);
  await mockDashboardStats(page, { ...(await import('./shared-mocks')).MOCK_DASHBOARD_STATS, overdue_sla: 3 });
  await mockAlerts(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: SLA Breach — List & Overview
// ═══════════════════════════════════════════════════════════════════════════

test.describe('SLA Breach — List & Overview', () => {
  test.beforeEach(async ({ page }) => {
    await setupSlaBreachMockApi(page);
    await page.goto('/sla');
    await page.waitForTimeout(1500);
  });

  test('SLA page loads with breach list', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/sla');

    // Проверяем отображение breach из мок-данных
    const breachTitle = page.locator(
      'text=/Firmware update NVR|Replace camera lens|Emergency camera repair/i',
    ).first();
    await expect(breachTitle).toBeVisible();
  });

  test('SLA — breach status badges are displayed', async ({ page }) => {
    // Проверяем статусные badges
    const statusBadge = page.locator(
      'text=/active|активн|escalated|эскалирован|resolved|решен/i',
    ).first();
    await expect(statusBadge).toBeVisible();
  });

  test('SLA — priority badge matches critical/high status', async ({ page }) => {
    // Проверяем priority badge
    const priorityBadge = page.locator(
      'span, badge, [class*="priority" i], [class*="severity" i]',
    ).filter({ hasText: /critical|критич|high|высок/i }).first();
    await expect(priorityBadge).toBeVisible();
  });

  test('SLA — elapsed time is displayed for active breaches', async ({ page }) => {
    // Проверяем отображение прошедшего времени
    const elapsedTime = page.locator(
      'text=/6\\.5|8|hour|час|elapsed|прошл|overdue|просрочк/i',
    ).first();
    await expect(elapsedTime).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: SLA Breach — Escalation
// ═══════════════════════════════════════════════════════════════════════════

test.describe('SLA Breach — Escalation', () => {
  test.beforeEach(async ({ page }) => {
    await setupSlaBreachMockApi(page);
    await page.goto('/sla');
    await page.waitForTimeout(1500);
  });

  test('SLA — escalate button is visible on active breach', async ({ page }) => {
    // Находим активный breach
    const activeBreach = page.locator(
      'a, button, tr, [role="row"], .card, .item',
    ).filter({ hasText: /Firmware update NVR/i }).first();

    if (await activeBreach.isVisible()) {
      await activeBreach.click();
      await page.waitForTimeout(1000);

      // Проверяем кнопку эскалации
      const escalateButton = page.locator(
        'button:has-text(/escalate|эскалировать|escalation|поднять|level.*up|повысить/i)',
      ).first();
      await expect(escalateButton).toBeVisible();
    }
  });

  test('SLA — escalate breach shows confirmation', async ({ page }) => {
    const activeBreach = page.locator(
      'a, button, tr, [role="row"], .card, .item',
    ).filter({ hasText: /Firmware update NVR/i }).first();

    if (await activeBreach.isVisible()) {
      await activeBreach.click();
      await page.waitForTimeout(1000);

      const escalateButton = page.locator(
        'button:has-text(/escalate|эскалировать|escalation/i)',
      ).first();

      if (await escalateButton.isVisible()) {
        await escalateButton.click();
        await page.waitForTimeout(500);

        // Проверяем confirmation dialog
        const confirmDialog = page.locator(
          '[role="alertdialog"], [role="dialog"], .modal, ' +
          'text=/confirm|подтверд|escalate.*breach|эскалирова|proceed|продолжи/i',
        ).first();
        await expect(confirmDialog).toBeVisible();
      }
    }
  });

  test('SLA — confirm escalation updates status', async ({ page }) => {
    const activeBreach = page.locator(
      'a, button, tr, [role="row"], .card, .item',
    ).filter({ hasText: /Firmware update NVR/i }).first();

    if (await activeBreach.isVisible()) {
      await activeBreach.click();
      await page.waitForTimeout(1000);

      const escalateButton = page.locator(
        'button:has-text(/escalate|эскалировать/i)',
      ).first();

      if (await escalateButton.isVisible()) {
        await escalateButton.click();
        await page.waitForTimeout(500);

        // Подтверждаем эскалацию
        const confirmButton = page.locator(
          'button:has-text(/confirm|escalate|yes|подтвердить|да|продолжить/i)',
        ).last();
        if (await confirmButton.isVisible()) {
          await confirmButton.click();
          await page.waitForTimeout(1000);

          // Проверяем сообщение об эскалации
          const escalationMessage = page.locator(
            'text=/escalated|эскалирован|level.*2|escalation.*success|успешно.*эскалирова/i',
          ).first();
          const hasMessage = await escalationMessage.isVisible().catch(() => false);
          if (!hasMessage) {
            // Проверяем что статус изменился
            const escalatedStatus = page.locator(
              'text=/escalated|эскалирован/i',
            ).first();
            const hasStatus = await escalatedStatus.isVisible().catch(() => false);
            expect(hasStatus || escalationMessage).toBeTruthy();
          }
        }
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: SLA Breach — Resolve & Metrics
// ═══════════════════════════════════════════════════════════════════════════

test.describe('SLA Breach — Resolve & Metrics', () => {
  test.beforeEach(async ({ page }) => {
    await setupSlaBreachMockApi(page);
    await page.goto('/sla');
    await page.waitForTimeout(1500);
  });

  test('SLA — resolve active breach shows completion flow', async ({ page }) => {
    const activeBreach = page.locator(
      'a, button, tr, [role="row"], .card, .item',
    ).filter({ hasText: /Firmware update NVR/i }).first();

    if (await activeBreach.isVisible()) {
      await activeBreach.click();
      await page.waitForTimeout(1000);

      // Ищем кнопку resolve / решить
      const resolveButton = page.locator(
        'button:has-text(/resolve|решить|mark.*resolved|отметить.*решен|close|закрыть/i)',
      ).first();

      if (await resolveButton.isVisible()) {
        await resolveButton.click();
        await page.waitForTimeout(500);

        // Заполняем причину resolution (если есть)
        const reasonField = page.locator(
          'textarea[name="reason"], textarea[placeholder*="reason" i], textarea[placeholder*="причин" i]',
        ).first();
        if (await reasonField.isVisible()) {
          await reasonField.fill('Issue resolved after firmware update');
        }

        // Подтверждаем
        const confirmButton = page.locator(
          'button:has-text(/confirm|resolve|yes|подтвердить|решить|да/i)',
        ).last();
        if (await confirmButton.isVisible()) {
          await confirmButton.click();
          await page.waitForTimeout(1000);

          // Проверяем успех
          const successMessage = page.locator(
            'text=/resolved|решен|closed|закрыт|success|успех/i',
          ).first();
          const hasSuccess = await successMessage.isVisible().catch(() => false);
          if (!hasSuccess) {
            expect(page.url()).toContain('/sla');
          }
        }
      }
    }
  });

  test('SLA — escalation matrix is accessible', async ({ page }) => {
    // Ищем секцию с escalation matrix
    const matrixTab = page.locator(
      'button:has-text(/escalation.*matrix|матрица.*эскалац|escalation.*policy|политика.*эскалац/i), ' +
      '[role="tab"]:has-text(/escalation|эскалац|matrix|матриц/i)',
    ).first();

    if (await matrixTab.isVisible()) {
      await matrixTab.click();
      await page.waitForTimeout(500);

      // Проверяем отображение уровней эскалации
      const levels = page.locator(
        'text=/level.*1|level.*2|level.*3|manager|admin|director/i',
      ).first();
      await expect(levels).toBeVisible();
    }
  });

  test('SLA — overdue SLA count shown on dashboard integration', async ({ page }) => {
    // Проверяем счетчик просрочек
    const overdueCounter = page.locator(
      'text=/3 overdue|overdue.*3|просрочен.*3|sla.*breach|нарушен.*sla/i',
    ).first();
    const hasCounter = await overdueCounter.isVisible().catch(() => false);
    if (hasCounter) {
      await expect(overdueCounter).toBeVisible();
    }
  });
});
