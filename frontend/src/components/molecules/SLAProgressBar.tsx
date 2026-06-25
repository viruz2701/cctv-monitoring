import React, { useMemo } from 'react';
import { Clock, AlertTriangle } from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// SLAProgressBar — цветовая индикация прогресса SLA
// Показывает elapsed / remaining / total time
// ═══════════════════════════════════════════════════════════════════════

type SLAVariant = 'success' | 'warning' | 'danger';

interface SLAProgressBarProps {
  /** Elapsed time in minutes */
  elapsedMinutes: number;
  /** Total SLA time in minutes */
  totalMinutes: number;
  /** Optional label override */
  label?: string;
  /** Compact mode (smaller) */
  compact?: boolean;
  className?: string;
}

function formatDuration(minutes: number): string {
  if (minutes <= 0) return '0м';
  const h = Math.floor(minutes / 60);
  const m = Math.round(minutes % 60);
  if (h === 0) return `${m}м`;
  if (m === 0) return `${h}ч`;
  return `${h}ч ${m}м`;
}

function getVariant(pct: number): SLAVariant {
  if (pct >= 90) return 'danger';
  if (pct >= 75) return 'warning';
  return 'success';
}

const variantBar: Record<SLAVariant, string> = {
  success: 'bg-emerald-500 dark:bg-emerald-400',
  warning: 'bg-amber-500 dark:bg-amber-400',
  danger: 'bg-red-500 dark:bg-red-400',
};

const variantTrack: Record<SLAVariant, string> = {
  success: 'bg-emerald-100 dark:bg-emerald-900/30',
  warning: 'bg-amber-100 dark:bg-amber-900/30',
  danger: 'bg-red-100 dark:bg-red-900/30',
};

const variantText: Record<SLAVariant, string> = {
  success: 'text-emerald-700 dark:text-emerald-400',
  warning: 'text-amber-700 dark:text-amber-400',
  danger: 'text-red-700 dark:text-red-400',
};

export function SLAProgressBar({
  elapsedMinutes,
  totalMinutes,
  label,
  compact = false,
  className = '',
}: SLAProgressBarProps) {
  const pct = totalMinutes > 0
    ? Math.min(100, Math.max(0, (elapsedMinutes / totalMinutes) * 100))
    : 0;

  const remaining = totalMinutes - elapsedMinutes;
  const isOverdue = remaining <= 0;
  const variant = getVariant(pct);
  const statusLabel = isOverdue
    ? `Просрочен на ${formatDuration(Math.abs(remaining))}`
    : `${formatDuration(remaining)} осталось`;

  return (
    <div className={className}>
      {/* Bar */}
      <div
        className={`w-full rounded-full overflow-hidden ${variantTrack[variant]} ${compact ? 'h-2' : 'h-2.5'}`}
        role="progressbar"
        aria-valuenow={elapsedMinutes}
        aria-valuemin={0}
        aria-valuemax={totalMinutes}
        aria-label={`SLA: ${Math.round(pct)}%`}
      >
        <div
          className={`h-full rounded-full transition-all duration-700 ease-out ${variantBar[variant]}`}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>

      {/* Label row */}
      <div className={`flex items-center justify-between mt-1.5 ${compact ? 'text-xs' : 'text-sm'}`}>
        <div className="flex items-center gap-1.5">
          {isOverdue ? (
            <AlertTriangle size={compact ? 12 : 14} className={variantText[variant]} />
          ) : (
            <Clock size={compact ? 12 : 14} className="text-slate-400 dark:text-slate-500" />
          )}
          <span className="text-slate-600 dark:text-slate-400">
            {label ?? `${formatDuration(elapsedMinutes)} / ${formatDuration(totalMinutes)}`}
          </span>
        </div>
        <span className={`font-medium tabular-nums ${variantText[variant]}`}>
          {statusLabel}
        </span>
      </div>
    </div>
  );
}
