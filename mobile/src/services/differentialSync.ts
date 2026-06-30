// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: Differential Sync Service (delta sync)
//
// DifferentialSyncService реализует полный sync cycle:
//   1. Pull — получить remote изменения с сервера (fetchDiff)
//   2. Apply — применить remote изменения в локальный SQLite
//   3. Collect — собрать локальные pending мутации
//   4. Resolve — разрешить конфликты (Last-Write-Wins)
//   5. Push — отправить локальные изменения на сервер (applyChanges)
//
// Интеграция:
//   - syncApi — HTTP клиент для backend sync endpoints
//   - offlineStorage — SQLite CRUD для work_orders, devices, sites
//   - useSyncStore — Zustand store для реактивного статуса
//
// Соответствует:
//   - IEC 62443-3-3 SR 3.1 (Queue-based processing)
//   - ISO 27001 A.12.4 (Audit trail)
// ═══════════════════════════════════════════════════════════════════════════

import * as SQLite from 'expo-sqlite';
import { syncApi, SyncChange, SyncDiffResponse } from '../api/sync';
import {
  upsertWorkOrders,
  deleteWorkOrder,
  upsertDevices,
  upsertSites,
  getPendingMutations,
  savePendingMutation,
  removePendingMutation,
  incrementPendingRetry,
  PendingMutation,
} from './offlineStorage';
import { WorkOrder } from '../types';

// ── Типы ────────────────────────────────────────────────────────────────

/** Результат синхронизации одной сущности */
export interface EntitySyncResult {
  entity: string;
  changesApplied: number;
  changesPushed: number;
  errors: number;
  durationMs: number;
}

/** Результат полного sync cycle */
export interface SyncResult {
  success: boolean;
  pullChanges: number;
  pushChanges: number;
  conflictsResolved: number;
  entities: EntitySyncResult[];
  totalDurationMs: number;
  error: string | null;
}

/** Тип конфликта */
export interface ConflictEntry {
  local: SyncChange;
  remote: SyncChange;
  resolved: boolean;
  resolution: 'local' | 'remote' | null;
}

// ── Константы ──────────────────────────────────────────────────────────

const DB_NAME = 'cctv-offline.db';
const MAX_RETRY_COUNT = 5;

/** Задержки exponential backoff (в ms) */
const RETRY_DELAYS = [1_000, 2_000, 4_000, 8_000, 16_000];

// ── DifferentialSyncService ─────────────────────────────────────────────

export class DifferentialSyncService {
  private db: SQLite.SQLiteDatabase | null = null;
  private lastSyncTime: string = '';

  /**
   * Полный sync cycle:
   * 1. Pull remote changes → apply локально
   * 2. Collect local changes → resolve conflicts → push на сервер
   */
  async sync(): Promise<SyncResult> {
    const startTime = Date.now();
    const entityResults: EntitySyncResult[] = [];
    let pullChanges = 0;
    let pushChanges = 0;
    let conflictsResolved = 0;

    try {
      // ── Фаза 1: Pull ──────────────────────────────────────────────
      const remoteResult = await this._pullPhase(entityResults);
      pullChanges = remoteResult.changesApplied;

      // ── Фаза 2: Push ──────────────────────────────────────────────
      const pushResult = await this._pushPhase(entityResults);
      pushChanges = pushResult.changesPushed;
      conflictsResolved = pushResult.conflictsResolved;

      const totalDurationMs = Date.now() - startTime;
      const hasError = entityResults.some((r) => r.errors > 0);

      return {
        success: !hasError,
        pullChanges,
        pushChanges,
        conflictsResolved,
        entities: entityResults,
        totalDurationMs,
        error: hasError ? 'One or more entities failed to sync' : null,
      };
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Unknown sync error';

      return {
        success: false,
        pullChanges,
        pushChanges,
        conflictsResolved,
        entities: entityResults,
        totalDurationMs: Date.now() - startTime,
        error: message,
      };
    }
  }

