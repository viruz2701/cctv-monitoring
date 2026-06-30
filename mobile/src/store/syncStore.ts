// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: Differential Sync Store (Zustand)
//
// Управляет состоянием дифференциальной синхронизации:
//   - Статус синхронизации (idle | syncing | error)
//   - Прогресс по каждой сущности
//   - Last sync time
//   - Количество ожидающих изменений
//
// Сохраняет обратную совместимость с существующими компонентами
// (addToQueue, processQueue, setOnline, conflicts).
//
// Соответствует:
//   - IEC 62443-3-3 SR 3.1 (Queue-based processing)
// ═══════════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { storage } from '../utils/storage';
import { workOrdersApi } from '../api/workOrders';
import { CompleteWorkOrderPayload } from '../types';
import { captureError } from '../lib/sentry';
import { syncService } from '../services/syncService';
import {
  DifferentialSyncService,
  SyncResult,
} from '../services/differentialSync';
import { syncApi, SyncStatusResponse } from '../api/sync';

// ── Legacy Types (from existing store) ──────────────────────────────────

interface ConflictData {
  id: string;
  label: string;
  local: Record<string, unknown>;
  server: Record<string, unknown>;
  field: string;
}

interface ConflictResolutionAction {
  conflictId: string;
  type: 'keep_local' | 'keep_server' | 'merge';
  mergedFields: Record<string, unknown>;
}

interface SyncAction {
  id: string;
  type: 'complete_work_order' | 'start_work_order' | 'checklist_update' | 'checklist_complete';
  workOrderId: string;
  payload?: CompleteWorkOrderPayload | Record<string, unknown>;
  timestamp: number;
  retryCount: number;
}

interface ChecklistUpdatePayload {
  itemOrder: number;
  status: 'passed' | 'failed' | 'skipped';
  timestamp: number;
}

interface ChecklistCompletePayload {
  workOrderId: string;
  regulationId: string;
  regionCode: string;
  items: unknown[];
  completedAt: string;
  passedCount: number;
  failedCount: number;
  skippedCount: number;
  totalCount: number;
  synced: boolean;
}

// ── P1-SYNC Types ───────────────────────────────────────────────────────

/** Статус синхронизации */
export type SyncStatus = 'idle' | 'syncing' | 'success' | 'error';

/** Прогресс по одной сущности */
export interface EntityProgress {
  entity: string;
  status: 'pending' | 'syncing' | 'done' | 'error';
  changesApplied: number;
  changesPushed: number;
  error: string | null;
}

/** Состояние sync store */
export interface SyncState {
  // ── P1-SYNC: Differential Sync ──────────────────────────────────────
  /** Текущий статус */
  dSyncStatus: SyncStatus;
  /** Прогресс по каждой сущности */
  dSyncProgress: Record<string, EntityProgress>;
  /** Время последней успешной синхронизации (ISO8601) */
  dSyncLastSyncTime: string | null;
  /** Общее количество изменений за последний sync */
  dSyncLastChangesCount: number;
  /** Длительность последнего sync в ms */
  dSyncLastDurationMs: number;
  /** Текст ошибки */
  dSyncError: string | null;
  /** Количество ожидающих локальных изменений */
  dSyncPendingCount: number;
  /** Статус с сервера (bandwidth, total syncs) */
  dSyncServerStatus: SyncStatusResponse | null;

  // ── Legacy: Queue & Conflicts ───────────────────────────────────────
  queue: SyncAction[];
  isOnline: boolean;
  conflicts: ConflictData[];

  // ── P1-SYNC Actions ─────────────────────────────────────────────────
  /** Запустить полный sync cycle */
  startSync: () => Promise<SyncResult>;
  /** Обновить статус синхронизации */
  setDSyncStatus: (status: SyncStatus) => void;
  /** Обновить прогресс по entity */
  updateEntityProgress: (entity: string, progress: Partial<EntityProgress>) => void;
  /** Запросить статус с сервера */
  fetchServerStatus: () => Promise<void>;
  /** Обновить счётчик ожидающих изменений */
  refreshPendingCount: () => Promise<void>;
  /** Сбросить состояние */
  resetDSync: () => void;

