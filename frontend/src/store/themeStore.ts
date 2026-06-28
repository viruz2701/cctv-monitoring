// ═══════════════════════════════════════════════════════════════════════
// Theme Store (Zustand) — UX-14.3.4: Theming Engine
//
// Поддержка custom themes:
//   - Режим: 'light' | 'dark' | 'system'
//   - Пресеты: 'default', 'ocean', 'forest', 'sunset'
//   - Кастомные: primary, accent цвета + radius
//   - CSS custom properties через applyTheme
//   - Persist в localStorage
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

export type Theme = 'light' | 'dark' | 'system';
export type ThemePreset = 'default' | 'ocean' | 'forest' | 'sunset';
export type RadiusSize = 'sm' | 'md' | 'lg';

export interface CustomTheme {
  mode: Theme;
  preset: ThemePreset;
  primary: string;
  accent: string;
  radius: RadiusSize;
}

// ═══ White-label types (P3-NICE.2) ═════════════════════════════════════

export interface WhiteLabelConfig {
  tenantName: string;
  logoUrl: string;
  faviconUrl: string;
  primaryColor: string;
  accentColor: string;
  fontFamily: string;
  customCSS: string;
  isActive: boolean;
}

export const DEFAULT_WHITE_LABEL: WhiteLabelConfig = {
  tenantName: 'CCTV Health Monitor',
  logoUrl: '',
  faviconUrl: '',
  primaryColor: '#2563eb',
  accentColor: '#6366f1',
  fontFamily: 'Inter, system-ui, sans-serif',
  customCSS: '',
  isActive: false,
};

interface ThemeState {
  theme: Theme;
  isDark: boolean;
  preset: ThemePreset;
  primary: string;
  accent: string;
  radius: RadiusSize;
  setTheme: (theme: Theme) => void;
  setPreset: (preset: ThemePreset) => void;
  setPrimary: (color: string) => void;
  setAccent: (color: string) => void;
  setRadius: (radius: RadiusSize) => void;
  resetToDefaults: () => void;
  getCustomTheme: () => CustomTheme;
  applyCustomTheme: (custom: CustomTheme) => void;
  // White-label (P3-NICE.2)
  whiteLabel: WhiteLabelConfig;
  setWhiteLabel: (config: WhiteLabelConfig) => void;
  toggleWhiteLabel: () => void;
  resetWhiteLabel: () => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Preset color schemes
// ═══════════════════════════════════════════════════════════════════════

export const PRESET_COLORS: Record<ThemePreset, { primary: string; accent: string }> = {
  default: { primary: '#2563eb', accent: '#6366f1' },
  ocean: { primary: '#0891b2', accent: '#06b6d4' },
  forest: { primary: '#059669', accent: '#10b981' },
  sunset: { primary: '#d97706', accent: '#f59e0b' },
};

export const RADIUS_MAP: Record<RadiusSize, string> = {
  sm: '0.375rem',
  md: '0.5rem',
  lg: '0.75rem',
};

const STORAGE_KEYS = {
  theme: 'theme',
  preset: 'theme-preset',
  primary: 'theme-primary',
  accent: 'theme-accent',
  radius: 'theme-radius',
};

// ═══════════════════════════════════════════════════════════════════════
// Init helpers
// ═══════════════════════════════════════════════════════════════════════

const getInitialTheme = (): Theme => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEYS.theme);
    return (saved as Theme) || 'system';
  }
  return 'system';
};

const getInitialPreset = (): ThemePreset => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEYS.preset);
    if (saved && Object.keys(PRESET_COLORS).includes(saved)) return saved as ThemePreset;
  }
  return 'default';
};

const getInitialPrimary = (): string => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEYS.primary);
    if (saved) return saved;
  }
  return PRESET_COLORS.default.primary;
};

const getInitialAccent = (): string => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEYS.accent);
    if (saved) return saved;
  }
  return PRESET_COLORS.default.accent;
};

const getInitialRadius = (): RadiusSize => {
  if (typeof window !== 'undefined') {
    const saved = localStorage.getItem(STORAGE_KEYS.radius);
    if (saved && Object.keys(RADIUS_MAP).includes(saved)) return saved as RadiusSize;
  }
  return 'md';
};

const getSystemDark = (): boolean => {
  if (typeof window !== 'undefined') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches;
  }
  return false;
};

