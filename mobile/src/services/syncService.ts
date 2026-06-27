import * as BackgroundFetch from 'expo-background-fetch';
import * as TaskManager from 'expo-task-manager';
import NetInfo, { NetInfoState } from '@react-native-community/netinfo';
import { workOrdersApi } from '../api/workOrders';
import { devicesApi } from '../api/devices';
import { CompleteWorkOrderPayload, WorkOrder } from '../types';
import {
  initDatabase,
  upsertWorkOrders,
  getPendingMutations,
  removePendingMutation,
  incrementPendingRetry,
  savePendingMutation,
  clearDevices,
  upsertDevices,
  clearSites,
  upsertSites,
  clearWorkOrders,
  upsertWorkOrder,
  getPendingMutationCount,
  getWorkOrders,
  DeviceRow,
  SiteRow,
} from './offlineStorage';

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
    const instance = syncService;

    // Если offline — ничего не делаем
    if (instance.status === 'offline') {
      return BackgroundFetch.BackgroundFetchResult.NoData;
    }

    // Запускаем синхронизацию
    await instance.syncWhenOnline();

    return BackgroundFetch.BackgroundFetchResult.NewData;
  } catch (error) {
    console.error('[SyncService] Background task failed:', error);
    return BackgroundFetch.BackgroundFetchResult.Failed;
  }
});

// ──────────────────────────────────────────────────
// Типы
// ──────────────────────────────────────────────────

export type SyncStatus = 'online' | 'offline' | 'syncing';

export interface SyncState {
  status: SyncStatus;
  pendingCount: number;
  lastSyncAt: number | null;
  lastError: string | null;
  isBackgroundRegistered: boolean;
}

type SyncListener = (state: SyncState) => void;

// ──────────────────────────────────────────────────
// SyncService
// ──────────────────────────────────────────────────

class SyncService {
  private _status: SyncStatus = 'online';
  private _lastSyncAt: number | null = null;
  private _lastError: string | null = null;
  private _listeners: Set<SyncListener> = new Set();
  private _isSyncing = false;
  private _unsubscribeNetInfo: (() => void) | null = null;
  private _pendingCount: number = 0;
  private _pendingCountInterval: ReturnType<typeof setInterval> | null = null;
  private _initialized = false;
  private _isBackgroundRegistered = false;

  // ── Getters ────────────────────────────────────

  get status(): SyncStatus {
    return this._status;
  }

  get lastSyncAt(): number | null {
    return this._lastSyncAt;
  }

  get lastError(): string | null {
    return this._lastError;
  }

  get pendingCount(): number {
    return this._pendingCount;
  }

  // ── Init ───────────────────────────────────────

  async initialize(): Promise<void> {
    if (this._initialized) return;
    this._initialized = true;

    await initDatabase();

    // Подписка на изменения сети
    this._unsubscribeNetInfo = NetInfo.addEventListener(
      this._handleNetworkChange,
    );

    // Проверяем текущее состояние сети
    const netState = await NetInfo.fetch();
    this._updateStatus(netState.isConnected ?? true ? 'online' : 'offline');

    // Запускаем polling pendingCount
    await this._refreshPendingCount();
    this._pendingCountInterval = setInterval(() => {
      this._refreshPendingCount();
    }, 5_000);

    // Если онлайн — сразу пробуем синхронизировать
    if (this._status === 'online') {
      await this.syncWhenOnline();
    }
  }

  destroy(): void {
    if (this._unsubscribeNetInfo) {
      this._unsubscribeNetInfo();
      this._unsubscribeNetInfo = null;
    }
    if (this._pendingCountInterval) {
      clearInterval(this._pendingCountInterval);
      this._pendingCountInterval = null;
    }
    this._listeners.clear();
  }

  // ── Подписка на изменения статуса ──────────────

  subscribe(listener: SyncListener): () => void {
    this._listeners.add(listener);
    // Немедленно уведомляем о текущем состоянии
    listener(this._getState());
    return () => {
      this._listeners.delete(listener);
    };
  }

  // ── Синхронизация ──────────────────────────────

  /**
   * Push pending мутаций на сервер, затем pull свежих данных.
   * Вызывается при восстановлении соединения.
   */
  async syncWhenOnline(): Promise<void> {
    if (this._isSyncing) return;
    if (this._status === 'offline') return;

    this._isSyncing = true;
    this._setStatus('syncing');

    try {
      // Фаза 1: Push — отправляем pending мутации
      await this._pushPendingMutations();

      // Фаза 2: Pull — получаем свежие данные с сервера
      await this.pullLatestData();

      this._lastSyncAt = Date.now();
      this._lastError = null;
      this._setStatus('online');
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Unknown sync error';
      this._lastError = message;
      console.error('[SyncService] sync failed:', message);
      // Не меняем статус на offline — сеть есть, но сервер недоступен
      this._setStatus('online');
    } finally {
      this._isSyncing = false;
    }
  }

