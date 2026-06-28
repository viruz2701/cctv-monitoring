// ═══════════════════════════════════════════════════════════════════════
// Work Orders, Maintenance, Spare Parts, Dashboard, Reports, Predictions
// ═══════════════════════════════════════════════════════════════════════

import { useQuery, useMutation, useQueryClient, type QueryClient } from '@tanstack/react-query';
import { api } from '../../services/api';
import { workOrdersApi } from '../../services/workOrdersApi';
import { maintenanceApi } from '../../services/maintenanceApi';
import { sparePartsApi } from '../../services/sparePartsApi';
import { queryKeys, CACHE } from './shared';
import type { WorkOrder } from './shared';
import type { CreateScheduleRequest } from './shared';
import type { CreateSparePartRequest } from './shared';

// ═══════════════════════════════════════════════════════════════════════
// Work Orders (List Data)
// ═══════════════════════════════════════════════════════════════════════

export function useWorkOrders(filters?: Record<string, string>) {
  return useQuery({
    queryKey: [...queryKeys.workOrders.all, filters],
    queryFn: () => workOrdersApi.getWorkOrders(filters),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
    refetchInterval: 120_000,
  });
}

export function useCreateWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: import('../../services/workOrdersApi').CreateWorkOrderRequest) =>
      workOrdersApi.createWorkOrder(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    },
  });
}

export function useUpdateWorkOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<WorkOrder> }) =>
      workOrdersApi.updateWorkOrder(id, data),

    // P0-UX.4: Optimistic update с rollback
    onMutate: async ({ id, data }) => {
      await queryClient.cancelQueries({ queryKey: queryKeys.workOrders.all });

      const previousWorkOrders = queryClient.getQueryData<WorkOrder[]>(queryKeys.workOrders.all);

      queryClient.setQueryData<WorkOrder[]>(queryKeys.workOrders.all, (old) => {
        if (!old) return old;
        return old.map((wo) =>
          wo.id === id ? { ...wo, ...data } : wo,
        );
      });

      return { previousWorkOrders };
    },

    onError: (_err, _vars, context) => {
      if (context?.previousWorkOrders) {
        queryClient.setQueryData(queryKeys.workOrders.all, context.previousWorkOrders);
      }
    },

    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    },
  });
}

export function useDeleteWorkOrder() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => workOrdersApi.deleteWorkOrder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Dashboard (List Data)
// ═══════════════════════════════════════════════════════════════════════

export function useDashboardStats() {
  return useQuery({
    queryKey: queryKeys.dashboard.stats,
    queryFn: () => api.getDashboardStats(),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
    refetchInterval: 60_000,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Reports (Reference Data)
// ═══════════════════════════════════════════════════════════════════════

export function useReports() {
  return useQuery({
    queryKey: queryKeys.reports.all,
    queryFn: () => api.getReports(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Maintenance Schedules (List Data)
// ═══════════════════════════════════════════════════════════════════════

export function useMaintenanceSchedules(filters?: Record<string, string>) {
  return useQuery({
    queryKey: [...queryKeys.maintenance.all, filters],
    queryFn: () => maintenanceApi.getSchedules(filters),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
  });
}

export function useCreateMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateScheduleRequest) =>
      maintenanceApi.createSchedule(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

export function useUpdateMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateScheduleRequest> }) =>
      maintenanceApi.updateSchedule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

export function useDeleteMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => maintenanceApi.deleteSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

export function useCompleteMaintenanceSchedule() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => maintenanceApi.completeSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.maintenance.all });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Parts (List Data)
// ═══════════════════════════════════════════════════════════════════════

export function useSpareParts(filters?: Record<string, string>) {
  return useQuery({
    queryKey: [...queryKeys.spareParts.all, filters],
    queryFn: () => sparePartsApi.getSpareParts(filters),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
  });
}

export function useLowStockParts() {
  return useQuery({
    queryKey: queryKeys.spareParts.lowStock,
    queryFn: () => sparePartsApi.getLowStockParts(),
    staleTime: CACHE.LIST_STALE,
    gcTime: CACHE.LIST_GC,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Spare Part Categories (Reference Data)
// ═══════════════════════════════════════════════════════════════════════

export function useSparePartCategories() {
  return useQuery({
    queryKey: queryKeys.spareParts.categories,
    queryFn: () => sparePartsApi.getCategories(),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
  });
}

export function useCreateSparePart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSparePartRequest) =>
      sparePartsApi.createSparePart(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useUpdateSparePart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<CreateSparePartRequest> }) =>
      sparePartsApi.updateSparePart(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useDeleteSparePart() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => sparePartsApi.deleteSparePart(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useAdjustStock() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, quantity }: { id: string; quantity: number }) =>
      sparePartsApi.adjustStock(id, quantity),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.lowStock });
    },
  });
}

export function useCreateSparePartCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: { name: string; description?: string; color?: string }) =>
      sparePartsApi.createCategory(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.categories });
    },
  });
}

export function useUpdateSparePartCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: { name?: string; description?: string; color?: string } }) =>
      sparePartsApi.updateCategory(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.categories });
    },
  });
}

export function useDeleteSparePartCategory() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => sparePartsApi.deleteCategory(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.spareParts.categories });
    },
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Predictions / Predictive Maintenance (KF-15.1.3)
// ═══════════════════════════════════════════════════════════════════════

import type { Prediction } from './shared';

export function usePredictions(deviceId?: string, limit?: number) {
  return useQuery({
    queryKey: [...queryKeys.predictions.all, deviceId, limit],
    queryFn: () => api.getPredictions(deviceId, limit),
    staleTime: CACHE.REF_STALE,
    gcTime: CACHE.REF_GC,
    refetchInterval: 300_000,
  });
}

// ═══════════════════════════════════════════════════════════════════════
// Prefetch utilities
// ═══════════════════════════════════════════════════════════════════════

/**
 * Prefetch work order detail on row hover.
 * Использование: onRowHover={(wo) => prefetchWorkOrder(queryClient, wo.id)}
 */
export function prefetchWorkOrder(client: QueryClient, id: string) {
  if (!id) return;
  client.prefetchQuery({
    queryKey: queryKeys.workOrders.detail(id),
    queryFn: () => workOrdersApi.getWorkOrder(id),
    staleTime: 30_000,
  });
}
