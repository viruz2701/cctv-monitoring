import React from 'react';

// ═══════════════════════════════════════════════════════════════════════
// ProgressBar Component
// Generic progress bar with variant, size, animation and label support.
// TailwindCSS v4 + dark mode.
// ═══════════════════════════════════════════════════════════════════════

type ProgressVariant = 'success' | 'warning' | 'danger' | 'info';
type ProgressSize = 'sm' | 'md' | 'lg';

interface ProgressBarProps {
  /** Current value (0 to max). Default: 0 */
  value?: number;
  /** Maximum value. Default: 100 */
  max?: number;
  /** Color variant. Default: 'info' */
  variant?: ProgressVariant;
  /** Bar size. Default: 'md' */
  size?: ProgressSize;
  /** Show percentage label on the right */
  showLabel?: boolean;
  /** Enable animated stripes effect */
  animated?: boolean;
  /** Additional CSS classes */
  className?: string;
}

const variantBar: Record<ProgressVariant, string> = {
  success: 'bg-emerald-500 dark:bg-emerald-400',
  warning: 'bg-amber-500 dark:bg-amber-400',
  danger: 'bg-red-500 dark:bg-red-400',
  info: 'bg-blue-500 dark:bg-blue-400',
};

const variantTrack: Record<ProgressVariant, string> = {
  success: 'bg-emerald-100 dark:bg-emerald-900/30',
  warning: 'bg-amber-100 dark:bg-amber-900/30',
  danger: 'bg-red-100 dark:bg-red-900/30',
  info: 'bg-blue-100 dark:bg-blue-900/30',
};

const sizeTrack: Record<ProgressSize, string> = {
  sm: 'h-1.5',
  md: 'h-2.5',
  lg: 'h-4',
};

const sizeLabel: Record<ProgressSize, string> = {
  sm: 'text-[10px]',
  md: 'text-xs',
  lg: 'text-sm',
};

export function ProgressBar({
  value = 0,
  max = 100,
  variant = 'info',
  size = 'md',
  showLabel = false,
  animated = false,
  className = '',
}: ProgressBarProps) {
  const pct = max > 0 ? Math.min(100, Math.max(0, (value / max) * 100)) : 0;

  return (
    <div className={`flex items-center gap-2 ${className}`} role="progressbar" aria-valuenow={value} aria-valuemin={0} aria-valuemax={max} aria-label={`Progress: ${Math.round(pct)}%`}>
      <div className={`flex-1 rounded-full overflow-hidden ${variantTrack[variant]} ${sizeTrack[size]}`}>
        <div
          className={`h-full rounded-full transition-all duration-500 ease-out ${variantBar[variant]} ${
            animated ? 'bg-[length:200%_100%] animate-[shimmer_2s_linear_infinite] bg-gradient-to-r from-transparent via-white/25 to-transparent' : ''
          }`}
          style={{ width: `${pct}%` }}
        />
      </div>
      {showLabel && (
        <span className={`font-medium text-slate-600 dark:text-slate-400 tabular-nums ${sizeLabel[size]}`}>
          {Math.round(pct)}%
        </span>
      )}
    </div>
  );
}
