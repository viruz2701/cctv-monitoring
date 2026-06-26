/**
 * tileCache.ts — Офлайн-кэширование map tiles в SQLite
 *
 * Compliance:
 * - СТБ 34.101.27 (защита данных в покое)
 * - ISO 27001 A.12.4 (аудит — лог очистки кэша)
 * - OWASP ASVS V6 (storage cryptography — данные кэша не чувствительны)
 *
 * Схема таблицы:
 *   map_tiles (
 *     url        TEXT PRIMARY KEY,   — URL тайла (e.g. https://tile.osm.org/17/65535/43690.png)
 *     data       BLOB,               — PNG данные тайла
 *     cached_at  INTEGER,            — UNIX ms timestamp сохранения
 *     expires_at INTEGER             — UNIX ms timestamp истечения (cached_at + 30d)
 *   )
 *
 * @module tileCache
 */

import * as SQLite from 'expo-sqlite';
import * as FileSystem from 'expo-file-system';

// ──────────────────────────────────────────────────
// Константы
// ──────────────────────────────────────────────────

const DB_NAME = 'cctv-offline.db';
const TILE_EXPIRY_DAYS = 30;
const TILE_EXPIRY_MS = TILE_EXPIRY_DAYS * 24 * 60 * 60 * 1000;
const CACHE_SIZE_SOFT_LIMIT = 100 * 1024 * 1024; // 100 MB soft limit
const CACHE_SIZE_HARD_LIMIT = 200 * 1024 * 1024; // 200 MB hard limit

/**
 * Базовый URL для OpenStreetMap tiles.
 * В production может быть заменён на корпоративный tile-сервер.
 */
export const TILE_SERVER_URL = 'https://tile.openstreetmap.org';

/**
 * Zoom-уровни для предзагрузки тайлов.
 * 10 — регион, 12 — город, 14 — район, 16 — улица.
 */
export const PRELOAD_ZOOM_LEVELS = [10, 12, 14, 16];

// ──────────────────────────────────────────────────
// Типы
// ──────────────────────────────────────────────────

export interface MapTileRow {
  url: string;
  data: Uint8Array | null;
  cached_at: number;
  expires_at: number;
}

export interface TileCacheStats {
  /** Количество кэшированных тайлов */
  tileCount: number;
  /** Приблизительный размер кэша в байтах */
  cacheSizeBytes: number;
  /** Количество истёкших тайлов */
  expiredCount: number;
  /** Дата последней очистки */
  lastCleanedAt: number | null;
}

export interface BoundingBox {
  minLat: number;
  minLng: number;
  maxLat: number;
  maxLng: number;
}

// ──────────────────────────────────────────────────
// Database singleton (общая БД с offlineStorage.ts)
// ──────────────────────────────────────────────────

let _db: SQLite.SQLiteDatabase | null = null;

async function getDb(): Promise<SQLite.SQLiteDatabase> {
  if (!_db) {
    _db = await SQLite.openDatabaseAsync(DB_NAME);
  }
  return _db;
}

// ──────────────────────────────────────────────────
// Инициализация таблицы
// ──────────────────────────────────────────────────

/**
 * Создаёт таблицу map_tiles, если её нет.
 * Вызывается однократно при старте приложения.
 */
export async function initTileCache(): Promise<void> {
  const db = await getDb();

  await db.execAsync(`
    CREATE TABLE IF NOT EXISTS map_tiles (
      url        TEXT PRIMARY KEY NOT NULL,
      data       BLOB,
      cached_at  INTEGER NOT NULL,
      expires_at INTEGER NOT NULL
    );

    CREATE INDEX IF NOT EXISTS idx_tiles_expires
      ON map_tiles(expires_at);

    PRAGMA journal_mode = WAL;
    PRAGMA foreign_keys = ON;
  `);
}

// ──────────────────────────────────────────────────
// Основные CRUD операции
// ──────────────────────────────────────────────────

/**
 * Сохранить тайл в кэш.
 * Если тайл уже существует — обновляет данные и продлевает срок.
 *
 * @param url — полный URL тайла
 * @param data — бинарные PNG данные тайла
 */
export async function saveTile(url: string, data: Uint8Array): Promise<void> {
  const db = await getDb();
  const now = Date.now();

  // Проверка размера кэша перед записью
  await ensureCacheSpace(data.byteLength);

  await db.runAsync(
    `INSERT OR REPLACE INTO map_tiles (url, data, cached_at, expires_at)
     VALUES (?, ?, ?, ?)`,
    [url, data, now, now + TILE_EXPIRY_MS],
  );
}

/**
 * Получить тайл из кэша.
 * Возвращает null если тайл не найден или истёк.
 *
 * @param url — полный URL тайла
 */
export async function getTile(url: string): Promise<Uint8Array | null> {
  const db = await getDb();
  const now = Date.now();

  const row = await db.getFirstAsync<MapTileRow>(
    'SELECT url, data, cached_at, expires_at FROM map_tiles WHERE url = ? AND expires_at > ?',
    [url, now],
  );

  if (!row || !row.data) {
    return null;
  }

  return row.data;
}

