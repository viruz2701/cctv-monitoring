import React, { useState } from 'react';
import {
  Save, Radio, Server, Network, Wifi, Shield, Lock, Globe, Database, Monitor,
  Loader2, CheckCircle2, AlertCircle, AudioWaveform, Bug, Activity, RefreshCw,
} from 'lucide-react';
import { Card, CardHeader, CardBody, Button, Input, Select, Tabs } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import type { ServicesSettings as ServicesSettingsType, GB28181Settings } from '../../types';

interface Props {
  servicesSettings: ServicesSettingsType | null;
  servicesLoading: boolean;
  servicesSaving: boolean;
  servicesStatus: Record<string, { status: string; port: number; message?: string }>;
  servicesStatusLoading: boolean;
  onGB28181Change: (field: string, value: any) => void;
  onServiceChange: (serviceKey: string, field: string, value: any) => void;
  onSave: () => void;
  onRefreshStatus: () => void;
  validateServerID: (id: string) => boolean;
  parseGB28181ID: (id: string) => { type: string; region: string; industry: string; network: string; serial: string } | null;
}

// ── StatusDot — цветной индикатор состояния сервиса ──────────────
const StatusDot: React.FC<{ status?: string; loading?: boolean }> = ({ status, loading }) => {
  if (loading) {
    return <Loader2 className="w-3 h-3 animate-spin text-slate-400" />;
  }
  switch (status) {
    case 'running':
      return <span className="w-3 h-3 rounded-full bg-emerald-500 inline-block shadow-sm shadow-emerald-400/50" title="Running" />;
    case 'stopped':
      return <span className="w-3 h-3 rounded-full bg-red-500 inline-block shadow-sm shadow-red-400/50" title="Stopped" />;
    case 'error':
      return <span className="w-3 h-3 rounded-full bg-amber-500 inline-block shadow-sm shadow-amber-400/50" title="Error" />;
    default:
      return <span className="w-3 h-3 rounded-full bg-slate-300 dark:bg-slate-600 inline-block" title="Disabled" />;
  }
};

const ServiceToggle: React.FC<{
  serviceKey: string;
  icon: React.FC<{ className?: string }>;
  iconColor: string;
  title: string;
  description: string;
  servicesSettings: ServicesSettingsType;
  servicesStatus: Record<string, { status: string; port: number; message?: string }>;
  servicesStatusLoading: boolean;
  onServiceChange: (serviceKey: string, field: string, value: any) => void;
}> = ({ serviceKey, icon: Icon, iconColor, title, description, servicesSettings, servicesStatus, servicesStatusLoading, onServiceChange }) => {
  const service = servicesSettings[serviceKey as keyof ServicesSettingsType] as any;
  if (!service) return null;

  const svcName = serviceKey.replace('services_', '');
  const st = servicesStatus[svcName];

  return (
    <div className="flex items-center gap-3">
      <div className={`p-2 bg-${iconColor}-50 dark:bg-${iconColor}-900/20 rounded-lg`}>
        <Icon className={`w-5 h-5 text-${iconColor}-600 dark:text-${iconColor}-400`} />
      </div>
      <div className="flex-1">
        <div className="flex items-center gap-2">
          <h4 className="font-medium text-slate-900 dark:text-white">{title}</h4>
          <StatusDot status={st?.status} loading={servicesStatusLoading} />
        </div>
        <p className="text-xs text-slate-500 dark:text-slate-400">
          {description}
          {st?.status === 'stopped' && st?.message && (
            <span className="text-red-500 ml-1">({st.message})</span>
          )}
          {st?.status === 'running' && (
            <span className="text-emerald-500 ml-1">:{st.port}</span>
          )}
        </p>
      </div>
      <label className="relative inline-flex items-center cursor-pointer">
        <input
          type="checkbox"
          className="sr-only peer"
          checked={service.enabled}
          onChange={(e) => onServiceChange(serviceKey, 'enabled', e.target.checked)}
        />
        <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
      </label>
    </div>
  );
};

