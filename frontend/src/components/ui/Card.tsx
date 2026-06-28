// ═══════════════════════════════════════════════════════════════════════
// Card — универсальный компонент карточки (P3-UI.2)
//
// Особенности:
//   - Hover-тень (поднимается на 2px при наведении)
//   - Ripple-эффект (опционально, для clickable карточек)
//   - Haptic feedback (опционально, для clickable карточек)
//   - CSS custom properties для border-radius
//   - Анимация появления (scaleIn)
//
// Варианты:
//   - elevated:  тень + белый фон (по умолчанию)
//   - outlined:  border вместо тени
//   - flat:      без тени и border
//   - interactive: кликабельная с hover-эффектом
//
// Соответствие:
//   - WCAG 2.1 SC 2.3.3 (prefers-reduced-motion)
//   - OWASP ASVS V7 (graceful degradation)
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback } from 'react';
import { useRipple } from '../../hooks/useRipple';
import { useHapticFeedback } from '../../hooks/useHapticFeedback';

type CardVariant = 'elevated' | 'outlined' | 'flat' | 'interactive';

interface CardProps {
  children: React.ReactNode;
  variant?: CardVariant;
  className?: string;
  padding?: 'none' | 'sm' | 'md' | 'lg';
  /** Анимация появления */
  animate?: boolean;
  /** Ripple эффект (только для interactive) */
  noRipple?: boolean;
  /** Haptic feedback (только для interactive) */
  noHaptic?: boolean;
  onClick?: (e: React.MouseEvent<HTMLDivElement>) => void;
  role?: string;
  'aria-label'?: string;
}

const variantClasses: Record<CardVariant, string> = {
  elevated:
    'bg-white dark:bg-slate-800 shadow-card border border-slate-200 dark:border-slate-700',
  outlined:
    'bg-white dark:bg-slate-800 border-2 border-slate-200 dark:border-slate-700',
  flat: 'bg-slate-50 dark:bg-slate-800/50',
  interactive:
    'bg-white dark:bg-slate-800 shadow-card border border-slate-200 dark:border-slate-700 card-hover cursor-pointer',
};

const paddingClasses: Record<string, string> = {
  none: '',
  sm: 'p-3',
  md: 'p-4',
  lg: 'p-6',
};

export function Card({
  children,
  variant = 'elevated',
  className = '',
  padding = 'md',
  animate = false,
  noRipple = false,
  noHaptic = false,
  onClick,
  role,
  'aria-label': ariaLabel,
}: CardProps) {
  const { createRipple, ripples } = useRipple();
  const haptics = useHapticFeedback();
  const isInteractive = variant === 'interactive' || !!onClick;

  const handleClick = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      if (!isInteractive) return;
      if (!noRipple) createRipple(e);
      if (!noHaptic) haptics.light();
      onClick?.(e);
    },
    [isInteractive, noRipple, noHaptic, createRipple, haptics, onClick],
  );

  return (
    <div
      className={`
        rounded-xl
        transition-normal
        ${variantClasses[variant]}
        ${paddingClasses[padding]}
        ${animate ? 'animate-scaleIn' : ''}
        ${isInteractive ? 'ripple-container' : ''}
        ${className}
      `}
      onClick={handleClick}
      role={isInteractive ? role || 'button' : role}
      aria-label={isInteractive ? ariaLabel : undefined}
      tabIndex={isInteractive ? 0 : undefined}
      onKeyDown={
        isInteractive
          ? (e: React.KeyboardEvent) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                handleClick(e as unknown as React.MouseEvent<HTMLDivElement>);
              }
            }
          : undefined
      }
    >
      {children}
      {ripples}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Card sub-components
// ═══════════════════════════════════════════════════════════════════════

export function CardHeader({
  children,
  action,
  className = '',
}: {
  children: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={`flex items-center justify-between mb-4 ${className}`}>
      <div className="flex-1 min-w-0">{children}</div>
      {action && <div className="flex-shrink-0 ml-2">{action}</div>}
    </div>
  );
}

export function CardTitle({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <h3 className={`text-lg font-semibold text-slate-900 dark:text-white ${className}`}>
      {children}
    </h3>
  );
}

export function CardContent({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={`text-slate-600 dark:text-slate-300 ${className}`}>{children}</div>;
}

export function CardFooter({
  children,
  className = '',
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`mt-4 pt-4 border-t border-slate-200 dark:border-slate-700 flex items-center gap-3 ${className}`}
    >
      {children}
    </div>
  );
}
