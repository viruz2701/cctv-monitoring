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

import { check, sleep, group } from 'k6';
import http from 'k6/http';
import { randomItem } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';
import { BASE_URL, authHeaders, publicHeaders, safeParseJSON } from './k6.config.js';

export const options = {
  vus: 2,        // Всего 2 виртуальных пользователя
  duration: '30s', // На 30 секунд
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% запросов < 1s
    http_req_failed: ['rate<0.01'],
    checks: ['rate>0.99'],
  },
};

export default function () {
  const headers = authHeaders();
  const pubHeaders = publicHeaders();

  group('Health & Monitoring', function () {
    // ── GET /health/live ─────────────────────────────────────────
    const liveResp = http.get(`${BASE_URL}/health/live`, {
      headers: pubHeaders,
      tags: { endpoint: 'health.live' },
      timeout: '3s',
    });

    check(liveResp, {
      'Health liveness endpoint доступен': (r) => r.status === 200,
    });

    // ── GET /health/ready ─────────────────────────────────────────
    const readyResp = http.get(`${BASE_URL}/health/ready`, {
      headers: pubHeaders,
      tags: { endpoint: 'health.ready' },
      timeout: '3s',
    });

    check(readyResp, {
      'Health readiness endpoint доступен': (r) => r.status === 200,
    });
  });

  group('Auth & Users', function () {
    // ── GET /users/me ───────────────────────────────────────────
    if (headers['Authorization']) {
      const meResp = http.get(`${BASE_URL}/api/v1/users/me`, {
        headers,
        tags: { endpoint: 'users.me' },
        timeout: '5s',
      });

      check(meResp, {
        'Current user endpoint доступен': (r) => r.status === 200,
        'Current user возвращает email': (r) => {
          const parsed = safeParseJSON(r.body);
          return parsed && typeof parsed.email === 'string';
        },
      });
    }
  });

  group('Devices', function () {
    // ── GET /devices ─────────────────────────────────────────────
    const devicesResp = http.get(`${BASE_URL}/api/v1/devices?page=1&page_size=10`, {
      headers,
      tags: { endpoint: 'devices.list' },
      timeout: '5s',
    });

    check(devicesResp, {
      'Devices endpoint доступен': (r) => r.status === 200,
      'Devices возвращает массив': (r) => {
        const parsed = safeParseJSON(r.body);
        const data = parsed?.data || parsed;
        return Array.isArray(data);
      },
      'Devices Content-Type JSON': (r) =>
        r.headers['Content-Type']?.includes('application/json') ?? false,
    });

    // ── GET /devices/:id (если есть данные) ──────────────────────
    if (devicesResp.status === 200) {
      const parsed = safeParseJSON(devicesResp.body);
      const data = parsed?.data || parsed;
      if (Array.isArray(data) && data.length > 0) {
        const deviceId = data[0].id;
        const detailResp = http.get(`${BASE_URL}/api/v1/devices/${deviceId}`, {
          headers,
          tags: { endpoint: 'devices.detail' },
          timeout: '5s',
        });

        check(detailResp, {
          'Device detail endpoint доступен': (r) => r.status === 200,
        });
      }
    }
  });

  group('Work Orders', function () {
    // ── GET /work-orders ─────────────────────────────────────────
    const woResp = http.get(`${BASE_URL}/api/v1/work-orders?limit=10`, {
      headers,
      tags: { endpoint: 'work-orders.list' },
      timeout: '5s',
    });

    check(woResp, {
      'Work Orders endpoint доступен': (r) => r.status === 200,
      'Work Orders Content-Type JSON': (r) =>
        r.headers['Content-Type']?.includes('application/json') ?? false,
    });
  });

  group('Sites & Analytics', function () {
    // ── GET /sites ───────────────────────────────────────────────
    const sitesResp = http.get(`${BASE_URL}/api/v1/sites`, {
      headers,
      tags: { endpoint: 'sites.list' },
      timeout: '5s',
    });

    check(sitesResp, {
      'Sites endpoint доступен': (r) => r.status === 200,
    });

    // ── GET /alarms ──────────────────────────────────────────────
    const alarmsResp = http.get(`${BASE_URL}/api/v1/alarms?limit=5`, {
      headers,
      tags: { endpoint: 'alarms.list' },
      timeout: '5s',
    });

    check(alarmsResp, {
      'Alarms endpoint доступен': (r) => r.status === 200,
    });
  });

  group('Compliance & Audit', function () {
    // ── GET /compliance/summary ──────────────────────────────────
    const complianceResp = http.get(`${BASE_URL}/api/v1/compliance/summary`, {
      headers,
      tags: { endpoint: 'compliance.summary' },
      timeout: '5s',
    });

    check(complianceResp, {
      'Compliance summary endpoint доступен': (r) => r.status === 200,
    });

    // ── GET /audit/log ───────────────────────────────────────────
    const auditResp = http.get(`${BASE_URL}/api/v1/audit/log?limit=10`, {
      headers,
      tags: { endpoint: 'audit.log' },
      timeout: '5s',
    });

    check(auditResp, {
      'Audit log endpoint доступен': (r) => r.status === 200,
    });
  });

  group('Notifications & Tickets', function () {
    // ── GET /notifications ───────────────────────────────────────
    const notifResp = http.get(`${BASE_URL}/api/v1/notifications?limit=5`, {
      headers,
      tags: { endpoint: 'notifications.list' },
      timeout: '5s',
    });

    check(notifResp, {
      'Notifications endpoint доступен': (r) => r.status === 200,
    });

    // ── GET /tickets ─────────────────────────────────────────────
    const ticketsResp = http.get(`${BASE_URL}/api/v1/tickets?limit=5`, {
      headers,
      tags: { endpoint: 'tickets.list' },
      timeout: '5s',
    });

    check(ticketsResp, {
      'Tickets endpoint доступен': (r) => r.status === 200,
    });
  });

  // Небольшая задержка между итерациями
  sleep(1);
}