  // ── Legacy Actions ──────────────────────────────────────────────────
  addToQueue: (action: Omit<SyncAction, 'id' | 'timestamp' | 'retryCount'>) => Promise<void>;
  processQueue: () => Promise<void>;
  setOnline: (online: boolean) => void;
  loadQueue: () => Promise<void>;
  addConflict: (conflict: ConflictData) => void;
  resolveConflict: (action: ConflictResolutionAction) => void;
  getConflicts: () => ConflictData[];
}

// ── Telemetry ───────────────────────────────────────────────────────────

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

// ── Default State ───────────────────────────────────────────────────────

const defaultDSyncState = {
  dSyncStatus: 'idle' as SyncStatus,
  dSyncProgress: {} as Record<string, EntityProgress>,
  dSyncLastSyncTime: null as string | null,
  dSyncLastChangesCount: 0,
  dSyncLastDurationMs: 0,
  dSyncError: null as string | null,
  dSyncPendingCount: 0,
  dSyncServerStatus: null as SyncStatusResponse | null,
};

// ── Store ────────────────────────────────────────────────────────────────

export const useSyncStore = create<SyncState>((set, get) => ({
  // ── Initial State ───────────────────────────────────────────────────
  ...defaultDSyncState,
  queue: [],
  isOnline: true,
  conflicts: [],

  // ══════════════════════════════════════════════════════════════════════
  // P1-SYNC Actions
  // ══════════════════════════════════════════════════════════════════════

  /**
   * Запустить полный differential sync cycle.
   * Обновляет статус и прогресс в реальном времени.
   */
  startSync: async (): Promise<SyncResult> => {
    const service = new DifferentialSyncService();
    set({ dSyncStatus: 'syncing', dSyncError: null });

    // Инициализируем прогресс по entity
    const entities = ['work_orders', 'devices', 'photos', 'audit'];
    const initialProgress: Record<string, EntityProgress> = {};
    for (const entity of entities) {
      initialProgress[entity] = {
        entity,
        status: 'pending',
        changesApplied: 0,
        changesPushed: 0,
        error: null,
      };
    }
    set({ dSyncProgress: initialProgress });

    try {
      const result = await service.sync();

      // Обновляем прогресс из результатов
      const updatedProgress = { ...get().dSyncProgress };
      for (const entityResult of result.entities) {
        updatedProgress[entityResult.entity] = {
          entity: entityResult.entity,
          status: entityResult.errors > 0 ? 'error' : 'done',
          changesApplied: entityResult.changesApplied,
          changesPushed: entityResult.changesPushed,
          error: entityResult.errors > 0 ? 'Sync error' : null,
        };
      }

      const lastSyncTime = new Date().toISOString();

      set({
        dSyncStatus: result.success ? 'success' : 'error',
        dSyncProgress: updatedProgress,
        dSyncLastSyncTime: lastSyncTime,
        dSyncLastChangesCount: result.pullChanges + result.pushChanges,
        dSyncLastDurationMs: result.totalDurationMs,
        dSyncError: result.error,
      });

      // Автоматически возвращаемся в idle через 3 секунды
      if (result.success) {
        setTimeout(() => {
          const current = get();
          if (current.dSyncStatus === 'success') {
            set({ dSyncStatus: 'idle' });
          }
        }, 3_000);
      }

      return result;
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Unknown sync error';

      set({
        dSyncStatus: 'error',
        dSyncError: message,
      });

      return {
        success: false,
        pullChanges: 0,
        pushChanges: 0,
        conflictsResolved: 0,
        entities: [],
        totalDurationMs: 0,
        error: message,
      };
    }
  },

  /**
   * Установить статус синхронизации вручную.
   */
  setDSyncStatus: (status: SyncStatus) => {
    set({ dSyncStatus: status });
  },

  /**
   * Обновить прогресс по конкретной сущности.
   */
  updateEntityProgress: (
    entity: string,
    progress: Partial<EntityProgress>,
  ) => {
    const current = get().dSyncProgress[entity];
    if (current) {
      set({
        dSyncProgress: {
          ...get().dSyncProgress,
          [entity]: { ...current, ...progress },
        },
      });
    }
  },

  /**
   * Запросить статус синхронизации с сервера.
   */
  fetchServerStatus: async () => {
    try {
      const status = await syncApi.getStatus();
      set({ dSyncServerStatus: status });
    } catch (error) {
      console.warn('[SyncStore] Failed to fetch server status:', error);
    }
  },

  /**
   * Обновить количество ожидающих локальных изменений.
   */
  refreshPendingCount: async () => {
    try {
      const { getPendingMutationCount } = await import(
        '../services/offlineStorage'
      );
      const count = await getPendingMutationCount();
      set({ dSyncPendingCount: count });
    } catch (error) {
      console.warn('[SyncStore] Failed to refresh pending count:', error);
    }
  },

  /**
   * Сбросить состояние дифференциальной синхронизации.
   */
  resetDSync: () => {
    set({ ...defaultDSyncState });
  },

  // ══════════════════════════════════════════════════════════════════════
  // Legacy Actions (обратная совместимость)
  // ══════════════════════════════════════════════════════════════════════

  /**
   * Добавить действие в очередь синхронизации.
   */
  addToQueue: async (action: Omit<SyncAction, 'id' | 'timestamp' | 'retryCount'>) => {
    const newAction: SyncAction = {
      ...action,
      id: `sync_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
      timestamp: Date.now(),
      retryCount: 0,
    };

    set((state) => {
      const updated = [...state.queue, newAction];
      storage.setSyncQueue(JSON.stringify(updated));
      return { queue: updated };
    });
  },

  /**
   * Обработать очередь синхронизации.
   */
  processQueue: async () => {
    const { queue, isOnline } = get();
    if (!isOnline || queue.length === 0) return;

    const updatedQueue = [...queue];
    let hasChanges = false;

    for (let i = 0; i < updatedQueue.length; i++) {
      const action = updatedQueue[i];

      try {
        if (action.type === 'complete_work_order' && action.payload) {
          const woPayload = action.payload as CompleteWorkOrderPayload;
          await workOrdersApi.completeWorkOrder(action.workOrderId, woPayload);
        } else if (action.type === 'start_work_order') {
          await workOrdersApi.startWorkOrder(action.workOrderId);
        } else if (action.type === 'checklist_update' || action.type === 'checklist_complete') {
          console.log(`Checklist action ${action.type} for WO ${action.workOrderId} recorded locally`);
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

  /**
   * Установить статус онлайн/офлайн.
   */
  setOnline: (online: boolean) => {
    set({ isOnline: online });
    if (online) {
      get().processQueue();
    }
  },

  /**
   * Загрузить очередь из storage.
   */
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

  // ── Conflict Management ─────────────────────────────────────────────

  /**
   * Добавить конфликт.
   */
  addConflict: (conflict: ConflictData) => {
    set((state) => {
      const exists = state.conflicts.some((c) => c.id === conflict.id);
      if (exists) {
        const updated = state.conflicts.map((c) =>
          c.id === conflict.id ? conflict : c,
        );
        return { conflicts: updated };
      }
      return { conflicts: [...state.conflicts, conflict] };
    });
  },

  /**
   * Разрешить конфликт.
   */
  resolveConflict: (action: ConflictResolutionAction) => {
    const { conflicts, queue } = get();
    const conflict = conflicts.find((c) => c.id === action.conflictId);
    if (!conflict) return;

    switch (action.type) {
      case 'keep_local': {
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

    set((state) => ({
      conflicts: state.conflicts.filter((c) => c.id !== action.conflictId),
    }));
  },

  /**
   * Получить список конфликтов.
   */
  getConflicts: () => {
    return get().conflicts;
  },
}));
