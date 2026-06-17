import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { workOrdersApi, WorkOrder, CreateWorkOrderRequest, PartUsage } from '../services/workOrdersApi';
import { useAuth } from '../hooks/useAuth';

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
}

const WorkOrdersContext = createContext<WorkOrdersContextType | undefined>(undefined);

export const WorkOrdersProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token } = useAuth();
  const [workOrders, setWorkOrders] = useState<WorkOrder[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchWorkOrders = useCallback(async (filters?: Record<string, string>) => {
    setLoading(true);
    setError(null);
    try {
      const data = await workOrdersApi.getWorkOrders(filters);
      setWorkOrders(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch work orders');
    } finally {
      setLoading(false);
    }
  }, []);

  const createWorkOrder = async (data: CreateWorkOrderRequest): Promise<WorkOrder> => {
    const wo = await workOrdersApi.createWorkOrder(data);
    setWorkOrders((prev) => [...prev, wo]);
    return wo;
  };

  const updateWorkOrder = async (id: string, data: Partial<WorkOrder>) => {
    await workOrdersApi.updateWorkOrder(id, data);
    setWorkOrders((prev) => prev.map((wo) => (wo.id === id ? { ...wo, ...data } : wo)));
  };

  const deleteWorkOrder = async (id: string) => {
    await workOrdersApi.deleteWorkOrder(id);
    setWorkOrders((prev) => prev.filter((wo) => wo.id !== id));
  };

  const assignWorkOrder = async (id: string, userId: string) => {
    await workOrdersApi.assignWorkOrder(id, userId);
    setWorkOrders((prev) => prev.map((wo) => (wo.id === id ? { ...wo, assigned_to: userId } : wo)));
  };

  const startWorkOrder = async (id: string) => {
    await workOrdersApi.startWorkOrder(id);
    setWorkOrders((prev) =>
      prev.map((wo) => (wo.id === id ? { ...wo, status: 'in_progress', started_at: new Date().toISOString() } : wo))
    );
  };

  const completeWorkOrder = async (id: string, notes: string, photos: string[], parts: PartUsage[]) => {
    await workOrdersApi.completeWorkOrder(id, notes, photos, parts);
    setWorkOrders((prev) =>
      prev.map((wo) =>
        wo.id === id
          ? { ...wo, status: 'completed', completed_at: new Date().toISOString(), notes, photos, parts_used: parts }
          : wo
      )
    );
  };

  const cancelWorkOrder = async (id: string, reason: string) => {
    await workOrdersApi.cancelWorkOrder(id, reason);
    setWorkOrders((prev) =>
      prev.map((wo) => (wo.id === id ? { ...wo, status: 'cancelled', notes: reason } : wo))
    );
  };

  useEffect(() => {
    if (!token) return;
    fetchWorkOrders();
  }, [fetchWorkOrders, token]);

  return (
    <WorkOrdersContext.Provider
      value={{
        workOrders, loading, error, fetchWorkOrders, createWorkOrder, updateWorkOrder,
        deleteWorkOrder, assignWorkOrder, startWorkOrder, completeWorkOrder, cancelWorkOrder,
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
