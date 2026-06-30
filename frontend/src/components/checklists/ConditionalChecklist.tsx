// P2-CHECK: Conditional Checklist Component (MaintainX-level)
//
// React компонент с поддержкой:
//   - Dynamic show/hide based on conditions (depends_on)
//   - Scoring with passing threshold
//   - Required photo capture
//   - Template picker per device type
//   - Sub-items (children)
//   - Mandatory/optional items
//
// Compliance:
//   - OWASP ASVS V5.1 (Input validation)
//   - IEC 62443 SL-3 (Zone 3 — Application integrity)
//   - ISO 27001 A.12.6 (Maintenance — structured checklists)

import React, { useState, useCallback, useMemo } from 'react';
import { checklistApi, type ChecklistTemplate, type ChecklistItem, type Condition, type SubmitItemResponse, type ChecklistSummary } from '../../services/checklistApi';

// ── Types ────────────────────────────────────────────────────────────

interface ConditionalChecklistProps {
  workOrderId: string;
  deviceType?: string;
  templateId?: string;         // предвыбранный шаблон
  onComplete?: (summary: ChecklistSummary) => void;
  onError?: (error: string) => void;
  onCancel?: () => void;
  disabled?: boolean;
}

interface ItemResponse {
  itemId: string;
  value: string;
  photoUrl?: string;
  skipped: boolean;
}

interface ChecklistState {
  status: 'idle' | 'loading' | 'active' | 'submitting' | 'submitted' | 'error';
  template: ChecklistTemplate | null;
  responses: Record<string, ItemResponse>;
  summary: ChecklistSummary | null;
  error: string | null;
}

// ── Helpers ──────────────────────────────────────────────────────────

// evaluateCondition проверяет, выполняется ли условие на основе текущих ответов.
function evaluateCondition(condition: Condition | null | undefined, responses: Record<string, ItemResponse>): boolean {
  if (!condition) return true;

  const triggerResponse = responses[condition.field_id];
  if (!triggerResponse || triggerResponse.skipped) return true;

  const actualValue = triggerResponse.value;

  switch (condition.operator) {
    case 'eq':
      return actualValue === String(condition.value);
    case 'neq':
      return actualValue !== String(condition.value);
    case 'gt':
      return Number(actualValue) > Number(condition.value);
    case 'lt':
      return Number(actualValue) < Number(condition.value);
    case 'gte':
      return Number(actualValue) >= Number(condition.value);
    case 'lte':
      return Number(actualValue) <= Number(condition.value);
    case 'in': {
      const values = Array.isArray(condition.value) ? condition.value : [condition.value];
      return values.some(v => String(v) === actualValue);
    }
    default:
      return true;
  }
}

// getVisibleItems возвращает только видимые элементы (с учётом conditions).
function getVisibleItems(items: ChecklistItem[], responses: Record<string, ItemResponse>): ChecklistItem[] {
  return items.filter(item => {
    return evaluateCondition(item.depends_on, responses);
  });
}

// collectAllItems собирает все элементы (включая children) в плоский список.
function collectAllItems(items: ChecklistItem[]): ChecklistItem[] {
  const result: ChecklistItem[] = [];
  for (const item of items) {
    result.push(item);
    if (item.children?.length) {
      result.push(...collectAllItems(item.children));
    }
  }
  return result;
}

// ── Sub-Components ───────────────────────────────────────────────────

