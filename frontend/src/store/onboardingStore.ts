// ═══════════════════════════════════════════════════════════════════════
// Onboarding Store (Zustand)
// UX-14.1.6: Состояние онбординг-тура для новых пользователей
//
// Хранит:
//   - completed: флаг, что тур завершён
//   - Персистентность через localStorage
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

const STORAGE_KEY = 'cctv_onboarding_completed';

interface OnboardingState {
  completed: boolean;
  running: boolean;
  markCompleted: () => void;
  startTour: () => void;
  stopTour: () => void;
  resetTour: () => void;
}

export const useOnboardingStore = create<OnboardingState>()((set) => ({
  completed: localStorage.getItem(STORAGE_KEY) === 'true',
  running: false,
  markCompleted: () => {
    localStorage.setItem(STORAGE_KEY, 'true');
    set({ completed: true, running: false });
  },
  startTour: () => set({ running: true }),
  stopTour: () => set({ running: false }),
  resetTour: () => {
    localStorage.removeItem(STORAGE_KEY);
    set({ completed: false, running: true });
  },
}));
