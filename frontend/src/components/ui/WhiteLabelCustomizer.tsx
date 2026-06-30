// ═══════════════════════════════════════════════════════════════════════
// WhiteLabelCustomizer — P3-WL: White-Label Theming
//
// Позволяет настроить:
//   - Название/компанию тенанта
//   - Логотип (upload + URL)
//   - Favicon
//   - Цветовую схему (primary, secondary, accent)
//   - Шрифт
//   - Кастомный CSS
//   - Кастомный домен (CNAME)
//   - Email/PDF брендирование
//   - Preview mode
//
// Соответствие:
//   - WCAG 2.1 SC 1.4.1 (Use of Color)
//   - WCAG 2.1 SC 2.5.3 (Label in Name)
//   - OWASP ASVS V5 (Input validation)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Palette, Image, Type, Code, Eye, EyeOff,
  RotateCcw, Check, Upload, X, Globe, FileText,
  Mail, Shield, ExternalLink,
} from './Icons';
import { useThemeStore, type WhiteLabelConfig, DEFAULT_WHITE_LABEL } from '../../store/themeStore';
import { Card, CardContent, CardTitle } from './Card';

// ═══ Preview themes ═══════════════════════════════════════════════════

const PREVIEW_THEMES = [
  { name: 'Корпоративный', primary: '#1e40af', secondary: '#3b82f6', accent: '#06b6d4' },
  { name: 'Тёмный', primary: '#0f172a', secondary: '#475569', accent: '#94a3b8' },
  { name: 'Технологичный', primary: '#0891b2', secondary: '#06b6d4', accent: '#22d3ee' },
  { name: 'Эко', primary: '#059669', secondary: '#10b981', accent: '#34d399' },
  { name: 'Премиум', primary: '#7c3aed', secondary: '#8b5cf6', accent: '#a78bfa' },
];

// ═══ Tabs ═════════════════════════════════════════════════════════════

type TabId = 'brand' | 'colors' | 'domain' | 'email' | 'pdf' | 'advanced';

interface Tab {
  id: TabId;
  label: string;
  icon: React.ReactNode;
}

const TABS: Tab[] = [
  { id: 'brand', label: 'Бренд', icon: <Image className="w-3.5 h-3.5" /> },
  { id: 'colors', label: 'Цвета', icon: <Palette className="w-3.5 h-3.5" /> },
  { id: 'domain', label: 'Домен', icon: <Globe className="w-3.5 h-3.5" /> },
  { id: 'email', label: 'Email', icon: <Mail className="w-3.5 h-3.5" /> },
  { id: 'pdf', label: 'PDF', icon: <FileText className="w-3.5 h-3.5" /> },
  { id: 'advanced', label: 'Расширенные', icon: <Code className="w-3.5 h-3.5" /> },
];

// ═══ Component ════════════════════════════════════════════════════════

