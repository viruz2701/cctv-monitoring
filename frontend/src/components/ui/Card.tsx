import React from 'react';

interface CardProps {
    children: React.ReactNode;
    className?: string;
    variant?: 'default' | 'elevated' | 'bordered';
    padding?: 'none' | 'sm' | 'md' | 'lg';
}

interface CardHeaderProps {
    children: React.ReactNode;
    className?: string;
    action?: React.ReactNode;
}

interface CardBodyProps {
    children: React.ReactNode;
    className?: string;
}

interface CardFooterProps {
    children: React.ReactNode;
    className?: string;
}

export function Card({
    children,
    className = '',
    variant = 'default',
    padding = 'md',
}: CardProps) {
    const variantClasses = {
        default: 'bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700',
        elevated: 'bg-white dark:bg-slate-800 shadow-lg border border-slate-100 dark:border-slate-700',
        bordered: 'bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-700',
    };

    const paddingClasses = {
        none: '',
        sm: 'p-3',
        md: 'p-5',
        lg: 'p-6',
    };

    return (
        <div
            className={`rounded-xl shadow-sm ${variantClasses[variant]} ${paddingClasses[padding]} ${className}`}
        >
            {children}
        </div>
    );
}

export function CardHeader({ children, className = '', action }: CardHeaderProps) {
    return (
        <div className={`flex items-center justify-between mb-4 ${className}`}>
            <div className="text-lg font-semibold text-slate-900 dark:text-white">{children}</div>
            {action && <div>{action}</div>}
        </div>
    );
}

export function CardBody({ children, className = '' }: CardBodyProps) {
    return <div className={className}>{children}</div>;
}

export function CardFooter({ children, className = '' }: CardFooterProps) {
    return (
        <div className={`mt-4 pt-4 border-t border-slate-100 dark:border-slate-700 ${className}`}>
            {children}
        </div>
    );
}
