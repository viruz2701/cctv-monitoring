// ═══════════════════════════════════════════════════════════════════════
// Empty State Component
// UX-14.1.7: Иллюстративные empty states с CTA для всех списков
//
// Использование:
//   <EmptyState
//     icon={<HardDrive className="w-12 h-12" />}
//     title="No devices found"
//     description="Add your first CCTV device to start monitoring"
//     action={{ label: "Add Device", onClick: handleAdd }}
//     secondaryAction={{ label: "Learn more", onClick: handleLearn }}
//   />
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Plus } from 'lucide-react';

interface EmptyStateAction {
  label: string;
  onClick: () => void;
}

interface EmptyStateProps {
  /** Основная иконка (lucide-react или кастомная) */
  icon: React.ReactNode;
  /** Заголовок */
  title: string;
  /** Описание (опционально) */
  description?: string;
  /** Дополнительный контекст/хинт */
  hint?: string;
  /** Основной CTA */
  action?: EmptyStateAction;
  /** Вторичный CTA */
  secondaryAction?: EmptyStateAction;
  /** Размер: sm — для карточек, md — для страниц, lg — для целых секций */
  size?: 'sm' | 'md' | 'lg';
}

const sizeConfig = {
  sm: {
    wrapper: 'py-8',
    icon: 'w-8 h-8',
    iconContainer: 'w-12 h-12',
    title: 'text-sm',
    description: 'text-xs',
    hint: 'text-[11px]',
    button: 'text-xs px-3 py-1.5',
  },
  md: {
    wrapper: 'py-12',
    icon: 'w-12 h-12',
    iconContainer: 'w-16 h-16',
    title: 'text-base',
    description: 'text-sm',
    hint: 'text-xs',
    button: 'text-sm px-4 py-2',
  },
  lg: {
    wrapper: 'py-20',
    icon: 'w-16 h-16',
    iconContainer: 'w-24 h-24',
    title: 'text-xl',
    description: 'text-base',
    hint: 'text-sm',
    button: 'text-sm px-5 py-2.5',
  },
};

export function EmptyState({
  icon,
  title,
  description,
  hint,
  action,
  secondaryAction,
  size = 'md',
}: EmptyStateProps) {
  const cfg = sizeConfig[size];

  return (
    <div
      className={`flex flex-col items-center justify-center text-center ${cfg.wrapper} px-6`}
    >
      {/* Icon container */}
      <div
        className={`flex items-center justify-center ${cfg.iconContainer} rounded-2xl bg-slate-100 dark:bg-slate-800/80 text-slate-300 dark:text-slate-500 mb-4 ring-1 ring-slate-200 dark:ring-slate-700/50`}
      >
        <div className={cfg.icon}>{icon}</div>
      </div>

      {/* Title */}
      <h3
        className={`font-semibold text-slate-900 dark:text-white ${cfg.title}`}
      >
        {title}
      </h3>

      {/* Description */}
      {description && (
        <p
          className={`text-slate-500 dark:text-slate-400 mt-1.5 max-w-sm ${cfg.description}`}
        >
          {description}
        </p>
      )}

      {/* Hint */}
      {hint && (
        <p
          className={`text-slate-400 dark:text-slate-500 mt-1 ${cfg.hint}`}
        >
          {hint}
        </p>
      )}

      {/* Actions */}
      {(action || secondaryAction) && (
        <div className="flex items-center gap-3 mt-6">
          {action && (
            <button
              onClick={action.onClick}
              className={`inline-flex items-center gap-1.5 font-medium text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 rounded-lg transition-colors shadow-sm ${cfg.button}`}
            >
              <Plus className={`${size === 'sm' ? 'w-3.5 h-3.5' : 'w-4 h-4'}`} />
              {action.label}
            </button>
          )}
          {secondaryAction && (
            <button
              onClick={secondaryAction.onClick}
              className={`inline-flex items-center gap-1.5 font-medium text-slate-600 dark:text-slate-400 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 rounded-lg transition-colors ${cfg.button}`}
            >
              {secondaryAction.label}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
