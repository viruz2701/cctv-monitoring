// ═══════════════════════════════════════════════════════════════════════
// SavedFiltersDropdown — сохранение/загрузка фильтров DataGrid
//
// P1-UX.8: Saved Filters
//   - Save filter presets per page
//   - Named filters: "Critical Overdue", "My Team"
//   - Default filters per role
//   - Share filters via URL
//   - Export/import filters как JSON
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Filter, Save, Trash2, Check, Download, Upload, Share2,
  ChevronDown, Plus, X, Star,
} from './Icons';
import { useFilterStore, type SavedView } from '../../store/filterStore';
import { useAuth } from '../../hooks/useAuth';

// ── Types ─────────────────────────────────────────────────────────────

export interface FilterState {
  filters: Record<string, string>;
  sort: {
    column: string;
    direction: 'asc' | 'desc';
  };
}

interface SavedFiltersDropdownProps {
  page: string;
  currentFilterState: FilterState;
  onApplyView: (view: SavedView) => void;
  className?: string;
}

// ── Default Filters per Role ──────────────────────────────────────────

const ROLE_DEFAULT_FILTERS: Record<string, Record<string, string>> = {
  technician: { quickFilter: 'mine' },
  manager: {},
  admin: {},
  viewer: {},
  owner: {},
  support: {},
};

// ═══════════════════════════════════════════════════════════════════════
// SavedFiltersDropdown
// ═══════════════════════════════════════════════════════════════════════

