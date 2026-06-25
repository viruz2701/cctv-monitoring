// ═══════════════════════════════════════════════════════════════════════
// Alert — Accessible alert component (WCAG 2.1 AA, UX-14.2.7)
//
// Используй для критических сообщений.
// - role="alert" для немедленного объявления screen reader
// - aria-live="assertive" для важных динамических обновлений
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { AlertCircle, AlertTriangle, CheckCircle, Info, X } from 'lucide-react';

type AlertVariant = 'info' | 'success' | 'warning' | 'error';

interface AlertProps {
    children: React.ReactNode;
    variant?: AlertVariant;
    title?: string;
    onClose?: () => void;
    className?: string;
    /** По умолчанию true для error/warning, false для info/success */
    assertive?: boolean;
}

const variantStyles: Record<AlertVariant, string> = {
    info: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-900/30 dark:border-blue-800 dark:text-blue-300',
    success: 'bg-emerald-50 border-emerald-200 text-emerald-800 dark:bg-emerald-900/30 dark:border-emerald-800 dark:text-emerald-300',
    warning: 'bg-amber-50 border-amber-200 text-amber-800 dark:bg-amber-900/30 dark:border-amber-800 dark:text-amber-300',
    error: 'bg-red-50 border-red-200 text-red-800 dark:bg-red-900/30 dark:border-red-800 dark:text-red-300',
};

const iconMap: Record<AlertVariant, React.ReactNode> = {
    info: <Info className="w-5 h-5" aria-hidden="true" />,
    success: <CheckCircle className="w-5 h-5" aria-hidden="true" />,
    warning: <AlertTriangle className="w-5 h-5" aria-hidden="true" />,
    error: <AlertCircle className="w-5 h-5" aria-hidden="true" />,
};

export function Alert({
    children,
    variant = 'info',
    title,
    onClose,
    className = '',
    assertive,
}: AlertProps) {
    const isAssertive = assertive ?? (variant === 'error' || variant === 'warning');

    return (
        <div
            role="alert"
            aria-live={isAssertive ? 'assertive' : 'polite'}
            aria-atomic="true"
            className={`
        flex items-start gap-3 p-4 rounded-lg border
        ${variantStyles[variant]}
        ${className}
      `}
        >
            <span className="flex-shrink-0 mt-0.5" aria-hidden="true">
                {iconMap[variant]}
            </span>

            <div className="flex-1 min-w-0">
                {title && (
                    <p className="font-medium text-sm mb-1">{title}</p>
                )}
                <div className="text-sm">{children}</div>
            </div>

            {onClose && (
                <button
                    type="button"
                    onClick={onClose}
                    className="flex-shrink-0 p-1 rounded-md hover:bg-black/10 dark:hover:bg-white/10 transition-colors"
                    aria-label="Закрыть уведомление"
                >
                    <X className="w-4 h-4" aria-hidden="true" />
                </button>
            )}
        </div>
    );
}
