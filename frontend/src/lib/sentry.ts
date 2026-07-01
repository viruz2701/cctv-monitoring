/**
 * sentry.ts — Sentry SDK интеграция для frontend (React 19)
 *
 * P1-QA.4: Frontend Error Monitoring
 *   - Инициализация Sentry с DSN из VITE_SENTRY_DSN
 *   - Performance monitoring (browserTracingIntegration)
 *   - Session Replay (replayIntegration) — maskAllText/blockAllMedia для КИИ
 *   - User context (id, email, role)
 *   - React ErrorBoundary с логированием в Sentry
 *   - Alerting: beforeSend с фильтрацией, beforeSendTransaction c bottleneck detection
 *   - Graceful degradation при отсутствии DSN
 *   - Соответствие: OWASP ASVS L3, Приказ ОАЦ №66 п.7.18.3
 *
 * @see https://docs.sentry.io/platforms/javascript/guides/react/
 */

import {
  init,
  captureException,
  captureMessage,
  setUser,
  setTag,
  setTags,
  setExtra,
  addBreadcrumb,
  withScope,
  browserTracingIntegration,
  replayIntegration,
  browserProfilingIntegration,
  type ErrorEvent,
  type EventHint,
} from '@sentry/react';

// TransactionEvent доступен из @sentry/core (транзитивная зависимость)
import type { TransactionEvent } from '@sentry/core';

import {
  Component,
  createElement,
  type ErrorInfo,
  type ReactNode,
} from 'react';

// ── Constants ────────────────────────────────────────────────────────────────

/** Максимальная длительность API-запроса (ms) для детекции узких мест */
const API_BOTTLENECK_THRESHOLD_MS = 5_000;

/** Максимальное время рендера компонента (ms) для детекции узких мест */
const RENDER_BOTTLENECK_THRESHOLD_MS = 1_000;

/** Игнорируемые ошибки (не отправляются в Sentry) */
const IGNORED_ERROR_PATTERNS = [
  /^NetworkError$/i,
  /^AbortError$/i,
  /^ChunkLoadError$/i,
  /^Loading chunk .* failed/i,
  /^ResizeObserver/i,
  /^Non-Error promise rejection captured/i,
  /^Error: \[requestError\]/i,
];

/** P2-MED-19: Rate limiter — макс. 10 ошибок в минуту, затем 1 в минуту */
const RATE_LIMIT_WINDOW_MS = 60_000;
const RATE_LIMIT_BURST = 10;
const RATE_LIMIT_SUSTAINED = 1;

let rateLimitState: { count: number; windowStart: number } = { count: 0, windowStart: Date.now() };

function checkRateLimit(): boolean {
  const now = Date.now();
  if (now - rateLimitState.windowStart > RATE_LIMIT_WINDOW_MS) {
    rateLimitState = { count: 0, windowStart: now };
  }
  rateLimitState.count++;
  if (rateLimitState.count <= RATE_LIMIT_BURST) return true;
  // After burst: allow only 1 per window
  const sustainedLimit = RATE_LIMIT_SUSTAINED;
  return rateLimitState.count <= RATE_LIMIT_BURST + sustainedLimit;
}

/** Теги окружения для алертинга */
const ALERT_TAGS: Record<string, string> = {
  kii: 'class-2',
  compliance: 'owasp-asvs-l3',
} as const;

// ── Типы ────────────────────────────────────────────────────────────────────

export interface SentryUser {
  id: string;
  email: string;
  role: string;
}

export interface SentryErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
  /** Дополнительный контекст, добавляемый к каждой ошибке в этом boundary */
  context?: Record<string, unknown>;
}

export interface SentryErrorBoundaryState {
  hasError: boolean;
}

export interface AlertingConfig {
  /** Порог длительности API-запроса для детекции узких мест (ms) */
  apiBottleneckThresholdMs?: number;
  /** Порог времени рендера для детекции узких мест (ms) */
  renderBottleneckThresholdMs?: number;
  /** Дополнительные паттерны ошибок для игнорирования */
  ignoredErrorPatterns?: RegExp[];
}

// ── Состояние ───────────────────────────────────────────────────────────────

let initialized = false;

// ── Вспомогательные функции ─────────────────────────────────────────────────

/**
 * Проверяет, нужно ли игнорировать ошибку.
 */
