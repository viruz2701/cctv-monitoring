// ═══════════════════════════════════════════════════════════════════════
// WorkflowToolbar — боковая панель с палитрой + тулбар (P2-2.1)
//
// Содержит:
//   - Палитру компонентов для drag&drop (Trigger, Condition, Action, Delay)
//   - Кнопки управления: Save, Save Version, Load Version, Test, Export, Import
//   - Version selector для загрузки предыдущих версий
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Zap,
  GitBranch,
  Play,
  Timer,
  Save,
  History,
  Upload,
  Download,
  Beaker,
  X,
  FileJson,
  Plus,
  ArrowLeft,
} from '../ui/Icons';
import {
  WORKFLOW_PALETTE,
  type PaletteItem,
  type WorkflowNode,
  type WorkflowDefinition,
} from '../../types/workflow';
import { useWorkflowStore } from '../../store/workflowStore';

// ═══════════════════════════════════════════════════════════════════════
// Icons map (lazy — используем строковые ключи)
// ═══════════════════════════════════════════════════════════════════════

const ICON_MAP: Record<string, React.ElementType> = {
  Zap,
  GitBranch,
  Play,
  Timer,
};

// ═══════════════════════════════════════════════════════════════════════
// Palette Item Component
// ═══════════════════════════════════════════════════════════════════════

