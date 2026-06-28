// =============================================================================
// k6 Scenario: GET /devices — Load Test для Device API
// =============================================================================
// Compliance: ISO 27001 A.12.6 (Capacity Management)
//             OWASP ASVS L3 V11 (Business Logic)
//             IEC 62443-3-3 SR 7.8 (Security Function Verification)
// =============================================================================
//
// Сценарий:
// - 1000 concurrent users (ramp up over 60s)
// - GET /api/v1/devices — список устройств (с пагинацией)
// - GET /api/v1/devices/{id} — детальная информация
// - GET /api/v1/devices/{id}/status — статус устройства
// - Фильтрация по status, device_type, site_id
// - 95th percentile < 500ms
//
// Запуск:
//   k6 run tests/load/devices.scenario.js
//
// С переменными:
//   k6 run -e BASE_URL=https://staging.example.com -e AUTH_TOKEN=xxx tests/load/devices.scenario.js
// =============================================================================

import { check, sleep, group } from 'k6';
import http from 'k6/http';
import { randomIntBetween, randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { BASE_URL, authHeaders, BASE_OPTIONS, safeParseJSON } from './k6.config.js';

// ── Тестовые данные для фильтрации ─────────────────────────────────────────

const STATUSES = ['online', 'offline', 'warning', 'error', 'maintenance'];
const DEVICE_TYPES = ['camera', 'nvr', 'sensor', 'gateway', 'server'];
const ASSET_CLASSES = ['critical', 'high', 'medium', 'low'];
const VENDOR_TYPES = ['hikvision', 'dahua', 'axis', 'bosch', 'hanwha'];

// ── Опции сценария ─────────────────────────────────────────────────────────

export const options = {
  ...BASE_OPTIONS,

  stages: [
    // Ramp up: 0 → 1000 за 60s
    { duration: '60s', target: 1000 },
    // Stay: 1000 в течение 180s
    { duration: '180s', target: 1000 },
    // Ramp down: 1000 → 0 за 30s
    { duration: '30s', target: 0 },
  ],

  thresholds: {
    http_req_duration: [
      'p(95)<500',
      'p(99)<1000',
      'avg<200',
    ],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],

    // Специфичные для endpoint'ов
    'http_req_duration{endpoint:devices.list}':    ['p(95)<500',  'p(99)<800'],
    'http_req_duration{endpoint:devices.detail}':  ['p(95)<300',  'p(99)<600'],
    'http_req_duration{endpoint:devices.status}':  ['p(95)<200',  'p(99)<400'],
    'http_req_duration{endpoint:devices.filter}':  ['p(95)<500',  'p(99)<1000'],
    'http_req_duration{endpoint:devices.search}':  ['p(95)<500',  'p(99)<1000'],
  },

  // IEC 62443-3-3 SR 7.8: Каждый VU независим
  noVUConnectionReuse: true,
};

// ── Основная функция ───────────────────────────────────────────────────────

export default function () {
  const headers = authHeaders();

  group('GET /devices — список и фильтрация', function () {
    // ── Сценарий A: Список устройств (пагинация) ─────────────────────
    // Разные пользователи запрашивают разные страницы
    const page = randomIntBetween(1, 5);
    const pageSize = randomItem([10, 20, 50, 100]);
    const listResp = http.get(
      `${BASE_URL}/api/v1/devices?page=${page}&page_size=${pageSize}`,
      {
        headers,
        tags: { endpoint: 'devices.list' },
        timeout: '5s',
      }
    );

    check(listResp, {
      'GET /devices — статус 200': (r) => r.status === 200,
      'GET /devices — body не пустой': (r) => r.body.length > 0,
      'GET /devices — Content-Type JSON': (r) =>
        r.headers['Content-Type']?.includes('application/json') ?? false,
      'GET /devices — ответ валидный JSON': (r) => safeParseJSON(r.body) !== null,
    });

    // ── Сценарий B: Фильтрация по параметрам ─────────────────────────
    // 30% запросов — с фильтрацией
    if (__ITER % 3 === 0) {
      const filterStatus = randomItem(STATUSES);
      const filterType = randomItem(DEVICE_TYPES);
      const filterResp = http.get(
        `${BASE_URL}/api/v1/devices?status=${filterStatus}&device_type=${filterType}`,
        {
          headers,
          tags: { endpoint: 'devices.filter' },
          timeout: '5s',
        }
      );

      check(filterResp, {
        'GET /devices?filter — статус 200': (r) => r.status === 200,
        'GET /devices?filter — тело не пустое': (r) => r.body.length > 0,
      });
    }

    // ── Сценарий C: Детальная информация ─────────────────────────────
    if (listResp.status === 200) {
      const body = safeParseJSON(listResp.body);

      if (body && Array.isArray(body.data || body) && body.length > 0) {
        const devices = body.data || body;
        // Берём случайное устройство из списка (не всегда первое)
        const deviceId = randomItem(devices).id;

        if (deviceId) {
          // GET /devices/{id} — детальная информация
          const detailResp = http.get(
            `${BASE_URL}/api/v1/devices/${deviceId}`,
            {
              headers,
              tags: { endpoint: 'devices.detail' },
              timeout: '5s',
            }
          );

          check(detailResp, {
            'GET /devices/:id — статус 200': (r) => r.status === 200,
            'GET /devices/:id — id совпадает': (r) => {
              const parsed = safeParseJSON(r.body);
              return parsed && parsed.id === deviceId;
            },
            'GET /devices/:id — Content-Type JSON': (r) =>
              r.headers['Content-Type']?.includes('application/json') ?? false,
          });

          // GET /devices/{id}/status — статус устройства (30%)
          if (__ITER % 3 === 1) {
            const statusResp = http.get(
              `${BASE_URL}/api/v1/devices/${deviceId}/status`,
              {
                headers,
                tags: { endpoint: 'devices.status' },
                timeout: '3s',
              }
            );

            check(statusResp, {
              'GET /devices/:id/status — статус 200': (r) => r.status === 200,
            });
          }
        }
      }
    }

    // ── Сценарий D: Поиск по названию (10%) ──────────────────────────
    if (__ITER % 10 === 0) {
      const searchTerms = ['cam', 'nvr', 'sensor', 'gateway', 'switch'];
      const searchResp = http.get(
        `${BASE_URL}/api/v1/devices?search=${randomItem(searchTerms)}`,
        {
          headers,
          tags: { endpoint: 'devices.search' },
          timeout: '5s',
        }
      );

      check(searchResp, {
        'GET /devices?search — статус 200': (r) => r.status === 200,
      });
    }
  });

  // Think time: 1-3 секунды между запросами (имитация пользователя)
  sleep(randomIntBetween(1, 3));
}
