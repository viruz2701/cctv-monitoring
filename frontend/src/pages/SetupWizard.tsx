import { useState, useEffect, useCallback } from 'react';
import { RegionSelector, type RegionInfo } from '../components/setup/RegionSelector';

const API_BASE = '/api/v1';

interface WizardStatus {
  started: boolean;
  completed: boolean;
  step: number;
  config?: Record<string, unknown>;
}

interface StepInfo {
  step: number;
  name: string;
  description: string;
}

const STEPS: StepInfo[] = [
  { step: 1, name: 'Region', description: 'Select deployment region' },
  { step: 2, name: 'Cryptography', description: 'Confirm crypto parameters' },
  { step: 3, name: 'Storage', description: 'Configure storage' },
  { step: 4, name: 'Admin', description: 'Create admin account' },
  { step: 5, name: 'Network', description: 'Configure network' },
  { step: 6, name: 'Notifications', description: 'Configure notifications' },
  { step: 7, name: 'Review', description: 'Review and complete' },
];

async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(err.error || 'Request failed');
  }
  return res.json();
}

async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(err.error || 'Request failed');
  }
  return res.json();
}

export function SetupWizard() {
  const [status, setStatus] = useState<WizardStatus | null>(null);
  const [regions, setRegions] = useState<RegionInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [stepData, setStepData] = useState<Record<string, unknown>>({});

  // Form state
  const [selectedRegion, setSelectedRegion] = useState('');
  const [cryptoConfirmed, setCryptoConfirmed] = useState(false);
  const [storageType, setStorageType] = useState<'local' | 's3'>('local');
  const [s3Endpoint, setS3Endpoint] = useState('');
  const [s3Bucket, setS3Bucket] = useState('');
  const [s3Region, setS3Region] = useState('');
  const [adminUsername, setAdminUsername] = useState('');
  const [adminEmail, setAdminEmail] = useState('');
  const [adminSignature, setAdminSignature] = useState('');
  const [apiPort, setApiPort] = useState(8080);
  const [tlsCert, setTlsCert] = useState('');
  const [tlsKey, setTlsKey] = useState('');
  const [telegramToken, setTelegramToken] = useState('');
  const [smtpHost, setSmtpHost] = useState('');
  const [smtpPort, setSmtpPort] = useState(587);
  const [smtpUsername, setSmtpUsername] = useState('');
  const [completing, setCompleting] = useState(false);

  const loadStatus = useCallback(async () => {
    try {
      const [statusData, regionsData] = await Promise.all([
        apiGet<WizardStatus>('/setup/status'),
        apiGet<{ regions: RegionInfo[] }>('/setup/regions'),
      ]);
      setStatus(statusData);
      setRegions(regionsData.regions);
      if (!statusData.started && !statusData.completed) {
        await apiPost('/setup/start');
        setStatus({ started: true, completed: false, step: 1 });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load setup');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStatus();
  }, [loadStatus]);

  const handleNext = useCallback(async () => {
    if (!status) return;
    setError(null);

    try {
      switch (status.step) {
        case 1: {
          if (!selectedRegion) {
            setError('Please select a region');
            return;
          }
          const data = await apiPost<{ step: number }>('/setup/region', { region: selectedRegion });
          setStatus({ ...status, step: data.step });
          break;
        }
        case 2: {
          if (!cryptoConfirmed) {
            setError('Please confirm cryptographic parameters');
            return;
          }
          const data = await apiPost<{ step: number }>('/setup/crypto', { confirmed: true });
          setStatus({ ...status, step: data.step });
          break;
        }
        case 3: {
          const data = await apiPost<{ step: number }>('/setup/storage', {
            type: storageType,
            s3_endpoint: s3Endpoint,
            s3_bucket: s3Bucket,
            s3_region: s3Region,
          });
          setStatus({ ...status, step: data.step });
          break;
        }
        case 4: {
          if (!adminUsername || !adminEmail) {
            setError('Username and email are required');
            return;
          }
          const data = await apiPost<{ step: number }>('/setup/admin', {
            username: adminUsername,
            email: adminEmail,
            signature: adminSignature || undefined,
          });
          setStatus({ ...status, step: data.step });
          break;
        }
        case 5: {
          const data = await apiPost<{ step: number }>('/setup/network', {
            api_port: apiPort,
            tls_cert: tlsCert || undefined,
            tls_key: tlsKey || undefined,
          });
          setStatus({ ...status, step: data.step });
          break;
        }
        case 6: {
          const data = await apiPost<{ step: number }>('/setup/notifications', {
            telegram_token: telegramToken || undefined,
            smtp_host: smtpHost || undefined,
            smtp_port: smtpPort,
            smtp_username: smtpUsername || undefined,
          });
          setStatus({ ...status, step: data.step });
          break;
        }
        case 7: {
          setCompleting(true);
          const data = await apiPost<{ completed: boolean; region: string; region_locked: boolean }>(
            '/setup/complete'
          );
          setStatus({ ...status, completed: true, step: 0 });
          setStepData({ ...data });
          break;
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Step failed');
    } finally {
      setCompleting(false);
    }
  }, [status, selectedRegion, cryptoConfirmed, storageType, s3Endpoint, s3Bucket, s3Region,
      adminUsername, adminEmail, adminSignature, apiPort, tlsCert, tlsKey,
      telegramToken, smtpHost, smtpPort, smtpUsername]);

  const handleSkip = useCallback(async () => {
    if (!status) return;
    setError(null);
    try {
      if (status.step === 5) {
        await apiPost('/setup/network', { api_port: 8080 });
        setStatus({ ...status, step: 6 });
      } else if (status.step === 6) {
        await apiPost('/setup/notifications', {});
        setStatus({ ...status, step: 7 });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Skip failed');
    }
  }, [status]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="text-center" role="status">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4" />
          <p className="text-gray-500 dark:text-gray-400">Loading setup wizard...</p>
        </div>
      </div>
    );
  }

  if (status?.completed) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="max-w-md mx-auto p-8 bg-white dark:bg-gray-800 rounded-lg shadow-lg text-center">
          <div className="text-4xl mb-4">✅</div>
          <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-4">
            Setup Complete
          </h1>
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            Region <strong>{stepData.region as string}</strong> has been configured.
          </p>
          <p className="text-sm text-gray-400 dark:text-gray-500 mb-6">
            Region is now locked and cannot be changed without a full data migration.
          </p>
          <a
            href="/login"
            className="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md
              hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Go to Login
          </a>
        </div>
      </div>
    );
  }

  const currentStep = STEPS.find((s) => s.step === status?.step) || STEPS[0];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 py-8 px-4">
      <div className="max-w-3xl mx-auto">
        {/* Progress indicator */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-2">
            <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">
              Setup Wizard
            </h1>
            <span className="text-sm text-gray-500 dark:text-gray-400">
              Step {status?.step} of 7
            </span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-500"
              style={{ width: `${((status?.step || 1) / 7) * 100}%` }}
              role="progressbar"
              aria-valuenow={status?.step || 1}
              aria-valuemin={1}
              aria-valuemax={7}
            />
          </div>
          <div className="flex justify-between mt-2">
            {STEPS.map((s) => (
              <div
                key={s.step}
                className={`text-xs ${
                  (status?.step || 0) >= s.step
                    ? 'text-blue-600 dark:text-blue-400 font-medium'
                    : 'text-gray-400 dark:text-gray-600'
                }`}
              >
                {s.name}
              </div>
            ))}
          </div>
        </div>

        {/* Error banner */}
        {error && (
          <div
            className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800
              rounded-lg text-sm text-red-700 dark:text-red-300"
            role="alert"
          >
            {error}
            <button
              type="button"
              className="ml-2 text-red-500 hover:text-red-700"
              onClick={() => setError(null)}
              aria-label="Dismiss error"
            >
              ✕
            </button>
          </div>
        )}

        {/* Step content */}
        <div className="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-1">
            {currentStep.name}
          </h2>
          <p className="text-sm text-gray-500 dark:text-gray-400 mb-6">
            {currentStep.description}
          </p>

          {status?.step === 1 && (
            <RegionSelector
              regions={regions}
              selected={selectedRegion}
              onSelect={setSelectedRegion}
            />
          )}

          {status?.step === 2 && (
            <div className="space-y-4">
              <div className="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-2">
                  Cryptographic Parameters
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  The following cryptographic algorithms will be used based on the selected region.
                  These are automatically configured and comply with regional standards.
                </p>
              </div>
              <label className="flex items-center space-x-3">
                <input
                  type="checkbox"
                  checked={cryptoConfirmed}
                  onChange={(e) => setCryptoConfirmed(e.target.checked)}
                  className="h-4 w-4 text-blue-600 rounded border-gray-300"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  I confirm the cryptographic parameters for the selected region
                </span>
              </label>
            </div>
          )}

          {status?.step === 3 && (
            <div className="space-y-4">
              <div className="flex space-x-4">
                <button
                  type="button"
                  onClick={() => setStorageType('local')}
                  className={`flex-1 p-4 border rounded-lg text-center transition-all
                    ${storageType === 'local'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300'
                    }`}
                >
                  <div className="text-2xl mb-2">💾</div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">Local Storage</div>
                  <div className="text-xs text-gray-500 mt-1">Store data on local filesystem</div>
                </button>
                <button
                  type="button"
                  onClick={() => setStorageType('s3')}
                  className={`flex-1 p-4 border rounded-lg text-center transition-all
                    ${storageType === 's3'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300'
                    }`}
                >
                  <div className="text-2xl mb-2">☁️</div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">S3 Storage</div>
                  <div className="text-xs text-gray-500 mt-1">Use S3-compatible object storage</div>
                </button>
              </div>
              {storageType === 's3' && (
                <div className="space-y-3 p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                  <input
                    type="text"
                    placeholder="S3 Endpoint"
                    value={s3Endpoint}
                    onChange={(e) => setS3Endpoint(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                      bg-white dark:bg-gray-800 text-sm"
                    aria-label="S3 Endpoint"
                  />
                  <input
                    type="text"
                    placeholder="S3 Bucket"
                    value={s3Bucket}
                    onChange={(e) => setS3Bucket(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                      bg-white dark:bg-gray-800 text-sm"
                    aria-label="S3 Bucket"
                  />
                  <input
                    type="text"
                    placeholder="S3 Region (optional)"
                    value={s3Region}
                    onChange={(e) => setS3Region(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                      bg-white dark:bg-gray-800 text-sm"
                    aria-label="S3 Region"
                  />
                </div>
              )}
            </div>
          )}

          {status?.step === 4 && (
            <div className="space-y-4">
              <input
                type="text"
                placeholder="Admin Username"
                value={adminUsername}
                onChange={(e) => setAdminUsername(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                  bg-white dark:bg-gray-800 text-sm"
                aria-label="Admin Username"
                required
              />
              <input
                type="email"
                placeholder="Admin Email"
                value={adminEmail}
                onChange={(e) => setAdminEmail(e.target.value)}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                  bg-white dark:bg-gray-800 text-sm"
                aria-label="Admin Email"
                required
              />
              {selectedRegion === 'BY' && (
                <div>
                  <input
                    type="text"
                    placeholder="Digital Signature (required for КИИ)"
                    value={adminSignature}
                    onChange={(e) => setAdminSignature(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                      bg-white dark:bg-gray-800 text-sm"
                    aria-label="Digital Signature"
                  />
                  <p className="mt-1 text-xs text-gray-500">
                    Digital signature is required for КИИ (Республика Беларусь) region
                  </p>
                </div>
              )}
            </div>
          )}

          {status?.step === 5 && (
            <div className="space-y-4">
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">API Port</span>
                <input
                  type="number"
                  value={apiPort}
                  onChange={(e) => setApiPort(parseInt(e.target.value) || 8080)}
                  min={1}
                  max={65535}
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="API Port"
                />
              </label>
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">TLS Certificate Path (optional)</span>
                <input
                  type="text"
                  value={tlsCert}
                  onChange={(e) => setTlsCert(e.target.value)}
                  placeholder="/etc/ssl/certs/server.crt"
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="TLS Certificate"
                />
              </label>
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">TLS Key Path (optional)</span>
                <input
                  type="text"
                  value={tlsKey}
                  onChange={(e) => setTlsKey(e.target.value)}
                  placeholder="/etc/ssl/private/server.key"
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="TLS Key"
                />
              </label>
            </div>
          )}

          {status?.step === 6 && (
            <div className="space-y-4">
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">Telegram Bot Token (optional)</span>
                <input
                  type="password"
                  value={telegramToken}
                  onChange={(e) => setTelegramToken(e.target.value)}
                  placeholder="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="Telegram Bot Token"
                />
              </label>
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">SMTP Host (optional)</span>
                <input
                  type="text"
                  value={smtpHost}
                  onChange={(e) => setSmtpHost(e.target.value)}
                  placeholder="smtp.example.com"
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="SMTP Host"
                />
              </label>
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">SMTP Port</span>
                <input
                  type="number"
                  value={smtpPort}
                  onChange={(e) => setSmtpPort(parseInt(e.target.value) || 587)}
                  min={1}
                  max={65535}
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="SMTP Port"
                />
              </label>
            </div>
          )}

          {status?.step === 7 && (
            <div className="space-y-4">
              <div className="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <h3 className="font-medium text-gray-900 dark:text-gray-100 mb-2">
                  Review Configuration
                </h3>
                <p className="text-sm text-gray-500 dark:text-gray-400">
                  Please review the configuration before completing the setup.
                  The region selection will be locked after completion.
                </p>
              </div>
              <div className="space-y-2 text-sm">
                <p><strong>Region:</strong> {selectedRegion}</p>
                <p><strong>Storage:</strong> {storageType === 's3' ? `S3 (${s3Bucket})` : 'Local'}</p>
                <p><strong>Admin:</strong> {adminUsername} ({adminEmail})</p>
                <p><strong>API Port:</strong> {apiPort}</p>
              </div>
            </div>
          )}

          {/* Navigation buttons */}
          <div className="mt-8 flex items-center justify-between">
            <div>
              {(status?.step === 5 || status?.step === 6) && (
                <button
                  type="button"
                  onClick={handleSkip}
                  className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700
                    dark:hover:text-gray-300 focus:outline-none"
                >
                  Skip this step
                </button>
              )}
            </div>
            <div className="flex space-x-3">
              <button
                type="button"
                onClick={handleNext}
                disabled={completing}
                className="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700
                  focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50
                  disabled:cursor-not-allowed transition-colors"
              >
                {completing ? (
                  <span className="flex items-center">
                    <span className="animate-spin mr-2">⏳</span>
                    Completing...
                  </span>
                ) : status?.step === 7 ? (
                  'Complete Setup'
                ) : (
                  'Continue'
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
