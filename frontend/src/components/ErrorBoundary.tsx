import { Component, type ErrorInfo, type ReactNode } from 'react';

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  traceId: string;
}

// Генерирует trace_id для связи ошибки с бэкендом
function generateTraceId(): string {
  const arr = new Uint8Array(16);
  crypto.getRandomValues(arr);
  return Array.from(arr)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

// Отправляет ошибку на backend POST /api/v1/errors
// БЕЗ sensitive data (ISO 27001 A.12.4.1, OWASP ASVS V7.1)
async function reportError(error: Error, traceId: string): Promise<void> {
  try {
    // Извлекаем только безопасную информацию
    const safeError = {
      trace_id: traceId,
      message: error.message, // только сообщение, не stacktrace с путями
      name: error.name,
      url: window.location.href, // для контекста
      timestamp: new Date().toISOString(),
    };

    await fetch('/api/v1/errors', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Request-ID': traceId,
      },
      body: JSON.stringify(safeError),
      // Не ждём ответа — fire and forget
      signal: AbortSignal.timeout(5000),
    });
  } catch {
    // Silent fail — не создаём циклических ошибок
  }
}

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false, error: null, traceId: '' };

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    const traceId = generateTraceId();

    // Логируем без sensitive data
    console.error('[ErrorBoundary] Uncaught error:', {
      trace_id: traceId,
      name: error.name,
      message: error.message,
      // НЕ логируем componentStack в production — он содержит пути
      componentStack: import.meta.env.DEV ? info.componentStack : undefined,
    });

    // Отправляем на backend (ISO 27001 A.12.4.1)
    reportError(error, traceId);

    this.setState({ traceId });
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, traceId: '' });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      const { traceId } = this.state;

      return (
        <div className="flex min-h-screen items-center justify-center bg-gray-50 p-6 dark:bg-gray-950">
          <div className="max-w-md rounded-xl border border-red-200 bg-white p-8 shadow-lg dark:border-red-900 dark:bg-gray-900">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
                <svg className="h-6 w-6 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" strokeWidth={1.5} stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z" />
                </svg>
              </div>
              <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
                Something went wrong
              </h2>
            </div>
            <p className="mb-2 text-sm text-gray-600 dark:text-gray-400">
              An unexpected error occurred. Please try refreshing the page.
            </p>

            {/* Показываем trace_id только в DEV (P2-MED-20) */}
            {import.meta.env.DEV && traceId && (
              <p className="mb-4 text-xs text-gray-400 dark:text-gray-500 font-mono">
                Error ID: <span className="text-gray-500 dark:text-gray-400">{traceId}</span>
              </p>
            )}

            {/* Только в DEV показываем детали ошибки (без sensitive data) */}
            {import.meta.env.DEV && this.state.error && (
              <pre className="mb-4 max-h-48 overflow-auto rounded-lg bg-gray-100 p-3 text-xs text-red-700 dark:bg-gray-800 dark:text-red-300">
                {this.state.error.name}: {this.state.error.message}
              </pre>
            )}

            <div className="flex gap-2">
              <button
                onClick={this.handleRetry}
                className="flex-1 rounded-lg bg-blue-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900"
              >
                Try Again
              </button>
              <button
                onClick={this.handleReload}
                className="flex-1 rounded-lg bg-gray-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900"
              >
                Reload Page
              </button>
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
