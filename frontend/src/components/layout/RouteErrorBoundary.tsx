// RouteErrorBoundary — Error boundary для целых страниц.
// P1-2.1: Unified Error Boundary System
// - Интеграция с Sentry/OTel (опционально)
// - Retry button
// - User-friendly error messages

import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, RefreshCw, Home } from 'lucide-react';
import { Link } from 'react-router-dom';

interface RouteErrorBoundaryProps {
    children: React.ReactNode;
    fallback?: React.ReactNode;
}

interface RouteErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
    errorInfo: React.ErrorInfo | null;
}

export class RouteErrorBoundary extends React.Component<RouteErrorBoundaryProps, RouteErrorBoundaryState> {
    constructor(props: RouteErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null, errorInfo: null };
    }

    static getDerivedStateFromError(error: Error): Partial<RouteErrorBoundaryState> {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        this.setState({ errorInfo });
        // P1-2.1: Отправка в telemetry
        console.error('[RouteErrorBoundary]', error, errorInfo);
        try {
            if (typeof window !== 'undefined' && (window as any).Sentry) {
                (window as any).Sentry.captureException(error, { extra: { errorInfo } });
            }
        } catch {
            /* ignore telemetry errors */
        }
    }

    handleRetry = () => {
        this.setState({ hasError: false, error: null, errorInfo: null });
    };

    render() {
        if (this.state.hasError) {
            if (this.props.fallback) return this.props.fallback;
            return <RouteErrorFallback error={this.state.error} onRetry={this.handleRetry} />;
        }
        return this.props.children;
    }
}

function RouteErrorFallback({ error, onRetry }: { error: Error | null; onRetry: () => void }) {
    const { t } = useTranslation();
    return (
        <div className="flex items-center justify-center min-h-screen bg-slate-50 dark:bg-slate-900 p-4">
            <div className="max-w-md w-full bg-white dark:bg-slate-800 rounded-xl shadow-lg border border-slate-200 dark:border-slate-700 p-8 text-center">
                <div className="mx-auto w-16 h-16 bg-red-100 dark:bg-red-900/30 rounded-full flex items-center justify-center mb-4">
                    <AlertTriangle className="w-8 h-8 text-red-600 dark:text-red-400" />
                </div>
                <h2 className="text-xl font-bold text-slate-900 dark:text-white mb-2">
                    {t('something_went_wrong') || 'Something went wrong'}
                </h2>
                <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                    {t('page_error_description') || 'An unexpected error occurred while loading this page. Our team has been notified.'}
                </p>
                {error && process.env.NODE_ENV === 'development' && (
                    <pre className="text-xs text-left bg-slate-100 dark:bg-slate-900 p-3 rounded-lg mb-6 overflow-auto max-h-32 text-red-600">
                        {error.message}
                    </pre>
                )}
                <div className="flex gap-3 justify-center">
                    <button
                        onClick={onRetry}
                        className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
                    >
                        <RefreshCw className="w-4 h-4" />
                        {t('try_again') || 'Try Again'}
                    </button>
                    <Link
                        to="/dashboard"
                        className="flex items-center gap-2 px-4 py-2 bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
                    >
                        <Home className="w-4 h-4" />
                        {t('go_home') || 'Go Home'}
                    </Link>
                </div>
            </div>
        </div>
    );
}
