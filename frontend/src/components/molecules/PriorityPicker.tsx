import React, { useState, useCallback } from 'react';

// ═══════════════════════════════════════════════════════════════════════
// PriorityPicker — visual priority selector
// Critical 🔴 / High 🟠 / Medium 🟡 / Low 🟢
// Keyboard accessible, dark mode support.
// ═══════════════════════════════════════════════════════════════════════

export type Priority = 'critical' | 'high' | 'medium' | 'low';

export interface PriorityOption {
  value: Priority;
  label: string;
  labelRu: string;
  color: string;
  dotColor: string;
  bgColor: string;
  ringColor: string;
}

export const PRIORITY_OPTIONS: PriorityOption[] = [
  {
    value: 'critical',
    label: 'Critical',
    labelRu: 'Критический',
    color: 'text-red-700 dark:text-red-300',
    dotColor: 'bg-red-500',
    bgColor: 'bg-red-50 dark:bg-red-950/30',
    ringColor: 'ring-red-500',
  },
  {
    value: 'high',
    label: 'High',
    labelRu: 'Высокий',
    color: 'text-orange-700 dark:text-orange-300',
    dotColor: 'bg-orange-500',
    bgColor: 'bg-orange-50 dark:bg-orange-950/30',
    ringColor: 'ring-orange-500',
  },
  {
    value: 'medium',
    label: 'Medium',
    labelRu: 'Средний',
    color: 'text-yellow-700 dark:text-yellow-300',
    dotColor: 'bg-yellow-500',
    bgColor: 'bg-yellow-50 dark:bg-yellow-950/30',
    ringColor: 'ring-yellow-500',
  },
  {
    value: 'low',
    label: 'Low',
    labelRu: 'Низкий',
    color: 'text-emerald-700 dark:text-emerald-300',
    dotColor: 'bg-emerald-500',
    bgColor: 'bg-emerald-50 dark:bg-emerald-950/30',
    ringColor: 'ring-emerald-500',
  },
];

interface PriorityPickerProps {
  /** Selected priority */
  value?: Priority;
  /** Change handler */
  onChange: (priority: Priority) => void;
  /** Read-only mode */
  readOnly?: boolean;
  /** Show labels next to dots */
  showLabels?: boolean;
  /** Use Russian labels */
  lang?: 'ru' | 'en';
  className?: string;
}

export function PriorityPicker({
  value,
  onChange,
  readOnly = false,
  showLabels = true,
  lang = 'ru',
  className = '',
}: PriorityPickerProps) {
  const [focusedIdx, setFocusedIdx] = useState(-1);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (readOnly) return;

      const currentIdx = value
        ? PRIORITY_OPTIONS.findIndex((o) => o.value === value)
        : -1;

      switch (e.key) {
        case 'ArrowRight':
        case 'ArrowDown': {
          e.preventDefault();
          const next = currentIdx < PRIORITY_OPTIONS.length - 1 ? currentIdx + 1 : 0;
          onChange(PRIORITY_OPTIONS[next].value);
          setFocusedIdx(next);
          break;
        }
        case 'ArrowLeft':
        case 'ArrowUp': {
          e.preventDefault();
          const prev = currentIdx > 0 ? currentIdx - 1 : PRIORITY_OPTIONS.length - 1;
          onChange(PRIORITY_OPTIONS[prev].value);
          setFocusedIdx(prev);
          break;
        }
        case ' ':
        case 'Enter':
          e.preventDefault();
          if (focusedIdx >= 0) {
            onChange(PRIORITY_OPTIONS[focusedIdx].value);
          }
          break;
      }
    },
    [value, onChange, readOnly, focusedIdx],
  );

  return (
    <div
      className={`inline-flex items-center gap-1 ${readOnly ? '' : 'focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2 dark:focus-within:ring-offset-slate-900 rounded-lg'} ${className}`}
      role="radiogroup"
      aria-label="Priority"
      tabIndex={readOnly ? -1 : 0}
      onKeyDown={handleKeyDown}
    >
      {PRIORITY_OPTIONS.map((opt) => {
        const isSelected = value === opt.value;
        return (
          <button
            key={opt.value}
            type="button"
            role="radio"
            aria-checked={isSelected}
            aria-label={lang === 'ru' ? opt.labelRu : opt.label}
            disabled={readOnly}
            onClick={() => !readOnly && onChange(opt.value)}
            className={`
              flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-sm font-medium
              transition-all duration-150
              ${readOnly ? 'cursor-default' : 'cursor-pointer'}
              ${isSelected ? `${opt.bgColor} ${opt.color} ring-2 ${opt.ringColor}` : 'text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'}
            `}
          >
            <span className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${opt.dotColor} ${isSelected ? 'shadow-sm' : 'opacity-50'}`} />
            {showLabels && <span>{lang === 'ru' ? opt.labelRu : opt.label}</span>}
          </button>
        );
      })}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// PriorityBadge — компактный badge для отображения приоритета
// ═══════════════════════════════════════════════════════════════════════

interface PriorityBadgeProps {
  priority: Priority;
  lang?: 'ru' | 'en';
  size?: 'sm' | 'md';
}

const badgeSize = {
  sm: 'text-[10px] px-1.5 py-0.5 gap-1',
  md: 'text-xs px-2 py-0.5 gap-1.5',
};

const badgeDot = {
  sm: 'w-1.5 h-1.5',
  md: 'w-2 h-2',
};

export function PriorityBadge({ priority, lang = 'ru', size = 'md' }: PriorityBadgeProps) {
  const opt = PRIORITY_OPTIONS.find((o) => o.value === priority);
  if (!opt) return null;

  return (
    <span
      className={`inline-flex items-center font-medium rounded-full ${opt.bgColor} ${opt.color} ${badgeSize[size]}`}
    >
      <span className={`rounded-full ${opt.dotColor} ${badgeDot[size]}`} />
      {lang === 'ru' ? opt.labelRu : opt.label}
    </span>
  );
}
