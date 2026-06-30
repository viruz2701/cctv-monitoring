// P2-CHECK: Conditional Checklists API service
import { request } from './api';

// ── Types ────────────────────────────────────────────────────────────

export interface Condition {
  field_id: string;
  operator: 'eq' | 'neq' | 'gt' | 'lt' | 'gte' | 'lte' | 'in';
  value: unknown;
}

export interface ChecklistItem {
  id: string;
  template_id: string;
  parent_id?: string;
  label: string;
  description: string;
  item_type: 'boolean' | 'text' | 'photo' | 'numeric' | 'signature' | 'select' | 'multi_select';
  mandatory: boolean;
  score: number;
  sort_order: number;
  options?: string[] | null;
  validation_min?: number | null;
  validation_max?: number | null;
  depends_on?: Condition | null;
  children?: ChecklistItem[];
  created_at: string;
  updated_at: string;
}

export interface ChecklistTemplate {
  id: string;
  name: string;
  description: string;
  device_types: string[];
  pass_threshold: number;
  is_active: boolean;
  items?: ChecklistItem[];
  created_at: string;
  updated_at: string;
}

export interface CreateTemplateRequest {
  name: string;
  description?: string;
  device_types: string[];
  pass_threshold?: number;
  items?: ChecklistItem[];
}

export interface WorkOrderChecklist {
  id: string;
  work_order_id: string;
  template_id: string;
  status: 'in_progress' | 'submitted' | 'verified';
  total_score: number;
  max_score: number;
  score_percent: number;
  passed: boolean;
  started_by: string;
  started_at: string;
  submitted_by?: string;
  submitted_at?: string;
  verified_by?: string;
  verified_at?: string;
  notes: string;
  responses?: ChecklistResponse[];
  template_name?: string;
  created_at: string;
  updated_at: string;
}

export interface ChecklistResponse {
  id: string;
  checklist_id: string;
  item_id: string;
  value: string;
  photo_url?: string;
  skipped: boolean;
  created_at: string;
  updated_at: string;
}

export interface SubmitItemResponse {
  item_id: string;
  value: string;
  photo_url?: string;
  skipped?: boolean;
}

export interface SubmitChecklistRequest {
  responses: SubmitItemResponse[];
  notes?: string;
}

export interface StartChecklistRequest {
  template_id: string;
}

export interface ChecklistSummary {
  id: string;
  work_order_id: string;
  template_name: string;
  status: string;
  score_percent: number;
  passed: boolean;
  total_items: number;
  completed_items: number;
  skipped_items: number;
  started_by: string;
  started_at: string;
  submitted_by?: string;
  submitted_at?: string;
}

// ── API Methods ──────────────────────────────────────────────────────

export const checklistApi = {
  // Templates
  listTemplates: (params?: { device_type?: string; active_only?: boolean; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams();
    if (params?.device_type) searchParams.set('device_type', params.device_type);
    if (params?.active_only) searchParams.set('active_only', 'true');
    if (params?.limit) searchParams.set('limit', String(params.limit));
    if (params?.offset) searchParams.set('offset', String(params.offset));
    const qs = searchParams.toString();
    return request<ChecklistTemplate[]>(`/checklists/templates${qs ? '?' + qs : ''}`);
  },

  getTemplate: (id: string) => {
    return request<ChecklistTemplate>(`/checklists/templates/${id}`);
  },

  createTemplate: (data: CreateTemplateRequest) => {
    return request<ChecklistTemplate>('/checklists/templates', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateTemplate: (id: string, data: CreateTemplateRequest) => {
    return request<ChecklistTemplate>(`/checklists/templates/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteTemplate: (id: string) => {
    return request<{ status: string }>(`/checklists/templates/${id}`, {
      method: 'DELETE',
    });
  },

  // Work Order Checklists
  getWorkOrderChecklist: (workOrderId: string) => {
    return request<WorkOrderChecklist>(`/work-orders/${workOrderId}/checklist`);
  },

  startChecklist: (workOrderId: string, data: StartChecklistRequest) => {
    return request<WorkOrderChecklist>(`/work-orders/${workOrderId}/checklist/start`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  submitChecklist: (workOrderId: string, data: SubmitChecklistRequest) => {
    return request<ChecklistSummary>(`/work-orders/${workOrderId}/checklist/submit`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },
};
