// ═══════════════════════════════════════════════════════════════════════
// WorkspaceSwitcher
// UX-14.3.1: Выпадающий список в Header для переключения между рабочими
// пространствами с возможностью создания, редактирования и удаления.
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect, useCallback } from 'react';
import {
  LayoutDashboard,
  Plus,
  Check,
  Pencil,
  Trash2,
  X,
  Palette,
  Monitor,
  Bell,
  BarChart3,
  Shield,
  Wrench,
  FileText,
  Activity,
  Globe,
  Settings,
  Users,
  ChevronDown,
} from 'lucide-react';
import { useWorkspaceStore, type Workspace } from '../../store/workspaceStore';
import { useTranslation } from 'react-i18next';

// ═══════════════════════════════════════════════════════════════════════
// Icon presets
// ═══════════════════════════════════════════════════════════════════════

const ICON_PRESETS: Record<string, React.ReactNode> = {
  LayoutDashboard: <LayoutDashboard className="w-4 h-4" />,
  Monitor: <Monitor className="w-4 h-4" />,
  Bell: <Bell className="w-4 h-4" />,
  BarChart3: <BarChart3 className="w-4 h-4" />,
  Shield: <Shield className="w-4 h-4" />,
  Wrench: <Wrench className="w-4 h-4" />,
  FileText: <FileText className="w-4 h-4" />,
  Activity: <Activity className="w-4 h-4" />,
  Globe: <Globe className="w-4 h-4" />,
  Settings: <Settings className="w-4 h-4" />,
  Users: <Users className="w-4 h-4" />,
  Palette: <Palette className="w-4 h-4" />,
};

const ICON_NAMES = Object.keys(ICON_PRESETS);

// ═══════════════════════════════════════════════════════════════════════
// IconPicker
// ═══════════════════════════════════════════════════════════════════════

interface IconPickerProps {
  value: string;
  onChange: (icon: string) => void;
  onClose: () => void;
}

function IconPicker({ value, onChange, onClose }: IconPickerProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [onClose]);

  return (
    <div
      ref={ref}
      className="absolute top-full left-0 mt-1 p-2 bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 z-50 grid grid-cols-4 gap-1 w-40"
    >
      {ICON_NAMES.map((name) => (
        <button
          key={name}
          onClick={() => { onChange(name); onClose(); }}
          className={`p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors ${
            value === name
              ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
              : 'text-slate-600 dark:text-slate-400'
          }`}
          title={name}
        >
          {ICON_PRESETS[name]}
        </button>
      ))}
    </div>
  );
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
      maxLength={32}
    />
  );
}

// ═══════════════════════════════════════════════════════════════════════
// WorkspaceSwitcher
// ═══════════════════════════════════════════════════════════════════════

export function WorkspaceSwitcher() {
  const { t } = useTranslation();
  const {
    workspaces,
    activeWorkspace,
    createWorkspace,
    updateWorkspace,
    deleteWorkspace,
    setActive,
  } = useWorkspaceStore();

  const [isOpen, setIsOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [showIconPicker, setShowIconPicker] = useState<string | null>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const active = workspaces.find((w) => w.id === activeWorkspace);

  // Close on click outside
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setIsOpen(false);
        setEditingId(null);
        setShowIconPicker(null);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen]);

  const handleCreate = useCallback(() => {
    const id = createWorkspace({
      name: `Workspace ${workspaces.length + 1}`,
      icon: DEFAULT_ICON,
      layout: [],
      visiblePages: [],
    });
    setEditingId(id);
  }, [createWorkspace, workspaces.length]);

  const handleDelete = useCallback(
    (e: React.MouseEvent, id: string) => {
      e.stopPropagation();
      if (workspaces.length <= 1) return; // Don't delete last workspace
      deleteWorkspace(id);
      setEditingId(null);
    },
    [deleteWorkspace, workspaces.length]
  );

  const handleSelect = useCallback(
    (id: string) => {
      setActive(id);
      setIsOpen(false);
      setEditingId(null);
    },
    [setActive]
  );

  const handleRename = useCallback(
    (id: string, name: string) => {
      updateWorkspace(id, { name });
      setEditingId(null);
    },
    [updateWorkspace]
  );

  const DEFAULT_ICON = 'LayoutDashboard';

  return (
    <div ref={menuRef} className="relative">
      {/* Trigger Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
      >
        <span className="text-blue-500">
          {active ? ICON_PRESETS[active.icon] || ICON_PRESETS[DEFAULT_ICON] : ICON_PRESETS[DEFAULT_ICON]}
        </span>
        <span className="max-w-[120px] truncate">{active?.name ?? t('workspace')}</span>
        <ChevronDown className={`w-3.5 h-3.5 text-slate-400 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute top-full left-0 mt-2 w-64 bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 overflow-hidden z-50">
          <div className="px-3 py-2 border-b border-slate-100 dark:border-slate-800">
            <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              {t('workspaces')}
            </p>
          </div>

          <div className="py-1 max-h-64 overflow-y-auto">
            {workspaces.map((ws) => (
              <div
                key={ws.id}
                onClick={() => { if (editingId !== ws.id) handleSelect(ws.id); }}
                className={`flex items-center gap-3 px-3 py-2.5 cursor-pointer transition-colors relative ${
                  activeWorkspace === ws.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                }`}
              >
                {/* Icon */}
                <div className="relative">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setShowIconPicker(showIconPicker === ws.id ? null : ws.id);
                    }}
                    className="p-1 rounded hover:bg-slate-200 dark:hover:bg-slate-700 transition-colors"
                  >
                    {ICON_PRESETS[ws.icon] || ICON_PRESETS[DEFAULT_ICON]}
                  </button>
                  {showIconPicker === ws.id && (
                    <IconPicker
                      value={ws.icon}
                      onChange={(icon) => { updateWorkspace(ws.id, { icon }); }}
                      onClose={() => setShowIconPicker(null)}
                    />
                  )}
                </div>

                {/* Name */}
                <div className="flex-1 min-w-0">
                  {editingId === ws.id ? (
                    <InlineEdit
                      value={ws.name}
                      onSave={(name) => handleRename(ws.id, name)}
                      onCancel={() => setEditingId(null)}
                    />
                  ) : (
                    <span
                      className="text-sm font-medium truncate block"
                      onDoubleClick={() => setEditingId(ws.id)}
                    >
                      {ws.name}
                    </span>
                  )}
                </div>

                {/* Checkmark for active */}
                {activeWorkspace === ws.id && (
                  <Check className="w-4 h-4 text-blue-500 flex-shrink-0" />
                )}

                {/* Actions */}
                <div className="flex items-center gap-0.5 flex-shrink-0">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      setEditingId(editingId === ws.id ? null : ws.id);
                    }}
                    className="p-1 rounded hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
                    title={t('rename')}
                  >
                    <Pencil className="w-3.5 h-3.5" />
                  </button>
                  {workspaces.length > 1 && (
                    <button
                      onClick={(e) => handleDelete(e, ws.id)}
                      className="p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/20 text-slate-400 hover:text-red-500 transition-colors"
                      title={t('delete')}
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>

          {/* Create new workspace */}
          <button
            onClick={handleCreate}
            className="flex items-center gap-3 w-full px-3 py-2.5 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 border-t border-slate-100 dark:border-slate-800 transition-colors font-medium"
          >
            <Plus className="w-4 h-4" />
            <span>{t('new_workspace')}</span>
          </button>
        </div>
      )}
    </div>
  );
}
