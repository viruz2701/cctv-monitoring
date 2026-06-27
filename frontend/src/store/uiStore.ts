// ═══════════════════════════════════════════════════════════════════════
// UI Store (Zustand)
// ARCH.1: UI-состояние (sidebar, модалки, панели).
// Не включает server state — только client-side UI.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

// ─── Types ──────────────────────────────────────────────────────────

export interface ModalConfig {
  isOpen: boolean;
  type?: string;
  data?: Record<string, unknown>;
}

export interface PanelState {
  isOpen: boolean;
  width?: number;
}

export interface UIState {
  // Sidebar
  sidebarOpen: boolean;
  sidebarPinned: boolean;

  // Command palette
  commandPaletteOpen: boolean;

  // Keyboard shortcuts help
  shortcutsHelpOpen: boolean;

  // Active modal
  modal: ModalConfig;

  // Right panel (inspector, details)
  rightPanel: PanelState;

  // Bottom panel (logs, terminal)
  bottomPanel: PanelState;

  // Bulk action mode
  bulkMode: boolean;
  selectedIds: string[];

  // Actions
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setSidebarPinned: (pinned: boolean) => void;
  setCommandPaletteOpen: (open: boolean) => void;
  setShortcutsHelpOpen: (open: boolean) => void;
  openModal: (type: string, data?: Record<string, unknown>) => void;
  closeModal: () => void;
  setRightPanel: (panel: Partial<PanelState>) => void;
  setBottomPanel: (panel: Partial<PanelState>) => void;
  setBulkMode: (active: boolean) => void;
  setSelectedIds: (ids: string[]) => void;
  toggleSelectedId: (id: string) => void;
  clearSelection: () => void;
}

// ─── Store ──────────────────────────────────────────────────────────

export const useUIStore = create<UIState>()((set, get) => ({
  // Initial state
  sidebarOpen: true,
  sidebarPinned: true,
  commandPaletteOpen: false,
  shortcutsHelpOpen: false,
  modal: { isOpen: false },
  rightPanel: { isOpen: false, width: 400 },
  bottomPanel: { isOpen: false, width: 300 },
  bulkMode: false,
  selectedIds: [],

  // Actions
  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),

  setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),

  setSidebarPinned: (sidebarPinned) => set({ sidebarPinned }),

  setCommandPaletteOpen: (commandPaletteOpen) => set({ commandPaletteOpen }),

  setShortcutsHelpOpen: (shortcutsHelpOpen) => set({ shortcutsHelpOpen }),

  openModal: (type, data) => set({ modal: { isOpen: true, type, data } }),

  closeModal: () => set({ modal: { isOpen: false } }),

  setRightPanel: (panel) =>
    set((s) => ({ rightPanel: { ...s.rightPanel, ...panel } })),

  setBottomPanel: (panel) =>
    set((s) => ({ bottomPanel: { ...s.bottomPanel, ...panel } })),

  setBulkMode: (bulkMode) => set({ bulkMode, selectedIds: bulkMode ? get().selectedIds : [] }),

  setSelectedIds: (selectedIds) => set({ selectedIds }),

  toggleSelectedId: (id) =>
    set((s) => ({
      selectedIds: s.selectedIds.includes(id)
        ? s.selectedIds.filter((i) => i !== id)
        : [...s.selectedIds, id],
    })),

  clearSelection: () => set({ selectedIds: [], bulkMode: false }),
}));