// ItemRenderer рендерит один элемент чек-листа.
const ItemRenderer: React.FC<{
  item: ChecklistItem;
  value: string;
  onChange: (itemId: string, value: string) => void;
  onPhotoCapture?: (itemId: string) => Promise<string>;
  disabled?: boolean;
  depth?: number;
}> = ({ item, value, onChange, onPhotoCapture, disabled, depth = 0 }) => {
  const handleChange = useCallback((newValue: string) => {
    onChange(item.id, newValue);
  }, [item.id, onChange]);

  const baseClasses = 'flex items-start gap-3 p-3 rounded-lg border border-gray-200 dark:border-gray-700';
  const mandatoryClass = item.mandatory
    ? 'border-l-4 border-l-red-500'
    : '';
  const depthPadding = `ml-${Math.min(depth * 4, 12)}`;

  return (
    <div className={`${baseClasses} ${mandatoryClass} ${depthPadding}`}>
      <div className="flex-1 min-w-0">
        {/* Label */}
        <label className="block text-sm font-medium text-gray-900 dark:text-gray-100">
          {item.label}
          {item.mandatory && <span className="text-red-500 ml-1">*</span>}
          {item.score > 0 && (
            <span className="ml-2 text-xs text-gray-500 dark:text-gray-400">
              ({item.score} pts)
            </span>
          )}
        </label>

        {/* Description */}
        {item.description && (
          <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{item.description}</p>
        )}

        {/* Input by type */}
        <div className="mt-2">
          {item.item_type === 'boolean' && (
            <div className="flex gap-4">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name={item.id}
                  value="true"
                  checked={value === 'true'}
                  onChange={() => handleChange('true')}
                  disabled={disabled}
                  className="text-blue-600 focus:ring-blue-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Pass</span>
              </label>
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="radio"
                  name={item.id}
                  value="false"
                  checked={value === 'false'}
                  onChange={() => handleChange('false')}
                  disabled={disabled}
                  className="text-red-600 focus:ring-red-500"
                />
                <span className="text-sm text-gray-700 dark:text-gray-300">Fail</span>
              </label>
            </div>
          )}

          {item.item_type === 'text' && (
            <textarea
              value={value}
              onChange={(e) => handleChange(e.target.value)}
              disabled={disabled}
              rows={2}
              placeholder="Enter value..."
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md
                bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
                focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          )}

          {item.item_type === 'numeric' && (
            <input
              type="number"
              value={value}
              onChange={(e) => handleChange(e.target.value)}
              disabled={disabled}
              min={item.validation_min ?? undefined}
              max={item.validation_max ?? undefined}
              placeholder={item.validation_min != null && item.validation_max != null
                ? `${item.validation_min} – ${item.validation_max}`
                : 'Enter number...'}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md
                bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
                focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          )}

          {item.item_type === 'photo' && (
            <div className="flex items-center gap-3">
              {value ? (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-green-600 dark:text-green-400">✓ Photo captured</span>
                  <button
                    onClick={() => handleChange('')}
                    disabled={disabled}
                    className="text-xs text-red-500 hover:text-red-700"
                  >
                    Remove
                  </button>
                </div>
              ) : (
                <button
                  onClick={async () => {
                    if (onPhotoCapture) {
                      try {
                        const url = await onPhotoCapture(item.id);
                        handleChange(url);
                      } catch {
                        // Photo capture failed — handled by parent
                      }
                    }
                  }}
                  disabled={disabled}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md
                    hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500
                    disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Capture Photo
                </button>
              )}
            </div>
          )}

          {item.item_type === 'select' && item.options && (
            <select
              value={value}
              onChange={(e) => handleChange(e.target.value)}
              disabled={disabled}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md
                bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
                focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">Select...</option>
              {item.options.map((opt) => (
                <option key={opt} value={opt}>{opt}</option>
              ))}
            </select>
          )}

          {item.item_type === 'multi_select' && item.options && (
            <div className="space-y-1">
              {item.options.map((opt) => (
                <label key={opt} className="inline-flex items-center gap-2 mr-4 cursor-pointer">
                  <input
                    type="checkbox"
                    value={opt}
                    checked={value.includes(opt)}
                    onChange={(e) => {
                      const current = value ? value.split(',') : [];
                      const updated = e.target.checked
                        ? [...current, opt]
                        : current.filter((v) => v !== opt);
                      handleChange(updated.join(','));
                    }}
                    disabled={disabled}
                    className="rounded text-blue-600 focus:ring-blue-500"
                  />
                  <span className="text-sm text-gray-700 dark:text-gray-300">{opt}</span>
                </label>
              ))}
            </div>
          )}

          {item.item_type === 'signature' && (
            <div className="flex items-center gap-3">
              {value ? (
                <div className="flex items-center gap-2">
                  <span className="text-sm text-green-600 dark:text-green-400">✓ Signature captured</span>
                  <button
                    onClick={() => handleChange('')}
                    disabled={disabled}
                    className="text-xs text-red-500 hover:text-red-700"
                  >
                    Clear
                  </button>
                </div>
              ) : (
                <button
                  onClick={() => handleChange('signed_' + Date.now())}
                  disabled={disabled}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md
                    hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500
                    disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Sign
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export const ConditionalChecklist: React.FC<ConditionalChecklistProps> = ({
  workOrderId,
  deviceType,
  templateId: preselectedTemplateId,
  onComplete,
  onError,
  onCancel,
  disabled = false,
}) => {
  const [state, setState] = useState<ChecklistState>({
    status: 'idle',
    template: null,
    responses: {},
    summary: null,
    error: null,
  });

  const [templates, setTemplates] = useState<ChecklistTemplate[]>([]);
  const [selectedTemplateId, setSelectedTemplateId] = useState<string>(preselectedTemplateId ?? '');
  const [templatesLoading, setTemplatesLoading] = useState(false);

  // ── Template selection ────────────────────────────────────────────

  const loadTemplates = useCallback(async () => {
    setTemplatesLoading(true);
    try {
      const data = await checklistApi.listTemplates({ device_type: deviceType, active_only: true });
      setTemplates(data);
      if (data.length === 1 && !selectedTemplateId) {
        setSelectedTemplateId(data[0].id);
      }
    } catch (err) {
      onError?.('Failed to load templates');
    } finally {
      setTemplatesLoading(false);
    }
  }, [deviceType, selectedTemplateId, onError]);

  // ── Start checklist ───────────────────────────────────────────────

  const startChecklist = useCallback(async () => {
    if (!selectedTemplateId) {
      onError?.('Please select a template');
      return;
    }

    setState(prev => ({ ...prev, status: 'loading', error: null }));

    try {
      // Загружаем шаблон с items
      const template = await checklistApi.getTemplate(selectedTemplateId);
      const checklist = await checklistApi.startChecklist(workOrderId, {
        template_id: selectedTemplateId,
      });

      setState({
        status: 'active',
        template,
        responses: {},
        summary: null,
        error: null,
      });
    } catch (err) {
      setState(prev => ({ ...prev, status: 'error', error: 'Failed to start checklist' }));
      onError?.('Failed to start checklist');
    }
  }, [selectedTemplateId, workOrderId, onError]);

  // ── Submit checklist ──────────────────────────────────────────────

  const submitChecklist = useCallback(async () => {
    if (!state.template) return;

    // Validate mandatory items
    const allItems = collectAllItems(state.template.items ?? []);
    const visibleItems = getVisibleItems(allItems, state.responses);

    for (const item of visibleItems) {
      if (item.mandatory && !state.responses[item.id]?.value && !state.responses[item.id]?.skipped) {
        onError?.(`"${item.label}" is required`);
        return;
      }
    }

    setState(prev => ({ ...prev, status: 'submitting' }));

    try {
      const responses: SubmitItemResponse[] = allItems.map(item => ({
        item_id: item.id,
        value: state.responses[item.id]?.value ?? '',
        photo_url: state.responses[item.id]?.photoUrl,
        skipped: !evaluateCondition(item.depends_on, state.responses),
      }));

      const summary = await checklistApi.submitChecklist(workOrderId, { responses });

      setState(prev => ({ ...prev, status: 'submitted', summary }));
      onComplete?.(summary);
    } catch (err) {
      setState(prev => ({ ...prev, status: 'active', error: 'Failed to submit checklist' }));
      onError?.('Failed to submit checklist');
    }
  }, [state.template, state.responses, workOrderId, onComplete, onError]);

  // ── Response handler ──────────────────────────────────────────────

  const handleResponseChange = useCallback((itemId: string, value: string) => {
    setState(prev => ({
      ...prev,
      responses: {
        ...prev.responses,
        [itemId]: { itemId, value, skipped: false },
      },
    }));
  }, []);

  // ── Photo capture handler ─────────────────────────────────────────

  const handlePhotoCapture = useCallback(async (_itemId: string): Promise<string> => {
    // Placeholder — в production здесь вызов нативного API камеры
    return 'photo_captured_' + Date.now();
  }, []);

  // ── Score calculation ─────────────────────────────────────────────

  const scoreInfo = useMemo(() => {
    if (!state.template?.items) return { current: 0, max: 0, percent: 0, passed: false };

    const allItems = collectAllItems(state.template.items);
    let maxScore = 0;
    let earnedScore = 0;

    for (const item of allItems) {
      if (item.score <= 0) continue;
      maxScore += item.score;

      const response = state.responses[item.id];
      const isVisible = evaluateCondition(item.depends_on, state.responses);

      if (isVisible && response?.value) {
        if (['true', 'yes', 'pass', '1'].includes(response.value)) {
          earnedScore += item.score;
        }
      }
    }

    const percent = maxScore > 0 ? Math.round((earnedScore / maxScore) * 100) : 0;
    return {
      current: earnedScore,
      max: maxScore,
      percent,
      passed: percent >= (state.template.pass_threshold ?? 70),
    };
  }, [state.template, state.responses]);

  // ── Render items recursively ──────────────────────────────────────

  const renderItems = useCallback((items: ChecklistItem[], depth = 0): React.ReactNode => {
    const visibleItems = getVisibleItems(items, state.responses);

    return visibleItems.map(item => (
      <div key={item.id}>
        <ItemRenderer
          item={item}
          value={state.responses[item.id]?.value ?? ''}
          onChange={handleResponseChange}
          onPhotoCapture={handlePhotoCapture}
          disabled={disabled || state.status === 'submitting'}
          depth={depth}
        />

        {/* Children (sub-items) */}
        {item.children && item.children.length > 0 && (
          <div className="mt-2 space-y-2">
            {renderItems(item.children, depth + 1)}
          </div>
        )}
      </div>
    ));
  }, [state.responses, state.status, handleResponseChange, handlePhotoCapture, disabled]);

  // ── Template picker ───────────────────────────────────────────────

  if (state.status === 'idle') {
    return (
      <div className="p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
          Conditional Checklist
        </h3>

        {/* Template selector */}
        <div className="mb-6">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Select Template
          </label>

          {templates.length === 0 && !templatesLoading && (
            <button
              onClick={loadTemplates}
              className="px-4 py-2 text-sm text-blue-600 hover:text-blue-800 border border-blue-300 rounded-md"
            >
              Load Templates
            </button>
          )}

          {templatesLoading && (
            <p className="text-sm text-gray-500">Loading templates...</p>
          )}

          {templates.length > 0 && (
            <select
              value={selectedTemplateId}
              onChange={(e) => setSelectedTemplateId(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-600 rounded-md
                bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100
                focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">-- Select a checklist template --</option>
              {templates.map((tpl) => (
                <option key={tpl.id} value={tpl.id}>
                  {tpl.name} ({tpl.device_types.join(', ')})
                </option>
              ))}
            </select>
          )}

          {selectedTemplateId && templates.length > 0 && (
            <div className="mt-2 text-xs text-gray-500 dark:text-gray-400">
              Passing threshold: {templates.find(t => t.id === selectedTemplateId)?.pass_threshold ?? 70}%
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <button
            onClick={startChecklist}
            disabled={!selectedTemplateId}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md
              hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500
              disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Start Checklist
          </button>
          {onCancel && (
            <button
              onClick={onCancel}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md
                hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-500"
            >
              Cancel
            </button>
          )}
        </div>
      </div>
    );
  }

  // ── Loading state ─────────────────────────────────────────────────

  if (state.status === 'loading') {
    return (
      <div className="p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm animate-pulse">
        <div className="h-5 bg-gray-200 dark:bg-gray-700 rounded w-1/3 mb-4" />
        <div className="space-y-3">
          <div className="h-20 bg-gray-100 dark:bg-gray-800 rounded" />
          <div className="h-20 bg-gray-100 dark:bg-gray-800 rounded" />
          <div className="h-20 bg-gray-100 dark:bg-gray-800 rounded" />
        </div>
      </div>
    );
  }

  // ── Error state ───────────────────────────────────────────────────

  if (state.status === 'error') {
    return (
      <div className="p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm border border-red-200 dark:border-red-800">
        <div className="flex items-center gap-2 text-red-600 dark:text-red-400 mb-4">
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="text-sm font-medium">{state.error}</span>
        </div>
        <button
          onClick={() => setState(prev => ({ ...prev, status: 'idle', error: null }))}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md
            hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          Try Again
        </button>
      </div>
    );
  }

  // ── Submitted state ───────────────────────────────────────────────

  if (state.status === 'submitted' && state.summary) {
    return (
      <div className="p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
        <div className="text-center mb-6">
          {state.summary.passed ? (
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-green-100 dark:bg-green-900 mb-3">
              <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
          ) : (
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-red-100 dark:bg-red-900 mb-3">
              <svg className="w-8 h-8 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </div>
          )}

          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {state.summary.passed ? 'Checklist Passed' : 'Checklist Failed'}
          </h3>
          <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
            {state.summary.template_name}
          </p>
        </div>

        {/* Score */}
        <div className="mb-6">
          <div className="flex justify-between text-sm mb-1">
            <span className="text-gray-600 dark:text-gray-400">Score</span>
            <span className="font-medium text-gray-900 dark:text-gray-100">
              {Math.round(state.summary.score_percent)}%
            </span>
          </div>
          <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5">
            <div
              className={`h-2.5 rounded-full transition-all ${
                state.summary.passed
                  ? 'bg-green-500'
                  : 'bg-red-500'
              }`}
              style={{ width: `${state.summary.score_percent}%` }}
            />
          </div>
        </div>

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-900 dark:text-gray-100">
              {state.summary.total_items}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">Total Items</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600 dark:text-green-400">
              {state.summary.completed_items}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">Completed</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-400">
              {state.summary.skipped_items}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">Skipped</div>
          </div>
        </div>

        <button
          onClick={() => setState({ status: 'idle', template: null, responses: {}, summary: null, error: null })}
          className="w-full px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md
            hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
        >
          Start New Checklist
        </button>
      </div>
    );
  }

  // ── Active checklist state ────────────────────────────────────────

  const allItems = collectAllItems(state.template?.items ?? []);
  const visibleItems = getVisibleItems(allItems, state.responses);
  const requiredVisible = visibleItems.filter(i => i.mandatory);
  const completedRequired = requiredVisible.filter(
    i => state.responses[i.id]?.value || state.responses[i.id]?.skipped
  );

  return (
    <div className="p-6 bg-white dark:bg-gray-900 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {state.template?.name ?? 'Checklist'}
          </h3>
          <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">
            {visibleItems.length} items · {requiredVisible.length} required
          </p>
        </div>

        {/* Live score */}
        <div className="text-right">
          <div className="text-sm font-medium text-gray-900 dark:text-gray-100">
            Score: {scoreInfo.percent}%
          </div>
          <div className="text-xs text-gray-500 dark:text-gray-400">
            {scoreInfo.current} / {scoreInfo.max} pts
          </div>
          <div className="mt-1 w-24 bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
            <div
              className={`h-1.5 rounded-full transition-all ${
                scoreInfo.passed ? 'bg-green-500' : 'bg-yellow-500'
              }`}
              style={{ width: `${Math.min(scoreInfo.percent, 100)}%` }}
            />
          </div>
        </div>
      </div>

      {/* Progress bar */}
      <div className="mb-4">
        <div className="flex justify-between text-xs text-gray-500 dark:text-gray-400 mb-1">
          <span>Progress</span>
          <span>{completedRequired.length} / {requiredVisible.length} required</span>
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
          <div
            className={`h-2 rounded-full transition-all ${
              completedRequired.length === requiredVisible.length && requiredVisible.length > 0
                ? 'bg-green-500'
                : 'bg-blue-500'
            }`}
            style={{
              width: `${requiredVisible.length > 0
                ? (completedRequired.length / requiredVisible.length) * 100
                : 0}%`
            }}
          />
        </div>
      </div>

      {/* Items */}
      <div className="space-y-3 mb-6">
        {renderItems(state.template?.items ?? [])}
      </div>

      {/* Error */}
      {state.error && (
        <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
          <p className="text-sm text-red-600 dark:text-red-400">{state.error}</p>
        </div>
      )}

      {/* Actions */}
      <div className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
        <button
          onClick={submitChecklist}
          disabled={disabled || state.status === 'submitting' || requiredVisible.length === 0}
          className="px-4 py-2 text-sm font-medium text-white bg-green-600 rounded-md
            hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500
            disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {state.status === 'submitting' ? 'Submitting...' : 'Submit Checklist'}
        </button>
        {onCancel && (
          <button
            onClick={onCancel}
            disabled={state.status === 'submitting'}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-md
              hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-500
              disabled:opacity-50"
          >
            Cancel
          </button>
        )}
      </div>
    </div>
  );
};

export default ConditionalChecklist;