// ═══════════════════════════════════════════════════════════════════════
// applyTheme — устанавливает CSS custom properties + классы темы
// ═══════════════════════════════════════════════════════════════════════

const applyTheme = (
  theme: Theme,
  primary: string,
  accent: string,
  radius: RadiusSize,
  preset: ThemePreset,
): boolean => {
  if (typeof window === 'undefined') return false;
  const root = window.document.documentElement;
  const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  const shouldBeDark = theme === 'dark' || (theme === 'system' && systemDark);

  // Dark mode class
  root.classList.toggle('dark', shouldBeDark);

  // Theme preset class (удаляем старые, добавляем новый)
  const allPresets = Object.keys(PRESET_COLORS) as ThemePreset[];
  allPresets.forEach((p) => root.classList.remove(`theme-${p}`));
  if (preset !== 'default') {
    root.classList.add(`theme-${preset}`);
  }

  // CSS custom properties
  root.style.setProperty('--color-primary', primary);
  root.style.setProperty('--color-accent', accent);
  root.style.setProperty('--radius', RADIUS_MAP[radius]);

  // Persist
  localStorage.setItem(STORAGE_KEYS.theme, theme);
  localStorage.setItem(STORAGE_KEYS.preset, preset);
  localStorage.setItem(STORAGE_KEYS.primary, primary);
  localStorage.setItem(STORAGE_KEYS.accent, accent);
  localStorage.setItem(STORAGE_KEYS.radius, radius);

  return shouldBeDark;
};

// ═══ White-label helpers (P3-NICE.2) ═══════════════════════════════════

const WL_STORAGE_KEY = 'white-label-config';

const getInitialWhiteLabel = (): WhiteLabelConfig => {
  if (typeof window !== 'undefined') {
    try {
      const saved = localStorage.getItem(WL_STORAGE_KEY);
      if (saved) return JSON.parse(saved) as WhiteLabelConfig;
    } catch { /* ignore */ }
  }
  return DEFAULT_WHITE_LABEL;
};

const applyWhiteLabel = (config: WhiteLabelConfig): void => {
  if (typeof window === 'undefined' || !config.isActive) return;

  const root = window.document.documentElement;

  // Logo as CSS variable for use in components
  if (config.logoUrl) {
    root.style.setProperty('--wl-logo-url', `url(${config.logoUrl})`);
  }

  // Brand colors override theme colors
  if (config.primaryColor) {
    root.style.setProperty('--color-primary', config.primaryColor);
  }
  if (config.accentColor) {
    root.style.setProperty('--color-accent', config.accentColor);
  }

  // Font
  if (config.fontFamily) {
    root.style.setProperty('--wl-font-family', config.fontFamily);
    root.style.fontFamily = config.fontFamily;
  }

  // Favicon
  if (config.faviconUrl) {
    const link = document.querySelector<HTMLLinkElement>('link[rel*="icon"]');
    if (link) {
      link.href = config.faviconUrl;
    }
  }

  // Document title
  document.title = config.tenantName || 'CCTV Health Monitor';

  // Custom CSS injection
  if (config.customCSS) {
    let styleEl = document.getElementById('wl-custom-css');
    if (!styleEl) {
      styleEl = document.createElement('style');
      styleEl.id = 'wl-custom-css';
      document.head.appendChild(styleEl);
    }
    styleEl.textContent = config.customCSS;
  } else {
    const styleEl = document.getElementById('wl-custom-css');
    if (styleEl) styleEl.remove();
  }

  // Persist
  localStorage.setItem(WL_STORAGE_KEY, JSON.stringify(config));
};

const removeWhiteLabel = (): void => {
  if (typeof window === 'undefined') return;
  const root = window.document.documentElement;

  root.style.removeProperty('--wl-logo-url');
  root.style.removeProperty('--wl-font-family');
  root.style.fontFamily = '';

  const styleEl = document.getElementById('wl-custom-css');
  if (styleEl) styleEl.remove();

  localStorage.removeItem(WL_STORAGE_KEY);
};

// ═══════════════════════════════════════════════════════════════════════
// Store
// ═══════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════
// ThemeProvider — React provider for backward compatibility
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext } from 'react';

type ThemeContextType = {
    theme: Theme;
    setTheme: (theme: Theme) => void;
    isDark: boolean;
};

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

