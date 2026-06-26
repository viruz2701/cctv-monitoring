// ═══════════════════════════════════════════════════════════════════════
// Workflow Store — P2-2.1: Workflow Builder UI
//
// Zustand store с persist middleware для:
//   - Управление workflow definitions (CRUD)
//   - Version control (save/load версий)
//   - Active workflow state для React Flow
//   - Export/Import workflow как JSON
//   - Test mode state
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import {
  type WorkflowDefinition,
  type WorkflowVersion,
  type WorkflowNode,
  type WorkflowEdge,
  type WorkflowTestRun,
  type WorkflowNodeStatus,
} from '../types/workflow';

// ═══════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════

interface WorkflowState {
  // ─── Workflow Definitions ───────────────────────────────────────────
  workflows: WorkflowDefinition[];
  activeWorkflowId: string | null;

  // ─── Version History ────────────────────────────────────────────────
  versions: Record<string, WorkflowVersion[]>;  // workflowId → versions

  // ─── Test Mode ──────────────────────────────────────────────────────
  testMode: boolean;
  currentTestRun: WorkflowTestRun | null;

  // ─── Editing State ──────────────────────────────────────────────────
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  selectedNodeId: string | null;
  isDirty: boolean;

  // ─── CRUD ───────────────────────────────────────────────────────────
  createWorkflow: (name: string, description?: string) => string;
  updateWorkflow: (id: string, data: Partial<WorkflowDefinition>) => void;
  deleteWorkflow: (id: string) => void;
  setActiveWorkflow: (id: string | null) => void;

  // ─── Graph Editing ──────────────────────────────────────────────────
  setNodes: (nodes: WorkflowNode[]) => void;
  setEdges: (edges: WorkflowEdge[]) => void;
  onNodesChange: (changes: any) => void;
  onEdgesChange: (changes: any) => void;
  onConnect: (connection: any) => void;
  selectNode: (nodeId: string | null) => void;
  addNode: (node: WorkflowNode) => void;
  removeNode: (nodeId: string) => void;
  updateNodeConfig: (nodeId: string, config: Record<string, unknown>) => void;
  updateNodeStatus: (nodeId: string, status: WorkflowNodeStatus, errorMessage?: string) => void;

  // ─── Persistence ────────────────────────────────────────────────────
  saveCurrentWorkflow: () => void;
  loadWorkflow: (id: string) => void;
  resetEditor: () => void;

  // ─── Version Control ────────────────────────────────────────────────
  saveVersion: (message?: string) => void;
  loadVersion: (workflowId: string, versionId: string) => void;
  getVersions: (workflowId: string) => WorkflowVersion[];

  // ─── Export / Import ────────────────────────────────────────────────
  exportWorkflow: (id: string) => WorkflowDefinition | null;
  importWorkflow: (data: WorkflowDefinition) => string;

