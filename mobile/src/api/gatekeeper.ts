import { apiClient } from './client';
import { VerificationRequest, VerificationResponse } from '../types';

export const gatekeeperApi = {
  verify: async (
    workOrderId: string,
    payload: VerificationRequest,
  ): Promise<VerificationResponse> => {
    const response = await apiClient.post<VerificationResponse>(
      `/mobile/work-orders/${workOrderId}/verify`,
      payload,
    );
    return response.data;
  },
};