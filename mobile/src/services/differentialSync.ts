// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: Differential Sync Service (delta sync)
//
// DifferentialSyncService реализует полный sync cycle:
//   1. Pull — получить remote изменения с сервера (fetchDiff)
//   2. Apply — применить remote изменения в локальный SQLite
//   3. Collect — собрать локальные pending мутации
//   4. Resolve — разрешить конфликты (3-way merge + server authority)
//   5. Push — отправить локальные изменения на сервер (applyChanges)
//
// 3-Way Merge Policy (P0-CR-05):
//   - Server-authoritative поля: status, priority, sla_deadline, sla_status, assigned_to
//     → всегда принимается серверная версия
//   - Client-authoritative поля: notes, parts_used, checklist, photos
//     → сохраняется локальная версия; если обе стороны изменили → ConflictResolution UI
//   - Immutable поля: device_id, type, created_by, created_at
//     → не участвуют в merge
//
// Интеграция:
//   - syncApi — HTTP клиент для backend sync endpoints
//   - offlineStorage — SQLite CRUD для work_orders, devices, sites
//   - useSyncStore — Zustand store для реактивного статуса
//   - ConflictResolutionModal — UI для ручного разрешения конфликтов
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
import type { ConflictField } from '../components/ConflictResolutionModal';

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
  /** Неразрешённые конфликты — требуется участие пользователя */
  unresolvedConflicts: ThreeWayConflict[];
  entities: EntitySyncResult[];
  totalDurationMs: number;
  error: string | null;
}

/** Тип конфликта (LWW — сохранён для обратной совместимости) */
export interface ConflictEntry {
  local: SyncChange;
  remote: SyncChange;
  resolved: boolean;
  resolution: 'local' | 'remote' | null;
}

/**
 * Конфликт 3-way merge, требующий UI resolution.
 * Используется ConflictResolutionModal для отображения diff.
 */
export interface ThreeWayConflict {
  /** ID сущности (work_order_id) */
  id: string;
  /** Человекочитаемое название */
  label: string;
  /** Локальный timestamp */
  localTimestamp: number;
  /** Серверный timestamp */
  serverTimestamp: number;
  /** Список конфликтующих полей (client-authoritative) */
  fields: ConflictField[];
}

// ── Константы ──────────────────────────────────────────────────────────

const DB_NAME = 'cctv-offline.db';
const MAX_RETRY_COUNT = 5;

/** Задержки exponential backoff (в ms) */
const RETRY_DELAYS = [1_000, 2_000, 4_000, 8_000, 16_000];

/** Максимальный случайный jitter для backoff (в ms) */
const JITTER_MAX_MS = 1_000;

// ── 3-Way Merge Field Classification (P0-CR-05) ─────────────────────────
//
// Server-authoritative поля — всегда принимается серверная версия.
// Клиентские изменения этих полей отбрасываются при конфликте.
const SERVER_AUTHORITATIVE_FIELDS = new Set([
  'status',
  'priority',
  'sla_deadline',
  'sla_status',
  'assigned_to',
]);

// Client-authoritative поля — локальная версия сохраняется.
// Если обе стороны изменили одно поле → создаётся ConflictEntry для UI.
const CLIENT_AUTHORITATIVE_FIELDS = new Set([
  'notes',
  'parts_used',
  'checklist',
  'photos',
]);

