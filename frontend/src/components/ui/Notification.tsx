// ═══════════════════════════════════════════════════════════════════════
// Notification — Accessible notification component (WCAG 2.1 AA)
//
// P2-MED-15: aria-live="polite" для динамических уведомлений
// - Используется для списка уведомлений, которые обновляются динамически
// - aria-live="polite" — не перебивает текущее действие пользователя
// - aria-atomic="true" — весь контент читается как одно целое
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';

type NotificationType = 'info' | 'success' | 'warning' | 'error';

interface NotificationProps {
    children: React.ReactNode;
    type?: NotificationType;
    title?: string;
    className?: string;
    /** aria-live режим: polite (по умолчанию) или assertive (для критических) */
    assertive?: boolean;
}

const typeStyles: Record<NotificationType, string> = {
    info: 'bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-900/30 dark:border-blue-800 dark:text-blue-300',
    success: 'bg-emerald-50 border-emerald-200 text-emerald-800 dark:bg-emerald-900/30 dark:border-emerald-800 dark:text-emerald-300',
    warning: 'bg-amber-50 border-amber-200 text-amber-800 dark:bg-amber-900/30 dark:border-amber-800 dark:text-amber-300',
    error: 'bg-red-50 border-red-200 text-red-800 dark:bg-red-900/30 dark:border-red-800 dark:text-red-300',
};

/**
 * Notification — компонент для динамических уведомлений с aria-live поддержкой.
 *
 * Особенности:
 * - aria-live="polite" по умолчанию (не перебивает пользователя)
 * - aria-live="assertive" для критических (type="error")
 * - aria-atomic="true" — озвучивается целиком
 */
export function Notification({
    children,
    type = 'info',
    title,
    className = '',
    assertive,
}: NotificationProps) {
    const isAssertive = assertive ?? (type === 'error');

    return (
        <div
            role="status"
            aria-live={isAssertive ? 'assertive' : 'polite'}
            aria-atomic="true"
            className={`
                flex items-start gap-3 p-4 rounded-lg border
                ${typeStyles[type]}
                ${className}
            `}
        >
            <div className="flex-1 min-w-0">
                {title && (
                    <p className="font-medium text-sm mb-1">{title}</p>
                )}
                <div className="text-sm">{children}</div>
            </div>
        </div>
    );
}

/**
 * NotificationList — обёртка для списка уведомлений с aria-live.
 *
 * Используется как контейнер с aria-live="polite", который объявляет
 * screen reader'у о добавлении/изменении уведомлений.
 */
export function NotificationList({
    children,
    label = 'Notifications',
    className = '',
}: {
    children: React.ReactNode;
    label?: string;
    className?: string;
}) {
    return (
        <div
            aria-live="polite"
            aria-label={label}
            aria-atomic="false"
            className={className}
        >
            {children}
        </div>
    );
}
