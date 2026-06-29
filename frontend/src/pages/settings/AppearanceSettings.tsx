// ═══════════════════════════════════════════════════════════════════════
// Appearance Settings — UX-14.3.4
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Palette } from '../components/ui/Icons';
import { useTranslation } from 'react-i18next';
import { Card, CardHeader, CardBody, Button } from '../../components/ui';
import { useThemeStore, PRESET_COLORS, type ThemePreset } from '../../store/themeStore';

interface Props {
  onOpenCustomizer: () => void;
}

export function AppearanceSettings({ onOpenCustomizer }: Props) {
  const { t } = useTranslation();
  const { theme, preset, primary, radius, setTheme, setPreset } = useThemeStore();

  const THEME_MODES = [
    { key: 'light' as const, label: t('light') || 'Light', emoji: '☀️' },
    { key: 'dark' as const, label: t('dark') || 'Dark', emoji: '🌙' },
    { key: 'system' as const, label: t('system') || 'System', emoji: '💻' },
  ];

  const PRESET_LIST: { key: ThemePreset; label: string }[] = [
    { key: 'default', label: t('default') || 'Default' },
    { key: 'ocean', label: 'Ocean' },
    { key: 'forest', label: 'Forest' },
    { key: 'sunset', label: 'Sunset' },
  ];

  return (
    <div className="space-y-6">
      {/* Theme Mode */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <span className="text-lg font-semibold text-slate-900 dark:text-white">
            {t('theme_mode') || 'Theme Mode'}
          </span>
        </CardHeader>
        <CardBody>
          <div className="flex gap-3">
            {THEME_MODES.map((mode) => (
              <button
                key={mode.key}
                onClick={() => setTheme(mode.key)}
                className={`flex-1 flex flex-col items-center gap-2 px-4 py-4 rounded-xl border transition-all ${
                  theme === mode.key
                    ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 shadow-sm'
                    : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
                }`}
                aria-pressed={theme === mode.key}
              >
                <span className="text-2xl">{mode.emoji}</span>
                <span className="text-sm font-medium">{mode.label}</span>
              </button>
            ))}
          </div>
        </CardBody>
      </Card>

      {/* Preset Themes */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <span className="text-lg font-semibold text-slate-900 dark:text-white">
            {t('preset_themes') || 'Preset Themes'}
          </span>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            {PRESET_LIST.map((p) => {
              const colors = PRESET_COLORS[p.key];
              const isActive = preset === p.key;
              return (
                <button
                  key={p.key}
                  onClick={() => setPreset(p.key)}
                  className={`flex flex-col items-center gap-2 px-3 py-4 rounded-xl border transition-all ${
                    isActive
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 ring-2 ring-blue-500/20'
                      : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'
                  }`}
                  aria-pressed={isActive}
                >
                  <div className="flex gap-1.5">
                    <span
                      className="w-6 h-6 rounded-full border-2 border-white dark:border-slate-600 shadow-sm"
                      style={{ backgroundColor: colors.primary }}
                    />
                    <span
                      className="w-6 h-6 rounded-full border-2 border-white dark:border-slate-600 shadow-sm"
                      style={{ backgroundColor: colors.accent }}
                    />
                  </div>
                  <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{p.label}</span>
                </button>
              );
            })}
          </div>

          <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-700">
            <Button onClick={onOpenCustomizer} icon={<Palette className="w-4 h-4" />}>
              {t('customize_theme') || 'Customize Theme'}
            </Button>
          </div>
        </CardBody>
      </Card>

      {/* Current Config Summary */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <span className="text-lg font-semibold text-slate-900 dark:text-white">
            {t('current_config') || 'Current Configuration'}
          </span>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <span className="text-xs text-slate-500 dark:text-slate-400 block">{t('mode') || 'Mode'}</span>
              <span className="font-medium text-slate-900 dark:text-white capitalize">{theme}</span>
            </div>
            <div>
              <span className="text-xs text-slate-500 dark:text-slate-400 block">{t('preset') || 'Preset'}</span>
              <span className="font-medium text-slate-900 dark:text-white capitalize">{preset}</span>
            </div>
            <div>
              <span className="text-xs text-slate-500 dark:text-slate-400 block">Primary</span>
              <div className="flex items-center gap-1.5">
                <span className="w-4 h-4 rounded-full border border-slate-300" style={{ backgroundColor: primary }} />
                <span className="font-mono text-xs text-slate-600 dark:text-slate-400">{primary}</span>
              </div>
            </div>
            <div>
              <span className="text-xs text-slate-500 dark:text-slate-400 block">Radius</span>
              <span className="font-medium text-slate-900 dark:text-white">{radius}</span>
            </div>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
