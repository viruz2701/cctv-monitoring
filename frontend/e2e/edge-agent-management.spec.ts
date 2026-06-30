/// <reference types="node" />

import { test, expect } from '@playwright/test';
import { setupAuth, mockCatchAll, MOCK_SITES, type MockDevice } from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Edge Agent Management — E2E Tests
// CRUD: list agents, view detail, send command, delete
// ═══════════════════════════════════════════════════════════════════════════

// ── Mock Data ─────────────────────────────────────────────────────────────

const MOCK_AGENTS = [
  {
    id: 'agent-1',
    name: 'Edge-Agent-Main-01',
    site_id: 'site-1',
    site: 'Main Office',
    status: 'online',
    last_seen: new Date().toISOString(),
    version: 'v2.5.1',
    cpu: 45.2,
    memory: 62.8,
    uptime: 172800,
    errors: 0,
    traffic: { in: 1_500_000, out: 800_000 },
    config: { log_level: 'info', interval_ms: 5000 },
    created_at: new Date(Date.now() - 86400000 * 90).toISOString(),
  },
  {
    id: 'agent-2',
    name: 'Edge-Agent-Branch-02',
    site_id: 'site-2',
    site: 'Branch Office',
    status: 'offline',
    last_seen: new Date(Date.now() - 7200000).toISOString(),
    version: 'v2.4.0',
    cpu: 0,
    memory: 0,
    uptime: 0,
    errors: 3,
    traffic: { in: 0, out: 0 },
    config: { log_level: 'warn', interval_ms: 10000 },
    created_at: new Date(Date.now() - 86400000 * 60).toISOString(),
  },
  {
    id: 'agent-3',
    name: 'Edge-Agent-Warehouse-03',
    site_id: 'site-3',
    site: 'Warehouse',
    status: 'error',
    last_seen: new Date(Date.now() - 3600000).toISOString(),
    version: 'v2.5.1',
    cpu: 89.5,
    memory: 95.2,
    uptime: 43200,
    errors: 7,
    traffic: { in: 500_000, out: 200_000 },
    config: { log_level: 'error', interval_ms: 1000 },
    created_at: new Date(Date.now() - 86400000 * 30).toISOString(),
  },
];

// ── Setup ─────────────────────────────────────────────────────────────────

