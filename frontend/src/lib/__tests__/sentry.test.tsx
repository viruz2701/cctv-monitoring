// ═══════════════════════════════════════════════════════════════════════════
// sentry.test.tsx — Unit tests для sentry.ts
//
// P1-QA.4: Frontend Error Monitoring
//   - Инициализация Sentry SDK
//   - Graceful degradation при отсутствии DSN
//   - User context capture
//   - Error capture с контекстом
//   - Bottleneck reporting
//   - SentryErrorBoundary рендеринг
// ═══════════════════════════════════════════════════════════════════════════

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';

// ── Mocks Sentry ────────────────────────────────────────────────────────────

const mockInit = vi.fn();
const mockCaptureException = vi.fn();
const mockCaptureMessage = vi.fn();
const mockSetUser = vi.fn();
const mockSetTag = vi.fn();
const mockSetTags = vi.fn();
const mockSetExtra = vi.fn();
const mockAddBreadcrumb = vi.fn();
const mockWithScope = vi.fn(
  (callback: (scope: Record<string, unknown>) => void) => {
    const scope = {
      setContext: vi.fn(),
      setTag: vi.fn(),
      setExtra: vi.fn(),
      setLevel: vi.fn(),
    };
    callback(scope);
    return scope;
  },
);

vi.mock('@sentry/react', async () => {
  const actual = await vi.importActual('@sentry/react');
  return {
    ...(actual as Record<string, unknown>),
    init: mockInit,
    captureException: mockCaptureException,
    captureMessage: mockCaptureMessage,
    setUser: mockSetUser,
    setTag: mockSetTag,
    setTags: mockSetTags,
    setExtra: mockSetExtra,
    addBreadcrumb: mockAddBreadcrumb,
    withScope: mockWithScope,
    browserTracingIntegration: vi.fn(() => ({ name: 'BrowserTracing' })),
    replayIntegration: vi.fn(() => ({ name: 'Replay' })),
    browserProfilingIntegration: vi.fn(() => ({ name: 'BrowserProfiling' })),
  };
});

// ── Helpers ─────────────────────────────────────────────────────────────────

/**
 * Заменяет Vite import.meta.env.PROD/DEV через хук vitest
 * (boolean, т.к. Vite определяет PROD/DEV как boolean).
 */
function setProdEnv(isProd: boolean): void {
  vi.stubEnv('PROD', isProd);
  vi.stubEnv('DEV', !isProd);
  vi.stubEnv('MODE', isProd ? 'production' : 'development');
}

// ── Тесты ───────────────────────────────────────────────────────────────────

