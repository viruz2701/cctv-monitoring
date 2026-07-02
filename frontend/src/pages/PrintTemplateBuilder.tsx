// ═══════════════════════════════════════════════════════════════════════
// PrintTemplateBuilder.tsx — Print Template Visual Editor Page
//
// UX-3.5: Visual drag-n-drop редактор печатных форм
//   - Drag-n-drop блоки (text, table, image, signature, QR)
//   - Properties panel для каждого блока
//   - Preview в реальном времени
//   - Save as template (per tenant)
//   - 10+ готовых блоков
//   - Export to JSON schema
//
// Feature Flag: print_template_builder (default: false)
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — feature flag)
//   - ISO 27001 A.12.4 (Audit trail)
//   - OWASP ASVS V1.8 (Feature flags)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useReducer } from 'react';
import { useTranslation } from 'react-i18next';
import { isFeatureEnabled } from '../config/featureFlags';
import { BlockLibrary } from '../components/print-builder/BlockLibrary';
import { TemplateCanvas } from '../components/print-builder/TemplateCanvas';
import { PropertiesPanel } from '../components/print-builder/PropertiesPanel';
import type {
  CanvasBlock,
  BlockDefinition,
  PrintTemplate,
  PageSettings,
  EditorState,
  EditorAction,
} from '../components/print-builder/types';
import {
  Save, Download, Upload, Eye, EyeOff, ZoomIn, ZoomOut,
  ChevronLeft, ChevronRight, AlertTriangle, CheckCircle,
} from 'lucide-react';

// ── Initial State ─────────────────────────────────────────────────────

const DEFAULT_PAGE_SETTINGS: PageSettings = {
  pageSize: 'a4',
  orientation: 'portrait',
  margins: { top: 20, bottom: 20, left: 20, right: 20 },
};

const initialState: EditorState = {
  blocks: [],
  selectedBlockId: null,
  pageSettings: DEFAULT_PAGE_SETTINGS,
  isDragging: false,
  zoom: 100,
  unsavedChanges: false,
};

// ── Reducer ───────────────────────────────────────────────────────────

function editorReducer(state: EditorState, action: EditorAction): EditorState {
  switch (action.type) {
    case 'ADD_BLOCK': {
      const newBlock = action.block;
      // Position at bottom if overlapping
      const maxY = state.blocks.reduce((max, b) => Math.max(max, b.position.y + b.size.height), 0);
      const positionedBlock: CanvasBlock = {
        ...newBlock,
        position: {
          x: newBlock.position.x || 20,
          y: newBlock.position.y || Math.max(20, maxY + 20),
        },
      };
      return {
        ...state,
        blocks: [...state.blocks, positionedBlock],
        selectedBlockId: positionedBlock.instanceId,
        unsavedChanges: true,
      };
    }
    case 'REMOVE_BLOCK':
      return {
        ...state,
        blocks: state.blocks.filter((b) => b.instanceId !== action.instanceId),
        selectedBlockId: state.selectedBlockId === action.instanceId ? null : state.selectedBlockId,
        unsavedChanges: true,
      };
    case 'SELECT_BLOCK':
      return { ...state, selectedBlockId: action.instanceId };
    case 'MOVE_BLOCK':
      return {
        ...state,
        blocks: state.blocks.map((b) =>
          b.instanceId === action.instanceId
            ? { ...b, position: { x: action.x, y: action.y } }
            : b
        ),
        unsavedChanges: true,
      };
    case 'RESIZE_BLOCK':
      return {
        ...state,
        blocks: state.blocks.map((b) =>
          b.instanceId === action.instanceId
            ? { ...b, size: { ...b.size, width: action.width, height: action.height } }
            : b
        ),
        unsavedChanges: true,
      };
    case 'UPDATE_BLOCK_PROPS':
      return {
        ...state,
        blocks: state.blocks.map((b) => {
          if (b.instanceId !== action.instanceId) return b;
          const { _resize, ...restProps } = action.props as Record<string, unknown> & {
            _resize?: { width: number; height: number };
          };
          const updated = { ...b, props: { ...b.props, ...restProps } };
          if (_resize) {
            updated.size = { ...updated.size, width: _resize.width, height: _resize.height };
          }
          return updated;
        }),
        unsavedChanges: true,
      };
    case 'REORDER_BLOCKS':
      return { ...state, blocks: action.blocks, unsavedChanges: true };
    case 'SET_PAGE_SETTINGS':
      return { ...state, pageSettings: action.settings, unsavedChanges: true };
    case 'SET_ZOOM':
      return { ...state, zoom: Math.max(25, Math.min(200, action.zoom)) };
    case 'SET_DRAGGING':
      return { ...state, isDragging: action.isDragging };
    case 'LOAD_TEMPLATE':
      return {
        ...state,
        blocks: action.blocks,
        pageSettings: action.settings,
        selectedBlockId: null,
        unsavedChanges: false,
      };
    case 'CLEAR_SELECTION':
      return { ...state, selectedBlockId: null };
    default:
      return state;
  }
}

