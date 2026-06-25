import React, { createContext, useContext, useState, useCallback } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  useWorkOrders as useWorkOrdersQuery,
  useCreateWorkOrder,
  useUpdateWorkOrder,
  useDeleteWorkOrder,
  queryKeys,
} from '../hooks/useApiQuery';
import { workOrdersApi, WorkOrder, CreateWorkOrderRequest, PartUsage } from '../services/workOrdersApi';

// ── Bulk Action Types (WO-4.2.1) ────────────────────────────────────

export type BulkActionType = 'status_change' | 'assign' | 'delete' | 'priority_change';

export interface BulkActionResult {
  id: string;
  status: 'success' | 'error';
  error?: string;
}

export interface BulkActionResponse {
  results: BulkActionResult[];
  total: number;
  success: number;
  failed: number;
}

interface WorkOrdersContextType {
  workOrders: WorkOrder[];
  loading: boolean;
  error: string | null;
  fetchWorkOrders: (filters?: Record<string, string>) => Promise<void>;
  createWorkOrder: (data: CreateWorkOrderRequest) => Promise<WorkOrder>;
  updateWorkOrder: (id: string, data: Partial<WorkOrder>) => Promise<void>;
  deleteWorkOrder: (id: string) => Promise<void>;
  assignWorkOrder: (id: string, userId: string) => Promise<void>;
  startWorkOrder: (id: string) => Promise<void>;
  completeWorkOrder: (id: string, notes: string, photos: string[], parts: PartUsage[]) => Promise<void>;
  cancelWorkOrder: (id: string, reason: string) => Promise<void>;
  bulkActionWorkOrders: (action: BulkActionType, ids: string[], value?: string) => Promise<BulkActionResponse>;
}

const WorkOrdersContext = createContext<WorkOrdersContextType | undefined>(undefined);

export const WorkOrdersProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token } = useAuth();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<Record<string, string> | undefined>(undefined);

  // ── React Query: work orders list ─────────────────────────────────
  // ARCH-02: Server state управляется React Query — кэш, рефетч, stale-while-revalidate
  // Compliance: IEC 62443 SR 7.1 (Resource availability — async data fetching)
  const {
    data: workOrders = [],
    isFetching,
    error: queryError,
    refetch,
  } = useQuery({
    queryKey: [...queryKeys.workOrders.all, filters],
    queryFn: () => workOrdersApi.getWorkOrders(filters),
    enabled: !!token,
    staleTime: 30_000,
    refetchInterval: 120_000,
  });

  // ── React Query Mutations ─────────────────────────────────────────
  const createMutation = useCreateWorkOrder();
  const updateMutation = useUpdateWorkOrder();
  const deleteMutation = useDeleteWorkOrder();

  // ── Context Methods ───────────────────────────────────────────────

  const fetchWorkOrders = useCallback(async (newFilters?: Record<string, string>) => {
    if (!token) return;
    setFilters(newFilters);
    await refetch();
  }, [refetch, token]);

  const createWorkOrderFn = async (data: CreateWorkOrderRequest): Promise<WorkOrder> => {
    return createMutation.mutateAsync(data);
  };

  const updateWorkOrderFn = async (id: string, data: Partial<WorkOrder>) => {
    await updateMutation.mutateAsync({ id, data });
  };

  const deleteWorkOrderFn = async (id: string) => {
    await deleteMutation.mutateAsync(id);
  };

  // ── Lifecycle Methods (custom API calls + cache invalidation) ────
  // Эти методы вызывают специализированные эндпоинты, затем инвалидируют кэш

  const assignWorkOrder = async (id: string, userId: string) => {
    await workOrdersApi.assignWorkOrder(id, userId);
    queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
  };

  const startWorkOrder = async (id: string) => {
    await workOrdersApi.startWorkOrder(id);
    queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
  };

  const completeWorkOrder = async (id: string, notes: string, photos: string[], parts: PartUsage[]) => {
    await workOrdersApi.completeWorkOrder(id, notes, photos, parts);
    queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
  };

  const cancelWorkOrder = async (id: string, reason: string) => {
    await workOrdersApi.cancelWorkOrder(id, reason);
    queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
  };

  // ── Bulk Actions (WO-4.2.1) ──────────────────────────────────────

  const bulkActionWorkOrders = async (action: BulkActionType, ids: string[], value?: string) => {
    const response = await workOrdersApi.bulkActions(action, ids, value);
    queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
    return response;
  };

  // ── Error normalisation ───────────────────────────────────────────
  const error = queryError instanceof Error ? queryError.message : queryError ? String(queryError) : null;

  return (
    <WorkOrdersContext.Provider
      value={{
        workOrders, loading: isFetching, error, fetchWorkOrders,
        createWorkOrder: createWorkOrderFn, updateWorkOrder: updateWorkOrderFn,
        deleteWorkOrder: deleteWorkOrderFn, assignWorkOrder, startWorkOrder,
        completeWorkOrder, cancelWorkOrder, bulkActionWorkOrders,
      }}
    >
      {children}
    </WorkOrdersContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useWorkOrders = () => {
  const context = useContext(WorkOrdersContext);
  if (!context) {
    throw new Error('useWorkOrders must be used within WorkOrdersProvider');
  }
  return context;
};
