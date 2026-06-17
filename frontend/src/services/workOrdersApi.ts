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
};