  // ─── Test Mode ──────────────────────────────────────────────────────
  toggleTestMode: () => void;
  startTestRun: (mockEvent?: Record<string, unknown>) => void;
  completeTestRun: () => void;
  clearTestResults: () => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

const generateId = (): string =>
  `wf-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;

const generateNodeId = (): string =>
  `node-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;

const createDefaultNodes = (): WorkflowNode[] => [
  {
    id: 'trigger-1',
    type: 'workflowNode',
    position: { x: 250, y: 0 },
    data: {
      kind: 'trigger',
      label: 'Motion Detected',
      config: { eventType: 'motion_detected', source: 'any' },
      status: 'idle',
    },
  },
  {
    id: 'condition-1',
    type: 'workflowNode',
    position: { x: 250, y: 150 },
    data: {
      kind: 'condition',
      label: 'Is Critical Area?',
      config: { celExpression: 'event.priority == "critical"', trueBranch: '', falseBranch: '' },
      status: 'idle',
    },
  },
  {
    id: 'action-1',
    type: 'workflowNode',
    position: { x: 100, y: 300 },
    data: {
      kind: 'action',
      label: 'Start Recording',
      config: { actionType: 'record', target: 'camera_01', params: { duration: 300 } },
      status: 'idle',
    },
  },
  {
    id: 'action-2',
    type: 'workflowNode',
    position: { x: 400, y: 300 },
    data: {
      kind: 'action',
      label: 'Send Alert',
      config: { actionType: 'notify', target: 'admin', params: { channel: 'email' } },
      status: 'idle',
    },
  },
  {
    id: 'delay-1',
    type: 'workflowNode',
    position: { x: 250, y: 450 },
    data: {
      kind: 'delay',
      label: 'Wait 30s',
      config: { duration: 30, unit: 'seconds' },
      status: 'idle',
    },
  },
];

const createDefaultEdges = (): WorkflowEdge[] => [
  {
    id: 'e-trigger-condition',
    source: 'trigger-1',
    target: 'condition-1',
  },
  {
    id: 'e-condition-action1',
    source: 'condition-1',
    target: 'action-1',
    label: 'true',
  },
  {
    id: 'e-condition-action2',
    source: 'condition-1',
    target: 'action-2',
    label: 'false',
  },
  {
    id: 'e-action1-delay',
    source: 'action-1',
    target: 'delay-1',
  },
];

// ═══════════════════════════════════════════════════════════════════════
// Store
// ═══════════════════════════════════════════════════════════════════════

export const useWorkflowStore = create<WorkflowState>()(
  persist(
    (set, get) => ({
      // ─── Initial State ──────────────────────────────────────────────
      workflows: [],
      activeWorkflowId: null,
      versions: {},
      testMode: false,
      currentTestRun: null,
      nodes: createDefaultNodes(),
      edges: createDefaultEdges(),
      selectedNodeId: null,
      isDirty: false,

      // ─── CRUD ───────────────────────────────────────────────────────

      createWorkflow: (name, description = '') => {
        const id = generateId();
        const now = new Date().toISOString();
        const { nodes, edges } = get();
        const workflow: WorkflowDefinition = {
          id,
          name,
          description,
          nodes: nodes.map((n) => ({ ...n, data: { ...n.data, status: 'idle', errorMessage: undefined } })),
          edges,
          createdAt: now,
          updatedAt: now,
          version: 1,
        };
        set((state) => ({
          workflows: [...state.workflows, workflow],
          activeWorkflowId: id,
          isDirty: false,
        }));
        return id;
      },

      updateWorkflow: (id, data) => {
        set((state) => ({
          workflows: state.workflows.map((w) =>
            w.id === id ? { ...w, ...data, updatedAt: new Date().toISOString() } : w
          ),
          isDirty: true,
        }));
      },

      deleteWorkflow: (id) => {
        set((state) => {
          const filtered = state.workflows.filter((w) => w.id !== id);
          return {
            workflows: filtered,
            activeWorkflowId:
              state.activeWorkflowId === id
                ? filtered[0]?.id ?? null
                : state.activeWorkflowId,
          };
        });
      },

      setActiveWorkflow: (id) => {
        if (id) {
          get().loadWorkflow(id);
        }
        set({ activeWorkflowId: id });
      },

      // ─── Graph Editing ──────────────────────────────────────────────

      setNodes: (nodes) => set({ nodes, isDirty: true }),

      setEdges: (edges) => set({ edges, isDirty: true }),

      onNodesChange: (changes) => {
        set((state) => {
          const newNodes = state.nodes.map((node) => {
            const change = changes.find((c: any) => c.id === node.id);
            if (!change) return node;
            if (change.type === 'position' && change.position) {
              return { ...node, position: change.position };
            }
            if (change.type === 'select') {
              return { ...node, selected: change.selected };
            }
            if (change.type === 'remove') {
              return null;
            }
            return node;
          }).filter(Boolean) as WorkflowNode[];
          return { nodes: newNodes, isDirty: true };
        });
      },

      onEdgesChange: (changes) => {
        set((state) => {
          const newEdges = state.edges
            .map((edge) => {
              const change = changes.find((c: any) => c.id === edge.id);
              if (!change) return edge;
              if (change.type === 'remove') return null;
              if (change.type === 'select') {
                return { ...edge, selected: change.selected };
              }
              return edge;
            })
            .filter(Boolean) as WorkflowEdge[];
          return { edges: newEdges, isDirty: true };
        });
      },

      onConnect: (connection) => {
        const newEdge: WorkflowEdge = {
          id: `e-${connection.source}-${connection.target}`,
          source: connection.source,
          target: connection.target,
          sourceHandle: connection.sourceHandle ?? undefined,
          targetHandle: connection.targetHandle ?? undefined,
        };
        set((state) => ({
          edges: [...state.edges, newEdge],
          isDirty: true,
        }));
      },

      selectNode: (nodeId) => set({ selectedNodeId: nodeId }),

      addNode: (node) => {
        set((state) => ({
          nodes: [...state.nodes, node],
          isDirty: true,
        }));
      },

      removeNode: (nodeId) => {
        set((state) => ({
          nodes: state.nodes.filter((n) => n.id !== nodeId),
          edges: state.edges.filter(
            (e) => e.source !== nodeId && e.target !== nodeId
          ),
          selectedNodeId:
            state.selectedNodeId === nodeId ? null : state.selectedNodeId,
          isDirty: true,
        }));
      },

      updateNodeConfig: (nodeId, config) => {
        set((state) => ({
          nodes: state.nodes.map((n) =>
            n.id === nodeId
              ? { ...n, data: { ...n.data, config: { ...n.data.config, ...config } } }
              : n
          ),
          isDirty: true,
        }));
      },

      updateNodeStatus: (nodeId, status, errorMessage) => {
        set((state) => ({
          nodes: state.nodes.map((n) =>
            n.id === nodeId
              ? { ...n, data: { ...n.data, status, errorMessage } }
              : n
          ),
        }));
      },

      // ─── Persistence ────────────────────────────────────────────────

      saveCurrentWorkflow: () => {
        const { activeWorkflowId, nodes, edges, workflows } = get();
        if (!activeWorkflowId) return;

        const now = new Date().toISOString();
        set((state) => ({
          workflows: state.workflows.map((w) =>
            w.id === activeWorkflowId
              ? {
                  ...w,
                  nodes: nodes.map((n) => ({
                    ...n,
                    data: { ...n.data, status: 'idle' as const, errorMessage: undefined },
                  })),
                  edges,
                  updatedAt: now,
                  version: w.version + 1,
                }
              : w
          ),
          isDirty: false,
        }));
      },

      loadWorkflow: (id) => {
        const workflow = get().workflows.find((w) => w.id === id);
        if (workflow) {
          set({
            nodes: workflow.nodes,
            edges: workflow.edges,
            isDirty: false,
            selectedNodeId: null,
          });
        }
      },

      resetEditor: () => {
        set({
          nodes: createDefaultNodes(),
          edges: createDefaultEdges(),
          selectedNodeId: null,
          isDirty: false,
          activeWorkflowId: null,
        });
      },

      // ─── Version Control ────────────────────────────────────────────

      saveVersion: (message) => {
        const { activeWorkflowId, nodes, edges, workflows } = get();
        if (!activeWorkflowId) return;

        const workflow = workflows.find((w) => w.id === activeWorkflowId);
        if (!workflow) return;

        const snapshot: WorkflowDefinition = {
          ...workflow,
          nodes: nodes.map((n) => ({
            ...n,
            data: { ...n.data, status: 'idle' as const, errorMessage: undefined },
          })),
          edges,
          updatedAt: new Date().toISOString(),
          version: workflow.version,
        };

        const versionEntry: WorkflowVersion = {
          id: generateId(),
          workflowId: activeWorkflowId,
          version: (get().versions[activeWorkflowId]?.length ?? 0) + 1,
          snapshot,
          createdAt: new Date().toISOString(),
          message,
        };

        set((state) => ({
          versions: {
            ...state.versions,
            [activeWorkflowId]: [
              ...(state.versions[activeWorkflowId] ?? []),
              versionEntry,
            ],
          },
          isDirty: false,
        }));
      },

      loadVersion: (workflowId, versionId) => {
        const version = get().versions[workflowId]?.find(
          (v) => v.id === versionId
        );
        if (version) {
          set({
            nodes: version.snapshot.nodes,
            edges: version.snapshot.edges,
            isDirty: false,
          });
        }
      },

      getVersions: (workflowId) => {
        return get().versions[workflowId] ?? [];
      },

      // ─── Export / Import ────────────────────────────────────────────

      exportWorkflow: (id) => {
        const workflow = get().workflows.find((w) => w.id === id);
        if (!workflow) return null;

        const { nodes, edges } = get();
        // Если экспортируем активный workflow — берём текущее состояние
        if (id === get().activeWorkflowId) {
          return {
            ...workflow,
            nodes: nodes.map((n) => ({
              ...n,
              data: { ...n.data, status: 'idle' as const, errorMessage: undefined },
            })),
            edges,
            updatedAt: new Date().toISOString(),
          };
        }
        return workflow;
      },

      importWorkflow: (data) => {
        const id = generateId();
        const now = new Date().toISOString();
        const workflow: WorkflowDefinition = {
          ...data,
          id,
          createdAt: now,
          updatedAt: now,
          version: 1,
        };
        set((state) => ({
          workflows: [...state.workflows, workflow],
          activeWorkflowId: id,
          nodes: workflow.nodes,
          edges: workflow.edges,
          isDirty: false,
        }));
        return id;
      },

      // ─── Test Mode ──────────────────────────────────────────────────

      toggleTestMode: () => {
        const { testMode } = get();
        if (testMode) {
          // Сброс статусов при выходе из тестового режима
          set((state) => ({
            testMode: false,
            currentTestRun: null,
            nodes: state.nodes.map((n) => ({
              ...n,
              data: { ...n.data, status: 'idle' as const, errorMessage: undefined },
            })),
          }));
        } else {
          set({ testMode: true });
        }
      },

      startTestRun: (mockEvent) => {
        const { activeWorkflowId } = get();
        const run: WorkflowTestRun = {
          id: generateId(),
          workflowId: activeWorkflowId ?? 'unknown',
          status: 'running',
          results: [],
          startedAt: new Date().toISOString(),
          mockEvent,
        };
        set({ currentTestRun: run });
      },

      completeTestRun: () => {
        set((state) => {
          if (!state.currentTestRun) return state;
          return {
            currentTestRun: {
              ...state.currentTestRun,
              status: 'completed',
              completedAt: new Date().toISOString(),
            },
          };
        });
      },

      clearTestResults: () => {
        set((state) => ({
          currentTestRun: null,
          nodes: state.nodes.map((n) => ({
            ...n,
            data: { ...n.data, status: 'idle' as const, errorMessage: undefined },
          })),
        }));
      },
    }),
    {
      name: 'cctv-workflows',
      partialize: (state) => ({
        workflows: state.workflows,
        versions: state.versions,
        activeWorkflowId: state.activeWorkflowId,
      }),
    }
  )
);
