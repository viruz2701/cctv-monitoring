// ═══════════════════════════════════════════════════════════════════════
// SavedViews
// UX-14.3.2: Компонент для сохранения/загрузки/управления фильтрами на
// страницах со списками (Devices, Sites, Tickets, WorkOrders, Alerts).
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect, useCallback } from 'react';
import {
  Bookmark,
  Check,
  ChevronDown,
  Trash2,
  Pencil,
  Plus,
  X,
  Save,
} from 'lucide-react';
import { useFilterStore, type SavedView } from '../../store/filterStore';
import { useTranslation } from 'react-i18next';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface FilterState {
  filters: Record<string, string>;
  sort: {
    column: string;
    direction: 'asc' | 'desc';
  };
}

interface SavedViewsProps {
  /** Уникальный идентификатор страницы (напр. 'devices', 'sites', 'tickets') */
  page: string;
  /** Текущее состояние фильтров */
  currentFilterState: FilterState;
  /** Callback для применения сохранённого фильтра */
  onApplyView: (view: SavedView) => void;
  /** Label для кнопки (по умолчанию 'Save View') */
  buttonLabel?: string;
  /** Дополнительные CSS-классы */
  className?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// InlineEdit
// ═══════════════════════════════════════════════════════════════════════

interface InlineEditProps {
  value: string;
  onSave: (value: string) => void;
  onCancel: () => void;
}

function InlineEdit({ value, onSave, onCancel }: InlineEditProps) {
  const [editValue, setEditValue] = useState(value);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
    inputRef.current?.select();
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      onSave(editValue.trim() || value);
    } else if (e.key === 'Escape') {
      onCancel();
    }
  };

  return (
    <input
      ref={inputRef}
      type="text"
      value={editValue}
      onChange={(e) => setEditValue(e.target.value)}
      onKeyDown={handleKeyDown}
      onBlur={() => onSave(editValue.trim() || value)}
      className="flex-1 text-sm bg-transparent border-b border-blue-500 outline-none text-slate-900 dark:text-white px-0 py-0.5"
      maxLength={48}
    />
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SavedViews
// ═══════════════════════════════════════════════════════════════════════

export function SavedViews({
  page,
  currentFilterState,
  onApplyView,
  buttonLabel,
  className = '',
}: SavedViewsProps) {
  const { t } = useTranslation();
  const { savedViews, saveView, deleteView, renameView, getViewsForPage } =
    useFilterStore();

  const [isOpen, setIsOpen] = useState(false);
  const [showSaveInput, setShowSaveInput] = useState(false);
  const [saveName, setSaveName] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const saveInputRef = useRef<HTMLInputElement>(null);

  const viewsForPage = getViewsForPage(page);
  const activeView = savedViews.find(
    (v) =>
      v.page === page &&
      JSON.stringify(v.filters) === JSON.stringify(currentFilterState.filters) &&
      v.sort.column === currentFilterState.sort.column &&
      v.sort.direction === currentFilterState.sort.direction
  );

  // Close on click outside
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setShowSaveInput(false);
        setEditingId(null);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen]);

  // Auto-focus save input
  useEffect(() => {
    if (showSaveInput) {
      saveInputRef.current?.focus();
    }
  }, [showSaveInput]);

  const handleSave = useCallback(() => {
    const name = saveName.trim();
    if (!name) return;
    saveView(name, page, currentFilterState.filters, currentFilterState.sort);
    setSaveName('');
    setShowSaveInput(false);
  }, [saveName, page, currentFilterState, saveView]);

  const handleApply = useCallback(
    (view: SavedView) => {
      onApplyView(view);
      setIsOpen(false);
    },
    [onApplyView]
  );

  const handleDelete = useCallback(
    (e: React.MouseEvent, id: string) => {
      e.stopPropagation();
      deleteView(id);
      if (editingId === id) setEditingId(null);
    },
    [deleteView, editingId]
  );

  const handleRename = useCallback(
    (id: string, name: string) => {
      renameView(id, name);
      setEditingId(null);
    },
    [renameView]
  );

  return (
    <div ref={menuRef} className={`relative ${className}`}>
      {/* Toggle Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center gap-2 px-3 py-2 text-sm font-medium rounded-lg border transition-colors ${
          activeView
            ? 'bg-blue-50 dark:bg-blue-900/20 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300'
            : 'bg-white dark:bg-slate-800 border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700'
        }`}
        title={activeView?.name}
      >
        <Bookmark className={`w-4 h-4 ${activeView ? 'fill-blue-500 text-blue-500' : ''}`} />
        <span className="hidden sm:inline">
          {activeView ? activeView.name : (buttonLabel || t('saved_views'))}
        </span>
        <ChevronDown className={`w-3.5 h-3.5 text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div className="absolute top-full right-0 mt-2 w-72 bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 overflow-hidden z-50">
          {/* Header */}
          <div className="px-3 py-2 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
            <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              {t('saved_views')}
            </p>
            <span className="text-xs text-slate-400">{viewsForPage.length}</span>
          </div>

          {/* Saved Views List */}
          <div className="py-1 max-h-56 overflow-y-auto">
            {viewsForPage.length === 0 && !showSaveInput && (
              <div className="px-4 py-6 text-center">
                <Bookmark className="w-8 h-8 mx-auto text-slate-300 dark:text-slate-600 mb-2" />
                <p className="text-sm text-slate-500 dark:text-slate-400">
                  {t('no_saved_views')}
                </p>
              </div>
            )}

            {viewsForPage.map((view) => (
              <div
                key={view.id}
                onClick={() => handleApply(view)}
                className={`flex items-center gap-3 px-3 py-2.5 cursor-pointer transition-colors ${
                  activeView?.id === view.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                }`}
              >
                <Bookmark className={`w-4 h-4 flex-shrink-0 ${
                  activeView?.id === view.id
                    ? 'fill-blue-500 text-blue-500'
                    : 'text-slate-400'
                }`} />

                <div className="flex-1 min-w-0">
                  {editingId === view.id ? (
                    <InlineEdit
                      value={view.name}
                      onSave={(name) => handleRename(view.id, name)}
                      onCancel={() => setEditingId(null)}
                    />
                  ) : (
                    <span className="text-sm font-medium truncate block">
                      {view.name}
                    </span>
                  )}
                  <span className="text-xs text-slate-400 truncate block mt-0.5">
                    {Object.keys(view.filters)
                      .filter((k) => view.filters[k] !== 'all' && view.filters[k] !== '')
                      .map((k) => `${k}: ${view.filters[k]}`)
                      .join(', ') || t('no_filters')}
                  </span>
                </div>

                {activeView?.id === view.id && (
                  <Check className="w-3.5 h-3.5 text-blue-500 flex-shrink-0" />
                )}

                <div className="flex items-center gap-0.5 flex-shrink-0">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setEditingId(editingId === view.id ? null : view.id);
                    }}
                    className="p-1 rounded hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                    title={t('rename')}
                  >
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                  <button
                    onClick={(e) => handleDelete(e, view.id)}
                    className="p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/20 text-slate-400 hover:text-red-500 transition-colors"
                    title={t('delete')}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            ))}
          </div>

          {/* Save Current View */}
          <div className="border-t border-slate-100 dark:border-slate-800 px-3 py-2.5">
            {showSaveInput ? (
              <div className="flex items-center gap-2">
                <input
                  ref={saveInputRef}
                  type="text"
                  value={saveName}
                  onChange={(e) => setSaveName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleSave();
                    if (e.key === 'Escape') setShowSaveInput(false);
                  }}
                  placeholder={t('view_name_placeholder')}
                  className="flex-1 text-sm px-2 py-1.5 border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder:text-slate-400 outline-none focus:ring-2 focus:ring-blue-500"
                  maxLength={48}
                />
                <button
                  onClick={handleSave}
                  disabled={!saveName.trim()}
                  className="p-1.5 rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  title={t('save')}
                >
                  <Save className="w-4 h-4" />
                </button>
                <button
                  onClick={() => { setShowSaveInput(false); setSaveName(''); }}
                  className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-slate-400 transition-colors"
                  title={t('cancel')}
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
                <span>{t('save_current_view')}</span>
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
