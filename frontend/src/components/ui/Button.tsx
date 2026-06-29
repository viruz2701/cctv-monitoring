import React, { useCallback } from 'react';
import { Loader2 } from './Icons';
import { useRipple } from '../../hooks/useRipple';
import { useHapticFeedback } from '../../hooks/useHapticFeedback';

type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger';
type ButtonSize = 'sm' | 'md' | 'lg';

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: ButtonVariant;
    size?: ButtonSize;
    loading?: boolean;
    icon?: React.ReactNode;
    iconPosition?: 'left' | 'right';
    fullWidth?: boolean;
    /** Отключить ripple-эффект */
    noRipple?: boolean;
    /** Отключить haptic feedback */
    noHaptic?: boolean;
}

const variantClasses: Record<ButtonVariant, string> = {
    primary:
        'bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800 shadow-sm dark:bg-blue-600 dark:hover:bg-blue-500',
    secondary:
        'bg-slate-100 text-slate-700 hover:bg-slate-200 active:bg-slate-300 dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700 dark:active:bg-slate-600',
    outline:
        'border border-slate-300 text-slate-700 hover:bg-slate-50 active:bg-slate-100 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-800 dark:active:bg-slate-800/80',
    ghost: 'text-slate-700 hover:bg-slate-100 active:bg-slate-200 dark:text-slate-300 dark:hover:bg-slate-800 dark:active:bg-slate-800/80',
    danger: 'bg-red-600 text-white hover:bg-red-700 active:bg-red-800 shadow-sm dark:bg-red-600 dark:hover:bg-red-500',
};

const sizeClasses: Record<ButtonSize, string> = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-sm',
    lg: 'px-5 py-2.5 text-base',
};

export function Button({
    children,
    variant = 'primary',
    size = 'md',
    loading = false,
    icon,
    iconPosition = 'left',
    fullWidth = false,
    disabled,
    className = '',
    noRipple = false,
    noHaptic = false,
    onClick,
    ...props
}: ButtonProps) {
    const isDisabled = disabled || loading;
    const { createRipple, ripples } = useRipple();
    const haptics = useHapticFeedback();

    const handleClick = useCallback(
        (e: React.MouseEvent<HTMLButtonElement>) => {
            if (isDisabled) return;
            if (!noRipple) createRipple(e);
            if (!noHaptic) haptics.light();
            onClick?.(e);
        },
        [isDisabled, noRipple, noHaptic, createRipple, haptics, onClick],
    );

    return (
        <button
            className={`
        inline-flex items-center justify-center gap-2 font-medium rounded-lg
        transition-all duration-150 ease-in-out
        focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
        disabled:opacity-50 disabled:cursor-not-allowed
        ${noRipple ? '' : 'ripple-container'}
        ${variantClasses[variant]}
        ${sizeClasses[size]}
        ${fullWidth ? 'w-full' : ''}
        ${className}
      `}
            disabled={isDisabled}
            aria-disabled={isDisabled || undefined}
            aria-busy={loading || undefined}
            onClick={handleClick}
            {...props}
        >
            {loading && <Loader2 className="w-4 h-4 animate-spin" aria-hidden="true" />}
            {!loading && icon && iconPosition === 'left' && icon}
            {children}
            {!loading && icon && iconPosition === 'right' && icon}
            {ripples}
        </button>
    );
}

// Icon-only button variant
interface IconButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
    icon: React.ReactNode;
    variant?: ButtonVariant;
    size?: 'sm' | 'md' | 'lg';
    label: string; // for accessibility
}

export function IconButton({
    icon,
    variant = 'ghost',
    size = 'md',
    label,
    className = '',
    noRipple = false,
    noHaptic = false,
    onClick,
    ...props
}: IconButtonProps & { noRipple?: boolean; noHaptic?: boolean }) {
    const { createRipple, ripples } = useRipple();
    const haptics = useHapticFeedback();

    const sizeClassMap = {
        sm: 'p-1.5',
        md: 'p-2',
        lg: 'p-2.5',
    };

    const handleClick = useCallback(
        (e: React.MouseEvent<HTMLButtonElement>) => {
            if (!noRipple) createRipple(e);
            if (!noHaptic) haptics.light();
            onClick?.(e);
        },
        [noRipple, noHaptic, createRipple, haptics, onClick],
    );

    return (
        <button
            className={`
        inline-flex items-center justify-center rounded-lg
        transition-all duration-150 ease-in-out
        focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
        disabled:opacity-50 disabled:cursor-not-allowed
        ${noRipple ? '' : 'ripple-container'}
        ${variantClasses[variant]}
        ${sizeClassMap[size]}
        ${className}
      `}
            aria-label={label}
            onClick={handleClick}
            {...props}
        >
            {icon}
            {ripples}
        </button>
    );
}
