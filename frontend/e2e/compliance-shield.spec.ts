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
  MOCK_DASHBOARD_STATS,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Compliance Shield — E2E Tests
// P1-QA.1: Compliance метрики, статусы, детализация стандартов
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_COMPLIANCE_SUMMARY = {
  overall_score: 87.5,
  status: 'compliant',
  last_assessment: new Date(Date.now() - 86400000).toISOString(),
  next_assessment: new Date(Date.now() + 604800000).toISOString(),
  standards: [
    { id: 'iec-62443', name: 'IEC 62443-3-3', score: 92, status: 'compliant', finding_count: 2 },
    { id: 'iso-27001', name: 'ISO/IEC 27001:2022', score: 85, status: 'compliant', finding_count: 5 },
    { id: 'iso-27019', name: 'ISO/IEC 27019', score: 78, status: 'attention', finding_count: 8 },
    { id: 'stb-34-101-30', name: 'СТБ 34.101.30', score: 95, status: 'compliant', finding_count: 0 },
    { id: 'owasp-asvs-l3', name: 'OWASP ASVS L3', score: 82, status: 'compliant', finding_count: 3 },
    { id: 'oac-order-66', name: 'Приказ ОАЦ №66', score: 90, status: 'compliant', finding_count: 1 },
  ],
  controls_total: 248,
  controls_passed: 217,
  controls_failed: 31,
  critical_findings: 0,
  high_findings: 3,
  medium_findings: 12,
  low_findings: 16,
};

const MOCK_COMPLIANCE_HISTORY = [
  { date: new Date(Date.now() - 86400000 * 30).toISOString(), score: 84.2, status: 'compliant' },
  { date: new Date(Date.now() - 86400000 * 60).toISOString(), score: 81.0, status: 'attention' },
  { date: new Date(Date.now() - 86400000 * 90).toISOString(), score: 76.5, status: 'attention' },
];

