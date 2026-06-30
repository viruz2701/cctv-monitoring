// ═══════════════════════════════════════════════════════════════════════
// workOrders.ts — Work Orders API service
// ARCH.2: Модульный API сервис для work orders
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';
import type { AnnotationElement } from '../../components/work-orders/annotationTypes';

// ── Types ──────────────────────────────────────────────────────────────

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

export interface LaborCost {
  total_hours: number;
  hourly_rate: number;
  total_cost: number;
  currency: string;
}

export interface PartWithCostRequest {
  part_id: string;
  quantity: number;
}

// ── Annotation types ──────────────────────────────────────────────────

export interface AnnotationSaveRequest {
  elements: AnnotationElement[];
}

export interface AnnotationResponse {
  id: string;
  work_order_id: string;
  photo_url: string;
  elements: AnnotationElement[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

// ── API service ───────────────────────────────────────────────────────

export const workOrdersApi = {
  // ── Work Orders CRUD ─────────────────────────────────────────────
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

  // ── Annotations (P1-PHOTO) ────────────────────────────────────────

  getAnnotations: (workOrderId: string, photoUrl: string) => {
    const encodedPhoto = encodeURIComponent(photoUrl);
    return request<AnnotationResponse>(
      `/work-orders/${workOrderId}/photos/${encodedPhoto}/annotations`,
    );
  },

  saveAnnotations: (workOrderId: string, photoUrl: string, data: AnnotationSaveRequest) => {
    const encodedPhoto = encodeURIComponent(photoUrl);
    return request<AnnotationResponse>(
      `/work-orders/${workOrderId}/photos/${encodedPhoto}/annotations`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      },
    );
  },

  updateAnnotations: (workOrderId: string, photoUrl: string, data: AnnotationSaveRequest) => {
    const encodedPhoto = encodeURIComponent(photoUrl);
    return request<AnnotationResponse>(
      `/work-orders/${workOrderId}/photos/${encodedPhoto}/annotations`,
      {
        method: 'PUT',
        body: JSON.stringify(data),
      },
    );
  },

  // ── Time Entries ──────────────────────────────────────────────────
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

  // ── Labor Cost ────────────────────────────────────────────────────
  getLaborCost: (workOrderId: string) => {
    return request<LaborCost>(`/work-orders/${workOrderId}/labor-cost`);
  },

  // ── Parts With Cost ───────────────────────────────────────────────
  addPartWithCost: (workOrderId: string, data: PartWithCostRequest) => {
    return request<{ status: string }>(`/work-orders/${workOrderId}/parts-with-cost`, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  // ── Bulk Actions ──────────────────────────────────────────────────
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
