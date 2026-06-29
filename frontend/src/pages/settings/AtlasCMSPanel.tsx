import React, { useState, useEffect } from 'react';
import { Globe, Loader2, RefreshCw, AlertCircle, Info, Link } from '../components/ui/Icons';
import { Button, useToast } from '../../components/ui';
import { useTranslation } from 'react-i18next';

export const AtlasCMSPanel: React.FC = () => {
  const { t } = useTranslation();
  const toast = useToast();
  const [healthStatus, setHealthStatus] = useState<string | null>(null);
  const [healthError, setHealthError] = useState<string | null>(null);
  const [queueSize, setQueueSize] = useState<number>(0);
  const [retrying, setRetrying] = useState(false);
  const [checking, setChecking] = useState(false);

  const checkHealth = async () => {
    setChecking(true);
    setHealthError(null);
    try {
      const { api } = await import('../../services/api');
      const result = await api.atlasHealthCheck();
      setHealthStatus(result.status);
      if (result.error) setHealthError(result.error);
      if (result.message) toast.info(result.message);
    } catch (e: any) {
      setHealthStatus('error');
      setHealthError(e.message);
    } finally {
      setChecking(false);
    }
  };

  const checkFallback = async () => {
    try {
      const { api } = await import('../../services/api');
      const result = await api.atlasFallbackStatus();
      setQueueSize(result.queue_size);
      if (result.message) toast.info(result.message);
    } catch (e: any) {
      toast.error(e.message);
    }
  };

  const retryFallback = async () => {
    setRetrying(true);
    try {
      const { api } = await import('../../services/api');
      const result = await api.atlasRetryFallback();
      toast.success(`Retried: ${result.success} succeeded, ${result.failed} failed`);
      await checkFallback();
    } catch (e: any) {
      toast.error(e.message);
    } finally {
      setRetrying(false);
    }
  };

  useEffect(() => {
    checkHealth();
    checkFallback();
  }, []);

  const statusColor = healthStatus === 'healthy'
    ? 'bg-emerald-500'
    : healthStatus === 'unhealthy' || healthStatus === 'error'
    ? 'bg-red-500'
    : healthStatus === 'not_configured'
    ? 'bg-amber-500'
    : 'bg-slate-300';

  const statusLabel = healthStatus === 'healthy'
    ? 'Connected'
    : healthStatus === 'unhealthy'
    ? 'Unhealthy'
    : healthStatus === 'not_configured'
    ? 'Not Configured'
    : healthStatus === 'error'
    ? 'Error'
    : 'Unknown';

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between p-4 bg-indigo-50/50 dark:bg-indigo-900/10 rounded-lg border border-indigo-100 dark:border-indigo-800/30">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-white dark:bg-slate-800 rounded-lg shadow-sm">
            <Globe className="w-5 h-5 text-indigo-600 dark:text-indigo-400" />
          </div>
          <div>
            <p className="font-medium text-slate-900 dark:text-white">Atlas CMMS Connection</p>
            <div className="flex items-center gap-2 mt-1">
              <span className={`w-2 h-2 rounded-full ${statusColor}`} />
              <span className="text-sm text-slate-500 dark:text-slate-400">{statusLabel}</span>
            </div>
            {healthError && (
              <p className="text-xs text-red-500 dark:text-red-400 mt-1">{healthError}</p>
            )}
          </div>
        </div>
        <Button
          variant="outline"
          size="sm"
          icon={checking ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
          onClick={checkHealth}
          disabled={checking}
        >
          {checking ? 'Checking...' : 'Check'}
        </Button>
      </div>

      <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-amber-600 dark:text-amber-400" />
            <h4 className="font-medium text-slate-900 dark:text-white">Fallback Queue</h4>
          </div>
          <Button variant="outline" size="sm" onClick={checkFallback}>Refresh</Button>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex-1">
            <p className="text-3xl font-bold text-slate-900 dark:text-white">{queueSize}</p>
            <p className="text-xs text-slate-500 dark:text-slate-400">Pending operations</p>
          </div>
          <Button
            icon={retrying ? <Loader2 className="w-4 h-4 animate-spin" /> : <RefreshCw className="w-4 h-4" />}
            onClick={retryFallback}
            disabled={retrying || queueSize === 0}
          >
            {retrying ? 'Retrying...' : 'Retry All'}
          </Button>
        </div>
        <p className="text-xs text-slate-400 dark:text-slate-500 mt-2">
          Operations that failed to sync with Atlas CMMS are stored here and retried automatically.
        </p>
      </div>

      <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
        <div className="flex items-center gap-2 mb-2">
          <Info className="w-4 h-4 text-blue-600 dark:text-blue-400" />
          <h4 className="font-medium text-slate-900 dark:text-white">Configuration</h4>
        </div>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          Atlas CMMS integration is configured via environment variables:
        </p>
        <div className="mt-2 space-y-1 text-xs font-mono text-slate-600 dark:text-slate-400">
          <p>GB_CMMS_ADAPTER=atlas</p>
          <p>GB_ATLAS_URL=https://atlas-cmms.example.com</p>
          <p>GB_ATLAS_CLIENT_ID=••••</p>
          <p>GB_ATLAS_CLIENT_SECRET=••••</p>
          <p>GB_ATLAS_TOKEN_URL=https://atlas-cmms.example.com/oauth/token</p>
          <p>GB_ATLAS_FALLBACK_DIR=/var/lib/gb-telemetry/fallback</p>
        </div>
      </div>

      <div className="p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
        <div className="flex items-center gap-2 mb-2">
          <Link className="w-4 h-4 text-emerald-600 dark:text-emerald-400" />
          <h4 className="font-medium text-slate-900 dark:text-white">Device Sync</h4>
        </div>
        <p className="text-sm text-slate-500 dark:text-slate-400 mb-3">
          Sync individual devices as assets to Atlas CMMS. Use the sync button on the device detail page.
        </p>
      </div>
    </div>
  );
};