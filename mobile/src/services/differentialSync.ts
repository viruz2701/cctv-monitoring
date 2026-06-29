// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: Differential Sync for Mobile (delta sync)
//
// DifferentialSync реализует:
//   - Priority-based sync (work_orders → devices → photos → audit)
//   - Delta sync (только changed fields с момента lastSync)
//   - Gzip/Brotli compression support
//   - Bandwidth monitoring
//   - Apply delta в локальный SQLite кэш
//
// Интегрируется с существующим syncService.ts:
//   - DifferentialSync используется как pull-стратегия
//   - syncService.ts продолжает управлять push-мутациями и background sync
//
// ═══════════════════════════════════════════════════════════════════════════
import AsyncStorage from '@react-native-async-storage/async-storage';
import { apiClient } from '../api/client';
import {
  upsertWorkOrders,
  deleteWorkOrder,
  upsertDevices,
  clearDevices,
  clearSites,
  upsertSites,
  DeviceRow,
  SiteRow,
} from './offlineStorage';
import { WorkOrder } from '../types';

// ── Типы ────────────────────────────────────────────────────────────────

/** Тип сжатия ответа */
export type CompressionType = 'gzip' | 'brotli' | 'none';

/** Приоритет синхронизации — сущности с более высоким приоритетом синхронизируются первыми */
export type SyncPriority = 'work_orders' | 'devices' | 'photos' | 'audit';

/** Запись изменения от сервера */
export interface ChangeEntry {
  id: string;
  type: 'created' | 'updated' | 'deleted';
  entity: string;
  fields?: Record<string, unknown>;
  updated_at: string;
}

/** Ответ сервера с дельтой */
export interface DeltaResponse {
  changes: ChangeEntry[];
  timestamp: string;
  compressed: boolean;
  entity: string;
  has_more: boolean;
  total_count: number;
}

/** Статус синхронизации (bandwidth usage, last sync) */
export interface SyncStatusResponse {
  bandwidth_usage_bytes: number;
  last_sync_at: Record<string, string>;
  total_syncs: number;
  total_changes: number;
}

/** Результат синхронизации одной сущности */
export interface EntitySyncResult {
  entity: string;
  changesApplied: number;
  bytesReceived: number;
  compressed: boolean;
  durationMs: number;
  error: string | null;
}

/** Результат полной синхронизации */
export interface SyncResult {
  success: boolean;
  entities: EntitySyncResult[];
  totalChanges: number;
  totalBytes: number;
  durationMs: number;
  error: string | null;
}

/** Состояние дифференциальной синхронизации */
export interface DifferentialSyncState {
  lastSync: Record<string, string>; // entity → ISO8601 timestamp
  totalBytesSaved: number; // estimated bytes saved vs full sync
  totalSyncs: number;
}

// ── Константы ──────────────────────────────────────────────────────────

/** Приоритет синхронизации — порядок имеет значение */
const DEFAULT_PRIORITY: SyncPriority[] = [
  'work_orders',
  'devices',
  'photos',
  'audit',
];

/** Максимальное количество записей на страницу */
const DEFAULT_PAGE_SIZE = 500;

/** Таймаут для одного sync-запроса (30s) */
const SYNC_REQUEST_TIMEOUT = 30_000;

// ── DifferentialSync ───────────────────────────────────────────────────

export class DifferentialSync {
  private lastSync: Record<string, string> = {};
  private priority: SyncPriority[];
  private compression: CompressionType;
  private totalBytesSaved: number = 0;
  private totalSyncs: number = 0;

  constructor(options?: {
    priority?: SyncPriority[];
    compression?: CompressionType;
  }) {
    this.priority = options?.priority ?? DEFAULT_PRIORITY;
    this.compression = options?.compression ?? 'gzip';

    // Восстанавливаем lastSync из AsyncStorage при инициализации
    this._loadState();
  }

  // ── Public API ──────────────────────────────────────────────────────

  /**
   * Полная синхронизация всех сущностей по приоритету.
   * Выполняет delta sync для каждой сущности последовательно.
   */
  async sync(): Promise<SyncResult> {
    const startTime = Date.now();
    const entityResults: EntitySyncResult[] = [];
    let totalChanges = 0;
    let totalBytes = 0;

    for (const entity of this.priority) {
      const result = await this._syncEntity(entity);
      entityResults.push(result);
      totalChanges += result.changesApplied;
      totalBytes += result.bytesReceived;
    }

    this.totalSyncs++;
    await this._saveState();

    const durationMs = Date.now() - startTime;
    const hasError = entityResults.some((r) => r.error !== null);

    return {
      success: !hasError,
      entities: entityResults,
      totalChanges,
      totalBytes,
      durationMs,
      error: hasError
        ? 'One or more entities failed to sync'
        : null,
    };
  }

  /**
   * Синхронизация одной конкретной сущности.
   * Полезна для принудительной синхронизации после мутации.
   */
  async syncEntity(entity: SyncPriority): Promise<EntitySyncResult> {
    return this._syncEntity(entity);
  }