  /**
   * Pull свежих данных с сервера в SQLite кэш.
   */
  async pullLatestData(): Promise<void> {
    try {
      // Получаем work orders
      const orders = await workOrdersApi.getMyWorkOrders();
      await upsertWorkOrders(orders);

      // Получаем устройства для карты
      const devicesResponse = await devicesApi.getDevicesForMap();
      const devices: DeviceRow[] = devicesResponse.devices.map((d) => ({
        id: d.device_id,
        name: d.name,
        device_type: d.device_type,
        status: d.status,
        site_name: d.site_name ?? null,
        latitude: d.latitude,
        longitude: d.longitude,
        health: d.health,
        updated_at: new Date().toISOString(),
      }));

      // Перезаписываем кэш устройств
      await clearDevices();
      await upsertDevices(devices);

      // Очищаем sites пока нет отдельного API — данные приходят с devices
      await clearSites();

      console.log(
        `[SyncService] Pulled ${orders.length} work orders, ${devices.length} devices`,
      );
    } catch (error) {
      console.error('[SyncService] pullLatestData failed:', error);
      throw error;
    }
  }

  /**
   * Добавить мутацию в очередь синхронизации.
   * Если онлайн — сразу выполняем.
   */
  async enqueueMutation(params: {
    entityType: 'work_order' | 'device' | 'site';
    entityId: string;
    mutationType: 'create' | 'update' | 'delete';
    payload: Record<string, unknown>;
  }): Promise<void> {
    const id = await savePendingMutation({
      entity_type: params.entityType,
      entity_id: params.entityId,
      mutation_type: params.mutationType,
      payload: JSON.stringify(params.payload),
    });

    // Если онлайн — пробуем выполнить немедленно
    if (this._status === 'online' && !this._isSyncing) {
      await this.syncWhenOnline();
    }

    this._notifyListeners();
  }

  /**
   * Получить количество ожидающих мутаций.
   */
  async getPendingCount(): Promise<number> {
    return getPendingMutationCount();
  }

  // ── Background Sync Management ──────────────────

  /**
   * Зарегистрировать expo-background-fetch задачу.
   * Автоматически вызывается из useBackgroundSync при монтировании.
   */
  async startBackgroundSync(): Promise<boolean> {
    try {
      // Проверяем статус background fetch
      const bfStatus = await BackgroundFetch.getStatusAsync();

      if (bfStatus === BackgroundFetch.BackgroundFetchStatus.Denied) {
        console.warn('[SyncService] Background fetch denied');
        this._isBackgroundRegistered = false;
        this._notifyListeners();
        return false;
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
        console.log('[SyncService] Background sync registered (15min interval)');
      }

      this._isBackgroundRegistered = true;
      this._notifyListeners();
      return true;
    } catch (error) {
      console.error('[SyncService] Failed to start background sync:', error);
      this._isBackgroundRegistered = false;
      this._notifyListeners();
      return false;
    }
  }

  /**
   * Отменить регистрацию expo-background-fetch задачи.
   */
  async stopBackgroundSync(): Promise<boolean> {
    try {
      const registered = await TaskManager.isTaskRegisteredAsync(
        BACKGROUND_SYNC_TASK,
      );

      if (registered) {
        await BackgroundFetch.unregisterTaskAsync(BACKGROUND_SYNC_TASK);
        console.log('[SyncService] Background sync unregistered');
      }

      this._isBackgroundRegistered = false;
      this._notifyListeners();
      return true;
    } catch (error) {
      console.error('[SyncService] Failed to stop background sync:', error);
      return false;
    }
  }

  /**
   * Немедленная синхронизация независимо от текущего статуса.
   * В отличие от syncWhenOnline() — работает и возвращает результат.
   */
  async syncNow(): Promise<{ success: boolean; error?: string }> {
    if (this._isSyncing) {
      return { success: false, error: 'Sync already in progress' };
    }

    this._isSyncing = true;
    this._setStatus('syncing');

    try {
      // Фаза 1: Push — отправляем pending мутации
      await this._pushPendingMutations();

      // Фаза 2: Pull — получаем свежие данные с сервера
      await this.pullLatestData();

      this._lastSyncAt = Date.now();
      this._lastError = null;
      this._setStatus(this._status === 'offline' ? 'offline' : 'online');
      return { success: true };
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Unknown sync error';
      this._lastError = message;
      console.error('[SyncService] syncNow failed:', message);
      this._setStatus(this._status === 'offline' ? 'offline' : 'online');
      return { success: false, error: message };
    } finally {
      this._isSyncing = false;
    }
  }

