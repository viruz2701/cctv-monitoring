// ═══════════════════════════════════════════════════════════════════════
// Auth Store (Zustand)
// ARCH.1: Миграция Auth Context → Zustand.
// Client-side state для аутентификации.
// Server state (проверка /users/me при загрузке) — через AuthProvider.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

// ─── Types ──────────────────────────────────────────────────────────

export interface AuthUser {
  id: string;
  username: string;
  role: 'admin' | 'support' | 'owner' | 'manager' | 'technician' | 'viewer';
  owner_id?: string | null;
  name?: string;
  email?: string;
  avatar?: string;
  sites?: string[];
}

export interface AuthState {
  // State
  user: AuthUser | null;
  token: string | null;
  isLoading: boolean;
  isInitialized: boolean;

  // Actions
  setUser: (user: AuthUser | null) => void;
  setToken: (token: string | null) => void;
  setLoading: (loading: boolean) => void;
  setInitialized: (initialized: boolean) => void;
  logout: () => void;
  updateUser: (data: Partial<AuthUser>) => void;
  hasPermission: (roles: string | string[]) => boolean;
}

// ─── Store ──────────────────────────────────────────────────────────

export const useAuthStore = create<AuthState>()((set, get) => ({
  // Initial state
  user: null,
  token: null,
  isLoading: true,
  isInitialized: false,

  // Actions
  setUser: (user) => set({ user }),

  setToken: (token) => set({ token }),

  setLoading: (isLoading) => set({ isLoading }),

  setInitialized: (isInitialized) => set({ isInitialized }),

  logout: () => set({ user: null, token: null }),

  updateUser: (data) => {
    const user = get().user;
    if (user) {
      set({ user: { ...user, ...data } });
    }
  },

  hasPermission: (roles) => {
    const user = get().user;
    if (!user) return false;
    if (user.role === 'admin') return true;
    const allowed = Array.isArray(roles) ? roles : [roles];
    return allowed.includes(user.role);
  },
}));
