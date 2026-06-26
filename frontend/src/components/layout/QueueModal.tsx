// QueueModal — модальное окно для просмотра offline очереди.
//
// P1-2.3: Queue modal с conflict resolution
// - Показывает все pending operations
// - Manual retry button
// - Exponential backoff visualization

import React from 'react';
import { useTranslation } from 'react-i18next';
import { X, RefreshCw, Clock, AlertCircle, CheckCircle } from 'lucide-react';

interface QueuedOperation {
    id: string;
    type: 'create' | 'update' | 'delete';
    entity: string;
    entityId: string;
    timestamp: string;
    status: 'pending' | 'in_flight' | 'failed' | 'completed';
    retryCount: number;
    error?: string;
}

interface QueueModalProps {
    isOpen: boolean;
    onClose: () => void;
    operations: QueuedOperation[];
    onRetry: (id: string) => void;
    onRetryAll: () => void;
    onClearCompleted: () => void;
}

const statusIcons: Record<string, React.ElementType> = {
    pending: Clock,
    in_flight: RefreshCw,
    failed: AlertCircle,
    completed: CheckCircle,
};

const statusColors: Record<string, string> = {
    pending: 'text-slate-400',
    in_flight: 'text-blue-500',
    failed: 'text-red-500',
    completed: 'text-emerald-500',
};

export function QueueModal({
    isOpen,
    onClose,
    operations,
    onRetry,
    onRetryAll,
    onClearCompleted,
}: QueueModalProps) {
    const { t } = useTranslation();
    if (!isOpen) return null;

    const failedCount = operations.filter((o) => o.status === 'failed').length;
    const completedCount = operations.filter((o) => o.status === 'completed').length;

    return (
        <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
            onClick={onClose}
        >
            <div
                className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col"
                onClick={(e) => e.stopPropagation()}
            >
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700">
                    <div>
                        <h2 className="text-lg font-bold text-slate-900 dark:text-white">
                            {t('sync_queue') || 'Sync Queue'}
                        </h2>
                        <p className="text-sm text-slate-500 dark:text-slate-400">
                            {operations.length} {t('operations') || 'operations'} (
                            {failedCount} {t('failed') || 'failed'})
                        </p>
                    </div>
                    <button
                        onClick={onClose}
                        className="p-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Operations list */}
                <div className="flex-1 overflow-y-auto p-4 space-y-2">
                    {operations.length === 0 ? (
                        <div className="text-center py-8 text-slate-400">
                            <CheckCircle className="w-12 h-12 mx-auto mb-2 text-emerald-400" />
                            <p className="font-medium">
                                {t('queue_empty') || 'Queue is empty'}
                            </p>
                            <p className="text-sm">
                                {t('all_synced') || 'All operations synced'}
                            </p>
                        </div>
                    ) : (
                        operations.map((op) => {
                            const StatusIcon = statusIcons[op.status] || Clock;
                            const statusColor =
                                statusColors[op.status] || 'text-slate-400';
                            return (
                                <div
                                    key={op.id}
                                    className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900 rounded-xl"
                                >
                                    <div className="flex items-center gap-3">
                                        <StatusIcon
                                            className={`w-5 h-5 ${statusColor}`}
                                        />
                                        <div>
                                            <p className="text-sm font-medium text-slate-900 dark:text-white capitalize">
                                                {op.type} {op.entity}
                                            </p>
                                            <p className="text-xs text-slate-400">
                                                {op.entityId}
                                                {op.retryCount > 0 &&
                                                    ` · ${op.retryCount} retries`}
                                                {' · '}
                                                {new Date(
                                                    op.timestamp,
                                                ).toLocaleTimeString()}
                                            </p>
                                            {op.error && (
                                                <p className="text-xs text-red-500 mt-0.5">
                                                    {op.error}
                                                </p>
                                            )}
                                        </div>
                                    </div>
                                    {op.status === 'failed' && (
                                        <button
                                            onClick={() => onRetry(op.id)}
                                            className="p-2 text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg"
                                        >
                                            <RefreshCw className="w-4 h-4" />
                                        </button>
                                    )}
                                </div>
                            );
                        })
                    )}
                </div>

                {/* Footer */}
                <div className="flex gap-2 p-4 border-t border-slate-200 dark:border-slate-700">
                    {failedCount > 0 && (
                        <button
                            onClick={onRetryAll}
                            className="flex-1 py-2.5 bg-blue-600 text-white rounded-xl hover:bg-blue-700 transition-colors text-sm font-medium"
                        >
                            {t('retry_all') || 'Retry All'} ({failedCount})
                        </button>
                    )}
                    {completedCount > 0 && (
                        <button
                            onClick={onClearCompleted}
                            className="px-4 py-2.5 bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-xl hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors text-sm font-medium"
                        >
                            {t('clear_completed') || 'Clear Completed'}
                        </button>
                    )}
                </div>
            </div>
        </div>
    );
}
