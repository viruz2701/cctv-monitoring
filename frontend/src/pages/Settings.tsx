// ═══════════════════════════════════════════════════════════════════════
// Settings — Tabbed settings page with deep linking & RBAC
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Bell, Shield, Globe, Settings as SettingsIcon, Lock, Server, Palette, FileText } from '../components/ui/Icons';
import { Tabs, useToast } from '../components/ui';
import { useSettingsStore } from '../store/settingsStore';
import { useServicesSettings, useServicesStatus, useUpdateServicesSettings } from '../hooks/useApiQuery';
import type { ServicesSettings as ServicesSettingsAPI } from '../services/api';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useAuth } from '../hooks/useAuth';
import { useTranslation } from 'react-i18next';
import { GeneralSettings } from './settings/GeneralSettings';
import { NotificationsSettings } from './settings/NotificationsSettings';
import { SecuritySettings } from './settings/SecuritySettings';
import { ServicesSettings } from './settings/ServicesSettings';
import { IntegrationsSettings } from './settings/IntegrationsSettings';
import { SSOSettings } from './settings/SSOSettings';
import { AppearanceSettings } from './settings/AppearanceSettings';
import { LoggingSettings, type LoggingConfig } from './settings/LoggingSettings';
import { ThemeCustomizer } from '../components/ui/ThemeCustomizer';
import { AtlasCMSPanel } from './settings/AtlasCMSPanel';