  /**
   * Применить remote изменения в локальный SQLite.
   * Вызывается из sync() или отдельно для принудительного apply.
   */
  async applyRemoteChanges(changes: SyncChange[]): Promise<void> {
    await this._ensureDb();

    for (const change of changes) {
      try {
        await this._applyChange(change);
      } catch (error) {
        console.error(
          `[DifferentialSyncService] Failed to apply change: ${change.table}:${change.id}`,
          error,
        );
      }
    }
  }

  /**
   * Собрать локальные изменения из pending_sync таблицы.
   * Возвращает массив SyncChange для отправки на сервер.
   */
  async collectLocalChanges(): Promise<SyncChange[]> {
    const mutations = await getPendingMutations();
    return mutations.map((m) => this._mutationToChange(m));
  }

  /**
   * Разрешить конфликты между локальными и remote изменениями.
   * Стратегия: Last-Write-Wins (LWW).
   * Возвращает массив победивших изменений для отправки на сервер.
   */
  async resolveConflicts(
    local: SyncChange[],
    remote: SyncChange[],
  ): Promise<SyncChange[]> {
    const resolved: SyncChange[] = [];
    const remoteMap = new Map<string, SyncChange>();

    for (const r of remote) {
      remoteMap.set(`${r.table}:${r.id}`, r);
    }

    for (const l of local) {
      const key = `${l.table}:${l.id}`;
      const r = remoteMap.get(key);

      if (!r) {
        // Нет конфликта — локальное изменение уникально
        resolved.push(l);
        continue;
      }

      // LWW: сравниваем timestamp
      const localTime = new Date(l.timestamp).getTime();
      const remoteTime = new Date(r.timestamp).getTime();

      if (localTime >= remoteTime) {
        // Локальное новее или равно — побеждает локальное
        resolved.push(l);
      }
      // Если remote новее — отбрасываем локальное
    }

    return resolved;
  }

  // ── Private: Pull Phase ────────────────────────────────────────────

  /**
   * Фаза pull: запросить remote изменения и применить локально.
   */
  private async _pullPhase(
    results: EntitySyncResult[],
  ): Promise<{ changesApplied: number }> {
    let totalApplied = 0;

    const entities = ['work_orders', 'devices', 'photos', 'audit'];

    for (const entity of entities) {
      const entityStart = Date.now();
      let changesApplied = 0;
      let errors = 0;

      try {
        const diff = await syncApi.fetchDiff(this.lastSyncTime, {
          entities: [entity],
        });

        if (diff.changes.length > 0) {
          await this.applyRemoteChanges(diff.changes);
          changesApplied = diff.changes.length;
        }

        // Обновляем lastSyncTime из ответа сервера
        if (diff.serverTime > this.lastSyncTime) {
          this.lastSyncTime = diff.serverTime;
        }
      } catch (error) {
        console.error(
          `[DifferentialSyncService] Pull failed for ${entity}:`,
          error,
        );
        errors = 1;
      }

      results.push({
        entity,
        changesApplied,
        changesPushed: 0,
        errors,
        durationMs: Date.now() - entityStart,
      });

      totalApplied += changesApplied;
    }

    return { changesApplied: totalApplied };
  }

  /**
   * Фаза push: собрать локальные изменения, разрешить конфликты, отправить.
   */
  private async _pushPhase(
    results: EntitySyncResult[],
  ): Promise<{
    changesPushed: number;
    conflictsResolved: number;
  }> {
    let totalPushed = 0;
    let totalConflicts = 0;

    try {
      // 1. Собираем локальные изменения
      const localChanges = await this.collectLocalChanges();
      if (localChanges.length === 0) {
        return { changesPushed: 0, conflictsResolved: 0 };
      }

      // 2. Получаем последний diff с сервера для conflict detection
      const remoteDiff = await syncApi.fetchDiff(this.lastSyncTime, {
        entities: [...new Set(localChanges.map((c) => c.table))],
      });

      // 3. Разрешаем конфликты (LWW)
      const resolved = await this.resolveConflicts(
        localChanges,
        remoteDiff.changes,
      );
      totalConflicts = localChanges.length - resolved.length;

      // 4. Отправляем победившие изменения на сервер
      if (resolved.length > 0) {
        const applyResult = await this._applyWithRetry(resolved);

        // 5. Удаляем успешно отправленные из pending_sync
        for (const change of resolved) {
          await this._removePendingForChange(change);
        }

        totalPushed = applyResult.applied;
      }
    } catch (error) {
      console.error(
        '[DifferentialSyncService] Push phase failed:',
        error,
      );
    }

    // Обновляем результаты для entity results
    for (const result of results) {
      result.changesPushed = totalPushed;
    }

    return { changesPushed: totalPushed, conflictsResolved: totalConflicts };
  }

