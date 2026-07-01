// ═══════════════════════════════════════════════════════════════════════
// SecureTunnel.tsx — Secure Tunnel Tab for Remote Troubleshooting (UX-2.4)
//
// Предоставляет одноразовый SSH/HTTPS tunnel к устройству через WebSocket.
//   - One-time token с TTL 1h
//   - Кнопка "Copy to clipboard" для tunnel URL
//   - QR code для мобильного доступа
//   - Audit log каждого подключения
//   - Auto-disconnect через 30 минут неактивности
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V3.3: One-time token generation
//   - ISO 27001 A.12.4: Audit trail (через tunnelApi.getLog)
//   - Приказ ОАЦ №66 п.7.18.2: mTLS 1.3 для всех соединений
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { tunnelApi } from '../../services/api/tunnel';
import type {
  TunnelTokenResponse,
  TunnelStatus,
  TunnelLogEntry,
  TunnelProtocol,
} from '../../services/api/tunnel';
import {
  Card,
  CardHeader,
  CardBody,
  Button,
  Badge,
  Modal,
} from '../ui';
import { QRCode } from '../ui/QRCode';
import {
  Shield,
  ShieldCheck,
  ShieldAlert,
  Copy,
  Smartphone,
  Clock,
  History,
  RefreshCw,
  PowerOff,
  Check,
  Code,
  Globe,
  ExternalLink,
  AlertTriangle,
  QrCode,
} from '../ui/Icons';

// ─── Constants ──────────────────────────────────────────────────────

/** Максимальное время неактивности до auto-disconnect (сек) */
const MAX_IDLE_SECONDS = 30 * 60; // 30 минут

/** TTL по умолчанию (сек) */
const DEFAULT_TTL = 60 * 60; // 1 час

/** Протоколы для выбора */
const PROTOCOLS: { value: TunnelProtocol; label: string; icon: React.ReactNode }[] = [
  { value: 'ssh', label: 'SSH', icon: <Code className="w-4 h-4" /> },
  { value: 'https', label: 'HTTPS Proxy', icon: <Globe className="w-4 h-4" /> },
];

// ─── Helpers ────────────────────────────────────────────────────────

function formatExpiry(isoString: string): string {
  const date = new Date(isoString);
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatRemaining(isoString: string): string {
  const remaining = new Date(isoString).getTime() - Date.now();
  if (remaining <= 0) return 'Expired';
  const minutes = Math.floor(remaining / 60000);
  const seconds = Math.floor((remaining % 60000) / 1000);
  return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

function formatIdle(seconds: number): string {
  const minutes = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${minutes}m ${secs}s`;
}

// ─── Props ──────────────────────────────────────────────────────────

export interface SecureTunnelProps {
  deviceId: string;
}

// ─── Sub-components ─────────────────────────────────────────────────

/** Protocol selector tabs */
function ProtocolSelector({
  value,
  onChange,
  disabled,
}: {
  value: TunnelProtocol;
  onChange: (p: TunnelProtocol) => void;
  disabled: boolean;
}) {
  return (
    <div className="flex gap-1 p-0.5 bg-slate-100 dark:bg-slate-800 rounded-lg w-fit">
      {PROTOCOLS.map((p) => (
        <button
          key={p.value}
          onClick={() => onChange(p.value)}
          disabled={disabled}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md transition-colors ${
            value === p.value
              ? 'bg-white dark:bg-slate-700 text-slate-900 dark:text-white shadow-sm'
              : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'
          } ${disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}`}
        >
          {p.icon}
          {p.label}
        </button>
      ))}
    </div>
  );
}

/** Countdown timer */
function CountdownTimer({ expiresAt }: { expiresAt: string }) {
  const [display, setDisplay] = useState(formatRemaining(expiresAt));
  const isExpired = display === 'Expired';

  useEffect(() => {
    const interval = setInterval(() => {
      setDisplay(formatRemaining(expiresAt));
    }, 1000);
    return () => clearInterval(interval);
  }, [expiresAt]);

  return (
    <div className={`flex items-center gap-1.5 text-sm font-mono ${
      isExpired ? 'text-red-500' : 'text-slate-600 dark:text-slate-400'
    }`}>
      <Clock className="w-4 h-4" />
      <span>{isExpired ? 'Expired' : display}</span>
    </div>
  );
}

