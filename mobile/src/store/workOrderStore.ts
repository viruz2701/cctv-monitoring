import { create } from 'zustand';
import { WorkOrder } from '../types';

interface WorkOrderState {
  cachedWorkOrders: Map<string, WorkOrder>;
  currentWorkOrder: WorkOrder | null;

  setCachedWorkOrders: (orders: WorkOrder[]) => void;
  updateCachedWorkOrder: (order: WorkOrder) => void;
  setCurrentWorkOrder: (order: WorkOrder | null) => void;
  getCachedWorkOrder: (id: string) => WorkOrder | undefined;
}

export const useWorkOrderStore = create<WorkOrderState>((set, get) => ({
  cachedWorkOrders: new Map(),
  currentWorkOrder: null,

  setCachedWorkOrders: (orders) => {
    const map = new Map<string, WorkOrder>();
    orders.forEach((order) => map.set(order.id, order));
    set({ cachedWorkOrders: map });
  },

  updateCachedWorkOrder: (order) => {
    const map = new Map(get().cachedWorkOrders);
    map.set(order.id, order);
    set({ cachedWorkOrders: map });
  },

  setCurrentWorkOrder: (order) => set({ currentWorkOrder: order }),

  getCachedWorkOrder: (id) => get().cachedWorkOrders.get(id),
}));