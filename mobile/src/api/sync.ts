// ═══════════════════════════════════════════════════════════════════════════
// P1-SYNC: Differential Sync — API клиент
//
// Предоставляет методы для взаимодействия с backend sync endpoints:
//   - GET  /api/v1/sync/{entity}?since=ISO8601 — получение дельты
//   - GET  /api/v1/sync/status                 — статус синхронизации
//   - POST /api/v1/sync/apply                  — отправка локальных изменений
//
// Соответствует:
//   - IEC 62443-3-3 SR 3.1 (Queue-based processing)
//   - OWASP ASVS L3 V1 (Input validation), V5 (Output encoding)
// ═══════════════════════════════════════════════════════════════════════════

import { apiClient } from './client';

// ── Типы ────────────────────────────────────────────────────────────────

/** Запрос на получение дельты */
export interface SyncDiffRequest {
  since: string;
}

/** Операция изменения сущности */
export interface SyncChange {
  table: string;
  id: string;
  operation: 'insert' | 'update' | 'delete';
  data: Record<string, unknown>;
  /** Поля, которые были изменены (для field-level 3-way merge) */
  changedFields?: string[];
  timestamp: string;
}

/** Ответ сервера с дельтой */
export interface SyncDiffResponse {
  changes: SyncChange[];
  serverTime: string;
  hasMore: boolean;
}

/** Статус синхронизации с сервера */
export interface SyncStatusResponse {
  bandwidth_usage_bytes: number;
  last_sync_at: Record<string, string>;
  total_syncs: number;
  total_changes: number;
}

/** Результат apply изменений */
export interface ApplyChangesResponse {
  applied: number;
  failed: number;
  errors: Array<{ id: string; error: string }>;
}

/** Тип сжатия */
export type CompressionType = 'gzip' | 'brotli' | 'none';

// ── Константы ──────────────────────────────────────────────────────────

const SYNC_TIMEOUT = 30_000;
const DEFAULT_PAGE_SIZE = 500;

// ── Sync API ────────────────────────────────────────────────────────────

export const syncApi = {
  /**
   * Получить diff-изменения с сервера.
   * Выполняет per-entity запросы для каждой разрешённой сущности
   * и агрегирует результат.
   */
  async fetchDiff(
    since: string,
    options?: {
      compression?: CompressionType;
      entities?: string[];
    },
  ): Promise<SyncDiffResponse> {
    const entities = options?.entities ?? [
      'work_orders',
      'devices',
      'photos',
      'audit',
    ];
    const compression = options?.compression ?? 'gzip';

    const allChanges: SyncChange[] = [];
    let serverTime = new Date().toISOString();
    let hasMore = false;

    for (const entity of entities) {
      const params = new URLSearchParams();
      params.set('since', since);
      params.set('compression', compression);
      params.set('page_size', String(DEFAULT_PAGE_SIZE));

      const response = await apiClient.get<{
        changes: Array<{
          id: string;
          type: 'created' | 'updated' | 'deleted';
          entity: string;
          fields?: Record<string, unknown>;
          updated_at: string;
        }>;
        timestamp: string;
        has_more: boolean;
        total_count: number;
      }>(`/sync/${entity}?${params.toString()}`, {
        timeout: SYNC_TIMEOUT,
        headers: {
          'Accept-Encoding': compression === 'none' ? 'identity' : 'gzip, deflate',
        },
        decompress: true,
      });

      const data = response.data;

      // Преобразуем ChangeEntry → SyncChange
      for (const change of data.changes) {
        allChanges.push({
          table: change.entity,
          id: change.id,
          operation: mapOperation(change.type),
          data: change.fields ?? {},
          timestamp: change.updated_at,
        });
      }

      serverTime = data.timestamp;
      if (data.has_more) {
        hasMore = true;
      }
    }

    return {
      changes: allChanges,
      serverTime,
      hasMore,
    };
  },

  /**
   * Отправить локальные изменения на сервер.
   * POST /api/v1/sync/apply
   */
  async applyChanges(
    changes: SyncChange[],
  ): Promise<ApplyChangesResponse> {
    if (changes.length === 0) {
      return { applied: 0, failed: 0, errors: [] };
    }

    const response = await apiClient.post<ApplyChangesResponse>(
      '/sync/apply',
      { changes },
      { timeout: SYNC_TIMEOUT },
    );

    return response.data;
  },

  /**
   * Получить статус синхронизации с сервера.
   * GET /api/v1/sync/status
   */
  async getStatus(): Promise<SyncStatusResponse | null> {
    try {
      const response = await apiClient.get<SyncStatusResponse>(
        '/sync/status',
        { timeout: SYNC_TIMEOUT },
      );
      return response.data;
    } catch {
      return null;
    }
  },

  // ── Helpers ─────────────────────────────────────────────────────────

};

/**
 * Преобразовать тип изменения из backend формата в SyncChange.
 */
function mapOperation(
  type: 'created' | 'updated' | 'deleted',
): 'insert' | 'update' | 'delete' {
  switch (type) {
    case 'created':
      return 'insert';
    case 'updated':
      return 'update';
    case 'deleted':
      return 'delete';
  }
}
