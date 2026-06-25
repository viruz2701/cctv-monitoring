import { Component, type ErrorInfo, type ReactNode } from 'react';

// ─────────────────────────────────────────────────────────────────────────────
// ErrorBoundaryLite — лёгкий ErrorBoundary для per-route использования
// ─────────────────────────────────────────────────────────────────────────────
// Отличия от глобального ErrorBoundary:
// 1. НЕ логирует в консоль (не засоряет devtools)
// 2. Минимальный UI: сообщение + кнопка Try Again
// 3. Показывает trace_id если доступен
// 4. Не отправляет ошибки на backend (это задача глобального)
// ─────────────────────────────────────────────────────────────────────────────

interface ErrorBoundaryLiteProps {
  children: ReactNode;
}

interface ErrorBoundaryLiteState {
  hasError: boolean;
  error: Error | null;
  traceId: string;
}

function generateTraceId(): string {
  const arr = new Uint8Array(16);
  crypto.getRandomValues(arr);
  return Array.from(arr)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

export class ErrorBoundaryLite extends Component<ErrorBoundaryLiteProps, ErrorBoundaryLiteState> {
  state: ErrorBoundaryLiteState = { hasError: false, error: null, traceId: '' };

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryLiteState> {
    return { hasError: true, error };
  }

  componentDidCatch(_error: Error, _info: ErrorInfo) {
    // Лёгкая версия — НЕ логируем в консоль
    const traceId = generateTraceId();
    this.setState({ traceId });
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null, traceId: '' });
  };

  render() {
    if (this.state.hasError) {
      const { traceId } = this.state;

      return (
        <div className="flex items-center justify-center p-12">
          <div className="max-w-md rounded-xl border border-red-200 bg-white p-8 text-center shadow-lg dark:border-red-900 dark:bg-gray-900">
            <div className="mb-4 flex justify-center">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/30">
                <svg
                  className="h-6 w-6 text-red-600 dark:text-red-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  strokeWidth={1.5}
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z"
                  />
                </svg>
              </div>
            </div>
            <h2 className="mb-2 text-lg font-semibold text-gray-900 dark:text-gray-100">
              Something went wrong
            </h2>
            <p className="mb-4 text-sm text-gray-600 dark:text-gray-400">
              An unexpected error occurred in this section.
            </p>
            {traceId && (
              <p className="mb-4 text-xs font-mono text-gray-400 dark:text-gray-500">
                Error ID:{' '}
                <span className="text-gray-500 dark:text-gray-400">{traceId}</span>
              </p>
            )}
            <button
              onClick={this.handleRetry}
              className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-gray-900"
            >
              Try Again
            </button>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}
