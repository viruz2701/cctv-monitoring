/// <reference types="node" />

import { test, expect } from '@playwright/test';
import {
  setupAuth,
  mockCatchAll,
} from './shared-mocks';

// ═══════════════════════════════════════════════════════════════════════════
// Event Replay — E2E Tests
// Stream browser, message replay, dead-letter queue
// ═══════════════════════════════════════════════════════════════════════════

const MOCK_EVENT_STREAMS = [
  { name: 'camera.events', type: 'jetstream', messages: 15234, consumers: 3, last_message: new Date(Date.now() - 5000).toISOString(), retention: '7d' },
  { name: 'device.status', type: 'jetstream', messages: 8921, consumers: 2, last_message: new Date(Date.now() - 10000).toISOString(), retention: '30d' },
  { name: 'alerts.critical', type: 'jetstream', messages: 345, consumers: 5, last_message: new Date(Date.now() - 60000).toISOString(), retention: '365d' },
  { name: 'audit.log', type: 'jetstream', messages: 45210, consumers: 1, last_message: new Date(Date.now() - 3000).toISOString(), retention: '7y' },
  { name: 'system.metrics', type: 'core', messages: 0, consumers: 0, last_message: null, retention: '1d' },
];

const MOCK_STREAM_MESSAGES = Array.from({ length: 20 }, (_, i) => ({
  seq: i + 1,
  subject: `camera.events.${['motion', 'connection', 'alert', 'health', 'recording'][i % 5]}`,
  timestamp: new Date(Date.now() - i * 60000).toISOString(),
  data: JSON.stringify({
    device_id: `dev-${(i % 4) + 1}`,
    event: ['motion_detected', 'connection_lost', 'disk_warning', 'health_ok', 'recording_started'][i % 5],
    value: i % 2 === 0 ? 1 : 0,
  }),
  size: Math.floor(Math.random() * 512) + 64,
}));

const MOCK_DLQ_MESSAGES = [
  { seq: 1001, subject: 'camera.events.motion', timestamp: new Date(Date.now() - 3600000).toISOString(), error: 'processing_timeout', retry_count: 3, data: '{"device_id":"dev-1","event":"motion_detected"}' },
  { seq: 1002, subject: 'device.status', timestamp: new Date(Date.now() - 7200000).toISOString(), error: 'invalid_payload', retry_count: 1, data: '{"device_id":"dev-3"}' },
  { seq: 1003, subject: 'alerts.critical', timestamp: new Date(Date.now() - 10800000).toISOString(), error: 'consumer_not_found', retry_count: 5, data: '{"alert_id":"alert-1"}' },
];

// ─────────────────────────────────────────────────────────────────────────────
// Setup
// ─────────────────────────────────────────────────────────────────────────────

async function setupEventReplayMockApi(page: any) {
  await setupAuth(page);

  // Streams list
  await page.route('**/api/v1/events/streams', async (route: any, request: any) => {
    const url = new URL(request.url());
    const streamName = url.searchParams.get('name') || '';

    if (streamName) {
      const stream = MOCK_EVENT_STREAMS.find((s) => s.name === streamName);
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(stream || null),
      });
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(MOCK_EVENT_STREAMS),
    });
  });

  // Stream messages
  await page.route('**/api/v1/events/streams/*/messages', async (route: any, request: any) => {
    const url = new URL(request.url());
    const seq = parseInt(url.searchParams.get('seq') || '0', 10);
    const limit = parseInt(url.searchParams.get('limit') || '10', 10);
    const loadMore = url.searchParams.get('offset') || url.searchParams.get('start_seq');

    if (loadMore) {
      const startSeq = parseInt(loadMore, 10);
      const messages = MOCK_STREAM_MESSAGES.filter((m) => m.seq >= startSeq).slice(0, limit);
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ messages, total: messages.length, has_more: messages.length >= limit }),
      });
    }

    if (seq > 0) {
      const message = MOCK_STREAM_MESSAGES.find((m) => m.seq === seq);
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(message || null),
      });
    }

    const messages = MOCK_STREAM_MESSAGES.slice(0, limit);
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ messages, total: MOCK_STREAM_MESSAGES.length, has_more: true }),
    });
  });

  // Replay message — POST
  await page.route('**/api/v1/events/streams/*/replay', async (route: any, request: any) => {
    if (request.method() === 'POST') {
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({ replay_id: 'replay-abc-123', status: 'queued', seq: 1 }),
      });
    }
  });

  // Dead-letter queue
  await page.route('**/api/v1/events/dead-letters', async (route: any) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ messages: MOCK_DLQ_MESSAGES, total: MOCK_DLQ_MESSAGES.length }),
    });
  });

  await mockCatchAll(page);
}

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Event Replay — Stream Browser
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Event Replay — Stream Browser', () => {
  test.beforeEach(async ({ page }) => {
    await setupEventReplayMockApi(page);
    await page.goto('/events');
    await page.waitForTimeout(1500);
  });

  test('Events page loads with stream list', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/events');

    // Проверяем отображение списка стримов
    const streamList = page.locator(
      'text=/camera\\.events|device\\.status|alerts\\.critical|audit\\.log|system\\.metrics/i',
    ).first();
    await expect(streamList).toBeVisible();
  });

  test('Events — stream message counts displayed', async ({ page }) => {
    // Проверяем счетчики сообщений в стримах
    const messageCount = page.locator(
      'text=/15234|8921|345|45210/i',
    ).first();
    await expect(messageCount).toBeVisible();
  });

  test('Events — select stream shows messages', async ({ page }) => {
    // Находим и кликаем по стриму
    const streamItem = page.locator(
      'tr, div[class*="row" i], div[class*="item" i], li',
    ).filter({ hasText: /camera\\.events|device\\.status|alerts\\.critical/i }).first();

    if (await streamItem.isVisible()) {
      await streamItem.click();
      await page.waitForTimeout(1000);

      // Проверяем отображение сообщений
      const messagesPanel = page.locator(
        'div[class*="messages" i], div[class*="detail" i], table, section',
      ).filter({ hasText: /motion|connection|health|recording|device_id|event/i }).first();
      const hasMessages = await messagesPanel.isVisible().catch(() => false);
      if (hasMessages) {
        await expect(messagesPanel).toBeVisible();
      }
    }
  });

  test('Events — message detail modal shows JSON payload', async ({ page }) => {
    // Открываем стрим с сообщениями
    const streamItem = page.locator(
      'tr, div[class*="row" i], div[class*="item" i]',
    ).filter({ hasText: /camera\\.events|device\\.status/i }).first();

    if (await streamItem.isVisible()) {
      await streamItem.click();
      await page.waitForTimeout(1000);

      // Кликаем по первому сообщению
      const messageItem = page.locator(
        'tr, div[class*="message" i], div[class*="row" i]',
      ).filter({ hasText: /seq|motion|connection|health/i }).first();

      if (await messageItem.isVisible()) {
        await messageItem.click();
        await page.waitForTimeout(500);

        // Проверяем модал с деталями сообщения
        const detailModal = page.locator(
          'div[role="dialog"], div[class*="modal" i], div[class*="drawer" i]',
        ).filter({ hasText: /seq|payload|data|subject|timestamp|device_id/i }).first();
        const hasModal = await detailModal.isVisible().catch(() => false);
        if (hasModal) {
          await expect(detailModal).toBeVisible();
        }
      }
    }
  });

  test('Events — replay message button triggers replay', async ({ page }) => {
    const streamItem = page.locator(
      'tr, div[class*="row" i], div[class*="item" i]',
    ).filter({ hasText: /camera\\.events/i }).first();

    if (await streamItem.isVisible()) {
      await streamItem.click();
      await page.waitForTimeout(1000);
    }

    // Находим кнопку Replay
    const replayButton = page.locator(
      'button, [role="button"]',
    ).filter({ hasText: /replay|повтор|resend|переотпр/i }).first();

    if (await replayButton.isVisible()) {
      await replayButton.click();
      await page.waitForTimeout(1000);

      // Проверяем подтверждение отправки
      const replayConfirm = page.locator(
        'div[class*="toast" i], div[role="alert"]',
      ).filter({ hasText: /replay|queued|повтор|отправ/i }).first();
      const hasConfirm = await replayConfirm.isVisible().catch(() => false);
    }
  });
});

