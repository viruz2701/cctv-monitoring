// ═══════════════════════════════════════════════════════════════════════
// Alert Store (Zustand)
// ARCH-02: UI-состояние для алертов (не server state).
// Server state (alarms из API) получаем через useAlarms() из React Query.
//
// Этот store управляет только UI-состоянием алертов:
//   - Временные всплывающие уведомления (toast alerts)
//   - Фильтры и сортировка списка алертов
//   - Выделенные алерты (bulk actions)
//
// UX-14.2.5: Toast Redesign
//   - undo-action, grouping, stacked layout, progress bar
//   - Default durations: success=3s, error=8s, warning=5s, info=4s
//   - Max 5 visible, grouping identical toasts with counter
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

// ─── Types ──────────────────────────────────────────────────────────

export interface ToastUndo {
  label: string;
  onClick: () => void;
}

export interface ToastAlert {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  /** Действие для отмены (destructive actions) */
  undo?: ToastUndo;
  /** Количество сгруппированных одинаковых toast */
  count?: number;
  /** Кастомная длительность в ms. Если не указана — используется default по типу */
  duration?: number;
  /** Флаг: toast свёрнут после 3+ одинаковых */
  collapsed?: boolean;
}

// ─── Default durations ──────────────────────────────────────────────

export const TOAST_DURATIONS: Record<ToastAlert['type'], number> = {
  success: 3000,
  error: 8000,
  warning: 5000,
  info: 4000,
};

export const MAX_VISIBLE_TOASTS = 5;

// ─── Store interface ────────────────────────────────────────────────

interface AlertUIState {
  // Toast alerts (всплывающие уведомления)
  toasts: ToastAlert[];
  showMoreToasts: boolean;
  toggleShowMoreToasts: () => void;
  addToast: (alert: Omit<ToastAlert, 'id' | 'count'>) => string;
  removeToast: (id: string) => void;
  clearToasts: () => void;

  // Alert list filters (UI state)
  alertFilterStatus: string | null;
  alertFilterPriority: string | null;
  setAlertFilterStatus: (status: string | null) => void;
  setAlertFilterPriority: (priority: string | null) => void;

  // Selected alerts for bulk actions
  selectedAlertIds: string[];
  toggleAlertSelection: (id: string) => void;
  clearAlertSelection: () => void;
  selectAllAlerts: (ids: string[]) => void;
}

// ─── Counter for unique IDs ─────────────────────────────────────────

let toastCounter = 0;

// ─── Store ──────────────────────────────────────────────────────────

export const useAlertStore = create<AlertUIState>()((set) => ({
  // ── Toast Alerts ──────────────────────────────────────────────────
  toasts: [],
  showMoreToasts: false,

  toggleShowMoreToasts: () =>
    set((state) => ({ showMoreToasts: !state.showMoreToasts })),

  addToast: (alert) => {
    const id = `toast-${++toastCounter}-${Date.now()}`;
    const duration = alert.duration ?? TOAST_DURATIONS[alert.type];

    set((state) => {
      // Проверяем, есть ли уже toast с таким же title + message
      const existingIndex = state.toasts.findIndex(
        (t) =>
          t.type === alert.type &&
          t.title === alert.title &&
          t.message === alert.message
      );

      if (existingIndex !== -1) {
        // Группировка: увеличиваем count и обновляем id (сбрасываем таймер)
        const existing = state.toasts[existingIndex];
        const newCount = (existing.count ?? 1) + 1;
        const updated = [...state.toasts];
        updated[existingIndex] = {
          ...existing,
          count: newCount,
          id,
          duration,
          undo: alert.undo ?? existing.undo,
          // После 3 одинаковых — collapse (сворачиваем)
          collapsed: newCount >= 3 || existing.collapsed === true,
        };
        return { toasts: updated };
      }

      // Новый toast
      return {
        toasts: [...state.toasts, { ...alert, id, count: 1, duration }],
      };
    });

    return id;
  },

  removeToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),

  clearToasts: () => set({ toasts: [], showMoreToasts: false }),

  // ── Alert List Filters ────────────────────────────────────────────
  alertFilterStatus: null,
  alertFilterPriority: null,
  setAlertFilterStatus: (status) => set({ alertFilterStatus: status }),
  setAlertFilterPriority: (priority) => set({ alertFilterPriority: priority }),

  // ── Selected Alerts ───────────────────────────────────────────────
  selectedAlertIds: [],
  toggleAlertSelection: (id) =>
    set((state) => ({
      selectedAlertIds: state.selectedAlertIds.includes(id)
        ? state.selectedAlertIds.filter((i) => i !== id)
        : [...state.selectedAlertIds, id],
    })),
  clearAlertSelection: () => set({ selectedAlertIds: [] }),
  selectAllAlerts: (ids) => set({ selectedAlertIds: ids }),
}));

// ─── Selector hooks для оптимальных re-render'ов ────────────────────

export const useToastAlerts = () => useAlertStore((s) => s.toasts);
export const useToastShowMore = () => useAlertStore((s) => s.showMoreToasts);
export const useAlertFilters = () =>
  useAlertStore((s) => ({
    status: s.alertFilterStatus,
    priority: s.alertFilterPriority,
  }));
export const useSelectedAlertIds = () => useAlertStore((s) => s.selectedAlertIds);