describe('initSentry', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
  });

  it('должен инициализировать Sentry с DSN в production', async () => {
    setProdEnv(true);

    const { initSentry, resetSentryState } = await import('../sentry');
    resetSentryState();

    initSentry('https://test@dsn.ingest.sentry.io/12345', {
      environment: 'production',
      tracesSampleRate: 0.2,
    });

    expect(mockInit).toHaveBeenCalledTimes(1);
    const initCall = mockInit.mock.calls[0][0];
    expect(initCall.dsn).toBe('https://test@dsn.ingest.sentry.io/12345');
    expect(initCall.environment).toBe('production');
    expect(initCall.tracesSampleRate).toBe(0.2);
    expect(initCall.replaysSessionSampleRate).toBe(0.1);
    expect(initCall.replaysOnErrorSampleRate).toBe(1.0);
    expect(initCall.integrations).toBeDefined();
    expect(initCall.beforeSend).toBeDefined();
    expect(initCall.beforeSendTransaction).toBeDefined();
    expect(mockSetTags).toHaveBeenCalledWith({
      kii: 'class-2',
      compliance: 'owasp-asvs-l3',
    });
  });

  it('не должен инициализировать Sentry без DSN', async () => {
    setProdEnv(false);

    const { initSentry, resetSentryState } = await import('../sentry');
    resetSentryState();

    initSentry(undefined);

    expect(mockInit).not.toHaveBeenCalled();
  });

  it('не должен инициализировать Sentry с пустым DSN', async () => {
    setProdEnv(false);

    const { initSentry, resetSentryState } = await import('../sentry');
    resetSentryState();

    initSentry('');

    expect(mockInit).not.toHaveBeenCalled();
  });

  it('должен предотвращать повторную инициализацию', async () => {
    setProdEnv(true);

    const { initSentry, resetSentryState } = await import('../sentry');
    resetSentryState();

    initSentry('https://test@dsn.ingest.sentry.io/12345');
    initSentry('https://another@dsn.ingest.sentry.io/67890');

    expect(mockInit).toHaveBeenCalledTimes(1);
  });

  it('должен передавать алертинг-конфигурацию в beforeSendTransaction', async () => {
    setProdEnv(true);

    const { initSentry, resetSentryState } = await import('../sentry');
    resetSentryState();

    initSentry('https://test@dsn.ingest.sentry.io/12345', undefined, {
      apiBottleneckThresholdMs: 3000,
    });

    expect(mockInit).toHaveBeenCalledTimes(1);
    const { beforeSendTransaction } = mockInit.mock.calls[0][0];
    expect(beforeSendTransaction).toBeDefined();

    // Транзакция с медленным спаном
    const mockTx = {
      type: 'transaction' as const,
      spans: [
        {
          op: 'http.client',
          description: 'GET /api/devices',
          start_timestamp: 0,
          timestamp: 4.0, // 4000ms — выше порога 3000ms
        },
      ],
      tags: {} as Record<string, string>,
      extra: {} as Record<string, unknown>,
    };

    const result = beforeSendTransaction(mockTx, {});
    expect(result).not.toBeNull();
    expect(result!.tags).toHaveProperty('bottleneck_detected', 'true');
    expect(result!.tags).toHaveProperty('bottleneck_count', '1');
  });
});

describe('setSentryUser', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    setProdEnv(true);

    const { resetSentryState, initSentry } = await import('../sentry');
    resetSentryState();
    initSentry('https://test@dsn.ingest.sentry.io/12345');
  });

  it('должен устанавливать контекст пользователя', async () => {
    const { setSentryUser } = await import('../sentry');

    setSentryUser({ id: 'user-1', email: 'test@example.com', role: 'admin' });

    expect(mockSetUser).toHaveBeenCalledWith({
      id: 'user-1',
      email: 'test@example.com',
      role: 'admin',
    });
  });

  it('должен сбрасывать контекст пользователя при null', async () => {
    const { setSentryUser } = await import('../sentry');

    setSentryUser(null);

    expect(mockSetUser).toHaveBeenCalledWith(null);
  });
});

describe('captureError', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    setProdEnv(true);

    const { resetSentryState, initSentry } = await import('../sentry');
    resetSentryState();
    initSentry('https://test@dsn.ingest.sentry.io/12345');
  });

  it('должен отправлять ошибку без контекста', async () => {
    const { captureError } = await import('../sentry');

    const error = new Error('Test error');
    captureError(error);

    expect(mockCaptureException).toHaveBeenCalledWith(error);
    expect(mockWithScope).not.toHaveBeenCalled();
  });

  it('должен отправлять ошибку с контекстом', async () => {
    const { captureError } = await import('../sentry');

    const error = new Error('Test error with context');
    captureError(error, { operation: 'test', deviceId: 'cam-1' });

    expect(mockWithScope).toHaveBeenCalledTimes(1);
    expect(mockCaptureException).toHaveBeenCalledWith(error);
  });

  it('не должен падать при отсутствии инициализации', async () => {
    const { resetSentryState, captureError } = await import('../sentry');
    resetSentryState();

    expect(() => {
      captureError(new Error('Should not throw'));
    }).not.toThrow();
  });
});

