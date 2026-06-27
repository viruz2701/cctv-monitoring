import { useEffect, useRef, useState, useCallback } from 'react';
import { AppState, AppStateStatus } from 'react-native';
import { syncService, SyncStatus } from '../services/syncService';

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
  manualSync: () => Promise<{ success: boolean; error?: string }>;
  /** Обновить pending count из БД */
  refreshPendingCount: () => Promise<void>;
  /** Включить/выключить background sync */
  toggleBackgroundSync: () => Promise<void>;
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

  const manualSync = useCallback(async (): Promise<{ success: boolean; error?: string }> => {
    return syncService.syncNow();
  }, []);

  // ── Refresh pending count ────────────────────

  const refreshPendingCount = useCallback(async () => {
    const count = await syncService.getPendingCount();
    setPendingCount(count);
  }, []);

  // ── Toggle background sync ──────────────────

  const toggleBackgroundSync = useCallback(async () => {
    if (isRegistered) {
      await syncService.stopBackgroundSync();
      setIsRegistered(false);
    } else {
      const registered = await syncService.startBackgroundSync();
      setIsRegistered(registered);
    }
  }, [isRegistered]);

  // ── Init effect ──────────────────────────────

  useEffect(() => {
    let isMounted = true;

    const init = async () => {
      try {
        // Инициализируем SyncService (безопасно — guard на двойной вызов)
        await syncService.initialize();

        // Регистрируем background fetch через syncService
        const registered = await syncService.startBackgroundSync();
        if (isMounted) setIsRegistered(registered);
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
      setIsRegistered(state.isBackgroundRegistered);
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
    toggleBackgroundSync,
  };
}