// ── SNMP Multi-Version Panel ──────────────────────────────────────
const SNMPVersionPanel: React.FC<{
  snmp: any;
  onSNMPChange: (field: string, value: any) => void;
  version: 'v1_config' | 'v2c_config' | 'v3_config';
  label: string;
}> = ({ snmp, onSNMPChange, version, label }) => {
  const { t } = useTranslation();
  const cfg = snmp?.[version] || {};
  const isV3 = version === 'v3_config';

  return (
    <div className="space-y-4 p-4 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-amber-600 dark:text-amber-400" />
          <h5 className="text-sm font-semibold text-slate-700 dark:text-slate-300">{label}</h5>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={cfg.enabled}
            onChange={(e) => onSNMPChange(`${version}.enabled`, e.target.checked)}
          />
          <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-amber-500 rounded-full peer peer-checked:bg-amber-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
        </label>
      </div>

      {cfg.enabled && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Input
            label="Port"
            type="number"
            value={cfg.port}
            onChange={(e) => onSNMPChange(`${version}.port`, parseInt(e.target.value) || 162)}
          />
          {!isV3 ? (
            <Input
              label="Community String"
              type="password"
              value={cfg.community}
              onChange={(e) => onSNMPChange(`${version}.community`, e.target.value)}
            />
          ) : (
            <>
              <Input
                label="User (Security Name)"
                value={cfg.user}
                onChange={(e) => onSNMPChange(`${version}.user`, e.target.value)}
              />
              <Select
                label="Auth Protocol"
                options={[
                  { value: 'MD5', label: 'MD5' },
                  { value: 'SHA', label: 'SHA' },
                  { value: 'SHA256', label: 'SHA-256' },
                ]}
                value={cfg.auth_protocol || 'SHA'}
                onChange={(e) => onSNMPChange(`${version}.auth_protocol`, e.target.value)}
              />
              <Input
                label="Auth Password"
                type="password"
                value={cfg.auth_password}
                onChange={(e) => onSNMPChange(`${version}.auth_password`, e.target.value)}
              />
              <Select
                label="Privacy Protocol"
                options={[
                  { value: 'DES', label: 'DES' },
                  { value: 'AES', label: 'AES' },
                  { value: 'AES192', label: 'AES-192' },
                  { value: 'AES256', label: 'AES-256' },
                ]}
                value={cfg.priv_protocol || 'AES'}
                onChange={(e) => onSNMPChange(`${version}.priv_protocol`, e.target.value)}
              />
              <Input
                label="Privacy Password"
                type="password"
                value={cfg.priv_password}
                onChange={(e) => onSNMPChange(`${version}.priv_password`, e.target.value)}
              />
            </>
          )}
        </div>
      )}
    </div>
  );
};

// ── P2P Gateway per-vendor panels ─────────────────────────────────
const P2PVendorPanel: React.FC<{
  vendor: string;
  title: string;
  icon: React.FC<{ className?: string }>;
  iconColor: string;
  fields: { key: string; label: string; type: string; options?: { value: string; label: string }[] }[];
  values: Record<string, any>;
  onChange: (field: string, value: any) => void;
}> = ({ vendor, title, icon: Icon, iconColor, fields, values, onChange }) => {
  return (
    <div className="p-4 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
      <div className="flex items-center gap-2 mb-3">
        <div className={`p-1.5 bg-${iconColor}-50 dark:bg-${iconColor}-900/20 rounded`}>
          <Icon className={`w-4 h-4 text-${iconColor}-600 dark:text-${iconColor}-400`} />
        </div>
        <h5 className="text-sm font-semibold text-slate-700 dark:text-slate-300">{title}</h5>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {fields.map((f) =>
          f.type === 'select' ? (
            <Select
              key={f.key}
              label={f.label}
              options={f.options || []}
              value={values?.[f.key] || ''}
              onChange={(e) => onChange(`${vendor}.${f.key}`, e.target.value)}
            />
          ) : (
            <Input
              key={f.key}
              label={f.label}
              type={f.type === 'password' ? 'password' : 'text'}
              value={values?.[f.key] || ''}
              onChange={(e) =>
                onChange(`${vendor}.${f.key}`, f.type === 'number' ? parseInt(e.target.value) || 0 : e.target.value)
              }
            />
          )
        )}
      </div>
    </div>
  );
};