function PaletteItemCard({ item }: { item: PaletteItem }) {
  const Icon = ICON_MAP[item.icon] ?? Play;

  const handleDragStart = useCallback(
    (event: React.DragEvent<HTMLDivElement>) => {
      event.dataTransfer.setData(
        'application/reactflow',
        JSON.stringify({
          kind: item.kind,
          defaultConfig: item.defaultConfig,
          label: item.label,
        })
      );
      event.dataTransfer.effectAllowed = 'move';
    },
    [item]
  );

  return (
    <div
      draggable
      onDragStart={handleDragStart}
      className={[
        'flex items-start gap-3 p-3 rounded-lg border-2 cursor-grab',
        'hover:shadow-md active:cursor-grabbing active:shadow-lg',
        'transition-all duration-150 select-none',
        item.color,
      ].join(' ')}
    >
      <Icon className="w-5 h-5 mt-0.5 shrink-0 text-slate-600 dark:text-slate-300" />
      <div className="min-w-0">
        <p className="text-sm font-semibold text-slate-800 dark:text-slate-200">
          {item.label}
        </p>
        <p className="text-xs text-slate-500 dark:text-slate-400 leading-tight">
          {item.description}
        </p>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Version Selector
// ═══════════════════════════════════════════════════════════════════════

function VersionSelector() {
  const { t } = useTranslation();
  const versions = useWorkflowStore((s) =>
    s.activeWorkflowId ? s.getVersions(s.activeWorkflowId) : []
  );
  const loadVersion = useWorkflowStore((s) => s.loadVersion);
  const activeWorkflowId = useWorkflowStore((s) => s.activeWorkflowId);

  if (!activeWorkflowId || versions.length === 0) return null;

  return (
    <div className="space-y-1">
      <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
        {t('versions') || 'Versions'}
      </p>
      <div className="max-h-32 overflow-y-auto space-y-1">
        {versions
          .slice()
          .reverse()
          .slice(0, 5)
          .map((v) => (
            <button
              key={v.id}
              type="button"
              onClick={() => loadVersion(activeWorkflowId, v.id)}
              className="w-full text-left text-xs px-2 py-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400 truncate"
            >
              v{v.version}: {v.message ?? new Date(v.createdAt).toLocaleString()}
            </button>
          ))}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Toolbar Component
// ═══════════════════════════════════════════════════════════════════════

interface WorkflowToolbarProps {
  onClose: () => void;
}

export function WorkflowToolbar({ onClose }: WorkflowToolbarProps) {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const activeWorkflowId = useWorkflowStore((s) => s.activeWorkflowId);
  const isDirty = useWorkflowStore((s) => s.isDirty);
  const testMode = useWorkflowStore((s) => s.testMode);
  const workflows = useWorkflowStore((s) => s.workflows);

  const saveCurrentWorkflow = useWorkflowStore((s) => s.saveCurrentWorkflow);
  const saveVersion = useWorkflowStore((s) => s.saveVersion);
  const exportWorkflow = useWorkflowStore((s) => s.exportWorkflow);
  const importWorkflow = useWorkflowStore((s) => s.importWorkflow);
  const toggleTestMode = useWorkflowStore((s) => s.toggleTestMode);
  const createWorkflow = useWorkflowStore((s) => s.createWorkflow);
  const setActiveWorkflow = useWorkflowStore((s) => s.setActiveWorkflow);
  const resetEditor = useWorkflowStore((s) => s.resetEditor);

  // ─── Export ────────────────────────────────────────────────────────
  const handleExport = useCallback(() => {
    if (!activeWorkflowId) return;
    const data = exportWorkflow(activeWorkflowId);
    if (!data) return;

    const blob = new Blob([JSON.stringify(data, null, 2)], {
      type: 'application/json',
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${data.name.replace(/\s+/g, '_')}_v${data.version}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [activeWorkflowId, exportWorkflow]);

  // ─── Import ────────────────────────────────────────────────────────
  const handleImport = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (event) => {
        try {
          const data = JSON.parse(event.target?.result as string) as WorkflowDefinition;
          if (!data.nodes || !data.edges || !data.name) {
            alert('Invalid workflow file');
            return;
          }
          importWorkflow(data);
        } catch {
          alert('Failed to parse workflow file');
        }
      };
      reader.readAsText(file);
      // Reset input
      if (fileInputRef.current) fileInputRef.current.value = '';
    },
    [importWorkflow]
  );

  // ─── Save Version with prompt ──────────────────────────────────────
  const handleSaveVersion = useCallback(() => {
    const message = prompt('Version description (optional):');
    saveVersion(message ?? undefined);
  }, [saveVersion]);

  // ─── Create new workflow ──────────────────────────────────────────
  const handleCreateNew = useCallback(() => {
    const name = prompt('Workflow name:');
    if (!name?.trim()) return;
    createWorkflow(name.trim());
  }, [createWorkflow]);

  return (
    <aside className="w-72 shrink-0 border-r border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 flex flex-col h-full overflow-hidden">
      {/* ─── Header ───────────────────────────────────────────────── */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700">
        <h2 className="text-sm font-bold text-slate-800 dark:text-slate-200">
          {t('workflow_toolbar') || 'Workflow Builder'}
        </h2>
        <button
          type="button"
          onClick={onClose}
          className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-700"
          aria-label="Close"
        >
          <X className="w-4 h-4 text-slate-500" />
        </button>
      </div>

      {/* ─── Scrollable content ───────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto p-4 space-y-5">
        {/* ─── Workflow Selector ──────────────────────────────────── */}
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              {t('workflows') || 'Workflows'}
            </p>
            <button
              type="button"
              onClick={handleCreateNew}
              className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-700"
              title="New workflow"
            >
              <Plus className="w-4 h-4 text-slate-500" />
            </button>
          </div>
          <div className="max-h-32 overflow-y-auto space-y-1">
            <button
              type="button"
              onClick={resetEditor}
              className={[
                'w-full text-left text-xs px-2 py-1.5 rounded truncate',
                !activeWorkflowId
                  ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                  : 'hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400',
              ].join(' ')}
            >
              New Draft
            </button>
            {workflows.map((w) => (
              <button
                key={w.id}
                type="button"
                onClick={() => setActiveWorkflow(w.id)}
                className={[
                  'w-full text-left text-xs px-2 py-1.5 rounded truncate',
                  activeWorkflowId === w.id
                    ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                    : 'hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400',
                ].join(' ')}
              >
                {w.name}
                {w.version > 1 && ` (v${w.version})`}
              </button>
            ))}
          </div>
        </div>

        {/* ─── Palette ────────────────────────────────────────────── */}
        <div className="space-y-2">
          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            {t('components') || 'Components'}
          </p>
          <div className="space-y-2">
            {WORKFLOW_PALETTE.map((item) => (
              <PaletteItemCard key={item.kind} item={item} />
            ))}
          </div>
          <p className="text-[10px] text-slate-400 dark:text-slate-500 italic">
            Drag components onto the canvas
          </p>
        </div>

        {/* ─── Actions ────────────────────────────────────────────── */}
        <div className="space-y-2">
          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            {t('actions') || 'Actions'}
          </p>
          <div className="space-y-1.5">
            <button
              type="button"
              onClick={saveCurrentWorkflow}
              disabled={!isDirty}
              className={[
                'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                isDirty
                  ? 'bg-blue-600 text-white hover:bg-blue-700'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-400 dark:text-slate-500 cursor-not-allowed',
              ].join(' ')}
            >
              <Save className="w-4 h-4" />
              {t('save') || 'Save'}
              {isDirty && (
                <span className="ml-auto w-2 h-2 rounded-full bg-blue-300" />
              )}
            </button>

            <button
              type="button"
              onClick={handleSaveVersion}
              disabled={!activeWorkflowId}
              className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors disabled:opacity-50"
            >
              <History className="w-4 h-4" />
              {t('save_version') || 'Save Version'}
            </button>

            <button
              type="button"
              onClick={toggleTestMode}
              className={[
                'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
                testMode
                  ? 'bg-amber-500 text-white hover:bg-amber-600'
                  : 'bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600',
              ].join(' ')}
            >
              <Beaker className="w-4 h-4" />
              {t('test_mode') || 'Test Mode'}
              {testMode && (
                <span className="ml-auto text-[10px] bg-amber-400 text-amber-900 px-1.5 py-0.5 rounded-full font-bold">
                  ON
                </span>
              )}
            </button>
          </div>
        </div>

        {/* ─── Export / Import ────────────────────────────────────── */}
        <div className="space-y-2">
          <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            {t('export_import') || 'Export / Import'}
          </p>
          <div className="space-y-1.5">
            <button
              type="button"
              onClick={handleExport}
              disabled={!activeWorkflowId}
              className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors disabled:opacity-50"
            >
              <Download className="w-4 h-4" />
              {t('export_json') || 'Export JSON'}
            </button>

            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-600 transition-colors"
            >
              <Upload className="w-4 h-4" />
              {t('import_json') || 'Import JSON'}
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json"
              onChange={handleImport}
              className="hidden"
            />
          </div>
        </div>

        {/* ─── Version History ────────────────────────────────────── */}
        <VersionSelector />
      </div>
    </aside>
  );
}