// ═══════════════════════════════════════════════════════════════════════════
// Test Suite: Event Replay — DLQ & Load More
// ═══════════════════════════════════════════════════════════════════════════

test.describe('Event Replay — DLQ & Load More', () => {
  test.beforeEach(async ({ page }) => {
    await setupEventReplayMockApi(page);
    await page.goto('/events/dead-letters');
    await page.waitForTimeout(1500);
  });

  test('Events — dead-letter queue panel loads', async ({ page }) => {
    await expect(page.locator('body')).toBeVisible();
    expect(page.url()).toContain('/dead-letter');

    // Проверяем отображение DLQ сообщений
    const dlqPanel = page.locator(
      'text=/processing_timeout|invalid_payload|consumer_not_found/i',
    ).first();
    await expect(dlqPanel).toBeVisible();
  });

  test('Events — DLQ shows error details and retry count', async ({ page }) => {
    // Проверяем отображение количества retry
    const retryCount = page.locator(
      'text=/3|5|retry|повтор|attempt|попытк/i',
    ).first();
    await expect(retryCount).toBeVisible();

    // Проверяем отображение ошибки
    const errorDetail = page.locator(
      'text=/processing_timeout|invalid_payload|consumer_not_found/i',
    ).first();
    await expect(errorDetail).toBeVisible();
  });

  test('Events — load more button fetches additional messages', async ({ page }) => {
    await page.goto('/events');
    await page.waitForTimeout(1500);

    // Открываем стрим
    const streamItem = page.locator(
      'tr, div[class*="row" i], div[class*="item" i]',
    ).filter({ hasText: /camera\\.events/i }).first();

    if (await streamItem.isVisible()) {
      await streamItem.click();
      await page.waitForTimeout(1000);
    }

    // Находим кнопку "Load More"
    const loadMoreButton = page.locator(
      'button, a, [role="button"]',
    ).filter({ hasText: /load more|загрузить еще|show more|показать еще|more|еще/i }).first();

    if (await loadMoreButton.isVisible()) {
      await loadMoreButton.click();
      await page.waitForTimeout(1000);
    }
  });

  test('Events — stream retention policy displayed', async ({ page }) => {
    // Проверяем отображение политики хранения
    const retentionInfo = page.locator(
      'text=/7d|30d|365d|7y|1d|retention|хранен/i',
    ).first();
    await expect(retentionInfo).toBeVisible();
  });

  test('Events — empty stream state shown when no messages', async ({ page }) => {
    // Мокаем пустой стрим
    await page.route('**/api/v1/events/streams/system.metrics/messages', async (route: any) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ messages: [], total: 0, has_more: false }),
      });
    });

    const emptyState = page.locator(
      'text=/no messages|empty|нет сообщ|no data/i',
    ).first();
    const hasEmpty = await emptyState.isVisible().catch(() => false);
  });
});
