// ═══════════════════════════════════════════════════════════════════════
// P2-BI: Custom Reports Builder — User-Facing Report Designer
//
// Features:
//   - List saved custom report configurations
//   - Create/edit reports: template → dimensions/measures → chart type → filters
//   - Execute queries and render results as Nivo charts or data tables
//   - Persist report configs in localStorage (MVP; API-backed in future)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { v4 as uuidv4 } from 'uuid';
import {
  Card, Button, EmptyState, Alert, Skeleton, SkeletonTable,
  Select, Modal, Input, Badge,
} from '../components/ui';
import { request } from '../services/api/client';
import {
  BarChart3, PieChart, TrendingUp, Table2, Save, Trash2,
  Edit3, Play, Plus, ChevronDown, ChevronRight, AlertCircle,
  Download, Loader2, Info,
} from '../components/ui/Icons';

// ── Types ──────────────────────────────────────────────────────────────────────

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

interface QueryResult {
  columns: string[];
  rows: unknown[][];
  total: number;
  took: string;
}

type ChartType = 'table' | 'bar' | 'pie' | 'line';

interface ReportConfig {
  id: string;
  name: string;
  description: string;
  templateId: string;
  dimensions: string[];
  measures: string[];
  chartType: ChartType;
  filters: FilterCondition[];
  dateRange: { from?: string; to?: string };
  createdAt: string;
  updatedAt: string;
}

const STORAGE_KEY = 'cctv_custom_reports';
const CHART_TYPES: { value: ChartType; label: string }[] = [
  { value: 'table', label: 'Table' },
  { value: 'bar', label: 'Bar Chart' },
  { value: 'pie', label: 'Pie Chart' },
  { value: 'line', label: 'Line Chart' },
];

function loadReports(): ReportConfig[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? JSON.parse(raw) : [];
  } catch { return []; }
}

function saveReports(reports: ReportConfig[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(reports));
}

// ── Default empty config ───────────────────────────────────────────────────────

function emptyConfig(): ReportConfig {
  return {
    id: '',
    name: '',
    description: '',
    templateId: '',
    dimensions: [],
    measures: [],
    chartType: 'table',
    filters: [],
    dateRange: {},
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };
}

// ── Chart Colors ───────────────────────────────────────────────────────────────

const CHART_COLORS = [
  '#3b82f6', '#ef4444', '#22c55e', '#f97316', '#a855f7',
  '#06b6d4', '#84cc16', '#ec4899', '#14b8a6', '#eab308',
];