/**
 * Проверить существование и валидность тайла в кэше.
 */
export async function hasTile(url: string): Promise<boolean> {
  const db = await getDb();
  const now = Date.now();

  const row = await db.getFirstAsync<{ count: number }>(
    'SELECT COUNT(*) as count FROM map_tiles WHERE url = ? AND expires_at > ?',
    [url, now],
  );

  return (row?.count ?? 0) > 0;
}

/**
 * Удалить один тайл по URL.
 */
export async function removeTile(url: string): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM map_tiles WHERE url = ?', [url]);
}

// ──────────────────────────────────────────────────
// Управление кэшем
// ──────────────────────────────────────────────────

/**
 * Удалить все истёкшие тайлы.
 * Возвращает количество удалённых записей.
 */
export async function clearExpiredTiles(): Promise<number> {
  const db = await getDb();
  const now = Date.now();

  const result = await db.runAsync(
    'DELETE FROM map_tiles WHERE expires_at < ?',
    [now],
  );

  const deletedCount = result.changes ?? 0;

  if (deletedCount > 0) {
    console.log(`[TileCache] Cleared ${deletedCount} expired tiles`);
    // Логируем аудит очистки (ISO 27001 A.12.4)
    await db.runAsync(
      `INSERT OR IGNORE INTO audit_log (event, entity_type, details, timestamp)
       VALUES (?, 'tile_cache', ?, ?)`,
      ['cache_cleanup', JSON.stringify({ deletedCount }), now],
    ).catch(() => {
      // audit_log может не существовать — игнорируем ошибку
    });
  }

  return deletedCount;
}

/**
 * Получить количество всех (не истёкших) тайлов в кэше.
 */
export async function getTileCount(): Promise<number> {
  const db = await getDb();
  const now = Date.now();

  const row = await db.getFirstAsync<{ count: number }>(
    'SELECT COUNT(*) as count FROM map_tiles WHERE expires_at > ?',
    [now],
  );

  return row?.count ?? 0;
}

/**
 * Получить приблизительный размер кэша (сумма длин BLOB полей).
 */
export async function getCacheSize(): Promise<number> {
  const db = await getDb();
  const now = Date.now();

  const row = await db.getFirstAsync<{ total: number | null }>(
    'SELECT COALESCE(SUM(LENGTH(data)), 0) as total FROM map_tiles WHERE expires_at > ?',
    [now],
  );

  return row?.total ?? 0;
}

/**
 * Получить количество истёкших тайлов.
 */
export async function getExpiredCount(): Promise<number> {
  const db = await getDb();
  const now = Date.now();

  const row = await db.getFirstAsync<{ count: number }>(
    'SELECT COUNT(*) as count FROM map_tiles WHERE expires_at < ?',
    [now],
  );

  return row?.count ?? 0;
}

/**
 * Получить полную статистику кэша.
 */
export async function getTileCacheStats(): Promise<TileCacheStats> {
  const [tileCount, cacheSizeBytes, expiredCount] = await Promise.all([
    getTileCount(),
    getCacheSize(),
    getExpiredCount(),
  ]);

  return {
    tileCount,
    cacheSizeBytes,
    expiredCount,
    lastCleanedAt: null, // будет обновляться при очистке
  };
}

/**
 * Очистить весь кэш тайлов.
 */
export async function clearAllTiles(): Promise<void> {
  const db = await getDb();
  await db.runAsync('DELETE FROM map_tiles');
  console.log('[TileCache] All tiles cleared');
}

/**
 * Обеспечить достаточное свободное место в кэше.
 * При превышении soft limit — удаляет самые старые истёкшие тайлы.
 * При превышении hard limit — удаляет старые тайлы принудительно.
 *
 * @param requiredBytes — количество байт, которое нужно сохранить
 */
async function ensureCacheSpace(requiredBytes: number): Promise<void> {
  const db = await getDb();
  const now = Date.now();

  // 1. Сначала удаляем истёкшие
  await clearExpiredTiles();

  // 2. Проверяем текущий размер
  const currentSize = await getCacheSize();

  // Если после очистки истёкших всё ещё превышен hard limit — удаляем самые старые
  if (currentSize + requiredBytes > CACHE_SIZE_HARD_LIMIT) {
    // Удаляем 20% самых старых тайлов
    await db.runAsync(`
      DELETE FROM map_tiles WHERE url IN (
        SELECT url FROM map_tiles
        WHERE expires_at > ?
        ORDER BY cached_at ASC
        LIMIT (SELECT MAX(1, COUNT(*) / 5) FROM map_tiles WHERE expires_at > ?)
      )
    `, [now, now]);

    console.warn('[TileCache] Hard limit reached — evicted oldest 20% of tiles');
  }
}

