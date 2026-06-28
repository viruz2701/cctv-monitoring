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
// - POST /api/v1/work-orders — создание заявок (разные типы)
// - PATCH /api/v1/work-orders/{id} — обновление статуса (assign → start → complete)
// - GET /api/v1/work-orders — список с фильтрацией
// - 95th percentile < 500ms
// - Валидация входных данных (OWASP ASVS V5)
//
// Запуск:
//   k6 run tests/load/work-orders.scenario.js
//
// С переменными:
//   k6 run -e BASE_URL=https://staging.example.com -e AUTH_TOKEN=xxx tests/load/work-orders.scenario.js
// =============================================================================

import { check, sleep, group } from 'k6';
import http from 'k6/http';
import { randomItem, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { BASE_URL, authHeaders, BASE_OPTIONS, safeParseJSON } from './k6.config.js';

// ── Тестовые данные ────────────────────────────────────────────────────────

const PRIORITIES = ['low', 'medium', 'high', 'critical'];
const WORK_TYPES = ['repair', 'maintenance', 'inspection', 'installation', 'emergency'];
const WORK_STATUSES = [
  { action: 'assign',   endpoint: 'work-orders.assign' },
  { action: 'start',    endpoint: 'work-orders.start' },
  { action: 'complete', endpoint: 'work-orders.complete' },
  { action: 'cancel',   endpoint: 'work-orders.cancel' },
];
const SITE_IDS = Array.from({ length: 10 }, (_, i) => `site-${String(i + 1).padStart(3, '0')}`);
const DESCRIPTIONS = [
  'Camera offline — требуется диагностика',
  'Неисправность блока питания на камере #12',
  'Плановая замена HDD в NVR-03',
  'Обновление прошивки на камерах 3-го этажа',
  'Чистка оптики камер внешнего периметра',
  'Замена коммутатора в серверной',
  'Проверка заземления на опорах видеонаблюдения',
  'Настройка детекции движения на камерах периметра',
  'Замена ИК-подсветки на камере #7',
  'Восстановление кабельной трассы после ремонта',
];

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
      'avg<300',
    ],
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],

    // Per-endpoint thresholds
    'http_req_duration{endpoint:work-orders.create}':  ['p(95)<500',  'p(99)<1000'],
    'http_req_duration{endpoint:work-orders.assign}':  ['p(95)<400',  'p(99)<800'],
    'http_req_duration{endpoint:work-orders.start}':   ['p(95)<300',  'p(99)<600'],
    'http_req_duration{endpoint:work-orders.complete}': ['p(95)<300', 'p(99)<600'],
    'http_req_duration{endpoint:work-orders.cancel}':  ['p(95)<300',  'p(99)<600'],
    'http_req_duration{endpoint:work-orders.list}':    ['p(95)<500',  'p(99)<1000'],
    'http_req_duration{endpoint:work-orders.get}':     ['p(95)<300',  'p(99)<600'],
  },
};

// ── Генерация payload ──────────────────────────────────────────────────────

function generateWorkOrderPayload() {
  return JSON.stringify({
    title: `Load Test WO — ${Date.now()}-${__VU}-${__ITER}`,
    description: randomItem(DESCRIPTIONS),
    priority: randomItem(PRIORITIES),
    work_type: randomItem(WORK_TYPES),
    site_id: randomItem(SITE_IDS),
    assigned_to: null, // auto-assign
    scheduled_date: new Date(Date.now() + randomIntBetween(3600000, 604800000)).toISOString(),
    // Валидация: только разрешённые поля (OWASP ASVS V5.1)
    metadata: {
      source: 'k6-load-test',
      test_run: `${__VU}-${__ITER}`,
      scenario: 'p1-qa.5',
    },
  });
}

/**
 * Генерирует payload для обновления статуса заявки.
 */
function generateStatusUpdatePayload() {
  const now = new Date();
  return JSON.stringify({
    notes: `Status update via k6 load test — ${now.toISOString()}`,
    completed_at: now.toISOString(),
    resolution: 'completed',
    metadata: {
      source: 'k6-load-test',
      test_run: `${__VU}-${__ITER}`,
    },
  });
}

// ── Вспомогательная: создание заявки ───────────────────────────────────────

