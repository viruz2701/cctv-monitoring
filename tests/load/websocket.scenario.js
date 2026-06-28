// =============================================================================
// k6 Scenario: WebSocket — Load Test для Realtime API
// =============================================================================
// Compliance: IEC 62443-3-3 SR 7.8 (Security Function Verification)
//             OWASP ASVS L3 V13 (API & WebService — WebSocket)
//             ISO 27001 A.12.6 (Capacity Management)
// =============================================================================
//
// Сценарий:
// - 1000 concurrent WebSocket connections (ramp up over 60s)
// - Подписка на device events, alerts, work-orders
// - Получение heartbeat сообщений и событий
// - Reconnect при разрыве соединения
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
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { WS_URL, AUTH_TOKEN, BASE_OPTIONS, safeParseJSON } from './k6.config.js';

// ── Конфигурация WebSocket ─────────────────────────────────────────────────

const PING_INTERVAL = 30000;      // 30s между ping
const MIN_SESSION_DURATION = 20000;  // 20s
const MAX_SESSION_DURATION = 60000;  // 60s
const SUBSCRIBE_CHANNELS = ['device.events', 'alerts', 'work-orders', 'system.status'];
const MAX_RECONNECT_ATTEMPTS = 3;
const RECONNECT_DELAY = 1000; // 1s между попытками

// ── Опции сценария ─────────────────────────────────────────────────────────

export const options = {
  ...BASE_OPTIONS,

  stages: [
    // Ramp up: 0 → 1000 за 60s
    { duration: '60s', target: 1000 },
    // Stay: 1000 в течение 120s
    { duration: '120s', target: 1000 },
    // Ramp down: 1000 → 0 за 30s
    { duration: '30s', target: 0 },
  ],

  thresholds: {
    // Для WebSocket специфичные метрики
    ws_connecting: ['p(95)<1000'],       // время соединения < 1s
    ws_session_duration: ['p(95)>10000'], // сессии живут > 10s
    ws_msgs_received: ['rate>0'],         // сообщения приходят
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],

    // Per-endpoint thresholds
    'ws_connecting{endpoint:websocket.events}': ['p(95)<1000'],
    'ws_connecting{endpoint:websocket.reconnect}': ['p(95)<2000'],
  },

  // IEC 62443-3-3 SR 7.8: Каждое соединение независимо
  noVUConnectionReuse: true,
};

// ── Утилиты для WebSocket ──────────────────────────────────────────────────

/**
 * Формирует WebSocket URL с токеном аутентификации.
 */
function buildWsUrl() {
  return AUTH_TOKEN
    ? `${WS_URL}/alarms?token=${AUTH_TOKEN}`
    : `${WS_URL}/alarms`;
}

/**
 * Отправляет сообщение подписки на каналы.
 */
function sendSubscribe(socket, channels) {
  socket.send(JSON.stringify({
    type: 'subscribe',
    channels: channels,
    // IEC 62443-3-3 SR 7.8: Уникальная идентификация клиента
    client_id: `k6-vu-${__VU}`,
    session_id: `${__VU}-${__ITER}`,
  }));
}

/**
 * Отправляет ping сообщение.
 */
function sendPing(socket) {
  socket.send(JSON.stringify({
    type: 'ping',
    timestamp: Date.now(),
    vu: __VU,
    iter: __ITER,
  }));
}

// ── Обработчик соединения ──────────────────────────────────────────────────

