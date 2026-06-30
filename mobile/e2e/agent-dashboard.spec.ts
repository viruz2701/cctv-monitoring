// ──────────────────────────────────────────────────
// E2E: Agent Dashboard (Mobile)
//
// Проверяет:
//   - Список edge-агентов
//   - Статус агентов (online/offline/error)
//   - Детальная информация об агенте
//   - Отправка команд агенту
//   - Pull-to-refresh
// ──────────────────────────────────────────────────

import { waitForElement, waitForAnimation } from './helpers/testUtils';

// ── Constants ────────────────────────────────────

const API_BASE = 'http://localhost:3000';

// ── Mock Data ────────────────────────────────────

const mockAgents = [
  {
    id: 'agent-1',
    name: 'Edge-Agent-Main-01',
    site: 'Facility A',
    status: 'online',
    version: 'v2.5.1',
    cpu: 45.2,
    memory: 62.8,
    uptime: 172800,
    errors: 0,
    traffic: { in: 1_500_000, out: 800_000 },
    last_seen: new Date().toISOString(),
  },
  {
    id: 'agent-2',
    name: 'Edge-Agent-Branch-02',
    site: 'Facility B',
    status: 'offline',
    version: 'v2.4.0',
    cpu: 0,
    memory: 0,
    uptime: 0,
    errors: 3,
    traffic: { in: 0, out: 0 },
    last_seen: new Date(Date.now() - 7200000).toISOString(),
  },
  {
    id: 'agent-3',
    name: 'Edge-Agent-Warehouse-03',
    site: 'Warehouse',
    status: 'error',
    version: 'v2.5.1',
    cpu: 89.5,
    memory: 95.2,
    uptime: 43200,
    errors: 7,
    traffic: { in: 500_000, out: 200_000 },
    last_seen: new Date(Date.now() - 3600000).toISOString(),
  },
];

const mockAgentDetail = {
  ...mockAgents[0],
  config: { log_level: 'info', interval_ms: 5000, mtu: 1500 },
  network: { ip: '192.168.1.10', mac: 'AA:BB:CC:11:22:33', gateway: '192.168.1.1' },
  storage: { total_gb: 64, used_gb: 23, available_gb: 41 },
};

// ── Helpers ──────────────────────────────────────