/** Idle timer bar */
function IdleTimerBar({ idleSeconds, maxIdleSeconds }: { idleSeconds: number; maxIdleSeconds: number }) {
  const pct = Math.min((idleSeconds / maxIdleSeconds) * 100, 100);
  const isWarning = pct > 75;
  const isCritical = pct > 90;

  return (
    <div className="space-y-1">
      <div className="flex justify-between text-xs text-slate-500 dark:text-slate-400">
        <span>Idle time: {formatIdle(idleSeconds)}</span>
        <span className={isCritical ? 'text-red-500' : isWarning ? 'text-amber-500' : ''}>
          Auto-disconnect in {formatIdle(maxIdleSeconds - idleSeconds)}
        </span>
      </div>
      <div className="w-full h-1.5 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-1000 ${
            isCritical ? 'bg-red-500' : isWarning ? 'bg-amber-500' : 'bg-emerald-500'
          }`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}

// ─── Main Component ─────────────────────────────────────────────────

export function SecureTunnel({ deviceId }: SecureTunnelProps) {
  const { t } = useTranslation();

  // State
  const [protocol, setProtocol] = useState<TunnelProtocol>('ssh');
  const [generating, setGenerating] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);
  const [tunnelToken, setTunnelToken] = useState<TunnelTokenResponse | null>(null);
  const [tunnelStatus, setTunnelStatus] = useState<TunnelStatus | null>(null);
  const [logEntries, setLogEntries] = useState<TunnelLogEntry[]>([]);
  const [showQR, setShowQR] = useState(false);
  const [copied, setCopied] = useState(false);
  const [loadingLog, setLoadingLog] = useState(false);

  // Refs
  const statusPollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const logPollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // ── Generate Tunnel ──────────────────────────────────────────────

  const handleGenerate = useCallback(async () => {
    setGenerating(true);
    setError(null);
    setSuccessMsg(null);

    try {
      const token = await tunnelApi.createToken(deviceId, protocol);
      setTunnelToken(token);

      // Начинаем polling статуса
      const status = await tunnelApi.getStatus(deviceId);
      setTunnelStatus(status);

      setSuccessMsg(`Tunnel created — ${protocol.toUpperCase()} proxy active`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create tunnel');
    } finally {
      setGenerating(false);
    }
  }, [deviceId, protocol]);

  // ── Revoke Tunnel ────────────────────────────────────────────────

  const handleRevoke = useCallback(async () => {
    setRevoking(true);
    setError(null);

    try {
      await tunnelApi.revoke(deviceId);
      setTunnelToken(null);
      setTunnelStatus(null);
      setLogEntries([]);
      setSuccessMsg('Tunnel session revoked');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to revoke tunnel');
    } finally {
      setRevoking(false);
    }
  }, [deviceId]);

  // ── Copy to clipboard ────────────────────────────────────────────

  const handleCopy = useCallback(async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback для HTTP
      const textarea = document.createElement('textarea');
      textarea.value = text;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('copy');
      document.body.removeChild(textarea);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }, []);

  // ── Polling статуса и лога ───────────────────────────────────────

  useEffect(() => {
    if (!tunnelToken) {
      if (statusPollRef.current) clearInterval(statusPollRef.current);
      if (logPollRef.current) clearInterval(logPollRef.current);
      return;
    }

    // Poll status every 5 seconds
    statusPollRef.current = setInterval(async () => {
      try {
        const status = await tunnelApi.getStatus(deviceId);
        setTunnelStatus(status);
        if (status.status === 'expired' || status.status === 'revoked') {
          setTunnelToken(null);
          setTunnelStatus(null);
        }
      } catch {
        // Ignore poll errors
      }
    }, 5000);

    // Poll audit log every 10 seconds
    logPollRef.current = setInterval(async () => {
      try {
        const log = await tunnelApi.getLog(deviceId);
        setLogEntries(log);
      } catch {
        // Ignore poll errors
      }
    }, 10000);

    // Initial log load
    tunnelApi.getLog(deviceId).then(setLogEntries).catch(() => {});

    return () => {
      if (statusPollRef.current) clearInterval(statusPollRef.current);
      if (logPollRef.current) clearInterval(logPollRef.current);
    };
  }, [tunnelToken, deviceId]);

  // ── Refresh log manually ─────────────────────────────────────────

  const handleRefreshLog = useCallback(async () => {
    setLoadingLog(true);
    try {
      const log = await tunnelApi.getLog(deviceId);
      setLogEntries(log);
    } catch {
      // ignore
    } finally {
      setLoadingLog(false);
    }
  }, [deviceId]);

  // ── Render: No Active Tunnel ─────────────────────────────────────

  if (!tunnelToken) {
    return (
      <Card>
        <CardHeader>{t('secure_tunnel') || 'Secure Tunnel'}</CardHeader>
        <CardBody>
          {error && (
            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            </div>
          )}

          {successMsg && (
            <div className="mb-4 p-3 bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800 rounded-lg flex items-start gap-2">
              <Check className="w-4 h-4 text-emerald-500 mt-0.5 flex-shrink-0" />
              <p className="text-sm text-emerald-600 dark:text-emerald-400">{successMsg}</p>
            </div>
          )}

          {/* Protocol selector */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
              {t('tunnel_protocol') || 'Tunnel Protocol'}
            </label>
            <ProtocolSelector
              value={protocol}
              onChange={setProtocol}
              disabled={generating}
            />
          </div>

          {/* Description */}
          <div className="mb-6 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
            <div className="flex items-center gap-2 mb-2">
              <Shield className="w-5 h-5 text-blue-500" />
              <span className="text-sm font-medium text-slate-900 dark:text-white">
                {protocol === 'ssh'
                  ? (t('ssh_tunnel_desc') || 'SSH tunnel for remote CLI access')
                  : (t('https_proxy_desc') || 'HTTPS proxy for web interface access')}
              </span>
            </div>
            <ul className="space-y-1 text-xs text-slate-500 dark:text-slate-400 ml-7 list-disc">
              <li>{t('tunnel_feature_1') || 'One-time token with 1 hour TTL'}</li>
              <li>{t('tunnel_feature_2') || 'Auto-disconnect after 30 minutes of inactivity'}</li>
              <li>{t('tunnel_feature_3') || 'Full audit trail of all connections'}</li>
              <li>{t('tunnel_feature_4') || 'mTLS 1.3 encrypted tunnel'}</li>
            </ul>
          </div>

          {/* Generate button */}
          <Button
            onClick={handleGenerate}
            loading={generating}
            icon={<Shield className="w-4 h-4" />}
            size="lg"
          >
            {generating ? (t('generating_tunnel') || 'Generating...') : (t('generate_tunnel') || 'Generate Tunnel')}
          </Button>
        </CardBody>
      </Card>
    );
  }

  // ── Render: Active Tunnel ────────────────────────────────────────

  const isActive = tunnelStatus?.status === 'active';

  return (
    <div className="space-y-6">
      {/* Tunnel Status Card */}
      <Card>
        <CardHeader
          action={
            <Button
              variant="danger"
              size="sm"
              onClick={handleRevoke}
              loading={revoking}
              icon={<PowerOff className="w-4 h-4" />}
            >
              {t('revoke_tunnel') || 'Revoke'}
            </Button>
          }
        >
          <div className="flex items-center gap-2">
            {isActive ? (
              <ShieldCheck className="w-5 h-5 text-emerald-500" />
            ) : (
              <ShieldAlert className="w-5 h-5 text-amber-500" />
            )}
            <span>{t('active_tunnel') || 'Active Tunnel'}</span>
            <Badge variant={isActive ? 'success' : 'warning'} size="sm">
              {isActive ? (t('active') || 'Active') : (t('inactive') || 'Inactive')}
            </Badge>
          </div>
        </CardHeader>
        <CardBody>
          {error && (
            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg flex items-start gap-2">
              <AlertTriangle className="w-4 h-4 text-red-500 mt-0.5 flex-shrink-0" />
              <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
            </div>
          )}

          <div className="space-y-5">
            {/* Tunnel URL */}
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1.5 uppercase tracking-wider">
                {t('tunnel_url') || 'Tunnel URL'}
              </label>
              <div className="flex items-center gap-2">
                <div className="flex-1 p-2.5 bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg font-mono text-sm text-slate-900 dark:text-white overflow-x-auto whitespace-nowrap">
                  {tunnelToken.tunnel_url}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  icon={copied ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
                  onClick={() => handleCopy(tunnelToken.tunnel_url)}
                >
                  {copied ? (t('copied') || 'Copied!') : (t('copy') || 'Copy')}
                </Button>
              </div>
            </div>

            {/* Token */}
            <div>
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1.5 uppercase tracking-wider">
                {t('auth_token') || 'Auth Token'}
              </label>
              <div className="flex items-center gap-2">
                <div className="flex-1 p-2.5 bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg font-mono text-sm text-slate-900 dark:text-white">
                  {tunnelToken.token.slice(0, 16)}...
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  icon={copied ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
                  onClick={() => handleCopy(tunnelToken.token)}
                >
                  {t('copy_token') || 'Token'}
                </Button>
              </div>
            </div>

            {/* Info row */}
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
              <div className="p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg">
                <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 mb-1">
                  <Code className="w-3.5 h-3.5" />
                  <span>{t('protocol') || 'Protocol'}</span>
                </div>
                <p className="text-sm font-semibold text-slate-900 dark:text-white uppercase">
                  {tunnelToken.protocol}
                </p>
              </div>

              <div className="p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg">
                <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 mb-1">
                  <Clock className="w-3.5 h-3.5" />
                  <span>{t('expires') || 'Expires'}</span>
                </div>
                <p className="text-sm font-semibold text-slate-900 dark:text-white">
                  {formatExpiry(tunnelToken.expires_at)}
                </p>
              </div>

              <div className="p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg">
                <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 mb-1">
                  <History className="w-3.5 h-3.5" />
                  <span>{t('connections') || 'Connections'}</span>
                </div>
                <p className="text-sm font-semibold text-slate-900 dark:text-white">
                  {tunnelStatus?.connection_count ?? 0}
                </p>
              </div>

              <div className="p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg">
                <div className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400 mb-1">
                  <Smartphone className="w-3.5 h-3.5" />
                  <span>{t('mobile_access') || 'Mobile Access'}</span>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  icon={<QrCode className="w-4 h-4" />}
                  onClick={() => setShowQR(true)}
                  className="!px-2 !py-1 text-xs"
                >
                  QR
                </Button>
              </div>
            </div>

            {/* Countdown + Idle Timer */}
            <div className="space-y-3 p-4 bg-slate-50 dark:bg-slate-800/30 rounded-lg border border-slate-200 dark:border-slate-700">
              <div className="flex items-center justify-between">
                <CountdownTimer expiresAt={tunnelToken.expires_at} />
                {tunnelStatus && (
                  <span className="text-xs text-slate-500 dark:text-slate-400">
                    TTL: {Math.floor(tunnelToken.ttl_seconds / 60)} min
                  </span>
                )}
              </div>
              {tunnelStatus && (
                <IdleTimerBar
                  idleSeconds={tunnelStatus.idle_seconds ?? 0}
                  maxIdleSeconds={tunnelStatus.max_idle_seconds ?? MAX_IDLE_SECONDS}
                />
              )}
            </div>

            {/* Action buttons */}
            <div className="flex items-center gap-3 pt-2">
              <Button
                variant="outline"
                size="sm"
                icon={<ExternalLink className="w-4 h-4" />}
                onClick={() => window.open(tunnelToken.tunnel_url, '_blank')}
              >
                {t('open_tunnel') || 'Open Tunnel'}
              </Button>
            </div>
          </div>
        </CardBody>
      </Card>

      {/* Audit Log Card */}
      <Card>
        <CardHeader
          action={
            <Button
              variant="ghost"
              size="sm"
              icon={<RefreshCw className={`w-4 h-4 ${loadingLog ? 'animate-spin' : ''}`} />}
              onClick={handleRefreshLog}
            >
              {t('refresh') || 'Refresh'}
            </Button>
          }
        >
          <div className="flex items-center gap-2">
            <History className="w-5 h-5 text-slate-500" />
            <span>{t('connection_log') || 'Connection Log'}</span>
          </div>
        </CardHeader>
        <CardBody>
          {logEntries.length === 0 ? (
            <div className="text-center py-6">
              <History className="w-8 h-8 text-slate-300 dark:text-slate-600 mx-auto mb-2" />
              <p className="text-sm text-slate-500 dark:text-slate-400">
                {t('no_connections') || 'No connections yet'}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {logEntries.map((entry) => (
                <div
                  key={entry.id}
                  className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg border border-slate-200 dark:border-slate-700/50 text-sm"
                >
                  <div className={`mt-0.5 w-2 h-2 rounded-full flex-shrink-0 ${
                    entry.action === 'connected' ? 'bg-emerald-500'
                      : entry.action === 'disconnected' ? 'bg-slate-400'
                        : entry.action === 'created' ? 'bg-blue-500'
                          : 'bg-amber-500'
                  }`} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium text-slate-900 dark:text-white capitalize">
                        {entry.action}
                      </span>
                      <span className="text-xs text-slate-400 font-mono">
                        {new Date(entry.created_at).toLocaleTimeString()}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 mt-0.5 text-xs text-slate-500 dark:text-slate-400">
                      <span className="uppercase font-mono">{entry.protocol}</span>
                      <span>•</span>
                      <span>{entry.remote_ip}</span>
                      {entry.user_agent && (
                        <>
                          <span>•</span>
                          <span className="truncate max-w-[200px]">{entry.user_agent}</span>
                        </>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardBody>
      </Card>

      {/* QR Code Modal */}
      <Modal
        isOpen={showQR}
        onClose={() => setShowQR(false)}
        title={t('mobile_access_qr') || 'Mobile Access QR Code'}
        size="sm"
      >
        <div className="flex flex-col items-center p-4">
          <QRCode
            value={tunnelToken.tunnel_url}
            size={220}
            label={`${protocol.toUpperCase()} Tunnel`}
          />
          <p className="mt-4 text-sm text-slate-500 dark:text-slate-400 text-center">
            {t('qr_scan_instructions') || 'Scan with your mobile device to access the tunnel'}
          </p>
          <Button
            variant="outline"
            size="sm"
            icon={<Copy className="w-4 h-4" />}
            onClick={() => handleCopy(tunnelToken.tunnel_url)}
            className="mt-3"
          >
            {t('copy_url') || 'Copy URL'}
          </Button>
        </div>
      </Modal>
    </div>
  );
}

export default SecureTunnel;
