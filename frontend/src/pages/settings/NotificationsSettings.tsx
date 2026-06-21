import React from 'react';
import { Bell, Shield, Database, Mail, Smartphone } from 'lucide-react';
import { Card, CardHeader, CardBody } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import type { AppSettings } from '../../types';

interface Props {
  notifications: AppSettings['notifications'];
  onNotificationChange: (field: keyof AppSettings['notifications'], value: boolean) => void;
}

export const NotificationsSettings: React.FC<Props> = ({ notifications, onNotificationChange }) => {
  const { t } = useTranslation();

  const items = [
    { id: 'deviceOffline' as const, icon: Bell, title: t('device_offline_alerts'), desc: t('device_offline_desc'), color: 'blue' },
    { id: 'securityAlerts' as const, icon: Shield, title: t('security_alerts'), desc: t('security_alerts_desc'), color: 'red' },
    { id: 'storageWarnings' as const, icon: Database, title: t('storage_warnings'), desc: t('storage_warnings_desc'), color: 'amber' },
    { id: 'dailyReports' as const, icon: Mail, title: t('daily_report_email'), desc: t('daily_report_desc'), color: 'emerald' },
    { id: 'mobilePush' as const, icon: Smartphone, title: t('mobile_push'), desc: t('mobile_push_desc'), color: 'purple' },
  ];

  return (
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
  );
};