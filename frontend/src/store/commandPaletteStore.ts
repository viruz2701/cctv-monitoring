// ═══════════════════════════════════════════════════════════════════════
// Command Palette Store (Zustand)
// UX-14.1.5: Command Palette (⌘K) — UI-состояние палитры команд
// UX-14.2.6: Recent commands (последние 5), localStorage persistence
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

const RECENT_MAX = 5;
const STORAGE_KEY = 'cctv:recent-commands';

function loadRecent(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.slice(0, RECENT_MAX).filter((s): s is string => typeof s === 'string');
  } catch {
    return [];
  }
}

function saveRecent(ids: string[]): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
  } catch {
    // localStorage not available (SSR / restricted)
  }
}

interface CommandPaletteState {
  isOpen: boolean;
  recentCommands: string[];
  open: () => void;
  close: () => void;
  toggle: () => void;
  /** Add command ID to recent (front, dedup, max 5) */
  addRecent: (commandId: string) => void;
  /** Clear all recent commands */
  clearRecent: () => void;
}

export const useCommandPaletteStore = create<CommandPaletteState>()((set) => ({
  isOpen: false,
  recentCommands: loadRecent(),

  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((state) => ({ isOpen: !state.isOpen })),

  addRecent: (commandId: string) =>
    set((state) => {
      const filtered = state.recentCommands.filter((id) => id !== commandId);
      const updated = [commandId, ...filtered].slice(0, RECENT_MAX);
      saveRecent(updated);
      return { recentCommands: updated };
    }),

  clearRecent: () => {
    saveRecent([]);
    return { recentCommands: [] };
  },
}));
