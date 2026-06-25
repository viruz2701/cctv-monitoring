import React from 'react';
import { useTranslation } from 'react-i18next';

type BadgeVariant = 'success' | 'warning' | 'danger' | 'info' | 'neutral' | 'primary';
type BadgeSize = 'sm' | 'md' | 'lg';

interface BadgeProps {
    children: React.ReactNode;
    variant?: BadgeVariant;
    size?: BadgeSize;
    dot?: boolean;
    className?: string;
    /** aria-label для screen readers, если children не достаточно */
    ariaLabel?: string;
}

const variantClasses: Record<BadgeVariant, string> = {
    success: 'bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400 dark:border-emerald-800',
    warning: 'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-900/30 dark:text-amber-400 dark:border-amber-800',
    danger: 'bg-red-50 text-red-700 border-red-200 dark:bg-red-900/30 dark:text-red-400 dark:border-red-800',
    info: 'bg-cyan-50 text-cyan-700 border-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400 dark:border-cyan-800',
    neutral: 'bg-slate-100 text-slate-700 border-slate-200 dark:bg-slate-800 dark:text-slate-300 dark:border-slate-700',
    primary: 'bg-blue-50 text-blue-700 border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800',
};

const dotColors: Record<BadgeVariant, string> = {
    success: 'bg-emerald-500',
    warning: 'bg-amber-500',
    danger: 'bg-red-500',
    info: 'bg-cyan-500',
    neutral: 'bg-slate-500',
    primary: 'bg-blue-500',
};

const sizeClasses: Record<BadgeSize, string> = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-xs',
    lg: 'px-3 py-1.5 text-sm',
};

const variantLabels: Record<BadgeVariant, string> = {
    success: 'Success',
    warning: 'Warning',
    danger: 'Danger',
    info: 'Info',
    neutral: 'Neutral',
    primary: 'Primary',
};

export function Badge({
    children,
    variant = 'neutral',
    size = 'md',
    dot = false,
    className = '',
    ariaLabel,
}: BadgeProps) {
    // WCAG 2.1 AA: если dot индикатор, добавляем aria-label с описанием цвета (UX-14.2.7)
    const computedLabel = ariaLabel || (dot ? `${variantLabels[variant]} status` : undefined);

    return (
        <span
            className={`inline-flex items-center gap-1.5 font-medium rounded-full border ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
            aria-label={computedLabel}
        >
            {dot && (
                <span
                    className={`w-1.5 h-1.5 rounded-full ${dotColors[variant]}`}
                    aria-hidden="true"
                />
            )}
            {children}
        </span>
    );
}

// Статус устройства / объекта
export function StatusBadge({ status }: { status: 'online' | 'offline' | 'warning' | 'active' | 'inactive' | 'maintenance' }) {
    const { t } = useTranslation();
    const config = {
        online: { variant: 'success' as BadgeVariant, label: t('online') },
        offline: { variant: 'danger' as BadgeVariant, label: t('offline') },
        warning: { variant: 'warning' as BadgeVariant, label: t('warning') },
        active: { variant: 'success' as BadgeVariant, label: t('active') },
        inactive: { variant: 'neutral' as BadgeVariant, label: t('inactive') },
        maintenance: { variant: 'warning' as BadgeVariant, label: t('maintenance') },
    };
    const { variant, label } = config[status] || { variant: 'neutral' as BadgeVariant, label: status };
    return <Badge variant={variant} dot ariaLabel={`Status: ${label}`}>{label}</Badge>;
}

// Здоровье устройства
export function HealthBadge({ health }: { health: 'healthy' | 'faulty' | 'degraded' }) {
    const { t } = useTranslation();
    const config = {
        healthy: { variant: 'success' as BadgeVariant, label: t('healthy') },
        faulty: { variant: 'danger' as BadgeVariant, label: t('faulty') },
        degraded: { variant: 'warning' as BadgeVariant, label: t('degraded') },
    };
    const { variant, label } = config[health];
    return <Badge variant={variant} ariaLabel={`Device health: ${label}`}>{label}</Badge>;
}

// Приоритет заявки
export function PriorityBadge({ priority }: { priority: 'critical' | 'high' | 'medium' | 'low' }) {
    const { t } = useTranslation();
    const config = {
        critical: { variant: 'danger' as BadgeVariant, label: t('critical') },
        high: { variant: 'warning' as BadgeVariant, label: t('high') },
        medium: { variant: 'info' as BadgeVariant, label: t('medium') },
        low: { variant: 'neutral' as BadgeVariant, label: t('low') },
    };
    const { variant, label } = config[priority];
    return <Badge variant={variant} ariaLabel={`Priority: ${label}`}>{label}</Badge>;
}

// Статус заявки
export function TicketStatusBadge({ status }: { status: 'open' | 'in_progress' | 'pending' | 'resolved' | 'closed' }) {
    const { t } = useTranslation();
    const config = {
        open: { variant: 'danger' as BadgeVariant, label: t('open') },
        in_progress: { variant: 'primary' as BadgeVariant, label: t('in_progress') },
        pending: { variant: 'warning' as BadgeVariant, label: t('pending') },
        resolved: { variant: 'success' as BadgeVariant, label: t('resolved') },
        closed: { variant: 'neutral' as BadgeVariant, label: t('closed') },
    };
    const { variant, label } = config[status];
    return <Badge variant={variant} ariaLabel={`Ticket status: ${label}`}>{label}</Badge>;
}

// Роль пользователя
export function RoleBadge({ role }: { role: 'admin' | 'manager' | 'technician' | 'viewer' | 'owner' | 'support' }) {
    const { t } = useTranslation();
    const config = {
        admin: { variant: 'danger' as BadgeVariant, label: t('admin') },
        manager: { variant: 'primary' as BadgeVariant, label: t('manager') },
        technician: { variant: 'info' as BadgeVariant, label: t('technician') },
        viewer: { variant: 'neutral' as BadgeVariant, label: t('viewer') },
        owner: { variant: 'primary' as BadgeVariant, label: t('owner') },
        support: { variant: 'info' as BadgeVariant, label: t('support') },
    };
    const { variant, label } = config[role];
    return <Badge variant={variant} ariaLabel={`Role: ${label}`}>{label}</Badge>;
}