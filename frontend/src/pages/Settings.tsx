import React, { useState, useEffect } from 'react';
import {
  Bell, Shield, Globe, Settings as SettingsIcon, Lock, Server,
  Palette,
} from 'lucide-react';
import { Tabs, useToast } from '../components/ui';
import { useSettings } from '../context/DataContext';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';
import type { GB28181Settings } from '../types';

import { GeneralSettings } from './settings/GeneralSettings';
import { NotificationsSettings } from './settings/NotificationsSettings';
import { SecuritySettings } from './settings/SecuritySettings';
import { ServicesSettings } from './settings/ServicesSettings';
import { IntegrationsSettings } from './settings/IntegrationsSettings';
import { SSOSettings } from './settings/SSOSettings';
import { ThemeCustomizer } from '../components/ui/ThemeCustomizer';

// ── Atlas CMMS Integration Panel (встроенный мини-компонент) ──────────
import { AtlasCMSPanel } from './settings/AtlasCMSPanel';

export function Settings() {
  const { t } = useTranslation();
  const { settings, updateSettings, servicesSettings, servicesLoading, servicesStatus, servicesStatusLoading, updateServicesSettings, saveServicesSettings, refreshServicesStatus } = useSettings();
  const toast = useToast();

  const [formData, setFormData] = useState(settings);
  const [servicesSaving, setServicesSaving] = useState(false);
  const [activeTab, setActiveTab] = useState('general');
  const [themeCustomizerOpen, setThemeCustomizerOpen] = useState(false);

  const settingsTabs = [
    { id: 'general', label: t('general'), icon: <SettingsIcon className="w-4 h-4" /> },
    { id: 'appearance', label: t('appearance') || 'Appearance', icon: <Palette className="w-4 h-4" /> },
    { id: 'notifications', label: t('notifications'), icon: <Bell className="w-4 h-4" /> },
    { id: 'security', label: t('security'), icon: <Shield className="w-4 h-4" /> },
    { id: 'services', label: t('services') || 'Services', icon: <Server className="w-4 h-4" /> },
    { id: 'integrations', label: t('integrations'), icon: <Globe className="w-4 h-4" /> },
    { id: 'sso', label: t('sso') || 'SSO', icon: <Lock className="w-4 h-4" /> },
  ];

  useEffect(() => {
    setFormData(settings);
  }, [settings]);

  const handleTopLevelChange = (field: string, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  const handleNotificationChange = (field: keyof typeof formData.notifications, value: boolean) => {
    setFormData(prev => ({
      ...prev,
      notifications: { ...prev.notifications, [field]: value }
    }));
  };

  const handleSystemChange = (field: keyof typeof formData.system, value: number) => {
    setFormData(prev => ({
      ...prev,
      system: { ...prev.system, [field]: value }
    }));
  };

  const handleSave = () => {
    updateSettings(formData);
    toast.success(t('settings_saved') || 'Settings saved successfully!');
  };

  const handleServiceChange = (serviceKey: string, field: string, value: any) => {
    if (!servicesSettings) return;
    const currentService = servicesSettings[serviceKey as keyof typeof servicesSettings];
    updateServicesSettings({
      [serviceKey]: { ...currentService, [field]: value }
    });
  };

  const handleGB28181Change = (field: string, value: any) => {
    if (!servicesSettings?.services_gb28181) return;
    updateServicesSettings({
      services_gb28181: { ...servicesSettings.services_gb28181, [field]: value }
    });
  };

  const handleSaveServices = async () => {
    setServicesSaving(true);
    try {
      await saveServicesSettings();
      toast.success(t('services_saved') || 'Services settings saved!');
    } catch (e: any) {
      toast.error(e.message || 'Failed to save services settings');
    } finally {
      setServicesSaving(false);
    }
  };

  const validateServerID = (id: string) => /^\d{20}$/.test(id);
  const parseGB28181ID = (id: string) => {
    if (id.length !== 20) return null;
    return {
      type: id.substring(0, 2),
      region: id.substring(2, 6),
      industry: id.substring(6, 8),
      network: id.substring(8, 10),
      serial: id.substring(10, 20),
    };
  };

  return (
    <PermissionGuard
      requiredRole={['admin', 'manager']}
      fallback={
        <div className="flex flex-col items-center justify-center h-96">
          <Lock className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" />
          <h2 className="text-xl font-bold text-slate-900 dark:text-white">{t('access_denied')}</h2>
          <p className="text-slate-500 dark:text-slate-400 mt-2">{t('no_permission_settings')}</p>
        </div>
      }
    >
      <div className="max-w-5xl">
        <Tabs tabs={settingsTabs} activeTab={activeTab} onChange={setActiveTab} variant="pills" className="mb-6">
          <div />
        </Tabs>

        {activeTab === 'general' && (
          <GeneralSettings
            formData={formData}
            onTopLevelChange={handleTopLevelChange}
            onSystemChange={handleSystemChange}
            onSave={handleSave}
            onReset={() => setFormData(settings)}
          />
        )}

        {activeTab === 'appearance' && (
          <AppearanceSettings onOpenCustomizer={() => setThemeCustomizerOpen(true)} />
        )}

        {activeTab === 'notifications' && (
          <NotificationsSettings
            notifications={formData.notifications}
            onNotificationChange={handleNotificationChange}
          />
        )}

        {activeTab === 'security' && (
          <SecuritySettings
            security={formData.security}
            onChange={(security) => setFormData(prev => ({ ...prev, security }))}
          />
        )}

        {activeTab === 'services' && (
          <ServicesSettings
            servicesSettings={servicesSettings}
            servicesLoading={servicesLoading}
            servicesSaving={servicesSaving}
            servicesStatus={servicesStatus}
            servicesStatusLoading={servicesStatusLoading}
            onGB28181Change={handleGB28181Change}
            onServiceChange={handleServiceChange}
            onSave={handleSaveServices}
            onRefreshStatus={refreshServicesStatus}
            validateServerID={validateServerID}
            parseGB28181ID={parseGB28181ID}
          />
        )}

        {activeTab === 'integrations' && (
          <IntegrationsSettings>
            <AtlasCMSPanel />
          </IntegrationsSettings>
        )}

        {activeTab === 'sso' && (
          <SSOSettings />
        )}
      </div>

      {/* Theme Customizer Modal */}
      <ThemeCustomizer isOpen={themeCustomizerOpen} onClose={() => setThemeCustomizerOpen(false)} />
    </PermissionGuard>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Appearance Settings — UX-14.3.4
// ═══════════════════════════════════════════════════════════════════════

import { Card, CardHeader, CardBody, Button } from '../components/ui';
import { useThemeStore, PRESET_COLORS, type ThemePreset } from '../store/themeStore';

function AppearanceSettings({ onOpenCustomizer }: { onOpenCustomizer: () => void }) {
  const { t } = useTranslation();
  const { theme, preset, isDark, primary, accent, radius, setTheme, setPreset } = useThemeStore();

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
