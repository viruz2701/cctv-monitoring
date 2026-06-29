// ═══════════════════════════════════════════════════════════════════════
// WhiteLabelCustomizer — P3-NICE.2: White-label Theming
//
// Позволяет настроить:
//   - Название тенанта (brand name)
//   - Логотип (URL)
//   - Favicon
//   - Primary/accent цвета
//   - Шрифт
//   - Кастомный CSS
//   - Preview mode
//
// Соответствие:
//   - WCAG 2.1 SC 1.4.1 (Use of Color)
//   - WCAG 2.1 SC 2.5.3 (Label in Name)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Palette, Image, Type, Code, Eye, EyeOff,
  RotateCcw, Check, Upload, X, Globe,
} from './Icons';
import { useThemeStore, type WhiteLabelConfig, DEFAULT_WHITE_LABEL } from '../../store/themeStore';
import { Card, CardContent, CardTitle } from './Card';

// ═══ Preview themes ═══════════════════════════════════════════════════

const PREVIEW_THEMES = [
  { name: 'Корпоративный', primary: '#1e40af', accent: '#3b82f6' },
  { name: 'Тёмный', primary: '#0f172a', accent: '#475569' },
  { name: 'Технологичный', primary: '#0891b2', accent: '#06b6d4' },
  { name: 'Эко', primary: '#059669', accent: '#10b981' },
];

// ═══ Component ════════════════════════════════════════════════════════

