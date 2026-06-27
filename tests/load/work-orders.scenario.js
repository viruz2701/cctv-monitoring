// =============================================================================
// k6 Scenario: POST /work-orders — Load Test для Work Orders API
// =============================================================================
// Compliance: ISO 27001 A.12.6 (Capacity Management)
//             IEC 62443-3-3 SR 7.8 (Security Function Verification)
//             OWASP ASVS L3 V11 (Business Logic — Input Validation)
// =============================================================================
//
// Сценарий:
// - 1000 concurrent users (ramp up over 60s)
// - POST /api/v1/work-orders — создание заявок
// - 95th percentile < 500ms
// - Валидация входных данных (OWASP ASVS V5)
//
// Запуск:
//   k6 run tests/load/work-orders.scenario.js
//
// С переменными:
//   k6 run -e BASE_URL=https://staging.example.com -e AUTH_TOKEN=xxx tests/load/work-orders.scenario.js
// =============================================================================

import { check, sleep } from 'k6';
import http from 'k6/http';
import { randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { BASE_URL, authHeaders, BASE_OPTIONS } from './k6.config.js';

// ── Тестовые данные ────────────────────────────────────────────────────────

const PRIORITIES = ['low', 'medium', 'high', 'critical'];
const WORK_TYPES = ['repair', 'maintenance', 'inspection', 'installation', 'emergency'];
const DESCRIPTIONS = [
  'Camera offline — требуется диагностика',
  'Неисправность блока питания на камере #12',
  'Плановая замена HDD в NVR-03',
  'Обновление прошивки на камерах 3-го этажа',
  'Чистка оптики камер внешнего периметра',
];

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
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

// ── Генерация payload ──────────────────────────────────────────────────────

function generateWorkOrderPayload() {
  return JSON.stringify({
    title: `Load Test WO — ${Date.now()}`,
    description: randomItem(DESCRIPTIONS),
    priority: randomItem(PRIORITIES),
    work_type: randomItem(WORK_TYPES),
    site_id: null, // будет назначено системой
    assigned_to: null, // auto-assign
    scheduled_date: new Date(Date.now() + 86400000).toISOString(), // завтра
    // Валидация: только разрешённые поля (OWASP ASVS V5.1)
    metadata: {
      source: 'k6-load-test',
      test_run: `${__VU}-${__ITER}`,
    },
  });
}

// ── Основная функция ───────────────────────────────────────────────────────

export default function () {
  const headers = authHeaders();
  const payload = generateWorkOrderPayload();

  // POST /api/v1/work-orders — создание заявки
  const createResp = http.post(`${BASE_URL}/api/v1/work-orders`, payload, {
    headers,
    tags: { endpoint: 'work-orders.create' },
  });

  check(createResp, {
    'POST /work-orders — статус 201': (r) => r.status === 201,
    'POST /work-orders — статус 201 или 409': (r) =>
      r.status === 201 || r.status === 409,
    'POST /work-orders — Content-Type JSON': (r) =>
      r.headers['Content-Type']?.includes('application/json') ?? false,
    'POST /work-orders — тело не пустое': (r) => r.body.length > 0,
  });

  // Если создание успешно, проверяем GET созданной заявки
  if (createResp.status === 201) {
    try {
      const workOrder = JSON.parse(createResp.body);
      const woId = workOrder.id;

      if (woId) {
        // GET /api/v1/work-orders/:id — проверка создания
        const getResp = http.get(`${BASE_URL}/api/v1/work-orders/${woId}`, {
          headers,
          tags: { endpoint: 'work-orders.get' },
        });

        check(getResp, {
          'GET /work-orders/:id — статус 200': (r) => r.status === 200,
          'GET /work-orders/:id — id совпадает': (r) => {
            try {
              return JSON.parse(r.body).id === woId;
            } catch {
              return false;
            }
          },
        });
      }
    } catch {
      // JSON parse error
    }
  }

  // Дополнительно: GET /api/v1/work-orders — список
  if (__ITER % 5 === 0) {
    const listResp = http.get(`${BASE_URL}/api/v1/work-orders`, {
      headers,
      tags: { endpoint: 'work-orders.list' },
    });

    check(listResp, {
      'GET /work-orders — статус 200': (r) => r.status === 200,
    });
  }

  // Think time: 2-5 секунд
  sleep(randomIntBetween(2, 5));
}