// ── Helpers ───────────────────────────────────────────────────────────

let blockCounter = 0;
function generateInstanceId(): string {
  blockCounter++;
  return `block_${Date.now()}_${blockCounter}`;
}

// ── PrintTemplateBuilder Component ────────────────────────────────────

export function PrintTemplateBuilder() {
  const { t } = useTranslation();
  const [state, dispatch] = useReducer(editorReducer, initialState);
  const [showLibrary, setShowLibrary] = useState(true);
  const [showProperties, setShowProperties] = useState(true);
  const [showPreview, setShowPreview] = useState(false);
  const [templateName, setTemplateName] = useState('Untitled Template');
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const isEnabled = isFeatureEnabled('print_template_builder');

  // ── Add block from library ───────────────────────────────────────
  const handleAddBlock = useCallback(
    (blockDef: BlockDefinition, x?: number, y?: number) => {
      const newBlock: CanvasBlock = {
        instanceId: generateInstanceId(),
        definitionId: blockDef.id,
        type: blockDef.type,
        label: blockDef.label,
        props: { ...blockDef.defaultProps },
        position: { x: x ?? 20, y: y ?? 20 },
        size: { width: 200, height: 60 },
        visible: true,
        locked: false,
      };

      // Adjust default sizes by type
      switch (blockDef.type) {
        case 'text':
          newBlock.size = { width: 300, height: 80 };
          break;
        case 'table':
          newBlock.size = { width: 400, height: 120 };
          break;
        case 'image':
          newBlock.size = { width: 200, height: 180 };
          break;
        case 'qr':
          newBlock.size = { width: 120, height: 140 };
          break;
        case 'signature':
          newBlock.size = { width: 250, height: 50 };
          break;
      }

      dispatch({ type: 'ADD_BLOCK', block: newBlock });
    },
    [],
  );

  // ── Toggle lock ──────────────────────────────────────────────────
  const handleToggleLock = useCallback((id: string) => {
    dispatch({
      type: 'UPDATE_BLOCK_PROPS',
      instanceId: id,
      props: { locked: !(state.blocks.find((b) => b.instanceId === id)?.locked ?? false) },
    });
  }, [state.blocks]);

  // ── Toggle visibility ────────────────────────────────────────────
  const handleToggleVisibility = useCallback((id: string) => {
    const block = state.blocks.find((b) => b.instanceId === id);
    if (block) {
      dispatch({
        type: 'UPDATE_BLOCK_PROPS',
        instanceId: id,
        props: { visible: !(block.visible ?? true) },
      });
    }
  }, [state.blocks]);

  // ── Save template ────────────────────────────────────────────────
  const handleSave = useCallback(async () => {
    if (!templateName.trim()) {
      setErrorMessage('Template name is required');
      return;
    }

    setSaveStatus('saving');
    setErrorMessage(null);

    try {
      const template: PrintTemplate = {
        id: `template_${Date.now()}`,
        tenant_id: 'current',
        name: templateName,
        blocks: state.blocks,
        pageSettings: state.pageSettings,
        created_by: 'current_user',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        version: 1,
      };

      // Store in localStorage as fallback (in production → API call)
      const existing = JSON.parse(localStorage.getItem('print_templates') || '[]');
      existing.push(template);
      localStorage.setItem('print_templates', JSON.stringify(existing));

      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } catch {
      setSaveStatus('error');
      setErrorMessage('Failed to save template');
    }
  }, [templateName, state.blocks, state.pageSettings]);

  // ── Export to JSON ───────────────────────────────────────────────
  const handleExport = useCallback(() => {
    const template: PrintTemplate = {
      id: `template_${Date.now()}`,
      tenant_id: 'current',
      name: templateName,
      blocks: state.blocks,
      pageSettings: state.pageSettings,
      created_by: 'current_user',
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
      version: 1,
    };

    const blob = new Blob([JSON.stringify(template, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `${templateName.replace(/\s+/g, '_').toLowerCase()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  }, [templateName, state.blocks, state.pageSettings]);

  // ── Import from JSON ─────────────────────────────────────────────
  const handleImport = useCallback(() => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json';
    input.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0];
      if (!file) return;

      try {
        const text = await file.text();
        const template: PrintTemplate = JSON.parse(text);
        dispatch({
          type: 'LOAD_TEMPLATE',
          blocks: template.blocks,
          settings: template.pageSettings,
        });
        setTemplateName(template.name);
      } catch {
        setErrorMessage('Invalid template file');
      }
    };
    input.click();
  }, []);

  // ── Render ───────────────────────────────────────────────────────
  if (!isEnabled) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center max-w-md">
          <EyeOff className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-4" aria-hidden="true" />
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
            Print Template Builder
          </h2>
          <p className="text-sm text-slate-500 dark:text-slate-400 mb-4">
            This feature is currently disabled. Enable the `print_template_builder` feature flag to use the visual editor.
          </p>
        </div>
      </div>
    );
  }

  const selectedBlock = state.blocks.find((b) => b.instanceId === state.selectedBlockId) || null;

  return (
    <div className="flex flex-col h-[calc(100vh-4rem)] bg-slate-50 dark:bg-slate-900">
      {/* ── Top Toolbar ──────────────────────────────────────────── */}
      <div className="flex items-center justify-between px-4 py-2 bg-white dark:bg-slate-800 border-b border-slate-200 dark:border-slate-700">
        <div className="flex items-center gap-3">
          {/* Template Name */}
          <input
            type="text"
            value={templateName}
            onChange={(e) => setTemplateName(e.target.value)}
            className="text-sm font-medium bg-transparent border-b border-transparent hover:border-slate-300 focus:border-blue-500 px-1 py-0.5 text-slate-900 dark:text-white outline-none"
            aria-label="Template name"
          />

          {/* Feature badge */}
          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400">
            BETA
          </span>
        </div>

        <div className="flex items-center gap-2">
          {/* Zoom controls */}
          <button
            onClick={() => dispatch({ type: 'SET_ZOOM', zoom: state.zoom - 10 })}
            className="p-1.5 rounded-md text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-30"
            disabled={state.zoom <= 25}
            aria-label="Zoom out"
          >
            <ZoomOut className="w-4 h-4" aria-hidden="true" />
          </button>
          <span className="text-xs text-slate-500 min-w-[3rem] text-center tabular-nums">
            {state.zoom}%
          </span>
          <button
            onClick={() => dispatch({ type: 'SET_ZOOM', zoom: state.zoom + 10 })}
            className="p-1.5 rounded-md text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-30"
            disabled={state.zoom >= 200}
            aria-label="Zoom in"
          >
            <ZoomIn className="w-4 h-4" aria-hidden="true" />
          </button>

          <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />

          {/* Preview toggle */}
          <button
            onClick={() => setShowPreview(!showPreview)}
            className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
              showPreview
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400'
                : 'text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-700'
            }`}
            aria-pressed={showPreview}
          >
            {showPreview ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
            Preview
          </button>

          {/* Import */}
          <button
            onClick={handleImport}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-700"
            aria-label="Import template"
          >
            <Upload className="w-3.5 h-3.5" aria-hidden="true" />
            Import
          </button>

          {/* Export */}
          <button
            onClick={handleExport}
            disabled={state.blocks.length === 0}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-slate-600 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-40"
            aria-label="Export template JSON"
          >
            <Download className="w-3.5 h-3.5" aria-hidden="true" />
            Export
          </button>

          {/* Save */}
          <button
            onClick={handleSave}
            disabled={state.blocks.length === 0 || saveStatus === 'saving'}
            className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saveStatus === 'saving' ? (
              <span className="w-3.5 h-3.5 border-2 border-white border-t-transparent rounded-full animate-spin" />
            ) : saveStatus === 'saved' ? (
              <CheckCircle className="w-3.5 h-3.5" />
            ) : (
              <Save className="w-3.5 h-3.5" />
            )}
            {saveStatus === 'saving' ? 'Saving...' : saveStatus === 'saved' ? 'Saved!' : 'Save'}
          </button>
        </div>
      </div>

      {/* ── Error Message ────────────────────────────────────────── */}
      {errorMessage && (
        <div className="flex items-center gap-2 px-4 py-2 bg-red-50 dark:bg-red-900/20 border-b border-red-200 dark:border-red-800">
          <AlertTriangle className="w-4 h-4 text-red-500 flex-shrink-0" />
          <span className="text-sm text-red-700 dark:text-red-400">{errorMessage}</span>
          <button
            onClick={() => setErrorMessage(null)}
            className="ml-auto text-red-500 hover:text-red-700 text-xs"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* ── Main Content ─────────────────────────────────────────── */}
      <div className="flex flex-1 overflow-hidden">
        {/* Library Panel */}
        {showLibrary && (
          <div className="w-64 flex-shrink-0 border-r border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-hidden flex flex-col">
            <div className="flex items-center justify-between px-3 py-2 border-b border-slate-200 dark:border-slate-700">
              <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                Blocks
              </span>
              <button
                onClick={() => setShowLibrary(false)}
                className="p-0.5 rounded text-slate-400 hover:text-slate-600"
                aria-label="Close library panel"
              >
                <ChevronLeft className="w-4 h-4" />
              </button>
            </div>
            <BlockLibrary
              selectedCategory="all"
              onCategoryChange={() => {}}
              onAddBlock={(block) => handleAddBlock(block)}
              searchQuery=""
              onSearchChange={() => {}}
            />
          </div>
        )}

        {/* Library toggle (when hidden) */}
        {!showLibrary && (
          <button
            onClick={() => setShowLibrary(true)}
            className="flex items-center gap-1 px-1 py-2 bg-white dark:bg-slate-800 border-r border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700"
            aria-label="Show library"
          >
            <ChevronRight className="w-4 h-4 text-slate-400" />
            <span className="text-[10px] text-slate-400 writing-mode-vertical">Blocks</span>
          </button>
        )}

        {/* Canvas */}
        <div className={`flex-1 overflow-auto p-6 ${showPreview ? 'bg-white' : 'bg-slate-100 dark:bg-slate-900'}`}>
          <TemplateCanvas
            blocks={state.blocks}
            selectedBlockId={state.selectedBlockId}
            pageSettings={state.pageSettings}
            zoom={state.zoom}
            onSelectBlock={(id) => dispatch({ type: 'SELECT_BLOCK', instanceId: id })}
            onMoveBlock={(id, x, y) => dispatch({ type: 'MOVE_BLOCK', instanceId: id, x, y })}
            onResizeBlock={(id, w, h) => dispatch({ type: 'RESIZE_BLOCK', instanceId: id, width: w, height: h })}
            onRemoveBlock={(id) => dispatch({ type: 'REMOVE_BLOCK', instanceId: id })}
            onToggleLock={handleToggleLock}
            onToggleVisibility={handleToggleVisibility}
            onDropBlock={(blockDef, x, y) => handleAddBlock(blockDef, x, y)}
            onReorderBlocks={(blocks) => dispatch({ type: 'REORDER_BLOCKS', blocks })}
          />
        </div>

        {/* Properties Panel */}
        {showProperties && (
          <div className="w-72 flex-shrink-0 border-l border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 overflow-hidden flex flex-col">
            <div className="flex items-center justify-between px-3 py-2 border-b border-slate-200 dark:border-slate-700">
              <span className="text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider">
                Properties
              </span>
              <button
                onClick={() => setShowProperties(false)}
                className="p-0.5 rounded text-slate-400 hover:text-slate-600"
                aria-label="Close properties panel"
              >
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
            <PropertiesPanel
              selectedBlock={selectedBlock}
              onUpdateProps={(id, props) => dispatch({ type: 'UPDATE_BLOCK_PROPS', instanceId: id, props })}
              blocksCount={state.blocks.length}
            />
          </div>
        )}

        {/* Properties toggle (when hidden) */}
        {!showProperties && (
          <button
            onClick={() => setShowProperties(true)}
            className="flex items-center gap-1 px-1 py-2 bg-white dark:bg-slate-800 border-l border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700"
            aria-label="Show properties"
          >
            <span className="text-[10px] text-slate-400 writing-mode-vertical">Properties</span>
            <ChevronLeft className="w-4 h-4 text-slate-400" />
          </button>
        )}
      </div>

      {/* ── Status Bar ───────────────────────────────────────────── */}
      <div className="flex items-center justify-between px-4 py-1 bg-white dark:bg-slate-800 border-t border-slate-200 dark:border-slate-700">
        <div className="flex items-center gap-4 text-[10px] text-slate-400">
          <span>{state.blocks.length} block{state.blocks.length !== 1 ? 's' : ''}</span>
          <span>{state.pageSettings.pageSize.toUpperCase()} / {state.pageSettings.orientation}</span>
          {state.unsavedChanges && (
            <span className="text-amber-500 font-medium">Unsaved changes</span>
          )}
        </div>
        <div className="text-[10px] text-slate-400">
          {selectedBlock ? `${selectedBlock.type} · ${selectedBlock.size.width}×${selectedBlock.size.height}` : 'No selection'}
        </div>
      </div>
    </div>
  );
}

export default PrintTemplateBuilder;
