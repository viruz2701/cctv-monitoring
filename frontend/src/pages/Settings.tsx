import React, { useState, useEffect } from 'react';
import {
  Bell, Shield, Globe, Settings as SettingsIcon, Lock, Server,
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

// ── Atlas CMMS Integration Panel (встроенный мини-компонент) ──────────
import { AtlasCMSPanel } from './settings/AtlasCMSPanel';

export function Settings() {
  const { t } = useTranslation();
  const { settings, updateSettings, servicesSettings, servicesLoading, servicesStatus, servicesStatusLoading, updateServicesSettings, saveServicesSettings, refreshServicesStatus } = useSettings();
  const toast = useToast();

  const [formData, setFormData] = useState(settings);
  const [servicesSaving, setServicesSaving] = useState(false);
  const [activeTab, setActiveTab] = useState('general');

  const settingsTabs = [
    { id: 'general', label: t('general'), icon: <SettingsIcon className="w-4 h-4" /> },
    { id: 'notifications', label: t('notifications'), icon: <Bell className="w-4 h-4" /> },
    { id: 'security', label: t('security'), icon: <Shield className="w-4 h-4" /> },
    { id: 'services', label: t('services') || 'Services', icon: <Server className="w-4 h-4" /> },
    { id: 'integrations', label: t('integrations'), icon: <Globe className="w-4 h-4" /> },
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
      </div>
    </PermissionGuard>
  );
}