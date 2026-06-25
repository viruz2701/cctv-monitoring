// ═══════════════════════════════════════════════════════════════════════
// Command Palette Store (Zustand)
// UX-14.1.5: Command Palette (⌘K) — UI-состояние палитры команд
//
// Этот store управляет только открытием/закрытием Command Palette.
// Сами команды и навигация — в компоненте CommandPalette.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

interface CommandPaletteState {
  isOpen: boolean;
  open: () => void;
  close: () => void;
  toggle: () => void;
}

export const useCommandPaletteStore = create<CommandPaletteState>()((set) => ({
  isOpen: false,
  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((state) => ({ isOpen: !state.isOpen })),
}));