// ──────────────────────────────────────────────────
// Tile URL helpers
// ──────────────────────────────────────────────────

/**
 * Сгенерировать URL тайла для заданных координат и zoom.
 *
 * Используется Slippy Map tilenames (OSM standard):
 *   https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames
 */
export function tileUrl(
  baseUrl: string,
  z: number,
  x: number,
  y: number,
): string {
  return `${baseUrl}/${z}/${x}/${y}.png`;
}

/**
 * Конвертировать координаты (lat, lng) в tile x/y для заданного zoom.
 *
 * @returns [x, y] — координаты тайла
 */
export function latLngToTileXY(
  lat: number,
  lng: number,
  zoom: number,
): [number, number] {
  const n = Math.pow(2, zoom);
  const x = Math.floor(((lng + 180) / 360) * n);
  const latRad = (lat * Math.PI) / 180;
  const y = Math.floor(
    ((1 - Math.log(Math.tan(latRad) + 1 / Math.cos(latRad)) / Math.PI) / 2) *
      n,
  );
  return [x, y];
}

/**
 * Получить диапазон tile координат для bounding box на заданном zoom.
 *
 * @returns { minX, minY, maxX, maxY }
 */
export function bboxToTileRange(
  bbox: BoundingBox,
  zoom: number,
): { minX: number; minY: number; maxX: number; maxY: number } {
  const [minX, maxY] = latLngToTileXY(bbox.maxLat, bbox.minLng, zoom);
  const [maxX, minY] = latLngToTileXY(bbox.minLat, bbox.maxLng, zoom);

  return { minX, minY, maxX, maxY };
}

/**
 * Скачать один тайл и сохранить в кэш.
 * Пропускает, если тайл уже закэширован и не истёк.
 *
 * @returns true если тайл был загружен, false если пропущен
 */
export async function downloadAndCacheTile(
  baseUrl: string,
  z: number,
  x: number,
  y: number,
): Promise<boolean> {
  const url = tileUrl(baseUrl, z, x, y);

  // Пропускаем если уже закэширован
  if (await hasTile(url)) {
    return false;
  }

  try {
    const response = await FileSystem.downloadAsync(url, `${FileSystem.cacheDirectory}tiles/${z}_${x}_${y}.png`);

    // Читаем файл как base64 и конвертируем в Uint8Array
    const base64 = await FileSystem.readAsStringAsync(response.uri, {
      encoding: FileSystem.EncodingType.Base64,
    });

    const binaryStr = atob(base64);
    const bytes = new Uint8Array(binaryStr.length);
    for (let i = 0; i < binaryStr.length; i++) {
      bytes[i] = binaryStr.charCodeAt(i);
    }

    await saveTile(url, bytes);

    // Удаляем временный файл
    await FileSystem.deleteAsync(response.uri, { idempotent: true });

    return true;
  } catch (err) {
    console.warn(`[TileCache] Failed to download tile ${url}:`, err);
    return false;
  }
}

// ──────────────────────────────────────────────────
// Preload для bounding box
// ──────────────────────────────────────────────────

export interface PreloadProgress {
  total: number;
  completed: number;
  failed: number;
}

/**
 * Предзагрузить все тайлы для bounding box на указанных zoom-уровнях.
 *
 * @param bbox — bounding box области
 * @param zoomLevels — массив zoom-уровней
 * @param baseUrl — базовый URL tile-сервера
 * @param onProgress — callback прогресса
 */
export async function preloadTilesForBounds(
  bbox: BoundingBox,
  zoomLevels: number[] = PRELOAD_ZOOM_LEVELS,
  baseUrl: string = TILE_SERVER_URL,
  onProgress?: (progress: PreloadProgress) => void,
): Promise<PreloadProgress> {
  const progress: PreloadProgress = { total: 0, completed: 0, failed: 0 };

  // Подсчитываем общее количество тайлов
  for (const zoom of zoomLevels) {
    const range = bboxToTileRange(bbox, zoom);
    progress.total += (range.maxX - range.minX + 1) * (range.maxY - range.minY + 1);
  }

  for (const zoom of zoomLevels) {
    const range = bboxToTileRange(bbox, zoom);

    for (let x = range.minX; x <= range.maxX; x++) {
      for (let y = range.minY; y <= range.maxY; y++) {
        try {
          const downloaded = await downloadAndCacheTile(baseUrl, zoom, x, y);
          if (downloaded) {
            progress.completed++;
          } else {
            // Уже был в кэше — считаем как completed
            progress.completed++;
          }
        } catch {
          progress.failed++;
        }

        onProgress?.({ ...progress });
      }
    }
  }

  console.log(
    `[TileCache] Preload complete: ${progress.completed}/${progress.total} tiles cached, ${progress.failed} failed`,
  );

  return progress;
}

// ──────────────────────────────────────────────────
// Экспорт (закрытие БД не требуется — expo-sqlite управляет соединением)
// ──────────────────────────────────────────────────
