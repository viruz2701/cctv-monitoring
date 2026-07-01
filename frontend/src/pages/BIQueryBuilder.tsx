// ═══════════════════════════════════════════════════════════════════════
// P2-BI: Self-Service Analytics Query Builder
//
// Features:
//   - Template selection from BI templates catalog
//   - Dimension & measure selection for custom queries
//   - Date range filtering via DateRangePicker
//   - Query execution with results table
//
// Data sources:
//   - GET /api/v1/analytics/bi/templates → QueryTemplate[]
//   - POST /api/v1/analytics/bi/query → QueryResult
//
// Compliance:
//   - OWASP ASVS V2.1.1 (Input validation via Zod — не применимо, read-only query)
//   - IEC 62443 SR 3.1 (RBAC — через RoleProtectedRoute)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Button,
  Select,
  Pagination,
  EmptyState,
  Alert,
  Skeleton,
  SkeletonTable,
} from '../components/ui';
import { DateRangePicker, type DateRange } from '../components/molecules/DateRangePicker';
import { request } from '../services/api/client';
import {
  BarChart3,
  Filter,
  Play,
  Download,
  Table2,
  AlertCircle,
  Loader2,
  ChevronDown,
  ChevronRight,
  Info,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface Field {
  key: string;
  label: string;
  type: string;
  agg?: string;
  sql_expr?: string;
}

interface QueryTemplate {
  id: string;
  name: string;
  description: string;
  dimensions: Field[];
  measures: Field[];
  date_field: string;
}

interface FilterCondition {
  field: string;
  op: string;
  value: unknown;
}

interface QueryParams {
  template_id: string;
  dimensions?: string[];
  measures?: string[];
  filters?: FilterCondition[];
  time_from?: string;
  time_to?: string;
  limit?: number;
  offset?: number;
  order_by?: string;
  order_dir?: string;
}

interface QueryResult {
  columns: string[];
  rows: unknown[][];
  total: number;
  took: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const PAGE_SIZE = 25;

const FIELD_TYPE_ICONS: Record<string, string> = {
  string: 'Aa',
  number: '#',
  date: '📅',
  boolean: '✓',
  enum: '☰',
};

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function formatCellValue(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'number') return value.toLocaleString();
  if (typeof value === 'boolean') return value ? '✓' : '✗';
  if (value instanceof Date) return value.toLocaleDateString();
  return String(value);
}

function formatTook(took: string): string {
  const ms = parseInt(took, 10);
  if (Number.isNaN(ms)) return took;
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

interface FieldCheckboxListProps {
  fields: Field[];
  selected: string[];
  onChange: (keys: string[]) => void;
  label: string;
  icon: React.ReactNode;
}

function FieldCheckboxList({ fields, selected, onChange, label, icon }: FieldCheckboxListProps) {
  const [expanded, setExpanded] = useState(true);

  const toggleField = useCallback(
    (key: string) => {
      if (selected.includes(key)) {
        onChange(selected.filter((k) => k !== key));
      } else {
        onChange([...selected, key]);
      }
    },
    [selected, onChange],
  );

  const toggleAll = useCallback(() => {
    if (selected.length === fields.length) {
      onChange([]);
    } else {
      onChange(fields.map((f) => f.key));
    }
  }, [fields, selected, onChange]);

  if (fields.length === 0) {
    return (
      <div className="text-sm text-slate-400 dark:text-slate-500 italic px-1">
        {label} не доступны
      </div>
    );
  }

  return (
    <div className="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
      {/* Header */}
      <button
        type="button"
        onClick={() => setExpanded((p) => !p)}
        className="w-full flex items-center justify-between px-3 py-2.5 bg-slate-50 dark:bg-slate-800/50 text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700/50 transition-colors"
      >
        <span className="flex items-center gap-2">
          {icon}
          {label}
          <span className="text-xs text-slate-400 dark:text-slate-500 font-normal">
            ({selected.length}/{fields.length})
          </span>
        </span>
        {expanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
      </button>

      {/* Body */}
      {expanded && (
        <div className="px-3 py-2 space-y-1">
          {/* Toggle all */}
          <label className="flex items-center gap-2 py-1 cursor-pointer group">
            <input
              type="checkbox"
              checked={selected.length === fields.length && fields.length > 0}
              onChange={toggleAll}
              className="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 cursor-pointer"
            />
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-300 transition-colors">
              Выбрать все
            </span>
          </label>

          <div className="border-t border-slate-100 dark:border-slate-700/50 pt-1" />

          {/* Fields */}
          {fields.map((field) => (
            <label
              key={field.key}
              className="flex items-center gap-2 py-1.5 px-1 rounded-md hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer group transition-colors"
            >
              <input
                type="checkbox"
                checked={selected.includes(field.key)}
                onChange={() => toggleField(field.key)}
                className="w-4 h-4 rounded border-slate-300 dark:border-slate-600 text-blue-600 focus:ring-blue-500 cursor-pointer"
              />
              <span className="flex items-center gap-1.5 min-w-0 flex-1">
                <span className="text-[10px] font-mono text-slate-400 dark:text-slate-500 uppercase w-5 text-center flex-shrink-0">
                  {FIELD_TYPE_ICONS[field.type] || '?'}
                </span>
                <span className="text-sm text-slate-700 dark:text-slate-300 truncate group-hover:text-slate-900 dark:group-hover:text-white transition-colors">
                  {field.label}
                </span>
                {field.agg && (
                  <span className="text-[10px] font-mono text-blue-500 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 px-1.5 py-0.5 rounded flex-shrink-0">
                    {field.agg}
                  </span>
                )}
              </span>
            </label>
          ))}
        </div>
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Template Info Card
// ═══════════════════════════════════════════════════════════════════════

function TemplateInfo({ template }: { template: QueryTemplate }) {
  return (
    <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-4 space-y-2">
      <div className="flex items-start gap-2">
        <Info size={16} className="text-blue-500 mt-0.5 flex-shrink-0" />
        <div>
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
            {template.name}
          </h3>
          <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
            {template.description}
          </p>
        </div>
      </div>
      <div className="flex flex-wrap gap-3 text-xs text-slate-500 dark:text-slate-400">
        <span>
          Dimensions:{' '}
          <span className="font-medium text-slate-700 dark:text-slate-300">
            {template.dimensions.length}
          </span>
        </span>
        <span>
          Measures:{' '}
          <span className="font-medium text-slate-700 dark:text-slate-300">
            {template.measures.length}
          </span>
        </span>
        <span>
          Date field:{' '}
          <code className="font-mono text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/20 px-1 rounded">
            {template.date_field}
          </code>
        </span>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Results Table
// ═══════════════════════════════════════════════════════════════════════

function ResultsTable({ result }: { result: QueryResult }) {
  const [page, setPage] = useState(1);
  const totalPages = Math.ceil(result.total / PAGE_SIZE);

  const paginatedRows = result.rows.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  return (
    <div className="space-y-3">
      {/* Info bar */}
      <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
        <span className="flex items-center gap-1">
          <Table2 size={14} />
          {result.total.toLocaleString()} rows · took {formatTook(result.took)}
        </span>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => {
            // CSV export
            const header = result.columns.join(',');
            const body = result.rows.map((r) => r.map(formatCellValue).join(',')).join('\n');
            const blob = new Blob([`${header}\n${body}`], { type: 'text/csv;charset=utf-8;' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `bi-query-${Date.now()}.csv`;
            a.click();
            URL.revokeObjectURL(url);
          }}
        >
          <Download size={14} />
          CSV
        </Button>
      </div>

      {/* Table */}
      <div className="overflow-x-auto rounded-lg border border-slate-200 dark:border-slate-700">
        <table className="w-full text-sm">
          <thead>
            <tr className="bg-slate-50 dark:bg-slate-800/50 border-b border-slate-200 dark:border-slate-700">
              {result.columns.map((col) => (
                <th
                  key={col}
                  className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider whitespace-nowrap"
                >
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 dark:divide-slate-700/50">
            {paginatedRows.length === 0 ? (
              <tr>
                <td
                  colSpan={result.columns.length}
                  className="px-4 py-12 text-center text-slate-500 dark:text-slate-400"
                >
                  No results
                </td>
              </tr>
            ) : (
              paginatedRows.map((row, ri) => (
                <tr
                  key={ri}
                  className="hover:bg-slate-50 dark:hover:bg-slate-800/30 transition-colors"
                >
                  {row.map((cell, ci) => (
                    <td
                      key={`${ri}-${ci}`}
                      className="px-4 py-2.5 text-sm text-slate-700 dark:text-slate-300 whitespace-nowrap"
                    >
                      {formatCellValue(cell)}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <Pagination
          currentPage={page}
          totalPages={totalPages}
          onPageChange={setPage}
          totalItems={result.total}
          itemsPerPage={PAGE_SIZE}
        />
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export function BIQueryBuilder() {
  const { t } = useTranslation();

  // ── State ──────────────────────────────────────────────────────────
  const [templates, setTemplates] = useState<QueryTemplate[]>([]);
  const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');
  const [selectedDimensions, setSelectedDimensions] = useState<string[]>([]);
  const [selectedMeasures, setSelectedMeasures] = useState<string[]>([]);
  const [dateRange, setDateRange] = useState<DateRange>({
    start: new Date(Date.now() - 6 * 24 * 60 * 60 * 1000),
    end: new Date(),
    preset: 'last7',
  });
  const [result, setResult] = useState<QueryResult | null>(null);
  const [executing, setExecuting] = useState(false);
  const [loadingTemplates, setLoadingTemplates] = useState(true);
  const [error, setError] = useState('');
  const [validationError, setValidationError] = useState('');

  // Derived: current template
  const selectedTemplate = templates.find((t) => t.id === selectedTemplateId) || null;

  // ── Fetch templates ────────────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;

    async function fetchTemplates() {
      try {
        setLoadingTemplates(true);
        const data = await request<QueryTemplate[]>('/analytics/bi/templates');
        if (!cancelled) {
          setTemplates(data || []);
        }
      } catch (err: unknown) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load BI templates');
        }
      } finally {
        if (!cancelled) setLoadingTemplates(false);
      }
    }

    fetchTemplates();
    return () => {
      cancelled = true;
    };
  }, []);

  // ── Reset selections when template changes ─────────────────────────
  useEffect(() => {
    if (selectedTemplate) {
      setSelectedDimensions([]);
      setSelectedMeasures([]);
      setValidationError('');
    }
  }, [selectedTemplateId]);

  // ── Execute query ──────────────────────────────────────────────────
  const executeQuery = useCallback(async () => {
    if (!selectedTemplate) {
      setValidationError('Please select a template first');
      return;
    }

    if (selectedDimensions.length === 0 && selectedMeasures.length === 0) {
      setValidationError('Select at least one dimension or measure');
      return;
    }

    setValidationError('');
    setExecuting(true);
    setError('');

    try {
      const params: QueryParams = {
        template_id: selectedTemplate.id,
        dimensions: selectedDimensions.length > 0 ? selectedDimensions : undefined,
        measures: selectedMeasures.length > 0 ? selectedMeasures : undefined,
        time_from: dateRange.start.toISOString(),
        time_to: dateRange.end.toISOString(),
        limit: 1000,
      };

      const data = await request<QueryResult>('/analytics/bi/query', {
        method: 'POST',
        body: JSON.stringify(params),
      });

      setResult(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Query execution failed');
    } finally {
      setExecuting(false);
    }
  }, [selectedTemplate, selectedDimensions, selectedMeasures, dateRange]);

  // ── Template select handler ────────────────────────────────────────
  const handleTemplateChange = useCallback((value: string) => {
    setSelectedTemplateId(value);
    setResult(null);
    setError('');
    setValidationError('');
  }, []);

  // ══════════════════════════════════════════════════════════════════
  // Render
  // ══════════════════════════════════════════════════════════════════

  const templateOptions = templates.map((tmpl) => ({
    value: tmpl.id,
    label: tmpl.name,
  }));

  const canExecute = selectedTemplate && (selectedDimensions.length > 0 || selectedMeasures.length > 0);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <BarChart3 className="w-6 h-6 text-blue-500" />
            {t('bi_query_builder', 'BI Query Builder')}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('bi_query_builder_desc', 'Self-service analytics — build and execute BI queries')}
          </p>
        </div>
      </div>

      {/* ══════════════════════════════════════════════════════════════
          Template Selection
          ══════════════════════════════════════════════════════════════ */}
      <Card>
        <div className="p-5 space-y-4">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-white flex items-center gap-2">
            <Filter size={16} className="text-slate-400" />
            {t('bi_template_selection', 'Template Selection')}
          </h2>

          <Select
            label={t('bi_template', 'BI Template')}
            options={[
              { value: '', label: t('select_template', 'Select a template...') },
              ...templateOptions,
            ]}
            value={selectedTemplateId}
            onChange={(e) => handleTemplateChange(e.target.value)}
            error={validationError && !selectedTemplate ? validationError : undefined}
          />

          {selectedTemplate && <TemplateInfo template={selectedTemplate} />}

          {loadingTemplates && (
            <div className="space-y-2">
              <Skeleton width="60%" height={14} />
              <Skeleton width="40%" height={14} />
            </div>
          )}
        </div>
      </Card>

      {/* ══════════════════════════════════════════════════════════════
          Query Configuration
          ══════════════════════════════════════════════════════════════ */}
      {selectedTemplate && (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Dimensions */}
          <Card>
            <div className="p-5">
              <FieldCheckboxList
                fields={selectedTemplate.dimensions}
                selected={selectedDimensions}
                onChange={setSelectedDimensions}
                label={t('bi_dimensions', 'Dimensions')}
                icon={<BarChart3 size={14} className="text-indigo-500" />}
              />
            </div>
          </Card>

          {/* Measures */}
          <Card>
            <div className="p-5">
              <FieldCheckboxList
                fields={selectedTemplate.measures}
                selected={selectedMeasures}
                onChange={setSelectedMeasures}
                label={t('bi_measures', 'Measures')}
                icon={<BarChart3 size={14} className="text-emerald-500" />}
              />
            </div>
          </Card>

          {/* Filters & Execute */}
          <div className="space-y-4">
            {/* Date Range */}
            <Card>
              <div className="p-5 space-y-3">
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                  {t('bi_date_range', 'Date Range')}
                </h3>
                <DateRangePicker value={dateRange} onChange={setDateRange} />
                <p className="text-[11px] text-slate-400 dark:text-slate-500">
                  {t('bi_date_field_hint', 'Applies to field:')}{' '}
                  <code className="font-mono text-blue-500">{selectedTemplate.date_field}</code>
                </p>
              </div>
            </Card>

            {/* Execute Button */}
            <Card>
              <div className="p-5 space-y-3">
                <Button
                  variant="primary"
                  size="lg"
                  className="w-full"
                  disabled={!canExecute || executing}
                  onClick={executeQuery}
                >
                  {executing ? (
                    <>
                      <Loader2 size={16} className="animate-spin" />
                      {t('bi_executing', 'Executing...')}
                    </>
                  ) : (
                    <>
                      <Play size={16} />
                      {t('bi_execute', 'Execute')}
                    </>
                  )}
                </Button>

                {validationError && (
                  <Alert variant="warning">
                    {validationError}
                  </Alert>
                )}
              </div>
            </Card>
          </div>
        </div>
      )}

      {/* ══════════════════════════════════════════════════════════════
          Error Alert
          ══════════════════════════════════════════════════════════════ */}
      {error && (
        <Alert variant="error">
          <div className="flex items-center gap-2">
            <AlertCircle size={16} />
            {error}
          </div>
        </Alert>
      )}

      {/* ══════════════════════════════════════════════════════════════
          Results
          ══════════════════════════════════════════════════════════════ */}
      {executing && (
        <Card>
          <div className="p-5">
            <SkeletonTable rows={8} columns={selectedTemplate?.dimensions.length ?? 3 + (selectedTemplate?.measures.length ?? 2)} />
          </div>
        </Card>
      )}

      {result && !executing && (
        <Card>
          <div className="p-5">
            <h2 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <Table2 size={16} className="text-slate-400" />
              {t('bi_results', 'Query Results')}
            </h2>
            <ResultsTable result={result} />
          </div>
        </Card>
      )}

      {!selectedTemplate && !loadingTemplates && !error && (
        <Card>
          <div className="p-8">
            <EmptyState
              icon={<BarChart3 className="w-12 h-12 text-slate-300 dark:text-slate-600" />}
              title={t('bi_select_template', 'Select a BI Template')}
              description={t('bi_select_template_desc', 'Choose a template above to start building your query')}
            />
          </div>
        </Card>
      )}
    </div>
  );
}
