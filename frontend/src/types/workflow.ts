// ═══════════════════════════════════════════════════════════════════════
// Workflow Types — P2-2.1: Workflow Builder UI
//
// Определяет типы для построителя workflow с drag&drop на React Flow:
//   - Node types: Trigger, Condition, Action, Delay
//   - Edge types: стандартные + CEL-condition
//   - WorkflowDefinition: полный граф для export/import
//   - WorkflowVersion: история версий
// ═══════════════════════════════════════════════════════════════════════

import type { Node, Edge } from '@xyflow/react';

// ─── Node Types ─────────────────────────────────────────────────────────

export type WorkflowNodeKind =
  | 'trigger'
  | 'condition'
  | 'action'
  | 'delay';

export type WorkflowNodeStatus =
  | 'idle'
  | 'running'
  | 'success'
  | 'error';

// ─── Node Data (хранится в node.data) ────────────────────────────────────

export interface WorkflowNodeData {
  [key: string]: unknown;
  kind: WorkflowNodeKind;
  label: string;
  description?: string;
  config: Record<string, unknown>;
  status?: WorkflowNodeStatus;
  errorMessage?: string;
}

// ─── Node Types for React Flow ──────────────────────────────────────────

export type WorkflowNode = Node<WorkflowNodeData, 'workflowNode'>;
export type WorkflowEdge = Edge<{ condition?: string; label?: string }>;

// ─── Palette Item (для sidebar drag&drop) ───────────────────────────────

export interface PaletteItem {
  kind: WorkflowNodeKind;
  label: string;
  description: string;
  icon: string;          // lucide-react icon name
  defaultConfig: Record<string, unknown>;
  color: string;         // Tailwind border/accent color
}

// ─── Workflow Definition (полный граф) ──────────────────────────────────

export interface WorkflowDefinition {
  id: string;
  name: string;
  description: string;
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
  createdAt: string;
  updatedAt: string;
  version: number;
  tags?: string[];
}

// ─── Workflow Version ───────────────────────────────────────────────────

export interface WorkflowVersion {
  id: string;
  workflowId: string;
  version: number;
  snapshot: WorkflowDefinition;
  createdAt: string;
  message?: string;
}

// ─── Test Run ───────────────────────────────────────────────────────────

export interface WorkflowTestRun {
  id: string;
  workflowId: string;
  status: 'running' | 'completed' | 'failed';
  results: WorkflowTestResult[];
  startedAt: string;
  completedAt?: string;
  mockEvent?: Record<string, unknown>;
}

export interface WorkflowTestResult {
  nodeId: string;
  status: WorkflowNodeStatus;
  output?: unknown;
  error?: string;
  durationMs: number;
}

// ─── Palette presets ────────────────────────────────────────────────────

export const WORKFLOW_PALETTE: PaletteItem[] = [
  {
    kind: 'trigger',
    label: 'Trigger',
    description: 'Start workflow on event (motion, alert, schedule)',
    icon: 'Zap',
    defaultConfig: { eventType: 'motion_detected', source: 'any' },
    color: 'border-purple-500 bg-purple-50 dark:bg-purple-950/20',
  },
  {
    kind: 'condition',
    label: 'Condition',
    description: 'Branch logic with CEL expression',
    icon: 'GitBranch',
    defaultConfig: { celExpression: '', trueBranch: '', falseBranch: '' },
    color: 'border-amber-500 bg-amber-50 dark:bg-amber-950/20',
  },
  {
    kind: 'action',
    label: 'Action',
    description: 'Execute an action (record, notify, PTZ)',
    icon: 'Play',
    defaultConfig: { actionType: 'record', target: '', params: {} },
    color: 'border-blue-500 bg-blue-50 dark:bg-blue-950/20',
  },
  {
    kind: 'delay',
    label: 'Delay',
    description: 'Wait for a specified duration',
    icon: 'Timer',
    defaultConfig: { duration: 30, unit: 'seconds' },
    color: 'border-teal-500 bg-teal-50 dark:bg-teal-950/20',
  },
];
