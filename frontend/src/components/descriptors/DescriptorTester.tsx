// ═══════════════════════════════════════════════════════════════════════
// DescriptorTester — тестирование дескриптора (PROTO-06)
//
// Позволяет ввести IP/URL устройства, credentials и выполнить
// тестовый запрос к выбранному endpoint'у, чтобы увидеть результат.
//
// Compliance:
//   - WCAG 2.1 AA (labels, aria, loading states)
//   - OWASP ASVS V5 (input validation на стороне клиента)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Play,
  Loader2,
  CheckCircle,
  XCircle,
  Clock,
} from '../../components/ui/Icons';
import {
  Button,
  Input,
  Select,
  Alert,
} from '../../components/ui';
import {
  useCurrentDescriptor,
  useDescriptorStore,
} from '../../store/descriptorStore';
import { descriptorsApi } from '../../services/api/descriptors';
import type {
  DescriptorTestResponse,
  DescriptorTestRequest,
} from '../../types/descriptor';

// ─── Component ──────────────────────────────────────────────────────

export function DescriptorTester() {
  const { t } = useTranslation();
  const descriptor = useCurrentDescriptor();
  const error = useDescriptorStore((s) => s.error);

  // ─── Form state ────────────────────────────────────────────
  const [baseUrl, setBaseUrl] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [token, setToken] = useState('');
  const [selectedEndpointId, setSelectedEndpointId] = useState('');
  const [testing, setTesting] = useState(false);
  const [result, setResult] = useState<DescriptorTestResponse | null>(null);
  const [testError, setTestError] = useState<string | null>(null);

  const selectedEndpoint = descriptor.endpoints.find(
    (ep) => ep.id === selectedEndpointId,
  );

  // ─── Param values state ────────────────────────────────────
  const [paramValues, setParamValues] = useState<Record<string, string>>({});

  const handleParamChange = useCallback(
    (name: string, value: string) => {
      setParamValues((prev) => ({ ...prev, [name]: value }));
    },
    [],
  );

  // Reset params when endpoint changes
  const handleEndpointChange = useCallback((endpointId: string) => {
    setSelectedEndpointId(endpointId);
    setParamValues({});
    setResult(null);
    setTestError(null);
  }, []);

  // ─── Run test ──────────────────────────────────────────────
  const handleTest = useCallback(async () => {
    if (!selectedEndpoint || !baseUrl) return;

    setTesting(true);
    setTestError(null);
    setResult(null);

    const request: DescriptorTestRequest = {
      vendor: descriptor.vendor,
      endpointId: selectedEndpoint.id,
      baseUrl,
      credentials: {
        username: username || undefined,
        password: password || undefined,
        token: token || undefined,
      },
      params: paramValues,
    };

    try {
      const response = await descriptorsApi.test(request);
      setResult(response);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : 'Test request failed';
      setTestError(message);
    } finally {
      setTesting(false);
    }
  }, [
    selectedEndpoint,
    baseUrl,
    username,
    password,
    token,
    descriptor.vendor,
    paramValues,
  ]);

  // ─── Endpoint options ──────────────────────────────────────
  const endpointOptions = descriptor.endpoints.map((ep) => ({
    value: ep.id,
    label: `${ep.method} ${ep.path}${ep.name ? ` — ${ep.name}` : ''}`,
  }));

  return (
    <div className="space-y-4">
      {error && (
        <Alert variant="error" onClose={() => useDescriptorStore.getState().clearError()}>
          {error}
        </Alert>
      )}

      {/* ── Connection Settings ──────────────────────────── */}
      <section className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 space-y-4">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">
          {t('descriptors.testConnection')}
        </h3>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="md:col-span-2">
            <label
              htmlFor="baseUrl"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.deviceUrl')} <span className="text-red-500">*</span>
            </label>
            <Input
              id="baseUrl"
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              placeholder="http://192.168.1.100"
              required
              aria-required="true"
            />
          </div>

          <div>
            <label
              htmlFor="testUsername"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.username')}
            </label>
            <Input
              id="testUsername"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('descriptors.usernamePlaceholder')}
              autoComplete="off"
            />
          </div>

          <div>
            <label
              htmlFor="testPassword"
              className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
            >
              {t('descriptors.password')}
            </label>
            <Input
              id="testPassword"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              autoComplete="off"
            />
          </div>

          {descriptor.auth?.type === 'bearer' && (
            <div className="md:col-span-2">
              <label
                htmlFor="testToken"
                className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
              >
                {t('descriptors.token')}
              </label>
              <Input
                id="testToken"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={t('descriptors.tokenPlaceholder')}
                autoComplete="off"
              />
            </div>
          )}
        </div>
      </section>

      {/* ── Endpoint Selection ────────────────────────────── */}
      <section className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 space-y-4">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">
          {t('descriptors.testEndpoint')}
        </h3>

        <div>
          <label
            htmlFor="testEndpoint"
            className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1"
          >
            {t('descriptors.selectEndpoint')}
          </label>
          <Select
            id="testEndpoint"
            value={selectedEndpointId}
            onChange={(e) => handleEndpointChange(e.target.value)}
            options={[
              { value: '', label: t('descriptors.selectEndpointPlaceholder') },
              ...endpointOptions,
            ]}
          />
        </div>

        {/* Dynamic params for selected endpoint */}
        {selectedEndpoint?.queryParams && selectedEndpoint.queryParams.length > 0 && (
          <div className="space-y-2">
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
              {t('descriptors.endpointParams')}
            </span>
            {selectedEndpoint.queryParams.map((param) => (
              <div key={param.name}>
                <label
                  htmlFor={`param-${param.name}`}
                  className="block text-xs text-slate-400 mb-0.5"
                >
                  {param.name}
                  {param.required && <span className="text-red-500 ml-0.5">*</span>}
                  {param.description && (
                    <span className="ml-1 text-slate-400 italic">
                      — {param.description}
                    </span>
                  )}
                </label>
                <Input
                  id={`param-${param.name}`}
                  value={paramValues[param.name] || ''}
                  onChange={(e) => handleParamChange(param.name, e.target.value)}
                  placeholder={typeof param.default === 'string' ? param.default : ''}
                />
              </div>
            ))}
          </div>
        )}

        <Button
          variant="primary"
          onClick={handleTest}
          disabled={testing || !baseUrl || !selectedEndpointId}
        >
          {testing ? (
            <>
              <Loader2 className="w-4 h-4 mr-1 animate-spin" />
              {t('descriptors.testing')}
            </>
          ) : (
            <>
              <Play className="w-4 h-4 mr-1" />
              {t('descriptors.runTest')}
            </>
          )}
        </Button>
      </section>

      {/* ── Results ────────────────────────────────────────── */}
      {testError && (
        <Alert variant="error">{testError}</Alert>
      )}

      {result && (
        <section className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4 space-y-3">
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">
            {t('descriptors.testResult')}
          </h3>

          {/* Status Summary */}
          <div className="flex items-center gap-4 text-sm">
            <span className="flex items-center gap-1.5">
              {result.success ? (
                <CheckCircle className="w-4 h-4 text-emerald-500" />
              ) : (
                <XCircle className="w-4 h-4 text-red-500" />
              )}
              <span className={result.success ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'}>
                {result.success ? t('descriptors.success') : t('descriptors.failed')}
              </span>
            </span>
            {result.statusCode && (
              <span className="flex items-center gap-1 text-slate-500">
                <span className="font-mono text-xs">{result.statusCode}</span>
              </span>
            )}
            {result.durationMs !== undefined && (
              <span className="flex items-center gap-1 text-slate-500">
                <Clock className="w-3.5 h-3.5" />
                <span>{result.durationMs}ms</span>
              </span>
            )}
          </div>

          {/* Error message */}
          {result.error && (
            <Alert variant="warning">{String(result.error)}</Alert>
          )}

          {/* Response Body */}
          {result.parsedResult && (
            <div>
              <span className="text-xs font-medium text-slate-500 dark:text-slate-400 mb-1 block">
                {t('descriptors.parsedResult')}
              </span>
              <pre className="p-3 bg-slate-50 dark:bg-slate-900 rounded text-xs font-mono overflow-auto max-h-40">
                {JSON.stringify(result.parsedResult, null, 2)}
              </pre>
            </div>
          )}

          {result.body && (
            <div>
              <span className="text-xs font-medium text-slate-500 dark:text-slate-400 mb-1 block">
                {t('descriptors.rawResponse')}
              </span>
              <pre className="p-3 bg-slate-50 dark:bg-slate-900 rounded text-xs font-mono overflow-auto max-h-60">
                {JSON.stringify(result.body, null, 2)}
              </pre>
            </div>
          )}
        </section>
      )}
    </div>
  );
}
