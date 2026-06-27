// ═══════════════════════════════════════════════════════════════════════
// ThemeCustomizer — UI для настройки темы (P3-UI.1 / UX-14.3.4)
//
// Позволяет менять:
//   - Тему: светлая / тёмная / системная
//   - Пресет: default / ocean / forest / sunset
//   - Radius: sm / md / lg
//
// Использует useThemeStore напрямую (Zustand).
//
// Соответствие:
//   - WCAG 2.1 SC 1.4.1 (Use of Color — пресеты различимы)
//   - WCAG 2.1 SC 2.5.3 (Label in Name)
//   - OWASP ASVS V7 (Error handling)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Sun, Moon, Monitor, Check } from 'lucide-react';
import { useThemeStore } from '../../store';
import { type Theme, type ThemePreset, type RadiusSize, PRESET_COLORS, RADIUS_MAP } from '../../store/themeStore';
import { Card, CardContent, CardTitle } from './Card';

// ═══ Theme mode options ═══

const THEME_OPTIONS: { value: Theme; icon: React.ReactNode; label: string }[] = [
  { value: 'light', icon: <Sun className="w-4 h-4" />, label: 'Светлая' },
  { value: 'dark', icon: <Moon className="w-4 h-4" />, label: 'Тёмная' },
  { value: 'system', icon: <Monitor className="w-4 h-4" />, label: 'Системная' },
];

// ═══ Preset options ═══

const PRESET_OPTIONS: { value: ThemePreset; label: string }[] = [
  { value: 'default', label: 'Default' },
  { value: 'ocean', label: 'Ocean' },
  { value: 'forest', label: 'Forest' },
  { value: 'sunset', label: 'Sunset' },
];

const RADIUS_OPTIONS: { value: RadiusSize; label: string }[] = [
  { value: 'sm', label: 'Маленький' },
  { value: 'md', label: 'Средний' },
  { value: 'lg', label: 'Большой' },
];

// ═══ Component ═══

export function ThemeCustomizer() {
  const theme = useThemeStore((s) => s.theme);
  const preset = useThemeStore((s) => s.preset);
  const radius = useThemeStore((s) => s.radius);
  const setTheme = useThemeStore((s) => s.setTheme);
  const setPreset = useThemeStore((s) => s.setPreset);
  const setRadius = useThemeStore((s) => s.setRadius);
  const resetToDefaults = useThemeStore((s) => s.resetToDefaults);

  return (
    <Card variant="outlined" padding="md" className="space-y-5">
      <CardTitle>Настройки темы</CardTitle>

      <CardContent className="space-y-5">
        {/* ── Theme mode ────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Режим
          </legend>
          <div className="flex gap-2">
            {THEME_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                onClick={() => setTheme(opt.value)}
                className={`
                  flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg
                  transition-all duration-150
                  ${theme === opt.value
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 ring-2 ring-blue-500'
                    : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                  }
                `}
                aria-pressed={theme === opt.value}
                aria-label={opt.label}
              >
                {opt.icon}
                <span className="hidden sm:inline">{opt.label}</span>
              </button>
            ))}
          </div>
        </fieldset>

        {/* ── Color preset ──────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Цветовая схема
          </legend>
          <div className="flex gap-2">
            {PRESET_OPTIONS.map((opt) => {
              const colors = PRESET_COLORS[opt.value];
              const isActive = preset === opt.value;
              return (
                <button
                  key={opt.value}
                  onClick={() => setPreset(opt.value)}
                  className={`
                    relative flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg
                    transition-all duration-150
                    ${isActive
                      ? 'ring-2 ring-blue-500 bg-white dark:bg-slate-800'
                      : 'bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700'
                    }
                  `}
                  aria-pressed={isActive}
                  aria-label={opt.label}
                >
                  <span
                    className="w-4 h-4 rounded-full"
                    style={{ backgroundColor: colors.primary }}
                    aria-hidden="true"
                  />
                  <span
                    className="w-3 h-3 rounded-full -ml-1.5"
                    style={{ backgroundColor: colors.accent }}
                    aria-hidden="true"
                  />
                  <span className="hidden sm:inline">{opt.label}</span>
                  {isActive && (
                    <Check className="w-3 h-3 text-blue-500" aria-hidden="true" />
                  )}
                </button>
              );
            })}
          </div>
        </fieldset>

        {/* ── Radius ────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Радиус скругления
          </legend>
          <div className="flex gap-2">
            {RADIUS_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                onClick={() => setRadius(opt.value)}
                className={`
                  px-3 py-2 text-sm font-medium rounded-lg transition-all duration-150
                  ${radius === opt.value
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300 ring-2 ring-blue-500'
                    : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:hover:bg-slate-700'
                  }
                `}
                style={{ borderRadius: RADIUS_MAP[opt.value] }}
                aria-pressed={radius === opt.value}
                aria-label={opt.label}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </fieldset>

        {/* ── Reset ─────────────────────────── */}
        <button
          onClick={resetToDefaults}
          className="text-sm text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200 underline underline-offset-2 transition-colors"
        >
          Сбросить на стандартные
        </button>
      </CardContent>
    </Card>
  );
}