export const ServicesSettings: React.FC<Props> = ({
  servicesSettings,
  servicesLoading,
  servicesSaving,
  servicesStatus,
  servicesStatusLoading,
  onGB28181Change,
  onServiceChange,
  onSave,
  onRefreshStatus,
  validateServerID,
  parseGB28181ID,
}) => {
  const { t } = useTranslation();
  const [snmpTab, setSnmpTab] = useState('v1');

  const snmpTabs = [
    { id: 'v1', label: 'SNMP v1', icon: <Bug className="w-4 h-4" /> },
    { id: 'v2c', label: 'SNMP v2c', icon: <Bug className="w-4 h-4" /> },
    { id: 'v3', label: 'SNMP v3', icon: <Shield className="w-4 h-4" /> },
  ];

  // Обработчик для вложенных SNMP полей
  const handleSNMPChange = (field: string, value: any) => {
    // Парсим version.enabled, version.port, etc.
    const parts = field.split('.');
    if (parts.length === 2) {
      const [ver, f] = parts;
      const snmp = (servicesSettings?.services_snmp as any) || {};
      onServiceChange('services_snmp', ver, { ...snmp[ver], [f]: value });
    }
  };

  // Обработчик для вложенных P2P полей
  const handleP2PChange = (field: string, value: any) => {
    const parts = field.split('.');
    if (parts.length === 2) {
      const [vendor, f] = parts;
      const p2p = servicesSettings?.services_p2p_gateway as any;
      onServiceChange('services_p2p_gateway', vendor, { ...p2p[vendor], [f]: value });
    }
  };

  return (
    <div className="space-y-6">
      {/* GB/T 28181 SIP SERVER */}
      <Card>
        <CardHeader className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Radio className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
            <div>
              <span>{t('gb28181_settings') || 'GB/T 28181 (China National Standard)'}</span>
              <p className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">
                {t('gb28181_desc') || 'SIP-based protocol for CCTV interoperability. Used by Hikvision, Dahua, Uniview NVRs.'}
              </p>
            </div>
          </div>
          <StatusDot status={servicesStatus['gb28181']?.status} loading={servicesStatusLoading} />
        </CardHeader>
        <CardBody>
          {servicesLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-indigo-600" />
              <span className="ml-3 text-slate-600 dark:text-slate-400">Loading GB28181 settings...</span>
            </div>
          ) : servicesSettings?.services_gb28181 ? (
            <div className="space-y-6">
              <div className="flex items-center justify-between p-4 bg-indigo-50/50 dark:bg-indigo-900/10 rounded-lg border border-indigo-100 dark:border-indigo-800/30">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-white dark:bg-slate-800 rounded-lg shadow-sm">
                    <Server className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
                  </div>
                  <div className="flex items-center gap-2">
                    <div>
                      <p className="font-medium text-slate-900 dark:text-white">
                        {t('enable') || 'Enable'} GB28181 SIP Server
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        UDP/TCP Port {servicesSettings.services_gb28181.port}
                        {servicesStatus['gb28181']?.status === 'running' && (
                          <span className="text-emerald-500 ml-1">· Online</span>
                        )}
                        {servicesStatus['gb28181']?.status === 'stopped' && (
                          <span className="text-red-500 ml-1">· Offline</span>
                        )}
                      </p>
                    </div>
                  </div>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={servicesSettings.services_gb28181.enabled}
                    onChange={(e) => onGB28181Change('enabled', e.target.checked)}
                  />
                  <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-indigo-500 rounded-full peer peer-checked:bg-indigo-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                </label>
              </div>

              {servicesSettings.services_gb28181.enabled && (
                <>
                  <div>
                    <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-3 flex items-center gap-2">
                      <Wifi className="w-4 h-4" />
                      {t('gb28181_network') || 'Network & Transport'}
                    </h4>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Input label="Bind Host" value={servicesSettings.services_gb28181.host} onChange={(e) => onGB28181Change('host', e.target.value)} placeholder="0.0.0.0" />
                      <Input label="SIP Port (UDP/TCP)" type="number" value={servicesSettings.services_gb28181.port} onChange={(e) => onGB28181Change('port', parseInt(e.target.value) || 5060)} />
                    </div>
                  </div>

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
                          type="text" maxLength={20}
                          value={servicesSettings.services_gb28181.server_id}
                          onChange={(e) => { const val = e.target.value.replace(/\D/g, '').slice(0, 20); onGB28181Change('server_id', val); }}
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
                        <Input label={t('gb28181_server_ip') || 'Public IP / Contact Address'} value={servicesSettings.services_gb28181.server_ip} onChange={(e) => onGB28181Change('server_ip', e.target.value)} placeholder="auto (from incoming packets)" helperText={t('gb28181_server_ip_help') || 'IP that devices behind NAT will use to reach this server'} />
                        <Input label={t('gb28181_realm') || 'SIP Realm / Domain'} value={servicesSettings.services_gb28181.realm} onChange={(e) => onGB28181Change('realm', e.target.value)} placeholder="3402000000" />
                      </div>
                    </div>
                  </div>

                  <div>
                    <div className="flex items-center justify-between mb-3">
                      <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider flex items-center gap-2">
                        <Lock className="w-4 h-4" />
                        {t('gb28181_auth') || 'Authentication'}
                      </h4>
                      <label className="relative inline-flex items-center cursor-pointer">
                        <input type="checkbox" className="sr-only peer" checked={servicesSettings.services_gb28181.auth_enabled} onChange={(e) => onGB28181Change('auth_enabled', e.target.checked)} />
                        <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-indigo-500 rounded-full peer peer-checked:bg-indigo-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                      </label>
                    </div>
                    {servicesSettings.services_gb28181.auth_enabled && (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                        <Input label={t('gb28181_auth_user') || 'Authentication User'} value={servicesSettings.services_gb28181.auth_user} onChange={(e) => onGB28181Change('auth_user', e.target.value)} />
                        <Input label={t('gb28181_auth_password') || 'Authentication Password'} type="password" value={servicesSettings.services_gb28181.auth_password} onChange={(e) => onGB28181Change('auth_password', e.target.value)} />
                      </div>
                    )}
                  </div>

                  <div>
                    <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-3">
                      {t('gb28181_behavior') || 'Behavior'}
                    </h4>
                    <div className="space-y-2">
                      {[
                        { key: 'auto_catalog', title: t('gb28181_auto_catalog') || 'Auto-request device catalog on register', desc: t('gb28181_auto_catalog_desc') || 'Automatically discover cameras connected to NVRs' },
                        { key: 'auto_device_info', title: t('gb28181_auto_device_info') || 'Auto-request device info', desc: t('gb28181_auto_device_info_desc') || 'Query manufacturer, model, firmware on register' },
                        { key: 'log_sip_messages', title: t('gb28181_log_sip') || 'Log raw SIP messages (debug)', desc: 'Log raw SIP packets to parsed_logs table' },
                      ].map((item) => (
                        <div key={item.key} className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
                          <div>
                            <p className="text-sm font-medium text-slate-900 dark:text-white">{item.title}</p>
                            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">{item.desc}</p>
                          </div>
                          <label className="relative inline-flex items-center cursor-pointer">
                            <input type="checkbox" className="sr-only peer" checked={(servicesSettings.services_gb28181?.[item.key as keyof GB28181Settings] ?? false) as boolean} onChange={(e) => onGB28181Change(item.key, e.target.checked)} />
                            <div className="w-11 h-6 bg-slate-300 dark:bg-slate-700 peer-focus:ring-2 peer-focus:ring-indigo-500 rounded-full peer peer-checked:bg-indigo-600 after:content-[''] after:absolute after:top-0.5 after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:after:translate-x-full" />
                          </label>
                        </div>
                      ))}
                    </div>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
                      <Input label={t('gb28181_keepalive_interval') || 'Expected Keepalive Interval (sec)'} type="number" min={10} max={3600} value={servicesSettings.services_gb28181.keepalive_interval} onChange={(e) => onGB28181Change('keepalive_interval', parseInt(e.target.value) || 60)} />
                      <Input label={t('gb28181_keepalive_timeout') || 'Offline Timeout (sec)'} type="number" min={30} max={7200} value={servicesSettings.services_gb28181.keepalive_timeout} onChange={(e) => onGB28181Change('keepalive_timeout', parseInt(e.target.value) || 180)} />
                      <Input label={t('gb28181_max_sub_channels') || 'Max child devices per NVR'} type="number" min={1} max={1024} value={servicesSettings.services_gb28181.max_sub_channels} onChange={(e) => onGB28181Change('max_sub_channels', parseInt(e.target.value) || 64)} />
                    </div>
                  </div>
                </>
              )}
            </div>
          ) : (
            <div className="text-center py-12 text-slate-500 dark:text-slate-400">Failed to load GB28181 settings</div>
          )}
        </CardBody>
      </Card>

      {/* NETWORK SERVICES & PROTOCOLS */}
      <Card>
        <CardHeader className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Network className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <div>
              <span>{t('network_services') || 'Network Services & Protocols'}</span>
              <p className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">Configure protocol receivers and external service connections</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            {/* Status legend */}
            <div className="hidden sm:flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
              <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-emerald-500" /> Running</span>
              <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-red-500" /> Stopped</span>
              <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-slate-300 dark:bg-slate-600" /> Disabled</span>
            </div>
            <Button
              variant="ghost"
              size="sm"
              onClick={onRefreshStatus}
              disabled={servicesStatusLoading}
              icon={servicesStatusLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
              title="Refresh service status"
            />
          </div>
        </CardHeader>
        <CardBody>
          {servicesLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
              <span className="ml-3 text-slate-600 dark:text-slate-400">{t('loading_services') || 'Loading services configuration...'}</span>
            </div>
          ) : servicesSettings ? (
            <div className="space-y-6">
              {[
                { key: 'services_syslog', icon: Server, color: 'blue', title: 'Syslog Receiver', desc: 'UDP/TCP syslog messages from devices', fields: [{ key: 'udp_port', label: 'Syslog UDP Port', type: 'number' }, { key: 'tcp_port', label: 'Syslog TCP Port', type: 'number' }] },
                { key: 'services_ftp', icon: Database, color: 'emerald', title: 'FTP Server', desc: 'Receive snapshots and logs via FTP', fields: [{ key: 'port', label: 'FTP Port', type: 'number' }, { key: 'root_path', label: 'Root Path', type: 'text' }, { key: 'user', label: 'Username', type: 'text' }, { key: 'password', label: 'Password', type: 'password' }] },
                { key: 'services_http', icon: Globe, color: 'purple', title: 'HTTP Log Receiver', desc: 'Receive logs via HTTP POST', fields: [{ key: 'port', label: 'HTTP Port', type: 'number' }] },
                { key: 'services_dahua', icon: Monitor, color: 'cyan', title: 'Dahua Private Protocol', desc: 'Proprietary Dahua protocol for events', fields: [{ key: 'ports', label: 'Ports (comma-separated)', type: 'text' }] },
                { key: 'services_hisilicon', icon: Monitor, color: 'indigo', title: 'Hisilicon Protocol', desc: 'Hisilicon-based devices events', fields: [{ key: 'port', label: 'Port', type: 'number' }] },
                { key: 'services_tvt', icon: Monitor, color: 'pink', title: 'TVT Protocol', desc: 'TVT-based devices events', fields: [{ key: 'port', label: 'Port', type: 'number' }] },
              ].map((svc) => (
                <div key={svc.key} className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey={svc.key}
                    icon={svc.icon}
                    iconColor={svc.color}
                    title={svc.title}
                    description={svc.desc}
                    servicesSettings={servicesSettings}
                    servicesStatus={servicesStatus}
                    servicesStatusLoading={servicesStatusLoading}
                    onServiceChange={onServiceChange}
                  />
                  {(servicesSettings[svc.key as keyof ServicesSettingsType] as any)?.enabled && (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-4">
                      {svc.fields.map((f: any) => (
                        f.type === 'select' ? (
                          <Select key={f.key} label={f.label} options={f.options} value={(servicesSettings[svc.key as keyof ServicesSettingsType] as any)?.[f.key] ?? ''} onChange={(e) => onServiceChange(svc.key, f.key, e.target.value)} />
                        ) : (
                          <Input
                            key={f.key}
                            label={f.label}
                            type={f.type === 'password' ? 'password' : 'text'}
                            value={f.key === 'ports' ? ((servicesSettings[svc.key as keyof ServicesSettingsType] as any)?.ports || []).join(', ') : (servicesSettings[svc.key as keyof ServicesSettingsType] as any)?.[f.key] ?? ''}
                            onChange={(e) => {
                              if (f.key === 'ports') {
                                const ports = e.target.value.split(',').map((p: string) => parseInt(p.trim())).filter((p: number) => !isNaN(p));
                                onServiceChange(svc.key, 'ports', ports);
                              } else {
                                onServiceChange(svc.key, f.key, f.type === 'number' ? parseInt(e.target.value) : e.target.value);
                              }
                            }}
                            placeholder={f.placeholder}
                          />
                        )
                      ))}
                    </div>
                  )}
                </div>
              ))}

              {/* ── SNMP Trap Receiver — Multi-Version ─────────────── */}
              <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 bg-amber-50 dark:bg-amber-900/20 rounded-lg">
                    <Shield className="w-5 h-5 text-amber-600 dark:text-amber-400" />
                  </div>
                  <div className="flex items-center gap-2">
                    <div>
                      <h4 className="font-medium text-slate-900 dark:text-white">SNMP Trap Receiver</h4>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        Receive SNMP traps from network devices. Each SNMP version (v1/v2c/v3) can run concurrently with its own port and credentials.
                      </p>
                    </div>
                    <StatusDot status={servicesStatus['snmp']?.status} loading={servicesStatusLoading} />
                  </div>
                </div>

                {servicesSettings.services_snmp && (
                  <div className="space-y-4">
                    <Tabs
                      tabs={snmpTabs}
                      activeTab={snmpTab}
                      onChange={setSnmpTab}
                      variant="pills"
                      className="mb-3"
                    >
                      <div />
                    </Tabs>

                    {snmpTab === 'v1' && (
                      <SNMPVersionPanel
                        snmp={servicesSettings.services_snmp}
                        onSNMPChange={handleSNMPChange}
                        version="v1_config"
                        label="SNMP v1 — для старых устройств (community string)"
                      />
                    )}
                    {snmpTab === 'v2c' && (
                      <SNMPVersionPanel
                        snmp={servicesSettings.services_snmp}
                        onSNMPChange={handleSNMPChange}
                        version="v2c_config"
                        label="SNMP v2c — для большинства современных устройств"
                      />
                    )}
                    {snmpTab === 'v3' && (
                      <SNMPVersionPanel
                        snmp={servicesSettings.services_snmp}
                        onSNMPChange={handleSNMPChange}
                        version="v3_config"
                        label="SNMP v3 — для устройств с повышенными требованиями безопасности"
                      />
                    )}
                  </div>
                )}
              </div>

              {/* ── P2P Gateway — Per-Vendor ──────────────────────── */}
              <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 bg-orange-50 dark:bg-orange-900/20 rounded-lg">
                    <Network className="w-5 h-5 text-orange-600 dark:text-orange-400" />
                  </div>
                  <div className="flex items-center gap-2">
                    <div>
                      <h4 className="font-medium text-slate-900 dark:text-white">P2P Gateway</h4>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        Connection to P2P gateway service. Each vendor has its own P2P cloud API settings.
                      </p>
                    </div>
                    <StatusDot status={servicesStatus['p2p_gateway']?.status} loading={servicesStatusLoading} />
                  </div>
                </div>

                {/* General P2P settings */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4 p-4 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
                  <Input
                    label="Gateway URL"
                    value={servicesSettings.services_p2p_gateway.url}
                    onChange={(e) => onServiceChange('services_p2p_gateway', 'url', e.target.value)}
                    placeholder="http://localhost:8082"
                  />
                  <Input
                    label="Gateway API Key"
                    type="password"
                    value={servicesSettings.services_p2p_gateway.api_key}
                    onChange={(e) => onServiceChange('services_p2p_gateway', 'api_key', e.target.value)}
                  />
                </div>

                {/* Per-vendor P2P cloud API settings */}
                <div className="space-y-3">
                  <h5 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider flex items-center gap-2">
                    <AudioWaveform className="w-4 h-4" />
                    Vendor P2P Cloud APIs
                  </h5>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    <P2PVendorPanel
                      vendor="hikvision"
                      title="Hikvision P2P Cloud"
                      icon={Monitor}
                      iconColor="blue"
                      fields={[
                        { key: 'username', label: 'Hik-Connect Username', type: 'text' },
                        { key: 'password', label: 'Hik-Connect Password', type: 'password' },
                      ]}
                      values={servicesSettings.services_p2p_gateway.hikvision}
                      onChange={handleP2PChange}
                    />
                    <P2PVendorPanel
                      vendor="dahua"
                      title="Dahua P2P Cloud"
                      icon={Monitor}
                      iconColor="cyan"
                      fields={[
                        { key: 'python_path', label: 'Python Path', type: 'text' },
                        { key: 'script_path', label: 'DH-P2P Script Path', type: 'text' },
                      ]}
                      values={servicesSettings.services_p2p_gateway.dahua}
                      onChange={handleP2PChange}
                    />
                    <P2PVendorPanel
                      vendor="reolink"
                      title="Reolink P2P Cloud"
                      icon={Monitor}
                      iconColor="green"
                      fields={[
                        { key: 'proxy_bin_path', label: 'Neolink Proxy Binary', type: 'text' },
                      ]}
                      values={servicesSettings.services_p2p_gateway.reolink}
                      onChange={handleP2PChange}
                    />
                    <P2PVendorPanel
                      vendor="xiongmai"
                      title="Xiongmai (Jftech) P2P Cloud"
                      icon={Monitor}
                      iconColor="purple"
                      fields={[
                        { key: 'uuid', label: 'UUID', type: 'text' },
                        { key: 'app_key', label: 'App Key', type: 'password' },
                        { key: 'app_secret', label: 'App Secret', type: 'password' },
                        { key: 'endpoint', label: 'API Endpoint', type: 'text' },
                        { key: 'region', label: 'Region', type: 'text' },
                        { key: 'move_card', label: 'Move Card', type: 'number' },
                      ]}
                      values={servicesSettings.services_p2p_gateway.xiongmai}
                      onChange={handleP2PChange}
                    />
                    <P2PVendorPanel
                      vendor="ezviz"
                      title="EZVIZ P2P Cloud"
                      icon={Monitor}
                      iconColor="teal"
                      fields={[
                        { key: 'app_key', label: 'App Key', type: 'password' },
                        { key: 'app_secret', label: 'App Secret', type: 'password' },
                      ]}
                      values={servicesSettings.services_p2p_gateway.ezviz}
                      onChange={handleP2PChange}
                    />
                  </div>
                </div>
              </div>

              <div className="flex justify-end pt-4 border-t border-slate-200 dark:border-slate-700">
                <Button
                  icon={servicesSaving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                  onClick={onSave}
                  disabled={servicesSaving}
                >
                  {servicesSaving ? (t('saving') || 'Saving...') : (t('save_and_restart_services') || 'Save & Restart Services')}
                </Button>
              </div>
            </div>
          ) : (
            <div className="text-center py-12 text-slate-500 dark:text-slate-400">{t('failed_to_load_services') || 'Failed to load services configuration'}</div>
          )}
        </CardBody>
      </Card>
    </div>
  );
};
