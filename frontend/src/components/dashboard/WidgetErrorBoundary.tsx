// WidgetErrorBoundary — Error boundary для отдельных виджетов дашборда.
// P1-2.1: Crash widget не ломает весь dashboard
// - Показывает только упавший виджет
// - Retry button
// - Error info в development

import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, RefreshCw } from 'lucide-react';

interface WidgetErrorBoundaryProps {
    children: React.ReactNode;
    widgetId?: string;
    widgetName?: string;
}

interface WidgetErrorBoundaryState {
    hasError: boolean;
    error: Error | null;
}

export class WidgetErrorBoundary extends React.Component<WidgetErrorBoundaryProps, WidgetErrorBoundaryState> {
    constructor(props: WidgetErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false, error: null };
    }

    static getDerivedStateFromError(error: Error): Partial<WidgetErrorBoundaryState> {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error) {
        console.error(`[WidgetErrorBoundary] ${this.props.widgetId || 'unknown'}:`, error);
    }

    handleRetry = () => {
        this.setState({ hasError: false, error: null });
    };

    render() {
        if (this.state.hasError) {
            return (
                <WidgetErrorFallback
                    widgetName={this.props.widgetName}
                    onRetry={this.handleRetry}
                    error={this.state.error}
                />
            );
        }
        return this.props.children;
    }
}

function WidgetErrorFallback({
    widgetName,
    onRetry,
    error,
}: {
    widgetName?: string;
    onRetry: () => void;
    error: Error | null;
}) {
    const { t } = useTranslation();
    return (
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4 h-full flex flex-col items-center justify-center text-center">
            <AlertCircle className="w-8 h-8 text-red-500 mb-2" />
            <p className="text-sm font-medium text-slate-900 dark:text-white mb-1">
                {widgetName || t('widget_error') || 'Widget Error'}
            </p>
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-3">
                {t('widget_error_description') || 'Failed to load this widget'}
            </p>
            {error && process.env.NODE_ENV === 'development' && (
                <pre className="text-xs text-red-500 mb-3 max-w-full overflow-hidden text-ellipsis">
                    {error.message}
                </pre>
            )}
            <button
                onClick={onRetry}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
                <RefreshCw className="w-3 h-3" />
                {t('retry') || 'Retry'}
            </button>
        </div>
    );
}