  /**
   * Получить статус синхронизации с сервера.
   */
  async getServerStatus(): Promise<SyncStatusResponse | null> {
    try {
      const response = await apiClient.get<SyncStatusResponse>(
        '/sync/status',
        { timeout: SYNC_REQUEST_TIMEOUT },
      );
      return response.data;
    } catch (error) {
      console.error('[DifferentialSync] Failed to get server status:', error);
      return null;
    }
  }

  /**
   * Получить текущее состояние синхронизации клиента.
   */
  getState(): DifferentialSyncState {
    return {
      lastSync: { ...this.lastSync },
      totalBytesSaved: this.totalBytesSaved,
      totalSyncs: this.totalSyncs,
    };
  }

  /**
   * Сбросить lastSync для всех сущностей (вызовет full resync при следующем sync).
   */
  resetSyncState(): void {
    this.lastSync = {};
    this.totalBytesSaved = 0;
    this.totalSyncs = 0;
    this._saveState();
  }

  /**
   * Установить временную метку lastSync для конкретной сущности.
   */
  setLastSync(entity: string, timestamp: string): void {
    this.lastSync[entity] = timestamp;
    this._saveState();
  }

  // ── Private ─────────────────────────────────────────────────────────

  /**
   * Синхронизировать одну сущность с пагинацией (has_more).
   */
  private async _syncEntity(entity: string): Promise<EntitySyncResult> {
    const startTime = Date.now();
    let changesApplied = 0;
    let bytesReceived = 0;
    let hasError: string | null = null;
    let compressed = false;

    try {
      let hasMore = true;
      let currentSince = this.lastSync[entity];

      while (hasMore) {
        const result = await this._fetchDelta(entity, currentSince);

        if (!result) {
          hasError = 'Failed to fetch delta';
          break;
        }

        compressed = result.compressed;
        bytesReceived += this._estimatePayloadSize(result);

        // Apply changes to local storage
        await this._applyDelta(entity, result);

        changesApplied += result.changes.length;
        hasMore = result.has_more;

        // Update cursor for next page
        if (hasMore) {
          currentSince = result.timestamp;
        } else {
          this.lastSync[entity] = result.timestamp;
        }
      }

      // Estimate bytes saved vs full sync (heuristic: full sync ≈ 10x delta)
      this.totalBytesSaved += bytesReceived * 9;
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Unknown sync error';
      console.error(
        `[DifferentialSync] Failed to sync ${entity}:`,
        message,
      );
      hasError = message;
    }

    return {
      entity,
      changesApplied,
      bytesReceived,
      compressed,
      durationMs: Date.now() - startTime,
      error: hasError,
    };
  }

  /**
   * Запросить delta с сервера.
   */
  private async _fetchDelta(
    entity: string,
    since?: string,
  ): Promise<DeltaResponse | null> {
    try {
      const params = new URLSearchParams();

      if (since) {
        params.set('since', since);
      }

      if (this.compression !== 'none') {
        params.set('compression', this.compression);
      }

      params.set('page_size', String(DEFAULT_PAGE_SIZE));

      const response = await apiClient.get<DeltaResponse>(
        `/sync/${entity}?${params.toString()}`,
        {
          timeout: SYNC_REQUEST_TIMEOUT,
          // Если сервер поддерживает сжатие на уровне HTTP, используем Accept-Encoding
          headers: {
            'Accept-Encoding': this.compression === 'none'
              ? 'identity'
              : 'gzip, deflate',
          },
          // Распознаём сжатый ответ автоматически
          decompress: true,
        },
      );

      return response.data;
    } catch (error) {
      console.error(
        `[DifferentialSync] fetchDelta failed for ${entity}:`,
        error,
      );
      return null;
    }
  }

  /**
   * Применить delta к локальному SQLite кэшу.
   */
  private async _applyDelta(
    entity: string,
    delta: DeltaResponse,
  ): Promise<void> {
    for (const change of delta.changes) {
      try {
        switch (change.type) {
          case 'created':
          case 'updated':
            await this._upsertEntity(entity, change);
            break;
          case 'deleted':
            await this._deleteEntity(entity, change);
            break;
        }
      } catch (error) {
        console.error(
          `[DifferentialSync] Failed to apply change ${change.id} for ${entity}:`,
          error,
        );
        // Продолжаем с остальными изменениями
      }
    }
  }

  /**
   * Upsert (create or update) сущности в локальном кэше.
   */
  private async _upsertEntity(
    entity: string,
    change: ChangeEntry,
  ): Promise<void> {
    switch (entity) {
      case 'work_orders': {
        if (change.fields) {
          // Собираем полный WorkOrder из delta-полей
          const wo = this._buildWorkOrder(change.id, change.fields);
          await upsertWorkOrders([wo]);
        }
        break;
      }

      case 'devices': {
        if (change.fields) {
          const device = this._buildDeviceRow(change.id, change.fields);
          await upsertDevices([device]);
        }
        break;
      }

      case 'photos': {
        // Photos хранятся как часть work_orders в поле photos[]
        // При создании/обновлении фото — пересинхронизируем work_order
        if (change.fields?.work_order_id) {
          await this.syncEntity('work_orders');
        }
        break;
      }

      case 'audit': {
        // Audit лог хранится локально только для отладки — не upsert'им
        console.log(
          `[DifferentialSync] Audit entry: ${change.fields?.action} for ${change.fields?.entity_type}:${change.fields?.entity_id}`,
        );
        break;
      }
    }
  }