function shouldIgnoreError(event: ErrorEvent): boolean {
  const errorMessage = event.message ?? '';
  const exceptionValue = event.exception?.values?.[0]?.value ?? '';
  const combined = `${errorMessage} ${exceptionValue}`;

  return IGNORED_ERROR_PATTERNS.some((pattern) => pattern.test(combined));
}

/**
 * Детекция узких мест производительности на транзакциях.
 * Добавляет теги и extra-данные при обнаружении медленных спанов.
 */
function detectTransactionBottlenecks(
  event: TransactionEvent,
  thresholds: { api: number; render: number },
): TransactionEvent {
  if (!event.spans || event.spans.length === 0) {
    return event;
  }

  const slowSpans = event.spans.filter((span) => {
    const start = span.start_timestamp ?? 0;
    const end = span.timestamp ?? 0;
    const durationMs = (end - start) * 1000;

    if (!span.op) return false;

    if (span.op.startsWith('http') || span.op === 'xhr' || span.op === 'fetch') {
      return durationMs > thresholds.api;
    }
    if (span.op.startsWith('ui.react')) {
      return durationMs > thresholds.render;
    }

    return false;
  });

  if (slowSpans.length > 0) {
    event.tags = {
      ...event.tags,
      bottleneck_detected: 'true',
      bottleneck_count: String(slowSpans.length),
    };

    event.extra = {
      ...event.extra,
      slow_spans: slowSpans.map((s) => ({
        op: s.op,
        description: s.description,
        duration_ms: ((s.timestamp ?? 0) - (s.start_timestamp ?? 0)) * 1000,
      })),
    };
  }

  return event;
}

/**
 * Stripping sensitive data из событий перед отправкой.
 * Соответствие: OWASP ASVS V5.3.3, Приказ ОАЦ №66 п.7.18.3
 */
function stripSensitiveData(event: ErrorEvent | TransactionEvent): ErrorEvent | TransactionEvent {
  if (event.request?.headers) {
    const sanitized: Record<string, string> = {};
    for (const [key, value] of Object.entries(event.request.headers)) {
      if (!/^(authorization|cookie|set-cookie|x-api-key|x-csrf-token)$/i.test(key)) {
        sanitized[key] = String(value);
      }
    }
    event.request.headers = sanitized;
  }

  if (event.request?.data && typeof event.request.data === 'object') {
    const sensitiveFields = [
      'password', 'token', 'secret', 'apiKey', 'api_key',
      'accessToken', 'refreshToken', 'csrf', 'creditCard', 'ssn',
    ];
    const sanitize = (obj: Record<string, unknown>): Record<string, unknown> => {
      const result: Record<string, unknown> = {};
      for (const [key, value] of Object.entries(obj)) {
        if (sensitiveFields.some((f) => key.toLowerCase().includes(f.toLowerCase()))) {
          result[key] = '[REDACTED]';
        } else if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
          result[key] = sanitize(value as Record<string, unknown>);
        } else {
          result[key] = value;
        }
      }
      return result;
    };
    event.request.data = sanitize(event.request.data as Record<string, unknown>);
  }

  return event;
}

// ── Инициализация ───────────────────────────────────────────────────────────

/**
 * Инициализирует Sentry SDK с полной конфигурацией.
 *
 * @param dsn - DSN из VITE_SENTRY_DSN. Если не передан или пуст — инициализация не происходит.
 * @param options - Дополнительные опции для Sentry.init() (без dsn)
 * @param alerting - Конфигурация алертинга и детекции узких мест
 *
 * @example
 * ```ts
 * import { initSentry } from '@/lib/sentry';
 *
 * initSentry(import.meta.env.VITE_SENTRY_DSN, {
 *   environment: import.meta.env.MODE,
 *   tracesSampleRate: import.meta.env.PROD ? 0.2 : 0.0,
 * });
 * ```
 */