async function setupMockRoutes(): Promise<void> {
  // Auth
  await device.mockRoute(
    { url: `${API_BASE}/mobile/auth/login`, method: 'POST' },
    {
      status: 200,
      body: { token: 'mock-token', refresh_token: 'mock-refresh', user: { id: 'tech-1', username: 'johntech', role: 'technician' } },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Agent list
  await device.mockRoute(
    { url: `${API_BASE}/mobile/agents`, method: 'GET' },
    { status: 200, body: { agents: mockAgents, total: mockAgents.length }, headers: { 'Content-Type': 'application/json' } },
  );

  // Agent detail
  await device.mockRoute(
    { url: `${API_BASE}/mobile/agents/agent-1`, method: 'GET' },
    { status: 200, body: mockAgentDetail, headers: { 'Content-Type': 'application/json' } },
  );

  // Agent command
  await device.mockRoute(
    { url: `${API_BASE}/mobile/agents/agent-1/command`, method: 'POST' },
    {
      status: 200,
      body: { agent_id: 'agent-1', command: 'ping', status: 'sent', created_at: new Date().toISOString() },
      headers: { 'Content-Type': 'application/json' },
    },
  );

  // Device map
  await device.mockRoute(
    { url: `${API_BASE}/mobile/devices/map`, method: 'GET' },
    { status: 200, body: { devices: [] }, headers: { 'Content-Type': 'application/json' } },
  );

  // Profile
  await device.mockRoute(
    { url: `${API_BASE}/mobile/technician/profile`, method: 'GET' },
    { status: 200, body: { user_id: 'tech-1', user_name: 'John Technician' }, headers: { 'Content-Type': 'application/json' } },
  );
}

async function teardownMockRoutes(): Promise<void> {
  await device.unmockRoute({ url: `${API_BASE}/mobile/auth/login` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/agents` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/agents/agent-1` });
  await device.unmockRoute({ url: `${API_BASE}/mobile/agents/agent-1/command` });
}

// ── Init ─────────────────────────────────────────

beforeAll(async () => {
  await device.launchApp({
    newInstance: true,
    permissions: { location: 'always' },
  });
  await setupMockRoutes();
});

afterAll(async () => {
  await teardownMockRoutes();
});

beforeEach(async () => {
  await device.reloadReactNative();
  await waitForAnimation(2000);
});

// ── Tests ────────────────────────────────────────

describe('Agent Dashboard', () => {
  // ── 1. Agent list displays all agents ──

  it('Должен отобразить список всех edge-агентов', async () => {
    await waitForAnimation(3000);

    // All agents visible
    await expect(element(by.text('Edge-Agent-Main-01'))).toBeVisible();
    await expect(element(by.text('Edge-Agent-Branch-02'))).toBeVisible();
    await expect(element(by.text('Edge-Agent-Warehouse-03'))).toBeVisible();
  });

  // ── 2. Agent online indicator visible ──

  it('Должен показать статус online для активного агента', async () => {
    await waitForAnimation(3000);

    // Online indicator
    await expect(element(by.id('agent-status-online'))).toBeVisible();
  });

  // ── 3. Agent offline indicator visible ──

  it('Должен показать статус offline для неактивного агента', async () => {
    await waitForAnimation(3000);

    // Offline indicator
    await expect(element(by.id('agent-status-offline'))).toBeVisible();
  });

  // ── 4. Agent error indicator visible ──

  it('Должен показать статус error для проблемного агента', async () => {
    await waitForAnimation(3000);

    // Error indicator
    await expect(element(by.id('agent-status-error'))).toBeVisible();
  });

  // ── 5. Agent detail shows metrics ──

  it('Должен отобразить детальную информацию об агенте после нажатия', async () => {
    await waitForAnimation(3000);

    // Tap on first agent
    await element(by.text('Edge-Agent-Main-01')).tap();
    await waitForAnimation(2000);

    // CPU metric
    await expect(element(by.id('agent-cpu-metric'))).toBeVisible();
    await expect(element(by.id('agent-cpu-value'))).toBeVisible();

    // Memory metric
    await expect(element(by.id('agent-memory-metric'))).toBeVisible();
    await expect(element(by.id('agent-memory-value'))).toBeVisible();

    // Uptime
    await expect(element(by.id('agent-uptime'))).toBeVisible();

    // Last seen timestamp
    await expect(element(by.id('agent-last-seen'))).toBeVisible();
  });

  // ── 6. Send command to agent ──

  it('Должен отправить команду агенту и получить подтверждение', async () => {
    await waitForAnimation(3000);

    // Tap on first agent
    await element(by.text('Edge-Agent-Main-01')).tap();
    await waitForAnimation(2000);

    // Send command button
    await expect(element(by.id('agent-send-command-btn'))).toBeVisible();

    // Tap send command
    await element(by.id('agent-send-command-btn')).tap();
    await waitForAnimation(1000);

    // Command modal visible
    await expect(element(by.id('command-modal'))).toBeVisible();

    // Enter command
    await element(by.id('command-input')).tap();
    await element(by.id('command-input')).typeText('ping');
    await element(by.id('command-send-btn')).tap();
    await waitForAnimation(2000);

    // Command sent confirmation
    await expect(element(by.id('command-sent-confirmation'))).toBeVisible();
  });

  // ── 7. Agent network traffic display ──

  it('Должен отобразить информацию о сетевом трафике агента', async () => {
    await waitForAnimation(3000);

    // Tap on first agent
    await element(by.text('Edge-Agent-Main-01')).tap();
    await waitForAnimation(2000);

    // Traffic in
    await expect(element(by.id('agent-traffic-in'))).toBeVisible();
    await expect(element(by.id('agent-traffic-in-value'))).toBeVisible();

    // Traffic out
    await expect(element(by.id('agent-traffic-out'))).toBeVisible();
    await expect(element(by.id('agent-traffic-out-value'))).toBeVisible();
  });

  // ── 8. Pull to refresh ──

  it('Должен обновить список агентов через Pull-to-Refresh', async () => {
    await waitForAnimation(3000);

    // Pull to refresh on agent list
    await element(by.id('agent-list-scrollview')).swipe('down', 'fast', 0.5);
    await waitForAnimation(2000);

    // Agents still visible after refresh
    await expect(element(by.text('Edge-Agent-Main-01'))).toBeVisible();
  });

  // ── 9. Back navigation ──

  it('Должен вернуться к списку агентов через Back navigation', async () => {
    await waitForAnimation(3000);

    // Navigate to detail
    await element(by.text('Edge-Agent-Main-01')).tap();
    await waitForAnimation(2000);

    // Back button
    await element(by.id('agent-back-btn')).tap();
    await waitForAnimation(1000);

    // Back to agent list
    await expect(element(by.id('agent-list-scrollview'))).toBeVisible();
    await expect(element(by.text('Edge-Agent-Main-01'))).toBeVisible();
  });
});
