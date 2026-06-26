import { create } from 'zustand';
import { storage } from '../utils/storage';
import { workOrdersApi } from '../api/workOrders';
import { CompleteWorkOrderPayload } from '../types';
import { ConflictData, ConflictResolutionAction } from '../components/ConflictResolutionModal';

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
  conflicts: ConflictData[];

  addToQueue: (action: Omit<SyncAction, 'id' | 'timestamp' | 'retryCount'>) => Promise<void>;
  processQueue: () => Promise<void>;
  setOnline: (online: boolean) => void;
  loadQueue: () => Promise<void>;
  addConflict: (conflict: ConflictData) => void;
  resolveConflict: (action: ConflictResolutionAction) => void;
  getConflicts: () => ConflictData[];
}

export const useSyncStore = create<SyncState>((set, get) => ({
  queue: [],
  isOnline: true,
  conflicts: [],

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

  // ── Conflict Management ────────────────────────

  addConflict: (conflict: ConflictData) => {
    set((state) => {
      // Не добавляем дубликат, если conflict с таким id уже есть
      const exists = state.conflicts.some((c) => c.id === conflict.id);
      if (exists) {
        // Обновляем существующий
        const updated = state.conflicts.map((c) =>
          c.id === conflict.id ? conflict : c,
        );
        return { conflicts: updated };
      }
      return { conflicts: [...state.conflicts, conflict] };
    });
  },

  resolveConflict: (action: ConflictResolutionAction) => {
    set((state) => {
      const remaining = state.conflicts.filter(
        (c) => c.id !== action.conflictId,
      );
      return { conflicts: remaining };
    });
  },

  getConflicts: () => {
    return get().conflicts;
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