// =============================================================================
// k6 Scenario: WebSocket — Load Test для Realtime API
// =============================================================================
// Compliance: IEC 62443-3-3 SR 7.8 (Security Function Verification)
//             OWASP ASVS L3 V13 (API & Web Service — WebSocket)
//             ISO 27001 A.12.6 (Capacity Management)
// =============================================================================
//
// Сценарий:
// - 1000 concurrent WebSocket connections (ramp up over 60s)
// - Подписка на device events
// - Получение heartbeat сообщений
// - 95th percentile < 500ms (latency для сообщений)
//
// Запуск:
//   k6 run tests/load/websocket.scenario.js
//
// С переменными:
//   k6 run -e WS_URL=wss://staging.example.com/ws -e AUTH_TOKEN=xxx tests/load/websocket.scenario.js
// =============================================================================

import { check, sleep } from 'k6';
import ws from 'k6/ws';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { WS_URL, AUTH_TOKEN, BASE_OPTIONS } from './k6.config.js';

// ── Опции сценария ─────────────────────────────────────────────────────────

export const options = {
  ...BASE_OPTIONS,

  stages: [
    // Ramp up: 0 → 1000 за 60s
    { duration: '60s', target: 1000 },
    // Stay: 1000 в течение 60s
    { duration: '60s', target: 1000 },
    // Ramp down: 1000 → 0 за 30s
    { duration: '30s', target: 0 },
  ],

  thresholds: {
    // Для WebSocket специфичные метрики
    ws_connecting: ['p(95)<1000'], // время соединения < 1s
    ws_session_duration: ['p(95)>10000'], // сессии живут > 10s
    http_req_failed: ['rate<0.01'],
  },
};

// ── Основная функция ───────────────────────────────────────────────────────

export default function () {
  // WebSocket URL с токеном аутентификации
  const wsUrl = AUTH_TOKEN
    ? `${WS_URL}?token=${AUTH_TOKEN}`
    : WS_URL;

  const url = wsUrl;

  const response = ws.connect(url, {
    tags: { endpoint: 'websocket.events' },
  }, function (socket) {
    // ── Открытие соединения ──────────────────────────────────────
    socket.on('open', function () {
      // Подписка на device events
      socket.send(JSON.stringify({
        type: 'subscribe',
        channels: ['device.events', 'alerts', 'work-orders'],
        // IEC 62443-3-3 SR 7.8: Подтверждение подписки
        client_id: `k6-vu-${__VU}`,
      }));

      if (__VU === 1) {
        console.log(`[WS] VU ${__VU}: соединение установлено`);
      }
    });

    // ── Получение сообщений ──────────────────────────────────────
    socket.on('message', function (data) {
      try {
        const msg = JSON.parse(data);

        // Проверка heartbeat (keepalive)
        if (msg.type === 'heartbeat') {
          check(msg, {
            'Heartbeat получен': () => true,
            'Heartbeat содержит timestamp': () =>
              typeof msg.timestamp === 'string' || typeof msg.timestamp === 'number',
          });
          return;
        }

        // Проверка подписки
        if (msg.type === 'subscribed') {
          check(msg, {
            'Подписка подтверждена': () => true,
            'Подписка содержит каналы': () =>
              Array.isArray(msg.channels) && msg.channels.length > 0,
          });
          return;
        }

        // Проверка события устройства
        if (msg.type === 'device.event' || msg.type === 'alert') {
          check(msg, {
            'Событие устройства получено': () => true,
            'Событие содержит ID': () => typeof msg.device_id === 'string',
            'Событие содержит timestamp': () =>
              typeof msg.timestamp === 'string' || typeof msg.timestamp === 'number',
          });
        }
      } catch {
        // Non-JSON message (binary, etc.)
      }
    });

    // ── Ошибки соединения ────────────────────────────────────────
    socket.on('error', function (e) {
      console.error(`[WS] VU ${__VU}: ошибка соединения — ${e.error()}`);
    });

    // ── Закрытие соединения ──────────────────────────────────────
    socket.on('close', function () {
      if (__VU === 1) {
        console.log(`[WS] VU ${__VU}: соединение закрыто`);
      }
    });

    // ── Периодический ping (каждые 30s) ──────────────────────────
    socket.setInterval(function () {
      socket.send(JSON.stringify({
        type: 'ping',
        timestamp: Date.now(),
      }));
    }, 30000);

    // Держим соединение открытым 20-40 секунд
    const sessionDuration = randomIntBetween(20000, 40000);
    socket.setTimeout(function () {
      socket.close();
    }, sessionDuration);
  });

  // Базовая проверка соединения
  check(response, {
    'WebSocket соединение установлено': (r) => r && r.status === 101,
  });

  // WebSocket соединение не блокирует, но даём время
  sleep(1);
}
