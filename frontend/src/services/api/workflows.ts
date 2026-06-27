// ═══════════════════════════════════════════════════════════════════════
// Workflows API — P2-2.1: Workflow Builder CRUD
//
// REST endpoints for workflow definitions:
//   GET    /workflows        — list all
//   GET    /workflows/:id     — get one
//   POST   /workflows        — create
//   PUT    /workflows/:id     — update
//   DELETE /workflows/:id     — delete
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ───────────────────────────────────────────────────────────────

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

export interface WorkflowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: Record<string, unknown>;
  selected?: boolean;
}

export interface WorkflowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
  label?: string;
  condition?: string;
}

export interface WorkflowListResponse {
  workflows: WorkflowDefinition[];
  total: number;
}

// ─── API Calls ───────────────────────────────────────────────────────────

/** Get all workflows */
export async function getWorkflows(): Promise<WorkflowListResponse> {
  return request<WorkflowListResponse>('/workflows');
}

/** Get single workflow by ID */
export async function getWorkflow(id: string): Promise<WorkflowDefinition> {
  return request<WorkflowDefinition>(`/workflows/${id}`);
}

/** Create a new workflow */
export async function createWorkflow(
  data: Omit<WorkflowDefinition, 'id' | 'createdAt' | 'updatedAt' | 'version'>
): Promise<WorkflowDefinition> {
  return request<WorkflowDefinition>('/workflows', {
    method: 'POST',
    body: JSON.stringify(data),
  });
}

/** Update an existing workflow */
export async function updateWorkflow(
  id: string,
  data: Partial<WorkflowDefinition>
): Promise<WorkflowDefinition> {
  return request<WorkflowDefinition>(`/workflows/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

/** Delete a workflow */
export async function deleteWorkflow(id: string): Promise<void> {
  await request<void>(`/workflows/${id}`, {
    method: 'DELETE',
  });
}
