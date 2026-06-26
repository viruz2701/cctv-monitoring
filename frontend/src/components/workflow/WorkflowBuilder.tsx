// ═══════════════════════════════════════════════════════════════════════
// WorkflowBuilder — drag&drop конструктор workflow (заглушка).
//
// P2-2.1: Workflow Builder UI
//   - React Flow для drag&drop (будущее)
//   - CEL conditions editor (будущее)
//   - Workflow testing mode (будущее)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { GitBranch, Play, Save } from 'lucide-react';

export function WorkflowBuilder() {
  const { t } = useTranslation();

  return (
    <div className="p-4 md:p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            {t('workflow_builder') || 'Workflow Builder'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {t('workflow_builder_desc') || 'Design and manage automation workflows'}
          </p>
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            className="flex items-center gap-2 px-4 py-2 bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300 rounded-lg hover:bg-slate-200 text-sm"
          >
            <Play className="w-4 h-4" />
            {t('test') || 'Test'}
          </button>
          <button
            type="button"
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm"
          >
            <Save className="w-4 h-4" />
            {t('save') || 'Save'}
          </button>
        </div>
      </div>

      {/* Coming Soon Placeholder */}
      <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 flex flex-col items-center justify-center text-slate-400">
        <GitBranch className="w-16 h-16 mb-4 opacity-50" />
        <p className="text-lg font-medium text-slate-500 dark:text-slate-300 mb-2">
          {t('workflow_coming_soon') || 'Workflow Builder — Coming Soon'}
        </p>
        <p className="text-sm text-center max-w-md">
          {t('workflow_coming_soon_desc') ||
            'Visual drag-and-drop workflow builder with CEL conditions editor will be available in the next release.'}
        </p>
      </div>
    </div>
  );
}
