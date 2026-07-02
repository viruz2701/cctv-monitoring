// ═══════════════════════════════════════════════════════════════════════
// ChainVerifier.tsx — Verifier tool для аудиторов
//
// UX-3.6: Hash-Chain Digital Signatures (СТБ 34.101.27)
//   - Verifier tool для аудиторов
//   - Визуализация всей цепочки подписей
//   - Детальная проверка каждого звена
//
// Compliance:
//   - СТБ 34.101.27 п. 6.3 (Hash chain verification)
//   - ISO 27001 A.12.4 (Audit trail)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { signatureApi } from '../../services/signatureApi';
import type { SignatureChain, VerificationResult, ChainStatistics } from '../../services/signatureApi';
import {
  Shield, CheckCircle, XCircle, AlertTriangle, Loader2,
  X, Search, Hash, Clock, Users, FileText, Download,
} from 'lucide-react';

// ── Helpers ───────────────────────────────────────────────────────────

function truncateHash(hash: string, len = 12): string {
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

interface ChainVerifierProps {
  chainId: string;
  workOrderId: string;
  onClose: () => void;
}

// ── ChainVerifier Component ───────────────────────────────────────────

export function ChainVerifier({ chainId, workOrderId, onClose }: ChainVerifierProps) {
  const { t } = useTranslation();
  const [chains, setChains] = useState<SignatureChain[]>([]);
  const [loading, setLoading] = useState(true);
  const [verifying, setVerifying] = useState(false);
  const [results, setResults] = useState<Map<string, VerificationResult>>(new Map());
  const [statistics, setStatistics] = useState<ChainStatistics | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedChain, setSelectedChain] = useState<string | null>(null);

  // ── Load all chains ─────────────────────────────────────────────
  const loadAllChains = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const [allChains, stats] = await Promise.all([
        signatureApi.getAllChains({ limit: 50 }),
        signatureApi.getStatistics(workOrderId).catch(() => null),
      ]);
      setChains(allChains);
      setStatistics(stats);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load chains');
    } finally {
      setLoading(false);
    }
  }, [workOrderId]);

  useEffect(() => {
    loadAllChains();
  }, [loadAllChains]);

  // ── Verify specific chain ───────────────────────────────────────
  const handleVerify = useCallback(async (id: string) => {
    setVerifying(true);

    try {
      const result = await signatureApi.verifySignature(id);
      setResults((prev) => new Map(prev).set(id, result));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setVerifying(false);
    }
  }, []);

  // ── Verify all ──────────────────────────────────────────────────
  const handleVerifyAll = useCallback(async () => {
    setVerifying(true);

    for (const chain of chains) {
      try {
        const result = await signatureApi.verifySignature(chain.id);
        setResults((prev) => new Map(prev).set(chain.id, result));
      } catch {
        // Continue with next chain
      }
    }

    setVerifying(false);
  }, [chains]);

  // ── Filter chains ───────────────────────────────────────────────
  const filteredChains = chains.filter((chain) => {
    if (!searchQuery) return true;
    const q = searchQuery.toLowerCase();
    return (
      chain.work_order_id.toLowerCase().includes(q) ||
      chain.signed_by.toLowerCase().includes(q) ||
      chain.certificate_fingerprint.toLowerCase().includes(q) ||
      chain.current_hash.toLowerCase().includes(q)
    );
  });

  // ── Render ──────────────────────────────────────────────────────
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/50 backdrop-blur-sm">
      <div
        className="relative w-full max-w-3xl max-h-[85vh] bg-white dark:bg-slate-800 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden flex flex-col"
        role="dialog"
        aria-modal="true"
        aria-label="Chain Verifier"
      >
        {/* ── Header ────────────────────────────────────────────── */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700">
          <div className="flex items-center gap-3">
            <Shield className="w-6 h-6 text-blue-600 dark:text-blue-400" aria-hidden="true" />
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                Hash Chain Verifier
              </h2>
              <p className="text-xs text-slate-500">
                СТБ 34.101.27 · Audit tool for digital signature chains
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 text-slate-500 hover:text-slate-700 dark:hover:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
            aria-label="Close verifier"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* ── Stats Bar ─────────────────────────────────────────── */}
        {statistics && (
          <div className="grid grid-cols-5 gap-3 px-6 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-700">
            <div className="text-center">
              <div className="text-lg font-bold text-slate-900 dark:text-white tabular-nums">
                {statistics.total_signatures}
              </div>
              <div className="text-[10px] text-slate-500">Total Chains</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-bold text-emerald-600 tabular-nums">
                {statistics.verified_count}
              </div>
              <div className="text-[10px] text-slate-500">Verified</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-bold text-amber-600 tabular-nums">
                {statistics.pending_verification}
              </div>
              <div className="text-[10px] text-slate-500">Pending</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-bold text-red-600 tabular-nums">
                {statistics.broken_links}
              </div>
              <div className="text-[10px] text-slate-500">Broken Links</div>
            </div>
            <div className="text-center">
              <div className="text-lg font-bold text-slate-900 dark:text-white tabular-nums">
                {statistics.average_verification_time_ms}ms
              </div>
              <div className="text-[10px] text-slate-500">Avg Time</div>
            </div>
          </div>
        )}

        {/* ── Search + Actions ──────────────────────────────────── */}
        <div className="flex items-center gap-3 px-6 py-3 border-b border-slate-200 dark:border-slate-700">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" aria-hidden="true" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search by WO ID, signer, hash..."
              className="w-full pl-9 pr-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
              aria-label="Search signature chains"
            />
          </div>
          <button
            onClick={handleVerifyAll}
            disabled={verifying || chains.length === 0}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50 disabled:opacity-50 transition-colors"
          >
            {verifying ? (
              <Loader2 className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Shield className="w-3.5 h-3.5" />
            )}
            Verify All
          </button>
          <button
            onClick={loadAllChains}
            disabled={loading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-600 bg-slate-100 hover:bg-slate-200 dark:text-slate-400 dark:bg-slate-700 dark:hover:bg-slate-600 disabled:opacity-50 transition-colors"
          >
            <Loader2 className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>

        {/* ── Error ─────────────────────────────────────────────── */}
        {error && (
          <div className="mx-6 mt-3 flex items-center gap-2 p-2 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800">
            <XCircle className="w-4 h-4 text-red-500 flex-shrink-0" />
            <span className="text-xs text-red-700 dark:text-red-400">{error}</span>
            <button onClick={() => setError(null)} className="ml-auto text-xs text-red-500 hover:text-red-700">
              Dismiss
            </button>
          </div>
        )}

        {/* ── Chain List ────────────────────────────────────────── */}
        <div className="flex-1 overflow-y-auto p-4">
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-6 h-6 animate-spin text-blue-500" />
            </div>
          ) : filteredChains.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <Hash className="w-10 h-10 text-slate-300 dark:text-slate-600 mb-3" aria-hidden="true" />
              <p className="text-sm text-slate-500">
                {searchQuery ? 'No chains match your search' : 'No signature chains found'}
              </p>
              <p className="text-xs text-slate-400 mt-1">
                Sign a document to create the first chain entry
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {filteredChains.map((chain, index) => {
                const result = results.get(chain.id);
                const isSelected = selectedChain === chain.id;

                return (
                  <div
                    key={chain.id}
                    className={`rounded-lg border transition-all ${
                      isSelected
                        ? 'border-blue-300 dark:border-blue-600 shadow-sm'
                        : 'border-slate-200 dark:border-slate-700 hover:border-slate-300'
                    }`}
                  >
                    {/* Chain Header */}
                    <button
                      onClick={() => setSelectedChain(isSelected ? null : chain.id)}
                      className="w-full flex items-center gap-3 px-4 py-3 text-left"
                    >
                      {/* Chain status icon */}
                      {result ? (
                        result.overall_valid ? (
                          <CheckCircle className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                        ) : (
                          <XCircle className="w-5 h-5 text-red-500 flex-shrink-0" />
                        )
                      ) : chain.verified ? (
                        <CheckCircle className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                      ) : chain.is_fallback ? (
                        <AlertTriangle className="w-5 h-5 text-amber-500 flex-shrink-0" />
                      ) : (
                        <Clock className="w-5 h-5 text-slate-400 flex-shrink-0" />
                      )}

                      {/* Chain info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-sm font-medium text-slate-900 dark:text-white">
                            WO-{chain.work_order_id.slice(0, 8).toUpperCase()}
                          </span>
                          {chain.is_fallback && (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
                              FALLBACK
                            </span>
                          )}
                          {chain.algorithm !== 'bash-256' && (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400">
                              {chain.algorithm.toUpperCase()}
                            </span>
                          )}
                        </div>
                        <div className="flex items-center gap-3 text-xs text-slate-500 mt-0.5">
                          <span className="flex items-center gap-1">
                            <Users className="w-3 h-3" aria-hidden="true" />
                            {chain.signed_by}
                          </span>
                          <span className="flex items-center gap-1">
                            <Clock className="w-3 h-3" aria-hidden="true" />
                            {formatDateTime(chain.signed_at)}
                          </span>
                          <code className="font-mono text-[10px] text-slate-400">
                            {truncateHash(chain.current_hash, 8)}
                          </code>
                        </div>
                      </div>

                      {/* Action: Verify */}
                      {!result && !chain.is_fallback && (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleVerify(chain.id);
                          }}
                          disabled={verifying}
                          className="px-2.5 py-1 rounded-md text-xs font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50 disabled:opacity-50 transition-colors"
                        >
                          Verify
                        </button>
                      )}

                      {/* Result badge */}
                      {result && (
                        <span className={`text-xs font-medium ${
                          result.overall_valid
                            ? 'text-emerald-600'
                            : 'text-red-600'
                        }`}>
                          {result.overall_valid ? '✓ Valid' : '✗ Invalid'}
                        </span>
                      )}
                    </button>

                    {/* Expanded Details */}
                    {isSelected && (
                      <div className="px-4 pb-4 border-t border-slate-100 dark:border-slate-700 pt-3 space-y-3">
                        {/* Hash Details */}
                        <div className="bg-slate-50 dark:bg-slate-900/50 rounded-lg p-3 space-y-2">
                          <h4 className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                            Hash Details
                          </h4>
                          <div className="grid grid-cols-2 gap-2 text-xs">
                            <div>
                              <span className="text-slate-500">Current Hash:</span>
                              <code className="ml-1 font-mono text-slate-800 dark:text-slate-200 break-all">
                                {chain.current_hash}
                              </code>
                            </div>
                            <div>
                              <span className="text-slate-500">Algorithm:</span>
                              <span className="ml-1 text-slate-800 dark:text-slate-200">
                                {chain.algorithm.toUpperCase()}
                              </span>
                            </div>
                            {chain.previous_hash && (
                              <div className="col-span-2">
                                <span className="text-slate-500">Previous Hash:</span>
                                <code className="ml-1 font-mono text-slate-800 dark:text-slate-200 break-all">
                                  {chain.previous_hash}
                                </code>
                              </div>
                            )}
                            {chain.previous_document_label && (
                              <div className="col-span-2">
                                <span className="text-slate-500">Previous Document:</span>
                                <span className="ml-1 text-slate-800 dark:text-slate-200">
                                  {chain.previous_document_label} ({chain.previous_document_id})
                                </span>
                              </div>
                            )}
                          </div>
                        </div>

                        {/* Signature Details */}
                        <div className="grid grid-cols-2 gap-2 text-xs">
                          <div>
                            <span className="text-slate-500">Signed by:</span>
                            <span className="ml-1 text-slate-800 dark:text-slate-200 font-medium">{chain.signed_by}</span>
                          </div>
                          <div>
                            <span className="text-slate-500">Certificate:</span>
                            <code className="ml-1 font-mono text-[10px] text-slate-800 dark:text-slate-200">
                              {truncateHash(chain.certificate_fingerprint, 8)}
                            </code>
                          </div>
                          <div>
                            <span className="text-slate-500">Signed at:</span>
                            <span className="ml-1 text-slate-800 dark:text-slate-200">{formatDateTime(chain.signed_at)}</span>
                          </div>
                          <div>
                            <span className="text-slate-500">Document type:</span>
                            <span className="ml-1 text-slate-800 dark:text-slate-200">{chain.document_type.replace(/_/g, ' ')}</span>
                          </div>
                        </div>

                        {/* Verification Result */}
                        {result && (
                          <div className={`rounded-lg p-3 ${
                            result.overall_valid
                              ? 'bg-emerald-50 dark:bg-emerald-900/20 border border-emerald-200 dark:border-emerald-800'
                              : 'bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800'
                          }`}>
                            <div className="flex items-center gap-2">
                              {result.overall_valid ? (
                                <CheckCircle className="w-5 h-5 text-emerald-500" />
                              ) : (
                                <XCircle className="w-5 h-5 text-red-500" />
                              )}
                              <div>
                                <p className={`text-sm font-medium ${
                                  result.overall_valid
                                    ? 'text-emerald-800 dark:text-emerald-300'
                                    : 'text-red-800 dark:text-red-300'
                                }`}>
                                  {result.overall_valid ? 'Verification Passed' : 'Verification Failed'}
                                </p>
                                <p className="text-xs text-slate-500">
                                  Hash: {result.hash_valid ? '✓' : '✗'} · 
                                  Signature: {result.signature_valid ? '✓' : '✗'} · 
                                  Chain: {result.chain_valid === null ? 'N/A' : result.chain_valid ? '✓' : '✗'}
                                </p>
                              </div>
                            </div>
                            {result.errors.length > 0 && (
                              <ul className="mt-2 space-y-0.5">
                                {result.errors.map((err, i) => (
                                  <li key={i} className="text-xs text-red-600 dark:text-red-400 flex items-start gap-1">
                                    <span>•</span>
                                    <span>{err}</span>
                                  </li>
                                ))}
                              </ul>
                            )}
                          </div>
                        )}

                        {/* Re-verify */}
                        <button
                          onClick={() => handleVerify(chain.id)}
                          disabled={verifying}
                          className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-md text-xs font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50 disabled:opacity-50 transition-colors"
                        >
                          {verifying ? (
                            <Loader2 className="w-3 h-3 animate-spin" />
                          ) : (
                            <Shield className="w-3 h-3" />
                          )}
                          Re-verify
                        </button>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* ── Footer ────────────────────────────────────────────── */}
        <div className="flex items-center justify-between px-6 py-3 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50">
          <span className="text-xs text-slate-500">
            {filteredChains.length} chain{filteredChains.length !== 1 ? 's' : ''}
            {chains.length > filteredChains.length && ` (filtered from ${chains.length})`}
          </span>
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
              <CheckCircle className="w-2.5 h-2.5" />
              {results.size} verified
            </span>
            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[10px] font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
              <AlertTriangle className="w-2.5 h-2.5" />
              {chains.filter((c) => c.is_fallback).length} fallback
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

export default ChainVerifier;