function createWorkOrder(headers) {
  const payload = generateWorkOrderPayload();
  const createResp = http.post(`${BASE_URL}/api/v1/work-orders`, payload, {
    headers,
    tags: { endpoint: 'work-orders.create' },
    timeout: '5s',
  });

  const checks = check(createResp, {
    'POST /work-orders — статус 201': (r) => r.status === 201,
    'POST /work-orders — статус 201 или 409': (r) =>
      r.status === 201 || r.status === 409,
    'POST /work-orders — Content-Type JSON': (r) =>
      r.headers['Content-Type']?.includes('application/json') ?? false,
    'POST /work-orders — тело не пустое': (r) => r.body.length > 0,
    'POST /work-orders — валидный JSON': (r) => safeParseJSON(r.body) !== null,
  });

  if (createResp.status === 201) {
    return safeParseJSON(createResp.body);
  }
  return null;
}

// ── Основная функция ───────────────────────────────────────────────────────

export default function () {
  const headers = authHeaders();

  group('POST /work-orders — создание', function () {
    const workOrder = createWorkOrder(headers);

    if (!workOrder || !workOrder.id) {
      // Если не удалось создать — проверяем список заявок
      const listResp = http.get(`${BASE_URL}/api/v1/work-orders?limit=10`, {
        headers,
        tags: { endpoint: 'work-orders.list' },
        timeout: '5s',
      });

      check(listResp, {
        'GET /work-orders — статус 200 (fallback)': (r) => r.status === 200,
      });

      sleep(randomIntBetween(2, 5));
      return;
    }

    // ── Сценарий A: Проверка создания ────────────────────────────────
    const woId = workOrder.id;

    // GET созданной заявки для верификации
    const getResp = http.get(`${BASE_URL}/api/v1/work-orders/${woId}`, {
      headers,
      tags: { endpoint: 'work-orders.get' },
      timeout: '5s',
    });

    check(getResp, {
      'GET /work-orders/:id — статус 200': (r) => r.status === 200,
      'GET /work-orders/:id — id совпадает': (r) => {
        const parsed = safeParseJSON(r.body);
        return parsed && parsed.id === woId;
      },
    });

    // ── Сценарий B: Статус-транзишны (50%) ──────────────────────────
    // assign → start → complete (или cancel)
    if (__ITER % 2 === 0) {
      // 1. ASSIGN
      const assignResp = http.post(
        `${BASE_URL}/api/v1/work-orders/${woId}/assign`,
        generateStatusUpdatePayload(),
        { headers, tags: { endpoint: 'work-orders.assign' }, timeout: '5s' }
      );

      check(assignResp, {
        'POST /work-orders/:id/assign — статус 200': (r) => r.status === 200,
      });

      if (assignResp.status === 200) {
        // 2. START
        const startResp = http.post(
          `${BASE_URL}/api/v1/work-orders/${woId}/start`,
          generateStatusUpdatePayload(),
          { headers, tags: { endpoint: 'work-orders.start' }, timeout: '5s' }
        );

        check(startResp, {
          'POST /work-orders/:id/start — статус 200': (r) => r.status === 200,
        });

        if (startResp.status === 200) {
          // 3. COMPLETE или CANCEL (80% complete, 20% cancel)
          if (__ITER % 5 !== 0) {
            const completeResp = http.post(
              `${BASE_URL}/api/v1/work-orders/${woId}/complete`,
              generateStatusUpdatePayload(),
              { headers, tags: { endpoint: 'work-orders.complete' }, timeout: '5s' }
            );

            check(completeResp, {
              'POST /work-orders/:id/complete — статус 200': (r) => r.status === 200,
            });
          } else {
            const cancelResp = http.post(
              `${BASE_URL}/api/v1/work-orders/${woId}/cancel`,
              generateStatusUpdatePayload(),
              { headers, tags: { endpoint: 'work-orders.cancel' }, timeout: '5s' }
            );

            check(cancelResp, {
              'POST /work-orders/:id/cancel — статус 200': (r) => r.status === 200,
            });
          }
        }
      }
    }

    // ── Сценарий C: Список заявок с фильтрацией (каждый 5-й) ─────────
    if (__ITER % 5 === 0) {
      const filters = [
        `?status=open&limit=20`,
        `?priority=${randomItem(PRIORITIES)}&limit=20`,
        `?type=${randomItem(WORK_TYPES)}&limit=20`,
        `?assigned_to=${randomItem(['tech-001', 'tech-002', 'tech-003'])}&limit=20`,
      ];

      const listResp = http.get(
        `${BASE_URL}/api/v1/work-orders${randomItem(filters)}`,
        {
          headers,
          tags: { endpoint: 'work-orders.list' },
          timeout: '5s',
        }
      );

      check(listResp, {
        'GET /work-orders — статус 200': (r) => r.status === 200,
        'GET /work-orders — body не пустое': (r) => r.body.length > 0,
      });
    }
  });

  // Think time: 2-5 секунд
  sleep(randomIntBetween(2, 5));
}