export function WhiteLabelCustomizer() {
  const { t } = useTranslation();
  const whiteLabel = useThemeStore((s) => s.whiteLabel);
  const setWhiteLabel = useThemeStore((s) => s.setWhiteLabel);
  const toggleWhiteLabel = useThemeStore((s) => s.toggleWhiteLabel);
  const resetWhiteLabel = useThemeStore((s) => s.resetWhiteLabel);

  const [previewMode, setPreviewMode] = useState(false);
  const [localConfig, setLocalConfig] = useState<WhiteLabelConfig>({ ...whiteLabel });

  // ── Update local config ──────────────────────────────────────────
  const update = (partial: Partial<WhiteLabelConfig>) => {
    setLocalConfig((prev) => ({ ...prev, ...partial }));
  };

  // ── Apply to store ───────────────────────────────────────────────
  const handleApply = () => {
    setWhiteLabel({ ...localConfig, isActive: true });
  };

  // ── Toggle preview ───────────────────────────────────────────────
  const handlePreview = () => {
    if (!previewMode) {
      // Save current, apply local for preview
      setWhiteLabel({ ...localConfig, isActive: true });
    } else {
      // Restore saved
      setWhiteLabel(whiteLabel);
    }
    setPreviewMode(!previewMode);
  };

  // ── Quick theme picker ───────────────────────────────────────────
  const handleQuickTheme = (colors: { primary: string; accent: string }) => {
    update({ primaryColor: colors.primary, accentColor: colors.accent });
  };

  return (
    <Card variant="outlined" padding="md" className="space-y-5">
      <CardTitle className="flex items-center gap-2">
        <Palette className="w-4 h-4" />
        {t('whiteLabel.title') || 'White-label настройки'}
      </CardTitle>

      <CardContent className="space-y-5">
        {/* ── Enable toggle ─────────────────────────────────────── */}
        <label className="flex items-center gap-3 cursor-pointer">
          <button
            onClick={toggleWhiteLabel}
            className={`
              relative w-10 h-5 rounded-full transition-colors
              ${localConfig.isActive ? 'bg-blue-600' : 'bg-slate-300 dark:bg-slate-600'}
            `}
            role="switch"
            aria-checked={localConfig.isActive}
            aria-label={t('whiteLabel.enable') || 'Включить white-label'}
          >
            <span
              className={`
                absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full shadow transition-transform
                ${localConfig.isActive ? 'translate-x-5' : 'translate-x-0'}
              `}
            />
          </button>
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
            {t('whiteLabel.enable') || 'Включить white-label'}
          </span>
        </label>

        {/* ── Tenant name ────────────────────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
            <Globe className="w-3.5 h-3.5" />
            {t('whiteLabel.tenantName') || 'Название тенанта'}
          </legend>
          <input
            type="text"
            value={localConfig.tenantName}
            onChange={(e) => update({ tenantName: e.target.value })}
            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                       bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                       focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            placeholder={DEFAULT_WHITE_LABEL.tenantName}
            aria-label={t('whiteLabel.tenantName') || 'Название тенанта'}
          />
        </fieldset>

        {/* ── Logo URL ────────────────────────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
            <Image className="w-3.5 h-3.5" />
            {t('whiteLabel.logoUrl') || 'URL логотипа'}
          </legend>
          <div className="flex gap-2">
            <input
              type="text"
              value={localConfig.logoUrl}
              onChange={(e) => update({ logoUrl: e.target.value })}
              className="flex-1 px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                         bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                         focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
              placeholder="https://example.com/logo.png"
              aria-label={t('whiteLabel.logoUrl') || 'URL логотипа'}
            />
            {localConfig.logoUrl && (
              <div className="w-10 h-10 rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden flex-shrink-0">
                <img
                  src={localConfig.logoUrl}
                  alt="Preview"
                  className="w-full h-full object-contain"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = 'none';
                  }}
                />
              </div>
            )}
          </div>
        </fieldset>

        {/* ── Favicon URL ─────────────────────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
            {t('whiteLabel.faviconUrl') || 'URL фавиконки'}
          </legend>
          <input
            type="text"
            value={localConfig.faviconUrl}
            onChange={(e) => update({ faviconUrl: e.target.value })}
            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                       bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                       focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            placeholder="https://example.com/favicon.ico"
            aria-label={t('whiteLabel.faviconUrl') || 'URL фавиконки'}
          />
        </fieldset>

        {/* ── Colors ──────────────────────────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2 flex items-center gap-1.5">
            <Palette className="w-3.5 h-3.5" />
            {t('whiteLabel.colors') || 'Цвета бренда'}
          </legend>

          {/* Quick themes */}
          <div className="flex flex-wrap gap-2 mb-3">
            {PREVIEW_THEMES.map((theme) => (
              <button
                key={theme.name}
                onClick={() => handleQuickTheme(theme)}
                className={`
                  flex items-center gap-1.5 px-2 py-1 text-xs rounded-md border transition-all
                  ${localConfig.primaryColor === theme.primary
                    ? 'border-blue-500 ring-1 ring-blue-500 bg-blue-50 dark:bg-blue-900/20'
                    : 'border-slate-200 dark:border-slate-700 hover:border-slate-300'
                  }
                `}
                aria-label={`${theme.name} тема`}
              >
                <span className="w-3 h-3 rounded-full" style={{ backgroundColor: theme.primary }} />
                <span className="w-2 h-2 rounded-full" style={{ backgroundColor: theme.accent }} />
                <span className="text-slate-600 dark:text-slate-400">{theme.name}</span>
              </button>
            ))}
          </div>

          {/* Primary color */}
          <div className="flex items-center gap-3 mb-2">
            <label className="text-xs text-slate-500 dark:text-slate-400 w-20">
              {t('whiteLabel.primary') || 'Primary'}
            </label>
            <div className="flex items-center gap-2 flex-1">
              <input
                type="color"
                value={localConfig.primaryColor}
                onChange={(e) => update({ primaryColor: e.target.value })}
                className="w-8 h-8 rounded cursor-pointer border border-slate-300 dark:border-slate-600"
                aria-label={t('whiteLabel.primary') || 'Primary цвет'}
              />
              <input
                type="text"
                value={localConfig.primaryColor}
                onChange={(e) => update({ primaryColor: e.target.value })}
                className="flex-1 px-2 py-1 text-xs font-mono border border-slate-300 dark:border-slate-600 rounded
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
              />
            </div>
          </div>

          {/* Accent color */}
          <div className="flex items-center gap-3">
            <label className="text-xs text-slate-500 dark:text-slate-400 w-20">
              {t('whiteLabel.accent') || 'Accent'}
            </label>
            <div className="flex items-center gap-2 flex-1">
              <input
                type="color"
                value={localConfig.accentColor}
                onChange={(e) => update({ accentColor: e.target.value })}
                className="w-8 h-8 rounded cursor-pointer border border-slate-300 dark:border-slate-600"
                aria-label={t('whiteLabel.accent') || 'Accent цвет'}
              />
              <input
                type="text"
                value={localConfig.accentColor}
                onChange={(e) => update({ accentColor: e.target.value })}
                className="flex-1 px-2 py-1 text-xs font-mono border border-slate-300 dark:border-slate-600 rounded
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
              />
            </div>
          </div>
        </fieldset>

        {/* ── Font ─────────────────────────────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
            <Type className="w-3.5 h-3.5" />
            {t('whiteLabel.font') || 'Шрифт'}
          </legend>
          <select
            value={localConfig.fontFamily}
            onChange={(e) => update({ fontFamily: e.target.value })}
            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                       bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                       focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
            aria-label={t('whiteLabel.font') || 'Шрифт'}
          >
            <option value="Inter, system-ui, sans-serif">Inter</option>
            <option value="system-ui, sans-serif">System UI</option>
            <option value="'IBM Plex Sans', system-ui, sans-serif">IBM Plex Sans</option>
            <option value="'Segoe UI', system-ui, sans-serif">Segoe UI</option>
            <option value="'Roboto', system-ui, sans-serif">Roboto</option>
            <option value="'Montserrat', system-ui, sans-serif">Montserrat</option>
            <option value="'Open Sans', system-ui, sans-serif">Open Sans</option>
            <option value="'Noto Sans', system-ui, sans-serif">Noto Sans</option>
          </select>
        </fieldset>

        {/* ── Custom CSS ──────────────────────────────────────────── */}
        <fieldset>
          <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
            <Code className="w-3.5 h-3.5" />
            {t('whiteLabel.customCSS') || 'Кастомный CSS'}
          </legend>
          <textarea
            value={localConfig.customCSS}
            onChange={(e) => update({ customCSS: e.target.value })}
            rows={4}
            className="w-full px-3 py-2 text-xs font-mono border border-slate-300 dark:border-slate-600 rounded-lg
                       bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                       focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none resize-y"
            placeholder="/* Добавьте кастомные CSS-правила */&#10;.sidebar { background: var(--wl-primary); }"
            aria-label={t('whiteLabel.customCSS') || 'Кастомный CSS'}
          />
        </fieldset>

        {/* ── Preview toggle ──────────────────────────────────────── */}
        <div className="flex items-center gap-2">
          <button
            onClick={handlePreview}
            className={`
              flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg transition-all
              ${previewMode
                ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300 ring-2 ring-amber-500'
                : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300'
              }
            `}
            aria-pressed={previewMode}
          >
            {previewMode ? (
              <><EyeOff className="w-4 h-4" /> {t('whiteLabel.exitPreview') || 'Выйти из预览'}</>
            ) : (
              <><Eye className="w-4 h-4" /> {t('whiteLabel.preview') || 'Предпросмотр'}</>
            )}
          </button>

          <button
            onClick={handleApply}
            className="flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg
                       bg-blue-600 text-white hover:bg-blue-700 transition-colors"
          >
            <Check className="w-4 h-4" />
            {t('whiteLabel.apply') || 'Применить'}
          </button>

          <button
            onClick={resetWhiteLabel}
            className="flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg
                       bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300
                       transition-colors ml-auto"
          >
            <RotateCcw className="w-4 h-4" />
            {t('whiteLabel.reset') || 'Сбросить'}
          </button>
        </div>

        {/* ── Preview indicator ────────────────────────────────────── */}
        {previewMode && (
          <div className="px-3 py-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg text-xs text-amber-700 dark:text-amber-300">
            {t('whiteLabel.previewMode') || '🔍 Режим предпросмотра — изменения видны только вам'}
          </div>
        )}

        {/* ── Active indicator ─────────────────────────────────────── */}
        {whiteLabel.isActive && (
          <div className="px-3 py-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 rounded-lg text-xs text-green-700 dark:text-green-300 flex items-center gap-1.5">
            <Check className="w-3 h-3" />
            {t('whiteLabel.active') || 'White-label активен'} — {whiteLabel.tenantName}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ═══ WhiteLabelLogo — компонент для отображения логотипа тенанта ═══════

export function WhiteLabelLogo({ className = 'h-8' }: { className?: string }) {
  const whiteLabel = useThemeStore((s) => s.whiteLabel);

  if (!whiteLabel.isActive || !whiteLabel.logoUrl) return null;

  return (
    <img
      src={whiteLabel.logoUrl}
      alt={whiteLabel.tenantName}
      className={className + ' object-contain'}
      onError={(e) => {
        (e.target as HTMLImageElement).style.display = 'none';
      }}
    />
  );
}

// ═══ WhiteLabelProvider — CSS variables provider ═════════════════════════

export function WhiteLabelStyles() {
  const whiteLabel = useThemeStore((s) => s.whiteLabel);

  if (!whiteLabel.isActive) return null;

  return (
    <style>{`
      :root {
        ${whiteLabel.logoUrl ? `--wl-logo: url(${whiteLabel.logoUrl});` : ''}
        ${whiteLabel.primaryColor ? `--wl-primary: ${whiteLabel.primaryColor};` : ''}
        ${whiteLabel.accentColor ? `--wl-accent: ${whiteLabel.accentColor};` : ''}
        ${whiteLabel.fontFamily ? `--wl-font: ${whiteLabel.fontFamily};` : ''}
      }
    `}</style>
  );
}