  /**
   * Отправить изменения на сервер с exponential backoff retry.
   */
  private async _applyWithRetry(
    changes: SyncChange[],
    attempt: number = 0,
  ): Promise<{ applied: number }> {
    try {
      const result = await syncApi.applyChanges(changes);
      return { applied: result.applied };
    } catch (error) {
      if (attempt < MAX_RETRY_COUNT - 1) {
        const delay = RETRY_DELAYS[attempt] ?? RETRY_DELAYS[RETRY_DELAYS.length - 1];
        await this._sleep(delay);
        return this._applyWithRetry(changes, attempt + 1);
      }
      throw error;
    }
  }

  // ── Private: Apply Change ──────────────────────────────────────────

  /**
   * Применить одно изменение к локальной БД.
   */
  private async _applyChange(change: SyncChange): Promise<void> {
    switch (change.table) {
      case 'work_orders':
        await this._applyWorkOrderChange(change);
        break;

      case 'devices':
        await this._applyDeviceChange(change);
        break;

      case 'photos':
        // Фото — часть work_order, обновляем родительский WO
        if (change.data?.work_order_id) {
          // При изменении фото — пересинхронизируем work_order
          console.log(
            `[DifferentialSyncService] Photo changed for WO ${change.data.work_order_id}, will resync`,
          );
        }
        break;

      case 'audit':
        // Audit лог не хранится локально
        break;

      default:
        console.warn(
          `[DifferentialSyncService] Unknown table: ${change.table}`,
        );
    }
  }

  /**
   * Применить изменение work_order.
   */
  private async _applyWorkOrderChange(change: SyncChange): Promise<void> {
    switch (change.operation) {
      case 'insert':
      case 'update': {
        const wo = this._buildWorkOrder(change.id, change.data);
        await upsertWorkOrders([wo]);
        break;
      }
      case 'delete': {
        await deleteWorkOrder(change.id);
        break;
      }
    }
  }

  /**
   * Применить изменение device.
   */
  private async _applyDeviceChange(change: SyncChange): Promise<void> {
    switch (change.operation) {
      case 'insert':
      case 'update': {
        const device = this._buildDeviceRow(change.id, change.data);
        await upsertDevices([device]);
        break;
      }
      case 'delete':
        // Устройства не удаляются из локального кэша
        console.log(
          `[DifferentialSyncService] Device ${change.id} deleted remotely`,
        );
        break;
    }
  }

  // ── Private: Helpers ───────────────────────────────────────────────

  /**
   * Построить WorkOrder из SyncChange.data.
   */
  private _buildWorkOrder(
    id: string,
    data: Record<string, unknown>,
  ): WorkOrder {
    return {
      id,
      schedule_id: (data.schedule_id as string) ?? undefined,
      device_id: (data.device_id as string) ?? '',
      device_name: (data.device_name as string) ?? undefined,
      site_name: (data.site_name as string) ?? undefined,
      type: (data.type as WorkOrder['type']) ?? 'preventive',
      status: (data.status as WorkOrder['status']) ?? 'open',
      priority: (data.priority as WorkOrder['priority']) ?? 'medium',
      assigned_to: (data.assigned_to as string) ?? undefined,
      sla_deadline: (data.sla_deadline as string) ?? undefined,
      checklist: (data.checklist as WorkOrder['checklist']) ?? [],
      started_at: (data.started_at as string) ?? undefined,
      completed_at: (data.completed_at as string) ?? undefined,
      notes: (data.notes as string) ?? undefined,
      photos: (data.photos as WorkOrder['photos']) ?? [],
      parts_used: (data.parts_used as WorkOrder['parts_used']) ?? [],
      created_by: (data.created_by as string) ?? undefined,
      created_at:
        (data.created_at as string) ?? new Date().toISOString(),
      updated_at:
        (data.updated_at as string) ?? new Date().toISOString(),
      device_name_display:
        (data.device_name_display as string) ?? undefined,
      assignee_name: (data.assignee_name as string) ?? undefined,
      sla_status: (data.sla_status as string) ?? undefined,
    };
  }