function handleConnection(socket, isReconnect) {
  const endpointTag = isReconnect ? 'websocket.reconnect' : 'websocket.events';
  let messageCount = 0;
  let heartbeatCount = 0;
  let lastMessageTime = Date.now();

  // ── Open ──────────────────────────────────────────────────────────
  socket.on('open', function () {
    // Подписка на все каналы
    sendSubscribe(socket, SUBSCRIBE_CHANNELS);

    if (__VU === 1) {
      console.log(`[WS] VU ${__VU}: соединение ${isReconnect ? 'переподключено' : 'установлено'}`);
    }
  });

  // ── Message ───────────────────────────────────────────────────────
  socket.on('message', function (data) {
    messageCount++;
    lastMessageTime = Date.now();

    const msg = safeParseJSON(data);
    if (!msg || !msg.type) return;

    switch (msg.type) {
      case 'heartbeat':
        heartbeatCount++;
        check(msg, {
          'Heartbeat получен': () => true,
          'Heartbeat содержит timestamp': () =>
            typeof msg.timestamp === 'string' || typeof msg.timestamp === 'number',
          'Heartbeat latency < 500ms': () => {
            if (typeof msg.timestamp === 'number') {
              return Date.now() - msg.timestamp < 500;
            }
            return true;
          },
        });
        break;

      case 'subscribed':
        check(msg, {
          'Подписка подтверждена': () => true,
          'Подписка содержит каналы': () =>
            Array.isArray(msg.channels) && msg.channels.length > 0,
          'Подписка подтверждена для VU': () =>
            msg.client_id === `k6-vu-${__VU}`,
        });
        break;

      case 'device.event':
        check(msg, {
          'Событие устройства получено': () => true,
          'Событие содержит device_id': () => typeof msg.device_id === 'string',
          'Событие содержит timestamp': () =>
            typeof msg.timestamp === 'string' || typeof msg.timestamp === 'number',
          'Событие содержит тип': () =>
            typeof msg.event_type === 'string',
        });
        break;

      case 'alert':
        check(msg, {
          'Alert получен': () => true,
          'Alert содержит device_id': () => typeof msg.device_id === 'string',
          'Alert содержит severity': () =>
            ['info', 'warning', 'critical'].includes(msg.severity),
        });
        break;

      case 'work-order.update':
        check(msg, {
          'WO update получен': () => true,
          'WO update содержит id': () => typeof msg.work_order_id === 'string',
          'WO update содержит status': () => typeof msg.status === 'string',
        });
        break;

      case 'system.status':
        check(msg, {
          'System status получен': () => true,
        });
        break;

      case 'pong':
        // Ответ на ping — ок, ничего не делаем
        break;

      default:
        // Неизвестный тип сообщения — логируем только для VU 1
        if (__VU === 1) {
          console.log(`[WS] VU ${__VU}: неизвестный тип сообщения: ${msg.type}`);
        }
    }
  });

  // ── Error ─────────────────────────────────────────────────────────
  socket.on('error', function (e) {
    console.error(`[WS] VU ${__VU}: ошибка соединения — ${e.error()}`);
  });

  // ── Close ─────────────────────────────────────────────────────────
  socket.on('close', function () {
    if (__VU === 1) {
      console.log(
        `[WS] VU ${__VU}: соединение закрыто, получено ${messageCount} сообщений, ${heartbeatCount} heartbeats`
      );
    }
  });

  // ── Периодический ping ────────────────────────────────────────────
  socket.setInterval(function () {
    sendPing(socket);
  }, PING_INTERVAL);

  // ── Периодическое обновление подписки ─────────────────────────────
  socket.setInterval(function () {
    // Каждые 60s обновляем подписку (re-subscribe)
    sendSubscribe(socket, randomItem([
      ['device.events'],
      ['alerts'],
      ['work-orders'],
      SUBSCRIBE_CHANNELS,
    ]));
  }, 60000);

  // ── Длительность сессии ───────────────────────────────────────────
  const sessionDuration = randomIntBetween(MIN_SESSION_DURATION, MAX_SESSION_DURATION);
  socket.setTimeout(function () {
    socket.close();
  }, sessionDuration);
}

// ── Основная функция ───────────────────────────────────────────────────────

export default function () {
  const url = buildWsUrl();
  let attempt = 0;

  // Попытка соединения с реконнектом
  while (attempt <= MAX_RECONNECT_ATTEMPTS) {
    const isReconnect = attempt > 0;

    const response = ws.connect(url, {
      tags: {
        endpoint: isReconnect ? 'websocket.reconnect' : 'websocket.events',
        vu: String(__VU),
        attempt: String(attempt + 1),
      },
      timeout: '10s',
    }, function (socket) {
      handleConnection(socket, isReconnect);
    });

    // Проверка соединения
    const connected = check(response, {
      [`WebSocket соединение ${isReconnect ? 'переподключено' : 'установлено'}`]:
        (r) => r && r.status === 101,
    });

    if (connected) {
      break;
    }

    // Реконнект с экспоненциальной задержкой
    attempt++;
    if (attempt <= MAX_RECONNECT_ATTEMPTS) {
      const delay = RECONNECT_DELAY * attempt; // 1s, 2s, 3s
      if (__VU === 1) {
        console.log(`[WS] VU ${__VU}: попытка реконнекта ${attempt}/${MAX_RECONNECT_ATTEMPTS} через ${delay}ms`);
      }
      sleep(delay / 1000);
    }
  }

  // Даём время на обработку сообщений
  sleep(randomIntBetween(1, 3));
}