export function initSentry(
  dsn: string | undefined,
  options?: Omit<Partial<Parameters<typeof init>[0]>, 'dsn'>,
  alerting?: AlertingConfig,
): void {
  if (initialized) {
    return;
  }

  // Graceful degradation: если DSN не задан — не инициализируем
  if (!dsn || dsn.trim() === '') {
    if (import.meta.env.DEV) {
      console.warn(
        '[Sentry] DSN не задан (VITE_SENTRY_DSN). Sentry не будет инициализирован.',
      );
    }
    return;
  }

  const apiThreshold = alerting?.apiBottleneckThresholdMs ?? API_BOTTLENECK_THRESHOLD_MS;
  const renderThreshold = alerting?.renderBottleneckThresholdMs ?? RENDER_BOTTLENECK_THRESHOLD_MS;

  init({
    dsn,
    environment: import.meta.env.MODE || 'production',
    // Не отправляем ошибки в dev-режиме (но инициализируем для тестирования)
    enabled: import.meta.env.PROD || import.meta.env.VITE_SENTRY_DEV === 'true',
    // ── Performance Monitoring ──
    // Трейсинг: 20% в production, 0% в dev
    tracesSampleRate: options?.tracesSampleRate ?? (import.meta.env.PROD ? 0.2 : 0.0),
    // Профилирование: 10% от трейсов
    profilesSampleRate: import.meta.env.PROD ? 0.1 : 0.0,
    // ── Session Replay ──
    // 10% сессий в production, 100% при ошибке
    replaysSessionSampleRate: import.meta.env.PROD ? 0.1 : 0.0,
    replaysOnErrorSampleRate: import.meta.env.PROD ? 1.0 : 0.0,
    // ── Integrations ──
    integrations: [
      browserTracingIntegration({
        instrumentNavigation: true,
        instrumentPageLoad: true,
        enableHTTPTimings: true,
        // Игнорируем спаны ресурсов, которые не нужны для мониторинга
        ignoreResourceSpans: [],
      }),
      replayIntegration({
        // Mask all text content for security (Приказ ОАЦ №66 п.7.18.3)
        maskAllText: true,
        // Block sensitive media elements
        blockAllMedia: true,
      }),
      // Profiling только в production
      ...(import.meta.env.PROD ? [browserProfilingIntegration()] : []),
    ],
    // ── Alerting: beforeSend (только для ошибок) ──
    beforeSend(event: ErrorEvent, _hint: EventHint): ErrorEvent | null {
      // 1. Rate limiting (P2-MED-19: предотвращение DSN abuse)
      if (!checkRateLimit()) {
        if (import.meta.env.DEV) {
          console.warn('[Sentry] Rate limit exceeded, dropping error event');
        }
        return null;
      }

      // 2. Игнорируем известные ошибки
      if (shouldIgnoreError(event)) {
        return null;
      }

      // 3. Добавляем compliance-теги
      event.tags = {
        ...event.tags,
        ...ALERT_TAGS,
        sentry_zone: 'frontend-zone-1',
      };

      // 3. Добавляем уровень критичности для алертинга
      if (event.exception) {
        const isUnhandled = event.exception.values?.some(
          (v) => v.mechanism?.handled === false,
        );
        event.tags = {
          ...event.tags,
          error_severity: isUnhandled ? 'critical' : 'warning',
          unhandled: isUnhandled ? 'true' : 'false',
        };
      }

      // 4. Stripping sensitive data (OWASP ASVS V5.3.3)
      event = stripSensitiveData(event) as ErrorEvent;

      return event;
    },
    // ── Alerting: beforeSendTransaction (только для транзакций производительности) ──
    beforeSendTransaction(event: TransactionEvent, _hint: EventHint): TransactionEvent | null {
      // Добавляем compliance-теги
      event.tags = {
        ...event.tags,
        ...ALERT_TAGS,
        sentry_zone: 'frontend-zone-1',
      };

      // Детекция узких мест
      event = detectTransactionBottlenecks(event, {
        api: apiThreshold,
        render: renderThreshold,
      });

      return event;
    },
    // ── Alerting: beforeSendBreadcrumb ──
    beforeBreadcrumb(breadcrumb) {
      // Фильтруем чувствительные URL из breadcrumbs
      if (
        breadcrumb.category === 'fetch' ||
        breadcrumb.category === 'xhr'
      ) {
        const url = breadcrumb.data?.url as string | undefined;
        if (url && /\/api\/v1\/auth\//.test(url)) {
          breadcrumb.data = {
            ...breadcrumb.data,
            url: url.replace(/\/api\/v1\/auth\/(login|logout|refresh)/, '/api/v1/auth/[REDACTED]'),
          };
        }
      }
      return breadcrumb;
    },
    // ── Дополнительные опции (кроме dsn) ──
    ...options,
  });

  // Устанавливаем compliance-теги уровня приложения
  setTags(ALERT_TAGS);
  setTag('framework', 'react-19');

  initialized = true;

  if (import.meta.env.DEV) {
    console.info('[Sentry] SDK инициализирован', {
      environment: import.meta.env.MODE,
      tracesSampleRate: options?.tracesSampleRate ?? (import.meta.env.PROD ? 0.2 : 0.0),
    });
  }
}

