import AsyncStorage from '@react-native-async-storage/async-storage';

const KEYS = {
  TOKEN: 'token',
  REFRESH_TOKEN: 'refreshToken',
  USER: 'user',
  SYNC_QUEUE: 'syncQueue',
  PUSH_TOKEN: 'pushToken',
} as const;

export const storage = {
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

  async getUser(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.USER);
  },

  async setUser(user: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.USER, user);
  },

  async removeUser(): Promise<void> {
    return AsyncStorage.removeItem(KEYS.USER);
  },

  async getSyncQueue(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.SYNC_QUEUE);
  },

  async setSyncQueue(data: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.SYNC_QUEUE, data);
  },

  async getPushToken(): Promise<string | null> {
    return AsyncStorage.getItem(KEYS.PUSH_TOKEN);
  },

  async setPushToken(token: string): Promise<void> {
    return AsyncStorage.setItem(KEYS.PUSH_TOKEN, token);
  },

  async clearAll(): Promise<void> {
    const keys = Object.values(KEYS);
    await AsyncStorage.multiRemove(keys);
  },
};