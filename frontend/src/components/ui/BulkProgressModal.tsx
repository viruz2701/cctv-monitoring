// ═══════════════════════════════════════════════════════════════════════
// BulkProgressModal — Modal with real-time progress for bulk operations
//
// P1-1.6: Bulk Operations Progress
//   - Progress bar with percentage
//   - Real-time status per item (pending, processing, done, failed)
//   - Cancel in-flight operations
//   - Error retry for failed items
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
    Loader2,
    CheckCircle,
    XCircle,
    AlertTriangle,
    X,
    RotateCcw,
    Square,
    Ban,
} from 'lucide-react';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export type BulkItemStatus = 'pending' | 'processing' | 'done' | 'failed' | 'cancelled';

export interface BulkProgressItem {
    id: string;
    label: string;
    status: BulkItemStatus;
    error?: string;
}

export interface BulkProgressState {
    /** Total number of items */
    total: number;
    /** Current items with their status */
    items: BulkProgressItem[];
    /** Whether the operation is running */
    isRunning: boolean;
    /** Whether the operation was cancelled by user */
    isCancelled: boolean;
    /** Operation label (e.g. "Deleting work orders...") */
    operationLabel: string;
}

interface BulkProgressModalProps {
    /** Current progress state */
    state: BulkProgressState;
    /** Called when user clicks cancel */
    onCancel: () => void;
    /** Called when user clicks retry for failed items */
    onRetryAll: () => void;
    /** Called when user clicks retry on a specific item */
    onRetryItem?: (itemId: string) => void;
    /** Called to close the modal (when operation completes/cancelled) */
    onClose: () => void;
    /** Optional className */
    className?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Helper
// ═══════════════════════════════════════════════════════════════════════

function getStatusIcon(status: BulkItemStatus, size = 16): React.ReactNode {
    switch (status) {
        case 'processing':
            return <Loader2 size={size} className="animate-spin text-blue-500" />;
        case 'done':
            return <CheckCircle size={size} className="text-emerald-500" />;
        case 'failed':
            return <XCircle size={size} className="text-red-500" />;
        case 'cancelled':
            return <Ban size={size} className="text-slate-400" />;
        default:
            return <span className="w-4 h-4 inline-block rounded-full border-2 border-slate-300" />;
    }
}

function getStatusClass(status: BulkItemStatus): string {
    switch (status) {
        case 'processing': return 'bg-blue-50 dark:bg-blue-900/10';
        case 'done': return 'bg-emerald-50 dark:bg-emerald-900/10';
        case 'failed': return 'bg-red-50 dark:bg-red-900/10';
        case 'cancelled': return 'bg-slate-50 dark:bg-slate-800/50 opacity-60';
        default: return '';
    }
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function BulkProgressModal({
    state,
    onCancel,
    onRetryAll,
    onRetryItem,
    onClose,
    className = '',
}: BulkProgressModalProps) {
    const { t } = useTranslation();
    const listRef = useRef<HTMLDivElement>(null);

    const completed = state.items.filter(i => i.status === 'done').length;
    const failed = state.items.filter(i => i.status === 'failed').length;
    const cancelled = state.items.filter(i => i.status === 'cancelled').length;
    const processing = state.items.filter(i => i.status === 'processing').length;
    const pending = state.items.filter(i => i.status === 'pending').length;

    const progress = state.total > 0
        ? Math.round(((completed + failed + cancelled) / state.total) * 100)
        : 0;

    const isComplete = !state.isRunning || state.isCancelled;
    const hasFailed = failed > 0;

    // Auto-scroll to latest processing item
    useEffect(() => {
        if (!listRef.current) return;
        const processingEl = listRef.current.querySelector('[data-status="processing"]');
        if (processingEl) {
            processingEl.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
        }
    }, [state.items]);

    return (
        <div className={`fixed inset-0 z-50 flex items-center justify-center ${className}`}>
            {/* Overlay */}
            <div className="absolute inset-0 bg-black/40" onClick={isComplete ? onClose : undefined} />

            {/* Modal */}
            <div className="relative w-full max-w-lg bg-white dark:bg-slate-800 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden mx-4">
                {/* Header */}
                <div className="flex items-center justify-between px-5 py-4 border-b border-slate-200 dark:border-slate-700">
                    <div className="flex items-center gap-3">
                        {state.isRunning && !state.isCancelled ? (
                            <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
                        ) : hasFailed ? (
                            <AlertTriangle className="w-5 h-5 text-amber-500" />
                        ) : (
                            <CheckCircle className="w-5 h-5 text-emerald-500" />
                        )}
                        <div>
                            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                                {state.operationLabel}
                            </h3>
                            <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                {completed}/{state.total} {t('completed') || 'completed'}
                                {failed > 0 && ` · ${failed} ${t('failed') || 'failed'}`}
                                {cancelled > 0 && ` · ${cancelled} ${t('cancelled') || 'cancelled'}`}
                            </p>
                        </div>
                    </div>
                    {isComplete && (
                        <button
                            onClick={onClose}
                            className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                            aria-label={t('close') || 'Close'}
                        >
                            <X className="w-4 h-4" />
                        </button>
                    )}
                </div>

                {/* Progress Bar */}
                <div className="px-5 py-3">
                    <div className="flex items-center justify-between mb-1.5">
                        <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
                            {progress}%
                        </span>
                        <span className="text-xs text-slate-500 dark:text-slate-400">
                            {processing > 0
                                ? (t('processing_items', { count: processing }) || `${processing} processing...`)
                                : pending > 0
                                    ? (t('pending_items', { count: pending }) || `${pending} remaining`)
                                    : t('done') || 'Done'
                            }
                        </span>
                    </div>
                    <div className="w-full h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
                        <div
                            className="h-full rounded-full transition-all duration-300 ease-out"
                            style={{
                                width: `${progress}%`,
                                backgroundColor: hasFailed ? '#f59e0b' : '#22c55e',
                            }}
                        />
                    </div>
                    {/* Status breakdown */}
                    <div className="flex items-center gap-4 mt-2 text-[10px] text-slate-400">
                        {pending > 0 && <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-slate-300" />{pending} pending</span>}
                        {processing > 0 && <span className="flex items-center gap-1"><Loader2 className="w-2.5 h-2.5 animate-spin text-blue-500" />{processing}</span>}
                        {completed > 0 && <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-emerald-500" />{completed} done</span>}
                        {failed > 0 && <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-red-500" />{failed} failed</span>}
                    </div>
                </div>

                {/* Item List */}
                <div
                    ref={listRef}
                    className="max-h-60 overflow-y-auto border-t border-slate-100 dark:border-slate-700 divide-y divide-slate-100 dark:divide-slate-700/50"
                >
                    {state.items.map((item) => (
                        <div
                            key={item.id}
                            data-status={item.status}
                            className={`flex items-center gap-3 px-5 py-2.5 text-sm transition-colors ${getStatusClass(item.status)}`}
                        >
                            {getStatusIcon(item.status)}
                            <span className={`flex-1 min-w-0 truncate ${
                                item.status === 'cancelled'
                                    ? 'text-slate-400 line-through'
                                    : item.status === 'failed'
                                        ? 'text-red-700 dark:text-red-400'
                                        : 'text-slate-700 dark:text-slate-300'
                            }`}>
                                {item.label}
                            </span>
                            {item.status === 'failed' && onRetryItem && (
                                <button
                                    onClick={() => onRetryItem(item.id)}
                                    className="shrink-0 p-1 text-slate-400 hover:text-blue-600 rounded hover:bg-slate-100 dark:hover:bg-slate-700 transition-colors"
                                    title={t('retry') || 'Retry'}
                                    aria-label={`Retry ${item.label}`}
                                >
                                    <RotateCcw className="w-3.5 h-3.5" />
                                </button>
                            )}
                            {item.status === 'failed' && item.error && (
                                <span
                                    className="shrink-0 group relative cursor-help"
                                    title={item.error}
                                >
                                    <AlertTriangle className="w-3.5 h-3.5 text-amber-500" />
                                </span>
                            )}
                        </div>
                    ))}
                </div>

                {/* Footer Actions */}
                <div className="flex items-center justify-between px-5 py-3 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50">
                    <div className="text-xs text-slate-400">
                        {state.isRunning && !state.isCancelled
                            ? (t('processing_items_n', { n: state.total - completed - failed - cancelled }) || `${state.total - completed - failed - cancelled} remaining`)
                            : (t('completed_items_n', { n: completed }) || `${completed} completed`)
                        }
                    </div>
                    <div className="flex items-center gap-2">
                        {state.isRunning && !state.isCancelled ? (
                            <button
                                onClick={onCancel}
                                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors"
                            >
                                <Square className="w-3.5 h-3.5" />
                                {t('cancel') || 'Cancel'}
                            </button>
                        ) : hasFailed ? (
                            <button
                                onClick={onRetryAll}
                                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-amber-600 hover:bg-amber-700 rounded-lg transition-colors"
                            >
                                <RotateCcw className="w-3.5 h-3.5" />
                                {t('retry_failed', { count: failed }) || `Retry ${failed} failed`}
                            </button>
                        ) : isComplete ? (
                            <button
                                onClick={onClose}
                                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
                            >
                                <CheckCircle className="w-3.5 h-3.5" />
                                {t('done') || 'Done'}
                            </button>
                        ) : null}
                    </div>
                </div>
            </div>
        </div>
    );
}