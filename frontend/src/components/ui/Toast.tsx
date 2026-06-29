// ═══════════════════════════════════════════════════════════════════════
// Toast Notifications — UX-14.2.5 Redesign
//
// Features:
//   - Stacked layout (bottom-right corner)
//   - Undo action for destructive operations
//   - Grouping identical toasts with counter badge
//   - Progress bar countdown
//   - Icons per variant (CheckCircle, XCircle, AlertTriangle, Info)
//   - Slide-in / fade-out animations
//   - Max 5 visible, "Show N more" toggle
//   - Dark theme via TailwindCSS v4 `dark:` classes
//   - z-[200] — above all modals
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useState, useCallback } from 'react';
import {
  CheckCircle,
  XCircle,
  AlertTriangle,
  Info,
  X,
} from './Icons';
import {
  useAlertStore,
  type ToastAlert,
  MAX_VISIBLE_TOASTS,
} from '../../store/alertStore';

// ─── Variant configuration ──────────────────────────────────────────

const VARIANT_STYLES = {
  success: {
    bg: 'bg-emerald-50 dark:bg-emerald-900/40',
    border: 'border-emerald-200 dark:border-emerald-800',
    text: 'text-emerald-800 dark:text-emerald-200',
    icon: 'text-emerald-500 dark:text-emerald-400',
    bar: 'bg-emerald-500 dark:bg-emerald-400',
  },
  error: {
    bg: 'bg-red-50 dark:bg-red-900/40',
    border: 'border-red-200 dark:border-red-800',
    text: 'text-red-800 dark:text-red-200',
    icon: 'text-red-500 dark:text-red-400',
    bar: 'bg-red-500 dark:bg-red-400',
  },
  warning: {
    bg: 'bg-amber-50 dark:bg-amber-900/40',
    border: 'border-amber-200 dark:border-amber-800',
    text: 'text-amber-800 dark:text-amber-200',
    icon: 'text-amber-500 dark:text-amber-400',
    bar: 'bg-amber-500 dark:bg-amber-400',
  },
  info: {
    bg: 'bg-blue-50 dark:bg-blue-900/40',
    border: 'border-blue-200 dark:border-blue-800',
    text: 'text-blue-800 dark:text-blue-200',
    icon: 'text-blue-500 dark:text-blue-400',
    bar: 'bg-blue-500 dark:bg-blue-400',
  },
} as const;

const ICON_MAP = {
  success: CheckCircle,
  error: XCircle,
  warning: AlertTriangle,
  info: Info,
} as const;

// ─── Individual Toast Notification ──────────────────────────────────

