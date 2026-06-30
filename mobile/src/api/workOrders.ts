import { apiClient } from './client';
import { WorkOrder, CompleteWorkOrderPayload } from '../types';

// ── Annotation types ──────────────────────────────────────────────────

export interface AnnotationElement {
  id: string;
  type: 'arrow' | 'freehand' | 'text' | 'highlight' | 'circle' | 'blur' | 'measurement';
  color: string;
  strokeWidth: number;
  [key: string]: unknown;
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

export const workOrdersApi = {
  getMyWorkOrders: async (): Promise<WorkOrder[]> => {
    const response = await apiClient.get<WorkOrder[]>('/mobile/work-orders');
    return response.data;
  },

  getWorkOrder: async (id: string): Promise<WorkOrder> => {
    const response = await apiClient.get<WorkOrder>(`/mobile/work-orders/${id}`);
    return response.data;
  },

  startWorkOrder: async (id: string): Promise<WorkOrder> => {
    const response = await apiClient.post<WorkOrder>(`/mobile/work-orders/${id}/start`);
    return response.data;
  },

  completeWorkOrder: async (
    id: string,
    payload: CompleteWorkOrderPayload,
  ): Promise<WorkOrder> => {
    const response = await apiClient.post<WorkOrder>(
      `/mobile/work-orders/${id}/complete`,
      payload,
    );
    return response.data;
  },

  uploadPhoto: async (
    workOrderId: string,
    photoUri: string,
  ): Promise<{ url: string }> => {
    const formData = new FormData();
    formData.append('photo', {
      uri: photoUri,
      type: 'image/jpeg',
      name: `wo_${workOrderId}_${Date.now()}.jpg`,
    } as any);

    const response = await apiClient.post(
      `/mobile/work-orders/${workOrderId}/photos`,
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
      },
    );
    return response.data;
  },

  getTechnicianProfile: async () => {
    const response = await apiClient.get('/mobile/technician/profile');
    return response.data;
  },

  getTechnicianStats: async () => {
    const response = await apiClient.get('/mobile/technician/stats');
    return response.data;
  },

  // ── Annotations (P1-PHOTO) ────────────────────────────────────────

  getAnnotations: async (
    workOrderId: string,
    photoUrl: string,
  ): Promise<AnnotationResponse> => {
    const encodedPhoto = encodeURIComponent(photoUrl);
    const response = await apiClient.get<AnnotationResponse>(
      `/work-orders/${workOrderId}/photos/${encodedPhoto}/annotations`,
    );
    return response.data;
  },

  saveAnnotations: async (
    workOrderId: string,
    photoUrl: string,
    elements: AnnotationElement[],
  ): Promise<AnnotationResponse> => {
    const encodedPhoto = encodeURIComponent(photoUrl);
    const response = await apiClient.post<AnnotationResponse>(
      `/work-orders/${workOrderId}/photos/${encodedPhoto}/annotations`,
      { elements },
    );
    return response.data;
  },
};