  /**
   * Удалить сущность из локального кэша.
   */
  private async _deleteEntity(
    entity: string,
    change: ChangeEntry,
  ): Promise<void> {
    switch (entity) {
      case 'work_orders':
        await deleteWorkOrder(change.id);
        break;

      case 'devices':
        // Для devices удаление не поддерживается в offlineStorage
        // Просто логируем
        console.log(
          `[DifferentialSync] Device ${change.id} deleted remotely`,
        );
        break;

      case 'photos':
        // Фото — не удаляем отдельно
        break;

      case 'audit':
        // Audit — не удаляем
        break;
    }
  }

  /**
   * Собрать WorkOrder из delta-полей.
   */
  private _buildWorkOrder(
    id: string,
    fields: Record<string, unknown>,
  ): WorkOrder {
    return {
      id,
      schedule_id: (fields.schedule_id as string) ?? undefined,
      device_id: (fields.device_id as string) ?? '',
      device_name: (fields.device_name as string) ?? undefined,
      site_name: (fields.site_name as string) ?? undefined,
      type: (fields.type as WorkOrder['type']) ?? 'preventive',
      status: (fields.status as WorkOrder['status']) ?? 'open',
      priority: (fields.priority as WorkOrder['priority']) ?? 'medium',
      assigned_to: (fields.assigned_to as string) ?? undefined,
      sla_deadline: (fields.sla_deadline as string) ?? undefined,
      checklist: (fields.checklist as WorkOrder['checklist']) ?? [],
      started_at: (fields.started_at as string) ?? undefined,
      completed_at: (fields.completed_at as string) ?? undefined,
      notes: (fields.notes as string) ?? undefined,
      photos: (fields.photos as WorkOrder['photos']) ?? [],
      parts_used: (fields.parts_used as WorkOrder['parts_used']) ?? [],
      created_by: (fields.created_by as string) ?? undefined,
      created_at: (fields.created_at as string) ?? new Date().toISOString(),
      updated_at: (fields.updated_at as string) ?? new Date().toISOString(),
      device_name_display: (fields.device_name_display as string) ?? undefined,
      assignee_name: (fields.assignee_name as string) ?? undefined,
      sla_status: (fields.sla_status as string) ?? undefined,
    };
  }

  /**
   * Собрать DeviceRow из delta-полей.
   */
  private _buildDeviceRow(
    id: string,
    fields: Record<string, unknown>,
  ): DeviceRow {
    return {
      id,
      name: (fields.name as string) ?? '',
      device_type: (fields.device_type as string) ?? '',
      status: (fields.status as string) ?? 'OFFLINE',
      site_name: (fields.site_name as string) ?? null,
      latitude: (fields.latitude as number) ?? 0,
      longitude: (fields.longitude as number) ?? 0,
      health: (fields.health as string) ?? 'healthy',
      updated_at: (fields.updated_at as string) ?? new Date().toISOString(),
    };
  }

  /**
   * Оценить размер payload'а в байтах.
   */
  private _estimatePayloadSize(delta: DeltaResponse): number {
    // Примерный расчёт: JSON.stringify размер
    try {
      return new TextEncoder().encode(JSON.stringify(delta)).length;
    } catch {
      // Fallback: грубая оценка
      return delta.total_count * 200;
    }
  }

  // ── State Persistence ───────────────────────────────────────────────

  /**
   * Загрузить состояние из AsyncStorage.
   */
  private async _loadState(): Promise<void> {
    try {
      const saved = await AsyncStorage.getItem(
        'cctv_differential_sync_state',
      );
      if (saved) {
        const state: DifferentialSyncState = JSON.parse(saved);
        this.lastSync = state.lastSync ?? {};
        this.totalBytesSaved = state.totalBytesSaved ?? 0;
        this.totalSyncs = state.totalSyncs ?? 0;
      }
    } catch (error) {
      console.warn(
        '[DifferentialSync] Failed to load state:',
        error,
      );
    }
  }

  /**
   * Сохранить состояние в AsyncStorage.
   */
  private async _saveState(): Promise<void> {
    try {
      await AsyncStorage.setItem(
        'cctv_differential_sync_state',
        JSON.stringify(this.getState()),
      );
    } catch (error) {
      console.warn(
        '[DifferentialSync] Failed to save state:',
        error,
      );
    }
  }
}

// ── Singleton ───────────────────────────────────────────────────────────

/** Глобальный экземпляр DifferentialSync */
export const differentialSync = new DifferentialSync({
  compression: 'gzip',
  priority: ['work_orders', 'devices', 'photos', 'audit'],
});
