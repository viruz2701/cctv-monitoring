// ═══════════════════════════════════════════════════════════════════════
// WOCompletionFlow.tsx — Auto-fill TO Journals при закрытии WorkOrder
//
// UX-3.2: При закрытии Work Order (статус "completed") автоматически
// создаются записи в TO-журналах с pre-fill полями.
//
// Feature Flag: to_auto_generation (default: false)
//
// Pre-fill: device, date, technician, location, time (из work_order)
// Required fields (manual): checklist_notes, defects, customer_signature
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — фича контролируется флагом)
//   - OWASP ASVS V1.8 (Feature flags не раскрывают sensitive functionality)
//   - OWASP ASVS V5.1 (Input validation)
//   - ISO 27001 A.12.4 (Audit trail)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect, useCallback } from 'react';
import { isFeatureEnabled } from '../../config/featureFlags';
import { request } from '../../services/api/client';
import type { WorkOrder } from '../../services/api/workOrders';

// ── Types ──────────────────────────────────────────────────────────────

export interface TOJournalEntry {
  id: string;
  work_order_id: string;
  device_id: string;
  device_name: string;
  technician_id: string;
  technician_name: string;
  site_name: string;
  entry_type: 'to_auto_filled' | 'to_manual_update';
  started_at?: string;
  completed_at: string;
  duration_minutes: number;
  checklist_notes: string;
  defects: string;
  customer_signature: string;
  is_completed: boolean;
  required_fields: Array<'checklist_notes' | 'defects' | 'customer_signature'>;
  created_at: string;
  updated_at: string;
}

export interface TOJournalSummary {
  entries: TOJournalEntry[];
  total_auto_filled: number;
  pending_manual: number;
  is_complete: boolean;
}

export interface RegulatoryChecklistResult {
  all_required_filled: boolean;
  missing_fields?: Array<'checklist_notes' | 'defects' | 'customer_signature'>;
  entries_pending: number;
}

// ── Required Field Labels ──────────────────────────────────────────────

const REQUIRED_FIELD_LABELS: Record<string, { label: string; placeholder: string }> = {
  checklist_notes: {
    label: 'Checklist Notes',
    placeholder: 'Enter checklist completion notes...',
  },
  defects: {
    label: 'Defects Found',
    placeholder: 'Describe any defects found during inspection...',
  },
  customer_signature: {
    label: 'Customer Signature',
    placeholder: 'Enter customer name or signature reference...',
  },
};

// ── Props ──────────────────────────────────────────────────────────────

interface WOCompletionFlowProps {
  /** Work Order для которого выполняется completion flow */
  workOrder: WorkOrder;
  /** Колбэк при успешном завершении всех required полей */
  onComplete?: () => void;
  /** Колбэк при отмене flow */
  onCancel?: () => void;
}

// ── WOCompletionFlow Component ─────────────────────────────────────────

/**
 * WOCompletionFlow — компонент auto-fill TO журналов при закрытии WorkOrder.
 *
 * Показывается при статусе "completed" и включённом feature flag to_auto_generation.
 * Отображает список созданных записей TO-журнала и форму для required полей.
 * Блокирует закрытие если не заполнены все required поля.
 */
