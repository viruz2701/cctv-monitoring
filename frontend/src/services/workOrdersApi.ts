import { request } from './api';

export interface WorkOrder {
  id: string;
  schedule_id?: string;
  device_id: string;
  type: 'preventive' | 'corrective' | 'emergency';
  status: 'open' | 'in_progress' | 'completed' | 'cancelled';
  priority: 'critical' | 'high' | 'medium' | 'low';
  assigned_to?: string;
  sla_deadline?: string;
  checklist: ChecklistItem[];
  started_at?: string;
  completed_at?: string;
  notes?: string;
  photos: string[];
  parts_used: PartUsage[];
  created_by?: string;
  created_at: string;
  updated_at: string;
  device_name?: string;
  assignee_name?: string;
  sla_status?: 'on_track' | 'at_risk' | 'breached' | 'completed' | 'no_sla';
  total_labor_cost?: number;
  total_parts_cost?: number;
  total_cost?: number;
}

export interface ChecklistItem {
  task: string;
  completed: boolean;
}

export interface PartUsage {
  part_id: string;
  quantity: number;
}

export interface CreateWorkOrderRequest {
  device_id: string;
  type: string;
  priority?: string;
  assigned_to?: string;
  checklist?: ChecklistItem[];
  notes?: string;
}

// ── TimeEntry Types ──────────────────────────────────────────────────

export interface TimeEntry {
  id: string;
  work_order_id: string;
  started_at: string;
  paused_at?: string;
  resumed_at?: string;
  stopped_at?: string;
  notes?: string;
  hourly_rate: number;
  total_seconds: number;
  status: 'running' | 'paused' | 'stopped';
}

export interface CreateTimeEntryRequest {
  notes?: string;
  hourly_rate: number;
}

// ── LaborCost Types ──────────────────────────────────────────────────

export interface LaborCost {
  total_hours: number;
  hourly_rate: number;
  total_cost: number;
  currency: string;
}

// ── PartWithCost Types ───────────────────────────────────────────────

export interface PartWithCostRequest {
  part_id: string;
  quantity: number;
}

export const workOrdersApi = {
  getWorkOrders: (filters?: Record<string, string>) => {
    const params = new URLSearchParams(filters).toString();
    return request<WorkOrder[]>(`/work-orders${params ? '?' + params : ''}`);
  },

  getWorkOrder: (id: string) => {
    return request<WorkOrder>(`/work-orders/${id}`);
  },

  createWorkOrder: (data: CreateWorkOrderRequest) => {
    return request<WorkOrder>('/work-orders', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateWorkOrder: (id: string, data: Partial<WorkOrder>) => {
    return request<{ status: string }>(`/work-orders/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteWorkOrder: (id: string) => {
    return request<{ status: string }>(`/work-orders/${id}`, {
      method: 'DELETE',
    });
  },

  assignWorkOrder: (id: string, userId: string) => {
    return request<{ status: string }>(`/work-orders/${id}/assign`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    });
  },

  startWorkOrder: (id: string) => {
    return request<{ status: string }>(`/work-orders/${id}/start`, {
      method: 'POST',
    });
  },

  completeWorkOrder: (id: string, notes: string, photos: string[], parts: PartUsage[]) => {
    return request<{ status: string }>(`/work-orders/${id}/complete`, {
      method: 'POST',
      body: JSON.stringify({ notes, photos, parts }),
    });
  },

  cancelWorkOrder: (id: string, reason: string) => {
    return request<{ status: string }>(`/work-orders/${id}/cancel`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    });
  },

  uploadPhotos: (id: string, files: File[]) => {
    const formData = new FormData();
    files.forEach((file) => formData.append('photos', file));
    return request<{ photos: string[] }>(`/work-orders/${id}/photos`, {
      method: 'POST',
      body: formData,
    });
  },

  addParts: (id: string, parts: PartUsage[]) => {
    return request<{ status: string }>(`/work-orders/${id}/parts`, {
      method: 'POST',
      body: JSON.stringify({ parts }),
    });
  },

  // ── Time Entries (WO-5.1) ────────────────────────────────────────

  getTimeEntries: (workOrderId: string) => {
    return request<TimeEntry[]>(`/work-orders/${workOrderId}/time-entries`);
  },

  createTimeEntry: (workOrderId: string, data: CreateTimeEntryRequest) => {
    return request<TimeEntry>(`/work-orders/${workOrderId}/time-entries`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  pauseTimeEntry: (id: string) => {
    return request<TimeEntry>(`/time-entries/${id}/pause`, {
      method: 'PUT',
    });
  },

  resumeTimeEntry: (id: string) => {
    return request<TimeEntry>(`/time-entries/${id}/resume`, {
      method: 'PUT',
    });
  },

  stopTimeEntry: (id: string) => {
    return request<TimeEntry>(`/time-entries/${id}/stop`, {
      method: 'PUT',
    });
  },

  deleteTimeEntry: (id: string) => {
    return request<{ status: string }>(`/time-entries/${id}`, {
      method: 'DELETE',
    });
  },

  // ── Labor Cost (WO-5.2) ──────────────────────────────────────────

  getLaborCost: (workOrderId: string) => {
    return request<LaborCost>(`/work-orders/${workOrderId}/labor-cost`);
  },

  // ── Parts With Cost (WO-5.3) ─────────────────────────────────────

  addPartWithCost: (workOrderId: string, data: PartWithCostRequest) => {
    return request<{ status: string }>(`/work-orders/${workOrderId}/parts-with-cost`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  // ── Bulk Actions (WO-4.2.1) ──────────────────────────────────────

  bulkActions: (action: string, ids: string[], value?: string) => {
    return request<{
      results: { id: string; status: 'success' | 'error'; error?: string }[];
      total: number;
      success: number;
      failed: number;
    }>('/work-orders/bulk', {
      method: 'POST',
      body: JSON.stringify({ action, ids, value }),
    });
  },
};