// ── User Context ────────────────────────────────────────────────────────────

/**
 * Устанавливает контекст пользователя для Sentry.
 * Вызывать после успешной аутентификации / при смене пользователя.
 *
 * @param user - Объект пользователя { id, email, role } или null для сброса
 *
 * @example
 * ```ts
 * import { setSentryUser } from '@/lib/sentry';
 *
 * setSentryUser({ id: user.id, email: user.email, role: user.role });
 * ```
 */
export function setSentryUser(user: SentryUser | null): void {
  if (!initialized) {
    return;
  }

  if (user === null) {
    setUser(null);
    return;
  }

  setUser({
    id: user.id,
    email: user.email,
    role: user.role,
  });
}

// ── Capture Error ───────────────────────────────────────────────────────────

/**
 * Отправляет ошибку в Sentry с дополнительным контекстом.
 *
 * @param error - Объект ошибки (Error, string, или unknown)
 * @param context - Дополнительный контекст (ключ-значение)
 *
 * @example
 * ```ts
 * import { captureError } from '@/lib/sentry';
 *
 * try {
 *   await riskyOperation();
 * } catch (err) {
 *   captureError(err, { operation: 'riskyOperation', deviceId });
 * }
 * ```
 */
export function captureError(
  error: unknown,
  context?: Record<string, unknown>,
): void {
  if (!initialized) {
    if (import.meta.env.DEV) {
      console.error('[Sentry] Не инициализирован. Ошибка не отправлена:', error);
    }
    return;
  }

  if (context && Object.keys(context).length > 0) {
    withScope((scope) => {
      scope.setContext('additional', context);
      captureException(error);
    });
  } else {
    captureException(error);
  }
}

// ── Capture Message ─────────────────────────────────────────────────────────

/**
 * Отправляет сообщение в Sentry (для пользовательских событий).
 *
 * @param message - Текст сообщения
 * @param level - Уровень: 'info' | 'warning' | 'error'
 * @param context - Дополнительный контекст
 *
 * @example
 * ```ts
 * import { captureSentryMessage } from '@/lib/sentry';
 *
 * captureSentryMessage('User performed bulk action', 'info', { action: 'export', count: 50 });
 * ```
 */
export function captureSentryMessage(
  message: string,
  level: 'info' | 'warning' | 'error' = 'info',
  context?: Record<string, unknown>,
): void {
  if (!initialized) {
    return;
  }

  withScope((scope) => {
    scope.setLevel(level);
    if (context && Object.keys(context).length > 0) {
      scope.setContext('additional', context);
    }
    captureMessage(message);
  });
}

// ── Performance Monitoring ──────────────────────────────────────────────────

/**
 * Добавляет тег производительности к текущей транзакции.
 * Используется для ручного мониторинга узких мест.
 *
 * @param key - Ключ тега
 * @param value - Значение тега
 *
 * @example
 * ```ts
 * import { setPerfTag } from '@/lib/sentry';
 *
 * setPerfTag('api_duration_ms', '3200');
 * ```
 */
export function setPerfTag(key: string, value: string): void {
  if (!initialized) return;
  setTag(`perf.${key}`, value);
}

/**
 * Добавляет breadcrumb (след) в Sentry для отладки пользовательского пути.
 *
 * @param category - Категория breadcrumb
 * @param message - Сообщение
 * @param data - Дополнительные данные
 *
 * @example
 * ```ts
 * import { addSentryBreadcrumb } from '@/lib/sentry';
 *
 * addSentryBreadcrumb('navigation', 'User navigated to dashboard', { from: '/login' });
 * ```
 */
export function addSentryBreadcrumb(
  category: string,
  message: string,
  data?: Record<string, unknown>,
): void {
  if (!initialized) return;

  addBreadcrumb({
    category,
    message,
    data,
    level: 'info',
  });
}

// ── Alerting Helpers ────────────────────────────────────────────────────────

/**
 * Репортит bottleneck (узкое место) производительности в Sentry.
 * Вызывается при обнаружении медленных операций.
 *
 * @param operation - Название операции
 * @param durationMs - Длительность в ms
 * @param thresholdMs - Порог в ms
 * @param context - Дополнительный контекст
 *
 * @example
 * ```ts
 * import { reportBottleneck } from '@/lib/sentry';
 *
 * reportBottleneck('work-orders.fetch', 6200, 5000, { page: 1, filters: 'active' });
 * ```
 */