export function Settings() {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { tab = 'general' } = useParams<{ tab?: string }>();
  const navigate = useNavigate();
  const { settings, updateSettings } = useSettingsStore();
  const { data: servicesSettingsRaw, isLoading: servicesLoading } = useServicesSettings();
  const { data: servicesStatusRaw, isLoading: servicesStatusLoading, refetch: refreshServicesStatus } = useServicesStatus();
  const servicesMutation = useUpdateServicesSettings();

  // Services draft — local editing state (was in old SettingsContext bridge)
  const [servicesDraft, setServicesDraft] = useState<ServicesSettingsAPI | null>(null);
  useEffect(() => {
    if (servicesSettingsRaw && !servicesDraft) {
      setServicesDraft(servicesSettingsRaw);
    }
  }, [servicesSettingsRaw]);

  const servicesSettings = servicesDraft ?? servicesSettingsRaw ?? null;
  const servicesStatus = servicesStatusRaw ?? {};
  const updateServicesSettings = (updates: Partial<ServicesSettingsAPI>) => {
    setServicesDraft(prev => prev ? { ...prev, ...updates } : null);
  };
  const saveServicesSettings = async () => {
    if (servicesDraft) {
      await servicesMutation.mutateAsync(servicesDraft);
    }
  };
  const toast = useToast();

  const [formData, setFormData] = useState(settings);
  const [servicesSaving, setServicesSaving] = useState(false);
  const [themeCustomizerOpen, setThemeCustomizerOpen] = useState(false);
  const [loggingConfig, setLoggingConfig] = useState<LoggingConfig>({ level: 'info', retention_days: 30 });

  const isAdmin = user?.role === 'admin';

  const settingsTabs = [
    { id: 'general', label: t('general'), icon: <SettingsIcon className="w-4 h-4" /> },
    { id: 'appearance', label: t('appearance') || 'Appearance', icon: <Palette className="w-4 h-4" /> },
    { id: 'notifications', label: t('notifications'), icon: <Bell className="w-4 h-4" /> },
    { id: 'logging', label: t('logging') || 'Logging', icon: <FileText className="w-4 h-4" /> },
    ...(isAdmin ? [
      { id: 'security', label: t('security'), icon: <Shield className="w-4 h-4" /> },
      { id: 'services', label: t('services') || 'Services', icon: <Server className="w-4 h-4" /> },
      { id: 'integrations', label: t('integrations'), icon: <Globe className="w-4 h-4" /> },
      { id: 'sso', label: t('sso') || 'SSO', icon: <Lock className="w-4 h-4" /> },
    ] : []),
  ];

  const validTabs = settingsTabs.map((t) => t.id);
  const activeTab = validTabs.includes(tab) ? tab : 'general';

  useEffect(() => { setFormData(settings); }, [settings]);

  const handleTabChange = (tabId: string) => navigate(`/settings/${tabId}`, { replace: true });
  const handleTopLevelChange = (field: string, value: any) => setFormData(prev => ({ ...prev, [field]: value }));
  const handleNotificationChange = (field: keyof typeof formData.notifications, value: boolean) => setFormData(prev => ({ ...prev, notifications: { ...prev.notifications, [field]: value } }));
  const handleSystemChange = (field: keyof typeof formData.system, value: number) => setFormData(prev => ({ ...prev, system: { ...prev.system, [field]: value } }));
  const handleSave = () => { updateSettings(formData); toast.success(t('settings_saved') || 'Settings saved successfully!'); };
  const handleServiceChange = (serviceKey: string, field: string, value: any) => {
    if (!servicesSettings) return;
    updateServicesSettings({ [serviceKey]: { ...servicesSettings[serviceKey as keyof typeof servicesSettings], [field]: value } });
  };
  const handleGB28181Change = (field: string, value: any) => {
    if (!servicesSettings?.services_gb28181) return;
    updateServicesSettings({ services_gb28181: { ...servicesSettings.services_gb28181, [field]: value } });
  };
  const handleSaveServices = async () => {
    setServicesSaving(true);
    try { await saveServicesSettings(); toast.success(t('services_saved') || 'Services settings saved!'); }
    catch (e: any) { toast.error(e.message || 'Failed to save services settings'); }
    finally { setServicesSaving(false); }
  };
  const validateServerID = (id: string) => /^\d{20}$/.test(id);
  const parseGB28181ID = (id: string) => id.length !== 20 ? null : { type: id.substring(0, 2), region: id.substring(2, 6), industry: id.substring(6, 8), network: id.substring(8, 10), serial: id.substring(10, 20) };

  return (
    <PermissionGuard requiredRole={['admin', 'manager']} fallback={
      <div className="flex flex-col items-center justify-center h-96">
        <Lock className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" />
        <h2 className="text-xl font-bold text-slate-900 dark:text-white">{t('access_denied')}</h2>
        <p className="text-slate-500 dark:text-slate-400 mt-2">{t('no_permission_settings')}</p>
      </div>
    }>
      <div className="max-w-5xl">
        <Tabs tabs={settingsTabs} activeTab={activeTab} onChange={handleTabChange} variant="pills" className="mb-6"><div /></Tabs>

        {activeTab === 'general' && (
          <GeneralSettings formData={formData} onTopLevelChange={handleTopLevelChange} onSystemChange={handleSystemChange} onSave={handleSave} onReset={() => setFormData(settings)} />
        )}

        {activeTab === 'appearance' && <AppearanceSettings onOpenCustomizer={() => setThemeCustomizerOpen(true)} />}

        {activeTab === 'logging' && <LoggingSettings logging={loggingConfig} onChange={setLoggingConfig} />}

        {activeTab === 'notifications' && (
          <NotificationsSettings notifications={formData.notifications} onNotificationChange={handleNotificationChange} />
        )}

        {isAdmin && activeTab === 'security' && (
          <SecuritySettings security={formData.security} onChange={(security) => setFormData(prev => ({ ...prev, security }))} />
        )}

        {isAdmin && activeTab === 'services' && (
          <ServicesSettings servicesSettings={servicesSettings} servicesLoading={servicesLoading} servicesSaving={servicesSaving} servicesStatus={servicesStatus} servicesStatusLoading={servicesStatusLoading} onGB28181Change={handleGB28181Change} onServiceChange={handleServiceChange} onSave={handleSaveServices} onRefreshStatus={refreshServicesStatus} validateServerID={validateServerID} parseGB28181ID={parseGB28181ID} />
        )}

        {isAdmin && activeTab === 'integrations' && (
          <IntegrationsSettings><AtlasCMSPanel /></IntegrationsSettings>
        )}

        {isAdmin && activeTab === 'sso' && <SSOSettings />}
      </div>

      <ThemeCustomizer isOpen={themeCustomizerOpen} onClose={() => setThemeCustomizerOpen(false)} />
    </PermissionGuard>
  );
}
