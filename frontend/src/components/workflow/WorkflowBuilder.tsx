// ═══════════════════════════════════════════════════════════════════════
// WorkflowBuilder — drag&drop конструктор workflow на React Flow (P2-2.1)
//
// Интеграция:
//   - React Flow (@xyflow/react) для графа workflow
//   - Custom nodes: WorkflowNode (4 вида: trigger/condition/action/delay)
//   - Sidebar: WorkflowToolbar с палитрой компонентов
//   - CEL editor: WorkflowCELInput для редактирования condition нод
//   - Test mode: WorkflowTestPanel для запуска с mock event
//
// Features:
//   - Drag&drop из палитры в canvas
//   - Custom nodes с цветовой дифференциацией
//   - Condition node с true/false handles
//   - Node config editing в правой панели (inspector)
//   - Test mode со статусами выполнения
//   - Export/Import как JSON
//   - Version control
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  useReactFlow,
  type ReactFlowInstance,
  type Node as FlowNode,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import {
  Undo2,
  Redo2,
  ZoomIn,
  ZoomOut,
  Maximize2,
  Save,
  Upload,
  Download,
  FileJson,
  Cloud,
  CloudOff,
} from 'lucide-react';

import { WorkflowNode } from './WorkflowNode';
import { WorkflowToolbar } from './WorkflowToolbar';
import { WorkflowCELInput } from './WorkflowCELInput';
import { WorkflowTestPanel } from './WorkflowTestPanel';
import { useWorkflowStore } from '../../store/workflowStore';
import type { WorkflowNodeData, WorkflowNodeKind } from '../../types/workflow';
import { getWorkflow, createWorkflow, updateWorkflow } from '../../services/api/workflows';

// ═══════════════════════════════════════════════════════════════════════
// Node Types Registration
// ═══════════════════════════════════════════════════════════════════════

const nodeTypes = {
  workflowNode: WorkflowNode,
};

// ═══════════════════════════════════════════════════════════════════════
// Default edge options
// ═══════════════════════════════════════════════════════════════════════

const defaultEdgeOptions = {
  animated: true,
  style: { stroke: '#94a3b8', strokeWidth: 2 },
};

// ═══════════════════════════════════════════════════════════════════════
// Generate a unique node ID
// ═══════════════════════════════════════════════════════════════════════