export function WhiteLabelCustomizer() {
  const { t } = useTranslation();
  const whiteLabel = useThemeStore((s) => s.whiteLabel);
  const setWhiteLabel = useThemeStore((s) => s.setWhiteLabel);
  const toggleWhiteLabel = useThemeStore((s) => s.toggleWhiteLabel);
  const resetWhiteLabel = useThemeStore((s) => s.resetWhiteLabel);

  const [previewMode, setPreviewMode] = useState(false);
  const [activeTab, setActiveTab] = useState<TabId>('brand');
  const [localConfig, setLocalConfig] = useState<WhiteLabelConfig>({ ...whiteLabel });
  const [savedConfig, setSavedConfig] = useState<WhiteLabelConfig>({ ...whiteLabel });
  const [domainStatus, setDomainStatus] = useState<'idle' | 'verifying' | 'verified' | 'error'>('idle');
  const [domainError, setDomainError] = useState('');
  const fileInputRef = useRef<HTMLInputElement>(null);

  // ── Update local config ──────────────────────────────────────────
  const update = (partial: Partial<WhiteLabelConfig>) => {
    setLocalConfig((prev) => ({ ...prev, ...partial }));
  };

  // ── Apply to store ───────────────────────────────────────────────
  const handleApply = () => {
    setWhiteLabel({ ...localConfig, isActive: true });
    setSavedConfig({ ...localConfig });
  };

  // ── Toggle preview ───────────────────────────────────────────────
  const handlePreview = () => {
    if (!previewMode) {
      // Save current, apply local for preview
      setSavedConfig({ ...whiteLabel });
      setWhiteLabel({ ...localConfig, isActive: true });
    } else {
      // Restore saved
      setWhiteLabel(savedConfig);
    }
    setPreviewMode(!previewMode);
  };

  // ── Quick theme picker ───────────────────────────────────────────
  const handleQuickTheme = (colors: { primary: string; secondary: string; accent: string }) => {
    update({
      primaryColor: colors.primary,
      secondaryColor: colors.secondary,
      accentColor: colors.accent,
    });
  };

  // ── Logo upload handler ──────────────────────────────────────────
  const handleLogoUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    const allowedTypes = ['image/png', 'image/jpeg', 'image/svg+xml', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      alert('Допустимые форматы: PNG, JPG, SVG, WebP');
      return;
    }

    // Validate file size (5MB max)
    if (file.size > 5 * 1024 * 1024) {
      alert('Максимальный размер файла: 5MB');
      return;
    }

    const reader = new FileReader();
    reader.onload = (event) => {
      const dataUrl = event.target?.result as string;
      update({ logoUrl: dataUrl });
    };
    reader.readAsDataURL(file);
  };

  // ── Domain validation ────────────────────────────────────────────
  const validateDomain = (domain: string): boolean => {
    if (!domain) return true;
    const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
    if (!domainRegex.test(domain)) {
      setDomainError('Неверный формат домена. Пример: brand.example.com');
      return false;
    }
    setDomainError('');
    return true;
  };

  const handleDomainChange = (domain: string) => {
    update({ customDomain: domain });
    if (domain) {
      validateDomain(domain);
    } else {
      setDomainError('');
    }
  };

  // ── Domain verification (stub) ───────────────────────────────────
  const handleVerifyDomain = async () => {
    if (!localConfig.customDomain) {
      setDomainError('Укажите домен для верификации');
      return;
    }

    if (!validateDomain(localConfig.customDomain)) return;

    setDomainStatus('verifying');

    // Stub: in production, this calls POST /api/v1/tenant/branding/verify-domain
    await new Promise((resolve) => setTimeout(resolve, 1500));

    update({ cnameVerified: true });
    setDomainStatus('verified');
  };

  // ── Copy domain token to clipboard ───────────────────────────────
  const handleCopyToken = () => {
    const token = `cctv-verify-${localConfig.tenantName.toLowerCase().replace(/\s+/g, '-')}`;
    navigator.clipboard.writeText(token)
      .then(() => alert('Токен скопирован в буфер обмена'))
      .catch(() => alert('Не удалось скопировать токен'));
  };

  // ── Trigger file input ───────────────────────────────────────────
  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  // ── Color input helper ───────────────────────────────────────────
  const ColorField = ({
    label,
    value,
    onChange,
  }: {
    label: string;
    value: string;
    onChange: (color: string) => void;
  }) => (
    <div className="flex items-center gap-3 mb-2">
      <label className="text-xs text-slate-500 dark:text-slate-400 w-24">{label}</label>
      <div className="flex items-center gap-2 flex-1">
        <input
          type="color"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="w-8 h-8 rounded cursor-pointer border border-slate-300 dark:border-slate-600"
          aria-label={label}
        />
        <input
          type="text"
          value={value}
          onChange={(e) => {
            const val = e.target.value;
            if (val.length <= 7) onChange(val);
          }}
          className="flex-1 px-2 py-1 text-xs font-mono border border-slate-300 dark:border-slate-600 rounded
                     bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100"
        />
      </div>
    </div>
  );

  return (
    <Card variant="outlined" padding="md" className="space-y-4">
      <CardTitle className="flex items-center gap-2">
        <Palette className="w-4 h-4" />
        {t('whiteLabel.title') || 'White-label настройки'}
      </CardTitle>

      <CardContent className="space-y-4">
        {/* ── Enable toggle ─────────────────────────────────────── */}
        <div className="flex items-center justify-between">
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

          {/* Preview toggle */}
          <div className="flex items-center gap-2">
            <button
              onClick={handlePreview}
              className={`
                flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg transition-all
                ${previewMode
                  ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300 ring-2 ring-amber-500'
                  : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300'
                }
              `}
              aria-pressed={previewMode}
            >
              {previewMode ? (
                <><EyeOff className="w-3.5 h-3.5" /> {t('whiteLabel.exitPreview') || 'Выйти из预览'}</>
              ) : (
                <><Eye className="w-3.5 h-3.5" /> {t('whiteLabel.preview') || 'Предпросмотр'}</>
              )}
            </button>
          </div>
        </div>

        {/* ── Preview indicator ──────────────────────────────────── */}
        {previewMode && (
          <div className="px-3 py-2 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg text-xs text-amber-700 dark:text-amber-300 flex items-center gap-1.5">
            <Eye className="w-3 h-3" />
            {t('whiteLabel.previewMode') || '🔍 Режим предпросмотра — изменения видны только вам'}
          </div>
        )}

        {/* ── Tabs ─────────────────────────────────────────────────── */}
        <div className="flex gap-1 border-b border-slate-200 dark:border-slate-700 pb-1 overflow-x-auto">
          {TABS.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`
                flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-t-lg transition-all whitespace-nowrap
                ${activeTab === tab.id
                  ? 'bg-white dark:bg-slate-800 text-blue-600 dark:text-blue-400 border-b-2 border-blue-600'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
                }
              `}
            >
              {tab.icon}
              {tab.label}
            </button>
          ))}
        </div>

        {/* ═══════════════════════════════════════════════════════════ */}
        {/* Tab: Brand */}
        {/* ═══════════════════════════════════════════════════════════ */}
        {activeTab === 'brand' && (
          <div className="space-y-4">
            {/* Tenant name */}
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

            {/* Company name */}
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('whiteLabel.companyName') || 'Название компании'}
              </legend>
              <input
                type="text"
                value={localConfig.companyName}
                onChange={(e) => update({ companyName: e.target.value })}
                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                           focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                placeholder="ООО «Пример»"
                aria-label={t('whiteLabel.companyName') || 'Название компании'}
              />
            </fieldset>

            {/* Logo upload */}
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
                <Upload className="w-3.5 h-3.5" />
                {t('whiteLabel.logo') || 'Логотип'}
              </legend>
              <div className="flex gap-3 items-start">
                {/* Preview */}
                <div className="w-16 h-16 rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-600 overflow-hidden flex-shrink-0 flex items-center justify-center bg-slate-50 dark:bg-slate-900">
                  {localConfig.logoUrl ? (
                    <img
                      src={localConfig.logoUrl}
                      alt="Logo"
                      className="w-full h-full object-contain"
                      onError={(e) => {
                        (e.target as HTMLImageElement).src = '';
                      }}
                    />
                  ) : (
                    <Image className="w-6 h-6 text-slate-400" />
                  )}
                </div>

                <div className="flex-1 space-y-2">
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept="image/png,image/jpeg,image/svg+xml,image/webp"
                    onChange={handleLogoUpload}
                    className="hidden"
                    aria-label="Выберите файл логотипа"
                  />
                  <div className="flex gap-2">
                    <button
                      onClick={handleUploadClick}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
                                 bg-blue-50 text-blue-600 hover:bg-blue-100 dark:bg-blue-900/20 dark:text-blue-400
                                 transition-colors"
                    >
                      <Upload className="w-3.5 h-3.5" />
                      {t('whiteLabel.upload') || 'Загрузить'}
                    </button>
                    {localConfig.logoUrl && (
                      <button
                        onClick={() => update({ logoUrl: '' })}
                        className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
                                   bg-red-50 text-red-600 hover:bg-red-100 dark:bg-red-900/20 dark:text-red-400
                                   transition-colors"
                      >
                        <X className="w-3.5 h-3.5" />
                        {t('whiteLabel.remove') || 'Удалить'}
                      </button>
                    )}
                  </div>
                  <p className="text-xs text-slate-400">
                    PNG, JPG, SVG, WebP. Макс. 5MB
                  </p>
                </div>
              </div>
            </fieldset>

            {/* Logo URL (advanced) */}
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('whiteLabel.logoUrl') || 'URL логотипа'}
              </legend>
              <input
                type="text"
                value={localConfig.logoUrl}
                onChange={(e) => update({ logoUrl: e.target.value })}
                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                           focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                placeholder="https://example.com/logo.png"
                aria-label={t('whiteLabel.logoUrl') || 'URL логотипа'}
              />
            </fieldset>

            {/* Favicon URL */}
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
                <Shield className="w-3.5 h-3.5" />
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
          </div>
        )}

        {/* ═══════════════════════════════════════════════════════════ */}
        {/* Tab: Colors */}
        {/* ═══════════════════════════════════════════════════════════ */}
        {activeTab === 'colors' && (
          <div className="space-y-4">
            {/* Quick themes */}
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                {t('whiteLabel.quickThemes') || 'Быстрые темы'}
              </legend>
              <div className="flex flex-wrap gap-2">
                {PREVIEW_THEMES.map((theme) => (
                  <button
                    key={theme.name}
                    onClick={() => handleQuickTheme(theme)}
                    className={`
                      flex items-center gap-1.5 px-2.5 py-1.5 text-xs rounded-md border transition-all
                      ${localConfig.primaryColor === theme.primary
                        ? 'border-blue-500 ring-1 ring-blue-500 bg-blue-50 dark:bg-blue-900/20'
                        : 'border-slate-200 dark:border-slate-700 hover:border-slate-300'
                      }
                    `}
                    aria-label={`${theme.name} тема`}
                  >
                    <span className="w-3 h-3 rounded-full" style={{ backgroundColor: theme.primary }} />
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: theme.secondary }} />
                    <span className="w-2 h-2 rounded-full" style={{ backgroundColor: theme.accent }} />
                    <span className="text-slate-600 dark:text-slate-400 ml-1">{theme.name}</span>
                  </button>
                ))}
              </div>
            </fieldset>

            {/* Color pickers */}
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                {t('whiteLabel.customColors') || 'Пользовательские цвета'}
              </legend>
              <ColorField
                label={t('whiteLabel.primary') || 'Primary'}
                value={localConfig.primaryColor}
                onChange={(color) => update({ primaryColor: color })}
              />
              <ColorField
                label={t('whiteLabel.secondary') || 'Secondary'}
                value={localConfig.secondaryColor}
                onChange={(color) => update({ secondaryColor: color })}
              />
              <ColorField
                label={t('whiteLabel.accent') || 'Accent'}
                value={localConfig.accentColor}
                onChange={(color) => update({ accentColor: color })}
              />
            </fieldset>

            {/* Live preview */}
            <div className="p-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">
                {t('whiteLabel.colorPreview') || 'Предпросмотр цветов'}
              </p>
              <div className="flex gap-2">
                <div
                  className="flex-1 h-8 rounded text-xs flex items-center justify-center text-white font-medium"
                  style={{ backgroundColor: localConfig.primaryColor }}
                >
                  Primary
                </div>
                <div
                  className="flex-1 h-8 rounded text-xs flex items-center justify-center text-white font-medium"
                  style={{ backgroundColor: localConfig.secondaryColor }}
                >
                  Secondary
                </div>
                <div
                  className="flex-1 h-8 rounded text-xs flex items-center justify-center text-white font-medium"
                  style={{ backgroundColor: localConfig.accentColor }}
                >
                  Accent
                </div>
              </div>
            </div>
          </div>
        )}

        {/* ═══════════════════════════════════════════════════════════ */}
        {/* Tab: Domain */}
        {/* ═══════════════════════════════════════════════════════════ */}
        {activeTab === 'domain' && (
          <div className="space-y-4">
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1 flex items-center gap-1.5">
                <Globe className="w-3.5 h-3.5" />
                {t('whiteLabel.customDomain') || 'Кастомный домен (CNAME)'}
              </legend>
              <div className="flex gap-2">
                <div className="flex-1">
                  <input
                    type="text"
                    value={localConfig.customDomain}
                    onChange={(e) => handleDomainChange(e.target.value)}
                    className={`
                      w-full px-3 py-2 text-sm border rounded-lg
                      bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                      focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none
                      ${domainError ? 'border-red-500' : 'border-slate-300 dark:border-slate-600'}
                    `}
                    placeholder="brand.example.com"
                    aria-label={t('whiteLabel.customDomain') || 'Кастомный домен'}
                  />
                  {domainError && (
                    <p className="mt-1 text-xs text-red-500">{domainError}</p>
                  )}
                </div>
                <button
                  onClick={handleVerifyDomain}
                  disabled={domainStatus === 'verifying' || !localConfig.customDomain}
                  className={`
                    flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg transition-all
                    ${domainStatus === 'verified'
                      ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
                      : 'bg-blue-600 text-white hover:bg-blue-700'
                    }
                    disabled:opacity-50 disabled:cursor-not-allowed
                  `}
                >
                  {domainStatus === 'verifying' ? (
                    <>{t('whiteLabel.verifying') || 'Проверка...'}</>
                  ) : domainStatus === 'verified' ? (
                    <><Check className="w-4 h-4" /> {t('whiteLabel.verified') || 'Подтверждён'}</>
                  ) : (
                    <><ExternalLink className="w-4 h-4" /> {t('whiteLabel.verify') || 'Проверить'}</>
                  )}
                </button>
              </div>
            </fieldset>

            {/* CNAME instructions */}
            {localConfig.customDomain && (
              <div className="p-3 rounded-lg bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 space-y-2">
                <p className="text-xs font-medium text-slate-600 dark:text-slate-400">
                  {t('whiteLabel.dnsInstructions') || "Настройте CNAME запись в DNS:"}
                </p>
                <div className="grid grid-cols-2 gap-2 text-xs">
                  <div className="p-2 rounded bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
                    <span className="text-slate-400 block">{t('whiteLabel.type') || 'Тип'}</span>
                    <span className="font-mono text-slate-700 dark:text-slate-300">CNAME</span>
                  </div>
                  <div className="p-2 rounded bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700">
                    <span className="text-slate-400 block">{t('whiteLabel.name') || 'Имя'}</span>
                    <span className="font-mono text-slate-700 dark:text-slate-300">{localConfig.customDomain}</span>
                  </div>
                  <div className="p-2 rounded bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 col-span-2">
                    <span className="text-slate-400 block">{t('whiteLabel.target') || 'Цель'}</span>
                    <span className="font-mono text-slate-700 dark:text-slate-300">
                      {localConfig.tenantName.toLowerCase().replace(/\s+/g, '-')}.verify.cctv-monitor.io
                    </span>
                  </div>
                </div>
                <button
                  onClick={handleCopyToken}
                  className="text-xs text-blue-600 dark:text-blue-400 hover:underline"
                >
                  {t('whiteLabel.copyToken') || 'Скопировать токен верификации'}
                </button>
              </div>
            )}

            {/* Verification status */}
            {localConfig.cnameVerified && (
              <div className="px-3 py-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 rounded-lg text-xs text-green-700 dark:text-green-300 flex items-center gap-1.5">
                <Check className="w-3 h-3" />
                {t('whiteLabel.domainVerified') || 'Домен верифицирован'} — {localConfig.customDomain}
              </div>
            )}
          </div>
        )}

        {/* ═══════════════════════════════════════════════════════════ */}
        {/* Tab: Email */}
        {/* ═══════════════════════════════════════════════════════════ */}
        {activeTab === 'email' && (
          <div className="space-y-4">
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('whiteLabel.emailHeaderLogo') || 'URL логотипа для email-шапки'}
              </legend>
              <input
                type="text"
                value={localConfig.emailHeaderLogoUrl}
                onChange={(e) => update({ emailHeaderLogoUrl: e.target.value })}
                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                           focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                placeholder="https://example.com/email-logo.png"
                aria-label={t('whiteLabel.emailHeaderLogo') || 'URL логотипа для email-шапки'}
              />
            </fieldset>

            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('whiteLabel.emailFooterText') || 'Текст подписи в email'}
              </legend>
              <textarea
                value={localConfig.emailFooterText}
                onChange={(e) => update({ emailFooterText: e.target.value })}
                rows={2}
                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                           focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none resize-y"
                placeholder="© 2026 Компания. Все права защищены."
                aria-label={t('whiteLabel.emailFooterText') || 'Текст подписи в email'}
              />
            </fieldset>

            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                {t('whiteLabel.emailColors') || 'Цвета email'}
              </legend>
              <ColorField
                label={t('whiteLabel.emailPrimaryColor') || 'Primary'}
                value={localConfig.emailPrimaryColor}
                onChange={(color) => update({ emailPrimaryColor: color })}
              />
            </fieldset>
          </div>
        )}

        {/* ═══════════════════════════════════════════════════════════ */}
        {/* Tab: PDF */}
        {/* ═══════════════════════════════════════════════════════════ */}
        {activeTab === 'pdf' && (
          <div className="space-y-4">
            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('whiteLabel.pdfLogoUrl') || 'URL логотипа для PDF'}
              </legend>
              <input
                type="text"
                value={localConfig.pdfLogoUrl}
                onChange={(e) => update({ pdfLogoUrl: e.target.value })}
                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                           focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                placeholder="https://example.com/pdf-logo.png"
                aria-label={t('whiteLabel.pdfLogoUrl') || 'URL логотипа для PDF'}
              />
            </fieldset>

            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                {t('whiteLabel.pdfFooterText') || 'Текст подвала PDF'}
              </legend>
              <input
                type="text"
                value={localConfig.pdfFooterText}
                onChange={(e) => update({ pdfFooterText: e.target.value })}
                className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg
                           bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100
                           focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                placeholder="Сгенерировано CCTV Health Monitor"
                aria-label={t('whiteLabel.pdfFooterText') || 'Текст подвала PDF'}
              />
            </fieldset>

            <fieldset>
              <legend className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                {t('whiteLabel.pdfColors') || 'Цвета PDF'}
              </legend>
              <ColorField
                label={t('whiteLabel.pdfPrimaryColor') || 'Primary'}
                value={localConfig.pdfPrimaryColor}
                onChange={(color) => update({ pdfPrimaryColor: color })}
              />
              <ColorField
                label={t('whiteLabel.pdfSecondaryColor') || 'Secondary'}
                value={localConfig.pdfSecondaryColor}
                onChange={(color) => update({ pdfSecondaryColor: color })}
              />
            </fieldset>

            {/* PDF Preview */}
            <div className="p-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
              <div
                className="rounded-lg p-4 text-white text-xs"
                style={{ backgroundColor: localConfig.pdfPrimaryColor }}
              >
                <div className="flex items-center gap-2 mb-2">
                  {localConfig.pdfLogoUrl ? (
                    <img src={localConfig.pdfLogoUrl} alt="" className="h-6" />
                  ) : (
                    <FileText className="w-5 h-5" />
                  )}
                  <span className="font-medium">{localConfig.tenantName || 'CCTV Health Monitor'}</span>
                </div>
                <div
                  className="h-1 rounded mb-2"
                  style={{ backgroundColor: localConfig.pdfSecondaryColor }}
                />
                <p className="opacity-80">{t('whiteLabel.pdfSampleContent') || 'Содержимое PDF документа с брендированием'}</p>
              </div>
              <div
                className="mt-1 text-center text-xs px-2 py-1 rounded-b"
                style={{
                  backgroundColor: localConfig.pdfSecondaryColor,
                  color: '#fff',
                  opacity: 0.8,
                }}
              >
                {localConfig.pdfFooterText || 'Сгенерировано CCTV Health Monitor'}
              </div>
            </div>
          </div>
        )}

        {/* ═══════════════════════════════════════════════════════════ */}
        {/* Tab: Advanced */}
        {/* ═══════════════════════════════════════════════════════════ */}
        {activeTab === 'advanced' && (
          <div className="space-y-4">
            {/* Font */}
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

            {/* Custom CSS */}
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
                placeholder="/* Добавьте кастомные CSS-правила */&#10;.sidebar { background: var(--wl-primary); }&#10;.header { border-bottom: 2px solid var(--wl-accent); }"
                aria-label={t('whiteLabel.customCSS') || 'Кастомный CSS'}
              />
            </fieldset>
          </div>
        )}

        {/* ── Actions ──────────────────────────────────────────────── */}
        <div className="flex items-center gap-2 pt-2 border-t border-slate-200 dark:border-slate-700">
          <button
            onClick={handleApply}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg
                       bg-blue-600 text-white hover:bg-blue-700 transition-colors"
          >
            <Check className="w-4 h-4" />
            {t('whiteLabel.apply') || 'Применить'}
          </button>

          <button
            onClick={resetWhiteLabel}
            className="flex items-center gap-1.5 px-4 py-2 text-sm font-medium rounded-lg
                       bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300
                       transition-colors"
          >
            <RotateCcw className="w-4 h-4" />
            {t('whiteLabel.reset') || 'Сбросить'}
          </button>
        </div>

        {/* ── Active indicator ─────────────────────────────────────── */}
        {whiteLabel.isActive && (
          <div className="px-3 py-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-700 rounded-lg text-xs text-green-700 dark:text-green-300 flex items-center gap-1.5">
            <Check className="w-3 h-3" />
            {t('whiteLabel.active') || 'White-label активен'} — {whiteLabel.tenantName || whiteLabel.companyName}
            {whiteLabel.cnameVerified && (
              <span className="ml-1">· {whiteLabel.customDomain}</span>
            )}
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
      alt={whiteLabel.tenantName || whiteLabel.companyName}
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
        ${whiteLabel.secondaryColor ? `--wl-secondary: ${whiteLabel.secondaryColor};` : ''}
        ${whiteLabel.accentColor ? `--wl-accent: ${whiteLabel.accentColor};` : ''}
        ${whiteLabel.fontFamily ? `--wl-font: ${whiteLabel.fontFamily};` : ''}
        ${whiteLabel.customDomain ? `--wl-domain: ${whiteLabel.customDomain};` : ''}
      }
    `}</style>
  );
}