export function reportBottleneck(
  operation: string,
  durationMs: number,
  thresholdMs: number,
  context?: Record<string, unknown>,
): void {
  if (!initialized) return;

  withScope((scope) => {
    scope.setTag('bottleneck', 'true');
    scope.setTag('bottleneck_operation', operation);
    scope.setExtra('duration_ms', durationMs);
    scope.setExtra('threshold_ms', thresholdMs);
    scope.setExtra('excess_ms', durationMs - thresholdMs);

    if (context) {
      scope.setContext('bottleneck_context', context);
    }

    captureMessage(`[Bottleneck] ${operation} took ${durationMs}ms`, 'warning');
  });
}

// ── Sentry Error Boundary ───────────────────────────────────────────────────

/**
 * React ErrorBoundary компонент, который логирует ошибки в Sentry.
 *
 * Отличается от глобального ErrorBoundary тем, что:
 * 1. Отправляет ошибку в Sentry через captureException
 * 2. Добавляет componentStack в контекст Sentry
 * 3. Поддерживает кастомный fallback UI
 * 4. Добавляет это как breadcrumb для трейсинга пользовательского пути
 *
 * @example
 * ```tsx
 * import { SentryErrorBoundary } from '@/lib/sentry';
 *
 * <SentryErrorBoundary context={{ page: 'dashboard' }}>
 *   <Dashboard />
 * </SentryErrorBoundary>
 * ```
 */
export class SentryErrorBoundary extends Component<
  SentryErrorBoundaryProps,
  SentryErrorBoundaryState
> {
  state: SentryErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): Partial<SentryErrorBoundaryState> {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    if (!initialized) {
      return;
    }

    // Добавляем breadcrumb для контекста
    addBreadcrumb({
      category: 'error',
      message: `ErrorBoundary caught: ${error.message}`,
      data: {
        componentName: this.props.context?.componentName ?? 'unknown',
      },
      level: 'error',
    });

    withScope((scope) => {
      scope.setContext('react', {
        componentStack: info.componentStack,
      });

      if (this.props.context && Object.keys(this.props.context).length > 0) {
        scope.setContext('additional', this.props.context);
      }

      // Теги для алертинга
      scope.setTag('error_boundary', 'true');
      scope.setTag('error_layer', (this.props.context?.layer as string) ?? 'unknown');

      captureException(error);
    });
  }

  handleRetry = (): void => {
    this.setState({ hasError: false });
  };

  render(): ReactNode {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return createElement(
        'div',
        {
          role: 'alert',
          className:
            'flex min-h-[200px] items-center justify-center rounded-lg border border-red-200 bg-red-50 p-6 dark:border-red-900 dark:bg-red-950/20',
        },
        createElement(
          'div',
          { className: 'text-center' },
          createElement(
            'div',
            { className: 'mb-3 flex justify-center' },
            createElement(
              'div',
              {
                className:
                  'flex h-10 w-10 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30',
              },
              createElement(
                'svg',
                {
                  className: 'h-5 w-5 text-red-600 dark:text-red-400',
                  fill: 'none',
                  viewBox: '0 0 24 24',
                  strokeWidth: 1.5,
                  stroke: 'currentColor',
                },
                createElement('path', {
                  strokeLinecap: 'round',
                  strokeLinejoin: 'round',
                  d: 'M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z',
                }),
              ),
            ),
          ),
          createElement(
            'h3',
            {
              className:
                'mb-1 text-sm font-semibold text-gray-900 dark:text-gray-100',
            },
            'Component Error',
          ),
          createElement(
            'p',
            {
              className: 'mb-4 text-sm text-gray-600 dark:text-gray-400',
            },
            'Something went wrong in this section.',
          ),
          createElement(
            'button',
            {
              onClick: this.handleRetry,
              className:
                'rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900',
            },
            'Try Again',
          ),
        ),
      );
    }

    return this.props.children;
  }
}

// ── Вспомогательные функции ────────────────────────────────────────────────

/**
 * Проверяет, инициализирован ли Sentry SDK.
 * Может использоваться для условного логирования.
 */
export function isSentryInitialized(): boolean {
  return initialized;
}

/**
 * Сбрасывает состояние инициализации (для тестов).
 * @internal
 */
export function resetSentryState(): void {
  initialized = false;
}
