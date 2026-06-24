import React, { useState, useCallback, useRef } from 'react';
import { Upload, FileText, CheckCircle, AlertTriangle, ArrowLeft, ArrowRight, X, Download, Columns } from 'lucide-react';
import { Button } from './Button';
import { Card, CardHeader, CardBody } from './Card';
import { Badge } from './Badge';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type WizardStep = 'upload' | 'preview' | 'match' | 'review' | 'importing' | 'complete';

interface ImportColumn {
  key: string;
  label: string;
  required?: boolean;
  type?: 'string' | 'number' | 'date' | 'email';
  validation?: (value: string) => string | null; // returns error message or null
}

interface ImportRow {
  index: number;
  data: Record<string, string>;
  errors: Record<string, string>;
  valid: boolean;
}

interface ImportWizardProps {
  title?: string;
  description?: string;
  columns: ImportColumn[];
  entityType: string; // "devices", "work_orders", "spare_parts", "sites"
  onImport: (rows: ImportRow[]) => Promise<{ success: number; errors: number }>;
  onClose: () => void;
  maxFileSizeMB?: number;
  sampleData?: Record<string, string>[];
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function ImportWizard({
  title = 'Import Data',
  description = 'Upload a CSV or Excel file to import data',
  columns,
  entityType,
  onImport,
  onClose,
  maxFileSizeMB = 20,
  sampleData,
}: ImportWizardProps) {
  const [step, setStep] = useState<WizardStep>('upload');
  const [file, setFile] = useState<File | null>(null);
  const [rows, setRows] = useState<ImportRow[]>([]);
  const [columnMapping, setColumnMapping] = useState<Record<string, string>>({});
  const [importResult, setImportResult] = useState<{ success: number; errors: number } | null>(null);
  const [importing, setImporting] = useState(false);
  const [dragOver, setDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Parse CSV
  const parseFile = useCallback(async (file: File) => {
    const text = await file.text();
    const lines = text.split('\n').filter(l => l.trim());
    if (lines.length < 2) {
      alert('File must have at least a header row and one data row');
      return;
    }

    const headers = parseCSVLine(lines[0]);
    const autoMapping: Record<string, string> = {};
    headers.forEach(h => {
      const match = columns.find(c =>
        c.label.toLowerCase() === h.toLowerCase() ||
        c.key.toLowerCase() === h.toLowerCase()
      );
      if (match) autoMapping[h] = match.key;
    });
    setColumnMapping(autoMapping);

    const parsedRows: ImportRow[] = [];
    for (let i = 1; i < lines.length; i++) {
      const values = parseCSVLine(lines[i]);
      const data: Record<string, string> = {};
      const errors: Record<string, string> = {};
      headers.forEach((h, idx) => {
        const colKey = autoMapping[h] || h;
        data[colKey] = values[idx] || '';
      });

      // Validate
      let valid = true;
      columns.forEach(col => {
        const val = data[col.key] || '';
        if (col.required && !val.trim()) {
          errors[col.key] = `${col.label} is required`;
          valid = false;
        }
        if (col.validation && val) {
          const err = col.validation(val);
          if (err) {
            errors[col.key] = err;
            valid = false;
          }
        }
      });

      parsedRows.push({ index: i, data, errors, valid });
    }

    setRows(parsedRows);
    setStep('preview');
  }, [columns]);

  const handleFileDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragOver(false);
    const f = e.dataTransfer.files[0];
    if (f) {
      setFile(f);
      parseFile(f);
    }
  }, [parseFile]);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (f) {
      setFile(f);
      parseFile(f);
    }
  }, [parseFile]);

  const handleImport = async () => {
    setStep('importing');
    setImporting(true);
    try {
      const result = await onImport(rows);
      setImportResult(result);
      setStep('complete');
    } catch (err) {
      alert('Import failed: ' + (err as Error).message);
    } finally {
      setImporting(false);
    }
  };

  const validRows = rows.filter(r => r.valid);
  const errorRows = rows.filter(r => !r.valid);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
      <div className="bg-white dark:bg-slate-800 rounded-2xl shadow-2xl w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 dark:border-slate-700">
          <div>
            <h2 className="text-lg font-bold text-slate-900 dark:text-white">{title}</h2>
            <p className="text-sm text-slate-500 dark:text-slate-400">{description}</p>
          </div>
          <button onClick={onClose} className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded">
            <X className="w-5 h-5 text-slate-500" />
          </button>
        </div>

        {/* Steps indicator */}
        <div className="flex items-center gap-2 px-6 py-3 bg-slate-50 dark:bg-slate-900/50 border-b border-slate-200 dark:border-slate-700">
          {(['upload', 'preview', 'match', 'review', 'complete'] as WizardStep[]).map((s, i) => {
            const isActive = step === s;
            const isDone = ['upload', 'preview', 'match', 'review'].indexOf(step) > i || step === 'complete' && i < 4;
            return (
              <div key={s} className="flex items-center gap-2">
                <div className={`w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold ${
                  isActive ? 'bg-blue-600 text-white' :
                  isDone ? 'bg-emerald-500 text-white' :
                  'bg-slate-200 dark:bg-slate-700 text-slate-500'
                }`}>
                  {isDone ? <CheckCircle className="w-3.5 h-3.5" /> : i + 1}
                </div>
                <span className={`text-xs capitalize ${isActive ? 'text-blue-600 font-medium' : 'text-slate-500'}`}>
                  {s === 'importing' ? 'Importing...' : s}
                </span>
                {i < 4 && <div className="w-6 h-px bg-slate-300 dark:bg-slate-600" />}
              </div>
            );
          })}
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto p-6">
          {step === 'upload' && (
            <div
              onDragOver={(e) => { e.preventDefault(); setDragOver(true); }}
              onDragLeave={() => setDragOver(false)}
              onDrop={handleFileDrop}
              className={`border-2 border-dashed rounded-xl p-12 text-center transition-all cursor-pointer ${
                dragOver
                  ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
                  : 'border-slate-300 dark:border-slate-600 hover:border-blue-400'
              }`}
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="w-12 h-12 mx-auto mb-4 text-slate-400" />
              <p className="text-lg font-medium text-slate-700 dark:text-slate-300 mb-2">
                Drop your file here or click to browse
              </p>
              <p className="text-sm text-slate-500 mb-4">
                Supports CSV, XLSX, JSON (max {maxFileSizeMB}MB)
              </p>
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,.xlsx,.xls,.json"
                onChange={handleFileSelect}
                className="hidden"
              />
              <Button variant="outline">Choose File</Button>

              {sampleData && (
                <div className="mt-6 pt-6 border-t border-slate-200 dark:border-slate-700">
                  <p className="text-xs font-medium text-slate-500 mb-2">Expected columns:</p>
                  <div className="inline-flex flex-wrap gap-2">
                    {columns.map(col => (
                      <Badge key={col.key} variant={col.required ? 'primary' : 'neutral'} size="sm">
                        {col.label}{col.required ? ' *' : ''}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {step === 'preview' && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                    {rows.length} rows found in {file?.name}
                  </p>
                  <p className="text-xs text-slate-500">
                    {validRows.length} valid, {errorRows.length} with errors
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <Badge variant={errorRows.length === 0 ? 'success' : 'warning'}>
                    {errorRows.length === 0 ? 'All valid' : `${errorRows.length} errors`}
                  </Badge>
                </div>
              </div>

              {/* Preview table */}
              <div className="overflow-x-auto border border-slate-200 dark:border-slate-700 rounded-lg">
                <table className="w-full text-xs">
                  <thead>
                    <tr className="bg-slate-50 dark:bg-slate-900">
                      <th className="px-3 py-2 text-left text-slate-500">#</th>
                      {columns.map(col => (
                        <th key={col.key} className="px-3 py-2 text-left text-slate-500">
                          {col.label}
                          {col.required && <span className="text-red-500 ml-0.5">*</span>}
                        </th>
                      ))}
                      <th className="px-3 py-2 text-left text-slate-500">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100 dark:divide-slate-800">
                    {rows.slice(0, 50).map(row => (
                      <tr key={row.index} className={row.valid ? '' : 'bg-red-50 dark:bg-red-900/10'}>
                        <td className="px-3 py-2 text-slate-400">{row.index}</td>
                        {columns.map(col => (
                          <td key={col.key} className="px-3 py-2 text-slate-700 dark:text-slate-300 max-w-[200px] truncate">
                            {row.errors[col.key] ? (
                              <span className="text-red-500" title={row.errors[col.key]}>
                                {row.data[col.key] || '(empty)'}
                              </span>
                            ) : row.data[col.key]}
                          </td>
                        ))}
                        <td className="px-3 py-2">
                          {row.valid
                            ? <CheckCircle className="w-4 h-4 text-emerald-500" />
                            : <AlertTriangle className="w-4 h-4 text-red-500" />}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {rows.length > 50 && (
                <p className="text-xs text-slate-400 text-center">...and {rows.length - 50} more rows</p>
              )}
            </div>
          )}

          {step === 'match' && (
            <div className="space-y-4">
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Map file columns to system fields. Auto-detected mappings are shown below.
              </p>
              <div className="space-y-2">
                {Object.entries(columnMapping).map(([fileCol, systemCol]) => (
                  <div key={fileCol} className="flex items-center gap-3 p-3 bg-slate-50 dark:bg-slate-900 rounded-lg">
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300 min-w-[150px]">
                      {fileCol}
                    </span>
                    <ArrowRight className="w-4 h-4 text-slate-400" />
                    <select
                      value={systemCol}
                      onChange={(e) => setColumnMapping(prev => ({ ...prev, [fileCol]: e.target.value }))}
                      className="flex-1 px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800"
                    >
                      <option value="">— Skip —</option>
                      {columns.map(col => (
                        <option key={col.key} value={col.key}>{col.label}</option>
                      ))}
                    </select>
                  </div>
                ))}
              </div>
            </div>
          )}

          {step === 'review' && (
            <div className="space-y-4">
              <div className="grid grid-cols-3 gap-4">
                <Card>
                  <CardBody className="text-center">
                    <p className="text-2xl font-bold text-blue-600">{validRows.length}</p>
                    <p className="text-xs text-slate-500">Valid rows</p>
                  </CardBody>
                </Card>
                <Card>
                  <CardBody className="text-center">
                    <p className="text-2xl font-bold text-red-500">{errorRows.length}</p>
                    <p className="text-xs text-slate-500">Rows with errors</p>
                  </CardBody>
                </Card>
                <Card>
                  <CardBody className="text-center">
                    <p className="text-2xl font-bold text-emerald-600">{columns.length}</p>
                    <p className="text-xs text-slate-500">Columns to import</p>
                  </CardBody>
                </Card>
              </div>

              {errorRows.length > 0 && (
                <Card>
                  <CardHeader className="flex items-center gap-2 text-amber-600">
                    <AlertTriangle className="w-4 h-4" />
                    <span className="text-sm font-medium">Errors to review</span>
                  </CardHeader>
                  <CardBody>
                    <div className="space-y-1 max-h-32 overflow-y-auto">
                      {errorRows.slice(0, 10).map(row => (
                        <div key={row.index} className="text-xs text-red-500">
                          Row {row.index}: {Object.values(row.errors).join('; ')}
                        </div>
                      ))}
                    </div>
                  </CardBody>
                </Card>
              )}

              <p className="text-xs text-slate-400">
                {errorRows.length > 0
                  ? `Rows with errors will be skipped during import. ${validRows.length} valid rows will be imported.`
                  : `All ${validRows.length} rows are valid and ready for import.`}
              </p>
            </div>
          )}

          {step === 'importing' && (
            <div className="flex flex-col items-center justify-center py-12">
              <div className="w-12 h-12 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mb-4" />
              <p className="text-lg font-medium text-slate-700 dark:text-slate-300">Importing data...</p>
              <p className="text-sm text-slate-500">Processing {validRows.length} rows</p>
            </div>
          )}

          {step === 'complete' && importResult && (
            <div className="flex flex-col items-center justify-center py-8">
              <div className={`w-16 h-16 rounded-full flex items-center justify-center mb-4 ${
                importResult.errors === 0 ? 'bg-emerald-100 dark:bg-emerald-900/30' : 'bg-amber-100 dark:bg-amber-900/30'
              }`}>
                {importResult.errors === 0
                  ? <CheckCircle className="w-8 h-8 text-emerald-600" />
                  : <AlertTriangle className="w-8 h-8 text-amber-600" />}
              </div>
              <h3 className="text-lg font-bold text-slate-900 dark:text-white mb-2">
                Import Complete
              </h3>
              <div className="flex gap-4 mb-6">
                <div className="text-center">
                  <p className="text-2xl font-bold text-emerald-600">{importResult.success}</p>
                  <p className="text-xs text-slate-500">Imported</p>
                </div>
                {importResult.errors > 0 && (
                  <div className="text-center">
                    <p className="text-2xl font-bold text-red-500">{importResult.errors}</p>
                    <p className="text-xs text-slate-500">Errors</p>
                  </div>
                )}
              </div>
              <Button onClick={onClose}>Done</Button>
            </div>
          )}
        </div>

        {/* Footer */}
        {step !== 'complete' && step !== 'importing' && (
          <div className="flex items-center justify-between px-6 py-4 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50">
            <Button variant="outline" onClick={onClose}>Cancel</Button>
            <div className="flex items-center gap-3">
              {step !== 'upload' && (
                <Button variant="outline"
                  onClick={() => setStep(step === 'preview' ? 'upload' : step === 'match' ? 'preview' : 'match')}
                  icon={<ArrowLeft className="w-4 h-4" />}
                >
                  Back
                </Button>
              )}
              {step === 'preview' && (
                <Button onClick={() => setStep('review')} icon={<ArrowRight className="w-4 h-4" />}>
                  Continue
                </Button>
              )}
              {step === 'review' && (
                <Button onClick={handleImport} loading={importing} icon={<Upload className="w-4 h-4" />}>
                  Import {validRows.length} rows
                </Button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// CSV Parser
// ═══════════════════════════════════════════════════════════════════════

function parseCSVLine(line: string): string[] {
  const result: string[] = [];
  let current = '';
  let inQuotes = false;

  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (char === '"') {
      inQuotes = !inQuotes;
    } else if (char === ',' && !inQuotes) {
      result.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }
  result.push(current.trim());
  return result;
}

// ═══════════════════════════════════════════════════════════════════════
// Export helpers
// ═══════════════════════════════════════════════════════════════════════

interface ExportOptions {
  columns: { key: string; label: string }[];
  data: Record<string, unknown>[];
  filename: string;
  format?: 'csv' | 'json';
}

export function exportData({ columns, data, filename, format = 'csv' }: ExportOptions) {
  if (format === 'json') {
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    downloadBlob(blob, `${filename}.json`);
    return;
  }

  // CSV
  const header = columns.map(c => c.label).join(',');
  const rows = data.map(item =>
    columns.map(c => {
      const val = String(item[c.key] ?? '');
      return val.includes(',') ? `"${val}"` : val;
    }).join(',')
  );
  const csv = [header, ...rows].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  downloadBlob(blob, `${filename}.csv`);
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}