export function ThemeProvider({ children }: { children: React.ReactNode }) {
    const theme = useThemeStore((s) => s.theme);
    const setTheme = useThemeStore((s) => s.setTheme);
    const isDark = useThemeStore((s) => s.isDark);

    return (
        <ThemeContext.Provider value={{ theme, setTheme, isDark }}>
            {children}
        </ThemeContext.Provider>
    );
}

export function useTheme() {
    const context = useContext(ThemeContext);
    if (context === undefined) {
        throw new Error('useTheme must be used within a ThemeProvider');
    }
    return context;
}

export const useThemeStore = create<ThemeState>()((set, get) => {
  const initialPreset = getInitialPreset();
  const initialPrimary = getInitialPrimary();
  const initialAccent = getInitialAccent();
  const initialRadius = getInitialRadius();
  const initialTheme = getInitialTheme();
  const initialIsDark = applyTheme(initialTheme, initialPrimary, initialAccent, initialRadius, initialPreset);
  const initialWL = getInitialWhiteLabel();

  // Следим за системной темой
  if (typeof window !== 'undefined') {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', () => {
      const state = get();
      if (state.theme === 'system') {
        const isDark = applyTheme('system', state.primary, state.accent, state.radius, state.preset);
        set({ isDark });
      }
    });
  }

  // Apply initial white-label if active
  if (initialWL.isActive) {
    applyWhiteLabel(initialWL);
  }

  return {
    theme: initialTheme,
    isDark: initialIsDark,
    preset: initialPreset,
    primary: initialPrimary,
    accent: initialAccent,
    radius: initialRadius,

    setTheme: (theme: Theme) => {
      const state = get();
      const isDark = applyTheme(theme, state.primary, state.accent, state.radius, state.preset);
      set({ theme, isDark });
    },

    setPreset: (preset: ThemePreset) => {
      const colors = PRESET_COLORS[preset];
      const state = get();
      const isDark = applyTheme(state.theme, colors.primary, colors.accent, state.radius, preset);
      set({ preset, primary: colors.primary, accent: colors.accent, isDark });
    },

    setPrimary: (color: string) => {
      const state = get();
      const isDark = applyTheme(state.theme, color, state.accent, state.radius, state.preset);
      set({ primary: color, isDark });
    },

    setAccent: (color: string) => {
      const state = get();
      const isDark = applyTheme(state.theme, state.primary, color, state.radius, state.preset);
      set({ accent: color, isDark });
    },

    setRadius: (radius: RadiusSize) => {
      const state = get();
      const isDark = applyTheme(state.theme, state.primary, state.accent, radius, state.preset);
      set({ radius, isDark });
    },

    resetToDefaults: () => {
      const preset: ThemePreset = 'default';
      const colors = PRESET_COLORS[preset];
      const radius: RadiusSize = 'md';
      const theme: Theme = 'system';
      const isDark = applyTheme(theme, colors.primary, colors.accent, radius, preset);
      set({
        theme,
        isDark,
        preset,
        primary: colors.primary,
        accent: colors.accent,
        radius,
      });
    },

    getCustomTheme: (): CustomTheme => {
      const state = get();
      return {
        mode: state.theme,
        preset: state.preset,
        primary: state.primary,
        accent: state.accent,
        radius: state.radius,
      };
    },

    applyCustomTheme: (custom: CustomTheme) => {
      const isDark = applyTheme(custom.mode, custom.primary, custom.accent, custom.radius, custom.preset);
      set({
        theme: custom.mode,
        preset: custom.preset,
        primary: custom.primary,
        accent: custom.accent,
        radius: custom.radius,
        isDark,
      });
    },

    // ── White-label methods (P3-NICE.2) ────────────────────────────
    whiteLabel: initialWL,

    setWhiteLabel: (config: WhiteLabelConfig) => {
      applyWhiteLabel(config);
      set({ whiteLabel: config });
    },

    toggleWhiteLabel: () => {
      const state = get();
      const newConfig = { ...state.whiteLabel, isActive: !state.whiteLabel.isActive };
      if (newConfig.isActive) {
        applyWhiteLabel(newConfig);
      } else {
        removeWhiteLabel();
        // Re-apply theme colors
        applyTheme(state.theme, state.primary, state.accent, state.radius, state.preset);
      }
      set({ whiteLabel: newConfig });
    },

    resetWhiteLabel: () => {
      removeWhiteLabel();
      const state = get();
      applyTheme(state.theme, state.primary, state.accent, state.radius, state.preset);
      set({ whiteLabel: DEFAULT_WHITE_LABEL });
    },
  };
});