  // ── Экстренное сохранение work order в offline ──

  /**
   * Сохранить work order локально (без синхронизации).
   * Используется когда пользователь offline и нужно обновить статус.
   */
  async saveWorkOrderLocally(wo: WorkOrder): Promise<void> {
    await upsertWorkOrder(wo);
  }

  /**
   * Получить work orders из локального кэша.
   */
  async getLocalWorkOrders(
    status?: WorkOrder['status'],
  ): Promise<WorkOrder[]> {
    return getWorkOrders(status);
  }

  // ── Private ────────────────────────────────────

  private async _pushPendingMutations(): Promise<void> {
    const mutations = await getPendingMutations();

    if (mutations.length === 0) return;

    console.log(`[SyncService] Pushing ${mutations.length} pending mutations`);

    for (const mutation of mutations) {
      try {
        await this._executeMutation(mutation);
        await removePendingMutation(mutation.id);
      } catch (error) {
        const message =
          error instanceof Error ? error.message : 'Unknown error';
        console.error(
          `[SyncService] Mutation ${mutation.id} failed: ${message}`,
        );
        await incrementPendingRetry(mutation.id, message);

        // После 3 неудачных попыток — пропускаем
        if (mutation.retry_count >= 2) {
          console.warn(
            `[SyncService] Dropping mutation ${mutation.id} after 3 retries`,
          );
          await removePendingMutation(mutation.id);
        }
      }
    }
  }

  private async _executeMutation(mutation: {
    entity_type: string;
    entity_id: string;
    mutation_type: string;
    payload: string;
  }): Promise<void> {
    const payload = JSON.parse(mutation.payload);

    switch (mutation.entity_type) {
      case 'work_order': {
        switch (mutation.mutation_type) {
          case 'update': {
            if (mutation.payload.includes('"status":"in_progress"')) {
              await workOrdersApi.startWorkOrder(mutation.entity_id);
            } else if (
              mutation.payload.includes('"status":"completed"')
            ) {
              await workOrdersApi.completeWorkOrder(
                mutation.entity_id,
                payload.payload as CompleteWorkOrderPayload,
              );
            }
            break;
          }
          default:
            console.warn(
              `[SyncService] Unsupported mutation: ${mutation.mutation_type} for ${mutation.entity_type}`,
            );
        }
        break;
      }
      default:
        console.warn(
          `[SyncService] Unsupported entity type: ${mutation.entity_type}`,
        );
    }
  }

  private _handleNetworkChange = (state: NetInfoState): void => {
    const isConnected = state.isConnected ?? false;

    if (isConnected && this._status === 'offline') {
      console.log('[SyncService] Network restored — starting sync');
      this._updateStatus('online');
      this.syncWhenOnline();
    } else if (!isConnected && this._status !== 'offline') {
      console.log('[SyncService] Network lost — going offline');
      this._updateStatus('offline');
    }
  };

  private _updateStatus(status: SyncStatus): void {
    this._status = status;
    this._notifyListeners();
  }

  private _setStatus(status: SyncStatus): void {
    this._status = status;
    this._notifyListeners();
  }

  private _notifyListeners(): void {
    const state = this._getState();
    for (const listener of this._listeners) {
      try {
        listener(state);
      } catch (error) {
        console.error('[SyncService] Listener error:', error);
      }
    }
  }

  private _getState(): SyncState {
    return {
      status: this._status,
      pendingCount: this._pendingCount,
      lastSyncAt: this._lastSyncAt,
      lastError: this._lastError,
      isBackgroundRegistered: this._isBackgroundRegistered,
    };
  }

  /**
   * Обновить кэшированное количество ожидающих мутаций.
   */
  private async _refreshPendingCount(): Promise<void> {
    try {
      const count = await getPendingMutationCount();
      if (count !== this._pendingCount) {
        this._pendingCount = count;
        this._notifyListeners();
      }
    } catch (error) {
      console.error('[SyncService] Failed to refresh pending count:', error);
    }
  }
}

// ──────────────────────────────────────────────────
// Singleton
// ──────────────────────────────────────────────────

export const syncService = new SyncService();
