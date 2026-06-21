import React from 'react';
import { Shield, Lock } from 'lucide-react';
import { Card, CardHeader, CardBody, Select } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import type { AppSettings } from '../../types';

interface Props {
  security: AppSettings['security'];
  onChange: (security: AppSettings['security']) => void;
}

export const SecuritySettings: React.FC<Props> = ({ security, onChange }) => {
  const { t } = useTranslation();

  return (
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
                checked={security.requires2FA}
                onChange={(e) => onChange({ ...security, requires2FA: e.target.checked })}
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
                value={security.passwordPolicy}
                onChange={(e) => onChange({ ...security, passwordPolicy: e.target.value as 'basic' | 'strong' })}
              />
            </div>
          </div>
        </div>
      </CardBody>
    </Card>
  );
};