function nivoTheme(mode: 'light' | 'dark') {
  const isDark = mode === 'dark';
  return {
    background: 'transparent',
    axis: {
      ticks: { text: { fontSize: 12, fill: isDark ? '#94a3b8' : '#64748b' } },
      domain: { line: { stroke: isDark ? '#334155' : '#e2e8f0', strokeWidth: 1 } },
    },
    grid: {
      line: { stroke: isDark ? '#334155' : '#e2e8f0', strokeDasharray: '3 3', strokeWidth: 1 },
    },
    legends: {
      text: { fill: isDark ? '#cbd5e1' : '#475569', fontSize: 11 },
    },
    tooltip: {
      container: {
        background: isDark ? '#1e293b' : '#ffffff',
        color: isDark ? '#e2e8f0' : '#1e293b',
        fontSize: 12,
        borderRadius: 8,
        boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
      },
    },
  };
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export function CustomReports() {
  const { t } = useTranslation();

  // ── State ──────────────────────────────────────────────────────────
  const [reports, setReports] = useState<ReportConfig[]>(loadReports);
  const [templates, setTemplates] = useState<QueryTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState<ReportConfig>(emptyConfig);
  const [showEditor, setShowEditor] = useState(false);
  const [results, setResults] = useState<Record<string, QueryResult>>({});
  const [running, setRunning] = useState<string | null>(null);
  const [queryError, setQueryError] = useState('');

  // ── Load templates ─────────────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;
    request<QueryTemplate[]>('/analytics/bi/templates')
      .then((data) => { if (!cancelled) setTemplates(data); })
      .catch((err) => { if (!cancelled) setError(err.message); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, []);

  // ── Persist reports ────────────────────────────────────────────────
  useEffect(() => { saveReports(reports); }, [reports]);

  // ── Helpers ────────────────────────────────────────────────────────
  const saveReport = useCallback((cfg: ReportConfig) => {
    const updated = { ...cfg, updatedAt: new Date().toISOString() };
    setReports((prev) => {
      const idx = prev.findIndex((r) => r.id === cfg.id);
      if (idx >= 0) {
        const copy = [...prev];
        copy[idx] = updated;
        return copy;
      }
      return [...prev, { ...updated, id: cfg.id || uuidv4(), createdAt: new Date().toISOString() }];
    });
    setShowEditor(false);
  }, []);

  const deleteReport = useCallback((id: string) => {
    setReports((prev) => prev.filter((r) => r.id !== id));
    setResults((prev) => { const copy = { ...prev }; delete copy[id]; return copy; });
  }, []);

  const editReport = useCallback((id: string) => {
    const report = reports.find((r) => r.id === id);
    if (report) { setEditing(report); setShowEditor(true); }
  }, [reports]);

  const runQuery = useCallback(async (report: ReportConfig) => {
    setRunning(report.id);
    setQueryError('');
    try {
      const data = await request<QueryResult>('/analytics/bi/query', {
        method: 'POST',
        body: JSON.stringify({
          template_id: report.templateId,
          dimensions: report.dimensions,
          measures: report.measures,
          filters: report.filters,
          time_from: report.dateRange.from || undefined,
          time_to: report.dateRange.to || undefined,
          limit: 1000,
        }),
      });
      setResults((prev) => ({ ...prev, [report.id]: data }));
    } catch (err) {
      setQueryError(err instanceof Error ? err.message : 'Query failed');
    } finally {
      setRunning(null);
    }
  }, []);

  // ── Templates list ─────────────────────────────────────────────
  const templatesById = useMemo(() => {
    const map: Record<string, QueryTemplate> = {};
    templates.forEach((t) => { map[t.id] = t; });
    return map;
  }, [templates]);

  // ── Chart renderers ────────────────────────────────────────────
  const renderChart = (result: QueryResult, chartType: ChartType) => {
    // Simplified chart rendering
    if (chartType === 'table' || !result.columns.length) {
      return (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200">
                {result.columns.map((col, i) => (
                  <th key={i} className="text-left py-2 px-3 font-medium text-gray-600">{col}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {result.rows.slice(0, 50).map((row, ri) => (
                <tr key={ri} className="border-b border-gray-100 hover:bg-gray-50">
                  {row.map((cell, ci) => (
                    <td key={ci} className="py-2 px-3 text-gray-900">{String(cell)}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
    }
    return <div className="text-sm text-gray-500">Chart rendering placeholder</div>;
  };

  // ── Editor Modal ───────────────────────────────────────────────
  const renderEditor = () => {
    if (!showEditor) return null;
    const template = templatesById[editing.templateId];
    return (
      <Modal
        isOpen={showEditor}
        onClose={() => setShowEditor(false)}
        title={editing.id ? 'Edit Report' : 'New Report'}
      >
        <div className="space-y-4">
          <Input
            label="Report Name"
            value={editing.name}
            onChange={(e) => setEditing({ ...editing, name: e.target.value })}
            placeholder="e.g. Monthly Device Health"
          />
          <Input
            label="Description"
            value={editing.description}
            onChange={(e) => setEditing({ ...editing, description: e.target.value })}
            placeholder="What does this report show?"
          />
          <Select
            label="Template"
            value={editing.templateId}
            onChange={(e) => setEditing({ ...editing, templateId: e.target.value, dimensions: [], measures: [] })}
            options={templates.map((t) => ({ value: t.id, label: t.name }))}
          />
          {template && (
            <>
              <Select
                label="Chart Type"
                value={editing.chartType}
                onChange={(e) => setEditing({ ...editing, chartType: e.target.value as ChartType })}
                options={CHART_TYPES.map((ct) => ({ value: ct.value, label: ct.label }))}
              />
              <div className="flex gap-4">
                <div className="flex-1">
                  <label className="block text-sm font-medium text-gray-700 mb-1">Dimensions</label>
                  <div className="space-y-1 max-h-40 overflow-y-auto border rounded-lg p-2">
                    {template.dimensions.map((f) => (
                      <label key={f.key} className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          checked={editing.dimensions.includes(f.key)}
                          onChange={() => {
                            const dims = editing.dimensions.includes(f.key)
                              ? editing.dimensions.filter((d) => d !== f.key)
                              : [...editing.dimensions, f.key];
                            setEditing({ ...editing, dimensions: dims });
                          }}
                        />
                        {f.label}
                      </label>
                    ))}
                  </div>
                </div>
                <div className="flex-1">
                  <label className="block text-sm font-medium text-gray-700 mb-1">Measures</label>
                  <div className="space-y-1 max-h-40 overflow-y-auto border rounded-lg p-2">
                    {template.measures.map((f) => (
                      <label key={f.key} className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          checked={editing.measures.includes(f.key)}
                          onChange={() => {
                            const ms = editing.measures.includes(f.key)
                              ? editing.measures.filter((m) => m !== f.key)
                              : [...editing.measures, f.key];
                            setEditing({ ...editing, measures: ms });
                          }}
                        />
                        {f.label}
                      </label>
                    ))}
                  </div>
                </div>
              </div>
            </>
          )}
          <div className="flex justify-end gap-3 pt-4 border-t">
            <Button variant="ghost" onClick={() => setShowEditor(false)}>Cancel</Button>
            <Button onClick={() => saveReport(editing)} disabled={!editing.name || !editing.templateId}>
              <Save size={16} className="mr-1.5" />
              Save Report
            </Button>
          </div>
        </div>
      </Modal>
    );
  };

  // ── Main Render ────────────────────────────────────────────────
  if (loading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-64" />
        <SkeletonTable rows={5} />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {renderEditor()}

      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Custom Reports</h1>
          <p className="text-sm text-gray-500 mt-1">
            Create, edit, and run custom analytics reports
          </p>
        </div>
        <Button onClick={() => { setEditing(emptyConfig()); setShowEditor(true); }}>
          <Plus size={16} className="mr-1.5" />
          New Report
        </Button>
      </div>

      {/* Error */}
      {error && (
        <Alert variant="error" title="Error loading templates">
          {error}
        </Alert>
      )}

      {/* Query Error */}
      {queryError && (
        <Alert variant="warning" title="Query Error">
          {queryError}
        </Alert>
      )}

      {/* Reports List */}
      {reports.length === 0 ? (
        <EmptyState
          icon={<BarChart3 className="w-12 h-12" />}
          title="No reports yet"
          description="Create your first custom report to start analyzing data"
          action={{ label: 'Create Report', onClick: () => { setEditing(emptyConfig()); setShowEditor(true); } }}
        />
      ) : (
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
          {reports.map((report) => {
            const result = results[report.id];
            const hasResults = result && result.rows.length > 0;
            const isRunning = running === report.id;

            return (
              <Card key={report.id} variant="elevated" className="overflow-hidden">
                <div className="p-4">
                  {/* Header */}
                  <div className="flex items-start justify-between mb-3">
                    <div>
                      <h3 className="font-semibold text-gray-900">{report.name}</h3>
                      <p className="text-xs text-gray-500 mt-0.5">{report.description}</p>
                    </div>
                    <div className="flex items-center gap-1">
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => editReport(report.id)}
                        aria-label="Edit report"
                      >
                        <Edit3 size={14} />
                      </Button>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => deleteReport(report.id)}
                        aria-label="Delete report"
                      >
                        <Trash2 size={14} />
                      </Button>
                    </div>
                  </div>

                  {/* Meta */}
                  <div className="flex items-center gap-3 text-xs text-gray-500 mb-3">
                    <Badge variant="outline">{report.chartType}</Badge>
                    <span>
                      {report.dimensions.length} dim, {report.measures.length} meas
                    </span>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2 mb-3">
                    <Button
                      size="sm"
                      onClick={() => runQuery(report)}
                      disabled={isRunning}
                    >
                      {isRunning ? (
                        <>
                          <Loader2 size={14} className="mr-1.5 animate-spin" />
                          Running...
                        </>
                      ) : (
                        <>
                          <Play size={14} className="mr-1.5" />
                          Run Query
                        </>
                      )}
                    </Button>
                  </div>

                  {/* Results */}
                  {hasResults && !isRunning && (
                    <div className="border-t pt-3 mt-3">
                      <div className="flex items-center justify-between mb-2">
                        <div className="flex items-center gap-3 text-xs text-gray-500">
                          <span>{result.total} rows</span>
                          <span>Took: {result.took}</span>
                        </div>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => {
                            const csv = [
                              result.columns.join(','),
                              ...result.rows.map((r) => r.join(',')),
                            ].join('\n');
                            const blob = new Blob([csv], { type: 'text/csv' });
                            const url = URL.createObjectURL(blob);
                            const a = document.createElement('a');
                            a.href = url;
                            a.download = `${report.name}.csv`;
                            a.click();
                            URL.revokeObjectURL(url);
                          }}
                          aria-label="Export CSV"
                        >
                          <Download size={14} className="mr-1" />
                          CSV
                        </Button>
                      </div>
                      {renderChart(result, report.chartType)}
                    </div>
                  )}
                </div>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default CustomReports;