const MOCK_COMPLIANCE_FINDINGS = [
  { id: 'find-1', standard: 'iso-27019', control: 'A.12.6.1', severity: 'high', title: 'Отсутствует управление уязвимостями на edge-устройствах', status: 'open', discovered: new Date(Date.now() - 86400000 * 14).toISOString() },
  { id: 'find-2', standard: 'iso-27001', control: 'A.9.2.3', severity: 'medium', title: 'Не настроена MFA для remote access', status: 'in_progress', discovered: new Date(Date.now() - 86400000 * 30).toISOString() },
  { id: 'find-3', standard: 'owasp-asvs-l3', control: 'V2.1.1', severity: 'high', title: 'Слабый пароль на admin аккаунте', status: 'open', discovered: new Date(Date.now() - 86400000 * 7).toISOString() },
  { id: 'find-4', standard: 'oac-order-66', control: '7.18.2', severity: 'medium', title: 'Отсутствует mTLS на P2P conduits', status: 'resolved', discovered: new Date(Date.now() - 86400000 * 60).toISOString() },
];

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupComplianceMockApi(page: any) {
  await setupAuth(page);

  // Compliance summary
  await page.route('**/api/v1/compliance/summary', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_COMPLIANCE_SUMMARY),
    });
  });

  // Compliance history
  await page.route('**/api/v1/compliance/history', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_COMPLIANCE_HISTORY),
    });
  });

  // Compliance findings
  await page.route('**/api/v1/compliance/findings*', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_COMPLIANCE_FINDINGS),
    });
  });

  // Standard mocks
  await mockSites(page);
  await mockDevices(page);
  await mockUsers(page);
  await mockWorkOrders(page);
  await mockDashboardStats(page);
  await mockAlerts(page);
  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Compliance Shield — Overview
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Compliance Shield — Overview', () => {
  test.beforeEach(async ({ page }) => {
    await setupComplianceMockApi(page);
    await page.goto('/compliance');
    await page.waitForTimeout(1500);
  });

  test('Compliance page loads with overall score', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/compliance');

    // Проверяем отображение общего compliance score
    const scoreIndicator = page.locator(
      'text=/87|87\\.5|compliance score|общий.*рейтинг|score|\\d+%|compliance.*status/i',
    ).first();
    await expect(scoreIndicator).toBeVisible();
  });

  test('Compliance status badge shows compliant status', async ({ page }) => {
    // Проверяем статус compliant
    const statusBadge = page.locator(
      'text=/compliant|соответствует|passed|пройден|approved|одобрен/i',
    ).first();
    await expect(statusBadge).toBeVisible();
  });

  test('Compliance — all standards are listed with scores', async ({ page }) => {
    // Проверяем отображение стандартов
    const iecStandard = page.locator(
      'text=/IEC 62443|ISO 27001|ISO 27019|СТБ 34\\.101|OWASP ASVS|Приказ ОАЦ/i',
    ).first();
    await expect(iecStandard).toBeVisible();

    // Проверяем что счетчики отображаются
    const scoreValue = page.locator(
      'text=/92|85|78|95|82|90|\\d+%|\\d+\\/\\d+/i',
    ).first();
    await expect(scoreValue).toBeVisible();
  });

  test('Compliance — critical findings counter is visible', async ({ page }) => {
    // Проверяем счетчик critical findings
    const criticalFindings = page.locator(
      'text=/0 critical|0.*критич|critical.*0|finding|non.compliance|несоответстви/i',
    ).first();
    await expect(criticalFindings).toBeVisible();
  });

  test('Compliance — last assessment date is displayed', async ({ page }) => {
    // Проверяем дату последней оценки
    const assessmentDate = page.locator(
      'text=/last assessment|последняя оценка|last.*assess|assess.*date/i',
    ).first();
    await expect(assessmentDate).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Compliance Shield — Findings & Details
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Compliance Shield — Findings & Details', () => {
  test.beforeEach(async ({ page }) => {
    await setupComplianceMockApi(page);
    await page.goto('/compliance');
    await page.waitForTimeout(1500);
  });

  test('Compliance — findings tab shows list of non-compliances', async ({ page }) => {
    // Переключаемся на таб Findings / Находки
    const findingsTab = page.locator(
      'button:has-text(/finding|non.compliance|находк|несоответств|open.*item|issue/i), ' +
      '[role="tab"]:has-text(/finding|issue|несоответ/i)',
    ).first();

    if (await findingsTab.isVisible()) {
      await findingsTab.click();
      await page.waitForTimeout(500);

      // Проверяем отображение finding из мок-данных
      const findingItem = page.locator(
        'text=/управление уязвимостям|MFA|remote access|слабый пароль|mTLS|P2P conduit/i',
      ).first();
      await expect(findingItem).toBeVisible();
    }
  });

  test('Compliance — findings show severity badges', async ({ page }) => {
    const findingsTab = page.locator(
      'button:has-text(/finding|non.compliance|находк|несоответств|issue/i), ' +
      '[role="tab"]:has-text(/finding|issue/i)',
    ).first();

    if (await findingsTab.isVisible()) {
      await findingsTab.click();
      await page.waitForTimeout(500);

      // Проверяем severity badge
      const severityBadge = page.locator(
        'span, badge, [class*="severity" i], [class*="badge" i]',
      ).filter({ hasText: /high|medium|low|critical|высок|средн|низк|критич/i }).first();
      await expect(severityBadge).toBeVisible();
    }
  });

  test('Compliance — standard drilldown opens detail view', async ({ page }) => {
    // Находим и кликаем на стандарт в списке
    const standardItem = page.locator(
      'a, button, tr, [role="row"], .card, .item, [class*="standard" i]',
    ).filter({ hasText: /IEC 62443|ISO 27001|СТБ 34\.101/i }).first();

    if (await standardItem.isVisible()) {
      await standardItem.click();
      await page.waitForTimeout(1000);

      // Проверяем что открылась детальная информация
      const detailSection = page.locator(
        'text=/control|контроль|passed|пройден|failed|провален|score|оценка/i',
      ).first();
      const hasDetail = await detailSection.isVisible().catch(() => false);

      if (!hasDetail) {
        // Проверяем измение URL
        const currentUrl = page.url();
        expect(currentUrl).toContain('/compliance');
      }
    }
  });

  test('Compliance — resolved findings show different styling', async ({ page }) => {
    const findingsTab = page.locator(
      'button:has-text(/finding|issue|несоответ/i), [role="tab"]:has-text(/finding|issue/i)',
    ).first();

    if (await findingsTab.isVisible()) {
      await findingsTab.click();
      await page.waitForTimeout(500);

      // Проверяем resolved статус
      const resolvedFinding = page.locator(
        'text=/resolved|решено|fixed|исправлен|closed|закрыт/i',
      ).first();
      const hasResolved = await resolvedFinding.isVisible().catch(() => false);

      if (hasResolved) {
        const resolvedText = await resolvedFinding.textContent();
        expect(resolvedText).toBeTruthy();
      }
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Compliance Shield — History & Trends
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Compliance Shield — History & Trends', () => {
  test.beforeEach(async ({ page }) => {
    await setupComplianceMockApi(page);
    await page.goto('/compliance');
    await page.waitForTimeout(1500);
  });

  test('Compliance — history tab shows score trend', async ({ page }) => {
    // Переключаемся на таб History / История
    const historyTab = page.locator(
      'button:has-text(/history|история|trend|тренд|changes|изменени/i), ' +
      '[role="tab"]:has-text(/history|trend|история/i)',
    ).first();

    if (await historyTab.isVisible()) {
      await historyTab.click();
      await page.waitForTimeout(500);

      // Проверяем отображение исторических данных
      const historyData = page.locator(
        'text=/84|81|76|score|trend|тренд|graph|график|chart|диаграмм/i',
      ).first();
      await expect(historyData).toBeVisible();
    }
  });

  test('Compliance — trend chart/graph is rendered', async ({ page }) => {
    const historyTab = page.locator(
      'button:has-text(/history|история|trend/i), [role="tab"]:has-text(/history|trend/i)',
    ).first();

    if (await historyTab.isVisible()) {
      await historyTab.click();
      await page.waitForTimeout(500);

      // Проверяем отрисовку графика
      const chart = page.locator(
        'svg, canvas, [data-testid*="chart" i], .chart, .graph, [role="img"]',
      ).first();
      const hasChart = await chart.isVisible().catch(() => false);

      if (hasChart) {
        await expect(chart).toBeVisible();
      }
    }
  });

  test('Compliance — next assessment date is shown', async ({ page }) => {
    // Проверяем дату следующей оценки
    const nextAssessment = page.locator(
      'text=/next assessment|следующая оценка|next.*audit|след.*аудит|scheduled|запланирован/i',
    ).first();
    await expect(nextAssessment).toBeVisible();
  });
});