const generateNodeId = (kind: WorkflowNodeKind): string =>
  `${kind}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;

// ═══════════════════════════════════════════════════════════════════════
// Props
// ═══════════════════════════════════════════════════════════════════════

interface WorkflowBuilderProps {
  sidebarInitiallyHidden?: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function WorkflowBuilder({ sidebarInitiallyHidden = false }: WorkflowBuilderProps) {
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [rfInstance, setRfInstance] = useState<ReactFlowInstance | null>(null);
  const [showSidebar, setShowSidebar] = useState(!sidebarInitiallyHidden);
  const [selectedNodeData, setSelectedNodeData] =
    useState<{ id: string; data: WorkflowNodeData } | null>(null);
  const [zoomLevel, setZoomLevel] = useState(1);
  const [saveStatus, setSaveStatus] = useState<'saved' | 'unsaved' | 'saving'>('saved');
  const [apiWorkflowId, setApiWorkflowId] = useState<string | null>(null);

  // ─── Store ─────────────────────────────────────────────────────────
  const nodes = useWorkflowStore((s) => s.nodes);
  const edges = useWorkflowStore((s) => s.edges);
  const testMode = useWorkflowStore((s) => s.testMode);
  const isDirty = useWorkflowStore((s) => s.isDirty);
  const activeWorkflowId = useWorkflowStore((s) => s.activeWorkflowId);
  const onNodesChange = useWorkflowStore((s) => s.onNodesChange);
  const onEdgesChange = useWorkflowStore((s) => s.onEdgesChange);
  const onConnect = useWorkflowStore((s) => s.onConnect);
  const addNode = useWorkflowStore((s) => s.addNode);
  const removeNode = useWorkflowStore((s) => s.removeNode);
  const selectNode = useWorkflowStore((s) => s.selectNode);
  const updateNodeConfig = useWorkflowStore((s) => s.updateNodeConfig);
  const undo = useWorkflowStore((s) => s.undo);
  const redo = useWorkflowStore((s) => s.redo);
  const pushHistory = useWorkflowStore((s) => s.pushHistory);
  const canUndo = useWorkflowStore((s) => s.canUndo);
  const canRedo = useWorkflowStore((s) => s.canRedo);

  // ─── Drag & Drop ──────────────────────────────────────────────────

  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      if (!rfInstance || !reactFlowWrapper.current) return;

      const rawData = event.dataTransfer.getData('application/reactflow');
      if (!rawData) return;

      try {
        const { kind, defaultConfig, label } = JSON.parse(rawData) as {
          kind: WorkflowNodeKind;
          defaultConfig: Record<string, unknown>;
          label: string;
        };

        const position = rfInstance.screenToFlowPosition({
          x: event.clientX,
          y: event.clientY,
        });

        const newNode: FlowNode<WorkflowNodeData, 'workflowNode'> = {
          id: generateNodeId(kind),
          type: 'workflowNode',
          position,
          data: {
            kind,
            label: `${label} ${nodes.length + 1}`,
            config: { ...defaultConfig },
            status: 'idle',
          },
        };

        addNode(newNode);
      } catch (err) {
        console.warn('[WorkflowBuilder] Invalid drop data', err);
      }
    },
    [rfInstance, addNode, nodes.length]
  );

  // ─── Node Selection (show inspector) ──────────────────────────────

  const onNodeClick = useCallback(
    (_: React.MouseEvent, node: FlowNode) => {
      const data = node.data as WorkflowNodeData;
      selectNode(node.id);
      setSelectedNodeData({ id: node.id, data: { ...data } });
    },
    [selectNode]
  );

  const onPaneClick = useCallback(() => {
    selectNode(null);
    setSelectedNodeData(null);
  }, [selectNode]);

  // ─── Node Remove (Delete key) ─────────────────────────────────────
  const handleNodesDelete = useCallback(
    (deletedNodes: FlowNode[]) => {
      deletedNodes.forEach((n) => removeNode(n.id));
      if (selectedNodeData && deletedNodes.some((n) => n.id === selectedNodeData.id)) {
        setSelectedNodeData(null);
      }
    },
    [removeNode, selectedNodeData]
  );

  // ─── Update node config from inspector ───────────────────────────
  const handleConfigChange = useCallback(
    (field: string, value: unknown) => {
      if (!selectedNodeData) return;
      const newConfig = {
        ...selectedNodeData.data.config,
        [field]: value,
      };
      updateNodeConfig(selectedNodeData.id, newConfig);
      setSelectedNodeData((prev) =>
        prev
          ? { ...prev, data: { ...prev.data, config: newConfig } }
          : null
      );
    },
    [selectedNodeData, updateNodeConfig]
  );

  const handleLabelChange = useCallback(
    (label: string) => {
      if (!selectedNodeData) return;
      setSelectedNodeData((prev) =>
        prev ? { ...prev, data: { ...prev.data, label } } : null
      );
      useWorkflowStore.getState().setNodes(
        useWorkflowStore.getState().nodes.map((n) =>
          n.id === selectedNodeData.id
            ? { ...n, data: { ...n.data, label } }
            : n
        )
      );
    },
    [selectedNodeData]
  );

  // ─── Close sidebar ────────────────────────────────────────────
  const handleCloseSidebar = useCallback(() => setShowSidebar(false), []);

  // ─── Keyboard Shortcuts (Ctrl+Z / Ctrl+Shift+Z) ──────────────────
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
        e.preventDefault();
        if (canUndo()) undo();
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && e.shiftKey) {
        e.preventDefault();
        if (canRedo()) redo();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [undo, redo, canUndo, canRedo]);

  // ─── Track zoom level via React Flow callback ───────────────────
  const onViewportChange = useCallback(
    (viewport: { x: number; y: number; zoom: number }) => {
      setZoomLevel(viewport.zoom);
    },
    []
  );

  // ─── Track save status via isDirty ──────────────────────────────
  useEffect(() => {
    setSaveStatus(isDirty ? 'unsaved' : 'saved');
  }, [isDirty]);

  // ─── Zoom Handlers ───────────────────────────────────────────────
  const handleZoomIn = useCallback(() => {
    rfInstance?.zoomIn({ duration: 200 });
  }, [rfInstance]);

  const handleZoomOut = useCallback(() => {
    rfInstance?.zoomOut({ duration: 200 });
  }, [rfInstance]);

  const handleZoomReset = useCallback(() => {
    rfInstance?.fitView({ padding: 0.3, duration: 200 });
  }, [rfInstance]);

  // ─── Undo / Redo ─────────────────────────────────────────────────
  const handleUndo = useCallback(() => {
    pushHistory();
    undo();
  }, [undo, pushHistory]);

  const handleRedo = useCallback(() => {
    redo();
  }, [redo]);

  // ─── Save to API (new) ──────────────────────────────────────────
  const handleSaveToAPI = useCallback(async () => {
    if (!activeWorkflowId) {
      // If no active workflow, prompt for name
      const name = prompt('Workflow name:');
      if (!name?.trim()) return;
      // Create via store first to get an ID
      useWorkflowStore.getState().createWorkflow(name.trim());
      // Now save
    }
    setSaveStatus('saving');
    try {
      const state = useWorkflowStore.getState();
      const wfId = state.activeWorkflowId;
      if (!wfId) return;

      const workflowDef = state.workflows.find((w) => w.id === wfId);
      if (!workflowDef) return;

      const payload = {
        name: workflowDef.name,
        description: workflowDef.description,
        nodes: state.nodes.map((n) => ({
          ...n,
          data: { ...n.data, status: 'idle', errorMessage: undefined },
        })),
        edges: state.edges,
        tags: workflowDef.tags,
      };

      if (apiWorkflowId) {
        await updateWorkflow(apiWorkflowId, payload as any);
      } else {
        const created = await createWorkflow(payload as any);
        setApiWorkflowId(created.id);
      }
      state.saveCurrentWorkflow();
      setSaveStatus('saved');
    } catch (err) {
      console.error('[WorkflowBuilder] API save failed', err);
      setSaveStatus('unsaved');
      alert('Failed to save workflow to server');
    }
  }, [activeWorkflowId, apiWorkflowId]);

  // ─── Load from API ──────────────────────────────────────────────
  const handleLoadFromAPI = useCallback(async () => {
    const id = prompt('Enter Workflow ID to load:');
    if (!id?.trim()) return;
    try {
      const data = await getWorkflow(id.trim());
      useWorkflowStore.getState().importWorkflow(data as any);
      setApiWorkflowId(id.trim());
      setSaveStatus('saved');
    } catch (err) {
      console.error('[WorkflowBuilder] API load failed', err);
      alert('Failed to load workflow from server');
    }
  }, []);

  // ─── Export as JSON file ─────────────────────────────────────────
  const handleExportJSON = useCallback(() => {
    const state = useWorkflowStore.getState();
    const id = state.activeWorkflowId;
    if (!id) {
      // Export current canvas as ad-hoc JSON
      const data = {
        name: 'Exported Workflow',
        description: '',
        nodes: state.nodes,
        edges: state.edges,
      };
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `workflow_export_${Date.now()}.json`;
      a.click();
      URL.revokeObjectURL(url);
      return;
    }
    const data = state.exportWorkflow(id);
    if (!data) return;
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${data.name.replace(/\s+/g, '_')}_v${data.version}.json`;
    a.click();
    URL.revokeObjectURL(url);
  }, []);

  // ─── Import from JSON file ───────────────────────────────────────
  const handleImportJSON = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = (event) => {
        try {
          const raw = JSON.parse(event.target?.result as string);
          // Validate structure
          if (!raw.nodes || !Array.isArray(raw.nodes) || !raw.edges || !Array.isArray(raw.edges)) {
            alert('Invalid workflow file: missing nodes or edges array');
            return;
          }
          if (!raw.name) {
            raw.name = file.name.replace(/\.json$/i, '');
          }
          useWorkflowStore.getState().importWorkflow(raw);
          setApiWorkflowId(null);
          setSaveStatus('unsaved');
        } catch {
          alert('Failed to parse workflow file');
        }
      };
      reader.readAsText(file);
      if (fileInputRef.current) fileInputRef.current.value = '';
    },
    []
  );

  // ─── Render ───────────────────────────────────────────────────────
  return (
    <div className="flex h-[calc(100vh-4rem)] w-full">
      {/* ─── Left Sidebar (Toolbar + Palette) ──────────────────────── */}
      {showSidebar && <WorkflowToolbar onClose={handleCloseSidebar} />}

      {/* ─── Main Canvas ───────────────────────────────────────────── */}
      <div className="flex-1 relative" ref={reactFlowWrapper}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onConnect={onConnect}
          onInit={setRfInstance}
          onDragOver={onDragOver}
          onDrop={onDrop}
          onNodeClick={onNodeClick}
          onPaneClick={onPaneClick}
          onNodesDelete={handleNodesDelete}
          onViewportChange={onViewportChange}
          nodeTypes={nodeTypes}
          defaultEdgeOptions={defaultEdgeOptions}
          deleteKeyCode={['Backspace', 'Delete']}
          fitView
          fitViewOptions={{ padding: 0.3 }}
          proOptions={{ hideAttribution: true }}
          className="bg-slate-50 dark:bg-slate-900"
        >
          <Background
            variant={BackgroundVariant.Dots}
            gap={20}
            size={1}
            color="#cbd5e1"
          />
          <Controls
            className="!rounded-lg !border !border-slate-200 dark:!border-slate-700 !shadow-sm"
          />
          <MiniMap
            nodeStrokeColor="#3b82f6"
            nodeColor={(node) => {
              const d = node.data as WorkflowNodeData;
              switch (d?.kind) {
                case 'trigger': return '#a855f7';
                case 'condition': return '#f59e0b';
                case 'action': return '#3b82f6';
                case 'delay': return '#14b8a6';
                default: return '#94a3b8';
              }
            }}
            maskColor="rgba(0,0,0,0.1)"
            className="!rounded-lg !border !border-slate-200 dark:!border-slate-700"
          />
        </ReactFlow>

        {/* ─── Top Toolbar Overlay ────────────────────────────────── */}
        <div className="absolute top-4 left-1/2 -translate-x-1/2 z-10 flex items-center gap-1 bg-white/90 dark:bg-slate-800/90 backdrop-blur-sm border border-slate-200 dark:border-slate-700 rounded-lg shadow-sm px-2 py-1.5">
          {/* ─── Toggle Sidebar ──────────────────────────────────── */}
          {!showSidebar && (
            <button
              type="button"
              onClick={() => setShowSidebar(true)}
              className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
              title="Show Components"
            >
              <FileJson className="w-4 h-4" />
            </button>
          )}

          <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />

          {/* ─── Undo ────────────────────────────────────────────── */}
          <button
            type="button"
            onClick={handleUndo}
            disabled={!canUndo()}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400 disabled:opacity-30 disabled:cursor-not-allowed"
            title="Undo (Ctrl+Z)"
          >
            <Undo2 className="w-4 h-4" />
          </button>

          {/* ─── Redo ────────────────────────────────────────────── */}
          <button
            type="button"
            onClick={handleRedo}
            disabled={!canRedo()}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400 disabled:opacity-30 disabled:cursor-not-allowed"
            title="Redo (Ctrl+Shift+Z)"
          >
            <Redo2 className="w-4 h-4" />
          </button>

          <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />

          {/* ─── Zoom Controls ────────────────────────────────────── */}
          <button
            type="button"
            onClick={handleZoomOut}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
            title="Zoom Out"
          >
            <ZoomOut className="w-4 h-4" />
          </button>

          <span className="min-w-[48px] text-center text-xs font-mono text-slate-600 dark:text-slate-400 select-none">
            {Math.round(zoomLevel * 100)}%
          </span>

          <button
            type="button"
            onClick={handleZoomIn}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
            title="Zoom In"
          >
            <ZoomIn className="w-4 h-4" />
          </button>

          <button
            type="button"
            onClick={handleZoomReset}
            className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
            title="Fit View"
          >
            <Maximize2 className="w-4 h-4" />
          </button>

          <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />

          {/* ─── Save to API ──────────────────────────────────────── */}
          <button
            type="button"
            onClick={handleSaveToAPI}
            className="flex items-center gap-1 px-2 py-1.5 rounded text-sm font-medium hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400"
            title="Save to Server"
          >
            <Cloud className="w-4 h-4" />
            Save
          </button>

          {/* ─── Load from API ────────────────────────────────────── */}
          <button
            type="button"
            onClick={handleLoadFromAPI}
            className="flex items-center gap-1 px-2 py-1.5 rounded text-sm font-medium hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400"
            title="Load from Server"
          >
            <CloudOff className="w-4 h-4" />
            Load
          </button>

          <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />

          {/* ─── Export JSON ──────────────────────────────────────── */}
          <button
            type="button"
            onClick={handleExportJSON}
            className="flex items-center gap-1 px-2 py-1.5 rounded text-sm font-medium hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400"
            title="Export as JSON"
          >
            <Download className="w-4 h-4" />
            Export
          </button>

          {/* ─── Import JSON ──────────────────────────────────────── */}
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            className="flex items-center gap-1 px-2 py-1.5 rounded text-sm font-medium hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-600 dark:text-slate-400"
            title="Import from JSON"
          >
            <Upload className="w-4 h-4" />
            Import
          </button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".json"
            onChange={handleImportJSON}
            className="hidden"
          />

          <div className="w-px h-5 bg-slate-200 dark:bg-slate-700 mx-1" />

          {/* ─── Save Status Indicator ────────────────────────────── */}
          <div className="flex items-center gap-1.5 px-2">
            {saveStatus === 'saving' ? (
              <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
            ) : saveStatus === 'unsaved' ? (
              <span className="w-2 h-2 rounded-full bg-amber-500" />
            ) : (
              <span className="w-2 h-2 rounded-full bg-green-500" />
            )}
            <span className="text-[11px] font-medium text-slate-500 dark:text-slate-400">
              {saveStatus === 'saving'
                ? 'Saving...'
                : saveStatus === 'unsaved'
                  ? 'Unsaved'
                  : 'Saved'}
            </span>
          </div>
        </div>

        {/* ─── Test Mode Banner ────────────────────────────────────── */}
        {testMode && (
          <div className="absolute top-16 left-1/2 -translate-x-1/2 z-10 px-4 py-2 bg-amber-500 text-white rounded-lg shadow-lg text-sm font-bold flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-white animate-pulse" />
            Test Mode — Results panel on the right
          </div>
        )}
      </div>

      {/* ─── Right Panel: Inspector or Test Results ────────────────── */}
      {testMode ? (
        <div className="w-96 shrink-0">
          <WorkflowTestPanel />
        </div>
      ) : selectedNodeData ? (
        <div className="w-80 shrink-0 border-l border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 flex flex-col h-full overflow-hidden">
          {/* ─── Inspector Header ──────────────────────────────────── */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700">
            <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
              Node Config
            </h3>
            <button
              type="button"
              onClick={() => setSelectedNodeData(null)}
              className="p-1 rounded hover:bg-slate-100 dark:hover:bg-slate-700"
            >
              <svg className="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* ─── Inspector Content ─────────────────────────────────── */}
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {/* Label */}
            <div className="space-y-1">
              <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
                Label
              </label>
              <input
                type="text"
                value={selectedNodeData.data.label}
                onChange={(e) => handleLabelChange(e.target.value)}
                className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200 focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            {/* Kind badge */}
            <div className="space-y-1">
              <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
                Type
              </label>
              <span className="inline-block px-2 py-1 text-xs font-medium rounded-full bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400">
                {selectedNodeData.data.kind}
              </span>
            </div>

            {/* Kind-specific config */}
            {selectedNodeData.data.kind === 'trigger' && (
              <TriggerConfig
                config={selectedNodeData.data.config}
                onChange={handleConfigChange}
              />
            )}

            {selectedNodeData.data.kind === 'condition' && (
              <ConditionConfig
                config={selectedNodeData.data.config}
                onChange={handleConfigChange}
              />
            )}

            {selectedNodeData.data.kind === 'action' && (
              <ActionConfig
                config={selectedNodeData.data.config}
                onChange={handleConfigChange}
              />
            )}

            {selectedNodeData.data.kind === 'delay' && (
              <DelayConfig
                config={selectedNodeData.data.config}
                onChange={handleConfigChange}
              />
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Config Sub-Components
// ═══════════════════════════════════════════════════════════════════════

interface ConfigProps {
  config: Record<string, unknown>;
  onChange: (field: string, value: unknown) => void;
}

// ─── Trigger Config ───────────────────────────────────────────────────

function TriggerConfig({ config, onChange }: ConfigProps) {
  return (
    <>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Event Type
        </label>
        <select
          value={(config.eventType as string) ?? 'motion_detected'}
          onChange={(e) => onChange('eventType', e.target.value)}
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        >
          <option value="motion_detected">Motion Detected</option>
          <option value="device_offline">Device Offline</option>
          <option value="device_online">Device Online</option>
          <option value="alert_triggered">Alert Triggered</option>
          <option value="schedule">Schedule</option>
          <option value="manual">Manual Trigger</option>
        </select>
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Source
        </label>
        <input
          type="text"
          value={(config.source as string) ?? ''}
          onChange={(e) => onChange('source', e.target.value)}
          placeholder="any, camera_01, sensor_*"
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        />
      </div>
    </>
  );
}

// ─── Condition Config ────────────────────────────────────────────────

function ConditionConfig({ config, onChange }: ConfigProps) {
  return (
    <>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          True Branch Label
        </label>
        <input
          type="text"
          value={(config.trueBranch as string) ?? ''}
          onChange={(e) => onChange('trueBranch', e.target.value)}
          placeholder="e.g., Notify Security"
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          False Branch Label
        </label>
        <input
          type="text"
          value={(config.falseBranch as string) ?? ''}
          onChange={(e) => onChange('falseBranch', e.target.value)}
          placeholder="e.g., Log Only"
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        />
      </div>
      <WorkflowCELInput
        value={(config.celExpression as string) ?? ''}
        onChange={(val) => onChange('celExpression', val)}
        label="CEL Expression"
      />
    </>
  );
}

// ─── Action Config ───────────────────────────────────────────────────

function ActionConfig({ config, onChange }: ConfigProps) {
  return (
    <>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Action Type
        </label>
        <select
          value={(config.actionType as string) ?? 'record'}
          onChange={(e) => onChange('actionType', e.target.value)}
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        >
          <option value="record">Start Recording</option>
          <option value="stop_record">Stop Recording</option>
          <option value="notify">Send Notification</option>
          <option value="ptz">PTZ Command</option>
          <option value="snapshot">Take Snapshot</option>
          <option value="webhook">Call Webhook</option>
          <option value="api">API Call</option>
        </select>
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Target
        </label>
        <input
          type="text"
          value={(config.target as string) ?? ''}
          onChange={(e) => onChange('target', e.target.value)}
          placeholder="camera_01, admin, webhook_url"
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Parameters (JSON)
        </label>
        <textarea
          value={JSON.stringify((config.params as Record<string, unknown>) ?? {}, null, 2)}
          onChange={(e) => {
            try {
              onChange('params', JSON.parse(e.target.value));
            } catch {
              // Allow editing even if invalid JSON
            }
          }}
          className="w-full h-20 text-xs font-mono border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200 p-2 resize-y"
        />
      </div>
    </>
  );
}

// ─── Delay Config ────────────────────────────────────────────────────

function DelayConfig({ config, onChange }: ConfigProps) {
  return (
    <>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Duration
        </label>
        <input
          type="number"
          value={Number(config.duration ?? 30)}
          onChange={(e) => onChange('duration', parseInt(e.target.value, 10) || 0)}
          min={1}
          max={86400}
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        />
      </div>
      <div className="space-y-1">
        <label className="text-xs font-medium text-slate-500 dark:text-slate-400">
          Unit
        </label>
        <select
          value={(config.unit as string) ?? 'seconds'}
          onChange={(e) => onChange('unit', e.target.value)}
          className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-900 text-slate-800 dark:text-slate-200"
        >
          <option value="seconds">Seconds</option>
          <option value="minutes">Minutes</option>
          <option value="hours">Hours</option>
        </select>
      </div>
    </>
  );
}
