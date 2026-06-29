import React, { useState, useEffect } from 'react';
import { Shield, Lock, Smartphone, MessageCircle, Key, CheckCircle, XCircle, Loader } from '../../components/ui/Icons';
import { Card, CardHeader, CardBody, CardFooter, Button, Input, Select, QRCode, useToast } from '../../components/ui';
import { useTranslation } from 'react-i18next';
import { api } from '../../services/api';
import type { AppSettings } from '../../types';

interface Props {
  security: AppSettings['security'];
  onChange: (security: AppSettings['security']) => void;
}

export const SecuritySettings: React.FC<Props> = ({ security, onChange }) => {
  const { t } = useTranslation();
  const toast = useToast();

  // TOTP 2FA state
  const [tfaEnabled, setTfaEnabled] = useState(false);
  const [tfaLoading, setTfaLoading] = useState(false);
  const [tfaSecret, setTfaSecret] = useState('');
  const [tfaUri, setTfaUri] = useState('');
  const [tfaQrDataUri, setTfaQrDataUri] = useState('');
  const [tfaVerificationCode, setTfaVerificationCode] = useState('');
  const [tfaSetupMode, setTfaSetupMode] = useState(false);
  const [tfaDisablePassword, setTfaDisablePassword] = useState('');

  // Telegram 2FA state
  const [tgStatus, setTgStatus] = useState<{ linked: boolean; alerts: boolean; tfa: boolean } | null>(null);
  const [tgLoading, setTgLoading] = useState(false);

  useEffect(() => {
    loadTelegramStatus();
    check2FAStatus();
  }, []);

  const check2FAStatus = async () => {
    try {
      const user = await api.getCurrentUser();
      setTfaEnabled(!!(user as any).twoFactorEnabled);
    } catch {
      // silently fail
    }
  };

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

  const handleSetupTOTP = async () => {
    try {
      setTfaLoading(true);
      const data = await api.setup2FA();
      setTfaSecret(data.secret);
      setTfaUri(data.uri);

      // Convert otpauth:// URI to a data URI for QR display
      // The QRCode component accepts the raw URI value
      setTfaQrDataUri(data.uri);
      setTfaSetupMode(true);
    } catch (err: any) {
      toast.error(err.message || t('2fa_setup_error') || 'Failed to setup 2FA');
    } finally {
      setTfaLoading(false);
    }
  };

  const handleVerifyTOTP = async () => {
    if (!tfaVerificationCode.trim()) return;
    try {
      setTfaLoading(true);
      await api.verify2FA(tfaVerificationCode);
      toast.success(t('2fa_enabled') || 'Two-factor authentication enabled');
      setTfaEnabled(true);
      setTfaSetupMode(false);
      setTfaVerificationCode('');
      setTfaSecret('');
      setTfaUri('');
      setTfaQrDataUri('');
    } catch (err: any) {
      toast.error(err.message || t('2fa_verify_error') || 'Invalid verification code');
    } finally {
      setTfaLoading(false);
    }
  };

  const handleDisable2FA = async () => {
    if (!tfaDisablePassword.trim()) {
      toast.error(t('password_required') || 'Password is required');
      return;
    }
    try {
      setTfaLoading(true);
      await api.disable2FA(tfaDisablePassword);
      toast.success(t('2fa_disabled') || 'Two-factor authentication disabled');
      setTfaEnabled(false);
      setTfaDisablePassword('');
    } catch (err: any) {
      toast.error(err.message || t('2fa_disable_error') || 'Failed to disable 2FA');
    } finally {
      setTfaLoading(false);
    }
  };

  const handleTelegramToggle = async (field: 'alerts' | 'tfa', value: boolean) => {
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

  return (
    <div className="space-y-6">
      {/* SECURITY SETTINGS (existing) */}
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

      {/* TWO-FACTOR AUTHENTICATION (TOTP) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Smartphone className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
          <span>{t('two_factor_authentication') || 'Two-Factor Authentication'}</span>
          {tfaEnabled && (
            <span className="ml-2 inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400">
              <CheckCircle className="w-3 h-3" />
              {t('enabled') || 'Enabled'}
            </span>
          )}
        </CardHeader>
        <CardBody>
          {!tfaSetupMode && !tfaEnabled && (
            <div className="text-center py-6">
              <div className="p-3 bg-indigo-50 dark:bg-indigo-900/20 rounded-full w-fit mx-auto mb-4">
                <Key className="w-8 h-8 text-indigo-600 dark:text-indigo-400" />
              </div>
              <p className="text-slate-700 dark:text-slate-300 mb-2">
                {t('2fa_description') || 'Enhance your account security by setting up two-factor authentication.'}
              </p>
              <p className="text-sm text-slate-500 dark:text-slate-400 mb-6">
                {t('2fa_scan_qr') || 'Scan the QR code with your authenticator app (Google Authenticator, Authy, etc.)'}
              </p>
              <Button
                icon={<Key className="w-4 h-4" />}
                onClick={handleSetupTOTP}
                disabled={tfaLoading}
              >
                {tfaLoading ? (
                  <><Loader className="w-4 h-4 animate-spin mr-2" />{t('setting_up') || 'Setting up...'}</>
                ) : (
                  t('setup_totp') || 'Setup TOTP'
                )}
              </Button>
            </div>
          )}

          {tfaSetupMode && (
            <div className="space-y-6">
              <div className="flex flex-col items-center">
                <QRCode value={tfaQrDataUri} size={200} label={t('scan_qr_code') || 'Scan this QR code'} />
              </div>

              <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-2">{t('secret_key') || 'Secret Key'}</p>
                <code className="block p-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-600 rounded text-sm font-mono break-all">
                  {tfaSecret}
                </code>
              </div>

              <div>
                <Input
                  label={t('verification_code') || 'Verification Code'}
                  placeholder={t('enter_6_digit_code') || 'Enter 6-digit code'}
                  value={tfaVerificationCode}
                  onChange={(e) => setTfaVerificationCode(e.target.value.replace(/\D/g, '').slice(0, 6))}
                  maxLength={6}
                />
              </div>

              <div className="flex gap-3">
                <Button
                  onClick={handleVerifyTOTP}
                  disabled={tfaVerificationCode.length !== 6 || tfaLoading}
                  icon={<CheckCircle className="w-4 h-4" />}
                >
                  {tfaLoading ? t('verifying') || 'Verifying...' : t('verify_enable') || 'Verify & Enable'}
                </Button>
                <Button
                  variant="outline"
                  onClick={() => {
                    setTfaSetupMode(false);
                    setTfaVerificationCode('');
                    setTfaSecret('');
                    setTfaUri('');
                    setTfaQrDataUri('');
                  }}
                >
                  {t('cancel')}
                </Button>
              </div>
            </div>
          )}

          {tfaEnabled && !tfaSetupMode && (
            <div className="space-y-4">
              <div className="flex items-center gap-3 p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <CheckCircle className="w-5 h-5 text-green-600 dark:text-green-400" />
                <div>
                  <p className="font-medium text-green-800 dark:text-green-200">
                    {t('2fa_active') || 'Two-factor authentication is active'}
                  </p>
                  <p className="text-sm text-green-600 dark:text-green-400">
                    {t('2fa_active_desc') || 'Your account is protected with TOTP-based 2FA'}
                  </p>
                </div>
              </div>
              <div className="max-w-sm">
                <Input
                  label={t('enter_password') || 'Enter your password'}
                  type="password"
                  placeholder={t('password_to_disable') || 'Password to disable 2FA'}
                  value={tfaDisablePassword}
                  onChange={(e) => setTfaDisablePassword(e.target.value)}
                />
              </div>
              <Button
                variant="outline"
                className="text-red-600 border-red-200 hover:bg-red-50 dark:text-red-400 dark:border-red-900/30 dark:hover:bg-red-900/20"
                onClick={handleDisable2FA}
                disabled={!tfaDisablePassword.trim() || tfaLoading}
                icon={<XCircle className="w-4 h-4" />}
              >
                {tfaLoading ? t('disabling') || 'Disabling...' : t('disable_2fa') || 'Disable 2FA'}
              </Button>
            </div>
          )}
        </CardBody>
      </Card>

      {/* TELEGRAM 2FA */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <MessageCircle className="w-5 h-5 text-sky-600 dark:text-sky-400" />
          <span>{t('telegram_2fa') || 'Telegram 2FA'}</span>
        </CardHeader>
        <CardBody>
          {tgLoading && !tgStatus ? (
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
                    {t('telegram_connected_desc') || 'Your Telegram account is linked'}
                  </p>
                </div>
              </div>
              <div className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg">
                <div>
                  <p className="font-medium text-slate-900 dark:text-white">
                    {t('telegram_2fa_alerts') || 'Telegram alerts'}
                  </p>
                  <p className="text-sm text-slate-500 dark:text-slate-400">
                    {t('telegram_2fa_alerts_desc') || 'Receive 2FA codes via Telegram'}
                  </p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    className="sr-only peer"
                    checked={tgStatus.tfa}
                    onChange={(e) => handleTelegramToggle('tfa', e.target.checked)}
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
                {t('telegram_not_connected_desc') || 'Connect your Telegram account to receive 2FA codes via Telegram'}
              </p>
              <Button
                icon={<MessageCircle className="w-4 h-4" />}
                onClick={async () => {
                  try {
                    const data = await api.generateTelegramLink();
                    window.open(`https://t.me/${data.token}`, '_blank');
                    toast.success(t('telegram_link_generated') || 'Telegram link generated! Check your Telegram app.');
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
