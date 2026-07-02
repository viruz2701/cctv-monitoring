// ═══════════════════════════════════════════════════════════════════════
// HashChainSignature.tsx — Hash-Chain Digital Signature Component
//
// UX-3.6: Hash-Chain Digital Signatures (СТБ 34.101.27)
//   - Подпись включает hash предыдущего ТО
//   - Visual chain: "Previous TO: TO-200 · Hash: a3f4...2b1c ✓"
//   - Verifier tool для аудиторов
//   - Fallback на простую подпись (с warning)
//
// Feature Flag: hash_chain_signatures (default: false)
//
// Compliance:
//   - СТБ 34.101.27 п. 6.3 (Hash chain integrity)
//   - СТБ 34.101.30 (bash-256/belt-gcm)
//   - ISO 27001 A.12.4 (Audit trail)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { isFeatureEnabled } from '../../config/featureFlags';
import { signatureApi } from '../../services/signatureApi';
import type { SignatureChain, VerificationResult } from '../../services/signatureApi';
import { ChainVerifier } from './ChainVerifier';
import {
  Fingerprint, CheckCircle, XCircle, AlertTriangle, Loader2,
  ChevronDown, ChevronUp, Shield, Download, ExternalLink,
} from 'lucide-react';

// ── Helpers ───────────────────────────────────────────────────────────

function truncateHash(hash: string, len = 8): string {
  if (hash.length <= len * 2 + 3) return hash;
  return `${hash.slice(0, len)}...${hash.slice(-len)}`;
}

