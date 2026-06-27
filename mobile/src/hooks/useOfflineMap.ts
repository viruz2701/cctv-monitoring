/**
 * useOfflineMap — хук для загрузки, кеширования и фильтрации устройств на карте.
 *
 * Offline-first стратегия (UX-02):
 * 1. При загрузке — показываем кешированные данные из AsyncStorage
 * 2. Если есть интернет — параллельно загружаем свежие данные с сервера
 * 3. Обновляем кеш при получении свежих данных
 * 4. При офлайн — работаем только с кешем
 *
 * Offline Tile Caching (P0-3.3):
 * - Тайлы кэшируются в SQLite (expo-sqlite) для работы карты офлайн
 * - Preload при назначении на site
 * - Автоматическая очистка истёкших тайлов (>30 дней)
 * - Offline indicator с количеством кэшированных тайлов
 *
 * Compliance:
 * - СТБ 34.101.27 (защита данных в покое)
 * - IEC 62443 SR 3.1 (зоны безопасности)
 * - ISO 27001 A.12.4 (аудит кэша)
 *
 * @module useOfflineMap
 */

import { useState, useEffect, useCallback } from 'react';
import NetInfo from '@react-native-community/netinfo';
import { devicesApi, DeviceMapData } from '../api/devices';
import { useDeviceMapStore } from '../store/deviceMapStore';
import { useLocation } from './useLocation';
import {
  initTileCache,
  getTileCount,
  getCacheSize,
  getExpiredCount,
  clearExpiredTiles,
  clearAllTiles,
  preloadTilesForBounds,
  saveCacheMetadata,
  getPreloadedSites,
  getPreloadedSitesStats,
  TILE_SERVER_URL,
  PRELOAD_ZOOM_LEVELS,
  type BoundingBox,
  type PreloadProgress,
  type TileCacheStats,
  type CacheMetadata,
} from '../services/tileCache';

// ──────────────────────────────────────────────────
// Типы
// ──────────────────────────────────────────────────

import type { PreloadSiteStatus } from '../store/deviceMapStore';

type DeviceStatusFilter = 'all' | 'ONLINE' | 'OFFLINE' | 'WARNING';
type DeviceTypeFilter = 'all' | 'camera' | 'nvr' | 'dvr' | 'switch';

export interface TileCacheStatus {
  /** Количество кэшированных тайлов */
  tileCount: number;
  /** Размер кэша в байтах */
  cacheSizeBytes: number;
  /** Количество истёкших тайлов */
  expiredCount: number;
  /** Идёт ли предзагрузка */
  isPreloading: boolean;
  /** Прогресс предзагрузки */
  preloadProgress: PreloadProgress | null;
  /** Дата последней очистки */
  lastCleanedAt: number | null;
}

interface UseOfflineMapReturn {
  /** Устройства с координатами (из кеша или онлайн) */
  devices: DeviceMapData[];
  /** Текущая позиция пользователя */
  currentLocation: { latitude: number; longitude: number } | null;
  /** Загрузка */
  isMapLoading: boolean;
  /** Ошибка */
  mapError: string | null;
  /** Время последней синхронизации */
  lastSyncAt: Date | null;
  /** Фильтр по статусу */
  statusFilter: DeviceStatusFilter;
  /** Установить фильтр статуса */
  setStatusFilter: (filter: DeviceStatusFilter) => void;
  /** Фильтр по типу устройства */
  deviceTypeFilter: DeviceTypeFilter;
  /** Установить фильтр типа устройства */
  setDeviceTypeFilter: (filter: DeviceTypeFilter) => void;
  /** Отфильтрованные устройства */
  filteredDevices: DeviceMapData[];
  /** Обновить данные с сервера */
  refreshDevices: () => Promise<void>;
  /** Количество устройств по статусам */
  statusCounts: {
    ONLINE: number;
    OFFLINE: number;
    WARNING: number;
  };
  /** Есть ли интернет */
  isOnline: boolean;

  /** Список предзагруженных сайтов (из SQLite cache_metadata) */
  preloadedSites: CacheMetadata[];
  /** Статусы предзагрузки по сайтам (из стора) */
  preloadStatuses: Record<string, PreloadSiteStatus>;

  // ── Tile Cache API ──────────────────────────────

  /** Статус кэша тайлов */
  tileCacheStatus: TileCacheStatus;
  /** Предзагрузить тайлы для bounding box site */
  preloadTilesForSite: (
    bbox: BoundingBox,
    zoomLevels?: number[],
    siteId?: string,
    areaName?: string,
  ) => Promise<PreloadProgress>;
  /** Очистить кэш тайлов */
  clearTileCache: () => Promise<void>;
  /** Принудительно очистить истёкшие тайлы */
  cleanExpiredTiles: () => Promise<number>;
  /** Обновить статистику кэша */
  refreshTileCacheStats: () => Promise<void>;
}

