import { useState, useEffect, useCallback, useRef } from 'react';
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

interface ComplianceReport {
  region: string;
  region_name: string;
  compliance: string[];
  crypto: string;
  hash: string;
  signature: string;
  storage_type: string;
  admin: string;
  completed_at: string;
}

const KII_REGIONS = ['BY', 'RU'];

const STEPS: StepInfo[] = [
  { step: 1, name: 'Region', description: 'Select deployment region and compliance profile' },
  { step: 2, name: 'Cryptography', description: 'Confirm cryptographic parameters for the region' },
  { step: 3, name: 'Storage', description: 'Configure data storage backend' },
  { step: 4, name: 'Admin', description: 'Create initial administrator account' },
  { step: 5, name: 'Network', description: 'Configure TLS, ports, and network settings' },
  { step: 6, name: 'Notifications', description: 'Configure Telegram, Email, SMS notifications' },
  { step: 7, name: 'Review', description: 'Review configuration and complete setup' },
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
  const [complianceReport, setComplianceReport] = useState<ComplianceReport | null>(null);
  const mainRef = useRef<HTMLElement>(null);

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
  const [signatureConfirmed, setSignatureConfirmed] = useState(false);
  const [apiPort, setApiPort] = useState(8080);
  const [tlsCert, setTlsCert] = useState('');
  const [tlsKey, setTlsKey] = useState('');
  const [telegramToken, setTelegramToken] = useState('');
  const [smtpHost, setSmtpHost] = useState('');
  const [smtpPort, setSmtpPort] = useState(587);
  const [smtpUsername, setSmtpUsername] = useState('');
  const [completing, setCompleting] = useState(false);

  const isKIIRegion = KII_REGIONS.includes(selectedRegion);

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

  // Focus management: focus main content on step change
  useEffect(() => {
    if (!loading && mainRef.current) {
      mainRef.current.focus();
    }
  }, [status?.step, loading]);

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
          if (isKIIRegion && !adminSignature) {
            setError('Digital signature is required for КИИ regions (BY, RU)');
            return;
          }
          if (isKIIRegion && !signatureConfirmed) {
            setError('Please confirm the digital signature acknowledgment');
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
          const data = await apiPost<{
            completed: boolean;
            region: string;
            region_locked: boolean;
            completed_at: string;
            compliance_report?: string;
          }>('/setup/complete');
          setStatus({ ...status, completed: true, step: 0 });
          setStepData({ ...data });

          // Build compliance report
          const selectedRegionInfo = regions.find((r) => r.region === selectedRegion);
          setComplianceReport({
            region: data.region,
            region_name: selectedRegionInfo?.name || data.region,
            compliance: selectedRegionInfo?.compliance || [],
            crypto: selectedRegionInfo?.crypto_info.encryption || '',
            hash: selectedRegionInfo?.crypto_info.hash || '',
            signature: selectedRegionInfo?.crypto_info.signature || '',
            storage_type: storageType,
            admin: adminUsername,
            completed_at: data.completed_at || new Date().toISOString(),
          });
          break;
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Step failed');
    } finally {
      setCompleting(false);
    }
  }, [status, selectedRegion, cryptoConfirmed, storageType, s3Endpoint, s3Bucket, s3Region,
      adminUsername, adminEmail, adminSignature, signatureConfirmed, apiPort, tlsCert, tlsKey,
      telegramToken, smtpHost, smtpPort, smtpUsername, isKIIRegion, regions]);

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
        <div className="text-center" role="status" aria-live="polite">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500 mx-auto mb-4" aria-hidden="true" />
          <p className="text-gray-500 dark:text-gray-400">Loading setup wizard...</p>
        </div>
      </div>
    );
  }

  if (status?.completed) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 py-8 px-4">
        <div className="max-w-2xl mx-auto p-8 bg-white dark:bg-gray-800 rounded-lg shadow-lg">
          {/* Completion success */}
          <div className="text-center mb-8">
            <div className="text-5xl mb-4" aria-hidden="true">✅</div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100 mb-2">
              Setup Complete
            </h1>
            <p className="text-gray-500 dark:text-gray-400">
              Region <strong>{stepData.region as string}</strong> has been configured and locked.
            </p>
          </div>

          {/* Compliance Report */}
          {complianceReport && (
            <div className="mb-8 p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg border border-gray-200 dark:border-gray-700">
              <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
                Compliance Report
              </h2>
              <div className="space-y-3 text-sm">
                <div className="flex justify-between py-1 border-b border-gray-200 dark:border-gray-700">
                  <span className="text-gray-500 dark:text-gray-400">Region</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">{complianceReport.region} - {complianceReport.region_name}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-gray-200 dark:border-gray-700">
                  <span className="text-gray-500 dark:text-gray-400">Encryption</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">{complianceReport.crypto}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-gray-200 dark:border-gray-700">
                  <span className="text-gray-500 dark:text-gray-400">Hash Algorithm</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">{complianceReport.hash}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-gray-200 dark:border-gray-700">
                  <span className="text-gray-500 dark:text-gray-400">Signature Algorithm</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">{complianceReport.signature}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-gray-200 dark:border-gray-700">
                  <span className="text-gray-500 dark:text-gray-400">Storage</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">{complianceReport.storage_type}</span>
                </div>
                <div className="flex justify-between py-1 border-b border-gray-200 dark:border-gray-700">
                  <span className="text-gray-500 dark:text-gray-400">Administrator</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">{complianceReport.admin}</span>
                </div>
                <div className="flex justify-between py-1">
                  <span className="text-gray-500 dark:text-gray-400">Completed At</span>
                  <span className="font-medium text-gray-900 dark:text-gray-100">
                    {new Date(complianceReport.completed_at).toLocaleString()}
                  </span>
                </div>
              </div>

              {/* Compliance standards badges */}
              {complianceReport.compliance.length > 0 && (
                <div className="mt-4">
                  <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">Compliance Standards</p>
                  <div className="flex flex-wrap gap-1.5">
                    {complianceReport.compliance.map((std) => (
                      <span
                        key={std}
                        className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium
                          bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300"
                      >
                        {std}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          <p className="text-sm text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 p-3 rounded-lg mb-6">
            <span aria-hidden="true">🔒</span> Region is now locked. It cannot be changed without a full data migration and re-installation.
          </p>

          <a
            href="/login"
            className="block w-full text-center px-4 py-3 bg-blue-600 text-white rounded-md
              hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
              transition-colors"
          >
            Proceed to Login
          </a>
        </div>
      </div>
    );
  }

  const currentStep = STEPS.find((s) => s.step === status?.step) || STEPS[0];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Skip to content link for accessibility */}
      <a
        href="#setup-main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4
          focus:z-50 focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white focus:rounded-md"
      >
        Skip to main content
      </a>

      <div className="max-w-3xl mx-auto py-8 px-4">
        {/* Progress indicator */}
        <div className="mb-8" role="navigation" aria-label="Setup progress">
          <div className="flex items-center justify-between mb-2">
            <h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">
              Setup Wizard
            </h1>
            <span className="text-sm text-gray-500 dark:text-gray-400" aria-live="polite">
              Step {status?.step} of 7
            </span>
          </div>
          <div
            className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2"
            role="progressbar"
            aria-valuenow={status?.step || 1}
            aria-valuemin={1}
            aria-valuemax={7}
            aria-label={`Step ${status?.step} of 7`}
          >
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-500"
              style={{ width: `${((status?.step || 1) / 7) * 100}%` }}
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
                aria-current={status?.step === s.step ? 'step' : undefined}
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
            aria-live="assertive"
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
        <main
          id="setup-main-content"
          ref={mainRef}
          tabIndex={-1}
          className="bg-white dark:bg-gray-800 rounded-lg shadow p-6 outline-none"
          aria-label={`Step ${currentStep.step}: ${currentStep.name}`}
        >
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
                  This selection is legally binding for your deployment.
                </p>
              </div>
              <label className="flex items-start space-x-3 cursor-pointer">
                <input
                  type="checkbox"
                  checked={cryptoConfirmed}
                  onChange={(e) => setCryptoConfirmed(e.target.checked)}
                  className="h-4 w-4 mt-0.5 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
                  aria-describedby="crypto-description"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300" id="crypto-description">
                  I confirm the cryptographic parameters for the selected region. I understand that
                  these settings are legally binding and cannot be changed after setup without a full
                  data migration.
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
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 ring-2 ring-blue-500'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300'
                    }`}
                  aria-pressed={storageType === 'local'}
                  aria-label="Local storage"
                >
                  <div className="text-2xl mb-2" aria-hidden="true">💾</div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">Local Storage</div>
                  <div className="text-xs text-gray-500 mt-1">Store data on local filesystem</div>
                </button>
                <button
                  type="button"
                  onClick={() => setStorageType('s3')}
                  className={`flex-1 p-4 border rounded-lg text-center transition-all
                    ${storageType === 's3'
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20 ring-2 ring-blue-500'
                      : 'border-gray-200 dark:border-gray-700 hover:border-gray-300'
                    }`}
                  aria-pressed={storageType === 's3'}
                  aria-label="S3 storage"
                >
                  <div className="text-2xl mb-2" aria-hidden="true">☁️</div>
                  <div className="font-medium text-gray-900 dark:text-gray-100">S3 Storage</div>
                  <div className="text-xs text-gray-500 mt-1">Use S3-compatible object storage</div>
                </button>
              </div>
              {storageType === 's3' && (
                <div className="space-y-3 p-4 bg-gray-50 dark:bg-gray-900/50 rounded-lg">
                  <label className="block">
                    <span className="sr-only">S3 Endpoint</span>
                    <input
                      type="text"
                      placeholder="S3 Endpoint"
                      value={s3Endpoint}
                      onChange={(e) => setS3Endpoint(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                        bg-white dark:bg-gray-800 text-sm"
                      aria-label="S3 Endpoint"
                    />
                  </label>
                  <label className="block">
                    <span className="sr-only">S3 Bucket</span>
                    <input
                      type="text"
                      placeholder="S3 Bucket"
                      value={s3Bucket}
                      onChange={(e) => setS3Bucket(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                        bg-white dark:bg-gray-800 text-sm"
                      aria-label="S3 Bucket"
                    />
                  </label>
                  <label className="block">
                    <span className="sr-only">S3 Region</span>
                    <input
                      type="text"
                      placeholder="S3 Region (optional)"
                      value={s3Region}
                      onChange={(e) => setS3Region(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                        bg-white dark:bg-gray-800 text-sm"
                      aria-label="S3 Region"
                    />
                  </label>
                </div>
              )}
            </div>
          )}

          {status?.step === 4 && (
            <div className="space-y-4">
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">Admin Username</span>
                <input
                  type="text"
                  placeholder="e.g., admin"
                  value={adminUsername}
                  onChange={(e) => setAdminUsername(e.target.value)}
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="Admin Username"
                  required
                />
              </label>
              <label className="block">
                <span className="text-sm text-gray-700 dark:text-gray-300">Admin Email</span>
                <input
                  type="email"
                  placeholder="e.g., admin@example.com"
                  value={adminEmail}
                  onChange={(e) => setAdminEmail(e.target.value)}
                  className="mt-1 w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                    bg-white dark:bg-gray-800 text-sm"
                  aria-label="Admin Email"
                  required
                />
              </label>

              {/* Digital signature for КИИ regions (BY, RU) */}
              {isKIIRegion && (
                <div className="p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-lg">
                  <h4 className="font-medium text-amber-800 dark:text-amber-200 mb-2">
                    Digital Signature Required (КИИ Region)
                  </h4>
                  <p className="text-xs text-amber-600 dark:text-amber-400 mb-3">
                    {selectedRegion === 'BY'
                      ? 'СТБ 34.101.45 requires a digital signature (bign-curve256v1) for КИИ compliance.'
                      : 'ГОСТ Р 34.10-2012 requires a digital signature for ФСТЭК compliance.'}
                    {' '}This signature verifies the administrator's identity and authorizes the
                    setup configuration.
                  </p>
                  <label className="block">
                    <span className="sr-only">Digital Signature</span>
                    <input
                      type="text"
                      placeholder="Enter your digital signature"
                      value={adminSignature}
                      onChange={(e) => setAdminSignature(e.target.value)}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                        bg-white dark:bg-gray-800 text-sm"
                      aria-label="Digital Signature for КИИ"
                    />
                  </label>
                  <label className="flex items-start space-x-3 mt-3 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={signatureConfirmed}
                      onChange={(e) => setSignatureConfirmed(e.target.checked)}
                      className="h-4 w-4 mt-0.5 text-amber-600 rounded border-amber-300 focus:ring-amber-500"
                    />
                    <span className="text-xs text-amber-700 dark:text-amber-300">
                      I confirm that the digital signature is valid and I am authorized to perform
                      this setup. I understand that this action is legally binding.
                    </span>
                  </label>
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
              <p className="text-xs text-gray-500 dark:text-gray-400">
                These settings can be changed later in the system settings.
              </p>
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
              <fieldset className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                <legend className="text-sm font-medium text-gray-700 dark:text-gray-300 px-1">
                  SMTP Configuration (optional)
                </legend>
                <div className="space-y-3 mt-3">
                  <label className="block">
                    <span className="sr-only">SMTP Host</span>
                    <input
                      type="text"
                      value={smtpHost}
                      onChange={(e) => setSmtpHost(e.target.value)}
                      placeholder="smtp.example.com"
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                        bg-white dark:bg-gray-800 text-sm"
                      aria-label="SMTP Host"
                    />
                  </label>
                  <label className="block">
                    <span className="sr-only">SMTP Port</span>
                    <input
                      type="number"
                      value={smtpPort}
                      onChange={(e) => setSmtpPort(parseInt(e.target.value) || 587)}
                      min={1}
                      max={65535}
                      className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md
                        bg-white dark:bg-gray-800 text-sm"
                      aria-label="SMTP Port"
                    />
                  </label>
                </div>
              </fieldset>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                Notifications can be configured later in the system settings.
              </p>
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
                  The region selection will be <strong>permanently locked</strong> after completion.
                </p>
              </div>

              <div className="space-y-3 text-sm">
                <div className="flex justify-between py-2 border-b border-gray-100 dark:border-gray-700">
                  <span className="text-gray-500">Region</span>
                  <span className="font-medium">{selectedRegion}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-gray-100 dark:border-gray-700">
                  <span className="text-gray-500">Storage</span>
                  <span className="font-medium">{storageType === 's3' ? `S3 (${s3Bucket})` : 'Local'}</span>
                </div>
                <div className="flex justify-between py-2 border-b border-gray-100 dark:border-gray-700">
                  <span className="text-gray-500">Admin</span>
                  <span className="font-medium">{adminUsername} ({adminEmail})</span>
                </div>
                <div className="flex justify-between py-2 border-b border-gray-100 dark:border-gray-700">
                  <span className="text-gray-500">API Port</span>
                  <span className="font-medium">{apiPort}</span>
                </div>
                {isKIIRegion && (
                  <div className="flex justify-between py-2 border-b border-gray-100 dark:border-gray-700">
                    <span className="text-gray-500">Digital Signature</span>
                    <span className="font-medium text-amber-600">{adminSignature ? '✓ Provided' : '—'}</span>
                  </div>
                )}
              </div>

              <div className="p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg text-xs text-yellow-800 dark:text-yellow-200">
                <span aria-hidden="true">⚠️</span> By completing this setup, you confirm that all
                configuration settings are correct. The region selection cannot be changed after
                this step. A compliance report will be generated for audit purposes.
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
                    dark:hover:text-gray-300 focus:outline-none focus:underline"
                  aria-label={`Skip ${currentStep.name} step`}
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
                  focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                  disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                aria-label={status?.step === 7 ? 'Complete setup' : `Continue to next step`}
              >
                {completing ? (
                  <span className="flex items-center">
                    <span className="animate-spin mr-2" aria-hidden="true">⏳</span>
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
        </main>
      </div>
    </div>
  );
}
