import { useMutation } from '@tanstack/react-query';
import { gatekeeperApi } from '../api/gatekeeper';
import { VerificationRequest } from '../types';

export function useVerifyWorkOrder() {
  return useMutation({
    mutationFn: ({
      workOrderId,
      payload,
    }: {
      workOrderId: string;
      payload: VerificationRequest;
    }) => gatekeeperApi.verify(workOrderId, payload),
  });
}