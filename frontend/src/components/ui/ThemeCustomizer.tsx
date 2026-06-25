// ═══════════════════════════════════════════════════════════════════════
// ThemeCustomizer — UX-14.3.4: Theming Engine
//
// Modal для кастомизации темы:
//   - Preset themes: default, ocean, forest, sunset
//   - Color picker для primary + accent
//   - Radius slider (sm/md/lg)
//   - Preview card
//   - Reset to Default
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import {
  Palette, Sun, Moon, Monitor, RotateCcw, Check,
  Undo2,
} from 'lucide-react';
import { useThemeStore, PRESET_COLORS, RADIUS_MAP, type ThemePreset, type RadiusSize } from '../../store/themeStore';
import { Modal } from './Modal';

interface ThemeCustomizerProps {
  isOpen: boolean;
  onClose: () => void;
}

const PRESET_INFO: { key: ThemePreset; label: string; description: string }[] = [
  { key: 'default', label: 'Default', description: 'Classic blue theme' },
  { key: 'ocean', label: 'Ocean', description: 'Cool cyan tones' },
  { key: 'forest', label: 'Forest', description: 'Natural green' },
  { key: 'sunset', label: 'Sunset', description: 'Warm amber glow' },
];

const THEME_MODES = [
  { key: 'light' as const, label: 'Light', icon: Sun },
  { key: 'dark' as const, label: 'Dark', icon: Moon },
  { key: 'system' as const, label: 'System', icon: Monitor },
];

