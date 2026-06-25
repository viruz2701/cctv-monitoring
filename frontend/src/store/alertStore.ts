// ═══════════════════════════════════════════════════════════════════════
// Alert Store (Zustand)
// ARCH-02: UI-состояние для алертов (не server state).
// Server state (alarms из API) получаем через useAlarms() из React Query.
//
// Этот store управляет только UI-состоянием алертов:
//   - Временные всплывающие уведомления (toast alerts)
//   - Фильтры и сортировка списка алертов
//   - Выделенные алерты (bulk actions)
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

export interface ToastAlert {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number; // ms, default 5000
  action?: {
    label: string;
    onClick: () => void;
  };
}

interface AlertUIState {
  // Toast alerts (всплывающие уведомления)
  toasts: ToastAlert[];
  addToast: (alert: Omit<ToastAlert, 'id'>) => string;
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

let toastCounter = 0;

export const useAlertStore = create<AlertUIState>()((set) => ({
  // ── Toast Alerts ──────────────────────────────────────────────────
  toasts: [],
  addToast: (alert) => {
    const id = `toast-${++toastCounter}-${Date.now()}`;
    const duration = alert.duration ?? 5000;
    set((state) => ({
      toasts: [...state.toasts, { ...alert, id }],
    }));
    // Auto-remove после таймаута
    if (duration > 0) {
      setTimeout(() => {
        set((state) => ({
          toasts: state.toasts.filter((t) => t.id !== id),
        }));
      }, duration);
    }
    return id;
  },
  removeToast: (id) =>
    set((state) => ({
      toasts: state.toasts.filter((t) => t.id !== id),
    })),
  clearToasts: () => set({ toasts: [] }),

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

// Selector hooks для оптимальных re-render'ов
export const useToastAlerts = () => useAlertStore((s) => s.toasts);
export const useAlertFilters = () =>
  useAlertStore((s) => ({
    status: s.alertFilterStatus,
    priority: s.alertFilterPriority,
  }));
export const useSelectedAlertIds = () => useAlertStore((s) => s.selectedAlertIds);
