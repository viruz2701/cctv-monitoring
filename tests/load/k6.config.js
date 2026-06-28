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
const API_VERSION = 'v1';

// ── HTTP-таймауты ──────────────────────────────────────────────────────────
// Для разных типов запросов (чтобы не ждать вечно упавший endpoint)
export const TIMEOUTS = {
  FAST:    '500ms',   // health, liveness
  NORMAL:  '5s',      // API endpoints
  SLOW:    '10s',     // export, reports
  STREAM:  '30s',     // WebSocket, streaming
};

// ── Общие опции для всех сценариев ─────────────────────────────────────────

export const BASE_OPTIONS = {
  // Глобальные threshold'ы (P1-QA.7)
  thresholds: {
    // 95th percentile для всех запросов < 500ms
    http_req_duration: ['p(95)<500'],
    // 99th percentile < 1000ms
    http_req_duration: ['p(99)<1000'],
    // Ошибок < 1%
    http_req_failed: ['rate<0.01'],
    // Проверки проходят >99%
    checks: ['rate>0.99'],
  },

  // Тег для идентификации окружения
  tags: {
    app: 'cctv-health-monitor',
    test_suite: 'load-testing',
    compliance: 'p1-qa.5',
  },

  // Глобальные настройки
  noConnectionReuse: false,
  noVUConnectionReuse: false,
  discardResponseBodies: false,
};

// ── Вспомогательные функции ────────────────────────────────────────────────

/**
 * Возвращает HTTP-заголовки для авторизованных запросов.
 */
export function authHeaders() {
  const headers = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    'X-Client-Type': 'k6-load-test',
  };

  if (AUTH_TOKEN) {
    headers['Authorization'] = `Bearer ${AUTH_TOKEN}`;
  }

  return headers;
}

/**
 * Возвращает заголовки для анонимных запросов (health, public).
 */
export function publicHeaders() {
  return {
    'Accept': 'application/json',
  };
}

/**
 * Форматирует длительность в читаемый вид.
 */
export function formatDuration(ms) {
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

/**
 * Безопасный парсинг JSON с возвратом null при ошибке.
 */
export function safeParseJSON(body) {
  try {
    return JSON.parse(body);
  } catch {
    return null;
  }
}

/**
 * Проверяет, что JSON-тело ответа содержит поле с ожидаемым типом.
 */
export function checkFieldType(body, field, expectedType) {
  const parsed = safeParseJSON(body);
  if (!parsed) return false;
  return typeof parsed[field] === expectedType;
}

/**
 * Генерирует случайную строку указанной длины.
 */
export function randomString(length) {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < length; i++) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
}

export {
  BASE_URL,
  AUTH_TOKEN,
  WS_URL,
  API_VERSION,
};
