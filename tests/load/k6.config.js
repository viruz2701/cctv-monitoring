// =============================================================================
// k6 Load Testing Configuration — CCTV Health Monitor
// =============================================================================
// Compliance: ISO 27001 A.12.6 (Capacity Management)
//             IEC 62443-3-3 SR 7.8 (Security Function Verification)
//             OWASP ASVS L3 V11 (Business Logic — Rate Limiting)
// =============================================================================

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const AUTH_TOKEN = __ENV.AUTH_TOKEN || '';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080/ws';

// ── Общие опции для всех сценариев ─────────────────────────────────────────

export const BASE_OPTIONS = {
  // Глобальные threshold'ы
  thresholds: {
    // 95th percentile для всех запросов < 500ms (P1-QA.7)
    http_req_duration: ['p(95)<500'],
    // 0 ошибок
    http_req_failed: ['rate<0.01'],
    // Проверки проходят >99%
    checks: ['rate>0.99'],
  },

  // Тег для идентификации окружения
  tags: {
    app: 'cctv-health-monitor',
    test_suite: 'load-testing',
  },
};

// ── Вспомогательные функции ────────────────────────────────────────────────

/**
 * Возвращает HTTP-заголовки для авторизованных запросов.
 */
export function authHeaders() {
  const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };

  if (AUTH_TOKEN) {
    headers['Authorization'] = `Bearer ${AUTH_TOKEN}`;
  }

  return headers;
}

/**
 * Форматирует длительность в читаемый вид.
 */
export function formatDuration(ms) {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

export { BASE_URL, AUTH_TOKEN, WS_URL };