function ToastNotification({
  toast,
  onDismiss,
}: {
  toast: ToastAlert;
  onDismiss: (id: string) => void;
}) {
  const [isVisible, setIsVisible] = useState(false);
  const [isExiting, setIsExiting] = useState(false);
  const styles = VARIANT_STYLES[toast.type];
  const Icon = ICON_MAP[toast.type];
  const duration = toast.duration ?? 5000;

  useEffect(() => {
    // Enter animation on mount
    requestAnimationFrame(() => setIsVisible(true));

    // Auto-dismiss after duration
    const dismissTimeout = setTimeout(() => {
      setIsExiting(true);
      setTimeout(() => onDismiss(toast.id), 300); // wait for exit animation
    }, duration);

    return () => clearTimeout(dismissTimeout);
  }, [toast.id, duration, onDismiss]);

  const handleDismiss = useCallback(() => {
    setIsExiting(true);
    setTimeout(() => onDismiss(toast.id), 300);
  }, [toast.id, onDismiss]);

  const handleUndo = useCallback(() => {
    toast.undo?.onClick();
    handleDismiss();
  }, [toast.undo, handleDismiss]);

  return (
    <div
      className={`
        relative flex items-start gap-3 px-4 py-3 pb-3 rounded-xl border shadow-lg
        backdrop-blur-sm overflow-hidden
        transition-all duration-300 ease-out
        w-full max-w-[400px]
        ${styles.bg} ${styles.border}
        ${
          isVisible && !isExiting
            ? 'opacity-100 translate-x-0 scale-100'
            : 'opacity-0 translate-x-8 scale-95'
        }
        ${toast.collapsed ? 'opacity-70 scale-[0.97]' : ''}
      `}
      role={toast.type === 'error' ? 'alert' : 'status'}
      aria-live={toast.type === 'error' ? 'assertive' : 'polite'}
      aria-atomic="true"
    >
      {/* Progress bar — animated countdown */}
      <div
        className={`absolute bottom-0 left-0 h-0.5 rounded-full ${styles.bar}`}
        style={{
          animation: `toast-progress ${duration}ms linear forwards`,
        }}
      />

      {/* Icon */}
      <Icon className={`w-5 h-5 flex-shrink-0 mt-0.5 ${styles.icon}`} />

      {/* Content area */}
      <div className="flex-1 min-w-0">
        {/* Title row + group badge */}
        <div className="flex items-start justify-between gap-2">
          <p className={`text-sm font-semibold ${styles.text} truncate`}>
            {toast.title}
          </p>

          {/* Grouping counter badge */}
          {toast.count !== undefined && toast.count > 1 && (
            <span
              className={`
                flex-shrink-0 inline-flex items-center justify-center
                min-w-[20px] h-5 px-1.5 rounded-full
                text-[11px] font-bold leading-none
                ${styles.bg} ${styles.text} border ${styles.border}
              `}
              aria-label={`${toast.count} occurrences`}
            >
              {toast.count}
            </span>
          )}
        </div>

        {/* Collapsed: show minimal info with "N×" prefix on title */}
        {toast.collapsed ? (
          <p className={`text-xs mt-0.5 ${styles.text} opacity-70 line-clamp-1`}>
            {toast.count}× {toast.message || 'Repeated notification'}
          </p>
        ) : (
          <>
            {/* Optional message */}
            {toast.message && (
              <p
                className={`text-xs mt-0.5 ${styles.text} opacity-80 line-clamp-2`}
              >
                {toast.message}
              </p>
            )}

            {/* Undo action button */}
            {toast.undo && (
              <button
                onClick={handleUndo}
                className={`
                  mt-1.5 text-xs font-semibold uppercase tracking-wider
                  px-2.5 py-1 rounded-lg
                  hover:bg-black/10 dark:hover:bg-white/10
                  transition-colors ${styles.text}
                `}
              >
                {toast.undo.label}
              </button>
            )}
          </>
        )}
      </div>

      {/* Manual dismiss button */}
      <button
        onClick={handleDismiss}
        className={`
          flex-shrink-0 p-1 rounded-lg
          hover:bg-black/10 dark:hover:bg-white/10
          transition-colors ${styles.text} opacity-60 hover:opacity-100
        `}
        aria-label="Dismiss notification"
      >
        <X className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}

// ─── Toast Provider ─────────────────────────────────────────────────

const MemoizedToast = React.memo(ToastNotification);

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const toasts = useAlertStore((s) => s.toasts);
  const removeToast = useAlertStore((s) => s.removeToast);
  const showMoreToasts = useAlertStore((s) => s.showMoreToasts);
  const toggleShowMoreToasts = useAlertStore((s) => s.toggleShowMoreToasts);

  const visibleToasts = showMoreToasts
    ? toasts
    : toasts.slice(0, MAX_VISIBLE_TOASTS);

  const hiddenCount = toasts.length - MAX_VISIBLE_TOASTS;

  return (
    <>
      {children}

      {/* Toast container — fixed bottom-right, above all modals */}
      <div
        className="fixed bottom-4 right-4 z-[200] flex flex-col-reverse gap-2 pointer-events-none w-[calc(100%-2rem)] sm:w-auto sm:max-w-[400px]"
        aria-live="polite"
        aria-label="Notifications"
      >
        {visibleToasts.map((toast) => (
          <div key={toast.id} className="pointer-events-auto w-full">
            <MemoizedToast toast={toast} onDismiss={removeToast} />
          </div>
        ))}

        {/* Overflow indicator — "Show N more" */}
        {!showMoreToasts && hiddenCount > 0 && (
          <button
            onClick={toggleShowMoreToasts}
            className="pointer-events-auto text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors text-center py-1 bg-white/80 dark:bg-gray-900/80 rounded-lg backdrop-blur-sm"
          >
            +{hiddenCount} more
          </button>
        )}

        {showMoreToasts && hiddenCount > 0 && (
          <button
            onClick={toggleShowMoreToasts}
            className="pointer-events-auto text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 transition-colors text-center py-1"
          >
            Show less
          </button>
        )}
      </div>
    </>
  );
}

// ─── useToast Hook (backward-compatible) ────────────────────────────

export type ToastOptions =
  | string
  | {
      title: string;
      message?: string;
      undo?: { label: string; onClick: () => void };
      duration?: number;
    };

export function useToast() {
  const addToast = useAlertStore((s) => s.addToast);

  const show = useCallback(
    (type: ToastAlert['type'], options: ToastOptions) => {
      if (typeof options === 'string') {
        return addToast({ type, title: options });
      }
      return addToast({ type, ...options });
    },
    [addToast],
  );

  return {
    success: useCallback(
      (options: ToastOptions) => show('success', options),
      [show],
    ),
    error: useCallback(
      (options: ToastOptions) => show('error', options),
      [show],
    ),
    warning: useCallback(
      (options: ToastOptions) => show('warning', options),
      [show],
    ),
    info: useCallback(
      (options: ToastOptions) => show('info', options),
      [show],
    ),
  };
}
