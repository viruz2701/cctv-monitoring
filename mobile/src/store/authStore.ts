import { create } from 'zustand';
import { storage } from '../utils/storage';

interface User {
  id: string;
  username: string;
  role: string;
  email?: string;
}

interface AuthState {
  token: string | null;
  refreshToken: string | null;
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;

  setAuth: (token: string, refreshToken: string, user: User) => Promise<void>;
  logout: () => Promise<void>;
  loadStoredAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  refreshToken: null,
  user: null,
  isLoading: true,
  isAuthenticated: false,

  setAuth: async (token, refreshToken, user) => {
    await storage.setToken(token);
    await storage.setRefreshToken(refreshToken);
    await storage.setUser(JSON.stringify(user));
    set({ token, refreshToken, user, isAuthenticated: true, isLoading: false });
  },

  logout: async () => {
    await storage.removeToken();
    await storage.removeRefreshToken();
    await storage.removeUser();
    set({ token: null, refreshToken: null, user: null, isAuthenticated: false, isLoading: false });
  },

  loadStoredAuth: async () => {
    try {
      const token = await storage.getToken();
      const refreshToken = await storage.getRefreshToken();
      const userStr = await storage.getUser();
      if (token && refreshToken && userStr) {
        const user = JSON.parse(userStr);
        set({ token, refreshToken, user, isAuthenticated: true, isLoading: false });
      } else {
        set({ isLoading: false });
      }
    } catch (error) {
      console.error('Failed to load stored auth:', error);
      set({ isLoading: false });
    }
  },
}));