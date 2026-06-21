import React from 'react';
import { Save, Settings as SettingsIcon, Zap } from 'lucide-react';
import { Card, CardHeader, CardBody, CardFooter, Button, Input, Select } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import type { AppSettings } from '../../types';

interface Props {
  formData: AppSettings;
  onTopLevelChange: (field: string, value: any) => void;
  onSystemChange: (field: keyof AppSettings['system'], value: number) => void;
  onSave: () => void;
  onReset: () => void;
}

export const GeneralSettings: React.FC<Props> = ({
  formData,
  onTopLevelChange,
  onSystemChange,
  onSave,
  onReset,
}) => {
  const { t } = useTranslation();

  return (
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
                onChange={(e) => onTopLevelChange('organizationName', e.target.value)}
              />
              <Input
                label={t('system_email')}
                type="email"
                value={formData.systemEmail}
                onChange={(e) => onTopLevelChange('systemEmail', e.target.value)}
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
                onChange={(e) => onTopLevelChange('timezone', e.target.value)}
              />
              <Select
                label={t('date_format')}
                options={[
                  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
                  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
                  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
                ]}
                value={formData.dateFormat}
                onChange={(e) => onTopLevelChange('dateFormat', e.target.value)}
              />
            </div>
          </div>
        </CardBody>
        <CardFooter>
          <Button icon={<Save className="w-4 h-4" />} onClick={onSave}>
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
              onChange={(e) => onSystemChange('healthCheckInterval', parseInt(e.target.value) || 5)}
              helperText="Minutes between health checks"
            />
            <Input
              label={t('session_timeout')}
              type="number"
              value={formData.system.sessionTimeout}
              onChange={(e) => onSystemChange('sessionTimeout', parseInt(e.target.value) || 30)}
              helperText="Auto-logout after inactivity (minutes)"
            />
            <Input
              label={t('max_recording_gap')}
              type="number"
              value={formData.system.maxRecordingGap}
              onChange={(e) => onSystemChange('maxRecordingGap', parseInt(e.target.value) || 15)}
              helperText="Alert threshold for recording gaps"
            />
            <Input
              label={t('alert_threshold')}
              type="number"
              value={formData.system.alertThreshold}
              onChange={(e) => onSystemChange('alertThreshold', parseInt(e.target.value) || 85)}
              helperText="Percentage threshold for alerts"
            />
          </div>
        </CardBody>
        <CardFooter>
          <div className="flex gap-3">
            <Button icon={<Save className="w-4 h-4" />} onClick={onSave}>
              {t('save_configuration')}
            </Button>
            <Button variant="outline" onClick={onReset}>
              {t('reset_defaults')}
            </Button>
          </div>
        </CardFooter>
      </Card>
    </div>
  );
};