import { create } from 'zustand';
import { storage } from '../utils/storage';
import { workOrdersApi } from '../api/workOrders';
import { CompleteWorkOrderPayload } from '../types';
import { ConflictData, ConflictResolutionAction } from '../components/ConflictResolutionModal';
import { captureError } from '../lib/sentry';
import { syncService } from '../services/syncService';

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

// ── Telemetry ─────────────────────────────────────

function logTelemetry(
  event: string,
  payload: Record<string, unknown>,
): void {
  console.log(
    JSON.stringify({
      event: `conflict_resolution.${event}`,
      timestamp: Date.now(),
      payload,
    }),
  );
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
    const { conflicts, queue } = get();
    const conflict = conflicts.find((c) => c.id === action.conflictId);
    if (!conflict) return;

    switch (action.type) {
      case 'keep_local': {
        // Применяем локальные изменения — ставим action обратно в очередь
        const hasPending = queue.some(
          (q) => q.workOrderId === action.conflictId && q.type === 'complete_work_order',
        );
        if (!hasPending) {
          get().addToQueue({
            type: 'complete_work_order',
            workOrderId: action.conflictId,
            payload: { notes: '', checklist: [], photos: [], parts_used: [] },
          });
        }
        logTelemetry('resolved_keep_local', {
          conflictId: action.conflictId,
          label: conflict.label,
        });
        break;
      }

      case 'keep_server': {
        // Отбрасываем локальные изменения, пулим серверную версию
        syncService.pullLatestData().catch((err) => {
          captureError(err, {
            context: 'conflict_resolution_keep_server',
            conflictId: action.conflictId,
          });
        });
        logTelemetry('resolved_keep_server', {
          conflictId: action.conflictId,
          label: conflict.label,
        });
        break;
      }

      case 'merge': {
        // Применяем объединённые поля — добавляем мутацию в syncService
        syncService.enqueueMutation({
          entityType: 'work_order',
          entityId: action.conflictId,
          mutationType: 'update',
          payload: action.mergedFields,
        }).catch((err) => {
          captureError(err, {
            context: 'conflict_resolution_merge_enqueue',
            conflictId: action.conflictId,
          });
        });
        logTelemetry('resolved_merge', {
          conflictId: action.conflictId,
          label: conflict.label,
          mergedFields: Object.keys(action.mergedFields),
        });
        break;
      }
    }

    // Удаляем конфликт из списка
    set((state) => ({
      conflicts: state.conflicts.filter((c) => c.id !== action.conflictId),
    }));
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