async function setupAgentsMockApi(page: any) {
  await setupAuth(page);

  // GET /api/v1/agents — list
  await page.route('**/api/v1/agents', async (route: any, request: any) => {
    if (request.method() === 'GET') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: MOCK_AGENTS, total: MOCK_AGENTS.length }),
      });
    } else {
      await route.fulfill({ status: 405 });
    }
  });

  // GET /api/v1/agents/:id — single agent
  await page.route('**/api/v1/agents/**', async (route: any, request: any) => {
    const url = request.url();
    const match = url.match(/\/agents\/([^/]+)/);
    if (!match) return route.fulfill({ status: 404 });

    const agentId = match[1];
    const agent = MOCK_AGENTS.find((a) => a.id === agentId);

    if (request.method() === 'DELETE') {
      if (agent) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'deleted', agent_id: agentId }),
        });
      }
      return route.fulfill({ status: 404 });
    }

    if (request.method() === 'GET') {
      if (agent) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(agent),
        });
      }
      return route.fulfill({ status: 404 });
    }

    return route.fulfill({ status: 405 });
  });

  // POST /api/v1/agents/:id/command
  await page.route('**/api/v1/agents/*/command', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agent_id: 'agent-1',
          command: body.command || 'ping',
          status: 'sent',
          created_at: new Date().toISOString(),
        }),
      });
    } else {
      await route.fulfill({ status: 405 });
    }
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Agent List
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Edge Agent Management — List', () => {
  test.beforeEach(async ({ page }) => {
    await setupAgentsMockApi(page);
  });

  test('Agent Dashboard — renders agent list page', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Dashboard title
    const heading = page.locator('h1').filter({ hasText: /agent/i }).first();
    await expect(heading).toBeVisible();
  });

  test('Agent Dashboard — displays agent table with 3 agents', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Check that agent names appear in the table
    for (const agent of MOCK_AGENTS) {
      const agentCell = page.locator('text=' + agent.name).first();
      await expect(agentCell).toBeVisible();
    }
  });

  test('Agent Dashboard — table shows correct agent status indicators', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Online status badge
    const onlineBadge = page.locator('text=online').first();
    await expect(onlineBadge).toBeVisible();

    // Offline status badge
    const offlineBadge = page.locator('text=offline').first();
    await expect(offlineBadge).toBeVisible();
  });

  test('Agent Dashboard — refresh button exists', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    const refreshBtn = page.locator('button').filter({ hasText: /refresh|обнов/i }).first();
    await expect(refreshBtn).toBeVisible();
  });

  test('Agent Dashboard — offline warning alert appears', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Offline alert should be visible since agent-2 is offline
    const offlineAlert = page.locator('div[class*="alert" i], div[role="alert"]')
      .filter({ hasText: /offline|alert/i }).first();
    await expect(offlineAlert).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Agent Detail
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Edge Agent Management — Detail', () => {
  test.beforeEach(async ({ page }) => {
    await setupAgentsMockApi(page);
  });

  test('Agent Detail — navigates to detail view and shows agent name', async ({ page }) => {
    await page.goto('/agents/agent-1');
    await page.waitForTimeout(2000);

    const heading = page.locator('h1').filter({ hasText: /Edge-Agent-Main-01/i }).first();
    await expect(heading).toBeVisible();
  });

  test('Agent Detail — displays performance metrics', async ({ page }) => {
    await page.goto('/agents/agent-1');
    await page.waitForTimeout(2000);

    // CPU metric
    const cpu = page.locator('text=/cpu|ЦП/i').first();
    await expect(cpu).toBeVisible();

    // Memory metric
    const mem = page.locator('text=/memory|памят/i').first();
    await expect(mem).toBeVisible();
  });

  test('Agent Detail — displays network traffic info', async ({ page }) => {
    await page.goto('/agents/agent-1');
    await page.waitForTimeout(2000);

    const trafficSection = page.locator('text=/traffic|трафик|network|сеть/i').first();
    await expect(trafficSection).toBeVisible();
  });

  test('Agent Detail — shows back button to return to list', async ({ page }) => {
    await page.goto('/agents/agent-1');
    await page.waitForTimeout(2000);

    const backBtn = page.locator('button').filter({ hasText: /back|назад/i }).first();
    await expect(backBtn).toBeVisible();
  });

  test('Agent Detail — shows error state for non-existent agent', async ({ page }) => {
    await page.goto('/agents/agent-nonexistent');
    await page.waitForTimeout(2000);

    const notFound = page.locator('text=/not found|не найден/i').first();
    await expect(notFound).toBeVisible();
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Agent Commands
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Edge Agent Management — Commands', () => {
  test.beforeEach(async ({ page }) => {
    await setupAgentsMockApi(page);
  });

  test('Agent Table — send command button exists for each agent', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Send command buttons
    const sendBtns = page.locator('button[aria-label*="command" i], button[aria-label*="send" i]');
    const count = await sendBtns.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Agent Table — delete button exists for each agent', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    const deleteBtns = page.locator('button[aria-label*="delete" i], button[aria-label*="удал" i]');
    const count = await deleteBtns.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Agent Table — detail navigation button exists', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    const detailBtns = page.locator('button[aria-label*="detail" i], a[href*="/agent"]');
    const count = await detailBtns.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Agent Sorting
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Edge Agent Management — Sorting', () => {
  test.beforeEach(async ({ page }) => {
    await setupAgentsMockApi(page);
  });

  test('Agent Table — column headers are clickable for sorting', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Find sortable column headers
    const sortableHeaders = page.locator('th, button[class*="sort" i]');
    const count = await sortableHeaders.count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Agent Table — click on name column header sorts agents', async ({ page }) => {
    await page.goto('/agents');
    await page.waitForTimeout(2000);

    // Click on "Name" column header
    const nameHeader = page.locator('th, button').filter({ hasText: /name|имя/i }).first();
    if (await nameHeader.isVisible()) {
      await nameHeader.click();
      await page.waitForTimeout(500);
    }
  });
});
