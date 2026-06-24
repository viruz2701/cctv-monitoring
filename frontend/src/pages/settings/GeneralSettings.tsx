import React, { useState, useEffect } from 'react';
import { Save, Settings as SettingsIcon, Zap, Key, Plus, Trash2, Copy, Shield, Calendar } from 'lucide-react';
import { Card, CardHeader, CardBody, CardFooter, Button, Input, Select, Modal, Badge, useToast } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import { api } from '../../services/api';
import type { AppSettings } from '../../types';

interface Props {
  formData: AppSettings;
  onTopLevelChange: (field: string, value: any) => void;
  onSystemChange: (field: keyof AppSettings['system'], value: number) => void;
  onSave: () => void;
  onReset: () => void;
}

interface APIKey {
    id: string;
    name: string;
    permissions: string[];
    expires_at: string | null;
    last_used_at: string | null;
    created_at: string;
}

const TIMEZONES = Intl.supportedValuesOf('timeZone');

export const GeneralSettings: React.FC<Props> = ({
  formData,
  onTopLevelChange,
  onSystemChange,
  onSave,
  onReset,
}) => {
  const { t } = useTranslation();
  const toast = useToast();
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showKeyModal, setShowKeyModal] = useState(false);
  const [newKey, setNewKey] = useState<string>('');
  const [formKeyData, setFormKeyData] = useState({
      name: '',
      permissions: ['read'],
      expires_at: '',
  });

  useEffect(() => {
    loadKeys();
  }, []);

  const loadKeys = async () => {
    try {
      setLoading(true);
      const data = await api.getAPIKeys();
      setKeys(Array.isArray(data) ? data : []);
    } catch (err: any) {
      // silently fail - API keys may not be available
      setKeys([]);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateKey = async () => {
    try {
      const data = await api.createAPIKey({
        name: formKeyData.name,
        permissions: formKeyData.permissions,
        expires_at: formKeyData.expires_at || undefined,
      });
      setNewKey(data.api_key);
      setShowCreateModal(false);
      setShowKeyModal(true);
      setFormKeyData({ name: '', permissions: ['read'], expires_at: '' });
      loadKeys();
    } catch (err: any) {
      toast.error(err.message || t('api_key_create_error'));
    }
  };

  const handleRevokeKey = async (id: string) => {
    if (!confirm(t('api_key_revoke_confirm'))) return;
    try {
      await api.revokeAPIKey(id);
      toast.success(t('api_key_revoked'));
      loadKeys();
    } catch (err: any) {
      toast.error(err.message || t('api_key_revoke_error'));
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success(t('copied_to_clipboard'));
  };

  const permissionOptions = [
    { value: 'read', label: t('permission_read') },
    { value: 'write', label: t('permission_write') },
    { value: 'admin', label: t('permission_admin') },
  ];

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
                options={TIMEZONES.map(tz => ({ value: tz, label: tz }))}
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

      {/* API KEYS MANAGEMENT */}
      <Card>
        <CardHeader className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Key className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
            <span>{t('api_keys')}</span>
          </div>
          <Button size="sm" onClick={() => setShowCreateModal(true)} icon={<Plus className="w-4 h-4" />}>
            {t('create_api_key')}
          </Button>
        </CardHeader>
        <CardBody>
          {loading ? (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600"></div>
              <span className="ml-3 text-sm text-slate-500 dark:text-slate-400">{t('loading')}</span>
            </div>
          ) : keys.length === 0 ? (
            <div className="text-center py-8">
              <Key className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
              <p className="text-sm text-slate-500 dark:text-slate-400">{t('no_api_keys')}</p>
              <Button size="sm" className="mt-3" onClick={() => setShowCreateModal(true)} icon={<Plus className="w-4 h-4" />}>
                {t('create_first_key')}
              </Button>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-200 dark:border-slate-700">
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('name')}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('permissions')}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('created')}</th>
                    <th className="px-4 py-2 text-left text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('expires')}</th>
                    <th className="px-4 py-2 text-right text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">{t('actions')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                  {keys.map((key) => (
                    <tr key={key.id} className="hover:bg-slate-50 dark:hover:bg-slate-700/50">
                      <td className="px-4 py-3 whitespace-nowrap">
                        <div className="flex items-center gap-2">
                          <Key className="w-3.5 h-3.5 text-slate-400" />
                          <span className="font-medium text-slate-900 dark:text-white">{key.name}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap">
                        <div className="flex gap-1">
                          {Array.isArray(key.permissions) && key.permissions.map((perm) => (
                            <Badge key={perm} variant={perm === 'admin' ? 'danger' : 'info'}>{perm}</Badge>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-slate-500 dark:text-slate-400">
                        <div className="flex items-center gap-1">
                          <Calendar className="w-3 h-3" />
                          {new Date(key.created_at).toLocaleDateString()}
                        </div>
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-slate-500 dark:text-slate-400">
                        {key.expires_at ? (
                          <div className="flex items-center gap-1">
                            <Calendar className="w-3 h-3" />
                            {new Date(key.expires_at).toLocaleDateString()}
                          </div>
                        ) : (
                          <span className="text-slate-400">{t('never')}</span>
                        )}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-right">
                        <Button variant="ghost" size="sm" onClick={() => handleRevokeKey(key.id)} className="text-red-600 hover:text-red-700 dark:text-red-400" icon={<Trash2 className="w-4 h-4" />}>
                          {t('revoke')}
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Create API Key Modal */}
      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title={t('create_api_key')} size="md">
        <div className="space-y-4">
          <Input label={t('name')} value={formKeyData.name} onChange={(e) => setFormKeyData({ ...formKeyData, name: e.target.value })} placeholder={t('api_key_name_placeholder')} />
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">{t('permissions')}</label>
            <div className="space-y-2">
              {permissionOptions.map((option) => (
                <label key={option.value} className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formKeyData.permissions.includes(option.value)}
                    onChange={(e) => {
                      if (e.target.checked) {
                        setFormKeyData({ ...formKeyData, permissions: [...formKeyData.permissions, option.value] });
                      } else {
                        setFormKeyData({ ...formKeyData, permissions: formKeyData.permissions.filter(p => p !== option.value) });
                      }
                    }}
                    className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  <span className="text-sm text-slate-700 dark:text-slate-300">{option.label}</span>
                </label>
              ))}
            </div>
          </div>
          <Input label={t('expires_at')} type="date" value={formKeyData.expires_at} onChange={(e) => setFormKeyData({ ...formKeyData, expires_at: e.target.value })} />
          <div className="flex justify-end gap-3 pt-4">
            <Button variant="ghost" onClick={() => setShowCreateModal(false)}>{t('cancel')}</Button>
            <Button onClick={handleCreateKey} disabled={!formKeyData.name}>{t('create')}</Button>
          </div>
        </div>
      </Modal>

      {/* Show New Key Modal */}
      <Modal isOpen={showKeyModal} onClose={() => setShowKeyModal(false)} title={t('api_key_created')} size="md">
        <div className="space-y-4">
          <div className="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
            <div className="flex items-start gap-3">
              <Shield className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-amber-800 dark:text-amber-200">{t('save_key_warning')}</p>
                <p className="text-xs text-amber-600 dark:text-amber-400 mt-1">{t('key_wont_be_shown_again')}</p>
              </div>
            </div>
          </div>
          <div className="p-4 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
            <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">{t('your_api_key')}</p>
            <div className="flex items-center gap-2">
              <code className="flex-1 p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-600 rounded text-sm font-mono break-all">{newKey}</code>
              <Button variant="ghost" size="sm" onClick={() => copyToClipboard(newKey)} icon={<Copy className="w-4 h-4" />} />
            </div>
          </div>
          <div className="flex justify-end pt-4">
            <Button onClick={() => setShowKeyModal(false)}>{t('done')}</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};
