import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { workOrdersApi } from '../api/workOrders';
import { CompleteWorkOrderPayload } from '../types';
import { useWorkOrderStore } from '../store/workOrderStore';

export function useWorkOrders() {
  const setCached = useWorkOrderStore((s) => s.setCachedWorkOrders);

  return useQuery({
    queryKey: ['myWorkOrders'],
    queryFn: async () => {
      const orders = await workOrdersApi.getMyWorkOrders();
      setCached(orders);
      return orders;
    },
  });
}

export function useWorkOrder(id: string) {
  return useQuery({
    queryKey: ['workOrder', id],
    queryFn: () => workOrdersApi.getWorkOrder(id),
    enabled: !!id,
  });
}

export function useStartWorkOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => workOrdersApi.startWorkOrder(id),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: ['myWorkOrders'] });
      queryClient.invalidateQueries({ queryKey: ['workOrder', id] });
    },
  });
}

export function useCompleteWorkOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, payload }: { id: string; payload: CompleteWorkOrderPayload }) =>
      workOrdersApi.completeWorkOrder(id, payload),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: ['myWorkOrders'] });
      queryClient.invalidateQueries({ queryKey: ['workOrder', id] });
    },
  });
}

export function useUploadPhoto() {
  return useMutation({
    mutationFn: ({ workOrderId, photoUri }: { workOrderId: string; photoUri: string }) =>
      workOrdersApi.uploadPhoto(workOrderId, photoUri),
  });
}

export function useTechnicianProfile() {
  return useQuery({
    queryKey: ['technicianProfile'],
    queryFn: () => workOrdersApi.getTechnicianProfile(),
  });
}

export function useTechnicianStats() {
  return useQuery({
    queryKey: ['technicianStats'],
    queryFn: () => workOrdersApi.getTechnicianStats(),
  });
}