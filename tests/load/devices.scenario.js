// =============================================================================
// k6 Scenario: GET /devices — Load Test для Device API
// =============================================================================
// Compliance: ISO 27001 A.12.6 (Capacity Management)
//             OWASP ASVS L3 V11 (Business Logic)
// =============================================================================
//
// Сценарий:
// - 1000 concurrent users (ramp up over 60s)
// - GET /api/v1/devices — список устройств
// - 95th percentile < 500ms
//
// Запуск:
//   k6 run tests/load/devices.scenario.js
//
// С переменными:
//   k6 run -e BASE_URL=https://staging.example.com -e AUTH_TOKEN=xxx tests/load/devices.scenario.js
// =============================================================================

import { check, sleep } from 'k6';
import http from 'k6/http';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { BASE_URL, authHeaders, BASE_OPTIONS } from './k6.config.js';

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
    // Специфичные для device endpoint
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

// ── Основная функция ───────────────────────────────────────────────────────

export default function () {
  const headers = authHeaders();

  // GET /api/v1/devices — список устройств
  const listResp = http.get(`${BASE_URL}/api/v1/devices`, {
    headers,
    tags: { endpoint: 'devices.list' },
  });

  check(listResp, {
    'GET /devices — статус 200': (r) => r.status === 200,
    'GET /devices — body не пустой': (r) => r.body.length > 0,
    'GET /devices — Content-Type JSON': (r) =>
      r.headers['Content-Type']?.includes('application/json') ?? false,
  });

  // Дополнительные проверки для разных сценариев
  if (listResp.status === 200) {
    try {
      const body = JSON.parse(listResp.body);

      // Если есть данные, получаем детали первого устройства
      if (Array.isArray(body) && body.length > 0) {
        const deviceId = body[0].id;
        const detailResp = http.get(`${BASE_URL}/api/v1/devices/${deviceId}`, {
          headers,
          tags: { endpoint: 'devices.detail' },
        });

        check(detailResp, {
          'GET /devices/:id — статус 200': (r) => r.status === 200,
          'GET /devices/:id — id совпадает': (r) => {
            try {
              return JSON.parse(r.body).id === deviceId;
            } catch {
              return false;
            }
          },
        });
      }
    } catch {
      // JSON parse error — ничего не делаем
    }
  }

  // Think time: 1-3 секунды между запросами (имитация пользователя)
  sleep(randomIntBetween(1, 3));
}
