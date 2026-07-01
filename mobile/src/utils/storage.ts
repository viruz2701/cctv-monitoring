import AsyncStorage from '@react-native-async-storage/async-storage';

/**
 * Hybrid storage — AsyncStorage for simple key-value (auth tokens, push token)
 * + WatermelonDB for structured offline data (work_orders, devices, sites, mutations).
 *
 * SYNC_QUEUE has been migrated to WatermelonDB `pending_mutations` table.
 * See `mobile/src/database/` for the reactive database layer.
 */

const KEYS = {
  TOKEN: 'token',
  REFRESH_TOKEN: 'refreshToken',
  USER: 'user',
  PUSH_TOKEN: 'pushToken',
  DEVICE_MAP_CACHE: 'deviceMapCache',
} as const;

export const storage = {
  // ── Auth tokens ──────────────────────────────────────────────

  async getToken(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.TOKEN);
  },

  async setToken(token: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.TOKEN, token);
  },

  async removeToken(): Promise<void> {
    return AsyncStorage.removeItem(KEYS.TOKEN);
  },

  async getRefreshToken(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.REFRESH_TOKEN);
  },

  async setRefreshToken(token: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.REFRESH_TOKEN, token);
  },

  async removeRefreshToken(): Promise<void> {
    return AsyncStorage.removeItem(KEYS.REFRESH_TOKEN);
  },

  // ── User profile (cached JSON) ───────────────────────────────

  async getUser(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.USER);
  },

  async setUser(user: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.USER, user);
  },

  async removeUser(): Promise<void> {
    return AsyncStorage.removeItem(KEYS.USER);
  },

  // ── Push notification token ──────────────────────────────────

  async getPushToken(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.PUSH_TOKEN);
  },

  async setPushToken(token: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.PUSH_TOKEN, token);
  },

  // ── Device map cache (minimal GeoJSON for map tiles) ─────────

  async getDeviceMapCache(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.DEVICE_MAP_CACHE);
  },

  async setDeviceMapCache(data: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.DEVICE_MAP_CACHE, data);
  },

  // ── Generic key-value methods (for ad-hoc caching) ───────────

  async getItem(key: string): Promise<string | null> {
    return AsyncStorage.getItem(key);
  },

  async setItem(key: string, value: string): Promise<void> {
    return AsyncStorage.setItem(key, value);
  },

  async removeItem(key: string): Promise<void> {
    return AsyncStorage.removeItem(key);
  },

  // ── Cleanup ──────────────────────────────────────────────────

  async clearAll(): Promise<void> {
    const keys = Object.values(KEYS);
    await AsyncStorage.multiRemove(keys);
  },
};