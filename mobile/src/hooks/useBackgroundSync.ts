import { useEffect, useRef, useState, useCallback } from 'react';
import { AppState, AppStateStatus } from 'react-native';
import * as BackgroundFetch from 'expo-background-fetch';
import * as TaskManager from 'expo-task-manager';
import { syncService, SyncStatus } from '../services/syncService';

// ──────────────────────────────────────────────────
// Constants
// ──────────────────────────────────────────────────

const BACKGROUND_SYNC_TASK = 'cctv-background-sync';
const SYNC_INTERVAL_MINUTES = 15;

// ──────────────────────────────────────────────────
// Background Task Definition
// ──────────────────────────────────────────────────

TaskManager.defineTask(BACKGROUND_SYNC_TASK, async () => {
  try {
    const { status } = syncService;

    // Если offline — ничего не делаем
    if (status === 'offline') {
      return BackgroundFetch.BackgroundFetchResult.NoData;
    }

    // Запускаем синхронизацию
    await syncService.syncWhenOnline();

    return BackgroundFetch.BackgroundFetchResult.NewData;
  } catch (error) {
    console.error('[BackgroundSync] Task failed:', error);
    return BackgroundFetch.BackgroundFetchResult.Failed;
  }
});

// ──────────────────────────────────────────────────
// State (what the hook exposes)
// ──────────────────────────────────────────────────

export interface BackgroundSyncState {
  /** Текущий статус соединения/синхронизации */
  status: SyncStatus;
  /** Количество ожидающих мутаций в очереди */
  pendingCount: number;
  /** Timestamp последней успешной синхронизации */
  lastSyncAt: number | null;
  /** Текст последней ошибки (если была) */
  lastError: string | null;
  /** Зарегистрирована ли background fetch задача */
  isRegistered: boolean;
}

export interface BackgroundSyncActions {
  /** Запустить синхронизацию вручную */
  manualSync: () => Promise<void>;
  /** Обновить pending count из БД */
  refreshPendingCount: () => Promise<void>;
}

// ──────────────────────────────────────────────────
// Hook
// ──────────────────────────────────────────────────

export function useBackgroundSync(): BackgroundSyncState & BackgroundSyncActions {
  const appStateRef = useRef<AppStateStatus>(AppState.currentState);

  // Реактивное состояние — подписка на syncService
  const [status, setStatus] = useState<SyncStatus>(syncService.status);
  const [pendingCount, setPendingCount] = useState(syncService.pendingCount);
  const [lastSyncAt, setLastSyncAt] = useState<number | null>(syncService.lastSyncAt);
  const [lastError, setLastError] = useState<string | null>(syncService.lastError);
  const [isRegistered, setIsRegistered] = useState(false);

  // ── Manual sync ──────────────────────────────

  const manualSync = useCallback(async () => {
    await syncService.syncWhenOnline();
  }, []);

  // ── Refresh pending count ────────────────────

  const refreshPendingCount = useCallback(async () => {
    const count = await syncService.getPendingCount();
    setPendingCount(count);
  }, []);

  // ── Init effect ──────────────────────────────

  useEffect(() => {
    let isMounted = true;

    const init = async () => {
      try {
        // Инициализируем SyncService (безопасно — guard на двойной вызов)
        await syncService.initialize();

        // Проверяем статус background fetch
        const bfStatus = await BackgroundFetch.getStatusAsync();

        if (bfStatus === BackgroundFetch.BackgroundFetchStatus.Denied) {
          console.warn('[BackgroundSync] Background fetch denied');
          if (isMounted) setIsRegistered(false);
          return;
        }

        // Регистрируем задачу, если ещё не зарегистрирована
        const registered = await TaskManager.isTaskRegisteredAsync(
          BACKGROUND_SYNC_TASK,
        );

        if (!registered) {
          await BackgroundFetch.registerTaskAsync(BACKGROUND_SYNC_TASK, {
            minimumInterval: SYNC_INTERVAL_MINUTES * 60, // 15 минут
            stopOnTerminate: false,
            startOnBoot: true,
          });
        }

        if (isMounted) setIsRegistered(true);
      } catch (error) {
        console.error('[BackgroundSync] Init failed:', error);
        if (isMounted) setIsRegistered(false);
      }
    };

    init();

    return () => {
      isMounted = false;
    };
  }, []);

  // ── Sync service subscription ────────────────

  useEffect(() => {
    const unsubscribe = syncService.subscribe((state) => {
      setStatus(state.status);
      setPendingCount(state.pendingCount);
      setLastSyncAt(state.lastSyncAt);
      setLastError(state.lastError);
    });

    return () => {
      unsubscribe();
    };
  }, []);

  // ── App state listener ───────────────────────

  useEffect(() => {
    const subscription = AppState.addEventListener(
      'change',
      (nextAppState: AppStateStatus) => {
        // Возвращаемся в foreground — проверяем очередь
        if (
          appStateRef.current.match(/inactive|background/) &&
          nextAppState === 'active'
        ) {
          syncService.syncWhenOnline();
        }

        appStateRef.current = nextAppState;
      },
    );

    return () => {
      subscription.remove();
    };
  }, []);

  // ── Периодическое обновление pending count ───

  useEffect(() => {
    const interval = setInterval(async () => {
      const count = await syncService.getPendingCount();
      setPendingCount(count);
    }, 10_000); // каждые 10 секунд

    return () => {
      clearInterval(interval);
    };
  }, []);

  return {
    status,
    pendingCount,
    lastSyncAt,
    lastError,
    isRegistered,
    manualSync,
    refreshPendingCount,
  };
}
