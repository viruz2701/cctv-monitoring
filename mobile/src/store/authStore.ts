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
  user: User | null;
  isLoading: boolean;
  isAuthenticated: boolean;

  setAuth: (token: string, user: User) => Promise<void>;
  logout: () => Promise<void>;
  loadStoredAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  isLoading: true,
  isAuthenticated: false,

  setAuth: async (token, user) => {
    await storage.setToken(token);
    await storage.setUser(JSON.stringify(user));
    set({ token, user, isAuthenticated: true, isLoading: false });
  },

  logout: async () => {
    await storage.removeToken();
    await storage.removeUser();
    set({ token: null, user: null, isAuthenticated: false, isLoading: false });
  },

  loadStoredAuth: async () => {
    try {
      const token = await storage.getToken();
      const userStr = await storage.getUser();
      if (token && userStr) {
        const user = JSON.parse(userStr);
        set({ token, user, isAuthenticated: true, isLoading: false });
      } else {
        set({ isLoading: false });
      }
    } catch (error) {
      console.error('Failed to load stored auth:', error);
      set({ isLoading: false });
    }
  },
}));