  /**
   * Построить DeviceRow из SyncChange.data.
   */
  private _buildDeviceRow(
    id: string,
    data: Record<string, unknown>,
  ): import('./offlineStorage').DeviceRow {
    return {
      id,
      name: (data.name as string) ?? '',
      device_type: (data.device_type as string) ?? '',
      status: (data.status as string) ?? 'OFFLINE',
      site_name: (data.site_name as string) ?? null,
      latitude: (data.latitude as number) ?? 0,
      longitude: (data.longitude as number) ?? 0,
      health: (data.health as string) ?? 'healthy',
      updated_at:
        (data.updated_at as string) ?? new Date().toISOString(),
    };
  }

  /**
   * Преобразовать PendingMutation → SyncChange.
   */
  private _mutationToChange(m: PendingMutation): SyncChange {
    let payload: Record<string, unknown> = {};
    try {
      payload = JSON.parse(m.payload);
    } catch {
      payload = {};
    }

    return {
      table: this._entityToTable(m.entity_type),
      id: m.entity_id,
      operation: this._mutationToOperation(m.mutation_type),
      data: payload,
      timestamp: new Date(m.timestamp).toISOString(),
    };
  }

  /**
   * Преобразовать entity_type → table name.
   */
  private _entityToTable(
    entityType: PendingMutation['entity_type'],
  ): string {
    switch (entityType) {
      case 'work_order':
        return 'work_orders';
      case 'device':
        return 'devices';
      case 'site':
        return 'sites';
    }
  }

  /**
   * Преобразовать mutation_type → SyncChange operation.
   */
  private _mutationToOperation(
    mutationType: PendingMutation['mutation_type'],
  ): SyncChange['operation'] {
    switch (mutationType) {
      case 'create':
        return 'insert';
      case 'update':
        return 'update';
      case 'delete':
        return 'delete';
    }
  }

  /**
   * Удалить pending запись для отправленного изменения.
   */
  private async _removePendingForChange(
    change: SyncChange,
  ): Promise<void> {
    try {
      const entityType = this._tableToEntity(change.table);
      const mutations = await getPendingMutations();

      const match = mutations.find(
        (m) =>
          m.entity_type === entityType && m.entity_id === change.id,
      );

      if (match) {
        await removePendingMutation(match.id);
      }
    } catch (error) {
      console.warn(
        `[DifferentialSyncService] Failed to remove pending for ${change.id}:`,
        error,
      );
    }
  }

  /**
   * Преобразовать table name → entity_type.
   */
  private _tableToEntity(
    table: string,
  ): PendingMutation['entity_type'] {
    switch (table) {
      case 'work_orders':
        return 'work_order';
      case 'devices':
        return 'device';
      case 'sites':
        return 'site';
      default:
        return 'work_order';
    }
  }

  // ── Private: Database ──────────────────────────────────────────────

  /**
   * Убедиться, что БД инициализирована.
   */
  private async _ensureDb(): Promise<SQLite.SQLiteDatabase> {
    if (!this.db) {
      this.db = await SQLite.openDatabaseAsync(DB_NAME);
    }
    return this.db;
  }

  // ── Private: Utils ─────────────────────────────────────────────────

  private _sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

// ── Singleton ───────────────────────────────────────────────────────────

/** Глобальный экземпляр DifferentialSyncService */
export const differentialSyncService = new DifferentialSyncService();
