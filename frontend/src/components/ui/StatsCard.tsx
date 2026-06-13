import React from 'react';
import { LucideIcon, TrendingUp, TrendingDown } from 'lucide-react';

interface StatsCardProps {
    title: string;
    value: string | number;
    subtitle?: string;
    icon: LucideIcon;
    iconColor?: string;
    iconBgColor?: string;
    trend?: {
        value: number;
        label: string;
        direction: 'up' | 'down';
    };
    className?: string;
}

export function StatsCard({
    title,
    value,
    subtitle,
    icon: Icon,
    iconColor = 'text-blue-600',
    iconBgColor = 'bg-blue-50',
    trend,
    className = '',
}: StatsCardProps) {
    return (
        <div
            className={`bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 shadow-sm p-5 hover:shadow-md transition-shadow ${className}`}
        >
            <div className="flex items-start justify-between">
                <div className="flex-1">
                    <p className="text-sm font-medium text-slate-500 dark:text-slate-300">{title}</p>
                    <p className="mt-2 text-3xl font-bold text-slate-900 dark:text-white">{value}</p>
                    {subtitle && (
                        <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">{subtitle}</p>
                    )}
                    {trend && (
                        <div className="mt-2 flex items-center gap-1">
                            {trend.direction === 'up' ? (
                                <TrendingUp className="w-4 h-4 text-emerald-500" />
                            ) : (
                                <TrendingDown className="w-4 h-4 text-red-500" />
                            )}
                            <span
                                className={`text-sm font-medium ${trend.direction === 'up' ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'
                                    }`}
                            >
                                {trend.value}%
                            </span>
                            <span className="text-sm text-slate-500 dark:text-slate-400">{trend.label}</span>
                        </div>
                    )}
                </div>
                <div className={`p-3 rounded-xl ${iconBgColor}`}>
                    <Icon className={`w-6 h-6 ${iconColor}`} />
                </div>
            </div>
        </div>
    );
}

// Compact version for dashboard grids
interface MiniStatsCardProps {
    title: string;
    value: string | number;
    icon: LucideIcon;
    color?: 'blue' | 'green' | 'red' | 'amber' | 'purple';
}

const colorClasses = {
    blue: { icon: 'text-blue-600', bg: 'bg-blue-50', accent: 'border-l-blue-500' },
    green: { icon: 'text-emerald-600', bg: 'bg-emerald-50', accent: 'border-l-emerald-500' },
    red: { icon: 'text-red-600', bg: 'bg-red-50', accent: 'border-l-red-500' },
    amber: { icon: 'text-amber-600', bg: 'bg-amber-50', accent: 'border-l-amber-500' },
    purple: { icon: 'text-purple-600', bg: 'bg-purple-50', accent: 'border-l-purple-500' },
};

export function MiniStatsCard({
    title,
    value,
    icon: Icon,
    color = 'blue',
}: MiniStatsCardProps) {
    const colors = colorClasses[color];

    return (
        <div
            className={`bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 shadow-sm p-4 border-l-4 ${colors.accent} hover:shadow-md transition-shadow`}
        >
            <div className="flex items-center gap-3">
                <div className={`p-2 rounded-lg ${colors.bg}`}>
                    <Icon className={`w-5 h-5 ${colors.icon}`} />
                </div>
                <div>
                    <p className="text-2xl font-bold text-slate-900 dark:text-white">{value}</p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">{title}</p>
                </div>
            </div>
        </div>
    );
}
