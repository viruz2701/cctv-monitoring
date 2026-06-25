import React from 'react';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonBaseProps {
    className?: string;
}

interface SkeletonRepeatProps extends SkeletonBaseProps {
    count?: number;
}

// ═══════════════════════════════════════════════════════════════════════
// Skeleton — базовый блок с pulse-анимацией
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonProps extends SkeletonRepeatProps {
    variant?: 'text' | 'circular' | 'rectangular' | 'rounded';
    width?: string | number;
    height?: string | number;
}

export function Skeleton({
    variant = 'rounded',
    width,
    height,
    className = '',
    count = 1,
}: SkeletonProps) {
    const baseClass = 'bg-slate-200 dark:bg-slate-700 animate-pulse';

    const variantClass = {
        text: 'rounded h-4',
        circular: 'rounded-full',
        rectangular: 'rounded-none',
        rounded: 'rounded-lg',
    }[variant];

    const items = Array.from({ length: count }, (_, i) => i);

    return (
        <>
            {items.map((i) => (
                <div
                    key={i}
                    className={`${baseClass} ${variantClass} ${className}`}
                    style={{ width, height }}
                    aria-hidden="true"
                />
            ))}
        </>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonLine — текстовая строка разной ширины
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonLineProps extends SkeletonRepeatProps {
    widths?: string[];
    lastWidth?: string;
}

export function SkeletonLine({
    className = '',
    count = 1,
    widths,
    lastWidth = '60%',
}: SkeletonLineProps) {
    const items = Array.from({ length: count }, (_, i) => i);

    return (
        <>
            {items.map((i) => (
                <div key={i} className={`space-y-2 ${className}`} aria-hidden="true">
                    <div
                        className="h-3.5 bg-slate-200 dark:bg-slate-700 animate-pulse rounded"
                        style={{ width: widths?.[0] || '100%' }}
                    />
                    <div
                        className="h-3.5 bg-slate-200 dark:bg-slate-700 animate-pulse rounded"
                        style={{ width: widths?.[1] || lastWidth || '60%' }}
                    />
                </div>
            ))}
        </>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonAvatar — круглый аватар
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonAvatarProps extends SkeletonRepeatProps {
    size?: 'sm' | 'md' | 'lg' | 'xl';
}

const avatarSizes = {
    sm: 'w-8 h-8',
    md: 'w-10 h-10',
    lg: 'w-14 h-14',
    xl: 'w-20 h-20',
};

export function SkeletonAvatar({
    size = 'md',
    className = '',
    count = 1,
}: SkeletonAvatarProps) {
    const items = Array.from({ length: count }, (_, i) => i);

    return (
        <div className={`flex items-center gap-3 ${className}`} aria-hidden="true">
            {items.map((i) => (
                <div
                    key={i}
                    className={`${avatarSizes[size]} bg-slate-200 dark:bg-slate-700 animate-pulse rounded-full flex-shrink-0`}
                />
            ))}
        </div>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonStatsCard — скелетон для StatsCard
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonStatsCardProps extends SkeletonRepeatProps {
    withTrend?: boolean;
}

export function SkeletonStatsCard({
    className = '',
    count = 1,
    withTrend = false,
}: SkeletonStatsCardProps) {
    const items = Array.from({ length: count }, (_, i) => i);

    return (
        <>
            {items.map((i) => (
                <div
                    key={i}
                    className={`bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-5 ${className}`}
                    aria-hidden="true"
                >
                    <div className="flex items-start justify-between">
                        <div className="flex-1 space-y-3">
                            {/* title */}
                            <div className="h-4 w-24 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                            {/* value */}
                            <div className="h-8 w-20 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                            {/* subtitle */}
                            <div className="h-3.5 w-32 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                            {withTrend && (
                                <div className="h-3.5 w-28 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                            )}
                        </div>
                        {/* icon */}
                        <div className="p-3 rounded-xl bg-slate-200 dark:bg-slate-700 animate-pulse">
                            <div className="w-6 h-6" />
                        </div>
                    </div>
                </div>
            ))}
        </>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonCard — карточка с header + body
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonCardProps extends SkeletonRepeatProps {
    headerLines?: number;
    bodyLines?: number;
    avatar?: boolean;
}

export function SkeletonCard({
    className = '',
    count = 1,
    headerLines = 1,
    bodyLines = 3,
    avatar = false,
}: SkeletonCardProps) {
    const items = Array.from({ length: count }, (_, i) => i);

    return (
        <>
            {items.map((i) => (
                <div
                    key={i}
                    className={`bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm overflow-hidden ${className}`}
                    aria-hidden="true"
                >
                    {/* Header */}
                    <div className="p-4 border-b border-slate-100 dark:border-slate-700">
                        <div className="flex items-center gap-3">
                            {avatar && (
                                <div className="w-10 h-10 bg-slate-200 dark:bg-slate-700 animate-pulse rounded-full flex-shrink-0" />
                            )}
                            <div className="flex-1 space-y-2">
                                {Array.from({ length: headerLines }, (_, h) => (
                                    <div
                                        key={h}
                                        className="h-4 bg-slate-200 dark:bg-slate-700 animate-pulse rounded"
                                        style={{ width: `${100 - h * 20}%` }}
                                    />
                                ))}
                            </div>
                        </div>
                    </div>
                    {/* Body */}
                    <div className="p-4 space-y-3">
                        {Array.from({ length: bodyLines }, (_, b) => (
                            <div
                                key={b}
                                className="h-3.5 bg-slate-200 dark:bg-slate-700 animate-pulse rounded"
                                style={{ width: `${100 - b * 10}%` }}
                            />
                        ))}
                    </div>
                </div>
            ))}
        </>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonTable — таблица с рядами и колонками
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonTableProps {
    rows?: number;
    columns?: number;
    className?: string;
}

export function SkeletonTable({
    rows = 5,
    columns = 4,
    className = '',
}: SkeletonTableProps) {
    return (
        <div className={`bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden ${className}`} aria-hidden="true">
            {/* Header */}
            <div className="grid grid-cols-12 gap-4 px-6 py-4 border-b border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
                {Array.from({ length: columns }, (_, c) => (
                    <div
                        key={c}
                        className="h-4 bg-slate-300 dark:bg-slate-600 animate-pulse rounded col-span-3"
                    />
                ))}
            </div>
            {/* Rows */}
            {Array.from({ length: rows }, (_, r) => (
                <div
                    key={r}
                    className="grid grid-cols-12 gap-4 px-6 py-4 border-b border-slate-100 dark:border-slate-700 last:border-b-0"
                >
                    {Array.from({ length: columns }, (_, c) => (
                        <div
                            key={c}
                            className="h-4 bg-slate-200 dark:bg-slate-700 animate-pulse rounded col-span-3"
                            style={{ width: c === 0 ? '85%' : c === columns - 1 ? '60%' : '70%' }}
                        />
                    ))}
                </div>
            ))}
        </div>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonChart — график-скелетон
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonChartProps {
    height?: number | string;
    className?: string;
    withHeader?: boolean;
}

export function SkeletonChart({
    height = 260,
    className = '',
    withHeader = true,
}: SkeletonChartProps) {
    return (
        <div className={`bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm ${className}`} aria-hidden="true">
            {withHeader && (
                <div className="p-4 border-b border-slate-100 dark:border-slate-700">
                    <div className="h-4 w-40 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                </div>
            )}
            <div
                className="flex items-center justify-center p-4"
                style={{ height }}
            >
                <div className="w-full h-full relative">
                    {/* Y-axis lines */}
                    <div className="absolute inset-0 flex flex-col justify-between py-2">
                        {Array.from({ length: 4 }, (_, i) => (
                            <div key={i} className="h-px bg-slate-100 dark:bg-slate-700 w-full" />
                        ))}
                    </div>
                    {/* Bars mock */}
                    <div className="absolute inset-0 flex items-end justify-around pb-4 px-4 gap-2">
                        {Array.from({ length: 6 }, (_, i) => (
                            <div
                                key={i}
                                className="flex-1 bg-slate-200 dark:bg-slate-700 animate-pulse rounded-t-md"
                                style={{ height: `${30 + Math.random() * 50}%` }}
                            />
                        ))}
                    </div>
                </div>
            </div>
        </div>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonFilterBar — скелетон для фильтров
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonFilterBar({ className = '' }: SkeletonBaseProps) {
    return (
        <div className={`flex flex-col sm:flex-row gap-3 p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl border border-slate-200 dark:border-slate-700 ${className}`} aria-hidden="true">
            <div className="flex-1 max-w-md space-y-2">
                <div className="h-3.5 w-12 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                <div className="h-10 w-full bg-slate-200 dark:bg-slate-700 animate-pulse rounded-lg" />
            </div>
            <div className="flex gap-3">
                {Array.from({ length: 2 }, (_, i) => (
                    <div key={i} className="space-y-2">
                        <div className="h-3.5 w-12 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                        <div className="h-10 w-[140px] bg-slate-200 dark:bg-slate-700 animate-pulse rounded-lg" />
                    </div>
                ))}
            </div>
        </div>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonPage — полный скелетон страницы (заголовок + контент)
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonPageProps {
    title?: boolean;
    subtitle?: boolean;
    filter?: boolean;
    children: React.ReactNode;
    className?: string;
}

export function SkeletonPage({
    title = true,
    subtitle = true,
    filter = false,
    children,
    className = '',
}: SkeletonPageProps) {
    return (
        <div className={`space-y-6 ${className}`} aria-label="Loading content">
            {/* Header */}
            {(title || subtitle) && (
                <div className="space-y-2">
                    {title && (
                        <div className="h-7 w-48 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                    )}
                    {subtitle && (
                        <div className="h-4 w-64 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                    )}
                </div>
            )}
            {/* Filters */}
            {filter && <SkeletonFilterBar />}
            {/* Content */}
            {children}
        </div>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonProfileField — скелетон для поля профиля
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonProfileFieldProps {
    className?: string;
}

export function SkeletonProfileField({ className = '' }: SkeletonProfileFieldProps) {
    return (
        <div className={`space-y-2 ${className}`} aria-hidden="true">
            <div className="h-3 w-20 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
            <div className="h-10 w-full bg-slate-200 dark:bg-slate-700 animate-pulse rounded-xl" />
        </div>
    );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonNotification — скелетон для уведомления
// ═══════════════════════════════════════════════════════════════════════

interface SkeletonNotificationProps {
    count?: number;
    className?: string;
}

export function SkeletonNotification({ count = 3, className = '' }: SkeletonNotificationProps) {
    const items = Array.from({ length: count }, (_, i) => i);

    return (
        <div className={`space-y-2 ${className}`} aria-hidden="true">
            {items.map((i) => (
                <div
                    key={i}
                    className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-4 flex items-start gap-3"
                >
                    <div className="w-8 h-8 bg-slate-200 dark:bg-slate-700 animate-pulse rounded-full flex-shrink-0" />
                    <div className="flex-1 space-y-2">
                        <div className="flex items-start justify-between gap-2">
                            <div className="h-4 w-3/5 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                            <div className="h-3 w-12 bg-slate-200 dark:bg-slate-700 animate-pulse rounded flex-shrink-0" />
                        </div>
                        <div className="h-3.5 w-full bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                        <div className="h-3 w-2/5 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
                    </div>
                </div>
            ))}
        </div>
    );
}
