// =============================================================================
// k6 Smoke Test — CCTV Health Monitor
// =============================================================================
// Быстрый smoke test для проверки работы API перед полноценным load test.
// Используется в CI/CD pipeline для быстрой валидации.
//
// Запуск:
//   k6 run tests/load/smoke-test.js
//
// С переменными:
//   k6 run -e BASE_URL=https://staging.example.com -e AUTH_TOKEN=xxx tests/load/smoke-test.js
// =============================================================================

import { check, sleep } from 'k6';
import http from 'k6/http';
import { BASE_URL, authHeaders } from './k6.config.js';

export const options = {
  vus: 2,       // Всего 2 виртуальных пользователя
  duration: '30s', // На 30 секунд
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% запросов < 1s
    http_req_failed: ['rate<0.01'],
  },
};

export default function () {
  const headers = authHeaders();

  // ── GET /health ────────────────────────────────────────────────
  const healthResp = http.get(`${BASE_URL}/api/v1/health`, {
    headers,
    tags: { endpoint: 'health' },
  });

  check(healthResp, {
    'Health endpoint доступен': (r) => r.status === 200,
  });

  // ── GET /devices ───────────────────────────────────────────────
  const devicesResp = http.get(`${BASE_URL}/api/v1/devices`, {
    headers,
    tags: { endpoint: 'devices.list' },
  });

  check(devicesResp, {
    'Devices endpoint доступен': (r) => r.status === 200,
    'Devices возвращает массив': (r) => {
      try {
        return Array.isArray(JSON.parse(r.body));
      } catch {
        return false;
      }
    },
  });

  // ── GET /work-orders ───────────────────────────────────────────
  const woResp = http.get(`${BASE_URL}/api/v1/work-orders`, {
    headers,
    tags: { endpoint: 'work-orders.list' },
  });

  check(woResp, {
    'Work Orders endpoint доступен': (r) => r.status === 200,
  });

  // ── GET /sites ─────────────────────────────────────────────────
  const sitesResp = http.get(`${BASE_URL}/api/v1/sites`, {
    headers,
    tags: { endpoint: 'sites.list' },
  });

  check(sitesResp, {
    'Sites endpoint доступен': (r) => r.status === 200,
  });

  // Небольшая задержка между итерациями
  sleep(1);
}
