// ═══════════════════════════════════════════════════════════════════════
// Notification Store (Zustand)
// ARCH.1: Client-side состояние для уведомлений.
// Server state (список уведомлений из API) — через React Query.
// Этот store управляет: фильтрами, пагинацией, UI-состоянием.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

// ─── Types ──────────────────────────────────────────────────────────

export type NotificationType = 'success' | 'warning' | 'error' | 'info';
export type NotificationFilter = 'all' | 'unread' | 'read';

export interface NotificationItem {
  id: string;
  title: string;
  message: string;
  type: NotificationType;
  timestamp: string;
  read: boolean;
  link?: string;
}

export interface NotificationState {
  // Client-side filter
  filter: NotificationFilter;
  searchQuery: string;

  // Pagination (client-side)
  page: number;
  pageSize: number;

  // UI state
  dropdownOpen: boolean;
  unreadCount: number;

  // Actions
  setFilter: (filter: NotificationFilter) => void;
  setSearchQuery: (query: string) => void;
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  setDropdownOpen: (open: boolean) => void;
  setUnreadCount: (count: number) => void;
  decrementUnread: () => void;
  reset: () => void;
}

// ─── Constants ──────────────────────────────────────────────────────

const DEFAULT_PAGE_SIZE = 20;

// ─── Store ──────────────────────────────────────────────────────────

export const useNotificationStore = create<NotificationState>()((set, get) => ({
  // Initial state
  filter: 'all',
  searchQuery: '',
  page: 1,
  pageSize: DEFAULT_PAGE_SIZE,
  dropdownOpen: false,
  unreadCount: 0,

  // Actions
  setFilter: (filter) => set({ filter, page: 1 }),

  setSearchQuery: (searchQuery) => set({ searchQuery, page: 1 }),

  setPage: (page) => set({ page }),

  setPageSize: (pageSize) => set({ pageSize, page: 1 }),

  setDropdownOpen: (dropdownOpen) => set({ dropdownOpen }),

  setUnreadCount: (unreadCount) => set({ unreadCount }),

  decrementUnread: () => {
    const count = get().unreadCount;
    if (count > 0) {
      set({ unreadCount: count - 1 });
    }
  },

  reset: () =>
    set({
      filter: 'all',
      searchQuery: '',
      page: 1,
      pageSize: DEFAULT_PAGE_SIZE,
      dropdownOpen: false,
      unreadCount: 0,
    }),
}));
