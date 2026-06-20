import { apiClient } from './client';
import { WorkOrder, CompleteWorkOrderPayload } from '../types';

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
};