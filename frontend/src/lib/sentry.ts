/**
 * sentry.ts — Sentry SDK интеграция для frontend (React 19)
 *
 * Обеспечивает:
 * - Инициализацию Sentry с DSN из VITE_SENTRY_DSN
 * - Capture unhandled exceptions
 * - User context (id, email, role)
 * - React ErrorBoundary с логированием в Sentry
 * - Graceful degradation при отсутствии DSN
 *
 * @see https://docs.sentry.io/platforms/javascript/guides/react/
 */

import {
  init,
  captureException,
  setUser,
  withScope,
} from '@sentry/react';
import {
  Component,
  createElement,
  type ErrorInfo,
  type ReactNode,
} from 'react';

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

// ── Состояние ───────────────────────────────────────────────────────────────

let initialized = false;

// ── Инициализация ───────────────────────────────────────────────────────────

/**
 * Инициализирует Sentry SDK.
 *
 * @param dsn - DSN из VITE_SENTRY_DSN. Если не передан или пуст — инициализация не происходит.
 * @param options - Дополнительные опции для Sentry.init()
 *
 * @example
 * ```ts
 * import { initSentry } from '@/lib/sentry';
 *
 * initSentry(import.meta.env.VITE_SENTRY_DSN, {
 *   environment: import.meta.env.MODE,
 * });
 * ```
 */
export function initSentry(
  dsn: string | undefined,
  options?: Omit<Partial<Parameters<typeof init>[0]>, 'dsn'>,
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

  init({
    dsn,
    environment: import.meta.env.MODE || 'production',
    // Не отправляем ошибки в dev-режиме (но инициализируем для тестирования)
    enabled: import.meta.env.PROD || import.meta.env.VITE_SENTRY_DEV === 'true',
    // Уровень логирования SDK
    logLevel: import.meta.env.DEV ? 'debug' : 'error',
    ...options,
  });

  initialized = true;

  if (import.meta.env.DEV) {
    console.info('[Sentry] SDK инициализирован');
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
    withScope((scope: { setContext: (key: string, ctx: Record<string, unknown>) => void }) => {
      scope.setContext('additional', context);
      captureException(error);
    });
  } else {
    captureException(error);
  }
}

// ── Sentry Error Boundary ───────────────────────────────────────────────────

/**
 * React ErrorBoundary компонент, который логирует ошибки в Sentry.
 *
 * Отличается от глобального ErrorBoundary тем, что:
 * 1. Отправляет ошибку в Sentry через captureException
 * 2. Добавляет componentStack в контекст Sentry
 * 3. Поддерживает кастомный fallback UI
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

    withScope((scope: { setContext: (key: string, ctx: Record<string, unknown>) => void }) => {
      scope.setContext('react', {
        componentStack: info.componentStack,
      });

      if (this.props.context && Object.keys(this.props.context).length > 0) {
        scope.setContext('additional', this.props.context);
      }

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
