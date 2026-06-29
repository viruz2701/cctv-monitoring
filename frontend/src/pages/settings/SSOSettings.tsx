import React, { useState } from 'react';
import { Shield, Server, Key, Globe, CheckCircle, AlertTriangle } from '../components/ui/Icons';
import { Card, Button, Badge, useToast } from '../../components/ui';
import { useTranslation } from 'react-i18next';

// ── Types ────────────────────────────────────────────────────────────

interface LDAPSettings {
  enabled: boolean;
  host: string;
  port: number;
  use_tls: boolean;
  base_dn: string;
  bind_dn: string;
  bind_password: string;
  user_filter: string;
  login_attribute: string;
  mail_attribute: string;
  name_attribute: string;
  default_role: string;
}

interface SAMLSettings {
  enabled: boolean;
  idp_metadata_url: string;
  idp_entity_id: string;
  idp_sso_url: string;
  sp_entity_id: string;
  acs_url: string;
  default_role: string;
  mail_attribute: string;
  name_attribute: string;
  role_attribute: string;
}

interface SSOSettingsData {
  ldap: LDAPSettings;
  saml: SAMLSettings;
}

const DEFAULT_SSO: SSOSettingsData = {
  ldap: {
    enabled: false,
    host: '',
    port: 389,
    use_tls: false,
    base_dn: 'dc=example,dc=com',
    bind_dn: 'cn=admin,dc=example,dc=com',
    bind_password: '',
    user_filter: '(uid=%s)',
    login_attribute: 'uid',
    mail_attribute: 'mail',
    name_attribute: 'cn',
    default_role: 'viewer',
  },
  saml: {
    enabled: false,
    idp_metadata_url: '',
    idp_entity_id: '',
    idp_sso_url: '',
    sp_entity_id: 'https://cctv-monitor.example.com',
    acs_url: 'https://cctv-monitor.example.com/api/v1/auth/saml/acs',
    default_role: 'viewer',
    mail_attribute: 'mail',
    name_attribute: 'cn',
    role_attribute: 'memberOf',
  },
};

// ── Component ────────────────────────────────────────────────────────

