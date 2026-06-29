import React, { useState, useEffect } from 'react';
import { Bell, Shield, Database, Mail, MessageCircle, Smartphone, Loader, CheckCircle } from '../components/ui/Icons';
import { Card, CardHeader, CardBody, Input, Button, useToast } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import { api } from '../../services/api';
import type { AppSettings } from '../../types';

interface Props {
  notifications: AppSettings['notifications'];
  onNotificationChange: (field: keyof AppSettings['notifications'], value: any) => void;
}

export const NotificationsSettings: React.FC<Props> = ({ notifications, onNotificationChange }) => {
  const { t } = useTranslation();
  const toast = useToast();

  // Telegram state
  const [tgStatus, setTgStatus] = useState<{ linked: boolean; alerts: boolean; tfa: boolean } | null>(null);
  const [tgLoading, setTgLoading] = useState(false);

  // Email state
  const [emailAddress, setEmailAddress] = useState('');

  useEffect(() => {
    loadTelegramStatus();
  }, []);

  const loadTelegramStatus = async () => {
    try {
      setTgLoading(true);
      const status = await api.getTelegramStatus();
      setTgStatus(status);
    } catch {
      // silently fail
    } finally {
      setTgLoading(false);
    }
  };

  const handleTelegramToggle = async (field: 'alerts', value: boolean) => {
    if (!tgStatus) return;
    try {
      const newSettings = { ...tgStatus, [field]: value };
      await api.updateTelegramSettings({ alerts: newSettings.alerts, tfa: newSettings.tfa });
      setTgStatus(newSettings);
      toast.success(t('telegram_settings_updated') || 'Telegram settings updated');
    } catch (err: any) {
      toast.error(err.message || t('telegram_update_error') || 'Failed to update Telegram settings');
    }
  };

  const items = [
    { id: 'deviceOffline' as const, icon: Bell, title: t('device_offline_alerts'), desc: t('device_offline_desc'), color: 'blue' },
    { id: 'securityAlerts' as const, icon: Shield, title: t('security_alerts'), desc: t('security_alerts_desc'), color: 'red' },
    { id: 'storageWarnings' as const, icon: Database, title: t('storage_warnings'), desc: t('storage_warnings_desc'), color: 'amber' },
    { id: 'dailyReports' as const, icon: Mail, title: t('daily_report_email'), desc: t('daily_report_desc'), color: 'emerald' },
  ];

  return (
    <div className="space-y-6">
      {/* NOTIFICATION PREFERENCES */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Bell className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <span>{t('notification_preferences')}</span>
        </CardHeader>
        <CardBody>
          <div className="space-y-3">
            {items.map((item) => (
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
                    checked={notifications[item.id]}
                    onChange={(e) => onNotificationChange(item.id, e.target.checked)}
                  />
                  <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                </label>
              </div>
            ))}
          </div>
        </CardBody>
      </Card>

      {/* EMAIL NOTIFICATIONS */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Mail className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          <span>{t('email_notifications') || 'Email Notifications'}</span>
        </CardHeader>
        <CardBody>
          <div className="max-w-md">
            <Input
              label={t('notification_email') || 'Notification Email'}
              type="email"
              placeholder={t('enter_email') || 'Enter your email address'}
              value={emailAddress}
              onChange={(e) => setEmailAddress(e.target.value)}
            />
            <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
              {t('email_notifications_desc') || 'Receive alert summaries and daily reports via email'}
            </p>
          </div>
        </CardBody>
      </Card>

      {/* SMS NOTIFICATIONS (RocketSMS) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Smartphone className="w-5 h-5 text-violet-600 dark:text-violet-400" />
          <span>{t('sms_notifications') || 'SMS Уведомления'}</span>
        </CardHeader>
        <CardBody>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
              <div>
                <p className="font-medium text-slate-900 dark:text-white">
                  {t('sms_enabled') || 'SMS уведомления'}
                </p>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {t('sms_enabled_desc') || 'Отправка SMS при нарушении SLA через RocketSMS'}
                </p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  className="sr-only peer"
                  checked={notifications.smsEnabled}
                  onChange={(e) => onNotificationChange('smsEnabled', e.target.checked)}
                />
                <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-violet-500 rounded-full peer peer-checked:bg-violet-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
              </label>
            </div>

            {notifications.smsEnabled && (
              <>
                <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <div>
                    <p className="font-medium text-slate-900 dark:text-white">
                      {t('sms_critical_only') || 'Только критические'}
                    </p>
                    <p className="text-sm text-slate-500 dark:text-slate-400">
                      {t('sms_critical_only_desc') || 'Отправлять SMS только для CRITICAL/HIGH приоритетов'}
                    </p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer">
                    <input
                      type="checkbox"
                      className="sr-only peer"
                      checked={notifications.smsForCriticalOnly}
                      onChange={(e) => onNotificationChange('smsForCriticalOnly', e.target.checked)}
                    />
                    <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-violet-500 rounded-full peer peer-checked:bg-violet-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                  </label>
                </div>

                <div className="p-4 bg-violet-50 dark:bg-violet-900/20 rounded-lg space-y-3">
                  <p className="text-sm font-medium text-violet-800 dark:text-violet-200">
                    {t('rocketsms_settings') || 'Настройки RocketSMS'}
                  </p>
                  <Input
                    label={t('rocketsms_login') || 'Логин'}
                    value={notifications.rocketsms?.login || ''}
                    onChange={(e) => onNotificationChange('rocketsms', { ...notifications.rocketsms, login: e.target.value })}
                    placeholder="your_rocketsms_login"
                  />
                  <div className="grid grid-cols-2 gap-3">
                    <Input
                      label={t('rocketsms_sender') || 'Отправитель (SMS)'}
                      value={notifications.rocketsms?.sender || 'CCTV'}
                      onChange={(e) => onNotificationChange('rocketsms', { ...notifications.rocketsms, sender: e.target.value })}
                      placeholder="CCTV"
                    />
                    <Input
                      label={t('rocketsms_api_url') || 'API URL'}
                      value={notifications.rocketsms?.apiUrl || 'https://api.rocketsms.by'}
                      onChange={(e) => onNotificationChange('rocketsms', { ...notifications.rocketsms, apiUrl: e.target.value })}
                      placeholder="https://api.rocketsms.by"
                    />
                  </div>
                  <p className="text-xs text-violet-600 dark:text-violet-400">
                    {t('rocketsms_password_hint') || 'Пароль RocketSMS настраивается через переменную окружения ROCKET_SMS_PASSWORD'}
                  </p>
                </div>
              </>
            )}
          </div>
        </CardBody>
      </Card>

      {/* EMAIL FOR MANAGERS */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Mail className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          <span>{t('email_for_managers') || 'Email для менеджеров'}</span>
        </CardHeader>
        <CardBody>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
              <div>
                <p className="font-medium text-slate-900 dark:text-white">
                  {t('email_managers_enabled') || 'Уведомления менеджеров'}
                </p>
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {t('email_managers_desc') || 'Отправлять email менеджерам при критическом SLA и breach'}
                </p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  className="sr-only peer"
                  checked={notifications.emailForManagers}
                  onChange={(e) => onNotificationChange('emailForManagers', e.target.checked)}
                />
                <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-emerald-500 rounded-full peer peer-checked:bg-emerald-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
              </label>
            </div>
            <p className="text-xs text-slate-500 dark:text-slate-400">
              {t('email_managers_hint') || 'Email менеджеров настраивается в профилях пользователей (роль manager/owner)'}
            </p>

            {(notifications.emailForManagers || notifications.smtp?.host) && (
              <div className="p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg space-y-3">
                <p className="text-sm font-medium text-emerald-800 dark:text-emerald-200">
                  {t('smtp_settings') || 'Настройки SMTP'}
                </p>
                <div className="grid grid-cols-2 gap-3">
                  <Input
                    label={t('smtp_host') || 'SMTP Host'}
                    value={notifications.smtp?.host || ''}
                    onChange={(e) => onNotificationChange('smtp', { ...notifications.smtp, host: e.target.value })}
                    placeholder="smtp.gmail.com"
                  />
                  <Input
                    label={t('smtp_port') || 'SMTP Port'}
                    type="number"
                    value={notifications.smtp?.port || 587}
                    onChange={(e) => onNotificationChange('smtp', { ...notifications.smtp, port: parseInt(e.target.value) || 587 })}
                    placeholder="587"
                  />
                </div>
                <Input
                  label={t('smtp_user') || 'SMTP User'}
                  value={notifications.smtp?.user || ''}
                  onChange={(e) => onNotificationChange('smtp', { ...notifications.smtp, user: e.target.value })}
                  placeholder="user@example.com"
                />
                <div className="grid grid-cols-2 gap-3">
                  <Input
                    label={t('smtp_from') || 'From (отправитель)'}
                    value={notifications.smtp?.from || ''}
                    onChange={(e) => onNotificationChange('smtp', { ...notifications.smtp, from: e.target.value })}
                    placeholder="cctv@example.com"
                  />
                  <div className="flex items-end pb-2">
                    <p className="text-xs text-emerald-600 dark:text-emerald-400">
                      {t('smtp_password_hint') || 'Пароль SMTP через SMTP_PASSWORD'}
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </CardBody>
      </Card>

      {/* TELEGRAM INTEGRATION */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <MessageCircle className="w-5 h-5 text-sky-600 dark:text-sky-400" />
          <span>{t('telegram_integration') || 'Telegram Integration'}</span>
          {tgStatus?.linked && (
            <span className="ml-2 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
              <CheckCircle className="w-3 h-3" />
              {t('connected') || 'Connected'}
            </span>
          )}
        </CardHeader>
        <CardBody>
          {tgLoading ? (
            <div className="flex items-center justify-center py-4">
              <Loader className="w-5 h-5 animate-spin text-sky-500" />
              <span className="ml-2 text-sm text-slate-500">{t('loading')}</span>
            </div>
          ) : tgStatus?.linked ? (
            <div className="space-y-4">
              <div className="flex items-center gap-3 p-4 bg-sky-50 dark:bg-sky-900/20 rounded-lg">
                <MessageCircle className="w-5 h-5 text-sky-600 dark:text-sky-400" />
                <div>
                  <p className="font-medium text-sky-800 dark:text-sky-200">
                    {t('telegram_connected') || 'Telegram connected'}
                  </p>
                  <p className="text-sm text-sky-600 dark:text-sky-400">
                    {t('telegram_connected_notif_desc') || 'Notifications via Telegram are enabled'}
                  </p>
                </div>
              </div>
              <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                <div>
                  <p className="font-medium text-slate-900 dark:text-white">
                    {t('telegram_alerts') || 'Telegram alerts'}
                  </p>
                  <p className="text-sm text-slate-500 dark:text-slate-400">
                    {t('telegram_alerts_desc') || 'Send alert notifications to Telegram'}
                  </p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={tgStatus.alerts}
                    onChange={(e) => handleTelegramToggle('alerts', e.target.checked)}
                  />
                  <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-sky-500 rounded-full peer peer-checked:bg-sky-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                </label>
              </div>
            </div>
          ) : (
            <div className="text-center py-6">
              <div className="p-3 bg-sky-50 dark:bg-sky-900/20 rounded-full w-fit mx-auto mb-4">
                <MessageCircle className="w-8 h-8 text-sky-600 dark:text-sky-400" />
              </div>
              <p className="text-slate-700 dark:text-slate-300 mb-2">
                {t('telegram_not_connected') || 'Telegram not connected'}
              </p>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                {t('telegram_not_connected_notif_desc') || 'Connect your Telegram account to receive real-time notifications'}
              </p>
              <Button
                icon={<MessageCircle className="w-4 h-4" />}
                onClick={async () => {
                  try {
                    const data = await api.generateTelegramLink();
                    window.open(`https://t.me/${data.token}`, '_blank');
                    toast.success(t('telegram_link_generated') || 'Telegram link generated!');
                  } catch (err: any) {
                    toast.error(err.message || t('telegram_link_error') || 'Failed to generate Telegram link');
                  }
                }}
              >
                {t('connect_telegram') || 'Connect Telegram'}
              </Button>
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  );
};
