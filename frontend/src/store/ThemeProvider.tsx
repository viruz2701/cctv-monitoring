// ═══════════════════════════════════════════════════════════════════════
// ThemeProvider — React provider for backward compatibility
// ARCH-02: Part of Context→Zustand migration (P1-ARCH.1)
//
// Новый код: используй useThemeStore напрямую из '../store'
// Legacy: используй ThemeProvider + useTheme из этого файла
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext } from 'react';
import { useThemeStore, type Theme } from './themeStore';

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
