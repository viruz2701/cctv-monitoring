// ═══════════════════════════════════════════════════════════════════════
// ThemeContext — Bridge to Zustand Theme Store (ARCH-02)
//
// Эта обёртка обеспечивает обратную совместимость с существующим кодом.
// Новый код ДОЛЖЕН импортировать useThemeStore напрямую из store/.
//
// Миграция:
//   Было:  import { useTheme } from './context/ThemeContext'
//   Стало: import { useThemeStore } from '../store'
//
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext } from 'react';
import { useThemeStore, type Theme } from '../store';

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
    // Новый код: используй useThemeStore напрямую
    // Legacy: используй этот хук
    const context = useContext(ThemeContext);
    if (context === undefined) {
        throw new Error('useTheme must be used within a ThemeProvider');
    }
    return context;
}
