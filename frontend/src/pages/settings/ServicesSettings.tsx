import React from 'react';
import {
  Save, Radio, Server, Network, Wifi, Shield, Lock, Globe, Database, Monitor,
  Loader2, CheckCircle2, AlertCircle,
} from 'lucide-react';
import { Card, CardHeader, CardBody, Button, Input, Select } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import type { ServicesSettings as ServicesSettingsType, GB28181Settings } from '../../types';

interface Props {
  servicesSettings: ServicesSettingsType | null;
  servicesLoading: boolean;
  servicesSaving: boolean;
  onGB28181Change: (field: string, value: any) => void;
  onServiceChange: (serviceKey: string, field: string, value: any) => void;
  onSave: () => void;
  validateServerID: (id: string) => boolean;
  parseGB28181ID: (id: string) => { type: string; region: string; industry: string; network: string; serial: string } | null;
}

const ServiceToggle: React.FC<{
  serviceKey: string;
  icon: React.FC<{ className?: string }>;
  iconColor: string;
  title: string;
  description: string;
  servicesSettings: ServicesSettingsType;
  onServiceChange: (serviceKey: string, field: string, value: any) => void;
}> = ({ serviceKey, icon: Icon, iconColor, title, description, servicesSettings, onServiceChange }) => {
  const service = servicesSettings[serviceKey as keyof ServicesSettingsType] as any;
  if (!service) return null;

  return (
    <div className="flex items-center gap-3">
      <div className={`p-2 bg-${iconColor}-50 dark:bg-${iconColor}-900/20 rounded-lg`}>
        <Icon className={`w-5 h-5 text-${iconColor}-600 dark:text-${iconColor}-400`} />
      </div>
      <div className="flex-1">
        <h4 className="font-medium text-slate-900 dark:text-white">{title}</h4>
        <p className="text-xs text-slate-500 dark:text-slate-400">{description}</p>
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

export const ServicesSettings: React.FC<Props> = ({
  servicesSettings,
  servicesLoading,
  servicesSaving,
  onGB28181Change,
  onServiceChange,
  onSave,
  validateServerID,
  parseGB28181ID,
}) => {
  const { t } = useTranslation();

  return (
    <div className="space-y-6">
      {/* GB/T 28181 SIP SERVER */}
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
        <CardHeader className="flex items-center gap-2">
          <Network className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <div>
            <span>{t('network_services') || 'Network Services & Protocols'}</span>
            <p className="text-xs font-normal text-slate-500 dark:text-slate-400 mt-0.5">Configure protocol receivers and external service connections</p>
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
                { key: 'services_snmp', icon: Shield, color: 'amber', title: 'SNMP Trap Receiver', desc: 'Receive SNMP traps from network devices', fields: [{ key: 'port', label: 'SNMP Port', type: 'number' }, { key: 'version', label: 'SNMP Version', type: 'select', options: [{ value: 'v1', label: 'SNMP v1' }, { value: 'v2c', label: 'SNMP v2c' }, { value: 'v3', label: 'SNMP v3' }] }, { key: 'community', label: 'Community String', type: 'text' }] },
                { key: 'services_http', icon: Globe, color: 'purple', title: 'HTTP Log Receiver', desc: 'Receive logs via HTTP POST', fields: [{ key: 'port', label: 'HTTP Port', type: 'number' }] },
                { key: 'services_dahua', icon: Monitor, color: 'cyan', title: 'Dahua Private Protocol', desc: 'Proprietary Dahua protocol for events', fields: [{ key: 'ports', label: 'Ports (comma-separated)', type: 'text' }] },
                { key: 'services_hisilicon', icon: Monitor, color: 'indigo', title: 'Hisilicon Protocol', desc: 'Hisilicon-based devices events', fields: [{ key: 'port', label: 'Port', type: 'number' }] },
                { key: 'services_tvt', icon: Monitor, color: 'pink', title: 'TVT Protocol', desc: 'TVT-based devices events', fields: [{ key: 'port', label: 'Port', type: 'number' }] },
                { key: 'services_sip', icon: Globe, color: 'teal', title: 'SIP / GB28181 (Legacy)', desc: 'SIP signaling for GB28181 devices', fields: [{ key: 'port', label: 'SIP Port', type: 'number' }, { key: 'host', label: 'Host', type: 'text', placeholder: '0.0.0.0' }] },
              ].map((svc) => (
                <div key={svc.key} className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                  <ServiceToggle
                    serviceKey={svc.key}
                    icon={svc.icon}
                    iconColor={svc.color}
                    title={svc.title}
                    description={svc.desc}
                    servicesSettings={servicesSettings}
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
                  <Input label="Gateway URL" value={servicesSettings.services_p2p_gateway.url} onChange={(e) => onServiceChange('services_p2p_gateway', 'url', e.target.value)} placeholder="http://localhost:8082" />
                  <Input label="API Key" type="password" value={servicesSettings.services_p2p_gateway.api_key} onChange={(e) => onServiceChange('services_p2p_gateway', 'api_key', e.target.value)} />
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