// ──────────────────────────────────────────────────
// Хук
// ──────────────────────────────────────────────────

/**
 * useOfflineMap — основной хук для работы с картой в офлайн-режиме.
 *
 * Возвращает:
 * - Данные устройств (кэш/онлайн)
 * - Фильтрацию по статусу и типу
 * - Статус кэша тайлов
 * - Методы предзагрузки и очистки кэша
 */
export function useOfflineMap(): UseOfflineMapReturn {
  const {
    cachedDevices,
    lastSyncAt,
    isLoading: storeLoading,
    error: storeError,
    setCachedDevices,
    loadCachedDevices,
    preloadedSites,
    setPreloadedSites,
    updatePreloadStatus,
    removePreloadedSite,
    setTileCacheStats,
  } = useDeviceMapStore();

  const { latitude, longitude, loading: locationLoading } = useLocation();
  const [isOnline, setIsOnline] = useState(true);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<DeviceStatusFilter>('all');
  const [deviceTypeFilter, setDeviceTypeFilter] = useState<DeviceTypeFilter>('all');

  // ── Tile cache state ────────────────────────────

  const [tileCount, setTileCount] = useState(0);
  const [cacheSizeBytes, setCacheSizeBytes] = useState(0);
  const [expiredCount, setExpiredCount] = useState(0);
  const [isPreloading, setIsPreloading] = useState(false);
  const [preloadProgress, setPreloadProgress] = useState<PreloadProgress | null>(null);
  const [lastCleanedAt, setLastCleanedAt] = useState<number | null>(null);

  // ── Обновление статистики кэша ──────────────────

  const refreshTileCacheStats = useCallback(async () => {
    try {
      const [count, size, expired, preloaded, stats] = await Promise.all([
        getTileCount(),
        getCacheSize(),
        getExpiredCount(),
        getPreloadedSites(),
        getPreloadedSitesStats(),
      ]);
      setTileCount(count);
      setCacheSizeBytes(size);
      setExpiredCount(expired);
      setPreloadedSites(preloaded);
      setTileCacheStats(stats.totalTiles, stats.totalSizeBytes);
    } catch (err) {
      console.error('[TileCache] Failed to refresh stats:', err);
    }
  }, [setPreloadedSites, setTileCacheStats]);

  // ── Мониторинг сетевого статуса ────────────────

  useEffect(() => {
    const unsubscribe = NetInfo.addEventListener((state) => {
      setIsOnline(state.isConnected ?? true);
    });
    return () => unsubscribe();
  }, []);

  // ── Инициализация при монтировании ──────────────

  useEffect(() => {
    const init = async () => {
      setIsLoading(true);

      try {
        // 1. Инициализируем tile cache таблицу
        await initTileCache();

        // 2. Очищаем истёкшие тайлы при старте
        const cleaned = await clearExpiredTiles();
        if (cleaned > 0) {
          setLastCleanedAt(Date.now());
        }

        // 3. Загружаем статистику кэша + предзагруженные сайты
        await refreshTileCacheStats();

        // 4. Загружаем кэш устройств
        await loadCachedDevices();
      } catch (err) {
        console.error('[useOfflineMap] Init error:', err);
      }

      setIsLoading(false);
    };
    init();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── Автоматическая синхронизация при появлении интернета ──

  useEffect(() => {
    if (isOnline) {
      refreshDevices();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOnline]);

  // ── Обновление устройств ────────────────────────

  const refreshDevices = useCallback(async () => {
    if (!isOnline) return;

    try {
      setError(null);
      const response = await devicesApi.getDevicesForMap();
      await setCachedDevices(response.devices);
    } catch (err) {
      console.error('Failed to refresh device map data:', err);
      // Graceful degradation: при ошибке сети продолжаем показывать кеш
      setError('Не удалось обновить данные. Используется кеш.');
    }
  }, [isOnline, setCachedDevices]);

  // ── Preload тайлов для site ─────────────────────

  /**
   * Предзагрузить тайлы для bounding box объекта (site).
   * Загружает тайлы на zoom-уровнях 10, 12, 14, 16.
   * Сохраняет метаданные кэша в SQLite после завершения.
   *
   * @param bbox — bounding box site (minLat, minLng, maxLat, maxLng)
   * @param areaName — название сайта для метаданных кэша
   * @param siteId — ID сайта для метаданных кэша
   * @param zoomLevels — уровни детализации (по умолчанию PRELOAD_ZOOM_LEVELS)
   */
  const preloadTilesForSite = useCallback(
    async (
      bbox: BoundingBox,
      zoomLevels: number[] = PRELOAD_ZOOM_LEVELS,
      siteId?: string,
      areaName?: string,
    ): Promise<PreloadProgress> => {
      if (isPreloading) {
        console.warn('[TileCache] Preload already in progress');
        return { total: 0, completed: 0, failed: 0 };
      }

      setIsPreloading(true);
      setPreloadProgress({ total: 0, completed: 0, failed: 0 });

      // Устанавливаем статус preloading в сторе
      if (siteId) {
        updatePreloadStatus(siteId, {
          siteId,
          status: 'preloading',
          progress: 0,
          tileCount: 0,
          sizeBytes: 0,
        });
      }

      try {
        const result = await preloadTilesForBounds(
          bbox,
          zoomLevels,
          TILE_SERVER_URL,
          (progress) => {
            setPreloadProgress({ ...progress });
            // Обновляем прогресс в сторе
            if (siteId) {
              updatePreloadStatus(siteId, {
                progress: progress.total > 0
                  ? progress.completed / progress.total
                  : 0,
                tileCount: progress.completed,
              });
            }
          },
        );

        // Сохраняем метаданные кэша в SQLite
        if (siteId && areaName) {
          const now = Date.now();
          const expiryDays = 30;
          const expiryMs = expiryDays * 24 * 60 * 60 * 1000;
          const metadata: CacheMetadata = {
            siteId,
            areaName,
            zoomLevels,
            bbox,
            tileCount: result.completed,
            sizeBytes: 0, // будет обновлено при следующем refresh stats
            preloadedAt: now,
            expiresAt: now + expiryMs,
            status: result.failed > 0 ? 'failed' : 'complete',
          };

          await saveCacheMetadata(metadata);

          // Обновляем статус в сторе
          updatePreloadStatus(siteId, {
            status: result.failed > 0 ? 'failed' : 'complete',
            progress: 1,
            tileCount: result.completed,
          });
        }

        await refreshTileCacheStats();
        return result;
      } finally {
        setIsPreloading(false);
      }
    },
    [isPreloading, refreshTileCacheStats, updatePreloadStatus],
  );

  // ── Очистка кэша ────────────────────────────────

  const clearTileCache = useCallback(async () => {
    try {
      await clearAllTiles();
      setPreloadedSites([]);
      await refreshTileCacheStats();
      setLastCleanedAt(Date.now());
    } catch (err) {
      console.error('[TileCache] Failed to clear cache:', err);
    }
  }, [refreshTileCacheStats, setPreloadedSites]);

  const cleanExpiredTiles = useCallback(async (): Promise<number> => {
    try {
      const count = await clearExpiredTiles();
      if (count > 0) {
        setLastCleanedAt(Date.now());
        await refreshTileCacheStats();
      }
      return count;
    } catch (err) {
      console.error('[TileCache] Failed to clean expired tiles:', err);
      return 0;
    }
  }, [refreshTileCacheStats]);

  // ── Фильтрация устройств ────────────────────────

  const filteredDevices = cachedDevices.filter((device) => {
    if (statusFilter !== 'all' && device.status !== statusFilter) return false;
    if (deviceTypeFilter !== 'all' && device.device_type !== deviceTypeFilter) return false;
    return true;
  });

  // Количество по статусам
  const statusCounts = {
    ONLINE: cachedDevices.filter((d) => d.status === 'ONLINE').length,
    OFFLINE: cachedDevices.filter((d) => d.status === 'OFFLINE').length,
    WARNING: cachedDevices.filter((d) => d.status === 'WARNING').length,
  };

  // Собираем статус кэша
  const tileCacheStatus: TileCacheStatus = {
    tileCount,
    cacheSizeBytes,
    expiredCount,
    isPreloading,
    preloadProgress,
    lastCleanedAt,
  };

  return {
    devices: cachedDevices,
    currentLocation: locationLoading ? null : { latitude, longitude },
    isMapLoading: isLoading || storeLoading,
    mapError: error || storeError,
    lastSyncAt: lastSyncAt ? new Date(lastSyncAt) : null,
    statusFilter,
    setStatusFilter,
    deviceTypeFilter,
    setDeviceTypeFilter,
    filteredDevices,
    refreshDevices,
    statusCounts,
    isOnline,
    preloadedSites,
    preloadStatuses: useDeviceMapStore.getState().preloadStatuses,

    // Tile Cache API
    tileCacheStatus,
    preloadTilesForSite,
    clearTileCache,
    cleanExpiredTiles,
    refreshTileCacheStats,
  };
}
