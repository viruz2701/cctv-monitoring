import React, { useState, useEffect } from 'react';
import { Save, Bell, Shield, Database, Globe, Mail, Smartphone, Monitor, Lock } from 'lucide-react';
import { Card, CardHeader, CardBody, CardFooter, Button, Input, Select, useToast } from '../components/ui';
import { useSettings } from '../context/DataContext';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { useTranslation } from 'react-i18next';

export function Settings() {
    const { t } = useTranslation();
    const { settings, updateSettings } = useSettings();
    const toast = useToast();

    const [formData, setFormData] = useState(settings);

    useEffect(() => {
        setFormData(settings);
    }, [settings]);

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
            <div className="space-y-6 max-w-4xl">
                {/* General Settings */}
                <Card>
                    <CardHeader>{t('general_settings')}</CardHeader>
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
                        <Button icon={<Save className="w-4 h-4" />} onClick={handleSave}>{t('save_changes')}</Button>
                    </CardFooter>
                </Card>

                {/* Notification Settings */}
                <Card>
                    <CardHeader>{t('notification_preferences')}</CardHeader>
                    <CardBody>
                        <div className="space-y-4">
                            {[
                                { id: 'deviceOffline', icon: Bell, title: t('device_offline_alerts'), desc: t('device_offline_desc') },
                                { id: 'securityAlerts', icon: Shield, title: t('security_alerts'), desc: t('security_alerts_desc') },
                                { id: 'storageWarnings', icon: Database, title: t('storage_warnings'), desc: t('storage_warnings_desc') },
                                { id: 'dailyReports', icon: Mail, title: t('daily_report_email'), desc: t('daily_report_desc') },
                                { id: 'mobilePush', icon: Smartphone, title: t('mobile_push'), desc: t('mobile_push_desc') },
                            ].map((item) => (
                                <div key={item.id} className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg transition-colors">
                                    <div className="flex items-center gap-4">
                                        <div className="p-2 bg-white dark:bg-slate-800 rounded-lg shadow-sm"><item.icon className="w-5 h-5 text-slate-600 dark:text-slate-300" /></div>
                                        <div>
                                            <p className="font-medium text-slate-900 dark:text-white">{item.title}</p>
                                            <p className="text-sm text-slate-500 dark:text-slate-300">{item.desc}</p>
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

                {/* Security Settings */}
                <Card>
                    <CardHeader>{t('security_settings')}</CardHeader>
                    <CardBody>
                        <div className="space-y-6">
                            <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg transition-colors">
                                <div className="flex items-center gap-4">
                                    <div className="p-2 bg-white dark:bg-slate-800 rounded-lg shadow-sm"><Shield className="w-5 h-5 text-slate-600 dark:text-slate-300" /></div>
                                    <div>
                                        <p className="font-medium text-slate-900 dark:text-white">{t('two_factor_auth')}</p>
                                        <p className="text-sm text-slate-500 dark:text-slate-300">{t('enforce_2fa')}</p>
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
                                    <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                                </label>
                            </div>

                            <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg transition-colors">
                                <div className="flex items-center gap-4">
                                    <div className="p-2 bg-white dark:bg-slate-800 rounded-lg shadow-sm"><Lock className="w-5 h-5 text-slate-600 dark:text-slate-300" /></div>
                                    <div>
                                        <p className="font-medium text-slate-900 dark:text-white">{t('password_policy')}</p>
                                        <p className="text-sm text-slate-500 dark:text-slate-300">{t('password_policy_desc')}</p>
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

                {/* System Configuration */}
                <Card>
                    <CardHeader>{t('system_configuration')}</CardHeader>
                    <CardBody>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                            <Input
                                label={t('health_check_interval')}
                                type="number"
                                value={formData.system.healthCheckInterval}
                                onChange={(e) => handleSystemChange('healthCheckInterval', parseInt(e.target.value))}
                            />
                            <Input
                                label={t('session_timeout')}
                                type="number"
                                value={formData.system.sessionTimeout}
                                onChange={(e) => handleSystemChange('sessionTimeout', parseInt(e.target.value))}
                            />
                            <Input
                                label={t('max_recording_gap')}
                                type="number"
                                value={formData.system.maxRecordingGap}
                                onChange={(e) => handleSystemChange('maxRecordingGap', parseInt(e.target.value))}
                            />
                            <Input
                                label={t('alert_threshold')}
                                type="number"
                                value={formData.system.alertThreshold}
                                onChange={(e) => handleSystemChange('alertThreshold', parseInt(e.target.value))}
                            />
                        </div>
                    </CardBody>
                    <CardFooter>
                        <div className="flex gap-3">
                            <Button icon={<Save className="w-4 h-4" />} onClick={handleSave}>{t('save_configuration')}</Button>
                            <Button variant="outline" onClick={() => setFormData(settings)}>{t('reset_defaults')}</Button>
                        </div>
                    </CardFooter>
                </Card>

                {/* Integrations */}
                <Card>
                    <CardHeader>{t('integrations')}</CardHeader>
                    <CardBody>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                            {[
                                { name: 'Slack', status: t('connected'), color: 'emerald' },
                                { name: 'Email SMTP', status: t('connected'), color: 'emerald' },
                                { name: 'PagerDuty', status: t('not_connected'), color: 'slate' },
                            ].map((int) => (
                                <div key={int.name} className="p-4 border border-slate-200 dark:border-slate-700 rounded-lg">
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
            </div>
        </PermissionGuard>
    );
}