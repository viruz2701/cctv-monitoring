// ═══════════════════════════════════════════════════════════════════════
// WebhookRetryPolicy — конфигурация политики повторных отправок (P2-3.1)
//
// Features:
//   - Exponential backoff toggle
//   - Retry interval selector (10-3600s)
//   - Max retry duration (60-86400s)
//
// Compliance:
//   - OWASP ASVS V5 (Input validation — bounded numeric fields)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw, Info } from '../ui/Icons';
import { Card } from '../ui';
import type { UseFormRegister, FieldErrors } from 'react-hook-form';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface RetryPolicyValues {
  retry_count: number;
  timeout_seconds: number;
  retry_interval_seconds: number;
  retry_backoff: boolean;
  max_retry_duration_seconds: number;
}

interface WebhookRetryPolicyProps {
  register: UseFormRegister<any>;
  errors: FieldErrors;
  watchRetryBackoff: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const RETRY_INTERVAL_OPTIONS = [
  { value: 10, label: '10s' },
  { value: 30, label: '30s' },
  { value: 60, label: '1 min' },
  { value: 120, label: '2 min' },
  { value: 300, label: '5 min' },
  { value: 600, label: '10 min' },
  { value: 1800, label: '30 min' },
  { value: 3600, label: '1 hour' },
];

const MAX_DURATION_OPTIONS = [
  { value: 60, label: '1 min' },
  { value: 300, label: '5 min' },
  { value: 600, label: '10 min' },
  { value: 1800, label: '30 min' },
  { value: 3600, label: '1 hour' },
  { value: 7200, label: '2 hours' },
  { value: 14400, label: '4 hours' },
  { value: 43200, label: '12 hours' },
  { value: 86400, label: '24 hours' },
];

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function WebhookRetryPolicy({
  register,
  errors,
  watchRetryBackoff,
}: WebhookRetryPolicyProps) {
  const { t } = useTranslation();

  return (
    <Card>
      <div className="p-4 space-y-4">
        <div className="flex items-center gap-2">
          <RefreshCw className="w-4 h-4 text-slate-500" />
          <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
            {t('retry_policy') || 'Retry Policy'}
          </h3>
        </div>

        <div className="grid grid-cols-2 gap-4">
          {/* Retry Count */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">
              {t('retry_count') || 'Max Retries'}
            </label>
            <input
              type="number"
              min={0}
              max={10}
              className="w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('retry_count', { valueAsNumber: true })}
            />
            {errors.retry_count && (
              <p className="mt-1 text-sm text-red-600">{errors.retry_count.message as string}</p>
            )}
          </div>

          {/* Timeout */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">
              {t('timeout_seconds') || 'Timeout (s)'}
            </label>
            <input
              type="number"
              min={1}
              max={120}
              className="w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('timeout_seconds', { valueAsNumber: true })}
            />
            {errors.timeout_seconds && (
              <p className="mt-1 text-sm text-red-600">{errors.timeout_seconds.message as string}</p>
            )}
          </div>
        </div>

        {/* Exponential Backoff Toggle */}
        <label className="flex items-center gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors">
          <input
            type="checkbox"
            className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
            {...register('retry_backoff')}
          />
          <div className="flex-1">
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-medium text-slate-700 dark:text-slate-200">
                {t('exponential_backoff') || 'Exponential Backoff'}
              </span>
              <Info className="w-3.5 h-3.5 text-slate-400" />
            </div>
            <p className="text-[10px] text-slate-400">
              {t('backoff_hint') ||
                'Doubles the retry interval after each attempt (e.g., 60s → 120s → 240s)'}
            </p>
          </div>
        </label>

        <div className="grid grid-cols-2 gap-4">
          {/* Retry Interval */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">
              {t('retry_interval') || 'Retry Interval'}
            </label>
            <select
              className="w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('retry_interval_seconds', { valueAsNumber: true })}
            >
              {RETRY_INTERVAL_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            {errors.retry_interval_seconds && (
              <p className="mt-1 text-sm text-red-600">
                {errors.retry_interval_seconds.message as string}
              </p>
            )}
            {watchRetryBackoff && (
              <p className="mt-1 text-[10px] text-amber-600 dark:text-amber-400">
                {t('backoff_interval_hint') ||
                  'Base interval — will double on each retry'}
              </p>
            )}
          </div>

          {/* Max Retry Duration */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">
              {t('max_retry_duration') || 'Max Retry Duration'}
            </label>
            <select
              className="w-full px-3.5 py-2.5 text-sm text-slate-900 dark:text-white bg-white dark:bg-slate-900 border border-slate-300 dark:border-slate-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              {...register('max_retry_duration_seconds', { valueAsNumber: true })}
            >
              {MAX_DURATION_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
            {errors.max_retry_duration_seconds && (
              <p className="mt-1 text-sm text-red-600">
                {errors.max_retry_duration_seconds.message as string}
              </p>
            )}
          </div>
        </div>

        {/* Summary */}
        <div className="px-3 py-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
          <p className="text-[10px] text-slate-500 dark:text-slate-400">
            {watchRetryBackoff
              ? (t('retry_summary_backoff') ||
                  'Retries will use exponential backoff starting from the configured interval, up to the max duration.')
              : (t('retry_summary_fixed') ||
                  'Retries will use a fixed interval between attempts, up to the max duration.')}
          </p>
        </div>
      </div>
    </Card>
  );
}
