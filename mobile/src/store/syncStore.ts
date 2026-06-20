import { create } from 'zustand';
import { storage } from '../utils/storage';
import { workOrdersApi } from '../api/workOrders';
import { CompleteWorkOrderPayload } from '../types';

interface SyncAction {
  id: string;
  type: 'complete_work_order' | 'start_work_order';
  workOrderId: string;
  payload?: CompleteWorkOrderPayload;
  timestamp: number;
  retryCount: number;
}

interface SyncState {
  queue: SyncAction[];
  isOnline: boolean;

  addToQueue: (action: Omit<SyncAction, 'id' | 'timestamp' | 'retryCount'>) => Promise<void>;
  processQueue: () => Promise<void>;
  setOnline: (online: boolean) => void;
  loadQueue: () => Promise<void>;
}

export const useSyncStore = create<SyncState>((set, get) => ({
  queue: [],
  isOnline: true,

  addToQueue: async (action: Omit<SyncAction, 'id' | 'timestamp' | 'retryCount'>) => {
    const newAction: SyncAction = {
      ...action,
      id: `sync_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: Date.now(),
      retryCount: 0,
    };

    set((state: SyncState) => {
      const updated = [...state.queue, newAction];
      storage.setSyncQueue(JSON.stringify(updated));
      return { queue: updated };
    });
  },

  processQueue: async () => {
    const { queue, isOnline } = get();
    if (!isOnline || queue.length === 0) return;

    const updatedQueue = [...queue];
    let hasChanges = false;

    for (let i = 0; i < updatedQueue.length; i++) {
      const action = updatedQueue[i];

      try {
        if (action.type === 'complete_work_order' && action.payload) {
          await workOrdersApi.completeWorkOrder(action.workOrderId, action.payload);
        } else if (action.type === 'start_work_order') {
          await workOrdersApi.startWorkOrder(action.workOrderId);
        }

        updatedQueue.splice(i, 1);
        i--;
        hasChanges = true;
      } catch (error: unknown) {
        console.error(`Sync failed for action ${action.id}:`, error);
        action.retryCount++;

        if (action.retryCount >= 3) {
          updatedQueue.splice(i, 1);
          i--;
          hasChanges = true;
        }
      }
    }

    if (hasChanges) {
      set({ queue: updatedQueue });
      storage.setSyncQueue(JSON.stringify(updatedQueue));
    }
  },

  setOnline: (online: boolean) => {
    set({ isOnline: online });
    if (online) {
      get().processQueue();
    }
  },

  loadQueue: async () => {
    try {
      const stored = await storage.getSyncQueue();
      if (stored) {
        set({ queue: JSON.parse(stored) });
      }
    } catch (error: unknown) {
      console.error('Failed to load sync queue:', error);
    }
  },
}));