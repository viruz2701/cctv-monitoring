import React, { useState, useEffect } from 'react';
import { 
  Save, Bell, Shield, Database, Globe, Mail, Smartphone, Monitor, 
  Lock, Server, Network, Loader2, Radio, AlertCircle, CheckCircle2, 
  Info, Settings as SettingsIcon, Zap, Wifi
} from 'lucide-react';
import { Card, CardHeader, CardBody, CardFooter, Button, Input, Select, useToast, Tabs } from '../components/ui';
import { useSettings } from '../context/DataContext';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';
import type { GB28181Settings } from '../types';

export function Settings() {
  const { t } = useTranslation();
  const { settings, updateSettings, servicesSettings, servicesLoading, updateServicesSettings, saveServicesSettings } = useSettings();
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

  // ─── General Settings Handlers ─────────────────────────────────────
  const handleTopLevelChange = (field: keyof typeof formData, value: any) => {
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

  // ─── Services Settings Handlers ────────────────────────────────────
  const handleServiceChange = (serviceKey: string, field: string, value: any) => {
    if (!servicesSettings) return;
    
    const currentService = servicesSettings[serviceKey as keyof typeof servicesSettings];
    updateServicesSettings({
      ...servicesSettings,
      [serviceKey]: {
        ...currentService,
        [field]: value
      }
    });
  };

  const handleGB28181Change = (field: string, value: any) => {
    if (!servicesSettings?.services_gb28181) return;
    updateServicesSettings({
      ...servicesSettings,
      services_gb28181: {
        ...servicesSettings.services_gb28181,
        [field]: value,
      },
    });
  };

  const handleSaveServices = async () => {
    setServicesSaving(true);
    try {
      await saveServicesSettings();
      toast.success(t('services_settings_saved') || 'Services settings saved and restarted!');
    } catch (error: any) {
      toast.error(error.message || t('services_settings_error') || 'Failed to save services settings');
    } finally {
      setServicesSaving(false);
    }
  };

  // ─── GB28181 Helpers ───────────────────────────────────────────────
  const validateServerID = (id: string): boolean => /^\d{20}$/.test(id);

  const parseGB28181ID = (id: string) => {
    if (id.length !== 20) return null;
    return {
      type: id.substring(0, 2),
      region: id.substring(2, 6),
      industry: id.substring(6, 10),
      network: id.substring(10, 12),
      serial: id.substring(12, 20),
    };
  };

  // ─── Service Toggle Component ──────────────────────────────────────
  const ServiceToggle = ({ 
    serviceKey, 
    icon: Icon, 
    iconColor, 
    title, 
    description 
  }: {
    serviceKey: string;
    icon: React.ElementType;
    iconColor: string;
    title: string;
    description: string;
  }) => {
    const service = servicesSettings?.[serviceKey as keyof typeof servicesSettings];
    if (!service) return null;

    return (
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-lg bg-${iconColor}-50 dark:bg-${iconColor}-900/20`}>
            <Icon className={`w-5 h-5 text-${iconColor}-600 dark:text-${iconColor}-400`} />
          </div>
          <div>
            <h4 className="font-medium text-slate-900 dark:text-white">{title}</h4>
            <p className="text-xs text-slate-500 dark:text-slate-400">{description}</p>
          </div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={service.enabled}
            onChange={(e) => handleServiceChange(serviceKey, 'enabled', e.target.checked)}
          />
          <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
        </label>
      </div>
    );
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
          <div className="space-y-6">
            {/* GENERAL SETTINGS */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <SettingsIcon className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>{t('general_settings')}</span>
              </CardHeader>
              <CardBody>
                <div className="space-y-6">
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Input
                      label={t('organization_name')}
                      value={formData.organizationName}
                      onChange={(e) => handleTopLevelChange('organizationName', e.target.value)}
                    />
                    <Input
                      label={t('system_email')}
                      type="email"
                      value={formData.systemEmail}
                      onChange={(e) => handleTopLevelChange('systemEmail', e.target.value)}
                    />
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <Select
                      label={t('timezone')}
                      options={[
                        { value: 'UTC', label: 'UTC' },
                        { value: 'EST', label: t('eastern_time') },
                        { value: 'PST', label: t('pacific_time') },
                        { value: 'IST', label: t('india_time') },
                      ]}
                      value={formData.timezone}
                      onChange={(e) => handleTopLevelChange('timezone', e.target.value)}
                    />
                    <Select
                      label={t('date_format')}
                      options={[
                        { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
                        { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
                        { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
                      ]}
                      value={formData.dateFormat}
                      onChange={(e) => handleTopLevelChange('dateFormat', e.target.value)}
                    />
                  </div>
                </div>
              </CardBody>
              <CardFooter>
                <Button icon={<Save className="w-4 h-4" />} onClick={handleSave}>
                  {t('save_changes')}
                </Button>
              </CardFooter>
            </Card>

            {/* SYSTEM CONFIGURATION */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Zap className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                <span>{t('system_configuration')}</span>
              </CardHeader>
              <CardBody>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Input
                    label={t('health_check_interval')}
                    type="number"
                    value={formData.system.healthCheckInterval}
                    onChange={(e) => handleSystemChange('healthCheckInterval', parseInt(e.target.value) || 5)}
                    helperText="Minutes between health checks"
                  />
                  <Input
                    label={t('session_timeout')}
                    type="number"
                    value={formData.system.sessionTimeout}
                    onChange={(e) => handleSystemChange('sessionTimeout', parseInt(e.target.value) || 30)}
                    helperText="Auto-logout after inactivity (minutes)"
                  />
                  <Input
                    label={t('max_recording_gap')}
                    type="number"
                    value={formData.system.maxRecordingGap}
                    onChange={(e) => handleSystemChange('maxRecordingGap', parseInt(e.target.value) || 15)}
                    helperText="Alert threshold for recording gaps"
                  />
                  <Input
                    label={t('alert_threshold')}
                    type="number"
                    value={formData.system.alertThreshold}
                    onChange={(e) => handleSystemChange('alertThreshold', parseInt(e.target.value) || 85)}
                    helperText="Percentage threshold for alerts"
                  />
                </div>
              </CardBody>
              <CardFooter>
                <div className="flex gap-3">
                  <Button icon={<Save className="w-4 h-4" />} onClick={handleSave}>
                    {t('save_configuration')}
                  </Button>
                  <Button variant="outline" onClick={() => setFormData(settings)}>
                    {t('reset_defaults')}
                  </Button>
                </div>
              </CardFooter>
            </Card>
          </div>
        )}

        {activeTab === 'notifications' && (
          <Card>
            <CardHeader className="flex items-center gap-2">
              <Bell className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              <span>{t('notification_preferences')}</span>
            </CardHeader>
            <CardBody>
              <div className="space-y-3">
                {[
                  { id: 'deviceOffline', icon: Bell, title: t('device_offline_alerts'), desc: t('device_offline_desc'), color: 'blue' },
                  { id: 'securityAlerts', icon: Shield, title: t('security_alerts'), desc: t('security_alerts_desc'), color: 'red' },
                  { id: 'storageWarnings', icon: Database, title: t('storage_warnings'), desc: t('storage_warnings_desc'), color: 'amber' },
                  { id: 'dailyReports', icon: Mail, title: t('daily_report_email'), desc: t('daily_report_desc'), color: 'emerald' },
                  { id: 'mobilePush', icon: Smartphone, title: t('mobile_push'), desc: t('mobile_push_desc'), color: 'purple' },
                ].map((item) => (
                  <div key={item.id} className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg transition-colors hover:bg-slate-100 dark:hover:bg-slate-800">
                    <div className="flex items-center gap-4">
                      <div className={`p-2 bg-${item.color}-50 dark:bg-${item.color}-900/20 rounded-lg`}>
                        <item.icon className={`w-5 h-5 text-${item.color}-600 dark:text-${item.color}-400`} />
                      </div>
                      <div>
                        <p className="font-medium text-slate-900 dark:text-white">{item.title}</p>
                        <p className="text-sm text-slate-500 dark:text-slate-400">{item.desc}</p>
                      </div>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={formData.notifications[item.id as keyof typeof formData.notifications]}
                        onChange={(e) => handleNotificationChange(item.id as keyof typeof formData.notifications, e.target.checked)}
                      />
                      <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                    </label>
                  </div>
                ))}
              </div>
            </CardBody>
          </Card>
        )}

        {activeTab === 'security' && (
          <Card>
            <CardHeader className="flex items-center gap-2">
              <Shield className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              <span>{t('security_settings')}</span>
            </CardHeader>
            <CardBody>
              <div className="space-y-6">
                <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <div className="flex items-center gap-4">
                    <div className="p-2 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg">
                      <Shield className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                    </div>
                    <div>
                      <p className="font-medium text-slate-900 dark:text-white">{t('two_factor_auth')}</p>
                      <p className="text-sm text-slate-500 dark:text-slate-400">{t('enforce_2fa')}</p>
                    </div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      className="sr-only peer"
                      checked={formData.security?.requires2FA ?? false}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        security: { ...prev.security, requires2FA: e.target.checked }
                      }))}
                    />
                    <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-emerald-500 rounded-full peer peer-checked:bg-emerald-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                  </label>
                </div>

                <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <div className="flex items-center gap-4">
                    <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                      <Lock className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                    </div>
                    <div>
                      <p className="font-medium text-slate-900 dark:text-white">{t('password_policy')}</p>
                      <p className="text-sm text-slate-500 dark:text-slate-400">{t('password_policy_desc')}</p>
                    </div>
                  </div>
                  <div className="w-48">
                    <Select
                      options={[
                        { value: 'basic', label: t('basic_policy') },
                        { value: 'strong', label: t('strong_policy') },
                      ]}
                      value={formData.security?.passwordPolicy ?? 'basic'}
                      onChange={(e) => setFormData(prev => ({
                        ...prev,
                        security: { ...prev.security, passwordPolicy: e.target.value as 'basic' | 'strong' }
                      }))}
                    />
                  </div>
                </div>
              </div>
            </CardBody>
          </Card>
        )}

        {activeTab === 'services' && (
          <div className="space-y-6">

        {/* ═══════════════════════════════════════════════════════════════
            GB/T 28181 SIP SERVER
        ═══════════════════════════════════════════════════════════════ */}
        <Card>
          <CardHeader className="flex items-center gap-2">
            <Radio className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
            <div>
              <span>{t('gb28181_settings') || 'GB/T 28181 (China National Standard)'}</span>
              <p className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">
                {t('gb28181_desc') || 'SIP-based protocol for CCTV interoperability. Used by Hikvision, Dahua, Uniview NVRs.'}
              </p>
            </div>
          </CardHeader>
          <CardBody>
            {servicesLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="w-8 h-8 animate-spin text-indigo-600" />
                <span className="ml-3 text-slate-600 dark:text-slate-400">Loading GB28181 settings...</span>
              </div>
            ) : servicesSettings?.services_gb28181 ? (
              <div className="space-y-6">
                {/* Enable Toggle */}
                <div className="flex items-center justify-between p-4 bg-indigo-50/50 dark:bg-indigo-900/10 rounded-lg border border-indigo-100 dark:border-indigo-800/30">
                  <div className="flex items-center gap-3">
                    <div className="p-2 bg-white dark:bg-slate-800 rounded-lg shadow-sm">
                      <Server className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                    </div>
                    <div>
                      <p className="font-medium text-slate-900 dark:text-white">
                        {t('enable') || 'Enable'} GB28181 SIP Server
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        UDP/TCP Port {servicesSettings.services_gb28181.port}
                      </p>
                    </div>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      className="sr-only peer"
                      checked={servicesSettings.services_gb28181.enabled}
                      onChange={(e) => handleGB28181Change('enabled', e.target.checked)}
                    />
                    <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-indigo-500 rounded-full peer peer-checked:bg-indigo-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                  </label>
                </div>

                {servicesSettings.services_gb28181.enabled && (
                  <>
                    {/* Network & Transport */}
                    <div>
                      <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-3 flex items-center gap-2">
                        <Wifi className="w-4 h-4" />
                        {t('gb28181_network') || 'Network & Transport'}
                      </h4>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <Input
                          label="Bind Host"
                          value={servicesSettings.services_gb28181.host}
                          onChange={(e) => handleGB28181Change('host', e.target.value)}
                          placeholder="0.0.0.0"
                        />
                        <Input
                          label="SIP Port (UDP/TCP)"
                          type="number"
                          value={servicesSettings.services_gb28181.port}
                          onChange={(e) => handleGB28181Change('port', parseInt(e.target.value) || 5060)}
                        />
                      </div>
                    </div>

                    {/* Server Identity */}
                    <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                      <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-3 flex items-center gap-2">
                        <Shield className="w-4 h-4" />
                        {t('gb28181_server_identity') || 'Server Identity'}
                      </h4>

                      <div className="space-y-3">
                        <div>
                          <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">
                            {t('gb28181_server_id') || 'Server Device ID (20 digits)'}
                          </label>
                          <input
                            type="text"
                            maxLength={20}
                            value={servicesSettings.services_gb28181.server_id}
                            onChange={(e) => {
                              const val = e.target.value.replace(/\D/g, '').slice(0, 20);
                              handleGB28181Change('server_id', val);
                            }}
                            className={`w-full px-3.5 py-2.5 text-sm font-mono tracking-wider bg-white dark:bg-slate-900 border rounded-lg focus:outline-none focus:ring-2 ${
                              validateServerID(servicesSettings.services_gb28181.server_id)
                                ? 'border-emerald-300 focus:ring-emerald-500 dark:border-emerald-700'
                                : 'border-red-300 focus:ring-red-500 dark:border-red-700'
                            }`}
                            placeholder="34020000002000000001"
                          />
                          {validateServerID(servicesSettings.services_gb28181.server_id) ? (
                            <div className="flex items-start gap-2 text-xs text-emerald-600 dark:text-emerald-400 mt-2">
                              <CheckCircle2 className="w-3.5 h-3.5 flex-shrink-0 mt-0.5" />
                              <div>
                                <p className="font-medium">Valid GB28181 ID</p>
                                {(() => {
                                  const p = parseGB28181ID(servicesSettings.services_gb28181.server_id);
                                  return p ? (
                                    <p className="text-slate-500 dark:text-slate-400 mt-0.5">
                                      Type: <code className="bg-slate-100 dark:bg-slate-800 px-1 rounded">{p.type}</code> ·
                                      Region: <code className="bg-slate-100 dark:bg-slate-800 px-1 rounded">{p.region}</code> ·
                                      Industry: <code className="bg-slate-100 dark:bg-slate-800 px-1 rounded">{p.industry}</code> ·
                                      Network: <code className="bg-slate-100 dark:bg-slate-800 px-1 rounded">{p.network}</code> ·
                                      Serial: <code className="bg-slate-100 dark:bg-slate-800 px-1 rounded">{p.serial}</code>
                                    </p>
                                  ) : null;
                                })()}
                              </div>
                            </div>
                          ) : (
                            <div className="flex items-center gap-2 text-xs text-red-600 dark:text-red-400 mt-2">
                              <AlertCircle className="w-3.5 h-3.5" />
                              <span>{t('gb28181_invalid_id') || 'Server ID must be exactly 20 digits'} ({servicesSettings.services_gb28181.server_id.length}/20)</span>
                            </div>
                          )}
                        </div>

                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                          <Input
                            label={t('gb28181_server_ip') || 'Public IP / Contact Address'}
                            value={servicesSettings.services_gb28181.server_ip}
                            onChange={(e) => handleGB28181Change('server_ip', e.target.value)}
                            placeholder="auto (from incoming packets)"
                            helperText={t('gb28181_server_ip_help') || 'IP that devices behind NAT will use to reach this server'}
                          />
                          <Input
                            label={t('gb28181_realm') || 'SIP Realm / Domain'}
                            value={servicesSettings.services_gb28181.realm}
                            onChange={(e) => handleGB28181Change('realm', e.target.value)}
                            placeholder="3402000000"
                          />
                        </div>
                      </div>
                    </div>

                    {/* Authentication */}
                    <div>
                      <div className="flex items-center justify-between mb-3">
                        <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider flex items-center gap-2">
                          <Lock className="w-4 h-4" />
                          {t('gb28181_auth') || 'Authentication'}
                        </h4>
                        <label className="relative inline-flex items-center cursor-pointer">
                          <input
                            type="checkbox"
                            className="sr-only peer"
                            checked={servicesSettings.services_gb28181.auth_enabled}
                            onChange={(e) => handleGB28181Change('auth_enabled', e.target.checked)}
                          />
                          <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-indigo-500 rounded-full peer peer-checked:bg-indigo-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                        </label>
                      </div>
                      {servicesSettings.services_gb28181.auth_enabled && (
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                          <Input
                            label={t('gb28181_auth_user') || 'Authentication User'}
                            value={servicesSettings.services_gb28181.auth_user}
                            onChange={(e) => handleGB28181Change('auth_user', e.target.value)}
                          />
                          <Input
                            label={t('gb28181_auth_password') || 'Authentication Password'}
                            type="password"
                            value={servicesSettings.services_gb28181.auth_password}
                            onChange={(e) => handleGB28181Change('auth_password', e.target.value)}
                          />
                        </div>
                      )}
                    </div>

                    {/* Behavior */}
                    <div>
                      <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-3">
                        {t('gb28181_behavior') || 'Behavior'}
                      </h4>
                      <div className="space-y-2">
                        {[
                          {
                            key: 'auto_catalog',
                            title: t('gb28181_auto_catalog') || 'Auto-request device catalog on register',
                            desc: t('gb28181_auto_catalog_desc') || 'Automatically discover cameras connected to NVRs',
                          },
                          {
                            key: 'auto_device_info',
                            title: t('gb28181_auto_device_info') || 'Auto-request device info',
                            desc: t('gb28181_auto_device_info_desc') || 'Query manufacturer, model, firmware on register',
                          },
                          {
                            key: 'log_sip_messages',
                            title: t('gb28181_log_sip') || 'Log raw SIP messages (debug)',
                            desc: 'Log raw SIP packets to parsed_logs table',
                          },
                        ].map((item) => (
                          <div
                            key={item.key}
                            className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700"
                          >
                            <div>
                              <p className="text-sm font-medium text-slate-900 dark:text-white">{item.title}</p>
                              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{item.desc}</p>
                            </div>
                            <label className="relative inline-flex items-center cursor-pointer">
                              <input
                                type="checkbox"
                                className="sr-only peer"
                                checked={
                                  (servicesSettings.services_gb28181?.[
                                    item.key as keyof GB28181Settings
                                  ] ?? false) as boolean
                                }
                                onChange={(e) => handleGB28181Change(item.key, e.target.checked)}
                              />
                              <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-indigo-500 rounded-full peer peer-checked:bg-indigo-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                            </label>
                          </div>
                        ))}
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
                        <Input
                          label={t('gb28181_keepalive_interval') || 'Expected Keepalive Interval (sec)'}
                          type="number"
                          min={10}
                          max={3600}
                          value={servicesSettings.services_gb28181.keepalive_interval}
                          onChange={(e) =>
                            handleGB28181Change('keepalive_interval', parseInt(e.target.value) || 60)
                          }
                        />
                        <Input
                          label={t('gb28181_keepalive_timeout') || 'Offline Timeout (sec)'}
                          type="number"
                          min={30}
                          max={7200}
                          value={servicesSettings.services_gb28181.keepalive_timeout}
                          onChange={(e) =>
                            handleGB28181Change('keepalive_timeout', parseInt(e.target.value) || 180)
                          }
                        />
                        <Input
                          label={t('gb28181_max_sub_channels') || 'Max child devices per NVR'}
                          type="number"
                          min={1}
                          max={1024}
                          value={servicesSettings.services_gb28181.max_sub_channels}
                          onChange={(e) =>
                            handleGB28181Change('max_sub_channels', parseInt(e.target.value) || 64)
                          }
                        />
                      </div>
                    </div>
                  </>
                )}
              </div>
            ) : (
              <div className="text-center py-12 text-slate-500 dark:text-slate-400">
                Failed to load GB28181 settings
              </div>
            )}
          </CardBody>
        </Card>

        {/* ═══════════════════════════════════════════════════════════════
            NETWORK SERVICES & PROTOCOLS
        ═══════════════════════════════════════════════════════════════ */}
        <Card>
          <CardHeader className="flex items-center gap-2">
            <Network className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <div>
              <span>{t('network_services') || 'Network Services & Protocols'}</span>
              <p className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">
                Configure protocol receivers and external service connections
              </p>
            </div>
          </CardHeader>
          <CardBody>
            {servicesLoading ? (
              <div className="flex items-center justify-center py-12">
                <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
                <span className="ml-3 text-slate-600 dark:text-slate-400">
                  {t('loading_services') || 'Loading services configuration...'}
                </span>
              </div>
            ) : servicesSettings ? (
              <div className="space-y-6">
                {/* Syslog */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_syslog"
                    icon={Server}
                    iconColor="blue"
                    title="Syslog Receiver"
                    description="UDP/TCP syslog messages from devices"
                  />
                  {servicesSettings.services_syslog?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="Syslog UDP Port"
                        type="number"
                        value={servicesSettings.services_syslog.udp_port}
                        onChange={(e) => handleServiceChange('services_syslog', 'udp_port', parseInt(e.target.value))}
                      />
                      <Input
                        label="Syslog TCP Port"
                        type="number"
                        value={servicesSettings.services_syslog.tcp_port}
                        onChange={(e) => handleServiceChange('services_syslog', 'tcp_port', parseInt(e.target.value))}
                      />
                    </div>
                  )}
                </div>

                {/* FTP */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_ftp"
                    icon={Database}
                    iconColor="emerald"
                    title="FTP Server"
                    description="Receive snapshots and logs via FTP"
                  />
                  {servicesSettings.services_ftp?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="FTP Port"
                        type="number"
                        value={servicesSettings.services_ftp.port}
                        onChange={(e) => handleServiceChange('services_ftp', 'port', parseInt(e.target.value))}
                      />
                      <Input
                        label="Root Path"
                        value={servicesSettings.services_ftp.root_path}
                        onChange={(e) => handleServiceChange('services_ftp', 'root_path', e.target.value)}
                      />
                      <Input
                        label="Username"
                        value={servicesSettings.services_ftp.user}
                        onChange={(e) => handleServiceChange('services_ftp', 'user', e.target.value)}
                      />
                      <Input
                        label="Password"
                        type="password"
                        value={servicesSettings.services_ftp.password}
                        onChange={(e) => handleServiceChange('services_ftp', 'password', e.target.value)}
                      />
                    </div>
                  )}
                </div>

                {/* SNMP */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_snmp"
                    icon={Shield}
                    iconColor="amber"
                    title="SNMP Trap Receiver"
                    description="Receive SNMP traps from network devices"
                  />
                  {servicesSettings.services_snmp?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="SNMP Port"
                        type="number"
                        value={servicesSettings.services_snmp.port}
                        onChange={(e) => handleServiceChange('services_snmp', 'port', parseInt(e.target.value))}
                      />
                      <Select
                        label="SNMP Version"
                        options={[
                          { value: 'v1', label: 'SNMP v1' },
                          { value: 'v2c', label: 'SNMP v2c' },
                          { value: 'v3', label: 'SNMP v3' },
                        ]}
                        value={servicesSettings.services_snmp.version}
                        onChange={(e) => handleServiceChange('services_snmp', 'version', e.target.value)}
                      />
                      <Input
                        label="Community String"
                        value={servicesSettings.services_snmp.community}
                        onChange={(e) => handleServiceChange('services_snmp', 'community', e.target.value)}
                      />
                    </div>
                  )}
                </div>

                {/* HTTP Log Receiver */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_http"
                    icon={Globe}
                    iconColor="purple"
                    title="HTTP Log Receiver"
                    description="Receive logs via HTTP POST"
                  />
                  {servicesSettings.services_http?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="HTTP Port"
                        type="number"
                        value={servicesSettings.services_http.port}
                        onChange={(e) => handleServiceChange('services_http', 'port', parseInt(e.target.value))}
                      />
                    </div>
                  )}
                </div>

                {/* Dahua Protocol */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_dahua"
                    icon={Monitor}
                    iconColor="cyan"
                    title="Dahua Private Protocol"
                    description="Proprietary Dahua protocol for events"
                  />
                  {servicesSettings.services_dahua?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="Ports (comma-separated)"
                        value={servicesSettings.services_dahua.ports.join(', ')}
                        onChange={(e) => {
                          const ports = e.target.value.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p));
                          handleServiceChange('services_dahua', 'ports', ports);
                        }}
                        placeholder="37777, 37778"
                      />
                    </div>
                  )}
                </div>

                {/* Hisilicon Protocol */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_hisilicon"
                    icon={Monitor}
                    iconColor="indigo"
                    title="Hisilicon Protocol"
                    description="Hisilicon-based devices events"
                  />
                  {servicesSettings.services_hisilicon?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="Port"
                        type="number"
                        value={servicesSettings.services_hisilicon.port}
                        onChange={(e) => handleServiceChange('services_hisilicon', 'port', parseInt(e.target.value))}
                      />
                    </div>
                  )}
                </div>

                {/* TVT Protocol */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_tvt"
                    icon={Monitor}
                    iconColor="pink"
                    title="TVT Protocol"
                    description="TVT-based devices events"
                  />
                  {servicesSettings.services_tvt?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="Port"
                        type="number"
                        value={servicesSettings.services_tvt.port}
                        onChange={(e) => handleServiceChange('services_tvt', 'port', parseInt(e.target.value))}
                      />
                    </div>
                  )}
                </div>

                {/* Legacy SIP */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey="services_sip"
                    icon={Globe}
                    iconColor="teal"
                    title="SIP / GB28181 (Legacy)"
                    description="SIP signaling for GB28181 devices"
                  />
                  {servicesSettings.services_sip?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      <Input
                        label="SIP Port"
                        type="number"
                        value={servicesSettings.services_sip.port}
                        onChange={(e) => handleServiceChange('services_sip', 'port', parseInt(e.target.value))}
                      />
                      <Input
                        label="Host"
                        value={servicesSettings.services_sip.host}
                        onChange={(e) => handleServiceChange('services_sip', 'host', e.target.value)}
                        placeholder="0.0.0.0"
                      />
                    </div>
                  )}
                </div>

                {/* P2P Gateway */}
                <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="p-2 bg-orange-50 dark:bg-orange-900/20 rounded-lg">
                      <Network className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                    </div>
                    <div>
                      <h4 className="font-medium text-slate-900 dark:text-white">P2P Gateway</h4>
                      <p className="text-xs text-slate-500 dark:text-slate-400">Connection to P2P gateway service</p>
                    </div>
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                    <Input
                      label="Gateway URL"
                      value={servicesSettings.services_p2p_gateway.url}
                      onChange={(e) => handleServiceChange('services_p2p_gateway', 'url', e.target.value)}
                      placeholder="http://localhost:8082"
                    />
                    <Input
                      label="API Key"
                      type="password"
                      value={servicesSettings.services_p2p_gateway.api_key}
                      onChange={(e) => handleServiceChange('services_p2p_gateway', 'api_key', e.target.value)}
                    />
                  </div>
                </div>

                {/* Save Button */}
                <div className="flex justify-end pt-4 border-t border-slate-200 dark:border-slate-700">
                  <Button
                    icon={servicesSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                    onClick={handleSaveServices}
                    disabled={servicesSaving}
                  >
                    {servicesSaving ? (t('saving') || 'Saving...') : (t('save_and_restart_services') || 'Save & Restart Services')}
                  </Button>
                </div>
              </div>
            ) : (
              <div className="text-center py-12 text-slate-500 dark:text-slate-400">
                {t('failed_to_load_services') || 'Failed to load services configuration'}
              </div>
            )}
          </CardBody>
        </Card>
          </div>
        )}

        {activeTab === 'integrations' && (
          <Card>
            <CardHeader className="flex items-center gap-2">
              <Globe className="w-5 h-5 text-slate-600 dark:text-slate-400" />
              <span>{t('integrations')}</span>
            </CardHeader>
            <CardBody>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {[
                  { name: 'Slack', status: t('connected'), color: 'emerald' },
                  { name: 'Email SMTP', status: t('connected'), color: 'emerald' },
                  { name: 'PagerDuty', status: t('not_connected'), color: 'slate' },
                ].map((int) => (
                  <div key={int.name} className="p-4 border border-slate-200 dark:border-slate-700 rounded-lg hover:shadow-md transition-shadow">
                    <div className="flex items-center justify-between mb-2">
                      <h4 className="font-medium text-slate-900 dark:text-white">{int.name}</h4>
                      <span className={`w-2 h-2 rounded-full ${int.color === 'emerald' ? 'bg-emerald-500' : 'bg-slate-300'}`} />
                    </div>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mb-3">{int.status}</p>
                    <Button
                      variant="outline"
                      size="sm"
                      fullWidth
                      onClick={() => toast.info(`${t('configure')} ${int.name} ${t('coming_soon')}`)}
                    >
                      {t('configure')}
                    </Button>
                  </div>
                ))}
              </div>
            </CardBody>
          </Card>
        )}
      </div>
    </PermissionGuard>
  );
}