export function ThemeCustomizer({ isOpen, onClose }: ThemeCustomizerProps) {
  const {
    theme, preset, primary, accent, radius, isDark,
    setTheme, setPreset, setPrimary, setAccent, setRadius, resetToDefaults,
  } = useThemeStore();

  const [localPrimary, setLocalPrimary] = useState(primary);
  const [localAccent, setLocalAccent] = useState(accent);
  const [hasCustomColor, setHasCustomColor] = useState(false);

  // Синхронизируем локальное состояние при открытии
  React.useEffect(() => {
    if (isOpen) {
      setLocalPrimary(primary);
      setLocalAccent(accent);
      setHasCustomColor(false);
    }
  }, [isOpen, primary, accent]);

  const handlePresetSelect = useCallback((p: ThemePreset) => {
    setPreset(p);
    const colors = PRESET_COLORS[p];
    setLocalPrimary(colors.primary);
    setLocalAccent(colors.accent);
    setHasCustomColor(false);
  }, [setPreset]);

  const handlePrimaryChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const color = e.target.value;
    setLocalPrimary(color);
    setPrimary(color);
    setHasCustomColor(true);
  }, [setPrimary]);

  const handleAccentChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const color = e.target.value;
    setLocalAccent(color);
    setAccent(color);
    setHasCustomColor(true);
  }, [setAccent]);

  const previewBgColor = isDark ? '#1e293b' : '#ffffff';
  const previewTextColor = isDark ? '#f1f5f9' : '#1e293b';
  const previewRadius = RADIUS_MAP[radius];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Theme Customizer" size="md">
      <div className="space-y-6">
        {/* ── Mode Switcher ──────────────────────────────────────────── */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Theme Mode
          </label>
          <div className="flex gap-2">
            {THEME_MODES.map((mode) => {
              const Icon = mode.icon;
              const isActive = theme === mode.key;
              return (
                <button
                  key={mode.key}
                  onClick={() => setTheme(mode.key)}
                  className={`flex items-center gap-2 px-4 py-2 text-sm rounded-lg border transition-all ${
                    isActive
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 shadow-sm'
                      : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
                  }`}
                  aria-pressed={isActive}
                >
                  <Icon size={16} />
                  {mode.label}
                </button>
              );
            })}
          </div>
        </div>

        {/* ── Preset Themes ───────────────────────────────────────────── */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Preset Themes
          </label>
          <div className="grid grid-cols-2 gap-2">
            {PRESET_INFO.map((p) => {
              const colors = PRESET_COLORS[p.key];
              const isActive = preset === p.key && !hasCustomColor;
              return (
                <button
                  key={p.key}
                  onClick={() => handlePresetSelect(p.key)}
                  className={`flex items-center gap-3 px-3 py-3 rounded-lg border transition-all text-left ${
                    isActive
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 ring-2 ring-blue-500/20'
                      : 'border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800'
                  }`}
                  aria-pressed={isActive}
                >
                  <div className="flex flex-col gap-1">
                    <div className="flex gap-1">
                      <span
                        className="w-5 h-5 rounded-full border border-slate-200 dark:border-slate-600"
                        style={{ backgroundColor: colors.primary }}
                      />
                      <span
                        className="w-5 h-5 rounded-full border border-slate-200 dark:border-slate-600"
                        style={{ backgroundColor: colors.accent }}
                      />
                    </div>
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
                      {p.label}
                    </span>
                    <span className="text-xs text-slate-400 dark:text-slate-500">
                      {p.description}
                    </span>
                  </div>
                  {isActive && (
                    <Check size={16} className="ml-auto text-blue-600 flex-shrink-0" />
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* ── Custom Colors ──────────────────────────────────────────── */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Custom Colors
          </label>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">
                Primary Color
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  value={localPrimary}
                  onChange={handlePrimaryChange}
                  className="w-10 h-10 p-0.5 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer bg-transparent"
                  aria-label="Primary color"
                />
                <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
                  {localPrimary}
                </span>
              </div>
            </div>
            <div>
              <label className="block text-xs text-slate-500 dark:text-slate-400 mb-1">
                Accent Color
              </label>
              <div className="flex items-center gap-2">
                <input
                  type="color"
                  value={localAccent}
                  onChange={handleAccentChange}
                  className="w-10 h-10 p-0.5 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer bg-transparent"
                  aria-label="Accent color"
                />
                <span className="text-xs font-mono text-slate-500 dark:text-slate-400">
                  {localAccent}
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* ── Radius ─────────────────────────────────────────────────── */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Border Radius: <span className="font-mono text-xs text-slate-400">{radius}</span>
          </label>
          <div className="flex gap-2">
            {(['sm', 'md', 'lg'] as RadiusSize[]).map((r) => {
              const isActive = radius === r;
              return (
                <button
                  key={r}
                  onClick={() => setRadius(r)}
                  className={`flex-1 px-3 py-2 text-sm rounded-lg border transition-all ${
                    isActive
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                      : 'border-slate-200 dark:border-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800'
                  }`}
                  aria-pressed={isActive}
                >
                  <div
                    className="w-8 h-2 mx-auto mb-1 bg-slate-300 dark:bg-slate-600"
                    style={{ borderRadius: RADIUS_MAP[r] }}
                  />
                  {r.toUpperCase()}
                </button>
              );
            })}
          </div>
        </div>

        {/* ── Preview ────────────────────────────────────────────────── */}
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            Preview
          </label>
          <div
            className="p-4 rounded-xl border border-slate-200 dark:border-slate-700 space-y-3 transition-all"
            style={{
              backgroundColor: previewBgColor,
              color: previewTextColor,
              borderRadius: `calc(${previewRadius} + 0.5rem)`,
            }}
          >
            <div className="flex items-center gap-3">
              <div
                className="w-10 h-10 rounded-lg flex items-center justify-center text-white font-bold text-sm"
                style={{
                  backgroundColor: `var(--color-primary, ${localPrimary})`,
                  borderRadius: previewRadius,
                }}
              >
                A
              </div>
              <div>
                <p className="text-sm font-medium">Sample Card</p>
                <p className="text-xs opacity-60">This is how components will look</p>
              </div>
            </div>
            <div className="flex gap-2">
              <span
                className="px-3 py-1.5 text-xs font-medium text-white rounded-md"
                style={{
                  backgroundColor: `var(--color-primary, ${localPrimary})`,
                  borderRadius: previewRadius,
                }}
              >
                Button
              </span>
              <span
                className="px-3 py-1.5 text-xs font-medium text-white rounded-md"
                style={{
                  backgroundColor: `var(--color-accent, ${localAccent})`,
                  borderRadius: previewRadius,
                }}
              >
                Accent
              </span>
              <span
                className="px-3 py-1.5 text-xs font-medium rounded-md border"
                style={{
                  borderColor: 'var(--color-primary, #2563eb)',
                  color: `var(--color-primary, ${localPrimary})`,
                  borderRadius: previewRadius,
                }}
              >
                Outline
              </span>
            </div>
          </div>
        </div>

        {/* ── Actions ────────────────────────────────────────────────── */}
        <div className="flex items-center justify-between pt-2 border-t border-slate-200 dark:border-slate-700">
          <button
            onClick={resetToDefaults}
            className="flex items-center gap-2 px-3 py-2 text-sm text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
            aria-label="Reset to default theme"
          >
            <Undo2 size={16} />
            Reset to Default
          </button>
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
          >
            Done
          </button>
        </div>
      </div>
    </Modal>
  );
}