function formatDateTime(dateStr: string): string {
  try {
    return new Intl.DateTimeFormat('ru-RU', {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date(dateStr));
  } catch {
    return dateStr;
  }
}

// ── Props ─────────────────────────────────────────────────────────────

interface HashChainSignatureProps {
  /** Work Order ID */
  workOrderId: string;
  /** Предыдущий номер ТО (TO-200) */
  previousDocumentLabel?: string;
  /** Предыдущий хеш */
  previousHash?: string;
  /** Кем подписывается */
  signedBy: string;
  /** Отпечаток сертификата */
  certificateFingerprint?: string;
  /** Колбэк успешной подписи */
  onSigned?: (chain: SignatureChain) => void;
  /** Колбэк ошибки */
  onError?: (error: string) => void;
  /** Компактный режим */
  compact?: boolean;
}

// ── HashChainSignature Component ──────────────────────────────────────

export function HashChainSignature({
  workOrderId,
  previousDocumentLabel,
  previousHash,
  signedBy,
  certificateFingerprint = 'bign-cc-0001',
  onSigned,
  onError,
  compact = false,
}: HashChainSignatureProps) {
  const { t } = useTranslation();
  const [chain, setChain] = useState<SignatureChain | null>(null);
  const [loading, setLoading] = useState(true);
  const [signing, setSigning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(!compact);
  const [showVerifier, setShowVerifier] = useState(false);
  const [verificationResult, setVerificationResult] = useState<VerificationResult | null>(null);
  const [useFallback, setUseFallback] = useState(false);

  const isEnabled = isFeatureEnabled('hash_chain_signatures');

  // ── Load existing chain ─────────────────────────────────────────
  const loadChain = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const existingChain = await signatureApi.getChain(workOrderId);
      setChain(existingChain);
    } catch {
      // No chain exists yet — that's OK
      setChain(null);
    } finally {
      setLoading(false);
    }
  }, [workOrderId]);

  useEffect(() => {
    if (isEnabled) {
      loadChain();
    }
  }, [isEnabled, loadChain]);

  // ── Sign ────────────────────────────────────────────────────────
  const handleSign = useCallback(async () => {
    if (useFallback) {
      await handleFallbackSign();
      return;
    }

    setSigning(true);
    setError(null);

    try {
      // Generate a simulated bash-256 hash (in production, this comes from backend)
      const currentHash = `bash256_${Date.now().toString(16)}_${workOrderId.slice(0, 8)}`;

      const result = await signatureApi.signDocument({
        work_order_id: workOrderId,
        document_type: 'work_order',
        previous_hash: previousHash || chain?.current_hash || undefined,
        previous_document_id: chain?.work_order_id || undefined,
        signature: currentHash,
        algorithm: 'bash-256',
        certificate_fingerprint: certificateFingerprint,
        signed_by: signedBy,
      });

      setChain(result);
      onSigned?.(result);

      // Auto-verify after signing
      const verification = await signatureApi.verifySignature(result.id);
      setVerificationResult(verification);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Signing failed';
      setError(msg);
      onError?.(msg);
    } finally {
      setSigning(false);
    }
  }, [workOrderId, previousHash, chain, certificateFingerprint, signedBy, useFallback, onSigned, onError]);

  // ── Fallback sign ───────────────────────────────────────────────
  const handleFallbackSign = useCallback(async () => {
    setSigning(true);
    setError(null);

    try {
      const result = await signatureApi.signFallback({
        work_order_id: workOrderId,
        document_type: 'work_order',
        signature: `fallback_${Date.now().toString(16)}`,
        signed_by: signedBy,
        reason: 'Hash chain unavailable — fallback signature used',
      });

      setChain(result);
      onSigned?.(result);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Fallback signing failed';
      setError(msg);
      onError?.(msg);
    } finally {
      setSigning(false);
    }
  }, [workOrderId, signedBy, onSigned, onError]);

  // ── Verify ──────────────────────────────────────────────────────
  const handleVerify = useCallback(async () => {
    if (!chain) return;

    try {
      const result = await signatureApi.verifySignature(chain.id);
      setVerificationResult(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    }
  }, [chain]);

  // ── Render: Feature Disabled ────────────────────────────────────
  if (!isEnabled) {
    return (
      <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4">
        <div className="flex items-center gap-3">
          <Shield className="w-5 h-5 text-slate-400" aria-hidden="true" />
          <div>
            <p className="text-sm text-slate-600 dark:text-slate-400">
              Hash-chain signatures are disabled
            </p>
            <p className="text-xs text-slate-400 mt-0.5">
              Enable `hash_chain_signatures` feature flag
            </p>
          </div>
        </div>
      </div>
    );
  }

  // ── Render: Loading ─────────────────────────────────────────────
  if (loading) {
    return (
      <div className="flex items-center gap-2 p-4 text-sm text-slate-500">
        <Loader2 className="w-4 h-4 animate-spin" />
        Loading signature chain...
      </div>
    );
  }

  // ── Render: Already signed ──────────────────────────────────────
  if (chain) {
    const isVerified = verificationResult?.overall_valid ?? chain.verified;
    const prevDoc = chain.previous_document_label || previousDocumentLabel;

    return (
      <div className={`rounded-lg border ${
        isVerified
          ? 'border-emerald-200 dark:border-emerald-800'
          : 'border-amber-200 dark:border-amber-800'
      } bg-white dark:bg-slate-800 overflow-hidden`}>
        {/* Header */}
        <button
          onClick={() => setExpanded(!expanded)}
          className="w-full flex items-center justify-between px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors"
          aria-expanded={expanded}
          aria-label={expanded ? 'Collapse signature details' : 'Expand signature details'}
        >
          <div className="flex items-center gap-3">
            {isVerified ? (
              <CheckCircle className="w-5 h-5 text-emerald-500" aria-hidden="true" />
            ) : chain.is_fallback ? (
              <AlertTriangle className="w-5 h-5 text-amber-500" aria-hidden="true" />
            ) : (
              <Fingerprint className="w-5 h-5 text-blue-500" aria-hidden="true" />
            )}
            <div className="text-left">
              <p className="text-sm font-medium text-slate-900 dark:text-white">
                {chain.is_fallback ? 'Fallback Signature' : 'Digital Signature (Hash Chain)'}
              </p>
              <p className="text-xs text-slate-500">
                {chain.algorithm.toUpperCase()} · Signed by {chain.signed_by}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {chain.is_fallback && (
              <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                FALLBACK
              </span>
            )}
            {expanded ? <ChevronUp className="w-4 h-4 text-slate-400" /> : <ChevronDown className="w-4 h-4 text-slate-400" />}
          </div>
        </button>

        {/* Expanded Details */}
        {expanded && (
          <div className="px-4 pb-4 space-y-3 border-t border-slate-100 dark:border-slate-700 pt-3">
            {/* Hash Chain Visualization */}
            <div className="bg-slate-50 dark:bg-slate-900/50 rounded-lg p-3 space-y-2">
              <h4 className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                Hash Chain
              </h4>

              {/* Previous Document */}
              {prevDoc && (
                <div className="flex items-center gap-2 text-sm">
                  <div className="w-2 h-2 rounded-full bg-slate-300 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-slate-600 dark:text-slate-400 font-medium">
                        Previous: {prevDoc}
                      </span>
                      {chain.previous_hash && (
                        <code className="text-[10px] font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-1 py-0.5 rounded">
                          Hash: {truncateHash(chain.previous_hash)}
                        </code>
                      )}
                    </div>
                  </div>
                  {chain.chain_valid && (
                    <CheckCircle className="w-3.5 h-3.5 text-emerald-500 flex-shrink-0" aria-hidden="true" />
                  )}
                </div>
              )}

              {/* Chain Arrow */}
              {prevDoc && (
                <div className="flex justify-center">
                  <svg width="16" height="20" viewBox="0 0 16 20" className="text-slate-300 dark:text-slate-600">
                    <line x1="8" y1="0" x2="8" y2="14" stroke="currentColor" strokeWidth="2" />
                    <polyline points="2,10 8,16 14,10" fill="none" stroke="currentColor" strokeWidth="2" />
                  </svg>
                </div>
              )}

              {/* Current Document */}
              <div className="flex items-center gap-2 text-sm">
                <div className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  isVerified ? 'bg-emerald-500' : 'bg-amber-500'
                }`} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-slate-800 dark:text-slate-200">
                      Current: WO-{workOrderId.slice(0, 8).toUpperCase()}
                    </span>
                    <code className="text-[10px] font-mono text-slate-500 bg-slate-100 dark:bg-slate-800 px-1 py-0.5 rounded">
                      {truncateHash(chain.current_hash)}
                    </code>
                  </div>
                </div>
                {isVerified ? (
                  <CheckCircle className="w-3.5 h-3.5 text-emerald-500 flex-shrink-0" aria-hidden="true" />
                ) : (
                  <AlertTriangle className="w-3.5 h-3.5 text-amber-500 flex-shrink-0" aria-hidden="true" />
                )}
              </div>
            </div>

            {/* Signature Info */}
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <span className="text-slate-500">Signed by:</span>
                <span className="ml-1 text-slate-800 dark:text-slate-200 font-medium">{chain.signed_by}</span>
              </div>
              <div>
                <span className="text-slate-500">Algorithm:</span>
                <span className="ml-1 text-slate-800 dark:text-slate-200">{chain.algorithm.toUpperCase()}</span>
              </div>
              <div>
                <span className="text-slate-500">Certificate:</span>
                <span className="ml-1 text-slate-800 dark:text-slate-200 font-mono text-[10px]">
                  {truncateHash(chain.certificate_fingerprint, 6)}
                </span>
              </div>
              <div>
                <span className="text-slate-500">Date:</span>
                <span className="ml-1 text-slate-800 dark:text-slate-200">{formatDateTime(chain.signed_at)}</span>
              </div>
            </div>

            {/* Verification Status */}
            {verificationResult && (
              <div className={`rounded-lg p-2 ${
                verificationResult.overall_valid
                  ? 'bg-emerald-50 dark:bg-emerald-900/20'
                  : 'bg-red-50 dark:bg-red-900/20'
              }`}>
                <div className="flex items-center gap-2">
                  {verificationResult.overall_valid ? (
                    <CheckCircle className="w-4 h-4 text-emerald-500" />
                  ) : (
                    <XCircle className="w-4 h-4 text-red-500" />
                  )}
                  <span className={`text-xs font-medium ${
                    verificationResult.overall_valid
                      ? 'text-emerald-700 dark:text-emerald-400'
                      : 'text-red-700 dark:text-red-400'
                  }`}>
                    {verificationResult.overall_valid ? 'Signature Verified' : 'Verification Failed'}
                  </span>
                  <span className="text-[10px] text-slate-400 ml-auto">
                    {formatDateTime(verificationResult.verified_at)}
                  </span>
                </div>
                {verificationResult.errors.length > 0 && (
                  <ul className="mt-1 space-y-0.5">
                    {verificationResult.errors.map((err, i) => (
                      <li key={i} className="text-[10px] text-red-600 dark:text-red-400">{err}</li>
                    ))}
                  </ul>
                )}
              </div>
            )}

            {/* Actions */}
            {!compact && (
              <div className="flex items-center gap-2 pt-1">
                <button
                  onClick={handleVerify}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50 transition-colors"
                >
                  <Shield className="w-3 h-3" aria-hidden="true" />
                  Verify
                </button>
                <button
                  onClick={() => setShowVerifier(true)}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 dark:text-slate-400 dark:bg-slate-800 dark:hover:bg-slate-700 transition-colors"
                >
                  <ExternalLink className="w-3 h-3" aria-hidden="true" />
                  Chain Verifier
                </button>
                <button
                  onClick={async () => {
                    try {
                      const result = await signatureApi.exportChain(workOrderId);
                      if (result.csv_url) window.open(result.csv_url, '_blank');
                    } catch {}
                  }}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 dark:text-slate-400 dark:bg-slate-800 dark:hover:bg-slate-700 transition-colors"
                >
                  <Download className="w-3 h-3" aria-hidden="true" />
                  Export
                </button>
              </div>
            )}
          </div>
        )}

        {/* Chain Verifier Modal */}
        {showVerifier && (
          <ChainVerifier
            chainId={chain.id}
            workOrderId={workOrderId}
            onClose={() => setShowVerifier(false)}
          />
        )}
      </div>
    );
  }

  // ── Render: Sign Form ───────────────────────────────────────────
  return (
    <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 p-4 space-y-4">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Fingerprint className="w-5 h-5 text-blue-500" aria-hidden="true" />
        <div>
          <h3 className="text-sm font-medium text-slate-900 dark:text-white">
            Sign Document with Hash Chain
          </h3>
          <p className="text-xs text-slate-500">
            СТБ 34.101.27 · bash-256 hash chain
          </p>
        </div>
      </div>

      {/* Hash Chain Preview */}
      <div className="bg-slate-50 dark:bg-slate-900/50 rounded-lg p-3">
        <div className="flex items-center gap-2 text-sm">
          <div className="w-2 h-2 rounded-full bg-slate-300 flex-shrink-0" />
          <span className="text-xs text-slate-500">
            {previousDocumentLabel
              ? `Previous: ${previousDocumentLabel}`
              : 'First document in chain'}
          </span>
          {previousHash && (
            <code className="text-[10px] font-mono text-slate-400 bg-slate-100 dark:bg-slate-800 px-1 py-0.5 rounded">
              {truncateHash(previousHash)}
            </code>
          )}
        </div>
        <div className="flex justify-center py-1">
          <svg width="12" height="16" viewBox="0 0 12 16" className="text-slate-300 dark:text-slate-600">
            <line x1="6" y1="0" x2="6" y2="10" stroke="currentColor" strokeWidth="2" />
            <polyline points="1,6 6,12 11,6" fill="none" stroke="currentColor" strokeWidth="2" />
          </svg>
        </div>
        <div className="flex items-center gap-2 text-sm">
          <div className="w-2 h-2 rounded-full bg-blue-500 flex-shrink-0" />
          <span className="text-xs font-medium text-slate-700 dark:text-slate-300">
            Current: WO-{workOrderId.slice(0, 8).toUpperCase()}
          </span>
          <span className="text-[10px] text-slate-400">(will be hashed)</span>
        </div>
      </div>

      {/* Fallback Toggle */}
      <label className="flex items-center gap-2 text-xs text-slate-600 dark:text-slate-400 cursor-pointer">
        <input
          type="checkbox"
          checked={useFallback}
          onChange={(e) => setUseFallback(e.target.checked)}
          className="rounded border-slate-300 dark:border-slate-600"
        />
        Use fallback signature (no hash chain)
      </label>

      {useFallback && (
        <div className="flex items-start gap-2 p-2 rounded-lg bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800">
          <AlertTriangle className="w-4 h-4 text-amber-500 flex-shrink-0 mt-0.5" aria-hidden="true" />
          <p className="text-xs text-amber-700 dark:text-amber-400">
            Fallback signatures do not include hash-chain verification.
            This reduces audit trail integrity (СТБ 34.101.27 §6.3).
          </p>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="flex items-center gap-2 p-2 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
          <XCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
          <span className="text-xs text-red-700 dark:text-red-400">{error}</span>
        </div>
      )}

      {/* Sign Button */}
      <div className="flex items-center gap-2">
        <button
          onClick={handleSign}
          disabled={signing}
          className="inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          {signing ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin" />
              Signing...
            </>
          ) : useFallback ? (
            <>
              <AlertTriangle className="w-4 h-4" />
              Sign (Fallback)
            </>
          ) : (
            <>
              <Fingerprint className="w-4 h-4" />
              Sign with Hash Chain
            </>
          )}
        </button>
        <button
          onClick={loadChain}
          disabled={signing}
          className="px-3 py-2 rounded-lg text-sm font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 dark:text-slate-400 dark:bg-slate-700 dark:hover:bg-slate-600 disabled:opacity-50 transition-colors"
        >
          Refresh
        </button>
      </div>
    </div>
  );
}

export default HashChainSignature;