describe('captureSentryMessage', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    setProdEnv(true);

    const { resetSentryState, initSentry } = await import('../sentry');
    resetSentryState();
    initSentry('https://test@dsn.ingest.sentry.io/12345');
  });

  it('должен отправлять сообщение с уровнем info', async () => {
    const { captureSentryMessage } = await import('../sentry');

    captureSentryMessage('User logged in', 'info', { userId: 'user-1' });

    expect(mockWithScope).toHaveBeenCalledTimes(1);
    expect(mockCaptureMessage).toHaveBeenCalledWith('User logged in');
  });

  it('должен отправлять сообщение без контекста', async () => {
    const { captureSentryMessage } = await import('../sentry');

    captureSentryMessage('Simple message');

    expect(mockCaptureMessage).toHaveBeenCalledWith('Simple message');
  });
});

describe('reportBottleneck', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    setProdEnv(true);

    const { resetSentryState, initSentry } = await import('../sentry');
    resetSentryState();
    initSentry('https://test@dsn.ingest.sentry.io/12345');
  });

  it('должен отправлять bottleneck-сообщение', async () => {
    const { reportBottleneck } = await import('../sentry');

    reportBottleneck('api.fetch-devices', 6200, 5000, { page: 1 });

    expect(mockWithScope).toHaveBeenCalledTimes(1);
    expect(mockCaptureMessage).toHaveBeenCalledWith(
      '[Bottleneck] api.fetch-devices took 6200ms',
      'warning',
    );
  });
});

describe('SentryErrorBoundary', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    setProdEnv(true);

    const { resetSentryState, initSentry } = await import('../sentry');
    resetSentryState();
    initSentry('https://test@dsn.ingest.sentry.io/12345');
  });

  it('должен рендерить children при отсутствии ошибки', async () => {
    const { SentryErrorBoundary } = await import('../sentry');

    render(
      <SentryErrorBoundary>
        <div>Test content</div>
      </SentryErrorBoundary>,
    );

    expect(screen.getByText('Test content')).toBeDefined();
  });

  it('должен показывать fallback при ошибке и кнопку Try Again', async () => {
    const { SentryErrorBoundary } = await import('../sentry');

    const ThrowError = () => {
      throw new Error('Test crash');
    };

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <SentryErrorBoundary>
        <ThrowError />
      </SentryErrorBoundary>,
    );

    expect(screen.getByText('Component Error')).toBeDefined();
    expect(
      screen.getByText('Something went wrong in this section.'),
    ).toBeDefined();
    expect(screen.getByText('Try Again')).toBeDefined();
    expect(mockCaptureException).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it('должен использовать кастомный fallback', async () => {
    const { SentryErrorBoundary } = await import('../sentry');

    const ThrowError = () => {
      throw new Error('Test crash with custom fallback');
    };

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <SentryErrorBoundary fallback={<div>Custom error UI</div>}>
        <ThrowError />
      </SentryErrorBoundary>,
    );

    expect(screen.getByText('Custom error UI')).toBeDefined();
    expect(screen.queryByText('Component Error')).toBeNull();

    consoleSpy.mockRestore();
  });

  it('должен восстанавливаться после нажатия Try Again', async () => {
    const { SentryErrorBoundary } = await import('../sentry');

    let shouldThrow = true;
    const ConditionalError = () => {
      if (shouldThrow) {
        throw new Error('Conditional crash');
      }
      return <div>Recovered content</div>;
    };

    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    render(
      <SentryErrorBoundary>
        <ConditionalError />
      </SentryErrorBoundary>,
    );

    expect(screen.getByText('Component Error')).toBeDefined();

    shouldThrow = false;
    fireEvent.click(screen.getByText('Try Again'));

    expect(screen.getByText('Recovered content')).toBeDefined();

    consoleSpy.mockRestore();
  });
});

describe('isSentryInitialized / resetSentryState', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('isSentryInitialized должен возвращать false после сброса', async () => {
    const { resetSentryState, isSentryInitialized } = await import('../sentry');
    resetSentryState();

    expect(isSentryInitialized()).toBe(false);
  });

  it('isSentryInitialized должен возвращать true после инициализации', async () => {
    setProdEnv(true);

    const { resetSentryState, initSentry, isSentryInitialized } =
      await import('../sentry');
    resetSentryState();

    initSentry('https://test@dsn.ingest.sentry.io/12345');

    expect(isSentryInitialized()).toBe(true);
  });
});