export function SavedFiltersDropdown({
  page, currentFilterState, onApplyView, className = '',
}: SavedFiltersDropdownProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { savedViews, saveView, deleteView, getViewsForPage } = useFilterStore();
  const [isOpen, setIsOpen] = useState(false);
  const [showSaveInput, setShowSaveInput] = useState(false);
  const [saveName, setSaveName] = useState('');
  const menuRef = useRef<HTMLDivElement>(null);

  const viewsForPage = getViewsForPage(page);

  const activeView = savedViews.find(
    (v) =>
      v.page === page &&
      JSON.stringify(v.filters) === JSON.stringify(currentFilterState.filters) &&
      v.sort.column === currentFilterState.sort.column &&
      v.sort.direction === currentFilterState.sort.direction
  );

  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setShowSaveInput(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen]);

  const handleSave = useCallback(() => {
    const name = saveName.trim();
    if (!name) return;
    saveView(name, page, currentFilterState.filters, currentFilterState.sort);
    setSaveName('');
    setShowSaveInput(false);
  }, [saveName, page, currentFilterState, saveView]);

  const handleExportJSON = useCallback(() => {
    const data = JSON.stringify(viewsForPage, null, 2);
    const blob = new Blob([data], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `filters-${page}-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, [viewsForPage, page]);

  const handleImportJSON = useCallback(() => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = async (e: Event) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;
      try {
        const text = await file.text();
        const views = JSON.parse(text);
        if (Array.isArray(views)) {
          views.forEach((v: SavedView) => {
            saveView(v.name, page, v.filters, v.sort);
          });
        }
      } catch {
        console.error('Failed to import filters');
      }
    };
    input.click();
  }, [page, saveView]);

  const shareUrl = useCallback((view: SavedView) => {
    const state = encodeFilterState({
      filters: view.filters,
      sort: view.sort,
    });
    const url = new URL(window.location.href);
    url.searchParams.set('filters', state);
    navigator.clipboard.writeText(url.toString());
  }, []);

  return (
    <div ref={menuRef} className={`relative ${className}`}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-colors ${
          activeView
            ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300'
            : 'bg-white dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
        }`}
        title={activeView?.name}
      >
        <Filter className={`w-4 h-4 ${activeView ? 'text-blue-500' : ''}`} />
        <span className="hidden sm:inline">
          {activeView ? activeView.name : (t('saved_filters') || 'Saved Filters')}
        </span>
        <ChevronDown className={`w-3.5 h-3.5 text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {isOpen && (
        <div className="absolute top-full right-0 mt-2 w-72 bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 overflow-hidden z-50">
          {/* Header */}
          <div className="px-3 py-2 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
            <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              {t('saved_filters') || 'Saved Filters'}
            </p>
            <div className="flex items-center gap-1">
              <button
                onClick={handleImportJSON}
                className="p-1 hover:bg-slate-100 dark:hover:bg-slate-800 rounded text-slate-400 hover:text-slate-600"
                title={t('import') || 'Import'}
              >
                <Upload className="w-3.5 h-3.5" />
              </button>
              <button
                onClick={handleExportJSON}
                className="p-1 hover:bg-slate-100 dark:hover:bg-slate-800 rounded text-slate-400 hover:text-slate-600"
                title={t('export') || 'Export'}
              >
                <Download className="w-3.5 h-3.5" />
              </button>
              <span className="text-xs text-slate-400 ml-1">{viewsForPage.length}</span>
            </div>
          </div>

          {/* Saved Filters List */}
          <div className="py-1 max-h-56 overflow-y-auto">
            {viewsForPage.length === 0 && !showSaveInput && (
              <div className="px-4 py-6 text-center">
                <Filter className="w-8 h-8 mx-auto text-slate-300 dark:text-slate-600 mb-2" />
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {t('no_saved_filters') || 'No saved filters'}
                </p>
                <p className="text-xs text-slate-400 mt-1">
                  {t('save_filters_hint') || 'Save your current filter settings for quick access'}
                </p>
              </div>
            )}

            {viewsForPage.map((view) => (
              <div
                key={view.id}
                onClick={() => { onApplyView(view); setIsOpen(false); }}
                className={`flex items-center gap-3 px-3 py-2.5 cursor-pointer transition-colors ${
                  activeView?.id === view.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                }`}
              >
                {activeView?.id === view.id ? (
                  <Star className="w-4 h-4 fill-blue-500 text-blue-500 flex-shrink-0" />
                ) : (
                  <Filter className="w-4 h-4 text-slate-400 flex-shrink-0" />
                )}

                <div className="flex-1 min-w-0">
                  <span className="text-sm font-medium truncate block">
                    {view.name}
                  </span>
                  <span className="text-xs text-slate-400 truncate block mt-0.5">
                    {Object.keys(view.filters)
                      .filter((k) => view.filters[k] !== 'all' && view.filters[k] !== '')
                      .map((k) => `${k}: ${view.filters[k]}`)
                      .join(', ') || (t('no_filters') || 'No filters')}
                  </span>
                </div>

                {activeView?.id === view.id && (
                  <Check className="w-3.5 h-3.5 text-blue-500 flex-shrink-0" />
                )}

                <div className="flex items-center gap-0.5 flex-shrink-0">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      shareUrl(view);
                    }}
                    className="p-1 rounded hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-400 hover:text-blue-500 transition-colors"
                    title={t('share') || 'Share'}
                  >
                    <Share2 className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={(e) => { e.stopPropagation(); deleteView(view.id); }}
                    className="p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/20 text-slate-400 hover:text-red-500 transition-colors"
                    title={t('delete') || 'Delete'}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>

          {/* Save Current Filter */}
          <div className="border-t border-slate-100 dark:border-slate-800 px-3 py-2.5">
            {showSaveInput ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={saveName}
                  onChange={(e) => setSaveName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleSave();
                    if (e.key === 'Escape') setShowSaveInput(false);
                  }}
                  placeholder={t('filter_name_placeholder') || 'Filter name...'}
                  className="flex-1 text-sm px-2 py-1.5 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder:text-slate-400 outline-none focus:ring-2 focus:ring-blue-500"
                  maxLength={48}
                  autoFocus
                />
                <button
                  onClick={handleSave}
                  disabled={!saveName.trim()}
                  className="p-1.5 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  <Save className="w-4 h-4" />
                </button>
                <button
                  onClick={() => { setShowSaveInput(false); setSaveName(''); }}
                  className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-400 transition-colors"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            ) : (
              <button
                onClick={() => setShowSaveInput(true)}
                className="flex items-center gap-2 w-full text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors font-medium"
              >
                <Plus className="w-4 h-4" />
                <span>{t('save_current_filters') || 'Save current filters'}</span>
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ── Helper: encode filter state for URL sharing ───────────────────────

function encodeFilterState(state: FilterState): string {
  try {
    return btoa(JSON.stringify(state));
  } catch {
    return '';
  }
}

export function decodeFilterState(encoded: string): FilterState | null {
  try {
    return JSON.parse(atob(encoded));
  } catch {
    return null;
  }
}

export default SavedFiltersDropdown;
