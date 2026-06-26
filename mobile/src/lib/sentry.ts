/**
 * sentry.ts — Sentry SDK интеграция для Mobile (React Native + Expo 52)
 *
 * Обеспечивает:
 * - Инициализацию Sentry с DSN из EXPO_PUBLIC_SENTRY_DSN
 * - Capture unhandled exceptions и React component errors
 * - User context (id, email, role)
 * - React ErrorBoundary с логированием в Sentry
 * - Graceful degradation при отсутствии DSN
 *
 * @see https://docs.sentry.io/platforms/react-native/
 */

import {
  init,
  captureException,
  setUser,
  withScope,
  type ReactNativeOptions,
} from '@sentry/react-native';
import {
  Component,
  createElement,
  type ErrorInfo,
  type ReactNode,
} from 'react';
import { Text, View, TouchableOpacity } from 'react-native';

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
 * Инициализирует Sentry SDK для React Native.
 *
 * @param dsn - DSN из EXPO_PUBLIC_SENTRY_DSN. Если не передан или пуст — инициализация не происходит.
 * @param options - Дополнительные опции для Sentry.init()
 *
 * @example
 * ```ts
 * import { initSentry } from '@/lib/sentry';
 *
 * initSentry(process.env.EXPO_PUBLIC_SENTRY_DSN, {
 *   environment: process.env.NODE_ENV,
 * });
 * ```
 */
export function initSentry(
  dsn: string | undefined,
  options?: Omit<Partial<ReactNativeOptions>, 'dsn'>,
): void {
  if (initialized) {
    return;
  }

  // Graceful degradation: если DSN не задан — не инициализируем
  if (!dsn || dsn.trim() === '') {
    if (__DEV__) {
      console.warn(
        '[Sentry] DSN не задан (EXPO_PUBLIC_SENTRY_DSN). Sentry не будет инициализирован.',
      );
    }
    return;
  }

  init({
    dsn,
    environment: options?.environment || (__DEV__ ? 'development' : 'production'),
    // В dev-режиме не отправляем, но инициализируем (для отладки интеграции)
    enabled: !__DEV__,
    // Трейсинг (опционально)
    tracesSampleRate: options?.tracesSampleRate ?? 0.2,
    ...options,
  });

  initialized = true;

  if (__DEV__) {
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
    if (__DEV__) {
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

const styles = {
  container: {
    flex: 1,
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
    backgroundColor: '#FEF2F2',
    padding: 24,
  },
  iconContainer: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#FEE2E2',
    justifyContent: 'center' as const,
    alignItems: 'center' as const,
    marginBottom: 12,
  },
  iconText: {
    fontSize: 24,
    color: '#DC2626',
  },
  title: {
    fontSize: 16,
    fontWeight: '600' as const,
    color: '#111827',
    marginBottom: 4,
  },
  message: {
    fontSize: 14,
    color: '#6B7280',
    textAlign: 'center' as const,
    marginBottom: 16,
  },
  button: {
    backgroundColor: '#2563EB',
    paddingHorizontal: 16,
    paddingVertical: 10,
    borderRadius: 8,
  },
  buttonText: {
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: '500' as const,
  },
};

/**
 * React Native ErrorBoundary компонент, который логирует ошибки в Sentry.
 *
 * @example
 * ```tsx
 * import { SentryErrorBoundary } from '@/lib/sentry';
 *
 * <SentryErrorBoundary context={{ screen: 'Dashboard' }}>
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
        View,
        { style: styles.container },
        createElement(
          View,
          { style: styles.iconContainer },
          createElement(Text, { style: styles.iconText }, '!'),
        ),
        createElement(Text, { style: styles.title }, 'Component Error'),
        createElement(
          Text,
          { style: styles.message },
          'Something went wrong in this section.',
        ),
        createElement(
          TouchableOpacity,
          { style: styles.button, onPress: this.handleRetry },
          createElement(Text, { style: styles.buttonText }, 'Try Again'),
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