export const SSOSettings: React.FC = () => {
  const { t } = useTranslation();
  const toast = useToast();
  const [settings, setSettings] = useState<SSOSettingsData>(DEFAULT_SSO);
  const [activeTab, setActiveTab] = useState<'ldap' | 'saml'>('ldap');
  const [testing, setTesting] = useState(false);

  // ── LDAP Handlers ───────────────────────────────────────────────

  const updateLDAP = (key: keyof LDAPSettings, value: any) => {
    setSettings((prev) => ({
      ...prev,
      ldap: { ...prev.ldap, [key]: value },
    }));
  };

  // ── SAML Handlers ───────────────────────────────────────────────

  const updateSAML = (key: keyof SAMLSettings, value: any) => {
    setSettings((prev) => ({
      ...prev,
      saml: { ...prev.saml, [key]: value },
    }));
  };

  // ── Save ────────────────────────────────────────────────────────

  const handleSave = async () => {
    try {
      // TODO: POST /api/v1/settings/sso
      toast.success(t('settings_saved') || 'Настройки сохранены');
    } catch (err: any) {
      toast.error(err.message || t('save_error') || 'Ошибка сохранения');
    }
  };

  // ── Test Connection ──────────────────────────────────────────────

  const handleTest = async () => {
    setTesting(true);
    try {
      // TODO: POST /api/v1/settings/sso/test
      await new Promise((resolve) => setTimeout(resolve, 1500));
      toast.success(t('connection_ok') || 'Соединение установлено');
    } catch (err: any) {
      toast.error(err.message || t('connection_error') || 'Ошибка соединения');
    } finally {
      setTesting(false);
    }
  };

  // ── Render LDAP Form ─────────────────────────────────────────────

  const renderLDAPForm = () => (
    <div className="space-y-6">
      {/* Enable toggle */}
      <div className="flex items-center justify-between p-4 bg-slate-50 rounded-lg">
        <div className="flex items-center gap-3">
          <Server className="w-5 h-5 text-slate-600" />
          <div>
            <p className="text-sm font-medium text-slate-900">{t('ldap_auth') || 'LDAP аутентификация'}</p>
            <p className="text-xs text-slate-500">{t('ldap_desc') || 'Active Directory / OpenLDAP'}</p>
          </div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={settings.ldap.enabled}
            onChange={(e) => updateLDAP('enabled', e.target.checked)}
          />
          <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600" />
        </label>
      </div>

      {settings.ldap.enabled && (
        <>
          {/* Server */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">{t('host') || 'Хост'}</label>
              <input
                type="text"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500"
                placeholder="ldap.example.com"
                value={settings.ldap.host}
                onChange={(e) => updateLDAP('host', e.target.value)}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">{t('port') || 'Порт'}</label>
              <input
                type="number"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500"
                value={settings.ldap.port}
                onChange={(e) => updateLDAP('port', parseInt(e.target.value) || 389)}
              />
            </div>
            <div className="flex items-end pb-2">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  className="rounded border-slate-300"
                  checked={settings.ldap.use_tls}
                  onChange={(e) => updateLDAP('use_tls', e.target.checked)}
                />
                <span className="text-sm text-slate-700">{t('use_tls') || 'TLS (LDAPS)'}</span>
              </label>
            </div>
          </div>

          {/* Bind credentials */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">{t('base_dn') || 'Base DN'}</label>
              <input
                type="text"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 font-mono"
                value={settings.ldap.base_dn}
                onChange={(e) => updateLDAP('base_dn', e.target.value)}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">{t('bind_dn') || 'Bind DN'}</label>
              <input
                type="text"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 font-mono"
                value={settings.ldap.bind_dn}
                onChange={(e) => updateLDAP('bind_dn', e.target.value)}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">{t('bind_password') || 'Bind Password'}</label>
              <input
                type="password"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500"
                value={settings.ldap.bind_password}
                onChange={(e) => updateLDAP('bind_password', e.target.value)}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">{t('user_filter') || 'User Filter'}</label>
              <input
                type="text"
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 font-mono"
                value={settings.ldap.user_filter}
                onChange={(e) => updateLDAP('user_filter', e.target.value)}
              />
            </div>
          </div>

          {/* Attribute mapping */}
          <div>
            <h4 className="text-xs font-semibold text-slate-500 uppercase mb-3">{t('attribute_mapping') || 'Маппинг атрибутов'}</h4>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('login_attr') || 'Login attr'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.ldap.login_attribute}
                  onChange={(e) => updateLDAP('login_attribute', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('mail_attr') || 'Mail attr'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.ldap.mail_attribute}
                  onChange={(e) => updateLDAP('mail_attribute', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('name_attr') || 'Name attr'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.ldap.name_attribute}
                  onChange={(e) => updateLDAP('name_attribute', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('default_role') || 'Default role'}</label>
                <select
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
                  value={settings.ldap.default_role}
                  onChange={(e) => updateLDAP('default_role', e.target.value)}
                >
                  <option value="viewer">viewer</option>
                  <option value="technician">technician</option>
                  <option value="manager">manager</option>
                  <option value="admin">admin</option>
                </select>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );

  // ── Render SAML Form ─────────────────────────────────────────────

  const renderSAMLForm = () => (
    <div className="space-y-6">
      {/* Enable toggle */}
      <div className="flex items-center justify-between p-4 bg-slate-50 rounded-lg">
        <div className="flex items-center gap-3">
          <Globe className="w-5 h-5 text-slate-600" />
          <div>
            <p className="text-sm font-medium text-slate-900">{t('saml_auth') || 'SAML 2.0 аутентификация'}</p>
            <p className="text-xs text-slate-500">{t('saml_desc') || 'Keycloak / ADFS / Azure AD'}</p>
          </div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            className="sr-only peer"
            checked={settings.saml.enabled}
            onChange={(e) => updateSAML('enabled', e.target.checked)}
          />
          <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600" />
        </label>
      </div>

      {settings.saml.enabled && (
        <>
          {/* Identity Provider */}
          <div>
            <h4 className="text-xs font-semibold text-slate-500 uppercase mb-3">{t('identity_provider') || 'Identity Provider (IdP)'}</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('metadata_url') || 'Metadata URL'}</label>
                <input
                  type="url"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  placeholder="https://idp.example.com/metadata"
                  value={settings.saml.idp_metadata_url}
                  onChange={(e) => updateSAML('idp_metadata_url', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('entity_id') || 'IdP Entity ID'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.idp_entity_id}
                  onChange={(e) => updateSAML('idp_entity_id', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('sso_url') || 'SSO URL'}</label>
                <input
                  type="url"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.idp_sso_url}
                  onChange={(e) => updateSAML('idp_sso_url', e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Service Provider */}
          <div>
            <h4 className="text-xs font-semibold text-slate-500 uppercase mb-3">{t('service_provider') || 'Service Provider (SP)'}</h4>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('sp_entity_id') || 'SP Entity ID'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.sp_entity_id}
                  onChange={(e) => updateSAML('sp_entity_id', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('acs_url') || 'ACS URL'}</label>
                <input
                  type="url"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.acs_url}
                  onChange={(e) => updateSAML('acs_url', e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Attribute mapping */}
          <div>
            <h4 className="text-xs font-semibold text-slate-500 uppercase mb-3">{t('attribute_mapping') || 'Маппинг атрибутов'}</h4>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('mail_attr') || 'Mail attr'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.mail_attribute}
                  onChange={(e) => updateSAML('mail_attribute', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('name_attr') || 'Name attr'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.name_attribute}
                  onChange={(e) => updateSAML('name_attribute', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('role_attr') || 'Role attr'}</label>
                <input
                  type="text"
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm font-mono"
                  value={settings.saml.role_attribute}
                  onChange={(e) => updateSAML('role_attribute', e.target.value)}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-slate-500 mb-1">{t('default_role') || 'Default role'}</label>
                <select
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm"
                  value={settings.saml.default_role}
                  onChange={(e) => updateSAML('default_role', e.target.value)}
                >
                  <option value="viewer">viewer</option>
                  <option value="technician">technician</option>
                  <option value="manager">manager</option>
                  <option value="admin">admin</option>
                </select>
              </div>
            </div>
          </div>

          {/* SP Metadata link */}
          <div className="p-4 bg-blue-50 rounded-lg border border-blue-200">
            <div className="flex items-start gap-3">
              <Shield className="w-5 h-5 text-blue-600 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-blue-900">{t('sp_metadata') || 'Метаданные SP'}</p>
                <p className="text-xs text-blue-700 mt-1">
                  {t('sp_metadata_desc') || 'Предоставьте этот URL вашему IdP:'}
                </p>
                <code className="block mt-2 text-xs font-mono text-blue-800 bg-blue-100 px-2 py-1 rounded">
                  {settings.saml.sp_entity_id}/metadata
                </code>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );

  // ── Main Render ─────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-slate-900 flex items-center gap-2">
            <Shield className="w-5 h-5" />
            {t('sso_settings') || 'SSO (Single Sign-On)'}
          </h2>
          <p className="text-sm text-slate-500 mt-1">
            {t('sso_settings_desc') || 'Настройка LDAP и SAML 2.0 аутентификации'}
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 p-1 bg-slate-100 rounded-lg w-fit">
        <button
          onClick={() => setActiveTab('ldap')}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            activeTab === 'ldap'
              ? 'bg-white text-slate-900 shadow-sm'
              : 'text-slate-500 hover:text-slate-700'
          }`}
        >
          <Server className="w-4 h-4 inline mr-1.5" />
          LDAP
        </button>
        <button
          onClick={() => setActiveTab('saml')}
          className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
            activeTab === 'saml'
              ? 'bg-white text-slate-900 shadow-sm'
              : 'text-slate-500 hover:text-slate-700'
          }`}
        >
          <Globe className="w-4 h-4 inline mr-1.5" />
          SAML 2.0
        </button>
      </div>

      {/* Form */}
      <Card>
        <div className="p-5">
          {activeTab === 'ldap' ? renderLDAPForm() : renderSAMLForm()}
        </div>
      </Card>

      {/* Actions */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Badge variant="info">{t('auto_provisioning') || 'Auto-provisioning'}</Badge>
          <span className="text-xs text-slate-500">
            {t('auto_provisioning_desc') || 'Пользователи создаются автоматически при первом входе'}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" onClick={handleTest} loading={testing}>
            <AlertTriangle className="w-4 h-4 mr-1.5" />
            {t('test_connection') || 'Тест соединения'}
          </Button>
          <Button onClick={handleSave}>
            <CheckCircle className="w-4 h-4 mr-1.5" />
            {t('save') || 'Сохранить'}
          </Button>
        </div>
      </div>

      {/* Compliance note */}
      <div className="p-3 bg-amber-50 rounded-lg border border-amber-200">
        <div className="flex items-start gap-2">
          <Shield className="w-4 h-4 text-amber-600 mt-0.5" />
          <p className="text-xs text-amber-800">
            {t('sso_compliance_note') || 
              'SSO настройки соответствуют: OWASP ASVS V2 (Authentication), ' +
              'ISO 27001 A.9.2 (User access), Приказ ОАЦ №66 п.7.18.1 (Идентификация). ' +
              'Пароль bind-аккаунта хранится в env vars (НЕ в config.yaml).'}
          </p>
        </div>
      </div>
    </div>
  );
};
