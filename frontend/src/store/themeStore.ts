// ═══════════════════════════════════════════════════════════════════════
// Theme Store (Zustand)
// ARCH-02: Миграция ThemeContext → Zustand для предотвращения
// каскадных re-render'ов.
//
// Тема — это UI-состояние (не server state), поэтому используем Zustand.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

export type Theme = 'light' | 'dark' | 'system';

interface ThemeState {
  theme: Theme;
  isDark: boolean;
  setTheme: (theme: Theme) => void;
}

const getInitialTheme = (): Theme => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem('theme');
    return (saved as Theme) || 'system';
  }
  return 'system';
};

const getSystemDark = (): boolean => {
  if (typeof window !== 'undefined') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  return false;
};

const applyTheme = (theme: Theme): boolean => {
  if (typeof window === 'undefined') return false;
  const root = window.document.documentElement;
  const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const shouldBeDark = theme === 'dark' || (theme === 'system' && systemDark);

  if (shouldBeDark) {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }

  localStorage.setItem('theme', theme);
  return shouldBeDark;
};

export const useThemeStore = create<ThemeState>()((set) => {
  const initialTheme = getInitialTheme();
  const initialIsDark = applyTheme(initialTheme);

  // Следим за системной темой при 'system' режиме
  if (typeof window !== 'undefined') {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', () => {
      const state = useThemeStore.getState();
      if (state.theme === 'system') {
        const isDark = applyTheme('system');
        set({ isDark });
      }
    });
  }

  return {
    theme: initialTheme,
    isDark: initialIsDark,
    setTheme: (theme: Theme) => {
      const isDark = applyTheme(theme);
      set({ theme, isDark });
    },
  };
});
