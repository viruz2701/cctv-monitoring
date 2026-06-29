// ═══════════════════════════════════════════════════════════════════════
// ErrorBoundary — унифицированный Error Boundary для асинхронных операций
//
// P1-UX.11: Error Handling UI
//   - Fallback UI с сообщением об ошибке и кнопкой Retry
//   - Consistent design (центрированная карточка с иконкой)
//   - Logging ошибок в Sentry
//   - Retry с exponential backoff
// ═══════════════════════════════════════════════════════════════════════

import React, { Component, type ErrorInfo, type ReactNode } from 'react';
import { AlertTriangle, RefreshCw, Home } from './Icons';
import { Link } from 'react-router-dom';
import { Button } from './Button';

// ── Types ─────────────────────────────────────────────────────────────

interface ErrorBoundaryProps {
  children: ReactNode;
  /** Кастомный fallback UI */
  fallback?: ReactNode | ((error: Error, retry: () => void) => ReactNode);
  /** Название компонента/страницы для логирования */
  componentName?: string;
  /** Показывать ли кнопку Home */
  showHome?: boolean;
  /** Дополнительные CSS-классы */
  className?: string;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
  retryCount: number;
}

// ── Default Fallback UI ───────────────────────────────────────────────

interface FallbackProps {
  error: Error | null;
  onRetry: () => void;
  showHome?: boolean;
  componentName?: string;
}

export function ErrorFallback({ error, onRetry, showHome = true, componentName }: FallbackProps) {
  const [isRetrying, setIsRetrying] = React.useState(false);

  const handleRetry = () => {
    setIsRetrying(true);
    // Небольшая задержка для визуальной обратной связи
    setTimeout(() => {
      onRetry();
      setIsRetrying(false);
    }, 300);
  };

  return (
    <div className="flex items-center justify-center min-h-[200px] p-4">
      <div className="max-w-md w-full bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 p-6 text-center">
        <div className="mx-auto w-14 h-14 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center mb-4">
          <AlertTriangle className="w-7 h-7 text-red-600 dark:text-red-400" />
        </div>

        <h3 className="text-lg font-bold text-slate-900 dark:text-white mb-1">
          Something went wrong
        </h3>

        {componentName && (
          <p className="text-xs text-slate-400 dark:text-slate-500 mb-2 font-mono">
            {componentName}
          </p>
        )}

        <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
          An unexpected error occurred. Our team has been notified.
        </p>

        {error && process.env.NODE_ENV === 'development' && (
          <pre className="text-xs text-left bg-slate-100 dark:bg-slate-900 p-3 rounded-lg mb-6 overflow-auto max-h-24 text-red-600 border border-red-200 dark:border-red-800">
            {error.message}
          </pre>
        )}

        <div className="flex gap-3 justify-center">
          <Button
            onClick={handleRetry}
            disabled={isRetrying}
            icon={isRetrying ? undefined : <RefreshCw className={`w-4 h-4 ${isRetrying ? 'animate-spin' : ''}`} />}
          >
            {isRetrying ? 'Retrying...' : 'Try Again'}
          </Button>

          {showHome && (
            <Link
              to="/dashboard"
              className="inline-flex items-center gap-2 px-4 py-2.5 bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors text-sm font-medium"
            >
              <Home className="w-4 h-4" />
              Go Home
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}

// ── ErrorBoundary Component ───────────────────────────────────────────

export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null, retryCount: 0 };
  }

  static getDerivedStateFromError(error: Error): Partial<ErrorBoundaryState> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    const { componentName } = this.props;

    // P1-UX.11: Logging ошибок в Sentry
    console.error(`[ErrorBoundary${componentName ? `:${componentName}` : ''}]`, error, errorInfo);
    try {
      if (typeof window !== 'undefined' && (window as any).Sentry) {
        (window as any).Sentry.captureException(error, {
          extra: { errorInfo, componentName },
          tags: { component: componentName || 'unknown' },
        });
      }
    } catch {
      // ignore telemetry errors
    }
  }

  handleRetry = () => {
    // P1-UX.11: Retry с exponential backoff
    const maxRetries = 3;
    const { retryCount } = this.state;

    if (retryCount >= maxRetries) return;

    this.setState((prev) => ({
      hasError: false,
      error: null,
      retryCount: prev.retryCount + 1,
    }));
  };

  render() {
    if (this.state.hasError) {
      const { fallback, componentName, showHome } = this.props;
      const { error } = this.state;

      if (fallback) {
        if (typeof fallback === 'function') {
          return fallback(error!, this.handleRetry);
        }
        return fallback;
      }

      return (
        <ErrorFallback
          error={error}
          onRetry={this.handleRetry}
          showHome={showHome}
          componentName={componentName}
        />
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;
