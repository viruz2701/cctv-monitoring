import React, { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Alert, Modal, Card } from '../ui';

// ═══════════════════════════════════════════════════════════════════════════
// WebAuthnSetup — компонент для настройки FIDO2/WebAuthn аутентификации.
//
// P1-SEC.1: Позволяет пользователю:
//   - Зарегистрировать hardware token (YubiKey)
//   - Использовать биометрию (Touch ID / Face ID)
//   - Просмотреть recovery codes
//   - Удалить credentials
// ═══════════════════════════════════════════════════════════════════════════

interface WebAuthnCredential {
  id: string;
  name: string;
  type: string;
  created_at: string;
}

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export const WebAuthnSetup: React.FC<Props> = ({ isOpen, onClose, onSuccess }) => {
  const { t } = useTranslation();
  const [credentials, setCredentials] = useState<WebAuthnCredential[]>([]);
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [showRecoveryCodes, setShowRecoveryCodes] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  // Регистрация WebAuthn credentials
  const handleRegister = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // 1. Получаем challenge с сервера
      const regResp = await fetch('/api/v1/auth/webauthn/register/begin', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
      });
      if (!regResp.ok) throw new Error(await regResp.text());
      const { creationOptions, sessionId } = await regResp.json();

      // 2. Вызываем браузерный WebAuthn API
      const cred = await navigator.credentials.create({
        publicKey: creationOptions,
      });

      if (!cred) throw new Error('User cancelled');

      // 3. Отправляем ответ серверу
      const finishResp = await fetch('/api/v1/auth/webauthn/register/finish', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
        body: JSON.stringify({ sessionId, credential: cred }),
      });
      if (!finishResp.ok) throw new Error(await finishResp.text());

      // 4. Загружаем recovery codes
      const codesResp = await fetch('/api/v1/auth/webauthn/recovery-codes', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
      });
      if (codesResp.ok) {
        const { codes } = await codesResp.json();
        setRecoveryCodes(codes);
        setShowRecoveryCodes(true);
      }

      await loadCredentials();
      onSuccess?.();
    } catch (err: any) {
      setError(err.message || 'Registration failed');
    } finally {
      setLoading(false);
    }
  }, [onSuccess]);

  // Загрузка списка credentials
  const loadCredentials = useCallback(async () => {
    try {
      const resp = await fetch('/api/v1/auth/webauthn/credentials', {
        credentials: 'include',
        headers: { 'X-CSRF-Token': getCSRFToken() },
      });
      if (resp.ok) {
        const data = await resp.json();
        setCredentials(data.credentials || []);
      }
    } catch {
      // silently fail
    }
  }, []);

  // Удаление credential
  const handleRemove = useCallback(async (credId: string) => {
    try {
      const resp = await fetch(`/api/v1/auth/webauthn/credentials/${credId}`, {
        method: 'DELETE',
        credentials: 'include',
        headers: { 'X-CSRF-Token': getCSRFToken() },
      });
      if (resp.ok) {
        await loadCredentials();
      }
    } catch {
      // silently fail
    }
  }, [loadCredentials]);

  // Копирование recovery codes
  const handleCopyCodes = useCallback(() => {
    navigator.clipboard.writeText(recoveryCodes.join('\n'));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [recoveryCodes]);

  // Загрузка при открытии
  React.useEffect(() => {
    if (isOpen) {
      loadCredentials();
      setShowRecoveryCodes(false);
      setError(null);
    }
  }, [isOpen, loadCredentials]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={t('webauthn.title')}>
      <div className="space-y-4">
        {error && <Alert variant="error">{error}</Alert>}

        {/* Current credentials */}
        <Card>
          <h3 className="text-lg font-medium mb-2">{t('webauthn.registered_devices')}</h3>
          {credentials.length === 0 ? (
            <p className="text-sm text-gray-500">{t('webauthn.no_devices')}</p>
          ) : (
            <ul className="space-y-2">
              {credentials.map((cred) => (
                <li key={cred.id} className="flex items-center justify-between py-1">
                  <span className="text-sm">
                    {cred.name || cred.type} — {new Date(cred.created_at).toLocaleDateString()}
                  </span>
                  <Button variant="danger" size="sm" onClick={() => handleRemove(cred.id)}>
                    {t('common.remove')}
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </Card>

        {/* Register new device */}
        <Button onClick={handleRegister} disabled={loading} className="w-full">
          {loading ? t('common.loading') : t('webauthn.register_device')}
        </Button>

        {/* Recovery codes modal */}
        {showRecoveryCodes && recoveryCodes.length > 0 && (
          <Modal isOpen={true} onClose={() => setShowRecoveryCodes(false)} title={t('webauthn.recovery_codes')}>
            <Alert variant="warning">{t('webauthn.recovery_warning')}</Alert>
            <div className="my-4 p-3 bg-gray-100 rounded font-mono text-sm space-y-1">
              {recoveryCodes.map((code, i) => (
                <div key={i}>{code}</div>
              ))}
            </div>
            <div className="flex gap-2">
              <Button onClick={handleCopyCodes}>
                {copied ? t('common.copied') : t('common.copy')}
              </Button>
              <Button variant="secondary" onClick={() => setShowRecoveryCodes(false)}>
                {t('common.close')}
              </Button>
            </div>
          </Modal>
        )}
      </div>
    </Modal>
  );
};

function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? decodeURIComponent(match[1]) : '';
}

export default WebAuthnSetup;