export function WOCompletionFlow({ workOrder, onComplete, onCancel }: WOCompletionFlowProps) {
  const [summary, setSummary] = useState<TOJournalSummary | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Проверяем feature flag
  const isEnabled = isFeatureEnabled('to_auto_generation');

  // Флаг — показывать ли completion flow
  // Показываем только если статус completed и флаг включён
  const shouldShow = isEnabled && workOrder.status === 'completed';

  // ── Auto-fill TO Journal ───────────────────────────────────────────
  const handleAutoFill = useCallback(async () => {
    if (!shouldShow) return;

    setLoading(true);
    setError(null);

    try {
      // Сначала проверяем, есть ли уже записи
      const existing = await request<TOJournalSummary>(
        `/work-orders/${workOrder.id}/to-journal`,
      );

      if (existing.entries.length > 0) {
        setSummary(existing);
        setLoading(false);
        return;
      }

      // Рассчитываем длительность
      let durationMin = 0;
      if (workOrder.started_at && workOrder.completed_at) {
        const start = new Date(workOrder.started_at);
        const end = new Date(workOrder.completed_at);
        durationMin = Math.round((end.getTime() - start.getTime()) / 60000);
      }

      // Auto-fill
      const result = await request<{ entries: TOJournalEntry[]; total: number }>(
        `/work-orders/${workOrder.id}/to-journal/auto-fill`,
        {
          method: 'POST',
          body: JSON.stringify({
            device_id: workOrder.device_id,
            technician_id: workOrder.assigned_to,
            technician_name: workOrder.assignee_name || '',
            site_name: workOrder.device_name || '',
            started_at: workOrder.started_at,
            completed_at: workOrder.completed_at || new Date().toISOString(),
            duration_minutes: durationMin,
          }),
        },
      );

      // Получаем полный summary
      const updatedSummary = await request<TOJournalSummary>(
        `/work-orders/${workOrder.id}/to-journal`,
      );
      setSummary(updatedSummary);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to auto-fill TO journal');
    } finally {
      setLoading(false);
    }
  }, [workOrder, shouldShow]);

  // ── Update Required Field ──────────────────────────────────────────
  const handleUpdateField = useCallback(
    async (entryId: string, field: string, value: string) => {
      setSubmitting(true);
      setError(null);

      try {
        const updateBody: Record<string, string> = {};
        updateBody[field] = value;

        await request<TOJournalEntry>(`/to-journal/${entryId}`, {
          method: 'PUT',
          body: JSON.stringify(updateBody),
        });

        // Обновляем summary
        const updatedSummary = await request<TOJournalSummary>(
          `/work-orders/${workOrder.id}/to-journal`,
        );
        setSummary(updatedSummary);

        // Если все completed — вызываем onComplete
        if (updatedSummary.is_complete) {
          onComplete?.();
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to update field');
      } finally {
        setSubmitting(false);
      }
    },
    [workOrder.id, onComplete],
  );

  // ── Check Regulatory Compliance ───────────────────────────────────
  const [complianceResult, setComplianceResult] = useState<RegulatoryChecklistResult | null>(null);

  const checkCompliance = useCallback(async () => {
    try {
      const result = await request<RegulatoryChecklistResult>(
        `/work-orders/${workOrder.id}/to-journal/check`,
      );
      setComplianceResult(result);
      return result;
    } catch {
      return null;
    }
  }, [workOrder.id]);

  // ── Load on mount ─────────────────────────────────────────────────
  useEffect(() => {
    if (shouldShow) {
      handleAutoFill();
      checkCompliance();
    }
  }, [shouldShow, handleAutoFill, checkCompliance]);

  // ── Render: Feature Disabled ──────────────────────────────────────
  if (!isEnabled) {
    return null;
  }

  // ── Render: Not Completed ─────────────────────────────────────────
  if (workOrder.status !== 'completed') {
    return null;
  }

  // ── Render: Loading ───────────────────────────────────────────────
  if (loading) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex items-center justify-center space-x-2">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-blue-600 border-t-transparent" />
          <span className="text-sm text-gray-600">Auto-filling TO journals...</span>
        </div>
      </div>
    );
  }

  // ── Render: Error ─────────────────────────────────────────────────
  if (error) {
    return (
      <div className="rounded-lg border border-red-200 bg-red-50 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-medium text-red-800">TO Journal Error</h3>
            <p className="mt-1 text-sm text-red-600">{error}</p>
          </div>
          <button
            onClick={() => setError(null)}
            className="text-sm text-red-500 hover:text-red-700"
          >
            Dismiss
          </button>
        </div>
      </div>
    );
  }

  // ── Render: No Entries ────────────────────────────────────────────
  if (!summary || summary.entries.length === 0) {
    return (
      <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-gray-900">TO Journal</h3>
        <p className="mt-2 text-sm text-gray-500">No TO journal entries found.</p>
      </div>
    );
  }

  // ── Render: Compliance Warning ────────────────────────────────────
  const hasComplianceWarning =
    complianceResult && !complianceResult.all_required_filled;

  // ── Render: Main Component ────────────────────────────────────────
  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold text-gray-900">
              TO Journal Entries
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Auto-filled from work order completion
            </p>
          </div>
          <span className="inline-flex items-center rounded-full bg-blue-50 px-2.5 py-0.5 text-xs font-medium text-blue-700">
            {summary.total_auto_filled} entries
          </span>
        </div>

        {/* Compliance Warning */}
        {hasComplianceWarning && (
          <div className="mt-4 rounded-md border border-amber-200 bg-amber-50 p-4">
            <div className="flex">
              <span className="text-amber-600 text-lg mr-2">⚠</span>
              <div>
                <h4 className="text-sm font-medium text-amber-800">
                  Regulatory Checklist Incomplete
                </h4>
                <p className="mt-1 text-sm text-amber-700">
                  {complianceResult!.entries_pending} entry(ies) require manual fields.
                  Complete all required fields before closing.
                </p>
                {complianceResult!.missing_fields && complianceResult!.missing_fields.length > 0 && (
                  <ul className="mt-2 list-disc list-inside text-sm text-amber-700">
                    {complianceResult!.missing_fields.map((field) => (
                      <li key={field}>
                        {REQUIRED_FIELD_LABELS[field]?.label || field}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Entries List */}
      {summary.entries.map((entry) => (
        <div
          key={entry.id}
          className={`rounded-lg border p-6 shadow-sm ${
            entry.is_completed
              ? 'border-green-200 bg-green-50'
              : 'border-amber-200 bg-white'
          }`}
        >
          {/* Pre-filled Fields */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider">
                Device
              </label>
              <p className="mt-1 text-sm font-medium text-gray-900">
                {entry.device_name || entry.device_id}
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider">
                Date
              </label>
              <p className="mt-1 text-sm text-gray-900">
                {new Date(entry.completed_at).toLocaleDateString()}
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider">
                Technician
              </label>
              <p className="mt-1 text-sm text-gray-900">
                {entry.technician_name || entry.technician_id || '—'}
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider">
                Location
              </label>
              <p className="mt-1 text-sm text-gray-900">
                {entry.site_name || '—'}
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider">
                Duration
              </label>
              <p className="mt-1 text-sm text-gray-900">
                {entry.duration_minutes > 0
                  ? `${entry.duration_minutes} min`
                  : '—'}
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-gray-500 uppercase tracking-wider">
                Type
              </label>
              <p className="mt-1 text-sm text-gray-900">
                {entry.entry_type === 'to_auto_filled' ? 'Auto-filled' : 'Manual update'}
              </p>
            </div>
          </div>

          {/* Required Fields (Manual) */}
          <div className="mt-6 border-t border-gray-200 pt-4">
            <h4 className="text-sm font-semibold text-gray-700 mb-3">
              Required Fields <span className="text-amber-600">⚠ manual</span>
            </h4>
            <div className="space-y-4">
              {entry.required_fields.map((field) => {
                const fieldConfig = REQUIRED_FIELD_LABELS[field];
                if (!fieldConfig) return null;

                const isFilled =
                  (field === 'checklist_notes' && entry.checklist_notes) ||
                  (field === 'defects' && entry.defects) ||
                  (field === 'customer_signature' && entry.customer_signature);

                return (
                  <div key={field}>
                    <label
                      htmlFor={`${entry.id}-${field}`}
                      className="block text-sm font-medium text-gray-700"
                    >
                      {fieldConfig.label}{' '}
                      <span className="text-amber-500">*</span>
                      {isFilled && (
                        <span className="ml-2 text-green-600 text-xs">✓ Filled</span>
                      )}
                    </label>
                    <div className="mt-1 flex gap-2">
                      <input
                        id={`${entry.id}-${field}`}
                        type="text"
                        className={`block w-full rounded-md border px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 ${
                          isFilled
                            ? 'border-green-300 bg-green-50 text-gray-700'
                            : 'border-amber-300 bg-white text-gray-900'
                        }`}
                        placeholder={fieldConfig.placeholder}
                        defaultValue={
                          field === 'checklist_notes'
                            ? entry.checklist_notes
                            : field === 'defects'
                            ? entry.defects
                            : field === 'customer_signature'
                            ? entry.customer_signature
                            : ''
                        }
                        disabled={entry.is_completed || submitting}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter') {
                            e.preventDefault();
                            const target = e.target as HTMLInputElement;
                            handleUpdateField(entry.id, field, target.value);
                          }
                        }}
                      />
                      <button
                        onClick={(e) => {
                          const input = (e.target as HTMLElement)
                            .closest('div')
                            ?.querySelector('input');
                          if (input) {
                            handleUpdateField(entry.id, field, input.value);
                          }
                        }}
                        disabled={entry.is_completed || submitting}
                        className={`inline-flex items-center rounded-md px-3 py-2 text-sm font-medium shadow-sm ${
                          isFilled
                            ? 'bg-green-100 text-green-700 hover:bg-green-200'
                            : 'bg-blue-600 text-white hover:bg-blue-700'
                        } disabled:opacity-50 disabled:cursor-not-allowed`}
                      >
                        {submitting ? '...' : isFilled ? 'Update' : 'Save'}
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Status Badge */}
          <div className="mt-4 flex items-center justify-between">
            <span
              className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                entry.is_completed
                  ? 'bg-green-100 text-green-800'
                  : 'bg-amber-100 text-amber-800'
              }`}
            >
              {entry.is_completed ? '✓ Completed' : '⚠ Pending manual fields'}
            </span>
          </div>
        </div>
      ))}

      {/* Actions */}
      <div className="flex justify-end space-x-3 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        {onCancel && (
          <button
            onClick={onCancel}
            className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50"
          >
            Cancel
          </button>
        )}
        {onComplete && (
          <button
            onClick={async () => {
              const result = await checkCompliance();
              if (result && !result.all_required_filled) {
                setError(
                  `Cannot close: ${result.entries_pending} entry(ies) have unfilled required fields`,
                );
                return;
              }
              onComplete();
            }}
            disabled={hasComplianceWarning || submitting}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Confirm Close
          </button>
        )}
      </div>
    </div>
  );
}

export default WOCompletionFlow;