// Immutable поля — не участвуют в merge, игнорируются при сравнении.
const IMMUTABLE_FIELDS = new Set([
  'id',
  'device_id',
  'type',
  'created_by',
  'created_at',
  'schedule_id',
]);

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
        unresolvedConflicts: pushResult.unresolvedConflicts,
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
        unresolvedConflicts: [],
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
   * Стратегия: 3-way merge + server authority (P0-CR-05).
   *
   * Server-authoritative поля (status, priority, sla_deadline, sla_status, assigned_to):
   *   → всегда принимается серверная версия
   * Client-authoritative поля (notes, parts_used, checklist, photos):
   *   → если обе стороны изменили одно поле → ThreeWayConflict для UI
   *   → если только локальная сторона → сохраняется локальная версия
   *
   * Возвращает:
   *   resolved — изменения для отправки на сервер (смерженные)
   *   conflicts — конфликты, требующие UI resolution
   */
  async resolveConflicts3Way(
    local: SyncChange[],
    remote: SyncChange[],
  ): Promise<{
    resolved: SyncChange[];
    conflicts: ThreeWayConflict[];
  }> {
    const resolved: SyncChange[] = [];
    const conflicts: ThreeWayConflict[] = [];
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

      // Конфликт обнаружен — выполняем 3-way merge
      const mergeResult = this._threeWayMerge(l, r);

      if (mergeResult.resolvedChange) {
        resolved.push(mergeResult.resolvedChange);
      }
      if (mergeResult.conflict) {
        conflicts.push(mergeResult.conflict);
      }
    }

    return { resolved, conflicts };
  }

  /**
   * 3-way merge одного конфликтующего изменения (P0-CR-05).
   *
   * Алгоритм:
   * 1. Берём remote.data как базовую версию
   * 2. Для каждого поля из local.changedFields:
   *    a. Server-authoritative → оставляем remote значение
   *    b. Client-authoritative, изменено на обеих сторонах и значения разные → Conflict
   *    c. Client-authoritative, изменено только локально → берём local значение
   *    d. Immutable → игнорируем
   * 3. Поля, изменённые только на сервере → остаются серверные
   *
   * Возвращает смерженный SyncChange и опциональный ThreeWayConflict.
   */
  private _threeWayMerge(
    local: SyncChange,
    remote: SyncChange,
  ): {
    resolvedChange: SyncChange | null;
    conflict: ThreeWayConflict | null;
  } {
    // Определяем, какие поля изменила каждая сторона
    const localFields = new Set(
      local.changedFields ?? Object.keys(local.data),
    );
    const remoteFields = new Set(
      remote.changedFields ?? Object.keys(remote.data),
    );

    // Начинаем с remote как базовой версии
    const mergedData: Record<string, unknown> = { ...remote.data };
    const conflictingFields: ConflictField[] = [];
    let hasClientChanges = false;

    // Проходим по всем полям, изменённым локально
    for (const field of localFields) {
      if (IMMUTABLE_FIELDS.has(field)) {
        continue;
      }

      if (SERVER_AUTHORITATIVE_FIELDS.has(field)) {
        // Server wins — оставляем remote значение (уже в mergedData)
        continue;
      }

      if (CLIENT_AUTHORITATIVE_FIELDS.has(field)) {
        const localVal = local.data[field];
        const remoteVal = remote.data[field];
        const bothChanged = remoteFields.has(field);
        const valuesDiffer =
          JSON.stringify(localVal) !== JSON.stringify(remoteVal);

        if (bothChanged && valuesDiffer) {
          // Обе стороны изменили одно client-authoritative поле → конфликт
          conflictingFields.push({
            name: field,
            localValue: this._stringifyValue(localVal),
            serverValue: this._stringifyValue(remoteVal),
            isChanged: true,
          });
          // В mergedData кладём local значение (может быть изменено UI)
          mergedData[field] = localVal;
          hasClientChanges = true;
        } else {
          // Только локальная сторона изменила (или значения совпадают)
          mergedData[field] = localVal;
          hasClientChanges = true;
        }
        continue;
      }

      // Другие поля (не классифицированные) — берем локальное значение
      // если оно отличается от remote
      if (remoteFields.has(field)) {
        const localVal = local.data[field];
        const remoteVal = remote.data[field];
        if (JSON.stringify(localVal) !== JSON.stringify(remoteVal)) {
          mergedData[field] = localVal;
          hasClientChanges = true;
        }
      } else {
        mergedData[field] = local.data[field];
        hasClientChanges = true;
      }
    }

    // Если есть только server-authoritative изменения — не отправляем локальное
    if (!hasClientChanges) {
      return { resolvedChange: null, conflict: null };
    }

    // Строим resolved change с обновлённым timestamp
    const resolvedChange: SyncChange = {
      table: local.table,
      id: local.id,
      operation: local.operation,
      data: mergedData,
      changedFields: [
        ...new Set([
          ...localFields,
          ...conflictingFields.map((f) => f.name),
        ]),
      ],
      timestamp: new Date().toISOString(),
    };

    if (conflictingFields.length > 0) {
      const localTs = new Date(local.timestamp).getTime();
      const remoteTs = new Date(remote.timestamp).getTime();

      return {
        resolvedChange,
        conflict: {
          id: local.id,
          label: `${local.table}:${local.id}`,
          localTimestamp: localTs,
          serverTimestamp: remoteTs,
          fields: conflictingFields,
        },
      };
    }

    return { resolvedChange, conflict: null };
  }

  /**
   * Преобразовать значение в строку для отображения в ConflictResolution UI.
   * Объекты/массивы сериализуются в JSON, примитивы — в String().
   */
  private _stringifyValue(value: unknown): string {
    if (value === null || value === undefined) {
      return '';
    }
    if (typeof value === 'object') {
      return JSON.stringify(value, null, 2);
    }
    return String(value);
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
   *
   * Использует 3-way merge (P0-CR-05) вместо LWW.
   * Возвращает unresolvedConflicts для отображения в ConflictResolutionModal.
   */
  private async _pushPhase(
    results: EntitySyncResult[],
  ): Promise<{
    changesPushed: number;
    conflictsResolved: number;
    unresolvedConflicts: ThreeWayConflict[];
  }> {
    let totalPushed = 0;
    let totalConflicts = 0;
    let unresolvedConflicts: ThreeWayConflict[] = [];

    try {
      // 1. Собираем локальные изменения
      const localChanges = await this.collectLocalChanges();
      if (localChanges.length === 0) {
        return { changesPushed: 0, conflictsResolved: 0, unresolvedConflicts: [] };
      }

      // 2. Получаем последний diff с сервера для conflict detection
      const remoteDiff = await syncApi.fetchDiff(this.lastSyncTime, {
        entities: [...new Set(localChanges.map((c) => c.table))],
      });

      // 3. Разрешаем конфликты (3-way merge + server authority)
      const { resolved, conflicts } = await this.resolveConflicts3Way(
        localChanges,
        remoteDiff.changes,
      );
      unresolvedConflicts = conflicts;
      totalConflicts = conflicts.length;

      // 4. Отправляем смерженные изменения на сервер
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

    return {
      changesPushed: totalPushed,
      conflictsResolved: totalConflicts,
      unresolvedConflicts,
    };
  }

  /**
   * Отправить изменения на сервер с exponential backoff retry + jitter.
   *
   * Jitter предотвращает thundering herd problem при параллельных retry
   * нескольких клиентов. Задержка:
   *   backoff = baseDelay + random(0, JITTER_MAX_MS)
   *
   * Соответствует:
   *   - IEC 62443-3-3 SR 3.1 (Queue-based processing with backoff)
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
        const baseDelay =
          RETRY_DELAYS[attempt] ?? RETRY_DELAYS[RETRY_DELAYS.length - 1];
        const delay = this._jitter(baseDelay);
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
   * Заполняет changedFields ключами payload для field-level merge.
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
      changedFields: Object.keys(payload),
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

  /**
   * Добавить случайный jitter к задержке backoff.
   * Возвращает baseDelay + random(0, JITTER_MAX_MS).
   * Предотвращает thundering herd при параллельных retry.
   */
  private _jitter(baseDelay: number): number {
    const jitter = Math.floor(Math.random() * (JITTER_MAX_MS + 1));
    return baseDelay + jitter;
  }

  private _sleep(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}

// ── Singleton ───────────────────────────────────────────────────────────

/** Глобальный экземпляр DifferentialSyncService */
export const differentialSyncService = new DifferentialSyncService();
